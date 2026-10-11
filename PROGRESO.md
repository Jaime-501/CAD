# PROGRESO — Comfort Anime Desktop (T01–T36 hechas)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivos históricos**: el estado de **T01–T29 + T30** (v1.0) está en **`docs/PROGRESO_v1.0.md`** y el de **T30–T35** (AnimeAV1, v1.1.0, con sus desviaciones por tarea, fixtures y pruebas manuales) en **`docs/PROGRESO_v1.1.md`**. Aquí solo queda lo vigente. Estado del producto: **v1.1.0 publicada** (probada con el `.exe` en Windows).

## Protocolo de sesión

1. Leer `AGENTS.md`, `tareas/00_REGLAS.md`, este archivo, `docs/firmas/<paquetes de los que depende la tarea>` y el archivo de la tarea (`tareas/Txx.md`).
2. Ejecutar **solo** la tarea indicada. Respetar "Puedes tocar" / "NO debes tocar".
3. Correr la comprobación de la tarea. Después, el humano la repite por su cuenta (`gofmt -l .`, `go vet ./...`, `go test -race ./...` y `npm run check` en frontend): no confiar solo en el reporte del agente.
4. Actualizar este archivo: marcar la tarea, anotar desviaciones y pendientes; las firmas nuevas van en `docs/firmas/<paquete>.md` (formato `### Txx — <paquete>`, ej. `docs/firmas/animeav1.md`).
5. Detenerse. No empezar la siguiente tarea.

> **Aviso de entorno (T36):** en este repositorio no hay `.gitattributes` y `core.autocrlf=true`: un merge/checkout hecho **desde Windows** puede dejar los `.go` tocados con fin de línea **CRLF** en el worktree, y entonces `gofmt -l .` (en el contenedor Linux) los marca a todos aunque el contenido esté bien formateado. La solución es normalizarlos a LF (quitar solo el `\r` final; git no registra ningún cambio) — decisión del humano en T36. `git status` puede seguir mostrándolos como "M" con diff vacío: es el fantasma de autocrlf (mismo caso que `frontend/wailsjs/go/models.ts`).

## Prompt base por sesión

```text
Lee AGENTS.md, tareas/00_REGLAS.md, PROGRESO.md, tareas/<Txx>.md y
docs/firmas/<paquetes de los que depende la tarea>.
Ejecuta SOLO la tarea <Txx>. Respeta "Puedes tocar" y "NO debes tocar".
Si necesitas tocar un archivo fuera de la lista, detente y repórtalo.
Al terminar: corre la comprobación, actualiza PROGRESO.md (estado, desviaciones,
pendientes) y docs/firmas/<paquete>.md si creaste firmas, y detente.
No empieces la siguiente tarea.
```

**Qué firmas leer:** solo las de los paquetes de los que depende tu tarea (`docs/firmas/<paquete>.md`; la lista por paquete está en `tareas/00_REGLAS.md` → "Documentación y estado"). La tabla por tarea de T31–T35 quedó archivada en `docs/PROGRESO_v1.1.md`.

## Prerrequisitos: fixtures reales (humano o agente con curl)

Antes de las tareas que parsean HTML/JSON real debe existir un **fixture real y sanitizado** (sin cookies ni tokens) en el `testdata/` correspondiente. Un fixture es la respuesta **cruda** del servidor guardada en un archivo, para que los tests corran sin red. Lo puede capturar el humano a mano o el agente con `curl.exe` (ver "Captura de fixtures" más abajo). Si no existe el fixture de la tarea, el agente se detiene y lo pide: **no inventa HTML ni selectores**.

- Los fixtures ya capturados están inventariados en los archivos históricos: JKAnime (v1.0) en `docs/PROGRESO_v1.0.md`, AnimeAV1 (T30–T33, 20 fixtures) en `docs/PROGRESO_v1.1.md`.

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

## Estado de tareas

Leyenda: `[ ]` pendiente · `[~]` en curso · `[x]` hecha y verificada

| Estado | Tarea | Título | Depende de |
|---|---|---|---|
| [x] | T01 | Logging y observabilidad base | — |
| [x] | T02 | Infraestructura HTTP, errores y reintentos | T01 |
| [x] | T03 | Modelos de dominio | — |
| [x] | T04 | Interfaz Provider y registro | T03 |
| [x] | T05 | Referencia de episodio y películas | T03 |
| [x] | T06 | JKAnime — excepciones | T02 |
| [x] | T07 | JKAnime — búsqueda | T02–T06 |
| [x] | T08 | JKAnime — detalle y metadatos | T07 |
| [x] | T09 | JKAnime — episodios AJAX con streaming | T08 |
| [x] | T10 | JKAnime — fetch de página de episodio | T02 |
| [x] | T11 | Arquitectura de extractores | T03, T10 |
| [x] | T12 | Extractor JKPlayer (softsubs) | T11 |
| [x] | T13 | Extractor JKPHP | T11 |
| [x] | T14 | Extractor `var servers` | T11 |
| [x] | T15 | Resolver MediaFire | T11 |
| [x] | T16 | Normalizador de VideoSource | T12–T15 |
| [x] | T17 | Selector de mejor fuente + GetVideoSources | T16 |
| [x] | T18 | AnimeFLV — búsqueda con fallback (histórico: retirado en T35) | T02, T04 |
| [x] | T19 | Proxy local de streaming | T02, T03 |
| [x] | T20 | Integración app.go y eventos Wails | T09, T17, T19 |
| [x] | T21 | Caché local | T20 |
| [x] | T22 | Persistencia del historial | T03 |
| [x] | T23 | Regenerar bindings Wails | T20–T22 |
| [x] | T24 | Estado Svelte | T23 |
| [x] | T25 | UI de búsqueda y navegación | T23 |
| [x] | T26 | Vista de detalle y episodios | T24, T25 |
| [x] | T27 | Reproductor multi-recurso | T19, T24 |
| [x] | T28 | E2E (offline + real) | T17, T20 |
| [x] | T29 | Ajustes finales y refactor | todas |
| [x] | T30 | Investigación AnimeAV1 (informe + fixtures) | — |
| [x] | T31 | AnimeAV1: datos de SvelteKit y búsqueda | T30 |
| [x] | T32 | AnimeAV1: detalle y episodios | T31 |
| [x] | T33 | AnimeAV1: fuentes de vídeo | T32 |
| [x] | T34 | Integrar AnimeAV1 en la app (registro, búsqueda combinada, frontend) | T33 |
| [x] | T35 | E2E de AnimeAV1, retirada de AnimeFLV y release v1.1.0 | T34 |
| [x] | T36 | Cierre de la v1.1.0 y limpieza de documentación y comentarios | T35 |
| [x] | T37 | Preparación para Android (refactor sin cambio de comportamiento) 🔒 | T36 |
| [x] | T38 | Cierre documental de T37 | T37 |
| [x] | T39 | Investigación del Home (informe + fixtures) | — |
| [x] | T40 | `models`: progreso ampliado, Favorite, HomeSection, ResumeInfo, NextEpisode | — |
| [x] | T41 | `db`: historial con fecha, listado reciente y borrados 🔒 | T40 |
| [x] | T42 | `db`: favoritos 🔒 | T40 |
| [x] | T43 | `HomeProvider` + Home de JKAnime | T39, T40 |
| [x] | T44 | Home de AnimeAV1 | T39, T43 |
| [x] | T45 | `Nucleo` + `app.go`: Home, favoritos, borrados 🔒 | T41–T44 |
| [ ] | T46 | `Nucleo.Retomar` y `SiguienteEpisodio` 🔒 | T45 |
| [ ] | T47 | Regenerar bindings + `api.ts` | T46 |
| [ ] | T48 | UI: shell con barra lateral | T47 |
| [ ] | T49 | UI: barra lateral con historial por proveedor | T48 |
| [ ] | T50 | UI: retomar con un clic | T49 |
| [ ] | T51 | UI: borrado individual (✕, deslizar, Deshacer) | T50 |
| [ ] | T52 | UI: borrar por proveedor y borrar todo | T51 |
| [ ] | T53 | UI: Home "Descubrir" | T48 |
| [ ] | T54 | UI: favoritos | T53 |
| [ ] | T55 | UI: botón "Siguiente ▸" | T46, T50 |
| [ ] | T56 | Cierre v1.2.0: título CAD, docs, firmas, prueba manual, release | todas |

T01–T29 = v1.0 (`docs/PROGRESO_v1.0.md`) · T30–T35 = v1.1.0 AnimeAV1 (`docs/PROGRESO_v1.1.md`) · T36 = cierre documental de la v1.1.0 · T37 = portabilidad Android · T38 = cierre documental de T37.
T39–T56 = v1.2.0 (Home, historial lateral, favoritos, "Siguiente"); plan en `tareas/v1.2_PLAN.md`.

## Notas v1.2 (T39–T56)

### T39 — Investigación del Home (informe + fixtures)

- **Resultado**: solo investigación, sin código. Comprobaciones reales: `gofmt -l .` → **sin salida**; `go test -race -count=1 ./...` → **11 paquetes `ok`**, **303 `--- PASS`** (mismo recuento antes → después: nada de Go cambió). `git status` muestra solo los archivos permitidos (más los `tareas/T*.md`/`v1.2_PLAN.md` que ya estaban sin confirmar **antes** de empezar la sesión y no se tocaron). Evidencia de la investigación: ~11 peticiones a JKAnime + 3 al CDN y ~24 a AnimeAV1 (máx. 40 por sitio), selectores verificados con regex/perl sobre el HTML **crudo**, Playwright solo para abrir los menús de orden/filtros de `/catalogo` (no están en el crudo) y para observar la URL del filtro de estado.
- **Archivos tocados**: `docs/investigacion/home.md` (informe nuevo) · fixtures nuevos: `internal/scraper/jkanime/testdata/home_portada.html`, `home_directorio.html`, `home_directorio_emision.html`, `home_directorio_popularidad.html`, `home_estrenos.html` · `internal/scraper/animeav1/testdata/home_portada.html`, `home_portada_data.json`, `home_catalogo.html`, `home_catalogo_data.json`, `home_catalogo_emision.html`, `home_catalogo_emision_data.json` · `PROGRESO.md` (esta nota).
- **Desviaciones**: ninguna respecto a la ficha (ningún archivo tocado fuera de "Puedes tocar"). Sanitizado: los 5 fixtures de JKAnime traían **un CSRF real** en `<meta name="csrf-token" content="…">` (1 aparición cada uno) → sustituido por `TOKEN_DE_PRUEBA`; tras el escaneo solo queda el nombre del meta, 0 valores reales. AnimeAV1: 0 tokens. Sin HTML ni datos artificiales. Hallazgos clave para T43/T44 (detalle en el informe): el directorio de JKAnime pinta tarjetas con JS pero el listado vive como **JSON estricto `var animes`** dentro de un `<script>` (25848 B, 0 `<` internos) → extraíble sin XHR; filtros exactos `estado=emision|finalizados|estrenos`, `filtro=popularidad|nombre`, `?p=`; la portada de JKAnime ofrece "Animes recientes" (32 tarjetas SSR), "Top animes" (10) y "Programación" (32+32+8 enlaces de episodio), y "Continuar mirando" **exige sesión** → no usable. AnimeAV1: `/` trae `latestEpisodes` (20) y `latestMedia` (20) en SSR y `/__data.json`; el catálogo filtra por **slug** (`?status=emision|finalizado|proximamente`, el número **no** filtra) y ordena con `order=default|score|popular|title|latest_added|latest_released` (los 3 primeros slugs por orden documentados); `?status=emision` → 58 títulos. Slugs del Home = formato de `Anime.ID`; 0 coincidencias con `excepciones.json`; portadas JKAnime (`cdn.jkdesa.com`) 200 sin Referer.
- **Pendientes**: T43 (Home de JKAnime: `HomeProvider` + parsers contra estos fixtures; decidir si "En emisión" va con el JSON del directorio o con la portada) y T44 (Home de AnimeAV1: usar `home_catalogo_emision_data.json`/`home_portada_data.json`; la guarda de `filters.status` evita que un slug de status mal renombrado devuelva todo el catálogo sin error). TTL de caché sugerido en el informe: 10 min (`home:<proveedor>`), coherente con el plan.

### T40 — models (progreso ampliado, Favorite, HomeSection, ResumeInfo, NextEpisode)

- **Resultado**: solo Go en `internal/models`. **`Episode.Number` es `int`** → `EpisodeNumber`/`Number` de los tipos nuevos son `int`. Comprobaciones reales (Docker Desktop estaba parado; se arrancó para poder ejecutarlas): `gofmt -l .` → **sin salida**; `go vet ./...` y `go vet -tags integration ./...` → **limpios**; `go build ./...` → **OK**; `go test -race -count=1 ./...` → **11 paquetes `ok`** (`miapp` sin tests en la raíz), E2E offline incluidos. Tests: **303 → 313** (`--- PASS` de nivel superior); `internal/models`: **14 → 24** (`ids_test.go` 5 + `models_test.go` 9→13 + `v12_test.go` 6). Los 10 nombres nuevos salen en PASS: `TestWatchProgressJSONAntiguoSinCamposNuevos`, `TestWatchProgressJSONRoundTripCamposNuevos`, `TestWatchProgressProviderName`, `TestWatchProgressTagsJSON`, `TestFavoriteKey`, `TestFavoriteJSONTags`, `TestHomeSectionJSONTags`, `TestResumeInfoJSONTags`, `TestNextEpisodeJSONTags`, `TestResumeActionConstantes`.
- **Archivos tocados**: `internal/models/models.go` (solo campos nuevos al final de `WatchProgress` + su comentario) · `internal/models/models_test.go` (4 tests nuevos + helper `clavesJSON` + **JSON esperado de `TestWatchProgressIdaYVuelta`**, ver desviaciones) · `internal/models/v12.go` (nuevo: `Favorite`/`Key`, `HomeSection`, constantes `ResumeAction*`, `ResumeInfo`, `NextEpisode`, `WatchProgress.ProviderName`) · `internal/models/v12_test.go` (nuevo: 6 tests) · `docs/firmas/models.md` (bloque `### T40 — models`) · `PROGRESO.md` (esta nota).
- **Desviaciones**: (1) **`TestWatchProgressIdaYVuelta` (test existente) tuvo que cambiar su JSON esperado**: la ficha dice "solo añadir tests" y "los existentes no cambian", pero ese test compara el JSON **exacto** de `WatchProgress` y los 5 campos nuevos van **sin `omitempty`** (exigido por `v1.2_PLAN.md` y por el test nuevo `TestWatchProgressTagsJSON`): sin tocarlo, `go test` queda en rojo de forma inevitable. Cambio mínimo y necesario: se le añadieron al `want` las 5 claves en cero (`"provider":"","anime_title":"","cover_url":"","episode_number":0,"updated_at":0}`) y un comentario que explica el porqué; **nombre, número de tests y resto del cuerpo intactos** (14 → 24, sin quitar ni renombrar). Ningún archivo tocado fuera de "Puedes tocar". (2) Sin datos ni HTML artificiales: todo el JSON de los tests es inventado pero es JSON de prueba del propio modelo, no fixtures de sitios.
- **Pendientes**: T41 (db) y T42 (favoritos) consumen estos tipos; T41 debe rellenar `Provider`/`AnimeTitle`/`CoverURL`/`EpisodeNumber`/`UpdatedAt` en `Save` (regla del plan: `UpdatedAt == 0` → `now.Unix()`, distinto de 0 se conserva; `Provider == ""` → `ProviderName()`). `frontend/wailsjs/**` se regenera en T47 (los modelos TS de `WatchProgress` cambiarán al regenerar). Nada más en `models`.

### T41 — db (historial con fecha, listado reciente y borrados)

- **Resultado**: solo Go en `internal/db`. Comprobaciones reales (Docker en marcha): `gofmt -l .` → **sin salida**; `go vet ./...` y `go vet -tags integration ./...` → **limpios**; `go build ./...` → **OK**; `go test -race -count=1 ./...` → **11 paquetes `ok`** (E2E offline de `internal/app` incluidos, sin red). Tests: **313 → 329** (`--- PASS` de nivel superior); `internal/db`: **13 → 29**; `internal/app` intacto en **34**. Los **16 nombres exactos** de la ficha salen todos en PASS, con `-race` (entre ellos `TestBorradoConcurrenteConSave`). Firmas contrastadas con `go doc -all ./internal/db`.
- **Archivos tocados**: `internal/db/db.go` (reloj inyectable `now`, `SaveProgress` sella `UpdatedAt` y `Provider`, nuevos exportados `LastPerAnime`/`DeleteAnime`/`DeleteProvider`/`Clear` con los privados `borrar`/`reindexar`/`mejorQue` y el tipo `claveAnime`) · `internal/db/historial_reciente_test.go` (**nuevo**: los 16 tests) · `docs/firmas/db.md` (bloque `### T41 — db`) · `PROGRESO.md` (esta nota). **`internal/db/db_test.go` no se tocó**: los 13 tests de T22 siguen en verde tal cual.
- **Desviaciones**: ningún archivo tocado fuera de "Puedes tocar". (1) Decisión de implementación propia (la ficha no la fija): **con el almacén cerrado `LastPerAnime` devuelve lista vacía** —su firma no admite `error`—, mientras `SaveProgress`/`GetProgress`/borrados siguen devolviendo `ErrCerrado`; queda documentado para que T45 no espere un error de ese método. (2) Los registros "antiguos" de los tests (sin `updated_at` ni `provider`) se fabrican **escribiendo el JSON a mano** en `historial.json` (`escribirHistorialViejo`): es la única forma de simularlos, porque `Save` ahora los rellena. Es JSON de prueba propio, sin HTML ni datos de sitios. (3) `DeleteProvider("")` borraría los registros sin proveedor (comparación directa con `ProviderName()`); caso inalcanzable desde la UI, sin tratamiento especial. (4) Si la escritura de un borrado falla, se devuelve `(borrados, err)` con los borrados ya aplicados en memoria (mismo criterio que `SaveProgress`: el próximo guardado los persiste).
- **Pendientes**: T45 (`Nucleo` + `app.go`) debe cablear `LastPerAnime`/`DeleteAnime`/`DeleteProvider`/`Clear` en `HistorialReciente`/`BorrarHistorial*` y decidir qué error mostrar cuando el almacén esté cerrado (T41 devuelve lista vacía sin error). T42 (favoritos) corre en paralelo y no depende de esto. Ningún cambio pendiente en `models` ni en `internal/app`.

### T42 — db (favoritos)

- **Resultado**: solo Go en `internal/db`, **dos archivos nuevos**. Comprobaciones reales (Docker en marcha): `gofmt -l .` → **sin salida**; `go vet ./...` y `go vet -tags integration ./...` → **limpios**; `go build ./...` → **OK**; `go test -race -count=1 ./...` → **11 paquetes `ok`** (E2E offline de `internal/app` incluidos, sin red). Tests: **329 → 343** (`--- PASS` de nivel superior); `internal/db`: **29 → 43**. Los **14 nombres exactos** de la ficha salen todos en PASS, con `-race` (entre ellos `TestFavoritosConcurrentes`). Firmas contrastadas con `go doc -all ./internal/db`.
- **Archivos tocados**: `internal/db/favoritos.go` (**nuevo**: `Favorites`, `OpenFavorites`, `Add`, `Remove`, `Has`, `List`, la constante `nombreFavoritos` y los privados `cargar`/`enMemoria`/`reindexar`/`respaldar`/`escribir`) · `internal/db/favoritos_test.go` (**nuevo**: los 14 tests; reutiliza los helpers `abrir` y `sinTemporales` de `db_test.go`) · `docs/firmas/db.md` (bloque `### T42 — db favoritos`) · `PROGRESO.md` (esta nota). **`internal/db/db.go` sin tocar** (`git diff internal/db/db.go` vacío), igual que `db_test.go`, `internal/models/**`, `internal/app/**` y el frontend.
- **Desviaciones**: ningún archivo tocado fuera de "Puedes tocar". (1) La escritura atómica de `DB.escribir` no es reutilizable sin tocar `db.go` → se duplicó la lógica mínima en `favoritos.go`, tal como prevé la ficha; solo se reutilizan del paquete el símbolo `ErrDirectorioVacio` y la constante `sufijoRespaldo` (`.bak`). (2) Decisiones que la ficha no fija: **sin `Close`** (la API exacta del plan no lo incluye y no hay ciclo de vida: cada `Add`/`Remove` ya escribe a disco); el JSON va **indentado con 2 espacios** (`MarshalIndent`, como pide la ficha); si la escritura falla, `Add` devuelve `(eraNuevo, err)` y `Remove` `(true, err)` —mismo criterio que el `borrar` de T41—; el error de `Add` inválido nombra los dos campos ("el favorito necesita provider y anime_id"); cargar un `favoritos.json` con claves duplicadas (editado a mano) hace upsert por `Key()` conservando la posición. (3) Sin datos ni HTML artificiales: los tests solo fabrican `models.Favorite` propios y basura de JSON de prueba.
- **Pendientes**: T45 (`Nucleo` + `app.go`) debe abrir `db.OpenFavorites` en `Arrancar` (mismo directorio que el historial), decidir el mensaje de UI cuando no hay almacén (`ErrFavoritosNoDisponible`, fijada en la ficha de T45) y cablear `AgregarFavorito`/`QuitarFavorito`/`ListarFavoritos`/`EsFavorito` sobre este almacén; T45 no debe esperar un `Close` ni un error de `List`. Nada pendiente en `models` ni en el resto de `internal/db`.

### T43 — `HomeProvider` + Home de JKAnime

- **Resultado**: solo Go en `internal/scraper` y `internal/scraper/jkanime`. **Paso 0** verificado antes de codificar (coincide con la ficha; detalle abajo). Comprobaciones reales (Docker en marcha): `gofmt -l .` → **sin salida** (tras `gofmt -w` de `home_test.go`, solo alineación de constantes); `go vet ./...` y `go vet -tags integration ./...` → **limpios**; `go build ./...` → **OK**; `go test -race -count=1 ./...` → **11 paquetes `ok`** (E2E offline incluidos, sin red). Tests: **343 → 355** (`--- PASS` de nivel superior); `internal/scraper`: **7 → 8**; `internal/scraper/jkanime`: **42 → 53**; el resto intacto (`internal/app` 34, `db` 43…). Los **12 nombres exactos** de la ficha salen todos en PASS con `-race`. Smoke en verde contra el sitio real: `go test -tags integration ./internal/scraper/jkanime/ -run HomeRedReal -v` → 2 secciones (**30 + 32** items; el primer slug de "En emisión" hoy es `shuiro-no-kamen`, feed vivo). Firmas contrastadas con `go doc` y el código.
- **Paso 0 (a)**: `go doc ./internal/models Anime` → `Anime{ID, Title, CoverURL, Provider string}`; **no existe campo `Type`** → los items del Home salen sin tipo (el `type`/`tipo` del JSON del directorio se ignora), tal como prevé la ficha.
- **Paso 0 (b)** — `testdata/home_directorio_emision.html` línea 545 (único `var animes =` del archivo; los otros 29 `"slug":` están en esa misma línea): (1) el valor es un **objeto** paginador de Laravel, **no** un array directo; la clave con los 30 items es **`data`** (el objeto trae además `current_page, first_page_url, from, last_page, last_page_url, links, next_page_url, path, per_page, prev_page_url, to, total`; `per_page:30`, `total:85`, `last_page:3`); (2) claves del item: `id, title, synopsis, image, studios, slug, type, status, url, estado, tipo, base64_id, short_title`; (3) `image` es **URL absoluta** (`https://cdn.jkdesa.com/assets/images/animes/image/<slug>.jpg`); (4) slug del último item: **`hotel-inhumans-2nd-season`** (primero: `hyouken-no-majutsushi-ga-sekai-wo-suberu-ii`). Tras el JSON: `;` y `var mode = …`; `json.NewDecoder` se detiene solo. Portada: 32 `custom_thumb_home`, primer slug `holy-knight`, último **`dark-summoner-to-dekiteiru`**, portadas `/<ID>.jpg` y `alt`==título en las 32.
- **Paso 0 (c)**: en `search.go` la inyección es `New(doer, baseURL, resolver)` con campos exportados `BaseURL`/`Doer`/`Resolver`; el slug sale de `slugFromURL`, las URLs se absolutizan con `absoluta(base, ref)` y las excepciones se aplican como `resolver.Resolve(ctx, absoluta(base, href))` → `slugFromURL(resuelta.URL)`. **Todos reutilizables desde `home.go` sin tocar `search.go`** (mismo paquete); también `pausaPaginas`/`esperar` de `episodes.go`.
- **Ajuste 2 (verificación antes de codificar)**: `ExceptionResolver.Resolve` **no** es 100% local en su primera llamada: `Resolve → load → fetch` descarga el mapa remoto **una sola vez** por proceso (bajo mutex con `loaded`; el fallo no se cachea y se reintenta). Después es solo lookup en memoria: **cero peticiones por item** (los 30+32 items comparten esa única descarga). Es el diseño de T06 y el mismo flujo que ya usa `Search`; no se detuvo la tarea. Lo comprueba `TestHomeAplicaExcepciones`: 4 items resueltos en dos secciones ⇒ **1** sola petición al servidor del mapa. Con `Doer` nil (tests sin mapa) no hay red (`ErrNilDoer` silencioso, siguen sin excepciones).
- **Archivos tocados**: `internal/scraper/provider.go` (solo añade `HomeProvider` al final) · `internal/scraper/registry_test.go` (solo añade `fakeConHome` + `TestHomeProviderEsOpcional`) · `internal/scraper/jkanime/home.go` (**nuevo**: `Home`, `paginaHome`, `parseEmision`, `parseRecientes`, `dedupHome`, `directorioPagina`/`directorioItem`, constantes `rutaEmision`/`rutaPortada`/`maxItemsEmision`/`maxItemsRecientes`/`varAnimes` y el assert `HomeProvider`) · `internal/scraper/jkanime/home_test.go` (**nuevo**: los 11 tests + helpers `servidorHome`/`fixturesHome`/`emisionDePrueba`/`portadaDePrueba`/`buscarSeccion`/`idsDe`) · `internal/scraper/jkanime/home_integration_test.go` (**nuevo**: `TestHomeRedReal`) · `docs/firmas/scraper.md` (bloque `### T43`) · `docs/firmas/jkanime.md` (bloque `### T43`) · `PROGRESO.md` (esta nota). `search.go`, `detail.go`, `episodes.go`, los fixtures `home_*` de T39, `internal/models/**`, `internal/app/**` y el frontend **sin tocar**.
- **Desviaciones**: ningún archivo tocado fuera de "Puedes tocar". (1) **Ajuste pedido por el humano**: `ExceptionResolver` se aplica **también a la sección `emision`** (la ficha base solo lo fijaba en "Comunes"/recientes): `absoluta(base, slug)` → `Resolve` → `slugFromURL`, idéntico a Search; `TestHomeAplicaExcepciones` cubre **ambas secciones** con claves reales de `excepciones.json`. (2) **Límite de lectura**: se reutiliza **`maxSearchBytes`** (10 MB, la constante que ya usaba el paquete para HTML) en lugar de definir `maxHomeBytes`, por el ajuste 3 del humano. (3) **HTML/JSON artificial** (prohibido inventar selectores; los parsers se validan contra los fixtures reales, que cubren los casos felices): `emisionDePrueba`/`portadaDePrueba` en 4 tests — `TestHomeDeduplicaPorSlug` (JSON `{"data":[…]}` con item repetido + tarjeta repetida), `TestHomeAplicaExcepciones` (items con la URL **absoluta** rota de `excepciones.json`, igual que los hrefs absolutos de `search.html`, porque el test corre contra un `httptest.Server` cuya base no es `jkanime.net`), `TestHomeSeccionVaciaSeOmite` (portada vacía) y `TestHomeEmisionSinJSONDevuelveError` (directorio sin `var animes`). Ningún fixture real se modificó. (4) Decisión propia no fijada por la ficha: si `ctx` se cancela **durante** el Home (pausa o petición) se devuelve el error de cancelación envuelto aunque ya hubiera secciones completas — con ctx precancelado (el test) devuelve vacío + `context.Canceled` y 0 peticiones. (5) Fallo total ⇒ `"el Home de JKAnime no está disponible: %w"` con el **primer** error (el de `emision`, el primero pedido); `errors.Is` sigue funcionando (`httpx.ErrServerError`).
- **Pendientes**: T44 (Home de AnimeAV1) debe implementar `scraper.HomeProvider` para `animeav1` (fixtures `home_*` de T39; la guarda de `filters.status` que anotó T39). T45 (`Nucleo` + `app.go`) debe detectar el `HomeProvider` con aserción de tipo (`provider.(scraper.HomeProvider)`) y cachear `home:<proveedor>` 10 min, y puede reutilizar el patrón de fallo parcial de este `Home`. El `Type` de `models.Anime` sigue sin existir: si el producto quiere tipo/estado en las tarjetas del Home hay que ampliar `models` en una tarea nueva. `frontend/wailsjs/**` se regenera en T47.

### T44 — Home de AnimeAV1

- **Resultado**: solo Go en `internal/scraper/animeav1`. **Paso 0** verificado antes de codificar (todas las rutas de la ficha existen en los fixtures; no hubo que detenerse; detalle abajo). Comprobaciones reales (Docker en marcha): `gofmt -l .` → **sin salida** (tras `gofmt -w` de `home_test.go`, solo alineación de comentarios); `go vet ./...` y `go vet -tags integration ./...` → **limpios**; `go build ./...` → **OK**; `go test -race -count=1 ./...` → **11 paquetes `ok`** (E2E offline incluidos, sin red). Tests: **355 → 366** (`--- PASS` de nivel superior); `internal/scraper/animeav1`: **59 → 70**; el resto intacto (`internal/app` 34, `jkanime` 53…). Los **11 nombres exactos** de la ficha salen todos en PASS con `-race`. Smoke en verde contra el sitio real: `go test -tags integration ./internal/scraper/animeav1/ -run HomeRedReal -v` → **2 secciones × 20 items** (emision: `battle-spirits-re-zekkai-no-kuu`; recientes es feed vivo y ese día empezaba por `kyoufu-collector`, id 4491).
- **Paso 0 (a)** — `go doc`: `PageData(crudo []byte) (map[string]any, error)` y `Unflatten(plano []any) (any, error)` están **exportadas** y se reutilizan **sin modificarlas**. Helpers de la búsqueda, sin exportar pero del mismo paquete (reutilizables desde `home.go` sin tocar `animeav1.go`): `resultadosAnime(datos map[string]any) ([]models.Anime, error)` (lee `results`; slug+título obligatorios, `ID` = slug), `portada(id any) string` (string o float64 → `covers/<id>.jpg`) e `idTexto(id any) string` (float64 → `strconv.FormatFloat(v, 'f', -1, 64)`, sin notación científica). La búsqueda **no** rellena categoría/tipo (`models.Anime` no tiene `Type`) → no hay mapeo de categoría que reutilizar y los items del Home salen **sin tipo**, como predijo T43.
- **Paso 0 (b)** — rutas exactas dentro del JSON ya desempaquetado (verificadas con un test temporal en el paquete, borrado después): (1) `testdata/home_catalogo_emision_data.json`: `results` está en la **raíz** que devuelve `PageData` (`datos["results"]`), lista de **20 items**; `filters.status = 2` (float64); `id` de results = **string** (p. ej. `"4479"`); primer slug **`battle-spirits-re-zekkai-no-kuu`**, último **`marronnier-oukoku-no-shichinin-no-kishi`**. (2) `testdata/home_catalogo_data.json` (sin filtro): misma raíz (9 claves); `filters.status` = **`null`**. (3) `testdata/home_portada_data.json`: la raíz de `PageData` (5 claves: `latestTopComments`, `featured`, `latestComments`, `latestEpisodes`, `latestMedia`) **trae `latestMedia` directamente** — no hizo falta fijar índice de nodo—, lista de **20 items**; `id` = **numérico** (float64, p. ej. 4489); primer slug **`hotaru-no-yomeiri`**, último **`mouse-cursor-de-genjitsu-wo-sousa-dekiru-you-ni-natta-node-onna-no-ko-wo-ippai-click-shimaasu`**.
- **Paso 0 (c)** — los 20 slugs de `results` de los dos fixtures de catálogo son los **mismos y en el mismo orden** (coherente con los 5 bytes de diferencia): el orden por defecto del catálogo lista primero los títulos en emisión. Informativo; no cambió el diseño.
- **Archivos tocados**: `internal/scraper/animeav1/home.go` (**nuevo**: `Home`, `seccionHome`, `parseEmision`, `filtroEmisionAplicado`, `parseRecientes`, `itemsHome`, `dedupHome`, `esperar`, constantes `rutaHome`/`maxItemsHome`/`pausaSecciones`/`statusFiltroEmision` y el assert `HomeProvider`) · `internal/scraper/animeav1/home_test.go` (**nuevo**: los 11 tests + helpers `servidorHome`/`fixturesHome`/`payloadDePrueba`/`buscarSeccion`/`idsDe`) · `internal/scraper/animeav1/animeav1_integration_test.go` (solo añade `TestHomeRedReal`) · `docs/firmas/animeav1.md` (bloque `### T44 — animeav1`) · `PROGRESO.md` (esta nota). `animeav1.go`, `devalue.go`, `detail.go`, `episodes.go`, `sources.go`, los fixtures `home_*` de T39, `internal/scraper/provider.go`, `internal/app/**` y el frontend **sin tocar**.
- **Desviaciones**: ningún archivo tocado fuera de "Puedes tocar". (1) **JSON artificial** (prohibido inventar selectores; los parsers se validan contra los fixtures reales, que cubren los casos felices): `payloadDePrueba` arma cuerpos devalue mínimos (literales sueltos en el array plano, objetos solo con referencias por índice, como trabaja devalue) en `TestHomeDeduplicaPorSlug` (item repetido en cada sección) y `TestHomeSeccionVaciaSeOmite` (`latestMedia` vacío; el sitio siempre trae 20). `TestHomeEmisionFiltroNoAplicadoEsError` sirve el **fixture real sin filtro** `home_catalogo_data.json` en la ruta `status=emision` (la guarda dispara y `recientes` se sirve igualmente). Ningún fixture real se modificó. (2) Decisiones que la ficha no fija: mensajes `"el Home de animeav1 no está disponible: %w"` (primer error, el de `emision`) y `"el Home de animeav1 se interrumpió: %w"` (patrón de T43); `esperar(ctx, d)` **local** en `home.go` (jkanime lo tiene en `episodes.go`, archivo ajeno al alcance); `itemsHome(datos, clave)` replica el criterio de `resultadosAnime` para `latestMedia` **sin modificar** `resultadosAnime`; la guarda de estado compara `filters["status"].(float64) == 2` (así llega desde `encoding/json`/`Unflatten`); `Operation: "home"` y límite `maxBusquedaBytes` como en la búsqueda. (3) El smoke confirma que "recientes" es un feed vivo (el primer slug del día de la prueba no es el del fixture de T39): comportamiento esperado, cubierto por la caché de 10 min que prevé T45.
- **Pendientes**: T45 (`Nucleo` + `app.go`) debe detectar el `HomeProvider` de **ambos** proveedores con aserción de tipo y cachear `home:<proveedor>` 10 min (los errores no se cachean, `noCache` salta la lectura); puede reutilizar el patrón de fallo parcial de estos `Home`. El `Type` de `models.Anime` sigue sin existir (los items del Home salen sin tipo). `frontend/wailsjs/**` se regenera en T47. Nada pendiente en el resto de `animeav1` ni en `internal/scraper`.

### T45 — `Nucleo` + `app.go`: Home, favoritos, borrados

- **Resultado**: solo Go en `internal/app/nucleo.go`, `app.go` y los tests. Comprobaciones reales (Docker): `gofmt -l .` sin salida · `go vet ./...` y `go vet -tags integration ./...` limpios · `go build ./...` OK · `go test -race -count=1 ./...` → **11 paquetes `ok`, en verde sin red**: tests **366 → 382** (`--- PASS`), `internal/app` **34 → 50** (los 16 nuevos de la ficha). Guardia `portabilidad_test.go` en verde; `go list -deps ./internal/app | grep wailsapp` sin resultados (P2 intacto); `git diff frontend/` vacío. E2E de red real con el paso Home nuevo en verde: `go test -tags integration ./internal/app -run 'RedReal'` → JKAnime 15 s y AnimeAV1 3 s, ambos con "2 secciones (2 con items)".
- **Archivos tocados**: `internal/app/nucleo.go` (sentinels `ErrHomeNoDisponible`/`ErrFavoritosNoDisponible`; campos `favoritos *db.Favorites`, `muHome`, `home map[string]entradaHome`; `Arrancar` abre `db.OpenFavorites(n.rutaHistorial)` con `slog.Warn` si falla; métodos `Home`/`AgregarFavorito`/`QuitarFavorito`/`ListarFavoritos`/`EsFavorito`/`HistorialReciente`/`BorrarHistorialAnime`/`BorrarHistorialProveedor`/`BorrarHistorialTodo`) · `app.go` (los 9 métodos Wails delegando, tras `GetProgress`) · `internal/app/nucleo_home_test.go` (**nuevo**: 6 tests + doble `homeFalso`) · `internal/app/nucleo_favoritos_test.go` (**nuevo**: 5 tests) · `internal/app/nucleo_historial_test.go` (**nuevo**: 5 tests) · `internal/app/app_e2e_test.go` y `app_e2e_integration_test.go` (solo los pasos `1b`/`5b`/`6b` descritos en las firmas, más fixtures y rutas de los fakes) · `docs/firmas/app.md` (bloque `### T45 — app`) · `PROGRESO.md` (esta nota). `main.go`, `frontend/**`, `internal/{db,models,scraper,cache}/**` **sin tocar**.
- **Desviaciones**: ningún archivo tocado fuera de "Puedes tocar". (1) **Caché de Home propia en `nucleo.go`** en vez de reusar `internal/cache` (decisión de la ficha): el paquete solo admite `Entrada{Detail, Episodes, Complete}` y no puede guardar `[]models.HomeSection` sin tocarlo (fuera de alcance). Se replica el comportamiento previsto: clave `cache.Clave("home", proveedor)` = `home:<proveedor>`, TTL `ttlCacheDetalle` (10 min), `noCache` salta la lectura, errores jamás cacheados, secciones devueltas con copia defensiva. (2) `db.Favorites` no tiene `Close` → `Cerrar()` no la cierra (documentado en T42; cerrar no invalida el store). (3) Validación de argumentos de los borrados ANTES de tocar db (la nota de T41: un proveedor `""` borraría registros sin proveedor): errores `"borrar historial necesita proveedor y anime: provider=%q animeID=%q"` / `"borrar historial necesita un proveedor: provider=%q"`. (4) El doble `homeFalso` de los tests embebe `proveedorFalso` (mismo patrón que `fakeConHome` de T43). (5) Fixtures del fake: JKAnime carga `home_portada.html`/`home_directorio_emision.html` con la misma reescritura de `jkanime.net` que el resto; AnimeAV1 distingue el catálogo del Home por `?status=emision` en la MISMA ruta `catalogo/__data.json`.
- **Pendientes**: T46 (`Nucleo.Retomar` y `SiguienteEpisodio`) puede empezar; el Deshacer de T51 necesita `UpdatedAt ≠ 0` (ya garantizado, testeado en `TestGuardarProgresoConservaUpdatedAtNoCero`). `frontend/wailsjs/**` sigue sin regenerar: los bindings de los 9 métodos nuevos llegan en T47 (el frontend no llama a ninguno aún, así que `npm run check`/build siguen en verde). El `Type` de `models.Anime` sigue sin existir (items del Home sin tipo, pendiente de producto). Nada pendiente en `internal/app` ni en `app.go`.

## Auditoría de portabilidad (T37)

> Paso 1 de T37, hecha **antes de mover nada**. Lo que contradice la ficha o queda fuera de "Puedes tocar" se reporta aquí (regla 7).

**(a) `os.UserConfigDir` / `UserHomeDir` / `Getenv` / `runtime.GOOS` / `exec` / cgo / build tags de SO** (`internal/**` + raíz):

- **`os.UserConfigDir`: `internal/db/db.go:107`** (función `rutaPorDefecto()`, llamada por `Open("")` en `db.go:83`), más sus comentarios (`db.go:3, 73, 105`) y el test `internal/db/db_test.go:555` (`TestRutaPorDefectoEsConfigDelUsuarioMasMiapp`). En la raíz **solo comentarios** (`app_nucleo.go:129, 177`). → **contradice P3**; la ficha lo prevé en el paso 4 ("si el cálculo está dentro de `internal/db`, puedes tocarlo solo para recibir la ruta por parámetro"): `app.go` pasa la ruta y `db.Open` deja de calcularla. Ver decisión 3 abajo.
- `os.UserHomeDir`: **ninguno**.
- `Getenv`: **`app.go:46`** (`MIAPP_DEV`); **ninguno en `internal/**`**.
- `runtime.GOOS` / `runtime.GOARCH`: **ninguno**.
- `os/exec`: **ninguno**.
- cgo (`import "C"`, `#cgo`): **ninguno**.
- Build tags de SO: **ninguno**. Los únicos `//go:build` del repositorio son `integration` (7 archivos de test): `app_integration_test.go`, `app_e2e_integration_test.go`, `internal/scraper/jkanime/{search,detail,episodes,get_video_sources}_integration_test.go` y `internal/scraper/animeav1/animeav1_integration_test.go`.

**(b) imports de `github.com/wailsapp/*` fuera de `app.go`/`main.go`: ninguno.** Solo `main.go` (`wails`, `pkg/options`, `pkg/options/assetserver`) y `app.go` (`pkg/runtime`). `internal/**` está limpio → P2 ya se cumple.

**(c) archivos de `frontend/src` que importan `wailsjs`: 6**

| archivo | qué importa |
|---|---|
| `api.ts:8-15` | funciones de `go/main/App.js` + tipos `main`, `models` |
| `stores.ts:19-20` | **`EventsOn`/`EventsOff`** de `runtime/runtime.js` + tipos `models` |
| `VistaBusqueda.svelte:4`, `VistaDetalle.svelte:4`, `TarjetaEpisodio.svelte:2`, `VistaReproductor.svelte:5` | solo `import type { models }` |

`VistaReproductor.svelte` ya importa además `VideoSourceView` de `./api` (correcto). → es el trabajo del paso 2.

**(d) `app.go` frente a `app_nucleo.go`:**

- **`app.go`** (118 líneas, `package main`): `App{mu, ctx, core *nucleo}`, `NewApp()`, `startup` (guarda el ctx de Wails, `slog.SetDefault(logx.New(MIAPP_DEV))`, emisor `func(ev, data) { runtime.EventsEmit(ctx, …) }`, `core.arrancar(emisor)`), `shutdown` (`core.cerrar()`), `contexto()` y los **5 métodos Wails** (`SearchAnime`, `GetAnimeDetails`, `GetVideoSources`, `SaveProgress`, `GetProgress`), que solo delegan en el núcleo. Es el único archivo con `os.Getenv` y con `wailsapp` (junto con `main.go`).
- **`app_nucleo.go`** (698 líneas, `package main`): todo el orquestador `nucleo` (registro `jkanime`+`animeav1`, búsqueda combinada, caché, streams de detalle, proxy, historial, eventos), **sin ningún import de Wails**. Exportados hoy: `Emisor`, `RegistroProviders`, `ProxyLocal`, `Resolutor`, `DetalleResultado`, `EpisodiosCompletados`, `ErrDetalleNoDisponible`, `ErrHistorialNoDisponible`; el resto (`nucleo`, `nuevoNucleo`, `nuevoNucleoPorDefecto`, métodos) es interno.
- **Tests**: 8 archivos `app*_test.go` → **37 `func Test`** (3 con tag `integration`: los 2 del E2E real y el smoke de búsqueda; más `TestMain`, que no es test) = **33 efectivos en `miapp`**. Todos en `package main` y referenciando `nuevoNucleo…`/`&App{core: n}`.
- **`main.go`** (37 líneas): `//go:embed` + `wails.Run` con `OnStartup`/`OnShutdown`/`Bind`.

**Tensiones detectadas (regla 7, reportadas antes de tocar):**

1. **`DetalleResultado` y los bindings**: si el tipo se mueve a `internal/app`, Wails regeneraría el namespace TS `main` → `app` y `git diff frontend/wailsjs` dejaría de estar vacío (comprobación de la ficha). **Resolución**: el núcleo conserva su `DetalleResultado` (paquete `app`, mismos campos/tags) y `app.go` define el suyo y traduce (`DetalleResultado{Detail, Episodes}`): 2 structs idénticos, JSON y bindings idénticos.
2. **Los E2E construyen `&App{core: n}`** (tipo de `package main`): al moverse a `internal/app` llamarán al núcleo directamente (`n.Buscar(ctx, …)`, `n.Detalle(…)`, … con `context.Background()`, que es exactamente lo que hace `App.contexto()` sin `startup`). Mismas aserciones, mismo número de tests.
3. **`db.Open("")` pasará a devolver error** (P3 exige que el cálculo salga de `internal/db`); el test `TestRutaPorDefectoEsConfigDelUsuarioMasMiapp` se reescribe para cubrir el nuevo contrato (**13 tests en `internal/db` antes y después**, 14 `func Test` contando `TestMain`). La ruta de escritorio **no cambia**: `app.go` calcula `os.UserConfigDir()/miapp` y la pasa por parámetro → `%AppData%\miapp\historial.json`, sin migración. Único efecto observable: en el (prácticamente imposible) caso de que `os.UserConfigDir()` falle, el `slog.Warn` de `arrancar` lleva el mensaje de `db` en vez del de `rutaPorDefecto` — la app sigue igualmente sin historial.
4. **`DESARROLLO.md` no existe en la raíz**: el documento vive en **`docs/DESARROLLO.md`**; lo trato como el que nombra la ficha (paso 7).
5. **`docs/firmas/db.md` y `docs/firmas/cache.md`** quedan con referencias a `app_nucleo.go`, `nuevoNucleo` y al `Open("")` por defecto: **no están en "Puedes tocar"** → no se tocan; quedan como desviación pendiente (ver "Pendientes activos").

**Estado base (antes de tocar nada):**

- `gofmt -l .` → **`app.go`, `app_nucleo.go`** (el fantasma CRLF del aviso de T36: esos dos archivos están con `\r\n` en el worktree y el contenido está bien formateado; los demás `.go` son LF). Se normalizan a LF al tocarlos.
- `go vet ./...` y `go vet -tags integration ./...` → limpios · `go build ./...` → OK.
- `go test -race -count=1 ./...` → **11 paquetes `ok`, 302 tests**: `miapp` 33 · cache 9 · db 13 · httpx 27 · logx 7 · models 14 · scraper 7 · animeav1 59 · jkanime 42 · extractors 68 · stream 23. E2E offline (JKAnime y AnimeAV1) en verde.

---

## T37 — ejecución y comprobación real

**Qué se hizo (pasos 2–6):**

- **Frontend (paso 2):** `api.ts` es el único archivo con `wailsjs`: añade `EventsOn`/`EventsOff`, reexporta los tipos (`export type { main, models }`), movió `parsePayload` desde `stores.ts` y expone las suscripciones **`onEpisodesUpdate(cb)` / `onEpisodesComplete(cb)`**, que devuelven la función que cancela con `EventsOff(evento)`. `stores.ts` importa de `./api` y sus `cancelarUpdate`/`cancelarComplete` las llaman en `disposeEvents` (mismo comportamiento: quitar el evento quita todos sus listeners). Los 4 `.svelte` pasan `import type { models }` a `'./api'`.
- **Mover el núcleo (paso 3):** `git mv` de `app_nucleo.go → internal/app/nucleo.go` y de los 8 `app*_test.go → internal/app/` (el historial de git se conserva: `git status` los pinta como `R`). `package app`; se exportó lo mínimo: `Nucleo`, `NuevoNucleo`, `NuevoNucleoPorDefecto(dir)` y los métodos `Arrancar/Cerrar/Buscar/Detalle/Fuentes/GuardarProgreso/VerProgreso` (todo lo demás sigue sin exportar). Los E2E sustituyen `&App{core: n}` por llamadas directas al núcleo con `context.Background()` y sus fixtures pasan a rutas relativas `../scraper/…` (el cwd del test ahora es `internal/app/`).
- **Directorio de datos (paso 4):** `app.go` añade `dirHistorial()` (único `os.UserConfigDir` del código) y se lo pasa a `NuevoNucleoPorDefecto`; en `internal/db` se eliminó `rutaPorDefecto()` y `Open("")` devuelve el nuevo **`db.ErrDirectorioVacio`**. Ruta final idéntica: `%AppData%\miapp\historial.json`.
- **`app.go` delgado (paso 5):** solo traduce; define su `DetalleResultado` idéntico al de `internal/app` (misma traducción de un struct al otro en `GetAnimeDetails`) para que Wails lo siga generando en el namespace `main`. **`main.go` sin cambios.**
- **Guardia (paso 6):** `internal/app/portabilidad_test.go` barre `internal/**` con `go/parser` y falla con `github.com/wailsapp/*` (P2) o `os.UserConfigDir`/`UserHomeDir` (P3). Verificado en ambos sentidos: en verde con el código actual, en rojo con un `internal/tmpcheck` temporal que importaba `wailsapp` y llamaba a `os.UserConfigDir`.

**Comprobaciones de la ficha (resultado real):**

| comprobación | resultado |
|---|---|
| `gofmt -l .` | **sin salida** (`app.go` e `internal/app/nucleo.go` normalizados a LF) |
| `go vet ./...` · `go vet -tags integration ./...` | **limpios** |
| `go build ./...` | **OK** |
| `go test -race -count=1 ./...` | **11 paquetes `ok`**; E2E offline JKAnime y AnimeAV1 **en verde** |
| tests antes → después | **302 → 303** (`miapp` 33 → `internal/app` 33+1 = 34; `db` 13 y el resto intactos; los mismos 302, más la guardia) |
| `go list -deps ./internal/app \| grep wailsapp` | **vacío** |
| `grep -rn "wailsjs" frontend/src` | **solo `api.ts`** |
| `wails generate module` y `wails build` → `git diff --stat frontend/wailsjs` | **vacío** (bindings idénticos) |
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | **OK** (el aviso de chunk >500 kB es preexistente) |
| `GOOS=android GOARCH=arm64 go vet ./internal/...` (informativo) | **sin salida**: `internal/**` compila limpio para android/arm64 |
| `wails build -platform windows/amd64` | **OK** → `build/bin/miapp.exe` (13,2 MB) |

**Desviaciones nuevas de esta ejecución (regla 7 / fuera de "Puedes tocar"):**

- **`DESARROLLO.md` no existe en la raíz** (decisión 4): se editó **`docs/DESARROLLO.md`**, el documento que nombra la ficha.
- **`docs/firmas/db.md` y `docs/firmas/cache.md` quedan desfasados** (decisión 5): siguen mencionando `app_nucleo.go`, `nuevoNucleo…` y `Open("")` con el directorio por defecto. **No están en "Puedes tocar" → no se han tocado**; hay que actualizarlos en una tarea posterior (ver "Pendientes activos").
- **Rutas de fixtures de los E2E**: `internal/scraper/…/testdata/…` → `../scraper/…/testdata/…` (consecuencia directa de mover el cwd de los tests; mismo archivo, misma ruta real).
- **`NuevoNucleoPorDefecto()` ahora recibe el directorio**: los 3 call sites de test pasan `t.TempDir()` (antes el E2E real escribía el historial en un temporal **después** de construir el núcleo; ahora lo recibe, misma efectividad).
- **Normalización LF** de `app.go`/`internal/app/nucleo.go` (el CRLF fantasma que ya señaló T36): `gofmt -l .` queda limpio, pero el diff de esos dos archivos muestra las líneas completas.
- **`db.ErrDirectorioVacio`** es un símbolo nuevo exportado (no es de la ficha, pero la firma `Open("")` lo exige para no devolver un error anónimo).

---

## T38 — cierre documental de T37

Solo `.md`, sin tocar código. Contraste con `go doc ./internal/db`, `go doc ./internal/cache` y `go doc -all ./internal/app`:

- **`docs/firmas/db.md`**: `Open(dir)` ya **no** tiene directorio por defecto — con `dir == ""` devuelve **`db.ErrDirectorioVacio`** (símbolo nuevo de T37); añadido `ErrDirectorioVacio` a las firmas del paquete y corregido el test de "ruta por defecto" → `TestOpenConDirectorioVacioFalla` (13 tests, igual que antes). Bloque de integración reescrito al núcleo actual: `internal/app/nucleo.go` (`package app`), `type Nucleo`, `NuevoNucleo(reg, px, sel, emis, c, rutaHistorial)`, `NuevoNucleoPorDefecto(rutaHistorial)`, métodos exportados `Arrancar`/`Cerrar`/`GuardarProgreso`/`VerProgreso` y tests en `internal/app/`.
- **`docs/firmas/cache.md`**: el bloque de paquete **ya coincidía** con `go doc ./internal/cache` (sin cambios); corregidas las referencias al núcleo (título, nota de bloques, ruta `app_nucleo.go` → `internal/app/nucleo.go`, `nuevoNucleo` → `NuevoNucleo` con el 6º argumento de T22, nombres exportados de los métodos y ruta de los tests). `nuevoEntornoConCache` es helper de test, no firma: se conserva tal cual.
- **`README.md`, `AGENTS.md`, `docs/DESARROLLO.md`, `tareas/00_REGLAS.md`**: 0 referencias obsoletas a `app_nucleo.go` ni al núcleo en `package main` (los "package main" que aparecen son el `DetalleResultado` de `app.go` y la regla P1, ambos vigentes). **No se tocaron.**
- **`docs/firmas/app.md`**: **fuera de "Puedes tocar"**; sus bloques T20–T34 ya llevan la nota de que el bloque **T37** manda, así que no desinforman. No se tocó.
- **Identificadores en minúscula en "Desviaciones…"**: `nucleo.fuentes`→`Nucleo.Fuentes`, `nuevoNucleoPorDefecto()`→`NuevoNucleoPorDefecto(dir)`, `nucleo.buscar`→`Nucleo.Buscar` (discrepancia real con el código, en archivo tocable).
- Sin desviaciones de alcance: ningún archivo tocado fuera de la lista.

**Reporte (regla 15, NO tocado por estar fuera del alcance de T38):** `tareas/00_REGLAS.md` → el diagrama de layout sigue listando **`animeflv/ # paquete animeflv (T18)`** (línea 29) y la regla de dependencias no menciona `scraper/animeav1`; el paquete se retiró en T35. Cambio propuesto: quitar esa línea del árbol y añadir `animeav1` a la dependencia `scraper → internal/app`. Lo dejo para una tarea con `00_REGLAS.md` en "Puedes tocar" con ese objetivo explícito.


## Desviaciones que afectan a tareas futuras (resumen)

> Solo lo que sigue vigente; el texto completo (por tarea) está en `docs/PROGRESO_v1.0.md` (T01–T29) y `docs/PROGRESO_v1.1.md` (T30–T35).

- **Modelos (T03)**: `ResourceType` en **minúsculas** (`hls`, `mp4`, `iframe`); `omitempty` solo en `VideoSource.Subtitles`; `Headers`/`Referer` vacíos serializan `null`/`""`. Cualquier fuente nueva debe emitir esos strings.
- **Normalización y selección (T16/T17)**: `Normalize` corre **dentro** de `Resolver.Resolve` (dedup por recurso sin query pero con fragmento; preserva `Referer`/`Headers`/`Subtitles`) y `SelectBestSource` jerarquiza **HLS > MP4 > iframe > `Type` vacío**, desempate por `Quality` y gana la primera. Consumidor real: `Nucleo.Fuentes` (mejor primero; nombre desde T37).
- **Proxy (T19/T20/T29)**: `NewProxy(nil)` con anti-SSRF, `Register` **solo** con `Type` `hls|mp4` y **sin** `Referer`/`Headers` hacia el frontend (viven en la sesión); `stream.esManifiesto` detecta el manifiesto por el **cuerpo**, no por la extensión. `Sec-Fetch-Site: same-origin` (HLS de AnimeAV1) se declara en `VideoSource.Headers` y `stream/reenvio.go` la inyecta — **nunca** debe limpiarse `Headers` en `Register` (403 en los segmentos).
- **Registro y búsqueda (T20/T34/T35)**: `NuevoNucleoPorDefecto(dir)` registra **`jkanime` + `animeav1`** y `Nucleo.Buscar` los consulta **en paralelo** (timeout 20 s, salida JKAnime primero; error solo si fallan todos; nombres desde T37). `animeflv` **ya no existe** (retirado en T35); `scraper.ErrNoDetails` → `ErrDetalleNoDisponible` se conserva como flujo genérico. El string `"animeflv"` en algunos tests es un provider falso arbitrario, sigue siendo válido.
- **Caché (T21)**: detalle/episodios en memoria con TTL de **10 min**; `GetVideoSources` **nunca** se cachea (regla 9) y `noCache` solo fuerza repetir la extracción.
- **AnimeAV1 (T31–T33)**: vía principal **`GET <ruta>/__data.json`** (devalue: `PageData` + `Unflatten`), `models.Anime.ID` = slug, **lotes de 50** (`tamLote`), dedup de fuentes por **recurso base + idioma** (SUB y DUB son fuentes distintas), HLS derivado **sin petición** (`/play/<32hex>` → `/m3u8/<32hex>`, `/play/*` nunca se pide) con `Type: "hls"` y sus `Headers`; los embeds ajenos (UPNShare/Voe/Byse/MP4Upload) salen **iframe sin resolver**.
- **Progreso (T34)**: `anime_id` es slug compartido entre proveedores → `episode_id` lleva prefijo `<proveedor>:` y toda comprobación de "sigue abierto" compara **proveedor + slug** (no solo el slug).
- **E2E (T28/T35)**: los E2E exigen **peticiones exactas** al sitio falso y cortan hosts ajenos; el de AnimeAV1 demuestra que el proxy inyecta `Sec-Fetch-Site` (segmento por proxy → 206, directo → 403). Si cambia el flujo de peticiones, hay que actualizar el E2E a la vez.
- **Modelos (T40)**: `WatchProgress` lleva ahora 5 campos más (`provider`, `anime_title`, `cover_url`, `episode_number` int, `updated_at` Unix s) **sin `omitempty`** → `historial.json` nuevo siempre serializa las 10 claves (los archivos viejos siguen cargando con los nuevos en cero); T41/T42 rellenan y consumen esos campos, y los modelos TS de `WatchProgress` cambian al regenerar los bindings (T47).
- **Historial (T41)**: `SaveProgress` rellena `UpdatedAt` (`now().Unix()` solo si viene 0; si trae fecha se conserva) y `Provider` (con `ProviderName()`), pero solo al guardar —los archivos viejos se leen tal cual—. `LastPerAnime()` es el listado de la barra lateral (un registro por proveedor+slug; nunca nil; **cerrado ⇒ vacío sin error**) y `DeleteAnime`/`DeleteProvider`/`Clear` devuelven `(n, error)` con `ErrCerrado` si el almacén está cerrado; 0 borrados **no** reescribe el archivo. `GetProgress` y el formato del JSON no cambian.
- **Favoritos (T42)**: `db.OpenFavorites(dir)` (`dir == ""` ⇒ el mismo `ErrDirectorioVacio`) abre `*db.Favorites` sobre `favoritos.json` (JSON indentado, escritura atómica, corrupto ⇒ `.bak`); **no tiene `Close`**. `Add` es idempotente (misma clave ⇒ `false`, conserva `AddedAt`, actualiza `Title`/`CoverURL`; `AddedAt == 0` se sella con `now().Unix()`), `Remove` no reescribe si no existía, `List` nunca nil (copias, `AddedAt` desc y `Key()` asc); clave = `Provider + ":" + AnimeID`. T45 lo abre en `Arrancar` junto al historial.
- **Home JKAnime (T43)**: `scraper.HomeProvider` (interfaz opcional, `Registry`/`Provider` intactos) la implementa `jkanime.Home` con `emision` (JSON `var animes`, clave `data`) + `recientes` (SSR); el `ExceptionResolver` se aplica a **ambas** secciones (1 sola descarga lazy del mapa por proceso, nunca por item) y el límite de lectura es `maxSearchBytes`. `models.Anime` **no tiene `Type`** → los items salen sin tipo. T44 la implementa para `animeav1`; T45 la detecta con aserción de tipo y cachea `home:<proveedor>` 10 min.
- **Home AnimeAV1 (T44)**: `animeav1` cumple `scraper.HomeProvider` en `home.go` con `emision` (`GET /catalogo/__data.json?status=emision`, **guarda `filters.status == 2`** — un slug de status ya no reconocido devuelve todo el catálogo sin error —) y `recientes` (`GET /__data.json`, `latestMedia` con `id` numérico convertido sin notación científica); dedup por slug, 20 items máx. por sección, pausa 300 ms, fallos parciales/total y ctx como en T43 (mensajes "el Home de animeav1 …"). T45 la detecta con la misma aserción de tipo que la de JKAnime y cachea `home:<proveedor>` 10 min.

## Pendientes activos

> Solo lo que una tarea futura debe resolver. Los cerrados/históricos están en `docs/BITACORA.md`; los de T01–T29 en `docs/PROGRESO_v1.0.md`; los de T30–T35 en `docs/PROGRESO_v1.1.md`.

- **Heredados de la v1.0, sin tarea asignada** (detalle en `docs/PROGRESO_v1.0.md` → "Pendientes activos"): (1) título de la ventana (`main.go:20` es el técnico `miapp`); (2) marca de tiempo en `WatchProgress` ("Continuar" elige por número, no por recencia); (3) borrado de registros del historial (`internal/db` no lo expone); (4) verificación visual de los avisos de error que ningún test provoca (hoy: el aviso `episodesError` con un fallo real del backend).
- **Cerrados en T38** (cierre documental de T37, ver sección "T38" arriba): ~~firmas desfasadas de **`docs/firmas/db.md`** y **`docs/firmas/cache.md`**~~ → **corregidas**: `db.Open(dir)` con `ErrDirectorioVacio` (sin `rutaPorDefecto` ni `os.UserConfigDir` en el paquete) y el núcleo documentado como `internal/app` con `NuevoNucleo`/`NuevoNucleoPorDefecto(dir)`. Las referencias a `app_nucleo.go` que siguen en `docs/BITACORA.md`, `docs/PROGRESO_v1.0.md`/`v1.1.md`, `docs/investigacion/animeav1.md` y en los bloques T20–T34 de `docs/firmas/app.md` **son históricas o ya llevan la nota del bloque T37**: no se tocan.
- **Cerrados en T36** (documentados, sin acción pendiente): ~~comentarios de `app.go`/`app_nucleo.go` que presentaban a AnimeFLV como proveedor registrado o fallback~~ → **corregidos** (solo comentarios; ahora describen `jkanime` + `animeav1` con búsqueda combinada en paralelo). ~~Reproducir el HLS de AnimeAV1 end-to-end~~ → **cerrado**: E2E offline + real en verde y **prueba manual con el `.exe` en Windows correcta** (HLS SUB/DUB, SUB/DUB e iframes, búsqueda combinada, mismo slug en ambos proveedores, progreso, JKAnime intacto, sin consola); **release v1.1.0 publicada**.
