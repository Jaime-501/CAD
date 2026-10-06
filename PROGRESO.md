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
- [ ] Página con `var servers` → T14
- [ ] Página de MediaFire → T15

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
| [ ] | T14 | Extractor `var servers` | T11 | Agrupable T12–T15 |
| [ ] | T15 | Resolver MediaFire | T11 | Agrupable T12–T15 |
| [ ] | T16 | Normalizador de VideoSource | T12–T15 | Con T17 |
| [ ] | T17 | Selector de mejor fuente + GetVideoSources | T16 | Con T16 |
| [ ] | T18 | AnimeFLV — búsqueda con fallback | T02, T04 | Sola |
| [ ] | T19 | Proxy local de streaming | T02, T03 | Sola (pesada) |
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
- T04: **`AnimeFLV` no cumple `Provider`** (su `Search` no lleva `ctx` y `GetAnimeDetails` devuelve `models.Anime` en vez de `models.AnimeDetail`); `app.go` sigue llamando a su `Search(query)`. Fuera de "Puedes tocar" en T04 → lo adapta **T18**.
- T11: los extractores reciben `extractors.Page`, no `jkanime.EpisodePage` (regla de dependencias); convertir `EpisodePage → Page` es de **T17**.
- T12: `VideoSource.Referer` = URL del **propio iframe** del player (decisión de la captura con Playwright: el `.m3u8` del CDN responde 200 con ese referer). **T19 inyecta el Referer en el proxy local**: coordinar allí para no duplicar la cabecera.
- T13: `VideoSource.Referer` = URL del **propio iframe** (mismo criterio que T12). Ojo: la respuesta de `jk.php` declara `<meta name="referrer" content="never"/>`, así que el navegador no enviaría referer al `stream/jkmedia/…`; si **T16/T19** prefieren otro criterio (o vacío), decidirlo allí.

## Pendientes activos

> Solo lo que una tarea futura debe resolver, con la tarea asignada. Los cerrados/históricos están en `docs/BITACORA.md`.

- T01: `app.go` todavía no llama a `logx.New(...)` ni usa `logx.Request` → **T20**.
- T02: `internal/scraper/jkanime` ya consume `httpx` (desde T06/T07), pero `internal/scraper/animeflv.go` sigue con su propio `http.Get` → migrarlo en **T18**.
- T03: los modelos ya los usan T08 (detalle) y T09 (episodios/lotes); siguen sin consumidor en proxy, historial y serialización a Wails → **T19, T20, T22**.
- T04: nadie registra providers en el `Registry` (el montaje es `jkanime` + `animeflv`) → **T20**.
- T04: `SourceOpts.NoCache` no lo consume nadie → **T17, T27**.
- T05: `ParseEpisodeID` solo lo usan sus tests (`NewEpisodeID` ya lo usa T09) → **T20, T26**.
- T06: el aviso `slog.Warn` va al logger por defecto de `slog`, no al de `logx`; para unificar, instalar `slog.SetDefault(logx.New(dev))` → **T20/T29**.
- T07: la primera búsqueda descarga el mapa de excepciones de GitHub (máx. 5 s, cacheado); `ExceptionResolver` compartido o precarga al arrancar → **T20**.
- T07: `GetVideoSources` devuelve `ErrNotImplemented` a propósito → implementarlo y borrar esa rama en **T17/T20**.
- T09: `StreamAnimeDetails` todavía no tiene consumidor (lo usará T20 con `runtime.EventsEmit` por lote; `GetAnimeDetails` ya lo usa vía `jkanime.go`) → **T20**. Sin consumidor, el único test de cancelación real es `TestStreamCancelacionSinFugas`.
- T09: `pausaPaginas` es constante de 300 ms compartida por stream y `GetAnimeDetails`; otro ritmo ⇒ inyectarla → **T20**.
- T09: si falla una página de fondo, el lote final lleva `Err` y los episodios ya emitidos **no** se revocan; `GetAnimeDetails` fallando a mitad devuelve detalle vacío y **sin** episodios parciales → si T20 quiere parciales, es cambio de contrato en **T20**.
- T10: `fetchEpisodePage` todavía no lo llama nadie fuera de sus tests (T09 encadena detalle + AJAX, no `/slug/<n>/`) → **T17**.
- T10: `EpisodePage.Headers` es `http.Header` (mapa `string → []string`) y `EpisodePage` no se serializa a Wails hoy → si se expone, **T20/T27**.
- T10: la sesión depende de que el doer lleve cookiejar (`httpx.NewClient` lo pone): sin jar, el `_token` no se envía y la web devuelve 419 (T09 refresca una vez y, si sigue sin jar, acaba con el 419 original en el lote final) → **T17/T20**.
- T11: nadie construye todavía el `Resolver`: falta la cadena de extractores reales (T12–T15) y la conversión `EpisodePage → Page`, que es de **T17**; hasta entonces `Resolve` solo se prueba con falsos.
- T11: `Resolve` agrega **todas** las fuentes en orden canónico de extractor, sin ordenar por calidad, sin deduplicar y sin descartar por tipo: eso es **T16/T17**. Si alguna tarea quiere "parar en la primera usable", es cambio de política y debe registrarse aquí.
- T11: `Resolve` no valida el esquema de las URLs ni completa `Headers`/`Referer`/`Subtitles` → **T16**.
- T12: `JKPlayerExtractor` todavía no lo construye nadie (la cadena se arma con `New(...)`) → **T17**.
- T12: `Type` puede quedar `""` si ni la URL ni el cuerpo del player dan señal (hoy siempre hay `.m3u8`) → tratar el tipo vacío en **T16/T17**.
- T12: cada iframe distinto dispara su propia petición (hasta 4 por página) → deduplicar en **T16/T17** si molesta.
- T13: `JKPHPExtractor` todavía no lo construye nadie → **T17**.
- T13: la descarga de la página del player está duplicada con `JKPlayerExtractor.fuentesDePlayer` (mismo límite y mismos helpers) → unificar en **T29** en una sola función parametrizada por `Server`/`Extractor`.
