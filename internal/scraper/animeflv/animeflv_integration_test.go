//go:build integration

package animeflv

import (
	"context"
	"strings"
	"testing"

	"miapp/internal/httpx"
)

// Smoke de red: busca en el sitio real (animeflv.or.at) y comprueba que los
// selectores capturados en testdata/search.html siguen vigentes. Requiere
// conectividad y el build tag integration:
//
//	go test -tags integration ./internal/scraper/animeflv/ -run RedReal -v
func TestSearchRedReal(t *testing.T) {
	a := New(httpx.NewClient(httpx.ClientOptions{}), "")

	got, err := a.Search(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("Search falló: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Search devolvió 0 resultados en el sitio real")
	}
	primero := got[0]
	if primero.Provider != providerName {
		t.Errorf("Provider = %q, quería %q", primero.Provider, providerName)
	}
	if primero.ID == "" || primero.Title == "" {
		t.Errorf("primer resultado incompleto: %+v", primero)
	}
	if strings.Contains(primero.ID, "/") || strings.HasPrefix(primero.ID, "http") {
		t.Errorf("id %q debe ser el slug del href, sin barra ni dominio", primero.ID)
	}
	t.Logf("Resultados: %d | primero: %+v", len(got), primero)
}
