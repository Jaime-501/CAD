package httpx

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"miapp/internal/logx"
)

// Causas tipadas que puede llevar ScraperError. Se comprueban con
// errors.Is(err, httpx.ErrNotFound) y compañía.
var (
	// ErrCloudflareChallenge: el sitio devolvió un challenge de Cloudflare
	// (cabecera cf-mitigated o cuerpo "Just a moment...") en vez del recurso.
	ErrCloudflareChallenge = errors.New("challenge de Cloudflare")
	ErrForbidden           = errors.New("acceso denegado (403)")
	ErrNotFound            = errors.New("recurso no encontrado (404)")
	ErrSessionExpired      = errors.New("sesión caducada (419): requiere refrescar la sesión")
	ErrTooManyRequests     = errors.New("demasiadas peticiones (429)")
	ErrServerError         = errors.New("error interno del servidor (5xx)")
	ErrClientError         = errors.New("petición incorrecta (4xx)")
	ErrNilDoer             = errors.New("HTTPDoer nulo")
	ErrNilResponse         = errors.New("respuesta HTTP nula")
)

// ScraperError es el error tipado de cualquier fallo HTTP del scraper.
// Provider/Operation/URL identifican la llamada, StatusCode es 0 si no hubo
// respuesta, Retryable indica si el fallo admite reintento (política de
// Retry) y Cause es la causa subyacente (sentinela o error de red).
type ScraperError struct {
	Provider   string
	Operation  string
	URL        string
	StatusCode int
	Retryable  bool
	Cause      error

	// retryAfter es la espera mínima indicada por el servidor (Retry-After
	// del 429). Se expone con RetryAfter().
	retryAfter time.Duration
}

// Error implementa error. La URL se redacta para que un token en la query
// no acabe en logs de otra capa.
func (e *ScraperError) Error() string {
	op := e.Operation
	switch {
	case e.Provider != "" && op != "":
		op = e.Provider + "/" + op
	case e.Provider != "":
		op = e.Provider
	}
	if op != "" {
		op += " "
	}
	status := "sin respuesta"
	if e.StatusCode > 0 {
		status = strconv.Itoa(e.StatusCode)
	}
	msg := fmt.Sprintf("httpx: %s%s: status %s", op, logx.Redact(e.URL), status)
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap permite errors.Is/As contra la causa (p. ej. ErrNotFound).
func (e *ScraperError) Unwrap() error { return e.Cause }

// RetryAfter devuelve la espera mínima que pidió el servidor (429
// Retry-After); 0 si no la hubo.
func (e *ScraperError) RetryAfter() time.Duration { return e.retryAfter }

// classify convierte la respuesta en un *ScraperError tipado, o nil si la
// respuesta es utilizable. prefix es el inicio del cuerpo ya leído (para
// detectar el texto del challenge) y allowRetry recoge si el llamador permite
// reintentos: GET sí, POST solo si declara idempotencia.
//
// Clasificación: 403 no reintentable (salvo Cloudflare), 404 definitivo,
// 419 recuperable (requiere refrescar sesión, ver T09), 429 reintentable
// respetando Retry-After y 5xx reintentable.
func classify(opts ReqOptions, url string, resp *http.Response, prefix []byte, allowRetry bool) *ScraperError {
	status := resp.StatusCode
	se := &ScraperError{
		Provider:   opts.Provider,
		Operation:  opts.Operation,
		URL:        url,
		StatusCode: status,
	}

	if isCloudflare(resp.Header, prefix) {
		se.Cause = ErrCloudflareChallenge
		// El challenge es la excepción reintentable del 403 (y puede
		// aparecer con cualquier status, incluso 200).
		se.Retryable = allowRetry
		return se
	}
	if status < 400 {
		return nil
	}

	switch {
	case status == http.StatusForbidden: // 403
		se.Cause = ErrForbidden
	case status == http.StatusNotFound: // 404
		se.Cause = ErrNotFound
	case status == 419: // 419: sesión caducada
		se.Cause = ErrSessionExpired
		se.Retryable = allowRetry
	case status == http.StatusTooManyRequests: // 429
		se.Cause = ErrTooManyRequests
		se.Retryable = allowRetry
		se.retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	case status >= 500:
		se.Cause = ErrServerError
		se.Retryable = allowRetry
	default: // resto de 4xx
		se.Cause = ErrClientError
	}
	return se
}

// isCloudflare detecta un challenge de Cloudflare: cabecera cf-mitigated,
// cuerpo con "Just a moment..." o un status de error con esas marcas.
func isCloudflare(h http.Header, body []byte) bool {
	if strings.TrimSpace(h.Get("cf-mitigated")) != "" {
		return true
	}
	if bytes.Contains(body, []byte("Just a moment")) ||
		bytes.Contains(body, []byte("just a moment")) {
		return true
	}
	return false
}

// parseRetryAfter interpreta la cabecera Retry-After: segundos o fecha HTTP.
// Devuelve 0 si no se puede interpretar o la fecha ya pasó.
func parseRetryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}
