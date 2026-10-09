<script lang="ts">
  import { onMount } from 'svelte';
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
  import { atras, navegarA, seleccion } from './views';
  import TarjetaEpisodio from './TarjetaEpisodio.svelte';

  let cargando = true;
  let error = '';

  onMount(() => {
    const sel = get(seleccion);
    if (!sel) {
      cargando = false;
      error = 'No hay ningún anime seleccionado.';
      return;
    }
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
    openAnime(sel.provider, sel.animeID)
      .catch((e) => {
        error = String(e);
      })
      .finally(() => {
        cargando = false;
      });
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
      <div class="rejilla">
        {#each episodios as ep (ep.id)}
          <TarjetaEpisodio episodio={ep} progreso={$progressMap[ep.id]} abrir={abrirEpisodio} />
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
    border-radius: 8px;
    border: none;
    background: #4ade80;
    color: #111;
    font-weight: bold;
    cursor: pointer;
    white-space: nowrap;
  }
  .ficha {
    display: flex;
    gap: 24px;
    margin-bottom: 2rem;
    background: #1e1e24;
    border-radius: 12px;
    padding: 20px;
  }
  .portada {
    width: 220px;
    max-width: 35%;
    height: fit-content;
    aspect-ratio: 2 / 3;
    object-fit: cover;
    border-radius: 8px;
    background: #2a2a35;
    flex-shrink: 0;
  }
  .datos {
    flex: 1;
    min-width: 0;
  }
  .linea {
    margin: 0 0 10px;
    color: #a1a1aa;
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
    background: #2a2a35;
    color: #4ade80;
    font-size: 12px;
    padding: 4px 10px;
    border-radius: 999px;
  }
  .sinopsis {
    margin: 0 0 16px;
    line-height: 1.5;
    color: #d4d4d8;
    white-space: pre-line;
  }
  .continuar {
    padding: 10px 20px;
    border-radius: 8px;
    border: none;
    background: #4ade80;
    color: #111;
    font-weight: bold;
    cursor: pointer;
  }
  .continuar:disabled {
    background: #555;
    color: #999;
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
  .estado {
    color: #a1a1aa;
  }
  .error {
    color: #f87171;
  }
  .aviso {
    color: #fbbf24;
    background: #2a2a35;
    border-radius: 8px;
    padding: 10px 14px;
    margin-bottom: 1rem;
  }
  .cargando {
    color: #a1a1aa;
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
