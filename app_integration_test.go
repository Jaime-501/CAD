//go:build integration

package main

import (
	"context"
	"testing"
)

// TestBusquedaRedReal es el smoke de T20: monta el núcleo con los proveedores
// REALES (registro jkanime + animeflv) y busca en la red. Comprueba que el
// registro está bien cableado y que gana el proveedor principal; el fallback
// de AnimeFLV tiene su propio smoke en internal/scraper/animeflv. No se
// descarga ningún vídeo (regla 13).
func TestBusquedaRedReal(t *testing.T) {
	n := nuevoNucleoPorDefecto()

	resultados, err := n.buscar(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("búsqueda contra la red real: %v", err)
	}
	if len(resultados) == 0 {
		t.Fatal("la búsqueda de prueba no devolvió ningún resultado")
	}
	if resultados[0].Provider != proveedorPrincipal {
		t.Errorf("el primer resultado es de %q; se esperaba %q (si falla, el principal está caído y actuó el fallback)",
			resultados[0].Provider, proveedorPrincipal)
	}
	for _, r := range resultados {
		if r.ID == "" || r.Title == "" || r.Provider == "" {
			t.Errorf("resultado incompleto: %+v", r)
		}
	}
	t.Logf("%d resultados; el primero: %q (%s)", len(resultados), resultados[0].Title, resultados[0].Provider)
}
