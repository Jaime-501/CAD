// Package logx proporciona logging estructurado con redacción de secretos.
//
// Regla del proyecto: nunca se registran en logs cookies, tokens CSRF ni
// cabeceras Authorization/Set-Cookie, y tampoco URLs firmadas completas
// (una query que autentica el recurso: ?e=/t=/st=/op= de los players o
// los X-Amz-* de S3). Todo lo que pase por este paquete (mensaje y
// atributos) se limpia con Redact antes de escribirse.
package logx

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"sync/atomic"
	"time"
)

// Patrones de redacción, en orden de aplicación.
var (
	// "cookie":"valor" / "csrf_token":"valor" (JSON, bodies)
	reJSON = regexp.MustCompile(`(?i)("(?:set-cookie|proxy-authorization|authorization|cookie|csrf(?:[-_]?token)?|xsrf[-_]?token|_token|access_token|token|api[-_]?key|apikey)"\s*:\s*")([^"]*)(")`)
	// ?token=abc&... en URLs
	reParam = regexp.MustCompile(`(?i)([?&](?:csrf(?:[-_]?token)?|xsrf[-_]?token|_token|access_token|token|api[-_]?key|apikey|auth|authorization)=)[^&\s"'<>]+`)
	// Cookie: ... / Authorization: ... en cabeceras o texto plano (resto de la línea)
	reLine = regexp.MustCompile(`(?i)\b(set-cookie|proxy-authorization|authorization|cookie)(\s*[:=]\s*)[^\r\n]+`)
	// csrf=..., _token=..., token=..., apikey=... sueltos
	reKV = regexp.MustCompile(`(?i)\b(csrf(?:[-_]?token)?|xsrf[-_]?token|_token|access_token|token|api[-_]?key|apikey|sid)(\s*[:=]\s*)([^\s,;&"'<>]+)`)
	// Firmas de recurso en la query: no son secretos de sesión, pero SÍ
	// autentican el vídeo, así que una URL firmada completa tampoco se
	// registra (T29: ?e=..&t=..&op=.. de jkplayer, ?st=.. del CDN,
	// ?Signature=.. y los X-Amz-*/awsaccesskeyid/policy de S3/CloudFront).
	reFirma = regexp.MustCompile(`(?i)([?&](?:e|t|st|op|key|sig|signature|sign|expires|expiry|policy|key-pair-id|awsaccesskeyid|x-amz-algorithm|x-amz-credential|x-amz-date|x-amz-expires|x-amz-signedheaders|x-amz-signature|x-amz-security-token|x-amz-key-pair-id)=)[^&\s"'<>]+`)
)

// Redact enmascara secretos en una cadena: cabeceras Cookie/Set-Cookie/
// Authorization, parámetros csrf/token/_token en URLs y cuerpos, sus
// variantes JSON y las URLs firmadas (query que autentica el recurso).
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = reJSON.ReplaceAllString(s, "${1}***${3}")
	s = reParam.ReplaceAllString(s, "${1}***")
	s = reLine.ReplaceAllString(s, "${1}${2}***")
	s = reKV.ReplaceAllString(s, "${1}${2}***")
	s = reFirma.ReplaceAllString(s, "${1}***")
	return s
}

func redactAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(Redact(a.Value.String()))
	}
	return a
}

// redactHandler es un slog.Handler envolvente que aplica Redact al mensaje y
// a cada atributo de la cadena de handlers.
type redactHandler struct {
	slog.Handler
}

// NewHandler envuelve cualquier handler con la redacción de secretos.
func NewHandler(h slog.Handler) slog.Handler {
	return redactHandler{Handler: h}
}

func (h redactHandler) Handle(ctx context.Context, r slog.Record) error {
	rec := slog.NewRecord(r.Time, r.Level, Redact(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		rec.AddAttrs(redactAttr(a))
		return true
	})
	return h.Handler.Handle(ctx, rec)
}

func (h redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	red := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		red[i] = redactAttr(a)
	}
	return redactHandler{Handler: h.Handler.WithAttrs(red)}
}

func (h redactHandler) WithGroup(name string) slog.Handler {
	return redactHandler{Handler: h.Handler.WithGroup(name)}
}

var defaultLogger atomic.Pointer[slog.Logger]

// SetDefault define el logger que usa Request.
func SetDefault(l *slog.Logger) {
	defaultLogger.Store(l)
}

func current() *slog.Logger {
	if l := defaultLogger.Load(); l != nil {
		return l
	}
	return New(false)
}

// New crea un logger estructurado con redacción de secretos.
// dev=true baja el nivel a debug y usa salida de texto; en producción se
// registra a nivel info con salida JSON. También lo instala como logger por
// defecto de este paquete (lo que usa Request).
func New(dev bool) *slog.Logger {
	level := slog.LevelInfo
	if dev {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	var base slog.Handler
	if dev {
		base = slog.NewTextHandler(os.Stderr, opts)
	} else {
		base = slog.NewJSONHandler(os.Stderr, opts)
	}
	l := slog.New(NewHandler(base))
	SetDefault(l)
	return l
}

// Request registra una petición HTTP del scraper: proveedor, operación, URL
// (redactada), código de estado, duración y extractor usado.
func Request(provider, op, url string, status int, dur time.Duration, extractor string) {
	level := slog.LevelInfo
	if status >= 400 || status == 0 {
		level = slog.LevelWarn
	}
	current().Log(context.Background(), level, "http_request",
		slog.String("provider", provider),
		slog.String("op", op),
		slog.String("url", url),
		slog.Int("status", status),
		slog.Duration("dur", dur),
		slog.String("extractor", extractor),
	)
}
