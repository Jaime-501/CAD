<script lang="ts">
  /**
   * T72: bloque esqueleto para estados de carga. `alto`, `ancho` y `radio`
   * aceptan cualquier valor CSS (por defecto, tokens); el shimmer usa
   * `--dur-slow` y se desactiva con `prefers-reduced-motion: reduce`.
   */
  export let alto = 'var(--sp-4)';
  export let ancho = '100%';
  export let radio = 'var(--r-md)';
</script>

<div
  class="esqueleto"
  style="height: {alto}; width: {ancho}; border-radius: {radio};"
  aria-hidden="true"
></div>

<style>
  .esqueleto {
    background: linear-gradient(
      90deg,
      var(--surface-chip) 0%,
      var(--surface-chip-hover) 50%,
      var(--surface-chip) 100%
    );
    background-size: 200% 100%;
    animation: esqueleto-brillo var(--dur-slow) linear infinite;
  }

  @keyframes esqueleto-brillo {
    from {
      background-position: 200% 0;
    }
    to {
      background-position: -200% 0;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .esqueleto {
      animation: none;
    }
  }
</style>
