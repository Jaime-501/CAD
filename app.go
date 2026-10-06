package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"miapp/internal/logx"
	"miapp/internal/models"
)

// App es la estructura enlazada con Wails. Es deliberadamente delgada: solo
// traduce el ciclo de vida y el contexto de Wails, y delega en el núcleo
// testeable (app_nucleo.go), que no conoce Wails ni saca ningún test de la
// red real.
type App struct {
	// mu protege ctx: antes de startup es nil y los métodos podrían
	// leerlo desde otra goroutine.
	mu   sync.RWMutex
	ctx  context.Context
	core *nucleo
}

// NewApp crea la aplicación con sus dependencias reales (registro de
// proveedores, proxy local y selector de mejor fuente). No toca la red ni
// escucha puertos: eso ocurre en startup.
func NewApp() *App {
	return &App{core: nuevoNucleoPorDefecto()}
}

// startup se llama cuando la app arranca: guarda el contexto de Wails —el
// que exige runtime.EventsEmit—, instala el logger con redacción de
// secretos (T01) también como logger por defecto de slog (T06, los warns de
// jkanime) y arranca el núcleo (emisor + proxy escuchando en 127.0.0.1).
//
// No hay flag público de debug en Wails (internal/app.IsDebug es interno):
// MIAPP_DEV activa los logs en modo texto/debug.
func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()

	slog.SetDefault(logx.New(os.Getenv("MIAPP_DEV") != ""))

	emisor := func(evento string, data any) {
		runtime.EventsEmit(ctx, evento, data)
	}
	if err := a.core.arrancar(emisor); err != nil {
		slog.Warn("la app sigue sin proxy local de streaming", "err", err)
	}
}

// shutdown se llama al cerrar la app (OnShutdown de main.go): cancela el
// stream activo y detiene el proxy. Sin este hook el listener y las
// goroutines de los streams vivirían hasta que el proceso muera.
func (a *App) shutdown(_ context.Context) {
	if err := a.core.cerrar(); err != nil {
		slog.Warn("error al cerrar el proxy local de streaming", "err", err)
	}
}

// contexto devuelve el contexto de Wails; antes de startup no lo hay y se
// usa context.Background() (el frontend no puede llamar antes de arrancar,
// pero así nil nunca llega a los proveedores).
func (a *App) contexto() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// SearchAnime busca anime y conserva la firma que ya usa App.svelte:
// JKAnime primero y, si falla o no devuelve nada, AnimeFLV (solo búsqueda).
// No hay método Search: no cambies este nombre sin avisar a T25.
func (a *App) SearchAnime(query string) ([]models.Anime, error) {
	return a.core.buscar(a.contexto(), query)
}

// GetAnimeDetails devuelve el detalle y la primera tanda de episodios. Si el
// provider emite episodios por lotes (jkanime), los siguientes llegan por
// los eventos "episodes:update" y "episodes:complete". Un proveedor sin
// detalle (animeflv) devuelve ErrDetalleNoDisponible; un nombre inexistente,
// un error claro con los proveedores disponibles.
func (a *App) GetAnimeDetails(provider string, id string) (DetalleResultado, error) {
	return a.core.detalle(a.contexto(), provider, id)
}

// GetVideoSources devuelve las fuentes de un episodio con la mejor PRIMERO.
// Las HLS/MP4 vienen con la URL apuntando al proxy local (127.0.0.1); los
// iframes pasan sin proxy. noCache se acepta, se reenvía al proveedor y se
// sigue ignorando: la caché local (T21) guarda detalle y episodios, nunca
// fuentes de vídeo (los enlaces son temporales, regla 9).
func (a *App) GetVideoSources(provider string, episodeID string, noCache bool) ([]models.VideoSource, error) {
	return a.core.fuentes(a.contexto(), provider, episodeID, noCache)
}
