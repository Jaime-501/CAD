package embeds

import (
	"context"
	"errors"
	"strings"
	"testing"

	"miapp/internal/httpx"
)

func TestMP4UploadCanHandle(t *testing.T) {
	casos := []struct {
		nombre string
		url    string
		quiere bool
	}{
		{"embed con www", "https://www.mp4upload.com/embed-i8l69rmmcvhi.html", true},
		{"embed sin www", "https://mp4upload.com/embed-i8l69rmmcvhi.html", true},
		{"forma directa con www", "https://www.mp4upload.com/i8l69rmmcvhi", true},
		{"forma directa sin www", "https://mp4upload.com/i8l69rmmcvhi", true},
		{"mayúsculas", "HTTPS://WWW.MP4UPLOAD.COM/EMBED-I8L69RMMCVHI.HTML", true},
		{"embed con query", "https://www.mp4upload.com/embed-i8l69rmmcvhi.html?x=1", true},
		{"otro host", "https://voe.sx/e/mwpqlc2nfdwp", false},
		{"cdn del vídeo", "https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4", false},
		{"recurso interno", "https://www.mp4upload.com/player/videojs/video.min.js", false},
		{"sin esquema", "//www.mp4upload.com/embed-i8l69rmmcvhi.html", false},
		{"texto suelto", "no-es-una-url", false},
		{"vacío", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := (MP4Upload{}).CanHandle(c.url); got != c.quiere {
				t.Errorf("CanHandle(%q) = %v, quería %v", c.url, got, c.quiere)
			}
		})
	}
}

func TestMP4UploadResuelveFixtureReal(t *testing.T) {
	srv := nuevoServidorEmbed(t, 0, cargarFixture(t))
	r := NewMP4Upload(srv.Client())

	// CanHandle está cubierto en la tabla (hosts reales de MP4Upload);
	// Resolve no exige ese host, así que se sirve el fixture en local.
	embed := srv.URL + "/embed-i8l69rmmcvhi.html"
	fuentes, err := r.Resolve(context.Background(), embed, "")
	if err != nil {
		t.Fatalf("Resolve falló: %v", err)
	}
	comprobarFuente(t, fuentes, embed)
	if srv.contar() != 1 {
		t.Errorf("peticiones al embed = %d, quería 1", srv.contar())
	}
}

func TestMP4UploadResuelveFormaDirecta(t *testing.T) {
	srv := nuevoServidorEmbed(t, 0, cargarFixture(t))
	r := NewMP4Upload(srv.Client())

	directa := srv.URL + "/i8l69rmmcvhi"
	fuentes, err := r.Resolve(context.Background(), directa, "")
	if err != nil {
		t.Fatalf("Resolve falló: %v", err)
	}
	comprobarFuente(t, fuentes, directa)
}

func TestMP4UploadEnviaReferer(t *testing.T) {
	srv := nuevoServidorEmbed(t, 0, cargarFixture(t))
	r := NewMP4Upload(srv.Client())

	const referer = "https://jkanime.net/bleach-sennen-kessen-hen/1/"
	if _, err := r.Resolve(context.Background(), srv.URL+"/embed-i8l69rmmcvhi.html", referer); err != nil {
		t.Fatalf("Resolve falló: %v", err)
	}
	req := srv.ultimaPeticion()
	if req == nil {
		t.Fatal("no llegó ninguna petición al embed")
	}
	if got := req.Header.Get("Referer"); got != referer {
		t.Errorf("Referer enviado = %q, quería %q", got, referer)
	}
	if got := req.Header.Get("User-Agent"); got == "" {
		t.Error("User-Agent vacío: httpx debía poner el suyo por defecto")
	}
}

func TestMP4UploadEmbedInexistente(t *testing.T) {
	srv := nuevoServidorEmbed(t, 404, "")
	r := NewMP4Upload(srv.Client())

	_, err := r.Resolve(context.Background(), srv.URL+"/embed-borrado.html", "")
	if !errors.Is(err, httpx.ErrNotFound) {
		t.Fatalf("error = %v, quería errors.Is(err, httpx.ErrNotFound)", err)
	}
}

func TestMP4UploadErrorServidor(t *testing.T) {
	srv := nuevoServidorEmbed(t, 500, "")
	r := NewMP4Upload(srv.Client())

	_, err := r.Resolve(context.Background(), srv.URL+"/embed-i8l69rmmcvhi.html", "")
	if !errors.Is(err, httpx.ErrServerError) {
		t.Fatalf("error = %v, quería errors.Is(err, httpx.ErrServerError)", err)
	}
}

func TestMP4UploadPaginaSinMedio(t *testing.T) {
	// HTML artificial: un embed degradado sin el setup player.src. El sitio
	// real siempre lo trae, así que este caso no tiene fixture propio.
	const html = `<!DOCTYPE html><html><body><video id="player" class="video-js"></video></body></html>`
	srv := nuevoServidorEmbed(t, 0, html)
	r := NewMP4Upload(srv.Client())

	_, err := r.Resolve(context.Background(), srv.URL+"/embed-i8l69rmmcvhi.html", "")
	if !errors.Is(err, ErrSinMedio) {
		t.Fatalf("error = %v, quería errors.Is(err, ErrSinMedio)", err)
	}
}

func TestMP4UploadContextoCancelado(t *testing.T) {
	srv := nuevoServidorEmbed(t, 0, cargarFixture(t))
	r := NewMP4Upload(srv.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Resolve(ctx, srv.URL+"/embed-i8l69rmmcvhi.html", "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, quería errors.Is(err, context.Canceled)", err)
	}
}

func TestMP4UploadCuerpoEnorme(t *testing.T) {
	srv := nuevoServidorEmbed(t, 0, strings.Repeat("x", maxEmbedBytes+1))
	r := NewMP4Upload(srv.Client())

	_, err := r.Resolve(context.Background(), srv.URL+"/embed-i8l69rmmcvhi.html", "")
	if !errors.Is(err, ErrCuerpoGrande) {
		t.Fatalf("error = %v, quería errors.Is(err, ErrCuerpoGrande)", err)
	}
}

func TestMP4UploadSinDoer(t *testing.T) {
	r := NewMP4Upload(nil)
	_, err := r.Resolve(context.Background(), "https://www.mp4upload.com/embed-i8l69rmmcvhi.html", "")
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Fatalf("error = %v, quería errors.Is(err, httpx.ErrNilDoer)", err)
	}
}

func TestMP4UploadURLInvalida(t *testing.T) {
	srv := nuevoServidorEmbed(t, 0, cargarFixture(t))
	r := NewMP4Upload(srv.Client())

	_, err := r.Resolve(context.Background(), "no-es-una-url", "")
	if !errors.Is(err, ErrURLInvalida) {
		t.Fatalf("error = %v, quería errors.Is(err, ErrURLInvalida)", err)
	}
	if srv.contar() != 0 {
		t.Errorf("peticiones = %d, una URL inválida no debe pedir nada", srv.contar())
	}
}
