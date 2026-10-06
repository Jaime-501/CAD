package extractors

import (
	"regexp"
	"strconv"
	"strings"

	"miapp/internal/models"
)

// reCalidad extrae el primer número de una etiqueta de calidad
// ("1080p", "720", "HD 480"...). Sin número no hay desempate.
var reCalidad = regexp.MustCompile(`\d+`)

// SelectBestSource elige la mejor fuente reproducible de la lista
// (T17), siguiendo el orden de preferencia del reproductor:
//
//	HLS directo > MP4 directo > iframe soportado > resto
//
// Los Iframe **nunca** se auto-seleccionan si existe HLS o MP4: solo
// quedan los primeros si no hay nada mejor. Dentro de un mismo rango
// gana la mayor `Quality` ("1080p" → 1080; sin número = 0) y, a
// igualdad, se conserva la primera de la lista —el orden canónico de la
// cadena de extractores—. Las fuentes sin URL se ignoran: no son
// reproducibles. Devuelve false cuando no queda ninguna.
func SelectBestSource(fuentes []models.VideoSource) (models.VideoSource, bool) {
	var (
		mejor      models.VideoSource
		hayMejor   bool
		rangoMejor int
		calMejor   int
	)
	for _, f := range fuentes {
		if strings.TrimSpace(f.URL) == "" {
			continue
		}
		rango, cal := rangoDeTipo(f.Type), calidadDe(f.Quality)
		if !hayMejor || rango > rangoMejor || (rango == rangoMejor && cal > calMejor) {
			mejor, hayMejor, rangoMejor, calMejor = f, true, rango, cal
		}
	}
	return mejor, hayMejor
}

// rangoDeTipo ordena los tipos por preferencia de reproducción: mayor
// gana. Un `Type` vacío o desconocido queda el último por detrás del
// iframe (T16 conserva esas fuentes; no hay señal de qué son).
func rangoDeTipo(t models.ResourceType) int {
	switch t {
	case models.ResourceHLS:
		return 4
	case models.ResourceMP4:
		return 3
	case models.ResourceIframe:
		return 2
	default:
		return 1
	}
}

// calidadDe convierte la etiqueta de calidad en un número comparable;
// sin número utilizable devuelve 0 (el desempate no actúa).
func calidadDe(quality string) int {
	digitos := reCalidad.FindString(quality)
	if digitos == "" {
		return 0
	}
	n, err := strconv.Atoi(digitos)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
