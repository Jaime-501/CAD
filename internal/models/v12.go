package models

import "strings"

// Favorite es un anime marcado con la estrella de la vista de detalle y
// listado en la pestaña "Favoritos" del Home (T42/T54). La clave es
// proveedor + ":" + anime_id: el slug se comparte entre proveedores.
type Favorite struct {
	Provider string `json:"provider"`
	AnimeID  string `json:"anime_id"`
	Title    string `json:"title"`
	CoverURL string `json:"cover_url"`
	AddedAt  int64  `json:"added_at"`
}

// Key devuelve la clave única del favorito (proveedor + ":" + anime_id).
func (f Favorite) Key() string {
	return f.Provider + ":" + f.AnimeID
}

// HomeSection es una fila horizontal del Home ("En emisión", "Recién
// agregados"…); lo que ofrece cada sitio lo decide su provider (T43/T44).
type HomeSection struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Items []Anime `json:"items"`
}

// Acciones posibles de ResumeInfo: qué hace el clic de "retomar" según el
// último registro del anime (tabla del plan v1.2).
const (
	ResumeActionResume    = "resume"
	ResumeActionNext      = "next"
	ResumeActionCompleted = "completed"
	ResumeActionUpToDate  = "up_to_date"
)

// ResumeInfo resuelve el clic de "Retomar" sobre un anime del historial
// (T46). EpisodeID y EpisodeNumber quedan en cero cuando la acción es
// completed o up_to_date (no hay episodio que reproducir).
type ResumeInfo struct {
	Provider      string  `json:"provider"`
	AnimeID       string  `json:"anime_id"`
	Action        string  `json:"action"`
	EpisodeID     string  `json:"episode_id"`
	EpisodeNumber int     `json:"episode_number"`
	StartAt       float64 `json:"start_at"`
}

// NextEpisode es el siguiente episodio de un anime: el de menor número
// estrictamente mayor que el actual (tolera huecos). Found=false cuando no
// existe (T46/T55).
type NextEpisode struct {
	Found     bool   `json:"found"`
	EpisodeID string `json:"episode_id"`
	Number    int    `json:"number"`
}

// ProviderName devuelve el proveedor del registro: el campo Provider si está
// informado; si no, el prefijo de EpisodeID antes del primer ":"; si no hay
// ":", "".
func (p WatchProgress) ProviderName() string {
	if p.Provider != "" {
		return p.Provider
	}
	i := strings.Index(p.EpisodeID, ":")
	if i < 0 {
		return ""
	}
	return p.EpisodeID[:i]
}
