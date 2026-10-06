package extractors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// ServersExtractor extrae las fuentes de la lista `var servers` que la
// página del episodio declara en su HTML. Cada entrada trae el enlace del
// host cifrado en base64 (remote) y su etiqueta (server). El JavaScript
// de la página convierte esos remotes en iframes —construye
// jkplayer/c1?u=<remote>&s=<server>, cuya respuesta es un iframe hacia
// el remote descodificado—, así que este extractor emite ese iframe
// directamente, sin la vuelta intermedia. Las entradas que la propia
// página excluye del bloque de vídeo (Mediafire, 1Fichier: solo publican
// enlace de descarga) se delegan en un RemoteResolver si la cadena trae
// alguno que las soporte (T15: MediaFireResolver).
//
// No hace ninguna petición propia: recibe el doer compartido del flujo
// (Base) solo para pasárselo a los resolvers, de modo que cookies,
// User-Agent y sesión siguen viajando (reglas 3 y 4).
type ServersExtractor struct {
	Base
	// Resolvers secundarios para los servidores que no son un iframe
	// directo. La cadena completa se compone en T17.
	Resolvers []RemoteResolver
}

// RemoteResolver resuelve una entrada de `var servers` que la página no
// convierte en iframe reproducible. Lo implementa T15
// (MediaFireResolver); ServersExtractor lo invoca pasándole el doer
// compartido del flujo, nunca uno propio.
type RemoteResolver interface {
	// CanResolve decide si este resolver se encarga del servidor dado
	// (el nombre de la entrada, p. ej. "Mediafire"). Debe ser barato y
	// sin red.
	CanResolve(server string) bool
	// Resolve devuelve las fuentes de una entrada ya decodificada.
	Resolve(ctx context.Context, doer httpx.HTTPDoer, remote Remote) ([]models.VideoSource, error)
}

// Server es una entrada cruda de la lista `var servers`, con los campos
// tal y como los declara el JSON de la página (fixture real
// testdata/servers.html).
type Server struct {
	Remote string `json:"remote"` // URL del host, cifrada en base64
	Slug   string `json:"slug"`
	Server string `json:"server"` // nombre visible: "VOE", "Mediafire"…
	Lang   int    `json:"lang"`
	Size   string `json:"size"`
	Append int    `json:"append"`
}

// Remote es una entrada ya decodificada y validada: el enlace real del
// host, listo para convertirse en fuente Iframe o para delegarse en un
// RemoteResolver.
type Remote struct {
	Nombre string   // etiqueta del servidor (Server.Server)
	Slug   string   // Server.slug, por si el resolver lo necesita
	Tamano string   // Server.size ("150 MB")
	URL    *url.URL // remote decodificado de base64, http(s) con host
}

// reListaServers localiza la declaración de la variable en el HTML crudo
// del episodio (está dentro de un <script>, no en el DOM renderizado).
var reListaServers = regexp.MustCompile(`var\s+servers\s*=\s*`)

// servidoresSoloDescarga son los que el JavaScript de la propia página
// excluye del bloque de vídeo (val.server !== 'Mediafire' && val.server
// !== '1Fichier'): solo publican un enlace de descarga, así que no
// producen un iframe reproducible y toca resolverlos aparte.
func servidoresSoloDescarga(nombre string) bool {
	switch strings.ToLower(strings.TrimSpace(nombre)) {
	case "mediafire", "1fichier":
		return true
	default:
		return false
	}
}

// Name identifica al extractor en la cadena (T11): puesto canónico 2
// (JKPlayer → JKPHP → Servers).
func (ServersExtractor) Name() string { return NameServers }

// CanHandle reconoce la declaración de la lista, con la misma regla que
// usa Extract para localizarla. Comprobación barata sobre el HTML crudo,
// sin red.
func (ServersExtractor) CanHandle(page Page) bool {
	return reListaServers.MatchString(page.HTML)
}

// Extract parsea `var servers`, filtra las entradas inválidas y devuelve
// las fuentes: iframe con el remote descodificado para los servidores
// normales y delegación en el RemoteResolver para los de solo descarga.
// Los enlaces repetidos del mismo host no salen duplicados. Si todos los
// enlaces fallan o no producen fuente, devuelve el error que los explica.
func (e ServersExtractor) Extract(ctx context.Context, page Page) ([]models.VideoSource, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entradas, err := listaServers(page.HTML)
	if err != nil {
		return nil, err
	}

	var (
		fuentes  []models.VideoSource
		errores  []error
		visitas  = map[string]bool{}
		producir = map[string]bool{}
	)
	for _, entrada := range entradas {
		remote, err := decodificarRemote(entrada)
		if err != nil {
			errores = append(errores, err)
			continue
		}
		if visitas[recursoBase(remote.URL)] {
			continue // misma entrada repetida: no repetir la delegación
		}
		visitas[recursoBase(remote.URL)] = true

		producidas, err := e.fuentesDeEntrada(ctx, remote)
		if err != nil {
			errores = append(errores, err)
			continue
		}
		for _, f := range producidas {
			clave := f.URL
			if u, err := url.Parse(f.URL); err == nil {
				clave = recursoBase(u)
			}
			if producir[clave] {
				continue // mismo recurso con otra firma: no duplicar
			}
			producir[clave] = true
			fuentes = append(fuentes, f)
		}
	}
	if len(fuentes) == 0 {
		if len(errores) > 0 {
			return nil, errors.Join(errores...)
		}
		return nil, fmt.Errorf("%s: ninguna entrada de `var servers` produce una fuente reproducible", NameServers)
	}
	return fuentes, nil
}

// fuentesDeEntrada convierte una entrada ya decodificada. Los servidores
// de solo descarga se delegan en el RemoteResolver que los soporte; si no
// hay ninguno no hay fuente, porque la propia página no les crea iframe.
// Los demás se emiten como fuente Iframe con el remote descodificado:
// el mismo destino al que llega el wrapper jkplayer/c1 de la página.
// Referer, Quality y Headers quedan vacíos (T16): la fuente Iframe se
// carga en el reproductor, no se peta desde Go.
func (e ServersExtractor) fuentesDeEntrada(ctx context.Context, remote Remote) ([]models.VideoSource, error) {
	if !servidoresSoloDescarga(remote.Nombre) {
		return []models.VideoSource{{
			URL:    remote.URL.String(),
			Type:   models.ResourceIframe,
			Server: remote.Nombre,
		}}, nil
	}
	for _, r := range e.Resolvers {
		if r == nil || !r.CanResolve(remote.Nombre) {
			continue
		}
		return r.Resolve(ctx, e.Doer, remote)
	}
	return nil, fmt.Errorf("%s: %s solo publica descarga y no hay ningún resolver para él", NameServers, remote.Nombre)
}

// listaServers localiza `var servers =` en el HTML crudo y decodifica el
// valor como lista de Server. json.Decoder lee exactamente un valor JSON,
// así que un corchete dentro de una cadena no corta la lista.
func listaServers(html string) ([]Server, error) {
	local := reListaServers.FindStringIndex(html)
	if local == nil {
		return nil, fmt.Errorf("%s: el HTML no declara `var servers`", NameServers)
	}
	cola := html[local[1]:]
	inicio := strings.Index(cola, "[")
	if inicio < 0 {
		return nil, fmt.Errorf("%s: `var servers` no contiene ninguna lista JSON", NameServers)
	}
	var entradas []Server
	if err := json.NewDecoder(strings.NewReader(cola[inicio:])).Decode(&entradas); err != nil {
		return nil, fmt.Errorf("%s: `var servers` no es JSON válido: %w", NameServers, err)
	}
	return entradas, nil
}

// decodificarRemote valida una entrada y devuelve su remote descodificado:
// base64 → URL http(s) con host. Las entradas sin nombre, sin remote o
// con un remote que no es una URL reproducible se descartan aquí.
func decodificarRemote(entrada Server) (Remote, error) {
	nombre := strings.TrimSpace(entrada.Server)
	if nombre == "" {
		return Remote{}, errors.New("entrada de `var servers` sin nombre de servidor")
	}
	cifrado := strings.TrimSpace(entrada.Remote)
	if cifrado == "" {
		return Remote{}, fmt.Errorf("%s: entrada sin remote", nombre)
	}
	bruto, err := descifrarBase64(cifrado)
	if err != nil {
		return Remote{}, fmt.Errorf("%s: remote no es base64 válido: %w", nombre, err)
	}
	// absolverURL recorta el salto de línea final que el sitio deja en
	// algunos remotes y exige http(s) con host (descarta javascript:…).
	remote, err := absolverURL(string(bruto), "")
	if err != nil {
		return Remote{}, fmt.Errorf("%s: remote descodificado no es una URL http(s): %w", nombre, err)
	}
	return Remote{
		Nombre: nombre,
		Slug:   strings.TrimSpace(entrada.Slug),
		Tamano: strings.TrimSpace(entrada.Size),
		URL:    remote,
	}, nil
}
