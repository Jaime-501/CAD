# PROGRESO — Comfort Anime Desktop (T01–T58 hechas)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivos históricos**: el estado de **T01–T29 + T30** (v1.0) está en **`docs/PROGRESO_v1.0.md`**, el de **T30–T35** (AnimeAV1, v1.1.0, con sus desviaciones por tarea, fixtures y pruebas manuales) en **`docs/PROGRESO_v1.1.md`** y el de **T39–T55** (v1.2.0) en **`docs/PROGRESO_v1.2.md`**. Aquí solo queda lo vigente. Estado del producto: **v1.2.0 PUBLICADA** — ciclo **T39–T57** (incluye T53b y T57); la **prueba manual con el `.exe`** y la **release** las hizo el humano; el cierre documental post-publicación quedó etiquetado **T58** (bloque al final). **Colisión de numeración (T58):** el archivo `tareas/T58.md` define una tarea distinta —dos bugs visuales de la v1.2.0— que es la fila **T58** de la tabla y el bloque "T58 — bugs visuales" al final; el cierre documental conserva su bloque propio titulado "Cierre documental post-v1.2.0 (etiquetado T58)".

## Protocolo de sesión

1. Leer `AGENTS.md`, `tareas/00_REGLAS.md`, este archivo, `docs/firmas/<paquetes de los que depende la tarea>` y el archivo de la tarea (`tareas/Txx.md`).
2. Ejecutar **solo** la tarea indicada. Respetar "Puedes tocar" / "NO debes tocar".
3. Correr la comprobación de la tarea. Después, el humano la repite por su cuenta (`gofmt -l .`, `go vet ./...`, `go test -race ./...` y `npm run check` en frontend): no confiar solo en el reporte del agente.
4. Actualizar este archivo: marcar la tarea, anotar desviaciones y pendientes; las firmas nuevas van en `docs/firmas/<paquete>.md` (formato `### Txx — <paquete>`, ej. `docs/firmas/animeav1.md`).
5. Detenerse. No empezar la siguiente tarea.

> **Aviso de entorno (T36):** en este repositorio no hay `.gitattributes` y `core.autocrlf=true`: un merge/checkout hecho **desde Windows** puede dejar los `.go` tocados con fin de línea **CRLF** en el worktree, y entonces `gofmt -l .` (en el contenedor Linux) los marca a todos aunque el contenido esté bien formateado. La solución es normalizarlos a LF (quitar solo el `\r` final; git no registra ningún cambio) — decisión del humano en T36. `git status` puede seguir mostrándolos como "M" con diff vacío: es el fantasma de autocrlf (mismo caso que `frontend/wailsjs/go/models.ts`).

## Prompt base por sesión

```text
Lee AGENTS.md, tareas/00_REGLAS.md, PROGRESO.md, tareas/<Txx>.md y
docs/firmas/<paquetes de los que depende la tarea>.
Ejecuta SOLO la tarea <Txx>. Respeta "Puedes tocar" y "NO debes tocar".
Si necesitas tocar un archivo fuera de la lista, detente y repórtalo.
Al terminar: corre la comprobación, actualiza PROGRESO.md (estado, desviaciones,
pendientes) y docs/firmas/<paquete>.md si creaste firmas, y detente.
No empieces la siguiente tarea.
```

**Qué firmas leer:** solo las de los paquetes de los que depende tu tarea (`docs/firmas/<paquete>.md`; la lista por paquete está en `tareas/00_REGLAS.md` → "Documentación y estado"). La tabla por tarea de T31–T35 quedó archivada en `docs/PROGRESO_v1.1.md`.

## Prerrequisitos: fixtures reales (humano o agente con curl)

Antes de las tareas que parsean HTML/JSON real debe existir un **fixture real y sanitizado** (sin cookies ni tokens) en el `testdata/` correspondiente. Un fixture es la respuesta **cruda** del servidor guardada en un archivo, para que los tests corran sin red. Lo puede capturar el humano a mano o el agente con `curl.exe` (ver "Captura de fixtures" más abajo). Si no existe el fixture de la tarea, el agente se detiene y lo pide: **no inventa HTML ni selectores**.

- Los fixtures ya capturados están inventariados en los archivos históricos: JKAnime (v1.0) en `docs/PROGRESO_v1.0.md`, AnimeAV1 (T30–T33, 20 fixtures) en `docs/PROGRESO_v1.1.md` y el Home de ambos proveedores (T39) en `docs/PROGRESO_v1.2.md`.

### Captura de fixtures

**Regla de oro:** el fixture debe ser lo que recibirá goquery, es decir, el HTML **tal como lo envía el servidor**, no el DOM ya modificado por JavaScript. goquery no ejecuta JS: un dato que solo aparece tras cargar scripts no existe para el scraper en Go.

Ojo con las herramientas de Playwright: `browser_snapshot` devuelve el árbol de accesibilidad, **no** HTML, y no sirve como fixture. Los nombres exactos de las herramientas dependen de la versión de tu servidor MCP; usa las equivalentes de navegar, evaluar JS y listar peticiones de red.

Procedimiento, en orden:

1. **Explorar:** navegar a la URL con Playwright y, con la lista de peticiones de red, identificar la petición exacta (URL, método, cabeceras y cuerpo necesarios, sin copiar valores secretos).
2. **Obtener la respuesta cruda**, por orden de preferencia:
   - Con `curl` o un programa Go corto desde Bash, guardando directo a archivo (`-o testdata/archivo`). Es lo preferible porque no pasa por el contexto y reproduce lo que hará el scraper. **Así se capturaron los fixtures de AnimeAV1 (T30).**
   - Si el sitio bloquea clientes HTTP (Cloudflare) pero el navegador pasa: dentro de la página, `fetch(url).then(r => r.text())` mediante evaluación de JS. Devuelve la respuesta del servidor sin modificar por scripts.
3. **Comprobar que no es un bloqueo:** si el archivo contiene "Just a moment" o similar, no es un fixture válido. Anotarlo en "Pendientes activos" (abajo).
4. **Guardar en archivo, no pegar en el chat.** Inspeccionar con `grep` o fragmentos de pocas líneas, nunca leyendo la página completa.
5. **Recortar solo si es enorme:** conservar el `<head>` con lo necesario y el bloque relevante con 2 o 3 elementos de ejemplo. No alterar la estructura ni los nombres de clases.
6. **Sanitizar:** reemplazar cookies, tokens CSRF reales y cualquier dato de sesión por valores falsos como `TOKEN_DE_PRUEBA`.
7. **Verificar que el dato está en el archivo crudo** (`grep`). Si solo aparece en el DOM renderizado, goquery no puede extraerlo: detenerse, reportarlo y buscar el endpoint que entrega ese dato.
8. **Validar con goquery:** el test de la tarea carga el fixture y comprueba los selectores. Ese test es la prueba de que la captura sirve.

Para cuidar el contexto (modelos gratuitos):

- `animeav1.com` **no tiene challenge ni rate-limit observado** (T30): con `curl.exe` + UA de `httpx` basta; usa el MCP de Playwright solo si un sitio bloquea al cliente HTTP.
- No pedir capturas ni lecturas de página completa en el chat; solo resultados acotados.
- Si hay un montón de peticiones de red, filtrar por la URL que interesa en lugar de listarlas todas.

## Estado de tareas

Leyenda: `[ ]` pendiente · `[~]` en curso · `[x]` hecha y verificada

| Estado | Tarea | Título | Depende de |
|---|---|---|---|
| [x] | T01 | Logging y observabilidad base | — |
| [x] | T02 | Infraestructura HTTP, errores y reintentos | T01 |
| [x] | T03 | Modelos de dominio | — |
| [x] | T04 | Interfaz Provider y registro | T03 |
| [x] | T05 | Referencia de episodio y películas | T03 |
| [x] | T06 | JKAnime — excepciones | T02 |
| [x] | T07 | JKAnime — búsqueda | T02–T06 |
| [x] | T08 | JKAnime — detalle y metadatos | T07 |
| [x] | T09 | JKAnime — episodios AJAX con streaming | T08 |
| [x] | T10 | JKAnime — fetch de página de episodio | T02 |
| [x] | T11 | Arquitectura de extractores | T03, T10 |
| [x] | T12 | Extractor JKPlayer (softsubs) | T11 |
| [x] | T13 | Extractor JKPHP | T11 |
| [x] | T14 | Extractor `var servers` | T11 |
| [x] | T15 | Resolver MediaFire | T11 |
| [x] | T16 | Normalizador de VideoSource | T12–T15 |
| [x] | T17 | Selector de mejor fuente + GetVideoSources | T16 |
| [x] | T18 | AnimeFLV — búsqueda con fallback (histórico: retirado en T35) | T02, T04 |
| [x] | T19 | Proxy local de streaming | T02, T03 |
| [x] | T20 | Integración app.go y eventos Wails | T09, T17, T19 |
| [x] | T21 | Caché local | T20 |
| [x] | T22 | Persistencia del historial | T03 |
| [x] | T23 | Regenerar bindings Wails | T20–T22 |
| [x] | T24 | Estado Svelte | T23 |
| [x] | T25 | UI de búsqueda y navegación | T23 |
| [x] | T26 | Vista de detalle y episodios | T24, T25 |
| [x] | T27 | Reproductor multi-recurso | T19, T24 |
| [x] | T28 | E2E (offline + real) | T17, T20 |
| [x] | T29 | Ajustes finales y refactor | todas |
| [x] | T30 | Investigación AnimeAV1 (informe + fixtures) | — |
| [x] | T31 | AnimeAV1: datos de SvelteKit y búsqueda | T30 |
| [x] | T32 | AnimeAV1: detalle y episodios | T31 |
| [x] | T33 | AnimeAV1: fuentes de vídeo | T32 |
| [x] | T34 | Integrar AnimeAV1 en la app (registro, búsqueda combinada, frontend) | T33 |
| [x] | T35 | E2E de AnimeAV1, retirada de AnimeFLV y release v1.1.0 | T34 |
| [x] | T36 | Cierre de la v1.1.0 y limpieza de documentación y comentarios | T35 |
| [x] | T37 | Preparación para Android (refactor sin cambio de comportamiento) 🔒 | T36 |
| [x] | T38 | Cierre documental de T37 | T37 |
| [x] | T39 | Investigación del Home (informe + fixtures) | — |
| [x] | T40 | `models`: progreso ampliado, Favorite, HomeSection, ResumeInfo, NextEpisode | — |
| [x] | T41 | `db`: historial con fecha, listado reciente y borrados 🔒 | T40 |
| [x] | T42 | `db`: favoritos 🔒 | T40 |
| [x] | T43 | `HomeProvider` + Home de JKAnime | T39, T40 |
| [x] | T44 | Home de AnimeAV1 | T39, T43 |
| [x] | T45 | `Nucleo` + `app.go`: Home, favoritos, borrados 🔒 | T41–T44 |
| [x] | T46 | `Nucleo.Retomar` y `SiguienteEpisodio` 🔒 | T45 |
| [x] | T47 | Regenerar bindings + `api.ts` | T46 |
| [x] | T48 | UI: shell con barra lateral | T47 |
| [x] | T49 | UI: barra lateral con historial por proveedor | T48 |
| [x] | T50 | UI: retomar con un clic | T49 |
| [x] | T51 | UI: borrado individual (✕, deslizar, Deshacer) | T50 |
| [x] | T52 | UI: borrar por proveedor y borrar todo | T51 |
| [x] | T53 | UI: Home "Descubrir" | T48 |
| [x] | T53b | UI: pulido visual del Home | T53 |
| [x] | T54 | UI: favoritos | T53 |
| [x] | T55 | UI: botón "Siguiente ▸" | T46, T50 |
| [x] | T56 | Cierre v1.2.0: título CAD, docs, firmas, prueba manual, release | todas |
| [x] | T57 | UI: botón "Siguiente ▸" discreto durante la reproducción | T55 |
| [x] | T58 | Carátula perdida al retomar y contador de favoritos (ver colisión de numeración en el encabezado) | T57 |

T01–T29 = v1.0 (`docs/PROGRESO_v1.0.md`) · T30–T35 = v1.1.0 AnimeAV1 (`docs/PROGRESO_v1.1.md`) · T36 = cierre de la v1.1.0 · T37 = portabilidad Android · T38 = cierre documental de T37.
T39–T57 = v1.2.0 (Home, historial lateral, favoritos, "Siguiente" y su estado discreto; **incluye T53b y T57**); plan en `tareas/v1.2_PLAN.md`, detalle T39–T55 archivado en `docs/PROGRESO_v1.2.md`, cierres T56/T57, el cierre documental (etiquetado T58) y la T58 de bugs visuales al final de este archivo.
La **v1.2.0 está PUBLICADA**: la **prueba manual de la v1.2.0 con el `.exe`** (checklist al final) y la **release** las hizo el humano.

## Auditoría de portabilidad (T37)

> Paso 1 de T37, hecha **antes de mover nada**. Lo que contradice la ficha o queda fuera de "Puedes tocar" se reporta aquí (regla 7).

**(a) `os.UserConfigDir` / `UserHomeDir` / `Getenv` / `runtime.GOOS` / `exec` / cgo / build tags de SO** (`internal/**` + raíz):

- **`os.UserConfigDir`: `internal/db/db.go:107`** (función `rutaPorDefecto()`, llamada por `Open("")` en `db.go:83`), más sus comentarios (`db.go:3, 73, 105`) y el test `internal/db/db_test.go:555` (`TestRutaPorDefectoEsConfigDelUsuarioMasMiapp`). En la raíz **solo comentarios** (`app_nucleo.go:129, 177`). → **contradice P3**; la ficha lo prevé en el paso 4 ("si el cálculo está dentro de `internal/db`, puedes tocarlo solo para recibir la ruta por parámetro"): `app.go` pasa la ruta y `db.Open` deja de calcularla. Ver decisión 3 abajo.
- `os.UserHomeDir`: **ninguno**.
- `Getenv`: **`app.go:46`** (`MIAPP_DEV`); **ninguno en `internal/**`**.
- `runtime.GOOS` / `runtime.GOARCH`: **ninguno**.
- `os/exec`: **ninguno**.
- cgo (`import "C"`, `#cgo`): **ninguno**.
- Build tags de SO: **ninguno**. Los únicos `//go:build` del repositorio son `integration` (7 archivos de test): `app_integration_test.go`, `app_e2e_integration_test.go`, `internal/scraper/jkanime/{search,detail,episodes,get_video_sources}_integration_test.go` y `internal/scraper/animeav1/animeav1_integration_test.go`.

**(b) imports de `github.com/wailsapp/*` fuera de `app.go`/`main.go`: ninguno.** Solo `main.go` (`wails`, `pkg/options`, `pkg/options/assetserver`) y `app.go` (`pkg/runtime`). `internal/**` está limpio → P2 ya se cumple.

**(c) archivos de `frontend/src` que importan `wailsjs`: 6**

| archivo | qué importa |
|---|---|
| `api.ts:8-15` | funciones de `go/main/App.js` + tipos `main`, `models` |
| `stores.ts:19-20` | **`EventsOn`/`EventsOff`** de `runtime/runtime.js` + tipos `models` |
| `VistaBusqueda.svelte:4`, `VistaDetalle.svelte:4`, `TarjetaEpisodio.svelte:2`, `VistaReproductor.svelte:5` | solo `import type { models }` |

`VistaReproductor.svelte` ya importa además `VideoSourceView` de `./api` (correcto). → es el trabajo del paso 2.

**(d) `app.go` frente a `app_nucleo.go`:**

- **`app.go`** (118 líneas, `package main`): `App{mu, ctx, core *nucleo}`, `NewApp()`, `startup` (guarda el ctx de Wails, `slog.SetDefault(logx.New(MIAPP_DEV))`, emisor `func(ev, data) { runtime.EventsEmit(ctx, …) }`, `core.arrancar(emisor)`), `shutdown` (`core.cerrar()`), `contexto()` y los **5 métodos Wails** (`SearchAnime`, `GetAnimeDetails`, `GetVideoSources`, `SaveProgress`, `GetProgress`), que solo delegan en el núcleo. Es el único archivo con `os.Getenv` y con `wailsapp` (junto con `main.go`).
- **`app_nucleo.go`** (698 líneas, `package main`): todo el orquestador `nucleo` (registro `jkanime`+`animeav1`, búsqueda combinada, caché, streams de detalle, proxy, historial, eventos), **sin ningún import de Wails**. Exportados hoy: `Emisor`, `RegistroProviders`, `ProxyLocal`, `Resolutor`, `DetalleResultado`, `EpisodiosCompletados`, `ErrDetalleNoDisponible`, `ErrHistorialNoDisponible`; el resto (`nucleo`, `nuevoNucleo`, `nuevoNucleoPorDefecto`, métodos) es interno.
- **Tests**: 8 archivos `app*_test.go` → **37 `func Test`** (3 con tag `integration`: los 2 del E2E real y el smoke de búsqueda; más `TestMain`, que no es test) = **33 efectivos en `miapp`**. Todos en `package main` y referenciando `nuevoNucleo…`/`&App{core: n}`.
- **`main.go`** (37 líneas): `//go:embed` + `wails.Run` con `OnStartup`/`OnShutdown`/`Bind`.

**Tensiones detectadas (regla 7, reportadas antes de tocar):**

1. **`DetalleResultado` y los bindings**: si el tipo se mueve a `internal/app`, Wails regeneraría el namespace TS `main` → `app` y `git diff frontend/wailsjs` dejaría de estar vacío (comprobación de la ficha). **Resolución**: el núcleo conserva su `DetalleResultado` (paquete `app`, mismos campos/tags) y `app.go` define el suyo y traduce (`DetalleResultado{Detail, Episodes}`): 2 structs idénticos, JSON y bindings idénticos.
2. **Los E2E construyen `&App{core: n}`** (tipo de `package main`): al moverse a `internal/app` llamarán al núcleo directamente (`n.Buscar(ctx, …)`, `n.Detalle(…)`, … con `context.Background()`, que es exactamente lo que hace `App.contexto()` sin `startup`). Mismas aserciones, mismo número de tests.
3. **`db.Open("")` pasará a devolver error** (P3 exige que el cálculo salga de `internal/db`); el test `TestRutaPorDefectoEsConfigDelUsuarioMasMiapp` se reescribe para cubrir el nuevo contrato (**13 tests en `internal/db` antes y después**, 14 `func Test` contando `TestMain`). La ruta de escritorio **no cambia**: `app.go` calcula `os.UserConfigDir()/miapp` y la pasa por parámetro → `%AppData%\miapp\historial.json`, sin migración. Único efecto observable: en el (prácticamente imposible) caso de que `os.UserConfigDir()` falle, el `slog.Warn` de `arrancar` lleva el mensaje de `db` en vez del de `rutaPorDefecto` — la app sigue igualmente sin historial.
4. **`DESARROLLO.md` no existe en la raíz**: el documento vive en **`docs/DESARROLLO.md`**; lo trato como el que nombra la ficha (paso 7).
5. **`docs/firmas/db.md` y `docs/firmas/cache.md`** quedan con referencias a `app_nucleo.go`, `nuevoNucleo` y al `Open("")` por defecto: **no están en "Puedes tocar"** → no se tocan; quedan como desviación pendiente (ver "Pendientes activos").

**Estado base (antes de tocar nada):**

- `gofmt -l .` → **`app.go`, `app_nucleo.go`** (el fantasma CRLF del aviso de T36: esos dos archivos están con `\r\n` en el worktree y el contenido está bien formateado; los demás `.go` son LF). Se normalizan a LF al tocarlos.
- `go vet ./...` y `go vet -tags integration ./...` → limpios · `go build ./...` → OK.
- `go test -race -count=1 ./...` → **11 paquetes `ok`, 302 tests**: `miapp` 33 · cache 9 · db 13 · httpx 27 · logx 7 · models 14 · scraper 7 · animeav1 59 · jkanime 42 · extractors 68 · stream 23. E2E offline (JKAnime y AnimeAV1) en verde.

---

## T37 — ejecución y comprobación real

**Qué se hizo (pasos 2–6):**

- **Frontend (paso 2):** `api.ts` es el único archivo con `wailsjs`: añade `EventsOn`/`EventsOff`, reexporta los tipos (`export type { main, models }`), movió `parsePayload` desde `stores.ts` y expone las suscripciones **`onEpisodesUpdate(cb)` / `onEpisodesComplete(cb)`**, que devuelven la función que cancela con `EventsOff(evento)`. `stores.ts` importa de `./api` y sus `cancelarUpdate`/`cancelarComplete` las llaman en `disposeEvents` (mismo comportamiento: quitar el evento quita todos sus listeners). Los 4 `.svelte` pasan `import type { models }` a `'./api'`.
- **Mover el núcleo (paso 3):** `git mv` de `app_nucleo.go → internal/app/nucleo.go` y de los 8 `app*_test.go → internal/app/` (el historial de git se conserva: `git status` los pinta como `R`). `package app`; se exportó lo mínimo: `Nucleo`, `NuevoNucleo`, `NuevoNucleoPorDefecto(dir)` y los métodos `Arrancar/Cerrar/Buscar/Detalle/Fuentes/GuardarProgreso/VerProgreso` (todo lo demás sigue sin exportar). Los E2E sustituyen `&App{core: n}` por llamadas directas al núcleo con `context.Background()` y sus fixtures pasan a rutas relativas `../scraper/…` (el cwd del test ahora es `internal/app/`).
- **Directorio de datos (paso 4):** `app.go` añade `dirHistorial()` (único `os.UserConfigDir` del código) y se lo pasa a `NuevoNucleoPorDefecto`; en `internal/db` se eliminó `rutaPorDefecto()` y `Open("")` devuelve el nuevo **`db.ErrDirectorioVacio`**. Ruta final idéntica: `%AppData%\miapp\historial.json`.
- **`app.go` delgado (paso 5):** solo traduce; define su `DetalleResultado` idéntico al de `internal/app` (misma traducción de un struct al otro en `GetAnimeDetails`) para que Wails lo siga generando en el namespace `main`. **`main.go` sin cambios.**
- **Guardia (paso 6):** `internal/app/portabilidad_test.go` barre `internal/**` con `go/parser` y falla con `github.com/wailsapp/*` (P2) o `os.UserConfigDir`/`UserHomeDir` (P3). Verificado en ambos sentidos: en verde con el código actual, en rojo con un `internal/tmpcheck` temporal que importaba `wailsapp` y llamaba a `os.UserConfigDir`.

**Comprobaciones de la ficha (resultado real):**

| comprobación | resultado |
|---|---|
| `gofmt -l .` | **sin salida** (`app.go` e `internal/app/nucleo.go` normalizados a LF) |
| `go vet ./...` · `go vet -tags integration ./...` | **limpios** |
| `go build ./...` | **OK** |
| `go test -race -count=1 ./...` | **11 paquetes `ok`**; E2E offline JKAnime y AnimeAV1 **en verde** |
| tests antes → después | **302 → 303** (`miapp` 33 → `internal/app` 33+1 = 34; `db` 13 y el resto intactos; los mismos 302, más la guardia) |
| `go list -deps ./internal/app \| grep wailsapp` | **vacío** |
| `grep -rn "wailsjs" frontend/src` | **solo `api.ts`** |
| `wails generate module` y `wails build` → `git diff --stat frontend/wailsjs` | **vacío** (bindings idénticos) |
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (el aviso de chunk >500 kB es preexistente) |
| `GOOS=android GOARCH=arm64 go vet ./internal/...` (informativo) | **sin salida**: `internal/**` compila limpio para android/arm64 |
| `wails build -platform windows/amd64` | **OK** → `build/bin/miapp.exe` (13,2 MB) |

**Desviaciones nuevas de esta ejecución (regla 7 / fuera de "Puedes tocar"):**

- **`DESARROLLO.md` no existe en la raíz** (decisión 4): se editó **`docs/DESARROLLO.md`**, el documento que nombra la ficha.
- **`docs/firmas/db.md` y `docs/firmas/cache.md` quedan desfasados** (decisión 5): siguen mencionando `app_nucleo.go`, `nuevoNucleo…` y `Open("")` con el directorio por defecto. **No están en "Puedes tocar" → no se han tocado**; hay que actualizarlos en una tarea posterior (ver "Pendientes activos").
- **Rutas de fixtures de los E2E**: `internal/scraper/…/testdata/…` → `../scraper/…/testdata/…` (consecuencia directa de mover el cwd de los tests; mismo archivo, misma ruta real).
- **`NuevoNucleoPorDefecto()` ahora recibe el directorio**: los 3 call sites de test pasan `t.TempDir()` (antes el E2E real escribía el historial en un temporal **después** de construir el núcleo; ahora lo recibe, misma efectividad).
- **Normalización LF** de `app.go`/`internal/app/nucleo.go` (el CRLF fantasma que ya señaló T36): `gofmt -l .` queda limpio, pero el diff de esos dos archivos muestra las líneas completas.
- **`db.ErrDirectorioVacio`** es un símbolo nuevo exportado (no es de la ficha, pero la firma `Open("")` lo exige para no devolver un error anónimo).

---

## T38 — cierre documental de T37

Solo `.md`, sin tocar código. Contraste con `go doc ./internal/db`, `go doc ./internal/cache` y `go doc -all ./internal/app`:

- **`docs/firmas/db.md`**: `Open(dir)` ya **no** tiene directorio por defecto — con `dir == ""` devuelve **`db.ErrDirectorioVacio`** (símbolo nuevo de T37); añadido `ErrDirectorioVacio` a las firmas del paquete y corregido el test de "ruta por defecto" → `TestOpenConDirectorioVacioFalla` (13 tests, igual que antes). Bloque de integración reescrito al núcleo actual: `internal/app/nucleo.go` (`package app`), `type Nucleo`, `NuevoNucleo(reg, px, sel, emis, c, rutaHistorial)`, `NuevoNucleoPorDefecto(rutaHistorial)`, métodos exportados `Arrancar`/`Cerrar`/`GuardarProgreso`/`VerProgreso` y tests en `internal/app/`.
- **`docs/firmas/cache.md`**: el bloque de paquete **ya coincidía** con `go doc ./internal/cache` (sin cambios); corregidas las referencias al núcleo (título, nota de bloques, ruta `app_nucleo.go` → `internal/app/nucleo.go`, `nuevoNucleo` → `NuevoNucleo` con el 6º argumento de T22, nombres exportados de los métodos y ruta de los tests). `nuevoEntornoConCache` es helper de test, no firma: se conserva tal cual.
- **`README.md`, `AGENTS.md`, `docs/DESARROLLO.md`, `tareas/00_REGLAS.md`**: 0 referencias obsoletas a `app_nucleo.go` ni al núcleo en `package main` (los "package main" que aparecen son el `DetalleResultado` de `app.go` y la regla P1, ambos vigentes). **No se tocaron.**
- **`docs/firmas/app.md`**: **fuera de "Puedes tocar"**; sus bloques T20–T34 ya llevan la nota de que el bloque **T37** manda, así que no desinforman. No se tocó.
- **Identificadores en minúscula en "Desviaciones…"**: `nucleo.fuentes`→`Nucleo.Fuentes`, `nuevoNucleoPorDefecto()`→`NuevoNucleoPorDefecto(dir)`, `nucleo.buscar`→`Nucleo.Buscar` (discrepancia real con el código, en archivo tocable).
- Sin desviaciones de alcance: ningún archivo tocado fuera de la lista.

**Reporte (regla 15, NO tocado por estar fuera del alcance de T38):** `tareas/00_REGLAS.md` → el diagrama de layout sigue listando **`animeflv/ # paquete animeflv (T18)`** (línea 29) y la regla de dependencias no menciona `scraper/animeav1`; el paquete se retiró en T35. Cambio propuesto: quitar esa línea del árbol y añadir `animeav1` a la dependencia `scraper → internal/app`. Lo dejo para una tarea con `00_REGLAS.md` en "Puedes tocar" con ese objetivo explícito.


## Desviaciones que afectan a tareas futuras (resumen)

> Solo lo que sigue vigente; el texto completo (por tarea) está en `docs/PROGRESO_v1.0.md` (T01–T29), `docs/PROGRESO_v1.1.md` (T30–T35) y `docs/PROGRESO_v1.2.md` (T39–T55).

- **Modelos (T03)**: `ResourceType` en **minúsculas** (`hls`, `mp4`, `iframe`); `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""`. Cualquier fuente nueva debe emitir esos strings.
- **Normalización y selección (T16/T17)**: `Normalize` corre **dentro** de `Resolver.Resolve` (dedup por recurso sin query pero con fragmento; preserva `Referer`/`Headers`/`Subtitles`) y `SelectBestSource` jerarquiza **HLS > MP4 > iframe > `Type` vacío**, desempate por `Quality` y gana la primera. Consumidor real: `Nucleo.Fuentes` (mejor primero; nombre desde T37).
- **Proxy (T19/T20/T29)**: `NewProxy(nil)` con anti-SSRF, `Register` **solo** con `Type` `hls|mp4` y **sin** `Referer`/`Headers` hacia el frontend (viven en la sesión); `stream.esManifiesto` detecta el manifiesto por el **cuerpo**, no por la extensión. `Sec-Fetch-Site: same-origin` (HLS de AnimeAV1) se declara en `VideoSource.Headers` y `stream/reenvio.go` la inyecta — **nunca** debe limpiarse `Headers` en `Register` (403 en los segmentos).
- **Registro y búsqueda (T20/T34/T35)**: `NuevoNucleoPorDefecto(dir)` registra **`jkanime` + `animeav1`** y `Nucleo.Buscar` los consulta **en paralelo** (timeout 20 s, salida JKAnime primero; error solo si fallan todos; nombres desde T37). `animeflv` **ya no existe** (retirado en T35); `scraper.ErrNoDetails` → `ErrDetalleNoDisponible` se conserva como flujo genérico. El string `"animeflv"` en algunos tests es un provider falso arbitrario, sigue siendo válido.
- **Caché (T21)**: detalle/episodios en memoria con TTL de **10 min**; `GetVideoSources` **nunca** se cachea (regla 9) y `noCache` solo fuerza repetir la extracción.
- **AnimeAV1 (T31–T33)**: vía principal **`GET <ruta>/__data.json`** (devalue: `PageData` + `Unflatten`), `models.Anime.ID` = slug, **lotes de 50** (`tamLote`), dedup de fuentes por **recurso base + idioma** (SUB y DUB son fuentes distintas), HLS derivado **sin petición** (`/play/<32hex>` → `/m3u8/<32hex>`, `/play/*` nunca se pide) con `Type: "hls"` y sus `Headers`; los embeds ajenos (UPNShare/Voe/Byse/MP4Upload) salen **iframe sin resolver**.
- **Progreso (T34)**: `anime_id` es slug compartido entre proveedores → `episode_id` lleva prefijo `<proveedor>:` y toda comprobación de "sigue abierto" compara **proveedor + slug** (no solo el slug).
- **E2E (T28/T35)**: los E2E exigen **peticiones exactas** al sitio falso y cortan hosts ajenos; el de AnimeAV1 demuestra que el proxy inyecta `Sec-Fetch-Site` (segmento por proxy → 206, directo → 403). Si cambia el flujo de peticiones, hay que actualizar el E2E a la vez.
- **Modelos (T40)**: `WatchProgress` lleva ahora 5 campos más (`provider`, `anime_title`, `cover_url`, `episode_number` int, `updated_at` Unix s) **sin `omitempty`** → `historial.json` nuevo siempre serializa las 10 claves (los archivos viejos siguen cargando con los nuevos en cero); T41/T42 rellenan y consumen esos campos, y los modelos TS de `WatchProgress` cambian al regenerar los bindings (T47).
- **Historial (T41)**: `SaveProgress` rellena `UpdatedAt` (`now().Unix()` solo si viene 0; si trae fecha se conserva) y `Provider` (con `ProviderName()`), pero solo al guardar —los archivos viejos se leen tal cual—. `LastPerAnime()` es el listado de la barra lateral (un registro por proveedor+slug; nunca nil; **cerrado ⇒ vacío sin error**) y `DeleteAnime`/`DeleteProvider`/`Clear` devuelven `(n, error)` con `ErrCerrado` si el almacén está cerrado; 0 borrados **no** reescribe el archivo. `GetProgress` y el formato del JSON no cambian.
- **Favoritos (T42)**: `db.OpenFavorites(dir)` (`dir == ""` ⇒ el mismo `ErrDirectorioVacio`) abre `*db.Favorites` sobre `favoritos.json` (JSON indentado, escritura atómica, corrupto ⇒ `.bak`); **no tiene `Close`**. `Add` es idempotente (misma clave ⇒ `false`, conserva `AddedAt`, actualiza `Title`/`CoverURL`; `AddedAt == 0` se sella con `now().Unix()`), `Remove` no reescribe si no existía, `List` nunca nil (copias, `AddedAt` desc y `Key()` asc); clave = `Provider + ":" + AnimeID`. T45 lo abre en `Arrancar` junto al historial.
- **Home JKAnime (T43)**: `scraper.HomeProvider` (interfaz opcional, `Registry`/`Provider` intactos) la implementa `jkanime.Home` con `emision` (JSON `var animes`, clave `data`) + `recientes` (SSR); el `ExceptionResolver` se aplica a **ambas** secciones (1 sola descarga lazy del mapa por proceso, nunca por item) y el límite de lectura es `maxSearchBytes`. `models.Anime` **no tiene `Type`** → los items salen sin tipo. T44 la implementa para `animeav1`; T45 la detecta con aserción de tipo y cachea `home:<proveedor>` 10 min.
- **Home AnimeAV1 (T44)**: `animeav1` cumple `scraper.HomeProvider` en `home.go` con `emision` (`GET /catalogo/__data.json?status=emision`, **guarda `filters.status == 2`** — un slug de status ya no reconocido devuelve todo el catálogo sin error —) y `recientes` (`GET /__data.json`, `latestMedia` con `id` numérico convertido sin notación científica); dedup por slug, 20 items máx. por sección, pausa 300 ms, fallos parciales/total y ctx como en T43 (mensajes "el Home de animeav1 …"). T45 la detecta con la misma aserción de tipo que la de JKAnime y cachea `home:<proveedor>` 10 min.
- **Retomar (T46)**: `Nucleo.Retomar`/`Nucleo.SiguienteEpisodio` implementados en `internal/app/retomar.go`; la lista completa de episodios se obtiene con `Detalle` + espera sobre la caché (sin leer internos del stream), nunca una lista parcial. T47 (bindings + `api.ts`) y T55 ("Siguiente ▸") dependen de que T46 se cierre (ya cerrada). Detalle y (a)/(b) verificados: nota `### T46` arriba (la puerta 0 se conserva en una línea en esa misma nota).
- **Bindings (T47)**: `frontend/wailsjs/**` regenerado → `models.WatchProgress` en TS pasó de 5 a **10 campos obligatorios** (T40 sin `omitempty`): todo literal de `WatchProgress` en el frontend debe llevarlos (el de `VistaReproductor.svelte:301` se completó en T47 con excepción autorizada; `cover_url`/`episode_number` provisionales → T49); `api.ts` ya expone los 11 métodos nuevos (`getHomeSection`, `addFavorite`, `removeFavorite`, `listFavorites`, `isFavorite`, `getRecentProgress`, `deleteProgressAnime`, `deleteProviderProgress`, `clearProgress`, `resumeAnime`, `getNextEpisode`) y la constante `RESUME_ACTION`.
- **Historial lateral (T49)**: `frontend/src/historial.ts` es el módulo de la lista de la barra (`$historial`/`$historialEstado`, `cargarHistorial`, agrupación por proveedor y helpers de formato) y `BarraLateral.svelte` ya acepta la prop `onOpen` (clic = T50); `TarjetaHistorial.svelte` pinta `resolviendo`/`mensaje` (T50/T51). El guardado del reproductor rellena `cover_url` desde `currentDetail` y `episode_number` parseando el `episode_id`; la lista se refresca al montar `App` y al **salir** del reproductor (no al entrar ni en otros cambios de vista).
- **Borrado individual (T51)**: `historial.ts` exporta además `borrarAnime(provider, animeID)` y `deshacerBorrado(registros)` (el deshacer re-guarda con `updated_at ≠ 0` → recupera el orden); el toast es **de una sola ranura** (`toasts.ts`/`Toast.svelte`, montado en `App.svelte`): un aviso nuevo sustituye al anterior. `TarjetaHistorial.svelte` tiene la prop `onBorrar` y su raíz ya es un **envoltorio `div`** (acción `deslizar.ts` + botón ✕ hermano del botón de retomar, que por eso no puede ser la raíz); el capture del puntero es **perezoso** (al decidirse el gesto horizontal) porque capturarlo en `pointerdown` rompe el `click` de los botones en Chromium.
- **Botón "Siguiente ▸" (T55)**: cambiar de episodio **dentro del reproductor** ya no remonta el componente: `navegarA('reproductor', sel)` con la vista activa cambia `seleccion` y el watcher local `sincronizarEpisodio` de `VistaReproductor.svelte` reinicia fuentes y pide el siguiente (`getNextEpisode`); el episodio abierto con el botón arranca en 0 aunque tenga progreso guardado (`empezarDesdeCero` en `alMetadatos`, un solo intento). El guardado (throttle 5 s, disparadores) no cambió: con el navegador no hay posición (`currentTime = 0`) y no se guarda — probar el caso con el `.exe` en T56.

## Pendientes activos

> Solo lo que una tarea futura debe resolver. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`; los de T30–T35 en `docs/PROGRESO_v1.1.md`.

- **Heredados de la v1.0** (detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos"): ~~(1) título de la ventana~~ → **cerrado en T56** (`main.go` → `CAD · Comfort Anime Desktop`; el ejecutable sigue siendo `miapp.exe`); ~~(2) marca de tiempo en `WatchProgress`~~ → **cerrado** (`updated_at` desde T40/T41: la barra lateral ordena por recencia; el botón "Continuar" del detalle conserva su criterio de mayor número con progreso y así queda documentado en `docs/DESARROLLO.md` → "Limitaciones conocidas"); ~~(3) borrado de registros del historial~~ → **cerrado en T51–T52** (individual con Deshacer, por proveedor y total). **Sigue activo (4)**: verificación visual de los avisos de error que ningún test provoca (hoy: el aviso `episodesError` con un fallo real del backend).
- ~~**Prueba manual de la v1.2.0 con el `.exe`** (checklist en "Cierre de la v1.2.0 (T56)" abajo)~~ → **cerrada (T58)**: la prueba manual con el `.exe` y la release las hizo el humano; la v1.2.0 está **publicada**. Cierra de paso los pendientes de la nota T57 (aclara al pausar, al 90 % y estado `reproduciendo` con vídeo real).
- **Verificación con el `.exe` de los fixes T58** (carátula al retomar y contador de favoritos al arrancar): en el flujo navegador no hay vídeo real (proxy local, AGENTS.md), así que el guardado con posición no se ejercitó; los fixes se verificaron por lógica + log y end-to-end del contador (ver bloque "T58 — bugs visuales"). Probar con el `.exe` en la próxima release (v1.2.1 o posterior).
- **Cerrados en T38** (cierre documental de T37, ver sección "T38" arriba): ~~firmas desfasadas de **`docs/firmas/db.md`** y **`docs/firmas/cache.md`**~~ → **corregidas**: `db.Open(dir)` con `ErrDirectorioVacio` (sin `rutaPorDefecto` ni `os.UserConfigDir` en el paquete) y el núcleo documentado como `internal/app` con `NuevoNucleo`/`NuevoNucleoPorDefecto(dir)`. Las referencias a `app_nucleo.go` que siguen en `docs/BITACORA.md`, `docs/PROGRESO_v1.0.md`/`v1.1.md`, `docs/investigacion/animeav1.md` y en los bloques T20–T34 de `docs/firmas/app.md` **son históricas o ya llevan la nota del bloque T37**: no se tocan.
- **Cerrados en T36** (documentados, sin acción pendiente): ~~comentarios de `app.go`/`app_nucleo.go` que presentaban a AnimeFLV como proveedor registrado o fallback~~ → **corregidos** (solo comentarios; ahora describen `jkanime` + `animeav1` con búsqueda combinada en paralelo). ~~Reproducir el HLS de AnimeAV1 end-to-end~~ → **cerrado**: E2E offline + real en verde y **prueba manual con el `.exe` en Windows correcta** (HLS SUB/DUB, SUB/DUB e iframes, búsqueda combinada, mismo slug en ambos proveedores, progreso, JKAnime intacto, sin consola); **release v1.1.0 publicada**.

---

## Cierre de la v1.2.0 (T56)

Hecha el **2026-10-12** en `feature/home-historial-favoritos`, **sin crear tag ni publicar** (lo hace el humano; ver "Release" abajo). Solo se tocaron archivos de la lista "Puedes tocar" de `tareas/T56.md`:

- **`main.go`**: únicamente el `Title` → `"CAD · Comfort Anime Desktop"` (`git diff main.go` = 1 línea). El ejecutable (`miapp.exe`), el módulo y `%AppData%\miapp` **no cambian**.
- **`wails.json`**: únicamente `info.productVersion` → `"1.2.0"`.
- **`tareas/00_REGLAS.md`** (reporte pendiente de T38, cerrado aquí): fuera del árbol `animeflv/` (retirado en T35), dentro `animeav1/`, `extractors/` comentado y `db/` = "historial y favoritos (JSON en disco) (T22, T42)"; diagrama de dependencias reescrito con `scraper/animeav1` y `extractors` en la rama de `httpx` y `internal/app` importando scraper, jkanime, animeav1, extractors, cache, db, httpx, models y stream.
- **`docs/firmas/*.md`**: contrastadas con `go doc -all` (`models` T40, `db` T41/T42, `scraper` T43, `jkanime` T43, `animeav1` T44) → **coincidían**; solo difirió **`docs/firmas/app.md`** (bloque T45): `AgregarFavorito` hace `TrimSpace`, valida campos no vacíos **y proveedor registrado** (vía `n.proveedor`), orden argumentos → proveedor → `ErrFavoritosNoDisponible` → `db.Add`.
- **`README.md`**: funciones de la v1.2 (Home con pestañas Descubrir/Favoritos, historial lateral con retomar, borrado con Deshacer, favoritos, "Siguiente ▸") y nombre **CAD** en la introducción; "Tus datos" incluye `favoritos.json`; fuera la frase obsoleta de "no hay borrado de entradas individuales del historial".
- **`docs/DESARROLLO.md`**: `favoritos.json` y `updated_at` en "Datos que guarda la app"; `retomar.go` y `animeav1` en la estructura de paquetes; caché de las secciones del Home (10 min, `home:<proveedor>`); bloque de bindings regenerados en T47; **"Limitaciones conocidas" reescrita** (los puntos "«Continuar» sin marca de tiempo" y "no hay borrado de registros" ya no aplican; queda documentado que el botón "Continuar" del detalle sigue eligiendo por número); "Trabajo con agentes" con T39–T56.
- **`AGENTS.md`**: estado T1–T56 / v1.2.0, `docs/PROGRESO_v1.2.md` en la lista de historiales y `Title` descrito como `CAD · Comfort Anime Desktop`.
- **`docs/PROGRESO_v1.2.md`** (nuevo): notas **T39–T55** movidas de aquí (246 líneas, texto literal).
- **`docs/BITACORA.md`**: fila de sesión de T56 y las desviaciones del ciclo v1.2 (caché `home:<proveedor>`, capture perezoso del puntero, `TarjetaAnime` extraída de `FilaAnimes`).
- **`PROGRESO.md`** (este archivo): T56 `[x]`, heredados (1)(2)(3) cerrados y este bloque con la compuerta, el checklist manual y la release.

### Compuerta (paso 8) — 2026-10-12, todo en verde

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** (ningún archivo por CRLF) |
| `go vet ./...` | limpio |
| `go vet -tags integration ./...` | limpio |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (`miapp` sin tests de paquete); **407 `--- PASS`** — mismo recuento que antes de empezar |
| `go test -tags integration ./... -run "E2E\|Home" -v` | PASS: `TestE2EFlujoCompletoRedReal` (15,75 s), `TestE2EAnimeAV1RedReal` (2,69 s), `TestE2EFlujoCompletoOffline` (0,66 s), `TestE2EAnimeAV1FlujoCompletoOffline` (0,34 s), `TestHomeRedReal` de animeav1 (1,16 s) y de jkanime (1,33 s) más los `TestHome*` de `internal/app`/`models`/`scraper`; **0 FAIL** |
| `npm run check` | **svelte-check found 0 errors and 0 warnings** |
| `npm run build` | OK (2,29 s; aviso preexistente de chunk >500 kB) |
| `wails build -platform windows/amd64` | OK → `build/bin/miapp.exe` (13,34 s) |

Recuento real por paquete (`go test -count=1 -v`, líneas `--- PASS`): `internal/app` **75** · `cache` **9** · `db` **43** · `httpx` **27** · `logx` **7** · `models` **24** · `scraper` **8** · `scraper/animeav1` **70** · `scraper/jkanime` **53** · `scraper/jkanime/extractors` **68** · `stream` **23** · `miapp` 0 → **total 407**.

`git status --porcelain` solo con los archivos listados arriba; `git diff main.go` muestra únicamente la línea del `Title`.
---

## T57 — UI: botón "Siguiente ▸" discreto durante la reproducción

### Paso 0 (antes de tocar código) — 2026-10-12

- **Cómo se sabe hoy si el vídeo reproduce o está en pausa** (`frontend/src/VistaReproductor.svelte`): **no hay estado booleano guardado**; la única vía son los handlers del `<video>` registrados en `onMount` (líneas 84-89) y retirados en `onDestroy` (106-112): `timeupdate` → **`alAvanzar`**, `pause` → **`alPausar`**, `ended` → **`alTerminar`** (más `loadedmetadata` → `alMetadatos`). La fuente de verdad de "en pausa" es la propiedad **`paused` del elemento `videoEl`** (`bind:this`, línea 589). No existe ningún listener `play`/`playing` hoy, y Svelte no reactiva `videoEl.paused` por sí mismo.
- **De dónde sale el cálculo `currentTime / duration ≥ 0.90`** (lo que activa `resaltado`): la función **`marcarResaltadoSiTermina`** (líneas 233-238), llamada **solo** desde `alAvanzar` (línea 460, `timeupdate`) y `alTerminar` (línea 467, `ended`). Guardas: `resaltadoSiguiente` ya marcado o `modo !== 'video'` → sale; `!(v.duration > 0)` → sale; con `v.currentTime / v.duration >= 0.9` fija **`resaltadoSiguiente = true`** (pegajoso; se limpia en `reiniciarEpisodio`, línea 174).
- **Vía reutilizada (sin listeners paralelos)**: estado nuevo **`reproduciendo`** que se actualiza **dentro de esos mismos handlers ya montados** — `alAvanzar` → `!v.paused` (un `timeupdate` con `paused` a `true` es un *seek* en pausa, no reproducción), `alPausar` → `false`, `alTerminar` → `false`. Cero listeners nuevos, cero cambios en reproducción, guardado, fuentes, proxy, navegación, `getNextEpisode` ni la tecla `N`.
- **Conclusión: NO se detiene** (regla 7 no aplica): hay vía para saber "reproduciendo" reutilizando la existente.

### Resultado

- **`frontend/src/BotonSiguiente.svelte`**: prop `discreto = false` y clase aplicada con la guardia **`discreto && estado === 'listo'`** (la ficha: aplica solo a `listo`; `resaltado` nunca es discreto, `cargando`/`oculto` sin cambios). CSS nuevo **solo** condicionado a `.discreto`: `opacity: 0.4` + `filter: blur(2px)` + `transition: opacity 200ms, filter 200ms`; se aclara a `opacity: 1` + `filter: none` con `:hover:not(:disabled)`, `:focus-visible` y `:focus-within`; bloque `@media (prefers-reduced-motion: reduce)` con `transition: none`. Sin colores nuevos (solo tokens `--cad-*` ya usados), sin cambios de posición/tamaño, sin `pointer-events`/`visibility`/`display`.
- **`frontend/src/VistaReproductor.svelte`** (solo para calcular y pasar la prop): estado `reproduciendo` seteado **dentro de `alAvanzar` (`!v.paused`), `alPausar` y `alTerminar`** (cero listeners nuevos), `$: discretoSiguiente = calcularDiscreto(modo, estadoSiguiente, reproduciendo, resaltadoSiguiente)` y `discreto={discretoSiguiente}` en el `<BotonSiguiente>`. **Sin tocar**: cuándo/cada cuánto se guarda el progreso, fuentes, proxy, navegación, `getNextEpisode` ni la tecla `N`.
- **Comprobaciones estáticas (Docker, estado final)**: `docker compose run --rm app sh -c "cd frontend && npm run check"` → **svelte-check found 0 errors and 0 warnings**; `… npm run build` → **✓ built in 2.31s** (`index-Deyavszo.js` 687.27 kB, hash idéntico al previo a las ediciones temporales; aviso preexistente de chunk >500 kB). Go sin tocar (tarea de UI).
- **Verificación con Playwright MCP en `http://localhost:34115`** (sin vídeo real; la prop se forzó editando la línea del template a `discreto={true}`/`discreto={false}` y **se revirtió sin commitear**; con bleach/jkanime Ep. 1–3 y la película Bleach Movie 4):
  1. **`discreto=true`** — medido primero con el **cálculo real** (fuente iframe mega.nz: la tabla "iframe ⇒ `true` siempre" funcionó sin forzar) y luego con la prop forzada: `opacity: 0.4`, `filter: blur(2px)`, `transition: opacity 0.2s, filter 0.2s` (`transitionDuration: 0.2s, 0.2s`).
  2. **Hover y foco con Tab** — hover → `opacity: 1`, `filter: none`; Tab (3.ª parada: ← Atrás → iframe → botón) → `:focus-visible: true`, `opacity: 1`, `filter: none`.
  3. **`discreto=false` forzado** (con el cálculo real en `true` por iframe → la prop manda): sin clase `discreto`, `opacity: 1`, `filter: none`, `transitionProperty: all` con **`transitionDuration: 0s`** (default de CSS = sin transición; aspecto idéntico al de T55).
  4. **`prefers-reduced-motion: reduce` emulado** → `transition: none`, `transitionDuration: 0s` (el aspecto discreto 0.4/blur(2px) se mantiene); sin emular → `0.2s`.
  5. **Clic y foco estando discreto** — el botón recibe Tab y el clic navega: cabecera pasa a "Ep. 2 · Bleach - 2" con botón "Siguiente ▸ Ep 3" (repetido también durante la prueba del retardo).
  6. **`cargando` y `oculto` sin cambios** — con `GetNextEpisode` parcheado a **+4 s** en la sesión (patrón de T53/T55): "Siguiente ▸" con `disabled: true` y `aria-busy="true"` **~4 s** (última muestra disabled a los **3650 ms**) **sin clase `discreto`** pese a `discreto={true}` forzado, con `opacity 0.55`/`filter none` (aspecto T55); a los **4061 ms** pasa a "Siguiente ▸ Ep 3" con clase `discreto` y la transición en curso (`opacity 0.810428`). `oculto` con la prop forzada `true` (película, Episodios (1)): **0 botones "Siguiente"** en el DOM.
  - Errores de consola: **solo preexistentes/ajenos** — 16 × `ERR_CONNECTION_REFUSED` al proxy local `127.0.0.1:41157/proxy` (limitación conocida del flujo navegador), 2 × `TypeError … 'nodes'` de `wails/ipc.js` (1 por recarga) y 1 × 404 `favicon.ico`. **0 errores nuevos**.
- **Archivos tocados**: `frontend/src/BotonSiguiente.svelte` (prop `discreto` + guardia de estado + CSS discreto) · `frontend/src/VistaReproductor.svelte` (estado `reproduciendo` en los 3 handlers, `calcularDiscreto`, `discretoSiguiente` y la prop en el `<BotonSiguiente>`) · `PROGRESO.md` (fila T57 + este bloque). `api.ts`, `stores.ts`, `frontend/wailsjs/**`, `*.go` y el resto del frontend **sin tocar**.
- **Desviaciones**: ningún archivo tocado fuera de "Puedes tocar" (regla 7 no se activó). (1) **Las forzuras `discreto={true}`/`discreto={false}` fueron solo de sesión**: se revirtieron; `git diff` final muestra el cálculo real (`discreto={discretoSiguiente}`) y `git status --porcelain` solo los 3 archivos de la lista. (2) **La transición vive en la regla `.discreto`** (donde la sitúa la ficha): al entrar en discreto y al aclarar por hover/foco transiciona 200 ms; al pasar `discreto` a `false` la regla desaparece y el cambio es instantáneo (resultado final igual: `opacity 1`/`filter none`). (3) **El primer parche de sesión con `getNextEpisode` (camelCase) no surtió efecto** (el binding real es **`GetNextEpisode`**); el estado `cargando` quedó observado con el nombre correcto y, además, la primera muestra (~100 ms) de la recarga inicial lo capturó igualmente. El parche se restauró al terminar. (4) `reproduciendo` se actualiza en `timeupdate` con `!v.paused` (un `timeupdate` de *seek* en pausa no cuenta como reproducción); el primer `timeupdate` tras pulsar reproducir llega en <250 ms.
- **Pendientes (para el humano con el `.exe`, vídeo real)**: se aclara **al pausar**, **al 90 %** (resaltado de T55 sin desenfoque) y el estado `reproduciendo` durante un **HLS/MP4 real** (en este flujo el proxy local no es alcanzable, AGENTS.md) — ninguno de esos tres casos se declara probado. `frontend/wailsjs/**` sin regenerar (no hizo falta). El `Type` de `models.Anime` sigue sin existir (pendiente de producto). → **Cerrados en T58**: la prueba manual con el `.exe` y la release las hizo el humano (v1.2.0 publicada).

---

## Cierre documental post-v1.2.0 (etiquetado T58 en su momento)

> **Nota (colisión de numeración, T58):** esta tarea se documentó con el número T58 al cerrar la v1.2.0; después el humano cargó `tareas/T58.md` con otra tarea (bugs visuales de la v1.2.0) que es la fila **T58** de la tabla y el bloque siguiente. Ambos bloques se conservan; ninguno de los dos desinforma.

Solo `.md`, sin tocar código (la regla 7 no se activó: nada fuera de la lista hizo falta). Nombres de métodos verificados con `go doc -all ./internal/app` y leyendo `app.go` (la lista de los 16 métodos Wails y los exportados del núcleo coinciden con el código; **manda el código**):

- **`PROGRESO.md`** (este archivo): encabezado y estado → **T01–T57 hechas**; **T39–T57 = v1.2.0** (con T53b y T57 dentro del ciclo); v1.2.0 **PUBLICADA** (prueba manual con el `.exe` y release hechas por el humano); la prueba manual sale de "Pendientes activos" (cerrada, y con ella los pendientes de vídeo real de la nota T57); fila **T58 `[x]`**.
- **`AGENTS.md`**: estado **T1–T57** / v1.2.0 publicada; en "Arquitectura (Go)", la lista de métodos Wails de `app.go` corregida a **16** (los 5 originales + los 11 de la v1.2, con sus firmas) y los métodos exportados del núcleo completados (`Arrancar/Cerrar/Buscar/Detalle/Fuentes/GuardarProgreso/VerProgreso` más `Home`, favoritos, borrados, `Retomar` y `SiguienteEpisodio`).
- **`docs/BITACORA.md`**: filas de sesión de **T53b**, **T57** y **T58**, y la desviación de **T57** (`reproduciendo` en los handlers existentes del `<video>`; botón discreto también en iframe → pendiente del port a Android por la regla **P6**).
- **`tareas/00_REGLAS.md`**: regla 6 → "salvo en las tareas de regeneración de bindings (T23, T47)".
- **`docs/DESARROLLO.md`**: "Trabajo con agentes" llevado hasta **T57** (incluye T53b).
- **`README.md`**: texto de usuario actualizado (Descubrir por proveedor con Reintentar, favoritos, "Siguiente ▸" discreto, "Cómo se usa" con barra lateral y Deshacer, filas nuevas de "Problemas frecuentes"); **`docs/img/Inicio.png` añadida como primera imagen** (el archivo **existe**; nota: en el worktree estaba sin trackear, `??`, preexistente a la sesión).
- **`docs/firmas/app.md`**: estaba en la lista de tocables pero **no necesitó cambios** (los bloques T37/T45/T46 ya describen los métodos; la lista resumida y verificada vive en `AGENTS.md`).
- **Archivos no tocados pese a estar desfasados** (fuera de la lista, regla 7): el pie de `docs/PROGRESO_v1.2.md` aún dice "prueba manual pendiente del humano" — es un archivo histórico de solo lectura; el estado vigente es el de este `PROGRESO.md`.
- **Comprobación**: `git status --porcelain` solo con los `.md` de "Puedes tocar" (más los artefactos de `docs/img/` que **ya estaban** sin commitear antes de empezar: `Inicio.png` `??`, `Detalles.png`/`Reproduccion.png` `M`); ningún `.go` ni archivo de `frontend/` tocado; nombres citados verificados contra `go doc`/`app.go`.

---

## T58 — bugs visuales: carátula perdida al retomar y contador de favoritos

Dos bugs no funcionales de la v1.2.0 detectados con el `.exe` (`tareas/T58.md`). Fecha: **2026-10-14**.

### Paso 0 (diagnóstico) — hipótesis confirmadas contra el código real

**Bug A — CONFIRMADO.** Cadena causal completa:
1. `retomar.ts` (`cargarDetalleEnSegundoPlano`) lanza `openAnime` en segundo plano —que hace `currentDetail.set(null)` al empezar (`stores.ts`)— y navega al reproductor sin esperar el detalle.
2. `VistaReproductor.guardarProgreso` construía `cover_url: portadaDe(sel)` y la antigua `portadaDe` devolvía **`''`** si `currentDetail` era null (detalle aún en vuelo, el caso típico del primer `timeupdate`/del desmontaje) o si era de otro anime (slugs compartidos entre proveedores: solo comparaba `detalle.id`, sin proveedor).
3. `internal/db.SaveProgress` **sobrescribe el registro completo** (solo rellena `UpdatedAt`/`Provider` si vienen en cero) → la carátula válida del historial se pisaba con `''`.
4. Al "← Atrás", `App.refrescarHistorialAlSalir` recarga la barra y `TarjetaHistorial` pinta el placeholder (initial) porque `cover_url === ''`. El segundo retomar "se arreglaba" solo: `cargarDetalleEnSegundoPlano` no recargaba la pareja ya cargada y `currentDetail` seguía siendo el correcto. `anime_title` **no** se pisaba hoy (salía de `sel.title`, correcto en todos los flujos), pero aplica la misma regla por la ficha.

**Bug B — CONFIRMADO.** `cargarFavoritos` solo se llamaba al entrar en la pestaña Favoritos (`VistaInicio.seleccionar`) o al montar un `BotonFavorito` en el detalle; `App.onMount` solo cargaba el historial. El contador `★ Favoritos ({$favoritos.length})` lee el store, que arrancaba vacío → "(0)" hasta que alguien abría la pestaña.

### Correcciones (solo archivos de "Puedes tocar")

- **`frontend/src/VistaReproductor.svelte`** (Bug A): `portadaDe` reescrita con cascada — 1) `currentDetail` **solo si coincide proveedor+slug** (`currentProvider` + `detalle.id`); 2) el registro del historial en memoria (`progressMap[episodeID]` o el último `progresoPrevio`, que al montar viene de `recuperarProgreso`); 3) `''` solo sin ninguna fuente. `tituloDe` con la misma cascada (detalle → `sel.title` → historial). `guardarProgreso` usa `tituloDe(sel)`; **sin tocar** throttle, disparadores, fuentes, proxy, navegación, `getNextEpisode` ni la tecla N.
- **`frontend/src/App.svelte`** (Bug B): `onMount` añade `cargarFavoritosAlArranque()` (mismo patrón que `cargarHistorial`); si el primer intento deja el estado en `error` (p. ej. backend aún no respondió), **reintenta una vez** pasados 500 ms; si vuelve a fallar, la cuadrícula de la pestaña conserva su "Reintentar". Sin carrera con la pestaña: `cargarFavoritos` corta en `loading`/`ready`.
- `historial.ts`, `stores.ts`, `favoritos.ts`, `api.ts`, `*.go` y `frontend/wailsjs/**`: **sin tocar** (regla 7 no se activó).

### Compuerta — 2026-10-14, todo en verde

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** (antes: 25 `.go` marcados por el fantasma CRLF del aviso T36 —worktree Windows tras el merge de la v1.2—; normalizados a LF con `sed 's/\r$//'`, sin registro en git) |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** |
| recuento `^--- PASS` (`-count=1 -v`) | **407** — idéntico al de la compuerta de T56 |
| `npm run check` | **0 errores, 0 warnings** (también tras revertir el parche de sesión) |
| `npm run build` | OK (2,65 s; aviso preexistente de chunk >500 kB) |

### Verificación con Playwright (dev server :34115)

- **Bug B end-to-end**: con 2 favoritos ya guardados en el backend del contenedor, la pestaña mostró **"Favoritos (2)"** al arrancar; se añadió **Dark Machine** desde la UI (★ del detalle → "Quitar de favoritos" pressed) → contador **(3)** con Descubrir activa; **recarga de la página** → "★ Favoritos (3)" visible **sin entrar en la pestaña**; con Favoritos activa, lista completa de 3 (Dark Machine, Bleach, Dragon Ball Super: Beerus).
- **Bug A** (con parche de sesión: log en `arrancar` tras `recuperarProgreso`, **revertido**): registro inyectado con el binding real (`SaveProgress` con `cover_url` de JKAnime para `jkanime:bleach:episode:1`), retomado **desde la barra lateral** —el repro del bug— dos veces (con `currentDetail` en vuelo y con el detalle de **otro** anime abierto): `[T58-check]` con `cover`/`title` **no vacíos** en ambos (`prevCover` con la carátula del historial; `mapCover: null` demostró operativo el fallback de memoria). Tras "← Atrás": la tarjeta de la barra **conserva la portada** (sin placeholder) y `GetRecentProgress` confirma el registro en disco con `cover_url`/`anime_title` intactos.
- Errores de consola: **solo preexistentes/ajenos** — favicon 404, `TypeError … 'nodes'` de `wails/ipc.js` (1 por recarga) y `ERR_CONNECTION_REFUSED` al proxy local `127.0.0.1` (limitación conocida del flujo navegador). **0 errores nuevos**.
- Entorno de prueba limpiado al terminar: registro de Bleach y favorito Dark Machine borrados (vuelta a los 2 favoritos previos; historial vacío, como al empezar en el contenedor).

### Desviaciones

- **Colisión de numeración T58** (regla 7, decisión documental): `PROGRESO.md` ya tenía una "T58 — cierre documental post-v1.2.0" hecha; el archivo `tareas/T58.md` (cargado por el humano en el commit previo) define esta otra tarea. Resolución: la **fila T58 de la tabla** es esta de bugs; el cierre documental conserva su bloque con el título aclarado "(etiquetado T58 en su momento)" y una nota de colisión. Ningún contenido se borró.
- **Fantasma CRLF (T36)**: la compuerta inicial marcó 25 `.go` en `gofmt -l` (merge de la v1.2 hecho desde Windows); normalizados a LF sin registrar diff (mismo procedimiento decidido por el humano en T36). `frontend/wailsjs/**` muestra "M" con diff vacío por el mismo motivo (el `wails dev` los reescribe con LF); **sin cambios de contenido**.
- **Verificación del Bug A por lógica + log** (la ficha lo autoriza): en el navegador el proxy local no reproduce vídeo (AGENTS.md) → `modo !== 'video'` y el guardado real con posición **no se ejercitó**; el log de `portadaDe`/`tituloDe` en el estado exacto del montaje (detalle null/en vuelo) más el registro en disco intacto cubren la causa; la prueba final del guardado con vídeo real es con el `.exe`.

### Pendientes

- **Verificar con el `.exe`** (v1.2.x futura): retomar → reproducir un poco → "← Atrás" y comprobar que la tarjeta de la barra conserva la carátula, y el contador de Favoritos correcto al arrancar. Ninguno de los dos fixes puede probarse con vídeo real en el flujo navegador.
- `Type` de `models.Anime` sigue sin existir (pendiente de producto heredado, sin cambio con esta tarea).
