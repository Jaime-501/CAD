package extractors

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// MediaFireResolver resuelve las entradas de MediaFire de `var servers`.
// La página del archivo publica el enlace real de descarga —un archivo
// temporal con firma en downloadN.mediafire.com— dentro de su propio
// HTML, así que Resolve hace una sola petición y devuelve ese enlace tal
// cual. La firma caduca en minutos: el enlace no se persiste ni se
// cachea, se entrega al reproductor y punto (regla general 9).
//
// Cumple RemoteResolver (T14): ServersExtractor le entrega su doer
// compartido, así que cookies y User-Agent del flujo viajan (reglas 3 y
// 4); no lleva estado y se instancia como MediaFireResolver{}.
type MediaFireResolver struct{}

const (
	// nombreMediaFire es la etiqueta del servidor que sale en
	// VideoSource.Server cuando el remote no trae nombre.
	nombreMediaFire = "Mediafire"
	// maxMediaFireBytes limita la lectura de la página del archivo (el
	// fixture real pesa ~330 KB; el límite es contra respuestas gigantes).
	maxMediaFireBytes = 2 << 20
)

var (
	// reAnclaDescarga recoge los <a ...> del cuerpo. El orden de los
	// atributos cambia entre plantillas (en el fixture real href va
	// antes que id), así que primero se localiza el ancla por id o
	// aria-label y después se extrae su href.
	reAnclaDescarga = regexp.MustCompile(`(?is)<a\s[^>]*>`)
	// reBotonDescarga reconoce el ancla del botón del fixture real:
	// <a class="input popsok" aria-label="Download file" href="…" id="downloadButton">.
	reBotonDescarga = regexp.MustCompile(`(?is)\bid\s*=\s*["']downloadButton["']|\baria-label\s*=\s*["']Download file["']`)
	reHref          = regexp.MustCompile(`(?is)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	// reHostDescarga exige el host del enlace firmado
	// (downloadN.mediafire.com): cualquier otro destino es basura.
	reHostDescarga = regexp.MustCompile(`(?i)^download[0-9]*\.mediafire\.com$`)
)

// CanResolve decide si este resolver se encarga de la entrada: el nombre
// del servidor, sin distinguir mayúsculas ni espacios (la página escribe
// "Mediafire"). Comprobación barata y sin red.
func (MediaFireResolver) CanResolve(server string) bool {
	return strings.EqualFold(strings.TrimSpace(server), nombreMediaFire)
}

// Resolve pide la página del archivo y devuelve el enlace de descarga
// como fuente MP4. El Referer queda vacío: se comprobó con probes de 1
// byte (curl.exe) que downloadN.mediafire.com responde 206 con y sin
// Referer, así que no hace falta enviarlo. Si el host llegara a exigirlo,
// aquí iría la URL de la propia página.
func (MediaFireResolver) Resolve(ctx context.Context, doer httpx.HTTPDoer, remote Remote) ([]models.VideoSource, error) {
	if remote.URL == nil {
		return nil, fmt.Errorf("%s: entrada sin URL", nombreMediaFire)
	}
	pagina := remote.URL.String()
	resp, err := httpx.Get(ctx, doer, pagina, httpx.ReqOptions{
		Provider:  "jkanime",
		Operation: "mediafire",
		Extractor: nombreMediaFire,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaFireBytes))
	if err != nil {
		return nil, fmt.Errorf("%s: leyendo la página del archivo: %w", nombreMediaFire, err)
	}

	bruto, err := enlaceDeDescarga(string(cuerpo))
	if err != nil {
		return nil, err
	}
	enlace, err := absolverURL(bruto, pagina)
	if err != nil {
		return nil, fmt.Errorf("%s: el enlace de descarga no es una URL http(s): %w", nombreMediaFire, err)
	}
	if !reHostDescarga.MatchString(enlace.Host) {
		return nil, fmt.Errorf("%s: %q no es un host de descarga de Mediafire", nombreMediaFire, enlace.Host)
	}

	nombre := strings.TrimSpace(remote.Nombre)
	if nombre == "" {
		nombre = nombreMediaFire
	}
	return []models.VideoSource{{
		URL:    enlace.String(),
		Type:   tipoDeRecurso(enlace, string(cuerpo)),
		Server: nombre,
	}}, nil
}

// enlaceDeDescarga extrae el href del ancla de descarga de la página.
// Tolerante al orden de atributos y a comillas simples o dobles; un href
// vacío o "#" (estado "Preparing Download" del fixture) no sirve.
func enlaceDeDescarga(cuerpo string) (string, error) {
	for _, ancla := range reAnclaDescarga.FindAllString(cuerpo, -1) {
		if !reBotonDescarga.MatchString(ancla) {
			continue
		}
		m := reHref.FindStringSubmatch(ancla)
		if m == nil {
			continue
		}
		href := strings.TrimSpace(m[1] + m[2])
		if href == "" || href == "#" {
			continue
		}
		return href, nil
	}
	return "", fmt.Errorf("%s: la página no trae ningún enlace de descarga (id=downloadButton / aria-label)", nombreMediaFire)
}
