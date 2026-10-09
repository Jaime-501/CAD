/**
 * Estado de la aplicación (Svelte stores) + suscripción a los eventos de Go.
 *
 * T24: sin UI todavía. Los componentes (T25/T26) llaman a `initEvents()` en su
 * arranque (onMount) y a `disposeEvents()` al desmontar, para no dejar listeners.
 *
 * Clave de la tarea: los lotes `episodes:update` llegan ANTES de que resuelva
 * `GetAnimeDetails`, así que `currentAnimeID` se fija antes de esa llamada y
 * todo lo que no coincida con él se descarta.
 *
 * T34 (aislamiento entre proveedores): `anime_id` es el slug, y un slug como
 * "bleach" existe en JKAnime y en AnimeAV1. Además de `currentAnimeID` se
 * fija `currentProvider` y se filtra por el prefijo de proveedor del
 * `episode_id`, para que el progreso y los episodios de un proveedor nunca
 * aparezcan ni influzan en la lista del otro. El historial NO se migra: la
 * separación es una vista sobre los mismos registros.
 */
import { get, writable } from 'svelte/store';
import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime.js';
import type { models } from '../wailsjs/go/models';
import { getAnimeDetails, getProgress, searchAnime } from './api';
import type { EpisodesCompletePayload, EpisodesUpdatePayload } from './api';

export interface SearchState {
  loading: boolean;
  error: string;
  query: string;
}

export const currentAnimeID = writable<string>('');
/** Proveedor de la lista abierta ahora (jkanime, animeav1…). */
export const currentProvider = writable<string>('');
export const currentDetail = writable<models.AnimeDetail | null>(null);
export const episodes = writable<models.Episode[]>([]);
export const episodesComplete = writable<boolean>(false);
export const episodesError = writable<string>('');
export const progressMap = writable<Record<string, models.WatchProgress>>({});
export const searchResults = writable<models.Anime[]>([]);
export const searchState = writable<SearchState>({ loading: false, error: '', query: '' });

/** Mezcla dos listas de episodios deduplicando por `Episode.id` (gana el primero; orden de llegada). */
export function mergeEpisodes(
  actuales: models.Episode[],
  entrantes: models.Episode[]
): models.Episode[] {
  const porID = new Map<string, models.Episode>();
  for (const ep of actuales) {
    if (!porID.has(ep.id)) porID.set(ep.id, ep);
  }
  for (const ep of entrantes) {
    if (!porID.has(ep.id)) porID.set(ep.id, ep);
  }
  return Array.from(porID.values());
}

/** ¿Pertenece este evento al anime que se está viendo ahora? (nunca con el ID vacío). */
export function isCurrentBatch(currentID: string, payload: { anime_id: string }): boolean {
  return currentID !== '' && payload.anime_id === currentID;
}

/**
 * ¿Pertenece este `episode_id` al proveedor indicado? El id siempre es
 * `<proveedor>:<slug>…` (ver `models.ParseEpisodeID`), y un proveedor nunca
 * lleva ':' dentro, así que el prefijo identifica sin ambigüedad.
 */
export function esDeProveedor(episodeID: string, proveedor: string): boolean {
  return proveedor !== '' && episodeID.startsWith(proveedor + ':');
}

/** Filtra episodios por el prefijo de proveedor de su id (T34). */
export function episodiosDeProveedor(
  episodios: models.Episode[],
  proveedor: string
): models.Episode[] {
  return episodios.filter((ep) => esDeProveedor(ep.id, proveedor));
}

/**
 * Historial → mapa indexado por `episode_id`, SOLO con los registros del
 * proveedor indicado: el historial se guarda por slug y un mismo slug puede
 * tener progreso de dos proveedores (T34).
 */
export function progressToMap(
  historial: models.WatchProgress[],
  proveedor: string
): Record<string, models.WatchProgress> {
  const mapa: Record<string, models.WatchProgress> = {};
  for (const p of historial) {
    if (esDeProveedor(p.episode_id, proveedor)) mapa[p.episode_id] = p;
  }
  return mapa;
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

function hayPuenteWails(): boolean {
  return typeof window !== 'undefined' && 'runtime' in window;
}

let listenersActive = false;

/** Registra los listeners de eventos una sola vez (llamar al montar). */
export function initEvents(): void {
  if (listenersActive || !hayPuenteWails()) return;

  EventsOn('episodes:update', (crudo: unknown) => {
    const payload = parsePayload<EpisodesUpdatePayload>(crudo);
    if (!payload) return;
    // Lote de otro anime (o llegado tarde): se descarta. El slug por sí solo
    // no basta (T34): existe en varios proveedores, así que también se
    // descartan los episodios cuyo id no lleva el prefijo del proveedor abierto.
    if (!isCurrentBatch(get(currentAnimeID), payload)) return;
    const propios = episodiosDeProveedor(payload.episodes ?? [], get(currentProvider));
    if (propios.length === 0) return;
    episodes.update((lista) => mergeEpisodes(lista, propios));
  });

  EventsOn('episodes:complete', (crudo: unknown) => {
    const payload = parsePayload<EpisodesCompletePayload>(crudo);
    if (!payload) return;
    // `episodes:complete` solo lleva `anime_id` (modelo de Go, no tocable) y
    // Go emite un único stream activo, cancelando los anteriores: con el
    // slug no hay atribución por proveedor, pero tampoco stream concurrente.
    if (!isCurrentBatch(get(currentAnimeID), payload)) return;
    episodesComplete.set(true);
    episodesError.set(payload.error ?? '');
  });

  listenersActive = true;
}

/** Quita los listeners (llamar al desmontar; sin fugas). */
export function disposeEvents(): void {
  if (!listenersActive) return;
  EventsOff('episodes:update', 'episodes:complete');
  listenersActive = false;
}

/**
 * Abre un anime. Orden importante:
 *  1. fija `currentAnimeID` y `currentProvider` ANTES de la llamada (los lotes
 *     empiezan a llegar enseguida y hay que poder filtrarlos por proveedor);
 *  2. limpia el estado del anime anterior (episodes/complete/error/progreso);
 *  3. mezcla los episodios que traiga la respuesta con los ya llegados (misma
 *     deduplicación, y solo los del proveedor abierto);
 *  4. puebla `progressMap` con el historial de ESE proveedor (el slug puede
 *     tener progreso de otro).
 * Si el usuario cambia de anime mientras tanto, la respuesta vieja se descarta.
 */
export async function openAnime(provider: string, animeID: string): Promise<void> {
  currentAnimeID.set(animeID);
  currentProvider.set(provider);
  currentDetail.set(null);
  episodes.set([]);
  episodesComplete.set(false);
  episodesError.set('');
  progressMap.set({});

  const res = await getAnimeDetails(provider, animeID);
  // El slug por sí solo no basta (T34): "bleach" existe en dos proveedores,
  // así que la respuesta solo se acepta si sigue abierta ESA pareja.
  if (get(currentAnimeID) !== animeID || get(currentProvider) !== provider) return;
  currentDetail.set(res.detail ?? null);
  episodes.update((lista) =>
    mergeEpisodes(lista, episodiosDeProveedor(res.episodes ?? [], provider))
  );

  try {
    const historial = await getProgress(animeID);
    if (get(currentAnimeID) !== animeID || get(currentProvider) !== provider) return;
    progressMap.set(progressToMap(historial, provider));
  } catch (e) {
    // Historial no disponible (T22): no es un fallo grave, queda vacío.
    console.warn('[stores] historial no disponible:', String(e));
  }
}

/**
 * Búsqueda → `searchResults`/`searchState`. Las respuestas obsoletas se
 * descartan por contador de secuencia (T25: cambio mínimo autorizado) y, además,
 * si la query ya no es la vigente.
 */
let secuenciaBusqueda = 0;

export async function search(query: string): Promise<void> {
  const seq = ++secuenciaBusqueda;
  const q = query.trim();
  if (!q) {
    searchResults.set([]);
    searchState.set({ loading: false, error: '', query: '' });
    return;
  }

  searchResults.set([]);
  searchState.set({ loading: true, error: '', query: q });
  try {
    const resultados = await searchAnime(q);
    if (seq !== secuenciaBusqueda || get(searchState).query !== q) return;
    searchResults.set(resultados);
    searchState.set({ loading: false, error: '', query: q });
  } catch (e) {
    if (seq !== secuenciaBusqueda || get(searchState).query !== q) return;
    searchResults.set([]);
    searchState.set({ loading: false, error: String(e), query: q });
  }
}
