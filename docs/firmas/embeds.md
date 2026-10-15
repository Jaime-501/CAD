# Firmas reales — internal/scraper/embeds

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper/embeds`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T60 — internal/scraper/embeds` → creado por **T60** (Resolver de MP4Upload en paquete neutro; `embeds.go` + `mp4upload.go`).

---

### T60 — internal/scraper/embeds

Paquete **neutro** de resolución de embeds de terceros sin ejecutar JavaScript. Solo importa `models`, `httpx` y `logx` (no `jkanime`, ni `extractors`, ni `animeav1`; guardia ejecutable en `embeds_test.go`, `TestEmbedsNoDependeDeProveedores`, que corre `go list -deps .`). **Sin cableado**: ningún proveedor lo usa todavía (es la tarea siguiente a T60).

- `type Resolver interface { Name() string; CanHandle(url string) bool; Resolve(ctx context.Context, url, referer string) ([]models.VideoSource, error) }` — contrato de un resolver. `CanHandle` es barato y sin red; `Resolve` pide el embed y devuelve fuentes normalizadas (`Type` en minúsculas: `hls`, `mp4`, `iframe`). Un resolver **nunca** devuelve fuentes sin URL: si no encuentra el medio, devuelve un error tipado. `referer` es la página que embebe (vacío si se desconoce); no se inventa.
- Errores tipados sentinela (`errors.Is`): `ErrSinMedio` (la página del embed no trae URL de vídeo), `ErrCuerpoGrande` (el HTML supera `maxEmbedBytes`), `ErrURLInvalida` (la URL recibida no es `http(s)` con host). Los fallos HTTP (4xx/5xx) llegan **tal cual** de `httpx` como `*ScraperError` (`errors.Is` contra `httpx.ErrNotFound`/`ErrServerError`/`ErrNilDoer`/`context.Canceled`); `ctx` cancelado también envuelve `context.Canceled`. Nunca hay pánico.
- `type MP4Upload struct { Doer httpx.HTTPDoer }` — resolver de MP4Upload; doer inyectado por campo (regla 4). Valor cero = sin doer ⇒ `httpx.Get` devuelve `ErrNilDoer` sin tocar la red.
- `func NewMP4Upload(doer httpx.HTTPDoer) *MP4Upload` — constructor del resolver.
- `func (MP4Upload) Name() string` → `"MP4Upload"` (etiqueta de `VideoSource.Server`).
- `func (MP4Upload) CanHandle(url string) bool` — reconoce el host `mp4upload.com`/`www.mp4upload.com` y las **dos formas** del informe (`docs/investigacion/adds.md`, P1): el embed `/embed-<id>.html` (con o sin `www`) y la forma directa `/<id>` (una sola ruta alfanumérica de 6+ caracteres). El host del CDN del vídeo (`a2.mp4upload.com`) **no** se reconoce (ya es la fuente, no un embed). Requiere esquema `http(s)`.
- `func (r MP4Upload) Resolve(ctx context.Context, url, referer string) ([]models.VideoSource, error)` — una sola petición `GET url` con `httpx.Get` (`Provider`/`Operation`/`Extractor` = `"MP4Upload"`; `Referer` = `referer` si viene), lee el cuerpo limitado a `maxEmbedBytes = 2 << 20` (+1 byte para detectar exceso) y devuelve **una** fuente:
  - `URL` = la del MP4 del HTML crudo, `Type = models.ResourceMP4`, `Server = "MP4Upload"`, `Headers` vacío.
  - **`Referer` = la propia `url` del embed** (ver "Cabeceras del CDN" abajo): el proxy local lo inyecta al servir el medio.
- Internos (no exportados): `enlaceDePlayer` (aísla el bloque `player.src({ … })` con `rePlayerSrc` y lee su `src` con `reSrcAtributo`; comillas simples o dobles), `urlHTTP` (exige `http(s)` con host), `hostsMP4Upload`, `reEmbedMP4Upload`, `reDirectoMP4Upload`, `rePlayerSrc`, `reSrcAtributo`, `nombreMP4Upload`, `maxEmbedBytes`.

**Pasos de extracción (verificados contra el fixture `testdata/mp4upload_embed.html`, copia de `docs/investigacion/adds/mp4upload_embed.html`):**

1. Descargar el HTML del embed (una petición).
2. Aislar el bloque `var player = videojs("player", {}, function(){ player.src({ … }); … })` → concretamente el interior de `player.src({ … })`.
3. Leer su propiedad `src` (`type: "video/mp4"`), que trae la URL del MP4 **en claro**: `https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4` (fixture, token sanitizado).
4. Devolver esa URL como `VideoSource{Type: "mp4", Server: "MP4Upload", Referer: <url del embed>}`.

Sin ejecutar JavaScript. El informe anota **1 ocurrencia** de `player.src` y **cero** etiquetas `<source>`; el fixture lo confirma. El test `TestMP4UploadResuelveFixtureReal` carga el fixture, sirve el HTML en un `httptest.Server` y exige exactamente esa URL, `Type=mp4`, `Server=MP4Upload`, `Referer` = URL del embed y una sola petición.

**Cabeceras del CDN (medición en vivo, 2026-10-15):**

- **Hallazgo que desvía del informe**: el informe (`adds.md`, P4) daba el CDN por "sin comprobación de Referer", pero en vivo **responde 403 sin Referer y 206 con él**. Cualquier `Referer` del dominio `mp4upload.com` vale (`https://www.mp4upload.com/<embed>`, `https://www.mp4upload.com/`, `https://mp4upload.com/` → 206); un Referer ajeno (`https://example.com/`) → 403. Por eso la fuente lleva `Referer` = la URL del embed. `Headers`: **ninguno**.
- El CDN responde a `Range: bytes=0-0` con `206 Partial Content`, `Server: nginx`, `Content-Type: application/octet-stream`, `Content-Range: bytes 0-0/114919837` (114 MB, idéntico al informe).

**¿Expira o va atada a la IP la URL del MP4?**

- La URL **no lleva query**: ni `expires`, ni `ip`, ni `token` como parámetros; el token va embebido en el **path** (`/d/<token>/video.mp4`), igual que en el informe. No hay, por tanto, un `expires` visible que la caduque ni un `ip=` que la ate explícitamente a una IP.
- No obstante, es una URL **temporal**: se trata como tal (regla general 9: no se cachea ni se persiste) y **nunca** se registra completa en logs (el propio `httpx.Get` la pasa por `logx.Redact`); el smoke imprime el token recortado. No se midió si caduca por tiempo ni si se ata a la IP (solo se comprobó que la URL recién extraída responde 206 desde la misma sesión/IP), así que la cautela es la del proyecto.

**Smoke con red (`embeds_integration_test.go`, `//go:build integration`):**

- `TestMP4UploadRedReal` resuelve un embed real y comprueba con `Range: bytes=0-0` que el CDN responde 206 (sin descargar vídeo). La URL del embed **no está en el repositorio**: se pasa en la variable de entorno **`MIAPP_MP4UPLOAD_EMBED`** (id de un episodio real vigente); sin ella, el test se salta (`t.Skip`).
- Ejecutado con un embed vivo: **PASS** (`Content-Range: bytes 0-0/114919837`). El id usado no se pegó en el repositorio ni en `PROGRESO.md`.

**Tests (recuento):** `internal/scraper/embeds` suma **13** tests top-level sin tag (2 en `embeds_test.go` + 11 en `mp4upload_test.go`) y **1** más con `-tags integration` (`TestMP4UploadRedReal`). Red de nada salvo `httptest.Server` local (el smoke, con tag, sí sale a la red). HTML artificial declarado en `PROGRESO.md` (caso "página sin medio" y tablas de `CanHandle`). `TestMain` propio con logger descartable.
