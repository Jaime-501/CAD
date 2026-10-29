# PROGRESO v1.5.0 — pantalla completa del dispositivo (T91–T94)

> Historial de la serie `feature/pantalla-completa`. Aquí quedan las notas largas de
> T91–T93b (texto literal movido desde `PROGRESO.md` en T94) y el cierre de la
> versión. `PROGRESO.md` conserva una línea por tarea y los pendientes vigentes.
>
> **Estado**: v1.5.0 **preparada** (`productVersion` 1.5.0), **sin tag ni release**;
> el tag y la release los hace el humano tras la prueba manual con el `.exe` release.
>
> Go **intacto** en todas las tareas (16 métodos Wails, sin cambios): la pantalla
> completa usa el runtime de Wails desde `api.ts` (`WindowFullscreen`,
> `WindowUnfullscreen`, `WindowIsFullscreen`).

## Notas de ejecución (texto literal de PROGRESO.md)

- **Pantalla completa (T91)**: `frontend/src/pantallacompleta.ts` es puro y **sin dependencias** (`elementoEnPantallaCompleta` + `crearSincronizador` con cola de promesas, estado `propia`/`liberado`, consulta tratada como `false` si rechaza y `entrar`/`salir` a prueba de excepciones); `api.ts` añade `ventanaPantallaCompleta(activar)` (nunca lanza) y `esVentanaPantallaCompleta()` (nunca rechaza; `false` ante fallo), importando `WindowFullscreen`/`WindowUnfullscreen`/`WindowIsFullscreen` en la línea de `runtime.js`. Sin consumidores todavía (50 comprobaciones en `pantallacompleta.test.ts`).
- **Pantalla completa (T92)**: `VistaReproductor.svelte` cablea el módulo **solo en el `<script>`** (markup y `<style>` intactos): una instancia `sincronizador = crearSincronizador({ esPantallaCompleta: esVentanaPantallaCompleta, entrar/salir: ventanaPantallaCompleta })`, listener `fullscreenchange` (solo el evento sin prefijo) en `onMount` y `alCambiarPantallaCompleta()` → `alCambiarDocumento(elementoEnPantallaCompleta(document) !== null)`. En `onDestroy`, tras el orden existente: quita el listener, si `document.fullscreenElement` existe llama a `exitFullscreen().catch(() => {})` y por último `sincronizador.liberar()`. Nada nuevo en `reiniciarEpisodio`/`sincronizarEpisodio`: el `<video>` no se remonta y el fullscreen de `.marco` sobrevive al cambio de episodio; el `<iframe>` **sí** se desmonta (`modo='nada'` + `fuentes=[]`), pero el navegador cierra por su cuenta el fullscreen del elemento retirado y el listener sincroniza la ventana. Verificación: modo nuevo `pantallacompleta` en `verificar_ui.mjs` (30/30, casos A–E) y `reproductor` 54/54 + `siguiente` 59/59 sin cambios; Go intacto (435 `--- PASS`). Notas: el modo necesita un clic de gesto fuera del `.marco` (iframe/vídeo cubren el div y el clic en el iframe no llega al documento padre), y filtra el `pageerror` «Cannot read properties of null (reading 'nodes')», ruido **preexistente** del dev server (aparece ya en el Home). `frontend/package.json.md5` lo regeneran `wails dev`/`wails build` (estaba obsoleto); se restauró para no salir de «Puedes tocar».
- **Pantalla completa (T93)**: `VistaReproductor.svelte` pone en fullscreen el `.marco` (vídeo/iframe + chip «Siguiente ▸» + overlay + botón propio) en lugar del `<video>` suelto: `let marcoEl` con `bind:this` (más asignación robusta en `onMount` por `document.querySelector('.marco')`, porque el `bind:this` no asignaba en el dev server por HMR) y `enPantallaCompleta` actualizado en `alCambiarPantallaCompleta` **antes** de llamar al sincronizador (`document.fullscreenElement === marcoEl`); `alternarPantallaCompleta()` alterna con `requestFullscreen()`/`exitFullscreen()` (ambas con `.catch` → `console.warn`). El `<video>` lleva `controlsList="nofullscreen"` (sin botón nativo de pantalla completa). Markup: `<button class="pantalla-completa">` dentro de `.marco`, visible solo si `modo !== 'nada'`, oculto bajo el overlay de carga (z-index 4), arriba a la derecha, 44×44, `aria-pressed`/`aria-label` alternan («Pantalla completa» / «Salir de pantalla completa»), iconos `pantalla-completa`/`salir-pantalla-completa` añadidos a `Icono.svelte`. Estilos con tokens (opacidad 0,6 en puntero fino que se aclara en hover/foco; en táctil siempre 1; `:active` escala .97; reduced-motion sin transición) y `.marco:fullscreen` (sin radio/borde/sombra, `aspect-ratio: auto`, 100×100 %, fondo `--bg-0`) con `object-fit: contain` en el vídeo. `verificar_ui.mjs` amplía el modo `pantallacompleta` con los casos F–K. Resultado: modo `pantallacompleta` **63/63 OK** (A–K); modo `siguiente` sin regresiones; Go intacto (gofmt/vet/build/test -race en verde) y `git diff --stat -- '*.go'` vacío. `frontend/package.json.md5` restaurado tras el build. El modo `reproductor` da **50/52** en este entorno (2 fallos de «no se llegó al iframe» en el caso D) tanto con mis cambios como revertidos: el proxy local del contenedor sirve el HLS correctamente, por lo que el reproductor no cae al iframe; es un fallo preexistente del test (depende de que el HLS falle en el navegador) y no está causado por T93. Prueba manual pendiente con el `.exe` (doble clic en el vídeo no debe entrar en fullscreen nativo; verificar con el `.exe`).
- **Pantalla completa (T93b)**: `VistaReproductor.svelte` oculta el botón nativo del `<video>` **solo en el `<style>`**: `:global(.marco .video::-webkit-media-controls-fullscreen-button) { display: none; }` (con comentario; el pseudo-elemento vive en el shadow DOM del UA, de ahí `:global`), y `controlsList="nofullscreen"` se mantiene como respaldo. Svelte compila el selector sin clase de alcance (`dist/assets/index-*.css` → `.marco .video::-webkit-media-controls-fullscreen-button{display:none}`). `verificar_ui.mjs` añade el caso **L** al modo `pantallacompleta` (2 comprobaciones: `<video>` con `nofullscreen` y la regla `display: none` presente en las hojas del navegador) → **66/66 OK** (A–L). **Desviación**: el efecto real no es medible en este entorno — `getComputedStyle(video, '::-webkit-media-controls-fullscreen-button')` devuelve `inline` incluso para un pseudo inexistente y `CSS.getComputedStyleForNode` no expone `pseudoElements`; queda para la prueba manual del `.exe`. Modo `siguiente` **59/59** sin cambios; modo `estados` **98/102** con los mismos 4 fallos de `contain-intrinsic-size` **con el cambio y con la regla revertida** (preexistentes del entorno, no de T93b). `npm run check` 0/0, `npm run build` OK, tests .ts **143** (32+50+48+13); Go intacto (gofmt sin salida, vet/build/test -race en verde) y `git diff --stat -- '*.go'` vacío. `.exe` recompilado con `-debug`; `frontend/package.json.md5` restaurado.

## Diagnóstico y cierre (T94)

**Diagnóstico (solo lectura).** Se compararon los modos `estados` y `reproductor` del código
actual con el de **T89b** (`876d14c`), con el mismo script y el mismo dev server, en **dos
pasadas por estado** (actual → T89b → actual → T89b). Resultado: **idénticos** en los dos
estados (`estados` 98/102 y `reproductor` 54/54 en las cuatro pasadas) → **no hay regresión
de código**, solo entorno/datos:

- **`estados` (98/102)**: los 4 fallos de `contain-intrinsic-size` necesitan una tarjeta de
  episodio con `.franja` (progreso parcial) o `.visto`; el contenedor no tiene
  `historial.json`, así que las 366 tarjetas de «bleach» son «ninguno» y no hay muestra.
  Medido aparte, el valor declarado es el esperado: `auto 146px` (360/390), `auto 179px`
  (768) y `auto 170px` (1280). **Confirmado después**: al sembrar historial en el contenedor
  (`a11y_auditoria.mjs` lo hace para la fila «Continuar»), el mismo código da **102/102**.
- **`reproductor` (54/54)**: el 50/52 de T93/T93b no se reproduce aquí; el proxy de loopback
  del contenedor da `ERR_CONNECTION_REFUSED` desde el navegador del host, el reproductor cae
  al iframe y el caso D se ejercita (2 comprobaciones más). Igual con el código de T89b.

Informe completo: `docs/investigacion/ui_regresiones_v1.5.md`.

**Verificación final (28/10/2026, dev server + contenedor).**

| modo (`verificar_ui.mjs`) | resultado |
|---|---|
| `capturar` | utilidad: 12 capturas + medidas; sin comprobaciones |
| `comparar` | utilidad (diff de dos directorios); requiere dos capturas |
| `pixel` | utilidad (lee un píxel de un PNG) |
| `tactiles` | **1 fallo**: `waitForSelector('.btn')`, idéntico con T89b → expectativa obsoleta |
| `shell` | 23/23 (con historial sembrado) |
| `cabecera` | 39/39 |
| `buscador` | 61/61 (con favoritos sembrados) |
| `filas` | 63/63 |
| `continuar` | 43/43 (total de siempre; el 46 de la tabla de T89 es una errata) |
| `busqueda` | 60/60 |
| `ficha` | 106/106 |
| `cta` | 56/56 (total de siempre; el 59 de la tabla de T89 es una errata) |
| `episodios` | 46/46 |
| `orden` | 89/89 |
| `reproductor` | 54/54 |
| `siguiente` | 59/59 |
| `pantallacompleta` | 66/66 |
| `estados` | 102/102 (con historial sembrado; 98/102 sin él) |
| `favoritos` | 62/62 (una primera pasada dio 19/19 con el contenedor cayéndose: 6 errores de `goto`/puente; repetida con el servidor sano, 62/62) |
| `rendimiento` | sin incumplimientos (0 frames > 50 ms; tarjeta de episodio 146/179/170,38 px; reduce OK) |
| `a11y_auditoria.mjs` | 11 vistas, 0 fallos de contraste/nombre/encabezado/foco, 1 `h1` por vista |

Compuerta: `gofmt -l .` sin salida, `go vet ./...` y `go vet -tags integration ./...` OK,
`go build ./...` OK, `go test -race -count=1 ./...` **435 `--- PASS`**, `npm run check`
**0 errores y 0 avisos**, tests `.ts` **143** (32+48+13+50), `grep -rn wailsjs frontend/src`
solo `api.ts`, `git diff` de `*.go`, `frontend/wailsjs` y `frontend/src` vacío, y
`wails build -platform windows/amd64` release → `build/bin/miapp.exe` (13,5 MB); el
`frontend/package.json.md5` regenerado se restauró con `git checkout`.

Los totales **`cta` 56** y **`continuar` 43** son los de siempre (BITACORA T80 y T77) y se
reprodujeron también con el código de T89b; la tabla de T89 de `docs/PROGRESO_v1.4.md` los
anota como 59 y 46 por una **errata** (+3), no por comprobaciones perdidas. `favoritos` da
**62/62** en una pasada válida (la de 19/19 fue una corrida abortada por la caída del
contenedor). Detalle y aritmética: `docs/investigacion/ui_regresiones_v1.5.md`.

**Cierre.** `productVersion` **1.5.0** en `wails.json` (único cambio del archivo); entrada
1.5.0 en `CHANGELOG.md` (con el typo «aTokens» de la 1.4.0 y la viñeta vacía de la 1.2.0
corregidos); `AGENTS.md`, `docs/DESARROLLO.md`, `tareas/00_REGLAS.md` y `README.md` al día;
filas T91–T94 en `docs/BITACORA.md`; notas largas de T91–T93b movidas aquí; fichas cerradas
en `tareas/archivo/`. **Sin tag ni release**: los hace el humano tras la prueba manual con
el `.exe` release.
