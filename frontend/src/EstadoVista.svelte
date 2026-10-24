<script lang="ts">
  /**
   * T72: estado de una vista (`vacio | error | cargando`). Pinta un `Icono`,
   * un título (`h2`), un texto y un slot de acción opcional.
   *
   * Nota para T91: la app todavía no tiene `h1`; aquí se usa `h2` como pide la
   * ficha (la jerarquía de encabezados se resuelve en T91).
   */
  import Icono from './Icono.svelte';

  export let variante: 'vacio' | 'error' | 'cargando' = 'vacio';
  export let titulo = '';
  export let texto = '';

  const ICONO_POR_VARIANTE: Record<string, string> = {
    vacio: 'brujula',
    error: 'alerta',
    cargando: 'refrescar',
  };

  $: nombreIcono = ICONO_POR_VARIANTE[variante] ?? 'brujula';
  $: rol = variante === 'error' ? 'alert' : variante === 'cargando' ? 'status' : undefined;
</script>

<section class="estado-vista variante-{variante}" role={rol} aria-busy={variante === 'cargando' ? 'true' : undefined}>
  <span class="icono">
    <Icono nombre={nombreIcono} tam={24} />
  </span>
  {#if titulo !== ''}
    <h2 class="titulo">{titulo}</h2>
  {/if}
  {#if texto !== ''}
    <p class="texto">{texto}</p>
  {/if}
  <div class="accion"><slot /></div>
</section>

<style>
  .estado-vista {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--sp-3);
    padding: var(--sp-6) var(--sp-5);
    text-align: center;
  }
  .icono {
    display: inline-flex;
    color: var(--text-3);
  }
  .titulo {
    margin: 0;
    color: var(--text-1);
    font-size: var(--fs-lg);
    line-height: var(--lh-tight);
  }
  .texto {
    margin: 0;
    max-width: 42ch;
    color: var(--text-3);
    font-size: var(--fs-sm);
    line-height: var(--lh);
  }
  .accion:empty {
    display: none;
  }

  .variante-error .icono {
    color: var(--danger);
  }
  .variante-cargando .icono {
    color: var(--brand-400);
  }
</style>
