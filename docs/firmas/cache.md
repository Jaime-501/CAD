# Firmas reales — internal/cache (+ integración en app_nucleo.go)

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/cache`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T21 — internal/cache` → creado por **T21** (Caché local).
- `### T21 — integración en el núcleo` → creado por **T21** (el bloque de `app.md` **no se tocó**: T21 no lo tenía en "Puedes tocar"; quien vuelva a tocar `app_nucleo.go` debe rellenarlo allí).

---

### T21 — internal/cache

Caché en memoria con TTL de detalle + episodios por anime. Cero valor de `Cache` no es válido: usa `New`. Segura para concurrencia (`sync.RWMutex`); por sí sola **no** crea goroutines (solo con `Arrancar`).

- `type Option func(*Cache)` — configuración inyectable en `New`:
  - `func ConReloj(ahora func() time.Time) Option` — reloj de la caché (los tests avanzan el tiempo sin `time.Sleep`). Un nil se ignora.
  - `func ConIntervalo(d time.Duration) Option` — cadencia de la limpieza periódica; con `<= 0` no hay rutina.
- `func New(ttl time.Duration, opts ...Option) *Cache` — **entregable de T21**. `ttl <= 0` ⇒ "no caduca" y no hay rutina de limpieza.
- `type Entrada struct { Detail models.AnimeDetail; Episodes []models.Episode; Complete bool }` — lo que devuelve `Get`.
- `func Clave(proveedor, animeID string) string` — clave de entrada (`"jkanime:bleach"`); el prefijo separa proveedores.
- `func (c *Cache) Get(clave string) (Entrada, bool)` — **devuelve copia** (episodios y `Detail.Genres`: mutar lo devuelto no toca la caché). Una entrada caducada se elimina ahí mismo (limpieza en lectura) y se devuelve como ausente.
- `func (c *Cache) Set(clave string, det models.AnimeDetail)` — guarda **SOLO el detalle** (crea la entrada si falta): no toca `Episodes` ni `Complete`, que solo mueven `Append` y `MarkComplete`. Así el detalle puede llegar después de los lotes sin pisarlos.
- `func (c *Cache) Append(clave string, eps []models.Episode)` — acumula deduplicando por `Episode.ID` (una entrada **sin ID** se añade siempre), renueva el TTL y no cambia `Detail` ni `Complete`. Lista vacía = no hace nada (ni crea entradas).
- `func (c *Cache) MarkComplete(clave string)` — marca `Complete = true` (crea la entrada si falta: el Done puede llegar antes que el detalle).
- `func (c *Cache) Delete(clave string)` — borra la entrada (existente o no).
- `func (c *Cache) Arrancar()` / `func (c *Cache) Cerrar()` — rutina de poda: `Arrancar` es idempotente y no arranca nada con `ttl <= 0`; `Cerrar` **espera a que termine** la goroutine (ninguna sobrevive al shutdown) y también es idempotente. El núcleo las llama en su `arrancar`/`cerrar`.

**Comportamiento** (todo testeado en `internal/cache/cache_test.go` con reloj falso):

- **Caducidad por deslizamiento**: cada escritura (`Set`/`Append`/`MarkComplete`) renueva el TTL; `Get` no lo renueva. Una escritura sobre una entrada ya caducada la **reinicia** (no revive datos viejos).
- **Limpieza**: en la propia lectura (`Get`) + rutina periódica con cadencia `ttl/2` acotada a `1 s..60 s` (por defecto `intervaloPara`).
- **Copias también al escribir**: `Set` copia `Genres` y `Append` no retiene el slice del llamador.

---

### T21 — integración en el núcleo

(`app_nucleo.go`; `app.md` documenta el bloque T20 y **no se tocó** en T21 → véase arriba.)

- `const ttlCacheDetalle = 10 * time.Minute` — TTL con el que la app crea su caché.
- `type streamActivo struct { clave string; ctx context.Context; cancel context.CancelFunc; fallo atomic.Bool }` — reemplaza al `cancel` suelto de T20: el ÚNICO stream de detalle vivo, bajo el mutex del núcleo. `fallo` lo pone `lote` cuando el lote final (`Done`) traía `Err`; es atómico porque lo escribe quien recibe los lotes y lo lee `detalle()`.
- `func nuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor, c *cache.Cache) *nucleo` — **firmas T21**: añade la caché; una nil se sustituye por `cache.New(ttlCacheDetalle)`. `nuevoNucleoPorDefecto()` la crea con el TTL por defecto.
- `func (n *nucleo) arrancar(emis Emisor) error` — además de emisor y proxy, llama `cacheLocal.Arrancar()`.
- `func (n *nucleo) cerrar() error` — cancela el stream activo, **borra su entrada incompleta**, llama `cacheLocal.Cerrar()` y para el proxy. Idempotente.
- `func (n *nucleo) detalle(ctx context.Context, proveedor, id string) (DetalleResultado, error)` — punto de la caché:
  - entrada `Complete=true` → hit: devuelve **todo** ordenado por `Number`, suelta el stream de OTRO anime y emite `episodes:complete {anime_id, error:""}` **de inmediato**;
  - entrada `Complete=false` con **stream vivo** (`streamDe` + `ctx.Err()==nil`) → se **adjunta**: devuelve lo acumulado y NO lanza otro stream;
  - entrada incompleta **sin stream vivo** → huérfana: se borra y se pide de nuevo;
  - miss → `iniciarStream` (cancela el de OTRO anime y borra su entrada incompleta) y, si el provider es `scraper.EpisodeStreamer`, streaming con `lote`; si no (flujo síncrono) **no se cachea**.
  - Si el stream cierra con `Err` (`st.fallo`), no se escribe nada más y se borra la entrada: los errores no se cachean.
- `func (n *nucleo) lote(ctx context.Context, st *streamActivo, id string) func(models.EpisodeBatch)` — callback de lotes: `Append` (solo si `Err == ""`), `Done` sin error → `MarkComplete`, `Done` con `Err` → `st.fallo = true` **antes** de `Delete`, y emite `episodes:update` (+ `episodes:complete` si `Done`) comprobando antes `ctx.Err()`.
- Auxiliares: `func (n *nucleo) iniciarStream(base context.Context, clave string) (context.Context, *streamActivo)` · `soltarStreamDistinto(clave string)` · `streamDe(clave string) *streamActivo` · `liberarStream(st *streamActivo)` · `borrarSiIncompleta(clave string)` · `emitirCompleto(ctx context.Context, animeID, texto string)` · `func detalleDesdeCache(e cache.Entrada) DetalleResultado` (ordena por `Number`).

**Qué NO se cachea**: `GetVideoSources` (regla 9 → `noCache` se acepta, reenvía e **ignora**), errores, streams terminados con `Err` ni el flujo síncrono.

**Tests** (`app_nucleo_cache_test.go`, 6 tests, sin red; `app_test.go` aporta los dobles y `nuevoEntornoConCache`): hit completo no vuelve al proveedor · hit a mitad de paginación se adjunta (un solo `StreamAnimeDetails`, los mismos lotes) · cambio de anime borra la entrada incompleta (y el lote tardío del cancelado no la recrea) · `Done` con `Err` y `StreamAnimeDetails` con error no se cachean · entrada caducada vuelve al proveedor · `cerrar` borra la entrada del stream cancelado.
