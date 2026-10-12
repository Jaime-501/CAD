/**
 * Retomar con un clic sobre una tarjeta del historial (T50).
 *
 * Resuelve la acción con `resumeAnime` (tabla de acciones de `v1.2_PLAN.md`) y
 * navega por la MISMA vía de hoy (anotada en el Paso 0 de `PROGRESO.md`):
 *  - `resume`/`next` → carga el detalle en segundo plano con `openAnime` (la vía
 *    de `VistaDetalle`) y encadena detalle → reproductor, para que "← Atrás"
 *    desapile el detalle con su lista de episodios ya cargada;
 *  - `completed`/`up_to_date` → abre el detalle con `navegarA('detalle', sel)`
 *    (la vía de `VistaBusqueda.abrir`).
 * El `startAt` no cabe en `seleccion` (`views.ts` queda fuera del alcance de
 * T50): se publica en el store `startAt` de `stores.ts`, que
 * `VistaReproductor` consume una sola vez al montar.
 */
import { get } from 'svelte/store';
import { RESUME_ACTION, resumeAnime, type models } from './api';
import { proveedorDe, tituloVisible } from './historial';
import { currentAnimeID, currentDetail, currentProvider, openAnime, startAt } from './stores';
import { navegarA, vista, type SeleccionAnime } from './views';

export type AccionRetomar = 'resume' | 'next' | 'completed' | 'up_to_date';

export type RetomarResultado =
  | { ok: true; accion: AccionRetomar }
  | { ok: false; error: string };

export async function retomar(item: models.WatchProgress): Promise<RetomarResultado> {
  const proveedor = proveedorDe(item);
  const animeID = item.anime_id;

  let info: models.ResumeInfo;
  try {
    info = await resumeAnime(proveedor, animeID);
  } catch (e) {
    return { ok: false, error: legible(e) };
  }

  // Anime mínimo de la tarjeta (paso 3): de ahí salen los datos de la selección.
  const anime: models.Anime = {
    id: animeID,
    title: tituloVisible(item),
    cover_url: item.cover_url,
    provider: proveedor,
  };
  const sel: SeleccionAnime = {
    provider: anime.provider,
    animeID: anime.id,
    title: anime.title,
  };
  const accion = info.action;

  if (accion === RESUME_ACTION.resume || accion === RESUME_ACTION.next) {
    if (info.episode_id === '') {
      return { ok: false, error: 'retomar no devolvió ningún episodio' };
    }
    cargarDetalleEnSegundoPlano(proveedor, animeID);
    // Las dos navegaciones van juntas: sin render intermedio del detalle, la
    // pila queda [..., 'detalle'] y "← Atrás" desde el reproductor desapila a él.
    navegarA('detalle', sel);
    startAt.set(info.start_at);
    navegarA('reproductor', {
      ...sel,
      episodeID: info.episode_id,
      episodeNumber: info.episode_number,
    });
    return { ok: true, accion };
  }

  if (accion === RESUME_ACTION.completed || accion === RESUME_ACTION.up_to_date) {
    // Si ya se está en el detalle no habrá remontaje (con lo que `VistaDetalle`
    // no volvería a llamar a `openAnime`): se carga aquí, en segundo plano.
    const yaEnDetalle = get(vista) === 'detalle';
    navegarA('detalle', sel);
    if (yaEnDetalle) cargarDetalleEnSegundoPlano(proveedor, animeID);
    return { ok: true, accion };
  }

  return { ok: false, error: `acción de retomar desconocida: ${accion}` };
}

/**
 * Carga detalle + episodios por la vía existente (`openAnime`), sin esperar:
 * si tarda, `VistaDetalle.onMount` la repetirá al desmontar el reproductor.
 * Si ESA pareja ya está cargada no se recarga (vaciaría la lista de episodios).
 */
function cargarDetalleEnSegundoPlano(proveedor: string, animeID: string): void {
  if (
    get(currentAnimeID) === animeID &&
    get(currentProvider) === proveedor &&
    get(currentDetail)
  ) {
    return;
  }
  void openAnime(proveedor, animeID).catch((e) => {
    console.warn('[retomar] detalle no cargado en segundo plano:', legible(e));
  });
}

function legible(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
