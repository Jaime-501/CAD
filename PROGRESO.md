# PROGRESO — Comfort Anime Desktop (T01–T36 hechas)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivos históricos**: el estado de **T01–T29 + T30** (v1.0) está en **`docs/PROGRESO_v1.0.md`** y el de **T30–T35** (AnimeAV1, v1.1.0, con sus desviaciones por tarea, fixtures y pruebas manuales) en **`docs/PROGRESO_v1.1.md`**. Aquí solo queda lo vigente. Estado del producto: **v1.1.0 publicada** (probada con el `.exe` en Windows).

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

- Los fixtures ya capturados están inventariados en los archivos históricos: JKAnime (v1.0) en `docs/PROGRESO_v1.0.md`, AnimeAV1 (T30–T33, 20 fixtures) en `docs/PROGRESO_v1.1.md`.

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

T01–T29 = v1.0 (`docs/PROGRESO_v1.0.md`) · T30–T35 = v1.1.0 AnimeAV1 (`docs/PROGRESO_v1.1.md`) · T36 = cierre documental de la v1.1.0 · T37 = portabilidad Android · T38 = cierre documental de T37.

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

> Solo lo que sigue vigente; el texto completo (por tarea) está en `docs/PROGRESO_v1.0.md` (T01–T29) y `docs/PROGRESO_v1.1.md` (T30–T35).

- **Modelos (T03)**: `ResourceType` en **minúsculas** (`hls`, `mp4`, `iframe`); `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""`. Cualquier fuente nueva debe emitir esos strings.
- **Normalización y selección (T16/T17)**: `Normalize` corre **dentro** de `Resolver.Resolve` (dedup por recurso sin query pero con fragmento; preserva `Referer`/`Headers`/`Subtitles`) y `SelectBestSource` jerarquiza **HLS > MP4 > iframe > `Type` vacío**, desempate por `Quality` y gana la primera. Consumidor real: `Nucleo.Fuentes` (mejor primero; nombre desde T37).
- **Proxy (T19/T20/T29)**: `NewProxy(nil)` con anti-SSRF, `Register` **solo** con `Type` `hls|mp4` y **sin** `Referer`/`Headers` hacia el frontend (viven en la sesión); `stream.esManifiesto` detecta el manifiesto por el **cuerpo**, no por la extensión. `Sec-Fetch-Site: same-origin` (HLS de AnimeAV1) se declara en `VideoSource.Headers` y `stream/reenvio.go` la inyecta — **nunca** debe limpiarse `Headers` en `Register` (403 en los segmentos).
- **Registro y búsqueda (T20/T34/T35)**: `NuevoNucleoPorDefecto(dir)` registra **`jkanime` + `animeav1`** y `Nucleo.Buscar` los consulta **en paralelo** (timeout 20 s, salida JKAnime primero; error solo si fallan todos; nombres desde T37). `animeflv` **ya no existe** (retirado en T35); `scraper.ErrNoDetails` → `ErrDetalleNoDisponible` se conserva como flujo genérico. El string `"animeflv"` en algunos tests es un provider falso arbitrario, sigue siendo válido.
- **Caché (T21)**: detalle/episodios en memoria con TTL de **10 min**; `GetVideoSources` **nunca** se cachea (regla 9) y `noCache` solo fuerza repetir la extracción.
- **AnimeAV1 (T31–T33)**: vía principal **`GET <ruta>/__data.json`** (devalue: `PageData` + `Unflatten`), `models.Anime.ID` = slug, **lotes de 50** (`tamLote`), dedup de fuentes por **recurso base + idioma** (SUB y DUB son fuentes distintas), HLS derivado **sin petición** (`/play/<32hex>` → `/m3u8/<32hex>`, `/play/*` nunca se pide) con `Type: "hls"` y sus `Headers`; los embeds ajenos (UPNShare/Voe/Byse/MP4Upload) salen **iframe sin resolver**.
- **Progreso (T34)**: `anime_id` es slug compartido entre proveedores → `episode_id` lleva prefijo `<proveedor>:` y toda comprobación de "sigue abierto" compara **proveedor + slug** (no solo el slug).
- **E2E (T28/T35)**: los E2E exigen **peticiones exactas** al sitio falso y cortan hosts ajenos; el de AnimeAV1 demuestra que el proxy inyecta `Sec-Fetch-Site` (segmento por proxy → 206, directo → 403). Si cambia el flujo de peticiones, hay que actualizar el E2E a la vez.

## Pendientes activos

> Solo lo que una tarea futura debe resolver. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`; los de T30–T35 en `docs/PROGRESO_v1.1.md`.

- **Heredados de la v1.0, sin tarea asignada** (detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos"): (1) título de la ventana (`main.go:20` es el técnico `miapp`); (2) marca de tiempo en `WatchProgress` ("Continuar" elige por número, no por recencia); (3) borrado de registros del historial (`internal/db` no lo expone); (4) verificación visual de los avisos de error que ningún test provoca (hoy: el aviso `episodesError` con un fallo real del backend).
- **Cerrados en T38** (cierre documental de T37, ver sección "T38" arriba): ~~firmas desfasadas de **`docs/firmas/db.md`** y **`docs/firmas/cache.md`**~~ → **corregidas**: `db.Open(dir)` con `ErrDirectorioVacio` (sin `rutaPorDefecto` ni `os.UserConfigDir` en el paquete) y el núcleo documentado como `internal/app` con `NuevoNucleo`/`NuevoNucleoPorDefecto(dir)`. Las referencias a `app_nucleo.go` que siguen en `docs/BITACORA.md`, `docs/PROGRESO_v1.0.md`/`v1.1.md`, `docs/investigacion/animeav1.md` y en los bloques T20–T34 de `docs/firmas/app.md` **son históricas o ya llevan la nota del bloque T37**: no se tocan.
- **Cerrados en T36** (documentados, sin acción pendiente): ~~comentarios de `app.go`/`app_nucleo.go` que presentaban a AnimeFLV como proveedor registrado o fallback~~ → **corregidos** (solo comentarios; ahora describen `jkanime` + `animeav1` con búsqueda combinada en paralelo). ~~Reproducir el HLS de AnimeAV1 end-to-end~~ → **cerrado**: E2E offline + real en verde y **prueba manual con el `.exe` en Windows correcta** (HLS SUB/DUB, SUB/DUB e iframes, búsqueda combinada, mismo slug en ambos proveedores, progreso, JKAnime intacto, sin consola); **release v1.1.0 publicada**.
