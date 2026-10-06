package main

import (
	"context"
	"fmt"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper/animeflv"
)

// App struct
type App struct {
	ctx             context.Context
	animeFlvScraper *animeflv.AnimeFLV
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		animeFlvScraper: animeflv.New(httpx.NewClient(httpx.ClientOptions{}), ""),
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
	return a.animeFlvScraper.Search(a.ctx, query)
}
