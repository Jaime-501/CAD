# PROGRESO v1.1.0 — Archivo (T30–T35, AnimeAV1)

> **Archivo de solo lectura** (mover, no borrar). Copia literal del `PROGRESO.md` de la
> **v1.1.0**: investigación T30, desarrollo de **T31–T35** (AnimeAV1), guardada el
> **2026-10-09** al cerrar la v1.1.0 con **T36**.
> El estado **vigente** vive en [`PROGRESO.md`](../PROGRESO.md): T01–T36 hechas y los
> pendientes activos. Aquí quedan el **estado de tareas** T30–T35, las **desviaciones por
> tarea** (texto completo), los **pendientes** de ese ciclo, la **investigación T30**, los
> **fixtures** de AnimeAV1 y las **pruebas manuales** de la v1.1.0.
> Historial de sesiones y decisiones cerradas: [`BITACORA.md`](BITACORA.md) ·
> firmas reales: `firmas/<paquete>.md`.
>
> **Cierre de la v1.1.0**: las **pruebas manuales de la v1.1.0 se hicieron con el `.exe`**
> (`build/bin/miapp.exe` compilado sin `-debug`) en Windows y **salieron bien** — HLS de
> AnimeAV1 (SUB y DUB) con el `Sec-Fetch-Site` inyectado por el proxy, SUB/DUB e iframes,
> búsqueda combinada, mismo slug en ambos proveedores sin mezclar fichas, progreso por
> proveedor, JKAnime intacto y sin consola —. Tras esa prueba, la **release v1.1.0 está
> publicada** en GitHub (tag + `.zip` con SHA256 en las notas).

---

# PROGRESO — AnimeAV1 (T31–T35)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivo de la v1.0**: el estado de **T01–T29** (cierre incluido) está archivado en **`docs/PROGRESO_v1.0.md`** — allí viven sus desviaciones por tarea, sus pendientes y las pruebas manuales del `.exe` de la v1.0. Aquí solo queda lo vigente: **T31–T35** (AnimeAV1). Rama de trabajo: `feature/animeav1`.

## Prompt base por sesión (tal como quedó en la v1.1.0)

```text
Lee AGENTS.md, tareas/00_REGLAS.md, PROGRESO.md, tareas/archivo/T31.md y
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

Antes de las tareas que parsean HTML/JSON real debe existir un **fixture real y sanitizado** (sin cookies ni tokens) en el `testdata/` correspondiente. Un fixture es la respuesta **cruda** del servidor guardada en un archivo, para que los tests corran sin red. Lo puede capturar el humano a mano o el agente con `curl.exe` (el procedimiento "Captura de fixtures" sigue vivo en `PROGRESO.md`). Si no existe el fixture de la tarea, el agente se detiene y lo pide: **no inventa HTML ni selectores**.

- [x] **AnimeAV1 (investigación T30) → T31–T33** — `internal/scraper/animeav1/testdata/` con **6 fixtures crudos** de `https://animeav1.com` capturados con `curl.exe` y UA de navegador: `search.html` (`GET /catalogo?search=bleach`, 200, **61934 B**, 11 tarjetas), `search_empty.html` (`GET /catalogo?search=zzzznoexisteestetitulo`, 200, **22298 B**, 0 tarjetas y sin texto de "sin resultados"), `detail.html` (`GET /media/bleach`, 200, **89187 B**, `media` + **366** episodios embebidos y 50 tarjetas SSR), `detail_movie.html` (`GET /media/bleach-movie-1-memories-of-nobody`, 200, **21324 B**, 1 episodio), `episode_page.html` (`GET /media/bleach/1`, 200, **42441 B**, `episode` + `embeds`/`downloads` en la misma página) y `episode_data.json` (`GET /media/bleach/1/__data.json`, 200, **18447 B**, formato devalue: respaldo, no vía principal). Verificado con probe sobre el HTML **crudo**: `article.group/item.relative.text-body`, `img.aspect-poster`, `h3.line-clamp-2`, `href="/media/<slug>"`, `media:{id:195,…}`, `episodes:[{id,number}]`, `embeds:{DUB:[{server,url}]}`, `button` "1 - 50" y los 101 enlaces `/media/bleach/<n>`. **Sin challenge de Cloudflare, sin `Set-Cookie`, sin límite de tasa** (~40 peticiones → todas 200). **Sanitización: nada que sanitizar** (escaneo de `csrf|token|nonce|set-cookie|authorization|password|secret|signature|expires=|st=|cf_clearance|PHPSESSID|jwplayer` → 0 coincidencias; los únicos `localStorage` son el tema). Informe completo con las 7 preguntas, evidencias, riesgos y propuesta **T31–T35** en **`docs/investigacion/animeav1.md`**.
- [x] **AnimeAV1 fuentes (T33, paso 0)** — 2 fixtures crudos nuevos en `internal/scraper/animeav1/testdata/`, capturados con `curl.exe` + `httpx.DefaultUserAgent`, ambos `200 application/json`: `episode_movie_data.json` (`GET /media/bleach-movie-1-memories-of-nobody/1/__data.json`, **3500 B**, raíz `{media,episode,embeds,downloads}` con `embeds` **solo SUB**: UPNShare, Voe, Byse, MP4Upload y el reproductor propio) y `episode_404_data.json` (`GET /media/bleach/367/__data.json`, **205 B**, HTTP **200** + nodo `{"type":"error","error":{"message":"No encontrado"},"status":404}`). Sanitización: mismo escaneo de tokens → **0 coincidencias**, sin `Set-Cookie`, sin challenge.
  - Formato observado del nodo de página de episodio: `{"media":1,"episode":1167,"embeds":1170,"downloads":1198}`; `embeds` es `{"SUB":[{server,url}×5],"DUB":[{server,url}×5]}` (en la película solo hay clave `SUB`) y `downloads` (5 enlaces de descarga) **se ignora**. El `<id>` del reproductor aparece como `https://player.zilla-networks.com/play/<32 hex>` en ambas claves.
- [x] **AnimeAV1 `__data.json` (T31, paso 0)** — 4 fixtures crudos nuevos en `internal/scraper/animeav1/testdata/`, capturados con `curl.exe` + `httpx.DefaultUserAgent`, todos `200 application/json`: `catalogo_data.json` (`GET /catalogo/__data.json?search=bleach`, **16135 B**), `catalogo_vacio_data.json` (`GET /catalogo/__data.json?search=zzzznoexisteestetitulo`, **5003 B**), `detail_data.json` (`GET /media/bleach/__data.json`, **16764 B**) y `detail_movie_data.json` (`GET /media/bleach-movie-1-memories-of-nobody/__data.json`, **2538 B**).
  - **Equivalencia con los HTML de T30 comprobada** (script de sondeo sobre los 6 archivos): catálogo → **11 resultados** con `id` (string)/`title`/`slug` **idénticos** a los del `<script>` de hidratación de `search.html` y presentes en las 11 tarjetas SSR; vacío → `results:[]` y `totalRecords:0` igual que `search_empty.html` (0 `<article>`); detalle → `media.id=195`, `media.slug="bleach"`, `episodesCount=366` y los **366 pares `{id,number}` idénticos** a `detail.html`; película → `episodesCount=1` y el par `14703:1` idéntico a `detail_movie.html`. **Son equivalentes → la vía `__data.json` sirve.**
  - Formato observado: `{"type":"data","nodes":[null,{…nodo auth…},{nodo de página}]}`; el **nodo de página es el último** con `type:"data"`; dentro de cada nodo, `data` es un array plano devalue donde los números **dentro** de objetos/arrays son referencias por índice (`results:1` → `data[1]`), los números sueltos del array son valores literales y `-1` = `undefined` (visto en `mirrors:-1` de `episode_data.json`). Los 4 fixtures nuevos solo usan `-1` como constante especial: sin tags (`Date`/`Map`/…), sin huecos ni `sparse`.
  - Sanitizado: escaneo de tokens/firmas (mismo patrón que T30) → **0 coincidencias** en los 4 archivos, sin `Set-Cookie`, sin challenge.
- [x] **AnimeAV1 detalle (T32, paso 0)** — 8 fixtures crudos nuevos en `internal/scraper/animeav1/testdata/`, capturados con `curl.exe` + `httpx.DefaultUserAgent`, todos `200 application/json`: `detail_one-piece_data.json` (`GET /media/one-piece/__data.json`, **54096 B**, status 1 y **1180** episodios), `detail_battle-spirits-re-zekkai-no-kuu_data.json` (**1263 B**, status 2), `detail_ajin-ova_data.json` (**1618 B**, status 0, `category.slug=ova`), `detail_one-piece-film-strong-world-episode-0_data.json` (**1715 B**, slug con "episode-0" pero `number:1`), `detail_boku-no-hero-academia-more_data.json` (**1707 B**, `category.slug=especial`), `detail_gintama-season-1_data.json` (**10635 B**, hueco de numeración), `detail_girls-band-cry-movie_data.json` (**1267 B**, película de **2** episodios) y `detail_404_data.json` (**205 B**, `GET /media/<slug-inexistente>/__data.json` → **HTTP 200** con `{"type":"error","error":{…},"status":404}`). Sanitización: mismo escaneo de tokens → **0 coincidencias**, sin `Set-Cookie`, sin challenge.

Nombres de archivo (dentro de `internal/scraper/animeav1/testdata/`): `search.html`, `search_empty.html`, `detail.html`, `detail_movie.html`, `episode_page.html`, `episode_data.json` (T30) · `catalogo_data.json`, `catalogo_vacio_data.json`, `detail_data.json`, `detail_movie_data.json` (T31) · `detail_one-piece_data.json`, `detail_battle-spirits-re-zekkai-no-kuu_data.json`, `detail_ajin-ova_data.json`, `detail_one-piece-film-strong-world-episode-0_data.json`, `detail_boku-no-hero-academia-more_data.json`, `detail_gintama-season-1_data.json`, `detail_girls-band-cry-movie_data.json`, `detail_404_data.json` (T32) · `episode_movie_data.json`, `episode_404_data.json` (T33).

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

## Estado de tareas (v1.1.0)

Leyenda: `[ ]` pendiente · `[~]` en curso · `[x]` hecha y verificada

| Estado | Tarea | Título | Depende de |
|---|---|---|---|
| [x] | T30 | Investigación AnimeAV1 (informe + fixtures) | — |
| [x] | T31 | AnimeAV1: datos de SvelteKit y búsqueda | T30 |
| [x] | T32 | AnimeAV1: detalle y episodios | T31 |
| [x] | T33 | AnimeAV1: fuentes de vídeo | T32 |
| [x] | T34 | Integrar AnimeAV1 en la app (registro, búsqueda combinada, frontend) | T33 |
| [x] | T35 | E2E de AnimeAV1, retirada de AnimeFLV y release v1.1.0 | T34 |

Orden estricto **T31 → T35** (cada una depende de la anterior): ninguna es paralelizable. **T31–T33 son comprobables sin red** (fixtures de T30); T34 añade smoke con red y verificación con el `.exe`; T35 cierra con E2E, retirada de AnimeFLV y release 1.1.0.

## Desviaciones que afectan a tareas futuras (texto completo de T30–T35)

> Tal como quedaron al cerrar T35. Las de T01–T29 están en `docs/PROGRESO_v1.0.md` → "Desviaciones que afectan a tareas futuras".

- T03: `ResourceType` en **minúsculas** (`hls`, `mp4`, `iframe`; `models.ResourceHLS/MP4/Iframe`): los comparan `Normalize`/`SelectBestSource` (T16/T17) y el frontend (T27) → **T33** debe emitir esos strings. `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""`.
- T12–T15: `Referer` de un embed = URL del **propio iframe** cuando se verificó que lo exige, o **vacío** si está comprobado que no (MediaFire); las URLs firmadas son temporales y **nunca** se cachean ni se persisten (regla 9). Criterio de referencia para los `embeds` de **T33**.
- T16: `Normalize` corre **dentro** de `Resolver.Resolve` (dedup por recurso **sin query pero con fragmento**; `Type` solo se infiere si viene vacío; preserva `Referer`/`Headers`/`Subtitles`) → si **T33** reutiliza `extractors`, no añada un paso de normalización aparte; si construye las fuentes a mano, respete esas reglas.
- T17: jerarquía de `SelectBestSource` = **HLS (4) > MP4 (3) > iframe (2) > `Type` vacío (1)** (un iframe nunca se auto-elige si hay HLS/MP4), desempate por `Quality` (hoy vacía en todos) y gana la primera de la lista. Consumidor real: `nucleo.fuentes`, que pone la mejor primero → **T34** no lo cambia.
- T19/T20/T29: proxy local: `NewProxy(nil)` (anti-SSRF, sin cookies), `Escuchar()` en `startup`, `Register` **solo** con `Type` `hls|mp4` (iframes pasan intactos) y devuelve la fuente **sin `Referer`/`Headers`** (viven en la sesión del proxy); sin base, la fuente sale sin registrar. `stream.esManifiesto` detecta el manifiesto por el **cuerpo** (`#EXTM3U`), no por la extensión → el HLS de AnimeAV1 (URL sin `.m3u8`) se reescribe **sin tocar `internal/stream`** (T30).
- T19/T30: `Sec-Fetch-Site: same-origin` (obligatoria en los segmentos de `player.zilla-networks.com`) se declara en `models.VideoSource.Headers` —**T33**— y `stream/reenvio.go` ya la inyecta en la sesión; riesgo 1 del informe: si `Register` llegara a limpiar `Headers`, el HLS deja de reproducir → **cerrado en T34** con `TestAnimeAV1CabecerasSoloVivenEnElProxy` (proxy **real** + origen `httptest`: el origen recibe la cabecera y el JSON hacia el frontend no la contiene). Queda el paso final: **reproducir de verdad con el `.exe`** (T35).
- T20: registro/fallback de `nuevoNucleoPorDefecto()` = `jkanime` + `animeflv` (doer compartido), búsqueda JKAnime→AnimeFLV y error **solo** si fallan ambos; `scraper.ErrNoDetails` se traduce en `ErrDetalleNoDisponible` ("Detalle no disponible para este proveedor"). → **hecho en T34**: ahora es `jkanime` + `animeav1` y **búsqueda combinada en paralelo** (JKAnime primero); `animeflv` dejó de registrarse; **T35 retira `animeflv`** (**conserva** `scraper.ErrNoDetails` y `ErrDetalleNoDisponible`). *(Hecho en T35.)*
- T20: contrato de eventos: cada lote lleva el `models.EpisodeBatch` completo (`anime_id`, `batch`, `episodes`, `done`, `err`) y `episodes:complete` lleva `{anime_id, error}` (texto en español, vacío = OK); los lotes pueden llegar **antes** de que resuelva `GetAnimeDetails` (el frontend ya los acumula, T24/T26). Los 366 episodios de AnimeAV1 vienen en el detalle (T30) → el **tamaño de lote** lo fija T32 y debe casar con lo que espera el E2E de T35 ("lotes de 50"): anotar la decisión al cerrar T32.
- T21: la caché de detalle es de **10 min** (`nucleo.detalle`); `GetVideoSources` **nunca** se cachea (regla 9) y `noCache` solo fuerza repetir la extracción (el reproductor lo pasa en las recargas).
- T24/T25: el frontend ~~distinguía AnimeFLV por `provider === 'animeflv'` (la tarjeta no navegaba y mostraba "Detalle no disponible para este proveedor")~~ → **eliminado en T34** (toda tarjeta navega; el manejo de errores de detalle sigue siendo genérico). `VideoSourceView` descarta `headers`/`referer` **y desde T34 conserva `language`**; los payloads de eventos van definidos a mano en `frontend/src/api.ts` → **T35** limpia los textos restantes (p. ej. el comentario del smoke `app_integration_test.go`, que sigue nombrando a `animeflv`). *(Hecho en T35.)*
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
  - **Desviación**: el comentario de `app_integration_test.go::TestBusquedaRedReal` todavía dice "registro jkanime + animeflv" (el archivo no se puede tocar en T34; el test pasa) → corregirlo en **T35**. *(Hecho en T35.)*
  - Firmas: `docs/firmas/app.md` → `### T34 — búsqueda combinada + AnimeAV1 en la app`.
- T35 (raíz + docs, rama `feature/animeav1`): E2E, retirada de AnimeFLV y release 1.1.0; lo que una tarea futura debe saber:
  - **E2E offline de AnimeAV1** (`TestE2EAnimeAV1FlujoCompletoOffline`, en `app_e2e_test.go`): un único `httptest.Server` (`sitioAV1`) sirve el catálogo/ficha/episodio de los fixtures reales del paquete + la búsqueda de JKAnime (búsqueda combinada), y un reproductor artificial con el contrato de T30 (segmento **403 sin `Sec-Fetch-Site`**, 206 con `Range: bytes=0-0`). El proxy del test recibe un doer con **`transporteJugador`**: solo admite `player.zilla-networks.com` (traducido al sitio falso) y el propio sitio; cualquier otro host se corta y hace fallar el test. Exige registro EXACTO de peticiones (9: búsqueda, excepciones, catálogo, ficha, episodio, manifiesto, 3× segmento) y que el segmento llegue al origen CON la cabecera (206) mientras el directo sin ella da 403 — **comprobado que muerde** (cambio temporal en `stream/reenvio.go` → el test falla con 403 en el origen; revertido).
  - **E2E real** (`TestE2EAnimeAV1RedReal`, `app_e2e_integration_test.go`, tag `integration`): búsqueda combinada → detalle por lotes → fuentes → manifiesto real por el proxy (24830 B) + **un segmento con `Range: bytes=0-0` → 206** (`Content-Range: bytes 0-0/4484017`); sin vídeo entero ni URLs firmadas en la salida.
  - **Sobre la aserción del segmento reescrito**: el manifiesto se pide a `https://player.zilla-networks.com/m3u8/<id>` y su segmento va con ruta absoluta `/segmento1.ts`; la URI reescrita apunta a `http://<sitio-falso>/segmento1.ts` porque el proxy resuelve contra `resp.Request.URL` (reenvio.go), que ya refleja la traducción del `transporteJugador`. La expectativa correcta es `sitio.srv.URL + "/segmento1.ts"`, no el host del jugador (la sospecha de la ficha no se confirmó: el test pasaba ya).
  - **AnimeFLV retirado**: `internal/scraper/animeflv/` y `docs/firmas/animeflv.md` borrados (`git ls-files | Select-String animeflv` → vacío). **Conservados**: `scraper.ErrNoDetails` y `ErrDetalleNoDisponible` (genéricos); los tests que usan el nombre `"animeflv"` como provider falso (`app_test.go`, `app_nucleo_progreso_test.go`, `registry_test.go`, `ids_test.go`, `cache_test.go`) siguen siendo válidos (el nombre es un string arbitrario). Comentario de `app_integration_test.go` corregido; meta `no-referrer` de `index.html` mantenida con el comentario actualizado.
  - `wails.json` gana `info.productVersion = "1.1.0"` (sección `info`, según el esquema de Wails v2; antes no había sección `info`). `CHANGELOG.md` nuevo.
  - Release compilada: `docker compose run --rm app wails build -platform windows/amd64` (sin `-debug`) → **`build/bin/miapp.exe`, 13.221.376 bytes** (~12,6 MiB). **Sin tag ni release de GitHub** (los hace el humano tras la prueba manual).
  - **Pendientes heredados sin tarea asignada** (sin cambio): iframes sin resolver a MP4/HLS, título de la ventana `miapp`, marca de tiempo en `WatchProgress`, borrado de registros del historial. Queda **el único paso no automatizable**: probar el `.exe` en Windows (lista de pruebas manuales más abajo).

## Pruebas manuales de la v1.1.0 (con el `.exe` en Windows)

> Compilar con `docker compose run --rm app wails build -platform windows/amd64` (release, sin `-debug`) y ejecutar `build/bin/miapp.exe` en Windows. En Docker el vídeo HLS/MP4 da `ERR_CONNECTION_REFUSED` (el proxy es `127.0.0.1` del contenedor): eso NO es un fallo.
>
> **Resultado (2026-10-09): las 7 pruebas se ejecutaron con el `.exe` en Windows y salieron
> bien.** Tras ellas, la **release v1.1.0 está publicada** en GitHub.

1. **Reproducción HLS de AnimeAV1 (la clave):** buscar `bleach`, abrir el resultado de animeav1, abrir el episodio 1 y verificar que el HLS (`HLS · SUB`) reproduce de verdad (segmentos con `Sec-Fetch-Site` vía proxy). Probar también `HLS · DUB`.
2. **SUB/DUB e iframes:** en el selector de fuentes comprobar las etiquetas `Servidor · Idioma`; abrir un iframe (p. ej. Voe) y confirmar el aviso de "sin progreso".
3. **Búsqueda combinada con un proveedor lento/caído:** buscar un título y confirmar que llegan resultados de jkanime (primero) y animeav1; si uno falla, el otro sigue respondiendo.
4. **Abrir rápido el mismo slug en ambos proveedores:** abrir `bleach` de jkanime y a continuación `bleach` de animeav1 (y viceversa): la ficha, los episodios y el selector deben corresponder al proveedor elegido (no la ficha vieja).
5. **Progreso y "Continuar" sin mezclar proveedores:** ver un poco un episodio de animeav1, volver, continuar; repetir con jkanime; el progreso de cada uno no debe contaminar al otro (los ids llevan prefijo `<proveedor>:`).
6. **JKAnime sigue funcionando:** búsqueda, detalle por lotes y reproducción HLS/MP4 del proveedor principal, igual que en la v1.0.
7. **Sin consola:** el `.exe` de release no debe abrir ventana de consola.

## Pendientes al cerrar la v1.1.0 (cerrados o heredados, texto completo)

> Copia literal de la sección "Pendientes activos" del `PROGRESO.md` de la v1.1.0 al cerrar T35.
> Los que siguen vivos (sin tarea asignada) se replican —resumidos— en `PROGRESO.md`.

- T30 → T31–T35: informe y fixtures listos. ~~Todavía no se ha reproducido ningún vídeo de AnimeAV1 end-to-end~~ → **cerrado**: los E2E de T35 cubren el flujo offline y el real (manifiesto + segmento con `Range` → 206 por el proxy). **Único paso restante**: la prueba manual con el `.exe` en Windows (lista arriba), porque en Docker el proxy (`127.0.0.1` del contenedor) no es alcanzable desde el navegador del host. La decisión de proveedor está aplicada y AnimeFLV está retirado. → **cerrado en T36**: la prueba manual con el `.exe` salió bien (ver cabecera de este archivo).
- ~~T31 → T32/T33~~ · ~~T32 → T33~~ · ~~T33 → T34~~ · ~~T34 → T35~~: cerradas y verificadas (detalle en "Desviaciones que afectan a tareas futuras" y en `docs/BITACORA.md`).
- **T35 cerrada y verificada**: E2E offline + real de AnimeAV1 en verde (incluye la demostración de que el proxy inyecta `Sec-Fetch-Site`: segmento vía proxy → 206, directo sin cabecera → 403, y el test muerde — cambio temporal en `stream/reenvio.go` lo confirmó y se revirtió); `internal/scraper/animeflv/` y `docs/firmas/animeflv.md` borrados (`git ls-files | Select-String animeflv` → vacío); `scraper.ErrNoDetails` conservado; docs (README, DESARROLLO, AGENTS) al estado final; `wails.json` con `info.productVersion = "1.1.0"` y `CHANGELOG.md` nuevo; release compilada `build/bin/miapp.exe` (**13.221.376 bytes**, sin `-debug`). Verificación completa en verde: `gofmt -l .` sin salida · `go vet ./...` y `go vet -tags integration ./...` · `go build ./...` · `go test -race -count=1 ./...` (11 paquetes) · `go test -tags integration ./... -run E2E -v` (los 4 E2E: JKAnime offline/real y AnimeAV1 offline/real) · `npm run check` (0 errores / 0 avisos). **Tareas heredadas sin asignar**: (1) prueba manual del `.exe` (lista de pruebas manuales de la v1.1.0 arriba; el humano hace el tag y la release de GitHub tras ella); (2) iframes sin resolver a MP4/HLS; (3) pendientes de la v1.0 sin cambio (título de la ventana, marca de tiempo en `WatchProgress`, borrado de registros del historial).
- Heredados de la v1.0 **sin tarea asignada** (detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos" y "Pruebas manuales finales"): título de la ventana (`main.go:20` es el técnico `miapp`), marca de tiempo en `WatchProgress`, borrado de registros del historial, prueba manual del `.exe` de la v1.0 y verificación de los avisos de `episodesError` que ningún test provoca.
