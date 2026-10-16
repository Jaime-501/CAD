/**
 * Revelado animado de tarjetas al entrar en el viewport (T65).
 *
 * Acción Svelte `use:revelar={{ id, indice, columnas }}` para el envoltorio
 * `.celda` de cada tarjeta de episodio. La memoria `reveladas` vive a NIVEL DE
 * MÓDULO (sobrevive al remontaje de VistaDetalle, mismo patrón que
 * `scrollPorClave` de `views.ts`): al volver del reproductor ninguna tarjeta ya
 * revelada se vuelve a animar.
 *
 * - Un único IntersectionObserver compartido (threshold 0.15) revela cada nodo
 *   al intersectar: añade la clase `entra`, fija el `animation-delay` con
 *   `retardoEntrada` (de `ventana.ts`) y hace `unobserve`.
 * - El estado inicial oculto (`pendiente`) se pone de forma síncrona en la
 *   acción, antes del primer pintado, para no parpadear visible→oculta.
 * - Sin IntersectionObserver, con `prefers-reduced-motion: reduce` o si el id ya
 *   estaba revelado, no se oculta ni se anima: la tarjeta se ve normal.
 */

import { COLUMNAS_POR_DEFECTO, retardoEntrada } from './ventana';

export interface ParametrosRevelar {
  /** Clave de la tarjeta (episodio.id, ya lleva prefijo de proveedor). */
  id: string;
  /** Índice del episodio en la lista ordenada. */
  indice: number;
  /** Columnas de la rejilla en el momento de revelar (para el retardo por fila). */
  columnas: number;
}

interface Estado {
  params: ParametrosRevelar;
  observando: boolean;
}

/** Ids ya revelados esta sesión: no se vuelven a animar al remontar la vista. */
const reveladas = new Set<string>();

const estados = new WeakMap<HTMLElement, Estado>();

let observador: IntersectionObserver | null = null;

function prefiereMovimientoReducido(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  );
}

function obtenerObservador(): IntersectionObserver | null {
  if (observador !== null) return observador;
  if (typeof IntersectionObserver === 'undefined') return null;
  observador = new IntersectionObserver(
    (entradas) => {
      for (const entrada of entradas) {
        if (!entrada.isIntersecting) continue;
        const el = entrada.target as HTMLElement;
        const estado = estados.get(el);
        if (!estado) continue;
        revelarElemento(el, estado.params);
        observador?.unobserve(el);
        estado.observando = false;
      }
    },
    { threshold: 0.15 }
  );
  return observador;
}

function revelarElemento(el: HTMLElement, params: ParametrosRevelar): void {
  reveladas.add(params.id);
  el.classList.remove('pendiente');
  el.classList.add('entra');
  el.style.animationDelay = `${retardoEntrada(params.indice, params.columnas)}ms`;
}

/** Sólo para pruebas: olvida qué episodios se revelaron. */
export function reiniciarReveladas(): void {
  reveladas.clear();
}

/** Cuenta las pistas de `grid-template-columns`; si no puede, valor por defecto. */
export function medirColumnas(rejilla: HTMLElement): number {
  try {
    const valor = getComputedStyle(rejilla).gridTemplateColumns;
    if (!valor || valor === 'none') return COLUMNAS_POR_DEFECTO;
    const pistas = valor.split(' ').filter((t) => t.trim() !== '');
    return pistas.length > 0 ? pistas.length : COLUMNAS_POR_DEFECTO;
  } catch {
    return COLUMNAS_POR_DEFECTO;
  }
}

export function revelar(nodo: HTMLElement, params: ParametrosRevelar) {
  const estado: Estado = { params, observando: false };
  estados.set(nodo, estado);

  const puedeAnimar =
    !reveladas.has(params.id) && !prefiereMovimientoReducido() && obtenerObservador() !== null;

  if (puedeAnimar) {
    nodo.classList.add('pendiente');
    estado.observando = true;
    obtenerObservador()?.observe(nodo);
  } else {
    nodo.classList.remove('pendiente');
  }

  return {
    update(nuevos: ParametrosRevelar) {
      estado.params = nuevos;
    },
    destroy() {
      if (estado.observando) observador?.unobserve(nodo);
      estados.delete(nodo);
    },
  };
}
