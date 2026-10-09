<script lang="ts">
  import type { VideoSourceView } from './api';
  import { etiquetaFuente, fuentesVisibles, type FiltroIdioma } from './etiquetas';

  /**
   * Selector manual de servidor/idioma/calidad (T27/T34): lista TODAS las
   * fuentes tal como las devuelve `getVideoSources` (la primera es la
   * preferida) y marca la que se está usando. Cada botón es texto
   * (`Servidor · Idioma · Calidad`, ver `etiquetaFuente`), nunca solo color.
   *
   * `filtro` queda a `null` (sin filtro, SUB y DUB a la vista) hasta que haya
   * un control de idioma: cambiarlo filtra sin tocar el resto.
   */
  export let fuentes: VideoSourceView[] = [];
  export let filtro: FiltroIdioma = null;
  export let actual = 0;
  /** Recibe el índice en la lista COMPLETA, aunque se muestre filtrada. */
  export let elegir: (indice: number) => void;

  $: visibles = fuentesVisibles(fuentes, filtro).map((f) => ({
    fuente: f,
    indice: fuentes.indexOf(f),
  }));
</script>

<div class="selector">
  <span class="titulo">Servidor · Idioma</span>
  <div class="botones">
    {#each visibles as fila (fila.indice)}
      <button
        type="button"
        class:activa={fila.indice === actual}
        on:click={() => elegir(fila.indice)}
      >
        {etiquetaFuente(fila.fuente)}
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
