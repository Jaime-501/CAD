# Firmas reales — internal/httpx

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/httpx`).**

**Bloques de este archivo y tarea que los creó:**

- `### T02 — internal/httpx` → creado por **T02** (Infraestructura HTTP, errores y reintentos).

---

<!-- Ejemplo de formato:
### T02 — internal/httpx
- `type HTTPDoer interface { Do(*http.Request) (*http.Response, error) }`
- `func NewClient(opts ClientOptions) *http.Client`
- `func Get(ctx context.Context, d HTTPDoer, url string, opts ReqOptions) (*Response, error)`
-->

### T02 — internal/httpx

- `type HTTPDoer interface { Do(*http.Request) (*http.Response, error) }` — interfaz mínima de `Get`/`Post`; la cumple `*http.Client`.
- `const DefaultUserAgent string` — UA de navegador que aplica `Get`/`Post` si `ReqOptions.UserAgent` está vacío. `const DefaultTimeout = 15 * time.Second` (timeout por defecto del cliente).
- `type ClientOptions struct { Timeout time.Duration; Jar http.CookieJar; Transport http.RoundTripper; CheckRedirect func(*http.Request, []*http.Request) error }` — valor cero válido (timeout default, jar propio, transporte por defecto).
- `var NewTransport func() http.RoundTripper` — fábrica del `RoundTripper` por defecto de `NewClient`; es el hueco de sustitución (utls u otro transporte). Además `ClientOptions.Transport` lo cambia por cliente.
- `func NewClient(opts ClientOptions) *http.Client` — `http.Client` + `cookiejar` + transporte, listo para inyectar como `HTTPDoer`.
- `type ReqOptions struct { Provider, Operation, Extractor, UserAgent, Referer string; Header http.Header; Idempotent bool }` — `Idempotent` es la declaración con la que el llamador permite reintentar un POST.
- `func Get(ctx context.Context, d HTTPDoer, url string, opts ReqOptions) (*http.Response, error)` — devuelve la respuesta si `status < 400`; si no, `nil` + `*ScraperError` (cuerpo cerrado). Propaga `ctx`, aplica UA/Referer/cabeceras extra y registra cada petición con `logx.Request(provider, op, url, status, dur, extractor)`.
- `func Post(ctx context.Context, d HTTPDoer, url string, body []byte, opts ReqOptions) (*http.Response, error)` — ídem; el body es `[]byte` para que sea repetible en reintentos. Sin `Idempotent` ningún fallo del POST queda reintentable.
- `type ScraperError struct { Provider, Operation, URL string; StatusCode int; Retryable bool; Cause error }` (+ campo no exportado `retryAfter`):
  - `func (e *ScraperError) Error() string` — `httpx: <provider>/<op> <url redactada con logx.Redact>: status <n>: <causa>`; `func (e *ScraperError) Unwrap() error` → `Cause`; `func (e *ScraperError) RetryAfter() time.Duration` → espera del 429 `Retry-After` (segundos o fecha HTTP).
  - Causas sentinela (para `errors.Is`): `ErrCloudflareChallenge`, `ErrForbidden` (403), `ErrNotFound` (404), `ErrSessionExpired` (419), `ErrTooManyRequests` (429), `ErrServerError` (5xx), `ErrClientError` (resto 4xx), `ErrNilDoer`, `ErrNilResponse`.
  - Clasificación: 403 no reintentable salvo challenge de Cloudflare, 404 definitivo, 419 recuperable (refrescar sesión, T09), 429 reintentable con `Retry-After` como espera mínima, 5xx reintentable; transporte sin status reintentable si `ctx` sigue vivo y el método es idempotente.
  - Detección de challenge: cabecera `cf-mitigated`, cuerpo con `Just a moment` (case-insensitive), con cualquier status (incluso 200).
- `type Policy struct { MaxAttempts int; BaseDelay, MaxDelay time.Duration; Sleep func(context.Context, time.Duration) error }` — valor cero = 3 intentos, 250ms duplicándose hasta 5s; `Sleep` inyectable para tests.
- `func Retry(ctx context.Context, p Policy, fn func(context.Context) error) error` — máximo 3 intentos, backoff exponencial con jitter (mitad–total), espera mínima `Retry-After`, corta por `ctx` antes y durante cada espera; solo reintenta errores reintentables.
- `func Retryable(err error) bool` — `ScraperError.Retryable` manda; errores de red sueltos (`net.Error`) sí; contexto/cancelación y errores desconocidos, no.
- Tests: `internal/httpx/httpx_test.go` + `internal/httpx/retry_test.go` (26 tests, todos con `httptest.Server`, red de nada; `TestMain` instala un logger descartable para `logx.Request`).
