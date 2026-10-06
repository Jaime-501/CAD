# PROGRESO — Scraper JKAnime + reproductor HLS/MP4

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**

## Protocolo de sesión

1. Leer `AGENTS.md`, `tareas/00_REGLAS.md`, este archivo, `docs/firmas/<paquetes de los que depende la tarea>` y el archivo de la tarea (`tareas/Txx.md`).
2. Ejecutar **solo** la tarea indicada. Respetar "Puedes tocar" / "NO debes tocar".
3. Correr la comprobación de la tarea. Después, el humano la repite por su cuenta (`gofmt -l .`, `go vet ./...`, `go test -race ./...` y `npm run check` en frontend): no confiar solo en el reporte del agente.
4. Actualizar este archivo: marcar la tarea, anotar desviaciones y pendientes; las firmas nuevas van en `docs/firmas/<paquete>.md` (formato `### Txx — <paquete>`, ej. `docs/firmas/httpx.md`).
5. Detenerse. No empezar la siguiente tarea.

## Prompt base por sesión

```text
Lee AGENTS.md, tareas/00_REGLAS.md, PROGRESO.md, tareas/T07.md y
docs/firmas/<paquetes de los que depende T07 — mapa abajo>.
Ejecuta SOLO la tarea T07. Respeta "Puedes tocar" y "NO debes tocar".
Si necesitas tocar un archivo fuera de la lista, detente y repórtalo.
Al terminar: corre la comprobación, actualiza PROGRESO.md (estado, desviaciones,
pendientes) y docs/firmas/<paquete>.md si creaste firmas, y detente.
No empieces la siguiente tarea.
```

| Tarea | Lee docs/firmas/ de |
|:--------:|:--------:|
| T14, T15 |	extractors, models |
| T16 |	extractors, models |
| T17 |	extractors, jkanime, scraper, models |
| T18 |	scraper, httpx, models |
| T19 |	httpx, models |
| T20 |	scraper, jkanime, extractors, stream, models |
| T21 |	scraper, models |
| T22 |	models |
| T23 a T27 |	no hay firmas Go; el código generado de frontend/wailsjs/ hace de referencia |

## Prerrequisitos: fixtures reales (humano o Playwright MCP)

Antes de las tareas que parsean HTML/JSON real debe existir un **fixture real y sanitizado** (sin cookies ni tokens) en el `testdata/` correspondiente. Un fixture es la respuesta **cruda** del servidor guardada en un archivo, para que los tests corran sin red. Lo puede capturar el humano a mano o el agente con el MCP de Playwright (ver "Captura con Playwright MCP" más abajo). Si no existe el fixture de la tarea, el agente se detiene y lo pide: **no inventa HTML ni selectores**.

- [x] Página de búsqueda de JKAnime → T07
- [x] Página de detalle de un anime → T08
- [x] Respuesta JSON de `/ajax/episodes/<id>/<página>` (al menos 2 páginas) → T09 — 3 páginas de bleach (`episodes_p1/p2/last.json`): 16 + 16 + **14** episodios (1–16, 17–32, 353–366); la última se sabe por el campo **`last_page`** de la respuesta (23; `total`=366 y `per_page`=16)
- [x] Página de un episodio → T10
- [x] Respuestas del player `jkplayer/um?...` → T12 — `jkplayer_um.html` (DPlayer, URL cifrada con `atob`, 16932 B) y `jkplayer_umv.html` (video.js, el player que la página carga por defecto, 3906 B). **El sitio no publica `.vtt`**: se muestrearon ~130 episodios (homepage, clásicos, películas, especiales y una serie en chino) en páginas de episodio, respuestas `um`/`umv`, playlists m3u8 y peticiones reales del navegador → 0 pistas; el test de subtítulos usa HTML artificial (autorizado en sesión: "una variante con .vtt si existe").
- [x] Respuesta de `jk.php` → T13 — `jkphp.html` (de `https://jkplayers.com/jk.php?u=…`, 200, 6582 B) y `jkplayer_jk.html` (de `jkanime.net/jkplayer/jk?u=…`, la referencia que hoy emiten las páginas, 6528 B). `jkanime.net/jk.php` responde 404 (nginx).
- [x] Página con `var servers` → T14 — `extractors/testdata/servers.html` (recorte **verbatim** de `jkanime/testdata/episode_page.html`, bleach/1: `<head>` completo + bloque `.bg-servers` + los dos `<script>` que declaran/consumen `var servers`; 24166 B). Verificado con `grep` sobre el HTML crudo: `var servers` está en la respuesta del servidor (línea 1089 del fixture de T10), no en el DOM renderizado. 10 servidores (Mediafire, Mega, VOE, Vidhide, Filemoon, Mixdrop, Mp4upload, Streamtape, Streamwish, Doodstream). Sanitizado: `csrf-token` ya era `TOKEN_DE_PRUEBA` (T10) y los parámetros `e=` cifrados de los iframes `jkplayer/um|umv` → `E_PRUEBA` (regla 12).
- [x] Página de MediaFire → T15 — `extractors/testdata/mediafire.html` (recorte **verbatim** de la respuesta cruda de `https://www.mediafire.com/file/aaxudss27tqq7c8/` con `curl.exe`: 200, 330502 B → 114644 B, `<head>` + bloque `.dl-btn-cont` con el `#downloadButton`). El remote de Mediafire del fixture de T14 apunta a esta página. Verificado en el HTML crudo: el enlace de descarga (`id="downloadButton"`, `aria-label="Download file"`) está **en la respuesta del servidor** (con Playwright no se registró ninguna XHR que lo entregue: está en el HTML, no tras ejecutar JS). Sanitizado: la firma del host (`download1529.mediafire.com/<firma>/`) → `FIRMA_DE_PRUEBA`, el token del `input name="security"` → `SEGURIDAD_DE_PRUEBA` y `dkey=` → `DKEY_DE_PRUEBA`; sin cookies (las `Set-Cookie` no se guardan). Comprobado además con `curl.exe` que `mediafire.com/file/…` responde 302 a `www.mediafire.com` y que el enlace de descarga responde 206 (probe de 1 byte) con y sin Referer.
- [x] Búsqueda de AnimeFLV → T18 — `internal/scraper/animeflv/testdata/search.html` (`GET /?s=bleach` con `curl.exe` y UA de navegador: 200, 60253 B, **1** tarjeta: `bleach-sennen-kessen-hen-kashin-tan`) y `search_empty.html` (`GET /?s=shingeki`: 200, 60097 B, 0 tarjetas, bloque `.no-results`). Verificado con `grep` en el crudo: `.search-series-card`, `a.thumbnail-link`, `h2.entry-title`, `img.anime-image` están en la respuesta del servidor. **Sin challenge de Cloudflare** (sin "Just a moment" ni `cf-mitigated`). Sanitizado: `commentsNonce` → `NONCE_DE_PRUEBA` (regla 12), sin cookies. Ojo: el buscador del sitio devuelve muy pocas tarjetas (`bleach`→1, `one piece`→1, `naruto`/`shingeki`/`dragon ball`/`detective conan`→0).

Nombres sugeridos de archivo (dentro del `testdata/` de cada paquete): `search.html`, `detail.html`, `episodes_p1.json`, `episodes_p2.json`, `episode_page.html`, `jkplayer_um.html` (y una variante con `.vtt`), `jkphp.html`, `servers.html`, `mediafire.html`.

### Captura con Playwright MCP

**Regla de oro:** el fixture debe ser lo que recibirá goquery, es decir, el HTML **tal como lo envía el servidor**, no el DOM ya modificado por JavaScript. goquery no ejecuta JS: un dato que solo aparece tras cargar scripts no existe para el scraper en Go.

Ojo con las herramientas de Playwright: `browser_snapshot` devuelve el árbol de accesibilidad, **no** HTML, y no sirve como fixture. Los nombres exactos de las herramientas dependen de la versión de tu servidor MCP; usa las equivalentes de navegar, evaluar JS y listar peticiones de red.

Procedimiento, en orden:

1. **Explorar:** navegar a la URL con Playwright y, con la lista de peticiones de red, identificar la petición exacta (URL, método, cabeceras y cuerpo necesarios, sin copiar valores secretos).
2. **Obtener la respuesta cruda**, por orden de preferencia:
   - Con `curl` o un programa Go corto desde Bash, guardando directo a archivo (`-o testdata/archivo`). Es lo preferible porque no pasa por el contexto y reproduce lo que hará el scraper.
   - Si el sitio bloquea clientes HTTP (Cloudflare) pero el navegador pasa: dentro de la página, `fetch(url).then(r => r.text())` mediante evaluación de JS. Devuelve la respuesta del servidor sin modificar por scripts.
3. **Comprobar que no es un bloqueo:** si el archivo contiene "Just a moment" o similar, no es un fixture válido. Anotarlo en "Pendientes activos" (abajo), porque afecta al diseño de T02.
4. **Guardar en archivo, no pegar en el chat.** Inspeccionar con `grep` o fragmentos de pocas líneas, nunca leyendo la página completa.
5. **Recortar solo si es enorme:** conservar el `<head>` con lo necesario (por ejemplo la meta del CSRF para T08) y el bloque relevante con 2 o 3 elementos de ejemplo. No alterar la estructura ni los nombres de clases.
6. **Sanitizar:** reemplazar cookies, tokens CSRF reales y cualquier dato de sesión por valores falsos como `TOKEN_DE_PRUEBA`.
7. **Verificar que el dato está en el archivo crudo** (`grep`). Si solo aparece en el DOM renderizado, goquery no puede extraerlo: detenerse, reportarlo y buscar el endpoint AJAX que entrega ese dato.
8. **Validar con goquery:** el test de la tarea carga el fixture y comprueba los selectores. Ese test es la prueba de que la captura sirve.

Para cuidar el contexto (modelos gratuitos):

- Activar el MCP de Playwright **solo** en las sesiones de T07–T10 y T12–T15, y desactivarlo en el resto.
- No pedir capturas ni lecturas de página completa en el chat; solo resultados acotados.
- Si hay un montón de peticiones de red, filtrar por la URL que interesa en lugar de listarlas todas.

## Estado de tareas

Leyenda: `[ ]` pendiente · `[~]` en curso · `[x]` hecha y verificada

| Estado | Tarea | Título | Depende de | Sesión sugerida |
|---|---|---|---|---|
| [x] | T01 | Logging y observabilidad base | — | Sola |
| [x] | T02 | Infraestructura HTTP, errores y reintentos | T01 | Sola (pesada) |
| [x] | T03 | Modelos de dominio | — | Con T05 |
| [x] | T04 | Interfaz Provider y registro | T03 | Sola |
| [x] | T05 | Referencia de episodio y películas | T03 | Con T03 |
| [x] | T06 | JKAnime — excepciones | T02 | Sola |
| [x] | T07 | JKAnime — búsqueda | T02–T06 | Sola |
| [x] | T08 | JKAnime — detalle y metadatos | T07 | Sola |
| [x] | T09 | JKAnime — episodios AJAX con streaming | T08 | Sola (pesada) |
| [x] | T10 | JKAnime — fetch de página de episodio | T02 | Sola |
| [x] | T11 | Arquitectura de extractores | T03, T10 | Sola |
| [x] | T12 | Extractor JKPlayer (softsubs) | T11 | Agrupable T12–T15 |
| [x] | T13 | Extractor JKPHP | T11 | Agrupable T12–T15 |
| [x] | T14 | Extractor `var servers` | T11 | Agrupable T12–T15 |
| [x] | T15 | Resolver MediaFire | T11 | Agrupable T12–T15 |
| [x] | T16 | Normalizador de VideoSource | T12–T15 | Con T17 |
| [x] | T17 | Selector de mejor fuente + GetVideoSources | T16 | Con T16 |
| [x] | T18 | AnimeFLV — búsqueda con fallback | T02, T04 | Sola |
| [x] | T19 | Proxy local de streaming | T02, T03 | Sola (pesada) |
| [ ] | T20 | Integración app.go y eventos Wails | T09, T17, T19 | Sola (pesada) |
| [ ] | T21 | Caché local | T20 | Sola (pesada) |
| [ ] | T22 | Persistencia del historial | T03 | Sola (pesada) |
| [ ] | T23 | Regenerar bindings Wails | T20–T22 | Sola |
| [ ] | T24 | Estado Svelte | T23 | Sola |
| [ ] | T25 | UI de búsqueda y navegación | T23 | Sola |
| [ ] | T26 | Vista de detalle y episodios | T24, T25 | Sola |
| [ ] | T27 | Reproductor multi-recurso | T19, T24 | Sola (pesada) |
| [ ] | T28 | E2E (offline + real) | T17, T20 | Sola |
| [ ] | T29 | Ajustes finales y refactor | todas | Sola (modelo fuerte) |

Paralelizables (si usas varias sesiones a la vez): T06/T09/T10 · T12–T15 entre sí · T17/T18 · T21/T22 · T24/T25 · T26/T27.

## Desviaciones que afectan a tareas futuras

> Cambios respecto a `tareas_v4.md` que una tarea futura debe tener en cuenta. Las demás (históricas, resueltas o sin impacto) están en `docs/BITACORA.md`.

- T03: valores de `ResourceType` en minúsculas (`hls`, `mp4`, `iframe`): **T16/T17** comparan contra `models.ResourceHLS/MP4/Iframe` y la frontend (**T27**) contra esos strings.
- T03: `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""` (sin `omitempty`) → si molesta a la frontend, lo cambian **T23/T27**.
- T03: `WatchProgress.CurrentTime`/`TotalDuration` son segundos en `float64` (valores con decimales del `<video>`); calcular `IsWatched` es de **T22**.
- T04: ~~**`AnimeFLV` no cumple `Provider`** (su `Search` no lleva `ctx` y `GetAnimeDetails` devuelve `models.Anime` en vez de `models.AnimeDetail`); `app.go` sigue llamando a su `Search(query)`~~ → **resuelto en T18**: paquete movido a `internal/scraper/animeflv/`, cumple `Provider` completo y `app.go` llama `Search(a.ctx, query)` (edición mínima; el montaje definitivo con `Registry` es de **T20**).
- T11: los extractores reciben `extractors.Page`, no `jkanime.EpisodePage` (regla de dependencias); convertir `EpisodePage → Page` **ya existe** (`jkanime.paginaExtractores`, montada en **T17**).
- T12: `VideoSource.Referer` = URL del **propio iframe** del player (decisión de la captura con Playwright: el `.m3u8` del CDN responde 200 con ese referer). **T19 inyecta el Referer en el proxy local**: coordinar allí para no duplicar la cabecera.
- T13: `VideoSource.Referer` = URL del **propio iframe** (mismo criterio que T12). Ojo: la respuesta de `jk.php` declara `<meta name="referrer" content="never"/>`, así que el navegador no enviaría referer al `stream/jkmedia/…`; si **T16/T19** prefieren otro criterio (o vacío), decidirlo allí.
- T14: **`extractor.go` no necesitó cambios** (mismo caso que T12/T13): registrar el extractor es devolver `NameServers`, que ya estaba en `ordenCanonico` desde T11.
- T14: `ServersExtractor` emite como Iframe el **remote decodificado** (p. ej. `https://voe.sx/e/mlnsng0cwwn7`), no el wrapper `jkplayer/c1?u=…&s=…` que construye el JS de la página: verificado con `curl.exe` que `c1` responde con `<iframe src="<remote descodificado">` — solo decodifica, así que se ahorra una petición. Si algún host exigiera el wrapper (referer de jkanime.net), reconsiderarlo en **T17/T19**.
- T14: las entradas `Mediafire`/`1Fichier` (las que la propia página excluye del bloque de vídeo: solo publican descarga) **no producen fuente si no hay resolver**. El contrato de delegación vive en `servers.go`: `RemoteResolver{ CanResolve(server) bool; Resolve(ctx, doer, remote) }`, y `ServersExtractor` le pasa su `Base.Doer` (no crea uno propio). **T15** implementa `MediaFireResolver` con esa interfaz y **T17** monta la cadena con `&ServersExtractor{Base: Base{Doer: cli}, Resolvers: []RemoteResolver{&MediaFireResolver{}}}` *(así está en `jkanime.cadenaDeExtractores`)*.
- T14: `VideoSource.Server` de las fuentes Iframe es el **nombre real del host** (`"VOE"`, `"Mega"`, `"Filemoon"`…), no `"servers"` (el Resolver solo rellena la etiqueta si viene vacía) → **T16/T17** no deben asumir que `Server` es siempre un nombre de extractor.
- T14: `testdata/servers_invalido.html` es un fixture **artificial** (el sitio solo sirve `var servers` con JSON válido): reproduce una lista cortada a media entrada para el caso de JSON inválido que pide la comprobación. El resto de datos del archivo (el objeto de VOE) es real, recortado del fixture real.
- T15: `MediaFireResolver` **no es un `VideoExtractor`** y no tiene `Name()`: cumple el `RemoteResolver` de T14 y se instancia sin estado (`MediaFireResolver{}`). **T17** monta `&ServersExtractor{Base: Base{Doer: cli}, Resolvers: []RemoteResolver{&MediaFireResolver{}}}`; la operación de log es `mediafire`.
- T15: `VideoSource.Referer` sale **vacío** porque `downloadN.mediafire.com` no lo exige: probes de 1 byte con `curl.exe` devuelven 206 con y sin cabecera (`content-type: video/mp4`, `accept-ranges: bytes`, `157058980` B = 149.78 MB, que coincide con el "Download (149.78MB)" del botón). Si llegara a exigirlo, va la URL de la página del archivo: decidirlo en **T19** (T16 no lo tocó: preserva lo que el extractor manda).
- T15: la URL que devuelve es un enlace **firmado y temporal**: no se persiste ni se cachea en ninguna capa (regla 9). **T16/T17** no la guardan ni deduplican por ella literalmente (deduplicación por recurso base, como ya hace `ServersExtractor` y refuerza `Normalize`).
- T15: `Type` sale de `tipoDeRecurso` (hoy siempre `.mp4`); un enlace sin extensión reconocible quedaría `""` → **T16** lo infiere por extensión y, si no hay señal, la fuente queda con `Type=""` y el selector de **T17** la deja el último (resuelto).
- T15: solo resuelve MediaFire; `1Fichier` —la otra entrada de solo descarga que excluye el JS de la página— sigue sin resolver (**T17 no lo amplió**) → si el sitio llega a servirla, ampliar en **T27/T29**.
- T15: el HTML de los casos negativos de `mediafire_test.go` es **artificial** (sin botón de descarga, `href="#"` en estado "Preparing", enlace a otro host, comillas simples): el sitio solo sirve la página completa con el enlace correcto. Regla 11.
- T16: **`extractor.go` no estaba en "Puedes tocar"** pero T16 pide integrar el normalizador al final de `Resolver.Resolve`; el prompt de la sesión autorizó ese punto de integración. El cambio es de 3 líneas: `total = Normalize(total)` justo antes de la comprobación `len(total) == 0` (si el normalizador se queda sin nada, sigue saliendo `ErrSinFuentes`).
- T16: **`extractor_test.go` se parcheó con autorización explícita del humano** (2 tests, ~7 líneas): `TestNewOrdenaLaCadena` y `TestNewCopiaYDescartaNulos` devolvían URLs falsas sin esquema (`s1`, `f1`, `p1`, `j1`, `z1`, `x`) que `Normalize` descarta; ahora son `https://cdn.example.test/<n>.m3u8` y la aserción de URLs del segundo test apunta a esas. Nada más de ese archivo cambió.
- T16: clave de deduplicación = **recurso sin query, con fragmento** (esquema y host en minúsculas): la query solo cambia la firma temporal del mismo vídeo, pero el fragmento de `mega.nz/embed/<id>#<clave>` **es** la clave de descifrado y distingue un vídeo de otro. La URL de salida se conserva íntegra (query y fragmento incluidos); gana la primera aparición y el duplicado solo rellena los campos vacíos (`Type`, `Quality`, `Server`, `Referer`, `Headers`, `Subtitles`).
- T16: `Type` **solo se infiere cuando viene vacío** (`.m3u8`/`.m3u` → `hls`, `.mp4`/`.mkv`/`.webm` → `mp4`); un tipo existente nunca se pisa, así los Iframe con `.mp4` en la ruta (`streamtape.com/e/…/01.mp4`) siguen siendo `iframe`. Si tras inferir sigue vacío, la fuente **se conserva con `Type=""`** → la jerarquía de `SelectBestSource` (T17) la trata por debajo de `iframe`.
- T16: `Normalize` **preserva** `Referer`/`Headers`/`Subtitles` tal cual; no completa `Referer` de Iframe/Mediafire ni reescribe el de T12/T13 (los criterios de T12–T15 siguen vigentes) → la decisión de Referer del embed queda en **T19**.
- T16: `Normalize` se ejecuta **dentro de `Resolve`**, así que **T17 no necesita un paso de normalización aparte** en `GetVideoSources`: `fetchEpisodePage → Resolver.Resolve` ya sale normalizado.
- T16: `TestNormalizeConFixtures` reescribe en el test `https://jkanime.net` → URL del `httptest` y el remote base64 de Mediafire → servidor local (el fixture trae ambos apuntando al sitio real; sin esa reescritura la cadena saldría a la red).
- T17: **`ErrNotImplemented` se eliminó entero de `jkanime.go`** (T17 pide borrar el correspondiente): `GetVideoSources` era su último usuario, así que desaparecen la var y el import `errors` de ese archivo. Si alguna tarea futura vuelve a necesitarla, se redeclara (afecta a **T29**, que preveía "eliminar stubs").
- T17: `search_test.go` se tocó **solo dentro de `TestMetodosNoImplementados`** (la comprobación de `ErrNotImplemented` ya no aplica: no queda ningún método sin implementar) y, obligatoriamente, se quitó el import `miapp/internal/scraper` del archivo, que quedó sin uso. El nombre de la función queda como vestigio (solo comprueba `Name()`) → **T29** puede renombrarlo.
- T17: se crearon **dos archivos nuevos** fuera de "Puedes tocar", porque las pruebas que pide la tarea no caben en los archivos listados: `internal/scraper/jkanime/get_video_sources_test.go` (el test offline con servidor falso completo) y `.../get_video_sources_integration_test.go` (el smoke de red, permitido explícitamente). Reportado aquí por la regla de escape.
- T17: el test offline **lee fixtures del `testdata/` de `extractors/`** (solo lectura, `extractors/testdata/jkplayer_um.html`, `jkplayer_umv.html`, `jkphp.html`, `mediafire.html`): la página de episodio es `testdata/episode_page.html`. Para no salir a la red reescribe en el test `https://jkanime.net` → URL del `httptest` y el remote base64 de Mediafire → servidor local, y monta un `Transport` (`transporteSoloLocal`) que **falla cualquier petición ajena al servidor del test** (regla 13, garantía offline real).
- T17: la página real de bleach/1 **no trae ningún iframe de `jk.php`/`jkplayer/jk`** (verificado sobre `episode_page.html`), así que JKPHP no se ejecutaría en el flujo: el test `TestGetVideoSourcesCadenaCompletaConJKPHP` añade ese iframe como **HTML artificial** (regla 11) para recorrer la cadena completa; el otro test usa solo fixtures reales. Anotado también en `docs/firmas/jkanime.md`.
- T17: jerarquía de `SelectBestSource` = **HLS (4) > MP4 (3) > iframe (2) > `Type` vacío/desconocido (1)**; los Iframe nunca se auto-eligen si hay HLS o MP4. "Iframe soportado" = **todos** los iframes (hoy no hay lista de hosts soportados; si T27 la define, se aplica allí). Desempate por `Quality` con el primer número de la etiqueta (`"1080p"` → 1080, sin número = 0) y, a igualdad, gana la **primera** de la lista (orden canónico de la cadena).
- T17: `GetVideoSources` **no añade un paso de normalización**: `Normalize` ya corre dentro de `Resolver.Resolve` (T16). La cadena se monta en cada llamada (`cadenaDeExtractores`, barata y sin estado) con el `Doer` del proveedor, y `opts.NoCache` **se ignora** porque no hay ninguna caché (regla 9) → **T27**.
- T17: smoke de red en verde contra el sitio real: `GetVideoSources` sobre `jkanime:bleach:episode:1` devolvió **11 fuentes** — `jkplayer` 1 HLS, `Mediafire` 1 MP4 y 9 iframes (Mega, VOE, Vidhide, Filemoon, Mixdrop, Mp4upload, Streamtape, Streamwish, Doodstream) — y `SelectBestSource` eligió el HLS (`https://nika.playmudos.com/…m3u8`, el mismo CDN que fija T12 como Referer). No se descargó ningún vídeo.
- T18: **movimiento de archivo**: `internal/scraper/animeflv.go` y `animeflv_test.go` (paquete `scraper`) desaparecieron y viven ahora en `internal/scraper/animeflv/` (paquete `animeflv`), como pide el layout de `00_REGLAS.md`. `app.go` se editó **solo** lo mínimo para compilar (autorizado en el prompt): import, tipo `*animeflv.AnimeFLV`, constructor `animeflv.New(httpx.NewClient(httpx.ClientOptions{}), "")` y `Search(a.ctx, query)`; el método `SearchAnime` que usa `App.svelte` no cambió de firma → **T20** lo rehace con el `Registry`.
- T18: `AGENTS.md` sigue hablando de `internal/scraper/animeflv.go` y del smoke en `go test -tags integration ./internal/scraper/ -v` → **queda desactualizado** (el paquete y el smoke ahora son `./internal/scraper/animeflv/`). No se tocó `AGENTS.md` (fuera de "Puedes tocar"); actualizarlo en **T20/T29**.
- T18: el plan pedía "parsear título, slug, tipo, sinopsis", pero la tarjeta `.search-series-card` **solo** trae enlace, título y portada (verificado en el fixture crudo; el `span.Type` y demás viven en la barra lateral "ANIMES EN EMISIÓN", no en los resultados) y `models.Anime` no tiene campos de tipo/sinopsis (T03 fuera de "Puedes tocar"). Con `HasDetails: false` tampoco hay página de detalle que parsear → **no se parsean tipo ni sinopsis**.
- T18: no existe `HasDetails` en `scraper.Provider` (T04 no lo definió): T18 lo materializa como `GetAnimeDetails` → `scraper.ErrNoDetails`. Si una tarea futura necesita un flag explícito, es cambio de interfaz → **T20/T29**.
- T18: la **decisión de producto** (mostrar AnimeFLV solo si JKAnime no tiene resultado equivalente por título normalizado, y "Detalle no disponible para este proveedor" al seleccionarlo) **no vive en Go**: `animeflv` no puede importar `jkanime` (regla de dependencias) y el mensaje depende de la UI → **T20** (orquestación/registro) y **T25** (comparación y aviso).
- T18: el slug del href se toma por **ruta** (`/anime/<slug>/`), ignorando el host: los href del fixture son absolutos de `animeflv.or.at` y, con host estricto, no casarían contra el `httptest.Server` de los tests. Contrapartida: un enlace ajeno con esa misma ruta se aceptaría (no ocurre en el sitio).
- T18: `TestSearchDescartaEnlacesInesperados` usa **HTML artificial** (regla 11): el sitio no sirve tarjetas con rutas distintas de `/anime/<slug>/` ni sin título, así que el descarte solo se puede probar inventando marcado.
- T18: `testdata/search_empty.html` es la respuesta real a `/?s=shingeki`, pero el test la pide como `zzzznoexisteestetitulo` (mismo criterio que `jkanime/search_empty.html`: lo que importa es que el HTML trae 0 tarjetas).
- T19 (**sondeo de red, 2026-10-06**; sin vídeos descargados): fuente HLS fresca de `jkanime:bleach:episode:1` vía `TestGetVideoSourcesRedReal` (11 fuentes; la mejor, el HLS de `jkplayer`). El `.m3u8` vive en un CDN (`nika.playmudos.com`) con URL **firmada temporal** (`st` + `e`; caduca en minutos) y los **segmentos** van a **otros 6 hosts** (`cdn1..6.ducvomes.com`) con URLs **absolutas y sin firma** → el reescribir debe resolver contra la URL final del manifiesto y aceptar URIs absolutas de hosts distintos.
- T19 (sondeo): la playlist es de **medios** (VOD, `#EXT-X-ENDLIST`, 353 segmentos, 47680 B) y **no trae `#EXT-X-KEY` ni `#EXT-X-MAP` ni sub-playlists**; aun así Register reescribe `URI="…"` (cubre KEY/MAP/MEDIA) y líneas de URI para que sirva cualquier manifiesto.
- T19 (sondeo): el `.m3u8` responde **200 con y sin `Referer`** (el de la fuente, `https://jkanime.net/` y sin cabecera), con o sin User-Agent de navegador → **hoy no exige nada**. Devuelve `Content-Type: application/vnd.apple.mpegurl`, `Access-Control-Allow-Origin: *` y `Accept-Ranges: bytes`.
- T19 (sondeo): probe de 1 byte (`Range: bytes=0-0`) a un segmento → **206** con `Content-Type: video/mp2t`, `Content-Range: bytes 0-0/21620` y `Access-Control-Allow-Origin: *`; idéntico con y sin `Referer` y sin User-Agent → **el CDN de segmentos tampoco exige ninguna cabecera**. Aun así Register inyecta en la sesión el `Referer`/`Headers` del `VideoSource` (T12/T13 fijan uno y otros hosts sí lo exigen), y el proxy pone CORS siempre: el upstream de `OPTIONS` devuelve **405** (nginx), así que el preflight por `Range` lo responde **el proxy**, nunca se reenvía.
- T19 (**implementación**): paquete nuevo `internal/stream` (`stream.go`, `reenvio.go`, `manifiesto.go`, `sesiones.go`, `red.go`). Entregable = `NewProxy(doer, opts...)` + `Handler()` + `Register(...)`, más `Escuchar()/Cerrar()` (bind `127.0.0.1:0`, obligatorio para que `Register` tenga base) y las opciones `PermitirRedPrivada/ConTTL/ConLimiteSesiones/ConBaseURL`. **`Register` sin `Escuchar()` ni `ConBaseURL` devuelve la fuente sin registrar** (no hay dónde apuntar) → **T20** debe llamar `Escuchar()` al arrancar y `Cerrar()` al parar.
- T19: `NewProxy(nil)` construye el cliente con anti-SSRF (lo que debe hacer la app); **un doer inyectado se usa tal cual y sin el control de red**, porque el paquete no puede meter su `Dialer` en un transporte ajeno → si **T20** inyecta doer propio, asume esa pérdida (o pasa `nil`).
- T19: reenvío con `httpx.HTTPDoer.Do` + `io.Copy` (sin editar `httpx`), **sin `Timeout` global** en el cliente (el streaming dura minutos; `ResponseHeaderTimeout` 30 s) y con el `ctx` de la petición del reproductor, que es lo que cancela el upstream si cierra la conexión. `Proxy` del entorno **desactivado** en el cliente interno: con `HTTP_PROXY` el control validaría la IP del proxy y no la del destino.
- T19: cabeceras del reproductor reenviadas solo por lista blanca (`Range, If-Range, Accept, Accept-Language, Origin, If-None-Match, If-Modified-Since`); **`Cookie`/`Authorization` del cliente nunca** (los de la sesión sí) y **`Accept-Encoding` no se reenvía** (lo pone el transporte, que descomprime → `Content-Length` solo se copia si `!resp.Uncompressed`). De la respuesta: lista blanca (`Content-Type, Content-Range, Accept-Ranges, ETag, Last-Modified, Cache-Control, Content-Disposition`) + `Content-Length` calculado; **sin `Location`** (el cliente interno sigue los redirects y el control se aplica a cada salto).
- T19: manifiestos → lectura limitada a **5 MB**, solo con status 200 y cuerpo que empieza por `#EXTM3U` (si no, se devuelve **sin tocar**); se reescriben las líneas de URI (relativas y absolutas, resueltas contra la URL final tras los redirects) y **cualquier atributo `URI="…"`/`URI=…`** de tag, no solo `#EXT-X-KEY`/`#EXT-X-MAP`. `.vtt` → `Content-Type: text/vtt` forzado. El proxy **no cachea nada** (regla 9).
- T19: **logs**: `logx.Redact` no cubre los parámetros firmados reales (`st=`, `e=`), así que el proxy registra la URL **reducida a esquema+host+ruta** (`urlSegura`). Si más adelante se quiere pasar la URL entera por `logx.Request`, hay que ampliar los patrones de `internal/logx` (fuera de "Puedes tocar" en T19) → **T20/T29**.
- T19 (**tests**): 22 tests en `internal/stream/{stream,manifiesto,red}_test.go`, todos con `httptest.Server` y sin red; `go test -race -count=1 ./...` en verde. El caso "redirect a loopback rechazado por defecto" usa un **primer salto ficticio** (transporte truca que devuelve el 302 sin conectar): con la opción por defecto ningún host local es alcanzable, así que no se puede encadenar un redirect local real sin salir a la red (regla 13).
- T19 (**fixtures**): `internal/stream/testdata/manifiesto.m3u8` es **real** (playlist de bleach capturada en el sondeo, 47680 B, sin firmas ni cookies) y `testdata/manifiesto_etiquetas.m3u8` es **artificial** (regla 11: el manifiesto real de este contenido no trae `#EXT-X-KEY`, `#EXT-X-MAP` ni URIs relativas y T19 exige reescribirlos).
- T19: se creó y borró un `*_integration_test.go` temporal en `jkanime` (`sondeo_hls_*`) solo para volcar el Referer de la fuente HLS al `curl.exe`; no queda en el repo. **No se tocó ningún archivo fuera de `internal/stream/**`** (más `PROGRESO.md` y `docs/firmas/stream.md`): `frontend/**` y `app.go` intactos.
- T19 (**cerrada**): firmas reales en **`docs/firmas/stream.md`**. Comprobación final: `gofmt -l .` sin salida · `go vet ./...` OK · `go vet -tags integration ./...` OK · `go test -race -count=1 ./...` OK (los 8 paquetes, `internal/stream` 22 tests) · `go build ./...` OK.

## Pendientes activos

> Solo lo que una tarea futura debe resolver, con la tarea asignada. Los cerrados/históricos están en `docs/BITACORA.md`.

- T01: `app.go` todavía no llama a `logx.New(...)` ni usa `logx.Request` → **T20**.
- T02: ~~`internal/scraper/animeflv.go` sigue con su propio `http.Get`~~ → **resuelto en T18**: el paquete `animeflv` usa `httpx.Get` con `httpx.DefaultUserAgent` y errores tipados.
- T03: los modelos ya los usan T08 (detalle), T09 (episodios/lotes) y **T19** (`VideoSource` en `Register`); siguen sin consumidor en historial y serialización a Wails → **T20, T22**.
- T04: nadie registra providers en el `Registry` (el montaje es `jkanime` + `animeflv`) → **T20**.
- T04: `SourceOpts.NoCache` no lo consume nadie → **T27** (`GetVideoSources` lo ignora porque no hay caché, regla 9).
- T05: `ParseEpisodeID` solo lo usan sus tests (`NewEpisodeID` ya lo usa T09) → **T20, T26**.
- T06: el aviso `slog.Warn` va al logger por defecto de `slog`, no al de `logx`; para unificar, instalar `slog.SetDefault(logx.New(dev))` → **T20/T29**.
- T07: la primera búsqueda descarga el mapa de excepciones de GitHub (máx. 5 s, cacheado); `ExceptionResolver` compartido o precarga al arrancar → **T20**.
- T07: ~~`GetVideoSources` devuelve `ErrNotImplemented` a propósito~~ → **resuelto en T17**: implementado y `ErrNotImplemented` eliminado de `jkanime.go` (ya sin ningún usuario).
- T09: `StreamAnimeDetails` todavía no tiene consumidor (lo usará T20 con `runtime.EventsEmit` por lote; `GetAnimeDetails` ya lo usa vía `jkanime.go`) → **T20**. Sin consumidor, el único test de cancelación real es `TestStreamCancelacionSinFugas`.
- T09: `pausaPaginas` es constante de 300 ms compartida por stream y `GetAnimeDetails`; otro ritmo ⇒ inyectarla → **T20**.
- T09: si falla una página de fondo, el lote final lleva `Err` y los episodios ya emitidos **no** se revocan; `GetAnimeDetails` fallando a mitad devuelve detalle vacío y **sin** episodios parciales → si T20 quiere parciales, es cambio de contrato en **T20**.
- T10: ~~`fetchEpisodePage` todavía no lo llama nadie fuera de sus tests~~ → **lo llama `GetVideoSources` desde T17**.
- T10: `EpisodePage.Headers` es `http.Header` (mapa `string → []string`) y `EpisodePage` no se serializa a Wails hoy → si se expone, **T20/T27**.
- T10: la sesión depende de que el doer lleve cookiejar (`httpx.NewClient` lo pone): sin jar, el `_token` no se envía y la web devuelve 419 (T09 refresca una vez y, si sigue sin jar, acaba con el 419 original en el lote final). **T17** ya consume `fetchEpisodePage` con el doer del proveedor → lo que quede por garantizar al montar la app es de **T20**.
- T11: ~~nadie construye todavía el `Resolver`~~ → **resuelto en T17**: `JkAnime.cadenaDeExtractores` monta JKPlayer + JKPHP + Servers (con `MediaFireResolver`) y `paginaExtractores` convierte `EpisodePage → Page`; la cadena se reconstruye en cada llamada (barata y sin estado) → si **T20** quiere inyectarla o cachearla, es allí.
- T11: `Resolve` agrega **todas** las fuentes en orden canónico de extractor, sin ordenar por calidad ni elegir la mejor: descarte y deduplicación los hace **T16** (`Normalize` dentro de `Resolve`) y la elección de la mejor vive en `SelectBestSource` (**T17**), que **hoy solo lo usan sus tests** → consumidor en **T20/T27**. Si alguna tarea quiere "parar en la primera usable", es cambio de política y debe registrarse aquí.
- T11: `Resolve` ya valida el esquema de las URLs y completa `Type` vacío (vía `Normalize`, **T16**). `Headers`/`Referer`/`Subtitles` **se preservan**, no se inventan → el criterio de Referer del embed sigue siendo de **T19**.
- T12: ~~`JKPlayerExtractor` todavía no lo construye nadie~~ → lo monta `cadenaDeExtractores` desde **T17**.
- T12: `Type` puede quedar `""` si ni la URL ni el cuerpo del player dan señal → **T16** lo infiere por extensión; si sigue vacío, la fuente se conserva y la jerarquía de `SelectBestSource` la deja por detrás de `iframe` (decisión de **T17**, anotada en desviaciones).
- T12: cada iframe distinto dispara su propia petición (hasta 4 por página): los repetidos ya no se piden (dedup por URL dentro del extractor) y las fuentes del mismo vídeo con firmas distintas se caen en la dedup por recurso base (extractor y `Normalize`, **T16**). Quedan 2 peticiones cuando la página emite `um` y `umv` —se descubren iguales solo al pedirlas—, así que no hay nada pendiente aquí.
- T13: ~~`JKPHPExtractor` todavía no lo construye nadie~~ → lo monta `cadenaDeExtractores` desde **T17**.
- T13: la descarga de la página del player está duplicada con `JKPlayerExtractor.fuentesDePlayer` (mismo límite y mismos helpers) → unificar en **T29** en una sola función parametrizada por `Server`/`Extractor`.
- T14: ~~`ServersExtractor` todavía no lo construye nadie~~ → lo monta `cadenaDeExtractores` con `Resolvers: []RemoteResolver{&MediaFireResolver{}}` desde **T17**.
- T14: las fuentes Iframe salen con `Referer`, `Quality`, `Headers` y `Subtitles` vacíos → **T16** los preserva vacíos (no inventa Referer); el criterio de Referer para el embed quedó en **T19/T27**: `Register` inyecta en la sesión el `Referer` que traiga la fuente (vacío si lo trae vacío).
- T14: `Remote.Tamano` (el `size` del sitio, "150 MB") y `Server.Lang` llegan parseados pero nadie los consume → **T26/T27** si la UI quiere mostrarlos (T16 no los toca: no forma parte de `VideoSource`).
- T15: ~~`MediaFireResolver` todavía no lo monta nadie~~ → lo monta `cadenaDeExtractores` (`ServersExtractor.Resolvers`) desde **T17**.
- T15: la fuente sale con `Referer`, `Quality`, `Headers` y `Subtitles` vacíos → **T16** los preserva vacíos (el Referer vacío de Mediafire está verificado; **T19** lo inyecta tal cual, es decir, no manda ninguno).
- T15: el enlace firmado caduca en minutos; si el reproductor arranca tarde o retoma tras un ratón, el MP4 puede responder 403. **T17 no re-resuelve** (no cachea ni guarda nada, regla 9) y **T19 tampoco** (el proxy solo devuelve el status del origen, 403 incluido) → el re-resolve antes de reproducir es de **T27**.
- T16: `Normalize` no ordena ni elige: la cadena sale en orden canónico de extractor. La jerarquía (HLS > MP4 > iframe > `Type=""`) y el desempate por `Quality` los fija **T17** en `SelectBestSource`; `Quality` hoy llega vacía de todos los extractores, así que el desempate solo actuará cuando la rellene alguien (**T26/T27**).
- T16: las fuentes con `Type=""` (URL válida sin extensión reconocible) se conservan y **T17** las dejó **por detrás de `iframe`** en la jerarquía (último recurso).
- T16: `Normalize` vive en `extractors` y solo se aplica dentro de `Resolver.Resolve`; su único consumidor real hoy es `GetVideoSources` (**T17**) vía `Resolve`, sin paso extra.
- T17: `SelectBestSource` **solo lo usan sus tests** (offline y smoke): el consumidor real será el reproductor → **T20/T27**.
- T17: `GetVideoSources` ignora `opts.NoCache` (no hay caché, regla 9) y devuelve la lista completa sin elegir; si el reproductor quiere "la mejor ya elegida" o un reintento sin caché → **T27**.
- T18: nadie registra `animeflv` en el `Registry` (la app lo llama directamente desde `app.go`) → **T20** (`Register` con `animeflv.New(...)`, junto a `jkanime`).
- T18: `GetVideoSources` de AnimeFLV **falla siempre** con un error descriptivo (el proveedor es solo fallback de búsqueda) → **T20/T27** no deben ofrecer reproducción desde este proveedor sin antes comprobarlo.
- T18: `StreamAnimeDetails` no aplica (AnimeFLV no implementa `EpisodeStreamer`) y `GetAnimeDetails` devuelve `scraper.ErrNoDetails` → el mensaje "Detalle no disponible para este proveedor" y la búsqueda del equivalente en JKAnime por título normalizado son de **T20/T25**.
- T18: `app.go` construye el doer con `httpx.NewClient(httpx.ClientOptions{})` dentro de `NewApp()`; cuando **T20** rehaga `app.go`, decidir allí si el cliente se comparte con JKAnime (un solo jar/timeout) o cada proveedor lleva el suyo.
- T19: nadie monta todavía el proxy (ni `Escuchar()`, ni `Register`, ni `Cerrar()`) → **T20** (arranque/parada de la app, `Register` justo antes de entregar la fuente al reproductor). Orden: `NewProxy(nil) → Escuchar() → Register(fuente)`; sin `Escuchar()`/`ConBaseURL`, `Register` devuelve la fuente sin registrar.
- T19: el doer del proxy es interno (`NewProxy(nil)`) con anti-SSRF y **sin cookies**: si **T20/T27** prefieren inyectar el `httpx.NewClient` compartido, pierden el control de red (documentado en `docs/firmas/stream.md`) → decidirlo allí.
- T19: los logs del paquete usan `logx.Request(provider="stream", …)` con la URL truncada; solo se ven si **T20** instala `logx.SetDefault(logx.New(dev))` (pendiente de T01).
- T19: el proxy **no re-resuelve** ni cachea (regla 9) y devuelve tal cual los 4xx/5xx del origen (p. ej. 403 de una URL firmada caducada) → la recuperación es de **T27**. Tampoco filtra por `Type`: registra lo que le den, incluidos iframes.
- T19: no consume `SelectBestSource` ni `Quality`/`Server` (solo `URL`, `Referer`, `Headers`, `Subtitles`) → el resto de campos siguen esperando a **T26/T27**.
