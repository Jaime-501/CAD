<script lang="ts">
  /**
   * Home (T53, pulido en T53b, con pestañas en T54): "Descubrir" es la página
   * "Novedades" con una fila por sección de cada proveedor. Cada proveedor se
   * pinta según SU estado (`home.ts`): skeletons mientras carga, bloque con
   * "Reintentar" si falla (el otro sigue visible) y sus filas cuando responde.
   *
   * La carga es una vez por sesión: al montar solo se pide lo que aún no tiene
   * estado, así que volver del detalle o vaciar la búsqueda no repite la
   * petición. El clic de cada tarjeta lo recibe por `onAbrir` desde
   * `VistaBusqueda` (la misma función de "abrir anime" de los resultados).
   *
   * T53b: el contenido va en un único `<section class="inicio">` sin padding
   * propio, así buscador, las pestañas y las filas comparten borde (el padding
   * vive solo en el `main.contenedor` de `VistaBusqueda`).
   *
   * T54: dos pestañas ARIA ("Descubrir" y "★ Favoritos (N)") con *roving
   * tabindex*. El panel inactivo NO usa `hidden` (`display: none`): Chrome pone
   * a cero el `scrollLeft` de las filas y al volver se perdería el desplazamiento;
   * con `position: absolute` + `visibility: hidden` sale del flujo y del árbol
   * de accesibilidad conservando la posición de scroll.
   */
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import FilaAnimes from './FilaAnimes.svelte';
  import GridFavoritos from './GridFavoritos.svelte';
  import { favoritos, cargarFavoritos } from './favoritos';
  import { cargarProveedor, cargarTodo, home, PROVEEDORES_HOME } from './home';
  import { isNarrow, pestanaHome } from './ui';

  export let onAbrir: (a: models.Anime) => void;

  type Pestana = 'descubrir' | 'favoritos';

  /**
   * Pestaña visible: la comparte el store `pestanaHome` (T73), así que ya NO se
   * reinicia a "Descubrir" al volver de una búsqueda: la selección persiste en
   * la sesión (desviación anotada en PROGRESO.md).
   */
  $: pestana = $pestanaHome;
  let tabDescubrir: HTMLButtonElement;
  let tabFavoritos: HTMLButtonElement;

  onMount(() => {
    const estadoActual = get(home);
    if (PROVEEDORES_HOME.some((p) => !(p.id in estadoActual))) cargarTodo();
  });

  /** "Actualizar" deshabilitado mientras algún proveedor tenga la petición en curso. */
  $: cargando = PROVEEDORES_HOME.some((p) => ($home[p.id]?.estado ?? 'loading') === 'loading');

  /** Ambos proveedores respondieron y ninguno trajo secciones. */
  $: sinContenido = PROVEEDORES_HOME.every((p) => {
    const estado = $home[p.id];
    return estado !== undefined && estado.estado === 'ok' && estado.secciones.length === 0;
  });

  function seleccionar(destino: Pestana): void {
    pestanaHome.set(destino);
    if (destino === 'favoritos') void cargarFavoritos();
  }

  /** Flechas ← → y Inicio/Fin entre pestañas (patrón ARIA de *tabs*). */
  function alTeclar(e: KeyboardEvent, actual: Pestana): void {
    const orden: Pestana[] = ['descubrir', 'favoritos'];
    const i = orden.indexOf(actual);
    let j = -1;
    if (e.key === 'ArrowRight') j = (i + 1) % orden.length;
    else if (e.key === 'ArrowLeft') j = (i - 1 + orden.length) % orden.length;
    else if (e.key === 'Home') j = 0;
    else if (e.key === 'End') j = orden.length - 1;
    else return;
    e.preventDefault();
    const ref = orden[j] === 'descubrir' ? tabDescubrir : tabFavoritos;
    ref.focus();
    seleccionar(orden[j]);
  }
</script>

<section class="inicio">
  {#if !$isNarrow}
    <div class="pestanas" role="tablist" aria-label="Secciones del Home">
      <button
        type="button"
        class="pestana"
        role="tab"
        id="pestana-descubrir"
        aria-controls="panel-descubrir"
        aria-selected={pestana === 'descubrir' ? 'true' : 'false'}
        tabindex={pestana === 'descubrir' ? 0 : -1}
        bind:this={tabDescubrir}
        on:click={() => seleccionar('descubrir')}
        on:keydown={(e) => alTeclar(e, 'descubrir')}
      >
        Descubrir
      </button>
      <button
        type="button"
        class="pestana"
        role="tab"
        id="pestana-favoritos"
        aria-controls="panel-favoritos"
        aria-selected={pestana === 'favoritos' ? 'true' : 'false'}
        tabindex={pestana === 'favoritos' ? 0 : -1}
        bind:this={tabFavoritos}
        on:click={() => seleccionar('favoritos')}
        on:keydown={(e) => alTeclar(e, 'favoritos')}
      >
        <span aria-hidden="true">★</span> Favoritos ({$favoritos.length})
      </button>
    </div>
  {/if}

  <div
    id="panel-descubrir"
    role={$isNarrow ? undefined : 'tabpanel'}
    aria-labelledby={$isNarrow ? undefined : 'pestana-descubrir'}
    class="panel"
    class:inactivo={pestana !== 'descubrir'}
  >
    <div class="cabecera">
      <h2>Novedades</h2>
      <button
        type="button"
        class="actualizar"
        on:click={() => cargarTodo(true)}
        disabled={cargando}
      >
        Actualizar
      </button>
    </div>

    {#each PROVEEDORES_HOME as proveedor (proveedor.id)}
      {@const estado = $home[proveedor.id]}
      {#if estado === undefined || estado.estado === 'loading'}
        <section class="fila-fantasma" aria-label="{proveedor.nombre} · cargando">
          <div class="cabecera-fantasma">
            <span class="proveedor-fantasma">{proveedor.nombre}</span>
          </div>
          <div class="carrusel-fantasma">
            {#each [0, 1, 2, 3, 4, 5, 6, 7] as indice (indice)}
              <div class="esqueleto" aria-hidden="true"></div>
            {/each}
          </div>
        </section>
      {:else if estado.estado === 'error'}
        <div class="error-proveedor">
          <p class="error-texto">No se pudo cargar {proveedor.nombre}.</p>
          <p class="error-detalle">{estado.error}</p>
          <button
            type="button"
            class="reintentar"
            on:click={() => void cargarProveedor(proveedor.id, true)}
          >
            Reintentar
          </button>
        </div>
      {:else}
        {#each estado.secciones as seccion (seccion.id)}
          <FilaAnimes proveedor={proveedor.nombre} titulo={seccion.title} items={seccion.items} {onAbrir} />
        {/each}
      {/if}
    {/each}

    {#if sinContenido}
      <p class="vacio">Sin novedades por ahora.</p>
    {/if}
  </div>

  <div
    id="panel-favoritos"
    role={$isNarrow ? undefined : 'tabpanel'}
    aria-labelledby={$isNarrow ? undefined : 'pestana-favoritos'}
    class="panel"
    class:inactivo={pestana !== 'favoritos'}
  >
    {#if $isNarrow}
      <h2>Favoritos ({$favoritos.length})</h2>
    {/if}
    <GridFavoritos {onAbrir} />
  </div>
</section>

<style>
  /* Sin padding ni max-width propios (T53b): el ancho lo fija el
     `main.contenedor` de `VistaBusqueda`, con el que comparten borde. */
  .inicio {
    position: relative;
    text-align: left;
  }
  /* Panel fuera de flujo (ver cabecera): conserva el scroll de las filas. */
  .panel.inactivo {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    visibility: hidden;
  }
  .pestanas {
    display: flex;
    gap: 4px;
    border-bottom: 1px solid var(--border);
    margin-bottom: 1.5rem;
  }
  .pestana {
    padding: 8px 14px;
    /* Cubre el borde de la fila con el indicador de 2 px. */
    margin-bottom: -1px;
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--text-3);
    font-family: inherit;
    font-size: 15px;
    font-weight: bold;
    cursor: pointer;
  }
  .pestana:hover {
    color: var(--text-1);
  }
  .pestana[aria-selected='true'] {
    color: var(--text-1);
    border-bottom-color: var(--brand-400);
  }
  .pestana:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: -2px;
  }
  .cabecera {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 1.5rem;
  }
  h2 {
    margin: 0;
    text-align: left;
  }
  .actualizar {
    padding: 8px 16px;
    border-radius: var(--r-md);
    border: none;
    background: var(--brand-500);
    color: var(--on-brand);
    font-weight: bold;
    font-family: inherit;
    cursor: pointer;
  }
  .actualizar:disabled {
    background: var(--bg-2);
    color: var(--text-3);
    cursor: not-allowed;
  }
  .actualizar:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  .error-proveedor {
    margin: 0 0 28px;
    padding: 12px;
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
  }
  .error-texto {
    margin: 0 0 4px;
    color: var(--text-1);
  }
  .error-detalle {
    margin: 0 0 10px;
    font-size: 13px;
    color: var(--text-3);
    overflow-wrap: anywhere;
  }
  .reintentar {
    box-sizing: border-box;
    padding: 6px 12px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    color: var(--brand-400);
    font-family: inherit;
    font-weight: bold;
    cursor: pointer;
  }
  .reintentar:focus-visible {
    outline: 2px solid var(--brand-400);
    outline-offset: 2px;
  }
  .vacio {
    color: var(--text-3);
    text-align: center;
  }
  /* Fila de carga (T53b): mismas medidas que la fila real (8 tarjetas
     140×264: portada 210 + bloque de título de 2 líneas). */
  .fila-fantasma {
    margin: 0 0 28px;
  }
  .cabecera-fantasma {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin: 0 0 10px;
    min-height: 22px;
  }
  .proveedor-fantasma {
    font-size: 12px;
    color: var(--text-3);
  }
  .carrusel-fantasma {
    display: flex;
    gap: 12px;
    overflow: hidden;
  }
  .esqueleto {
    flex: 0 0 140px;
    width: 140px;
    height: 264px;
    border-radius: var(--r-md);
    background: linear-gradient(
      100deg,
      var(--border-strong) 45%,
      var(--border) 50%,
      var(--border-strong) 55%
    );
    background-size: 200% 100%;
  }
  @media (prefers-reduced-motion: no-preference) {
    .esqueleto {
      animation: brillar 1.4s linear infinite;
    }
  }
  @keyframes brillar {
    from {
      background-position: 130% 0;
    }
    to {
      background-position: -30% 0;
    }
  }
  /* T75: en modo estrecho los botones del Home alcanzan el objetivo táctil. */
  @media (max-width: 899px) {
    .actualizar,
    .reintentar {
      min-height: var(--tap);
    }
  }
</style>
