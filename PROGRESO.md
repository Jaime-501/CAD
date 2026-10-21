    # PROGRESO — Comfort Anime Desktop (T01–T89 hechas)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivos históricos**: el estado de **T01–T29 + T30** (v1.0) está en **`docs/PROGRESO_v1.0.md`**, el de **T30–T35** (AnimeAV1, v1.1.0, con sus desviaciones por tarea, fixtures y pruebas manuales) en **`docs/PROGRESO_v1.1.md`** y el de **T39–T58** (v1.2.0: Home, historial, favoritos, "Siguiente" y sus bugs; incluye los cierres T56/T59) en **`docs/PROGRESO_v1.2.md`**, el de **T64–T67** (v1.3.0: `ventana.ts`, revelado animado y scroll del detalle) en **`docs/PROGRESO_v1.3.md`** y el de **T69–T88** (v1.4.0: rediseño y enfoque móvil) en **`docs/PROGRESO_v1.4.md`**. Aquí solo queda lo vigente. Estado del producto: **v1.2.1 PUBLICADA** (tag `v1.2.1`; ciclo **T39–T59**: v1.2.0 publicada en T56–T58, con T53b y T57; los dos bugs visuales son **T58**; el cierre documental —post-v1.2.0 y de la v1.2.1— es **T59**), **v1.3.0 PREPARADA** — serie **T64–T68** en la rama **`feature/episodios-incrementales`** (episodios incrementales + revelado animado; cierre documental en **T68**)— y **v1.4.0 PREPARADA** — serie **T69–T89** en la rama **`feature/rediseno-movil`** (rediseño y enfoque móvil; cierre documental en **T89**). La **prueba manual con el `.exe` de la v1.2.0** y la **release** las hizo el humano; T57 se probó con el `.exe` sin incidencias. `wails.json` → `productVersion` **1.4.0**: **sin tag ni release aún** (los hace el humano). **Numeración (colisión resuelta en T59):** `tareas/T58.md` = bugs visuales (fila/bloque T58); el cierre documental pasó de T58 a **T59** (sin borrar contenido). **Serie v1.4.0 (rediseño y enfoque móvil, rama `feature/rediseno-movil`):** **T69** (auditoría de UI base, solo lectura) → **T70** (tokens de diseño) → **T71** (base global: Inter/Outfit locales, safe-area, fondo corregido) → **T72/T72b** (primitivas e iconos, puente de color) → **T73/T73b** (barra inferior móvil y cajón opaco) → **T74–T82** (cabecera, buscador, filas del Home, «Continuar viendo», rejilla de resultados, ficha, CTA fija, tarjeta de episodio, orden) → **T83–T86** (reproductor, chip «Siguiente», favoritos/toasts, estados) → **T87** (movimiento y rendimiento) → **T88** (accesibilidad AA) → **T89** (cierre: `lang="es"`, trampa de foco del cajón con `inert`, retirada de las `--cad-*` y de Nunito, `productVersion` 1.4.0). **El detalle largo está en `docs/PROGRESO_v1.4.md`**; aquí solo un resumen por tarea.

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
| [x] | T58 | Carátula perdida al retomar y contador de favoritos | T57 |
| [x] | T59 | Cierre documental post-v1.2.0 y de la v1.2.1 (renombrado: el cierre era T58; los bugs se quedan con T58) | T58 |
| [x] | T60 | Resolver de MP4Upload en paquete neutro `embeds` (sin cableado) | Informe de adds |
| [x] | T61 | Conectar el resolver de MP4Upload a JKAnime y AnimeAV1 | T60 |
| [x] | T62 | Redactar tokens en el PATH de las URLs de medio (`logx` + `stream`) | T60 |
| [x] | T63 | UI: selector de fuentes (principales primero, iframes como extra) | T61 |
| [x] | T64 | Lógica pura de ventana de episodios (`ventana.ts` + tests) | — |
| [x] | T65 | Revelado animado de tarjetas de episodio por visibilidad (sin paginar) | T64 |
| [x] | T66 | Clave de scroll del detalle por proveedor + slug | T65 |
| [x] | T67 | Restaurar el scroll del detalle tras la recarga por lotes | T66 |
| [x] | T68 | Cierre documental de la v1.3.0 (T64–T67) | T67 |
| [x] | T69 | Auditoría de UI base (v1.4.0, solo lectura) | — |
| [x] | T70 | Tokens de diseño en `style.css` (v1.4.0) | T69 |
| [x] | T71 | Base global: tipografía, viewport, fondo y safe-area (v1.4.0) | T70 |
| [x] | T72 | Primitivas de UI e iconos (v1.4.0) | T70, T71 |
| [x] | T72b | Puente de color: literales a tokens (v1.4.0) | T70, T71, T72 |
| [x] | T73 | Barra inferior móvil y cajón de historial (v1.4.0) | T70–T72b |
| [x] | T73b | Cajón opaco (v1.4.0) | T73 |
| [x] | T74 | Cabecera compacta de detalle y reproductor (v1.4.0) | T73b |
| [x] | T75 | Buscador y pestañas del Home (v1.4.0) | T74 |
| [x] | T76 | Filas y tarjetas del Home (v1.4.0) | T75 |
| [x] | T77 | Fila "Continuar viendo" del Home (v1.4.0) | T76 |
| [x] | T78 | Rejilla de resultados de búsqueda y estados (v1.4.0) | T77 |
| [x] | T79 | Ficha de detalle: portada, metadatos, géneros y sinopsis (v1.4.0) | T74, T78 |
| [x] | T80 | Barra de acción fija «Continuar / Reiniciar» del detalle en estrecho (v1.4.0) | T73b, T74, T79 |
| [x] | T81 | Tarjeta de episodio y rejilla en modo estrecho (v1.4.0) | T72, T79, T80 |
| [x] | T82 | Orden ascendente/descendente de episodios (v1.4.0) | T70–T81 |
| [x] | T83 | Layout del reproductor y selector de fuentes (v1.4.0) | T70–T82 |
| [x] | T84 | Chip "Siguiente" del reproductor (v1.4.0) | T83 |
| [x] | T85 | Favoritos, toasts y verificación del deslizar (v1.4.0) | T84 |
| [x] | T86 | Estados restantes, token `--scrim` y barrido de px (v1.4.0) | T70–T85 |
| [x] | T87 | Movimiento y rendimiento (v1.4.0) | T70–T86 |
| [x] | T88 | Accesibilidad AA (v1.4.0) | T70–T87 |
| [x] | T89 | Cierre de la v1.4.0 (lang, foco del cajón, limpieza del legado, versión) | T88 |

T01–T29 = v1.0 (`docs/PROGRESO_v1.0.md`) · T30–T35 = v1.1.0 AnimeAV1 (`docs/PROGRESO_v1.1.md`) · T36 = cierre de la v1.1.0 · T37 = portabilidad Android · T38 = cierre documental de T37.
T39–T57 = v1.2.0 (Home, historial lateral, favoritos, "Siguiente" y su estado discreto; **incluye T53b y T57**); plan en `tareas/v1.2_PLAN.md`, detalle **T39–T58 + cierres T56/T59** archivado en `docs/PROGRESO_v1.2.md`, con resúmenes al final de este archivo.
La **v1.2.0 está PUBLICADA** (prueba manual con el `.exe` y release hechas por el humano). La **v1.2.1** (T58 bugs + T59 cierre documental) está **preparada**: `productVersion` 1.2.1, sin tag ni release aún — los hace el humano.

**v1.3.0 — episodios incrementales** (rama `feature/episodios-incrementales`): serie **T64–T69** para mostrar los episodios del detalle de forma incremental y con animación de entrada. **T64** (lógica pura de `ventana.ts` + tests), **T65** (revelado animado por visibilidad, sin paginar), **T66** (clave de scroll del detalle por proveedor + slug) y **T67** (restauración del scroll del detalle cuando la lista se recarga) hechas; **T68** (cierre documental de la v1.3.0) hecho: `productVersion` **1.3.0**, **sin tag ni release** (los hace el humano). Los **T66–T67 originales** (paginado / ventana client-side) quedan **descartados** por las mediciones de T65 (sin jank y con 0 saltos de scroll; el coste real es de red, ya cubierto por el mostrado por lotes); **T69 pendiente** de la serie. Notas largas de T64–T67 archivadas en `docs/PROGRESO_v1.3.md`.

**v1.4.0 — rediseño y enfoque móvil** (rama `feature/rediseno-movil`): serie **T69–T89** que rediseña la interfaz para móvil y tableta **sin cambiar la geometría del escritorio** (≥ 900 px). **T69** (auditoría de UI base, solo lectura) → **T70** (tokens de diseño) → **T71** (base global: Inter/Outfit locales, safe-area, fondo corregido) → **T72/T72b** (primitivas e iconos, puente de color) → **T73/T73b** (barra inferior móvil y cajón opaco) → **T74–T82** (cabecera, buscador, filas del Home, «Continuar viendo», rejilla de resultados, ficha, CTA fija, tarjeta de episodio, orden) → **T83–T86** (reproductor, chip «Siguiente», favoritos/toasts, estados unificados) → **T87** (movimiento y rendimiento) → **T88** (accesibilidad AA) → **T89** (cierre: `lang="es"`, trampa de foco del cajón con `inert`, retirada de las `--cad-*` y de Nunito, `productVersion` 1.4.0 y cierre documental). **El detalle largo de T69–T88 está en `docs/PROGRESO_v1.4.md`** (con la tabla de correspondencia del plan inicial T83–T93) y el cierre en el bloque de T89 más abajo. `wails.json` → `productVersion` **1.4.0**: **sin tag ni release aún** (los hace el humano).

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

> Solo lo que una tarea futura debe resolver. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`; los de T30–T35 en `docs/PROGRESO_v1.1.md`; los de T39–T58 en `docs/PROGRESO_v1.2.md`; los de T64–T67 en `docs/PROGRESO_v1.3.md`; los de T69–T88 en `docs/PROGRESO_v1.4.md`.

- **Prueba manual con el `.exe` de la v1.4.0** (la hace el humano): ventana **estrecha** (barra inferior móvil, cajón de historial con la trampa de foco, CTA fija, sinopsis plegable, orden de episodios), **reproducción HLS/MP4 real**, **chip «Siguiente»** y **pantalla completa**. El flujo de navegador no ejercita el vídeo real (el proxy local solo es alcanzable desde el `.exe`, ver AGENTS.md).
- **Release de la v1.4.0** (tag + release): la hace el humano. `productVersion` está en **1.4.0** a la espera de tag/release. La **v1.2.1 ya está taggeada** (`v1.2.1`); la v1.3.0 sigue **sin tag**.
- **Deriva de scroll T66/T67 del detalle** (`+52 px`, 2743 → 2795): medida en T89 con el código actual **y** con T87+T88 revertidos — **idéntica en los dos casos**, así que es previa a T87. No se corrige (T89 solo la midió). Queda por diagnosticar el asentamiento de la rejilla al recargar por lotes.
- **Foco del cajón**: el `inert` de T89 ya impide que `Tab` alcance el contenido de fondo, pero quedan **2 fugas** en `a11y_auditoria.mjs`: el `<body>` (respaldo del documento) y el **velo** («Cerrar historial»), que es un `<button>` y que la ficha de T89 exigía **no** inertizar. Para llegar a 0 fugas habría que sacar el velo del orden de tabulación (`tabindex="-1"` en `frontend/src/BarraLateral.svelte`) o añadir una trampa de foco explícita: **tarea nueva**.
- **«Episodios siguientes»** y **«pantalla completa»**: quedaron **FUERA** de la v1.4.0 (ver la tabla de correspondencia en `docs/PROGRESO_v1.4.md`). Son ideas de producto sin tarea.
- **Extractores del informe de `adds`** (paquete `internal/scraper/embeds`): numerados **T94–T99** — Voe, Byse, UPNShare, Streamtape, purga de hosts muertos y `sandbox` por host.
- **Verificación con el `.exe` de T65/T67** (revelado animado y restauración del scroll del detalle): el flujo navegador no ejercita el vídeo real (proxy local, AGENTS.md) ni todos los ciclos de recarga; T65 y T67 se verificaron con Playwright (8/8 y 6/6) y por lógica+log. Probar con el `.exe` en la próxima release. Los **fixes T58** (carátula al retomar y contador de favoritos) se dieron por cerrados con la **v1.2.1 publicada**.
- **(4) Verificación visual de los avisos de error** (heredado de la v1.0, detalle en `docs/PROGRESO_v1.0.md` → «Pendientes activos»): los avisos de error que ningún test provoca (hoy: el aviso `episodesError` con un fallo real del backend). Los heredados (1) título de la ventana, (2) marca de tiempo y (3) borrado del historial están **cerrados** (T56, T40/T41 y T51–T52).
- **`Type` de `models.Anime`** sigue sin existir (pendiente de producto heredado: los items del Home y la búsqueda salen sin tipo; si el producto lo quiere hay que ampliar `models` en una tarea nueva).
- **Idea futura (no es tarea): chips de rango de episodios.** T65 descartó paginar; la ventana de `ventana.ts` (T64) queda disponible y probada por si más adelante se quiere una selección por rangos en lugar del mostrado por lotes.

---

## Informe de adds (14/10/2026)

> Investigación **solo de lectura** sobre la publicidad que traen los iframes de terceros.
> Informe completo (metodología, tablas y evidencias): **`docs/investigacion/adds.md`**;
> fixtures y la página de prueba: **`docs/investigacion/adds/`**. No se tocó **ningún**
> archivo de código (regla 7 no se activó).

**Resultado en una línea**: ni el `sandbox` ni un proxy de HTML evitan el anuncio; lo que
funciona es resolver el medio a MP4/HLS en Go, host a host (H1 no sirve como solución
general y queda no concluyente sobre popups tras clic y WebView2, H2 se sostiene con
fricción, H3 se sostiene).

- **Qué se midió**: 13 páginas de episodio de JKAnime + 10 `__data.json` de AnimeAV1
  (inventario de hosts + P1c), 7 hosts × 3 cargas sin sandbox (P2), 7 hosts × 4 variantes
  de `sandbox` (P3) y resolución del medio por host con `Range: 0-0` (P4). ~71 peticiones
  deliberadas, **sin 429**. No se pulsó "play" ni ningún anuncio.
- **P1c (qué elige hoy el selector)**: `SelectBestSource` (`selector.go:26`, HLS 4 > MP4 3
  > iframe 2 > resto) es el del núcleo; sobre las 23 muestras el iframe es la **única**
  opción en **0 de 10** de AnimeAV1 (las 10 traen HLS derivado de zilla) y en **13 de 13**
  de JKAnime con lo que el HTML determina sin red (JKPlayer está presente en las 13 pero
  su aporte necesita la página del player: no concluyente).
- **Hallazgos fuertes**: (1) Voe y Streamtape **detectan el sandbox** y sirven `/blocked`;
  (2) **MP4Upload es RESOLVIBLE limpio** (URL del MP4 en claro, **sin query** — el token va
  en el path — y CDN que responde a `Range` sin mirar Referer, 114 MB confirmados y sigue
  respondiendo a los ~45 min); (3) Voe/Byse/UPNShare/Streamtape son **RESOLVIBLES
  FRÁGILES** (cifrado de 2 caracteres + ROT13 / API + proof-of-work / API cifrada /
  token + captcha); (4) **Mega es NO RESOLVIBLE** (E2E) y, de paso, **no cargó ni un
  dominio de anuncio**; (5) **`sfastwish.com` está parkado** (GoDaddy + Yahoo Ads) pese a
  ser el host con más apariciones en JKAnime; (6) ningún host envía `X-Frame-Options`;
  (7) el anuncio del reproductor vive en el JS del host, **pero** Streamtape trae sus
  scripts de popunder en el HTML crudo del embed y la página de episodio de AnimeAV1
  trae GTM + un script de anuncio (que la app no ejecuta).
- **Fixtures** (`docs/investigacion/adds/`, sanitizados; escaneo final **0 coincidencias
  reales**): `jkanime_var_servers.html`, `animeav1_embeds.json`, `voe_e_redirect.html`,
  `voe_destino_embed.html`, `mp4upload_embed.html`, `byse_embed.html`,
  `upnshare_embed.html`, `streamtape_embed.html`, `mega_embed.html`,
  `cabeceras_embeds.txt` y `sandbox_probe.html` (la página de prueba, con
  `?h=host,host&v=A,B,C,D`). El token de `/d/` de MP4Upload, los `expires`/`ip`/`token`
  de Streamtape, el blob cifrado y el CSRF de Voe y la clave de descifrado de Mega están
  como `TOKEN_DE_PRUEBA`; el enlace de Mega **no se guarda** en el repositorio.
- **Pendiente de producto** (decisión del humano, no técnica): si se quiere invertir en
  extractores de medio directo (T60–T64 abajo) o dejar los iframes como están y solo
  purgar los hosts muertos.
- **Tareas propuestas** (sin renumerar la tabla de arriba; se numeran cuando se arranquen):
  **T60, T61 y T62 hechas** (MP4Upload: resolver neutro `embeds` —T60—, cableado a JKAnime y
  AnimeAV1 —T61— y redacción del token del path en logs —T62—; ver sus bloques aquí abajo) ·
  **T63** reorganización del selector de fuentes (principales primero, iframes como extra;
  hecha, ver bloque al final) · **T70** extractor Voe (portar el descifrado del `loader`) ·
  **T71** extractor Byse (`/api/videos/<id>/embed/details`; el proof-of-work queda fuera) ·
  **T72** UPNShare (API cifrada) · **T73** Streamtape (token + captcha) · **T74** purga de
  hosts muertos del inventario (`sfastwish.com`; vigilar `mixdrop`/`dsvplay`/`vidhidevip`, no
  muestreados) · **T75** (solo si se quiere) `sandbox` **por host** y sin `allow-same-origin`,
  nunca global — **NO PROBADA** en esta investigación: solo se probó en Chromium de escritorio,
  sin clic y sin WebView2.
  **Nota de renumeración (T64):** los números **T64–T69** quedaron reservados para la serie
  **v1.3.0 — episodios incrementales** (rama `feature/episodios-incrementales`; T64 = lógica
  pura de `ventana.ts`, ver bloque al final), por lo que estas propuestas de extractores pasan
  a **T70–T75**. El texto histórico de arriba (T64 = Voe, etc.) no se borra: la numeración
  vigente es la de esta lista.

---

## Resumen de los cierres de la v1.2.x (detalle largo en `docs/PROGRESO_v1.2.md`)

> En T59 se movieron a `docs/PROGRESO_v1.2.md` (texto literal) las notas largas de T56, T57, T58 y del cierre documental. Aquí queda lo esencial.

### Cierre de la v1.2.0 (T56) — resumen

Hecha el **2026-10-12** en `feature/home-historial-favoritos`: `main.go` solo el `Title` (`CAD · Comfort Anime Desktop`), `wails.json` → `1.2.0`, `00_REGLAS.md`, `README.md`, `DESARROLLO.md`, `AGENTS.md`, firmas contrastadas y notas T39–T55 archivadas. Compuerta en verde (407 tests, `npm run check` 0/0, `wails build` OK). La **prueba manual con el `.exe`** y la **release** las hizo el humano: **v1.2.0 publicada**.

### T57 — botón "Siguiente ▸" discreto durante la reproducción — resumen

`BotonSiguiente.svelte` con prop `discreto` (guardia `discreto && estado === 'listo'`; `opacity 0.4` + `blur(2px)` + transición 200 ms; se aclara con hover/foco; `prefers-reduced-motion` sin transición) y `VistaReproductor.svelte` con estado `reproduciendo` seteado **dentro de los handlers existentes del `<video>`** (`alAvanzar` con `!v.paused`, `alPausar`, `alTerminar`; cero listeners nuevos). **Probada con el `.exe` sin incidencias** (se aclara al pausar, al 90 % y con HLS/MP4 real): anotado en T59.

### T58 — bugs visuales: carátula perdida al retomar y contador de favoritos — resumen

Detectados con el `.exe`. **Bug A**: `portadaDe`/`tituloDe` reescritos en cascada (detalle del mismo proveedor+slug → registro en memoria → vacío) en `VistaReproductor.svelte`; ya no se pisa con `''` la carátula del historial al retomar con el detalle en vuelo o con otro anime abierto. **Bug B**: `App.svelte` carga los favoritos al arrancar (`cargarFavoritosAlArranque` con un reintento a los 500 ms) → el contador "★ Favoritos (N)" es correcto sin abrir la pestaña. **Pendiente**: verificar los dos fixes con el `.exe` (arriba).

### T59 — cierre documental (post-v1.2.0 renumerado + v1.2.1) — bloque de esta sesión

Cierre documental de la **v1.2.1**, solo documentación y versión, sin cambios de comportamiento (fecha **2026-10-14**):

- **Renumeración (paso 1)**: los bugs visuales se quedan como **T58**; el cierre documental post-v1.2.0 pasó de T58 a **T59** (fila de la tabla + títulos de bloque, con línea aclaratoria; ningún contenido borrado). La nota de la colisión del encabezado y la fila/bitácora quedan actualizadas.
- **T57 (paso 2)**: el apartado de pendientes del `.exe` se sustituyó por la anotación de que **T57 se probó con el `.exe` sin incidencias** (pausar, 90 %, HLS/MP4 real).
- **Archivo histórico (paso 3)**: notas largas de **T56, T57, T58** y del cierre documental movidas **texto literal** a `docs/PROGRESO_v1.2.md` (con los dos ajustes anteriores); aquí solo resúmenes.
- **`wails.json` (paso 4)**: `info.productVersion` → **`"1.2.1"`** (único cambio del archivo).
- **`CHANGELOG.md` (paso 5)**: entrada **1.2.1** con los dos arreglos de T58 y la mejora T57. Comprobado contra el CHANGELOG previo: **solo existía la entrada 1.1.0** (no había 1.2.0 ni la mejora T57) — anotado en la propia entrada.
- **`AGENTS.md` y `docs/DESARROLLO.md` (paso 6)**: estado **T1–T59 / v1.2.1 PUBLICADA** y rango de "Trabajo con agentes" hasta T59; añadido el aviso breve del **fantasma CRLF** (autocrlf sin `.gitattributes`) a ambos (no estaba).
- **`docs/BITACORA.md` (paso 7)**: fila de sesión de **T58** (bugs) y de **T59**, la desviación de la colisión de numeración y la renumeración del cierre.
- **Pendientes activos (paso 8)**: solo (4) avisos de error, la verificación `.exe` de los fixes T58, la release de la v1.2.1 (humano) y el `Type` de `models.Anime`.
- **Archivos tocados**: `PROGRESO.md`, `CHANGELOG.md`, `AGENTS.md`, `docs/DESARROLLO.md`, `docs/PROGRESO_v1.2.md`, `docs/BITACORA.md` y `wails.json` (solo `info.productVersion`). **Nada tocado fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `main.go`, `frontend/**`, `tareas/00_REGLAS.md` y `tareas/T58.md` intactos; `frontend/wailsjs/**` sin regenerar ni cambios (el `wails build` no alteró los bindings).

### Comprobación — 2026-10-14, todo en verde

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (`miapp` sin tests de paquete) |
| recuento `^--- PASS` (`-count=1 -v`) | **407** — idéntico al de T56/T58 (mismo recuento exigido) |
| `npm run check` (frontend) | **svelte-check found 0 errors and 0 warnings** |
| `wails build -platform windows/amd64` | **OK** → `build/bin/miapp.exe` en 18,7 s (compila con `productVersion` 1.2.1) |
| `git status --porcelain` | solo los 6 archivos de "Puedes tocar" (más `tareas/T59.md` sin trackear, previo a la sesión) |
| `git diff --stat --ignore-cr-at-eol -- '*.go'` | **vacío** (ningún `.go` con cambios reales) |

**No se creó tag ni release** (lo hace el humano). Detalle largo de T56/T57/T58/cierre en `docs/PROGRESO_v1.2.md`; sesión en `docs/BITACORA.md`.

---

## T60 — Resolver de MP4Upload en paquete neutro `embeds` (sin cableado)

Hecha el **2026-10-15** (primera tarea del informe de adds que se ejecuta). Crea el paquete **neutro** `internal/scraper/embeds` con la interfaz `Resolver` y el resolver de MP4Upload —el único host RESOLVIBLE limpio del informe— y **sin cablearlo** a ningún proveedor (eso es la tarea siguiente). Depende solo de `models`, `httpx` y `logx`; no importa `jkanime`, `extractors` ni `animeav1`.

**Qué se hizo:**

- **Fixture (requisito 1)**: copiado `docs/investigacion/adds/mp4upload_embed.html` → `internal/scraper/embeds/testdata/mp4upload_embed.html`. Comprobado que contiene el elemento del que se extrae la URL: el `src` del `player.src({ type: "video/mp4", src: "https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4" })` (token ya sanitizado). Sin ejecutar JS.
- **Interfaz `Resolver`** (`embeds.go`): `Name() string`, `CanHandle(url string) bool`, `Resolve(ctx, url, referer string) ([]models.VideoSource, error)`.
- **Resolver MP4Upload** (`mp4upload.go`): descarga el embed con el `httpx.HTTPDoer` inyectado (límite `2 << 20`), extrae la URL del bloque `player.src({ … })` con dos regex (bloque + atributo `src`) y devuelve una fuente `Type: "mp4"`, `Server: "MP4Upload"` y `Referer` = la URL del embed. `CanHandle` reconoce `mp4upload.com`/`www.mp4upload.com` en las formas `/embed-<id>.html` y `/<id>`.
- **Errores tipados** (`errors.Is`): `ErrSinMedio`, `ErrCuerpoGrande`, `ErrURLInvalida`; 4xx/5xx y `ctx` cancelado llegan de `httpx` como `*ScraperError` (`ErrNotFound`/`ErrServerError`/`ErrNilDoer`/`context.Canceled`); nunca pánico.
- **Tests**: `embeds_test.go` (2: interfaz + guardia de dependencias con `go list -deps .`), `mp4upload_test.go` (11: tabla de `CanHandle`, fixture real con `httptest.Server`, forma directa, Referer de la petición, 404, 500, página sin medio, ctx cancelado, cuerpo enorme, sin doer, URL inválida) y `embeds_integration_test.go` (1 con `-tags integration`).
- **Smoke con red**: resuelve un embed real y verifica `Range: bytes=0-0` → **206**. La URL del embed **no se persiste**: sale de la variable de entorno `MIAPP_MP4UPLOAD_EMBED` (id de un episodio real vigente); sin ella el test se salta.

**Pasos de extracción verificados contra el fixture:** descargar el HTML → aislar `player.src({ … })` → leer su `src` (URL del MP4 en claro) → devolver `VideoSource{Type: mp4, Server: MP4Upload, Referer: <embed>}`. Coinciden con `adds.md` (1 ocurrencia de `player.src`, cero `<source>`).

**Cabeceras del CDN y temporalidad:** el CDN responde a `Range: bytes=0-0` con **206** (`nginx`, `Content-Range: bytes 0-0/114919837`, 114 MB). **Desviación del informe**: en vivo (2026-10-15) el CDN **exige Referer** —403 sin él y 206 con cualquier Referer del dominio `mp4upload.com`; 403 con un Referer ajeno—, por eso la fuente lleva `Referer` = la URL del embed (`Headers` vacío). La URL del MP4 **no lleva query** (el token va en el path `/d/<token>/video.mp4`): no hay `expires`/`ip` visibles, pero se trata como temporal (regla 9, no se cachea ni persiste; no se registra completa —`logx.Redact`—).

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** |
| recuento `^--- PASS` (`-count=1 -v`) | **420** (antes **407**; +13 de `embeds`) |
| `go test -tags integration ./internal/scraper/embeds/ -v` | **PASS** con `MIAPP_MP4UPLOAD_EMBED` (smoke real 206); **SKIP** sin ella |
| `go list -deps ./internal/scraper/embeds` (grep jkanime/extractors/animeav1/wailsapp) | **vacío** |
| `npm run check` (frontend) | **0 errores, 0 avisos** (sin cambios) |

**Archivos tocados:** `internal/scraper/embeds/**` (nuevo: `embeds.go`, `mp4upload.go`, `embeds_test.go`, `mp4upload_test.go`, `embeds_integration_test.go`, `testdata/mp4upload_embed.html`), `docs/firmas/embeds.md` (nuevo), `tareas/00_REGLAS.md` (árbol + regla de dependencias: `embeds` neutro) y `PROGRESO.md`. **Nada tocado fuera de "Puedes tocar"** (regla 7 no se activó): `extractors/**`, `jkanime/**`, `animeav1/**`, `internal/app/**`, `stream/**`, `frontend/**`, `app.go`, `main.go`, `wails.json` y `AGENTS.md` intactos.

**Desviaciones:**

- **El CDN sí exige Referer** (el informe lo daba por no exigido): manda la medición en vivo; la fuente lo lleva. Anotado en `docs/firmas/embeds.md`.
- **HTML artificial declarado**: el caso "página sin medio" (embed degradado) y las tablas de `CanHandle` con URLs ficticias; el resto usa el fixture real. No hay fixture real para esos casos.
- **El id del smoke no se persiste**: se pasa en `MIAPP_MP4UPLOAD_EMBED`; ejecutado con un embed vivo (PASS) sin pegar el id en el repositorio ni en `PROGRESO.md`.
- **No se midió** si la URL del MP4 caduca por tiempo o si se ata a la IP (solo el 206 desde la misma sesión/IP).

**Pendientes:**

- **Cablear el resolver a `jkanime` y `animeav1`** (tarea siguiente): convertir el iframe de MP4Upload en fuente MP4 directa desde ambos proveedores; hoy el paquete `embeds` no lo consume nadie.
- Sin tag ni release: no aplica a esta tarea.

---

## T62 — Redactar tokens en el PATH de las URLs de medio (logx + stream)

Hecha el **2026-10-15**, tarea propia nacida del tripwire del req6 de T61: al servir un MP4 resuelto de MP4Upload por el proxy, el log registraba `https://a2.mp4upload.com:183/d/<token>/video.mp4` con el token en el **path** (`logx.Redact` solo cubría la query y `stream.urlSegura` conservaba esquema+host+ruta). **No retoma T61.**

**Qué se hizo:**

- **`internal/logx`**: `Redact` aplica ahora `redactarPath` (tras `reFirma`), que enmascara con `***` los **segmentos opacos** del path de cada URL `http(s)`, conservando esquema, host, puerto, la query y el último segmento con extensión. Un segmento es opaco si no tiene punto y (a) encaja en `^[A-Za-z0-9_+=]{12,}$` o (b) es alfanumérico de 10+ que mezcla letras y dígitos. Los guiones quedan fuera de ambas reglas (no se tocan los slugs). Internos nuevos (sin cambio de API): `reURLPath`, `reSegmentoLargo`, `reSegmentoAlnum`, `redactarPath`, `redactarSegmentos`, `segmentoOpaco`, `mezclaLetrasYDigitos`.
- **`internal/stream`**: `urlSegura` devuelve `logx.Redact(scheme://host/path)` (solo `urlSegura` y sus tests, como autoriza la tarea). Sigue sin query; la URL real que el proxy pide al origen **no se toca**, solo el log.
- **Tests**: `internal/logx/logx_test.go::TestRedactPath` (8 subtests) y `internal/stream/stream_test.go::TestURLSeguraRedactaTokensDelPath`.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go build ./...` | OK |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go test -race -count=1 ./...` | **10 de 11 paquetes `ok`**; `internal/app` FAIL por los **2 E2E offline pre-existentes de T61** (salida a `mp4upload.com` en el sitio falso, req7 de T61) — **no** por los logs |
| recuento `^--- PASS` (`-count=1 -v`) | **432** top-level (**430** antes; +2) y **281** subtests (**273** antes; +8) |
| `npm run check` (frontend) | **0 errores, 0 avisos** (sin cambios) |

**Desviaciones:**

- **Diseño (b) elegido frente a (a):** regla general de segmentos opacos en vez de un patrón específico `/d/<token>/`. (a) es frágil (un patrón por CDN), no cubre `/m3u8/<32 hex>` ni los segmentos base64 de zilla, y envejece con cada CDN; (b) cubre el caso real de MP4Upload y los manifiestos/segmentos HLS sin enumerar rutas, y se refina excluyendo segmentos con guiones para no destruir el diagnóstico (los slugs `/bleach/1/`, `demon-slayer-kimetsu-no-yaiba-2019` y `__data.json` salen intactos). Detalle en `docs/firmas/logx.md`.
- **Umbral del blob largo en 12, no 16:** `TOKEN_DE_PRUEBA` mide 15 y los ids reales de MP4Upload, 12.
- **2 FAIL pre-existentes en `internal/app`:** `TestE2EFlujoCompletoOffline` y `TestE2EAnimeAV1FlujoCompletoOffline` fallan en `app_e2e_test.go:958/1785` ("el flujo intentó salir del sitio falso … mp4upload.com") porque T61 aún no ha hecho su req7 (servidor falso del embed). `comprobarLogsSinSecretos` (paso 8 / paso 6) **sí pasa**: el requisito 4 de T62 se cumple. Fuera del alcance de T62 (`internal/app` no es tocable; no se retoma T61).

**Pendientes:**

- Ninguno propio. **T61** sigue en curso (req5–req8); el fix de T62 **desbloquea su req6**.

**Archivos tocados:** `internal/logx/logx.go`, `internal/logx/logx_test.go`, `internal/stream/stream.go` (**solo `urlSegura`**), `internal/stream/stream_test.go`, `docs/firmas/logx.md`, `docs/firmas/stream.md`, `PROGRESO.md`. Nada fuera de "Puedes tocar" (regla 7 no se activó).

---

## T61 — Conectar el resolver de MP4Upload a JKAnime y AnimeAV1

Hecha el **2026-10-15**. Cablea el resolver neutro de embeds (T60) a los dos proveedores: cuando un episodio ofrece **MP4Upload** como iframe (`var servers` en JKAnime; `embeds` SUB/DUB en AnimeAV1), se publica **además** la fuente `mp4` directa, conservando el iframe como respaldo (`SelectBestSource` ya prefiere MP4 > iframe). Sin ejecutar JavaScript y sin cachear ninguna URL resuelta (regla 9).

**Qué se hizo:**

- **Requisito 1 (un fallo no rompe el resto):** `enriquecerConEmbeds` recorre las fuentes ya extraídas y, por cada iframe que un resolver reconozca (`CanHandle`, sin red), llama a `Resolve` con `Referer` vacío y `context.WithTimeout(ctx, tiempoResolucionEmbeds)` (**8 s**). Un fallo registra `slog.Warn` con `logx.Redact` (sin la URL del CDN) y **conserva el iframe**; el `ctx` del flujo se respeta.
- **Requisito 2 (JKAnime):** nuevo `internal/scraper/jkanime/embeds.go`; `GetVideoSources` devuelve `enriquecerConEmbeds(ctx, fuentes)`. Dedup por `claveRecurso` (esquema+host+ruta sin query, con fragmento): la misma URL con otra firma no se duplica. Campo nuevo `JkAnime.Embeds []embeds.Resolver`; `New` inyecta por defecto `embeds.NewMP4Upload(doer)` si es nil.
- **Requisito 3 (AnimeAV1):** `sources.go` enriquece al final de `GetVideoSources` (mismo patrón); campo `AnimeAV1.Embeds` e inyección por defecto en `New`. **Conserva el `Language`** de cada iframe en la fuente resuelta (SUB y DUB siguen siendo **dos** fuentes aunque el CDN sea la misma URL). `animeav1` importa solo `embeds` (ni `jkanime` ni `extractors`).
- **Requisito 4 (Referer):** la `VideoSource` resuelta lleva `Referer` = URL del embed; el proxy (`Register`) lo guarda en la sesión y **no** lo envía al frontend (contrato T29). Verificado en los E2E y en un test nuevo con el proxy REAL.
- **Requisito 9:** ninguna URL resuelta se cachea; `opts`/`NoCache` siguen ignorándose.
- **Fixture y servidores falsos (req7):** los E2E offline sirven el fixture **real** `internal/scraper/embeds/testdata/mp4upload_embed.html` y un CDN falso con `Range` → 206; el transporte traduce los hosts `www.mp4upload.com` y `a2.mp4upload.com:183` al sitio falso y **sigue cortando cualquier otro host** antes de conectar. Vídeo y manifiestos artificiales declarados.

**Requisito 5 (proxy + puerto no estándar) — evidencia:**

- Test nuevo `internal/app/app_nucleo_animeav1_test.go::TestMP4UploadRefererSoloViveEnElProxy`: proxy **real** (`stream.NewProxy`, con `PermitirRedPrivada` solo porque el CDN falso vive en el `127.0.0.1` del `httptest`) + origen en **puerto NO estándar**. Comprueba que (a) la fuente que cruza a Wails sale **sin `Referer` ni `Headers`** (y el JSON no contiene `Sec-Fetch-Site`/`Referer`/`mp4upload.com/embed-`), (b) el origen **sí** recibe el `Referer` del embed y `Sec-Fetch-Site`, y (c) el `Range: bytes=0-0` viaja y el origen responde **206** con `Content-Range`. Si el `Referer` no se guardase en la sesión, el CDN daría 403.
- **`internal/stream` NO necesitó cambios:** su anti-SSRF (`red.go`) valida la **IP resuelta** (`IsLoopback`/`IsPrivate`/`IsLinkLocalUnicast`…), **no el puerto**; un host público como `a2.mp4upload.com:183` pasa. La E2E offline ya pedía el mp4 por el proxy en un puerto no estándar y en producción no hace falta excepción alguna.

**Requisito 6 (el token del path no aparece en logs) — evidencia y tripwire:**

- `comprobarLogsSinSecretos` (usado por los 2 E2E offline y por los 2 E2E de integración) añade el bloqueo de **`TOKEN_DE_PRUEBA`**: el token del path del CDN (`/d/<token>/video.mp4`) no puede quedar en claro en ningún log. El `Referer`, el `Range` y el 206 se comprueban al pedir el mp4 por el proxy (paso "5b" de cada E2E).
- **El guard muerde:** revirtiendo temporalmente `logx.redactarPath` a `return s`, el E2E falla con `el registro "TOKEN_DE_PRUEBA" aparece en los logs: … url=https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4 …`; restaurado, **PASS**. (La redacción del path la añadió **T62**, que desbloqueó este requisito.)

**Requisito 7 (peticiones exactas):** los E2E exigen la lista **exacta** al sitio falso, que ahora incluye `GET /embed-<id>.html` (JKAnime: 1; AnimeAV1: SUB y DUB) y, al pedir el mp4 por el proxy, `GET /d/<token>/video.mp4`. Fuentes: `TestE2EFlujoCompletoOffline` **11 → 12** (2 MP4, 1 HLS, 9 iframe); `TestE2EAnimeAV1FlujoCompletoOffline` **10 → 12** (2 HLS, 8 iframe, 2 MP4). La mejor sigue siendo HLS en ambos.

**Requisito 8 (medición, sin cambiar comportamiento):** en las **2 muestras disponibles** (fixtures reales `episode_page.html` de JKAnime y `episode_data.json` de AnimeAV1), **2/2** episodios ofrecían MP4Upload y **2/2** quedaron con `mp4` resuelto en la lista; **0/2** lo tienen como fuente elegida por `SelectBestSource` (gana el HLS). Se registra en el log `T61 medida: …` de cada E2E.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** (normalizadas a LF las líneas CRLF pre-existentes — fantasma T36) |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (los 2 E2E offline que fallaban en T62 ahora pasan) |
| recuento `^--- PASS` (`-count=1 -v`) | **435** (antes **432**, con los 2 E2E offline en FAIL; +1 test nuevo +2 E2E que pasan). *(La ficha citaba 420: es anterior a T60, que sumó 13 tests.)* |
| `go test -tags integration ./... -run "E2E\|MP4Upload" -v` | **PASS**: `TestE2EFlujoCompletoRedReal` (15,5 s), `TestE2EAnimeAV1RedReal` (3,2 s) y los 2 offline; `TestMP4UploadRedReal` **SKIP** (sin `MIAPP_MP4UPLOAD_EMBED`) |
| `npm run check` (frontend) | **0 errores, 0 avisos** (sin cambios de frontend) |

**Archivos tocados:** `internal/scraper/jkanime/{jkanime.go,embeds.go}` (nuevo `embeds.go`), `internal/scraper/animeav1/{animeav1.go,sources.go}`, `internal/app/{app_e2e_test.go,app_nucleo_animeav1_test.go}` (solo tests/fakes), `internal/scraper/jkanime/embeds_test.go` (nuevo), `internal/scraper/animeav1/embeds_test.go` (nuevo), `docs/firmas/{jkanime,animeav1}.md`, `PROGRESO.md`. **Nada fuera de "Puedes tocar"**: `embeds/**`, `stream/**`, `logx/**`, `frontend/**`, `app.go`, `main.go` y `models/**` intactos. (Las líneas CRLF del fantasma T36 se normalizaron a LF en el worktree —sin cambio de contenido— para que `gofmt -l .` salga limpio.)

**Desviaciones:**

- **`stream` no necesitó cambios** (req5): el anti-SSRF es por IP, no por puerto (ver arriba).
- **El guard del token (req6) solo pasa gracias a T62**; sin la redacción del path de `logx`, el E2E de T61 no podía pasar. Por eso T62 se hizo como tarea propia antes de cerrar T61.
- **Checklist req8 sin muestra nueva:** solo había 2 fixtures de episodio (uno por proveedor); ambos con MP4Upload y resuelto a `mp4`, ninguno elegido como mejor. No se buscó un episodio real donde el HLS faltara (fuera de alcance: T61 no debe cambiar comportamiento).
- **Cuerpo de vídeo artificial** en los CDN falsos (no hay fixture de vídeo; no se descargan vídeos, regla 12): se documenta como artificial.
- **Renumeradas las "tareas propuestas" del informe de adds** en este archivo (T60–T62 ya son tareas reales; las propuestas pasan a T63+).

**Pendientes:**

- **Prueba manual del humano con el `.exe`** (T61 lo pide): reproducción real de un MP4Upload **resuelto** —pausa larga y *seek*— por si el token temporal de la URL caduca o se ata a la IP; si el mp4 se niega tras la pausa, quedará como comportamiento esperado (la URL es temporal, no se refrigera).
- Sin tag ni release: no aplica a esta tarea.

---

## T63 — Selector de fuentes: principales primero, iframes como extra

Hecha el **2026-10-15**. Solo frontend (`etiquetas.ts` + `etiquetas.test.ts`, `SelectorFuente.svelte` y `VistaReproductor.svelte` en lo justo). El selector deja de ser una fila plana: los **principales** (tipo `hls`/`mp4`: jkplayer, MediaFire, el HLS de AnimeAV1 y el MP4Upload resuelto de T61) van primero, y los **iframes** (embebidos de terceros, pueden traer publicidad) quedan debajo en una sección plegable cerrada por defecto. Se arregla así el chip ambiguo `Mp4upload` (iframe con anuncios) vs `MP4Upload` (directo sin anuncios) que hacía elegir el equivocado.

**Qué se hizo:**

- **Clasificación por TIPO, nunca por nombre** (`etiquetas.ts`): `esPrincipal` = `hls`\|`mp4`; `esExtra` = `iframe`. `agruparFuentes(fuentes, filtro)` separa ambos grupos conservando el orden del backend dentro de cada uno y etiqueta cada fila.
- **Sufijo de colisión**: cuando el mismo nombre de servidor (comparado **sin mayúsculas**) aparece en los dos grupos, el principal se marca `· directo` y el extra `· iframe`; si no hay colisión, la etiqueta queda como siempre. `etiquetaFuente` recibe un `sufijo` opcional que se añade al final (con calidad/idioma previos intactos: "MP4Upload · SUB · directo").
- **UI** (`SelectorFuente.svelte`): fila de principales + botón plegable `aria-expanded` con el texto exacto "Otras fuentes (embebidas, pueden incluir publicidad)". **Cerrada por defecto**; se despliega sola si la fuente activa es un iframe o si no hay ningún principal. Funciona con teclado (`:focus-visible`), sin depender de hover (P6). El filtro de idioma sigue en `null` (no se activó).
- **Orden y fallback** (`VistaReproductor.svelte`): la fuente que se abre sigue siendo la del backend (índice 0). El salto automático ahora recorre `ordenReproduccion(fuentes)` —el resto de **principales** primero y los **iframes solo al final**—, porque `Nucleo.Fuentes` solo mueve la mejor a cabeza y el mp4 resuelto de T61 se **añade al final** (podía quedar detrás de los iframes). `elegirFuente` reubica `intentos` en ese orden.

**Pruebas (`etiquetas.test.ts`, esbuild de Vite):** **32** comprobaciones (20 previas + 12 nuevas): clasificación por tipo, colisión `Mp4upload` iframe vs `MP4Upload` mp4 con `· directo`/`· iframe` e índices sobre la lista completa, sin colisión (HLS JKAnime vs VOE, SUB/DUB intactos), lista solo de iframes y lista vacía, y `ordenReproduccion` (principales antes que iframes; la del backend siempre primera).

**Verificación visual (Playwright, `localhost:34115`; sin vídeo real, el proxy no es alcanzable desde el navegador —documentado—):**

- **JKAnime "bleach"** (Ep. 1) → principales `jkplayer`, `Mediafire`, `MP4Upload · directo`; extras al desplegar: `Mega`, `Streamwish`, `Vidhide`, `VOE`, `Filemoon`, `Mixdrop`, `Mp4upload · iframe`, `Streamtape`, `Doodstream`.
- **AnimeAV1 "bleach"** (Ep. 1) → principales `HLS · SUB`, `HLS · DUB`, `MP4Upload · SUB · directo`, `MP4Upload · DUB · directo`; extras: `UPNShare · SUB/DUB`, `Voe · SUB/DUB`, `Byse · SUB/DUB`, `MP4Upload · SUB/DUB · iframe`.
- Pliegue **cerrado por defecto** con un principal activo (`aria-expanded="false"`, extras ocultos) en los dos proveedores; el botón abre/cierra alternando `aria-expanded`; **autoapertura** con iframe activo (el activo acaba en iframe porque el proxy da `ERR_CONNECTION_REFUSED`); **foco de teclado** alcanza el botón en 3 `Tab` con `:focus-visible`. A **700 px** no hay desbordamiento horizontal (selector 636 px, etiquetas largas envuelven). Capturas en `.playwright-mcp/t63-selector-{jkanime,animeav1}-700.png`.
- **Consola**: sin errores nuevos de nuestro código (0 coincidencias de `etiquetas`/`SelectorFuente`/`VistaReproductor`). Lo único que aparece es el `favicon.ico` 404 y el `TypeError` de `wails/ipc.js` (ambos ya presentes en la carga inicial, ajenos a T63), los `ERR_CONNECTION_REFUSED` al proxy `127.0.0.1` (limitación del navegador) y mensajes de anuncios **dentro de los iframes de terceros**.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** (los 10 `.go` del fantasma CRLF T36 se normalizaron a LF; `git diff --ignore-cr-at-eol -- '*.go'` **vacío**) |
| `go build ./...` · `go vet ./...` | OK · limpio (sin cambios de Go) |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (mismo recuento; T63 no toca Go) |
| `etiquetas.test.ts` (esbuild + node) | **OK (32 comprobaciones)** |
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | OK (aviso de chunk >500 kB preexistente) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Desviaciones:**

- **`gofmt -l .` marcaba 10 `.go` con el fantasma CRLF (T36)** aunque no se tocó ni una línea de Go; se normalizaron a LF (quitar solo el `\r` final, git no registra cambio real) para dejar la compuerta limpia. Archivos afectados: `internal/app/{app_e2e_test.go,app_nucleo_animeav1_test.go}` y `internal/scraper/{animeav1/{animeav1.go,embeds_test.go,sources.go,sources_test.go},jkanime/{embeds.go,embeds_test.go,get_video_sources_test.go,jkanime.go}}`. Ningún cambio de contenido.
- **Sufijo al final de la etiqueta** (tras `Idioma`/`Calidad`): con idioma queda "MP4Upload · SUB · directo". Es la lectura literal de "sufijo"; sin colisión no se añade nada.
- **Colisión por nombre de servidor sin distinguir mayúsculas** (el caso real JKAnime: iframe `Mp4upload` vs directo `MP4Upload`).
- **Colisión de numeración del informe de adds**: `T63` estaba propuesto como extractor de Voe; el selector de fuentes se queda con `T63` y las propuestas pasan a **T64+** (Voe T64, Byse T65, UPNShare T66, Streamtape T67, purga T68, sandbox T69).
- **Los errores de consola del navegador** (`favicon` 404, `wails/ipc.js` TypeError, `ERR_CONNECTION_REFUSED` del proxy y anuncios dentro de los iframes) son previos/ajenos a T63 (documentados en AGENTS.md); no se introdujo ninguno nuevo.

**Pendientes:**

- **Prueba manual del humano con el `.exe`**: con el proxy real, comprobar que al fallar una fuente el salto recorre los principales antes que los iframes y que el selector se pliega/despliega como se espera; el MP4Upload `· directo` debe reproducirse sin anuncios y sin saltar al `· iframe`.
- Sin tag ni release: no aplica a esta tarea.

---

## Resumen de la v1.3.0 (T64–T67) — detalle largo en `docs/PROGRESO_v1.3.md`

> En T68 se movieron a `docs/PROGRESO_v1.3.md` (texto literal) las notas largas de T64–T67. Aquí queda lo esencial.

### T64 — Lógica pura de ventana de episodios (`ventana.ts` + tests) — resumen

Solo frontend: `frontend/src/ventana.ts` (nuevo, TS puro) exporta las constantes (`FILAS_INICIAL = 5`, `FILAS_TRAMO = 4`, `COLUMNAS_POR_DEFECTO = 4`, `RETARDO_POR_COLUMNA_MS = 50`, `RETARDO_MAXIMO_MS = 300`), las funciones puras `normalizarColumnas`/`tamanoInicial`/`siguienteTamano`/`cubrirIndice`/`retardoEntrada`/`claveVentana` y la memoria por sesión `recordarVisibles`/`recuperarVisibles`/`olvidarVisibles`; ninguna entrada inválida (NaN, Infinity, negativos) lanza. `ventana.test.ts`: **48 comprobaciones** (esbuild + node). Compuerta de Go **no ejecutada** en esa sesión (daemon de Docker parado; Go no se tocó) → repetida en verde en T65.

### T65 — Revelado animado de tarjetas por visibilidad (sin paginar) — resumen

Solo frontend: `frontend/src/revelar.ts` (nuevo), `VistaDetalle.svelte` (envoltorio `.celda`, acción `use:revelar`, keyframes `entra-celda`, medición de columnas) y `TarjetaEpisodio.svelte` (`contain-intrinsic-size: auto 210px → auto 170px`, único cambio exigido por la medición). Un `IntersectionObserver` compartido (threshold 0.15) revela cada tarjeta al entrar, con retardo por columna (`retardoEntrada`); no re-anima ids ya revelados, ni con `prefers-reduced-motion: reduce`, ni sin `IntersectionObserver`. **Mediciones Playwright**: tarjeta 170,39 px reales frente a 210 px reservados (desajuste ~39,6 px/tarjeta, > 20 px → ajuste); 366 tarjetas / 1830 nodos y hasta 1180 (One Piece) **sin jank y 0 saltos de scroll**; JKAnime tarda ~16 s (red, no render). **Decisión: NO paginar** (los T66–T67 originales, paginado / ventana client-side, quedan descartados). Go intacto: **435 `--- PASS`**.

### T66 — Clave de scroll del detalle por proveedor + slug — resumen

Solo frontend: `frontend/src/views.ts` (`claveScroll` del detalle y del reproductor incluyen el proveedor: `detalle:<provider>:<animeID>`) y `views.test.ts` (nuevo, **13 comprobaciones**). Corrige que «bleach» en JKAnime y AnimeAV1 compartieran posición de scroll. Go intacto: **435 `--- PASS`**.

### T67 — Restaurar el scroll del detalle tras la recarga por lotes — resumen

Solo frontend: `frontend/src/VistaDetalle.svelte` (`scrollPendiente` + `intentarRestaurarScroll` reactivo por lote dentro de `requestAnimationFrame`, con cancelación del usuario por gesto). Cierra la observación de T66 (el restaurado se recortaba a 0 si la lista no estaba pintada). Verificación Playwright **6/6**. Go intacto: **435 `--- PASS`**.

### T68 — Cierre documental de la v1.3.0 — bloque de esta sesión

Cierre documental de la **v1.3.0**, solo documentación y versión, sin cambios de comportamiento (fecha **2026-10-16**):

- **`wails.json` (paso 1)**: `info.productVersion` → **`"1.3.0"`** (único cambio del archivo).
- **`CHANGELOG.md` (paso 2)**: entrada **1.3.0** (Añadido: revelado animado de las tarjetas de episodio, respeta `reduced-motion`; Corregido: scroll del detalle separado por proveedor+slug, restauración del scroll al recargar la lista y altura reservada a 170 px). La **v1.2.1 está taggeada** (`v1.2.1`), así que la nota «sin tag» ya no aplica.
- **Archivo histórico (paso 3)**: notas largas de **T64–T67** movidas **texto literal** a `docs/PROGRESO_v1.3.md`; aquí solo resúmenes. Incluye la decisión de no paginar con las mediciones de T65.
- **`ventana.ts` (paso 4)**: se documenta (en `docs/PROGRESO_v1.3.md` y en este bloque) que del módulo **solo se usa `retardoEntrada`** y que el resto queda disponible para una posible **paginación futura**; **NO se borró código ni se tocó `frontend/**`** (regla "NO debes tocar" de T68, coherente con «no se toca frontend»).
- **`AGENTS.md` y `docs/DESARROLLO.md` (paso 5)**: estado **T1–T68 / v1.3.0 preparada**, rango de "Trabajo con agentes" hasta T68 y mención de `revelar.ts`, `ventana.ts` y sus tests.
- **`README.md` (paso 6)**: línea en funciones (animación de episodios al aparecer).
- **`docs/BITACORA.md` (paso 7)**: filas T64–T68 y desviaciones vigentes.
- **Pendientes activos (paso 8)**: verificación con el `.exe` de T65/T67, release de la v1.2.1/v1.3.0 por el humano e idea futura (chips de rango).
- **Archivos tocados**: `PROGRESO.md`, `CHANGELOG.md`, `AGENTS.md`, `README.md`, `docs/DESARROLLO.md`, `docs/BITACORA.md`, `docs/PROGRESO_v1.3.md` (nuevo) y `wails.json` (solo `info.productVersion`). **Nada tocado fuera de "Puedes tocar"**: `*.go`, `frontend/**`, `tareas/00_REGLAS.md` y `docs/firmas/**` intactos.

### Comprobación — 2026-10-16, todo en verde

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (`miapp` sin tests de paquete) |
| recuento `^--- PASS` (`-count=1 -v`) | **435** — idéntico al de T63/T65/T66/T67 (Go intacto) |
| `npm run check` (frontend) | **svelte-check found 0 errors and 0 warnings** |
| `wails build -platform windows/amd64` | **OK** → `build/bin/miapp.exe` (compila con `productVersion` 1.3.0) |
| `git status --porcelain` | solo los archivos de "Puedes tocar" |
| `git diff --stat -- '*.go'` | **vacío** (ningún `.go` con cambios reales) |

**No se creó tag ni release** (lo hace el humano). Detalle largo de T64–T67 en `docs/PROGRESO_v1.3.md`; sesión en `docs/BITACORA.md`.


---

## Resumen de la v1.4.0 (T69–T89) — detalle largo en `docs/PROGRESO_v1.4.md`

> En T89 se movieron a **`docs/PROGRESO_v1.4.md`** (texto literal) las notas largas de **T69–T88**,
> junto con la **tabla de correspondencia** con el plan inicial (T83–T93). Aquí queda **una línea por
> tarea** y el bloque de cierre (**T89**).

| Tarea | Una línea |
|---|---|
| T69 | Auditoría de UI base (**solo lectura**): inventario, literales CSS, 24 capturas a 360/768/1280, métricas por vista y paleta de Stitch → `docs/investigacion/ui_base.md`; **sin cambios de código**. |
| T70 | Tokens de diseño en `style.css`: un bloque `:root` **añadido** con 65 variables (colores, superficies, bordes, texto, marca, semánticos, radios, sombras, blur, movimiento, espacio, táctil, gutter y capas), **sin cambio visual** → `docs/investigacion/ui_tokens.md`. |
| T71 | Base global móvil: fuentes locales **Inter/Outfit** (subconjunto latin, peso variable), escala tipográfica, safe-area, `viewport-fit=cover`, **corrección del fondo dividido**, `:focus-visible` y ajustes táctiles; herramienta `docs/investigacion/ui/verificar_ui.mjs`. |
| T72 | Primitivas de UI e iconos: **6 componentes nuevos** (`Icono`, `Btn`, `Chip`, `Insignia`, `Esqueleto`, `EstadoVista`) con tokens, **sin migrar** ningún componente existente. |
| T72b | Puente de color: los **15 `.svelte` existentes** dejan de escribir literales de color/radio/sombra y `--cad-*` y pasan a los tokens de T70 (**solo dentro de `<style>`**) → `docs/investigacion/ui_mapeo_color.md`. |
| T73 | Barra de navegación inferior móvil (4 destinos, `< 900 px`) que reutiliza el **cajón de historial**; en modo ancho no cambia nada. |
| T73b | Cajón **opaco**: `.barra` pasa a `--bg-1` y el cajón gana `--sh-panel` (deja de traslucir el contenido y el pie). |
| T74 | `CabeceraVista.svelte` (nuevo): cabecera unificada de **detalle y reproductor** («Atrás», título con elipsis, estrella), fija en el detalle en modo estrecho. |
| T75 | Buscador y **pestañas del Home** en modo estrecho: lupa, botón ✕, `h1` del buscador; en ≥ 900 px no cambia un píxel. |
| T76 | **Filas (carruseles) y tarjeta del Home**: sangrado al borde en móvil, flechas de 44 px, marco/radio/sombra de tokens; geometría de escritorio intacta (263,8 px). |
| T77 | Fila **«Continuar viendo»** en el Home móvil (3–10 entradas, 240×86 px, barra de progreso), sin lógica nueva (reutiliza `historial.ts`/`retomar.ts`). |
| T78 | **Rejilla de resultados de búsqueda** y sus estados (cargando con 6 esqueletos, error con «Reintentar», vacío), 2/3/auto-fill columnas. |
| T79 | **Ficha de detalle**: portada enmarcada, metadatos en insignias, géneros en una línea con scroll y **sinopsis plegable** con «Ver más» en estrecho. |
| T80 | `BarraAccion.svelte` (nuevo): CTA fija «Continuar / Reiniciar» pegada a la barra inferior en el detalle estrecho. |
| T81 | **Tarjeta de episodio** (miniatura 16:9, insignia, franja de progreso) y **rejilla** a 2/3 columnas en estrecho, con **alto constante** por rango (146/179/170,38 px). |
| T82 | **Orden ascendente/descendente** de episodios (botón con `aria-pressed`); solo presentación, la carga y la CTA no cambian. |
| T83 | **Layout del reproductor** (marco 16:9 con borde/sombra) y **selector de fuentes** con chips táctiles en una fila con scroll horizontal en estrecho. |
| T84 | Chip **«Siguiente ▸»** del reproductor: de ~34 a **44 px**, recolocado y con el overlay por encima. |
| T85 | **Favoritos** (rejilla 2/3/auto-fill con estados unificados), **toast** opaco y accesible, y **verificación del deslizar** en el cajón. |
| T86 | **Estados restantes** unificados (Home, detalle, reproductor, cajón), token `--scrim`/`--scrim-strong` y **barrido de px** a tokens. |
| T87 | **Movimiento y rendimiento**: `prefers-reduced-motion` global, respaldos de `prefers-reduced-transparency`, `decoding="async"` y medición A/B (0 frames > 50 ms) → `docs/investigacion/ui_rendimiento.md`. |
| T88 | **Accesibilidad AA** en 11 vistas: 1 `h1` por vista, 0 saltos, 0 fallos de contraste (normal/foco/hover), 0 interactivos sin nombre, anillo de foco en todas las paradas → `docs/investigacion/ui_a11y.md`. |
| T89 | **Cierre de la v1.4.0**: `lang="es"`, trampa de foco del cajón (`inert`), retirada de `--cad-*` y de Nunito, `productVersion` 1.4.0 y cierre documental. |

---

## T89 — Cierre de la v1.4.0 (T69–T89) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-20**.
Cierre documental y de limpieza, más tres arreglos pequeños heredados de T88. **Sin cambios de
diseño**; `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts` y `ui.ts` **intactos**.

### Qué se hizo

- **`lang` (paso 1)**: `frontend/index.html` → `<html lang="es">` (única línea tocada del archivo).
  El `<meta name="referrer" content="no-referrer">` y el viewport (`width=device-width,
  initial-scale=1, viewport-fit=cover`) se conservan; comprobado con `grep` sobre el `<head>`.
- **Trampa de foco del cajón (paso 2)**: con el cajón abierto en modo estrecho, `main.area-principal`
  y la barra inferior llevan el atributo **`inert`**. El cajón y su velo **NO** se inertizan: los
  renderiza `BarraLateral` dentro de `.app-shell` y son **hermanos** de `main.area-principal`
  (el caso que la ficha advertía: si el cajón fuera hermano del contenido inertizado, se
  inertizaría a sí mismo → **no se aplica**, se documenta como deuda). Verificado en el navegador:
  - con el cajón abierto, `main` y la zona de la barra tienen `inert`, y el cajón y el velo **no**;
  - **14 `Tab` seguidos**: las 14 paradas caen **dentro** del cajón; el botón de la barra **deja de
    ser enfocable** (`focus()` no cuela) y el buscador tampoco es alcanzable;
  - `Escape` **cierra** el cajón y **devuelve el foco** al destino «Historial»;
  - al cerrar, el `inert` **desaparece** y la barra vuelve a ser enfocable.
  - **Implemación**: Svelte 5 **no reenvía atributos a un componente** (`<BarraInferior inert={…}>`
    falla el `svelte-check` con *"'"inert"' does not exist in type …"*) y **no admite `use:` sobre
    componentes** (*"This type of directive is not valid on components"*). Por eso el `inert` de la
    barra va en un **envoltorio sin caja** (`<div class="zona-barra-inferior">` con
    `display: contents`, el mismo patrón que ya usa `.raiz`) dentro de `App.svelte`. Comprobado que
    `display: contents` **no** anula el `inert`.
- **Limpieza del legado (paso 3)**: `grep` de `--cad-`, `Nunito` y `nunito` en `frontend/src`,
  `frontend/index.html` y docs vivas → **0 usos** fuera de las propias definiciones. Se retiraron:
  las **8 variables `--cad-*`** y el `@font-face "Nunito"` de `frontend/src/style.css`, y los
  archivos `frontend/src/assets/fonts/nunito-v16-latin-regular.woff2` y `OFL.txt` (la carpeta
  `assets/fonts/` desaparece; `assets/images/` se queda). **Constancia de licencias**: Inter y Outfit
  siguen viniendo de los paquetes npm `@fontsource-variable/inter` y `@fontsource-variable/outfit`
  (ambos **SIL OFL 1.1**; el `LICENSE` completo va en cada paquete) y se anota en el comentario de
  `style.css`. **Peso de `frontend/dist`: 864 433 B → 845 393 B (−19 040 B)**: no aumenta.
- **Deriva de scroll (paso 4)**: medida con el código actual y con **T87+T88 revertidos**
  (`git checkout` del commit de T86 en el worktree; el árbol quedó como estaba). **Ambos valores:
  +52 px** (2743 → 2795), idénticos en 3 pasadas: la deriva **no la introducen T87 ni T88**, es
  previa. **No se corrige** (queda en «Pendientes activos»).
- **Versión y changelog (paso 5)**: `wails.json` → `info.productVersion` **"1.4.0"** (único cambio
  del archivo) y entrada **1.4.0** en `CHANGELOG.md` (Añadido / Cambiado / Corregido, mencionando
  que el escritorio conserva su geometría).
- **Archivo histórico (paso 6)**: notas largas de **T69–T88** movidas **texto literal** a
  **`docs/PROGRESO_v1.4.md`** (con la tabla de correspondencia del plan inicial); aquí queda el
  resumen de una línea por tarea.
- **Documentos vivos (paso 7)**: `AGENTS.md`, `docs/DESARROLLO.md`, `README.md`, `docs/BITACORA.md`,
  `tareas/00_REGLAS.md` (árbol del frontend) y `docs/investigacion/ui_tokens.md` (`--cad-*` marcado
  como **retirado**).
- **Pendientes activos (paso 8)**: prueba manual con el `.exe`, release de la v1.4.0 (humano), deriva
  de scroll, «Episodios siguientes», pantalla completa, extractores T94–T99 y `Type` de
  `models.Anime`.

### Verificación — modos de `verificar_ui.mjs` y `a11y_auditoria.mjs` re-ejecutados

Todos los modos se re-ejecutaron contra `http://localhost:34115` (dev server de Wails en Docker, con
red real). **Tabla modo → comprobaciones OK/total:**

| Modo | OK/total | Fallos |
|---|---:|---|
| `shell` | **21 / 23** | `detalle → Favoritos deja el Home`; `reduced-motion: sin transición en la barra` |
| `cabecera` | **36 / 39** | `360: título largo en una línea`; `360: título largo recortado`; `reduced-motion: sin transición en la cabecera` |
| `buscador` | **60 / 61** | `reduced-motion: sin transición en el limpiar` |
| `filas` | **62 / 63** | `reduced-motion: tarjeta sin transición` |
| `continuar` | **45 / 46** | `F: reduced-motion sin transición en la tarjeta` |
| `busqueda` | **59 / 60** | `G: reduced-motion sin transición en la tarjeta` |
| `ficha` | **105 / 106** | `K: reduced-motion sin transición en "Ver más"` |
| `cta` | **58 / 59** | `G: reduced-motion sin transición en la CTA` |
| `episodios` | **46 / 46** | — |
| `orden` | **88 / 89** | `F: reduced-motion sin transición en el botón` |
| `reproductor` | **52 / 54** | `reduced-motion: chips sin transición`; `reduced-motion: plegable sin transición` |
| `siguiente` | **58 / 59** | `reduced-motion: sin transición` |
| `favoritos` | **61 / 62** | `reduced-motion: botones sin transición` |
| `estados` | **102 / 102** | — |
| `rendimiento` | OK | 0 frames > 50 ms; geometría 263,8 / 146 / 179 / 170,38; `overflowH = 0`; reduced-motion 366/366 celdas; **T66/T67 con reduce: OK (28984 → 28984)**; 3 reglas `@supports not (backdrop-filter)` |
| `a11y_auditoria.mjs` | OK | **0** fallos de contraste (normal, foco y hover), **0** interactivos sin nombre, **0** saltos de jerarquía, **1 `h1`** en las 11 vistas, **0** paradas sin anillo, `idiomas: ["es"]`, 0 errores |

**Los 15 fallos son PREEISTENTES (no los introduce T89).** Se comprobó re-ejecutando `shell` y
`cabecera` con los cambios de T89 guardados en un `git stash` (es decir, en el estado T88): los
**mismos 5 fallos** salen idénticos, y los 10 restantes son el mismo patrón
`reduced-motion → "1e-05s"` (la regla global de T87 deja `transition-duration: 0.01ms`, y solo el
modo `estados` se actualizó en T87 para aceptar `transition-property: none`; los demás modos siguen
exigiendo `0s` literal).

**Foco del cajón (medido por `a11y_auditoria.mjs`):** `focoInicial` = «Cerrar barra», `dentro` =
**true**, `escapeCierra` = **true**, `focoDevuelto` = **true**. Las **2 fugas** que quedan ya **no
son el contenido de fondo** (en T88 eran el buscador y la barra inferior): son el `<body>`
(respaldo del documento al final de la lista de tabulables) y el **velo** («Cerrar historial»), que
es un `<button>` y por la ficha **no debe inertizarse**. Para llegar a 0 fugas habría que quitar el
velo del orden de tabulación (`tabindex="-1"` en `BarraLateral.svelte`) — **fuera de "Puedes
tocar"** de T89 → queda como deuda (ver «Pendientes activos»).

### Capturas finales

`docs/img/v1.4/cierre/`: `descubrir_{360,768,1280}.png`, `busqueda_{360,768,1280}.png`,
`detalle_animeav1_{360,768,1280}.png`, `detalle_jkanime_{360,768,1280}.png`,
`reproductor_{360,1280}.png`, `cajon_360.png`, `cajon_360_inert.png` (cajón abierto con el `inert`
visible), `cajon_1280.png` (a 1280 el historial es la barra lateral fija, no un cajón) y
`resumen.json` (`overflowH = 0` en las 12 vistas×anchos). `docs/img` está ignorado por
`.gitignore`, como en las tareas anteriores.

### Compuerta completa (2026-10-20, todo en verde)

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | **OK** |
| `go test -race -count=1 -v ./...` | **12 paquetes `ok`**; **435 `--- PASS`** de nivel superior, **0 FAIL** (Go intacto) |
| `npm run check` (frontend) | **svelte-check found 0 errors and 0 warnings** |
| `npm run build` | **OK** (176 módulos; aviso preexistente de chunk > 500 kB) |
| peso de `frontend/dist` | **864 433 B → 845 393 B (−19 040 B)** (Nunito fuera) |
| tests `.ts` (esbuild + node) | **32 / 48 / 13** OK (`etiquetas`, `ventana`, `views`) |
| `wails build -platform windows/amd64` | **OK** → `build/bin/miapp.exe` (13,7 MB) en 18,7 s |
| `git diff --stat -- frontend/wailsjs` | **vacío** (bindings idénticos) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `git diff --stat -- '*.go'` | **vacío** (ningún `.go` con cambios reales) |
| `git status --porcelain` | solo archivos de «Puedes tocar» (+ `tareas/T89.md`, que ya estaba sin trackear antes de la sesión) |

**No se creó tag ni release** (los hace el humano). Detalle largo de T69–T88 en
`docs/PROGRESO_v1.4.md`; sesión en `docs/BITACORA.md`.

### Desviaciones y notas

- **Svelte 5 no reenvía atributos a los componentes**: `inert` no se puede pasar como atributo a
  `<BarraInferior>` (el `svelte-check` lo rechaza) ni con `use:` sobre un componente. Solución
  adoptada: envoltorio **sin caja** (`display: contents`) en `App.svelte`. `BarraLateral.svelte` y
  `BarraInferior.svelte` **no se tocaron**.
- **Las «fugas» del cajón no son 0** (son 2: `<body>` y el velo) y **no se pueden bajar a 0** sin
  tocar `BarraLateral.svelte` (`tabindex="-1"` en el velo), que está fuera de «Puedes tocar». Lo que
  sí se cumple es el objetivo: **el contenido de fondo (buscador, barra inferior, detalle) ya no es
  alcanzable con `Tab`** (en T88 lo era).
- **`docs/investigacion/ui_a11y.md` y `a11y_auditoria.mjs` no se tocaron** (no figuran en «Puedes
  tocar»): el informe de T88 sigue diciendo `lang="en"` y «el cajón no atrapa el foco», que es la
  foto de antes del cierre; el estado actual está en este bloque y en `docs/PROGRESO_v1.4.md`.
- **Los 15 fallos de los modos son preexistentes** (ver arriba): se compararon contra el estado T88
  con `git stash` y son idénticos.
- **`docs/investigacion/ui_base.md` y `ui_mapeo_color.md`** (informes de auditoría de T69 y T72b)
  siguen describiendo el estado **anterior** (mencionan las `--cad-*` y Nunito): son documentos
  históricos de investigación, no documentos vivos; no se editan.
- **Los informes de `docs/investigacion/`** que sí son vivos (`ui_tokens.md`) se actualizan.
- **`tareas/T89.md`** ya estaba en el árbol de trabajo sin trackear; no se creó aquí.
- **`docs/img`** está ignorado por `.gitignore`, como en las tareas anteriores.
