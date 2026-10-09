/**
 * Envoltorios sobre los bindings generados de Wails (`frontend/wailsjs/**`, no editar).
 *
 * **ÚNICO punto del frontend que importa `wailsjs`** (T37/P4): el resto de
 * archivos (`stores.ts` y los componentes) importan de aquí — funciones,
 * suscripciones tipadas y tipos reexportados—. Ningún otro archivo debe
 * importar `../wailsjs/**`.
 *
 * Los payloads de los eventos NO se generan (solo viajan por `runtime.EventsOn`),
 * así que se declaran a mano aquí con los mismos nombres JSON que emite Go:
 * ver `docs/firmas/app.md` (eventos) y `docs/firmas/models.md` (modelos).
 */
import {
  GetAnimeDetails,
  GetProgress,
  GetVideoSources,
  SaveProgress,
  SearchAnime,
} from '../wailsjs/go/main/App.js';
import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime.js';
import type { main, models } from '../wailsjs/go/models';

// Tipos generados, reexportados para que los consumidores no importen wailsjs.
export type { main, models };

/** Payload de `episodes:update` = `models.EpisodeBatch` de Go (el error va en `err`, no en `error`). */
export interface EpisodesUpdatePayload {
  anime_id: string;
  batch: number;
  episodes: models.Episode[];
  done: boolean;
  err: string;
}

/** Payload de `episodes:complete` = `EpisodiosCompletados` de Go (string vacío = todo OK). */
export interface EpisodesCompletePayload {
  anime_id: string;
  error: string;
}

/**
 * Fuente de vídeo tal como la consume el frontend: **sin `headers` ni `referer`**.
 * El proxy local (T19) inyecta esas cabeceras en su sesión del lado de Go;
 * el navegador no debe usarlas ni mostrarlas.
 */
export interface VideoSourceView {
  url: string;
  type: string;
  quality: string;
  server: string;
  /** SUB/DUB (T33); vacío si el proveedor no lo declara (p. ej. JKAnime). */
  language: string;
  subtitles?: models.Subtitle[];
}

/** Búsqueda combinada del backend (T34): JKAnime y AnimeAV1 en paralelo, JKAnime primero. */
export function searchAnime(query: string): Promise<models.Anime[]> {
  return SearchAnime(query);
}

/** Detalle + episodios ya acumulados (caché/streaming); los lotes llegan por evento. */
export function getAnimeDetails(provider: string, animeID: string): Promise<main.DetalleResultado> {
  return GetAnimeDetails(provider, animeID);
}

/** Fuentes ordenadas con la mejor primero; `headers`/`referer` se descartan aquí. */
export async function getVideoSources(
  provider: string,
  episodeID: string,
  noCache = false
): Promise<VideoSourceView[]> {
  const fuentes = await GetVideoSources(provider, episodeID, noCache);
  return fuentes.map((f) => ({
    url: f.url,
    type: f.type,
    quality: f.quality,
    server: f.server,
    language: f.language ?? '',
    subtitles: f.subtitles,
  }));
}

/**
 * Guarda el progreso. Un fallo (historial no disponible, app cerrándose, disco...)
 * **no se lanza**: devuelve `false` y solo avisa por consola, para que el
 * reproductor (T27) nunca se rompa por esto.
 */
export async function saveProgress(progress: models.WatchProgress): Promise<boolean> {
  try {
    await SaveProgress(progress);
    return true;
  } catch (e) {
    console.warn('[api] progreso no guardado:', String(e));
    return false;
  }
}

/** Historial de un anime; `animeID` vacío = historial entero (puede fallar: ver app.md T22). */
export function getProgress(animeID: string): Promise<models.WatchProgress[]> {
  return GetProgress(animeID);
}

/** Wails serializa a JSON: el payload puede llegar como objeto o como string. */
function parsePayload<T>(crudo: unknown): T | null {
  if (typeof crudo === 'string') {
    try {
      return JSON.parse(crudo) as T;
    } catch {
      return null;
    }
  }
  if (crudo && typeof crudo === 'object') return crudo as T;
  return null;
}

/**
 * Suscripción a `episodes:update` (lotes de episodios). Devuelve la función
 * que cancela la suscripción —`EventsOff` del propio evento—; `stores.ts` la
 * guarda en `initEvents` y la llama en `disposeEvents`. Como los listeners
 * solo se crean aquí, quitar el evento los quita todos (mismo comportamiento
 * que el `EventsOff` directo de antes de T37).
 *
 * Requiere el puente Wails (window.runtime), igual que cualquier otra
 * función de este módulo.
 */
export function onEpisodesUpdate(cb: (payload: EpisodesUpdatePayload) => void): () => void {
  EventsOn('episodes:update', (crudo: unknown) => {
    const payload = parsePayload<EpisodesUpdatePayload>(crudo);
    if (payload) cb(payload);
  });
  return () => EventsOff('episodes:update');
}

/** Suscripción a `episodes:complete` (cierre del stream). Ver `onEpisodesUpdate`. */
export function onEpisodesComplete(cb: (payload: EpisodesCompletePayload) => void): () => void {
  EventsOn('episodes:complete', (crudo: unknown) => {
    const payload = parsePayload<EpisodesCompletePayload>(crudo);
    if (payload) cb(payload);
  });
  return () => EventsOff('episodes:complete');
}
