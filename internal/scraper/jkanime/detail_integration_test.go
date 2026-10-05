//go:build integration

package jkanime

import (
	"context"
	"strconv"
	"testing"
)

// Smoke de red: descarga el detalle real de una serie y de una película y
// comprueba que los selectores capturados en testdata/detail*.html siguen
// vigentes. Tolera cambios de contenido (no fija episodios, géneros ni
// sinopsis): solo pide que los campos vengan rellenos y con forma válida.
//
//	go test -tags integration ./internal/scraper/jkanime/ -run RedReal -v
func TestDetalleRedReal(t *testing.T) {
	j := New(clientePrueba(), "", nil)
	casos := []struct {
		slug, tipo string
	}{
		{"bleach", tipoSerie},
		{"bleach-the-diamonddust-rebellion", tipoPelicula},
	}
	for _, c := range casos {
		t.Run(c.slug, func(t *testing.T) {
			detalle, meta, err := j.fetchDetail(context.Background(), c.slug)
			if err != nil {
				t.Fatalf("fetchDetail falló: %v", err)
			}
			if detalle.ID != c.slug {
				t.Errorf("ID = %q, quería el slug %q", detalle.ID, c.slug)
			}
			if detalle.Provider != providerName {
				t.Errorf("Provider = %q, quería %q", detalle.Provider, providerName)
			}
			if detalle.Title == "" {
				t.Error("Title vacío")
			}
			if detalle.Type != c.tipo {
				t.Errorf("Type = %q, quería %q", detalle.Type, c.tipo)
			}
			if detalle.Synopsis == "" {
				t.Error("Synopsis vacía")
			}
			if detalle.Cover == "" {
				t.Error("Cover vacía")
			}
			if detalle.Status == "" {
				t.Error("Status vacío")
			}
			if len(detalle.Genres) == 0 {
				t.Error("Genres sin ningún género")
			}
			if detalle.TotalEpisodes <= 0 {
				t.Errorf("TotalEpisodes = %d, quería un total positivo", detalle.TotalEpisodes)
			}
			if _, err := strconv.Atoi(meta.NumericID); err != nil || meta.NumericID == "" {
				t.Errorf("NumericID = %q, quería un id numérico", meta.NumericID)
			}
			if meta.CSRFToken == "" {
				t.Error("CSRFToken vacío")
			}
			t.Logf("detalle: %+v | meta: {NumericID:%s CSRFToken:len=%d}", detalle, meta.NumericID, len(meta.CSRFToken))
		})
	}
}
