package app

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"miapp/internal/models"
)

// TestHistorialRecienteDelegaEnDb: HistorialReciente es LastPerAnime (T41):
// un registro por anime, el más reciente primero.
func TestHistorialRecienteDelegaEnDb(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "bleach", EpisodeID: proveedorPrincipal + ":bleach:episode:1", UpdatedAt: 1000,
	}); err != nil {
		t.Fatalf("GuardarProgreso(bleach): %v", err)
	}
	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "naruto", EpisodeID: proveedorPrincipal + ":naruto:episode:3", UpdatedAt: 2000,
	}); err != nil {
		t.Fatalf("GuardarProgreso(naruto): %v", err)
	}

	reciente, err := e.n.HistorialReciente()
	if err != nil {
		t.Fatalf("HistorialReciente: %v", err)
	}
	if len(reciente) != 2 {
		t.Fatalf("HistorialReciente = %d entradas; se esperaban 2 (una por anime)", len(reciente))
	}
	if reciente[0].AnimeID != "naruto" || reciente[1].AnimeID != "bleach" {
		t.Errorf("orden = %q, %q; se esperaba naruto primero (UpdatedAt mayor)", reciente[0].AnimeID, reciente[1].AnimeID)
	}
	if reciente[0].Provider != proveedorPrincipal {
		t.Errorf("Provider = %q; db lo rellena desde el episode_id", reciente[0].Provider)
	}
}

// TestBorrarHistorialAnimeProveedorYTodo: los tres borrados de T41 con sus
// recuentos, y el historial queda vacío al final.
func TestBorrarHistorialAnimeProveedorYTodo(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	// A: jkanime/bleach con DOS episodios; B: animeav1/naruto con uno, para
	// que borrar por proveedor no se cruce con el anime de A.
	for ep := 1; ep <= 2; ep++ {
		if err := e.n.GuardarProgreso(models.WatchProgress{
			AnimeID:   "bleach",
			EpisodeID: proveedorPrincipal + ":bleach:episode:" + strconv.Itoa(ep),
			UpdatedAt: int64(1000 + ep),
		}); err != nil {
			t.Fatalf("GuardarProgreso(bleach/%d): %v", ep, err)
		}
	}
	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "naruto", EpisodeID: proveedorComplemento + ":naruto:episode:1", UpdatedAt: 2000,
	}); err != nil {
		t.Fatalf("GuardarProgreso(naruto): %v", err)
	}

	borrados, err := e.n.BorrarHistorialAnime(proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("BorrarHistorialAnime: %v", err)
	}
	if borrados != 2 {
		t.Errorf("BorrarHistorialAnime = %d; se esperaban 2 (los dos episodios de bleach)", borrados)
	}
	reciente, err := e.n.HistorialReciente()
	if err != nil {
		t.Fatalf("HistorialReciente: %v", err)
	}
	if len(reciente) != 1 || reciente[0].AnimeID != "naruto" {
		t.Errorf("tras borrar bleach = %+v; se esperaba solo naruto", reciente)
	}

	borrados, err = e.n.BorrarHistorialProveedor(proveedorComplemento)
	if err != nil {
		t.Fatalf("BorrarHistorialProveedor: %v", err)
	}
	if borrados != 1 {
		t.Errorf("BorrarHistorialProveedor = %d; se esperaba 1 (naruto)", borrados)
	}
	if reciente, err = e.n.HistorialReciente(); err != nil || len(reciente) != 0 {
		t.Errorf("tras borrar el proveedor = %+v, %v; se esperaba vacío sin error", reciente, err)
	}

	// BorrarHistorialTodo sobre algo nuevo.
	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "bleach", EpisodeID: proveedorPrincipal + ":bleach:episode:1", UpdatedAt: 3000,
	}); err != nil {
		t.Fatalf("GuardarProgreso (tras borrar): %v", err)
	}
	borrados, err = e.n.BorrarHistorialTodo()
	if err != nil {
		t.Fatalf("BorrarHistorialTodo: %v", err)
	}
	if borrados != 1 {
		t.Errorf("BorrarHistorialTodo = %d; se esperaba 1", borrados)
	}
	if reciente, err = e.n.HistorialReciente(); err != nil || len(reciente) != 0 {
		t.Errorf("tras borrar todo = %+v, %v; se esperaba vacío sin error", reciente, err)
	}
}

// TestBorrarHistorialSinDirectorioDevuelveErrHistorialNoDisponible: sin
// almacén abierto (antes de arrancar o sin directorio de datos) los
// borrados y el reciente devuelven el sentinel de T22.
func TestBorrarHistorialSinDirectorioDevuelveErrHistorialNoDisponible(t *testing.T) {
	e := nuevoEntorno(t)

	if _, err := e.n.HistorialReciente(); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("HistorialReciente = %v; se esperaba ErrHistorialNoDisponible", err)
	}
	if _, err := e.n.BorrarHistorialAnime(proveedorPrincipal, "bleach"); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("BorrarHistorialAnime = %v; se esperaba ErrHistorialNoDisponible", err)
	}
	if _, err := e.n.BorrarHistorialProveedor(proveedorPrincipal); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("BorrarHistorialProveedor = %v; se esperaba ErrHistorialNoDisponible", err)
	}
	if _, err := e.n.BorrarHistorialTodo(); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("BorrarHistorialTodo = %v; se esperaba ErrHistorialNoDisponible", err)
	}
}

// TestGuardarProgresoConservaUpdatedAtNoCero: la regla de sellar la fecha
// vive en db.Save (T41); el núcleo pasa el registro tal cual y no pisa una
// fecha distinta de 0 (clave del Deshacer de T46).
func TestGuardarProgresoConservaUpdatedAtNoCero(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "bleach", EpisodeID: proveedorPrincipal + ":bleach:episode:1", UpdatedAt: 12345,
	}); err != nil {
		t.Fatalf("GuardarProgreso: %v", err)
	}
	guardado, err := e.n.VerProgreso("bleach")
	if err != nil {
		t.Fatalf("VerProgreso: %v", err)
	}
	if len(guardado) != 1 || guardado[0].UpdatedAt != 12345 {
		t.Errorf("UpdatedAt = %+v; el núcleo no debe pisar una fecha distinta de 0", guardado)
	}

	// UpdatedAt 0 lo sella db con su reloj, no el núcleo.
	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "bleach", EpisodeID: proveedorPrincipal + ":bleach:episode:2",
	}); err != nil {
		t.Fatalf("GuardarProgreso (sin fecha): %v", err)
	}
	guardado, err = e.n.VerProgreso("bleach")
	if err != nil {
		t.Fatalf("VerProgreso (2): %v", err)
	}
	var sinFecha *models.WatchProgress
	for i := range guardado {
		if guardado[i].EpisodeID == proveedorPrincipal+":bleach:episode:2" {
			sinFecha = &guardado[i]
		}
	}
	if sinFecha == nil {
		t.Fatalf("no está el episodio 2 en %+v", guardado)
	}
	if sinFecha.UpdatedAt == 0 {
		t.Error("UpdatedAt = 0 tras guardar: db debe sellar la fecha")
	}
}

// TestBorrarHistorialRechazaProveedorOIDVacio: los argumentos vacíos llegan
// como error en español ANTES de tocar db (la nota de T41: un "" borraría
// registros sin proveedor) y el historial queda intacto.
func TestBorrarHistorialRechazaProveedorOIDVacio(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID: "bleach", EpisodeID: proveedorPrincipal + ":bleach:episode:1", UpdatedAt: 1000,
	}); err != nil {
		t.Fatalf("GuardarProgreso: %v", err)
	}

	casos := []struct {
		nombre string
		llamar func() (int, error)
	}{
		{"animeID vacío", func() (int, error) { return e.n.BorrarHistorialAnime(proveedorPrincipal, "") }},
		{"proveedor vacío", func() (int, error) { return e.n.BorrarHistorialAnime("", "bleach") }},
		{"proveedor del borrado por proveedor vacío", func() (int, error) { return e.n.BorrarHistorialProveedor("") }},
	}
	for _, c := range casos {
		n, err := c.llamar()
		if err == nil {
			t.Errorf("%s: borrado aceptado con %d registros eliminados; se esperaba error", c.nombre, n)
			continue
		}
		if !strings.Contains(err.Error(), "proveedor") {
			t.Errorf("%s: error = %q; debe decir en español qué falta", c.nombre, err)
		}
	}

	guardado, err := e.n.VerProgreso("bleach")
	if err != nil {
		t.Fatalf("VerProgreso: %v", err)
	}
	if len(guardado) != 1 {
		t.Errorf("historial = %d registros; los argumentos vacíos no deben tocarlo", len(guardado))
	}
}
