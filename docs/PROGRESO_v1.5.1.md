# PROGRESO v1.5.1 — higiene de la verificación y deriva de scroll (T95–T98)

> Historial de la serie `chore/higiene-v1.5`. Aquí quedan las notas largas de **T95–T97b**
> (texto literal movido desde `PROGRESO.md` en T98) y el cierre de la versión. `PROGRESO.md`
> conserva **una línea por tarea** y los pendientes vigentes.
>
> **Estado**: v1.5.1 **preparada** (`productVersion` 1.5.1), **sin tag ni release**; el tag y
> la release los hace el humano tras la prueba manual con el `.exe` release.
>
> **Producto**: el único cambio de comportamiento de la serie es el **scroll del detalle en
> móvil** (T97b). T95 y T96 fueron herramienta y diagnóstico; T97, solo diagnóstico. Go
> **intacto** en toda la serie (16 métodos Wails, sin cambios en `*.go` ni en
> `frontend/wailsjs/**`).

## Notas de ejecución (texto literal de PROGRESO.md)

- **`tactiles` (T95)**: el modo abortaba siempre porque esperaba `page.waitForSelector('.btn')` en el Home y el Home sano **no renderiza `.btn`** (la actualización es `button.actualizar`; `.btn` solo aparece en los estados de error con «Reintentar»). Se reescribió la rama `tactiles` de `verificar_ui.mjs` para: esperar la carga del Home con `.fila .carrusel .tarjeta` (nunca `.btn`), medir `MEDIR_TACTILES` en el Home sano, **forzar el error** de una fila reemplazando `window.go.main.App.GetHomeSection` por un rechazo (restaurándolo siempre en `try/finally`), pulsar `button.actualizar`, esperar el `.btn` «Reintentar» y medir de nuevo. **Anti falso verde**: imprime los conteos de `.btn` y `.chip` por ancho, falla si el total medido de un ancho es 0 y falla si el forzado funcionó pero no se midió ningún `.btn`. Resultado: 3 anchos (360/768/1280), **btn=2 por ancho** (Reintentar 103,72×44, OK), **chip=0** (el Home no renderiza `.chip`; las pestañas usan `pestana`), **salida 0**, sin errores de navegador; la **prueba negativa** (nada que medir, en una copia temporal fuera del repo) da **salida 1**. Comparación con T89b: mismo resultado de medición. **Desviación:** la mención obsoleta de `AGENTS.md` no se editó en T95 (estaba fuera de «Puedes tocar»); se corrige en T98.
- **`pageerror` del dev server (T96)**: `Cannot read properties of null (reading 'nodes')` es **ruido del dev server**. La pila apunta **solo** a `/wails/ipc.js` (inyector de Wails que existe únicamente en `wails dev`), `nodes` no aparece en `frontend/src` (0 coincidencias) y el bundle de producción servido estáticamente da **0 `pageerror` en 3/3** repeticiones (los avisos `reading 'main'` son otra cosa: `console.warn` de la app sin el puente `window.go`). Reproducción en el dev server: **3/3 determinista**, 1 error por carga, a los 187/123/128 ms (justo tras `DOMContentLoaded`). **No se abre T96b.** Dos cosas **no medidas, solo inferidas**: la comparación con T89b (no se ejecutó; el origen está fuera de `frontend/src`) y que `wails build` no inyecte `/wails/ipc.js` en el `.exe` (se confirma con `-debug` + F12 en la prueba manual). Informe: `docs/investigacion/pageerror_nodes.md`.
- **Deriva de scroll (T97/T97b)**: causa identificada y corregida. Los **+52 px** (2743 → 2795) no eran de la rejilla ni de `content-visibility`: eran el hueco del botón **«Ver más»** de la sinopsis (`.ver-mas`: 44 px de `min-height` + 8 px de `margin-bottom`), que **no existía en el frame de montaje** porque `sinopsisDesborda` se decidía en un `requestAnimationFrame` **posterior** al `scrollTo` de `App.svelte`; el *scroll anchoring* de Chromium convertía ese crecimiento tardío en +52 (`scrollHeight` 29 672 → 29 724, `.rejilla` en el viewport −2073 → −2125). Determinista 3/3 (8 ejecuciones), independiente del anime (Bleach y Naruto) y ausente a 1280 (sin botón). Las cifras 2743/2795 salen de `medirDerivaScroll` de `a11y_auditoria.mjs:536-554`, **no** del modo `shell` B7b, que mide al fondo y da Δ 0 porque el recorte al máximo y la compensación se anulan. Idéntica con T89b (2/2) → previa a T87; entra con el botón de **T79**. **Fix (T97b)**: medición **síncrona** de la sinopsis al engancharse `sinopsisEl` (`VistaDetalle.svelte`), con el `rAF` conservado **solo** para `resize`; sin `overflow-anchor: none` y sin reservar 52 px. **Medido tras el fix**: 360×740 Bleach `2743 → 2743` (Δ0, 3/3), Naruto `2743 → 2743` (Δ0, 3/3+1), `.rejilla` en el viewport −2073 antes y después; 1280×800 `1692 → 1692` (Δ0, sin botón, `.datos` 43,89 sin cambios). Informe: `docs/investigacion/deriva_scroll_detalle.md`.

## Cifras contrastadas contra los scripts (T98)

- **`shell`**: el script tiene **23 `check(`** en la función `shell()`, pero **3 ramas son alternativas** (B6: 2 de 3; B7b: 1 de 2; B8: 1 de 3), así que **ejecuta 20 comprobaciones**. Con historial sembrado pasa **20/20** (B8 toma la rama del toast); sin historial, **19/20** (B8 informa «sin historial sembrado»). La tabla de T94 (`PROGRESO_v1.5.md`) anota `shell` **23/23**: es el número de **sitios de llamada**, no de comprobaciones ejecutadas → **errata de recuento**, no de resultado.
- **`tactiles`**: la rama nueva mide `.btn` (2 por ancho, forzando el error) y `.chip` (0, porque el Home no los renderiza). Cobertura real de `.chip` **pendiente** (ver `PROGRESO.md`).
- **`cta` 56 y `continuar` 43**: totales reales del script (reproducidos en T94). La tabla de T89 en `docs/PROGRESO_v1.4.md` sigue anotando **59** y **46**; el archivo histórico **no se toca** y la errata queda anotada en `PROGRESO.md` y en `docs/investigacion/ui_regresiones_v1.5.md`.

## Cierre (T98)

- `wails.json` → `info.productVersion` **`1.5.1`** (único cambio del archivo); entrada **1.5.1** en `CHANGELOG.md` con un solo apartado **Corregido** (scroll del detalle en móvil). Nada más como cambio de producto.
- `AGENTS.md` y `docs/DESARROLLO.md`: fuera la mención de que `tactiles` aborta por `.btn`; estado **v1.5.1 preparada**; `docs/PROGRESO_v1.5.1.md` añadido a los historiales.
- `tareas/00_REGLAS.md`: rango de fichas archivadas hasta **T98** y nuevo historial en la lista de `docs/`.
- `frontend/package.json.md5`: **decisión del humano = ignorarlo** → línea en `.gitignore` y `git rm --cached` (el archivo sigue en disco; `wails dev`/`wails build` lo regeneran y ya no ensucia `git status`).
- Fichas **T95, T96, T97, T97b y T98** movidas a `tareas/archivo/`. Regla de higiene T90 aplicada.
- **Sin tag ni release**: los hace el humano.
