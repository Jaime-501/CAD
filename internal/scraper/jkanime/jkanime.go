package jkanime

import (
	"context"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/jkanime/extractors"
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

// GetAnimeDetails delega en el flujo de episodios de T09 (episodes.go): el
// detalle de la página de serie más TODOS los episodios del AJAX, sin emitir
// lotes.
func (j *JkAnime) GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	return j.getAnimeDetails(ctx, id)
}

// GetVideoSources descarga la página del episodio (T10), la pasa por la
// cadena de extractores —JKPlayer → JKPHP → Servers con MediaFire— y
// devuelve las fuentes ya normalizadas: la validación, la deduplicación
// y la completado de tipos los hace Normalize dentro de Resolve (T16).
// No se cachea nada: los enlaces son temporales (regla 9), por lo que
// opts.NoCache hoy no cambia el comportamiento.
//
// Los errores llegan tal cual de fetchEpisodePage (404, 403, 419...) o
// de la cadena (extractors.ErrSinFuentes cuando ningún extractor aporta
// una fuente reproducible, envolviendo también ctx.Err() si se canceló).
func (j *JkAnime) GetVideoSources(ctx context.Context, episodeID string, opts scraper.SourceOpts) ([]models.VideoSource, error) {
	pagina, err := j.fetchEpisodePage(ctx, episodeID)
	if err != nil {
		return nil, err
	}
	return j.cadenaDeExtractores().Resolve(ctx, paginaExtractores(pagina))
}

// cadenaDeExtractores monta la cadena de T17 con el doer del proveedor
// (mismo cliente y cookiejar que el resto del flujo, reglas 3 y 4) y con
// MediaFire inyectado como RemoteResolver del extractor de servidores.
// New se encarga de ordenarla en canónico: JKPlayer → JKPHP → Servers.
func (j *JkAnime) cadenaDeExtractores() *extractors.Resolver {
	base := extractors.Base{Doer: j.Doer}
	return extractors.New(
		&extractors.JKPlayerExtractor{Base: base},
		&extractors.JKPHPExtractor{Base: base},
		&extractors.ServersExtractor{
			Base:      base,
			Resolvers: []extractors.RemoteResolver{&extractors.MediaFireResolver{}},
		},
	)
}

// paginaExtractores convierte la página cruda del episodio (T10) en la
// Page que procesan los extractores (T11): mismos campos y semántica
// (URL final tras redirecciones, HTML crudo, Headers sin Set-Cookie),
// porque extractors no puede importar este paquete (regla de dependencias).
func paginaExtractores(p EpisodePage) extractors.Page {
	return extractors.Page{URL: p.URL, HTML: p.HTML, Headers: p.Headers}
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
