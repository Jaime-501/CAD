# AGENTS.md

Wails v2 desktop app (Go module and app name: `miapp`): Go backend + Svelte 5 / TypeScript / Vite frontend in `frontend/`. Scrapes animeflv. Comments and UI strings are in Spanish — match that.

## Commands

- `docker compose up` — main dev flow (Go/wails are **not on PATH** on this machine; the image has Go 1.25, Node 20, wails CLI). Runs `xvfb-run -a wails dev -browser=false -devserver 0.0.0.0:34115`; app UI in browser at http://localhost:34115, Vite at :5173.
- Focused Go checks without starting the app: `docker compose run --rm app go build ./...` / `go vet ./...` / `go test -tags integration ./internal/scraper/ -v` (el test usa el sitio real — requiere red).
- Frontend only, from `frontend/`: `npm install` (required first), `npm run dev`, `npm run build`, `npm run check` (svelte-check; the only typecheck). All were verified green.
- No linters, no CI. Verificación = `gofmt -l .` (sin salida) + `go vet ./...` + `go test -race ./...` + `npm run check` (este último solo en tareas de frontend).

## Gotchas

- `wails dev` binds its dev server to `127.0.0.1` by default → unreachable through Docker port mapping (host curl gets connection refused while the container looks fine). The `-devserver 0.0.0.0:34115` flag in `docker-compose.yml` is required, not cosmetic; the flag needs `host:port` form.
- In-browser dev must use **`localhost:34115`** (Wails dev server), not `:5173`. Vite's own server has no `window.go` bridge — calls fail with `Cannot read properties of undefined (reading 'main')`. `App.svelte` now guards this and shows a message pointing at :34115.
- `frontend/wailsjs/**` is generated (DO NOT EDIT). It only regenerates when `wails dev`/`wails build` runs with compiling Go code. If `App.svelte` imports a function missing there, `npm run build`/`check` fail until bindings regenerate.
- `main.go` embeds `frontend/dist` (`//go:embed all:frontend/dist`): `go build` needs `frontend/dist` to exist (gitignored; produce it with `npm run build`).
- Vite is pinned: `0.0.0.0:5173`, `strictPort`, polling watch — deliberate, for Docker/Windows file watching. Don't remove.
- Wails JSON-serializes with Go's `encoding/json`: model fields need lowercase `json:` tags or the frontend sees nothing (it reads `anime.cover_url`). Tags also drive the generated TS models.
- `frontend/index.html` has `<meta name="referrer" content="no-referrer">`: animeflv.or.at returns **403 for cover images** when any Referer header is sent (hotlink protection). Don't remove the meta.

## Scraper

- `internal/scraper/animeflv.go` targets **`https://animeflv.or.at`** — a WordPress site, not the classic animeflv.net (that domain is dead/moved; don't "fix" the URL back).
  - Search URL: `/?s=<query>`; results in `.search-series-card` (`a.thumbnail-link`, `h2.entry-title`, `img.anime-image`).
  - Old CSS selectors (`article.Anime`, `h3.Title`, `/browse?q=`) do NOT exist here.
  - Site homepage lists episodes as `h2.Title`; series pages live at `/anime/<slug>/`; each series is also a WP category.
  - Sends a browser User-Agent (Go's default may be blocked) and fails on non-200.
- Smoke test: `internal/scraper/animeflv_test.go` queries "bleach" (known to exist); a site outage or domain change makes it fail — that's the point.

## Config

- `wails.json` drives frontend hooks: install/build/dev = `npm install` / `npm run build` / `npm run dev`, `serverUrl: auto`.
- `.gitignore` excludes `build/bin`, `node_modules`, `frontend/dist`.

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
