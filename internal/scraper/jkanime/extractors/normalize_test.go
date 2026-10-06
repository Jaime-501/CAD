package extractors

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// base64 del remote Mediafire del fixture servers.html: el test lo
// sustituye por el del servidor local para no salir a la red (mismo
// truco que T15 en TestMediaFireDesdeLaCadena).
const remoteMediafireFixture = "aHR0cHM6Ly9tZWRpYWZpcmUuY29tL2ZpbGUvYWF4dWRzczI3dHFxN2M4Lwo="

func TestNormalizeDescartaInvalidas(t *testing.T) {
	casos := []struct {
		nombre string
		url    string
	}{
		{"vacía", ""},
		{"solo espacios", "   "},
		{"javascript", "javascript:alert(1)"},
		{"data", "data:text/plain,ola"},
		{"ftp", "ftp://cdn.example.test/video.mp4"},
		{"sin esquema", "cdn.example.test/video.mp4"},
		{"relativa", "/video.m3u8"},
		{"protocolo relativo", "//cdn.example.test/video.m3u8"},
		{"sin host", "http:///video.mp4"},
		{"sin host con puerto", "http://:8080/video.mp4"},
		{"no parseable", "ht tp://cdn.example.test/"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := Normalize([]models.VideoSource{{URL: c.url, Type: models.ResourceHLS}})
			if len(got) != 0 {
				t.Errorf("Normalize(%q) = %+v, quería que la descartara", c.url, got)
			}
		})
	}
}

func TestNormalizeCompletaTipo(t *testing.T) {
	casos := []struct {
		nombre string
		url    string
		actual models.ResourceType
		queria models.ResourceType
	}{
		{"m3u8 vacío", "https://cdn.example.test/v.m3u8?st=1", "", models.ResourceHLS},
		{"m3u vacío", "https://cdn.example.test/v.m3u", "", models.ResourceHLS},
		{"mp4 vacío", "https://cdn.example.test/v.mp4", "", models.ResourceMP4},
		{"mkv vacío", "https://cdn.example.test/v.mkv", "", models.ResourceMP4},
		{"webm vacío", "https://cdn.example.test/v.webm", "", models.ResourceMP4},
		{"extensión en mayúsculas", "https://cdn.example.test/V.MP4", "", models.ResourceMP4},
		{"sin extensión se queda vacío", "https://cdn.example.test/sin-extension", "", ""},
		{"tipo existente no se pisa", "https://cdn.example.test/sin-extension", models.ResourceMP4, models.ResourceMP4},
		{"iframe con .mp4 en la ruta sigue siendo iframe",
			"https://streamtape.com/e/be9yBR9xL2HPjOq/01.mp4", models.ResourceIframe, models.ResourceIframe},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := Normalize([]models.VideoSource{{URL: c.url, Type: c.actual}})
			if len(got) != 1 {
				t.Fatalf("fuentes = %d, quería 1", len(got))
			}
			if got[0].Type != c.queria {
				t.Errorf("Type = %q, quería %q", got[0].Type, c.queria)
			}
		})
	}
}

func TestNormalizeDeduplicaPorRecurso(t *testing.T) {
	t.Run("misma URL con firmas distintas", func(t *testing.T) {
		got := Normalize([]models.VideoSource{
			{URL: "https://cdn.example.test/v.m3u8?st=AAA&e=1", Type: models.ResourceHLS},
			{URL: "https://cdn.example.test/v.m3u8?st=BBB&e=2", Type: models.ResourceHLS},
		})
		if len(got) != 1 {
			t.Fatalf("fuentes = %d, quería 1 (la query solo cambia la firma)", len(got))
		}
		if queria := "https://cdn.example.test/v.m3u8?st=AAA&e=1"; got[0].URL != queria {
			t.Errorf("URL = %q, quería la primera %q", got[0].URL, queria)
		}
	})

	t.Run("esquema y host sin distinguir mayúsculas", func(t *testing.T) {
		got := Normalize([]models.VideoSource{
			{URL: "HTTPS://CDN.Example.test/v.mp4", Type: models.ResourceMP4},
			{URL: "https://cdn.example.test/v.mp4?x=1", Type: models.ResourceMP4},
		})
		if len(got) != 1 {
			t.Fatalf("fuentes = %d, quería 1", len(got))
		}
		if queria := "https://cdn.example.test/v.mp4"; got[0].URL != queria {
			t.Errorf("URL = %q, quería %q (host y esquema en minúsculas)", got[0].URL, queria)
		}
	})

	t.Run("recursos distintos no se mezclan", func(t *testing.T) {
		got := Normalize([]models.VideoSource{
			{URL: "https://cdn.example.test/uno.m3u8", Type: models.ResourceHLS},
			{URL: "https://cdn.example.test/dos.m3u8", Type: models.ResourceHLS},
			{URL: "https://otro.example.test/uno.m3u8", Type: models.ResourceHLS},
		})
		if len(got) != 3 {
			t.Errorf("fuentes = %d, quería 3 (paths y hosts distintos)", len(got))
		}
	})

	t.Run("el fragmento distingue embeds", func(t *testing.T) {
		got := Normalize([]models.VideoSource{
			{URL: "https://mega.nz/embed/abc#clave1", Type: models.ResourceIframe},
			{URL: "https://mega.nz/embed/abc#clave2", Type: models.ResourceIframe},
		})
		if len(got) != 2 {
			t.Errorf("fuentes = %d, quería 2 (el fragmento es la clave de descifrado)", len(got))
		}
	})

	t.Run("conserva el fragmento en la salida", func(t *testing.T) {
		const mega = "https://mega.nz/embed/jiIgHSaC#aqPRNZt2bEIZBTgvNx_BRZb-z9CLJUmG_gMaMFeHVmI"
		got := Normalize([]models.VideoSource{{URL: mega, Type: models.ResourceIframe}})
		if len(got) != 1 || got[0].URL != mega {
			t.Errorf("fuentes = %+v, quería la URL íntegra con fragmento", got)
		}
	})
}

func TestNormalizePreservaCampos(t *testing.T) {
	completa := models.VideoSource{
		URL:       "https://cdn.example.test/v.m3u8?st=1",
		Type:      models.ResourceHLS,
		Quality:   "1080p",
		Server:    "jkplayer",
		Referer:   "https://jkanime.net/jkplayer/um?e=1",
		Headers:   map[string]string{"Origin": "https://cdn.example.test"},
		Subtitles: []models.Subtitle{{URL: "https://cdn.example.test/es.vtt", Lang: "es"}},
	}
	got := Normalize([]models.VideoSource{completa})
	if len(got) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(got))
	}
	g := got[0]
	if g.Quality != "1080p" || g.Server != "jkplayer" || g.Referer != "https://jkanime.net/jkplayer/um?e=1" {
		t.Errorf("campos = %+v, quería Quality/Server/Referer intactos", g)
	}
	if g.Headers["Origin"] != "https://cdn.example.test" {
		t.Errorf("Headers = %v, quería conservados", g.Headers)
	}
	if len(g.Subtitles) != 1 || g.Subtitles[0].Lang != "es" {
		t.Errorf("Subtitles = %v, quería conservados", g.Subtitles)
	}

	t.Run("el duplicado completa lo que falta sin pisar nada", func(t *testing.T) {
		vacia := models.VideoSource{
			URL:       "https://cdn.example.test/v.m3u8?st=2",
			Type:      models.ResourceHLS,
			Referer:   "https://jkanime.net/jkplayer/umv?e=2",
			Subtitles: []models.Subtitle{{URL: "https://cdn.example.test/en.vtt", Lang: "en"}},
		}
		got := Normalize([]models.VideoSource{completa, vacia})
		if len(got) != 1 {
			t.Fatalf("fuentes = %d, quería 1", len(got))
		}
		g := got[0]
		if g.Referer != "https://jkanime.net/jkplayer/um?e=1" {
			t.Errorf("Referer = %q, quería el de la primera (no se pisa)", g.Referer)
		}
		if len(g.Subtitles) != 1 || g.Subtitles[0].Lang != "es" {
			t.Errorf("Subtitles = %v, quería los de la primera", g.Subtitles)
		}
		if g.Server != "jkplayer" || g.Quality != "1080p" {
			t.Errorf("campos = %+v, quería los de la primera intactos", g)
		}
	})
}

func TestNormalizeSinEntradas(t *testing.T) {
	if got := Normalize(nil); got != nil {
		t.Errorf("Normalize(nil) = %v, quería nil", got)
	}
	if got := Normalize([]models.VideoSource{}); got != nil {
		t.Errorf("Normalize(vacío) = %v, quería nil", got)
	}
}

// TestNormalizeSeAplicaEnResolve comprueba la integración al final de
// Resolver.Resolve (T16): las fuentes de los extractores llegan ya
// validadas, tipificadas y deduplicadas.
func TestNormalizeSeAplicaEnResolve(t *testing.T) {
	conFirmaA := "https://cdn.example.test/v.m3u8?st=AAA"
	conFirmaB := "https://cdn.example.test/v.m3u8?st=BBB"
	bruto := &extractorFalso{
		nombre: NameJKPlayer, marcador: "data-src",
		fuentes: []models.VideoSource{
			{URL: "s1", Type: models.ResourceHLS}, // sin esquema: fuera
			{URL: conFirmaA, Type: models.ResourceHLS},
			{URL: conFirmaB, Type: models.ResourceHLS},
		},
	}
	otro := &extractorFalso{
		nombre: NameJKPHP, marcador: "JKPHP",
		fuentes: []models.VideoSource{{URL: "https://cdn.example.test/otro.mp4"}},
	}
	r := New(bruto, otro)

	fuentes, err := r.Resolve(context.Background(), Page{URL: "https://jkanime.net/x/1/", HTML: htmlEpisodio})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(fuentes) != 2 {
		t.Fatalf("fuentes = %d, quería 2 (una inválida fuera y dos firmas del mismo vídeo en una)", len(fuentes))
	}
	if fuentes[0].URL != conFirmaA {
		t.Errorf("fuentes[0].URL = %q, quería %q (gana la primera)", fuentes[0].URL, conFirmaA)
	}
	if fuentes[1].Type != models.ResourceMP4 {
		t.Errorf("fuentes[1].Type = %q, quería %q (tipo completado por extensión)", fuentes[1].Type, models.ResourceMP4)
	}
	if fuentes[1].Server != NameJKPHP {
		t.Errorf("fuentes[1].Server = %q, quería %q", fuentes[1].Server, NameJKPHP)
	}
}

// TestNormalizeConFixtures recorre la cadena real sobre los fixtures de
// testdata/ (players um/umv, `var servers` con sus 10 servidores y la
// página de Mediafire) y comprueba que la salida de Resolve sale
// normalizada: los tres tipos conviven, sin duplicados por recurso y
// todas las URL reproducibles.
func TestNormalizeConFixtures(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{
		"/jkplayer/um":  cargarFixture(t, "jkplayer_um.html"),
		"/jkplayer/umv": cargarFixture(t, "jkplayer_umv.html"),
	})
	s := nuevoServidorMediaFire(t, cargarFixture(t, "mediafire.html"))
	// El fixture trae los iframes del player apuntando a jkanime.net y el
	// remote de Mediafire al sitio real: se reescriben al servidor local
	// para que la cadena corra sin red.
	html := strings.Replace(cargarFixture(t, "servers.html"),
		"https://jkanime.net", ps.srv.URL, -1)
	html = strings.Replace(html,
		remoteMediafireFixture,
		base64.StdEncoding.EncodeToString([]byte(s.srv.URL+rutaPaginaMediaFire)), 1)
	cli := httpx.NewClient(httpx.ClientOptions{})
	r := New(
		&JKPlayerExtractor{Base: Base{Doer: cli}},
		&ServersExtractor{Base: Base{Doer: cli}, Resolvers: []RemoteResolver{&MediaFireResolver{}}},
	)

	fuentes, err := r.Resolve(context.Background(), Page{URL: ps.srv.URL + "/bleach/1/", HTML: html})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	porTipo := map[models.ResourceType]int{}
	vistas := map[string]bool{}
	for _, f := range fuentes {
		u, err := url.Parse(f.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
			t.Errorf("fuente sin validar en la salida: %q", f.URL)
		}
		if vistas[claveDe(u)] {
			t.Errorf("fuente duplicada por recurso: %q", f.URL)
		}
		vistas[claveDe(u)] = true
		porTipo[f.Type]++
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
	if fuentes[0].Server != NameJKPlayer {
		t.Errorf("fuentes[0].Server = %q, quería %q (el orden canónico se conserva)", fuentes[0].Server, NameJKPlayer)
	}
}
