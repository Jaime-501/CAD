# Guía de desarrollo

Aplicación de escritorio **Wails v2** (backend en Go, frontend en **Svelte 5 + TypeScript + Vite**) que hace scraping de **JKAnime** y **AnimeAV1** (ambos proveedores completos: búsqueda, detalle, episodios y fuentes de vídeo) y reproduce HLS/MP4. Los textos de la interfaz, los comentarios del código y esta documentación están en español.

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
app.go                   capa Wails delgada: métodos expuestos + ciclo de vida + dirHistorial (T37)
internal/app             núcleo (package app): orquestación testeable con dobles (registro, proxy, caché, historial, favoritos, Home)
  nucleo.go              el núcleo; sin dependencia de Wails (reglas P1–P3)
  retomar.go             "Retomar" y "Siguiente ▸" (lógica de la tabla de acciones, T46)
  portabilidad_test.go   guardia P2/P3 con go/parser: internal/** sin wailsapp ni os.UserConfigDir
frontend/                Svelte 5 + TS + Vite; wailsjs/ es código GENERADO (no editar)
internal/logx            logging estructurado con redacción de secretos
internal/httpx           cliente HTTP con cookiejar, errores tipados y reintentos
internal/models          modelos de dominio (sin dependencias del proyecto)
internal/scraper         interfaz Provider, registro y errores comunes
internal/scraper/jkanime            JKAnime: búsqueda, detalle, episodios AJAX, página de episodio, Home
internal/scraper/jkanime/extractors extractores de vídeo (jkplayer, jk.php, var servers, MediaFire)
internal/scraper/animeav1           AnimeAV1 (SvelteKit): __data.json + devalue, lotes de 50, HLS derivado, Home
internal/stream          proxy local de streaming (127.0.0.1) + reescritura de manifiestos
internal/cache           caché de detalle/episodios con TTL
internal/db              historial de visualización y favoritos (JSON en disco)
docs/firmas/<pkg>.md     firmas reales de cada paquete (si chocan con el código, manda el código)
tareas/                  plan de trabajo por tareas (reglas en 00_REGLAS.md, detalle en Txx.md)
PROGRESO.md              estado del plan, deudas y pendientes
docs/BITACORA.md         desviaciones históricas y pendientes cerrados
AGENTS.md                guía para agentes de código
```

Regla de dependencias: `models` no importa nada del proyecto; los extractores no importan `jkanime`; los providers implementan `scraper.Provider` y se registran en `scraper/registry.go`.

## Los dos proveedores

- **JKAnime** (`internal/scraper/jkanime/`): scraping con goquery sobre HTML crudo; episodios por AJAX con cookie de sesión + CSRF; extractores de vídeo (jkplayer, jk.php, var servers, MediaFire).
- **AnimeAV1** (`internal/scraper/animeav1/`): sitio **SvelteKit SSR** sin cookies ni CSRF. La vía principal es **`GET <ruta>/__data.json`**, JSON en formato *devalue* (referencias por índice; se hidrata con `PageData` + `Unflatten`), no el `<script>` de hidratación. Búsqueda en `/catalogo/__data.json?search=…`, detalle en `/media/<slug>/__data.json` (con todos los episodios embebidos) y fuentes en `/media/<slug>/<n>/__data.json` (campo `embeds`, con idiomas `SUB`/`DUB`). Los episodios se reparten en **lotes de 50** (`tamLote`).
- La búsqueda combina ambos **en paralelo** (JKAnime primero, timeout de 20 s por consulta) y tolera que uno esté caído.
- **`Sec-Fetch-Site: same-origin`**: los segmentos HLS de `player.zilla-networks.com` responden **403 sin esa cabecera**. Las fuentes HLS de AnimeAV1 la declaran en `models.VideoSource.Headers` y `stream/reenvio.go` la inyecta desde la sesión del proxy local — nunca viaja en la URL ni cruza el puente de Wails hacia el frontend. El manifiesto se deriva sin petición (`/play/<id>` → `/m3u8/<id>`); `/play/*` jamás se pide.

## Tests y tags

- **Sin red (por defecto):** `go test -race -count=1 ./...`. Todo corre sin internet (`httptest.Server` y fixtures en `testdata/`).
- **Con red (`integration`):** los smoke tests contra los sitios reales llevan `//go:build integration` y viven en archivos `*_integration_test.go`:

  ```sh
  docker compose run --rm app go test -tags integration -count=1 ./...
  ```

  Incluyen la búsqueda real de JKAnime y AnimeAV1 y los **E2E completos** (`internal/app/app_e2e_integration_test.go`: `TestE2EFlujoCompletoRedReal` de JKAnime y `TestE2EAnimeAV1RedReal` de AnimeAV1: búsqueda → detalle con streaming → fuentes → manifiesto y un segmento con `Range` por el proxy → historial). Si un dominio cambia o el sitio cae, fallan **a propósito**.
- Los E2E **offline** (`TestE2EFlujoCompletoOffline` y `TestE2EAnimeAV1FlujoCompletoOffline`, en `internal/app/app_e2e_test.go`) recorren el mismo flujo contra un sitio falso montado con los fixtures reales, exigen que las peticiones sean exactamente las esperadas y que nada salga a la red. El de AnimeAV1 demuestra además que el proxy inyecta `Sec-Fetch-Site: same-origin` (segmento por el proxy → 206; directo sin cabecera → 403).
- **Guardia de portabilidad** (`internal/app/portabilidad_test.go`, sin red): recorre `internal/**` con `go/parser` y falla si el núcleo importa `github.com/wailsapp/*` (P2) o llama a `os.UserConfigDir`/`os.UserHomeDir` (P3; esas rutas las resuelve `app.go`).
- **Fixtures:** son la respuesta cruda del servidor, sanitizada (`TOKEN_DE_PRUEBA`, `E_PRUEBA`…), sin cookies ni tokens reales. Cápturalos con `curl.exe` y revísalos antes de subirlos.

## Regenerar los bindings de Wails

`frontend/wailsjs/` es **generado**: nunca se edita a mano.

```sh
docker compose run --rm app wails generate module
```

`wails dev` y `wails build` también lo regeneran. Los métodos de `app.go` deben devolver `(valor, error)` como máximo (Wails ignora retornos extra). Si el frontend importa algo que no existe, `npm run build` y `npm run check` fallan hasta regenerar. Los payloads de eventos (`episodes:update`, `episodes:complete`) no se generan: se definen a mano en `frontend/src/api.ts`.

Los bindings se regeneraron por última vez en **T47** (v1.2): `App.d.ts`/`App.js` añaden los 11 métodos nuevos (`GetHomeSection`, `AddFavorite`, `RemoveFavorite`, `ListFavorites`, `IsFavorite`, `GetRecentProgress`, `DeleteProgressAnime`, `DeleteProviderProgress`, `ClearProgress`, `ResumeAnime`, `GetNextEpisode`) y `models.ts` las clases `Favorite`, `HomeSection`, `ResumeInfo`, `NextEpisode` más `WatchProgress` con sus 5 campos nuevos. Las envolturas tipadas que usa el frontend viven en `frontend/src/api.ts` (único archivo con imports de `wailsjs`, regla P4).

## Compilar el .exe de Windows desde Docker

```sh
# Release: la app NO abre consola
docker compose run --rm app wails build -platform windows/amd64

# Depuración: abre una consola con los logs y habilita las herramientas de desarrollo (F12)
docker compose run --rm app wails build -platform windows/amd64 -debug
```

El ejecutable queda en `build/bin/miapp.exe` (ignorado por git). La compilación regenera los bindings y recompila el frontend. La variable de entorno `MIAPP_DEV` activa logs en nivel `Debug` con formato legible (solo cambia nivel y formato; la redacción de secretos se aplica siempre).

## Datos que guarda la app

- **Historial:** `%AppData%\miapp\historial.json` (`os.UserConfigDir()/miapp`, calculado en `app.go` por la regla P3: el núcleo solo recibe la ruta). Un registro por episodio (`anime_id`, `episode_id`, `current_time`, `total_duration`, `is_watched`; umbral 0.90) más, desde la v1.2, `provider`, `anime_title`, `cover_url`, `episode_number` y `updated_at` (Unix s; **0 = registro antiguo**, un `historial.json` de la v1.1 sigue cargando). Escritura atómica; si el archivo se corrompe se renombra a `historial.json.bak` y se empieza vacío. El listado de la barra lateral es `LastPerAnime` (un registro por proveedor+slug, ordenado por `updated_at`).
- **Favoritos:** `%AppData%\miapp\favoritos.json` — mismo directorio que el historial, array JSON **indentado** de `models.Favorite` (`provider`, `anime_id`, `title`, `cover_url`, `added_at`), escritura atómica y el mismo `.bak` ante un archivo corrupto (`internal/db/favoritos.go`, T42).
- **Carpeta `miapp.exe`:** la crea WebView2 para su perfil (caché, cookies del webview). No es de la app.
- **Caché de detalle/episodios y de las secciones del Home:** solo en memoria, con TTL de 10 min (`home:<proveedor>` en `internal/app/nucleo.go`); no se persiste.
- En Docker, estos archivos se escriben dentro del contenedor, no en `%AppData%`.

## Limitaciones conocidas (técnicas)

1. **Algunos servidores son iframes sin resolver:** los embeds de AnimeAV1 que no son el reproductor propio (UPNShare, Voe, Byse, MP4Upload) y los iframes de JKAnime se publican tal cual: se ven, pero no guardan progreso.
2. **Proxy no alcanzable desde el navegador del host en Docker** (ver arriba). A propósito: es loopback y con anti-SSRF.
3. **El botón "Continuar" de la vista de detalle** sigue abriendo el episodio de mayor número con progreso, no el más reciente; la barra lateral del historial sí ordena por recencia (campo `updated_at`, v1.2).

Dos limitaciones de la v1.1 quedaron **resueltas en la v1.2** y ya no figuran aquí: *«"Continuar" sin marca de tiempo»* (`WatchProgress` guarda `updated_at` desde T40/T41) y *«no hay borrado de registros del historial»* (T51–T52: deslizar, ✕ con Deshacer, borrado por proveedor y de todo el historial).

## Seguridad

- El proxy de streaming solo escucha en `127.0.0.1`, bloquea destinos privados/loopback/link-local por **IP resuelta** (anti-SSRF) y mantiene las cabeceras requeridas (`Referer`, etc.) en la sesión, **nunca** en la URL.
- Los logs pasan por `internal/logx`: cookies, CSRF, `Authorization`/`Set-Cookie` y **URLs firmadas** (`?e=`, `?st=`, …) salen redactados.
- `frontend/index.html` declara `<meta name="referrer" content="no-referrer">`: algunos orígenes devuelven 403 en las portadas si se envía `Referer` (se mantuvo al retirar AnimeFLV; el CDN de AnimeAV1 no lo exige). No la quites.

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

El proyecto se construyó con un plan de tareas (`tareas/`) ejecutadas por un agente de código: T01–T29 en la v1.0, T30–T36 (AnimeAV1, v1.1.0; T30–T35 en la rama `feature/animeav1`, ya fusionada en `main` — T36 fue el cierre documental), T37 (preparación para Android: núcleo en `internal/app`, reglas P1–P7) y T39–T56 (v1.2.0: Home por proveedor, historial lateral con retomar, favoritos y el botón "Siguiente ▸", rama `feature/home-historial-favoritos`). `AGENTS.md` resume las reglas para agentes, `PROGRESO.md` es la memoria entre sesiones (los estados históricos, en `docs/PROGRESO_v1.0.md`, `docs/PROGRESO_v1.1.md` y `docs/PROGRESO_v1.2.md`) y `docs/firmas/` contiene las firmas reales de cada paquete.
