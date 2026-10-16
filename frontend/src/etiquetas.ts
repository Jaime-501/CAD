/**
 * Etiquetas visuales y utilidades de fuentes (T34).
 *
 * Todo aquí es puro (sin stores ni Wails) para que sea fácil de probar y para
 * que la lógica de SUB/DUB se pueda reutilizar cuando haya un filtro real.
 */
import type { VideoSourceView } from './api';

/**
 * Filtro de idioma para `fuentesVisibles`: `null` = sin filtro (lo que hay hoy;
 * el selector muestra SUB y DUB por igual). Solo cambia este valor para
 * añadir el filtro sin reescribir nada.
 */
export type FiltroIdioma = 'SUB' | 'DUB' | null;

/** Tipos de detalle que traen los dos proveedores → texto amable para la ficha. */
const TIPOS: Record<string, string> = {
  'tv-anime': 'TV',
  series: 'Serie',
  movie: 'Película',
  pelicula: 'Película',
  ova: 'OVA',
  ona: 'ONA',
  special: 'Especial',
  especial: 'Especial',
};

/**
 * Etiqueta del tipo de detalle (`AnimeDetail.type`): traduce los valores
 * crudos de los proveedores (`tv-anime`, `series`, `movie`…) y deja el resto
 * tal cual. Vacío/`unknown` → sin etiqueta.
 */
export function etiquetaTipo(tipo: string): string {
  const t = tipo.trim();
  if (t === '' || t === 'unknown') return '';
  return TIPOS[t.toLowerCase()] ?? t;
}

/**
 * Idioma estructurado de la fuente (`VideoSource.language`): `SUB`/`DUB` tal
 * como lo declaró el backend. Sin `Language` (JKAnime) → cadena vacía, y en
 * ese caso NO se pinta etiqueta de idioma.
 */
export function etiquetaIdioma(fuente: VideoSourceView): string {
  return (fuente.language ?? '').trim();
}

/**
 * Etiqueta de una fuente en el selector: `Servidor · Idioma · Calidad`, con
 * las partes que haya (siempre texto, nunca solo color). Sin servidor se usa
 * el tipo de recurso (HLS/MP4/Iframe), y el idioma solo aparece si el
 * proveedor lo declara. `sufijo` (T63) es una marca opcional que se añade al
 * final para desambiguar dos fuentes del mismo servidor (`· directo` en el
 * medio resuelto, `· iframe` en el embebido).
 */
export function etiquetaFuente(fuente: VideoSourceView, sufijo = ''): string {
  const tipo =
    fuente.type === 'hls'
      ? 'HLS'
      : fuente.type === 'mp4'
        ? 'MP4'
        : fuente.type === 'iframe'
          ? 'Iframe'
          : 'Fuente';
  const partes: string[] = [];
  const servidor = fuente.server.trim();
  partes.push(servidor !== '' ? servidor : tipo);
  const idioma = etiquetaIdioma(fuente);
  if (idioma !== '') partes.push(idioma);
  const calidad = fuente.quality.trim();
  if (calidad !== '') partes.push(calidad);
  if (sufijo !== '') partes.push(sufijo);
  return partes.join(' · ');
}

/**
 * Fuentes a mostrar en el selector. Sin filtro devuelve TODAS en el orden del
 * backend (la primera es la preferida). Con filtro, solo las de ese idioma: una
 * fuente sin `Language` no coincide con ningún filtro.
 */
export function fuentesVisibles(
  fuentes: VideoSourceView[],
  filtro: FiltroIdioma
): VideoSourceView[] {
  if (filtro === null) return fuentes;
  return fuentes.filter((f) => etiquetaIdioma(f) === filtro);
}

/**
 * Clasificación por TIPO (T63), nunca por nombre: una fuente es **principal**
 * cuando se reproduce en el `<video>` propio —`hls` (jkplayer, el HLS de
 * AnimeAV1) o `mp4` (MediaFire y el MP4Upload resuelto)—; es **extra** cuando
 * es un `iframe` embebido de un tercero (puede traer publicidad). El nombre
 * del servidor varía entre proveedores, por eso no decide nada aquí.
 */
export function esPrincipal(fuente: VideoSourceView): boolean {
  return fuente.type === 'hls' || fuente.type === 'mp4';
}

/** Extra = `iframe` (T63): embebido de terceros, fuera del control de progreso. */
export function esExtra(fuente: VideoSourceView): boolean {
  return fuente.type === 'iframe';
}

/** Fila del selector: la fuente, su índice en la lista COMPLETA y su etiqueta. */
export interface FilaFuente {
  fuente: VideoSourceView;
  indice: number;
  etiqueta: string;
}

/** Fuentes separadas para el selector (T63): principales primero, extras aparte. */
export interface FuentesAgrupadas {
  principales: FilaFuente[];
  extras: FilaFuente[];
}

/**
 * Separa las fuentes en principales y extras (T63) conservando el orden del
 * backend dentro de cada grupo, y etiqueta cada fila. Cuando el mismo nombre
 * de servidor —comparado sin mayúsculas— aparece en los dos grupos (p. ej. el
 * `Mp4upload` iframe y el `MP4Upload` directo de T61), el principal se marca
 * con `· directo` y el extra con `· iframe`; si no hay colisión la etiqueta
 * queda como siempre. Un tipo desconocido nunca se oculta: viaja con los
 * extras (hoy los proveedores solo emiten hls/mp4/iframe).
 */
export function agruparFuentes(
  fuentes: VideoSourceView[],
  filtro: FiltroIdioma = null
): FuentesAgrupadas {
  const visibles = fuentesVisibles(fuentes, filtro);
  const principales = visibles.filter(esPrincipal);
  const extras = visibles.filter((f) => !esPrincipal(f));
  const nombres = (lista: VideoSourceView[]): Set<string> =>
    new Set(
      lista.map((f) => f.server.trim().toLowerCase()).filter((nombre) => nombre !== '')
    );
  const enPrincipales = nombres(principales);
  const enExtras = nombres(extras);
  const hayColision = (f: VideoSourceView): boolean => {
    const nombre = f.server.trim().toLowerCase();
    return nombre !== '' && enPrincipales.has(nombre) && enExtras.has(nombre);
  };
  const fila = (f: VideoSourceView, sufijo: string): FilaFuente => ({
    fuente: f,
    indice: fuentes.indexOf(f),
    etiqueta: etiquetaFuente(f, hayColision(f) ? sufijo : ''),
  });
  return {
    principales: principales.map((f) => fila(f, 'directo')),
    extras: extras.map((f) => fila(f, 'iframe')),
  };
}

/**
 * Orden de reproducción/fallback (T63): la primera es la del backend (la
 * mejor, índice 0) y después van el resto de principales y, solo al final,
 * los extras (iframes). Devuelve índices sobre la lista recibida, así que el
 * salto ante un fallo recorre primero los principales restantes.
 */
export function ordenReproduccion(fuentes: VideoSourceView[]): number[] {
  const principales: number[] = [];
  const extras: number[] = [];
  fuentes.forEach((f, i) => {
    if (esPrincipal(f)) principales.push(i);
    else extras.push(i);
  });
  const orden = [...principales, ...extras];
  const posCero = orden.indexOf(0);
  if (posCero > 0) {
    orden.splice(posCero, 1);
    orden.unshift(0);
  }
  return orden;
}
