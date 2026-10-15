package embeds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// MP4Upload resuelve los embeds de MP4Upload. Según el informe de adds
// (docs/investigacion/adds.md, P4) es el único host RESOLVIBLE limpio: el
// HTML del embed trae la URL del MP4 en claro dentro del setup de videojs
// —player.src({ type: "video/mp4", src: … })—. La URL no lleva query (el
// token va embebido en el path) y es temporal: ni se cachea ni se persiste.
// (El informe daba por no exigido el Referer, pero en vivo el CDN responde
// 403 sin un Referer del dominio mp4upload.com; ver el bloque de firmas.)
//
// Recibe su httpx.HTTPDoer por campo/constructor (regla 4): así conserva
// cookies, User-Agent y sesión del flujo que lo use. Sin doer, httpx
// devuelve ErrNilDoer sin tocar la red.
type MP4Upload struct {
	// Doer con el que se pide el embed (httpx.HTTPDoer; *http.Client lo
	// cumple).
	Doer httpx.HTTPDoer
}

var _ Resolver = (*MP4Upload)(nil)

const (
	// nombreMP4Upload es la etiqueta del servidor que sale en
	// VideoSource.Server y el prefijo de los mensajes de error.
	nombreMP4Upload = "MP4Upload"

	// maxEmbedBytes limita la lectura del HTML del embed (el fixture real
	// pesa ~5 KB; el límite es contra respuestas gigantes).
	maxEmbedBytes = 2 << 20
)

// Errores tipados (errors.Is). El HTML de las páginas válidas se parsea sin
// ejecutar JavaScript; los fallos HTTP (4xx/5xx) llegan tal cual de httpx
// como *ScraperError.
var (
	// ErrSinMedio: la página del embed no trae ninguna URL de vídeo
	// utilizable (embed borrado/degradado o formato cambiado).
	ErrSinMedio = errors.New("embeds: la página del embed no trae ninguna URL de vídeo")
	// ErrCuerpoGrande: el HTML del embed supera maxEmbedBytes.
	ErrCuerpoGrande = errors.New("embeds: la página del embed supera el tamaño máximo")
	// ErrURLInvalida: la URL recibida no es http(s) con host.
	ErrURLInvalida = errors.New("embeds: la URL del embed no es http(s) con host")
)

// hostsMP4Upload son los hosts del sitio de MP4Upload (con o sin www). El
// host del CDN del vídeo (a2.mp4upload.com) no está: esa ya es la fuente, no
// un embed que resolver.
var hostsMP4Upload = map[string]struct{}{
	"mp4upload.com":     {},
	"www.mp4upload.com": {},
}

var (
	// reEmbedMP4Upload es la forma del iframe que publican ambos
	// proveedores: /embed-<id>.html (informe de adds, P1 y fixtures).
	reEmbedMP4Upload = regexp.MustCompile(`(?i)^/embed-[A-Za-z0-9]+\.html$`)
	// reDirectoMP4Upload es la forma directa de la página del archivo:
	// /<id> (una sola ruta alfanumérica de 6+ caracteres; los ids reales
	// son de 12), la que aparece en los enlaces de descarga de AnimeAV1.
	reDirectoMP4Upload = regexp.MustCompile(`(?i)^/[A-Za-z0-9]{6,}/?$`)
	// rePlayerSrc localiza el bloque del setup de videojs que trae el
	// medio: player.src({ … }).
	rePlayerSrc = regexp.MustCompile(`(?is)player\.src\s*\(\s*\{(.*?)\}\s*\)`)
	// reSrcAtributo extrae la propiedad src del bloque (comillas simples o
	// dobles).
	reSrcAtributo = regexp.MustCompile(`(?is)\bsrc\s*:\s*(?:"([^"]*)"|'([^']*)')`)
)

// NewMP4Upload crea el resolver con el doer inyectado.
func NewMP4Upload(doer httpx.HTTPDoer) *MP4Upload {
	return &MP4Upload{Doer: doer}
}

// Name identifica al resolver y etiqueta sus fuentes.
func (MP4Upload) Name() string { return nombreMP4Upload }

// CanHandle reconoce las URLs de MP4Upload del informe: la forma de embed
// (/embed-<id>.html, con o sin www) y la forma directa (/<id>). Comprobación
// barata y sin red; cualquier otro host, esquema o ruta es falso.
func (MP4Upload) CanHandle(url string) bool {
	u, err := urlHTTP(strings.TrimSpace(url))
	if err != nil {
		return false
	}
	if _, ok := hostsMP4Upload[strings.ToLower(u.Hostname())]; !ok {
		return false
	}
	return reEmbedMP4Upload.MatchString(u.Path) || reDirectoMP4Upload.MatchString(u.Path)
}

// Resolve descarga el embed con el doer inyectado y devuelve una única
// fuente MP4 con la URL que publica el setup de videojs. No se ejecuta JS:
// la URL sale del HTML crudo.
//
// La fuente lleva Referer = la propia URL del embed: el CDN responde 403 sin
// un Referer del dominio mp4upload.com y 206 con él (verificado en vivo el
// 2026-10-15; desviación del informe de adds, que lo daba por no exigido).
// Headers queda vacío.
//
// referer, si se indica, se envía al pedir el embed (el propio host puede
// condicionar la página a que venga de un episodio). Un embed sin medios
// devuelve ErrSinMedio; los fallos HTTP se propagan como *ScraperError.
func (r MP4Upload) Resolve(ctx context.Context, url, referer string) ([]models.VideoSource, error) {
	if _, err := urlHTTP(strings.TrimSpace(url)); err != nil {
		return nil, fmt.Errorf("%s: %w", nombreMP4Upload, ErrURLInvalida)
	}
	opts := httpx.ReqOptions{
		Provider:  nombreMP4Upload,
		Operation: "mp4upload",
		Extractor: nombreMP4Upload,
	}
	if referer != "" {
		opts.Referer = referer
	}
	resp, err := httpx.Get(ctx, r.Doer, url, opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, maxEmbedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%s: leyendo la página del embed: %w", nombreMP4Upload, err)
	}
	if len(cuerpo) > maxEmbedBytes {
		return nil, fmt.Errorf("%s: %w (%d bytes)", nombreMP4Upload, ErrCuerpoGrande, len(cuerpo))
	}

	enlace, err := enlaceDePlayer(string(cuerpo))
	if err != nil {
		return nil, err
	}
	return []models.VideoSource{{
		URL:     enlace,
		Type:    models.ResourceMP4,
		Server:  nombreMP4Upload,
		Referer: url,
	}}, nil
}

// enlaceDePlayer extrae la URL del MP4 del bloque player.src({ ... }) del
// setup de videojs. Es el único elemento que el informe de adds anota para
// MP4Upload (1 ocurrencia en el embed, cero etiquetas <source>): primero se
// aísla el bloque y después se lee su propiedad src.
func enlaceDePlayer(cuerpo string) (string, error) {
	bloque := rePlayerSrc.FindStringSubmatch(cuerpo)
	if bloque == nil {
		return "", fmt.Errorf("%s: %w: no aparece player.src({...})", nombreMP4Upload, ErrSinMedio)
	}
	m := reSrcAtributo.FindStringSubmatch(bloque[1])
	if m == nil {
		return "", fmt.Errorf("%s: %w: player.src sin propiedad src", nombreMP4Upload, ErrSinMedio)
	}
	bruto := strings.TrimSpace(m[1] + m[2])
	if bruto == "" {
		return "", fmt.Errorf("%s: %w: player.src con src vacío", nombreMP4Upload, ErrSinMedio)
	}
	u, err := urlHTTP(bruto)
	if err != nil {
		return "", fmt.Errorf("%s: %w: la URL del vídeo no es http(s) con host", nombreMP4Upload, ErrSinMedio)
	}
	return u.String(), nil
}

// urlHTTP parsea una URL y exige http/https con host; devuelve ErrURLInvalida
// en cualquier otro caso (relativas, javascript:, data:, sin host).
func urlHTTP(bruta string) (*url.URL, error) {
	u, err := url.Parse(bruta)
	if err != nil {
		return nil, ErrURLInvalida
	}
	esquema := strings.ToLower(u.Scheme)
	if (esquema != "http" && esquema != "https") || u.Host == "" {
		return nil, ErrURLInvalida
	}
	return u, nil
}
