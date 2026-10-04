<script lang="ts">
  // Importamos la función mágica que Wails generó para nosotros
  import { SearchAnime } from '../wailsjs/go/main/App.js';

  let query = '';
  let results: any[] = [];
  let isSearching = false;
  let errorMsg = '';

  async function handleSearch() {
    if (!query.trim()) return;

    // window.go solo lo inyecta el dev server de Wails (:34115),
    // no el servidor de Vite (:5173)
    if (!(window as any).go) {
      errorMsg =
        'Sin conexión con Go. Abre la app en http://localhost:34115 — el puerto 5173 (Vite) no tiene el puente de Wails.';
      results = [];
      return;
    }

    isSearching = true;
    errorMsg = '';
    try {
      // ¡Llamamos a Go desde JavaScript!
      results = await SearchAnime(query);
      console.log("Resultados:", results);
    } catch (err) {
      console.error("Error al buscar:", err);
      errorMsg = 'Error al buscar: ' + String(err);
      results = [];
    } finally {
      isSearching = false;
    }
  }
</script>

<main class="container">
  <h2>Buscador de AnimeFLV</h2>

  <div class="search-bar">
    <input
      type="text"
      bind:value={query}
      placeholder="Ej: Death Note, Jujutsu Kaisen..."
      on:keydown={(e) => e.key === 'Enter' && handleSearch()}
    />
    <button on:click={handleSearch} disabled={isSearching}>
      {isSearching ? 'Buscando...' : 'Buscar'}
    </button>
  </div>

  {#if errorMsg}
    <p class="error">{errorMsg}</p>
  {/if}

  <div class="grid">
    {#each results as anime}
      <div class="card">
        <!-- Usamos la URL de la portada que scrapeó Go -->
        <img src={anime.cover_url} alt={anime.title} />
        <p>{anime.title}</p>
      </div>
    {/each}

    {#if results.length === 0 && !isSearching && query !== '' && !errorMsg}
      <p>No se encontraron resultados.</p>
    {/if}
  </div>
</main>

<style>
  .container {
    padding: 2rem;
    font-family: system-ui, sans-serif;
    color: white;
  }
  .search-bar {
    display: flex;
    gap: 10px;
    margin-bottom: 2rem;
  }
  .error {
    color: #f87171;
    margin-bottom: 1rem;
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
    cursor: not-allowed;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 20px;
  }
  .card {
    background: #1e1e24;
    border-radius: 10px;
    overflow: hidden;
    text-align: center;
    transition: transform 0.2s;
    cursor: pointer;
  }
  .card:hover {
    transform: translateY(-5px);
  }
  .card img {
    width: 100%;
    height: 230px;
    object-fit: cover;
  }
  .card p {
    padding: 10px;
    margin: 0;
    font-size: 14px;
  }
</style>
