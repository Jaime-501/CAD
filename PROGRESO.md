    # PROGRESO — Comfort Anime Desktop (T01–T97b hechas)

> Este archivo es la **memoria entre sesiones**. Cada sesión lo lee al inicio y lo actualiza al final.
> Reglas y layout: `tareas/00_REGLAS.md`. Detalle de cada tarea: `tareas/Txx.md`.
> Firmas reales por paquete: `docs/firmas/<paquete>.md` · Bitácora, desviaciones históricas y pendientes cerrados: `docs/BITACORA.md`.
> **Si una firma de `docs/firmas` no coincide con el código, manda el código (`go doc`).**
> **Archivos históricos**: `docs/PROGRESO_v1.0.md` … `docs/PROGRESO_v1.5.md` (uno por versión) y `docs/PROGRESO_embeds_adds.md` (informe de adds y T60–T63). Aquí solo queda lo vigente. **Estado del producto: v1.5.0 preparada** (T1–T94, rama `feature/pantalla-completa`, **sin tag ni release**); la última publicada es la **v1.4.0** (tag `v1.4.0`, `main`). Siguiente trabajo sin ficha: ver "Pendientes activos".

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

- Los fixtures ya capturados están inventariados en los archivos históricos: JKAnime (v1.0) en `docs/PROGRESO_v1.0.md`, AnimeAV1 (T30–T33, 20 fixtures) en `docs/PROGRESO_v1.1.md` y el Home de ambos proveedores (T39) en `docs/PROGRESO_v1.2.md`.

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
| [x] | T01–T29 | v1.0 — base, scraping de JKAnime y E2E | docs/PROGRESO_v1.0.md |
| [x] | T30–T36 | v1.1.0 — AnimeAV1 y cierre | docs/PROGRESO_v1.1.md |
| [x] | T37–T38 | Portabilidad a Android y cierre documental | docs/BITACORA.md |
| [x] | T39–T59 | v1.2.0/v1.2.1 — Home, historial, favoritos, "Siguiente" | docs/PROGRESO_v1.2.md |
| [x] | T60–T63 | Informe de adds y resolutor MP4Upload | docs/PROGRESO_embeds_adds.md |
| [x] | T64–T68 | v1.3.0 — episodios incrementales | docs/PROGRESO_v1.3.md |
| [x] | T69–T89b | v1.4.0 — rediseño y enfoque móvil | docs/PROGRESO_v1.4.md |
| [x] | T91 | Módulo puro de pantalla completa + envoltorios en `api.ts` | — (nota en `docs/PROGRESO_v1.5.md`) |
| [x] | T92 | Sincronizar la ventana nativa con `fullscreenchange` | T91 |
| [x] | T93 | Botón de pantalla completa sobre `.marco` | T92 |
| [x] | T93b | Ocultar el botón nativo de pantalla completa del `<video>` | T93 |
| [x] | T94 | Cierre de la v1.5.0 (diagnóstico de `estados`/`reproductor` + versión y docs) | T91–T93b |
| [x] | T95 | Arreglar el modo `tactiles` de `verificar_ui.mjs` | T94 |
| [x] | T96 | Diagnóstico del pageerror «Cannot read properties of null (reading 'nodes')» | T95 |
| [x] | T97 | Diagnóstico de la deriva de scroll +52 px del detalle (T66/T67) | T96 |
| [x] | T97b | Corregir la deriva de scroll +52 px del detalle (medición síncrona de la sinopsis) | T97 |


## Desviaciones que afectan a tareas futuras (resumen)

> Solo lo que sigue vigente; el texto completo (por tarea) está en `docs/PROGRESO_v1.0.md` (T01–T29), `docs/PROGRESO_v1.1.md` (T30–T35) y `docs/PROGRESO_v1.2.md` (T39–T55).

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
- **Retomar (T46)**: `Nucleo.Retomar`/`Nucleo.SiguienteEpisodio` implementados en `internal/app/retomar.go`; la lista completa de episodios se obtiene con `Detalle` + espera sobre la caché (sin leer internos del stream), nunca una lista parcial. T47 (bindings + `api.ts`) y T55 ("Siguiente ▸") dependen de que T46 se cierre (ya cerrada). Detalle y (a)/(b) verificados: nota `### T46` arriba (la puerta 0 se conserva en una línea en esa misma nota).
- **Bindings (T47)**: `frontend/wailsjs/**` regenerado → `models.WatchProgress` en TS pasó de 5 a **10 campos obligatorios** (T40 sin `omitempty`): todo literal de `WatchProgress` en el frontend debe llevarlos (el de `VistaReproductor.svelte:301` se completó en T47 con excepción autorizada; `cover_url`/`episode_number` provisionales → T49); `api.ts` ya expone los 11 métodos nuevos (`getHomeSection`, `addFavorite`, `removeFavorite`, `listFavorites`, `isFavorite`, `getRecentProgress`, `deleteProgressAnime`, `deleteProviderProgress`, `clearProgress`, `resumeAnime`, `getNextEpisode`) y la constante `RESUME_ACTION`.
- **Historial lateral (T49)**: `frontend/src/historial.ts` es el módulo de la lista de la barra (`$historial`/`$historialEstado`, `cargarHistorial`, agrupación por proveedor y helpers de formato) y `BarraLateral.svelte` ya acepta la prop `onOpen` (clic = T50); `TarjetaHistorial.svelte` pinta `resolviendo`/`mensaje` (T50/T51). El guardado del reproductor rellena `cover_url` desde `currentDetail` y `episode_number` parseando el `episode_id`; la lista se refresca al montar `App` y al **salir** del reproductor (no al entrar ni en otros cambios de vista).
- **Borrado individual (T51)**: `historial.ts` exporta además `borrarAnime(provider, animeID)` y `deshacerBorrado(registros)` (el deshacer re-guarda con `updated_at ≠ 0` → recupera el orden); el toast es **de una sola ranura** (`toasts.ts`/`Toast.svelte`, montado en `App.svelte`): un aviso nuevo sustituye al anterior. `TarjetaHistorial.svelte` tiene la prop `onBorrar` y su raíz ya es un **envoltorio `div`** (acción `deslizar.ts` + botón ✕ hermano del botón de retomar, que por eso no puede ser la raíz); el capture del puntero es **perezoso** (al decidirse el gesto horizontal) porque capturarlo en `pointerdown` rompe el `click` de los botones en Chromium.
- **Botón "Siguiente ▸" (T55)**: cambiar de episodio **dentro del reproductor** ya no remonta el componente: `navegarA('reproductor', sel)` con la vista activa cambia `seleccion` y el watcher local `sincronizarEpisodio` de `VistaReproductor.svelte` reinicia fuentes y pide el siguiente (`getNextEpisode`); el episodio abierto con el botón arranca en 0 aunque tenga progreso guardado (`empezarDesdeCero` en `alMetadatos`, un solo intento). El guardado (throttle 5 s, disparadores) no cambió: con el navegador no hay posición (`currentTime = 0`) y no se guarda — probar el caso con el `.exe` en T56.

- **Pantalla completa (T91)**: `frontend/src/pantallacompleta.ts` (puro, sin imports: `elementoEnPantallaCompleta` + `crearSincronizador`) y `pantallacompleta.test.ts` (50 comprobaciones); `api.ts` añade `ventanaPantallaCompleta(activar)` y `esVentanaPantallaCompleta()` con `WindowFullscreen`/`WindowUnfullscreen`/`WindowIsFullscreen` del runtime (**sin métodos nuevos en Go: siguen 16**). Nota larga en `docs/PROGRESO_v1.5.md`.
- **Pantalla completa (T92)**: `VistaReproductor.svelte` cablea el sincronizador **solo en el `<script>`** (listener `fullscreenchange`, orden de `onDestroy`, `liberar()`); modo `pantallacompleta` A–E (30/30). Nota larga en `docs/PROGRESO_v1.5.md`.
- **Pantalla completa (T93)**: el fullscreen pasa a ser el **`.marco`** (chip «Siguiente ▸» dentro) con botón propio 44×44, `controlsList="nofullscreen"` y `.marco:fullscreen`; casos F–K (63/63). Nota larga en `docs/PROGRESO_v1.5.md`.
- **Pantalla completa (T93b)**: el control nativo del `<video>` se oculta **solo en el `<style>`** (pseudo-elemento `::-webkit-media-controls-fullscreen-button` + `controlsList` de respaldo); caso L (66/66). Nota larga en `docs/PROGRESO_v1.5.md`.
- **Cierre v1.5.0 (T94)**: diagnóstico de `estados` y `reproductor` (dos pasadas por estado, actual vs T89b) → **sin regresión de código** (entorno/datos; informe en `docs/investigacion/ui_regresiones_v1.5.md`), `productVersion` 1.5.0, CHANGELOG/docs y regla de higiene T90. Sin tag ni release. **Desviación:** el `git status` inicial tenía `?? tareas/T94.md` (la ficha, sin trackear); era lo único fuera de sitio y se documentó antes de seguir.
- **Sinopsis del detalle (T97b)**: `sinopsisDesborda` se mide **síncronamente** en cuanto `bind:this` sujeta el párrafo (`VistaDetalle.svelte`), no en un `rAF` propio; el `rAF` queda **solo** para `resize`. No se usa `overflow-anchor: none` ni se reservan 52 px fijos. **No volver a mover la medición a un `rAF`**: reintroduce la deriva de +52 px (`docs/investigacion/deriva_scroll_detalle.md`).

## Pendientes activos

- **Deriva de scroll T66/T67 del detalle** (`+52 px`, 2743 → 2795): **CERRADA en T97b**. Causa (diagnóstico T97, informe en `docs/investigacion/deriva_scroll_detalle.md`): el hueco del botón «Ver más» de la sinopsis (`.ver-mas`: 44 + 8 = **52 px**) no existía en el frame de montaje del detalle porque `sinopsisDesborda` se decidía en un `requestAnimationFrame` **posterior** al `scrollTo` de `App.svelte`; el *scroll anchoring* de Chromium convertía ese crecimiento tardío en +52. **Fix**: medición **síncrona** de la sinopsis al engancharse `sinopsisEl` (`VistaDetalle.svelte`), con el `rAF` conservado **solo** para `resize`; sin `overflow-anchor: none` y sin reservar 52 px. **Medido**: 360×740 Bleach `2743 → 2743` (Δ0, 3/3) y Naruto `2743 → 2743` (Δ0, 3/3+1), `.rejilla` en el viewport −2073 antes y después; 1280×800 `1692 → 1692` (Δ0, sin botón, `.datos` 43,89 sin cambios).
- **Foco del cajón**: el `inert` de T89 impide que `Tab` alcance el contenido de fondo y, desde **T89b**, el velo «Cerrar historial» tampoco es tabulable (`tabindex="-1"` en `frontend/src/BarraLateral.svelte`). Queda **1 fuga**: el `<body>` (respaldo del documento al final de la lista de tabulables). Bajarla a 0 exigiría una trampa de foco explícita; es deuda de producto, no un fallo del cajón.
- **«Episodios siguientes»**: quedó **FUERA** de la v1.4.0 (ver la tabla de correspondencia en `docs/PROGRESO_v1.4.md`) y sigue siendo una idea de producto sin tarea. La **pantalla completa** ya está hecha y cerrada (T91–T94, v1.5.0 preparada).
- **Prueba manual pendiente con el `.exe` release de la v1.5.0 (humano)**: (a) con un episodio HLS/MP4, la barra nativa del vídeo **no** debe mostrar ningún icono de pantalla completa (efecto de T93b, no medible desde Playwright: pseudo-elemento en el shadow DOM del UA) y el botón propio debe seguir funcionando con progreso/volumen/subtítulos intactos; (b) botón «Pantalla completa» → el `.marco` ocupa todo el monitor, el chip «Siguiente ▸» sigue dentro y `aria-pressed`/la etiqueta alternan; (c) segundo clic y Esc salen; el **doble clic en el vídeo no** debe entrar en fullscreen nativo (si entra, interceptar `dblclick` y reportarlo); (d) HLS/MP4 e iframe: «Siguiente ▸» (clic y tecla N) sin salir de pantalla completa ni llamadas extra a la ventana; (e) ventana normal → fullscreen → Esc restaura el tamaño; maximizada → fullscreen → salir vuelve maximizada; Atrás/cerrar la app en fullscreen no deja la ventana bloqueada; segundo monitor (ocupa el monitor de la ventana).
- **Limitación del iframe (producto)**: con una fuente **iframe**, el botón interno del reproductor embebido sigue pidiendo pantalla completa **solo de su iframe** (sin el chip «Siguiente ▸»); el botón propio de la app y la ventana nativa funcionan igual. Heredado de T93, documentado en CHANGELOG y README.
- **`pageerror` del dev server (preexistente)**: `Cannot read properties of null (reading 'nodes')` aparece ya en el Home con el dev server y lo filtra el modo `pantallacompleta`; no viene de T91–T94. **Diagnosticado en T96:** es ruido del dev server (origen `wails/ipc.js`, inyector de Wails sólo en modo desarrollo; 3/3 repeticiones deterministas), no existe en el bundle de producción (0 pageerror) ni en `frontend/src`, y **no afecta al `.exe`**. Informe en `docs/investigacion/pageerror_nodes.md`. Pendiente cerrado; no hay T96b.
- **Sensibilidad a los datos guardados de `estados`, `shell` y `buscador`**: las 4 comprobaciones de `contain-intrinsic-size` necesitan un episodio con progreso parcial en el historial, el toast de `shell` necesita una entrada que borrar y el encabezado de `buscador` necesita algún favorito (con 0 el panel queda oculto en la medición). Sin datos dan **98/102**, un fallo y un fallo; con datos sembrados dan **102/102**, `shell` completo y **61/61**, con el mismo código. No hay regresión: informe en `docs/investigacion/ui_regresiones_v1.5.md`.
- **`tactiles` aborta por un selector obsoleto**: CERRADO en T95. El modo esperaba
  `page.waitForSelector('.btn')` en el Home, pero el Home sano no renderiza `.btn`
  (la actualización es `button.actualizar`; `.btn` solo aparece en estados de error
  con `Reintentar`). Se reescribió la rama `tactiles` de `verificar_ui.mjs` para:
  esperar `.fila .carrusel .tarjeta` (carga del Home), medir `MEDIR_TACTILES` en el
  Home sano, forzar el error rechazando `window.go.main.App.GetHomeSection` (restaurando
  siempre en `try/finally`), pulsar `button.actualizar`, esperar el `.btn` «Reintentar»
  y medir de nuevo. Anti falso verde: imprime conteos de `.btn` y `.chip` por ancho,
  falla si el total medido de un ancho es 0 y falla si el forzado funcionó pero no se
  midió ningún `.btn`. Resultado: 3 anchos (360/768/1280), btn=2 (Reintentar 103.72×44,
  OK), chip=0 (el Home no renderiza `.chip`; las pestañas usan `pestana`), salida 0,
  sin errores de navegador. Prueba negativa (nada que medir) → salida 1. Comparación
  con T89b → mismo resultado. Fuera de `Puedes tocar`: la mención obsoleta en
  `AGENTS.md` no se editó (pendiente de autorización humana).
- **`reproductor` depende de si el proxy de loopback es alcanzable**: da **54/54** en el navegador del contenedor (HLS/MP4 con `ERR_CONNECTION_REFUSED` → caso D por iframe) y **50/52** si el HLS se sirviera; en T93/T93b fue 50/52. No es una regresión de código.
- **Totales de modos con errata en la v1.4**: `cta` son **56** comprobaciones (no 59) y `continuar` **43** (no 46); las cifras altas salen de la tabla «todos los modos» de T89 en `docs/PROGRESO_v1.4.md` y no se corresponden con el script (mismos sitios `check(` desde T77/T80). Reproducido con el código de T89b: 56/56 y 43/43. No se toca el archivo histórico; queda anotado en `docs/investigacion/ui_regresiones_v1.5.md`.
- **`frontend/package.json.md5` versionado y obsoleto**: `wails dev`/`wails build` lo regeneran en cada arranque/build. **Decisión del humano**: commitear una vez el valor regenerado o ignorarlo; por ahora se restaura con `git checkout` en cada tarea. **No se toca** en T94.
- **Verificación con el `.exe` de T65/T67** (revelado animado y restauración del scroll del detalle): el flujo navegador no ejercita el vídeo real (proxy local, AGENTS.md) ni todos los ciclos de recarga; T65 y T67 se verificaron con Playwright (8/8 y 6/6) y por lógica+log. Probar con el `.exe` en la próxima release. Los **fixes T58** (carátula al retomar y contador de favoritos) se dieron por cerrados con la **v1.2.1 publicada**.
- **(4) Verificación visual de los avisos de error** (heredado de la v1.0, detalle en `docs/PROGRESO_v1.0.md` → «Pendientes activos»): los avisos de error que ningún test provoca (hoy: el aviso `episodesError` con un fallo real del backend). Los heredados (1) título de la ventana, (2) marca de tiempo y (3) borrado del historial están **cerrados** (T56, T40/T41 y T51–T52).
- **`Type` de `models.Anime`** sigue sin existir (pendiente de producto heredado: los items del Home y la búsqueda salen sin tipo; si el producto lo quiere hay que ampliar `models` en una tarea nueva).
- **Idea futura (no es tarea): chips de rango de episodios.** T65 descartó paginar; la ventana de `ventana.ts` (T64) queda disponible y probada por si más adelante se quiere una selección por rangos en lugar del mostrado por lotes.

- **Pendiente de producto del informe de adds** (14/10/2026): decidir si se invierte en extractores de medio directo o se dejan los iframes como están y solo se purgan los hosts muertos (ver `docs/PROGRESO_embeds_adds.md`).

---

Bloques movidos (texto literal en los destinos; índice por bloque):
- Ver `docs/BITACORA.md`, sección «Archivo: portabilidad T37/T38».
- Ver `docs/PROGRESO_embeds_adds.md`, sección «Informe de adds y T60–T63».
- Ver `docs/PROGRESO_v1.2.md`, sección «Cierres de la v1.2.x (T56–T59)».
- Ver `docs/PROGRESO_v1.3.md`, sección «Resúmenes largos y ejecución T64–T68».
- Ver `docs/PROGRESO_v1.4.md`, sección «Ejecución T69–T89b».
