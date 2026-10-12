/**
 * Home "Descubrir" (T53): estado por proveedor y carga de las secciones.
 *
 * Cada proveedor se carga y falla de forma **independiente**: el store `home`
 * guarda un `EstadoProveedor` por id, y un fallo no toca al resto. La carga
 * ocurre **una vez por sesión** (la decide `VistaInicio` al montar: si el
 * proveedor ya tiene estado no se vuelve a pedir); el botón "Actualizar"
 * relanza todo con `noCache = true`.
 */
import { writable, type Writable } from 'svelte/store';
import { getHomeSection, type models } from './api';

/** Filas del Home, en el orden en que se pintan (fijado por el plan v1.2). */
export const PROVEEDORES_HOME = [
  { id: 'jkanime', nombre: 'JKAnime' },
  { id: 'animeav1', nombre: 'AnimeAV1' },
] as const;

/** Estado visible de un proveedor: cargando, cargado (con sus secciones) o en error. */
export type EstadoProveedor = {
  estado: 'loading' | 'ok' | 'error';
  secciones: models.HomeSection[];
  error: string;
};

/** Vacío al arrancar: cada proveedor entra a `'loading'` cuando se pide. */
export const home: Writable<Record<string, EstadoProveedor>> = writable({});

/** Proveedores con una petición en curso (evita dobles cargas/reintentos simultáneos). */
const enCurso = new Set<string>();

/**
 * Carga (o recarga) las secciones de un proveedor. Pasa a `'loading'`, luego a
 * `'ok'` con las secciones o a `'error'` con el mensaje del fallo; los errores
 * no se guardan como secciones y el estado de los demás proveedores no cambia.
 */
export async function cargarProveedor(id: string, noCache = false): Promise<void> {
  if (enCurso.has(id)) return;
  enCurso.add(id);
  home.update((h) => ({ ...h, [id]: { estado: 'loading', secciones: [], error: '' } }));
  try {
    const secciones = (await getHomeSection(id, noCache)) ?? [];
    home.update((h) => ({ ...h, [id]: { estado: 'ok', secciones, error: '' } }));
  } catch (e) {
    console.warn('[home] no se pudo cargar', id, String(e));
    home.update((h) => ({ ...h, [id]: { estado: 'error', secciones: [], error: String(e) } }));
  } finally {
    enCurso.delete(id);
  }
}

/**
 * Lanza la carga de **todos** los proveedores en paralelo (sin `await`
 * encadenado): cada uno resuelve por su cuenta.
 */
export function cargarTodo(noCache = false): void {
  for (const proveedor of PROVEEDORES_HOME) {
    void cargarProveedor(proveedor.id, noCache);
  }
}
