package app

import (
	"errors"
	"strings"
	"testing"

	"miapp/internal/cache"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/jkanime/extractors"
)

// TestFavoritosAgregarListarQuitar: el ciclo completo de la estrella sobre
// un almacén abierto en arrancar (T42/T45).
func TestFavoritosAgregarListarQuitar(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	favorito := models.Favorite{
		Provider: proveedorPrincipal,
		AnimeID:  "bleach",
		Title:    "Bleach",
		CoverURL: "https://cdn.jkdesa.com/cover.jpg",
	}
	if err := e.n.AgregarFavorito(favorito); err != nil {
		t.Fatalf("AgregarFavorito: %v", err)
	}
	ok, err := e.n.EsFavorito(proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("EsFavorito: %v", err)
	}
	if !ok {
		t.Error("EsFavorito = false tras agregar")
	}
	favoritos, err := e.n.ListarFavoritos()
	if err != nil {
		t.Fatalf("ListarFavoritos: %v", err)
	}
	if len(favoritos) != 1 || favoritos[0].AnimeID != "bleach" || favoritos[0].Provider != proveedorPrincipal {
		t.Fatalf("ListarFavoritos = %+v; se esperaba solo bleach de jkanime", favoritos)
	}
	if favoritos[0].Title != "Bleach" || favoritos[0].CoverURL != "https://cdn.jkdesa.com/cover.jpg" {
		t.Errorf("el favorito no conserva título/portada: %+v", favoritos[0])
	}

	if err := e.n.QuitarFavorito(proveedorPrincipal, "bleach"); err != nil {
		t.Fatalf("QuitarFavorito: %v", err)
	}
	ok, err = e.n.EsFavorito(proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("EsFavorito tras quitar: %v", err)
	}
	if ok {
		t.Error("EsFavorito = true tras quitar")
	}
	favoritos, err = e.n.ListarFavoritos()
	if err != nil {
		t.Fatalf("ListarFavoritos tras quitar: %v", err)
	}
	if len(favoritos) != 0 {
		t.Errorf("ListarFavoritos = %+v; se esperaba vacío", favoritos)
	}
}

// TestFavoritosAgregarProveedorInexistenteFalla: la clave del favorito es
// proveedor+slug (T40): un proveedor inventado no se guarda.
func TestFavoritosAgregarProveedorInexistenteFalla(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	err := e.n.AgregarFavorito(models.Favorite{Provider: "noexiste", AnimeID: "bleach"})
	if err == nil {
		t.Fatal("AgregarFavorito con proveedor inexistente no debe funcionar")
	}
	if !strings.Contains(err.Error(), "proveedor desconocido") {
		t.Errorf("error = %q; se esperaba \"proveedor desconocido\"", err)
	}
	favoritos, err := e.n.ListarFavoritos()
	if err != nil {
		t.Fatalf("ListarFavoritos: %v", err)
	}
	if len(favoritos) != 0 {
		t.Errorf("el favorito inválido no debe guardarse: %+v", favoritos)
	}
}

// TestFavoritosSinDirectorioDevuelveErrFavoritosNoDisponible: un núcleo
// construido sin directorio de datos (P3) no abre almacén y cada método
// devuelve el sentinel con su texto de UI.
func TestFavoritosSinDirectorioDevuelveErrFavoritosNoDisponible(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	e.n.rutaHistorial = "" // sin directorio de datos no hay almacén
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar sin directorio: %v", err)
	}

	if _, err := e.n.ListarFavoritos(); !errors.Is(err, ErrFavoritosNoDisponible) {
		t.Errorf("ListarFavoritos = %v; se esperaba ErrFavoritosNoDisponible", err)
	} else if err.Error() != "Los favoritos no están disponibles" {
		t.Errorf("mensaje = %q; debe estar pensado para la UI", err.Error())
	}
	if _, err := e.n.EsFavorito(proveedorPrincipal, "bleach"); !errors.Is(err, ErrFavoritosNoDisponible) {
		t.Errorf("EsFavorito = %v; se esperaba ErrFavoritosNoDisponible", err)
	}
	if err := e.n.AgregarFavorito(models.Favorite{Provider: proveedorPrincipal, AnimeID: "bleach"}); !errors.Is(err, ErrFavoritosNoDisponible) {
		t.Errorf("AgregarFavorito = %v; se esperaba ErrFavoritosNoDisponible", err)
	}
	if err := e.n.QuitarFavorito(proveedorPrincipal, "bleach"); !errors.Is(err, ErrFavoritosNoDisponible) {
		t.Errorf("QuitarFavorito = %v; se esperaba ErrFavoritosNoDisponible", err)
	}
}

// TestFavoritosPersistenEntreArranques: lo guardado vive en favoritos.json
// del directorio inyectado; un segundo núcleo sobre el mismo directorio lo
// vuelve a leer.
func TestFavoritosPersistenEntreArranques(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	if err := e.n.AgregarFavorito(models.Favorite{Provider: proveedorPrincipal, AnimeID: "bleach", Title: "Bleach"}); err != nil {
		t.Fatalf("AgregarFavorito: %v", err)
	}
	if err := e.n.Cerrar(); err != nil {
		t.Fatalf("cerrar: %v", err)
	}

	n2 := NuevoNucleo(scraper.NewRegistry(), nuevoProxyFalso(), extractors.SelectBestSource, nil,
		cache.New(ttlCacheDetalle), e.dirHistorial)
	if err := n2.Arrancar(nil); err != nil {
		t.Fatalf("segundo arrancar: %v", err)
	}
	t.Cleanup(func() { _ = n2.Cerrar() })

	favoritos, err := n2.ListarFavoritos()
	if err != nil {
		t.Fatalf("ListarFavoritos tras reabrir: %v", err)
	}
	if len(favoritos) != 1 || favoritos[0].AnimeID != "bleach" || favoritos[0].Provider != proveedorPrincipal {
		t.Errorf("favoritos tras reabrir = %+v; se esperaba bleach de jkanime", favoritos)
	}
}

// TestFavoritosMismoSlugDosProveedores: la clave es proveedor+slug, así que
// el mismo anime se guarda por sitio y quitar uno no toca el otro.
func TestFavoritosMismoSlugDosProveedores(t *testing.T) {
	e := nuevoEntorno(t,
		&proveedorFalso{nombre: proveedorPrincipal},
		&proveedorFalso{nombre: proveedorComplemento})
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	if err := e.n.AgregarFavorito(models.Favorite{Provider: proveedorPrincipal, AnimeID: "bleach", Title: "Bleach"}); err != nil {
		t.Fatalf("AgregarFavorito(jkanime): %v", err)
	}
	if err := e.n.AgregarFavorito(models.Favorite{Provider: proveedorComplemento, AnimeID: "bleach", Title: "Bleach"}); err != nil {
		t.Fatalf("AgregarFavorito(animeav1): %v", err)
	}

	favoritos, err := e.n.ListarFavoritos()
	if err != nil {
		t.Fatalf("ListarFavoritos: %v", err)
	}
	if len(favoritos) != 2 {
		t.Fatalf("favoritos = %d; se esperaban 2 (el slug se comparte entre proveedores)", len(favoritos))
	}
	for _, proveedor := range []string{proveedorPrincipal, proveedorComplemento} {
		ok, err := e.n.EsFavorito(proveedor, "bleach")
		if err != nil || !ok {
			t.Errorf("EsFavorito(%s, bleach) = %v, %v; se esperaba true sin error", proveedor, ok, err)
		}
	}

	if err := e.n.QuitarFavorito(proveedorComplemento, "bleach"); err != nil {
		t.Fatalf("QuitarFavorito(animeav1): %v", err)
	}
	ok, err := e.n.EsFavorito(proveedorPrincipal, "bleach")
	if err != nil || !ok {
		t.Errorf("quitar uno no debe borrar el del otro proveedor: EsFavorito(jkanime) = %v, %v", ok, err)
	}
	favoritos, err = e.n.ListarFavoritos()
	if err != nil {
		t.Fatalf("ListarFavoritos tras quitar: %v", err)
	}
	if len(favoritos) != 1 || favoritos[0].Provider != proveedorPrincipal {
		t.Errorf("favoritos tras quitar = %+v; se esperaba el de jkanime", favoritos)
	}
}
