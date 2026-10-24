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
  import Btn from './Btn.svelte';
  import Esqueleto from './Esqueleto.svelte';
  import EstadoVista from './EstadoVista.svelte';
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
  <EstadoVista variante="error" titulo="No se pudieron cargar tus favoritos">
    <Btn variante="secundario" onClick={() => void cargarFavoritos()}>Reintentar</Btn>
  </EstadoVista>
{:else if $favoritosEstado === 'ready' && $favoritos.length === 0}
  <EstadoVista
    variante="vacio"
    titulo="Aún no tienes favoritos"
    texto="Marca la ★ en el detalle de un anime."
  />
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
  <div class="cuadricula" aria-busy="true">
    {#each [0, 1, 2, 3, 4, 5] as indice (indice)}
      <div class="esqueleto-tarjeta" aria-hidden="true">
        <Esqueleto alto="100%" ancho="100%" radio="var(--r-lg)" />
      </div>
    {/each}
  </div>
{/if}

<style>
  .cuadricula {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--sp-3);
    text-align: left;
  }
  .esqueleto-tarjeta {
    aspect-ratio: 140 / 264;
  }
  @media (min-width: 480px) and (max-width: 899px) {
    .cuadricula {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }
  @media (min-width: 900px) {
    .cuadricula {
      grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    }
  }
</style>
