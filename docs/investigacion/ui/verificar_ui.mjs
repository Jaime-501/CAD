// T71 — Verificación visual reutilizable de la UI (v1.4.0).
//
// Herramienta **solo de lectura** (no toca código): lee los tokens de diseño por
// `getComputedStyle`, captura las vistas a varios anchos midiendo `overflowH`, y
// compara dos carpetas de capturas con tolerancia por canal (≤ 2 por defecto).
//
// Uso (Playwright desde el MCP global, igual que `medir_ui.mjs`):
//   NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" \
//     node docs/investigacion/ui/verificar_ui.mjs capturar [dirOut]
//   NODE_PATH=... node docs/investigacion/ui/verificar_ui.mjs comparar <dirBase> <dirActual> [tol]
//   NODE_PATH=... node docs/investigacion/ui/verificar_ui.mjs pixel <png> <x> <y>
//
// `capturar` deja los PNG en <dirOut> (por defecto
// `docs/investigacion/ui/verif`) y un `resumen.json` con `overflowH` por vista.
// `comparar` informa de los píxeles/cajas que difieren más de `tol` por canal.
// `pixel` imprime el color (hex + rgba) de un punto de una captura: sirve para
// comprobar fondos (p. ej. que y=100 y y=1000 tienen el mismo color).

import { createRequire } from 'node:module';
import { readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { inflateSync } from 'node:zlib';

const require = createRequire(import.meta.url);
let chromium;
try {
  ({ chromium } = require('playwright'));
} catch {
  ({ chromium } = require('playwright-core'));
}

const BASE = process.env.UI_URL ?? 'http://localhost:34115';
const DIR_DEF = 'docs/investigacion/ui/verif';

// Tokens del sistema (T70/T71) que se leen por computed style.
const TOKENS = [
  '--bg-0', '--bg-1', '--bg-2', '--text-1', '--text-2', '--text-3', '--text-4',
  '--brand-400', '--brand-500', '--brand-600', '--brand-glow', '--brand-teal', '--on-brand',
  '--grad-brand', '--grad-chip-active', '--danger', '--alert', '--fav', '--indigo',
  '--r-sm', '--r-md', '--r-lg', '--r-xl', '--r-full', '--ring', '--frame-poster',
  '--blur-sm', '--blur-md', '--blur-lg', '--ease-out', '--ease-std',
  '--dur-fast', '--dur', '--dur-slow', '--sp-1', '--sp-8', '--tap', '--gutter',
  '--z-base', '--z-toast', '--font-sans', '--font-display',
  '--fs-xs', '--fs-sm', '--fs-md', '--fs-lg', '--fs-xl', '--fs-2xl',
  '--lh-tight', '--lh', '--fw-regular', '--fw-medium', '--fw-semibold', '--fw-bold',
  '--sat', '--sar', '--sab', '--sal',
];

const ANCHOS = [
  { w: 360, h: 740, nombre: '360' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1280, h: 800, nombre: '1280' },
];

function log(...a) { process.stderr.write(['[verificar_ui]', ...a].join(' ') + '\n'); }

// ---------------------------------------------------------------------------
// PNG: decodificación mínima (8 bits, color type 2 RGB / 6 RGBA) para comparar.
// ---------------------------------------------------------------------------

function decodePNG(buf) {
  if (buf.readUInt32BE(0) !== 0x89504e47) throw new Error('no es PNG');
  let off = 8;
  let width = 0, height = 0, colorType = 6, bitDepth = 8;
  const idat = [];
  while (off < buf.length) {
    const len = buf.readUInt32BE(off);
    const type = buf.toString('ascii', off + 4, off + 8);
    const data = buf.subarray(off + 8, off + 8 + len);
    if (type === 'IHDR') {
      width = data.readUInt32BE(0);
      height = data.readUInt32BE(4);
      bitDepth = data[8];
      colorType = data[9];
    } else if (type === 'IDAT') {
      idat.push(data);
    } else if (type === 'IEND') {
      break;
    }
    off += 12 + len;
  }
  if (bitDepth !== 8) throw new Error('bitDepth no soportado: ' + bitDepth);
  const channels = colorType === 6 ? 4 : colorType === 2 ? 3 : 0;
  if (!channels) throw new Error('colorType no soportado: ' + colorType);
  const raw = inflateSync(Buffer.concat(idat));
  const stride = width * channels;
  const out = Buffer.alloc(stride * height);
  let pos = 0;
  const prev = Buffer.alloc(stride);
  for (let y = 0; y < height; y++) {
    const filter = raw[pos++];
    const line = raw.subarray(pos, pos + stride);
    pos += stride;
    const cur = out.subarray(y * stride, (y + 1) * stride);
    for (let x = 0; x < stride; x++) {
      const a = x >= channels ? cur[x - channels] : 0;
      const b = prev[x];
      const c = x >= channels ? prev[x - channels] : 0;
      const v = line[x];
      let r;
      switch (filter) {
        case 0: r = v; break;
        case 1: r = v + a; break;
        case 2: r = v + b; break;
        case 3: r = v + ((a + b) >> 1); break;
        case 4: {
          const p = a + b - c;
          const pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c);
          r = v + (pa <= pb && pa <= pc ? a : pb <= pc ? b : c);
          break;
        }
        default: throw new Error('filtro PNG ' + filter);
      }
      cur[x] = r & 0xff;
    }
    prev.set(cur);
  }
  return { width, height, channels, data: out };
}

function pixelAt(img, x, y) {
  const i = (y * img.width + x) * img.channels;
  const d = img.data;
  return [d[i], d[i + 1], d[i + 2], img.channels === 4 ? d[i + 3] : 255];
}

const hex = (p) => '#' + p.slice(0, 3).map((v) => v.toString(16).padStart(2, '0')).join('');

/** Diferencias por canal > tol entre dos PNG; devuelve conteo y primera caja. */
function diffPNG(a, b, tol) {
  if (a.width !== b.width || a.height !== b.height) {
    return { dims: false, a: `${a.width}x${a.height}`, b: `${b.width}x${b.height}` };
  }
  let distintos = 0, maxDelta = 0;
  let minX = Infinity, minY = Infinity, maxX = -1, maxY = -1;
  for (let y = 0; y < a.height; y++) {
    for (let x = 0; x < a.width; x++) {
      const i = (y * a.width + x) * a.channels;
      let d = 0;
      for (let c = 0; c < 3; c++) d = Math.max(d, Math.abs(a.data[i + c] - b.data[i + c]));
      if (d > tol) {
        distintos++;
        if (d > maxDelta) maxDelta = d;
        if (x < minX) minX = x;
        if (y < minY) minY = y;
        if (x > maxX) maxX = x;
        if (y > maxY) maxY = y;
      }
    }
  }
  return {
    dims: true,
    width: a.width,
    height: a.height,
    distintos,
    maxDelta,
    caja: distintos ? { x: minX, y: minY, w: maxX - minX + 1, h: maxY - minY + 1 } : null,
  };
}

// ---------------------------------------------------------------------------
// Captura
// ---------------------------------------------------------------------------

const MEDIR = () => {
  const d = document.documentElement;
  const buscador = document.querySelector('.buscador input');
  const tarjetaEp = document.querySelector('.rejilla .celda .tarjeta');
  const tarjetaHome = document.querySelector('.fila .carrusel .tarjeta');
  const pestana = document.querySelector('.pestana');
  const botones = [...document.querySelectorAll('button')].filter((b) => b.offsetParent !== null);
  const alto = (el) => (el ? Math.round(el.getBoundingClientRect().height * 100) / 100 : null);
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    scrollWidth: d.scrollWidth,
    clientWidth: d.clientWidth,
    innerWidth: window.innerWidth,
    hayBuscador: !!buscador,
    alturas: {
      tarjetaEpisodio: alto(tarjetaEp),
      tarjetaHome: alto(tarjetaHome),
      pestana: alto(pestana),
    },
    // Alturas de los botones visibles (para comparar contra T69).
    botones: botones.map((b) => ({
      texto: (b.textContent || '').trim().slice(0, 24),
      w: Math.round(b.getBoundingClientRect().width),
      h: Math.round(b.getBoundingClientRect().height),
    })),
  };
};

async function gotoApp(page) {
  await page.goto(BASE, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('.buscador input', { timeout: 20000 });
}

async function volverABusqueda(page) {
  for (let i = 0; i < 4; i++) {
    if (await page.locator('.buscador input').isVisible().catch(() => false)) return;
    await page.locator('.cabecera .atras').first().click().catch(() => {});
    await page.waitForTimeout(600);
  }
}

async function buscar(page, texto) {
  await volverABusqueda(page);
  const input = page.locator('.buscador input');
  await input.fill('');
  await input.fill(texto);
  await page.waitForSelector('main .rejilla .tarjeta', { timeout: 45000 });
  await page.waitForTimeout(500);
}

async function abrirDetalle(page, proveedor) {
  const tarjetas = page.locator('main .rejilla .tarjeta');
  const n = await tarjetas.count();
  for (let i = 0; i < n; i++) {
    const t = tarjetas.nth(i);
    const prov = ((await t.locator('.proveedor').first().textContent()) ?? '').trim();
    if (prov === proveedor) { await t.click(); break; }
  }
  await page.waitForSelector('.ficha, .estado', { timeout: 45000 });
  await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 60000 }).catch(() => {});
  await page.waitForTimeout(700);
}

async function leerTokens(page) {
  return page.evaluate((tokens) => {
    const cs = getComputedStyle(document.documentElement);
    const vals = {};
    const vacios = [];
    for (const t of tokens) {
      const v = cs.getPropertyValue(t).trim();
      vals[t] = v;
      if (!v) vacios.push(t);
    }
    const h1 = document.querySelector('h1');
    return {
      valores: vals,
      vacios,
      bodyFont: getComputedStyle(document.body).fontFamily,
      headingFont: h1 ? getComputedStyle(h1).fontFamily : null,
      headingTag: h1 ? 'h1' : (document.querySelector('h2') ? 'h2' : null),
      headingFontFallback: !h1 && document.querySelector('h2') ? getComputedStyle(document.querySelector('h2')).fontFamily : null,
    };
  }, TOKENS);
}

async function capturar(dirOut) {
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const resumen = { tokens: null, vistas: {}, errores: [] };
  try {
    for (const a of ANCHOS) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      resumen.tokens = resumen.tokens ?? (await leerTokens(page));
      const guardar = async (vista, fullPage = false) => {
        const archivo = join(dirOut, `${vista}_${a.nombre}.png`);
        await page.screenshot({ path: archivo, fullPage }).catch((e) => resumen.errores.push(`${vista}@${a.nombre}: ${e.message}`));
        resumen.vistas[`${vista}@${a.nombre}`] = await page.evaluate(MEDIR).catch((e) => ({ error: e.message }));
        log('captura', vista, a.nombre, fullPage ? '(fullPage)' : '');
      };
      await guardar('descubrir', true);
      await buscar(page, 'bleach');
      await guardar('busqueda', true);
      await abrirDetalle(page, 'animeav1');
      await guardar('detalle_animeav1');
      await buscar(page, 'bleach');
      await abrirDetalle(page, 'jkanime');
      await guardar('detalle_jkanime');
    }
  } catch (e) {
    resumen.errores.push(e.message);
  }
  writeFileSync(join(dirOut, 'resumen.json'), JSON.stringify(resumen, null, 2));
  await browser.close();
  return resumen;
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

const modo = process.argv[2] ?? 'capturar';

if (modo === 'comparar') {
  const [dirA, dirB] = [process.argv[3], process.argv[4]];
  const tol = Number(process.argv[5] ?? 2);
  if (!dirA || !dirB || !existsSync(dirA) || !existsSync(dirB)) {
    log('uso: comparar <dirBase> <dirActual> [tol]');
    process.exit(1);
  }
  const nombres = readdirSync(dirA).filter((f) => f.toLowerCase().endsWith('.png')).sort();
  let iguales = 0, distintos = 0, faltan = 0;
  console.log(`Comparación por canal (tol ≤ ${tol}): ${dirA} vs ${dirB}`);
  for (const nombre of nombres) {
    const pa = join(dirA, nombre), pb = join(dirB, nombre);
    if (!existsSync(pa) || !existsSync(pb)) { faltan++; continue; }
    const r = diffPNG(decodePNG(readFileSync(pa)), decodePNG(readFileSync(pb)), tol);
    if (!r.dims) { console.log(`- ${nombre}: DIMENSIONES ${r.a} vs ${r.b}`); distintos++; continue; }
    if (r.distintos === 0) { iguales++; console.log(`- ${nombre}: idénticas (${r.width}x${r.height})`); }
    else {
      distintos++;
      console.log(`- ${nombre}: ${r.distintos} px distintos (máx Δ${r.maxDelta}), caja ${JSON.stringify(r.caja)}`);
    }
  }
  console.log(`Resumen: ${iguales} idénticas, ${distintos} con diferencias, ${faltan} sin par.`);
} else if (modo === 'pixel') {
  const [png, x, y] = [process.argv[3], Number(process.argv[4]), Number(process.argv[5])];
  const img = decodePNG(readFileSync(png));
  const p = pixelAt(img, x, y);
  console.log(`${png} (${img.width}x${img.height}) px(${x},${y}) = ${hex(p)} rgba(${p.join(',')})`);
} else {
  const dirOut = process.argv[3] ?? DIR_DEF;
  const res = await capturar(dirOut);
  console.log('Tokens: vacíos =', res.tokens?.vacios?.length ? res.tokens.vacios : 'ninguno');
  console.log('body font =', res.tokens?.bodyFont);
  console.log(`heading (${res.tokens?.headingTag}) font =`, res.tokens?.headingFont ?? res.tokens?.headingFontFallback);
  console.log('\n| vista@ancho | overflowH | tarj. episodio | tarj. Home | pestaña |');
  console.log('|---|---:|---:|---:|---:|');
  for (const [k, m] of Object.entries(res.vistas)) {
    const a = m.alturas ?? {};
    console.log(`| ${k} | ${m.overflowH} | ${a.tarjetaEpisodio ?? '—'} | ${a.tarjetaHome ?? '—'} | ${a.pestana ?? '—'} |`);
  }
  if (res.errores.length) console.log('\nErrores:', res.errores);
  console.log(`\nCapturas y resumen en ${dirOut}`);
}
