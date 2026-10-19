<script lang="ts">
  import type { models } from './api';
  import Icono from './Icono.svelte';

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
    {#if enProgreso}
      <div
        class="franja"
        role="progressbar"
        aria-label="Progreso del episodio"
        aria-valuemin="0"
        aria-valuemax="100"
        aria-valuenow={Math.round(porcentaje)}
      >
        <span style="width: {porcentaje}%"></span>
      </div>
    {/if}
  </div>

  <div class="cuerpo">
    <span class="titulo">{titulo}</span>
    {#if visto}
      <span class="visto">
        <Icono nombre="check" tam={16} />
        Visto
      </span>
    {:else if enProgreso}
      <span class="porcentaje">{Math.round(porcentaje)} %</span>
    {/if}
  </div>
</button>

<style>
  .tarjeta {
    /* Rendimiento con listas largas (366 episodios): solo se pinta lo cercano
       y se reserva el hueco de cada tarjeta para que el scroll no salte. */
    content-visibility: auto;
    contain-intrinsic-size: auto 170px;

    box-sizing: border-box;
    height: 170.39px;
    background: var(--surface-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    overflow: hidden;
    text-align: left;
    transition: transform var(--dur-fast) var(--ease-std), box-shadow var(--dur-fast) var(--ease-std);
    cursor: pointer;
    padding: 0;
    color: var(--text-1);
    font: inherit;
    display: flex;
    flex-direction: column;
  }
  .miniatura {
    position: relative;
    flex: 0 1 auto;
    min-height: 0;
    aspect-ratio: 16 / 9;
    background: var(--bg-2);
    overflow: hidden;
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
    font-size: var(--fs-2xl);
    font-weight: var(--fw-bold);
    color: var(--text-3);
    background: repeating-linear-gradient(
      45deg,
      var(--bg-2),
      var(--bg-2) var(--sp-3),
      var(--bg-1) var(--sp-3),
      var(--bg-1) calc(2 * var(--sp-3))
    );
  }
  .insignia {
    position: absolute;
    left: var(--sp-2);
    bottom: var(--sp-2);
    background: var(--surface-panel);
    color: var(--brand-400);
    font-size: var(--fs-xs);
    font-weight: var(--fw-bold);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--r-sm);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .franja {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: var(--sp-1);
    background: var(--border-strong);
    overflow: hidden;
  }
  .franja span {
    display: block;
    height: 100%;
    background: var(--brand-500);
    transition: width var(--dur) var(--ease-std);
  }
  .cuerpo {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    padding: var(--sp-1) var(--sp-3);
  }
  .titulo {
    font-size: var(--fs-sm);
    line-height: var(--lh-tight);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .visto {
    align-self: flex-start;
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
    min-height: calc(var(--fs-md) + var(--sp-1));
    padding: 0 var(--sp-2);
    background: var(--surface-chip);
    color: var(--brand-400);
    border: 1px solid var(--border-accent);
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
  }
  .porcentaje {
    font-size: var(--fs-xs);
    line-height: var(--lh-tight);
    color: var(--text-3);
  }
  @media (hover: hover) and (pointer: fine) {
    .tarjeta:hover {
      transform: translateY(-3px);
      box-shadow: var(--sh-card-hover);
    }
  }
  .tarjeta:active {
    transform: scale(0.97);
  }
  .tarjeta:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  /* Modo estrecho (T81): altura constante por rango de columnas, ver medida en
     verificar_ui.mjs (episodios). 2 columnas < 600 px; 3 columnas 600-899 px. */
  @media (max-width: 899px) {
    .tarjeta {
      height: 146px;
      contain-intrinsic-size: auto 146px;
    }
  }
  @media (min-width: 600px) and (max-width: 899px) {
    .tarjeta {
      height: 179px;
      contain-intrinsic-size: auto 179px;
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
    .franja span {
      transition: none;
    }
  }
</style>
