# Firmas reales — internal/stream

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/stream`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T19 — internal/stream` → creado por **T19** (Proxy local de streaming).

---

### T19 — internal/stream

- `type Proxy struct { … }` — cero valor no válido; se construye con `NewProxy`. Campos no exportados: `doer httpx.HTTPDoer`, `cliente *http.Client` (solo si el proxy construyó el suyo), `base/ln/srv` (mutex), `sesiones`, `permitirRedPrivada`.
- `type Option func(*Proxy)` — configuración inyectable en `NewProxy`:
  - `func PermitirRedPrivada() Option` — desactiva **SOLO** el control de IP resuelta del cliente interno (anti-SSRF). **Solo lo usan los tests** (los `httptest.Server` escuchan en 127.0.0.1); sin esta opción el proxy es **restrictivo por defecto**.
  - `func ConTTL(d time.Duration) Option` / `func ConLimiteSesiones(n int) Option` — TTL y cupo de sesiones.
  - `func ConBaseURL(u string) Option` — base para `Register` cuando el `Handler` se monta en otro servidor (AssetServer de Wails). Si no se indica, la fija `Escuchar`.
- `func NewProxy(doer httpx.HTTPDoer, opts ...Option) *Proxy` — **entregable de T19**. Si `doer == nil` (lo que hace la app) el proxy construye su `http.Client` con anti-SSRF; si se pasa un doer se usa **tal cual** y la seguridad del destino queda en el llamador (el paquete no puede meter su control en un transporte ajeno). `opts` se aplican **antes** de construir el cliente, así que `PermitirRedPrivada` afecta al que se crea.
- `func (p *Proxy) Handler() http.Handler` — **entregable**. Ruta **`/proxy`** (cualquier otra → 404). Aplica CORS a **todas** las respuestas y responde él mismo `OPTIONS` (204): el upstream devuelve **405** (comprobado en el sondeo), así que el preflight por `Range` nunca se reenvía.
  - CORS siempre: `Access-Control-Allow-Origin: *` y `Access-Control-Expose-Headers: Content-Range, Accept-Ranges, Content-Length`. En `OPTIONS` añade `Access-Control-Allow-Methods: GET, HEAD, OPTIONS` y `Access-Control-Allow-Headers` (eco de `Access-Control-Request-Headers` si viene; si no, `Range, If-Range, Accept, Accept-Language, Origin, Content-Type`).
- `func (p *Proxy) Register(source models.VideoSource) (proxiedSource models.VideoSource)` — **entregable**. Crea una sesión con `Referer` + `Headers` de la fuente y devuelve la misma fuente con `URL` y `Subtitles[].URL` apuntando al proxy. **La URL del proxy solo lleva `s` (sesión) y `u` (destino): las cabeceras jamás viajan por la URL.** El resto de campos (`Type`, `Quality`, `Server`) se copia sin cambios, pero **T29 vacía `Referer` y `Headers`** en la fuente devuelta: ya viven en la sesión, que es lo único que los lee al reenviar, y esa fuente es la que cruza el puente de Wails hacia el frontend. **Sin base** (no se llamó a `Escuchar`/`ConBaseURL`) la fuente se devuelve intacta, sin registrar.
  - **Requiere `Escuchar()` o `ConBaseURL` antes**: sin base devuelve la fuente **sin registrar** (y sin crear sesión). También devuelve la fuente intacta si `URL` está vacía o no es `http/https`.
  - Los subtítulos se reescriben con la **misma sesión** (así heredan CORS y sus cabeceras); un subtítulo sin esquema http/https se deja como está.
- `func (p *Proxy) Escuchar() (string, error)` — enlaza **`127.0.0.1:0`** (solo loopback, puerto aleatorio), sirve el `Handler` en un goroutine y devuelve la base (`http://127.0.0.1:<puerto>`); idempotente (devuelve la misma base). **Orden recomendado en T20:** `NewProxy(nil) → Escuchar() → Register(fuente)`.
- `func (p *Proxy) Cerrar() error` — detiene el servidor de `Escuchar` (si no hay, `nil`).
- `const TTLDefecto = 90 * time.Minute`, `const LimiteDefecto = 128` — sesiones: caducidad **por deslizamiento** (cada acceso renueva); al llenarse el cupo se purgan las caducadas y, si siguen llenas, la que antes caduca. IDs con `crypto/rand.Text()`.
- `var ErrDestinoBloqueado = errors.New("destino bloqueado por la anti-SSRF")` — causa sentinela: `errors.Is` funciona sobre el error del cliente (envuelto en `*url.Error`/`*net.OpError`).

**Protocolo `/proxy?s=<sesión>&u=<url>`** (`u` con `url.QueryEscape`):

- Métodos: **GET y HEAD** (otros → `405` con `Allow: GET, HEAD`).
- Códigos: `404` sesión inexistente o **caducada** · `400` destino sin host o que no es `http/https` · `403` **anti-SSRF** (`ErrDestinoBloqueado`, destino o redirect a red privada) · `502` fallo de conexión, respuesta nula o manifiesto ilegible/de más de 5 MB · `206` si el origen devuelve 206 (Range reenviado) · resto: se devuelve el status del origen.
- **Reenvío**: `httpx.HTTPDoer.Do` directo con `ctx` de la petición del reproductor (si cierra la conexión se cancela el upstream) y cuerpo en **streaming** (`io.Copy`), sin cargarlo en memoria. Sin `http.Client.Timeout` global (un vídeo dura minutos); el cliente interno pone `ResponseHeaderTimeout=30s` y `Dialer.Timeout=15s`.
- **Cabeceras al origen** (en este orden): lista blanca del cliente `Range, If-Range, Accept, Accept-Language, Origin, If-None-Match, If-Modified-Since` → `User-Agent: httpx.DefaultUserAgent` → cabeceras de la sesión (mandan) → `Referer` de la sesión (el último manda). **Nunca** se reenvían `Cookie`/`Authorization` del reproductor, ni `Accept-Encoding` (lo gestiona el transporte, que descomprime), ni ninguna hop-by-hop. `Proxy` del entorno desactivado en el cliente interno (si no, el control validaría la IP del proxy, no la del destino).
- **Cabeceras de respuesta** (lista blanca): `Content-Type, Content-Range, Accept-Ranges, ETag, Last-Modified, Cache-Control, Content-Disposition` + `Content-Length` **solo si** `resp.ContentLength >= 0 && !resp.Uncompressed && Content-Encoding == ""`. Sin `Location` (el cliente interno ya sigue los redirects y el control de red se aplica a cada salto).
- **Manifiestos** (`.m3u8`/`.m3u` por extensión o `Content-Type` con `mpegurl`, solo `200`): se leen como mucho **5 MB** (`maxManifiesto`) y se reescriben **todas las URIs**: líneas de segmento/sub-playlist (relativas y absolutas, resueltas contra la URL final tras los redirects) y atributos `URI="…"`/`URI=…` de los tags (`#EXT-X-KEY`, `#EXT-X-MAP`, `#EXT-X-MEDIA`, …). Lo que no sea `http/https` (`data:`) queda intacto. Si el cuerpo no empieza por `#EXTM3U` se devuelve **sin tocar**.
- **Subtítulos**: si la extensión es `.vtt` (o el origen ya manda `text/vtt`) se responde con **`Content-Type: text/vtt`**.
- **Logs** (`logx.Request` con `provider="stream"`, `op="proxy"|"register"`): la URL (la del **destino** del reenvío o de la fuente registrada) se reduce a **esquema+host+ruta** (`urlSegura`) → **nunca se registra la query**. Motivo tras **T29**: `logx.Redact` ya enmascara los parámetros de firma que conocemos (`?e=`, `?st=`, `X-Amz-*`, …) y por eso `httpx` sí puede registrar sus URLs enteras, pero el nombre del parámetro firma lo elige cada CDN y no se puede enumerar → aquí se mantiene sin query.

**Anti-SSRF** (`red.go`): `net.Dialer.Control` sobre la **IP ya resuelta** (`net.SplitHostPort` + `net.ParseIP`), invocado antes de conectar **y en cada salto de redirect** (mismo transporte). Rechaza `IsLoopback`, `IsPrivate`, `IsLinkLocalUnicast/Multicast`, `IsMulticast`, `IsUnspecified` y todo `0.0.0.0/8` (IPv4 e IPv6, incluidas mapeadas) → envuelve `ErrDestinoBloqueado`. `ipPublica(ip)` es la función pura.

**Tests** (`internal/stream/`, 23 tests, red de nada, `go test -race` verde; `TestMain` instala logger descartable para `logx.Request`):

- `stream_test.go`: bind solo loopback · **`Cerrar` termina el goroutine de `Serve` y el puerto queda muerto** (`runtime.NumGoroutine()` vuelve a la base, T29) · inyección de `Referer`/`Headers` de sesión + rechazo de `Cookie`/`Authorization` del cliente + la fuente devuelta **sin cabeceras** (T29) · `Range` → `206` con `Content-Range` · sesión caducada/desconocida → `404` · cancelar al reproductor cancela el upstream · `OPTIONS` → `204` + CORS · CORS también en `404`/`400`/`405` · **destino privado rechazado con la opción por defecto (403, upstream sin tocar)** · **redirect a loopback rechazado por defecto** (primer salto ficticio con un transporte truca para no salir a la red) · `400` con destino no http/https · `.vtt` → `text/vtt` · límite de sesiones · doer inyectado · `Register` sin base.
- `manifiesto_test.go`: reescritura de tags/relativas (fixture **artificial**, `testdata/manifiesto_etiquetas.m3u8`) · manifiesto **real** (`testdata/manifiesto.m3u8`, 353 segmentos, capturado en el sondeo) · end-to-end con Referer, `Content-Type` y `Content-Length` · límite de 5 MB → `502` · cuerpo sin `#EXTM3U` → se pasa tal cual.
- `red_test.go`: tabla `ipPublica` (loopback/privada/link-local/no especificada/multicast v4+v6 vs. globales) · `controlRed(false)` activo y `controlRed(true)` nil · dialer del cliente restrictivo bloquea loopback/privadas con `ErrDestinoBloqueado`.
