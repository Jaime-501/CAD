export const FILAS_INICIAL = 5;
export const FILAS_TRAMO = 4;
export const COLUMNAS_POR_DEFECTO = 4;
export const RETARDO_POR_COLUMNA_MS = 50;
export const RETARDO_MAXIMO_MS = 300;

export function normalizarColumnas(n: number): number {
  if (!Number.isFinite(n) || n < 1) return COLUMNAS_POR_DEFECTO;
  return Math.floor(n);
}

export function tamanoInicial(total: number, columnas: number): number {
  if (!Number.isFinite(total) || total <= 0) return 0;
  return Math.min(total, FILAS_INICIAL * normalizarColumnas(columnas));
}

export function siguienteTamano(actual: number, total: number, columnas: number): number {
  if (!Number.isFinite(total) || total <= 0) return 0;
  const base = Number.isFinite(actual) ? actual : 0;
  if (base >= total) return total;
  const cols = normalizarColumnas(columnas);
  const objetivo = base + FILAS_TRAMO * cols;
  const redondeado = Math.ceil(objetivo / cols) * cols;
  const limitado = Math.min(redondeado, total);
  return Math.max(limitado, Math.min(base, total));
}

export function cubrirIndice(
  actual: number,
  indice: number,
  total: number,
  columnas: number
): number {
  if (!Number.isFinite(total) || total <= 0) return 0;
  const base = Number.isFinite(actual) && actual >= 0 ? actual : 0;
  let idx = Number.isFinite(indice) ? indice : 0;
  if (idx < 0) idx = 0;
  if (idx >= total) return total;
  const cols = normalizarColumnas(columnas);
  const necesario = Math.ceil((idx + 1) / cols) * cols;
  const limitado = Math.min(necesario, total);
  return Math.max(limitado, Math.min(base, total));
}

export function retardoEntrada(indice: number, columnas: number): number {
  const cols = normalizarColumnas(columnas);
  let idx = Number.isFinite(indice) ? indice : 0;
  if (idx < 0) idx = 0;
  const columna = idx % cols;
  return Math.min(columna * RETARDO_POR_COLUMNA_MS, RETARDO_MAXIMO_MS);
}

export function claveVentana(provider: string, animeID: string): string {
  return `${provider}:${animeID}`;
}

const visiblesPorClave = new Map<string, number>();

export function recordarVisibles(clave: string, n: number): void {
  if (!Number.isFinite(n) || n < 0) return;
  visiblesPorClave.set(clave, n);
}

export function recuperarVisibles(clave: string): number | undefined {
  return visiblesPorClave.get(clave);
}

export function olvidarVisibles(clave: string): void {
  visiblesPorClave.delete(clave);
}
