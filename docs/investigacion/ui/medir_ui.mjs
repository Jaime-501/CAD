// T69 — Capturas + métricas de UI por vista y ancho (Playwright, solo lectura).
//
// Requisitos:
//   1) el dev server de Wails debe estar en http://localhost:34115
//      (`docker compose up`, ver AGENTS.md);
//   2) Playwright disponible. En esta máquina se usa el que ya trae el MCP global:
//        NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" \
//          node docs/investigacion/ui/medir_ui.mjs
//
// Salida:
//   - capturas  -> docs/img/v1.4/base/<vista>_<ancho>.png
//   - métricas  -> docs/investigacion/ui/metricas.json  (+ log por stdout)

import { createRequire } from 'node:module';
import { writeFileSync, mkdirSync } from 'node:fs';

// Resuelve Playwright desde el proyecto o desde el MCP global (NODE_PATH).
const require = createRequire(import.meta.url);
let chromium;
try {
  ({ chromium } = require('playwright'));
} catch {
  ({ chromium } = require('playwright-core'));
}

const BASE = process.env.UI_URL ?? 'http://localhost:34115';
const DIR_IMG = 'docs/img/v1.4/base';
const DIR_OUT = 'docs/investigacion/ui';

const ANCHOS = [
  { w: 360, h: 740, nombre: '360' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1280, h: 800, nombre: '1280' },
];

mkdirSync(DIR_IMG, { recursive: true });
mkdirSync(DIR_OUT, { recursive: true });

const metricas = {};
const errores = [];

function log(...a) {
  process.stderr.write(['[medir_ui]', ...a].join(' ') + '\n');
}

const MEDIR = () => {
  function visible(el) {
    const r = el.getBoundingClientRect();
    const s = getComputedStyle(el);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none';
  }
  function ruta(el) {
    let p = el.tagName.toLowerCase();
    if (el.id) p += '#' + el.id;
    const cls = typeof el.className === 'string' ? el.className.trim().split(/\s+/).filter(Boolean) : [];
    if (cls.length) p += '.' + cls.slice(0, 3).join('.');
    return p;
  }
  function texto(el) {
    return (el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 40);
  }
  function esHojaConTexto(el) {
    if (el.childElementCount !== 0) return false;
    for (const n of el.childNodes) {
      if (n.nodeType === 3 && n.textContent.trim() !== '') return true;
    }
    return false;
  }

  const doc = document.documentElement;
  const interactivos = [];
  for (const el of document.querySelectorAll('button, a[href], input, select, textarea, [role=button]')) {
    if (!visible(el)) continue;
    const r = el.getBoundingClientRect();
    if (r.width < 44 || r.height < 44) {
      interactivos.push({ selector: ruta(el), w: Math.round(r.width), h: Math.round(r.height), texto: texto(el) });
    }
  }

  const fuentesMenores12 = [];
  const cuerpoMenores14 = [];
  for (const el of document.querySelectorAll('body *')) {
    if (!esHojaConTexto(el) || !visible(el)) continue;
    const fs = parseFloat(getComputedStyle(el).fontSize);
    const item = { selector: ruta(el), px: fs, texto: texto(el) };
    if (fs < 12) fuentesMenores12.push(item);
    if (fs < 14) cuerpoMenores14.push(item);
  }

  const tarjetaEp = document.querySelector('.rejilla .celda .tarjeta');
  const tarjetaHome = document.querySelector('.fila .carrusel .tarjeta');
  const rejilla = document.querySelector('.rejilla');
  const carrusel = document.querySelector('.fila .carrusel');

  return {
    scrollWidth: doc.scrollWidth,
    clientWidth: doc.clientWidth,
    overflowH: doc.scrollWidth - doc.clientWidth,
    interactivosPequenos: interactivos,
    fuentesMenores12,
    cuerpoMenores14,
    alturas: {
      tarjetaEpisodio: tarjetaEp ? Math.round(tarjetaEp.getBoundingClientRect().height * 100) / 100 : null,
      tarjetaHome: tarjetaHome ? Math.round(tarjetaHome.getBoundingClientRect().height * 100) / 100 : null,
      containIntrinsic: tarjetaEp ? getComputedStyle(tarjetaEp).containIntrinsicSize : null,
      columnasRejilla: rejilla ? getComputedStyle(rejilla).gridTemplateColumns.split(' ').filter(Boolean).length : null,
      rejillaPlantilla: rejilla ? getComputedStyle(rejilla).gridTemplateColumns : null,
      carruselDesborda: carrusel ? carrusel.scrollWidth > carrusel.clientWidth + 1 : null,
    },
  };
};

const browser = await chromium.launch();
const page = await browser.newPage();

setTimeout(() => {
  log('ABORTADO: tiempo máximo de 8 min alcanzado');
  process.exit(2);
}, 480000).unref?.();

async function gotoApp() {
  await page.goto(BASE, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('.buscador input', { timeout: 20000 });
}

/** Ejecuta un evaluate con tope de tiempo para no colgarse si la página recarga. */
function evaluarCon(fn, ms) {
  return Promise.race([
    page.evaluate(fn),
    new Promise((_, rej) => setTimeout(() => rej(new Error('evaluate timeout')), ms)),
  ]).catch(() => {});
}

/** Fuerza la carga de imágenes perezosas recorriendo el documento (acotado). */
async function forzarCargas() {
  log('forzarCargas: inicio');
  await evaluarCon(async () => {
    const total = document.documentElement.scrollHeight;
    const paso = window.innerHeight;
    const iter = Math.min(50, Math.ceil(total / paso) + 2);
    for (let i = 0; i <= iter; i++) {
      window.scrollTo(0, i * paso);
      await new Promise((r) => setTimeout(r, 60));
    }
    window.scrollTo(0, 0);
    await new Promise((r) => setTimeout(r, 60));
  }, 12000);
  await evaluarCon(
    () =>
      Promise.all(
        [...document.images].map((img) =>
          img.complete
            ? null
            : new Promise((r) => {
                img.addEventListener('load', r, { once: true });
                img.addEventListener('error', r, { once: true });
              })
        )
      ),
    8000
  );
  await page.evaluate(() => window.scrollTo(0, 0)).catch(() => {});
  await page.waitForTimeout(150);
  log('forzarCargas: fin');
}

async function snap(vista, ancho, fullPage = false) {
  const archivo = `${DIR_IMG}/${vista}_${ancho}.png`;
  log('captura:', vista, ancho, fullPage ? '(fullPage)' : '(viewport)');
  try {
    await Promise.race([
      page.screenshot({ path: archivo, fullPage }),
      new Promise((_, rej) => setTimeout(() => rej(new Error('screenshot timeout')), 30000)),
    ]);
    log('captura ok:', archivo);
  } catch (e) {
    errores.push(`captura ${archivo}: ${e.message}`);
  }
}

async function capturar(vista, ancho, fullPage = false) {
  await forzarCargas();
  await snap(vista, ancho, fullPage);
  try {
    metricas[`${vista}@${ancho}`] = await page.evaluate(MEDIR);
  } catch (e) {
    errores.push(`métricas ${vista}@${ancho}: ${e.message}`);
  }
}

async function esperarHome() {
  await page
    .waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 })
    .catch(() => errores.push('home: no aparecieron filas en 60 s'));
}

/** Vuelve pulsando "Atrás" hasta que reaparezca el buscador (máx. 4 saltos). */
async function volverABusqueda() {
  for (let i = 0; i < 4; i++) {
    if (await page.locator('.buscador input').isVisible().catch(() => false)) return;
    await page.locator('.cabecera .atras').first().click().catch(() => {});
    await page.waitForTimeout(700);
  }
}

/** Deja el buscador con resultados para `texto` (sin debounce). */
async function buscar(texto) {
  await volverABusqueda();
  const input = page.locator('.buscador input');
  await input.fill('');
  await input.fill(texto);
  await page.waitForSelector('main .rejilla .tarjeta', { timeout: 45000 });
  await page.waitForTimeout(600);
}

async function abrirDetalle(proveedor) {
  const tarjetas = page.locator('main .rejilla .tarjeta');
  const n = await tarjetas.count();
  for (let i = 0; i < n; i++) {
    const t = tarjetas.nth(i);
    const prov = ((await t.locator('.proveedor').first().textContent()) ?? '').trim();
    if (prov === proveedor) {
      await t.click();
      break;
    }
  }
  await page.waitForSelector('.ficha, .estado', { timeout: 45000 });
  await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 60000 }).catch(() => {});
  await page.waitForTimeout(800);
}

async function abrirReproductor() {
  await page.locator('.rejilla .celda .tarjeta').first().click();
  await page.waitForSelector('.marco', { timeout: 45000 });
  await page.waitForTimeout(1500);
}

for (const a of ANCHOS) {
  log(`==== ${a.w}x${a.h} ====`);
  await page.setViewportSize({ width: a.w, height: a.h });
  try {
    await gotoApp();
    await esperarHome();

    // 1. Descubrir (Home con datos de ambos proveedores)
    await capturar('descubrir', a.nombre, true);

    // 2. Favoritos vacío (solo en el primer ancho, antes de marcar ninguno)
    if (a.nombre === '1280') {
      await page.locator('.pestanas .pestana').nth(1).click();
      await page.waitForTimeout(600);
      await capturar('favoritos_vacio', a.nombre, true);
      await page.locator('.pestanas .pestana').nth(0).click();
      await page.waitForTimeout(400);
    }

    // 3. Búsqueda con resultados
    await buscar('bleach');
    await capturar('busqueda', a.nombre, true);

    // 4. Detalle AnimeAV1 (+ favorito en el primer ancho)
    await abrirDetalle('animeav1');
    await capturar('detalle_animeav1', a.nombre);
    if (a.nombre === '1280') {
      await page.locator('.estrella').first().click().catch(() => {});
      await page.waitForTimeout(500);
    }

    // 5. Reproductor con HLS activa + pliegue de "otras fuentes" abierto (T63)
    await abrirReproductor();
    // Fuerza la primera fuente principal (HLS) como activa antes de la captura.
    await page.locator('.selector .botones button').first().click().catch(() => {});
    await page.waitForTimeout(300);
    await capturar('reproductor_hls', a.nombre);
    const plegable = page.locator('.selector .plegable');
    if (await plegable.count()) {
      await plegable.first().click().catch(() => {});
      await page.waitForTimeout(400);
      await capturar('reproductor_extras', a.nombre);
    }

    // 6. Detalle JKAnime
    await buscar('bleach');
    await abrirDetalle('jkanime');
    await capturar('detalle_jkanime', a.nombre);

    // 7. Favoritos con datos (solo primer ancho)
    if (a.nombre === '1280') {
      await volverABusqueda();
      await page.locator('.buscador input').fill('');
      await page.waitForSelector('.pestanas .pestana', { timeout: 15000 });
      await page.locator('.pestanas .pestana').nth(1).click();
      await page.waitForTimeout(700);
      await capturar('favoritos_datos', a.nombre, true);
    }

    // 8. Barra lateral: expandida/contraída (ancho) o cajón (estrecho)
    if (a.w >= 900) {
      await page.locator('aside.barra .alternar').first().click();
      await page.waitForTimeout(500);
      await capturar('lateral_contraida', a.nombre);
      await page.locator('aside.barra .alternar').first().click();
      await page.waitForTimeout(500);
      await capturar('lateral_expandida', a.nombre);
    } else {
      await page.locator('.hamburguesa').first().click();
      await page.waitForTimeout(600);
      await capturar('lateral_cajon', a.nombre);
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    }
  } catch (e) {
    errores.push(`ancho ${a.nombre}: ${e.message}`);
    await snap(`error_${a.nombre}`, a.nombre).catch(() => {});
  }
}

writeFileSync(`${DIR_OUT}/metricas.json`, JSON.stringify({ metricas, errores }, null, 2));
log('errores:', errores.length ? errores : 'ninguno');
log('métricas escritas en', `${DIR_OUT}/metricas.json`);
await browser.close();
