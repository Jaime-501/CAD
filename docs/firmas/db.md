# Firmas reales — internal/db

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/db`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T22 — internal/db` → creado por **T22** (Persistencia del historial); corregido en **T38** (contrato T37 de `Open(dir)`: `dir == ""` ⇒ `ErrDirectorioVacio`, sin `rutaPorDefecto`).
- `### T22 — integración en el núcleo` → creado por **T22** (documentada también en `docs/firmas/app.md`); actualizado en **T38** (el núcleo vive desde T37 en `internal/app/nucleo.go`, `package app`).

---

### T22 — internal/db

Historial de visualización persistido en un archivo **JSON** dentro de un directorio **inyectable**. Sin dependencias nuevas (nada de sqlite/CGO) y sin goroutines: todo es mutex + E/S.

- `const UmbralIsWatched = 0.90` — umbral del criterio "visto": `CurrentTime/TotalDuration >= 0.90`. **Único punto donde se decide** (constante configurable en el código).
- `var ErrCerrado = errors.New("el historial de visualización está cerrado")` — para `errors.Is`; lo devuelven `SaveProgress`/`GetProgress` después de `Close`.
- `var ErrDirectorioVacio = errors.New("el historial necesita un directorio no vacío")` — lo que devuelve `Open` con `dir == ""` (símbolo nuevo en **T37**: este paquete **no** decide rutas — P3; la ruta por defecto la calcula el llamador).
- `func Open(dir string) (*DB, error)` — **entregable de T22** (contrato **T37**: el directorio lo elige quien llama). Crea el directorio si falta (`MkdirAll`). **`dir == ""` ⇒ `ErrDirectorioVacio`** (no hay directorio por defecto: la app production pasa `os.UserConfigDir()/miapp`, calculado en `app.go`; los tests inyectan `t.TempDir()`). Carga el archivo `historial.json` del directorio:
  - no existe (primera ejecución) o **vacío** ⇒ almacén vacío, **sin `.bak`**;
  - **no parseable como JSON** ⇒ se renombra a `historial.json.bak` (sustituye a un `.bak` previo) y se empieza **vacío sin devolver error** (un corrupto nunca hace fallar el arranque); si el renombrado falla, solo `slog.Warn`;
  - error de E/S real (permisos, ruta inaccesible) ⇒ sí devuelve error.
  - Los registros cargados pasan por el mismo normalizado que al guardar (`is_watched` al día).
- `func (d *DB) SaveProgress(progress models.WatchProgress) error` — guarda o **actualiza** un episodio y lo persiste **ahora mismo** (write-through, para el llamado de ~5 s del reproductor):
  - exige `AnimeID` y `EpisodeID` no vacíos;
  - **`IsWatched` lo calcula el backend** (se ignora el que traiga el llamador);
  - registro idéntico al guardado ⇒ **no reescribe** el archivo;
  - si la escritura a disco falla, el dato **queda en memoria** (el próximo guardado reintenta) y se devuelve el error;
  - con `cerrado` ⇒ `ErrCerrado`.
- `func (d *DB) GetProgress(animeID string) ([]models.WatchProgress, error)` — registros de un anime **en orden de primera guardada** (una guardada posterior actualiza en su sitio, sin moverlo). **`animeID == ""` ⇒ historial entero.** Devuelve **copias** (mutarlo no toca el almacén). Sin registros ⇒ lista vacía, no nil y sin error.
- `func (d *DB) Close() error` — marca el almacén como cerrado (**idempotente**). No hay búfer que volcar —cada `SaveProgress` ya escribió a disco—, así que solo evita guardados tardíos tras el shutdown. No borra ni toca el archivo.

**Formato y concurrencia:**

- Archivo: `<dir>/historial.json`, **array plano** de `models.WatchProgress` (tags snake_case: `anime_id`, `episode_id`, `current_time`, `total_duration`, `is_watched`), compacto. Clave de upsert = `(anime_id, episode_id)` (`type clave struct{anime, episodio string}`); **no hay borrado**.
- Escritura **atómica**: `os.CreateTemp` **en el mismo directorio** (para que el rename sea del mismo sistema de ficheros) → `Write` → `Sync` → `Close` → `os.Rename` sobre el definitivo; si algo falla se elimina el temporal y el archivo anterior queda intacto. Los tests comprueban que no quedan `*.tmp`.
- Un **único `sync.Mutex`** cubre memoria + marshal + escritura: dos guardadas nunca se entrelazan y `GetProgress` espera como mucho a esa escritura (JSON pequeño).
- **`IsWatched`**: `esVisto(current, total)` = `total > 0 && current/total >= UmbralIsWatched`. `TotalDuration <= 0` ⇒ **nunca** visto. `NaN`/`±Inf` (el `<video>` los produce; JSON no los admite) se sanean a `0` antes de guardar.
- **No hay `time.Sleep`, goroutines ni reloj**: los tests usan `t.TempDir()` y cierres explícitos.

**Tests** (`internal/db/db_test.go`, 13 tests con subtests, sin red; logs descartados con `TestMain`): guardar/recibir con orden y filtros · actualizar sin duplicar ni mover · copia devuelta · sin identificadores falla · umbral de `IsWatched` (7 subtests: justo en el umbral, por debajo, completado, sin empezar, `TotalDuration` 0/negativa, `CurrentTime` negativo) · `esVisto` con el umbral (`Nextafter`) · `NaN`/`Inf` · **corrupto ⇒ `.bak` con la basura + historial nuevo** · inexistente/vacío sin `.bak` · **persistencia tras reabrir** · **escrituras concurrentes** (8 goroutines × 25 guardadas + 4 peleando por el mismo episodio + lectoras, y relectura desde disco) · `Close` idempotente y bloqueante · `Open` con `dir == ""` ⇒ `ErrDirectorioVacio` (`TestOpenConDirectorioVacioFalla`, desde T37).

---

### T22 — integración en el núcleo

(Actualizado en **T38**: desde **T37** el núcleo vive en `internal/app/nucleo.go` (`package app`); los nombres exportados de abajo son los actuales. El detalle de los métodos Wails está en `docs/firmas/app.md`, cuyo bloque **T37** manda para el estado actual del núcleo.)

- `app.ErrHistorialNoDisponible = errors.New("El historial de visualización no está disponible")` — error tipado con mensaje para la UI (mismo criterio que `ErrDetalleNoDisponible`). Aparece cuando **no hay almacén abierto**: antes de `Arrancar`, o si su apertura falló.
- `type Nucleo` gana dos campos: **`rutaHistorial string`** (directorio inyectado por el llamador; **vacío no es un valor especial** desde T37: `db.Open` devuelve `ErrDirectorioVacio`) y **`historial *db.DB`** (nil hasta abrirse).
- `func NuevoNucleo(reg RegistroProviders, px ProxyLocal, sel Resolutor, emis Emisor, c *cache.Cache, rutaHistorial string) *Nucleo` — el **6º argumento** es el directorio (los tests pasan `t.TempDir()`); `NuevoNucleoPorDefecto(rutaHistorial)` monta el resto con las dependencias reales y recibe el mismo directorio (`app.go` calcula `os.UserConfigDir()/miapp`).
- `func (n *Nucleo) Arrancar(emis Emisor) error` — además de emisor, caché y proxy, **abre el historial**. Si `db.Open` falla **NO se devuelve error**: solo `slog.Warn("la app sigue sin historial de visualización")` y la app continúa sin historial (el proxy y la reproducción no dependen de él).
- `func (n *Nucleo) Cerrar() error` — además de lo de T21, **cierra el historial** (idempotente). **No borra el puntero**: tras el shutdown, `GuardarProgreso`/`VerProgreso` devuelven `db.ErrCerrado` (no hay carrera con un `Arrancar` posterior).
- `func (n *Nucleo) GuardarProgreso(progress models.WatchProgress) error` / `func (n *Nucleo) VerProgreso(animeID string) ([]models.WatchProgress, error)` — toman el almacén con `historialAbierto()` (mutex) y delegan; sin almacén ⇒ `ErrHistorialNoDisponible`.
- `func (n *Nucleo) historialAbierto() *db.DB` — lectura del puntero bajo `n.mu`.
- `app.go` solo delega: **`func (a *App) SaveProgress(progress models.WatchProgress) error`** (un único retorno: `binding.BoundMethod.Call` admite 1 o 2) y **`func (a *App) GetProgress(animeID string) ([]models.WatchProgress, error)`**.

**Tests** (`internal/app/app_nucleo_historial_test.go`, 3 tests, sin red): abrir en `Arrancar` + guardar/leer + cerrar y relectura desde disco con otro `db.Open` · sin arrancar ⇒ `ErrHistorialNoDisponible` con su mensaje de UI · ruta inaccesible ⇒ `Arrancar` sigue funcionando (proxy escucha) y el historial queda nil.
