# Auditoría base de UI (T69, v1.4.0)

> Fotografía **del estado ya existente** de la interfaz (rama `feature/rediseno-movil`,
> commit `fc34a64`) como punto de partida del rediseño móvil de la serie v1.4.0. Es una
> tarea **de solo lectura sobre el código**: no se ha tocado ni un `.svelte`, ni un `.ts`,
> ni un `.go`, ni `frontend/wailsjs/**`, ni `package.json`, ni `wails.json`. Lo único creado
> son este documento, los scripts de medición en `docs/investigacion/ui/`, las capturas en
> `docs/img/v1.4/base/` y la entrada correspondiente en `PROGRESO.md`.
> Fecha: **2026-10-16**. Todas las cifras salen de scripts reproducibles versionados junto
> a este documento (sección «Método»), no de observaciones manuales.

## Método

- Servidor de desarrollo real vía Docker: `docker compose up -d`
  (`xvfb-run wails dev -browser=false -devserver 0.0.0.0:34115`); la app se abre en el
  navegador de Playwright en **`http://localhost:34115`** (el servidor de Wails, no el
  `:5173` de Vite) con datos reales de JKAnime y AnimeAV1.
- Medición y capturas con Playwright (Chromium) desde Node, usando la copia global del
  paquete del MCP de Playwright: `NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" node <script>`.
- Conteo de literales CSS sobre los `.svelte` y `style.css` **sin renderizar** (regex sobre
  el texto fuente).
- Estado persistido (historial y favoritos) **de prueba**: para que la barra lateral tuviera
  contenido se escribió un `historial.json` de ejemplo en `/root/.config/miapp/` dentro del
  contenedor (el `<video>` no avanza en el navegador, así que el reproductor no genera
  historial real). Las portadas del historial sembrado van vacías → las tarjetas muestran el
  **respaldo con inicial** («B», «O»), que es también un estado a documentar. El contador de
  favoritos aparece en **(1)** porque el propio guion marca un favorito durante la captura
  de `favoritos_datos`.
- Scripts (en `docs/investigacion/ui/`):
  - `contar_literales.mjs` → tabla de literales CSS y colores hex por archivo.
  - `medir_ui.mjs` → capturas `<vista>_<ancho>.png` + volcado de métricas a `metricas.json`.
  - `resumir_metricas.mjs` → tabla markdown de métricas y listado de objetivos táctiles.

---

## 1. Inventario del frontend

### Componentes (`.svelte`, 3 895 líneas)

| Archivo | Líneas | Rol |
|---|---:|---|
| `App.svelte` | 170 | Raíz: cabecera, buscador, pestañas Descubrir/Favoritos, barra lateral, montaje de vistas. |
| `BarraLateral.svelte` | 575 | Cajón/panel de historial («Continuar viendo») y navegación lateral; colapsable. |
| `VistaInicio.svelte` | 334 | Home por proveedor (filas «En emisión» / «Recién agregados»). |
| `VistaBusqueda.svelte` | 192 | Rejilla de resultados de búsqueda. |
| `VistaDetalle.svelte` | 452 | Ficha del anime + rejilla de episodios. |
| `VistaReproductor.svelte` | 818 | Reproducción (HLS/MP4/iframe), selector de fuentes, cola de episodios, botón «Siguiente». |
| `FilaAnimes.svelte` | 205 | Fila con carrusel horizontal de tarjetas (Home). |
| `TarjetaAnime.svelte` | 113 | Tarjeta de portada en rejillas/filas. |
| `TarjetaEpisodio.svelte` | 164 | Tarjeta de episodio (altura fija ~170 px, ver §4). |
| `TarjetaHistorial.svelte` | 244 | Fila del historial en la barra lateral. |
| `GridFavoritos.svelte` | 93 | Rejilla de favoritos. |
| `SelectorFuente.svelte` | 155 | Selector servidor·idioma. |
| `BotonFavorito.svelte` | 142 | Estrella de favorito. |
| `BotonSiguiente.svelte` | 103 | Botón «Siguiente ▸» (reproductor). |
| `Toast.svelte` | 93 | Avisos emergentes. |
| `style.css` | 42 | Variables globales, `@font-face` y resets. |

### Módulos TypeScript (`.ts`, ~1 900 líneas incl. tests)

| Archivo | Líneas | Rol |
|---|---:|---|
| `api.ts` | 223 | **Único** punto de contacto con `frontend/wailsjs/**` (regla P4). |
| `stores.ts` | 218 | Estado global (Svelte stores) y suscripción a eventos del backend. |
| `historial.ts` | 217 | Lógica del historial (agrupado por proveedor). |
| `deslizar.ts` | 217 | Gesto de deslizar (cajón lateral). |
| `etiquetas.ts` | 175 | Etiquetas/traducciones de tipos de episodio. |
| `revelar.ts` | 118 | Revelado animado de tarjetas por visibilidad (T65). |
| `retomar.ts` | 101 | Cálculo de «retomar». |
| `favoritos.ts` | 84 | Lógica de favoritos. |
| `views.ts` | 79 | Clave de scroll por proveedor+slug (T66). |
| `ventana.ts` | 70 | Lógica pura de ventana incremental (T64). |
| `home.ts` | 60 | Carga de secciones del Home. |
| `toasts.ts` | 53 | Cola de avisos. |
| `ui.ts` | 47 | Utilidades de UI. |
| `main.ts` | 7 | Arranque de Svelte. |
| `*.test.ts` | 399 | `etiquetas` (193), `ventana` (115), `views` (91) — lógica pura, sin DOM. |

### Recursos

| Recurso | Ruta |
|---|---|
| Fuente UI (Google Fonts, v16 latin) | `frontend/src/assets/fonts/nunito-v16-latin-regular.woff2` (+ `OFL.txt`) |
| Logotipo | `frontend/src/assets/images/logo-universal.png` |

La fuente se declara con `@font-face "Nunito"` en `style.css`. **No** hay carpeta de iconos:
los símbolos visibles son texto/emoji (`★`, `▸`, `←`, `⏸`, etc.).

---

## 2. Sistema visual actual (`style.css`)

Variables globales (tema oscuro único):

| Variable | Valor | Uso |
|---|---|---|
| `--cad-fondo` | `#151515` | Fondo base |
| `--cad-superficie` | `#1e1e24` | Superficies/cards |
| `--cad-borde` | `#3f3f46` | Bordes |
| `--cad-texto` | `#ffffff` | Texto principal |
| `--cad-texto-suave` | `#a1a1aa` | Texto secundario |
| `--cad-acento` | `#4ade80` | Verde de acento (botones, progreso, activo) |
| `--cad-peligro` | `#f87171` | Errores/peligro |
| `--cad-radio` | `8px` | Radio por defecto |

**Colores hex hardcodeados más repetidos** (fuera de las variables): `#4ade80` (16 veces
— el acento se repite a mano), `#2a2a35` (12), `#a1a1aa` (10), `#111` (8), `#1e1e24` (5),
`#f87171` (5), `#3f3f46` (4). Es decir: el acento y varias superficies aparecen **tanto como
variable global como literal suelto** en los componentes → no hay una única fuente de verdad
de color.

---

## 3. Tabla de literales CSS (análisis estático)

Salida de `contar_literales.mjs` (conteo de patrones sobre el fuente; **no** cuenta lo que
Tailwind no aplica porque el proyecto no usa Tailwind):

| archivo | líneas | hex | rgb(a) | hsl | box-shadow | border-radius | backdrop-filter | blur() | ancho/alto px | :hover | @media |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| App.svelte | 170 | 1 | 0 | 0 | 0 | 2 | 0 | 0 | 1 | 0 | 0 |
| BarraLateral.svelte | 575 | 0 | 2 | 0 | 0 | 8 | 0 | 0 | 8 | 2 | 2 |
| BotonFavorito.svelte | 142 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 0 | 1 | 0 |
| BotonSiguiente.svelte | 103 | 1 | 2 | 0 | 2 | 1 | 0 | 1 | 0 | 2 | 2 |
| FilaAnimes.svelte | 205 | 0 | 0 | 0 | 0 | 1 | 0 | 0 | 2 | 1 | 0 |
| GridFavoritos.svelte | 93 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 0 | 0 | 0 |
| SelectorFuente.svelte | 155 | 13 | 0 | 0 | 0 | 2 | 0 | 0 | 0 | 2 | 0 |
| TarjetaAnime.svelte | 113 | 1 | 1 | 0 | 2 | 2 | 0 | 0 | 2 | 2 | 1 |
| TarjetaEpisodio.svelte | 164 | 13 | 1 | 0 | 0 | 4 | 0 | 0 | 2 | 1 | 0 |
| TarjetaHistorial.svelte | 244 | 0 | 0 | 0 | 0 | 4 | 0 | 0 | 3 | 3 | 1 |
| Toast.svelte | 93 | 0 | 0 | 0 | 0 | 2 | 0 | 0 | 1 | 3 | 0 |
| VistaBusqueda.svelte | 192 | 10 | 0 | 0 | 0 | 3 | 0 | 0 | 2 | 1 | 0 |
| VistaDetalle.svelte | 452 | 17 | 0 | 0 | 0 | 6 | 0 | 0 | 4 | 0 | 2 |
| VistaInicio.svelte | 334 | 3 | 1 | 0 | 0 | 4 | 0 | 0 | 4 | 1 | 1 |
| VistaReproductor.svelte | 818 | 15 | 1 | 0 | 0 | 5 | 0 | 0 | 2 | 0 | 0 |
| style.css | 42 | 8 | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| **TOTAL** | **3 895** | **82** | **9** | 0 | **4** | **47** | 0 | **1** | **31** | **19** | **9** |

### Breakpoints

- **Solo hay un `@media` con ancho en px**: `max-width: 640px` (1 vez, en `VistaDetalle`).
  Todo lo demás «responsive» se apoya en rejillas `auto-fit`/`flex` y no en media queries.
- Los otros 8 `@media` son de preferencias de usuario, no de tamaño:
  `prefers-reduced-motion` (×6, repartidos en `BarraLateral`, `BotonSiguiente`, `TarjetaAnime`,
  `VistaDetalle`, `VistaInicio`) y `@media (hover: none)` (×1, en `TarjetaHistorial`, para
  ocultar acciones en táctil).
- No hay `backdrop-filter` ni `hsl()` en todo el frontend: **no hay glassmorphism** en el
  código actual (4 `box-shadow` sueltos y 1 `blur()` aislado). El `backdrop-blur` del diseño
  de referencia (Stitch, §6) no existe hoy.

---

## 4. Métricas responsive medidas (navegador real)

Salida de `medir_ui.mjs` → `metricas.json` resumida por `resumir_metricas.mjs`.
Anchos: **360×740**, **768×1024**, **1280×800**.

| vista@ancho | overflowH | táctiles <44 px | texto <12 px | texto <14 px | tarj. episodio | tarj. Home | columnas rejilla | contain-intrinsic |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| descubrir@360 | 0 | 14 | 0 | 108 | — | 263.8 | — | — |
| busqueda@360 | 0 | 3 | 0 | 22 | — | — | 1 | — |
| detalle_animeav1@360 | 0 | 4 | 366 | 372 | 170.39 | — | 1 | auto 170px |
| reproductor_hls@360 | 0 | 6 | 1 | 7 | — | — | — | — |
| reproductor_extras@360 | 0 | 14 | 1 | 15 | — | — | — | — |
| detalle_jkanime@360 | 0 | 4 | 320 | 327 | 170.39 | — | 1 | auto 170px |
| lateral_cajon@360 | 0 | 8 | 366 | 384 | 170.39 | — | 1 | auto 170px |
| descubrir@768 | 0 | 14 | 0 | 108 | — | 263.8 | — | — |
| busqueda@768 | 0 | 3 | 0 | 22 | — | — | 4 | — |
| detalle_animeav1@768 | 0 | 4 | 366 | 372 | 170.39 | — | 3 | auto 170px |
| reproductor_hls@768 | 0 | 7 | 1 | 7 | — | — | — | — |
| reproductor_extras@768 | 0 | 15 | 1 | 15 | — | — | — | — |
| detalle_jkanime@768 | 0 | 4 | 366 | 373 | 170.39 | — | 3 | auto 170px |
| lateral_cajon@768 | 0 | 8 | 366 | 384 | 170.39 | — | 3 | auto 170px |
| descubrir@1280 | 0 | 17 | 0 | 119 | — | 263.8 | — | — |
| favoritos_vacio@1280 | 0 | 8 | 1 | 14 | — | 263.8 | — | — |
| busqueda@1280 | 0 | 6 | 0 | 33 | — | — | 5 | — |
| detalle_animeav1@1280 | 0 | 7 | 366 | 383 | 170.39 | — | 4 | auto 170px |
| reproductor_hls@1280 | 0 | 7 | 1 | 7 | — | — | — | — |
| reproductor_extras@1280 | 0 | 7 | 1 | 7 | — | — | — | — |
| detalle_jkanime@1280 | 0 | 7 | 366 | 384 | 170.39 | — | 4 | auto 170px |
| favoritos_datos@1280 | 0 | 8 | 0 | 12 | — | 263.8 | — | — |
| lateral_contraida@1280 | 0 | 5 | 0 | 1 | — | 263.8 | — | — |
| lateral_expandida@1280 | 0 | 8 | 0 | 12 | — | 263.8 | — | — |

Lecturas clave:

- **Sin desbordamiento horizontal** (`overflowH = 0`) en ninguna vista/ancho: no hay bugs de
  layout que rompan el ancho de página.
- **Rejilla de episodios**: 1 columna a 360, 3 a 768 (AnimeAV1; JKAnime 3), 4 a 1280
  (AnimeAV1) — JKAnime 4. La rejilla de búsqueda: 1 / 4 / 5 columnas.
- **Altura de tarjeta de episodio = 170,39 px**, reforzada con `contain-intrinsic-size:
  auto 170px` (acoplamiento fuerte a reseñas del rediseño, ver §7).
- **Tarjeta de Home = 263,8 px**; el carrusel (`FilaAnimes`) **desborda** a propósito
  (`carruselDesborda: true`) y se navega con flechas.
- **Objetivos táctiles**: 38 combinaciones únicas de `<44 px` (recomendación de accesibilidad).
  Los más problemáticos por vista y repetición:
  - `.pestana` Descubrir/Favoritos: **93×39** / **128×39** (pestañas de la cabecera, presentes
    en casi todas las vistas de escritorio).
  - `.hamburguesa`: **38×38** (abre el cajón; omnipresente en móvil/tablet).
  - `.estrella` (favorito), `.flecha` (carrusel) y `.alternar` (barra lateral): **32×32**.
  - `.papelera` (borrar historial): **24×24**.
  - Botón «Buscar»: **85×38**; «Actualizar»: **92×34**; «← Atrás»: **83×31**.
  - Chips de fuente `.s-z22NnteWGpUK`: alto **31 px**; el plegable
    «Otras fuentes…»: **341×29**.
- **Texto pequeño**: en las rejillas de episodios aparecen ~366 elementos con fuente <12 px
  (metadatos de episodio: número/título/etiquetas), acumulados por la propia rejilla en
  detalle y en el cajón lateral. Es el mayor foco de mejora tipográfica a 360 px.

---

## 5. Capturas (`docs/img/v1.4/base/`, 24 archivos)

Convención `<vista>_<ancho>.png`. `fullPage` para las vistas «altas» (Home, búsqueda,
favoritos); recorte de viewport para detalle/reproductor/lateral (evita páginas de decenas
de miles de píxeles).

| Vista | 360 | 768 | 1280 |
|---|:--:|:--:|:--:|
| Descubrir (Home) | `descubrir_360` | `descubrir_768` | `descubrir_1280` |
| Búsqueda | `busqueda_360` | `busqueda_768` | `busqueda_1280` |
| Detalle AnimeAV1 | `detalle_animeav1_360` | `…_768` | `…_1280` |
| Detalle JKAnime | `detalle_jkanime_360` | `…_768` | `…_1280` |
| Reproductor HLS | `reproductor_hls_360` | `…_768` | `…_1280` |
| Reproductor con extras | `reproductor_extras_360` | `…_768` | `…_1280` |
| Cajón lateral (móvil/tablet) | `lateral_cajon_360` | `lateral_cajon_768` | — |
| Barra lateral contraída | — | — | `lateral_contraida_1280` |
| Barra lateral expandida | — | — | `lateral_expandida_1280` |
| Favoritos vacío | — | — | `favoritos_vacio_1280` |
| Favoritos con datos | — | — | `favoritos_datos_1280` |

Observaciones visuales (de las capturas):

- A 360 px el Home apila 2 tarjetas por fila con carrusel desbordado y muestra el botón
  hamburguesa; las pestañas Descubrir/Favoritos y el buscador quedan muy ajustados.
- El cajón lateral a 360/768 ocupa casi todo el ancho como panel deslizante (≈304 px en el
  cajón de 360) con «Continuar viendo»/«Borrar todo el historial».
- En detalle, la etiqueta de proveedor aparece en minúsculas (`animeav1`, `jkanime`) junto a
  los metadatos; el respaldo de portada del historial sembrado muestra la **inicial**.

---

## 6. Referencia visual objetivo (Stitch) — `docs/investigacion/stitch/`

Cuatro exportes HTML de Stitch (Tailwind por CDN, `dark`), **leídos, no modificados**:
`stitch_home.html`, `stitch_detalle.html`, `stitch_reproductor_1.html`,
`stitch_reproductor_2.html`. Los cuatro comparten lenguaje: tema oscuro casi negro con
**acento esmeralda/neón**, superficies *glass* translúcidas con `backdrop-blur` y bordes
`rgba(255,255,255,0.06–0.08)`, glow esmeralda en botones activos, y tipografía
**Inter** (texto) + **Outfit** (display) / **Noto Serif JP** (créditos kanji) — frente a la
**Nunito** actual.

### Paleta consolidada (valores declarados en los `tailwind.config` de los 4 archivos)

| Token | Home | Detalle | Reproductor 1 | Reproductor 2 |
|---|---|---|---|---|
| Fondo base | `#090d16` (obsidian) | `#090d16` (brand.darkBg) | `#07090e` (body) / `#0a0d12` (dark.900) | `#090a0f` (brand.darkBg) |
| Superficie | `#0f172a` / card `rgba(17,24,39,.65)` | `rgba(15,23,42,.72)` / card `rgba(23,32,54,.65)` | `#11151e` / `#181f2a` | card `rgba(18,20,29,.75)` |
| Borde | `rgba(255,255,255,.08)` | `rgba(255,255,255,.08)` | `rgba(255,255,255,.08)` | `rgba(255,255,255,.08)` |
| Glow | `rgba(16,185,129,.15)` | `rgba(16,185,129,.4)` (hover) | `#00ffaa` | `rgba(34,197,94,.45)` |
| Verde acento | `#10b981` | `#10b981` | `#10b981` | `#10b981` |
| Verde profundo | `#059669` | `#059669` | `#059669` | `#059669` |
| Verde claro | `#34d399` | `#34d399` | `#34d399` / `#00f2a6` | `#34d399` |
| Teal/neón | `#14b8a6` (teal) | — | — | `#22c55e` (neon) |
| Rojo | — | — | — | `#dc2626` (glow-red, cinescope) |

### Tokens CSS reutilizables (clases *glass* de los exportes)

| Clase | Definición (Home) |
|---|---|
| `.glass-panel` | `background: rgba(15,23,42,.6); backdrop-filter: blur(14px); border: 1px solid rgba(255,255,255,.07)` |
| `.glass-input` | `rgba(18,26,45,.7); blur(10px); borde rgba(255,255,255,.09)` |
| `.glass-card` | `rgba(16,23,41,.55); blur(12px); borde rgba(255,255,255,.06); hover: translateY(-6px) scale(1.015) + borde verde + doble sombra` |

Sombras destacadas: `glow-emerald: 0 0 25px -4px rgba(16,185,129,.45)`,
`card-glass: 0 8px 32px 0 rgba(0,0,0,.37)`. En Reproductor 1: animaciones `glow-pulse`
(2.5 s) y `float-slow` (7 s). En Reproductor 2: focos ambientales `radial-gradient` verde
(izquierda) y rojo (centro) y un fondo «cinema-backdrop» rojo para el modo cinescope.

### Duraciones y transiciones (valores declarados)

| Archivo | Valor | Dónde |
|---|---|---|
| Home | `transition: all 0.35s cubic-bezier(0.16, 1, 0.3, 1)` | `.glass-card` (hover `translateY(-6px) scale(1.015)`) |
| Detalle | `transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1)` | `.glass-card-item` (hover `translateY(-4px)`) |
| Detalle | `backdrop-filter: blur(28px) saturate(180%)` | `.glass-panel` |
| Reproductor 1 | `transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1)` | `.glass-hover` (hover `translateY(-2px)`) |
| Reproductor 1 | `glow-pulse 2.5s infinite ease-in-out` · `float-slow 7s ease-in-out infinite` | animaciones |
| Reproductor 2 | `transition-all duration-200/300` | chips y botones (Tailwind) |

> Nota: esta sección **describe** la referencia de Stitch; no implica que el rediseño v1.4.0
> vaya a copiarla tal cual.

---

## 7. Acoplamientos y notas para el rediseño

Cosas a respetar o a coordinar antes de tocar el diseño (para no romper lógica ya probada):

1. **P4 — `frontend/wailsjs/**` solo se importa desde `api.ts`.** Ningún componente debe
   hablar directamente con el backend.
2. **Clases atómicas de Svelte (`s-<hash>`).** Los estilos por componente llevan sufijos
   generados (p. ej. `.s-ztPBer2NOpb-` en las pestañas). Los selectores medidos y cualquier
   test/automatización deben apuntar a **clases semánticas**, no a las `s-…`.
3. **Altura fija de la tarjeta de episodio = `170 px`**, reforzada con
   `contain-intrinsic-size: auto 170px`. Si el rediseño cambia esa altura hay que actualizar
   ambos sitios, o el scroll incremental (T64) y el revelado por visibilidad (T65) medirán
   mal.
4. **Clave de scroll por proveedor+slug** (`views.ts`, T66) y su restauración (T67): el
   rediseño no debe cambiar la identidad con la que se guarda la posición de scroll del
   detalle.
5. **Único breakpoint de ancho hoy: `640 px`** (solo en `VistaDetalle`). El layout responsive
   depende de `auto-fit`/`flex`; si el rediseño móvil introduce nuevos cortes (p. ej. 480,
   768), hay que añadirlos de forma consistente (hoy 360/768/1280 se resuelven sin media
   queries).
6. **Colores duplicados**: `#4ade80` y varias superficies están a la vez en `style.css`
   (variables `--cad-*`) y hardcodeados en componentes. Centralizar en variables es trabajo
   natural del rediseño.
7. **Sin glassmorphism ni iconos** en el código actual (no hay `backdrop-filter`, `hsl`, ni
   set de iconos). No existen componentes reutilizables de «card glass»/«chip» más allá de
   las tarjetas; el rediseño parte prácticamente de cero en ese lenguaje visual.
8. **No tocar** en esta tarea: `frontend/src/**`, `frontend/wailsjs/**`, `*.go`,
   `package.json`, `wails.json`. Los scripts de medición viven en `docs/investigacion/ui/`,
   fuera de `frontend/`, para no activar el watcher de Vite.

---

## 8. Entregables de la tarea

- `docs/investigacion/ui_base.md` (este documento).
- `docs/investigacion/ui/contar_literales.mjs`, `medir_ui.mjs`, `resumir_metricas.mjs` y
  `docs/investigacion/ui/metricas.json` (datos crudos).
- `docs/img/v1.4/base/*.png` (24 capturas).
- Entrada resumen en `PROGRESO.md`.
