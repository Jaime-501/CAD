# Guía de desarrollo

Aplicación de escritorio **Wails v2** (backend en Go, frontend en **Svelte 5 + TypeScript + Vite**) que hace scraping de **JKAnime** (proveedor principal: búsqueda, detalle, episodios y fuentes de vídeo) y **AnimeFLV** (solo fallback de búsqueda). Los textos de la interfaz, los comentarios del código y esta documentación están en español.

## Requisitos

- **Docker** (con `docker compose`): es la forma de desarrollar y compilar. Go y la CLI de `wails` **no están en el PATH del host**. La imagen trae Go 1.25, Node 20, Xvfb y la CLI de Wails.
- **Windows 11** para ejecutar el `.exe` resultante (WebView2 Runtime, incluido de serie).
- Node/npm en el host solo hacen falta si trabajas **fuera** del contenedor (`frontend/node_modules`).

## Desarrollo (Docker)

```sh
docker compose up
```

Arranca `wails dev` con el servidor de desarrollo en `0.0.0.0:34115` y Vite en `:5173`:

- **App en el navegador:** http://localhost:34115 — incluye el puente `window.go`, necesario para llamar a los métodos de Go.
- `http://localhost:5173` es el Vite "puro": **no sirve** para desarrollar contra la app (no tiene el puente).

Comprobaciones sueltas (desde la raíz, con `docker compose run --rm app …`):

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

> En el flujo Docker **no se puede reproducir vídeo real**: el proxy local escucha en `127.0.0.1` dentro del contenedor y el navegador del host no lo alcanza. Para probar la reproducción compila el `.exe` y ejecútalo en Windows.

## Estructura de paquetes

```
main.go                  entrada Wails; incrusta frontend/dist (//go:embed)
app.go                   métodos expuestos a Wails + ciclo de vida (startup/shutdown)
app_nucleo.go            orquestación testeable con dobles (registro, proxy, caché, historial)
frontend/                Svelte 5 + TS + Vite; wailsjs/ es código GENERADO (no editar)
internal/logx            logging estructurado con redacción de secretos
internal/httpx           cliente HTTP con cookiejar, errores tipados y reintentos
internal/models          modelos de dominio (sin dependencias del proyecto)
internal/scraper         interfaz Provider, registro y errores comunes
internal/scraper/jkanime            JKAnime: búsqueda, detalle, episodios AJAX, página de episodio
internal/scraper/jkanime/extractors extractores de vídeo (jkplayer, jk.php, var servers, MediaFire)
internal/scraper/animeflv           AnimeFLV: solo búsqueda (fallback)
internal/stream          proxy local de streaming (127.0.0.1) + reescritura de manifiestos
internal/cache           caché de detalle/episodios con TTL
internal/db              historial de visualización (JSON en disco)
docs/firmas/<pkg>.md     firmas reales de cada paquete (si chocan con el código, manda el código)
tareas/                  plan de trabajo por tareas (reglas en 00_REGLAS.md, detalle en Txx.md)
PROGRESO.md              estado del plan, deudas y pendientes
docs/BITACORA.md         desviaciones históricas y pendientes cerrados
AGENTS.md                guía para agentes de código
```

Regla de dependencias: `models` no importa nada del proyecto; los extractores no importan `jkanime`; los providers implementan `scraper.Provider` y se registran en `scraper/registry.go`.

## Tests y tags

- **Sin red (por defecto):** `go test -race -count=1 ./...`. Todo corre sin internet (`httptest.Server` y fixtures en `testdata/`).
- **Con red (`integration`):** los smoke tests contra los sitios reales llevan `//go:build integration` y viven en archivos `*_integration_test.go`:

  ```sh
  docker compose run --rm app go test -tags integration -count=1 ./...
  ```

  Incluyen la búsqueda real de AnimeFLV y JKAnime y el **E2E completo** (`app_e2e_integration_test.go`: búsqueda → detalle con streaming → fuentes → manifiesto por el proxy → historial). Si un dominio cambia o el sitio cae, fallan **a propósito**.
- El E2E **offline** (`TestE2EFlujoCompletoOffline`, en `app_e2e_test.go`) recorre el mismo flujo contra un sitio falso montado con los fixtures reales, exige que las peticiones sean exactamente las esperadas y que nada salga a la red.
- **Fixtures:** son la respuesta cruda del servidor, sanitizada (`TOKEN_DE_PRUEBA`, `E_PRUEBA`…), sin cookies ni tokens reales. Cápturalos con `curl.exe` y revísalos antes de subirlos.

## Regenerar los bindings de Wails

`frontend/wailsjs/` es **generado**: nunca se edita a mano.

```sh
docker compose run --rm app wails generate module
```

`wails dev` y `wails build` también lo regeneran. Los métodos de `app.go` deben devolver `(valor, error)` como máximo (Wails ignora retornos extra). Si el frontend importa algo que no existe, `npm run build` y `npm run check` fallan hasta regenerar. Los payloads de eventos (`episodes:update`, `episodes:complete`) no se generan: se definen a mano en `frontend/src/api.ts`.

## Compilar el .exe de Windows desde Docker

```sh
# Release: la app NO abre consola
docker compose run --rm app wails build -platform windows/amd64

# Depuración: abre una consola con los logs y habilita las herramientas de desarrollo (F12)
docker compose run --rm app wails build -platform windows/amd64 -debug
```

El ejecutable queda en `build/bin/miapp.exe` (ignorado por git). La compilación regenera los bindings y recompila el frontend. La variable de entorno `MIAPP_DEV` activa logs en nivel `Debug` con formato legible (solo cambia nivel y formato; la redacción de secretos se aplica siempre).

## Datos que guarda la app

- **Historial:** `%AppData%\miapp\historial.json` (`os.UserConfigDir()/miapp`). Un registro por episodio (`anime_id`, `episode_id`, `current_time`, `total_duration`, `is_watched`; umbral 0.90). Escritura atómica; si el archivo se corrompe se renombra a `historial.json.bak` y se empieza vacío.
- **Carpeta `miapp.exe`:** la crea WebView2 para su perfil (caché, cookies del webview). No es de la app.
- **Caché de detalle/episodios:** solo en memoria, con TTL; no se persiste.
- En Docker, el historial se escribe dentro del contenedor, no en `%AppData%`.

## Limitaciones conocidas (técnicas)

1. **AnimeFLV es solo búsqueda:** `GetAnimeDetails` devuelve `scraper.ErrNoDetails` y no expone fuentes de vídeo. Se descartó por ahora; hay un posible cambio de proveedor a futuro.
2. **"Continuar" sin marca de tiempo:** `WatchProgress` no guarda cuándo se guardó, así que el botón elige el episodio de mayor número con progreso. Arreglarlo exige un campo de fecha en `internal/models` + `internal/db` + bindings + frontend.
3. **Proxy no alcanzable desde el navegador del host en Docker** (ver arriba). A propósito: es loopback y con anti-SSRF.
4. No hay borrado de registros del historial.

## Seguridad

- El proxy de streaming solo escucha en `127.0.0.1`, bloquea destinos privados/loopback/link-local por **IP resuelta** (anti-SSRF) y mantiene las cabeceras requeridas (`Referer`, etc.) en la sesión, **nunca** en la URL.
- Los logs pasan por `internal/logx`: cookies, CSRF, `Authorization`/`Set-Cookie` y **URLs firmadas** (`?e=`, `?st=`, …) salen redactados.
- `frontend/index.html` declara `<meta name="referrer" content="no-referrer">`: AnimeFLV devuelve 403 en las portadas si se envía `Referer`. No la quites.

## Publicar una versión (release)

Versionado semántico (`vMAYOR.MENOR.PARCHE`).

1. Todo commiteado y subido; gate en verde (`gofmt -l .`, `go vet ./...`, `go test -race -count=1 ./...`, `npm run check`) y prueba manual con el `.exe`.
2. Sube `productVersion` en `wails.json`.
3. Compila la release (sin `-debug`) y empaqueta:

   ```powershell
   docker compose run --rm app wails build -platform windows/amd64
   Compress-Archive -Path build\bin\miapp.exe -DestinationPath miapp-vX.Y.Z-windows-amd64.zip
   Get-FileHash miapp-vX.Y.Z-windows-amd64.zip -Algorithm SHA256
   ```
4. Crea y sube el tag: `git tag -a vX.Y.Z -m "Versión X.Y.Z"` y `git push origin vX.Y.Z`.
5. En GitHub: **Releases → Draft a new release**, elige el tag, escribe las notas (incluye el SHA256 y el aviso de SmartScreen), adjunta el `.zip` y publica.

## Trabajo con agentes

El proyecto se construyó con un plan de 29 tareas (`tareas/`) ejecutadas por un agente de código. `AGENTS.md` resume las reglas para agentes, `PROGRESO.md` es la memoria entre sesiones y `docs/firmas/` contiene las firmas reales de cada paquete.
