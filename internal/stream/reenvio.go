package stream

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"miapp/internal/httpx"
	"miapp/internal/logx"
)

// cabecerasCliente es la lista blanca de cabeceras del reproductor que se
// reenvían al origen. Cookie y Authorization no están (higiene: la única
// fuente de cabeceras es la sesión), tampoco Accept-Encoding (la gestiona el
// transporte, que además descomprime y así no hay que copiar Content-Length
// de una respuesta transcodificada) ni ninguna hop-by-hop.
var cabecerasCliente = []string{
	"Range",
	"If-Range",
	"Accept",
	"Accept-Language",
	"Origin",
	"If-None-Match",
	"If-Modified-Since",
}

// cabecerasRespuesta es la lista blanca de cabeceras del origen que se
// copian al reproductor. Las hop-by-hop (Connection, Keep-Alive,
// Transfer-Encoding, TE, Trailer, Upgrade, Proxy-*) no están y nunca se
// copian; Content-Length se calcula aparte. La cabecera Date tampoco: pone
// la propia respuesta.
var cabecerasRespuesta = []string{
	"Content-Type",
	"Content-Range",
	"Accept-Ranges",
	"ETag",
	"Last-Modified",
	"Cache-Control",
	"Content-Disposition",
}

// reenviar atiende /proxy?s=<sesión>&u=<url>: valida la sesión y el destino,
// reenvía con doer (sin cargar el cuerpo en memoria) y devuelve la
// respuesta con CORS. La petición upstream comparte contexto con la del
// reproductor: si cierra la conexión, se cancela el reenvío.
func (p *Proxy) reenviar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		logx.Request("stream", "proxy", r.URL.Path, http.StatusMethodNotAllowed, 0, "metodo")
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("s")
	ses := p.sesiones.obtener(id)
	if ses == nil {
		logx.Request("stream", "proxy", r.URL.Path, http.StatusNotFound, 0, "sesion")
		http.Error(w, "sesión no encontrada o caducada", http.StatusNotFound)
		return
	}

	u, err := url.Parse(q.Get("u"))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		logx.Request("stream", "proxy", r.URL.Path, http.StatusBadRequest, 0, "destino")
		http.Error(w, "url de destino no válida", http.StatusBadRequest)
		return
	}
	seguro := urlSegura(u)

	up, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), nil)
	if err != nil {
		logx.Request("stream", "proxy", seguro, http.StatusBadRequest, 0, "destino")
		http.Error(w, "url de destino no válida", http.StatusBadRequest)
		return
	}
	// 1) lista blanca del cliente (Range y variantes de validación).
	for _, k := range cabecerasCliente {
		if vs := r.Header.Values(k); len(vs) > 0 {
			up.Header[http.CanonicalHeaderKey(k)] = append([]string(nil), vs...)
		}
	}
	// 2) User-Agent de navegador salvo que la sesión fije el suyo.
	up.Header.Set("User-Agent", httpx.DefaultUserAgent)
	// 3) cabeceras de la sesión: mandan sobre cualquier cosa del cliente.
	for k, v := range ses.headers {
		up.Header.Set(k, v)
	}
	if ses.referer != "" {
		up.Header.Set("Referer", ses.referer)
	}

	inicio := time.Now()
	resp, err := p.doer.Do(up)
	dur := time.Since(inicio)
	if err != nil {
		if r.Context().Err() != nil {
			return // el reproductor ya se fue: nada que responder
		}
		estado := http.StatusBadGateway
		if errors.Is(err, ErrDestinoBloqueado) {
			estado = http.StatusForbidden
		}
		logx.Request("stream", "proxy", seguro, estado, dur, "reenvio")
		http.Error(w, "no se pudo contactar con el origen", estado)
		return
	}
	if resp == nil {
		logx.Request("stream", "proxy", seguro, http.StatusBadGateway, dur, "reenvio")
		http.Error(w, "el origen no respondió", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.Body == nil {
		resp.Body = http.NoBody
	}
	// No se reenvía nada delante de la respuesta: si algo falla al leer el
	// cuerpo, el error ya no se puede cambiar de estado.
	copiarCabeceras(w.Header(), resp.Header)
	if esSubtituloVTT(u, resp.Header.Get("Content-Type")) {
		w.Header().Set("Content-Type", "text/vtt")
	}
	manifiesto := esManifiesto(u, resp.Header.Get("Content-Type"))
	if manifiesto && resp.StatusCode == http.StatusOK {
		p.escribirManifiesto(w, resp, u, id, seguro, inicio)
		return
	}
	if resp.ContentLength >= 0 && !resp.Uncompressed && resp.Header.Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	}
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		// Cuerpo en streaming: si el reproductor corta la conexión, este
		// Copy falla, se cierra el cuerpo y el ctx cancela el upstream.
		if _, err := io.Copy(w, resp.Body); err != nil && r.Context().Err() == nil {
			logx.Request("stream", "proxy", seguro, resp.StatusCode, time.Since(inicio), "stream")
		}
	}
	logx.Request("stream", "proxy", seguro, resp.StatusCode, time.Since(inicio), tipoRecurso(u, manifiesto))
}

// escribirManifiesto lee como mucho maxManifiesto bytes, reescribe las URIs
// para que vuelvan a pasar por el proxy y responde con el cuerpo nuevo.
// Solo se reescribe un 200 con #EXTM3U: el resto (206, páginas de error…)
// se devolvería incompleto o roto.
func (p *Proxy) escribirManifiesto(w http.ResponseWriter, resp *http.Response, u *url.URL, id, seguro string, inicio time.Time) {
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, maxManifiesto+1))
	if err != nil {
		logx.Request("stream", "proxy", seguro, http.StatusBadGateway, time.Since(inicio), "manifiesto")
		http.Error(w, "no se pudo leer el manifiesto", http.StatusBadGateway)
		return
	}
	if len(cuerpo) > maxManifiesto {
		logx.Request("stream", "proxy", seguro, http.StatusBadGateway, time.Since(inicio), "manifiesto")
		http.Error(w, "manifiesto demasiado grande", http.StatusBadGateway)
		return
	}
	if !esPlaylist(cuerpo) {
		w.Header().Set("Content-Length", strconv.Itoa(len(cuerpo)))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(cuerpo)
		logx.Request("stream", "proxy", seguro, resp.StatusCode, time.Since(inicio), "manifiesto")
		return
	}
	// De dónde salen las URIs relativas: la URL final tras los redirects.
	base := u
	if resp.Request != nil && resp.Request.URL != nil {
		base = resp.Request.URL
	}
	reescrito := reescribir(cuerpo, base, func(abs string) string { return p.urlProxy(id, abs) })
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(reescrito)))
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(reescrito)
	logx.Request("stream", "proxy", seguro, resp.StatusCode, time.Since(inicio), "manifiesto")
}

// copiarCabeceras copia la lista blanca de cabeceras del origen al
// reproductor. No se copia nada hop-by-hop ni Cookie/Authorization.
func copiarCabeceras(dst, src http.Header) {
	for _, k := range cabecerasRespuesta {
		if vs := src.Values(k); len(vs) > 0 {
			dst[k] = append([]string(nil), vs...)
		}
	}
}

// tipoRecurso etiqueta la línea de log según lo que se acabó de servir.
func tipoRecurso(u *url.URL, manifiesto bool) string {
	switch {
	case manifiesto:
		return "manifiesto"
	case esSubtituloVTT(u, ""):
		return "vtt"
	default:
		return "recurso"
	}
}
