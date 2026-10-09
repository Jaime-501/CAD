# Firmas reales — app.go (raíz) / núcleo (internal/app)

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/app` y `go doc .` en la raíz del proyecto).**
> Añade bloques nuevos con el formato `### Txx — <paquete>`.
>
> **Desde T37** el núcleo vive en `internal/app` (`package app`) y los bloques T20/T21/T22/T34 de este archivo describen ese mismo código **antes** del movimiento (nombres en minúscula, `app_nucleo.go` en la raíz). Para el estado actual, el bloque **T37** manda.

**Bloques de este archivo y tarea que los creó:**

- `### T20 — app.go / app_nucleo.go` → creado por **T20** (Integración en app.go y emisión Wails).
- `### T21 — integración de la caché` → creado por **T21**; rellenado por **T22** (que sí pudo tocar `app_nucleo.go`; el detalle vive en `docs/firmas/cache.md`).
- `### T22 — historial de visualización` → creado por **T22** (Persistencia del historial).
- `### T34 — búsqueda combinada + AnimeAV1 en la app` → creado por **T34**; sus cambios de firma también están anotados *in situ* en el bloque T20.
- `### T37 — portabilidad del núcleo` → creado por **T37** (núcleo en `internal/app`, directorio de datos por parámetro).

---

### T20 — app.go / app_nucleo.go

**Archivos:** `app.go` (delgado: métodos expuestos a Wails y ciclo de vida, nada más), `app_nucleo.go` (toda la orquestación, testeable con dobles), `app_test.go` (18 tests, sin red ni Wails), `app_integration_test.go` (smoke con red, `//go:build integration`). `main.go` se tocó **solo** para añadir `OnShutdown: app.shutdown` (autorizado por el humano).

#### Wails (`app.go`)

- `type App struct { mu sync.RWMutex; ctx context.Context; core *nucleo }` — `mu` protege `ctx`, que es **nil antes de `startup`**; `contexto()` devuelve `context.Background()` en ese caso (el nil nunca llega a los proveedores).
- `func NewApp() *App` — construye `nuevoNucleoPorDefecto()`; **no** toca la red ni escucha puertos.
- `func (a *App) startup(ctx context.Context)` — guarda el ctx, `slog.SetDefault(logx.New(os.Getenv("MIAPP_DEV") != ""))` (instala logx en `app.go`, pendiente de **T01**, y unifica el `slog` de jkanime, pendiente de **T06**) y `core.arrancar(emisor)` con `emisor = func(ev string, data any) { runtime.EventsEmit(ctx, ev, data) }`.
- `func (a *App) shutdown(_ context.Context)` — `core.cerrar()`; registrado como `OnShutdown` en `main.go`.
- `func (a *App) contexto() context.Context`
- **T29 eliminó `Greet`** (plantilla Wails, sin ningún consumidor en `frontend/src`); los bindings se regeneraron con `wails generate module` → ya no aparece en `frontend/wailsjs/go/main/App.{js,d.ts}`.
- `func (a *App) SearchAnime(query string) ([]models.Anime, error)` — **conservado** tal cual lo consume `App.svelte`; **no existe `Search`**. Desde **T34** su interior es la **búsqueda combinada** (véase `nucleo.buscar`): JKAnime primero y AnimeAV1 después, en paralelo.
- `func (a *App) GetAnimeDetails(provider string, id string) (DetalleResultado, error)`
- `func (a *App) GetVideoSources(provider string, episodeID string, noCache bool) ([]models.VideoSource, error)`
- Desde **T22** (véase su bloque): `SaveProgress(progress models.WatchProgress) error` y `GetProgress(animeID string) ([]models.WatchProgress, error)`.
- **Límite de Wails descubierto:** `binding.BoundMethod.Call` solo procesa 1 o 2 valores de retorno; con 3 devuelve `(nil, nil)` y **traga el error**. Por eso los métodos expuestos devuelven siempre `(valor, error)` y el detalle llega en un struct.

#### Núcleo (`app_nucleo.go`)

- `const proveedorPrincipal = "jkanime"`, `const proveedorComplemento = "animeav1"`, `const tiempoLimiteBusqueda = 20 * time.Second` — orden y límite de la **búsqueda combinada** (T34; antes eran `proveedorPrincipal` + `proveedorFallback = "animeflv"`, retirado en T34).
- `var ErrDetalleNoDisponible = errors.New("Detalle no disponible para este proveedor")` — traducción de `scraper.ErrNoDetails` (AnimeFLV, `HasDetails=false`). **Decisión:** no es un fallo grave → se devuelve como error tipado con mensaje listo para la UI, se loguea a nivel `info` y no se emite ningún evento. `errors.Is(err, scraper.ErrNoDetails)` **no** se conserva fuera del núcleo (Wails solo entrega el texto al frontend, y este es el útil).
- Dependencias inyectadas (interfaces en el propio archivo):
  - `type RegistroProviders interface { Get(name string) (scraper.Provider, bool); All() []scraper.Provider }`
  - `type ProxyLocal interface { Register(models.VideoSource) models.VideoSource; Escuchar() (string, error); Cerrar() error }` — lo cumple `*stream.Proxy`.
  - `type Resolutor func(fuentes []models.VideoSource) (models.VideoSource, bool)` — en producción `extractors.SelectBestSource` (T17).
  - `type Emisor func(evento string, data any)` — en producción `runtime.EventsEmit`; nil = no emitir (antes de `startup`).
  - **T21/T22:** además, dos valores concretos en el struct: `cacheLocal *cache.Cache` (caché de detalle+episodios) y `historial *db.DB` + `rutaHistorial string` (historial de visualización).
- `type DetalleResultado struct { Detail models.AnimeDetail \`json:"detail"\`; Episodes []models.Episode \`json:"episodes"\` }` — respuesta de `GetAnimeDetails`: detalle + primera tanda de episodios.
- `type EpisodiosCompletados struct { AnimeID string \`json:"anime_id"\`; Error string \`json:"error"\` }` — payload de `episodes:complete`; `Error` es texto en español (vacío = todo OK). Un `error` de Go **nunca** se serializa.
- **Firma actual (T21/T22):** `func nuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor, c *cache.Cache, rutaHistorial string) *nucleo` — una caché nil se sustituye por `cache.New(ttlCacheDetalle)`; `rutaHistorial` es el directorio del historial (`""` = el por defecto de `db`), los tests inyectan `t.TempDir()`.
- `func nuevoNucleoPorDefecto() *nucleo` — un único `httpx.NewClient(httpx.ClientOptions{})` compartido por `jkanime.New(doer, "", nil)` y **`animeav1.New(doer, "")`** (T34: se dejó de registrar `animeflv`; mismo jar; el jar separa por host, no se mezclan sesiones), `stream.NewProxy(nil)` (cliente propio del proxy con anti-SSRF y **sin cookies**; inyectar el doer de arriba haría perder el control de red, decisión T19), `extractors.SelectBestSource` (T17), `cache.New(ttlCacheDetalle)` (T21) e historial en el directorio por defecto de `db` (T22). El `ExceptionResolver` de JKAnime queda dentro de su `JkAnime` (una sola descarga del mapa de excepciones por proceso, T06); **no se precarga** en el arranque.
- `func (n *nucleo) arrancar(emis Emisor) error` — instala el emisor, llama `cacheLocal.Arrancar()`, **abre el historial** (`db.Open(rutaHistorial)`) y llama `proxy.Escuchar()` (sin eso, `Register` devuelve las fuentes sin registrar, T19). Si el historial **no** puede abrirse solo hay `slog.Warn` y la app sigue **sin historial** (no se devuelve ese error: el proxy y la reproducción no dependen de él).
- `func (n *nucleo) cerrar() error` — cancela el stream activo (borrando su entrada incompleta), `cacheLocal.Cerrar()`, **`historial.Close()`** y `proxy.Cerrar()`; idempotente. El puntero del historial **no se borra**, así que tras el shutdown un `SaveProgress`/`GetProgress` tardío recibe `db.ErrCerrado`. Se llama desde `shutdown`.
- `func (n *nucleo) buscar(ctx context.Context, query string) ([]models.Anime, error)` — **búsqueda combinada (T34, reemplaza al fallback de T20)**: recorre `[jkanime, animeav1]` **en paralelo** (una goroutine por proveedor registrado, `WaitGroup` y una casilla de respuesta por índice) con `context.WithTimeout(ctx, tiempoLimiteBusqueda)`; los resultados se concatenan **en el orden de `nombres`**, es decir JKAnime primero y AnimeAV1 después (cada `models.Anime` ya lleva su `provider`). Tolera que uno falle: su fallo se loguea con `slog.Warn` (sin URLs firmadas) y sigue habiendo respuesta con lo del resto; un proveedor **no registrado** no cuenta como fallo. Devuelve **lista vacía sin error** si al menos uno pudo buscar —incluso con resultados vacíos— y `errors.Join` de todos los fallos (`"no se pudo buscar %q: …"`) **solo si ninguno pudo**. Sin proveedores registrados → lista vacía sin error. El `ctx` cancelado del llamador se propaga (los `Search` lo ven).
- `func (n *nucleo) detalle(ctx context.Context, proveedor, id string) (DetalleResultado, error)` — **decisión T21 (reemplaza a la de T20)**: mira primero la caché local (clave `cache.Clave(proveedor, id)`):
  1. proveedor desconocido → `"proveedor desconocido: %q (disponibles: jkanime, animeav1)"` (la lista sale de `n.nombres()`, que enumera los **registrados**);
  2. **hit `Complete=true`** → devuelve todo ordenado por `Number`, suelta el stream de OTRO anime y emite `episodes:complete {anime_id, error:""}` **de inmediato**;
  3. **hit `Complete=false` con stream vivo** → devuelve lo acumulado y **se adjunta** al stream en curso: **NO** cancela ni lanza otro stream para el mismo anime (antes T20 cancelaba también al repetir el mismo anime);
  4. entrada incompleta **sin stream vivo** (huérfana) → se borra y se pide de nuevo;
  5. miss → `iniciarStream` (cancela solo el de OTRO anime) y, si el provider cumple `scraper.EpisodeStreamer`, `StreamAnimeDetails(ctx, lote(...))` sin cancelar su ctx (lo cancelará la próxima llamada o `cerrar`); si no, `GetAnimeDetails` síncrono, cancela su ctx al salir, **no emite eventos** y **no se cachea**;
  6. un stream que cierra con `Err` (o que devuelve error) **borra su entrada**: los errores no se cachean;
  7. `scraper.ErrNoDetails` → `ErrDetalleNoDisponible`.
- `func (n *nucleo) fuentes(ctx, proveedor, episodeID string, noCache bool) ([]models.VideoSource, error)` — `GetVideoSources` del provider → `SelectBestSource` (mejor **primero**) → `proxy.Register` **solo** para `Type == "hls" | "mp4"` (los iframes y los tipos vacíos quedan intactos) → reordena con la mejor en cabeza y el resto en orden original. **`noCache` se acepta y se reenvía al provider en `SourceOpts`, pero no lo consume nadie**: la caché local (T21) es de detalle+episodios y **nunca** cachea fuentes (regla 9), T17 lo ignora → **T27**.
- **Historial (T22):** `func (n *nucleo) guardarProgreso(progress models.WatchProgress) error` · `func (n *nucleo) verProgreso(animeID string) ([]models.WatchProgress, error)` — toman el almacén con `historialAbierto()` y delegan en `internal/db`; sin almacén abierto devuelven `ErrHistorialNoDisponible`.
- Internas: `proveedor`, `iniciarStream`, `soltarStreamDistinto`, `streamDe`, `liberarStream`, `borrarSiIncompleta`, `historialAbierto`, `lote`, `emitirCompleto`, `emisorActual`, `nombres`, `detalleDesdeCache`; libres `errorDeDetalle`, `indiceDe` (la mejor se localiza por URL+tipo; `Normalize` garantiza URLs únicas, T16).
- El núcleo **no crea goroutines** (salvo la limpieza de la caché, que `Arrancar`/`Cerrar` gestionan): el streaming lo hace el provider; aquí solo se pasa el `onBatch` y se cancela el ctx.

#### Eventos

| Evento | Payload | Cuándo |
|---|---|---|
| `episodes:update` | `models.EpisodeBatch` (`anime_id`, `batch`, `episodes`, `done`, `err`) | cada lote que entrega `onBatch`, **incluido** el final (`done=true`, suele venir sin episodios) |
| `episodes:complete` | `EpisodiosCompletados` = `{anime_id, error}` | cuando llega un lote con `Done`; `error` vacío si todo fue bien |

- Antes de **cada** emisión se comprueba `ctx.Err()` (y que haya emisor instalado): un stream cancelado —por pedir otro anime o por shutdown— no emite nada más, ni siquiera el lote final.
- No se emite nada cuando: aún no ha corrido `startup` (emisor nil), el provider no implementa `EpisodeStreamer`, o la llamada falla de forma síncrona (el error vuelve por el valor de retorno).
- **Ojo (para T26):** los eventos pueden llegar **antes** de que la promesa de `GetAnimeDetails` se resuelva, porque jkanime emite desde su goroutine: acumular por `anime_id` + `batch`, sin asumir orden.

#### Tests

- `app_test.go` — con dobles (`scraper.NewRegistry` real + providers/proxy/emisor falsos), sin red y sin Wails, en verde con `go test -race`:
  - búsqueda combinada (6, T34): orden JKAnime→AnimeAV1 aunque AnimeAV1 responda primero (y los dos se consultan siempre) · tolera el caído en las **dos** direcciones (subtests) · ambos caídos → error con los dos mensajes · `ctx` cancelado se propaga · sin resultados no es error · sin proveedores registrados no revienta;
  - detalle (6): proveedor desconocido con la lista de disponibles · `ErrNoDetails` → `ErrDetalleNoDisponible` (y no arrastra el sentinel de scraper) · sin `EpisodeStreamer` no emite · lotes + `episodes:complete` · fallo de página llega como `error` en `complete` · **cambio de anime cancela el stream anterior y sus lotes tardíos se suprimen**;
  - ciclo de vida (3): `shutdown` cancela el stream y cierra el proxy (y es idempotente) · sin emisor no falla · `arrancar` instala el emisor y hace `Escuchar`;
  - fuentes (3): solo HLS/MP4 pasan por el proxy con la mejor primero (el iframe queda intacto y sin registrar, **conservando su `Referer`**; las proxificadas salen **sin `Referer`/`Headers`**, T29, y la sesión del doble sí los recibe) · `noCache` se reenvía y no hay caché (dos llamadas = dos peticiones al provider) · proveedor desconocido.
- `app_integration_test.go` — `TestBusquedaRedReal` (`//go:build integration`): `nuevoNucleoPorDefecto()` + `buscar("bleach")` contra la red real; exige resultados y que el primero sea de `jkanime`. **T34 lo dejó pasando sin tocarlo** (su comentario interno todavía menciona el registro `jkanime + animeflv`: corregirlo en **T35**, que retira `animeflv`). Ejecutado en verde tras T34: **21 resultados**, ambos proveedores contestaron 200.

#### Decisiones y límites

1. **`main.go` editado con autorización explícita** (1 línea, `OnShutdown: app.shutdown`): reportado antes de tocar nada por la regla de escape, porque sin ese hook no hay parada del proxy ni cancelación de streams.
2. El emisor se instala **solo en `startup`** y captura ese ctx: `runtime.EventsEmit` con un ctx sin valores de Wails termina el proceso (`log.Fatalf` dentro de `getEvents`), así que emitir con `context.Background()` está prohibido.
3. El logger se decide con la variable `MIAPP_DEV` (Wails no expone su flag de debug desde fuera: `internal/app.IsDebug` es interno).
4. Si `Escuchar()` falla en `startup`, la app sigue viva: se loguea `slog.Warn` y las fuentes saldrían sin registrar (hoy sin aviso en la UI → **T27**).
5. `GetVideoSources` de AnimeFLV seguía devolviendo su error descriptivo mientras fue solo fallback de búsqueda; **desde T34 el paquete `animeflv` ni siquiera se registra**, así que esas rutas son inalcanzables (el paquete se retira en **T35** y `ErrDetalleNoDisponible` se conserva).

---

### T21 — integración de la caché (rellenado en T22)

`app_nucleo.go` cambió en T21 y el bloque T20 de arriba ya refleja esos cambios. Lo esencial (detalle completo en **`docs/firmas/cache.md`**):

- `nuevoNucleo` añadió el parámetro **`c *cache.Cache`** (nil ⇒ `cache.New(ttlCacheDetalle)`) y el campo `cacheLocal`; `nuevoNucleoPorDefecto` la crea con `ttlCacheDetalle = 10 * time.Minute`.
- **Regla de cancelación (reemplaza a la de T20):** `iniciarStream` **ya no cancela siempre**. Un hit completo devuelve y emite `episodes:complete` de inmediato (soltando el stream de OTRO anime); un hit incompleto con stream vivo **se adjunta** (no se cancela ni se relanza para el mismo anime); la cancelación es **solo al cambiar a otro anime** (o en `cerrar`). Entradas incompletas huérfanas se borran; las completas no.
- `arrancar` llama `cacheLocal.Arrancar()` y `cerrar` llama `cacheLocal.Cerrar()` (además de borrar la entrada incompleta del stream cancelado).
- `lote(ctx, st *streamActivo, id)` recibe el stream activo y marca `st.fallo` **antes** de borrar la entrada cuando el lote final trae `Err`; `detalle()` comprueba ese flag para no recrear una entrada borrada (bug encontrado por los tests de T21).
- `GetVideoSources` sigue **sin** caché (regla 9): `noCache` se acepta, se reenvía y se ignora → **T27**.
- Tests: `app_nucleo_cache_test.go` (6).

---

### T22 — historial de visualización (app.go / app_nucleo.go)

El paquete, su formato y sus tests están en **`docs/firmas/db.md`**. Aquí, lo que cambia en la app:

#### Wails (`app.go`)

- `func (a *App) SaveProgress(progress models.WatchProgress) error` — delega en `nucleo.guardarProgreso`. **Un único retorno** (Wails admite 1 o 2). El backend calcula `is_watched` (0.90) y escribe a disco de forma atómica: el reproductor lo llamará cada ~5 s (**T27**).
- `func (a *App) GetProgress(animeID string) ([]models.WatchProgress, error)` — delega en `nucleo.verProgreso`; **`animeID` vacío = historial entero** (p. ej. la lista "continuar viendo").
- Errores que puede recibir el frontend: `ErrHistorialNoDisponible` (antes de `arrancar` o si el almacén no pudo abrirse) y `db.ErrCerrado` (tras `shutdown`).

#### Núcleo (`app_nucleo.go`)

- **`rutaHistorial string`** en `type nucleo`: directorio del historial; `""` ⇒ `os.UserConfigDir()/miapp` (lo resuelve `db.Open`). `nuevoNucleo` lo recibe como **6º argumento** (antes eran 5); `nuevoNucleoPorDefecto` pasa `""`.
- **`historial *db.DB`**: se **abre en `arrancar`** y se **cierra en `cerrar`** (el puntero no se borra). Apertura fallida ⇒ solo `slog.Warn` (decisión documentada arriba, en `arrancar`): la app sigue sin historial y las llamadas devuelven `ErrHistorialNoDisponible`.
- `historialAbierto()` lee el puntero bajo `n.mu`; `guardarProgreso`/`verProgreso` delegan en `db`.
- `app.go` **no** abre ni cierra nada: solo delega (ciclo de vida = núcleo, como el proxy y la caché).

#### Tests y ayudas

- `app_test.go`: los helpers `nuevoEntorno`/`nuevoEntornoConCache` inyectan **`t.TempDir()`** como `rutaHistorial` (ningún test toca la configuración real) y exponen `entorno.dirHistorial`; `nuevoNucleo` cambió de firma en sus 3 call sites.
- `app_nucleo_historial_test.go` (3 tests, permitido por `app_nucleo_*_test.go`): abrir en `arrancar` + guardar/leer/cerrar y relectura desde disco · sin arrancar ⇒ `ErrHistorialNoDisponible` con su texto de UI · ruta inaccesible ⇒ `arrancar` funciona igual (proxy escuchando) y el historial queda nil.

---

### T34 — búsqueda combinada + AnimeAV1 en la app (app.go / app_nucleo.go / frontend)

**Archivos tocados:** `app.go` (solo doc de `SearchAnime`), `app_nucleo.go` (registro y `buscar`), `app_test.go` (tests de búsqueda), `app_nucleo_animeav1_test.go` y `app_nucleo_progreso_test.go` (nuevos), `frontend/src/**`, `frontend/wailsjs/go/models.ts` (**regenerado**, no editado a mano: solo añade `language?: string`), `PROGRESO.md`, `docs/firmas/app.md` y `docs/BITACORA.md`.

#### Núcleo (`app_nucleo.go`)

- **Registro**: `nuevoNucleoPorDefecto()` registra `jkanime` + **`animeav1`**; **`animeflv` ya no se registra** (el paquete sigue existiendo hasta T35). Las firmas expuestas a Wails **no cambian**.
- **Búsqueda combinada**: `buscar` (arriba) sustituye al fallback "solo si JKAnime falla o viene vacío". Constantes: `proveedorPrincipal = "jkanime"`, `proveedorComplemento = "animeav1"`, `tiempoLimiteBusqueda = 20 * time.Second` (cada `Search` va además con el timeout por defecto de `httpx`, 15 s).
- `detalle`/`fuentes` **no cambiaron**: siguen enrutando por `provider`, así que AnimeAV1 funciona de punta a punta sin código nuevo (los tests lo demuestran con dobles).

#### Aislamiento de progreso (contrato frontend ⇄ backend)

- `GetProgress(animeID)` devuelve **todos** los proveedores de ese slug (la clave es `anime_id` = slug) y **no se migra ni se reescribe** nada: la separación es una vista en el frontend.
- El criterio es el **prefijo del `episode_id`** (`<proveedor>:<slug>…`, sin `:` dentro del proveedor): `stores.esDeProveedor` / `episodiosDeProveedor` / `progressToMap(historial, proveedor)` filtran lotes, lista y `progressMap`; `openAnime` fija `currentProvider` junto a `currentAnimeID`.
- **Slug compartido**: como `anime_id` no basta, las comprobaciones de "sigue abierta la misma cosa" (`openAnime`, `refrescarProgreso` y el early-return de `VistaDetalle`) comparan **proveedor + slug**. Sin esto, abrir `jkanime:bleach` con `animeav1:bleach` ya cargado mostraba la ficha vieja (bug encontrado en la verificación visual de T34).
- `episodes:complete` solo lleva `anime_id` (modelo de Go, no tocable): se atribuye al slug. Es seguro porque el núcleo tiene **un único stream activo** (`iniciarStream` cancela el anterior y `lote`/`emitirCompleto` callan al cancelado), pero si algún día hubiera streams concurrentes habría que meter el proveedor en el payload.
- Test (Go, porque el frontend no tiene runner): `app_nucleo_progreso_test.go::TestProgresoDelMismoSlugSeAislaPorPrefijoDelEpisodeID`.

#### Frontend

- **Regla especial de AnimeFLV eliminada**: ya no existe `PROVEEDOR_SIN_DETALLE`/`aviso` en `VistaBusqueda` ni ninguna comprobación `provider === 'animeflv'`; toda tarjeta navega al detalle.
- **`frontend/src/etiquetas.ts`** (nuevo, puro): `etiquetaTipo` (`tv-anime`→`TV`, `series`→`Serie`, `movie`/`pelicula`→`Película`, `ova`, `ona`, `special`/`especial`, desconocidos tal cual), `etiquetaIdioma`, `etiquetaFuente` (`Servidor · Idioma · Calidad`, sin partes vacías) y `fuentesVisibles(fuentes, filtro)` (**`filtro` es `null` hoy**: sin filtro se ven todas, SUB y DUB; cambiar el filtro a `'SUB'`/`'DUB'` no toca nada más).
- **`SelectorFuente`**: título "Servidor · Idioma"; cada botón sale de `etiquetaFuente` (texto, nunca solo color) y emite **el índice de la lista completa** aunque se muestre filtrada.
- **`VistaReproductor`**: `elegirFuente` guarda el progreso **antes** de cambiar de índice → `alMetadatos` restaura la posición en la nueva fuente (mismo segundo). Límites documentados en el código: fuente `iframe` (sin metadatos) y episodio ya visto (arranca en 0). `filtroIdioma: FiltroIdioma = null` es el enganche para el futuro filtro.
- **`api.ts`**: `VideoSourceView.language` (`''` si el proveedor no lo declara) y doc de `searchAnime` actualizada.
- **`wailsjs`**: regenerado con `docker compose run --rm app wails generate module`; el diff es **solo** `language?: string` + su asignación en `VideoSource.create`.

#### Tests nuevos

- `app_test.go` → los 6 de búsqueda combinada (lista arriba).
- `app_nucleo_animeav1_test.go` (3): detalle+lotes+caché de T21 con doble `animeav1` (2ª llamada = hit, `StreamAnimeDetails` una sola vez) · cancelación del ctx corta los lotes siguientes sin `episodes:complete` · **`TestAnimeAV1CabecerasSoloVivenEnElProxy`** con **proxy real** + `httptest` como origen: el origen SÍ recibe `Sec-Fetch-Site: same-origin` a través del proxy, el JSON que cruza el puente de Wails no contiene esa cabecera (`Headers == nil`, `Referer == ""`) y `Language` sí sobrevive para la etiqueta.
- `frontend/src/etiquetas.test.ts` (20 comprobaciones: SUB, DUB, sin idioma, filtros y tipos) — se ejecuta con el esbuild de Vite, comando en la cabecera del archivo (no se pudo añadir script a `package.json`, fuera de "Puedes tocar").
- Verificación visual en `http://localhost:34115`: búsqueda de `bleach` con tarjetas de **los dos** proveedores, ficha `animeav1 · TV · Finalizado · 366 episodios` cargada por lotes, selector con `HLS · SUB` / `HLS · DUB` / `UPNShare · SUB`… y cambio manual de fuente funcional. La reproducción real (proxy en `127.0.0.1`) solo con el `.exe` → **T35**.

---

### T37 — portabilidad del núcleo (internal/app + app.go)

**Archivos:** `app.go` (raíz, capa Wails delgada), `internal/app/nucleo.go` (el núcleo, `package app`) y los tests en `internal/app/` (`app_test.go`, `app_e2e_test.go`, `app_e2e_integration_test.go`, `app_integration_test.go`, `app_nucleo_*_test.go`, `portabilidad_test.go`). `app_nucleo.go` **desapareció de la raíz** (se movió con `git mv`); los bloques de arriba (T20/T21/T22/T34) siguen describiendo el mismo código **antes** de ese movimiento y con los nombres en minúscula.

#### Núcleo (`internal/app`, package `app`)

- `type Nucleo struct` — era `nucleo`; mismos campos y mismo comportamiento.
- `func NuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor, c *cache.Cache, rutaHistorial string) *Nucleo` — era `nuevoNucleo`.
- `func NuevoNucleoPorDefecto(rutaHistorial string) *Nucleo` — era `nuevoNucleoPorDefecto()` sin parámetros: **ahora el directorio lo calcula el llamador** (P3, paso 4). El resto del montaje es el mismo (httpx compartido, jkanime+animeav1, `stream.NewProxy(nil)`, `SelectBestSource`, `cache.New(ttlCacheDetalle)`).
- Métodos exportados que consume `app.go`: `Arrancar(emis Emisor) error` · `Cerrar() error` · `Buscar(ctx context.Context, query string) ([]models.Anime, error)` · `Detalle(ctx, proveedor, id string) (DetalleResultado, error)` · `Fuentes(ctx, proveedor, episodeID string, noCache bool) ([]models.VideoSource, error)` · `GuardarProgreso(progress models.WatchProgress) error` · `VerProgreso(animeID string) ([]models.WatchProgress, error)`.
- Sigue sin exportar (los tests comparten paquete): `proveedor`, `iniciarStream`, `soltarStreamDistinto`, `streamDe`, `liberarStream`, `borrarSiIncompleta`, `historialAbierto`, `lote`, `emitirCompleto`, `emisorActual`, `nombres`, `detalleDesdeCache`, `errorDeDetalle`, `indiceDe`, `nuevoEntorno`, `streamActivo`, `ttlCacheDetalle`, `proveedorPrincipal/Complemento`.
- Exportados en este paquete (sin cambios): `DetalleResultado`, `EpisodiosCompletados`, `ErrDetalleNoDisponible`, `ErrHistorialNoDisponible`, `RegistroProviders`, `ProxyLocal`, `Resolutor`, `Emisor`.
- **Reglas fijadas:** no importa `github.com/wailsapp/*` (P2) y no llama a `os.UserConfigDir`/`os.UserHomeDir` (P3); la guardia `internal/app/portabilidad_test.go` lo verifica con `go/parser`.

#### Wails (`app.go`)

- `type App struct { mu sync.RWMutex; ctx context.Context; core *app.Nucleo }`.
- `func dirHistorial() string` — **único** `os.UserConfigDir` del código (P3): `filepath.Join(base, "miapp")`; si el sistema no lo da devuelve `""` y `Arrancar` se queda sin historial con el mismo `slog.Warn` de antes.
- `func NewApp() *App` — `&App{core: app.NuevoNucleoPorDefecto(dirHistorial())}`.
- `type DetalleResultado struct { Detail models.AnimeDetail \`json:"detail"\`; Episodes []models.Episode \`json:"episodes"\` }` — **duplicado a propósito en `package main`**, con los mismos campos y tags que `app.DetalleResultado`, para que Wails lo siga generando como `main.DetalleResultado` en `frontend/wailsjs/go/models.ts` (diff vacío en `wailsjs`, paso 5). `GetAnimeDetails` traduce de una a otra.
- El resto de métodos expuestos delegan tal cual: firmas **idénticas** a las de T22/T34 (arriba).
- `MIAPP_DEV` sigue siendo el único `os.Getenv` (P7, en `startup`).

#### `internal/db` (tocado por el paso 4)

- `func Open(dir string) (*DB, error)`: con `dir == ""` devuelve el nuevo **`ErrDirectorioVacio`**; se eliminó `rutaPorDefecto()` y con ella `os.UserConfigDir` del paquete. La ruta final en escritorio **no cambia**: `%AppData%\miapp\historial.json` la calcula ahora `app.go`.
- Test: `TestOpenConDirectorioVacioFalla` sustituye a `TestRutaPorDefectoEsConfigDelUsuarioMasMiapp` (13 tests en `db` antes y después).

#### Tests

- **302 tests de paquete antes → 303 después** (los mismos 302, más `TestNucleoPortableSinWailsNiRutasDeUsuario`): `miapp` 33 → `miapp/internal/app` 33+1; el resto de paquetes intactos (`db` 13, `extractors` 68…).
- Los E2E ahora llaman al núcleo (`n.Buscar`, `n.Detalle`, `n.Fuentes`, `n.GuardarProgreso`, `n.VerProgreso` con `context.Background()`) porque `&App{core: n}` ya no compila en `internal/app` (`App` es de `package main`); sus fixtures pasan a rutas relativas `../scraper/…` (el cwd del test ahora es `internal/app/`).
