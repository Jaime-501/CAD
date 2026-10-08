// Package animeav1 implementa scraper.Provider sobre animeav1.com, un sitio
// SvelteKit (SSR) sin cookies, sin CSRF y sin rate-limit observado (informe
// T30). Cada página expone los mismos datos en <ruta>/__data.json, un JSON en
// formato devalue; esa es la vía principal de este paquete (T31), más robusta
// que extraer el literal JavaScript del <script> de hidratación.
//
// Alcance de T31: búsqueda (GET /catalogo/__data.json?search=…) y el
// desempaquetador devalue. El detalle llega en T32 y las fuentes de vídeo en
// T33; por eso GetAnimeDetails y GetVideoSources devuelven un error
// descriptivo todavía.
package animeav1

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

const (
	// DefaultBaseURL es la URL base del sitio. La usa New cuando la base
	// viene vacía y el valor cero de AnimeAV1 al construir la petición.
	DefaultBaseURL = "https://animeav1.com"

	// providerName es el identificador estable del proveedor: el que
	// devuelve Name, el que se registra en el Registry y el que viaja en
	// models.Anime.Provider.
	providerName = "animeav1"

	// rutaBusqueda es el endpoint de búsqueda: el __data.json del catálogo,
	// con el mismo parámetro search que usa el formulario del sitio
	// (<form method="get" action="/catalogo">). No existe /?s= ni /browse?q=.
	rutaBusqueda = "/catalogo/__data.json"

	// portadaCDN es el prefijo de las portadas: todo el arte vive en
	// cdn.animeav1.com, no exige Referer y responde siempre .jpg
	// (informe T30, P6). Las portadas van por media id, no por slug.
	portadaCDN = "https://cdn.animeav1.com/covers/"

	// maxBusquedaBytes limita el JSON aceptado en una búsqueda (10 MB): un
	// cuerpo mayor se rechaza sin parsearlo.
	maxBusquedaBytes = 10 << 20
)

// AnimeAV1 implementa scraper.Provider sobre animeav1.com. Sus dependencias
// (doer y URL base) se inyectan para que los tests corran contra un
// httptest.Server sin salir a la red (regla 4). El valor cero es utilizable:
// BaseURL vacía cae en DefaultBaseURL y, sin doer, httpx devuelve ErrNilDoer
// sin red.
type AnimeAV1 struct {
	// BaseURL del sitio, sin barra final (vacía → DefaultBaseURL).
	BaseURL string
	// Doer con el que se hacen las peticiones (httpx.HTTPDoer; *http.Client
	// lo cumple).
	Doer httpx.HTTPDoer
}

var _ scraper.Provider = (*AnimeAV1)(nil)

// New crea el proveedor con el doer y la URL base inyectados. La base se
// normaliza (sin espacios ni barra final) y, si viene vacía, se usa
// DefaultBaseURL.
func New(doer httpx.HTTPDoer, baseURL string) *AnimeAV1 {
	a := &AnimeAV1{BaseURL: baseURL, Doer: doer}
	a.BaseURL = a.siteURL()
	return a
}

// Name identifica al proveedor ante el Registry.
func (a *AnimeAV1) Name() string { return providerName }

// Search busca query en el catálogo pidiendo el __data.json de /catalogo y
// devuelve cada resultado como models.Anime: ID es el slug (nunca el título),
// Title el título y CoverURL la portada absoluta del CDN. El id del sitio va
// como string y solo se usa para la portada.
//
// Devuelve la primera página del catálogo, con 20 registros por página como
// mucho (pagination.recordsPerPage del propio JSON); no se pagina ni se piden
// más páginas. Una query vacía no hace ninguna petición y un catálogo sin
// resultados devuelve un slice vacío, no un error.
func (a *AnimeAV1) Search(ctx context.Context, query string) ([]models.Anime, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.Anime{}, nil
	}
	peticion := a.siteURL() + rutaBusqueda + "?search=" + url.QueryEscape(query)
	resp, err := httpx.Get(ctx, a.Doer, peticion, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "search",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBusquedaBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leyendo el __data.json de la búsqueda: %w", err)
	}
	if len(data) > maxBusquedaBytes {
		return nil, fmt.Errorf("el __data.json de la búsqueda pesa %d bytes y el máximo es %d", len(data), maxBusquedaBytes)
	}
	datos, err := PageData(data)
	if err != nil {
		return nil, fmt.Errorf("parseando el __data.json de la búsqueda: %w", err)
	}
	return resultadosAnime(datos)
}

// GetAnimeDetails se implementa en T32 (detalle de /media/<slug> con los
// episodios embebidos). Devuelve un error descriptivo, no scraper.ErrNoDetails
// (el proveedor sí ofrecerá detalle).
func (a *AnimeAV1) GetAnimeDetails(_ context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	return models.AnimeDetail{}, nil, fmt.Errorf("animeav1: no implementado todavía: T32 (detalle de %q)", id)
}

// GetVideoSources se implementa en T33 (página /media/<slug>/<n>). Devuelve
// un error descriptivo; opts no llega a usarse mientras no haya extracción.
func (a *AnimeAV1) GetVideoSources(_ context.Context, episodeID string, _ scraper.SourceOpts) ([]models.VideoSource, error) {
	return nil, fmt.Errorf("animeav1: no implementado todavía: T33 (fuentes de vídeo de %q)", episodeID)
}

// resultadosAnime convierte los datos de la página en models.Anime. Cada
// entrada de results debe ser un objeto con slug y título; las entradas
// incompletas o de otro tipo se descartan (mismo criterio que JKAnime y
// AnimeFLV con sus tarjetas) y una lista ausente es un error de formato.
func resultadosAnime(datos map[string]any) ([]models.Anime, error) {
	crudo, existe := datos["results"]
	if !existe {
		return nil, fmt.Errorf("animeav1: el __data.json no trae results")
	}
	items, ok := crudo.([]any)
	if !ok {
		return nil, fmt.Errorf("animeav1: results es %T, quería una lista", crudo)
	}
	out := make([]models.Anime, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		slug, _ := obj["slug"].(string)
		titulo, _ := obj["title"].(string)
		if slug == "" || titulo == "" {
			continue
		}
		out = append(out, models.Anime{
			ID:       slug,
			Title:    titulo,
			CoverURL: portada(obj["id"]),
			Provider: providerName,
		})
	}
	return out, nil
}

// portada arma la URL de la portada a partir del id numérico del sitio, que
// results trae como string ("195"). Un id que no se pueda leer deja la
// portada vacía en lugar de romper la búsqueda.
func portada(id any) string {
	texto := ""
	switch t := id.(type) {
	case string:
		texto = t
	case float64:
		texto = strconv.FormatFloat(t, 'f', -1, 64)
	}
	if texto == "" {
		return ""
	}
	return portadaCDN + texto + ".jpg"
}

// siteURL devuelve la URL base sin barra final; con el valor cero (o en
// blanco) devuelve DefaultBaseURL.
func (a *AnimeAV1) siteURL() string {
	base := strings.TrimRight(strings.TrimSpace(a.BaseURL), "/")
	if base == "" {
		return DefaultBaseURL
	}
	return base
}
