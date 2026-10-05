package jkanime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miapp/internal/httpx"
)

func TestFetchEpisodePageDescargaElFixture(t *testing.T) {
	html := fixtureDetalle(t, "episode_page.html")
	var (
		mu                sync.Mutex
		ruta, referer, ua string
	)
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		mu.Lock()
		ruta, referer, ua = r.URL.EscapedPath(), r.Header.Get("Referer"), r.Header.Get("User-Agent")
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, private")
		// La sesión viaja en el jar del cliente; no debe salir en Headers.
		w.Header().Add("Set-Cookie", "jkanime-session=SECRETO_DE_SESION; path=/")
		_, _ = w.Write(html)
	}))
	t.Cleanup(srv.Close)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.fetchEpisodePage(context.Background(), "jkanime:bleach:episode:1")
	if err != nil {
		t.Fatalf("fetchEpisodePage falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}

	mu.Lock()
	rutaPedida, refererPedido, uaPedido := ruta, referer, ua
	mu.Unlock()
	if rutaPedida != "/bleach/1/" {
		t.Errorf("ruta pedida = %q, quería %q", rutaPedida, "/bleach/1/")
	}
	if refererPedido != srv.URL+"/bleach/" {
		t.Errorf("Referer = %q, quería la página de la serie %q", refererPedido, srv.URL+"/bleach/")
	}
	if uaPedido != httpx.DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería httpx.DefaultUserAgent", uaPedido)
	}

	if got.URL != srv.URL+"/bleach/1/" {
		t.Errorf("URL = %q, quería la URL final %q", got.URL, srv.URL+"/bleach/1/")
	}
	if got.HTML != string(html) {
		t.Errorf("HTML = %d bytes, quería el cuerpo crudo del fixture (%d)", len(got.HTML), len(html))
	}
	if !strings.Contains(got.HTML, "<title>Bleach 1 Sub Español Online gratis") {
		t.Error("el fixture no es la página del episodio 1 de Bleach")
	}
	if ct := got.Headers.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, quería el de la respuesta", ct)
	}
	if cc := got.Headers.Get("Cache-Control"); cc != "no-cache, private" {
		t.Errorf("Cache-Control = %q, quería el de la respuesta", cc)
	}
	if sc := got.Headers.Get("Set-Cookie"); sc != "" {
		t.Errorf("Set-Cookie = %q, no debe salir de la sesión (regla 10)", sc)
	}
}

func TestFetchEpisodePageRutasSegunID(t *testing.T) {
	html := []byte("<!DOCTYPE html><html><body>ok</body></html>")
	casos := []struct{ nombre, id, ruta string }{
		{"episodio", "jkanime:bleach:episode:1", "/bleach/1/"},
		{"episodio alto", "jkanime:bleach:episode:366", "/bleach/366/"},
		{"especial", "jkanime:bleach-memories-in-the-rain:special:1", "/bleach-memories-in-the-rain/1/"},
		{"pelicula", "jkanime:bleach-the-diamonddust-rebellion:movie", "/bleach-the-diamonddust-rebellion/pelicula/"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, peticiones := servidorHTML(t, html, c.ruta)
			j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

			got, err := j.fetchEpisodePage(context.Background(), c.id)
			if err != nil {
				t.Fatalf("fetchEpisodePage(%q) falló: %v", c.id, err)
			}
			if n := peticiones.Load(); n != 1 {
				t.Errorf("peticiones = %d, quería 1", n)
			}
			if got.URL != srv.URL+c.ruta {
				t.Errorf("URL = %q, quería %q", got.URL, srv.URL+c.ruta)
			}
			if got.HTML != string(html) {
				t.Errorf("HTML = %q, quería el cuerpo servido", got.HTML)
			}
		})
	}
}

func TestFetchEpisodePageErroresHTTP(t *testing.T) {
	casos := []struct {
		nombre     string
		status     int
		cuerpo     string
		headers    map[string]string
		espera     error
		retryAfter time.Duration
	}{
		{"404", http.StatusNotFound, "no existe", nil, httpx.ErrNotFound, 0},
		{"403", http.StatusForbidden, "prohibido", nil, httpx.ErrForbidden, 0},
		{"429", http.StatusTooManyRequests, "espera", map[string]string{"Retry-After": "7"}, httpx.ErrTooManyRequests, 7 * time.Second},
		{"500", http.StatusInternalServerError, "error interno", nil, httpx.ErrServerError, 0},
		{
			"challenge",
			http.StatusOK,
			"<html><head><title>Just a moment...</title></head><body></body></html>",
			nil,
			httpx.ErrCloudflareChallenge,
			0,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range c.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.cuerpo))
			}))
			t.Cleanup(srv.Close)
			j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

			got, err := j.fetchEpisodePage(context.Background(), "jkanime:bleach:episode:1")
			if err == nil {
				t.Fatal("quería error")
			}
			if !errors.Is(err, c.espera) {
				t.Errorf("err = %v, quería %v", err, c.espera)
			}
			if got.URL != "" || got.HTML != "" || got.Headers != nil {
				t.Errorf("en error la página debe quedar vacía, got = %+v", got)
			}
			if c.retryAfter > 0 {
				var se *httpx.ScraperError
				if !errors.As(err, &se) {
					t.Fatalf("err = %T, quería *httpx.ScraperError", err)
				}
				if se.RetryAfter() != c.retryAfter {
					t.Errorf("RetryAfter() = %v, quería %v", se.RetryAfter(), c.retryAfter)
				}
			}
		})
	}
}

func TestFetchEpisodePageIdInvalidoNoPideRed(t *testing.T) {
	ids := []string{
		"",
		"   ",
		"bleach",
		"jkanime:bleach",
		"jkanime::episode:1",
		"jkanime:bleach:especial:1",
		"jkanime:bleach:episode:0",
		"jkanime:bleach:episode:01",
		"animeflv:bleach:episode:1",
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
				t.Error("con un id inválido no debía hacerse ninguna petición")
				return nil, errors.New("sin red (doer de prueba)")
			}}
			j := New(d, "", nil)

			_, err := j.fetchEpisodePage(context.Background(), id)
			if err == nil {
				t.Fatal("quería error con un id inválido")
			}
		})
	}
}

func TestFetchEpisodePageSigueLaRedireccion(t *testing.T) {
	// El sitio redirige /<slug>/1/ hacia /<slug>/pelicula/ en las películas
	// (comprobado en vivo); la página que queda es la canónica.
	html := []byte("<!DOCTYPE html><html><body>pagina de la pelicula</body></html>")
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		switch r.URL.EscapedPath() {
		case "/peli/1/":
			http.Redirect(w, r, "/peli/pelicula/", http.StatusFound)
		case "/peli/pelicula/":
			_, _ = w.Write(html)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.fetchEpisodePage(context.Background(), "jkanime:peli:episode:1")
	if err != nil {
		t.Fatalf("fetchEpisodePage falló: %v", err)
	}
	if n := peticiones.Load(); n != 2 {
		t.Errorf("peticiones = %d, quería 2 (ida y vuelta de la redirección)", n)
	}
	if got.URL != srv.URL+"/peli/pelicula/" {
		t.Errorf("URL = %q, quería la URL final tras el 302", got.URL)
	}
	if got.HTML != string(html) {
		t.Errorf("HTML = %q, quería el cuerpo de la página final", got.HTML)
	}
}

func TestFetchEpisodePageDemasiadoGrande(t *testing.T) {
	cuerpo := strings.Repeat("a", maxEpisodePageBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(srv.Close)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	_, err := j.fetchEpisodePage(context.Background(), "jkanime:bleach:episode:1")
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxEpisodePageBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}
