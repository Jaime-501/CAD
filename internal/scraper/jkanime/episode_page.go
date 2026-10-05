package jkanime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

const (
	// maxEpisodePageBytes limita el HTML de la página de un episodio
	// (10 MB), igual que maxSearchBytes y maxDetailBytes.
	maxEpisodePageBytes = 10 << 20

	// rutaPelicula es la URL canónica de una película: el sitio redirige
	// /<slug>/1/ con 302 hacia /<slug>/pelicula/ (verificado en vivo).
	rutaPelicula = "pelicula"
)

// EpisodePage es la página cruda de un episodio tal como la devolvió el
// servidor, sin procesar: la descarga la separa de la extracción del vídeo
// (T12–T14). Solo lectura; no se cachea.
type EpisodePage struct {
	// URL final del episodio, después de las redirecciones del sitio.
	URL string
	// HTML crudo del cuerpo de la respuesta.
	HTML string
	// Headers de la respuesta final, sin Set-Cookie: las cookies de sesión
	// viven solo en el jar del cliente (regla 10) y nunca salen de aquí.
	Headers http.Header
}

// fetchEpisodePage descarga la página del episodio identificado por
// episodeID (p. ej. "jkanime:bleach:episode:1") usando el mismo cliente y
// cookiejar que el resto del proveedor. Aplica el User-Agent por defecto y
// envía como Referer la página de la serie, de la que suele llegar la
// navegación. No extrae fuentes de vídeo: eso es trabajo de los extractores.
//
// Los errores HTTP llegan tal cual de httpx (404 ErrNotFound, 403
// ErrForbidden, 429 ErrTooManyRequests, 5xx ErrServerError y
// ErrCloudflareChallenge con cualquier status). Un id inválido no toca la red.
func (j *JkAnime) fetchEpisodePage(ctx context.Context, episodeID string) (EpisodePage, error) {
	base := j.siteURL()
	proveedor, slug, tipo, numero, err := models.ParseEpisodeID(episodeID)
	if err != nil {
		return EpisodePage{}, err
	}
	if proveedor != providerName {
		return EpisodePage{}, fmt.Errorf("jkanime: el id %q pertenece al proveedor %q", episodeID, proveedor)
	}
	ruta, err := rutaEpisodio(slug, tipo, numero)
	if err != nil {
		return EpisodePage{}, err
	}

	resp, err := httpx.Get(ctx, j.Doer, base+ruta, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "episode_page",
		Referer:   base + "/" + url.PathEscape(slug) + "/",
	})
	if err != nil {
		return EpisodePage{}, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxEpisodePageBytes+1))
	if err != nil {
		return EpisodePage{}, err
	}
	if len(data) > maxEpisodePageBytes {
		return EpisodePage{}, fmt.Errorf("la página del episodio pesa %d bytes y el máximo es %d", len(data), maxEpisodePageBytes)
	}

	return EpisodePage{
		URL:     finalURL(resp, base+ruta),
		HTML:    string(data),
		Headers: cabecerasDePagina(resp.Header),
	}, nil
}

// rutaEpisodio traduce slug+tipo+número a la ruta real del sitio:
// /<slug>/<n>/ para episodios y especiales y /<slug>/pelicula/ para
// películas (que no llevan número). Un episodio con número < 1 es un error.
func rutaEpisodio(slug string, tipo models.EpisodeKind, numero int) (string, error) {
	escapado := url.PathEscape(slug)
	if tipo == models.KindMovie {
		return "/" + escapado + "/" + rutaPelicula + "/", nil
	}
	if numero < 1 {
		return "", fmt.Errorf("jkanime: número de episodio inválido %d para %q", numero, slug)
	}
	return "/" + escapado + "/" + strconv.Itoa(numero) + "/", nil
}

// finalURL devuelve la URL definitiva de la respuesta (tras seguir las
// redirecciones del sitio); sin request asociada, la que se pidió.
func finalURL(resp *http.Response, pedida string) string {
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String()
	}
	return pedida
}

// cabecerasDePagina copia las cabeceras de la respuesta quitando
// Set-Cookie: la sesión se mantiene en el cookiejar del cliente y estos
// valores no deben viajar en structs ni llegar a logs (regla 10).
func cabecerasDePagina(h http.Header) http.Header {
	copia := h.Clone()
	if copia != nil {
		copia.Del("Set-Cookie")
	}
	return copia
}
