<script lang="ts">
  import type { models } from './api';

  /** Tarjeta de un episodio del grid (T26): miniatura, estado y apertura. */
  export let episodio: models.Episode;
  /** Progreso de este episodio en `progressMap` (clave `episode_id`); sin entrada = sin progreso. */
  export let progreso: models.WatchProgress | undefined = undefined;
  export let abrir: (ep: models.Episode) => void;

  /** La miniatura puede venir vacía o fallar al cargar: se pinta un hueco con el número. */
  let falloMiniatura = false;

  $: visto = progreso?.is_watched ?? false;
  $: segundos = progreso?.current_time ?? 0;
  $: duracion = progreso?.total_duration ?? 0;
  $: enProgreso = !visto && segundos > 0;
  $: porcentaje = duracion > 0 ? Math.min(100, Math.max(0, (segundos / duracion) * 100)) : 0;

  $: insignia =
    episodio.number > 0
      ? episodio.kind === 'special'
        ? `Especial ${episodio.number}`
        : `Ep. ${episodio.number}`
      : episodio.kind === 'movie'
        ? 'Película'
        : 'Especial';
  $: titulo = episodio.title.trim() !== '' ? episodio.title : insignia;
  $: hueco = episodio.number > 0 ? String(episodio.number) : '—';
</script>

<button type="button" class="tarjeta" on:click={() => abrir(episodio)}>
  <div class="miniatura">
    {#if episodio.thumbnail !== '' && !falloMiniatura}
      <img
        src={episodio.thumbnail}
        alt={titulo}
        loading="lazy"
        on:error={() => (falloMiniatura = true)}
      />
    {:else}
      <div class="sin-miniatura">{hueco}</div>
    {/if}
    <span class="insignia">{insignia}</span>
  </div>

  <span class="titulo">{titulo}</span>

  {#if visto}
    <span class="visto">Visto</span>
  {:else if enProgreso}
    <div
      class="barra"
      role="progressbar"
      aria-label="Progreso del episodio"
      aria-valuemin="0"
      aria-valuemax="100"
      aria-valuenow={Math.round(porcentaje)}
    >
      <span style="width: {porcentaje}%"></span>
    </div>
    <span class="porcentaje">{Math.round(porcentaje)}%</span>
  {/if}
</button>

<style>
  .tarjeta {
    /* Rendimiento con listas largas (366 episodios): solo se pinta lo cercano
       y se reserva el hueco de cada tarjeta para que el scroll no salte. */
    content-visibility: auto;
    contain-intrinsic-size: auto 170px;

    background: var(--surface-card);
    border: none;
    border-radius: var(--r-md);
    overflow: hidden;
    text-align: left;
    transition: transform 0.2s;
    cursor: pointer;
    padding: 0;
    color: var(--text-1);
    font-family: inherit;
    display: flex;
    flex-direction: column;
  }
  .tarjeta:hover {
    transform: translateY(-5px);
  }
  .miniatura {
    position: relative;
    height: 118px;
    background: var(--bg-2);
  }
  .miniatura img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .sin-miniatura {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 34px;
    font-weight: bold;
    color: #52525b;
    background: repeating-linear-gradient(
      45deg,
      #2a2a35,
      #2a2a35 12px,
      #26262e 12px,
      #26262e 24px
    );
  }
  .insignia {
    position: absolute;
    left: 8px;
    bottom: 8px;
    background: var(--surface-panel);
    color: var(--brand-400);
    font-size: 11px;
    font-weight: bold;
    padding: 3px 7px;
    border-radius: var(--r-sm);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .titulo {
    padding: 10px 10px 6px;
    font-size: 14px;
    line-height: 1.3;
    min-height: 2.6em;
  }
  .visto {
    margin: 0 10px 10px;
    align-self: flex-start;
    background: #14532d;
    color: var(--brand-400);
    font-size: 12px;
    font-weight: bold;
    padding: 3px 8px;
    border-radius: var(--r-sm);
  }
  .barra {
    height: 6px;
    margin: 0 10px;
    background: var(--border-strong);
    border-radius: var(--r-sm);
    overflow: hidden;
  }
  .barra span {
    display: block;
    height: 100%;
    background: var(--brand-500);
    transition: width 0.3s;
  }
  .porcentaje {
    padding: 4px 10px 10px;
    font-size: 11px;
    color: var(--text-3);
  }
</style>
