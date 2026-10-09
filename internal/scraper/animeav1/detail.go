package animeav1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

const (
	// maxDetalleBytes limita el __data.json del detalle (10 MB), al estilo
	// de maxBusquedaBytes: un cuerpo mayor se rechaza sin parsearlo.
	maxDetalleBytes = 10 << 20

	// rutaDetalleMedia y rutaDatosPagina componen el endpoint del detalle:
	// <base>/media/<slug>/__data.json (misma raíz que la página del sitio).
	rutaDetalleMedia = "/media/"
	rutaDatosPagina  = "/__data.json"

	// miniaturaCDN es el prefijo de las miniaturas de episodio
	// (screenshots/<mediaId>/<n>.jpg). Verificado en T30: responde 200 si
	// el episodio existe y 403 si no, sin exigir Referer.
	miniaturaCDN = "https://cdn.animeav1.com/screenshots/"

	// Textos de models.AnimeDetail.Status. Son los tres valores que el
	// sitio pinta en la ficha (comprobados en el muestreo de T32 contra el
	// HTML de la página): cualquier otro número, un status no numérico o un
	// campo ausente queda en estadoDesconocido — no se inventa significado.
	estadoFinalizado   = "Finalizado"
	estadoProximamente = "Próximamente"
	estadoEnEmision    = "En emisión"
	estadoDesconocido  = "Desconocido"
)

// errSlugVacio marca un id de anime sin slug: no se pide nada a la red.
var errSlugVacio = errors.New("animeav1: el id del anime no contiene slug")

// GetAnimeDetails descarga el __data.json de /media/<slug> y devuelve el
// detalle más TODOS los episodios del array episodes, en una sola petición y
// sin emitir lotes (el flujo por lotes es StreamAnimeDetails, episodes.go).
//
// Un id vacío no toca la red; los fallos HTTP llegan tal cual de httpx y un
// slug inexistente (que SvelteKit contesta con HTTP 200 y un nodo de error
// 404) devuelve el mismo error tipado de "no encontrado" que recibe JKAnime.
func (a *AnimeAV1) GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	return a.detalleYEpisodios(ctx, id)
}

// detalleYEpisodios es el flujo común de GetAnimeDetails y
// StreamAnimeDetails: una única petición al __data.json del detalle y el
// desempaquetado devalue con el helper de T31 (PageData).
func (a *AnimeAV1) detalleYEpisodios(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	slug := strings.Trim(strings.TrimSpace(id), "/")
	if slug == "" {
		return models.AnimeDetail{}, nil, errSlugVacio
	}
	ruta := a.siteURL() + rutaDetalleMedia + url.PathEscape(slug) + rutaDatosPagina
	resp, err := httpx.Get(ctx, a.Doer, ruta, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "detail",
	})
	if err != nil {
		return models.AnimeDetail{}, nil, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDetalleBytes+1))
	if err != nil {
		return models.AnimeDetail{}, nil, fmt.Errorf("leyendo el __data.json del detalle: %w", err)
	}
	if len(data) > maxDetalleBytes {
		return models.AnimeDetail{}, nil, fmt.Errorf("el __data.json del detalle pesa %d bytes y el máximo es %d", len(data), maxDetalleBytes)
	}
	return detalleDeJSON(slug, ruta, data)
}

// detalleDeJSON convierte el cuerpo del __data.json en detalle + episodios.
// Sin campo media no hay con qué construir el detalle: error descriptivo,
// salvo que el cuerpo traiga el nodo de error 404 de SvelteKit, en cuyo caso
// se devuelve el error tipado de recurso no encontrado.
func detalleDeJSON(slug, ruta string, data []byte) (models.AnimeDetail, []models.Episode, error) {
	datos, err := PageData(data)
	if err != nil {
		if paginaInexistente(data) {
			return models.AnimeDetail{}, nil, errorNoEncontrado("detail", ruta)
		}
		return models.AnimeDetail{}, nil, fmt.Errorf("parseando el __data.json del detalle de %q: %w", slug, err)
	}
	media, ok := datos["media"].(map[string]any)
	if !ok {
		if paginaInexistente(data) {
			return models.AnimeDetail{}, nil, errorNoEncontrado("detail", ruta)
		}
		return models.AnimeDetail{}, nil, fmt.Errorf("animeav1: el __data.json del detalle de %q no trae media", slug)
	}
	return detalleDeMedia(slug, media)
}

// detalleDeMedia mapea el objeto media a models.AnimeDetail y convierte su
// lista de episodios. Es tolerante a campos ausentes: cada hueco queda en su
// valor cero (Type "", Status "Desconocido", Genres y Episodes vacíos) sin
// tumbar el conjunto; solo un episodes de tipo inesperado es error.
func detalleDeMedia(slug string, media map[string]any) (models.AnimeDetail, []models.Episode, error) {
	categoria, _ := media["category"].(map[string]any)
	tipo, _ := categoria["slug"].(string)

	lista, err := listaEpisodios(media)
	if err != nil {
		return models.AnimeDetail{}, nil, fmt.Errorf("animeav1: el detalle de %q: %w", slug, err)
	}
	kind := kindEpisodio(tipo, len(lista))
	episodios := convertirEpisodios(slug, textoDe(media["title"]), idTexto(media["id"]), kind, lista)

	// TotalEpisodes = episodesCount del sitio (el campo "Episodios" de la
	// ficha, como lo lee JKAnime); si no viene legible se usa la longitud
	// real de la lista. Ojo: no siempre coinciden (muestreo T32: Gintama
	// declara 201 y trae 200 pares, con el episodio 2 ausente).
	detalle := models.AnimeDetail{
		ID:            slug,
		Title:         textoDe(media["title"]),
		Provider:      providerName,
		Type:          tipo,
		Status:        textoEstado(media["status"]),
		Genres:        nombresGeneros(media["genres"]),
		Synopsis:      textoDe(media["synopsis"]),
		Cover:         portada(media["id"]),
		TotalEpisodes: enteroDe(media["episodesCount"], len(episodios)),
	}
	return detalle, episodios, nil
}

// listaEpisodios devuelve el array media.episodes. El campo puede venir
// ausente o null (una ficha sin episodios): eso es una lista vacía, no un
// error. Cualquier otro tipo sí lo es, porque no hay con qué poblar la UI.
func listaEpisodios(media map[string]any) ([]any, error) {
	campo, existe := media["episodes"]
	if !existe || campo == nil {
		return nil, nil
	}
	lista, ok := campo.([]any)
	if !ok {
		return nil, fmt.Errorf("episodes es %T, quería una lista", campo)
	}
	return lista, nil
}

// convertirEpisodios traduce la lista {id, number} del detalle a
// models.Episode. Las entradas mal formadas (sin número entero ≥ 0) se
// descartan y las repetidas dentro de la misma lista no se emiten dos veces
// (mismo criterio que JKAnime): nada de esto tumba el resto del detalle.
func convertirEpisodios(slug, titulo, mediaID string, kind models.EpisodeKind, lista []any) []models.Episode {
	episodios := make([]models.Episode, 0, len(lista))
	vistos := make(map[string]struct{}, len(lista))
	for _, item := range lista {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		num, ok := numeroDe(obj["number"])
		if !ok || num < 0 || num > math.MaxInt32 || num != math.Trunc(num) {
			continue
		}
		n := int(num)
		id, err := models.NewEpisodeID(providerName, slug, kind, n)
		if err != nil {
			continue
		}
		if _, ya := vistos[id]; ya {
			continue
		}
		vistos[id] = struct{}{}
		numero := n
		if kind == models.KindMovie {
			// Contrato de models.Episode: el número de una película es 0 (el
			// id ya no lo lleva) y así la tarjeta pinta "Película" en vez de
			// "Ep. 1". El número real del sitio se sigue usando para la
			// miniatura, que sí depende de él.
			numero = 0
		}
		episodios = append(episodios, models.Episode{
			ID:        id,
			Number:    numero,
			Title:     tituloEpisodio(titulo, kind, obj["title"], n),
			Thumbnail: miniatura(mediaID, n),
			Kind:      kind,
		})
	}
	return episodios
}

// tituloEpisodio genera el título por defecto cuando la página no trae uno
// (title null en todas las muestras). El patrón es el de los datos reales de
// JKAnime: "<Título del anime> - <número>"; una película de episodio único
// se queda con el título del anime, igual que en JKAnime. Un título del
// sitio, si algún día llega, manda sobre el generado.
func tituloEpisodio(titulo string, kind models.EpisodeKind, crudo any, n int) string {
	if sitio, ok := crudo.(string); ok && strings.TrimSpace(sitio) != "" {
		return strings.TrimSpace(sitio)
	}
	if kind == models.KindMovie {
		return titulo
	}
	if titulo == "" {
		return ""
	}
	return fmt.Sprintf("%s - %d", titulo, n)
}

// kindEpisodio decide models.EpisodeKind a partir de category.slug del
// sitio: especial → special, película → movie y el resto (TV Anime, OVA,
// ONA…) → episode. La película solo es movie con UN episodio: el id de movie
// no lleva número (T05), así que con dos o más colisionarían — el sitio sí
// tiene películas de más de un episodio (muestreo T32).
func kindEpisodio(categoria string, n int) models.EpisodeKind {
	switch categoria {
	case "pelicula":
		if n == 1 {
			return models.KindMovie
		}
	case "especial":
		return models.KindSpecial
	}
	return models.KindEpisode
}

// textoEstado traduce media.status al texto que pinta la propia página. Los
// tres valores observados (0/1/2) casan literalmente con su HTML; todo lo
// demás —otros números, fracciones o el campo ausente— es "Desconocido".
func textoEstado(v any) string {
	n, ok := numeroDe(v)
	if !ok || n < 0 || n > 100 || n != math.Trunc(n) {
		return estadoDesconocido
	}
	switch int(n) {
	case 0:
		return estadoFinalizado
	case 1:
		return estadoProximamente
	case 2:
		return estadoEnEmision
	}
	return estadoDesconocido
}

// enteroDe lee un entero no negativo del payload; lo que no se pueda leer
// (ausente, null, negativo o fracción) deja el valor por defecto. Se usa con
// episodesCount (0 también cuenta como ilegible: el total real es la lista).
func enteroDe(v any, porDefecto int) int {
	n, ok := numeroDe(v)
	if !ok || n <= 0 || n > math.MaxInt32 || n != math.Trunc(n) {
		return porDefecto
	}
	return int(n)
}

// nombresGeneros extrae genres[].name; la respuesta puede no traer géneros
// y entonces la lista sale vacía pero nunca nil (JSON [], como JKAnime).
func nombresGeneros(v any) []string {
	generos := make([]string, 0)
	lista, ok := v.([]any)
	if !ok {
		return generos
	}
	for _, item := range lista {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if nombre := textoDe(obj["name"]); nombre != "" {
			generos = append(generos, nombre)
		}
	}
	return generos
}

// miniatura arma la URL de la miniatura del episodio: screenshots/<mediaId>/
// <n>.jpg en el CDN. Sin id del sitio no hay miniatura (la tarjeta pinta el
// número en su hueco).
func miniatura(mediaID string, n int) string {
	if mediaID == "" {
		return ""
	}
	return miniaturaCDN + mediaID + "/" + strconv.Itoa(n) + ".jpg"
}

// paginaInexistente detecta el nodo de error 404 de SvelteKit: cuando
// /media/<slug> no existe, el __data.json responde HTTP 200 con
// {"type":"error","error":{…},"status":404} en vez de un 404 real (y el nodo
// de página pasa a ser el de auth, que no trae media).
func paginaInexistente(data []byte) bool {
	var doc struct {
		Nodes []json.RawMessage `json:"nodes"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return false
	}
	for _, nodo := range doc.Nodes {
		var e struct {
			Type   string `json:"type"`
			Status int    `json:"status"`
		}
		if json.Unmarshal(nodo, &e) == nil && e.Type == "error" && e.Status == http.StatusNotFound {
			return true
		}
	}
	return false
}

// errorNoEncontrado construye el mismo error tipado que httpx devuelve ante
// un 404 de verdad (el que recibe JKAnime): errors.Is(err, httpx.ErrNotFound)
// y *httpx.ScraperError con StatusCode 404. op es la operación que pidió el
// recurso ("detail" en el detalle, "sources" en las fuentes, T33).
func errorNoEncontrado(op, ruta string) error {
	return &httpx.ScraperError{
		Provider:   providerName,
		Operation:  op,
		URL:        ruta,
		StatusCode: http.StatusNotFound,
		Cause:      httpx.ErrNotFound,
	}
}

// textoDe lee una cadena del payload; lo que no sea string (null, número,
// objeto) deja la cadena vacía.
func textoDe(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
