<script lang="ts">
  import Hls from 'hls.js';
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import { getNextEpisode, getProgress, getVideoSources, saveProgress } from './api';
  import type { VideoSourceView } from './api';
  import type { FiltroIdioma } from './etiquetas';
  import { currentDetail, episodes, progressMap, startAt } from './stores';
  import { atras, navegarA, seleccion, type SeleccionAnime } from './views';
  import BotonSiguiente from './BotonSiguiente.svelte';
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
  /**
   * T50: posición que pidió `retomar` (`startAt`, en segundos). Se consume al
   * montar (el store vuelve a 0), así que solo afecta al **primer**
   * `loadedmetadata` de este montaje: no se reaplica al cambiar de fuente ni a
   * los iframes.
   */
  let inicioPendiente = 0;
  let ultimoGuardado = 0;
  /**
   * T55 — botón "Siguiente ▸": el cálculo lo hace Go (`getNextEpisode`, al
   * montar y con cada cambio de episodio). `sigCargando` es la llamada en
   * vuelo; `sigEncontrado`/`sigNumero`/`sigEpisodioID` son la respuesta
   * vigente (`found: false` o error ⇒ botón `oculto`); `resaltadoSiguiente`
   * se activa una vez (pegajoso) al 90 % de un vídeo HLS/MP4 y se limpia al
   * cambiar de episodio; nunca se resalta en iframes.
   */
  let sigCargando = true;
  let sigEncontrado = false;
  let sigNumero = 0;
  let sigEpisodioID = '';
  let resaltadoSiguiente = false;
  /**
   * T55: al pasar de episodio con el botón se arranca en 0 (`startAt = 0` de
   * la ficha): el primer `loadedmetadata` del episodio nuevo NO restaura
   * `progresoPrevio` (aunque arranque tras retomar con posición, ese camino
   * ya lo cubre `inicioPendiente` en el montaje).
   */
  let empezarDesdeCero = false;
  /** Episodio cuyas fuentes/`getNextEpisode` están activos (guardia de carreras). */
  let episodioActual = '';
  /**
   * T57: si el `<video>` está reproduciéndose. Se actualiza dentro de los
   * handlers que ya existen (`alAvanzar`/`alPausar`/`alTerminar`), sin
   * listeners nuevos; `v.paused` es la fuente de verdad en cada evento.
   */
  let reproduciendo = false;

  $: estadoSiguiente = calcularEstadoSiguiente(sigCargando, sigEncontrado, resaltadoSiguiente);
  $: discretoSiguiente = calcularDiscreto(modo, estadoSiguiente, reproduciendo, resaltadoSiguiente);
  $: sincronizarEpisodio($seleccion?.episodeID ?? '');

  $: fuenteActual = fuentes[indice] ?? null;
  $: avisoProxy = avisoDeProxy(fuenteActual);
  $: pistas = modo === 'video' && fuenteActual ? (fuenteActual.subtitles ?? []) : [];
  $: lineaEpisodio = etiquetaEpisodio($seleccion);

  onMount(() => {
    // T50: la posición de "retomar" se toma UNA vez aquí y el store queda a 0.
    inicioPendiente = get(startAt);
    startAt.set(0);

    const sel = get(seleccion);
    if (videoEl) {
      videoEl.addEventListener('timeupdate', alAvanzar);
      videoEl.addEventListener('pause', alPausar);
      videoEl.addEventListener('ended', alTerminar);
      videoEl.addEventListener('loadedmetadata', alMetadatos);
    }
    window.addEventListener('beforeunload', alSalir);
    window.addEventListener('keydown', alTecla);

    if (!sel?.episodeID) {
      cargando = false;
      sigCargando = false;
      error = 'No hay ningún episodio seleccionado.';
      return;
    }
    episodioActual = sel.episodeID;
    void arrancar(sel);
    void pedirSiguiente(sel);
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
    window.removeEventListener('keydown', alTecla);
    limpiarReproduccion();
  });

  async function arrancar(sel: SeleccionAnime): Promise<void> {
    progresoPrevio = await recuperarProgreso(sel);
    if (get(seleccion)?.episodeID !== sel.episodeID) return;
    await cargarFuentes(false);
  }

  /**
   * T55: pide a Go el siguiente episodio (menor número estrictamente mayor).
   * Si el episodio cambia mientras la llamada está en vuelo, la respuesta
   * vieja se ignora (comparación con el `episodeID` vigente).
   */
  async function pedirSiguiente(sel: SeleccionAnime): Promise<void> {
    if (!sel.episodeID) {
      sigCargando = false;
      sigEncontrado = false;
      return;
    }
    sigCargando = true;
    try {
      const sig = await getNextEpisode(sel.provider, sel.animeID, sel.episodeID);
      if (get(seleccion)?.episodeID !== sel.episodeID) return;
      sigCargando = false;
      if (!sig.found) {
        sigEncontrado = false;
        return;
      }
      sigEncontrado = true;
      sigNumero = sig.number;
      sigEpisodioID = sig.episode_id;
    } catch (e) {
      if (get(seleccion)?.episodeID !== sel.episodeID) return;
      // Película/serie sin siguiente o red caída: el botón queda oculto.
      console.warn('[VistaReproductor] siguiente episodio no disponible:', texto(e));
      sigCargando = false;
      sigEncontrado = false;
    }
  }

  /**
   * Cambio de episodio DENTRO del reproductor (T55, misma vía de T50 sin
   * pasar por el detalle): `navegarA('reproductor', …)` fija `seleccion` sin
   * remontar la vista, así que aquí se reinicia la carga de fuentes y se
   * vuelve a pedir el siguiente. El primer cambio lo ignora: es el montaje.
   */
  function sincronizarEpisodio(episodeID: string): void {
    if (episodeID === episodioActual) return;
    const eraInicial = episodioActual === '';
    episodioActual = episodeID;
    if (eraInicial) return;
    reiniciarEpisodio();
  }

  function reiniciarEpisodio(): void {
    const sel = get(seleccion);
    if (!sel?.episodeID) return;
    limpiarReproduccion();
    resaltadoSiguiente = false;
    empezarDesdeCero = true;
    fuentes = [];
    indice = 0;
    intentos = 0;
    modo = 'nada';
    ocupado = false;
    error = '';
    cargando = true;
    cargandoTexto = 'Cargando fuentes de vídeo…';
    sigCargando = true;
    sigEncontrado = false;
    void arrancar(sel);
    void pedirSiguiente(sel);
  }

  /** Clic del botón (o tecla N): guarda la posición actual y salta al siguiente. */
  function alPulsarSiguiente(): void {
    if (estadoSiguiente !== 'listo' && estadoSiguiente !== 'resaltado') return;
    if (sigEpisodioID === '') return;
    const sel = get(seleccion);
    if (!sel?.episodeID) return;
    guardarProgreso(true);
    // Mismo objeto de sel + el episodio devuelto; `startAt = 0` (no se toca el
    // store: `empezarDesdeCero` cubre el reinicio dentro del reproductor).
    const titulo = get(episodes).find((ep) => ep.id === sigEpisodioID)?.title ?? '';
    navegarA('reproductor', {
      provider: sel.provider,
      animeID: sel.animeID,
      title: sel.title,
      episodeID: sigEpisodioID,
      episodeNumber: sigNumero,
      episodeTitle: titulo,
    });
  }

  /** Atajo N (T55): igual que clic si el botón está listo/resaltado. */
  function alTecla(e: KeyboardEvent): void {
    if (e.key.toLowerCase() !== 'n') return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;
    const destino = e.target as HTMLElement | null;
    if (destino) {
      const tag = destino.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || destino.isContentEditable) return;
    }
    alPulsarSiguiente();
  }

  function calcularEstadoSiguiente(
    cargandoSiguiente: boolean,
    encontrado: boolean,
    resaltado: boolean
  ): 'cargando' | 'listo' | 'resaltado' | 'oculto' {
    if (cargandoSiguiente) return 'cargando';
    if (!encontrado) return 'oculto';
    return resaltado ? 'resaltado' : 'listo';
  }

  /**
   * T57: aspecto discreto del botón. Tabla de la ficha: HLS/MP4 reproduciendo
   * con progreso < 0.90 ⇒ `true`; en pausa/sin iniciar/terminado/progreso
   * ≥ 0.90 (⇒ `resaltadoSiguiente`, pegajoso) ⇒ `false`; fuente iframe ⇒
   * `true` siempre. Solo aplica en el estado `listo` (`resaltado` nunca es
   * discreto; `cargando`/`oculto` no cambian).
   */
  function calcularDiscreto(
    modoActual: 'nada' | 'video' | 'iframe',
    estado: 'cargando' | 'listo' | 'resaltado' | 'oculto',
    reproduciendoActual: boolean,
    resaltado: boolean
  ): boolean {
    if (estado !== 'listo') return false;
    if (resaltado) return false;
    if (modoActual === 'iframe') return true;
    if (modoActual !== 'video') return false;
    return reproduciendoActual;
  }

  /** Realce al 90 % (T55): solo con control de progreso (HLS/MP4), pegajoso. */
  function marcarResaltadoSiTermina(): void {
    if (resaltadoSiguiente || modo !== 'video') return;
    const v = videoEl;
    if (!v || !(v.duration > 0)) return;
    if (v.currentTime / v.duration >= 0.9) resaltadoSiguiente = true;
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
   * Restaura la posición cuando el recurso ya tiene metadatos.
   * T50 (retomar): si quedó `inicioPendiente`, manda él — un solo intento, en
   * el primer `<video>` con metadatos; si `startAt` no deja margen de 5 s antes
   * del final (o no hay duración utilizable) se empieza en 0. Consumido, no se
   * vuelve a aplicar al cambiar de fuente.
   * T55 (siguiente): si el episodio se abrió con el botón "Siguiente ▸" se
   * arranca en 0 (`empezarDesdeCero`, un solo intento igual que `inicioPendiente`).
   * T27 (progreso guardado): en los demás casos, la posición de `progresoPrevio`
   * —igual que antes—, tanto al abrir como al cambiar de fuente (`elegirFuente`
   * guarda justo antes): si la fuente nueva no llega a metadatos (iframe, error
   * de red) se queda en 0 y la posición sigue guardada para el próximo intento.
   */
  function alMetadatos(): void {
    const v = videoEl;
    if (!v || modo !== 'video') return;

    if (inicioPendiente > 0) {
      const objetivo = inicioPendiente;
      inicioPendiente = 0;
      const duracion = v.duration;
      if (v.currentTime === 0 && objetivo < duracion - 5) {
        try {
          v.currentTime = objetivo;
        } catch {
          /* posición aún no válida */
        }
      }
      return;
    }

    if (empezarDesdeCero) {
      empezarDesdeCero = false;
      return;
    }

    const p = progresoPrevio;
    if (!p || p.is_watched) return;
    if (!(p.current_time > 0) || v.currentTime !== 0) return;
    try {
      v.currentTime = p.current_time;
    } catch {
      /* posición aún no válida */
    }
  }

  function alAvanzar(): void {
    // T57: un `timeupdate` con `paused` a `true` es un seek en pausa.
    const v = videoEl;
    reproduciendo = !!v && !v.paused;
    guardarProgreso(false);
    marcarResaltadoSiTermina();
  }
  function alPausar(): void {
    reproduciendo = false;
    guardarProgreso(true);
  }
  function alTerminar(): void {
    reproduciendo = false;
    guardarProgreso(true);
    marcarResaltadoSiTermina();
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
      provider: sel.provider,
      anime_title: sel.title,
      cover_url: portadaDe(sel),
      episode_number: numeroDe(sel.episodeID),
      updated_at: 0,
    };
    void saveProgress(progresoPrevio);
  }

  /**
   * Portada del anime abierto en la app (T49): sale del detalle que ya cargó
   * `VistaDetalle` para este (proveedor, slug); '' si no está disponible.
   */
  function portadaDe(sel: SeleccionAnime): string {
    const detalle = get(currentDetail);
    if (!detalle || detalle.id !== sel.animeID) return '';
    return detalle.cover ?? '';
  }

  /** Número del episodio parseado del id (`…:episode:N` / `…:special:N`); 0 en película. */
  function numeroDe(episodeID: string): number {
    const coincidencia = /:(?:episode|special):(\d+)$/.exec(episodeID);
    return coincidencia ? Number.parseInt(coincidencia[1], 10) : 0;
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

    <BotonSiguiente
      estado={estadoSiguiente}
      numero={sigNumero}
      discreto={discretoSiguiente}
      onClick={alPulsarSiguiente}
    />
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
