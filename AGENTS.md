# AGENTS.md
 
Aplicación de escritorio Wails v2 (módulo Go y nombre de la app: `miapp`): Backend en Go + Frontend en Svelte 5 / TypeScript / Vite en `frontend/`. Hace scraping de **JKAnime** (proveedor principal) y **AnimeFLV** (fallback, solo búsqueda por ahora) y reproduce HLS/MP4. Los comentarios y los textos de la interfaz están en español — mantén esa coherencia.
 
## Comandos
 
- `docker compose up` — flujo principal de desarrollo (Go/wails **no están en el PATH** en esta máquina; la imagen tiene Go 1.25, Node 20, y la CLI de wails). Ejecuta `xvfb-run -a wails dev -browser=false -devserver 0.0.0.0:34115`; interfaz de la app en el navegador en http://localhost:34115, Vite en :5173.
- Comprobaciones específicas de Go sin iniciar la app: `docker compose run --rm app go build ./...` / `go vet ./...` / `go test -race -count=1 ./...`.
- Tests con red real (smoke): llevan build tag `integration`, p. ej. `docker compose run --rm app go test -tags integration ./internal/scraper/animeflv/ -v` (o el paquete que corresponda). `go test ./...` normal debe pasar **sin red**.
- Solo frontend, desde `frontend/`: `npm install` (requerido primero), `npm run dev`, `npm run build`, `npm run check` (svelte-check; la única comprobación de tipos).
- Sin linters, sin CI. Verificación = `gofmt -l .` (sin salida) + `go vet ./...` (también con `-tags integration`) + `go test -race -count=1 ./...` + `go build ./...` + `npm run check` (este último solo en tareas de frontend). `-race` funciona en el contenedor (CGO=1, gcc).
## Consideraciones importantes (Gotchas)
 
- `wails dev` enlaza su servidor de desarrollo a `127.0.0.1` por defecto → inaccesible a través del mapeo de puertos de Docker. La bandera `-devserver 0.0.0.0:34115` en `docker-compose.yml` es requerida, no cosmética; la bandera necesita el formato `host:puerto`.
- El desarrollo en el navegador debe usar **`localhost:34115`** (servidor de desarrollo de Wails), no `:5173`. El servidor propio de Vite no tiene el puente `window.go` — las llamadas fallan con `Cannot read properties of undefined (reading 'main')`. `App.svelte` lo protege y muestra un mensaje apuntando a :34115.
- `frontend/wailsjs/**` es generado automáticamente (NO EDITAR). Solo se regenera con `wails dev`/`wails build`/`wails generate module` (previsto en T23). Si `App.svelte` importa una función que falta ahí, `npm run build`/`check` fallarán hasta regenerar.
- `main.go` incrusta `frontend/dist` (`//go:embed all:frontend/dist`): `go build` necesita que `frontend/dist` exista (está en el .gitignore; prodúcelo con `npm run build`).
- La configuración de Vite está fijada: `0.0.0.0:5173`, `strictPort`, y observación por polling — deliberado, para la observación de archivos entre Docker y Windows. No lo elimines.
- Wails serializa a JSON con `encoding/json`: los campos de los modelos necesitan etiquetas `json:` en minúsculas y snake_case (el frontend lee `anime.cover_url`). Las etiquetas también determinan la generación de los modelos de TS.
- `frontend/index.html` tiene `<meta name="referrer" content="no-referrer">`: animeflv.or.at devuelve **403 para las imágenes de portada** cuando se envía cualquier Referer. No elimines esta etiqueta. El Referer que exigen los servidores de vídeo se inyecta en el proxy local (T19), no en el navegador.
## Arquitectura (Go)
 
- Paquetes: `internal/{logx,httpx,models,scraper,scraper/jkanime,scraper/jkanime/extractors,scraper/animeflv,stream,cache,db}`. Layout completo y reglas en `tareas/00_REGLAS.md`.
- Regla de dependencias: `models` no importa nada del proyecto; `extractors` no importa `jkanime`; los providers implementan `scraper.Provider` y se registran en `scraper/registry.go`.
- Toda petición HTTP pasa por `internal/httpx` (`HTTPDoer`, `NewClient` con cookiejar, `ScraperError`, `ErrCloudflareChallenge`, reintentos). Los logs pasan por `internal/logx` (redacta tokens y URLs firmadas): nunca registres URLs firmadas completas.
- Firmas reales de cada paquete: `docs/firmas/<paquete>.md`. Si el código y la documentación discrepan, **manda el código**; corrige el documento.
## Scraper
 
- Scraping con **goquery** sobre el HTML crudo del servidor (sin JS). Los fixtures en `testdata/` son la respuesta cruda sanitizada (`TOKEN_DE_PRUEBA`, sin cookies); captúralos con `curl.exe` y usa Playwright MCP solo si hay Cloudflare o hay que observar peticiones de red. No inventes HTML ni selectores.
- **JKAnime** (`internal/scraper/jkanime/`): búsqueda, detalle, episodios por AJAX (`POST /ajax/episodes/<animeID>/<página>` con cookiejar + CSRF; 419 sin cookie/token → recuperación de sesión), página de episodio y extractores (`jkplayer`, `jk.php`, `var servers`, MediaFire). URLs de episodio: `/<slug>/<n>/`, película `/<slug>/pelicula/`, especial redirige a `/especial/`. `GetVideoSources` devuelve fuentes normalizadas; `SelectBestSource` prefiere HLS > MP4 > iframe (un iframe nunca se elige si hay HLS/MP4).
- **AnimeFLV** (`internal/scraper/animeflv/`, paquete `animeflv`): apunta a **`https://animeflv.or.at`** (WordPress; no "arregles" la URL al antiguo animeflv.net).
  - Búsqueda `/?s=<query>`; resultados en `.search-series-card` (`a.thumbnail-link`, `h2.entry-title`, `img.anime-image`). Las series viven en `/anime/<slug>/`.
  - Los selectores antiguos (`article.Anime`, `h3.Title`, `/browse?q=`) NO existen aquí.
  - Estado actual: **solo búsqueda** (`HasDetails=false`; `GetAnimeDetails` devuelve `scraper.ErrNoDetails`; `GetVideoSources` devuelve error descriptivo). La extracción de episodios/vídeo se **descartó por ahora** (posible cambio de proveedor a futuro). No la implementes sin una tarea explícita.
  - Envía un User-Agent de navegador y falla si el código HTTP no es 200.
- Smoke test: `internal/scraper/animeflv/animeflv_integration_test.go` (build tag `integration`) busca "bleach"; una caída o cambio de dominio lo hará fallar — es el objetivo.
## Configuración
 
- `wails.json` maneja los hooks del frontend: install/build/dev = `npm install` / `npm run build` / `npm run dev`, `serverUrl: auto`.
- `.gitignore` excluye `build/bin`, `node_modules`, `frontend/dist` y `.playwright-mcp/`.
## Plan de trabajo por tareas
 
- Reglas y layout: `tareas/00_REGLAS.md`. Estado: `PROGRESO.md`. Detalle: `tareas/Txx.md`. Firmas: `docs/firmas/*.md`. Bitácora: `docs/BITACORA.md`.
- Trabaja **SOLO** la tarea indicada. Lee solo su `tareas/Txx.md` y los `docs/firmas/*.md` de los paquetes de los que depende; no leas tareas futuras. Respeta "Puedes tocar" / "NO debes tocar": si necesitas un archivo fuera de la lista, detente y repórtalo.
- Actualiza `PROGRESO.md` en cada hito (no solo al final) y al terminar: estado, firmas, desviaciones, pendientes. No empieces la siguiente tarea.
- Todo comando Go/wails se ejecuta con `docker compose run --rm app <comando>`. Donde un archivo de tarea diga `go ...` o `wails ...`, aplica esto. `npm` se ejecuta en `frontend/`.
- Los smoke tests con red van en un archivo aparte `*_integration_test.go` (mismo paquete, `//go:build integration`). Está permitido crearlo aunque no figure en "Puedes tocar".
- `app.go` expone `SearchAnime(query string) ([]models.Anime, error)`, que usa `App.svelte`. Consérvalo (puede delegar en el registro de providers) o anota el cambio de nombre en `PROGRESO.md` para que T25 actualice `App.svelte`. No lo rompas sin autorización explícita de la tarea.
- Los modelos nuevos usan tags `json` en minúsculas y snake_case (como `cover_url`).
- **Windows 11 + PowerShell** en el host: no uses `grep`, `head` ni `sed`. Usa `curl.exe` (no `curl`), `Select-String` y `Select-Object -First N`, o ejecuta dentro del contenedor con `docker compose run --rm app sh -c "..."`. Ejecuta desde la raíz del proyecto.
- Seguridad: el proxy local (T19) escucha solo en `127.0.0.1`, bloquea destinos privados/loopback/link-local por IP resuelta y mantiene las cabeceras requeridas en la sesión, nunca en la URL.
