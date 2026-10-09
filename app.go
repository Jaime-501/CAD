package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"miapp/internal/app"
	"miapp/internal/logx"
	"miapp/internal/models"
)

// App es la estructura enlazada con Wails. Es deliberadamente delgada: solo
// traduce el ciclo de vida, el contexto y el directorio de datos de Wails, y
// delega en el núcleo testeable (internal/app), que no conoce Wails ni saca
// ningún test de la red real. Este archivo es lo único de la capa Go que
// depende de Wails: ni internal/** ni el núcleo importan github.com/wailsapp.
type App struct {
	// mu protege ctx: antes de startup es nil y los métodos podrían
	// leerlo desde otra goroutine.
	mu   sync.RWMutex
	ctx  context.Context
	core *app.Nucleo
}

// DetalleResultado es la respuesta de GetAnimeDetails: el detalle y la
// primera tanda de episodios que ya trae el provider. Los lotes siguientes
// llegan por el evento episodes:update. Es la única forma de devolver dos
// valores a Wails: BoundMethod.Call solo admite (valor, error).
//
// Vive en package main —con los mismos campos y tags que
// app.DetalleResultado de internal/app— para que Wails la siga generando en
// el namespace `main` y frontend/wailsjs/** no cambie (T37/P5).
type DetalleResultado struct {
	Detail   models.AnimeDetail `json:"detail"`
	Episodes []models.Episode   `json:"episodes"`
}

// NewApp crea la aplicación con sus dependencias reales (registro de
// proveedores, proxy local y selector de mejor fuente). No toca la red ni
// escucha puertos: eso ocurre en startup.
func NewApp() *App {
	return &App{core: app.NuevoNucleoPorDefecto(dirHistorial())}
}

// dirHistorial es el directorio del historial de visualización (T22):
// os.UserConfigDir()/miapp. Es el ÚNICO punto del código que pregunta la
// configuración del usuario (P3): todo lo demás vive en internal/** y es
// portable. Si el sistema no la da, devuelve "" y el historial no se abre —
// el aviso lo emite arrancar, igual que antes.
func dirHistorial() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "miapp")
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
	if err := a.core.Arrancar(emisor); err != nil {
		slog.Warn("la app sigue sin proxy local de streaming", "err", err)
	}
}

// shutdown se llama al cerrar la app (OnShutdown de main.go): cancela el
// stream activo y detiene el proxy. Sin este hook el listener y las
// goroutines de los streams vivirían hasta que el proceso muera.
func (a *App) shutdown(_ context.Context) {
	if err := a.core.Cerrar(); err != nil {
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

// SearchAnime busca anime y conserva la firma que ya usa App.svelte: la
// búsqueda es COMBINADA (T34) —JKAnime y AnimeAV1 en paralelo, con los
// resultados del primero por delante— y solo falla si fallan los dos.
// No hay método Search: no cambies este nombre sin avisar a T25.
func (a *App) SearchAnime(query string) ([]models.Anime, error) {
	return a.core.Buscar(a.contexto(), query)
}

// GetAnimeDetails devuelve el detalle y la primera tanda de episodios. Si el
// provider emite episodios por lotes (jkanime y animeav1), los siguientes
// llegan por los eventos "episodes:update" y "episodes:complete". Un
// proveedor sin página de detalle (scraper.ErrNoDetails) devuelve
// ErrDetalleNoDisponible; un nombre inexistente, un error claro con los
// proveedores disponibles.
func (a *App) GetAnimeDetails(provider string, id string) (DetalleResultado, error) {
	r, err := a.core.Detalle(a.contexto(), provider, id)
	if err != nil {
		return DetalleResultado{}, err
	}
	return DetalleResultado{Detail: r.Detail, Episodes: r.Episodes}, nil
}

// GetVideoSources devuelve las fuentes de un episodio con la mejor PRIMERO.
// Las HLS/MP4 vienen con la URL apuntando al proxy local (127.0.0.1); los
// iframes pasan sin proxy. noCache se acepta, se reenvía al proveedor y se
// sigue ignorando: la caché local (T21) guarda detalle y episodios, nunca
// fuentes de vídeo (los enlaces son temporales, regla 9).
func (a *App) GetVideoSources(provider string, episodeID string, noCache bool) ([]models.VideoSource, error) {
	return a.core.Fuentes(a.contexto(), provider, episodeID, noCache)
}

// SaveProgress guarda el progreso de un episodio (lo llamará el reproductor
// cada ~5 s): el almacén escribe a disco de forma atómica y calcula aquí el
// is_watched (CurrentTime/TotalDuration >= 0.90; el que traiga el llamador
// se ignora). Falla con ErrHistorialNoDisponible antes de arrancar o si el
// almacén no pudo abrirse, y con db.ErrCerrado tras el shutdown.
func (a *App) SaveProgress(progress models.WatchProgress) error {
	return a.core.GuardarProgreso(progress)
}

// GetProgress devuelve el historial guardado de un anime, en orden de
// primera guardada; animeID vacío devuelve el historial entero. Cada
// registro trae is_watched ya calculado por el backend.
func (a *App) GetProgress(animeID string) ([]models.WatchProgress, error) {
	return a.core.VerProgreso(animeID)
}
