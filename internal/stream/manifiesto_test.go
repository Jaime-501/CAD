package stream

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"miapp/internal/models"
)

// prefijoFalso es la base de las URLs del proxy en los tests unitarios.
const prefijoFalso = "http://127.0.0.1:9/proxy?s=SID&u="

func proxificarFalso(abs string) string {
	return prefijoFalso + url.QueryEscape(abs)
}

// TestReescribirEtiquetasYRelativas cubre lo que pide T19: segmentos,
// #EXT-X-KEY, #EXT-X-MAP, sub-playlists, URLs relativas y absolutas. El
// fixture es artificial (regla 11): el manifiesto real de bleach no trae
// claves, mapas ni URIs relativas (ver desviación en PROGRESO.md).
func TestReescribirEtiquetasYRelativas(t *testing.T) {
	cuerpo, err := os.ReadFile("testdata/manifiesto_etiquetas.m3u8")
	if err != nil {
		t.Fatalf("leyendo fixture: %v", err)
	}
	base, err := url.Parse("https://origen.example.test/hls/master.m3u8")
	if err != nil {
		t.Fatalf("base: %v", err)
	}

	reescrito := string(reescribir(cuerpo, base, proxificarFalso))

	queria := []string{
		// atributos URI= de los tags (con sus comillas y el resto intactos)
		`URI="` + prefijoFalso + url.QueryEscape("https://origen.example.test/hls/audio/es.m3u8") + `"`,
		`URI="` + prefijoFalso + url.QueryEscape("https://claves.example.test/k.bin") + `",IV=0x9f7a2c1d0e5b4a392817065544332211`,
		`URI="` + prefijoFalso + url.QueryEscape("https://origen.example.test/hls/pistas/init.mp4") + `"`,
		// líneas de URI: sub-playlist relativa, relativa con .., ruta
		// absoluta, relativa simple y absoluta de otro host
		prefijoFalso + url.QueryEscape("https://origen.example.test/hls/variantes/720.m3u8"),
		prefijoFalso + url.QueryEscape("https://origen.example.test/segmentos/uno.ts"),
		prefijoFalso + url.QueryEscape("https://origen.example.test/absoluto/dos.ts"),
		prefijoFalso + url.QueryEscape("https://origen.example.test/hls/tres.ts"),
		prefijoFalso + url.QueryEscape("https://cdn4.ducvomes.com/jp/segs/cuatro.ts"),
		// lo que no es http/https queda intacto
		"data:text/plain,esto-no-se-toca",
		"#EXT-X-ENDLIST",
	}
	for _, esperado := range queria {
		if !strings.Contains(reescrito, esperado) {
			t.Errorf("falta en el manifiesto reescrito:\n  %s", esperado)
		}
	}
	noQueria := []string{
		"\n../segmentos/uno.ts",
		"\n/absoluto/dos.ts",
		"\ntres.ts",
		"\nvariantes/720.m3u8",
		"\nhttps://cdn4.ducvomes.com/jp/segs/cuatro.ts",
	}
	for _, basal := range noQueria {
		if strings.Contains(reescrito, basal) {
			t.Errorf("quedó una URI sin reescribir: %q", basal)
		}
	}
}

// TestReescribirManifiestoReal usa el manifiesto capturado en el sondeo:
// 353 segmentos absolutos repartidos en 6 hosts. Las líneas de tag no se
// tocan y todas las de URI pasan por el proxy.
func TestReescribirManifiestoReal(t *testing.T) {
	cuerpo, err := os.ReadFile("testdata/manifiesto.m3u8")
	if err != nil {
		t.Fatalf("leyendo fixture: %v", err)
	}
	base, err := url.Parse("https://nika.playmudos.com/beach/playlist.m3u8")
	if err != nil {
		t.Fatalf("base: %v", err)
	}

	salida := reescribir(cuerpo, base, proxificarFalso)
	antes, despues := bytes.Split(cuerpo, []byte("\n")), bytes.Split(salida, []byte("\n"))
	if len(antes) != len(despues) {
		t.Fatalf("líneas: %d antes, %d después", len(antes), len(despues))
	}

	segmentos := 0
	for i := range antes {
		linea := string(despues[i])
		if len(antes[i]) > 0 && antes[i][0] == '#' {
			if linea != string(antes[i]) {
				t.Errorf("tag modificado: %q → %q", antes[i], despues[i])
			}
			continue
		}
		if strings.TrimSpace(linea) == "" {
			if string(antes[i]) != linea {
				t.Errorf("línea vacía modificada")
			}
			continue
		}
		segmentos++
		if !strings.HasPrefix(linea, prefijoFalso) {
			t.Errorf("segmento sin reescribir: %q", linea)
		}
	}
	if segmentos != 353 {
		t.Errorf("segmentos reescritos = %d, el fixture trae 353", segmentos)
	}

	// Y la URL decodificada vuelve a ser el segmento original del CDN.
	primero := ""
	for _, l := range despues {
		if strings.HasPrefix(string(l), prefijoFalso) {
			primero = string(l)
			break
		}
	}
	if primero == "" {
		t.Fatal("no hay ninguna línea reescrita")
	}
	linea, err := url.Parse(primero)
	if err != nil {
		t.Fatalf("línea del proxy ilegible: %v", err)
	}
	destino, err := url.Parse(linea.Query().Get("u"))
	if err != nil {
		t.Fatalf("u ilegible: %v", err)
	}
	if !strings.HasPrefix(destino.Host, "cdn") {
		t.Errorf("destino decodificado = %q, debía seguir apuntando al CDN", destino.Host)
	}
}

// TestManifiestoEndToEnd: el .m3u8 pasa por el proxy con el Referer de la
// sesión, sale reescrito, con su Content-Type y con CORS.
func TestManifiestoEndToEnd(t *testing.T) {
	fixture, err := os.ReadFile("testdata/manifiesto.m3u8")
	if err != nil {
		t.Fatalf("leyendo fixture: %v", err)
	}
	referer := make(chan string, 1)
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case referer <- r.Header.Get("Referer"):
		default:
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = w.Write(fixture)
	})

	p, dir := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{
		URL:     up.URL + "/hls/playlist.m3u8",
		Type:    models.ResourceHLS,
		Referer: "https://jkanime.net/jkplayer/um?e=E_PRUEBA",
	})
	sesion := obtenerSesion(t, reg.URL)

	resp := pedir(t, reg.URL, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, quería 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/vnd.apple.mpegurl" {
		t.Errorf("Content-Type = %q", got)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("falta CORS en el manifiesto")
	}
	if got := <-referer; got != "https://jkanime.net/jkplayer/um?e=E_PRUEBA" {
		t.Errorf("Referer upstream = %q", got)
	}

	cuerpoRespuesta := []byte(cuerpo(t, resp))
	if got := resp.Header.Get("Content-Length"); got != strconv.Itoa(len(cuerpoRespuesta)) {
		t.Errorf("Content-Length = %q, quería %d", got, len(cuerpoRespuesta))
	}
	if bytes.Equal(cuerpoRespuesta, fixture) {
		t.Error("el manifiesto salió idéntico: no se reescribieron las URIs")
	}

	prefijo := dir + "/proxy?s=" + url.QueryEscape(sesion) + "&u="
	segmentos := 0
	for _, linea := range bytes.Split(cuerpoRespuesta, []byte("\n")) {
		if len(linea) == 0 || linea[0] == '#' {
			continue
		}
		segmentos++
		if !strings.HasPrefix(string(linea), prefijo) {
			t.Fatalf("línea sin reescribir: %q", linea)
		}
	}
	if segmentos != 353 {
		t.Errorf("segmentos = %d, quería 353", segmentos)
	}
}

// TestManifiestoDemasiadoGrande: se limita la lectura (5 MB) y se responde
// error en vez de hinchar la memoria.
func TestManifiestoDemasiadoGrande(t *testing.T) {
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxManifiesto+1))
	})

	p, _ := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{URL: up.URL + "/gigante.m3u8", Type: models.ResourceHLS})

	resp := pedir(t, reg.URL, nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, quería 502", resp.StatusCode)
	}
}

// TestManifiestoQueNoEsPlaylist: un .m3u8 con cuerpo que no empieza por
// #EXTM3U (p. ej. una página de error) se devuelve tal cual, sin reescribir.
func TestManifiestoQueNoEsPlaylist(t *testing.T) {
	up := upstreamPrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>error</html>"))
	})

	p, _ := proxyDePrueba(t)
	reg := p.Register(models.VideoSource{URL: up.URL + "/roto.m3u8", Type: models.ResourceHLS})

	resp := pedir(t, reg.URL, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, quería 200", resp.StatusCode)
	}
	if got := cuerpo(t, resp); got != "<html>error</html>" {
		t.Errorf("cuerpo = %q, debía pasarse sin tocar", got)
	}
	if cl := resp.Header.Get("Content-Length"); cl != strconv.Itoa(len("<html>error</html>")) {
		t.Errorf("Content-Length = %q, quería %d", cl, len("<html>error</html>"))
	}
}
