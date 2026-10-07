package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"miapp/internal/cache"
	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/animeflv"
	"miapp/internal/scraper/jkanime"
	"miapp/internal/scraper/jkanime/extractors"
	"miapp/internal/stream"
)

// ---------------------------------------------------------------------------
// Fixtures reales que sirve el sitio falso (solo lectura).
//
// El HTML/JSON sale del testdata de jkanime y de extractors: no se inventa
// marcado. Las únicas transformaciones son las mismas que ya hace T17 en su
// test offline (reescribir https://jkanime.net a la URL del httptest y el
// remote base64 de Mediafire al servidor local) más las necesarias para que
// el manifiesto HLS y el segmento vivan en el propio servidor (el E2E los
// pide a través del proxy, cosa que T17 no hace). Anotado en PROGRESO.md.
// ---------------------------------------------------------------------------
const (
	rutaFixtureBusqueda  = "internal/scraper/jkanime/testdata/search.html"
	rutaFixtureDetalle   = "internal/scraper/jkanime/testdata/detail.html"
	rutaFixturePagina1   = "internal/scraper/jkanime/testdata/episodes_p1.json"
	rutaFixturePagina2   = "internal/scraper/jkanime/testdata/episodes_p2.json"
	rutaFixtureEpisodio  = "internal/scraper/jkanime/testdata/episode_page.html"
	rutaFixturePlayerUM  = "internal/scraper/jkanime/extractors/testdata/jkplayer_um.html"
	rutaFixturePlayerUMv = "internal/scraper/jkanime/extractors/testdata/jkplayer_umv.html"
	rutaFixtureMediaFire = "internal/scraper/jkanime/extractors/testdata/mediafire.html"
)

// Rutas del sitio falso.
const (
	rutaBusqueda    = "/buscar/"
	rutaExcepciones = "/excepciones.json"
	rutaDetalle     = "/bleach/"
	rutaEpisodio    = "/bleach/1/"
	rutaAJAX        = "/ajax/episodes/"
	rutaPlayerUM    = "/jkplayer/um"
	rutaPlayerUMv   = "/jkplayer/umv"
	rutaMediaFire   = "/file/aaxudss27tqq7c8/"
	rutaManifiesto  = "/hls/bleach_1.m3u8"
	rutaSegmento    = "/hls/segmento1.ts"
)

// Constantes del contrato del AJAX (T09): cookie de sesión y token CSRF, que
// el sitio exige con 419 si faltan. El token se extrae del propio fixture del
// detalle, así que el test no lo duplica.
const (
	cookieSesion = "jkanime_session"
	valorSesion  = "SESION_DE_PRUEBA"
)

// remoteMediafirePagina es el base64 del remote Mediafire tal y como viene en
// testdata/episode_page.html (lo reutiliza T17): sin reescribirlo el resolver
// pediría mediafire.com y el test se saldría de la red.
const remoteMediafirePagina = "aHR0cHM6Ly9tZWRpYWZpcmUuY29tL2ZpbGUvYWF4dWRzczI3dHFxN2M4Lwo="

// manifiestoPrueba es el HLS que sirve el sitio: dos líneas de encabezado y un
// segmento relativo, que el proxy (T19) debe resolver contra la URL final y
// reescribir a /proxy. HTML artificial, autorizado por T28 (regla 11).
const manifiestoPrueba = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:4
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:4.0,
segmento1.ts
#EXT-X-ENDLIST
`

var (
	reCSRF       = regexp.MustCompile(`name="csrf-token" content="([^"]+)"`)
	reIDNumerico = regexp.MustCompile(`data-anime="(\d+)"`)
	reLastPage   = regexp.MustCompile(`"last_page":\s*\d+`)
	reAtob       = regexp.MustCompile(`atob\('([A-Za-z0-9+/=]+)'\)`)
	reM3u8       = regexp.MustCompile(`https://[^\s"'<>]+\.m3u8[^\s"'<>]*`)
)

// fixturePrueba lee un fixture real del proyecto (solo lectura).
func fixturePrueba(t *testing.T, ruta string) string {
	t.Helper()
	dato, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("fixture %s: %v", ruta, err)
	}
	return string(dato)
}

// ---------------------------------------------------------------------------
// Transporte solo local: cualquier host distinto del sitio falso se corta
// ANTES de conectar y queda registrado, así el E2E es offline de verdad
// (regla 13) y un enlace mal reescrito hace fallar la prueba en lugar de
// salir a internet.
// ---------------------------------------------------------------------------

type transporteSoloLocal struct {
	servidor string // host:puerto del sitio falso
	base     http.RoundTripper

	mu    sync.Mutex
	fuera []string
}

func (t *transporteSoloLocal) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != t.servidor {
		t.mu.Lock()
		t.fuera = append(t.fuera, r.URL.String())
		t.mu.Unlock()
		return nil, fmt.Errorf("petición fuera del sitio falso: %s", r.URL)
	}
	return t.base.RoundTrip(r)
}

// intentosFuera devuelve las peticiones que el transporte cortó (ninguna debe
// aparecer en un flujo offline correcto).
func (t *transporteSoloLocal) intentosFuera() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.fuera...)
}

// ---------------------------------------------------------------------------
// Sitio falso: un jkanime completo en un httptest (búsqueda, detalle con
// cookie y CSRF, AJAX de episodios, página de episodio, players, Mediafire,
// manifiesto HLS y segmento). Guarda todas las peticiones para poder exigir
// exactamente qué se pidió.
// ---------------------------------------------------------------------------

type sitioFalso struct {
	srv *httptest.Server

	mu         sync.Mutex
	peticiones []string

	htmlBusqueda string
	htmlDetalle  string
	htmlEpisodio string
	playerUM     string
	playerUMv    string
	htmlMediaFir string
	paginas      map[string]string
	csrf         string
	idNumerico   string
	segmento     []byte
}

// nuevoSitioFalso monta el servidor y prepara el contenido (que necesita la
// URL propia). El contenido se escribe bajo el mutex que usa el handler, así
// que no hay carrera con las peticiones.
func nuevoSitioFalso(t *testing.T) *sitioFalso {
	t.Helper()
	s := &sitioFalso{}
	s.srv = httptest.NewServer(s)
	t.Cleanup(s.srv.Close)
	s.preparar(t)
	return s
}

func (s *sitioFalso) preparar(t *testing.T) {
	t.Helper()
	base := s.srv.URL

	detalle := strings.ReplaceAll(fixturePrueba(t, rutaFixtureDetalle), "https://jkanime.net", base)
	csrf := extraer(t, reCSRF, detalle, "csrf-token del detalle")
	idNumerico := extraer(t, reIDNumerico, detalle, "id numérico del detalle")

	// El fixture real trae 23 páginas de episodios (bleach); el E2E reduce el
	// sitio a 2 para que el stream termine en una sola pausa de 300 ms.
	paginas := map[string]string{
		"1": dosPaginas(fixturePrueba(t, rutaFixturePagina1)),
		"2": dosPaginas(fixturePrueba(t, rutaFixturePagina2)),
	}

	// Página del episodio: los iframes del player y el remote de Mediafire se
	// reescriben al servidor local (mismo criterio que T17).
	episodio := strings.ReplaceAll(fixturePrueba(t, rutaFixtureEpisodio), "https://jkanime.net", base)
	episodio = strings.Replace(episodio, remoteMediafirePagina,
		base64.StdEncoding.EncodeToString([]byte(base+rutaMediaFire)), 1)

	// Players: el .m3u8 debe vivir en el sitio falso, porque el E2E lo pide a
	// través del proxy (T17 no lo hacía y no hacía falta reescribirlo). En `um`
	// hay que tocar tanto el payload de atob como la copia literal del mismo
	// .m3u8 (url: y hls.loadSource): si no, las dos candidatas quedan con
	// recursos base distintos y el dedup devuelve 2 HLS en vez de 1.
	destino := base + rutaManifiesto
	playerUM := reescribirAtob(fixturePrueba(t, rutaFixturePlayerUM), destino+"?st=ST_UM&e=1700000000")
	playerUM = reM3u8.ReplaceAllString(playerUM, destino+"?st=ST_UM&e=1700000000")
	playerUMv := reM3u8.ReplaceAllString(fixturePrueba(t, rutaFixturePlayerUMv), destino+"?st=ST_UMV&e=1700000000")

	s.mu.Lock()
	defer s.mu.Unlock()
	s.htmlBusqueda = strings.ReplaceAll(fixturePrueba(t, rutaFixtureBusqueda), "https://jkanime.net", base)
	s.htmlDetalle = detalle
	s.htmlEpisodio = episodio
	s.playerUM = playerUM
	s.playerUMv = playerUMv
	s.htmlMediaFir = fixturePrueba(t, rutaFixtureMediaFire)
	s.paginas = paginas
	s.csrf = csrf
	s.idNumerico = idNumerico
	s.segmento = bytes.Repeat([]byte("SEGMENTO_DE_PRUEBA\n"), 256)
}

// dosPaginas deja el JSON real con last_page=2 (la página 1 la devuelve el
// stream y la 2 la pide la goroutine de fondo).
func dosPaginas(cuerpo string) string {
	return reLastPage.ReplaceAllString(cuerpo, `"last_page":2`)
}

// reescribirAtob sustituye el payload cifrado con atob por la URL del
// manifiesto del sitio falso (si el payload no es un .m3u8 se conserva).
func reescribirAtob(html, destino string) string {
	return reAtob.ReplaceAllStringFunc(html, func(muestra string) string {
		grupo := reAtob.FindStringSubmatch(muestra)
		dato, err := base64.StdEncoding.DecodeString(grupo[1])
		if err != nil || !strings.Contains(string(dato), ".m3u8") {
			return muestra
		}
		return "atob('" + base64.StdEncoding.EncodeToString([]byte(destino)) + "')"
	})
}

func extraer(t *testing.T, re *regexp.Regexp, html, que string) string {
	t.Helper()
	grupo := re.FindStringSubmatch(html)
	if len(grupo) != 2 || grupo[1] == "" {
		t.Fatalf("el fixture del detalle no trae %s", que)
	}
	return grupo[1]
}

func (s *sitioFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, tipo, cuerpo, cookie := s.responder(r)
	if cookie != "" {
		http.SetCookie(w, &http.Cookie{Name: cookieSesion, Value: cookie, Path: "/"})
	}
	w.Header().Set("Content-Type", tipo)
	w.WriteHeader(status)
	_, _ = w.Write(cuerpo)
}

// responder decide la respuesta con el mutex puesto: así el registro de
// peticiones y la lectura del contenido quedan sincronizados con preparar.
func (s *sitioFalso) responder(r *http.Request) (int, string, []byte, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peticiones = append(s.peticiones, r.Method+" "+r.URL.Path)

	ruta := r.URL.Path
	switch {
	case r.Method == http.MethodPost && strings.HasPrefix(ruta, rutaAJAX):
		return s.episodiosAJAX(r)
	case ruta == rutaExcepciones:
		return http.StatusOK, "application/json", []byte("{}"), ""
	case strings.HasPrefix(ruta, rutaBusqueda):
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.htmlBusqueda), ""
	case ruta == rutaDetalle:
		// El detalle abre sesión: sin esta cookie el AJAX responde 419.
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.htmlDetalle), valorSesion
	case ruta == rutaEpisodio:
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.htmlEpisodio), ""
	case ruta == rutaPlayerUM:
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.playerUM), ""
	case ruta == rutaPlayerUMv:
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.playerUMv), ""
	case ruta == rutaMediaFire:
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.htmlMediaFir), ""
	case ruta == rutaManifiesto:
		return http.StatusOK, "application/vnd.apple.mpegurl", []byte(manifiestoPrueba), ""
	case ruta == rutaSegmento:
		return http.StatusOK, "video/mp2t", append([]byte(nil), s.segmento...), ""
	case r.Method == http.MethodPost:
		return http.StatusMethodNotAllowed, "text/plain; charset=utf-8", []byte("method not allowed"), ""
	default:
		return http.StatusNotFound, "text/plain; charset=utf-8", []byte("no encontrado"), ""
	}
}

// episodiosAJAX reproduce el contrato de T09: POST, cookie de sesión y
// _token correctos; sin ellos responde 419 como el sitio real.
func (s *sitioFalso) episodiosAJAX(r *http.Request) (int, string, []byte, string) {
	sesion, err := r.Cookie(cookieSesion)
	if err != nil || sesion.Value != valorSesion {
		return http.StatusGone, "text/plain; charset=utf-8", []byte("session expired"), ""
	}
	if err := r.ParseForm(); err != nil {
		return http.StatusBadRequest, "text/plain; charset=utf-8", []byte("bad request"), ""
	}
	if r.FormValue("_token") != s.csrf {
		return http.StatusGone, "text/plain; charset=utf-8", []byte("bad token"), ""
	}
	parte := strings.Split(strings.Trim(r.URL.Path, "/"), "/") // ajax episodes <id> <n>
	if len(parte) != 4 || parte[2] != s.idNumerico {
		return http.StatusNotFound, "text/plain; charset=utf-8", []byte("no encontrado"), ""
	}
	cuerpo, ok := s.paginas[parte[3]]
	if !ok {
		return http.StatusNotFound, "text/plain; charset=utf-8", []byte("no encontrado"), ""
	}
	return http.StatusOK, "application/json", []byte(cuerpo), ""
}

func (s *sitioFalso) listaPeticiones() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.peticiones...)
}

func (s *sitioFalso) cuantas(prefijo string) int {
	n := 0
	for _, p := range s.listaPeticiones() {
		if strings.HasPrefix(p, prefijo) {
			n++
		}
	}
	return n
}

func (s *sitioFalso) idNum() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idNumerico
}

func (s *sitioFalso) contenidoSegmento() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.segmento...)
}

// hostDe devuelve host:puerto de una URL.
func hostDe(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}

// ---------------------------------------------------------------------------
// Ayudas de validación compartidas con el E2E de red (app_e2e_integration_test.go).
// ---------------------------------------------------------------------------

// validarURL exige una URL http(s) con host: es el eslabón que debe poder
// reproducirse (o descargar) en cada paso del flujo.
func validarURL(t *testing.T, contexto, raw string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Errorf("%s: %q no es una URL parseable: %v", contexto, raw, err)
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		t.Errorf("%s: el esquema de %q no es http/https", contexto, raw)
	}
	if u.Host == "" {
		t.Errorf("%s: %q no tiene host", contexto, raw)
	}
}

// validarTipo exige un tipo de recurso reconocido (T03: hls, mp4, iframe).
func validarTipo(t *testing.T, contexto string, tipo models.ResourceType) {
	t.Helper()
	switch tipo {
	case models.ResourceHLS, models.ResourceMP4, models.ResourceIframe:
	default:
		t.Errorf("%s: tipo de recurso no reconocido %q", contexto, tipo)
	}
}

// validarEpisodio comprueba que el id es un id de episodio parseable y del
// proveedor esperado (T05).
func validarEpisodio(t *testing.T, contexto string, ep models.Episode) {
	t.Helper()
	proveedor, slug, kind, numero, err := models.ParseEpisodeID(ep.ID)
	if err != nil {
		t.Errorf("%s: id de episodio inválido %q: %v", contexto, ep.ID, err)
		return
	}
	if proveedor != proveedorPrincipal || slug == "" || numero <= 0 {
		t.Errorf("%s: id de episodio inesperado %q", contexto, ep.ID)
	}
	if kind != models.KindEpisode && kind != models.KindSpecial && kind != models.KindMovie {
		t.Errorf("%s: kind no reconocido %q en %q", contexto, kind, ep.ID)
	}
	if ep.Kind != kind {
		t.Errorf("%s: el kind del episodio (%q) no coincide con su id (%q)", contexto, ep.Kind, ep.ID)
	}
}

// esperarCompleto bloquea hasta que llegue episodes:complete para ese anime
// (sin time.Sleep: espera por el canal del emisor, con un tope de seguridad).
func esperarCompleto(t *testing.T, emis *emisorFalso, animeID string, limite time.Duration) bool {
	t.Helper()
	caduca := time.After(limite)
	for {
		for _, ev := range emis.lista() {
			if ev.Nombre == "episodes:complete" && completoDe(t, ev).AnimeID == animeID {
				return true
			}
		}
		select {
		case <-emis.cambio:
		case <-caduca:
			return false
		}
	}
}

// ---------------------------------------------------------------------------
// E2E offline: recorre la aplicación entera contra el sitio falso.
// ---------------------------------------------------------------------------

// TestE2EFlujoCompletoOffline encadena SearchAnime → GetAnimeDetails (con
// streaming de episodios y eventos) → GetVideoSources → SelectBestSource →
// petición HTTP real a través del proxy local (T19), todo sobre un
// httptest.Server que hace de jkanime completo. Sin red: el transporte del
// proveedor corta cualquier host ajeno y el proxy solo se le pide el
// manifiesto y el segmento del propio sitio.
func TestE2EFlujoCompletoOffline(t *testing.T) {
	sitio := nuevoSitioFalso(t)

	transporte := &transporteSoloLocal{servidor: hostDe(sitio.srv.URL), base: http.DefaultTransport}
	doer := httpx.NewClient(httpx.ClientOptions{Transport: transporte})

	// El proxy se construye con la opción de red privada SOLO aquí: el
	// httptest escucha en 127.0.0.1 y el valor por defecto lo bloquearía
	// (T19). En producción la app usa NewProxy(nil) sin esa opción.
	px := stream.NewProxy(nil, stream.PermitirRedPrivada())

	reg := scraper.NewRegistry()
	proveedores := []scraper.Provider{
		jkanime.New(doer, sitio.srv.URL,
			&jkanime.ExceptionResolver{URL: sitio.srv.URL + rutaExcepciones, Doer: doer}),
		animeflv.New(doer, sitio.srv.URL),
	}
	for _, p := range proveedores {
		if err := reg.Register(p); err != nil {
			t.Fatalf("registrando el proveedor %q: %v", p.Name(), err)
		}
	}

	dirHistorial := t.TempDir()
	n := nuevoNucleo(reg, px, extractors.SelectBestSource, nil, cache.New(ttlCacheDetalle), dirHistorial)
	emis := nuevoEmisorFalso()
	if err := n.arrancar(emis.emitir); err != nil {
		t.Fatalf("arrancando el núcleo: %v", err)
	}
	t.Cleanup(func() { _ = n.cerrar() })
	a := &App{core: n}

	// --- 1. Búsqueda ------------------------------------------------------
	resultados, err := a.SearchAnime("bleach")
	if err != nil {
		t.Fatalf("SearchAnime: %v", err)
	}
	if len(resultados) == 0 {
		t.Fatal("SearchAnime no devolvió ningún resultado")
	}
	slug := ""
	for _, r := range resultados {
		if r.ID == "" || r.Title == "" {
			t.Errorf("resultado incompleto: %+v", r)
		}
		if r.Provider != proveedorPrincipal {
			t.Errorf("el resultado %q viene del proveedor %q; se esperaba %q", r.ID, r.Provider, proveedorPrincipal)
		}
		validarURL(t, "portada del resultado "+r.ID, r.CoverURL)
		if r.ID == "bleach" {
			slug = r.ID
		}
	}
	if slug == "" {
		ids := make([]string, 0, len(resultados))
		for _, r := range resultados {
			ids = append(ids, r.ID)
		}
		t.Fatalf("entre los %d resultados no está el slug bleach (hace falta el detalle del fixture): %v",
			len(resultados), ids)
	}
	t.Logf("búsqueda: %d resultados; se sigue %q", len(resultados), slug)

	// --- 2. Detalle con streaming y eventos -------------------------------
	primera, err := a.GetAnimeDetails(proveedorPrincipal, slug)
	if err != nil {
		t.Fatalf("GetAnimeDetails: %v", err)
	}
	if primera.Detail.ID != slug || primera.Detail.Title == "" {
		t.Errorf("detalle inesperado: %+v", primera.Detail)
	}
	validarURL(t, "portada del detalle", primera.Detail.Cover)
	if len(primera.Episodes) == 0 {
		t.Fatal("la primera tanda de episodios llegó vacía")
	}
	for _, ep := range primera.Episodes {
		validarEpisodio(t, "primera tanda", ep)
	}
	t.Logf("detalle %q (%s); %d episodios en la primera tanda",
		primera.Detail.Title, primera.Detail.Type, len(primera.Episodes))

	if !esperarCompleto(t, emis, slug, 30*time.Second) {
		t.Fatal("no llegó episodes:complete: el stream de episodios no cerró")
	}
	lotes := lotesDe(t, emis.lista(), slug)
	conEpisodios, final := 0, false
	for _, l := range lotes {
		if len(l.Episodes) > 0 {
			conEpisodios++
			for _, ep := range l.Episodes {
				validarEpisodio(t, "lote emitido", ep)
			}
		}
		if l.Done && l.Err == "" {
			final = true
		}
	}
	if conEpisodios == 0 {
		t.Error("no se emitió ningún lote con episodios")
	}
	if !final {
		t.Error("falta el lote final Done sin error")
	}
	if completados := completadosDe(t, emis.lista(), slug); len(completados) == 0 || completados[0].Error != "" {
		t.Errorf("episodes:complete = %+v; quería un error vacío", completados)
	}
	t.Logf("eventos: %d lotes (con episodios: %d)", len(lotes), conEpisodios)

	// --- 3. La caché evita un segundo fetch -------------------------------
	detallesAntes, ajaxAntes := sitio.cuantas("GET "+rutaDetalle), sitio.cuantas("POST "+rutaAJAX)
	segunda, err := a.GetAnimeDetails(proveedorPrincipal, slug)
	if err != nil {
		t.Fatalf("segunda GetAnimeDetails: %v", err)
	}
	if detalle, ajax := sitio.cuantas("GET "+rutaDetalle), sitio.cuantas("POST "+rutaAJAX); detalle != detallesAntes || ajax != ajaxAntes {
		t.Errorf("la segunda llamada volvió al proveedor: detalle %d→%d, AJAX %d→%d",
			detallesAntes, detalle, ajaxAntes, ajax)
	}
	if len(segunda.Episodes) <= len(primera.Episodes) {
		t.Errorf("la caché devolvió %d episodios; la primera llamada ya trajo %d",
			len(segunda.Episodes), len(primera.Episodes))
	}
	for _, ep := range segunda.Episodes {
		validarEpisodio(t, "segunda tanda", ep)
	}
	t.Logf("caché: sin nuevo fetch y con los %d episodios acumulados", len(segunda.Episodes))

	// --- 4. Fuentes de vídeo y elección de la mejor -----------------------
	epID := primera.Episodes[0].ID
	fuentes, err := a.GetVideoSources(proveedorPrincipal, epID, false)
	if err != nil {
		t.Fatalf("GetVideoSources: %v", err)
	}
	if len(fuentes) == 0 {
		t.Fatal("GetVideoSources no devolvió ninguna fuente")
	}
	baseProxy, err := px.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	for _, f := range fuentes {
		if f.Server == "" {
			t.Errorf("fuente sin etiqueta de servidor: %+v", f)
		}
		validarURL(t, "fuente "+f.Server, f.URL)
		validarTipo(t, "fuente "+f.Server, f.Type)
		u, parseErr := url.Parse(f.URL)
		enProxy := parseErr == nil && u.Path == "/proxy" && u.Host == hostDe(baseProxy)
		if f.Type == models.ResourceHLS || f.Type == models.ResourceMP4 {
			if !enProxy {
				t.Errorf("la fuente %s (%s) no pasó por el proxy: %q", f.Server, f.Type, f.URL)
			}
		} else if enProxy {
			t.Errorf("la fuente %s (%s) no debía pasar por el proxy: %q", f.Server, f.Type, f.URL)
		}
	}
	conteo := map[models.ResourceType]int{}
	for _, f := range fuentes {
		conteo[f.Type]++
	}
	if conteo[models.ResourceHLS] != 1 {
		t.Errorf("fuentes HLS = %d; se esperaba 1 (um y umv comparten el .m3u8 y deduplican)",
			conteo[models.ResourceHLS])
	}
	if conteo[models.ResourceMP4] != 1 {
		t.Errorf("fuentes MP4 = %d; se esperaba 1 (el remote de Mediafire)", conteo[models.ResourceMP4])
	}
	if fuentes[0].Type != models.ResourceHLS {
		t.Errorf("la primera fuente (la mejor) es %q (%s); se esperaba hls",
			fuentes[0].Type, fuentes[0].Server)
	}
	mejor, ok := extractors.SelectBestSource(fuentes)
	if !ok {
		t.Fatal("SelectBestSource no eligió ninguna fuente con la lista devuelta")
	}
	if mejor.URL != fuentes[0].URL {
		t.Errorf("SelectBestSource eligió %q pero GetVideoSources la trajo en la posición %d de %d",
			mejor.URL, indiceDe(fuentes, mejor)+1, len(fuentes))
	}
	destino := destinoProxy(t, mejor.URL)
	validarURL(t, "destino de la mejor fuente", destino)
	if !strings.HasPrefix(destino, sitio.srv.URL+rutaManifiesto) {
		t.Errorf("la mejor fuente apunta a %q; se esperaba el manifiesto del sitio falso", destino)
	}
	t.Logf("fuentes: %d; mejor = %s (%s) vía proxy\n%s",
		len(fuentes), mejor.Type, mejor.Server, resumenFuentes(fuentes))

	// --- 5. Petición HTTP real a través del proxy -------------------------
	cliente := &http.Client{Timeout: 30 * time.Second}
	resp, err := cliente.Get(mejor.URL)
	if err != nil {
		t.Fatalf("GET del manifiesto a través del proxy: %v", err)
	}
	cuerpo, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("leyendo el manifiesto: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("el proxy respondió %d para el manifiesto", resp.StatusCode)
	}
	if !strings.HasPrefix(string(cuerpo), "#EXTM3U") {
		t.Fatalf("el proxy no devolvió un manifiesto HLS: %.120q", cuerpo)
	}
	segmento := primeraURIDelManifiesto(string(cuerpo))
	if segmento == "" {
		t.Fatal("el manifiesto reescrito no trae ninguna línea de segmento")
	}
	validarURL(t, "segmento reescrito", segmento)
	if destinoProxy(t, segmento) != sitio.srv.URL+rutaSegmento {
		t.Errorf("el segmento reescrito apunta a %q; se esperaba %q",
			destinoProxy(t, segmento), sitio.srv.URL+rutaSegmento)
	}

	respSeg, err := cliente.Get(segmento)
	if err != nil {
		t.Fatalf("GET del segmento a través del proxy: %v", err)
	}
	cuerpoSeg, err := io.ReadAll(respSeg.Body)
	respSeg.Body.Close()
	if err != nil {
		t.Fatalf("leyendo el segmento: %v", err)
	}
	if respSeg.StatusCode != http.StatusOK && respSeg.StatusCode != http.StatusPartialContent {
		t.Errorf("el proxy respondió %d para el segmento", respSeg.StatusCode)
	}
	if !bytes.Equal(cuerpoSeg, sitio.contenidoSegmento()) {
		t.Errorf("el segmento devuelto tiene %d bytes y no coincide con el del sitio (%d)",
			len(cuerpoSeg), len(sitio.contenidoSegmento()))
	}
	t.Logf("proxy: manifiesto de %d bytes con el segmento reescrito y descargado (%d bytes, %s)",
		len(cuerpo), len(cuerpoSeg), respSeg.Header.Get("Content-Type"))

	// --- 6. Historial de visualización (T22) ------------------------------
	if err := a.SaveProgress(models.WatchProgress{
		AnimeID: slug, EpisodeID: epID, CurrentTime: 90, TotalDuration: 240,
	}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	guardado, err := a.GetProgress(slug)
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(guardado) != 1 {
		t.Fatalf("GetProgress devolvió %d registros; se esperaba 1", len(guardado))
	}
	if guardado[0].CurrentTime != 90 || guardado[0].TotalDuration != 240 || guardado[0].IsWatched {
		t.Errorf("round-trip del historial: %+v", guardado[0])
	}
	if err := a.SaveProgress(models.WatchProgress{
		AnimeID: slug, EpisodeID: epID, CurrentTime: 230, TotalDuration: 240,
	}); err != nil {
		t.Fatalf("SaveProgress (actualización): %v", err)
	}
	guardado, err = a.GetProgress(slug)
	if err != nil {
		t.Fatalf("GetProgress (2): %v", err)
	}
	if len(guardado) != 1 || !guardado[0].IsWatched {
		t.Errorf("el upsert no actualizó en el sitio: %+v", guardado)
	}
	todo, err := a.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress (historial entero): %v", err)
	}
	if len(todo) != 1 {
		t.Errorf("historial entero: %d registros; se esperaba 1", len(todo))
	}
	archivoHistorial := filepath.Join(dirHistorial, "historial.json")
	if info, err := os.Stat(archivoHistorial); err != nil {
		t.Errorf("el historial no se escribió en %s: %v", archivoHistorial, err)
	} else {
		t.Logf("historial: %s (%d bytes)", filepath.Base(archivoHistorial), info.Size())
	}

	// --- 7. Nada salió a la red -------------------------------------------
	if fuera := transporte.intentosFuera(); len(fuera) > 0 {
		t.Errorf("el flujo intentó salir del sitio falso %d vez/veces: %v", len(fuera), fuera)
	}

	// Registro completo para el diagnóstico y la comprobación de eslabones.
	peticiones := sitio.listaPeticiones()
	queria := []string{
		"GET " + rutaExcepciones,
		"GET " + rutaBusqueda + "bleach",
		"GET " + rutaDetalle,
		"POST " + rutaAJAX + sitio.idNum() + "/1",
		"POST " + rutaAJAX + sitio.idNum() + "/2",
		"GET " + rutaEpisodio,
		"GET " + rutaPlayerUM,
		"GET " + rutaPlayerUMv,
		"GET " + rutaMediaFire,
		"GET " + rutaManifiesto,
		"GET " + rutaSegmento,
	}
	for _, q := range queria {
		if !contiene(peticiones, q) {
			t.Errorf("faltó la petición %q al sitio falso; se hicieron: %v", q, peticiones)
		}
	}
	t.Logf("sitio falso: %d peticiones\n  %s", len(peticiones), strings.Join(peticiones, "\n  "))
}

func contiene(lista []string, buscado string) bool {
	for _, s := range lista {
		if s == buscado {
			return true
		}
	}
	return false
}

// resumenFuentes vota, por servidor y tipo, cuántas fuentes devolvió la
// cadena (diagnóstico del E2E).
func resumenFuentes(fuentes []models.VideoSource) string {
	agregado := map[string]int{}
	for _, f := range fuentes {
		clave := fmt.Sprintf("%s/%s", f.Server, f.Type)
		agregado[clave]++
	}
	claves := make([]string, 0, len(agregado))
	for k := range agregado {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	var b strings.Builder
	for _, k := range claves {
		fmt.Fprintf(&b, "  %s → %d\n", k, agregado[k])
	}
	return strings.TrimRight(b.String(), "\n")
}

// primeraURIDelManifiesto devuelve la primera línea que no es un tag (una
// URI de segmento o de sub-playlist).
func primeraURIDelManifiesto(manifiesto string) string {
	for _, linea := range strings.Split(manifiesto, "\n") {
		linea = strings.TrimSpace(linea)
		if linea == "" || strings.HasPrefix(linea, "#") {
			continue
		}
		return linea
	}
	return ""
}
