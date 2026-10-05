package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

	"miapp/internal/logx"
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de logx.Request.
	logx.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func servidorPrueba(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func errorDe(t *testing.T, err error) *ScraperError {
	t.Helper()
	if err == nil {
		t.Fatal("se esperaba un error y no lo hay")
	}
	var se *ScraperError
	if !errors.As(err, &se) {
		t.Fatalf("se esperaba *ScraperError y llego %T: %v", err, err)
	}
	return se
}

func TestGetOKEnviaUserAgentYReferer(t *testing.T) {
	var mu sync.Mutex
	var ua, referer string
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ua, referer = r.Header.Get("User-Agent"), r.Header.Get("Referer")
		mu.Unlock()
		fmt.Fprint(w, "listo")
	})

	resp, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL,
		ReqOptions{Provider: "jkanime", Operation: "search", Referer: "https://ejemplo.test/"})
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, quería 200", resp.StatusCode)
	}
	cuerpo, err := io.ReadAll(resp.Body)
	if err != nil || string(cuerpo) != "listo" {
		t.Errorf("cuerpo = %q (err %v), quería %q", cuerpo, err, "listo")
	}
	mu.Lock()
	defer mu.Unlock()
	if ua != DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería el predeterminado del paquete", ua)
	}
	if referer != "https://ejemplo.test/" {
		t.Errorf("Referer = %q, quería el referer indicado", referer)
	}
}

func TestGet404Definitivo(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	resp, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	if resp != nil {
		t.Errorf("la respuesta debería ser nil en error: %v", resp)
	}
	se := errorDe(t, err)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false: %v", err)
	}
	if se.StatusCode != 404 {
		t.Errorf("StatusCode = %d, quería 404", se.StatusCode)
	}
	if se.Retryable {
		t.Error("404 no debe ser reintentable")
	}
}

func TestGet403NoReintentable(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("errors.Is(err, ErrForbidden) = false: %v", err)
	}
	if se.Retryable {
		t.Error("403 sin marcas de Cloudflare no debe ser reintentable")
	}
}

func TestGet419Recuperable(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(419)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("errors.Is(err, ErrSessionExpired) = false: %v", err)
	}
	if !se.Retryable {
		t.Error("419 es recuperable (refrescando sesión) y debe quedar reintentable")
	}
}

func TestGet429ConRetryAfter(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, ErrTooManyRequests) {
		t.Errorf("errors.Is(err, ErrTooManyRequests) = false: %v", err)
	}
	if !se.Retryable {
		t.Error("429 debe ser reintentable")
	}
	if se.RetryAfter() != 2*time.Second {
		t.Errorf("RetryAfter() = %s, quería 2s", se.RetryAfter())
	}
}

func TestGet429ConRetryAfterFecha(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", time.Now().Add(3*time.Second).UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if se.RetryAfter() <= 0 || se.RetryAfter() > 4*time.Second {
		t.Errorf("RetryAfter() = %s, quería una espera de unos segundos", se.RetryAfter())
	}
}

func TestGet503Reintentable(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, ErrServerError) {
		t.Errorf("errors.Is(err, ErrServerError) = false: %v", err)
	}
	if !se.Retryable {
		t.Error("5xx debe ser reintentable")
	}
}

func TestCloudflarePorCuerpo(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "<html><title>Just a moment...</title></html>")
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, ErrCloudflareChallenge) {
		t.Errorf("errors.Is(err, ErrCloudflareChallenge) = false: %v", err)
	}
	if !se.Retryable {
		t.Error("el challenge de Cloudflare es la excepción reintentable del 403")
	}
}

func TestCloudflarePorCabecera(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, ErrCloudflareChallenge) {
		t.Errorf("errors.Is(err, ErrCloudflareChallenge) = false: %v", err)
	}
	if se.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, quería 503", se.StatusCode)
	}
}

func TestCloudflareAunqueElStatusSea200(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<title>Just a moment...</title>")
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	if !errors.Is(err, ErrCloudflareChallenge) {
		t.Errorf("errors.Is(err, ErrCloudflareChallenge) = false: %v", err)
	}
}

func TestCookiesSeMantienenEntrePeticiones(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sesion", Value: "valor_sesion", Path: "/"})
	})
	mux.HandleFunc("/lee", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("sesion")
		if err != nil {
			http.Error(w, "sin cookie", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, c.Value)
	})
	srv := servidorPrueba(t, mux.ServeHTTP)

	cliente := NewClient(ClientOptions{})
	if cliente.Jar == nil {
		t.Fatal("NewClient debe crear un cookiejar por defecto")
	}
	if _, err := Get(context.Background(), cliente, srv.URL+"/set", ReqOptions{}); err != nil {
		t.Fatalf("GET /set falló: %v", err)
	}
	resp, err := Get(context.Background(), cliente, srv.URL+"/lee", ReqOptions{})
	if err != nil {
		t.Fatalf("GET /lee falló: %v", err)
	}
	defer resp.Body.Close()
	cuerpo, _ := io.ReadAll(resp.Body)
	if string(cuerpo) != "valor_sesion" {
		t.Errorf("la cookie no viajó entre peticiones: cuerpo = %q", cuerpo)
	}
}

func TestPostSoloReintentableSiEsIdempotente(t *testing.T) {
	var intentos atomic.Int32
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		intentos.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	cliente := NewClient(ClientOptions{})

	_, err := Post(context.Background(), cliente, srv.URL, nil, ReqOptions{Operation: "post"})
	se := errorDe(t, err)
	if se.Retryable {
		t.Error("un POST sin declarar idempotencia no debe ser reintentable")
	}

	intentos.Store(0)
	_, err = Post(context.Background(), cliente, srv.URL, nil,
		ReqOptions{Operation: "post", Idempotent: true})
	se = errorDe(t, err)
	if !se.Retryable {
		t.Error("un POST declarado idempotente sí debe ser reintentable")
	}
}

func TestFalloDeTransporte(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {})
	url := srv.URL
	srv.Close() // ya no hay nadie escuchando

	_, err := Get(context.Background(), NewClient(ClientOptions{}), url, ReqOptions{})
	se := errorDe(t, err)
	if se.StatusCode != 0 {
		t.Errorf("StatusCode = %d, quería 0 (sin respuesta)", se.StatusCode)
	}
	if !se.Retryable {
		t.Error("un fallo de red transitorio debe ser reintentable")
	}
}

func TestGetConContextoCancelado(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Get(ctx, NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	se := errorDe(t, err)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false: %v", err)
	}
	if se.Retryable {
		t.Error("una cancelación no debe ser reintentable")
	}
}

func TestNewClientTransportReemplazable(t *testing.T) {
	original := NewTransport
	t.Cleanup(func() { NewTransport = original })

	propio := &http.Transport{}
	NewTransport = func() http.RoundTripper { return propio }
	c := NewClient(ClientOptions{})
	if c.Transport != propio {
		t.Errorf("NewClient no usó la fábrica NewTransport: %T", c.Transport)
	}
	if c.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %s, quería %s", c.Timeout, DefaultTimeout)
	}
	if c.Jar == nil {
		t.Error("NewClient debe montar cookiejar")
	}

	otro := &http.Transport{}
	c2 := NewClient(ClientOptions{Transport: otro, Timeout: time.Second})
	if c2.Transport != otro || c2.Timeout != time.Second {
		t.Error("ClientOptions debe poder sustituir transporte y timeout")
	}
	NewTransport = original
}

func TestErrorRedactaLaURL(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	_, err := Get(context.Background(), NewClient(ClientOptions{}),
		srv.URL+"/anime?token=SECRETO_URL&other=1",
		ReqOptions{Provider: "jkanime", Operation: "buscar"})
	errorDe(t, err)
	msg := err.Error()
	if strings.Contains(msg, "SECRETO_URL") {
		t.Errorf("el token aparece en el mensaje de error: %q", msg)
	}
	if !strings.Contains(msg, "token=***") {
		t.Errorf("falta la máscara en el mensaje: %q", msg)
	}
	if !strings.Contains(msg, "jkanime/buscar") {
		t.Errorf("falta provider/operación en el mensaje: %q", msg)
	}
}

func TestCuerpoGrandeNoSePierde(t *testing.T) {
	// Más grande que sniffLimit (64 KiB): la lectura por delante no debe
	// comerse ni un byte.
	datos := bytes.Repeat([]byte("anime"), 40000)
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(datos)
	})
	resp, err := Get(context.Background(), NewClient(ClientOptions{}), srv.URL, ReqOptions{})
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	defer resp.Body.Close()
	leido, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leyendo el cuerpo: %v", err)
	}
	if !bytes.Equal(leido, datos) {
		t.Errorf("cuerpo alterado: %d bytes leídos, quería %d", len(leido), len(datos))
	}
}
