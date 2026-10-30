# Deriva de scroll +52 px del detalle (T66/T67) — diagnóstico (T97)

Fecha: 2026-10-29 · Rama: `chore/higiene-v1.5` · Código medido: HEAD `cd2b508`
(v1.5.0 preparada, sin cambios de producto: **T97 es solo diagnóstico**).

**Veredicto: causa identificada en código.** Los 52 px **no** son de la rejilla de episodios ni
de la restauración: son el hueco del botón **«Ver más» de la sinopsis** (`.ver-mas`:
`min-height: var(--tap)` **44 px** + `margin-bottom: var(--sp-2)` **8 px** = **52 px**), que
**no existe en el frame de montaje** del detalle (se decide en un `requestAnimationFrame`) y
aparece **un frame después**. La restauración de `App.svelte` corre en ese mismo frame, con el
documento 52 px más corto (29 672 vs 29 724) y en una posición que ya no es la misma; cuando el
botón aparece, el *scroll anchoring* de Chromium conserva la posición visual y sube `scrollY`
+52 px (2 743 → 2 795). Determinista **3/3** (8 ejecuciones en total). No es la rejilla, no es
`content-visibility`, no es la carga de imágenes ni la barra de acción. **No se corrige nada**
(propuesta mínima en el último apartado; justifica **T97b**).

---

## 1. Escenario exacto (de dónde salen 2743 y 2795)

Hay que separar dos mediciones distintas, porque la ficha las atribuye al modo `shell` y no lo son:

| Origen | Dónde | Pasos | Resultado real medido |
|---|---|---|---|
| `shell` → **B7b** «restauración de scroll T66/T67» | `docs/investigacion/ui/verificar_ui.mjs:839-856` | 360×740 · `buscar('bleach')` · `abrirDetalle('animeav1')` · 900 ms · `scrollTo(0, scrollHeight)` · 500 ms · clic en **la última** tarjeta · 1800 ms · «← Atrás» · 1200 ms | **28 984 → 28 984 (Δ 0)** |
| `a11y_auditoria.mjs` → `medirDerivaScroll` (invocado a 360 en la línea 774) | `docs/investigacion/ui/a11y_auditoria.mjs:536-554` | 360×740 · mismo `buscar`/`abrirDetalle` · `nth(min(30, total-1)).scrollIntoViewIfNeeded()` · 500 ms · `antes` · clic · 1500 ms · «← Atrás» · 2500 ms · `despues` | **2 743 → 2 795 (Δ +52)** |

Las cifras **2743 → 2795** de T89 y de `docs/investigacion/ui_a11y.md:140-142` salen de
`medirDerivaScroll` de `a11y_auditoria.mjs` (tarjeta **intermedia**, índice 30), **no** de B7b:
B7b mide al fondo y da Δ 0. Ambos escenarios se han reproducido; el `+52` es el mismo fenómeno y
en B7b queda oculto por el recorte al máximo (ver §4).

Contexto del escenario (datos reales, sin sembrar nada): anime **Bleach** de **AnimeAV1**,
**366** episodios, rejilla de **2** columnas a 360 px, tarjeta de episodio **146 px**,
`scrollHeight` **29 724**; en el frame de montaje del detalle son **29 672** (52 menos).
`frontend/src` se ha restaurado a HEAD y el diff de producto ha quedado vacío
(`git diff --stat -- '*.go' frontend/src frontend/wailsjs` sin líneas).

## 2. Reproducción (3 repeticiones)

Script temporal `deriva.mjs` (fuera del repo) que copia literalmente `medirDerivaScroll`:

| # | Celdas | `antes` | `despues` | Δ | `scrollHeight` antes/después | `.ficha` antes/después |
|---:|---:|---:|---:|---:|---|---|
| 1 | 366 | 2743 | 2795 | **+52** | 29 724 / 29 724 | 517,22 / 517,22 |
| 2 | 366 | 2743 | 2795 | **+52** | 29 724 / 29 724 | 517,22 / 517,22 |
| 3 | 366 | 2743 | 2795 | **+52** | 29 724 / 29 724 | 517,22 / 517,22 |

- **Determinista**: 3/3 idéntico, y 8/8 contando las demás ejecuciones de la sesión
  (2 con T89b, 1 de la variación base, 1 del trazado de mecanismo y 1 final de comprobación).
- **El `scrollHeight` final no cambia** (29 724 antes y después): no se «asienta la rejilla»,
  no hay filas nuevas ni `contain-intrinsic-size` de por medio. Lo que cambia es `scrollY`.
- **No depende del anime ni del número de episodios**: Bleach (366 eps) y Naruto dan +52;
  el factor es que la sinopsis desborde el recorte a 360 px (ver §5). Los 4 títulos sondeados
  (`ponyo`, `your name`, `suzume`, `bleach`) tienen `.ver-mas` y sinopsis recortada a **96 px**.

## 3. Instrumentación frame a frame (sin editar código)

Muestreo con `requestAnimationFrame` desde un `addInitScript` (solo navegador), más un envoltorio
de `window.scrollTo` para registrar cada llamada con su pila. Transición real (360×740, Bleach,
tarjeta 30):

| t | vista | `scrollY` | `scrollHeight` | alto `.datos` | `.ver-mas` | nº celdas |
|---:|---|---:|---:|---:|---|---:|
| 1572 ms | reproductor | 0 | 740 | — | no | 0 |
| **1607 ms** | **detalle (montaje)** | **2743** | **29 672** | **215,78** | **no** | 366 |
| **1612 ms** | **detalle (asentado)** | **2795** | **29 724** | **267,78** | **sí (44 px)** | 366 |
| 1622 ms | detalle | 2795 | 29 724 | 267,78 | sí | 366 |
| … | detalle | 2795 | 29 724 | 267,78 | sí | 366 |

`scrollTo` registrado en la vuelta: **`pedido 2743` con `scrollHeight` 29 672 y `max` 28 932**
(no se recorta), emitido por `sincronizarVista` (`frontend/src/App.svelte:67-73`, la línea 72).

Diff de bloques entre el frame de montaje y el asentado (coordenadas de documento, `rect.top +
scrollY`):

| bloque | montaje | asentado | Δ |
|---|---|---|---|
| `.ficha` | alto 465,22 | alto 517,22 | **+52** |
| `.datos` (hijo de `.ficha`) | alto 215,78 | alto 267,78 | **+52** |
| `.rejilla` | docTop 618 | docTop 670 | +52 |
| `.rejilla` alto | 28 902 | 28 902 | 0 |
| `.celda` 0 y 30 | alto 146 | alto 146 | 0 |
| `.cabecera` | alto 61 (sticky) | alto 61 | 0 |
| `.barra-accion` | 61 (fixed) | 61 | 0 |
| `scrollHeight` | 29 672 | 29 724 | **+52** |

Es decir: **lo único que crece son los 52 px de `.datos`**; la rejilla no cambia ni un píxel y todo
lo que hay debajo se desplaza 52 px. No es cabecera, ni sinopsis plegada/expandida (`.sinopsis`
mide 96 px en ambos frames), ni esqueletos, ni imágenes, ni barra de acción.

## 4. Por qué B7b da Δ 0 y `medirDerivaScroll` da +52 (mismo mecanismo)

Mismo trazado en el escenario `shell` B7b (scroll al fondo):

| t | vista | `scrollY` | `scrollHeight` | `.datos` | `.ver-mas` |
|---:|---|---:|---:|---:|---|
| 1893 ms | reproductor | 0 | 740 | — | no |
| **1922 ms** | **montaje** | **28 932** | **29 672** | 215,78 | no |
| **1929 ms** | **asentado** | **28 984** | **29 724** | 267,78 | sí (44) |

`scrollTo` pedido: **28 984**, pero con `scrollHeight` 29 672 el máximo es **28 932** → se recorta
52 px; al aparecer el botón, el documento crece 52 y Chromium deja el scroll en el nuevo máximo
(28 984). **Δ neto 0 por casualidad**: el recorte de 52 y la compensación de 52 se anulan. El
defecto existe igual (el frame de montaje está 52 px por encima de lo que debería), pero B7b no lo
ve porque mide al fondo. `medirDerivaScroll` mide a media lista, donde no hay recorte y el error se
manifiesta como **+52**.

## 5. Causa concreta (archivo / regla)

1. `frontend/src/VistaDetalle.svelte:251-264` — `sinopsisDesborda` arranca en `false` y se decide
   dentro de un `requestAnimationFrame`:
   `sinopsisDesborda = $isNarrow && sinopsisEl.scrollHeight > sinopsisEl.clientHeight + 1;`
   (`:260`), programado por `$: if ($currentDetail?.synopsis && $isNarrow) programarMedirSinopsis();`
   (`:264`). Al **remontar** el detalle el estado vuelve a `false` y el botón **no está** en el
   primer frame.
2. `frontend/src/VistaDetalle.svelte:340-350` — el botón sólo se renderiza
   `{#if $isNarrow && sinopsisDesborda}`.
3. `frontend/src/VistaDetalle.svelte:498-504` — `.ver-mas { min-height: var(--tap); margin: 0 0
   var(--sp-2); }` con `--tap: 44px` (`frontend/src/style.css:93`) y `--sp-2: 8px`
   (`frontend/src/style.css:84`) → **44 + 8 = 52 px**. Medido en el frame asentado: alto 44 y
   `margin-bottom` 8.
4. `frontend/src/App.svelte:67-73` — `sincronizarVista` guarda el scroll de la vista que se deja
   (`:69`) y restaura `window.scrollTo(0, recuperarScroll(clave))` tras `await tick()` (`:71-72`).
   Ese `tick` **no** incluye el `requestAnimationFrame` posterior de la sinopsis: la restauración
   se aplica con el documento 52 px más corto.
5. Al pintarse el botón, `.datos` crece 52 px **por encima** de la posición restaurada; el
   *scroll anchoring* de Chromium mantiene el ancla visible y sube `scrollY` +52. Resultado
   numérico 2 743 → 2 795 y, en coordenadas de contenido, la lista queda **52 px más adelante**
   de donde estaba (`.rejilla` en el viewport: **-2073 → -2125**).

Antigüedad: `.ver-mas`/`sinopsisDesborda` **entran en T79** (`0f2c151`, ficha del detalle,
`VistaDetalle.svelte:216/225/301/304/432`) y **no existen en T67** (`39f7ab2`: 0 apariciones). Es
decir, la deriva es **posterior a T67 y anterior a T87**, coherente con lo que midió T89.

## 6. Variaciones (una a la vez, solo desde el navegador: `addStyleTag`)

Mismo escenario (Bleach, 360×740, tarjeta 30); `rejillaView` = `rect.top` de `.rejilla` en el
viewport antes → después (posición visual del contenido):

| Variación | `y` antes → después | Δ `y` | Δ `.datos` | `rejillaView` | ¿elimina los 52? |
|---|---|---|---:|---:|---|---|
| **Base** | 2743 → 2795 | **+52** | 0 (ya tenía botón) | -2073 → -2125 | — |
| `.ver-mas { display: none !important }` | 2691 → 2691 | **0** | 0 | -2073 → -2073 | **sí** (Δ 0 y posición igual) |
| `.sinopsis { display: none !important }` | 2631 → 2631 | **0** | 155,78 → 103,78 | -2073 → -2125 | **sí el número** (no reaparece el botón; el hueco de 52 no existe) |
| `.rejilla .celda { content-visibility: visible !important }` | 2743 → 2795 | **+52** | 0 | -2073 → -2125 | **no** (no es la rejilla) |
| `html { overflow-anchor: none !important }` | 2743 → 2743 | **0** | 0 | -2073 → -2073 | **sí el número** (el salto de 52 lo absorbe el layout) |
| **1280×800** (ancho: `$isNarrow` falso → sin botón) | 1692 → 1692 | **0** | 0 (`.datos` 43,89 sin botón) | -1004 → -1004 | **sí** |
| **Naruto** a 360 (otro anime, misma sinopsis recortada) | 2743 → 2795 | **+52** | 0 | -2073 → -2125 | no (mismo caso) |

Lectura:

- Ocultar el botón o la sinopsis **elimina el +52**; ocultar la rejilla *no*; a 1280 (donde no hay
  botón) **no hay deriva**. La rejilla y `content-visibility` quedan descartados.
- `overflow-anchor: none` demuestra el mecanismo: sin anclaje, `scrollY` se queda en 2 743 (Δ 0)
  y el contenido **no** se mueve de sitio; el anclaje es lo que convierte el desfase de montaje en
  +52. Por eso la corrección no es desactivar el anclaje, sino **que no haya crecimiento tardío**.
- Caso aparte (no es una segunda causa): un anime de **1 episodio** (Ponyo, `scrollHeight` 948)
  da 55 → 156 (Δ 101). Traza: el valor que el detalle **guardó** fue **208** (la página se
  desplazó entre la medición de `antes` y el clic) y se restauró sobre el documento corto
  (`scrollHeight` 896, máximo 156) → recorte a 156, sin compensación porque ya está en el fondo.
  El déficit de montaje es el mismo (896 vs 948, 52).

## 7. Comparación con T89b y con T66/T67

- **T89b (`876d14c`)**: `git checkout 876d14c -- frontend/src` (solo cambian `api.ts`,
  `Icono.svelte` y `VistaReproductor.svelte` respecto a HEAD; el detalle es idéntico) → mismo
  escenario **2/2: 2743 → 2795 (+52)**, `scrollHeight` 29 724. Restaurado después con
  `git checkout HEAD -- frontend/src`.
- **T67 (`39f7ab2`)**: se intentó el mismo escenario con `git checkout 39f7ab2 -- frontend/src`;
  el script no completó (se quedó esperando selectores de la UI de v1.3 más allá de 600 s), así
  que **no se usa como medición**. El árbol se restauró (`git checkout HEAD -- frontend/src` +
  borrado de los dos ficheros de fuente de Nunito que ese checkout reintrodujo, ya retirados en
  T89) y `git status --porcelain` quedó vacío. Lo que sí queda demostrado por código: `.ver-mas`
  **no existe en T67** y sí desde T79.
- Conclusión: la deriva **no la introducen T87 ni T88** (T89b da lo mismo que HEAD) y **sí entra
  con el botón «Ver más» de T79**.

## 8. Log del script y salud del servidor

- Sin `ERR_EMPTY_RESPONSE` ni `Cannot read properties of undefined (reading 'main')` en ninguna
  ejecución: la app respondió 200 en `:34115` antes y después de cada tanda (incluidas las dos
  reversiones de `frontend/src` y su restauración), y todas las pasadas del escenario principal
  dieron el mismo estado sano (366 celdas, sin estados de error ni esqueletos).
- Único `pageerror`: `Cannot read properties of null (reading 'nodes')`, filtrado como ruido del
  dev server ya diagnosticado en T96 (`docs/investigacion/pageerror_nodes.md`).
- Único `requestfailed`: `net::ERR_ABORTED` sobre `http://127.0.0.1:<puerto>/proxy?...` (manifiesto
  HLS del proxy local) cuando el reproductor se desmonta al pulsar «← Atrás». Es una petición
  **cancelada por la navegación**, no un fallo del servidor: en el navegador del contenedor el
  proxy de vídeo no es alcanzable (AGENTS.md) y el reproductor cae al iframe. No invalida las
  mediciones (no dependen del vídeo).

## 9. Corrección mínima propuesta (NO aplicada) y ¿T97b?

La causa es de código y pequeña; **recomiendo T97b**. Opciones, de menos a más invasiva:

1. **Medir la sinopsis de forma síncrona al montar** (preferida): en `VistaDetalle.svelte`, hacer
   la medición de `sinopsisDesborda` en `onMount`/al cambiar de detalle **antes del primer
   pintado** (el `scrollTo` de `App.svelte` corre después de `tick()`, así que el botón ya
   existiría y `scrollHeight` sería el definitivo al restaurar). Mantener el `rAF` solo para
   `resize` y re-medir tras `document.fonts.ready`. Coste: ~1 línea movida; efecto lateral
   positivo: desaparece también el salto de 52 px del primer pintado en cualquier detalle estrecho.
2. **Reservar el hueco**: renderizar el contenedor de `.ver-mas` con alto fijo cuando hay sinopsis
   en estrecho (52 px siempre), sin depender de la medición. Más simple pero añade 52 px vacíos
   a las sinopsis cortas.
3. **Retrasar la restauración un frame** en `App.svelte` (`requestAnimationFrame` + `tick` antes
   del `scrollTo`): quitaría el +52 numérico, pero acopla `App` al asentamiento del detalle y no
   corrige el salto de layout del primer pintado.

Criterio de aceptación para T97b: 360×740, Bleach AnimeAV1, tarjeta 30 → `despues - antes = 0`
(±1), `.rejilla` en el viewport igual antes y después (-2073), B7b sigue en Δ 0 y a 1280 no cambia
nada; `shell` (B7b) y `a11y_auditoria.mjs` sin cambios de recuento.

## 10. Limitaciones

- Un único entorno (contenedor, Chromium del MCP global, 360×740 y 1280×800): no se ha medido en
  el `.exe` de Windows ni con motores distintos de Blink; el *scroll anchoring* es la pieza
  específica del motor y en WebView2 (Chromium) debería comportarse igual, pero **no está
  verificado**.
- El escenario depende de la red (AnimeAV1 «bleach» real). Los valores absolutos (2 743 / 29 724)
  pueden cambiar si el proveedor cambia el número de episodios o la sinopsis; el **Δ 52** está
  atado a `.ver-mas` (44 + 8) y a que `$isNarrow` sea cierto.
- T67 no se pudo medir (apartado 7); la atribución temporal se apoya en el código de T67/T79.

## 11. Trazabilidad

- Scripts temporales **fuera del repo**: `C:\Users\azuls\AppData\Local\Temp\opencode\t97\`
  (`repro.mjs` = B7b, `deriva.mjs` = `medirDerivaScroll`, `traza.mjs`, `diff.mjs`,
  `mecanismo.mjs`, `variaciones.mjs`, `animes.mjs` + sus JSON de salida).
- Ejecución: `NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" node <script>` contra
  `http://localhost:34115` (`docker compose up`).
- No se tocaron `frontend/src/**`, `*.go`, `frontend/wailsjs/**`, `verificar_ui.mjs`,
  `a11y_auditoria.mjs` ni los datos guardados (`historial.json`/`favoritos.json` conservan su
  mtime previo: 2026-10-29 01:02 y 00:55 UTC).
- `frontend/package.json.md5` intacto (`2e45587472c047ae3ac1add21e0807cc`).
