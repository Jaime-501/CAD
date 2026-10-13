/**
 * Toast de una sola ranura (T51): aviso breve abajo-centro con acción
 * opcional ("Deshacer"). `mostrarToast` reemplaza el anterior y reinicia el
 * temporizador; el valor del store lleva un `id` para que el cierre automático
 * solo afecte al aviso que lo programó.
 */
import { writable, type Writable } from 'svelte/store';

/** Acción del toast: botón con su etiqueta y manejador. */
export interface AccionToast {
  etiqueta: string;
  fn: () => void;
}

export interface AvisoToast {
  id: number;
  mensaje: string;
  accion?: AccionToast;
}

export const toast: Writable<AvisoToast | null> = writable(null);

/** Duración por defecto del aviso (ms). */
const DURACION_MS = 5000;

let siguienteId = 1;
let temporizador: ReturnType<typeof setTimeout> | null = null;

/**
 * Muestra un aviso nuevo (reemplaza el anterior y reinicia el temporizador).
 * `duracionMs` por defecto 5000; sin `accion` no se pinta botón de acción.
 */
export function mostrarToast(
  mensaje: string,
  opciones?: { accion?: AccionToast; duracionMs?: number }
): void {
  const duracion = opciones?.duracionMs ?? DURACION_MS;
  if (temporizador !== null) clearTimeout(temporizador);
  const id = siguienteId++;
  toast.set({ id, mensaje, accion: opciones?.accion });
  temporizador = setTimeout(() => {
    temporizador = null;
    // Solo se cierra el aviso que programó este temporizador.
    toast.update((actual) => (actual !== null && actual.id === id ? null : actual));
  }, duracion);
}

/** Cierra el aviso actual y limpia su temporizador. */
export function cerrarToast(): void {
  if (temporizador !== null) clearTimeout(temporizador);
  temporizador = null;
  toast.set(null);
}
