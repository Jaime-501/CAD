// Package httpx reúne la infraestructura HTTP del scraper: cliente
// reutilizable con timeout y cookies, peticiones Get/Post con User-Agent de
// navegador, errores tipados (ScraperError), detección de challenges de
// Cloudflare y una política de reintentos con backoff exponencial.
//
// Dirección de dependencias: httpx solo importa logx (logx ← todos). Nunca
// se registran cookies, tokens ni cabeceras Authorization: la URL que pasa
// por logx.Request se redacta con logx.Redact.
package httpx

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"miapp/internal/logx"
)

// DefaultUserAgent es el User-Agent de navegador que se envía cuando
// ReqOptions.UserAgent está vacío. Go usa "Go-http-client/1.1" por defecto y
// hay servidores que lo bloquean.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// DefaultTimeout es el tiempo máximo que NewClient espera por petición
// cuando ClientOptions.Timeout es cero.
const DefaultTimeout = 15 * time.Second

// sniffLimit es el número de bytes del cuerpo que se leen por delante para
// detectar marcas de challenge. El resto se devuelve diferido: no se pierde
// ningún byte del cuerpo (importante si el recurso es grande).
const sniffLimit = 64 << 10

// HTTPDoer es la interfaz mínima que necesitan Get y Post. *http.Client la
// cumple; en los tests se inyecta un doer falso o un cliente con transporte
// propio (inyección de dependencias, regla 4 de 00_REGLAS).
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientOptions configura NewClient. El valor cero es válido y aplica los
// valores por defecto del paquete.
type ClientOptions struct {
	// Timeout máximo de la petición completa (0 → DefaultTimeout).
	Timeout time.Duration
	// Jar de cookies (nil → jar propio, necesario para mantener la sesión
	// entre peticiones consecutivas).
	Jar http.CookieJar
	// Transport sustituye al RoundTripper por defecto: es el hueco para un
	// transporte con otra huella TLS (utls) o para tests.
	Transport http.RoundTripper
	// CheckRedirect, si se indica, reemplaza la política de redirecciones.
	CheckRedirect func(req *http.Request, via []*http.Request) error
}

// NewTransport es la fábrica del RoundTripper por defecto: NewClient la usa
// cuando ClientOptions.Transport es nil. Es el punto de sustitución para
// cambiar el transporte (p. ej. utls) sin tocar el resto del paquete;
// reemplázala antes de crear los clientes.
var NewTransport = defaultTransport

func defaultTransport() http.RoundTripper {
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	}
	clone := tr.Clone()
	clone.MaxIdleConnsPerHost = 16
	return clone
}

// NewClient crea un http.Client reutilizable con timeout, cookiejar y el
// RoundTripper de NewTransport (o el indicado en opts). Devuelve
// *http.Client porque cumple HTTPDoer.
func NewClient(opts ClientOptions) *http.Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	jar := opts.Jar
	if jar == nil {
		// cookiejar.New solo falla con Options personalizadas; con nil no.
		jar, _ = cookiejar.New(nil)
	}
	tr := opts.Transport
	if tr == nil {
		tr = NewTransport()
	}
	c := &http.Client{Timeout: timeout, Jar: jar, Transport: tr}
	if opts.CheckRedirect != nil {
		c.CheckRedirect = opts.CheckRedirect
	}
	return c
}

// ReqOptions personaliza una petición concreta (Get/Post).
type ReqOptions struct {
	// Provider, Operation y Extractor solo se usan para la línea de log
	// (logx.Request) y para el mensaje de error.
	Provider  string
	Operation string
	Extractor string
	// UserAgent vacío → DefaultUserAgent.
	UserAgent string
	// Referer, si se indica, se envía como cabecera Referer.
	Referer string
	// Header añade cabeceras adicionales (Content-Type, tokens de sesión...).
	Header http.Header
	// Idempotent declara que un POST puede reintentarse. Sin esta marca
	// ningún fallo de un POST se considera reintentable: los POST no se
	// reintentan a ciegas (regla del plan de T02).
	Idempotent bool
}

// Get realiza una petición GET y devuelve la respuesta si el status es
// utilizable (< 400). Cualquier otro resultado llega como *ScraperError y la
// respuesta se cierra dentro. Propaga ctx, aplica User-Agent y Referer
// opcional, y registra la petición con logx.Request.
func Get(ctx context.Context, d HTTPDoer, url string, opts ReqOptions) (*http.Response, error) {
	return do(ctx, d, http.MethodGet, url, nil, opts)
}

// Post realiza una petición POST con el body indicado (puede ser nil). Solo
// se marca como reintentable si opts.Idempotent es true.
func Post(ctx context.Context, d HTTPDoer, url string, body []byte, opts ReqOptions) (*http.Response, error) {
	return do(ctx, d, http.MethodPost, url, body, opts)
}

func do(ctx context.Context, d HTTPDoer, method, url string, body []byte, opts ReqOptions) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if d == nil {
		return nil, &ScraperError{
			Provider: opts.Provider, Operation: opts.Operation, URL: url,
			Cause: ErrNilDoer,
		}
	}
	// allowRetry: GET/HEAD son idempotentes por naturaleza; un POST solo si
	// el llamador lo declara.
	allowRetry := method != http.MethodPost || opts.Idempotent

	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, &ScraperError{
			Provider: opts.Provider, Operation: opts.Operation, URL: url,
			Cause: err,
		}
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
	if opts.Referer != "" {
		req.Header.Set("Referer", opts.Referer)
	}
	for k, vs := range opts.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	start := time.Now()
	resp, err := d.Do(req)
	dur := time.Since(start)
	if err == nil && resp == nil {
		logx.Request(opts.Provider, opts.Operation, url, 0, dur, opts.Extractor)
		return nil, &ScraperError{
			Provider: opts.Provider, Operation: opts.Operation, URL: url,
			Cause: ErrNilResponse,
		}
	}
	if err != nil {
		// Fallo de transporte: no hay status. No se reintenta si el contexto
		// ya no está vivo o si el POST no es idempotente.
		se := &ScraperError{
			Provider: opts.Provider, Operation: opts.Operation, URL: url,
			Cause: err,
		}
		se.Retryable = allowRetry && ctx.Err() == nil
		logx.Request(opts.Provider, opts.Operation, url, 0, dur, opts.Extractor)
		return nil, se
	}
	if resp.Body == nil {
		resp.Body = http.NoBody
	}

	// Lectura por delante para detectar challenges; el cuerpo se recompone
	// para no consumir ni un byte de más.
	prefix, readErr := io.ReadAll(io.LimitReader(resp.Body, sniffLimit))
	resp.Body = &replayBody{prefix: prefix, rest: resp.Body}
	if readErr != nil {
		_ = resp.Body.Close()
		logx.Request(opts.Provider, opts.Operation, url, resp.StatusCode, dur, opts.Extractor)
		se := &ScraperError{
			Provider: opts.Provider, Operation: opts.Operation, URL: url,
			StatusCode: resp.StatusCode,
			Cause:      readErr,
		}
		se.Retryable = allowRetry && ctx.Err() == nil
		return nil, se
	}

	if se := classify(opts, url, resp, prefix, allowRetry); se != nil {
		logx.Request(opts.Provider, opts.Operation, url, resp.StatusCode, dur, opts.Extractor)
		_ = resp.Body.Close()
		return nil, se
	}
	logx.Request(opts.Provider, opts.Operation, url, resp.StatusCode, dur, opts.Extractor)
	return resp, nil
}

// replayBody devuelve primero el trozo ya leído (prefix) y después el resto
// del cuerpo original, para que la lectura por delante no pierda datos.
type replayBody struct {
	prefix []byte
	rest   io.ReadCloser
}

func (b *replayBody) Read(p []byte) (int, error) {
	if len(b.prefix) > 0 {
		n := copy(p, b.prefix)
		b.prefix = b.prefix[n:]
		return n, nil
	}
	return b.rest.Read(p)
}

func (b *replayBody) Close() error {
	return b.rest.Close()
}
