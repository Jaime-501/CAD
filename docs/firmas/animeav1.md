# Firmas reales — internal/scraper/animeav1

> Firmas **tal como quedaron**, verificadas contra el código con `go doc ./internal/scraper/animeav1`.
> **Si una firma de este archivo no coincide con el código, manda el código (`go doc ./internal/scraper/animeav1`).**
> Añade bloques nuevos con el formato `### Txx — <paquete>`; no reescribas los antiguos salvo error real (en tal caso anótalo aquí mismo).

**Bloques de este archivo y tarea que los creó:**

- `### T31 — internal/scraper/animeav1` → creado por **T31** (desempaquetador devalue + búsqueda; `devalue.go`, `animeav1.go`).
  (T32 y T33 añadirán sus bloques cuando toque.)

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
// hoy: (models.AnimeDetail{}, nil, error "no implementado todavía: T32 …")

// GetVideoSources se implementa en T33 (página /media/<slug>/<n>). Devuelve un
// error descriptivo; opts no llega a usarse mientras no haya extracción.
func (a *AnimeAV1) GetVideoSources(_ context.Context, episodeID string, _ scraper.SourceOpts) ([]models.VideoSource, error)
// hoy: (nil, error "no implementado todavía: T33 …")
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
func (a *AnimeAV1) siteURL(ruta string) string        // base + ruta, sin barra doble
func resultadosAnime(datos map[string]any) ([]models.Anime, error)
func portada(id any) string                            // "" → ""; int/float64 → portadaCDN + n + ".jpg"
func numeroDe(v any) (float64, bool)                   // int, int64, float64, json.Number
```

- `resultadosAnime`: exige `results` **lista** (si no está → "no trae el campo results"); cada elemento con `slug` y `title` se convierte en `models.Anime`, y los que no tengan ambos se **ignoran**. `ID = slug`, `Title`, `CoverURL = portadaCDN + id + ".jpg"`, `Provider = providerName`. `id` en los fixtures es **string** ("195") pero se tolera numérico.
- `Search` no pagina: devuelve como mucho `pagination.recordsPerPage` (20 en los fixtures) de la primera página. Query vacía o solo espacios → `[]models.Anime{}, nil` **sin petición**. Catálogo sin resultados → slice **vacío no-nulo**, no error.

#### Tests

- **22 tests** (33 subtests) en `animeav1_test.go` + `devalue_test.go`; sin red, `-race` en verde.
- `animeav1_test.go`: `TestMain` (nivel `Info` → `io.Discard`), `doerGrabador`/`clientePrueba`/`fixture`/`servidorJSON`, y: Name, NewNormaliza, SearchRutaYResultados, EscapaLaQueryYUsaUserAgent, QueryVaciaNoPideRed, SinResultados, JSONCorrupto, RespuestaInesperada, ErrorHTTP (404/500/403 → `ErrNotFound`/`ErrServerError`/`ErrForbidden`), CtxCancelado, SinDoer (`ErrNilDoer`), CuerpoDemasiadoGrande, MetodosPendientes (ni `scraper.ErrNoDetails` ni `ErrNotImplemented`).
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
- `episodes[i]` es `[idNúmero, n]` (array, no objeto) en el detalle; en `episode_data.json` la raíz es `{media, episode, embeds, downloads}` y `episode.mirrors` viene `undefined` → `nil`.
- Sin cookies/CSRF/rate-limit observado → `httpx.Get` con la UA por defecto basta (a diferencia de JKAnime).

---
