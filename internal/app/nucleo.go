package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"miapp/internal/cache"
	"miapp/internal/db"
	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/animeav1"
	"miapp/internal/scraper/jkanime"
	"miapp/internal/scraper/jkanime/extractors"
	"miapp/internal/stream"
)

const (
	// proveedorPrincipal encabeza la búsqueda combinada: sus resultados van
	// primero en la lista que ve el frontend.
	proveedorPrincipal = "jkanime"
	// proveedorComplemento es el segundo proveedor de la búsqueda combinada
	// (T34): AnimeAV1 se consulta EN PARALELO con JKAnime y sus resultados
	// van a continuación. AnimeFLV se retiró en T35 (histórico: ya no se
	// registra).
	proveedorComplemento = "animeav1"

	// tiempoLimiteBusqueda acota la búsqueda combinada completa. Cada
	// proveedor ya va con el timeout por petición de httpx (15 s); este es
	// la red de seguridad por si alguno se quedara colgado sin llegar a
	// devolver su error.
	tiempoLimiteBusqueda = 20 * time.Second

	// ttlCacheDetalle es la vida de cada entrada de la caché local (T21). Es
	// por deslizamiento: cada lote la renueva, así que una lista larga que
	// sigue bajando nunca caduca a mitad de camino.
	ttlCacheDetalle = 10 * time.Minute
)

// ErrDetalleNoDisponible es lo que ve el frontend cuando un proveedor no
// ofrece página de detalle (scraper.ErrNoDetails). Se devuelve como error
// tipado —no como fallo grave— con este mensaje ya pensado para la UI; el
// sentinel interno de scraper NO se arrastra, porque Wails entrega al
// frontend solo el texto y este dice algo útil. Ningún proveedor registrado
// lo devuelve hoy (jkanime y animeav1 sí ofrecen detalle); se conserva para
// el flujo síncrono y para un proveedor futuro que no lo ofrezca.
var ErrDetalleNoDisponible = errors.New("Detalle no disponible para este proveedor")

// ErrHistorialNoDisponible es lo que ven SaveProgress/GetProgress cuando no
// hay almacén abierto: antes de arrancar, si su apertura falló (el
// núcleo lo loguea y sigue, la app no muere por un historial) o si se pide
// historial sin haber arrancado nunca. Tras un shutdown el almacén sigue
// abierto pero marcado como cerrado, y ahí el error es db.ErrCerrado.
var ErrHistorialNoDisponible = errors.New("El historial de visualización no está disponible")

// ErrHomeNoDisponible es lo que ve el Home cuando el proveedor existe en el
// registro pero no implementa scraper.HomeProvider (interfaz opcional, T43):
// no es un fallo de red, es que ese sitio no ofrece secciones de inicio.
var ErrHomeNoDisponible = errors.New("El Home no está disponible para este proveedor")

// ErrFavoritosNoDisponible es lo que ven los métodos de favoritos cuando no
// hay almacén abierto: antes de arrancar, si su apertura falló (el núcleo lo
// loguea y sigue, la app no muere por los favoritos) o si el núcleo se
// construyó sin directorio de datos (T45).
var ErrFavoritosNoDisponible = errors.New("Los favoritos no están disponibles")

// Emisor entrega un evento al frontend. En producción es un cierre sobre
// runtime.EventsEmit con el contexto de startup; en tests, un doble que
// graba. nil = no emitir (antes de arrancar o en pruebas sin UI).
type Emisor func(evento string, data any)

// RegistroProviders es el trozo de scraper.Registry que usa el núcleo.
type RegistroProviders interface {
	Get(name string) (scraper.Provider, bool)
	All() []scraper.Provider
}

// ProxyLocal es el trozo de *stream.Proxy que usa el núcleo: registrar
// fuentes, escuchar al arrancar y cerrar al parar.
type ProxyLocal interface {
	Register(source models.VideoSource) models.VideoSource
	Escuchar() (string, error)
	Cerrar() error
}

// Resolutor elige la mejor fuente de una lista, sin red (T17:
// extractors.SelectBestSource).
type Resolutor func(fuentes []models.VideoSource) (models.VideoSource, bool)

// DetalleResultado es la respuesta de GetAnimeDetails: el detalle y la
// primera tanda de episodios que ya trae el provider. Los lotes siguientes
// llegan por el evento episodes:update. Es la única forma de devolver dos
// valores a Wails: BoundMethod.Call solo admite (valor, error).
type DetalleResultado struct {
	Detail   models.AnimeDetail `json:"detail"`
	Episodes []models.Episode   `json:"episodes"`
}

// EpisodiosCompletados es el payload de episodes:complete. Error es un
// string en español, vacío si todo fue bien: un error de Go nunca se
// serializa como payload (JSON lo dejaría vacío e inútil).
type EpisodiosCompletados struct {
	AnimeID string `json:"anime_id"`
	Error   string `json:"error"`
}

// streamActivo es el ÚNICO stream de detalle que corre a la vez: su clave de
// caché, su contexto (para saber si sigue vivo) y su cancel. Vive bajo el
// mutex del núcleo.
//
// fallo marca que el lote final (Done) traía Err: la entrada de caché ya se
// borró y detalle() no debe recrearla con el detalle que devuelve el provider.
// Es atómico porque lo escribe la goruta que recibe los lotes y lo lee
// detalle().
type streamActivo struct {
	clave  string
	ctx    context.Context
	cancel context.CancelFunc
	fallo  atomic.Bool
}

// entradaHome es una fila del caché de Home (T45): las secciones que devolvió
// un proveedor y cuándo caducan (mismo TTL deslizante que el detalle).
type entradaHome struct {
	secciones []models.HomeSection
	caducaEn  time.Time
}

// Nucleo es la lógica de orquestación: proveedores, eventos, proxy, la
// caché local de detalle, el Home, los favoritos y el historial de
// visualización. Vive en internal/app —fuera de app.go y de Wails— para
// poder testearla sin Wails ni red: todas sus dependencias entran por
// interfaz o por inyección y el emisor es una función inyectada.
type Nucleo struct {
	registro RegistroProviders
	proxy    ProxyLocal
	mejor    Resolutor
	// cacheLocal guarda detalle+episodios por clave proveedor:animeID (T21).
	cacheLocal *cache.Cache
	// rutaHistorial es el directorio del historial (T22): lo decide quien
	// monta el núcleo (app.go con os.UserConfigDir()/miapp); un "" haría
	// que db.Open devolviera ErrDirectorioVacio y la app siguiera sin
	// historial. Los tests inyectan t.TempDir(). Solo la lee arrancar.
	rutaHistorial string
	// historial se abre en arrancar y se cierra en cerrar; nil mientras no
	// abra (antes de arrancar o si su apertura falló). cerrar NO lo borra,
	// así que un guardado tardío ve db.ErrCerrado y no una carrera.
	historial *db.DB
	// favoritos se abre en arrancar con db.OpenFavorites sobre el MISMO
	// directorio que el historial (T42/T45); nil mientras no abra (antes de
	// arrancar, fallo de apertura o núcleo sin directorio de datos). No se
	// cierra en cerrar: db.Favorites no tiene Close (cada Add/Remove ya
	// escribió a disco).
	favoritos *db.Favorites

	// muHome protege home: las secciones del Home por proveedor (T45). Es un
	// caché PROPIO del núcleo con la misma clave (cache.Clave("home", p)) y
	// el mismo TTL que el detalle: internal/cache solo admite entradas
	// detalle+episodios y no puede guardar []models.HomeSection (desviación
	// anotada en PROGRESO.md y en docs/firmas/app.md).
	muHome sync.Mutex
	home   map[string]entradaHome

	mu sync.Mutex
	// emisor es el que empuja eventos al frontend (nil hasta arrancar).
	emisor Emisor
	// stream es el stream activo; cada stream nuevo lo sustituye —cancelando
	// el anterior solo si es de OTRO anime— y cerrar() cancela el que quede.
	stream *streamActivo
}

// NuevoNucleo crea el núcleo con las dependencias inyectadas. Una caché nil
// se sustituye por una por defecto (nunca debe llegar nil: rompería detalle).
// rutaHistorial es el directorio del historial de visualización (T22): vacío
// no es un valor especial (P3: db.ErrDirectorioVacio).
func NuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor, c *cache.Cache, rutaHistorial string) *Nucleo {
	if c == nil {
		c = cache.New(ttlCacheDetalle)
	}
	return &Nucleo{
		registro:      reg,
		proxy:         px,
		mejor:         sel,
		emisor:        emis,
		cacheLocal:    c,
		rutaHistorial: rutaHistorial,
		home:          map[string]entradaHome{},
	}
}

// NuevoNucleoPorDefecto monta las dependencias reales de la app:
//   - un único httpx.NewClient compartido por JKAnime y AnimeAV1 (mismo
//     cookiejar y timeout; el jar separa por host, así que no se mezclan
//     sesiones; AnimeAV1 además no necesita cookies, T31),
//   - el ExceptionResolver de JKAnime queda compartido por el proceso dentro
//     del propio JkAnime (la caché del mapa de excepciones es por instancia,
//     T06: una sola descarga como mucho),
//   - stream.NewProxy(nil): cliente propio del proxy con anti-SSRF y sin
//     cookies (si se inyectara el cliente de arriba se perdería el control de
//     red, decisión de T19),
//   - extractors.SelectBestSource como selector de la mejor fuente (T17),
//   - cache.New(ttlCacheDetalle) como caché local de detalle+episodios (T21);
//     su rutina de limpieza se arranca con arrancar y se para con cerrar,
//   - el historial de visualización (T22) y los favoritos (T42/T45) en el
//     directorio que calcule el llamador (en la app, app.go con
//     os.UserConfigDir()/miapp): ambos se abren en arrancar, en el mismo
//     directorio.
//
// Registro: jkanime + animeav1 (AnimeFLV se retiró en T35); la búsqueda
// consulta los dos en paralelo.
func NuevoNucleoPorDefecto(rutaHistorial string) *Nucleo {
	doer := httpx.NewClient(httpx.ClientOptions{})

	reg := scraper.NewRegistry()
	for _, p := range []scraper.Provider{
		jkanime.New(doer, "", nil),
		animeav1.New(doer, ""),
	} {
		if err := reg.Register(p); err != nil {
			slog.Error("no se pudo registrar un proveedor", "provider", p.Name(), "err", err)
		}
	}

	return NuevoNucleo(reg, stream.NewProxy(nil), extractors.SelectBestSource, nil,
		cache.New(ttlCacheDetalle), rutaHistorial)
}

// arrancar prepara el núcleo para servir: instala el emisor de eventos, pone
// a funcionar la limpieza de la caché local, abre el historial de
// visualización (T22) y los favoritos (T42, en el MISMO directorio) y el
// proxy a escuchar en 127.0.0.1 (sin Escuchar, Register devuelve las fuentes
// sin registrar, T19). Se llama desde App.startup.
//
// El historial y los favoritos son lo único cuyo fallo NO se devuelve: si no
// se pueden abrir (sin permisos, ruta inexistente...) se loguea y la app sigue
// viva sin ese almacén —guardar o leer progreso dará
// ErrHistorialNoDisponible, y los favoritos darán ErrFavoritosNoDisponible—,
// porque el proxy y la reproducción no dependen de ellos.
func (n *Nucleo) Arrancar(emis Emisor) error {
	n.mu.Lock()
	n.emisor = emis
	n.mu.Unlock()

	n.cacheLocal.Arrancar()

	if h, err := db.Open(n.rutaHistorial); err != nil {
		slog.Warn("la app sigue sin historial de visualización", "err", err)
	} else {
		n.mu.Lock()
		n.historial = h
		n.mu.Unlock()
	}
	if favs, err := db.OpenFavorites(n.rutaHistorial); err != nil {
		slog.Warn("la app sigue sin favoritos", "err", err)
	} else {
		n.mu.Lock()
		n.favoritos = favs
		n.mu.Unlock()
	}

	base, err := n.proxy.Escuchar()
	if err != nil {
		return fmt.Errorf("no pude arrancar el proxy local de streaming: %w", err)
	}
	slog.Info("proxy local de streaming listo", "base", base)
	return nil
}

// cerrar cancela el stream activo (si lo hay) —borrando su entrada de caché
// incompleta, que ya no tendrá stream que la complete—, detiene la limpieza
// de la caché (sin goroutines huérfanas), cierra el historial de
// visualización y para el proxy. Se llama desde App.shutdown, que registra
// OnShutdown en main.go. Es idempotente.
func (n *Nucleo) Cerrar() error {
	n.mu.Lock()
	st := n.stream
	n.stream = nil
	n.mu.Unlock()

	if st != nil {
		st.cancel()
		n.borrarSiIncompleta(st.clave)
	}
	n.cacheLocal.Cerrar()
	if h := n.historialAbierto(); h != nil {
		if err := h.Close(); err != nil {
			slog.Warn("no pude cerrar el historial de visualización", "err", err)
		}
	}
	return n.proxy.Cerrar()
}

// buscar resuelve la búsqueda COMBINADA (T34): JKAnime y AnimeAV1 se
// consultan EN PARALELO con un tiempo límite (tiempoLimiteBusqueda) sobre el
// ctx recibido, y los resultados se devuelven JKAnime primero y AnimeAV1
// después (cada uno ya trae su provider). Se tolera que uno falle: su fallo
// se loguea con slog.Warn (nunca con URLs firmadas) y sigue habiendo
// respuesta con lo del resto. Solo es error si fallan TODOS los proveedores
// registrados; si al menos uno pudo buscar, la lista —aunque venga vacía— no
// es error. Sin proveedores registrados devuelve lista vacía sin error.
func (n *Nucleo) Buscar(ctx context.Context, query string) ([]models.Anime, error) {
	q := strings.TrimSpace(query)
	nombres := []string{proveedorPrincipal, proveedorComplemento}

	// Cada proveedor escribe en su propia casilla: los índices no se pisan
	// y WaitGroup espera a los dos antes de leer.
	type respuesta struct {
		resultados []models.Anime
		err        error
	}
	respuestas := make([]respuesta, len(nombres))
	registrado := make([]bool, len(nombres))

	ctxBusqueda, cancel := context.WithTimeout(ctx, tiempoLimiteBusqueda)
	defer cancel()

	var wg sync.WaitGroup
	for i, nombre := range nombres {
		p, ok := n.registro.Get(nombre)
		if !ok || p == nil {
			continue // un proveedor no registrado no es un fallo
		}
		registrado[i] = true
		wg.Add(1)
		go func(i int, p scraper.Provider) {
			defer wg.Done()
			resultados, err := p.Search(ctxBusqueda, q)
			respuestas[i] = respuesta{resultados: resultados, err: err}
		}(i, p)
	}
	wg.Wait()

	var fallos []error
	salida := make([]models.Anime, 0, len(respuestas))
	pudoBuscar := false
	for i, nombre := range nombres {
		if !registrado[i] {
			continue
		}
		r := respuestas[i]
		if r.err != nil {
			fallos = append(fallos, fmt.Errorf("%s: %w", nombre, r.err))
			slog.Warn("la búsqueda de un proveedor falló; se conservan los resultados del resto",
				"provider", nombre, "err", r.err)
			continue
		}
		pudoBuscar = true
		salida = append(salida, r.resultados...)
	}

	if pudoBuscar || len(fallos) == 0 {
		return salida, nil
	}
	return nil, fmt.Errorf("no se pudo buscar %q: %w", q, errors.Join(fallos...))
}

// detalle devuelve el detalle de un anime y su primera tanda de episodios.
//
// Antes de tocar al proveedor consulta la caché local (clave proveedor:anime):
//   - entrada COMPLETA → se devuelve todo y se emite episodes:complete de
//     inmediato (si había otro anime en curso se cancela: cambio de anime);
//   - entrada INCOMPLETA con su stream vivo → se devuelve lo acumulado y se
//     ADJUNTA al stream en curso: no se cancela ni se lanza otro stream para
//     el mismo anime (esto reemplaza la regla de T20 de cancelar también al
//     repetir el mismo anime);
//   - entrada incompleta sin stream vivo → es huérfana: se borra y se pide
//     de nuevo.
//
// Al fallar un stream se borra su entrada: los errores no se cachean. Si el
// provider implementa scraper.EpisodeStreamer se usa ese flujo y los lotes
// siguientes salen por episodes:update / episodes:complete; si no, se pide el
// detalle completo de forma síncrona y NO se cachea (solo se cachea el flujo
// por lotes). scraper.ErrNoDetails no es un fallo grave (ningún proveedor
// registrado lo devuelve hoy): se traduce en ErrDetalleNoDisponible y se
// registra a nivel info.
func (n *Nucleo) Detalle(ctx context.Context, proveedor, id string) (DetalleResultado, error) {
	proveedor = strings.TrimSpace(proveedor)
	p, err := n.proveedor(proveedor)
	if err != nil {
		return DetalleResultado{}, err
	}
	id = strings.TrimSpace(id)
	clave := cache.Clave(proveedor, id)

	if e, ok := n.cacheLocal.Get(clave); ok {
		if e.Complete {
			n.soltarStreamDistinto(clave)
			n.emitirCompleto(ctx, id, "")
			return detalleDesdeCache(e), nil
		}
		if st := n.streamDe(clave); st != nil && st.ctx.Err() == nil {
			return detalleDesdeCache(e), nil
		}
		// Entrada huérfana: su stream ya murió sin limpiar. Nada debe
		// esperar a un stream muerto, así que se borra y se pide de nuevo.
		n.cacheLocal.Delete(clave)
	}

	// Falta: stream nuevo. Cancela el de OTRO anime (y borra su entrada
	// incompleta); para el mismo anime esto no se llega a ejecutar nunca.
	streamCtx, st := n.iniciarStream(ctx, clave)

	if estream, ok := p.(scraper.EpisodeStreamer); ok {
		det, eps, err := estream.StreamAnimeDetails(streamCtx, id, n.lote(streamCtx, st, id))
		if err != nil {
			st.cancel()
			n.liberarStream(st)
			n.cacheLocal.Delete(clave)
			return DetalleResultado{}, errorDeDetalle(proveedor, err)
		}
		// El stream pudo haber cerrado con error (lote Done con Err) mientras
		// volvía esta llamada: lote() ya borró la entrada y no se recrea.
		if st.fallo.Load() {
			st.cancel()
			n.liberarStream(st)
			n.cacheLocal.Delete(clave)
			return DetalleResultado{Detail: det, Episodes: eps}, nil
		}
		// Solo el detalle y la primera tanda: los lotes que hayan llegado
		// mientras volvía esta llamada ya están en la entrada (Append no
		// pisa el detalle y Set no pisa los episodios, así que el orden de
		// estas dos escrituras da igual).
		n.cacheLocal.Set(clave, det)
		n.cacheLocal.Append(clave, eps)
		// Relectura tras escribir: si el cierre con Err llegó en paralelo,
		// aquí no queda nada de ese stream.
		if st.fallo.Load() {
			n.cacheLocal.Delete(clave)
		}
		return DetalleResultado{Detail: det, Episodes: eps}, nil
	}

	det, eps, err := p.GetAnimeDetails(streamCtx, id)
	st.cancel() // flujo síncrono: el ctx ya no hace falta
	n.liberarStream(st)
	return DetalleResultado{Detail: det, Episodes: eps}, errorDeDetalle(proveedor, err)
}

// fuentes pide las fuentes de un episodio, las registra en el proxy local
// (solo HLS y MP4: los iframes NO pasan por el proxy) y devuelve la lista
// con la mejor fuente PRIMERO (SelectBestSource, T17).
//
// noCache se acepta y se reenvía al provider, pero NO lo consume nadie: la
// caché local (T21) guarda detalle+episodios y NUNCA GetVideoSources (los
// enlaces son temporales, regla 9), y T17 lo ignora. Se conserva la opción
// para no romper la firma que consumirá T27.
func (n *Nucleo) Fuentes(ctx context.Context, proveedor, episodeID string, noCache bool) ([]models.VideoSource, error) {
	p, err := n.proveedor(proveedor)
	if err != nil {
		return nil, err
	}
	fuentes, err := p.GetVideoSources(ctx, episodeID, scraper.SourceOpts{NoCache: noCache})
	if err != nil {
		return nil, err
	}
	if len(fuentes) == 0 {
		return nil, fmt.Errorf("el proveedor %q no devolvió fuentes de vídeo para %q", proveedor, episodeID)
	}
	if n.mejor == nil {
		return nil, errors.New("el núcleo no tiene selector de fuentes (Resolutor)")
	}

	// La mejor se elige sobre las originales; después se registra cada una
	// en el proxy (una sola vez) y se reordena con la mejor en cabeza.
	mejor, ok := n.mejor(fuentes)
	if !ok {
		return nil, fmt.Errorf("no hay fuentes de vídeo reproducibles para %q", episodeID)
	}
	idx := indiceDe(fuentes, mejor)

	proxied := make([]models.VideoSource, len(fuentes))
	for i, f := range fuentes {
		if f.Type == models.ResourceHLS || f.Type == models.ResourceMP4 {
			proxied[i] = n.proxy.Register(f)
			continue
		}
		proxied[i] = f
	}
	if idx < 0 {
		// No debería pasar (mejor es copia de una entrada): se conserva el
		// orden original en lugar de duplicar una fuente.
		return proxied, nil
	}

	salida := make([]models.VideoSource, 0, len(proxied))
	salida = append(salida, proxied[idx])
	for i, f := range proxied {
		if i == idx {
			continue
		}
		salida = append(salida, f)
	}
	return salida, nil
}

// guardarProgreso delega en el historial de visualización (SaveProgress,
// T22). El núcleo solo aporta el almacén abierto en arrancar: el cálculo de
// IsWatched y la escritura atómica son del paquete db.
func (n *Nucleo) GuardarProgreso(progress models.WatchProgress) error {
	h := n.historialAbierto()
	if h == nil {
		return ErrHistorialNoDisponible
	}
	return h.SaveProgress(progress)
}

// verProgreso devuelve el historial guardado de un anime (animeID vacío =
// entero); ver db.DB.GetProgress para el orden y el resto de contratos.
func (n *Nucleo) VerProgreso(animeID string) ([]models.WatchProgress, error) {
	h := n.historialAbierto()
	if h == nil {
		return nil, ErrHistorialNoDisponible
	}
	return h.GetProgress(animeID)
}

// HistorialReciente devuelve el ÚLTIMO registro de cada anime (LastPerAnime,
// T41): es lo que pinta la barra lateral y el frontend lo filtra por
// proveedor con el prefijo del episode_id (T34). Sin almacén abierto da
// ErrHistorialNoDisponible; tras cerrar el almacén devuelve lista vacía sin
// error (decisión de T41: db.LastPerAnime no admite error), y aquí no se
// traduce en error.
func (n *Nucleo) HistorialReciente() ([]models.WatchProgress, error) {
	h := n.historialAbierto()
	if h == nil {
		return nil, ErrHistorialNoDisponible
	}
	return h.LastPerAnime(), nil
}

// BorrarHistorialAnime borra TODOS los episodios guardados de un anime y
// devuelve el nº de registros eliminados (0 si no había). proveedor y
// animeID no pueden venir vacíos —la nota de T41: un "" llegaría a
// db.DeleteAnime tal cual—, así que el error sale aquí, en español y sin
// llamar a db, para no borrar de más.
func (n *Nucleo) BorrarHistorialAnime(proveedor, animeID string) (int, error) {
	proveedor = strings.TrimSpace(proveedor)
	animeID = strings.TrimSpace(animeID)
	if proveedor == "" || animeID == "" {
		return 0, fmt.Errorf("borrar historial necesita proveedor y anime: provider=%q animeID=%q", proveedor, animeID)
	}
	h := n.historialAbierto()
	if h == nil {
		return 0, ErrHistorialNoDisponible
	}
	return h.DeleteAnime(proveedor, animeID)
}

// BorrarHistorialProveedor borra todos los registros de un proveedor
// (papelera del encabezado de la barra lateral); mismas reglas de
// validación que BorrarHistorialAnime.
func (n *Nucleo) BorrarHistorialProveedor(proveedor string) (int, error) {
	proveedor = strings.TrimSpace(proveedor)
	if proveedor == "" {
		return 0, fmt.Errorf("borrar historial necesita un proveedor: provider=%q", proveedor)
	}
	h := n.historialAbierto()
	if h == nil {
		return 0, ErrHistorialNoDisponible
	}
	return h.DeleteProvider(proveedor)
}

// BorrarHistorialTodo vacía el historial entero y devuelve el nº de
// registros eliminados (el frontend pide doble confirmación antes).
func (n *Nucleo) BorrarHistorialTodo() (int, error) {
	h := n.historialAbierto()
	if h == nil {
		return 0, ErrHistorialNoDisponible
	}
	return h.Clear()
}

// historialAbierto devuelve el almacén bajo el mutex del núcleo. Es nil
// antes de arrancar o si su apertura falló; cerrar no lo borra, así que
// después del shutdown la llamada llega al almacén y devuelve db.ErrCerrado
// sin carrera con arrancar.
func (n *Nucleo) historialAbierto() *db.DB {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.historial
}

// Home devuelve las secciones del Home del proveedor (T45): pide a
// scraper.HomeProvider —interfaz opcional, T43— y cachea el resultado con la
// misma vida que el detalle (ttlCacheDetalle, 10 min) bajo la clave
// cache.Clave("home", proveedor). Los errores no se cachean; noCache salta
// la lectura y refresca la entrada al terminar bien. Las secciones salen
// COPIADAS (la caché y el llamador no comparten memoria). Un proveedor sin
// HomeProvider devuelve ErrHomeNoDisponible; uno desconocido, el mismo error
// que ya devuelve Detalle.
func (n *Nucleo) Home(ctx context.Context, proveedor string, noCache bool) ([]models.HomeSection, error) {
	proveedor = strings.TrimSpace(proveedor)
	p, err := n.proveedor(proveedor)
	if err != nil {
		return nil, err
	}
	hp, ok := p.(scraper.HomeProvider)
	if !ok {
		return nil, ErrHomeNoDisponible
	}

	clave := cache.Clave("home", proveedor)
	if !noCache {
		if secciones, ok := n.homeDe(clave); ok {
			return secciones, nil
		}
	}
	secciones, err := hp.Home(ctx)
	if err != nil {
		return nil, err // los errores no se cachean (T45)
	}
	n.ponerHome(clave, secciones)
	return copiaHome(secciones), nil
}

// homeDe lee la caché del Home bajo muHome: false si no hay entrada o si ya
// caducó (la lectura no la renueva: el TTL es por deslizamiento desde la
// escritura, igual que el detalle).
func (n *Nucleo) homeDe(clave string) ([]models.HomeSection, bool) {
	n.muHome.Lock()
	defer n.muHome.Unlock()
	e, ok := n.home[clave]
	if !ok || time.Now().After(e.caducaEn) {
		return nil, false
	}
	return copiaHome(e.secciones), true
}

// ponerHome guarda una copia de las secciones con el TTL del detalle.
func (n *Nucleo) ponerHome(clave string, secciones []models.HomeSection) {
	n.muHome.Lock()
	defer n.muHome.Unlock()
	n.home[clave] = entradaHome{
		secciones: copiaHome(secciones),
		caducaEn:  time.Now().Add(ttlCacheDetalle),
	}
}

// AgregarFavorito marca un anime como favorito (T42/T45): valida que
// proveedor y animeID no estén vacíos y que el proveedor esté registrado
// (la clave del favorito es proveedor+slug, T40: un proveedor inventado
// solo ensuciaría la pestaña Favoritos). Add es idempotente, nunca toggle
// (db.Favorites.Add).
func (n *Nucleo) AgregarFavorito(f models.Favorite) error {
	f.Provider = strings.TrimSpace(f.Provider)
	f.AnimeID = strings.TrimSpace(f.AnimeID)
	if f.Provider == "" || f.AnimeID == "" {
		return fmt.Errorf("el favorito necesita proveedor y anime: provider=%q animeID=%q", f.Provider, f.AnimeID)
	}
	if _, err := n.proveedor(f.Provider); err != nil {
		return err
	}
	fav := n.favoritosAbierto()
	if fav == nil {
		return ErrFavoritosNoDisponible
	}
	_, err := fav.Add(f)
	return err
}

// QuitarFavorito quita la estrella; quitar un favorito que no estaba no es
// un error (el estado ya es el pedido).
func (n *Nucleo) QuitarFavorito(proveedor, animeID string) error {
	fav := n.favoritosAbierto()
	if fav == nil {
		return ErrFavoritosNoDisponible
	}
	_, err := fav.Remove(strings.TrimSpace(proveedor), strings.TrimSpace(animeID))
	return err
}

// ListarFavoritos devuelve los favoritos en copias y orden AddedAt
// descendente (db.Favorites.List); nunca nil.
func (n *Nucleo) ListarFavoritos() ([]models.Favorite, error) {
	fav := n.favoritosAbierto()
	if fav == nil {
		return nil, ErrFavoritosNoDisponible
	}
	return fav.List(), nil
}

// EsFavorito dice si ese anime está marcado con la estrella; sin almacén
// devuelve error (el frontend no debe pintar una estrella apagada como si
// fuera verdad).
func (n *Nucleo) EsFavorito(proveedor, animeID string) (bool, error) {
	fav := n.favoritosAbierto()
	if fav == nil {
		return false, ErrFavoritosNoDisponible
	}
	return fav.Has(strings.TrimSpace(proveedor), strings.TrimSpace(animeID)), nil
}

// favoritosAbierto devuelve el almacén de favoritos bajo el mutex del
// núcleo: nil antes de arrancar o si su apertura falló. db.Favorites no
// tiene Close, así que cerrar no lo invalida (a diferencia del historial).
func (n *Nucleo) favoritosAbierto() *db.Favorites {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.favoritos
}

// copiaHome copia las secciones y sus items (models.Anime es un valor
// plano, sin punteros compartidos): la caché y el llamador nunca comparten
// memoria.
func copiaHome(secciones []models.HomeSection) []models.HomeSection {
	if secciones == nil {
		return nil
	}
	out := make([]models.HomeSection, len(secciones))
	for i, s := range secciones {
		out[i] = models.HomeSection{ID: s.ID, Title: s.Title}
		if s.Items != nil {
			out[i].Items = make([]models.Anime, len(s.Items))
			copy(out[i].Items, s.Items)
		}
	}
	return out
}

// proveedor busca un provider por nombre; si no existe, devuelve un error
// claro en español con los disponibles.
func (n *Nucleo) proveedor(nombre string) (scraper.Provider, error) {
	nombre = strings.TrimSpace(nombre)
	if p, ok := n.registro.Get(nombre); ok && p != nil {
		return p, nil
	}
	return nil, fmt.Errorf("proveedor desconocido: %q (disponibles: %s)", nombre, n.nombres())
}

// iniciarStream crea el contexto del stream nuevo y lo registra como el
// activo. Si había otro —de OTRO anime— lo cancela y borra su entrada de
// caché incompleta (huérfana: ya no hay stream que la complete). Si el
// anterior es del MISMO anime no se llega a llamar: detalle() se habría
// adjuntado a su stream en la comprobación de la caché.
func (n *Nucleo) iniciarStream(base context.Context, clave string) (context.Context, *streamActivo) {
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithCancel(base)
	st := &streamActivo{clave: clave, ctx: ctx, cancel: cancel}

	n.mu.Lock()
	anterior := n.stream
	n.stream = st
	n.mu.Unlock()

	if anterior != nil {
		anterior.cancel()
		n.borrarSiIncompleta(anterior.clave)
	}
	return ctx, st
}

// soltarStreamDistinto cancela el stream activo solo si es de un anime
// distinto al dado (cambio de anime) y borra su entrada incompleta.
func (n *Nucleo) soltarStreamDistinto(clave string) {
	n.mu.Lock()
	st := n.stream
	n.mu.Unlock()

	if st == nil || st.clave == clave {
		return
	}
	st.cancel()
	n.liberarStream(st)
	n.borrarSiIncompleta(st.clave)
}

// streamDe devuelve el stream activo si es el de esa clave.
func (n *Nucleo) streamDe(clave string) *streamActivo {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stream != nil && n.stream.clave == clave {
		return n.stream
	}
	return nil
}

// liberarStream quita de la lista el stream que ya terminó, solo si sigue
// siendo el registrado (otro pudo haberlo sustituido ya). No cancela nada:
// ese trabajo es de quien lo sustituye o de cerrar().
func (n *Nucleo) liberarStream(st *streamActivo) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stream == st {
		n.stream = nil
	}
}

// borrarSiIncompleta elimina la entrada de la caché solo si existe y todavía
// no está completa: así un stream cancelado no deja nada esperándolo, pero
// una lista ya terminada se conserva para el próximo hit.
func (n *Nucleo) borrarSiIncompleta(clave string) {
	if e, ok := n.cacheLocal.Get(clave); ok && !e.Complete {
		n.cacheLocal.Delete(clave)
	}
}

// lote devuelve el callback con el que el provider entrega cada lote de
// episodios:
//   - los episodios se acumulan en la caché (deduplicados por ID) ANTES de
//     emitir; un lote con Err no se cachea y, si cierra el stream con error,
//     se borra la entrada entera y se marca el stream como fallado (los
//     errores no se cachean, T21);
//   - si el lote cierra el stream (Done) sin error, la entrada queda
//     marcada como completa;
//   - cada lote se emite como "episodes:update" con models.EpisodeBatch
//     (anime_id, batch, episodes, done, err — tags snake_case de T03);
//   - si el lote cierra el stream (Done) se emite además "episodes:complete"
//     con EpisodiosCompletados{anime_id, error}.
//
// Antes de CADA emisión se comprueba ctx.Err(): un stream cancelado (por
// pedir otro anime o por shutdown) no emite nada más. Si no hay emisor
// instalado (antes de startup) tampoco se emite.
func (n *Nucleo) lote(ctx context.Context, st *streamActivo, id string) func(models.EpisodeBatch) {
	clave := st.clave
	return func(lote models.EpisodeBatch) {
		if ctx.Err() != nil {
			return
		}

		animeID := lote.AnimeID
		if animeID == "" {
			animeID = id
		}
		lote.AnimeID = animeID

		if lote.Err == "" && len(lote.Episodes) > 0 {
			n.cacheLocal.Append(clave, lote.Episodes)
		}
		if lote.Done {
			if lote.Err != "" {
				// Se marca ANTES de borrar: detalle() puede estar escribiendo
				// el detalle en paralelo y debe verlo para no recrearla.
				st.fallo.Store(true)
				n.cacheLocal.Delete(clave)
			} else {
				n.cacheLocal.MarkComplete(clave)
			}
		}

		emis := n.emisorActual()
		if emis == nil {
			return
		}

		emis("episodes:update", lote)
		if !lote.Done {
			return
		}
		n.emitirCompleto(ctx, animeID, lote.Err)
	}
}

// emitirCompleto emite "episodes:complete" con EpisodiosCompletados si el
// ctx sigue vivo y hay emisor instalado (antes de startup no se emite).
// texto es el fallo del lote final: vacío = todo bien. Un error de Go nunca
// se serializa como payload.
func (n *Nucleo) emitirCompleto(ctx context.Context, animeID, texto string) {
	if ctx != nil && ctx.Err() != nil {
		return
	}
	emis := n.emisorActual()
	if emis == nil {
		return
	}
	emis("episodes:complete", EpisodiosCompletados{AnimeID: animeID, Error: texto})
}

// emisorActual devuelve el emisor bajo el mutex (nil si aún no arrancó).
func (n *Nucleo) emisorActual() Emisor {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.emisor
}

// nombres lista los proveedores registrados, para el error de proveedor
// desconocido.
func (n *Nucleo) nombres() string {
	todos := n.registro.All()
	if len(todos) == 0 {
		return "ninguno"
	}
	noms := make([]string, 0, len(todos))
	for _, p := range todos {
		noms = append(noms, p.Name())
	}
	return strings.Join(noms, ", ")
}

// errorDeDetalle traduce el error de un provider al lenguaje del frontend:
// scraper.ErrNoDetails no es un fallo grave (el proveedor simplemente no
// ofrece esa página) → ErrDetalleNoDisponible, con aviso a nivel info. El
// resto de errores se devuelve tal cual para conservar errors.Is/As.
func errorDeDetalle(proveedor string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, scraper.ErrNoDetails) {
		slog.Info("el proveedor no ofrece página de detalle", "provider", proveedor)
		return ErrDetalleNoDisponible
	}
	return err
}

// indiceDe localiza la fuente elegida dentro de la lista original por su
// URL (única tras la deduplicación de Normalize, T16) y su tipo.
func indiceDe(fuentes []models.VideoSource, mejor models.VideoSource) int {
	for i, f := range fuentes {
		if f.URL == mejor.URL && f.Type == mejor.Type {
			return i
		}
	}
	return -1
}

// detalleDesdeCache traduce una entrada a la respuesta de GetAnimeDetails.
// Los episodios salen ordenados por número: el orden de llegada puede
// mezclarse (un lote puede emitirse antes de que el stream devuelva la
// página 1) y la caché solo garantiza que no haya duplicados. Los slices ya
// vienen copiados de Get, así que ordenarlos aquí no toca la caché.
func detalleDesdeCache(e cache.Entrada) DetalleResultado {
	if len(e.Episodes) > 1 {
		sort.SliceStable(e.Episodes, func(i, j int) bool {
			return e.Episodes[i].Number < e.Episodes[j].Number
		})
	}
	return DetalleResultado{Detail: e.Detail, Episodes: e.Episodes}
}
