<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { get } from 'svelte/store';
  import BarraLateral from './BarraLateral.svelte';
  import { cargarFavoritos, favoritosEstado } from './favoritos';
  import { cargarHistorial } from './historial';
  import { retomar } from './retomar';
  import { disposeEvents, initEvents } from './stores';
  import Toast from './Toast.svelte';
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
  /** Para refrescar el historial al salir del reproductor (T49). */
  let vistaPreviaHistorial: VistaNombre = 'busqueda';

  onMount(() => {
    // window.go solo lo inyecta el dev server de Wails (:34115),
    // no el servidor de Vite (:5173)
    puente = typeof window !== 'undefined' && Boolean((window as any).go);
    initEvents();
    void cargarHistorial();
    void cargarFavoritosAlArranque();
  });

  /**
   * T58: los favoritos se cargan al montar (mismo patrón que el historial)
   * para que el contador de la pestaña "★ Favoritos (N)" no arranque en 0
   * hasta que alguien abra la pestaña. `cargarFavoritos` no lanza (deja el
   * estado en `error`); si el primer intento falla (p. ej. el backend aún no
   * respondió) se reintenta una vez pasados 500 ms; si vuelve a fallar, la
   * cuadrícula de la pestaña conserva su botón "Reintentar".
   */
  async function cargarFavoritosAlArranque(): Promise<void> {
    await cargarFavoritos();
    if (get(favoritosEstado) !== 'error') return;
    await new Promise((resuelve) => setTimeout(resuelve, 500));
    await cargarFavoritos();
  }

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

  // El historial se refresca al salir del reproductor (T49): `tick` deja que
  // el reproductor se desmonte y lance su guardado final antes de leer.
  $: refrescarHistorialAlSalir($vista);

  async function refrescarHistorialAlSalir(destino: VistaNombre) {
    if (destino === vistaPreviaHistorial) return;
    const saliaDelReproductor = vistaPreviaHistorial === 'reproductor';
    vistaPreviaHistorial = destino;
    if (!saliaDelReproductor) return;
    await tick();
    await cargarHistorial();
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
    <BarraLateral onOpen={retomar} />
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

<!-- Toast de una ranura (T51): montado una sola vez, fijo abajo-centro. -->
<Toast />

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
