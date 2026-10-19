<script lang="ts">
  /**
   * T80: barra de acción fija del detalle en modo estrecho (< 900 px). Va
   * pegada justo encima de la barra inferior (T73) y da acceso permanente a la
   * acción principal (Continuar / Empezar) y, si hay progreso, a "Reiniciar".
   * Solo presentacional: los textos y los callbacks los decide `VistaDetalle`.
   *
   * Sintaxis clásica de Svelte (`export let`), sin runes; solo tokens.
   */
  import Btn from './Btn.svelte';
  import Icono from './Icono.svelte';

  export let etiquetaPrincipal: string;
  export let onPrincipal: () => void;
  export let deshabilitado = false;
  export let etiquetaSecundaria = '';
  export let onSecundaria: () => void = () => {};
</script>

<div class="barra-accion" role="group" aria-label="Acciones del anime">
  <div class="principal">
    <Btn bloque disabled={deshabilitado} onClick={onPrincipal}>
      <svelte:fragment slot="icono-izq">
        <Icono nombre="play" tam={20} />
      </svelte:fragment>
      {etiquetaPrincipal}
    </Btn>
  </div>
  {#if etiquetaSecundaria !== ''}
    <Btn variante="secundario" onClick={onSecundaria}>{etiquetaSecundaria}</Btn>
  {/if}
</div>

<style>
  /* Pegada justo sobre la barra inferior (su alto + safe-area) y con su misma
     capa para que el cajón de historial (`--z-drawer`, por encima) la cubra. */
  .barra-accion {
    box-sizing: border-box;
    position: fixed;
    left: 0;
    right: 0;
    bottom: calc(var(--bar-h) + var(--sab));
    z-index: var(--z-bar);
    display: flex;
    align-items: stretch;
    gap: var(--sp-2);
    padding: var(--sp-2) var(--gutter);
    background: var(--surface-panel);
    border-top: 1px solid var(--border);
    backdrop-filter: blur(var(--blur-md));
    -webkit-backdrop-filter: blur(var(--blur-md));
  }
  .principal {
    flex: 1 1 auto;
    min-width: 0;
  }
  @media (prefers-reduced-motion: reduce) {
    .barra-accion {
      transition: none;
    }
  }
</style>
