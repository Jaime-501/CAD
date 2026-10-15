# Firmas reales — internal/logx

> Firmas **tal como quedaron** (no como estaban en el plan); las sesiones siguientes se basan en esto.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/logx`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>` (ejemplo de formato en `docs/firmas/httpx.md`).

**Bloques de este archivo y tarea que los creó:**

- `### T01 — internal/logx` → creado por **T01** (Logging y observabilidad base).
- `### T29 — internal/logx` → creado por **T29** (Ajustes finales y refactor): `Redact` ampliado con los parámetros de firma.
- `### T62 — internal/logx` → creado por **T62** (Redactar tokens en el PATH de las URLs de medio): `Redact` enmascara también los segmentos opacos del path.

---

### T01 — internal/logx

- `func New(dev bool) *slog.Logger` — nivel `LevelDebug` si `dev` (texto en stderr), `LevelInfo` en producción (JSON en stderr); envuelve el handler con redacción y lo instala como default del paquete.
- `func Redact(s string) string` — enmascara con `***`: cabeceras `Cookie`/`Set-Cookie`/`Authorization`/`Proxy-Authorization` (resto de la línea), pares JSON `"cookie"|"authorization"|"csrf_token"|"token"|...`, parámetros `?token=`/`?csrf=`/`?_token=`/... en URLs, claves sueltas `csrf=`, `_token=`, `token=`, `apikey=`, `sid=` y **los parámetros de firma que añade T29** (véase su bloque). Case-insensitive; no toca el resto de la cadena (p. ej. `other=1` o `"nombre":"naruto"`).
- `func NewHandler(h slog.Handler) slog.Handler` — `slog.Handler` envolvente que aplica `Redact` al mensaje y a cada atributo (message y atributos en `Handle`, `WithAttrs`, `WithGroup`).
- `func SetDefault(l *slog.Logger)` — fija el logger que usa `Request`.
- `func Request(provider, op, url string, status int, dur time.Duration, extractor string)` — log `http_request` con atributos `provider`, `op`, `url` (redactada por el handler), `status`, `dur`, `extractor`; nivel `Warn` si `status >= 400` o `0`, `Info` en caso contrario.
- Tests: `internal/logx/logx_test.go` (**7 tests**, red de nada).

---

### T29 — internal/logx

- **`Redact` ampliado con las URLs firmadas** (regla del proyecto: tampoco se registra una URL firmada completa). Nuevo patrón `reFirma`, aplicado el último en `Redact`, que enmascara el valor de cualquier parámetro de query que autentique el recurso:
  - players de JKAnime: `?e=`, `?t=`, `?op=` (lo que `httpx.Request` registraba entero al pedir `…/jkplayer/um?e=…`);
  - CDN de HLS/MP4: `?st=`;
  - firma genérica: `?key=`, `?sig=`, `?signature=`, `?sign=`, `?expires=`, `?expiry=`, `?policy=`, `?key-pair-id=`, `?awsaccesskeyid=`;
  - S3/CloudFront: `?X-Amz-Algorithm|Credential|Date|Expires|SignedHeaders|Signature|Security-Token|Key-Pair-Id=`.
  - Case-insensitive; el valor se corta en `&`, espacio o comilla; **no** toca parámetros ajenos (`page=2`, `lang=es`, `other=1` siguen intactos) ni cambia el orden de la query.
- **Sin cambios de firma**: `New`, `NewHandler`, `SetDefault`, `Request` y los demás patrones de `Redact` son los de T01.
- **Quién se beneficia**: `logx.Request` (los `http_url` de `httpx`) y `ScraperError.Error()`, que ya pasaba la URL por `Redact`. El proxy (`internal/stream`) **sigue** registrando sin query con `urlSegura` (no se puede enumerar el nombre de firma de cada CDN).
- **Verificación de flujo completo (T29, paso 7)**: `app_e2e_test.go` añade `capturarLogs(t)` (redirige `slog` y `logx` a un buffer con mutex y los restaura al salir) y `comprobarLogsSinSecretos(t, reg)` → regex `reFirmaEnLog` (query firmada o token con valor de 4+ caracteres = sin redactar) + frases bloqueadas (`SESION_DE_PRUEBA`, `jkanime_session=`, `Set-Cookie`, `_token=`, `csrf-token`, `Authorization:`). Ambos E2E lo ejecutan al final (`TestE2EFlujoCompletoOffline` paso 8 y `TestE2EFlujoCompletoRedReal` paso 6); la captura vacía es un error (la comprobación probaría nada). Comprobado que muerde: sin `reFirma` el offline falla con `?e=TWc1Mlc5WjlBYnNIclZ4…`.
- Tests nuevos: `TestRedactURLFirmada` (4 subtests: player, CDN `st=`, S3/CloudFront, `Signature`/`Expires`/`Policy` + parámetros inocentes intactos) → el paquete pasa de 6 a **7 tests**.

---

### T62 — internal/logx

- **Motivo**: al servir un MP4 resuelto de MP4Upload por el proxy (T61), el log registraba `https://a2.mp4upload.com:183/d/<token>/video.mp4` con el token **en el path**; `Redact` solo cubría la query (y `stream.urlSegura` conservaba esquema+host+ruta). Regla 10 / "nunca URLs firmadas en logs".
- **`Redact` ampliado** (misma firma): tras `reFirma` se aplica `redactarPath`, que enmascara con `***` los **segmentos opacos** del path de cada URL absoluta (`http`/`https`). Conserva esquema, host, puerto, la query y el **último segmento con extensión** (el nombre del archivo, p. ej. `video.mp4` o `10v1x.ts`).
- **Qué es un segmento opaco** (`segmentoOpaco`): un segmento **sin punto** que (a) encaja en `^[A-Za-z0-9_+=]{12,}$` (blob largo: hex/base64url/ids; el umbral es **12**, no 16: los ids reales de MP4Upload miden 12 y el token de prueba `TOKEN_DE_PRUEBA` mide 15) **o** (b) es alfanumérico puro de 10+ caracteres que **mezcla letras y dígitos** (`mezclaLetrasYDigitos`). Los **guiones** quedan fuera de ambas reglas a propósito: separan las palabras de un slug (`demon-slayer-kimetsu-no-yaiba-2019`) y por eso no se enmascaran (el diagnóstico no se destruye).
- **Internos nuevos** (no exportados, sin cambios de API): `reURLPath`, `reSegmentoLargo`, `reSegmentoAlnum`, `redactarPath`, `redactarSegmentos`, `segmentoOpaco`, `mezclaLetrasYDigitos`. Orden de `Redact`: `reJSON` → `reParam` → `reLine` → `reKV` → `reFirma` → `redactarPath`.
- **Diseño elegido (b), no (a)**: se comparó un patrón específico `/d/<token>/` (a) contra la regla general de segmentos opacos (b). Se elige **(b)**: (a) es frágil (un patrón por CDN), no cubre `/m3u8/<32 hex>` ni los segmentos base64 de zilla que el informe ya documenta, y envejece con cada CDN. (b) cubre el caso real de MP4Upload **y** los manifiestos/segmentos HLS sin enumerar rutas; se refina excluyendo segmentos con guiones para no tocar slugs de anime (los casos `/bleach/1/`, `demon-slayer-kimetsu-no-yaiba-2019`, `__data.json` salen intactos).
- **Efecto observable**: `https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4` → `https://a2.mp4upload.com:183/d/***/video.mp4`; `https://player.zilla-networks.com/m3u8/<32hex>` → `…/m3u8/***`; `https://cdn5.ducvomes.com/jp/segs/1/v2x/<b64>/<b64>/10v1x.ts` → `…/jp/segs/1/v2x/***/***/10v1x.ts`. **Nada cambia fuera de los logs**: la URL real que el proxy pide al origen no se toca.
- Tests nuevos: `TestRedactPath` (8 subtests: fixture `TOKEN_DE_PRUEBA`, token realista de 12, token + query de firma, manifiesto HLS, segmento HLS base64, `/segs/<id>/NNN.html`, slug corto y slug largo) → el paquete pasa de 7 a **8 tests**.
