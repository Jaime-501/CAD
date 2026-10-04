//go:build integration

package scraper

import "testing"

// Smoke test de red: requiere conectividad a animeflv.net.
func TestSearch(t *testing.T) {
	a := NewAnimeFLV()

	results, err := a.Search("bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search devolvió 0 resultados")
	}

	t.Logf("Resultados: %d | primero: %+v", len(results), results[0])
}
