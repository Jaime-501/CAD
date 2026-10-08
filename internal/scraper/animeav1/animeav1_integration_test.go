//go:build integration

package animeav1

import (
	"context"
	"testing"
)

// Smoke de red: busca en el sitio real y comprueba que el __data.json del
// catálogo sigue con el mismo formato que los fixtures de T30. Requiere
// conectividad, con el mismo criterio que jkanime/search_integration_test.go.
//
//	go test -tags integration ./internal/scraper/animeav1/ -v
func TestSearchRedReal(t *testing.T) {
	a := New(clientePrueba(), "")

	got, err := a.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Search devolvió 0 resultados en el sitio real")
	}

	var bleach bool
	for _, r := range got {
		if r.Provider != providerName {
			t.Errorf("Provider = %q, quería %q", r.Provider, providerName)
		}
		if r.ID == "" || r.Title == "" {
			t.Errorf("resultado incompleto: %+v", r)
		}
		if r.ID == "bleach" {
			bleach = true
			if r.Title != "Bleach" {
				t.Errorf("título de bleach = %q", r.Title)
			}
		}
	}
	if !bleach {
		t.Errorf("ningún resultado con slug == \"bleach\" en %d resultados", len(got))
	}
	t.Logf("Resultados: %d | primero: %+v", len(got), got[0])
}
