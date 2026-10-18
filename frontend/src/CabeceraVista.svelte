<script lang="ts">
  /**
   * T74: cabecera compacta y única de las vistas de detalle y reproductor.
   *
   * "← Atrás" (icono `flecha-izq` + texto), bloque de título (`h2`) y subtítulo
   * opcional, con un slot `acciones` a la derecha (la estrella de favorito). En
   * modo estrecho (`< 900 px`, mismo corte que `isNarrow`) gana el objetivo
   * táctil de `--tap`, el título se recorta a una línea con elipsis y, con
   * `fija`, la cabecera queda pegada arriba con `--surface-panel` + blur.
   *
   * Sintaxis clásica de Svelte (`export let`), sin runes; solo tokens.
   */
  import Icono from './Icono.svelte';

  export let titulo: string;
  export let subtitulo = '';
  /** `true` ⇒ cabecera pegajosa arriba en modo estrecho (detalle). */
  export let fija = false;
  export let onAtras: () => void = () => {};
</script>

<header class="cabecera" class:fija>
  <button type="button" class="atras" aria-label="Volver" on:click={onAtras}>
    <Icono nombre="flecha-izq" tam={20} />
    <span class="etiqueta-atras">Atrás</span>
  </button>

  <div class="titulos">
    <h2>{titulo}</h2>
    {#if subtitulo !== ''}
      <p class="subtitulo">{subtitulo}</p>
    {/if}
  </div>

  <div class="acciones">
    <slot name="acciones" />
  </div>
</header>

<style>
  .cabecera {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 1.5rem;
  }
  .titulos {
    min-width: 0;
  }
  .cabecera h2 {
    margin: 0;
    text-align: left;
    font-family: var(--font-display);
  }
  .subtitulo {
    margin: 4px 0 0;
    color: var(--text-3);
    font-size: 14px;
  }
  .acciones {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }
  /* El icono de 20 px es más ancho que el glifo "←" que sustituye; se recorta
     el padding lateral (14.27 px) para que el botón conserve su ancho exterior
     y la fila de la cabecera no cambie en ≥ 900 px. */
  .atras {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    padding: 8px 14.27px;
    border-radius: var(--r-md);
    border: none;
    background: var(--brand-500);
    color: var(--on-brand);
    font-weight: bold;
    cursor: pointer;
    white-space: nowrap;
  }
  /* El icono de 20 px no debe agrandar el alto de la caja del botón (hoy 31 px). */
  .atras :global(svg) {
    margin-block: calc(-1 * var(--sp-1));
  }

  /* Modo estrecho (< 900 px): táctiles, una línea con elipsis y cabecera fija. */
  @media (max-width: 899px) {
    .cabecera {
      padding: calc(var(--sat) + var(--sp-2)) var(--gutter) var(--sp-2);
    }
    .atras {
      justify-content: center;
      min-width: var(--tap);
      min-height: var(--tap);
    }
    .cabecera h2 {
      font-size: var(--fs-lg);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .subtitulo {
      font-size: var(--fs-sm);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .acciones {
      margin-left: auto;
    }
    /* Fondo a todo el ancho del `.contenedor`: los márgenes negativos cancelan
       su padding lateral y el padding propio (gutter) vuelve a alinear el
       contenido. */
    .cabecera.fija {
      position: sticky;
      top: 0;
      z-index: var(--z-sticky);
      margin: 0 calc(-1 * var(--gutter));
      padding: calc(var(--sat) + var(--sp-2)) var(--gutter) var(--sp-2);
      background: var(--surface-panel);
      border-bottom: 1px solid var(--border);
      backdrop-filter: blur(var(--blur-md));
      -webkit-backdrop-filter: blur(var(--blur-md));
    }
    @media (prefers-reduced-motion: reduce) {
      .cabecera,
      .atras {
        transition: none;
      }
    }
  }
</style>
