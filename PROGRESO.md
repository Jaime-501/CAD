# PROGRESO — Scraper JKAnime + reproductor HLS/MP4

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.

## Protocolo de sesión

1. Leer `AGENTS.md`, `tareas/00_REGLAS.md`, este archivo y el archivo de la tarea (`tareas/Txx.md`).
2. Ejecutar **solo** la tarea indicada. Respetar "Puedes tocar" / "NO debes tocar".
3. Correr la comprobación de la tarea.
4. Actualizar este archivo: marcar la tarea, anotar firmas reales, desviaciones y pendientes.
5. Detenerse. No empezar la siguiente tarea.

## Prompt base por sesión

```text
Lee AGENTS.md, tareas/00_REGLAS.md, PROGRESO.md y tareas/T07.md.
Ejecuta SOLO la tarea T07. Respeta "Puedes tocar" y "NO debes tocar".
Si necesitas tocar un archivo fuera de la lista, detente y repórtalo.
Al terminar: corre la comprobación, actualiza PROGRESO.md (firmas creadas,
desviaciones, pendientes) y detente. No empieces la siguiente tarea.
```

## Prerrequisitos: fixtures reales (humano o Playwright MCP)

Antes de las tareas que parsean HTML/JSON real debe existir un **fixture real y sanitizado** (sin cookies ni tokens) en el `testdata/` correspondiente. Un fixture es la respuesta **cruda** del servidor guardada en un archivo, para que los tests corran sin red. Lo puede capturar el humano a mano o el agente con el MCP de Playwright (ver "Captura con Playwright MCP" más abajo). Si no existe el fixture de la tarea, el agente se detiene y lo pide: **no inventa HTML ni selectores**.

- [ ] Página de búsqueda de JKAnime → T07
- [ ] Página de detalle de un anime → T08
- [ ] Respuesta JSON de `/ajax/episodes/<id>/<página>` (al menos 2 páginas) → T09
- [ ] Página de un episodio → T10
- [ ] Respuestas del player `jkplayer/um?...` (con y sin `.vtt`) → T12
- [ ] Respuesta de `jk.php` → T13
- [ ] Página con `var servers` → T14
- [ ] Página de MediaFire → T15

Nombres sugeridos de archivo (dentro del `testdata/` de cada paquete): `search.html`, `detail.html`, `episodes_p1.json`, `episodes_p2.json`, `episode_page.html`, `jkplayer_um.html` (y una variante con `.vtt`), `jkphp.html`, `servers.html`, `mediafire.html`.

### Captura con Playwright MCP

**Regla de oro:** el fixture debe ser lo que recibirá goquery, es decir, el HTML **tal como lo envía el servidor**, no el DOM ya modificado por JavaScript. goquery no ejecuta JS: un dato que solo aparece tras cargar scripts no existe para el scraper en Go.

Ojo con las herramientas de Playwright: `browser_snapshot` devuelve el árbol de accesibilidad, **no** HTML, y no sirve como fixture. Los nombres exactos de las herramientas dependen de la versión de tu servidor MCP; usa las equivalentes de navegar, evaluar JS y listar peticiones de red.

Procedimiento, en orden:

1. **Explorar:** navegar a la URL con Playwright y, con la lista de peticiones de red, identificar la petición exacta (URL, método, cabeceras y cuerpo necesarios, sin copiar valores secretos).
2. **Obtener la respuesta cruda**, por orden de preferencia:
   - Con `curl` o un programa Go corto desde Bash, guardando directo a archivo (`-o testdata/archivo`). Es lo preferible porque no pasa por el contexto y reproduce lo que hará el scraper.
   - Si el sitio bloquea clientes HTTP (Cloudflare) pero el navegador pasa: dentro de la página, `fetch(url).then(r => r.text())` mediante evaluación de JS. Devuelve la respuesta del servidor sin modificar por scripts.
3. **Comprobar que no es un bloqueo:** si el archivo contiene "Just a moment" o similar, no es un fixture válido. Anotarlo en "Problemas pendientes", porque afecta al diseño de T02.
4. **Guardar en archivo, no pegar en el chat.** Inspeccionar con `grep` o fragmentos de pocas líneas, nunca leyendo la página completa.
5. **Recortar solo si es enorme:** conservar el `<head>` con lo necesario (por ejemplo la meta del CSRF para T08) y el bloque relevante con 2 o 3 elementos de ejemplo. No alterar la estructura ni los nombres de clases.
6. **Sanitizar:** reemplazar cookies, tokens CSRF reales y cualquier dato de sesión por valores falsos como `TOKEN_DE_PRUEBA`.
7. **Verificar que el dato está en el archivo crudo** (`grep`). Si solo aparece en el DOM renderizado, goquery no puede extraerlo: detenerse, reportarlo y buscar el endpoint AJAX que entrega ese dato.
8. **Validar con goquery:** el test de la tarea carga el fixture y comprueba los selectores. Ese test es la prueba de que la captura sirve.

Para cuidar el contexto (modelos gratuitos):

- Activar el MCP de Playwright **solo** en las sesiones de T07–T10 y T12–T15, y desactivarlo en el resto.
- No pedir capturas ni lecturas de página completa en el chat; solo resultados acotados.
- Si hay un montón de peticiones de red, filtrar por la URL que interesa en lugar de listarlas todas.

### Comprobaciones después de cada tarea

Después de cada tarea, el humano ejecuta por su cuenta: `gofmt -l .`, `go vet ./...`, `go test -race ./...` (y `npm run check` en frontend). No confiar solo en lo que reporte el agente.

## Estado de tareas

Leyenda: `[ ]` pendiente · `[~]` en curso · `[x]` hecha y verificada

| Estado | Tarea | Título | Depende de | Sesión sugerida |
|---|---|---|---|---|
| [x] | T01 | Logging y observabilidad base | — | Sola |
| [x] | T02 | Infraestructura HTTP, errores y reintentos | T01 | Sola (pesada) |
| [x] | T03 | Modelos de dominio | — | Con T05 |
| [x] | T04 | Interfaz Provider y registro | T03 | Sola |
| [x] | T05 | Referencia de episodio y películas | T03 | Con T03 |
| [ ] | T06 | JKAnime — excepciones | T02 | Sola |
| [ ] | T07 | JKAnime — búsqueda | T02–T06 | Sola |
| [ ] | T08 | JKAnime — detalle y metadatos | T07 | Sola |
| [ ] | T09 | JKAnime — episodios AJAX con streaming | T08 | Sola (pesada) |
| [ ] | T10 | JKAnime — fetch de página de episodio | T02 | Sola |
| [ ] | T11 | Arquitectura de extractores | T03, T10 | Sola |
| [ ] | T12 | Extractor JKPlayer (softsubs) | T11 | Agrupable T12–T15 |
| [ ] | T13 | Extractor JKPHP | T11 | Agrupable T12–T15 |
| [ ] | T14 | Extractor `var servers` | T11 | Agrupable T12–T15 |
| [ ] | T15 | Resolver MediaFire | T11 | Agrupable T12–T15 |
| [ ] | T16 | Normalizador de VideoSource | T12–T15 | Con T17 |
| [ ] | T17 | Selector de mejor fuente + GetVideoSources | T16 | Con T16 |
| [ ] | T18 | AnimeFLV — búsqueda con fallback | T02, T04 | Sola |
| [ ] | T19 | Proxy local de streaming | T02, T03 | Sola (pesada) |
| [ ] | T20 | Integración app.go y eventos Wails | T09, T17, T19 | Sola (pesada) |
| [ ] | T21 | Caché local | T20 | Sola (pesada) |
| [ ] | T22 | Persistencia del historial | T03 | Sola (pesada) |
| [ ] | T23 | Regenerar bindings Wails | T20–T22 | Sola |
| [ ] | T24 | Estado Svelte | T23 | Sola |
| [ ] | T25 | UI de búsqueda y navegación | T23 | Sola |
| [ ] | T26 | Vista de detalle y episodios | T24, T25 | Sola |
| [ ] | T27 | Reproductor multi-recurso | T19, T24 | Sola (pesada) |
| [ ] | T28 | E2E (offline + real) | T17, T20 | Sola |
| [ ] | T29 | Ajustes finales y refactor | todas | Sola (modelo fuerte) |

Paralelizables (si usas varias sesiones a la vez): T06/T09/T10 · T12–T15 entre sí · T17/T18 · T21/T22 · T24/T25 · T26/T27.

## Firmas reales creadas

> Cada sesión añade aquí las firmas **tal como quedaron** (no como estaban en el plan). Las sesiones siguientes se basan en esto.

<!-- Ejemplo de formato:
### T02 — internal/httpx
- `type HTTPDoer interface { Do(*http.Request) (*http.Response, error) }`
- `func NewClient(opts ClientOptions) *http.Client`
- `func Get(ctx context.Context, d HTTPDoer, url string, opts ReqOptions) (*Response, error)`
-->

### T01 — internal/logx

- `func New(dev bool) *slog.Logger` — nivel `LevelDebug` si `dev` (texto en stderr), `LevelInfo` en producción (JSON en stderr); envuelve el handler con redacción y lo instala como default del paquete.
- `func Redact(s string) string` — enmascara con `***`: cabeceras `Cookie`/`Set-Cookie`/`Authorization`/`Proxy-Authorization` (resto de la línea), pares JSON `"cookie"|"authorization"|"csrf_token"|"token"|...`, parámetros `?token=`/`?csrf=`/`?_token=`/... en URLs y claves sueltas `csrf=`, `_token=`, `token=`, `apikey=`, `sid=`. Case-insensitive; no toca el resto de la cadena (p. ej. `other=1` o `"nombre":"naruto"`).
- `func NewHandler(h slog.Handler) slog.Handler` — `slog.Handler` envolvente que aplica `Redact` al mensaje y a cada atributo (message y atributos en `Handle`, `WithAttrs`, `WithGroup`).
- `func SetDefault(l *slog.Logger)` — fija el logger que usa `Request`.
- `func Request(provider, op, url string, status int, dur time.Duration, extractor string)` — log `http_request` con atributos `provider`, `op`, `url` (redactada por el handler), `status`, `dur`, `extractor`; nivel `Warn` si `status >= 400` o `0`, `Info` en caso contrario.
- Tests: `internal/logx/logx_test.go` (6 tests, red de nada).

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

### T03 — internal/models

- `type Anime struct { ID, Title, CoverURL, Provider string }` — sin cambios; tags `id`, `title`, `cover_url`, `provider`.
- `type EpisodeKind string` + `KindEpisode = "episode"`, `KindMovie = "movie"`, `KindSpecial = "special"`.
- `type AnimeDetail struct { ID, Title, Provider, Type, Status, Synopsis, Cover string; Genres []string; TotalEpisodes int }` — tags `id`, `title`, `provider`, `type`, `status`, `genres`, `synopsis`, `cover`, `total_episodes`.
- `type Episode struct { ID string; Number int; Title, Thumbnail string; Kind EpisodeKind }` — tags `id`, `number`, `title`, `thumbnail`, `kind`.
- `type ResourceType string` + `ResourceHLS = "hls"`, `ResourceMP4 = "mp4"`, `ResourceIframe = "iframe"`.
- `type Subtitle struct { URL, Lang string }` — tags `url`, `lang`.
- `type VideoSource struct { URL string; Type ResourceType; Quality, Server string; Headers map[string]string; Referer string; Subtitles []Subtitle }` — tags `url`, `type`, `quality`, `server`, `headers`, `referer`, `subtitles` (**`omitempty` solo en `subtitles`**).
- `type WatchProgress struct { AnimeID, EpisodeID string; CurrentTime, TotalDuration float64; IsWatched bool }` — tags `anime_id`, `episode_id`, `current_time`, `total_duration`, `is_watched`.
- `type EpisodeBatch struct { AnimeID string; Batch int; Episodes []Episode; Done bool; Err string }` — tags `anime_id`, `batch`, `episodes`, `done`, `err`.
- Tests: `internal/models/models_test.go` (8 tests con subtests, red de nada): ida y vuelta JSON de cada tipo, etiquetas exactas y omisión de `subtitles`.

### T05 — internal/models (ids.go)

- `var ErrInvalidEpisodeID = errors.New("id de episodio inválido")` — causa sentinela para `errors.Is`; los errores concretos se envuelven con `fmt.Errorf("%w: …", ErrInvalidEpisodeID)` y mensaje en español.
- `func NewEpisodeID(provider, slug string, kind EpisodeKind, number int) (string, error)` — `provider:slug:episode:<n>`, `provider:slug:special:<n>`, `provider:slug:movie` (para `KindMovie` el `number` **no** se serializa, tampoco si llega ≠ 0). Rechaza proveedor/slug vacío o con `:`, `number < 0` y `kind` desconocido.
- `func ParseEpisodeID(id string) (provider, slug string, kind EpisodeKind, number int, err error)` — inverso exacto de `NewEpisodeID`; 3 partes ⇒ `KindMovie` con `number = 0`; 4 partes ⇒ `KindEpisode`/`KindSpecial` con número **canónico** (`strconv.Itoa(n) == parte`: rechaza `01`, `+1`, `-1`, `abc`).
- Tests: `internal/models/ids_test.go` (5 tests de tabla, 37 subtests, red de nada): New válido, New inválido, Parse válido, Parse inválido y `Parse(New(x)) == x` (+ id reconstruido idéntico).

### T04 — internal/scraper

- `type Provider interface { Name() string; Search(ctx context.Context, query string) ([]models.Anime, error); GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error); GetVideoSources(ctx context.Context, episodeID string, opts SourceOpts) ([]models.VideoSource, error) }` — exactamente la firma del plan.
- `type EpisodeStreamer interface { StreamAnimeDetails(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) }` — interfaz opcional (T09/T20).
- `type SourceOpts struct { NoCache bool }` con tag `json:"no_cache"` — el reproductor pide una fuente nueva tras un error fatal.
- `var ErrNoDetails = errors.New("scraper: este proveedor no ofrece detalle de anime")` — causa sentinela para `errors.Is` (T18).
- `type Registry struct { … }` + `func NewRegistry() *Registry`, `func (r *Registry) Register(p Provider) error`, `func (r *Registry) Get(name string) (Provider, bool)`, `func (r *Registry) All() []Provider`. `sync.RWMutex`, valor cero utilizable (mapa perezoso), duplicado ⇒ reemplazo (gana el último), `All()` devuelve una copia ordenada por nombre.
- Tests: `internal/scraper/registry_test.go` (7 tests, red de nada, corren con `-race`): `fakeProvider` (falso con `Provider` + `EpisodeStreamer`), Get/Register, duplicados, registros inválidos, valor cero, orden y snapshot de `All()`, concurrencia (50 goroutines) y `ErrNoDetails`.

## Desviaciones del plan

> Cualquier cambio respecto a `tareas_v4.md` (nombres, decisiones, archivos extra tocados con permiso).

- T01: además de lo pedido en el entregable se exportan `NewHandler` y `SetDefault` (necesarios para envolver un handler propio y para que `Request` sea testeable); ambos dentro de `internal/logx/**`.
- T01: el entregado `logx.Request(...)` toma el logger del default del paquete (fijado por `New`), ya que la firma pedida no lleva logger.
- T01: salida del logger en `os.Stderr` (texto en dev, JSON en producción). No hay archivo de log todavía: si más adelante se quiere rotar a fichero, será decisión de otra tarea.
- Sin archivos tocados fuera de `internal/logx/**`.
- T02: `ScraperError` mantiene los campos exportados exactamente los pedidos y añade el campo **no exportado** `retryAfter` + método `RetryAfter()` para poder respetar el `Retry-After` del 429 (el plan no preveía dónde guardarlo).
- T02: además de `ErrCloudflareChallenge` se exportan las causas `ErrForbidden`/`ErrNotFound`/`ErrSessionExpired`/`ErrTooManyRequests`/`ErrServerError`/`ErrClientError` (+ `ErrNilDoer`/`ErrNilResponse`) para que las tareas siguientes clasifiquen con `errors.Is`.
- T02: la no-idempotencia del POST se aplica en `Post` (fuerza `Retryable=false` salvo `ReqOptions.Idempotent`); `Policy` no lleva bandera de idempotencia. Garantía pedida intacta: `Retry` solo reintenta lo que está marcado reintentable y un POST a ciegas nunca lo está.
- T02: `Post` recibe `body []byte` (repetible) en lugar de `io.Reader`.
- T02: en error (status ≥ 400 o fallo de transporte) `Get`/`Post` devuelven `nil` y cierran el cuerpo dentro; no se expone la respuesta de error.
- T02: `httpx` importa `logx` y registra cada petición con `logx.Request(...)` (regla `logx ← todos`); la redacción de la URL en el mensaje de error también pasa por `logx.Redact`.
- T02: detección de challenge por lectura por delante de 64 KiB (`sniffLimit`) con el cuerpo recompuesto (`replayBody`) para no perder bytes (test con cuerpo de 200 KB).
- Sin archivos tocados fuera de `internal/httpx/**`.
- T03: `AnimeDetail` no existía; se creó con los campos base `ID`/`Title`/`Provider` además de los pedidos (`Type`, `Status`, `Genres`, `Synopsis`, `Cover`, `TotalEpisodes`). **No** embebe `Anime`: así el detalle expone una sola portada (`cover`) y no conviven `cover_url` y `cover` en el mismo objeto.
- T03: los valores de `ResourceType` son minúsculas (`hls`, `mp4`, `iframe`), coherentes con `EpisodeKind` (`episode|movie|special`) y con los valores de `Type` que normaliza T08 (`series|movie|ova|special|unknown`). T16/T17 comparan contra las constantes `models.ResourceHLS/MP4/Iframe`; la frontend (T27) debe comparar contra esos strings.
- T03: `WatchProgress.CurrentTime`/`TotalDuration` son `float64` en segundos (valores con decimales del `<video>`); el cálculo de `IsWatched` lo hace T22.
- T03: `AnimeDetail.Type` y `Status` son `string` corriente (sin tipo nombrado) para no acotar los valores que T08 normaliza.
- T03: `omitempty` solo en `VideoSource.Subtitles`, como pide la comprobación; `headers` y `referer` llevan etiqueta plana (si el JSON con `null` molesta a la frontend, decidirlo en T23/T27).
- Sin archivos tocados fuera de `internal/models/models.go` y `internal/models/models_test.go`.
- T05: `NewEpisodeID` devuelve `(string, error)` (el plan no fijó la devolución): se validan proveedor/slug/tipo/número en origen en lugar de fabricar un id que `ParseEpisodeID` rechazaría después. **T09 debe manejar ese error** al aplicarlo a cada episodio.
- T05: `ParseEpisodeID` devuelve los cuatro campos + error (inverso exacto de `NewEpisodeID`) en lugar de una struct.
- T05: el número debe ser canónico (`1`, no `01`/`+1`) para que `Parse(New(x)) == x` sea una identidad exacta; `KindMovie` devuelve siempre `number = 0`.
- T05: el prefijo del id es el proveedor (`jkanime:…`), genérico y sin validar contra el `Registry` (T04); `Parse` acepta cualquier proveedor no vacío.
- Sin archivos tocados fuera de `internal/models/ids.go` e `internal/models/ids_test.go`.
- T04: el plan no fijó las devoluciones de `Registry`; quedan `Register(Provider) error` (error con provider `nil` o `Name()` vacío; duplicado ⇒ reemplazo, gana el último), `Get(name) (Provider, bool)` (idiomático, el llamador decide el mensaje) y `All() []Provider` (copia ordenada por nombre, así el orden es determinista en tests y UI).
- T04: `Registry` funciona con valor cero (`var r Registry`) — el mapa se crea perezosamente en `Register` — además de `NewRegistry()`.
- T04: `ErrNoDetails` es un error sentinela (se compara con `errors.Is`), como las causas de `httpx`; el mensaje va en español.
- T04: `SourceOpts` lleva tag `json:"no_cache"` por convención del proyecto aunque hoy no se expone a Wails (T20 recibe un `bool` suelto).
- T04: los asserts `var _ Provider`/`var _ EpisodeStreamer` están solo sobre el **fake de tests**; el de `JkAnime` queda para T07, como pide la tarea.
- T04: **`AnimeFLV` todavía no cumple `Provider`**: su `Search(query string)` no lleva `ctx` y `GetAnimeDetails(id)` devuelve `models.Anime` en lugar de `models.AnimeDetail`. `internal/scraper/animeflv.go` está fuera de "Puedes tocar" en T04, así que no se tocó; **T18 deberá adaptarlo**.
- Sin archivos tocados fuera de `internal/scraper/provider.go`, `internal/scraper/registry.go` y `internal/scraper/registry_test.go` (ni `animeflv.go`, ni su test con tag `integration`, ni `app.go`, ni `frontend/**`, ni `internal/models/**`).

## Problemas pendientes / deuda técnica

> Cosas que quedaron a medias, tests omitidos, TODOs que otra tarea debe resolver.

- `app.go` todavía no llama a `logx.New(...)` ni usa `logx.Request` (tocar `app.go` está fuera del alcance de T01); la integración queda para T20.
- `go test -race` y `gofmt` corrieron en el contenedor; el humano debe repetirlos por su cuenta según el protocolo.
- `internal/httpx` todavía no lo usa nadie: `internal/scraper/animeflv.go` sigue con su propio `http.Get`; migrarlo es trabajo de T18 (y de T06/T07/T10 con JKAnime).
- El transporte por defecto no añade cabeceras `Accept`/`Accept-Language`; si JKAnime las exige se añaden en su tarea vía `ReqOptions.Header`.
- `Retry-After` se respeta sin tope (un 429 con `Retry-After: 3600` esperaría una hora); lo corta el `ctx`. Si hay que limitarlo, decidirlo en la tarea que lo necesite.
- Un 419 queda reintentable a propósito: el reintento solo sirve si el `fn` de `Retry` refresca la sesión antes (detalle para T09).
- T03: los modelos aún no los consume nadie salvo los tests; los usa T08/T09/T19/T22/T20.
- T03: `VideoSource.Headers`/`Referer` serializan como `null`/`""` cuando están vacíos (sin `omitempty`).
- T05: `NewEpisodeID`/`ParseEpisodeID` todavía no los llama nadie fuera de sus tests; los usa T09 (con su manejo de error) y, por extensión, T20/T26.
- T04: nadie registra providers todavía: el `Registry` se monta en T20 (`jkanime` + `animeflv`).
- T04: `AnimeFLV` no cumple `Provider` (ver desviaciones); `app.go` sigue llamando a su `Search(query)` actual, sin `ctx`.
- T04: `SourceOpts.NoCache` no lo consume nadie todavía: lo leerán T17/T27.

## Bitácora

| Fecha | Tarea | Modelo usado | Resultado | Notas |
|---|---|---|---|---|
| 2026-10-04 | T01 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` y `go test -race ./...` en verde (`internal/logx` 6 tests PASS) |
| 2026-10-04 | T02 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (`internal/httpx` 26 tests PASS, `internal/logx` 6 PASS); solo se creó `internal/httpx/**` |
| 2026-10-04 | T03 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde (`internal/models` 8 tests PASS); solo se tocaron `internal/models/models.go` y `models_test.go` |
| 2026-10-04 | T05 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde (`internal/models` 5 tests/37 subtests PASS); solo se crearon `internal/models/ids.go` e `ids_test.go` |
| 2026-10-04 | T04 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (`internal/scraper` 7 tests PASS con `-race`); solo se crearon `internal/scraper/provider.go`, `registry.go` y `registry_test.go` |
