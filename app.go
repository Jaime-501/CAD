package main

import (
	"context"
	"fmt"

	"miapp/internal/models"
	"miapp/internal/scraper"
)

// App struct
type App struct {
	ctx             context.Context
	animeFlvScraper *scraper.AnimeFLV
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		animeFlvScraper: scraper.NewAnimeFLV(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) SearchAnime(query string) ([]models.Anime, error) {
	return a.animeFlvScraper.Search(query)
}
