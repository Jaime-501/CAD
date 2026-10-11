<script lang="ts">
  /**
   * Barra lateral del historial (T48): estructura y estado, sin datos todavía.
   * Sin props: habla con `ui.ts` (`sidebarCollapsed`, `drawerOpen`, `isNarrow`).
   *
   * - Pantalla ancha: columna de la rejilla, sticky a `top:0` con `height:100vh`;
   *   el botón contrae/expandir (300px ↔ 56px) y el estado se persiste.
   * - Modo estrecho (`isNarrow`): cajón fijo sobre el contenido; `sidebarCollapsed`
   *   se ignora, el botón cierra, el foco va al botón al abrir y `Escape` cierra.
   */
  import { tick } from 'svelte';
  import { get } from 'svelte/store';
  import { drawerOpen, isNarrow, sidebarCollapsed } from './ui';

  let boton: HTMLButtonElement | null = null;

  // Al abrir el cajón el foco va al botón de cerrar/contraer (regla de
  // teclado de T48): con `tick` el `bind:this` ya está aplicado.
  $: enfocarCajon($isNarrow, $drawerOpen);

  async function enfocarCajon(narrow: boolean, abierto: boolean) {
    if (!narrow || !abierto) return;
    await tick();
    boton?.focus();
  }

  function alternar() {
    if (get(isNarrow)) {
      drawerOpen.set(false);
      return;
    }
    sidebarCollapsed.update((v) => !v);
  }

  function cerrarCajon() {
    drawerOpen.set(false);
  }

  function alTeclar(e: KeyboardEvent) {
    if (e.key === 'Escape' && get(isNarrow) && get(drawerOpen)) {
      drawerOpen.set(false);
    }
  }
</script>

<svelte:window on:keydown={alTeclar} />

{#if $isNarrow && $drawerOpen}
  <button type="button" class="fondo-cajon" aria-label="Cerrar historial" on:click={cerrarCajon}></button>
{/if}

{#if !$isNarrow || $drawerOpen}
  <aside
    id="barra-historial"
    class="barra"
    class:cajon={$isNarrow}
    class:contraida={!$isNarrow && $sidebarCollapsed}
  >
    <div class="cabecera">
      <span class="titulo">Continuar viendo</span>
      <button
        bind:this={boton}
        type="button"
        class="alternar"
        aria-label={$isNarrow
          ? 'Cerrar barra'
          : $sidebarCollapsed
            ? 'Expandir barra'
            : 'Contraer barra'}
        aria-expanded={$isNarrow ? $drawerOpen : !$sidebarCollapsed}
        aria-controls="barra-historial"
        on:click={alternar}
      >
        {#if $isNarrow}
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true" focusable="false">
            <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        {:else if $sidebarCollapsed}
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true" focusable="false">
            <path d="M9 6l6 6-6 6" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        {:else}
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true" focusable="false">
            <path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        {/if}
      </button>
    </div>
    <div class="cuerpo">
      {#if $isNarrow || !$sidebarCollapsed}
        <p class="vacio">Aún no hay nada que retomar.</p>
      {/if}
    </div>
  </aside>
{/if}

<style>
  .barra {
    position: sticky;
    top: 0;
    height: 100vh;
    align-self: start;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    width: 300px;
    background: var(--cad-superficie);
    color: var(--cad-texto);
    border-right: 1px solid var(--cad-borde);
    text-align: left;
    transition: width 180ms ease;
  }
  .barra.contraida {
    width: 56px;
  }
  .cabecera {
    box-sizing: border-box;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px;
    border-bottom: 1px solid var(--cad-borde);
  }
  .titulo {
    flex: 1;
    font-weight: bold;
    white-space: nowrap;
    overflow: hidden;
  }
  .barra.contraida .titulo {
    display: none;
  }
  .alternar {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    padding: 6px;
    background: transparent;
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    color: var(--cad-texto);
    cursor: pointer;
  }
  .alternar:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .cuerpo {
    flex: 1;
    box-sizing: border-box;
    overflow-y: auto;
    padding: 12px;
  }
  .vacio {
    margin: 0;
    color: var(--cad-texto-suave);
  }
  /* Modo estrecho: cajón fijo sobre el contenido (el ancho colapsado se ignora). */
  .barra.cajon {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: min(320px, 85vw);
    height: 100vh;
    z-index: 50;
    border-right: none;
    transition: none;
  }
  .fondo-cajon {
    position: fixed;
    inset: 0;
    z-index: 40;
    padding: 0;
    border: none;
    cursor: pointer;
    /* Mismo fondo oscuro semitransparente que las superposiciones existentes
       (TarjetaEpisodio.svelte: rgba(0,0,0,0.75)). */
    background: rgba(0, 0, 0, 0.75);
  }
  .fondo-cajon:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: -2px;
  }
  @media (prefers-reduced-motion: reduce) {
    .barra {
      transition: none;
    }
  }
</style>
