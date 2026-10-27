/**
 * Tests del módulo puro de pantalla completa (T91): lectura del documento y
 * sincronización de la ventana nativa (cola, `propia`, `liberado` y robustez
 * ante fallos de las dependencias).
 *
 * El frontend no tiene runner de tests (fuera de `frontend/src/**` no se puede
 * tocar `package.json`), así que este archivo se ejecuta a mano con el
 * esbuild que ya trae Vite:
 *
 *   docker compose run --rm app sh -c \
 *     "cd frontend && ./node_modules/.bin/esbuild src/pantallacompleta.test.ts \
 *       --bundle --platform=node --format=cjs --outfile=/tmp/pantallacompleta.test.js \
 *       && node /tmp/pantallacompleta.test.js"
 */
import { crearSincronizador, elementoEnPantallaCompleta } from './pantallacompleta';
import type {
  DependenciasVentana,
  DocumentoConPantallaCompleta,
  Sincronizador,
} from './pantallacompleta';

let comprobaciones = 0;

function comprueba(nombre: string, obtenido: unknown, esperado: unknown): void {
  comprobaciones += 1;
  const a = JSON.stringify(obtenido);
  const b = JSON.stringify(esperado);
  if (a !== b) {
    throw new Error(`${nombre}: obtenido ${a}, esperado ${b}`);
  }
}

/** Estado observable de la ventana nativa falsa. */
interface EstadoVentana {
  completo: boolean;
  consultas: number;
  entradas: number;
  salidas: number;
}

/**
 * Doble con estado: `entrar()`/`salir()` cambian el estado de la ventana falsa,
 * así una consulta posterior refleja lo que hizo la llamada anterior.
 */
function ventanaFalsa(
  inicial = false
): { dep: DependenciasVentana; estado: EstadoVentana } {
  const estado: EstadoVentana = {
    completo: inicial,
    consultas: 0,
    entradas: 0,
    salidas: 0,
  };
  const dep: DependenciasVentana = {
    async esPantallaCompleta() {
      estado.consultas += 1;
      return estado.completo;
    },
    entrar() {
      estado.entradas += 1;
      estado.completo = true;
    },
    salir() {
      estado.salidas += 1;
      estado.completo = false;
    },
  };
  return { dep, estado };
}

/** Deja correr las microtareas pendientes (una entrada de la cola). */
async function tick(): Promise<void> {
  await Promise.resolve();
  await Promise.resolve();
}

async function principal(): Promise<void> {
  // 1. elementoEnPantallaCompleta: null, undefined, objeto y documento sin la
  //    propiedad.
  const video = { tagName: 'VIDEO' } as unknown as Element;
  comprueba('elemento null', elementoEnPantallaCompleta({ fullscreenElement: null }), null);
  comprueba(
    'elemento undefined',
    elementoEnPantallaCompleta({ fullscreenElement: undefined }),
    null
  );
  comprueba('documento sin la propiedad', elementoEnPantallaCompleta({}), null);
  comprueba(
    'elemento presente se devuelve tal cual',
    elementoEnPantallaCompleta({ fullscreenElement: video }) === video,
    true
  );
  comprueba(
    'documento nulo no lanza',
    elementoEnPantallaCompleta(null as unknown as DocumentoConPantallaCompleta),
    null
  );

  // 2. Ventana normal: entrar al haber elemento y salir al quitarlo.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    comprueba('normal: consulta 1', estado.consultas, 1);
    comprueba('normal: entra 1', estado.entradas, 1);
    comprueba('normal: sale 0', estado.salidas, 0);
    comprueba('normal: ventana completa', estado.completo, true);
    await sinc.alCambiarDocumento(false);
    comprueba('normal: sale 1 al quitar el elemento', estado.salidas, 1);
    comprueba('normal: ventana fuera', estado.completo, false);
  }

  // 3. Ventana ya en fullscreen (ajena): se respeta al salir.
  {
    const { dep, estado } = ventanaFalsa(true);
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    comprueba('ya completa: consulta 1', estado.consultas, 1);
    comprueba('ya completa: no entra', estado.entradas, 0);
    await sinc.alCambiarDocumento(false);
    comprueba('ya completa: no sale (no era propia)', estado.salidas, 0);
    comprueba('ya completa: sigue completa', estado.completo, true);
  }

  // 4. Dos `true` seguidos (video -> iframe): solo se entra y se consulta una vez.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    await sinc.alCambiarDocumento(true);
    comprueba('dos true: consulta 1', estado.consultas, 1);
    comprueba('dos true: entra 1', estado.entradas, 1);
    comprueba('dos true: sale 0', estado.salidas, 0);
  }

  // 5. `false` sin haber entrado: nada.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(false);
    comprueba('false sin propia: consulta 0', estado.consultas, 0);
    comprueba('false sin propia: entra 0', estado.entradas, 0);
    comprueba('false sin propia: sale 0', estado.salidas, 0);
  }

  // 6. `true` y `false` lanzados sin await: la cola respeta el orden.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    const primera = sinc.alCambiarDocumento(true);
    const segunda = sinc.alCambiarDocumento(false);
    await Promise.all([primera, segunda]);
    comprueba('sin await: entra 1', estado.entradas, 1);
    comprueba('sin await: sale 1', estado.salidas, 1);
    comprueba('sin await: estado final fuera', estado.completo, false);
  }

  // 7. `false`/`true` rápidos tras haber entrado: salir y volver a entrar.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    const salida = sinc.alCambiarDocumento(false);
    const vuelta = sinc.alCambiarDocumento(true);
    await Promise.all([salida, vuelta]);
    comprueba('rapido: entra 2', estado.entradas, 2);
    comprueba('rapido: sale 1', estado.salidas, 1);
    comprueba('rapido: consulta 2', estado.consultas, 2);
    comprueba('rapido: estado final completo', estado.completo, true);
  }

  // 8. `liberar`: solo restaura si la ventana la puso esta instancia; idempotente.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    sinc.liberar();
    comprueba('liberar con propia: sale 1 síncrono', estado.salidas, 1);
    comprueba('liberar con propia: ventana fuera', estado.completo, false);
    sinc.liberar();
    comprueba('liberar dos veces: sigue sale 1', estado.salidas, 1);
  }
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    sinc.liberar();
    comprueba('liberar sin propia: sale 0', estado.salidas, 0);
    comprueba('liberar sin propia: entra 0', estado.entradas, 0);
  }

  // 9. Tras liberar, los eventos del documento no hacen nada.
  {
    const { dep, estado } = ventanaFalsa(false);
    const sinc = crearSincronizador(dep);
    sinc.liberar();
    await sinc.alCambiarDocumento(true);
    comprueba('liberado: consulta 0', estado.consultas, 0);
    comprueba('liberado: entra 0', estado.entradas, 0);
    await sinc.alCambiarDocumento(false);
    comprueba('liberado: sale 0', estado.salidas, 0);
  }

  // 10. `liberar` con una consulta en vuelo: al resolverse no se entra.
  {
    const estado: EstadoVentana = { completo: false, consultas: 0, entradas: 0, salidas: 0 };
    const control: { resolver: ((valor: boolean) => void) | null } = { resolver: null };
    const dep: DependenciasVentana = {
      esPantallaCompleta() {
        estado.consultas += 1;
        return new Promise<boolean>((resolver) => {
          control.resolver = resolver;
        });
      },
      entrar() {
        estado.entradas += 1;
        estado.completo = true;
      },
      salir() {
        estado.salidas += 1;
        estado.completo = false;
      },
    };
    const sinc: Sincronizador = crearSincronizador(dep);
    const enVuelo = sinc.alCambiarDocumento(true);
    await tick();
    comprueba('en vuelo: consulta 1', estado.consultas, 1);
    const resolver = control.resolver;
    if (!resolver) throw new Error('en vuelo: no se lanzó esPantallaCompleta');
    sinc.liberar();
    resolver(false);
    await enVuelo;
    comprueba('en vuelo + liberar: entra 0', estado.entradas, 0);
    comprueba('en vuelo + liberar: sale 0', estado.salidas, 0);
  }

  // 11. `esPantallaCompleta` rechaza: se trata como false y se entra.
  {
    const { dep, estado } = ventanaFalsa(false);
    dep.esPantallaCompleta = () => Promise.reject(new Error('sin puente'));
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    comprueba('rechazo: entra 1', estado.entradas, 1);
    comprueba('rechazo: ventana completa', estado.completo, true);
    await sinc.alCambiarDocumento(false);
    comprueba('rechazo: sale 1', estado.salidas, 1);
  }

  // 12. `entrar` lanza: no activa `propia` y la cola sigue viva.
  {
    const { dep, estado } = ventanaFalsa(false);
    const entrarOriginal = dep.entrar;
    let fallos = 1;
    dep.entrar = () => {
      if (fallos > 0) {
        fallos -= 1;
        throw new Error('ventana nativa no disponible');
      }
      entrarOriginal();
    };
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    comprueba('entrar lanza: no cuenta entrada', estado.entradas, 0);
    comprueba('entrar lanza: ventana fuera', estado.completo, false);
    await sinc.alCambiarDocumento(true);
    comprueba('entrar lanza: la siguiente sí entra', estado.entradas, 1);
    comprueba('entrar lanza: consulta 2', estado.consultas, 2);
    await sinc.alCambiarDocumento(false);
    comprueba('entrar lanza: sale al quitar', estado.salidas, 1);
  }

  // 13. `salir` lanza: `propia` queda en false igualmente.
  {
    const { dep, estado } = ventanaFalsa(false);
    dep.salir = () => {
      estado.salidas += 1;
      estado.completo = false;
      throw new Error('ventana nativa no disponible');
    };
    const sinc = crearSincronizador(dep);
    await sinc.alCambiarDocumento(true);
    await sinc.alCambiarDocumento(false);
    comprueba('salir lanza: se intentó una vez', estado.salidas, 1);
    await sinc.alCambiarDocumento(false);
    comprueba('salir lanza: no reintenta (propia ya false)', estado.salidas, 1);
    await sinc.alCambiarDocumento(true);
    comprueba('salir lanza: puede volver a entrar', estado.entradas, 2);
  }

  console.log(`OK (${comprobaciones} comprobaciones)`);
}

principal().catch((e: unknown) => {
  console.error(e instanceof Error ? e.message : String(e));
  throw e;
});
