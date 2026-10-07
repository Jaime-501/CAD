<script lang="ts">
  import { atras, seleccion } from './views';
</script>

<main class="contenedor">
  <div class="cabecera">
    <button type="button" class="atras" on:click={atras}>← Atrás</button>
    <h2>{$seleccion?.title ?? 'Reproductor'}</h2>
  </div>

  <p class="estado">El reproductor real llega en T27.</p>
</main>

<style>
  .contenedor {
    padding: 2rem;
    text-align: left;
    max-width: 1100px;
    margin: 0 auto;
  }
  .cabecera {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 1.5rem;
  }
  .cabecera h2 {
    margin: 0;
    text-align: left;
  }
  .atras {
    padding: 8px 16px;
    border-radius: 8px;
    border: none;
    background: #4ade80;
    color: #111;
    font-weight: bold;
    cursor: pointer;
    white-space: nowrap;
  }
  .estado {
    color: #a1a1aa;
  }
</style>
