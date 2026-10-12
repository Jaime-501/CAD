<script lang="ts">
  /**
   * Tarjeta del historial en la barra lateral (T49): portada, título,
   * episodio/tiempo y barra de progreso. T50: el clic va a `onOpen`
   * (`alRetomar` de la barra); mientras `resolviendo` la tarjeta queda
   * desactivada y muestra "Buscando…", y `mensaje` pinta el resultado
   * ("Completado", "Al día", "No se pudo retomar") durante 4 s.
   */
  import type { models } from './api';
  import { formatearTiempo, fraccionProgreso, numeroDeEpisodio, tituloVisible } from './historial';

  export let item: models.WatchProgress;
  export let onOpen: (item: models.WatchProgress) => void = () => {};
  /** T50: mientras se resuelve "retomar" la tarjeta queda desactivada. */
  export let resolviendo: boolean = false;
  /** T50: aviso bajo la barra (p. ej. el error de la resolución). */
  export let mensaje: string = '';

  /** La portada puede venir vacía o fallar al cargar: se pinta la inicial. */
  let falloPortada = false;

  $: titulo = tituloVisible(item);
  $: inicial = titulo.trim().charAt(0).toUpperCase() || '—';
  $: episodio = numeroDeEpisodio(item);
  $: visto = item.is_watched;
  $: tiempo = visto ? 'Visto ✓' : formatearTiempo(item.current_time);
  $: secundaria = [episodio, tiempo].filter((parte) => parte !== '').join(' · ');
  $: fraccion = fraccionProgreso(item);
  $: porcentaje = Math.round(fraccion * 100);
</script>

<button
  type="button"
  class="tarjeta"
  disabled={resolviendo}
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
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px;
    margin: 0 0 6px;
    background: transparent;
    border: none;
    border-radius: var(--cad-radio);
    color: var(--cad-texto);
    font-family: inherit;
    text-align: left;
    cursor: pointer;
  }
  .tarjeta:hover:not(:disabled) {
    background: var(--cad-borde);
  }
  .tarjeta:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: -2px;
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
    border-radius: var(--cad-radio);
    object-fit: cover;
    background: var(--cad-fondo);
  }
  .sin-portada {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    font-weight: bold;
    color: var(--cad-texto-suave);
  }
  .datos {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .titulo {
    font-size: 13px;
    line-height: 1.3;
    /* Máximo 2 líneas con elipsis. */
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .secundaria {
    font-size: 12px;
    color: var(--cad-texto-suave);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .barra {
    display: block;
    height: 4px;
    background: var(--cad-borde);
    border-radius: 4px;
    overflow: hidden;
  }
  .barra-relleno {
    display: block;
    height: 100%;
    background: var(--cad-acento);
  }
  .mensaje {
    font-size: 11px;
    color: var(--cad-texto-suave);
  }
</style>
