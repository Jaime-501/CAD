# Firmas reales — internal/scraper/jkanime

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper/jkanime`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T06 — internal/scraper/jkanime` → creado por **T06** (Excepciones; `exceptions.go`).
- `### T07 — internal/scraper/jkanime` → creado por **T07** (Búsqueda; `jkanime.go`, `search.go`).
- `### T08 — internal/scraper/jkanime` → creado por **T08** (Detalle y metadatos; `detail.go`).
- `### T10 — internal/scraper/jkanime` → creado por **T10** (Página de episodio; `episode_page.go`).
- `### T09 — internal/scraper/jkanime` → creado por **T09** (Episodios AJAX; `episodes.go`).

---

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
