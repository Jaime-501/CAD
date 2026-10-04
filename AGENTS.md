# AGENTS.md

Aplicación de escritorio Wails v2 (módulo Go y nombre de la app: `miapp`): Backend en Go + Frontend en Svelte 5 / TypeScript / Vite en `frontend/`. Hace scraping de animeflv. Los comentarios y los textos de la interfaz están en español — mantén esa coherencia.

## Comandos

- `docker compose up` — flujo principal de desarrollo (Go/wails **no están en el PATH** en esta máquina; la imagen tiene Go 1.25, Node 20, y la CLI de wails). Ejecuta `xvfb-run -a wails dev -browser=false -devserver 0.0.0.0:34115`; interfaz de la app en el navegador en http://localhost:34115, Vite en :5173.
- Comprobaciones específicas de Go sin iniciar la app: `docker compose run --rm app go build ./...` / `go vet ./...` / `go test -tags integration ./internal/scraper/ -v` (el test usa el sitio real — requiere red).
- Solo frontend, desde `frontend/`: `npm install` (requerido primero), `npm run dev`, `npm run build`, `npm run check` (svelte-check; la única comprobación de tipos). Todos fueron verificados y pasan correctamente.
- Sin linters, sin CI. Verificación = `gofmt -l .` (sin salida) + `go vet ./...` + `go test -race ./...` + `npm run check` (este último solo en tareas de frontend).

## Consideraciones importantes (Gotchas)

- `wails dev` enlaza su servidor de desarrollo a `127.0.0.1` por defecto → inaccesible a través del mapeo de puertos de Docker (el `curl` del host obtiene conexión rechazada mientras el contenedor parece estar bien). La bandera `-devserver 0.0.0.0:34115` en `docker-compose.yml` es requerida, no cosmética; la bandera necesita el formato `host:puerto`.
- El desarrollo en el navegador debe usar **`localhost:34115`** (servidor de desarrollo de Wails), no `:5173`. El servidor propio de Vite no tiene el puente `window.go` — las llamadas fallan con `Cannot read properties of undefined (reading 'main')`. `App.svelte` ahora protege esto y muestra un mensaje apuntando a :34115.
- `frontend/wailsjs/**` es generado automáticamente (NO EDITAR). Solo se regenera cuando se ejecuta `wails dev`/`wails build` al compilar código Go. Si `App.svelte` importa una función que falta ahí, `npm run build`/`check` fallarán hasta que los bindings se regeneren.
- `main.go` incrusta `frontend/dist` (`//go:embed all:frontend/dist`): `go build` necesita que `frontend/dist` exista (está en el .gitignore; prodúcelo con `npm run build`).
- La configuración de Vite está fijada: `0.0.0.0:5173`, `strictPort`, y observación por polling (polling watch) — esto es deliberado, para la observación de archivos entre Docker y Windows. No lo elimines.
- Wails serializa a JSON con `encoding/json` de Go: los campos de los modelos necesitan etiquetas `json:` en minúsculas o el frontend no verá nada (lee `anime.cover_url`). Las etiquetas también determinan la generación de los modelos de TS.
- `frontend/index.html` tiene `<meta name="referrer" content="no-referrer">`: animeflv.or.at devuelve **403 para las imágenes de portada** cuando se envía cualquier cabecera Referer (protección contra hotlinking). No elimines esta etiqueta meta.

## Scraper

- `internal/scraper/animeflv.go` apunta a **`https://animeflv.or.at`** — un sitio de WordPress, no al clásico animeflv.net (ese dominio está muerto/movido; no "arregles" la URL devolviéndola al original).
  - URL de búsqueda: `/?s=<query>`; resultados en `.search-series-card` (`a.thumbnail-link`, `h2.entry-title`, `img.anime-image`).
  - Los selectores CSS antiguos (`article.Anime`, `h3.Title`, `/browse?q=`) NO existen aquí.
  - La página principal del sitio lista los episodios como `h2.Title`; las páginas de las series se encuentran en `/anime/<slug>/`; cada serie es también una categoría de WP.
  - Envía un User-Agent de navegador (el predeterminado de Go puede ser bloqueado) y falla si el código HTTP no es 200.
- Prueba de humo (Smoke test): `internal/scraper/animeflv_test.go` busca "bleach" (se sabe que existe); una caída del sitio o un cambio de dominio hará que falle — ese es el objetivo de la prueba.

## Configuración

- `wails.json` maneja los hooks del frontend: install/build/dev = `npm install` / `npm run build` / `npm run dev`, `serverUrl: auto`.
- `.gitignore` excluye `build/bin`, `node_modules`, `frontend/dist`.

## Plan de trabajo por tareas

- Reglas y layout de paquetes: `tareas/00_REGLAS.md`. Estado del trabajo: `PROGRESO.md`. Detalle de cada tarea: `tareas/Txx.md`.
- Trabaja SOLO la tarea indicada y actualiza `PROGRESO.md` al terminar.
- Go, wails y gofmt no están en el PATH: ejecuta todo con `docker compose run --rm app <comando>` (por ejemplo `go vet ./...`, `go test -race ./...`, `gofmt -l .`, `wails generate module`). Donde un archivo de tarea diga `go ...` o `wails ...`, aplica esto. `npm` se ejecuta en `frontend/`.
- Los tests que usan la red real llevan el build tag `integration` (`go test -tags integration ./...`). `go test ./...` normal debe pasar sin red.
- `internal/scraper/animeflv.go` (paquete `scraper`) y su test siguen funcionando hasta T18, que los mueve a `internal/scraper/animeflv/`. No los borres antes y conserva los selectores de animeflv.or.at descritos arriba.
- `app.go` ya expone `SearchAnime(query string) ([]models.Anime, error)`, que usa `App.svelte`. En T20 consérvalo (puede delegar en el nuevo registro de providers) o anota en `PROGRESO.md` el cambio de nombre para que T25 actualice `App.svelte`.
- Los modelos nuevos usan tags `json` en minúsculas y snake_case (como `cover_url`).
- Mantén el `<meta name="referrer" content="no-referrer">`: el Referer que exigen los servidores de vídeo se inyecta en el proxy local (T19), no en el navegador.
- El host es Windows 11 con PowerShell: no uses `grep`, `head` ni `sed` en el host. Usa `curl.exe` (no `curl`), `Select-String` y `Select-Object -First N`, o ejecuta dentro del contenedor con `docker compose run --rm app sh -c "..."`.
