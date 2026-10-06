package main

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
	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/animeflv"
	"miapp/internal/scraper/jkanime"
	"miapp/internal/scraper/jkanime/extractors"
	"miapp/internal/stream"
)

const (
	// proveedorPrincipal es el sitio que se consulta primero en la búsqueda.
	proveedorPrincipal = "jkanime"
	// proveedorFallback solo se usa si el principal falla o no devuelve nada:
	// AnimeFLV ofrece únicamente búsqueda (sin detalle ni fuentes, T18).
	proveedorFallback = "animeflv"

	// ttlCacheDetalle es la vida de cada entrada de la caché local (T21). Es
	// por deslizamiento: cada lote la renueva, así que una lista larga que
	// sigue bajando nunca caduca a mitad de camino.
	ttlCacheDetalle = 10 * time.Minute
)

// ErrDetalleNoDisponible es lo que ve el frontend cuando un proveedor no
// ofrece página de detalle (scraper.ErrNoDetails de AnimeFLV). Se devuelve
// como error tipado —no como fallo grave— con este mensaje ya pensado para
// la UI; el sentinel interno de scraper NO se arrastra, porque Wails entrega
// al frontend solo el texto y este dice algo útil.
var ErrDetalleNoDisponible = errors.New("Detalle no disponible para este proveedor")

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

// nucleo es la lógica de orquestación: proveedores, eventos, proxy y la
// caché local de detalle. Vive fuera de app.go para poder testearla sin Wails
// ni red: todas sus dependencias entran por interfaz o por inyección y el
// emisor es una función inyectada.
type nucleo struct {
	registro RegistroProviders
	proxy    ProxyLocal
	mejor    Resolutor
	// cacheLocal guarda detalle+episodios por clave proveedor:animeID (T21).
	cacheLocal *cache.Cache

	mu sync.Mutex
	// emisor es el que empuja eventos al frontend (nil hasta arrancar).
	emisor Emisor
	// stream es el stream activo; cada stream nuevo lo sustituye —cancelando
	// el anterior solo si es de OTRO anime— y cerrar() cancela el que quede.
	stream *streamActivo
}

// nuevoNucleo crea el núcleo con las dependencias inyectadas. Una caché nil
// se sustituye por una por defecto (nunca debe llegar nil: rompería detalle).
func nuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor, c *cache.Cache) *nucleo {
	if c == nil {
		c = cache.New(ttlCacheDetalle)
	}
	return &nucleo{registro: reg, proxy: px, mejor: sel, emisor: emis, cacheLocal: c}
}

// nuevoNucleoPorDefecto monta las dependencias reales de la app:
//   - un único httpx.NewClient para JKAnime y AnimeFLV (mismo cookiejar y
//     timeout; el jar separa por host, así que no se mezclan sesiones),
//   - el ExceptionResolver de JKAnime queda compartido por el proceso dentro
//     del propio JkAnime (la caché del mapa de excepciones es por instancia,
//     T06: una sola descarga como mucho),
//   - stream.NewProxy(nil): cliente propio del proxy con anti-SSRF y sin
//     cookies (si se inyectara el cliente de arriba se perdería el control de
//     red, decisión de T19),
//   - extractors.SelectBestSource como selector de la mejor fuente (T17),
//   - cache.New(ttlCacheDetalle) como caché local de detalle+episodios (T21);
//     su rutina de limpieza se arranca con arrancar y se para con cerrar.
func nuevoNucleoPorDefecto() *nucleo {
	doer := httpx.NewClient(httpx.ClientOptions{})

	reg := scraper.NewRegistry()
	for _, p := range []scraper.Provider{
		jkanime.New(doer, "", nil),
		animeflv.New(doer, ""),
	} {
		if err := reg.Register(p); err != nil {
			slog.Error("no se pudo registrar un proveedor", "provider", p.Name(), "err", err)
		}
	}

	return nuevoNucleo(reg, stream.NewProxy(nil), extractors.SelectBestSource, nil,
		cache.New(ttlCacheDetalle))
}

// arrancar prepara el núcleo para servir: instala el emisor de eventos, pone
// a funcionar la limpieza de la caché local y el proxy a escuchar en
// 127.0.0.1 (sin Escuchar, Register devuelve las fuentes sin registrar,
// T19). Se llama desde App.startup.
func (n *nucleo) arrancar(emis Emisor) error {
	n.mu.Lock()
	n.emisor = emis
	n.mu.Unlock()

	n.cacheLocal.Arrancar()

	base, err := n.proxy.Escuchar()
	if err != nil {
		return fmt.Errorf("no pude arrancar el proxy local de streaming: %w", err)
	}
	slog.Info("proxy local de streaming listo", "base", base)
	return nil
}

// cerrar cancela el stream activo (si lo hay) —borrando su entrada de caché
// incompleta, que ya no tendrá stream que la complete—, detiene la limpieza
// de la caché (sin goroutines huérfanas) y para el proxy. Se llama desde
// App.shutdown, que registra OnShutdown en main.go. Es idempotente.
func (n *nucleo) cerrar() error {
	n.mu.Lock()
	st := n.stream
	n.stream = nil
	n.mu.Unlock()

	if st != nil {
		st.cancel()
		n.borrarSiIncompleta(st.clave)
	}
	n.cacheLocal.Cerrar()
	return n.proxy.Cerrar()
}

// buscar resuelve la búsqueda con fallback: JKAnime primero y, si falla o
// devuelve la lista vacía, AnimeFLV (solo búsqueda). Devuelve error solo si
// NINGÚN proveedor pudo buscar; si al menos uno respondió y no hubo
// resultados, devuelve lista vacía sin error.
func (n *nucleo) buscar(ctx context.Context, query string) ([]models.Anime, error) {
	var fallos []error
	pudoBuscar := false

	for _, nombre := range []string{proveedorPrincipal, proveedorFallback} {
		p, ok := n.registro.Get(nombre)
		if !ok {
			continue
		}
		resultados, err := p.Search(ctx, strings.TrimSpace(query))
		if err != nil {
			fallos = append(fallos, fmt.Errorf("%s: %w", nombre, err))
			slog.Warn("la búsqueda falló; se prueba el siguiente proveedor",
				"provider", nombre, "err", err)
			continue
		}
		if len(resultados) > 0 {
			return resultados, nil
		}
		pudoBuscar = true
	}

	if pudoBuscar || len(fallos) == 0 {
		return []models.Anime{}, nil
	}
	return nil, fmt.Errorf("no se pudo buscar %q: %w", query, errors.Join(fallos...))
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
// por lotes). scraper.ErrNoDetails (AnimeFLV) no es un fallo grave: se
// traduce en ErrDetalleNoDisponible y se registra a nivel info.
func (n *nucleo) detalle(ctx context.Context, proveedor, id string) (DetalleResultado, error) {
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
func (n *nucleo) fuentes(ctx context.Context, proveedor, episodeID string, noCache bool) ([]models.VideoSource, error) {
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

// proveedor busca un provider por nombre; si no existe, devuelve un error
// claro en español con los disponibles.
func (n *nucleo) proveedor(nombre string) (scraper.Provider, error) {
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
func (n *nucleo) iniciarStream(base context.Context, clave string) (context.Context, *streamActivo) {
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
func (n *nucleo) soltarStreamDistinto(clave string) {
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
func (n *nucleo) streamDe(clave string) *streamActivo {
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
func (n *nucleo) liberarStream(st *streamActivo) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stream == st {
		n.stream = nil
	}
}

// borrarSiIncompleta elimina la entrada de la caché solo si existe y todavía
// no está completa: así un stream cancelado no deja nada esperándolo, pero
// una lista ya terminada se conserva para el próximo hit.
func (n *nucleo) borrarSiIncompleta(clave string) {
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
func (n *nucleo) lote(ctx context.Context, st *streamActivo, id string) func(models.EpisodeBatch) {
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
func (n *nucleo) emitirCompleto(ctx context.Context, animeID, texto string) {
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
func (n *nucleo) emisorActual() Emisor {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.emisor
}

// nombres lista los proveedores registrados, para el error de proveedor
// desconocido.
func (n *nucleo) nombres() string {
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
