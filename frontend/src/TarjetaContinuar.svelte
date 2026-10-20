<script lang="ts">
  /**
   * Tarjeta de "Continuar viendo" del Home móvil (T77): la misma información
   * que `TarjetaHistorial` (portada, título, episodio/tiempo, barra de
   * progreso) pero en formato horizontal de ancho fijo (240 px) para el
   * carrusel de la fila del Home. Reutiliza los helpers de `historial.ts`, sin
   * lógica nueva.
   *
   * - El clic va a `onOpen` (el `alRetomar` de `FilaContinuar`); mientras
   *   `resolviendo` la tarjeta queda desactivada y muestra "Buscando…", y
   *   `mensaje` pinta el resultado durante 4 s.
   * - La portada puede venir vacía o fallar al cargar: se pinta la inicial del
   *   título (misma regla que `TarjetaHistorial`).
   */
  import type { models } from './api';
  import {
    formatearTiempo,
    fraccionProgreso,
    numeroDeEpisodio,
    proveedorDe,
    NOMBRE_PROVEEDOR,
    tituloVisible,
  } from './historial';

  export let item: models.WatchProgress;
  export let onOpen: (item: models.WatchProgress) => void = () => {};
  /** T77: mientras se resuelve "retomar" la tarjeta queda desactivada. */
  export let resolviendo: boolean = false;
  /** T77: aviso bajo la barra (p. ej. el error de la resolución). */
  export let mensaje: string = '';

  /** La portada puede venir vacía o fallar al cargar: se pinta la inicial. */
  let falloPortada = false;

  $: titulo = tituloVisible(item);
  $: inicial = titulo.trim().charAt(0).toUpperCase() || '—';
  $: episodio = numeroDeEpisodio(item);
  $: visto = item.is_watched;
  $: tiempo = visto ? 'Visto ✓' : formatearTiempo(item.current_time);
  $: secundaria = [episodio, tiempo].filter((parte) => parte !== '').join(' · ');
  $: proveedor = proveedorDe(item);
  $: nombreProveedor = NOMBRE_PROVEEDOR[proveedor] ?? proveedor;
  $: fraccion = fraccionProgreso(item);
  $: porcentaje = Math.round(fraccion * 100);
</script>

<button
  type="button"
  class="tarjeta"
  disabled={resolviendo}
  title={titulo}
  on:click={() => onOpen(item)}
>
  {#if item.cover_url !== '' && !falloPortada}
    <img
      class="portada"
      src={item.cover_url}
      alt=""
      loading="lazy"
      on:error={() => (falloPortada = true)}
    />
  {:else}
    <span class="portada sin-portada" aria-hidden="true">{inicial}</span>
  {/if}

  <span class="datos">
    <span class="titulo">{titulo}</span>
    {#if secundaria !== ''}
      <span class="secundaria">{secundaria}</span>
    {/if}
    {#if nombreProveedor !== ''}
      <span class="proveedor">{nombreProveedor}</span>
    {/if}
    <span
      class="barra"
      role="progressbar"
      aria-label="Progreso"
      aria-valuemin="0"
      aria-valuemax="100"
      aria-valuenow={porcentaje}
    >
      <span class="barra-relleno" style="width: {porcentaje}%"></span>
    </span>
    {#if resolviendo}
      <span class="mensaje" aria-live="polite">Buscando…</span>
    {:else if mensaje !== ''}
      <span class="mensaje" aria-live="polite">{mensaje}</span>
    {/if}
  </span>
</button>

<style>
  .tarjeta {
    box-sizing: border-box;
    /* Ancho fijo del carrusel (T77): el botón ES el elemento flex. */
    flex: 0 0 240px;
    width: 240px;
    min-height: var(--tap);
    display: flex;
    align-items: center;
    gap: 10px;
    padding: var(--sp-2);
    background: var(--bg-1);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    box-shadow: var(--sh-card);
    color: var(--text-1);
    font-family: inherit;
    text-align: left;
    cursor: pointer;
    scroll-snap-align: start;
    transition:
      transform var(--dur-fast) var(--ease-std),
      border-color var(--dur-fast) var(--ease-std),
      box-shadow var(--dur-fast) var(--ease-std);
  }
  .tarjeta:active {
    transform: scale(0.97);
  }
  .tarjeta:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  .tarjeta:disabled {
    cursor: default;
    opacity: 0.6;
  }
  .portada {
    box-sizing: border-box;
    flex-shrink: 0;
    width: 48px;
    height: 68px;
    border-radius: var(--r-md);
    object-fit: cover;
    background: var(--bg-0);
  }
  .sin-portada {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: var(--fs-xl);
    font-weight: var(--fw-bold);
    color: var(--text-3);
  }
  .datos {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  .titulo {
    font-size: var(--fs-sm);
    line-height: 1.3;
    /* Máximo 2 líneas con elipsis. */
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .secundaria,
  .proveedor {
    font-size: var(--fs-xs);
    color: var(--text-3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .barra {
    display: block;
    height: 4px;
    background: var(--border-strong);
    border-radius: var(--r-sm);
    overflow: hidden;
  }
  .barra-relleno {
    display: block;
    height: 100%;
    background: var(--brand-500);
  }
  .mensaje {
    font-size: var(--fs-xs);
    color: var(--text-3);
  }
  /* Hover solo con puntero fino (P6); en táctil manda `:active`. */
  @media (hover: hover) and (pointer: fine) {
    .tarjeta:hover:not(:disabled) {
      border-color: var(--brand-400);
      box-shadow: var(--sh-card-hover);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .tarjeta {
      transition: none;
    }
    .tarjeta:hover,
    .tarjeta:active {
      transform: none;
    }
  }
</style>
