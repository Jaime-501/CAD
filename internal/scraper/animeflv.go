package scraper

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"miapp/internal/models"
)

// Estructura vacía que implementará la interfaz Provider
type AnimeFLV struct{}

func NewAnimeFLV() *AnimeFLV {
	return &AnimeFLV{}
}

// Método para buscar animes
func (a *AnimeFLV) Search(query string) ([]models.Anime, error) {
	// El sitio es WordPress: la búsqueda es ?s= y los resultados
	// viven en .search-series-grid (animeflv.net ya no funciona)
	searchURL := fmt.Sprintf("https://animeflv.or.at/?s=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	// Algunos sitios bloquean el User-Agent por defecto de Go
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("animeflv devolvió HTTP %d", res.StatusCode)
	}

	// Parseamos el HTML con goquery
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	var results []models.Anime

	// Cada resultado es una tarjeta .search-series-card con su enlace, título e imagen
	doc.Find(".search-series-card").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Find("a.thumbnail-link").Attr("href") // ej: https://animeflv.or.at/anime/bleach/
		title := strings.TrimSpace(s.Find("h2.entry-title").Text())
		cover, _ := s.Find("img.anime-image").Attr("src")

		// Extraemos solo el slug del final de la URL
		id := strings.TrimSuffix(strings.TrimPrefix(href, "https://animeflv.or.at/anime/"), "/")
		if id == href {
			return // enlace inesperado, lo descartamos
		}

		results = append(results, models.Anime{
			ID:       id,
			Title:    title,
			CoverURL: cover,
			Provider: "animeflv",
		})
	})

	return results, nil
}

// Métodos vacíos (por ahora) para cumplir con la interfaz
func (a *AnimeFLV) GetAnimeDetails(id string) (models.Anime, []models.Episode, error) {
	return models.Anime{}, nil, nil
}

func (a *AnimeFLV) GetVideoURL(episodeID string) (string, error) {
	return "", nil
}
