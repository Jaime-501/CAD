/**
 * Envoltorios sobre los bindings generados de Wails (`frontend/wailsjs/**`, no editar).
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
import type { main, models } from '../wailsjs/go/models';

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
  subtitles?: models.Subtitle[];
}

/** Búsqueda con fallback del backend (JKAnime primero, AnimeFLV después). */
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
