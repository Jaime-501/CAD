package stream

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"miapp/internal/logx"
	"miapp/internal/models"
)

func TestMain(m *testing.M) {
	// Los tests no ensucian la salida con los logs de logx.Request.
	logx.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// clientePrueba habla con el proxy sin pasar por el proxy de entorno
// (HTTP_PROXY), por si el contenedor lo define.
var clientePrueba = &http.Client{Transport: &http.Transport{}}

// proxyDePrueba monta un proxy escuchando en loopback. Activa
// PermitirRedPrivada porque los upstream de los tests son httptest.Server,
// que escuchan en 127.0.0.1.
func proxyDePrueba(t *testing.T, opts ...Option) (*Proxy, string) {
	t.Helper()
	p := NewProxy(nil, append([]Option{PermitirRedPrivada()}, opts...)...)
	dir, err := p.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	if !strings.HasPrefix(dir, "http://127.0.0.1:") {
		t.Fatalf("dir = %q, tenía que escuchar solo en 127.0.0.1", dir)
	}
	t.Cleanup(func() { _ = p.Cerrar() })
	return p, dir
}

func upstreamPrueba(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func pedir(t *testing.T, u string, preparar func(*http.Request)) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if preparar != nil {
		preparar(req)
	}
	resp, err := clientePrueba.Do(req)
	if err != nil {
		t.Fatalf("petición a %s: %v", urlSeguraDe(u), err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func urlSeguraDe(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	return urlSegura(parsed)
}

func cuerpo(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leyendo cuerpo: %v", err)
	}
	return string(b)
}

// obtenerSesion extrae el id de sesión de la URL que devuelve Register.
func obtenerSesion(t *testing.T, u string) string {
	t.Helper()
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("URL del proxy: %v", err)
	}
	id := parsed.Query().Get("s")
	if id == "" {
		t.Fatalf("la URL del proxy no lleva sesión: %s", u)
	}
	return id
}

// TestEscuchaSoloEnLoopbackYPuertoAleatorio comprueba el bind del proxy y
// que Handler responde (404 por sesión inexistente) en esa dirección.
func TestEscuchaSoloEnLoopbackYPuertoAleatorio(t *testing.T) {
	p := NewProxy(nil, PermitirRedPrivada())
	defer func() { _ = p.Cerrar() }()

	dir, err := p.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	if !strings.HasPrefix(dir, "http://127.0.0.1:") || strings.HasSuffix(dir, ":0") {
		t.Errorf("dir = %q, quería 127.0.0.1 con puerto aleatorio", dir)
	}
	otra, err := p.Escuchar()
	if err != nil || otra != dir {
		t.Errorf("segunda Escuchar = (%q, %v), debía devolver la misma base", otra, err)
	}

	resp := pedir(t, dir+"/proxy?s=ida&u="+url.QueryEscape("https://cdn.example.test/a.mp4"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, quería 404 con sesión inexistente", resp.StatusCode)
	}
}

// TestCerrarDetieneElServidor comprueba que Cerrar termina el goroutine de
// Serve que arranca Escuchar (nada sobrevive al shutdown) y que el puerto
// queda inutilizable.
func TestCerrarDetieneElServidor(t *testing.T) {
	p := NewProxy(nil, PermitirRedPrivada())
	base := runtime.NumGoroutine()

	dir, err := p.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}

	resp := pedir(t, dir+"/proxy?s=ida&u="+url.QueryEscape("https://cdn.example.test/a.mp4"), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, quería 404 con sesión inexistente", resp.StatusCode)
	}
	_ = resp.Body.Close()

	if err := p.Cerrar(); err != nil {
		t.Fatalf("Cerrar: %v", err)
	}
	clientePrueba.Transport.(*http.Transport).CloseIdleConnections()

	if _, err := http.Get(dir + "/proxy"); err == nil {
		t.Error("tras Cerrar el proxy seguía respondiendo")
	}

	limite := time.Now().Add(3 * time.Second)
	for runtime.NumGoroutine() > base && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > base {
		t.Errorf("goroutines = %d (base %d): Cerrar dejó algo corriendo", n, base)
	}
}

// TestInyectaCabecerasDeLaSesionYNoLasDelCliente: el Referer y los Headers
// del VideoSource van en la sesión y llegan al origen; Cookie y
// Authorization del reproductor jamás; Range sí.
func TestInyectaCabecerasDeLaSesionYNoLasDelCliente(t *testing.T) {
	recibido := make(chan http.Header, 1)
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		recibido <- r.Header.Clone()
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("cuerpo"))
	})

	p, dir := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{
		URL:       up.URL + "/archivo.mp4",
		Type:      models.ResourceMP4,
		Referer:   "https://jkanime.net/jkplayer/um?e=E_PRUEBA",
		Headers:   map[string]string{"X-Prueba": "sesion", "User-Agent": "UA-de-sesion"},
		Subtitles: []models.Subtitle{{URL: up.URL + "/sub.vtt", Lang: "es"}},
	})

	if reg.URL == up.URL+"/archivo.mp4" {
		t.Fatal("Register no reescribió la URL")
	}
	u, err := url.Parse(reg.URL)
	if err != nil {
		t.Fatalf("URL del proxy: %v", err)
	}
	if !strings.HasPrefix(reg.URL, dir+"/proxy?") {
		t.Errorf("URL = %q, debía apuntar al proxy (%q)", reg.URL, dir)
	}
	claves := u.Query()
	if len(claves) != 2 || claves.Get("s") == "" || claves.Get("u") != up.URL+"/archivo.mp4" {
		t.Errorf("query = %v, solo deben ir s y u (nunca cabeceras)", claves)
	}
	if len(reg.Subtitles) != 1 || !strings.HasPrefix(reg.Subtitles[0].URL, dir+"/proxy?") {
		t.Errorf("subtítulo no reescrito: %+v", reg.Subtitles)
	}
	// T29: las cabeceras se quedan en la sesión; la fuente devuelta (la que
	// recibe el frontend) sale limpia. Prueba de que siguen ahí: lo de abajo.
	if reg.Referer != "" || reg.Headers != nil {
		t.Errorf("la fuente devuelta arrastra cabeceras: referer=%q headers=%v",
			reg.Referer, reg.Headers)
	}

	resp := pedir(t, reg.URL, func(r *http.Request) {
		r.Header.Set("Cookie", "sesion=SECRETA")
		r.Header.Set("Authorization", "Bearer SECRETO")
		r.Header.Set("Range", "bytes=0-4")
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, quería 200", resp.StatusCode)
	}
	if got := cuerpo(t, resp); got != "cuerpo" {
		t.Errorf("cuerpo = %q", got)
	}

	h := <-recibido
	if h.Get("Referer") != "https://jkanime.net/jkplayer/um?e=E_PRUEBA" {
		t.Errorf("Referer upstream = %q, no se inyectó el de la sesión", h.Get("Referer"))
	}
	if h.Get("X-Prueba") != "sesion" {
		t.Errorf("X-Prueba upstream = %q", h.Get("X-Prueba"))
	}
	if h.Get("User-Agent") != "UA-de-sesion" {
		t.Errorf("User-Agent upstream = %q, debía ser el de la sesión", h.Get("User-Agent"))
	}
	if h.Get("Range") != "bytes=0-4" {
		t.Errorf("Range upstream = %q, debía reenviarse", h.Get("Range"))
	}
	if h.Get("Cookie") != "" {
		t.Error("se reenvió la Cookie del cliente")
	}
	if h.Get("Authorization") != "" {
		t.Error("se reenvió la Authorization del cliente")
	}
}

// TestRangeDevuelve206: el Range del reproductor llega al origen y su 206
// (con Content-Range) vuelve íntegro y con CORS expuesto.
func TestRangeDevuelve206(t *testing.T) {
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=0-0" {
			t.Errorf("Range upstream = %q, quería bytes=0-0", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", "bytes 0-0/100")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte{42})
	})

	p, _ := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{URL: up.URL + "/pelicula.mp4", Type: models.ResourceMP4})

	resp := pedir(t, reg.URL, func(r *http.Request) { r.Header.Set("Range", "bytes=0-0") })
	if resp.StatusCode != http.StatusPartialContent {
		t.Errorf("status = %d, quería 206", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Range"); got != "bytes 0-0/100" {
		t.Errorf("Content-Range = %q", got)
	}
	if got := resp.Header.Get("Accept-Ranges"); got != "bytes" {
		t.Errorf("Accept-Ranges = %q", got)
	}
	if got := resp.Header.Get("Content-Length"); got != "1" {
		t.Errorf("Content-Length = %q, quería 1", got)
	}
	if !strings.Contains(resp.Header.Get("Access-Control-Expose-Headers"), "Content-Range") {
		t.Errorf("Expose-Headers = %q", resp.Header.Get("Access-Control-Expose-Headers"))
	}
	if b := cuerpo(t, resp); len(b) != 1 || b[0] != 42 {
		t.Errorf("cuerpo = %v, quería 1 byte", []byte(b))
	}
}

// TestSesionExpiradaDevuelve404: caducada la TTL, la URL registrada deja de
// servir (y una sesión desconocida también).
func TestSesionExpiradaDevuelve404(t *testing.T) {
	var llamadas atomic.Int32
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	p, dir := proxyDePrueba(t, ConTTL(30*time.Millisecond))
	reg := p.Register(models.VideoSource{URL: up.URL + "/a.mp4"})

	if resp := pedir(t, reg.URL, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("status antes de caducar = %d, quería 200", resp.StatusCode)
	}
	time.Sleep(80 * time.Millisecond)
	if resp := pedir(t, reg.URL, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("status con sesión caducada = %d, quería 404", resp.StatusCode)
	}
	despues := dir + "/proxy?s=sin-registrar&u=" + url.QueryEscape(up.URL+"/a.mp4")
	if resp := pedir(t, despues, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("status con sesión desconocida = %d, quería 404", resp.StatusCode)
	}
	if n := llamadas.Load(); n != 1 {
		t.Errorf("el upstream recibió %d peticiones: solo la primera (con sesión viva)", n)
	}
}

// TestCancelarElClienteCancelaElUpstream: si el reproductor corta la
// conexión, la petición al origen se cancela (no se queda colgado).
func TestCancelarElClienteCancelaElUpstream(t *testing.T) {
	arrancado := make(chan struct{}, 1)
	cancelado := make(chan struct{}, 1)
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case arrancado <- struct{}{}:
		default:
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
		cancelado <- struct{}{}
	})

	p, _ := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{URL: up.URL + "/largo.mp4", Type: models.ResourceMP4})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reg.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	terminado := make(chan struct{})
	go func() {
		defer close(terminado)
		if resp, err := clientePrueba.Do(req); err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-arrancado:
	case <-time.After(5 * time.Second):
		t.Fatal("el upstream nunca recibió la petición")
	}
	time.Sleep(50 * time.Millisecond) // deja que el proxy empiece a copiar
	cancel()

	select {
	case <-cancelado:
	case <-time.After(5 * time.Second):
		t.Fatal("el upstream no se canceló al cerrar el reproductor la conexión")
	}
	select {
	case <-terminado:
	case <-time.After(5 * time.Second):
		t.Fatal("la petición del reproductor no terminó")
	}
}

// TestOPTIONSDevuelveCabecerasCORS: el preflight (necesario por Range) lo
// responde el proxy; el upstream devuelve 405 (comprobado en el sondeo).
func TestOPTIONSDevuelveCabecerasCORS(t *testing.T) {
	p := NewProxy(nil, PermitirRedPrivada())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/proxy?s=x&u=https%3A%2F%2Fcdn.example.test%2Fa.m3u8", nil)
	req.Header.Set("Access-Control-Request-Headers", "range")
	p.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, quería 204", rec.Code)
	}
	h := rec.Header()
	if h.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Allow-Origin = %q", h.Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(h.Get("Access-Control-Allow-Headers"), "range") {
		t.Errorf("Allow-Headers = %q, debía cubrir Range", h.Get("Access-Control-Allow-Headers"))
	}
	if !strings.Contains(h.Get("Access-Control-Allow-Methods"), http.MethodGet) {
		t.Errorf("Allow-Methods = %q", h.Get("Access-Control-Allow-Methods"))
	}
	if h.Get("Access-Control-Expose-Headers") != "Content-Range, Accept-Ranges, Content-Length" {
		t.Errorf("Expose-Headers = %q", h.Get("Access-Control-Expose-Headers"))
	}
}

// TestCorsEnTodasLasRespuestas: también en los caminos de error (404, 400,
// 405), no solo en las respuestas de vídeo.
func TestCorsEnTodasLasRespuestas(t *testing.T) {
	p, dir := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{URL: "https://cdn.example.test/a.mp4"})
	sesion := url.QueryEscape(obtenerSesion(t, reg.URL))

	casos := []struct {
		nombre string
		u      string
		method string
		status int
	}{
		{"ruta inexistente", dir + "/otra", http.MethodGet, http.StatusNotFound},
		{"sesión desconocida", dir + "/proxy?s=otra-sesion&u=" + url.QueryEscape("https://cdn.example.test/a.mp4"), http.MethodGet, http.StatusNotFound},
		{"destino inválido", dir + "/proxy?s=" + sesion + "&u=" + url.QueryEscape("ftp://interno/a"), http.MethodGet, http.StatusBadRequest},
		{"método no permitido", reg.URL, http.MethodPost, http.StatusMethodNotAllowed},
	}
	for _, c := range casos {
		req, err := http.NewRequest(c.method, c.u, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.nombre, err)
		}
		resp, err := clientePrueba.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", c.nombre, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != c.status {
			t.Errorf("%s: status = %d, quería %d", c.nombre, resp.StatusCode, c.status)
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s: sin Access-Control-Allow-Origin", c.nombre)
		}
		if !strings.Contains(resp.Header.Get("Access-Control-Expose-Headers"), "Content-Range") {
			t.Errorf("%s: sin Access-Control-Expose-Headers", c.nombre)
		}
	}
}

// TestDestinoPrivadoRechazadoPorDefecto: con la opción por defecto (la
// restrictiva) el proxy no llega ni a conectar con un destino privado.
func TestDestinoPrivadoRechazadoPorDefecto(t *testing.T) {
	var llamadas atomic.Int32
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	p := NewProxy(nil) // sin PermitirRedPrivada: valor por defecto
	dir, err := p.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	defer func() { _ = p.Cerrar() }()

	reg := p.Register(models.VideoSource{URL: up.URL + "/privado.mp4", Type: models.ResourceMP4})
	resp := pedir(t, reg.URL, nil)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, quería 403 (destino privado bloqueado)", resp.StatusCode)
	}
	if n := llamadas.Load(); n != 0 {
		t.Errorf("el upstream recibió %d peticiones, ninguna debía llegar", n)
	}
	if dir == "" {
		t.Fatal("dir vacío")
	}
}

// transporteSalto responde un 302 hacia destino para el host ficticio
// remoto.test y delega el resto en el transporte real: permite encadenar un
// redirect sin salir a la red.
type transporteSalto struct {
	real    http.RoundTripper
	destino string
}

func (t *transporteSalto) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Hostname() != "remoto.test" {
		return t.real.RoundTrip(r)
	}
	return &http.Response{
		StatusCode: http.StatusFound,
		Status:     "302 Found",
		Header:     http.Header{"Location": []string{t.destino}},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    r,
		ProtoMajor: 1,
		ProtoMinor: 1,
	}, nil
}

// TestRedirectALoopbackRechazadoPorDefecto: el control de red aplica también
// a cada salto de un redirect, así que un origen que manda a loopback no se
// llega a tocar.
func TestRedirectALoopbackRechazadoPorDefecto(t *testing.T) {
	var llamadas atomic.Int32
	victima := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	p := NewProxy(nil) // valor por defecto: restrictivo
	if p.cliente == nil {
		t.Fatal("el proxy debía construir su cliente con anti-SSRF")
	}
	p.cliente.Transport = &transporteSalto{
		real:    p.cliente.Transport,
		destino: victima.URL + "/interno",
	}
	if _, err := p.Escuchar(); err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	defer func() { _ = p.Cerrar() }()

	reg := p.Register(models.VideoSource{URL: "http://remoto.test/video.mp4", Type: models.ResourceMP4})
	resp := pedir(t, reg.URL, nil)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, quería 403 (redirect a loopback bloqueado)", resp.StatusCode)
	}
	if n := llamadas.Load(); n != 0 {
		t.Errorf("el destino de loopback recibió %d peticiones, ninguna debía llegar", n)
	}
}

// TestURLDestinoInvalidaDevuelve400: solo http/https.
func TestURLDestinoInvalidaDevuelve400(t *testing.T) {
	p, dir := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{URL: "https://cdn.example.test/a.mp4"})
	sesion := obtenerSesion(t, reg.URL)

	for _, destino := range []string{"ftp://cdn.example.test/a", "javascript:alert(1)", "no-es-una-url"} {
		u := dir + "/proxy?s=" + url.QueryEscape(sesion) + "&u=" + url.QueryEscape(destino)
		if resp := pedir(t, u, nil); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("destino %q: status = %d, quería 400", destino, resp.StatusCode)
		}
	}
}

// TestVTTSeSirveConContentTypeTextVTT: el origen manda text/plain y el
// proxy lo fija como text/vtt (con CORS).
func TestVTTSeSirveConContentTypeTextVTT(t *testing.T) {
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("WEBVTT\n\n00:00.000 --> 00:01.000\nhola\n"))
	})

	p, _ := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{
		URL:       up.URL + "/subtitulo.vtt",
		Subtitles: []models.Subtitle{{URL: up.URL + "/subtitulo.vtt", Lang: "es"}},
	})

	resp := pedir(t, reg.Subtitles[0].URL, nil)
	if got := resp.Header.Get("Content-Type"); got != "text/vtt" {
		t.Errorf("Content-Type = %q, quería text/vtt", got)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("falta CORS en la respuesta del subtítulo")
	}
	if c := cuerpo(t, resp); !strings.HasPrefix(c, "WEBVTT") {
		t.Errorf("cuerpo = %q", c)
	}
	if reg.URL == "" {
		t.Fatal("Register devolvió URL vacía")
	}
}

// TestLimiteDeSesiones: nunca se supera el cupo; al llenarse se evapora la
// que antes caduca.
func TestLimiteDeSesiones(t *testing.T) {
	p, _ := proxyDePrueba(t, ConLimiteSesiones(3))
	for i := 0; i < 8; i++ {
		p.Register(models.VideoSource{URL: "https://cdn.example.test/" + string(rune('a'+i)) + ".m3u8"})
	}
	p.sesiones.mu.Lock()
	vivas := len(p.sesiones.vivas)
	p.sesiones.mu.Unlock()
	if vivas > 3 {
		t.Errorf("sesiones vivas = %d, el límite era 3", vivas)
	}
}

// TestDoerInyectadoSeUsa: si el llamador pasa su doer, el proxy lo usa y no
// construye cliente propio.
func TestDoerInyectadoSeUsa(t *testing.T) {
	falso := &doerFalso{}
	p := NewProxy(falso)
	if p.cliente != nil {
		t.Error("con doer inyectado no debe construirse cliente propio")
	}
	dir, err := p.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	defer func() { _ = p.Cerrar() }()

	reg := p.Register(models.VideoSource{URL: "https://cdn.example.test/a.mp4"})
	resp := pedir(t, reg.URL, nil)
	if resp.StatusCode != http.StatusOK || cuerpo(t, resp) != "ok" {
		t.Errorf("status = %d, debía salir la respuesta del doer inyectado", resp.StatusCode)
	}
	if falso.llamadas.Load() != 1 {
		t.Errorf("el doer inyectado se llamó %d veces", falso.llamadas.Load())
	}
	if dir == "" {
		t.Fatal("dir vacío")
	}
}

// TestRegisterSinBaseSeDevuelveIgual: sin Escuchar ni ConBaseURL no hay
// dónde apuntar y la fuente sale intacta (sin sesión creada).
func TestRegisterSinBaseSeDevuelveIgual(t *testing.T) {
	p := NewProxy(nil, PermitirRedPrivada())
	reg := p.Register(models.VideoSource{URL: "https://cdn.example.test/a.mp4"})
	if reg.URL != "https://cdn.example.test/a.mp4" {
		t.Errorf("URL = %q, debía quedarse sin registrar", reg.URL)
	}
	p.sesiones.mu.Lock()
	vivas := len(p.sesiones.vivas)
	p.sesiones.mu.Unlock()
	if vivas != 0 {
		t.Errorf("sesiones creadas = %d, no debía crear ninguna", vivas)
	}
}

// TestURLSeguraRedactaTokensDelPath: T62. urlSegura registra la URL de
// destino del reenvío; algunos CDN esconden el token en el PATH (MP4Upload
// /d/<token>/video.mp4, zilla /m3u8/<32 hex>), así que se enmascara sin
// tocar host, puerto ni el último segmento con extensión. Un slug normal no
// se toca.
func TestURLSeguraRedactaTokensDelPath(t *testing.T) {
	casos := []struct {
		in, quiere string
	}{
		{
			"https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4?st=FIRMA_ST",
			"https://a2.mp4upload.com:183/d/***/video.mp4",
		},
		{
			"https://a2.mp4upload.com:183/d/Xq7bK2mNpR4t/video.mp4",
			"https://a2.mp4upload.com:183/d/***/video.mp4",
		},
		{
			"https://player.zilla-networks.com/m3u8/65f33e9a44ff7767faf60d1dae9eb1e6",
			"https://player.zilla-networks.com/m3u8/***",
		},
		{
			"https://jkanime.net/bleach/1/",
			"https://jkanime.net/bleach/1/",
		},
	}
	for _, c := range casos {
		u, err := url.Parse(c.in)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", c.in, err)
		}
		if got := urlSegura(u); got != c.quiere {
			t.Errorf("urlSegura(%q) = %q; quería %q", c.in, got, c.quiere)
		}
	}
	if urlSegura(nil) != "" {
		t.Errorf("urlSegura(nil) = %q; quería vacío", urlSegura(nil))
	}
}

// doerFalso responde siempre 200 "ok" y cuenta las llamadas.
type doerFalso struct {
	llamadas atomic.Int32
}

func (d *doerFalso) Do(r *http.Request) (*http.Response, error) {
	d.llamadas.Add(1)
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Header:        http.Header{"Content-Type": []string{"video/mp4"}},
		Body:          io.NopCloser(strings.NewReader("ok")),
		ContentLength: 2,
		Request:       r,
		ProtoMajor:    1,
		ProtoMinor:    1,
	}, nil
}
