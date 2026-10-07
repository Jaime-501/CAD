<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from '../wailsjs/go/models';
  import { search, searchResults, searchState } from './stores';
  import { navegarA, type SeleccionAnime } from './views';

  /** false = no hay puente `window.go` (Vite puro en :5173); lo decide App.svelte. */
  export let puente = true;

  const RETRASO_BUSQUEDA = 400;
  /** AnimeFLV es solo búsqueda (T18): no tiene página de detalle, se decide por proveedor. */
  const PROVEEDOR_SIN_DETALLE = 'animeflv';

  let texto = '';
  let aviso = '';
  let temporizador: ReturnType<typeof setTimeout> | null = null;

  onMount(() => {
    // Al volver de otra vista se recupera la última búsqueda (los resultados
    // siguen en `searchResults`, que no se limpia al desmontar).
    texto = get(searchState).query;
  });

  onDestroy(() => {
    if (temporizador !== null) clearTimeout(temporizador);
  });

  /** Cada pulsación reinicia el *debounce* de ~400 ms. */
  function programar() {
    aviso = '';
    if (temporizador !== null) clearTimeout(temporizador);
    if (!texto.trim()) {
      temporizador = null;
      void search('');
      return;
    }
    if (!puente) return;
    temporizador = setTimeout(() => {
      temporizador = null;
      void search(texto);
    }, RETRASO_BUSQUEDA);
  }

  /** Enter o botón "Buscar": salta el *debounce*. */
  function buscarAhora() {
    aviso = '';
    if (temporizador !== null) {
      clearTimeout(temporizador);
      temporizador = null;
    }
    if (!puente) return;
    void search(texto);
  }

  function abrir(anime: models.Anime) {
    if (anime.provider === PROVEEDOR_SIN_DETALLE) {
      aviso = 'Detalle no disponible para este proveedor';
      return;
    }
    aviso = '';
    const sel: SeleccionAnime = {
      provider: anime.provider,
      animeID: anime.id,
      title: anime.title,
    };
    navegarA('detalle', sel);
  }
</script>

<main class="contenedor">
  <h2>Buscador de anime</h2>

  <div class="buscador">
    <input
      type="text"
      bind:value={texto}
      placeholder="Ej: Death Note, Jujutsu Kaisen..."
      on:input={programar}
      on:keydown={(e) => e.key === 'Enter' && buscarAhora()}
    />
    <button type="button" on:click={buscarAhora} disabled={!puente || $searchState.loading}>
      {$searchState.loading ? 'Buscando...' : 'Buscar'}
    </button>
  </div>

  {#if aviso}
    <p class="aviso">{aviso}</p>
  {/if}

  {#if $searchState.loading}
    <p class="estado">Buscando…</p>
  {:else if $searchState.error}
    <p class="error">Error al buscar: {$searchState.error}</p>
  {:else if $searchResults.length > 0}
    <div class="rejilla">
      {#each $searchResults as anime (anime.provider + ':' + anime.id)}
        <button type="button" class="tarjeta" on:click={() => abrir(anime)}>
          <img src={anime.cover_url} alt={anime.title} loading="lazy" />
          <span class="titulo">{anime.title}</span>
          <span class="proveedor">{anime.provider}</span>
        </button>
      {/each}
    </div>
  {:else if $searchState.query !== ''}
    <p class="estado">No se encontraron resultados para “{$searchState.query}”.</p>
  {:else}
    <p class="estado">Escribe el nombre de un anime para empezar a buscar.</p>
  {/if}
</main>

<style>
  .contenedor {
    padding: 2rem;
    text-align: left;
    max-width: 1100px;
    margin: 0 auto;
  }
  h2 {
    text-align: center;
    margin-top: 0;
  }
  .buscador {
    display: flex;
    gap: 10px;
    margin-bottom: 1.5rem;
  }
  input {
    flex: 1;
    padding: 10px 15px;
    border-radius: 8px;
    border: none;
    background: #2a2a35;
    color: white;
    font-size: 16px;
  }
  button {
    padding: 10px 20px;
    border-radius: 8px;
    border: none;
    background: #4ade80;
    color: #111;
    font-weight: bold;
    cursor: pointer;
  }
  button:disabled {
    background: #555;
    color: #999;
    cursor: not-allowed;
  }
  .estado {
    color: #a1a1aa;
    text-align: center;
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
  .rejilla {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 20px;
  }
  .tarjeta {
    background: #1e1e24;
    border: none;
    border-radius: 10px;
    overflow: hidden;
    text-align: center;
    transition: transform 0.2s;
    cursor: pointer;
    padding: 0;
    color: white;
    font-family: inherit;
    display: flex;
    flex-direction: column;
  }
  .tarjeta:hover {
    transform: translateY(-5px);
  }
  .tarjeta img {
    width: 100%;
    height: 230px;
    object-fit: cover;
    background: #2a2a35;
  }
  .titulo {
    padding: 10px 10px 4px;
    font-size: 14px;
    line-height: 1.3;
  }
  .proveedor {
    padding-bottom: 10px;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #4ade80;
  }
</style>
