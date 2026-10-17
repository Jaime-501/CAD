<script lang="ts">
  /**
   * Barra lateral del historial (T48/T49): grupos por proveedor con tarjetas
   * de retomar. Habla con `ui.ts` (`sidebarCollapsed`, `drawerOpen`, `isNarrow`)
   * y con `historial.ts` (`$historial`, `$historialEstado`, `cargarHistorial`).
   *
   * - Pantalla ancha: columna de la rejilla, sticky a `top:0` con `height:100vh`;
   *   el botón contrae/expandir (300px ↔ 56px) y el estado se persiste; la
   *   barra contraída (56px) oculta el contenido.
   * - Modo estrecho (`isNarrow`): cajón fijo sobre el contenido; `sidebarCollapsed`
   *   se ignora, el botón cierra, el foco va al botón al abrir y `Escape` cierra.
   *
   * T50 (retomar con un clic): `onOpen` lo cablea `App.svelte` con `retomar`;
   * aquí vive el estado local del clic (`resolviendoId`, `mensajes`) —una
   * resolución a la vez—, el mapeo de resultados a mensajes y el cierre del
   * cajón al navegar en modo estrecho.
   *
   * T51 (borrado individual): cada tarjeta recibe `onBorrar`, que limpia el
   * mensaje de esa tarjeta y llama a `borrarAnime(proveedor, anime_id)` (el
   * backend borra solo ese proveedor; el toast ofrece "Deshacer").
   *
   * T52 (borrado masivo): papelera en el encabezado de cada grupo
   * (`borrarProveedor`, con Deshacer en el toast) y pie fijo con
   * "Borrar todo el historial" bajo doble confirmación local (6 s, sin
   * `window.confirm`, sin Deshacer; `Escape` cancela y el foco cae en
   * "Cancelar").
   */
  import { onDestroy, tick } from 'svelte';
  import { get } from 'svelte/store';
  import {
    agruparPorProveedor,
    borrarAnime,
    borrarProveedor,
    borrarTodo,
    cargarHistorial,
    historial,
    historialEstado,
    proveedorDe,
    type GrupoHistorial,
  } from './historial';
  import type { RetomarResultado } from './retomar';
  import { drawerOpen, isNarrow, sidebarCollapsed } from './ui';
  import TarjetaHistorial from './TarjetaHistorial.svelte';
  import type { models } from './api';

  /** Resolución del clic: la cablea `App.svelte` con `retomar` (T50). */
  export let onOpen: (item: models.WatchProgress) => Promise<RetomarResultado>;

  /**
   * T73: se llama al cerrar el cajón con Escape, velo o botón (no al cerrarlo
   * al retomar) para que `App.svelte` devuelva el foco al destino Historial de
   * la barra inferior.
   */
  export let onCerrarCajon: () => void = () => {};

  /** Duración del mensaje de tarjeta ("Completado"…) antes de borrarlo (T50). */
  const MENSAJE_MS = 4000;
  /** Espera de la confirmación de "borrar todo" antes de cancelarse sola (T52). */
  const CONFIRMACION_MS = 6000;

  let boton: HTMLButtonElement | null = null;
  /** Clave `proveedor:anime_id` que se está resolviendo; null = ninguna. */
  let resolviendoId: string | null = null;
  let mensajes: Record<string, string> = {};
  let temporizadores: Record<string, ReturnType<typeof setTimeout>> = {};
  /** Confirmación del "borrar todo" en curso (T52). */
  let confirmando = false;
  let temporizadorConfirmacion: ReturnType<typeof setTimeout> | null = null;
  let botonCancelar: HTMLButtonElement | null = null;

  onDestroy(() => {
    for (const temporizador of Object.values(temporizadores)) clearTimeout(temporizador);
    temporizadores = {};
    if (temporizadorConfirmacion !== null) clearTimeout(temporizadorConfirmacion);
    temporizadorConfirmacion = null;
  });

  function claveDe(item: models.WatchProgress): string {
    return `${proveedorDe(item)}:${item.anime_id}`;
  }

  /** Clic en una tarjeta: si ya hay una resolución en curso, se ignora (T50). */
  async function alRetomar(item: models.WatchProgress): Promise<void> {
    if (resolviendoId !== null) return;
    const clave = claveDe(item);
    resolviendoId = clave;
    borrarMensaje(clave);

    let res: RetomarResultado;
    try {
      res = await onOpen(item);
    } catch (e) {
      res = { ok: false, error: e instanceof Error ? e.message : String(e) };
    }
    resolviendoId = null;

    if (!res.ok) {
      console.warn('[BarraLateral] no se pudo retomar:', res.error);
      ponerMensaje(clave, 'No se pudo retomar');
      return;
    }
    if (res.accion === 'completed') ponerMensaje(clave, 'Completado');
    else if (res.accion === 'up_to_date') ponerMensaje(clave, 'Al día');
    // Las 4 acciones navegan: en modo estrecho el cajón se cierra (T48/T50).
    if (get(isNarrow)) drawerOpen.set(false);
  }

  function ponerMensaje(clave: string, texto: string): void {
    borrarMensaje(clave);
    mensajes = { ...mensajes, [clave]: texto };
    temporizadores[clave] = setTimeout(() => borrarMensaje(clave), MENSAJE_MS);
  }

  function borrarMensaje(clave: string): void {
    clearTimeout(temporizadores[clave]);
    delete temporizadores[clave];
    if (!(clave in mensajes)) return;
    mensajes = Object.fromEntries(Object.entries(mensajes).filter(([k]) => k !== clave));
  }

  /** ✕ / deslizar / Delete (T51): borra el anime de SU proveedor, sin diálogo. */
  function alBorrar(item: models.WatchProgress): void {
    borrarMensaje(claveDe(item));
    void borrarAnime(proveedorDe(item), item.anime_id);
  }

  /** Papelera del encabezado de grupo (T52): borra el proveedor; hay Deshacer. */
  function alBorrarProveedor(grupo: GrupoHistorial): void {
    void borrarProveedor(grupo.provider, grupo.nombre);
  }

  /** Entra en la confirmación del "borrar todo" y enfoca "Cancelar" (T52). */
  async function entrarConfirmacion(): Promise<void> {
    confirmando = true;
    if (temporizadorConfirmacion !== null) clearTimeout(temporizadorConfirmacion);
    temporizadorConfirmacion = setTimeout(() => cancelarConfirmacion(), CONFIRMACION_MS);
    await tick();
    botonCancelar?.focus();
  }

  function cancelarConfirmacion(): void {
    if (temporizadorConfirmacion !== null) clearTimeout(temporizadorConfirmacion);
    temporizadorConfirmacion = null;
    confirmando = false;
  }

  function confirmarBorrado(): void {
    cancelarConfirmacion();
    void borrarTodo();
  }

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
      onCerrarCajon();
      return;
    }
    sidebarCollapsed.update((v) => !v);
  }

  function cerrarCajon() {
    drawerOpen.set(false);
    onCerrarCajon();
  }

  function alTeclar(e: KeyboardEvent) {
    if (e.key !== 'Escape') return;
    // Escape con la confirmación abierta la cancela (T52); si no, cierra el cajón.
    if (confirmando) {
      cancelarConfirmacion();
      return;
    }
    if (get(isNarrow) && get(drawerOpen)) {
      drawerOpen.set(false);
      onCerrarCajon();
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
        {#if $historialEstado === 'error'}
          <p class="vacio">No se pudo cargar el historial.</p>
          <button type="button" class="reintentar" on:click={() => void cargarHistorial()}>
            Reintentar
          </button>
        {:else if ($historialEstado === 'idle' || $historialEstado === 'loading') && $historial.length === 0}
          {#each [0, 1, 2] as indice (indice)}
            <div class="esqueleto" aria-hidden="true">
              <span class="esqueleto-portada"></span>
              <span class="esqueleto-datos">
                <span class="esqueleto-linea ancha"></span>
                <span class="esqueleto-linea estrecha"></span>
                <span class="esqueleto-barra"></span>
              </span>
            </div>
          {/each}
        {:else if $historial.length === 0}
          <p class="vacio">Aún no hay nada que retomar.</p>
        {:else}
          {#each agruparPorProveedor($historial) as grupo (grupo.provider)}
            <section class="grupo" aria-label="Historial de {grupo.nombre}">
              <h2 class="grupo-titulo">
                <span>{grupo.nombre}</span>
                <span class="grupo-contador">{grupo.items.length}</span>
                <button
                  type="button"
                  class="papelera"
                  aria-label="Borrar historial de {grupo.nombre}"
                  on:click={() => alBorrarProveedor(grupo)}
                >
                  <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true" focusable="false">
                    <path
                      d="M4 7h16M10 11v6M14 11v6M6 7l1 12a2 2 0 002 2h6a2 2 0 002-2l1-12M9 7V4h6v3"
                      stroke="currentColor"
                      stroke-width="2"
                      fill="none"
                    />
                  </svg>
                </button>
              </h2>
              {#each grupo.items as item (claveDe(item))}
                <TarjetaHistorial
                  {item}
                  onOpen={alRetomar}
                  onBorrar={() => alBorrar(item)}
                  resolviendo={resolviendoId === claveDe(item)}
                  mensaje={mensajes[claveDe(item)] ?? ''}
                />
              {/each}
            </section>
          {/each}
        {/if}
      {/if}
    </div>
    {#if ($isNarrow || !$sidebarCollapsed) && $historial.length > 0}
      <div class="pie">
        {#if !confirmando}
          <button type="button" class="borrar-todo" on:click={() => void entrarConfirmacion()}>
            Borrar todo el historial
          </button>
        {:else}
          <p class="confirmar-texto">¿Borrar todo el historial?</p>
          <div class="confirmar-acciones">
            <button type="button" class="si-borrar" on:click={confirmarBorrado}>
              Sí, borrar todo
            </button>
            <button type="button" class="cancelar" bind:this={botonCancelar} on:click={cancelarConfirmacion}>
              Cancelar
            </button>
          </div>
        {/if}
      </div>
    {/if}
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
    /* T73b: fondo OPACO del cajón (evita que el contenido y el pie se
       transparenten). Opaque también en escritorio. */
    background: var(--bg-1);
    color: var(--text-1);
    border-right: 1px solid var(--border);
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
    border-bottom: 1px solid var(--border);
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
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--text-1);
    cursor: pointer;
  }
  .alternar:focus-visible {
    outline: 2px solid var(--brand-400);
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
    color: var(--text-3);
  }
  .reintentar {
    box-sizing: border-box;
    padding: 6px 12px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--brand-400);
    font-family: inherit;
    font-weight: bold;
    cursor: pointer;
  }
  .reintentar:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  .grupo {
    margin: 0 0 14px;
  }
  .grupo-titulo {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0 0 8px;
    font-size: 12px;
    font-weight: bold;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-3);
  }
  .grupo-contador {
    color: var(--text-3);
  }
  /* Papelera del grupo (T52): siempre visible (regla P6), al final de la fila. */
  .papelera {
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin-left: auto;
    padding: 4px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--text-3);
    cursor: pointer;
  }
  .papelera:hover {
    color: var(--danger);
    border-color: var(--danger);
  }
  .papelera:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  /* Pie fijo con el "borrar todo" (T52): solo con historial y barra expandida. */
  .pie {
    box-sizing: border-box;
    padding: var(--sp-3);
    /* T73: hueco para el área segura inferior del dispositivo. */
    padding-bottom: calc(var(--sp-3) + var(--sab));
    border-top: 1px solid var(--border);
  }
  .borrar-todo {
    box-sizing: border-box;
    width: 100%;
    padding: 6px 12px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--text-3);
    font-family: inherit;
    text-align: left;
    cursor: pointer;
  }
  .borrar-todo:hover {
    border-color: var(--danger);
    color: var(--danger);
  }
  .borrar-todo:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  .confirmar-texto {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--text-1);
  }
  .confirmar-acciones {
    display: flex;
    gap: 8px;
  }
  .si-borrar {
    box-sizing: border-box;
    flex: 1;
    min-width: 0;
    padding: 6px 8px;
    background: transparent;
    border: 1px solid var(--danger);
    border-radius: var(--r-md);
    color: var(--danger);
    font-family: inherit;
    font-weight: bold;
    cursor: pointer;
  }
  .cancelar {
    box-sizing: border-box;
    flex: 1;
    min-width: 0;
    padding: 6px 8px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--text-1);
    font-family: inherit;
    cursor: pointer;
  }
  .si-borrar:focus-visible,
  .cancelar:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  /* Fantasmas de carga (T49): mismas medidas que la tarjeta real. */
  .esqueleto {
    box-sizing: border-box;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px;
    margin: 0 0 6px;
  }
  .esqueleto-portada {
    flex-shrink: 0;
    width: 48px;
    height: 68px;
    border-radius: var(--r-md);
    background: var(--border-strong);
  }
  .esqueleto-datos {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .esqueleto-linea,
  .esqueleto-barra {
    display: block;
    border-radius: var(--r-sm);
    background: var(--border-strong);
  }
  .esqueleto-linea.ancha {
    width: 80%;
    height: 12px;
  }
  .esqueleto-linea.estrecha {
    width: 50%;
    height: 10px;
  }
  .esqueleto-barra {
    width: 100%;
    height: 4px;
  }
  @media (prefers-reduced-motion: no-preference) {
    .esqueleto-linea,
    .esqueleto-barra,
    .esqueleto-portada {
      animation: pulsar 1.4s ease-in-out infinite;
    }
  }
  @keyframes pulsar {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.45;
    }
  }
  /* Modo estrecho: cajón fijo sobre el contenido (el ancho colapsado se ignora). */
  .barra.cajon {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: min(320px, 85vw);
    height: 100dvh;
    z-index: 50;
    border-right: none;
    /* T73b: solo en el cajón, se separa del velo con la sombra de panel. */
    box-shadow: var(--sh-panel);
    transition: none;
  }
  /* T73: en el cajón, los controles del historial alcanzan el objetivo táctil
     mínimo (44 px) en ancho y alto. */
  .barra.cajon .alternar,
  .barra.cajon .papelera,
  .barra.cajon .borrar-todo,
  .barra.cajon .reintentar,
  .barra.cajon .si-borrar,
  .barra.cajon .cancelar {
    min-width: var(--tap);
    min-height: var(--tap);
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
    outline: 2px solid var(--brand-400);
    outline-offset: -2px;
  }
  @media (prefers-reduced-motion: reduce) {
    .barra {
      transition: none;
    }
  }
</style>
