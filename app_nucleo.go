package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

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

// nucleo es la lógica de orquestación: proveedores, eventos y proxy. Vive
// fuera de app.go para poder testearla sin Wails ni red: todas sus
// dependencias entran por interfaz y el emisor es una función inyectada.
type nucleo struct {
	registro RegistroProviders
	proxy    ProxyLocal
	mejor    Resolutor

	mu sync.Mutex
	// emisor es el que empuja eventos al frontend (nil hasta arrancar).
	emisor Emisor
	// cancel es el del ÚNICO stream activo; cada llamada a detalle lo
	// sustituye (cancelando el anterior) y cerrar() lo cancela.
	cancel context.CancelFunc
}

// nuevoNucleo crea el núcleo con las dependencias inyectadas.
func nuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor) *nucleo {
	return &nucleo{registro: reg, proxy: px, mejor: sel, emisor: emis}
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
//   - extractors.SelectBestSource como selector de la mejor fuente (T17).
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

	return nuevoNucleo(reg, stream.NewProxy(nil), extractors.SelectBestSource, nil)
}

// arrancar prepara el núcleo para servir: instala el emisor de eventos y
// pone el proxy a escuchar en 127.0.0.1 (sin Escuchar, Register devuelve
// las fuentes sin registrar, T19). Se llama desde App.startup.
func (n *nucleo) arrancar(emis Emisor) error {
	n.mu.Lock()
	n.emisor = emis
	n.mu.Unlock()

	base, err := n.proxy.Escuchar()
	if err != nil {
		return fmt.Errorf("no pude arrancar el proxy local de streaming: %w", err)
	}
	slog.Info("proxy local de streaming listo", "base", base)
	return nil
}

// cerrar cancela el stream activo (si lo hay) y para el proxy. Se llama
// desde App.shutdown, que registra OnShutdown en main.go.
func (n *nucleo) cerrar() error {
	n.mu.Lock()
	cancel := n.cancel
	n.cancel = nil
	n.mu.Unlock()

	if cancel != nil {
		cancel()
	}
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
// Cancela SIEMPRE el stream anterior antes de arrancar el nuevo (un solo
// stream activo, protegido con mutex): pedir otro anime —o el mismo otra
// vez— cierra el que había. Si el provider implementa scraper.EpisodeStreamer
// se usa ese flujo y los lotes siguientes salen por episodes:update /
// episodes:complete; si no, se pide el detalle completo de forma síncrona y
// no se emite ningún evento. scraper.ErrNoDetails (AnimeFLV) no es un fallo
// grave: se traduce en ErrDetalleNoDisponible y se registra a nivel info.
func (n *nucleo) detalle(ctx context.Context, proveedor, id string) (DetalleResultado, error) {
	p, err := n.proveedor(proveedor)
	if err != nil {
		return DetalleResultado{}, err
	}
	id = strings.TrimSpace(id)

	// Cancela el stream anterior y crea el ctx de este.
	streamCtx, cancel := n.iniciarStream(ctx)

	if estream, ok := p.(scraper.EpisodeStreamer); ok {
		det, eps, err := estream.StreamAnimeDetails(streamCtx, id, n.lote(streamCtx, id))
		if err != nil {
			cancel() // el flujo no arrancó: no queda nada corriendo
			return DetalleResultado{}, errorDeDetalle(proveedor, err)
		}
		// El cancel queda guardado: lo disparará la próxima llamada o cerrar.
		return DetalleResultado{Detail: det, Episodes: eps}, nil
	}

	det, eps, err := p.GetAnimeDetails(streamCtx, id)
	cancel() // flujo síncrono: el ctx ya no hace falta
	return DetalleResultado{Detail: det, Episodes: eps}, errorDeDetalle(proveedor, err)
}

// fuentes pide las fuentes de un episodio, las registra en el proxy local
// (solo HLS y MP4: los iframes NO pasan por el proxy) y devuelve la lista
// con la mejor fuente PRIMERO (SelectBestSource, T17).
//
// noCache se acepta y se reenvía al provider, pero hoy no lo consume nadie:
// no hay ninguna caché (regla 9) —la caché local es T21— y T17 ignora la
// opción. Se conserva para no romper la firma que consumirá T21/T27.
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

// iniciarStream cancela el stream activo —si lo hay— y crea el contexto del
// nuevo. Solo hay un cancel por stream activo y vive bajo el mutex.
func (n *nucleo) iniciarStream(base context.Context) (context.Context, context.CancelFunc) {
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithCancel(base)

	n.mu.Lock()
	anterior := n.cancel
	n.cancel = cancel
	n.mu.Unlock()

	if anterior != nil {
		anterior()
	}
	return ctx, cancel
}

// lote devuelve el callback con el que el provider entrega cada lote de
// episodios:
//   - cada lote se emite como "episodes:update" con models.EpisodeBatch
//     (anime_id, batch, episodes, done, err — tags snake_case de T03);
//   - si el lote cierra el stream (Done) se emite además "episodes:complete"
//     con EpisodiosCompletados{anime_id, error}.
//
// Antes de CADA emisión se comprueba ctx.Err(): un stream cancelado (por
// pedir otro anime o por shutdown) no emite nada más. Si no hay emisor
// instalado (antes de startup) tampoco se emite.
func (n *nucleo) lote(ctx context.Context, id string) func(models.EpisodeBatch) {
	return func(lote models.EpisodeBatch) {
		if ctx.Err() != nil {
			return
		}
		emis := n.emisorActual()
		if emis == nil {
			return
		}

		animeID := lote.AnimeID
		if animeID == "" {
			animeID = id
		}
		lote.AnimeID = animeID

		emis("episodes:update", lote)
		if !lote.Done {
			return
		}
		if ctx.Err() != nil {
			return
		}
		emis("episodes:complete", EpisodiosCompletados{AnimeID: animeID, Error: lote.Err})
	}
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
