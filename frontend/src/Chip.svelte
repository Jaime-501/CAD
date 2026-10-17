<script lang="ts">
  /**
   * T72: chip seleccionable. Se usa como `<button>` con `aria-pressed`; el texto
   * envuelve sin desbordar y el alto táctil llega a 40 px (`--tap` − `--sp-1`)
   * sin escribir ningún px literal.
   *
   * Activo: degradado `--grad-chip-active`, texto `--on-brand` y un punto
   * indicador estático (sin animación). Inactivo: `--surface-chip` + `--border`.
   */
  export let activo = false;
  export let onClick: () => void = () => {};
</script>

<button
  type="button"
  class="chip"
  class:activo
  aria-pressed={activo}
  on:click={onClick}
>
  {#if activo}
    <span class="punto" aria-hidden="true"></span>
  {/if}
  <span class="texto"><slot /></span>
</button>

<style>
  .chip {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    gap: var(--sp-2);
    max-width: 100%;
    min-height: calc(var(--tap) - var(--sp-1));
    padding: var(--sp-1) var(--sp-3);
    border: 1px solid var(--border);
    border-radius: var(--r-full);
    background: var(--surface-chip);
    color: var(--text-2);
    font-family: var(--font-sans);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    line-height: var(--lh-tight);
    text-align: left;
    cursor: pointer;
    white-space: normal;
    overflow-wrap: anywhere;
    transition: background var(--dur-fast) var(--ease-std), color var(--dur-fast) var(--ease-std);
    -webkit-tap-highlight-color: transparent;
  }
  .texto {
    min-width: 0;
  }
  .activo {
    background: var(--grad-chip-active);
    border-color: transparent;
    color: var(--on-brand);
  }
  .punto {
    flex-shrink: 0;
    width: var(--sp-2);
    height: var(--sp-2);
    border-radius: var(--r-full);
    background: var(--on-brand);
  }

  @media (hover: hover) and (pointer: fine) {
    .chip:not(.activo):hover {
      background: var(--surface-chip-hover);
    }
  }

  .chip:active {
    transform: scale(0.98);
  }
  .chip:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }

  @media (prefers-reduced-motion: reduce) {
    .chip {
      transition: none;
    }
    .chip:active {
      transform: none;
    }
  }
</style>
