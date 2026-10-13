<script lang="ts">
  /**
   * Pestaña "Favoritos" del Home (T54): cuadrícula con los favoritos en el
   * orden que trae el backend (más reciente primero), con la misma tarjeta del
   * Home y una insignia del proveedor por tarjeta (la lista mezcla proveedores).
   *
   * Estados: `error` con "Reintentar", vacío con el aviso del diseño, y la
   * lista cuando `favoritosEstado` es `ready`. La carga no vive aquí: la
   * dispara `VistaInicio` al seleccionar la pestaña (`cargarFavoritos`).
   * El clic reutiliza `onAbrir`, la misma función de "abrir anime" del Home.
   */
  import type { models } from './api';
  import TarjetaAnime from './TarjetaAnime.svelte';
  import { cargarFavoritos, favoritos, favoritosEstado } from './favoritos';
  import { NOMBRE_PROVEEDOR } from './historial';

  export let onAbrir: (a: models.Anime) => void;

  /** Un `Favorite` en la forma de tarjeta que espera `TarjetaAnime`. */
  function aAnime(f: models.Favorite): models.Anime {
    return {
      id: f.anime_id,
      title: f.title,
      cover_url: f.cover_url,
      provider: f.provider,
    };
  }
</script>

{#if $favoritosEstado === 'error'}
  <div class="error-favoritos">
    <p class="error-texto">No se pudieron cargar tus favoritos.</p>
    <button type="button" class="reintentar" on:click={() => void cargarFavoritos()}>
      Reintentar
    </button>
  </div>
{:else if $favoritosEstado === 'ready' && $favoritos.length === 0}
  <p class="vacio">Aún no tienes favoritos. Marca la ★ en el detalle de un anime.</p>
{:else if $favoritosEstado === 'ready'}
  <div class="cuadricula">
    {#each $favoritos as fav (fav.provider + ':' + fav.anime_id)}
      <TarjetaAnime
        anime={aAnime(fav)}
        etiqueta={NOMBRE_PROVEEDOR[fav.provider] ?? fav.provider}
        {onAbrir}
      />
    {/each}
  </div>
{:else}
  <p class="cargando">Cargando favoritos…</p>
{/if}

<style>
  .cuadricula {
    display: grid;
    /* Mismo mínimo que la tarjeta del Home: la columna crece hasta llenarse. */
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 12px;
    text-align: left;
  }
  .error-favoritos {
    padding: 12px;
    background: var(--cad-superficie);
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    text-align: left;
  }
  .error-texto {
    margin: 0 0 10px;
    color: var(--cad-texto);
  }
  .reintentar {
    box-sizing: border-box;
    padding: 6px 12px;
    background: transparent;
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    color: var(--cad-acento);
    font-family: inherit;
    font-weight: bold;
    cursor: pointer;
  }
  .reintentar:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .vacio,
  .cargando {
    color: var(--cad-texto-suave);
    text-align: center;
  }
</style>
