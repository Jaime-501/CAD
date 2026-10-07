<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { disposeEvents, initEvents } from './stores';
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
</script>

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

<style>
  .aviso-puente {
    color: #f87171;
    margin: 1rem auto;
    max-width: 640px;
    padding: 10px 14px;
    background: #2a2a35;
    border-radius: 8px;
  }
</style>
