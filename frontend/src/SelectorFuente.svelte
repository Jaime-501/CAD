<script lang="ts">
  import type { VideoSourceView } from './api';
  import { agruparFuentes, type FiltroIdioma } from './etiquetas';
  import Icono from './Icono.svelte';

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
          aria-pressed={fila.indice === actual}
          on:click={() => elegir(fila.indice)}
        >
          {#if fila.indice === actual}
            <span class="check" aria-hidden="true"><Icono nombre="check" tam={16} /></span>
          {/if}
          <span class="texto">{fila.etiqueta}</span>
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
      <span class="flecha" aria-hidden="true">
        <Icono nombre={abiertoVisible ? 'chevron-abajo' : 'chevron-der'} tam={16} />
      </span>
      <span class="texto-plegable">Otras fuentes (embebidas, pueden incluir publicidad)</span>
    </button>
    {#if abiertoVisible}
      <div class="botones extras">
        {#each grupos.extras as fila (fila.indice)}
          <button
            type="button"
            class:activa={fila.indice === actual}
            aria-pressed={fila.indice === actual}
            on:click={() => elegir(fila.indice)}
          >
            {#if fila.indice === actual}
              <span class="check" aria-hidden="true"><Icono nombre="check" tam={16} /></span>
            {/if}
            <span class="texto">{fila.etiqueta}</span>
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
    color: var(--text-3);
    font-size: 13px;
    margin-bottom: var(--sp-2);
  }
  .botones {
    display: flex;
    flex-wrap: wrap;
    gap: var(--sp-2);
  }
  .botones button {
    padding: 7px 14px;
    border-radius: var(--r-full);
    border: 1px solid var(--border);
    background: var(--surface-card);
    color: var(--text-2);
    font-size: 13px;
    cursor: pointer;
  }
  .botones button:focus-visible,
  .plegable:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  .botones button.activa {
    background: var(--brand-500);
    border-color: var(--brand-400);
    color: var(--on-brand);
    font-weight: bold;
  }
  .plegable {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-top: var(--sp-3);
    padding: 6px 10px;
    border-radius: var(--r-md);
    border: 1px solid transparent;
    background: transparent;
    color: var(--text-3);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
  }
  .check {
    display: none;
  }
  .flecha {
    display: inline-flex;
    align-items: center;
  }
  .botones.extras {
    margin-top: var(--sp-2);
  }
  @media (hover: hover) and (pointer: fine) {
    .botones button:hover {
      border-color: var(--brand-400);
    }
    .plegable:hover {
      color: var(--text-2);
      border-color: var(--border);
    }
  }

  /* Modo estrecho (< 900 px): una fila con scroll horizontal y chips táctiles. */
  @media (max-width: 899px) {
    .selector {
      margin-top: var(--sp-4);
    }
    .titulo {
      font-size: var(--fs-sm);
      margin-bottom: var(--sp-2);
    }
    .botones {
      flex-wrap: nowrap;
      gap: var(--sp-2);
      overflow-x: auto;
      scroll-snap-type: x proximity;
      scrollbar-width: none;
      margin-inline: calc(-1 * var(--gutter));
      padding-inline: var(--gutter);
      scroll-padding-inline: var(--gutter);
      overscroll-behavior-x: contain;
    }
    .botones::-webkit-scrollbar {
      display: none;
    }
    .botones button {
      display: inline-flex;
      align-items: center;
      gap: var(--sp-2);
      flex-shrink: 0;
      scroll-snap-align: start;
      min-height: var(--tap);
      padding: 0 var(--sp-4);
      font-size: var(--fs-sm);
      white-space: nowrap;
    }
    .check {
      display: inline-flex;
      align-items: center;
    }
    .botones button.activa {
      background: var(--grad-chip-active);
      border-color: var(--brand-400);
      color: var(--on-brand);
      font-weight: var(--fw-semibold);
    }
    .plegable {
      min-height: var(--tap);
      font-size: var(--fs-sm);
      border-radius: var(--r-md);
      padding: var(--sp-2) var(--sp-3);
    }
    .texto-plegable {
      white-space: normal;
    }
    .botones.extras {
      margin-top: var(--sp-2);
    }
    .botones button,
    .plegable {
      transition:
        transform var(--dur-fast) var(--ease-std),
        background-color var(--dur-fast) var(--ease-std),
        border-color var(--dur-fast) var(--ease-std),
        color var(--dur-fast) var(--ease-std);
    }
    .botones button:active,
    .plegable:active {
      transform: scale(0.97);
    }
    @media (prefers-reduced-motion: reduce) {
      .botones button,
      .plegable {
        transition: none;
      }
    }
  }
</style>
