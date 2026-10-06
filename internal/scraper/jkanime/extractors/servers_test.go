package extractors

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// Remotes reales del fixture servers.html (bleach/1) ya decodificados:
// son los que el wrapper jkplayer/c1 de la página devuelve en su iframe
// (verificado con curl.exe en la captura de T14).
const (
	remoteMediafire = "https://mediafire.com/file/aaxudss27tqq7c8/"
	remoteMega      = "https://mega.nz/embed/jiIgHSaC#aqPRNZt2bEIZBTgvNx_BRZb-z9CLJUmG_gMaMFeHVmI"
	remoteVOE       = "https://voe.sx/e/mlnsng0cwwn7"
	remoteVidhide   = "https://vidhidevip.com/embed/"
	remoteFilemoon  = "https://bysekoze.com/e/gkazguzqot0z/"
	remoteMixdrop   = "https://mixdrop.top/e/el6qxlw7sqlwq0x"
	remoteMp4       = "https://www.mp4upload.com/embed-7uaibtwfgccu.html"
	remoteStreamtap = "https://streamtape.com/e/be9yBR9xL2HPjOq/01.mp4"
	remoteStreamwis = "https://sfastwish.com/e/evqmkewf1izz"
	remoteDood      = "https://dsvplay.com/e/hdxkobqafn74"
)

// entradaDelFixture recorta del fixture real la entrada JSON de un
// servidor concreto: así las páginas mínimas de los tests llevan datos
// reales en lugar de valores inventados.
func entradaDelFixture(t *testing.T, servidor string) string {
	t.Helper()
	re := regexp.MustCompile(`\{"remote":"[^"]*","slug":"[^"]*","server":"` +
		regexp.QuoteMeta(servidor) + `","lang":\d+,"size":"[^"]*","append":\d+\}`)
	entrada := re.FindString(cargarFixture(t, "servers.html"))
	if entrada == "" {
		t.Fatalf("el fixture servers.html no trae la entrada de %q", servidor)
	}
	return entrada
}

// paginaConServers construye un HTML mínimo con `var servers` y las
// entradas indicadas (del fixture real). El envoltorio es artificial: el
// sitio no publica páginas con una sola entrada (ver PROGRESO.md).
func paginaConServers(entradas ...string) string {
	return `<!DOCTYPE html><html><body><script>var servers = [` +
		strings.Join(entradas, ",") + `];</script></body></html>`
}

// falsoRemoteResolver es un RemoteResolver de prueba: reconoce los
// servidores de su lista y devuelve siempre las mismas fuentes (o el
// mismo error), registrando lo que recibe.
type falsoRemoteResolver struct {
	servidores []string
	fuentes    []models.VideoSource
	err        error

	llamadas int
	remoto   Remote
	doer     httpx.HTTPDoer
}

func (f *falsoRemoteResolver) CanResolve(server string) bool {
	for _, s := range f.servidores {
		if strings.EqualFold(s, server) {
			return true
		}
	}
	return false
}

func (f *falsoRemoteResolver) Resolve(_ context.Context, doer httpx.HTTPDoer, remote Remote) ([]models.VideoSource, error) {
	f.llamadas++
	f.remoto = remote
	f.doer = doer
	return f.fuentes, f.err
}

func TestServersCanHandle(t *testing.T) {
	casos := []struct {
		nombre string
		html   string
		queria bool
	}{
		{"fixture real", cargarFixture(t, "servers.html"), true},
		{"declaración suelta", `var servers = [];`, true},
		{"con sangría", "\t\tvar  servers=[{ }];", true},
		{"sin declaración", `<script>var video = [];</script>`, false},
		{"mención de la clase css", `<div class="bg-servers"></div>`, false},
		{"html vacío", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := (ServersExtractor{}).CanHandle(Page{URL: "https://jkanime.net/x/1/", HTML: c.html})
			if got != c.queria {
				t.Errorf("CanHandle = %v, quería %v", got, c.queria)
			}
		})
	}
}

// TestServersExtraeDelFixture recorre el JSON real del fixture: 10
// entradas, de las que 9 son iframes directos (Mediafire no lo es) y
// todas conservan el orden y la etiqueta del sitio.
func TestServersExtraeDelFixture(t *testing.T) {
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: cargarFixture(t, "servers.html")}
	fuentes, err := (ServersExtractor{}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	queria := []struct{ url, server string }{
		{remoteMega, "Mega"},
		{remoteVOE, "VOE"},
		{remoteVidhide, "Vidhide"},
		{remoteFilemoon, "Filemoon"},
		{remoteMixdrop, "Mixdrop"},
		{remoteMp4, "Mp4upload"},
		{remoteStreamtap, "Streamtape"},
		{remoteStreamwis, "Streamwish"},
		{remoteDood, "Doodstream"},
	}
	if len(fuentes) != len(queria) {
		t.Fatalf("fuentes = %d, quería %d (las 10 del fixture menos Mediafire, que sin resolver no produce iframe)", len(fuentes), len(queria))
	}
	for i, f := range fuentes {
		if f.URL != queria[i].url {
			t.Errorf("fuentes[%d].URL = %q, quería %q", i, f.URL, queria[i].url)
		}
		if f.Server != queria[i].server {
			t.Errorf("fuentes[%d].Server = %q, quería %q", i, f.Server, queria[i].server)
		}
		if f.Type != models.ResourceIframe {
			t.Errorf("fuentes[%d].Type = %q, quería %q", i, f.Type, models.ResourceIframe)
		}
		if f.Referer != "" || f.Quality != "" || f.Headers != nil || f.Subtitles != nil {
			t.Errorf("fuentes[%d] = %+v: Referer/Quality/Headers/Subtitles deben quedar vacíos (T16)", i, f)
		}
	}
}

// TestServersDelegaMediafire comprueba la delegación de T14→T15: la
// entrada de solo descarga llega al resolver con el remote ya
// decodificado y con el doer compartido del flujo (reglas 3 y 4), y sus
// fuentes se agregan a las de los demás servidores.
func TestServersDelegaMediafire(t *testing.T) {
	cli := httpx.NewClient(httpx.ClientOptions{})
	resuelta := []models.VideoSource{{
		URL:    "https://download.example.test/01.mp4",
		Type:   models.ResourceMP4,
		Server: "Mediafire",
	}}
	mf := &falsoRemoteResolver{servidores: []string{"Mediafire"}, fuentes: resuelta}
	ext := ServersExtractor{Base: Base{Doer: cli}, Resolvers: []RemoteResolver{mf}}

	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: cargarFixture(t, "servers.html")}
	fuentes, err := ext.Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 10 {
		t.Fatalf("fuentes = %d, quería 10 (9 iframes + la delegada)", len(fuentes))
	}
	// Mediafire es la primera entrada del JSON: la delegada conserva su
	// posición y Doodstream (la última) cierra la lista.
	if fuentes[0].URL != resuelta[0].URL || fuentes[9].URL != remoteDood {
		t.Errorf("primera %q, última %q; quería la delegada y luego %q", fuentes[0].URL, fuentes[9].URL, remoteDood)
	}
	if mf.llamadas != 1 {
		t.Fatalf("llamadas al resolver = %d, quería 1 (solo Mediafire)", mf.llamadas)
	}
	if mf.remoto.Nombre != "Mediafire" || mf.remoto.Slug != "upp07829rIJoW" {
		t.Errorf("remote = %+v, quería Nombre Mediafire y slug upp07829rIJoW", mf.remoto)
	}
	if mf.remoto.URL == nil || mf.remoto.URL.String() != remoteMediafire {
		t.Errorf("remote.URL = %v, quería %q ya decodificado", mf.remoto.URL, remoteMediafire)
	}
	if mf.doer != cli {
		t.Errorf("doer = %v, quería el doer compartido del extractor", mf.doer)
	}
}

func TestServersSinResolverParaElServidor(t *testing.T) {
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: paginaConServers(entradaDelFixture(t, "Mediafire"))}
	_, err := (ServersExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error: Mediafire sin resolver no produce ninguna fuente")
	}
	if !strings.Contains(err.Error(), "solo publica descarga") {
		t.Errorf("err = %v, quería el mensaje de servidor de solo descarga", err)
	}
}

func TestServersJSONInvalido(t *testing.T) {
	page := Page{URL: "https://jkanime.net/x/1/", HTML: cargarFixture(t, "servers_invalido.html")}
	_, err := (ServersExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error con la lista JSON cortada")
	}
	if !strings.Contains(err.Error(), "no es JSON válido") {
		t.Errorf("err = %v, quería el mensaje de JSON inválido", err)
	}
}

// TestServersFiltraEntradasInvalidas prueba el filtro del paso 2: de
// cinco entradas solo la real (VOE) sobrevive; si ninguna lo hace, el
// error explica cada descarte.
func TestServersFiltraEntradasInvalidas(t *testing.T) {
	javascript := base64.StdEncoding.EncodeToString([]byte("javascript:alert(1)"))
	valida := entradaDelFixture(t, "VOE")
	invalidas := []string{
		`{"remote":"aHR0cHM6Ly92b2Uuc3gvZS9tbG5zbmcwY3d3bjc=","slug":"sin-nombre","server":"","lang":1,"size":"1 MB","append":0}`,
		`{"remote":"%%no-es-base64%%","slug":"roto","server":"Roto","lang":1,"size":"1 MB","append":0}`,
		`{"remote":"` + javascript + `","slug":"js","server":"Javascript","lang":1,"size":"1 MB","append":0}`,
		`{"remote":"","slug":"vacio","server":"Vacio","lang":1,"size":"1 MB","append":0}`,
	}

	t.Run("con una entrada válida", func(t *testing.T) {
		page := Page{URL: "https://jkanime.net/x/1/", HTML: paginaConServers(append([]string{valida}, invalidas...)...)}
		fuentes, err := (ServersExtractor{}).Extract(context.Background(), page)
		if err != nil {
			t.Fatalf("Extract: %v", err)
		}
		if len(fuentes) != 1 || fuentes[0].URL != remoteVOE {
			t.Fatalf("fuentes = %+v, quería solo la de VOE", fuentes)
		}
	})

	t.Run("todas inválidas", func(t *testing.T) {
		page := Page{URL: "https://jkanime.net/x/1/", HTML: paginaConServers(invalidas...)}
		_, err := (ServersExtractor{}).Extract(context.Background(), page)
		if err == nil {
			t.Fatal("quería error cuando ninguna entrada es válida")
		}
		queria := []string{"sin nombre de servidor", "no es base64 válido", "no es una URL http(s)", "entrada sin remote"}
		for _, m := range queria {
			if !strings.Contains(err.Error(), m) {
				t.Errorf("err = %v, faltaba %q", err, m)
			}
		}
	})
}

func TestServersDedupEntradaRepetida(t *testing.T) {
	voe := entradaDelFixture(t, "VOE")
	page := Page{URL: "https://jkanime.net/x/1/", HTML: paginaConServers(voe, voe)}
	fuentes, err := (ServersExtractor{}).Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 1 {
		t.Errorf("fuentes = %d, quería 1 (la misma entrada repetida)", len(fuentes))
	}
}

func TestServersSinDeclaracion(t *testing.T) {
	page := Page{URL: "https://jkanime.net/x/1/", HTML: `<script>var video = [];</script>`}
	_, err := (ServersExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error sin la declaración de var servers")
	}
	if !strings.Contains(err.Error(), "no declara") {
		t.Errorf("err = %v, quería el mensaje de sin declaración", err)
	}
}

func TestServersListaVacia(t *testing.T) {
	page := Page{URL: "https://jkanime.net/x/1/", HTML: `var servers = [];`}
	_, err := (ServersExtractor{}).Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error con la lista vacía")
	}
	if !strings.Contains(err.Error(), "ninguna entrada") {
		t.Errorf("err = %v, quería el mensaje de sin fuentes", err)
	}
}

func TestServersContextoCancelado(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	page := Page{URL: "https://jkanime.net/x/1/", HTML: cargarFixture(t, "servers.html")}
	_, err := (ServersExtractor{}).Extract(ctx, page)
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería que envolviera context.Canceled", err)
	}
}

// resolverConRed hace una petición con el doer que le entregan: sirve
// para probar que ServersExtractor pasa el suyo (o nil) al resolver.
type resolverConRed struct {
	url string
}

func (resolverConRed) CanResolve(server string) bool {
	return strings.EqualFold(server, "mediafire")
}

func (r resolverConRed) Resolve(ctx context.Context, doer httpx.HTTPDoer, remote Remote) ([]models.VideoSource, error) {
	resp, err := httpx.Get(ctx, doer, r.url, httpx.ReqOptions{
		Provider:  "jkanime",
		Operation: "servers",
		Extractor: NameServers,
		Referer:   remote.URL.String(),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return []models.VideoSource{{URL: r.url, Type: models.ResourceMP4, Server: "Mediafire"}}, nil
}

func TestServersSinDoer(t *testing.T) {
	ext := ServersExtractor{Resolvers: []RemoteResolver{resolverConRed{url: "https://example.test/archivo.mp4"}}}
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: paginaConServers(entradaDelFixture(t, "Mediafire"))}
	_, err := ext.Extract(context.Background(), page)
	if err == nil {
		t.Fatal("quería error sin doer inyectado")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer (el doer viaja hasta el resolver)", err)
	}
}

// TestServersDelegaConRed encadena delegación + red real contra un
// httptest.Server local: el resolver pide con el doer compartido del
// flujo (User-Agent de navegador y cookies del jar, reglas 3 y 4) y su
// fuente se agrega a las del fixture.
func TestServersDelegaConRed(t *testing.T) {
	var mu sync.Mutex
	var userAgent, cookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		userAgent, cookie = r.Header.Get("User-Agent"), r.Header.Get("Cookie")
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<a id="downloadButton" href="https://download.example.test/01.mp4">Download</a>`))
	}))
	t.Cleanup(srv.Close)

	cli := httpx.NewClient(httpx.ClientOptions{})
	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parsear la URL del servidor: %v", err)
	}
	cli.Jar.SetCookies(base, []*http.Cookie{{Name: "mf-session", Value: "prueba-sesion"}})

	ext := ServersExtractor{
		Base:      Base{Doer: cli},
		Resolvers: []RemoteResolver{resolverConRed{url: srv.URL + "/file/aaxudss27tqq7c8/01.mp4"}},
	}
	page := Page{URL: "https://jkanime.net/bleach/1/", HTML: cargarFixture(t, "servers.html")}
	fuentes, err := ext.Extract(context.Background(), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(fuentes) != 10 {
		t.Fatalf("fuentes = %d, quería 10 (9 iframes + la delegada)", len(fuentes))
	}
	queria := srv.URL + "/file/aaxudss27tqq7c8/01.mp4"
	// Mediafire va la primera en `var servers`, así que su fuente abre
	// la lista y Doodstream (la última entrada) la cierra.
	if fuentes[0].URL != queria || fuentes[0].Type != models.ResourceMP4 {
		t.Errorf("fuentes[0] = %+v, quería %q como MP4", fuentes[0], queria)
	}
	if fuentes[9].URL != remoteDood {
		t.Errorf("fuentes[9].URL = %q, quería %q (la última entrada)", fuentes[9].URL, remoteDood)
	}
	mu.Lock()
	defer mu.Unlock()
	if userAgent != httpx.DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería el de navegador del doer compartido", userAgent)
	}
	if !strings.Contains(cookie, "mf-session=prueba-sesion") {
		t.Errorf("Cookie = %q, quería que viajara el jar del doer compartido", cookie)
	}
}

// TestServersEnLaCadenaCanonica repite el contrato de T12/T13: el Name()
// canónico coloca al extractor el último de la cadena y Resolve agrega
// las fuentes de los tres en orden JKPlayer → JKPHP → Servers.
func TestServersEnLaCadenaCanonica(t *testing.T) {
	if got := (ServersExtractor{}).Name(); got != NameServers {
		t.Fatalf("Name() = %q, quería %q (prioridad canónica)", got, NameServers)
	}

	ps := nuevoServidorPlayer(t, map[string]string{
		"/jkplayer/um": cargarFixture(t, "jkplayer_um.html"),
		"/jk.php":      cargarFixture(t, "jkphp.html"),
	})
	cli := httpx.NewClient(httpx.ClientOptions{})
	r := New(
		&ServersExtractor{},
		&JKPHPExtractor{Base: Base{Doer: cli}},
		&JKPlayerExtractor{Base: Base{Doer: cli}},
	)

	page := Page{
		URL: ps.srv.URL + "/bleach/1/",
		HTML: paginaConIframes(rutaUM, rutaJKPHP) +
			paginaConServers(entradaDelFixture(t, "VOE")),
	}
	fuentes, err := r.Resolve(context.Background(), page)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	queria := []string{urlUM, urlJKPHP, remoteVOE}
	servers := []string{NameJKPlayer, NameJKPHP, "VOE"}
	if len(fuentes) != len(queria) {
		t.Fatalf("fuentes = %d, quería %d", len(fuentes), len(queria))
	}
	for i, f := range fuentes {
		if f.URL != queria[i] {
			t.Errorf("fuentes[%d].URL = %q, quería %q", i, f.URL, queria[i])
		}
		if f.Server != servers[i] {
			t.Errorf("fuentes[%d].Server = %q, quería %q", i, f.Server, servers[i])
		}
	}
}
