// T88 — Auditoría de accesibilidad AA propia (v1.4.0).
//
// Herramienta de solo lectura sobre la app (no modifica código ni datos de
// forma permanente): recorre Home, búsqueda con resultados, detalle de los dos
// proveedores, reproductor y cajón a 360×740 y 1280×800 y comprueba:
//
//   - contraste de texto con colores computados y alfa compuesto (WCAG AA),
//   - nombres accesibles de todo interactivo,
//   - jerarquía de encabezados (exactamente una h1, sin saltos),
//   - roles/estados ARIA pedidos por la ficha (status, live, current, pressed,
//     busy),
//   - anillo de foco visible recorriendo con Tab,
//   - gestión del foco del cajón (Escape devuelve el foco y si atrapa),
//   - deriva de scroll del detalle T66/T67,
//   - geometría de regresión de la serie.
//
// Uso (Playwright desde el MCP global, igual que verificar_ui.mjs):
//   NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" \
//     node docs/investigacion/ui/a11y_auditoria.mjs antes
//   NODE_PATH=... node docs/investigacion/ui/a11y_auditoria.mjs despues
//
// Deja `a11y_<etiqueta>.json` y capturas en docs/img/v1.4/a11y/.

import { createRequire } from 'node:module';
import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const require = createRequire(import.meta.url);
let chromium;
try {
  ({ chromium } = require('playwright'));
} catch {
  ({ chromium } = require('playwright-core'));
}

const BASE = process.env.UI_URL ?? 'http://localhost:34115';
const ETIQUETA = process.argv[2] ?? 'auditoria';
const DIR = process.argv[3] ?? 'docs/img/v1.4/a11y';
mkdirSync(DIR, { recursive: true });

const ANCHOS = [
  { w: 360, h: 740, nombre: '360' },
  { w: 1280, h: 800, nombre: '1280' },
];

function log(...a) {
  process.stderr.write(['[a11y]', ...a].join(' ') + '\n');
}

// ---------------------------------------------------------------------------
// Navegación (mismos helpers que verificar_ui.mjs)
// ---------------------------------------------------------------------------

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
  const tarjetas = page.locator('main .rejilla > .tarjeta');
  const n = await tarjetas.count();
  for (let i = 0; i < n; i++) {
    const t = tarjetas.nth(i);
    const prov = ((await t.locator('.etiqueta').first().textContent().catch(() => '')) ?? '').trim();
    if (prov.toLowerCase() === proveedor.toLowerCase()) {
      await t.click();
      break;
    }
  }
  await page.waitForSelector('.ficha, .estado', { timeout: 45000 });
  await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 60000 }).catch(() => {});
  await page.waitForTimeout(700);
}

async function abrirReproductor(page) {
  const ep = page.locator('.rejilla .celda .tarjeta').first();
  if (!(await ep.count())) return false;
  await ep.click();
  await page.waitForSelector('.cabecera .atras', { timeout: 30000 }).catch(() => {});
  await page.waitForTimeout(1200);
  return true;
}

async function abrirCajon(page) {
  if (await page.locator('.barra.cajon').count()) return;
  await page.locator('.barra-inferior .destino', { hasText: 'Historial' }).click();
  await page.waitForSelector('.barra.cajon', { timeout: 15000 });
  await page.waitForTimeout(300);
}

async function cerrarCajon(page) {
  if (!(await page.locator('.barra.cajon').count())) return;
  await page.keyboard.press('Escape');
  await page.waitForSelector('.barra.cajon', { state: 'detached', timeout: 5000 }).catch(() => {});
  await page.waitForTimeout(250);
}

// ---------------------------------------------------------------------------
// Sembrado temporal del historial (para la fila Continuar y el cajón)
// ---------------------------------------------------------------------------

async function sembrarHistorial(page, cantidad = 4) {
  return page.evaluate(async (n) => {
    const App = window.go.main.App;
    const originales = await App.GetProgress('');
    await App.ClearProgress();
    let items = [];
    for (const prov of ['animeav1', 'jkanime']) {
      try {
        const secciones = await App.GetHomeSection(prov, false);
        for (const s of secciones ?? []) for (const a of s.items ?? []) items.push({ ...a, provider: a.provider || prov });
      } catch {
        /* sin Home disponible */
      }
    }
    if (items.length < n) {
      try {
        const res = await App.SearchAnime('bleach');
        for (const a of res ?? []) items.push(a);
      } catch {
        /* sin búsqueda disponible */
      }
    }
    const vistos = new Set();
    const elegidos = [];
    for (const a of items) {
      if (!a || !a.id) continue;
      const prov = a.provider || 'animeav1';
      const clave = prov + ':' + a.id;
      if (vistos.has(clave)) continue;
      vistos.add(clave);
      elegidos.push({ ...a, provider: prov });
      if (elegidos.length >= n) break;
    }
    const base = Math.floor(Date.now() / 1000) - elegidos.length * 60;
    for (let i = 0; i < elegidos.length; i++) {
      const a = elegidos[i];
      await App.SaveProgress({
        anime_id: a.id,
        episode_id: `${a.provider}:${a.id}:episode:1`,
        current_time: 1200,
        total_duration: 1440,
        is_watched: false,
        provider: a.provider,
        anime_title: a.title || a.id,
        cover_url: a.cover_url || '',
        episode_number: 1,
        updated_at: base + i,
      });
    }
    return originales;
  }, cantidad);
}

async function restaurarHistorial(page, originales) {
  if (!originales) return;
  await page.evaluate(async (orig) => {
    const App = window.go.main.App;
    await App.ClearProgress();
    for (const r of orig) await App.SaveProgress(r);
  }, originales);
}

// ---------------------------------------------------------------------------
// Sonda inyectada: contraste, nombres, encabezados y ARIA
// ---------------------------------------------------------------------------

const SONDA = () => {
  const opacidadDe = (el) => {
    let o = 1;
    for (let n = el; n; n = n.parentElement) {
      o *= Number(getComputedStyle(n).opacity) || 1;
      if (n === document.documentElement) break;
    }
    return o;
  };

  const visibles = (el) => {
    if (!el || !(el instanceof Element)) return false;
    const st = getComputedStyle(el);
    if (st.display === 'none' || st.visibility === 'hidden' || st.visibility === 'collapse') return false;
    // T65 revela las tarjetas al entrar en pantalla: mientras la opacidad
    // acumulada no es ~1 el texto aún no está visible para el usuario.
    if (opacidadDe(el) < 0.95) return false;
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  };

  const parseColor = (txt) => {
    const m = String(txt).match(/rgba?\(([^)]+)\)/i);
    if (!m) return null;
    const p = m[1].split(/[,\s/]+/).filter(Boolean).map(Number);
    if (p.length < 3 || p.some((v) => Number.isNaN(v))) return null;
    return { r: p[0], g: p[1], b: p[2], a: p.length > 3 ? p[3] : 1 };
  };

  const promedio = (cols) => {
    if (!cols.length) return null;
    const s = cols.reduce((acc, c) => ({ r: acc.r + c.r, g: acc.g + c.g, b: acc.b + c.b, a: acc.a + c.a }), { r: 0, g: 0, b: 0, a: 0 });
    return { r: s.r / cols.length, g: s.g / cols.length, b: s.b / cols.length, a: s.a / cols.length };
  };

  const fondoDe = (el) => {
    const capas = [];
    for (let n = el; n; n = n.parentElement) {
      const st = getComputedStyle(n);
      const bg = parseColor(st.backgroundColor);
      if (bg && bg.a > 0) capas.push(bg);
      const img = st.backgroundImage;
      if (img && img !== 'none') {
        const cols = [...img.matchAll(/rgba?\([^)]+\)/gi)].map((m) => parseColor(m[0])).filter(Boolean);
        const prom = promedio(cols);
        if (prom && prom.a > 0) capas.push(prom);
      }
      if (n === document.documentElement) break;
    }
    let r = 9, g = 13, b = 22;
    for (let i = capas.length - 1; i >= 0; i--) {
      const c = capas[i];
      r = c.r * c.a + r * (1 - c.a);
      g = c.g * c.a + g * (1 - c.a);
      b = c.b * c.a + b * (1 - c.a);
    }
    return { r, g, b };
  };

  const luminancia = (c) => {
    const f = (v) => {
      const s = v / 255;
      return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
    };
    return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
  };

  const ratio = (a, b) => {
    const la = luminancia(a), lb = luminancia(b);
    const hi = Math.max(la, lb), lo = Math.min(la, lb);
    return (hi + 0.05) / (lo + 0.05);
  };

  const selector = (el) => {
    let s = el.tagName.toLowerCase();
    if (el.id) s += '#' + el.id;
    if (typeof el.className === 'string' && el.className.trim()) s += '.' + el.className.trim().split(/\s+/).slice(0, 2).join('.');
    return s;
  };

  const textoDirecto = (el) =>
    [...el.childNodes]
      .filter((n) => n.nodeType === Node.TEXT_NODE)
      .map((n) => n.nodeValue ?? '')
      .join(' ')
      .replace(/\s+/g, ' ')
      .trim();

  // --- Encabezados ---
  const encabezados = [...document.querySelectorAll('h1,h2,h3,h4,h5,h6')]
    .filter((h) => visibles(h) && !h.closest('[aria-hidden="true"]'))
    .map((h) => {
      const st = getComputedStyle(h);
      const r = h.getBoundingClientRect();
      return {
        nivel: Number(h.tagName[1]),
        texto: (h.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 80),
        fontSize: st.fontSize,
        fontWeight: st.fontWeight,
        height: Math.round(r.height * 100) / 100,
      };
    });
  const h1 = encabezados.filter((h) => h.nivel === 1).length;
  const saltos = [];
  let anterior = 0;
  for (const h of encabezados) {
    if (anterior > 0 && h.nivel > anterior + 1) saltos.push(`${anterior}→${h.nivel} (${h.texto})`);
    anterior = h.nivel;
  }

  // --- Contraste de texto ---
  const fallosContraste = [];
  for (const el of document.querySelectorAll('body *')) {
    if (['SCRIPT', 'STYLE', 'NOSCRIPT', 'TITLE'].includes(el.tagName)) continue;
    if (el.closest('[aria-hidden="true"]')) continue;
    if (!visibles(el)) continue;
    const texto = textoDirecto(el);
    if (!texto) continue;
    const st = getComputedStyle(el);
    const fg = parseColor(st.color);
    if (!fg) continue;
    const opacidad = opacidadDe(el);
    const bg = fondoDe(el);
    const fgEf = {
      r: fg.r * opacidad + bg.r * (1 - opacidad),
      g: fg.g * opacidad + bg.g * (1 - opacidad),
      b: fg.b * opacidad + bg.b * (1 - opacidad),
      a: 1,
    };
    const fs = parseFloat(st.fontSize) || 16;
    const fw = parseInt(st.fontWeight, 10) || 400;
    const grande = fs >= 24 || (fs >= 18.66 && fw >= 700);
    const requerido = grande ? 3 : 4.5;
    const rel = ratio(fgEf, bg);
    if (rel + 0.01 < requerido) {
      fallosContraste.push({
        texto: texto.slice(0, 60),
        selector: selector(el),
        color: st.color,
        fondo: `rgb(${Math.round(bg.r)}, ${Math.round(bg.g)}, ${Math.round(bg.b)})`,
        ratio: Math.round(rel * 100) / 100,
        requerido,
        fontSize: st.fontSize,
        fontWeight: st.fontWeight,
      });
    }
  }

  // --- Nombres accesibles ---
  const interactivos = [
    ...document.querySelectorAll(
      'button, a[href], input:not([type="hidden"]), select, textarea, iframe, [role="button"], [role="tab"], [role="link"], [role="checkbox"], [role="switch"]'
    ),
  ].filter((el) => visibles(el) && !el.closest('[aria-hidden="true"]'));

  const nombreDe = (el) => {
    const aria = (el.getAttribute('aria-label') ?? '').trim();
    if (aria) return aria;
    const ref = el.getAttribute('aria-labelledby');
    if (ref) {
      const txt = ref
        .split(/\s+/)
        .map((id) => document.getElementById(id)?.textContent ?? '')
        .join(' ')
        .replace(/\s+/g, ' ')
        .trim();
      if (txt) return txt;
    }
    if (el.tagName === 'INPUT' || el.tagName === 'SELECT' || el.tagName === 'TEXTAREA') {
      const labels = el.labels ? [...el.labels].map((l) => l.textContent ?? '').join(' ') : '';
      if (labels.trim()) return labels.trim();
      const title = (el.getAttribute('title') ?? '').trim();
      if (title) return title;
      const ph = (el.getAttribute('placeholder') ?? '').trim();
      if (ph) return ph;
      if (el.tagName === 'INPUT' && ['submit', 'button', 'reset'].includes(el.type)) return (el.value ?? '').trim();
    }
    const txt = (el.textContent ?? '').replace(/\s+/g, ' ').trim();
    if (txt) return txt;
    const title = (el.getAttribute('title') ?? '').trim();
    if (title) return title;
    const alt = el.tagName === 'IMG' ? (el.getAttribute('alt') ?? '').trim() : '';
    if (alt) return alt;
    return '';
  };

  const sinNombre = interactivos
    .map((el) => ({ el, nombre: nombreDe(el) }))
    .filter((x) => !x.nombre)
    .map((x) => ({ tag: x.el.tagName.toLowerCase(), selector: selector(x.el), html: x.el.outerHTML.slice(0, 160) }));

  // --- ARIA / anuncios ---
  const aria = {
    toastStatus: !!document.querySelector('.toast[role="status"]'),
    toastLive: !!document.querySelector('.toast[aria-live]'),
    contadorLive: !!document.querySelector('.contador[aria-live]'),
    barraCurrent: !!document.querySelector('.barra-inferior [aria-current="page"]'),
    pressed: document.querySelectorAll('[aria-pressed]').length,
    busy: document.querySelectorAll('[aria-busy="true"]').length,
    busySelectores: [...document.querySelectorAll('[aria-busy="true"]')].map((el) => selector(el)).slice(0, 8),
    tablist: document.querySelectorAll('[role="tablist"]').length,
    tabpanel: document.querySelectorAll('[role="tabpanel"]').length,
  };

  // --- Geometría de regresión ---
  const caja = (sel) => {
    const el = document.querySelector(sel);
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { w: Math.round(r.width * 100) / 100, h: Math.round(r.height * 100) / 100 };
  };
  const geometria = {
    tarjetaHome: caja('.fila .carrusel > .tarjeta'),
    tarjetaEpisodio: caja('.rejilla .celda .tarjeta'),
    cabecera: caja('.cabecera'),
    cta: caja('.barra-accion'),
    pestana: caja('.pestana'),
    buscadorInput: caja('.buscador input'),
    overflowH: Math.max(0, (document.documentElement.scrollWidth ?? 0) - (document.documentElement.clientWidth ?? 0)),
  };

  return {
    url: location.href,
    encabezados,
    h1,
    saltos,
    fallosContraste,
    interactivos: interactivos.length,
    sinNombre,
    aria,
    geometria,
    idioma: document.documentElement.lang || null,
  };
};

// ---------------------------------------------------------------------------
// Anillo de foco recorriendo con Tab
// ---------------------------------------------------------------------------

async function recorrerFoco(page, max = 45) {
  const vistos = [];
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.mouse.click(2, 2);
  const claves = new Set();
  let cuerpos = 0;
  for (let i = 0; i < max; i++) {
    await page.keyboard.press('Tab');
    await page.waitForTimeout(120);
    const info = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el || el === document.body || el === document.documentElement) return null;
      // El anillo puede dibujarlo un ancestro (p. ej. `.campo` con
      // `:focus-within`), así que se mira hasta dos niveles hacia arriba.
      let pista = '';
      let n = el;
      for (let i = 0; n && n !== document.body && i < 3; i++, n = n.parentElement) {
        const st = getComputedStyle(n);
        pista += ` ${st.boxShadow} ${st.outlineColor} ${st.outlineStyle} ${st.outlineWidth}`;
      }
      const marca = /52,\s*211,\s*153/.test(pista);
      return {
        tag: el.tagName.toLowerCase(),
        clase: typeof el.className === 'string' ? el.className.split(/\s+/).slice(0, 2).join('.') : '',
        nombre: (el.getAttribute('aria-label') || el.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 60),
        anillo: marca,
        marca,
        pista: pista.trim().slice(0, 180),
      };
    });
    if (!info) {
      cuerpos += 1;
      if (cuerpos > 2) break;
      continue;
    }
    cuerpos = 0;
    const clave = `${info.tag}|${info.clase}|${info.nombre}`;
    if (claves.has(clave)) break;
    claves.add(clave);
    vistos.push(info);
  }
  const sinAnillo = vistos.filter((v) => !v.marca);
  return { total: vistos.length, sinAnillo, elementos: vistos.slice(0, 12) };
}

// ---------------------------------------------------------------------------
// Captura de una vista
// ---------------------------------------------------------------------------

const SELECTORES_HOVER = [
  '.btn',
  '.tarjeta',
  '.destino',
  '.pestana',
  '.chip',
  '.orden',
  '.atras',
  '.alternar',
  '.limpiar',
  '.accion',
  '.cerrar',
  '.flecha',
  '.borrar',
  '.papelera',
  '.borrar-todo',
  '.boton-siguiente',
  '.plegable',
  '.botones button',
];

async function auditarHover(page, vista, ancho) {
  const muestras = [];
  for (const sel of SELECTORES_HOVER) {
    const el = page.locator(sel).first();
    if (!(await el.count()) || !(await el.isVisible().catch(() => false))) continue;
    try {
      await el.hover({ force: true, timeout: 3000 });
      await page.waitForTimeout(40);
      const d = await page.evaluate(SONDA);
      if (d.fallosContraste.length) muestras.push({ selector: sel, fallos: d.fallosContraste });
    } catch {
      /* elemento no interactuable en esta vista */
    }
  }
  await page.mouse.move(1, 1).catch(() => {});
  await page.waitForTimeout(40);
  return muestras;
}

async function auditar(page, vista, ancho, rep) {
  const datos = await page.evaluate(SONDA);
  datos.vista = vista;
  datos.ancho = ancho;
  datos.foco = await recorrerFoco(page);
  datos.contrasteFoco = (await page.evaluate(SONDA)).fallosContraste;
  datos.contrasteHover = await auditarHover(page, vista, ancho);
  try {
    await page.screenshot({ path: join(DIR, `${vista}_${ancho}.png`) });
    datos.captura = `${vista}_${ancho}.png`;
  } catch (e) {
    datos.captura = null;
    rep.errores.push(`captura ${vista}@${ancho}: ${e.message}`);
  }
  rep.vistas.push(datos);
  log(vista, ancho, `h1=${datos.h1} saltos=${datos.saltos.length} contraste=${datos.fallosContraste.length} sinNombre=${datos.sinNombre.length}`);
  return datos;
}

// ---------------------------------------------------------------------------
// Deriva de scroll T66/T67
// ---------------------------------------------------------------------------

async function medirDerivaScroll(page) {
  const total = await page.locator('.rejilla .celda .tarjeta').count();
  if (!total) return { antes: null, despues: null, delta: null, motivo: 'sin episodios' };
  // Se elige una tarjeta intermedia y se deja que Playwright la traiga a la
  // vista ANTES de medir: así el punto de partida es el scroll real desde el
  // que se pulsa, no una posición que el propio click va a corregir.
  const ep = page.locator('.rejilla .celda .tarjeta').nth(Math.min(30, total - 1));
  await ep.scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  const antes = await page.evaluate(() => Math.round(window.scrollY));
  await ep.click();
  await page.waitForSelector('.cabecera .atras', { timeout: 30000 }).catch(() => {});
  await page.waitForTimeout(1500);
  await page.locator('.cabecera .atras').first().click().catch(() => {});
  await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);
  const despues = await page.evaluate(() => Math.round(window.scrollY));
  return { antes, despues, delta: despues - antes };
}

// ---------------------------------------------------------------------------
// Foco del cajón
// ---------------------------------------------------------------------------

async function auditarCajonFoco(page) {
  const rep = { focoInicial: null, dentro: false, escapeCierra: false, focoDevuelto: false, fugas: 0, tabulados: [] };
  rep.focoInicial = await page.evaluate(() => {
    const el = document.activeElement;
    return el ? { tag: el.tagName.toLowerCase(), label: (el.getAttribute('aria-label') || el.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 60) } : null;
  });
  rep.dentro = await page.evaluate(() => !!document.activeElement?.closest('.barra.cajon'));

  for (let i = 0; i < 12; i++) {
    await page.keyboard.press('Tab');
    await page.waitForTimeout(25);
    const info = await page.evaluate(() => {
      const el = document.activeElement;
      return {
        dentro: !!el?.closest('.barra.cajon'),
        tag: el?.tagName?.toLowerCase() ?? '',
        label: (el?.getAttribute?.('aria-label') || el?.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 40),
      };
    });
    rep.tabulados.push(info);
    if (!info.dentro) rep.fugas += 1;
  }

  await page.keyboard.press('Escape');
  await page.waitForTimeout(400);
  rep.escapeCierra = await page.evaluate(() => !document.querySelector('.barra.cajon'));
  rep.focoDevuelto = await page.evaluate(() => {
    const el = document.activeElement;
    return !!el?.closest('.barra-inferior') && /historial/i.test(el.textContent ?? '');
  });
  return rep;
}

// ---------------------------------------------------------------------------
// Toast real (borrar + Deshacer) para comprobar role/live
// ---------------------------------------------------------------------------

async function probarToast(page) {
  const tarjeta = page.locator('.barra.cajon .envoltorio').first();
  if (!(await tarjeta.count())) return { ok: false, motivo: 'sin historial' };
  await tarjeta.hover();
  await tarjeta.locator('.borrar').click();
  await page.waitForSelector('.toast', { timeout: 10000 }).catch(() => {});
  const toast = await page.evaluate(() => {
    const el = document.querySelector('.toast');
    if (!el) return null;
    return {
      role: el.getAttribute('role'),
      live: el.getAttribute('aria-live'),
      mensaje: (el.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 80),
      acciones: [...el.querySelectorAll('button')].map((b) => (b.getAttribute('aria-label') || b.textContent || '').trim()),
    };
  });
  const deshacer = page.locator('.toast .accion').first();
  if (await deshacer.count()) {
    await deshacer.click();
    await page.waitForTimeout(400);
  }
  return { ok: !!toast, toast };
}

// ---------------------------------------------------------------------------
// Estados de carga: `aria-busy` en tiempo de ejecución
// ---------------------------------------------------------------------------

async function auditarEstadosCarga(browser) {
  const rep = { esqueletoHome: null, lateralCargando: null, estadoVistaCargando: null, errores: [] };
  const contexto = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  // Se cuelgan las llamadas que pintan los estados de carga; el resto sigue
  // funcionando con el puente real, así se puede navegar a un detalle.
  // El puente se parchea en sitio (no se sustituye `window.go`): sustituirlo
  // hace que el runtime reasigne su propio proxy y el llamador entre en
  // recursión infinita.
  await contexto.addInitScript(() => {
    const colgar = { GetHomeSection: true, GetRecentProgress: true, GetAnimeDetails: true };
    const parchear = () => {
      const App = window.go?.main?.App;
      if (!App) return false;
      for (const nombre of Object.keys(colgar)) {
        if (typeof App[nombre] === 'function' && !App['__colgado_' + nombre]) {
          App['__colgado_' + nombre] = true;
          App[nombre] = () => new Promise(() => {});
        }
      }
      return true;
    };
    const timer = setInterval(() => {
      if (parchear()) clearInterval(timer);
    }, 30);
  });

  const page = await contexto.newPage();
  try {
    await gotoApp(page);
    await page.waitForTimeout(2500);
    rep.esqueletoHome = await page.evaluate(() => ({
      total: document.querySelectorAll('.fila-fantasma').length,
      conBusy: document.querySelectorAll('.fila-fantasma[aria-busy="true"]').length,
    }));
    rep.lateralCargando = await page.evaluate(() => {
      const cuerpo = document.querySelector('.barra .cuerpo');
      return cuerpo ? { existe: true, busy: cuerpo.getAttribute('aria-busy') } : { existe: false, busy: null };
    });

    // Detalle: la búsqueda sí funciona, `GetAnimeDetails` queda colgado.
    await page.locator('.buscador input').fill('bleach');
    await page.waitForSelector('main .rejilla .tarjeta', { timeout: 45000 });
    await page.locator('main .rejilla > .tarjeta').first().click();
    await page.waitForSelector('.estado', { timeout: 30000 }).catch(() => {});
    await page.waitForTimeout(1200);
    rep.estadoVistaCargando = await page.evaluate(() => {
      const el = document.querySelector('.estado-vista[aria-busy="true"]');
      return el
        ? { existe: true, texto: (el.textContent ?? '').replace(/\s+/g, ' ').trim().slice(0, 60), busy: el.getAttribute('aria-busy'), role: el.getAttribute('role') }
        : { existe: false };
    });
  } catch (e) {
    rep.errores.push(e.message);
  } finally {
    await contexto.close();
  }
  return rep;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function medirGeometrias(page) {
  const salida = {};
  for (const w of [360, 768, 1280]) {
    await page.setViewportSize({ width: w, height: 800 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(1200);
    salida[w] = {
      home: await page.evaluate(() => {
        const caja = (sel) => {
          const el = document.querySelector(sel);
          if (!el) return null;
          const r = el.getBoundingClientRect();
          return { w: Math.round(r.width * 100) / 100, h: Math.round(r.height * 100) / 100 };
        };
        return {
          tarjetaHome: caja('.fila .carrusel > .tarjeta'),
          pestana: caja('.pestana'),
          overflowH: Math.max(0, document.documentElement.scrollWidth - document.documentElement.clientWidth),
        };
      }),
    };
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    salida[w].detalle = await page.evaluate(() => {
      const caja = (sel) => {
        const el = document.querySelector(sel);
        if (!el) return null;
        const r = el.getBoundingClientRect();
        return { w: Math.round(r.width * 100) / 100, h: Math.round(r.height * 100) / 100 };
      };
      return {
        tarjetaEpisodio: caja('.rejilla .celda .tarjeta'),
        cabecera: caja('.cabecera'),
        cta: caja('.barra-accion'),
        overflowH: Math.max(0, document.documentElement.scrollWidth - document.documentElement.clientWidth),
      };
    });
  }
  return salida;
}

async function main() {
  const browser = await chromium.launch();
  if (ETIQUETA === 'carga') {
    const carga = await auditarEstadosCarga(browser);
    console.log(JSON.stringify(carga, null, 2));
    await browser.close();
    return;
  }
  const rep = {
    etiqueta: ETIQUETA,
    fecha: new Date().toISOString(),
    url: BASE,
    vistas: [],
    cajon: null,
    cajonTrasDetalle: null,
    geometrias: null,
    estadosCarga: null,
    derivaScroll: null,
    toast: null,
    errores: [],
  };

  const page = await browser.newPage();
  let historialOriginal = null;

  try {
    await page.setViewportSize({ width: 1280, height: 800 });
    await gotoApp(page);
    historialOriginal = await sembrarHistorial(page, 4);
    log('historial sembrado para la fila Continuar y el cajón');

    for (const a of ANCHOS) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await page.waitForTimeout(1800);
      await auditar(page, 'home', a.nombre, rep);

      await buscar(page, 'bleach');
      await auditar(page, 'busqueda', a.nombre, rep);

      await abrirDetalle(page, 'animeav1');
      await auditar(page, 'detalle_animeav1', a.nombre, rep);
      if (a.nombre === '360') {
        rep.derivaScroll = await medirDerivaScroll(page);
        log('deriva de scroll del detalle:', JSON.stringify(rep.derivaScroll));
      }

      await buscar(page, 'bleach');
      await abrirDetalle(page, 'jkanime');
      await auditar(page, 'detalle_jkanime', a.nombre, rep);

      const hayReproductor = await abrirReproductor(page);
      if (hayReproductor) await auditar(page, 'reproductor', a.nombre, rep);

      if (a.nombre === '360') {
        if (hayReproductor) {
          await page.locator('.cabecera .atras').first().click().catch(() => {});
          await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 45000 }).catch(() => {});
          await page.waitForTimeout(2500);
          // Sonda aparte: abrir el cajón justo al volver del reproductor deja
          // que la restauración de scroll (T67) compita con el foco de T73.
          await abrirCajon(page);
          rep.cajonTrasDetalle = await auditarCajonFoco(page);
        }
        await page.locator('.barra-inferior .destino', { hasText: 'Descubrir' }).click();
        await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 45000 }).catch(() => {});
        await page.waitForTimeout(700);
        await abrirCajon(page);
        rep.cajon = await auditarCajonFoco(page);
        await abrirCajon(page);
        await auditar(page, 'cajon', a.nombre, rep);
        await cerrarCajon(page);
        await abrirCajon(page);
        rep.toast = await probarToast(page);
      }
    }

    rep.geometrias = await medirGeometrias(page);
    rep.estadosCarga = await auditarEstadosCarga(browser);
    log('estados de carga:', JSON.stringify(rep.estadosCarga));
  } catch (e) {
    rep.errores.push(e.message);
  } finally {
    try {
      await restaurarHistorial(page, historialOriginal);
    } catch (e) {
      rep.errores.push(`restaurar historial: ${e.message}`);
    }
    await browser.close();
  }

  // Resumen
  rep.resumen = {
    vistas: rep.vistas.length,
    fallosContraste: rep.vistas.reduce((n, v) => n + v.fallosContraste.length, 0),
    fallosContrasteFoco: rep.vistas.reduce((n, v) => n + v.contrasteFoco.length, 0),
    fallosContrasteHover: rep.vistas.reduce(
      (n, v) => n + v.contrasteHover.reduce((m, h) => m + h.fallos.length, 0),
      0
    ),
    sinNombre: rep.vistas.reduce((n, v) => n + v.sinNombre.length, 0),
    saltos: rep.vistas.reduce((n, v) => n + v.saltos.length, 0),
    vistasSinH1: rep.vistas.filter((v) => v.h1 !== 1).map((v) => `${v.vista}@${v.ancho}`),
    focoSinAnillo: rep.vistas.reduce((n, v) => n + v.foco.sinAnillo.length, 0),
    idiomas: [...new Set(rep.vistas.map((v) => v.idioma))],
  };

  writeFileSync(join(DIR, `a11y_${ETIQUETA}.json`), JSON.stringify(rep, null, 2));
  console.log(JSON.stringify(rep.resumen, null, 2));
  log('informe:', join(DIR, `a11y_${ETIQUETA}.json`));
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
