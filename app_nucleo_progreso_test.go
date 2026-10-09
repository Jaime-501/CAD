package main

import (
	"strings"
	"testing"

	"miapp/internal/db"
	"miapp/internal/models"
)

// registrosDeProveedor espeja el filtro que aplica el frontend (stores.ts:
// `filtrarPorPrefijo`) sobre el historial que devuelve verProgreso. El
// almacén se guarda por slug de anime, y un slug como "bleach" existe en
// jkanime Y en animeav1, así que el que decide qué ve cada lista es el
// prefijo de proveedor del episode_id. Vive en el test porque el frontend no
// tiene runner de tests (fuera de frontend/src/** no podemos tocar
// package.json); el día que la lógica se mueva a TS, este es el contrato.
func registrosDeProveedor(proveedor string, registros []models.WatchProgress) []models.WatchProgress {
	prefijo := proveedor + ":"
	propios := make([]models.WatchProgress, 0, len(registros))
	for _, r := range registros {
		if strings.HasPrefix(r.EpisodeID, prefijo) {
			propios = append(propios, r)
		}
	}
	return propios
}

// TestProgresoDelMismoSlugSeAislaPorPrefijoDelEpisodeID cubre el riesgo de
// "fuga de progreso" de T30: el progreso se guarda por anime_id (slug), que
// es el mismo para los dos proveedores. Comprueba que (a) los dos registros
// coexisten sin migrarse ni reescribirse, (b) el prefijo del episode_id
// separa cada proveedor de forma exacta (sin mezclar ni perder), y (c) lo
// escrito en disco sigue intacto tras releerlo con otro almacén.
func TestProgresoDelMismoSlugSeAislaPorPrefijoDelEpisodeID(t *testing.T) {
	e := nuevoEntorno(t)
	if err := e.n.arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}

	// "bleach" en los dos proveedores: mismo anime_id (slug), distinto
	// prefijo en episode_id. Se guardan los dos tal cual, sin migración.
	progresos := []models.WatchProgress{
		{AnimeID: "bleach", EpisodeID: "jkanime:bleach:episode:3", CurrentTime: 95, TotalDuration: 100},
		{AnimeID: "bleach", EpisodeID: "animeav1:bleach:episode:1", CurrentTime: 30, TotalDuration: 240},
	}
	for _, p := range progresos {
		if err := e.n.guardarProgreso(p); err != nil {
			t.Fatalf("guardarProgreso(%s): %v", p.EpisodeID, err)
		}
	}

	todos, err := e.n.verProgreso("bleach")
	if err != nil {
		t.Fatalf("verProgreso: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("verProgreso(bleach) = %+v; quería los DOS proveedores (la clave es el slug)", todos)
	}

	// (b) el filtro por prefijo: cada proveedor ve exactamente lo suyo, con
	// sus valores, y nada del otro.
	querido := map[string]struct {
		episodio      string
		currentTime   float64
		totalDuration float64
	}{
		"jkanime":  {"jkanime:bleach:episode:3", 95, 100},
		"animeav1": {"animeav1:bleach:episode:1", 30, 240},
	}
	for proveedor, q := range querido {
		propios := registrosDeProveedor(proveedor, todos)
		if len(propios) != 1 {
			t.Fatalf("%s: registros = %+v; quería exactamente 1 (sin fugas del otro)", proveedor, propios)
		}
		r := propios[0]
		if r.EpisodeID != q.episodio || r.CurrentTime != q.currentTime || r.TotalDuration != q.totalDuration {
			t.Errorf("%s: registro = %+v; quería %+v", proveedor, r, q)
		}
		// El prefijo que usa el frontend es exactamente el proveedor que
		// declara el propio episode_id: no hay que interpretar nada más.
		prov, slug, _, _, err := models.ParseEpisodeID(r.EpisodeID)
		if err != nil {
			t.Fatalf("ParseEpisodeID(%q): %v", r.EpisodeID, err)
		}
		if prov != proveedor || slug != "bleach" {
			t.Errorf("ParseEpisodeID(%q) = %s/%s; quería %s/bleach", r.EpisodeID, prov, slug, proveedor)
		}
	}
	if fugados := registrosDeProveedor("animeflv", todos); len(fugados) != 0 {
		t.Errorf("un proveedor sin registros no debe heredar nada: %+v", fugados)
	}

	// (c) nada reescrito: el historial entero sigue con los dos y otro
	// almacén sobre el mismo directorio los vuelve a leer igual.
	historial, err := e.n.verProgreso("")
	if err != nil {
		t.Fatalf("historial entero: %v", err)
	}
	if len(historial) != 2 {
		t.Errorf("historial = %+v; quería los 2 registros", historial)
	}
	if err := e.n.cerrar(); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
	reabierto, err := db.Open(e.dirHistorial)
	if err != nil {
		t.Fatalf("reabrir el historial: %v", err)
	}
	defer reabierto.Close()
	enDisco, err := reabierto.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(enDisco) != 2 {
		t.Errorf("en disco = %+v; quería los 2 registros sin migrar", enDisco)
	}
}
