<script lang="ts">
  /**
   * Tarjeta del historial en la barra lateral (T49/T50/T51): portada, título,
   * episodio/tiempo y barra de progreso. El clic va a `onOpen` (`alRetomar`
   * de la barra); mientras `resolviendo` la tarjeta queda desactivada y
   * muestra "Buscando…", y `mensaje` pinta el resultado durante 4 s.
   *
   * T51 (borrado individual): botón ✕ a la derecha (visible con ratón o con
   * foco dentro, y siempre en táctil), deslizar a izquierda o derecha con
   * `use:deslizar` y `Delete` con la tarjeta enfocada llaman a `onBorrar`
   * (borra el anime entero de ese proveedor, con "Deshacer").
   */
  import type { models } from './api';
  import { deslizar } from './deslizar';
  import { formatearTiempo, fraccionProgreso, numeroDeEpisodio, tituloVisible } from './historial';

  export let item: models.WatchProgress;
  export let onOpen: (item: models.WatchProgress) => void = () => {};
  /** T50: mientras se resuelve "retomar" la tarjeta queda desactivada. */
  export let resolviendo: boolean = false;
  /** T50: aviso bajo la barra (p. ej. el error de la resolución). */
  export let mensaje: string = '';
  /** T51: borra el historial de este anime (todos sus episodios). */
  export let onBorrar: () => void = () => {};

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

  /** Delete con la tarjeta enfocada: mismo flujo que el ✕ (T51). */
  function alTeclarTarjeta(e: KeyboardEvent): void {
    if (e.key !== 'Delete') return;
    e.preventDefault();
    e.stopPropagation();
    onBorrar();
  }
</script>

<!--
  El envoltorio lleva el gesto de deslizar y contiene a los DOS botones: el ✕
  es hermano del botón de retomar (no puede vivir dentro: un <button> no
  anida otro), así que su clic nunca dispara "retomar".
-->
<div class="envoltorio" use:deslizar={{ onBorrar, umbral: 96 }}>
  <button
    type="button"
    class="tarjeta"
    disabled={resolviendo}
    on:click={() => onOpen(item)}
    on:keydown={alTeclarTarjeta}
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

  <button
    type="button"
    class="borrar"
    disabled={resolviendo}
    aria-label="Borrar historial de {titulo}"
    on:click={onBorrar}
  >
    <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true" focusable="false">
      <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" fill="none" />
    </svg>
  </button>
</div>

<style>
  .envoltorio {
    position: relative;
    width: 100%;
    margin: 0 0 6px;
  }
  .tarjeta {
    box-sizing: border-box;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    /* Margen derecho para que el texto no quede bajo el ✕ (T51). */
    padding: 8px 36px 8px 8px;
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
  /* ✕ de borrar (T51): oculto hasta hover o foco dentro del envoltorio; en
     pantallas táctiles (sin hover) siempre visible. `visibility` lo saca del
     orden de tabulación y de los clics mientras está oculto. */
  .borrar {
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 6px;
    background: var(--cad-superficie);
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    color: var(--cad-texto-suave);
    cursor: pointer;
    visibility: hidden;
    opacity: 0;
  }
  .envoltorio:hover .borrar,
  .envoltorio:focus-within .borrar {
    visibility: visible;
    opacity: 1;
  }
  .borrar:hover:not(:disabled) {
    color: var(--cad-peligro);
    border-color: var(--cad-peligro);
  }
  .borrar:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
    visibility: visible;
    opacity: 1;
  }
  .borrar:disabled {
    cursor: default;
  }
  @media (hover: none) {
    .borrar {
      visibility: visible;
      opacity: 1;
    }
  }
</style>
