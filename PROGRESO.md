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
- [ ] `episode_movie_data.json` (`GET /media/bleach-movie-1-memories-of-nobody/1/__data.json`) → lo captura **T33** (paso 0); hoy no existe.

Nombres de archivo (dentro de `internal/scraper/animeav1/testdata/`): `search.html`, `search_empty.html`, `detail.html`, `detail_movie.html`, `episode_page.html`, `episode_data.json` (+ `episode_movie_data.json`, T33).

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
| [ ] | T31 | AnimeAV1: datos de SvelteKit y búsqueda | T30 |
| [ ] | T32 | AnimeAV1: detalle y episodios | T31 |
| [ ] | T33 | AnimeAV1: fuentes de vídeo | T32 |
| [ ] | T34 | Integrar AnimeAV1 en la app (registro, búsqueda combinada, frontend) | T33 |
| [ ] | T35 | E2E de AnimeAV1, retirada de AnimeFLV y release v1.1.0 | T34 |

Orden estricto **T31 → T35** (cada una depende de la anterior): ninguna es paralelizable. **T31–T33 son comprobables sin red** (fixtures de T30); T34 añade smoke con red y verificación con el `.exe`; T35 cierra con E2E, retirada de AnimeFLV y release 1.1.0.

## Desviaciones que afectan a tareas futuras

> Solo las vigentes de cara a **T31–T35**, resumidas; el texto completo y el resto (T01–T29) están en `docs/PROGRESO_v1.0.md` → "Desviaciones que afectan a tareas futuras".

- T03: `ResourceType` en **minúsculas** (`hls`, `mp4`, `iframe`; `models.ResourceHLS/MP4/Iframe`): los comparan `Normalize`/`SelectBestSource` (T16/T17) y el frontend (T27) → **T33** debe emitir esos strings. `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""`.
- T12–T15: `Referer` de un embed = URL del **propio iframe** cuando se verificó que lo exige, o **vacío** si está comprobado que no (MediaFire); las URLs firmadas son temporales y **nunca** se cachean ni se persisten (regla 9). Criterio de referencia para los `embeds` de **T33**.
- T16: `Normalize` corre **dentro** de `Resolver.Resolve` (dedup por recurso **sin query pero con fragmento**; `Type` solo se infiere si viene vacío; preserva `Referer`/`Headers`/`Subtitles`) → si **T33** reutiliza `extractors`, no añada un paso de normalización aparte; si construye las fuentes a mano, respete esas reglas.
- T17: jerarquía de `SelectBestSource` = **HLS (4) > MP4 (3) > iframe (2) > `Type` vacío (1)** (un iframe nunca se auto-elige si hay HLS/MP4), desempate por `Quality` (hoy vacía en todos) y gana la primera de la lista. Consumidor real: `nucleo.fuentes`, que pone la mejor primero → **T34** no lo cambia.
- T19/T20/T29: proxy local: `NewProxy(nil)` (anti-SSRF, sin cookies), `Escuchar()` en `startup`, `Register` **solo** con `Type` `hls|mp4` (iframes pasan intactos) y devuelve la fuente **sin `Referer`/`Headers`** (viven en la sesión del proxy); sin base, la fuente sale sin registrar. `stream.esManifiesto` detecta el manifiesto por el **cuerpo** (`#EXTM3U`), no por la extensión → el HLS de AnimeAV1 (URL sin `.m3u8`) se reescribe **sin tocar `internal/stream`** (T30).
- T19/T30: `Sec-Fetch-Site: same-origin` (obligatoria en los segmentos de `player.zilla-networks.com`) se declara en `models.VideoSource.Headers` —**T33**— y `stream/reenvio.go` ya la inyecta en la sesión; riesgo 1 del informe: si `Register` llegara a limpiar `Headers`, el HLS deja de reproducir → **T34/T35** lo cubren con test (E2E offline con segmentos que devuelven 403 sin la cabecera).
- T20: registro/fallback de `nuevoNucleoPorDefecto()` = `jkanime` + `animeflv` (doer compartido), búsqueda JKAnime→AnimeFLV y error **solo** si fallan ambos; `scraper.ErrNoDetails` se traduce en `ErrDetalleNoDisponible` ("Detalle no disponible para este proveedor"). → **T34** registra `animeav1`, deja de registrar `animeflv` y pasa a búsqueda **combinada en paralelo**; **T35** retira `animeflv` (**conserva** `scraper.ErrNoDetails`).
- T20: contrato de eventos: cada lote lleva el `models.EpisodeBatch` completo (`anime_id`, `batch`, `episodes`, `done`, `err`) y `episodes:complete` lleva `{anime_id, error}` (texto en español, vacío = OK); los lotes pueden llegar **antes** de que resuelva `GetAnimeDetails` (el frontend ya los acumula, T24/T26). Los 366 episodios de AnimeAV1 vienen en el detalle (T30) → el **tamaño de lote** lo fija T32 y debe casar con lo que espera el E2E de T35 ("lotes de 50"): anotar la decisión al cerrar T32.
- T21: la caché de detalle es de **10 min** (`nucleo.detalle`); `GetVideoSources` **nunca** se cachea (regla 9) y `noCache` solo fuerza repetir la extracción (el reproductor lo pasa en las recargas).
- T24/T25: el frontend distingue AnimeFLV por `provider === 'animeflv'` (la tarjeta no navega y muestra "Detalle no disponible para este proveedor") y `VideoSourceView` descarta `headers`/`referer`; los payloads de eventos van definidos a mano en `frontend/src/api.ts` → **T34** elimina esa regla especial (todos los proveedores tendrán detalle) **sin** tocar `frontend/wailsjs/**`, y T35 limpia los textos restantes.

## Pendientes activos

> Solo lo que una tarea futura debe resolver, con la tarea asignada. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`.

- T30 → T31–T35: informe y 6 fixtures listos (**sin red** para T31–T33). **Abierto**: todavía **no se ha reproducido ningún vídeo de AnimeAV1** end-to-end (riesgo 1 del informe: `Sec-Fetch-Site` en los segmentos y que `Register` no limpie `Headers` antes de tiempo) → se cierra con el smoke de red y la prueba con el `.exe` de **T34/T35**. La decisión de proveedor ya está en los archivos de tarea: **T34** registra `animeav1` y deja de registrar `animeflv` (búsqueda combinada), **T35** retira AnimeFLV y deja `productVersion` en `1.1.0`; si al llegar allí se reabre, anotarlo aquí.
- T32: fijar el **tamaño de lote** de los episodios (T30 sugiere un solo lote con `done=true`; el E2E de T35 espera "lotes de 50") y anotarlo en este archivo para que T35 lo respete.
- Heredados de la v1.0 **sin tarea asignada** (detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos" y "Pruebas manuales finales"): título de la ventana (`main.go:20` es el técnico `miapp`), marca de tiempo en `WatchProgress`, borrado de registros del historial, prueba manual del `.exe` de la v1.0 y verificación de los avisos de AnimeFLV/`episodesError` que ningún test provoca. **T35** generará la lista nueva de pruebas manuales de la v1.1.0.
