package jkanime

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miapp/internal/httpx"
	"miapp/internal/logx"
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de httpx.Request ni con
	// el aviso de Resolve cuando el mapa remoto cae. No definir otro
	// TestMain en este paquete (ver PROGRESO.md).
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

// servidorJSON crea un servidor local que responde 200 con el body indicado
// y cuenta las peticiones recibidas.
func servidorJSON(t *testing.T, body []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &peticiones
}

// fixtureExcepciones carga el mapa real guardado en testdata.
func fixtureExcepciones(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/excepciones.json")
	if err != nil {
		t.Fatalf("no pude leer testdata/excepciones.json: %v", err)
	}
	return data
}

func TestResolveURLNormal(t *testing.T) {
	fixture := fixtureExcepciones(t)
	srv, _ := servidorJSON(t, fixture)
	r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}

	slug := "https://jkanime.net/one-piece/"
	got := r.Resolve(context.Background(), slug)
	if got.UsedException {
		t.Error("UsedException = true para una URL que no es excepción")
	}
	if got.URL != slug {
		t.Errorf("URL = %q, quería el slug tal cual %q", got.URL, slug)
	}
}

func TestResolveSustituyeURLRota(t *testing.T) {
	fixture := fixtureExcepciones(t)
	srv, _ := servidorJSON(t, fixture)
	r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}

	const (
		rota     = "https://jkanime.net/class-de-2-banme-ni-kawaii-onnanoko-to-tomodachi-n"
		correcta = "https://jkanime.net/class-de-2-banme-ni-kawaii-onnanoko-to-tomodachi-ni-natta/"
	)
	casos := []struct {
		nombre, entrada, queria string
	}{
		{"exacta", rota, correcta},
		{"con barra final", rota + "/", correcta},
		{"con espacios", "  " + rota + "  ", correcta},
		{"en mayúsculas", strings.ToUpper(rota), correcta},
		{"otra entrada del mapa", "https://jkanime.net/5-toubun-no-hanayome", "https://jkanime.net/5-toubun-no-hanayomee/"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := r.Resolve(context.Background(), c.entrada)
			if !got.UsedException {
				t.Error("UsedException = false, quería true")
			}
			if got.URL != c.queria {
				t.Errorf("URL = %q, quería %q", got.URL, c.queria)
			}
		})
	}
}

func TestResolveMapaCaido(t *testing.T) {
	casos := []struct {
		nombre string
		servir func(t *testing.T) string
	}{
		{"status 500", func(t *testing.T) string {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "error interno", http.StatusInternalServerError)
			}))
			t.Cleanup(srv.Close)
			return srv.URL
		}},
		{"servidor caído", func(t *testing.T) string {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			url := srv.URL
			srv.Close()
			return url
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := &ExceptionResolver{URL: c.servir(t), Doer: clientePrueba()}
			slug := "https://jkanime.net/class-de-2-banme-ni-kawaii-onnanoko-to-tomodachi-n"
			got := r.Resolve(context.Background(), slug)
			if got.UsedException {
				t.Error("UsedException = true con el mapa remoto caído")
			}
			if got.URL != slug {
				t.Errorf("URL = %q, quería el slug tal cual %q", got.URL, slug)
			}
		})
	}
}

func TestResolveRespuestaDemasiadoGrande(t *testing.T) {
	cuerpo := []byte(`{"https://jkanime.net/` + strings.Repeat("a", maxExceptionsBytes) + `": "https://jkanime.net/x/"}`)
	if len(cuerpo) <= maxExceptionsBytes {
		t.Fatalf("el cuerpo de prueba debe superar el límite: %d ≤ %d", len(cuerpo), maxExceptionsBytes)
	}
	srv, peticiones := servidorJSON(t, cuerpo)
	r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}

	slug := "https://jkanime.net/one-piece/"
	got := r.Resolve(context.Background(), slug)
	if got.UsedException {
		t.Error("UsedException = true con una respuesta que supera el límite")
	}
	if got.URL != slug {
		t.Errorf("URL = %q, quería el slug tal cual %q", got.URL, slug)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}
}

func TestResolveJSONInvalido(t *testing.T) {
	casos := []struct {
		nombre, cuerpo string
	}{
		{"json cortado", `{"https://jkanime.net/a": `},
		{"array en vez de objeto", `[1, 2, 3]`},
		{"valor que no es string", `{"https://jkanime.net/a": 5}`},
		{"null", `null`},
		{"texto", `"hola"`},
		{"vacío", ``},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, _ := servidorJSON(t, []byte(c.cuerpo))
			r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}
			slug := "https://jkanime.net/one-piece/"
			got := r.Resolve(context.Background(), slug)
			if got.UsedException {
				t.Error("UsedException = true con JSON inválido")
			}
			if got.URL != slug {
				t.Errorf("URL = %q, quería el slug tal cual %q", got.URL, slug)
			}
		})
	}
}

func TestResolveDescartaEntradasInvalidas(t *testing.T) {
	const cuerpo = `{
		"": "https://jkanime.net/vacia/",
		"   ": "https://jkanime.net/solo-espacios/",
		"https://jkanime.net/valida": "https://jkanime.net/valida-corregida/",
		"https://jkanime.net/valor-ftp": "ftp://ejemplo.test/archivo.zip",
		"https://jkanime.net/valor-relativo": "/ruta/relativa/",
		"https://jkanime.net/valor-sin-host": "https://",
		"https://jkanime.net/valor-vacio": "",
		"ftp://ejemplo.test/clave-ftp": "https://jkanime.net/clave-ftp-ok/",
		"javascript:alert(1)": "https://jkanime.net/js/",
		"jkanime.net/sin-esquema": "https://jkanime.net/sin-esquema-ok/"
	}`
	srv, _ := servidorJSON(t, []byte(cuerpo))
	r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}

	validas := []struct {
		entrada, queria string
	}{
		{"https://jkanime.net/valida/", "https://jkanime.net/valida-corregida/"},
		{"HTTPS://Jkanime.Net/Valida", "https://jkanime.net/valida-corregida/"},
	}
	for _, c := range validas {
		got := r.Resolve(context.Background(), c.entrada)
		if !got.UsedException {
			t.Errorf("entrada %q: UsedException = false, quería true", c.entrada)
		}
		if got.URL != c.queria {
			t.Errorf("entrada %q: URL = %q, quería %q", c.entrada, got.URL, c.queria)
		}
	}

	descartadas := []string{
		"https://jkanime.net/valor-ftp/",
		"https://jkanime.net/valor-relativo/",
		"https://jkanime.net/valor-sin-host/",
		"https://jkanime.net/valor-vacio/",
		"ftp://ejemplo.test/clave-ftp",
		"javascript:alert(1)",
		"jkanime.net/sin-esquema",
		"",
		"   ",
	}
	for _, entrada := range descartadas {
		got := r.Resolve(context.Background(), entrada)
		if got.UsedException {
			t.Errorf("entrada %q no debía resolverse como excepción", entrada)
		}
		if got.URL != entrada {
			t.Errorf("entrada %q: URL = %q, quería el slug tal cual", entrada, got.URL)
		}
	}
}

func TestResolveDescargaUnaSolaVez(t *testing.T) {
	fixture := fixtureExcepciones(t)
	srv, peticiones := servidorJSON(t, fixture)
	r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}

	for i := 0; i < 5; i++ {
		got := r.Resolve(context.Background(), "https://jkanime.net/5-toubun-no-hanayome")
		if !got.UsedException {
			t.Fatalf("intento %d: no aplicó la excepción", i)
		}
		if got.URL != "https://jkanime.net/5-toubun-no-hanayomee/" {
			t.Fatalf("intento %d: URL = %q", i, got.URL)
		}
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1 (la caché en memoria debe evitar repetir la descarga)", n)
	}
}

func TestResolveConcurrente(t *testing.T) {
	fixture := fixtureExcepciones(t)
	srv, peticiones := servidorJSON(t, fixture)
	r := &ExceptionResolver{URL: srv.URL, Doer: clientePrueba()}

	const n = 50
	resultados := make([]ResolvedAnimeURL, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			resultados[i] = r.Resolve(context.Background(), "https://jkanime.net/5-toubun-no-hanayome")
		}(i)
	}
	wg.Wait()

	for i, got := range resultados {
		if !got.UsedException || got.URL != "https://jkanime.net/5-toubun-no-hanayomee/" {
			t.Errorf("resultado %d: %+v, quería la excepción aplicada", i, got)
		}
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1 (mutex durante la descarga)", n)
	}
}

func TestResolveTimeoutCorto(t *testing.T) {
	var conDeadline bool
	var restante time.Duration
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		conDeadline, restante = ok, time.Until(deadline)
		return nil, errors.New("sin red (doer de prueba)")
	}}
	r := &ExceptionResolver{URL: "https://ejemplo.test/excepciones.json", Doer: d}

	slug := "https://jkanime.net/one-piece/"
	got := r.Resolve(context.Background(), slug)
	if got.UsedException || got.URL != slug {
		t.Errorf("Resultado = %+v, quería el slug tal cual sin excepción", got)
	}
	if !conDeadline {
		t.Fatal("la descarga no tenía deadline")
	}
	if restante <= 0 || restante > exceptionsTimeout {
		t.Errorf("deadline a %v, quería 0 < d ≤ %v", restante, exceptionsTimeout)
	}
}

func TestURLPorDefecto(t *testing.T) {
	const esperada = "https://raw.githubusercontent.com/Zhuchii/ani-es/refs/heads/main/excepciones.json"
	if DefaultExceptionsURL != esperada {
		t.Fatalf("DefaultExceptionsURL = %q, quería %q", DefaultExceptionsURL, esperada)
	}

	var pedida string
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		pedida = req.URL.String()
		return nil, errors.New("sin red (doer de prueba)")
	}}
	r := &ExceptionResolver{Doer: d} // URL vacía → la constante por defecto.
	r.Resolve(context.Background(), "https://jkanime.net/one-piece/")
	if pedida != DefaultExceptionsURL {
		t.Errorf("pidió %q, quería %q", pedida, DefaultExceptionsURL)
	}
}

func TestSinDoerNoHayRed(t *testing.T) {
	r := &ExceptionResolver{}
	slug := "https://jkanime.net/one-piece/"
	got := r.Resolve(context.Background(), slug)
	if got.UsedException {
		t.Error("UsedException = true sin Doer")
	}
	if got.URL != slug {
		t.Errorf("URL = %q, quería el slug tal cual %q", got.URL, slug)
	}
}
