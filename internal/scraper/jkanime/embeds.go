package jkanime

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"miapp/internal/logx"
	"miapp/internal/models"
	"miapp/internal/scraper/embeds"
)

// tiempoResolucionEmbeds acota cada resolución de embed (T61): es una
// petición corta al HTML del embed, no la descarga del vídeo. Un embed
// lento no puede bloquear el resto de fuentes del episodio.
const tiempoResolucionEmbeds = 8 * time.Second

// embedsResolvers devuelve los resolvers de embeds inyectados (New deja
// uno de MP4Upload con el doer del proveedor; una lista distinta permite a
// los tests inyectar un resolver falso o desactivar el enriquecimiento con
// una lista vacía).
func (j *JkAnime) embedsResolvers() []embeds.Resolver {
	return j.Embeds
}

// enriquecerConEmbeds recorre las fuentes ya extraídas y, por cada iframe
// que algún resolver reconozca (T61), resuelve el embed y añade sus
// fuentes directas (mp4) **conservando el iframe** como respaldo. Un
// fallo del resolver nunca rompe el resto: se registra un aviso sin URL y
// se sigue con las demás fuentes. No se duplica un recurso que ya esté en
// la lista (mismo esquema+host+ruta, sin query).
func (j *JkAnime) enriquecerConEmbeds(ctx context.Context, fuentes []models.VideoSource) []models.VideoSource {
	resolvers := j.embedsResolvers()
	if len(resolvers) == 0 || len(fuentes) == 0 {
		return fuentes
	}
	if ctx == nil {
		ctx = context.Background()
	}
	salida := append([]models.VideoSource(nil), fuentes...)
	vistas := make(map[string]struct{}, len(salida))
	for _, f := range salida {
		if clave := claveRecurso(f.URL); clave != "" {
			vistas[clave] = struct{}{}
		}
	}
	for _, base := range fuentes {
		if base.Type != models.ResourceIframe || strings.TrimSpace(base.URL) == "" {
			continue
		}
		resolver := resolverParaEmbeds(resolvers, base.URL)
		if resolver == nil {
			continue
		}
		resueltas, err := resolverConTimeout(ctx, resolver, base.URL)
		if err != nil {
			slog.Warn("jkanime: la resolución del embed falló; se conserva el iframe",
				"resolver", resolver.Name(),
				"server", base.Server,
				"err", logx.Redact(err.Error()))
			continue
		}
		for _, r := range resueltas {
			clave := claveRecurso(r.URL)
			if clave == "" {
				continue
			}
			if _, ya := vistas[clave]; ya {
				continue // mismo recurso con otra firma: no duplicar
			}
			vistas[clave] = struct{}{}
			salida = append(salida, r)
		}
	}
	return salida
}

// resolverParaEmbeds devuelve el primer resolver que reconozca la URL del
// iframe, o nil si ninguno lo hace. Comprobación barata y sin red.
func resolverParaEmbeds(resolvers []embeds.Resolver, url string) embeds.Resolver {
	for _, r := range resolvers {
		if r != nil && r.CanHandle(url) {
			return r
		}
	}
	return nil
}

// resolverConTimeout resuelve el embed con un tope corto, derivado del
// contexto del flujo: si el reproductor cancela, la petición al embed se
// corta igual. Referer vacío: el resolver no inventa la página que embebe
// y MP4Upload solo exige Referer del dominio para el CDN, no para el HTML.
func resolverConTimeout(ctx context.Context, r embeds.Resolver, url string) ([]models.VideoSource, error) {
	ctxRes, cancel := context.WithTimeout(ctx, tiempoResolucionEmbeds)
	defer cancel()
	return r.Resolve(ctxRes, url, "")
}

// claveRecurso identifica un recurso para deduplicar: esquema y host en
// minúsculas, sin la query (que solo cambia la firma temporal) y con el
// fragmento. Devuelve "" si la URL no es parseable o no tiene host.
func claveRecurso(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.User == nil {
		u.Host = strings.ToLower(u.Host)
	}
	u.RawQuery = ""
	return u.String()
}
