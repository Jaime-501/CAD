/**
 * Favoritos (T54): store, carga y marcado/desmarcado.
 *
 * La lista la pinta `GridFavoritos.svelte` en el orden que decide el backend
 * (`List`, por `added_at` descendente): aquí no se reordena. Las operaciones
 * son **optimistas** —se aplica el cambio local, se llama al backend y, si
 * falla, se revierte la lista anterior y se propaga el error— para que la
 * estrella y el contador reaccionen sin esperar.
 *
 * Son idempotentes (el backend nunca hace *toggle*, solo `Add`/`Remove`): quien
 * decide el sentido del cambio es el botón, según `clavesFavoritos`.
 */
import { derived, get, writable, type Readable, type Writable } from 'svelte/store';
import { addFavorite, listFavorites, removeFavorite, type models } from './api';

export const favoritos: Writable<models.Favorite[]> = writable([]);
export const favoritosEstado: Writable<'idle' | 'loading' | 'ready' | 'error'> =
  writable('idle');

/** Claves `provider:anime_id` de la lista (mismo formato que `Favorite.Key()` de Go). */
export const clavesFavoritos: Readable<Set<string>> = derived(favoritos, (lista) =>
  new Set((lista ?? []).map((f) => `${f.provider}:${f.anime_id}`))
);

/**
 * Carga la lista desde el backend. Solo se pide una vez por sesión (igual que
 * el Home): si ya está cargada o en curso no se repite, porque las operaciones
 * de este módulo mantienen la lista al día. Un error deja el estado en `error`
 * para que la cuadrícula ofrezca "Reintentar".
 */
export async function cargarFavoritos(): Promise<void> {
  const estadoActual = get(favoritosEstado);
  if (estadoActual === 'loading' || estadoActual === 'ready') return;
  favoritosEstado.set('loading');
  try {
    const lista = await listFavorites();
    favoritos.set(lista ?? []);
    favoritosEstado.set('ready');
  } catch (e) {
    console.warn('[favoritos] no se pudo cargar:', String(e));
    favoritosEstado.set('error');
  }
}

/**
 * Marca como favorito con actualización optimista: se inserta AL PRINCIPIO (la
 * lista viene por recencia) y, si `addFavorite` falla, se restaura la lista
 * anterior y se lanza el error para que el botón lo muestre.
 */
export async function marcarFavorito(datos: {
  provider: string;
  anime_id: string;
  title: string;
  cover_url: string;
}): Promise<void> {
  const previo = get(favoritos);
  const clave = `${datos.provider}:${datos.anime_id}`;
  const yaEsta = previo.some((f) => `${f.provider}:${f.anime_id}` === clave);
  const nuevo: models.Favorite = { ...datos, added_at: 0 };
  if (!yaEsta) favoritos.set([nuevo, ...previo]);
  try {
    // `added_at: 0`: el backend lo sella con su propio reloj si viene en cero.
    await addFavorite(nuevo);
  } catch (e) {
    if (!yaEsta) favoritos.set(previo);
    throw e;
  }
}

/**
 * Quita un favorito con actualización optimista: se filtra de la lista local y,
 * si `removeFavorite` falla, se restaura la lista anterior y se lanza el error.
 */
export async function desmarcarFavorito(provider: string, animeID: string): Promise<void> {
  const previo = get(favoritos);
  const clave = `${provider}:${animeID}`;
  favoritos.set(previo.filter((f) => `${f.provider}:${f.anime_id}` !== clave));
  try {
    await removeFavorite(provider, animeID);
  } catch (e) {
    favoritos.set(previo);
    throw e;
  }
}
