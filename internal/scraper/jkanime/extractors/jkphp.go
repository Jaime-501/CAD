package extractors

import (
	"context"
	"regexp"
	"strings"

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
//
// T29: la implementación es la compartida con JKPlayerExtractor
// (extraeIframesDePlayer), que solo cambia las etiquetas y los textos.
func (e JKPHPExtractor) Extract(ctx context.Context, page Page) ([]models.VideoSource, error) {
	return extraeIframesDePlayer(ctx, page, reJkPHP, cfgJKPHP, e.Doer)
}

// cfgJKPHP es la parametrización de JKPHPExtractor para la descarga
// compartida de páginas de player (T29).
var cfgJKPHP = cfgPlayer{
	nombre:  NameJKPHP,
	mencion: "jk.php",
	pagina:  "de jk.php",
	sujeto:  "la página",
}
