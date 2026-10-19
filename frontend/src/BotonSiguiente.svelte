<script lang="ts">
  /**
   * T55: botón "Siguiente ▸" del reproductor (abajo a la derecha, dentro del
   * marco del vídeo). El cálculo del siguiente lo hace Go (`getNextEpisode`);
   * este componente solo pinta uno de los 4 estados fijos de la ficha:
   *  - `oculto`: no se renderiza (película, sin siguiente o error de la llamada);
   *  - `cargando`: visible pero deshabilitado, "Siguiente ▸", opacidad reducida;
   *  - `listo`: habilitado, "Siguiente ▸ Ep {numero}";
   *  - `resaltado`: como `listo` con realce de acento y pulso suave (solo bajo
   *    `prefers-reduced-motion: no-preference`).
   * T57: `discreto` atenúa y desenfoca el botón mientras se reproduce; solo
   * se aplica en el estado `listo` (`resaltado` nunca es discreto; `cargando`
   * y `oculto` no cambian). Sigue siendo clicable y accesible discreto.
   */
  export let estado: 'cargando' | 'listo' | 'resaltado' | 'oculto' = 'cargando';
  export let numero = 0;
  export let discreto = false;
  export let onClick: () => void = () => {};
</script>

{#if estado !== 'oculto'}
  <button
    type="button"
    class="boton-siguiente"
    class:resaltado={estado === 'resaltado'}
    class:discreto={discreto && estado === 'listo'}
    disabled={estado === 'cargando'}
    aria-busy={estado === 'cargando'}
    aria-label="Ir al siguiente episodio"
    on:click={onClick}
  >
    {estado === 'cargando' ? 'Siguiente ▸' : `Siguiente ▸ Ep ${numero}`}
  </button>
{/if}

<style>
  /* T84: chip de 44 px (--tap) sobre los controles nativos del <video> (el
     marco es position:relative), sin los 72 px sueltos de antes. En escritorio
     crece de ~34 a 44 px para ser un objetivo táctil legible. */
  .boton-siguiente {
    position: absolute;
    right: var(--sp-4);
    bottom: calc(var(--tap) + var(--sp-4));
    z-index: 3;
    min-height: var(--tap);
    padding: 0 var(--sp-4);
    border-radius: var(--r-full);
    border: 1px solid var(--border);
    background: var(--surface-panel);
    backdrop-filter: blur(var(--blur-sm));
    color: var(--text-1);
    font-family: inherit;
    font-weight: var(--fw-semibold);
    font-size: var(--fs-sm);
    cursor: pointer;
  }
  .boton-siguiente:disabled {
    opacity: 0.55;
    cursor: default;
  }
  /* T84: con el overlay de carga visible (z-index 4) el chip no debe asomar
     tras el scrim; sigue en el DOM y enfocable, y reaparece al terminar. */
  :global(.marco:has(.cargando-overlay)) .boton-siguiente {
    opacity: 0;
  }
  .boton-siguiente:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  /* T84: el hover solo existe con puntero fino. */
  @media (hover: hover) and (pointer: fine) {
    .boton-siguiente:hover:not(:disabled) {
      border-color: var(--brand-400);
    }
  }
  /* T57/T84: discreto = atenuado y desenfocado mientras se reproduce; solo
     con estado `listo` (la clase no se aplica en los demás). En puntero fino
     se aclara con hover/foco; en táctil queda visible (opacidad 0,6), sin blur
     y siempre clicable. */
  @media (hover: hover) and (pointer: fine) {
    .boton-siguiente.discreto {
      opacity: 0.15;
      filter: blur(2px);
      transition: opacity var(--dur), filter var(--dur);
    }
    .boton-siguiente.discreto:hover:not(:disabled),
    .boton-siguiente.discreto:focus-visible,
    .boton-siguiente.discreto:focus-within {
      opacity: 1;
      filter: none;
    }
  }
  @media (hover: none), (pointer: coarse) {
    .boton-siguiente.discreto {
      opacity: 0.6;
      filter: none;
    }
    .boton-siguiente.discreto:active:not(:disabled) {
      opacity: 1;
    }
  }
  .boton-siguiente.resaltado {
    border-color: var(--brand-400);
    background: var(--grad-brand);
    color: var(--on-brand);
  }
  @media (prefers-reduced-motion: no-preference) {
    .boton-siguiente.resaltado {
      animation: pulso-siguiente 2s ease-in-out infinite;
    }
  }
  /* T84: pulso con el token de marca; cero literales de color. */
  @keyframes pulso-siguiente {
    0%,
    100% {
      box-shadow: 0 0 0 0 var(--brand-glow);
    }
    50% {
      box-shadow: 0 0 0 var(--sp-2) transparent;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .boton-siguiente,
    .boton-siguiente.discreto {
      transition: none;
    }
  }
</style>
