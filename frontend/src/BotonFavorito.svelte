<script lang="ts">
  /**
   * Estrella de favorito (T54): alterna la marca del anime que se está viendo.
   *
   * El estado sale de `clavesFavoritos` (la lista en memoria): no hay petición
   * `isFavorite` por montaje, solo `cargarFavoritos()` si aún no se ha pedido.
   * El cambio es optimista en `favoritos.ts`; si el backend falla se revierte
   * y aquí se muestra "No se pudo actualizar" durante 3 s.
   */
  import { onDestroy, onMount } from 'svelte';
  import {
    clavesFavoritos,
    cargarFavoritos,
    desmarcarFavorito,
    favoritosEstado,
    marcarFavorito,
  } from './favoritos';

  export let provider: string;
  export let animeId: string;
  export let titulo: string;
  export let cover: string;

  const AVISO_MS = 3000;

  let ocupado = false;
  let aviso = '';
  let temporizador: ReturnType<typeof setTimeout> | null = null;

  onMount(() => {
    if ($favoritosEstado === 'idle') void cargarFavoritos();
  });

  onDestroy(() => {
    if (temporizador !== null) clearTimeout(temporizador);
  });

  $: clave = `${provider}:${animeId}`;
  $: marcada = $clavesFavoritos.has(clave);

  async function alternar(): Promise<void> {
    if (ocupado) return;
    ocupado = true;
    try {
      if ($clavesFavoritos.has(clave)) {
        await desmarcarFavorito(provider, animeId);
      } else {
        await marcarFavorito({
          provider,
          anime_id: animeId,
          title: titulo,
          cover_url: cover,
        });
      }
      aviso = '';
    } catch (e) {
      console.warn('[BotonFavorito] no se pudo actualizar:', String(e));
      mostrarAviso('No se pudo actualizar');
    } finally {
      ocupado = false;
    }
  }

  function mostrarAviso(texto: string): void {
    if (temporizador !== null) clearTimeout(temporizador);
    aviso = texto;
    temporizador = setTimeout(() => {
      aviso = '';
      temporizador = null;
    }, AVISO_MS);
  }
</script>

<span class="favorito">
  <button
    type="button"
    class="estrella"
    class:marcada
    aria-pressed={marcada}
    aria-label={marcada ? 'Quitar de favoritos' : 'Añadir a favoritos'}
    title={marcada ? 'Quitar de favoritos' : 'Añadir a favoritos'}
    disabled={ocupado}
    on:click={() => void alternar()}
  >
    <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" focusable="false">
      <path
        d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"
      />
    </svg>
  </button>
  {#if aviso !== ''}
    <span class="aviso" role="status">{aviso}</span>
  {/if}
</span>

<style>
  .favorito {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }
  .estrella {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 6px;
    background: transparent;
    border: none;
    border-radius: var(--cad-radio);
    color: var(--cad-texto-suave);
    cursor: pointer;
  }
  .estrella svg {
    fill: none;
    stroke: currentColor;
    stroke-width: 1.75;
    stroke-linejoin: round;
  }
  .estrella.marcada {
    color: var(--cad-acento);
  }
  .estrella.marcada svg {
    fill: currentColor;
  }
  .estrella:hover:not(:disabled) {
    color: var(--cad-acento);
  }
  .estrella:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .estrella:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .aviso {
    font-size: 13px;
    color: var(--cad-peligro);
  }
</style>
