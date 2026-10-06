// Package animeflv implementa scraper.Provider sobre animeflv.or.at, un
// sitio WordPress (no el clásico animeflv.net, dominio muerto). Solo ofrece
// búsqueda: es el fallback de búsqueda del proyecto, sin detalle ni fuentes
// de vídeo (T18). Las peticiones pasan por httpx, que envía User-Agent de
// navegador y devuelve errores tipados.
package animeflv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

const (
	// DefaultBaseURL es la URL base del sitio. La usa New cuando la base
	// viene vacía y el valor cero de AnimeFLV al construir la petición.
	DefaultBaseURL = "https://animeflv.or.at"

	// providerName es el identificador estable del proveedor: el que
	// devuelve Name, el que se registra en el Registry (T04) y el que viaja
	// en models.Anime.Provider.
	providerName = "animeflv"

	// searchQuery es la clave del parámetro de búsqueda del sitio: la
	// búsqueda es /?s=<query> (WordPress), nunca /browse?q=.
	searchQuery = "s"

	// rutaAnime es el prefijo del que se extrae el slug de cada resultado:
	// https://animeflv.or.at/anime/<slug>/ → <slug>.
	rutaAnime = "/anime/"

	// maxSearchBytes limita el HTML aceptado en una búsqueda (10 MB): un
	// cuerpo mayor se rechaza sin parsearlo.
	maxSearchBytes = 10 << 20
)

// AnimeFLV implementa scraper.Provider sobre animeflv.or.at. Sus
// dependencias (doer y URL base) se inyectan para que los tests corran contra
// un httptest.Server sin salir a la red (regla 4). El valor cero es
// utilizable: BaseURL vacía cae en DefaultBaseURL y, sin doer, httpx
// devuelve ErrNilDoer sin red.
type AnimeFLV struct {
	// BaseURL del sitio, sin barra final (vacía → DefaultBaseURL).
	BaseURL string
	// Doer con el que se hacen las peticiones (httpx.HTTPDoer; *http.Client
	// lo cumple).
	Doer httpx.HTTPDoer
}

var _ scraper.Provider = (*AnimeFLV)(nil)

// New crea el proveedor con el doer y la URL base inyectados (entregable de
// T18). La base se normaliza (sin espacios ni barra final) y, si viene
// vacía, se usa DefaultBaseURL.
func New(doer httpx.HTTPDoer, baseURL string) *AnimeFLV {
	a := &AnimeFLV{BaseURL: baseURL, Doer: doer}
	a.BaseURL = a.siteURL()
	return a
}

// Name identifica al proveedor ante el Registry.
func (a *AnimeFLV) Name() string { return providerName }

// Search busca query en el sitio (/?s=…) y devuelve los resultados con el
// slug del href como id, nunca derivado del título. La query se escapa con
// url.QueryEscape. Una página sin resultados no es error; una tarjeta sin
// título o con un enlace que no apunta a /anime/<slug>/ se omite.
//
// La petición pasa por httpx.Get: envía User-Agent de navegador
// (DefaultUserAgent; el UA por defecto de Go puede ser bloqueado), propaga
// ctx y falla con *httpx.ScraperError si el status no es utilizable.
func (a *AnimeFLV) Search(ctx context.Context, query string) ([]models.Anime, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.Anime{}, nil
	}
	base := a.siteURL()
	resp, err := httpx.Get(ctx, a.Doer, base+"/?"+searchQuery+"="+url.QueryEscape(query), httpx.ReqOptions{
		Provider:  providerName,
		Operation: "search",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSearchBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leyendo la página de búsqueda: %w", err)
	}
	if len(data) > maxSearchBytes {
		return nil, fmt.Errorf("la página de búsqueda pesa %d bytes y el máximo es %d", len(data), maxSearchBytes)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parseando la página de búsqueda: %w", err)
	}
	return a.parseSearch(doc, base), nil
}

// GetAnimeDetails no existe para AnimeFLV: el proveedor no ofrece página de
// detalle (HasDetails: false), así que devuelve scraper.ErrNoDetails. El
// llamador lo comprueba con errors.Is para mostrar "Detalle no disponible
// para este proveedor".
func (a *AnimeFLV) GetAnimeDetails(_ context.Context, _ string) (models.AnimeDetail, []models.Episode, error) {
	return models.AnimeDetail{}, nil, scraper.ErrNoDetails
}

// GetVideoSources no existe para AnimeFLV (solo es fallback de búsqueda):
// devuelve un error descriptivo, distinto de scraper.ErrNoDetails, que habla
// de detalle y no de fuentes.
func (a *AnimeFLV) GetVideoSources(_ context.Context, episodeID string, _ scraper.SourceOpts) ([]models.VideoSource, error) {
	return nil, fmt.Errorf("animeflv: este proveedor no ofrece fuentes de vídeo (episodio %q)", episodeID)
}

// parseSearch convierte cada .search-series-card del documento en
// models.Anime, en el orden del HTML. Cada tarjeta aporta el enlace
// (a.thumbnail-link), el título (h2.entry-title) y la portada
// (img.anime-image); de ahí se extrae el slug que hace de id.
//
// El slug sale de la ruta del href (/anime/<slug>/), no de su host: los
// enlaces del sitio son absolutos apuntando a animeflv.or.at, y mirar solo
// la ruta mantiene el parseo igual con el sitio real y con un
// httptest.Server que sirve el fixture tal cual.
func (a *AnimeFLV) parseSearch(doc *goquery.Document, base string) []models.Anime {
	out := make([]models.Anime, 0)
	doc.Find(".search-series-card").Each(func(_ int, s *goquery.Selection) {
		href := strings.TrimSpace(s.Find("a.thumbnail-link").AttrOr("href", ""))
		titulo := strings.TrimSpace(s.Find("h2.entry-title").Text())
		if titulo == "" || href == "" {
			return
		}
		slug := slugDe(href)
		if slug == "" {
			return // enlace que no es /anime/<slug>/: lo descartamos
		}
		out = append(out, models.Anime{
			ID:       slug,
			Title:    titulo,
			CoverURL: absoluta(base, s.Find("img.anime-image").AttrOr("src", "")),
			Provider: providerName,
		})
	})
	return out
}

// slugDe extrae el slug de un enlace del sitio
// (https://animeflv.or.at/anime/bleach/ o /anime/bleach/ → bleach).
// Devuelve "" si la ruta no es exactamente /anime/<slug> (sin slug, con
// segmentos de más o con una ruta distinta).
func slugDe(href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(u.Path, rutaAnime) {
		return ""
	}
	slug := strings.Trim(strings.TrimPrefix(u.Path, rutaAnime), "/")
	if slug == "" || strings.Contains(slug, "/") {
		return ""
	}
	return slug
}

// absoluta resuelve ref (portada u otro enlace relativo) contra la URL base;
// una URL ya absoluta o un ref vacío se devuelve tal cual.
func absoluta(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	b, err := url.Parse(strings.TrimRight(base, "/") + "/")
	if err != nil {
		return ref
	}
	u, err := b.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}

// siteURL devuelve la URL base sin barra final; con el valor cero (o en
// blanco) devuelve DefaultBaseURL.
func (a *AnimeFLV) siteURL() string {
	base := strings.TrimRight(strings.TrimSpace(a.BaseURL), "/")
	if base == "" {
		return DefaultBaseURL
	}
	return base
}
