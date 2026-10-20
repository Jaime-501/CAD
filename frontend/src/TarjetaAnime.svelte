<script lang="ts">
  /**
   * Tarjeta de anime (T54): la MISMA tarjeta que pintaba `FilaAnimes` (T53,
   * pulida en T53b), extraída a componente para poder reutilizarla en la
   * cuadrícula de favoritos sin duplicar su CSS.
   *
   * `etiqueta` (opcional) añade la insignia del proveedor arriba a la izquierda
   * de la portada: la cuadrícula de favoritos mezcla proveedores (T54), mientras
   * las filas del Home no la pasan y su tarjeta queda idéntica a la de T53b.
   *
   * El ancho de la fila vive aquí (`flex: 0 0 140px`) porque el botón ES el
   * elemento flex del carrusel; dentro de una cuadrícula (grid) esa regla se
   * ignora y la tarjeta llena su columna. Por eso NO se fija `width`: recortaría
   * la tarjeta a 140 px dentro de columnas más anchas. `overflow: hidden` es
   * imprescindible para que `flex-basis` no se ensanche con títulos largos.
   */
  import type { models } from './api';

  export let anime: models.Anime;
  export let onAbrir: (a: models.Anime) => void;
  export let etiqueta: string | undefined = undefined;
</script>

<button type="button" class="tarjeta" title={anime.title} on:click={() => onAbrir(anime)}>
  <img src={anime.cover_url} alt={anime.title} loading="lazy" decoding="async" />
  {#if etiqueta !== undefined && etiqueta !== ''}
    <span class="etiqueta">{etiqueta}</span>
  {/if}
  <span class="titulo">{anime.title}</span>
</button>

<style>
  .tarjeta {
    position: relative;
    box-sizing: border-box;
    flex: 0 0 140px;
    padding: 0;
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    overflow: hidden;
    background: var(--surface-card);
    box-shadow: var(--sh-card);
    color: var(--text-1);
    font-family: inherit;
    text-align: center;
    cursor: pointer;
    scroll-snap-align: start;
    display: flex;
    flex-direction: column;
    transition:
      transform var(--dur-fast) var(--ease-std),
      border-color var(--dur-fast) var(--ease-std),
      box-shadow var(--dur-fast) var(--ease-std);
  }
  /* :active (táctil) por delante de :hover: con ratón el hover eleva; al
     pulsar, la tarjeta se hunde ligeramente. */
  .tarjeta:active {
    transform: scale(0.97);
  }
  .tarjeta:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  /* Portada (T76): `aspect-ratio` 2:3 en lugar de `height` fijo. Con el ancho
     del padding box (138 px por el borde de 1 px) el `aspect-ratio` daría
     138×207 y la tarjeta perdería 3 px de alto (260,8 en vez de 263,8). Para
     conservarla, la portada ocupa el ancho COMPLETO del marco (140 px) con
     `calc(100% + 2px)` y márgenes negativos: pasa a 140×210 y la tarjeta sigue
     en 263,8. La portada gana así 2 px de ancho (antes 138 px) y queda en 2:3
     exacto; `overflow: hidden` recorta lo que sobresale del marco. */
  .tarjeta img {
    display: block;
    width: calc(100% + 2px);
    margin: 0 -1px;
    aspect-ratio: 2 / 3;
    object-fit: cover;
    background: var(--bg-2);
  }
  /* Insignia del proveedor (T54): fuera del flujo para no alterar la columna
     de la portada; solo la pinta la cuadrícula de favoritos. */
  .etiqueta {
    position: absolute;
    top: 6px;
    left: 6px;
    padding: 2px 6px;
    background: var(--bg-0);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--text-1);
    font-size: var(--fs-xs);
    line-height: 1.3;
    pointer-events: none;
  }
  /* Bloque de título con altura fija de 2 líneas (T53b): todas las tarjetas
     miden igual aunque el título sea de 1 línea; el clamp (`-webkit-line-clamp`
     + `line-clamp`) corta con "…" y `overflow: hidden` recorta. El padding
     inferior se traslada a `margin` para que la 3.ª línea fantasma quede fuera
     del área de recorte (el clip es en el borde del padding box). */
  .titulo {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    box-sizing: content-box;
    height: calc(1.3em * 2);
    padding: var(--sp-2) var(--sp-2) 0;
    margin-bottom: 10px;
    font-size: 13px;
    line-height: 1.3;
  }
  /* El realce al pasar el ratón solo donde hay puntero fino (P6): en táctil no
     existen estilos de hover pegajosos; ahí manda `:active`. */
  @media (hover: hover) and (pointer: fine) {
    .tarjeta:hover {
      transform: translateY(-3px);
      border-color: var(--brand-400);
      box-shadow: var(--sh-card-hover);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .tarjeta {
      transition: none;
    }
    .tarjeta:hover,
    .tarjeta:active {
      transform: none;
    }
  }
</style>
