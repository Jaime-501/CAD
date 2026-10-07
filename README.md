# miapp — comfort-anime-desktop

Aplicación de escritorio **Wails v2** (backend en Go, frontend en **Svelte 5 + TypeScript + Vite**)
que busca anime, muestra sus episodios y los reproduce (HLS/MP4 o iframe) haciendo **scraping**
de dos sitios: **JKAnime** (proveedor principal: búsqueda, detalle, episodios y fuentes de vídeo)
y **AnimeFLV** (solo fallback de búsqueda, ver [Limitación de AnimeFLV](#limitación-de-animeflv)).

Los textos de la interfaz, los comentarios del código y esta documentación están en español.

## Desarrollo (Docker)

En esta máquina **Go y la CLI de `wails` no están en el PATH**: todo se ejecuta dentro del
contenedor del proyecto.

```sh
docker compose up
```

Arranca `wails dev` con servidor de desarrollo en `0.0.0.0:34115` y Vite en `:5173`:

- **App en el navegador (este flujo):** http://localhost:34115 — con el puente `window.go`,
  que es el que permite llamar a los métodos de Go.
- `http://localhost:5173` es el Vite "puro": **no sirve** para desarrollar contra la app
  (no tiene el puente y las llamadas a Go fallan).

Comprobaciones sueltas (siempre desde la raíz y con `docker compose run --rm app …`):

```sh
docker compose run --rm app go build ./...
docker compose run --rm app go vet ./...
docker compose run --rm app go vet -tags integration ./...
docker compose run --rm app sh -c "gofmt -l ."   # sin salida = correcto
docker compose run --rm app go test -race -count=1 ./...
```

Frontend (dentro de `frontend/`, `npm install` la primera vez):

```sh
npm run check   # svelte-check: la única comprobación de tipos
npm run build
```

## Estructura de paquetes

```
main.go                  entrada Wails; incrusta frontend/dist (//go:embed)
app.go                   métodos expuestos a Wails + ciclo de vida (startup/shutdown)
app_nucleo.go            orquestación entera, testeable con dobles (registro, proxy, caché, historial)
frontend/                Svelte 5 + TS + Vite; wailsjs/ es código GENERADO (no editar)
internal/logx            logging estructurado con redacción de secretos
internal/httpx           cliente HTTP con cookiejar, errores tipados y reintentos
internal/models          modelos de dominio (sin dependencias del proyecto)
internal/scraper         interfaz Provider, registro y errores comunes
internal/scraper/jkanime  JKAnime: búsqueda, detalle, episodios AJAX, página de episodio
internal/scraper/jkanime/extractors  extractores de vídeo (jkplayer, jk.php, var servers, MediaFire)
internal/scraper/animeflv AnimeFLV: solo búsqueda (fallback)
internal/stream          proxy local de streaming (127.0.0.1) + reescritura de manifiestos
internal/cache           caché de detalle/episodios con TTL
internal/db              historial de visualización (JSON en disco)
docs/firmas/<pkg>.md     firmas reales de cada paquete (si chocan con el código, manda el código)
tareas/                  reglas (00_REGLAS.md) y detalle de cada tarea (Txx.md)
PROGRESO.md              memoria entre sesiones: estado, deudas y pendientes
docs/BITACORA.md         desviaciones históricas y pendientes cerrados
```

Regla de dependencias: `models` no importa nada del proyecto; los extractores no importan
`jkanime`; los providers implementan `scraper.Provider` y se registran en `scraper/registry.go`.

## Tests y tags

- **Sin red (por defecto):** `go test -race -count=1 ./...` — todos los tests del proyecto
  corren sin salir a internet (`httptest.Server` y fixtures en `testdata/`).
- **Con red (`integration`):** los smoke tests que hablan con los sitios reales llevan el build
  tag `//go:build integration` y viven en archivos `*_integration_test.go`. Se ejecutan con:

  ```sh
  docker compose run --rm app go test -tags integration -count=1 ./...
  ```

  Incluyen: búsqueda real de AnimeFLV (`internal/scraper/animeflv/`), búsqueda real de JKAnime
  (`app_integration_test.go`) y el **E2E completo** (`app_e2e_integration_test.go`:
  búsqueda → detalle con streaming → fuentes → manifiesto por el proxy → historial, ~14 s).
  Si un dominio cambia o el sitio cae, estos tests fallan **a propósito**.
- El E2E **offline** (`TestE2EFlujoCompletoOffline` en `app_e2e_test.go`) recorre exactamente
  el mismo flujo contra un sitio falso montado con los fixtures reales, exige que las peticiones
  sean **exactamente** las esperadas y que **nada salga a la red**.

## Regenerar los bindings de Wails

`frontend/wailsjs/` es **generado** por la CLI de Wails: nunca se edita a mano. Se regenera con:

```sh
docker compose run --rm app wails generate module
```

`wails dev` y `wails build` también lo regeneran. Si añades o renombras un método en `app.go`
(hay que devolver siempre `(valor, error)`: Wails se traga los retornos de más) y el frontend
importa algo que no exista, `npm run build`/`npm run check` fallarán hasta regenerar.

## Compilar el .exe de Windows desde Docker

```sh
docker compose run --rm app wails build -platform windows/amd64
```

El ejecutable queda en **`build/bin/miapp.exe`** (directorio ignorado por git). El build
regenera bindings y recompila el frontend (`npm run build`). Para probar la reproducción real
(HLS/MP4) hay que usar el `.exe` en Windows: en el flujo Docker el proxy escucha en
`127.0.0.1` **dentro del contenedor** y el navegador del host no lo alcanza.

## Limitación de AnimeFLV

AnimeFLV está registrado como **fallback de búsqueda únicamente**:

- `SearchAnime` lo usa si JKAnime falla o no devuelve resultados.
- **No** expone detalle ni episodios (`GetAnimeDetails` → `scraper.ErrNoDetails`, la UI muestra
  "Detalle no disponible para este proveedor") y **no** expone fuentes de vídeo: la extracción de
  episodios/vídeo se descartó por ahora.

## Notas de seguridad

- El proxy de streaming solo escucha en `127.0.0.1`, bloquea destinos privados/loopback/link-local
  por IP resuelta (anti-SSRF) y mantiene las cabeceras requeridas (`Referer`, `Headers`) en la
  sesión, **nunca** en la URL.
- Los logs pasan por `internal/logx`: cookies, CSRF, `Authorization`/`Set-Cookie` y **URLs
  firmadas** (`?e=`, `?st=`, `X-Amz-*`, …) salen redactados. El E2E lo comprueba sobre la salida
  real de un flujo completo.
- `frontend/index.html` declara `<meta name="referrer" content="no-referrer">`: AnimeFLV devuelve
  403 en las portadas si se envía cualquier `Referer`.
