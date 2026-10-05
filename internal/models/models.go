// Package models define los modelos de dominio compartidos entre el backend,
// los providers del scraper y el frontend. Todos se serializan a JSON con
// etiquetas en minúsculas y snake_case (Wails usa encoding/json).
package models

// Anime es el resultado de una búsqueda (lo que ve la UI en la cuadrícula).
type Anime struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	CoverURL string `json:"cover_url"`
	Provider string `json:"provider"`
}

// EpisodeKind distingue entre episodio de serie, película y especial.
type EpisodeKind string

const (
	KindEpisode EpisodeKind = "episode"
	KindMovie   EpisodeKind = "movie"
	KindSpecial EpisodeKind = "special"
)

// AnimeDetail son los metadatos completos de una serie o película.
// Type se normaliza a series|movie|ova|special|unknown (T08).
type AnimeDetail struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Provider      string   `json:"provider"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
	Genres        []string `json:"genres"`
	Synopsis      string   `json:"synopsis"`
	Cover         string   `json:"cover"`
	TotalEpisodes int      `json:"total_episodes"`
}

// Episode es un episodio, película o especial de una serie.
// Para Kind==KindMovie el Number es 0: el identificador es el propio "movie".
type Episode struct {
	ID        string      `json:"id"`
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	Thumbnail string      `json:"thumbnail"`
	Kind      EpisodeKind `json:"kind"`
}

// ResourceType es el tipo de recurso de vídeo recibido del extractor.
type ResourceType string

const (
	ResourceHLS    ResourceType = "hls"
	ResourceMP4    ResourceType = "mp4"
	ResourceIframe ResourceType = "iframe"
)

// Subtitle es una pista de subtítulos asociada a un VideoSource.
type Subtitle struct {
	URL  string `json:"url"`
	Lang string `json:"lang"`
}

// VideoSource es una fuente reproducible. Las URLs son temporales:
// nunca se cachean ni se persisten (regla general 9).
type VideoSource struct {
	URL       string            `json:"url"`
	Type      ResourceType      `json:"type"`
	Quality   string            `json:"quality"`
	Server    string            `json:"server"`
	Headers   map[string]string `json:"headers"`
	Referer   string            `json:"referer"`
	Subtitles []Subtitle        `json:"subtitles,omitempty"`
}

// WatchProgress es el progreso de visualización de un episodio (T22).
type WatchProgress struct {
	AnimeID       string  `json:"anime_id"`
	EpisodeID     string  `json:"episode_id"`
	CurrentTime   float64 `json:"current_time"`
	TotalDuration float64 `json:"total_duration"`
	IsWatched     bool    `json:"is_watched"`
}

// EpisodeBatch es un lote de episodios emitido por streaming (T09/T20).
// Done marca el lote final, con o sin Err.
type EpisodeBatch struct {
	AnimeID  string    `json:"anime_id"`
	Batch    int       `json:"batch"`
	Episodes []Episode `json:"episodes"`
	Done     bool      `json:"done"`
	Err      string    `json:"err"`
}
