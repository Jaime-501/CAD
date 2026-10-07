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
