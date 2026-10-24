# Mapeo de color a tokens (T72b, v1.4.0)

> Puente de color de la serie v1.4.0 (rama `feature/rediseno-movil`). Pasada
> **mecánica**: solo color, sombra y radio dentro de los `<style>` de los `.svelte`
> existentes. **No** se toca `<script>`, markup, tamaños, márgenes, paddings,
> `display`, `grid`, `flex` ni layout. Lista generada **antes** de editar (grep sobre
> el fuente con los números de línea del estado previo a T72b).
> Referencia de tokens: `docs/investigacion/ui_tokens.md` y el bloque `:root` de
> `frontend/src/style.css` (T70/T71).
> Fecha: **2026-10-17**.

## Reglas aplicadas

### Color

| Literal | Token | Criterio |
|---|---|---|
| `#151515` (y `--cad-fondo`) | `--bg-0` | fondo base |
| `#1e1e24` (y `--cad-superficie`) | `--surface-card` | superficies (tarjetas, paneles, cajas, botones) |
| `#2a2a35` | `--surface-chip` / `--bg-2` / `--surface-input` | ver columna *criterio* de cada fila |
| `#3f3f46` (y `--cad-borde`) | `--border` / `--border-strong` | `--border` si es borde de 1 px; `--border-strong` si es relleno/fondo |
| `#ffffff`, `white` (y `--cad-texto`) | `--text-1` | texto principal |
| `#a1a1aa` (y `--cad-texto-suave`) | `--text-3` | texto terciario/metadatos |
| `#4ade80` (y `--cad-acento`) | `--brand-400` / `--brand-500` | `--brand-400` en texto/iconos/bordes; `--brand-500` en rellenos |
| `#111` | `--on-brand` | texto sobre relleno de marca |
| `#f87171` (y `--cad-peligro`) | `--danger` | error/peligro |
| `#fbbf24` | `--fav` | favorito |
| `#d4d4d8` | `--text-2` | aproximación por luminosidad (gris claro) |
| `#f4f4f5` | `--text-1` | aproximación por luminosidad |
| `#555` | `--bg-2` | aproximación (estado `:disabled`; no hay token de deshabilitado) |
| `#999` | `--text-3` | aproximación (estado `:disabled`) |
| `#000` | `--bg-0` | aproximación (fondo de vídeo/iframe); `--bg-0` es el negro del tema |
| `rgba(255,255,255,0.08)` | `--border` | mismo valor que el token |

### Radio

| Literal | Token | Criterio |
|---|---|---|
| `var(--cad-radio)` (8 px), `8px` | `--r-md` (12 px) | regla del enunciado |
| `10px` | `--r-md` | aproximación: más cerca de 12 que de 6 |
| `12px` | `--r-md` | valor exacto |
| `4px`, `6px` | `--r-sm` (6 px) | radios pequeños (≤ 6 px) |
| `999px`, `50%` | `--r-full` | píldora/círculo |

### Sombra

| Literal | Token | Criterio |
|---|---|---|
| `0 6px 16px rgba(0,0,0,0.45)` (hover de tarjeta) | `--sh-card-hover` | sombra de tarjeta en hover |

## Mapeo por archivo

> Formato: `línea` · `declaración actual` → **token**.

### App.svelte (`<style>` 129–169)

| línea | literal | token |
|---:|---|---|
| 151 | `background: var(--cad-superficie)` | `--surface-card` |
| 152 | `border: 1px solid var(--cad-borde)` | `--border` |
| 153 | `border-radius: var(--cad-radio)` | `--r-md` |
| 154 | `color: var(--cad-texto)` | `--text-1` |
| 158 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 162 | `color: var(--cad-peligro)` | `--danger` |
| 166 | `background: #2a2a35` | `--bg-2` (bloque de aviso) |
| 167 | `border-radius: 8px` | `--r-md` |

### BarraLateral.svelte (`<style>` 301–574)

| línea | literal | token |
|---:|---|---|
| 311 | `background: var(--cad-superficie)` | `--surface-card` |
| 312 | `color: var(--cad-texto)` | `--text-1` |
| 313 | `border-right: 1px solid var(--cad-borde)` | `--border` |
| 326 | `border-bottom: 1px solid var(--cad-borde)` | `--border` |
| 345 | `border: 1px solid var(--cad-borde)` | `--border` |
| 346 | `border-radius: var(--cad-radio)` | `--r-md` |
| 347 | `color: var(--cad-texto)` | `--text-1` |
| 351 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 362 | `color: var(--cad-texto-suave)` | `--text-3` |
| 368 | `border: 1px solid var(--cad-borde)` | `--border` |
| 369 | `border-radius: var(--cad-radio)` | `--r-md` |
| 370 | `color: var(--cad-acento)` | `--brand-400` |
| 376 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 391 | `color: var(--cad-texto-suave)` | `--text-3` |
| 394 | `color: var(--cad-texto-suave)` | `--text-3` |
| 406 | `border: 1px solid var(--cad-borde)` | `--border` |
| 407 | `border-radius: var(--cad-radio)` | `--r-md` |
| 408 | `color: var(--cad-texto-suave)` | `--text-3` |
| 412 | `color: var(--cad-peligro)` | `--danger` |
| 413 | `border-color: var(--cad-peligro)` | `--danger` |
| 416 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 423 | `border-top: 1px solid var(--cad-borde)` | `--border` |
| 430 | `border: 1px solid var(--cad-borde)` | `--border` |
| 431 | `border-radius: var(--cad-radio)` | `--r-md` |
| 432 | `color: var(--cad-texto-suave)` | `--text-3` |
| 438 | `border-color: var(--cad-peligro)` | `--danger` |
| 439 | `color: var(--cad-peligro)` | `--danger` |
| 442 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 448 | `color: var(--cad-texto)` | `--text-1` |
| 460 | `border: 1px solid var(--cad-peligro)` | `--danger` |
| 461 | `border-radius: var(--cad-radio)` | `--r-md` |
| 462 | `color: var(--cad-peligro)` | `--danger` |
| 473 | `border: 1px solid var(--cad-borde)` | `--border` |
| 474 | `border-radius: var(--cad-radio)` | `--r-md` |
| 475 | `color: var(--cad-texto)` | `--text-1` |
| 481 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 498 | `border-radius: var(--cad-radio)` | `--r-md` |
| 499 | `background: var(--cad-borde)` | `--border-strong` (relleno) |
| 511 | `border-radius: 4px` | `--r-sm` |
| 512 | `background: var(--cad-borde)` | `--border-strong` (relleno) |
| 563 | `background: rgba(0, 0, 0, 0.75)` | **excepción** (scrim del cajón) |
| 566 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |

### BotonFavorito.svelte (`<style>` 96–141)

| línea | literal | token |
|---:|---|---|
| 110 | `border-radius: var(--cad-radio)` | `--r-md` |
| 111 | `color: var(--cad-texto-suave)` | `--text-3` |
| 121 | `color: var(--cad-acento)` | `--brand-400` |
| 127 | `color: var(--cad-acento)` | `--brand-400` |
| 134 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 139 | `color: var(--cad-peligro)` | `--danger` |

### BotonSiguiente.svelte (`<style>` 36–102)

| línea | literal | token |
|---:|---|---|
| 45 | `border-radius: var(--cad-radio)` | `--r-md` |
| 46 | `border: 1px solid var(--cad-borde)` | `--border` |
| 47 | `background: var(--cad-superficie)` | `--surface-card` |
| 48 | `color: var(--cad-texto)` | `--text-1` |
| 59 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 63 | `border-color: var(--cad-acento)` | `--brand-400` |
| 84 | `border-color: var(--cad-acento)` | `--brand-400` |
| 85 | `background: var(--cad-acento)` | `--brand-500` (relleno) |
| 86 | `color: #111` | `--on-brand` |
| 96 | `box-shadow: 0 0 0 0 rgba(74, 222, 128, 0.55)` | **excepción** (keyframe `pulso-siguiente`) |
| 99 | `box-shadow: 0 0 0 6px rgba(74, 222, 128, 0)` | **excepción** (keyframe `pulso-siguiente`) |

### FilaAnimes.svelte (`<style>` 110–204)

| línea | literal | token |
|---:|---|---|
| 123 | `color: var(--cad-texto-suave)` | `--text-3` |
| 129 | `color: var(--cad-texto)` | `--text-1` |
| 144 | `background: var(--cad-superficie)` | `--surface-card` |
| 145 | `border: 1px solid var(--cad-borde)` | `--border` |
| 146 | `border-radius: 50%` | `--r-full` |
| 147 | `color: var(--cad-texto)` | `--text-1` |
| 151 | `border-color: var(--cad-acento)` | `--brand-400` |
| 152 | `color: var(--cad-acento)` | `--brand-400` |
| 159 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 185, 186, 189, 190, 194, 195, 197, 198 | `black` en `mask-image` | **excepción** (máscara de degradado, no es color del tema) |

### GridFavoritos.svelte (`<style>` 53–92)

| línea | literal | token |
|---:|---|---|
| 63 | `background: var(--cad-superficie)` | `--surface-card` |
| 64 | `border: 1px solid var(--cad-borde)` | `--border` |
| 65 | `border-radius: var(--cad-radio)` | `--r-md` |
| 70 | `color: var(--cad-texto)` | `--text-1` |
| 76 | `border: 1px solid var(--cad-borde)` | `--border` |
| 77 | `border-radius: var(--cad-radio)` | `--r-md` |
| 78 | `color: var(--cad-acento)` | `--brand-400` |
| 84 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 89 | `color: var(--cad-texto-suave)` | `--text-3` |

### SelectorFuente.svelte (`<style>` 89–154)

| línea | literal | token |
|---:|---|---|
| 95 | `color: #a1a1aa` | `--text-3` |
| 106 | `border-radius: 999px` | `--r-full` |
| 107 | `border: 1px solid #3f3f46` | `--border` |
| 108 | `background: #1e1e24` | `--surface-card` |
| 109 | `color: #d4d4d8` | `--text-2` |
| 114 | `border-color: #4ade80` | `--brand-400` |
| 117 | `outline: 2px solid #4ade80` | `--brand-400` |
| 121 | `background: #4ade80` | `--brand-500` (relleno) |
| 122 | `border-color: #4ade80` | `--brand-400` |
| 123 | `color: #111` | `--on-brand` |
| 132 | `border-radius: 8px` | `--r-md` |
| 135 | `color: #a1a1aa` | `--text-3` |
| 141 | `color: #d4d4d8` | `--text-2` |
| 142 | `border-color: #3f3f46` | `--border` |
| 145 | `outline: 2px solid #4ade80` | `--brand-400` |

### TarjetaAnime.svelte (`<style>` 32–112)

| línea | literal | token |
|---:|---|---|
| 39 | `border-radius: 10px` | `--r-md` |
| 41 | `background: var(--cad-superficie)` | `--surface-card` |
| 42 | `color: var(--cad-texto)` | `--text-1` |
| 56 | `border-color: var(--cad-acento)` | `--brand-400` |
| 57 | `box-shadow: 0 6px 16px rgba(0, 0, 0, 0.45)` | `--sh-card-hover` |
| 60 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 68 | `background: #2a2a35` | `--bg-2` (fondo tras la portada) |
| 77 | `background: var(--cad-fondo)` | `--bg-0` |
| 78 | `border: 1px solid var(--cad-borde)` | `--border` |
| 79 | `border-radius: var(--cad-radio)` | `--r-md` |
| 80 | `color: var(--cad-texto)` | `--text-1` |

### TarjetaEpisodio.svelte (`<style>` 65–163)

| línea | literal | token |
|---:|---|---|
| 72 | `background: #1e1e24` | `--surface-card` |
| 74 | `border-radius: 10px` | `--r-md` |
| 80 | `color: white` | `--text-1` |
| 91 | `background: #2a2a35` | `--bg-2` (miniatura) |
| 107 | `color: #52525b` | **excepción** (inicial decorativa; sin token) |
| 110–113 | gradiente `#2a2a35` / `#26262e` | **excepción** (trama decorativa; `#26262e` sin token) |
| 120 | `background: rgba(0, 0, 0, 0.75)` | `--surface-panel` |
| 121 | `color: #4ade80` | `--brand-400` |
| 125 | `border-radius: 6px` | `--r-sm` |
| 138 | `background: #14532d` | **excepción** (verde oscuro de la insignia; sin token) |
| 139 | `color: #4ade80` | `--brand-400` |
| 143 | `border-radius: 6px` | `--r-sm` |
| 148 | `background: #3f3f46` | `--border-strong` (relleno de la barra) |
| 149 | `border-radius: 4px` | `--r-sm` |
| 155 | `background: #4ade80` | `--brand-500` (relleno) |
| 161 | `color: #a1a1aa` | `--text-3` |

### TarjetaHistorial.svelte (`<style>` 108–243)

| línea | literal | token |
|---:|---|---|
| 124 | `border-radius: var(--cad-radio)` | `--r-md` |
| 125 | `color: var(--cad-texto)` | `--text-1` |
| 131 | `background: var(--cad-borde)` | `--border-strong` (fondo hover) |
| 134 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 146 | `border-radius: var(--cad-radio)` | `--r-md` |
| 148 | `background: var(--cad-fondo)` | `--bg-0` |
| 156 | `color: var(--cad-texto-suave)` | `--text-3` |
| 177 | `color: var(--cad-texto-suave)` | `--text-3` |
| 185 | `background: var(--cad-borde)` | `--border-strong` (pista de barra) |
| 186 | `border-radius: 4px` | `--r-sm` |
| 192 | `background: var(--cad-acento)` | `--brand-500` (relleno) |
| 196 | `color: var(--cad-texto-suave)` | `--text-3` |
| 211 | `background: var(--cad-superficie)` | `--surface-card` |
| 212 | `border: 1px solid var(--cad-borde)` | `--border` |
| 213 | `border-radius: var(--cad-radio)` | `--r-md` |
| 214 | `color: var(--cad-texto-suave)` | `--text-3` |
| 225 | `color: var(--cad-peligro)` | `--danger` |
| 226 | `border-color: var(--cad-peligro)` | `--danger` |
| 229 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |

### Toast.svelte (`<style>` 33–92)

| línea | literal | token |
|---:|---|---|
| 46 | `background: var(--cad-superficie)` | `--surface-card` |
| 47 | `color: var(--cad-texto)` | `--text-1` |
| 48 | `border: 1px solid var(--cad-borde)` | `--border` |
| 49 | `border-radius: var(--cad-radio)` | `--r-md` |
| 66 | `border: 1px solid var(--cad-borde)` | `--border` |
| 67 | `border-radius: var(--cad-radio)` | `--r-md` |
| 72 | `color: var(--cad-acento)` | `--brand-400` |
| 77 | `color: var(--cad-texto-suave)` | `--text-3` |
| 81 | `border-color: var(--cad-acento)` | `--brand-400` |
| 84 | `color: var(--cad-peligro)` | `--danger` |
| 85 | `border-color: var(--cad-peligro)` | `--danger` |
| 89 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |

### VistaBusqueda.svelte (`<style>` 104–191)

| línea | literal | token |
|---:|---|---|
| 123 | `border-radius: 8px` | `--r-md` |
| 125 | `background: #2a2a35` | `--surface-input` (campo de texto) |
| 126 | `color: white` | `--text-1` |
| 131 | `border-radius: 8px` | `--r-md` |
| 133 | `background: #4ade80` | `--brand-500` (relleno) |
| 134 | `color: #111` | `--on-brand` |
| 139 | `background: #555` | `--bg-2` (aprox., `:disabled`) |
| 140 | `color: #999` | `--text-3` (aprox., `:disabled`) |
| 144 | `color: #a1a1aa` | `--text-3` |
| 148 | `color: #f87171` | `--danger` |
| 156 | `background: #1e1e24` | `--surface-card` |
| 158 | `border-radius: 10px` | `--r-md` |
| 164 | `color: white` | `--text-1` |
| 176 | `background: #2a2a35` | `--bg-2` |
| 189 | `color: #4ade80` | `--brand-400` |

### VistaDetalle.svelte (`<style>` 293–451)

| línea | literal | token |
|---:|---|---|
| 312 | `border-radius: 8px` | `--r-md` |
| 314 | `background: #4ade80` | `--brand-500` (relleno) |
| 315 | `color: #111` | `--on-brand` |
| 324 | `background: #1e1e24` | `--surface-card` |
| 325 | `border-radius: 12px` | `--r-md` |
| 334 | `border-radius: 8px` | `--r-md` |
| 335 | `background: #2a2a35` | `--bg-2` (fondo tras la portada) |
| 344 | `color: #a1a1aa` | `--text-3` |
| 356 | `background: #2a2a35` | `--surface-chip` (chips de género) |
| 357 | `color: #4ade80` | `--brand-400` |
| 360 | `border-radius: 999px` | `--r-full` |
| 365 | `color: #d4d4d8` | `--text-2` |
| 370 | `border-radius: 8px` | `--r-md` |
| 372 | `background: #4ade80` | `--brand-500` (relleno) |
| 373 | `color: #111` | `--on-brand` |
| 378 | `background: #555` | `--bg-2` (aprox., `:disabled`) |
| 379 | `color: #999` | `--text-3` (aprox., `:disabled`) |
| 425 | `color: #a1a1aa` | `--text-3` |
| 428 | `color: #f87171` | `--danger` |
| 431 | `color: #fbbf24` | `--fav` |
| 432 | `background: #2a2a35` | `--bg-2` (bloque de aviso) |
| 433 | `border-radius: 8px` | `--r-md` |
| 438 | `color: #a1a1aa` | `--text-3` |

### VistaInicio.svelte (`<style>` 175–333)

| línea | literal | token |
|---:|---|---|
| 193 | `border-bottom: 1px solid var(--cad-borde)` | `--border` |
| 203 | `color: var(--cad-texto-suave)` | `--text-3` |
| 210 | `color: var(--cad-texto)` | `--text-1` |
| 213 | `color: var(--cad-texto)` | `--text-1` |
| 214 | `border-bottom-color: var(--cad-acento)` | `--brand-400` |
| 217 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 233 | `border-radius: var(--cad-radio)` | `--r-md` |
| 235 | `background: var(--cad-acento)` | `--brand-500` (relleno) |
| 236 | `color: #111` | `--on-brand` |
| 242 | `background: #555` | `--bg-2` (aprox., `:disabled`) |
| 243 | `color: #999` | `--text-3` (aprox., `:disabled`) |
| 247 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 253 | `background: var(--cad-superficie)` | `--surface-card` |
| 254 | `border: 1px solid var(--cad-borde)` | `--border` |
| 255 | `border-radius: var(--cad-radio)` | `--r-md` |
| 259 | `color: var(--cad-texto)` | `--text-1` |
| 264 | `color: var(--cad-texto-suave)` | `--text-3` |
| 271 | `border: 1px solid var(--cad-borde)` | `--border` |
| 272 | `border-radius: var(--cad-radio)` | `--r-md` |
| 273 | `color: var(--cad-acento)` | `--brand-400` |
| 279 | `outline: 2px solid var(--cad-acento)` | `--brand-400` |
| 283 | `color: var(--cad-texto-suave)` | `--text-3` |
| 300 | `color: var(--cad-texto-suave)` | `--text-3` |
| 311 | `border-radius: 10px` | `--r-md` |
| 314 | `var(--cad-borde) 45%` (banda) | `--border-strong` |
| 315 | `rgba(255, 255, 255, 0.08) 50%` | `--border` |
| 316 | `var(--cad-borde) 55%` (banda) | `--border-strong` |

### VistaReproductor.svelte (`<style>` 701–817)

| línea | literal | token |
|---:|---|---|
| 723 | `color: #a1a1aa` | `--text-3` |
| 728 | `border-radius: 8px` | `--r-md` |
| 730 | `background: #4ade80` | `--brand-500` (relleno) |
| 731 | `color: #111` | `--on-brand` |
| 737 | `background: #2a2a35` | `--bg-2` (caja de error) |
| 738 | `border: 1px solid #f87171` | `--danger` |
| 739 | `border-radius: 8px` | `--r-md` |
| 749 | `color: #f87171` | `--danger` |
| 755 | `border-radius: 8px` | `--r-md` |
| 757 | `background: #4ade80` | `--brand-500` (relleno) |
| 758 | `color: #111` | `--on-brand` |
| 763 | `color: #fbbf24` | `--fav` |
| 764 | `background: #2a2a35` | `--bg-2` (bloque de aviso) |
| 765 | `border-radius: 8px` | `--r-md` |
| 773 | `background: #000` | `--bg-0` (aprox.) |
| 774 | `border-radius: 12px` | `--r-md` |
| 783 | `background: #000` | `--bg-0` (aprox.) |
| 794 | `background: #000` | `--bg-0` (aprox.) |
| 803 | `color: #a1a1aa` | `--text-3` |
| 812 | `background: rgba(0, 0, 0, 0.65)` | **excepción** (scrim de carga) |
| 813 | `color: #f4f4f5` | `--text-1` |

### style.css

Solo **define** las `--cad-*` (líneas 6–13). No las usa en ninguna declaración y
tampoco hay `--cad-` en `*.ts` ni en `index.html`. El enunciado pide **no borrar** las
definiciones (se retiran en T93) → **style.css queda sin cambios**.

## Excepciones (justificadas)

| archivo:línea | literal | motivo |
|---|---|---|
| BarraLateral.svelte:563 | `rgba(0, 0, 0, 0.75)` | scrim negro del cajón (permitido por el enunciado) |
| BotonSiguiente.svelte:96, 99 | `rgba(74, 222, 128, 0.55)` / `rgba(74, 222, 128, 0)` | anillo pulsante (`@keyframes pulso-siguiente`); necesita alfa intermedio sin token |
| FilaAnimes.svelte:185, 186, 189, 190, 194, 195, 197, 198 | `black` en `mask-image` | geometría de máscara, no color del tema |
| TarjetaEpisodio.svelte:107, 110–113 | `#52525b`, gradiente `#2a2a35`/`#26262e` | inicial y trama del póster ausente (decorativo; `#26262e` sin token equivalente) |
| TarjetaEpisodio.svelte:138 | `#14532d` | fondo de la insignia «visto» (verde oscuro sin token; `--brand-600` es mucho más claro) |
| VistaReproductor.svelte:812 | `rgba(0, 0, 0, 0.65)` | scrim negro de carga (permitido por el enunciado) |

> **Sin excepciones de estructura**: ninguna sustitución exige cambiar markup,
> `<script>`, tamaños, paddings, `display`, `grid` ni `flex`. Todas las sustituciones
> caen dentro de `<style>`.

## Resumen

- Literales hex en `.svelte` antes: **82** (según `contar_literales.mjs` de T69) →
  tras T72b quedan **6**, todos en `TarjetaEpisodio.svelte` (excepciones listadas).
- Usos de `--cad-` en `.svelte` antes: **142** → tras T72b: **0**.
- `--cad-*` siguen definidas en `style.css` (legado, sin uso; se retiran en T93).
- Radios: ningún valor literal en `.svelte` tras T72b.
