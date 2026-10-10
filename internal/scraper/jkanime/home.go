package jkanime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

const (
	// rutaEmision es el directorio filtrado por estado, primera sección del
	// Home (T39): solo la página 1, sin paginar (?p=).
	rutaEmision = "/directorio?estado=emision"

	// rutaPortada es la portada del sitio, de la que sale la sección de
	// animes recientes (T39).
	rutaPortada = "/"

	// maxItemsEmision y maxItemsRecientes son los topes por sección: los 30
	// items del JSON del directorio y las 32 tarjetas SSR de la portada.
	maxItemsEmision   = 30
	maxItemsRecientes = 32
)

// varAnimes marca el script del directorio que embebe el listado como JSON
// estricto (var animes = {…}); el Home no hace POST y no depende del meta
// csrf-token.
const varAnimes = "var animes ="

var _ scraper.HomeProvider = (*JkAnime)(nil)

// directorioPagina es el fragmento del JSON embebido que consume Home para
// la sección "emision": una paginación de Laravel con el listado en data.
// El resto de campos (current_page, links, per_page, total…) se ignoran.
type directorioPagina struct {
	Data []directorioItem `json:"data"`
}

// directorioItem es un anime del listado del directorio. Solo se usan slug,
// title e image; los demás campos del sitio (id, synopsis, studios, type,
// status, url, estado, tipo, base64_id, short_title) se ignoran.
type directorioItem struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Image string `json:"image"`
}

// Home devuelve las secciones del Home de JKAnime, en este orden:
// "Emision" (directorio filtrado, JSON embebido) y "Recientes" (bloque SSR
// de la portada). Cada sección se pide con el doer del paquete y la pausa
// habitual entre peticiones; el fallo de una sección no anula la otra (se
// avisa con slog y se sigue), y si fallan todas se devuelve el primer error.
// Los items pasan por el resolver de excepciones igual que en Search. Un ctx
// cancelado corta sin más peticiones y el error cumple errors.Is con
// context.Canceled.
func (j *JkAnime) Home(ctx context.Context) ([]models.HomeSection, error) {
	base := j.siteURL()
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("URL base inválida %q: %w", base, err)
	}

	pedidas := []struct {
		id, titulo, ruta string
		maxItems         int
		parsear          func(context.Context, *goquery.Document, *url.URL) ([]models.Anime, error)
	}{
		{"emision", "En emisión", rutaEmision, maxItemsEmision, j.parseEmision},
		{"recientes", "Animes recientes", rutaPortada, maxItemsRecientes, j.parseRecientes},
	}

	secciones := make([]models.HomeSection, 0, len(pedidas))
	var primerError error
	for i, p := range pedidas {
		if i > 0 && !esperar(ctx, pausaPaginas) {
			return secciones, fmt.Errorf("el Home de JKAnime se interrumpió: %w", ctx.Err())
		}
		if err := ctx.Err(); err != nil {
			return secciones, fmt.Errorf("el Home de JKAnime se interrumpió: %w", err)
		}
		doc, err := j.paginaHome(ctx, base, p.ruta)
		if err == nil {
			var items []models.Anime
			items, err = p.parsear(ctx, doc, baseURL)
			if err == nil {
				items = dedupHome(items, p.maxItems)
				if len(items) == 0 {
					continue
				}
				secciones = append(secciones, models.HomeSection{
					ID:    p.id,
					Title: p.titulo,
					Items: items,
				})
				continue
			}
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return secciones, fmt.Errorf("el Home de JKAnime se interrumpió: %w", ctxErr)
		}
		if primerError == nil {
			primerError = err
		}
		slog.Warn("jkanime: la sección del Home no está disponible",
			"seccion", p.id, "err", logx.Redact(err.Error()))
	}
	if len(secciones) == 0 && primerError != nil {
		return nil, fmt.Errorf("el Home de JKAnime no está disponible: %w", primerError)
	}
	return secciones, nil
}

// paginaHome descarga una página del Home con el doer del paquete (sin
// cliente nuevo) y el mismo límite de lectura que el resto de páginas del
// paquete (maxSearchBytes): un cuerpo mayor se rechaza sin parsearlo.
func (j *JkAnime) paginaHome(ctx context.Context, base, ruta string) (*goquery.Document, error) {
	resp, err := httpx.Get(ctx, j.Doer, base+ruta, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "home",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSearchBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leyendo la página del Home: %w", err)
	}
	if len(data) > maxSearchBytes {
		return nil, fmt.Errorf("la página del Home pesa %d bytes y el máximo es %d", len(data), maxSearchBytes)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parseando la página del Home: %w", err)
	}
	return doc, nil
}

// parseEmision extrae la sección "En emisión": el servidor no pinta tarjetas
// del directorio, embebe el listado como JSON estricto dentro del primer
// <script> que contiene var animes =. Se toma el texto tras la primera
// aparición y json.Decoder se detiene al final del primer valor JSON, sin
// depender de ;var mode ni de ningún otro delimitador. Sin ese script o con
// JSON inválido la sección falla (sin volcar HTML).
func (j *JkAnime) parseEmision(ctx context.Context, doc *goquery.Document, base *url.URL) ([]models.Anime, error) {
	var bloque string
	doc.Find("script").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		texto := s.Text()
		if i := strings.Index(texto, varAnimes); i >= 0 {
			bloque = strings.TrimSpace(texto[i+len(varAnimes):])
			return false
		}
		return true
	})
	if bloque == "" {
		return nil, errors.New("la página del directorio no trae el JSON var animes")
	}
	var pagina directorioPagina
	if err := json.NewDecoder(strings.NewReader(bloque)).Decode(&pagina); err != nil {
		return nil, fmt.Errorf("decodificando el JSON del directorio: %w", err)
	}

	resolver := j.resolver()
	out := make([]models.Anime, 0, len(pagina.Data))
	for _, item := range pagina.Data {
		slugJSON := strings.TrimSpace(item.Slug)
		titulo := strings.TrimSpace(item.Title)
		if slugJSON == "" || titulo == "" {
			slog.Debug("jkanime: item del directorio descartado (sin slug o título)", "slug", slugJSON)
			continue
		}
		// Mismo tratamiento de excepciones que Search: URL absoluta →
		// mapa → slug definitivo (T43 lo aplica también al directorio).
		resuelta := resolver.Resolve(ctx, absoluta(base, slugJSON))
		slug := slugFromURL(resuelta.URL)
		if slug == "" {
			continue
		}
		out = append(out, models.Anime{
			ID:       slug,
			Title:    titulo,
			CoverURL: absoluta(base, strings.TrimSpace(item.Image)),
			Provider: providerName,
		})
	}
	return out, nil
}

// parseRecientes extrae el bloque "Animes recientes" de la portada: tarjetas
// SSR .custom_thumb_home con el enlace y la portada dentro. El título está
// en el h5.card-title del padre (div.p-3.d-flex); si falta, se usa el alt
// del img. Esta sección no trae número de episodio y no se inventa ninguno.
func (j *JkAnime) parseRecientes(ctx context.Context, doc *goquery.Document, base *url.URL) ([]models.Anime, error) {
	resolver := j.resolver()
	out := make([]models.Anime, 0, maxItemsRecientes)
	doc.Find("div.custom_thumb_home").Each(func(_ int, s *goquery.Selection) {
		href := strings.TrimSpace(s.Find("a").First().AttrOr("href", ""))
		if href == "" {
			return
		}
		img := s.Find("img").First()
		titulo := strings.TrimSpace(s.Parent().Find("h5.card-title a").First().Text())
		if titulo == "" {
			titulo = strings.TrimSpace(img.AttrOr("alt", ""))
		}
		if titulo == "" {
			slog.Debug("jkanime: tarjeta de recientes descartada (sin título)", "href", href)
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
			CoverURL: absoluta(base, strings.TrimSpace(img.AttrOr("src", ""))),
			Provider: providerName,
		})
	})
	return out, nil
}

// dedupHome descarta los IDs repetidos conservando el orden de aparición y
// recorta la sección al máximo permitido.
func dedupHome(items []models.Anime, max int) []models.Anime {
	vistas := make(map[string]struct{}, len(items))
	out := make([]models.Anime, 0, len(items))
	for _, a := range items {
		if _, ok := vistas[a.ID]; ok {
			continue
		}
		vistas[a.ID] = struct{}{}
		out = append(out, a)
		if max > 0 && len(out) == max {
			break
		}
	}
	return out
}
