// Package stream implementa el proxy local de streaming: permite que
// <video> y hls.js reproduzcan fuentes que exigen Referer u otras cabeceras
// personalizadas o que bloquean por CORS.
//
// El reproductor nunca habla con el CDN: pide a
// http://127.0.0.1:<puerto>/proxy?s=<sesión>&u=<url> y el proxy reenvía la
// petición con httpx.HTTPDoer.Do (cuerpo en streaming, sin cargarlo en
// memoria) usando las cabeceras guardadas en la sesión por Register: las
// cabeceras requeridas viven en la sesión (TTL + límite), nunca en la URL.
//
// Seguridad:
//   - Escucha solo en 127.0.0.1 con puerto aleatorio (Escuchar).
//   - Anti-SSRF: el cliente que construye el proxy usa un net.Dialer cuyo
//     Control valida la IP ya resuelta de cada conexión —también la de cada
//     redirect, porque el mismo transporte las resuelve— y rechaza loopback,
//     privadas, link-local, no especificadas y multicast. La opción
//     PermitirRedPrivada desactiva SOLO ese control y la usan los tests
//     (httptest.Server escucha en 127.0.0.1); el valor por defecto es el
//     restrictivo.
//   - No se reenvían Cookie/Authorization del cliente, se descartan las
//     cabeceras hop-by-hop y de la respuesta solo se copia una lista blanca.
//     Los logs pasan por logx con la URL reducida a esquema+host+ruta:
//     nunca se registran URLs firmadas completas.
package stream

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/models"
)

// Opción de configuración de NewProxy.
type Option func(*Proxy)

// PermitirRedPrivada desactiva SOLO el control de IP resuelta del cliente
// interno (la anti-SSRF). Solo la usan los tests: con el control activo
// ningún destino local sería alcanzable y httptest.Server escucha en
// 127.0.0.1. Sin esta opción el proxy es restrictivo por defecto.
func PermitirRedPrivada() Option {
	return func(p *Proxy) { p.permitirRedPrivada = true }
}

// ConTTL cambia la vida de las sesiones (por defecto TTLDefecto). La
// caducidad es por deslizamiento: cada acceso la renueva.
func ConTTL(d time.Duration) Option {
	return func(p *Proxy) { p.sesiones.ttl = d }
}

// ConLimiteSesiones cambia el número máximo de sesiones vivas (por defecto
// LimiteDefecto). Al superarlo se purgan las caducadas y, si siguen llenas,
// la que antes caduca.
func ConLimiteSesiones(n int) Option {
	return func(p *Proxy) { p.sesiones.limite = n }
}

// ConBaseURL fija la URL base que usa Register para construir las URLs del
// proxy. Hace falta cuando el Handler se monta en otro servidor (p. ej. el
// AssetServer de Wails); si no se indica, la fija Escuchar.
func ConBaseURL(u string) Option {
	return func(p *Proxy) { p.base = u }
}

// Proxy es el proxy local de streaming. Cero valor no es válido: usa
// NewProxy.
type Proxy struct {
	// doer es con el que se reenvían las peticiones. Si se construyó el
	// cliente propio, cliente y doer apuntan al mismo *http.Client.
	doer    httpx.HTTPDoer
	cliente *http.Client // nil si el doer lo ha inyectado el llamador

	mu       sync.Mutex
	base     string
	ln       net.Listener
	srv      *http.Server
	sesiones *sesiones

	permitirRedPrivada bool
}

// NewProxy crea el proxy local. doer es el HTTPDoer con el que se reenvían
// las peticiones: si es nil, el proxy construye su propio http.Client con
// anti-SSRF (lo que hace la app). Si se pasa un doer, se usa tal cual y la
// seguridad del destino queda en manos del llamador, porque el paquete no
// puede meter su control de red en un transporte ajeno.
func NewProxy(doer httpx.HTTPDoer, opts ...Option) *Proxy {
	clientePropio := doer == nil
	p := &Proxy{
		doer:     doer,
		sesiones: nuevasSesiones(TTLDefecto, LimiteDefecto),
	}
	for _, o := range opts {
		o(p)
	}
	if clientePropio {
		p.cliente = nuevoCliente(p.permitirRedPrivada)
		p.doer = p.cliente
	}
	return p
}

// Escuchar enlaza el proxy en 127.0.0.1 con un puerto aleatorio, lo sirve en
// un goroutine y devuelve la URL base (p. ej. http://127.0.0.1:41234) que
// Register necesita para construir las URLs. Si ya escuchaba, devuelve la
// misma base.
func (p *Proxy) Escuchar() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ln != nil {
		return p.base, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	srv := &http.Server{Handler: p.Handler()}
	p.ln, p.srv = ln, srv
	if p.base == "" {
		p.base = "http://" + ln.Addr().String()
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	return p.base, nil
}

// Cerrar detiene el servidor iniciado por Escuchar. Con el Handler montado
// en otro servidor no hace nada.
func (p *Proxy) Cerrar() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.srv == nil {
		return nil
	}
	err := p.srv.Close()
	p.srv, p.ln = nil, nil
	return err
}

// Handler devuelve el http.Handler del proxy: montable en el AssetServer de
// Wails o servido por Escuchar. Aplica CORS a TODAS las respuestas y
// responde él mismo los preflight OPTIONS (el upstream devuelve 405).
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aplicarCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path != "/proxy" {
			http.NotFound(w, r)
			return
		}
		p.reenviar(w, r)
	})
}

// Register asocia las cabeceras requeridas de la fuente (Referer y Headers)
// a una sesión nueva con TTL y devuelve la misma fuente con las URLs del
// recurso y de los subtítulos apuntando al proxy. Solo viajan por la URL el
// id de sesión y la URL de destino codificada: las cabeceras jamás.
//
// Requiere que antes se haya llamado a Escuchar (o ConBaseURL); sin base no
// hay dónde apuntar y se devuelve la fuente sin registrar.
func (p *Proxy) Register(source models.VideoSource) (proxiedSource models.VideoSource) {
	if source.URL == "" {
		return source
	}
	u, err := url.Parse(source.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		logx.Request("stream", "register", urlSegura(u), http.StatusBadRequest, 0, "destino")
		return source
	}
	base := p.baseURL()
	if base == "" {
		logx.Request("stream", "register", urlSegura(u), http.StatusBadRequest, 0, "sin_base")
		return source
	}
	id := p.sesiones.crear(source.Referer, source.Headers)

	proxiedSource = source
	proxiedSource.URL = p.urlProxy(id, source.URL)
	if len(source.Subtitles) > 0 {
		subs := make([]models.Subtitle, len(source.Subtitles))
		for i, sub := range source.Subtitles {
			subs[i] = sub
			if sub.URL == "" {
				continue
			}
			if su, err := url.Parse(sub.URL); err == nil && su.Host != "" &&
				(su.Scheme == "http" || su.Scheme == "https") {
				subs[i].URL = p.urlProxy(id, sub.URL)
			}
		}
		proxiedSource.Subtitles = subs
	}
	return proxiedSource
}

// urlProxy construye la URL del proxy para el destino dado.
func (p *Proxy) urlProxy(id, destino string) string {
	q := url.Values{}
	q.Set("s", id)
	q.Set("u", destino)
	return p.baseURL() + "/proxy?" + q.Encode()
}

func (p *Proxy) baseURL() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.base
}

// aplicarCORS escribe las cabeceras CORS en TODAS las respuestas del proxy
// y, si es un preflight, las de método/cabeceras permitidas. El upstream no
// sirve para esto: se comprobó en el sondeo que su OPTIONS devuelve 405.
func aplicarCORS(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Expose-Headers", "Content-Range, Accept-Ranges, Content-Length")
	if r.Method != http.MethodOptions {
		return
	}
	h.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	// El preflight por Range pide estas cabeceras; si el navegador pide
	// otras, se le devuelven tal cual (no viajan credenciales).
	if pedidas := r.Header.Get("Access-Control-Request-Headers"); pedidas != "" {
		h.Set("Access-Control-Allow-Headers", pedidas)
		return
	}
	h.Set("Access-Control-Allow-Headers", "Range, If-Range, Accept, Accept-Language, Origin, Content-Type")
}

// urlSegura reduce una URL a esquema+host+ruta para los logs: las URLs de
// vídeo van firmadas (query con la firma temporal) y no se registran
// completas. Los parámetros firmados no los cubre logx.Redact.
func urlSegura(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + u.Host + u.Path
}
