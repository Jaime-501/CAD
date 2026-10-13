//go:build integration

package jkanime

import (
	"context"
	"strings"
	"testing"
)

// Smoke de red: pide las dos secciones del Home al sitio real y comprueba
// que siguen existiendo (el JSON var animes del directorio y las tarjetas
// de la portada). Requiere conectividad.
//
//	go test -tags integration ./internal/scraper/jkanime/ -run HomeRedReal -v
func TestHomeRedReal(t *testing.T) {
	j := New(clientePrueba(), "", nil)

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Home devolvió 0 secciones en el sitio real")
	}
	var conItems int
	for _, s := range got {
		if len(s.Items) == 0 {
			t.Errorf("sección %q sin items", s.ID)
			continue
		}
		conItems++
		for i, a := range s.Items {
			if a.ID == "" || strings.Contains(a.ID, "/") {
				t.Errorf("sección %s item %d: id %q sin formato de slug", s.ID, i, a.ID)
			}
			if a.Provider != providerName {
				t.Errorf("sección %s item %d: Provider = %q, quería %q", s.ID, i, a.Provider, providerName)
			}
		}
		t.Logf("sección %q (%s): %d items, primero: %+v", s.ID, s.Title, len(s.Items), s.Items[0])
	}
	if conItems == 0 {
		t.Fatal("ninguna sección del Home trajo items en el sitio real")
	}
}
