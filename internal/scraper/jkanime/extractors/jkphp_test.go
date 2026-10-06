package extractors

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// Rutas y recursos de los fixtures de testdata/ (sanitizados: el path
// stream/jkmedia/<hash>/<hash>/1/<ip> quedó como aaa…/bbb…/1/2001:db8::1,
// ver PROGRESO.md).
const (
	rutaJKPHP = "/jk.php?u=stream/jkmedia/prueba"
	urlJKPHP  = "https://jkanime.net/stream/jkmedia/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/1/2001:db8::1/"
	// urlJKPlayerJK es el mismo recurso servido por el otro dominio: la
	// respuesta de jkplayer/jk?u=… apunta a jkplayers.com.
	urlJKPlayerJK = "https://jkplayers.com/stream/jkmedia/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/1/2001:db8::1/"
)

func TestJKPHPCanHandle(t *testing.T) {
	casos := []struct {
		nombre string
		html   string
		queria bool
	}{
		{"iframe jk.php absoluto", `<iframe src="https://jkanime.net/jk.php?u=stream/jkmedia/x"></iframe>`, true},
		{"iframe jk.php relativo", paginaConIframes(rutaJKPHP), true},
		{"iframe jk.php protocolo-relativo", `<iframe src="//jkplayers.com/jk.php?u=stream/jkmedia/x"></iframe>`, true},
		{"referencia actual jkplayer/jk", paginaConIframes("/jkplayer/jk?u=stream/jkmedia/x"), true},
		{"jkplayer/um no es este player", paginaConIframes(rutaUM), false},
		{"lista de servidores", `var servers = [{ name: "fembed", url: "https://fembed.example.test/v/abc" }];`, false},
		{"html vacío", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := (JKPHPExtractor{}).CanHandle(Page{URL: "https://jkanime.net/x/1/", HTML: c.html})
			if got != c.queria {
				t.Errorf("CanHandle = %v, quería %v", got, c.queria)
			}
		})
	}
}

// TestJKPHPExtraeDelFixtureJKPHP sigue la forma clásica —iframe
// raíz-relativo a jk.php— y sirve la respuesta real del fixture. De paso
// comprueba que la petición conserva Referer, User-Agent y cookies
// (regla 3) con el doer compartido.
func TestJKPHPExtraeDelFixtureJKPHP(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jk.php": cargarFixture(t, "jkphp.html")})
	cli := httpx.NewClient(httpx.ClientOptions{})
	base, err := url.Parse(ps.srv.URL)
	if err != nil {
		t.Fatalf("parsear la URL del servidor: %v", err)
	}
	cli.Jar.SetCookies(base, []*http.Cookie{{Name: "jkanime-session", Value: "prueba-sesion"}})

	page := Page{
		URL:  ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(rutaJKPHP),
	}
	fuentes, err := (JKPHPExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != urlJKPHP {
		t.Errorf("URL = %q, quería %q", f.URL, urlJKPHP)
	}
	if f.Type != models.ResourceMP4 {
		t.Errorf("Type = %q, quería %q (el player declara type: 'mp4')", f.Type, models.ResourceMP4)
	}
	if f.Server != NameJKPHP {
		t.Errorf("Server = %q, quería %q", f.Server, NameJKPHP)
	}
	if queria := ps.srv.URL + rutaJKPHP; f.Referer != queria {
		t.Errorf("Referer de la fuente = %q, quería %q (la URL del iframe)", f.Referer, queria)
	}
	if len(f.Subtitles) != 0 {
		t.Errorf("Subtitles = %v, quería vacío: el fixture no trae .vtt", f.Subtitles)
	}

	referer, userAgent, cookie := ps.ultimaPeticion()
	if referer != page.URL {
		t.Errorf("Referer de la petición = %q, quería %q (la página del episodio)", referer, page.URL)
	}
	if userAgent != httpx.DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería el de navegador", userAgent)
	}
	if !strings.Contains(cookie, "jkanime-session=prueba-sesion") {
		t.Errorf("Cookie = %q, quería que viajara la sesión del doer compartido", cookie)
	}
	if pedidas := ps.rutasPedidas(); len(pedidas) != 1 || pedidas[0] != "/jk.php" {
		t.Errorf("rutas pedidas = %v, quería [/jk.php]", pedidas)
	}
}

// TestJKPHPExtraeDelFixtureJKPlayerJK usa la otra forma real —el iframe
// jkplayer/jk?u=… que hoy emiten las páginas— con su propio fixture.
func TestJKPHPExtraeDelFixtureJKPlayerJK(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/jk": cargarFixture(t, "jkplayer_jk.html")})
	cli := httpx.NewClient(httpx.ClientOptions{})

	iframe := ps.srv.URL + "/jkplayer/jk?u=stream/jkmedia/prueba"
	page := Page{
		URL:  ps.srv.URL + "/bleach/1/",
		HTML: `<iframe class="player_conte" src="` + iframe + `" width="565"></iframe>`,
	}
	fuentes, err := (JKPHPExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != urlJKPlayerJK {
		t.Errorf("URL = %q, quería %q", f.URL, urlJKPlayerJK)
	}
	if f.Type != models.ResourceMP4 {
		t.Errorf("Type = %q, quería %q", f.Type, models.ResourceMP4)
	}
	if f.Server != NameJKPHP {
		t.Errorf("Server = %q, quería %q", f.Server, NameJKPHP)
	}
	if f.Referer != iframe {
		t.Errorf("Referer = %q, quería la URL del iframe %q", f.Referer, iframe)
	}
	referer, _, _ := ps.ultimaPeticion()
	if referer != page.URL {
		t.Errorf("Referer de la petición = %q, quería %q", referer, page.URL)
	}
}

// TestJKPHPDedupMismoIframe prueba a la vez las dos formas de referencia
// (raíz-relativa y absoluta) del mismo iframe: una sola petición y una
// sola fuente.
func TestJKPHPDedupMismoIframe(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jk.php": cargarFixture(t, "jkphp.html")})
	cli := httpx.NewClient(httpx.ClientOptions{})

	page := Page{
		URL: ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(
			rutaJKPHP,
			ps.srv.URL+rutaJKPHP, // forma absoluta: también se sigue
		),
	}
	fuentes, err := (JKPHPExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1 (el mismo iframe por dos vías)", len(fuentes))
	}
	if pedidas := ps.rutasPedidas(); len(pedidas) != 1 || pedidas[0] != "/jk.php" {
		t.Errorf("rutas pedidas = %v, quería una sola petición a /jk.php", pedidas)
	}
}

// TestJKPHPSubtitulos usa un HTML artificial de player: los fixtures
// reales no incluyen .vtt (ver PROGRESO.md), pero el escaneo genérico que
// reutiliza de jkplayer.go debe rellenar Subtitles con URL absoluta e
// idioma inferido del nombre del archivo.
func TestJKPHPSubtitulos(t *testing.T) {
	const playerConSubs = `<!DOCTYPE html><html><body>
<video id="video" controls>
<source src='https://cdn.example.test/video.mp4' type='video/mp4'>
</video>
<track kind="subtitles" src="/subs/espanol.vtt" srclang="es" label="Español">
</body></html>`
	ps := nuevoServidorPlayer(t, map[string]string{"/jk.php": playerConSubs})
	cli := httpx.NewClient(httpx.ClientOptions{})

	page := Page{URL: ps.srv.URL + "/x/1/", HTML: paginaConIframes(rutaJKPHP)}
	fuentes, err := (JKPHPExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != "https://cdn.example.test/video.mp4" {
		t.Errorf("URL = %q, quería el <source> del player", f.URL)
	}
	queria := []models.Subtitle{{URL: ps.srv.URL + "/subs/espanol.vtt", Lang: "es"}}
	if len(f.Subtitles) != len(queria) || f.Subtitles[0] != queria[0] {
		t.Errorf("Subtitles = %v, quería %v", f.Subtitles, queria)
	}
}

func TestJKPHPErrorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	page := Page{URL: srv.URL + "/x/1/", HTML: paginaConIframes("/jk.php?u=prueba")}
	_, err := (JKPHPExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error con un 500 del servidor")
	}
	if !errors.Is(err, httpx.ErrServerError) {
		t.Errorf("err = %v, quería httpx.ErrServerError", err)
	}
}

func TestJKPHPSinDoer(t *testing.T) {
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: paginaConIframes(rutaJKPHP)}
	_, err := (JKPHPExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error sin doer inyectado")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer", err)
	}
}

func TestJKPHPContextoCancelado(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jk.php": cargarFixture(t, "jkphp.html")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	page := Page{URL: ps.srv.URL + "/x/1/", HTML: paginaConIframes(rutaJKPHP)}
	_, err := (JKPHPExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(ctx, page)
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería que envolviera context.Canceled", err)
	}
}

func TestJKPHPSinURLDeIframe(t *testing.T) {
	page := Page{URL: "https://jkanime.net/x/1/", HTML: `el markup habla de jk.php pero no incrusta ningún iframe`}
	_, err := (JKPHPExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error cuando no hay URL de iframe")
	}
	if !strings.Contains(err.Error(), "ninguna URL de iframe") {
		t.Errorf("err = %v, quería el mensaje de sin URL de iframe", err)
	}
}

func TestJKPHPPaginaSinVideo(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jk.php": `<html><body><p>sin vídeo</p></body></html>`})

	page := Page{URL: ps.srv.URL + "/x/1/", HTML: paginaConIframes(rutaJKPHP)}
	_, err := (JKPHPExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error cuando la respuesta no expone ninguna URL de vídeo")
	}
	if !strings.Contains(err.Error(), "ninguna URL de vídeo válida") {
		t.Errorf("err = %v, quería el mensaje de sin URL de vídeo", err)
	}
}

// TestJKPHPEnLaCadenaCanonica comprueba que el extractor queda registrado
// en la cadena del mismo modo que JKPlayer en T12: su Name() es el
// canónico, así que New lo ordena JKPlayer → JKPHP → Servers, y Resolve
// agrega las fuentes de los tres en ese orden sobre una página que
// menciona a los tres.
func TestJKPHPEnLaCadenaCanonica(t *testing.T) {
	if got := (JKPHPExtractor{}).Name(); got != NameJKPHP {
		t.Fatalf("Name() = %q, quería %q (prioridad canónica)", got, NameJKPHP)
	}

	ps := nuevoServidorPlayer(t, map[string]string{
		"/jkplayer/um": cargarFixture(t, "jkplayer_um.html"),
		"/jk.php":      cargarFixture(t, "jkphp.html"),
	})
	cli := httpx.NewClient(httpx.ClientOptions{})
	r := New(
		&extractorFalso{
			nombre: NameServers, marcador: "var servers",
			fuentes: fuentesDe("https://cdn.example.test/servers.m3u8"),
		},
		&JKPHPExtractor{Base: Base{Doer: cli}},
		&JKPlayerExtractor{Base: Base{Doer: cli}},
	)

	page := Page{
		URL: ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(rutaUM, rutaJKPHP) +
			`<script>var servers = [{ name: "fembed", url: "https://fembed.example.test/v/abc" }];</script>`,
	}
	fuentes, err := r.Resolve(context.Background(), page)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	queria := []string{urlUM, urlJKPHP, "https://cdn.example.test/servers.m3u8"}
	if len(fuentes) != len(queria) {
		t.Fatalf("fuentes = %d, quería %d", len(fuentes), len(queria))
	}
	servers := []string{NameJKPlayer, NameJKPHP, NameServers}
	for i, f := range fuentes {
		if f.URL != queria[i] {
			t.Errorf("fuentes[%d].URL = %q, quería %q", i, f.URL, queria[i])
		}
		if f.Server != servers[i] {
			t.Errorf("fuentes[%d].Server = %q, quería %q", i, f.Server, servers[i])
		}
	}
}
