// Package extractors define la arquitectura de extractores de vídeo del
// proveedor jkanime: cada extractor reconoce la página que sabe procesar
// (CanHandle) y extrae de ella las fuentes reproducibles (Extract); el
// Resolver ejecuta la cadena completa y agrega el resultado. Así ninguna
// extracción depende de una sola regex (regla 1) y la extracción de
// servidores queda separada de la resolución del recurso final (regla 2).
//
// Regla de dependencias (00_REGLAS): este paquete NO importa jkanime. La
// página llega como Page —jkanime la construye desde su EpisodePage— y el
// cliente HTTP se inyecta por constructor en cada extractor concreto
// (Base.Doer), de modo que cookies, User-Agent y sesión se conservan en
// todo el flujo (reglas 3 y 4).
package extractors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/models"
)

// Nombres canónicos de los extractores reales (contrato para T12–T15):
// son lo que Name() debe devolver y, en este orden, la cadena que ejecuta
// el Resolver: JKPlayer → JKPHP → Servers.
const (
	NameJKPlayer = "jkplayer"
	NameJKPHP    = "jkphp"
	NameServers  = "servers"
)

// ordenCanonico fija el orden de ejecución: el índice es la prioridad
// (menor = antes). Un extractor con otro nombre queda al final sin perder
// su orden de registro.
var ordenCanonico = []string{NameJKPlayer, NameJKPHP, NameServers}

// Page es la página cruda de un episodio que procesan los extractores. Es
// el equivalente a jkanime.EpisodePage (que este paquete no puede importar
// por la regla de dependencias): T17 la construye desde esa página.
type Page struct {
	// URL final del episodio, después de las redirecciones del sitio: se
	// usa para absolutizar rutas relativas y como Referer.
	URL string
	// HTML crudo del cuerpo de la respuesta.
	HTML string
	// Headers de la respuesta final, sin Set-Cookie (regla 10).
	Headers http.Header
}

// VideoExtractor extrae fuentes de vídeo de una página de episodio. Los
// extractores concretos (T12 JKPlayer, T13 JKPHP, T14 `var servers`,
// T15 MediaFire) reciben su httpx.HTTPDoer por constructor —véase Base— y
// deben etiquetar cada fuente con su VideoSource.Server para permitir el
// fallback de servidor del reproductor (si llega vacío, el Resolver la
// etiqueta con Name()).
type VideoExtractor interface {
	// Name identifica al extractor en la cadena y en los logs.
	Name() string
	// CanHandle decide si el extractor sabe procesar esta página. Debe
	// ser barato y sin efectos secundarios (no hace red).
	CanHandle(page Page) bool
	// Extract devuelve las fuentes de la página. Un error se registra en
	// el log y no detiene a los demás extractores.
	Extract(ctx context.Context, page Page) ([]models.VideoSource, error)
}

// Base es la base de los extractores concretos: cada uno recibe su
// httpx.HTTPDoer por constructor y lo incrusta aquí (regla 4, inyección de
// dependencias). Así el paquete no importa jkanime y los tests inyectan un
// doer falso o un cliente apuntando a un httptest.Server.
type Base struct {
	// Doer es el cliente HTTP compartido: conserva las cookies, la sesión
	// y el User-Agent del flujo (regla 3).
	Doer httpx.HTTPDoer
}

// ErrSinFuentes es la causa sentinela (errors.Is) que devuelve Resolve
// cuando la página no produjo ninguna fuente válida: ni una sola URL.
var ErrSinFuentes = errors.New("extractors: ningún extractor aportó fuentes de vídeo válidas")

// Resolver ejecuta la cadena de extractores sobre una página. La cadena se
// fija en New (no hay registro posterior), por eso Resolve solo la recorre
// y puede llamarse desde varias goroutines. El valor cero es utilizable y
// no aporta fuentes.
type Resolver struct {
	extractores []VideoExtractor
}

// New crea un resolver con los extractores recibidos, reordenados según el
// orden canónico (JKPlayer → JKPHP → Servers); los extractores con otro
// nombre conservan su orden de registro y quedan al final. La lista se
// copia: modificarla después no afecta al resolver.
func New(extractores ...VideoExtractor) *Resolver {
	copia := make([]VideoExtractor, 0, len(extractores))
	for _, e := range extractores {
		if e == nil {
			continue
		}
		copia = append(copia, e)
	}
	sort.SliceStable(copia, func(i, j int) bool {
		return prioridad(copia[i]) < prioridad(copia[j])
	})
	return &Resolver{extractores: copia}
}

// prioridad devuelve el puesto en la cadena: los del orden canónico van
// primero (en ese orden) y los desconocidos al final, en orden de registro
// gracias a la ordenación estable.
func prioridad(e VideoExtractor) int {
	nombre := strings.ToLower(strings.TrimSpace(e.Name()))
	for i, canonico := range ordenCanonico {
		if nombre == canonico {
			return i
		}
	}
	return len(ordenCanonico)
}

// Resolve ejecuta todos los extractores cuyo CanHandle acepta la página,
// en orden, y agrega sus fuentes válidas (con URL). No se detiene en el
// primero: un extractor que falla se registra en el log y se continúa con
// el siguiente. Devuelve error solo si el total de fuentes válidas es 0
// (ErrSinFuentes); si el contexto se canceló, el error lo envuelve también.
func (r *Resolver) Resolve(ctx context.Context, page Page) ([]models.VideoSource, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var total []models.VideoSource
	for _, e := range r.extractores {
		if !e.CanHandle(page) {
			continue
		}
		fuentes, err := e.Extract(ctx, page)
		if err != nil {
			slog.Warn("extractors: extractor falló; se continúa con el siguiente",
				"extractor", e.Name(),
				"err", logx.Redact(err.Error()))
			continue
		}
		for _, f := range fuentes {
			if strings.TrimSpace(f.URL) == "" {
				continue // sin URL no es reproducible: no cuenta
			}
			if f.Server == "" {
				f.Server = e.Name() // etiqueta para el fallback del reproductor
			}
			total = append(total, f)
		}
	}
	if len(total) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrSinFuentes, err)
		}
		return nil, ErrSinFuentes
	}
	return total, nil
}
