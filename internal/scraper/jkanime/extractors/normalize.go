package extractors

import (
	"net/url"
	"path"
	"strings"

	"miapp/internal/models"
)

// Normalize convierte la lista heterogénea que agrega la cadena de
// extractores en un conjunto de fuentes reproducibles y comparables
// (T16), sin tocar la red:
//
//  1. valida que cada URL sea http(s) con host — descarta el resto
//     (javascript:, data:, relativas sueltas, vacías...);
//  2. completa Type cuando venga vacío, a partir de la extensión del
//     recurso (.m3u8 → HLS, .mp4 → MP4);
//  3. deduplica por recurso normalizado — esquema y host en minúsculas,
//     sin la query de firma temporal y conservando el fragmento, que en
//     algunos embeds (mega.nz) forma parte del descifrado—;
//  4. preserva Headers, Referer, Subtitles, Quality y Server: si una
//     fuente repetida trae datos que la guardada no tiene, se completan.
//
// El orden de entrada se mantiene (gana la primera aparición) y las URL
// firmadas se devuelven tal cual: son temporales y no se cachean
// (regla 9). Si no queda ninguna fuente devuelve nil.
func Normalize(fuentes []models.VideoSource) []models.VideoSource {
	if len(fuentes) == 0 {
		return nil
	}
	salida := make([]models.VideoSource, 0, len(fuentes))
	indice := make(map[string]int, len(fuentes))
	for _, f := range fuentes {
		u, ok := urlValida(f.URL)
		if !ok {
			continue // sin esquema http(s) no es reproducible
		}
		f.URL = u.String()
		if f.Type == "" {
			f.Type = tipoPorExtension(u)
		}
		clave := claveDe(u)
		if i, visto := indice[clave]; visto {
			completar(&salida[i], f)
			continue
		}
		indice[clave] = len(salida)
		salida = append(salida, f)
	}
	if len(salida) == 0 {
		return nil
	}
	return salida
}

// urlValida parsea la URL cruda —incluida la entidad &amp; del HTML—,
// exige un destino http(s) con host y normaliza esquema y host a
// minúsculas (el resto de la URL se conserva tal cual, porque el path
// distingue mayúsculas en muchos CDNs).
func urlValida(bruta string) (*url.URL, bool) {
	s := strings.ReplaceAll(strings.TrimSpace(bruta), "&amp;", "&")
	if s == "" {
		return nil, false
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, false
	}
	if u.Hostname() == "" {
		return nil, false
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.User == nil {
		u.Host = strings.ToLower(u.Host)
	}
	return u, true
}

// tipoPorExtension decide el tipo de un recurso directo por su extensión:
// es lo que hace falta para completar las fuentes que llegaron con Type
// vacío (pendiente de T12/T15). Sin extensión reconocible devuelve "" y
// la fuente se queda con el tipo que traiga —o sin él—.
func tipoPorExtension(u *url.URL) models.ResourceType {
	switch strings.ToLower(path.Ext(u.Path)) {
	case ".m3u8", ".m3u":
		return models.ResourceHLS
	case ".mp4", ".mkv", ".webm":
		return models.ResourceMP4
	}
	return ""
}

// claveDe identifica un recurso para deduplicar: sin la query —que solo
// cambia la firma temporal del mismo vídeo— y con el fragmento, que en
// los embeds de mega.nz lleva la clave de descifrado y sí distingue un
// vídeo de otro. Esquema y host ya llegan en minúsculas.
func claveDe(u *url.URL) string {
	sinFirma := *u
	sinFirma.RawQuery = ""
	return sinFirma.String()
}

// completar rellena con la fuente repetida los campos que la guardada
// trae vacíos, para no perder la información que aporta el duplicado
// (T16: preservar Headers, Referer y Subtitles). Nunca pisa un valor ya
// existente: gana la primera aparición.
func completar(guardada *models.VideoSource, repetida models.VideoSource) {
	if guardada.Type == "" {
		guardada.Type = repetida.Type
	}
	if guardada.Quality == "" {
		guardada.Quality = repetida.Quality
	}
	if guardada.Server == "" {
		guardada.Server = repetida.Server
	}
	if guardada.Referer == "" {
		guardada.Referer = repetida.Referer
	}
	if guardada.Headers == nil {
		guardada.Headers = repetida.Headers
	}
	if len(guardada.Subtitles) == 0 {
		guardada.Subtitles = repetida.Subtitles
	}
}
