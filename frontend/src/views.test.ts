/**
 * Tests de la clave de scroll de las vistas (T66): la clave debe distinguir el
 * proveedor, porque «bleach» existe en JKAnime y AnimeAV1 y antes compartían
 * posición de scroll.
 *
 * El frontend no tiene runner de tests (fuera de `frontend/src/**` no se puede
 * tocar `package.json`), así que este archivo se ejecuta a mano con el
 * esbuild que ya trae Vite:
 *
 *   docker compose run --rm app sh -c \
 *     "cd frontend && ./node_modules/.bin/esbuild src/views.test.ts \
 *       --bundle --platform=node --format=cjs --outfile=/tmp/views.test.js \
 *       && node /tmp/views.test.js"
 *
 * `views.ts` importa `svelte/store`; el bundle lo resuelve desde
 * `frontend/node_modules` (si no, el propio esbuild lo reporta al ejecutar).
 */
import { claveScroll, guardarScroll, recuperarScroll, seleccion } from './views';
import type { SeleccionAnime } from './views';

let comprobaciones = 0;

function comprueba(nombre: string, obtenido: unknown, esperado: unknown): void {
  comprobaciones += 1;
  const a = JSON.stringify(obtenido);
  const b = JSON.stringify(esperado);
  if (a !== b) {
    throw new Error(`${nombre}: obtenido ${a}, esperado ${b}`);
  }
}

function sel(partial: Partial<SeleccionAnime>): SeleccionAnime {
  return { provider: '', animeID: '', title: '', ...partial };
}

// 1. Mismo slug, distinto proveedor => claves distintas (detalle y reproductor).
seleccion.set(sel({ provider: 'jkanime', animeID: 'bleach', title: 'Bleach' }));
comprueba('detalle JKAnime bleach', claveScroll('detalle'), 'detalle:jkanime:bleach');

seleccion.set(
  sel({ provider: 'jkanime', animeID: 'bleach', episodeID: 'jkanime:episode:1' })
);
comprueba('reproductor JKAnime bleach ep1', claveScroll('reproductor'), 'reproductor:jkanime:jkanime:episode:1');

seleccion.set(sel({ provider: 'animeav1', animeID: 'bleach', title: 'Bleach' }));
comprueba('detalle AnimeAV1 bleach', claveScroll('detalle'), 'detalle:animeav1:bleach');
comprueba(
  'mismo slug distinto proveedor => claves distintas',
  claveScroll('detalle') === 'detalle:jkanime:bleach',
  false
);

comprueba(
  'mismos proveedores distintos slugs => claves distintas',
  (() => {
    seleccion.set(sel({ provider: 'jkanime', animeID: 'bleach' }));
    const a = claveScroll('detalle');
    seleccion.set(sel({ provider: 'jkanime', animeID: 'naruto' }));
    return a === claveScroll('detalle');
  })(),
  false
);

// 2. guardarScroll/recuperarScroll con dos claves del mismo slug (distinto
// proveedor) no se pisan.
seleccion.set(sel({ provider: 'jkanime', animeID: 'bleach' }));
const claveJk = claveScroll('detalle');
seleccion.set(sel({ provider: 'animeav1', animeID: 'bleach' }));
const claveAv1 = claveScroll('detalle');

comprueba('claves de scroll distintas', claveJk === claveAv1, false);
guardarScroll(claveJk, 800);
guardarScroll(claveAv1, 0);
comprueba('scroll JKAnime conserva 800', recuperarScroll(claveJk), 800);
comprueba('scroll AnimeAV1 conserva 0', recuperarScroll(claveAv1), 0);

// 3. Clave inexistente => 0.
comprueba('clave inexistente', recuperarScroll('detalle:nadie:ninguno'), 0);

// 4. Selección null => clave estable sin lanzar.
seleccion.set(null);
comprueba('detalle sin selección', claveScroll('detalle'), 'detalle::');
comprueba('reproductor sin selección', claveScroll('reproductor'), 'reproductor::');
comprueba('búsqueda es única', claveScroll('busqueda'), 'busqueda');

// La clave vacía sigue siendo estable tras guardar/recuperar (no colisiona con
// la búsqueda, que es su propio literal).
guardarScroll(claveScroll('busqueda'), 120);
comprueba('scroll de búsqueda', recuperarScroll('busqueda'), 120);

console.log(`OK (${comprobaciones} comprobaciones)`);
