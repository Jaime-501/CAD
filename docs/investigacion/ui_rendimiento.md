# T87 — Movimiento y rendimiento (v1.4.0)

Fecha: **2026-10-20**. Rama: `feature/rediseno-movil`. Alcance: `prefers-reduced-motion`,
`prefers-reduced-transparency`, respaldos de `backdrop-filter`, `decoding="async"` y medición
del scroll. **Sin cambios visuales con la configuración por defecto.** No se tocó ningún
`<script>` de producto, ningún `*.go` ni `frontend/wailsjs/**`.

## Cambios

- **`frontend/src/style.css`**: regla global de movimiento reducido y regla de transparencia
  reducida para `.barra-accion`, `.barra-inferior` y `.cabecera.fija`.
- **`BarraAccion`, `BarraInferior`, `CabeceraVista`**: respaldo `@supports not
  (backdrop-filter: blur(1px))` con fondo `var(--bg-1)` (la cabecera, dentro de su modo
  estrecho).
- **`VistaInicio`**: rama `reduce` explícita para `.esqueleto` (la global ya la cubría).
- **`TarjetaAnime`, `TarjetaEpisodio`, `TarjetaContinuar`**: `decoding="async"` en la `<img>`.
- **`docs/investigacion/ui/verificar_ui.mjs`**: modo `rendimiento` (medición, geometría,
  reduced-motion, reduced-transparency y T66/T67) y ajuste de la comprobación de transición de
  `Btn` para el nuevo valor global.

### Regla global aplicada

```css
@media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
        animation-duration: 0.01ms !important;
        animation-iteration-count: 1 !important;
        transition-duration: 0.01ms !important;
        transition-property: none !important;
        scroll-behavior: auto !important;
    }
}
```

**Desviación consciente:** se añade `transition-property: none !important` a la regla de la
ficha. Sin esa declaración, `transition-duration` activa la propiedad por defecto
`transition-property: all` en **todos** los elementos: al volver del reproductor, el
`padding-bottom` del `main` y el alto de la ficha conservan durante un frame el valor
anterior. El `scrollTo(0, 28984)` de T67 se ejecuta en ese frame y el navegador lo recorta a
**28908** (falla T66/T67 por 76 px). Con `transition-property: none` no hay transición
implícita y la restauración vuelve a **28984 → 28984**. La comprobación de `Btn` del modo
`estados` acepta ahora `transition-duration: 0s` **o** `transition-property: none`
(`1e-05s / none`), que es el resultado correcto de la regla.

## Método de medición

- Chromium de Playwright, `--enable-precise-memory-info`, CDP `Performance.getMetrics` y
  `requestAnimationFrame`.
- Detalle de **AnimeAV1 «bleach»** (366 episodios) a **360×740** y **1280×800**; scroll de
  arriba abajo en pasos de **1000 px**. Home y búsqueda (`bleach`, 21 resultados) con el
  mismo procedimiento.
- Para separar la mejora real del ruido de la máquina se hizo una comparación **A/B alterna**:
  se midió T86 (`git stash` solo de `frontend/src`), se restauró T87 (`git stash pop`) y se
  repitió, **3 pasadas por lado**. La tabla usa la **mediana** de esas 3 pasadas.
- `scripting`, `tarea` y `layout` son acumulados de la pasada (CDP, ms). `frames >50` cuenta
  frames largos; `p95`/`máx` son duraciones de frame (ms).

## Antes / después (medianas A/B, 3 pasadas)

| vista | ancho | frames | >50 ms | p95 | máx | scripting | tarea | layout | nodos | alto |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| home | 360 | 37 | 0 | 16.7 | 16.8 | 1.3 | 11.5 | 0 | 522 | 1828 |
| home | 1280 | 30 | 0 | 16.7 | 16.8 | 1.2 | 10.9 | 0 | 561 | 1629 |
| búsqueda | 360 | 44 | 0 | 16.7 | 16.8 | 1.0 | 12.6 | 0 | 162 | 3570 |
| detalle | 360 | 243 | 0 | 16.8 | 16.8 | 6.1 | 911.9 | 70.5 | 2680 | 29724 |
| detalle | 1280 | 159 | 0 | 16.7 | 16.8 | 4.1 | 552.1 | 25.4 | 2791 | 18033 |

**T87 (después):**

| vista | ancho | frames | >50 ms | p95 | máx | scripting | tarea | layout | nodos | alto |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| home | 360 | 37 | 0 | 16.7 | 16.8 | 1.2 | 12.7 | 0 | 522 | 1828 |
| home | 1280 | 29 | 0 | 16.7 | 16.8 | 0.6 | 6.6 | 0 | 561 | 1629 |
| búsqueda | 360 | 45 | 0 | 16.8 | 16.8 | 0.9 | 11.5 | 0 | 162 | 3570 |
| detalle | 360 | 240 | 0 | 16.8 | 16.8 | 5.4 | 843.3 | 67.7 | 2680 | 29724 |
| detalle | 1280 | 157 | 0 | 16.8 | 16.8 | 3.6 | 462.9 | 20.9 | 2791 | 18033 |

**Variación (después − antes, sobre la mediana):**

| vista | scripting | tarea | layout | nodos | frames >50 |
|---|---:|---:|---:|---:|---:|
| home 360 | −7,7 % | +10,4 %* | — | 0 | 0 |
| home 1280 | −50,0 % | −39,4 % | — | 0 | 0 |
| búsqueda 360 | −10,0 % | −8,7 % | — | 0 | 0 |
| detalle 360 | **−11,5 %** | **−7,5 %** | **−4,0 %** | 0 | 0 |
| detalle 1280 | **−12,2 %** | **−16,2 %** | **−17,7 %** | 0 | 0 |

\* Home 360 es el único valor con mediana peor: 11,5 → 12,7 ms. Es una vista de 2 pasos y
~37 frames; la **media** baja de 13,0 a 11,5 ms y las tres pasadas de después (8,2 / 13,6 /
12,7 ms) solapan con las de antes (11,5 / 20,4 / 7,2 ms). No hay frames largos. No se
considera regresión real.

Ninguna medición de las vistas grandes empeora; el detalle (que es el caso de la ficha)
mejora en scripting, tarea y layout, con **0 frames >50 ms** y los mismos nodos DOM. No fue
necesario revertir ningún cambio.

## Movimiento reducido

Medido a 360×740 y 1280×800 (`reducedMotion: 'reduce'`):

| ancho | animaciones | con duración | celdas | visibles | pendientes | esqueletos animados |
|---:|---:|---:|---:|---:|---:|---:|
| 360 | 0 | 0 | 366 | 366 | 0 | 0 |
| 1280 | 0 | 0 | 366 | 366 | 0 | 0 |

- **T66/T67 con reduce (360): OK**, `y 28984 → 28984`.
- El revelado T65 deja las 366 celdas visibles (no se queda en `.pendiente`).
- `Btn` computa `transition-duration: 0.00001s` y `transition-property: none`; no hay
  transición efectiva.

## Transparencia reducida

Con `prefers-reduced-transparency: reduce`, a 360:

| superficie | background-color | backdrop-filter |
|---|---|---|
| barraInferior | `rgb(15, 23, 42)` (`--bg-1`) | `none` |
| barraAccion | `rgb(15, 23, 42)` (`--bg-1`) | `none` |
| cabeceraFija | `rgb(15, 23, 42)` (`--bg-1`) | `none` |

Se encontraron **3** reglas `@supports not (backdrop-filter)` (una por componente). Con la
configuración por defecto no se aplica ninguna y el blur sigue igual.

## Comparación visual y geometría

- `comparar antes despues 2`: **6 capturas idénticas**, 4 con diferencias. Las diferencias
  son de **portadas/miniaturas vivas** (`home_360` 97 px, máx Δ6; `detalle_768` 28 px,
  máx Δ5; `reduced_motion_1280` 95 px, máx Δ6). La misma caja de `detalle_768`
  (28 px en `y=740`) aparece al comparar **dos pasadas de después**, así que es contenido
  remoto, no un cambio de layout. `reduced_transparency_360` difiere a propósito (es la
  función que se está verificando).
- Geometría idéntica: Home **263,8**; tarjeta de episodio **146 / 179 / 170,38**;
  `overflowH = 0` en todos los anchos; nodos DOM idénticos (522/561/162/2680/2791).
- Capturas: `docs/img/v1.4/rendimiento/antes/`, `despues/` y las pasadas A/B
  `control_antes{1,2,3}/`, `control_despues{1,2,3}/`; `estados/` guarda la regresión de
  `estados` (102/102).

## Duraciones sueltas que se dejan

- **`BarraLateral:329` — `width 180ms`**: animación de plegado/despliegue de la barra; tiene
  rama local `prefers-reduced-motion: reduce { transition: none; }`. Cambiar la duración
  alteraría el movimiento por defecto, que la ficha manda conservar.
- **`VistaDetalle:610` — `entra-celda 320ms`**: entrada de las tarjetas de episodio de T65;
  tiene rama local `reduce { animation: none; }`. Es la animación que T65 midió sin jank; se
  mantiene.
- **`VistaInicio:338` — `brillar 1.4s`**: shimmer de los esqueletos, solo dentro de
  `prefers-reduced-motion: no-preference`; con `reduce` no existe.
- **`BotonSiguiente:109` — `pulso-siguiente 2s`**: pulso del chip resaltado, también solo en
  `no-preference`; con `reduce` no existe.
- Las cuatro quedan cubiertas por la regla global y por sus ramas locales. No se tokenizan
  porque son duraciones de animación concreta, no escalas de movimiento.

## Verificación

| Comprobación | Resultado |
|---|---|
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** (176 módulos; aviso preexistente de chunk >500 kB) |
| `etiquetas.test.ts` | **32** comprobaciones OK |
| `ventana.test.ts` | **48** comprobaciones OK |
| `views.test.ts` | **13** comprobaciones OK |
| `go test -race -count=1 -v ./...` | **435 `--- PASS`** de nivel superior, **0 FAIL** |
| `verificar_ui.mjs estados` | **102 / 102 OK** |
| `verificar_ui.mjs rendimiento despues` | T66/T67 **OK**, 0 animaciones, 366/366 visibles, barras opacas, geometría intacta |
| `grep -rn wailsjs frontend/src` | solo **`api.ts`** |

Go no se tocó (`*.go` intactos); los 435 PASS se ejecutaron igualmente como compuerta.

## Notas y pendientes

- El modo `estados` se ejecutó con salida en `docs/img/v1.4/rendimiento/estados` para no
  escribir fuera de las rutas autorizadas por T87.
- La comprobación `reduced-motion: Btn sin transición` se ajustó a la regla global: antes
  exigía `0s`; ahora acepta `transition-property: none`, que es el efecto real de la regla.
- `docs/img` está ignorado por `.gitignore`, como en las tareas anteriores; las capturas
  existen en el worktree.
- **T88 no iniciada.** Sin tag ni release; los hace el humano.
