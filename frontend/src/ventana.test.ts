/**
 * Tests de la lógica pura de ventana de episodios (T64): tamaño inicial,
 * tramos, cobertura de índice, retardos de entrada y memoria por sesión.
 *
 * El frontend no tiene runner de tests (fuera de `frontend/src/**` no se puede
 * tocar `package.json`), así que este archivo se ejecuta a mano con el
 * esbuild que ya trae Vite:
 *
 *   docker compose run --rm app sh -c \
 *     "cd frontend && ./node_modules/.bin/esbuild src/ventana.test.ts \
 *       --bundle --platform=node --format=cjs --outfile=/tmp/ventana.test.js \
 *       && node /tmp/ventana.test.js"
 */
import {
  COLUMNAS_POR_DEFECTO,
  cubrirIndice,
  claveVentana,
  normalizarColumnas,
  olvidarVisibles,
  recordarVisibles,
  recuperarVisibles,
  retardoEntrada,
  siguienteTamano,
  tamanoInicial,
} from './ventana';

let comprobaciones = 0;

function comprueba(nombre: string, obtenido: unknown, esperado: unknown): void {
  comprobaciones += 1;
  const a = JSON.stringify(obtenido);
  const b = JSON.stringify(esperado);
  if (a !== b) {
    throw new Error(`${nombre}: obtenido ${a}, esperado ${b}`);
  }
}

// 1. normalizarColumnas: no finito / < 1 / NaN → 4; no entero se trunca.
comprueba('normalizarColumnas 0', normalizarColumnas(0), COLUMNAS_POR_DEFECTO);
comprueba('normalizarColumnas -3', normalizarColumnas(-3), COLUMNAS_POR_DEFECTO);
comprueba('normalizarColumnas NaN', normalizarColumnas(NaN), COLUMNAS_POR_DEFECTO);
comprueba('normalizarColumnas Infinity', normalizarColumnas(Infinity), COLUMNAS_POR_DEFECTO);
comprueba('normalizarColumnas 4.9', normalizarColumnas(4.9), 4);
comprueba('normalizarColumnas 5', normalizarColumnas(5), 5);

// 2. tamanoInicial: min(total, 5 * columnas); total <= 0 o inválido => 0.
comprueba('tamanoInicial total 0', tamanoInicial(0, 4), 0);
comprueba('tamanoInicial total 7 con 4 cols (menos de 5 filas)', tamanoInicial(7, 4), 7);
comprueba('tamanoInicial 366 con 3 cols', tamanoInicial(366, 3), 15);
comprueba('tamanoInicial 366 con 4 cols', tamanoInicial(366, 4), 20);
comprueba('tamanoInicial 366 con 5 cols', tamanoInicial(366, 5), 25);

// 3. siguienteTamano: redondeo hacia arriba al múltiplo de columnas, tope total.
comprueba('siguienteTamano 20 -> 40 con 5 cols', siguienteTamano(20, 366, 5), 40);
comprueba('siguienteTamano 20 con 3 cols (32 -> 33)', siguienteTamano(20, 366, 3), 33);
comprueba('siguienteTamano 0 con 4 cols', siguienteTamano(0, 366, 4), 16);
comprueba('siguienteTamano 18 con 5 cols (38 -> 40)', siguienteTamano(18, 366, 5), 40);
comprueba('siguienteTamano tope en total 25', siguienteTamano(20, 25, 4), 25);
comprueba('siguienteTamano actual >= total', siguienteTamano(30, 20, 4), 20);
comprueba('siguienteTamano total 0', siguienteTamano(5, 0, 4), 0);

// 4. cubrirIndice: tamaño mínimo múltiplo de columnas que incluye el índice;
// nunca reduce por debajo de min(actual, total).
comprueba('cubrirIndice índice ya visible no reduce', cubrirIndice(20, 3, 366, 4), 20);
comprueba('cubrirIndice índice en el siguiente tramo', cubrirIndice(20, 25, 366, 4), 28);
comprueba('cubrirIndice índice límite de bloque', cubrirIndice(0, 3, 366, 4), 4);
comprueba('cubrirIndice índice no-múltiplo de columnas', cubrirIndice(0, 5, 366, 4), 8);
comprueba('cubrirIndice índice >= total', cubrirIndice(20, 366, 366, 4), 366);
comprueba('cubrirIndice índice negativo se trata como 0', cubrirIndice(0, -5, 10, 4), 4);
comprueba('cubrirIndice nunca reduce', cubrirIndice(30, 2, 366, 4), 30);

// 5. retardoEntrada: min((indice % columnas) * 50, 300).
comprueba('retardoEntrada columna 0', retardoEntrada(0, 4), 0);
comprueba('retardoEntrada columna 1', retardoEntrada(1, 4), 50);
comprueba('retardoEntrada índice 5 usa módulo (col 1)', retardoEntrada(5, 4), 50);
comprueba('retardoEntrada índice grande 13 con 4 cols', retardoEntrada(13, 4), 50);
comprueba('retardoEntrada índice 9 con 10 cols tope 300', retardoEntrada(9, 10), 300);
comprueba('retardoEntrada índice 6 con 10 cols = 300 exacto', retardoEntrada(6, 10), 300);
comprueba('retardoEntrada índice negativo', retardoEntrada(-3, 4), 0);
comprueba('retardoEntrada columnas 0 se normaliza a 4', retardoEntrada(2, 0), 100);

// 6. Memoria por sesión: recordar/recuperar/olvidar, sin lanzar.
comprueba('claveVentana', claveVentana('jkanime', 'bleach'), 'jkanime:bleach');
recordarVisibles('jkanime:bleach', 8);
comprueba('Map recupera lo recordado', recuperarVisibles('jkanime:bleach'), 8);
comprueba('Map clave inexistente', recuperarVisibles('nadie'), undefined);
recordarVisibles('jkanime:bleach', NaN);
recordarVisibles('jkanime:bleach', -1);
comprueba('Map ignora n no finito y negativo', recuperarVisibles('jkanime:bleach'), 8);
recordarVisibles('jkanime:bleach', 12);
comprueba('Map sobrescribe con n válido', recuperarVisibles('jkanime:bleach'), 12);
olvidarVisibles('jkanime:bleach');
comprueba('Map olvidar borra la clave', recuperarVisibles('jkanime:bleach'), undefined);

// 7. Propiedad para columnas 3, 4 y 5 y total 366: la secuencia desde el tamaño
// inicial es estrictamente creciente, termina exactamente en 366 y cada valor
// intermedio (todos menos el último) es múltiplo de columnas.
for (const columnas of [3, 4, 5]) {
  const serie: number[] = [tamanoInicial(366, columnas)];
  while (serie[serie.length - 1] < 366) {
    serie.push(siguienteTamano(serie[serie.length - 1], 366, columnas));
  }
  let creciente = true;
  for (let i = 1; i < serie.length; i += 1) {
    if (serie[i] <= serie[i - 1]) creciente = false;
  }
  const intermediosMultiplos = serie
    .slice(0, -1)
    .every((v) => v % columnas === 0);
  comprueba(`propiedad ${columnas} cols: estrictamente creciente`, creciente, true);
  comprueba(`propiedad ${columnas} cols: termina en 366`, serie[serie.length - 1], 366);
  comprueba(`propiedad ${columnas} cols: intermedios múltiplos`, intermediosMultiplos, true);
}

console.log(`OK (${comprobaciones} comprobaciones)`);
