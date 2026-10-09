# PROGRESO — AnimeAV1 (T31–T35)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivo de la v1.0**: el estado de **T01–T29** (cierre incluido) está archivado en **`docs/PROGRESO_v1.0.md`** — allí viven sus desviaciones por tarea, sus pendientes y las pruebas manuales del `.exe` de la v1.0. Aquí solo queda lo vigente: **T31–T35** (AnimeAV1). Rama de trabajo: `feature/animeav1`.

## Protocolo de sesión

1. Leer `AGENTS.md`, `tareas/00_REGLAS.md`, este archivo, `docs/firmas/<paquetes de los que depende la tarea>` y el archivo de la tarea (`tareas/Txx.md`).
2. Ejecutar **solo** la tarea indicada. Respetar "Puedes tocar" / "NO debes tocar".
3. Correr la comprobación de la tarea. Después, el humano la repite por su cuenta (`gofmt -l .`, `go vet ./...`, `go test -race ./...` y `npm run check` en frontend): no confiar solo en el reporte del agente.
4. Actualizar este archivo: marcar la tarea, anotar desviaciones y pendientes; las firmas nuevas van en `docs/firmas/<paquete>.md` (formato `### Txx — <paquete>`, ej. `docs/firmas/animeav1.md`).
5. Detenerse. No empezar la siguiente tarea.

## Prompt base por sesión

```text
Lee AGENTS.md, tareas/00_REGLAS.md, PROGRESO.md, tareas/T31.md y
docs/firmas/<paquetes de los que depende T31 — tabla de abajo>.
Ejecuta SOLO la tarea T31. Respeta "Puedes tocar" y "NO debes tocar".
Si necesitas tocar un archivo fuera de la lista, detente y repórtalo.
Al terminar: corre la comprobación, actualiza PROGRESO.md (estado, desviaciones,
pendientes) y docs/firmas/<paquete>.md si creaste firmas, y detente.
No empieces la siguiente tarea.
```

| Tarea | Lee docs/firmas/ de |
|:--------:|:--------:|
| T31 | scraper, models, httpx, jkanime |
| T32 | animeav1, scraper, models, jkanime, httpx |
| T33 | animeav1, scraper, models, extractors, stream, httpx |
| T34 | app, animeav1, scraper, cache, db, models |
| T35 | app, animeav1, stream, scraper |

## Prerrequisitos: fixtures reales (humano o agente con curl)

Antes de las tareas que parsean HTML/JSON real debe existir un **fixture real y sanitizado** (sin cookies ni tokens) en el `testdata/` correspondiente. Un fixture es la respuesta **cruda** del servidor guardada en un archivo, para que los tests corran sin red. Lo puede capturar el humano a mano o el agente con `curl.exe` (ver "Captura de fixtures" más abajo). Si no existe el fixture de la tarea, el agente se detiene y lo pide: **no inventa HTML ni selectores**.

- [x] **AnimeAV1 (investigación T30) → T31–T33** — `internal/scraper/animeav1/testdata/` con **6 fixtures crudos** de `https://animeav1.com` capturados con `curl.exe` y UA de navegador: `search.html` (`GET /catalogo?search=bleach`, 200, **61934 B**, 11 tarjetas), `search_empty.html` (`GET /catalogo?search=zzzznoexisteestetitulo`, 200, **22298 B**, 0 tarjetas y sin texto de "sin resultados"), `detail.html` (`GET /media/bleach`, 200, **89187 B**, `media` + **366** episodios embebidos y 50 tarjetas SSR), `detail_movie.html` (`GET /media/bleach-movie-1-memories-of-nobody`, 200, **21324 B**, 1 episodio), `episode_page.html` (`GET /media/bleach/1`, 200, **42441 B**, `episode` + `embeds`/`downloads` en la misma página) y `episode_data.json` (`GET /media/bleach/1/__data.json`, 200, **18447 B**, formato devalue: respaldo, no vía principal). Verificado con probe sobre el HTML **crudo**: `article.group/item.relative.text-body`, `img.aspect-poster`, `h3.line-clamp-2`, `href="/media/<slug>"`, `media:{id:195,…}`, `episodes:[{id,number}]`, `embeds:{DUB:[{server,url}]}`, `button` "1 - 50" y los 101 enlaces `/media/bleach/<n>`. **Sin challenge de Cloudflare, sin `Set-Cookie`, sin límite de tasa** (~40 peticiones → todas 200). **Sanitización: nada que sanitizar** (escaneo de `csrf|token|nonce|set-cookie|authorization|password|secret|signature|expires=|st=|cf_clearance|PHPSESSID|jwplayer` → 0 coincidencias; los únicos `localStorage` son el tema). Informe completo con las 7 preguntas, evidencias, riesgos y propuesta **T31–T35** en **`docs/investigacion/animeav1.md`**.
- [x] **AnimeAV1 fuentes (T33, paso 0)** — 2 fixtures crudos nuevos en `internal/scraper/animeav1/testdata/`, capturados con `curl.exe` + `httpx.DefaultUserAgent`, ambos `200 application/json`: `episode_movie_data.json` (`GET /media/bleach-movie-1-memories-of-nobody/1/__data.json`, **3500 B**, raíz `{media,episode,embeds,downloads}` con `embeds` **solo SUB**: UPNShare, Voe, Byse, MP4Upload y el reproductor propio) y `episode_404_data.json` (`GET /media/bleach/367/__data.json`, **205 B**, HTTP **200** + nodo `{"type":"error","error":{"message":"No encontrado"},"status":404}`). Sanitización: mismo escaneo de tokens → **0 coincidencias**, sin `Set-Cookie`, sin challenge.
  - Formato observado del nodo de página de episodio: `{"media":1,"episode":1167,"embeds":1170,"downloads":1198}`; `embeds` es `{"SUB":[{server,url}×5],"DUB":[{server,url}×5]}` (en la película solo hay clave `SUB`) y `downloads` (5 enlaces de descarga) **se ignora**. El `<id>` del reproductor aparece como `https://player.zilla-networks.com/play/<32 hex>` en ambas claves.
- [x] **AnimeAV1 `__data.json` (T31, paso 0)** — 4 fixtures crudos nuevos en `internal/scraper/animeav1/testdata/`, capturados con `curl.exe` + `httpx.DefaultUserAgent`, todos `200 application/json`: `catalogo_data.json` (`GET /catalogo/__data.json?search=bleach`, **16135 B**), `catalogo_vacio_data.json` (`GET /catalogo/__data.json?search=zzzznoexisteestetitulo`, **5003 B**), `detail_data.json` (`GET /media/bleach/__data.json`, **16764 B**) y `detail_movie_data.json` (`GET /media/bleach-movie-1-memories-of-nobody/__data.json`, **2538 B**).
  - **Equivalencia con los HTML de T30 comprobada** (script de sondeo sobre los 6 archivos): catálogo → **11 resultados** con `id` (string)/`title`/`slug` **idénticos** a los del `<script>` de hidratación de `search.html` y presentes en las 11 tarjetas SSR; vacío → `results:[]` y `totalRecords:0` igual que `search_empty.html` (0 `<article>`); detalle → `media.id=195`, `media.slug="bleach"`, `episodesCount=366` y los **366 pares `{id,number}` idénticos** a `detail.html`; película → `episodesCount=1` y el par `14703:1` idéntico a `detail_movie.html`. **Son equivalentes → la vía `__data.json` sirve.**
  - Formato observado: `{"type":"data","nodes":[null,{…nodo auth…},{nodo de página}]}`; el **nodo de página es el último** con `type:"data"`; dentro de cada nodo, `data` es un array plano devalue donde los números **dentro** de objetos/arrays son referencias por índice (`results:1` → `data[1]`), los números sueltos del array son valores literales y `-1` = `undefined` (visto en `mirrors:-1` de `episode_data.json`). Los 4 fixtures nuevos solo usan `-1` como constante especial: sin tags (`Date`/`Map`/…), sin huecos ni `sparse`.
  - Sanitizado: escaneo de tokens/firmas (mismo patrón que T30) → **0 coincidencias** en los 4 archivos, sin `Set-Cookie`, sin challenge.
- [x] **AnimeAV1 detalle (T32, paso 0)** — 8 fixtures crudos nuevos en `internal/scraper/animeav1/testdata/`, capturados con `curl.exe` + `httpx.DefaultUserAgent`, todos `200 application/json`: `detail_one-piece_data.json` (`GET /media/one-piece/__data.json`, **54096 B**, status 1 y **1180** episodios), `detail_battle-spirits-re-zekkai-no-kuu_data.json` (**1263 B**, status 2), `detail_ajin-ova_data.json` (**1618 B**, status 0, `category.slug=ova`), `detail_one-piece-film-strong-world-episode-0_data.json` (**1715 B**, slug con "episode-0" pero `number:1`), `detail_boku-no-hero-academia-more_data.json` (**1707 B**, `category.slug=especial`), `detail_gintama-season-1_data.json` (**10635 B**, hueco de numeración), `detail_girls-band-cry-movie_data.json` (**1267 B**, película de **2** episodios) y `detail_404_data.json` (**205 B**, `GET /media/<slug-inexistente>/__data.json` → **HTTP 200** con `{"type":"error","error":{…},"status":404}`). Sanitización: mismo escaneo de tokens → **0 coincidencias**, sin `Set-Cookie`, sin challenge.

Nombres de archivo (dentro de `internal/scraper/animeav1/testdata/`): `search.html`, `search_empty.html`, `detail.html`, `detail_movie.html`, `episode_page.html`, `episode_data.json` (T30) · `catalogo_data.json`, `catalogo_vacio_data.json`, `detail_data.json`, `detail_movie_data.json` (T31) · `detail_one-piece_data.json`, `detail_battle-spirits-re-zekkai-no-kuu_data.json`, `detail_ajin-ova_data.json`, `detail_one-piece-film-strong-world-episode-0_data.json`, `detail_boku-no-hero-academia-more_data.json`, `detail_gintama-season-1_data.json`, `detail_girls-band-cry-movie_data.json`, `detail_404_data.json` (T32) · `episode_movie_data.json`, `episode_404_data.json` (T33).

### Captura de fixtures

**Regla de oro:** el fixture debe ser lo que recibirá goquery, es decir, el HTML **tal como lo envía el servidor**, no el DOM ya modificado por JavaScript. goquery no ejecuta JS: un dato que solo aparece tras cargar scripts no existe para el scraper en Go.

Ojo con las herramientas de Playwright: `browser_snapshot` devuelve el árbol de accesibilidad, **no** HTML, y no sirve como fixture. Los nombres exactos de las herramientas dependen de la versión de tu servidor MCP; usa las equivalentes de navegar, evaluar JS y listar peticiones de red.

Procedimiento, en orden:

1. **Explorar:** navegar a la URL con Playwright y, con la lista de peticiones de red, identificar la petición exacta (URL, método, cabeceras y cuerpo necesarios, sin copiar valores secretos).
2. **Obtener la respuesta cruda**, por orden de preferencia:
   - Con `curl` o un programa Go corto desde Bash, guardando directo a archivo (`-o testdata/archivo`). Es lo preferible porque no pasa por el contexto y reproduce lo que hará el scraper. **Así se capturaron los fixtures de AnimeAV1 (T30).**
   - Si el sitio bloquea clientes HTTP (Cloudflare) pero el navegador pasa: dentro de la página, `fetch(url).then(r => r.text())` mediante evaluación de JS. Devuelve la respuesta del servidor sin modificar por scripts.
3. **Comprobar que no es un bloqueo:** si el archivo contiene "Just a moment" o similar, no es un fixture válido. Anotarlo en "Pendientes activos" (abajo).
4. **Guardar en archivo, no pegar en el chat.** Inspeccionar con `grep` o fragmentos de pocas líneas, nunca leyendo la página completa.
5. **Recortar solo si es enorme:** conservar el `<head>` con lo necesario y el bloque relevante con 2 o 3 elementos de ejemplo. No alterar la estructura ni los nombres de clases.
6. **Sanitizar:** reemplazar cookies, tokens CSRF reales y cualquier dato de sesión por valores falsos como `TOKEN_DE_PRUEBA`.
7. **Verificar que el dato está en el archivo crudo** (`grep`). Si solo aparece en el DOM renderizado, goquery no puede extraerlo: detenerse, reportarlo y buscar el endpoint que entrega ese dato.
8. **Validar con goquery:** el test de la tarea carga el fixture y comprueba los selectores. Ese test es la prueba de que la captura sirve.

Para cuidar el contexto (modelos gratuitos):

- `animeav1.com` **no tiene challenge ni rate-limit observado** (T30): con `curl.exe` + UA de `httpx` basta; usa el MCP de Playwright solo si un sitio bloquea al cliente HTTP.
- No pedir capturas ni lecturas de página completa en el chat; solo resultados acotados.
- Si hay un montón de peticiones de red, filtrar por la URL que interesa en lugar de listarlas todas.

## Investigación T30 — base de T31–T35

- T30 (**informe, sin código**): `docs/investigacion/animeav1.md` responde las 7 preguntas
  (búsqueda, detalle, episodios, fuentes, protecciones, CDN, estrategia) con evidencias y
  fixtures. Hallazgos que **T31–T35** deben tener en cuenta:
  - **Todo el dato viene en el HTML** (SvelteKit SSR): búsqueda, detalle y episodio no
    hacen **ningún XHR**; los objetos viven en el `<script>` de hidratación como literal
    **JS** (`void 0`, flechas) → hay que hacer *brace-matching* + JS→JSON, o parsear el
    marcado visible con goquery. Existe `GET …/__data.json` (JSON puro pero **devalue**,
    referencias por índice) documentado como respaldo, no como vía principal.
  - **`Sec-Fetch-Site: same-origin` es obligatorio para los segmentos HLS** de
    `player.zilla-networks.com` (403 sin ella; 206 con `Range` con ella; el **Referer es
    indiferente**). Se declara en `models.VideoSource.Headers` y `stream/reenvio.go` ya la
    inyecta; el manifiesto (`Content-Type: application/x-mpegURL`) responde 200 con esa
    cabecera y `stream.esManifiesto` lo detecta **sin depender de la extensión** (la URL
    no acaba en `.m3u8`) → **sin cambios en `internal/stream`**.
  - La URL del manifiesto se **deriva sin petición**: `play/<32hex>` → `m3u8/<32hex>`
    (regla del JS del player: último segmento del path, longitud 32). `/play/*` exige
    Referer de animeav1 → **nunca se pide esa página**.
  - **Episodios completos en el detalle** (`episodes:[{id,number}]`, 366 en Bleach): el
    servidor pinta 50 tarjetas y el resto es un desplegable client-side (`button`
    "1 - 50"…"351 - 366") **sin red** → el stream de lotes de T09 no aplica aquí; T31–T33
    pueden emitir un solo lote.
  - **`status`**: `0` Finalizado · `1` Próximamente · `2` En emisión. **`aka`** es mapa por
    locale (`en-us`/`ja-jp`/`es-419`) y **`search` casa título y alias, no sinopsis**.
    `results[].id` es **string** mientras `media.id` es **número**.
  - **Sin cookies, sin CSRF, sin rate-limit observado**; imágenes solo en
    `cdn.animeav1.com` (`covers/`, `screenshots/`, `backdrops/`, `thumbnails/`) y **sin
    exigir Referer** → la meta `no-referrer` de `frontend/index.html` sigue siendo segura.
  - Películas y especiales van a `/media/<slug>/1` (sin esquema de URL propio); fuera de
    rango (`/media/bleach/367`) → **404**.
  - Archivos: `docs/investigacion/animeav1.md` (nuevo), `internal/scraper/animeav1/testdata/`
    (6 fixtures nuevos) y este `PROGRESO.md`. **Cero cambios en Go y en el frontend**.

## Estado de tareas

Leyenda: `[ ]` pendiente · `[~]` en curso · `[x]` hecha y verificada

| Estado | Tarea | Título | Depende de |
|---|---|---|---|
| [x] | T30 | Investigación AnimeAV1 (informe + fixtures) | — |
| [x] | T31 | AnimeAV1: datos de SvelteKit y búsqueda | T30 |
| [x] | T32 | AnimeAV1: detalle y episodios | T31 |
| [x] | T33 | AnimeAV1: fuentes de vídeo | T32 |
| [x] | T34 | Integrar AnimeAV1 en la app (registro, búsqueda combinada, frontend) | T33 |
| [ ] | T35 | E2E de AnimeAV1, retirada de AnimeFLV y release v1.1.0 | T34 |

Orden estricto **T31 → T35** (cada una depende de la anterior): ninguna es paralelizable. **T31–T33 son comprobables sin red** (fixtures de T30); T34 añade smoke con red y verificación con el `.exe`; T35 cierra con E2E, retirada de AnimeFLV y release 1.1.0.

## Desviaciones que afectan a tareas futuras

> Solo las vigentes de cara a **T31–T35**, resumidas; el texto completo y el resto (T01–T29) están en `docs/PROGRESO_v1.0.md` → "Desviaciones que afectan a tareas futuras".

- T03: `ResourceType` en **minúsculas** (`hls`, `mp4`, `iframe`; `models.ResourceHLS/MP4/Iframe`): los comparan `Normalize`/`SelectBestSource` (T16/T17) y el frontend (T27) → **T33** debe emitir esos strings. `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""`.
- T12–T15: `Referer` de un embed = URL del **propio iframe** cuando se verificó que lo exige, o **vacío** si está comprobado que no (MediaFire); las URLs firmadas son temporales y **nunca** se cachean ni se persisten (regla 9). Criterio de referencia para los `embeds` de **T33**.
- T16: `Normalize` corre **dentro** de `Resolver.Resolve` (dedup por recurso **sin query pero con fragmento**; `Type` solo se infiere si viene vacío; preserva `Referer`/`Headers`/`Subtitles`) → si **T33** reutiliza `extractors`, no añada un paso de normalización aparte; si construye las fuentes a mano, respete esas reglas.
- T17: jerarquía de `SelectBestSource` = **HLS (4) > MP4 (3) > iframe (2) > `Type` vacío (1)** (un iframe nunca se auto-elige si hay HLS/MP4), desempate por `Quality` (hoy vacía en todos) y gana la primera de la lista. Consumidor real: `nucleo.fuentes`, que pone la mejor primero → **T34** no lo cambia.
- T19/T20/T29: proxy local: `NewProxy(nil)` (anti-SSRF, sin cookies), `Escuchar()` en `startup`, `Register` **solo** con `Type` `hls|mp4` (iframes pasan intactos) y devuelve la fuente **sin `Referer`/`Headers`** (viven en la sesión del proxy); sin base, la fuente sale sin registrar. `stream.esManifiesto` detecta el manifiesto por el **cuerpo** (`#EXTM3U`), no por la extensión → el HLS de AnimeAV1 (URL sin `.m3u8`) se reescribe **sin tocar `internal/stream`** (T30).
- T19/T30: `Sec-Fetch-Site: same-origin` (obligatoria en los segmentos de `player.zilla-networks.com`) se declara en `models.VideoSource.Headers` —**T33**— y `stream/reenvio.go` ya la inyecta en la sesión; riesgo 1 del informe: si `Register` llegara a limpiar `Headers`, el HLS deja de reproducir → **cerrado en T34** con `TestAnimeAV1CabecerasSoloVivenEnElProxy` (proxy **real** + origen `httptest`: el origen recibe la cabecera y el JSON hacia el frontend no la contiene). Queda el paso final: **reproducir de verdad con el `.exe`** (T35).
- T20: registro/fallback de `nuevoNucleoPorDefecto()` = `jkanime` + `animeflv` (doer compartido), búsqueda JKAnime→AnimeFLV y error **solo** si fallan ambos; `scraper.ErrNoDetails` se traduce en `ErrDetalleNoDisponible` ("Detalle no disponible para este proveedor"). → **hecho en T34**: ahora es `jkanime` + `animeav1` y **búsqueda combinada en paralelo** (JKAnime primero); `animeflv` dejó de registrarse; **T35** retira `animeflv` (**conserva** `scraper.ErrNoDetails` y `ErrDetalleNoDisponible`).
- T20: contrato de eventos: cada lote lleva el `models.EpisodeBatch` completo (`anime_id`, `batch`, `episodes`, `done`, `err`) y `episodes:complete` lleva `{anime_id, error}` (texto en español, vacío = OK); los lotes pueden llegar **antes** de que resuelva `GetAnimeDetails` (el frontend ya los acumula, T24/T26). Los 366 episodios de AnimeAV1 vienen en el detalle (T30) → el **tamaño de lote** lo fija T32 y debe casar con lo que espera el E2E de T35 ("lotes de 50"): anotar la decisión al cerrar T32.
- T21: la caché de detalle es de **10 min** (`nucleo.detalle`); `GetVideoSources` **nunca** se cachea (regla 9) y `noCache` solo fuerza repetir la extracción (el reproductor lo pasa en las recargas).
- T24/T25: el frontend ~~distinguía AnimeFLV por `provider === 'animeflv'` (la tarjeta no navegaba y mostraba "Detalle no disponible para este proveedor")~~ → **eliminado en T34** (toda tarjeta navega; el manejo de errores de detalle sigue siendo genérico). `VideoSourceView` descarta `headers`/`referer` **y desde T34 conserva `language`**; los payloads de eventos van definidos a mano en `frontend/src/api.ts` → **T35** limpia los textos restantes (p. ej. el comentario del smoke `app_integration_test.go`, que sigue nombrando a `animeflv`).
- T31 (paquete `internal/scraper/animeav1`, rama `feature/animeav1`): la vía principal es **`__data.json`** (formato *devalue*), no el `<script>` de hidratación que P7 del informe T30 descartó en su momento. **T32/T33 deben reutilizar lo ya hecho sin reimplementarlo**:
  - `animeav1.PageData(crudo []byte) (map[string]any, error)` entrega la raíz ya hidratada; el nodo de página es el **último** de `nodes` con `type:"data"` **y** `data` array (no usar `nodes[1]`). Tolerancia: nodos `null`/`{"type":"skip"}`/`{"type":"error"}`/sin `data` se saltan; **falla** con cuerpo no-JSON, `type` ≠ `"data"`, `data` sin array y raíz que no sea objeto/`null`.
  - `animeav1.Unflatten(plano []any) (any, error)`: referencias por índice, constantes `-1`…`-6` (`undefined` y hueco → `nil`, NaN/±Inf/−0 numéricos) y ciclos resueltos; **falla** (sin `panic`) ante tipos con etiqueta (`Date`, `Map`, …) y arrays dispersos (`-7`). Si en T32/T33 apareciera un tag, ampliar el desempaquetador en lugar de parsear a mano: hoy los payloads traen **fechas como cadena**.
  - `Search` no pagina (una sola página; `pagination.recordsPerPage` = 20), query vacía → `[]models.Anime{}` **sin petición** y catálogo vacío → slice **no-nulo sin error**. `models.Anime.ID` = **slug** (no el `id` numérico del sitio) y `Provider` = `animeav1`; portada = `https://cdn.animeav1.com/covers/<id>.jpg` (`results[].id` es string).
  - ~~`GetVideoSources` devuelve hoy un error descriptivo `"no implementado todavía: T33"`~~ → **resuelto en T33** (ver el bloque T33 más abajo); nunca fue `scraper.ErrNoDetails`, porque sí hay fuentes. `GetAnimeDetails` está implementado desde T32.
  - Sin cookies/CSRF/rate-limit: `httpx.Get` con la UA por defecto basta (a diferencia de JKAnime). Firmas en `docs/firmas/animeav1.md` → `### T31 — animeav1`.
- T32 (paquete `internal/scraper/animeav1`, rama `feature/animeav1`): detalle y episodios en `detail.go` + `episodes.go`; **T33–T35 deben tener en cuenta**:
  - **Tamaño de lote = 50** (`tamLote`, `episodes.go`): Bleach sale como primera tanda de 50 + **7 lotes** (6×50 + 16) con el último `done=true`, que es lo que espera el E2E de **T35**. Todo el reparto sale de la memoria: **una sola petición** al `__data.json` por anime, sin peticiones por lote (a diferencia de JKAnime). **T35 debe respetarlo.**
  - `StreamAnimeDetails` devuelve los primeros 50 como retorno (no como lote) y lanza la goroutine de los lotes 2..N **solo si `onBatch != nil`**; el `ctx` se comprueba antes de cada emisión (cancelar a mitad deja el resto sin emitir y sin goroutines huérfanas: `TestStreamCancelacionSinFugas`). Sin resto (≤50 episodios o ficha sin episodios) se emite **un único lote `Batch:1, Done:true` vacío**.
  - `GetAnimeDetails` lee `media` del `__data.json` de `/media/<slug>/__data.json` en una petición: `Type = category.slug` (propuesta P7 de T30), `TotalEpisodes = episodesCount` (si es ilegible, `len(episodios)`), `Genres = genres[].name`, `Cover = https://cdn.animeav1.com/covers/<id>.jpg`, miniatura `https://cdn.animeav1.com/screenshots/<mediaId>/<n>.jpg`, `Provider = animeav1`, `ID = slug`.
  - `Status` (comprobado contra el HTML de la ficha): `0` → "Finalizado", `1` → "Próximamente", `2` → "En emisión"; cualquier otro número, fracción, `null` o campo ausente → **"Desconocido"** (no se inventa significado).
  - `Kind`: `category.slug` `pelicula` **con 1 episodio** → `KindMovie` (id `animeav1:<slug>:movie` **y `Number` 0**, contrato de `models.Episode` y de la tarjeta, que pinta "Película" en vez de "Ep. 1"), `especial` → `KindSpecial`, todo lo demás (incluida una película de 2+ episodios, que el sitio sí tiene) → `KindEpisode`. El número del sitio se sigue usando para la miniatura.
  - Título por defecto `"<Título> - <n>"` (patrón JKAnime); película de 1 episodio → solo el título; si el sitio trae `title`, manda.
  - Slug inexistente: el sitio contesta **HTTP 200** con `{"type":"error",…,"status":404}` → se devuelve `&httpx.ScraperError{Operation:"detail", StatusCode:404, Cause: httpx.ErrNotFound}` (mismo error tipado que JKAnime: `errors.Is(err, httpx.ErrNotFound)`).
  - Numeración rara comprobada con `detail_gintama-season-1_data.json`: declara `episodesCount=201` pero la lista trae **200** pares saltándose el 2 (1, 3…201) → la lista manda para los episodios y `episodesCount` para el total.
  - **Desviación (regla 11)**: `TestDetalleEstadoDesconocido` usa un payload devalue **artificial** (`cuerpoSintetico` de `detail_test.go`) porque ningún fixture real trae un status fuera de 0/1/2; respeta el formato real (cadenas sueltas en el array plano, referencias por índice dentro de los objetos) y solo cubre el mapeo de `status`.
  - Firmas en `docs/firmas/animeav1.md` → `### T32 — animeav1`.
- T33 (paquete `internal/scraper/animeav1`, rama `feature/animeav1`): fuentes de vídeo en `sources.go`; **T34–T35 deben tener en cuenta**:
  - **Desviación (regla 11)**: la ficha decía "deduplica por URL"; la clave real es **recurso base + idioma** (`claveFuente`: esquema y host en minúsculas, path, **sin query** y **con fragmento** — criterio de `Normalize`, T16 — más `Language`). Mismo vídeo en SUB y en DUB = **dos** fuentes (cada una con su `Language`); misma URL con el mismo idioma = una; dos URLs que solo difieren en el fragmento **sobreviven ambas**. Cubierto por `TestFuentesDedupConservaElIdioma`, `TestFuentesDedupConservaElFragmento` y `TestFuentesDedupMismaUrlMismoIdioma`.
  - **Desviación**: **no se reutiliza `extractors`** → los embeds que no son el reproductor propio (UPNShare, Voe, Byse, MP4Upload) salen como **iframe sin resolver** con su URL tal cual. Hoy nadie los convierte a MP4 (`SelectBestSource` solo llega a un iframe si no hay HLS/MP4, T17) → si se quieren MP4, resolverlos en **T34/T35** (ahí o ampliando el paquete).
  - Orden de salida **estable**: primero el grupo con HLS propio, después por orden de aparición, con **SUB y DUB adyacentes** dentro de cada servidor (SUB antes que DUB; recorrido de claves fijo en `idiomasEmbeds`, no la iteración del mapa). `bleach/1` → **10 fuentes**: HLS·SUB, HLS·DUB, UPNShare·SUB/DUB, Voe·SUB/DUB, Byse·SUB/DUB, MP4Upload·SUB/DUB; película → **5**, todas SUB.
  - HLS: derivado **sin petición** `/play/<32hex>` → `/m3u8/<32hex>` con `Headers: {"Sec-Fetch-Site":"same-origin"}` (riesgo 1 de T30: si `Register` llegara a limpiar `Headers`, los segmentos dan 403 → **T34/T35** lo cubren con test). `Type: "hls"` en minúsculas (T03) para que `Normalize`/`SelectBestSource` (T16/T17) y el proxy (T19: `Register` solo con `hls|mp4`) lo traten bien. `/play/*` **nunca se pide**.
  - **`models.VideoSource.Language`** (`"SUB"`/`"DUB"`, `omitempty`, entre `server` y `headers`): único cambio en `internal/models`. **T34 debe regenerar `frontend/wailsjs/**`** con `docker compose run --rm app wails generate module` (nunca a mano) y commitear el resultado tal cual.
  - `GetVideoSources` pide `__data.json` del episodio (1 petición, sin cookies/CSRF) con `Operation:"sources"`, ignora `opts.NoCache` (las fuentes nunca se cachean, regla 9), en `KindMovie` pide **siempre** `/1/` y devuelve error descriptivo (nunca un slice vacío) si no hay embeds reproducibles; fuera de rango → `errors.Is(err, httpx.ErrNotFound)` como en el detalle.
  - Firmas: `docs/firmas/animeav1.md` → `### T33 — animeav1`; `docs/firmas/models.md` → `### T33 — internal/models (VideoSource.Language)`.
- T34 (app + frontend, rama `feature/animeav1`): integración de AnimeAV1 en la app; **T35 debe tener en cuenta**:
  - **Búsqueda combinada** en `nucleo.buscar`: JKAnime y AnimeAV1 **en paralelo** con `context.WithTimeout(20 s)`, salida JKAnime primero; error **solo** si fallan todos los registrados (`slog.Warn` por fallo individual, sin URLs firmadas); proveedor no registrado ≠ fallo; sin proveedores → lista vacía sin error. `nuevoNucleoPorDefecto()` registra `jkanime` + `animeav1` (**`animeflv` ya no**, pero el paquete sigue).
  - **Aislamiento de progreso sin migrar nada**: `anime_id` es slug compartido (`bleach` en los dos), `episode_id` lleva el prefijo `<proveedor>:`. El frontend filtra lotes, `episodes` y `progressMap` por ese prefijo (`stores.esDeProveedor`/`episodiosDeProveedor`/`progressToMap(historial, proveedor)`) y toda comprobación de "sigue abierto" compara **proveedor + slug** (si no, abrir `jkanime:bleach` con `animeav1:bleach` cargado reusaba la ficha vieja: bug encontrado y corregido en la verificación visual). `episodes:complete` sigue atribuyéndose solo por slug (payload de Go intacto; seguro porque hay **un único stream activo**).
  - **Etiquetas** en `frontend/src/etiquetas.ts` (puro): `etiquetaTipo`, `etiquetaIdioma`, `etiquetaFuente` (`Servidor · Idioma · Calidad`) y `fuentesVisibles(fuentes, filtro)` con **`filtro = null` hoy** → cambiarlo a `'SUB'`/`'DUB'` activa el filtro sin reescribir nada. `SelectorFuente` emite el índice de la lista **completa** aunque muestre filtrada.
  - **Reanudación al cambiar de fuente**: `elegirFuente` llama `guardarProgreso(true)` antes de mover el índice y `alMetadatos` restaura. Límites: fuente `iframe` (sin metadatos, la posición queda guardada para el próximo vídeo) e episodio `is_watched` (arranca en 0).
  - **`frontend/wailsjs` regenerado** con `wails generate module` (diff = `language?: string` en `models.VideoSource`); no editar a mano.
  - Tests nuevos: `app_test.go` (6 de búsqueda), `app_nucleo_animeav1_test.go` (3: lotes+caché, cancelación, **cabeceras solo en el proxy** con proxy real), `app_nucleo_progreso_test.go` (aislamiento por prefijo) y `frontend/src/etiquetas.test.ts` (20 comprobaciones; el frontend **no tiene runner de tests**, se ejecuta con el esbuild de Vite — comando en su cabecera; `package.json` está fuera de "Puedes tocar").
  - **Desviación**: el comentario de `app_integration_test.go::TestBusquedaRedReal` todavía dice "registro jkanime + animeflv" (el archivo no se puede tocar en T34; el test pasa) → corregirlo en **T35**.
  - Firmas: `docs/firmas/app.md` → `### T34 — búsqueda combinada + AnimeAV1 en la app`.

## Pendientes activos

> Solo lo que una tarea futura debe resolver, con la tarea asignada. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`.

- T30 → T31–T35: informe y 6 fixtures listos (**sin red** para T31–T33). **Abierto**: todavía **no se ha reproducido ningún vídeo de AnimeAV1** end-to-end (riesgo 1 del informe: `Sec-Fetch-Site` en los segmentos y que `Register` no limpie `Headers` antes de tiempo; la parte del test la cerró **T34**) → se cierra con la prueba con el `.exe` de **T35**. La decisión de proveedor ya está aplicada: **T34** registró `animeav1`, dejó de registrar `animeflv` e hizo la búsqueda combinada; **T35** retira AnimeFLV y deja `productVersion` en `1.1.0`.
- T31 → T32/T33: **T31 cerrada y verificada** (paquete nuevo `internal/scraper/animeav1`, 22 tests + 33 subtests en verde con `-race`, smoke `TestSearchRedReal` contra la red real: 11 resultados y el primero `bleach`). Para el detalle/episodio **ya existen** los fixtures `detail_data.json`, `detail_movie_data.json` y `episode_data.json` (T30/T31): no hace falta recapturarlos; `page_data.json` de la página de episodio (`episode_page.html` de T30) basta con el HTML si hiciera falta.
- ~~T32: fijar el **tamaño de lote**~~ → **resuelto**: lote de **50** (`animeav1.tamLote`), anotado en "Desviaciones que afectan a tareas futuras" para que **T35** lo respete.
- T32 → T33: **T32 cerrada y verificada** (nuevos `detail.go` y `episodes.go`, tests offline en verde con `-race`, smoke `TestDetalleRedReal` contra la red real: `bleach`, `total 366`, **366 episodios**, `Finalizado`). T33 parte de los ids ya construidos (`animeav1:<slug>:episode:<n>`, `special:<n>`, `movie`) y de los fixtures `episode_page.html`/`episode_data.json` (T30); aún **no se ha reproducido ningún vídeo de AnimeAV1** (riesgo 1 de T30, pendiente heredado de arriba).
- T33 → T34: **T33 cerrada y verificada** (nuevo `sources.go` + `models.VideoSource.Language`; `sondeo_test.go` temporal borrado; 19 tests / 26 subtests de fuentes en verde con `-race` y `gofmt -l .` sin salida). Smoke `TestFuentesRedReal` contra la red real: **10 fuentes (HLS 2, iframe 8, con idioma 10)** y el manifiesto `…/m3u8/65f3…` responde **200** con `Sec-Fetch-Site: same-origin`, `Content-Type` MPEGURL y cuerpo `#EXTM3U` — solo se pidió el `.m3u8`, **sin segmentos ni vídeo**. Tareas heredadas por **T34**: ~~regenerar `frontend/wailsjs/**`~~ → **hecho** (diff = solo `language?: string`), ~~cubrir con test que `Register` no limpia `Headers` antes de tiempo~~ → **hecho** (`TestAnimeAV1CabecerasSoloVivenEnElProxy`); sigue pendiente para **T35**: comprobar con el `.exe` que el HLS se reproduce de verdad y resolver los **iframes sin resolver** (UPNShare/Voe/Byse/MP4Upload).
- T34 → T35: **T34 cerrada y verificada** (registro `jkanime`+`animeav1` con búsqueda combinada en paralelo; frontend sin reglas de AnimeFLV; aislamiento de progreso por prefijo de `episode_id`; etiquetas `etiquetaTipo`/`etiquetaFuente`/`fuentesVisibles`; reanudación al cambiar de fuente). Verificación completa en verde: `gofmt -l .` sin salida · `go vet ./...` y `go vet -tags integration ./...` · `go build ./...` · `go test -race -count=1 ./...` (todos los paquetes) · `go test -tags integration -run TestBusquedaRedReal` (21 resultados, primero `jkanime`, ambos proveedores 200) · `npm run check` (0 errores / 0 avisos) · `etiquetas.test.ts` (20 comprobaciones) · visual en `:34115` (búsqueda con tarjetas de ambos proveedores, ficha animeav1 con 366 episodios por lotes, selector `HLS · SUB`/`HLS · DUB`… y cambio manual de fuente). **Tareas heredadas por T35**: (1) prueba de reproducción con el `.exe` (en el navegador el proxy queda en `127.0.0.1` del contenedor → `ERR_CONNECTION_REFUSED`, comportamiento documentado, no es un fallo); (2) E2E offline/red de AnimeAV1 esperando lotes de 50; (3) retirada de `animeflv` (y limpieza de su texto en `app_integration_test.go`); (4) iframes sin resolver a MP4/HLS; (5) `productVersion` 1.1.0.
- Heredados de la v1.0 **sin tarea asignada** (detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos" y "Pruebas manuales finales"): título de la ventana (`main.go:20` es el técnico `miapp`), marca de tiempo en `WatchProgress`, borrado de registros del historial, prueba manual del `.exe` de la v1.0 y verificación de los avisos de AnimeFLV/`episodesError` que ningún test provoca. **T35** generará la lista nueva de pruebas manuales de la v1.1.0.
