<script lang="ts">
  /**
   * Toast fijo abajo-centro (T51): `role="status"` con `aria-live="polite"`,
   * botón con la etiqueta de la acción (p. ej. "Deshacer") y botón ✕ de
   * cerrar. El temporizador vive en `toasts.ts`; al desmontar se limpia con
   * `cerrarToast()`.
   */
  import { onDestroy } from 'svelte';
  import Icono from './Icono.svelte';
  import { cerrarToast, toast } from './toasts';

  function alAccionar(): void {
    const accion = $toast?.accion;
    accion?.fn();
  }

  onDestroy(() => cerrarToast());
</script>

{#if $toast !== null}
  <div class="toast" role="status" aria-live="polite">
    <span class="mensaje">{$toast.mensaje}</span>
    {#if $toast.accion}
      <button type="button" class="accion" on:click={alAccionar}>{$toast.accion.etiqueta}</button>
    {/if}
    <button type="button" class="cerrar" aria-label="Cerrar aviso" on:click={cerrarToast}>
      <Icono nombre="cerrar" tam={16} />
    </button>
  </div>
{/if}

<style>
  .toast {
    position: fixed;
    left: 50%;
    bottom: calc(var(--sp-6) + var(--bar-offset, 0px));
    transform: translateX(-50%);
    z-index: var(--z-toast);
    box-sizing: border-box;
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    max-width: min(420px, calc(100vw - 2 * var(--gutter)));
    padding: var(--sp-2) var(--sp-3);
    background: var(--bg-2);
    color: var(--text-1);
    border: 1px solid var(--border);
    border-radius: var(--r-lg);
    box-shadow: var(--sh-panel);
    font-size: var(--fs-sm);
    text-align: left;
    animation: toast-entrada var(--dur) var(--ease-out);
  }
  .mensaje {
    flex: 1;
    min-width: 0;
  }
  .accion,
  .cerrar {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    padding: var(--sp-1) var(--sp-2);
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: inherit;
    font-family: inherit;
    font-size: inherit;
    cursor: pointer;
  }
  .accion {
    color: var(--brand-400);
    font-weight: var(--fw-bold);
  }
  .cerrar {
    padding: var(--sp-1);
    color: var(--text-3);
  }
  .accion:active,
  .cerrar:active {
    transform: scale(0.97);
  }
  .accion:focus-visible,
  .cerrar:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  @media (hover: hover) and (pointer: fine) {
    .accion:hover {
      border-color: var(--brand-400);
    }
    .cerrar:hover {
      color: var(--danger);
      border-color: var(--danger);
    }
  }
  @media (max-width: 899px) {
    .accion,
    .cerrar {
      min-width: var(--tap);
      min-height: var(--tap);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .toast {
      animation: none;
    }
    .accion,
    .cerrar {
      transition: none;
    }
    .accion:active,
    .cerrar:active {
      transform: none;
    }
  }
  @keyframes toast-entrada {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
</style>
