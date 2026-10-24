<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import { getProgress } from './api';
  import { etiquetaTipo } from './etiquetas';
  import {
    currentAnimeID,
    currentDetail,
    currentProvider,
    episodes,
    episodesComplete,
    episodesError,
    openAnime,
    progressMap,
    progressToMap,
  } from './stores';
  import { atras, claveScroll, navegarA, recuperarScroll, seleccion } from './views';
  import { isNarrow } from './ui';
  import { COLUMNAS_POR_DEFECTO } from './ventana';
  import { medirColumnas, revelar } from './revelar';
  import BarraAccion from './BarraAccion.svelte';
  import BotonFavorito from './BotonFavorito.svelte';
  import CabeceraVista from './CabeceraVista.svelte';
  import EstadoVista from './EstadoVista.svelte';
  import Insignia from './Insignia.svelte';
  import Icono from './Icono.svelte';
  import TarjetaEpisodio from './TarjetaEpisodio.svelte';

  let cargando = true;
  let error = '';

  // T65: columnas reales de la rejilla para el retardo de entrada por fila.
  let rejillaEl: HTMLDivElement | undefined = undefined;
  let columnas = COLUMNAS_POR_DEFECTO;
  let frameResize: number | null = null;

  // T67: restauración del scroll del detalle cuando la lista se recarga. Al
  // volver a un detalle cuyos stores no están intactos, `openAnime` vacía la
  // lista y el `scrollTo` de `App.svelte` se recorta a 0 (todavía no hay
  // altura). Aquí se arma una restauración pendiente que se aplica en cuanto
  // la lista crece lo suficiente. `0` = desarmado.
  let scrollPendiente = 0;
  let quitarOyentesScroll: (() => void) | null = null;

  function medirColumnasRejilla(): void {
    if (rejillaEl) columnas = medirColumnas(rejillaEl);
  }

  function alRedimensionar(): void {
    if (frameResize !== null) return;
    // rAF: no medir en cada píxel del arrastre de la ventana.
    frameResize = requestAnimationFrame(() => {
      frameResize = null;
      medirColumnasRejilla();
    });
  }

  onMount(() => {
    window.addEventListener('resize', alRedimensionar);
    return () => {
      window.removeEventListener('resize', alRedimensionar);
      if (frameResize !== null) cancelAnimationFrame(frameResize);
    };
  });

  // La rejilla sólo existe cuando hay episodios; se mide al montar y tras el resize.
  $: if (rejillaEl) medirColumnasRejilla();

  /** Desarma la restauración pendiente y quita sus oyentes (sin mover el scroll). */
  function desarmarRestauracionScroll(): void {
    scrollPendiente = 0;
    if (quitarOyentesScroll) {
      quitarOyentesScroll();
      quitarOyentesScroll = null;
    }
  }

  /** Arma la restauración y registra los oyentes que la cancelan al primer gesto del usuario. */
  function armarRestauracionScroll(objetivo: number): void {
    scrollPendiente = objetivo;
    if (quitarOyentesScroll) return;
    const cancelar = (): void => desarmarRestauracionScroll();
    window.addEventListener('wheel', cancelar, { passive: true });
    window.addEventListener('touchstart', cancelar, { passive: true });
    window.addEventListener('keydown', cancelar, { passive: true });
    window.addEventListener('pointerdown', cancelar, { passive: true });
    quitarOyentesScroll = () => {
      window.removeEventListener('wheel', cancelar);
      window.removeEventListener('touchstart', cancelar);
      window.removeEventListener('keydown', cancelar);
      window.removeEventListener('pointerdown', cancelar);
    };
  }

  /**
   * Intenta aplicar la restauración pendiente. Se llama en cada cambio de la
   * lista (lotes de episodios) y al completar. Mide dentro de un rAF para que
   * el layout ya esté pintado; si la altura alcanza, aplica y desarma; si el
   * stream terminó y no alcanza, baja al máximo posible y desarma.
   */
  function intentarRestaurarScroll(_longitud: number, completo: boolean): void {
    if (scrollPendiente <= 0) return;
    requestAnimationFrame(() => {
      if (scrollPendiente <= 0) return;
      const maximo = document.documentElement.scrollHeight - window.innerHeight;
      if (maximo >= scrollPendiente) {
        window.scrollTo(0, scrollPendiente);
        desarmarRestauracionScroll();
      } else if (completo) {
        window.scrollTo(0, Math.max(0, maximo));
        desarmarRestauracionScroll();
      }
    });
  }

  onMount(() => {
    const sel = get(seleccion);
    if (!sel) {
      cargando = false;
      error = 'No hay ningún anime seleccionado.';
      return;
    }
    // T67: la posición que hay que recuperar, antes de decidir la rama.
    const objetivo = recuperarScroll(claveScroll('detalle'));
    // Vuelta desde el reproductor: el detalle y los episodios siguen en los
    // stores, se conservan tal cual (y con ellos la lista y su scroll); solo
    // se refresca el historial, que puede haber cambiado en el reproductor.
    // El slug NO identifica al anime (T34): "bleach" es jkanime y animeav1,
    // así que hace falta coincidir también el proveedor.
    if (
      get(currentAnimeID) === sel.animeID &&
      get(currentProvider) === sel.provider &&
      get(currentDetail)
    ) {
      cargando = false;
      refrescarProgreso(sel.animeID, sel.provider);
      return;
    }
    // Rama de recarga: la lista se vacía y crece por lotes, así que el scroll
    // se aplica cuando haya altura suficiente.
    if (objetivo > 0) armarRestauracionScroll(objetivo);
    openAnime(sel.provider, sel.animeID)
      .catch((e) => {
        error = String(e);
      })
      .finally(() => {
        cargando = false;
      });
  });

  onDestroy(() => {
    desarmarRestauracionScroll();
  });

  /** Recarga `progressMap` sin vaciar la lista (mismo criterio de descarte que `openAnime`). */
  async function refrescarProgreso(animeID: string, proveedor: string): Promise<void> {
    try {
      const historial = await getProgress(animeID);
      if (get(currentAnimeID) !== animeID || get(currentProvider) !== proveedor) return;
      // Solo el progreso del proveedor abierto: el slug puede ser compartido
      // y el historial trae registros de todos (T34).
      progressMap.set(progressToMap(historial, proveedor));
    } catch (e) {
      console.warn('[VistaDetalle] historial no disponible:', String(e));
    }
  }

  // Los episodios llegan deduplicados por T24 (y con lotes que pueden llegar
  // desordenados o antes de resolver la llamada): aquí solo se ordenan.
  $: episodios = [...$episodes].sort((a, b) => a.number - b.number || a.id.localeCompare(b.id));

  // T82: orden de presentación de la rejilla; la lista ascendente sigue siendo
  // la fuente de la CTA, de `episodioContinuar` y de la restauración de scroll.
  let ordenDesc = false;
  $: episodiosVista = ordenDesc ? [...episodios].reverse() : episodios;
  $: if ($currentAnimeID) ordenDesc = false;
  function alternarOrden(): void {
    ordenDesc = !ordenDesc;
  }

  // T67: reintenta la restauración pendiente con cada cambio de la lista.
  $: intentarRestaurarScroll(episodios.length, $episodesComplete);

  /** Tipo del detalle en texto amable (T34): `tv-anime` → `TV`; vacío si no hay. */
  $: tipoEtiqueta = etiquetaTipo($currentDetail?.type ?? '');

  // "Continuar": el episodio de mayor número con progreso; si ya está visto,
  // el siguiente de la lista (si existe).
  $: episodioContinuar = paraContinuar(episodios, $progressMap);

  function paraContinuar(
    lista: models.Episode[],
    progreso: Record<string, models.WatchProgress>
  ): models.Episode | null {
    let mejor: models.Episode | null = null;
    for (const ep of lista) {
      const p = progreso[ep.id];
      if (!p || !(p.current_time > 0)) continue;
      if (mejor === null || ep.number > mejor.number) mejor = ep;
    }
    if (mejor === null) return null;
    if (progreso[mejor.id].is_watched) {
      const i = lista.indexOf(mejor);
      return i >= 0 && i + 1 < lista.length ? lista[i + 1] : null;
    }
    return mejor;
  }

  function abrirEpisodio(ep: models.Episode): void {
    const sel = get(seleccion);
    if (!sel) return;
    navegarA('reproductor', {
      ...sel,
      episodeID: ep.id,
      episodeNumber: ep.number,
      episodeTitle: ep.title,
    });
  }

  function continuar(): void {
    if (episodioContinuar) abrirEpisodio(episodioContinuar);
  }

  // T80: CTA fija del modo estrecho. `episodioInicio` es el primero de la lista
  // ordenada (destino de "Reiniciar" y del respaldo "Empezar"); `accionPrincipal`
  // decide la etiqueta y la acción de la CTA según haya continuidad o no.
  $: episodioInicio = episodios[0] ?? null;

  function reiniciar(): void {
    if (episodioInicio) abrirEpisodio(episodioInicio);
  }

  function accionPrincipal(
    continuarCon: models.Episode | null,
    inicio: models.Episode | null
  ): { etiqueta: string; accion: () => void } {
    if (continuarCon)
      return { etiqueta: `Continuar · Ep. ${continuarCon.number}`, accion: continuar };
    if (inicio)
      return { etiqueta: `Empezar · Ep. ${inicio.number}`, accion: () => abrirEpisodio(inicio) };
    return { etiqueta: 'Empezar', accion: () => {} };
  }

  // T80: la llamada a una función en el template se compila `untrack` en Svelte 5
  // legacy (no crea dependencias); con los argumentos explícitos, `$:` sí las ve.
  $: accion = accionPrincipal(episodioContinuar, episodioInicio);

  // T79: sinopsis con "Ver más" sólo en estrecho; se mide tras pintar y en resize.
  let sinopsisAbierta = false;
  let sinopsisDesborda = false;
  let sinopsisEl: HTMLParagraphElement | undefined = undefined;
  let frameSinopsis: number | null = null;

  function programarMedirSinopsis(): void {
    if (frameSinopsis !== null) return;
    frameSinopsis = requestAnimationFrame(() => {
      frameSinopsis = null;
      if (sinopsisAbierta || !sinopsisEl) return;
      sinopsisDesborda = $isNarrow && sinopsisEl.scrollHeight > sinopsisEl.clientHeight + 1;
    });
  }

  $: if ($currentDetail?.synopsis && $isNarrow) programarMedirSinopsis();

  onMount(() => {
    window.addEventListener('resize', programarMedirSinopsis);
    return () => {
      window.removeEventListener('resize', programarMedirSinopsis);
      if (frameSinopsis !== null) cancelAnimationFrame(frameSinopsis);
    };
  });
</script>

<main class="contenedor" class:con-cta={$isNarrow && $currentDetail && episodios.length > 0}>
  <CabeceraVista
    fija
    nivel={1}
    titulo={$currentDetail?.title ?? $seleccion?.title ?? 'Detalle'}
    onAtras={atras}
  >
    <svelte:fragment slot="acciones">
      {#if $currentDetail}
        <BotonFavorito
          provider={$currentDetail.provider}
          animeId={$currentDetail.id}
          titulo={$currentDetail.title}
          cover={$currentDetail.cover}
        />
      {/if}
    </svelte:fragment>
  </CabeceraVista>

  {#if error}
    <EstadoVista variante="error" titulo="No se pudo cargar el detalle" texto={error} />
  {:else if cargando && !$currentDetail}
    <EstadoVista variante="cargando" titulo="Cargando detalle…" />
  {:else if $currentDetail}
    <section class="ficha">
      {#if $currentDetail.cover !== ''}
        <div class="portada-marco">
          <img class="portada" src={$currentDetail.cover} alt={$currentDetail.title} />
        </div>
      {/if}
      <div class="datos">
        <div class="metadatos">
          <Insignia tono="marca">{$currentDetail.provider}</Insignia>
          {#if tipoEtiqueta !== ''}
            <Insignia tono="neutro">{tipoEtiqueta}</Insignia>
          {/if}
          {#if $currentDetail.status !== ''}
            <Insignia tono="neutro">
              <span
                class="punto"
                class:emision={$currentDetail.status
                  .normalize('NFD')
                  .replace(/[\u0300-\u036f]/g, '')
                  .toLowerCase() === 'en emision'}
                aria-hidden="true"
              ></span>
              {$currentDetail.status}
            </Insignia>
          {/if}
          <Insignia tono="neutro">
            {$currentDetail.total_episodes}
            {$currentDetail.total_episodes === 1 ? 'episodio' : 'episodios'}
          </Insignia>
        </div>
        {#if $currentDetail.genres?.length}
          <ul class="generos">
            {#each $currentDetail.genres as genero (genero)}
              <li>{genero}</li>
            {/each}
          </ul>
        {/if}
        {#if $currentDetail.synopsis !== ''}
          <p class="sinopsis" class:recortada={!sinopsisAbierta} bind:this={sinopsisEl}>
            {$currentDetail.synopsis}
          </p>
          {#if $isNarrow && sinopsisDesborda}
            <button
              type="button"
              class="ver-mas"
              aria-expanded={sinopsisAbierta}
              on:click={() => (sinopsisAbierta = !sinopsisAbierta)}
            >
              {sinopsisAbierta ? 'Ver menos' : 'Ver más'}
              <Icono nombre="chevron-abajo" tam={16} />
            </button>
          {/if}
        {/if}
        <button
          type="button"
          class="continuar"
          disabled={!episodioContinuar}
          on:click={continuar}
        >
          {#if episodioContinuar}
            Continuar · Ep. {episodioContinuar.number}
          {:else}
            Continuar
          {/if}
        </button>
      </div>
    </section>

    <div class="fila-titular">
      <h2 class="titular">Episodios {#if episodios.length > 0}({episodios.length}){/if}</h2>
      {#if episodios.length > 1}
        <button
          type="button"
          class="orden"
          aria-label={ordenDesc
            ? 'Ordenar del más antiguo al más reciente'
            : 'Ordenar del más reciente al más antiguo'}
          aria-pressed={ordenDesc}
          on:click={alternarOrden}
        >
          <Icono nombre={ordenDesc ? 'orden-asc' : 'orden-desc'} tam={20} />
          <span class="orden-texto">{ordenDesc ? 'Más antiguos' : 'Más recientes'}</span>
        </button>
      {/if}
    </div>

    {#if $episodesError !== ''}
      <p class="aviso">No se pudieron cargar todos los episodios: {$episodesError}</p>
    {/if}

    {#if episodios.length > 0}
      <div class="rejilla" bind:this={rejillaEl}>
        {#each episodiosVista as ep, i (ep.id)}
          <div class="celda" use:revelar={{ id: ep.id, indice: i, columnas }}>
            <TarjetaEpisodio episodio={ep} progreso={$progressMap[ep.id]} abrir={abrirEpisodio} />
          </div>
        {/each}
      </div>
    {/if}

    {#if !$episodesComplete}
      <p class="cargando" aria-live="polite">Cargando más episodios…</p>
    {:else if episodios.length === 0}
      <EstadoVista variante="vacio" titulo="No hay episodios disponibles." />
    {/if}
  {:else}
    <EstadoVista variante="vacio" titulo="Sin detalle disponible." />
  {/if}
</main>

{#if $isNarrow && $currentDetail && episodios.length > 0}
  <BarraAccion
    etiquetaPrincipal={accion.etiqueta}
    onPrincipal={accion.accion}
    etiquetaSecundaria={episodioContinuar ? 'Reiniciar' : ''}
    onSecundaria={reiniciar}
  />
{/if}

<style>
  .contenedor {
    /* T80: alto de la CTA fija (tópico local, no global). */
    --cta-h: 60px;
    padding: 2rem;
    text-align: left;
    max-width: 1100px;
    margin: 0 auto;
  }
  .ficha {
    display: flex;
    gap: var(--sp-5);
    margin-bottom: 2rem;
    background: var(--surface-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-xl);
    box-shadow: var(--sh-card);
    padding: var(--sp-4);
  }
  .portada-marco {
    box-sizing: border-box;
    flex-shrink: 0;
    width: 220px;
    max-width: 35%;
    height: fit-content;
    padding: 3px;
    background: var(--frame-poster);
    border-radius: var(--r-lg);
    box-shadow: var(--sh-panel);
  }
  .portada {
    display: block;
    width: 100%;
    aspect-ratio: 2 / 3;
    object-fit: cover;
    border-radius: var(--r-md);
    background: var(--bg-2);
  }
  .datos {
    flex: 1;
    min-width: 0;
  }
  .metadatos {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--sp-2);
    margin: 0 0 var(--sp-3);
  }
  .punto {
    flex-shrink: 0;
    width: var(--sp-2);
    height: var(--sp-2);
    border-radius: var(--r-full);
    background: var(--text-3);
  }
  .punto.emision {
    background: var(--brand-400);
  }
  .generos {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: var(--sp-2);
    margin: 0 0 var(--sp-3);
    padding: 0;
  }
  .generos li {
    background: var(--surface-chip);
    color: var(--brand-400);
    font-size: var(--fs-xs);
    padding: var(--sp-1) var(--sp-3);
    border-radius: var(--r-full);
  }
  .sinopsis {
    margin: 0 0 var(--sp-4);
    line-height: var(--lh);
    color: var(--text-2);
    white-space: pre-line;
  }
  .ver-mas {
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
    min-height: var(--tap);
    margin: 0 0 var(--sp-2);
    padding: 0;
    border: none;
    background: none;
    color: var(--brand-400);
    font: inherit;
    font-size: var(--fs-sm);
    font-weight: var(--fw-semibold);
    cursor: pointer;
  }
  .continuar {
    padding: 10px var(--sp-4);
    border-radius: var(--r-md);
    border: none;
    background: var(--brand-500);
    color: var(--on-brand);
    font-weight: bold;
    cursor: pointer;
  }
  .continuar:disabled {
    background: var(--bg-2);
    color: var(--text-3);
    cursor: not-allowed;
  }
  .fila-titular {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-3);
    margin: 0 0 1rem;
  }
  .titular {
    margin: 0;
    /* Conserva el tamaño del h3 anterior (T88). */
    font-size: 1.17em;
  }
  .orden {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    gap: var(--sp-1);
    min-width: var(--tap);
    min-height: var(--tap);
    padding: 0 var(--sp-3);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--surface-chip);
    color: var(--text-2);
    font: inherit;
    font-size: var(--fs-sm);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    transition:
      background var(--dur-fast) var(--ease-std),
      border-color var(--dur-fast) var(--ease-std),
      transform var(--dur-fast) var(--ease-std);
    -webkit-tap-highlight-color: transparent;
  }
  .orden-texto {
    display: none;
  }
  @media (min-width: 480px) {
    .orden-texto {
      display: inline;
    }
  }
  @media (min-width: 900px) {
    .orden {
      height: 36px;
      min-height: 36px;
    }
  }
  @media (hover: hover) and (pointer: fine) {
    .orden:hover {
      background: var(--surface-chip-hover);
      color: var(--text-1);
    }
  }
  .orden:active {
    transform: scale(0.98);
  }
  .orden:focus-visible {
    outline: none;
    box-shadow: var(--ring);
  }
  @media (prefers-reduced-motion: reduce) {
    .orden {
      transition: none;
    }
    .orden:active {
      transform: none;
    }
  }
  .rejilla {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
    gap: 18px;
  }
  /* T65: la animación de entrada vive en el envoltorio `.celda` (no en la
     tarjeta) para no chocar con el transform del hover ni con
     content-visibility. La tarjeta ocupa todo el alto disponible. */
  .celda {
    display: flex;
    flex-direction: column;
  }
  .celda :global(.tarjeta) {
    flex: 1 1 auto;
  }
  .celda:global(.pendiente) {
    opacity: 0;
  }
  .celda:global(.entra) {
    animation: entra-celda 320ms cubic-bezier(0.22, 0.61, 0.36, 1) both;
  }
  @keyframes entra-celda {
    from {
      opacity: 0;
      transform: translateX(-18px);
    }
    to {
      opacity: 1;
      transform: none;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .celda:global(.pendiente) {
      opacity: 1;
    }
    .celda:global(.entra) {
      animation: none;
    }
  }
  .aviso {
    color: var(--fav);
    background: var(--bg-2);
    border-radius: var(--r-md);
    padding: 10px 14px;
    margin-bottom: 1rem;
  }
  .cargando {
    color: var(--text-3);
    font-size: var(--fs-sm);
    text-align: center;
    padding: 1rem 0;
  }
  @media (max-width: 899px) {
    /* T74: el padding superior lo da la cabecera fija. */
    .contenedor {
      padding: 0 var(--gutter) var(--sp-4);
    }
    /* T80: el botón "Continuar" en línea se sustituye por la CTA fija. */
    .continuar {
      display: none;
    }
    /* T80: deja hueco para la CTA fija (solo cuando está visible). */
    .contenedor.con-cta {
      padding-bottom: calc(var(--sp-4) + var(--cta-h));
    }
    /* T79: 641–899 px también pasa a columna (antes sólo < 641 px): portada
       centrada arriba y datos debajo. */
    .ficha {
      flex-direction: column;
      padding: var(--sp-4);
    }
    .portada-marco {
      width: min(45%, 200px);
      max-width: 200px;
      align-self: center;
    }
    /* T79: géneros en una línea con scroll horizontal, sangrados al borde de la
       ficha (márgenes negativos = padding de `.ficha`), sin barra nativa. */
    .generos {
      flex-wrap: nowrap;
      overflow-x: auto;
      margin-left: calc(-1 * var(--sp-4));
      margin-right: calc(-1 * var(--sp-4));
      padding: 0 var(--sp-4);
      scroll-snap-type: x proximity;
      scrollbar-width: none;
    }
    .generos::-webkit-scrollbar {
      display: none;
    }
    .generos li {
      flex-shrink: 0;
      scroll-snap-align: start;
    }
    /* T79: sinopsis recortada a 4 líneas mientras no está abierta. */
    .sinopsis.recortada {
      display: -webkit-box;
      -webkit-line-clamp: 4;
      line-clamp: 4;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
    /* T81: la rejilla de episodios pasa a 2 columnas en estrecho. */
    .rejilla {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--sp-3);
    }
  }
  /* T81: 600–899 px usa 3 columnas. */
  @media (min-width: 600px) and (max-width: 899px) {
    .rejilla {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }
</style>
