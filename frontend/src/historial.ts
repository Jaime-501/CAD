/**
 * Historial de la barra lateral (T49): store, carga y helpers de formato.
 *
 * La lista la pinta `BarraLateral.svelte` agrupada por proveedor. El orden lo
 * decide el backend (`LastPerAnime`, ya viene por recencia): aquí solo se
 * agrupa y se formatea, sin reordenar dentro de cada grupo.
 */
import { writable, type Writable } from 'svelte/store';
import { getRecentProgress, type models } from './api';

export const historial: Writable<models.WatchProgress[]> = writable([]);
export const historialEstado: Writable<'idle' | 'loading' | 'ready' | 'error'> =
  writable('idle');

/**
 * Recarga la lista desde el backend. En error: estado `error` y la lista
 * anterior se conserva (la barra ofrece "Reintentar").
 */
export async function cargarHistorial(): Promise<void> {
  historialEstado.set('loading');
  try {
    const items = await getRecentProgress();
    historial.set(items ?? []);
    historialEstado.set('ready');
  } catch (e) {
    console.warn('[historial] no se pudo cargar:', String(e));
    historialEstado.set('error');
  }
}

/** Orden fijo de los grupos en la barra (solo los que traigan items). */
export const PROVEEDORES_ORDEN = ['jkanime', 'animeav1'] as const;

export const NOMBRE_PROVEEDOR: Record<string, string> = {
  jkanime: 'JKAnime',
  animeav1: 'AnimeAV1',
};

/** Proveedor de un registro: `provider` o el prefijo de `episode_id` (T41/plan). */
export function proveedorDe(p: models.WatchProgress): string {
  if (p.provider) return p.provider;
  const i = (p.episode_id ?? '').indexOf(':');
  return i > 0 ? p.episode_id.slice(0, i) : '';
}

export interface GrupoHistorial {
  provider: string;
  nombre: string;
  items: models.WatchProgress[];
}

/**
 * Agrupa por proveedor: primero `PROVEEDORES_ORDEN` (solo los que tengan
 * items) y después los desconocidos por nombre. Dentro de cada grupo se
 * CONSERVA el orden recibido (ya viene por recencia desde Go).
 */
export function agruparPorProveedor(items: models.WatchProgress[]): GrupoHistorial[] {
  const porProveedor = new Map<string, models.WatchProgress[]>();
  for (const item of items ?? []) {
    const clave = proveedorDe(item);
    const lista = porProveedor.get(clave);
    if (lista) lista.push(item);
    else porProveedor.set(clave, [item]);
  }

  const grupos: GrupoHistorial[] = [];
  for (const provider of PROVEEDORES_ORDEN) {
    const lista = porProveedor.get(provider);
    if (lista && lista.length > 0) {
      grupos.push({ provider, nombre: NOMBRE_PROVEEDOR[provider] ?? provider, items: lista });
      porProveedor.delete(provider);
    }
  }
  const desconocidos = [...porProveedor.keys()].sort((a, b) => {
    const na = NOMBRE_PROVEEDOR[a] ?? a;
    const nb = NOMBRE_PROVEEDOR[b] ?? b;
    return na.localeCompare(nb) || a.localeCompare(b);
  });
  for (const provider of desconocidos) {
    grupos.push({
      provider,
      nombre: NOMBRE_PROVEEDOR[provider] ?? provider,
      items: porProveedor.get(provider) ?? [],
    });
  }
  return grupos;
}

/** Segundos → `m:ss` (<1 h) o `h:mm:ss`; negativos/NaN → `0:00`. */
export function formatearTiempo(seg: number): string {
  if (!Number.isFinite(seg) || seg < 0) return '0:00';
  const s = Math.floor(seg);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = String(s % 60).padStart(2, '0');
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${ss}`;
  return `${m}:${ss}`;
}

const RE_EPISODIO = /:(?:episode|special):(\d+)$/;

/**
 * Etiqueta del episodio: `Ep N` (de `episode_number` si >0; si no, del id),
 * `Película` si el id termina en `:movie`; '' si nada.
 */
export function numeroDeEpisodio(p: models.WatchProgress): string {
  if (p.episode_number > 0) return `Ep ${p.episode_number}`;
  const coincidencia = RE_EPISODIO.exec(p.episode_id ?? '');
  if (coincidencia) return `Ep ${coincidencia[1]}`;
  if ((p.episode_id ?? '').endsWith(':movie')) return 'Película';
  return '';
}

/** Título visible: `anime_title` o el slug del id con espacios y mayúscula inicial. */
export function tituloVisible(p: models.WatchProgress): string {
  const titulo = (p.anime_title ?? '').trim();
  if (titulo !== '') return titulo;
  const base = (p.anime_id ?? '').replace(/-/g, ' ');
  return base.charAt(0).toUpperCase() + base.slice(1);
}

/** Fracción 0..1 del progreso: visto = 1; sin duración utilizable = 0. */
export function fraccionProgreso(p: models.WatchProgress): number {
  if (p.is_watched) return 1;
  const total = p.total_duration;
  if (!Number.isFinite(total) || total <= 0) return 0;
  const fraccion = p.current_time / total;
  if (!Number.isFinite(fraccion)) return 0;
  return Math.min(1, Math.max(0, fraccion));
}
