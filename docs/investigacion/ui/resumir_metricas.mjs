// T69 — Resume docs/investigacion/ui/metricas.json para el informe.
// Uso: node docs/investigacion/ui/resumir_metricas.mjs
import { readFileSync } from 'node:fs';

const { metricas, errores } = JSON.parse(readFileSync('docs/investigacion/ui/metricas.json', 'utf8'));

const claves = Object.keys(metricas);
console.log('| vista@ancho | overflowH | táctiles<44 | <12px | <14px | tarj. episodio | tarj. Home | columnas | contain-intrinsic |');
console.log('|---|---:|---:|---:|---:|---:|---:|---:|---|');
for (const k of claves) {
  const m = metricas[k];
  const a = m.alturas;
  console.log(
    `| ${k} | ${m.overflowH} | ${m.interactivosPequenos.length} | ${m.fuentesMenores12.length} | ${m.cuerpoMenores14.length} | ${a.tarjetaEpisodio ?? '—'} | ${a.tarjetaHome ?? '—'} | ${a.columnasRejilla ?? '—'} | ${a.containIntrinsic ?? '—'} |`
  );
}

// Objetivos táctiles únicos
const tactiles = new Map();
for (const k of claves) {
  for (const it of metricas[k].interactivosPequenos) {
    const key = `${it.selector}|${it.w}x${it.h}|${it.texto}`;
    if (!tactiles.has(key)) tactiles.set(key, new Set());
    tactiles.get(key).add(k);
  }
}
console.log(`\n## Objetivos táctiles < 44 px (${tactiles.size} combinaciones únicas)`);
for (const [key, vistas] of [...tactiles.entries()].sort()) {
  const [sel, wh, texto] = key.split('|');
  console.log(`- \`${sel}\` ${wh} «${texto}» — ${[...vistas].join(', ')}`);
}

// Fuentes < 12 px únicas
const fuentes = new Map();
for (const k of claves) {
  for (const it of metricas[k].fuentesMenores12) {
    const key = `${it.px}|${it.selector}`;
    if (!fuentes.has(key)) fuentes.set(key, new Set());
    fuentes.get(key).add(k);
  }
}
console.log(`\n## Textos con font-size < 12 px (${fuentes.size} combinaciones)`);
for (const [key, vistas] of [...fuentes.entries()].sort()) {
  const [px, sel] = key.split('|');
  console.log(`- ${px}px \`${sel}\` — ${[...vistas].slice(0, 4).join(', ')}${vistas.size > 4 ? '…' : ''}`);
}

console.log('\n## errores:', errores.length ? errores : 'ninguno');
