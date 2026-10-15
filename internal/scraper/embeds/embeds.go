// Package embeds resuelve el medio directo de embeds de terceros sin
// ejecutar JavaScript: descarga el HTML crudo del embed y extrae la URL del
// vídeo que el reproductor del host deja en claro en su configuración.
//
// Es un paquete neutro: solo depende de models, httpx y logx (regla de
// dependencias de tareas/00_REGLAS.md). No importa scraper/jkanime, ni
// jkanime/extractors, ni animeav1, de modo que ambos proveedores puedan
// consumirlo por igual cuando se cablee (tarea siguiente a T60).
//
// Las URLs que devuelve son temporales —el token suele ir embebido en el
// path o la query—: nunca se cachean ni se persisten (regla general 9) y
// tampoco se registran completas en logs (logx.Redact, regla general 10).
package embeds

import (
	"context"

	"miapp/internal/models"
)

// Resolver resuelve un embed de un host concreto a fuentes reproducibles
// directas. CanHandle debe ser barato y sin red; Resolve hace la petición al
// embed y devuelve las fuentes normalizadas (Type en minúsculas: hls, mp4,
// iframe). Un resolver no devuelve fuentes sin URL: si no encuentra el
// medio, devuelve un error tipado.
type Resolver interface {
	// Name identifica al resolver y etiqueta sus fuentes (VideoSource.Server).
	Name() string
	// CanHandle decide, sin red, si este resolver reconoce la URL del embed.
	CanHandle(url string) bool
	// Resolve descarga el embed y devuelve sus fuentes. referer es la página
	// desde la que se embebe (vacío si no se conoce): se envía a la petición
	// del embed si hace falta, nunca se inventa.
	Resolve(ctx context.Context, url, referer string) ([]models.VideoSource, error)
}
