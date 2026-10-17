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
  /* right/bottom 16 px sobre los 56 px reservados para los controles nativos
     del <video> (el marco es position:relative): el botón nunca los tapa. */
  .boton-siguiente {
    position: absolute;
    right: 16px;
    bottom: 72px;
    z-index: 3;
    padding: 8px 16px;
    border-radius: var(--r-md);
    border: 1px solid var(--border);
    background: var(--surface-card);
    color: var(--text-1);
    font-family: inherit;
    font-weight: bold;
    font-size: 14px;
    cursor: pointer;
  }
  .boton-siguiente:disabled {
    opacity: 0.55;
    cursor: default;
  }
  .boton-siguiente:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  .boton-siguiente:hover:not(:disabled) {
    border-color: var(--brand-400);
  }
  /* T57: discreto = atenuado y desenfocado mientras se reproduce; solo con
     estado `listo` (la clase no se aplica en los demás). Clicable y con foco. */
  .boton-siguiente.discreto {
    opacity: 0.15;
    filter: blur(2px);
    transition: opacity 200ms, filter 200ms;
  }
  .boton-siguiente.discreto:hover:not(:disabled),
  .boton-siguiente.discreto:focus-visible,
  .boton-siguiente.discreto:focus-within {
    opacity: 1;
    filter: none;
  }
  @media (prefers-reduced-motion: reduce) {
    .boton-siguiente.discreto {
      transition: none;
    }
  }
  .boton-siguiente.resaltado {
    border-color: var(--brand-400);
    background: var(--brand-500);
    color: var(--on-brand);
  }
  @media (prefers-reduced-motion: no-preference) {
    .boton-siguiente.resaltado {
      animation: pulso-siguiente 2s ease-in-out infinite;
    }
  }
  @keyframes pulso-siguiente {
    0%,
    100% {
      box-shadow: 0 0 0 0 rgba(74, 222, 128, 0.55);
    }
    50% {
      box-shadow: 0 0 0 6px rgba(74, 222, 128, 0);
    }
  }
</style>
