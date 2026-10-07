package extractors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// Rutas y recursos de los fixtures de testdata/ (sanitizados: la firma
// temporal de la URL del vídeo se sustituyó por ST_UM/ST_UMV, ver
// PROGRESO.md).
const (
	rutaUM  = "/jkplayer/um?e=123&t=456&op=NzM1NjQ="
	rutaUMV = "/jkplayer/umv?e=789&t=abc&op=NzM1NjQ="
	urlUM   = "https://nika.playmudos.com/beach/TVB1LzB1Z2tlaWhvaWdsTnJBYXpmQT09.m3u8?st=ST_UM&e=1700000000"
	urlUMV  = "https://nika.playmudos.com/beach/TVB1LzB1Z2tlaWhvaWdsTnJBYXpmQT09.m3u8?st=ST_UMV&e=1700000000"
)

// cargarFixture lee un HTML real de testdata/.
func cargarFixture(t *testing.T, nombre string) string {
	t.Helper()
	dato, err := os.ReadFile(filepath.Join("testdata", nombre))
	if err != nil {
		t.Fatalf("fixture %s: %v", nombre, err)
	}
	return string(dato)
}

// servidorPlayer sirve los HTML indicados (ruta → cuerpo) y registra lo
// que recibe cada petición para poder verificar Referer, User-Agent y
// cookies (regla 3).
type servidorPlayer struct {
	srv        *httptest.Server
	mu         sync.Mutex
	referer    string
	userAgent  string
	cookie     string
	peticiones []string
}

func nuevoServidorPlayer(t *testing.T, rutas map[string]string) *servidorPlayer {
	t.Helper()
	ps := &servidorPlayer{}
	ps.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ps.mu.Lock()
		ps.referer = r.Referer()
		ps.userAgent = r.Header.Get("User-Agent")
		ps.cookie = r.Header.Get("Cookie")
		ps.peticiones = append(ps.peticiones, r.URL.Path)
		ps.mu.Unlock()
		cuerpo, ok := rutas[r.URL.Path]
		if !ok {
			http.Error(w, "no encontrado", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, cuerpo)
	}))
	t.Cleanup(ps.srv.Close)
	return ps
}

// ultimaPeticion devuelve las cabeceras de la última petición recibida.
func (ps *servidorPlayer) ultimaPeticion() (referer, userAgent, cookie string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.referer, ps.userAgent, ps.cookie
}

// rutasPedidas devuelve una copia de los paths recibidos.
func (ps *servidorPlayer) rutasPedidas() []string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return append([]string(nil), ps.peticiones...)
}

// paginaConIframes construye el HTML del episodio con un iframe por ruta,
// tal y como lo emite el JavaScript del sitio (src dentro de comillas
// simples; la ruta puede ser raíz-relativa o absoluta).
func paginaConIframes(rutas ...string) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><body><div id=\"video\">")
	for i, ruta := range rutas {
		fmt.Fprintf(&b, `<script>video[%d] = '<iframe class="player_conte" src="%s" width="565"></iframe>';</script>`, i, ruta)
	}
	b.WriteString("</div></body></html>")
	return b.String()
}

func TestJKPlayerCanHandle(t *testing.T) {
	casos := []struct {
		nombre string
		html   string
		queria bool
	}{
		{"iframe um relativo", paginaConIframes(rutaUM), true},
		{"iframe umv absoluto", `<iframe src="https://jkanime.net/jkplayer/umv?e=1&t=2&op=3"></iframe>`, true},
		{"mención de umv suelta", `src="/jkplayer/umv?e=1"`, true},
		{"jkplayer/jk no es este player", paginaConIframes("/jkplayer/jk?u=stream/jkmedia/x"), false},
		{"jk.php no es este player", `<iframe src="https://jkanime.net/jk.php?u=stream/jkmedia/x"></iframe>`, false},
		{"lista de servidores", `var servers = [{ name: "fembed", url: "https://fembed.example.test/v/abc" }];`, false},
		{"html vacío", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := (JKPlayerExtractor{}).CanHandle(Page{URL: "https://jkanime.net/x/1/", HTML: c.html})
			if got != c.queria {
				t.Errorf("CanHandle = %v, quería %v", got, c.queria)
			}
		})
	}
}

func TestJKPlayerExtraeHLSDelFixtureUM(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/um": cargarFixture(t, "jkplayer_um.html")})
	cli := httpx.NewClient(httpx.ClientOptions{})
	base, err := url.Parse(ps.srv.URL)
	if err != nil {
		t.Fatalf("parsear la URL del servidor: %v", err)
	}
	cli.Jar.SetCookies(base, []*http.Cookie{{Name: "jkanime-session", Value: "prueba-sesion"}})

	page := Page{
		URL:  ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(rutaUM),
	}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != urlUM {
		t.Errorf("URL = %q, quería %q", f.URL, urlUM)
	}
	if f.Type != models.ResourceHLS {
		t.Errorf("Type = %q, quería %q", f.Type, models.ResourceHLS)
	}
	if f.Server != NameJKPlayer {
		t.Errorf("Server = %q, quería %q", f.Server, NameJKPlayer)
	}
	if queria := ps.srv.URL + rutaUM; f.Referer != queria {
		t.Errorf("Referer de la fuente = %q, quería %q (la URL del iframe)", f.Referer, queria)
	}
	if len(f.Subtitles) != 0 {
		t.Errorf("Subtitles = %v, quería vacío: el sitio actual no publica .vtt", f.Subtitles)
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
}

func TestJKPlayerExtraeHLSDelFixtureUMV(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/umv": cargarFixture(t, "jkplayer_umv.html")})
	cli := httpx.NewClient(httpx.ClientOptions{})

	page := Page{
		URL:  ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(rutaUMV),
	}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	if fuentes[0].URL != urlUMV {
		t.Errorf("URL = %q, quería %q", fuentes[0].URL, urlUMV)
	}
	if fuentes[0].Type != models.ResourceHLS {
		t.Errorf("Type = %q, quería %q", fuentes[0].Type, models.ResourceHLS)
	}
	if fuentes[0].Referer != ps.srv.URL+rutaUMV {
		t.Errorf("Referer = %q, quería la URL del iframe umv", fuentes[0].Referer)
	}
}

// TestJKPlayerDedupPorRecurso prueba a la vez las dos formas de referencia
// del sitio (raíz-relativa y absoluta) y que um y umv —mismo vídeo con
// firmas distintas— no producen fuentes duplicadas.
func TestJKPlayerDedupPorRecurso(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{
		"/jkplayer/um":  cargarFixture(t, "jkplayer_um.html"),
		"/jkplayer/umv": cargarFixture(t, "jkplayer_umv.html"),
	})
	cli := httpx.NewClient(httpx.ClientOptions{})

	page := Page{
		URL: ps.srv.URL + "/one-piece/1/",
		HTML: paginaConIframes(
			rutaUM,
			ps.srv.URL+rutaUMV, // forma absoluta: también se sigue
		),
	}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1 (um y umv son el mismo vídeo)", len(fuentes))
	}
	if fuentes[0].URL != urlUM {
		t.Errorf("URL = %q, quería la primera (%q)", fuentes[0].URL, urlUM)
	}
	pedidas := ps.rutasPedidas()
	if len(pedidas) != 2 || pedidas[0] != "/jkplayer/um" || pedidas[1] != "/jkplayer/umv" {
		t.Errorf("rutas pedidas = %v, quería [/jkplayer/um /jkplayer/umv]", pedidas)
	}
}

// TestJKPlayerVariasCandidatasDelMismoVideoUnaFuente cubre el dedup por
// recurso DENTRO de una sola página de player (T29): la misma URL de vídeo
// aparece cifrada con atob, en dos hls.loadSource y en <source>, siempre
// con una firma temporal distinta, y debe salir una sola fuente (la
// primera en orden de prioridad); los recursos distintos de esa misma
// página sí salen, y la página solo se pide una vez.
func TestJKPlayerVariasCandidatasDelMismoVideoUnaFuente(t *testing.T) {
	const playerConCandidatas = `<!DOCTYPE html><html><body>
<script>
  var player = new DPlayer({ video: { url: atob('aHR0cHM6Ly9jZG4uZXhhbXBsZS50ZXN0L3YubTN1OD9zdD1BQUE=') } });
  hls.loadSource('https://cdn.example.test/v.m3u8?st=BBB');
  hls.loadSource('https://respaldo.example.test/v.m3u8?st=XXX');
</script>
<source src='https://cdn.example.test/v.m3u8?st=CCC' type='application/x-mpegURL'>
<video><source src='https://cdn.example.test/extra.mp4'></video>
</body></html>`
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/um": playerConCandidatas})
	cli := httpx.NewClient(httpx.ClientOptions{})

	page := Page{
		URL:  ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(rutaUM),
	}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	queria := []struct {
		url  string
		tipo models.ResourceType
	}{
		{"https://cdn.example.test/v.m3u8?st=AAA", models.ResourceHLS},
		{"https://respaldo.example.test/v.m3u8?st=XXX", models.ResourceHLS},
		{"https://cdn.example.test/extra.mp4", models.ResourceMP4},
	}
	if len(fuentes) != len(queria) {
		t.Fatalf("fuentes = %d, quería %d (las 3 firmas de v.m3u8 en 1, el respaldo y el mp4): %+v",
			len(fuentes), len(queria), fuentes)
	}
	for i, q := range queria {
		if fuentes[i].URL != q.url {
			t.Errorf("fuentes[%d].URL = %q, quería %q", i, fuentes[i].URL, q.url)
		}
		if fuentes[i].Type != q.tipo {
			t.Errorf("fuentes[%d].Type = %q, quería %q", i, fuentes[i].Type, q.tipo)
		}
		if fuentes[i].Server != NameJKPlayer {
			t.Errorf("fuentes[%d].Server = %q, quería %q", i, fuentes[i].Server, NameJKPlayer)
		}
		if fuentes[i].Referer == "" {
			t.Errorf("fuentes[%d].Referer vacío: cada fuente debe llevar la URL de su iframe", i)
		}
	}
	if pedidas := ps.rutasPedidas(); len(pedidas) != 1 || pedidas[0] != "/jkplayer/um" {
		t.Errorf("rutas pedidas = %v, quería [/jkplayer/um] (una sola petición por página)", pedidas)
	}
}

// TestJKPlayerSubtitulos usa un HTML artificial de player: los fixtures
// reales de hoy no incluyen ninguna pista .vtt (el sitio no publica
// softsubs; ver PROGRESO.md), pero el escaneo genérico debe rellenar
// Subtitles con URL absoluta e idioma inferido del nombre del archivo.
func TestJKPlayerSubtitulos(t *testing.T) {
	const playerConSubs = `<!DOCTYPE html><html><body>
<video id="video" controls>
<source src='https://cdn.example.test/video.m3u8?token=abc' type='application/x-mpegURL'>
</video>
<track kind="subtitles" src="/subs/espanol.vtt" srclang="es" label="Español">
<track kind="subtitles" src="/subs/english.vtt" srclang="en" label="English">
</body></html>`
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/um": playerConSubs})
	cli := httpx.NewClient(httpx.ClientOptions{})

	page := Page{
		URL:  ps.srv.URL + "/li-xiongmao/1/",
		HTML: paginaConIframes(rutaUM),
	}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: cli}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1 (las candidatas del mismo vídeo se deduplican)", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != "https://cdn.example.test/video.m3u8?token=abc" {
		t.Errorf("URL = %q, quería el .m3u8 del <source>", f.URL)
	}
	if f.Type != models.ResourceHLS {
		t.Errorf("Type = %q, quería %q", f.Type, models.ResourceHLS)
	}
	queria := []models.Subtitle{
		{URL: ps.srv.URL + "/subs/espanol.vtt", Lang: "es"},
		{URL: ps.srv.URL + "/subs/english.vtt", Lang: "en"},
	}
	if len(f.Subtitles) != len(queria) {
		t.Fatalf("Subtitles = %v, quería %v", f.Subtitles, queria)
	}
	for i, s := range f.Subtitles {
		if s != queria[i] {
			t.Errorf("Subtitles[%d] = %+v, quería %+v", i, s, queria[i])
		}
	}
}

func TestJKPlayerNormalizaAmp(t *testing.T) {
	const playerConAmp = `<source src='https://cdn.example.test/v.m3u8?a=1&amp;b=2' type='application/x-mpegURL'>`
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/um": playerConAmp})

	page := Page{URL: ps.srv.URL + "/x/1/", HTML: paginaConIframes(rutaUM)}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1", len(fuentes))
	}
	if queria := "https://cdn.example.test/v.m3u8?a=1&b=2"; fuentes[0].URL != queria {
		t.Errorf("URL = %q, quería %q (&amp; normalizado)", fuentes[0].URL, queria)
	}
}

func TestJKPlayerDescartaEsquemasNoHTTP(t *testing.T) {
	const playerBasura = `<script>window.dp = new DPlayer({ video: { url: 'javascript:alert(1)' } });</script>
<source src='data:text/plain,ola'>`
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/um": playerBasura})

	page := Page{URL: ps.srv.URL + "/x/1/", HTML: paginaConIframes(rutaUM)}
	fuentes, err := (JKPlayerExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error cuando ninguna candidata es http(s)")
	}
	if fuentes != nil {
		t.Errorf("fuentes = %v, quería nil", fuentes)
	}
}

func TestJKPlayerErrorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	page := Page{URL: srv.URL + "/x/1/", HTML: paginaConIframes(rutaUM)}
	_, err := (JKPlayerExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error con un 500 del servidor")
	}
	if !errors.Is(err, httpx.ErrServerError) {
		t.Errorf("err = %v, quería httpx.ErrServerError", err)
	}
}

func TestJKPlayerSinDoer(t *testing.T) {
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: paginaConIframes(rutaUM)}
	_, err := (JKPlayerExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error sin doer inyectado")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer", err)
	}
}

func TestJKPlayerContextoCancelado(t *testing.T) {
	ps := nuevoServidorPlayer(t, map[string]string{"/jkplayer/um": cargarFixture(t, "jkplayer_um.html")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	page := Page{URL: ps.srv.URL + "/x/1/", HTML: paginaConIframes(rutaUM)}
	_, err := (JKPlayerExtractor{Base: Base{Doer: httpx.NewClient(httpx.ClientOptions{})}}).Extract(ctx, page)
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería que envolviera context.Canceled", err)
	}
}

func TestJKPlayerSinURLDeIframe(t *testing.T) {
	page := Page{URL: "https://jkanime.net/x/1/", HTML: `el markup habla de jkplayer/um pero no incrusta ningún iframe`}
	_, err := (JKPlayerExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error cuando no hay URL de iframe")
	}
	if !strings.Contains(err.Error(), "ninguna URL de iframe") {
		t.Errorf("err = %v, quería el mensaje de sin URL de iframe", err)
	}
}

// TestURLsDeVideoPriorizaAtob comprueba con el fixture real que la
// candidata primera es la que decodifica atob() —la configuración activa
// de DPlayer— y no la copia comentada del HTML.
func TestURLsDeVideoPriorizaAtob(t *testing.T) {
	candidatas := urlsDeVideo(cargarFixture(t, "jkplayer_um.html"))
	if len(candidatas) == 0 {
		t.Fatal("quería al menos una candidata en el fixture um")
	}
	if candidatas[0] != urlUM {
		t.Errorf("candidatas[0] = %q, quería la descodificación de atob %q", candidatas[0], urlUM)
	}
}
