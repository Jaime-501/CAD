<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import BarraLateral from './BarraLateral.svelte';
  import { disposeEvents, initEvents } from './stores';
  import { drawerOpen, isNarrow } from './ui';
  import {
    claveScroll,
    guardarScroll,
    recuperarScroll,
    vista,
    type VistaNombre,
  } from './views';
  import VistaBusqueda from './VistaBusqueda.svelte';
  import VistaDetalle from './VistaDetalle.svelte';
  import VistaReproductor from './VistaReproductor.svelte';

  const MENSAJE_PUENTE =
    'Sin conexión con Go. Abre la app en http://localhost:34115 — el puerto 5173 (Vite) no tiene el puente de Wails.';

  let puente = true;
  let vistaPrevia: VistaNombre = 'busqueda';

  onMount(() => {
    // window.go solo lo inyecta el dev server de Wails (:34115),
    // no el servidor de Vite (:5173)
    puente = typeof window !== 'undefined' && Boolean((window as any).go);
    initEvents();
  });

  onDestroy(() => {
    disposeEvents();
  });

  // Al cambiar de vista: se guarda el scroll de la vista que se deja (la de
  // búsqueda por su clave única; la de detalle, por anime; la del reproductor,
  // por episodio) y se restaura al volver. Las pantallas nuevas empiezan arriba.
  $: sincronizarVista($vista);

  async function sincronizarVista(destino: VistaNombre) {
    if (destino === vistaPrevia) return;
    guardarScroll(claveScroll(vistaPrevia), window.scrollY);
    vistaPrevia = destino;
    await tick();
    window.scrollTo(0, recuperarScroll(claveScroll(destino)));
  }

  function abrirHistorial() {
    drawerOpen.set(true);
  }
</script>

<!--
  Shell (T48): barra a la izquierda + área principal. El scroll sigue siendo el
  de la ventana (min-height, sin overflow en el área) para no romper
  sincronizarVista. Sin barra en el reproductor y en modo estrecho (donde la
  barra pasa a cajón fijo y no ocupa columna): ahí la rejilla pasa a 1fr para
  que el área principal ocupe todo el ancho, como antes.
-->
<div class="app-shell" class:sin-barra={$vista === 'reproductor' || $isNarrow}>
  {#if $vista !== 'reproductor'}
    <BarraLateral />
  {/if}

  <main class="area-principal">
    {#if $isNarrow && $vista !== 'reproductor'}
      <div class="cabecera-estrecha">
        <button type="button" class="hamburguesa" aria-label="Abrir historial" on:click={abrirHistorial}>
          <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" focusable="false">
            <path d="M4 6h16M4 12h16M4 18h16" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        </button>
      </div>
    {/if}

    {#if !puente}
      <p class="aviso-puente">{MENSAJE_PUENTE}</p>
    {/if}

    {#if $vista === 'busqueda'}
      <VistaBusqueda {puente} />
    {:else if $vista === 'detalle'}
      <VistaDetalle />
    {:else}
      <VistaReproductor />
    {/if}
  </main>
</div>

<style>
  .app-shell {
    display: grid;
    grid-template-columns: auto 1fr;
    min-height: 100vh;
  }
  .app-shell.sin-barra {
    grid-template-columns: 1fr;
  }
  .area-principal {
    min-width: 0;
  }
  .cabecera-estrecha {
    text-align: left;
    padding: 8px;
  }
  .hamburguesa {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 8px;
    background: var(--cad-superficie);
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    color: var(--cad-texto);
    cursor: pointer;
  }
  .hamburguesa:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .aviso-puente {
    color: var(--cad-peligro);
    margin: 1rem auto;
    max-width: 640px;
    padding: 10px 14px;
    background: #2a2a35;
    border-radius: 8px;
  }
</style>
