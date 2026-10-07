package extractors

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// JKPlayerExtractor extrae las fuentes del player moderno de jkanime. La
// página de episodio embebe un iframe con la ruta jkplayer/um?... —o su
// variante umv?...— y la respuesta es una página DPlayer/video.js con la
// URL del .m3u8 (a menudo cifrada con atob) y, si la serie los tiene,
// pistas .vtt adjuntas (softsubs). Recibe su httpx.HTTPDoer por
// constructor (Base) para conservar cookies y sesión (regla 3).
type JKPlayerExtractor struct {
	Base
}

// maxPlayerBytes limita la respuesta del player: las reales pesan ~4–17 KB.
const maxPlayerBytes = 5 << 20

// rePlayerJK localiza la URL del iframe del player en el HTML del episodio:
// con esquema (https://jkanime.net/jkplayer/um?...) o raíz-relativa
// (/jkplayer/um?...), que es como la escribe el JavaScript del sitio antes
// de sustituir src="/" por el host.
var rePlayerJK = regexp.MustCompile(`(?:https?:)?//[^\s"'<>\\]+jkplayer/um[^\s"'<>\\]*|/[^\s"'<>\\]*jkplayer/um[^\s"'<>\\]*`)

// Candidatas a recurso dentro de la página del player. Cada una cubre una
// variante real de las respuestas de testdata/: la URL cifrada con atob de
// la configuración activa de DPlayer, la llamada a hls.loadSource, la
// etiqueta <source> del reproductor nativo y la propiedad url: del bloque
// video: {...} (también la copia comentada que el sitio deja en el HTML).
var (
	reURLAtob        = regexp.MustCompile(`\burl\s*:\s*atob\(\s*'([A-Za-z0-9+/=]+)'\s*\)`)
	reLoadSource     = regexp.MustCompile(`loadSource\(\s*'([^']+)'\s*\)`)
	reEtiquetaSource = regexp.MustCompile(`<source[^>]*\bsrc\s*=\s*['"]([^'"]+)['"]`)
	reURLDPlayer     = regexp.MustCompile(`\bvideo\s*:\s*\{[^{}]*?\burl\s*:\s*'([^']+)'`)
	reVTT            = regexp.MustCompile(`[^\s"'<>()\\,;={}]+\.vtt(?:\?[^\s"'<>\\]*)?`)
)

// Name identifica al extractor en la cadena (T11).
func (JKPlayerExtractor) Name() string { return NameJKPlayer }

// CanHandle reconoce cualquier página que mencione el player moderno.
// "jkplayer/um" cubre um?... y umv?...: es la misma familia de player.
func (JKPlayerExtractor) CanHandle(page Page) bool {
	return strings.Contains(page.HTML, "jkplayer/um")
}

// Extract sigue cada iframe del player, descifra su página y devuelve las
// fuentes. Cada fuente fija como Referer la URL del propio iframe: es la
// cabecera que el CDN del .m3u8 exige (captura con Playwright MCP, véase
// PROGRESO.md). Los enlaces repetidos del mismo vídeo —um y umv— se
// resuelven por recurso base, sin la query de firma temporal.
//
// T29: la implementación es la compartida con JKPHPExtractor
// (extraeIframesDePlayer), que solo cambia las etiquetas y los textos.
func (e JKPlayerExtractor) Extract(ctx context.Context, page Page) ([]models.VideoSource, error) {
	return extraeIframesDePlayer(ctx, page, rePlayerJK, cfgJKPlayer, e.Doer)
}

// cfgJKPlayer es la parametrización de JKPlayerExtractor para la descarga
// compartida de páginas de player (T29).
var cfgJKPlayer = cfgPlayer{
	nombre:  NameJKPlayer,
	mencion: "el player",
	pagina:  "del player",
	sujeto:  "el player",
}

// cfgPlayer parametriza lo que los dos extractores de página de player
// hacen con ella: la descarga, el parseo y la deduplicación son idénticos
// (T29) y solo cambian las etiquetas de log/resultado y los textos de error.
type cfgPlayer struct {
	// nombre es NameJKPlayer o NameJKPHP: Operation y Extractor del log
	// y Server de cada fuente (hoy coinciden los tres).
	nombre string
	// mencion es lo que el HTML declara mencionar: "el player" | "jk.php".
	mencion string
	// pagina es la frase de la página tras la preposición:
	// "del player" | "de jk.php".
	pagina string
	// sujeto encabeza el error cuando la página no expone vídeo:
	// "el player" | "la página".
	sujeto string
}

// extraeIframesDePlayer recorre los iframes que re localiza en la página
// del episodio, descarga cada página del player una sola vez (dedup por
// URL de iframe) y agrega sus fuentes deduplicándolas por recurso base (la
// query de firma temporal no distingue vídeos). Implementación compartida
// de JKPlayerExtractor.Extract y JKPHPExtractor.Extract (T29).
func extraeIframesDePlayer(ctx context.Context, page Page, re *regexp.Regexp, cfg cfgPlayer, doer httpx.HTTPDoer) ([]models.VideoSource, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	enlaces := re.FindAllString(page.HTML, -1)
	if len(enlaces) == 0 {
		return nil, fmt.Errorf("%s: el HTML menciona %s pero no contiene ninguna URL de iframe", cfg.nombre, cfg.mencion)
	}

	var (
		fuentes  []models.VideoSource
		errores  []error
		visitas  = map[string]bool{}
		producir = map[string]bool{}
	)
	for _, bruto := range enlaces {
		player, err := absolverURL(bruto, page.URL)
		if err != nil {
			errores = append(errores, fmt.Errorf("iframe %s: %w", cfg.pagina, err))
			continue
		}
		if visitas[player.String()] {
			continue // misma URL de iframe: no repetir la petición
		}
		visitas[player.String()] = true

		producidas, err := fuentesDePaginaDePlayer(ctx, doer, cfg, page, player)
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
				continue // mismo vídeo con otra firma: no duplicar
			}
			producir[clave] = true
			fuentes = append(fuentes, f)
		}
	}
	if len(fuentes) == 0 {
		if len(errores) > 0 {
			return nil, errors.Join(errores...)
		}
		return nil, fmt.Errorf("%s: sin URL de vídeo reproducible en la página %s", cfg.nombre, cfg.pagina)
	}
	return fuentes, nil
}

// fuentesDePaginaDePlayer descarga la página de un iframe del player y
// traduce sus candidatas a VideoSource con los helpers de parseo de este
// archivo (urlsDeVideo, absolverURL, recursoBase, tipoDeRecurso,
// subtitulosDe) y el límite maxPlayerBytes. La petición lleva el Referer
// de la página del episodio y, con el doer compartido, también sus cookies
// (regla 3); cada fuente usa como Referer la URL del propio iframe, que es
// la cabecera que exige su CDN.
func fuentesDePaginaDePlayer(ctx context.Context, doer httpx.HTTPDoer, cfg cfgPlayer, page Page, player *url.URL) ([]models.VideoSource, error) {
	resp, err := httpx.Get(ctx, doer, player.String(), httpx.ReqOptions{
		Provider:  "jkanime",
		Operation: cfg.nombre,
		Extractor: cfg.nombre,
		Referer:   page.URL,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, maxPlayerBytes+1))
	if err != nil {
		return nil, err
	}
	if len(cuerpo) > maxPlayerBytes {
		return nil, fmt.Errorf("la página %s pesa %d bytes y el máximo es %d", cfg.pagina, len(cuerpo), maxPlayerBytes)
	}

	subs := subtitulosDe(string(cuerpo), player)
	var fuentes []models.VideoSource
	visitas := map[string]bool{}
	for _, bruto := range urlsDeVideo(string(cuerpo)) {
		u, err := absolverURL(bruto, player.String())
		if err != nil {
			continue // javascript:/data:/basuras varias: no es reproducible
		}
		if visitas[recursoBase(u)] {
			continue
		}
		visitas[recursoBase(u)] = true
		fuentes = append(fuentes, models.VideoSource{
			URL:       u.String(),
			Type:      tipoDeRecurso(u, string(cuerpo)),
			Server:    cfg.nombre,
			Referer:   player.String(),
			Subtitles: subs,
		})
	}
	if len(fuentes) == 0 {
		return nil, fmt.Errorf("%s: %s no expone ninguna URL de vídeo válida", cfg.nombre, cfg.sujeto)
	}
	return fuentes, nil
}

// urlsDeVideo extrae del HTML/JS del player las candidatas a recurso en
// orden de prioridad: primero la configuración activa de DPlayer (URL
// cifrada con atob), después hls.loadSource, la etiqueta <source> y por
// último la propiedad url: del bloque video: {...}. Solo se aceptan
// candidatas que absolverURL valida como http(s).
func urlsDeVideo(cuerpo string) []string {
	var salidas []string
	for _, m := range reURLAtob.FindAllStringSubmatch(cuerpo, -1) {
		descifrado, err := descifrarBase64(m[1])
		if err != nil {
			continue
		}
		salidas = append(salidas, string(descifrado))
	}
	for _, re := range []*regexp.Regexp{reLoadSource, reEtiquetaSource, reURLDPlayer} {
		for _, m := range re.FindAllStringSubmatch(cuerpo, -1) {
			salidas = append(salidas, m[1])
		}
	}
	return salidas
}

// descifrarBase64 decodifica el argumento de atob(), tolerando el formato
// con y sin relleno final.
func descifrarBase64(cifrado string) ([]byte, error) {
	descifrado, err := base64.StdEncoding.DecodeString(cifrado)
	if err == nil {
		return descifrado, nil
	}
	return base64.RawStdEncoding.DecodeString(strings.TrimRight(cifrado, "="))
}

// subtitulosDe recoge las pistas .vtt referenciadas en la página del
// player (softsubs): absolutiza cada URL contra la propia página e infiere
// Lang del nombre del archivo o del parámetro lang= cuando es posible.
// Los fixtures reales actuales no incluyen .vtt (ver PROGRESO.md), por eso
// la inferencia se prueba con un HTML artificial en el test.
func subtitulosDe(cuerpo string, base *url.URL) []models.Subtitle {
	var subs []models.Subtitle
	visitas := map[string]bool{}
	for _, bruto := range reVTT.FindAllString(cuerpo, -1) {
		u, err := absolverURL(bruto, base.String())
		if err != nil {
			continue
		}
		if visitas[recursoBase(u)] {
			continue
		}
		visitas[recursoBase(u)] = true
		subs = append(subs, models.Subtitle{
			URL:  u.String(),
			Lang: codigoDeIdioma(u),
		})
	}
	return subs
}

// codigoDeIdioma infiere el código de idioma de una pista .vtt a partir
// del parámetro lang=/language= o del nombre del archivo (es.vtt,
// sub-en.vtt, espanol.vtt...). Devuelve "" si no se puede inferir.
func codigoDeIdioma(u *url.URL) string {
	q := u.Query()
	for _, clave := range []string{"lang", "language"} {
		if v := q.Get(clave); v != "" {
			if codigo := normalizaCodigo(v); codigo != "" {
				return codigo
			}
		}
	}
	nombre := strings.TrimSuffix(path.Base(u.Path), path.Ext(u.Path))
	for _, token := range strings.FieldsFunc(nombre, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' '
	}) {
		if codigo := normalizaCodigo(token); codigo != "" {
			return codigo
		}
	}
	return ""
}

// normalizaCodigo traduce alias habituales de idioma a su código ISO.
func normalizaCodigo(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "es", "spa", "esp", "spanish", "espanol", "español", "castellano":
		return "es"
	case "en", "eng", "english":
		return "en"
	case "pt", "por", "portuguese", "portugues", "portugués":
		return "pt"
	case "ja", "jpn", "jp", "japanese", "japones", "japonés":
		return "ja"
	case "fr", "fre", "fra", "french", "frances", "français":
		return "fr"
	case "it", "ita", "italian", "italiano":
		return "it"
	case "de", "ger", "deu", "german", "aleman", "alemán":
		return "de"
	case "ru", "rus", "russian", "ruso":
		return "ru"
	case "zh", "chi", "zho", "chinese", "chino":
		return "zh"
	case "ko", "kor", "korean", "coreano":
		return "ko"
	case "ar", "ara", "arabic", "arabe", "árabe":
		return "ar"
	case "ca", "catalan", "catala", "català":
		return "ca"
	case "gl", "galician", "gallego", "galego":
		return "gl"
	}
	return ""
}

// tipoDeRecurso decide el tipo por la extensión de la URL y, si no la
// tiene, por lo que declara el propio player. Sin señal devuelve "" y lo
// normaliza T16.
func tipoDeRecurso(u *url.URL, cuerpo string) models.ResourceType {
	switch strings.ToLower(path.Ext(u.Path)) {
	case ".m3u8", ".m3u":
		return models.ResourceHLS
	case ".mp4", ".mkv", ".webm":
		return models.ResourceMP4
	}
	switch {
	case strings.Contains(cuerpo, "application/x-mpegURL"),
		strings.Contains(cuerpo, "customHls"),
		strings.Contains(cuerpo, "loadSource"):
		return models.ResourceHLS
	case strings.Contains(cuerpo, "type: 'mp4'"), strings.Contains(cuerpo, `type: "mp4"`):
		return models.ResourceMP4
	}
	return ""
}

// absolverURL normaliza una URL cruda —incluida la entidad &amp; del
// HTML—, la resuelve contra base y exige un destino http(s) reproducible.
func absolverURL(bruto, base string) (*url.URL, error) {
	s := strings.ReplaceAll(strings.TrimSpace(bruto), "&amp;", "&")
	if s == "" {
		return nil, errors.New("URL vacía")
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if base != "" {
		if b, errBase := url.Parse(base); errBase == nil {
			u = b.ResolveReference(u)
		}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("esquema no soportado: %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, errors.New("URL sin host")
	}
	return u, nil
}

// recursoBase identifica un recurso sin su query de firma: las URLs del
// player cambian de firma a cada llamada pero apuntan al mismo vídeo.
func recursoBase(u *url.URL) string {
	sinQuery := *u
	sinQuery.RawQuery = ""
	sinQuery.Fragment = ""
	return sinQuery.String()
}
