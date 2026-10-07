/**
 * Estado de navegación entre vistas (T25): sin librería de rutas.
 *
 * Tres vistas encadenadas: búsqueda → detalle → reproductor. El historial vive
 * en una pila simple; el botón "atrás" desapila y vuelve a la vista anterior.
 * Los resultados de la búsqueda no se guardan aquí: siguen en `stores.ts`
 * (`searchResults`/`searchState`), así que al volver siguen intactos. La
 * posición de scroll de la búsqueda se guarda/recupera con las funciones de
 * abajo (las llama `App.svelte` al cambiar de vista).
 */
import { get, writable } from 'svelte/store';

export type VistaNombre = 'busqueda' | 'detalle' | 'reproductor';

/** Anime elegido en la búsqueda: lo consumen la vista de detalle y la de reproductor. */
export interface SeleccionAnime {
  provider: string;
  animeID: string;
  title: string;
}

export const vista = writable<VistaNombre>('busqueda');
export const seleccion = writable<SeleccionAnime | null>(null);

/** Historial para "atrás" (solo se apilan las vistas distintas de la actual). */
let historial: VistaNombre[] = [];
/** Posición de scroll de la vista de búsqueda, guardada al salir de ella. */
let scrollBusqueda = 0;

/**
 * Navega a otra vista. Con `sel` fija el anime seleccionado (detalle/reproductor);
 * sin `sel` se conserva el anterior.
 */
export function navegarA(destino: VistaNombre, sel?: SeleccionAnime): void {
  const actual = get(vista);
  if (actual !== destino) historial.push(actual);
  if (sel) seleccion.set(sel);
  vista.set(destino);
}

/** Botón "atrás": vuelve a la vista anterior; sin historial, a la búsqueda. */
export function atras(): void {
  vista.set(historial.pop() ?? 'busqueda');
}

export function guardarScrollBusqueda(y: number): void {
  scrollBusqueda = y;
}

export function recuperarScrollBusqueda(): number {
  return scrollBusqueda;
}
