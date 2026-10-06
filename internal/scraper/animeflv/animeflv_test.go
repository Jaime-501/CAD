package animeflv

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/scraper"
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de httpx.Request (mismo
	// criterio que jkanime). No definir otro TestMain en este paquete.
	desecho := slog.New(slog.NewTextHandler(io.Discard, nil))
	logx.SetDefault(desecho)
	slog.SetDefault(desecho)
	os.Exit(m.Run())
}

// doerGrabador es un httpx.HTTPDoer falso: no toca la red, solo ejecuta fn.
type doerGrabador struct {
	fn func(*http.Request) (*http.Response, error)
}

var _ httpx.HTTPDoer = (*doerGrabador)(nil)

func (d *doerGrabador) Do(req *http.Request) (*http.Response, error) { return d.fn(req) }

func clientePrueba() httpx.HTTPDoer {
	return httpx.NewClient(httpx.ClientOptions{})
}

// fixture carga un HTML real de testdata (respuesta cruda del servidor,
// capturada con curl.exe; ver PROGRESO.md).
func fixture(t *testing.T, nombre string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatalf("no pude leer testdata/%s: %v", nombre, err)
	}
	return data
}

// servidorHTML sirve html y comprueba dentro del handler la ruta pedida, la
// query cruda y el User-Agent: así se valida la URL exacta que arma Search
// (/?s=…) sin estado compartido entre goroutines. Devuelve también el
// contador de peticiones.
func servidorHTML(t *testing.T, html []byte, rutaEsperada, rawQueryEsperada string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		if got := r.URL.EscapedPath(); got != rutaEsperada {
			t.Errorf("ruta pedida = %q, quería %q", got, rutaEsperada)
		}
		if rawQueryEsperada != "" && r.URL.RawQuery != rawQueryEsperada {
			t.Errorf("query pedida = %q, quería %q", r.URL.RawQuery, rawQueryEsperada)
		}
		if ua := r.Header.Get("User-Agent"); ua != httpx.DefaultUserAgent {
			t.Errorf("User-Agent = %q, quería el de navegador %q", ua, httpx.DefaultUserAgent)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	}))
	t.Cleanup(srv.Close)
	return srv, &peticiones
}

func TestSearchRutaResultadosYUserAgent(t *testing.T) {
	html := fixture(t, "search.html")
	srv, peticiones := servidorHTML(t, html, "/", "s=bleach")
	a := New(clientePrueba(), srv.URL)

	got, err := a.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}
	if len(got) != 1 {
		t.Fatalf("resultados = %d, quería 1 (el del fixture)", len(got))
	}
	// El id es el slug del href, no una derivación del título.
	const id, titulo = "bleach-sennen-kessen-hen-kashin-tan", "Bleach: Sennen Kessen-hen - Kashin-tan"
	if got[0].ID != id || got[0].Title != titulo {
		t.Errorf("primer resultado = %+v, quería id=%q title=%q", got[0], id, titulo)
	}
	if got[0].CoverURL != "https://animeflv.or.at/wp-content/uploads/2026/07/4429-230x200.jpg" {
		t.Errorf("CoverURL = %q, quería la portada absoluta del fixture", got[0].CoverURL)
	}
	if got[0].Provider != providerName {
		t.Errorf("Provider = %q, quería %q", got[0].Provider, providerName)
	}
	if strings.Contains(got[0].ID, "/") || strings.HasPrefix(got[0].ID, "http") {
		t.Errorf("id %q debe ser el slug, sin barra ni dominio", got[0].ID)
	}
}

func TestSearchEscapaLaQuery(t *testing.T) {
	html := fixture(t, "search.html")
	casos := []struct {
		nombre, query, rawQuery string
	}{
		{"espacio", "naruto shippuden", "s=naruto+shippuden"},
		{"barra", "one/piece", "s=one%2Fpiece"},
		{"signos", "steins;gate?", "s=steins%3Bgate%3F"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, _ := servidorHTML(t, html, "/", c.rawQuery)
			a := New(clientePrueba(), srv.URL)

			got, err := a.Search(context.Background(), c.query)
			if err != nil {
				t.Fatalf("Search falló: %v", err)
			}
			if len(got) == 0 {
				t.Error("quería resultados (el HTML del fixture es el mismo)")
			}
		})
	}
}

func TestSearchQueryVaciaNoPideRed(t *testing.T) {
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		t.Error("Search no debía hacer ninguna petición")
		return nil, errors.New("sin red (doer de prueba)")
	}}
	a := New(d, "")

	for _, query := range []string{"", "   ", "\t\n "} {
		got, err := a.Search(context.Background(), query)
		if err != nil {
			t.Errorf("query %q: %v", query, err)
		}
		if got == nil {
			t.Errorf("query %q: resultado nil, quería un slice vacío (la UI espera [] no null)", query)
			continue
		}
		if len(got) != 0 {
			t.Errorf("query %q: %d resultados, quería 0", query, len(got))
		}
	}
}

func TestSearchSinResultados(t *testing.T) {
	html := fixture(t, "search_empty.html")
	srv, _ := servidorHTML(t, html, "/", "s=zzzznoexisteestetitulo")
	a := New(clientePrueba(), srv.URL)

	got, err := a.Search(context.Background(), "zzzznoexisteestetitulo")
	if err != nil {
		t.Fatalf("una página sin resultados no debe ser error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("resultados = %#v, quería un slice vacío", got)
	}
}

func TestSearchErrorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error interno", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	_, err := a.Search(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un 500")
	}
	if !errors.Is(err, httpx.ErrServerError) {
		t.Errorf("err = %v, quería httpx.ErrServerError", err)
	}
}

func TestSearchDemasiadoGrande(t *testing.T) {
	cuerpo := bytes.Repeat([]byte("a"), maxSearchBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(cuerpo)
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	_, err := a.Search(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxSearchBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}

// TestSearchDescartaEnlacesInesperados usa HTML artificial (regla 11): el
// sitio solo devuelve tarjetas con enlaces /anime/<slug>/, así que el
// descarte de rutas ajenas, vacías, anidadas y tarjetas sin título no se
// puede reproducir con un fixture real. La portada relativa comprueba que
// absoluta la resuelve contra la URL base.
func TestSearchDescartaEnlacesInesperados(t *testing.T) {
	html := []byte(`
<div class="search-series-grid">
  <div class="search-series-card"><a class="thumbnail-link" href="https://animeflv.or.at/anime/bleach/"><img class="anime-image" src="/cover.jpg"><h2 class="entry-title">Bleach</h2></a></div>
  <div class="search-series-card"><a class="thumbnail-link" href="/anime/one-piece/"><h2 class="entry-title">One Piece</h2></a></div>
  <div class="search-series-card"><a class="thumbnail-link" href="https://otro.example/series/bleach/"><h2 class="entry-title">Ruta ajena</h2></a></div>
  <div class="search-series-card"><a class="thumbnail-link" href="https://animeflv.or.at/anime/"><h2 class="entry-title">Sin slug</h2></a></div>
  <div class="search-series-card"><a class="thumbnail-link" href="https://animeflv.or.at/otra/ruta/"><h2 class="entry-title">Sin prefijo</h2></a></div>
  <div class="search-series-card"><a class="thumbnail-link" href="https://animeflv.or.at/anime/a/b/"><h2 class="entry-title">Anidado</h2></a></div>
  <div class="search-series-card"><a class="thumbnail-link" href="https://animeflv.or.at/anime/sin-titulo/"></a></div>
</div>`)
	srv, _ := servidorHTML(t, html, "/", "")
	a := New(clientePrueba(), srv.URL)

	got, err := a.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("resultados = %d, quería 2 (los enlaces /anime/<slug> con título); got=%+v", len(got), got)
	}
	if got[0].ID != "bleach" || got[0].Title != "Bleach" {
		t.Errorf("primer resultado = %+v, quería id=bleach title=Bleach", got[0])
	}
	if got[1].ID != "one-piece" || got[1].Title != "One Piece" {
		t.Errorf("segundo resultado = %+v, quería id=one-piece title=One Piece", got[1])
	}
	if want := srv.URL + "/cover.jpg"; got[0].CoverURL != want {
		t.Errorf("CoverURL = %q, quería la portada resuelta contra la base %q", got[0].CoverURL, want)
	}
	if got[1].CoverURL != "" {
		t.Errorf("CoverURL = %q, quería vacío (la tarjeta no trae imagen)", got[1].CoverURL)
	}
}

func TestGetAnimeDetailsDevuelveErrNoDetails(t *testing.T) {
	a := New(nil, "")

	detalle, episodios, err := a.GetAnimeDetails(context.Background(), "bleach")
	if !errors.Is(err, scraper.ErrNoDetails) {
		t.Fatalf("err = %v, quería scraper.ErrNoDetails", err)
	}
	if detalle.ID != "" || detalle.Title != "" || detalle.Synopsis != "" || len(detalle.Genres) != 0 {
		t.Errorf("detalle = %+v, quería el valor cero", detalle)
	}
	if episodios != nil {
		t.Errorf("episodios = %#v, quería nil", episodios)
	}
}

func TestGetVideoSourcesDevuelveErrorDescriptivo(t *testing.T) {
	a := New(nil, "")

	fuentes, err := a.GetVideoSources(context.Background(), "animeflv:bleach:episode:1", scraper.SourceOpts{})
	if err == nil {
		t.Fatal("quería error: AnimeFLV no ofrece fuentes")
	}
	if errors.Is(err, scraper.ErrNoDetails) {
		t.Error("el error debe ser descriptivo, no scraper.ErrNoDetails (habla de detalle)")
	}
	if !strings.Contains(err.Error(), "fuentes de vídeo") {
		t.Errorf("err = %v, quería un mensaje que mencione las fuentes de vídeo", err)
	}
	if !strings.Contains(err.Error(), "animeflv:bleach:episode:1") {
		t.Errorf("err = %v, quería el id del episodio en el mensaje", err)
	}
	if fuentes != nil {
		t.Errorf("fuentes = %#v, quería nil", fuentes)
	}
}

func TestNameYNewNormaliza(t *testing.T) {
	if (&AnimeFLV{}).Name() != providerName {
		t.Errorf("Name() = %q, quería %q", (&AnimeFLV{}).Name(), providerName)
	}

	a := New(nil, "  https://ejemplo.test///  ")
	if a.BaseURL != "https://ejemplo.test" {
		t.Errorf("BaseURL = %q, quería sin espacios ni barra final", a.BaseURL)
	}

	a = New(nil, "")
	if a.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, quería %q", a.BaseURL, DefaultBaseURL)
	}

	// Valor cero de AnimeFLV (sin New): mismos criterios por defecto.
	cero := &AnimeFLV{}
	if got := cero.siteURL(); got != DefaultBaseURL {
		t.Errorf("siteURL() = %q, quería %q", got, DefaultBaseURL)
	}
}
