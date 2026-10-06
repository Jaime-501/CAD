package stream

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// maxManifiesto limita cuánto se lee de un manifiesto para reescribirlo: si
// el origen manda algo mayor, se responde error en lugar de hinchar la
// memoria. Un .m3u8 real pesa decenas de KB (el de bleach, 47 KB).
const maxManifiesto = 5 << 20 // 5 MB

// reAtributoURI detecta URI="…" y URI=sin_comillas en una línea de tag:
// #EXT-X-KEY, #EXT-X-MAP, #EXT-X-MEDIA, #EXT-X-I-FRAME-STREAM-INF, etc.
var reAtributoURI = regexp.MustCompile(`URI=("([^"]*)"|([^,\s"]+))`)

// esManifiesto decide si la respuesta es un manifiesto HLS a partir de la
// extensión pedida y del Content-Type del origen.
func esManifiesto(u *url.URL, contentType string) bool {
	ext := strings.ToLower(path.Ext(u.Path))
	if ext == ".m3u8" || ext == ".m3u" {
		return true
	}
	return strings.Contains(strings.ToLower(contentType), "mpegurl")
}

// esSubtituloVTT decide si hay que servir el cuerpo como text/vtt: la
// extensión .vtt o un Content-Type ya correcto. El Content-Type del origen
// no siempre es fiable (a veces manda text/plain u octet-stream), así que el
// proxy lo fija.
func esSubtituloVTT(u *url.URL, contentType string) bool {
	if strings.ToLower(path.Ext(u.Path)) == ".vtt" {
		return true
	}
	return strings.Contains(strings.ToLower(contentType), "text/vtt")
}

// esPlaylist comprueba la marca #EXTM3U del cuerpo: sin ella no se reescribe
// nada (un .m3u8 con página de error o una respuesta rara se devuelve tal
// cual).
func esPlaylist(cuerpo []byte) bool {
	cuerpo = bytes.TrimLeft(cuerpo, "\xef\xbb\xbf \t\r\n")
	return bytes.HasPrefix(cuerpo, []byte("#EXTM3U"))
}

// reescribir recorre un manifiesto y devuelve una copia con todas sus URIs
// convertidas en URLs del proxy: tanto las líneas sueltas (segmentos y
// sub-playlists, absolutas o relativas) como los atributos URI= de los tags
// (claves #EXT-X-KEY, mapas #EXT-X-MAP, pistas #EXT-X-MEDIA). Las URIs
// relativas se resuelven contra base, la URL final del manifiesto (después
// de los redirects).
func reescribir(cuerpo []byte, base *url.URL, proxificar func(absoluta string) string) []byte {
	lineas := bytes.Split(cuerpo, []byte("\n"))
	for i, linea := range lineas {
		if len(linea) == 0 {
			continue
		}
		if linea[0] == '#' {
			lineas[i] = reescribirAtributos(linea, base, proxificar)
			continue
		}
		abs, ok := absoluta(base, string(linea))
		if !ok {
			continue
		}
		lineas[i] = []byte(proxificar(abs))
	}
	return bytes.Join(lineas, []byte("\n"))
}

// reescribirAtributos sustituye cada URI="…" / URI=… de un tag por su
// equivalente en el proxy, conservando las comillas y el resto de la línea.
// Los que no son http/https (data:, blob:,…) se dejan intactos.
func reescribirAtributos(linea []byte, base *url.URL, proxificar func(absoluta string) string) []byte {
	return reAtributoURI.ReplaceAllFunc(linea, func(match []byte) []byte {
		sub := reAtributoURI.FindSubmatch(match)
		referencia := sub[2]
		entrecomillado := len(referencia) > 0
		if !entrecomillado {
			referencia = sub[3]
		}
		abs, ok := absoluta(base, string(referencia))
		if !ok {
			return match
		}
		nueva := proxificar(abs)
		if entrecomillado {
			return []byte(`URI="` + nueva + `"`)
		}
		return []byte("URI=" + nueva)
	})
}

// absoluta resuelve una referencia contra base y valida que el resultado sea
// http/https con host. Devuelve false para todo lo demás (data:, urn:,
// referencias vacías o ilegibles).
func absoluta(base *url.URL, referencia string) (string, bool) {
	referencia = strings.TrimSpace(referencia)
	if referencia == "" || base == nil {
		return "", false
	}
	u, err := url.Parse(referencia)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	abs := base.ResolveReference(u)
	if abs.Host == "" || (abs.Scheme != "http" && abs.Scheme != "https") {
		return "", false
	}
	return abs.String(), true
}
