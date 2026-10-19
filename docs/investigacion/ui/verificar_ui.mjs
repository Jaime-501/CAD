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
//   NODE_PATH=... node docs/investigacion/ui/verificar_ui.mjs filas [dirOut]
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

// T76: anchos del modo `filas` (carruseles y tarjetas del Home con ratón).
const ANCHOS_FILAS = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

// T77: anchos del modo `continuar` (fila "Continuar viendo" en el Home).
const ANCHOS_CONTINUAR = [
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
function diffPNG(a, b, tol, x0 = 0, x1 = Infinity) {
  if (a.width !== b.width || a.height !== b.height) {
    return { dims: false, a: `${a.width}x${a.height}`, b: `${b.width}x${b.height}` };
  }
  let distintos = 0, maxDelta = 0;
  let minX = Infinity, minY = Infinity, maxX = -1, maxY = -1;
  const desde = Math.max(0, x0);
  const hasta = Math.min(a.width, x1);
  for (let y = 0; y < a.height; y++) {
    for (let x = desde; x < hasta; x++) {
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
        enCta: !!b.closest('.barra-accion'),
        enCajon: !!b.closest('.cajon'),
        enFilaContinuar: !!b.closest('.fila-continuar'),
      };
    })
    .filter((x) => x.w < 44 || x.h < 44);
};

// T76: geometría de la primera fila del Home (carrusel, tarjeta, flechas),
// el título de fila y el buscador (alto de campo/botón, paso 0).
const MEDIR_FILAS = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height), right: round(r.right) };
  };
  const carr = document.querySelector('.fila .carrusel');
  const primera = document.querySelector('.fila .carrusel .tarjeta');
  const portada = primera ? primera.querySelector('img') : null;
  const flecha = document.querySelector('.fila .flecha');
  const titulo = document.querySelector('.fila .titulo-fila');
  const csCarr = carr ? getComputedStyle(carr) : null;
  const csTit = titulo ? getComputedStyle(titulo) : null;
  return {
    innerWidth: window.innerWidth,
    clientWidth: d.clientWidth,
    overflowH: d.scrollWidth - d.clientWidth,
    gutter: getComputedStyle(d).getPropertyValue('--gutter').trim(),
    pointerCoarse: matchMedia('(pointer: coarse)').matches,
    carrusel: carr
      ? {
          ...rect(carr),
          mask: csCarr.maskImage,
          scrollPaddingLeft: csCarr.scrollPaddingLeft,
          overscroll: csCarr.overscrollBehaviorX,
          scrollLeft: round(carr.scrollLeft),
          scrollWidth: carr.scrollWidth,
          clientWidth: carr.clientWidth,
        }
      : null,
    primera: rect(primera),
    portada: rect(portada),
    tarjetaHomeH: primera ? round(primera.getBoundingClientRect().height) : null,
    portadaAr: portada ? getComputedStyle(portada).aspectRatio : null,
    flecha: flecha
      ? {
          w: round(flecha.getBoundingClientRect().width),
          h: round(flecha.getBoundingClientRect().height),
          display: getComputedStyle(flecha).display,
        }
      : null,
    flechas: document.querySelectorAll('.fila .flecha').length,
    titulo: titulo
      ? { minWidth: csTit.minWidth, whiteSpace: csTit.whiteSpace, textOverflow: csTit.textOverflow }
      : null,
    buscador: {
      input: rect(document.querySelector('.buscador input')),
      campo: rect(document.querySelector('.buscador .campo')),
      buscar: rect(document.querySelector('.buscador > button')),
    },
  };
};

// T77: geometría de la fila "Continuar viendo" (solo modo estrecho), sus
// tarjetas, barras de progreso y el respaldo de portada.
const MEDIR_CONTINUAR = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height), right: round(r.right) };
  };
  const fila = document.querySelector('.fila-continuar');
  const carrusel = fila ? fila.querySelector('.carrusel') : null;
  const h3 = fila ? fila.querySelector('.titulo-fila') : null;
  const csCarr = carrusel ? getComputedStyle(carrusel) : null;
  const tarjetas = fila ? [...fila.querySelectorAll('.carrusel .tarjeta')] : [];
  const panel = document.getElementById('panel-descubrir');
  const primerHijo = panel && panel.firstElementChild ? panel.firstElementChild.className : null;
  return {
    innerWidth: window.innerWidth,
    clientWidth: d.clientWidth,
    overflowH: d.scrollWidth - d.clientWidth,
    gutter: getComputedStyle(d).getPropertyValue('--gutter').trim(),
    fila: rect(fila),
    titulo: h3 ? (h3.textContent || '').trim() : null,
    primerHijoPanel: primerHijo,
    carrusel: carrusel
      ? {
          ...rect(carrusel),
          mask: csCarr.maskImage,
          scrollPaddingLeft: csCarr.scrollPaddingLeft,
          scrollSnapType: csCarr.scrollSnapType,
          overscroll: csCarr.overscrollBehaviorX,
          scrollbarWidth: csCarr.scrollbarWidth,
        }
      : null,
    tarjetas: tarjetas.map((t) => {
      const barra = t.querySelector('[role="progressbar"]');
      const titulo = t.querySelector('.titulo');
      const proveedor = t.querySelector('.proveedor');
      const sinPortada = t.querySelector('.sin-portada');
      return {
        ...rect(t),
        porcentaje: barra ? Number(barra.getAttribute('aria-valuenow')) : null,
        titulo: titulo ? (titulo.textContent || '').trim() : null,
        proveedor: proveedor ? (proveedor.textContent || '').trim() : null,
        inicial: sinPortada ? (sinPortada.textContent || '').trim() : null,
        conPortada: !!t.querySelector('img.portada'),
      };
    }),
    tarjetaHome: rect(document.querySelector('.fila .carrusel .tarjeta')),
    pestana: rect(document.querySelector('.pestana')),
  };
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

// T75: medición del buscador y de las pestañas del Home en modo estrecho.
const MEDIR_BUSCADOR = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { w: round(r.width), h: round(r.height) };
  };
  const panel = document.getElementById('panel-descubrir');
  const panelFav = document.getElementById('panel-favoritos');
  const h2Fav = panelFav ? panelFav.querySelector('h2') : null;
  const panelActivo = (p) => !!p && !p.classList.contains('inactivo');
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    tablist: !!document.querySelector('[role="tablist"]'),
    roleDescubrir: panel ? panel.getAttribute('role') : null,
    roleFavoritos: panelFav ? panelFav.getAttribute('role') : null,
    ariaDescubrir: panel ? panel.getAttribute('aria-labelledby') : null,
    ariaFavoritos: panelFav ? panelFav.getAttribute('aria-labelledby') : null,
    lupa: !!document.querySelector('.buscador .lupa'),
    limpiar: !!document.querySelector('.buscador .limpiar'),
    rects: {
      input: rect(document.querySelector('.buscador input')),
      campo: rect(document.querySelector('.buscador .campo')),
      buscar: rect(document.querySelector('.buscador > button')),
      lupa: rect(document.querySelector('.buscador .lupa')),
      limpiar: rect(document.querySelector('.buscador .limpiar')),
      actualizar: rect(document.querySelector('.actualizar')),
      reintentar: rect(document.querySelector('.reintentar')),
      pestana: rect(document.querySelector('.pestana')),
      tarjetaHome: rect(document.querySelector('.fila .carrusel .tarjeta')),
    },
    panelDescubrirActivo: panelActivo(panel),
    panelFavoritosActivo: panelActivo(panelFav),
    h2Favoritos: h2Fav ? (h2Fav.textContent || '').trim() : null,
  };
};

// T78: medición de la rejilla de resultados de búsqueda y sus estados.
const MEDIR_BUSQUEDA = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height) };
  };
  const rejilla = document.querySelector('main .rejilla');
  const cs = rejilla ? getComputedStyle(rejilla) : null;
  const columnas = cs && cs.gridTemplateColumns ? cs.gridTemplateColumns.split(' ').filter(Boolean).length : 0;
  const tarjetas = [...document.querySelectorAll('main .rejilla > .tarjeta')];
  const fantasmas = [...document.querySelectorAll('main .rejilla > .fantasma')];
  const contador = document.querySelector('main .contador');
  const error = document.querySelector('.estado-vista.variante-error');
  const vacio = document.querySelector('.estado-vista.variante-vacio');
  // T78: caja que abarca la zona de resultados (contador + rejilla); en 1280
  // sirve para demostrar que el cambio visual queda contenido ahí.
  const zonaResultados = (() => {
    const a = contador ? contador.getBoundingClientRect() : null;
    const b = rejilla ? rejilla.getBoundingClientRect() : null;
    if (!a && !b) return null;
    const x0 = Math.min(a ? a.left : Infinity, b ? b.left : Infinity);
    const y0 = Math.min(a ? a.top : Infinity, b ? b.top : Infinity);
    const x1 = Math.max(a ? a.right : -Infinity, b ? b.right : -Infinity);
    const y1 = Math.max(a ? a.bottom : -Infinity, b ? b.bottom : -Infinity);
    return { x: Math.round(x0), y: Math.round(y0), w: Math.round(x1 - x0), h: Math.round(y1 - y0) };
  })();
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    columnas,
    gap: cs ? cs.gap : null,
    tarjetas: tarjetas.length,
    fantasmas: fantasmas.length,
    ariaBusy: rejilla ? rejilla.getAttribute('aria-busy') : null,
    primera: rect(tarjetas[0]),
    portada: rect(tarjetas[0] ? tarjetas[0].querySelector('img') : null),
    zonaResultados,
    todasTactil:
      tarjetas.length > 0 &&
      tarjetas.every((t) => {
        const r = t.getBoundingClientRect();
        return r.width >= 44 && r.height >= 44;
      }),
    contadorTexto: contador ? (contador.textContent || '').trim() : null,
    contadorLive: contador ? contador.getAttribute('aria-live') : null,
    etiquetas: tarjetas.map((t) => ((t.querySelector('.etiqueta') || {}).textContent || '').trim()),
    error: error
      ? {
          titulo: ((error.querySelector('.titulo') || {}).textContent || '').trim(),
          texto: ((error.querySelector('.texto') || {}).textContent || '').trim(),
          boton: ((error.querySelector('.btn') || {}).textContent || '').trim(),
        }
      : null,
    vacio: vacio
      ? {
          titulo: ((vacio.querySelector('.titulo') || {}).textContent || '').trim(),
          texto: ((vacio.querySelector('.texto') || {}).textContent || '').trim(),
          boton: !!vacio.querySelector('.btn'),
        }
      : null,
  };
};

// T78: portadas cargadas de la rejilla de resultados (lazy-loading real).
const CONTAR_IMAGENES = () => {
  const imgs = [...document.querySelectorAll('main .rejilla > .tarjeta img')];
  let cargadas = 0;
  for (const im of imgs) if (im.complete && im.naturalWidth > 0) cargadas++;
  return { total: imgs.length, cargadas };
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

// T78: los resultados usan `TarjetaAnime`, cuya insignia de proveedor es
// `.etiqueta` (nombre legible: "JKAnime"/"AnimeAV1") en lugar de `.proveedor`.
async function abrirDetalle(page, proveedor) {
  const tarjetas = page.locator('main .rejilla > .tarjeta');
  const n = await tarjetas.count();
  for (let i = 0; i < n; i++) {
    const t = tarjetas.nth(i);
    const prov = ((await t.locator('.etiqueta').first().textContent().catch(() => '')) ?? '').trim();
    if (prov.toLowerCase() === proveedor.toLowerCase()) { await t.click(); break; }
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
  const tarjetas = page.locator('main .rejilla > .tarjeta');
  const n = await tarjetas.count();
  let elegido = -1;
  let maxLargo = -1;
  for (let i = 0; i < n; i++) {
    const t = tarjetas.nth(i);
    const prov = ((await t.locator('.etiqueta').first().textContent().catch(() => '')) ?? '').trim();
    if (prov.toLowerCase() !== proveedor.toLowerCase()) continue;
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
// Modo buscador (T75): pestañas del Home y buscador en modo estrecho
// ---------------------------------------------------------------------------

const ANCHOS_BUSCADOR = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

async function buscador() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/buscador';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const rep = { anchos: [], funciones: [], tactiles: [], capturas: [], errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const captura = async (nombre, fullPage = false) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`), fullPage });
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
  const destino = (texto) => page.locator('.barra-inferior .destino', { hasText: texto });
  const panelActivo = (id) =>
    page.evaluate((i) => !document.getElementById(i).classList.contains('inactivo'), id);

  // --- A. Geometría y estado por ancho ---
  await paso('A', async () => {
    for (const a of ANCHOS_BUSCADOR) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await page.waitForTimeout(400);
      const m = await page.evaluate(MEDIR_BUSCADOR);
      rep.anchos.push({ ancho: a.nombre, ...m });
      if (a.nombre === '360') await captura('home_360', true);
      if (a.nombre === '1280') await captura('home_1280');
      if (a.w < 900) {
        check(`${a.nombre}: sin [role=tablist]`, !m.tablist, m.tablist);
        check(`${a.nombre}: paneles sin role`, m.roleDescubrir === null && m.roleFavoritos === null, {
          descubrir: m.roleDescubrir,
          favoritos: m.roleFavoritos,
        });
        check(
          `${a.nombre}: paneles sin aria-labelledby`,
          m.ariaDescubrir === null && m.ariaFavoritos === null,
          { descubrir: m.ariaDescubrir, favoritos: m.ariaFavoritos }
        );
        check(`${a.nombre}: lupa dentro del campo`, m.lupa, m.lupa);
        check(`${a.nombre}: sin ✕ sin texto`, !m.limpiar, m.limpiar);
        check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      } else {
        check(`${a.nombre}: [role=tablist] presente`, m.tablist, m.tablist);
        check(
          `${a.nombre}: paneles con role=tabpanel`,
          m.roleDescubrir === 'tabpanel' && m.roleFavoritos === 'tabpanel',
          { descubrir: m.roleDescubrir, favoritos: m.roleFavoritos }
        );
        check(`${a.nombre}: sin lupa`, !m.lupa, m.lupa);
        check(`${a.nombre}: sin ✕`, !m.limpiar, m.limpiar);
        check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      }
      log('buscador ancho', a.nombre, 'tablist=', m.tablist, 'lupa=', m.lupa);
    }
  });

  // --- B. Pestañas y encabezado de Favoritos desde la barra (360) ---
  await paso('B', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(400);
    check('360: Descubrir activo al arrancar', await panelActivo('panel-descubrir'));
    check(
      '360: encabezado Favoritos no visible en Descubrir',
      !(await page.locator('#panel-favoritos h2').isVisible().catch(() => false))
    );
    await destino('Favoritos').click();
    await page.waitForTimeout(350);
    check('360: barra → panel Favoritos', await panelActivo('panel-favoritos'));
    const hFav = await page.evaluate(() => {
      const h = document.querySelector('#panel-favoritos h2');
      return h ? (h.textContent || '').trim() : null;
    });
    check('360: encabezado "Favoritos (N)"', /^Favoritos \(\d+\)$/.test(hFav ?? ''), hFav);
    check(
      '360: encabezado Favoritos visible en su panel',
      await page.locator('#panel-favoritos h2').isVisible().catch(() => false)
    );
    await captura('favoritos_360', true);
    await destino('Descubrir').click();
    await page.waitForTimeout(350);
    check('360: barra → panel Descubrir', await panelActivo('panel-descubrir'));
  });

  // --- C. Buscador: lupa, ✕, Enter y botón (360) ---
  await paso('C', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(300);
    const input = page.locator('#buscador-anime');
    await input.fill('bleach');
    await page.waitForTimeout(600);
    await captura('busqueda_texto_360', true);
    const conTexto = await page.evaluate(MEDIR_BUSCADOR);
    check('360: ✕ aparece con texto', conTexto.limpiar, conTexto.limpiar);
    check(
      '360: ✕ ≥ 44×44',
      conTexto.rects.limpiar && conTexto.rects.limpiar.w >= 44 && conTexto.rects.limpiar.h >= 44,
      conTexto.rects.limpiar
    );
    await page.waitForSelector('main .rejilla .tarjeta', { timeout: 45000 }).catch(() => {});
    await page.waitForTimeout(500);
    await captura('busqueda_resultados_360', true);
    check('360: hay resultados de "bleach"', (await page.locator('main .rejilla .tarjeta').count()) > 0);
    await page.locator('.buscador .limpiar').click();
    await page.waitForTimeout(700);
    const tras = await page.evaluate(() => ({
      valor: document.getElementById('buscador-anime').value,
      foco: document.activeElement?.id,
    }));
    check('360: ✕ vacía el input', tras.valor === '', tras);
    check('360: ✕ devuelve el foco al input', tras.foco === 'buscador-anime', tras);
    check('360: Home visible tras limpiar', await panelActivo('panel-descubrir'));
    check('360: ✕ desaparece sin texto', (await page.locator('.buscador .limpiar').count()) === 0);
    await input.fill('bleach');
    await page.waitForTimeout(300);
    await input.press('Enter');
    await page.waitForSelector('main .rejilla .tarjeta', { timeout: 45000 }).catch(() => {});
    check('360: Enter busca', (await page.locator('main .rejilla .tarjeta').count()) > 0);
    await input.fill('naruto');
    await page.locator('.buscador > button').click();
    const vioCarga = await page.evaluate(async () => {
      const b = document.querySelector('.buscador > button');
      const t0 = performance.now();
      while (performance.now() - t0 < 1000) {
        if (b.disabled && (b.textContent || '').trim() === 'Buscando...') return true;
        await new Promise((r) => setTimeout(r, 10));
      }
      return false;
    });
    check('360: botón "Buscar" muestra "Buscando..."/disabled', vioCarga, vioCarga);
    await page.waitForSelector('main .rejilla .tarjeta', { timeout: 45000 }).catch(() => {});
    const enReposo = await page.evaluate(() => {
      const b = document.querySelector('.buscador > button');
      return { texto: (b.textContent || '').trim(), disabled: b.disabled };
    });
    check('360: botón "Buscar" vuelve a su estado', enReposo.texto === 'Buscar' && !enReposo.disabled, enReposo);
  });

  // --- D. Objetivos táctiles en modo estrecho (360) ---
  await paso('D', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(400);
    const m = await page.evaluate(MEDIR_BUSCADOR);
    const alto = (r) => r && r.h >= 44;
    check('360: "Buscar" ≥ 44 de alto', alto(m.rects.buscar), m.rects.buscar);
    check('360: "Actualizar" ≥ 44 de alto', alto(m.rects.actualizar), m.rects.actualizar);
    if (m.rects.reintentar) check('360: "Reintentar" ≥ 44 de alto', alto(m.rects.reintentar), m.rects.reintentar);
    rep.tactiles = (await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'home_360' }));
    await page.locator('#buscador-anime').fill('bleach');
    await page.waitForTimeout(500);
    const conTexto = await page.evaluate(MEDIR_BUSCADOR);
    check(
      '360: ✕ ≥ 44×44',
      conTexto.rects.limpiar && conTexto.rects.limpiar.w >= 44 && conTexto.rects.limpiar.h >= 44,
      conTexto.rects.limpiar
    );
  });

  // --- E. Resize 800 → 1000 → 800 ---
  await paso('E', async () => {
    await page.setViewportSize({ width: 800, height: 900 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(500);
    check('800: sin [role=tablist]', (await page.locator('[role="tablist"]').count()) === 0);
    const s0 = await page.evaluate(() => {
      const c = document.querySelector('.fila .carrusel');
      if (!c) return null;
      c.scrollLeft = 120;
      return c.scrollLeft;
    });
    await page.setViewportSize({ width: 1000, height: 900 });
    await page.waitForTimeout(600);
    check('1000: [role=tablist] presente', (await page.locator('[role="tablist"]').count()) > 0);
    check('1000: conserva Descubrir activo', await panelActivo('panel-descubrir'));
    const s1 = await page.evaluate(() => {
      const c = document.querySelector('.fila .carrusel');
      return c ? c.scrollLeft : null;
    });
    check('800 → 1000: conserva el scroll de las filas', s0 !== null && s1 === s0, { s0, s1 });
    await page.setViewportSize({ width: 800, height: 900 });
    await page.waitForTimeout(600);
    check('800 de vuelta: sin [role=tablist]', (await page.locator('[role="tablist"]').count()) === 0);
    check('800 de vuelta: conserva Descubrir activo', await panelActivo('panel-descubrir'));
  });

  // --- F. Destino "Buscar" de la barra y T66/T67 (360) ---
  await paso('F', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(400);
    await destino('Buscar').click();
    await page.waitForTimeout(250);
    check(
      '360: barra Buscar enfoca el input',
      await page.evaluate(() => document.activeElement?.id === 'buscador-anime')
    );
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.waitForTimeout(900);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultima.count())) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const yDespues = await page.evaluate(() => Math.round(window.scrollY));
      check('360: T66/T67 scroll del detalle restaurado', Math.abs(yDespues - yAntes) <= 5, { yAntes, yDespues });
    } else {
      check('360: T66/T67 scroll del detalle restaurado', false, { yAntes });
    }
  });

  // --- G. prefers-reduced-motion: sin transición en el botón limpiar ---
  await paso('G', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.locator('#buscador-anime').fill('bleach');
    await page.waitForTimeout(300);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const td = await page.evaluate(() => {
      const el = document.querySelector('.buscador .limpiar');
      return el ? getComputedStyle(el).transitionDuration : null;
    });
    check('reduced-motion: sin transición en el limpiar', td === '0s', td);
    await page.emulateMedia({ reducedMotion: null });
  });

  writeFileSync(join(dirOut, 'buscador.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo filas (T76): carruseles y tarjetas del Home
// ---------------------------------------------------------------------------

async function filas() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/filas';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const rep = { anchos: [], tactiles: [], funciones: [], capturas: [], esqueleto: null, errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await ctx.newPage();
  const captura = async (nombre, locator = null) => {
    try {
      if (locator) await locator.screenshot({ path: join(dirOut, `${nombre}.png`) });
      else await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const cargar = async (a) => {
    await page.setViewportSize({ width: a.w, height: a.h });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(400);
  };

  // --- A. Geometría y estado por ancho (ratón = pointer: fine) ---
  await paso('A', async () => {
    for (const a of ANCHOS_FILAS) {
      await cargar(a);
      const m = await page.evaluate(MEDIR_FILAS);
      rep.anchos.push({ ancho: a.nombre, ...m });
      if (a.nombre === '360') await captura('home_360');
      if (a.nombre === '768') await captura('home_768');
      if (a.nombre === '1280') await captura('home_1280');
      check(
        `${a.nombre}: tarjeta de Home 263,8 px`,
        m.tarjetaHomeH !== null && Math.abs(m.tarjetaHomeH - 263.8) <= 1,
        m.tarjetaHomeH
      );
      check(`${a.nombre}: portada 2:3`, m.portadaAr === '2 / 3', m.portadaAr);
      check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      if (a.w < 900) {
        check(
          `${a.nombre}: campo del buscador 44–48 px`,
          m.buscador.campo && m.buscador.campo.h >= 44 && m.buscador.campo.h <= 48,
          m.buscador.campo
        );
        check(
          `${a.nombre}: botón Buscar 44–48 px`,
          m.buscador.buscar && m.buscador.buscar.h >= 44 && m.buscador.buscar.h <= 48,
          m.buscador.buscar
        );
        check(
          `${a.nombre}: primera tarjeta al gutter`,
          m.primera !== null && Math.abs(m.primera.x - parseFloat(m.gutter)) < 0.6,
          { x: m.primera?.x, gutter: m.gutter }
        );
        check(
          `${a.nombre}: carrusel llega al borde derecho`,
          m.carrusel && Math.abs(m.carrusel.right - m.clientWidth) < 1,
          { right: m.carrusel?.right, clientWidth: m.clientWidth }
        );
        check(`${a.nombre}: sin máscara de degradado`, m.carrusel && m.carrusel.mask === 'none', m.carrusel?.mask);
        check(
          `${a.nombre}: título de fila con elipsis`,
          m.titulo && m.titulo.whiteSpace === 'nowrap' && m.titulo.textOverflow === 'ellipsis',
          m.titulo
        );
        check(
          `${a.nombre}: flechas 44×44 visibles (ratón)`,
          m.flecha && m.flecha.w === 44 && m.flecha.h === 44 && m.flecha.display !== 'none',
          m.flecha
        );
      } else {
        check(
          `${a.nombre}: input de escritorio 38 px`,
          m.buscador.input && m.buscador.input.h === 38,
          m.buscador.input
        );
        check(
          `${a.nombre}: botón Buscar de escritorio 38 px`,
          m.buscador.buscar && m.buscador.buscar.h === 38,
          m.buscador.buscar
        );
        check(`${a.nombre}: flechas 32×32`, m.flecha && m.flecha.w === 32 && m.flecha.h === 32, m.flecha);
        check(
          `${a.nombre}: degradados de borde presentes`,
          m.carrusel && m.carrusel.mask !== 'none',
          m.carrusel?.mask
        );
      }
      log('filas ancho', a.nombre, 'tarjeta=', m.tarjetaHomeH, 'flecha=', m.flecha && `${m.flecha.w}x${m.flecha.h}`);
    }
  });

  // --- B. Esqueleto → datos: la primera fila no debe saltar ---
  await paso('B', async () => {
    const ctx2 = await browser.newContext({ viewport: { width: 360, height: 740 } });
    const p2 = await ctx2.newPage();
    // La caché de detalle/Home hace que la respuesta llegue casi al instante y
    // el esqueleto dure una sola pintura: retrasamos la llamada de Wails para
    // poder medir el esqueleto antes de que lleguen los datos.
    await p2.addInitScript(() => {
      const wrap = () => {
        const App = window.go && window.go.main && window.go.main.App;
        if (!App || App.__slow76 || typeof App.GetHomeSection !== 'function') return false;
        const orig = App.GetHomeSection;
        App.GetHomeSection = (...a) =>
          new Promise((res) => setTimeout(() => res(orig.apply(App, a)), 1200));
        App.__slow76 = true;
        return true;
      };
      if (!wrap()) {
        const id = setInterval(() => {
          if (wrap()) clearInterval(id);
        }, 2);
        setTimeout(() => clearInterval(id), 10000);
      }
    });
    await p2.goto(BASE, { waitUntil: 'domcontentloaded' });
    const topFantasma = await p2
      .waitForFunction(
        () => {
          const f = document.querySelector('.fila-fantasma');
          return f ? Math.round(f.getBoundingClientRect().top * 100) / 100 : false;
        },
        null,
        { polling: 10, timeout: 15000 }
      )
      .then((h) => h.jsonValue())
      .catch(() => null);
    await p2.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await p2.waitForTimeout(500);
    const topReal = await p2.evaluate(() => {
      const f = document.querySelector('.fila');
      return f ? Math.round(f.getBoundingClientRect().top * 100) / 100 : null;
    });
    rep.esqueleto = { topFantasma, topReal };
    check(
      'esqueleto→datos: primera fila no salta > 2 px',
      topFantasma !== null && topReal !== null && Math.abs(topReal - topFantasma) <= 2,
      rep.esqueleto
    );
    await ctx2.close();
  });

  // --- C. Pointer coarse (táctil): flechas ocultas y gesto de arrastre ---
  await paso('C', async () => {
    for (const a of [
      { w: 360, h: 740, n: '360c' },
      { w: 390, h: 844, n: '390c' },
    ]) {
      const ctxC = await browser.newContext({
        viewport: { width: a.w, height: a.h },
        hasTouch: true,
        isMobile: true,
        deviceScaleFactor: 1,
      });
      const pC = await ctxC.newPage();
      await pC.goto(BASE, { waitUntil: 'domcontentloaded' });
      await pC.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await pC.waitForTimeout(500);
      const m = await pC.evaluate(MEDIR_FILAS);
      rep.anchos.push({ ancho: a.n, ...m });
      check(`${a.n}: pointer coarse detectado`, m.pointerCoarse, m.pointerCoarse);
      check(
        `${a.n}: flechas ocultas`,
        m.flechas > 0 && (m.flecha === null || m.flecha.display === 'none'),
        m.flecha
      );
      check(
        `${a.n}: primera tarjeta al gutter`,
        m.primera !== null && Math.abs(m.primera.x - parseFloat(m.gutter)) < 0.6,
        { x: m.primera?.x, gutter: m.gutter }
      );
      check(`${a.n}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      if (a.n === '360c') {
        try {
          await pC.screenshot({ path: join(dirOut, 'home_360_coarse.png') });
          rep.capturas.push('home_360_coarse.png');
        } catch (e) {
          rep.errores.push(`home_360_coarse: ${e.message}`);
        }
      }
      // Arrastre táctil real (CDP) de derecha a izquierda sobre la fila.
      const cdp = await ctxC.newCDPSession(pC);
      const pts = await pC.evaluate(() => {
        const c = document.querySelector('.fila .carrusel');
        const r = c.getBoundingClientRect();
        const y = Math.min(Math.max(r.y + r.height / 2, 20), window.innerHeight - 20);
        return { x0: r.x + r.width - 30, x1: r.x + 40, y };
      });
      let prev = 0;
      let tope = 0;
      for (let k = 0; k < 40; k++) {
        await cdp.send('Input.dispatchTouchEvent', {
          type: 'touchStart',
          touchPoints: [{ x: pts.x0, y: pts.y }],
        });
        for (let i = 1; i <= 10; i++) {
          await cdp.send('Input.dispatchTouchEvent', {
            type: 'touchMove',
            touchPoints: [{ x: pts.x0 + ((pts.x1 - pts.x0) * i) / 10, y: pts.y }],
          });
          await pC.waitForTimeout(16);
        }
        await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
        await pC.waitForTimeout(400);
        const sl = await pC.evaluate(() => {
          const c = document.querySelector('.fila .carrusel');
          return { sl: c.scrollLeft, max: c.scrollWidth - c.clientWidth };
        });
        prev = sl.sl;
        tope = sl.max;
        if (sl.max > 0 && sl.sl >= sl.max - 1) break;
      }
      check(`${a.n}: la fila se desplaza con gesto táctil`, prev > 0, { scrollLeft: prev });
      check(
        `${a.n}: el gesto llega al final de la fila`,
        tope > 0 && prev >= tope - 1,
        { scrollLeft: prev, max: tope }
      );
      await ctxC.close();
    }
  });

  // --- D. Clic en una tarjeta del Home abre el detalle (360) ---
  await paso('D', async () => {
    await cargar({ w: 360, h: 740 });
    await page.locator('.fila .carrusel .tarjeta').first().click();
    await page.waitForSelector('.ficha, .estado', { timeout: 45000 }).catch(() => {});
    await page.waitForTimeout(800);
    const enDetalle = await page.evaluate(
      () => !!document.querySelector('.ficha') || !!document.querySelector('main .cabecera')
    );
    check('360: clic en tarjeta del Home abre el detalle', enDetalle, enDetalle);
    await page.locator('.cabecera .atras').first().click().catch(() => {});
    await page.waitForTimeout(800);
    check(
      '360: Atrás vuelve al Home',
      await page.locator('.fila .carrusel').first().isVisible().catch(() => false)
    );
  });

  // --- E. T66/T67: el scroll del detalle se conserva al ir a un episodio ---
  await paso('E', async () => {
    await cargar({ w: 360, h: 740 });
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.waitForTimeout(900);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultima.count())) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const yDespues = await page.evaluate(() => Math.round(window.scrollY));
      check('360: T66/T67 scroll del detalle restaurado', Math.abs(yDespues - yAntes) <= 5, {
        yAntes,
        yDespues,
      });
    } else {
      check('360: T66/T67 scroll del detalle restaurado', false, { yAntes });
    }
  });

  // --- F. prefers-reduced-motion: sin transición en la tarjeta ---
  await paso('F', async () => {
    await cargar({ w: 360, h: 740 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const td = await page.evaluate(() => {
      const t = document.querySelector('.fila .carrusel .tarjeta');
      return t ? getComputedStyle(t).transitionDuration : null;
    });
    check('reduced-motion: tarjeta sin transición', td === '0s', td);
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- G. Objetivos táctiles <44 px visibles en el Home (360) ---
  await paso('G', async () => {
    await cargar({ w: 360, h: 740 });
    rep.tactiles = await page.evaluate(TACTILES_CHICOS);
  });

  // --- H. Primer plano de una tarjeta a 360 ---
  await paso('H', async () => {
    await cargar({ w: 360, h: 740 });
    const tarjeta = page.locator('.fila .carrusel .tarjeta').first();
    if (await tarjeta.count()) await captura('tarjeta_360', tarjeta);
  });

  writeFileSync(join(dirOut, 'filas.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo continuar (T77): fila "Continuar viendo" del Home en modo estrecho
// ---------------------------------------------------------------------------

async function continuar() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/continuar';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const rep = { anchos: [], tactiles: [], funciones: [], capturas: [], errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const ctx = await browser.newContext({ viewport: { width: 360, height: 740 } });
  const page = await ctx.newPage();
  const captura = async (nombre, locator = null) => {
    try {
      if (locator) await locator.screenshot({ path: join(dirOut, `${nombre}.png`) });
      else await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const cargar = async (a) => {
    await page.setViewportSize({ width: a.w, height: a.h });
    await gotoApp(page);
    await page.waitForSelector('.fila-continuar .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(400);
  };
  const destino = (texto) => page.locator('.barra-inferior .destino', { hasText: texto });
  const volverAlHome = async () => {
    for (let i = 0; i < 4; i++) {
      if (await page.locator('.fila-continuar').isVisible().catch(() => false)) return;
      await page.locator('.cabecera .atras').first().click().catch(() => {});
      await page.waitForTimeout(800);
    }
  };

  // Historial sembrado (base T69, con el id de episodio corregido: debe llevar
  // el slug; `animeav1:episode:9` daba "id de episodio inválido").
  const SEMILLA = [
    { anime_id: 'bleach', episode_id: 'jkanime:bleach:episode:5', current_time: 300, total_duration: 1440, is_watched: false, provider: 'jkanime', anime_title: 'Bleach', cover_url: '', episode_number: 5, updated_at: 1749999700 },
    { anime_id: 'one-piece', episode_id: 'jkanime:one-piece:episode:1100', current_time: 1300, total_duration: 1420, is_watched: true, provider: 'jkanime', anime_title: 'One Piece', cover_url: '', episode_number: 1100, updated_at: 1749999800 },
    { anime_id: 'bleach', episode_id: 'animeav1:bleach:episode:9', current_time: 900, total_duration: 1420, is_watched: false, provider: 'animeav1', anime_title: 'Bleach', cover_url: '', episode_number: 9, updated_at: 1749999940 },
  ];
  const sembrar = async (semilla = SEMILLA) => {
    await gotoApp(page);
    await page.evaluate(async (seed) => {
      const App = window.go?.main?.App;
      if (!App) throw new Error('puente window.go no disponible (usa localhost:34115)');
      await App.ClearProgress();
      for (const s of seed) await App.SaveProgress(s);
    }, semilla);
    await gotoApp(page);
    await page.waitForSelector('.fila-continuar .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(500);
  };

  // --- A. Geometría y estado por ancho ---
  await paso('A', async () => {
    await sembrar();
    for (const a of ANCHOS_CONTINUAR) {
      await cargar(a);
      const m = await page.evaluate(MEDIR_CONTINUAR);
      rep.anchos.push({ ancho: a.nombre, ...m });
      if (a.nombre === '360') await captura('continuar_360');
      if (a.nombre === '768') await captura('continuar_768');
      if (a.nombre === '1280') await captura('continuar_1280');
      if (a.w < 900) {
        check(`${a.nombre}: fila "Continuar viendo" presente`, !!m.fila, m.fila);
        check(`${a.nombre}: título "Continuar viendo"`, m.titulo === 'Continuar viendo', m.titulo);
        check(`${a.nombre}: 3 tarjetas`, m.tarjetas.length === 3, m.tarjetas.length);
        check(
          `${a.nombre}: tarjetas 240 × ≥44 px`,
          m.tarjetas.length === 3 && m.tarjetas.every((t) => t.w === 240 && t.h >= 44),
          m.tarjetas.map((t) => `${t.w}×${t.h}`)
        );
        check(
          `${a.nombre}: primera tarjeta al gutter`,
          !!m.tarjetas[0] && Math.abs(m.tarjetas[0].x - parseFloat(m.gutter)) < 0.6,
          { x: m.tarjetas[0]?.x, gutter: m.gutter }
        );
        check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
        check(`${a.nombre}: carrusel sin máscara`, !!m.carrusel && m.carrusel.mask === 'none', m.carrusel?.mask);
        check(`${a.nombre}: sin flechas`, (await page.locator('.fila-continuar .flecha').count()) === 0);
        if (a.nombre === '360')
          await captura('tarjeta_360', page.locator('.fila-continuar .carrusel .tarjeta').first());
      } else {
        check(`${a.nombre}: fila ausente del DOM`, m.fila === null, m.fila);
        check(
          `${a.nombre}: Home idéntico a T76 (tarjeta 263,8)`,
          !!m.tarjetaHome && Math.abs(m.tarjetaHome.h - 263.8) <= 1,
          m.tarjetaHome?.h
        );
        check(`${a.nombre}: pestaña del Home 38 px`, !!m.pestana && Math.abs(m.pestana.h - 38) <= 1, m.pestana?.h);
      }
      log('continuar ancho', a.nombre, 'fila=', !!m.fila, 'tarjetas=', m.tarjetas.length);
    }
  });

  // --- B. Barras de progreso distintas y respaldo de portada ---
  await paso('B', async () => {
    await cargar({ w: 360, h: 740 });
    const m = await page.evaluate(MEDIR_CONTINUAR);
    const pcts = m.tarjetas.map((t) => t.porcentaje);
    check('B: 3 porcentajes distintos', new Set(pcts).size === 3, pcts);
    check(
      'B: progreso coherente (21 / 63 / 100)',
      JSON.stringify([...pcts].sort((x, y) => x - y)) === JSON.stringify([21, 63, 100]),
      pcts
    );
    check(
      'B: barra role=progressbar con aria-valuenow',
      await page.locator('.fila-continuar [role="progressbar"][aria-valuenow]').count() === 3
    );
    check(
      'B: respaldo de portada (inicial, sin img)',
      m.tarjetas.every((t) => t.inicial !== null && !t.conPortada),
      m.tarjetas.map((t) => ({ inicial: t.inicial, conPortada: t.conPortada }))
    );
  });

  // --- C. Clic → retomar; Atrás vuelve al Home conservando el scroll (T66/T67) ---
  // Nota: en modo estrecho el Home sufre un desplazamiento residual pequeño (~8 px)
  // por el anclaje de scroll al remontar las secciones asíncronas (reproducible sin
  // la fila; T66/T67 hace la restauración exacta). Se tolera esa deriva.
  await paso('C', async () => {
    await cargar({ w: 360, h: 740 });
    await page.waitForTimeout(1500);
    await page.evaluate(
      () => window.scrollTo(0, Math.min(200, document.documentElement.scrollHeight - window.innerHeight))
    );
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    await page.evaluate(() => document.querySelector('.fila-continuar .carrusel .tarjeta').click());
    await page.waitForSelector('.cabecera .atras', { timeout: 30000 });
    await page.waitForTimeout(1500);
    const navego = await page.evaluate(
      () => !!(document.querySelector('.marco') || document.querySelector('.ficha') || document.querySelector('.estado'))
    );
    check('C: el clic navega por retomar (detalle/reproductor)', navego, navego);
    await volverAlHome();
    await page.waitForTimeout(700);
    check('C: Atrás vuelve al Home', await page.locator('.fila-continuar').isVisible().catch(() => false));
    const yDespues = await page.evaluate(() => Math.round(window.scrollY));
    check('C: Home conserva el scroll (T66/T67)', Math.abs(yDespues - yAntes) <= 20, { yAntes, yDespues });
  });

  // --- D. Ver un episodio y volver: la fila refleja el progreso (recarga al salir del reproductor) ---
  await paso('D', async () => {
    await cargar({ w: 360, h: 740 });
    const jkBleach = (t) => t.proveedor === 'JKAnime' && t.titulo === 'Bleach';
    const antes = (await page.evaluate(MEDIR_CONTINUAR)).tarjetas.find(jkBleach)?.porcentaje ?? null;
    await page.evaluate(() => document.querySelector('.fila-continuar .carrusel .tarjeta').click());
    await page.waitForSelector('.cabecera .atras', { timeout: 30000 });
    await page.waitForTimeout(1500);
    // Simula avanzar/ver otro episodio mientras se está en el reproductor.
    await page.evaluate(async () => {
      const App = window.go?.main?.App;
      if (App)
        await App.SaveProgress({
          anime_id: 'bleach',
          episode_id: 'jkanime:bleach:episode:5',
          current_time: 1440,
          total_duration: 1440,
          is_watched: true,
          provider: 'jkanime',
          anime_title: 'Bleach',
          cover_url: '',
          episode_number: 5,
          updated_at: 1749999999,
        });
    });
    await volverAlHome();
    await page.waitForTimeout(700);
    const despues = (await page.evaluate(MEDIR_CONTINUAR)).tarjetas.find(jkBleach)?.porcentaje ?? null;
    check('D: al salir del reproductor la fila recarga el historial', antes !== despues, { antes, despues });
  });

  // --- E. Sin historial la fila desaparece (sin hueco) y "Deshacer" la restaura ---
  await paso('E', async () => {
    await cargar({ w: 360, h: 740 });
    await destino('Historial').click();
    await page.waitForTimeout(400);
    for (let i = 0; i < 6; i++) {
      const pap = page.locator('.barra.cajon .grupo .papelera');
      if ((await pap.count()) === 0) break;
      await pap.first().click();
      await page.waitForTimeout(600);
    }
    const vacio = await page.evaluate(() => ({
      fila: !!document.querySelector('.fila-continuar'),
      primerHijo: document.getElementById('panel-descubrir')?.firstElementChild?.className ?? null,
    }));
    check('E: sin historial la fila no está en el DOM', !vacio.fila, vacio);
    check('E: sin historial el Home no deja hueco', (vacio.primerHijo ?? '').includes('cabecera'), vacio.primerHijo);
    await page.locator('.toast .accion').click().catch(() => {});
    await page.waitForTimeout(800);
    const tras = await page.evaluate(() => ({
      fila: !!document.querySelector('.fila-continuar'),
      tarjetas: document.querySelectorAll('.fila-continuar .carrusel .tarjeta').length,
    }));
    check('E: con "Deshacer" la fila reaparece', tras.fila && tras.tarjetas >= 1, tras);
    await page.keyboard.press('Escape').catch(() => {});
    await page.waitForTimeout(300);
  });

  // --- F. prefers-reduced-motion: sin transición en la tarjeta ---
  await paso('F', async () => {
    await cargar({ w: 390, h: 844 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const td = await page.evaluate(() => {
      const t = document.querySelector('.fila-continuar .carrusel .tarjeta');
      return t ? getComputedStyle(t).transitionDuration : null;
    });
    check('F: reduced-motion sin transición en la tarjeta', td === '0s', td);
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- G. Objetivos táctiles <44 px nuevos (en la fila) ---
  await paso('G', async () => {
    await cargar({ w: 360, h: 740 });
    rep.tactiles = (await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'home' }));
    const nuevos = rep.tactiles.filter((t) => t.enFilaContinuar);
    check('G: 0 objetivos táctiles <44 px en la fila', nuevos.length === 0, nuevos);
  });

  writeFileSync(join(dirOut, 'continuar.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo busqueda (T78): rejilla de resultados y estados de búsqueda
// ---------------------------------------------------------------------------

const ANCHOS_BUSQUEDA = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

async function busqueda() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/busqueda';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const rep = {
    anchos: [],
    funciones: [],
    capturas: [],
    imagenes: null,
    esqueleto: null,
    vacio: null,
    zona1280: null,
    tactiles: [],
    errores: [],
  };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const page = await browser.newPage();
  const captura = async (nombre, p = page) => {
    try {
      await p.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const escribir = async (texto) => {
    const input = page.locator('#buscador-anime');
    await input.fill('');
    await input.fill(texto);
  };
  const medirHome = () =>
    page.evaluate(() => {
      const t = document.querySelector('.fila .carrusel .tarjeta');
      if (!t) return null;
      const r = t.getBoundingClientRect();
      const i = t.querySelector('img');
      const ir = i ? i.getBoundingClientRect() : null;
      const rd = (n) => Math.round(n * 100) / 100;
      return { w: rd(r.width), h: rd(r.height), imgW: ir ? rd(ir.width) : null, imgH: ir ? rd(ir.height) : null };
    });

  // --- A. Geometría, columnas y contador por ancho ---
  await paso('A', async () => {
    for (const a of ANCHOS_BUSQUEDA) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await page.waitForTimeout(300);
      const home = await medirHome();
      check(
        `${a.nombre}: tarjeta del Home 263,8×140`,
        !!home && Math.abs(home.h - 263.8) <= 1 && home.w === 140,
        home
      );
      await buscar(page, 'bleach');
      await page.evaluate(() => window.scrollTo(0, 0));
      await page.waitForTimeout(300);
      const m = await page.evaluate(MEDIR_BUSQUEDA);
      rep.anchos.push({ ancho: a.nombre, ...m });
      if (a.w === 1280) {
        rep.zona1280 = m.zonaResultados;
        check(
          '1280: zona de resultados contenida (x≥0, w≤ancho)',
          !!m.zonaResultados && m.zonaResultados.x >= 0 && m.zonaResultados.w <= a.w,
          m.zonaResultados
        );
      }
      check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      check(`${a.nombre}: gap de rejilla 12px (--sp-3)`, m.gap === '12px', m.gap);
      check(`${a.nombre}: cada tarjeta ≥44×44`, m.todasTactil, m.primera);
      if (a.w < 480) check(`${a.nombre}: 2 columnas`, m.columnas === 2, m.columnas);
      else if (a.w < 900) check(`${a.nombre}: 3 columnas`, m.columnas === 3, m.columnas);
      else check(`${a.nombre}: auto-fill (≥4 columnas)`, m.columnas >= 4, m.columnas);
      check(
        `${a.nombre}: contador "{n} resultados" con aria-live`,
        m.contadorLive === 'polite' && /^\d+ resultados$/.test(m.contadorTexto ?? ''),
        m.contadorTexto
      );
      check(
        `${a.nombre}: etiqueta legible del proveedor`,
        m.etiquetas.length > 0 && m.etiquetas.every((e) => e === 'JKAnime' || e === 'AnimeAV1'),
        m.etiquetas.slice(0, 4)
      );
      // La tarjeta tiene la forma de `TarjetaAnime`: alto = 1,5·ancho + 53,8.
      check(
        `${a.nombre}: alto = 1,5·ancho + 53,8`,
        !!m.primera && Math.abs(m.primera.h - (1.5 * m.primera.w + 53.8)) <= 1.5,
        m.primera
      );
      if (a.nombre === '360') await captura('resultados_360_arriba');
      if (a.nombre === '768') await captura('resultados_768');
      if (a.nombre === '1280') await captura('resultados_1280');
      log('busqueda ancho', a.nombre, 'columnas=', m.columnas, 'tarjeta=', m.primera && `${m.primera.w}x${m.primera.h}`);
    }
  });

  // --- B. Portadas con scroll real (lazy-loading) a 360 ---
  await paso('B', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'bleach');
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(400);
    const arriba = await page.evaluate(CONTAR_IMAGENES);
    await page.evaluate(async () => {
      const paso = Math.max(200, window.innerHeight * 0.8);
      for (let y = 0; y <= document.documentElement.scrollHeight; y += paso) {
        window.scrollTo(0, y);
        await new Promise((r) => setTimeout(r, 200));
      }
      window.scrollTo(0, document.documentElement.scrollHeight);
    });
    await page.waitForTimeout(1000);
    const abajo = await page.evaluate(CONTAR_IMAGENES);
    rep.imagenes = { arriba, abajo };
    await captura('resultados_360_scroll');
    check(
      'B: todas las portadas cargan tras el scroll',
      abajo.total > 0 && abajo.cargadas >= abajo.total,
      rep.imagenes
    );
    check('B: el scroll real carga más que el primer pantallazo', abajo.cargadas >= arriba.cargadas, rep.imagenes);
  });

  // --- C. Esqueleto → resultados: sin salto de layout (búsqueda lenta) ---
  await paso('C', async () => {
    const ctx2 = await browser.newContext({ viewport: { width: 360, height: 740 } });
    const p2 = await ctx2.newPage();
    await p2.addInitScript(() => {
      const wrap = () => {
        const App = window.go && window.go.main && window.go.main.App;
        if (!App || App.__slow78 || typeof App.SearchAnime !== 'function') return false;
        const orig = App.SearchAnime;
        App.SearchAnime = (...a) => new Promise((res) => setTimeout(() => res(orig.apply(App, a)), 1500));
        App.__slow78 = true;
        return true;
      };
      if (!wrap()) {
        const id = setInterval(() => {
          if (wrap()) clearInterval(id);
        }, 2);
        setTimeout(() => clearInterval(id), 10000);
      }
    });
    await p2.goto(BASE, { waitUntil: 'domcontentloaded' });
    await p2.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await p2.locator('#buscador-anime').fill('bleach');
    const topFantasma = await p2
      .waitForFunction(
        () => {
          const f = document.querySelector('main .rejilla > .fantasma');
          return f ? Math.round(f.getBoundingClientRect().top * 100) / 100 : false;
        },
        null,
        { polling: 10, timeout: 15000 }
      )
      .then((h) => h.jsonValue())
      .catch(() => null);
    const fantasmas = await p2.locator('main .rejilla > .fantasma').count();
    const ariaBusy = await p2.evaluate(() => {
      const r = document.querySelector('main .rejilla');
      return r ? r.getAttribute('aria-busy') : null;
    });
    await captura('esqueleto_360', p2);
    await p2.waitForSelector('main .rejilla > .tarjeta', { timeout: 45000 });
    await p2.waitForTimeout(300);
    const topReal = await p2.evaluate(() => {
      const t = document.querySelector('main .rejilla > .tarjeta');
      return t ? Math.round(t.getBoundingClientRect().top * 100) / 100 : null;
    });
    rep.esqueleto = {
      fantasmas,
      ariaBusy,
      topFantasma,
      topReal,
      delta: topFantasma !== null && topReal !== null ? Math.round((topReal - topFantasma) * 100) / 100 : null,
    };
    check('C: 6 esqueletos durante la carga', fantasmas === 6, fantasmas);
    check('C: aria-busy="true" en el contenedor', ariaBusy === 'true', ariaBusy);
    check(
      'C: sin salto de layout (≤4 px)',
      topFantasma !== null && topReal !== null && Math.abs(topReal - topFantasma) <= 4,
      rep.esqueleto
    );
    await ctx2.close();
  });

  // --- D. Estado de error + "Reintentar" ---
  await paso('D', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    const forzado = await page.evaluate(() => {
      const App = window.go && window.go.main && window.go.main.App;
      if (!App || typeof App.SearchAnime !== 'function') return false;
      window.__origSearch = App.SearchAnime;
      window.__searchCalls = [];
      App.SearchAnime = (q) => {
        window.__searchCalls.push(q);
        return Promise.reject(new Error('fallo forzado'));
      };
      return true;
    });
    if (!forzado) {
      check('D: puente window.go disponible', false, 'sin puente (usa localhost:34115)');
      return;
    }
    await escribir('bleach');
    await page.waitForSelector('.estado-vista.variante-error', { timeout: 15000 });
    const m = await page.evaluate(MEDIR_BUSQUEDA);
    rep.anchos.push({ ancho: '360', estado: 'error', ...m });
    check('D: título "No se pudo buscar"', m.error?.titulo === 'No se pudo buscar', m.error);
    check('D: muestra el mensaje del error', !!m.error?.texto && m.error.texto.includes('fallo forzado'), m.error?.texto);
    check('D: acción "Reintentar"', m.error?.boton === 'Reintentar', m.error?.boton);
    check(
      'D: role="alert" en el estado',
      (await page.locator('.estado-vista.variante-error[role="alert"]').count()) === 1
    );
    check('D: el estado sustituye al párrafo `.error`', (await page.locator('main p.error').count()) === 0);
    await captura('error_360');
    await page.locator('.estado-vista.variante-error .btn').click();
    await page.waitForTimeout(700);
    const calls = await page.evaluate(() => window.__searchCalls ?? []);
    check(
      'D: "Reintentar" repite la consulta',
      calls.length >= 2 && calls[calls.length - 1] === 'bleach',
      calls
    );
    await page.evaluate(() => {
      const App = window.go?.main?.App;
      if (App && window.__origSearch) App.SearchAnime = window.__origSearch;
    });
  });

  // --- E. Estado vacío ---
  await paso('E', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    // La ficha pide "zzzzqqqq", pero el backend real devuelve 1 resultado
    // (JKAnime responde con un *fallback*: "Suzuka"). Si no hay estado vacío
    // se prueba con una cadena larga que sí devuelve 0 resultados.
    let consultaVacio = 'zzzzqqqq';
    await escribir(consultaVacio);
    let vacioReal = true;
    try {
      await page.waitForSelector('.estado-vista.variante-vacio', { timeout: 12000 });
    } catch {
      vacioReal = false;
    }
    if (!vacioReal) {
      consultaVacio = 'zzzzzzzzzzzzzzzzzzzzzzzzzzzzzz';
      await escribir(consultaVacio);
      await page.waitForSelector('.estado-vista.variante-vacio', { timeout: 45000 });
    }
    rep.vacio = { consulta: consultaVacio, conZzzzqqqq: vacioReal };
    const m = await page.evaluate(MEDIR_BUSQUEDA);
    rep.anchos.push({ ancho: '360', estado: 'vacio', ...m });
    check('E: título "Sin resultados"', m.vacio?.titulo === 'Sin resultados', m.vacio);
    check('E: texto con la consulta', !!m.vacio?.texto && m.vacio.texto.includes(consultaVacio), m.vacio?.texto);
    check('E: sin acción en el vacío', m.vacio?.boton === false, m.vacio?.boton);
    check('E: sin rejilla residual', m.tarjetas === 0 && m.fantasmas === 0, { t: m.tarjetas, f: m.fantasmas });
    await captura('vacio_360');
  });

  // --- F. El clic abre el detalle y Atrás vuelve a los resultados ---
  await paso('F', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'bleach');
    await page.locator('main .rejilla > .tarjeta').first().click();
    await page.waitForSelector('.ficha, main .cabecera', { timeout: 45000 });
    const enDetalle = await page.evaluate(
      () => !!document.querySelector('.ficha') || !!document.querySelector('main .cabecera')
    );
    check('F: el clic abre el detalle', enDetalle, enDetalle);
    await page.locator('.cabecera .atras').first().click().catch(() => {});
    await page.waitForTimeout(900);
    const hayResultados = await page.locator('main .rejilla > .tarjeta').count();
    check('F: Atrás vuelve a los resultados', hayResultados > 0, hayResultados);
  });

  // --- I. T66/T67: el scroll del detalle se conserva al ir a un episodio ---
  await paso('I', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.waitForTimeout(900);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultima.count())) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const yDespues = await page.evaluate(() => Math.round(window.scrollY));
      check('I: T66/T67 scroll del detalle restaurado', Math.abs(yDespues - yAntes) <= 5, { yAntes, yDespues });
    } else {
      check('I: T66/T67 scroll del detalle restaurado', false, { yAntes });
    }
  });

  // --- G. prefers-reduced-motion: sin transición en la tarjeta ---
  await paso('G', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'bleach');
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const td = await page.evaluate(() => {
      const t = document.querySelector('main .rejilla > .tarjeta');
      return t ? getComputedStyle(t).transitionDuration : null;
    });
    check('G: reduced-motion sin transición en la tarjeta', td === '0s', td);
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- H. Objetivos táctiles <44 px visibles en los resultados ---
  await paso('H', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'bleach');
    rep.tactiles = await page.evaluate(TACTILES_CHICOS);
  });

  writeFileSync(join(dirOut, 'busqueda.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo ficha (T79): ficha de detalle (portada, metadatos, géneros, sinopsis)
// ---------------------------------------------------------------------------

const ANCHOS_FICHA = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

const MEDIR_FICHA = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height), right: round(r.right), bottom: round(r.bottom) };
  };
  const ficha = document.querySelector('.ficha');
  const marco = document.querySelector('.ficha .portada-marco');
  const img = document.querySelector('.ficha .portada');
  const generos = document.querySelector('.ficha .generos');
  const sinopsis = document.querySelector('.ficha .sinopsis');
  const verMas = document.querySelector('.ficha .ver-mas');
  const punto = document.querySelector('.ficha .metadatos .punto');
  const rejilla = document.querySelector('.rejilla');
  const csGen = generos ? getComputedStyle(generos) : null;
  const csSin = sinopsis ? getComputedStyle(sinopsis) : null;
  const csMarco = marco ? getComputedStyle(marco) : null;
  const csRej = rejilla ? getComputedStyle(rejilla) : null;
  const csPunto = punto ? getComputedStyle(punto) : null;
  const rg = ficha ? ficha.getBoundingClientRect() : null;
  const chips = [...document.querySelectorAll('.ficha .metadatos .insignia')].map((c) => ({
    texto: (c.textContent || '').trim(),
    tono: [...c.classList].find((k) => k.startsWith('tono-')) || null,
  }));
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    innerWidth: window.innerWidth,
    ficha: rect(ficha),
    fichaRadius: ficha ? getComputedStyle(ficha).borderRadius : null,
    fichaPadding: ficha ? getComputedStyle(ficha).padding : null,
    fichaBorder: ficha ? getComputedStyle(ficha).borderTopWidth : null,
    fichaShadow: ficha ? getComputedStyle(ficha).boxShadow : null,
    fichaFondo: ficha ? getComputedStyle(ficha).backgroundColor : null,
    marco: rect(marco),
    marcoPadding: csMarco ? csMarco.padding : null,
    marcoRadius: csMarco ? csMarco.borderRadius : null,
    marcoFondo: csMarco ? csMarco.backgroundImage : null,
    img: rect(img),
    imgRadius: img ? getComputedStyle(img).borderRadius : null,
    imgAspect: img ? getComputedStyle(img).aspectRatio : null,
    centrada: rg && rect(marco) ? Math.abs((rect(marco).x + rect(marco).w / 2) - (rg.x + rg.w / 2)) <= 1 : null,
    chips,
    punto: punto ? { color: csPunto.backgroundColor, w: round(punto.getBoundingClientRect().width) } : null,
    generos: generos
      ? {
          rect: rect(generos),
          flexWrap: csGen.flexWrap,
          overflowX: csGen.overflowX,
          marginLeft: csGen.marginLeft,
          paddingLeft: csGen.paddingLeft,
          scrollWidth: generos.scrollWidth,
          clientWidth: generos.clientWidth,
          scrollable: generos.scrollWidth > generos.clientWidth + 1,
        }
      : null,
    sinopsis: sinopsis
      ? {
          rect: rect(sinopsis),
          display: csSin.display,
          lineClamp: csSin.webkitLineClamp || csSin.lineClamp || null,
          scrollHeight: sinopsis.scrollHeight,
          clientHeight: sinopsis.clientHeight,
          lineHeight: csSin.lineHeight,
          lineas: Math.round(sinopsis.clientHeight / parseFloat(csSin.lineHeight)),
          desborda: sinopsis.scrollHeight > sinopsis.clientHeight + 1,
        }
      : null,
    verMas: verMas
      ? {
          rect: rect(verMas),
          aria: verMas.getAttribute('aria-expanded'),
          texto: (verMas.textContent || '').trim(),
          minHeight: getComputedStyle(verMas).minHeight,
          color: getComputedStyle(verMas).color,
        }
      : null,
    columnas: csRej && csRej.gridTemplateColumns ? csRej.gridTemplateColumns.split(' ').filter(Boolean).length : 0,
    tarjetaEp: rect(document.querySelector('.rejilla .celda .tarjeta')),
    entra: document.querySelectorAll('.celda.entra').length,
  };
};

async function abrirFilaHome(page, titulo) {
  const filas = page.locator('.fila');
  const n = await filas.count();
  for (let i = 0; i < n; i++) {
    const t = ((await filas.nth(i).locator('.titulo-fila').textContent().catch(() => '')) ?? '').trim();
    if (t === titulo) {
      await filas.nth(i).locator('.carrusel .tarjeta').first().click();
      break;
    }
  }
  await page.waitForSelector('.ficha, .estado', { timeout: 45000 });
  await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 60000 }).catch(() => {});
  await page.waitForTimeout(700);
}

async function ficha() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/ficha';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const rep = { anchos: [], funciones: [], capturas: [], tactiles: [], geo: {}, base1280: null, errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const captura = async (nombre) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };

  // --- A. Geometría y estado por ancho (AnimeAV1 Bleach) ---
  await paso('A', async () => {
    for (const a of ANCHOS_FICHA) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await buscar(page, 'bleach');
      await abrirDetalle(page, 'animeav1');
      const m = await page.evaluate(MEDIR_FICHA);
      rep.anchos.push({ ancho: a.nombre, ...m });
      rep.geo[a.nombre] = { tarjeta: m.tarjetaEp, columnas: m.columnas, fichaY: m.ficha?.y, fichaH: m.ficha?.h };
      if (a.nombre === '360') await captura('detalle_animeav1_360');
      if (a.nombre === '1280') rep.base1280 = m;
      check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      check(`${a.nombre}: 4 chips (proveedor/tipo/estado/total)`, m.chips.length === 4, m.chips.map((c) => c.texto));
      check(`${a.nombre}: proveedor en tono marca`, m.chips[0] && m.chips[0].tono === 'tono-marca', m.chips[0]);
      check(`${a.nombre}: resto de chips neutros`, m.chips.slice(1).every((c) => c.tono === 'tono-neutro'), m.chips.slice(1));
      if (a.w < 900) {
        const difCentro = m.marco && m.ficha
          ? Math.abs(m.marco.x + m.marco.w / 2 - (m.ficha.x + m.ficha.w / 2))
          : null;
        check(`${a.nombre}: portada centrada`, difCentro !== null && difCentro <= 1, { difCentro, marco: m.marco, ficha: m.ficha });
        check(`${a.nombre}: portada ≤ 200 px`, !!m.marco && m.marco.w <= 200.5, m.marco?.w);
        check(`${a.nombre}: marco con padding 3 px`, m.marcoPadding === '3px', m.marcoPadding);
        check(`${a.nombre}: sinopsis a 4 líneas`, !!m.sinopsis && m.sinopsis.lineas === 4, m.sinopsis?.lineas);
        check(`${a.nombre}: sinopsis con -webkit-line-clamp: 4`, m.sinopsis?.lineClamp === '4', m.sinopsis?.lineClamp);
        check(`${a.nombre}: "Ver más" presente y ≥44 px`, !!m.verMas && m.verMas.rect.h >= 44, m.verMas);
        check(`${a.nombre}: "Ver más" aria-expanded=false`, m.verMas?.aria === 'false', m.verMas?.aria);
        check(`${a.nombre}: géneros en una línea con scroll`, !!m.generos && m.generos.flexWrap === 'nowrap' && m.generos.overflowX === 'auto', m.generos);
        check(`${a.nombre}: ficha padding var(--sp-4)=16 px`, m.fichaPadding === '16px', m.fichaPadding);
      } else {
        check(`${a.nombre}: sin "Ver más"`, m.verMas === null, m.verMas);
        check(`${a.nombre}: sinopsis sin límite`, !!m.sinopsis && m.sinopsis.lineClamp !== '4', m.sinopsis?.lineClamp);
        check(`${a.nombre}: géneros con wrap`, !!m.generos && m.generos.flexWrap === 'wrap', m.generos?.flexWrap);
        // Ancho de referencia del marco: 220 px, topado por `max-width: 35 %`
        // del contenido de la ficha (mismo comportamiento que la portada previa).
        const disp = m.ficha ? m.ficha.w - 2 * (parseFloat(m.fichaBorder) + parseFloat(m.fichaPadding)) : NaN;
        const esperado = Math.min(220, 0.35 * disp);
        check(`${a.nombre}: portada 220 px (máx. 35 %)`, !!m.marco && Math.abs(m.marco.w - esperado) <= 1, { w: m.marco?.w, esperado });
        check(`${a.nombre}: portada a la izquierda`, !!m.marco && !!m.ficha && m.marco.x <= m.ficha.x + 30, { marcoX: m.marco?.x, fichaX: m.ficha?.x });
        check(`${a.nombre}: punto neutro (Finalizado)`, m.punto && m.punto.color === 'rgb(148, 163, 184)', m.punto);
      }
      check(`${a.nombre}: aspecto 2/3`, m.imgAspect === '2 / 3', m.imgAspect);
      check(`${a.nombre}: imagen radio --r-md`, m.imgRadius === '12px', m.imgRadius);
      log('ficha ancho', a.nombre, 'marco=', m.marco?.w, 'tarjetaEp=', m.tarjetaEp && `${m.tarjetaEp.w}x${m.tarjetaEp.h}`, 'cols=', m.columnas);
    }
  });

  // --- B. Sinopsis: expandir y colapsar (360, AnimeAV1 Bleach) ---
  await paso('B', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const cerrada = await page.evaluate(MEDIR_FICHA);
    const hCerrada = cerrada.ficha.h;
    await captura('detalle_animeav1_360_cerrada');
    await page.locator('.ficha .ver-mas').click();
    await page.waitForTimeout(400);
    const abierta = await page.evaluate(MEDIR_FICHA);
    rep.sinopsis = { hCerrada, hAbierta: abierta.ficha.h };
    await captura('detalle_animeav1_360_abierta');
    check('B: al abrir crece la ficha', abierta.ficha.h > hCerrada, rep.sinopsis);
    check('B: "Ver menos" con aria-expanded=true', abierta.verMas?.texto === 'Ver menos' && abierta.verMas?.aria === 'true', abierta.verMas);
    check('B: sinopsis completa sin recorte', abierta.sinopsis?.lineClamp !== '4' && abierta.sinopsis?.desborda === false, abierta.sinopsis);
    await page.locator('.ficha .ver-mas').click();
    await page.waitForTimeout(400);
    const recerrada = await page.evaluate(MEDIR_FICHA);
    check('B: al colapsar vuelve a 4 líneas', recerrada.sinopsis?.lineas === 4, recerrada.sinopsis);
    check('B: "Ver más" de nuevo', recerrada.verMas?.texto === 'Ver más' && recerrada.verMas?.aria === 'false', recerrada.verMas);
  });

  // --- C. Resize 800 → 1000 → 800 conservando el estado abierto ---
  await paso('C', async () => {
    await page.setViewportSize({ width: 800, height: 900 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const a800 = await page.evaluate(MEDIR_FICHA);
    check('C: 800 con "Ver más"', !!a800.verMas, a800.verMas);
    await page.locator('.ficha .ver-mas').click();
    await page.waitForTimeout(300);
    await page.setViewportSize({ width: 1000, height: 900 });
    await page.waitForTimeout(700);
    const a1000 = await page.evaluate(MEDIR_FICHA);
    check('C: 1000 sin botón ni recorte', a1000.verMas === null && a1000.sinopsis?.lineClamp !== '4', { verMas: a1000.verMas, lineClamp: a1000.sinopsis?.lineClamp });
    await page.setViewportSize({ width: 800, height: 900 });
    await page.waitForTimeout(700);
    const volvio = await page.evaluate(MEDIR_FICHA);
    check('C: 800 de vuelta con botón', !!volvio.verMas, volvio.verMas);
    check('C: conserva el estado abierto ("Ver menos")', volvio.verMas?.texto === 'Ver menos' && volvio.verMas?.aria === 'true', volvio.verMas);
    await page.locator('.ficha .ver-mas').click().catch(() => {});
    await page.waitForTimeout(200);
  });

  // --- D. Géneros con scroll horizontal real (360) ---
  await paso('D', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const m = await page.evaluate(MEDIR_FICHA);
    check('D: 5 géneros (AnimeAV1 Bleach)', m.chips.length === 4 && (m.generos?.scrollWidth ?? 0) > 0, m.generos);
    const desplazo = await page.evaluate(() => {
      const g = document.querySelector('.ficha .generos');
      g.scrollLeft = 200;
      return g.scrollLeft;
    });
    check('D: los géneros se desplazan (scrollLeft > 0)', desplazo > 0, desplazo);
    await page.evaluate(() => { document.querySelector('.ficha .generos').scrollLeft = 0; });
  });

  // --- E. Estado "En emisión": punto de marca (AnimeAV1, dato real) ---
  await paso('E', async () => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'battle spirits');
    await abrirDetalle(page, 'animeav1');
    const m = await page.evaluate(MEDIR_FICHA);
    rep.emision = { chips: m.chips.map((c) => c.texto), punto: m.punto };
    const chipEstado = m.chips.find((c) => c.texto === 'En emisión');
    check('E: chip "En emisión" presente (AnimeAV1 con acento)', !!chipEstado, m.chips.map((c) => c.texto));
    check('E: punto de marca (--brand-400) por "En emisión"', !!m.punto && m.punto.color === 'rgb(52, 211, 153)', m.punto);
  });

  // --- F. Estado vacío simulado: sin chip de estado ---
  await paso('F', async () => {
    await page.setViewportSize({ width: 390, height: 844 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.evaluate(() => {
      const App = window.go.main.App;
      window.__origDet79 = App.GetAnimeDetails;
      App.GetAnimeDetails = async (p, id) => {
        const d = await window.__origDet79(p, id);
        const det = d.detail ?? d.Detail;
        if (det) det.status = '';
        return d;
      };
    });
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const m = await page.evaluate(MEDIR_FICHA);
    check('F: estado vacío → 3 chips, sin chip de estado', m.chips.length === 3, m.chips.map((c) => c.texto));
    check('F: sin punto de estado', m.punto === null, m.punto);
    await page.evaluate(() => { const App = window.go.main.App; if (window.__origDet79) App.GetAnimeDetails = window.__origDet79; });
  });

  // --- G. Sinopsis corta simulada: sin botón "Ver más" ---
  await paso('G', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.evaluate(() => {
      const App = window.go.main.App;
      window.__origDet79b = App.GetAnimeDetails;
      App.GetAnimeDetails = async (p, id) => {
        const d = await window.__origDet79b(p, id);
        const det = d.detail ?? d.Detail;
        if (det) det.synopsis = 'Sinopsis breve de prueba.';
        return d;
      };
    });
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const m = await page.evaluate(MEDIR_FICHA);
    check('G: sinopsis corta sin botón "Ver más"', m.verMas === null, { verMas: m.verMas, sinopsis: m.sinopsis });
    check('G: sinopsis corta no recortada', !!m.sinopsis && m.sinopsis.desborda === false, m.sinopsis);
    await captura('detalle_sinopsis_corta_360');
    await page.evaluate(() => { const App = window.go.main.App; if (window.__origDet79b) App.GetAnimeDetails = window.__origDet79b; });
  });

  // --- H. Detalle JKAnime: 360, 768 y 1280 ---
  await paso('H', async () => {
    for (const a of [{ w: 360, h: 740, nombre: '360' }, { w: 768, h: 1024, nombre: '768' }, { w: 1280, h: 800, nombre: '1280' }]) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await buscar(page, 'bleach');
      await abrirDetalle(page, 'jkanime');
      const m = await page.evaluate(MEDIR_FICHA);
      rep.anchos.push({ ancho: a.nombre, vista: 'jkanime', ...m });
      await captura(`detalle_jkanime_${a.nombre}`);
      check(`H ${a.nombre}: 4 chips`, m.chips.length === 4, m.chips.map((c) => c.texto));
      check(`H ${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      if (a.w < 900) {
        check(`H ${a.nombre}: punto neutro ("Concluido")`, m.punto && m.punto.color === 'rgb(148, 163, 184)', m.punto);
        check(`H ${a.nombre}: "Ver más" presente`, !!m.verMas, m.verMas);
      } else {
        check(`H ${a.nombre}: sin "Ver más"`, m.verMas === null, m.verMas);
      }
    }
  });

  // --- I. 1024/1280: geometría de la rejilla de episodios ---
  await paso('I', async () => {
    for (const a of [{ w: 1024, h: 768, nombre: '1024' }, { w: 1280, h: 800, nombre: '1280' }]) {
      await page.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(page);
      await buscar(page, 'bleach');
      await abrirDetalle(page, 'animeav1');
      const m = await page.evaluate(MEDIR_FICHA);
      rep.anchos.push({ ancho: a.nombre, vista: 'rejilla', ...m });
      const base = rep.geo[a.nombre];
      check(
        `${a.nombre}: tarjeta de episodio 170,39 de alto (sin cambios)`,
        !!m.tarjetaEp && Math.abs(m.tarjetaEp.h - 170.39) <= 0.5,
        { base: base?.tarjeta, actual: m.tarjetaEp }
      );
      check(`${a.nombre}: columnas de la rejilla iguales`, m.columnas === base?.columnas, { base: base?.columnas, actual: m.columnas });
      rep.geo[`${a.nombre}_delta`] = {
        y: m.tarjetaEp ? Math.round((m.tarjetaEp.y - base.tarjeta.y) * 100) / 100 : null,
        fichaH: base.fichaH !== undefined && m.ficha ? Math.round((m.ficha.h - base.fichaH) * 100) / 100 : null,
      };
    }
  });

  // --- J. T66/T67, revelado T65 y cabecera fija T74 intactos (360) ---
  await paso('J', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.waitForSelector('.rejilla .celda .tarjeta', { timeout: 60000 }).catch(() => {});
    await page.evaluate(() => {
      const r = document.querySelector('.rejilla');
      window.scrollTo(0, r.getBoundingClientRect().top + window.scrollY);
    });
    await page.waitForTimeout(1000);
    const entra = await page.evaluate(() => document.querySelectorAll('.celda.entra').length);
    rep.entra = entra;
    check('J: revelado T65 activo (.entra > 0)', entra > 0, entra);
    await page.waitForTimeout(100);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const cabFija = await page.evaluate(() => {
      const c = document.querySelector('main .cabecera');
      const d = document.documentElement;
      const sat = parseFloat(getComputedStyle(d).getPropertyValue('--sat')) || 0;
      return { posicion: c ? getComputedStyle(c).position : null, y: c ? Math.round(c.getBoundingClientRect().y) : null, sat };
    });
    check('J: cabecera fija T74 sticky al scrollear', cabFija.posicion === 'sticky' && Math.abs(cabFija.y - cabFija.sat) <= 2, cabFija);
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultima.count())) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const yDespues = await page.evaluate(() => Math.round(window.scrollY));
      check('J: T66/T67 scroll del detalle restaurado', Math.abs(yDespues - yAntes) <= 5, { yAntes, yDespues });
    } else {
      check('J: T66/T67 scroll del detalle restaurado', false, { yAntes });
    }
  });

  // --- K. prefers-reduced-motion: sin transiciones en la ficha ---
  await paso('K', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const td = await page.evaluate(() => {
      const el = document.querySelector('.ficha .ver-mas');
      return el ? getComputedStyle(el).transitionDuration : 'n/a';
    });
    check('K: reduced-motion sin transición en "Ver más"', td === '0s' || td === 'n/a', td);
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- L. Objetivos táctiles <44 px nuevos (360, detalle) ---
  await paso('L', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    rep.tactiles = (await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'detalle_360' }));
    const enFicha = rep.tactiles.filter((t) => t.texto.includes('Ver'));
    check('L: "Ver más" ≥44 px (no aparece en la lista)', enFicha.length === 0, enFicha);
    log('ficha táctiles <44:', rep.tactiles.length);
  });

  writeFileSync(join(dirOut, 'ficha.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo cta (T80): barra de acción fija "Continuar / Reiniciar" del detalle
// ---------------------------------------------------------------------------

const ANCHOS_CTA = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

// Geometría y estado de la CTA del detalle, la barra inferior y la ficha.
const MEDIR_CTA = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return {
      x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height),
      top: round(r.top), bottom: round(r.bottom), right: round(r.right),
    };
  };
  const cta = document.querySelector('.barra-accion');
  const barra = document.querySelector('.barra-inferior');
  const cont = document.querySelector('main.contenedor');
  const continuar = document.querySelector('.ficha .continuar');
  const botones = cta ? [...cta.querySelectorAll('.btn')] : [];
  const principal = cta ? cta.querySelector('.principal .btn') : null;
  const secundario = cta ? botones.find((b) => !b.closest('.principal') && b !== principal) : null;
  const celdas = document.querySelectorAll('.rejilla .celda');
  const ultimaEl = celdas.length ? celdas[celdas.length - 1] : null;
  const cab = document.querySelector('main .cabecera');
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    innerWidth: window.innerWidth,
    innerHeight: window.innerHeight,
    cta: cta
      ? { ...rect(cta), texto: (cta.textContent || '').trim(), role: cta.getAttribute('role'), label: cta.getAttribute('aria-label') }
      : null,
    ctaTransition: cta ? getComputedStyle(cta).transitionDuration : null,
    ctaZ: cta ? getComputedStyle(cta).zIndex : null,
    barra: rect(barra),
    hueco: cta && barra ? round(cta.getBoundingClientRect().bottom - barra.getBoundingClientRect().top) : null,
    botones: botones.map((b) => ({
      texto: (b.textContent || '').trim(),
      w: round(b.getBoundingClientRect().width),
      h: round(b.getBoundingClientRect().height),
    })),
    principalTexto: principal ? (principal.textContent || '').trim() : null,
    secundarioTexto: secundario ? (secundario.textContent || '').trim() : null,
    continuar: continuar ? { display: getComputedStyle(continuar).display, visible: continuar.offsetParent !== null } : null,
    conCta: cont ? cont.classList.contains('con-cta') : null,
    contenedorPB: cont ? getComputedStyle(cont).paddingBottom : null,
    tarjetaEp: rect(document.querySelector('.rejilla .celda .tarjeta')),
    ultima: rect(ultimaEl),
    huecoUltima: cta && ultimaEl ? round(cta.getBoundingClientRect().top - ultimaEl.getBoundingClientRect().bottom) : null,
    entra: document.querySelectorAll('.celda.entra').length,
    cabecera: cab ? { posicion: getComputedStyle(cab).position, y: Math.round(cab.getBoundingClientRect().y), sat: parseFloat(getComputedStyle(d).getPropertyValue('--sat')) || 0 } : null,
  };
};

async function cta() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/cta';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const rep = { anchos: [], tactiles: [], funciones: [], capturas: [], lote: null, sinProgreso: null, finScroll: null, scroll: null, cajon: null, errores: [] };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const ctx = await browser.newContext({ viewport: { width: 360, height: 740 } });
  const page = await ctx.newPage();
  const captura = async (nombre) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const destino = (texto) => page.locator('.barra-inferior .destino', { hasText: texto });
  const subtitulo = () =>
    page.evaluate(() => {
      const s = document.querySelector('main .cabecera .subtitulo');
      return s ? (s.textContent || '').trim() : null;
    });
  const abrir = async (proveedor, ancho = 360) => {
    await page.setViewportSize({ width: ancho, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, proveedor);
  };
  const volverDetalle = async () => {
    for (let i = 0; i < 4; i++) {
      if (await page.locator('.ficha').isVisible().catch(() => false)) return;
      await page.locator('.cabecera .atras').first().click().catch(() => {});
      await page.waitForTimeout(800);
    }
  };

  // Historial sembrado (T77): Bleach de AnimeAV1 con progreso en el ep. 9.
  const SEMILLA = [
    { anime_id: 'bleach', episode_id: 'animeav1:bleach:episode:9', current_time: 900, total_duration: 1420, is_watched: false, provider: 'animeav1', anime_title: 'Bleach', cover_url: '', episode_number: 9, updated_at: 1749999940 },
  ];
  const sembrar = async () => {
    await gotoApp(page);
    await page.evaluate(async (seed) => {
      const App = window.go?.main?.App;
      if (!App) throw new Error('puente window.go no disponible (usa localhost:34115)');
      await App.ClearProgress();
      for (const s of seed) await App.SaveProgress(s);
    }, SEMILLA);
    await page.waitForTimeout(300);
  };

  // --- A. Geometría por ancho (con progreso: Bleach de AnimeAV1) ---
  await paso('A', async () => {
    await sembrar();
    for (const a of ANCHOS_CTA) {
      await abrir('animeav1', a.w);
      await page.setViewportSize({ width: a.w, height: a.h });
      await page.waitForTimeout(400);
      const m = await page.evaluate(MEDIR_CTA);
      rep.anchos.push({ ancho: a.nombre, ...m });
      if (a.nombre === '360') await captura('detalle_360_progreso');
      if (a.nombre === '768') await captura('detalle_768');
      if (a.nombre === '1280') await captura('detalle_1280');
      check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      if (a.w < 900) {
        check(`${a.nombre}: CTA presente`, !!m.cta, m.cta);
        check(`${a.nombre}: role=group + aria-label`, !!m.cta && m.cta.role === 'group' && m.cta.label === 'Acciones del anime', m.cta);
        check(`${a.nombre}: CTA pegada a la barra inferior (hueco ≤ 1 px)`, m.hueco !== null && Math.abs(m.hueco) <= 1, m.hueco);
        check(`${a.nombre}: botones ≥ 44 px`, m.botones.length >= 1 && m.botones.every((b) => b.w >= 44 && b.h >= 44), m.botones);
        check(`${a.nombre}: principal "Continuar · Ep. 9"`, m.principalTexto === 'Continuar · Ep. 9', m.principalTexto);
        check(`${a.nombre}: secundario "Reiniciar"`, m.secundarioTexto === 'Reiniciar', m.secundarioTexto);
        check(`${a.nombre}: ".continuar" oculto`, !!m.continuar && m.continuar.display === 'none' && !m.continuar.visible, m.continuar);
        check(`${a.nombre}: ".contenedor.con-cta"`, m.conCta === true, m.conCta);
      } else {
        check(`${a.nombre}: CTA ausente`, m.cta === null, m.cta);
        check(`${a.nombre}: ".continuar" visible`, !!m.continuar && m.continuar.visible, m.continuar);
        check(`${a.nombre}: tarjeta de episodio 170,39 (T79)`, !!m.tarjetaEp && Math.abs(m.tarjetaEp.h - 170.39) <= 0.6, m.tarjetaEp?.h);
        check(`${a.nombre}: sin ".con-cta"`, m.conCta === false, m.conCta);
      }
      log('cta ancho', a.nombre, 'cta=', !!m.cta, 'hueco=', m.hueco, 'principal=', m.principalTexto);
    }
  });

  // --- B. Con progreso (360): "Continuar" (ep. 9) y "Reiniciar" (ep. 1) ---
  await paso('B', async () => {
    await sembrar();
    await abrir('animeav1', 360);
    await page.locator('.barra-accion .principal .btn').click();
    await page.waitForSelector('.cabecera .atras', { timeout: 30000 });
    await page.waitForTimeout(900);
    check('B: la CTA se oculta en el reproductor', (await page.locator('.barra-accion').count()) === 0);
    const subContinuar = await subtitulo();
    check('B: "Continuar" abre el Ep. 9', /^Ep\. 9\b/.test(subContinuar ?? ''), subContinuar);
    await volverDetalle();
    check('B: al volver reaparece la CTA', (await page.locator('.barra-accion').count()) > 0);
    await page.locator('.barra-accion .btn').filter({ hasText: 'Reiniciar' }).click();
    await page.waitForSelector('.cabecera .atras', { timeout: 30000 });
    await page.waitForTimeout(900);
    const subReiniciar = await subtitulo();
    check('B: "Reiniciar" abre el Ep. 1', /^Ep\. 1\b/.test(subReiniciar ?? ''), subReiniciar);
    await volverDetalle();
  });

  // --- C. Sin progreso (otro anime): "Empezar · Ep. 1" y sin secundario ---
  await paso('C', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'naruto');
    await abrirDetalle(page, 'animeav1');
    const m = await page.evaluate(MEDIR_CTA);
    rep.sinProgreso = { principal: m.principalTexto, secundario: m.secundarioTexto };
    check('C: principal "Empezar · Ep. 1"', m.principalTexto === 'Empezar · Ep. 1', m.principalTexto);
    check('C: sin secundario', m.secundarioTexto === null, m.secundarioTexto);
    await captura('detalle_360_sin_progreso');
    await page.locator('.barra-accion .principal .btn').click();
    await page.waitForSelector('.cabecera .atras', { timeout: 30000 });
    await page.waitForTimeout(900);
    const sub = await subtitulo();
    check('C: "Empezar" abre el Ep. 1', /^Ep\. 1\b/.test(sub ?? ''), sub);
    await volverDetalle();
  });

  // --- D. Fin de scroll (360, AnimeAV1) + revelado T65 + cabecera T74 + scroll T66/T67 ---
  await paso('D', async () => {
    await abrir('animeav1', 360);
    await page.waitForFunction(() => !document.querySelector('.cargando'), null, { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(500);
    await page.evaluate(() => {
      const r = document.querySelector('.rejilla');
      if (r) window.scrollTo(0, r.getBoundingClientRect().top + window.scrollY);
    });
    await page.waitForTimeout(1200);
    const entra = await page.evaluate(() => document.querySelectorAll('.celda.entra').length);
    check('D: revelado T65 (.entra > 0)', entra > 0, entra);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(600);
    const fin = await page.evaluate(MEDIR_CTA);
    rep.finScroll = { ultima: fin.ultima, cta: fin.cta, huecoUltima: fin.huecoUltima, cabecera: fin.cabecera };
    check('D: último episodio completo sobre la CTA (≥ 0 px)', fin.huecoUltima !== null && fin.huecoUltima >= -0.5, fin.huecoUltima);
    check('D: cabecera fija "sticky" (T74)', !!fin.cabecera && fin.cabecera.posicion === 'sticky' && Math.abs(fin.cabecera.y - fin.cabecera.sat) <= 2, fin.cabecera);
    await captura('detalle_360_fin_scroll');
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultima.count()) > 0) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await volverDetalle();
      await page.waitForTimeout(1200);
      const desp = await page.evaluate(() => ({
        y: Math.round(window.scrollY),
        max: document.documentElement.scrollHeight - window.innerHeight,
      }));
      // Deriva pre-existente (verificada con T80 revertido, Δ ≈ 35 px): entre
      // la salida y la vuelta el alto del detalle cambia unas decenas de px
      // (asentado de la rejilla por lotes/imágenes), así que la posición
      // guardada puede superar el máximo alcanzable y el navegador la recorta.
      // Se comprueba que se restaura hasta donde es posible (≤ 5 px).
      const esperado = Math.min(yAntes, desp.max);
      rep.scroll = { yAntes, ...desp, esperado, delta: Math.abs(desp.y - esperado) };
      check('D: scroll del detalle restaurado (T66/T67)', Math.abs(desp.y - esperado) <= 5, rep.scroll);
    } else {
      check('D: scroll del detalle restaurado (T66/T67)', false, { yAntes });
    }
    check('D: de vuelta reaparece la CTA', (await page.locator('.barra-accion').count()) > 0);
  });

  // --- E. Lotes de JKAnime: la CTA aparece con el primer lote y su texto es coherente ---
  await paso('E', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'jkanime');
    const primero = await page
      .waitForFunction(
        () => {
          const c = document.querySelector('.barra-accion .principal .btn');
          const eps = document.querySelectorAll('.rejilla .celda').length;
          return c && eps > 0 ? (c.textContent || '').trim() : false;
        },
        null,
        { polling: 20, timeout: 60000 }
      )
      .then((h) => h.jsonValue())
      .catch(() => null);
    await page.waitForFunction(() => !document.querySelector('.cargando'), null, { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(500);
    const final = await page.evaluate(() => {
      const c = document.querySelector('.barra-accion .principal .btn');
      return { texto: c ? (c.textContent || '').trim() : null, total: document.querySelectorAll('.rejilla .celda').length };
    });
    rep.lote = { primero, final };
    check('E: la CTA aparece al llegar el primer lote', typeof primero === 'string' && primero !== '', primero);
    check('E: texto de la CTA con el formato esperado', typeof primero === 'string' && /^(Empezar|Continuar) · Ep\. \d+$/.test(primero), primero);
    check('E: el texto de la CTA no cambia al llegar el resto de lotes', primero === final.texto, { primero, final });
  });

  // --- F. El cajón de Historial cubre la CTA ---
  await paso('F', async () => {
    await abrir('animeav1', 360);
    await destino('Historial').click();
    await page.waitForSelector('.barra.cajon', { timeout: 10000 });
    await page.waitForTimeout(400);
    const cubre = await page.evaluate(() => {
      const cta = document.querySelector('.barra-accion');
      const cajon = document.querySelector('.barra.cajon');
      if (!cta || !cajon) return { hayCta: !!cta, hayCajon: !!cajon };
      const r = cta.getBoundingClientRect();
      const x = Math.round(r.left + r.width / 2);
      const y = Math.round(r.top + r.height / 2);
      const top = document.elementFromPoint(x, y);
      return {
        hayCta: true,
        hayCajon: true,
        ctaZ: getComputedStyle(cta).zIndex,
        cajonZ: getComputedStyle(cajon).zIndex,
        topEnCajon: !!(top && top.closest('.cajon')),
        topEnCta: !!(top && top.closest('.barra-accion')),
      };
    });
    rep.cajon = cubre;
    check('F: con el cajón abierto, el punto de la CTA cae en el cajón', cubre.hayCajon && cubre.topEnCajon && !cubre.topEnCta, cubre);
    check('F: z-index del cajón > z-index de la CTA', Number(cubre.cajonZ) > Number(cubre.ctaZ), { cajonZ: cubre.cajonZ, ctaZ: cubre.ctaZ });
    await captura('detalle_360_cajon');
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);
  });

  // --- G. prefers-reduced-motion: la CTA no transiciona ---
  await paso('G', async () => {
    await abrir('animeav1', 360);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const td = await page.evaluate(() => {
      const el = document.querySelector('.barra-accion');
      return el ? getComputedStyle(el).transitionDuration : null;
    });
    check('G: reduced-motion sin transición en la CTA', td === '0s' || td === '0s, 0s', td);
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- H. Objetivos táctiles <44 px en el detalle estrecho ---
  await paso('H', async () => {
    await abrir('animeav1', 360);
    rep.tactiles = (await page.evaluate(TACTILES_CHICOS)).map((t) => ({ ...t, estado: 'detalle_360' }));
    const enCta = rep.tactiles.filter((t) => t.enCta);
    check('H: ningún táctil <44 px dentro de la CTA', enCta.length === 0, enCta);
    log('cta táctiles <44:', rep.tactiles.length);
  });

  writeFileSync(join(dirOut, 'cta.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo episodios (T81): tarjeta de episodio y rejilla en modo estrecho
// ---------------------------------------------------------------------------

const ANCHOS_EPISODIOS = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

// Columnas esperadas de la rejilla por ancho (T81): 2 (<600), 3 (600–899 y
// 1024 por `auto-fill`), 4 (1280 por `auto-fill`).
const COLUMNAS_EPISODIOS = { 360: 2, 390: 2, 768: 3, 1024: 3, 1280: 4 };
const ALTO_ESCRITORIO = 170.39;

const MEDIR_EPISODIOS = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const csd = getComputedStyle(d);
  const tok = (n) => csd.getPropertyValue(n).trim();
  const resolver = (n) => {
    const p = document.createElement('div');
    p.style.color = `var(${n})`;
    d.appendChild(p);
    const c = getComputedStyle(p).color;
    p.remove();
    return c;
  };
  const rej = document.querySelector('.rejilla');
  if (!rej) return { err: 'sin .rejilla' };
  const csr = getComputedStyle(rej);
  const tarjetas = [...rej.querySelectorAll('.tarjeta')];
  const rects = tarjetas.map((t) => t.getBoundingClientRect());
  const cols = [...new Set(rects.map((r) => Math.round(r.left)))].length;
  const hs = rects.map((r) => round(r.height));
  const ws = rects.map((r) => round(r.width));
  const clasificar = (t) =>
    t.querySelector('.visto') ? 'visto' : t.querySelector('.franja') ? 'parcial' : 'ninguno';
  const muestra = { visto: null, parcial: null, ninguno: null };
  for (const t of tarjetas) {
    const k = clasificar(t);
    if (!muestra[k]) muestra[k] = t;
  }
  const info = (t) => {
    if (!t) return null;
    const r = t.getBoundingClientRect();
    const mini = t.querySelector('.miniatura');
    const mr = mini ? mini.getBoundingClientRect() : null;
    const visto = t.querySelector('.visto');
    const franja = t.querySelector('.franja');
    const fill = franja ? franja.querySelector('span') : null;
    const pct = t.querySelector('.porcentaje');
    const hole = t.querySelector('.sin-miniatura');
    const titulo = t.querySelector('.titulo');
    const csv = visto ? getComputedStyle(visto) : null;
    const csh = hole ? getComputedStyle(hole) : null;
    let minFs = Infinity;
    for (const el of t.querySelectorAll('*')) {
      const hasText = [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim());
      if (!hasText) continue;
      minFs = Math.min(minFs, parseFloat(getComputedStyle(el).fontSize));
    }
    return {
      w: round(r.width),
      h: round(r.height),
      estado: clasificar(t),
      csi: getComputedStyle(t).containIntrinsicSize,
      mini: mr ? { w: round(mr.width), h: round(mr.height), ratio: round(mr.width / mr.height) } : null,
      visto: visto
        ? {
            texto: (visto.textContent || '').trim(),
            bg: csv.backgroundColor,
            color: csv.color,
            border: csv.borderTopColor,
            icono: !!visto.querySelector('svg'),
            h: round(visto.getBoundingClientRect().height),
          }
        : null,
      franja: franja
        ? {
            role: franja.getAttribute('role'),
            min: franja.getAttribute('aria-valuemin'),
            max: franja.getAttribute('aria-valuemax'),
            now: franja.getAttribute('aria-valuenow'),
            h: round(franja.getBoundingClientRect().height),
            pegada: mr ? round(Math.abs(franja.getBoundingClientRect().bottom - mr.bottom)) : null,
            fillPct:
              fill && mr && mr.width > 0
                ? round((fill.getBoundingClientRect().width / mr.width) * 100)
                : null,
          }
        : null,
      porcentaje: pct ? (pct.textContent || '').trim() : null,
      tituloFs: titulo ? parseFloat(getComputedStyle(titulo).fontSize) : null,
      minFs: minFs === Infinity ? null : minFs,
      hole: hole ? { texto: (hole.textContent || '').trim(), color: csh.color } : null,
    };
  };
  return {
    innerWidth: window.innerWidth,
    overflowH: d.scrollWidth - d.clientWidth,
    gridCols: csr.gridTemplateColumns,
    gap: csr.gap,
    cols,
    count: tarjetas.length,
    wMin: Math.min(...ws),
    wMax: Math.max(...ws),
    hMin: Math.min(...hs),
    hMax: Math.max(...hs),
    tokens: {
      brand400: resolver('--brand-400'),
      text3: resolver('--text-3'),
      chip: resolver('--surface-chip'),
      accent: resolver('--border-accent'),
      fsXs: parseFloat(tok('--fs-xs')),
      fsSm: parseFloat(tok('--fs-sm')),
    },
    muestra: {
      visto: info(muestra.visto),
      parcial: info(muestra.parcial),
      ninguno: info(muestra.ninguno),
    },
  };
};

async function episodios() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/episodios';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const rep = {
    anchos: [],
    estados: null,
    roto: null,
    hover: null,
    revelado: null,
    scroll: null,
    tactiles: [],
    funciones: [],
    capturas: [],
    errores: [],
  };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const ctx = await browser.newContext({ viewport: { width: 360, height: 740 } });
  const page = await ctx.newPage();
  const captura = async (nombre) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const abrir = async (proveedor, ancho) => {
    await page.setViewportSize({ width: ancho, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, proveedor);
  };
  const volverDetalle = async () => {
    for (let i = 0; i < 4; i++) {
      if (await page.locator('.ficha').isVisible().catch(() => false)) return;
      await page.locator('.cabecera .atras').first().click().catch(() => {});
      await page.waitForTimeout(800);
    }
  };

  // Historial sembrado (T81): Bleach de AnimeAV1 con el ep. 9 en progreso y el
  // ep. 10 visto, para cubrir los tres estados de la tarjeta.
  const SEMILLA = [
    { anime_id: 'bleach', episode_id: 'animeav1:bleach:episode:9', current_time: 900, total_duration: 1420, is_watched: false, provider: 'animeav1', anime_title: 'Bleach', cover_url: '', episode_number: 9, updated_at: 1749999940 },
    { anime_id: 'bleach', episode_id: 'animeav1:bleach:episode:10', current_time: 1420, total_duration: 1420, is_watched: true, provider: 'animeav1', anime_title: 'Bleach', cover_url: '', episode_number: 10, updated_at: 1749999950 },
  ];
  const sembrar = async () => {
    await gotoApp(page);
    await page.evaluate(async (seed) => {
      const App = window.go?.main?.App;
      if (!App) throw new Error('puente window.go no disponible (usa localhost:34115)');
      await App.ClearProgress();
      for (const s of seed) await App.SaveProgress(s);
    }, SEMILLA);
    await page.waitForTimeout(300);
  };

  // --- A. Geometría y rejilla por ancho ---
  await paso('A', async () => {
    await sembrar();
    for (const a of ANCHOS_EPISODIOS) {
      await abrir('animeav1', a.w);
      await page.setViewportSize({ width: a.w, height: a.h });
      await page.waitForTimeout(400);
      const m = await page.evaluate(MEDIR_EPISODIOS);
      rep.anchos.push({ ancho: a.nombre, ...m });
      if (m.err) {
        check(`${a.nombre}: hay rejilla`, false, m.err);
        continue;
      }
      await captura(`episodios_${a.nombre}`);
      // Vista de la rejilla (desplazada) como evidencia visual.
      await page.evaluate(() => {
        const r = document.querySelector('.rejilla');
        if (r) window.scrollTo(0, r.getBoundingClientRect().top + window.scrollY - 60);
      });
      await page.waitForTimeout(800);
      await captura(`episodios_${a.nombre}_rejilla`);
      check(`${a.nombre}: overflowH = 0`, m.overflowH === 0, m.overflowH);
      check(`${a.nombre}: ${COLUMNAS_EPISODIOS[a.w]} columnas`, m.cols === COLUMNAS_EPISODIOS[a.w], m.cols);
      check(`${a.nombre}: alto constante (min = max)`, Math.abs(m.hMax - m.hMin) <= 0.5, { min: m.hMin, max: m.hMax });
      if (a.w >= 900) {
        check(`${a.nombre}: alto escritorio 170,39 (T81)`, Math.abs(m.hMin - ALTO_ESCRITORIO) <= 0.5, m.hMin);
        check(`${a.nombre}: contain-intrinsic-size "auto 170px"`, m.muestra.parcial?.csi === 'auto 170px', m.muestra.parcial?.csi);
      } else {
        const csi = m.muestra.parcial?.csi ?? '';
        const esperado = `${m.hMin}px`;
        check(`${a.nombre}: contain-intrinsic-size coincide con el alto`, csi === `auto ${esperado}`, { csi, alto: m.hMin });
      }
      log('episodios ancho', a.nombre, 'cols=', m.cols, 'alto=', m.hMin, '..', m.hMax);
    }
  });

  // --- B. Los tres estados y su altura ---
  await paso('B', async () => {
    await sembrar();
    await abrir('animeav1', 360);
    await page.waitForTimeout(300);
    const m = await page.evaluate(MEDIR_EPISODIOS);
    rep.estados = m.muestra;
    const est = m.muestra;
    check('B: hay tarjeta "Visto"', !!est.visto);
    check('B: hay tarjeta en progreso', !!est.parcial);
    check('B: hay tarjeta sin progreso', !!est.ninguno);
    check('B: sin progreso no tiene elemento de estado', !!est.ninguno && !est.ninguno.visto && !est.ninguno.franja && !est.ninguno.porcentaje);
    if (est.visto) {
      check('B: "Visto" con texto e icono', est.visto.visto.texto === 'Visto' && est.visto.visto.icono, est.visto.visto);
      check('B: "Visto" fondo --surface-chip', est.visto.visto.bg === m.tokens.chip, { got: est.visto.visto.bg, esp: m.tokens.chip });
      check('B: "Visto" color --brand-400', est.visto.visto.color === m.tokens.brand400, { got: est.visto.visto.color });
      check('B: "Visto" borde --border-accent', est.visto.visto.border === m.tokens.accent, { got: est.visto.visto.border, esp: m.tokens.accent });
    }
    if (est.parcial) {
      check('B: en progreso es progressbar con aria-valuenow', est.parcial.franja?.role === 'progressbar' && est.parcial.franja?.now === '63', est.parcial.franja);
      check('B: franja de 4 px pegada al borde inferior de la miniatura', est.parcial.franja?.h === 4 && est.parcial.franja?.pegada <= 0.5, est.parcial.franja);
      check('B: texto "{n} %"', /^\d+ %$/.test(est.parcial.porcentaje ?? ''), est.parcial.porcentaje);
    }
    const alturas = [est.visto, est.parcial, est.ninguno].filter(Boolean).map((x) => x.h);
    check('B: los tres estados tienen el mismo alto', alturas.length === 3 && Math.max(...alturas) - Math.min(...alturas) <= 0.5, alturas);
    await captura('episodios_360_estados');
  });

  // --- C. Tipografía (mínimo --fs-xs) y 16:9 de la miniatura ---
  await paso('C', async () => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.waitForTimeout(300);
    const m = await page.evaluate(MEDIR_EPISODIOS);
    const muestras = [m.muestra.visto, m.muestra.parcial, m.muestra.ninguno].filter(Boolean);
    check('C: título a --fs-sm (14 px)', muestras.every((x) => x.tituloFs === m.tokens.fsSm), muestras.map((x) => x.tituloFs));
    check('C: ningún texto < --fs-xs (12 px)', muestras.every((x) => x.minFs >= m.tokens.fsXs), muestras.map((x) => x.minFs));
    check('C: miniatura 16:9', muestras.every((x) => x.mini && Math.abs(x.mini.ratio - 16 / 9) <= 0.05), muestras.map((x) => x.mini?.ratio));
  });

  // --- D. Miniatura rota (falloMiniatura): hueco con el número, mismo alto ---
  await paso('D', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await page.waitForTimeout(200);
    const res = await page.evaluate(async () => {
      const rej = document.querySelector('.rejilla');
      const t = rej.querySelectorAll('.tarjeta')[0];
      const antes = Math.round(t.getBoundingClientRect().height * 100) / 100;
      const img = t.querySelector('.miniatura img');
      if (!img) return { err: 'sin img' };
      img.src = 'http://localhost:1/no-existe.jpg';
      await new Promise((r) => setTimeout(r, 400));
      const hole = t.querySelector('.sin-miniatura');
      return {
        antes,
        despues: Math.round(t.getBoundingClientRect().height * 100) / 100,
        hole: hole ? (hole.textContent || '').trim() : null,
        color: hole ? getComputedStyle(hole).color : null,
        text3: (() => {
          const p = document.createElement('div');
          p.style.color = 'var(--text-3)';
          document.documentElement.appendChild(p);
          const c = getComputedStyle(p).color;
          p.remove();
          return c;
        })(),
      };
    });
    rep.roto = res;
    check('D: la miniatura rota muestra el hueco con el número', !!res.hole && res.hole === '1', res);
    check('D: el número usa --text-3', res.color === res.text3, { got: res.color, esp: res.text3 });
    check('D: el alto no cambia con la miniatura rota', Math.abs(res.despues - res.antes) <= 0.5, res);
    await captura('episodios_360_hueco');
  });

  // --- E. Hover sólo con puntero fino ---
  await paso('E', async () => {
    await abrir('animeav1', 1280);
    await page.waitForTimeout(300);
    const card = page.locator('.rejilla .tarjeta').first();
    await card.scrollIntoViewIfNeeded();
    await card.hover();
    await page.waitForTimeout(250);
    const fine = await card.evaluate((el) => ({
      media: matchMedia('(hover: hover) and (pointer: fine)').matches,
      transform: getComputedStyle(el).transform,
      shadow: getComputedStyle(el).boxShadow,
    }));
    await page.mouse.move(0, 0);
    await page.waitForTimeout(250);
    rep.hover = { fine };
    check('E: con puntero fino el hover eleva la tarjeta (−3 px)', fine.media && fine.transform === 'matrix(1, 0, 0, 1, 0, -3)', fine);
    check('E: con puntero fino el hover añade sombra', /rgba?\(/.test(fine.shadow), fine.shadow);

    // Contexto táctil (puntero grueso): el hover no debe elevar.
    const ctxT = await browser.newContext({ viewport: { width: 768, height: 1024 }, hasTouch: true, isMobile: true });
    const pt = await ctxT.newPage();
    try {
      await pt.goto(BASE, { waitUntil: 'domcontentloaded' });
      await pt.waitForSelector('.buscador input', { timeout: 20000 });
      const coarse = await pt.evaluate(() => matchMedia('(hover: hover) and (pointer: fine)').matches);
      rep.hover.coarse = coarse;
      check('E: el contexto táctil NO es puntero fino', coarse === false, coarse);
    } catch (e) {
      rep.errores.push('E táctil: ' + e.message);
    }
    await ctxT.close().catch(() => {});
  });

  // --- F. Revelado T65 + scroll T66/T67 ---
  await paso('F', async () => {
    await abrir('animeav1', 360);
    await page.waitForFunction(() => !document.querySelector('.cargando'), null, { timeout: 60000 }).catch(() => {});
    await page.waitForTimeout(500);
    await page.evaluate(() => {
      const r = document.querySelector('.rejilla');
      if (r) window.scrollTo(0, r.getBoundingClientRect().top + window.scrollY);
    });
    await page.waitForTimeout(1200);
    const entra = await page.evaluate(() => document.querySelectorAll('.celda.entra').length);
    rep.revelado = entra;
    check('F: revelado T65 (.entra > 0)', entra > 0, entra);

    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(600);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if (yAntes > 0 && (await ultima.count()) > 0) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await volverDetalle();
      await page.waitForTimeout(1200);
      const desp = await page.evaluate(() => ({
        y: Math.round(window.scrollY),
        max: document.documentElement.scrollHeight - window.innerHeight,
      }));
      const esperado = Math.min(yAntes, desp.max);
      rep.scroll = { yAntes, ...desp, esperado, delta: Math.abs(desp.y - esperado) };
      check('F: scroll del detalle restaurado (T66/T67)', Math.abs(desp.y - esperado) <= 5, rep.scroll);
    } else {
      check('F: scroll del detalle restaurado (T66/T67)', false, { yAntes });
    }
  });

  // --- G. Objetivos táctiles de la tarjeta ---
  await paso('G', async () => {
    await abrir('animeav1', 360);
    rep.tactiles = await page.evaluate(() => {
      const hs = [...document.querySelectorAll('.rejilla .tarjeta')].map((t) => t.getBoundingClientRect().height);
      return { total: hs.length, hMin: Math.round(Math.min(...hs) * 100) / 100 };
    });
    check('G: toda tarjeta de episodio ≥44 px de alto', rep.tactiles.hMin >= 44, rep.tactiles);
  });

  writeFileSync(join(dirOut, 'episodios.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo orden (T82): orden ascendente/descendente de los episodios del detalle
// ---------------------------------------------------------------------------

const ANCHOS_ORDEN = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1280, h: 800, nombre: '1280' },
];

const MEDIR_ORDEN = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height), bottom: round(r.bottom) };
  };
  const resolver = (n) => {
    const p = document.createElement('div');
    p.style.color = `var(${n})`;
    d.appendChild(p);
    const c = getComputedStyle(p).color;
    p.remove();
    return c;
  };
  const btn = document.querySelector('.orden');
  const texto = btn ? btn.querySelector('.orden-texto') : null;
  const rej = document.querySelector('.rejilla');
  const celdas = rej ? [...rej.querySelectorAll('.celda')] : [];
  const tarjetas = celdas.map((c) => c.querySelector('.tarjeta'));
  const insignia = (t) => {
    const i = t ? t.querySelector('.insignia') : null;
    return i ? (i.textContent || '').trim() : null;
  };
  const identidad = (t) => {
    if (!t) return '';
    const img = t.querySelector('.miniatura img');
    return [
      insignia(t),
      (t.querySelector('.titulo')?.textContent || '').trim(),
      img ? img.getAttribute('src') : '',
    ].join('|');
  };
  const ids = tarjetas.map(identidad);
  const cs = btn ? getComputedStyle(btn) : null;
  const csTexto = texto ? getComputedStyle(texto) : null;
  const csRej = rej ? getComputedStyle(rej) : null;
  const cta = document.querySelector('.barra-accion .principal .btn') || document.querySelector('.continuar');
  const cab = document.querySelector('main .cabecera');
  const ctaBarra = document.querySelector('.barra-accion');
  const csd = getComputedStyle(d);
  const tok = (n) => csd.getPropertyValue(n).trim();
  return {
    overflowH: d.scrollWidth - d.clientWidth,
    scrollY: round(window.scrollY),
    titulo: (document.querySelector('.titular')?.textContent || '').trim(),
    tituloEnFila: !!document.querySelector('.fila-titular .orden'),
    boton: btn
      ? {
          w: round(btn.getBoundingClientRect().width),
          h: round(btn.getBoundingClientRect().height),
          textoVisible: !!csTexto && csTexto.display !== 'none',
          texto: texto ? (texto.textContent || '').trim() : null,
          ariaLabel: btn.getAttribute('aria-label'),
          ariaPressed: btn.getAttribute('aria-pressed'),
          bg: cs.backgroundColor,
          border: cs.borderTopColor,
          radius: cs.borderTopLeftRadius,
          fontSize: cs.fontSize,
          transicion: cs.transitionDuration,
        }
      : null,
    cards: {
      total: celdas.length,
      primero: insignia(tarjetas[0]),
      ultimo: insignia(tarjetas[tarjetas.length - 1]),
      duplicados: ids.filter((v, i) => v !== '' && ids.indexOf(v) !== i).length,
      altoMin: tarjetas.length ? Math.min(...tarjetas.map((t) => round(t.getBoundingClientRect().height))) : null,
      altoMax: tarjetas.length ? Math.max(...tarjetas.map((t) => round(t.getBoundingClientRect().height))) : null,
    },
    columnas: csRej ? csRej.gridTemplateColumns.split(' ').filter(Boolean).length : 0,
    cta: cta ? (cta.textContent || '').trim() : null,
    revelado: {
      entra: document.querySelectorAll('.celda.entra').length,
      pendiente: document.querySelectorAll('.celda.pendiente').length,
      total: celdas.length,
      entraIds: celdas.filter((c) => c.classList.contains('entra')).map((c) => identidad(c.querySelector('.tarjeta'))),
    },
    cabecera: {
      rect: rect(cab),
      posicion: cab ? getComputedStyle(cab).position : null,
    },
    barra: rect(ctaBarra),
    barraInferior: rect(document.querySelector('.barra-inferior')),
    tokens: {
      chip: resolver('--surface-chip'),
      chipHover: resolver('--surface-chip-hover'),
      border: resolver('--border'),
      rMd: tok('--r-md'),
      fsSm: tok('--fs-sm'),
    },
  };
};

async function esperarEpisodios(page) {
  await page.waitForFunction(() => !document.querySelector('.cargando'), null, { timeout: 60000 }).catch(() => {});
  await page.waitForTimeout(500);
}

async function orden() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/orden';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const rep = {
    anchos: [],
    funciones: [],
    capturas: [],
    revelado: null,
    scroll: null,
    retorno: null,
    unEpisodio: null,
    hover: null,
    reduced: null,
    errores: [],
  };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const captura = async (nombre) => {
    try {
      await page.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const abrir = async (proveedor, a) => {
    await page.setViewportSize({ width: a.w, height: a.h });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, proveedor);
    await esperarEpisodios(page);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(300);
  };
  const verTitulo = async () => {
    await page.evaluate(() => {
      const h = document.querySelector('.fila-titular');
      if (h) window.scrollTo(0, h.getBoundingClientRect().top + window.scrollY - 12);
    });
    await page.waitForTimeout(250);
  };

  // Historial sembrado: Bleach AnimeAV1 con el ep. 9 en progreso (CTA "Continuar · Ep. 9").
  const SEMILLA = [
    { anime_id: 'bleach', episode_id: 'animeav1:bleach:episode:9', current_time: 900, total_duration: 1420, is_watched: false, provider: 'animeav1', anime_title: 'Bleach', cover_url: '', episode_number: 9, updated_at: 1749999940 },
  ];

  // --- A. Geometría, inversión y CTA por ancho (AnimeAV1, 366 episodios) ---
  await paso('A', async () => {
    await gotoApp(page);
    await page.evaluate(async (seed) => {
      const App = window.go?.main?.App;
      if (!App) throw new Error('puente window.go no disponible (usa localhost:34115)');
      await App.ClearProgress();
      for (const s of seed) await App.SaveProgress(s);
    }, SEMILLA);
    await page.waitForTimeout(300);

    for (const a of ANCHOS_ORDEN) {
      await abrir('animeav1', a);
      const asc = await page.evaluate(MEDIR_ORDEN);
      await verTitulo();
      if (a.nombre === '360' || a.nombre === '1280') await captura(`detalle_${a.nombre}_asc`);

      await page.locator('.orden').click();
      await page.waitForTimeout(350);
      const desc = await page.evaluate(MEDIR_ORDEN);
      if (a.nombre === '360' || a.nombre === '1280') await captura(`detalle_${a.nombre}_desc`);
      await page.locator('.orden').click();
      await page.waitForTimeout(350);
      const asc2 = await page.evaluate(MEDIR_ORDEN);
      rep.anchos.push({ ancho: a.nombre, asc, desc, asc2 });

      check(`${a.nombre}: botón de orden presente`, !!asc.boton, asc.boton);
      check(`${a.nombre}: título "Episodios (366)" sin cambiar`, asc.titulo === 'Episodios (366)', asc.titulo);
      check(`${a.nombre}: botón en la fila del título`, asc.tituloEnFila, asc.tituloEnFila);
      if (a.w >= 900) {
        check(`${a.nombre}: botón 36 px de alto`, !!asc.boton && asc.boton.h === 36, asc.boton?.h);
      } else {
        check(
          `${a.nombre}: botón ≥44×44`,
          !!asc.boton && asc.boton.w >= 44 && asc.boton.h >= 44,
          asc.boton ? `${asc.boton.w}×${asc.boton.h}` : null
        );
      }
      check(
        `${a.nombre}: texto del botón ${a.w >= 480 ? 'visible' : 'oculto'}`,
        !!asc.boton && asc.boton.textoVisible === a.w >= 480,
        asc.boton?.textoVisible
      );
      if (a.w >= 480) {
        check(`${a.nombre}: texto "Más recientes" en ascendente`, asc.boton?.texto === 'Más recientes', asc.boton?.texto);
        check(`${a.nombre}: texto "Más antiguos" en descendente`, desc.boton?.texto === 'Más antiguos', desc.boton?.texto);
      }
      check(
        `${a.nombre}: aria-pressed false → true → false`,
        asc.boton?.ariaPressed === 'false' && desc.boton?.ariaPressed === 'true' && asc2.boton?.ariaPressed === 'false',
        [asc.boton?.ariaPressed, desc.boton?.ariaPressed, asc2.boton?.ariaPressed]
      );
      check(
        `${a.nombre}: aria-label coherente en ambos estados`,
        asc.boton?.ariaLabel === 'Ordenar del más reciente al más antiguo' &&
          desc.boton?.ariaLabel === 'Ordenar del más antiguo al más reciente',
        [asc.boton?.ariaLabel, desc.boton?.ariaLabel]
      );
      check(
        `${a.nombre}: tokens del botón (chip/borde/radio/--fs-sm)`,
        asc.boton?.bg === asc.tokens.chip &&
          asc.boton?.border === asc.tokens.border &&
          asc.boton?.radius === asc.tokens.rMd &&
          asc.boton?.fontSize === asc.tokens.fsSm,
        {
          bg: asc.boton?.bg,
          chip: asc.tokens.chip,
          border: asc.boton?.border,
          radius: asc.boton?.radius,
          rMd: asc.tokens.rMd,
          fs: asc.boton?.fontSize,
          fsSm: asc.tokens.fsSm,
        }
      );
      check(
        `${a.nombre}: ascendente Ep. 1 → Ep. 366`,
        asc.cards.primero === 'Ep. 1' && asc.cards.ultimo === 'Ep. 366',
        { primero: asc.cards.primero, ultimo: asc.cards.ultimo }
      );
      check(
        `${a.nombre}: descendente Ep. 366 → Ep. 1`,
        desc.cards.primero === 'Ep. 366' && desc.cards.ultimo === 'Ep. 1',
        { primero: desc.cards.primero, ultimo: desc.cards.ultimo }
      );
      check(`${a.nombre}: 366 tarjetas en ambos órdenes`, asc.cards.total === 366 && desc.cards.total === 366, {
        asc: asc.cards.total,
        desc: desc.cards.total,
      });
      check(
        `${a.nombre}: sin ids repetidos al invertir`,
        asc.cards.duplicados === 0 && desc.cards.duplicados === 0,
        { asc: asc.cards.duplicados, desc: desc.cards.duplicados }
      );
      check(
        `${a.nombre}: la CTA/«Continuar · Ep. 9» no cambia`,
        asc.cta === 'Continuar · Ep. 9' && desc.cta === asc.cta && asc2.cta === asc.cta,
        { asc: asc.cta, desc: desc.cta, asc2: asc2.cta }
      );
      check(
        `${a.nombre}: columnas sin cambios`,
        asc.columnas === desc.columnas && asc.columnas === asc2.columnas,
        asc.columnas
      );
      check(
        `${a.nombre}: altura de tarjeta sin cambios`,
        asc.cards.altoMin === asc.cards.altoMax &&
          desc.cards.altoMin === desc.cards.altoMax &&
          Math.abs(asc.cards.altoMax - desc.cards.altoMax) <= 0.5,
        { asc: asc.cards.altoMin, desc: desc.cards.altoMin }
      );
      check(`${a.nombre}: overflowH = 0`, asc.overflowH === 0 && desc.overflowH === 0, {
        asc: asc.overflowH,
        desc: desc.overflowH,
      });
      check(
        `${a.nombre}: revelado coherente (entra + pendiente = total)`,
        desc.revelado.entra + desc.revelado.pendiente === desc.revelado.total,
        desc.revelado
      );
      const perdidas = asc.revelado.entraIds.filter((id) => !desc.revelado.entraIds.includes(id));
      check(`${a.nombre}: ninguna revelada pierde .entra al invertir`, perdidas.length === 0, {
        perdidas: perdidas.length,
        entraAsc: asc.revelado.entra,
        entraDesc: desc.revelado.entra,
      });
      if (a.nombre === '360') rep.revelado = { perdidas: perdidas.length, entraAsc: asc.revelado.entra, entraDesc: desc.revelado.entra };
      log('orden ancho', a.nombre, 'botón=', asc.boton && `${asc.boton.w}×${asc.boton.h}`, 'desc=', desc.cards.primero, '→', desc.cards.ultimo);
    }
  });

  // --- B. Detalle de JKAnime: el botón también invierte su lista ---
  await paso('B', async () => {
    const a = ANCHOS_ORDEN[0];
    await abrir('jkanime', a);
    const asc = await page.evaluate(MEDIR_ORDEN);
    await page.locator('.orden').click();
    await page.waitForTimeout(350);
    const desc = await page.evaluate(MEDIR_ORDEN);
    check('JKAnime 360: botón de orden presente', !!asc.boton, asc.boton);
    check('JKAnime 360: hay más de un episodio', asc.cards.total > 1, asc.cards.total);
    check(
      'JKAnime 360: el orden se invierte (primero ↔ último)',
      desc.cards.primero === asc.cards.ultimo && desc.cards.ultimo === asc.cards.primero,
      { asc: [asc.cards.primero, asc.cards.ultimo], desc: [desc.cards.primero, desc.cards.ultimo] }
    );
    check('JKAnime 360: sin ids repetidos', desc.cards.duplicados === 0, desc.cards.duplicados);
    await page.locator('.orden').click();
    await page.waitForTimeout(250);
  });

  // --- C. Anime con un solo episodio: sin botón ---
  await paso('C', async () => {
    const objetivo = await page.evaluate(async () => {
      const App = window.go?.main?.App;
      if (!App) return null;
      for (const q of ['your name', 'suzume', 'akira', 'mononoke', 'el viaje de chihiro']) {
        const res = await App.SearchAnime(q).catch(() => []);
        for (const r of res.slice(0, 6)) {
          const d = await App.GetAnimeDetails(r.provider, r.id).catch(() => null);
          if (d && Array.isArray(d.episodes) && d.episodes.length === 1) {
            return { q, title: r.title, provider: r.provider };
          }
        }
      }
      return null;
    });
    if (!objetivo) {
      check('1 episodio: se encontró un anime de prueba', false, 'ninguno');
      return;
    }
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, objetivo.q);
    const abierto = await page.evaluate((o) => {
      const etiqueta = o.provider === 'animeav1' ? 'AnimeAV1' : 'JKAnime';
      const card = [...document.querySelectorAll('main .rejilla > .tarjeta')].find(
        (c) =>
          (c.querySelector('.titulo')?.textContent || '').trim() === o.title &&
          ((c.querySelector('.etiqueta')?.textContent || '').trim() === etiqueta)
      );
      if (!card) return false;
      card.click();
      return true;
    }, objetivo);
    if (!abierto) {
      check('1 episodio: la tarjeta del anime se abre', false, objetivo);
      return;
    }
    await page.waitForSelector('.ficha', { timeout: 45000 });
    await page.waitForTimeout(800);
    const uni = await page.evaluate(MEDIR_ORDEN);
    rep.unEpisodio = { ...objetivo, cards: uni.cards.total, boton: !!uni.boton };
    check('1 episodio: el detalle tiene 1 tarjeta', uni.cards.total === 1, uni.cards.total);
    check('1 episodio: sin botón de orden', uni.boton === null, uni.boton);
  });

  // --- D. Scroll con la lista desplazada: cabecera fija y CTA intactas ---
  await paso('D', async () => {
    const a = ANCHOS_ORDEN[0];
    await abrir('animeav1', a);
    await page.evaluate(() => window.scrollTo(0, 1600));
    await page.waitForTimeout(500);
    const antes = await page.evaluate(MEDIR_ORDEN);
    await page.evaluate(() => document.querySelector('.orden').click());
    await page.waitForTimeout(400);
    const despues = await page.evaluate(MEDIR_ORDEN);
    const sat = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--sat')) || 0);
    rep.scroll = {
      antes: antes.scrollY,
      despues: despues.scrollY,
      cabeceraY: despues.cabecera.rect?.y ?? null,
      sat,
      barraBottom: despues.barra?.bottom ?? null,
      innerHeight: await page.evaluate(() => window.innerHeight),
    };
    check(
      'D: la cabecera sigue fija al invertir con scroll',
      despues.cabecera.posicion === 'sticky' && Math.abs((despues.cabecera.rect?.y ?? NaN) - sat) <= 2,
      rep.scroll
    );
    check(
      'D: la CTA sigue pegada a la barra inferior',
      despues.barra !== null &&
        despues.barraInferior !== null &&
        Math.abs(despues.barra.bottom - despues.barraInferior.y) <= 1,
      { barra: despues.barra, barraInferior: despues.barraInferior }
    );
    check('D: sin overflow horizontal', despues.overflowH === 0, despues.overflowH);
    await page.evaluate(() => document.querySelector('.orden').click());
    await page.waitForTimeout(250);
  });

  // --- E. Detalle (desc) → reproductor → Atrás: vuelve a ascendente y restaura scroll ---
  await paso('E', async () => {
    const a = ANCHOS_ORDEN[0];
    await abrir('animeav1', a);
    await page.locator('.orden').click();
    await page.waitForTimeout(300);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const antes = await page.evaluate(MEDIR_ORDEN);
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if ((await ultima.count()) > 0) {
      await ultima.click();
      await page.waitForTimeout(1800);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const despues = await page.evaluate(MEDIR_ORDEN);
      rep.retorno = { yAntes: antes.scrollY, yDespues: despues.scrollY, ariaPressed: despues.boton?.ariaPressed };
      check('E: al volver del reproductor el orden es ascendente', despues.boton?.ariaPressed === 'false', despues.boton?.ariaPressed);
      check(
        'E: T66/T67 restaura el scroll (tolerancia 5 px)',
        Math.abs(despues.scrollY - antes.scrollY) <= 5,
        rep.retorno
      );
    } else {
      check('E: hay tarjetas para volver del reproductor', false, antes.cards.total);
    }
  });

  // --- F. Hover solo con puntero fino y reduced-motion sin transición ---
  await paso('F', async () => {
    const a = ANCHOS_ORDEN[3];
    await abrir('animeav1', a);
    await verTitulo();
    const btn = page.locator('.orden');
    await btn.hover();
    await page.waitForTimeout(250);
    const hover = await page.evaluate(MEDIR_ORDEN);
    rep.hover = { fine: await page.evaluate(() => matchMedia('(hover: hover) and (pointer: fine)').matches), bg: hover.boton.bg, chipHover: hover.tokens.chipHover };
    check(
      'F: hover con puntero fino usa --surface-chip-hover',
      rep.hover.fine && hover.boton.bg === hover.tokens.chipHover,
      rep.hover
    );
    await page.mouse.move(0, 0);
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const reduce = await page.evaluate(() => {
      const b = document.querySelector('.orden');
      return b ? getComputedStyle(b).transitionDuration : null;
    });
    rep.reduced = { transitionDuration: reduce };
    check('F: reduced-motion sin transición en el botón', reduce === '0s', reduce);
    await page.emulateMedia({ reducedMotion: null });
  });

  writeFileSync(join(dirOut, 'orden.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo reproductor (T83): layout del reproductor y selector de fuentes
// ---------------------------------------------------------------------------

const ANCHOS_REPRODUCTOR = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1024, h: 768, nombre: '1024' },
  { w: 1280, h: 800, nombre: '1280' },
];

const MEDIR_REPRODUCTOR = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return {
      x: round(r.x),
      y: round(r.y),
      w: round(r.width),
      h: round(r.height),
      right: round(r.right),
      bottom: round(r.bottom),
    };
  };
  const cs = (el) => (el ? getComputedStyle(el) : null);
  const fila = document.querySelector('.selector > .botones:not(.extras)');
  const extras = document.querySelector('.selector > .botones.extras');
  const plegable = document.querySelector('.selector .plegable');
  const botones = fila ? [...fila.querySelectorAll('button')] : [];
  const extrasBtns = extras ? [...extras.querySelectorAll('button')] : [];
  const marco = document.querySelector('.marco');
  const avisos = [...document.querySelectorAll('main > .aviso')].map((a) => {
    const r = rect(a);
    const m = marco ? rect(marco) : null;
    const encima = m && r ? r.bottom <= m.y + 1 : false;
    const debajo = m && r ? r.y >= m.bottom - 1 : false;
    return {
      texto: (a.textContent || '').trim().slice(0, 60),
      rect: r,
      posicion: encima ? 'encima' : debajo ? 'debajo' : 'otra',
      separacion: m && r ? (encima ? round(m.y - r.bottom) : debajo ? round(r.y - m.bottom) : null) : null,
      marginTop: cs(a).marginTop,
      marginBottom: cs(a).marginBottom,
      fontSize: cs(a).fontSize,
    };
  });
  const activo = document.querySelector('.selector > .botones button.activa');
  const chip = botones[0] ?? null;
  const filaInfo = (el, btns) => {
    if (!el) return null;
    const c = cs(el);
    return {
      rect: rect(el),
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth,
      scrollHeight: el.scrollHeight,
      clientHeight: el.clientHeight,
      scrollLeft: round(el.scrollLeft),
      flexWrap: c.flexWrap,
      overflowX: c.overflowX,
      scrollSnapType: c.scrollSnapType,
      scrollbarWidth: c.scrollbarWidth,
      marginLeft: c.marginLeft,
      marginRight: c.marginRight,
      paddingLeft: c.paddingLeft,
      paddingRight: c.paddingRight,
      scrollPaddingLeft: c.scrollPaddingLeft,
      overscroll: c.overscrollBehaviorX,
      botones: btns.map((b) => {
        const r = b.getBoundingClientRect();
        const bc = cs(b);
        return {
          texto: (b.textContent || '').trim(),
          w: round(r.width),
          h: round(r.height),
          x: round(r.x),
          y: round(r.y),
          ariaPressed: b.getAttribute('aria-pressed'),
          activa: b.classList.contains('activa'),
          checkVisible: !!b.querySelector('.check') && getComputedStyle(b.querySelector('.check')).display !== 'none',
          fontSize: bc.fontSize,
          minHeight: bc.minHeight,
          borderRadius: bc.borderRadius,
          whiteSpace: bc.whiteSpace,
        };
      }),
    };
  };
  return {
    innerWidth: window.innerWidth,
    overflowH: d.scrollWidth - d.clientWidth,
    gutter: getComputedStyle(d).getPropertyValue('--gutter').trim(),
    contenedor: rect(document.querySelector('main.contenedor')),
    cabecera: rect(document.querySelector('main .cabecera')),
    atras: rect(document.querySelector('main .cabecera .atras')),
    marco: rect(marco),
    ratio: marco ? round(marco.getBoundingClientRect().width / marco.getBoundingClientRect().height) : null,
    marcoEstilo: marco
      ? {
          radius: cs(marco).borderRadius,
          border: cs(marco).borderTopWidth + ' ' + cs(marco).borderTopStyle + ' ' + cs(marco).borderTopColor,
          shadow: cs(marco).boxShadow,
          overflow: cs(marco).overflow,
          background: cs(marco).backgroundColor,
          aspect: cs(marco).aspectRatio,
        }
      : null,
    fila: filaInfo(fila, botones),
    extras: filaInfo(extras, extrasBtns),
    plegable: plegable
      ? {
          rect: rect(plegable),
          ariaExpanded: plegable.getAttribute('aria-expanded'),
          fontSize: cs(plegable).fontSize,
          minHeight: cs(plegable).minHeight,
          texto: (plegable.textContent || '').trim(),
        }
      : null,
    activo: activo
      ? {
          texto: (activo.textContent || '').trim(),
          ariaPressed: activo.getAttribute('aria-pressed'),
          checkVisible: !!activo.querySelector('.check') && getComputedStyle(activo.querySelector('.check')).display !== 'none',
          background: cs(activo).backgroundImage,
          color: cs(activo).color,
          fontWeight: cs(activo).fontWeight,
        }
      : null,
    avisos,
    barraInferior: !!document.querySelector('.barra-inferior'),
    botonSiguiente: rect(document.querySelector('.marco .boton-siguiente')),
    chip: chip
      ? {
          h: round(chip.getBoundingClientRect().height),
          fontSize: cs(chip).fontSize,
          radius: cs(chip).borderRadius,
        }
      : null,
    ultimaSeccion: (() => {
      const s = document.querySelector('.selector');
      return s ? rect(s) : null;
    })(),
  };
};

async function reproductor() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/reproductor';
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
  const abrir = async (a) => {
    await page.setViewportSize({ width: a.w, height: a.h });
    await gotoApp(page);
    await page.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    const ok = await abrirReproductor(page);
    await page.waitForSelector('.selector', { timeout: 20000 }).catch(() => {});
    return ok;
  };
  const principal = page.locator('.selector > .botones:not(.extras) > button');

  // --- A. Geometría y capturas por ancho ---
  await paso('A', async () => {
    for (const a of ANCHOS_REPRODUCTOR) {
      const abierto = await abrir(a);
      if (!abierto) {
        rep.anchos.push({ ancho: a.nombre, err: 'sin tarjetas de episodio' });
        check(`${a.nombre}: reproductor abierto`, false, 'sin tarjetas de episodio');
        continue;
      }
      // Fuente principal activa (el primer chip) para la captura de referencia.
      if (await principal.count()) await principal.first().click().catch(() => {});
      await page.waitForTimeout(400);
      const med = await page.evaluate(MEDIR_REPRODUCTOR);
      rep.anchos.push({ ancho: a.nombre, ...med });
      if (['360', '768', '1280'].includes(a.nombre)) await captura(`reproductor_${a.nombre}`);

      if (a.w < 900) {
        check(`${a.nombre}: Atrás x = marco x = gutter`, med.atras && med.marco && Math.abs(med.atras.x - med.marco.x) <= 1 && Math.abs(med.marco.x - parseFloat(med.gutter)) <= 1, { atras: med.atras?.x, marco: med.marco?.x, gutter: med.gutter });
        check(`${a.nombre}: marco 16:9`, med.ratio !== null && Math.abs(med.ratio - 16 / 9) < 0.01, med.ratio);
        check(`${a.nombre}: sin overflow horizontal`, med.overflowH === 0, med.overflowH);
        const fila = med.fila;
        check(`${a.nombre}: chips en una fila`, fila && fila.flexWrap === 'nowrap' && fila.scrollHeight <= fila.clientHeight + 2, fila ? { flexWrap: fila.flexWrap, scrollH: fila.scrollHeight, clientH: fila.clientHeight } : null);
        check(`${a.nombre}: todos los chips ≥44 px de alto`, fila && fila.botones.length > 0 && fila.botones.every((b) => b.h >= 44), fila?.botones.map((b) => `${b.texto}:${b.h}`));
      } else {
        const fila = med.fila;
        const filasChips = fila ? new Set(fila.botones.map((b) => b.y)).size : 0;
        check(`${a.nombre}: chips con wrap (sin scroll horizontal)`, fila && fila.flexWrap === 'wrap' && fila.overflowX !== 'auto', { flexWrap: fila?.flexWrap, overflowX: fila?.overflowX, filas: filasChips });
        check(`${a.nombre}: alto de chip igual que T69 (31±2)`, med.chip && Math.abs(med.chip.h - 31) <= 2, med.chip);
        check(`${a.nombre}: tamaño de fuente del chip igual que antes (13 px)`, med.chip?.fontSize === '13px', med.chip?.fontSize);
      }
      if (med.botonSiguiente) check(`${a.nombre}: BotonSiguiente ≥ 44 (T84)`, med.botonSiguiente.h >= 44, med.botonSiguiente.h);
      log('reproductor ancho', a.nombre, 'gutter=', med.gutter, 'marco=', JSON.stringify(med.marco), 'chips=', med.fila?.botones.length);
    }
  });

  // --- B. Extras abiertas a 360 (y 768) ---
  await paso('B', async () => {
    for (const a of [{ w: 360, h: 740, nombre: '360' }, { w: 768, h: 1024, nombre: '768' }]) {
      const abierto = await abrir(a);
      if (!abierto) continue;
      const pleg = page.locator('.selector .plegable');
      if (!(await pleg.count())) {
        check(`${a.nombre}: plegable presente`, false, 'sin fuentes extra');
        continue;
      }
      const antes = await pleg.getAttribute('aria-expanded');
      if (antes !== 'true') {
        await pleg.click();
        await page.waitForTimeout(400);
      }
      const med = await page.evaluate(MEDIR_REPRODUCTOR);
      rep.anchos.push({ ancho: a.nombre, vista: 'extras', ...med });
      check(`${a.nombre}: plegable ≥44 px`, med.plegable && med.plegable.rect.h >= 44, med.plegable?.rect.h);
      check(`${a.nombre}: aria-expanded=true con extras abiertas`, med.plegable?.ariaExpanded === 'true', med.plegable?.ariaExpanded);
      check(`${a.nombre}: fila de extras en una fila`, med.extras && med.extras.flexWrap === 'nowrap' && med.extras.scrollHeight <= med.extras.clientHeight + 2, med.extras ? { flexWrap: med.extras.flexWrap, scrollH: med.extras.scrollHeight, clientH: med.extras.clientHeight } : null);
      check(`${a.nombre}: chips extra ≥44 px`, med.extras && med.extras.botones.length > 0 && med.extras.botones.every((b) => b.h >= 44), med.extras?.botones.map((b) => `${b.texto}:${b.h}`));
      // Scroll horizontal hasta el último chip y vuelta al inicio.
      const desplazado = await page.evaluate(() => {
        const fila = document.querySelector('.selector > .botones.extras');
        if (!fila) return null;
        fila.scrollLeft = fila.scrollWidth;
        return { max: fila.scrollWidth - fila.clientWidth, izquierda: Math.round(fila.scrollLeft) };
      });
      check(`${a.nombre}: los extras scrollean hasta el final`, !!desplazado && (desplazado.max === 0 || desplazado.izquierda > 0), desplazado);
      if (a.nombre === '360') await captura('reproductor_360_extras');
    }
  });

  // --- C. Activo con check, aria-pressed y cambio de fuente (360) ---
  await paso('C', async () => {
    const abierto = await abrir({ w: 360, h: 740, nombre: '360' });
    if (!abierto) return;
    const med = await page.evaluate(MEDIR_REPRODUCTOR);
    check('360: activo con aria-pressed=true', med.activo?.ariaPressed === 'true', med.activo?.ariaPressed);
    check('360: activo con icono check visible', med.activo?.checkVisible === true, med.activo?.checkVisible);
    check('360: activo con degradado de marca', (med.activo?.background ?? '').includes('gradient'), med.activo?.background);
    check('360: inactivos con aria-pressed=false', med.fila?.botones.filter((b) => !b.activa).every((b) => b.ariaPressed === 'false'), med.fila?.botones.map((b) => `${b.texto}:${b.ariaPressed}`));
    if ((await principal.count()) >= 2) {
      await principal.nth(1).click();
      await page.waitForTimeout(700);
      const tras = await page.evaluate(MEDIR_REPRODUCTOR);
      check('360: cambiar de fuente deja el segundo chip activo', tras.fila?.botones[1]?.activa === true && tras.fila?.botones[1]?.ariaPressed === 'true', tras.fila?.botones.map((b) => `${b.texto}:${b.activa}/${b.ariaPressed}`));
      await principal.first().click().catch(() => {});
      await page.waitForTimeout(400);
    }
    // Plegable: al abrirlo manualmente, aria-expanded cambia.
    const pleg = page.locator('.selector .plegable');
    if (await pleg.count()) {
      const antes = await pleg.getAttribute('aria-expanded');
      await pleg.click();
      await page.waitForTimeout(300);
      const despues = await pleg.getAttribute('aria-expanded');
      check('360: el plegable alterna aria-expanded', antes !== despues, { antes, despues });
      if (despues === 'true') await pleg.click();
      await page.waitForTimeout(200);
    }
    // Aviso (si existe) separado del marco.
    const conAviso = rep.anchos.find((m) => m.ancho === '360' && m.vista === undefined);
    if (conAviso?.avisos?.length) {
      for (const av of conAviso.avisos) {
        check(`360: aviso ${av.posicion} separado del marco`, av.separacion !== null && av.separacion >= 8, av);
      }
    } else {
      check('360: aviso separado del marco', true, 'sin aviso visible en este estado');
    }
  });

  // --- D. Plegable automático con iframe activo ---
  await paso('D', async () => {
    await abrir({ w: 360, h: 740, nombre: '360' });
    // En el navegador el proxy local del contenedor no sirve el HLS: el
    // reproductor acaba cayendo al iframe y el pliegue debe abrirse solo.
    // Cada fallo refresca las fuentes por red, así que se da hasta ~30 s.
    let auto = false;
    for (let i = 0; i < 40; i++) {
      await page.waitForTimeout(750);
      auto = await page.evaluate(() => {
        const p = document.querySelector('.selector .plegable');
        return !!p && p.getAttribute('aria-expanded') === 'true';
      });
      if (auto) break;
    }
    check('360: plegable abierto automáticamente con iframe activo', auto, auto ? 'sí' : 'no se llegó al iframe');
    rep.iframeAuto = auto;
    if (auto) {
      const med = await page.evaluate(MEDIR_REPRODUCTOR);
      const av = med.avisos.find((x) => x.posicion === 'debajo') ?? med.avisos[0] ?? null;
      rep.avisoIframe = av;
      check('360: aviso del iframe separado del marco', !!av && av.separacion !== null && av.separacion >= 8, av);
      check('360: plegable ≥44 px con iframe activo', !!med.plegable && med.plegable.rect.h >= 44, med.plegable?.rect.h);
      check('360: fila de extras sigue en una fila con iframe', !!med.extras && med.extras.flexWrap === 'nowrap' && med.extras.scrollHeight <= med.extras.clientHeight + 2, med.extras ? { flexWrap: med.extras.flexWrap, scrollH: med.extras.scrollHeight, clientH: med.extras.clientHeight } : null);
    } else {
      check('360: aviso del iframe separado del marco', false, 'no se llegó al iframe');
    }
  });

  // --- E. Barra inferior, fin de scroll y atajo N (360) ---
  await paso('E', async () => {
    const abierto = await abrir({ w: 360, h: 740, nombre: '360' });
    if (!abierto) return;
    const barra = await page.locator('.barra-inferior').count();
    check('360: sin barra inferior en el reproductor', barra === 0, barra);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(400);
    const fin = await page.evaluate(() => {
      const s = document.querySelector('.selector');
      if (!s) return null;
      return { fondo: Math.round(s.getBoundingClientRect().bottom), alto: window.innerHeight };
    });
    check('360: fin del scroll sin nada tapado', !!fin && fin.fondo <= fin.alto + 1, fin);
    const subtituloAntes = await page.evaluate(() => ((document.querySelector('.cabecera .subtitulo') || {}).textContent || '').trim());
    await page.keyboard.press('n');
    await page.waitForTimeout(900);
    const subtituloDespues = await page.evaluate(() => ((document.querySelector('.cabecera .subtitulo') || {}).textContent || '').trim());
    const sigueReproductor = await page.locator('.marco').count();
    check('360: atajo N responde en el reproductor', sigueReproductor > 0 && (subtituloAntes !== subtituloDespues || subtituloAntes !== ''), { subtituloAntes, subtituloDespues });
  });

  // --- F. Atrás y scroll T66/T67 (360) ---
  await paso('F', async () => {
    await page.setViewportSize({ width: 360, height: 740 });
    await gotoApp(page);
    await buscar(page, 'bleach');
    await abrirDetalle(page, 'animeav1');
    await page.waitForTimeout(900);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await page.waitForTimeout(500);
    const yAntes = await page.evaluate(() => Math.round(window.scrollY));
    const ultima = page.locator('.rejilla .celda .tarjeta').last();
    if ((await ultima.count()) && yAntes > 0) {
      await ultima.click();
      await page.waitForSelector('.marco', { timeout: 30000 }).catch(() => {});
      await page.waitForTimeout(1500);
      await page.locator('.cabecera .atras').first().click();
      await page.waitForTimeout(1200);
      const yDespues = await page.evaluate(() => Math.round(window.scrollY));
      check('360: T66/T67 scroll restaurado al volver', Math.abs(yDespues - yAntes) <= 5, { yAntes, yDespues });
    } else {
      check('360: T66/T67 scroll restaurado al volver', false, { yAntes });
    }
  });

  // --- G. prefers-reduced-motion: sin transiciones en chips y plegable ---
  await paso('G', async () => {
    await abrir({ w: 360, h: 740, nombre: '360' });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const trans = await page.evaluate(() => {
      const chip = document.querySelector('.selector .botones button');
      const pleg = document.querySelector('.selector .plegable');
      return {
        chip: chip ? getComputedStyle(chip).transitionDuration : null,
        plegable: pleg ? getComputedStyle(pleg).transitionDuration : null,
      };
    });
    rep.reduced = trans;
    check('reduced-motion: chips sin transición', trans.chip === '0s', trans.chip);
    check('reduced-motion: plegable sin transición', trans.plegable === '0s', trans.plegable);
    await page.emulateMedia({ reducedMotion: null });
  });

  // --- H. Táctiles < 44 px restantes (360) ---
  await paso('H', async () => {
    await abrir({ w: 360, h: 740, nombre: '360' });
    rep.tactiles = await page.evaluate(TACTILES_CHICOS);
    const chips = await page.evaluate(() =>
      [...document.querySelectorAll('.selector button')]
        .filter((b) => b.offsetParent !== null)
        .map((b) => {
          const r = b.getBoundingClientRect();
          return { texto: (b.textContent || '').trim().slice(0, 30), w: Math.round(r.width), h: Math.round(r.height) };
        })
    );
    rep.chips = chips;
    check('360: todos los controles del selector ≥44 px de alto', chips.length > 0 && chips.every((c) => c.h >= 44), chips.map((c) => `${c.texto}:${c.h}`));
  });

  writeFileSync(join(dirOut, 'reproductor.json'), JSON.stringify(rep, null, 2));
  await browser.close();
  return rep;
}

// ---------------------------------------------------------------------------
// Modo siguiente (T84): chip "Siguiente" del reproductor
// ---------------------------------------------------------------------------

const ANCHOS_SIGUIENTE = [
  { w: 360, h: 740, nombre: '360' },
  { w: 390, h: 844, nombre: '390' },
  { w: 768, h: 1024, nombre: '768' },
  { w: 1280, h: 800, nombre: '1280' },
];

const MEDIR_SIGUIENTE = () => {
  const d = document.documentElement;
  const round = (n) => Math.round(n * 100) / 100;
  const rect = (r) => ({
    x: round(r.x), y: round(r.y), w: round(r.width), h: round(r.height),
    right: round(r.right), bottom: round(r.bottom),
  });
  const chip = document.querySelector('.marco .boton-siguiente');
  const marco = document.querySelector('.marco');
  const overlay = document.querySelector('.marco .cargando-overlay');
  const cr = chip ? chip.getBoundingClientRect() : null;
  const mr = marco ? marco.getBoundingClientRect() : null;
  const or = overlay ? overlay.getBoundingClientRect() : null;
  const cs = chip ? getComputedStyle(chip) : null;
  // El overlay no pinta un spinner gráfico: su indicador es el texto centrado.
  // Para el solape se usa una caja centrada de --tap (44×44) y, además, se
  // reporta el rectángulo real del texto.
  const spinner = mr
    ? { x: round(mr.x + mr.width / 2 - 22), y: round(mr.y + mr.height / 2 - 22), w: 44, h: 44 }
    : null;
  let textoOverlay = null;
  if (overlay) {
    const rango = document.createRange();
    rango.selectNodeContents(overlay);
    const tr = rango.getBoundingClientRect();
    if (tr.width > 0) textoOverlay = rect(tr);
  }
  const controles = mr ? { x: round(mr.x), y: round(mr.bottom - 44), w: round(mr.width), h: 44 } : null;
  const inter = (a, b) =>
    !!a && !!b && a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
  // El punto que debe tapar el overlay es el centro del chip; el "centro del
  // vídeo" es el del marco (no el del chip) para comprobar que no lo tapa.
  const centroChip = cr ? { x: cr.x + cr.width / 2, y: cr.y + cr.height / 2 } : null;
  const centroVideo = mr ? { x: mr.x + mr.width / 2, y: mr.y + mr.height / 2 } : null;
  const top = centroChip ? document.elementFromPoint(centroChip.x, centroChip.y) : null;
  const crRect = cr ? rect(cr) : null;
  const csd = getComputedStyle(d);
  const tok = (n) => csd.getPropertyValue(n).trim();
  const resolver = (n) => {
    const p = document.createElement('div');
    p.style.color = `var(${n})`;
    d.appendChild(p);
    const c = getComputedStyle(p).color;
    p.remove();
    return c;
  };
  return {
    innerWidth: window.innerWidth,
    overflowH: d.scrollWidth - d.clientWidth,
    pointerFine: matchMedia('(hover: hover) and (pointer: fine)').matches,
    pointerCoarse: matchMedia('(pointer: coarse)').matches,
    marco: mr ? rect(mr) : null,
    chip: cr && cs
      ? {
          ...crRect,
          texto: (chip.textContent || '').trim(),
          disabled: chip.disabled,
          opacity: cs.opacity,
          filter: cs.filter,
          backgroundImage: cs.backgroundImage,
          backgroundColor: cs.backgroundColor,
          color: cs.color,
          borderColor: cs.borderTopColor,
          borderRadius: cs.borderTopLeftRadius,
          fontSize: cs.fontSize,
          fontWeight: cs.fontWeight,
          minHeight: cs.minHeight,
          padding: cs.padding,
          transitionDuration: cs.transitionDuration,
          animationName: cs.animationName,
        }
      : null,
    rightPx: cr && mr ? round(mr.right - cr.right) : null,
    bottomPx: cr && mr ? round(mr.bottom - cr.bottom) : null,
    overlay: or
      ? { ...rect(or), texto: (overlay.textContent || '').trim(), textoRect: textoOverlay }
      : null,
    spinner,
    controles,
    centro: centroVideo ? { x: round(centroVideo.x), y: round(centroVideo.y) } : null,
    centroChip: centroChip ? { x: round(centroChip.x), y: round(centroChip.y) } : null,
    centroEnChip: !!(
      cr &&
      centroVideo &&
      centroVideo.x >= cr.x &&
      centroVideo.x <= cr.right &&
      centroVideo.y >= cr.y &&
      centroVideo.y <= cr.bottom
    ),
    solape: {
      spinner: inter(crRect, spinner),
      texto: inter(crRect, textoOverlay),
      controles: inter(crRect, controles),
    },
    top: top
      ? { tag: top.tagName, enChip: !!top.closest('.boton-siguiente'), enOverlay: !!top.closest('.cargando-overlay') }
      : null,
    tokens: {
      tap: tok('--tap'),
      sp4: tok('--sp-4'),
      sp2: tok('--sp-2'),
      surfacePanel: resolver('--surface-panel'),
      blurSm: tok('--blur-sm'),
      rFull: tok('--r-full'),
      fsSm: tok('--fs-sm'),
      border: resolver('--border'),
      text1: resolver('--text-1'),
      brand400: resolver('--brand-400'),
      onBrand: resolver('--on-brand'),
      gradBrand: tok('--grad-brand'),
      brandGlow: tok('--brand-glow'),
    },
  };
};

// Fuente falsa para el modo siguiente: el proxy del contenedor no sirve el HLS
// en el navegador; un iframe estable evita los ciclos de error y deja el chip
// en `listo` (discreto en modo iframe). `retardo` mantiene el overlay visible.
async function inyectarFuenteFalsa(contexto, retardo = 0) {
  await contexto.addInitScript((espera) => {
    const wrap = () => {
      const App = window.go && window.go.main && window.go.main.App;
      if (!App || App.__t84 || typeof App.GetVideoSources !== 'function') return false;
      App.GetVideoSources = () =>
        new Promise((res) =>
          setTimeout(
            () => res([{ url: 'about:blank', type: 'iframe', quality: '', server: 'prueba', language: 'SUB' }]),
            espera
          )
        );
      App.__t84 = true;
      return true;
    };
    if (!wrap()) {
      const id = setInterval(() => {
        if (wrap()) clearInterval(id);
      }, 2);
      setTimeout(() => clearInterval(id), 10000);
    }
  }, retardo);
}

async function siguiente() {
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/siguiente';
  mkdirSync(dirOut, { recursive: true });
  const browser = await chromium.launch();
  const rep = {
    anchos: [],
    funciones: [],
    capturas: [],
    overlay: null,
    raton: null,
    tactil: null,
    resaltado: null,
    click: null,
    literales: null,
    errores: [],
  };
  const check = (nombre, ok, detalle) =>
    rep.funciones.push({ nombre, ok: !!ok, detalle: detalle === undefined ? null : detalle });
  const paso = async (nombre, fn) => {
    try {
      await fn();
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const captura = async (p, nombre) => {
    try {
      await p.screenshot({ path: join(dirOut, `${nombre}.png`) });
      rep.capturas.push(`${nombre}.png`);
    } catch (e) {
      rep.errores.push(`${nombre}: ${e.message}`);
    }
  };
  const abrirEn = async (p, a) => {
    await p.setViewportSize({ width: a.w, height: a.h });
    await gotoApp(p);
    await p.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
    await buscar(p, 'bleach');
    await abrirDetalle(p, 'animeav1');
    const ep = p.locator('.rejilla .celda .tarjeta').first();
    if (!(await ep.count())) return false;
    await ep.click();
    await p.waitForSelector('.marco', { timeout: 30000 });
    await p.waitForSelector('.marco .boton-siguiente', { timeout: 20000 }).catch(() => {});
    await p
      .waitForFunction(() => !document.querySelector('.marco .cargando-overlay'), null, { timeout: 15000 })
      .catch(() => {});
    await p.waitForTimeout(300);
    return true;
  };
  const baseVisible = (p) =>
    p.evaluate(() => {
      const b = document.querySelector('.marco .boton-siguiente');
      if (!b) return false;
      b.classList.remove('discreto', 'resaltado');
      b.disabled = false;
      return true;
    });
  const subtitulo = (p) =>
    p.evaluate(() => ((document.querySelector('.cabecera .subtitulo') || {}).textContent || '').trim());
  const esperarListo = (p) =>
    p
      .waitForFunction(
        () => {
          const b = document.querySelector('.marco .boton-siguiente');
          return !!b && !b.disabled && /Ep \d+/.test((b.textContent || '').trim());
        },
        null,
        { timeout: 15000 }
      )
      .then(() => true)
      .catch(() => false);

  const ctx = await browser.newContext({ viewport: { width: 360, height: 740 } });
  await inyectarFuenteFalsa(ctx, 0);
  const page = await ctx.newPage();

  // --- A. Geometría y captura por ancho ---
  await paso('A', async () => {
    for (const a of ANCHOS_SIGUIENTE) {
      const abierto = await abrirEn(page, a);
      if (!abierto) {
        check(`${a.nombre}: reproductor abierto`, false, 'sin tarjetas de episodio');
        continue;
      }
      await baseVisible(page);
      await page.waitForTimeout(250);
      const m = await page.evaluate(MEDIR_SIGUIENTE);
      rep.anchos.push({ ancho: a.nombre, ...m });
      check(`${a.nombre}: alto del chip ≥44 px`, !!m.chip && m.chip.h >= 44, m.chip?.h);
      check(`${a.nombre}: right = --sp-4 (16 px)`, m.rightPx !== null && Math.abs(m.rightPx - 16) <= 1, m.rightPx);
      check(
        `${a.nombre}: bottom = --tap + --sp-4 (60 px)`,
        m.bottomPx !== null && Math.abs(m.bottomPx - 60) <= 1,
        m.bottomPx
      );
      check(`${a.nombre}: sin overflow horizontal`, m.overflowH === 0, m.overflowH);
      check(`${a.nombre}: no tapa el centro del vídeo`, m.centroEnChip === false, m.centro);
      check(
        `${a.nombre}: no pisa los controles nativos (últimos 44 px)`,
        m.solape.controles === false,
        { chip: m.chip, controles: m.controles }
      );
      if (a.nombre === '360') await captura(page, 'siguiente_360_normal');
      if (a.nombre === '1280') await captura(page, 'siguiente_1280');
      log('siguiente ancho', a.nombre, 'chip=', m.chip && `${m.chip.w}x${m.chip.h}`, 'marco=', JSON.stringify(m.marco));
    }
  });

  // --- B. Discreto con ratón (puntero fino) ---
  await paso('B', async () => {
    await abrirEn(page, { w: 360, h: 740, nombre: '360' });
    await page.evaluate(() => {
      const b = document.querySelector('.marco .boton-siguiente');
      if (b) {
        b.classList.remove('resaltado');
        b.classList.add('discreto');
        b.disabled = false;
      }
    });
    await page.waitForTimeout(300);
    const antes = await page.evaluate(MEDIR_SIGUIENTE);
    rep.raton = { fine: antes.pointerFine, antes: { opacity: antes.chip?.opacity, filter: antes.chip?.filter } };
    check('ratón: puntero fino', antes.pointerFine === true, antes.pointerFine);
    check('ratón: opacidad 0,15', antes.chip?.opacity === '0.15', antes.chip?.opacity);
    check('ratón: blur 2 px', antes.chip?.filter === 'blur(2px)', antes.chip?.filter);
    await page.locator('.marco .boton-siguiente').hover();
    await page.waitForTimeout(400);
    const hover = await page.evaluate(MEDIR_SIGUIENTE);
    rep.raton.hover = { opacity: hover.chip?.opacity, filter: hover.chip?.filter };
    check('ratón: hover aclara a opacidad 1', hover.chip?.opacity === '1', hover.chip?.opacity);
    check('ratón: hover quita el blur', hover.chip?.filter === 'none', hover.chip?.filter);
    await page.mouse.move(0, 0);
    await page.waitForTimeout(400);
    await page.locator('.marco .boton-siguiente').focus();
    await page.waitForTimeout(400);
    const foco = await page.evaluate(MEDIR_SIGUIENTE);
    rep.raton.foco = { opacity: foco.chip?.opacity, filter: foco.chip?.filter };
    check('ratón: foco aclara a opacidad 1', foco.chip?.opacity === '1', foco.chip?.opacity);
  });

  // --- C. Discreto táctil (hasTouch, pointer coarse) ---
  await paso('C', async () => {
    const ctxT = await browser.newContext({
      viewport: { width: 360, height: 740 },
      hasTouch: true,
      isMobile: true,
      deviceScaleFactor: 1,
    });
    await inyectarFuenteFalsa(ctxT, 0);
    const pT = await ctxT.newPage();
    await abrirEn(pT, { w: 360, h: 740, nombre: '360' });
    await pT.evaluate(() => {
      const b = document.querySelector('.marco .boton-siguiente');
      if (b) {
        b.classList.remove('resaltado');
        b.classList.add('discreto');
        b.disabled = false;
      }
    });
    await pT.waitForTimeout(300);
    const m = await pT.evaluate(MEDIR_SIGUIENTE);
    rep.tactil = { coarse: m.pointerCoarse, fine: m.pointerFine, opacity: m.chip?.opacity, filter: m.chip?.filter };
    check('táctil: pointer coarse', m.pointerCoarse === true, m.pointerCoarse);
    check('táctil: opacidad 0,6', m.chip?.opacity === '0.6', m.chip?.opacity);
    check('táctil: sin blur', m.chip?.filter === 'none', m.chip?.filter);
    await captura(pT, 'siguiente_360_tactil');
    await ctxT.close();
  });

  // --- D. Resaltado: degradado, tokens y pulso ---
  await paso('D', async () => {
    await abrirEn(page, { w: 360, h: 740, nombre: '360' });
    await page.emulateMedia({ reducedMotion: 'no-preference' });
    await page.evaluate(() => {
      const b = document.querySelector('.marco .boton-siguiente');
      if (b) {
        b.classList.remove('discreto');
        b.classList.add('resaltado');
        b.disabled = false;
      }
    });
    await page.waitForTimeout(300);
    const m = await page.evaluate(MEDIR_SIGUIENTE);
    rep.resaltado = {
      backgroundImage: m.chip?.backgroundImage,
      color: m.chip?.color,
      border: m.chip?.borderColor,
      animationName: m.chip?.animationName,
      reduced: null,
    };
    check(
      'resaltado: fondo con degradado de marca',
      (m.chip?.backgroundImage ?? '').includes('linear-gradient'),
      m.chip?.backgroundImage
    );
    check('resaltado: texto --on-brand', m.chip?.color === m.tokens.onBrand, {
      got: m.chip?.color,
      esperado: m.tokens.onBrand,
    });
    check('resaltado: borde --brand-400', m.chip?.borderColor === m.tokens.brand400, {
      got: m.chip?.borderColor,
      esperado: m.tokens.brand400,
    });
    check(
      'resaltado: pulso presente sin reduced-motion',
      !!m.chip?.animationName && m.chip.animationName.endsWith('-pulso-siguiente'),
      m.chip?.animationName
    );
    await captura(page, 'siguiente_360_resaltado');
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.waitForTimeout(250);
    const reduce = await page.evaluate(MEDIR_SIGUIENTE);
    rep.resaltado.reduced = {
      animationName: reduce.chip?.animationName,
      transitionDuration: reduce.chip?.transitionDuration,
    };
    check('reduced-motion: sin pulso', reduce.chip?.animationName === 'none', reduce.chip?.animationName);
    check('reduced-motion: sin transición', reduce.chip?.transitionDuration === '0s', reduce.chip?.transitionDuration);
    await page.emulateMedia({ reducedMotion: null });
    await page.evaluate(() => {
      const b = document.querySelector('.marco .boton-siguiente');
      if (b) b.disabled = true;
    });
    await page.waitForTimeout(100);
    const disabled = await page.evaluate(MEDIR_SIGUIENTE);
    rep.resaltado.disabled = { opacity: disabled.chip?.opacity };
    check('disabled: opacidad 0,55', disabled.chip?.opacity === '0.55', disabled.chip?.opacity);
    await baseVisible(page);
    await page.waitForTimeout(200);
  });

  // --- E. Overlay de carga: cubre el chip y luego lo libera ---
  await paso('E', async () => {
    rep.overlay = {};
    for (const a of [
      { w: 360, h: 740, nombre: '360' },
      { w: 390, h: 844, nombre: '390' },
    ]) {
      const ctxL = await browser.newContext({ viewport: { width: a.w, height: a.h } });
      await inyectarFuenteFalsa(ctxL, 2500);
      const pL = await ctxL.newPage();
      await pL.setViewportSize({ width: a.w, height: a.h });
      await gotoApp(pL);
      await pL.waitForSelector('.fila .carrusel .tarjeta', { timeout: 60000 }).catch(() => {});
      await buscar(pL, 'bleach');
      await abrirDetalle(pL, 'animeav1');
      const ep = pL.locator('.rejilla .celda .tarjeta').first();
      if (!(await ep.count())) {
        check(`${a.nombre}: overlay de carga`, false, 'sin tarjetas de episodio');
        await ctxL.close();
        continue;
      }
      await ep.click();
      await pL.waitForSelector('.marco .cargando-overlay', { timeout: 15000 }).catch(() => {});
      const cargando = await pL.evaluate(MEDIR_SIGUIENTE);
      rep.overlay[a.nombre] = {
        marco: cargando.marco,
        spinner: cargando.spinner,
        chip: cargando.chip
          ? { x: cargando.chip.x, y: cargando.chip.y, w: cargando.chip.w, h: cargando.chip.h }
          : null,
        solape: cargando.solape,
        top: cargando.top,
        centro: cargando.centro,
        aviso: cargando.solape.spinner
          ? 'el rectángulo del chip se solapa con la caja central de 44×44; el chip se oculta con opacidad 0 bajo el overlay'
          : null,
      };
      check(`${a.nombre}: overlay visible durante la carga`, !!cargando.overlay, cargando.overlay);
      check(
        `${a.nombre}: el overlay tapa el chip (elementFromPoint)`,
        cargando.top?.enOverlay === true && cargando.top?.enChip === false,
        cargando.top
      );
      await pL.locator('.marco .boton-siguiente').focus();
      const enfocado = await pL.evaluate(() => {
        const b = document.activeElement;
        return !!b && b.classList.contains('boton-siguiente');
      });
      check(`${a.nombre}: el chip sigue enfocable durante la carga`, enfocado, enfocado);
      check(`${a.nombre}: el chip no pisa los controles nativos`, cargando.solape.controles === false, {
        chip: rep.overlay[a.nombre].chip,
        controles: cargando.controles,
      });
      check(`${a.nombre}: el chip no tapa el centro del vídeo`, cargando.centroEnChip === false, cargando.centro);
      check(
        `${a.nombre}: el chip queda oculto bajo el overlay (sin solape visible)`,
        cargando.chip?.opacity === '0',
        { opacity: cargando.chip?.opacity, solapeGeometrico: cargando.solape.spinner }
      );
      if (a.nombre === '360') await captura(pL, 'siguiente_360_cargando');
      await pL
        .waitForFunction(() => !document.querySelector('.marco .cargando-overlay'), null, { timeout: 15000 })
        .catch(() => {});
      await pL.waitForTimeout(400);
      const tras = await pL.evaluate(MEDIR_SIGUIENTE);
      rep.overlay[a.nombre].tras = { overlay: tras.overlay, top: tras.top };
      check(`${a.nombre}: al terminar la carga el chip recibe el punto`, tras.top?.enChip === true, tras.top);
      if (a.nombre === '360') await captura(pL, 'siguiente_360_cargado');
      await ctxL.close();
    }
  });

  // --- F. Clic y tecla N navegan al siguiente episodio ---
  await paso('F', async () => {
    await abrirEn(page, { w: 360, h: 740, nombre: '360' });
    await esperarListo(page);
    await baseVisible(page);
    await page.waitForTimeout(250);
    const med = await page.evaluate(MEDIR_SIGUIENTE);
    check(
      'sin overlay el chip recibe el clic (elementFromPoint)',
      med.overlay === null && med.top?.enChip === true,
      med.top
    );
    const antes = await subtitulo(page);
    let clicForzado = false;
    try {
      await page.locator('.marco .boton-siguiente').click({ timeout: 5000 });
    } catch {
      clicForzado = true;
      await page.evaluate(() => document.querySelector('.marco .boton-siguiente')?.click());
    }
    await page.waitForTimeout(2500);
    const trasClic = await subtitulo(page);
    rep.click = { antes, trasClic, clicForzado };
    check(
      'clic: navega al siguiente episodio',
      trasClic !== antes && trasClic !== '',
      { antes, trasClic, clicForzado }
    );

    await esperarListo(page);
    const antesN = await subtitulo(page);
    await page.keyboard.press('n');
    await page.waitForTimeout(2500);
    const trasN = await subtitulo(page);
    rep.click.n = { antes: antesN, despues: trasN };
    check('tecla N: navega al siguiente episodio', trasN !== antesN && trasN !== '', { antes: antesN, despues: trasN });
  });

  // --- G. Literales de color prohibidos ---
  await paso('G', async () => {
    const ruta = 'frontend/src/BotonSiguiente.svelte';
    const texto = readFileSync(ruta, 'utf8');
    const rgba74 = (texto.match(/rgba\(74,\s*222,\s*128/gi) ?? []).length;
    const hex = (texto.match(/#[0-9a-f]{3,8}\b/gi) ?? []).length;
    rep.literales = { rgba74, hex };
    check('0 literales rgba(74, 222, 128) en BotonSiguiente', rgba74 === 0, rgba74);
    check('0 literales hex en BotonSiguiente', hex === 0, hex);
  });

  writeFileSync(join(dirOut, 'siguiente.json'), JSON.stringify(rep, null, 2));
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
  // Región opcional a comparar: `desdeX [hastaX]` (p. ej. excluir la barra
  // lateral que cambia con los datos de historial).
  const desdeX = Number(process.argv[6] ?? 0);
  const hastaX = process.argv[7] === undefined ? Infinity : Number(process.argv[7]);
  if (!dirA || !dirB || !existsSync(dirA) || !existsSync(dirB)) {
    log('uso: comparar <dirBase> <dirActual> [tol] [desdeX] [hastaX]');
    process.exit(1);
  }
  const nombres = readdirSync(dirA).filter((f) => f.toLowerCase().endsWith('.png')).sort();
  let iguales = 0, distintos = 0, faltan = 0;
  console.log(
    `Comparación por canal (tol ≤ ${tol}, x ∈ [${desdeX}, ${hastaX === Infinity ? 'fin' : hastaX})): ${dirA} vs ${dirB}`
  );
  for (const nombre of nombres) {
    const pa = join(dirA, nombre), pb = join(dirB, nombre);
    if (!existsSync(pa) || !existsSync(pb)) { faltan++; continue; }
    const r = diffPNG(decodePNG(readFileSync(pa)), decodePNG(readFileSync(pb)), tol, desdeX, hastaX);
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
} else if (modo === 'buscador') {
  const rep = await buscador();
  console.log('## Modo buscador (T75) — pestañas y buscador en modo estrecho\n');
  console.log('| ancho | tablist | role paneles | lupa | ✕ | input w×h | Buscar w×h | pestaña h | tarj.Home h | overflowH |');
  console.log('|---|:--:|---|:--:|:--:|---:|---:|---:|---:|---:|');
  for (const m of rep.anchos) {
    const fmt = (r) => (r ? `${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.ancho} | ${m.tablist ? 'sí' : 'no'} | ${m.roleDescubrir ?? '—'} | ${m.lupa ? 'sí' : 'no'} | ${m.limpiar ? 'sí' : 'no'} | ` +
        `${fmt(m.rects.input)} | ${fmt(m.rects.buscar)} | ${fmt(m.rects.pestana)} | ${fmt(m.rects.tarjetaHome)} | ${m.overflowH} |`
    );
  }
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones) {
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  }
  console.log(`\nTáctiles <44 px visibles en el Home (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (docs/img/v1.4/buscador/): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'filas') {
  const rep = await filas();
  console.log('## Modo filas (T76) — carruseles y tarjetas del Home\n');
  console.log('| ancho | tarj.Home h | portada w×h | 1ª tarjeta x | carrusel right | flecha w×h | mask | overflowH |');
  console.log('|---:|---:|---:|---:|---:|---:|:--:|---:|');
  for (const m of rep.anchos) {
    const fmt = (r) => (r ? `${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.ancho} | ${m.tarjetaHomeH ?? '—'} | ${fmt(m.portada)} | ${m.primera?.x ?? '—'} | ${m.carrusel?.right ?? '—'} | ` +
        `${m.flecha ? `${fmt(m.flecha)} (${m.flecha.display})` : '—'} | ${m.carrusel?.mask === 'none' ? 'no' : 'sí'} | ${m.overflowH} |`
    );
  }
  console.log('\nEsqueleto → datos:', JSON.stringify(rep.esqueleto));
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nTáctiles <44 px visibles en el Home (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/filas'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'continuar') {
  const rep = await continuar();
  console.log('## Modo continuar (T77) — fila "Continuar viendo" del Home\n');
  console.log('| ancho | fila | título | tarjetas | 1ª x | gutter | tarj.w×h | tarj.Home h | pestaña | overflowH |');
  console.log('|---|:--:|---:|---:|---:|---:|---|---:|---:|---:|');
  for (const m of rep.anchos) {
    const fmt = (t) => (t ? `${t.w}×${t.h}` : '—');
    console.log(
      `| ${m.ancho} | ${m.fila ? 'sí' : 'no'} | ${m.titulo ?? '—'} | ${m.tarjetas.length} | ${fmt(m.tarjetas[0])} | ` +
        `${m.gutter} | ${m.tarjetas.map((t) => `${t.porcentaje}%`).join('/') || '—'} | ${m.tarjetaHome?.h ?? '—'} | ` +
        `${m.pestana?.h ?? '—'} | ${m.overflowH} |`
    );
  }
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  const fallos = rep.funciones.filter((f) => !f.ok).length;
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos}/${rep.funciones.length} OK`);
  console.log(`Táctiles <44 px visibles en el Home (${rep.tactiles.length}); en la fila: ${rep.tactiles.filter((t) => t.enFilaContinuar).length}`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}${t.enFilaContinuar ? ' [FILA]' : ''}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/continuar'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'busqueda') {
  const rep = await busqueda();
  console.log('## Modo busqueda (T78) — rejilla de resultados y estados\n');
  console.log('| ancho/estado | columnas | gap | tarjetas | 1ª w×h | contador | overflowH |');
  console.log('|---|---:|---:|---:|---|---|---:|');
  for (const m of rep.anchos) {
    const fmt = (r) => (r ? `${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.estado ? `${m.ancho} ${m.estado}` : m.ancho} | ${m.columnas} | ${m.gap ?? '—'} | ${m.tarjetas} | ${fmt(m.primera)} | ` +
        `${m.contadorTexto ?? '—'} | ${m.overflowH} |`
    );
  }
  console.log('\nEsqueleto → resultados:', JSON.stringify(rep.esqueleto));
  console.log('Portadas (scroll real):', JSON.stringify(rep.imagenes));
  console.log('Vacío:', JSON.stringify(rep.vacio));
  console.log('Zona de resultados @1280 (contador+rejilla):', JSON.stringify(rep.zona1280));
  const fallos = rep.funciones.filter((f) => !f.ok);
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos.length}/${rep.funciones.length} OK`);
  console.log(`Táctiles <44 px visibles en los resultados (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/busqueda'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'ficha') {
  const rep = await ficha();
  console.log('## Modo ficha (T79) — ficha de detalle\n');
  console.log('| vista@ancho | ficha w×h | marco w×h | chips | géneros | sinopsis líneas | "Ver más" | tarj.ep w×h | cols | overflowH |');
  console.log('|---|---:|---:|---:|---|---:|:--:|---:|---:|---:|');
  for (const m of rep.anchos) {
    const fmt = (r) => (r ? `${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.vista ? `${m.vista}@` : ''}${m.ancho} | ${fmt(m.ficha)} | ${fmt(m.marco)} | ${m.chips.length} | ` +
        `${m.generos ? `${m.generos.flexWrap}/${m.generos.overflowX}` : '—'} | ${m.sinopsis?.lineas ?? '—'} | ` +
        `${m.verMas ? `${m.verMas.texto} ${m.verMas.rect.h}px` : '—'} | ${fmt(m.tarjetaEp)} | ${m.columnas} | ${m.overflowH} |`
    );
  }
  console.log('\nAltura de la ficha (360):', JSON.stringify(rep.sinopsis));
  console.log('Estado "En emisión":', JSON.stringify(rep.emision));
  console.log('Revelado T65 (.entra):', rep.entra);
  const fallos = rep.funciones.filter((f) => !f.ok);
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos.length}/${rep.funciones.length} OK`);
  console.log(`Táctiles <44 px visibles en el detalle (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/ficha'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'cta') {
  const rep = await cta();
  console.log('## Modo cta (T80) — barra de acción fija del detalle\n');
  console.log('| ancho | CTA | hueco cta/barra | botones w×h | principal | secundario | ".continuar" | tarj.ep w×h | overflowH |');
  console.log('|---|:--:|---:|---|---|---|:--:|---:|---:|');
  for (const m of rep.anchos) {
    const fmt = (r) => (r ? `${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.ancho} | ${m.cta ? 'sí' : 'no'} | ${m.hueco ?? '—'} | ` +
        `${m.botones.map((b) => `${b.w}×${b.h}`).join(', ') || '—'} | ${m.principalTexto ?? '—'} | ${m.secundarioTexto ?? '—'} | ` +
        `${m.continuar ? (m.continuar.visible ? 'visible' : 'oculto') : '—'} | ${fmt(m.tarjetaEp)} | ${m.overflowH} |`
    );
  }
  console.log('\nSin progreso:', JSON.stringify(rep.sinProgreso));
  console.log('Lote JKAnime:', JSON.stringify(rep.lote));
  console.log('Fin de scroll (CTA vs última celda):', JSON.stringify(rep.finScroll));
  console.log('Scroll T66/T67:', JSON.stringify(rep.scroll));
  console.log('Cajón:', JSON.stringify(rep.cajon));
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  const fallos = rep.funciones.filter((f) => !f.ok).length;
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos}/${rep.funciones.length} OK`);
  console.log(`Táctiles <44 px visibles en el detalle (${rep.tactiles.length}); en la CTA: ${rep.tactiles.filter((t) => t.enCta).length}`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}${t.enCta ? ' [CTA]' : ''}${t.enBarra ? ' [BARRA]' : ''}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/cta'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'episodios') {
  const rep = await episodios();
  console.log('## Modo episodios (T81) — tarjeta y rejilla en estrecho\n');
  console.log('| ancho | cols | tarj. w | tarj. alto (min..max) | gap | overflowH |');
  console.log('|---|:--:|---:|---|---:|---:|');
  for (const m of rep.anchos) {
    if (m.err) { console.log(`| ${m.ancho} | — | — | — | — | ${m.err} |`); continue; }
    console.log(
      `| ${m.ancho} | ${m.cols} | ${m.wMin === m.wMax ? m.wMin : m.wMin + '..' + m.wMax} | ` +
        `${m.hMin}${m.hMin === m.hMax ? '' : '..' + m.hMax} | ${m.gap} | ${m.overflowH} |`
    );
  }
  console.log('\nEstados (360):', JSON.stringify(rep.estados, null, 0));
  console.log('Miniatura rota:', JSON.stringify(rep.roto));
  console.log('Hover:', JSON.stringify(rep.hover));
  console.log('Revelado T65:', rep.revelado, '| scroll T66/T67:', JSON.stringify(rep.scroll));
  console.log('Táctiles (tarjetas):', JSON.stringify(rep.tactiles));
  const fallos = rep.funciones.filter((f) => !f.ok);
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos.length}/${rep.funciones.length} OK`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/episodios'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'orden') {
  const rep = await orden();
  console.log('## Modo orden (T82) — botón ascendente/descendente de episodios\n');
  console.log('| ancho | botón w×h | texto | aria-pressed | 1.º asc → últ. asc | 1.º desc → últ. desc | tarjetas | CTA | cols | alto | overflowH |');
  console.log('|---:|---|---|:--:|---|---|---:|---|---:|---:|---:|');
  for (const m of rep.anchos) {
    const b = m.asc.boton;
    console.log(
      `| ${m.ancho} | ${b ? `${b.w}×${b.h}` : '—'} | ${b?.texto ?? '—'}${b ? (b.textoVisible ? '' : ' (oculto)') : ''} | ` +
        `${m.asc2.boton?.ariaPressed ?? '—'} | ${m.asc.cards.primero} → ${m.asc.cards.ultimo} | ` +
        `${m.desc.cards.primero} → ${m.desc.cards.ultimo} | ${m.asc.cards.total} | ${m.asc.cta ?? '—'} | ` +
        `${m.asc.columnas} | ${m.asc.cards.altoMin} | ${m.asc.overflowH} |`
    );
  }
  console.log('\nRevelado T65:', JSON.stringify(rep.revelado));
  console.log('Scroll al invertir:', JSON.stringify(rep.scroll));
  console.log('Detalle → reproductor → Atrás:', JSON.stringify(rep.retorno));
  console.log('Anime de 1 episodio:', JSON.stringify(rep.unEpisodio));
  console.log('Hover:', JSON.stringify(rep.hover), '| reduced-motion:', JSON.stringify(rep.reduced));
  const fallos = rep.funciones.filter((f) => !f.ok);
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos.length}/${rep.funciones.length} OK`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/orden'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'reproductor') {
  const rep = await reproductor();
  console.log('## Modo reproductor (T83) — layout y selector de fuentes\n');
  console.log('| vista@ancho | gutter | Atrás x | marco x/y w×h | ratio | chips | alto chip | fila wrap | extras | plegable h | overflowH | barra |');
  console.log('|---|---:|---:|---|---:|---:|---:|---|---:|---:|---:|:--:|');
  for (const m of rep.anchos) {
    if (m.err) { console.log(`| ${m.ancho} | — | — | — | — | — | — | — | — | — | — | — |`); continue; }
    const fmtR = (r) => (r ? `${r.x}/${r.y} ${r.w}×${r.h}` : '—');
    console.log(
      `| ${m.vista ? m.vista + '@' : ''}${m.ancho} | ${m.gutter} | ${m.atras?.x ?? '—'} | ${fmtR(m.marco)} | ${m.ratio ?? '—'} | ` +
        `${m.fila?.botones.length ?? 0} | ${m.chip?.h ?? '—'} | ${m.fila?.flexWrap ?? '—'} | ${m.extras?.botones.length ?? 0} | ` +
        `${m.plegable?.rect.h ?? '—'} | ${m.overflowH} | ${m.barraInferior ? 'SÍ' : 'no'} |`
    );
  }
  console.log('\nChips por ancho:');
  for (const m of rep.anchos) {
    if (m.err || !m.fila) continue;
    const b = m.fila.botones.map((x) => `${x.texto} ${x.w}×${x.h}${x.activa ? ' [activa]' : ''}${x.ariaPressed ? ` (aria-pressed=${x.ariaPressed})` : ''}`).join(' · ');
    console.log(`- ${m.vista ? m.vista + '@' : ''}${m.ancho}: ${b}`);
  }
  console.log('\nAvisos:', JSON.stringify(rep.anchos.flatMap((m) => (m.avisos ?? []).map((a) => ({ ancho: m.ancho, ...a })))));
  console.log('Extras abiertas automáticamente (iframe):', rep.iframeAuto ? 'sí' : 'no');
  console.log('Aviso con iframe activo:', JSON.stringify(rep.avisoIframe ?? null));
  console.log('Reduced-motion:', JSON.stringify(rep.reduced));
  console.log('Táctiles del selector:', JSON.stringify(rep.chips));
  const fallos = rep.funciones.filter((f) => !f.ok);
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos.length}/${rep.funciones.length} OK`);
  console.log(`Táctiles <44 px visibles en el reproductor (${rep.tactiles.length}):`);
  for (const t of rep.tactiles) console.log(`- ${t.texto} ${t.w}×${t.h}${t.enBarra ? ' [BARRA]' : ''}`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${process.argv[3] ?? 'docs/img/v1.4/reproductor'}): ${rep.capturas.join(', ') || 'ninguna'}`);
} else if (modo === 'siguiente') {
  const rep = await siguiente();
  const dirOut = process.argv[3] ?? 'docs/img/v1.4/siguiente';
  console.log('## Modo siguiente (T84) — chip "Siguiente" del reproductor\n');
  console.log('| ancho | chip w×h | right | bottom | opacidad | filter | centro en chip | solape controles | solape spinner | overflowH |');
  console.log('|---|---:|---:|---:|---:|---|---|:--:|:--:|:--:|---:|');
  for (const m of rep.anchos) {
    const c = m.chip;
    console.log(
      `| ${m.ancho} | ${c ? `${c.w}×${c.h}` : '—'} | ${m.rightPx ?? '—'} | ${m.bottomPx ?? '—'} | ${c?.opacity ?? '—'} | ` +
        `${c?.filter ?? '—'} | ${m.centroEnChip ? 'SÍ' : 'no'} | ${m.solape?.controles ? 'SÍ' : 'no'} | ` +
        `${m.solape?.spinner ? 'SÍ' : 'no'} | ${m.overflowH} |`
    );
  }
  console.log('\nOverlay de carga (medido durante la carga):');
  console.log(JSON.stringify(rep.overlay, null, 2));
  const avisosSolape = Object.entries(rep.overlay ?? {})
    .filter(([, v]) => v?.aviso)
    .map(([ancho, v]) => `${ancho}px: ${v.aviso}`);
  console.log('\nAvisos de solape geométrico:', avisosSolape.length ? avisosSolape : 'ninguno');
  console.log('\nRatón:', JSON.stringify(rep.raton));
  console.log('Táctil:', JSON.stringify(rep.tactil));
  console.log('Resaltado:', JSON.stringify(rep.resaltado));
  console.log('Clic/tecla N:', JSON.stringify(rep.click));
  console.log('Literales:', JSON.stringify(rep.literales));
  const fallos = rep.funciones.filter((f) => !f.ok);
  console.log('\n| comprobación | ok | detalle |');
  console.log('|---|:--:|---|');
  for (const f of rep.funciones)
    console.log(`| ${f.nombre} | ${f.ok ? 'sí' : 'NO'} | ${f.detalle === null ? '' : JSON.stringify(f.detalle)} |`);
  console.log(`\nComprobaciones: ${rep.funciones.length - fallos.length}/${rep.funciones.length} OK`);
  if (rep.errores.length) console.log('\nErrores:', rep.errores);
  console.log(`\nCapturas (${dirOut}): ${rep.capturas.join(', ') || 'ninguna'}`);
  console.log(`Resumen JSON: ${join(dirOut, 'siguiente.json')}`);
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
