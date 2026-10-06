package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"miapp/internal/db"
	"miapp/internal/models"
)

// TestHistorialSeAbreEnArrancarYSeCierraEnCerrar comprueba el ciclo de vida
// del almacén (T22): arrancar lo abre en la ruta inyectada, por ahí se
// guarda y se lee, y cerrar lo deja cerrado sin tocar lo ya escrito.
func TestHistorialSeAbreEnArrancarYSeCierraEnCerrar(t *testing.T) {
	e := nuevoEntorno(t)

	if h := e.n.historialAbierto(); h != nil {
		t.Fatalf("antes de arrancar no debe haber historial abierto: %v", h)
	}
	if err := e.n.arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	if e.n.historialAbierto() == nil {
		t.Fatal("arrancar debe abrir el historial de visualización")
	}

	progreso := models.WatchProgress{
		AnimeID:       "bleach",
		EpisodeID:     "jkanime:bleach:episode:1",
		CurrentTime:   95,
		TotalDuration: 100,
	}
	if err := e.n.guardarProgreso(progreso); err != nil {
		t.Fatalf("guardarProgreso: %v", err)
	}
	registros, err := e.n.verProgreso("bleach")
	if err != nil {
		t.Fatalf("verProgreso: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("registros = %+v; quería el que acabo de guardar", registros)
	}
	if !registros[0].IsWatched {
		t.Error("IsWatched debe venir calculado por el backend (95/100)")
	}

	if err := e.n.cerrar(); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
	if err := e.n.guardarProgreso(progreso); !errors.Is(err, db.ErrCerrado) {
		t.Errorf("guardar tras cerrar = %v; quería db.ErrCerrado", err)
	}
	if _, err := e.n.verProgreso("bleach"); !errors.Is(err, db.ErrCerrado) {
		t.Errorf("leer tras cerrar = %v; quería db.ErrCerrado", err)
	}

	// Lo guardado por el núcleo está en disco: otro almacén sobre el
	// mismo directorio lo vuelve a leer.
	reabierto, err := db.Open(e.dirHistorial)
	if err != nil {
		t.Fatalf("reabrir el historial: %v", err)
	}
	defer reabierto.Close()
	enDisco, err := reabierto.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(enDisco) != 1 || !enDisco[0].IsWatched {
		t.Errorf("en disco = %+v; quería el registro con is_watched", enDisco)
	}
}

// TestGuardarProgresoSinHistorialDaErrorClaro cubre el periodo anterior a
// startup (y también un arranque en el que el almacén no pudo abrirse).
func TestGuardarProgresoSinHistorialDaErrorClaro(t *testing.T) {
	e := nuevoEntorno(t)

	progreso := models.WatchProgress{AnimeID: "bleach", EpisodeID: "jkanime:bleach:episode:1"}
	if err := e.n.guardarProgreso(progreso); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("guardarProgreso = %v; quería ErrHistorialNoDisponible", err)
	} else if err.Error() != "El historial de visualización no está disponible" {
		t.Errorf("mensaje = %q; debe estar pensado para la UI", err.Error())
	}
	if _, err := e.n.verProgreso("bleach"); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("verProgreso = %v; quería ErrHistorialNoDisponible", err)
	}
}

// TestFalloAlAbrirElHistorialNoTumbaElArrancar: un historial inaccesible
// (aquí, un directorio que no se puede crear) solo se loguea; el proxy y el
// resto de la app siguen funcionando.
func TestFalloAlAbrirElHistorialNoTumbaElArrancar(t *testing.T) {
	e := nuevoEntorno(t)

	// bloque es un ARCHIVO: MkdirAll dentro de él siempre falla.
	bloque := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(bloque, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	e.n.rutaHistorial = filepath.Join(bloque, "miapp")

	if err := e.n.arrancar(e.emis.emitir); err != nil {
		t.Fatalf("el fallo del historial no debe tumbar arrancar: %v", err)
	}
	if _, escuchadas, _ := e.proxy.contadores(); escuchadas != 1 {
		t.Errorf("escuchadas = %d; el proxy debe seguir escuchando", escuchadas)
	}
	if h := e.n.historialAbierto(); h != nil {
		t.Errorf("historial = %v; con la ruta inaccesible no debe abrirse", h)
	}
	if err := e.n.guardarProgreso(models.WatchProgress{AnimeID: "bleach"}); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Errorf("guardarProgreso = %v; quería ErrHistorialNoDisponible", err)
	}
}
