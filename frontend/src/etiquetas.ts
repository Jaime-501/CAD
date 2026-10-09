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
 * proveedor lo declara.
 */
export function etiquetaFuente(fuente: VideoSourceView): string {
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
