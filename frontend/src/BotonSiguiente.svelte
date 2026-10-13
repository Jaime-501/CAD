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
   */
  export let estado: 'cargando' | 'listo' | 'resaltado' | 'oculto' = 'cargando';
  export let numero = 0;
  export let onClick: () => void = () => {};
</script>

{#if estado !== 'oculto'}
  <button
    type="button"
    class="boton-siguiente"
    class:resaltado={estado === 'resaltado'}
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
    border-radius: var(--cad-radio);
    border: 1px solid var(--cad-borde);
    background: var(--cad-superficie);
    color: var(--cad-texto);
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
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .boton-siguiente:hover:not(:disabled) {
    border-color: var(--cad-acento);
  }
  .boton-siguiente.resaltado {
    border-color: var(--cad-acento);
    background: var(--cad-acento);
    color: #111;
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
