//go:build integration

package jkanime

import (
	"context"
	"strings"
	"testing"

	"miapp/internal/scraper"
	"miapp/internal/scraper/jkanime/extractors"
)

// Smoke de red: GetVideoSources sobre un episodio real de bleach con la
// cadena completa (JKPlayer → JKPHP → Servers + MediaFire). Imprime, por
// extractor, cuántas fuentes devolvió y de qué tipo (hls/mp4/iframe).
// No descarga ningún vídeo: solo se piden HTML.
//
//	go test -tags integration ./internal/scraper/jkanime/ -run RedReal -v
func TestGetVideoSourcesRedReal(t *testing.T) {
	const id = "jkanime:bleach:episode:1"
	ctx := context.Background()
	j := New(clientePrueba(), "", nil)

	fuentes, err := j.GetVideoSources(ctx, id, scraper.SourceOpts{})
	t.Logf("fuentes por extractor:\n%s", resumenPorExtractor(fuentes))
	t.Logf("total: %d fuente(s)", len(fuentes))

	if err != nil || len(fuentes) == 0 {
		// Diagnóstico con detalle: sin forzar el test, dejamos constancia
		// de en qué punto se rompe el flujo real (PROGRESO.md lo recoge).
		t.Logf("diagnóstico del flujo real (err = %v):", err)
		pagina, errPagina := j.fetchEpisodePage(ctx, id)
		if errPagina != nil {
			t.Logf("  fetchEpisodePage también falló: %v", errPagina)
		} else {
			t.Logf("  página %s — %d bytes", pagina.URL, len(pagina.HTML))
			t.Logf("  marcas: jkplayer/um=%v · jk.php/jkplayer/jk=%v · var servers=%v",
				strings.Contains(pagina.HTML, "jkplayer/um"),
				strings.Contains(pagina.HTML, "jk.php") || strings.Contains(pagina.HTML, "jkplayer/jk?"),
				strings.Contains(pagina.HTML, "var servers"))
			if err == nil && len(fuentes) == 0 {
				t.Log("  la página llegó pero ningún extractor aportó fuentes (¿selectores cambiados?)")
			}
		}
	}
	if err != nil {
		t.Fatalf("GetVideoSources falló contra el sitio real: %v", err)
	}
	if len(fuentes) == 0 {
		t.Fatal("GetVideoSources devolvió 0 fuentes contra el sitio real")
	}

	mejor, ok := extractors.SelectBestSource(fuentes)
	if !ok {
		t.Fatal("SelectBestSource no encontró ninguna fuente reproducible")
	}
	t.Logf("mejor fuente: tipo=%s servidor=%s calidad=%q url=%s",
		tipoOMismo(mejor.Type), mejor.Server, mejor.Quality, mejor.URL)
}
