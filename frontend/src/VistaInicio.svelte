<script lang="ts">
  /**
   * Home "Descubrir" (T53): página "Novedades" con una fila por sección de
   * cada proveedor. Cada proveedor se pinta según SU estado (`home.ts`):
   * skeletons mientras carga, bloque con "Reintentar" si falla (el otro sigue
   * visible) y sus filas cuando responde.
   *
   * La carga es una vez por sesión: al montar solo se pide lo que aún no tiene
   * estado, así que volver del detalle o vaciar la búsqueda no repete la
   * petición. El clic de cada tarjeta lo recibe por `onAbrir` desde
   * `VistaBusqueda` (la misma función de "abrir anime" de los resultados).
   */
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import FilaAnimes from './FilaAnimes.svelte';
  import { cargarProveedor, cargarTodo, home, PROVEEDORES_HOME } from './home';

  export let onAbrir: (a: models.Anime) => void;

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
</script>

<main class="inicio">
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
        <span class="titulo-fantasma">{proveedor.nombre}</span>
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
        <FilaAnimes
          titulo="{proveedor.nombre} · {seccion.title}"
          items={seccion.items}
          {onAbrir}
        />
      {/each}
    {/if}
  {/each}

  {#if sinContenido}
    <p class="vacio">Sin novedades por ahora.</p>
  {/if}
</main>

<style>
  .inicio {
    padding: 2rem;
    text-align: left;
    max-width: 1100px;
    margin: 0 auto;
  }
  .cabecera {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    margin-bottom: 1.5rem;
  }
  h2 {
    margin: 0;
    text-align: center;
  }
  .actualizar {
    padding: 8px 16px;
    border-radius: var(--cad-radio);
    border: none;
    background: var(--cad-acento);
    color: #111;
    font-weight: bold;
    font-family: inherit;
    cursor: pointer;
  }
  .actualizar:disabled {
    background: #555;
    color: #999;
    cursor: not-allowed;
  }
  .actualizar:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .error-proveedor {
    margin: 0 0 18px;
    padding: 12px;
    background: var(--cad-superficie);
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
  }
  .error-texto {
    margin: 0 0 4px;
    color: var(--cad-texto);
  }
  .error-detalle {
    margin: 0 0 10px;
    font-size: 13px;
    color: var(--cad-texto-suave);
    overflow-wrap: anywhere;
  }
  .reintentar {
    box-sizing: border-box;
    padding: 6px 12px;
    background: transparent;
    border: 1px solid var(--cad-borde);
    border-radius: var(--cad-radio);
    color: var(--cad-acento);
    font-family: inherit;
    font-weight: bold;
    cursor: pointer;
  }
  .reintentar:focus-visible {
    outline: 2px solid var(--cad-acento);
    outline-offset: 2px;
  }
  .vacio {
    color: var(--cad-texto-suave);
    text-align: center;
  }
  /* Fila de carga (T53): mismas medidas que la fila real (8 tarjetas 140×210). */
  .fila-fantasma {
    margin: 0 0 18px;
  }
  .titulo-fantasma {
    display: block;
    margin: 0 0 8px;
    font-size: 16px;
    font-weight: bold;
  }
  .carrusel-fantasma {
    display: flex;
    gap: 12px;
    overflow: hidden;
    padding-bottom: 6px;
  }
  .esqueleto {
    flex: 0 0 140px;
    width: 140px;
    height: 210px;
    border-radius: 10px;
    background: linear-gradient(
      100deg,
      var(--cad-borde) 45%,
      rgba(255, 255, 255, 0.08) 50%,
      var(--cad-borde) 55%
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
</style>
