package httpx

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"time"
)

// Valores por defecto de Policy: máximo 3 intentos con backoff exponencial.
const (
	defaultMaxAttempts = 3
	defaultBaseDelay   = 250 * time.Millisecond
	defaultMaxDelay    = 5 * time.Second
)

// Policy describe cómo reintenta Retry. El valor cero aplica los defaults:
// 3 intentos, backoff de 250ms duplicándose hasta 5s, con jitter.
type Policy struct {
	// MaxAttempts es el número total de intentos (0 → 3).
	MaxAttempts int
	// BaseDelay es la espera tras el primer fallo (0 → 250ms); se duplica en
	// cada intento siguiente hasta MaxDelay.
	BaseDelay time.Duration
	// MaxDelay limita el backoff (0 → 5s). No limita la espera que pide el
	// servidor con Retry-After: esa se respeta tal cual.
	MaxDelay time.Duration
	// Sleep, si se indica, sustituye la espera (p. ej. para tests). Debe
	// respetar ctx y devolver su error si se cancela.
	Sleep func(ctx context.Context, d time.Duration) error
}

// Retry ejecuta fn hasta que devuelva nil, se acaben los intentos o se cancele
// ctx. Solo reintenta errores reintentables (ver Retryable): los fallos de un
// POST no idempotente llegan marcados como no reintentables desde Post.
//
// Entre intentos espera con backoff exponencial y jitter, y respeta la espera
// mínima de Retry-After si el error la trae.
func Retry(ctx context.Context, p Policy, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	max := p.MaxAttempts
	if max <= 0 {
		max = defaultMaxAttempts
	}
	sleep := p.Sleep
	if sleep == nil {
		sleep = sleepCtx
	}

	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if attempt >= max || !Retryable(err) {
			return err
		}
		delay := p.backoff(attempt)
		if ra := retryAfterOf(err); ra > delay {
			delay = ra
		}
		if err := sleep(ctx, delay); err != nil {
			return err
		}
	}
}

// Retryable dice si un error admite reintento: un ScraperError lo decide su
// campo Retryable; los errores de red sueltos sí (transitorios); los de
// contexto (cancelación) y el resto, no.
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	var se *ScraperError
	if errors.As(err, &se) {
		return se.Retryable
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	return false
}

// backoff devuelve la espera tras el intento `attempt` (1-based): base,
// doble, doble... hasta MaxDelay, con jitter entre la mitad y el total.
func (p Policy) backoff(attempt int) time.Duration {
	base := p.BaseDelay
	if base <= 0 {
		base = defaultBaseDelay
	}
	max := p.MaxDelay
	if max <= 0 {
		max = defaultMaxDelay
	}
	d := base
	for i := 1; i < attempt; i++ {
		if d >= max {
			break
		}
		d *= 2
	}
	if d > max {
		d = max
	}
	half := d / 2
	return half + time.Duration(rand.Int64N(int64(half)+1))
}

// retryAfterOf extrae la espera mínima de un ScraperError (429 Retry-After).
func retryAfterOf(err error) time.Duration {
	var se *ScraperError
	if errors.As(err, &se) {
		return se.retryAfter
	}
	return 0
}

// sleepCtx espera d respetando la cancelación del contexto.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
