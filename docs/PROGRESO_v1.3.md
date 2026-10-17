# PROGRESO v1.3.0 — Archivo (T64–T67: episodios incrementales y revelado animado)

> **Archivo de solo lectura** (mover, no borrar). Contenido movido de `PROGRESO.md` al cerrar la
> **v1.3.0** con **T68**: las notas largas de **T64** (lógica pura de `ventana.ts`), **T65**
> (revelado animado de tarjetas por visibilidad, sin paginar), **T66** (clave de scroll del
> detalle por proveedor + slug) y **T67** (restauración del scroll del detalle tras la recarga por
> lotes). La serie **T64–T69** (rama `feature/episodios-incrementales`) muestra los episodios del
> detalle de forma incremental y con animación de entrada; los **T66–T67 originales** (paginado /
> ventana client-side) quedaron **descartados** por las mediciones de T65.
> El estado **vigente** vive en [`PROGRESO.md`](../PROGRESO.md): T01–T68 hechas y la **v1.3.0
> preparada** (`productVersion` 1.3.0; sin tag ni release — los hace el humano). Historial de
> sesiones y decisiones cerradas: [`BITACORA.md`](BITACORA.md) · firmas reales:
> `firmas/<paquete>.md`.

---

## Nota de cierre (T68) — uso actual de `ventana.ts`

`frontend/src/ventana.ts` (T64) expone más lógica de la que la UI consume hoy. **Del módulo solo
se usa `retardoEntrada`** (lo importa `revelar.ts` para escalonar la animación por columna). Las
demás funciones —`normalizarColumnas`, `tamanoInicial`, `siguienteTamano`, `cubrirIndice`,
`claveVentana` y la memoria por sesión `recordarVisibles`/`recuperarVisibles`/`olvidarVisibles`—
**quedan disponibles** para una posible **paginación/ventana de episodios futura**, que T65 dejó
**descartada**: las mediciones (366 tarjetas / 1830 nodos y hasta 1180 con One Piece) no mostraban
jank ni saltos de scroll, y el coste real de JKAnime (~16 s por 366 episodios) es de **red**, ya
cubierto por el mostrado por lotes. **No se borró código** y **no se tocó `frontend/**`** en el
cierre (T68 es solo documentación): la ventana queda probada por `ventana.test.ts` por si más
adelante se revierte la decisión.

---

## Notas v1.3.0 (T64–T67)

## T64 — Lógica pura de ventana de episodios (`ventana.ts` + tests)

Hecha el **2026-10-16**. Primera tarea de la serie **T64–T69** (v1.3.0, rama `feature/episodios-incrementales`): muestra los episodios del detalle de forma incremental y con animación de entrada. **Solo frontend**: `frontend/src/ventana.ts` (nuevo, TypeScript puro, sin imports del proyecto ni de Svelte) y su test. Ninguna UI, ningún Go.

**Qué se hizo:**

- **`ventana.ts`** exporta exactamente: constantes `FILAS_INICIAL = 5`, `FILAS_TRAMO = 4`, `COLUMNAS_POR_DEFECTO = 4`, `RETARDO_POR_COLUMNA_MS = 50`, `RETARDO_MAXIMO_MS = 300`; funciones puras `normalizarColumnas`, `tamanoInicial`, `siguienteTamano` (redondeo **hacia arriba** al múltiplo de columnas, tope en `total`, nunca por debajo de `min(actual, total)`), `cubrirIndice` (tamaño mínimo que incluye el índice, nunca reduce, `indice < 0` → 0, `indice >= total` → `total`), `retardoEntrada` y `claveVentana`; y la memoria por sesión `recordarVisibles`/`recuperarVisibles`/`olvidarVisibles` sobre un **`Map` a nivel de módulo** (mismo patrón que `scrollPorClave` de `views.ts`, sin stores de Svelte). Todas las entradas inválidas (NaN, Infinity, negativos) devuelven valor seguro **sin lanzar**.
- **`ventana.test.ts`**: mismo mecanismo que `etiquetas.test.ts` (esbuild de Vite + `node`, comando en la cabecera). **48 comprobaciones** con los valores esperados calculados a mano: `normalizarColumnas` (0, -3, NaN, Infinity, 4.9, 5), `tamanoInicial` (total 0, 7 con 4 cols, 366 con 3/4/5 → 15/20/25), `siguienteTamano` (20→40 con 5 cols; 20 con 3 cols → 33; 0→16; 18 con 5 → 40; tope 25; `actual >= total`; total 0), `cubrirIndice` (visible, siguiente tramo 20→28, límite de bloque, no-múltiplo 5→8, `>= total`, negativo, nunca reduce), `retardoEntrada` (col 0/1, módulo, tope 300 con 10 cols, negativo, columnas 0→4), Map (recordar/recuperar/olvidar, inexistente `undefined`, n inválido ignorado) y la propiedad para cols 3/4/5 con total 366 (secuencia estrictamente creciente, termina en 366, intermedios múltiplos).

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `ventana.test.ts` (esbuild + node) | **OK (48 comprobaciones)** |
| `npm run check` (frontend) | **0 errores, 0 avisos** |
| `npm run build` (frontend) | **OK** (aviso de chunk >500 kB preexistente) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` · `go vet ./...` · `go build ./...` · `go test -race -count=1 ./...` | **no ejecutados**: el daemon de Docker no era accesible en el host (`npipe:////./pipe/dockerDesktopLinuxEngine`) y Go no está en el PATH del host. **Go no se tocó**, así que el recuento (`--- PASS` 435) queda igual que al cerrar T63 a falta de repetir la compuerta. |
| `git status --porcelain` | solo `frontend/src/ventana.ts`, `frontend/src/ventana.test.ts` (nuevos) y `tareas/T64.md` (sin trackear, previo) |

**Desviaciones:**

- **Compuerta de Go no ejecutable en esta sesión**: el daemon de Docker no estaba levantado y el host no tiene Go/wails (AGENTS.md). Al no tocarse ni una línea de Go, se espera el mismo recuento que T63 (435), pero queda **pendiente de repetir** por el humano (regla 3).
- **La serie T64–T69 se reserva para la v1.3.0** (`feature/episodios-incrementales`): las propuestas del informe de adds (extractores) se renumeraron a **T70–T75** (nota en el informe; texto histórico conservado).
- **Sin tag ni release**: no aplica a esta tarea.

**Pendientes:**

- **Repetir la compuerta de Go** (`gofmt -l .`, `go vet ./...`, `go test -race -count=1 ./...`) cuando el contenedor esté disponible; sin cambios esperados (Go intacto).
- Sin tag ni release.


## T65 — Revelado animado de tarjetas de episodio por visibilidad (sin paginar)

Hecha el **2026-10-16**. Segunda tarea de la serie **T64–T69** (v1.3.0, rama `feature/episodios-incrementales`). **Solo frontend**: `frontend/src/revelar.ts` (nuevo), `VistaDetalle.svelte` (envoltorio `.celda`, acción, keyframes y medición de columnas) y `TarjetaEpisodio.svelte` (`contain-intrinsic-size`, exigido por la medición). Ningún Go. Con Docker accesible (a diferencia de T64), la compuerta de Go se ejecutó por completo.

### Paso 1 — Mediciones antes de tocar código (Playwright, `localhost:34115`)

Bleach en JKAnime y en AnimeAV1 (366 episodios, lista completa); valores **idénticos** en los dos proveedores (mismo CSS), así que van juntos:

| Config (Bleach, 366 ep.) | Columnas (gridTemplateColumns) | Ancho rejilla | Altura tarjeta | Nodos bajo `.rejilla` | Hijos |
|---|---|---|---|---|---|
| 1280 px · barra **expandida** | **4** (211,75 px ×4) | 901 px | **170,39 px** | **1830** | 366 |
| 1280 px · barra **contraída** | **5** (205,6 px ×5) | 1100 px | **170,39 px** | **1830** | 366 |
| 700 px · barra contraída | **3** (195 px ×3) | 621 px | **170,39 px** | **1830** | 366 |

- **(b) alturas**: la tarjeta **renderizada** mide **170,39 px** (constante); las tarjetas **fuera de pantalla** reservaban **210 px** por `contain-intrinsic-size: auto 210px` → desajuste de **~39,6 px por tarjeta** (origen de saltos de scroll al entrar). Diferencia > 20 px ⇒ **se ajustó** (paso 4 de la ficha).
- **(d) tiempos** (desde abrir el detalle hasta 366 hijos):
  - **JKAnime Bleach** (sin caché, tras reiniciar el contenedor): **16,3 s** — el detalle pinta pronto, pero los episodios llegan por AJAX en lotes de 50.
  - **AnimeAV1 Bleach**: ficha pintada a **0,40 s** y 366 hijos a **0,44 s** (todos los episodios vienen embebidos en el `__data.json` del detalle, en lotes de 50).
  - Control adicional (**One Piece** JKAnime, no cacheado): de 48 a **1180** tarjetas en ~49 s, con **scrollY constante** (ver verificación 4).
- **(c) nodos**: **1830** nodos bajo `.rejilla` (366 × ~5) en los tres anchos; `content-visibility: auto` mantiene el pintado barato. Tras el paso 3, el envoltorio `.celda` añade **1 nodo por tarjeta** (366) → **2196** en total; sin efecto en el pintado (verificación 4: 1180 celdas sin saltos).

**Tabla antes/después del único cambio exigido por la medición (`contain-intrinsic-size`):**

| Medida | Antes | Después |
|---|---|---|
| `contain-intrinsic-size` de `.tarjeta` | `auto 210px` | **`auto 170px`** |
| Altura real renderizada (getBoundingClientRect) | 170,39 px | 170,39 px (sin cambio) |
| Altura reservada fuera de pantalla | 210 px | ~170 px |
| Desajuste / salto potencial por tarjeta | ~39,6 px | ~0 px |

### Paso 2 — `revelar.ts` (nuevo)

Acción Svelte `use:revelar={{ id, indice, columnas }}` para el envoltorio de cada tarjeta:

- **`reveladas: Set<string>` a nivel de módulo** (clave = `episodio.id`, ya con prefijo de proveedor), mismo patrón que `scrollPorClave` de `views.ts`: sobrevive al remontaje de `VistaDetalle`, así que al volver del reproductor ningún id ya revelado se re-anima.
- **Un único `IntersectionObserver` compartido** (`threshold: 0.15`). Al intersectar: añade `entra`, fija `style.animationDelay = retardoEntrada(indice, columnas) + 'ms'` (de `ventana.ts`), registra el id y hace `unobserve`.
- **Estado inicial oculto síncrono**: la acción pone `pendiente` antes del primer pintado (sin parpadeo visible→oculta).
- **No oculta ni anima** si el id ya estaba en `reveladas`, si `prefers-reduced-motion: reduce` o si no hay `IntersectionObserver`.
- Exporta `medirColumnas(rejilla)` (cuenta pistas de `gridTemplateColumns`; si falla devuelve `COLUMNAS_POR_DEFECTO`) y `reiniciarReveladas()` (solo pruebas). `destroy` hace `unobserve` y limpia el registro interno.

### Paso 3 — `VistaDetalle.svelte`

- Cada tarjeta va dentro de un `div.celda` (`display:flex; flex-direction:column`; la tarjeta `flex: 1 1 auto` ocupa todo el alto) que lleva la acción. **La animación vive en `.celda`, no en la tarjeta**, para no chocar con el `transform` del hover ni con `content-visibility`.
- Keyframes `entra-celda` (opacity 0→1 y translateX(-18px)→0, 320 ms, `cubic-bezier(.22,.61,.36,1)`, `both`); `.celda.pendiente` = `opacity: 0`.
- `columnas` se mide con `medirColumnas` sobre `.rejilla` al montar y en `resize` con `requestAnimationFrame` (no se mide en cada píxel).
- `prefers-reduced-motion: reduce` desactiva animación y estado oculto **también por CSS**.
- La lista, el orden, `episodioContinuar`, el scroll y la caché **no cambian**.

### Verificación (Playwright, `localhost:34115`; 8/8)

1. **Solo el viewport se anima, izq→der por fila**: a 1280 expandida, primeras 4 celdas con `animationDelay` **0/50/100/150 ms** y la 5.ª fila reinicia en 0 (**8 `entra`**, 358 `pendiente` de 366).
2. **Scroll**: al bajar, `entra` pasó de 8 a 32; la primera fila conserva `entra` con sus retardos originales (no se repite).
3. **"← Atrás" desde el reproductor**: las celdas ya reveladas vuelven **sin** `entra` ni `pendiente` (visibles al instante, sin re-animación); el resto sigue `pendiente`.
4. **Lotes tardíos sin saltos**: One Piece (48→1180 celdas, ~49 s) con `scrollY` **exactamente 900** todo el tiempo (salto **0 px**).
5. **`prefers-reduced-motion: reduce`**: 0 celdas con `entra`/`pendiente`, `opacity: 1` al instante.
6. **Hover intacto**: `transform: matrix(1,0,0,1,0,-5)` = `translateY(-5px)`.
7. **Anchos coherentes**: expandida 4 cols → 0/50/100/150; contraída 5 cols → 0/50/100/150/**200**; 700 px 3 cols → 0/50/100 (sin desbordamiento horizontal).
8. **Consola**: sin errores nuevos de nuestro código (solo el `favicon.ico` 404, el `TypeError` de `wails/ipc.js` y los `ERR_CONNECTION_REFUSED` al proxy local, todos preexistentes y documentados).

### Compuerta (resultado real)

| Comando | Resultado |
|---|---|
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `ventana.test.ts` (esbuild + node) | **OK (48 comprobaciones)** — sin cambios |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (Go intacto) |
| recuento `^--- PASS` (`-count=1 -v`) | **435** — idéntico al de T63 (T65 no toca Go) |
| `git status --porcelain` | solo `frontend/src/revelar.ts` (nuevo), `frontend/src/VistaDetalle.svelte`, `frontend/src/TarjetaEpisodio.svelte`, `PROGRESO.md` y `tareas/T65.md` (sin trackear, previo) |

### Recomendación sobre T66–T67 (¿paginar?)

**Paginar T66–T67: NO.** Las mediciones no muestran problema de DOM/pintado que lo justifique: 366 tarjetas (1830 nodos, 170 px) y hasta 1180 (One Piece) se renderizan **sin jank y con 0 saltos de scroll** gracias a `content-visibility: auto` + el revelado por visibilidad. El coste real medido es **de red** (JKAnime tarda ~16 s en traer los 366 episodios por AJAX), y eso ya lo cubre el mostrado incremental por lotes; limitar/paginar tarjetas añadiría complejidad sin ganancia observable. Reserva: solo se midieron 2 proveedores (hasta 1180 episodios) en escritorio; si aparecen listas muy superiores o equipos de gama baja, el `content-visibility` actual ya acota el pintado. (La ventana de `ventana.ts` de T64 queda disponible y probada si más adelante se decide lo contrario.)

### Desviaciones

- **`contain-intrinsic-size` ajustado a `auto 170px`** (antes `auto 210px`): la medición (b) dio 170,39 px reales frente a 210 px reservados (desajuste ~39,6 px/tarjeta > 20 px). Es la única razón por la que se tocó `TarjetaEpisodio.svelte` (la ficha solo lo permite "si la medición del paso 1 lo exige").
- **Clases dinámicas con `:global`**: `pendiente`/`entra` las añade la acción en JS, así que Svelte las marcaba como "Unused CSS selector". Se escriben como `.celda:global(.pendiente)` / `.celda:global(.entra)`; el `@keyframes entra-celda` se scopea con hash y la referencia se reescribe correctamente (verificado en el CSS de `dist/`: `animation: svelte-…-entra-celda`).
- **`columnas` se recalcula solo al montar la rejilla y en `window.resize`** (según la ficha), **no** al plegar/desplegar la barra lateral (eso no dispara `resize`); con la barra cambiada sin redimensionar la ventana, un retardo nuevo podría usar el nº de columnas anterior hasta el siguiente resize.
- **Medición (d) interpretada "desde abrir el detalle"**: los 170 ms de diferencia entre "ficha pintada" y "366 hijos" en AnimeAV1 confirman que el grueso del tiempo de JKAnime (~16 s) es de red, no de render.

### Pendientes

- **Ninguno propio.** La compuerta de Go que T64 dejó pendiente por el daemon de Docker se ha **repetido aquí en verde** (435 tests, 11 paquetes ok): queda cerrada esa deuda.
- Sin tag ni release.


## T66 — Clave de scroll del detalle por proveedor + slug

Hecha el **2026-10-16**. Tercera tarea de la serie **T64–T69** (v1.3.0, rama `feature/episodios-incrementales`). **Solo frontend**: `frontend/src/views.ts` (clave de scroll) y `frontend/src/views.test.ts` (nuevo). Ningún Go.

Contexto: T65 decidió **no paginar** y descartó los **T66–T67 originales** (paginado / ventana client-side). Esta tarea recicla el número **T66** para corregir un bug de la revisión: `claveScroll('detalle')` usaba solo el slug y «bleach» existe en JKAnime y AnimeAV1, así que **compartían posición de scroll**.

### Cambio

- **`claveScroll` (`views.ts`)**: `detalle:${sel?.provider ?? ''}:${sel?.animeID ?? ''}` y `reproductor:${sel?.provider ?? ''}:${sel?.episodeID ?? ''}`; `busqueda` sin cambios. Actualizado el comentario de cabecera («Claves: …») y el de la propia función (T26/T66).
- El `episodeID` ya lleva el prefijo del proveedor (`<proveedor>:…`, T34), así que añadir el proveedor a la clave del reproductor es **redundante pero inocuo** (la clave es solo una cadena del `Record` interno y se usa el **mismo** `claveScroll` para guardar y recuperar); se añadió por coherencia con la del detalle.

### Tests (`views.test.ts`, esbuild de Vite, mismo mecanismo que `etiquetas.test.ts`/`ventana.test.ts`)

**13 comprobaciones**: mismo slug con distinto proveedor ⇒ claves distintas (detalle y reproductor); mismo proveedor con distinto slug ⇒ distintas; `guardarScroll`/`recuperarScroll` con las dos claves del mismo slug **no se pisan** (800 frente a 0); clave inexistente ⇒ **0**; selección `null` ⇒ `detalle::` / `reproductor::` **estables sin lanzar**; `busqueda` es única. `views.ts` importa `svelte/store`: el bundle de esbuild lo resuelve desde `frontend/node_modules` (verificado; el comando de la cabecera lo demuestra).

### Verificación (Playwright, `localhost:34115`)

- **JKAnime «bleach», Ep. 1**: bajar a `scrollY = 800` y volver (búsqueda → reabrir el detalle) ⇒ **conserva 800** (**3/3** ciclos; `scrollTo(0, 800)` sobre el detalle ya pintado, ~14 458 px).
- **AnimeAV1 «bleach», Ep. 1**: abrir ⇒ **empieza en 0**, no en 800 (`scrollTo(0, 0)` con la clave propia `detalle:animeav1:bleach`). Esa es la evidencia del fix: con la clave antigua por slug, AnimeAV1 habría heredado el 800 de JKAnime.
- **Observación (comportamiento previo, no de T66)**: al saltar **JKAnime → AnimeAV1 → JKAnime**, el detalle de JKAnime se **recarga** (`VistaDetalle`/`stores.ts` distinguen proveedor: `openAnime` vacía la lista y refetch), y el `scrollTo` de restauración corre tras `await tick()` **antes** de que la lista esté pintada (altura = viewport), por lo que se recorta a **0**. Sin el cambio de T66 el mismo salto daba IGUAL 0 en la vuelta (la lista también se recargaba), pero además AnimeAV1 heredaba el 800; con T66 el 800 se conserva en su clave y la vuelta **directa** (mismo proveedor, lista ya en los stores) lo restaura de forma estable. Es una característica del restaurado con detalle asíncrono, fuera del alcance de T66.
- **Consola**: solo el `favicon.ico` 404, el `TypeError` de `wails/ipc.js` y los `ERR_CONNECTION_REFUSED` al proxy local (todos preexistentes y documentados); **sin errores nuevos** de nuestro código.

### Compuerta (resultado real)

| Comando | Resultado |
|---|---|
| `views.test.ts` (esbuild + node) | **OK (13 comprobaciones)** |
| `ventana.test.ts` (esbuild + node) | **OK (48 comprobaciones)** (sin cambios) |
| `npm run check` (frontend) | **0 errores, 0 avisos** |
| `npm run build` (frontend) | **OK** (aviso de chunk >500 kB preexistente) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (Go intacto) |
| recuento `^--- PASS` (`-count=1 -v`) | **435** — idéntico a T63/T65 (T66 no toca Go) |
| `git status --porcelain` | `frontend/src/views.ts` (M), `frontend/src/views.test.ts` (nuevo), `tareas/T66.md` (sin trackear, previo) |

### Desviaciones

- **Clave del reproductor con proveedor aunque el `episodeID` ya lo lleve**: redundante pero inocuo (misma función para guardar y recuperar); se añadió por coherencia con la del detalle.
- **El restaurado del detalle se recorta a 0 si la lista no está pintada** (típicamente tras cambiar de proveedor y forzar un refetch). Previo a T66; no se tocó `App.svelte`, `stores.ts` ni `VistaDetalle.svelte` (fuera de «Puedes tocar»). Se documenta como observación, no como requisito de T66.
- **T66–T67 originales descartadas**: el paginado/ventana client-side no aporta nada según las mediciones de T65 (366–1180 tarjetas sin jank y con 0 saltos de scroll; el coste real de JKAnime es de **red**, ya cubierto por el mostrado por lotes). La ventana de `ventana.ts` (T64) queda disponible por si se revierte la decisión.

### Pendientes

- **T68 opcional, a reevaluar**; T69 de la serie v1.3.0 pendiente. **Ningún pendiente propio** de T66.
- Sin tag ni release.


## T67 — Restaurar el scroll del detalle tras la recarga por lotes

Hecha el **2026-10-16**. Cuarta tarea de la serie **T64–T69** (v1.3.0, rama `feature/episodios-incrementales`). **Solo frontend**: `frontend/src/VistaDetalle.svelte`. Ningún Go.

Contexto: cierra la observación que dejó T66. Al volver a un detalle cuyos stores **no** están intactos (típicamente tras cambiar de proveedor o reabrir el anime), `openAnime` vacía la lista de episodios y el `scrollTo` de `App.svelte` corre tras `await tick()` **antes** de que la lista esté pintada (altura = viewport), así que la posición guardada se recortaba a **0**. La lista luego crece por lotes y el usuario se quedaba arriba.

### Cambio

- **Restauración pendiente** (`scrollPendiente`, `0` = desarmado): en `onMount` se lee `recuperarScroll(claveScroll('detalle'))` **antes** de decidir la rama. La rama "stores intactos" (vuelta del reproductor) no se toca; en la rama de recarga, si el objetivo es `> 0` se arma la restauración antes de `openAnime`.
- **Reintento por cambio de lista**: `$: intentarRestaurarScroll(episodios.length, $episodesComplete)`. En cada lote (y al completar) mide dentro de un `requestAnimationFrame`; si `document.documentElement.scrollHeight - window.innerHeight >= scrollPendiente` aplica `scrollTo(0, scrollPendiente)` y desarma; si el stream **terminó** y no alcanza, baja al máximo posible y desarma (sin bucle).
- **Cancelación por gesto**: `armarRestauracionScroll` registra oyentes `wheel`/`touchstart`/`keydown`/`pointerdown` (todos `{ passive: true }`) que desarman al primer gesto del usuario; se limpian siempre (`quitarOyentesScroll`) y también en `onDestroy`.
- **Sin cambios** en `views.ts`, `stores.ts`, `App.svelte`, `api.ts`, `TarjetaEpisodio.svelte` ni ningún `.go`.

### Verificación (Playwright, `localhost:34115`; 6/6)

1. **Recarga por lotes con objetivo 800** (JKAnime "bleach" en frío, lotes de 16): bajar a 800, salir, abrir otro anime (evictor) y reabrir ⇒ restauración aplicada en el **2.º lote** (t≈1010 ms, altura 1951), `finalY=800` estable hasta 366 episodios. Con detalle **cacheado** (hit instantáneo) también `finalY=800`.
2. **Interrupción por gesto**: Dragon Ball Z (Original) en frío, `page.mouse.wheel(0,150)` a ~300 ms ⇒ `finalY=0`, `maxY=0` (no fuerza scroll; la restauración se canceló).
3. **Detalle → reproductor → "← Atrás"** (stores intactos): JKAnime "bleach" conserva `scrollY=800` (rama intacta, sin restauración).
4. **Objetivo inalcanzable**: objetivo 11007 con viewport 1280×2000 (max 9649) ⇒ `finalY=9644` (≈máximo), sin bucle ni error.
5. **Primera apertura (objetivo 0)**: Fairy Tail JKAnime ⇒ `finalY=0`, solo el `scrollTo(0,0)` de `App.svelte`; sin restauración ni oyentes.
6. **Consola**: sin errores nuevos de nuestro código (solo `favicon.ico` 404, `TypeError` de `wails/ipc.js` y `ERR_CONNECTION_REFUSED` del proxy local, todos preexistentes y documentados).

### Compuerta (resultado real)

| Comando | Resultado |
|---|---|
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (aviso de chunk >500 kB preexistente) |
| `views.test.ts` (esbuild + node) | **OK (13 comprobaciones)** — sin cambios |
| `ventana.test.ts` (esbuild + node) | **OK (48 comprobaciones)** — sin cambios |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (Go intacto) |
| recuento `^--- PASS` (`-count=1 -v`) | **435** — idéntico a T63/T65/T66 (T67 no toca Go) |
| `git status --porcelain` | `frontend/src/VistaDetalle.svelte` (M) y `tareas/T67.md` (sin trackear, previo) |

### Desviaciones

- **La caché de detalle del backend (TTL 10 min) hace que la 2.ª apertura no cargue por lotes** (hit instantáneo). Para poder probar "recarga por lotes con objetivo>0" se usó un anime **frío** y un **evictor** (abrir otro anime borra la entrada incompleta, `soltarStreamDistinto`); con detalle ya cacheado la restauración igualmente se aplica (la lista llega completa de golpe).
- **`browser_click` de Playwright auto-scrollea el elemento a la vista**, lo que rompe el `scrollY` a guardar; se interactuó con `element.click()` vía `browser_evaluate`/`browser_run_code_unsafe`.
- **Objetivo inalcanzable simulado con `resize` del viewport** (1280×2000) porque en pantalla normal el máximo siempre supera 800; confirma el camino "completo y no alcanza ⇒ máximo".
- **Interrupción simulada con `page.mouse.wheel`** (no hay forma de teclear/scroll nativo fiable en el navegador de Playwright sin alterar la posición).
- **Reiniciar el contenedor recarga la página** (pierde el estado JS y el scroll guardado), así que no sirve para conservar el scroll entre pruebas; se documenta para futuras sesiones.

### Pendientes

- **Ninguno propio.** Resuelve la observación de T66 ("el restaurado se recorta a 0 si la lista no está pintada").
- **T68 opcional, a reevaluar**; **T69** de la serie v1.3.0 pendiente.
- Sin tag ni release.
