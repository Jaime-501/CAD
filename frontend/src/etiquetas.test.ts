/**
 * Tests de las etiquetas de fuentes (T34) con dobles: SUB, DUB y sin idioma.
 *
 * El frontend no tiene runner de tests (fuera de `frontend/src/**` no se puede
 * tocar `package.json`), así que este archivo se ejecuta a mano con el
 * esbuild que ya trae Vite:
 *
 *   docker compose run --rm app sh -c \
 *     "cd frontend && ./node_modules/.bin/esbuild src/etiquetas.test.ts \
 *       --bundle --platform=node --format=cjs --outfile=/tmp/etiquetas.test.js \
 *       && node /tmp/etiquetas.test.js"
 */
import {
  etiquetaFuente,
  etiquetaIdioma,
  etiquetaTipo,
  fuentesVisibles,
} from './etiquetas';
import type { VideoSourceView } from './api';

let comprobaciones = 0;

function comprueba(nombre: string, obtenido: unknown, esperado: unknown): void {
  comprobaciones += 1;
  const a = JSON.stringify(obtenido);
  const b = JSON.stringify(esperado);
  if (a !== b) {
    throw new Error(`${nombre}: obtenido ${a}, esperado ${b}`);
  }
}

/** Doble de fuente: los mismos campos que entrega `getVideoSources`. */
function fuente(
  partial: Partial<VideoSourceView> & Pick<VideoSourceView, 'url' | 'type'>
): VideoSourceView {
  return { quality: '', server: '', language: '', ...partial };
}

const sub = fuente({
  url: 'http://127.0.0.1:1/proxy?s=1',
  type: 'hls',
  server: 'HLS',
  language: 'SUB',
  quality: '1080p',
});
const dub = fuente({
  url: 'http://127.0.0.1:1/proxy?s=2',
  type: 'hls',
  server: 'HLS',
  language: 'DUB',
});
const sinIdioma = fuente({
  url: 'https://voe.sx/e/abc',
  type: 'iframe',
  server: 'VOE',
  language: '',
  quality: '720p',
});
const lista = [sub, dub, sinIdioma];

// etiquetaIdioma: SUB, DUB y "sin idioma" (JKAnime) → cadena vacía.
comprueba('etiquetaIdioma SUB', etiquetaIdioma(sub), 'SUB');
comprueba('etiquetaIdioma DUB', etiquetaIdioma(dub), 'DUB');
comprueba('etiquetaIdioma sin idioma', etiquetaIdioma(sinIdioma), '');
comprueba('etiquetaIdioma sin campo', etiquetaIdioma(fuente({ url: 'u', type: 'mp4' })), '');

// etiquetaFuente: texto completo, jamás solo color.
comprueba('etiquetaFuente SUB', etiquetaFuente(sub), 'HLS · SUB · 1080p');
comprueba('etiquetaFuente DUB sin calidad', etiquetaFuente(dub), 'HLS · DUB');
comprueba(
  'etiquetaFuente sin idioma no pinta idioma',
  etiquetaFuente(sinIdioma),
  'VOE · 720p'
);
comprueba(
  'etiquetaFuente sin servidor usa el tipo',
  etiquetaFuente(fuente({ url: 'u', type: 'mp4', language: 'SUB' })),
  'MP4 · SUB'
);

// fuentesVisibles: sin filtro (lo que hay hoy) se ven TODAS, con el orden
// del backend; con filtro, solo las de ese idioma.
comprueba('fuentesVisibles sin filtro', fuentesVisibles(lista, null), lista);
comprueba(
  'fuentesVisibles SUB',
  fuentesVisibles(lista, 'SUB'),
  [sub]
);
comprueba(
  'fuentesVisibles DUB',
  fuentesVisibles(lista, 'DUB'),
  [dub]
);
comprueba(
  'fuentesVisibles sin idioma no coincide con ningún filtro',
  [fuentesVisibles(lista, 'SUB').includes(sinIdioma), fuentesVisibles(lista, 'DUB').includes(sinIdioma)],
  [false, false]
);

// etiquetaTipo: valores crudos de los dos proveedores → texto amable.
comprueba('etiquetaTipo tv-anime (AnimeAV1)', etiquetaTipo('tv-anime'), 'TV');
comprueba('etiquetaTipo series (JKAnime)', etiquetaTipo('series'), 'Serie');
comprueba('etiquetaTipo movie', etiquetaTipo('movie'), 'Película');
comprueba('etiquetaTipo pelicula (AnimeAV1)', etiquetaTipo('pelicula'), 'Película');
comprueba('etiquetaTipo especial', etiquetaTipo('especial'), 'Especial');
comprueba('etiquetaTipo unknown', etiquetaTipo('unknown'), '');
comprueba('etiquetaTipo vacío', etiquetaTipo(''), '');
comprueba('etiquetaTipo desconocido se respeta', etiquetaTipo('web'), 'web');

console.log(`OK (${comprobaciones} comprobaciones)`);
