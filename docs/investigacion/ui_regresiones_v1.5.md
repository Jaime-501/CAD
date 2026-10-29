# Diagnóstico de regresiones de la v1.5.0 (`estados` y `reproductor`)

Fecha: 27/10/2026 · Tarea **T94**, paso 1 (solo lectura).
Commit de referencia de T89b: **`876d14c`** («fix(ui): T89b cierre de fallos conocidos de la v1.4.0»).
Código actual comparado: `b8d4506` (T93b), rama `feature/pantalla-completa`.

## Método

Mismo script (`docs/investigacion/ui/verificar_ui.mjs`, sin cambios) y el mismo dev server
(`docker compose up`, `localhost:34115`, red real) en los dos estados. Se alternaron **dos
pasadas por estado** (actual → T89b → actual → T89b) para separar ruido de datos vivos:

| pasada | estado del árbol | cómo |
|---|---|---|
| actual 1 | `HEAD` (T93b) | `git checkout HEAD -- frontend/src` |
| T89b 1 | `876d14c` | `git checkout 876d14c -- frontend/src` |
| actual 2 | `HEAD` (T93b) | `git checkout HEAD -- frontend/src` |
| T89b 2 | `876d14c` | `git checkout 876d14c -- frontend/src` |

`frontend/src` en T89b frente a `HEAD` solo difiere en **3 archivos** (`api.ts`,
`Icono.svelte`, `VistaReproductor.svelte`), es decir, exactamente el trabajo de T91–T93b.
Tras cada pasada se restauró el árbol y `git status --porcelain` quedó limpio
(salvo la ficha `tareas/T94.md`, sin trackear: ver «Desviación de precondición»; al cerrar
se archivó en `tareas/archivo/T94.md`).

## Resultados

| modo | comprobación | actual (1 y 2) | T89b (1 y 2) | clasificación |
|---|---|---|---|---|
| `estados` | `360: contain-intrinsic-size coherente` | **NO** (`undefined`) | **NO** (`undefined`) | entorno/datos |
| `estados` | `390: contain-intrinsic-size coherente` | **NO** (`undefined`) | **NO** (`undefined`) | entorno/datos |
| `estados` | `768: contain-intrinsic-size coherente` | **NO** (`undefined`) | **NO** (`undefined`) | entorno/datos |
| `estados` | `1280: contain-intrinsic-size coherente` | **NO** (`undefined`) | **NO** (`undefined`) | entorno/datos |
| `estados` | resto (98) | sí | sí | — |
| `reproductor` | todas (54) | sí | sí | — |

Recuentos, idénticos en las cuatro pasadas:

| modo | actual 1 | actual 2 | T89b 1 | T89b 2 |
|---|---|---|---|---|
| `estados` | 98/102 | 98/102 | 98/102 | 98/102 |
| `reproductor` | **54/54** | **54/54** | **54/54** | **54/54** |

## Los 4 fallos de `contain-intrinsic-size` (`estados`)

- **Esperado por la comprobación** (`verificar_ui.mjs:6338`): a < 900 px,
  `auto <Math.round(hMin)>px`; a ≥ 900 px, `auto 170px`. Con la geometría medida
  (`hMin` = 146 / 146 / 179 / 170,38), eso es:
  - 360 y 390 → `auto 146px`
  - 768 → `auto 179px`
  - 1280 → `auto 170px`
- **Obtenido**: la comprobación lee `eps.muestra?.parcial?.csi`, que es **`null`**, así que
  falla sin poder comparar (`detalle` vacío en la tabla).
- **Causa**: `muestra.parcial` es la primera tarjeta de episodio con `.franja` (progreso
  entre 0 y 0,9) y `muestra.visto` la primera con `.visto`; el detalle de AnimeAV1 «bleach»
  devuelve **366 tarjetas, todas «ninguno»** (0 `.franja`, 0 `.visto`). En el contenedor
  **no existe `historial.json`** (`/root/.config/miapp/` está vacío), así que no hay ningún
  episodio parcialmente visto que muestrear. En T89 ese archivo sí tenía progreso y la
  comprobación pudo medir (102/102).
- **El CSS es correcto** (medido aparte, con un script temporal fuera del repo, sobre la
  primera tarjeta de la rejilla a los cuatro anchos): `contain-intrinsic-size` =
  **`auto 146px`** (360 y 390), **`auto 179px`** (768) y **`auto 170px`** (1280), con
  `content-visibility: auto`; las alturas reales coinciden (146 / 146 / 179 / 170,38).
  Es decir: la declaración no ha cambiado y el fallo es **solo de datos** (no hay muestra).
- **No depende del anime ni de la lista viva** salvo en el sentido de que hace falta un
  registro de progreso: la clasificación es por `.franja`/`.visto`, no por el título. Con
  historial vacío falla siempre; con un episodio a medias pasaría.
- **Confirmación (28/10, misma sesión)**: al sembrar historial en el contenedor
  (`a11y_auditoria.mjs` lo hace para probar la fila «Continuar» y el cajón), `estados`
  pasó a **102/102 OK** con el mismo código. La sensibilidad es, por tanto, de datos y no
  de código: sin historial no hay muestra que medir; con historial la comprobación mide
  `auto 146px` / `auto 179px` / `auto 170px` y pasa.

## El caso D de `reproductor`

El informe de T93/T93b anotó **50/52** con 2 fallos de «no se llegó al iframe» (caso D).
En esta máquina eso **no se reproduce**: las cuatro pasadas dan **54/54**, y en las cuatro
el caso D entra en la rama del iframe («Extras abiertas automáticamente (iframe): **sí**»),
de ahí que salgan 54 comprobaciones (52 + las 2 de la rama `auto`).

- **Fuente cargada**: el reproductor intenta primero el HLS (`HLS · SUB`), luego el MP4
  directo y, al fallar los dos, cae a un iframe; el plegable de fuentes se abre solo.
- **Por qué fallan HLS/MP4**: el `<video>` apunta al proxy local
  (`http://127.0.0.1:<puerto>/proxy?s=…`) y el navegador del host recibe
  **`net::ERR_CONNECTION_REFUSED`**: el proxy escucha en loopback **dentro** del contenedor
  y ese puerto no está publicado (gotcha documentado en `AGENTS.md` y en
  `docs/DESARROLLO.md`). Con HLS y MP4 inaccesibles, el reproductor salta al iframe.
- **Comparación con T89b**: idéntica en las dos pasadas (mismo `iframeAuto: sí`, mismas 54
  comprobaciones). El 50/52 era el escenario contrario (el proxy servía el HLS en el
  navegador y el reproductor nunca caía al iframe); no es una regresión de T91–T93b.

## Fallos ajenos al diagnóstico (encontrados en la verificación final)

Al correr los 20 modos completos aparecieron dos fallos más, ninguno de T91–T93b:

| modo | comprobación | actual | T89b | clasificación |
|---|---|---|---|---|
| `tactiles` | (bloqueo) `page.waitForSelector('.btn')` | NO: *Timeout 20000ms* | **NO: idéntico** | expectativa obsoleta (preexistente) |
| `shell` | `toast sobre la barra` | NO con historial vacío → **sí con historial sembrado** | — | entorno/datos |
| `buscador` | `360: encabezado Favoritos visible en su panel` | NO con 0 favoritos → **sí con favoritos** (61/61) | — | entorno/datos |

- **`tactiles`**: el modo espera un `.btn` visible en el Home para empezar a medir. En el
  Home ese primitivo **no existe**: la actualización de cada fila es `button.actualizar` y
  `Btn` (`.btn`) solo se renderiza en los estados de error (`Reintentar` de
  `VistaInicio`/`VistaBusqueda`/`GridFavoritos`/`BarraLateral`/`VistaReproductor`). Con el
  Home sano, el selector nunca aparece y el modo aborta con 1 objetivo incumplido. Se
  comprobó con el código de T89b (`git checkout 876d14c -- frontend/src`): **falla
  exactamente igual**, y `VistaInicio.svelte`/`Btn.svelte`/`style.css` ni siquiera cambian
  entre T89b y `HEAD`. Es una **expectativa obsoleta del verificador**, no una regresión;
  no se corrige aquí porque `verificar_ui.mjs` es de solo lectura en esta tarea.
- **`shell`**: la comprobación borra una entrada del historial del cajón para ver el toast
  y, sin historial, el propio script informa `"sin historial sembrado"`. Con historial
  sembrado pasa (`{"bottom":656,"topBarra":680}`), igual que `estados`: **entorno/datos**.
- **`buscador`**: con **0 favoritos**, el panel `#panel-favoritos` mantiene su `h2`
  («Favoritos (0)») con `visibility: hidden` en el momento de la comprobación (350 ms tras
  el clic), así que `isVisible()` da `false`. Con favoritos guardados (7 en esa sesión) el
  panel es visible y el modo pasa **61/61**; no se toca el `.mjs`.

## Por qué algunos totales no coinciden con la v1.4

Tres modos dan menos comprobaciones que las cifras que circulan de la v1.4. Comprobado uno a uno
(no asumido):

| modo | total real del script | cifra de la v1.4 | veredicto |
|---|---:|---:|---|
| `favoritos` | **62** | 62 (T85) | coincide; la cifra baja de mi primera pasada era una **corrida rota** |
| `cta` | **56** | 59 (tabla de T89) | **56 es el total desde T80**; el 59 no sale de este script |
| `continuar` | **43** | 46 (tabla de T89) | **43 es el total desde T77**; el 46 no sale de este script |

- **`favoritos` 19/19 (corrida inválida).** Esa pasada se ejecutó mientras el contenedor se
  estaba cayendo: el propio log lista **3 `page.goto: ERR_EMPTY_RESPONSE`** y **3
  `Cannot read properties of undefined (reading 'main')`** (puente `window.go` ausente), así que
  las secciones de estados, toast y deslizar **no llegaron a ejecutarse** y solo quedaron los
  checks de geometría (3 anchos × 6 + 1 = 19). Con el servidor sano, la misma pasada da
  **62/62** y 0 errores. Total histórico (BITACORA, T85): **62/62**.
- **`cta` 56/56 y `continuar` 43/43 no han bajado: son sus totales de siempre.**
  `BITACORA.md` documenta **`cta` 56/56** en T80 y **`continuar` 43/43** en T77, y
  `docs/PROGRESO_v1.4.md:787` repite **`cta` 56/56** en la re-ejecución de T80. Las cifras
  **59/59** y **46/46** aparecen solo en la tabla «todos los modos» de T89 (y de ahí pasaron al
  cierre de la v1.4): están **infladas en +3** respecto al script real.
- **El script no cambió en estas tres funciones.** Contando sitios `check(` dentro de cada
  función en **todas** las revisiones que han tocado `verificar_ui.mjs` desde su creación:
  `continuar` **24**, `cta` **33** y `favoritos` **46**, idénticos en `662cf8f` (T77),
  `8086740` (T80), `9be6d7a` (T85), `876d14c` (T89b) y `b8d4506` (T93b). T89b solo relajó las
  comprobaciones de `reduced-motion` (`duracionCorta`), sin añadir ni quitar ninguna.
- **Los totales se pueden recalcular a mano** (no dependen de los datos):
  `continuar` = 3 anchos estrechos × 8 + 2 anchos × 3 + B(4) + C(3) + D(1) + E(3) + F(1) + G(1)
  = **43**; `cta` = 3 × (8 + `overflowH`) + 2 × (4 + `overflowH`) + B(4) + C(3) + D(5) + E(3) +
  F(2) + G(1) + H(1) = **56**.
- **Confirmación empírica con el código de T89b**: `git checkout 876d14c -- frontend/src` y
  repetir ambos modos da **`cta` 56/56** y **`continuar` 43/43**, exactamente igual que con
  `HEAD`. No hay checks perdidos ni añadidos por T91–T94.

Conclusión de este punto: **no falta ninguna comprobación**; lo que hay es una **errata de la
tabla de T89** en `docs/PROGRESO_v1.4.md` (59 y 46 en lugar de 56 y 43), que no se corrige aquí
porque ese archivo es histórico y queda fuera de «Puedes tocar». Los totales válidos son los de
la tabla de arriba.

## Conclusión

**No hay ninguna regresión de código.** Las dos «degradaciones» del cierre de la v1.5.0:

1. `estados` 98/102 → los 4 fallos de `contain-intrinsic-size` son **entorno/datos**: falta
   un episodio con progreso parcial en el historial del contenedor; el valor declarado y
   medido es el esperado. Con historial sembrado el mismo código da **102/102**. (Ya ocurría
   en T93b con la regla revertida.)
2. `reproductor` 50/52 → **no se reproduce** en este entorno: sale 54/54 en ambos estados
   porque el proxy de loopback no es alcanzable desde el navegador del host y el caso D
   (iframe) sí se ejercita. Es **entorno/datos**.

Procede el cierre documental de la v1.5.0 (pasos 2–6) con el `.mjs` intacto. Las
situaciones quedan anotadas en «Pendientes activos» de `PROGRESO.md` como sensibilidad a los
datos guardados (`estados`, `shell`, `buscador`) y como expectativa obsoleta del verificador
(`tactiles`, selector `.btn` en el Home).

## Desviación de precondición (Paso 0)

`git status --porcelain` no estaba estrictamente vacío: mostraba `?? tareas/T94.md`
(la ficha de esta misma tarea, sin trackear). Todo el trabajo de T91–T93b está commiteado
(`8c6b47d`, `095915f`, `11f075f`, `b8d4506`) y no había ningún otro cambio, así que se
procedió dejando constancia aquí y en `PROGRESO.md`.
