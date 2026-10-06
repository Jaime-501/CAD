package jkanime

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"miapp/internal/httpx"
)

// fixtureBusqueda carga un HTML real de testdata (capturado con la respuesta
// cruda del servidor, ver PROGRESO.md).
func fixtureBusqueda(t *testing.T, nombre string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatalf("no pude leer testdata/%s: %v", nombre, err)
	}
	return data
}

// servidorHTML sirve html en cualquier ruta y cuenta las peticiones. Si la
// ruta escapada no coincide con la esperada, el test falla (así se comprueba
// la URL que pide Search sin estado compartido entre goroutines).
func servidorHTML(t *testing.T, html []byte, rutaEsperada string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		if got := r.URL.EscapedPath(); got != rutaEsperada {
			t.Errorf("ruta pedida = %q, quería %q", got, rutaEsperada)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	}))
	t.Cleanup(srv.Close)
	return srv, &peticiones
}

func TestSearchRutaYResultados(t *testing.T) {
	html := fixtureBusqueda(t, "search.html")
	srv, peticiones := servidorHTML(t, html, "/buscar/bleach")
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}
	if len(got) != 10 {
		t.Fatalf("resultados = %d, quería 10 (los del fixture)", len(got))
	}

	// El id es el slug del href, no una derivación del título.
	esperados := []struct {
		id, titulo string
	}{
		{"bleach", "Bleach"},
		{"bleach-movie-3-fade-to-black-kimi-no-na-wo-yobu", "Bleach Movie 3: Fade to Black - Kimi no Na wo Yobu"},
		{"bleach-memories-in-the-rain", "Bleach: Memories in the Rain"},
		{"bleach-the-diamonddust-rebellion", "Bleach: The DiamondDust Rebellion"},
	}
	for _, e := range esperados {
		var encontrado bool
		for _, a := range got {
			if a.ID == e.id {
				encontrado = true
				if a.Title != e.titulo {
					t.Errorf("id %q: Title = %q, quería %q", e.id, a.Title, e.titulo)
				}
			}
		}
		if !encontrado {
			t.Errorf("no encontré el resultado con id %q", e.id)
		}
	}

	primero := got[0]
	if primero.ID != "bleach" || primero.Title != "Bleach" {
		t.Errorf("primero = %+v, quería id=bleach title=Bleach", primero)
	}
	if primero.CoverURL != "https://cdn.jkdesa.com/assets/images/animes/image/bleach.jpg" {
		t.Errorf("CoverURL = %q, quería la portada absoluta del fixture", primero.CoverURL)
	}

	for i, a := range got {
		if a.Provider != providerName {
			t.Errorf("resultado %d: Provider = %q, quería %q", i, a.Provider, providerName)
		}
		if a.ID == "" || a.Title == "" {
			t.Errorf("resultado %d incompleto: %+v", i, a)
		}
		if strings.Contains(a.ID, "/") || strings.HasPrefix(a.ID, "http") {
			t.Errorf("resultado %d: el id %q debe ser el slug, sin barra ni dominio", i, a.ID)
		}
		if a.CoverURL != "" && !strings.HasPrefix(a.CoverURL, "https://cdn.jkdesa.com/") {
			t.Errorf("resultado %d: CoverURL = %q, quería una URL absoluta", i, a.CoverURL)
		}
	}
}

func TestSearchEscapaLaQuery(t *testing.T) {
	html := fixtureBusqueda(t, "search.html")
	casos := []struct {
		nombre, query, ruta string
	}{
		{"espacio", "naruto shippuden", "/buscar/naruto%20shippuden"},
		{"barra", "one/piece", "/buscar/one%2Fpiece"},
		{"signos", "steins;gate?", "/buscar/steins%3Bgate%3F"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, peticiones := servidorHTML(t, html, c.ruta)
			j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

			got, err := j.Search(context.Background(), c.query)
			if err != nil {
				t.Fatalf("Search falló: %v", err)
			}
			if n := peticiones.Load(); n != 1 {
				t.Errorf("peticiones = %d, quería 1", n)
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
	j := New(d, "", nil)

	for _, query := range []string{"", "   ", "\t\n "} {
		got, err := j.Search(context.Background(), query)
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
	html := fixtureBusqueda(t, "search_empty.html")
	srv, _ := servidorHTML(t, html, "/buscar/zzzznoexisteestetitulo")
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Search(context.Background(), "zzzznoexisteestetitulo")
	if err != nil {
		t.Fatalf("una página sin resultados no debe ser error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("resultados = %#v, quería un slice vacío", got)
	}
}

func TestSearchAplicaExcepciones(t *testing.T) {
	html := fixtureBusqueda(t, "search.html")
	srv, peticiones := servidorHTML(t, html, "/buscar/bleach")
	// Mapa al estilo del real: clave sin barra final, valor con ella.
	mapa := []byte(`{"https://jkanime.net/bleach": "https://jkanime.net/bleach-corregido/"}`)
	srvMapa, peticionesMapa := servidorJSON(t, mapa)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{URL: srvMapa.URL, Doer: clientePrueba()})

	got, err := j.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("resultados = %d, quería 10", len(got))
	}
	if got[0].ID != "bleach-corregido" {
		t.Errorf("id del primer resultado = %q, quería el slug corregido por el mapa", got[0].ID)
	}
	if got[1].ID != "bleach-movie-3-fade-to-black-kimi-no-na-wo-yobu" {
		t.Errorf("id = %q, quería el slug original (no está en el mapa)", got[1].ID)
	}
	if n := peticionesMapa.Load(); n != 1 {
		t.Errorf("el mapa de excepciones se descargó %d veces, quería 1 (caché en memoria)", n)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones de búsqueda = %d, quería 1", n)
	}
}

func TestSearchErrorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error interno", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	_, err := j.Search(context.Background(), "bleach")
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
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	_, err := j.Search(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxSearchBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}

func TestMetodosNoImplementados(t *testing.T) {
	// Desde T17 no queda ningún método de Provider sin implementar:
	// GetAnimeDetails (T09) y GetVideoSources (T17) ya funcionan, así que
	// solo queda comprobar la identidad del proveedor.
	j := New(nil, "", nil)
	if j.Name() != "jkanime" {
		t.Errorf("Name() = %q, quería %q", j.Name(), providerName)
	}
}

func TestNewNormaliza(t *testing.T) {
	j := New(nil, "  https://ejemplo.test///  ", nil)
	if j.BaseURL != "https://ejemplo.test" {
		t.Errorf("BaseURL = %q, quería sin espacios ni barra final", j.BaseURL)
	}
	if j.Resolver == nil {
		t.Error("New(nil, …, nil) debía crear un resolver por defecto")
	}

	j = New(nil, "", nil)
	if j.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, quería %q", j.BaseURL, DefaultBaseURL)
	}

	inyectado := &ExceptionResolver{}
	j = New(nil, "", inyectado)
	if j.Resolver != inyectado {
		t.Error("New debía conservar el resolver inyectado")
	}

	// Valor cero de JkAnime (sin New): mismos criterios por defecto.
	cero := &JkAnime{}
	if got := cero.siteURL(); got != DefaultBaseURL {
		t.Errorf("siteURL() = %q, quería %q", got, DefaultBaseURL)
	}
	if cero.resolver() == nil {
		t.Error("resolver() con valor cero no debe ser nil")
	}
}
