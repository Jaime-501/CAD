<script lang="ts">
  /**
   * T72: botón base (o enlace con `href`). Variantes `primario`, `secundario`,
   * `fantasma` e `icono`; tamaños `m` y `l`. Todas las variantes respetan el
   * objetivo táctil mínimo `--tap` (44 px) en ancho y alto.
   *
   * El hover vive SOLO bajo `@media (hover: hover) and (pointer: fine)` (P6):
   * en táctil no hay estados pegajosos. Slots `icono-izq` e `icono-der` para
   * los `Icono`; el contenido por defecto es el texto.
   */
  export let variante: 'primario' | 'secundario' | 'fantasma' | 'icono' = 'primario';
  export let tam: 'm' | 'l' = 'm';
  /** Ancho completo del contenedor. */
  export let bloque = false;
  /** Si se pasa, se renderiza un `<a>` en lugar de un `<button>`. */
  export let href: string | undefined = undefined;
  export let disabled = false;
  export let type: 'button' | 'submit' | 'reset' = 'button';
  /** `aria-label` para botones que solo llevan icono. */
  export let etiqueta = '';
  export let onClick: () => void = () => {};

  function alClic(evento: MouseEvent): void {
    if (disabled) {
      evento.preventDefault();
      return;
    }
    onClick();
  }
</script>

{#if href}
  <a
    class="btn tam-{tam} variante-{variante}"
    class:bloque
    class:deshabilitado={disabled}
    href={disabled ? undefined : href}
    aria-disabled={disabled ? 'true' : undefined}
    tabindex={disabled ? -1 : undefined}
    aria-label={etiqueta !== '' ? etiqueta : undefined}
    on:click={alClic}
  >
    <slot name="icono-izq" />
    <span class="texto"><slot /></span>
    <slot name="icono-der" />
  </a>
{:else}
  <button
    class="btn tam-{tam} variante-{variante}"
    class:bloque
    {type}
    {disabled}
    aria-label={etiqueta !== '' ? etiqueta : undefined}
    on:click={onClick}
  >
    <slot name="icono-izq" />
    <span class="texto"><slot /></span>
    <slot name="icono-der" />
  </button>
{/if}

<style>
  .btn {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--sp-2);
    min-width: var(--tap);
    min-height: var(--tap);
    border: 1px solid transparent;
    border-radius: var(--r-md);
    font-family: var(--font-sans);
    font-weight: var(--fw-semibold);
    line-height: var(--lh-tight);
    text-decoration: none;
    cursor: pointer;
    transition:
      transform var(--dur-fast) var(--ease-std),
      background var(--dur-fast) var(--ease-std),
      border-color var(--dur-fast) var(--ease-std);
    -webkit-tap-highlight-color: transparent;
  }
  .tam-m {
    padding: var(--sp-2) var(--sp-4);
    font-size: var(--fs-sm);
  }
  .tam-l {
    padding: var(--sp-3) var(--sp-5);
    font-size: var(--fs-md);
  }
  .bloque {
    width: 100%;
  }
  .texto:empty {
    display: none;
  }

  .variante-primario {
    background: var(--grad-brand);
    color: var(--on-brand);
    box-shadow: var(--sh-glow);
  }
  .variante-secundario {
    background: var(--surface-chip);
    border-color: var(--border);
    color: var(--text-1);
  }
  .variante-fantasma {
    background: none;
    color: var(--text-2);
  }
  .variante-icono {
    width: var(--tap);
    min-width: var(--tap);
    padding: 0;
    background: none;
    color: var(--text-2);
    border-radius: var(--r-full);
  }

  /* Hover solo en punteros finos; nunca en táctil (P6). */
  @media (hover: hover) and (pointer: fine) {
    .variante-primario:hover:not(:disabled):not(.deshabilitado) {
      box-shadow: var(--sh-card-hover);
    }
    .variante-secundario:hover:not(:disabled):not(.deshabilitado) {
      background: var(--surface-chip-hover);
    }
    .variante-fantasma:hover:not(:disabled):not(.deshabilitado),
    .variante-icono:hover:not(:disabled):not(.deshabilitado) {
      background: var(--surface-chip);
      color: var(--text-1);
    }
  }

  .btn:active:not(:disabled):not(.deshabilitado) {
    transform: scale(0.98);
  }
  .btn:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  .btn:disabled,
  .btn.deshabilitado {
    opacity: 0.5;
    cursor: not-allowed;
  }

  @media (prefers-reduced-motion: reduce) {
    .btn {
      transition: none;
    }
    .btn:active:not(:disabled):not(.deshabilitado) {
      transform: none;
    }
  }
</style>
