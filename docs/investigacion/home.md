# Investigación — Home (T39)

> Informe de investigación de **https://jkanime.net** y **https://animeav1.com** para valorar
> qué secciones tipo "En emisión" / "Recién agregados" puede ofrecer cada proveedor en el Home
> de la app (v1.2.0). **Sin cambios de código**: este documento, los fixtures `home_*` de
> `internal/scraper/{jkanime,animeav1}/testdata/` y `PROGRESO.md` son los únicos archivos
> tocados.
> Fecha de la investigación: **2026-10-09**. Todo está verificado contra el sitio en vivo;
> cada afirmación cita el fixture o el comando con el que se comprobó (regex/grep sobre el
> HTML **crudo**, nunca sobre el DOM renderizado).

## Metodología

- Peticiones con `curl.exe` + `httpx.DefaultUserAgent`
  (`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)
  Chrome/131.0.0.0 Safari/537.36`), guardando directo a archivo (`-o`): lo que ve `curl.exe`
  es exactamente lo que verá goquery.
- Pausa ≥ 1 s entre peticiones; **~11 peticiones a JKAnime** (más 3 al CDN de portadas) y
  **~24 a AnimeAV1** (más 2 cargas de página con Playwright), muy por debajo del máximo de
  40 por sitio.
- Verificación de selectores con regex/perl sobre el HTML crudo, en un solo buffer
  (`tr -d '\r\n'` o `perl -0777`) porque el HTML viene en una sola línea.
- Playwright MCP **solo** para dos cosas: (1) abrir el desplegable "Ordenar por" y el
  diálogo "Filtros" de `/catalogo` de AnimeAV1 cuyo contenido no está en el HTML crudo;
  (2) observar la URL que genera el filtro de estado (`/catalogo?status=emision`).
  `browser_snapshot` no se usó como fixture (devuelve accesibilidad, no HTML).
- Sanitizado: escaneo `csrf|token|nonce|set-cookie|authorization|session|st=|e=|cf_clearance`
  sobre los 11 fixtures nuevos → los 5 de JKAnime traían **un CSRF real** en
  `<meta name="csrf-token" content="…">` (40–43 chars, 1 aparición cada uno); sustituido por
  `TOKEN_DE_PRUEBA` con perl (`s/\Q<token>\E/TOKEN_DE_PRUEBA/g`). Tras la sustitución, el
  único `csrf` que queda es el **nombre** del meta (el valor es `TOKEN_DE_PRUEBA`); 0
  coincidencias de `set-cookie`, `authorization`, `cf_clearance`, `st=`, `e=` con valor. Los
  6 de AnimeAV1 traían 0 (coincidencias solo de scripts de analítica/ads, p. ej. `"SessionId"`
  dentro de JS de terceros, sin valor de sesión real).

---

## JKAnime

### P1 — Páginas con secciones candidatas

| Página | Método | curl pasa sin challenge | Evidencia |
|---|---|---|---|
| `GET /` (portada) | 200, 127860 B | **Sí** (sin "Just a moment", sin `cf-mitigated`, sin `Set-Cookie`) | fixture `home_portada.html` |
| `GET /directorio` | 200, 71488 B | Sí | fixture `home_directorio.html` |
| `GET /directorio?estado=emision` | 200, 70132 B | Sí | fixture `home_directorio_emision.html` |
| `GET /directorio?filtro=popularidad` | 200, 68897 B | Sí | fixture `home_directorio_popularidad.html` |
| `GET /estrenos` | 200, 37969 B | Sí | fixture `home_estrenos.html` |

La portada contiene, en este orden de secciones (verificado con grep de los títulos
`<h4>`):

1. **"Continuar mirando..."** — requiere sesión; el servidor pinta un `div#recientes_div`
   **vacío** con un loader (`<div id="loader"><div class="lds-ellipsis">`) y un botón
   `#loadmore` con `data-tipo="recientes"`, `data-guardado="https://jkanime.net/guardado"`,
   `data-historial="https://jkanime.net/historial"`. **Los items los pinta JS tras un XHR
   autenticado** → no usable para un Home anónimo.
2. **"Programación"** — `<h4><i class="ti ti-clock-play"></i> Programación</h4>`, con pestañas
   `#animes-tab` / `#donghuas-tab` / `#ovas-tab` (paneles `#animes`, `#donghuas`, `#ovas`).
   Conteo por panel (probe con node sobre el crudo): **animes 32 enlaces**, **donghuas 32**,
   **ovas 8** (todos `href="https://jkanime.net/<slug>/<n>/"`, es decir enlaces de
   **episodio**, no de detalle). Sin paginación.
3. **"Animes recientes"** — `<h4>Animes recientes</h4>`; **32 tarjetas SSR** en el crudo
   (32 apariciones de `custom_thumb_home`), **sin paginación** y sin enlace "Ver todo" (probe:
   los 32 hrefs distintos de la sección son todos `/<slug>/` de detalle).
4. **"Top animes"** — `<h4><i class="ti ti-clock-play"></i> Top animes</h4>` + `<a class="vt"
   href="https://jkanime.net/top">Ver todo</a>`; **10 tarjetas** (`div.col.toplist`) con rango
   `data-rank="1"…"10"` y likes `div.card-badge > i.ti-thumb-up`.

**Directorio con filtros**: el formulario es
`<form action="https://jkanime.net/directorio" method="get">` con estos **selects y valores
exactos** (extraídos del crudo, `home_directorio.html`):

| name | valores (value → texto) |
|---|---|
| `filtro` | `` → "Por fecha" · `nombre` → "Por nombre" · `popularidad` → "Por popularidad" |
| `letra` | `A`…`Z` (sin value explícito: el valor es la letra) |
| `demografia` | `` · `nios` · `shoujo` · `shounen` · `seinen` · `josei` |
| `categoria` | `` · `donghua` · `latino` |
| `tipo` | `` · `animes` · `peliculas` · `especiales` · `ovas` · `onas` |
| `estado` | `` ("Ninguno") · **`emision`** ("En emisión") · `finalizados` ("Finalizado") · `estrenos` ("Por Estrenar") |
| `temporada` | `` · `invierno` · `primavera` · `verano` · `otoño` |
| `orden` | `asc` ("Ascendente") · `` ("Descendente", el seleccionado) |

Paginación: `?p=<n>` (la paginación del propio sitio usa `?p=`; ej. `?p=2`…`?p=168`).

**`/estrenos`**: 10 tarjetas SSR (`div.mb-4.d-flex…div.card`) con
`href="https://jkanime.net/<slug>/"`, portada `cdn.jkdesa.com/assets/images/animes/image/
<slug>.jpg`, badges `badge TV badge-primary">Serie` + `badge ml-2 badge-jk">👍 <n>` y una
línea `<b>Estreno:</b> 2026-10-11`. Filtros propios: `filtro` (`` "Por popularidad" /
`fecha` / `nombre`), `tipo`, `orden`. **No hay `estado`** aquí.

### P2 — ¿Los items vienen en el HTML crudo o los pinta JS?

| Sección | ¿En el crudo? | Detalle |
|---|---|---|
| Programación | **Sí, SSR** | tarjetas `<a href="…/<slug>/<n>/">` con `span.badge.badge-primary">Ep <n>` y día `span.badge.ml-2.badge-secondary">…</span>`. Días observados: `Hoy`, `Ayer`, `Lunes 05`, `Martes 06`, `Miércoles 07`, `Sábado 03`, `Viernes 02` |
| Animes recientes | **Sí, SSR** | 32 tarjetas `custom_thumb_home` (ver selectores en P3) |
| Top animes | **Sí, SSR** | 10 tarjetas `div.col.toplist` |
| Directorio | **JSON embebido, no tarjetas** | el servidor **no** pinta tarjetas (las plantillas con `{slug}`/`{img}` son strings JS); pinta **`var animes = {…};` dentro de un `<script>`** y el cliente los renderiza con `render_animes()`. El JSON **sí está en el HTML crudo**: probe `perl -0777` → el segmento `var animes = ` … `;var mode` mide 25848 B, **JSON estricto** (`JSON.parse` OK), **0 caracteres `<`** dentro del JSON y sin `</script>` interno → goquery puede extraerlo del `.Text()` del `<script>` **sin XHR y sin JS** |
| Continuar mirando | **No** | `div#recientes_div` vacío + loader; XHR autenticado (`/guardado`, `/historial`) |

No hizo falta Playwright para localizar endpoints: todos los datos útiles ya están en el
crudo.

### P3 — Selectores reales de la tarjeta

**"Animes recientes" (portada)** — los 32 items casan con este patrón (probe con regex sobre
`home_portada.html`, 32/32):

```html
<div class="p-3 d-flex">
  <div class="custom_thumb_home">
    <a href="https://jkanime.net/holy-knight/">
      <img src="https://cdn.jkdesa.com/assets/images/animes/image/holy-knight.jpg"
           class="img-fluid rounded-start" alt="Holy Knight"></a></div>
  <div class="card-body-home">
    <h5 class="card-title"><a href="https://jkanime.net/holy-knight/">Holy Knight</a><span></span></h5>
    <div class="card-info">
      <p class="badge finished">Concluido</p>
      <p class="badge OVA"> OVA </p></div>
    <p class="card-text ep"><i class="ti ti-clock-hour-4"></i> Miercoles, 21 de Marzo de 2012 </p>
  </div>
</div>
```

- **slug**: `a[href]` de `.custom_thumb_home` (o de `.card-title a`), URL absoluta
  `https://jkanime.net/<slug>/` → slug = path sin barra final (mismo formato que `Anime.ID`
  de la búsqueda, ver P4).
- **título**: `h5.card-title a` (o `img[alt]`; coinciden en las 32 muestras).
- **portada**: `img[src]` → `https://cdn.jkdesa.com/assets/images/animes/image/<slug>.jpg`
  (el nombre de archivo es el slug en las 32 muestras).
- **tipo**: `p.badge` con clase del tipo (`OVA`, `Movie`, `TV`, `ONA`) y texto espaciado
  (` OVA `, ` Pelicula `…). Distribución observada: Serie 11, Pelicula 9, OVA 7, ONA 5.
- **estado**: `p.badge` con clase `finished`/`currently` y texto `Concluido`/`En emision`.
  Distribución: Concluido 25, En emision 7.
- **fecha**: `p.card-text.ep` con una fecha legible (`Miercoles, 21 de Marzo de 2012`); es
  la fecha de emisión del título, **no** un número de episodio (a pesar de la clase `ep`).
- **número del último episodio**: **no** aparece en esta sección. Solo "Programación" trae
  `Ep <n>`.
- **cantidad**: 32 items, **sin paginación**.

**"Programación"** — tarjeta (32 en el panel `#animes`):

```html
<a href="https://jkanime.net/tokyo-revengers-santen-sensou-hen/2/">
  <div class="d-thumb">
    <img class="card-img-top" src="https://cdn.jkdesa.com/assets/images/animes/image/tokyo-revengers-santen-sensou-hen.jpg"
         data-animepic="…" alt="Tokyo Revengers: Santen Sensou-hen - 2">
    <div class="badges badges-top">
      <span class="badge badge-primary">Ep 2</span>
      <span class="badge ml-2 badge-secondary"><i class="ti ti-clock-hour-5"></i> Hoy</span>
    </div></div>
  <div class="card-body d-flex flex-column">
    <h5 class="strlimit card-title">Tokyo Revengers: Santen Sensou-hen</h5></div>
</a>
```

- slug y episodio salen del `href` (`/<slug>/<n>/`); título en `h5.card-title`; portada en
  `img[data-animepic]` (**no** en `src`: en algunas tarjetas el `src` es un
  `jkvideo_… .jpg` de CDN distinto). Cuidado: el `alt` incluye ` - <n>`.
- Enlace de episodio, no de detalle → hay que reconstruir `https://jkanime.net/<slug>/`.

**"Top animes"** — tarjeta (10 items):

```html
<div class="col toplist mb-3"><div class="card">
  <a href="https://jkanime.net/one-piece/">
    <div class="card-img"><img class="card-img-top" src="…/image/one-piece.jpg" alt="One Piece"></div>
    <div class="card-badge"><i class="ti ti-thumb-up"></i> 13071</div>
    <div class="card-body">
      <div data-rank="1" class="ranking"><span>#1</span></div>
      <h5 class="card-title">One Piece </h5> …
```

**Directorio (JSON embebido)** — 30 objetos por página; claves del item (probe `JSON.parse`):
`id, title, synopsis, image, studios, slug, type, status, url, estado, tipo, base64_id,
short_title`. **No** hay campo de episodios (la plantilla JS usa `value.episodes`, pero el
JSON no lo trae). `status`/`estado` son el par código/texto; `type`/`tipo` idem
(`TV`/`Serie`, `ONA`/…); `url` = `https://jkanime.net/<slug>/`; `image` = portada CDN.

### P4 — Formato del slug y excepciones

- Slugs del Home = path de `https://jkanime.net/<slug>/` sin barra final, **idéntico al
  formato de `Anime.ID` de la búsqueda**: `search.html` contiene
  `href="https://jkanime.net/bleach/"` y `internal/scraper/jkanime/search.go:87` pone
  `ID: slug` con `slugFromURL(resuelta.URL)`.
- **Excepciones**: los 83 slugs distintos de la portada y los 30 del directorio en emisión
  se compararon con las claves de `testdata/excepciones.json` (claves truncadas a ~59 chars)
  → **0 coincidencias exactas** (`k.endsWith('/'+s)`). Las 14 coincidencias "de prefijo" son
  otros anime que *empiezan* por el mismo slug (`one-piece` vs
  `one-piece-episode-of-sabo---…`); el mapa no afecta a ningún slug del Home.

### P5 — Portadas y CDN

Host: **`cdn.jkdesa.com`**, ruta
`/assets/images/animes/image/<slug>.jpg`. Probe con `one-piece.jpg` (169189 B):

| Cabecera | Resultado |
|---|---|
| sin Referer | **200** |
| `Referer: https://example.com/` | **200** (mismo tamaño) |
| `Referer: https://jkanime.net/` | **200** (mismo tamaño) |

La meta `no-referrer` de `frontend/index.html` **no rompe nada**.

### P6 — Detalle de un slug del Home

`GET https://jkanime.net/hyouken-no-majutsushi-ga-sekai-wo-suberu-ii/` (primer slug de
`estado=emision`) → **200**, 63985 B. En su crudo: `<li><span>Estado:</span> <div
class="enemision currently">En emision</div>` (aparece 2 veces por el layout duplicado
móvil/escritorio).

---

## AnimeAV1

### P1 — Valores exactos de `status` y `order`

`/catalogo` es SvelteKit SSR: el diálogo de filtros y el menú de orden **no están en el HTML
crudo** (el `<dialog>` crudo solo declara `Año`/`Estado` como botones y el menú es
`<!---->`). Se abrieron con Playwright y se leyeron los `<input type="radio">`:

**`order`** (menú "Ordenar por", 6 valores, todos verificados con curl sobre
`__data.json`):

| value | texto | verificación |
|---|---|---|
| `default` | Predeterminado | `orderKey:"default"`, primer slug `battle-spirits-re-zekkai-no-kuu` |
| `score` | Puntuación | `orderKey:"score"`, primer slug `sousou-no-frieren` |
| `popular` | Populares | `orderKey:"popular"`, primeros `shingeki-no-kyojin`, `death-note`, `one-punch-man` |
| `title` | Titulo | `orderKey:"title"`, primer slug `kimi-wo-aisuru-ki-wa-nai-to-itta-…` |
| `latest_added` | Últimos Agregados | `orderKey:"latest_added"`, primeros `hotaru-no-yomeiri`, `toaru-anbu-no-item`, `hyouken-no-majutsushi-ga-sekai-wo-suberu-ii` |
| `latest_released` | Últimos Estrenos | `orderKey:"latest_released"`, primer slug `toaru-anbu-no-item` |

**`status`** (radio del diálogo: `value="0"` Finalizado, `value="1"` Próximamente,
`value="2"` "En emisión"). **La URL usa el slug, no el número**: al pulsar "Filtrar" con
Estado = En emisión, Playwright observó la navegación a
**`/catalogo?status=emision`**. Confirmado con curl (el JSON embebido del HTML refleja el
filtro en `filters.status`):

| URL | `filters.status` | `totalRecords` |
|---|---|---|
| `/catalogo?status=emision` | `2` | **58** |
| `/catalogo?status=finalizado` | `0` | (distinto, fixture) |
| `/catalogo?status=proximamente` | `1` | (distinto) |
| `/catalogo?status=2` (número) | **`null`** (ignorado) | 1000 |

O sea: `?status=<número>` **no filtra**; hay que usar `emision` / `finalizado` /
`proximamente`. Los tres devuelven 200.

También funciona **`?letter=A`** (`filters.letter:"A"`, `totalRecords:254`) y
**`?order=<valor>`** en la página HTML (fixture: `order=latest_added` →
`orderKey:"latest_added"` y los primeros `<article>` son los mismos que en el JSON).

La vía principal del código actual (`GET /catalogo/__data.json?…`) acepta los mismos
parámetros y devuelve el mismo listado (formato devalue; ver P4).

### P2 — `order` cambia de verdad el orden

Comparación de los 3 primeros slugs por orden (mismo día, fixtures/tabla de P1):

| order | 1.º | 2.º | 3.º |
|---|---|---|---|
| `default` | `battle-spirits-re-zekkai-no-kuu` | `shiotaiou-no-satou-san-ga-ore-ni-dake-amai` | `kyouran-reijou-nia-liston-…` |
| `latest_added` | `hotaru-no-yomeiri` | `toaru-anbu-no-item` | `hyouken-no-majutsushi-ga-sekai-wo-suberu-ii` |
| `popular` | `shingeki-no-kyojin` | `death-note` | `one-punch-man` |

`default` ≠ `latest_added` ≠ `popular` → el parámetro surte efecto.

### P3 — Qué ofrece la página principal `/`

`GET /` → 200, 200261 B (fixture `home_portada.html`). Tres bloques con `<h2>`:

1. **"Episodios"** + subtítulo `Recientemente Actualizado`: **20 tarjetas SSR**
   (`article.group/item.text-body.relative`) de episodio. Cada una: miniatura
   `https://cdn.animeav1.com/thumbnails/<mediaId>.jpg` (`img[alt="backdrop"]`), antigüedad
   (`hace 7 horas`), contador de comentarios (`ic-message-text-square-02 … >30</span>`),
   badge `Episodio <span>14</span>`, título en `div.text-2xs.text-subs.font-bold.uppercase`
   y enlace **`<a class="absolute inset-0 z-20 …" href="/media/<slug>/<n>">`**.
2. **"Actividad"** (`últimos comentarios`) + pestañas Recientes/Mejores: tarjetas de
   comentarios con avatar; **no sirve para el Home**.
3. **"Animes"** + subtítulo **`Recientemente Agregados`** + enlace `Catalogo de Animes`
   (`/catalogo`): **20 tarjetas SSR** (`article.group/item.relative.text-body`) con portada
   `https://cdn.animeav1.com/covers/<id>.jpg` (`alt="Portada de <título>"`), badge de
   categoría (`TV Anime`) y **`<a class="absolute inset-0 z-20 …" href="/media/<slug>">`**.

Además el hero muestra `featured` (2 títulos: One Piece, Sousou no Frieren 2nd Season).

**Formato**: los tres bloques vienen **en HTML crudo** (SSR) **y** en **`GET /__data.json`**
(fixture `home_portada_data.json`, 30796 B, devalue), cuyo `nodes[2].data[0]` es
`{featured, latestComments, latestEpisodes, latestMedia, latestTopComments}`:

- `latestEpisodes`: **20** objetos
  `{commentsCount, createdAt, id, media:{id, slug, title}, number, publishedAt}`.
- `latestMedia`: **20** objetos
  `{category:{id,name}, createdAt, id, poster (null), slug, synopsis, title}`.

### P4 — Campos disponibles por item y items por página

| Fuente | Items | Campos |
|---|---|---|
| Home `latestEpisodes` | 20 | `media.id` (numérico), `media.slug`, `media.title`, `number`, `publishedAt`, `commentsCount`; portada = `thumbnails/<mediaId>.jpg` |
| Home `latestMedia` | 20 | `id` (numérico), `slug`, `title`, `category.name`, `createdAt`, `synopsis`; portada = `covers/<id>.jpg` (el `poster` del JSON es `null`) |
| `/catalogo/__data.json` (`results`) | **20** por página | `id` (**string**), `title`, `synopsis`, `categoryId`, `slug`, `category:{id,name,slug,malId}`; portada = `covers/<id>.jpg` (en el HTML del catálogo, 20 `<article>`, mismo esquema que `search.html` de T30) |
| Paginación del catálogo | 20/página | `pagination:{currentPage, recordsPerPage:20, totalPages, totalRecords}`; `totalRecords` sigue **congelado en 1000** (50 páginas) salvo con filtro (`status=emision` → 58, 3 páginas) |

### P5 — Detalle de un slug de la lista

- `GET /media/hotaru-no-yomeiri/__data.json` (primer slug de "Recientemente Agregados") →
  **200**, 1868 B.
- `GET /media/uchi-no-otouto-domo-ga-sumimasen/14/__data.json` (primer enlace de
  "Episodios") → **200**, 3265 B.

---

## Tabla de secciones viables

| Proveedor | Sección (UI) | Origen | Items | Estabilidad |
|---|---|---|---|---|
| JKAnime | En emisión | `GET /directorio?estado=emision` → `var animes` JSON en `<script>` | 30/página, **85** total | Alta: JSON estricto embebido, sin XHR; el **orden por defecto no es por fecha de emisión** (es "Por fecha" de alta en el sitio) |
| JKAnime | Programación (semana) | `GET /` sección "Programación", SSR | 32+32+8 (Animes/Donghuas/Ovas) | Media-alta: SSR, pero son enlaces de **episodio** y los días son relativos ("Hoy"/"Ayer") |
| JKAnime | Animes recientes | `GET /` sección "Animes recientes", SSR | **32**, sin paginación | Alta: SSR puro; mezcla tipos (Serie/Pelicula/OVA/ONA) y no trae nº de episodio |
| JKAnime | Top animes | `GET /` sección "Top animes", SSR | **10** (rango 1–10 + likes) | Alta: SSR puro, estable (ranking del sitio) |
| JKAnime | Estrenos | `GET /estrenos`, SSR | **10** (con fecha `Estreno: YYYY-MM-DD`) | Media-alta: SSR, pero cambia cada semana |
| JKAnime | Populares | `GET /directorio?filtro=popularidad` → `var animes` | 30/página | Alta (mismo JSON que el directorio) |
| AnimeAV1 | Episodios recientes | `GET /` (`latestEpisodes`) o `__data.json` | **20** | Alta: SSR + devalue; orden por `publishedAt` |
| AnimeAV1 | Recién agregados | `GET /` (`latestMedia`) o `__data.json` | **20** | Alta |
| AnimeAV1 | En emisión | `GET /catalogo/__data.json?status=emision&order=latest_added` | 20/página, **58** total | Alta: JSON devalue con `filters.status:2` como confirmación |
| AnimeAV1 | Populares | `GET /catalogo/__data.json?order=popular` | 20/página | Alta |
| AnimeAV1 | Últimos estrenos | `GET /catalogo/__data.json?order=latest_released` | 20/página | Alta |

---

## Recomendación

**JKAnime — 2 secciones** (fila por proveedor en el Home):

1. **"En emisión"** → `GET /directorio?estado=emision` (fixture
   `home_directorio_emision.html`): 30 items, JSON estricto dentro del `<script>`
   (`var animes = ` … `;var mode`), extraíble sin XHR; `slug`, `title`, `image`, `estado`
   (`En emision`) y `tipo` de serie. Es la única fuente que responde literalmente a "en
   emisión" (**85** títulos).
2. **"Animes recientes"** → sección SSR de `GET /` (fixture `home_portada.html`): 32 items,
   portada, tipo y estado en el crudo. Sin paginación (una sola petición).

Alternativa/documentada: "Top animes" (10 items, SSR) si se prefiere una fila más estable
que "recientes".

**AnimeAV1 — 2 secciones**:

1. **"Episodios recientes"** → `GET /` → `latestEpisodes` (o el SSR equivalente): 20 items
   con `media.slug`, `number` y miniatura; enlace `/media/<slug>/<n>` (el `n` del href
   coincide con `number` del JSON en las 20 muestras).
2. **"En emisión"** → `GET /catalogo/__data.json?status=emision&order=latest_added`: 20
   items/página, 58 en total, con `slug`, `title`, `category.name` y portada
   `covers/<id>.jpg`.

Alternativa: "Recién agregados" (`latestMedia`) si se quiere una fila de solo anime; los
cinco órdenes del catálogo (`score`, `popular`, `title`, `latest_added`, `latest_released`)
quedan documentados por si el producto cambia de criterio.

**TTL de caché sugerido**: **10 min**, la misma clave `home:<proveedor>` que fija
`tareas/v1.2_PLAN.md` para el Home (los datos cambian en horas/días; 10 min evita
martilleo sin perder novedad). Los errores no se cachean y `noCache` salta la lectura.

**Estado/emisión para T46** (cómo se escribe "en emisión" en cada proveedor):

- **JKAnime** (texto del sitio, ya usado en el detalle): `En emision` (sin tilde, clase CSS
  `currently`), `Concluido` (`finished`), `Por estrenar` (`notyet`). Ejemplos reales:
  `hyouken-no-majutsushi-ga-sekai-wo-suberu-ii` → `<div class="enemision currently">En
  emision</div>` (fixture `home_directorio_emision.html` y detalle en vivo);
  `bleach` → `Concluido` (`testdata/detail.html`); `cardfight-vanguard-divinez-unmei-seisen-hen`
  → `"status":"notyet","estado":"Por estrenar"` (`home_directorio.html`).
- **AnimeAV1** (numérico → texto, ya mapeado en T30): `0` Finalizado, `1` Próximamente,
  `2` **En emisión**. Ejemplos reales de fixtures existentes:
  `detail_battle-spirits-re-zekkai-no-kuu_data.json` → `status:2`; `detail_one-piece_data.json`
  → `status:1`; `detail_ajin-ova_data.json` / `detail_data.json` (Bleach) → `status:0`.
  En el Home el estado **no** viene en los items de `latestEpisodes`/`latestMedia` (solo en
  `featured` y en el detalle); para la fila "En emisión" del catálogo el filtro ya garantiza
  el estado.

---

## Fixtures capturados

Respuesta **cruda** con `curl.exe` + UA de `httpx.DefaultUserAgent`, CSRF de JKAnime
sustituido por `TOKEN_DE_PRUEBA` (ver Metodología). Sin vídeos descargados.

| Archivo | Petición | HTTP | Bytes |
|---|---|---|---|
| `jkanime/testdata/home_portada.html` | `GET /` | 200 | 127860 |
| `jkanime/testdata/home_directorio.html` | `GET /directorio` | 200 | 71488 |
| `jkanime/testdata/home_directorio_emision.html` | `GET /directorio?estado=emision` | 200 | 70132 |
| `jkanime/testdata/home_directorio_popularidad.html` | `GET /directorio?filtro=popularidad` | 200 | 68897 |
| `jkanime/testdata/home_estrenos.html` | `GET /estrenos` | 200 | 37969 |
| `animeav1/testdata/home_portada.html` | `GET /` | 200 | 200261 |
| `animeav1/testdata/home_portada_data.json` | `GET /__data.json` | 200 | 30796 |
| `animeav1/testdata/home_catalogo.html` | `GET /catalogo` | 200 | 141686 |
| `animeav1/testdata/home_catalogo_data.json` | `GET /catalogo/__data.json` | 200 | 15836 |
| `animeav1/testdata/home_catalogo_emision.html` | `GET /catalogo?status=emision` | 200 | 141014 |
| `animeav1/testdata/home_catalogo_emision_data.json` | `GET /catalogo/__data.json?status=emision` | 200 | 15831 |

Los tres `*_data.json` de AnimeAV1 documentan la vía devalue (la misma que ya usa el
código para búsqueda/detalle); los HTML equivalentes quedan por si el producto prefiere el
SSR. No hay fixture de la sección "Continuar mirando" de JKAnime: **exige sesión** (XHR a
`/guardado`/`/historial`) y no es viable para un Home anónimo. Tampoco de la sección
"Actividad" de AnimeAV1 (comentarios, sin valor para el Home).

## Riesgos

1. **JKAnime pinta el directorio con JS**: aunque el JSON está en el crudo, el **contrato
   del `var animes`** (claves, orden, `studios` como string) no está versionado; un cambio
   de plantilla puede romper el parser → tests sobre fixture y tolerancia.
2. **`status` de AnimeAV1 por slug**: la URL usa `emision`/`finalizado`/`proximamente`, no
   el número; si el sitio renombra el slug el filtro deja de filtrar **sin error** (devuelve
   todo) → comprobar `filters.status` en la respuesta como guarda.
3. **`totalRecords` congelado en 1000** en el catálogo sin filtros (heredado de T30): no
   sirve para dimensionar; con `status=emision` sí da 58 real.
4. **Secciones de la portada de JKAnime sin paginación** (32 "recientes", 10 "top"): si el
   producto quiere más items hay que ir al directorio (JSON) o a `/top`.
5. **"Programación" usa días relativos** ("Hoy"/"Ayer") y enlaces de episodio: exige
   reconstruir el slug y puede confundir en una UI que espera tarjetas de anime.
6. **CSRF en el HTML de JKAnime**: cada página nueva trae un token distinto en el meta;
   los fixtures lo sanearon, pero el parser real **no debe** depender de ese valor (el Home
   no necesita POST).
7. **Feeds efímeros**: "Episodios recientes"/"Animes recientes" cambian en horas; la caché
   de 10 min es la única defensa, no una garantía de orden estable.

## Lo que no se verificó

- No se descargó **ningún vídeo** (regla 9/13): solo HTML/JSON y portadas (169 kB).
- No se comprobó `/top` ni `/historial` ni `/guardado` de JKAnime (fuera del alcance del
  Home anónimo).
- No se abrió la paginación del directorio más allá de `?p=2` implícita en los enlaces; el
  comportamiento de `?p=` con filtros combinados (`estado` + `p`) **no** se probó.
- No se comprobó el límite de tasa a largo plazo ni el comportamiento con VPN.
- No se verificó si "Animes recientes" de JKAnime ordena por alta en el sitio (la fecha que
  muestra es la de emisión del título, no la de alta): se constata solo que **no** está
   ordenada por esa fecha (muestras 2012, 1992, 2026…).
- No se reprodujo nada en la app (sin código en esta tarea).

## Resumen (10 líneas)

1. JKAnime sirve el Home completo en HTML crudo sin challenge: "Programación" (32+32+8
   enlaces de episodio), "Animes recientes" (32 tarjetas SSR), "Top animes" (10 con rango y
   likes); "Continuar mirando" exige sesión → descartada.
2. El directorio de JKAnime pinta tarjetas con JS, pero el listado vive como **JSON estricto
   `var animes`** dentro de un `<script>` (extraíble sin XHR); filtros exactos:
   `estado=emision` (85 títulos), `filtro=popularidad`, `tipo`, `orden`, `?p=`.
3. Slugs del Home = mismo formato que `Anime.ID` de la búsqueda; 0 coincidencias con
   `excepciones.json`; portadas en `cdn.jkdesa.com` 200 sin Referer; `GET /<slug>/` → 200.
4. AnimeAV1: `/` trae "Episodios" (20, `latestEpisodes`) y "Animes / Recientemente Agregados"
   (20, `latestMedia`) en SSR **y** `/__data.json`; "Actividad" (comentarios) no sirve.
5. `order` del catálogo: `default|score|popular|title|latest_added|latest_released` (todos
   verificados; los 3 primeros slugs documentados); `status` va por **slug**
   (`emision|finalizado|proximamente`), no por número.
6. `?status=emision` → `filters.status:2`, **58** títulos, 20/página; un slug de la lista
   abre `/media/<slug>/__data.json` con 200.
7. Recomendación: JKAnime = "En emisión" (directorio) + "Animes recientes" (portada);
   AnimeAV1 = "Episodios recientes" + "En emisión" (catálogo).
8. TTL de caché sugerido: 10 min (`home:<proveedor>`), coherente con el plan v1.2.
9. Estado: JKAnime `En emision`/`Concluido`/`Por estrenar`; AnimeAV1 `2`/`0`/`1` (3 ejemplos
   reales por proveedor en la sección de recomendación).
10. 11 fixtures crudos `home_*` capturados (CSRF de JKAnime → `TOKEN_DE_PRUEBA`); informe en
    `docs/investigacion/home.md`; sin cambios de código.
