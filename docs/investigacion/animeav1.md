# Investigación — AnimeAV1 (T30)

> Informe de investigación de **https://animeav1.com** para valorar un proveedor nuevo
> `animeav1` que sustituya a AnimeFLV. **Sin cambios de código**: este documento, los
> fixtures de `internal/scraper/animeav1/testdata/` y `PROGRESO.md` son los únicos
> archivos tocados.
> Fecha de la investigación: **2026-10-07**. Todo está verificado contra el sitio en vivo
> (sin inventar HTML ni selectores); la evidencia cita el fixture y el patrón con el que
> se comprobó (regex sobre el HTML **crudo**, `curl.exe` con UA de navegador y Playwright
> solo para observar peticiones de red).

## Metodología

- Peticiones con `curl.exe` + `httpx.DefaultUserAgent` (el mismo que usará el scraper),
  guardando directo a archivo: lo que ve `curl.exe` es exactamente lo que verá goquery.
- Verificación de selectores sobre el HTML **crudo** con un probe de regex (`probe.ps1`),
  nunca sobre el DOM renderizado.
- Playwright **solo** para dos cosas: comprobar qué peticiones de red dispara la página
  (¿hay XHR de episodios o de fuentes?) y abrir el desplegable de rangos.
- Sanitizado: escaneo de `csrf|token|nonce|set-cookie|authorization|password|secret|
  signature|expires=|st=|cf_clearance|PHPSESSID|session|jwplayer` sobre los 6 fixtures →
  **cero coincidencias**. Los únicos `localStorage` son el tema/modo oscuro. No hay
  cookies, tokens, CSRF ni URLs firmadas que ocultar.
- Límites: ~40 peticiones al sitio en total, **todas 200** (salvo los 404/403 esperados):
  sin 429 ni `Retry-After`.

---

## P1 — Búsqueda

**Endpoint:** `GET https://animeav1.com/catalogo?search=<query>` (formulario
`<form method="get" action="/catalogo">` con `<input name="search">`; **no existe `/?s=`**).
Respuesta `200 text/html`, `x-sveltekit-page: true`, **sin `Set-Cookie`** y sin challenge
de Cloudflare (`Server: cloudflare`, `cf-cache-status: DYNAMIC`).

**Resultados en el HTML crudo** (fixture `search.html`, 61934 B, 11 tarjetas):

| Elemento | Selector real (verificado con probe) |
|---|---|
| Tarjeta | `<article class="group/item relative text-body">` (11 apariciones) |
| Portada | `<img class="aspect-poster w-full rounded-lg object-cover" src="https://cdn.animeav1.com/covers/195.jpg" alt="Portada de Bleach">` |
| Tipo/categoría | `<div class="rounded bg-line px-2 py-1 text-xs font-bold text-subs">TV Anime</div>` (`TV Anime` / `Película` / `OVA` / `Especial`) |
| Título | `<h3 class="line-clamp-2 text-sm font-bold text-lead">Bleach</h3>` |
| Enlace | `<a class="absolute inset-0 z-20 grid place-content-center" href="/media/<slug>">` |

**Datos completos embebidos** en el `<script>` de hidratación de SvelteKit (índice 47639
del script que arranca en 46512; comprobado con `LastIndexOf("<script")`):

```
return {type:"data",data:{results:[{id:"195",title:"Bleach",synopsis:"…",categoryId:1,slug:"bleach",category:{…}}]}}
```

- **`results[].id` es string** (`"195"`), mientras que `media.id` del detalle es **número**
  (`id:195`) → mapear a `models.Anime.ID` (string) sin ambigüedad y no hacer aritmética.
- `pagination:{currentPage:1,recordsPerPage:20,totalPages:1,totalRecords:11}`.
- Parámetros admitidos (declarados en `uses.search_params`): `page, order, letter, status,
  genre, category, minYear, maxYear, search`. `?page=2` funciona (`currentPage:2`) y el
  catálogo entero está **congelado en `totalRecords:1000` (50 páginas)**.

**Semántica de `search` (importante para la UI):** busca por **título y alias**, **no** por
sinopsis. Evidencia:

- `search=titanes` → 5 resultados, todos `Shingeki no Kyojin…` cuyo `aka` es
  `{"en-us":"Attack on Titan","ja-jp":"…","es-419":"Ataque a los Titanes"}` (detalle de
  `/media/shingeki-no-kyojin`). También `search=ataque` lo encuentra.
- `search=kurosaki` → **0** y `search=secundaria` → **0**, pese a que el sinopsis de Bleach
  (que sí está en el index: `search=bleach` devuelve 11) contiene ambas palabras.

**Sin resultados** (fixture `search_empty.html`, 22298 B, `search=zzzznoexisteestetitulo`):
`totalRecords:0`, `totalPages:0`, `results:[]`, **0 `<article>`** y **ningún texto** de
"sin resultados" en el HTML: el aviso lo pinta el cliente (la app ya tiene el suyo).

---

## P2 — Detalle

**Endpoint:** `GET https://animeav1.com/media/<slug>` → `200` (fixture `detail.html`,
89187 B para Bleach).

**Marcado visible** (todo SSR, verificado en el crudo): `<h1>` título ·
`<h2 class="text-main truncate font-normal">` primer alias · fila de metadatos con
`<span>2004</span>` (año de `startDate`), temporada y estado en texto (`Finalizado` /
`En emisión` / `Próximamente`) · géneros como `<a href="/catalogo?genre=accion">` ·
sinopsis en `div.entry` · portada en `<img>` apuntando a `https://cdn.animeav1.com/covers/195.jpg`
(aunque el campo `poster` del JSON puede venir `null`).

**Datos embebidos** (mismo `<script>` de hidratación):

```
{type:"data",data:{media:{id:195,categoryId:1,title:"Bleach",aka:{"ja-jp":"BLEACH - …"},
 genres:[{id:1,name:"Acción",type:0,slug:"accion",malId:1},…],synopsis:"…",poster:null,
 backdrop:null,trailer:"-tviZNY6CSw",status:0,runtime:null,startDate:"2004-10-05",
 nextDate:null,endDate:null,episodesCount:366,slug:"bleach",malId:"TV",
 category:{id:1,name:"TV Anime",slug:"tv-anime",malId:"TV"},
 episodes:[{id:2892,number:1},{id:2893,number:2},…,{id:3257,number:366}],
 relations:[…]}}}
```

- **`status` numérico → texto** (comprobado en 3 series): `0` → *Finalizado* (Bleach),
  `1` → *Próximamente* (One Piece), `2` → *En emisión* (Battle Spirits, `startDate`
  2026-10-06). Traducir a `models.AnimeDetail.Status`.
- **`aka` es un mapa por locale** (`en-us`, `ja-jp`, `es-419`), no un string.
- **Los episodios NO se piden**: la página trae **los 366 pares `{id,number}`**.
  El servidor solo pinta **50 tarjetas SSR** (`<article class="group/item text-body
  relative">` × 50) y el resto las pinta el cliente con un desplegable
  `button#bits-s3` cuyo texto es `1 - 50`, con rangos `1 - 50, 51 - 100, …, 351 - 366`
  (abiert con Playwright). **No hay XHR**: `?page=2` devuelve exactamente los mismos
  89187 B y la lista de peticiones de red no registra nada nuevo al abrir el menú ni al
  hacer scroll al final (solo `/api/comments?type=media&id=195` y analíticas).
- Miniaturas de episodio: `https://cdn.animeav1.com/screenshots/<mediaId>/<n>.jpg`
  (50 apariciones en el crudo; 200 para `1`, `100` y `366`, 403 si `n` no existe).

**Película** (fixture `detail_movie.html`, 21324 B, `/media/bleach-movie-1-memories-of-nobody`):
`episodes:[{id:14703,number:1}]`, `episodesCount:1`, **1 tarjeta SSR y sin desplegable**
de rangos, más `relations:[…]`. El enlace del episodio es `/media/<slug>/1`.

**Especiales/OVA**: no hay esquema especial de URL. Probados
`/media/boku-no-hero-academia-more` y `/media/dr-stone-ryuusui` (categoría `Especial`,
`malId:"TV Special"`): `episodes:[{id:55628|5808,number:1}]` y enlace `/media/<slug>/1`.
Toda la numeración observada es **entera y ≥ 1**; `relativeNumber` y `season` existen en
el modelo de datos pero salen `null` en todas las muestras.

---

## P3 — Episodios

**URL:** `GET /media/<slug>/<n>` con `n` entero. Comprobaciones con `curl.exe`:

- `/media/bleach/1` → **200** (fixture `episode_page.html`, 42441 B).
- `/media/bleach/367` (fuera de rango) → **404** (4314 B) → tratar como "episodio no
  encontrado", igual que hace JKAnime.
- `/media/bleach-movie-1-memories-of-nobody/1` → **200** (20747 B).

**En el HTML crudo hay 101 enlaces** `href="/media/bleach/<n>"`: el paginador SSR sirve
**siempre la ventana 1…100** (comprobado en `/media/bleach/1` y en `/media/bleach/200`,
ambos con `min=1 max=100` en los enlaces del paginador; **no** es una ventana deslizante),
más la navegación prev/next: `aria-label="Anterior" href="/media/bleach/199"` y
`aria-label="Siguiente" href="/media/bleach/201"` (en el episodio 1, "Anterior" sale
**sin `href`**). No hay `<select>` ni `<option>`.

**La página del episodio también trae los 366 pares `{id,number}`** (y el objeto `media`
completo con sinopsis/géneros/estado), igual que el detalle: `episodes:[{id:2892,number:1},…]`
empezando en el índice 32385 (366 pares en el archivo). O sea que `episode_page.html` es
un **superconjunto** de `detail.html` + `episode` + `embeds` + `downloads`.

**Datos embebidos** (misma página, sin XHR):

```
episode:{id:2892,mediaId:195,title:null,number:1,season:null,relativeNumber:null,
         variants:{DUB:1,SUB:1},filler:false,createdAt:"…",mirrors:void 0, …}
```

`title` y `filler` pueden ser `null`/`false`: no depender de ellos para el UI.

---

## P4 — Fuentes de vídeo

**Todo viene en la propia página del episodio.** Con Playwright sobre `/media/bleach/1`
la única petición de datos es `GET /api/comments?type=episode&id=2892&order=popular`
(comentarios, no hacen falta) más las del reproductor. **No hay endpoint propio de
episodios ni de fuentes**: la única petición aparte es el `__data.json` genérico de
SvelteKit (ver al final de este apartado).

**Forma exacta** (fixture `episode_page.html`):

```
embeds:{DUB:[{server:"HLS",url:"https://player.zilla-networks.com/play/62f0…a3a"},
             {server:"UPNShare",url:"https://animeav1.uns.bio/#vyveci"},
             {server:"Voe",url:"https://voe.sx/e/mwpqlc2nfdwp"},
             {server:"Byse",url:"https://byselapuix.com/e/efptdd773cdu"},
             {server:"MP4Upload",url:"https://www.mp4upload.com/embed-wuf5r9fzzo9q.html"}],
        SUB:[…mismo esquema…]},
downloads:{SUB:[{server:"Mega",url:"https://mega.nz/file/…#…"},
                 {server:"1Fichier",url:"https://1fichier.com/?…"},
                 {server:"MP4Upload",url:"https://www.mp4upload.com/<id>"}],
           DUB:[…]}
```

El `<iframe>` por defecto del DOM es exactamente `embeds.DUB[0].url`
(`<iframe class="aspect-video …" src="https://player.zilla-networks.com/play/62f0…a3a"
title="Episodio Embebido">`), y los botones de servidor/idioma del DOM (`>DUB<`, `>SUB<`
×2 en el crudo) corresponden a esas mismas listas. **No hay `#EXT-X-KEY` ni `.vtt`** en
ninguna parte (búsqueda `\.vtt|subtitles|tracks` → 0 en el fixture).

### Cadena HLS de `player.zilla-networks.com` (la buena)

1. `url` del embed = `https://player.zilla-networks.com/play/<32hex>`.
   El JS del propio player declara `file: ".../m3u8/" + a` donde `a` es el **último
   segmento del path** («if `a.length !== 32` return»), o sea: **derivable sin petición**
   `play/<32hex>` → `m3u8/<32hex>`.
2. `GET /m3u8/<32hex>` → **200**, `Content-Type: application/x-mpegURL`, **14463 B**,
   sin ninguna cabecera especial, `Access-Control-Allow-Origin: *`. Es una **media
   playlist VOD** (`#EXT-X-ENDLIST`), **143 segmentos**, **1 `#EXT-X-MAP`** (init en
   `segs/<id>/init.html`), **0 `#EXT-X-KEY`** y segmentos con URL **absoluta**
   `https://player.zilla-networks.com/segs/<id>/NNN.html` (~3,8 MB cada uno).
3. **Los segmentos exigen `Sec-Fetch-Site: same-origin`** (y solo eso):
   - sin nada → **403**; `Sec-Fetch-Mode` solo → 403; `Sec-Fetch-Dest` solo → 403;
   - `Sec-Fetch-Site: same-origin` → **206** con `Range` (1024 B), y sigue dando 206
     con `Referer` de animeav1 **o con `Referer` ajeno** (el Referer es indiferente);
   - la cabecera **no** puede ponerla el navegador (es "forbidden header"): la pone el
     **proxy local**, igual que ya hace con `Referer`.
4. El `.m3u8` **sí** responde 200 con `Sec-Fetch-Site: same-origin`, así que declararla en
   la sesión no rompe la petición del manifiesto.
5. `GET /play/<32hex>` (la página del player) → **403** sin `Referer` de animeav1 y
   también **403** con solo `Sec-Fetch-Site` → **no hace falta**: nunca se pide esa
   página, solo se deriva la URL del manifiesto.
6. `Content-Type: application/x-mpegURL` hace que `stream.esManifiesto` lo detecte **sin
   depender de la extensión** (la URL no termina en `.m3u8`): el proxy de T19 reescribirá
   el manifiesto tal cual (líneas de URI y `URI=` de `#EXT-X-MAP`) sin tocar código.

**Servidores de solo descarga/iframe:** Mega, 1Fichier, MP4Upload, Voe, Byse, UPNShare.
Ya están cubiertos en `extractors/`: el `ServersExtractor` de JKAnime publica **Mega, VOE
y MP4Upload como iframe** (y resuelve MediaFire a MP4), así que estos enlaces se pueden
publicar **como `iframe` sin resolverlos** (decisión de T33) y, si se quiere, ampliar el
resolver a Mega igual que se hizo con MediaFire.

**`__data.json` como alternativa:** existe `GET /media/bleach/1/__data.json` → 200,
`application/json`, 18447 B (guardado como `episode_data.json`), con **exactamente los
mismos datos**. Está en formato **devalue** de SvelteKit (`{"type":"data","nodes":[…]}`
con referencias por índice), o sea que exige un *unflattener*; el JSON incrustado en el
HTML es JS (`void 0`, funciones flecha), no JSON puro. Ver estrategia en P7.

---

## P5 — Protecciones

| Protección | Evidencia |
|---|---|
| Cloudflare | `Server: cloudflare` + `cf-cache-status` en todas las respuestas, **pero sin challenge**: `curl.exe` recibe 200 directo (sin "Just a moment", sin `cf-mitigated`, sin `Set-Cookie`) |
| Cookies / CSRF / sesión | **No existen**: ninguna respuesta manda `Set-Cookie`; los fixtures no contienen tokens (escaneo a 0) |
| Límite de tasa | **No observado**: ~40 peticiones seguidas (búsquedas, detalles, episodios) → todas 200, sin 429 ni `Retry-After` |
| Bloqueo por Referer | En `animeav1.com`: no (todo responde con cualquier Referer). En `player.zilla-networks.com`: solo `/play/*` exige Referer de animeav1; los segmentos solo exigen `Sec-Fetch-Site` |
| Bloqueo por UA | No comprobado exhaustivamente; todas las peticiones usaron UA de navegador (el que usará `httpx`) |
| Anti-bot en player | `player.zilla-networks.com` está tras Cloudflare y emite `cdn-cgi/rum`; el manifiesto y los segmentos se sirven bien con las cabeceras indicadas |
| Anuncios/telemetría | La página incluye `googletagmanager`, `cdn.runative-syndicate.com` (ads) y `cdn-cgi/rum`; **no afectan al scraper** (no se ejecutan JS) |
| Privacidad de imágenes | `cdn.animeav1.com` responde 200 **sin ningún Referer** → la meta `no-referrer` de `frontend/index.html` sigue siendo compatible |

**Única cabecera que hay que declarar:** `Sec-Fetch-Site: same-origin` en el
`models.VideoSource.Headers` del HLS; `stream/reenvio.go` la inyecta en la sesión
(`for k, v := range ses.headers`) con prioridad sobre el cliente.

---

## P6 — Portadas y CDN

Todo el arte vive en **`https://cdn.animeav1.com`** (los únicos hosts de imagen de la home
son `cdn.animeav1.com`; `animeav1.com/thumbnails/…` devuelve 404 → usar siempre el CDN):

| Recurso | Patrón | Verificación |
|---|---|---|
| Portada | `/covers/<mediaId>.jpg` | `covers/195.jpg` 200 (24625 B) con/sin Referer ajeno; `covers/171.jpg` 200 |
| Miniatura de episodio | `/screenshots/<mediaId>/<n>.jpg` | `195/1.jpg`, `195/100.jpg`, `195/366.jpg` → 200; `195/9999.jpg` → 403 |
| Fondo/hero | `/backdrops/…` | 200 (usado en la home) |
| Episodios de la home | `/thumbnails/<id>.jpg` | `thumbnails/4442.jpg` → 200 (6579 B) |

Ninguna exige Referer ni cookies, así que **la meta `<meta name="referrer"
content="no-referrer">` no rompe nada** (regla de `AGENTS.md`: no eliminarla).

---

## P7 — Estrategia propuesta

1. **Provider nuevo** `internal/scraper/animeav1/` (paquete `animeav1`) que implementa
   `scraper.Provider` completo —a diferencia de AnimeFLV— y se registra en
   `scraper/registry.go` (o sustituye a `animeflv` en `nuevoNucleoPorDefecto()`; decisión
   de producto, ver T35). Usa `httpx.NewClient` (UA por defecto, cookiejar, 15 s) y errores
   `httpx.ScraperError`.
2. **Un solo HTML por operación** (como JKAnime): `Search` = `/catalogo?search=`,
   `GetAnimeDetails` = `/media/<slug>`, `GetVideoSources` = `/media/<slug>/<n>`. Sin
   paginación por XHR y sin `__data.json` obligatorio.
3. **Extracción de datos**: los tres objetos viven dentro del `<script>` de hidratación
   (`{type:"data",data:{…}}`). Plan: localizar el marcador `data:{results:` /
   `data:{media:` / `data:{episode:`, hacer *brace-matching* teniendo en cuenta cadenas
   y escapados, y **convertir el literal JS a JSON** (comillas en claves, `void 0` →
   `null`). El marcado **visible** (`article`, `h3`, `<img src>`, `h1`, géneros, estado)
   se parsea con goquery para los campos de UI que no están en el JSON (p. ej. el texto
   del estado y el alias renderizado). Alternativa documentada: `__data.json` (JSON puro
   pero en formato devalue, necesita *unflattener*) → **se descarta como vía principal**.
4. **Mapeo a modelos**: `Anime{ID: slug, Title, CoverURL: covers/<id>.jpg, Provider:
   "animeav1"}` · `AnimeDetail{ID, Title, Provider, Type: category.slug, Status: texto del
   estado, Genres[] (name), Synopsis, Cover, Total_episodes: episodesCount}` ·
   `Episode{ID: "animeav1:<slug>:episode:<n>", Number: n, Title, Thumbnail:
   screenshots/<mediaId>/<n>.jpg, Kind}` · lotes: todos los episodios en **un lote** (o
   partido de 50 para no bloquear), ya que vienen todos en el detalle.
5. **Fuentes**: HLS de `m3u8/<32hex>` derivado de `play/<32hex>` con
   `Headers: {"Sec-Fetch-Site": "same-origin"}` y `Referer` de animeav1 (inofensivo);
   el resto de `embeds` como `iframe`; `downloads` como `iframe` o ignorados.
   `SelectBestSource` ya prefiere HLS > MP4 > iframe, así que el HLS gana solo.
6. **Sin cambios en `internal/stream`**: `esManifiesto` detecta el manifiesto por
   `Content-Type: …mpegurl` y `Register` ya inyecta `Headers`/`Referer` de la sesión.
7. **Sin límite de peticiones nuevo**: se respeta el ritmo de `httpx`; no se observó
   rate-limit, pero se mantiene el reintentado por defecto.

---

## Fixtures capturados

Todos en `internal/scraper/animeav1/testdata/`, respuesta **cruda** con `curl.exe` + UA de
navegador, sin sanitizar (no había nada que sanitizar: escaneo a 0).

| Archivo | Petición | HTTP | Bytes |
|---|---|---|---|
| `search.html` | `GET /catalogo?search=bleach` | 200 | 61934 |
| `search_empty.html` | `GET /catalogo?search=zzzznoexisteestetitulo` | 200 | 22298 |
| `detail.html` | `GET /media/bleach` | 200 | 89187 |
| `detail_movie.html` | `GET /media/bleach-movie-1-memories-of-nobody` | 200 | 21324 |
| `episode_page.html` | `GET /media/bleach/1` | 200 | 42441 |
| `episode_data.json` | `GET /media/bleach/1/__data.json` | 200 | 18447 |

`episode_data.json` es **extra** (no hace falta para el flujo principal): documenta la
vía alternativa `__data.json` (formato devalue) por si el incrustado en HTML cambia.

## Resumen (10 líneas)

1. AnimeAV1 es SvelteKit SSR tras Cloudflare: todo lo necesario viene en el HTML, **sin
   cookies, sin CSRF, sin rate-limit** y sin challenges para `curl.exe`.
2. Búsqueda en `GET /catalogo?search=…` (20 por página), que casa **título y alias**
   (es-419 incluido), no sinopsis; vacío = 0 tarjetas y sin texto.
3. Detalle en `GET /media/<slug>` con el JSON de `media` **y los N episodios completos**
   (`{id,number}`) embebidos; el servidor pinta 50 tarjetas y el resto es cliente puro.
4. Episodios en `/media/<slug>/<n>` (entero; 404 fuera de rango; película y especial van
   a `/1`), que además reembebe `media`, los 366 `{id,number}` y `episode`+`embeds`+
   `downloads` en la misma página, sin XHR; su paginador SSR saca siempre 1…100.
5. Estado `status`: 0 Finalizado, 1 Próximamente, 2 En emisión; `aka` es mapa por locale.
6. El HLS sale de `player.zilla-networks.com`: `play/<32hex>` se deriva a `m3u8/<32hex>`
   (200, `application/x-mpegURL`, VOD, 143 segmentos, `#EXT-X-MAP`, sin claves).
7. Los **segmentos exigen `Sec-Fetch-Site: same-origin`** (403 sin ella) y el Referer es
   indiferente: se declara en `VideoSource.Headers` y el proxy la inyecta.
8. `/play/*` exige Referer de animeav1 → **no se pide nunca** (se deriva la URL del m3u8).
9. Imágenes solo en `cdn.animeav1.com` (`covers/`, `screenshots/`, `backdrops/`,
   `thumbnails/`) **sin exigir Referer** → la meta `no-referrer` es segura.
10. 6 fixtures crudos capturados (61934/22298/89187/21324/42441/18447 B), verificados con
    probe sobre el HTML crudo; informe completo en `docs/investigacion/animeav1.md`.

## Riesgos

1. **`Sec-Fetch-Site` en el proxy**: es la única cabecera obligatoria y es nueva para
   `internal/stream` (hoy solo inyecta `Referer`+`Headers` genéricos). Si algún día se
   limpia `Headers` en `Register`, el HLS deja de reproducir → cubrir con test.
2. **JS de hidratación frágil**: el JSON de datos está dentro de un `<script>` con sintaxis
   JS (`void 0`, flechas). Un cambio de build de SvelteKit puede romper el *brace-matching*
   → hacer el parser tolerante y con tests sobre fixtures.
3. **Proveedores/CDN de terceros**: `player.zilla-networks.com`, `uns.bio`, `voe.sx`,
   `byselapuix.com`, `mp4upload.com` pueden caer o bloquear; el HLS es el único resuelto,
   el resto van como iframe (sin control de progreso).
4. **El sitio no tiene API estable**: no hay contrato ni versionado; `totalRecords` con
   techo de 1000 y datos embebidos pueden cambiar sin aviso.
5. **Numeración/películas**: solo se verificaron 6 títulos; `relativeNumber`/`season`
   salen `null` y aún no se ha visto ninguna serie con decimales → confirmar en T32 con
   más muestras antes de asumir reglas de numeración.
6. **Sin límite de tasa observado ≠ sin límite**: mantener el ritmo de `httpx` y no
   disparar el detalle de todos los episodios en paralelo.
7. **Sustituir AnimeFLV no es trivial**: exige tocar `app_nucleo.go`, el registro, el
   frontend (`provider === 'animeflv'` en `VistaDetalle.svelte`/`stores`) y los E2E →
   coordinar en T35.

## Propuesta de tareas T31–T35

| Tarea | Contenido | Archivos |
|---|---|---|
| **T31** | Paquete `internal/scraper/animeav1`: `Provider` completo, `Search` sobre `/catalogo` + fixtures `search*.html`, mapa `status` y `aka`, IDs de episodio. Tests con fixtures, sin red | `animeav1/*.go`, `testdata/` |
| **T32** | `GetAnimeDetails` sobre `/media/<slug>`: extracción del literal JS del script (brace-matching + JS→JSON), 366 episodios, miniaturas `screenshots/`, variantes película/especial; fixtures `detail*.html` | `animeav1/*_test.go` |
| **T33** | `GetVideoSources` sobre `/media/<slug>/<n>`: `embeds`/`downloads` → `VideoSource`, derivación `play→m3u8` con `Headers: Sec-Fetch-Site`, iframes del resto, 404 fuera de rango; fixture `episode_page.html` (+ `episode_data.json` como respaldo) | `animeav1/`, `docs/firmas/animeav1.md` |
| **T34** | Integración y verificación con red: smoke `integration` tipo `animeflv_integration_test.go` (búsqueda + detalle + fuentes), revisión de `stream` (que el HLS con `Sec-Fetch-Site` se reproduzca con el `.exe`), fallback/proveedor principal | `*_integration_test.go`, `PROGRESO.md` |
| **T35** | Cambio de proveedor en la app: registro en `nuevoNucleoPorDefecto()`, decisión de sustituir o acompañar a AnimeFLV, avisos del frontend (`provider`), E2E y README/AGENTS | `app_nucleo.go`, `frontend/src/*`, `README.md` |

Orden sugerido estricto (T31 → T35); T31–T33 son comprobables **sin red** con los
fixtures de este informe.

## Lo que no se verificó (deja constancia)

- No se descargó **ningún vídeo** (regla 13): solo manifiesto, cabeceras y probes de 1 byte.
- No se comprobó la app con el `.exe` ni se reprodujo el HLS de AnimeAV1 end-to-end.
- No se muestran ni se resuelven Mega/1Fichier/MP4Upload (quedan como iframe).
- No se probó el comportamiento con VPN/proxy ni la existencia de rate-limit a largo plazo.
