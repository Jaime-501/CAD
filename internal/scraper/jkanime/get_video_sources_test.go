package jkanime

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/jkanime/extractors"
)

// Constantes compartidas por el flujo offline y el smoke de red (T17).
const (
	// remoteMediafirePagina es el base64 del remote Mediafire tal y como
	// viene en testdata/episode_page.html (y en extractors/testdata/
	// servers.html): apunta a mediafire.com y hay que reescribirlo al
	// servidor local para no salir a la red.
	remoteMediafirePagina = "aHR0cHM6Ly9tZWRpYWZpcmUuY29tL2ZpbGUvYWF4dWRzczI3dHFxN2M4Lwo="

	// rutaArchivoMediaFire es la ruta real de ese remote (fixture de T15).
	rutaArchivoMediaFire = "/file/aaxudss27tqq7c8/"

	// iframeJKPHP reproduce la marca que reconoce JKPHPExtractor.CanHandle.
	// Es HTML artificial: la página real de bleach/1 no trae ningún iframe
	// de jk.php/jkplayer/jk (verificado sobre el fixture), así que sin él
	// ese extractor no llegaría a ejecutarse en el flujo completo
	// (regla 11; anotado en PROGRESO.md).
	iframeJKPHP = `<iframe class="player_conte" src="/jkplayer/jk?u=stream/jkmedia/prueba" width="565" height="318"></iframe>`
)

// fixtureExtractores lee un HTML real del testdata de extractors/ (solo
// lectura): los players, `var servers` y la página de Mediafire viven allí.
func fixtureExtractores(t *testing.T, nombre string) string {
	t.Helper()
	dato, err := os.ReadFile(filepath.Join("extractors", "testdata", nombre))
	if err != nil {
		t.Fatalf("fixture extractors/testdata/%s: %v", nombre, err)
	}
	return string(dato)
}

// servidorFlujo sirve las páginas del flujo completo (episodio, players,
// Mediafire) en un único httptest y registra todas las peticiones que
// recibe, para comprobar que la cadena pide exactamente lo esperado.
type servidorFlujo struct {
	srv        *httptest.Server
	mu         sync.Mutex
	peticiones []string
}

// nuevoServidorFlujo monta el servidor con las rutas indicadas. Las que
// se añadan después (la del episodio, que necesita la URL del propio
// servidor para reescribir el HTML) se registran antes de la primera
// petición, así que no hay carrera con el handler.
func nuevoServidorFlujo(t *testing.T, rutas map[string]string) *servidorFlujo {
	t.Helper()
	f := &servidorFlujo{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.peticiones = append(f.peticiones, r.URL.Path)
		f.mu.Unlock()
		cuerpo, ok := rutas[r.URL.Path]
		if !ok {
			http.Error(w, "no encontrado", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, cuerpo)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *servidorFlujo) rutasPedidas() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.peticiones...)
}

// transporteSoloLocal corta cualquier petición que no vaya al servidor
// del test: así el flujo es offline de verdad y un iframe o un remote mal
// reescrito hace fallar la prueba en lugar de salir a internet
// (regla 13).
type transporteSoloLocal struct {
	servidor string
	base     http.RoundTripper
}

func (t transporteSoloLocal) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != t.servidor {
		return nil, fmt.Errorf("petición fuera del servidor de prueba: %s", r.URL)
	}
	return t.base.RoundTrip(r)
}

func hostDe(url string) string { return strings.TrimPrefix(url, "http://") }

// clienteFlujo construye el doer del flujo: apunta al servidor del test
// y bloquea cualquier otra red.
func clienteFlujo(srvURL string) httpx.HTTPDoer {
	return httpx.NewClient(httpx.ClientOptions{
		Transport: transporteSoloLocal{servidor: hostDe(srvURL), base: http.DefaultTransport},
	})
}

// resumenPorExtractor imprime, por extractor/etiqueta de servidor, cuántas
// fuentes devolvió la cadena y de qué tipo (hls/mp4/iframe). Lo usan el
// test offline y el smoke de red.
func resumenPorExtractor(fuentes []models.VideoSource) string {
	porServidor := map[string]map[models.ResourceType]int{}
	for _, f := range fuentes {
		if porServidor[f.Server] == nil {
			porServidor[f.Server] = map[models.ResourceType]int{}
		}
		porServidor[f.Server][f.Type]++
	}
	nombres := make([]string, 0, len(porServidor))
	for nombre := range porServidor {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)

	var b strings.Builder
	for _, nombre := range nombres {
		tipos := porServidor[nombre]
		claves := make([]models.ResourceType, 0, len(tipos))
		for t := range tipos {
			claves = append(claves, t)
		}
		slices.Sort(claves)
		var detalle []string
		total := 0
		for _, t := range claves {
			detalle = append(detalle, fmt.Sprintf("%s: %d", tipoOMismo(t), tipos[t]))
			total += tipos[t]
		}
		fmt.Fprintf(&b, "  %s → %d fuente(s) (%s)\n", nombre, total, strings.Join(detalle, ", "))
	}
	return b.String()
}

// tipoOMismo etiqueta los tipos para el volcado, incluido el Type vacío
// que T16 conserva cuando no hay extensión reconocible.
func tipoOMismo(t models.ResourceType) string {
	if t == "" {
		return "sin-tipo"
	}
	return string(t)
}

// paginaConMarcadores devuelve la página real con los iframes del player
// reescritos al servidor local (el fixture los escribe absolutos contra
// jkanime.net) y el remote de Mediafire apuntando a la página local.
func paginaConMarcadores(t *testing.T, srvURL string, extra ...string) string {
	t.Helper()
	pagina := strings.Replace(string(fixtureDetalle(t, "episode_page.html")),
		"https://jkanime.net", srvURL, -1)
	pagina = strings.Replace(pagina, remoteMediafirePagina,
		base64.StdEncoding.EncodeToString([]byte(srvURL+rutaArchivoMediaFire)), 1)
	return pagina + strings.Join(extra, "")
}

// TestGetVideoSourcesFlujoCompleto recorre GetVideoSources de principio a
// fin contra un servidor falso completo: descarga de la página del
// episodio (T10), cadena de extractores (T11–T15) y normalización
// (T16). Todo el HTML sale de los fixtures reales de testdata/.
func TestGetVideoSourcesFlujoCompleto(t *testing.T) {
	rutas := map[string]string{
		"/jkplayer/um":       fixtureExtractores(t, "jkplayer_um.html"),
		"/jkplayer/umv":      fixtureExtractores(t, "jkplayer_umv.html"),
		rutaArchivoMediaFire: fixtureExtractores(t, "mediafire.html"),
	}
	f := nuevoServidorFlujo(t, rutas)
	rutas["/bleach/1/"] = paginaConMarcadores(t, f.srv.URL)

	cli := clienteFlujo(f.srv.URL)
	j := New(cli, f.srv.URL, &ExceptionResolver{Doer: cli})
	// Sin resolvers de embeds: este test fija la extracción y la
	// normalización (T10–T16) sin la capa de T61, que tiene sus propias
	// pruebas offline con un resolver falso.
	j.Embeds = nil

	fuentes, err := j.GetVideoSources(context.Background(), "jkanime:bleach:episode:1", scraper.SourceOpts{})
	if err != nil {
		t.Fatalf("GetVideoSources: %v", err)
	}
	t.Logf("fuentes por extractor:\n%s", resumenPorExtractor(fuentes))

	porTipo := map[models.ResourceType]int{}
	for _, fuente := range fuentes {
		if fuente.Server == "" || fuente.URL == "" {
			t.Errorf("fuente incompleta: %+v", fuente)
		}
		if !strings.HasPrefix(fuente.URL, "http") {
			t.Errorf("URL no http(s): %q", fuente.URL)
		}
		porTipo[fuente.Type]++
	}
	if queria := map[models.ResourceType]int{
		models.ResourceHLS:    1, // um y umv: el mismo .m3u8 con firmas distintas
		models.ResourceMP4:    1, // el enlace firmado de Mediafire
		models.ResourceIframe: 9, // los 9 servidores con iframe directo
	}; len(fuentes) != 11 || porTipo[models.ResourceHLS] != queria[models.ResourceHLS] ||
		porTipo[models.ResourceMP4] != queria[models.ResourceMP4] ||
		porTipo[models.ResourceIframe] != queria[models.ResourceIframe] {
		t.Errorf("fuentes = %d con tipos %v, quería 11 con %v", len(fuentes), porTipo, queria)
	}

	mejor, ok := extractors.SelectBestSource(fuentes)
	if !ok || mejor.Type != models.ResourceHLS {
		t.Errorf("SelectBestSource = %+v (ok=%v), quería la fuente HLS", mejor, ok)
	}

	queria := []string{"/bleach/1/", "/jkplayer/um", "/jkplayer/umv", rutaArchivoMediaFire}
	if pedidas := f.rutasPedidas(); !slices.Equal(pedidas, queria) {
		t.Errorf("peticiones = %v, quería %v (nada más debe tocarse)", pedidas, queria)
	}
}

// TestGetVideoSourcesCadenaCompletaConJKPHP repite el flujo con el iframe
// de JKPHP añadido, para que la cadena completa (JKPlayer → JKPHP →
// Servers con MediaFire) se ejecute de verdad dentro de un mismo
// GetVideoSources.
func TestGetVideoSourcesCadenaCompletaConJKPHP(t *testing.T) {
	rutas := map[string]string{
		"/jkplayer/um":       fixtureExtractores(t, "jkplayer_um.html"),
		"/jkplayer/umv":      fixtureExtractores(t, "jkplayer_umv.html"),
		"/jkplayer/jk":       fixtureExtractores(t, "jkphp.html"),
		rutaArchivoMediaFire: fixtureExtractores(t, "mediafire.html"),
	}
	f := nuevoServidorFlujo(t, rutas)
	rutas["/bleach/1/"] = paginaConMarcadores(t, f.srv.URL, iframeJKPHP)

	cli := clienteFlujo(f.srv.URL)
	j := New(cli, f.srv.URL, &ExceptionResolver{Doer: cli})
	// Sin resolvers de embeds: este test fija la cadena completa de
	// extractores (T11–T15) sin la capa de T61 (pruebas propias).
	j.Embeds = nil

	fuentes, err := j.GetVideoSources(context.Background(), "jkanime:bleach:episode:1", scraper.SourceOpts{})
	if err != nil {
		t.Fatalf("GetVideoSources: %v", err)
	}
	t.Logf("fuentes por extractor:\n%s", resumenPorExtractor(fuentes))

	porTipo := map[models.ResourceType]int{}
	for _, fuente := range fuentes {
		porTipo[fuente.Type]++
	}
	if len(fuentes) != 12 || porTipo[models.ResourceHLS] != 1 ||
		porTipo[models.ResourceMP4] != 2 || porTipo[models.ResourceIframe] != 9 {
		t.Errorf("fuentes = %d con tipos %v, quería 12 con hls=1, mp4=2 (jkphp + mediafire), iframe=9",
			len(fuentes), porTipo)
	}

	// El orden canónico se respeta: jkplayer, después jkphp y después los
	// servidores (Mediafire, que es la primera entrada de `var servers`).
	if fuentes[0].Server != extractors.NameJKPlayer {
		t.Errorf("fuentes[0].Server = %q, quería %q", fuentes[0].Server, extractors.NameJKPlayer)
	}
	if fuentes[1].Server != extractors.NameJKPHP {
		t.Errorf("fuentes[1].Server = %q, quería %q", fuentes[1].Server, extractors.NameJKPHP)
	}
	if fuentes[2].Server != "Mediafire" || fuentes[2].Type != models.ResourceMP4 {
		t.Errorf("fuentes[2] = %+v, quería la delegada de Mediafire como MP4", fuentes[2])
	}

	mejor, ok := extractors.SelectBestSource(fuentes)
	if !ok || mejor.Type != models.ResourceHLS {
		t.Errorf("SelectBestSource = %+v (ok=%v), quería la fuente HLS", mejor, ok)
	}

	queria := []string{"/bleach/1/", "/jkplayer/um", "/jkplayer/umv", "/jkplayer/jk", rutaArchivoMediaFire}
	if pedidas := f.rutasPedidas(); !slices.Equal(pedidas, queria) {
		t.Errorf("peticiones = %v, quería %v", pedidas, queria)
	}
}
