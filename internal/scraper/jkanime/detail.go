package jkanime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

const (
	// maxDetailBytes limita el HTML aceptado en una página de detalle
	// (10 MB), al estilo de maxSearchBytes: un cuerpo mayor se rechaza sin
	// parsearlo.
	maxDetailBytes = 10 << 20

	// Valores normalizados de models.AnimeDetail.Type (T03/T08).
	tipoSerie       = "series"
	tipoPelicula    = "movie"
	tipoOVA         = "ova"
	tipoEspecial    = "special"
	tipoDesconocido = "unknown"
)

// errSlugVacio marca un id de anime sin slug: no se pide nada a la red.
var errSlugVacio = errors.New("jkanime: el id del anime no contiene slug")

// detailMeta reúne los datos internos de la página de detalle que no se
// exponen en models (el id numérico y el CSRF viven solo aquí) y que T09
// necesita para llamar a /ajax/episodes/<id>/. Regla 10: nunca se registran
// en logs.
type detailMeta struct {
	// NumericID es el id numérico del anime en el sitio (atributo
	// data-anime), p. ej. "278". Vacío si la página no lo trae.
	NumericID string
	// CSRFToken es el contenido del meta csrf-token que el sitio exige en
	// sus POST de AJAX. Vacío si la página no lo trae.
	CSRFToken string
}

// fetchDetail descarga la página de detalle de slug (GET <base>/<slug>/) y
// devuelve el metadato normalizado más los datos internos (id numérico y
// CSRF) que usa T09. Un slug vacío no hace ninguna petición; los fallos HTTP
// llegan como *httpx.ScraperError (404, 403, challenge de Cloudflare, 5xx…).
func (j *JkAnime) fetchDetail(ctx context.Context, slug string) (models.AnimeDetail, detailMeta, error) {
	slug = strings.Trim(strings.TrimSpace(slug), "/")
	if slug == "" {
		return models.AnimeDetail{}, detailMeta{}, errSlugVacio
	}
	base := j.siteURL()
	resp, err := httpx.Get(ctx, j.Doer, base+"/"+url.PathEscape(slug)+"/", httpx.ReqOptions{
		Provider:  providerName,
		Operation: "detail",
	})
	if err != nil {
		return models.AnimeDetail{}, detailMeta{}, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDetailBytes+1))
	if err != nil {
		return models.AnimeDetail{}, detailMeta{}, err
	}
	if len(data) > maxDetailBytes {
		return models.AnimeDetail{}, detailMeta{}, fmt.Errorf("la página de detalle pesa %d bytes y el máximo es %d", len(data), maxDetailBytes)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return models.AnimeDetail{}, detailMeta{}, err
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return models.AnimeDetail{}, detailMeta{}, err
	}
	detalle, meta := parseDetail(doc, slug, baseURL)
	return detalle, meta, nil
}

// parseDetail extrae los metadatos de la página de detalle. Es tolerante:
// cada campo ausente queda en su valor cero (y el tipo en "unknown") sin
// convertir el conjunto en un error, porque el sitio cambia el contenido de
// una serie a otra.
func parseDetail(doc *goquery.Document, slug string, base *url.URL) (models.AnimeDetail, detailMeta) {
	campos := camposAnimeData(doc)

	detalle := models.AnimeDetail{
		ID:            slug,
		Title:         textoPrimero(doc, ".anime_info h3"),
		Provider:      providerName,
		Type:          normalizarTipo(textoSinEtiqueta(campos["tipo"])),
		Status:        textoSinEtiqueta(campos["estado"]),
		Genres:        generosDeCampo(campos["generos"]),
		Synopsis:      textoPrimero(doc, ".anime_info p.scroll"),
		Cover:         portada(doc, base),
		TotalEpisodes: enteroDeCampo(campos["episodios"]),
	}
	return detalle, detailMeta{
		NumericID: idNumerico(doc),
		CSRFToken: csrfToken(doc),
	}
}

// camposAnimeData indexa los <li> de la ficha del anime por su etiqueta en
// minúsculas sin los dos puntos (tipo, generos, episodios, estado…). La
// página repite la ficha para escritorio y móvil: gana la primera aparición.
func camposAnimeData(doc *goquery.Document) map[string]*goquery.Selection {
	campos := make(map[string]*goquery.Selection)
	doc.Find(".anime_data li").Each(func(_ int, s *goquery.Selection) {
		etiqueta := etiquetaDe(s)
		if etiqueta == "" {
			return
		}
		if _, ya := campos[etiqueta]; ya {
			return
		}
		campos[etiqueta] = s
	})
	return campos
}

// etiquetaDe devuelve el nombre del campo de un <li> de la ficha: el texto de
// su primer <span> en minúsculas y sin ":" ("Estado:" → "estado").
func etiquetaDe(s *goquery.Selection) string {
	etiqueta := strings.TrimSpace(s.Find("span").First().Text())
	etiqueta = strings.TrimSuffix(etiqueta, ":")
	return strings.ToLower(strings.TrimSpace(etiqueta))
}

// textoSinEtiqueta devuelve el valor de un <li> de la ficha quitándole la
// propia etiqueta del <span> ("Episodios: 366" → "366"). Con nil → "".
func textoSinEtiqueta(s *goquery.Selection) string {
	if s == nil {
		return ""
	}
	etiqueta := strings.TrimSpace(s.Find("span").First().Text())
	texto := strings.TrimSpace(s.Text())
	texto = strings.TrimSpace(strings.TrimPrefix(texto, etiqueta))
	return strings.TrimSpace(strings.TrimPrefix(texto, ":"))
}

// generosDeCampo recoge los textos de los <a> del <li> de géneros; sin
// género devuelve un slice vacío (JSON [], no null).
func generosDeCampo(s *goquery.Selection) []string {
	generos := make([]string, 0)
	if s == nil {
		return generos
	}
	s.Find("a").Each(func(_ int, a *goquery.Selection) {
		if texto := strings.TrimSpace(a.Text()); texto != "" {
			generos = append(generos, texto)
		}
	})
	return generos
}

// enteroDeCampo interpreta el valor de un <li> numérico ("366" → 366). Un
// campo ausente o no numérico deja el total en 0.
func enteroDeCampo(s *goquery.Selection) int {
	n, err := strconv.Atoi(textoSinEtiqueta(s))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// textoPrimero devuelve el texto del primer elemento que case con selector,
// recortado; si no hay coincidencia, "".
func textoPrimero(doc *goquery.Document, selector string) string {
	return strings.TrimSpace(doc.Find(selector).First().Text())
}

// portada devuelve la URL absoluta de la imagen de portada: la del bloque
// .anime_pic y, si no está, la del meta og:image (ambas en el HTML crudo).
func portada(doc *goquery.Document, base *url.URL) string {
	src := strings.TrimSpace(doc.Find(".anime_pic img").First().AttrOr("src", ""))
	if src == "" {
		src = strings.TrimSpace(doc.Find(`meta[property="og:image"]`).First().AttrOr("content", ""))
	}
	return absoluta(base, src)
}

// idNumerico devuelve el id numérico del anime (atributo data-anime de la
// ficha). Solo se acepta si son dígitos; en caso contrario queda "".
func idNumerico(doc *goquery.Document) string {
	id := strings.TrimSpace(doc.Find("#guardar-anime").First().AttrOr("data-anime", ""))
	if id == "" {
		id = strings.TrimSpace(doc.Find("[data-anime]").First().AttrOr("data-anime", ""))
	}
	if _, err := strconv.Atoi(id); err != nil {
		return ""
	}
	return id
}

// csrfToken lee el meta csrf-token de la página. Se compara con EqualFold
// porque goquery/net/html no garantizan la caja de los atributos.
func csrfToken(doc *goquery.Document) string {
	var token string
	doc.Find("meta").EachWithBreak(func(_ int, m *goquery.Selection) bool {
		if !strings.EqualFold(m.AttrOr("name", ""), "csrf-token") {
			return true
		}
		token = strings.TrimSpace(m.AttrOr("content", ""))
		return false
	})
	return token
}

// normalizarTipo traduce el tipo del sitio (español, con o sin acentos) a
// los valores de models.AnimeDetail.Type: series|movie|ova|special|unknown.
// Un valor vacío o no reconocido es unknown.
func normalizarTipo(raw string) string {
	t := minusculasSinAcentos(raw)
	switch t {
	case "serie", "series", "tv", "tv serie", "tv series", "serietv":
		return tipoSerie
	case "pelicula", "movie", "film", "pelcula":
		return tipoPelicula
	case "ova":
		return tipoOVA
	case "especial", "special", "especiales":
		return tipoEspecial
	case "":
		return tipoDesconocido
	}
	// Tolerancia a variantes con texto extra ("Serie TV", "Película OVA").
	switch {
	case strings.Contains(t, "pelicula") || strings.Contains(t, "movie") || strings.Contains(t, "film"):
		return tipoPelicula
	case strings.Contains(t, "ova"):
		return tipoOVA
	case strings.Contains(t, "especial") || strings.Contains(t, "special"):
		return tipoEspecial
	case strings.Contains(t, "serie"):
		return tipoSerie
	}
	return tipoDesconocido
}

// minusculasSinAcentos baja a minúsculas y quita las tildes y la ñ para
// comparar el tipo del sitio sin depender de la forma exacta en que el
// autor lo escriba ("Película" = "Pelicula").
func minusculasSinAcentos(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return acentosReemplazados.Replace(s)
}

// acentosReemplazados cubre las vocales tildadas y la ñ del español.
var acentosReemplazados = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
)
