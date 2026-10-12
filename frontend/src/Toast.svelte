<script lang="ts">
  /**
   * Toast fijo abajo-centro (T51): `role="status"` con `aria-live="polite"`,
   * botón con la etiqueta de la acción (p. ej. "Deshacer") y botón ✕ de
   * cerrar. El temporizador vive en `toasts.ts`; al desmontar se limpia con
   * `cerrarToast()`.
   */
  import { onDestroy } from 'svelte';
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
      <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true" focusable="false">
        <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" fill="none" />
      </svg>
    </button>
  </div>
{/if}

<style>
  .toast {
    position: fixed;
    left: 50%;
    bottom: 24px;
    transform: translateX(-50%);
    z-index: 100;
    box-sizing: border-box;
    display: flex;
    align-items: center;
    gap: 12px;
    max-width: min(420px, calc(100vw - 32px));
    padding: 10px 12px;
    background: var(--cad-superficie);
    color: var(--cad-texto);
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    text-align: left;
  }
  .mensaje {
    flex: 1;
    min-width: 0;
    font-size: 13px;
  }
  .accion,
  .cerrar {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    padding: 4px 8px;
    background: transparent;
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    font-family: inherit;
    cursor: pointer;
  }
  .accion {
    color: var(--cad-acento);
    font-weight: bold;
  }
  .cerrar {
    padding: 4px;
    color: var(--cad-texto-suave);
  }
  .accion:hover,
  .cerrar:hover {
    border-color: var(--cad-acento);
  }
  .cerrar:hover {
    color: var(--cad-peligro);
    border-color: var(--cad-peligro);
  }
  .accion:focus-visible,
  .cerrar:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
</style>
