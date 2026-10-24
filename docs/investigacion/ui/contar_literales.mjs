// T69 — Contador de literales CSS por archivo (solo lectura de frontend/src).
// Uso:  node docs/investigacion/ui/contar_literales.mjs
// Salida: tabla Markdown por archivo + agregados (colores únicos, breakpoints).

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, extname } from 'node:path';

const RAIZ = 'frontend/src';

function archivos(dir) {
  const out = [];
  for (const nombre of readdirSync(dir)) {
    const p = join(dir, nombre);
    if (statSync(p).isDirectory()) continue;
    const ext = extname(nombre);
    if (ext === '.svelte' || ext === '.css') out.push(p);
  }
  return out.sort();
}

// Para .svelte solo cuenta dentro de <style>…</style>; para .css, todo el archivo.
function cssDe(archivo) {
  const texto = readFileSync(archivo, 'utf8');
  if (extname(archivo) === '.css') return texto;
  const bloques = [...texto.matchAll(/<style[^>]*>([\s\S]*?)<\/style>/g)];
  return bloques.map((m) => m[1]).join('\n');
}

const rxHex = /#[0-9a-fA-F]{3,8}\b/g;
const rxRgb = /\brgba?\(/g;
const rxHsl = /\bhsla?\(/g;
const rxSombra = /box-shadow\s*:/g;
const rxRadio = /border-radius\s*:/g;
const rxBackdrop = /backdrop-filter\s*:/g;
const rxBlur = /\bblur\(/g;
const rxFijoPx =
  /(?:^|[;{\s])(width|height|min-width|max-width|min-height|max-height|flex-basis|flex)\s*:[^;}]*?\d+(?:\.\d+)?px/g;
const rxHover = /:hover/g;
const rxMedia = /@media[^{]*/g;

function contar(texto, rx) {
  const m = texto.match(rx);
  return m ? m.length : 0;
}

const filas = [];
const colores = new Map();
const breakpoints = new Map();
const mediasPorArchivo = new Map();

for (const archivo of archivos(RAIZ)) {
  const css = cssDe(archivo);
  if (css.trim() === '') continue;
  const rel = archivo.replace(/\\/g, '/').replace('frontend/src/', '');
  const medias = (css.match(rxMedia) ?? []).map((s) => s.trim().replace(/\s+/g, ' '));
  mediasPorArchivo.set(rel, medias);
  for (const q of medias) {
    const bp = /(\d+)px/.exec(q);
    if (bp) breakpoints.set(bp[0], (breakpoints.get(bp[0]) ?? 0) + 1);
  }
  for (const h of css.match(rxHex) ?? []) {
    const n = h.toLowerCase();
    colores.set(n, (colores.get(n) ?? 0) + 1);
  }
  filas.push({
    archivo: rel,
    lineas: readFileSync(archivo, 'utf8').split('\n').length,
    hex: contar(css, rxHex),
    rgb: contar(css, rxRgb),
    hsl: contar(css, rxHsl),
    sombra: contar(css, rxSombra),
    radio: contar(css, rxRadio),
    backdrop: contar(css, rxBackdrop),
    blur: contar(css, rxBlur),
    fijoPx: contar(css, rxFijoPx),
    hover: contar(css, rxHover),
    media: medias.length,
  });
}

console.log('| archivo | líneas | hex | rgb(a) | hsl | box-shadow | border-radius | backdrop-filter | blur( ) | ancho/alto px | :hover | @media |');
console.log('|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|');
for (const f of filas) {
  console.log(
    `| ${f.archivo} | ${f.lineas} | ${f.hex} | ${f.rgb} | ${f.hsl} | ${f.sombra} | ${f.radio} | ${f.backdrop} | ${f.blur} | ${f.fijoPx} | ${f.hover} | ${f.media} |`
  );
}
const suma = (k) => filas.reduce((a, f) => a + f[k], 0);
console.log(
  `| **TOTAL** | ${suma('lineas')} | ${suma('hex')} | ${suma('rgb')} | ${suma('hsl')} | ${suma('sombra')} | ${suma('radio')} | ${suma('backdrop')} | ${suma('blur')} | ${suma('fijoPx')} | ${suma('hover')} | ${suma('media')} |`
);

console.log('\n## Breakpoints (@media con px)');
for (const [bp, n] of [...breakpoints.entries()].sort((a, b) => parseInt(a) - parseInt(b))) {
  console.log(`- ${bp}: ${n}`);
}

console.log('\n## @media por archivo');
for (const [rel, medias] of mediasPorArchivo) {
  if (medias.length === 0) continue;
  console.log(`- ${rel}:`);
  for (const q of medias) console.log(`    - ${q}`);
}

console.log('\n## Colores hex únicos (valor: veces)');
for (const [c, n] of [...colores.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))) {
  console.log(`- ${c}: ${n}`);
}
