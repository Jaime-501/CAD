<script lang="ts">
  /**
   * T72: set de iconos SVG inline (paths al estilo Heroicons, MIT). Sustituye a
   * los glifos sueltos (`★ ▸ ←`) que usaban las vistas.
   *
   * `tam` es el único lugar con tamaños en píxeles del set (16 | 20 | 24). Los
   * iconos «llenos» (`estrella-llena`, `marcador-lleno`, `play`, `pausa`) usan
   * `fill="currentColor"`; el resto son de trazo (`stroke="currentColor"`).
   */
  export let nombre: string;
  /** Tamaño en píxeles del icono; es el único px literal permitido en T72. */
  export let tam: 16 | 20 | 24 = 20;
  /** Si se pasa, el icono deja de ser decorativo: `role="img"` + `aria-label`. */
  export let etiqueta = '';

  interface DefIcono {
    /** true = relleno con `currentColor`; false = solo trazo. */
    lleno?: boolean;
    paths: string[];
  }

  const ICONOS: Record<string, DefIcono> = {
    'flecha-izq': { paths: ['M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18'] },
    'flecha-der': { paths: ['M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3'] },
    'chevron-izq': { paths: ['M15.75 19.5 8.25 12l7.5-7.5'] },
    'chevron-der': { paths: ['m8.25 4.5 7.5 7.5-7.5 7.5'] },
    'chevron-abajo': { paths: ['m19.5 8.25-7.5 7.5-7.5-7.5'] },
    play: {
      lleno: true,
      paths: [
        'M4.5 5.653c0-1.427 1.529-2.33 2.779-1.643l11.54 6.347c1.295.712 1.295 2.573 0 3.286L7.28 19.99c-1.25.687-2.779-.217-2.779-1.643V5.653Z',
      ],
    },
    pausa: {
      lleno: true,
      paths: [
        'M6.75 5.25A.75.75 0 0 1 7.5 4.5H9a.75.75 0 0 1 .75.75v13.5A.75.75 0 0 1 9 19.5H7.5a.75.75 0 0 1-.75-.75V5.25Z',
        'M14.25 5.25a.75.75 0 0 1 .75-.75h1.5a.75.75 0 0 1 .75.75v13.5a.75.75 0 0 1-.75.75H15a.75.75 0 0 1-.75-.75V5.25Z',
      ],
    },
    estrella: {
      paths: [
        'M11.48 3.499a.562.562 0 0 1 1.04 0l2.125 5.111a.563.563 0 0 0 .475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 0 0-.182.557l1.285 5.385a.562.562 0 0 1-.84.61l-4.725-2.885a.562.562 0 0 0-.586 0L6.982 20.54a.562.562 0 0 1-.84-.61l1.285-5.386a.562.562 0 0 0-.182-.557l-4.204-3.602a.562.562 0 0 1 .321-.988l5.518-.442a.563.563 0 0 0 .475-.345L11.48 3.5Z',
      ],
    },
    'estrella-llena': {
      lleno: true,
      paths: [
        'M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.006 5.404.434c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354 7.373 21.18c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.434 2.082-5.005Z',
      ],
    },
    marcador: {
      paths: [
        'M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z',
      ],
    },
    'marcador-lleno': {
      lleno: true,
      paths: [
        'M6.32 2.577a49.255 49.255 0 0 1 11.36 0c1.497.174 2.57 1.46 2.57 2.93V21a.75.75 0 0 1-1.085.67L12 18.089l-7.165 3.583A.75.75 0 0 1 3.75 21V5.507c0-1.47 1.073-2.756 2.57-2.93Z',
      ],
    },
    buscar: {
      paths: [
        'm21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z',
      ],
    },
    brujula: {
      paths: [
        'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z',
        'm15.75 8.25-2.25 5.25-5.25 2.25 2.25-5.25 5.25-2.25Z',
      ],
    },
    reloj: {
      paths: ['M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z'],
    },
    refrescar: {
      paths: [
        'M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99',
      ],
    },
    cerrar: { paths: ['M6 18 18 6M6 6l12 12'] },
    papelera: {
      paths: [
        'm14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0',
      ],
    },
    'orden-asc': {
      paths: ['M3 4.5h14.25M3 9h9.75M3 13.5h9.75m4.5-4.5v12m0 0-3.75-3.75M17.25 21 21 17.25'],
    },
    'orden-desc': {
      paths: ['M3 4.5h14.25M3 9h9.75M3 13.5h5.25m5.25-.75L17.25 9m0 0L21 12.75M17.25 9v12'],
    },
    historial: {
      paths: ['M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z'],
    },
    'pantalla-completa': {
      paths: [
        'M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15',
      ],
    },
    menu: { paths: ['M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5'] },
    check: { paths: ['m4.5 12.75 6 6 9-13.5'] },
    alerta: {
      paths: [
        'M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z',
      ],
    },
  };

  $: def = ICONOS[nombre] ?? { paths: [] };
  $: accesible = etiqueta !== '';
</script>

<svg
  class="icono"
  width={tam}
  height={tam}
  viewBox="0 0 24 24"
  fill={def.lleno ? 'currentColor' : 'none'}
  stroke={def.lleno ? 'none' : 'currentColor'}
  stroke-width="1.5"
  stroke-linecap="round"
  stroke-linejoin="round"
  role={accesible ? 'img' : undefined}
  aria-hidden={accesible ? undefined : 'true'}
  aria-label={accesible ? etiqueta : undefined}
  focusable="false"
>
  {#each def.paths as d}
    <path {d} />
  {/each}
</svg>

<style>
  .icono {
    display: inline-block;
    vertical-align: middle;
    flex-shrink: 0;
  }
</style>
