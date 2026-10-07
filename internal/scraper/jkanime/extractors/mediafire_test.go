package extractors

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// El fixture real (bleach/1) publica la página del archivo bajo /file/
// y, dentro, un enlace de descarga con firma (sanitizada en el fixture
// como FIRMA_DE_PRUEBA): el host real es download1529.mediafire.com.
const (
	rutaPaginaMediaFire = "/file/aaxudss27tqq7c8/"
	enlaceFirmado       = "https://download1529.mediafire.com/FIRMA_DE_PRUEBA/aaxudss27tqq7c8/01.mp4"
)

// MediaFireResolver cumple el contrato de delegación de T14.
var _ RemoteResolver = MediaFireResolver{}

// servidorMediaFire sirve el fixture real en /file/aaxudss27tqq7c8/
// registrando lo que recibe: una única petición de página.
type servidorMediaFire struct {
	srv    *httptest.Server
	mu     sync.Mutex
	paths  []string
	ua     string
	refer  string
	cuerpo string
}

func nuevoServidorMediaFire(t *testing.T, cuerpo string) *servidorMediaFire {
	t.Helper()
	s := &servidorMediaFire{cuerpo: cuerpo}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.paths = append(s.paths, r.URL.Path)
		s.ua = r.Header.Get("User-Agent")
		s.refer = r.Header.Get("Referer")
		s.mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		_, _ = w.Write([]byte(s.cuerpo))
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *servidorMediaFire) remote(t *testing.T, nombre string) Remote {
	t.Helper()
	u, err := url.Parse(s.srv.URL + rutaPaginaMediaFire)
	if err != nil {
		t.Fatalf("parsear la URL de la página: %v", err)
	}
	return Remote{Nombre: nombre, URL: u}
}

func (s *servidorMediaFire) peticiones() (paths []string, ua, refer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.paths...), s.ua, s.refer
}

func TestMediaFireCanResolve(t *testing.T) {
	casos := []struct {
		nombre   string
		servidor string
		queria   bool
	}{
		{"nombre del sitio", "Mediafire", true},
		{"minúsculas", "mediafire", true},
		{"mayúsculas", "MEDIAFIRE", true},
		{"con espacios", "  Mediafire ", true},
		{"1Fichier no", "1Fichier", false},
		{"otro servidor", "VOE", false},
		{"vacío", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := (MediaFireResolver{}).CanResolve(c.servidor)
			if got != c.queria {
				t.Errorf("CanResolve(%q) = %v, quería %v", c.servidor, got, c.queria)
			}
		})
	}
}

// TestMediaFireExtraeDelFixtureReal pide la página del archivo (fixture
// real) y devuelve el enlace firmado como única fuente, con la etiqueta
// del remote y el resto de campos vacíos (T16). La petición lleva el
// User-Agent del doer compartido y no envía Referer.
func TestMediaFireExtraeDelFixtureReal(t *testing.T) {
	s := nuevoServidorMediaFire(t, cargarFixture(t, "mediafire.html"))
	remote := s.remote(t, "Mediafire")

	fuentes, err := (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), remote)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != enlaceFirmado {
		t.Errorf("URL = %q, quería el enlace firmado del fixture %q", f.URL, enlaceFirmado)
	}
	if f.Type != models.ResourceMP4 {
		t.Errorf("Type = %q, quería %q (.mp4)", f.Type, models.ResourceMP4)
	}
	if f.Server != "Mediafire" {
		t.Errorf("Server = %q, quería la etiqueta del remote", f.Server)
	}
	if f.Referer != "" || f.Quality != "" || f.Headers != nil || f.Subtitles != nil {
		t.Errorf("fuente = %+v: Referer/Quality/Headers/Subtitles deben quedar vacíos (T16)", f)
	}

	paths, ua, refer := s.peticiones()
	if len(paths) != 1 || paths[0] != rutaPaginaMediaFire {
		t.Errorf("peticiones = %v, quería una sola a %q", paths, rutaPaginaMediaFire)
	}
	if ua != httpx.DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería el de navegador del doer compartido", ua)
	}
	if refer != "" {
		t.Errorf("Referer = %q, quería vacío (verificado con probes de 1 byte que el host no lo exige)", refer)
	}
}

func TestMediaFireConservaEtiquetaDelRemote(t *testing.T) {
	s := nuevoServidorMediaFire(t, cargarFixture(t, "mediafire.html"))
	fuentes, err := (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), s.remote(t, "MEDIAFIRE (sitio)"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fuentes[0].Server != "MEDIAFIRE (sitio)" {
		t.Errorf("Server = %q, quería la etiqueta tal y como la trae el remote", fuentes[0].Server)
	}
}

// TestMediaFireSinEnlaceUtil cubre los casos en los que la página no
// aporta un enlace reproducible: sin botón de descarga, con el botón en
// estado "Preparing" (href="#") o apuntando a otro host. El HTML es
// artificial: el sitio solo sirve la página completa (ver PROGRESO.md).
func TestMediaFireSinEnlaceUtil(t *testing.T) {
	casos := []struct {
		nombre string
		html   string
		queria string
	}{
		{
			nombre: "sin botón de descarga",
			html:   `<!DOCTYPE html><html><body><h1>01.mp4</h1></body></html>`,
			queria: "ningún enlace de descarga",
		},
		{
			nombre: "botón en preparación",
			html:   `<a class="preparing" href="#">Preparing Download</a><a id="downloadButton" href="#">Download</a>`,
			queria: "ningún enlace de descarga",
		},
		{
			nombre: "enlace de otro host",
			html:   `<a aria-label="Download file" href="https://evil.example.com/01.mp4" id="downloadButton">Download</a>`,
			queria: "no es un host de descarga",
		},
		{
			nombre: "enlace relativo",
			html:   `<a id="downloadButton" href="/file/aaxudss27tqq7c8/01.mp4">Download</a>`,
			queria: "no es un host de descarga",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s := nuevoServidorMediaFire(t, c.html)
			_, err := (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), s.remote(t, "Mediafire"))
			if err == nil {
				t.Fatal("quería error cuando no hay enlace de descarga utilizable")
			}
			if !strings.Contains(err.Error(), c.queria) {
				t.Errorf("err = %v, contenía %q", err, c.queria)
			}
		})
	}
}

// TestMediaFireComillasSimples comprueba que el href se lee también con
// comillas simples y con el atributo id antes que href (el orden de la
// página real es el contrario).
func TestMediaFireComillasSimples(t *testing.T) {
	html := `<a id='downloadButton' href='` + enlaceFirmado + `' aria-label='Download file'>Download</a>`
	s := nuevoServidorMediaFire(t, html)
	fuentes, err := (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), s.remote(t, "Mediafire"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fuentes[0].URL != enlaceFirmado {
		t.Errorf("URL = %q, quería %q", fuentes[0].URL, enlaceFirmado)
	}
}

func TestMediaFireSinURL(t *testing.T) {
	_, err := (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), Remote{Nombre: "Mediafire"})
	if err == nil {
		t.Fatal("quería error sin URL de página")
	}
	if !strings.Contains(err.Error(), "sin URL") {
		t.Errorf("err = %v, quería el mensaje de sin URL", err)
	}
}

func TestMediaFireSinDoer(t *testing.T) {
	s := nuevoServidorMediaFire(t, cargarFixture(t, "mediafire.html"))
	_, err := (MediaFireResolver{}).Resolve(context.Background(), nil, s.remote(t, "Mediafire"))
	if err == nil {
		t.Fatal("quería error sin doer inyectado")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer (el doer lo inyecta ServersExtractor)", err)
	}
}

func TestMediaFireContextoCancelado(t *testing.T) {
	s := nuevoServidorMediaFire(t, cargarFixture(t, "mediafire.html"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (MediaFireResolver{}).Resolve(ctx, httpx.NewClient(httpx.ClientOptions{}), s.remote(t, "Mediafire"))
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería que envolviera context.Canceled", err)
	}
}

func TestMediaFireStatusDeError(t *testing.T) {
	casos := []struct {
		nombre string
		status int
		queria error
	}{
		{"404", http.StatusNotFound, httpx.ErrNotFound},
		{"500", http.StatusInternalServerError, httpx.ErrServerError},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
			}))
			t.Cleanup(srv.Close)
			u, err := url.Parse(srv.URL + rutaPaginaMediaFire)
			if err != nil {
				t.Fatalf("parsear la URL: %v", err)
			}
			_, err = (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), Remote{Nombre: "Mediafire", URL: u})
			if !errors.Is(err, c.queria) {
				t.Errorf("err = %v, quería %v", err, c.queria)
			}
		})
	}
}

// TestMediaFireSigueRedireccion reproduce el salto real
// mediafire.com → www.mediafire.com (302): httpx lo sigue y el enlace
// sigue estando en la respuesta final.
func TestMediaFireSigueRedireccion(t *testing.T) {
	cuerpo := cargarFixture(t, "mediafire.html")
	s := &servidorMediaFire{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.paths = append(s.paths, r.URL.Path)
		s.mu.Unlock()
		if r.URL.Path != rutaPaginaMediaFire {
			http.Redirect(w, r, rutaPaginaMediaFire, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(s.srv.Close)

	// El remote es el dominio raíz (como mediafire.com) y la redirección
	// a la página con www sucede dentro de la petición.
	u, err := url.Parse(s.srv.URL + "/")
	if err != nil {
		t.Fatalf("parsear la URL: %v", err)
	}
	fuentes, err := (MediaFireResolver{}).Resolve(context.Background(), httpx.NewClient(httpx.ClientOptions{}), Remote{Nombre: "Mediafire", URL: u})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if fuentes[0].URL != enlaceFirmado {
		t.Errorf("URL = %q, quería %q", fuentes[0].URL, enlaceFirmado)
	}
	paths, _, _ := s.peticiones()
	queria := []string{"/", rutaPaginaMediaFire}
	if len(paths) != len(queria) {
		t.Fatalf("peticiones = %v, quería %v", paths, queria)
	}
	for i := range paths {
		if paths[i] != queria[i] {
			t.Errorf("peticiones[%d] = %q, quería %q", i, paths[i], queria[i])
		}
	}
}

// TestMediaFireDesdeLaCadena es el flujo completo de T14→T15: la
// entrada Mediafire de `var servers` se delega en MediaFireResolver con
// el doer compartido del extractor y su enlace firmado se agrega a las
// fuentes. El JSON de la entrada es artificial (el remote apunta al
// servidor local); slug, size y la página son datos reales.
func TestMediaFireDesdeLaCadena(t *testing.T) {
	s := nuevoServidorMediaFire(t, cargarFixture(t, "mediafire.html"))
	remoto := base64.StdEncoding.EncodeToString([]byte(s.srv.URL + rutaPaginaMediaFire))
	entrada := `{"remote":"` + remoto + `","slug":"upp07829rIJoW","server":"Mediafire","lang":1,"size":"150 MB","append":0}`

	ext := ServersExtractor{
		Base:      Base{Doer: httpx.NewClient(httpx.ClientOptions{})},
		Resolvers: []RemoteResolver{&MediaFireResolver{}},
	}
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: paginaConServers(entrada)}
	fuentes, err := ext.Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1 (la delegada a MediaFire)", len(fuentes))
	}
	if fuentes[0].URL != enlaceFirmado || fuentes[0].Type != models.ResourceMP4 || fuentes[0].Server != "Mediafire" {
		t.Errorf("fuentes[0] = %+v, quería %q como MP4 etiquetada Mediafire", fuentes[0], enlaceFirmado)
	}
}
