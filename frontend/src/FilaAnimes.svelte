<script lang="ts">
  /**
   * Fila horizontal de animes del Home (T53, pulida en T53b): carrusel sin
   * barra nativa, con degradados en los bordes según el estado de scroll,
   * flechas ‹ › circulares que desplazan el 80 % del ancho visible y se
   * deshabilitan en los extremos (recalculados con el evento `scroll` y al
   * montar/cambiar la lista). Si la fila no desborda no hay flechas ni
   * degradados.
   *
   * `proveedor` (opcional, T53b) divide el encabezado en dos partes: nombre
   * del proveedor pequeño y título de sección en negrita; sin `proveedor` el
   * encabezado se comporta como en T53 (solo `titulo`). T54 reutiliza este
   * componente con las mismas props.
   *
   * Cada tarjeta es un `<button>`: el clic llama a `onAbrir(anime)`, que es la
   * MISMA función de "abrir anime" de los resultados de búsqueda (se la pasa
   * `VistaBusqueda` a `VistaInicio`, sin vía de navegación nueva).
   */
  import { onMount, tick } from 'svelte';
  import type { models } from './api';

  export let titulo: string;
  export let items: models.Anime[] = [];
  export let onAbrir: (a: models.Anime) => void;
  export let proveedor: string | undefined = undefined;

  let carrusel: HTMLDivElement;
  /** ¿La fila desborda? Sin desborde no hay flechas ni degradados que mostrar. */
  let hayDesborde = false;
  let puedeIzquierda = false;
  let puedeDerecha = false;

  onMount(() => {
    void actualizar();
  });

  // Flechas y degradados al cambiar la lista (y tras pintarla): `tick` espera al DOM nuevo.
  $: items, void actualizar();

  async function actualizar(): Promise<void> {
    await tick();
    if (!carrusel) return;
    const { scrollLeft, scrollWidth, clientWidth } = carrusel;
    hayDesborde = scrollWidth > clientWidth + 1;
    puedeIzquierda = hayDesborde && scrollLeft > 1;
    puedeDerecha = hayDesborde && scrollLeft + clientWidth < scrollWidth - 1;
  }

  /** Desplaza el 80 % del ancho visible; sin animación bajo `prefers-reduced-motion`. */
  function desplazar(direccion: number): void {
    if (!carrusel) return;
    const paso = carrusel.clientWidth * 0.8;
    const sinAnimacion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    carrusel.scrollBy({ left: direccion * paso, behavior: sinAnimacion ? 'auto' : 'smooth' });
  }
</script>

<section class="fila" aria-label={proveedor !== undefined ? `${proveedor} · ${titulo}` : titulo}>
  <div class="cabecera">
    {#if proveedor !== undefined}
      <span class="proveedor">{proveedor}</span>
      <h3 class="titulo-fila">{titulo}</h3>
    {:else}
      <h3 class="titulo-fila">{titulo}</h3>
    {/if}
    {#if hayDesborde}
      <div class="flechas">
        <button
          type="button"
          class="flecha"
          aria-label="Desplazar a la izquierda"
          disabled={!puedeIzquierda}
          on:click={() => desplazar(-1)}
        >
          <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true" focusable="false">
            <path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        </button>
        <button
          type="button"
          class="flecha"
          aria-label="Desplazar a la derecha"
          disabled={!puedeDerecha}
          on:click={() => desplazar(1)}
        >
          <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true" focusable="false">
            <path d="M9 6l6 6-6 6" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        </button>
      </div>
    {/if}
  </div>

  <div
    class="carrusel"
    class:degradado-izquierda={puedeIzquierda}
    class:degradado-derecha={puedeDerecha}
    bind:this={carrusel}
    on:scroll={actualizar}
  >
    {#each items as anime (anime.provider + ':' + anime.id)}
      <button type="button" class="tarjeta" title={anime.title} on:click={() => onAbrir(anime)}>
        <img src={anime.cover_url} alt={anime.title} loading="lazy" />
        <span class="titulo">{anime.title}</span>
      </button>
    {/each}
  </div>
</section>

<style>
  .fila {
    margin: 0 0 28px;
    text-align: left;
  }
  .cabecera {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin: 0 0 10px;
  }
  .proveedor {
    font-size: 12px;
    color: var(--cad-texto-suave);
  }
  .titulo-fila {
    margin: 0;
    font-size: 16px;
    font-weight: bold;
    color: var(--cad-texto);
  }
  .flechas {
    margin-left: auto;
    display: flex;
    gap: 6px;
  }
  .flecha {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 0;
    background: var(--cad-superficie);
    border: 1px solid var(--cad-borde);
    border-radius: 50%;
    color: var(--cad-texto);
    cursor: pointer;
  }
  .flecha:hover:not(:disabled) {
    border-color: var(--cad-acento);
    color: var(--cad-acento);
  }
  .flecha:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
  .flecha:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .carrusel {
    display: flex;
    gap: 12px;
    overflow-x: auto;
    scroll-snap-type: x proximity;
    /* El snap debe alinearse con el borde de la fila, no con el padding box
       (si no, la primera tarjeta arranca desplazada 4 px y la flecha
       izquierda queda activa en el extremo inicial). */
    scroll-padding-left: 4px;
    scroll-padding-right: 4px;
    /* Barra nativa fuera (T53b): el desplazamiento sigue con las flechas,
       la rueda + Shift, el trackpad y el teclado. */
    scrollbar-width: none;
    /* Relleno vertical para que la elevación (-3px) y la sombra del hover no
       se recorten; los márgenes negativos anulan el relleno y no mueven la fila. */
    padding: 6px 4px 14px;
    margin: -6px -4px -14px;
  }
  .carrusel::-webkit-scrollbar {
    display: none;
  }
  /* Degradados de borde (T53b): solo si hay desborde por ese lado. */
  .carrusel.degradado-derecha {
    -webkit-mask-image: linear-gradient(to left, transparent 0, black 32px);
    mask-image: linear-gradient(to left, transparent 0, black 32px);
  }
  .carrusel.degradado-izquierda {
    -webkit-mask-image: linear-gradient(to right, transparent 0, black 32px);
    mask-image: linear-gradient(to right, transparent 0, black 32px);
  }
  .carrusel.degradado-izquierda.degradado-derecha {
    -webkit-mask-image:
      linear-gradient(to right, transparent 0, black 32px),
      linear-gradient(to left, transparent 0, black 32px);
    mask-image:
      linear-gradient(to right, transparent 0, black 32px),
      linear-gradient(to left, transparent 0, black 32px);
    -webkit-mask-composite: source-in;
    mask-composite: intersect;
  }
  /* Misma tarjeta que la de los resultados de búsqueda, a 140 px y portada 2:3. */
  .tarjeta {
    box-sizing: border-box;
    flex: 0 0 140px;
    width: 140px;
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
