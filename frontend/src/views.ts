/**
 * Estado de navegación entre vistas (T25): sin librería de rutas.
 *
 * Tres vistas encadenadas: búsqueda → detalle → reproductor. El historial vive
 * en una pila simple; el botón "atrás" desapila y vuelve a la vista anterior.
 * Los resultados de la búsqueda no se guardan aquí: siguen en `stores.ts`
 * (`searchResults`/`searchState`), así que al volver siguen intactos. La
 * posición de scroll se guarda/recupera por clave (las llama `App.svelte` al
 * cambiar de vista).
 *
 * T26: al navegar al reproductor la selección lleva además el episodio
 * (`episodeID`, número y título); T27 lee esa selección para cargar las fuentes.
 */
import { get, writable } from 'svelte/store';

export type VistaNombre = 'busqueda' | 'detalle' | 'reproductor';

/** Anime elegido en la búsqueda: lo consumen la vista de detalle y la de reproductor. */
export interface SeleccionAnime {
  provider: string;
  animeID: string;
  title: string;
  /** Rellenos al navegar al reproductor (T26); los consume T27. */
  episodeID?: string;
  episodeNumber?: number;
  episodeTitle?: string;
}

export const vista = writable<VistaNombre>('busqueda');
export const seleccion = writable<SeleccionAnime | null>(null);

/** Historial para "atrás" (solo se apilan las vistas distintas de la actual). */
let historial: VistaNombre[] = [];

/**
 * Scroll guardado por clave (T26: por vista + anime/episodio, para que al
 * volver de otra pantalla no se restaure el scroll de otro anime).
 * Claves: `busqueda` · `detalle:<animeID>` · `reproductor:<episodeID>`.
 */
const scrollPorClave: Record<string, number> = {};

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

export function guardarScroll(clave: string, y: number): void {
  scrollPorClave[clave] = y;
}

export function recuperarScroll(clave: string): number {
  return scrollPorClave[clave] ?? 0;
}

/**
 * Clave de scroll de una vista: la búsqueda es única; el detalle se recuerda
 * por anime y el reproductor por episodio (T26).
 */
export function claveScroll(v: VistaNombre): string {
  const sel = get(seleccion);
  if (v === 'detalle') return `detalle:${sel?.animeID ?? ''}`;
  if (v === 'reproductor') return `reproductor:${sel?.episodeID ?? ''}`;
  return v;
}
