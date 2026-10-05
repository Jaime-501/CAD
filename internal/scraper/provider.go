// Package scraper define la interfaz multi-proveedor y su registro.
// Cada sitio (JKAnime, AnimeFLV, …) vive en su propio subpaquete.
package scraper

import (
	"context"
	"errors"

	"miapp/internal/models"
)

// ErrNoDetails devuelven los providers que no ofrecen página de detalle
// (p. ej. AnimeFLV). El llamador lo comprueba con errors.Is para mostrar
// "Detalle no disponible para este proveedor".
var ErrNoDetails = errors.New("scraper: este proveedor no ofrece detalle de anime")

// SourceOpts son las opciones para pedir fuentes de vídeo.
// NoCache lo usa el reproductor para pedir una fuente nueva tras un error
// fatal (T27); las fuentes nunca se cachean (regla general 9), así que
// afecta solo a la re-resolución del extractor.
type SourceOpts struct {
	NoCache bool `json:"no_cache"`
}

// Provider es la interfaz que implementa cada sitio fuente de datos.
type Provider interface {
	// Name es el identificador estable del proveedor (p. ej. "jkanime").
	Name() string
	Search(ctx context.Context, query string) ([]models.Anime, error)
	GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error)
	GetVideoSources(ctx context.Context, episodeID string, opts SourceOpts) ([]models.VideoSource, error)
}

// EpisodeStreamer es la interfaz opcional de streaming de episodios (T09).
// Un provider que la implementa entrega el detalle y la primera página de
// episodios de inmediato; el resto se emite por onBatch hasta completar,
// error o cancelación de ctx.
type EpisodeStreamer interface {
	StreamAnimeDetails(ctx context.Context, id string,
		onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error)
}
