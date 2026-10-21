# T88 — Accesibilidad AA (v1.4.0)

Fecha: **2026-10-20**. Rama: `feature/rediseno-movil`. Barrido AA de Home, búsqueda con
resultados, detalle (AnimeAV1 y JKAnime), reproductor y cajón a **360×740** y **1280×800**.
Cambios mínimos de markup/estilo; **sin lógica nueva, sin rediseño, sin tocar `*.go` ni
`wailsjs/**`**.

## Herramienta

`docs/investigacion/ui/a11y_auditoria.mjs` (nuevo) usa el Chromium de Playwright global:

```sh
NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" \
  node docs/investigacion/ui/a11y_auditoria.mjs antes|despues
```

Recorre las 11 combinaciones vista×ancho y calcula, sobre el DOM real:

- **contraste** con `getComputedStyle`, alfa compuesto de fondos y texto (incluye
  `background-image` de gradientes), umbral 4,5:1 / 3:1 para texto grande;
- **nombres accesibles** de `button`, `a`, `input`, `select`, `textarea`, `iframe` y roles
  interactivos;
- **encabezados** (nivel, texto, tamaño, peso y alto de caja);
- **ARIA** de anuncios, estado y controles;
- **foco**: recorre con `Tab` hasta 45 paradas por vista y comprueba que el anillo de marca
  (`rgb(52, 211, 153)`) está presente, también cuando lo dibuja un ancestro
  (`:focus-within`);
- **cajón**: foco inicial, `Escape`, retorno del foco, fugas con `Tab`;
- **estados de carga**: en un contexto aparte se cuelgan `GetHomeSection`,
  `GetRecentProgress` y `GetAnimeDetails` (parcheando el puente en sitio) para
  comprobar `aria-busy` **en ejecución**, no solo en el markup;
- **geometría** y **deriva de scroll** T66/T67.

Los textos que aún están en opacidad de revelado T65 (< 0,95 acumulada) se excluyen: no son
visibles para el usuario y producían falsos positivos de contraste. Los estados `hover` se
muestrean sobre `.btn`, `.tarjeta`, destinos, pestañas, chips, orden, flechas, toasts y
botones del cajón; el estado `focus` se comprueba con el recorrido de teclado y una pasada de
contraste con el foco puesto. No hizo falta axe-core.

## Resultado global

| comprobación | 360 | 1280 |
|---|---:|---:|
| Vistas auditadas | 6 | 5 |
| Fallos de contraste (normal) | **0** | **0** |
| Fallos de contraste con foco | **0** | **0** |
| Fallos de contraste en hover | **0** | **0** |
| Interactivos sin nombre | **0** | **0** |
| Vistas con exactamente una `h1` | **6 / 6** | **5 / 5** |
| Saltos de nivel | **0** | **0** |
| Paradas de foco sin anillo de marca | **0** | **0** |
| `overflowH` | **0** | **0** |

Informes crudos: `docs/img/v1.4/a11y/a11y_antes.json` (baseline, con los falsos positivos
señalados abajo) y `a11y_despues.json` (resultado final). Capturas en
`docs/img/v1.4/a11y/*.png`: son del estado **después** (las dos pasadas comparten nombre de
archivo, así que la de "antes" se sobrescribe).

## Hallazgos y cambios

| # | Regla | Elemento / vista | Antes | Después |
|---|---|---|---|---|
| 1 | 1.3.1 / 2.4.1 | No había `h1` en ninguna vista | `h1 = 0` en las 11 vistas | `h1 = 1` en las 11 |
| 2 | 1.3.1 | `CabeceraVista` (detalle y reproductor) | Título en `h2` | `nivel={1}`; `svelte:element` dinámico |
| 3 | 1.3.1 | `VistaBusqueda` (Home y búsqueda) | “Buscador de anime” en `h2` | `h1`, mismo tamaño y peso |
| 4 | 1.3.1 | Home: `h1` → fila “Continuar viendo” | `h1` seguida de `h3` | `h2` |
| 5 | 1.3.1 | Detalle: “Episodios (N)” | `h3` tras `h1` | `h2`, mismo tamaño (`18,72 px`) |
| 6 | 1.4.3 | `TarjetaHistorial` en hover (cajón/barra lateral) | `--text-3` sobre `--border-strong`: **3,72:1** (5 combinaciones vista×ancho) | `--surface-chip-hover`: **5,45:1** |
| 7 | 4.1.3 | Esqueletos del Home | `aria-label`, sin `aria-busy` | `aria-busy="true"` en `.fila-fantasma` (2/2 en carga) |
| 8 | 4.1.3 | `EstadoVista` en carga | `role="status"`, sin `aria-busy` | `aria-busy="true"` si `variante="cargando"` (verificado) |
| 9 | 4.1.3 | Barra lateral/cajón cargando | Sin `aria-busy` | `aria-busy` en `.cuerpo` mientras carga (verificado) |
| 10 | 3.1.2 | Punto de estado “En emisión” | Solo `status === 'En emisión'` | Compara sin tildes ni mayúsculas |
| 11 | 3.1.1 | `<html lang="en">` | Idioma declarado erróneo | **No tocado** (prohibido): deuda reportada |

**Detalle de los encabezados.** Se conserva el aspecto: Home `h1` = **22 px** a 360 y
**24 px** a 1280 (antes `--fs-xl`/`1.5em` del `h2`); detalle/reproductor `h1` = **18 px** a
360 y **24 px** a 1280 (antes `h2` con `--fs-lg`/`1.5em`); “Episodios” `h2` = **18,72 px**
(antes `h3` por defecto). Peso **700** en todos. La jerarquía queda
`h1 → h2 → h3` en Home, `h1 → h2` en detalle y solo `h1` en reproductor.

**Nombres accesibles.** No se encontró ningún interactivo sin nombre; no fue necesario añadir
`aria-label`. Los iconos siguen siendo decorativos (`Icono` aplica `aria-hidden="true"`
cuando no recibe etiqueta).

**Contraste de componentes (≥ 3:1).** El único fallo real fue el hover de `TarjetaHistorial`
(texto secundario sobre `--border-strong`). Se corrigió con `--surface-chip-hover` en el
componente, sin cambiar ningún token global. El anillo de foco (`--brand-400`,
`rgb(52, 211, 153)`) da **9,29:1** sobre `--bg-1` y **7,61:1** sobre `--surface-2`, muy por
encima del 3:1. Ningún indicador depende solo del color: el punto “En emisión” acompaña
siempre al texto, los destinos activos llevan `aria-current="page"` y los chips y el selector
de orden llevan `aria-pressed`. `--text-4` no se usa en ningún texto legible
(`grep -rn "var(--text-4)" frontend/src` sin resultados).

## Foco y teclado

- **0** elementos sin anillo de marca en las 11 vistas (muestra de hasta 45 paradas por
  vista, incluidos todos los controles anteriores a la lista de episodios; la regla
  `:focus-visible` es compartida por las tarjetas).
- Orden lógico: buscador → acciones → filas/tarjetas → detalle → episodios → reproductor.
- El cajón **entra** con el foco en “Cerrar barra”, `Escape` lo cierra y devuelve el foco a
  “Historial” (`focoDevuelto = true`). **No atrapa el foco**: tras recorrer sus controles,
  `Tab` alcanza el buscador y la barra inferior del fondo (**2 fugas** en 12 pulsaciones).
  La ficha pedía reportarlo y no rediseñar, así que **se deja como deuda**.
- No se añadieron trampas de foco.

## Anuncios y estados

| comprobación | resultado |
|---|---|
| Toast real (borrar + Deshacer) | `role="status"`, `aria-live="polite"`, mensaje y acciones correctos |
| Contador de resultados | `aria-live` presente en búsqueda |
| Barra inferior | `aria-current="page"` presente en Home a 360 |
| Chips / orden | `aria-pressed` presente (detalle: 2 controles; reproductor: 1) |
| Esqueletos del Home (en carga) | **2 / 2** `.fila-fantasma` con `aria-busy="true"` |
| Barra lateral (en carga) | `.cuerpo` con `aria-busy="true"` |
| `EstadoVista` de detalle (en carga) | «Cargando detalle…» con `role="status"` y `aria-busy="true"` |
| Buscador (en carga) | `aria-busy` ya existente en el campo |
| Botón “Siguiente” | `aria-busy` durante la carga (ya existía) |

## Geometría de regresión

| ancho | tarjeta Home | tarjeta episodio | cabecera | CTA | pestaña | `overflowH` |
|---:|---:|---:|---:|---:|---:|---:|
| 360 | **263,8** | **146** | 61 | 61 | — | 0 |
| 768 | **263,8** | **179** | 61 | 61 | — | 0 |
| 1280 | **263,8** | **170,38** | 57 | — | 100,14×38 | 0 |

Sin cambios respecto a T87 (±1 px). T65 sigue revelando las tarjetas y T66/T67 no empeoran
por estos cambios.

## Deudas y notas

- **`<html lang="en">`** en `frontend/index.html`: debe ser `es`. No se toca por la ficha;
  queda para el cierre documental.
- **Cajón sin trap de foco**: el fondo es alcanzable con `Tab` tras el último control. Se
  reporta, no se rediseña.
- **Deriva de scroll del detalle**: con la tarjeta intermedia enfocada y el scroll a
  **2743 px**, al volver del reproductor queda en **2795 px** → **+52 px** (la referencia
  histórica era ~35 px). No se tocó T65/T66/T67.
- El `lang` y el trap son las únicas deudas abiertas de esta ficha; no hay fallos AA de
  contraste, nombres, `h1` ni anillo de foco.
- `docs/img` está ignorado por `.gitignore`, como en las tareas anteriores.
- **Falsos positivos del baseline**: `a11y_antes.json` marca 14 fallos de hover y 27 de foco;
  casi todos son tarjetas de episodio a media opacidad (revelado T65) y no son visibles para
  el usuario. El script los excluye (opacidad acumulada < 0,95) y espera 120 ms a la
  transición del anillo. El único fallo real del baseline es el de `TarjetaHistorial` (3,72:1).

## Verificación

| Comprobación | Resultado |
|---|---|
| `npm run check` | **0 errores / 0 avisos** |
| `npm run build` | **OK** (176 módulos; aviso preexistente de chunk >500 kB) |
| `etiquetas.test.ts` | **32** comprobaciones OK |
| `ventana.test.ts` | **48** comprobaciones OK |
| `views.test.ts` | **13** comprobaciones OK |
| `gofmt -l .` | **sin salida** |
| `go vet ./...` + `go vet -tags integration ./...` | **OK** |
| `go build ./...` | **OK** |
| `go test -race -count=1 -v ./...` | **435 `--- PASS`** de nivel superior, **0 FAIL** |
| `grep -rn wailsjs frontend/src` | solo **`api.ts`** |

Go no se tocó; los 435 PASS se ejecutaron igualmente como compuerta.

## Archivos tocados

`frontend/src/CabeceraVista.svelte`, `VistaDetalle.svelte`, `VistaReproductor.svelte`,
`VistaBusqueda.svelte`, `FilaContinuar.svelte`, `TarjetaHistorial.svelte`,
`VistaInicio.svelte`, `EstadoVista.svelte`, `BarraLateral.svelte`; y
`docs/investigacion/ui/a11y_auditoria.mjs`, `docs/investigacion/ui_a11y.md`,
`docs/img/v1.4/a11y/**`, `PROGRESO.md`.

**No se tocó** `style.css`, `index.html`, `*.go`, `frontend/wailsjs/**`, `api.ts`,
`stores.ts`, `views.ts`, `ui.ts`, `package.json` ni el lockfile. **Sin tag ni release** (los
hace el humano); **T89 no iniciada**.
