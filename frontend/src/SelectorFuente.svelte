<script lang="ts">
  import type { VideoSourceView } from './api';

  /**
   * Selector manual de servidor/calidad (T27): lista TODAS las fuentes tal
   * como las devuelve `getVideoSources` (la primera es la preferida) y marca
   * la que se está usando. El nombre sale del servidor real del extractor y,
   * si lo hay, de la calidad; sin servidor se usa el tipo de recurso.
   */
  export let fuentes: VideoSourceView[] = [];
  export let actual = 0;
  export let elegir: (indice: number) => void;

  function etiqueta(f: VideoSourceView): string {
    const tipo =
      f.type === 'hls'
        ? 'HLS'
        : f.type === 'mp4'
          ? 'MP4'
          : f.type === 'iframe'
            ? 'Iframe'
            : 'Fuente';
    const servidor = f.server.trim() !== '' ? f.server.trim() : tipo;
    const calidad = f.quality.trim();
    return calidad !== '' ? `${servidor} · ${calidad}` : servidor;
  }
</script>

<div class="selector">
  <span class="titulo">Servidor / calidad</span>
  <div class="botones">
    {#each fuentes as fuente, i}
      <button type="button" class:activa={i === actual} on:click={() => elegir(i)}>
        {etiqueta(fuente)}
      </button>
    {/each}
  </div>
</div>

<style>
  .selector {
    margin-top: 1rem;
  }
  .titulo {
    display: block;
    color: #a1a1aa;
    font-size: 13px;
    margin-bottom: 8px;
  }
  .botones {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .botones button {
    padding: 7px 14px;
    border-radius: 999px;
    border: 1px solid #3f3f46;
    background: #1e1e24;
    color: #d4d4d8;
    font-size: 13px;
    cursor: pointer;
  }
  .botones button:hover {
    border-color: #4ade80;
  }
  .botones button.activa {
    background: #4ade80;
    border-color: #4ade80;
    color: #111;
    font-weight: bold;
  }
</style>
