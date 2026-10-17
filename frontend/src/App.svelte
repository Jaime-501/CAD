<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { get } from 'svelte/store';
  import BarraInferior from './BarraInferior.svelte';
  import BarraLateral from './BarraLateral.svelte';
  import { cargarFavoritos, favoritosEstado } from './favoritos';
  import { cargarHistorial } from './historial';
  import { retomar } from './retomar';
  import { disposeEvents, initEvents, search, searchState } from './stores';
  import Toast from './Toast.svelte';
  import { drawerOpen, isNarrow, limpiarBusqueda, pestanaHome } from './ui';
  import {
    claveScroll,
    guardarScroll,
    navegarA,
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
  /** Referencia a la barra inferior (T73) para devolverle el foco. */
  let barraInferior: { enfocarHistorial: () => void } | null = null;

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

  // --- Barra inferior (T73) -------------------------------------------------

  /** La barra inferior solo existe en modo estrecho y fuera del reproductor. */
  $: barraVisible = $isNarrow && $vista !== 'reproductor';

  /**
   * Destino resaltado (T73): en `busqueda` con consulta, "buscar"; en
   * `busqueda` sin consulta, la pestaña del Home (`descubrir`|`favoritos`);
   * en `detalle` —y en el reproductor, donde la barra no se ve—, ninguno.
   */
  $: barraActiva =
    $vista === 'busqueda'
      ? $searchState.query !== ''
        ? ('buscar' as const)
        : $pestanaHome
      : null;

  function cerrarCajonSiAbierto(): void {
    if (get(drawerOpen)) drawerOpen.set(false);
  }

  /** Descubrir / Favoritos: fija la pestaña y vuelve al Home (limpia consulta). */
  function irAHome(pestana: 'descubrir' | 'favoritos'): void {
    cerrarCajonSiAbierto();
    pestanaHome.set(pestana);
    if (get(vista) !== 'busqueda') navegarA('busqueda');
    if (get(searchState).query !== '') {
      void search('');
      limpiarBusqueda.update((n) => n + 1);
    }
  }

  /** Buscar: vuelve a la búsqueda y enfoca el input (sin desplazarse). */
  async function irABuscar(): Promise<void> {
    cerrarCajonSiAbierto();
    if (get(vista) !== 'busqueda') navegarA('busqueda');
    await tick();
    window.scrollTo(0, 0);
    document.getElementById('buscador-anime')?.focus({ preventScroll: true });
  }

  /** Historial: abre el cajón existente. */
  function irAHistorial(): void {
    drawerOpen.set(true);
  }

  /** Al cerrar el cajón desde la barra, el foco vuelve al destino Historial. */
  async function enfocarHistorialBarra(): Promise<void> {
    await tick();
    barraInferior?.enfocarHistorial();
  }
</script>

<!--
  Shell (T48): barra a la izquierda + área principal. El scroll sigue siendo el
  de la ventana (min-height, sin overflow en el área) para no romper
  sincronizarVista. Sin barra en el reproductor y en modo estrecho (donde la
  barra pasa a cajón fijo y no ocupa columna): ahí la rejilla pasa a 1fr para
  que el área principal ocupe todo el ancho, como antes.

  T73: en modo estrecho, la navegación es la barra inferior (`BarraInferior`);
  `.raiz` (`display: contents`) expone `--bar-offset` (alto de la barra +
  safe-area) para que el `Toast` quede por encima de ella.
-->
<div
  class="raiz"
  style:--bar-offset={barraVisible ? 'calc(var(--bar-h) + var(--sab))' : '0px'}
>
  <div class="app-shell" class:sin-barra={$vista === 'reproductor' || $isNarrow}>
    {#if $vista !== 'reproductor'}
      <BarraLateral onOpen={retomar} onCerrarCajon={enfocarHistorialBarra} />
    {/if}

    <main class="area-principal" class:con-barra={barraVisible}>
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

  {#if barraVisible}
    <BarraInferior
      bind:this={barraInferior}
      activo={barraActiva}
      onDescubrir={() => irAHome('descubrir')}
      onBuscar={irABuscar}
      onFavoritos={() => irAHome('favoritos')}
      onHistorial={irAHistorial}
    />
  {/if}

  <!-- Toast de una ranura (T51): montado una sola vez, fijo abajo-centro. -->
  <Toast />
</div>

<style>
  /* T73: contenedor sin caja (display: contents) para exponer --bar-offset. */
  .raiz {
    display: contents;
  }
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
  /* Hueco inferior para que nada quede bajo la barra (T73). */
  .area-principal.con-barra {
    padding-bottom: calc(var(--bar-h) + var(--sab) + var(--sp-4));
  }
  .aviso-puente {
    color: var(--danger);
    margin: 1rem auto;
    max-width: 640px;
    padding: 10px 14px;
    background: var(--bg-2);
    border-radius: var(--r-md);
  }
</style>
