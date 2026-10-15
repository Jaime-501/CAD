package embeds

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"miapp/internal/logx"
	"miapp/internal/models"
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de httpx.Request. No
	// definir otro TestMain en este paquete.
	desecho := slog.New(slog.NewTextHandler(io.Discard, nil))
	logx.SetDefault(desecho)
	slog.SetDefault(desecho)
	os.Exit(m.Run())
}

// TestMP4UploadCumpleResolver comprueba en tiempo de ejecución lo que ya
// asegura el var _ Resolver del paquete: MP4Upload implementa la interfaz.
func TestMP4UploadCumpleResolver(t *testing.T) {
	var r Resolver = NewMP4Upload(nil)
	if r.Name() != nombreMP4Upload {
		t.Errorf("Name() = %q, quería %q", r.Name(), nombreMP4Upload)
	}
}

// TestEmbedsNoDependeDeProveedores es la guardia de neutralidad del paquete:
// embeds solo puede depender de models, httpx y logx. Se ejecuta el mismo
// comando que la compuerta (go list -deps) para no duplicar el criterio.
func TestEmbedsNoDependeDeProveedores(t *testing.T) {
	prohibidos := []string{
		"miapp/internal/scraper/jkanime",
		"miapp/internal/scraper/animeav1",
		"github.com/wailsapp/",
	}
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Skipf("no se pudo ejecutar go list -deps . (%v): %s", err, out)
	}
	for _, dep := range strings.Split(string(out), "\n") {
		dep = strings.TrimSpace(dep)
		for _, p := range prohibidos {
			if strings.Contains(dep, p) {
				t.Errorf("embeds no debe depender de %q (aparece %q)", p, dep)
			}
		}
	}
}

// servidorEmbed es un httptest.Server que sirve el mismo HTML en cualquier
// ruta, guarda la última petición y cuenta cuántas llegaron.
type servidorEmbed struct {
	*httptest.Server

	mu         sync.Mutex
	peticiones int
	ultima     *http.Request
}

func nuevoServidorEmbed(t *testing.T, status int, html string) *servidorEmbed {
	t.Helper()
	s := &servidorEmbed{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.peticiones++
		s.ultima = r.Clone(context.Background())
		s.mu.Unlock()
		if status != 0 {
			w.WriteHeader(status)
		}
		_, _ = io.WriteString(w, html)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *servidorEmbed) contar() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peticiones
}

func (s *servidorEmbed) ultimaPeticion() *http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ultima
}

// cargarFixture devuelve el HTML crudo del embed de MP4Upload capturado en
// docs/investigacion/adds/mp4upload_embed.html (token del path sanitizado
// como TOKEN_DE_PRUEBA).
func cargarFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/mp4upload_embed.html")
	if err != nil {
		t.Fatalf("leyendo el fixture: %v", err)
	}
	return string(b)
}

// urlDelMedio es la URL que el informe de adds señala como extraíble del
// fixture: el src del setup player.src({ type: "video/mp4", src: … }).
const urlDelMedio = "https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4"

// comprobarFuente valida la única fuente esperada del fixture. El Referer
// debe ser la URL del embed: el CDN responde 403 sin un Referer del dominio
// mp4upload.com (verificado en vivo, desviación del informe de adds).
func comprobarFuente(t *testing.T, fuentes []models.VideoSource, quiereReferer string) {
	t.Helper()
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1: %+v", len(fuentes), fuentes)
	}
	f := fuentes[0]
	if f.URL != urlDelMedio {
		t.Errorf("URL = %q, quería %q", f.URL, urlDelMedio)
	}
	if f.Type != models.ResourceMP4 {
		t.Errorf("Type = %q, quería %q", f.Type, models.ResourceMP4)
	}
	if f.Server != nombreMP4Upload {
		t.Errorf("Server = %q, quería %q", f.Server, nombreMP4Upload)
	}
	if f.Referer != quiereReferer {
		t.Errorf("Referer = %q, quería %q (sin él el CDN responde 403)", f.Referer, quiereReferer)
	}
	if len(f.Headers) != 0 {
		t.Errorf("Headers = %v, el CDN no exige ninguna: debían quedar vacíos", f.Headers)
	}
}
