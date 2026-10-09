# Firmas reales — internal/models

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/models`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T03 — internal/models` → creado por **T03** (Modelos de dominio; `models.go`).
- `### T05 — internal/models (ids.go)` → creado por **T05** (Referencia de episodio; `ids.go`).
- `### T33 — internal/models (VideoSource.Language)` → creado por **T33** (idioma estructurado de una fuente; `models.go`).

---

### T03 — internal/models

- `type Anime struct { ID, Title, CoverURL, Provider string }` — sin cambios; tags `id`, `title`, `cover_url`, `provider`.
- `type EpisodeKind string` + `KindEpisode = "episode"`, `KindMovie = "movie"`, `KindSpecial = "special"`.
- `type AnimeDetail struct { ID, Title, Provider, Type, Status, Synopsis, Cover string; Genres []string; TotalEpisodes int }` — tags `id`, `title`, `provider`, `type`, `status`, `genres`, `synopsis`, `cover`, `total_episodes`.
- `type Episode struct { ID string; Number int; Title, Thumbnail string; Kind EpisodeKind }` — tags `id`, `number`, `title`, `thumbnail`, `kind`.
- `type ResourceType string` + `ResourceHLS = "hls"`, `ResourceMP4 = "mp4"`, `ResourceIframe = "iframe"`.
- `type Subtitle struct { URL, Lang string }` — tags `url`, `lang`.
- `type VideoSource struct { URL string; Type ResourceType; Quality, Server string; Language string; Headers map[string]string; Referer string; Subtitles []Subtitle }` — tags `url`, `type`, `quality`, `server`, `language`, `headers`, `referer`, `subtitles` (**`omitempty` solo en `language` y `subtitles`**; `Language` lo añadió **T33**, ver el bloque de abajo).
- `type WatchProgress struct { AnimeID, EpisodeID string; CurrentTime, TotalDuration float64; IsWatched bool }` — tags `anime_id`, `episode_id`, `current_time`, `total_duration`, `is_watched`.
- `type EpisodeBatch struct { AnimeID string; Batch int; Episodes []Episode; Done bool; Err string }` — tags `anime_id`, `batch`, `episodes`, `done`, `err`.
- Tests: `internal/models/models_test.go` (8 tests con subtests, red de nada): ida y vuelta JSON de cada tipo, etiquetas exactas y omisión de `subtitles`. **T33 añadió un noveno test** (`TestVideoSourceIdioma`, ver abajo).

### T05 — internal/models (ids.go)

- `var ErrInvalidEpisodeID = errors.New("id de episodio inválido")` — causa sentinela para `errors.Is`; los errores concretos se envuelven con `fmt.Errorf("%w: …", ErrInvalidEpisodeID)` y mensaje en español.
- `func NewEpisodeID(provider, slug string, kind EpisodeKind, number int) (string, error)` — `provider:slug:episode:<n>`, `provider:slug:special:<n>`, `provider:slug:movie` (para `KindMovie` el `number` **no** se serializa, tampoco si llega ≠ 0). Rechaza proveedor/slug vacío o con `:`, `number < 0` y `kind` desconocido.
- `func ParseEpisodeID(id string) (provider, slug string, kind EpisodeKind, number int, err error)` — inverso exacto de `NewEpisodeID`; 3 partes ⇒ `KindMovie` con `number = 0`; 4 partes ⇒ `KindEpisode`/`KindSpecial` con número **canónico** (`strconv.Itoa(n) == parte`: rechaza `01`, `+1`, `-1`, `abc`).
- Tests: `internal/models/ids_test.go` (5 tests de tabla, 37 subtests, red de nada): New válido, New inválido, Parse válido, Parse inválido y `Parse(New(x)) == x` (+ id reconstruido idéntico).

### T33 — internal/models (VideoSource.Language)

Único cambio de `internal/models` en la rama `feature/animeav1` (T33, fuentes de vídeo de AnimeAV1).

- `const ( LanguageSub = "SUB"; LanguageDub = "DUB" )` — idiomas estructurados de una fuente. **No es un enum**: un proveedor sin idioma (JKAnime) o cuyo sitio no lo declara deja `Language` vacío.
- `VideoSource.Language string` con tag `json:"language,omitempty"`, colocada **entre `Server` y `Headers`** (ese orden fija el del modelo TS que genera Wails). Doc del campo: la etiqueta visible la compone el frontend a partir de `Server + Language`, y nadie debe parsearla.
- Con `omitempty`, un proveedor sin idioma serializa `VideoSource` igual que antes → JSON y payloads de eventos **retrocompatibles**; el modelo TS solo cambia al regenerarlo.
- Tests: `TestVideoSourceIdioma` (9 tests en `internal/models/models_test.go`, red de nada): constantes `SUB`/`DUB`, ida y vuelta con `Language` informado, etiqueta exacta en el JSON (`…"server":"HLS","language":"SUB","headers"…`) y omisión cuando es `""`.
- Consumidor: `animeav1.GetVideoSources` (T33) rellena el campo en todas sus fuentes. **Pendiente de T34**: regenerar `frontend/wailsjs/**` con `docker compose run --rm app wails generate module` (nunca a mano) para que el modelo TS lleve `language`.
