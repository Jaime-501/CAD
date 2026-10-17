// T70 — Relación de contraste WCAG del sistema de tokens (Node, sin dependencias).
// Uso:  node docs/investigacion/ui/contraste.mjs
// Salida: tabla Markdown lista para pegar en docs/investigacion/ui_tokens.md.
//
// Los valores espejan el bloque `:root` añadido a frontend/src/style.css (T70).

const TOKENS = {
  'bg-0': '#090d16',
  'brand-400': '#34d399',
  'brand-500': '#10b981',
  'text-1': '#f1f5f9',
  'text-2': '#cbd5e1',
  'text-3': '#94a3b8',
  'text-4': '#64748b',
  'on-brand': '#020617',
  danger: '#f87171',
};

// Pares del enunciado: [frente, fondo].
const PARES = [
  ['text-1', 'bg-0'],
  ['text-2', 'bg-0'],
  ['text-3', 'bg-0'],
  ['text-4', 'bg-0'],
  ['brand-400', 'bg-0'],
  ['on-brand', 'brand-500'],
  ['danger', 'bg-0'],
];

function hexARgb(hex) {
  let h = hex.trim().replace('#', '');
  if (h.length === 3) h = h.split('').map((c) => c + c).join('');
  if (h.length !== 6) throw new Error(`hex inválido: ${hex}`);
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
}

// Luminancia relativa WCAG 2.x.
function luminancia([r, g, b]) {
  const lin = (c) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  };
  const [R, G, B] = [lin(r), lin(g), lin(b)];
  return 0.2126 * R + 0.7152 * G + 0.0722 * B;
}

function contraste(frente, fondo) {
  const lf = luminancia(hexARgb(frente));
  const lb = luminancia(hexARgb(fondo));
  const [mayor, menor] = lf >= lb ? [lf, lb] : [lb, lf];
  return (mayor + 0.05) / (menor + 0.05);
}

const filas = PARES.map(([f, fondo]) => {
  const r = contraste(TOKENS[f], TOKENS[fondo]);
  const aa = r >= 4.5;
  return { frente: f, fondo, hexF: TOKENS[f], hexB: TOKENS[fondo], r, aa };
});

console.log('| frente | fondo | hex frente | hex fondo | relación | AA (≥ 4.5:1) |');
console.log('|---|---|---|---|---:|:--:|');
for (const f of filas) {
  console.log(
    `| \`--${f.frente}\` | \`--${f.fondo}\` | \`${f.hexF}\` | \`${f.hexB}\` | ${f.r.toFixed(2)}:1 | ${f.aa ? 'sí' : 'NO'} |`
  );
}

const fallos = filas.filter((f) => !f.aa).map((f) => `--${f.frente}`);
console.log(
  `\nAA texto normal (≥ 4.5:1): ${fallos.length === 0 ? 'todos los pares pasan' : 'NO pasan: ' + fallos.join(', ')}`
);
