package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
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
	"miapp/internal/logx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/animeav1"
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

// recortarURL reduce una URL a esquema+host+ruta corta (y "?…" si lleva
// query): los E2E no deben imprimir URLs firmadas ni ids completos ni
// siquiera en mensajes de fallo o de diagnóstico.
func recortarURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<ilegible>"
	}
	ruta := u.Path
	if len(ruta) > 10 {
		ruta = ruta[:10] + "…"
	}
	salida := u.Scheme + "://" + u.Host + ruta
	if u.RawQuery != "" {
		salida += "?…"
	}
	return salida
}

// recortarError sustituye la URL cruda dentro del texto de un error de red
// por su versión recortada.
func recortarError(err error, crudo string) string {
	if err == nil || crudo == "" {
		return ""
	}
	return strings.ReplaceAll(err.Error(), crudo, recortarURL(crudo))
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
// proveedor principal (T05).
func validarEpisodio(t *testing.T, contexto string, ep models.Episode) {
	t.Helper()
	validarEpisodioDe(t, contexto, ep, proveedorPrincipal)
}

// validarEpisodioDe es validarEpisodio con el proveedor explicitado: el E2E
// de AnimeAV1 valida episodios de animeav1 con el mismo criterio.
func validarEpisodioDe(t *testing.T, contexto string, ep models.Episode, proveedor string) {
	t.Helper()
	prov, slug, kind, numero, err := models.ParseEpisodeID(ep.ID)
	if err != nil {
		t.Errorf("%s: id de episodio inválido %q: %v", contexto, ep.ID, err)
		return
	}
	if prov != proveedor || slug == "" || numero <= 0 {
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
// Captura de logs de un flujo completo (T29): nada de lo que el flujo
// recorrido pida o reciba debe salir en claro en logx ni en slog.
// ---------------------------------------------------------------------------

// registroSeguro es un io.Writer con mutex: los logs llegan de varios
// goroutines (stream, proxy) y -race está activo.
type registroSeguro struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (r *registroSeguro) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.b.Write(p)
}

func (r *registroSeguro) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.b.String()
}

// reFirmaEnLog busca en el texto capturado una query que firma un recurso
// con un valor de 4+ caracteres: con la redacción activa solo quedan ***
// (3), así que cualquier coincidencia es una URL firmada filtrada.
var reFirmaEnLog = regexp.MustCompile(`(?i)[?&](?:e|t|st|op|key|sig|signature|sign|expires|expiry|policy|key-pair-id|awsaccesskeyid|x-amz-signature|token|csrf(?:[-_]?token)?|_token)=([^&\s"'<>]{4,})`)

// capturarLogs redirige slog (que TestMain descarta) y logx (JSON a stderr
// por defecto) a un buffer mientras dura el test, y los restaura al salir.
func capturarLogs(t *testing.T) *registroSeguro {
	t.Helper()
	reg := &registroSeguro{}
	l := slog.New(logx.NewHandler(slog.NewTextHandler(reg, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.SetDefault(l)
	logx.SetDefault(l)
	t.Cleanup(func() {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // como TestMain
		logx.New(false)                                                 // JSON a stderr, como en producción
	})
	return reg
}

// comprobarLogsSinSecretos falla si el flujo dejó cookies, CSRF, tokens o
// URLs firmadas completas en la salida capturada.
func comprobarLogsSinSecretos(t *testing.T, reg *registroSeguro) {
	t.Helper()
	salida := reg.String()
	if salida == "" {
		t.Error("los logs del flujo están vacíos: la captura no funciona y esta comprobación no probaría nada")
		return
	}
	t.Logf("logs del flujo capturados: %d bytes", len(salida))
	if !strings.Contains(salida, "http_request") {
		t.Error("la captura no contiene ninguna petición http registrada")
	}
	if coincidencia := reFirmaEnLog.FindString(salida); coincidencia != "" {
		t.Errorf("los logs contienen una URL firmada/token sin redactar: %q", coincidencia)
	}
	// Frases concretas que jamás deben aparecer en claro.
	bloqueadas := []string{
		"SESION_DE_PRUEBA", // cookie de sesión del sitio (offline)
		"jkanime_session=",
		"Set-Cookie",
		"_token=",
		"csrf-token",
		"Authorization:",
	}
	lower := strings.ToLower(salida)
	for _, secreto := range bloqueadas {
		if strings.Contains(lower, strings.ToLower(secreto)) {
			t.Errorf("el registro %q aparece en los logs:\n%s", secreto, salida)
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
	registro := capturarLogs(t)
	sitio := nuevoSitioFalso(t)

	transporte := &transporteSoloLocal{servidor: hostDe(sitio.srv.URL), base: http.DefaultTransport}
	doer := httpx.NewClient(httpx.ClientOptions{Transport: transporte})

	// El proxy se construye con la opción de red privada SOLO aquí: el
	// httptest escucha en 127.0.0.1 y el valor por defecto lo bloquearía
	// (T19). En producción la app usa NewProxy(nil) sin esa opción.
	px := stream.NewProxy(nil, stream.PermitirRedPrivada())

	reg := scraper.NewRegistry()
	// Solo JKAnime: este E2E recorre el flujo completo del proveedor
	// principal contra su sitio falso. El E2E de AnimeAV1
	// (TestE2EAnimeAV1FlujoCompletoOffline) monta el suyo con los dos
	// proveedores para cubrir además la búsqueda combinada (T34).
	proveedores := []scraper.Provider{
		jkanime.New(doer, sitio.srv.URL,
			&jkanime.ExceptionResolver{URL: sitio.srv.URL + rutaExcepciones, Doer: doer}),
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
		// T29: la fuente proxificada es la que cruza el puente de Wails, así
		// que sale sin cabeceras: el Referer/Headers vive solo en la sesión
		// del proxy, que es quien los inyecta al origen.
		if enProxy && (f.Referer != "" || f.Headers != nil) {
			t.Errorf("la fuente %s va hacia el frontend con cabeceras: %+v", f.Server, f)
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
	// T29: además de "todas presentes", la lista debe ser EXACTA: una
	// petición repetida (p. ej. una segunda búsqueda o una segunda descarga
	// del mapa de excepciones en la misma consulta) es un fallo, no un detalle.
	esperadas := append([]string(nil), queria...)
	hubieron := append([]string(nil), peticiones...)
	sort.Strings(esperadas)
	sort.Strings(hubieron)
	if strings.Join(esperadas, "\n") != strings.Join(hubieron, "\n") {
		t.Errorf("las peticiones no son exactas (quería %d, se hicieron %d)\n  quería:\n  %s\n  hizo:\n  %s",
			len(esperadas), len(hubieron),
			strings.Join(esperadas, "\n  "), strings.Join(hubieron, "\n  "))
	}
	if n := contar(peticiones, "GET "+rutaBusqueda+"bleach"); n != 1 {
		t.Errorf("la búsqueda a JKAnime se pidió %d veces; debe ser exactamente 1", n)
	}
	if n := contar(peticiones, "GET "+rutaExcepciones); n != 1 {
		t.Errorf("el mapa de excepciones se descargó %d veces; debe ser 1 por proceso", n)
	}
	t.Logf("sitio falso: %d peticiones\n  %s", len(peticiones), strings.Join(peticiones, "\n  "))

	// --- 8. Los logs del flujo no dejan secretos ni URLs firmadas --------
	// (T29): se revisa la salida REAL de todo lo anterior: búsqueda con
	// cookies y CSRF, extractor jkplayer con su firma y petición al proxy.
	comprobarLogsSinSecretos(t, registro)
}

func contiene(lista []string, buscado string) bool {
	return contar(lista, buscado) > 0
}

func contar(lista []string, buscado string) int {
	n := 0
	for _, s := range lista {
		if s == buscado {
			n++
		}
	}
	return n
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

// ---------------------------------------------------------------------------
// E2E offline de AnimeAV1 (T35).
//
// La misma cadena completa que el E2E de JKAnime —búsqueda, detalle con
// lotes, fuentes y proxy con manifiesto + segmento— pero contra AnimeAV1:
// catálogo, ficha y episodio salen de los fixtures reales del paquete y el
// reproductor es artificial con el contrato observado en T30 (403 sin
// Sec-Fetch-Site, 206 con Range bytes=0-0). El sitio también sirve la
// búsqueda de JKAnime, para que SearchAnime cubra la búsqueda combinada
// (T34) con los dos proveedores registrados.
// ---------------------------------------------------------------------------

// Fixtures reales de AnimeAV1 que sirve el sitio (solo lectura; las
// portadas son del CDN y no se piden, así que no hace falta reescribirlos).
const (
	rutaFixtureCatalogoAV1 = "internal/scraper/animeav1/testdata/catalogo_data.json"
	rutaFixtureDetalleAV1  = "internal/scraper/animeav1/testdata/detail_data.json"
	rutaFixtureEpisodioAV1 = "internal/scraper/animeav1/testdata/episode_data.json"
)

// Rutas del sitio falso de AnimeAV1, además de /buscar/ y /excepciones.json
// de JKAnime, que comparten el mismo httptest.
const (
	rutaCatalogoAV1   = "/catalogo/__data.json"
	rutaDetalleAV1    = "/media/bleach/__data.json"
	rutaEpisodioAV1   = "/media/bleach/1/__data.json"
	rutaManifiestoAV1 = "/m3u8/"
	rutaSegmentoAV1   = "/segmento1.ts"
)

// hostJugadorPrueba es el host del reproductor propio: sources.go deriva
// siempre contra él y jamás pide /play/*, así que el transporte del proxy
// solo lo traduce al sitio falso y corta cualquier otro destino.
const hostJugadorPrueba = "player.zilla-networks.com"

// Cabeceras del contrato de T30: el segmento responde 403 sin
// Sec-Fetch-Site: same-origin y con ella 200, o 206 si trae Range.
const (
	cabeceraFetchPrueba = "Sec-Fetch-Site"
	valorFetchPrueba    = "same-origin"
)

// manifiestoAV1Prueba es el HLS que sirve el reproductor falso. La línea del
// segmento va con ruta ABSOLUTA: la URL del manifiesto es /m3u8/<id> y la
// resolución no debe arrastrar esa carpeta (HTML artificial, regla 11).
const manifiestoAV1Prueba = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:4
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:4.0,
/segmento1.ts
#EXT-X-ENDLIST
`

// reManifiestoAV1 es la regla del propio player (idJugadorValido en
// sources.go): /m3u8/ + id de exactamente 32 caracteres hexadecimales.
var reManifiestoAV1 = regexp.MustCompile("^" + rutaManifiestoAV1 + `[0-9a-fA-F]{32}$`)

// ---------------------------------------------------------------------------
// Transporte del proxy para el E2E offline: el doer inyectado en
// stream.NewProxy solo admite el host del reproductor —traducido al sitio
// falso— y el del propio sitio; cualquier otra cosa se corta ANTES de
// conectar y queda registrada.
//
// Es el equivalente offline del cliente con anti-SSRF de NewProxy(nil):
// esa anti-SSRF cortaría el 127.0.0.1 del httptest y PermitirRedPrivada()
// saldría a la red real en cuanto el manifiesto apuntara al jugador. La
// cabecera Sec-Fetch-Site sigue viva igual que en producción, porque la
// inyecta stream/reenvio.go desde la sesión ANTES de este transporte.
// Desviación de T35 anotada en PROGRESO.md.
// ---------------------------------------------------------------------------

type transporteJugador struct {
	jugador string // host del reproductor (sin puerto)
	sitio   string // host:puerto del sitio falso
	base    http.RoundTripper

	mu    sync.Mutex
	fuera []string
}

func (t *transporteJugador) RoundTrip(r *http.Request) (*http.Response, error) {
	switch r.URL.Host {
	case t.sitio:
		return t.base.RoundTrip(r)
	case t.jugador:
		// Misma petición, otro destino. La resolución de URIs relativas del
		// manifiesto no importa (el E2E las sirve absolutas) y el host del
		// jugador nunca llega al sitio.
		destino := *r.URL
		destino.Scheme = "http"
		destino.Host = t.sitio
		copia := r.Clone(r.Context())
		copia.URL = &destino
		return t.base.RoundTrip(copia)
	default:
		t.mu.Lock()
		t.fuera = append(t.fuera, r.URL.String())
		t.mu.Unlock()
		return nil, fmt.Errorf("petición del proxy fuera del reproductor y del sitio falso: %s", r.URL)
	}
}

// intentosFuera devuelve las peticiones que el transporte cortó (ninguna
// debe aparecer en un flujo offline correcto).
func (t *transporteJugador) intentosFuera() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.fuera...)
}

// ---------------------------------------------------------------------------
// Sitio falso de AnimeAV1: catálogo, ficha y episodio con los fixtures
// reales del paquete, la búsqueda de JKAnime del propio E2E y un
// reproductor artificial con el contrato de T30. Además del method+ruta
// guarda query, Sec-Fetch-Site, Range y status de cada petición, para poder
// exigir la lista EXACTA y el comportamiento de las cabeceras visto desde
// el origen.
// ---------------------------------------------------------------------------

// peticionAV1 es lo que el sitio registró de una petición.
type peticionAV1 struct {
	metodo   string
	ruta     string
	consulta string
	fetch    string // Sec-Fetch-Site
	rango    string // Range
	status   int
}

type sitioAV1 struct {
	srv *httptest.Server

	mu         sync.Mutex
	peticiones []peticionAV1

	htmlBusqueda string
	catalogo     string
	detalle      string
	episodio     string
	segmento     []byte
}

// nuevoSitioAV1 monta el servidor y prepara el contenido (que necesita la
// URL propia para la búsqueda de JKAnime).
func nuevoSitioAV1(t *testing.T) *sitioAV1 {
	t.Helper()
	s := &sitioAV1{}
	s.srv = httptest.NewServer(s)
	t.Cleanup(s.srv.Close)
	s.preparar(t)
	return s
}

func (s *sitioAV1) preparar(t *testing.T) {
	t.Helper()
	busqueda := strings.ReplaceAll(fixturePrueba(t, rutaFixtureBusqueda), "https://jkanime.net", s.srv.URL)
	catalogo := fixturePrueba(t, rutaFixtureCatalogoAV1)
	detalle := fixturePrueba(t, rutaFixtureDetalleAV1)
	episodio := fixturePrueba(t, rutaFixtureEpisodioAV1)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.htmlBusqueda = busqueda
	s.catalogo = catalogo
	s.detalle = detalle
	s.episodio = episodio
	s.segmento = bytes.Repeat([]byte("SEGMENTO_AV1_DE_PRUEBA\n"), 256)
}

func (s *sitioAV1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, tipo, cuerpo, contentRange := s.responder(r)
	if contentRange != "" {
		w.Header().Set("Content-Range", contentRange)
	}
	w.Header().Set("Content-Type", tipo)
	w.WriteHeader(status)
	_, _ = w.Write(cuerpo)
}

// responder decide la respuesta y registra la petición con el mutex puesto:
// así el registro y la lectura del contenido quedan sincronizados con
// preparar.
func (s *sitioAV1) responder(r *http.Request) (int, string, []byte, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fetch := r.Header.Get(cabeceraFetchPrueba)
	rango := r.Header.Get("Range")
	status, tipo, cuerpo, contentRange := s.contenido(r, r.URL.Path, fetch, rango)
	s.peticiones = append(s.peticiones, peticionAV1{
		metodo:   r.Method,
		ruta:     r.URL.Path,
		consulta: r.URL.RawQuery,
		fetch:    fetch,
		rango:    rango,
		status:   status,
	})
	return status, tipo, cuerpo, contentRange
}

func (s *sitioAV1) contenido(r *http.Request, ruta, fetch, rango string) (int, string, []byte, string) {
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(ruta, rutaBusqueda):
		return http.StatusOK, "text/html; charset=utf-8", []byte(s.htmlBusqueda), ""
	case r.Method == http.MethodGet && ruta == rutaExcepciones:
		return http.StatusOK, "application/json", []byte("{}"), ""
	case r.Method == http.MethodGet && ruta == rutaCatalogoAV1:
		return http.StatusOK, "application/json", []byte(s.catalogo), ""
	case r.Method == http.MethodGet && ruta == rutaDetalleAV1:
		return http.StatusOK, "application/json", []byte(s.detalle), ""
	case r.Method == http.MethodGet && ruta == rutaEpisodioAV1:
		return http.StatusOK, "application/json", []byte(s.episodio), ""
	case r.Method == http.MethodGet && reManifiestoAV1.MatchString(ruta):
		return http.StatusOK, "application/vnd.apple.mpegurl", []byte(manifiestoAV1Prueba), ""
	case r.Method == http.MethodGet && ruta == rutaSegmentoAV1:
		return s.segmentoRespuesta(fetch, rango)
	default:
		return http.StatusNotFound, "text/plain; charset=utf-8", []byte("no encontrado"), ""
	}
}

// segmentoRespuesta reproduce el contrato observado en T30: sin
// Sec-Fetch-Site responde 403; con él, 206 para Range bytes=0-0 y 200 con
// el cuerpo entero para cualquier otra petición.
func (s *sitioAV1) segmentoRespuesta(fetch, rango string) (int, string, []byte, string) {
	if fetch != valorFetchPrueba {
		return http.StatusForbidden, "text/plain; charset=utf-8", []byte("forbidden"), ""
	}
	if rango == "bytes=0-0" {
		return http.StatusPartialContent, "video/mp2t",
			append([]byte(nil), s.segmento[:1]...),
			fmt.Sprintf("bytes 0-0/%d", len(s.segmento))
	}
	return http.StatusOK, "video/mp2t", append([]byte(nil), s.segmento...), ""
}

func (s *sitioAV1) listaPeticiones() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.peticiones))
	for _, p := range s.peticiones {
		out = append(out, p.metodo+" "+p.ruta)
	}
	return out
}

func (s *sitioAV1) todas() []peticionAV1 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]peticionAV1(nil), s.peticiones...)
}

// cuantas cuenta las peticiones GET a una ruta concreta.
func (s *sitioAV1) cuantas(ruta string) int {
	n := 0
	for _, p := range s.listaPeticiones() {
		if p == "GET "+ruta {
			n++
		}
	}
	return n
}

func (s *sitioAV1) contenidoSegmento() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.segmento...)
}

// peticionesEn filtra las peticiones registradas por ruta.
func peticionesEn(ps []peticionAV1, ruta string) []peticionAV1 {
	var out []peticionAV1
	for _, p := range ps {
		if p.ruta == ruta {
			out = append(out, p)
		}
	}
	return out
}

// TestE2EAnimeAV1FlujoCompletoOffline encadena SearchAnime (combinada) →
// GetAnimeDetails de AnimeAV1 con streaming por lotes → GetVideoSources →
// SelectBestSource → manifiesto y segmento a través del proxy local, todo
// contra un httptest. Sin red: los dos transportes cortan cualquier host
// ajeno y el registro EXACTO de peticiones prueba que no hubo ni una de más.
func TestE2EAnimeAV1FlujoCompletoOffline(t *testing.T) {
	registro := capturarLogs(t)
	sitio := nuevoSitioAV1(t)

	transporte := &transporteSoloLocal{servidor: hostDe(sitio.srv.URL), base: http.DefaultTransport}
	doer := httpx.NewClient(httpx.ClientOptions{Transport: transporte})
	transportePx := &transporteJugador{jugador: hostJugadorPrueba, sitio: hostDe(sitio.srv.URL), base: http.DefaultTransport}
	px := stream.NewProxy(&http.Client{Timeout: 30 * time.Second, Transport: transportePx})

	reg := scraper.NewRegistry()
	proveedores := []scraper.Provider{
		// Los dos proveedores de la búsqueda combinada (T34): JKAnime contra
		// su sitio completo y AnimeAV1 contra sus fixtures, en el mismo
		// httptest.
		jkanime.New(doer, sitio.srv.URL,
			&jkanime.ExceptionResolver{URL: sitio.srv.URL + rutaExcepciones, Doer: doer}),
		animeav1.New(doer, sitio.srv.URL),
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
	rutaManifest := "" // se fija en el paso 4 con la URL del manifiesto

	// --- 1. Búsqueda combinada (T34) -------------------------------------
	resultados, err := a.SearchAnime("bleach")
	if err != nil {
		t.Fatalf("SearchAnime: %v", err)
	}
	if len(resultados) == 0 {
		t.Fatal("SearchAnime no devolvió ningún resultado")
	}
	jk, av1 := 0, 0
	slug := ""
	for _, r := range resultados {
		if r.ID == "" || r.Title == "" {
			t.Errorf("resultado incompleto: %+v", r)
		}
		if r.CoverURL == "" {
			t.Logf("resultado %q (%s) sin portada", r.ID, r.Provider)
		} else {
			validarURL(t, "portada del resultado "+r.Provider+":"+r.ID, r.CoverURL)
		}
		switch r.Provider {
		case proveedorPrincipal:
			jk++
		case proveedorComplemento:
			av1++
			if r.ID == "bleach" {
				slug = "bleach"
			}
		default:
			t.Errorf("resultado del proveedor desconido %q", r.Provider)
		}
	}
	if jk != 10 || av1 != 11 {
		t.Errorf("la búsqueda combinada devolvió %d de jkanime y %d de animeav1; se esperaban 10 y 11 (los fixtures)", jk, av1)
	}
	for i := 1; i < len(resultados); i++ {
		if resultados[i-1].Provider == proveedorComplemento && resultados[i].Provider == proveedorPrincipal {
			t.Errorf("jkanime no encabeza la lista: %q en la posición %d va tras %q",
				resultados[i].Provider, i, resultados[i-1].Provider)
		}
	}
	if slug == "" {
		t.Fatal("entre los resultados de animeav1 no está el slug bleach (hace falta el fixture del catálogo)")
	}
	t.Logf("búsqueda combinada: %d resultados (%d jkanime + %d animeav1); se sigue %q",
		len(resultados), jk, av1, slug)

	// --- 2. Detalle de AnimeAV1 con streaming y eventos ------------------
	primera, err := a.GetAnimeDetails(proveedorComplemento, slug)
	if err != nil {
		t.Fatalf("GetAnimeDetails(animeav1): %v", err)
	}
	if primera.Detail.ID != slug || primera.Detail.Title == "" || primera.Detail.Provider != proveedorComplemento {
		t.Errorf("detalle inesperado: %+v", primera.Detail)
	}
	if primera.Detail.TotalEpisodes != 366 {
		t.Errorf("TotalEpisodes = %d; el fixture de bleach declara 366", primera.Detail.TotalEpisodes)
	}
	validarURL(t, "portada del detalle", primera.Detail.Cover)
	if len(primera.Episodes) != 50 {
		t.Errorf("la primera tanda trae %d episodios; se esperaban 50 (tamLote de episodes.go)", len(primera.Episodes))
	}
	for _, ep := range primera.Episodes {
		validarEpisodioDe(t, "primera tanda", ep, proveedorComplemento)
	}
	t.Logf("detalle %q (%s, %s, %d declarados); %d episodios en la primera tanda",
		primera.Detail.Title, primera.Detail.Type, primera.Detail.Status,
		primera.Detail.TotalEpisodes, len(primera.Episodes))

	if !esperarCompleto(t, emis, slug, 30*time.Second) {
		t.Fatal("no llegó episodes:complete: el stream de episodios no cerró")
	}
	lotes := lotesDe(t, emis.lista(), slug)
	queriaTam := []int{50, 50, 50, 50, 50, 50, 16}
	if len(lotes) != len(queriaTam) {
		t.Fatalf("lotes = %d; se esperaban %d (366 = 50 devueltos + 316 por lotes de 50)", len(lotes), len(queriaTam))
	}
	total := len(primera.Episodes)
	for i, l := range lotes {
		if l.AnimeID != slug {
			t.Errorf("lote %d con AnimeID %q; se esperaba %q", l.Batch, l.AnimeID, slug)
		}
		if l.Batch != i+2 {
			t.Errorf("lote con Batch %d en la posición %d; se esperaba %d", l.Batch, i, i+2)
		}
		if queria := queriaTam[i]; len(l.Episodes) != queria {
			t.Errorf("lote %d: %d episodios; se esperaban %d", l.Batch, len(l.Episodes), queria)
		}
		if l.Done != (i == len(lotes)-1) {
			t.Errorf("lote %d con Done = %v", l.Batch, l.Done)
		}
		if l.Err != "" {
			t.Errorf("lote %d con error %q", l.Batch, l.Err)
		}
		for _, ep := range l.Episodes {
			validarEpisodioDe(t, "lote emitido", ep, proveedorComplemento)
			total++
		}
	}
	if total != 366 {
		t.Errorf("episodios contados = %d; se esperaban 366 (50 + 6×50 + 16)", total)
	}
	if completados := completadosDe(t, emis.lista(), slug); len(completados) != 1 || completados[0].Error != "" {
		t.Errorf("episodes:complete = %+v; se esperaba exactamente 1 sin error", completados)
	}
	t.Logf("eventos: %d lotes con %d episodios emitidos", len(lotes), total-len(primera.Episodes))

	// --- 3. La caché evita un segundo fetch -------------------------------
	detallesAntes := sitio.cuantas(rutaDetalleAV1)
	segunda, err := a.GetAnimeDetails(proveedorComplemento, slug)
	if err != nil {
		t.Fatalf("segunda GetAnimeDetails: %v", err)
	}
	if trasFetch := sitio.cuantas(rutaDetalleAV1); trasFetch != detallesAntes {
		t.Errorf("la segunda llamada volvió al proveedor: %d → %d peticiones al detalle", detallesAntes, trasFetch)
	}
	if len(segunda.Episodes) != 366 {
		t.Errorf("la caché devolvió %d episodios; se esperaban los 366 completos", len(segunda.Episodes))
	}
	for _, ep := range segunda.Episodes {
		validarEpisodioDe(t, "caché", ep, proveedorComplemento)
	}
	if trasCache := completadosDe(t, emis.lista(), slug); len(trasCache) != 2 {
		t.Errorf("episodes:complete tras la relectura = %d; se esperaban 2 (stream y caché)", len(trasCache))
	}
	t.Logf("caché: sin nuevo fetch y con los %d episodios acumulados", len(segunda.Episodes))

	// --- 4. Fuentes de vídeo y elección de la mejor -----------------------
	epID := primera.Episodes[0].ID
	if epID != "animeav1:bleach:episode:1" {
		t.Errorf("el primer episodio es %q; se esperaba animeav1:bleach:episode:1", epID)
	}
	fuentes, err := a.GetVideoSources(proveedorComplemento, epID, false)
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
	conteo := map[models.ResourceType]int{}
	for _, f := range fuentes {
		if f.Server == "" {
			t.Errorf("fuente sin etiqueta de servidor: %+v", f)
		}
		if f.Language != models.LanguageSub && f.Language != models.LanguageDub {
			t.Errorf("la fuente %s sale sin idioma SUB/DUB: %q", f.Server, f.Language)
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
		// T29: lo que cruza el puente de Wails sale sin cabeceras: viven
		// solo en la sesión del proxy.
		if enProxy && (f.Referer != "" || f.Headers != nil) {
			t.Errorf("la fuente %s va hacia el frontend con cabeceras: %+v", f.Server, f)
		}
		conteo[f.Type]++
	}
	if len(fuentes) != 10 || conteo[models.ResourceHLS] != 2 || conteo[models.ResourceIframe] != 8 {
		t.Errorf("fuentes = %d (hls %d, iframe %d); el fixture de bleach/1 trae 10 (2 HLS + 8 iframes)",
			len(fuentes), conteo[models.ResourceHLS], conteo[models.ResourceIframe])
	}
	if fuentes[0].Type != models.ResourceHLS || fuentes[0].Language != models.LanguageSub {
		t.Errorf("la primera fuente es %s (%s, %s); se esperaba hls/SUB del reproductor propio",
			fuentes[0].Type, fuentes[0].Server, fuentes[0].Language)
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
	du, err := url.Parse(destino)
	if err != nil {
		t.Fatalf("destino de la mejor fuente no parseable: %v", err)
	}
	if du.Host != hostJugadorPrueba || !reManifiestoAV1.MatchString(du.Path) {
		t.Fatalf("la mejor fuente apunta a %q; se esperaba https://%s/m3u8/<id de 32 hex>",
			recortarURL(destino), hostJugadorPrueba)
	}
	rutaManifest = du.Path
	t.Logf("fuentes: %d; mejor = %s (%s, %s) vía proxy\n%s",
		len(fuentes), mejor.Type, mejor.Server, mejor.Language, resumenFuentes(fuentes))

	// --- 5. Manifiesto y segmento a través del proxy ----------------------
	cliente := &http.Client{Timeout: 30 * time.Second}
	resp, err := cliente.Get(mejor.URL)
	if err != nil {
		t.Fatalf("GET del manifiesto a través del proxy: %v", recortarError(err, mejor.URL))
	}
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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
	if destinoProxy(t, segmento) != sitio.srv.URL+rutaSegmentoAV1 {
		t.Errorf("el segmento reescrito apunta a %q; se esperaba %q",
			destinoProxy(t, segmento), sitio.srv.URL+rutaSegmentoAV1)
	}

	// El segmento a través del proxy lleva la cabecera de la sesión y Range
	// bytes=0-0: el origen debe devolver exactamente 1 byte con 206.
	reqRango, err := http.NewRequest(http.MethodGet, segmento, nil)
	if err != nil {
		t.Fatalf("preparando la petición con Range: %v", err)
	}
	reqRango.Header.Set("Range", "bytes=0-0")
	respRango, err := cliente.Do(reqRango)
	if err != nil {
		t.Fatalf("GET del segmento a través del proxy: %v", recortarError(err, segmento))
	}
	cuerpoRango, err := io.ReadAll(respRango.Body)
	respRango.Body.Close()
	if err != nil {
		t.Fatalf("leyendo el segmento: %v", err)
	}
	if respRango.StatusCode != http.StatusPartialContent {
		t.Errorf("el proxy respondió %d para el segmento con Range; se esperaba 206", respRango.StatusCode)
	}
	if cr := respRango.Header.Get("Content-Range"); !strings.HasPrefix(cr, "bytes 0-0/") {
		t.Errorf("Content-Range = %q; se esperaba bytes 0-0/<tamaño>", cr)
	}
	if !bytes.Equal(cuerpoRango, sitio.contenidoSegmento()[:1]) {
		t.Errorf("el segmento con Range tiene %d bytes y no es el primero del sitio", len(cuerpoRango))
	}

	// Directo SIN Sec-Fetch-Site → 403; directo CON la cabecera → 200.
	respSin, err := cliente.Get(sitio.srv.URL + rutaSegmentoAV1)
	if err != nil {
		t.Fatalf("GET directo del segmento sin cabecera: %v", err)
	}
	_, _ = io.Copy(io.Discard, respSin.Body)
	respSin.Body.Close()
	if respSin.StatusCode != http.StatusForbidden {
		t.Errorf("el sitio respondió %d al segmento sin Sec-Fetch-Site; se esperaba 403", respSin.StatusCode)
	}
	reqCon, err := http.NewRequest(http.MethodGet, sitio.srv.URL+rutaSegmentoAV1, nil)
	if err != nil {
		t.Fatalf("preparando la petición con cabecera: %v", err)
	}
	reqCon.Header.Set(cabeceraFetchPrueba, valorFetchPrueba)
	respCon, err := cliente.Do(reqCon)
	if err != nil {
		t.Fatalf("GET directo del segmento con cabecera: %v", err)
	}
	cuerpoCon, err := io.ReadAll(respCon.Body)
	respCon.Body.Close()
	if err != nil {
		t.Fatalf("leyendo el segmento directo: %v", err)
	}
	if respCon.StatusCode != http.StatusOK {
		t.Errorf("el sitio respondió %d al segmento con Sec-Fetch-Site; se esperaba 200", respCon.StatusCode)
	}
	if !bytes.Equal(cuerpoCon, sitio.contenidoSegmento()) {
		t.Errorf("el segmento directo tiene %d bytes y no coincide con el del sitio (%d)",
			len(cuerpoCon), len(sitio.contenidoSegmento()))
	}
	t.Logf("proxy: manifiesto de %d bytes con el segmento reescrito; Range → 206 (%d byte), directo → 403/200 según Sec-Fetch-Site",
		len(cuerpo), len(cuerpoRango))

	// --- 6. Nada salió a la red -------------------------------------------
	if fuera := transporte.intentosFuera(); len(fuera) > 0 {
		t.Errorf("el proveedor intentó salir del sitio falso %d vez/veces: %v", len(fuera), fuera)
	}
	if fuera := transportePx.intentosFuera(); len(fuera) > 0 {
		t.Errorf("el proxy intentó salir de sus destinos permitidos %d vez/veces: %v", len(fuera), fuera)
	}

	// Registro completo de peticiones: además de "todas presentes", la
	// lista debe ser EXACTA: una petición repetida (una segunda búsqueda,
	// un segundo fetch del detalle o del catálogo…) es un fallo, no un
	// detalle.
	peticiones := sitio.listaPeticiones()
	queria := []string{
		"GET " + rutaBusqueda + "bleach",
		"GET " + rutaExcepciones,
		"GET " + rutaCatalogoAV1,
		"GET " + rutaDetalleAV1,
		"GET " + rutaEpisodioAV1,
		"GET " + rutaManifest,
		"GET " + rutaSegmentoAV1,
		"GET " + rutaSegmentoAV1,
		"GET " + rutaSegmentoAV1,
	}
	esperadas := append([]string(nil), queria...)
	hubieron := append([]string(nil), peticiones...)
	sort.Strings(esperadas)
	sort.Strings(hubieron)
	if strings.Join(esperadas, "\n") != strings.Join(hubieron, "\n") {
		t.Errorf("las peticiones no son exactas (quería %d, se hicieron %d)\n  quería:\n  %s\n  hizo:\n  %s",
			len(esperadas), len(hubieron),
			strings.Join(esperadas, "\n  "), strings.Join(hubieron, "\n  "))
	}
	exactas := map[string]int{
		rutaBusqueda + "bleach": 1,
		rutaExcepciones:         1,
		rutaCatalogoAV1:         1,
		rutaDetalleAV1:          1,
		rutaEpisodioAV1:         1,
		rutaManifest:            1,
		rutaSegmentoAV1:         3,
	}
	for ruta, querido := range exactas {
		if n := contar(peticiones, "GET "+ruta); n != querido {
			t.Errorf("la petición %q se hizo %d veces; se esperaba %d", ruta, n, querido)
		}
	}
	for _, p := range sitio.todas() {
		if p.ruta == rutaCatalogoAV1 && p.consulta != "search=bleach" {
			t.Errorf("la búsqueda de animeav1 llegó con query %q; se esperaba search=bleach", p.consulta)
		}
	}

	// El contrato de las cabeceras, visto por el propio origen.
	segmentos := peticionesEn(sitio.todas(), rutaSegmentoAV1)
	var proxied, conCab, sinCab int
	for _, p := range segmentos {
		switch {
		case p.fetch == valorFetchPrueba && p.rango == "bytes=0-0":
			proxied++
			if p.status != http.StatusPartialContent {
				t.Errorf("el segmento con cabecera y Range devolvió %d en el origen; se esperaba 206", p.status)
			}
		case p.fetch == valorFetchPrueba && p.rango == "":
			conCab++
			if p.status != http.StatusOK {
				t.Errorf("el segmento con cabecera devolvió %d en el origen; se esperaba 200", p.status)
			}
		case p.fetch == "":
			sinCab++
			if p.status != http.StatusForbidden {
				t.Errorf("el segmento sin cabecera devolvió %d en el origen; se esperaba 403", p.status)
			}
		default:
			t.Errorf("petición de segmento con cabeceras inesperadas: %+v", p)
		}
	}
	if proxied != 1 || conCab != 1 || sinCab != 1 {
		t.Errorf("segmentos: %d vía proxy, %d directos con cabecera y %d sin; se esperaba 1 de cada uno",
			proxied, conCab, sinCab)
	}
	t.Logf("sitio falso: %d peticiones\n  %s", len(peticiones), strings.Join(peticiones, "\n  "))

	// --- 7. Los logs del flujo no dejan secretos ni URLs firmadas ---------
	comprobarLogsSinSecretos(t, registro)
}
