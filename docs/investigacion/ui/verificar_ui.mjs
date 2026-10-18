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

// T73: anchos del modo `shell` (barra inferior móvil).
const ANCHOS_SHELL = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
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

// T72: mide los objetivos táctiles de las primitivas (`.btn` debe llegar a
// 44×44 y `.chip` a 40 px de alto). Pensado para el arnés de galería temporal.
const MEDIR_TACTILES = () => {
  const medir = (sel) =>
    [...document.querySelectorAll(sel)].map((el) => {
      const r = el.getBoundingClientRect();
      return {
        texto: (el.textContent || '').trim().slice(0, 24),
        w: Math.round(r.width * 100) / 100,
        h: Math.round(r.height * 100) / 100,
      };
    });
  return { btn: medir('.btn'), chip: medir('.chip') };
};

// T73: medición del modo `shell` (barra inferior + geometría por ancho).
const MEDIR_SHELL = () => {
  const d = document.documentElement;
  const barra = document.querySelector('.barra-inferior');
  const destinos = barra ? [...barra.querySelectorAll('.destino')] : [];
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: Math.round(r.x), y: Math.round(r.y), w: round(r.width), h: round(r.height), bottom: round(r.bottom) };
  };
  const alto = (el) => (el ? round(el.getBoundingClientRect().height) : null);
  const panelVisible = (id) => {
    const p = document.getElementById(id);
    return !!p && !p.classList.contains('inactivo');
  };
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    innerHeight: window.innerHeight,
    hamburguesa: !!document.querySelector('.hamburguesa'),
    barra: {
      visible: !!barra,
      rect: rect(barra),
      destinos: destinos.map((b) => {
        const r = b.getBoundingClientRect();
        return {
          texto: (b.textContent || '').trim(),
          w: round(r.width),
          h: round(r.height),
          current: b.getAttribute('aria-current'),
        };
      }),
    },
    paneles: { descubrir: panelVisible('panel-descubrir'), favoritos: panelVisible('panel-favoritos') },
    alturas: {
      tarjetaEpisodio: alto(document.querySelector('.rejilla .celda .tarjeta')),
      tarjetaHome: alto(document.querySelector('.fila .carrusel .tarjeta')),
      pestana: alto(document.querySelector('.pestana')),
    },
  };
};

// T73: lista de objetivos táctiles < 44 px visibles, marcando si son de la
// barra inferior o del cajón (ninguno de esos dos debe aparecer en la lista).
const TACTILES_CHICOS = () => {
  const visibles = [...document.querySelectorAll('button, a, [role="button"], input')].filter(
    (b) => b.offsetParent !== null
  );
  return visibles
    .map((b) => {
      const r = b.getBoundingClientRect();
      return {
        texto:
          (b.textContent || '').trim().slice(0, 24) ||
          b.getAttribute('aria-label') ||
          b.id ||
          b.tagName.toLowerCase(),
        w: Math.round(r.width),
        h: Math.round(r.height),
        enBarra: !!b.closest('.barra-inferior'),
        enCajon: !!b.closest('.cajon'),
      };
    })
    .filter((x) => x.w < 44 || x.h < 44);
};

// T74: medición de la cabecera compacta (detalle y reproductor).
const MEDIR_CABECERA = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height), bottom: round(r.bottom) };
  };
  const cab = document.querySelector('main .cabecera');
  const atras = document.querySelector('main .cabecera .atras');
  const h2 = document.querySelector('main .cabecera h2');
  const sub = document.querySelector('main .cabecera .subtitulo');
  const accion = document.querySelector('main .cabecera .acciones .favorito');
  const estrella = document.querySelector('main .cabecera .acciones .estrella');
  const cs = cab ? getComputedStyle(cab) : null;
  const csH2 = h2 ? getComputedStyle(h2) : null;
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    scrollY: round(window.scrollY),
    sat: getComputedStyle(d).getPropertyValue('--sat').trim(),
    cabecera: rect(cab),
    posicion: cs ? cs.position : null,
    topCss: cs ? cs.top : null,
    atras: rect(atras),
    titulo: h2
      ? {
          texto: (h2.textContent || '').trim(),
          rect: rect(h2),
          whiteSpace: csH2.whiteSpace,
          overflow: csH2.overflow,
          textOverflow: csH2.textOverflow,
          recortado: h2.scrollWidth > h2.clientWidth + 1,
          textoCompleto: h2.scrollWidth <= h2.clientWidth + 1,
        }
      : null,
    subtitulo: sub ? { texto: (sub.textContent || '').trim(), rect: rect(sub) } : null,
    accion: rect(accion),
    estrella: rect(estrella),
    estrellaMarcada: estrella ? estrella.classList.contains('marcada') : null,
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
// Modo shell (T73): barra inferior móvil + cajón de historial
// ---------------------------------------------------------------------------

async function shell() {
  const dirOut = 'docs/img/v1.4/shell';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const rep = { anchos: [], funciones: [], tactiles: [], capturas: [], errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });

  const captura = async (nombre) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };

  try {
    // --- A. Medidas y capturas por ancho ---
    for (const a of ANCHOS_SHELL) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await page.waitForTimeout(500);
      const med = await page.evaluate(MEDIR_SHELL);
      if (['360', '768', '1280'].includes(a.nombre)) await captura(`home_${a.nombre}`);
      let tarjetaEp = med.alturas.tarjetaEpisodio;
      if (a.w >= 1024) {
        await buscar(page, 'bleach');
        await abrirDetalle(page, 'animeav1');
        tarjetaEp = await page.evaluate(() => {
          const el = document.querySelector('.rejilla .celda .tarjeta');
          return el ? Math.round(el.getBoundingClientRect().height * 100) / 100 : null;
        });
      }
      rep.anchos.push({ ancho: a.nombre, ...med, tarjetaEpisodioDetalle: tarjetaEp });
      log('shell ancho', a.nombre, 'barra=', med.barra.visible);
    }

    // --- B. Funcional a 360×740 ---
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(400);

    const destino = (texto) => page.locator('.barra-inferior .destino', { hasText: texto });
    const panelActivo = (id) => page.evaluate((i) => !document.getElementById(i).classList.contains('inactivo'), id);

    // B1. Descubrir/Favoritos desde Home.
    await destino('Favoritos').click();
    await page.waitForTimeout(250);
    check('Home → Favoritos', await panelActivo('panel-favoritos'));
    await destino('Descubrir').click();
    await page.waitForTimeout(250);
    check('Home → Descubrir', await panelActivo('panel-descubrir'));

    // B2. Buscar enfoca el input.
    await destino('Buscar').click();
    await page.waitForTimeout(250);
    check('Buscar enfoca el input', await page.evaluate(() => document.activeElement?.id === 'buscador-anime'));

    // B3. Buscar "bleach" → Descubrir: Home, input vacío.
    await buscar(page, 'bleach');
    await captura('busqueda_360');
    check('hay resultados de "bleach"', (await page.locator('main .rejilla .tarjeta').count()) > 0);
    await destino('Descubrir').click();
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 30000 }).catch(() => {});
    check('Descubrir tras búsqueda: input vacío', (await page.locator('#buscador-anime').inputValue()) === '');
    check('Descubrir tras búsqueda: Home visible', await page.locator('#panel-descubrir').isVisible());
    await buscar(page, 'bleach');
    await destino('Favoritos').click();
    await page.waitForTimeout(250);
    check('Favoritos tras búsqueda: panel favoritos', await panelActivo('panel-favoritos'));
    await destino('Descubrir').click();
    await page.waitForTimeout(200);

    // B4. Descubrir/Favoritos desde detalle.
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await captura('detalle_360');
    await destino('Favoritos').click();
    await page.waitForTimeout(400);
    check('detalle → Favoritos', await panelActivo('panel-favoritos'));
    check('detalle → Favoritos deja el Home', await page.locator('.pestanas').isVisible());
    await destino('Descubrir').click();
    await page.waitForTimeout(250);

    // B5. Historial abre el cajón; Escape cierra y devuelve el foco.
    await destino('Historial').click();
    await page.waitForTimeout(300);
    check('Historial abre el cajón', await page.locator('.barra.cajon').isVisible());
    await captura('cajon_360');
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);
    check('Escape cierra el cajón', !(await page.locator('.barra.cajon').isVisible().catch(() => false)));
    const foco = await page.evaluate(() => {
      const el = document.activeElement;
      const d = el?.closest?.('.destino');
      return d ? (d.textContent || '').trim() : null;
    });
    check('Escape devuelve el foco a Historial', foco === 'Historial', foco);

    // B6. Barra oculta en el reproductor y reaparece al volver.
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const ep = page.locator('.rejilla .celda .tarjeta').first();
    if (await ep.count()) {
      await ep.click();
      await page.waitForTimeout(1800);
      check('sin barra en el reproductor', (await page.locator('.barra-inferior').count()) === 0);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(600);
      check('barra reaparece al volver', (await page.locator('.barra-inferior').count()) > 0);
    } else {
      check('reproductor (sin episodios)', false, 'sin tarjetas de episodio');
    }

    // B7. Fin de scroll: Home y detalle.
    await buscar(page, 'bleach');
    await destino('Descubrir').click();
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 30000 }).catch(() => {});
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(400);
    const finHome = await page.evaluate(() => {
      const barra = document.querySelector('.barra-inferior');
      const filas = document.querySelectorAll('.fila');
      const ultima = filas[filas.length - 1];
      if (!barra || !ultima) return null;
      return { fondo: Math.round(ultima.getBoundingClientRect().bottom), topBarra: Math.round(barra.getBoundingClientRect().top) };
    });
    check('fin del Home no bajo la barra', !!finHome && finHome.fondo <= finHome.topBarra + 1, finHome);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(400);
    const finDetalle = await page.evaluate(() => {
      const barra = document.querySelector('.barra-inferior');
      const celdas = document.querySelectorAll('.rejilla .celda');
      const ultima = celdas[celdas.length - 1];
      if (!barra || !ultima) return null;
      return { fondo: Math.round(ultima.getBoundingClientRect().bottom), topBarra: Math.round(barra.getBoundingClientRect().top) };
    });
    check('fin del detalle no bajo la barra', !!finDetalle && finDetalle.fondo <= finDetalle.topBarra + 1, finDetalle);

    // B7b. Restauración de scroll T66/T67: detalle → reproductor → Atrás.
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.waitForTimeout(900);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultimaTarjeta = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultimaTarjeta.count())) {
      await ultimaTarjeta.click();
      await page.waitForTimeout(1800);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const yDespues = await page.evaluate(() => Math.round(window.scrollY));
      check('T66/T67: scroll del detalle restaurado al volver', Math.abs(yDespues - yAntes) <= 5, { yAntes, yDespues });
    } else {
      check('T66/T67: scroll del detalle restaurado al volver', false, { yAntes });
    }

    // B8. Toast sobre la barra (borrar y deshacer un historial).
    await destino('Historial').click();
    await page.waitForTimeout(400);
    try {
      const envoltorio = page.locator('.barra.cajon .envoltorio').first();
      if (await envoltorio.count()) {
        await envoltorio.hover(); // el ✕ solo es visible con hover/foco
        await page.waitForTimeout(150);
        await envoltorio.locator('.borrar').click();
        await page.waitForTimeout(300);
        const toast = await page.evaluate(() => {
          const t = document.querySelector('.toast');
          const b = document.querySelector('.barra-inferior');
          if (!t) return null;
          return { bottom: Math.round(t.getBoundingClientRect().bottom), topBarra: b ? Math.round(b.getBoundingClientRect().top) : null };
        });
        check('toast sobre la barra', !!toast && toast.topBarra !== null && toast.bottom <= toast.topBarra + 1, toast);
        await captura('toast_360');
        await page.locator('.toast .accion').click().catch(() => {});
      } else {
        check('toast sobre la barra', false, 'sin historial sembrado');
      }
    } catch (e) {
      check('toast sobre la barra', false, e.message);
    }
    await page.keyboard.press('Escape').catch(() => {});
    await page.waitForTimeout(4200); // deja que cualquier toast expire

    // B9. prefers-reduced-motion: sin transiciones en la barra.
    await page.emulateMedia({ reducedMotion: 'reduce' });
    check(
      'reduced-motion: sin transición en la barra',
      (await page.evaluate(() => {
        const el = document.querySelector('.destino');
        return el ? getComputedStyle(el).transitionDuration : null;
      })) === '0s'
    );
    await page.emulateMedia({ reducedMotion: null });

    // B10. Objetivos táctiles < 44 px (Home y cajón abierto).
    rep.tactiles = (await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'home' }));
    await destino('Historial').click();
    await page.waitForTimeout(300);
    rep.tactiles = rep.tactiles.concat((await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'cajon' })));
    await page.keyboard.press('Escape').catch(() => {});
    const malos = rep.tactiles.filter((t) => t.enBarra || t.enCajon);
    check('ningún táctil <44 px en barra ni cajón', malos.length === 0, malos.slice(0, 5));
  } catch (e) {
    rep.errores.push(e.message);
  }
  writeFileSync(join(dirOut, 'shell.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo cabecera (T74): cabecera compacta de detalle y reproductor
// ---------------------------------------------------------------------------

const ANCHOS_CABECERA = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1280, h: 800, nombre: '1280' },
];

/** Abre el detalle del proveedor cuya tarjeta tenga el título más largo. */
async function abrirDetalleLargo(page, proveedor) {
  const tarjetas = page.locator('main .rejilla .tarjeta');
  const n = await tarjetas.count();
  let elegido = -1;
  let maxLargo = -1;
  for (let i = 0; i < n; i++) {
    const t = tarjetas.nth(i);
    const prov = ((await t.locator('.proveedor').first().textContent()) ?? '').trim();
    if (prov !== proveedor) continue;
    const largo = ((await t.locator('.titulo').first().textContent()) ?? '').trim().length;
    if (largo > maxLargo) {
      maxLargo = largo;
      elegido = i;
    }
  }
  if (elegido >= 0) await tarjetas.nth(elegido).click();
  await page.waitForSelector('.ficha, .estado', { timeout: 45000 });
  await page.waitForTimeout(700);
}

async function abrirReproductor(page) {
  const ep = page.locator('.rejilla .celda .tarjeta').first();
  if (!(await ep.count())) return false;
  await ep.click();
  await page.waitForSelector('.cabecera .atras', { timeout: 30000 }).catch(() => {});
  await page.waitForTimeout(1500);
  return true;
}

async function cabecera() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/cabecera';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const rep = { anchos: [], funciones: [], tactiles: [], capturas: [], errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const captura = async (nombre) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };

  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };

  // --- A. Geometría por ancho (detalle AnimeAV1) ---
  await paso('A', async () => {
    for (const a of ANCHOS_CABECERA) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await buscar(page, 'bleach');
      await abrirDetalle(page, 'animeav1');
      const med = await page.evaluate(MEDIR_CABECERA);
      rep.anchos.push({ ancho: a.nombre, vista: 'detalle', ...med });
      if (['360', '768', '1280'].includes(a.nombre)) await captura(`detalle_${a.nombre}`);
      if (a.w < 900) {
        check(`${a.nombre}: Atrás ≥44×44`, med.atras && med.atras.w >= 44 && med.atras.h >= 44, med.atras);
        check(`${a.nombre}: estrella ≥44×44`, med.estrella && med.estrella.w >= 44 && med.estrella.h >= 44, med.estrella);
        check(`${a.nombre}: sin overflow horizontal`, med.overflowH === 0, med.overflowH);
        check(`${a.nombre}: subtítulo ausente en detalle`, med.subtitulo === null, med.subtitulo);
      }
      log('cabecera ancho', a.nombre, 'atras=', JSON.stringify(med.atras), 'pos=', med.posicion);
    }
  });

  // --- B. Título largo recortado a una línea (360) ---
  await paso('B', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'tensei');
    await abrirDetalleLargo(page, 'animeav1');
    const largo = await page.evaluate(MEDIR_CABECERA);
    check('360: título largo en una línea', largo.titulo && largo.titulo.whiteSpace === 'nowrap' && largo.titulo.textOverflow === 'ellipsis', largo.titulo?.whiteSpace);
    check('360: título largo recortado', !!largo.titulo?.recortado, largo.titulo);
    rep.anchos.push({ ancho: '360', vista: 'detalle_largo', ...largo });
    await captura('detalle_largo_360');
  });

  // --- C. Cabecera fija al hacer scroll (detalle 360) ---
  await paso('C', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.evaluate(() => window.scrollTo(0, 2000));
    await page.waitForTimeout(500);
    const fija = await page.evaluate(MEDIR_CABECERA);
    const satPx = parseFloat(fija.sat) || 0;
    check('360: cabecera fija pegada arriba al scrollear', fija.cabecera && Math.abs(fija.cabecera.y - satPx) <= 2, { y: fija.cabecera?.y, sat: fija.sat, scrollY: fija.scrollY });
    check('360: posición sticky', fija.posicion === 'sticky', fija.posicion);
    rep.anchos.push({ ancho: '360', vista: 'detalle_scroll', ...fija });
    await captura('detalle_scroll_360');
  });

  // --- D. Detalle JKAnime (360) ---
  await paso('D', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'jkanime');
    const jk = await page.evaluate(MEDIR_CABECERA);
    check('360: detalle JKAnime con cabecera', !!jk.cabecera, jk.cabecera);
    check('360: JKAnime Atrás ≥44×44', jk.atras && jk.atras.w >= 44 && jk.atras.h >= 44, jk.atras);
    rep.anchos.push({ ancho: '360', vista: 'detalle_jkanime', ...jk });
    await captura('detalle_jkanime_360');
  });

  // --- E. Reproductor (360 y 768) ---
  await paso('E', async () => {
    for (const a of [{ w: 360, h: 740, nombre: '360' }, { w: 768, h: 1024, nombre: '768' }]) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await buscar(page, 'bleach');
      await abrirDetalle(page, 'animeav1');
      const abierto = await abrirReproductor(page);
      if (!abierto) {
        check(`${a.nombre}: reproductor abierto`, false, 'sin tarjetas de episodio');
        continue;
      }
      const rep1 = await page.evaluate(MEDIR_CABECERA);
      check(`${a.nombre}: reproductor con cabecera`, !!rep1.cabecera, rep1.cabecera);
      check(`${a.nombre}: reproductor cabecera NO fija`, rep1.posicion !== 'sticky', rep1.posicion);
      check(`${a.nombre}: reproductor con estrella (detalle cargado)`, !!rep1.estrella, rep1.accion);
      check(`${a.nombre}: subtítulo del episodio presente`, !!rep1.subtitulo, rep1.subtitulo);
      if (a.w < 900) {
        check(`${a.nombre}: reproductor Atrás ≥44`, rep1.atras && rep1.atras.w >= 44 && rep1.atras.h >= 44, rep1.atras);
        check(`${a.nombre}: reproductor estrella ≥44`, rep1.estrella && rep1.estrella.w >= 44 && rep1.estrella.h >= 44, rep1.estrella);
        check(`${a.nombre}: reproductor sin overflow`, rep1.overflowH === 0, rep1.overflowH);
      }
      rep.anchos.push({ ancho: a.nombre, vista: 'reproductor', ...rep1 });
      await captura(`reproductor_${a.nombre}`);
      // La cabecera no fija se va con el scroll (si la vista tiene scroll).
      if (a.w < 900) {
        await page.evaluate(() => window.scrollTo(0, 1200));
        await page.waitForTimeout(400);
        const tras = await page.evaluate(MEDIR_CABECERA);
        if (tras.scrollY > 0) {
          check(`${a.nombre}: cabecera del reproductor se va con el scroll`, tras.cabecera && tras.cabecera.y < 0, { y: tras.cabecera?.y, scrollY: tras.scrollY });
        } else {
          check(`${a.nombre}: cabecera del reproductor se va con el scroll`, true, 'sin scroll (vista corta)');
        }
      }
    }
  });

  // --- F. Alternar la estrella desde la cabecera (360) ---
  await paso('F', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const antesFav = await page.evaluate(MEDIR_CABECERA);
    await page.locator("main .cabecera .acciones .estrella").click();
    await page.waitForTimeout(600);
    const despuesFav = await page.evaluate(MEDIR_CABECERA);
    check('360: la estrella alterna el estado', antesFav.estrellaMarcada !== despuesFav.estrellaMarcada, {
      antes: antesFav.estrellaMarcada,
      despues: despuesFav.estrellaMarcada,
    });
    await page.locator("main .cabecera .acciones .estrella").click();
    await page.waitForTimeout(400);
  });

  // --- G. prefers-reduced-motion: sin transición en la cabecera ---
  await paso('G', async () => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    check(
      'reduced-motion: sin transición en la cabecera',
      (await page.evaluate(() => {
        const el = document.querySelector('main .cabecera');
        return el ? getComputedStyle(el).transitionDuration : null;
      })) === '0s'
    );
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- H. Objetivos táctiles < 44 px restantes en las vistas (360) ---
  await paso('H', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await page.evaluate(() => window.scrollTo(0, 0));
    rep.tactiles = (await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'detalle_360' }));
    check(
      'ninguno de la cabecera <44 px',
      !(await page.evaluate(() =>
        [...document.querySelectorAll('main .cabecera button, main .cabecera a')]
          .filter((b) => b.offsetParent !== null)
          .some((b) => {
            const r = b.getBoundingClientRect();
            return r.width < 44 || r.height < 44;
          })
      ))
    );
  });

  // --- I. Aviso de error forzado: no debe mover la cabecera (360) ---
  await paso('I', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const forzado = await page.evaluate(() => {
      const app = window.go?.main?.App;
      if (!app || typeof app.AddFavorite !== 'function') return false;
      window.__origAddFav = app.AddFavorite;
      window.__origRemFav = app.RemoveFavorite;
      app.AddFavorite = () => Promise.reject(new Error('forzado'));
      app.RemoveFavorite = () => Promise.reject(new Error('forzado'));
      return true;
    });
    if (!forzado) {
      check('360: aviso de error no cambia la altura', false, 'sin puente window.go en el navegador');
      return;
    }
    const hAntes = (await page.evaluate(MEDIR_CABECERA)).cabecera?.h ?? null;
    await page.locator('main .cabecera .acciones .estrella').click();
    await page.waitForSelector('main .cabecera .aviso', { timeout: 5000 });
    const medAviso = await page.evaluate(MEDIR_CABECERA);
    const aviso = await page.evaluate(() => {
      const a = document.querySelector('main .cabecera .aviso');
      if (!a) return null;
      const r = a.getBoundingClientRect();
      return { x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height) };
    });
    check('360: aviso de error no cambia la altura', medAviso.cabecera?.h === hAntes, { antes: hAntes, despues: medAviso.cabecera?.h });
    check('360: aviso de error flotante (fuera del flujo)', aviso !== null, aviso);
    rep.anchos.push({ ancho: '360', vista: 'detalle_aviso', ...medAviso });
    await captura('detalle_aviso_360');
    await page.evaluate(() => {
      const app = window.go?.main?.App;
      if (app && window.__origAddFav) app.AddFavorite = window.__origAddFav;
      if (app && window.__origRemFav) app.RemoveFavorite = window.__origRemFav;
    });
  });

  writeFileSync(join(dirOut, 'cabecera.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
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
} else if (modo === 'tactiles') {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  let fallos = 0;
  try {
    console.log('| ancho | primitiva | texto | w | h | ok |');
    console.log('|---|---|---|---:|---:|:--:|');
    for (const a of ANCHOS) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await page.goto(BASE, { waitUntil: 'domcontentloaded' });
      await page.waitForSelector('.btn', { timeout: 20000 });
      const res = await page.evaluate(MEDIR_TACTILES);
      for (const [primitiva, items] of Object.entries(res)) {
        for (const it of items) {
          const ok = primitiva === 'btn' ? it.w >= 44 && it.h >= 44 : it.h >= 40;
          if (!ok) fallos++;
          console.log(
            `| ${a.nombre} | ${primitiva} | ${it.texto || '—'} | ${it.w} | ${it.h} | ${ok ? 'sí' : 'NO'} |`
          );
        }
      }
    }
  } catch (e) {
    log('error:', e.message);
    fallos++;
  }
  await browser.close();
  console.log(fallos === 0 ? '\nTodos los objetivos táctiles cumplen.' : `\n${fallos} objetivos NO cumplen.`);
  process.exit(fallos === 0 ? 0 : 1);
} else if (modo === 'shell') {
  const rep = await shell();
  console.log('## Modo shell (T73) — barra inferior móvil\n');
  console.log('| ancho | barra | pegada al borde | destinos | todos ≥44 | hamburguesa | overflowH | tarj.Home | tarj.ep | pestaña |');
  console.log('|---|:--:|:--:|---:|:--:|:--:|---:|---:|---:|---:|');
  for (const m of rep.anchos) {
    const b = m.barra;
    const todos = b.destinos.length === 4 && b.destinos.every((d) => d.w >= 44 && d.h >= 44);
    const pegada = b.rect ? Math.abs(b.rect.bottom - m.innerHeight) < 1 : false;
    console.log(
      `| ${m.ancho} | ${b.visible ? 'sí' : 'no'} | ${b.visible ? (pegada ? 'sí' : 'NO') : '—'} | ${b.destinos.length} | ` +
        `${b.visible ? (todos ? 'sí' : 'NO') : '—'} | ${m.hamburguesa ? 'SÍ' : 'no'} | ${m.overflowH} | ` +
        `${m.alturas.tarjetaHome ?? '—'} | ${m.tarjetaEpisodioDetalle ?? '—'} | ${m.alturas.pestana ?? '—'} |`
    );
  }
  console.log('\nDestinos por ancho:');
  for (const m of rep.anchos) {
    const d = m.barra.destinos.map((x) => `${x.texto} ${x.w}×${x.h}${x.current ? ' (activo)' : ''}`).join(' · ');
    console.log(`- ${m.ancho}: ${d || 'sin barra'}`);
  }
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones) {
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  }
  console.log(`\nTáctiles <44 px visibles (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) {
    console.log(`- ${t.estado}: ${t.texto} ${t.w}×${t.h}${t.enBarra ? ' [BARRA]' : ''}${t.enCajon ? ' [CAJÓN]' : ''}`);
  }
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (docs/img/v1.4/shell/): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'cabecera') {
  const rep = await cabecera();
  console.log('## Modo cabecera (T74) — cabecera compacta de detalle y reproductor\n');
  console.log('| vista@ancho | posición | cabecera y/h | Atrás w×h | estrella w×h | overflowH |');
  console.log('|---|---|---:|---:|---:|---:|');
  for (const m of rep.anchos) {
    const fmt = (r) => (r ? `${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.vista}@${m.ancho} | ${m.posicion ?? '—'} | ${m.cabecera ? `${m.cabecera.y}/${m.cabecera.h}` : '—'} | ` +
        `${fmt(m.atras)} | ${fmt(m.estrella)} | ${m.overflowH} |`
    );
  }
  console.log('\nTítulos por ancho:');
  for (const m of rep.anchos) {
    if (!m.titulo) continue;
    console.log(
      `- ${m.vista}@${m.ancho}: "${m.titulo.texto.slice(0, 60)}" whiteSpace=${m.titulo.whiteSpace} recortado=${m.titulo.recortado}`
    );
  }
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones) {
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  }
  console.log(`\nTáctiles <44 px visibles (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) {
    console.log(`- ${t.estado}: ${t.texto} ${t.w}×${t.h}`);
  }
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (docs/img/v1.4/cabecera/): ${rep.capturas.join(', ') || 'ninguna'}`);
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
