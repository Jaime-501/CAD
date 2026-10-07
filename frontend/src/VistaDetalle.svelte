<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { currentDetail, openAnime } from './stores';
  import { atras, seleccion } from './views';

  let cargando = true;
  let error = '';

  // Al entrar se ejercita el pipeline de T24 (openAnime: eventos, caché e historial).
  onMount(() => {
    const sel = get(seleccion);
    if (!sel) {
      cargando = false;
      error = 'No hay ningún anime seleccionado.';
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
</script>

<main class="contenedor">
  <div class="cabecera">
    <button type="button" class="atras" on:click={atras}>← Atrás</button>
    <h2>{$currentDetail?.title ?? $seleccion?.title ?? 'Detalle'}</h2>
  </div>

  {#if error}
    <p class="error">{error}</p>
  {:else if cargando}
    <p class="estado">Cargando detalle…</p>
  {:else if $currentDetail}
    <p class="estado">
      {$currentDetail.provider} · {$currentDetail.total_episodes} episodios · {$currentDetail.status}
    </p>
    <p class="nota">La vista de detalle completo (sinopsis y episodios) llega en T26.</p>
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
  .estado {
    color: #a1a1aa;
  }
  .error {
    color: #f87171;
  }
  .nota {
    color: #71717a;
    font-size: 14px;
  }
</style>
