package extractors

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/models"
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de httpx.Request ni con
	// el aviso de Resolve cuando un extractor falla. No definir otro
	// TestMain en este paquete (ver PROGRESO.md).
	desecho := slog.New(slog.NewTextHandler(io.Discard, nil))
	logx.SetDefault(desecho)
	slog.SetDefault(desecho)
	os.Exit(m.Run())
}

// htmlEpisodio es el HTML artificial de las pruebas: contiene las tres
// marcas que los extractores de prueba reconocen (data-src para JKPlayer,
// JKPHP para la API PHP y "var servers" para la lista de servidores).
const htmlEpisodio = `<!DOCTYPE html>
<html><head><title>Episodio 1</title></head>
<body>
<div id="video" data-src="https://jkanime.net/ver/one-piece-1"></div>
<script>
  var option = { file: "/JKPHP/api.php", type: "hls" };
  var servers = [{ name: "fembed", url: "https://fembed.example.test/v/abc" }];
</script>
</body></html>`

// extractorFalso es un VideoExtractor de prueba: reconoce la página si el
// HTML contiene su marcador (vacío = acepta siempre) y devuelve siempre las
// mismas fuentes o el mismo error. Los contadores prueban la cadena.
type extractorFalso struct {
	nombre   string
	marcador string
	fuentes  []models.VideoSource
	err      error
	orden    *[]string

	consultas int
	salidas   int
}

func (e *extractorFalso) Name() string { return e.nombre }

func (e *extractorFalso) CanHandle(page Page) bool {
	e.consultas++
	return e.marcador == "" || strings.Contains(page.HTML, e.marcador)
}

func (e *extractorFalso) Extract(context.Context, Page) ([]models.VideoSource, error) {
	e.salidas++
	if e.orden != nil {
		*e.orden = append(*e.orden, e.nombre)
	}
	return e.fuentes, e.err
}

func fuentesDe(nombres ...string) []models.VideoSource {
	out := make([]models.VideoSource, 0, len(nombres))
	for _, n := range nombres {
		out = append(out, models.VideoSource{URL: n, Type: models.ResourceHLS})
	}
	return out
}

func urls(fuentes []models.VideoSource) []string {
	out := make([]string, 0, len(fuentes))
	for _, f := range fuentes {
		out = append(out, f.URL)
	}
	return out
}

func TestNewOrdenaLaCadena(t *testing.T) {
	var orden []string
	// Registrados deliberadamente fuera de orden; los desconocidos (fembed,
	// zippy) deben conservar su orden de registro al final.
	servers := &extractorFalso{nombre: NameServers, marcador: "var servers", orden: &orden, fuentes: fuentesDe("s1")}
	fembed := &extractorFalso{nombre: "fembed", orden: &orden, fuentes: fuentesDe("f1")}
	jkphp := &extractorFalso{nombre: NameJKPHP, marcador: "JKPHP", orden: &orden, fuentes: fuentesDe("p1")}
	jkplayer := &extractorFalso{nombre: NameJKPlayer, marcador: "data-src", orden: &orden, fuentes: fuentesDe("j1")}
	zippy := &extractorFalso{nombre: "zippy", orden: &orden, fuentes: fuentesDe("z1")}

	r := New(servers, fembed, jkphp, jkplayer, zippy)
	fuentes, err := r.Resolve(context.Background(), Page{URL: "https://jkanime.net/ver/one-piece-1", HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	queria := []string{NameJKPlayer, NameJKPHP, NameServers, "fembed", "zippy"}
	if !slices.Equal(orden, queria) {
		t.Errorf("orden = %v, quería %v", orden, queria)
	}
	if len(fuentes) != 5 {
		t.Errorf("fuentes = %d, quería 5", len(fuentes))
	}
}

func TestPrioridadNormalizaNombre(t *testing.T) {
	casos := []struct {
		nombre string
		queria int
	}{
		{NameJKPlayer, 0},
		{" JKPHP ", 1},
		{"Servers", 2},
		{"fembed", len(ordenCanonico)},
		{"", len(ordenCanonico)},
	}
	for _, c := range casos {
		if got := prioridad(&extractorFalso{nombre: c.nombre}); got != c.queria {
			t.Errorf("prioridad(%q) = %d, quería %d", c.nombre, got, c.queria)
		}
	}
}

func TestNewCopiaYDescartaNulos(t *testing.T) {
	uno := &extractorFalso{nombre: "fembed", fuentes: fuentesDe("f1")}
	dos := &extractorFalso{nombre: "zippy", fuentes: fuentesDe("z1")}
	entrada := []VideoExtractor{uno, nil, dos}
	r := New(entrada...)

	// La lista se copia: reasignar el elemento no cambia la cadena.
	entrada[0] = &extractorFalso{nombre: "cambiado", fuentes: fuentesDe("x")}
	fuentes, err := r.Resolve(context.Background(), Page{HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	queria := []string{"f1", "z1"}
	if !slices.Equal(urls(fuentes), queria) {
		t.Errorf("URLs = %v, quería %v", urls(fuentes), queria)
	}
	if len(r.extractores) != 2 {
		t.Errorf("extractores = %d, quería 2 (el nil se descarta)", len(r.extractores))
	}
}

func TestResolveAgregaFuentesDeTodos(t *testing.T) {
	jkplayer := &extractorFalso{
		nombre: NameJKPlayer, marcador: "data-src",
		fuentes: []models.VideoSource{
			{URL: "https://cdn.example.test/uno.m3u8", Type: models.ResourceHLS},
			{URL: "https://cdn.example.test/dos.mp4", Type: models.ResourceMP4},
		},
	}
	jkphp := &extractorFalso{
		nombre: NameJKPHP, marcador: "JKPHP",
		fuentes: fuentesDe("https://cdn.example.test/tres.m3u8"),
	}
	servers := &extractorFalso{
		nombre: NameServers, marcador: "var servers",
		fuentes: fuentesDe("https://cdn.example.test/cuatro.m3u8"),
	}
	// No acepta la página: solo se le consulta, no se le ejecuta.
	niega := &extractorFalso{nombre: "youtube", marcador: "no-está-en-el-html", fuentes: fuentesDe("nunca")}

	r := New(servers, jkphp, niega, jkplayer)
	fuentes, err := r.Resolve(context.Background(), Page{URL: "https://jkanime.net/ver/one-piece-1", HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	queria := []string{
		"https://cdn.example.test/uno.m3u8",
		"https://cdn.example.test/dos.mp4",
		"https://cdn.example.test/tres.m3u8",
		"https://cdn.example.test/cuatro.m3u8",
	}
	if !slices.Equal(urls(fuentes), queria) {
		t.Errorf("URLs = %v, quería %v", urls(fuentes), queria)
	}
	if niega.consultas != 1 || niega.salidas != 0 {
		t.Errorf("niega: consultas=%d salidas=%d, quería 1 y 0", niega.consultas, niega.salidas)
	}
	for _, e := range []*extractorFalso{jkplayer, jkphp, servers} {
		if e.consultas != 1 || e.salidas != 1 {
			t.Errorf("%s: consultas=%d salidas=%d, quería 1 y 1", e.nombre, e.consultas, e.salidas)
		}
	}
}

// capturador es un slog.Handler que guarda los registros para las pruebas.
type capturador struct {
	mu       sync.Mutex
	registro []slog.Record
}

func (c *capturador) Enabled(context.Context, slog.Level) bool { return true }

func (c *capturador) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.registro = append(c.registro, r.Clone())
	return nil
}

func (c *capturador) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *capturador) WithGroup(string) slog.Handler      { return c }

func attrTexto(r slog.Record, clave string) string {
	var valor string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == clave {
			valor = a.Value.String()
			return false
		}
		return true
	})
	return valor
}

func TestResolveExtractorFallidoSigueConElSiguiente(t *testing.T) {
	previo := slog.Default()
	defer slog.SetDefault(previo)
	logCap := &capturador{}
	slog.SetDefault(slog.New(logCap))

	falla := &extractorFalso{
		nombre: NameJKPlayer, marcador: "data-src",
		err: errors.New("token CSRF vencido"),
	}
	sigue := &extractorFalso{
		nombre: NameJKPHP, marcador: "JKPHP",
		fuentes: fuentesDe("https://cdn.example.test/posterior.m3u8"),
	}
	r := New(falla, sigue)

	fuentes, err := r.Resolve(context.Background(), Page{URL: "https://jkanime.net/ver/one-piece-1", HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve: %v (un fallo no debe romper la cadena)", err)
	}
	if falla.salidas != 1 || sigue.salidas != 1 {
		t.Errorf("salidas: falla=%d sigue=%d, quería 1 y 1", falla.salidas, sigue.salidas)
	}
	queria := []string{"https://cdn.example.test/posterior.m3u8"}
	if !slices.Equal(urls(fuentes), queria) {
		t.Errorf("URLs = %v, quería %v", urls(fuentes), queria)
	}

	var avisos int
	for _, rec := range logCap.registro {
		if rec.Level != slog.LevelWarn || !strings.Contains(rec.Message, "extractor falló") {
			continue
		}
		avisos++
		if got := attrTexto(rec, "extractor"); got != NameJKPlayer {
			t.Errorf("aviso sin extractor=%q (fue %q)", NameJKPlayer, got)
		}
		if got := attrTexto(rec, "err"); !strings.Contains(got, "token CSRF vencido") {
			t.Errorf("aviso sin el error del extractor: %q", got)
		}
	}
	if avisos != 1 {
		t.Errorf("avisos de extractor fallido = %d, quería 1", avisos)
	}
}

func TestResolveSinFuentes(t *testing.T) {
	casos := []struct {
		nombre      string
		extractores []VideoExtractor
	}{
		{"sin extractores", nil},
		{"ninguno acepta la página", []VideoExtractor{
			&extractorFalso{nombre: NameJKPlayer, marcador: "no-está-en-el-html", fuentes: fuentesDe("https://cdn.example.test/x.m3u8")},
			&extractorFalso{nombre: NameJKPHP, marcador: "tampoco-está", fuentes: fuentesDe("https://cdn.example.test/y.m3u8")},
		}},
		{"solo URLs vacías", []VideoExtractor{
			&extractorFalso{nombre: NameServers, fuentes: []models.VideoSource{{URL: ""}, {URL: "   "}}},
		}},
		{"todos fallan", []VideoExtractor{
			&extractorFalso{nombre: NameJKPlayer, err: errors.New("caído")},
			&extractorFalso{nombre: NameJKPHP, err: errors.New("caído")},
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := New(c.extractores...)
			fuentes, err := r.Resolve(context.Background(), Page{HTML: htmlEpisodio})
			if !errors.Is(err, ErrSinFuentes) {
				t.Errorf("err = %v, quería errors.Is(err, ErrSinFuentes)", err)
			}
			if fuentes != nil {
				t.Errorf("fuentes = %v, quería nil", fuentes)
			}
		})
	}
}

func TestResolveEtiquetaServerYDescartaURLsVacias(t *testing.T) {
	e := &extractorFalso{
		nombre: NameJKPHP, marcador: "JKPHP",
		fuentes: []models.VideoSource{
			{URL: "", Server: "otro"},
			{URL: "   ", Server: "otro"},
			{URL: "https://cdn.example.test/sin-server.m3u8", Type: models.ResourceHLS},
			{URL: "https://cdn.example.test/con-server.m3u8", Type: models.ResourceHLS, Server: "fembed"},
		},
	}
	r := New(e)

	fuentes, err := r.Resolve(context.Background(), Page{HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(fuentes) != 2 {
		t.Fatalf("fuentes = %d, quería 2 (las dos sin URL se descartan)", len(fuentes))
	}
	if fuentes[0].Server != NameJKPHP {
		t.Errorf("fuentes[0].Server = %q, quería %q (etiqueta con Name)", fuentes[0].Server, NameJKPHP)
	}
	if fuentes[1].Server != "fembed" {
		t.Errorf("fuentes[1].Server = %q, quería %q (no pisa la etiqueta propia)", fuentes[1].Server, "fembed")
	}
}

func TestResolveContextoCancelado(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := &extractorFalso{nombre: NameJKPlayer, marcador: "data-src"}
	r := New(e)

	fuentes, err := r.Resolve(ctx, Page{HTML: htmlEpisodio})
	if !errors.Is(err, ErrSinFuentes) {
		t.Errorf("err = %v, quería errors.Is(err, ErrSinFuentes)", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería que envuelva context.Canceled", err)
	}
	if fuentes != nil {
		t.Errorf("fuentes = %v, quería nil", fuentes)
	}
}

func TestResolveContextoNulo(t *testing.T) {
	e := &extractorFalso{nombre: NameJKPlayer, marcador: "data-src", fuentes: fuentesDe("https://cdn.example.test/ok.m3u8")}
	r := New(e)

	fuentes, err := r.Resolve(nil, Page{HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve(nil, ...): %v", err)
	}
	if len(fuentes) != 1 {
		t.Errorf("fuentes = %d, quería 1", len(fuentes))
	}
}

// extractorConRed extrae del servidor local que le inyectan por Base.Doer
// (regla 4): sin doer no hay red y la extracción falla.
type extractorConRed struct {
	Base
}

func (e *extractorConRed) Name() string { return "fembed" }

func (e *extractorConRed) CanHandle(page Page) bool {
	return strings.Contains(page.HTML, "data-fuente")
}

func (e *extractorConRed) Extract(ctx context.Context, page Page) ([]models.VideoSource, error) {
	resp, err := httpx.Get(ctx, e.Doer, page.URL+"/fuente.m3u8", httpx.ReqOptions{
		Provider:  "jkanime",
		Operation: "extract",
		Extractor: e.Name(),
		Referer:   page.URL,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return []models.VideoSource{{
		URL:    strings.TrimSpace(string(body)),
		Type:   models.ResourceHLS,
		Server: e.Name(),
	}}, nil
}

func TestExtractorConDoerInyectado(t *testing.T) {
	const urlFuente = "https://cdn.example.test/video.m3u8"
	var mu sync.Mutex
	var referer, userAgent string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		referer, userAgent = r.Referer(), r.Header.Get("User-Agent")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = io.WriteString(w, urlFuente)
	}))
	t.Cleanup(srv.Close)

	t.Run("usa el doer del constructor", func(t *testing.T) {
		r := New(&extractorConRed{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}})
		page := Page{URL: srv.URL, HTML: `<div data-fuente="1"></div>`}

		fuentes, err := r.Resolve(context.Background(), page)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if len(fuentes) != 1 || fuentes[0].URL != urlFuente {
			t.Fatalf("fuentes = %+v, quería una con %q", fuentes, urlFuente)
		}
		if fuentes[0].Server != "fembed" {
			t.Errorf("Server = %q, quería %q", fuentes[0].Server, "fembed")
		}
		mu.Lock()
		defer mu.Unlock()
		if referer != srv.URL {
			t.Errorf("Referer = %q, quería %q (la página como referer)", referer, srv.URL)
		}
		if userAgent != httpx.DefaultUserAgent {
			t.Errorf("User-Agent = %q, quería el de navegador", userAgent)
		}
	})

	t.Run("sin doer no hay red", func(t *testing.T) {
		r := New(&extractorConRed{})
		fuentes, err := r.Resolve(context.Background(), Page{URL: srv.URL, HTML: `<div data-fuente="1"></div>`})
		if !errors.Is(err, ErrSinFuentes) {
			t.Errorf("err = %v, quería errors.Is(err, ErrSinFuentes)", err)
		}
		if fuentes != nil {
			t.Errorf("fuentes = %v, quería nil", fuentes)
		}
	})
}
