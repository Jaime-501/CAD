package extractors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// JKPHPExtractor extrae las fuentes del camino alternativo de jkanime
// basado en jk.php. Las páginas de episodio incrustan un iframe hacia
// jk.php?u=… —hoy servido por jkplayers.com, porque jkanime.net/jk.php
// responde 404— o hacia su equivalente actual jkplayer/jk?u=… (las dos
// formas están documentadas en PROGRESO.md). Las respuestas son una página
// DPlayer con el vídeo en la configuración video: { url: …, type: … }.
// Recibe su httpx.HTTPDoer por constructor (Base) para conservar cookies,
// User-Agent y sesión (reglas 3 y 4).
type JKPHPExtractor struct {
	Base
}

// reJkPHP localiza la URL del iframe en el HTML del episodio, en las dos
// formas reales —jk.php?u=… y jkplayer/jk?u=…— y con los mismos formatos
// que escribe el sitio: absoluto, protocolo-relativo (//host/…) o
// raíz-relativa (/jk.php?…).
var reJkPHP = regexp.MustCompile(`(?:https?:)?//[^\s"'<>\\]*(?:jk\.php|jkplayer/jk\?)[^\s"'<>\\]*|/[^\s"'<>\\]*(?:jk\.php|jkplayer/jk\?)[^\s"'<>\\]*`)

// Name identifica al extractor en la cadena (T11): es lo que hace que
// prioridad lo coloque en el puesto canónico 1 (JKPlayer → JKPHP →
// Servers), igual que JKPlayerExtractor con NameJKPlayer.
func (JKPHPExtractor) Name() string { return NameJKPHP }

// CanHandle reconoce la referencia al camino jk.php: la forma clásica
// jk.php?u=… y la que hoy emiten las páginas, jkplayer/jk?u=…. Es una
// comprobación barata sobre el HTML crudo, sin red.
func (JKPHPExtractor) CanHandle(page Page) bool {
	return strings.Contains(page.HTML, "jk.php") || strings.Contains(page.HTML, "jkplayer/jk?")
}

// Extract sigue cada iframe de jk.php, descarga su página y devuelve las
// fuentes reproducibles. Cada fuente fija como Referer la URL del propio
// iframe —mismo criterio que JKPlayerExtractor: es la página que emite el
// recurso— y Server NameJKPHP, para que el fallback del reproductor sepa
// de qué servidor vino. Los enlaces repetidos del mismo iframe no se
// piden dos veces y los que apuntan al mismo vídeo no salen duplicados.
func (e JKPHPExtractor) Extract(ctx context.Context, page Page) ([]models.VideoSource, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	enlaces := reJkPHP.FindAllString(page.HTML, -1)
	if len(enlaces) == 0 {
		return nil, fmt.Errorf("%s: el HTML menciona jk.php pero no contiene ninguna URL de iframe", NameJKPHP)
	}

	var (
		fuentes  []models.VideoSource
		errores  []error
		visitas  = map[string]bool{}
		producir = map[string]bool{}
	)
	for _, bruto := range enlaces {
		player, err := absolverURL(bruto, page.URL)
		if err != nil {
			errores = append(errores, fmt.Errorf("iframe de jk.php: %w", err))
			continue
		}
		if visitas[player.String()] {
			continue // misma URL de iframe: no repetir la petición
		}
		visitas[player.String()] = true

		producidas, err := e.fuentesDeJKPHP(ctx, page, player)
		if err != nil {
			errores = append(errores, err)
			continue
		}
		for _, f := range producidas {
			clave := f.URL
			if u, err := url.Parse(f.URL); err == nil {
				clave = recursoBase(u)
			}
			if producir[clave] {
				continue // mismo vídeo con otra firma: no duplicar
			}
			producir[clave] = true
			fuentes = append(fuentes, f)
		}
	}
	if len(fuentes) == 0 {
		if len(errores) > 0 {
			return nil, errors.Join(errores...)
		}
		return nil, fmt.Errorf("%s: sin URL de vídeo reproducible en la página del player", NameJKPHP)
	}
	return fuentes, nil
}

// fuentesDeJKPHP descarga la respuesta de un iframe de jk.php y traduce
// sus candidatas a VideoSource con los helpers de parseo de jkplayer.go
// (mismo paquete): urlsDeVideo, absolverURL, recursoBase, tipoDeRecurso,
// subtitulosDe y el límite maxPlayerBytes. Es el gemelo de
// JKPlayerExtractor.fuentesDePlayer: ese método no se reutiliza porque
// etiqueta con NameJKPlayer y jkplayer.go queda fuera de "Puedes tocar" en
// T13 (ver PROGRESO.md). La petición lleva el Referer de la página del
// episodio y, con el doer compartido, también sus cookies (regla 3).
func (e JKPHPExtractor) fuentesDeJKPHP(ctx context.Context, page Page, player *url.URL) ([]models.VideoSource, error) {
	resp, err := httpx.Get(ctx, e.Doer, player.String(), httpx.ReqOptions{
		Provider:  "jkanime",
		Operation: "jkphp",
		Extractor: NameJKPHP,
		Referer:   page.URL,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, maxPlayerBytes+1))
	if err != nil {
		return nil, err
	}
	if len(cuerpo) > maxPlayerBytes {
		return nil, fmt.Errorf("la página de jk.php pesa %d bytes y el máximo es %d", len(cuerpo), maxPlayerBytes)
	}

	subs := subtitulosDe(string(cuerpo), player)
	var fuentes []models.VideoSource
	visitas := map[string]bool{}
	for _, bruto := range urlsDeVideo(string(cuerpo)) {
		u, err := absolverURL(bruto, player.String())
		if err != nil {
			continue // javascript:/data:/basuras varias: no es reproducible
		}
		if visitas[recursoBase(u)] {
			continue
		}
		visitas[recursoBase(u)] = true
		fuentes = append(fuentes, models.VideoSource{
			URL:       u.String(),
			Type:      tipoDeRecurso(u, string(cuerpo)),
			Server:    NameJKPHP,
			Referer:   player.String(),
			Subtitles: subs,
		})
	}
	if len(fuentes) == 0 {
		return nil, fmt.Errorf("%s: la página no expone ninguna URL de vídeo válida", NameJKPHP)
	}
	return fuentes, nil
}
