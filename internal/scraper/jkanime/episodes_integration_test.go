//go:build integration

package jkanime

import (
	"context"
	"strings"
	"testing"
	"time"

	"miapp/internal/models"
)

// Smoke de red: pide el detalle real de bleach con streaming y comprueba que
// POST /ajax/episodes/<id>/<n> sigue devolviendo episodios con el formato
// capturado en testdata/episodes_p*.json (sesión, CSRF y paginación incluidos).
// Cancela tras los primeros lotes para no bajarse los 366 episodios.
//
//	go test -tags integration ./internal/scraper/jkanime/ -run RedReal -v
func TestEpisodiosRedReal(t *testing.T) {
	j := New(clientePrueba(), "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lotes := make(chan models.EpisodeBatch, 16)
	detalle, pagina1, err := j.StreamAnimeDetails(ctx, "bleach",
		func(b models.EpisodeBatch) {
			select {
			case lotes <- b:
			default:
			}
		})
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if detalle.ID != "bleach" || detalle.Title == "" {
		t.Errorf("detalle = %+v, quería el detalle de bleach", detalle)
	}
	if detalle.TotalEpisodes <= 0 {
		t.Errorf("TotalEpisodes = %d, quería un total positivo", detalle.TotalEpisodes)
	}
	if len(pagina1) == 0 || len(pagina1) > 16 {
		t.Fatalf("página 1 = %d episodios, quería entre 1 y 16", len(pagina1))
	}
	for _, ep := range pagina1 {
		if !strings.HasPrefix(ep.ID, "jkanime:bleach:episode:") {
			t.Errorf("id %q, quería el formato jkanime:<slug>:episode:<n>", ep.ID)
		}
		if ep.Kind != models.KindEpisode || ep.Number < 1 {
			t.Errorf("episodio = %+v, quería kind episode y número positivo", ep)
		}
		if ep.Thumbnail != "" && !strings.HasPrefix(ep.Thumbnail, "https://") {
			t.Errorf("Thumbnail = %q, quería una URL absoluta", ep.Thumbnail)
		}
	}

	// Dos lotes de episodios (o el Done, si el sitio cambia la paginación).
	lotesConEps := 0
	for lotesConEps < 2 {
		select {
		case b := <-lotes:
			if b.Err != "" {
				t.Fatalf("lote con Err: %s", b.Err)
			}
			if b.AnimeID != "bleach" {
				t.Errorf("AnimeID = %q, quería bleach", b.AnimeID)
			}
			if len(b.Episodes) > 0 {
				lotesConEps++
			}
			if b.Done {
				lotesConEps = 2
			}
		case <-time.After(30 * time.Second):
			t.Fatal("timeout esperando lotes del AJAX en vivo")
		}
	}
	cancel()
	t.Logf("detalle: %s | página 1: %d episodios", detalle.Title, len(pagina1))
}
