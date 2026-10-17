<script lang="ts">
  /**
   * T72: insignia/etiqueta de estado. Tonos `marca | info | alerta | neutro |
   * indigo`; `font-size` mínimo `--fs-xs` (12 px). `mayusculas` fuerza
   * `text-transform: uppercase`.
   */
  export let tono: 'marca' | 'info' | 'alerta' | 'neutro' | 'indigo' = 'neutro';
  export let mayusculas = false;
</script>

<span class="insignia tono-{tono}" class:mayusculas={mayusculas}>
  <slot />
</span>

<style>
  .insignia {
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
    padding: var(--sp-1) var(--sp-2);
    border: 1px solid transparent;
    border-radius: var(--r-full);
    font-family: var(--font-sans);
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    line-height: var(--lh-tight);
  }
  .mayusculas {
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .tono-marca {
    background: color-mix(in srgb, var(--brand-400) 14%, transparent);
    border-color: var(--border-accent);
    color: var(--brand-400);
  }
  .tono-info {
    background: color-mix(in srgb, var(--info) 14%, transparent);
    border-color: color-mix(in srgb, var(--info) 35%, transparent);
    color: var(--info);
  }
  .tono-alerta {
    background: color-mix(in srgb, var(--alert) 14%, transparent);
    border-color: color-mix(in srgb, var(--alert) 35%, transparent);
    color: var(--alert);
  }
  .tono-neutro {
    background: var(--surface-chip);
    border-color: var(--border);
    color: var(--text-2);
  }
  .tono-indigo {
    background: color-mix(in srgb, var(--indigo) 14%, transparent);
    border-color: color-mix(in srgb, var(--indigo) 35%, transparent);
    color: var(--indigo);
  }
</style>
