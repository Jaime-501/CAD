# AGENTS.md

Aplicación de escritorio Wails v2 (módulo Go y nombre de la app: `miapp`): Backend en Go + Frontend en Svelte 5 / TypeScript / Vite en `frontend/`. Hace scraping de **JKAnime** y **AnimeAV1** (ambos proveedores completos: búsqueda, detalle, episodios y fuentes) y reproduce HLS/MP4. Los comentarios y los textos de la interfaz están en español — mantén esa coherencia.

**Estado**: proyecto terminado (T1–T35, rama `feature/animeav1`, v1.1.0). Punto de entrada para una persona nueva: `README.md`. Estado de tareas y pendientes de producto: `PROGRESO.md`; decisiones históricas: `docs/BITACORA.md`; reglas de layout: `tareas/00_REGLAS.md`; firmas de API por paquete: `docs/firmas/*.md`. Si el código y la documentación discrepan, **manda el código** y corrige el documento.

## Comandos

- `docker compose up` — flujo principal de desarrollo (Go/wails **no están en el PATH** en esta máquina; la imagen tiene Go 1.25, Node 20, y la CLI de wails). Ejecuta `xvfb-run -a wails dev -browser=false -devserver 0.0.0.0:34115`; interfaz de la app en el navegador en http://localhost:34115, Vite en :5173.
- Comprobaciones específicas de Go sin iniciar la app: `docker compose run --rm app go build ./...` / `go vet ./...` / `go test -race -count=1 ./...`.
- Tests con red real (smoke): llevan build tag `integration` en un archivo aparte `*_integration_test.go`, p. ej. `docker compose run --rm app go test -tags integration ./internal/scraper/animeav1/ -v`; con `./...` se ejecutan además los E2E de la raíz (`TestE2EFlujoCompletoRedReal` y `TestE2EAnimeAV1RedReal`, ~14 s cada uno, y los offline `TestE2EFlujoCompletoOffline`/`TestE2EAnimeAV1FlujoCompletoOffline` van sin tag). `go test ./...` normal debe pasar **sin red**.
- Solo frontend, desde `frontend/`: `npm install` (requerido primero), `npm run dev`, `npm run build`, `npm run check` (svelte-check; la única comprobación de tipos).
- Build de Windows: `docker compose run --rm app wails build -platform windows/amd64` → `build/bin/miapp.exe` (release, sin consola). Con `-debug` sale una consola con los logs. Ver README.
- Sin linters, sin CI. Verificación = `gofmt -l .` (sin salida) + `go vet ./...` (también con `-tags integration`) + `go test -race -count=1 ./...` + `go build ./...` + `npm run check`. `-race` funciona en el contenedor (CGO=1, gcc).

## Consideraciones importantes (Gotchas)

- `wails dev` enlaza su servidor de desarrollo a `127.0.0.1` por defecto → inaccesible a través del mapeo de puertos de Docker. La bandera `-devserver 0.0.0.0:34115` en `docker-compose.yml` es requerida, no cosmética; la bandera necesita el formato `host:puerto`.
- El desarrollo en el navegador debe usar **`localhost:34115`** (servidor de desarrollo de Wails), no `:5173`. El servidor propio de Vite no tiene el puente `window.go` — las llamadas fallan con `Cannot read properties of undefined (reading 'main')`. `App.svelte` lo protege y muestra un mensaje apuntando a :34115.
- El proxy de streaming (T19) solo escucha en `127.0.0.1` **dentro del contenedor**, así que en el flujo de desarrollo por navegador el vídeo HLS/MP4 da `ERR_CONNECTION_REFUSED` y el reproductor salta al siguiente iframe. La reproducción real hay que probarla con el `.exe` en Windows.
- `frontend/wailsjs/**` es generado automáticamente (NO EDITAR). Solo se regenera con `wails dev`/`wails build`/`wails generate module`. Si `App.svelte` importa una función que falta ahí, `npm run build`/`check` fallarán hasta regenerar.
- `main.go` incrusta `frontend/dist` (`//go:embed all:frontend/dist`): `go build` necesita que `frontend/dist` exista (está en el .gitignore; prodúcelo con `npm run build`).
- `main.go` solo declara `OnStartup`/`OnShutdown`/`Bind`: no añadas más hooks ahí (el ciclo de vida vive en `app.go`/`app_nucleo.go`). El `Title` de la ventana sigue siendo el nombre técnico `miapp` (pendiente de producto: cambiarlo obliga a tocar `main.go`).
- La configuración de Vite está fijada: `0.0.0.0:5173`, `strictPort`, y observación por polling — deliberado, para la observación de archivos entre Docker y Windows. No lo elimines.
- Wails serializa a JSON con `encoding/json`: los campos de los modelos necesitan etiquetas `json:` en minúsculas y snake_case (el frontend lee `anime.cover_url`). Las etiquetas también determinan la generación de los modelos de TS.
- `frontend/index.html` tiene `<meta name="referrer" content="no-referrer">`: algunos orígenes devuelven **403 para las imágenes de portada** cuando se envía cualquier Referer (se mantuvo al retirar AnimeFLV; el CDN de AnimeAV1 no lo exige, T30). No elimines esta etiqueta. El Referer que exigen los servidores de vídeo se inyecta en el proxy local, no en el navegador.
- Los fixtures de `testdata/` son HTML crudo **sanitizado**: tokens `TOKEN_DE_PRUEBA`, CSRF `TOKEN_DE_PRUEBA`, firmas `FIRMA_DE_PRUEBA`/`E_PRUEBA`/`ST_UM`/`NONCE_DE_PRUEBA`. Nunca pegues ahí una respuesta real con `st=`, `e=`, cookies o `set-cookie`.

## Arquitectura (Go)

- Paquetes: `internal/{logx,httpx,models,scraper,scraper/jkanime,scraper/jkanime/extractors,scraper/animeav1,stream,cache,db}`. Layout completo y reglas en `tareas/00_REGLAS.md`.
- Regla de dependencias: `models` no importa nada del proyecto; `extractors` no importa `jkanime`; los providers implementan `scraper.Provider` y se registran en `scraper/registry.go`.
- Toda petición HTTP pasa por `internal/httpx` (`HTTPDoer`, `NewClient` con cookiejar, `ScraperError`, `ErrCloudflareChallenge`, reintentos). Los logs pasan por `internal/logx` (redacta tokens y URLs firmadas): nunca registres URLs firmadas completas.
- Firmas reales de cada paquete: `docs/firmas/<paquete>.md`.
- `app.go` expone a Wails `SearchAnime(query string) ([]models.Anime, error)`, `GetAnimeDetails(provider, id string)`, `GetVideoSources(provider, episodeID string, noCache bool)`, `SaveProgress(progress models.WatchProgress) error` y `GetProgress(animeID string)` (`""` = historial entero); todos con retorno `(valor, error)` y usados por `frontend/src/api.ts`. Eventos: `episodes:update` (`models.EpisodeBatch`, campo `err`) y `episodes:complete` (`{anime_id, error}`), consumidos en `frontend/src/stores.ts`.
- El guardado del progreso es atómico (`historial.json` → temporal → rename) con umbral 0.90; la caché de detalle/episodios es solo en memoria (TTL 10 min).

## Scraper

- Scraping con **goquery** sobre el HTML crudo del servidor (sin JS). Los fixtures en `testdata/` son la respuesta cruda sanitizada (ver arriba); captúralos con `curl.exe` y usa Playwright MCP solo si hay Cloudflare o hay que observar peticiones de red. No inventes HTML ni selectores.
- **JKAnime** (`internal/scraper/jkanime/`): búsqueda, detalle, episodios por AJAX (`POST /ajax/episodes/<animeID>/<página>` con cookiejar + CSRF; 419 sin cookie/token → recuperación de sesión), página de episodio y extractores (`jkplayer`, `jk.php`, `var servers`, MediaFire). URLs de episodio: `/<slug>/<n>/`, película `/<slug>/pelicula/`, especial redirige a `/especial/`. `GetVideoSources` devuelve fuentes normalizadas; `SelectBestSource` prefiere HLS > MP4 > iframe (un iframe nunca se elige si hay HLS/MP4).
- **AnimeAV1** (`internal/scraper/animeav1/`): sitio **SvelteKit SSR** sin cookies/CSRF/rate-limit. Vía principal **`GET <ruta>/__data.json`** (JSON *devalue*, referencias por índice; `PageData` + `Unflatten`), no el `<script>` de hidratación. Búsqueda `/catalogo/__data.json?search=…` (no pagina, 20 por página), detalle `/media/<slug>/__data.json` con **todos** los episodios embebidos y fuentes `/media/<slug>/<n>/__data.json` (campo `embeds` `{SUB:[…], DUB:[…]}`). `models.Anime.ID` = slug; portada en `cdn.animeav1.com/covers/<id>.jpg`.
  - **HLS derivado sin petición**: `player.zilla-networks.com/play/<32 hex>` → `…/m3u8/<id>` con `Headers: {"Sec-Fetch-Site":"same-origin"}` — **obligatoria** en los segmentos (403 sin ella); `stream/reenvio.go` la inyecta desde la sesión (T19). `/play/*` **nunca se pide**. Los demás embeds (UPNShare/Voe/Byse/MP4Upload) salen como **iframe sin resolver**.
  - Lot(es) de **50** episodios (`tamLote`); la primera tanda vuelve por valor y el resto por `StreamAnimeDetails`. Sin `Set-Cookie` ni challenge → `httpx.Get` con la UA por defecto basta.
- Smoke test: `internal/scraper/animeav1/animeav1_integration_test.go` (build tag `integration`) con `TestSearchRedReal`, `TestDetalleRedReal` y `TestFuentesRedReal` (pide solo el `.m3u8`, sin segmentos); E2E en la raíz: `TestE2EAnimeAV1FlujoCompletoOffline` (sin tag) y `TestE2EAnimeAV1RedReal` (tag `integration`).

## Configuración

- `wails.json` maneja los hooks del frontend: install/build/dev = `npm install` / `npm run build` / `npm run dev`, `serverUrl: auto`.
- `.gitignore` excluye `build/bin`, `node_modules`, `frontend/dist` y `.playwright-mcp/`.
- **Windows 11** en el host (PowerShell): no uses `grep`, `head` ni `sed` en el host. Usa `curl.exe` (no `curl`), `Select-String` y `Select-Object -First N`, o ejecuta dentro del contenedor con `docker compose run --rm app sh -c "..."`. Ejecuta desde la raíz del proyecto.
