// Package logx proporciona logging estructurado con redacción de secretos.
//
// Regla del proyecto: nunca se registran en logs cookies, tokens CSRF ni
// cabeceras Authorization/Set-Cookie, y tampoco URLs firmadas completas
// (una query que autentica el recurso: ?e=/t=/st=/op= de los players o
// los X-Amz-* de S3). Todo lo que pase por este paquete (mensaje y
// atributos) se limpia con Redact antes de escribirse.
//
// T62: además de la query, algunos CDN esconden el token en el PATH del
// medio (MP4Upload: /d/<token>/video.mp4; zilla: /m3u8/<32 hex>), así que
// Redact también enmascara los segmentos opacos de la ruta conservando
// host, puerto y el último segmento con extensión (el nombre del archivo,
// que es lo útil para el diagnóstico).
package logx

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
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
	// T62: token embebido en el PATH del medio (no en la query). reURLPath
	// aísla esquema+host y path de una URL absoluta; la query queda fuera
	// (la siguen cubriendo reParam/reFirma).
	reURLPath = regexp.MustCompile(`(?i)https?://[^\s/?#"'<>]+(?:/[^\s?#"'<>]*)?`)
	// reSegmentoLargo: blob sin separadores de palabra (hex, base64url, ids)
	// de 12+ caracteres con pinta de token. Los guiones quedan fuera a
	// propósito: separan las palabras de un slug ("demon-slayer-kimetsu-no-
	// yaiba-2019") y por eso no se enmascaran (el diagnóstico no se
	// destruye). `_`/`+`/`=` sí (TOKEN_DE_PRUEBA, padding base64). El umbral
	// es 12, no 16: los ids reales de MP4Upload son de 12 y el token de
	// prueba mide 15.
	reSegmentoLargo = regexp.MustCompile(`^[A-Za-z0-9_+=]{12,}$`)
	// reSegmentoAlnum: segmento alfanumérico puro, para ids cortos que
	// mezclan letras y dígitos sin separadores (el token de 12 de MP4Upload).
	reSegmentoAlnum = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

// Redact enmascara secretos en una cadena: cabeceras Cookie/Set-Cookie/
// Authorization, parámetros csrf/token/_token en URLs y cuerpos, sus
// variantes JSON, las URLs firmadas (query que autentica el recurso) y los
// segmentos opacos del path de una URL (token de un MP4/HLS escondido en la
// ruta).
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = reJSON.ReplaceAllString(s, "${1}***${3}")
	s = reParam.ReplaceAllString(s, "${1}***")
	s = reLine.ReplaceAllString(s, "${1}${2}***")
	s = reKV.ReplaceAllString(s, "${1}${2}***")
	s = reFirma.ReplaceAllString(s, "${1}***")
	s = redactarPath(s)
	return s
}

// redactarPath enmascara los segmentos opacos del path de cada URL absoluta
// que aparezca en s (T62). Conserva esquema, host, puerto, la query y el
// último segmento con extensión; solo cambia los segmentos intermedios que
// parecen un token/identificador (no los slugs de un anime).
func redactarPath(s string) string {
	return reURLPath.ReplaceAllStringFunc(s, func(u string) string {
		i := strings.Index(u, "://")
		if i < 0 {
			return u
		}
		resto := u[i+len("://"):]
		j := strings.IndexByte(resto, '/')
		if j < 0 {
			return u // sin path: nada que redactar
		}
		return u[:i+len("://")+j] + redactarSegmentos(resto[j:])
	})
}

// redactarSegmentos sustituye por *** los segmentos opacos de un path.
func redactarSegmentos(path string) string {
	partes := strings.Split(path, "/")
	for i, p := range partes {
		if p != "" && segmentoOpaco(p) {
			partes[i] = "***"
		}
	}
	return strings.Join(partes, "/")
}

// segmentoOpaco decide si un segmento del path es un secreto (token, hash,
// id) que no debe registrarse. Un nombre de archivo con extensión se
// conserva siempre: es lo útil para el diagnóstico ("video.mp4", "10v1x.ts").
func segmentoOpaco(s string) bool {
	if strings.ContainsRune(s, '.') {
		return false
	}
	// Blob sin separadores de palabra (hex/base64/ids) con pinta de token.
	if reSegmentoLargo.MatchString(s) {
		return true
	}
	// Id corto alfanumérico que mezcla letras y dígitos (token de MP4Upload).
	if len(s) >= 10 && reSegmentoAlnum.MatchString(s) && mezclaLetrasYDigitos(s) {
		return true
	}
	return false
}

// mezclaLetrasYDigitos indica si s contiene al menos una letra y un dígito.
func mezclaLetrasYDigitos(s string) bool {
	var letra, digito bool
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digito = true
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			letra = true
		}
		if letra && digito {
			return true
		}
	}
	return false
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
