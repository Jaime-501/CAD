## Informe de adds (14/10/2026)

> Investigación **solo de lectura** sobre la publicidad que traen los iframes de terceros.
> Informe completo (metodología, tablas y evidencias): **`docs/investigacion/adds.md`**;
> fixtures y la página de prueba: **`docs/investigacion/adds/`**. No se tocó **ningún**
> archivo de código (regla 7 no se activó).

**Resultado en una línea**: ni el `sandbox` ni un proxy de HTML evitan el anuncio; lo que
funciona es resolver el medio a MP4/HLS en Go, host a host (H1 no sirve como solución
general y queda no concluyente sobre popups tras clic y WebView2, H2 se sostiene con
fricción, H3 se sostiene).

- **Qué se midió**: 13 páginas de episodio de JKAnime + 10 `__data.json` de AnimeAV1
  (inventario de hosts + P1c), 7 hosts × 3 cargas sin sandbox (P2), 7 hosts × 4 variantes
  de `sandbox` (P3) y resolución del medio por host con `Range: 0-0` (P4). ~71 peticiones
  deliberadas, **sin 429**. No se pulsó "play" ni ningún anuncio.
- **P1c (qué elige hoy el selector)**: `SelectBestSource` (`selector.go:26`, HLS 4 > MP4 3
  > iframe 2 > resto) es el del núcleo; sobre las 23 muestras el iframe es la **única**
  opción en **0 de 10** de AnimeAV1 (las 10 traen HLS derivado de zilla) y en **13 de 13**
  de JKAnime con lo que el HTML determina sin red (JKPlayer está presente en las 13 pero
  su aporte necesita la página del player: no concluyente).
- **Hallazgos fuertes**: (1) Voe y Streamtape **detectan el sandbox** y sirven `/blocked`;
  (2) **MP4Upload es RESOLVIBLE limpio** (URL del MP4 en claro, **sin query** — el token va
  en el path — y CDN que responde a `Range` sin mirar Referer, 114 MB confirmados y sigue
  respondiendo a los ~45 min); (3) Voe/Byse/UPNShare/Streamtape son **RESOLVIBLES
  FRÁGILES** (cifrado de 2 caracteres + ROT13 / API + proof-of-work / API cifrada /
  token + captcha); (4) **Mega es NO RESOLVIBLE** (E2E) y, de paso, **no cargó ni un
  dominio de anuncio**; (5) **`sfastwish.com` está parkado** (GoDaddy + Yahoo Ads) pese a
  ser el host con más apariciones en JKAnime; (6) ningún host envía `X-Frame-Options`;
  (7) el anuncio del reproductor vive en el JS del host, **pero** Streamtape trae sus
  scripts de popunder en el HTML crudo del embed y la página de episodio de AnimeAV1
  trae GTM + un script de anuncio (que la app no ejecuta).
- **Fixtures** (`docs/investigacion/adds/`, sanitizados; escaneo final **0 coincidencias
  reales**): `jkanime_var_servers.html`, `animeav1_embeds.json`, `voe_e_redirect.html`,
  `voe_destino_embed.html`, `mp4upload_embed.html`, `byse_embed.html`,
  `upnshare_embed.html`, `streamtape_embed.html`, `mega_embed.html`,
  `cabeceras_embeds.txt` y `sandbox_probe.html` (la página de prueba, con
  `?h=host,host&v=A,B,C,D`). El token de `/d/` de MP4Upload, los `expires`/`ip`/`token`
  de Streamtape, el blob cifrado y el CSRF de Voe y la clave de descifrado de Mega están
  como `TOKEN_DE_PRUEBA`; el enlace de Mega **no se guarda** en el repositorio.
- **Pendiente de producto** (decisión del humano, no técnica): si se quiere invertir en
  extractores de medio directo (T60–T64 abajo) o dejar los iframes como están y solo
  purgar los hosts muertos.
- **Tareas propuestas** (sin renumerar la tabla de arriba; se numeran cuando se arranquen):
  **T60, T61 y T62 hechas** (MP4Upload: resolver neutro `embeds` —T60—, cableado a JKAnime y
  AnimeAV1 —T61— y redacción del token del path en logs —T62—; ver sus bloques aquí abajo) ·
  **T63** reorganización del selector de fuentes (principales primero, iframes como extra;
  hecha, ver bloque al final) · **T70** extractor Voe (portar el descifrado del `loader`) ·
  **T71** extractor Byse (`/api/videos/<id>/embed/details`; el proof-of-work queda fuera) ·
  **T72** UPNShare (API cifrada) · **T73** Streamtape (token + captcha) · **T74** purga de
  hosts muertos del inventario (`sfastwish.com`; vigilar `mixdrop`/`dsvplay`/`vidhidevip`, no
  muestreados) · **T75** (solo si se quiere) `sandbox` **por host** y sin `allow-same-origin`,
  nunca global — **NO PROBADA** en esta investigación: solo se probó en Chromium de escritorio,
  sin clic y sin WebView2.
  **Nota de renumeración (T64):** los números **T64–T69** quedaron reservados para la serie
  **v1.3.0 — episodios incrementales** (rama `feature/episodios-incrementales`; T64 = lógica
  pura de `ventana.ts`, ver bloque al final), por lo que estas propuestas de extractores pasan
  a **T70–T75**. El texto histórico de arriba (T64 = Voe, etc.) no se borra: la numeración
  vigente es la de esta lista.

## T60 — Resolver de MP4Upload en paquete neutro `embeds` (sin cableado)

Hecha el **2026-10-15** (primera tarea del informe de adds que se ejecuta). Crea el paquete **neutro** `internal/scraper/embeds` con la interfaz `Resolver` y el resolver de MP4Upload —el único host RESOLVIBLE limpio del informe— y **sin cablearlo** a ningún proveedor (eso es la tarea siguiente). Depende solo de `models`, `httpx` y `logx`; no importa `jkanime`, `extractors` ni `animeav1`.

**Qué se hizo:**

- **Fixture (requisito 1)**: copiado `docs/investigacion/adds/mp4upload_embed.html` → `internal/scraper/embeds/testdata/mp4upload_embed.html`. Comprobado que contiene el elemento del que se extrae la URL: el `src` del `player.src({ type: "video/mp4", src: "https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4" })` (token ya sanitizado). Sin ejecutar JS.
- **Interfaz `Resolver`** (`embeds.go`): `Name() string`, `CanHandle(url string) bool`, `Resolve(ctx, url, referer string) ([]models.VideoSource, error)`.
- **Resolver MP4Upload** (`mp4upload.go`): descarga el embed con el `httpx.HTTPDoer` inyectado (límite `2 << 20`), extrae la URL del bloque `player.src({ … })` con dos regex (bloque + atributo `src`) y devuelve una fuente `Type: "mp4"`, `Server: "MP4Upload"` y `Referer` = la URL del embed. `CanHandle` reconoce `mp4upload.com`/`www.mp4upload.com` en las formas `/embed-<id>.html` y `/<id>`.
- **Errores tipados** (`errors.Is`): `ErrSinMedio`, `ErrCuerpoGrande`, `ErrURLInvalida`; 4xx/5xx y `ctx` cancelado llegan de `httpx` como `*ScraperError` (`ErrNotFound`/`ErrServerError`/`ErrNilDoer`/`context.Canceled`); nunca pánico.
- **Tests**: `embeds_test.go` (2: interfaz + guardia de dependencias con `go list -deps .`), `mp4upload_test.go` (11: tabla de `CanHandle`, fixture real con `httptest.Server`, forma directa, Referer de la petición, 404, 500, página sin medio, ctx cancelado, cuerpo enorme, sin doer, URL inválida) y `embeds_integration_test.go` (1 con `-tags integration`).
- **Smoke con red**: resuelve un embed real y verifica `Range: bytes=0-0` → **206**. La URL del embed **no se persiste**: sale de la variable de entorno `MIAPP_MP4UPLOAD_EMBED` (id de un episodio real vigente); sin ella el test se salta.

**Pasos de extracción verificados contra el fixture:** descargar el HTML → aislar `player.src({ … })` → leer su `src` (URL del MP4 en claro) → devolver `VideoSource{Type: mp4, Server: MP4Upload, Referer: <embed>}`. Coinciden con `adds.md` (1 ocurrencia de `player.src`, cero `<source>`).

**Cabeceras del CDN y temporalidad:** el CDN responde a `Range: bytes=0-0` con **206** (`nginx`, `Content-Range: bytes 0-0/114919837`, 114 MB). **Desviación del informe**: en vivo (2026-10-15) el CDN **exige Referer** —403 sin él y 206 con cualquier Referer del dominio `mp4upload.com`; 403 con un Referer ajeno—, por eso la fuente lleva `Referer` = la URL del embed (`Headers` vacío). La URL del MP4 **no lleva query** (el token va en el path `/d/<token>/video.mp4`): no hay `expires`/`ip` visibles, pero se trata como temporal (regla 9, no se cachea ni persiste; no se registra completa —`logx.Redact`—).

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** |
| recuento `^--- PASS` (`-count=1 -v`) | **420** (antes **407**; +13 de `embeds`) |
| `go test -tags integration ./internal/scraper/embeds/ -v` | **PASS** con `MIAPP_MP4UPLOAD_EMBED` (smoke real 206); **SKIP** sin ella |
| `go list -deps ./internal/scraper/embeds` (grep jkanime/extractors/animeav1/wailsapp) | **vacío** |
| `npm run check` (frontend) | **0 errores, 0 avisos** (sin cambios) |

**Archivos tocados:** `internal/scraper/embeds/**` (nuevo: `embeds.go`, `mp4upload.go`, `embeds_test.go`, `mp4upload_test.go`, `embeds_integration_test.go`, `testdata/mp4upload_embed.html`), `docs/firmas/embeds.md` (nuevo), `tareas/00_REGLAS.md` (árbol + regla de dependencias: `embeds` neutro) y `PROGRESO.md`. **Nada tocado fuera de "Puedes tocar"** (regla 7 no se activó): `extractors/**`, `jkanime/**`, `animeav1/**`, `internal/app/**`, `stream/**`, `frontend/**`, `app.go`, `main.go`, `wails.json` y `AGENTS.md` intactos.

**Desviaciones:**

- **El CDN sí exige Referer** (el informe lo daba por no exigido): manda la medición en vivo; la fuente lo lleva. Anotado en `docs/firmas/embeds.md`.
- **HTML artificial declarado**: el caso "página sin medio" (embed degradado) y las tablas de `CanHandle` con URLs ficticias; el resto usa el fixture real. No hay fixture real para esos casos.
- **El id del smoke no se persiste**: se pasa en `MIAPP_MP4UPLOAD_EMBED`; ejecutado con un embed vivo (PASS) sin pegar el id en el repositorio ni en `PROGRESO.md`.
- **No se midió** si la URL del MP4 caduca por tiempo o si se ata a la IP (solo el 206 desde la misma sesión/IP).

**Pendientes:**

- **Cablear el resolver a `jkanime` y `animeav1`** (tarea siguiente): convertir el iframe de MP4Upload en fuente MP4 directa desde ambos proveedores; hoy el paquete `embeds` no lo consume nadie.
- Sin tag ni release: no aplica a esta tarea.

---

## T62 — Redactar tokens en el PATH de las URLs de medio (logx + stream)

Hecha el **2026-10-15**, tarea propia nacida del tripwire del req6 de T61: al servir un MP4 resuelto de MP4Upload por el proxy, el log registraba `https://a2.mp4upload.com:183/d/<token>/video.mp4` con el token en el **path** (`logx.Redact` solo cubría la query y `stream.urlSegura` conservaba esquema+host+ruta). **No retoma T61.**

**Qué se hizo:**

- **`internal/logx`**: `Redact` aplica ahora `redactarPath` (tras `reFirma`), que enmascara con `***` los **segmentos opacos** del path de cada URL `http(s)`, conservando esquema, host, puerto, la query y el último segmento con extensión. Un segmento es opaco si no tiene punto y (a) encaja en `^[A-Za-z0-9_+=]{12,}$` o (b) es alfanumérico de 10+ que mezcla letras y dígitos. Los guiones quedan fuera de ambas reglas (no se tocan los slugs). Internos nuevos (sin cambio de API): `reURLPath`, `reSegmentoLargo`, `reSegmentoAlnum`, `redactarPath`, `redactarSegmentos`, `segmentoOpaco`, `mezclaLetrasYDigitos`.
- **`internal/stream`**: `urlSegura` devuelve `logx.Redact(scheme://host/path)` (solo `urlSegura` y sus tests, como autoriza la tarea). Sigue sin query; la URL real que el proxy pide al origen **no se toca**, solo el log.
- **Tests**: `internal/logx/logx_test.go::TestRedactPath` (8 subtests) y `internal/stream/stream_test.go::TestURLSeguraRedactaTokensDelPath`.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** |
| `go build ./...` | OK |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go test -race -count=1 ./...` | **10 de 11 paquetes `ok`**; `internal/app` FAIL por los **2 E2E offline pre-existentes de T61** (salida a `mp4upload.com` en el sitio falso, req7 de T61) — **no** por los logs |
| recuento `^--- PASS` (`-count=1 -v`) | **432** top-level (**430** antes; +2) y **281** subtests (**273** antes; +8) |
| `npm run check` (frontend) | **0 errores, 0 avisos** (sin cambios) |

**Desviaciones:**

- **Diseño (b) elegido frente a (a):** regla general de segmentos opacos en vez de un patrón específico `/d/<token>/`. (a) es frágil (un patrón por CDN), no cubre `/m3u8/<32 hex>` ni los segmentos base64 de zilla, y envejece con cada CDN; (b) cubre el caso real de MP4Upload y los manifiestos/segmentos HLS sin enumerar rutas, y se refina excluyendo segmentos con guiones para no destruir el diagnóstico (los slugs `/bleach/1/`, `demon-slayer-kimetsu-no-yaiba-2019` y `__data.json` salen intactos). Detalle en `docs/firmas/logx.md`.
- **Umbral del blob largo en 12, no 16:** `TOKEN_DE_PRUEBA` mide 15 y los ids reales de MP4Upload, 12.
- **2 FAIL pre-existentes en `internal/app`:** `TestE2EFlujoCompletoOffline` y `TestE2EAnimeAV1FlujoCompletoOffline` fallan en `app_e2e_test.go:958/1785` ("el flujo intentó salir del sitio falso … mp4upload.com") porque T61 aún no ha hecho su req7 (servidor falso del embed). `comprobarLogsSinSecretos` (paso 8 / paso 6) **sí pasa**: el requisito 4 de T62 se cumple. Fuera del alcance de T62 (`internal/app` no es tocable; no se retoma T61).

**Pendientes:**

- Ninguno propio. **T61** sigue en curso (req5–req8); el fix de T62 **desbloquea su req6**.

**Archivos tocados:** `internal/logx/logx.go`, `internal/logx/logx_test.go`, `internal/stream/stream.go` (**solo `urlSegura`**), `internal/stream/stream_test.go`, `docs/firmas/logx.md`, `docs/firmas/stream.md`, `PROGRESO.md`. Nada fuera de "Puedes tocar" (regla 7 no se activó).

---

## T61 — Conectar el resolver de MP4Upload a JKAnime y AnimeAV1

Hecha el **2026-10-15**. Cablea el resolver neutro de embeds (T60) a los dos proveedores: cuando un episodio ofrece **MP4Upload** como iframe (`var servers` en JKAnime; `embeds` SUB/DUB en AnimeAV1), se publica **además** la fuente `mp4` directa, conservando el iframe como respaldo (`SelectBestSource` ya prefiere MP4 > iframe). Sin ejecutar JavaScript y sin cachear ninguna URL resuelta (regla 9).

**Qué se hizo:**

- **Requisito 1 (un fallo no rompe el resto):** `enriquecerConEmbeds` recorre las fuentes ya extraídas y, por cada iframe que un resolver reconozca (`CanHandle`, sin red), llama a `Resolve` con `Referer` vacío y `context.WithTimeout(ctx, tiempoResolucionEmbeds)` (**8 s**). Un fallo registra `slog.Warn` con `logx.Redact` (sin la URL del CDN) y **conserva el iframe**; el `ctx` del flujo se respeta.
- **Requisito 2 (JKAnime):** nuevo `internal/scraper/jkanime/embeds.go`; `GetVideoSources` devuelve `enriquecerConEmbeds(ctx, fuentes)`. Dedup por `claveRecurso` (esquema+host+ruta sin query, con fragmento): la misma URL con otra firma no se duplica. Campo nuevo `JkAnime.Embeds []embeds.Resolver`; `New` inyecta por defecto `embeds.NewMP4Upload(doer)` si es nil.
- **Requisito 3 (AnimeAV1):** `sources.go` enriquece al final de `GetVideoSources` (mismo patrón); campo `AnimeAV1.Embeds` e inyección por defecto en `New`. **Conserva el `Language`** de cada iframe en la fuente resuelta (SUB y DUB siguen siendo **dos** fuentes aunque el CDN sea la misma URL). `animeav1` importa solo `embeds` (ni `jkanime` ni `extractors`).
- **Requisito 4 (Referer):** la `VideoSource` resuelta lleva `Referer` = URL del embed; el proxy (`Register`) lo guarda en la sesión y **no** lo envía al frontend (contrato T29). Verificado en los E2E y en un test nuevo con el proxy REAL.
- **Requisito 9:** ninguna URL resuelta se cachea; `opts`/`NoCache` siguen ignorándose.
- **Fixture y servidores falsos (req7):** los E2E offline sirven el fixture **real** `internal/scraper/embeds/testdata/mp4upload_embed.html` y un CDN falso con `Range` → 206; el transporte traduce los hosts `www.mp4upload.com` y `a2.mp4upload.com:183` al sitio falso y **sigue cortando cualquier otro host** antes de conectar. Vídeo y manifiestos artificiales declarados.

**Requisito 5 (proxy + puerto no estándar) — evidencia:**

- Test nuevo `internal/app/app_nucleo_animeav1_test.go::TestMP4UploadRefererSoloViveEnElProxy`: proxy **real** (`stream.NewProxy`, con `PermitirRedPrivada` solo porque el CDN falso vive en el `127.0.0.1` del `httptest`) + origen en **puerto NO estándar**. Comprueba que (a) la fuente que cruza a Wails sale **sin `Referer` ni `Headers`** (y el JSON no contiene `Sec-Fetch-Site`/`Referer`/`mp4upload.com/embed-`), (b) el origen **sí** recibe el `Referer` del embed y `Sec-Fetch-Site`, y (c) el `Range: bytes=0-0` viaja y el origen responde **206** con `Content-Range`. Si el `Referer` no se guardase en la sesión, el CDN daría 403.
- **`internal/stream` NO necesitó cambios:** su anti-SSRF (`red.go`) valida la **IP resuelta** (`IsLoopback`/`IsPrivate`/`IsLinkLocalUnicast`…), **no el puerto**; un host público como `a2.mp4upload.com:183` pasa. La E2E offline ya pedía el mp4 por el proxy en un puerto no estándar y en producción no hace falta excepción alguna.

**Requisito 6 (el token del path no aparece en logs) — evidencia y tripwire:**

- `comprobarLogsSinSecretos` (usado por los 2 E2E offline y por los 2 E2E de integración) añade el bloqueo de **`TOKEN_DE_PRUEBA`**: el token del path del CDN (`/d/<token>/video.mp4`) no puede quedar en claro en ningún log. El `Referer`, el `Range` y el 206 se comprueban al pedir el mp4 por el proxy (paso "5b" de cada E2E).
- **El guard muerde:** revirtiendo temporalmente `logx.redactarPath` a `return s`, el E2E falla con `el registro "TOKEN_DE_PRUEBA" aparece en los logs: … url=https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4 …`; restaurado, **PASS**. (La redacción del path la añadió **T62**, que desbloqueó este requisito.)

**Requisito 7 (peticiones exactas):** los E2E exigen la lista **exacta** al sitio falso, que ahora incluye `GET /embed-<id>.html` (JKAnime: 1; AnimeAV1: SUB y DUB) y, al pedir el mp4 por el proxy, `GET /d/<token>/video.mp4`. Fuentes: `TestE2EFlujoCompletoOffline` **11 → 12** (2 MP4, 1 HLS, 9 iframe); `TestE2EAnimeAV1FlujoCompletoOffline` **10 → 12** (2 HLS, 8 iframe, 2 MP4). La mejor sigue siendo HLS en ambos.

**Requisito 8 (medición, sin cambiar comportamiento):** en las **2 muestras disponibles** (fixtures reales `episode_page.html` de JKAnime y `episode_data.json` de AnimeAV1), **2/2** episodios ofrecían MP4Upload y **2/2** quedaron con `mp4` resuelto en la lista; **0/2** lo tienen como fuente elegida por `SelectBestSource` (gana el HLS). Se registra en el log `T61 medida: …` de cada E2E.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** (normalizadas a LF las líneas CRLF pre-existentes — fantasma T36) |
| `go vet ./...` · `go vet -tags integration ./...` | limpios |
| `go build ./...` | OK |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (los 2 E2E offline que fallaban en T62 ahora pasan) |
| recuento `^--- PASS` (`-count=1 -v`) | **435** (antes **432**, con los 2 E2E offline en FAIL; +1 test nuevo +2 E2E que pasan). *(La ficha citaba 420: es anterior a T60, que sumó 13 tests.)* |
| `go test -tags integration ./... -run "E2E\|MP4Upload" -v` | **PASS**: `TestE2EFlujoCompletoRedReal` (15,5 s), `TestE2EAnimeAV1RedReal` (3,2 s) y los 2 offline; `TestMP4UploadRedReal` **SKIP** (sin `MIAPP_MP4UPLOAD_EMBED`) |
| `npm run check` (frontend) | **0 errores, 0 avisos** (sin cambios de frontend) |

**Archivos tocados:** `internal/scraper/jkanime/{jkanime.go,embeds.go}` (nuevo `embeds.go`), `internal/scraper/animeav1/{animeav1.go,sources.go}`, `internal/app/{app_e2e_test.go,app_nucleo_animeav1_test.go}` (solo tests/fakes), `internal/scraper/jkanime/embeds_test.go` (nuevo), `internal/scraper/animeav1/embeds_test.go` (nuevo), `docs/firmas/{jkanime,animeav1}.md`, `PROGRESO.md`. **Nada fuera de "Puedes tocar"**: `embeds/**`, `stream/**`, `logx/**`, `frontend/**`, `app.go`, `main.go` y `models/**` intactos. (Las líneas CRLF del fantasma T36 se normalizaron a LF en el worktree —sin cambio de contenido— para que `gofmt -l .` salga limpio.)

**Desviaciones:**

- **`stream` no necesitó cambios** (req5): el anti-SSRF es por IP, no por puerto (ver arriba).
- **El guard del token (req6) solo pasa gracias a T62**; sin la redacción del path de `logx`, el E2E de T61 no podía pasar. Por eso T62 se hizo como tarea propia antes de cerrar T61.
- **Checklist req8 sin muestra nueva:** solo había 2 fixtures de episodio (uno por proveedor); ambos con MP4Upload y resuelto a `mp4`, ninguno elegido como mejor. No se buscó un episodio real donde el HLS faltara (fuera de alcance: T61 no debe cambiar comportamiento).
- **Cuerpo de vídeo artificial** en los CDN falsos (no hay fixture de vídeo; no se descargan vídeos, regla 12): se documenta como artificial.
- **Renumeradas las "tareas propuestas" del informe de adds** en este archivo (T60–T62 ya son tareas reales; las propuestas pasan a T63+).

**Pendientes:**

- **Prueba manual del humano con el `.exe`** (T61 lo pide): reproducción real de un MP4Upload **resuelto** —pausa larga y *seek*— por si el token temporal de la URL caduca o se ata a la IP; si el mp4 se niega tras la pausa, quedará como comportamiento esperado (la URL es temporal, no se refrigera).
- Sin tag ni release: no aplica a esta tarea.

---

## T63 — Selector de fuentes: principales primero, iframes como extra

Hecha el **2026-10-15**. Solo frontend (`etiquetas.ts` + `etiquetas.test.ts`, `SelectorFuente.svelte` y `VistaReproductor.svelte` en lo justo). El selector deja de ser una fila plana: los **principales** (tipo `hls`/`mp4`: jkplayer, MediaFire, el HLS de AnimeAV1 y el MP4Upload resuelto de T61) van primero, y los **iframes** (embebidos de terceros, pueden traer publicidad) quedan debajo en una sección plegable cerrada por defecto. Se arregla así el chip ambiguo `Mp4upload` (iframe con anuncios) vs `MP4Upload` (directo sin anuncios) que hacía elegir el equivocado.

**Qué se hizo:**

- **Clasificación por TIPO, nunca por nombre** (`etiquetas.ts`): `esPrincipal` = `hls`\|`mp4`; `esExtra` = `iframe`. `agruparFuentes(fuentes, filtro)` separa ambos grupos conservando el orden del backend dentro de cada uno y etiqueta cada fila.
- **Sufijo de colisión**: cuando el mismo nombre de servidor (comparado **sin mayúsculas**) aparece en los dos grupos, el principal se marca `· directo` y el extra `· iframe`; si no hay colisión, la etiqueta queda como siempre. `etiquetaFuente` recibe un `sufijo` opcional que se añade al final (con calidad/idioma previos intactos: "MP4Upload · SUB · directo").
- **UI** (`SelectorFuente.svelte`): fila de principales + botón plegable `aria-expanded` con el texto exacto "Otras fuentes (embebidas, pueden incluir publicidad)". **Cerrada por defecto**; se despliega sola si la fuente activa es un iframe o si no hay ningún principal. Funciona con teclado (`:focus-visible`), sin depender de hover (P6). El filtro de idioma sigue en `null` (no se activó).
- **Orden y fallback** (`VistaReproductor.svelte`): la fuente que se abre sigue siendo la del backend (índice 0). El salto automático ahora recorre `ordenReproduccion(fuentes)` —el resto de **principales** primero y los **iframes solo al final**—, porque `Nucleo.Fuentes` solo mueve la mejor a cabeza y el mp4 resuelto de T61 se **añade al final** (podía quedar detrás de los iframes). `elegirFuente` reubica `intentos` en ese orden.

**Pruebas (`etiquetas.test.ts`, esbuild de Vite):** **32** comprobaciones (20 previas + 12 nuevas): clasificación por tipo, colisión `Mp4upload` iframe vs `MP4Upload` mp4 con `· directo`/`· iframe` e índices sobre la lista completa, sin colisión (HLS JKAnime vs VOE, SUB/DUB intactos), lista solo de iframes y lista vacía, y `ordenReproduccion` (principales antes que iframes; la del backend siempre primera).

**Verificación visual (Playwright, `localhost:34115`; sin vídeo real, el proxy no es alcanzable desde el navegador —documentado—):**

- **JKAnime "bleach"** (Ep. 1) → principales `jkplayer`, `Mediafire`, `MP4Upload · directo`; extras al desplegar: `Mega`, `Streamwish`, `Vidhide`, `VOE`, `Filemoon`, `Mixdrop`, `Mp4upload · iframe`, `Streamtape`, `Doodstream`.
- **AnimeAV1 "bleach"** (Ep. 1) → principales `HLS · SUB`, `HLS · DUB`, `MP4Upload · SUB · directo`, `MP4Upload · DUB · directo`; extras: `UPNShare · SUB/DUB`, `Voe · SUB/DUB`, `Byse · SUB/DUB`, `MP4Upload · SUB/DUB · iframe`.
- Pliegue **cerrado por defecto** con un principal activo (`aria-expanded="false"`, extras ocultos) en los dos proveedores; el botón abre/cierra alternando `aria-expanded`; **autoapertura** con iframe activo (el activo acaba en iframe porque el proxy da `ERR_CONNECTION_REFUSED`); **foco de teclado** alcanza el botón en 3 `Tab` con `:focus-visible`. A **700 px** no hay desbordamiento horizontal (selector 636 px, etiquetas largas envuelven). Capturas en `.playwright-mcp/t63-selector-{jkanime,animeav1}-700.png`.
- **Consola**: sin errores nuevos de nuestro código (0 coincidencias de `etiquetas`/`SelectorFuente`/`VistaReproductor`). Lo único que aparece es el `favicon.ico` 404 y el `TypeError` de `wails/ipc.js` (ambos ya presentes en la carga inicial, ajenos a T63), los `ERR_CONNECTION_REFUSED` al proxy `127.0.0.1` (limitación del navegador) y mensajes de anuncios **dentro de los iframes de terceros**.

**Compuerta (resultado real):**

| Comando | Resultado |
|---|---|
| `gofmt -l .` | **sin salida** (los 10 `.go` del fantasma CRLF T36 se normalizaron a LF; `git diff --ignore-cr-at-eol -- '*.go'` **vacío**) |
| `go build ./...` · `go vet ./...` | OK · limpio (sin cambios de Go) |
| `go test -race -count=1 ./...` | **11 paquetes `ok`** (mismo recuento; T63 no toca Go) |
| `etiquetas.test.ts` (esbuild + node) | **OK (32 comprobaciones)** |
| `npm run check` | **0 errores, 0 avisos** |
| `npm run build` | OK (aviso de chunk >500 kB preexistente) |
| `grep -rn wailsjs frontend/src` | **solo `api.ts`** |

**Desviaciones:**

- **`gofmt -l .` marcaba 10 `.go` con el fantasma CRLF (T36)** aunque no se tocó ni una línea de Go; se normalizaron a LF (quitar solo el `\r` final, git no registra cambio real) para dejar la compuerta limpia. Archivos afectados: `internal/app/{app_e2e_test.go,app_nucleo_animeav1_test.go}` y `internal/scraper/{animeav1/{animeav1.go,embeds_test.go,sources.go,sources_test.go},jkanime/{embeds.go,embeds_test.go,get_video_sources_test.go,jkanime.go}}`. Ningún cambio de contenido.
- **Sufijo al final de la etiqueta** (tras `Idioma`/`Calidad`): con idioma queda "MP4Upload · SUB · directo". Es la lectura literal de "sufijo"; sin colisión no se añade nada.
- **Colisión por nombre de servidor sin distinguir mayúsculas** (el caso real JKAnime: iframe `Mp4upload` vs directo `MP4Upload`).
- **Colisión de numeración del informe de adds**: `T63` estaba propuesto como extractor de Voe; el selector de fuentes se queda con `T63` y las propuestas pasan a **T64+** (Voe T64, Byse T65, UPNShare T66, Streamtape T67, purga T68, sandbox T69).
- **Los errores de consola del navegador** (`favicon` 404, `wails/ipc.js` TypeError, `ERR_CONNECTION_REFUSED` del proxy y anuncios dentro de los iframes) son previos/ajenos a T63 (documentados en AGENTS.md); no se introdujo ninguno nuevo.

**Pendientes:**

- **Prueba manual del humano con el `.exe`**: con el proxy real, comprobar que al fallar una fuente el salto recorre los principales antes que los iframes y que el selector se pliega/despliega como se espera; el MP4Upload `· directo` debe reproducirse sin anuncios y sin saltar al `· iframe`.
- Sin tag ni release: no aplica a esta tarea.

