<script lang="ts">
  /**
   * Fila "Continuar viendo" del Home móvil (T77). Solo aparece en modo estrecho
   * (`$isNarrow`, < 900 px) y con historial; en escritorio manda la barra
   * lateral. Reutiliza la lógica existente (`historial.ts` + `retomar.ts`) sin
   * lógica nueva: los 10 primeros registros, ya por recencia, sin agrupar.
   *
   * El clic replica el mapeo de `BarraLateral.alRetomar`: una resolución a la
   * vez, mensaje de 4 s ("No se pudo retomar" / "Completado" / "Al día"). No
   * hay borrado aquí (eso vive en el cajón de Historial).
   */
  import { onDestroy } from 'svelte';
  import type { models } from './api';
  import { historial, historialEstado, proveedorDe } from './historial';
  import { retomar, type RetomarResultado } from './retomar';
  import { isNarrow } from './ui';
  import TarjetaContinuar from './TarjetaContinuar.svelte';

  /** Máximo de tarjetas del carrusel (T77). */
  const MAX = 10;
  /** Duración del mensaje de tarjeta antes de borrarlo (igual que la barra). */
  const MENSAJE_MS = 4000;

  /** Clave `proveedor:anime_id` que se está resolviendo; null = ninguna. */
  let resolviendoId: string | null = null;
  let mensajes: Record<string, string> = {};
  let temporizadores: Record<string, ReturnType<typeof setTimeout>> = {};

  onDestroy(() => {
    for (const temporizador of Object.values(temporizadores)) clearTimeout(temporizador);
    temporizadores = {};
  });

  function claveDe(item: models.WatchProgress): string {
    return `${proveedorDe(item)}:${item.anime_id}`;
  }

  /** Clic en una tarjeta: si ya hay una resolución en curso, se ignora. */
  async function alRetomar(item: models.WatchProgress): Promise<void> {
    if (resolviendoId !== null) return;
    const clave = claveDe(item);
    resolviendoId = clave;
    borrarMensaje(clave);

    let res: RetomarResultado;
    try {
      res = await retomar(item);
    } catch (e) {
      res = { ok: false, error: e instanceof Error ? e.message : String(e) };
    }
    resolviendoId = null;

    if (!res.ok) {
      console.warn('[FilaContinuar] no se pudo retomar:', res.error);
      ponerMensaje(clave, 'No se pudo retomar');
      return;
    }
    if (res.accion === 'completed') ponerMensaje(clave, 'Completado');
    else if (res.accion === 'up_to_date') ponerMensaje(clave, 'Al día');
  }

  function ponerMensaje(clave: string, texto: string): void {
    borrarMensaje(clave);
    mensajes = { ...mensajes, [clave]: texto };
    temporizadores[clave] = setTimeout(() => borrarMensaje(clave), MENSAJE_MS);
  }

  function borrarMensaje(clave: string): void {
    clearTimeout(temporizadores[clave]);
    delete temporizadores[clave];
    if (!(clave in mensajes)) return;
    mensajes = Object.fromEntries(Object.entries(mensajes).filter(([k]) => k !== clave));
  }
</script>

{#if $isNarrow && $historial.length > 0}
  <section
    class="fila-continuar"
    aria-label="Continuar viendo"
    aria-busy={$historialEstado === 'loading'}
  >
    <h3 class="titulo-fila">Continuar viendo</h3>
    <div class="carrusel">
      {#each $historial.slice(0, MAX) as item (claveDe(item))}
        <TarjetaContinuar
          {item}
          onOpen={alRetomar}
          resolviendo={resolviendoId === claveDe(item)}
          mensaje={mensajes[claveDe(item)] ?? ''}
        />
      {/each}
    </div>
  </section>
{/if}

<style>
  .fila-continuar {
    margin: 0 0 28px;
    text-align: left;
  }
  /* Mismo estilo que `.titulo-fila` de FilaAnimes (T76). */
  .titulo-fila {
    margin: 0 0 10px;
    font-size: 16px;
    font-weight: bold;
    color: var(--text-1);
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  /* Carrusel con el mismo sangrado que T76: sin flechas ni degradados, el
     desplazamiento es por gesto/rueda. */
  .carrusel {
    display: flex;
    gap: 12px;
    overflow-x: auto;
    scroll-snap-type: x proximity;
    margin: -6px calc(-1 * var(--gutter)) -14px;
    padding: 6px var(--gutter) 14px;
    scroll-padding-left: var(--gutter);
    scroll-padding-right: var(--gutter);
    overscroll-behavior-x: contain;
    scrollbar-width: none;
  }
  .carrusel::-webkit-scrollbar {
    display: none;
  }
</style>
