<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { get } from 'svelte/store';
  import type { models } from './api';
  import { search, searchResults, searchState } from './stores';
  import { isNarrow, limpiarBusqueda } from './ui';
  import { navegarA, type SeleccionAnime } from './views';
  import VistaInicio from './VistaInicio.svelte';
  import Icono from './Icono.svelte';
  import TarjetaAnime from './TarjetaAnime.svelte';
  import EstadoVista from './EstadoVista.svelte';
  import Esqueleto from './Esqueleto.svelte';
  import Btn from './Btn.svelte';
  import { NOMBRE_PROVEEDOR } from './historial';

  /** false = no hay puente `window.go` (Vite puro en :5173); lo decide App.svelte. */
  export let puente = true;

  const RETRASO_BUSQUEDA = 400;

  let texto = '';
  let temporizador: ReturnType<typeof setTimeout> | null = null;

  // T73: la barra inferior sube este contador para vaciar el texto local (la
  // consulta efectiva la limpia `search('')` desde App.svelte al pulsar Home).
  const cancelarLimpiar = limpiarBusqueda.subscribe(() => {
    texto = '';
    if (temporizador !== null) {
      clearTimeout(temporizador);
      temporizador = null;
    }
  });

  onMount(() => {
    // Al volver de otra vista se recupera la última búsqueda (los resultados
    // siguen en `searchResults`, que no se limpia al desmontar).
    texto = get(searchState).query;
  });

  onDestroy(() => {
    cancelarLimpiar();
    if (temporizador !== null) clearTimeout(temporizador);
  });

  /** Cada pulsación reinicia el *debounce* de ~400 ms. */
  function programar() {
    if (temporizador !== null) clearTimeout(temporizador);
    if (!texto.trim()) {
      temporizador = null;
      void search('');
      return;
    }
    if (!puente) return;
    temporizador = setTimeout(() => {
      temporizador = null;
      void search(texto);
    }, RETRASO_BUSQUEDA);
  }

  /** Enter o botón "Buscar": salta el *debounce*. */
  function buscarAhora() {
    if (temporizador !== null) {
      clearTimeout(temporizador);
      temporizador = null;
    }
    if (!puente) return;
    void search(texto);
  }

  /** Botón ✕ (modo estrecho): vacía el texto, cancela el *debounce*, vuelve al Home. */
  function limpiar() {
    texto = '';
    if (temporizador !== null) {
      clearTimeout(temporizador);
      temporizador = null;
    }
    programar();
    document.getElementById('buscador-anime')?.focus();
  }

  function abrir(anime: models.Anime) {
    // Todos los proveedores del registro tienen detalle (T34): toda tarjeta
    // navega al detalle sin reglas especiales.
    const sel: SeleccionAnime = {
      provider: anime.provider,
      animeID: anime.id,
      title: anime.title,
    };
    navegarA('detalle', sel);
  }
</script>

<main class="contenedor">
  <h2>Buscador de anime</h2>

  <div class="buscador">
    <div class="campo">
      {#if $isNarrow}
        <span class="lupa" aria-hidden="true"><Icono nombre="buscar" tam={20} /></span>
      {/if}
      <input
        id="buscador-anime"
        type="text"
        bind:value={texto}
        placeholder="Ej: Death Note, Jujutsu Kaisen..."
        aria-label="Buscar anime"
        enterkeyhint="search"
        on:input={programar}
        on:keydown={(e) => e.key === 'Enter' && buscarAhora()}
      />
      {#if $isNarrow && texto !== ''}
        <button type="button" class="limpiar" aria-label="Borrar búsqueda" on:click={limpiar}>
          <Icono nombre="cerrar" tam={20} />
        </button>
      {/if}
    </div>
    <button type="button" on:click={buscarAhora} disabled={!puente || $searchState.loading}>
      {$searchState.loading ? 'Buscando...' : 'Buscar'}
    </button>
  </div>

  {#if $searchState.loading}
    <!-- T78: fantasma con la MISMA línea de contador para que la llegada de
         resultados no desplace la rejilla. -->
    <p class="contador" aria-live="polite"></p>
    <div class="rejilla" aria-busy="true" aria-label="Buscando resultados">
      {#each [0, 1, 2, 3, 4, 5] as indice (indice)}
        <div class="fantasma" aria-hidden="true">
          <div class="fantasma-portada">
            <Esqueleto alto="100%" ancho="100%" radio="0" />
          </div>
          <div class="fantasma-titulo">
            <Esqueleto alto="100%" ancho="100%" radio="var(--r-sm)" />
          </div>
        </div>
      {/each}
    </div>
  {:else if $searchState.error}
    <EstadoVista variante="error" titulo="No se pudo buscar" texto={$searchState.error}>
      <Btn variante="secundario" onClick={() => void search($searchState.query)}>Reintentar</Btn>
    </EstadoVista>
  {:else if $searchResults.length > 0}
    <p class="contador" aria-live="polite">{$searchResults.length} resultados</p>
    <div class="rejilla">
      {#each $searchResults as anime (anime.provider + ':' + anime.id)}
        <TarjetaAnime
          {anime}
          onAbrir={abrir}
          etiqueta={NOMBRE_PROVEEDOR[anime.provider] ?? anime.provider}
        />
      {/each}
    </div>
  {:else if $searchState.query !== ''}
    <EstadoVista
      variante="vacio"
      titulo="Sin resultados"
      texto={`No encontramos nada para “${$searchState.query}”.`}
    />
  {:else}
    <!-- Home "Descubrir" (T53): consulta vacía y sin resultados → Novedades.
         Al escribir se vuelve a buscar; al vaciar, se vuelve aquí (el estado
         de `home` se conserva: no se repite la petición). -->
    <VistaInicio onAbrir={abrir} />
  {/if}
</main>

<style>
  .contenedor {
    padding: 2rem;
    text-align: left;
    max-width: 1100px;
    margin: 0 auto;
  }
  h2 {
    text-align: center;
    margin-top: 0;
  }
  .buscador {
    display: flex;
    gap: 10px;
    margin-bottom: 1.5rem;
  }
  /* T75: en escritorio el campo no genera caja (`display: contents`), así que
     el input sigue siendo hijo directo del flex de `.buscador` (idéntico a
     antes). En modo estrecho pasa a ser el contenedor visual del buscador. */
  .campo {
    display: contents;
  }
  input {
    flex: 1;
    padding: 10px 15px;
    border-radius: var(--r-md);
    border: none;
    background: var(--surface-input);
    color: var(--text-1);
    font-size: 16px;
  }
  button {
    padding: 10px 20px;
    border-radius: var(--r-md);
    border: none;
    background: var(--brand-500);
    color: var(--on-brand);
    font-weight: bold;
    cursor: pointer;
  }
  button:disabled {
    background: var(--bg-2);
    color: var(--text-3);
    cursor: not-allowed;
  }
  /* Línea de contador (T78): existe también en el estado de carga (vacía, con
     `min-height`) para que la rejilla no se desplace al llegar los resultados. */
  .contador {
    min-height: calc(var(--fs-sm) * var(--lh));
    margin: 0 0 var(--sp-3);
    font-size: var(--fs-sm);
    line-height: var(--lh);
    color: var(--text-3);
    text-align: center;
  }
  /* T78: rejilla de resultados. Móvil (2) → tableta (3) → escritorio
     (auto-fill); mismas tarjetas que el Home (T76). */
  .rejilla {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--sp-3);
  }
  @media (min-width: 480px) {
    .rejilla {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }
  @media (min-width: 900px) {
    .rejilla {
      grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    }
  }
  /* Fantasma de carga (T78): misma forma que `TarjetaAnime` (portada 2:3 +
     bloque de título de 2 líneas) para no desplazar la rejilla al llegar los
     datos. Las medidas replican las de la tarjeta real. */
  .fantasma {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--surface-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-lg);
    box-shadow: var(--sh-card);
    font-size: 13px;
  }
  .fantasma-portada {
    width: calc(100% + 2px);
    margin: 0 -1px;
    aspect-ratio: 2 / 3;
  }
  .fantasma-titulo {
    box-sizing: content-box;
    height: calc(1.3em * 2);
    margin-bottom: 10px;
    padding: 8px 8px 0;
  }
  /* T75: buscador de modo estrecho (< 900 px): lupa dentro del campo, botón
     limpiar, táctiles de `--tap` y padding con safe-area. En ≥ 900 px nada de
     esto se aplica: el buscador de escritorio queda idéntico. */
  @media (max-width: 899px) {
    .contenedor {
      padding: calc(var(--sat) + var(--sp-4)) var(--gutter) var(--sp-4);
    }
    h2 {
      font-size: var(--fs-xl);
      margin: 0 0 var(--sp-3);
      text-align: left;
    }
    .campo {
      display: flex;
      align-items: center;
      position: relative;
      flex: 1;
      min-width: 0;
      background: var(--surface-input);
      border: 1px solid var(--border);
      border-radius: var(--r-md);
    }
    .campo:focus-within {
      box-shadow: var(--ring);
    }
    .campo input {
      box-sizing: border-box;
      min-width: 0;
      min-height: var(--tap);
      background: transparent;
      padding-left: calc(var(--sp-8) + var(--sp-3));
    }
    /* El anillo de foco lo dibuja el campo (`:focus-within`), no el input. */
    .campo input:focus-visible {
      box-shadow: none;
    }
    .lupa {
      position: absolute;
      left: var(--sp-3);
      display: inline-flex;
      color: var(--text-3);
      pointer-events: none;
    }
    .limpiar {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
      width: var(--tap);
      height: var(--tap);
      padding: 0;
      margin-right: var(--sp-1);
      background: transparent;
      border: none;
      border-radius: var(--r-full);
      color: var(--text-3);
      cursor: pointer;
      transition: color var(--dur-fast) var(--ease-std);
    }
    .limpiar:active {
      color: var(--text-1);
    }
    .limpiar:focus-visible {
      outline: none;
      box-shadow: var(--ring);
    }
    .buscador > button {
      box-sizing: border-box;
      min-height: var(--tap);
    }
    .buscador > button:active {
      background: var(--brand-600);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .limpiar {
      transition: none;
    }
  }
</style>
