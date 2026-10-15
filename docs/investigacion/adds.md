# Investigación — Publicidad en los iframes de terceros (adds)

> Objetivo: comprobar con evidencia si se puede evitar la publicidad que traen los
> reproductores embebidos (Voe, MP4Upload, Byse, UPNShare, Streamtape, Mega…) sin
> romper la reproducción, y qué camino es el realista.
>
> Hipótesis a validar:
> **H1** el `sandbox` del iframe corta el anuncio · **H2** se puede resolver el medio a
> MP4/HLS directo y saltarse el anuncio del host · **H3** servir el HTML por un proxy
> propio no es viable.

## Metodología

- Fecha de las mediciones: 14/10/2026. Todo en vivo contra los sitios, sin tocar el código.
- **HTTP**: `curl.exe` con UA de Chrome 131, sin cookies, con `sleep` entre peticiones.
  Solo manifiestos/HEAD/`Range: 0-0` para medir el medio (nunca descargas completas).
- **Navegador**: Playwright (Chromium) sobre la página `adds/sandbox_probe.html` servida
  desde `http://127.0.0.1:34999` (origen localhost, no `file://`). Cada celda crea un
  iframe con `referrerpolicy="no-referrer"` y `allowfullscreen`, como el de la app
  (`frontend/src/VistaReproductor.svelte:659-666`, que hoy **no lleva `sandbox` ni `allow`**).
- **Variantes del probe**: A sin sandbox (estado actual) · B
  `allow-scripts allow-same-origin allow-forms allow-presentation` · C = B +
  `allow-popups allow-popups-to-escape-sandbox` · D = B + `allow="fullscreen"`.
- **No se hizo clic en ningún anuncio** ni se pulsó "play": solo carga y observación.
- **P1c** se resolvió **sin red**: lectura de `SelectBestSource` en el código y análisis
  local de las 23 muestras ya descargadas (más un segundo `Range: 0-0` a la URL de
  MP4Upload para comprobar si caducaba, ~45 min después).
- Presupuesto: ~71 peticiones deliberadas al conjunto de sitios (44 HTTP + 27
  navegaciones del navegador). **Sin 429 ni `Retry-After` en ninguna.**

## P1 — Inventario: quién siembra iframes

Muestra: 13 páginas de episodio de JKAnime (15 descargadas; 2 devolvieron 404 y se
descartaron) + 10 `__data.json` de AnimeAV1 + su home. Conteo por número de páginas
en las que aparece cada host (una misma página puede listar varios).

| Host | JKAnime | AnimeAV1 | Total | Rol en la app |
|---|---:|---:|---:|---|
| `voe.sx` | 13 | 10 | 23 | iframe |
| `www.mp4upload.com` | 10 | 10 | 20 | iframe (+ enlaces de descarga en AV1) |
| `streamtape.com` | 13 | 0 | 13 | iframe |
| `sfastwish.com` | 13 | 0 | 13 | iframe |
| `mega.nz` | 13 | 10 (descargas) | 23 | iframe (JK) / descarga (AV1) |
| `bysekoze.com` → `byselapuix.com` | 10 | 10 | 20 | iframe |
| `vidhidevip.com` | 12 | 0 | 12 | iframe |
| `mediafire.com` | 12 | 0 | 12 | descarga (resuelta en T15) |
| `mixdrop.top` (+ `.is`/`mxdrop.to`) | 13 | 0 | 13 | iframe |
| `dsvplay.com` (+ `d-s.io`) | 13 | 0 | 13 | iframe |
| `1fichier.com` | 7 | 10 | 17 | descarga |
| `animeav1.uns.bio` (UPNShare) | 0 | 10 | 10 | iframe |
| `player.zilla-networks.com` | 0 | 10 | 10 | **HLS directo** (no iframe, T29) |
| `transfer.it` | 0 | 1 | 1 | descarga |

Dos apuntes sobre el HTML crudo:

- **JKAnime no siembra iframes de terceros en el HTML de la página**: el reproductor
  propio es same-origin (`jkplayer/`) y los iframes salen del bloque `var servers` en
  base64, que la app ya publica tal cual. El HTML crudo solo cita un dominio de anuncio
  en texto, sin `<script src>` de anuncio.
- **AnimeAV1 sí**: su HTML de episodio trae `googletagmanager` y un script de anuncio
  externo en claro. La app no lo ejecuta (hace scraping del `__data.json`, no renderiza
  la página del proveedor).

Se muestrearon 7 hosts en P2–P4: los 6 de la lista del encargo (Voe, MP4Upload, Byse,
UPNShare, Streamtape, Mega) más `sfastwish.com` por ser el host con más apariciones en
JKAnime y no estar en esa lista.

## P1c — Qué elige hoy `SelectBestSource` y cuándo es el iframe lo único

**El selector (leído del código, sin modificarlo)**: `SelectBestSource` está en
`internal/scraper/jkanime/extractors/selector.go:26` y es el selector que el núcleo
inyecta (`internal/app/nucleo.go:231`). Ordena por rango de tipo
(`rangoDeTipo`, `selector.go:48`): **HLS 4 > MP4 3 > iframe 2 > resto 1** (un `Type`
vacío o desconocido queda **por detrás del iframe**). Descarta las fuentes sin URL, gana
mayor `Quality` dentro del mismo rango y, a igualdad, se queda con la primera de la
lista. Un iframe **nunca se autoelige** si hay HLS o MP4 en la lista. AnimeAV1 no llama
a esa función, pero ordena sus fuentes "igual que lo elegiría `SelectBestSource`"
(`internal/scraper/animeav1/sources.go:138`), con el grupo que trae HLS primero.

**Sobre las 23 muestras** (13 páginas de episodio de JKAnime + 10 `__data.json` de
AnimeAV1), en cuántos episodios el iframe es la **única** opción:

| Proveedor | Muestras | Iframe = única opción | Detalle |
|---|---:|---:|---|
| AnimeAV1 | 10 | **0 de 10** | Las 10 traen el embed de `player.zilla-networks.com`, que `fuenteDeEmbed` convierte en **HLS directo** (SUB y DUB). `SelectBestSource` elegiría ese HLS; el iframe solo se usaría si el reproductor falla al reproducirlo |
| JKAnime | 13 | **13 de 13 con lo que el HTML permite determinar sin red** | En las 13, `var servers` solo publica **iframes** vía `ServersExtractor` (`servers.go:167`): `mediafire`/`1fichier` se delegan (12 de 13 páginas) y solo MediaFire tiene resolver. **Matiz**: en las 13 está presente el iframe `jkplayer/um` del extractor JKPlayer, que con una segunda petición a la página del player **puede** añadir HLS/MP4 directo; `jk.php` no aparece en ninguna (0 de 13). Ese aporte no es determinable sin red → "Lo que no se verificó" |

Es decir: en AnimeAV1 el iframe **nunca** es la única opción; en JKAnime lo es **si
solo miramos el HTML de la página**, y deja de serlo en los episodios en los que
JKPlayer devuelva un directo.

## P2 — Observación del anuncio (sin sandbox, 3 cargas por host)

Las 3 réplicas de cada host fueron idénticas (mismo número de peticiones y mismos
dominios). "Popups" = pestañas nuevas abiertas **sin clic**; "Nav sup." = intentos de
navegar el frame superior.

| Host | Iframe carga | Redirige a | Peticiones | Dominios externos | Anuncios observados | Popups | Nav sup. |
|---|---|---|---:|---:|---|---:|---:|
| Voe | sí | dominio rotatorio propio (el `voe.sx/e/…` es solo un alias) | 47 | 24 | IMA/DoubleClick (`imasdk`, `securepubads`), JWPlayer (`afs_ads`, `jw-overlays`), Connatix, Criteo, TripleLift, Bidgx, IntentIQ, ID5, `phruk.org`, overlay de "notificaciones" | 0 | 0 |
| MP4Upload | sí | no | 39–40 | 18 | `overlayads` en el DOM + `dopattaoutcant`, `talpiddullity`, `sxlzbz`, `secutorlunts`, `ccopyright`, `tryedesign`, `viradisazo.qpon`, `subduepaler` | 0 | 0 |
| Byse | sí | anida **un segundo iframe** a un host rotatorio (`<4 chars aleatorios>/<id>`) | 49 | 9 | `llvpn.com`, `luugy.com`, `img-place.com`; el jugador interno pide "clic en play para verificar que eres humano" (proof-of-work) | 0 | 0 |
| UPNShare | sí | no | 30 | 10 | IMA SDK + DoubleClick + `2mdn` + Yandex Metrica | 0 | 0 |
| Streamtape | sí | no | 40–43 | 17 | DoubleClick (`adb.ad … skyscraper`), **popunder en claro en el HTML** (`subgumgutwise`, `lumbarssweepup`), `llvpn`/`luugy`, reCAPTCHA v3 invisible | 0 | 0 |
| SFastWish | sí | `/lander` (dominio **parkado** en GoDaddy) | 14 | 4 | Yahoo Ads del parking | 0 | 0 |
| Mega | sí | no | 97 | solo `mega.co.nz` | **ninguno** | 0 | 0 |

Conclusión P2: sin hacer clic no se abren pestañas (el bloqueador de popups del
navegador ya corta `window.open` no ligado a gesto), **pero el anuncio sí se carga y se
pinta**: petición a IMA/DoubleClick/Connatix, `overlayads`, y en Streamtape los scripts
de popunder están en el HTML crudo (es decir, no hace falta ningún JS del host para que
sean parte de la página).

## P3 — Sandbox (tabla host × variante)

Medido con `sandbox_probe.html`. "Player" = se monta el `<video>`/el reproductor del
host; en Voe/MP4Upload se observó el elemento con su `src` y duración (no se dio a
reproducir). Popups: 0 en todas las variantes.

| Host | A (sin sandbox) | B (sandbox sin popups) | C (B + popups escapan) | D (B + `allow="fullscreen"`) |
|---|---|---|---|---|
| Voe | player + stack de ads | **navega a `/blocked`, título "Sandbox no permitido"** | igual que B | igual que B |
| MP4Upload | `<video>` con `src` directo, 1431 s, `overlayads` | **idéntico a A** | idéntico | idéntico |
| Byse | shell + iframe anidado al host rotatorio | idéntico a A (el anidado hereda el sandbox) | idéntico | idéntico |
| UPNShare | shell + IMA cargado | idéntico a A | idéntico | idéntico |
| Streamtape | player + DoubleClick + reCAPTCHA | **navega a `/blocked`** | contexto destruido (sigue navegando) | contexto destruido |
| SFastWish | lander de parking | idéntico | idéntico | idéntico |
| Mega | reproductor E2E (solo hosts `mega.co.nz`) | idéntico | idéntico | idéntico |

Evidencia adicional del navegador, en **todas** las variantes con sandbox:

- Streamtape además falla con
  *"requestStorageAccess: Refused … the document is sandboxed, and the
  'allow-storage-access-by-user-activation' keyword is not set"*.

## P4 — ¿Se puede resolver el medio sin el reproductor del host?

Solo se pidieron cabeceras, manifiestos o `Range: 0-0`. **Ninguna URL firmada o token
se guarda en el repositorio** (están sanitizados en los fixtures; se usaron en tiempo
de ejecución).

| Host | Clasificación | Evidencia |
|---|---|---|
| MP4Upload | **RESOLVIBLE** | El HTML trae la URL del vídeo en claro. `Range: 0-0` → `206`, `Content-Range: bytes 0-0/114919837` (114 MB), `nginx`, y **sí comprueba `Referer`** (medido en T60, 2026-10-15: sin `Referer` → `403`; con un `Referer` del dominio `mp4upload.com` → `206`; con uno ajeno → `403`). **La URL no lleva query**: ni `expires`, ni `ip`, ni ningún otro parámetro — el token va embebido en el path (`/d/<token>/video.mp4`), así que no es una URL firmada visible. Repetida ~45 min después, **volvió a dar `206` con el mismo tamaño total** (idénticos `Last-Modified` y `Cache-Control: no-cache`; sin `Expires` ni `Age`). El elemento del que se extrae es el `src` del `player.src({ type: "video/mp4", src: … })` del setup de videojs: **1 ocurrencia** en la página, **cero** etiquetas `<source>`. Con una sola muestra de HTML no se puede afirmar que la forma del elemento sea idéntica en todos los episodios |
| Voe | **RESOLVIBLE FRÁGIL** | Las fuentes vienen en un `<script type="application/json">` **cifrado**; el descifrador está en `/js/loader.<hash>.js` y es determinista: sustitución de pares de 2 caracteres (`@$`, `!!`, `^^`, `%?`, `#&`, `*~`) + ROT13, y luego `atob` + `JSON.parse`. Portable a Go, pero el nombre del loader rota con hash |
| Byse | **RESOLVIBLE FRÁGIL** | `GET /api/videos/<id>/embed/details` (JSON) devuelve el host del reproductor, que es **el mismo bundle servido desde un dominio espejo rotatorio**; el jugador exige proof-of-work al pulsar play (`/assets/pow-<hash>.js`) |
| UPNShare | **RESOLVIBLE FRÁGIL** | `GET /api/v1/info?id=…` devuelve un blob **hex cifrado** (1584 B binarios, no JSON). El descifrado vive en el bundle; el jugador es vidstack + hls.js con IMA y con cadenas "Banner Ads"/"Push Ads" |
| Streamtape | **RESOLVIBLE FRÁGIL** | El HTML oculta en `#robotlink` un `get_video?id=…&expires=…&ip=…&token=…`. Esa petición devolvió `{"status":500,"msg":"Sorry, error on our side!"}`: hace falta el flujo completo (captcha v3) |
| SFastWish | **NO RESOLVIBLE** | Dominio parkado; ya no hay reproductor |
| Mega | **NO RESOLVIBLE** | Cifrado extremo a extremo: el HTML es un shell y el medio solo se descifra en el cliente de Mega (API `g.api.mega.co.nz` + storage `*.userstorage.mega.co.nz`). No expone MP4/HLS |

## P5 — Veredicto de las hipótesis

| Hipótesis | Veredicto | Por qué |
|---|---|---|
| **H1** — el `sandbox` evita el anuncio | **NO sirve como solución general; NO CONCLUYENTE en el resto** | Voe y Streamtape **detectan el sandbox y sirven `/blocked`**, y cuando el host lo tolera (MP4Upload, Byse, UPNShare, Mega) el anuncio dentro del frame **no desaparece**: `overlayads`, IMA y las peticiones de ad-tech siguen saliendo. Queda **NO CONCLUYENTE** si el sandbox bloquea los popups **tras clic** (no se midió: regla de no tocar anuncios) y **no se midió en WebView2** |
| **H2** — resolver a MP4/HLS directo | **SOSTENIDA, con fricción** | MP4Upload sí, sin nada más. Voe/Byse/UPNShare/Streamtape sí, pero exigen portar un descifrado/flujo por host y se rompen cuando el host rota. Mega no |
| **H3** — proxy de HTML no viable | **SOSTENIDA** | Ningún embed manda `X-Frame-Options` (por eso se pueden embeber), pero el medio no está en el HTML: está tras JS que debe ejecutarse **en el origin del host** (POW de Byse, cifrado de UPNShare, blob+loader de Voe, captcha+token de Streamtape). Un proxy que solo sirve HTML no ejecuta ese JS, y si lo ejecutara tendría que reimplementar cada host y rompería los comprobaciones de origen/CORS/cookies del proveedor. **Impacto en el anti-SSRF**: servir el HTML desde nuestro origen convertiría cada visita del usuario en una petición saliente **nuestra** hacia el host del tercero; los proveedores podrían tachar ese patrón de SSRF (y el usuario quedaría asociado a un origen que no es el del sitio), además de tener que reenviar cookies y cabeceras de origen para que el JS del host siga funcionando |

Evidencia de cabeceras (resumen en `adds/cabeceras_embeds.txt`): **ningún** host envía
`X-Frame-Options`; los CSP relevantes son `upgrade-insecure-requests` (Voe), ausente
(MP4Upload/Byse/UPNShare/Streamtape) y el de Mega, que solo restringe lo que Mega
puede embeber, no lo que lo embebe. Eso confirma que el embebimiento funciona, pero no
da el medio.

## Fixtures capturados

Todos en `docs/investigacion/adds/`, sanitizados (token de `/d/` de MP4Upload,
`expires`/`ip`/`token` de Streamtape, blob cifrado y CSRF de Voe, y la clave de
descifrado de Mega → `TOKEN_DE_PRUEBA`). Escaneo final
(`token|signature|expires=|st=|e=|cookie|nonce|session|csrf` + decodificación de los
`remote` en base64): **0 coincidencias reales**. Los `remote` en base64 que no son de
Mega son los enlaces públicos del proveedor, igual que en
`internal/scraper/jkanime/extractors/testdata/servers.html`.

| Archivo | Qué es |
|---|---|
| `jkanime_var_servers.html` | Bloque `var servers` de una página de episodio de JKAnime |
| `animeav1_embeds.json` | `embeds` (SUB/DUB) extraídos de un `__data.json` de AnimeAV1 |
| `voe_e_redirect.html` | Embed de Voe: redirige por JS al dominio rotatorio |
| `voe_destino_embed.html` | Página real del reproductor de Voe (blob cifrado sustituido) |
| `mp4upload_embed.html` | Embed de MP4Upload con el `<video>` (token sanitizado) |
| `byse_embed.html` | Shell SPA de Byse |
| `upnshare_embed.html` | Shell SPA de UPNShare |
| `streamtape_embed.html` | Embed de Streamtape (popunder en claro + `#robotlink` sanitizado) |
| `mega_embed.html` | Shell del embed de Mega (sin URL ni clave) |
| `cabeceras_embeds.txt` | `X-Frame-Options`/CSP/Server por host |
| `sandbox_probe.html` | Página de prueba de P2/P3 (`?h=host,host&v=A,B,C,D`) |

No se guardó fixture de `sfastwish.com`: `curl.exe` falló dos veces (`000`) contra el
parking; solo se observó en navegador.

## Resumen (9 líneas)

1. El anuncio del **reproductor** no está en el HTML que la app embebe, sino en el JS
   del host (IMA/DoubleClick, Connatix, Criteo, TripleLift, ID5, overlays). **Pero hay
   dos excepciones**: Streamtape trae sus scripts de popunder **en el HTML crudo del
   embed**, y la página de episodio de AnimeAV1 trae GTM + un script de anuncio en claro
   (que la app no ejecuta: hace scraping del `__data.json`, no renderiza la página).
2. **Ningún host envía `X-Frame-Options`**, por eso los iframes funcionan hoy.
3. **El `sandbox` no sirve como solución general**: Voe y Streamtape detectan el sandbox
   y sirven `/blocked`; el resto lo tolera pero sigue sembrando anuncios. Sobre los
   popups tras clic y sobre WebView2, NO CONCLUYENTE.
4. **MP4Upload es el único RESOLVIBLE limpio**: la URL del MP4 sale en claro del HTML
   (sin query: el token va en el path) y el CDN responde a `Range` con un `Referer` del
   dominio `mp4upload.com` (sin él da `403`; medido en T60)
   (114 MB confirmados, y sigue respondiendo a los ~45 min).
5. **Voe, Byse, UPNShare y Streamtape son RESOLVIBLES FRÁGILES**: cada uno exige su
   descifrado o flujo (cifrado de 2 caracteres + ROT13, API + proof-of-work, API
   cifrada, token + captcha).
6. **Mega es NO RESOLVIBLE** (cifrado E2E en cliente) y, de paso, **no sembró ni un
   dominio de anuncio** en las mediciones.
7. **SFastWish ya no existe**: el dominio está parkado y sirve anuncios de Yahoo; es el
   host con más apariciones en JKAnime.
8. Sin clic no se abrieron pestañas en ninguna variante, pero el anuncio **se carga y se
   pinta** igual.
9. **H1 cae como solución general (y queda no concluyente en el resto), H2 se sostiene
   con fricción, H3 se sostiene**: la vía real es resolver el medio en Go host a host,
   no aislar el iframe ni montar un proxy.

## Riesgos

- **Rotación de hosts**: el destino de Voe cambia de dominio, el reproductor de Byse se
  sirve desde un espejo con ruta aleatoria, y el `loader` de Voe rota con hash. Cualquier
  extractor escrito hoy caduca en semanas/meses.
- **Mantenimiento por host**: son 5 desciframes distintos (o 1 extractor por host); sin
  tests de red periódicos no se detecta el cambio.
- **Legal/ToS**: resolver el medio directo salta la monetización del proveedor; es una
  decisión de producto, no solo técnica.
- **Sandbox como mitigación parcial**: **NO PROBADA**. Si algún día se pruebe, hay que
  hacerlo **por host** (solo los que lo toleran: Voe y Streamtape lo rechazan) y sin
  `allow-same-origin`; queda por medir el efecto sobre los popups tras clic y sobre
  WebView2.

## Recomendación final y tareas siguientes

**No intentes ni el sandbox general ni el proxy.** Ataca el medio directo, host a host,
por este orden (coste creciente / beneficio decreciente):

1. **MP4Upload (barato, ya validado)**: extraer la URL del `<video>` del HTML del embed
   con un regex/`goquery` sobre el atributo `src` de la configuración del player.
2. **Voe**: portar el descifrado del `loader` (sustitución de pares + ROT13 + `atob`) y
   cachear el `loader.<hash>.js` que anuncie la página.
3. **Byse**: consumir `/api/videos/<id>/embed/details` y, mientras no se resuelva el
   proof-of-work, dejarlo como iframe.
4. **UPNShare / Streamtape**: dejarlos como iframe (cifrado y captcha) salvo que se
   quiera invertir en el flujo.
5. **Mega**: dejarlo como iframe (E2E) — y de paso es el host menos publicitario.
6. **Higiene**: purgar del inventario los hosts muertos (`sfastwish.com` y, si siguen
   cayendo, `mixdrop`/`dsvplay`/`vidhidevip`, no muestreados aquí) para que la app no
   ofrezca fuentes que ya no existen.
7. **`sandbox` (NO PROBADA)**: no está implementada y aquí solo se probó en navegador
   (no en WebView2). Si se experimenta algún día, tabla por host, sin
   `allow-same-origin`, y midiendo además los popups tras clic.

## Lo que no se verificó

- **No se pulsó "play"** en ningún reproductor (regla: no tocar anuncios). Por eso no se
  midieron pre-rolls que empiezan al reproducir ni popups tras clic; sí se midió todo lo
  que se carga y se pinta antes del clic. En particular, **no se midió si el `sandbox`
  bloquea los popups tras clic**, que era el efecto que más interesaba de H1.
- **WebView2**: toda la medición de P2/P3 fue en Chromium de escritorio (Playwright).
  **No se midió en WebView2**, que es donde embebe la app de verdad.
- **Fullscreen efectivo**: solo se leyó `document.fullscreenEnabled` (true en todas las
  variantes salvo la D de Voe, que además estaba en `/blocked`).
- **Forma del elemento de MP4Upload entre episodios**: solo se analizó un HTML de embed;
  el `player.src({ type, src })` se observó una sola vez.
- **JKPlayer sobre las 13 páginas de JKAnime**: la presencia del iframe `jkplayer/um`
  está confirmada en las 13, pero su aporte real (HLS/MP4 directo) requiere la petición
  a la página del player y no se hizo aquí.
- **Hosts no muestreados**: `vidhidevip.com`, `mixdrop.*`, `dsvplay.com`/`d-s.io`,
  `1fichier.com`, `mediafire.com`, `transfer.it` y `player.zilla-networks.com` (este
  último ya resuelto a HLS directo en T29).
- **`sfastwish.com` con curl**: dos intentos con `000`; solo observación de navegador.
