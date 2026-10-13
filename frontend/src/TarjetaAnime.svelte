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
  <img src={anime.cover_url} alt={anime.title} loading="lazy" />
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
    border: 1px solid transparent;
    border-radius: 10px;
    overflow: hidden;
    background: var(--cad-superficie);
    color: var(--cad-texto);
    font-family: inherit;
    text-align: center;
    cursor: pointer;
    scroll-snap-align: start;
    display: flex;
    flex-direction: column;
    transition:
      transform 0.2s,
      border-color 0.2s,
      box-shadow 0.2s;
  }
  .tarjeta:hover {
    transform: translateY(-3px);
    border-color: var(--cad-acento);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.45);
  }
  .tarjeta:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .tarjeta img {
    display: block;
    width: 100%;
    height: 210px;
    object-fit: cover;
    background: #2a2a35;
  }
  /* Insignia del proveedor (T54): fuera del flujo para no alterar la columna
     de la portada; solo la pinta la cuadrícula de favoritos. */
  .etiqueta {
    position: absolute;
    top: 6px;
    left: 6px;
    padding: 2px 6px;
    background: var(--cad-fondo);
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    color: var(--cad-texto);
    font-size: 11px;
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
    padding: 8px 8px 0;
    margin-bottom: 10px;
    font-size: 13px;
    line-height: 1.3;
  }
  @media (prefers-reduced-motion: reduce) {
    .tarjeta {
      transition: none;
    }
    .tarjeta:hover {
      transform: none;
      box-shadow: none;
    }
  }
</style>
