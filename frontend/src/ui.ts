/**
 * Estado del shell con barra lateral (T48).
 *
 * - `sidebarCollapsed`: barra contraída/expandida en pantallas anchas; se
 *   persiste en `localStorage` bajo la clave `cad.sidebarCollapsed`, con todo
 *   el acceso en try/catch (un WebView sin storage no debe romper la app).
 *   Por defecto `false` (expandida).
 * - `drawerOpen`: cajón del historial en modo estrecho; no se persiste.
 *   Por defecto `false`.
 * - `isNarrow`: modo estrecho (< 900 px) vía `matchMedia`, con listener de
 *   cambios y limpieza al soltar el último suscriptor (readable).
 * - `pestanaHome`: pestaña activa del Home (`'descubrir' | 'favoritos'`), que
 *   comparten `VistaInicio` y la barra inferior (T73); no se persiste.
 * - `limpiarBusqueda`: contador que, al subir, pide a `VistaBusqueda` vaciar su
 *   `texto` local (T73); no se persiste.
 */
import { get, readable, writable } from 'svelte/store';

const CLAVE_CONTRAIDA = 'cad.sidebarCollapsed';

function leerContraida(): boolean {
  try {
    return localStorage.getItem(CLAVE_CONTRAIDA) === 'true';
  } catch {
    return false;
  }
}

export const sidebarCollapsed = writable<boolean>(leerContraida());

sidebarCollapsed.subscribe((valor) => {
  try {
    localStorage.setItem(CLAVE_CONTRAIDA, String(valor));
  } catch {
    // sin storage disponible: el estado sigue vivo en memoria
  }
});

export const drawerOpen = writable<boolean>(false);

/** T73: pestaña activa del Home, compartida con la barra inferior. */
export const pestanaHome = writable<'descubrir' | 'favoritos'>('descubrir');

/** T73: al subir, `VistaBusqueda` vacía su `texto` local (contador). */
export const limpiarBusqueda = writable(0);

export const isNarrow = readable<boolean>(false, (set) => {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    set(false);
    return () => {};
  }
  const consulta = window.matchMedia('(max-width: 899px)');
  const cambiar = (e: MediaQueryListEvent) => set(e.matches);
  set(consulta.matches);
  consulta.addEventListener('change', cambiar);
  return () => consulta.removeEventListener('change', cambiar);
});
