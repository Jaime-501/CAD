    # PROGRESO — Comfort Anime Desktop (T01–T88 hechas)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivos históricos**: el estado de **T01–T29 + T30** (v1.0) está en **`docs/PROGRESO_v1.0.md`**, el de **T30–T35** (AnimeAV1, v1.1.0, con sus desviaciones por tarea, fixtures y pruebas manuales) en **`docs/PROGRESO_v1.1.md`** y el de **T39–T58** (v1.2.0: Home, historial, favoritos, "Siguiente" y sus bugs; incluye los cierres T56/T59) en **`docs/PROGRESO_v1.2.md`**, y el de **T64–T67** (v1.3.0: `ventana.ts`, revelado animado y scroll del detalle) en **`docs/PROGRESO_v1.3.md`**. Aquí solo queda lo vigente. Estado del producto: **v1.2.1 PUBLICADA** (tag `v1.2.1`; ciclo **T39–T59**: v1.2.0 publicada en T56–T58, con T53b y T57; los dos bugs visuales son **T58**; el cierre documental —post-v1.2.0 y de la v1.2.1— es **T59**) y **v1.3.0 PREPARADA** — serie **T64–T68** en la rama **`feature/episodios-incrementales`** (episodios incrementales + revelado animado; cierre documental en **T68**). La **prueba manual con el `.exe` de la v1.2.0** y la **release** las hizo el humano; T57 se probó con el `.exe` sin incidencias. `wails.json` → `productVersion` **1.3.0**: **sin tag ni release aún** (los hace el humano). **Numeración (colisión resuelta en T59):** `tareas/T58.md` = bugs visuales (fila/bloque T58); el cierre documental pasó de T58 a **T59** (sin borrar contenido). **Nueva serie v1.4.0 (rediseño y enfoque móvil, rama `feature/rediseno-movil`)**: comienza con **T69** (auditoría de UI base, solo lectura) **hecha** — informe en `docs/investigacion/ui_base.md`, capturas en `docs/img/v1.4/base/` y scripts en `docs/investigacion/ui/`; **sin cambios de código**. Le sigue **T70** (tokens de diseño en `style.css`: solo variables nuevas, sin cambio visual) **hecha** — `docs/investigacion/ui_tokens.md` y `docs/investigacion/ui/contraste.mjs`; los `--cad-*` quedan como legado hasta T93— y **T71** (base global: fuentes locales Inter/Outfit, viewport con safe-area, corrección del fondo dividido, foco visible y ajustes táctiles) **hecha** — ver bloque al final y **T72** (primitivas de UI e iconos: `Icono`, `Btn`, `Chip`, `Insignia`, `Esqueleto` y `EstadoVista`, 6 componentes nuevos, sin migrar ninguno existente) **hecha** — ver bloque al final.

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

T01–T29 = v1.0 (`docs/PROGRESO_v1.0.md`) · T30–T35 = v1.1.0 AnimeAV1 (`docs/PROGRESO_v1.1.md`) · T36 = cierre de la v1.1.0 · T37 = portabilidad Android · T38 = cierre documental de T37.
T39–T57 = v1.2.0 (Home, historial lateral, favoritos, "Siguiente" y su estado discreto; **incluye T53b y T57**); plan en `tareas/v1.2_PLAN.md`, detalle **T39–T58 + cierres T56/T59** archivado en `docs/PROGRESO_v1.2.md`, con resúmenes al final de este archivo.
La **v1.2.0 está PUBLICADA** (prueba manual con el `.exe` y release hechas por el humano). La **v1.2.1** (T58 bugs + T59 cierre documental) está **preparada**: `productVersion` 1.2.1, sin tag ni release aún — los hace el humano.

**v1.3.0 — episodios incrementales** (rama `feature/episodios-incrementales`): serie **T64–T69** para mostrar los episodios del detalle de forma incremental y con animación de entrada. **T64** (lógica pura de `ventana.ts` + tests), **T65** (revelado animado por visibilidad, sin paginar), **T66** (clave de scroll del detalle por proveedor + slug) y **T67** (restauración del scroll del detalle cuando la lista se recarga) hechas; **T68** (cierre documental de la v1.3.0) hecho: `productVersion` **1.3.0**, **sin tag ni release** (los hace el humano). Los **T66–T67 originales** (paginado / ventana client-side) quedan **descartados** por las mediciones de T65 (sin jank y con 0 saltos de scroll; el coste real es de red, ya cubierto por el mostrado por lotes); **T69 pendiente** de la serie. Notas largas de T64–T67 archivadas en `docs/PROGRESO_v1.3.md`.

**v1.4.0 — rediseño y enfoque móvil** (rama `feature/rediseno-movil`): serie nueva que empieza con **T69** (auditoría de UI base, **solo lectura**: inventario, literales CSS, 24 capturas a 360/768/1280, métricas por vista y extracción de la paleta de los 4 HTML de Stitch), hecha y documentada en `docs/investigacion/ui_base.md`; los scripts de medición viven en `docs/investigacion/ui/` y las capturas en `docs/img/v1.4/base/`. **No se tocó código** (`frontend/**`, `*.go`, `package.json`, `wails.json` intactos). Le sigue **T70** (tokens de diseño: un bloque `:root` **añadido** a `frontend/src/style.css` con las variables del rediseño —colores, superficies, bordes, texto, marca, semánticos, radios, sombras, marco de póster, blur, movimiento, espacio, táctil, gutter y capas—, **sin modificar ninguna línea existente** y sin cambio visual; documentadas en `docs/investigacion/ui_tokens.md` con la tabla de contraste de `docs/investigacion/ui/contraste.mjs`; las `--cad-*` quedan como legado hasta **T93**) y **T71** (base global móvil: fuentes locales **Inter**/Outfit por subconjunto latin, escala tipográfica y safe-area como tokens, `viewport-fit=cover`, corrección del fondo dividido, `:focus-visible` y ajustes táctiles; herramienta reutilizable `docs/investigacion/ui/verificar_ui.mjs`). Le sigue **T72** (primitivas de UI e iconos: 6 componentes **nuevos** — `Icono`, `Btn`, `Chip`, `Insignia`, `Esqueleto` y `EstadoVista` — con tokens, sin migrar ningún componente existente; el arnés de galería fue temporal) y **T72b** (puente de color: los 15 `.svelte` existentes dejan de escribir literales de color/radio/sombra/`--cad-*` y pasan a los tokens de T70 —solo dentro de `<style>`—, documentado en `docs/investigacion/ui_mapeo_color.md`; sin cambio de geometría). Le sigue **T73** (barra de navegación inferior móvil y cajón de historial: en modo estrecho —< 900 px— la hamburguesa se sustituye por una barra fija de 4 destinos —Descubrir, Buscar, Favoritos, Historial— con tokens, safe-area y objetivos táctiles de 44 px, reutilizando el cajón existente; en modo ancho, sin cambios). Después siguieron **T74** (cabecera compacta de detalle/reproductor), **T75** (buscador y pestañas del Home), **T76** (filas y tarjetas del Home), **T77** (fila «Continuar viendo»), **T78** (rejilla de resultados de búsqueda y estados) y **T79** (ficha de detalle: portada, metadatos, géneros y sinopsis) y **T80** (barra de acción fija «Continuar / Reiniciar» del detalle en estrecho); cada una tiene su bloque al final de este archivo. Después se completaron **T81–T87** (tarjeta y rejilla de episodios, orden, reproductor, chip «Siguiente», favoritos/toasts, estados y movimiento/rendimiento); sus bloques están al final. **T88** (accesibilidad AA) **hecha**: 0 fallos de contraste/nombres/anillo en 11 vistas, una `h1` por vista, foco del cajón y deuda de `lang="en"` reportados en `docs/investigacion/ui_a11y.md`; sin cambios de geometría. Los `--cad-*` siguen **definidos** en `style.css` (sin uso) hasta **T93**.

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

> Solo lo que una tarea futura debe resolver. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`; los de T30–T35 en `docs/PROGRESO_v1.1.md`; los de T39–T58 en `docs/PROGRESO_v1.2.md`; los de T64–T67 en `docs/PROGRESO_v1.3.md`.

- **(4) Verificación visual de los avisos de error** (heredado de la v1.0, detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos"): los avisos de error que ningún test provoca (hoy: el aviso `episodesError` con un fallo real del backend). Los heredados (1) título de la ventana, (2) marca de tiempo y (3) borrado del historial están **cerrados** (T56, T40/T41 y T51–T52).
- **Verificación con el `.exe` de T65/T67** (revelado animado y restauración del scroll del detalle): el flujo navegador no ejercita el vídeo real (proxy local, AGENTS.md) ni todos los ciclos de recarga; T65 y T67 se verificaron con Playwright (8/8 y 6/6) y por lógica+log. Probar con el `.exe` en la próxima release (v1.3.0 o posterior). Los **fixes T58** (carátula al retomar y contador de favoritos) se dieron por cerrados con la **v1.2.1 publicada**.
- **Release de la v1.3.0** (tag + release): la hace el humano. La **v1.2.1 ya está taggeada** (`v1.2.1`); `productVersion` está en **1.3.0** a la espera de tag/release.
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

## Auditoría de UI base de la v1.4.0 (T69) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**, commit `fc34a64`. T69 es **de solo lectura sobre el código**: nada de `frontend/**`, `*.go`, `package.json` ni `wails.json` tocado. Fecha **2026-10-16**. Informe completo en **`docs/investigacion/ui_base.md`**.

- **Inventario (paso 1)**: 16 `.svelte` (3 895 líneas) + `style.css` (42) + 17 `.ts`; fuentes en `frontend/src/assets/fonts/nunito-v16-latin-regular.woff2` (cargada por `@font-face` en `style.css`) y logo en `assets/images/logo-universal.png`. Solo hay **1 fuente (Nunito)** y **ningún set de iconos**.
- **Literales (paso 2)**: **82 hex**, 9 `rgb(a)`, 0 `hsl`, 4 `box-shadow`, 47 `border-radius`, **0 `backdrop-filter`**, 1 `blur()`, 31 anchos/altos en px, 19 `:hover`, 9 `@media`. **Único breakpoint con px: `max-width: 640px`** (1, en `VistaDetalle`); los otros 8 `@media` son `prefers-reduced-motion` (6) y `(hover: none)` (1). `style.css` ya define 8 variables `--cad-*` (fondo `#151515`, superficie `#1e1e24`, borde `#3f3f46`, texto `#fff`, texto suave `#a1a1aa`, acento `#4ade80`, peligro `#f87171`, radio `8px`).
- **Capturas (paso 3)**: **24 PNG** en `docs/img/v1.4/base/` a 360×740, 768×1024 y 1280×800 (Descubrir, búsqueda, favoritos vacío/datos, detalle AnimeAV1/JKAnime, reproductor con HLS y con «otras fuentes» abierto, barra lateral expandida/contraída y cajón móvil).
- **Métricas (paso 4)**: `metricas.json` + `resumir_metricas.mjs`. **`overflowH = 0`** en todas las vistas/anchos; rejilla de episodios 1/3–4 columnas; **tarjeta de episodio = 170,39 px** ligada a `contain-intrinsic-size: auto 170px`; **38 objetivos táctiles < 44 px** (`.papelera` 24×24, `.estrella`/`.flecha`/`.alternar` 32×32, `.hamburguesa` 38×38, `.pestana` 39 px de alto…); muchos textos < 12 px en las rejillas de episodios.
- **Paleta de Stitch (paso 5)**: de los 4 HTML (`stitch_home`, `stitch_detalle`, `stitch_reproductor_1`, `stitch_reproductor_2`) extraídos **solo valores**: fondo `#090d16`/`#090a0f`/`#0a0d12`, acento esmeralda `#10b981`/`#059669`/`#34d399` (más teal `#14b8a6` y neón `#22c55e`), superficies *glass* `rgba(15,23,42,.6)` con borde `rgba(255,255,255,.08)`, glow `rgba(16,185,129,.45)`, sombras `glow-emerald`/`card-glass`, tipografías **Inter + Outfit** (y Noto Serif JP), transiciones 0,25–0,35 s `cubic-bezier(.16,1,.3,1)` y animaciones `glow-pulse`/`float-slow`. No se copió marcado ni clases Tailwind.
- **Acoplamientos (paso 6)**: altura fija de tarjeta de episodio (T64/T65), clave de scroll por proveedor+slug (T66/T67), P4 (wailsjs solo en `api.ts`), clases atómicas `s-<hash>` de Svelte, único breakpoint 640 px, colores duplicados variable/literal y ausencia de glassmorphism/iconos en el código actual.
- **Entregables**: `docs/investigacion/ui_base.md`, `docs/investigacion/ui/{contar_literales,medir_ui,resumir_metricas}.mjs`, `docs/investigacion/ui/metricas.json` y `docs/img/v1.4/base/*.png` (24).

### Comprobación — 2026-10-16, todo en verde (T69)

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go vet ./...` | limpio |
| `go build ./...` | OK |
| recuento `^--- PASS` (`go test -race -count=1 -v ./...`) | **435** (0 `--- FAIL`) — idéntico al de T63/T65–T68 |
| `npm run check` (frontend) | **svelte-check found 0 errors and 0 warnings** |
| `git status --porcelain` | solo `docs/img/v1.4/` y `docs/investigacion/ui/` (nuevos, sin trackear) |

**Sin cambios de código**; no se creó tag ni release. Próxima tarea de la serie v1.4.0: pendiente de definir.

---

## T70 — Tokens de diseño en `style.css` (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Añade el **sistema de tokens** del rediseño como **única fuente de verdad**, sin que ningún componente lo consuma todavía (eso llega con **T72b** y las tareas de vista) y **sin cambio visual**. Solo se **añaden** líneas a `frontend/src/style.css`: las variables `--cad-*` existentes **no se modifican** (quedan marcadas `/* legado: se retira en T93 */`).

**Qué se hizo:**

- **`frontend/src/style.css`** (solo adición): un bloque `:root` nuevo y comentado por grupos con los 65 tokens (Fondos, Superficies, Bordes, Texto —con el comentario obligatorio en `--text-4`—, Marca —incluidos `--grad-brand`/`--grad-chip-active` con `var()`—, Semánticos, Radios, Sombras —incluido `--ring` con `var()`—, Marco de póster, Blur, Movimiento, Espacio `--sp-1..8`, Táctil, Gutter y Capas), más las consultas `@media (max-width:767px){--blur-lg:14px}`, `@media (min-width:480px){--gutter:20px}`, `@media (min-width:768px){--gutter:24px}` y `@media (min-width:1024px){--gutter:32px}`. La línea `/* legado: se retira en T93 */` se insertó **dentro** del `:root` existente (línea nueva, ninguna borrada).
- **`docs/investigacion/ui_tokens.md`** (nuevo): tabla token → valor → uso previsto, breakpoints oficiales **480/768/1024** documentados como números (las variables CSS no resuelven dentro de `@media`), la regla «desde T72b, ningún componente escribe colores, radios ni sombras literales» y la tabla de contraste.
- **`docs/investigacion/ui/contraste.mjs`** (nuevo): script Node **sin dependencias** que calcula la relación WCAG de `--text-1..4`, `--brand-400` y `--danger` sobre `--bg-0` y `--on-brand` sobre `--brand-500`, y marca AA (≥ 4.5:1).

**Verificación de tokens (Playwright, `localhost:34115`, resultado real):** `getComputedStyle(document.documentElement).getPropertyValue()` devuelve **valor no vacío en los 65 tokens** (`vacios: []`); muestra: `--bg-0=#090d16`, `--grad-brand=linear-gradient(90deg, #10b981, #2dd4bf)`, `--ring=0 0 0 2px #090d16, 0 0 0 4px #34d399`, `--sp-8=32px`, `--tap=44px`, `--text-4=#64748b`. Breakpoints: **`--blur-lg` = `14px` a 360 px y `28px` a 1280 px**; `--gutter` = `16px` a 360 px y `32px` a 1280 px.

**Capturas antes/después (1280×800, `prefers-reduced-motion: reduce`, espera al revelado de T65):**

- **Detalle (animeav1, bleach)**: `detalle_antes.png` vs `detalle_despues.png` → **1 canal distinto de 3 072 000** (delta 1, ruido de render). La página sí varía entre ejecuciones por las imágenes de red: `detalle_despues.png` vs una repetición posterior difiere en ~21 000 canales (≈ 7 031 px) **con y sin** el cambio, y dos capturas consecutivas de la misma sesión son **idénticas** → la diferencia no la introduce el CSS.
- **Reproductor**: `reproductor_antes.png` vs `reproductor_despues.png` → **byte-idénticas** (`0` canales distintos). El estado se capturó una vez asentado (fuente embebida 12 s), estable.

**Contraste (salida real de `contraste.mjs`):** `--text-1` **17.74:1**, `--text-2` **13.09:1**, `--text-3` **7.58:1**, `--text-4` **4.08:1 (NO AA)**, `--brand-400` **10.11:1**, `--on-brand`/`--brand-500` **7.95:1**, `--danger` **7.02:1**. Pasan AA todos salvo **`--text-4`**, que es el token **decorativo previsto** (el propio `style.css` lo comenta); **`--text-3` y `--brand-400` sí pasan**, así que no hay que reportar fallo de esos. No se cambió ningún valor.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `git diff --numstat -- frontend/src/style.css` | **126 añadidas, 0 borradas** |
| verificación de tokens (Playwright) | **65/65 no vacíos**; `--blur-lg` 14px@360 / 28px@1280 |
| capturas antes/después | detalle **1 canal** (delta 1, ruido); reproductor **byte-idénticas** |
| `npm run check` (frontend) | **svelte-check found 0 errors and 0 warnings** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `*.test.ts` (esbuild + node) | **etiquetas 32 · ventana 48 · views 13** (todas OK) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go build ./...` | limpio · OK |
| `go test -race -count=1 -v ./...` | **11 paquetes `ok`**; recuento `^--- PASS` = **435**, `^--- FAIL` = **0** (Go intacto, idéntico a T63/T65–T69) |

**Desviaciones:**

- **No se versionó un script Playwright de verificación de tokens** (no figura en "Puedes tocar"): la comprobación de tokens y las capturas se hicieron con Playwright **vía el MCP** (`localhost:34115`) y la comparación antes/después con un decodificador PNG en Node; el script de diff y las capturas quedaron en el directorio temporal fuera del repositorio. El único script nuevo versionado es `contraste.mjs` (sí autorizado).
- **El detalle no es byte-idéntico** (1 canal de 3 072 000, delta 1) por ruido de render; se demuestra que no lo causa el CSS (variación run-to-run con y sin el cambio, y capturas consecutivas idénticas). El reproductor sí es byte-idéntico.
- **`--text-4` no cumple AA (4.08:1)** por diseño: es decorativo (comentario obligatorio en `style.css` y en `ui_tokens.md`). No se cambió el valor (como pide la ficha).
- **El comentario `/* legado: se retira en T93 */`** se añadió como **línea nueva** dentro del `:root` existente → `git diff` sin líneas `-`.

**Pendientes:**

- **Verificación de tokens/capturas no reproducible por script versionado** (ver desviación): si se quiere dejar como artefacto del repo, habría que añadir un `docs/investigacion/ui/verificar_tokens.mjs` en una tarea que lo autorice (o reutilizar `medir_ui.mjs`).
- Sin tag ni release: no aplica a esta tarea. Las `--cad-*` siguen vigentes hasta **T93**.

**Archivos tocados:** `frontend/src/style.css` (solo adiciones), `docs/investigacion/ui_tokens.md` (nuevo), `docs/investigacion/ui/contraste.mjs` (nuevo) y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): ningún `.svelte`, ningún `*.go`, `frontend/wailsjs/**`, `index.html` ni `package.json` tocados; `frontend/dist` regenerado por `npm run build` (ignorado por git).

---

## T71 — Base global: tipografía, viewport, fondo y safe-area (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Deja la base lista para móvil: fuentes locales **Inter** (texto) y **Outfit** (display) por subconjunto **latin**, escala tipográfica y safe-area como tokens, `viewport-fit=cover`, **corrección del fondo dividido** (bug T69), foco visible global y ajustes táctiles. Solo se tocó `frontend/src/style.css`, `frontend/index.html` (solo el `<meta viewport>`), `package.json`/`package-lock.json` (solo las 2 dependencias) y `PROGRESO.md`; la adenda autoriza además `docs/investigacion/ui/verificar_ui.mjs` (nuevo, reutilizable).

**Qué se hizo:**

- **Fuentes (paso 1):** instaladas `@fontsource-variable/inter@5.3.0` y `@fontsource-variable/outfit@5.3.0`. **No existe CSS por subconjunto** en los paquetes (`index.css`/`wght.css` traen **todos** los subsets; sumarían ~259 KB > 200 KB), así que se declara `@font-face` a mano en `style.css` apuntando al woff2 **latin** con peso variable (`inter-latin-wght-normal.woff2` 48,3 KB + `outfit-latin-wght-normal.woff2` 32,3 KB) con su `unicode-range` de latin. Tokens `--font-sans` (Inter) y `--font-display` (Outfit) con fallbacks de sistema; `body` deja de usar Nunito. El `@font-face` de Nunito se conserva (el archivo se retira en T93).
- **Escala tipográfica (paso 2):** tokens `--fs-xs..2xl` (12/14/16/18/22/28 px), `--lh-tight`/`--lh`, `--fw-regular..bold` (400/500/600/700) en un `:root` nuevo de T71. Mínimo 12 px.
- **Viewport (paso 3):** `index.html` → `width=device-width, initial-scale=1, viewport-fit=cover` (única línea tocada; el `referrer` intacto). Variables `--sat/--sar/--sab/--sal` = `env(safe-area-inset-*, 0px)`.
- **Fondo (paso 4):** `html { background: var(--bg-0); min-height: 100%; }` y `body { min-height: 100dvh; background: transparent; }` — sustituye el `#151515` y el azul pizarra `rgba(27,38,54,1)` que asomaba por debajo del contenido.
- **Base global (paso 5):** `color-scheme: dark`; `-webkit-tap-highlight-color: transparent`; `text-size-adjust: 100%`; `overscroll-behavior-y: none`; `touch-action: manipulation` en `button, a, [role="button"], input`; `:focus-visible { outline: none; box-shadow: var(--ring); }`; y un `@media (hover: none)` **documentado sin estilos de hover pegajosos** en el reset. Los títulos `h1..h6` usan `--font-display`.
- **Fondo ambiental (paso 6):** `body::before` con `position: fixed; inset: 0; z-index: -1; pointer-events: none` y dos `radial-gradient` de baja opacidad (esmeralda `--brand-500` arriba-izquierda, índigo `--indigo` abajo-derecha) vía `color-mix`. **Sin** `filter: blur` ni `backdrop-filter`.
- **Herramienta (adenda):** `docs/investigacion/ui/verificar_ui.mjs` (Playwright desde el MCP global): lee tokens por computed style, captura vistas midiendo `overflowH`, compara dos carpetas de PNG con tolerancia por canal ≤ 2 (impresas las cajas que difieren) y lee el color de un píxel. Modos `capturar`/`comparar`/`pixel`.

**Criterios de aceptación (resultado real):**

| criterio | resultado |
|---|---|
| `font-family` de `body` | **`"Inter Variable", system-ui, …`** (computed) |
| fuente de un título | no existe `h1`; se aplicó `--font-display` a `h1..h6` y el `h2` medido («Buscador de anime») computa **`"Outfit Variable", system-ui, …`** |
| peticiones de fuentes a dominios externos | **0**: las 2 peticiones (`…/inter-latin-wght-normal.woff2`, `…/outfit-latin-wght-normal.woff2`, 304) salen de **`localhost:34115`**; los únicos hosts externos son portadas (`cdn.animeav1.com`, `cdn.jkdesa.com`, pre-existentes) |
| peso de `frontend/dist` antes → después | **739 913 → 822 179 B (+82 266 B, ≈ +80,3 KB)** ≤ 200 KB |
| `descubrir_360` fullPage, píxel y=100 vs y=1000 | **iguales: `#090d16`** en y=100/500/1000/1500; la captura T69 tenía `#1b2636` (azul pizarra) desde y≈740 → **franja eliminada** |
| `overflowH` a 360 px vs `metricas.json` T69 | **0** en todas (medir_ui: 7 vistas a 360; + `favoritos@360` headless = 0) |
| cajas de elementos medidos (alto, vs T69) | tarjeta episodio **170,39** (=) · tarjeta Home **263,8** (=) · `.pestaña` **39→38** (−1 ✓) · `.boton-siguiente` **37→36** (−1 ✓) · `.buscar` **38** (=) · **`.actualizar` 34→32 (−2)** y **`.borrar-todo` 32→30 (−2)** (texto que reflujo, reportado) |
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `*.test.ts` (esbuild + node) | **etiquetas 32 · ventana 48 · views 13** (todas OK) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| Go (`-race`, `-v`) | **435 `--- PASS`**, `gofmt -l .` sin salida, `go vet`/`go build` OK — **sin cambios** |

**Diferencias de texto por el cambio de fuente (adenda, anchos):** `.pestaña` «Descubrir» **93→100** (+7) y `.actualizar` «Actualizar» **92→98** (+6) y `.boton-siguiente` **140→147** (+7): reflujo por las métricas de Inter frente a Nunito (el alto se mantiene dentro de 1 px salvo los 2 botones de texto). Los conteos de textos < 12/14 px y de táctiles < 44 px varían por **contenido vivo** de los proveedores (p. ej. `reproductor_extras@1280`), no por el CSS.

**Verificación visual (Playwright headless, `localhost:34115`):** `verificar_ui.mjs capturar` → tokens **0 vacíos**, `overflowH = 0` en descubrir/búsqueda/detalle_animeav1/detalle_jkanime @360/768/1280; `comparar` (base T69 vs T71) informa diferencias **esperadas** (cambia la fuente) con tope `tol ≤ 2`. El desborde de 7 px que muestra el navegador **con barra de scroll clásica** (MCP) desaparece en headless (misma metodología que `medir_ui.mjs`/T69).

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `git diff --numstat -- frontend/package.json` | **+2** (solo las 2 dependencias) |
| `frontend/package.json.md5` | regenerado por wails; **=`md5(package.json)`** (ver desviación) |
| `npm run check` | **svelte-check found 0 errors and 0 warnings** |
| `npm run build` | **OK**; `frontend/dist` **822 179 B** (antes 739 913 B) |
| `*.test.ts` | **32 · 48 · 13** |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` · `go vet ./...` · `go build ./...` | sin salida · limpio · OK |
| `go test -race -count=1 -v ./...` | **11 paquetes `ok`**; `^--- PASS` = **435**, `^--- FAIL` = **0** (Go intacto) |

**Desviaciones:**

- **No hay CSS por subconjunto latin** en los paquetes `@fontsource-variable/*`: el `@font-face` del subconjunto latin se escribe a mano en `style.css` (importar `wght.css` metería todos los subsets: ~259 KB, por encima del límite de 200 KB). El import se resuelve como especificador de paquete en `url()` (Vite lo resuelve y emite solo el woff2 latin).
- **No existe `h1` en la app**: la escala se aplica a `h1..h6` y la comprobación se hace sobre el `h2` («Buscador de anime») = Outfit. (El criterio está redactado como `h1`.)
- **`frontend/package.json.md5`** cambia **automáticamente** (lo regenera wails al detectar el cambio de `package.json`); está **fuera de "Puedes tocar"** pero es un fichero generado y su valor es el `md5` correcto de `package.json`. Se deja consistente y se reporta.
- **"9 vistas a 360":** `metricas.json` de T69 solo tiene **7** estados a 360; T71 confirma **0** en esos 7, más `favoritos@360 = 0`. Los estados `lateral_contraida/expandida` solo existen a ≥900 px. **Sin desborde horizontal nuevo.**
- **`.actualizar` y `.borrar-todo` bajan 2 px de alto** (texto que reflujo por la fuente): se reportan por la adenda (>1 px); el resto ≤1 px.
- **Capturas y `metricas.json` regenerados** por `medir_ui.mjs` a un directorio **temporal**, y **restaurados** los de T69 en el repositorio (la evidencia base de T69 no se pisa).

**Pendientes:**

- **Traslado de los tokens a los componentes**: la escala `--fs-*`, `--font-display` y el resto de tokens de T70 aún **no los consumen** las vistas (llega con **T72b** y las tareas de vista).
- **Objetivos táctiles < 44 px** (`.pestaña`, `.actualizar`, `.borrar-todo`, `.estrella`, `.hamburguesa`…) siguen pendientes; los resuelve la serie v1.4.0.
- Sin tag ni release: no aplica a esta tarea. `--cad-*` y el `@font-face` de Nunito siguen vigentes hasta **T93**.

**Archivos tocados:** `frontend/src/style.css`, `frontend/index.html` (solo el `<meta viewport>`), `frontend/package.json` y `frontend/package-lock.json` (solo las 2 dependencias) + `frontend/package.json.md5` (generado), `docs/investigacion/ui/verificar_ui.mjs` (nuevo) y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): ningún `.svelte`, ningún `*.go`, `frontend/wailsjs/**` ni `main.go` tocados; `frontend/src/main.ts` **no** necesitó cambios (el `@font-face` vive en `style.css`).

---

## T72 — Primitivas de UI e iconos (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Añade **6 componentes nuevos** con el sistema de tokens de T70 y la base de T71 (fuentes `--font-sans`/`--font-display`, escala `--fs-*`, safe-area) y un set de iconos SVG inline. **No migra ningún componente existente** (eso es T72b y las tareas de vista).

**Componentes (archivos NUEVOS en `frontend/src/`):**

- **`Icono.svelte`**: 24 iconos SVG inline (paths al estilo Heroicons, MIT) con `stroke="currentColor"`/`fill="none"` salvo los «llenos» (`play`, `pausa`, `estrella-llena`, `marcador-lleno`). Props `nombre`, `tam` (16|20|24, def. 20) y `etiqueta` (si se pasa, `role="img"` + `aria-label`; si no, `aria-hidden="true"`). Los px de `tam` son los únicos permitidos del set.
- **`Btn.svelte`**: variantes `primario` (`--grad-brand`, `--on-brand`, `--sh-glow`), `secundario` (`--surface-chip`+`--border`), `fantasma`, `icono`; tamaños `m`/`l`; `bloque`, `href` (renderiza `<a>`), `disabled`, `etiqueta`, slots `icono-izq`/`icono-der`. Alto y ancho mínimo `var(--tap)` (44 px) en todas. Hover SOLO en `@media (hover:hover) and (pointer:fine)`, `:active` escala .98, `:focus-visible` con `--ring`, `:disabled` atenuado.
- **`Chip.svelte`**: `<button>` con `aria-pressed`; activo (`--grad-chip-active`, `--on-brand`, punto estático sin animación) / inactivo (`--surface-chip`+`--border`); el texto envuelve (`white-space:normal`, `overflow-wrap:anywhere`); alto `calc(var(--tap) - var(--sp-1))` = 40 px (sin px literal).
- **`Insignia.svelte`**: tonos `marca|info|alerta|neutro|indigo` (`color-mix` sobre tokens), `font-size: var(--fs-xs)`, prop `mayusculas`.
- **`Esqueleto.svelte`**: bloque con props `alto`/`ancho`/`radio` (valores CSS, por defecto tokens); shimmer con `--dur-slow`, desactivado por `prefers-reduced-motion: reduce`.
- **`EstadoVista.svelte`**: variantes `vacio|error|cargando` con `Icono`, `h2`, texto y slot de acción; `role="alert"` (error) / `role="status"` (cargando) / sin rol (vacío). Usa `h2` (la app aún no tiene `h1`): **anotado para T91**.

**Solo tokens / cero literales** (`grep -nE "#[0-9a-fA-F]{3,8}|rgba?\(|[0-9]+px"` sobre los 6 componentes): los únicos px son el borde `1px solid` (Btn/Chip/Insignia) y ningún tamaño literal (los de `Icono` salen de la prop `tam`); `font-family`/`box-shadow` siempre con `var()`. Un falso positivo (el `{#each` de `Icono`). Sin colores, radios, sombras ni duraciones literales.

**Verificación — arnés temporal + capturas + medición táctil:**

- **Arnés `frontend/src/_galeria.svelte`** (montado temporalmente en `App.svelte` solo durante la sesión) → **borrado** y `App.svelte` **revertido** (`git diff` vacío; no aparece en `git status`).
- **Capturas** a **360 y 1280 px** en `docs/img/v1.4/primitivas/`: `iconos`, `botones`, `chips`, `insignias`, `esqueletos`, `estados` × 2 anchos = **12 PNG**.
- **Medición táctil** con el nuevo modo `tactiles` de `docs/investigacion/ui/verificar_ui.mjs` (autorizado por la adenda): **todos los `Btn` ≥ 44×44** (44–46 px de alto; el de icono 44×44) y **todos los `Chip` ≥ 40 px** (40 / 40 / 43,59) a 360, 768 y 1280 → «Todos los objetivos táctiles cumplen». El chip «Activo» (con punto) mide 84,44×40.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `*.test.ts` (esbuild + node) | **etiquetas 32 · ventana 48 · views 13** (todas OK) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` · `go vet ./...` · `go build ./...` | sin salida · limpio · OK |
| `go test -race -count=1 -v ./...` | **11 paquetes `ok`**; `^--- PASS` = **435**, `^--- FAIL` = **0** (Go intacto) |
| `git status --porcelain` | 6 componentes nuevos + `docs/img/v1.4/primitivas/` + `verificar_ui.mjs` + `PROGRESO.md` (ni `App.svelte` ni el arnés) |

**Desviaciones:**

- **El arnés de galería fue temporal** (como pide la ficha): se montó en `App.svelte` solo para capturar/medir y se revirtió; la captura/medición se hizo con Playwright (MCP + `verificar_ui.mjs tactiles`).
- **Modo `tactiles` añadido a `verificar_ui.mjs`** (autorizado por la adenda): mide `.btn` (≥44×44) y `.chip` (≥40) y termina con código ≠ 0 si algo no cumple. **No se creó otro script**; la captura de las 12 imágenes usó el código del MCP (sin script versionado nuevo).
- **`--text-4` no se usa** (es decorativo y no cumple AA): las primitivas usan `--text-2`/`--text-3`.
- **`h2` en `EstadoVista`** pese a que la app no tiene `h1`: lo indica la adenda; la jerarquía se resuelve en **T91**.
- **Sin colisiones de nombre** con `Boton*.svelte`, `Tarjeta*.svelte` ni `Toast.svelte`: se usaron los nombres exactos de la ficha.

**Pendientes:**

- **Migración de los componentes existentes** a estas primitivas: es **T72b** (y las tareas de vista); aquí **no** se migró ninguno.
- Sin tag ni release: no aplica a esta tarea. `--cad-*` siguen vigentes hasta **T93**.

**Archivos tocados:** `frontend/src/{Icono,Btn,Chip,Insignia,Esqueleto,EstadoVista}.svelte` (nuevos), `docs/img/v1.4/primitivas/*.png` (12 capturas nuevas, versionadas), `docs/investigacion/ui/verificar_ui.mjs` (solo el modo `tactiles` que autoriza la adenda) y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): ningún componente existente, ningún `*.go`, `frontend/wailsjs/**`, `main.go`, `package.json` ni `style.css` tocados; el arnés temporal se borró y `App.svelte` quedó idéntico.

---

## T72b — Puente de color: literales a tokens (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Pasada **MECÁNICA**: sustituye en los componentes **existentes** los literales de color (`#hex`/`rgba`), las variables `--cad-*` y los radios/sombras literales por los tokens de T70, para que toda la app use una sola paleta (verde de Stitch) **antes** de rediseñar las vistas. **No cambia layout, estructura, markup ni lógica**: solo color, sombra y radio, y **solo dentro de `<style>`**.

**Mapeo (fuente de verdad: `docs/investigacion/ui_mapeo_color.md`).** Reglas por uso real (no solo por luminosidad):

| Literal | Token | Nota |
|---|---|---|
| `#151515` | `--bg-0` | fondo base |
| `#1e1e24` | `--surface-card` | superficies/tarjetas |
| `#2a2a35` | `--surface-chip` (chips·botones) · `--bg-2` (bloques) · `--surface-input` (campo texto) | según uso |
| `#3f3f46` | `--border` (borde 1 px) · `--border-strong` (relleno) | según uso |
| `#ffffff`/`white` | `--text-1` | texto/iconos principales |
| `#a1a1aa` | `--text-3` | secundario |
| `#4ade80` | `--brand-400` (texto/iconos/bordes) · `--brand-500` (rellenos) | marca |
| `#111` sobre verde | `--on-brand` | texto sobre relleno verde |
| `#f87171` | `--danger` | semántico |
| `#fbbf24` | `--fav` | favorito |
| `#d4d4d8`/`#f4f4f5` | `--text-2`/`--text-1` | aproximados por luminosidad |
| `#555`/`#999` (`:disabled`) | `--bg-2`/`--text-3` | aproximados |
| `var(--cad-radio)`/`8px`/`10px`/`12px` | `--r-md` | radios medios |
| `4px`/`6px` | `--r-sm` | radios pequeños |
| `50%`/`999px` | `--r-full` | círculos |
| sombra de hover de `TarjetaAnime` `0 6px 16px rgba(0,0,0,0.45)` | `--sh-card-hover` | solo hover |
| overlay de la insignia de episodio `rgba(0,0,0,0.75)` | `--surface-panel` | contraste de etiqueta |
| fondo del degradado de `VistaInicio` `rgba(255,255,255,0.08)` | `--border` | |

**Excepciones dejadas literales (justificadas):**

- **`TarjetaEpisodio.svelte`** (6 líneas hex): `:107` `#52525b` (relleno de flecha inactiva), `:110-113` pendiente del degradado `#2a2a35`/`#26262e` (dos paradas de un mismo `linear-gradient`, sin token equivalente), `:138` `#14532d` (verde oscuro del contador «Sin ver»). No hay token exacto; se listan en `ui_mapeo_color.md`.
- **Scrims negros de degradado/overlay** (se quedan por diseño): `BarraLateral.svelte:563` (`rgba(0,0,0,0.75)`, con comentario que apunta a `TarjetaEpisodio`), `VistaReproductor.svelte:812` (`rgba(0,0,0,0.65)`).
- **Keyframes de `BotonSiguiente.svelte`** `:96`/`:99` (`rgba(74,222,128,·)` del pulso) — animación, no paleta.
- **Máscaras de `FilaAnimes.svelte`** `black` (`:185,186,189,190,194,195,197,198`) — `black` aquí es alfa de máscara, no color de la paleta.
- `--text-4` no aparece; no se usó.

**Archivos migrados (15 `.svelte` existentes; 237 inserciones / 237 borrados, 4–23 hunks/archivo):** `App`, `BarraLateral`, `BotonFavorito`, `BotonSiguiente`, `FilaAnimes`, `GridFavoritos`, `SelectorFuente`, `TarjetaAnime`, `TarjetaEpisodio`, `TarjetaHistorial`, `Toast`, `VistaBusqueda`, `VistaDetalle`, `VistaInicio`, `VistaReproductor`. **`style.css` NO se tocó**: las `--cad-*` quedan definidas y sin uso hasta **T93**. Ningún `*.ts`, `*.go`, `wailsjs/**`, `index.html` ni los 6 componentes de T72 tocados.

**Verificación (compuerta, resultado real):**

| Comprobación | Resultado |
|---|---|
| `grep -rnE "#[0-9a-fA-F]{3,8}\b" frontend/src/*.svelte` | **6 líneas**, todas excepción en `TarjetaEpisodio.svelte` |
| `grep -rn -- "--cad-" frontend/src/*.svelte` | **0 usos** (142 → 0) |
| `git diff` solo dentro de `<style>` | **15/15 OK** (primer cambio nuevo ≥ línea de `<style>` en cada archivo) |
| Geometría (`verificar_ui.mjs` a 360/768/1280) | **sin cambios**: tarj. episodio **170,39**, tarj. Home **263,8**, pestaña **38**, botones idénticos; **`overflowH = 0`** en todas las vistas y anchos |
| Capturas antes/después (12 PNG × 2) | **12 con diferencias de color** (esperado: paleta global), máx Δ 199–246 (tol ≤ 2); **0 idénticas** |
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `*.test.ts` (esbuild + node) | **etiquetas 32 · ventana 48 · views 13** (todas OK) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `go test -count=1 -v ./...` | **11 paquetes `ok`**; `^--- PASS` = **435**, `^--- FAIL` = **0** (Go intacto) |
| `git status` | 15 `.svelte` modificados + `verif_antes/` + `verif_despues/` + `ui_mapeo_color.md` + `PROGRESO.md` (+ `tareas/T72b.md`, ya presente) |

**Desviaciones:**

- **`docs/investigacion/ui_mapeo_color.md` generado antes de editar** (paso 1) con un script temporal **fuera del repo** (`%TEMP%/opencode/`), como pide la ficha.
- **Mapeos aproximados por luminosidad** (documentados en la tabla): `#d4d4d8`/`#f4f4f5` → `--text-2`/`--text-1`, `#555`/`#999` en `:disabled` → `--bg-2`/`--text-3`, `#000` → `--bg-0`. Ante duda se eligió el más cercano y se anotó.
- **Órdenes del resumen de geometría con nº de tarjetas distinto** (3 de 12 métricas): el conteo de tarjetas de episodio cargadas por lotes cambió entre capturas (48/32 en «antes» vs 366 en «después») por temporización de red —**no** por T72b—; el tamaño de cada tarjeta (170/216/223 × 170) no varía.
- **La sombra de hover de `TarjetaAnime`** pasa a `--sh-card-hover` (incluye un halo verde del rediseño): cambia el aspecto **solo en hover**, que las capturas (sin hover) no registran; el tamaño no cambia.
- **`tareas/T72b.md`** ya existía en el árbol de trabajo (sin trackear); no se creó aquí.
- **Carpetas de captura nuevas** `docs/investigacion/ui/verif_antes/` y `verif_despues/`: son la salida exigida por el paso 4 (capturas + `resumen.json`), no una edición de archivos existentes.

**Pendientes:**

- **Migración a las primitivas de T72** (`Btn`, `Chip`, `Insignia`, `Esqueleto`, `EstadoVista`, `Icono`): es de las **tareas de vista**, no de T72b; aquí solo se mapeó color/radio/sombra.
- **`--cad-*`**: siguen **definidas** en `style.css` (sin uso) hasta **T93**.
- Sin tag ni release: no aplica a esta tarea.

**Archivos tocados:** 15 `.svelte` existentes (solo `<style>`), `docs/investigacion/ui_mapeo_color.md` (nuevo), `docs/investigacion/ui/verif_antes/` y `docs/investigacion/ui/verif_despues/` (12 PNG + `resumen.json` cada uno, capturas de la comprobación) y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `style.css` intacto, ningún `*.ts`/`*.go`/`wailsjs/**`/`index.html` ni componente de T72 tocados.

---

## T73 — Barra inferior móvil y cajón de historial (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. En modo estrecho (`$isNarrow`, `< 900 px`) sustituye la **hamburguesa** por una **barra de navegación inferior fija** de 4 destinos (Descubrir, Buscar, Favoritos, Historial) que reutiliza el **cajón de historial** existente. En modo ancho el aspecto y el comportamiento **no cambian** (comprobado por píxeles).

**Qué se hizo:**

- **`style.css`**: token nuevo `--bar-h: 60px` (solo añadido).
- **`ui.ts`**: dos stores nuevos no persistidos: `pestanaHome` (`'descubrir'|'favoritos'`) y `limpiarBusqueda` (contador).
- **`VistaInicio.svelte`**: `pestana` pasa a `$: pestana = $pestanaHome` y `seleccionar` hace `pestanaHome.set(...)`; el resto intacto (incluido el `position:absolute` + `visibility:hidden` del panel inactivo).
- **`VistaBusqueda.svelte`**: `id="buscador-anime"` en el `<input>` y una suscripción a `limpiarBusqueda` que vacía `texto` y cancela el temporizador (7 líneas de código + `import` + `id` + `cancelarLimpiar()` en `onDestroy`); se cancela en `onDestroy`.
- **`BarraInferior.svelte`** (NUEVO): `<nav aria-label="Navegación principal">` con 4 `<button>` (Icono `brujula`/`buscar`/`estrella`/`historial`, `tam=24`), ancho a partes iguales y ≥ `--tap` de alto; props `activo`/`onDescubrir`/`onBuscar`/`onFavoritos`/`onHistorial`; el activo lleva `aria-current="page"`, `--brand-400` y un **indicador superior de 2 px** (no depende solo del color); expone `enfocarHistorial()`; `--surface-panel` + `backdrop-filter: blur(var(--blur-md))`, borde superior `--border`, `padding-bottom: var(--sab)`, `z-index: var(--z-bar)`, `position: fixed; inset: auto 0 0 0`, `--sh-card` de sombra, `:active`/`:focus-visible` (`--ring`) y sin transiciones con `prefers-reduced-motion: reduce`.
- **`App.svelte`**: renderiza la barra con `{#if $isNarrow && $vista !== 'reproductor'}`; **elimina** el bloque `.cabecera-estrecha`/hamburguesa (era narrow-only) y su CSS; `.area-principal.con-barra` añade `padding-bottom: calc(var(--bar-h) + var(--sab) + var(--sp-4))`; envuelve shell + `<Toast/>` en `<div class="raiz" style:--bar-offset={...}>` (`display: contents`); implementa los destinos (`irAHome`/`irABuscar`/`irAHistorial`), `barraActiva` y `enfocarHistorialBarra`.
- **`BarraLateral.svelte`**: prop `onCerrarCajon` (se llama al cerrar por velo, botón o Escape, y al alternar en estrecho) y estilos del cajón (`.alternar`/`.papelera`/`.borrar-todo`/`.reintentar`/`.si-borrar`/`.cancelar` ≥ `--tap`; `.pie` con `padding-bottom: calc(var(--sp-3) + var(--sab))`; `height: 100dvh`).
- **`TarjetaHistorial.svelte`**: solo `<style>` — `:global(.cajon) .tarjeta { min-height: var(--tap) }` y `:global(.cajon) .borrar { min-width/min-height: var(--tap) }`.
- **`Toast.svelte`**: `bottom: calc(24px + var(--bar-offset, 0px))`.

**Regla del destino activo** (documentada en `App.svelte`): en `busqueda` con `$searchState.query !== ''` → **`'buscar'`**; en `busqueda` sin consulta → el valor de **`$pestanaHome`** (`descubrir`|`favoritos`); en `detalle` —y en el reproductor, donde la barra no se ve— → **ninguno**.

**Verificación (modo `shell` de `verificar_ui.mjs`, resultados reales; 20/20 comprobaciones en verde):**

| Comprobación | Resultado |
|---|---|
| Barra a 360/390/768 | visible, **pegada al borde** (`bottom == innerHeight`), alto **60 px**, 4 destinos **≥44×44** (59 px de alto; 90 / 97,5 / 192 px de ancho), `aria-current="page"` en Descubrir |
| A 1024/1280 | **sin barra**, sin hamburguesa, `overflowH = 0`; geometría idéntica a T72b: tarjeta de Home **263,8**, pestaña **38**, tarjeta de episodio **170,39** |
| Hamburguesa | **ausente** en los 5 anchos |
| Destinos | Descubrir/Favoritos cambian de pestaña desde Home, desde detalle y desde una búsqueda; Buscar **enfoca** `#buscador-anime`; Historial **abre el cajón**; Escape lo cierra y **devuelve el foco** al botón Historial |
| Buscar "bleach" → Descubrir / Favoritos | aparece el Home/panel de Favoritos, **input vacío** |
| Reproductor | **sin barra**; al volver, **reaparece** |
| Fin de scroll | Home y detalle terminan **por encima** de la barra (618 y 636 < 680) |
| T66/T67 scroll del detalle | detalle → abajo → reproductor → "← Atrás": **69674 → 69673** px (≤ 5) |
| Toast sobre la barra | `bottom = 656` ≤ `topBarra = 680` (borrar + deshacer un historial) |
| `prefers-reduced-motion: reduce` | `transition-duration: 0s` en los destinos |
| Táctiles < 44 px | 6 visibles (`← Atrás` 83×31, `Quitar de favoritos` 32×32, `Continuar` 103×35), **ninguno de la barra ni del cajón** |
| Capturas a 1280 (antes T72b / después T73, tol ≤ 2) | **idénticas** (`descubrir_1280.png`, 1280×1629; **0 px distintos**) |

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `*.test.ts` (esbuild + node) | **etiquetas 32 · ventana 48 · views 13** (todas OK) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `go test -count=1 -v ./...` | **11 paquetes `ok`**; `^--- PASS` = **435**, `^--- FAIL` = **0** (Go intacto) |
| `git status --porcelain` | 8 `.svelte`/`.ts` + `style.css` modificados + `BarraInferior.svelte` (nuevo) + `docs/img/v1.4/shell/` + `verificar_ui.mjs` + `PROGRESO.md` (+ `tareas/T73.md`, ya presente) |

**Desviaciones:**

- **`--sh-card` como sombra de la barra** (lo pide reportar la ficha, punto 5): no hay token de sombra específico para la barra; se reutiliza `--sh-card`.
- **La pestaña del Home ya NO se reinicia** al volver de una búsqueda (persiste en la sesión): lo exige el punto 3 y queda anotado aquí.
- **Modo `shell` añadido a `verificar_ui.mjs`** (autorizado por la ficha): mide/verifica barra, destinos, geometría y las comprobaciones funcionales, y deja `shell.json` + las capturas. **No se creó otro script.**
- **`--bar-offset` con un `<div class="raiz">` de `display: contents`** (tal cual el punto 6): no altera el layout; a 1280 la captura sale **idéntica**.
- **El cajón no cambia su z-index** (ya cubre la barra), como pide el punto 8.
- **`BarraLateral.svelte`: se tocó su `<script>`** (no solo el `<style>`) para añadir la prop `onCerrarCajon` y llamarla en las 3 vías de cierre (velo, botón y Escape) — es **lo indicado en el punto 8** (devolver el foco al botón Historial).
- **Prueba del toast**: el ✕ de la tarjeta está `visibility:hidden` sin hover, así que el arnés hace `hover` sobre la tarjeta antes de pulsarlo (detalle de la verificación, no del código).

**Pendientes:**

- **Contador de favoritos en la barra**: lo decide **T75**; aquí no se puso.
- Sin tag ni release: no aplica a esta tarea. `--cad-*` siguen definidas y sin uso hasta **T93**.

**Archivos tocados:** `frontend/src/BarraInferior.svelte` (nuevo), `frontend/src/{App,BarraLateral,TarjetaHistorial,VistaInicio,VistaBusqueda,Toast}.svelte`, `frontend/src/ui.ts`, `frontend/src/style.css`, `docs/investigacion/ui/verificar_ui.mjs` (solo el modo `shell`), `docs/img/v1.4/shell/**` (7 capturas + `shell.json`) y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `stores.ts`, `views.ts`, `api.ts`, `frontend/wailsjs/**`, `*.go`, `index.html` y el `<script>` de `VistaDetalle`/`VistaReproductor` intactos.

---

## T73b — Cajón opaco (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Corrige la transparencia del cajón de historial: el fondo `.barra` (y `.barra.cajon`) era `var(--surface-card)` = `rgba(16,23,41,.55)`, que dejaba traslucir el contenido y el pie "Borrar todo" sobre la barra inferior. Solo `<style>` de `BarraLateral.svelte`; **no se añadió `--surface-solid`** porque `--bg-1` (#0f172a) ya es un fondo opaco equivalente (T70).

**Qué se hizo (solo `<style>`):**

- **`.barra`** (`:321`): `background: var(--surface-card)` → **`var(--bg-1)`** (opaco). Aplica también en escritorio.
- **`.barra.cajon`** (`:555`): añadido **`box-shadow: var(--sh-panel)`** para separar el cajón del velo (solo con `.cajon`; en escritorio `box-shadow: none`).
- **Sin cambios** en `<script>`, markup, `*.go`, `wailsjs/**` ni otros componentes.

**Verificación (Playwright, `localhost:34115`, resultado real):**

| Comprobación | Resultado |
|---|---|
| Cajón @360: fondo | `getComputedStyle(.barra.cajon).backgroundColor` = **`rgb(15, 23, 42)`** (opaco, `--bg-1`); no trasluce nada |
| Cajón @360: sombra | **`rgba(0,0,0,.5) 0 20px 50px 0, rgba(255,255,255,.1) 0 1px 0 0 inset`** (`--sh-panel`) |
| Cajón @360: medidas | cajón **306×740** (min(320,85vw)); pie `.pie` y=671 h=69, dentro del cajón; el cajón cubre la barra inferior (y=665 h=60) |
| **Tarjetas del historial @360** | **NO se salen del panel**: `.envoltorio` x=12, ancho 282, `right=294` < 306; el botón `.borrar` (44×44, absolute) x=242–286; **0 descendientes con `right > cajón`** (barrido de los 3 grupos). No hay selector/ancho que reportar |
| Escritorio @1280 | `.barra` 300×800 en x=0, `border-right: 1px solid var(--border)`, **`box-shadow: none`**; geometría idéntica; **único cambio** = tono de fondo (opaco). `overflowH` sin cambios |

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `*.test.ts` (esbuild + node) | **etiquetas 32 · ventana 48 · views 13** (todas OK) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` · `go vet ./...` · `go build ./...` | sin salida · limpio · OK |
| `go test -count=1 -v ./...` | **11 paquetes `ok`**; `^--- PASS` = **435**, `^--- FAIL` = **0** (Go intacto) |

**Desviaciones:**

- **No se usó `--surface-solid`**: `--bg-1` (`#0f172a`) ya es el fondo opaco del rediseño; la ficha lo permite expresamente ("o `--bg-1`").
- **Interprete de "el fondo de la barra debe verse igual que ahora":** se cumple la aceptación literal (**geometría idéntica**; **solo cambia el tono de fondo**, que pasa de semitransparente a opaco), que es el objeto de la tarea.
- **No se rediseñó `TarjetaHistorial`**: se comprobó que no se sale del cajón; no hubo nada que reportar (punto 3).

**Pendientes:**

- Sin tag ni release: no aplica a esta tarea. `--cad-*` siguen definidas y sin uso hasta **T93**.

**Archivos tocados:** `frontend/src/BarraLateral.svelte` (solo `<style>`), `docs/img/v1.4/shell/t73b-cajon-360.png` y `t73b-escritorio-1280.png` (capturas nuevas de la verificación) y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `style.css` intacto (no hizo falta el token), ningún `<script>`, markup, `*.go`, `wailsjs/**` ni otro componente tocados.

## T74 — Cabecera compacta de detalle y reproductor (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Unifica la cabecera de **detalle** y **reproductor** en un componente nuevo `CabeceraVista.svelte` (botón "Atrás" con `Icono` `flecha-izq`, título `h2` de una línea con elipsis, subtítulo opcional y slot `acciones` a la derecha para la estrella de `BotonFavorito`). En **modo estrecho** (`< 900 px`, mismo corte que `isNarrow`) gana **táctiles de 44 px**, gutter de 16 px y, en el detalle, **cabecera fija** (`sticky` con `--surface-panel` + blur y borde inferior). En **≥ 900 px** la fila no cambia salvo el glifo "←", que pasa a ser el icono. **Sin cambios de comportamiento**: "Atrás" sigue llamando a `atras()` y el scroll (T66/T67) funciona igual.

**Resultados de la verificación** (modo `cabecera` nuevo en `verificar_ui.mjs`, localhost:34115, red real):

| comprobación | resultado |
|---|---|
| 360/390/768: "Atrás" ≥ 44×44 | **sí** (82,63×44) |
| 360/390/768: estrella ≥ 44×44 | **sí** (44×44) |
| 360/390/768: `overflowH` | **0** en todas las vistas |
| 360: título largo ("Tensei Kenja no Isekai Life…", 90 car.) | **una línea** (`nowrap` + `ellipsis`), `recortado=true` |
| 360: cabecera del detalle fija tras bajar 2000 px | **sí** (`y=0` ≈ `--sat`, `position: sticky`) y sin desbordes |
| 360: detalle JKAnime | **cabecera OK**, "Atrás" 82,63×44 |
| 360/768: reproductor | cabecera **NO fija** (`static`), subtítulo del episodio presente, estrella **sí** (detalle coincide) |
| estrella desde la cabecera | **alterna** el estado (pulsar marca/desmarca) |
| `prefers-reduced-motion: reduce` | **sin** transición en la cabecera |
| táctiles < 44 px que siguen en modo estrecho | **solo "Continuar"** (103×35) — esperado, se trata en T80/T83 |
| **Capturas 1280 antes/después (tol ≤ 2)** | **342 px distintos, caja `x=347..399, y=41..54`** — **dentro del botón "Atrás"** (el glifo "←" → icono); `máx Δ179` |
| altura de "Atrás" en 1280 | **31 px** (igual que antes) |
| geometría ≥ 900 px (referencia) | tarjeta de Home **263,8**, tarjeta de episodio **170,39**, pestaña **38** (sin cambios) |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **48 / 13 / 32** comprobaciones OK (`ventana`, `views`, `etiquetas`) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| Go (`go test ./...`) | **716 `--- PASS`**, 0 FAIL (12 paquetes `ok`, `*.go` intactos) |

**Desviaciones (documentadas):**
- **`.atras` con `padding: 8px 14.27px`**: el icono (20 px) es más ancho que el glifo "←" (≈16,5 px); para que el botón **conserve su ancho exterior** y la fila de ≥ 900 px no se desplace, se recorta el padding lateral. Sin ese ajuste la caja de diferencia a 1280 llegaba hasta el título y la estrella (desplazados ~7,5 px).
- **Import en el `<script>` del reproductor**: se añadió `CabeceraVista` (imprescindible para usar el componente) **en lugar de** `Icono` (los iconos viven dentro de `CabeceraVista`); además del `BotonFavorito` y la única sentencia reactiva `$: detalleDelEpisodio = ...`. `VistaDetalle` solo añadió el import del componente.
- **Slot mediante `<svelte:fragment slot="acciones">`** (no `slot="acciones"` directo sobre `BotonFavorito`): es lo que `svelte-check` 4 / Svelte 5 5.55 acepta sin marcar el contenido como `children`; mismo resultado.
- **Cabecera del reproductor con doble gutter en estrecho** (x=32 a 360): la spec pide "fuera de `fija`, mismo padding sin fondo", y `.contenedor` ya aporta `var(--gutter)`; es el comportamiento literal de la especificación.

**Capturas** en `docs/img/v1.4/cabecera/`: `detalle_{360,768,1280}.png`, `detalle_largo_360.png`, `detalle_scroll_360.png`, `detalle_jkanime_360.png`, `reproductor_{360,768}.png` (+ `cabecera.json`).

**Archivos tocados:** `frontend/src/CabeceraVista.svelte` (nuevo), `frontend/src/{VistaDetalle,VistaReproductor,BotonFavorito}.svelte` (markup/`<style>`/padding de `.contenedor`; en el reproductor, además, el import y una sentencia reactiva), `docs/investigacion/ui/verificar_ui.mjs` (modo `cabecera`), `docs/img/v1.4/cabecera/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `BarraInferior`, `BotonSiguiente.svelte`, el selector de fuentes, las tarjetas, `revelar.ts`, `ventana.ts`, la lógica de scroll (T66/T67) e `index.html` intactos. **Sin tag ni release** (los hace el humano).

## T75 — Buscador y pestañas del Home (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Lleva al **modo estrecho** (`< 900 px`, mismo corte que `isNarrow`) dos piezas que aún tenían forma de escritorio: las **pestañas del Home** y el **buscador**. En modo estrecho las pestañas `.pestanas` (y el `role="tablist"` con sus `tabpanel`) **desaparecen** —la pestaña activa la manda la barra inferior de T73— y el buscador gana **lupa** dentro del campo, botón **✕ "Borrar búsqueda"** (solo con texto), `h2` alineado a la izquierda y objetivos táctiles de 44 px. En **≥ 900 px** todo queda **igual que en T74** (la barra de pestañas y el buscador de escritorio no cambian ni un píxel). **Sin cambios de comportamiento**: el debounce de 400 ms, el Enter, el vacío→Home, el foco desde el destino "Buscar" de la barra y el contador `limpiarBusqueda` de T73 actúan igual.

**Resultados de la verificación** (modo `buscador` nuevo en `verificar_ui.mjs`, localhost:34115, red real):

| ancho | tablist | role paneles | lupa | ✕ | input (w×h) | "Buscar" (w×h) | pestaña (h) | tarj. Home (h) | `overflowH` |
|---|:--:|:--:|:--:|:--:|---|---|---|---|---|
| 360 | no | — | sí | no | 230,8×64 | 85,2×66 | — | 263,8 | 0 |
| 390 | no | — | sí | no | 260,8×64 | 85,2×66 | — | 263,8 | 0 |
| 768 | no | — | sí | no | 622,8×64 | 85,2×66 | — | 263,8 | 0 |
| 1024 | **sí** | `tabpanel` | no | no | 564,8×38 | 85,2×38 | 100,14×38 | 263,8 | 0 |
| 1280 | **sí** | `tabpanel` | no | no | 820,8×38 | 85,2×38 | 100,14×38 | 263,8 | 0 |

| comprobación (55 en total) | resultado |
|---|---|
| 360/390/768: sin `[role=tablist]`, paneles sin `role` ni `aria-labelledby`, lupa presente, sin ✕ sin texto, `overflowH = 0` | **sí** (18/18) |
| 1024/1280: `[role=tablist]` presente, paneles `role="tabpanel"`, sin lupa ni ✕, `overflowH = 0` | **sí** (10/10) |
| 360: Descubrir activo al arrancar; barra → Favoritos → Descubrir | **sí** |
| 360: encabezado `Favoritos (N)` visible solo en su panel (detectado `Favoritos (1)`) | **sí** |
| 360: ✕ aparece con texto y desaparece sin él; **✕ ≥ 44×44** | **sí** (44×44) |
| 360: ✕ vacía el input y **devuelve el foco** a `#buscador-anime`; Home visible tras limpiar | **sí** (`{valor:"", foco:"buscador-anime"}`) |
| 360: Enter → resultados de "bleach"; botón "Buscar" muestra **"Buscando…"/disabled** y luego vuelve a "Buscar" | **sí** |
| 360: "Buscar" ≥ 44 de alto (85,2×66) · "Actualizar" ≥ 44 (98,44×44) | **sí** |
| 800 → 1000 → 800: pestañas reaparecen/desaparecen, **conserva Descubrir activo** y el **scroll de las filas** (`s0 = s1 = 152`) | **sí** |
| 360: destino "Buscar" de la barra **enfoca** el input | **sí** |
| 360: T66/T67 — scroll del detalle **restaurado** al reabrir (69 492 → 69 493, Δ1 px) | **sí** |
| `prefers-reduced-motion: reduce`: `transition-duration` del ✕ | **`0s`** |
| táctiles < 44 px que siguen en modo estrecho (Home) | **solo las flechas de carrusel** (32×32 ×8) — esperado, se tratan en T76; las estrellas de `TarjetaAnime` no se muestran sin hover |
| **Capturas 1280 antes (HEAD) / después (T75), tol ≤ 2** | **25 px distintos**, caja `x=333..471, y=639..648` (138×9), `máx Δ7` — **idéntico al ruido run-to-run** del mismo código (mismos 25 px y misma caja) ⇒ **sin cambio visual** |
| geometría ≥ 900 px | tarjeta de Home **263,8**, pestaña **38**, "Buscar" **85,2×38**, input de escritorio **38** de alto (sin cambios) |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| Go (`go test -race ./...`) | **435 `--- PASS` de nivel superior**, 0 FAIL (12 paquetes `ok`, `*.go` intactos) |

**Desviaciones (documentadas):**
- **`.campo { display: contents }` en la base (escritorio):** el campo de búsqueda se envuelve en un `<div class="campo">` que aloja la lupa y el ✕; con `display: contents` el wrapper **no genera caja** en escritorio y el input + botón "Buscar" siguen siendo los mismos hijos flex de `.buscador` (geometría byte-idéntica a T74). Sin ese `display: contents` el input pasaría a ser nieto flex y cambiaría de ancho.
- **`min-height: var(--tap)` en `.campo input` (solo estrecho):** la ficha no lo listaba para el input, pero sin él la caja táctil del campo quedaba por debajo de 44 px; con `min-height: var(--tap)` el campo mide 64 de alto en 360/390/768.
- **Regla `@media (prefers-reduced-motion: reduce)` para `.limpiar` colocada al final del `<style>`:** el ✕ define `transition` dentro del bloque de modo estrecho; la anulación va al final para ganar por orden (verificado: `0s`).
- **Lupa decorativa:** `Icono` `buscar` (`tam=20`, `--text-3`) va en un `<span class="lupa" aria-hidden>` posicionado dentro del campo (sin etiqueta accesible: el campo ya lleva `aria-label="Buscar anime"`).
- **`limpiar()` en `VistaBusqueda.svelte` (≤ 15 líneas, como pedía la ficha):** vacía `texto`, cancela el temporizador, reprograma la búsqueda y devuelve el foco a `#buscador-anime`. No toca `stores.ts`/`views.ts`/`ui.ts`.
- **Import añadido en el `<script>`:** solo `Icono` e `isNarrow` (además de la función `limpiar()`), en `VistaBusqueda`; solo `isNarrow` en `VistaInicio`. Nada más.
- **Comparación a 1280 contra HEAD (no contra `docs/img/v1.4/shell/`)**: la base `shell/home_1280.png` (T73, **pre-T73b**) ya difiere en el fondo del lateral (T73b lo hizo opaco) y el Home/historial son **contenido dinámico** (con `shell/home_1280.png` el diff era de 228 796 px por el lateral + filas). Por eso la comprobación honesta es **HEAD (stash) vs T75 capturadas seguidas**: 25 px en una banda de 138×9, **exactamente igual** al ruido entre dos capturas del mismo código ⇒ **0 cambio real a ≥ 900 px**.

**Capturas** en `docs/img/v1.4/buscador/`: `home_360.png` (página completa), `favoritos_360.png`, `busqueda_texto_360.png` (con ✕ visible), `busqueda_resultados_360.png`, `home_1280.png` (viewport, para la comparación a 1280) (+ `buscador.json`).

**Archivos tocados:** `frontend/src/VistaBusqueda.svelte` (markup del buscador, su `<style>` y el padding de `.contenedor`; en el `<script>`, solo `Icono`+`isNarrow` y `limpiar()`), `frontend/src/VistaInicio.svelte` (markup de pestañas/paneles y `h2` de "Favoritos", su `<style>`; en el `<script>`, solo `isNarrow`), `docs/investigacion/ui/verificar_ui.mjs` (modo `buscador`), `docs/img/v1.4/buscador/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `BarraInferior`, `FilaAnimes`, `GridFavoritos`, `TarjetaAnime`, la rejilla de resultados de búsqueda (T78) e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T76 no iniciada**.

## T76 — Filas y tarjetas del Home (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Rediseña los **carruseles** (`FilaAnimes`) y la **tarjeta** de anime (`TarjetaAnime`) con el lenguaje visual de la serie y lleva el buscador de **modo estrecho** al objetivo táctil. En **modo estrecho** (`< 900 px`) el carrusel **sangra hasta el borde** de la página (mismos márgenes/padding con `--gutter`, sin mangas de degradado), el **título de fila** corta con elipsis sin empujar las flechas, y las **flechas** pasan a `var(--tap)` (44×44) con ratón y se **ocultan** (`display: none`) con `pointer: coarse` (el desplazamiento es el gesto). La **tarjeta** gana marco (`--border-subtle`), radio `--r-lg`, sombra `--sh-card`, portada en `aspect-ratio: 2 / 3`, `:active`/`:focus-visible` con tokens y `:hover` solo con puntero fino. En **≥ 900 px** la geometría **no cambia** (tarjeta de Home **263,8**, flechas 32×32, degradados presentes, buscador 85×38). **Sin cambios de comportamiento**: el clic abre el anime, las flechas desplazan el 80 %, los extremos se deshabilitan, snap y carga de filas intactos.

**Resultados de la verificación** (modo `filas` nuevo en `verificar_ui.mjs`, localhost:34115, red real):

| ancho | tarj. Home (h) | portada (w×h) | 1.ª tarjeta (x) | carrusel right | flecha (w×h) | `mask` | `overflowH` |
|---|---:|---|---:|---:|---|:--:|---:|
| 360 | 263,8 | 140×210 | 16 | 360 | **44×44** (flex) | no | 0 |
| 390 | 263,8 | 140×210 | 16 | 390 | **44×44** (flex) | no | 0 |
| 768 | 263,8 | 140×210 | 24 | 768 | **44×44** (flex) | no | 0 |
| 1024 | 263,8 | 140×210 | 332 | 996 | 32×32 (flex) | **sí** | 0 |
| 1280 | 263,8 | 140×210 | 332 | 1252 | 32×32 (flex) | **sí** | 0 |
| 360 `coarse` | 263,8 | 140×210 | 16 | 360 | **0×0 (`none`)** | no | 0 |
| 390 `coarse` | 263,8 | 140×210 | 16 | 390 | **0×0 (`none`)** | no | 0 |

| comprobación (61 en total) | resultado |
|---|---|
| **Paso 0** — campo y botón "Buscar" en 360/390/768 ≤ 48 (y ≥ 44): **campo 46, botón 46** (input 44) — antes **64/66** | **sí** (6/6) |
| **Paso 0** — 1024/1280: input **38** y botón **38** (escritorio, sin cambio) | **sí** (4/4) |
| ≥ 1024: tarjeta de Home **263,8**; portada **140×210**; flechas **32×32**; **degradados presentes**; `overflowH = 0` | **sí** (10/10) |
| 360/390/768 (ratón): 1.ª tarjeta `left` = **gutter** (16/16/24); carrusel `right` = **ancho de ventana**; flechas **44×44**; **sin `mask-image`**; **título con elipsis** (`nowrap`/`ellipsis`/`min-width:0`); `overflowH = 0` | **sí** (28/28) |
| Tarjeta de Home = **263,8** en TODOS los anchos (tol > 1 px) | **sí** (7/7, Δ0) |
| **Esqueleto → datos:** posición vertical de la 1.ª fila `196 → 194` (**Δ2**, ≤ 2) | **sí** (2 px, en el límite) |
| 360/390 `coarse`: `pointer: coarse` detectado, flechas `display: none`, 1.ª tarjeta al gutter, `overflowH = 0` | **sí** (8/8) |
| 360 `coarse`: la fila se **desplaza con gesto táctil** (arrastre CDP) `scrollLeft = 4220 = max` (**llega al final**) | **sí** |
| 390 `coarse`: ídem `scrollLeft = 4190 = max` (**llega al final**) | **sí** |
| 360: clic en una tarjeta del Home **abre el detalle** y "Atrás" vuelve al Home | **sí** |
| 360: **T66/T67** — scroll del detalle **restaurado** al abrir un episodio (69 492 → 69 493, Δ1 px) | **sí** |
| `prefers-reduced-motion: reduce`: `transition-duration` de `.tarjeta` | **`0s`** |
| táctiles < 44 px visibles en el Home (360) | **0** (antes: 8 flechas de 32×32) |
| **Capturas 1280 antes (HEAD) / después (T76), tol ≤ 2** | **170 401 px distintos** (máx Δ229), caja `x=328, y=262, w=924, h=538` — **solo la zona de filas** |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| Go (`go test -race -count=1 -v ./...`) | **435 `--- PASS` de nivel superior**, 0 FAIL (11 paquetes `ok`, `*.go` intactos) |

**Desviaciones (documentadas):**
- **Portada 138 → 140 px (consecuencia directa del ítem 1).** Con el `aspect-ratio: 2 / 3` y el ancho del *padding box* (138 px que deja el borde de 1 px) la portada mediría 138×207 y la tarjeta bajaría a **260,8** (no 263,8). Para mantener las dos cifras que exige la ficha (**portada 140×210** y **tarjeta 263,8**), la portada ocupa el ancho COMPLETO del marco con `width: calc(100% + 2px)` + `margin: 0 -1px` (y `overflow: hidden` recorta). La portada crece pues **2 px** respecto al valor previo (138) y la comparación a 1280 lo refleja: además del **borde, radio y sombra** del marco, las **portadas se reescalan** (de ahí gran parte de los 170 401 px). **Fuera de la zona de filas** (cabecera y buscador) **no hay ninguna diferencia** (la caja del diff empieza en `y=262`, la primera fila).
- **Esqueleto → datos = 2 px** (en 360; la base HEAD daba 0). El sangrado del `.carrusel-fantasma` no cambia la posición de la fila, pero al resolver la carga el bloque de título de la tarjeta real y el fantasma difieren 2 px; la ficha tolera ≤ 2 px.
- **Arnés (no código):** el modo `filas` **retrasa la llamada `GetHomeSection`** (`addInitScript` sobre `window.go.main.App`) para poder medir el esqueleto: la caché de Home (TTL 10 min) hace que, en una segunda carga, la respuesta llegue en una sola pintura y el esqueleto no se pueda medir.
- **Sin cambios de markup ni `<script>`:** la ficha permitía tocar el markup de las flechas de `FilaAnimes`, pero **no hizo falta** (el estilo por media queries basta); el `<script>` de los cuatro componentes quedó **intacto**.

**Capturas** en `docs/img/v1.4/filas/`: `home_360.png` (viewport), `home_360_coarse.png` (con `pointer: coarse`), `home_768.png`, `home_1280.png`, `tarjeta_360.png` (primer plano de una tarjeta) (+ `filas.json`).

**Archivos tocados:** `frontend/src/TarjetaAnime.svelte` (SOLO `<style>`), `frontend/src/FilaAnimes.svelte` (SOLO `<style>`), `frontend/src/VistaInicio.svelte` (SOLO `<style>`), `frontend/src/VistaBusqueda.svelte` (SOLO `<style>` del buscador, paso 0), `docs/investigacion/ui/verificar_ui.mjs` (modo `filas`), `docs/img/v1.4/filas/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `GridFavoritos` (T87), la rejilla de resultados de búsqueda (T78), `BarraInferior` e `index.html` intactos; ningún `<script>` de componente cambiado. **Sin tag ni release** (los hace el humano); **T77 no iniciada**.

## T77 — Fila "Continuar viendo" del Home (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-17**. Añade al **Home móvil** (solo `$isNarrow`, `< 900 px`) una **fila de "Continuar viendo"** con las **3–10** entradas más recientes del historial, en **carrusel horizontal** sin flechas ni degradados (gesto/rueda), **primera** al `--gutter` y con el mismo **sangrado** que las filas de T76. Cada **tarjeta** (ancho fijo **240 px**, alto **86 px**) muestra **portada 48×68**, título a 2 líneas con elipsis, **episodio · tiempo** (o "Visto ✓"), **proveedor**, **barra de progreso** de 4 px (`role="progressbar"` + `aria-valuenow`) y, mientras se resuelve, "Buscando…"/mensaje de resultado. Reutiliza **sin lógica nueva** `historial.ts` (`tituloVisible`, `formatearTiempo`, `numeroDeEpisodio`, `fraccionProgreso`, `proveedorDe`, `NOMBRE_PROVEEDOR`) y `retomar.ts` (mismo mapeo que `BarraLateral.alRetomar`: `!ok`→"No se pudo retomar", `completed`→"Completado", `up_to_date`→"Al día", mensaje de 4 s, una resolución a la vez). En **≥ 900 px** la fila **no se monta** y el Home es **byte-idéntico a T76**. **Sin cambios** en `App.svelte`, `stores.ts`, `views.ts`, `historial.ts` ni `retomar.ts`.

**Resultados de la verificación** (modo `continuar` nuevo en `verificar_ui.mjs`, localhost:34115, red real):

| ancho | fila | tarjetas | 1.ª x | `--gutter` | tarjeta (w×h) | progreso | tarj. Home (h) | pestaña (h) | `overflowH` |
|---|:--:|---:|---:|---:|---|---|---:|---:|---:|
| 360 | **sí** | 3 | 16 | 16 px | **240×86** | 63 / 100 / 21 % | 263,8 | — | 0 |
| 390 | **sí** | 3 | 16 | 16 px | **240×86** | 63 / 100 / 21 % | 263,8 | — | 0 |
| 768 | **sí** | 3 | 24 | 24 px | **240×86** | 63 / 100 / 21 % | 263,8 | — | 0 |
| 1024 | **no** | 0 | — | 32 px | — | — | 263,8 | 38 | 0 |
| 1280 | **no** | 0 | — | 32 px | — | — | 263,8 | 38 | 0 |

| comprobación (43 en total) | resultado |
|---|---|
| 360/390/768: fila presente, título "Continuar viendo", **3 tarjetas 240 × ≥44 px**, 1.ª al `--gutter`, `overflowH = 0`, carrusel **sin `mask-image`**, **sin flechas** | **sí** (24/24) |
| 1024/1280: fila **ausente del DOM**; Home idéntico a T76 (tarj. Home **263,8**, pestaña **38**) | **sí** (6/6) |
| 3 **porcentajes distintos** y coherentes (21 / 63 / 100 %), 3 `progressbar` con `aria-valuenow`, **respaldo de portada** (inicial, sin `img`) | **sí** (4/4) |
| clic → **navega por `retomar`** (detalle/reproductor); "Atrás" **vuelve al Home** | **sí** (2/2) |
| **T66/T67** — Home conserva el scroll al volver (200 → 208, **Δ8**, ≤ 20; deriva de anclaje, ver desviaciones) | **sí** |
| **al salir del reproductor la fila recarga el historial** (JKAnime Bleach **21 → 100 %**) | **sí** |
| **sin historial** la fila no está en el DOM y el Home **no deja hueco** (1.er hijo = `.cabecera`); "**Deshacer**" la **restaura** (1 tarjeta) | **sí** (3/3) |
| `prefers-reduced-motion: reduce`: `transition-duration` de `.tarjeta` | **`0s`** |
| táctiles < 44 px visibles en el Home (360) | **0** (0 en la fila) |
| **Capturas 1280 (T76) vs (T77), tol ≤ 2, región `x ≥ 300`** | **idénticas** (1280×800) — la fila no existe en escritorio; el único diff (x < 300) es la **barra lateral** con datos de historial |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`; sin test nuevo, no hay lógica pura nueva) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| Go (`go test -count=1 -v ./...`) | **435 `--- PASS` de nivel superior**, 0 FAIL (`*.go` intactos) |

**Desviaciones (documentadas):**
- **Semilla del historial de T69 corrupta (arreglada).** El historial sembrado tenía `episode_id` sin *slug* (`animeav1:episode:9`), y `models.ParseEpisodeID` lo rechazaba ("id de episodio inválido: \"9\" no es un tipo válido sin número"), de modo que **`Retomar`/`GetNextEpisode` fallaban** para esas entradas. Se re-sembró por el puente (`ClearProgress` + `SaveProgress`) con el formato correcto **`provider:slug:episode:n`** (`animeav1:bleach:episode:9`, `jkanime:bleach:episode:5`, `jkanime:one-piece:episode:1100`); el modo `continuar` **siembra** con ese formato (auto-contenido). **No es un fallo del producto** (el `episode_id` lo genera el propio backend al guardar), sino de la semilla manual.
- **Deriva de scroll en el Home estrecho (~8 px).** Al volver al Home, `sincronizarVista` **restaura exactamente** la posición guardada (T66/T67), pero el **anclaje de scroll** del navegador la ajusta unos píxeles al **remontar las secciones asíncronas** del Home. La deriva es **previa a T77**: reproduciendo el mismo recorrido **sin la fila** (clic en una tarjeta de Novedades a 360) la desviación es **~14 px** (200 → 214). La fila no la introduce (queda por **debajo** de la zona observable). El modo `continuar` la **tolera (≤ 20 px)** y lo deja escrito; arreglarla del todo requeriría `overflow-anchor: none` en el contenedor de scroll (**CSS global / `App.svelte`**, fuera de "Puedes tocar"). Diagnóstico: con `html{overflow-anchor:none}` la restauración es exacta (200 → 200).
- **Filtro de región en el modo `comparar`.** `verificar_ui.mjs comparar` acepta ahora `[desdeX] [hastaX]` para excluir la **barra lateral** (que cambia con los datos de historial) y comparar solo el Home (`desdeX = 300`), lo que permite afirmar "idénticas a 1280" de forma honesta.
- **Clase `.fila-continuar` (no `.fila`).** Para no colisionar con los selectores de T76 (`filas`: `.fila .carrusel .tarjeta`) ni de T73 (`shell`: `.fila .carrusel .tarjeta`), la sección usa `class="fila-continuar"` con `aria-label="Continuar viendo"`. La ficha no fijaba la clase; se documenta.
- **Sin `<script>` en `VistaInicio.svelte` más allá del import.** El montaje es `<FilaContinuar />` como primer hijo del panel "Descubrir"; **no** se añadió lógica al `<script>` (solo la línea `import FilaContinuar`). `git diff` del archivo = **solo** esas 2 líneas.

**Capturas** en `docs/img/v1.4/continuar/`: `continuar_360.png` (viewport Home con la fila), `continuar_768.png`, `continuar_1280.png` (sin fila), `tarjeta_360.png` (primer plano) (+ `continuar.json`).

**Archivos tocados:** `frontend/src/TarjetaContinuar.svelte` (**nuevo**), `frontend/src/FilaContinuar.svelte` (**nuevo**), `frontend/src/VistaInicio.svelte` (**solo** `import` + `<FilaContinuar />`), `docs/investigacion/ui/verificar_ui.mjs` (modo `continuar` + `[desdeX] [hastaX]` en `comparar`), `docs/img/v1.4/continuar/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `historial.ts`, `retomar.ts`, `App.svelte`, `BarraLateral`, `TarjetaHistorial`, `FilaAnimes`, `TarjetaAnime`, `GridFavoritos` e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T78 no iniciada**.

## T78 — Rejilla de resultados de búsqueda y estados (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-18**. Rediseña la vista de **resultados de búsqueda** (`VistaBusqueda.svelte`) y sus tres estados para móvil, reutilizando las **primitivas de T72** y la **tarjeta del Home de T76**:

- **Cargando**: contador vacío (`aria-live`) + una **rejilla de 6 esqueletos** (`aria-busy="true"`) con la **misma forma** que las tarjetas (portada 2:3 + bloque de título a 2 líneas) para que **no haya salto de layout** al llegar los datos.
- **Error**: `EstadoVista variante="error"` con título **"No se pudo buscar"**, el mensaje del backend y un `Btn` **"Reintentar"** que repite la consulta (`search($searchState.query)`).
- **Resultados**: contador **"{n} resultados"** (`aria-live`) + rejilla de **`TarjetaAnime`** (misma tarjeta del Home) con la insignia `.etiqueta` del proveedor vía `NOMBRE_PROVEEDOR` de `historial.ts` (**"JKAnime"/"AnimeAV1"**).
- **Vacío**: `EstadoVista variante="vacio"` con título **"Sin resultados"** y texto `No encontramos nada para "…".`.

La **rejilla** pasa de `auto-fill minmax(160px)` + `gap: 20px` (valores fijos) a **2 columnas** en móvil → **3** en tableta (`≥ 480 px`) → **auto-fill `minmax(150px)`** en escritorio (`≥ 900 px`), con **`gap: var(--sp-3)`** (12 px). La **lógica de búsqueda del `<script>` no se tocó**: `git diff` solo añade los `import` de `TarjetaAnime`/`EstadoVista`/`Esqueleto`/`Btn`/`NOMBRE_PROVEEDOR` (el resto del `<script>` —debounce, `limpiar`, `programar`, `buscarAhora`, `abrir`— intacto).

**Resultados de la verificación** (modo `busqueda` **nuevo** en `verificar_ui.mjs`, localhost:34115, red real; consulta "bleach"):

| ancho | columnas | gap | 1.ª tarjeta (w×h) | contador | `overflowH` |
|---|:--:|---:|---|---|---:|
| 360 | 2 | 12 px | 158×290,8 | "21 resultados" | 0 |
| 390 | 2 | 12 px | 173×313,3 | "21 resultados" | 0 |
| 768 | 3 | 12 px | 232×401,8 | "21 resultados" | 0 |
| 1024 | 4 (auto-fill) | 12 px | 156×287,8 | "21 resultados" | 0 |
| 1280 | 5 (auto-fill) | 12 px | 173,59×314,2 | "21 resultados" | 0 |

| comprobación (60 en total) | resultado |
|---|---|
| A: geometría por ancho (2/3/auto-fill columnas, gap 12 px, cada tarjeta ≥44×44, alto = **1,5·ancho + 53,8**, `overflowH = 0`, contador con `aria-live`, `.etiqueta` del proveedor) | **sí** (35/35) |
| A: tarjeta del Home **intacta** (263,8×140, portada 210) en los 5 anchos | **sí** (5/5) |
| B: **21/21 portadas** cargan tras el scroll real (lazy-loading) | **sí** |
| C: **6 esqueletos** + `aria-busy="true"`; **sin salto de layout** (Δ = **0 px** entre el 1.er fantasma y la 1.ª tarjeta) | **sí** (3/3) |
| D: error → título "No se pudo buscar", mensaje, botón **"Reintentar"** que **repite la consulta**, `role="alert"` y **sustituye** al viejo párrafo `.error` | **sí** (6/6) |
| E: vacío → título "Sin resultados", texto con la consulta, **sin acción**, **sin rejilla residual** | **sí** (4/4) |
| F: el clic abre el detalle; **Atrás** vuelve a los resultados | **sí** |
| I: **T66/T67** — el scroll del detalle se conserva al ir a un episodio y volver (69492 → 69493, Δ1, ≤ 5) | **sí** |
| G: `prefers-reduced-motion: reduce` → `transition-duration` de `.tarjeta` = **`0s`** | **sí** |
| táctiles < 44 px visibles en los resultados (360) | **0** |
| 1280: **zona de resultados contenida** (`x=332, y=144, 916×1652`) — el cambio visual queda **por debajo** de la cabecera/buscador (véase "zona de resultados @1280" en `busqueda.json`) | **sí** |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** (`index-BsRVigXN.js`, 708,69 kB) |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`; sin test nuevo, no hay lógica pura nueva) |
| `gofmt -l .` / `go vet ./...` / `go test -race -count=1 ./...` | **sin salida** / **sin salida** / **435 `--- PASS` de nivel superior**, 0 FAIL (`*.go` intactos) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Desviaciones (documentadas):**
- **El estado vacío no se dispara con `"zzzzqqqq"`.** La ficha pedía buscar `zzzzqqqq`, pero el **backend real devuelve 1 resultado** (JKAnime responde con un *fallback*: "Suzuka"), así que `.variante-vacio` nunca aparece. El modo `busqueda` intenta primero `zzzzqqqq` (12 s) y, si no hay vacío, **cae a una cadena larga** (`"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"`) que sí devuelve `[]`; queda registrado en `busqueda.json` (`vacio.conZzzzqqqq = false`). **No es un fallo del producto** (el ranking del backend ya hace fallback), sino de la consulta elegida en la ficha.
- **`TarjetaAnime.svelte` no se tocó.** El `flex: 0 0 140px` de la tarjeta **se ignora dentro de un grid**, así que la misma tarjeta del Home funciona en la rejilla sin cambios; el alto resultante (**1,5·ancho + 53,8** = portada 2:3 + bloque de título) se midió y coincide en los 5 anchos.
- **`.etiqueta` (no `.proveedor`)** es la insignia del proveedor en `TarjetaAnime`. Los helpers `abrirDetalle`/`abrirDetalleLargo` de `verificar_ui.mjs` se ajustaron a **`main .rejilla > .tarjeta`** (hijo directo) y a `.etiqueta` para seguir localizando la tarjeta de AnimeAV1 en otros modos (T74/T75/T76/T77). Verificado por el paso **I** del modo `busqueda`, que ejerce `abrirDetalle` con `animeav1`.
- **Fantasmas con medidas replicadas de la tarjeta.** `.fantasma-portada` usa `width: calc(100% + 2px); margin: 0 -1px` para compensar el borde de 1 px y cuadrar la portada con la de `TarjetaAnime`; el contador existe también en el estado de carga (vacío, con `min-height: calc(var(--fs-sm) * var(--lh))`) para que la rejilla no se desplace. Estrechamente acoplado a la tarjeta del Home (T76): si cambia, revisar.

**Capturas** en `docs/img/v1.4/busqueda/`: `resultados_360_arriba.png`, `resultados_360_scroll.png`, `resultados_768.png`, `resultados_1280.png`, `esqueleto_360.png`, `error_360.png`, `vacio_360.png` (+ `busqueda.json`).

**Archivos tocados:** `frontend/src/VistaBusqueda.svelte` (**markup + `<style>`**, lógica intacta), `docs/investigacion/ui/verificar_ui.mjs` (modo `busqueda` **nuevo** + `abrirDetalle`/`abrirDetalleLargo` a `.etiqueta`), `docs/img/v1.4/busqueda/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `historial.ts`, `GridFavoritos`, `FilaAnimes`, `TarjetaAnime`, `VistaInicio`, `BarraInferior`, `index.html` y los componentes de T72 intactos. **Sin tag ni release** (los hace el humano); **T79 no iniciada**.

## T79 — Ficha de detalle: portada, metadatos, géneros y sinopsis (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-18**. Rediseña la **ficha de detalle** (`VistaDetalle.svelte`, dentro de `section.ficha`) reutilizando las **primitivas de T72** (`Insignia`, `Icono`) y la cabecera de T74. **Solo se toca el `<style>` de la ficha y su markup**; el `<script>` recibe únicamente el import de `Insignia`/`Icono`/`isNarrow` y **24 líneas** (comentario incluido) para la sinopsis plegable; **el resto de la lógica del detalle (T66/T67, revelado T65, carga, `abrirEpisodio`, `continuar` y el botón «Continuar») queda intacto**.

- **Portada**: se envuelve en `.portada-marco` (padding 3 px con `--frame-poster`, radio `--r-lg`, sombra `--sh-panel`); la imagen mantiene `aspect-ratio: 2/3` y radio `--r-md`. En estrecho (`< 900 px`) la portada va **centrada** arriba (`width: min(45%, 200px)`) y los datos debajo; en ancho queda a la izquierda (220 px, **máx. 35 %** como hoy).
- **Metadatos**: la línea `·`-separada pasa a **4 insignias** (`Insignia`): proveedor en `tono="marca"` (valor crudo `jkanime`/`animeav1`), tipo (`etiquetaTipo`) y total (`N episodios`) en `tono="neutro"`, y estado neutro con un **punto** de 8 px (`--sp-2`) delante. El punto usa **`--brand-400` solo con el valor exacto `"En emisión"`** (grafía de AnimeAV1); en cualquier otro texto (p. ej. el `"En emision"` sin tilde de JKAnime) es neutro (`--text-3`). Si el estado viene vacío, el chip no se pinta.
- **Géneros**: en estrecho, **una sola línea con scroll horizontal** (sin barra nativa) que sangra hasta el borde de la ficha con márgenes negativos (`calc(-1 * var(--sp-4))`); en ancho, `flex-wrap: wrap` como hoy.
- **Sinopsis**: en estrecho se **recorta a 4 líneas** (`-webkit-line-clamp: 4`) y aparece un botón **«Ver más»/«Ver menos»** con `aria-expanded` y `min-height: var(--tap)` (44 px) e icono `chevron-abajo`; solo se muestra si la sinopsis **desborda** (medido por `scrollHeight > clientHeight`, en `requestAnimationFrame` y al `resize`, nunca con la sinopsis abierta). En ancho no hay recorte ni botón. El botón **«Continuar»** sigue debajo, intacto (lo rediseña **T80**).

**Resultados de la verificación** (modo `ficha` **nuevo** en `verificar_ui.mjs`, localhost:34115, red real; AnimeAV1/JKAnime «bleach»):

| ancho | cols rejilla | ficha w×h | marco w×h | chips | géneros | sinopsis líneas | «Ver más» | tarj.ep w×h | `overflowH` |
|---|:--:|---:|---:|---:|---|---:|:--:|---:|---:|
| 360 | 1 | 328×521,22 | 132,3×195,44 | 4 | nowrap/auto | 4 | 44 px | 328×170,39 | 0 |
| 390 | 1 | 358×509,08 | 145,8×215,69 | 4 | nowrap/auto | 4 | 44 px | 358×170,39 | 0 |
| 768 | 3 | 720×590,39 | 200×297 | 4 | nowrap/auto | 4 | 44 px | 228×170,39 | 0 |
| 1024 | 3 | 660×740,39 | 216,3×321,44 | 4 | wrap | 24 | — | 208×170,39 | 0 |
| 1280 | 4 | 916×524,39 | 220×327 | 4 | wrap | 15 | — | 215,5×170,39 | 0 |

(JKAnime «bleach»: 360 → 328×521,22 · 768 → 720×590,39 · 1280 → 916×404,39.)

| comprobación (106 en total) | resultado |
|---|---|
| A: 5 anchos — `overflowH = 0`, **4 chips** (marca + 3 neutras), portada centrada ≤ 200 px (estrecho) / 220 px-máx 35 % (ancho), marco padding 3 px, aspecto 2/3, imagen radio 12 px | **sí** |
| A: estrecho — sinopsis **4 líneas** recortada (`-webkit-line-clamp: 4`), «Ver más» ≥ 44 px con `aria-expanded=false`, géneros `nowrap`/`auto`, ficha padding 16 px | **sí** |
| A: ancho — **sin** «Ver más», sinopsis sin límite, géneros `wrap`, punto neutro («Finalizado») | **sí** |
| B: 360 — al pulsar «Ver más» la ficha crece (**521,22 → 1169,22**) y pasa a «Ver menos» (`aria-expanded=true`); al colapsar vuelve a 4 líneas | **sí** |
| C: resize **800 → 1000 → 800** — a 1000 desaparece el botón/recorte; de vuelta a 800 reaparece **conservando el estado abierto** («Ver menos») | **sí** |
| D: los géneros **se desplazan de verdad** (`scrollLeft = 200`) | **sí** |
| E: dato real AnimeAV1 «En emisión» (`battle-spirits-re-zekkai-no-kuu`) → chip `En emisión` y **punto `--brand-400`** `rgb(52,211,153)` | **sí** |
| F: estado vacío simulado → **3 chips**, sin punto | **sí** |
| G: sinopsis corta simulada → **sin** botón «Ver más», sin recorte | **sí** |
| H: JKAnime 360/768/1280 — 4 chips, `overflowH = 0`, punto neutro («Concluido»), «Ver más» solo en estrecho | **sí** |
| I: rejilla de episodios **sin cambios** — tarjeta de episodio **170,39 px** de alto y mismas columnas (3 @1024, 4 @1280) | **sí** |
| J: revelado T65 (`.entra = 4`), cabecera fija T74 (`sticky`) y scroll T66/T67 restaurado | **sí** |
| K: `prefers-reduced-motion: reduce` → «Ver más» sin transición (`0s`) | **sí** |
| L: táctiles < 44 px en el detalle (360) — **1** (el botón «Continuar», 103×35; lo rediseña **T80**); «Ver más» **no** aparece en la lista (es ≥ 44) | **sí** |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** (`index-Qn4W6-Ir.js`, 710,34 kB) |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`; sin test nuevo, no hay lógica pura nueva) |
| `gofmt -l .` / `go vet ./...` / `go test -race -count=1 ./...` | **sin salida** / **sin salida** / **435 `--- PASS` de nivel superior** (+281 subtests), 0 FAIL (`*.go` intactos) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Geometría (desviaciones > 2 px, documentadas):**
- **Ancho estrecho (360):** la ficha pasa de **~1000/1216 px** (sinopsis completa, sin recorte) a **521,22 px** cerrada (**1169,22** abierta) — es el objetivo de T79 (la sinopsis se pliega).
- **641–899 px (p. ej. 768):** la portada deja de ir **en fila** (se mostraba a la derecha en 768) y pasa a **columna** con la portada centrada; la ficha queda en **590,39 px** (antes ~657). El breakpoint de la portada cambia de **640 px a 899 px** (641–899 ya no son «ancho»).
- **≥ 900 px:** la ficha crece **~11 px** (1280: **513 → 524,39**; JKAnime 393 → 404,39) por el borde de 1 px, las insignias (una fila de ~24 px frente a la línea de texto) y el marco de la portada. La **rejilla de episodios no se mueve más allá de ese corrimiento**: tarjeta **170,39 px** y columnas idénticas (paso I).

**Desviaciones (documentadas):**
- **`box-sizing: border-box` en `.portada-marco`.** Sin él, `max-width: 35 %` se aplicaba al **contenido** y el marco medía **226 px** (1280) / 222,3 (1024) en vez de cuadrar con la portada previa. Con `border-box`, el ancho **externo** coincide con el de hoy (**220 px** / 216,3 px @1024); la imagen interior queda ~6 px menor (el padding de 3 px del marco). Ancho «hoy» tomado de la portada previa (mismo `max-width: 35 %`, mismo comportamiento).
- **Breakpoint de la portada estrecha 640 → 899 px.** Consecuencia de unificar el modo estrecho en `< 900 px`; a 768 la ficha ahora es columna (antes fila). Documentado en la geometría de arriba.
- **`padding` de la ficha en estrecho 20 → 16 px** (`var(--sp-4)`), como el resto de tarjetas de la serie; se suma al cambio de geometría de las vistas estrechas.
- **El punto de estado distingue la tilde.** Solo `"En emisión"` (AnimeAV1) pinta el punto de **marca**; JKAnime devuelve `"En emision"` (sin tilde) y sale **neutro** — es lo esperado por el diseño (comparación literal), no un fallo.
- **`abrirFilaHome` queda en el arnés sin uso** tras reescribir el paso E con **dato real** (busca «battle spirits» y abre el primer resultado AnimeAV1, que es el que emite). Se deja por si futuras verificaciones necesitan abrir una fila del Home por su título.

**Capturas** en `docs/img/v1.4/ficha/`: `detalle_animeav1_360.png`, `detalle_animeav1_360_cerrada.png`, `detalle_animeav1_360_abierta.png`, `detalle_sinopsis_corta_360.png`, `detalle_jkanime_360.png`, `detalle_jkanime_768.png`, `detalle_jkanime_1280.png` (+ `ficha.json`).

**Archivos tocados:** `frontend/src/VistaDetalle.svelte` (**markup + `<style>` de la ficha**; `<script>`: imports + **24 líneas** de sinopsis plegable), `docs/investigacion/ui/verificar_ui.mjs` (modo `ficha` **nuevo**), `docs/img/v1.4/ficha/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `CabeceraVista`, `TarjetaEpisodio`, `BotonFavorito`, `revelar.ts`, `ventana.ts`, `index.html`, la rejilla de episodios y el botón «Continuar» (T80) intactos. **Sin tag ni release** (los hace el humano); **T80 no iniciada**.

## T80 — Barra de acción fija «Continuar / Reiniciar» del detalle en estrecho (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-18**. Añade al **detalle** una **CTA fija** en modo estrecho (`< 900 px`, `$isNarrow`) pegada **justo encima de la barra inferior** (T73): acción principal **«Continuar · Ep. N»** —o **«Empezar · Ep. N»** si no hay progreso— y secundaria **«Reiniciar»** (solo cuando hay continuidad) que abre el primer episodio. En **≥ 900 px no cambia nada**: la CTA no existe y sigue el botón en línea «Continuar · Ep. N».

- **Componente nuevo `BarraAccion.svelte`** (solo presentacional, Svelte clásico con `export let`): props `etiquetaPrincipal`, `onPrincipal`, `deshabilitado`, `etiquetaSecundaria` y `onSecundaria`; `role="group"` + `aria-label="Acciones del anime"`; primario `Btn` con `Icono play` y `bloque` dentro de un contenedor `flex: 1 1 auto`; secundario `Btn` solo si `etiquetaSecundaria !== ''`. Alto del botón `var(--tap)` (44 px).
- **Estilo (tokens, sin literales)**: `position: fixed; left/right: 0; bottom: calc(var(--bar-h) + var(--sab)); z-index: var(--z-bar)`; `padding: var(--sp-2) var(--gutter)`; `background: var(--surface-panel)`; `border-top: 1px solid var(--border)`; blur `--blur-md`.
- **`VistaDetalle.svelte`**: `--cta-h: 60px` local en `.contenedor`; en `@media (max-width: 899px)` se oculta `.continuar` (`display: none`) y `main.contenedor.con-cta` añade `padding-bottom: calc(var(--sp-4) + var(--cta-h))`; la CTA se pinta tras `</main>` bajo `{#if $isNarrow && $currentDetail && episodios.length > 0}`.
- **Lógica nueva (15 líneas efectivas en `<script>`, + import y comentarios)**: `$: episodioInicio = episodios[0] ?? null` (lista ya ordenada), `reiniciar()` → `abrirEpisodio(episodioInicio)`, `accionPrincipal(continuarCon, inicio)` → etiqueta/acción, y `$: accion = accionPrincipal(episodioContinuar, episodioInicio)`.
  - **Gotcha Svelte 5 legacy (importante)**: una **llamada a función en el template** se compila con `$.untrack(...)`, de modo que **no crea dependencias** y la etiqueta se quedaba congelada en «Empezar» aunque llegara el progreso. Por eso el resultado se guarda en `$: accion` **pasando los episodios como argumentos explícitos** (si no, el compilador genera un thunk de dependencias vacío). Es la razón de que `accionPrincipal` reciba parámetros en vez de leer las variables de módulo.

**Resultados de la verificación** (modo `cta` **nuevo** en `verificar_ui.mjs`, localhost:34115, red real; AnimeAV1/JKAnime «bleach», historial sembrado AnimeAV1 ep. 9):

| ancho | CTA | hueco CTA/barra | botones w×h | principal | secundario | `.continuar` | tarj.ep w×h | `overflowH` |
|---|:--:|---:|---|---|---|:--:|---:|---:|
| 360 | sí | 0 px | 227,45×44 · 92,55×44 | Continuar · Ep. 9 | Reiniciar | oculto | 328×170,39 | 0 |
| 390 | sí | 0 px | 257,45×44 · 92,55×44 | Continuar · Ep. 9 | Reiniciar | oculto | 358×170,39 | 0 |
| 768 | sí | 0 px | 619,45×44 · 92,55×44 | Continuar · Ep. 9 | Reiniciar | oculto | 228×170,39 | 0 |
| 1024 | no | — | — | — | — | visible | 208×170,39 | 0 |
| 1280 | no | — | — | — | — | visible | 215,5×170,39 | 0 |

| comprobación (56 en total) | resultado |
|---|---|
| A: 5 anchos — CTA presente/ausente según el corte, pegada a la barra (**hueco 0**), botones ≥ 44 px, `.continuar` oculto en estrecho y visible en ancho, tarjeta de episodio **170,39 px** (T79), `overflowH = 0` | **sí** |
| B: 360 con progreso — «Continuar» abre **Ep. 9**, la CTA **desaparece en el reproductor** y **reaparece al volver**; «Reiniciar» abre **Ep. 1** | **sí** |
| C: 360 sin progreso (Naruto) — **«Empezar · Ep. 1»**, **sin** secundario; el principal abre el **Ep. 1** | **sí** |
| D: fin de scroll — el último episodio (**366**) queda completo sobre la CTA (**28,97 px** de hueco); revelado T65 (`.entra = 4`), cabecera fija T74 (`sticky`) y scroll T66/T67 restaurado | **sí** |
| E: lotes JKAnime — la CTA aparece con el primer lote («Empezar · Ep. 1») y su texto **no cambia** al llegar el resto (**366** episodios) | **sí** |
| F: con el cajón de Historial abierto, el punto de la CTA cae **en el cajón** (`z-index` cajón **50** > CTA **30**), no se ve encima | **sí** |
| G: `prefers-reduced-motion: reduce` → CTA sin transición (`0s`) | **sí** |
| H: táctiles < 44 px en el detalle estrecho — **0** dentro de la CTA (y **0** en total; el antiguo «Continuar» de 35 px ya no se pinta) | **sí** |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`; sin lógica pura nueva) |
| `gofmt -l .` / `go vet ./...` / `go build ./...` / `go test -race -count=1 ./...` | **sin salida** / **OK** / **OK** / **435 `--- PASS`** de nivel superior, 0 FAIL (`*.go` intactos) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Desviaciones / notas (documentadas):**
- **`z-index` de la CTA = `var(--z-bar)` (30).** Es la misma capa que la barra inferior; el cajón (`--z-drawer`, 50) y su velo (40) la **cubren**, como pide la tarea. No fue necesario `calc(var(--z-bar) - 1)`.
- **El gotcha de `untrack`** obligó a añadir el derivado `$: accion` y a que `accionPrincipal` reciba los episodios por parámetro (ver arriba). El `<script>` queda en **15 líneas efectivas** (sin contar import ni comentarios), dentro del tope.
- **Deriva de scroll T66/T67 pre-existente (no es de T80).** Al ir a un episodio y volver desde el **fondo** del detalle, la posición guardada (**68 896**) puede superar el máximo tras la vuelta (**68 860**) porque el alto del detalle cambia unas decenas de px al asentarse la rejilla; el navegador recorta a **68 859** (**Δ1** respecto al máximo). Se comprobó **revirtiendo T80** (`git stash` de `VistaDetalle.svelte`): el modo `ficha` de T79 falla igual (**Δ≈35 px**). El modo `cta` mide contra `min(yAntes, máximo)` para no marcar una regresión inexistente; **T79 y T80 no rompen T66/T67**, pero conviene saber que el check de `ficha` es sensible a ese asentamiento (puede salir «NO» de forma intermitente).
- **Sin cambios en `.continuar` en ancho**: sigue con el mismo texto/acción (solo se oculta en estrecho).

**Capturas** en `docs/img/v1.4/cta/`: `detalle_360_progreso.png`, `detalle_360_sin_progreso.png`, `detalle_360_fin_scroll.png`, `detalle_360_cajon.png`, `detalle_768.png`, `detalle_1280.png` (+ `cta.json`).

**Archivos tocados:** `frontend/src/BarraAccion.svelte` (**nuevo**), `frontend/src/VistaDetalle.svelte` (import + **15 líneas** de `<script>`, `class:con-cta` y markup de la CTA, y reglas de `<style>`), `docs/investigacion/ui/verificar_ui.mjs` (modo `cta` **nuevo** + `enCta` en `TACTILES_CHICOS`), `docs/img/v1.4/cta/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `App.svelte`, `BarraInferior`, `CabeceraVista`, `TarjetaEpisodio`, `BotonFavorito`, `revelar.ts`, `ventana.ts`, `index.html`, la lógica de scroll (T66/T67) y el revelado (T65) intactos. **Sin tag ni release** (los hace el humano); **T81 no iniciada**.

## T81 — Tarjeta de episodio y rejilla en modo estrecho (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-18**. Rediseña la **tarjeta de episodio** (`TarjetaEpisodio.svelte`) y la **rejilla** del detalle a **2 columnas** en estrecho (**< 600 px**) y **3** entre **600–899 px**, con **altura constante por rango**. En **≥ 900 px** la rejilla sigue con `auto-fill` y la tarjeta conserva **170,39 px** (T79) — **sin cambio de comportamiento** (clic abre episodio, `abrir`, progreso, revelado T65 en `.celda`, `content-visibility: auto`).

- **Miniatura 16:9** (`aspect-ratio: 16/9`, `--bg-2`) con **insignia** abajo-izquierda (`--surface-panel`, `--brand-400`, `--fs-xs`, `--r-sm`) y **franja de progreso de 4 px** (`--sp-1`) pegada al borde inferior de la miniatura (`--border-strong` con relleno `--brand-500`, `role="progressbar"` + `aria-valuenow`). La **miniatura es `flex: 0 1 auto`** y **cede altura** si el cuerpo necesita más, en vez de recortarlo.
- **Cuerpo** (`.cuerpo`, `flex: 1 1 auto`, `justify-content: space-between`, padding `var(--sp-1) var(--sp-3)`): **título** a una línea con elipsis (`--fs-sm`, `--lh-tight`) y **estado**. **Visto** → insignia con `Icono check` (tam 16), texto «Visto», fondo `--surface-chip`, color `--brand-400` y borde `--border-accent`. **En progreso** → texto «{n} %» (`--fs-xs`, `--text-3`) **bajo el título**. **Sin progreso** → **sin** elemento de estado. Por eso el **alto no depende del estado**.
- **Altura fija por rango** (clave para el alto estable con `aspect-ratio`): `height: 170.39px` en escritorio; **146 px** con 2 columnas y **179 px** con 3 columnas, cada una con su `contain-intrinsic-size: auto <alto>px` (escritorio mantiene `auto 170px`). Como el ancho de tarjeta cambia con la columna, un `aspect-ratio` puro no podía mantener 170,39 en ≥ 900 (daría 113–120 px de miniatura); con la altura fija el objetivo se cumple exacto.
- **Sin literales**: eliminados `#14532d`, `#52525b`, `#2a2a35`, `#26262e`; el hueco sin miniatura usa `--bg-2` con el número en `--text-3` y `repeating-linear-gradient` de tokens (`--bg-2`/`--bg-1`). Tipografía mínima `--fs-xs` (12 px); `font: inherit`.
- **Interacción**: `:hover` (`translateY(-3px)` + `--sh-card-hover`) **solo** en `@media (hover: hover) and (pointer: fine)`; `:active` `scale(0.97)` con `--dur-fast`; `:focus-visible` con `box-shadow: var(--ring)`; `prefers-reduced-motion: reduce` sin transiciones ni transform. Radio `var(--r-lg)`, borde `var(--border-subtle)`.
- **Rejilla** (`VistaDetalle.svelte`, **solo `<style>`**): `@media (max-width: 899px) { .rejilla { grid-template-columns: repeat(2, minmax(0,1fr)); gap: var(--sp-3); } }` y `@media (min-width: 600px) and (max-width: 899px) { .rejilla { grid-template-columns: repeat(3, minmax(0,1fr)); } }`. En ≥ 900 no se toca el `auto-fill`.

**Resultados de la verificación** (modo `episodios` **nuevo** en `verificar_ui.mjs`, localhost:34115, red real; AnimeAV1 «bleach», historial sembrado ep. 9 en progreso y ep. 10 visto — 366 episodios):

| ancho | cols | tarj. w | tarj. alto (min..max) | gap | `overflowH` |
|---|:--:|---:|---|---:|---:|
| 360 | 2 | 158 | **146** | 12 px | 0 |
| 390 | 2 | 173 | **146** | 12 px | 0 |
| 768 | 3 | 232 | **179** | 12 px | 0 |
| 1024 | 3 | 208 | **170,39** | 18 px | 0 |
| 1280 | 4 | 215,5 | **170,39** | 18 px | 0 |

| comprobación (46 en total) | resultado |
|---|---|
| A: 5 anchos — `overflowH = 0`, columnas esperadas (2/2/3/3/4), alto **constante** por rango, `contain-intrinsic-size` = alto (escritorio `auto 170px`) | **sí** |
| B: los **tres estados** (visto / en progreso / sin progreso) existen; «Visto» con icono + `--surface-chip`/`--brand-400`/`--border-accent`; franja **4 px pegada** con `aria-valuenow = 63` y «63 %»; sin progreso **sin** elemento de estado; **los tres con el mismo alto** | **sí** |
| C: título `--fs-sm` (14), ningún texto < `--fs-xs` (12), miniatura **16:9** en los tres estados | **sí** |
| D: miniatura rota → hueco con el número en `--text-3` y **alto sin cambio** (146 → 146) | **sí** |
| E: hover solo con puntero fino (`translateY(-3px)` + sombra); contexto táctil **no** es puntero fino | **sí** |
| F: revelado T65 (`.entra = 10`) y scroll T66/T67 restaurado (Δ0) | **sí** |
| G: toda tarjeta de episodio ≥ 44 px de alto (mín **146**) | **sí** |
| modo `cta` (T80) re-ejecutado — **56/56 OK**, tarjeta de episodio **170,39** a 1024/1280 | **sí** |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`; sin lógica pura nueva) |
| `gofmt -l .` / `go vet ./...` / `go test -race -count=1 ./...` | **sin salida** / **OK** / **435 `--- PASS`** de nivel superior, 0 FAIL (`*.go` intactos) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Desviaciones / notas (documentadas):**
- **Altura fija por rango, no `aspect-ratio` puro.** Con el ancho de tarjeta de T79 (208 px a 1024; 215,5 a 1280) una miniatura puramente 16:9 daría 117–121 px de alto y la tarjeta variaría unos 4 px; en estrecho (158–232 px) la variación sería mayor. La tarjeta fija el alto por rango y deja que la miniatura **ceda** (`flex: 0 1 auto`) si hace falta, de modo que el alto es **exacto** (170,39 / 146 / 179) y **contenido** (sin recorte del cuerpo).
- **Contenido de la tarjeta**: solo se muestra **una línea** de título (con elipsis). Es lo que garantiza «altura constante sin depender del contenido»; se aceptó el recorte de títulos largos.
- **Rejilla 600–899**: se apila con la media del propio componente (`min-width: 600px`) y la de la vista; ambas coinciden porque la rejilla define las columnas por viewport. A **900 px** exactos la tarjeta ya es de escritorio (**170,39**) y la rejilla vuelve a `auto-fill` (2 columnas a 900 con el lateral expandido, 3 a 1024, 4 a 1280 y 1100).
- **Sin cambios en el resto**: la rejilla de resultados de búsqueda, el `Home`, la cabecera, la ficha, la CTA y la barra inferior son **byte-idénticos**; no se tocó ningún `<script>` salvo **importar `Icono`** en `TarjetaEpisodio.svelte`. El `<script>` de `VistaDetalle.svelte` quedó **intacto** (solo su `<style>`).

**Capturas** en `docs/img/v1.4/episodios/`: `episodios_{360,390,768,1024,1280}.png` (inicio), `episodios_{360,390,768,1024,1280}_rejilla.png` (rejilla desplazada), `episodios_360_estados.png`, `episodios_360_hueco.png` (+ `episodios.json`).

**Archivos tocados:** `frontend/src/TarjetaEpisodio.svelte` (**markup + `<style>`**; en `<script>`, solo `import Icono`), `frontend/src/VistaDetalle.svelte` (**solo `<style>`**: `.rejilla` en estrecho), `docs/investigacion/ui/verificar_ui.mjs` (modo `episodios` **nuevo**), `docs/img/v1.4/episodios/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `revelar.ts`, `ventana.ts`, `App.svelte`, `BarraInferior`, `BarraAccion`, `CabeceraVista`, `BotonFavorito`, `index.html`, la CTA (T80), la ficha (T79) y el **`<script>` de `VistaDetalle.svelte`** intactos. **Sin tag ni release** (los hace el humano); **T82 no iniciada**.

## T82 — Orden ascendente/descendente de episodios (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-19**. Añade un **botón** a la derecha de `h3.titular` («Episodios (N)», texto intacto) que **invierte la presentación** de la rejilla de episodios del detalle. **Solo presentación**: la lista ascendente sigue siendo la fuente de la CTA, de `episodioContinuar`/`episodioInicio`, de la carga y del scroll T66/T67.

- **`<script>` de `VistaDetalle.svelte` (8 líneas, contando comentarios):** `let ordenDesc = false;`, `$: episodiosVista = ordenDesc ? [...episodios].reverse() : episodios;`, el reinicio `$: if ($currentAnimeID) ordenDesc = false;` y `function alternarOrden(): void { ordenDesc = !ordenDesc; }`. `episodiosVista` se usa **solo** en el `{#each}` de la rejilla; la clave sigue siendo `ep.id`.
- **Botón**: `<button type="button" class="orden">` con `Icono orden-asc`/`orden-desc` (tam 20) y `<span class="orden-texto">` («Más recientes»/«Más antiguos») visible solo en **≥ 480 px**; `aria-label` completo siempre y `aria-pressed={ordenDesc}`; existe **solo** si `episodios.length > 1`. Estilo con `--surface-chip`, `--border`, `--r-md`, `--fs-sm`; mínimo `var(--tap)` (44) en estrecho, **36 px** de alto en **≥ 900**; `:hover` solo en `(hover: hover) and (pointer: fine)`; `:active` y `:focus-visible` (`--ring`); sin transición con `prefers-reduced-motion: reduce`.

**Resultados de la verificación** (modo `orden` **nuevo** en `verificar_ui.mjs`, localhost:34115, red real; AnimeAV1 «bleach» con historial sembrado ep. 9 → CTA «Continuar · Ep. 9» — 366 episodios — y JKAnime):

| ancho | botón w×h | texto | 1.º/últ. asc | 1.º/últ. desc | tarjetas | CTA | cols | alto | `overflowH` |
|---:|---|---|---|---|---:|---|---:|---:|---:|
| 360 | 46×44 | oculto | Ep. 1 / Ep. 366 | Ep. 366 / Ep. 1 | 366 | Continuar · Ep. 9 | 2 | 146 | 0 |
| 390 | 46×44 | oculto | Ep. 1 / Ep. 366 | Ep. 366 / Ep. 1 | 366 | Continuar · Ep. 9 | 2 | 146 | 0 |
| 768 | 145,14×44 | «Más recientes» | Ep. 1 / Ep. 366 | Ep. 366 / Ep. 1 | 366 | Continuar · Ep. 9 | 3 | 179 | 0 |
| 1280 | 145,14×36 | «Más recientes» | Ep. 1 / Ep. 366 | Ep. 366 / Ep. 1 | 366 | Continuar · Ep. 9 | 4 | 170,38 | 0 |

| comprobación (89 en total) | resultado |
|---|---|
| A: 4 anchos — botón presente, `aria-pressed` `false→true→false`, `aria-label` coherente, ≥ 44×44 (360/390/768) y 36 de alto (1280); texto visible solo ≥ 480; tokens `--surface-chip`/`--border`/`--r-md`/`--fs-sm`; inversión Ep. 1↔366; **366 tarjetas** y **0 ids repetidos** en ambos órdenes; CTA/«Continuar · Ep. 9» sin cambio; columnas y altura de tarjeta sin cambio; `overflowH = 0` | **sí** |
| A: revelado T65 coherente (`.entra` + `.pendiente` = 366) y **0 reveladas pierden `.entra`** al invertir (`.entra` asc 2/4/6/4 y desc 14/14/21/32 por ancho) | **sí** |
| B: JKAnime 360 — botón presente, más de un episodio, orden se invierte (primero ↔ último), sin ids repetidos | **sí** |
| C: anime de 1 episodio (`your name` → «Bleach Movie 3…», JKAnime) — 1 tarjeta y **sin botón** | **sí** |
| D: invertir con la lista desplazada (scroll 1600) — cabecera `sticky` en `y = --sat`, CTA pegada a la barra inferior, `overflowH = 0` | **sí** |
| E: detalle (desc) → reproductor → Atrás — orden vuelve a ascendente (`aria-pressed = false`) y scroll T66/T67 restaurado (Δ0) | **sí** |
| F: hover solo con puntero fino usa `--surface-chip-hover`; `prefers-reduced-motion` → `0s` de transición | **sí** |
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** |
| tests `.ts` | **13 / 48 / 32** comprobaciones OK (`views`, `ventana`, `etiquetas`; sin lógica pura nueva) |
| `gofmt -l .` / `go vet ./...` / `go vet -tags integration ./...` / `go test -race -count=1 ./...` | **sin salida** / **OK** / **OK** / **435 `--- PASS`** de nivel superior, 0 FAIL (`*.go` intactos) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Desviaciones / notas (documentadas):**
- **`.revelada` no existe**: T82 pedía reportar cuántas reveladas pierden `.revelada`; `revelar.ts` usa **`.entra`** (y `.pendiente`). El modo `orden` mide `.entra`/`.pendiente` y comprueba que **ninguna** revelada pierde `.entra` al invertir (0 pérdidas).
- **Reinicio del orden**: se hace con `$: if ($currentAnimeID) ordenDesc = false;`, sobre el id del anime seleccionado del store (`$currentAnimeID`), que también se resetea a `''` al cerrar el detalle. Reportado tal como pedía el punto 5.
- **Ajuste del propio modo `orden` durante la sesión** (no del producto): (1) los tokens `--r-md`/`--fs-sm` (radio y tamaño, no colores) se resuelven con `getComputedStyle(documentElement)`, porque el resolver de color del arnés no los devuelve; (2) la comprobación D compara la CTA con el borde superior de `.barra-inferior` (va pegada **encima** de ella), no con `innerHeight`.
- **Sin cambios en el resto**: la rejilla T81, la ficha T79, la CTA T80, la cabecera T74, el revelado T65, las columnas, la carga por lotes y el scroll T66/T67 quedan igual; el `<script>` de `VistaDetalle.svelte` solo recibe las 8 líneas indicadas.

**Capturas** en `docs/img/v1.4/orden/`: `detalle_360_asc.png`, `detalle_360_desc.png`, `detalle_1280_asc.png`, `detalle_1280_desc.png` (+ `orden.json`).

**Archivos tocados:** `frontend/src/VistaDetalle.svelte` (**`<script>`** con las 8 líneas indicadas + **markup** de `.fila-titular`/botón y `<style>` del botón), `docs/investigacion/ui/verificar_ui.mjs` (modo `orden` **nuevo**), `docs/img/v1.4/orden/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `revelar.ts`, `ventana.ts`, `TarjetaEpisodio`, `BarraAccion`, `CabeceraVista`, componentes T72, `index.html`, la CTA (T80), la ficha/rejilla (T79/T81) y el scroll T66/T67 intactos. **Sin tag ni release** (los hace el humano); **T83 no iniciada**.

## T83 — Layout del reproductor y selector de fuentes (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-19**. Cabecera alineada, marco 16:9 con borde/sombra de tokens y selector de fuentes con chips táctiles de 44 px en una fila con scroll horizontal en modo estrecho (`< 900 px`). En **≥ 900 px** la geometría del reproductor y de los chips no cambia (solo borde/sombra del marco).

- **`CabeceraVista.svelte` (solo `<style>`)**: en estrecho, `.cabecera:not(.fija)` deja `padding-inline: 0` (el gutter lo aporta el `.contenedor`); `.cabecera.fija` sigue exactamente igual, así que el detalle (cabecera fija T74) no cambia.
- **`VistaReproductor.svelte` (solo `<style>`)**: `.contenedor` estrecho → `padding: 0 var(--gutter) calc(var(--sp-4) + var(--sab))`; `.marco` → `--r-lg` + `1px solid var(--border-subtle)` + `var(--sh-player)` (mantiene 16:9, `overflow: hidden` y `--bg-0`); `.aviso`/`.error-caja` → `margin: var(--sp-3) 0`, `--fs-sm` y `--r-md`.
- **`SelectorFuente.svelte`**: en el `<script>` SOLO se añade `import Icono` (lógica intacta: `agruparFuentes`, `alternar`, pliegue). Markup: `aria-pressed` en todos los chips, `Icono check` (16) en el activo y `chevron-der`/`chevron-abajo` (16) en el plegable. Estrecho: `.botones` en una fila con `flex-wrap: nowrap`, scroll horizontal con sangrado al gutter, `scroll-snap`, `scrollbar-width: none` y `overscroll-behavior-x: contain`; chips `min-height: var(--tap)`, `padding: 0 var(--sp-4)`, `--fs-sm`, `--r-full`, `--surface-chip`; activo con `--grad-chip-active`, `--brand-400`, `--on-brand` y `--fw-semibold`; `:hover` solo con puntero fino, `:active` `scale(0.97)`, `:focus-visible` con `--ring`. Escritorio: envoltura y tamaño originales (13 px).

**Resultados reales** (modo `reproductor` **nuevo** en `verificar_ui.mjs`, localhost:34115, red real; AnimeAV1 «bleach»):

| ancho | gutter | Atrás x | marco x/y w×h | ratio | chips | alto chip | wrap | `overflowH` | barra inferior |
|---:|---:|---:|---|---:|---:|---:|---|---:|:--:|
| 360 | 16 px | 16 | 16/84 330×186,5 | 1,77 | 4 (+8 extras) | 44 | nowrap | 0 | no |
| 390 | 16 px | 16 | 16/84 360×203,38 | 1,77 | 4 | 44 | nowrap | 0 | no |
| 768 | 24 px | 24 | 24/84 722×407 | 1,77 | 4 (+8 extras) | 44 | nowrap | 0 | no |
| 1024 | 32 px | 32 | 32/107 962×542 | 1,77 | 4 | 31 | wrap | 0 | no |
| 1280 | 32 px | 90 | 90/107 1102×620,75 | 1,78 | 4 | 31 | wrap | 0 | no |

- **54 / 54 comprobaciones OK**. Chips en una fila (`scrollHeight = clientHeight = 44`) y scroll hasta el último extra (828 px de recorrido a 360, 434 a 768); plegable de 50 px (360) / 44 px (768) con `aria-expanded` correcto; activo con check visible, `aria-pressed="true"` y degradado de marca; cambiar de fuente deja activo el segundo chip (`aria-pressed` coherente). Con iframe activo el pliegue **se abre solo** (hasta ~30 s por los refrescos de fuentes en Docker) y el aviso queda **12 px** debajo del marco (`--fs-sm`, margen `--sp-3`).
- Escritorio: chips de **31 px** de alto y **13 px** de fuente, igual que T69 (anchos 95 / 94,3 / 186,78 / 187,5 vs 95 / 94 / 187 / 188 → Δ ≤ 1 px); sin scroll horizontal (`flex-wrap: wrap`), con una sola fila visual porque solo hay 4 fuentes. Plegable 30 px (T69: 29; Δ1 por el icono de 16 px).
- Sin barra inferior en el reproductor; fin del scroll sin nada tapado; atajo **N** (Ep. 1 → Ep. 2) y «← Atrás» OK; scroll T66/T67 restaurado (Δ0); `prefers-reduced-motion` → transición `0s` en chips y plegable.
- Literales de color en lo tocado: **0** (la única excepción es el scrim `rgba(0,0,0,0.65)` del spinner, ya documentado). Táctiles < 44 px restantes en T83: `BotonSiguiente` 147×36 (T84 lo corrige a 44 px).

**Desviaciones / notas (documentadas):**
- **768 px usa `--gutter: 24px`**, no 16: el token de T70 sube a 24 desde 768. Verificado que Atrás = marco = gutter (24) en 768.
- **`--fs-sm` = 14 px**, no 13: en escritorio el punto 6 pedía tokenizar `13px` solo si el tamaño resultante era el mismo, así que se deja `font-size: 13px` y se reporta (+1 px); en estrecho sí se usa `--fs-sm`.
- **`reanudar` al cambiar de fuente no se puede medir en Docker** (el proxy local no sirve el HLS y no hay `currentTime`); el cambio de fuente y su estado activo sí se verifican. La lógica de reanudación es la de T27/T63 y no se tocó.
- **Aviso del proxy ausente en el estado medido**: en el navegador el aviso que aparece es el del iframe («Fuente embebida: sin control de progreso»), separado 12 px del marco; el aviso de proxy solo se muestra si la URL no es local.
- **1024/1280 con una fila visual**: `flex-wrap: wrap` está activo y no hay scroll horizontal; con 4 fuentes caben en una fila. No es un fallo de la especificación de envoltura.
- **Modo nuevo del arnés**: `verificar_ui.mjs` añade `MEDIR_REPRODUCTOR` + `reproductor()` (+ dispatch); no toca código de producto.

**Comprobaciones finales:** `npm run check` **0 errores / 0 avisos** · `npm run build` **OK** · tests `.ts` **48 / 32 / 13** OK (`ventana`, `etiquetas`, `views`) · `gofmt -l .` **sin salida** · `go vet ./...` y `go vet -tags integration ./...` **OK** · `go test -race -count=1 ./...` **435 `--- PASS`** de nivel superior, 0 FAIL (`*.go` intactos) · `grep wailsjs frontend/src` **solo `api.ts`**.

**Capturas** en `docs/img/v1.4/reproductor/`: `reproductor_360.png`, `reproductor_360_extras.png`, `reproductor_768.png`, `reproductor_1280.png` (+ `reproductor.json`).

**Archivos tocados:** `frontend/src/SelectorFuente.svelte` (script: solo `import Icono`; markup y `<style>`), `frontend/src/VistaReproductor.svelte` (**solo `<style>`**), `frontend/src/CabeceraVista.svelte` (**solo `<style>`**), `docs/investigacion/ui/verificar_ui.mjs`, `docs/img/v1.4/reproductor/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `BotonSiguiente.svelte`, el `<script>`/markup de `VistaReproductor`, `BotonFavorito`, `etiquetas.ts` e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T84 completada en el bloque siguiente**.

## T84 — Chip "Siguiente" del reproductor (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-19**. Solo estilos: el chip «Siguiente ▸» pasa de **~34 a 44 px** (`--tap`), se recoloca a `right: var(--sp-4)` / `bottom: calc(var(--tap) + var(--sp-4))` y el overlay de carga sube a **`z-index: 4`** para cubrirlo. **No cambian** `<script>` ni markup de ningún componente, ni el comportamiento de clic, tecla N o estados.

- **`BotonSiguiente.svelte` (solo `<style>`)**: chip con `min-height: var(--tap)`, `padding: 0 var(--sp-4)`, `--r-full`, `--fs-sm`, `--surface-panel`, `backdrop-filter: blur(var(--blur-sm))`, `--border`, `--text-1`, `--fw-semibold` y `font-family: inherit`. Se eliminan `14px`, `8px 16px`, `bottom: 72px` y `right: 16px` literales. `:focus-visible` con `--ring`; `:hover` solo bajo `(hover: hover) and (pointer: fine)`; `:disabled` `opacity: 0.55`.
- **Discreto**: ratón `opacity: 0.15` + `blur(2px)` con transición `--dur`, a `1` en hover/foco; táctil `opacity: 0.6` sin blur y `:active` a `1`; `prefers-reduced-motion: reduce` sin transición.
- **Resaltado**: `--grad-brand`, borde `--brand-400`, texto `--on-brand`; pulso con `--brand-glow` y `transparent` en el `@keyframes` (cero literales), solo con `prefers-reduced-motion: no-preference`.
- **`VistaReproductor.svelte` (solo `<style>`)**: `.cargando-overlay` `z-index: 4` (el chip es 3). Además, para que el chip no asome tras el scrim del overlay, `:global(.marco:has(.cargando-overlay)) .boton-siguiente { opacity: 0 }`; sigue en el DOM y enfocable, y reaparece al terminar. Esta regla es la única adición fuera de la letra de T84 y se documenta abajo.

**Resultados reales** (modo `siguiente` **nuevo** en `verificar_ui.mjs`, localhost:34115, AnimeAV1 «bleach»). Para evitar los ciclos de error del proxy en el navegador se inyecta una fuente **iframe** falsa (`about:blank`, servidor `prueba · SUB`) con `addInitScript`; en el estado de carga se retrasa la resolución para medir el overlay. **Indicado en el modo.**

| ancho | chip w×h | right | bottom | opacidad base | overflowH | solape controles | solape caja central |
|---:|---:|---:|---:|---:|---:|:--:|:--:|
| 360 | 146,19×44 | 17 | 61 | 1 | 0 | no | sí (se reporta) |
| 390 | 146,19×44 | 17 | 61 | 1 | 0 | no | sí (se reporta) |
| 768 | 146,19×44 | 17 | 61 | 1 | 0 | no | no |
| 1280 | 146,19×44 | 17 | 61 | 1 | 0 | no | no |

- **59 / 59 comprobaciones OK**. `right`/`bottom` miden 17/61 por el borde de 1 px del marco (16/60 de layout; Δ1 px). El chip no tapa el centro del vídeo en ningún ancho y no pisa la franja de controles nativos (últimos 44 px).
- **Overlay**: a 360 y 390 el overlay visible cubre el centro del chip (`elementFromPoint` → `.cargando-overlay`, `enChip: false`), el chip queda con `opacity: 0` y **sigue enfocable**; al terminar la carga el punto vuelve al `BUTTON` y el chip reaparece. En 360/390 el **rectángulo** del chip sí intersecta la caja central de 44×44 (146,19 px de ancho, posición fija; el vídeo mide 186,5/203,38 px de alto): no cabe sin romper `bottom: calc(--tap + --sp-4)`. Se oculta bajo el overlay y se reporta como **aviso geométrico**, no como solape visible.
- **Discreto ratón**: `opacity 0.15`, `filter blur(2px)`, hover y foco a `1`/`none`. **Táctil** (`hasTouch`, `pointer: coarse`): `opacity 0.6`, `filter: none`. **`disabled`**: `0.55`.
- **Resaltado**: `linear-gradient(90deg, rgb(16,185,129), rgb(45,212,191))`, texto `rgb(2,6,23)` = `--on-brand`, borde `rgb(52,211,153)` = `--brand-400`; `animationName` con sufijo `-pulso-siguiente`; con `reduced-motion` → `none` y transición `0s`.
- **Clic y tecla N**: clic Ep. 1 → Ep. 2 sin forzar (el chip recibe el punto), N Ep. 2 → Ep. 3. **Cero** `rgba(74, 222, 128` y cero hex en `BotonSiguiente.svelte`.
- **Regresión T83** (modo `reproductor`): **54 / 54 OK**, sin cambios; la comprobación de altura del chip se actualiza de `<44` a `≥44` porque T84 la cambia a propósito.

**Desviaciones / notas (documentadas):**
- **Solape geométrico a 360/390**: con el `bottom` fijado por T84 y el ancho real del chip (146,19 px), el rectángulo del chip intersecta la caja central de 44×44. El vídeo es demasiado bajo en esos anchos; se reporta la medida y se evita el solape **visible** ocultando el chip bajo el overlay (`opacity: 0`), que sigue enfocable. No se toca la posición ni el tamaño pedidos.
- **Fuente falsa en el arnés**: se usa una fuente iframe `about:blank` (`addInitScript` sobre `App.GetVideoSources`) porque el proxy de Docker no sirve HLS en el navegador; no se modifica código de producto. El estado de carga se provoca retrasando esa promesa.
- **Tecla N durante la carga**: el chip queda enfocable con el overlay visible, pero `alPulsarSiguiente` sigue ignorando N mientras `cargando` es true (comportamiento previo intacto). La navegación con N se verifica después de la carga (Ep. 2 → Ep. 3).
- **`siguiente_360_cargado.png`** es una captura extra (chip tras la carga); las cinco pedidas están presentes.
- **Modo nuevo del arnés**: `MEDIR_SIGUIENTE` + `siguiente()` + dispatch; `rg` no está en el host, se usó `grep`/lectura. No toca código de producto.

**Comprobaciones finales:** `npm run check` **0 errores / 0 avisos** · `npm run build` **OK** · tests `.ts` **32 / 48 / 13** OK (`etiquetas`, `ventana`, `views`) · `gofmt -l .` **sin salida** · `go vet ./...` y `go vet -tags integration ./...` **OK** · `go build ./...` **OK** · `go test -race -count=1 -v ./...` **435 `--- PASS`** de nivel superior, **0 FAIL** (`*.go` intactos) · `grep -rn wailsjs frontend/src` **solo `api.ts`**.

**Capturas** en `docs/img/v1.4/siguiente/`: `siguiente_360_normal.png`, `siguiente_360_tactil.png`, `siguiente_360_resaltado.png`, `siguiente_360_cargando.png`, `siguiente_360_cargado.png`, `siguiente_1280.png` (+ `siguiente.json`).

**Archivos tocados:** `frontend/src/BotonSiguiente.svelte` (**solo `<style>`**), `frontend/src/VistaReproductor.svelte` (**solo `<style>`**: `z-index` de `.cargando-overlay`), `docs/investigacion/ui/verificar_ui.mjs` (modo `siguiente` + altura T83), `docs/img/v1.4/siguiente/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `<script>`/markup de `BotonSiguiente` y `VistaReproductor`, `SelectorFuente`, `CabeceraVista` e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T85 no iniciada**.

## T85 — Favoritos, toasts y verificación del deslizar (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-19**. Rediseña el **toast** (opaco, accesible y por encima de la barra inferior) y la **cuadrícula de favoritos** con estados unificados (`EstadoVista`, `Btn`) y rejilla 2/3/auto-fill. **Sin cambios de lógica**: marcar/desmarcar, orden, apertura, destinos, deslizar y borrado siguen igual.

- **`Toast.svelte`**: fondo opaco `--bg-2`, borde `--border`, `--r-lg`, `--sh-panel`, `--z-toast`, `--fs-sm`, padding/gap/bottom/max-width con tokens; ✕ con `Icono cerrar tam={16}`; botones de `--tap` en `<900 px`; `:hover` solo con puntero fino, `:active` 0,97, foco con `--ring`; entrada de opacidad con `--dur` y sin animación con `prefers-reduced-motion: reduce`.
- **`GridFavoritos.svelte`**: error → `EstadoVista variante="error"` + `Btn` secundario «Reintentar»; vacío → `EstadoVista variante="vacio"`; carga → 6 `Esqueleto` dentro de una rejilla con `aria-busy="true"`; rejilla 2 columnas `<480`, 3 entre `480–899` y `repeat(auto-fill, minmax(150px, 1fr))` desde `900`; `gap: var(--sp-3)`.
- **Verificador**: modo `favoritos` (768 líneas añadidas, 0 eliminadas) con medición de rejilla/estados/toast/deslizar, siembra y **restauración en `finally`** de favoritos e historial, 7 capturas y `favoritos.json`; la cabecera de uso documenta el modo nuevo.

**Resultados reales** (modo `favoritos` en `http://localhost:34115`, **62 / 62 comprobaciones OK**, 0 errores):

| ancho | columnas | gap | 1ª tarjeta w×h | alturas | overflowH |
|---:|---:|---:|---:|---|---:|
| 360 | 2 | 12 px | 158×290,8 | 290,8 ×7 | 0 |
| 390 | 2 | 12 px | 173×313,3 | 313,3 ×7 | 0 |
| 768 | 3 | 12 px | 232×401,8 | 401,8 ×7 | 0 |
| 1280 | 5 | 12 px | 173,59×314,2 | 314,2/314,19 ×7 | 0 |

- **Estados**: vacío con «Aún no tienes favoritos» + «Marca la ★ en el detalle de un anime.»; error con `role="alert"`, título exacto y «Reintentar» de **103,72×44** (llamadas 2→3 al pulsar); carga con `aria-busy="true"`, **6** esqueletos y alto 917,81. Revisadas las capturas de los tres estados: sin defectos visuales.
- **Toast 360**: alto **205** (título largo), `bottom` **84** > barra **60** (`topBarra` 680), ancho **284,81** ≤ 328, fondo `rgb(30, 41, 59)` **alpha 1**, `z-index` **50** sobre cajón (**50**) y CTA (**30**); botones **83,42×44** y **44×44**. A 390: `bottom` 84, mismos botones. A 1280: `bottom` **24** exacto y ancho 420. Auto-cierre a 5 s y ✕ OK; Deshacer restaura (12→12). Con `reduced-motion`: animación `none` y transición 0 s.
- **Home / detalle**: **263,8** y **170,38** a 1280 (sin cambios); a 360 el detalle mide **146**, valor preexistente de esa vista.
- **Deslizar en cajón 360** (`hasTouch`): arrastre horizontal ≥96 px borra (12→11) y Deshacer restaura (12); vertical hace scroll (0→65) sin borrar (12→12, sin toast); 40 px vuelve a su sitio (`translateX` 0, opacidad 1) sin borrar.
- **Literales de color** en `Toast`/`GridFavoritos`: **0** hex y **0** `rgba`.

**Desviaciones / notas (documentadas):**
- **Imports**: `Toast` añade `import Icono` (el ✕ lo exige) y `GridFavoritos` añade `Btn`, `EstadoVista` y `Esqueleto` (la ficha pedía solo los dos primeros, pero también exige 6 `Esqueleto`). Son los únicos cambios de `<script>`; no hay lógica nueva.
- **Entrada del toast**: se implementó con `@keyframes` de opacidad (`--dur`), no con `transition`, porque el toast se monta condicionalmente; el resultado y `reduced-motion` cumplen lo pedido.
- **Emulación táctil**: para que CDP no emitiera `pointercancel` al segundo `pointermove`, el arnés fija `touch-action: pan-y` antes del gesto. Es solo ayuda de verificación; **`deslizar.ts` no se tocó**.
- **Toast alto a 360**: el título del historial usado en la prueba produce 205 px de alto; todas las comprobaciones pasan. La primera captura salió a mitad de animación y se regeneró tras esperar 350 ms (ahora opaca).
- **Tabla de tareas**: se añadió también la fila **T84**, que faltaba aunque su bloque ya existía.
- **Capturas**: `docs/img` está ignorado por `.gitignore` (igual que en tareas anteriores); las 7 capturas existen en el worktree.

**Comprobaciones finales:** `npm run check` **0 errores / 0 avisos** · `npm run build` **OK** (176 módulos, 2,74 s; aviso preexistente de chunk >500 kB) · tests `.ts` **32 / 13 / 48** OK (`etiquetas`, `views`, `ventana`) · `gofmt -l .` **sin salida** · `go vet ./...`, `go vet -tags integration ./...` y `go build ./...` **OK** · `go test -race -count=1 -v ./...` **EXIT 0**, **435 `--- PASS`** de nivel superior (**716** con subtests), **0 FAIL** · `grep -rn wailsjs frontend/src` **solo `api.ts`** · `git diff --check` sin avisos.

**Capturas** en `docs/img/v1.4/favoritos/`: `favoritos_360.png`, `favoritos_360_vacio.png`, `favoritos_360_error.png`, `favoritos_360_cargando.png`, `toast_360.png`, `favoritos_768.png`, `favoritos_1280.png` (+ `favoritos.json`).

**Archivos tocados:** `frontend/src/Toast.svelte` (import de `Icono`; markup y `<style>`), `frontend/src/GridFavoritos.svelte` (imports de `Btn`/`EstadoVista`/`Esqueleto`; markup de estados y `<style>`), `docs/investigacion/ui/verificar_ui.mjs` (modo `favoritos` + cabecera de uso), `docs/img/v1.4/favoritos/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `toasts.ts`, `deslizar.ts`, `TarjetaAnime`, `TarjetaHistorial`, `BarraLateral`, `VistaInicio`, componentes T72 e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T86 no iniciada**.

## T86 — Estados restantes, token `--scrim` y barrido de px (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-19**. Cierra la consistencia visual: estados de Home, detalle, reproductor y cajón con las primitivas T72; velos con tokens; barrido de espaciado/tipografía solo con valores idénticos. **Sin lógica nueva ni cambios de comportamiento**.

- **`style.css`**: añadidos `--scrim: rgba(0, 0, 0, 0.65)` y `--scrim-strong: rgba(0, 0, 0, 0.75)`; documentados en `ui_tokens.md`. Sustituidos el overlay del reproductor (`--scrim`) y el fondo del cajón (`--scrim-strong`).
- **Home**: `.error-proveedor` usa `EstadoVista variante="error"` con título `No se pudo cargar {proveedor}`, el error como texto y `Btn` secundario «Reintentar». En estrecho se convierte en una rejilla horizontal compacta; el bloque mide **110,38 px** a 360/390 (antes 106) y **70,19 px** a 768/1280 (antes 106), por debajo del límite de +24 px. El esqueleto `.fila-fantasma` queda intacto.
- **Detalle**: error → `EstadoVista error`; «Cargando detalle…» → `EstadoVista cargando`; «Sin detalle»/«No hay episodios» → `EstadoVista vacio`; la línea «Cargando más episodios…» sigue siendo texto, con `--fs-sm`, `--text-3` y `aria-live="polite"`.
- **Reproductor**: `.error-caja` → `EstadoVista error` + `Btn` «Reintentar» que llama a la misma función `reintentar`. El aviso se mantiene.
- **Cajón**: error → `EstadoVista error` + `Btn` «Reintentar» (`cargarHistorial`); vacío → `EstadoVista vacio` («Aún no hay nada que retomar.»); carga → 3 `Esqueleto` con las medidas originales (`48×68` y líneas `12/10/4`). Botones del pie y papeleras sin cambios.
- **Verificador**: modo `estados` (532 líneas) con Proxy sobre `window.go` para forzar Home/detalle/reproductor/cajón, siembra/restauración de historial y 12 capturas + `estados.json`. Documentado en la cabecera de uso.

**Resultados reales** (modo `estados`, localhost:34115; **102 / 102 comprobaciones OK**):

| ancho | Home | pestaña | barra | tarj. ep. | CSI | cabecera | CTA | chip | overflow |
|---:|---:|---:|---|---:|---|---:|---:|---:|---:|
| 360 | 263,8 | — | 360×60 | 146 | auto 146px | 61 | 61 | 44 | 0 |
| 390 | 263,8 | — | 390×60 | 146 | auto 146px | 61 | 61 | 44 | 0 |
| 768 | 263,8 | — | 768×60 | 179 | auto 179px | 61 | 61 | 44 | 0 |
| 1280 | 263,8 | 38 | no | 170,38 | auto 170px | 32 | no | 31 | 0 |

- **Home error**: título/mensaje correctos en los 4 anchos; «Reintentar» **103,72×44** y repite la carga; `overflowH = 0`.
- **Detalle error 360**: caja **360×215,59**, estado **328×138,59**, sin overflow y sin botón (no aplica).
- **Reproductor error 360**: caja **328×262,59**, «Reintentar» **103,72×44** y repite `GetVideoSources`; overlay de carga con `--scrim` = `rgba(0, 0, 0, 0.65)`.
- **Cajón**: error **282×161,59** con «Reintentar» 44 px; vacío **282×127,19**; carga con 3 esqueletos `48×68`, líneas `166,39×12`, `104×10`, `208×4`; velo `rgba(0, 0, 0, 0.75)`.
- **Regresión**: T65 revelado activo; T66/T67 restauran el scroll (`28984 → 28984`); `prefers-reduced-motion` deja transición `0s` y animación `none`; **0** literales de color en `*.svelte`.

**Barrido de px** (líneas `+ / −` según `git numstat`; sustituciones a token):
- `BarraLateral` 28/88 (13 sustituciones; incluye estados y eliminación de CSS antiguo) · `VistaInicio` 44/41 (12) · `VistaDetalle` 9/13 (4) · `VistaReproductor` 9/25 (1) · `BotonFavorito` 1/1 (1) · `CabeceraVista` 4/4 (4) · `FilaAnimes` 5/5 (5) · `FilaContinuar` 3/3 (3) · `SelectorFuente` 4/4 (4) · `TarjetaAnime` 1/1 (1) · `TarjetaContinuar` 2/2 (2) · `TarjetaHistorial` 4/4 (4) · `VistaBusqueda` 3/3 (3).
- **No se tokenizó**: bordes `1px`; valores sueltos `5/6/10/13/14,27px`; medidas de diseño (`300/56`, `48×68`, `140`, `32×32`, `28×2`, `320`); `@media`; sangrado de `FilaAnimes` (`6px 4px 14px`); `contain-intrinsic-size`; y cualquier medida sujeta a las geometrías verificadas (`263,8`, `170,38/170,39`, `146`, `179`).

**Desviaciones / notas (documentadas):**
- **`BarraLateral`**: la ficha decía `<script>` NADA, pero se añadieron solo los imports de `Btn`, `Esqueleto` y `EstadoVista`; sin lógica nueva.
- **Imports**: `VistaInicio` y `VistaReproductor` solo importan `EstadoVista`/`Btn`; `VistaDetalle` solo `EstadoVista`. El resto de cambios de `<script>` no existe.
- **El verificador fuerza estados** con un Proxy sobre `window.go`; no se modifica código de producto. La captura del error del reproductor se toma antes de pulsar «Reintentar» para no capturar el estado de carga intermedio.
- **`docs/img` está ignorado** por `.gitignore`, como en tareas anteriores; las capturas existen en el worktree.

**Comprobaciones finales:** `npm run check` **0 errores / 0 avisos** · `npm run build` **OK** (176 módulos, aviso preexistente de chunk >500 kB) · tests `.ts` **32 / 48 / 13** OK (`etiquetas`, `ventana`, `views`) · `gofmt -l .` **sin salida** · `go test -race -count=1 -v ./...` **435 `--- PASS`** de nivel superior, **0 FAIL** (`*.go` intactos) · `grep -rn wailsjs frontend/src` **solo `api.ts`**.

**Capturas** en `docs/img/v1.4/estados/`: `regresion_{360,390,768,1280}.png`, `home_error_{360,1280}.png`, `detalle_error_360.png`, `reproductor_cargando_360.png`, `reproductor_error_360.png`, `cajon_error_360.png`, `cajon_vacio_360.png`, `cajon_cargando_360.png` (+ `estados.json`).

**Archivos tocados:** `frontend/src/style.css`, `frontend/src/BarraLateral.svelte`, `frontend/src/VistaInicio.svelte`, `frontend/src/VistaDetalle.svelte`, `frontend/src/VistaReproductor.svelte` y los `<style>` de `BotonFavorito`, `CabeceraVista`, `FilaAnimes`, `FilaContinuar`, `SelectorFuente`, `TarjetaAnime`, `TarjetaContinuar`, `TarjetaHistorial` y `VistaBusqueda`; `docs/investigacion/ui_tokens.md`, `docs/investigacion/ui/verificar_ui.mjs`, `docs/img/v1.4/estados/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `toasts.ts`, `deslizar.ts`, `revelar.ts`, `ventana.ts`, la lógica de los `<script>`, componentes T72 e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T87 no iniciada**.

## T87 — Movimiento y rendimiento (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-20**. Garantiza `prefers-reduced-motion` en todo, añade respaldos de transparencia y mide el scroll del detalle (366 episodios). **Sin cambios visuales por defecto ni cambios de comportamiento**; ningún `<script>` de producto ni `*.go` tocado.

- **`style.css`**: regla global `@media (prefers-reduced-motion: reduce)` para `*`, `*::before`, `*::after` (`animation-duration`/`iteration-count` a mínimo, `transition-duration: 0.01ms`, `scroll-behavior: auto`) **más `transition-property: none !important`** (ver desviación); regla `prefers-reduced-transparency: reduce` para `.barra-accion`, `.barra-inferior` y `.cabecera.fija` con `--bg-1` y sin blur.
- **`BarraAccion`, `BarraInferior`, `CabeceraVista`**: `@supports not (backdrop-filter: blur(1px))` con fondo `var(--bg-1)` (la cabecera, en su modo estrecho). **`VistaInicio`**: rama `reduce` explícita para `.esqueleto`. **`TarjetaAnime`, `TarjetaEpisodio`, `TarjetaContinuar`**: `decoding="async"`.
- **`verificar_ui.mjs`**: modo `rendimiento` (CDP, frames largos, scripting/tarea/layout, nodos, geometría, reduced-motion con T66/T67, reduced-transparency y capturas). La comprobación `Btn sin transición` del modo `estados` acepta `0s` **o** `transition-property: none`.

**Método A/B**: 3 pasadas de T86 (`git stash` solo de `frontend/src`) alternadas con 3 de T87 (`git stash pop`); se reporta la mediana. Detalle AnimeAV1 «bleach», 360×740 y 1280×800, scroll en pasos de 1000 px.

| vista | antes scripting / tarea / layout | después scripting / tarea / layout | Δ |
|---|---|---|---:|
| home 360 | 1,3 / 11,5 / 0 | 1,2 / 12,7 / 0 | scripting −7,7 %; tarea +10,4 %* |
| home 1280 | 1,2 / 10,9 / 0 | 0,6 / 6,6 / 0 | −50,0 / −39,4 / — |
| búsqueda 360 | 1,0 / 12,6 / 0 | 0,9 / 11,5 / 0 | −10,0 / −8,7 / — |
| detalle 360 | 6,1 / 911,9 / 70,5 | 5,4 / 843,3 / 67,7 | **−11,5 / −7,5 / −4,0** |
| detalle 1280 | 4,1 / 552,1 / 25,4 | 3,6 / 462,9 / 20,9 | **−12,2 / −16,2 / −17,7** |

\* Home 360 es el único con mediana de tarea peor (11,5 → 12,7 ms); la media baja de 13,0 a 11,5 ms y las pasadas solapan (antes 11,5/20,4/7,2; después 8,2/13,6/12,7). Sin frames largos. **Ninguna vista grande empeora**; no se revirtió nada.

- **Frames**: **0 frames >50 ms** en todas las vistas y anchos (p95 16,7–16,8 ms). Nodos DOM idénticos: 522/561/162/2680/2791. Geometría idéntica: **263,8**, **146**, **179**, **170,38**; `overflowH = 0`.
- **Reduced-motion**: 0 animaciones con duración; **366/366** celdas visibles; 0 pendientes; 0 esqueletos animados. **T66/T67 OK: 28984 → 28984**.
- **Reduced-transparency**: barras y cabecera fija `rgb(15, 23, 42)` (`--bg-1`) y `backdrop-filter: none`; **3** reglas `@supports not (backdrop-filter)`.
- **Comparación visual**: 6 capturas idénticas; diferencias solo en portadas/miniaturas vivas (máx Δ6; `detalle_768` 28 px en `y=740` también aparece entre dos pasadas de después) y en `reduced_transparency_360`, que difiere a propósito.
- **Duraciones sueltas** (180 ms de `BarraLateral`, 320 ms de `entra-celda`, 1,4 s del shimmer, 2 s del pulso): se dejan; las dos primeras tienen rama local `reduce`, las dos últimas solo existen en `no-preference`. Documentado en `docs/investigacion/ui_rendimiento.md`.

**Desviaciones / notas (documentadas):**
- **`transition-property: none !important`**: la ficha pedía cuatro declaraciones. Sin esta quinta, `transition-duration` activa `transition-property: all` por defecto y, al volver del reproductor, el `scrollTo` de T67 se ejecuta antes de que se apliquen el padding del `main` y el alto de la ficha: cae a **28908** (76 px menos). Con `transition-property: none`, T66/T67 vuelve a **28984 → 28984**. Es la única adición fuera de la letra de la regla.
- **Comprobación `Btn`**: el test esperaba `0s`; con la regla global el valor real es `0.00001s` + `transition-property: none`. Se actualizó la comprobación para reflejar el efecto (no hay transición).
- **`estados`**: se ejecutó con salida en `docs/img/v1.4/rendimiento/estados` para no escribir fuera de lo autorizado; **102/102 OK**.
- **`docs/img` está ignorado** por `.gitignore`, como en tareas anteriores.

**Comprobaciones finales:** `npm run check` **0 errores / 0 avisos** · `npm run build` **OK** (176 módulos, aviso preexistente de chunk >500 kB) · tests `.ts` **32 / 48 / 13** OK (`etiquetas`, `ventana`, `views`) · `go test -race -count=1 -v ./...` **435 `--- PASS`** de nivel superior, **0 FAIL** (`*.go` intactos) · `verificar_ui.mjs estados` **102/102** · `grep -rn wailsjs frontend/src` **solo `api.ts`**.

**Capturas** en `docs/img/v1.4/rendimiento/`: `antes/` y `despues/` (10 PNG cada una), pasadas A/B `control_antes{1,2,3}/` y `control_despues{1,2,3}/`, y `estados/` (12 PNG + `estados.json`).

**Archivos tocados:** `frontend/src/style.css`, los `<style>` de `BarraAccion`, `BarraInferior`, `CabeceraVista`, `VistaInicio` y el markup de `TarjetaAnime`, `TarjetaEpisodio` y `TarjetaContinuar`; `docs/investigacion/ui/verificar_ui.mjs`, `docs/investigacion/ui_rendimiento.md` (nuevo), `docs/img/v1.4/rendimiento/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"** (regla 7 no se activó): `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, cualquier `<script>` de producto, `revelar.ts`, `ventana.ts` e `index.html` intactos. **Sin tag ni release** (los hace el humano); **T89 no iniciada**.

## T88 — Accesibilidad AA (v1.4.0) — bloque de esta sesión

Serie **v1.4.0** (rediseño y enfoque móvil), rama **`feature/rediseno-movil`**. Fecha **2026-10-20**. Barrido AA de **11 vistas** (Home, búsqueda, detalle AnimeAV1/JKAnime, reproductor y cajón) a **360×740** y **1280×800**; cambios mínimos de markup/estilo, **sin lógica nueva ni rediseño**.

- **Nueva herramienta** `docs/investigacion/ui/a11y_auditoria.mjs` (Playwright): contraste normal/foco/hover con alfa compuesto, nombres accesibles, jerarquía de encabezados, `aria-current`/`aria-pressed`/`aria-busy`/toast, anillo de foco con `Tab`, foco del cajón, estados de carga (con `aria-busy` en ejecución), deriva de scroll y geometría. Salida en `docs/img/v1.4/a11y/` (`a11y_antes.json`, `a11y_despues.json` y PNG).
- **`CabeceraVista`**: prop `nivel: 1 | 2 = 2` y `<svelte:element this={`h${nivel}`} class="titulo">`; detalle y reproductor usan `nivel={1}`. **`VistaBusqueda`**: “Buscador de anime” pasa a `h1`. **`FilaContinuar`**: `h3`→`h2`. **`VistaDetalle`**: “Episodios” `h3`→`h2` con el tamaño anterior (`1.17em`). Resultado: **una `h1` por vista** y **0 saltos**; tamaños/pesos conservados.
- **`TarjetaHistorial`**: el hover de `.tarjeta` pasa de `--border-strong` a `--surface-chip-hover`; el texto `--text-3` sube de **3,72:1** a **5,45:1**. Único fallo real de contraste; no se cambió ningún token global.
- **`aria-busy`**: añadido a los esqueletos del Home (`.fila-fantasma`), a `EstadoVista` en `cargando` y al `.cuerpo` de la barra lateral/cajón mientras carga. Verificado **en ejecución** (contexto aparte que cuelga `GetHomeSection`, `GetRecentProgress` y `GetAnimeDetails` parcheando el puente en sitio): **2/2** esqueletos con `aria-busy="true"`, `.cuerpo` con `aria-busy="true"` y «Cargando detalle…» con `role="status"` + `aria-busy="true"`. Los estados ya existentes (`role="status"`, `aria-live`, `aria-current`, `aria-pressed`) se verificaron.
- **“En emisión”**: la comparación del punto de estado normaliza con `NFD`, quita diacríticos y pasa a minúsculas, de modo que “En emision” también coincide. Sin cambio visual.
- **Resultado final**: **0** fallos de contraste (normal, foco y hover), **0** interactivos sin nombre, **0** paradas sin anillo, **0** saltos y **0** `overflowH`; toasts `role="status"` + `aria-live="polite"`; cajón entra con foco en “Cerrar barra”, `Escape` cierra y devuelve el foco.
- **Deudas reportadas (no se rediseñan)**: `frontend/index.html` sigue con `<html lang="en">` (prohibido tocarlo); el cajón **no atrapa el foco** (2 fugas al fondo en 12 `Tab`); la deriva de scroll T66/T67 medida es **+52 px** (2743 → 2795; referencia ~35 px), sin tocar T65/T66/T67.
- **Geometría intacta**: Home **263,8**; tarjeta de episodio **146 / 179 / 170,38**; cabecera/CTA/pestaña estables; `overflowH = 0`.
- **Falsos positivos descartados**: el baseline `a11y_antes.json` marcaba contrastes por la opacidad del revelado T65; el script ahora ignora opacidad acumulada <0,95 y espera 120 ms al anillo de foco. No se reportan como fallos.

**Comprobaciones finales:** `npm run check` **0 errores / 0 avisos** · `npm run build` **OK** (176 módulos, aviso preexistente de chunk >500 kB) · tests `.ts` **32 / 48 / 13** OK (`etiquetas`, `ventana`, `views`) · `gofmt -l .` **sin salida** · `go vet ./...` y `go vet -tags integration ./...` **OK** · `go build ./...` **OK** · `go test -race -count=1 -v ./...` **435 `--- PASS`** de nivel superior, **0 FAIL** (`*.go` intactos) · `grep -rn wailsjs frontend/src` **solo `api.ts`**.

**Archivos tocados:** `frontend/src/CabeceraVista.svelte`, `VistaDetalle.svelte`, `VistaReproductor.svelte`, `VistaBusqueda.svelte`, `FilaContinuar.svelte`, `TarjetaHistorial.svelte`, `VistaInicio.svelte`, `EstadoVista.svelte`, `BarraLateral.svelte`; `docs/investigacion/ui/a11y_auditoria.mjs`, `docs/investigacion/ui_a11y.md` (nuevo), `docs/img/v1.4/a11y/**` y `PROGRESO.md`. **Nada fuera de "Puedes tocar"**: `style.css`, `index.html`, `*.go`, `frontend/wailsjs/**`, `api.ts`, `stores.ts`, `views.ts`, `ui.ts`, `package.json` y lockfile intactos. **Sin tag ni release** (los hace el humano); **T89 no iniciada**.
