/**
 * Acción Svelte de deslizar para borrar (T51): `use:deslizar={{ onBorrar, umbral: 96 }}`.
 *
 * Con *pointer events* (ratón y táctil por igual):
 * - `pointerdown` (solo botón principal) prepara el gesto y el nodo gana
 *   `touch-action: pan-y`. El puntero se captura **cuando el gesto se decide
 *   como horizontal** (ver Desviaciones de T51 en PROGRESO.md): capturarlo ya
 *   en `pointerdown` hace que Chromium retargete el `click` posterior al nodo
 *   envoltorio y entonces ni el ✕ ni el clic de retomar llegan a ejecutarse.
 * - `pointermove`: si el movimiento vertical supera 10 px **antes** que el
 *   horizontal, el gesto se cancela (deja scrollear); si no, `translateX(dx)`
 *   sin transición y opacidad de 1 a 0.4 según `|dx|/ancho`.
 * - `pointerup`: con `|dx| ≥ umbral` anima fuera de pantalla (150 ms) y llama
 *   a `onBorrar()`; si no, vuelve a 0 con transición de 150 ms.
 * - Si hubo desplazamiento > 6 px, el `click` inmediato posterior se suprime
 *   (no debe disparar "retomar").
 * - `pointercancel` restaura. Con `prefers-reduced-motion` las transiciones
 *   duran 0 ms. Los listeners se limpian en `destroy`.
 */

export interface ParamsDeslizar {
  /** Se llama al soltar con `|dx| ≥ umbral` (tras la animación de salida). */
  onBorrar: () => void;
  /** Distancia mínima en px; por defecto 96. */
  umbral?: number;
}

const UMBRAL_PX = 96;
/** Movimiento vertical u horizontal (px) que decide el eje del gesto. */
const EJE_PX = 10;
/** Desplazamiento (px) a partir del cual se suprime el click posterior. */
const SUPRIMIR_CLICK_PX = 6;
const MS_ANIMACION = 150;

export function deslizar(nodo: HTMLElement, params: ParamsDeslizar) {
  let paramsActuales = params;

  let pointerId: number | null = null;
  let capturado = false;
  let x0 = 0;
  let y0 = 0;
  let dx = 0;
  let arrastrando = false;
  let ejeDecidido = false;
  let horizontal = false;
  let cancelado = false;
  let desplazado = false;
  let temporizadorReset: ReturnType<typeof setTimeout> | null = null;
  let supresorClick: ((e: Event) => void) | null = null;
  let temporizadorSupresion: ReturnType<typeof setTimeout> | null = null;

  function duracion(ms: number): number {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : ms;
  }

  function opacidadPara(x: number): number {
    const ancho = nodo.offsetWidth || 1;
    const razon = Math.min(1, Math.abs(x) / ancho);
    return Math.max(0.4, 1 - 0.6 * razon);
  }

  function aplicar(x: number, opacidad: number, ms: number): void {
    nodo.style.transition = ms > 0 ? `transform ${ms}ms ease, opacity ${ms}ms ease` : 'none';
    nodo.style.transform = `translateX(${x}px)`;
    nodo.style.opacity = String(opacidad);
  }

  function restaurar(ms: number): void {
    aplicar(0, 1, ms);
  }

  function limpiarEstilos(): void {
    nodo.style.transition = 'none';
    nodo.style.transform = '';
    nodo.style.opacity = '';
    nodo.style.touchAction = '';
  }

  function soltarPuntero(): void {
    arrastrando = false;
    nodo.style.touchAction = '';
    if (capturado && pointerId !== null) {
      try {
        nodo.releasePointerCapture(pointerId);
      } catch {
        /* el navegador ya lo había soltado */
      }
    }
    capturado = false;
    pointerId = null;
  }

  function suprimirClick(): void {
    if (supresorClick !== null) return;
    const manejador = (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      quitarSupresion();
    };
    supresorClick = manejador;
    // Captura en el nodo envoltorio: el click del botón de retomar (interior)
    // no llega a dispararse.
    nodo.addEventListener('click', manejador, true);
    temporizadorSupresion = setTimeout(quitarSupresion, 400);
  }

  function quitarSupresion(): void {
    if (supresorClick !== null) {
      nodo.removeEventListener('click', supresorClick, true);
      supresorClick = null;
    }
    if (temporizadorSupresion !== null) {
      clearTimeout(temporizadorSupresion);
      temporizadorSupresion = null;
    }
  }

  function programarReset(): void {
    if (temporizadorReset !== null) clearTimeout(temporizadorReset);
    // Si el nodo sobrevive (p. ej. el borrado falló y la tarjeta sigue en la
    // lista), vuelve a su sitio sin animación; si se desmontó, no se ve nada.
    temporizadorReset = setTimeout(() => {
      temporizadorReset = null;
      limpiarEstilos();
    }, MS_ANIMACION + 20);
  }

  function alPulsar(e: PointerEvent): void {
    if (e.button !== 0 || !e.isPrimary || arrastrando) return;
    arrastrando = true;
    capturado = false;
    ejeDecidido = false;
    horizontal = false;
    cancelado = false;
    desplazado = false;
    dx = 0;
    x0 = e.clientX;
    y0 = e.clientY;
    pointerId = e.pointerId;
    // Sin setPointerCapture aquí: en Chromium retargetaría el `click` del
    // botón interior al envoltorio y rompería retomar/✕ (ver cabecera).
    nodo.style.touchAction = 'pan-y';
  }

  function alMover(e: PointerEvent): void {
    if (!arrastrando || cancelado || e.pointerId !== pointerId) return;
    dx = e.clientX - x0;
    const dy = e.clientY - y0;
    if (!ejeDecidido) {
      if (Math.abs(dy) > EJE_PX && Math.abs(dx) <= EJE_PX) {
        // El gesto es vertical: el navegador hace scroll y el nodo no se mueve.
        cancelado = true;
        soltarPuntero();
        limpiarEstilos();
        return;
      }
      if (Math.abs(dx) <= EJE_PX) return;
      ejeDecidido = true;
      horizontal = true;
      // Gesto horizontal decidido: desde aquí el puntero se captura para que
      // el arrastre siga aunque el puntero salga del nodo.
      try {
        nodo.setPointerCapture(e.pointerId);
        capturado = true;
      } catch {
        /* captura no disponible: el seguimiento sigue en el nodo */
      }
    }
    if (!horizontal) return;
    if (Math.abs(dx) > SUPRIMIR_CLICK_PX) desplazado = true;
    aplicar(dx, opacidadPara(dx), 0);
  }

  function alSoltar(e: PointerEvent): void {
    if (!arrastrando || e.pointerId !== pointerId) return;
    const umbral = paramsActuales.umbral ?? UMBRAL_PX;
    const onBorrar = paramsActuales.onBorrar;
    soltarPuntero();
    if (cancelado) return;
    if (desplazado) suprimirClick();
    if (horizontal && Math.abs(dx) >= umbral) {
      const fuera = Math.sign(dx) * (nodo.offsetWidth + 24);
      aplicar(fuera, 0, duracion(MS_ANIMACION));
      programarReset();
      onBorrar();
      return;
    }
    if (horizontal) restaurar(duracion(MS_ANIMACION));
  }

  function alCancelar(e: PointerEvent): void {
    if (!arrastrando || e.pointerId !== pointerId) return;
    soltarPuntero();
    if (cancelado) return;
    limpiarEstilos();
  }

  nodo.addEventListener('pointerdown', alPulsar);
  nodo.addEventListener('pointermove', alMover);
  nodo.addEventListener('pointerup', alSoltar);
  nodo.addEventListener('pointercancel', alCancelar);

  return {
    update(nuevos: ParamsDeslizar) {
      paramsActuales = nuevos;
    },
    destroy() {
      nodo.removeEventListener('pointerdown', alPulsar);
      nodo.removeEventListener('pointermove', alMover);
      nodo.removeEventListener('pointerup', alSoltar);
      nodo.removeEventListener('pointercancel', alCancelar);
      quitarSupresion();
      if (temporizadorReset !== null) clearTimeout(temporizadorReset);
      limpiarEstilos();
    },
  };
}
