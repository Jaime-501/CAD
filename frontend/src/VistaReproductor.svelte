<script lang="ts">
  import Hls from 'hls.js';
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import { getProgress, getVideoSources, saveProgress } from './api';
  import type { VideoSourceView } from './api';
  import type { FiltroIdioma } from './etiquetas';
  import { progressMap } from './stores';
  import { atras, seleccion, type SeleccionAnime } from './views';
  import SelectorFuente from './SelectorFuente.svelte';

  /** Throttle del guardado en `timeupdate` (T27: ~5 s). */
  const INTERVALO_GUARDADO = 5000;

  let cargando = true;
  let cargandoTexto = 'Cargando fuentes de vídeo…';
  let error = '';
  /** Fuentes tal y como las devuelve el backend: la primera es la preferida. */
  let fuentes: VideoSourceView[] = [];
  let indice = 0;
  /**
   * Filtro visual de idioma del selector (T34): `null` = sin filtro, que es
   * lo que hay hoy (se ven SUB y DUB por igual). Cambiar este valor a
   * `'SUB'`/`'DUB'` activa el filtro sin tocar nada más (la lógica vive en
   * `fuentesVisibles`).
   */
  let filtroIdioma: FiltroIdioma = null;
  /** Fuentes ya probadas: índice de la siguiente a intentar tras un fallo fatal. */
  let intentos = 0;
  let modo: 'nada' | 'video' | 'iframe' = 'nada';
  /** Evita recuperaciones concurrentes (varios errores seguidos). */
  let ocupado = false;
  let videoEl: HTMLVideoElement | null = null;
  let hls: Hls | null = null;
  /** Posición a restaurar en el próximo `<video>` (progreso guardado o en curso). */
  let progresoPrevio: models.WatchProgress | null = null;
  let ultimoGuardado = 0;

  $: fuenteActual = fuentes[indice] ?? null;
  $: avisoProxy = avisoDeProxy(fuenteActual);
  $: pistas = modo === 'video' && fuenteActual ? (fuenteActual.subtitles ?? []) : [];
  $: lineaEpisodio = etiquetaEpisodio($seleccion);

  onMount(() => {
    const sel = get(seleccion);
    if (videoEl) {
      videoEl.addEventListener('timeupdate', alAvanzar);
      videoEl.addEventListener('pause', alPausar);
      videoEl.addEventListener('ended', alTerminar);
      videoEl.addEventListener('loadedmetadata', alMetadatos);
    }
    window.addEventListener('beforeunload', alSalir);

    if (!sel?.episodeID) {
      cargando = false;
      error = 'No hay ningún episodio seleccionado.';
      return;
    }
    void arrancar(sel);
  });

  onDestroy(() => {
    guardarProgreso(true);
    if (videoEl) {
      videoEl.removeEventListener('timeupdate', alAvanzar);
      videoEl.removeEventListener('pause', alPausar);
      videoEl.removeEventListener('ended', alTerminar);
      videoEl.removeEventListener('loadedmetadata', alMetadatos);
      videoEl.removeEventListener('error', alErrorFuente);
    }
    window.removeEventListener('beforeunload', alSalir);
    limpiarReproduccion();
  });

  async function arrancar(sel: SeleccionAnime): Promise<void> {
    progresoPrevio = await recuperarProgreso(sel);
    if (get(seleccion)?.episodeID !== sel.episodeID) return;
    await cargarFuentes(false);
  }

  /** Progreso del episodio: primero el mapa de la store; si no está, el backend. */
  async function recuperarProgreso(sel: SeleccionAnime): Promise<models.WatchProgress | null> {
    const episodeID = sel.episodeID;
    if (!episodeID) return null;
    const local = get(progressMap)[episodeID];
    if (local) return local;
    try {
      const historial = await getProgress(sel.animeID);
      return historial.find((p) => p.episode_id === episodeID) ?? null;
    } catch (e) {
      // Historial no disponible (T22): se empieza desde cero, sin molestar.
      console.warn('[VistaReproductor] historial no disponible:', texto(e));
      return null;
    }
  }

  async function cargarFuentes(noCache: boolean): Promise<void> {
    const sel = get(seleccion);
    if (!sel?.episodeID) return;
    cargando = true;
    cargandoTexto = 'Cargando fuentes de vídeo…';
    error = '';
    try {
      const lista = await getVideoSources(sel.provider, sel.episodeID, noCache);
      if (get(seleccion)?.episodeID !== sel.episodeID) return;
      if (lista.length === 0) {
        cargando = false;
        error = 'No se encontraron fuentes de vídeo para este episodio.';
        return;
      }
      fuentes = lista;
      indice = 0;
      intentos = 1;
      cargando = false;
      reproducir();
    } catch (e) {
      if (get(seleccion)?.episodeID !== sel.episodeID) return;
      cargando = false;
      error = `No se pudieron cargar las fuentes de vídeo: ${texto(e)}`;
    }
  }

  /** Monta la fuente actual sobre el `<video>` o el `<iframe>`. */
  function reproducir(): void {
    const fuente = fuentes[indice];
    const v = videoEl;
    if (!fuente || !v) return;
    limpiarReproduccion();
    error = '';

    if (fuente.type === 'iframe') {
      modo = 'iframe';
      return;
    }

    modo = 'video';
    v.addEventListener('error', alErrorFuente);

    if (fuente.type === 'hls') {
      if (Hls.isSupported()) {
        const instancia = new Hls();
        hls = instancia;
        instancia.on(Hls.Events.ERROR, (_evento, datos) => {
          if (datos.fatal) void fallarFuente(`hls.js: ${datos.details}`);
        });
        instancia.attachMedia(v);
        instancia.loadSource(fuente.url);
        intentarReproducir();
        return;
      }
      // Sin MSE solo queda el HLS nativo (Safari); si tampoco hay, siguiente fuente.
      if (v.canPlayType('application/vnd.apple.mpegurl') === '') {
        void fallarFuente('el entorno no admite HLS');
        return;
      }
    }

    v.src = fuente.url;
    v.load();
    intentarReproducir();
  }

  /** Autoplay: si el webview lo bloquea, siguen los controles y sin error. */
  function intentarReproducir(): void {
    const v = videoEl;
    if (!v) return;
    try {
      const promesa = v.play();
      if (promesa && typeof promesa.catch === 'function') {
        promesa.catch(() => {
          /* autoplay bloqueado: no es un error */
        });
      }
    } catch {
      /* autoplay bloqueado: no es un error */
    }
  }

  /**
   * Fallo fatal (hls.js o `<video>`): se limpia todo, se piden fuentes frescas
   * (`noCache=true`; las URLs firmadas caducan en minutos) y se prueba la
   * siguiente. Si se agotan, error con botón "Reintentar".
   */
  async function fallarFuente(detalle: string): Promise<void> {
    if (ocupado) return;
    ocupado = true;
    guardarProgreso(true);
    cargando = true;
    cargandoTexto = 'Buscando una fuente alternativa…';
    limpiarReproduccion();

    const sel = get(seleccion);
    let lista = fuentes;
    if (sel?.episodeID) {
      try {
        const frescas = await getVideoSources(sel.provider, sel.episodeID, true);
        if (get(seleccion)?.episodeID !== sel.episodeID) {
          ocupado = false;
          return;
        }
        if (frescas.length > 0) lista = frescas;
      } catch (e) {
        // Sin refresco disponible: se prueban las que ya había.
        console.warn('[VistaReproductor] no se pudieron refrescar las fuentes:', texto(e));
      }
    }

    cargando = false;
    if (intentos >= lista.length) {
      modo = 'nada';
      error = `No se pudo reproducir ninguna fuente de vídeo (${detalle}).`;
      ocupado = false;
      return;
    }
    fuentes = lista;
    indice = intentos;
    intentos += 1;
    reproducir();
    ocupado = false;
  }

  /**
   * Cambio manual de fuente (p. ej. SUB → DUB): se guarda la posición ANTES
   * de soltar el `<video>`, y al cargar los metadatos de la nueva fuente
   * `alMetadatos` la restaura, así que se reanuda en el mismo segundo.
   * Límites: si la fuente nueva es un iframe no hay metadatos (seguirá en la
   * posición guardada al volver a un vídeo), y si el episodio ya está visto
   * arranca en cero.
   */
  function elegirFuente(nuevo: number): void {
    if (ocupado || cargando) return;
    if (nuevo < 0 || nuevo >= fuentes.length) return;
    if (nuevo === indice) return;
    guardarProgreso(true);
    error = '';
    indice = nuevo;
    if (intentos <= nuevo) intentos = nuevo + 1;
    reproducir();
  }

  function reintentar(): void {
    if (ocupado || cargando || !get(seleccion)?.episodeID) return;
    error = '';
    intentos = 0;
    indice = 0;
    void cargarFuentes(true);
  }

  function alErrorFuente(): void {
    void fallarFuente('el navegador no pudo cargar el recurso');
  }

  /**
   * Restaura la posición guardada cuando el recurso ya tiene metadatos.
   * Vale tanto al abrir como al cambiar de fuente (`elegirFuente` guarda
   * justo antes): si la nueva fuente no llega a metadatos (iframe, error de
   * red) se queda en 0 y la posición sigue guardada para el próximo intento.
   */
  function alMetadatos(): void {
    const v = videoEl;
    const p = progresoPrevio;
    if (!v || modo !== 'video' || !p || p.is_watched) return;
    if (!(p.current_time > 0) || v.currentTime !== 0) return;
    try {
      v.currentTime = p.current_time;
    } catch {
      /* posición aún no válida */
    }
  }

  function alAvanzar(): void {
    guardarProgreso(false);
  }
  function alPausar(): void {
    guardarProgreso(true);
  }
  function alTerminar(): void {
    guardarProgreso(true);
  }
  function alSalir(): void {
    guardarProgreso(true);
  }

  /**
   * Guarda el progreso con throttle (~5 s en `timeupdate`; siempre en pausa,
   * fin, cierre y desmontaje). `saveProgress` nunca lanza: si devuelve false
   * la reproducción sigue igual (sin avisos en la UI).
   */
  function guardarProgreso(forzar: boolean): void {
    const sel = get(seleccion);
    const v = videoEl;
    if (!sel?.episodeID || !v || modo !== 'video') return;
    const ahora = Date.now();
    if (!forzar && ahora - ultimoGuardado < INTERVALO_GUARDADO) return;
    const actual = sanea(v.currentTime);
    if (!(actual > 0)) return;
    ultimoGuardado = ahora;
    const total = sanea(v.duration);
    // La posición recién guardada es la que se restaura si cambia la fuente.
    progresoPrevio = {
      anime_id: sel.animeID,
      episode_id: sel.episodeID,
      current_time: actual,
      total_duration: total,
      is_watched: false,
    };
    void saveProgress(progresoPrevio);
  }

  /** Limpia hls.js, el `<video>` y el listener de error de la fuente actual. */
  function limpiarReproduccion(): void {
    if (hls) {
      hls.destroy();
      hls = null;
    }
    const v = videoEl;
    if (!v) return;
    v.removeEventListener('error', alErrorFuente);
    v.pause();
    if (v.getAttribute('src') !== null) {
      v.removeAttribute('src');
      v.load();
    }
  }

  /** El proxy registra HLS/MP4 en 127.0.0.1: si la URL no es local, no arrancó. */
  function avisoDeProxy(f: VideoSourceView | null): string {
    if (!f || (f.type !== 'hls' && f.type !== 'mp4')) return '';
    if (/^https?:\/\/(127\.0\.0\.1|localhost|\[::1\])([:/]|$)/i.test(f.url)) return '';
    return 'El proxy local no está activo: el vídeo puede fallar por CORS o por las cabeceras del servidor.';
  }

  function etiquetaEpisodio(sel: SeleccionAnime | null): string {
    if (!sel) return '';
    const partes: string[] = [];
    if (typeof sel.episodeNumber === 'number' && sel.episodeNumber > 0) {
      partes.push(`Ep. ${sel.episodeNumber}`);
    }
    const titulo = sel.episodeTitle?.trim() ?? '';
    if (titulo !== '') partes.push(titulo);
    return partes.join(' · ');
  }

  function sanea(n: number): number {
    return Number.isFinite(n) && n > 0 ? n : 0;
  }

  function texto(e: unknown): string {
    return e instanceof Error ? e.message : String(e);
  }
</script>

<main class="contenedor">
  <div class="cabecera">
    <button type="button" class="atras" on:click={atras}>← Atrás</button>
    <div class="titulos">
      <h2>{$seleccion?.title ?? 'Reproductor'}</h2>
      {#if lineaEpisodio !== ''}
        <p class="linea">{lineaEpisodio}</p>
      {/if}
    </div>
  </div>

  {#if error !== ''}
    <div class="error-caja">
      <p class="error">{error}</p>
      {#if $seleccion?.episodeID}
        <button type="button" class="reintentar" on:click={reintentar}>Reintentar</button>
      {/if}
    </div>
  {/if}

  {#if avisoProxy !== ''}
    <p class="aviso">{avisoProxy}</p>
  {/if}

  <div class="marco">
    <video
      bind:this={videoEl}
      class="video"
      class:visible={modo === 'video'}
      controls
      crossorigin="anonymous"
    >
      {#each pistas as pista}
        <track kind="subtitles" src={pista.url} srclang={pista.lang} label={pista.lang} />
      {/each}
    </video>

    {#if modo === 'iframe' && fuenteActual}
      <iframe
        class="embebido"
        src={fuenteActual.url}
        title="Fuente embebida"
        referrerpolicy="no-referrer"
        allowfullscreen
      ></iframe>
    {/if}

    {#if modo === 'nada' && !cargando}
      <p class="placeholder">Sin reproducción.</p>
    {/if}

    {#if cargando}
      <div class="cargando-overlay">{cargandoTexto}</div>
    {/if}
  </div>

  {#if modo === 'iframe'}
    <p class="aviso">Fuente embebida: sin control de progreso.</p>
  {/if}

  {#if fuenteActual}
    <SelectorFuente {fuentes} filtro={filtroIdioma} actual={indice} elegir={elegirFuente} />
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
  .titulos {
    min-width: 0;
  }
  .linea {
    margin: 4px 0 0;
    color: #a1a1aa;
    font-size: 14px;
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
  .error-caja {
    background: #2a2a35;
    border: 1px solid #f87171;
    border-radius: 8px;
    padding: 12px 14px;
    margin-bottom: 1rem;
    display: flex;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
  }
  .error {
    margin: 0;
    color: #f87171;
    flex: 1;
    min-width: 220px;
  }
  .reintentar {
    padding: 8px 16px;
    border-radius: 8px;
    border: none;
    background: #4ade80;
    color: #111;
    font-weight: bold;
    cursor: pointer;
  }
  .aviso {
    color: #fbbf24;
    background: #2a2a35;
    border-radius: 8px;
    padding: 10px 14px;
    margin: 0 0 1rem;
  }
  .marco {
    position: relative;
    width: 100%;
    aspect-ratio: 16 / 9;
    background: #000;
    border-radius: 12px;
    overflow: hidden;
  }
  .video {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    display: none;
    background: #000;
  }
  .video.visible {
    display: block;
  }
  .embebido {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border: none;
    background: #000;
  }
  .placeholder {
    position: absolute;
    inset: 0;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #a1a1aa;
    font-size: 15px;
  }
  .cargando-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.65);
    color: #f4f4f5;
    font-size: 15px;
    z-index: 2;
  }
</style>
