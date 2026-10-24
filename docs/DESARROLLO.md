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

> **Fantasma CRLF:** no hay `.gitattributes` y `core.autocrlf=true`, así que un merge/checkout hecho desde Windows puede dejar los `.go` con fin de línea CRLF en el worktree; `gofmt -l .` (Linux) los marca **aunque el contenido esté bien formateado**. Normalízalos a LF (quitar solo el `\r` final; git no registra el cambio) — decisión del humano en T36. `git status` puede seguir mostrando esos archivos "M" con diff vacío.

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
frontend/                Svelte 5 + TS + Vite; wailsjs/ es código GENERADO (no editar)
  src/
    api.ts                # ÚNICO punto de acceso al backend (P4)
    stores.ts  main.ts  views.ts  ui.ts  style.css
    favoritos.ts  historial.ts  home.ts  retomar.ts  toasts.ts
    etiquetas.ts  deslizar.ts  ventana.ts  revelar.ts  vite-env.d.ts (+ *.test.ts)
    App.svelte  Vista*.svelte  Tarjeta*.svelte  Boton*.svelte
    BarraLateral.svelte  FilaAnimes.svelte  GridFavoritos.svelte
    SelectorFuente.svelte  Toast.svelte
    # nuevos en la serie v1.4.0 (T69–T89):
    BarraInferior.svelte  CabeceraVista.svelte  BarraAccion.svelte
    FilaContinuar.svelte  TarjetaContinuar.svelte
    Icono.svelte  Btn.svelte  Chip.svelte  Insignia.svelte
    Esqueleto.svelte  EstadoVista.svelte
    assets/               # logo (las fuentes locales vienen de npm)
  wailsjs/                # bindings generados por Wails — NO EDITAR
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

## Verificación visual de la UI (v1.4.0)

Desde la serie **T69–T89** hay dos herramientas **de solo lectura** (no tocan código de producto) que
miden la interfaz en el navegador real. Necesitan la app levantada (`docker compose up`) y se ejecutan
contra el **dev server de Wails** (`http://localhost:34115`, con red real):

```sh
NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" \
  node docs/investigacion/ui/verificar_ui.mjs <modo> [dirSalida]
NODE_PATH="$(npm root -g)/@playwright/mcp/node_modules" \
  node docs/investigacion/ui/a11y_auditoria.mjs <etiqueta> [dirSalida]
```

| Modo de `verificar_ui.mjs` | Qué comprueba |
|---|---|
| `capturar` | Tokens por `getComputedStyle` + `overflowH` de cada vista a 360/768/1280 |
| `comparar` | Diferencias por canal (> `tol`) entre dos carpetas de capturas |
| `pixel` | Color (hex + rgba) de un punto de una captura |
| `tactiles` | Objetivos táctiles de las primitivas (`.btn` ≥ 44×44, `.chip` ≥ 40) |
| `shell` | Barra inferior móvil, destinos, cajón, geometría y scroll T66/T67 |
| `cabecera` | Cabecera compacta de detalle y reproductor (táctiles, sticky, título largo) |
| `buscador` | Buscador y pestañas del Home en modo estrecho (lupa, ✕, foco) |
| `filas` | Carruseles y tarjeta del Home (sangrado, flechas, puntero grueso) |
| `continuar` | Fila «Continuar viendo» (tarjetas, progreso, sembrado del historial) |
| `busqueda` | Rejilla de resultados y sus estados (cargando, error, vacío) |
| `ficha` | Ficha de detalle (portada, chips, géneros, sinopsis plegable) |
| `cta` | Barra de acción fija «Continuar / Reiniciar» |
| `episodios` | Tarjeta de episodio y rejilla (columnas, alto constante, estados) |
| `orden` | Orden ascendente/descendente de episodios |
| `reproductor` | Layout del reproductor y selector de fuentes |
| `siguiente` | Chip «Siguiente ▸» (tamaño, solape, discreto, teclado) |
| `favoritos` | Cuadrícula de favoritos, toast y deslizar en el cajón |
| `estados` | Estados de carga/error/vacío de Home, detalle, reproductor y cajón |
| `rendimiento` | Frames, scripting/tarea/layout, nodos, reduced-motion y reduced-transparency |

Cada modo imprime una tabla `| comprobación | ok | detalle |` y su recuento
(`Comprobaciones: X/Y OK`). `a11y_auditoria.mjs` recorre 11 combinaciones vista×ancho y resume
**contraste** (normal, con foco y en hover), **nombres accesibles**, **jerarquía de encabezados**,
**ARIA**, **anillo de foco**, **foco del cajón**, **estados de carga** y **deriva de scroll**.

> Los modos que siembran datos (historial, favoritos) los **restauran** al terminar. Las capturas
> van a `docs/img/v1.4/<modo>/` (ignorado por git).

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

El proyecto se construyó con un plan de tareas (`tareas/`) ejecutadas por un agente de código: T01–T29 en la v1.0, T30–T36 (AnimeAV1, v1.1.0; T30–T35 en la rama `feature/animeav1`, ya fusionada en `main` — T36 fue el cierre documental), T37 (preparación para Android: núcleo en `internal/app`, reglas P1–P7), T38 (su cierre documental) y T39–T57 (v1.2.0 **publicada**: Home por proveedor con su pulido visual —T53b—, historial lateral con retomar, favoritos y el botón "Siguiente ▸" con su estado discreto —T55 y T57—, rama `feature/home-historial-favoritos`), cerrada con T56 (título CAD, versionado y docs) y el **cierre documental** (etiquetado T58, luego **renumerado T59**) tras la prueba manual con el `.exe` y la release, que hizo el humano; **T58** (dos bugs visuales de la v1.2.0: carátula del historial al retomar y contador de favoritos al arrancar) y **T59** (cierre documental de la **v1.2.1 publicada**, `productVersion` 1.2.1, tag `v1.2.1`); **T64–T68** (v1.3.0 **preparada**, rama `feature/episodios-incrementales`: lógica pura de la ventana de episodios —`frontend/src/ventana.ts` + `ventana.test.ts`—, revelado animado de las tarjetas de episodio al entrar en el viewport —`frontend/src/revelar.ts`; sin paginar—, clave de scroll del detalle por proveedor+slug —`frontend/src/views.ts` + `views.test.ts`—, restauración del scroll del detalle al recargar la lista y cierre documental; `productVersion` 1.3.0, sin tag ni release); y **T69–T89** (v1.4.0 **preparada**, rama `feature/rediseno-movil`: rediseño y enfoque móvil —tokens y primitivas, barra inferior y cajón, cabecera, buscador, filas, «Continuar viendo», rejilla de resultados, ficha, CTA fija, tarjeta de episodio, orden, reproductor, chip «Siguiente», favoritos y toasts, estados, movimiento y rendimiento, accesibilidad AA y el cierre de **T89**; `productVersion` 1.4.0, sin tag ni release).
`AGENTS.md` resume las reglas para agentes, `PROGRESO.md` es la memoria entre sesiones (los estados históricos, en `docs/PROGRESO_v1.0.md`, `docs/PROGRESO_v1.1.md`, `docs/PROGRESO_v1.2.md`, `docs/PROGRESO_v1.3.md` y `docs/PROGRESO_v1.4.md`) y `docs/firmas/` contiene las firmas reales de cada paquete.
