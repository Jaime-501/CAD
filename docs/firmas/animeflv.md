# Firmas reales — internal/scraper/animeflv

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper/animeflv`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T18 — internal/scraper/animeflv` → creado por **T18** (AnimeFLV — búsqueda con fallback).

---

### T18 — internal/scraper/animeflv

- `const DefaultBaseURL = "https://animeflv.or.at"` — sitio WordPress real (el clásico animeflv.net está muerto; no "arreglar" la URL). Privadas: `providerName = "animeflv"`, `searchQuery = "s"` (la búsqueda es `/?s=<query>`), `rutaAnime = "/anime/"` (prefijo del que sale el slug) y `maxSearchBytes = 10 << 20` (10 MB).
- `type AnimeFLV struct { BaseURL string; Doer httpx.HTTPDoer }` — valor cero utilizable (base vacía → `DefaultBaseURL`; sin doer, `httpx` devuelve `ErrNilDoer` sin red).
- `func New(doer httpx.HTTPDoer, baseURL string) *AnimeFLV` — entregable de T18; normaliza la base (sin espacios ni barra final). Assert en el código: `var _ scraper.Provider = (*AnimeFLV)(nil)`.
- `func (a *AnimeFLV) Name() string` → `"animeflv"` (el mismo que viaja en `models.Anime.Provider`).
- `func (a *AnimeFLV) Search(ctx context.Context, query string) ([]models.Anime, error)`:
  - query vacía/espacios → `[]models.Anime{}` **sin** petición.
  - URL: `<base>/?s=<url.QueryEscape(query)>` vía `httpx.Get` con `ReqOptions{Provider: "animeflv", Operation: "search"}` → UA `httpx.DefaultUserAgent`, `ctx` propagado, status ≥ 400 como `*httpx.ScraperError`.
  - Selectores (fixture real): `.search-series-card` → `a.thumbnail-link` (href), `h2.entry-title` (título), `img.anime-image` (portada). El **slug** sale de la ruta `/anime/<slug>/` del href (host ignorado, para que el fixture funcione igual contra el sitio real y contra `httptest.Server`); rutas ajenas, sin slug o anidadas se descartan, igual que una tarjeta sin título.
  - Portada relativa → absoluta contra la base (`absoluta`); la absoluta se conserva tal cual. Sin resultados → slice vacío, no error.
- `func (a *AnimeFLV) GetAnimeDetails(ctx, id) (models.AnimeDetail, []models.Episode, error)` → `(models.AnimeDetail{}, nil, scraper.ErrNoDetails)` — `HasDetails: false`; el llamador usa `errors.Is` para mostrar "Detalle no disponible para este proveedor".
- `func (a *AnimeFLV) GetVideoSources(ctx, episodeID, opts scraper.SourceOpts) ([]models.VideoSource, error)` → `nil` + `fmt.Errorf("animeflv: este proveedor no ofrece fuentes de vídeo (episodio %q)", episodeID)` — error descriptivo, **no** `scraper.ErrNoDetails`.
- Fixtures (respuesta cruda con `curl.exe`, UA de navegador, HTTP 200; `commentsNonce` saneado a `NONCE_DE_PRUEBA`, regla 12):
  - `testdata/search.html` — `GET /?s=bleach`, 60253 B, 1 tarjeta: id `bleach-sennen-kessen-hen-kashin-tan`, título "Bleach: Sennen Kessen-hen - Kashin-tan", portada absoluta.
  - `testdata/search_empty.html` — `GET /?s=shingeki`, 60097 B, 0 tarjetas (bloque `.no-results`).
  - Sin challenge de Cloudflare (sin "Just a moment" ni cabecera `cf-mitigated`).
- Tests: `animeflv_test.go` (10 tests, 13 con subtests; `httptest.Server` + fixtures, red de nada; `TestMain` instala un logger descartable para `logx.Request`): ruta/query/UA exactas, escape de la query, query vacía sin red, sin resultados, 500 → `httpx.ErrServerError`, cuerpo > 10 MB, descarte de enlaces (HTML artificial, regla 11), `ErrNoDetails`, error descriptivo de fuentes y `New`/`Name`. `animeflv_integration_test.go` — smoke `TestSearchRedReal` con `//go:build integration` (pasó contra el sitio real: 1 resultado).
- Consumidor hoy: `app.go` (`SearchAnime` → `animeflv.New(httpx.NewClient(httpx.ClientOptions{}), "").Search(a.ctx, query)`); **T20** rehará `app.go` con el `Registry`.
