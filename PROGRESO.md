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

- [x] Página de búsqueda de JKAnime → T07
- [x] Página de detalle de un anime → T08
- [x] Respuesta JSON de `/ajax/episodes/<id>/<página>` (al menos 2 páginas) → T09 — 3 páginas de bleach (`episodes_p1/p2/last.json`): 16 + 16 + **14** episodios (1–16, 17–32, 353–366); la última se sabe por el campo **`last_page`** de la respuesta (23; `total`=366 y `per_page`=16)
- [x] Página de un episodio → T10
- [x] Respuestas del player `jkplayer/um?...` → T12 — `jkplayer_um.html` (DPlayer, URL cifrada con `atob`, 16932 B) y `jkplayer_umv.html` (video.js, el player que la página carga por defecto, 3906 B). **El sitio no publica `.vtt`**: se muestrearon ~130 episodios (homepage, clásicos, películas, especiales y una serie en chino) en páginas de episodio, respuestas `um`/`umv`, playlists m3u8 y peticiones reales del navegador → 0 pistas; el test de subtítulos usa HTML artificial (autorizado en sesión: "una variante con .vtt si existe").
- [x] Respuesta de `jk.php` → T13 — `jkphp.html` (de `https://jkplayers.com/jk.php?u=…`, 200, 6582 B) y `jkplayer_jk.html` (de `jkanime.net/jkplayer/jk?u=…`, la referencia que hoy emiten las páginas, 6528 B). `jkanime.net/jk.php` responde 404 (nginx).
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
| [x] | T06 | JKAnime — excepciones | T02 | Sola |
| [x] | T07 | JKAnime — búsqueda | T02–T06 | Sola |
| [x] | T08 | JKAnime — detalle y metadatos | T07 | Sola |
| [x] | T09 | JKAnime — episodios AJAX con streaming | T08 | Sola (pesada) |
| [x] | T10 | JKAnime — fetch de página de episodio | T02 | Sola |
| [x] | T11 | Arquitectura de extractores | T03, T10 | Sola |
| [x] | T12 | Extractor JKPlayer (softsubs) | T11 | Agrupable T12–T15 |
| [x] | T13 | Extractor JKPHP | T11 | Agrupable T12–T15 |
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

### T06 — internal/scraper/jkanime

- `const DefaultExceptionsURL = "https://raw.githubusercontent.com/Zhuchii/ani-es/refs/heads/main/excepciones.json"` — mapa remoto real (JSON plano `string→string`).
- Internos: `const exceptionsTimeout = 5 * time.Second` (deadline de la descarga vía `context.WithTimeout` sobre el ctx del llamador), `const maxExceptionsBytes = 1 << 20` (1 MB; se lee `max+1` bytes y si se supera se descarta entera sin parsear).
- `type ResolvedAnimeURL struct { URL string; UsedException bool }` — tags `url`, `used_exception`.
- `type ExceptionResolver struct { URL string; Doer httpx.HTTPDoer; … }` — `URL` es el campo configurable (vacío → `DefaultExceptionsURL`); campos no exportados `mu sync.Mutex`, `loaded bool`, `entries map[string]string` (caché en memoria). Valor cero utilizable: con `Doer` nil no se toca la red (`httpx.Get` devuelve `ErrNilDoer`) y se sigue sin excepciones.
- `func (r *ExceptionResolver) Resolve(ctx context.Context, slug string) ResolvedAnimeURL` — nunca devuelve error. Coincidencia **exacta** con el mapa tras normalizar de igual forma claves y slug (`TrimSpace` + minúsculas + sin barra final); si hay entrada → valor del mapa con `UsedException=true`, si no → slug tal cual. Sin reglas de truncado.
- Internas: `load` (descarga bajo el mutex: un solo Resolve dispara la petición), `fetch` (httpx.Get con `Provider: "jkanime"`, `Operation: "exceptions"`; límite de tamaño; aviso con `slog.Warn` si falla), `parseExceptions` (validación de esquema), `normalizeExceptionKey`, `isHTTPURL`.
- Tests: `internal/scraper/jkanime/exceptions_test.go` (11 tests / 24 con subtests, red de nada, corren con `-race`; `TestMain` instala logger descartable para `logx.Request` y `slog`). Fixture real: `internal/scraper/jkanime/testdata/excepciones.json` (solo lectura).

### T07 — internal/scraper/jkanime

- `const DefaultBaseURL = "https://jkanime.net"` — la usa `New` con baseURL vacía y el valor cero de `JkAnime`. Interno `const providerName = "jkanime"` (valor de `Name()` y de `models.Anime.Provider`).
- `var ErrNotImplemented = errors.New("jkanime: operación no implementada todavía")` — causa sentinela (`errors.Is`) para los métodos de `Provider` sin implementar; la eliminan T08 (detalle), T09/T10 (episodios) y T17/T20 (fuentes). *(Desde T09 solo queda `GetVideoSources`: ver firmas de T09.)*
- `type JkAnime struct { BaseURL string; Doer httpx.HTTPDoer; Resolver *ExceptionResolver }` — campos exportados e inyectables; valor cero utilizable (`siteURL()` → `DefaultBaseURL`, `resolver()` → `ExceptionResolver` con el mismo doer, sin doer → `ErrNilDoer` sin red).
- `var _ scraper.Provider = (*JkAnime)(nil)` — assert en `jkanime.go`.
- `func New(doer httpx.HTTPDoer, baseURL string, resolver *ExceptionResolver) *JkAnime` — normaliza la base (sin espacios ni barra final; vacía → `DefaultBaseURL`) y, si el resolver viene nil, crea `&ExceptionResolver{Doer: doer}`.
- `func (j *JkAnime) Name() string` → `"jkanime"`.
- `func (j *JkAnime) Search(ctx context.Context, query string) ([]models.Anime, error)` — `GET <base>/buscar/<url.PathEscape(query)>` con `httpx.Get` (`Provider: "jkanime"`, `Operation: "search"`, UA por defecto); cuerpo limitado a `maxSearchBytes = 10 << 20`. Selectores reales del fixture: `.page_directorio .anime__item`, título+href en `h5 a`, portada en `.anime__item__pic[data-setbg]` (absolutizada). El href se absolutiza, pasa por `ExceptionResolver.Resolve` (el mapa se consulta con la **URL completa**) y de ahí sale el **slug** que hace de `models.Anime.ID` — nunca derivado del título. Devuelve slice **no nil** vacío si no hay resultados, y con query vacía no hace ninguna petición.
- `func (j *JkAnime) GetAnimeDetails(ctx, id)` → `models.AnimeDetail{}, nil, ErrNotImplemented`; `func (j *JkAnime) GetVideoSources(ctx, episodeID, opts)` → `nil, ErrNotImplemented`.
- Internos: `searchPath = "/buscar/"`, `maxSearchBytes`, `siteURL()`, `resolver()`, `parseSearch`, `absoluta`, `slugFromURL`.
- Tests: `search_test.go` (9 tests / 12 con subtests, red de nada, corren con `-race`; **reutiliza** los helpers de `exceptions_test.go` y no define otro `TestMain`) + `search_integration_test.go` (`//go:build integration`, smoke contra el sitio real). Fixtures reales (solo lectura): `testdata/search.html` (búsqueda "bleach": 10 resultados) y `testdata/search_empty.html` (búsqueda sin resultados: 0 items).

### T08 — internal/scraper/jkanime

- `const maxDetailBytes = 10 << 20` — límite de la página de detalle (10 MB); un cuerpo mayor se rechaza sin parsearlo (ídem `maxSearchBytes` de T07).
- `const tipoSerie = "series"`, `tipoPelicula = "movie"`, `tipoOVA = "ova"`, `tipoEspecial = "special"`, `tipoDesconocido = "unknown"` — valores normalizados de `models.AnimeDetail.Type` (sin exportar; la frontend compara contra esos strings).
- `var errSlugVacio = errors.New("jkanime: el id del anime no contiene slug")` — causa sentinela (`errors.Is`); un slug vacío/`"/"` no hace ninguna petición.
- `type detailMeta struct { NumericID, CSRFToken string }` — **estructura interna**: id numérico del sitio (atributo `data-anime`, p. ej. `278`) y meta `csrf-token`. No se exponen en `models` (regla 10: nunca en logs); los usa T09.
- `func (j *JkAnime) fetchDetail(ctx context.Context, slug string) (models.AnimeDetail, detailMeta, error)` — `GET <base>/<slug>/` con `httpx.Get` (`Provider: "jkanime"`, `Operation: "detail"`, UA por defecto), cuerpo limitado a `maxDetailBytes`; los fallos HTTP llegan como `*httpx.ScraperError` (404/403/challenge/5xx).
- Internos: `parseDetail`, `camposAnimeData`, `etiquetaDe`, `textoSinEtiqueta`, `generosDeCampo`, `enteroDeCampo`, `textoPrimero`, `portada`, `idNumerico`, `csrfToken`, `normalizarTipo`, `minusculasSinAcentos`, `acentosReemplazados`.
- Selectores reales (fixtures `detail.html` / `detail_movie.html`): título `.anime_info h3`, sinopsis `.anime_info p.scroll`, portada `.anime_pic img[src]` con fallback a `meta[property="og:image"]`, ficha `.anime_data li` indexada por el texto de su primer `<span>` (`tipo`, `generos`, `episodios`, `estado` — la ficha aparece en la versión pc y en la móvil, **gana la primera**), id numérico `#guardar-anime[data-anime]` (solo si son dígitos), CSRF `meta[name="csrf-token"]` leído con `EqualFold`.
- `normalizarTipo(raw string) string` — minúsculas y sin acentos; primero coincidencia exacta (`serie|series|tv…`, `pelicula|movie|film`, `ova`, `especial|special`) y después por contenido; vacío o no reconocido → `unknown`.
- Tolerancia: una página sin ficha, sin portada o sin CSRF **no** es error; los campos faltantes quedan en cero (`Type=unknown`, `TotalEpisodes=0`) y `Genres` sale siempre no-nil (JSON `[]`).
- Tests: `detail_test.go` (7 tests / 11 con subtests, red de nada, corren con `-race`; reutiliza los helpers de `exceptions_test.go` y `servidorHTML` de `search_test.go`, sin otro `TestMain`) + `detail_integration_test.go` (`//go:build integration`, tolerante a cambios de contenido: no fija episodios, géneros ni sinopsis). Fixtures reales: `testdata/detail.html` (serie "bleach") y `testdata/detail_movie.html` (película "Bleach: The DiamondDust Rebellion").

### T10 — internal/scraper/jkanime

- `type EpisodePage struct { URL string; HTML string; Headers http.Header }` — página cruda del episodio separada de la extracción: `URL` es la **final** (tras las redirecciones del sitio), `HTML` el cuerpo tal cual lo devolvió el servidor y `Headers` las cabeceras de esa respuesta **sin `Set-Cookie`**. Es tipo interno del paquete (no está en `models`).
- `func (j *JkAnime) fetchEpisodePage(ctx context.Context, episodeID string) (EpisodePage, error)` — parsea el id con `models.ParseEpisodeID`, exige proveedor `jkanime`, y hace `GET` con `httpx.Get` (`Provider: "jkanime"`, `Operation: "episode_page"`, `Referer` = página de la serie, UA por defecto); cuerpo limitado a `maxEpisodePageBytes = 10 << 20`. **No** extrae fuentes.
- Internos: `rutaEpisodio(slug, kind, number)`, `finalURL(resp, pedida)`, `cabecerasDePagina(h)`; const `rutaPelicula = "pelicula"`.
- Errores: los devuelve `httpx` sin tocar (`ErrNotFound` 404, `ErrForbidden` 403, `ErrTooManyRequests` 429 con `RetryAfter()`, `ErrServerError` 5xx y `ErrCloudflareChallenge` con cualquier status); un id inválido (malformado, tipo desconocido, número 0/`01` o proveedor ajeno) devuelve error del paquete **sin tocar la red**.
- Mapeo id → URL, **verificado en vivo con `curl.exe`**:
  - `jkanime:<slug>:episode:<n>` → `GET <base>/<slug>/<n>/` (200; fuera de rango → 404, p. ej. `/bleach/367/` y `/bleach/99999/`; sin barra final también responde 200, sin redirección).
  - `jkanime:<slug>:special:<n>` → `GET <base>/<slug>/<n>/` (el sitio responde 302 → `/<slug>/especial/`, que da 200; `/<slug>/2/` → 404).
  - `jkanime:<slug>:movie` → `GET <base>/<slug>/pelicula/` (200 en 3 películas distintas; `/<slug>/1/` → 302 → esa misma URL).
  - El `Referer` no es obligatorio en esta página (200 con referer de la serie, con referer ajeno y sin él); aun así se envía el de la serie.
- Tests: `episode_page_test.go` (6 tests / 24 con subtests, red de nada, corren con `-race`; reutiliza `fixtureDetalle` de T08 y `servidorHTML`/`doerGrabador`, sin otro `TestMain`): fixture 200 (ruta, `Referer`, UA y cabeceras, con `Set-Cookie` presente en la respuesta y ausente en `Headers`), tabla de rutas por id (episodio/especial/película), 404/403/429 (+`Retry-After: 7`)/500/challenge, ids inválidos sin red, redirección 302 y cuerpo > 10 MB. Fixture real: `testdata/episode_page.html` (episodio 1 de bleach, 61 KB).

### T09 — internal/scraper/jkanime

- `const maxEpisodesBytes = 10 << 20` — límite del JSON de una página de episodios (10 MB); ídem `maxSearchBytes`/`maxDetailBytes`.
- `const pausaPaginas = 300 * time.Millisecond` — pausa entre peticiones de la goroutine de fondo.
- `var _ scraper.EpisodeStreamer = (*JkAnime)(nil)` — assert en `episodes.go`.
- `func (j *JkAnime) StreamAnimeDetails(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error)` — devuelve de inmediato el detalle + **página 1** (ésta no se emite) y lanza una goroutine que recorre las páginas 2..N con `pausaPaginas` de separación; cada página se emite como `models.EpisodeBatch{AnimeID: slug, Batch: <número de página>, Episodes: …}` y el stream cierra con el lote `Done: true` (`Batch` = última página, `Err` vacío o el fallo que lo paró). Si `ctx` se cancela (espera o petición) la goroutine termina **sin emitir nada más** —ni el Done— y sin fugas. `onBatch` nil no emite nada.
- `func (j *JkAnime) GetAnimeDetails(ctx, id)` — definido en `jkanime.go` y delega en `getAnimeDetails` de `episodes.go`: el mismo flujo pero en la propia goroutine y sin lotes; devuelve el detalle y **todos** los episodios. Si algo falla a mitad devuelve el error sin detalle.
- Internos: `episodiosPagina`/`episodioJSON` (esquema de la respuesta), `sesionEpisodios` (detalle, CSRF vigente, `last_page`, ids vistos y acumulado), `iniciarStream` (detalle + página 1, exige `NumericID` y `CSRFToken`), `restantes` (bucle 2..N con pausa, lotes y Done), `post` (recuperación de 419), `emitir`, `convertir` (aplica `models.NewEpisodeID` y dedupe), `postEpisodios`, `tipoEpisodioDe`, `miniatura`, `esperar`.
- **Contrato del endpoint** (capturado con Playwright MCP y reproducido con `curl.exe`): `POST <base>/ajax/episodes/<id numérico>/<página>` con cuerpo `_token=<meta csrf-token>` (`application/x-www-form-urlencoded`), cabeceras `X-Requested-With: XMLHttpRequest` y `Referer: <base>/<slug>/`, cookies del jar. Obligatorios: método POST (GET → **405**), cookie de sesión y `_token` correcto — sin sesión o con token erróneo → **419** (`httpx.ErrSessionExpired`). Refrescando el detalle (GET con el mismo jar → cookie y token nuevos) y reintentando responde **200** (verificado en vivo). `X-Requested-With` no es imprescindible (200 también sin él) pero se envía, porque es lo que manda el `$.ajax` real.
- **Respuesta**: paginación de Laravel — `current_page`, `data[] {id, number, title, image, timestamp}`, `last_page`, `per_page`, `total`, `links`, `path`. **16 episodios por página**; bleach: `total`=366 y **`last_page`=23** (la última página se sabe por `last_page`; 22×16 = 352, luego la 23 trae 14: 353–366). Película: 1 entrada (`number: 1`), `last_page`=1.
- Miniatura: `image` es solo el nombre de archivo; se antepone el CDN derivado de la portada (`/animes/image/` → `/animes/video/image_thumb/`), el mismo criterio del sitio; sin imagen (o < 11 caracteres) → cadena vacía. URL verificada en vivo (200).
- Kind: `models.KindMovie` si el detalle es `movie`, si no `models.KindEpisode`; `KindSpecial` no lo genera T09 (el AJAX solo sirve episodios numerados).
- Tests: `episodes_test.go` (7 tests, red de nada, corren con `-race`; reutiliza `fixtureDetalle`, `clientePrueba` y `doerGrabador`, sin otro `TestMain`): lotes ordenados + Done final (+ contrato de la petición comprobado dentro del handler), 419 → refresco → éxito, 419 persistente → `Err`, cancelación con `ctx` sin fugas (contador de `runtime.NumGoroutine()`), `GetAnimeDetails` con las 3 páginas, entradas inválidas descartadas y película de una sola página. Smoke: `episodes_integration_test.go` (`//go:build integration`). Fixtures reales (solo lectura): `testdata/episodes_p1.json` (16, 1–16), `episodes_p2.json` (16, 17–32), `episodes_last.json` (14, 353–366, todas sin imagen) y `episodes_movie.json` (película, 1 entrada).

### T11 — internal/scraper/jkanime/extractors

- `type Page struct { URL string; HTML string; Headers http.Header }` — página cruda del episodio que procesan los extractores, **equivalente a `jkanime.EpisodePage`** (este paquete no puede importarlo por la regla de dependencias); T17 la construye desde esa página. Mismos campos y semántica: `URL` final tras redirecciones, `HTML` crudo, `Headers` sin `Set-Cookie`.
- `type VideoExtractor interface { Name() string; CanHandle(page Page) bool; Extract(ctx context.Context, page Page) ([]models.VideoSource, error) }` — `CanHandle` debe ser barato y sin red; `Extract` etiqueta cada fuente con `Server` (si llega vacío la rellena el Resolver con `Name()`).
- `type Base struct { Doer httpx.HTTPDoer }` — los extractores concretos lo **incrustan** y reciben su doer por campo/constructor (regla 4); así conservan cookies, UA y sesión. `Base{}` con doer nil ⇒ `httpx.Get` devuelve `ErrNilDoer` sin tocar la red.
- Const exportadas (contrato de nombres para **T12–T15**): `NameJKPlayer = "jkplayer"`, `NameJKPHP = "jkphp"`, `NameServers = "servers"`; `var ordenCanonico = []string{NameJKPlayer, NameJKPHP, NameServers}`.
- `var ErrSinFuentes = errors.New("extractors: ningún extractor aportó fuentes de vídeo válidas")` — causa sentinela (`errors.Is`) de `Resolve`.
- `type Resolver struct { extractores []VideoExtractor }` (campo no exportado; valor cero utilizable y sin fuentes).
- `func New(extractores ...VideoExtractor) *Resolver` — **copia** la lista, descarta nils y la reordena con `sort.SliceStable` por `prioridad()`: primero el orden canónico (JKPlayer → JKPHP → Servers) y después los nombres desconocidos en su orden de registro. No hay registro posterior: la cadena se fija aquí.
- `func (r *Resolver) Resolve(ctx context.Context, page Page) ([]models.VideoSource, error)` — ejecuta **todos** los extractores cuyo `CanHandle` acepta (no se detiene en el primero) y **agrega** sus fuentes; un extractor que devuelve error ⇒ `slog.Warn("extractors: extractor falló; se continúa con el siguiente", "extractor", …, "err", logx.Redact(...))` y se sigue con el siguiente. Descarta fuentes con URL vacía/en blanco; rellena `Server` con `Name()` si viene vacío (no pisa la etiqueta propia). Devuelve **error solo si el total de fuentes válidas es 0**: `ErrSinFuentes`, y si `ctx.Err() != nil` lo envuelve también (`errors.Is` contra las dos). `ctx == nil` ⇒ `context.Background()`.
- Interno: `prioridad(e VideoExtractor) int` — posición en el orden canónico (nombre con `ToLower`+`TrimSpace`), `len(ordenCanonico)` para los desconocidos.
- Tests: `extractor_test.go` (**10 tests / 16 con subtests**, red de nada salvo `httptest.Server` local, corren con `-race`; define su **propio `TestMain`** con logger descartable — el paquete `jkanime` es otro, no hay conflicto). HTML artificial (`htmlEpisodio` con marcas `data-src`, `JKPHP`, `var servers`), falsos `extractorFalso` (contadores de consultas/salidas y registro de orden), captura del `slog.Warn` con un handler propio y extractor con red contra `httptest` para probar `Base.Doer`.

### T12 — internal/scraper/jkanime/extractors

- `type JKPlayerExtractor struct { Base }` — instanciar como `JKPlayerExtractor{Base: Base{Doer: cli}}` (métodos con receptor de valor; cumple `VideoExtractor`).
- `func (JKPlayerExtractor) Name() string` → `NameJKPlayer` (`"jkplayer"`).
- `func (JKPlayerExtractor) CanHandle(page Page) bool` — `strings.Contains(page.HTML, "jkplayer/um")`: cubre `jkplayer/um?...` **y** `jkplayer/umv?...` (subcadena; umv es el player que la página carga por defecto).
- `func (e JKPlayerExtractor) Extract(ctx context.Context, page Page) ([]models.VideoSource, error)` — localiza cada iframe del player (`rePlayerJK`: URLs con esquema o raíz-relativas `/jkplayer/um…`, que es como las escribe el JS del sitio), los sigue con `httpx.Get` (`Provider: "jkanime"`, `Operation: "jkplayer"`, `Extractor: "jkplayer"`, `Referer: page.URL`) y devuelve las fuentes. Dedup: iframes por URL completa, fuentes por **recurso base** (url sin query: `um` y `umv` son el mismo vídeo con firmas distintas). Si algunos iframes fallan y otros no, devuelve las buenas; error solo con 0 fuentes (`errors.Join` de los fallos individuales).
- Cada `VideoSource`: `Server: "jkplayer"`, `Referer:` la URL del **propio iframe** (es la cabecera que exige el CDN del .m3u8 — captura con Playwright MCP), `Type` por extensión (`.m3u8`→`models.ResourceHLS`, `.mp4`→`MP4`) o por el contenido del player, `Subtitles` de la misma página del player, `Quality`/`Headers` vacíos (T16).
- Internos (los de parseo los reutiliza **T13** desde `jkphp.go`; `fuentesDePlayer` no, porque etiqueta con `NameJKPlayer` — véase "Desviaciones" de T13): `fuentesDePlayer` (descarga limitada a `maxPlayerBytes = 5 << 20`, lee `max+1`), `urlsDeVideo` (candidatas en orden: `atob('…')` de la config activa → `hls.loadSource('…')` → `<source src>` → `url:` del bloque `video: {...}`), `descifrarBase64`, `subtitulosDe`, `codigoDeIdioma`/`normalizaCodigo` (alias → código ISO; infiere del nombre del archivo o de `lang=`), `tipoDeRecurso`, `absolverURL` (normaliza `&amp;`, resuelve contra base, exige `http(s)` con host — descarta `javascript:`/`data:`), `recursoBase` (url sin query/fragmento).
- Regex: `rePlayerJK` (iframe en la página), `reURLAtob`, `reLoadSource`, `reEtiquetaSource`, `reURLDPlayer` (el `\bvideo\s*:\s*\{` evita atrapar el `vast({url: …}` comentado) y `reVTT` (tokens `.vtt` con query opcional).
- Tests: `jkplayer_test.go` (**12 tests / 19 con subtests**, `httptest.Server` local, red de nada, corren con `-race`; reutiliza el `TestMain` de T11): tabla de `CanHandle` (7 casos), extracción HLS de los dos fixtures reales (URL, tipo, etiqueta, Referer del iframe, `Referer`/`UA`/cookie de la petición con jar compartido), dedup um+umv con formas relativa y absoluta, subtítulos con idioma inferido, normalización `&amp;`, descarte de esquemas no-HTTP, 500 (`ErrServerError`), sin doer (`ErrNilDoer`), ctx cancelado, sin URL de iframe y prioridad de `atob` sobre la copia comentada. Fixtures reales: `testdata/jkplayer_um.html` y `testdata/jkplayer_umv.html`.

### T13 — internal/scraper/jkanime/extractors

- `type JKPHPExtractor struct { Base }` — instanciar como `JKPHPExtractor{Base: Base{Doer: cli}}` (métodos con receptor de valor; cumple `VideoExtractor`).
- `func (JKPHPExtractor) Name() string` → `NameJKPHP` (`"jkphp"`): es lo que registra al extractor en la cadena, porque `prioridad` lo coloca en el puesto canónico 1 (JKPlayer → JKPHP → Servers).
- `func (JKPHPExtractor) CanHandle(page Page) bool` — `strings.Contains(page.HTML, "jk.php") || strings.Contains(page.HTML, "jkplayer/jk?")`: cubre la forma clásica `jk.php?u=…` **y** la que hoy emiten las páginas, `jkplayer/jk?u=…`. `jkplayer/um` no lo atrapa (eso es JKPlayer).
- `func (e JKPHPExtractor) Extract(ctx context.Context, page Page) ([]models.VideoSource, error)` — localiza cada iframe con `reJkPHP` (con esquema, protocolo-relativo `//host/…` o raíz-relativo, en las dos formas de `jk.php`/`jkplayer/jk`), lo sigue con `httpx.Get` (`Provider: "jkanime"`, `Operation: "jkphp"`, `Extractor: "jkphp"`, `Referer: page.URL`) y devuelve las fuentes. Dedup: iframes por URL completa, fuentes por **recurso base**; si algunos iframes fallan y otros no, devuelve las buenas (error solo con 0 fuentes, `errors.Join`).
- Cada `VideoSource`: `Server: "jkphp"`, `Referer:` la URL del **propio iframe** (mismo criterio que T12), `Type` por extensión o por el contenido del player (`type: 'mp4'` → `models.ResourceMP4` en los dos fixtures), `Subtitles` del escaneo genérico `.vtt` de la misma página, `Quality`/`Headers` vacíos (T16).
- Interno: `func (e JKPHPExtractor) fuentesDeJKPHP(ctx context.Context, page Page, player *url.URL) ([]models.VideoSource, error)` — descarga limitada a `maxPlayerBytes = 5 << 20` (lee `max+1`) y traduce las candidatas con los helpers de parseo de `jkplayer.go`: `urlsDeVideo` (y con él `descifrarBase64`), `absolverURL`, `recursoBase`, `tipoDeRecurso`, `subtitulosDe`.
- Regex: `reJkPHP` = ``(?:https?:)?//[^\s"'<>\\]*(?:jk\.php|jkplayer/jk\?)[^\s"'<>\\]*|/[^\s"'<>\\]*(?:jk\.php|jkplayer/jk\?)[^\s"'<>\\]*`` (mismo molde que `rePlayerJK`).
- `extractor.go` **no necesitó cambios**, igual que en T12: registrar el extractor es devolver `NameJKPHP`, que ya estaba en `ordenCanonico` desde T11.
- Tests: `jkphp_test.go` (**11 tests / 18 con subtests**, `httptest.Server` local, red de nada, corren con `-race`; reutiliza el `TestMain` de T11 y los helpers de `jkplayer_test.go`: `cargarFixture`, `nuevoServidorPlayer`/`servidorPlayer`, `paginaConIframes`, `rutaUM`/`urlUM`): tabla de `CanHandle` (7 casos), extracción de los **dos fixtures reales** (URL, `Type=mp4`, etiqueta, Referer del iframe, `Referer`/`UA`/cookie de la petición con jar compartido y una sola ruta pedida), dedup de la misma referencia relativa+absoluta, subtítulos con HTML artificial, 500 (`ErrServerError`), sin doer (`ErrNilDoer`), ctx cancelado, sin URL de iframe, respuesta sin vídeo y cadena canónica (`New` con JKPHP + JKPlayer reales y un `servers` falso → orden jkplayer → jkphp → servers). Fixtures reales: `testdata/jkphp.html` y `testdata/jkplayer_jk.html`.

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
- T05: `NewEpisodeID` devuelve `(string, error)` (el plan no fijó la devolución): se validan proveedor/slug/tipo/número en origen en lugar de fabricar un id que `ParseEpisodeID` rechazaría después. **T09 debe manejar ese error** al aplicarlo a cada episodio. *(Hecho en T09: las entradas con error se descartan.)*
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
- T06: la descarga del mapa ocurre **bajo el mutex** del resolver (`load`): varios `Resolve` concurrentes solo disparan una petición y, mientras dura (máx. `exceptionsTimeout`), los demás esperan.
- T06: el mapa descargado se cachea durante toda la ejecución; **un fallo no se cachea**, el siguiente `Resolve` reintenta la descarga (auto-recuperación tras un corte transitorio).
- T06: normalización idéntica para claves del mapa y slug de entrada (`TrimSpace` + minúsculas + sin barra final) con coincidencia **exacta**; el valor se devuelve tal cual del JSON. No se deduce ninguna regla de truncado del slug.
- T06: validación de esquema: se descartan las entradas cuya clave quede vacía o cuya clave/valor no sea URL absoluta http/https con host; un cuerpo que no sea un objeto JSON `string→string` (incluido `null`) se trata como JSON inválido y no se cachea.
- T06: `Resolve` no devuelve error (firma de la tarea): el fallo se expresa como `UsedException=false` + slug tal cual, con aviso `slog.Warn` (logx no exporta su logger; el error HTTP ya queda registrado por `logx.Request` dentro de `httpx.Get`).
- T06: `ResolvedAnimeURL` lleva tags `json` snake_case por convención del proyecto, aunque hoy no cruza a Wails (T20/T23 decidirán si se expone).
- Sin archivos tocados fuera de `internal/scraper/jkanime/exceptions.go` e `internal/scraper/jkanime/exceptions_test.go` (el fixture `testdata/excepciones.json` solo se leyó).
- T07: la ruta se verificó en vivo: `GET /buscar/<query>` responde 200 y es la ruta canónica — el formulario del sitio manda `GET /buscar?q=…` y el servidor **redirige** a `/buscar/<query>` (comprobado con `bleach` y con `naruto shippuden`). `Search` usa la ruta con `url.PathEscape(query)` (escape de segmento: espacio → `%20`, `/` → `%2F`, `;`/`?` → `%3B`/`%3F`), como pedía la tarea.
- T07: fixtures capturados con `curl.exe` directo a `testdata/` (HTTP 200, **sin challenge de Cloudflare**; no hizo falta Playwright): `search.html` (35 KB, 10 resultados de "bleach") y, como extra para el caso vacío, `search_empty.html` (23 KB, 0 items). Sin recortar (no eran enormes); solo se sanitizó el `<meta name="csrf-token">` a `TOKEN_DE_PRUEBA`.
- T07: el smoke de red vive en el archivo **nuevo** `search_integration_test.go` con `//go:build integration`, porque las etiquetas de build son por archivo y `search_test.go` debe correr en la puerta de calidad sin red. Es el único archivo creado fuera de la lista "Puedes tocar" (mismo paquete, no se modificó nada existente fuera de ella).
- T07: `Search` devuelve un slice **no nil** (JSON `[]`, no `null`) tanto en vacío como en sin resultados, pensando en la UI; con query vacía o en blanco no hace petición.
- T07: `Anime.ID` queda como **slug del href ya pasado por el resolver** (p. ej. `bleach`): T08 reconstruye `<base>/<slug>/` para el detalle; volver a resolverla no tiene efecto (el mapa se consulta por la URL completa y la clave rota ya no aparece).
- Sin archivos tocados fuera de `internal/scraper/jkanime/jkanime.go`, `search.go`, `search_test.go`, `search_integration_test.go` y `testdata/` (ni `exceptions.go`, ni `app.go`, ni `frontend/**`).
- T08: fixtures capturados con `curl.exe` directo a `testdata/` (HTTP 200, **sin challenge de Cloudflare**): `detail.html` (71 KB, serie "bleach") y `detail_movie.html` (68 KB, película "Bleach: The DiamondDust Rebellion", creada además para probar la normalización de tipo). Sin recortar; solo se sanitizó el `<meta name="csrf-token">` a `TOKEN_DE_PRUEBA` (el resto no traía cookies ni datos de sesión).
- T08: verificado con grep sobre el HTML crudo de ambos fixtures: id numérico `data-anime="278"`/`"504"`, CSRF en `<meta name="csrf-token">`, total de episodios `<li><span>Episodios:</span> 366</li>`/`1` y géneros en `<li><span>Generos:</span> <a …>…</a></li>`.
- T08: el tipo sí existe en el HTML (`<li rel="tipo"><span>Tipo:</span> Serie|Pelicula</li>`); `normalizarTipo` traduce a `series|movie|ova|special|unknown` y tolera acentos y variantes.
- T08: **`GetAnimeDetails` sigue devolviendo `ErrNotImplemented`**: su definición está en `jkanime.go` y `search_test.go::TestMetodosNoImplementados` exige ese error; ninguno de los dos figura en "Puedes tocar" de T08, así que **no se tocó**. Cablear `fetchDetail` al `Provider` exige modificar ambos → reportado como pendiente (ver "Problemas pendientes").
- T08: `errSlugVacio` es un error sentinela propio del paquete (el plan no fijaba qué devolver con slug vacío; decisión: no petición).
- T08: se creó además `detail_integration_test.go` con `//go:build integration` (permitido por `AGENTS.md`; `detail_test.go` debe seguir corriendo sin red en la puerta de calidad).
- Sin archivos tocados fuera de `internal/scraper/jkanime/detail.go`, `detail_test.go`, `detail_integration_test.go` y `testdata/` (ni `jkanime.go`, ni `search.go`/`search_test.go`, ni `exceptions.go`, ni `app.go`, ni `frontend/**`).
- T10: fixture capturado con `curl.exe` directo a `testdata/` (HTTP 200, **sin challenge de Cloudflare**): `episode_page.html` (61 KB, episodio 1 de bleach). Sin recortar; solo se sanitizó el `<meta name="csrf-token">` a `TOKEN_DE_PRUEBA`. Los tokens `t=…` de los iframes `jkplayer` y los `remote` de `var servers` **no** se tocaron: no son datos de sesión (la página se pidió sin cookies) y T12–T14 necesitan su forma real para parsearlos; las cookies de sesión (`jkanime-session`, `XSRF-TOKEN`) solo aparecen en las cabeceras `Set-Cookie`, nunca en el cuerpo.
- T10: `EpisodePage.Headers` copia las cabeceras de la respuesta final **menos `Set-Cookie`**: la sesión vive en el cookiejar del doer inyectado y no debe viajar en structs ni llegar a logs (regla 10).
- T10: `Referer` = `<base>/<slug>/` (la página de la serie, de donde navega el usuario). En vivo el sitio devuelve 200 con ese referer, con uno ajeno y sin él; se envía el de la serie por coherencia con "conservar Referer".
- T10: además del mapeo `episode:<n>` que pedía la tarea se resolvieron `movie` → `/<slug>/pelicula/` y `special:<n>` → `/<slug>/<n>/` (con el 302 del sitio hacia `/especial/`), porque `models.ParseEpisodeID` devuelve los tres kinds y T09/T17 pasarán ids de película; las tres rutas están verificadas en vivo.
- T10: la URL que devuelve `EpisodePage.URL` es la **final** tras seguir redirecciones (`resp.Request.URL`), no la pedida.
- T10: **no** se creó `episode_page_integration_test.go`: T10 no pide smoke de red (su comprobación es fixture + `httptest`); la verificación en vivo de las rutas se hizo con `curl.exe` y queda arriba.
- Sin archivos tocados fuera de `internal/scraper/jkanime/episode_page.go`, `episode_page_test.go` y `testdata/` (ni `jkanime.go`, ni `detail.go`/`detail_test.go`, ni `app.go`, ni `frontend/**`).
- T09: fixtures capturados con **Playwright MCP** (identificar la petición real: `POST /ajax/episodes/278/1`, body `_token=…`) y descargados con `curl.exe` (cookie jar + token de la página de detalle) directo a `testdata/`: `episodes_p1.json` (200, 3916 B), `episodes_p2.json` (200, 4026 B) y `episodes_last.json` (página 23, 200, 3119 B). **Sin sanitizar**: la respuesta no lleva cookies, tokens ni datos de sesión (verificado con grep). Fixture extra `episodes_movie.json` (película 504, `last_page` 1, 1 entrada, 712 B) para probar el camino de película.
- T09: verificado en vivo con `curl.exe` que **sin cookie de sesión, con `_token` incorrecto o con POST vacío la respuesta es 419** (y que `GET` da **405**), y que refrescar el detalle —GET con el mismo jar, que entrega cookie y token nuevos— y reintentar el POST devuelve **200**: ese es el flujo de `sesionEpisodios.post` y lo que reproducen `TestStream419RefrescaLaSesion` (el servidor exige que el `_token` enviado coincida con el **último CSRF entregado**) y `TestStream419PersistenteEmiteError` con `httptest`, nunca con la red.
- T09: la **página 1 no se emite como lote**: se devuelve en el retorno de `StreamAnimeDetails`; los lotes emitidos son las páginas 2..N con `Batch` = número de página, y el lote final es `Done: true` con `Batch` = última página. `EpisodeBatch.Err` solo aparece en el lote final.
- T09: `Provider.GetAnimeDetails` (`jkanime.go`) ya delega en `getAnimeDetails` (`episodes.go`) y `search_test.go::TestMetodosNoImplementados` dejó de exigir `ErrNotImplemented` en ese método: los dos archivos que el prompt autorizaba además de `episodes.go`/`episodes_test.go`/`testdata/`. `ErrNotImplemented` sigue vivo para `GetVideoSources` (T17/T20) y su comentario en `jkanime.go` se actualizó.
- T09: además se creó `episodes_integration_test.go` con `//go:build integration` (permitido por `AGENTS.md`, mismo patrón que T07/T08): smoke real que hace stream de bleach y cancela tras 2 lotes.
- T09: `pausaPaginas` (300 ms) se aplica también en el camino de `GetAnimeDetails` (un solo código para los dos): 366 episodios tardan ~7 s en completarse. Si la frontend lo nota, mover la pausa a un campo inyectable es trabajo de la tarea que lo necesite.
- T09: el kind sale del tipo del anime (`movie` → `KindMovie`, resto → `KindEpisode`). Si una película tuviera varias entradas en el JSON compartirían el mismo id `…:movie` y el dedupe las colapsaría en una (el sitio las enlaza todas a `/slug/pelicula/`).
- T09: `models.Episode.Number` es el número crudo del JSON (una película sale con `Number: 1` aunque su id no lo lleve).
- T09: las entradas cuyo id no se puede construir con `NewEpisodeID` (p. ej. `number: -1`) se **descartan** sin tumbar el flujo, y los ids ya vistos dentro del mismo stream no se emiten dos veces (dedupe por id generado).
- T09: si el detalle no trae `NumericID` o `CSRFToken`, `iniciarStream` devuelve error sin hacer ninguna petición AJAX (sin esos datos no hay endpoint al que llamar); un error en la página 1 tampoco emite lotes.
- T09: `restantes` respeta el `last_page` de la **página 1** como tope del bucle (aunque una página posterior trajera otro valor).
- Sin archivos tocados fuera de `internal/scraper/jkanime/episodes.go`, `episodes_test.go`, `episodes_integration_test.go`, `testdata/`, `jkanime.go` (solo la delegación de `GetAnimeDetails` y el comentario de `ErrNotImplemented`) y `search_test.go` (solo `TestMetodosNoImplementados`) (ni `detail.go`, ni `search.go`, ni `exceptions.go`, ni `app.go`, ni `frontend/**`).
- T11: se creó el paquete nuevo `internal/scraper/jkanime/extractors/` con los dos archivos de "Puedes tocar" (`extractor.go`, `extractor_test.go`); no se tocó nada fuera de esa lista (ni `frontend/**`).
- T11: la página llega como `extractors.Page`, **no** como `jkanime.EpisodePage`: `extractors` no puede importar `jkanime` (regla de dependencias) y el acoplamiento inverso está prohibido. **T17 es quien convierte** `EpisodePage → Page`.
- T11: el doer se inyecta con `Base{Doer httpx.HTTPDoer}` **incrustado** en cada extractor concreto (campo/constructor del extractor, regla 4) en lugar de un parámetro en la firma de la interfaz, que solo puede tener `Name`/`CanHandle`/`Extract`. Así T12–T15 hacen `type X struct{ Base }` y comparten cookies/sesión con el resto del flujo.
- T11: la cadena se fija en `New` (sin `Register` posterior): `sort.SliceStable` por prioridad canónica mantiene el orden de registro de los nombres desconocidos y hace determinista el orden en los tests.
- T11: `ErrSinFuentes` es sentinela propia del paquete (el plan pedía "error solo si el total de fuentes válidas es 0" sin decir cuál); con contexto cancelado el error la envuelve a las dos (`errors.Is` contra `ErrSinFuentes` y `context.Canceled`).
- T11: "fuente válida" = `TrimSpace(URL) != ""`; **no** se valida el esquema ni se normaliza nada (eso es T16) y no se descarta por tipo/calidad.
- T11: el aviso por extractor fallido usa `slog.Warn` con `logx.Redact(err.Error())` (mismo estilo que `exceptions.go`), no `logx.Request` (no hay petición propia en el Resolver).
- T11: tests sin fixtures reales ni red, como pide la comprobación: HTML artificial + falsos extractores + un `httptest.Server` local para el doer inyectado. El paquete `extractors` define su **propio `TestMain`** (el de `jkanime` es otro paquete; la regla "un solo TestMain por paquete" se mantiene).
- Sin archivos tocados fuera de `internal/scraper/jkanime/extractors/extractor.go` y `extractor_test.go` (ni `episode_page.go`, ni `jkanime.go`, ni `app.go`, ni `frontend/**`).
- T12: **prerrequisito "fixture con `.vtt`" incumplible**: el sitio actual no publica softsubs en ninguna parte (muestreo de ~130 episodios: páginas de episodio, respuestas `um`/`umv`, playlists m3u8 y peticiones reales de Playwright → 0 `.vtt`; de paso se comprobó que `cdn.jkanime.net/assets/dplayer.js` está caído con 522). Como el humano autorizó "una variante con `.vtt` si existe", el extractor hace un escaneo **genérico** de tokens `.vtt` (sin selectores propios del sitio) y el test de subtítulos usa un HTML artificial de player con `<track … src="/subs/espanol.vtt">`. Si el sitio publica pistas en el futuro, se recogen sin tocar el código (habría que capturar entonces el fixture real y sustituir el artificial).
- T12: `CanHandle` reconoce también `jkplayer/umv?...` aunque la tarea solo nombraba `um?`: umv es el player que la página carga por defecto (`video[1]`) y es el segundo fixture capturado (`jkplayer_umv.html`). Mantener la subcadena `jkplayer/um` cubre las dos variantes con una sola comprobación barata.
- T12: además del `jkplayer_um.html` pedido se capturó `jkplayer_umv.html` (y de paso `jkplayer_jk.html`, la respuesta real de `jkplayer/jk?u=…`, que es el fixture de T13): todos en `extractors/testdata/`.
- T12: sanitización de los fixtures del player: parámetros firmados de la URL de vídeo (`st=JmivOww_…`/`st=I5ctfgSi…` → `ST_UM`/`ST_UMV`, `e=1791251737`/`1791251787` → `1700000000`) y la base64 interna de `atob()` se **recalculó** para que decodifique la URL saneada (verificado con `base64 -d`). Los `t=`/`op=` del iframe en los tests son ficticios. En los fixtures de T13 se saneó también el path de acceso `stream/jkmedia/<hash>/<hash>/1/<ip>` → `aaa…/bbb…/1/2001:db8::1`. Cuerpos sin cookies ni tokens (grep 0).
- T12: las fuentes se **deduplican por recurso base** (url sin query) dentro del extractor: `um` y `umv` del mismo episodio devuelven el mismo `.m3u8` con firmas distintas y no debe salir duplicado. Es una decisión del extractor; T16 puede re-normalizar por su cuenta después.
- T12: los helpers de parseo comunes (`absolverURL`, `recursoBase`, `urlsDeVideo`, `tipoDeRecurso`, `subtitulosDe`, `descifrarBase64`, `maxPlayerBytes`, `codigoDeIdioma`) se definieron en `jkplayer.go` y **T13 los reutiliza** desde `jkphp.go` (mismo paquete). No se podían poner en `extractor.go`: su edición queda reservada a registrar la cadena.
- T12: `extractor.go` **no necesitó cambios**: `NameJKPlayer`, `NameJKPHP`, `NameServers` y el orden canónico ya existían desde T11 y nadie construye la cadena todavía (T17).
- T12: `VideoSource.Referer` = URL del iframe del player (decisión basada en la captura de Playwright: el `.m3u8` de `nika.playmudos.com` responde 200 con ese referer). **T19 inyecta el Referer en el proxy local**: coordinar allí para no duplicar la cabecera.
- T12: sin smoke de red (`jkplayer_integration_test.go`): la tarea no lo pide y la verificación en vivo se hizo al capturar los fixtures (Playwright MCP + `curl.exe`).
- Sin archivos tocados fuera de `internal/scraper/jkanime/extractors/jkplayer.go`, `jkplayer_test.go` y `testdata/` (ni `extractor.go`, ni nada del resto del paquete, ni `frontend/**`).
- T13: `extractor.go` **no se tocó**, igual que en T12: registrar el extractor en la cadena es devolver `NameJKPHP`, que ya estaba en `ordenCanonico` desde T11, y nadie construye la cadena todavía (T17). No hizo falta ninguna edición distinta de la de T12.
- T13: `CanHandle` reconoce también `jkplayer/jk?u=…` —la referencia que hoy emiten las páginas (prerrequisito de fixtures de T13 en este archivo)— además del `jk.php?u=…` que nombra la tarea; sin esa variante el fixture `jkplayer_jk.html` no tendría consumo.
- T13: `jkplayer.go` **no se tocó** (fuera de "Puedes tocar"): `JKPlayerExtractor.fuentesDePlayer` no es reutilizable porque etiqueta con `NameJKPlayer` y registra `Extractor: "jkplayer"` en `logx.Request`. `jkphp.go` define el gemelo `fuentesDeJKPHP` (misma descarga limitada a `maxPlayerBytes`) y **reutiliza los helpers de parseo** de `jkplayer.go`. La duplicación de la descarga es candidata a unificar en T29.
- T13: `VideoSource.Referer` = URL del **propio iframe** (mismo criterio que T12). Ojo: la respuesta de jk.php declara `<meta name="referrer" content="never"/>`, así que el navegador no enviaría referer al `stream/jkmedia/...`; si T16/T19 prefieren otro criterio (o vacío), decidirlo allí.
- T13: sin smoke de red (`jkphp_integration_test.go`): la tarea no lo pide (comprobación = fixture + `httptest.Server`) y los fixtures ya estaban capturados.
- Sin archivos tocados fuera de `internal/scraper/jkanime/extractors/jkphp.go` y `jkphp_test.go` (ni `jkplayer.go`, ni `extractor.go`, ni `extractor_test.go`, ni `testdata/` —solo se leyó—, ni `frontend/**`).

## Problemas pendientes / deuda técnica

> Cosas que quedaron a medias, tests omitidos, TODOs que otra tarea debe resolver.

- `app.go` todavía no llama a `logx.New(...)` ni usa `logx.Request` (tocar `app.go` está fuera del alcance de T01); la integración queda para T20.
- `go test -race` y `gofmt` corrieron en el contenedor; el humano debe repetirlos por su cuenta según el protocolo.
- `internal/httpx` ya lo consume `internal/scraper/jkanime/exceptions.go` (T06); `internal/scraper/animeflv.go` sigue con su propio `http.Get`, migrarlo es trabajo de T18 (y de T07/T10 con JKAnime).
- El transporte por defecto no añade cabeceras `Accept`/`Accept-Language`; si JKAnime las exige se añaden en su tarea vía `ReqOptions.Header`.
- `Retry-After` se respeta sin tope (un 429 con `Retry-After: 3600` esperaría una hora); lo corta el `ctx`. Si hay que limitarlo, decidirlo en la tarea que lo necesite.
- Un 419 queda reintentable a propósito: `httpx.Retry` solo sirve si el `fn` de `Retry` refresca la sesión antes. T09 ya refresca él mismo en `sesionEpisodios.post` (una sola vez y sin usar `Retry`); cualquier otro 419 sigue igual.
- T03: los modelos ya los usan T08 (detalle) y T09 (episodios/lotes); siguen sin consumidor T19, T22 y T20 (serialización a Wails).
- T03: `VideoSource.Headers`/`Referer` serializan como `null`/`""` cuando están vacíos (sin `omitempty`).
- T05: `NewEpisodeID`/`ParseEpisodeID` **ya los usa T09** (`NewEpisodeID` en `convertir`, con su error, descartando las entradas inválidas; `ParseEpisodeID` sigue solo en sus tests); T20/T26 los usarán por su lado.
- T04: nadie registra providers todavía: el `Registry` se monta en T20 (`jkanime` + `animeflv`).
- T04: `AnimeFLV` no cumple `Provider` (ver desviaciones); `app.go` sigue llamando a su `Search(query)` actual, sin `ctx`.
- T04: `SourceOpts.NoCache` no lo consume nadie todavía: lo leerán T17/T27.
- T06: `ExceptionResolver` todavía no lo usa nadie fuera de sus tests; lo monta T07 (y quizá T20, si precarga el mapa al arrancar).
- T06: un fallo de descarga se reintenta en cada `Resolve` (sin negativo-cache): durante un corte remoto cada llamada puede tardar hasta `exceptionsTimeout`, serializadas por el mutex. Si molesta, añadir negativo-cache con TTL en la tarea que lo necesite.
- T06: el aviso `slog.Warn` va al logger por defecto de `slog`, no al de `logx`; para unificar formato y redacción, T20/T29 puede instalar `slog.SetDefault(logx.New(dev))`.
- T06: el paquete `jkanime` ya define `TestMain` (en `exceptions_test.go`): **T07+ no debe definir otro** en el mismo paquete.
- T07: `GetVideoSources` devuelve `ErrNotImplemented` a propósito (y `GetAnimeDetails`, hasta que T09 lo implementó); T17/T20 lo implementan y borran esa rama.
- T07: contrato del id para T08/T09/T22 — `Anime.ID` es el slug sin barra ni dominio; la URL del detalle es `<base>/<slug>/`.
- T07: con el resolver por defecto (`New(..., nil)`), la primera búsqueda descarga el mapa de excepciones de GitHub (máx. 5 s, cacheado en memoria); T20 puede inyectar un `ExceptionResolver` compartido o precargarlo al arrancar.
- T07: no se envían cabeceras `Accept`/`Accept-Language` y el sitio respondió igual (verificado en el smoke); si JKAnime las exige, añadirlas vía `ReqOptions.Header`.
- T07: las páginas de búsqueda capturadas (`bleach`, `naruto shippuden`) no traen paginación; si aparece, tratarla en la tarea que la necesite.
- T08: ~~**pendiente de cablear (reportado)**~~ → **resuelto en T09**: `Provider.GetAnimeDetails` (`jkanime.go`) delega en `getAnimeDetails` (`episodes.go`) y `search_test.go::TestMetodosNoImplementados` dejó de exigir `ErrNotImplemented` en ese método. Queda `GetVideoSources` con `ErrNotImplemented` (T11/T17/T20).
- T08: `detailMeta` **ya lo usa T09** (`/ajax/episodes/<NumericID>/<página>` con `_token: <CSRFToken>`); sus tests siguen siendo los únicos que lo prueban aislado.
- T08: `fetchDetail` no lista episodios (solo metadatos): la página los pinta el JS vía AJAX → **resuelto en T09**, `GetAnimeDetails` devuelve el detalle más todos los episodios del endpoint.
- T08: el tipo se toma de `li[rel="tipo"]`; si una página no trae esa etiqueta, `Type` queda en `unknown` (no se infiere de `Episodios: 1`).
- T10: `fetchEpisodePage` todavía no lo llama nadie fuera de sus tests: `Provider.GetVideoSources` sigue en `ErrNotImplemented` (T11/T17). **T09 no lo usa**: encadena detalle → `/ajax/episodes/…` (`StreamAnimeDetails`/`GetAnimeDetails`), no `/slug/<n>/`.
- T10: `EpisodePage` no se serializa a Wails hoy (tipo interno); si T20/T27 la exponen, recordar que `Headers` es `http.Header` (mapa `string → []string`).
- T10: la sesión depende de que el doer inyectado lleve cookiejar (`httpx.NewClient` lo pone). **Ya en uso por T09**: sin jar, el `_token` no se envía y la web da 419 → `StreamAnimeDetails` refresca el detalle una vez y si sigue sin jar vuelve a fallar con el 419 original en el lote final.
- T09: `StreamAnimeDetails` todavía no tiene consumidor: lo usará T20 (con `runtime.EventsEmit` por lote) mientras `GetAnimeDetails` ya lo usa vía `jkanime.go`. Sin consumidor, el único test de cancelación real es `TestStreamCancelacionSinFugas`.
- T09: `pausaPaginas` es una constante de 300 ms (compartida por stream y `GetAnimeDetails`): si T20 quiere otro ritmo hay que inyectarla como campo/argumento; decidirlo allí.
- T09: si falla una página de fondo, el lote final lleva el `Err` pero los episodios ya emitidos no se revocan; si falla `GetAnimeDetails` a mitad se devuelve `models.AnimeDetail{}` con el error (**sin** episodios parciales). Si T20 prefiere parciales, es cambio de contrato en esa tarea.
- T09: la página 1 se pide dentro de `StreamAnimeDetails` **antes** de emitir nada: si `NumericID`/`CSRFToken` faltan, o la página 1 da error, se devuelve el error sin lotes (el consumidor decide qué mostrar).
- T11: **nadie construye todavía el `Resolver`**: falta la cadena de extractores reales (T12–T15) y la conversión `EpisodePage → Page`, que es de **T17**. Hasta entonces `Resolve` solo se prueba con falsos.
- T11: `Resolve` agrega **todas** las fuentes en orden canónico de extractor, sin ordenar por calidad, sin deduplicar y sin descartar por tipo: eso es T16 (normalizador) + T17 (selector de mejor fuente). Si alguna tarea quiere "parar en la primera usable", es cambio de política y debe registrarse aquí.
- T11: `Resolve` no valida el esquema de las URLs (`http(s)://…`) ni completa `Headers`/`Referer`/`Subtitles`; solo exige que `URL` no esté en blanco. La validación queda en T16.
- T12: `JKPlayerExtractor` todavía no lo construye nadie: la cadena se arma en **T17** (`New(&JKPlayerExtractor{…}, …)` + conversión `EpisodePage → Page`).
- T12: el escaneo de `.vtt` está probado solo con HTML artificial (el sitio no publica softsubs hoy); si aparecen pistas reales, capturar el fixture y validar la inferencia de `Lang` con él.
- T12: `Type` puede quedar en `""` si ni la URL ni el cuerpo del player dan señal (hoy siempre hay `.m3u8`); T16/T17 deben tratar el tipo vacío.
- T12: cada iframe distinto (`?e=…` varía por episodio) dispara su propia petición: en una página con 4 referencias um/umv se hacen hasta 4 peticiones al player. Si molesta, deduplicar iframes por recurso base en la tarea que lo optimice (T16/T17).
- T13: `JKPHPExtractor` todavía no lo construye nadie: la cadena se arma en **T17** (ídem JKPlayer, T12).
- T13: no hay ninguna página de episodio real en `testdata/` que contenga `jk.php`/`jkplayer/jk?` (el `episode_page.html` de bleach solo emite `jkplayer/um`, `umv` y `c1`): `CanHandle` está probado con HTML artificial para esa parte y la respuesta con los **fixtures reales**. Si aparece una página real con la referencia, capturarla y añadirla al test.
- T13: `reJkPHP` no sigue referencias construidas por concatenación en JS (p. ej. `jk.php?u='+val.x+'`): el match se corta en la comilla y se haría una petición con `u=` vacío. No hay evidencia de ese patrón en las páginas reales; si aparece, tratarlo aquí.
- T13: los dos fixtures no traen `.vtt` (el test de subtítulos usa HTML artificial, igual que T12) y el tipo sale `ResourceMP4` gracias a `type: 'mp4'` en el cuerpo: un player sin extensión ni declaración dejaría `Type` en `""` (T16 lo trata).
- T13: la descarga de la página del player está duplicada con `JKPlayerExtractor.fuentesDePlayer` (mismo límite y mismos helpers): unificar en T29 si se quiere una sola función parametrizada por `Server`/`Extractor`.

## Bitácora

| Fecha | Tarea | Modelo usado | Resultado | Notas |
|---|---|---|---|---|
| 2026-10-04 | T01 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` y `go test -race ./...` en verde (`internal/logx` 6 tests PASS) |
| 2026-10-04 | T02 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (`internal/httpx` 26 tests PASS, `internal/logx` 6 PASS); solo se creó `internal/httpx/**` |
| 2026-10-04 | T03 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde (`internal/models` 8 tests PASS); solo se tocaron `internal/models/models.go` y `models_test.go` |
| 2026-10-04 | T05 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde (`internal/models` 5 tests/37 subtests PASS); solo se crearon `internal/models/ids.go` e `ids_test.go` |
| 2026-10-04 | T04 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (`internal/scraper` 7 tests PASS con `-race`); solo se crearon `internal/scraper/provider.go`, `registry.go` y `registry_test.go` |
| 2026-10-04 | T06 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (`internal/scraper/jkanime` 11 tests/24 con subtests PASS con `-race`); solo se crearon `internal/scraper/jkanime/exceptions.go` y `exceptions_test.go` |
| 2026-10-04 | T07 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` (+ `-tags integration`) limpio, `go test -race ./...` en verde (paquete `jkanime`: 20 tests top-level PASS con `-race` = 11 de T06 + 9 de T07); smoke `go test -tags integration ./internal/scraper/jkanime/ -run RedReal -v` PASS con 10 resultados reales. Fixtures `search.html` y `search_empty.html` capturados con `curl.exe` (HTTP 200, sin challenge), ruta `/buscar/<query>` verificada en vivo; solo se tocaron `jkanime.go`, `search.go`, `search_test.go`, `search_integration_test.go` y `testdata/` |
| 2026-10-05 | T08 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` (+ `-tags integration`) limpio, `go test -race ./...` en verde (paquete `jkanime`: 27 tests top-level PASS con `-race` = 11 T06 + 9 T07 + 7 T08). Smoke `go test -tags integration ./internal/scraper/jkanime/ -run RedReal -v` PASS: `bleach` → Type=series, 366 episodios, NumericID=278; `bleach-the-diamonddust-rebellion` → Type=movie, 1 episodio, NumericID=504, CSRF de 40 chars. Fixtures `detail.html` y `detail_movie.html` capturados con `curl.exe` (HTTP 200, sin challenge); solo se tocaron `detail.go`, `detail_test.go`, `detail_integration_test.go` y `testdata/` |
| 2026-10-05 | T10 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` (+ `-tags integration`) limpio, `go test -race ./...` en verde (paquete `jkanime`: 33 tests top-level PASS con `-race` = 27 + 6 de T10). Rutas verificadas en vivo con `curl.exe`: `/bleach/1/` → 200 (61 KB), `/bleach/367/` → 404, `/bleach/<n>/` → 200, película `/slug/pelicula/` → 200 (y `/slug/1/` → 302 hacia ella), especial `/slug/especial/` → 200 (y `/slug/1/` → 302). Fixture `episode_page.html` capturado con `curl.exe` (HTTP 200, sin challenge); solo se tocaron `episode_page.go`, `episode_page_test.go` y `testdata/` |
| 2026-10-05 | T09 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` (+ `-tags integration`) limpio, `go test -race -count=1 ./...` en verde (paquete `jkanime`: **40 tests top-level PASS con `-race`** = 33 + 7 de T09). Smoke `go test -tags integration ./internal/scraper/jkanime/ -run RedReal -v` PASS (incluido `TestEpisodiosRedReal`: bleach, 16 episodios en la página 1, cancelado tras 2 lotes). Endpoint verificado con Playwright MCP + `curl.exe`: `POST /ajax/episodes/278/{1,2,23}` → 200 con jar+token (`X-Requested-With` opcional), **419** sin cookie/con `_token` erróneo, **405** con GET, 200 tras refrescar el detalle; `last_page`=23 y 16 eps/página. Fixtures `episodes_p1/p2/last.json` + `episodes_movie.json` capturados sin sanitizar (sin cookies/tokens en la respuesta); solo se tocaron `episodes.go`, `episodes_test.go`, `episodes_integration_test.go`, `testdata/`, `jkanime.go` y `search_test.go::TestMetodosNoImplementados` |
| 2026-10-05 | T11 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (paquete nuevo `internal/scraper/jkanime/extractors`: **10 tests / 16 con subtests PASS con `-race`**; el resto de paquetes sigue en verde). Comprobación de la tarea sin fixtures ni red: HTML artificial, falsos extractores y `httptest.Server` para `Base.Doer` (con captura del `slog.Warn`); solo se crearon `internal/scraper/jkanime/extractors/extractor.go` y `extractor_test.go` |
| 2026-10-05 | T12 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race ./...` en verde (paquete `extractors`: **22 tests / 35 con subtests PASS con `-race`** = 10/16 de T11 + **12/19 de T12**; resto de paquetes en verde). Fixtures del player capturados con Playwright MCP + `curl.exe` (`jkplayer_um.html`, `jkplayer_umv.html`, y de paso `jkplayer_jk.html` para T13), sanitizados (`st=`/`e=`/base64 `atob`); **sin `.vtt` reales** → test de subtítulos con HTML artificial (autorizado). `CanHandle` cubre `um` y `umv`; dedup por recurso base; solo se tocaron `jkplayer.go`, `jkplayer_test.go` y `testdata/` |
| 2026-10-05 | T13 | opencode/mimo-v2.6-flash-free | OK | `gofmt -l .` sin salida, `go vet ./...` limpio, `go test -race -count=1 ./...` en verde (paquete `extractors`: **33 tests / 53 con subtests PASS con `-race`** = 10/16 de T11 + 12/19 de T12 + **11/18 de T13**; resto de paquetes en verde). Comprobación de la tarea sin red: fixtures reales `jkphp.html` y `jkplayer_jk.html` servidos con `httptest.Server` (los dos → `Type=mp4`, `Server=jkphp`, Referer/UA/cookie comprobados), más dedup, subtítulos, 500, sin doer, ctx cancelado y cadena canónica. `extractor.go` sin cambios (igual que T12: `NameJKPHP` ya estaba en `ordenCanonico`); solo se crearon `jkphp.go` y `jkphp_test.go` |
