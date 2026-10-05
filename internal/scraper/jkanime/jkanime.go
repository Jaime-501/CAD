package jkanime

import (
	"context"
	"errors"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

const (
	// DefaultBaseURL es la URL base del sitio JKAnime. La usa New cuando la
	// baseURL viene vacía y el valor cero de JkAnime al construir la petición.
	DefaultBaseURL = "https://jkanime.net"

	// providerName es el identificador estable del proveedor: el que
	// devuelve Name, el que se registra en el Registry (T04) y el que viaja
	// en models.Anime.Provider.
	providerName = "jkanime"
)

// ErrNotImplemented marcan los métodos de Provider que todavía no tienen
// implementación: los sustituyen T08 (detalle), T09/T10 (episodios) y
// T17/T20 (fuentes de vídeo). Se comprueba con errors.Is.
var ErrNotImplemented = errors.New("jkanime: operación no implementada todavía")

// JkAnime implementa scraper.Provider sobre jkanime.net. Sus dependencias se
// inyectan (doer, URL base y resolver de excepciones) para que los tests
// corran contra un httptest.Server sin tocar la red. El valor cero es
// utilizable: BaseURL vacía cae en DefaultBaseURL, un resolver nil se crea
// con el doer del cliente y, sin doer, httpx devuelve ErrNilDoer sin red.
type JkAnime struct {
	// BaseURL del sitio, sin barra final (vacía → DefaultBaseURL).
	BaseURL string
	// Doer con el que se hacen las peticiones (httpx.HTTPDoer; *http.Client
	// lo cumple).
	Doer httpx.HTTPDoer
	// Resolver aplica el mapa remoto de excepciones (T06) a las URL de los
	// resultados; nil → ExceptionResolver con el mismo doer.
	Resolver *ExceptionResolver
}

var _ scraper.Provider = (*JkAnime)(nil)

// New crea el proveedor con el doer, la URL base y el resolver inyectados.
// La base se normaliza (sin espacios ni barra final) y, si viene vacía, se
// usa DefaultBaseURL; un resolver nil se crea con el mismo doer.
func New(doer httpx.HTTPDoer, baseURL string, resolver *ExceptionResolver) *JkAnime {
	j := &JkAnime{BaseURL: baseURL, Doer: doer, Resolver: resolver}
	j.BaseURL = j.siteURL()
	if j.Resolver == nil {
		j.Resolver = &ExceptionResolver{Doer: doer}
	}
	return j
}

// Name identifica al proveedor ante el Registry.
func (j *JkAnime) Name() string { return providerName }

// GetAnimeDetails se implementa en T08.
func (j *JkAnime) GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	return models.AnimeDetail{}, nil, ErrNotImplemented
}

// GetVideoSources se implementa en T17/T20, cuando existan los extractores.
func (j *JkAnime) GetVideoSources(ctx context.Context, episodeID string, opts scraper.SourceOpts) ([]models.VideoSource, error) {
	return nil, ErrNotImplemented
}

// siteURL devuelve la URL base sin barra final; con el valor cero (o en
// blanco) devuelve DefaultBaseURL.
func (j *JkAnime) siteURL() string {
	base := strings.TrimRight(strings.TrimSpace(j.BaseURL), "/")
	if base == "" {
		return DefaultBaseURL
	}
	return base
}

// resolver devuelve el resolver inyectado; si no lo hay, uno nuevo con el
// doer del cliente (el mapa remoto se descarga de forma perezosa, T06).
func (j *JkAnime) resolver() *ExceptionResolver {
	if j.Resolver != nil {
		return j.Resolver
	}
	return &ExceptionResolver{Doer: j.Doer}
}
