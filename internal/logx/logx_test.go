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
