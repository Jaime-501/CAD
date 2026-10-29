# Informe T96 — Diagnóstico del `pageerror` «Cannot read properties of null (reading 'nodes')»

## Resumen

El `pageerror` `TypeError: Cannot read properties of null (reading 'nodes')` que aparece ya en el Home con el dev server
(**`wails dev` en `localhost:34115`**) es **ruido del dev server**. No existe en el bundle de producción, no proviene del
código de la aplicación (`frontend/src`) y **no afecta al `.exe`**. No hay bug de código de la app; no se abre T96b.

## Método

1. Estado inicial: `git status --porcelain` vacío. Rama de trabajo: `chore/higiene-v1.5` (desviación anotada: AGENTS.md
   menciona `feature/pantalla-completa`; no se cambió de rama).
2. Script Playwright temporal **fuera del repo** (`C:\Users\azuls\AppData\Local\Temp\opencode\pageerror-t96/capture.mjs`):
   abre `http://localhost:34115/`, escucha `pageerror` y `console`, guarda `error.stack` completo, registra el momento
   relativo a `DOMContentLoaded` / `load` y un sondeo de pintado del Home, y repite 3 veces.
3. Se capturó también el fichero `http://localhost:34115/wails/ipc.js` para identificar su naturaleza.
4. Contraste con producción: `npm run build` (vite) → `frontend/dist`, servido con un servidor estático sencillo en
   `localhost:8080`, misma captura 3 veces. Sin el puente `window.go` la app emite avisos `reading 'main'` (esperados y
   capturados por la app, no son `pageerror`); se compara por mensaje exacto para no confundir con el investigado.
5. Se consultó `git checkout 876d14c -- frontend/src` (T89b) **no se aplicó**: no es material porque el origen está en
   infra del dev server (`wails/ipc.js`), independiente de `frontend/src`, y `nodes` no existe en `frontend/src` en
   ninguna versión. El contraste con producción ya demuestra que el patrón no viaja al bundle.

## Traza completa (3 repeticiones, dev server)

Todas las repeticiones son **deterministas**: 1 `pageerror` por repetición, mismo mensaje, misma pila, mismo origen.

URL de la página: `http://localhost:34115/` (sin parámetros firmados ni tokens; la pila no referencia URLs firmadas).

### Repetición 1
- `DOMContentLoaded`: 170 ms
- `load`: 204 ms
- `pageerror` a los **187 ms** (justo tras `DOMContentLoaded`, antes/al terminar el pintado inicial del Home)
- Mensaje: `Cannot read properties of null (reading 'nodes')`
- Pila:
```
TypeError: Cannot read properties of null (reading 'nodes')
    at Fe (http://localhost:34115/wails/ipc.js:32:26544)
    at bn (http://localhost:34115/wails/ipc.js:32:27002)
    at new In (http://localhost:34115/wails/ipc.js:41:87)
    at http://localhost:34115/wails/ipc.js:41:367
```

### Repetición 2
- `DOMContentLoaded`: 115 ms
- `load`: 158 ms
- `pageerror` a los **123 ms**
- Misma pila que la repetición 1.

### Repetición 3
- `DOMContentLoaded`: 120 ms
- `load`: 168 ms
- `pageerror` a los **128 ms**
- Misma pila que la repetición 1.

Conteo total: **3 `pageerror` en 3 repeticiones** (100 % determinista). Ningún error ajeno (`ERR_EMPTY_RESPONSE`, etc.)
en el log del script.

## Localización del origen

La pila apunta **exclusivamente** a `http://localhost:34115/wails/ipc.js` — un fichero servido por el servidor de
desarrollo de Wails, **no** a `frontend/src`, no a Vite, no al cliente de Wails de producción.

`wails/ipc.js` es el inyector de IPC del dev server (`wails dev`):
- abre con `console.log("%c wails dev %c ...", ...)`;
- sólo se activa cuando `process.env.NODE_ENV` no empieza por `"prod"`;
- contiene el runtime de Svelte 5 (efectos/bloques con campo `.nodes`).

Conclusión de la localización:
- **a. código de la app (`frontend/src`)**: NO. `grep nodes` en `frontend/src` → 0 coincidencias.
- **b. algo que sólo existe en el dev server**: SÍ. Es `wails/ipc.js`, el inyector/runtime de Svelte 5 que Wails
  inyecta únicamente en modo desarrollo.

## Contraste con el bundle de producción

`frontend/dist` servido estáticamente en `http://localhost:8080/` (misma captura, 3 repeticiones):

| Repetición | pageerror | console relevante (no son pageerror) |
|---|---|---|
| 1 | 0 | `reading 'main'` (x4 al inicio, x1 a los ~576 ms) |
| 2 | 0 | `reading 'main'` (x4 al inicio, x1 a los ~574 ms) |
| 3 | 0 | `reading 'main'` (x4 al inicio, x1 a los ~574 ms) |

- **Total pageerror en producción: 0 de 3.**
- Los únicos errores son avisos `TypeError: Cannot read properties of undefined (reading 'main')` emitidos por la app
  al no encontrar el puente `window.go` (ausente sin Wails). Son `console.warn` capturados por la app, **no**
  `pageerror`, y el mensaje es distinto (`reading 'main'` vs `reading 'nodes'`): se comparó por mensaje exacto, no hay
  confusión posible.
- El bundle de producción **no reproduce** el `pageerror` de `nodes`.

## Comparación con T89b (no medida)

No se ejecutó con T89b. Al provenir el error de /wails/ipc.js, ajeno a frontend/src, se infiere que el resultado sería idéntico; es una inferencia, no una medición.

## Veredicto (único)

**Ruido del dev server.** Justificación:
1. El `pageerror` surge íntegramente de `wails/ipc.js`, el inyector de Wails que sólo existe en `wails dev` (modo
   desarrollo, `NODE_ENV !== prod`). No es parte del código de la app ni del cliente de Wails de producción.
2. `nodes` no aparece en `frontend/src` (0 coincidencias).
3. El bundle de producción (`frontend/dist`) no reproduce el error: 0 `pageerror` en 3 repeticiones.
4. El `.exe` se construye con `wails build` (modo producción, sin `/wails/ipc.js` dev injector), por lo que este
   `pageerror` **no afecta al `.exe`**.

No es un bug de código de la aplicación; **no se recomienda abrir T96b**. Si el producto quiere eliminarlo incluso del
dev server, habría que investigar por qué el runtime de Svelte 5 en `wails/ipc.js` accede a `.nodes` de un efecto nulo
durante el teardown justo después de `DOMContentLoaded`, pero eso es ajeno al `.exe`.
Que wails build no inyecte /wails/ipc.js no se midió con el .exe; se confirma con -debug + F12 en la prueba manual de la v1.5.x

## Evidencia anexa

- Script temporal (fuera del repo): `C:\Users\azuls\AppData\Local\Temp\opencode/pageerror-t96/capture.mjs`
  (y su variante `capture-prod.mjs` para producción).
- Pila íntegra reproducida arriba (sin URLs firmadas ni tokens).
- `git status --porcelain` al finalizar: sólo `docs/investigacion/pageerror_nodes.md` y `PROGRESO.md`.
- `git diff --stat -- '*.go' frontend/src frontend/wailsjs`: vacío; `frontend/src` sin modificar respecto a HEAD.
- `frontend/package.json.md5`: no modificado (el `npm run build` de Vite no lo toca).

## Propuesta

Ninguna corrección de producto (no hay bug en la app). Cerrar el pendiente del `pageerror` en `PROGRESO.md` como
«ruido del dev server, no afecta al .exe». No iniciar T97.
