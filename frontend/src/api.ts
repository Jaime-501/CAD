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
  AddFavorite,
  ClearProgress,
  DeleteProgressAnime,
  DeleteProviderProgress,
  GetAnimeDetails,
  GetHomeSection,
  GetNextEpisode,
  GetProgress,
  GetRecentProgress,
  GetVideoSources,
  IsFavorite,
  ListFavorites,
  RemoveFavorite,
  ResumeAnime,
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

/**
 * Acciones de "retomar" (`models.ResumeAction*` de Go, T46): el valor es lo
 * que la tarjeta/retroceder decide al hacer clic. Como `as const`, el tipo
 * de `RESUME_ACTION.resume` es el literal `'resume'`.
 */
export const RESUME_ACTION = {
  resume: 'resume',
  next: 'next',
  completed: 'completed',
  up_to_date: 'up_to_date',
} as const;

/** Secciones del Home de un proveedor (caché 10 min en el backend; `noCache` salta la lectura). */
export function getHomeSection(provider: string, noCache = false): Promise<models.HomeSection[]> {
  return GetHomeSection(provider, noCache);
}

/** Añade un favorito (idempotente: sobre uno existente actualiza título/portada). */
export function addFavorite(f: models.Favorite): Promise<void> {
  return AddFavorite(f);
}

/** Quita un favorito; no falla si no existía. */
export function removeFavorite(provider: string, animeID: string): Promise<void> {
  return RemoveFavorite(provider, animeID);
}

/** Favoritos guardados, ordenados por `added_at` descendente. */
export function listFavorites(): Promise<models.Favorite[]> {
  return ListFavorites();
}

/** ¿Está (proveedor, anime) en favoritos? */
export function isFavorite(provider: string, animeID: string): Promise<boolean> {
  return IsFavorite(provider, animeID);
}

/** Último registro del historial por anime (lo que pinta la barra lateral). */
export function getRecentProgress(): Promise<models.WatchProgress[]> {
  return GetRecentProgress();
}

/** Borra todos los registros de un (proveedor, anime); devuelve los borrados. */
export function deleteProgressAnime(provider: string, animeID: string): Promise<number> {
  return DeleteProgressAnime(provider, animeID);
}

/** Borra todo el historial de un proveedor; devuelve los borrados. */
export function deleteProviderProgress(provider: string): Promise<number> {
  return DeleteProviderProgress(provider);
}

/** Borra el historial entero; devuelve los borrados. */
export function clearProgress(): Promise<number> {
  return ClearProgress();
}

/** Acción de "retomar" para un anime (tabla de acciones de v1.2_PLAN). */
export function resumeAnime(provider: string, animeID: string): Promise<models.ResumeInfo> {
  return ResumeAnime(provider, animeID);
}

/** Siguiente episodio (menor número estrictamente mayor); `found: false` si no hay. */
export function getNextEpisode(
  provider: string,
  animeID: string,
  episodeID: string
): Promise<models.NextEpisode> {
  return GetNextEpisode(provider, animeID, episodeID);
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
