# Firmas reales — internal/scraper

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T04 — internal/scraper` → creado por **T04** (Interfaz Provider y registro).

---

### T04 — internal/scraper

- `type Provider interface { Name() string; Search(ctx context.Context, query string) ([]models.Anime, error); GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error); GetVideoSources(ctx context.Context, episodeID string, opts SourceOpts) ([]models.VideoSource, error) }` — exactamente la firma del plan.
- `type EpisodeStreamer interface { StreamAnimeDetails(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) }` — interfaz opcional (T09/T20).
- `type SourceOpts struct { NoCache bool }` con tag `json:"no_cache"` — el reproductor pide una fuente nueva tras un error fatal.
- `var ErrNoDetails = errors.New("scraper: este proveedor no ofrece detalle de anime")` — causa sentinela para `errors.Is` (T18).
- `type Registry struct { … }` + `func NewRegistry() *Registry`, `func (r *Registry) Register(p Provider) error`, `func (r *Registry) Get(name string) (Provider, bool)`, `func (r *Registry) All() []Provider`. `sync.RWMutex`, valor cero utilizable (mapa perezoso), duplicado ⇒ reemplazo (gana el último), `All()` devuelve una copia ordenada por nombre.
- Tests: `internal/scraper/registry_test.go` (7 tests, red de nada, corren con `-race`): `fakeProvider` (falso con `Provider` + `EpisodeStreamer`), Get/Register, duplicados, registros inválidos, valor cero, orden y snapshot de `All()`, concurrencia (50 goroutines) y `ErrNoDetails`.
