/**
 * Lógica pura de pantalla completa (T91): mantiene la ventana nativa de Wails
 * sincronizada con la Fullscreen API de HTML.
 *
 * TS puro y sin dependencias: ni Svelte, ni `api.ts`, ni los bindings generados
 * de Wails. Las dependencias del mundo real entran por parámetro
 * (`DependenciasVentana`), así que el módulo se prueba sin navegador ni puente
 * Wails (regla 4 y P4).
 */

/** Lo mínimo que se necesita de un `Document` (permite dobles en tests). */
export interface DocumentoConPantallaCompleta {
  fullscreenElement?: Element | null;
}

/** Elemento en pantalla completa del documento, o null si no hay ninguno. */
export function elementoEnPantallaCompleta(
  doc: DocumentoConPantallaCompleta
): Element | null {
  if (!doc) return null;
  const elemento = doc.fullscreenElement;
  return elemento ?? null;
}

/** Operaciones de la ventana nativa, inyectadas (Wails en producción). */
export interface DependenciasVentana {
  esPantallaCompleta(): Promise<boolean>;
  entrar(): void;
  salir(): void;
}

/** Puente entre los eventos `fullscreenchange` del documento y la ventana. */
export interface Sincronizador {
  /** Llamar en cada `fullscreenchange` con `activo = hay elemento en fullscreen`. */
  alCambiarDocumento(activo: boolean): Promise<void>;
  /** Síncrono. Restaura la ventana si fue este sincronizador quien la puso en fullscreen. */
  liberar(): void;
}

/**
 * Crea el sincronizador de una vista.
 *
 * - `propia`: esta instancia puso la ventana en fullscreen (solo entonces la
 *   devuelve a ventana normal, para no cerrar un fullscreen ajeno).
 * - `liberado`: la vista se desmontó; a partir de ahí todo es inerte.
 * - Las llamadas se **serializan** en una cola, así el orden de los eventos
 *   `fullscreenchange` se respeta aunque `esPantallaCompleta()` tarde.
 */
export function crearSincronizador(dep: DependenciasVentana): Sincronizador {
  let propia = false;
  let liberado = false;
  let cola: Promise<void> = Promise.resolve();

  /** Consulta el estado nativo; cualquier fallo (o rechazo) cuenta como `false`. */
  function consultar(): Promise<boolean> {
    try {
      return Promise.resolve(dep.esPantallaCompleta()).then(
        (valor) => valor === true,
        () => false
      );
    } catch {
      return Promise.resolve(false);
    }
  }

  /** Cuerpo serializado: una entrada de la cola. Nunca lanza. */
  async function procesar(activo: boolean): Promise<void> {
    if (liberado) return;

    if (activo) {
      if (propia) return;
      const yaCompleta = await consultar();
      // Una operación en vuelo no debe entrar si mientras tanto se liberó.
      if (liberado || propia) return;
      if (yaCompleta) return;
      try {
        dep.entrar();
        propia = true;
      } catch {
        // Sin ventana nativa: se sigue sin marcar `propia` (la cola queda viva).
      }
      return;
    }

    if (!propia) return;
    propia = false;
    try {
      dep.salir();
    } catch {
      // `propia` ya quedó en false: el estado del módulo sigue siendo coherente.
    }
  }

  function alCambiarDocumento(activo: boolean): Promise<void> {
    const quiere = activo === true;
    const paso = cola.then(
      () => procesar(quiere),
      () => procesar(quiere)
    );
    // La cola nunca queda rechazada, pase lo que pase dentro de una entrada.
    cola = paso.catch(() => undefined);
    return cola;
  }

  function liberar(): void {
    liberado = true;
    if (!propia) return;
    propia = false;
    try {
      dep.salir();
    } catch {
      // Idempotente y sin lanzar: el desmontaje no debe romper la vista.
    }
  }

  return { alCambiarDocumento, liberar };
}
