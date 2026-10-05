package jkanime

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
)

const (
	// searchPath es la ruta real de búsqueda del sitio, verificada en vivo
	// (fixture testdata/search.html): /buscar/<query>. El formulario del
	// sitio manda ?q=… y el servidor redirige a esta misma ruta.
	searchPath = "/buscar/"

	// maxSearchBytes limita el HTML aceptado en una búsqueda (10 MB), al
	// estilo de maxExceptionsBytes (T06): un cuerpo mayor se rechaza sin
	// parsearlo.
	maxSearchBytes = 10 << 20
)

// Search busca query en JKAnime y devuelve los resultados con el id real
// tomado del href (el slug), nunca derivado del título. La query se escapa
// como segmento de ruta. Una página sin resultados no es error y un item sin
// título, sin href o sin slug se omite.
func (j *JkAnime) Search(ctx context.Context, query string) ([]models.Anime, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.Anime{}, nil
	}
	base := j.siteURL()
	resp, err := httpx.Get(ctx, j.Doer, base+searchPath+url.PathEscape(query), httpx.ReqOptions{
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
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("URL base inválida %q: %w", base, err)
	}
	return j.parseSearch(ctx, doc, baseURL), nil
}

// parseSearch convierte los items del directorio de resultados en
// models.Anime, en el orden del documento. Cada href se absolutiza contra la
// base, se pasa por el resolver de excepciones (repara enlaces truncados del
// sitio) y de ahí se extrae el slug que hace de id.
func (j *JkAnime) parseSearch(ctx context.Context, doc *goquery.Document, base *url.URL) []models.Anime {
	out := make([]models.Anime, 0)
	resolver := j.resolver()
	doc.Find(".page_directorio .anime__item").Each(func(_ int, s *goquery.Selection) {
		enlace := s.Find("h5 a").First()
		titulo := strings.TrimSpace(enlace.Text())
		href := strings.TrimSpace(enlace.AttrOr("href", ""))
		if titulo == "" || href == "" {
			return
		}
		resuelta := resolver.Resolve(ctx, absoluta(base, href))
		slug := slugFromURL(resuelta.URL)
		if slug == "" {
			return
		}
		out = append(out, models.Anime{
			ID:       slug,
			Title:    titulo,
			CoverURL: absoluta(base, strings.TrimSpace(s.Find(".anime__item__pic").First().AttrOr("data-setbg", ""))),
			Provider: providerName,
		})
	})
	return out
}

// absoluta resuelve ref (href o portada) contra base y devuelve la URL tal
// cual si no se puede parsear; un ref vacío se queda vacío.
func absoluta(base *url.URL, ref string) string {
	if ref == "" {
		return ""
	}
	u, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}

// slugFromURL extrae el slug del path de una URL del sitio
// (https://jkanime.net/bleach/ → bleach). Devuelve "" si no hay path.
func slugFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.Trim(u.Path, "/")
}
