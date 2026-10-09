# Firmas reales — internal/scraper/animeav1

> Firmas **tal como quedaron**, verificadas contra el código con `go doc ./internal/scraper/animeav1`.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper/animeav1`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>`; no reescribas los antiguos salvo error real (en tal caso anótalo aquí mismo).

**Bloques de este archivo y tarea que los creó:**

- `### T31 — internal/scraper/animeav1` → creado por **T31** (desempaquetador devalue + búsqueda; `devalue.go`, `animeav1.go`).
- `### T32 — internal/scraper/animeav1` → creado por **T32** (detalle y episodios por lotes; `detail.go`, `episodes.go`).
- `### T33 — internal/scraper/animeav1` → creado por **T33** (fuentes de vídeo; `sources.go`).

---

### T31 — internal/scraper/animeav1

Punto de partida: paquete nuevo en la rama `feature/animeav1`. **Vía principal = `__data.json`** (`<ruta>/__data.json`, JSON en formato *devalue*), no el `<script>` de hidratación: así lo manda T31 pese a lo que P7 del informe T30 (descartada en ese momento) dijera.

#### Paquete

```go
package animeav1 // import "miapp/internal/scraper/animeav1"
```

Documentación de paquete (lo que muestra `go doc`): sitio **SvelteKit SSR sin cookies, sin CSRF y sin rate-limit observado**; alcance de T31 = búsqueda + desempaquetador; detalle (T32) y fuentes de vídeo (T33) aún pendientes.

#### Constantes

```go
// Exportadas
const DefaultBaseURL = "https://animeav1.com" // la usa New con base vacía y el valor cero

// Internas (no exportadas, documentadas aquí para T32/T33)
const providerName  = "animeav1"                          // → scraper.Provider.Name() y models.Anime.Provider
const rutaBusqueda  = "/catalogo/__data.json"             // la usa Search (T31)
const portadaCDN    = "https://cdn.animeav1.com/covers/"  // portada = portadaCDN + <id> + ".jpg"
const maxBusquedaBytes = 10 << 20                         // tope de cuerpo leído en Search (10 MiB)

// Formato devalue (solo internos, usados por Unflatten)
const devalueUndefined = -1
const devalueHueco     = -2   // hueco → nil
const devalueNaN       = -3
const devalueInf       = -4
const devalueMenosInf  = -5
const devalueMenosCero = -6
const devalueDisperso  = -7   // → error (solo se lee en arrays dispersos)
```

#### Modelo y registro

```go
// AnimeAV1 implementa scraper.Provider sobre animeav1.com. Sus dependencias
// (doer y URL base) se inyectan para que los tests corran contra un
// httptest.Server sin salir a la red (regla 4). El valor cero es utilizable:
// BaseURL vacía cae en DefaultBaseURL y, sin doer, httpx devuelve ErrNilDoer
// sin red.
type AnimeAV1 struct {
	BaseURL string          // sin barra final; vacía → DefaultBaseURL
	Doer    httpx.HTTPDoer  // *http.Client lo cumple
}

var _ scraper.Provider = (*AnimeAV1)(nil)   // compilación
```

Registro en `scraper.Registry` (lo hace quien crea el proveedor, no este paquete; ahora mismo lo usan solo los tests y la integración de T32+).

#### API pública (firmas literales)

```go
// New crea el proveedor con el doer y la URL base inyectados. La base
// se normaliza (sin espacios ni barra final) y, si viene vacía, se usa
// DefaultBaseURL.
func New(doer httpx.HTTPDoer, baseURL string) *AnimeAV1

func (a *AnimeAV1) Name() string
// → "animeav1"

// Search busca query en el catálogo pidiendo el __data.json de /catalogo y
// devuelve cada resultado como models.Anime: ID es el slug (nunca el título),
// Title el título y CoverURL la portada absoluta del CDN. El id del sitio va
// como string y solo se usa para la portada.
//
// Devuelve la primera página del catálogo, con 20 registros por página como
// mucho (pagination.recordsPerPage del propio JSON); no se pagina ni se piden
// más páginas. Una query vacía no hace ninguna petición y un catálogo sin
// resultados devuelve un slice vacío, no un error.
func (a *AnimeAV1) Search(ctx context.Context, query string) ([]models.Anime, error)

// GetAnimeDetails se implementa en T32 (detalle de /media/<slug> con los
// episodios embebidos). Devuelve un error descriptivo, no scraper.ErrNoDetails
// (el proveedor sí ofrecerá detalle).
func (a *AnimeAV1) GetAnimeDetails(_ context.Context, id string) (models.AnimeDetail, []models.Episode, error)
// T31: (models.AnimeDetail{}, nil, error "no implementado todavía: T32 …")
// → implementado en T32: ver `### T32 — internal/scraper/animeav1` más abajo.

// GetVideoSources descarga el __data.json de /media/<slug>/<n>, extrae el
// campo embeds y devuelve el HLS del reproductor propio más el resto de
// servidores como iframe, cada fuente con su idioma (SUB/DUB).
func (a *AnimeAV1) GetVideoSources(_ context.Context, episodeID string, _ scraper.SourceOpts) ([]models.VideoSource, error)
// T31: (nil, error "no implementado todavía: T33 …")
// → implementado en T33: ver `### T33 — internal/scraper/animeav1` más abajo.
```

#### Desempaquetador (`devalue.go`) — API que usarán T32 y T33

```go
// Unflatten convierte el array plano de un nodo devalue (el campo data de un
// __data.json de SvelteKit) en los valores Go correspondientes, resolviendo
// las referencias por índice. El elemento 0 del array es la raíz.
//
// Soporta: null, booleanos, cadenas y números literales (los que van sueltos
// en el array), objetos, arrays, undefined, huecos (los dos últimos → nil) y
// los valores especiales numéricos NaN, ±Inf y -0. Devuelve error ante tipos
// desconocidos: arrays con etiqueta (Date, Map, Set, RegExp, Object, BigInt,
// arrays tipados, "null"), arrays dispersos (−7), un slot que no sea número o
// una referencia fuera de rango. Nunca hace panic.
//
// Los ciclos (un objeto que referencia a un ancestro) se resuelven porque
// cada contenedor se publica en valores[] antes de rellenarse, igual que hace
// devalue.
func Unflatten(plano []any) (any, error)

// PageData extrae la raíz del campo data de un __data.json y la devuelve ya
// hidratada. Acepta también `{"type":"data","nodes":[...]}`
func PageData(crudo []byte) (map[string]any, error)
```

**Contrato de tolerancia de `PageData`** (todo cubierto por `TestPageDataTolerancia`): cuerpo vacío o no-JSON → error con "JSON válido"; `type` distinto de `"data"` → error; `nodes` ausente o `null` → error; nodos `null`, `{"type":"skip"}`, `{"type":"error"}` y `{"type":"data"}` **sin campo `data`** se saltan; el nodo elegido es **el último** con `type:"data"` y `data` **array**; la raíz (índice 0) debe ser un objeto o `null` → si no, error; referencia rota dentro del array → error.

**Errores de `Unflatten`**: array vacío ("array vacío"); slot no numérico ("etiqueta desconocida" cuando el elemento suelto no es un número y no es literal soportado; "un slot debe ser un número"); −7 ("disperso"); referencia ≤ −8 o ≥ len ("fuera del array"); `data`/raíz no-objeto; tipos con etiqueta (MUY: si algún día aparece `Date`/`Map` en un payload de T32/T33, hará falta ampliar — hoy los payloads usan **cadenas** para fechas).

#### Internos

```go
func (a *AnimeAV1) siteURL() string                     // base sin barra final; vacía → DefaultBaseURL  [corregido en T32]
func resultadosAnime(datos map[string]any) ([]models.Anime, error)
func portada(id any) string                             // id string o float64 → portadaCDN + texto + ".jpg"; otro tipo → ""
func idTexto(id any) string                             // string o float64 → texto; otro tipo → ""  [T32]
func numeroDe(v any) (float64, bool)                    // float64 (encoding/json) o int (en tests)  [corregido en T32]
```

> Correcciones de T32: la firma real de `siteURL` es sin argumentos; `portada` acepta **string** (los resultados de búsqueda traen el id como texto) y `float64` (el detalle lo trae numérico), no `int`; y `numeroDe` solo acepta `float64`/`int` (no `int64` ni `json.Number`). Manda el código.

- `resultadosAnime`: exige `results` **lista** (si no está → "no trae el campo results"); cada elemento con `slug` y `title` se convierte en `models.Anime`, y los que no tengan ambos se **ignoran**. `ID = slug`, `Title`, `CoverURL = portadaCDN + id + ".jpg"`, `Provider = providerName`. `id` en los fixtures es **string** ("195") pero se tolera numérico.
- `Search` no pagina: devuelve como mucho `pagination.recordsPerPage` (20 en los fixtures) de la primera página. Query vacía o solo espacios → `[]models.Anime{}, nil` **sin petición**. Catálogo sin resultados → slice **vacío no-nulo**, no error.

#### Tests

- **22 tests** (33 subtests) en `animeav1_test.go` + `devalue_test.go`; sin red, `-race` en verde.
- `animeav1_test.go`: `TestMain` (nivel `Info` → `io.Discard`), `doerGrabador`/`clientePrueba`/`fixture`/`servidorJSON`, y: Name, NewNormaliza, SearchRutaYResultados, EscapaLaQueryYUsaUserAgent, QueryVaciaNoPideRed, SinResultados, JSONCorrupto, RespuestaInesperada, ErrorHTTP (404/500/403 → `ErrNotFound`/`ErrServerError`/`ErrForbidden`), CtxCancelado, SinDoer (`ErrNilDoer`), CuerpoDemasiadoGrande, MetodosPendientes (ni `scraper.ErrNoDetails` ni `ErrNotImplemented`).
  **T33 retiró `TestMetodosPendientes`** (ya no queda ningún método sin implementar) y con él el import de `miapp/internal/scraper` en `animeav1_test.go`.
- `devalue_test.go`: PageData sobre los 4 fixtures T31 (catálogo, vacío, detalle, película), PageData del episodio (`testdata/episode_data.json` de T30), UnflattenEspeciales (−1…−6, literales, lista, `"literal"`), UnflattenCiclos, UnflattenErrores (8), PageDataTolerancia (10).
- **`animeav1_integration_test.go` (tag `integration`)**: `TestSearchRedReal` busca "bleach" en el sitio real y exige ≥1 resultado con `slug == "bleach"` y `Title == "Bleach"`.

```sh
docker compose run --rm app go test -tags integration ./internal/scraper/animeav1/ -v
```

#### Fixtures

`testdata/` (raíz = HTML/JSON crudo sanitizado de T30/T31):

| archivo | origen |
| --- | --- |
| `search.html`, `search_empty.html`, `detail.html`, `detail_movie.html`, `episode_page.html`, `episode_data.json` | T30 (HTML con `<script>` de hidratación + 1 JSON) |
| `catalogo_data.json` | T31 · `GET /catalogo/__data.json?search=bleach` (16135 B) |
| `catalogo_vacio_data.json` | T31 · `?search=zzzznoexisteestetitulo` (5003 B) |
| `detail_data.json` | T31 · `GET /media/bleach/__data.json` (16764 B) |
| `detail_movie_data.json` | T31 · `GET /media/bleach-movie-1-memories-of-nobody/__data.json` (2538 B) |

Captura: `curl.exe -H "User-Agent: httpx.DefaultUserAgent"` (Chrome UA); **sanitizados** con las reglas de `tareas/00_REGLAS.md` (sin `st=`, `e=`, cookies, firmas, tokens).

#### Notas que afectan a T32/T33

- **Equivalencia HTML ↔ `__data.json` verificada** en T31 (script en `%TEMP%`, salida `TODO EQUIVALENTE`): resultados idénticos, `media.id=195`, `slug="bleach"`, `episodesCount=366`, `episodes` = 366 pares `{id,número}`, película `14703:1`.
- **Esquema del nodo de página**: `{"type":"data","nodes":[null,auth,página]}`; el nodo de datos es **el último** con `type:"data"` y `data` array. **No** usar `nodes[1]`.
- Rutas: búsqueda `/catalogo/__data.json?search=…`; detalle `/media/<slug>/__data.json`; episodio `/media/<slug>/<n>/__data.json`.
- `episodes[i]` es el **objeto** `{"id":n,"number":n}` (referencias por índice dentro del array plano), **no** un array `[id, n]` — nota corregida por **T32** al parsear los fixtures de detalle con `go doc`/pruebas: manda el código (`detalleDeMedia`). En `episode_data.json` la raíz es `{media, episode, embeds, downloads}` y `episode.mirrors` viene `undefined` → `nil`.
- Sin cookies/CSRF/rate-limit observado → `httpx.Get` con la UA por defecto basta (a diferencia de JKAnime).

---

### T32 — internal/scraper/animeav1

Detalle y episodios. Archivos nuevos: `detail.go` (detalle) y `episodes.go` (reparto por lotes). Vía: **una sola petición** a `GET /media/<slug>/__data.json` (el sitio embebe los episodios completos en la ficha, T30) + `PageData` de T31.

#### Constantes nuevas

```go
const maxDetalleBytes  = 10 << 20    // tope de cuerpo leído en el detalle (10 MiB)
const rutaDetalleMedia = "/media/"   // ruta = a.siteURL() + rutaDetalleMedia + PathEscape(slug) + rutaDatosPagina
const rutaDatosPagina  = "/__data.json"
const miniaturaCDN     = "https://cdn.animeav1.com/screenshots/" // + <mediaId> + "/" + <n> + ".jpg"

const estadoFinalizado   = "Finalizado"    // status 0
const estadoProximamente = "Próximamente"  // status 1
const estadoEnEmision    = "En emisión"    // status 2
const estadoDesconocido  = "Desconocido"   // otro número, fracción, null o ausente

const tamLote = 50   // episodios por models.EpisodeBatch (episodes.go)

var errSlugVacio = errors.New("animeav1: el id del anime no contiene slug") // no exportado
```

#### API pública (firmas literales)

```go
// GetAnimeDetails descarga el __data.json de /media/<slug> y devuelve el
// detalle más TODOS los episodios del array episodes, en una sola petición y
// sin emitir lotes (el flujo por lotes es StreamAnimeDetails, episodes.go).
//
// Un id vacío no toca la red; los fallos HTTP llegan tal cual de httpx y un
// slug inexistente (que SvelteKit contesta con HTTP 200 y un nodo de error
// 404) devuelve el mismo error tipado de "no encontrado" que recibe JKAnime.
func (a *AnimeAV1) GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error)

// StreamAnimeDetails entrega el detalle y los primeros tamLote episodios de
// inmediato (esa primera tanda NO se emite como lote, igual que la primera
// página de JKAnime) y el resto en una goroutine, un lote de 50 en lote,
// con el último marcado como Done y siempre con AnimeID = slug del detalle.
//
// Si onBatch es nil no se lanza ninguna goroutine. Si ctx se cancela —antes
// o entre lotes— la goroutine termina sin emitir nada más, ni siquiera el
// lote Done, y sin quedar huérfana; los datos ya están en memoria, así que
// no hay espera artificial entre lotes.
func (a *AnimeAV1) StreamAnimeDetails(ctx context.Context, id string,
	onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error)

var _ scraper.EpisodeStreamer = (*AnimeAV1)(nil)
```

#### Internos

```go
func (a *AnimeAV1) detalleYEpisodios(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error)
func detalleDeJSON(slug, ruta string, data []byte) (models.AnimeDetail, []models.Episode, error)
func detalleDeMedia(slug string, media map[string]any) (models.AnimeDetail, []models.Episode, error)
func listaEpisodios(media map[string]any) ([]any, error)
func convertirEpisodios(slug, titulo, mediaID string, kind models.EpisodeKind, lista []any) []models.Episode
func tituloEpisodio(titulo string, kind models.EpisodeKind, crudo any, n int) string
func kindEpisodio(categoria string, n int) models.EpisodeKind
func textoEstado(v any) string
func enteroDe(v any, porDefecto int) int
func nombresGeneros(v any) []string
func miniatura(mediaID string, n int) string
func paginaInexistente(data []byte) bool
func errorNoEncontrado(ruta string) error
func textoDe(v any) string
func emitirLotes(ctx context.Context, animeID string, resto []models.Episode, onBatch func(models.EpisodeBatch)) // episodes.go
```

#### Comportamiento (lo que T33–T35 deben saber)

- `models.AnimeDetail`:
  - `ID` = slug pedido (recortado de espacios y barras; vacío → `errSlugVacio`, **sin red**), `Provider = "animeav1"`, `Title`/`Synopsis` = `media.title`/`media.synopsis`.
  - `Type` = `category.slug` (`tv-anime`, `ova`, `ona`, `pelicula`, `especial`…; propuesta P7 de T30).
  - `Status`: `0/1/2` → Finalizado / Próximamente / En emisión; cualquier otro número, fracción, `null` o ausente → **"Desconocido"**.
  - `Genres` = `genres[].name`, **siempre no-nulo** (slice vacío si faltan).
  - `Cover` = `https://cdn.animeav1.com/covers/<media.id>.jpg` (el id llega **numérico** en el detalle).
  - `TotalEpisodes` = `media.episodesCount`; si es ilegible (≤0, fracción, ausente) → `len(episodios)`.
- `models.Episode` por cada entrada de `media.episodes` (las mal formadas y las repetidas se descartan):
  - `ID` = `models.NewEpisodeID("animeav1", slug, kind, n)`.
  - `Kind` = `pelicula` **con 1 episodio** → `KindMovie`; `especial` → `KindSpecial`; todo lo demás → `KindEpisode` (una película de 2+ episodios es `KindEpisode`: el id movie no admite número y colisionaría).
  - `Number` = `n` del sitio, salvo `KindMovie` → **0** (contrato de `models.Episode`; la miniatura sigue usando `n`).
  - `Title` = `"<media.title> - <n>"` (película de 1 ep → solo el título); si el sitio trae `title`, manda.
  - `Thumbnail` = `https://cdn.animeav1.com/screenshots/<media.id>/<n>.jpg` (vacío sin id).
- **404**: SvelteKit contesta HTTP 200 con `{"type":"error",…,"status":404}` y `PageData` acaba en un nodo sin `media` → `errorNoEncontrado` devuelve `&httpx.ScraperError{Provider, Operation:"detail", URL, StatusCode:404, Cause: httpx.ErrNotFound}` (`errors.Is` funciona). Un 200 válido **sin** `media` es otro error ("no trae media"), no un 404.
- **Lotes** (`tamLote = 50`): los primeros 50 vuelven por valor (sin emitirse); el resto sale de `emitirLotes` con `Batch = 2 + i/50`, `AnimeID = slug` y `Done` solo en el último. Sin resto (≤50 episodios o ninguno) → un único lote `Batch:1, Done:true` sin episodios. `ctx.Err()` se comprueba antes de cada emisión y `onBatch == nil` no arranca goroutine.
- Cuerpo > `maxDetalleBytes` → error que menciona "máximo", sin parsear.

#### Tests

- **`detail_test.go`**: RutaUserAgentYSlug, Bleach (1 petición, 366 episodios, géneros, portada, miniaturas, numeración 1..366), PeliculaDeUnEpisodio (`KindMovie` + `Number` 0), PeliculaDeVariosEpisodios (`KindEpisode`), EstadosYTipos (5 subtests: status 1/2/0, `ova`, `especial`, slug "episode-0" con `number:1`), EstadoDesconocido (4 subtests con payload **artificial** `cuerpoSintetico`), PaginaInexistente (`errors.Is(err, httpx.ErrNotFound)` + `ScraperError.StatusCode == 404`), SinMedia, RespuestaInesperada (5), SlugVacioNoPideRed, SinDoer, CtxCancelado, CuerpoDemasiadoGrande, NumeracionConHueco (Gintama: 201 declarados, 200 listados, hueco en el 2).
- **`episodes_test.go`**: LotesDe50 (366 → 50 + 7 lotes `Batch` 2..8 con tamaños 50×6+16, sin peticiones extra, nada tras el Done), PeliculaDeUnEpisodio (un solo lote Done vacío), FichaSinEpisodios, SinCallbackNoLanzaGoroutine, CancelacionSinFugas (cancela en el lote 3 → llegan 2 y 3, ninguno con Done, goroutines de vuelta a la base).
- `animeav1_test.go`: `TestMetodosPendientes` pasa a cubrir **solo** `GetVideoSources` (T33). → **T33 lo retiró** (ya no queda ningún método sin implementar; ver el bloque T31).
- **`animeav1_integration_test.go` (tag `integration`)**: nuevo `TestDetalleRedReal` → `GetAnimeDetails("bleach")` exige `TotalEpisodes ≥ 366` y `len(episodios) ≥ 366` sin ids repetidos.
- Verificación de T32: `gofmt -l .` sin salida · `go vet ./...` y `go vet -tags integration ./...` limpios · `go test -race -count=1 ./...` en verde · `go test -race -count=5 ./internal/scraper/animeav1/` en verde · `npm run check` 0 errores.

#### Fixtures nuevos (T32)

Captura: `curl.exe` + `httpx.DefaultUserAgent`, todos `200 application/json`, sanitizados (escaneo de tokens → 0 coincidencias).

| archivo | origen |
| --- | --- |
| `detail_one-piece_data.json` | `GET /media/one-piece/__data.json` (54096 B) · status 1, 1180 episodios |
| `detail_battle-spirits-re-zekkai-no-kuu_data.json` | `GET /media/battle-spirits-re-zekkai-no-kuu/__data.json` (1263 B) · status 2 |
| `detail_ajin-ova_data.json` | `GET /media/ajin-ova/__data.json` (1618 B) · status 0, `ova` |
| `detail_one-piece-film-strong-world-episode-0_data.json` | `GET /media/one-piece-film-strong-world-episode-0/__data.json` (1715 B) · slug con "episode-0" pero `number:1` |
| `detail_boku-no-hero-academia-more_data.json` | `GET /media/boku-no-hero-academia-more/__data.json` (1707 B) · `especial` |
| `detail_gintama-season-1_data.json` | `GET /media/gintama-season-1/__data.json` (10635 B) · hueco: lista 200 vs `episodesCount` 201 |
| `detail_girls-band-cry-movie_data.json` | `GET /media/girls-band-cry-movie/__data.json` (1267 B) · película de **2** episodios |
| `detail_404_data.json` | `GET /media/<slug-inexistente>/__data.json` (205 B) · HTTP 200 + nodo `{"type":"error","status":404}` |

---

### T33 — internal/scraper/animeav1

Fuentes de vídeo. Archivo nuevo: `sources.go`. Vía: **una sola petición** a `GET /media/<slug>/<n>/__data.json` (misma ruta que el detalle de T32, reutilizando `PageData`/`Unflatten`) y extracción del campo `embeds`; `downloads` se ignora (enlaces de descarga, no reproductores embebibles).

#### Constantes nuevas

```go
const maxFuentesBytes = 10 << 20   // tope de cuerpo leído (10 MiB), al estilo de maxBusquedaBytes/maxDetalleBytes

const hostJugador    = "player.zilla-networks.com" // único host del que se deriva un HLS
const esquemaJugador = "https://"
const rutaJugador    = "/play/"     // embed del reproductor: NUNCA se pide (exige Referer)
const rutaManifiesto = "/m3u8/"     // manifiesto derivado sin petición
const largoIDJugador = 32           // longitud que exige el JS del player (T30/P4)

const cabeceraFetch = "Sec-Fetch-Site"  // Headers de las fuentes HLS
const valorFetch    = "same-origin"

var idiomasEmbeds = []string{models.LanguageSub, models.LanguageDub} // recorrido determinista: SUB, DUB
```

Cambio en `detail.go`: `func errorNoEncontrado(op, ruta string) error` pasó a recibir la **operación** (`"detail"` en T32, `"sources"` en T33); misma firma interna, no exportada.

#### API pública (firmas literales)

```go
// GetVideoSources descarga el __data.json del episodio identificado por
// episodeID y devuelve el HLS del reproductor propio (derivado sin petición
// de /play/<id> a /m3u8/<id>) más el resto de servidores como iframe, cada
// fuente con su idioma estructurado en models.VideoSource.Language (SUB/DUB).
func (a *AnimeAV1) GetVideoSources(ctx context.Context, episodeID string, opts scraper.SourceOpts) ([]models.VideoSource, error)
```

#### Internos

```go
func fuentesDeJSON(slug, ruta string, data []byte) ([]models.VideoSource, error)
type grupoServidor struct {
	nombre   string                // models.VideoSource.Server
	conHLS   bool                  // el grupo con manifiesto derivado sale primero
	primero  int                   // orden de aparición de su primera fuente
	fuentes  []models.VideoSource  // SUB y DUB adyacentes, en orden de inserción
}
func fuentesDeEmbeds(embeds map[string]any) ([]models.VideoSource, error)
func fuenteDeEmbed(idioma string, item any) (models.VideoSource, bool)
func manifiestoDe(u *url.URL) (string, bool)
func idJugadorValido(id string) bool
func claveFuente(f models.VideoSource) string
```

#### Comportamiento (lo que T34–T35 deben saber)

- **Ruta**: `ParseEpisodeID` → proveedor debe ser `animeav1`; `KindMovie` (cuyo `number` es 0) pide **siempre** `/media/<slug>/1/__data.json`, el resto el número del id. `Operation = "sources"`. `opts` (`NoCache`) **no se usa**: las fuentes nunca se cachean (regla 9). Sin red con id vacío/no canónico/otro proveedor y con `Doer == nil` (`ErrNilDoer`).
- **`embeds`** es `{SUB:[{server,url}…], DUB:[…]}`; solo se recorren esas dos claves en ese orden (`idiomasEmbeds`), cada lista en su orden de aparición. Una entrada necesita objeto con `url` http(s) con host —lo demás se descarta (`Server` vacío → `Hostname()`). `Quality`, `Referer` y `Subtitles` quedan **vacíos** (nadie los exige hoy: T12–T15 los deja en vacío cuando está comprobado que no hacen falta).
- **HLS derivado (sin petición)**: un embed de `https://player.zilla-networks.com/play/<32 hex>` (sin query, sin fragmento) → `https://player.zilla-networks.com/m3u8/<mismo id>` con `Type: "hls"` y `Headers: {"Sec-Fetch-Site":"same-origin"}` (los segmentos responden 403 sin ella; `stream/reenvio.go` ya la inyecta, T19). Cualquier otra URL del jugador —otro host, `http`, id de 31/33, no hex, query, fragmento, path distinto— **no se deriva**: queda como iframe. **`/play/*` nunca se pide.**
- **Iframes sin resolver**: UPNShare, Voe, Byse, MP4Upload (y cualquier embed no derivable) se publican como `Type: "iframe"` con su URL tal cual. **No se reutilizó `extractors`** → ningún iframe se resuelve a MP4/HLS: pendiente explícito de **T34/T35** (si hace falta MP4, resolverlos ahí o ampliar el paquete).
- **Deduplicación (`claveFuente`)**: recurso base —esquema y host en minúsculas, path, **sin query** y **con fragmento** (criterio de `Normalize`, T16; la query solo cambia la firma temporal y el fragmento de un embed es a menudo su clave de descifrado)— **más el idioma**. **Desviación anotada de la ficha "deduplica por URL"**: la misma URL en SUB y en DUB son **dos** fuentes distintas (con `Language` distinto), y la misma URL con el mismo idioma se reduce a una; dos URLs que solo difieren en el fragmento **sobreviven ambas**.
- **Orden de salida**: primero el grupo con HLS propio (coincide con lo que elegiría `SelectBestSource`, T17) y después los grupos por orden de aparición; dentro de cada servidor, sus fuentes en orden de inserción (SUB antes que DUB) → SUB y DUB **adyacentes** y el resultado estable entre corridas. Orden esperado en `bleach/1` (10 fuentes): HLS·SUB, HLS·DUB, UPNShare·SUB/DUB, Voe·SUB/DUB, Byse·SUB/DUB, MP4Upload·SUB/DUB. Película (5, todas SUB): HLS, UPNShare, Voe, Byse, MP4Upload.
- **Errores**: los HTTP llegan tal cual de httpx; episodio fuera de rango (SvelteKit contesta HTTP 200 con `{"type":"error","status":404}`) → `errorNoEncontrado("sources", ruta)` = `&httpx.ScraperError{Operation:"sources", StatusCode:404, Cause: httpx.ErrNotFound}`; sin `embeds`, `embeds` no-objeto o ninguna entrada reproducible → error **descriptivo** (nunca un slice vacío silencioso); cuerpo > 10 MiB → error que menciona "máximo".

#### Tests

- **`sources_test.go` (19 tests, 26 subtests, sin red)**: RutaUserAgentYUnaPeticion (1 petición, UA por defecto, `Operation:"sources"`), UsaElNumeroDelID (episodio/especial), PeliculaPideElUno, HlsDerivado (URL + `Type` + `Headers`), ParIdiomaPorServidor (par **exacto** (SUB, DUB) por servidor, no solo adyacencia), IframesYDownloadsIgnorados, EmbedNoDerivaEsIframe (9 casos: id de 31/33, no hex, host, esquema, path, query, fragmento + control que sí deriva), DedupConservaElFragmento, DedupConservaElIdioma, DedupMismaUrlMismoIdioma, SinEmbeds (6), EntradaMalFormada, PaginaInexistente, ErrorHTTP (404/500), IdInvalidoNoPideRed (4), SinDoer, CtxCancelado, CuerpoDemasiadoGrande, RespuestaInesperada (3).
- **`animeav1_test.go`**: `TestMetodosPendientes` retirado (ver bloque T31).
- **`animeav1_integration_test.go` (tag `integration`)**: nuevo `TestFuentesRedReal` → `GetVideoSources("animeav1:bleach:episode:1")` exige HLS derivado, ≥1 iframe y fuentes con `Language`; después pide **solo el `.m3u8`** con `Sec-Fetch-Site: same-origin` y comprueba `Content-Type` MPEGURL y cuerpo `#EXTM3U` (ni segmentos ni vídeo; la URL se imprime recortada, sin id completo).
- Verificación de T33: `gofmt -l .` sin salida · `go vet ./...` y `go vet -tags integration ./...` limpios · `go build ./...` · `go test -race -count=1 ./...` en verde (animeav1: **59 tests / 73 subtests**; models: **14 / 40**) · smoke en verde: `Fuentes: 10 | HLS 2 | iframe 8 | con idioma 10 | manifiesto: https://player.zilla-networks.com/m3u8/65f3…`.

#### Fixtures nuevos (T33)

Captura: `curl.exe` + `httpx.DefaultUserAgent`, ambos `200 application/json`, sanitizados (escaneo de tokens → 0 coincidencias).

| archivo | origen |
| --- | --- |
| `episode_movie_data.json` | `GET /media/bleach-movie-1-memories-of-nobody/1/__data.json` (3500 B) · raíz `{media,episode,embeds,downloads}`; `embeds` trae **solo SUB** (5 servidores) |
| `episode_404_data.json` | `GET /media/bleach/367/__data.json` (205 B) · **HTTP 200** con nodo `{"type":"error","error":{"message":"No encontrado"},"status":404}` |

---
