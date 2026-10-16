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
  agruparFuentes,
  esExtra,
  esPrincipal,
  etiquetaFuente,
  etiquetaIdioma,
  etiquetaTipo,
  fuentesVisibles,
  ordenReproduccion,
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

// T63 — clasificación por TIPO (nunca por nombre): principal = hls/mp4,
// extra = iframe.
const mp4Directo = fuente({
  url: 'https://a2.mp4upload.com:183/d/tok/video.mp4',
  type: 'mp4',
  server: 'MP4Upload',
});
const iframeMp4 = fuente({
  url: 'https://www.mp4upload.com/embed-abc.html',
  type: 'iframe',
  server: 'Mp4upload',
});
const iframeByse = fuente({ url: 'https://byse.example/e/1', type: 'iframe', server: 'Byse' });

comprueba(
  'esPrincipal hls y mp4, no iframe',
  [esPrincipal(sub), esPrincipal(mp4Directo), esPrincipal(iframeMp4)],
  [true, true, false]
);
comprueba(
  'esExtra solo iframe',
  [esExtra(iframeMp4), esExtra(mp4Directo), esExtra(sub)],
  [true, false, false]
);

// Colisión de nombre (sin distinguir mayúsculas): el `Mp4upload` iframe y el
// `MP4Upload` mp4 directo se marcan `· directo` / `· iframe`.
const colision = agruparFuentes([mp4Directo, iframeMp4]);
comprueba(
  'colisión: principales con `· directo`',
  colision.principales.map((f) => f.etiqueta),
  ['MP4Upload · directo']
);
comprueba(
  'colisión: extras con `· iframe`',
  colision.extras.map((f) => f.etiqueta),
  ['Mp4upload · iframe']
);
comprueba(
  'colisión: índices sobre la lista completa',
  [colision.principales[0].indice, colision.extras[0].indice],
  [0, 1]
);

// Sin colisión (HLS de JKAnime vs iframe VOE): etiqueta como siempre.
const sinColision = agruparFuentes(lista);
comprueba(
  'sin colisión: principales intactos (SUB/DUB)',
  sinColision.principales.map((f) => f.etiqueta),
  ['HLS · SUB · 1080p', 'HLS · DUB']
);
comprueba('sin colisión: extra intacto', sinColision.extras.map((f) => f.etiqueta), ['VOE · 720p']);

// Lista solo de iframes: no hay principales y los extras conservan el orden.
const soloIframes = agruparFuentes([sinIdioma, iframeByse]);
comprueba('solo iframes: sin principales', soloIframes.principales.length, 0);
comprueba(
  'solo iframes: extras en orden',
  soloIframes.extras.map((f) => f.etiqueta),
  ['VOE · 720p', 'Byse']
);

// Lista vacía: nada en los dos grupos.
const vacio = agruparFuentes([]);
comprueba('lista vacía: grupos vacíos', [vacio.principales.length, vacio.extras.length], [0, 0]);

// ordenReproduccion: la del backend primero, luego el resto de principales y
// los iframes al final.
comprueba(
  'ordenReproduccion: hls, mp4 y luego iframes',
  ordenReproduccion([sub, sinIdioma, mp4Directo, iframeMp4]),
  [0, 2, 1, 3]
);
comprueba(
  'ordenReproduccion: la del backend (índice 0) va primero aunque sea iframe',
  ordenReproduccion([sinIdioma, sub]),
  [0, 1]
);

console.log(`OK (${comprobaciones} comprobaciones)`);
