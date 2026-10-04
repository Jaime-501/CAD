package models

type Anime struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	CoverURL string `json:"cover_url"`
	Provider string `json:"provider"`
}

type Episode struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
}
