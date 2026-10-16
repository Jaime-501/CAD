<script lang="ts">
  import type { VideoSourceView } from './api';
  import { agruparFuentes, type FiltroIdioma } from './etiquetas';

  /**
   * Selector manual de servidor/idioma/calidad (T27/T34/T63).
   *
   * Los **principales** (HLS/MP4: jkplayer, MediaFire, el HLS de AnimeAV1 y
   * el MP4Upload resuelto) van primero, en la fila de siempre. Los **extras**
   * (iframes embebidos, que pueden traer publicidad) quedan debajo en una
   * sección plegable, **cerrada por defecto**, que se despliega sola si la
   * fuente activa es un iframe o si no hay ningún principal (T63). La
   * clasificación es por TIPO, nunca por nombre, y cada botón es texto
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

  $: grupos = agruparFuentes(fuentes, filtro);
  $: activaEsIframe = fuentes[actual]?.type === 'iframe';
  $: requiereAbierto = activaEsIframe || grupos.principales.length === 0;

  /**
   * Estado del pliegue. Por defecto cerrado: `manualPara` guarda para qué
   * índice lo fijó el usuario (`abierto`), de modo que al cambiar de fuente
   * vuelve a mandar la regla automática. Sin ciclos reactivos.
   */
  let abierto = false;
  let manualPara = -1;

  $: manualActivo = manualPara === actual;
  $: abiertoVisible = manualActivo ? abierto : requiereAbierto;

  function alternar(): void {
    const base = manualPara === actual ? abierto : requiereAbierto;
    manualPara = actual;
    abierto = !base;
  }
</script>

<div class="selector">
  <span class="titulo">Servidor · Idioma</span>
  {#if grupos.principales.length > 0}
    <div class="botones">
      {#each grupos.principales as fila (fila.indice)}
        <button
          type="button"
          class:activa={fila.indice === actual}
          on:click={() => elegir(fila.indice)}
        >
          {fila.etiqueta}
        </button>
      {/each}
    </div>
  {/if}

  {#if grupos.extras.length > 0}
    <button
      type="button"
      class="plegable"
      aria-expanded={abiertoVisible}
      on:click={alternar}
    >
      <span class="flecha" aria-hidden="true">{abiertoVisible ? '▾' : '▸'}</span>
      Otras fuentes (embebidas, pueden incluir publicidad)
    </button>
    {#if abiertoVisible}
      <div class="botones extras">
        {#each grupos.extras as fila (fila.indice)}
          <button
            type="button"
            class:activa={fila.indice === actual}
            on:click={() => elegir(fila.indice)}
          >
            {fila.etiqueta}
          </button>
        {/each}
      </div>
    {/if}
  {/if}
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
  .botones button:focus-visible {
    outline: 2px solid #4ade80;
    outline-offset: 2px;
  }
  .botones button.activa {
    background: #4ade80;
    border-color: #4ade80;
    color: #111;
    font-weight: bold;
  }
  .plegable {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-top: 12px;
    padding: 6px 10px;
    border-radius: 8px;
    border: 1px solid transparent;
    background: transparent;
    color: #a1a1aa;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }
  .plegable:hover {
    color: #d4d4d8;
    border-color: #3f3f46;
  }
  .plegable:focus-visible {
    outline: 2px solid #4ade80;
    outline-offset: 2px;
  }
  .flecha {
    font-size: 11px;
  }
  .botones.extras {
    margin-top: 8px;
  }
</style>
