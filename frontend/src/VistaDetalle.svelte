<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import { getProgress } from './api';
  import { etiquetaTipo } from './etiquetas';
  import {
    currentAnimeID,
    currentDetail,
    currentProvider,
    episodes,
    episodesComplete,
    episodesError,
    openAnime,
    progressMap,
    progressToMap,
  } from './stores';
  import { atras, claveScroll, navegarA, recuperarScroll, seleccion } from './views';
  import { COLUMNAS_POR_DEFECTO } from './ventana';
  import { medirColumnas, revelar } from './revelar';
  import BotonFavorito from './BotonFavorito.svelte';
  import TarjetaEpisodio from './TarjetaEpisodio.svelte';

  let cargando = true;
  let error = '';

  // T65: columnas reales de la rejilla para el retardo de entrada por fila.
  let rejillaEl: HTMLDivElement | undefined = undefined;
  let columnas = COLUMNAS_POR_DEFECTO;
  let frameResize: number | null = null;

  // T67: restauración del scroll del detalle cuando la lista se recarga. Al
  // volver a un detalle cuyos stores no están intactos, `openAnime` vacía la
  // lista y el `scrollTo` de `App.svelte` se recorta a 0 (todavía no hay
  // altura). Aquí se arma una restauración pendiente que se aplica en cuanto
  // la lista crece lo suficiente. `0` = desarmado.
  let scrollPendiente = 0;
  let quitarOyentesScroll: (() => void) | null = null;

  function medirColumnasRejilla(): void {
    if (rejillaEl) columnas = medirColumnas(rejillaEl);
  }

  function alRedimensionar(): void {
    if (frameResize !== null) return;
    // rAF: no medir en cada píxel del arrastre de la ventana.
    frameResize = requestAnimationFrame(() => {
      frameResize = null;
      medirColumnasRejilla();
    });
  }

  onMount(() => {
    window.addEventListener('resize', alRedimensionar);
    return () => {
      window.removeEventListener('resize', alRedimensionar);
      if (frameResize !== null) cancelAnimationFrame(frameResize);
    };
  });

  // La rejilla sólo existe cuando hay episodios; se mide al montar y tras el resize.
  $: if (rejillaEl) medirColumnasRejilla();

  /** Desarma la restauración pendiente y quita sus oyentes (sin mover el scroll). */
  function desarmarRestauracionScroll(): void {
    scrollPendiente = 0;
    if (quitarOyentesScroll) {
      quitarOyentesScroll();
      quitarOyentesScroll = null;
    }
  }

  /** Arma la restauración y registra los oyentes que la cancelan al primer gesto del usuario. */
  function armarRestauracionScroll(objetivo: number): void {
    scrollPendiente = objetivo;
    if (quitarOyentesScroll) return;
    const cancelar = (): void => desarmarRestauracionScroll();
    window.addEventListener('wheel', cancelar, { passive: true });
    window.addEventListener('touchstart', cancelar, { passive: true });
    window.addEventListener('keydown', cancelar, { passive: true });
    window.addEventListener('pointerdown', cancelar, { passive: true });
    quitarOyentesScroll = () => {
      window.removeEventListener('wheel', cancelar);
      window.removeEventListener('touchstart', cancelar);
      window.removeEventListener('keydown', cancelar);
      window.removeEventListener('pointerdown', cancelar);
    };
  }

  /**
   * Intenta aplicar la restauración pendiente. Se llama en cada cambio de la
   * lista (lotes de episodios) y al completar. Mide dentro de un rAF para que
   * el layout ya esté pintado; si la altura alcanza, aplica y desarma; si el
   * stream terminó y no alcanza, baja al máximo posible y desarma.
   */
  function intentarRestaurarScroll(_longitud: number, completo: boolean): void {
    if (scrollPendiente <= 0) return;
    requestAnimationFrame(() => {
      if (scrollPendiente <= 0) return;
      const maximo = document.documentElement.scrollHeight - window.innerHeight;
      if (maximo >= scrollPendiente) {
        window.scrollTo(0, scrollPendiente);
        desarmarRestauracionScroll();
      } else if (completo) {
        window.scrollTo(0, Math.max(0, maximo));
        desarmarRestauracionScroll();
      }
    });
  }

  onMount(() => {
    const sel = get(seleccion);
    if (!sel) {
      cargando = false;
      error = 'No hay ningún anime seleccionado.';
      return;
    }
    // T67: la posición que hay que recuperar, antes de decidir la rama.
    const objetivo = recuperarScroll(claveScroll('detalle'));
    // Vuelta desde el reproductor: el detalle y los episodios siguen en los
    // stores, se conservan tal cual (y con ellos la lista y su scroll); solo
    // se refresca el historial, que puede haber cambiado en el reproductor.
    // El slug NO identifica al anime (T34): "bleach" es jkanime y animeav1,
    // así que hace falta coincidir también el proveedor.
    if (
      get(currentAnimeID) === sel.animeID &&
      get(currentProvider) === sel.provider &&
      get(currentDetail)
    ) {
      cargando = false;
      refrescarProgreso(sel.animeID, sel.provider);
      return;
    }
    // Rama de recarga: la lista se vacía y crece por lotes, así que el scroll
    // se aplica cuando haya altura suficiente.
    if (objetivo > 0) armarRestauracionScroll(objetivo);
    openAnime(sel.provider, sel.animeID)
      .catch((e) => {
        error = String(e);
      })
      .finally(() => {
        cargando = false;
      });
  });

  onDestroy(() => {
    desarmarRestauracionScroll();
  });

  /** Recarga `progressMap` sin vaciar la lista (mismo criterio de descarte que `openAnime`). */
  async function refrescarProgreso(animeID: string, proveedor: string): Promise<void> {
    try {
      const historial = await getProgress(animeID);
      if (get(currentAnimeID) !== animeID || get(currentProvider) !== proveedor) return;
      // Solo el progreso del proveedor abierto: el slug puede ser compartido
      // y el historial trae registros de todos (T34).
      progressMap.set(progressToMap(historial, proveedor));
    } catch (e) {
      console.warn('[VistaDetalle] historial no disponible:', String(e));
    }
  }

  // Los episodios llegan deduplicados por T24 (y con lotes que pueden llegar
  // desordenados o antes de resolver la llamada): aquí solo se ordenan.
  $: episodios = [...$episodes].sort((a, b) => a.number - b.number || a.id.localeCompare(b.id));

  // T67: reintenta la restauración pendiente con cada cambio de la lista.
  $: intentarRestaurarScroll(episodios.length, $episodesComplete);

  /** Tipo del detalle en texto amable (T34): `tv-anime` → `TV`; vacío si no hay. */
  $: tipoEtiqueta = etiquetaTipo($currentDetail?.type ?? '');

  // "Continuar": el episodio de mayor número con progreso; si ya está visto,
  // el siguiente de la lista (si existe).
  $: episodioContinuar = paraContinuar(episodios, $progressMap);

  function paraContinuar(
    lista: models.Episode[],
    progreso: Record<string, models.WatchProgress>
  ): models.Episode | null {
    let mejor: models.Episode | null = null;
    for (const ep of lista) {
      const p = progreso[ep.id];
      if (!p || !(p.current_time > 0)) continue;
      if (mejor === null || ep.number > mejor.number) mejor = ep;
    }
    if (mejor === null) return null;
    if (progreso[mejor.id].is_watched) {
      const i = lista.indexOf(mejor);
      return i >= 0 && i + 1 < lista.length ? lista[i + 1] : null;
    }
    return mejor;
  }

  function abrirEpisodio(ep: models.Episode): void {
    const sel = get(seleccion);
    if (!sel) return;
    navegarA('reproductor', {
      ...sel,
      episodeID: ep.id,
      episodeNumber: ep.number,
      episodeTitle: ep.title,
    });
  }

  function continuar(): void {
    if (episodioContinuar) abrirEpisodio(episodioContinuar);
  }
</script>

<main class="contenedor">
  <div class="cabecera">
    <button type="button" class="atras" on:click={atras}>← Atrás</button>
    <h2>{$currentDetail?.title ?? $seleccion?.title ?? 'Detalle'}</h2>
    {#if $currentDetail}
      <BotonFavorito
        provider={$currentDetail.provider}
        animeId={$currentDetail.id}
        titulo={$currentDetail.title}
        cover={$currentDetail.cover}
      />
    {/if}
  </div>

  {#if error}
    <p class="error">{error}</p>
  {:else if cargando && !$currentDetail}
    <p class="estado">Cargando detalle…</p>
  {:else if $currentDetail}
    <section class="ficha">
      {#if $currentDetail.cover !== ''}
        <img class="portada" src={$currentDetail.cover} alt={$currentDetail.title} />
      {/if}
      <div class="datos">
        <p class="linea">
          {$currentDetail.provider}
          {#if tipoEtiqueta !== ''}· {tipoEtiqueta}{/if}
          {#if $currentDetail.status !== ''}· {$currentDetail.status}{/if}
          · {$currentDetail.total_episodes}
          {$currentDetail.total_episodes === 1 ? 'episodio' : 'episodios'}
        </p>
        {#if $currentDetail.genres?.length}
          <ul class="generos">
            {#each $currentDetail.genres as genero (genero)}
              <li>{genero}</li>
            {/each}
          </ul>
        {/if}
        {#if $currentDetail.synopsis !== ''}
          <p class="sinopsis">{$currentDetail.synopsis}</p>
        {/if}
        <button
          type="button"
          class="continuar"
          disabled={!episodioContinuar}
          on:click={continuar}
        >
          {#if episodioContinuar}
            Continuar · Ep. {episodioContinuar.number}
          {:else}
            Continuar
          {/if}
        </button>
      </div>
    </section>

    <h3 class="titular">Episodios {#if episodios.length > 0}({episodios.length}){/if}</h3>

    {#if $episodesError !== ''}
      <p class="aviso">No se pudieron cargar todos los episodios: {$episodesError}</p>
    {/if}

    {#if episodios.length > 0}
      <div class="rejilla" bind:this={rejillaEl}>
        {#each episodios as ep, i (ep.id)}
          <div class="celda" use:revelar={{ id: ep.id, indice: i, columnas }}>
            <TarjetaEpisodio episodio={ep} progreso={$progressMap[ep.id]} abrir={abrirEpisodio} />
          </div>
        {/each}
      </div>
    {/if}

    {#if !$episodesComplete}
      <p class="cargando">Cargando más episodios…</p>
    {:else if episodios.length === 0}
      <p class="estado">No hay episodios disponibles.</p>
    {/if}
  {:else}
    <p class="estado">Sin detalle disponible.</p>
  {/if}
</main>

<style>
  .contenedor {
    padding: 2rem;
    text-align: left;
    max-width: 1100px;
    margin: 0 auto;
  }
  .cabecera {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 1.5rem;
  }
  .cabecera h2 {
    margin: 0;
    text-align: left;
  }
  .atras {
    padding: 8px 16px;
    border-radius: var(--r-md);
    border: none;
    background: var(--brand-500);
    color: var(--on-brand);
    font-weight: bold;
    cursor: pointer;
    white-space: nowrap;
  }
  .ficha {
    display: flex;
    gap: 24px;
    margin-bottom: 2rem;
    background: var(--surface-card);
    border-radius: var(--r-md);
    padding: 20px;
  }
  .portada {
    width: 220px;
    max-width: 35%;
    height: fit-content;
    aspect-ratio: 2 / 3;
    object-fit: cover;
    border-radius: var(--r-md);
    background: var(--bg-2);
    flex-shrink: 0;
  }
  .datos {
    flex: 1;
    min-width: 0;
  }
  .linea {
    margin: 0 0 10px;
    color: var(--text-3);
    font-size: 14px;
  }
  .generos {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 0 0 12px;
    padding: 0;
  }
  .generos li {
    background: var(--surface-chip);
    color: var(--brand-400);
    font-size: 12px;
    padding: 4px 10px;
    border-radius: var(--r-full);
  }
  .sinopsis {
    margin: 0 0 16px;
    line-height: 1.5;
    color: var(--text-2);
    white-space: pre-line;
  }
  .continuar {
    padding: 10px 20px;
    border-radius: var(--r-md);
    border: none;
    background: var(--brand-500);
    color: var(--on-brand);
    font-weight: bold;
    cursor: pointer;
  }
  .continuar:disabled {
    background: var(--bg-2);
    color: var(--text-3);
    cursor: not-allowed;
  }
  .titular {
    margin: 0 0 1rem;
  }
  .rejilla {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
    gap: 18px;
  }
  /* T65: la animación de entrada vive en el envoltorio `.celda` (no en la
     tarjeta) para no chocar con el transform del hover ni con
     content-visibility. La tarjeta ocupa todo el alto disponible. */
  .celda {
    display: flex;
    flex-direction: column;
  }
  .celda :global(.tarjeta) {
    flex: 1 1 auto;
  }
  .celda:global(.pendiente) {
    opacity: 0;
  }
  .celda:global(.entra) {
    animation: entra-celda 320ms cubic-bezier(0.22, 0.61, 0.36, 1) both;
  }
  @keyframes entra-celda {
    from {
      opacity: 0;
      transform: translateX(-18px);
    }
    to {
      opacity: 1;
      transform: none;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .celda:global(.pendiente) {
      opacity: 1;
    }
    .celda:global(.entra) {
      animation: none;
    }
  }
  .estado {
    color: var(--text-3);
  }
  .error {
    color: var(--danger);
  }
  .aviso {
    color: var(--fav);
    background: var(--bg-2);
    border-radius: var(--r-md);
    padding: 10px 14px;
    margin-bottom: 1rem;
  }
  .cargando {
    color: var(--text-3);
    text-align: center;
    padding: 1rem 0;
  }
  @media (max-width: 640px) {
    .ficha {
      flex-direction: column;
    }
    .portada {
      width: 160px;
      max-width: 160px;
    }
  }
</style>
