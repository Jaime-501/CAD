package logx

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRedactCookieYCsrf(t *testing.T) {
	in := "GET /api HTTP/1.1 Cookie: sid=SECRETO_COOKIE; Path=/ csrf=SECRETO_CSRF _token=SECRETO_TOKEN"
	out := Redact(in)
	for _, s := range []string{"SECRETO_COOKIE", "SECRETO_CSRF", "SECRETO_TOKEN"} {
		if strings.Contains(out, s) {
			t.Errorf("secreto %q visible en la salida: %q", s, out)
		}
	}
	if !strings.Contains(out, "***") {
		t.Errorf("no hay máscara en la salida: %q", out)
	}
}

func TestRedactURLConToken(t *testing.T) {
	in := "https://ejemplo.test/e?token=abc&other=1"
	out := Redact(in)
	if strings.Contains(out, "abc") {
		t.Errorf("el token sigue visible: %q", out)
	}
	if !strings.Contains(out, "token=***") {
		t.Errorf("falta token=***: %q", out)
	}
	if !strings.Contains(out, "other=1") {
		t.Errorf("parámetro no secreto alterado: %q", out)
	}
}

func TestRedactJSON(t *testing.T) {
	in := `{"cookie":"COOKIE_VAL","authorization":"Bearer AUTH_VAL","csrf_token":"CSRF_VAL","nombre":"naruto"}`
	out := Redact(in)
	for _, s := range []string{"COOKIE_VAL", "AUTH_VAL", "CSRF_VAL"} {
		if strings.Contains(out, s) {
			t.Errorf("secreto %q visible en la salida: %q", s, out)
		}
	}
	if !strings.Contains(out, "naruto") {
		t.Errorf("campo no secreto alterado: %q", out)
	}
	if !strings.Contains(out, `"nombre":"naruto"`) {
		t.Errorf("estructura JSON rota: %q", out)
	}
}

func TestHandlerRedactaMensajeYAtributos(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(NewHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	l.Info("fallo contra /api?token=SECRETO_URL",
		"cookie", "Cookie: sid=SECRETO_COOKIE",
		"url", "https://x.test/?csrf=SECRETO_CSRF",
		"op", "buscar",
	)
	out := buf.String()
	for _, s := range []string{"SECRETO_URL", "SECRETO_COOKIE", "SECRETO_CSRF"} {
		if strings.Contains(out, s) {
			t.Errorf("secreto %q visible en el log: %q", s, out)
		}
	}
	if !strings.Contains(out, "op=buscar") {
		t.Errorf("atributo no secreto alterado: %q", out)
	}
}

func TestNewDevUsaNivelDebug(t *testing.T) {
	var buf bytes.Buffer
	dev := New(true)
	SetDefault(slog.New(NewHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	defer SetDefault(dev)

	current().Debug("modo desarrollo")
	if !strings.Contains(buf.String(), "modo desarrollo") {
		t.Errorf("el logger no escribe en nivel debug: %q", buf.String())
	}
}

func TestRequestRedactaURL(t *testing.T) {
	var buf bytes.Buffer
	prev := current()
	SetDefault(slog.New(NewHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))))
	defer SetDefault(prev)

	Request("jkanime", "episode", "https://x.test/e?token=SECRETO_REQ", 200, 15*time.Millisecond, "jkplayer")
	out := buf.String()
	if strings.Contains(out, "SECRETO_REQ") {
		t.Errorf("el token de la URL sigue visible: %q", out)
	}
	if !strings.Contains(out, "provider=jkanime") || !strings.Contains(out, "extractor=jkplayer") {
		t.Errorf("faltan atributos de la petición: %q", out)
	}
}

// TestRedactURLFirmada: una URL que autentica el recurso (players de
// jkplayer, CDN con st=, S3/CloudFront) tampoco se registra completa
// (regla del proyecto; T29).
func TestRedactURLFirmada(t *testing.T) {
	casos := []struct {
		nombre, in, veido string
		secrets           []string
	}{
		{
			nombre: "player jkplayer",
			in:     "https://cdn.test/jkplayer/um?e=FIRMA_E&op=NzM1NjQ=&t=FIRMA_T",
			veido:  "?e=***&op=***&t=***",
			secrets: []string{
				"FIRMA_E", "FIRMA_T", "NzM1NjQ",
			},
		},
		{
			nombre:  "CDN con st",
			in:      "https://cdn.test/hls/playlist.m3u8?st=FIRMA_ST&other=1",
			veido:   "?st=***&other=1",
			secrets: []string{"FIRMA_ST"},
		},
		{
			nombre:  "S3/CloudFront",
			in:      "https://bucket.s3.test/v.mp4?X-Amz-Algorithm=AWS4&X-Amz-Signature=SECRETO_AMZ&X-Amz-Credential=CRED",
			veido:   "X-Amz-Algorithm=***&X-Amz-Signature=***&X-Amz-Credential=***",
			secrets: []string{"SECRETO_AMZ", "CRED", "AWS4"},
		},
		{
			nombre:  "Signature/Policy",
			in:      "https://dl.test/archivo?Signature=SECRETO_SIG&Expires=1700000000&Policy=POLITICA",
			veido:   "?Signature=***&Expires=***&Policy=***",
			secrets: []string{"SECRETO_SIG", "1700000000", "POLITICA"},
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			out := Redact(c.in)
			for _, s := range c.secrets {
				if strings.Contains(out, s) {
					t.Errorf("firma %q visible: %q", s, out)
				}
			}
			if !strings.Contains(out, c.veido) {
				t.Errorf("query = %q, quería ver %q", out, c.veido)
			}
		})
	}

	// Los parámetros que no firman nada siguen intactos.
	otros := Redact("https://jkanime.net/buscar/bleach?page=2&lang=es")
	if !strings.Contains(otros, "page=2&lang=es") {
		t.Errorf("parámetros inocentes alterados: %q", otros)
	}
}

// TestRedactPath: T62. El token de un medio puede ir en el PATH (no solo en
// la query), así que Redact enmascara los segmentos opacos conservando host,
// puerto y el último segmento con extensión; los slugs de anime y las
// queries de firma existentes no se rompen.
func TestRedactPath(t *testing.T) {
	casos := []struct {
		nombre  string
		in      string
		quiere  string // subcadena exacta que debe aparecer
		secrets []string
	}{
		{
			nombre:  "MP4Upload con TOKEN_DE_PRUEBA (fixture)",
			in:      "https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4",
			quiere:  "https://a2.mp4upload.com:183/d/***/video.mp4",
			secrets: []string{"TOKEN_DE_PRUEBA"},
		},
		{
			nombre:  "MP4Upload con token realista de 12 alfanuméricos",
			in:      "https://a2.mp4upload.com:183/d/Xq7bK2mNpR4t/video.mp4",
			quiere:  "https://a2.mp4upload.com:183/d/***/video.mp4",
			secrets: []string{"Xq7bK2mNpR4t"},
		},
		{
			nombre:  "MP4Upload con query de firma además del token",
			in:      "https://a2.mp4upload.com:183/d/TOKEN_DE_PRUEBA/video.mp4?st=FIRMA_ST&other=1",
			quiere:  "https://a2.mp4upload.com:183/d/***/video.mp4?st=***&other=1",
			secrets: []string{"TOKEN_DE_PRUEBA", "FIRMA_ST"},
		},
		{
			nombre:  "manifiesto HLS de zilla (/m3u8/<32 hex>)",
			in:      "https://player.zilla-networks.com/m3u8/65f33e9a44ff7767faf60d1dae9eb1e6",
			quiere:  "https://player.zilla-networks.com/m3u8/***",
			secrets: []string{"65f33e9a44ff7767faf60d1dae9eb1e6"},
		},
		{
			nombre: "segmento HLS real (base64) conserva el .ts",
			in:     "https://cdn5.ducvomes.com/jp/segs/1/v2x/azRGek01N1ZwOXFENll3RFRvL2dIUT09/TVB1LzB1Z2tlaWhvaWdsTnJBYXpmQT09/10v1x.ts",
			quiere: "https://cdn5.ducvomes.com/jp/segs/1/v2x/***/***/10v1x.ts",
			secrets: []string{
				"azRGek01N1ZwOXFENll3RFRvL2dIUT09",
				"TVB1LzB1Z2tlaWhvaWdsTnJBYXpmQT09",
			},
		},
		{
			nombre:  "segmento /segs/<id>/NNN.html (id opaco, número conservado)",
			in:      "https://cdn.example.test/segs/AbC123xY9zQ0wErT/000.html",
			quiere:  "https://cdn.example.test/segs/***/000.html",
			secrets: []string{"AbC123xY9zQ0wErT"},
		},
		{
			nombre: "slug corto de anime con número: NO se toca",
			in:     "https://jkanime.net/bleach/1/",
			quiere: "https://jkanime.net/bleach/1/",
		},
		{
			nombre: "slug largo con guiones y año: NO se toca",
			in:     "https://animeav1.com/media/demon-slayer-kimetsu-no-yaiba-2019/1/__data.json",
			quiere: "https://animeav1.com/media/demon-slayer-kimetsu-no-yaiba-2019/1/__data.json",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			out := Redact(c.in)
			for _, s := range c.secrets {
				if strings.Contains(out, s) {
					t.Errorf("secreto %q visible: %q", s, out)
				}
			}
			if !strings.Contains(out, c.quiere) {
				t.Errorf("Redact(%q) = %q; quería ver %q", c.in, out, c.quiere)
			}
		})
	}
}
