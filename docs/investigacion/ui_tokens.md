# Tokens de diseño (T70, v1.4.0)

> Sistema de tokens del rediseño móvil, definido en el bloque `:root` añadido a
> `frontend/src/style.css` (T70). Es la **única fuente de verdad** de la paleta, la tipografía, el
> espacio, los radios, las sombras y el movimiento: desde **T72b** ningún componente escribe
> colores, radios ni sombras literales, y desde **T89** tampoco quedan variables legadas.
> Fecha: **2026-10-17** (actualizado en **T89**, 2026-10-20).

## Estado de las variables legadas (`--cad-*`)

**RETIRADAS en T89.** Las 8 variables `--cad-*` (`--cad-fondo`, `--cad-superficie`, `--cad-borde`,
`--cad-texto`, `--cad-texto-suave`, `--cad-acento`, `--cad-peligro`, `--cad-radio`) se definieron en
T48 y dejaron de usarse en **T72b** (el puente de color); el `grep` de T89 dio **0 usos** fuera de
sus propias definiciones, así que se borraron del `style.css` junto con el `@font-face "Nunito"` y
los archivos de `frontend/src/assets/fonts/`. Eran el "T93" previsto en este documento: se anticipó
a T89 al comprobar que ya no quedaban usos.

Las **tipografías** que sí se usan (Inter y Outfit) llegan de los paquetes npm
`@fontsource-variable/inter` y `@fontsource-variable/outfit` (ambos **SIL OFL 1.1**; el `LICENSE`
completo va en cada paquete). El `@font-face` con el subconjunto **latin** y peso variable se declara
a mano en `style.css` (los paquetes no traen CSS por subconjunto: importar `wght.css` metería todos
los subsets, ~259 KB).

## Regla de uso

**Desde T72b, ningún componente escribe colores, radios ni sombras literales:** todo
sale de estos tokens. Los valores de este documento son la única referencia válida.

## Breakpoints oficiales

Las variables CSS **no valen dentro de `@media`** (el preprocesador del navegador no
las resuelve ahí), así que van como números documentados y repetidos en las consultas:

| Breakpoint | Rol |
|---|---|
| **480 px** | `--gutter: 20px` |
| **768 px** | `--gutter: 24px` |
| **1024 px** | `--gutter: 32px` |

A partir de 768 px se recupera `--blur-lg: 28px` (por debajo, `14px`).

## Tokens

| Token | Valor | Uso previsto |
|---|---|---|
| `--bg-0` | `#090d16` | Fondo base de la app |
| `--bg-1` | `#0f172a` | Fondo elevado (secciones, barras) |
| `--bg-2` | `#1e293b` | Fondo de realce (chips/hover) |
| `--surface-glass` | `rgba(15,23,42,.60)` | Superficie glass de paneles |
| `--surface-card` | `rgba(16,23,41,.55)` | Fondo de tarjeta |
| `--surface-panel` | `rgba(13,20,36,.75)` | Panel emergente/drawer |
| `--surface-input` | `rgba(18,26,45,.70)` | Campos de texto |
| `--surface-chip` | `rgba(255,255,255,.04)` | Chip en reposo |
| `--surface-chip-hover` | `rgba(255,255,255,.09)` | Chip en hover |
| `--border-subtle` | `rgba(255,255,255,.06)` | Borde casi invisible |
| `--border` | `rgba(255,255,255,.08)` | Borde por defecto |
| `--border-strong` | `rgba(255,255,255,.20)` | Borde marcado |
| `--border-accent` | `rgba(52,211,153,.35)` | Borde de acento (focus/activo) |
| `--text-1` | `#f1f5f9` | Texto principal |
| `--text-2` | `#cbd5e1` | Texto secundario |
| `--text-3` | `#94a3b8` | Texto terciario/metadatos |
| `--text-4` | `#64748b` | **Solo decorativo**; no cumple AA en texto pequeño |
| `--brand-400` | `#34d399` | Acento claro (iconos, foco) |
| `--brand-500` | `#10b981` | Acento principal |
| `--brand-600` | `#059669` | Acento profundo (pressed) |
| `--brand-glow` | `#00f2a6` | Acento neón para glow |
| `--brand-teal` | `#2dd4bf` | Acento teal complementario |
| `--on-brand` | `#020617` | Texto sobre superficies de marca |
| `--grad-brand` | `linear-gradient(90deg,var(--brand-500),var(--brand-teal))` | Degradado de marca |
| `--grad-chip-active` | `linear-gradient(135deg,var(--brand-400),var(--brand-500))` | Chip activo |
| `--info` | `#06b6d4` | Informativo |
| `--alert` | `#f43f5e` | Aviso |
| `--danger` | `#f87171` | Error/peligro |
| `--fav` | `#fbbf24` | Favorito (estrella) |
| `--indigo` | `#6366f1` | Acento indigo auxiliar |
| `--scrim` | `rgba(0,0,0,.65)` | Velo de superposición (overlay de carga) |
| `--scrim-strong` | `rgba(0,0,0,.75)` | Velo fuerte (fondo del cajón) |
| `--r-sm` | `6px` | Radio pequeño (chips) |
| `--r-md` | `12px` | Radio medio (inputs, botones) |
| `--r-lg` | `16px` | Radio grande (tarjetas) |
| `--r-xl` | `24px` | Radio extra (paneles) |
| `--r-full` | `9999px` | Píldora/círculo |
| `--sh-card` | `0 8px 32px rgba(0,0,0,.37)` | Sombra de tarjeta |
| `--sh-panel` | `0 20px 50px rgba(0,0,0,.5), inset 0 1px 0 rgba(255,255,255,.10)` | Sombra de panel |
| `--sh-glow` | `0 0 20px -3px rgba(16,185,129,.45)` | Glow de acento |
| `--sh-card-hover` | `0 16px 36px -10px rgba(0,0,0,.7), 0 0 20px -3px rgba(16,185,129,.25)` | Sombra de tarjeta en hover |
| `--sh-player` | `0 20px 50px rgba(0,0,0,.8)` | Sombra del reproductor |
| `--ring` | `0 0 0 2px var(--bg-0), 0 0 0 4px var(--brand-400)` | Anillo de foco |
| `--frame-poster` | `linear-gradient(to bottom,rgba(255,255,255,.20),rgba(255,255,255,.05),rgba(255,255,255,.10))` | Marco de póster |
| `--blur-sm` | `10px` | Blur ligero |
| `--blur-md` | `14px` | Blur medio |
| `--blur-lg` | `28px` | Blur grande (**14px** por debajo de 768 px) |
| `--ease-out` | `cubic-bezier(.16,1,.3,1)` | Curva de entrada |
| `--ease-std` | `cubic-bezier(.4,0,.2,1)` | Curva estándar |
| `--dur-fast` | `150ms` | Duración rápida |
| `--dur` | `250ms` | Duración por defecto |
| `--dur-slow` | `350ms` | Duración lenta |
| `--sp-1` | `4px` | Escala de espacio (múltiplos de 4) |
| `--sp-2` | `8px` | — |
| `--sp-3` | `12px` | — |
| `--sp-4` | `16px` | — |
| `--sp-5` | `20px` | — |
| `--sp-6` | `24px` | — |
| `--sp-7` | `28px` | — |
| `--sp-8` | `32px` | — |
| `--tap` | `44px` | Objetivo táctil mínimo |
| `--gutter` | `16px` | Margen lateral de página (**20/24/32** en 480/768/1024) |
| `--z-base` | `0` | Capa base |
| `--z-sticky` | `20` | Cabecera pegajosa |
| `--z-bar` | `30` | Barra lateral |
| `--z-drawer` | `40` | Cajón deslizante |
| `--z-toast` | `50` | Avisos emergentes |

## Contraste (WCAG)

Generado por `docs/investigacion/ui/contraste.mjs` (`node docs/investigacion/ui/contraste.mjs`):

| frente | fondo | hex frente | hex fondo | relación | AA (≥ 4.5:1) |
|---|---|---|---|---:|:--:|
| `--text-1` | `--bg-0` | `#f1f5f9` | `#090d16` | 17.74:1 | sí |
| `--text-2` | `--bg-0` | `#cbd5e1` | `#090d16` | 13.09:1 | sí |
| `--text-3` | `--bg-0` | `#94a3b8` | `#090d16` | 7.58:1 | sí |
| `--text-4` | `--bg-0` | `#64748b` | `#090d16` | 4.08:1 | NO |
| `--brand-400` | `--bg-0` | `#34d399` | `#090d16` | 10.11:1 | sí |
| `--on-brand` | `--brand-500` | `#020617` | `#10b981` | 7.95:1 | sí |
| `--danger` | `--bg-0` | `#f87171` | `#090d16` | 7.02:1 | sí |

`--text-3` (7.58:1) y `--brand-400` (10.11:1) **pasan** AA. El único que no pasa es
`--text-4` (4.08:1), **previsto**: es el token decorativo y no debe usarse como texto
pequeño (el propio `style.css` lo marca con el comentario obligatorio). No se ha
cambiado su valor.
