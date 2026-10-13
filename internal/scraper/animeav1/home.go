package animeav1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

const (
	// rutaHome es el __data.json de la portada, de la que sale la sección
	// de animes recientes (T39): latestMedia con 20 portadas en la raíz.
	rutaHome = "/__data.json"

	// maxItemsHome es el tope por sección del Home: 20 items, como mucho
	// la primera página del catálogo y el latestMedia de la portada.
	maxItemsHome = 20

	// pausaSecciones es la espera entre peticiones de secciones del Home,
	// la misma cadencia que usa JKAnime entre páginas (T43).
	pausaSecciones = 300 * time.Millisecond

	// statusFiltroEmision es el filters.status que el sitio devuelve
	// cuando el filtro ?status=emision se aplicó de verdad (informe T39):
	// 0 = Finalizado, 1 = Próximamente, 2 = En emisión.
	statusFiltroEmision = 2
)

var _ scraper.HomeProvider = (*AnimeAV1)(nil)

// Home devuelve las secciones del Home de AnimeAV1, en este orden:
// "emision" (catálogo filtrado por estado, página 1) y "recientes"
// (latestMedia de la portada). Cada sección se pide con el doer del
// paquete y la pausa habitual entre peticiones; el fallo de una sección no
// anula la otra (se avisa con slog y se sigue), y si fallan todas se
// devuelve el primer error. Un ctx cancelado corta sin más peticiones y el
// error cumple errors.Is con context.Canceled.
func (a *AnimeAV1) Home(ctx context.Context) ([]models.HomeSection, error) {
	pedidas := []struct {
		id, titulo, ruta string
		parsear          func([]byte) ([]models.Anime, error)
	}{
		{"emision", "En emisión", rutaBusqueda + "?status=emision", a.parseEmision},
		{"recientes", "Recién agregados", rutaHome, a.parseRecientes},
	}

	secciones := make([]models.HomeSection, 0, len(pedidas))
	var primerError error
	for i, p := range pedidas {
		if i > 0 && !esperar(ctx, pausaSecciones) {
			return secciones, fmt.Errorf("el Home de animeav1 se interrumpió: %w", ctx.Err())
		}
		if err := ctx.Err(); err != nil {
			return secciones, fmt.Errorf("el Home de animeav1 se interrumpió: %w", err)
		}
		items, err := a.seccionHome(ctx, p.ruta, p.parsear)
		if err == nil {
			items = dedupHome(items, maxItemsHome)
			if len(items) == 0 {
				continue
			}
			secciones = append(secciones, models.HomeSection{
				ID:    p.id,
				Title: p.titulo,
				Items: items,
			})
			continue
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return secciones, fmt.Errorf("el Home de animeav1 se interrumpió: %w", ctxErr)
		}
		if primerError == nil {
			primerError = err
		}
		slog.Warn("animeav1: la sección del Home no está disponible",
			"seccion", p.id, "err", logx.Redact(err.Error()))
	}
	if len(secciones) == 0 && primerError != nil {
		return nil, fmt.Errorf("el Home de animeav1 no está disponible: %w", primerError)
	}
	return secciones, nil
}

// seccionHome descarga el __data.json de una sección con el doer del
// paquete (sin cliente nuevo) y el mismo límite de lectura de la búsqueda
// (maxBusquedaBytes): un cuerpo mayor se rechaza sin parsearlo.
func (a *AnimeAV1) seccionHome(ctx context.Context, ruta string, parsear func([]byte) ([]models.Anime, error)) ([]models.Anime, error) {
	resp, err := httpx.Get(ctx, a.Doer, a.siteURL()+ruta, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "home",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBusquedaBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leyendo el __data.json del Home: %w", err)
	}
	if len(data) > maxBusquedaBytes {
		return nil, fmt.Errorf("el __data.json del Home pesa %d bytes y el máximo es %d", len(data), maxBusquedaBytes)
	}
	return parsear(data)
}

// parseEmision extrae la sección "En emisión" del catálogo filtrado. Antes
// de convertir los results se comprueba que el filtro de estado se aplicó:
// si el sitio dejara de reconocer el slug emision devolvería todo el
// catálogo sin error (riesgo 2 del informe T39) y esa lista no es "En
// emisión". Los items salen del mismo mapeo que la búsqueda.
func (a *AnimeAV1) parseEmision(data []byte) ([]models.Anime, error) {
	datos, err := PageData(data)
	if err != nil {
		return nil, fmt.Errorf("parseando el __data.json del Home: %w", err)
	}
	if err := filtroEmisionAplicado(datos); err != nil {
		return nil, err
	}
	return resultadosAnime(datos)
}

// filtroEmisionAplicado exige filters.status == 2 en la respuesta del
// catálogo. Un filters ausente o con otro valor (null cuando el slug dejó de
// existir) hace fallar la sección entera.
func filtroEmisionAplicado(datos map[string]any) error {
	filters, ok := datos["filters"].(map[string]any)
	if !ok {
		return errors.New("animeav1: el __data.json del Home no trae filters")
	}
	status, _ := filters["status"].(float64)
	if status != statusFiltroEmision {
		return fmt.Errorf("animeav1: el filtro de estado no se aplicó (filters.status = %v)", filters["status"])
	}
	return nil
}

// parseRecientes extrae la sección "Recién agregados" del latestMedia de la
// portada: 20 objetos con id numérico, slug y título. El id solo se usa
// para la portada (idTexto/FormatFloat, nunca notación científica).
func (a *AnimeAV1) parseRecientes(data []byte) ([]models.Anime, error) {
	datos, err := PageData(data)
	if err != nil {
		return nil, fmt.Errorf("parseando el __data.json del Home: %w", err)
	}
	return itemsHome(datos, "latestMedia")
}

// itemsHome convierte la lista de animes bajo la clave dada del
// __data.json del Home en models.Anime, con el mismo criterio que
// resultadosAnime de la búsqueda: slug y título obligatorios (los que falten
// se descartan), portada desde el id del sitio y models.Anime.ID = slug.
// latestMedia trae el id numérico y results lo trae como string; portada ya
// acepta los dos.
func itemsHome(datos map[string]any, clave string) ([]models.Anime, error) {
	crudo, existe := datos[clave]
	if !existe {
		return nil, fmt.Errorf("animeav1: el __data.json del Home no trae %s", clave)
	}
	items, ok := crudo.([]any)
	if !ok {
		return nil, fmt.Errorf("animeav1: %s del Home es %T, quería una lista", clave, crudo)
	}
	out := make([]models.Anime, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		slug, _ := obj["slug"].(string)
		titulo, _ := obj["title"].(string)
		if slug == "" || titulo == "" {
			continue
		}
		out = append(out, models.Anime{
			ID:       slug,
			Title:    titulo,
			CoverURL: portada(obj["id"]),
			Provider: providerName,
		})
	}
	return out, nil
}

// dedupHome descarta los IDs repetidos conservando el orden de aparición y
// recorta la sección al máximo permitido.
func dedupHome(items []models.Anime, max int) []models.Anime {
	vistas := make(map[string]struct{}, len(items))
	out := make([]models.Anime, 0, len(items))
	for _, a := range items {
		if _, ok := vistas[a.ID]; ok {
			continue
		}
		vistas[a.ID] = struct{}{}
		out = append(out, a)
		if max > 0 && len(out) == max {
			break
		}
	}
	return out
}

// esperar espera d sin saltarse la cancelación: devuelve false si el
// contexto se termina antes de tiempo (mismo criterio que jkanime.esperar).
func esperar(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
