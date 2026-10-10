# Firmas reales — internal/scraper

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T04 — internal/scraper` → creado por **T04** (Interfaz Provider y registro).
- `### T43 — internal/scraper` → creado por **T43** (interfaz opcional `HomeProvider`).

---

### T04 — internal/scraper

- `type Provider interface { Name() string; Search(ctx context.Context, query string) ([]models.Anime, error); GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error); GetVideoSources(ctx context.Context, episodeID string, opts SourceOpts) ([]models.VideoSource, error) }` — exactamente la firma del plan.
- `type EpisodeStreamer interface { StreamAnimeDetails(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) }` — interfaz opcional (T09/T20).
- `type SourceOpts struct { NoCache bool }` con tag `json:"no_cache"` — el reproductor pide una fuente nueva tras un error fatal.
- `var ErrNoDetails = errors.New("scraper: este proveedor no ofrece detalle de anime")` — causa sentinela para `errors.Is` (T18).
- `type Registry struct { … }` + `func NewRegistry() *Registry`, `func (r *Registry) Register(p Provider) error`, `func (r *Registry) Get(name string) (Provider, bool)`, `func (r *Registry) All() []Provider`. `sync.RWMutex`, valor cero utilizable (mapa perezoso), duplicado ⇒ reemplazo (gana el último), `All()` devuelve una copia ordenada por nombre.
- Tests: `internal/scraper/registry_test.go` (7 tests, red de nada, corren con `-race`): `fakeProvider` (falso con `Provider` + `EpisodeStreamer`), Get/Register, duplicados, registros inválidos, valor cero, orden y snapshot de `All()`, concurrencia (50 goroutines) y `ErrNoDetails`.

### T43 — internal/scraper

- `type HomeProvider interface { Home(ctx context.Context) ([]models.HomeSection, error) }` — interfaz **opcional** al estilo de `EpisodeStreamer` (T09): `Provider` y `Registry` no cambian y nadie la exige; un provider que la implementa ofrece secciones para la vista de inicio (v1.2, `tareas/v1.2_PLAN.md`). Añadida al final de `provider.go`.
- Tests: **`TestHomeProviderEsOpcional`** en `internal/scraper/registry_test.go` (con el `fakeProvider` existente y el nuevo `fakeConHome` que embebe `fakeProvider` y añade `Home`): un `Provider` sin `Home` no cumple `HomeProvider`, uno con `Home` sí, su `Home` devuelve una sección y el `Registry` sigue registrando providers sin `Home` sin error. `internal/scraper`: **7 → 8** tests.
