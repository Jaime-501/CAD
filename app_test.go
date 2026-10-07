package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"miapp/internal/cache"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/jkanime/extractors"
)

// TestMain descarta los logs: los tests no deben ensuciar la salida.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Dobles: sin red, sin Wails.
// ---------------------------------------------------------------------------

// eventoGrabado es un evento que el núcleo habría enviado al frontend.
type eventoGrabado struct {
	Nombre string
	Datos  any
}

// emisorFalso graba los eventos emitidos (hace de runtime.EventsEmit).
type emisorFalso struct {
	mu      sync.Mutex
	eventos []eventoGrabado
	cambio  chan struct{}
}

func nuevoEmisorFalso() *emisorFalso {
	return &emisorFalso{cambio: make(chan struct{}, 1)}
}

func (e *emisorFalso) emitir(nombre string, data any) {
	e.mu.Lock()
	e.eventos = append(e.eventos, eventoGrabado{Nombre: nombre, Datos: data})
	e.mu.Unlock()
	select {
	case e.cambio <- struct{}{}:
	default:
	}
}

func (e *emisorFalso) lista() []eventoGrabado {
	e.mu.Lock()
	defer e.mu.Unlock()
	copia := make([]eventoGrabado, len(e.eventos))
	copy(copia, e.eventos)
	return copia
}

// esperar bloquea hasta que haya al menos n eventos (o falla el test).
func (e *emisorFalso) esperar(t *testing.T, n int) {
	t.Helper()
	limite := time.After(2 * time.Second)
	for {
		e.mu.Lock()
		hay := len(e.eventos)
		e.mu.Unlock()
		if hay >= n {
			return
		}
		select {
		case <-e.cambio:
		case <-limite:
			t.Fatalf("timeout esperando %d eventos; hay %d", n, hay)
		}
	}
}

// proxyFalso hace de stream.Proxy: cuenta las llamadas y devuelve una URL
// local, sin escuchar ni tocar la red.
type proxyFalso struct {
	base        string
	mu          sync.Mutex
	registradas []models.VideoSource
	escuchadas  int
	cerradas    int
}

func nuevoProxyFalso() *proxyFalso {
	return &proxyFalso{base: "http://127.0.0.1:39999"}
}

func (p *proxyFalso) Register(s models.VideoSource) models.VideoSource {
	// Como el proxy real (T29): la sesión se queda con el Referer/Headers
	// (aquí, en `registradas`) y la fuente devuelta sale sin ellos.
	p.mu.Lock()
	p.registradas = append(p.registradas, s)
	p.mu.Unlock()
	s.URL = p.base + "/proxy?s=falsa&u=" + url.QueryEscape(s.URL)
	s.Referer = ""
	s.Headers = nil
	return s
}

func (p *proxyFalso) Escuchar() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.escuchadas++
	return p.base, nil
}

func (p *proxyFalso) Cerrar() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cerradas++
	return nil
}

func (p *proxyFalso) contadores() (registradas, escuchadas, cerradas int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.registradas), p.escuchadas, p.cerradas
}

// proveedorFalso implementa scraper.Provider con funciones inyectables y
// registra las operaciones que recibe.
type proveedorFalso struct {
	nombre string

	buscarF  func(ctx context.Context, query string) ([]models.Anime, error)
	detalleF func(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error)
	fuentesF func(ctx context.Context, episodeID string, opts scraper.SourceOpts) ([]models.VideoSource, error)

	mu         sync.Mutex
	ops        []string
	ultimoOpts scraper.SourceOpts
}

func (p *proveedorFalso) Name() string { return p.nombre }

func (p *proveedorFalso) Search(ctx context.Context, query string) ([]models.Anime, error) {
	p.registrar("Search")
	if p.buscarF != nil {
		return p.buscarF(ctx, query)
	}
	return []models.Anime{}, nil
}

func (p *proveedorFalso) GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	p.registrar("GetAnimeDetails")
	if p.detalleF != nil {
		return p.detalleF(ctx, id)
	}
	return models.AnimeDetail{}, nil, nil
}

func (p *proveedorFalso) GetVideoSources(ctx context.Context, episodeID string, opts scraper.SourceOpts) ([]models.VideoSource, error) {
	p.registrar("GetVideoSources")
	p.mu.Lock()
	p.ultimoOpts = opts
	p.mu.Unlock()
	if p.fuentesF != nil {
		return p.fuentesF(ctx, episodeID, opts)
	}
	return nil, nil
}

func (p *proveedorFalso) registrar(op string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ops = append(p.ops, op)
}

func (p *proveedorFalso) operaciones() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	copia := make([]string, len(p.ops))
	copy(copia, p.ops)
	return copia
}

func (p *proveedorFalso) optsRecibidos() scraper.SourceOpts {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ultimoOpts
}

// streamerFalso añade scraper.EpisodeStreamer al doble anterior.
type streamerFalso struct {
	proveedorFalso
	streamF func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error)
}

func (s *streamerFalso) StreamAnimeDetails(ctx context.Context, id string,
	onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
	s.registrar("StreamAnimeDetails")
	return s.streamF(ctx, id, onBatch)
}

// entorno junta el núcleo con sus dobles.
type entorno struct {
	n     *nucleo
	proxy *proxyFalso
	emis  *emisorFalso
	cache *cache.Cache
	// dirHistorial es el directorio inyectado para el historial de T22:
	// arrancar lo abre ahí (nunca en la configuración real del usuario).
	dirHistorial string
}

func nuevoEntorno(t *testing.T, ps ...scraper.Provider) *entorno {
	t.Helper()
	return nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), ps...)
}

// nuevoEntornoConCache monta el núcleo con una caché inyectada (los tests de
// T21 necesitan controlar el reloj o leer la caché directamente). El cleanup
// cierra el núcleo: así ninguna rutina de limpieza de la caché sobrevive al
// test (sin goroutines huérfanas) y el historial abierto queda cerrado.
func nuevoEntornoConCache(t *testing.T, c *cache.Cache, ps ...scraper.Provider) *entorno {
	t.Helper()
	reg := scraper.NewRegistry()
	for _, p := range ps {
		if err := reg.Register(p); err != nil {
			t.Fatalf("no pude registrar el doble %q: %v", p.Name(), err)
		}
	}
	emis := nuevoEmisorFalso()
	px := nuevoProxyFalso()
	dirHistorial := t.TempDir()
	e := &entorno{
		n:            nuevoNucleo(reg, px, extractors.SelectBestSource, emis.emitir, c, dirHistorial),
		proxy:        px,
		emis:         emis,
		cache:        c,
		dirHistorial: dirHistorial,
	}
	t.Cleanup(func() { _ = e.n.cerrar() })
	return e
}

// ---------------------------------------------------------------------------
// Ayudas de aserción.
// ---------------------------------------------------------------------------

func loteDe(t *testing.T, ev eventoGrabado) models.EpisodeBatch {
	t.Helper()
	lote, ok := ev.Datos.(models.EpisodeBatch)
	if !ok {
		t.Fatalf("el payload de %q no es models.EpisodeBatch: %T", ev.Nombre, ev.Datos)
	}
	return lote
}

func completoDe(t *testing.T, ev eventoGrabado) EpisodiosCompletados {
	t.Helper()
	c, ok := ev.Datos.(EpisodiosCompletados)
	if !ok {
		t.Fatalf("el payload de %q no es EpisodiosCompletados: %T", ev.Nombre, ev.Datos)
	}
	return c
}

func lotesDe(t *testing.T, eventos []eventoGrabado, animeID string) []models.EpisodeBatch {
	t.Helper()
	var out []models.EpisodeBatch
	for _, ev := range eventos {
		if ev.Nombre != "episodes:update" {
			continue
		}
		l := loteDe(t, ev)
		if l.AnimeID == animeID {
			out = append(out, l)
		}
	}
	return out
}

func completadosDe(t *testing.T, eventos []eventoGrabado, animeID string) []EpisodiosCompletados {
	t.Helper()
	var out []EpisodiosCompletados
	for _, ev := range eventos {
		if ev.Nombre != "episodes:complete" {
			continue
		}
		c := completoDe(t, ev)
		if c.AnimeID == animeID {
			out = append(out, c)
		}
	}
	return out
}

// destinoProxy lee el parámetro u de una URL del proxy (si no lo lleva,
// devuelve la URL tal cual).
func destinoProxy(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("URL no parseable: %q", raw)
	}
	destino := u.Query().Get("u")
	if destino == "" {
		return raw
	}
	return destino
}

// ---------------------------------------------------------------------------
// Búsqueda con fallback.
// ---------------------------------------------------------------------------

func TestBusquedaJKAnimePrimeroSinLlamarAlFallback(t *testing.T) {
	principal := &proveedorFalso{
		nombre: "jkanime",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return []models.Anime{{ID: "bleach", Title: "Bleach", Provider: "jkanime"}}, nil
		},
	}
	fallback := &proveedorFalso{
		nombre: "animeflv",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return []models.Anime{{ID: "no-deberia-llamarse"}}, nil
		},
	}
	e := nuevoEntorno(t, principal, fallback)

	resultados, err := e.n.buscar(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if len(resultados) != 1 || resultados[0].ID != "bleach" {
		t.Fatalf("resultados = %+v; quería los de jkanime", resultados)
	}
	if ops := fallback.operaciones(); len(ops) != 0 {
		t.Errorf("el fallback no debe llamarse con resultados del principal; ops = %v", ops)
	}
}

func TestBusquedaFallbackCuandoElPrincipalNoDevuelveNada(t *testing.T) {
	principal := &proveedorFalso{nombre: "jkanime"}
	fallback := &proveedorFalso{
		nombre: "animeflv",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return []models.Anime{{ID: "bleach-sennen", Provider: "animeflv"}}, nil
		},
	}
	e := nuevoEntorno(t, principal, fallback)

	resultados, err := e.n.buscar(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if len(resultados) != 1 || resultados[0].Provider != "animeflv" {
		t.Fatalf("resultados = %+v; quería los de animeflv", resultados)
	}
	if ops := principal.operaciones(); len(ops) != 1 || ops[0] != "Search" {
		t.Errorf("el principal debe buscarse una vez; ops = %v", ops)
	}
}

func TestBusquedaFallbackCuandoElPrincipalFalla(t *testing.T) {
	principal := &proveedorFalso{
		nombre: "jkanime",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return nil, errors.New("jkanime caído")
		},
	}
	fallback := &proveedorFalso{
		nombre: "animeflv",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return []models.Anime{{ID: "bleach-sennen", Provider: "animeflv"}}, nil
		},
	}
	e := nuevoEntorno(t, principal, fallback)

	resultados, err := e.n.buscar(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("el fallback debe tapar el fallo del principal: %v", err)
	}
	if len(resultados) != 1 || resultados[0].Provider != "animeflv" {
		t.Fatalf("resultados = %+v", resultados)
	}
}

func TestBusquedaAmbosFallanDevuelveErrorEnEspanol(t *testing.T) {
	principal := &proveedorFalso{
		nombre: "jkanime",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return nil, errors.New("fallo del principal")
		},
	}
	fallback := &proveedorFalso{
		nombre: "animeflv",
		buscarF: func(context.Context, string) ([]models.Anime, error) {
			return nil, errors.New("fallo del fallback")
		},
	}
	e := nuevoEntorno(t, principal, fallback)

	resultados, err := e.n.buscar(context.Background(), "bleach")
	if err == nil {
		t.Fatal("con los dos proveedores caídos debe devolver error")
	}
	if resultados != nil {
		t.Errorf("resultados = %+v; debía ser nil", resultados)
	}
	for _, texto := range []string{"jkanime", "animeflv", "fallo del principal", "fallo del fallback"} {
		if !strings.Contains(err.Error(), texto) {
			t.Errorf("el error no menciona %q: %v", texto, err)
		}
	}
}

func TestBusquedaSinResultadosNoEsError(t *testing.T) {
	e := nuevoEntorno(t,
		&proveedorFalso{nombre: "jkanime"},
		&proveedorFalso{nombre: "animeflv"},
	)

	resultados, err := e.n.buscar(context.Background(), "zzz-no-existe")
	if err != nil {
		t.Fatalf("sin resultados pero sin fallos no debe haber error: %v", err)
	}
	if len(resultados) != 0 {
		t.Errorf("resultados = %+v; debía estar vacío", resultados)
	}
}

func TestBusquedaSinProveedoresDevuelveVacio(t *testing.T) {
	e := nuevoEntorno(t)

	resultados, err := e.n.buscar(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("sin proveedores registrados no debe explotar: %v", err)
	}
	if len(resultados) != 0 {
		t.Errorf("resultados = %+v; debía estar vacío", resultados)
	}
}

// ---------------------------------------------------------------------------
// Detalle: proveedor desconocido, ErrNoDetails y streaming de lotes.
// ---------------------------------------------------------------------------

func TestDetalleProveedorDesconocidoDaErrorClaro(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: "jkanime"})

	_, err := e.n.detalle(context.Background(), "noexiste", "bleach")
	if err == nil {
		t.Fatal("un proveedor desconocido debe dar error")
	}
	for _, texto := range []string{"proveedor desconocido", "noexiste", "jkanime"} {
		if !strings.Contains(err.Error(), texto) {
			t.Errorf("el error no menciona %q: %v", texto, err)
		}
	}
	if eventos := e.emis.lista(); len(eventos) != 0 {
		t.Errorf("no debe emitirse nada: %+v", eventos)
	}
}

func TestDetalleErrNoDetailsDeAnimeFLVEsErrorTipado(t *testing.T) {
	flvs := &proveedorFalso{
		nombre: "animeflv",
		detalleF: func(context.Context, string) (models.AnimeDetail, []models.Episode, error) {
			return models.AnimeDetail{}, nil, scraper.ErrNoDetails
		},
	}
	e := nuevoEntorno(t, flvs)

	res, err := e.n.detalle(context.Background(), "animeflv", "bleach-sennen")
	if !errors.Is(err, ErrDetalleNoDisponible) {
		t.Fatalf("err = %v; quería ErrDetalleNoDisponible", err)
	}
	if errors.Is(err, scraper.ErrNoDetails) {
		t.Error("el sentinel interno de scraper no debe filtrarse al frontend")
	}
	if err.Error() != "Detalle no disponible para este proveedor" {
		t.Errorf("mensaje = %q; debe ser el pensado para la UI", err.Error())
	}
	if res.Detail.ID != "" || len(res.Episodes) != 0 {
		t.Errorf("resultado = %+v; debía venir vacío", res)
	}
	if eventos := e.emis.lista(); len(eventos) != 0 {
		t.Errorf("ErrNoDetails no es un fallo del stream: no debe emitir eventos: %+v", eventos)
	}
}

func TestDetalleSinEpisodeStreamerNoEmiteEventos(t *testing.T) {
	detalle := models.AnimeDetail{ID: "bleach", Provider: "animeflv"}
	episodios := []models.Episode{{ID: "animeflv:bleach:episode:1", Number: 1}}
	flvs := &proveedorFalso{
		nombre: "animeflv",
		detalleF: func(context.Context, string) (models.AnimeDetail, []models.Episode, error) {
			return detalle, episodios, nil
		},
	}
	e := nuevoEntorno(t, flvs)

	res, err := e.n.detalle(context.Background(), "animeflv", "bleach")
	if err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if res.Detail.ID != "bleach" || len(res.Episodes) != 1 {
		t.Errorf("resultado = %+v", res)
	}
	if eventos := e.emis.lista(); len(eventos) != 0 {
		t.Errorf("sin EpisodeStreamer no hay lotes que emitir: %+v", eventos)
	}
	if ops := flvs.operaciones(); len(ops) != 1 || ops[0] != "GetAnimeDetails" {
		t.Errorf("ops = %v; se esperaba GetAnimeDetails", ops)
	}
}

func TestDetalleEmiteLotesYPorcentajeCompleto(t *testing.T) {
	detalle := models.AnimeDetail{ID: "bleach", Title: "Bleach"}
	primera := []models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}}
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{
				AnimeID:  id,
				Batch:    2,
				Episodes: []models.Episode{{ID: "jkanime:bleach:episode:2", Number: 2}},
			})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return detalle, primera, nil
		},
	}
	e := nuevoEntorno(t, fake)

	res, err := e.n.detalle(context.Background(), "jkanime", "bleach")
	if err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if res.Detail.ID != "bleach" || len(res.Episodes) != 1 {
		t.Fatalf("resultado = %+v; debía traer el detalle y la página 1", res)
	}

	eventos := e.emis.lista()
	if len(eventos) != 3 {
		t.Fatalf("eventos = %+v; quería update de cada lote + complete", eventos)
	}
	if eventos[0].Nombre != "episodes:update" {
		t.Errorf("primer evento = %q", eventos[0].Nombre)
	}
	lote := loteDe(t, eventos[0])
	if lote.AnimeID != "bleach" || lote.Batch != 2 || lote.Done || len(lote.Episodes) != 1 {
		t.Errorf("lote = %+v", lote)
	}
	// El lote final (Done) también se emite como update —el payload lleva
	// done=true— y además cierra el stream con episodes:complete.
	if eventos[1].Nombre != "episodes:update" || !loteDe(t, eventos[1]).Done {
		t.Errorf("segundo evento = %+v; debía ser el lote final", eventos[1])
	}
	if eventos[2].Nombre != "episodes:complete" {
		t.Errorf("tercer evento = %q", eventos[2].Nombre)
	}
	completado := completoDe(t, eventos[2])
	if completado.AnimeID != "bleach" || completado.Error != "" {
		t.Errorf("completado = %+v; el error debe ir vacío", completado)
	}
}

func TestDetalleConFalloDePaginaLoCuentaElEventoCompleto(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true, Err: "la página 3 falló"})
			return models.AnimeDetail{ID: id}, nil, nil
		},
	}
	e := nuevoEntorno(t, fake)

	if _, err := e.n.detalle(context.Background(), "jkanime", "bleach"); err != nil {
		t.Fatalf("el fallo de una página no tumba la llamada: %v", err)
	}

	eventos := e.emis.lista()
	completados := completadosDe(t, eventos, "bleach")
	if len(completados) != 1 {
		t.Fatalf("completados = %+v", completados)
	}
	if completados[0].Error != "la página 3 falló" {
		t.Errorf("Error = %q; el fallo debe llegar al frontend como texto", completados[0].Error)
	}
}

func TestCambioDeAnimeCancelaElStreamAnterior(t *testing.T) {
	liberar := make(chan struct{})
	terminado := make(chan struct{})

	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			if id == "anime-a" {
				go func() {
					defer close(terminado)
					onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2})
					// El test libera este bloqueo DESPUÉS de cancelar el
					// stream: los lotes siguientes deben suprimirse en el
					// núcleo aunque el provider los entregue.
					<-liberar
					onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3})
					onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
				}()
			}
			return models.AnimeDetail{ID: id}, []models.Episode{{ID: id + ":1", Number: 1}}, nil
		},
	}
	e := nuevoEntorno(t, fake)

	if _, err := e.n.detalle(context.Background(), "jkanime", "anime-a"); err != nil {
		t.Fatalf("detalle anime-a: %v", err)
	}
	e.emis.esperar(t, 1) // el lote 2 de anime-a ya salió

	// Pedir otro anime cancela el stream anterior.
	res, err := e.n.detalle(context.Background(), "jkanime", "anime-b")
	if err != nil {
		t.Fatalf("detalle anime-b: %v", err)
	}
	if res.Detail.ID != "anime-b" {
		t.Errorf("detalle = %+v", res)
	}

	close(liberar) // anime-a intenta emitir más, ya con el ctx cancelado
	select {
	case <-terminado:
	case <-time.After(2 * time.Second):
		t.Fatal("el stream anterior no terminó tras cancelarse")
	}

	eventos := e.emis.lista()
	if lotes := lotesDe(t, eventos, "anime-a"); len(lotes) != 1 || lotes[0].Batch != 2 {
		t.Errorf("lotes de anime-a = %+v; solo debe llegar el primero", lotes)
	}
	if completados := completadosDe(t, eventos, "anime-a"); len(completados) != 0 {
		t.Errorf("un stream cancelado no emite episodes:complete: %+v", completados)
	}
}

func TestShutdownCancelaElStreamYCierraElProxy(t *testing.T) {
	terminado := make(chan struct{})
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			go func() {
				defer close(terminado)
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2})
				<-ctx.Done()
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3})
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
			}()
			return models.AnimeDetail{ID: id}, nil, nil
		},
	}
	e := nuevoEntorno(t, fake)

	if err := e.n.arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	if _, escuchadas, _ := e.proxy.contadores(); escuchadas != 1 {
		t.Fatalf("escuchadas = %d; el proxy debe escuchar al arrancar", escuchadas)
	}

	if _, err := e.n.detalle(context.Background(), "jkanime", "bleach"); err != nil {
		t.Fatalf("detalle: %v", err)
	}
	e.emis.esperar(t, 1)

	if err := e.n.cerrar(); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
	select {
	case <-terminado:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown no canceló el stream activo")
	}
	if err := e.n.cerrar(); err != nil {
		t.Fatalf("cerrar debe ser idempotente: %v", err)
	}

	_, _, cerradas := e.proxy.contadores()
	if cerradas != 2 {
		t.Errorf("cerradas = %d; se esperaba 2 (shutdown + idempotente)", cerradas)
	}
	eventos := e.emis.lista()
	if lotes := lotesDe(t, eventos, "bleach"); len(lotes) != 1 {
		t.Errorf("tras shutdown no debe emitir más lotes: %+v", lotes)
	}
	if completados := completadosDe(t, eventos, "bleach"); len(completados) != 0 {
		t.Errorf("el stream cancelado no emite episodes:complete: %+v", completados)
	}
}

func TestSinEmisorNoEmiteNiFalla(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return models.AnimeDetail{ID: id}, nil, nil
		},
	}
	reg := scraper.NewRegistry()
	if err := reg.Register(fake); err != nil {
		t.Fatalf("registro: %v", err)
	}
	px := nuevoProxyFalso()
	// Núcleo SIN emisor: es como la app antes de startup.
	n := nuevoNucleo(reg, px, extractors.SelectBestSource, nil, cache.New(ttlCacheDetalle), t.TempDir())

	res, err := n.detalle(context.Background(), "jkanime", "bleach")
	if err != nil {
		t.Fatalf("detalle sin emisor: %v", err)
	}
	if res.Detail.ID != "bleach" {
		t.Errorf("resultado = %+v", res)
	}
}

func TestArrancarInstalaElEmisorYElProxy(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return models.AnimeDetail{ID: id}, nil, nil
		},
	}
	reg := scraper.NewRegistry()
	if err := reg.Register(fake); err != nil {
		t.Fatalf("registro: %v", err)
	}
	px := nuevoProxyFalso()
	emis := nuevoEmisorFalso()
	n := nuevoNucleo(reg, px, extractors.SelectBestSource, nil, cache.New(ttlCacheDetalle), t.TempDir())
	// arrancar pone a funcionar la limpieza de la caché: que no quede la
	// goroutine suelta al terminar el test.
	t.Cleanup(func() { _ = n.cerrar() })

	if err := n.arrancar(emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	if _, escuchadas, _ := px.contadores(); escuchadas != 1 {
		t.Errorf("escuchadas = %d", escuchadas)
	}
	if _, err := n.detalle(context.Background(), "jkanime", "bleach"); err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if len(emis.lista()) != 3 {
		t.Errorf("eventos = %+v; con emisor instalado deben emitirse los tres", emis.lista())
	}
}

// ---------------------------------------------------------------------------
// Fuentes de vídeo.
// ---------------------------------------------------------------------------

func TestFuentesSoloHLSYMP4PasanPorElProxyYLaMejorVaPrimero(t *testing.T) {
	originales := []models.VideoSource{
		{URL: "https://cdn.example/uno.m3u8", Type: models.ResourceHLS, Referer: "https://jkanime.net/"},
		{URL: "https://cdn.example/dos.mp4", Type: models.ResourceMP4},
		{URL: "https://voe.sx/e/abc123", Type: models.ResourceIframe, Server: "VOE", Referer: "https://voe.sx/"},
	}
	prov := &proveedorFalso{
		nombre: "jkanime",
		fuentesF: func(context.Context, string, scraper.SourceOpts) ([]models.VideoSource, error) {
			return originales, nil
		},
	}
	e := nuevoEntorno(t, prov)

	salida, err := e.n.fuentes(context.Background(), "jkanime", "jkanime:bleach:episode:1", false)
	if err != nil {
		t.Fatalf("fuentes: %v", err)
	}
	if len(salida) != 3 {
		t.Fatalf("salida = %+v; quería las 3 fuentes", salida)
	}

	// Mejor primero: el HLS (SelectBestSource de T17), ya con URL del proxy.
	if salida[0].Type != models.ResourceHLS {
		t.Errorf("primera = %+v; debía ser el HLS", salida[0])
	}
	if !strings.HasPrefix(salida[0].URL, "http://127.0.0.1:39999/proxy?") {
		t.Errorf("el HLS debe venir con la URL del proxy: %q", salida[0].URL)
	}
	if destino := destinoProxy(t, salida[0].URL); destino != originales[0].URL {
		t.Errorf("el destino del proxy = %q; quería %q", destino, originales[0].URL)
	}
	// El MP4 también pasa por el proxy.
	if !strings.HasPrefix(salida[1].URL, "http://127.0.0.1:39999/proxy?") {
		t.Errorf("el MP4 debe venir con la URL del proxy: %q", salida[1].URL)
	}
	// El iframe NO pasa por el proxy y conserva sus campos.
	if salida[2].URL != "https://voe.sx/e/abc123" {
		t.Errorf("el iframe no debe tocarse: %q", salida[2].URL)
	}
	if salida[2].Server != "VOE" || salida[2].Type != models.ResourceIframe {
		t.Errorf("el iframe perdió campos: %+v", salida[2])
	}
	// El iframe no pasa por el proxy, así que sus campos (poco) se conservan.
	if salida[2].Referer != "https://voe.sx/" {
		t.Errorf("el iframe debía conservar su Referer: %+v", salida[2])
	}

	// El proxy solo debe haber registrado las dos reproducibles.
	registradas, _, _ := e.proxy.contadores()
	if registradas != 2 {
		t.Errorf("registradas = %d; solo HLS y MP4 (los iframes no)", registradas)
	}
	// T29: el Referer/Headers se quedan en la sesión (registradas) y la
	// fuente devuelta —la que cruza el puente de Wails— sale sin ellos.
	if e.proxy.registradas[0].Referer != "https://jkanime.net/" {
		t.Errorf("la sesión no recibió el Referer: %+v", e.proxy.registradas[0])
	}
	if salida[0].Referer != "" || salida[0].Headers != nil {
		t.Errorf("la fuente hacia el frontend arrastra cabeceras: %+v", salida[0])
	}
}

func TestFuentesNoCacheSeAceptaYNoHayNadaCacheado(t *testing.T) {
	prov := &proveedorFalso{
		nombre: "jkanime",
		fuentesF: func(context.Context, string, scraper.SourceOpts) ([]models.VideoSource, error) {
			return []models.VideoSource{{URL: "https://cdn.example/a.m3u8", Type: models.ResourceHLS}}, nil
		},
	}
	e := nuevoEntorno(t, prov)

	if _, err := e.n.fuentes(context.Background(), "jkanime", "jkanime:bleach:episode:1", true); err != nil {
		t.Fatalf("fuentes: %v", err)
	}
	if opts := prov.optsRecibidos(); !opts.NoCache {
		t.Errorf("noCache=%v; la opción debe aceptarse y pasarse al provider", opts.NoCache)
	}
	// Segunda petición idéntica: el provider se vuelve a llamar, es decir,
	// el núcleo no cachea nada (regla 9; la caché es T21).
	if _, err := e.n.fuentes(context.Background(), "jkanime", "jkanime:bleach:episode:1", true); err != nil {
		t.Fatalf("fuentes: %v", err)
	}
	if ops := prov.operaciones(); len(ops) != 2 {
		t.Errorf("ops = %v; deben ser 2 peticiones (sin caché)", ops)
	}
}

func TestFuentesProveedorDesconocido(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: "jkanime"})

	_, err := e.n.fuentes(context.Background(), "noexiste", "jkanime:bleach:episode:1", false)
	if err == nil || !strings.Contains(err.Error(), "proveedor desconocido") {
		t.Fatalf("err = %v; quería el error claro de proveedor desconocido", err)
	}
}
