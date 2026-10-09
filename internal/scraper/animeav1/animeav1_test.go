package animeav1

import (
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
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de httpx.Request. No
	// definir otro TestMain en este paquete (mismo criterio que jkanime).
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

// fixture carga un JSON real de testdata (respuesta cruda del servidor).
func fixture(t *testing.T, nombre string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatalf("no pude leer testdata/%s: %v", nombre, err)
	}
	return data
}

// servidorJSON sirve body en la ruta indicada y cuenta las peticiones. Si la
// ruta pedida no coincide, el test falla (así se comprueba la URL que arma
// Search sin estado compartido entre goroutines).
func servidorJSON(t *testing.T, body []byte, rutaEsperada string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		if got := r.URL.EscapedPath(); got != rutaEsperada {
			t.Errorf("ruta pedida = %q, quería %q", got, rutaEsperada)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &peticiones
}

func TestName(t *testing.T) {
	a := New(nil, "")
	if a.Name() != "animeav1" {
		t.Errorf("Name() = %q, quería %q", a.Name(), providerName)
	}
}

func TestNewNormaliza(t *testing.T) {
	a := New(nil, "  https://ejemplo.test///  ")
	if a.BaseURL != "https://ejemplo.test" {
		t.Errorf("BaseURL = %q, quería sin espacios ni barra final", a.BaseURL)
	}

	a = New(nil, "")
	if a.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, quería %q", a.BaseURL, DefaultBaseURL)
	}
	if DefaultBaseURL != "https://animeav1.com" {
		t.Errorf("DefaultBaseURL = %q, quería https://animeav1.com", DefaultBaseURL)
	}

	// Valor cero de AnimeAV1 (sin New): misma base por defecto.
	cero := &AnimeAV1{}
	if got := cero.siteURL(); got != DefaultBaseURL {
		t.Errorf("siteURL() = %q, quería %q", got, DefaultBaseURL)
	}
}

func TestSearchRutaYResultados(t *testing.T) {
	srv, peticiones := servidorJSON(t, fixture(t, "catalogo_data.json"), "/catalogo/__data.json")
	a := New(clientePrueba(), srv.URL)

	got, err := a.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}
	if len(got) != 11 {
		t.Fatalf("resultados = %d, quería 11 (los del fixture)", len(got))
	}

	// id = slug del sitio (results[].id es string y solo sirve de portada),
	// nunca derivado del título.
	primero := got[0]
	if primero.ID != "bleach" || primero.Title != "Bleach" {
		t.Errorf("primero = %+v, quería id=bleach title=Bleach", primero)
	}
	if primero.CoverURL != "https://cdn.animeav1.com/covers/195.jpg" {
		t.Errorf("CoverURL = %q, quería la portada del CDN con el id del sitio", primero.CoverURL)
	}
	if primero.Provider != providerName {
		t.Errorf("Provider = %q, quería %q", primero.Provider, providerName)
	}

	encontrado := false
	for _, r := range got {
		if r.ID == "bleach-movie-1-memories-of-nobody" {
			encontrado = true
			if r.Title != "Bleach Movie 1: Memories of Nobody" {
				t.Errorf("título de la película = %q", r.Title)
			}
			if r.CoverURL != "https://cdn.animeav1.com/covers/905.jpg" {
				t.Errorf("portada de la película = %q", r.CoverURL)
			}
		}
	}
	if !encontrado {
		t.Error("no encontré el resultado con id bleach-movie-1-memories-of-nobody")
	}

	for i, r := range got {
		if r.ID == "" || r.Title == "" || r.Provider != providerName {
			t.Errorf("resultado %d incompleto: %+v", i, r)
		}
		if strings.Contains(r.ID, "/") || strings.HasPrefix(r.ID, "http") {
			t.Errorf("resultado %d: el id %q debe ser el slug, sin barra ni dominio", i, r.ID)
		}
		if r.CoverURL != "" && !strings.HasPrefix(r.CoverURL, "https://cdn.animeav1.com/covers/") {
			t.Errorf("resultado %d: CoverURL = %q, quería el CDN", i, r.CoverURL)
		}
	}
}

func TestSearchEscapaLaQueryYUsaUserAgent(t *testing.T) {
	casos := []struct {
		nombre, query, consulta string
	}{
		{"simple", "bleach", "search=bleach"},
		{"espacio", "one piece", "search=one+piece"},
		{"ampersand", "a&b", "search=a%26b"},
		{"signos", "steins;gate?", "search=steins%3Bgate%3F"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var consulta, ua string
			var peticiones int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				peticiones++
				if r.URL.EscapedPath() != "/catalogo/__data.json" {
					t.Errorf("ruta pedida = %q", r.URL.EscapedPath())
				}
				consulta = r.URL.RawQuery
				ua = r.Header.Get("User-Agent")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(fixture(t, "catalogo_data.json"))
			}))
			t.Cleanup(srv.Close)
			a := New(clientePrueba(), srv.URL)

			got, err := a.Search(context.Background(), c.query)
			if err != nil {
				t.Fatalf("Search falló: %v", err)
			}
			if peticiones != 1 {
				t.Errorf("peticiones = %d, quería 1", peticiones)
			}
			if consulta != c.consulta {
				t.Errorf("consulta = %q, quería %q", consulta, c.consulta)
			}
			if ua != httpx.DefaultUserAgent {
				t.Errorf("User-Agent = %q, quería el de httpx", ua)
			}
			if len(got) != 11 {
				t.Errorf("resultados = %d, quería 11", len(got))
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
	srv, _ := servidorJSON(t, fixture(t, "catalogo_vacio_data.json"), "/catalogo/__data.json")
	a := New(clientePrueba(), srv.URL)

	got, err := a.Search(context.Background(), "zzzznoexisteestetitulo")
	if err != nil {
		t.Fatalf("un catálogo vacío no debe ser error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("resultados = %#v, quería un slice vacío", got)
	}
}

func TestSearchJSONCorrupto(t *testing.T) {
	srv, _ := servidorJSON(t, []byte(`{"type":"data","nodes":`), "/catalogo/__data.json")
	a := New(clientePrueba(), srv.URL)

	_, err := a.Search(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un JSON cortado")
	}
	if !strings.Contains(err.Error(), "parseando el __data.json") {
		t.Errorf("err = %v, quería el contexto de parseo", err)
	}
}

func TestSearchRespuestaInesperada(t *testing.T) {
	casos := []struct {
		nombre, cuerpo, contiene string
	}{
		{"type de error", `{"type":"error","nodes":[]}`, `type = "error"`},
		{"sin nodo de página", `{"type":"data","nodes":[null]}`, "nodo de página"},
		{"nodo skip", `{"type":"data","nodes":[{"type":"skip"}]}`, "nodo de página"},
		{"nodo data sin array", `{"type":"data","nodes":[{"type":"data"}]}`, "array data"},
		{"raíz que no es objeto", `{"type":"data","nodes":[{"type":"data","data":[1,2]}]}`, "no son un objeto"},
		{"referencia fuera de rango", `{"type":"data","nodes":[{"type":"data","data":[{"x":99}]}]}`, "fuera del array"},
		{"sin results", `{"type":"data","nodes":[{"type":"data","data":[{"total":1},5]}]}`, "no trae results"},
		{"results que no es lista", `{"type":"data","nodes":[{"type":"data","data":[{"results":1},"x"]} ]}`, "quería una lista"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, _ := servidorJSON(t, []byte(c.cuerpo), "/catalogo/__data.json")
			a := New(clientePrueba(), srv.URL)

			got, err := a.Search(context.Background(), "bleach")
			if err == nil {
				t.Fatalf("quería error con %s", c.nombre)
			}
			if got != nil {
				t.Errorf("resultados = %#v, quería nil cuando falla", got)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}

func TestSearchErrorHTTP(t *testing.T) {
	casos := []struct {
		nombre string
		status int
		queria error
	}{
		{"404", http.StatusNotFound, httpx.ErrNotFound},
		{"500", http.StatusInternalServerError, httpx.ErrServerError},
		{"403", http.StatusForbidden, httpx.ErrForbidden},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "error", c.status)
			}))
			t.Cleanup(srv.Close)
			a := New(clientePrueba(), srv.URL)

			_, err := a.Search(context.Background(), "bleach")
			if err == nil {
				t.Fatalf("quería error con un %d", c.status)
			}
			if !errors.Is(err, c.queria) {
				t.Errorf("err = %v, quería %v", err, c.queria)
			}
		})
	}
}

func TestSearchCtxCancelado(t *testing.T) {
	var peticiones int
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		peticiones++
		<-req.Context().Done()
		return nil, req.Context().Err()
	}}
	a := New(d, DefaultBaseURL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.Search(ctx, "bleach")
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería context.Canceled", err)
	}
	if peticiones != 1 {
		t.Errorf("peticiones = %d, quería 1 (la petición se lanza y cancela)", peticiones)
	}
}

func TestSearchSinDoer(t *testing.T) {
	a := New(nil, "")
	_, err := a.Search(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error sin doer")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer", err)
	}
}

func TestSearchCuerpoDemasiadoGrande(t *testing.T) {
	cuerpo := strings.Repeat("a", maxBusquedaBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, cuerpo)
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	_, err := a.Search(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxBusquedaBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}
