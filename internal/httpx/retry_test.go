package httpx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryExitoTrasFalloTransitorio(t *testing.T) {
	var intentos, esperas atomic.Int32
	politica := Policy{
		Sleep: func(ctx context.Context, d time.Duration) error {
			esperas.Add(1)
			return nil
		},
	}
	err := Retry(context.Background(), politica, func(ctx context.Context) error {
		if intentos.Add(1) == 1 {
			return &ScraperError{StatusCode: 503, Retryable: true, Cause: ErrServerError}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Retry devolvió error: %v", err)
	}
	if intentos.Load() != 2 {
		t.Errorf("intentos = %d, quería 2", intentos.Load())
	}
	if esperas.Load() != 1 {
		t.Errorf("esperas = %d, quería 1", esperas.Load())
	}
}

func TestRetryMaximoTresIntentos(t *testing.T) {
	var intentos atomic.Int32
	fallo := &ScraperError{StatusCode: 503, Retryable: true, Cause: ErrServerError}
	err := Retry(context.Background(), Policy{
		Sleep: func(ctx context.Context, d time.Duration) error { return nil },
	}, func(ctx context.Context) error {
		intentos.Add(1)
		return fallo
	})
	if !errors.Is(err, ErrServerError) {
		t.Errorf("error final = %v, quería el último fallo", err)
	}
	if intentos.Load() != 3 {
		t.Errorf("intentos = %d, quería 3 (máximo)", intentos.Load())
	}
}

func TestRetryNoReintentaErrorDefinitivo(t *testing.T) {
	var intentos atomic.Int32
	err := Retry(context.Background(), Policy{
		Sleep: func(ctx context.Context, d time.Duration) error { return nil },
	}, func(ctx context.Context) error {
		intentos.Add(1)
		return &ScraperError{StatusCode: 404, Retryable: false, Cause: ErrNotFound}
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, quería ErrNotFound", err)
	}
	if intentos.Load() != 1 {
		t.Errorf("intentos = %d, quería 1: un 404 es definitivo", intentos.Load())
	}
}

func TestRetryRespetaCancelacionPrevia(t *testing.T) {
	var intentos atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Retry(ctx, Policy{
		Sleep: func(ctx context.Context, d time.Duration) error { return nil },
	}, func(ctx context.Context) error {
		intentos.Add(1)
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, quería context.Canceled", err)
	}
	if intentos.Load() != 0 {
		t.Errorf("intentos = %d, quería 0: el ctx ya estaba cancelado", intentos.Load())
	}
}

func TestRetryCancelaDuranteLaEspera(t *testing.T) {
	var intentos atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	err := Retry(ctx, Policy{BaseDelay: time.Hour}, func(ctx context.Context) error {
		intentos.Add(1)
		cancel() // el usuario cierra la app mientras espera
		return &ScraperError{StatusCode: 503, Retryable: true, Cause: ErrServerError}
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, quería context.Canceled", err)
	}
	if intentos.Load() != 1 {
		t.Errorf("intentos = %d, quería 1: la espera debe cortarse al cancelar", intentos.Load())
	}
}

func TestRetryNoReintentaPostNoIdempotente(t *testing.T) {
	var peticiones atomic.Int32
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	cliente := NewClient(ClientOptions{})

	err := Retry(context.Background(), Policy{
		Sleep: func(ctx context.Context, d time.Duration) error { return nil },
	}, func(ctx context.Context) error {
		_, e := Post(ctx, cliente, srv.URL, nil, ReqOptions{Operation: "post"})
		return e
	})
	if err == nil {
		t.Fatal("se esperaba un error")
	}
	if peticiones.Load() != 1 {
		t.Errorf("peticiones al servidor = %d, quería 1: un POST no idempotente no se reintenta a ciegas", peticiones.Load())
	}
	var se *ScraperError
	if !errors.As(err, &se) || se.Retryable {
		t.Errorf("el error no debe quedar marcado como reintentable: %v", err)
	}
}

func TestRetryReintentaPostIdempotente(t *testing.T) {
	var peticiones atomic.Int32
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	cliente := NewClient(ClientOptions{})

	err := Retry(context.Background(), Policy{
		Sleep: func(ctx context.Context, d time.Duration) error { return nil },
	}, func(ctx context.Context) error {
		_, e := Post(ctx, cliente, srv.URL, nil, ReqOptions{Operation: "post", Idempotent: true})
		return e
	})
	if err == nil {
		t.Fatal("se esperaba un error")
	}
	if peticiones.Load() != 3 {
		t.Errorf("peticiones al servidor = %d, quería 3", peticiones.Load())
	}
}

func TestRetryRespetaRetryAfter(t *testing.T) {
	srv := servidorPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	cliente := NewClient(ClientOptions{})

	var esperas []time.Duration
	err := Retry(context.Background(), Policy{
		Sleep: func(ctx context.Context, d time.Duration) error {
			esperas = append(esperas, d)
			return nil
		},
	}, func(ctx context.Context) error {
		_, e := Get(ctx, cliente, srv.URL, ReqOptions{Operation: "search"})
		return e
	})
	if err == nil {
		t.Fatal("se esperaba un error")
	}
	if len(esperas) != 2 {
		t.Fatalf("esperas = %d, quería 2", len(esperas))
	}
	for i, d := range esperas {
		if d < 2*time.Second {
			t.Errorf("espera %d = %s, debía respetar Retry-After (2s)", i+1, d)
		}
	}
}

func TestRetryBackoffExponencialConJitter(t *testing.T) {
	var esperas []time.Duration
	err := Retry(context.Background(), Policy{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  10 * time.Second,
		Sleep: func(ctx context.Context, d time.Duration) error {
			esperas = append(esperas, d)
			return nil
		},
	}, func(ctx context.Context) error {
		return &ScraperError{StatusCode: 503, Retryable: true, Cause: ErrServerError}
	})
	if err == nil {
		t.Fatal("se esperaba un error")
	}
	if len(esperas) != 2 {
		t.Fatalf("esperas = %d, quería 2", len(esperas))
	}
	if esperas[0] < 50*time.Millisecond || esperas[0] > 100*time.Millisecond {
		t.Errorf("1ª espera = %s, quería entre 50ms y 100ms", esperas[0])
	}
	if esperas[1] < 100*time.Millisecond || esperas[1] > 200*time.Millisecond {
		t.Errorf("2ª espera = %s, quería entre 100ms y 200ms", esperas[1])
	}
}

func TestRetryable(t *testing.T) {
	if Retryable(nil) {
		t.Error("nil no es reintentable")
	}
	if !Retryable(&ScraperError{Retryable: true}) {
		t.Error("ScraperError reintentable debe ser reintentable")
	}
	if Retryable(&ScraperError{StatusCode: 404}) {
		t.Error("ScraperError sin Retryable no debe ser reintentable")
	}
	if Retryable(fmt.Errorf("envuelto: %w", &ScraperError{Retryable: false, Cause: ErrNotFound})) {
		t.Error("el error envuelto conserva su clasificación")
	}
	if !Retryable(fmt.Errorf("envuelto: %w", &ScraperError{Retryable: true})) {
		t.Error("el error envuelto reintentable debe ser reintentable")
	}
	if Retryable(context.Canceled) {
		t.Error("la cancelación no se reintenta")
	}
	if Retryable(errors.New("fallo cualquiera")) {
		t.Error("un error desconocido no se reintenta")
	}
	if !Retryable(&net.OpError{Op: "dial", Err: errors.New("conexión reiniciada")}) {
		t.Error("un fallo de red suelto sí debe ser reintentable")
	}
}
