# Firmas reales — app.go / núcleo (paquete main)

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc .` en la raíz del proyecto).**
> Añade bloques nuevos con el formato `### Txx — <paquete>`.

**Bloques de este archivo y tarea que los creó:**

- `### T20 — app.go / app_nucleo.go` → creado por **T20** (Integración en app.go y emisión Wails).

---

### T20 — app.go / app_nucleo.go

**Archivos:** `app.go` (delgado: métodos expuestos a Wails y ciclo de vida, nada más), `app_nucleo.go` (toda la orquestación, testeable con dobles), `app_test.go` (18 tests, sin red ni Wails), `app_integration_test.go` (smoke con red, `//go:build integration`). `main.go` se tocó **solo** para añadir `OnShutdown: app.shutdown` (autorizado por el humano).

#### Wails (`app.go`)

- `type App struct { mu sync.RWMutex; ctx context.Context; core *nucleo }` — `mu` protege `ctx`, que es **nil antes de `startup`**; `contexto()` devuelve `context.Background()` en ese caso (el nil nunca llega a los proveedores).
- `func NewApp() *App` — construye `nuevoNucleoPorDefecto()`; **no** toca la red ni escucha puertos.
- `func (a *App) startup(ctx context.Context)` — guarda el ctx, `slog.SetDefault(logx.New(os.Getenv("MIAPP_DEV") != ""))` (instala logx en `app.go`, pendiente de **T01**, y unifica el `slog` de jkanime, pendiente de **T06**) y `core.arrancar(emisor)` con `emisor = func(ev string, data any) { runtime.EventsEmit(ctx, ev, data) }`.
- `func (a *App) shutdown(_ context.Context)` — `core.cerrar()`; registrado como `OnShutdown` en `main.go`.
- `func (a *App) contexto() context.Context`
- `func (a *App) Greet(name string) string` — se conserva (plantilla Wails).
- `func (a *App) SearchAnime(query string) ([]models.Anime, error)` — **conservado** tal cual lo consume `App.svelte`; **no existe `Search`**.
- `func (a *App) GetAnimeDetails(provider string, id string) (DetalleResultado, error)`
- `func (a *App) GetVideoSources(provider string, episodeID string, noCache bool) ([]models.VideoSource, error)`
- **Límite de Wails descubierto:** `binding.BoundMethod.Call` solo procesa 1 o 2 valores de retorno; con 3 devuelve `(nil, nil)` y **traga el error**. Por eso los métodos expuestos devuelven siempre `(valor, error)` y el detalle llega en un struct.

#### Núcleo (`app_nucleo.go`)

- `const proveedorPrincipal = "jkanime"`, `const proveedorFallback = "animeflv"` — orden del fallback de búsqueda.
- `var ErrDetalleNoDisponible = errors.New("Detalle no disponible para este proveedor")` — traducción de `scraper.ErrNoDetails` (AnimeFLV, `HasDetails=false`). **Decisión:** no es un fallo grave → se devuelve como error tipado con mensaje listo para la UI, se loguea a nivel `info` y no se emite ningún evento. `errors.Is(err, scraper.ErrNoDetails)` **no** se conserva fuera del núcleo (Wails solo entrega el texto al frontend, y este es el útil).
- Dependencias inyectadas (interfaces en el propio archivo):
  - `type RegistroProviders interface { Get(name string) (scraper.Provider, bool); All() []scraper.Provider }`
  - `type ProxyLocal interface { Register(models.VideoSource) models.VideoSource; Escuchar() (string, error); Cerrar() error }` — lo cumple `*stream.Proxy`.
  - `type Resolutor func(fuentes []models.VideoSource) (models.VideoSource, bool)` — en producción `extractors.SelectBestSource` (T17).
  - `type Emisor func(evento string, data any)` — en producción `runtime.EventsEmit`; nil = no emitir (antes de `startup`).
- `type DetalleResultado struct { Detail models.AnimeDetail \`json:"detail"\`; Episodes []models.Episode \`json:"episodes"\` }` — respuesta de `GetAnimeDetails`: detalle + primera tanda de episodios.
- `type EpisodiosCompletados struct { AnimeID string \`json:"anime_id"\`; Error string \`json:"error"\` }` — payload de `episodes:complete`; `Error` es texto en español (vacío = todo OK). Un `error` de Go **nunca** se serializa.
- `func nuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor) *nucleo`
- `func nuevoNucleoPorDefecto() *nucleo` — un único `httpx.NewClient(httpx.ClientOptions{})` compartido por `jkanime.New(doer, "", nil)` y `animeflv.New(doer, "")` (mismo jar; el jar separa por host, no se mezclan sesiones), `stream.NewProxy(nil)` (cliente propio del proxy con anti-SSRF y **sin cookies**; inyectar el doer de arriba haría perder el control de red, decisión T19) y `extractors.SelectBestSource`. El `ExceptionResolver` de JKAnime queda dentro de su `JkAnime` (una sola descarga del mapa de excepciones por proceso, T06); **no se precarga** en el arranque.
- `func (n *nucleo) arrancar(emis Emisor) error` — instala el emisor y llama `proxy.Escuchar()` (sin eso, `Register` devuelve las fuentes sin registrar, T19).
- `func (n *nucleo) cerrar() error` — cancela el stream activo (si lo hay) y `proxy.Cerrar()`; idempotente. Se llama desde `shutdown`.
- `func (n *nucleo) buscar(ctx context.Context, query string) ([]models.Anime, error)` — recorre `[jkanime, animeflv]`: si un proveedor falla se loguea (`slog.Warn`) y sigue; si devuelve resultados, los devuelve ahí mismo; si devuelve vacío, pasa al siguiente. Devuelve **lista vacía sin error** si al menos un proveedor pudo buscar, y `errors.Join` de todos los fallos (mensaje en español con los nombres) **solo si ninguno pudo**.
- `func (n *nucleo) detalle(ctx context.Context, proveedor, id string) (DetalleResultado, error)`:
  1. proveedor desconocido → `"proveedor desconocido: %q (disponibles: jkanime, animeflv)"`;
  2. `iniciarStream` cancela **siempre** el stream anterior y crea el ctx del nuevo (un solo `cancel` activo, bajo mutex);
  3. si el provider cumple `scraper.EpisodeStreamer` → `StreamAnimeDetails(ctx, lote(ctx, id))`, devuelve detalle + página 1 y **no** cancela su ctx (lo cancelará la próxima llamada o `cerrar`);
  4. si no → `GetAnimeDetails` síncrono, cancela su ctx al salir y **no emite eventos**;
  5. `scraper.ErrNoDetails` → `ErrDetalleNoDisponible`.
- `func (n *nucleo) fuentes(ctx, proveedor, episodeID string, noCache bool) ([]models.VideoSource, error)` — `GetVideoSources` del provider → `SelectBestSource` (mejor **primero**) → `proxy.Register` **solo** para `Type == "hls" | "mp4"` (los iframes y los tipos vacíos quedan intactos) → reordena con la mejor en cabeza y el resto en orden original. **`noCache` se acepta y se reenvía al provider en `SourceOpts`, pero no lo consume nadie**: no hay ninguna caché (regla 9), T17 lo ignora y la caché local es **T21**.
- Internas: `proveedor`, `iniciarStream`, `lote`, `emisorActual`, `nombres`; libres `errorDeDetalle`, `indiceDe` (la mejor se localiza por URL+tipo; `Normalize` garantiza URLs únicas, T16).
- El núcleo **no crea goroutines**: el streaming lo hace el provider; aquí solo se pasa el `onBatch` y se cancela el ctx.

#### Eventos

| Evento | Payload | Cuándo |
|---|---|---|
| `episodes:update` | `models.EpisodeBatch` (`anime_id`, `batch`, `episodes`, `done`, `err`) | cada lote que entrega `onBatch`, **incluido** el final (`done=true`, suele venir sin episodios) |
| `episodes:complete` | `EpisodiosCompletados` = `{anime_id, error}` | cuando llega un lote con `Done`; `error` vacío si todo fue bien |

- Antes de **cada** emisión se comprueba `ctx.Err()` (y que haya emisor instalado): un stream cancelado —por pedir otro anime o por shutdown— no emite nada más, ni siquiera el lote final.
- No se emite nada cuando: aún no ha corrido `startup` (emisor nil), el provider no implementa `EpisodeStreamer`, o la llamada falla de forma síncrona (el error vuelve por el valor de retorno).
- **Ojo (para T26):** los eventos pueden llegar **antes** de que la promesa de `GetAnimeDetails` se resuelva, porque jkanime emite desde su goroutine: acumular por `anime_id` + `batch`, sin asumir orden.

#### Tests

- `app_test.go` — **18 tests** con dobles (`scraper.NewRegistry` real + providers/proxy/emisor falsos), sin red y sin Wails, en verde con `go test -race`:
  - búsqueda (6): gana JKAnime sin tocar el fallback · fallback por lista vacía · fallback por error · ambos caídos → error con los dos mensajes · sin resultados no es error · sin proveedores registrados no revienta;
  - detalle (6): proveedor desconocido con la lista de disponibles · `ErrNoDetails` → `ErrDetalleNoDisponible` (y no arrastra el sentinel de scraper) · sin `EpisodeStreamer` no emite · lotes + `episodes:complete` · fallo de página llega como `error` en `complete` · **cambio de anime cancela el stream anterior y sus lotes tardíos se suprimen**;
  - ciclo de vida (3): `shutdown` cancela el stream y cierra el proxy (y es idempotente) · sin emisor no falla · `arrancar` instala el emisor y hace `Escuchar`;
  - fuentes (3): solo HLS/MP4 pasan por el proxy con la mejor primero (el iframe queda intacto y sin registrar) · `noCache` se reenvía y no hay caché (dos llamadas = dos peticiones al provider) · proveedor desconocido.
- `app_integration_test.go` — `TestBusquedaRedReal` (`//go:build integration`): `nuevoNucleoPorDefecto()` + `buscar("bleach")` contra la red real; exige resultados y que el primero sea de `jkanime` (si falla, el fallback lo taparía y el smoke lo dice en el mensaje). Ejecutado en verde: **10 resultados**.

#### Decisiones y límites

1. **`main.go` editado con autorización explícita** (1 línea, `OnShutdown: app.shutdown`): reportado antes de tocar nada por la regla de escape, porque sin ese hook no hay parada del proxy ni cancelación de streams.
2. El emisor se instala **solo en `startup`** y captura ese ctx: `runtime.EventsEmit` con un ctx sin valores de Wails termina el proceso (`log.Fatalf` dentro de `getEvents`), así que emitir con `context.Background()` está prohibido.
3. El logger se decide con la variable `MIAPP_DEV` (Wails no expone su flag de debug desde fuera: `internal/app.IsDebug` es interno).
4. Si `Escuchar()` falla en `startup`, la app sigue viva: se loguea `slog.Warn` y las fuentes saldrían sin registrar (hoy sin aviso en la UI → **T27**).
5. `GetVideoSources` de AnimeFLV sigue devolviendo su error descriptivo (el proveedor es solo fallback de búsqueda): no se ofrece reproducción desde ese proveedor.
