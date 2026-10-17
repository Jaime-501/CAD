<script lang="ts">
  /**
   * T73: barra de navegación inferior (solo modo estrecho, < 900 px). Sustituye
   * a la hamburguesa de la cabecera con 4 destinos: Descubrir, Buscar,
   * Favoritos e Historial. Usa tokens, safe-area y objetivos táctiles de
   * `--tap` (44 px). El destino activo se marca con `aria-current="page"`, el
   * color de marca y un indicador superior de 2 px (no depende solo del color).
   *
   * Cada destino dispara su callback: los cablea `App.svelte` (cambiar de
   * pestaña, enfocar el buscador o abrir el cajón de historial). No lleva
   * contador de favoritos (lo decide T75).
   */
  import Icono from './Icono.svelte';

  type DestinoBarra = 'descubrir' | 'buscar' | 'favoritos' | 'historial';

  /** Destino resaltado; `null` cuando ninguno aplica (p. ej. en el detalle). */
  export let activo: DestinoBarra | null = null;
  export let onDescubrir: () => void = () => {};
  export let onBuscar: () => void = () => {};
  export let onFavoritos: () => void = () => {};
  export let onHistorial: () => void = () => {};

  const TAM = 24 as const;

  let botonHistorial: HTMLButtonElement | null = null;

  /** Devuelve el foco al destino Historial (al cerrar el cajón; T73). */
  export function enfocarHistorial(): void {
    botonHistorial?.focus();
  }
</script>

<nav class="barra-inferior" aria-label="Navegación principal">
  <button
    type="button"
    class="destino"
    aria-current={activo === 'descubrir' ? 'page' : undefined}
    on:click={onDescubrir}
  >
    <span class="indicador" aria-hidden="true"></span>
    <Icono nombre="brujula" tam={TAM} />
    <span class="etiqueta">Descubrir</span>
  </button>

  <button
    type="button"
    class="destino"
    aria-current={activo === 'buscar' ? 'page' : undefined}
    on:click={onBuscar}
  >
    <span class="indicador" aria-hidden="true"></span>
    <Icono nombre="buscar" tam={TAM} />
    <span class="etiqueta">Buscar</span>
  </button>

  <button
    type="button"
    class="destino"
    aria-current={activo === 'favoritos' ? 'page' : undefined}
    on:click={onFavoritos}
  >
    <span class="indicador" aria-hidden="true"></span>
    <Icono nombre="estrella" tam={TAM} />
    <span class="etiqueta">Favoritos</span>
  </button>

  <button
    type="button"
    class="destino"
    aria-current={activo === 'historial' ? 'page' : undefined}
    bind:this={botonHistorial}
    on:click={onHistorial}
  >
    <span class="indicador" aria-hidden="true"></span>
    <Icono nombre="historial" tam={TAM} />
    <span class="etiqueta">Historial</span>
  </button>
</nav>

<style>
  /* La barra solo existe en modo estrecho; su altura real es
     `calc(var(--bar-h) + var(--sab))` (60 px + área segura de iOS). */
  .barra-inferior {
    box-sizing: border-box;
    position: fixed;
    inset: auto 0 0 0;
    z-index: var(--z-bar);
    display: flex;
    align-items: stretch;
    min-height: calc(var(--bar-h) + var(--sab));
    padding-bottom: var(--sab);
    background: var(--surface-panel);
    border-top: 1px solid var(--border);
    box-shadow: var(--sh-card);
    backdrop-filter: blur(var(--blur-md));
    -webkit-backdrop-filter: blur(var(--blur-md));
  }
  .destino {
    position: relative;
    flex: 1 1 0;
    min-width: 0;
    min-height: var(--tap);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--sp-1);
    padding: var(--sp-1) 0;
    background: none;
    border: none;
    color: var(--text-3);
    font-family: var(--font-sans);
    font-size: var(--fs-xs);
    line-height: var(--lh-tight);
    cursor: pointer;
    transition: color var(--dur-fast) var(--ease-std);
    -webkit-tap-highlight-color: transparent;
  }
  .etiqueta {
    font-size: var(--fs-xs);
  }
  /* Indicador del activo: barra superior de 2 px (no depende solo del color). */
  .indicador {
    position: absolute;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    width: 28px;
    height: 2px;
    background: var(--brand-400);
    opacity: 0;
  }
  .destino[aria-current='page'] {
    color: var(--brand-400);
  }
  .destino[aria-current='page'] .indicador {
    opacity: 1;
  }
  /* Sin hover como única señal: `:active` y `:focus-visible`. */
  @media (hover: hover) and (pointer: fine) {
    .destino:hover {
      color: var(--text-1);
    }
    .destino[aria-current='page']:hover {
      color: var(--brand-400);
    }
  }
  .destino:active {
    color: var(--text-1);
  }
  .destino:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  @media (prefers-reduced-motion: reduce) {
    .destino {
      transition: none;
    }
  }
</style>
