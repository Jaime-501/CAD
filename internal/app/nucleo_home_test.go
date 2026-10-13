package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"miapp/internal/models"
)

// homeFalso añade scraper.HomeProvider al doble de proveedor: cuenta las
// llamadas de Home y devuelve las secciones (o el error) inyectados.
type homeFalso struct {
	proveedorFalso
	mu        sync.Mutex
	llamadas  int
	secciones []models.HomeSection
	err       error
}

func (h *homeFalso) Home(ctx context.Context) ([]models.HomeSection, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.llamadas++
	if h.err != nil {
		return nil, h.err
	}
	return h.secciones, nil
}

func (h *homeFalso) totalLlamadas() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.llamadas
}

// cambiar fija el resultado de las próximas llamadas (secciones o error).
func (h *homeFalso) cambiar(secciones []models.HomeSection, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.secciones, h.err = secciones, err
}

// seccionesDePrueba son las dos filas del Home del doble.
func seccionesDePrueba() []models.HomeSection {
	return []models.HomeSection{
		{ID: "emision", Title: "En emisión", Items: []models.Anime{
			{ID: "bleach", Title: "Bleach", Provider: proveedorPrincipal},
		}},
		{ID: "recientes", Title: "Recién agregados", Items: []models.Anime{
			{ID: "naruto", Title: "Naruto", Provider: proveedorPrincipal},
		}},
	}
}

// TestHomeDevuelveSeccionesDelProveedor: el núcleo delega en
// scraper.HomeProvider y devuelve sus secciones tal cual.
func TestHomeDevuelveSeccionesDelProveedor(t *testing.T) {
	h := &homeFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		secciones:      seccionesDePrueba(),
	}
	e := nuevoEntorno(t, h)

	secciones, err := e.n.Home(context.Background(), proveedorPrincipal, false)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if len(secciones) != 2 {
		t.Fatalf("secciones = %d; se esperaban 2", len(secciones))
	}
	if secciones[0].ID != "emision" || secciones[0].Title != "En emisión" || len(secciones[0].Items) != 1 {
		t.Errorf("sección 0 = %+v; se esperaba emision con su item", secciones[0])
	}
	if secciones[1].ID != "recientes" || secciones[1].Items[0].ID != "naruto" {
		t.Errorf("sección 1 = %+v; se esperaba recientes con naruto", secciones[1])
	}
	if got := h.totalLlamadas(); got != 1 {
		t.Errorf("llamadas al proveedor = %d; se esperaba 1", got)
	}
}

// TestHomeProveedorDesconocido: el error es el mismo que ya devuelve
// Detalle, con la lista de proveedores disponibles.
func TestHomeProveedorDesconocido(t *testing.T) {
	h := &homeFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		secciones:      seccionesDePrueba(),
	}
	e := nuevoEntorno(t, h)

	_, err := e.n.Home(context.Background(), "noexiste", false)
	if err == nil {
		t.Fatal("Home de un proveedor desconocido no debe devolver secciones")
	}
	if !strings.Contains(err.Error(), "proveedor desconocido") {
		t.Errorf("error = %q; se esperaba \"proveedor desconocido\"", err)
	}
	if !strings.Contains(err.Error(), proveedorPrincipal) {
		t.Errorf("el error debe listar los proveedores disponibles: %q", err)
	}
	if got := h.totalLlamadas(); got != 0 {
		t.Errorf("el proveedor no debe consultarse: %d llamadas", got)
	}
}

// TestHomeProveedorSinHomeProviderDevuelveErrHomeNoDisponible: proveedor
// registrado pero sin la interfaz opcional T43 ⇒ error tipado, no fallo de
// red.
func TestHomeProveedorSinHomeProviderDevuelveErrHomeNoDisponible(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})

	_, err := e.n.Home(context.Background(), proveedorPrincipal, false)
	if !errors.Is(err, ErrHomeNoDisponible) {
		t.Fatalf("Home = %v; se esperaba ErrHomeNoDisponible", err)
	}
	if err.Error() != "El Home no está disponible para este proveedor" {
		t.Errorf("mensaje = %q; debe estar pensado para la UI", err.Error())
	}
}

// TestHomeSeCacheaYNoCacheLoSalta: contador de llamadas del proveedor
// 1 → 1 (caché) → 2 (noCache refresca) → 2 (vuelve a caché).
func TestHomeSeCacheaYNoCacheLoSalta(t *testing.T) {
	h := &homeFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		secciones:      seccionesDePrueba(),
	}
	e := nuevoEntorno(t, h)

	if _, err := e.n.Home(context.Background(), proveedorPrincipal, false); err != nil {
		t.Fatalf("primera Home: %v", err)
	}
	if _, err := e.n.Home(context.Background(), proveedorPrincipal, false); err != nil {
		t.Fatalf("segunda Home: %v", err)
	}
	if got := h.totalLlamadas(); got != 1 {
		t.Errorf("con caché, llamadas = %d; se esperaba 1", got)
	}
	if _, err := e.n.Home(context.Background(), proveedorPrincipal, true); err != nil {
		t.Fatalf("Home sin caché: %v", err)
	}
	if got := h.totalLlamadas(); got != 2 {
		t.Errorf("noCache debe saltar la lectura: llamadas = %d; se esperaba 2", got)
	}
	if _, err := e.n.Home(context.Background(), proveedorPrincipal, false); err != nil {
		t.Fatalf("cuarta Home: %v", err)
	}
	if got := h.totalLlamadas(); got != 2 {
		t.Errorf("tras refrescar, llamadas = %d; se esperaba 2 (la caché volvió a poblarse)", got)
	}
}

// TestHomeErrorNoSeCachea: un fallo del proveedor no crea entrada; la
// próxima llamada vuelve a preguntar y puede recuperarse.
func TestHomeErrorNoSeCachea(t *testing.T) {
	h := &homeFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		err:            errors.New("el sitio está caído"),
	}
	e := nuevoEntorno(t, h)

	if _, err := e.n.Home(context.Background(), proveedorPrincipal, false); err == nil {
		t.Fatal("la primera Home debe devolver el error del proveedor")
	}
	if _, err := e.n.Home(context.Background(), proveedorPrincipal, false); err == nil {
		t.Fatal("el error no se cachea: la segunda Home también debe fallar")
	}
	if got := h.totalLlamadas(); got != 2 {
		t.Errorf("los errores no se cachean: llamadas = %d; se esperaban 2", got)
	}

	h.cambiar(seccionesDePrueba(), nil)
	secciones, err := e.n.Home(context.Background(), proveedorPrincipal, false)
	if err != nil {
		t.Fatalf("Home tras recuperarse: %v", err)
	}
	if len(secciones) != 2 {
		t.Errorf("secciones = %d; se esperaban 2 (la caché no guardó el fallo)", len(secciones))
	}
}

// TestHomeCopiaDefensiva: mutar lo devuelto no altera lo que guarda la
// caché (la segunda llamada sale intacta).
func TestHomeCopiaDefensiva(t *testing.T) {
	h := &homeFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		secciones:      seccionesDePrueba(),
	}
	e := nuevoEntorno(t, h)

	secciones, err := e.n.Home(context.Background(), proveedorPrincipal, false)
	if err != nil {
		t.Fatalf("primera Home: %v", err)
	}
	secciones[0].Title = "mutado"
	secciones[0].Items[0].Title = "mutado también"
	secciones[0].Items = append(secciones[0].Items, models.Anime{ID: "inventado"})

	segunda, err := e.n.Home(context.Background(), proveedorPrincipal, false)
	if err != nil {
		t.Fatalf("segunda Home: %v", err)
	}
	if segunda[0].Title != "En emisión" || segunda[0].Items[0].Title != "Bleach" || len(segunda[0].Items) != 1 {
		t.Errorf("la mutación del primer resultado alteró la caché: %+v", segunda[0])
	}
}
