//go:build integration

package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"miapp/internal/models"
	"miapp/internal/scraper/jkanime/extractors"
)

// TestE2EFlujoCompletoRedReal encadena SearchAnime → GetAnimeDetails (con
// streaming de episodios y eventos) → GetVideoSources → SelectBestSource →
// petición HTTP real al manifiesto a través del proxy local (T19), la misma
// cadena que TestE2EFlujoCompletoOffline pero contra jkanime real.
//
// Tolerante a cambios de contenido (T28): lo que depende del sitio se anota
// con t.Errorf/t.Log (portadas, número de episodios, tipo de la mejor fuente)
// y solo es fatal lo que rompe la cadena del E2E. Del HLS se pide únicamente
// el manifiesto: ningún segmento se descarga (regla 13).
func TestE2EFlujoCompletoRedReal(t *testing.T) {
	registro := capturarLogs(t) // T29: la salida real tampoco filtra secretos
	n := nuevoNucleoPorDefecto()
	n.rutaHistorial = t.TempDir() // el historial del E2E vive en un temporal, no en el perfil
	emis := nuevoEmisorFalso()
	if err := n.arrancar(emis.emitir); err != nil {
		t.Fatalf("arrancando el núcleo: %v", err)
	}
	t.Cleanup(func() { _ = n.cerrar() })
	a := &App{core: n}

	// --- 1. Búsqueda ------------------------------------------------------
	resultados, err := a.SearchAnime("bleach")
	if err != nil {
		t.Fatalf("SearchAnime contra la red real: %v", err)
	}
	if len(resultados) == 0 {
		t.Fatal("la búsqueda de prueba no devolvió ningún resultado")
	}
	if resultados[0].Provider != proveedorPrincipal {
		t.Errorf("el primer resultado es de %q; se esperaba %q (si falla, el principal está caído y actuó el fallback)",
			resultados[0].Provider, proveedorPrincipal)
	}
	slug := ""
	for _, r := range resultados {
		if r.ID == "" || r.Title == "" {
			t.Errorf("resultado incompleto: %+v", r)
			continue
		}
		if r.CoverURL == "" {
			t.Logf("resultado %q sin portada", r.ID)
		} else {
			validarURL(t, "portada de "+r.ID, r.CoverURL)
		}
		if r.ID == "bleach" {
			slug = r.ID
		}
	}
	if slug == "" {
		slug = resultados[0].ID
		t.Logf("no está el slug bleach entre los %d resultados; se continúa con el primero %q", len(resultados), slug)
	}
	t.Logf("búsqueda: %d resultados; se sigue %q", len(resultados), slug)

	// --- 2. Detalle con streaming y eventos -------------------------------
	detalle, err := a.GetAnimeDetails(proveedorPrincipal, slug)
	if err != nil {
		t.Fatalf("GetAnimeDetails: %v", err)
	}
	if detalle.Detail.ID == "" || detalle.Detail.Title == "" {
		t.Errorf("detalle incompleto: %+v", detalle.Detail)
	}
	if detalle.Detail.Cover != "" {
		validarURL(t, "portada del detalle", detalle.Detail.Cover)
	}
	if len(detalle.Episodes) == 0 {
		t.Fatal("la primera tanda de episodios llegó vacía: sin episodios no hay E2E")
	}
	for _, ep := range detalle.Episodes {
		validarEpisodio(t, "primera tanda", ep)
	}
	t.Logf("detalle %q (%s); %d episodios en la primera tanda",
		detalle.Detail.Title, detalle.Detail.Type, len(detalle.Episodes))

	if esperarCompleto(t, emis, slug, 90*time.Second) {
		lotes := lotesDe(t, emis.lista(), slug)
		conEpisodios, final := 0, false
		for _, l := range lotes {
			if len(l.Episodes) > 0 {
				conEpisodios++
			}
			if l.Done && l.Err == "" {
				final = true
			}
		}
		if conEpisodios == 0 {
			t.Error("no se emitió ningún lote con episodios")
		}
		if !final {
			t.Error("falta el lote final Done sin error")
		}
		if completados := completadosDe(t, emis.lista(), slug); len(completados) == 0 || completados[0].Error != "" {
			t.Errorf("episodes:complete = %+v; quería un error vacío", completados)
		}
		t.Logf("eventos: %d lotes (con episodios: %d)", len(lotes), conEpisodios)
	} else {
		t.Error("no llegó episodes:complete en 90 s: el stream de episodios no cerró")
	}

	// --- 3. Fuentes de vídeo y elección de la mejor -----------------------
	epID := detalle.Episodes[0].ID
	fuentes, err := a.GetVideoSources(proveedorPrincipal, epID, false)
	if err != nil {
		t.Fatalf("GetVideoSources: %v", err)
	}
	if len(fuentes) == 0 {
		t.Fatal("GetVideoSources no devolvió ninguna fuente")
	}
	baseProxy, err := n.proxy.Escuchar() // idempotente: arrancar ya la escuchó
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	for _, f := range fuentes {
		if f.Server == "" {
			t.Errorf("fuente sin etiqueta de servidor: %+v", f)
		}
		validarURL(t, "fuente "+f.Server, f.URL)
		validarTipo(t, "fuente "+f.Server, f.Type)
		u, parseErr := url.Parse(f.URL)
		enProxy := parseErr == nil && u.Path == "/proxy" && u.Host == hostDe(baseProxy)
		if f.Type == models.ResourceHLS || f.Type == models.ResourceMP4 {
			if !enProxy {
				t.Errorf("la fuente %s (%s) no pasó por el proxy: %q", f.Server, f.Type, f.URL)
			}
		} else if enProxy {
			t.Errorf("la fuente %s (%s) no debía pasar por el proxy: %q", f.Server, f.Type, f.URL)
		}
	}
	mejor, ok := extractors.SelectBestSource(fuentes)
	if !ok {
		t.Fatal("SelectBestSource no eligió ninguna fuente con la lista devuelta")
	}
	if mejor.URL != fuentes[0].URL {
		t.Errorf("SelectBestSource eligió %q pero GetVideoSources la trajo en la posición %d de %d",
			mejor.URL, indiceDe(fuentes, mejor)+1, len(fuentes))
	}
	t.Logf("fuentes: %d; mejor = %s (%s)\n%s",
		len(fuentes), mejor.Type, mejor.Server, resumenFuentes(fuentes))

	// --- 4. Petición HTTP real a través del proxy -------------------------
	switch mejor.Type {
	case models.ResourceHLS:
		cliente := &http.Client{Timeout: 45 * time.Second}
		resp, err := cliente.Get(mejor.URL)
		if err != nil {
			t.Errorf("GET del manifiesto real a través del proxy: %v", err)
			break
		}
		cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			t.Errorf("leyendo el manifiesto real: %v", err)
			break
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("el proxy respondió %d para el manifiesto real", resp.StatusCode)
			break
		}
		if !strings.HasPrefix(string(cuerpo), "#EXTM3U") {
			t.Errorf("el proxy no devolvió un manifiesto HLS: %.120q", cuerpo)
			break
		}
		t.Logf("proxy: manifiesto real de %d bytes (%s); los segmentos no se descargan (regla 13)",
			len(cuerpo), resp.Header.Get("Content-Type"))
	case models.ResourceMP4:
		t.Logf("la mejor fuente es MP4: no se descarga (regla 13)")
	default:
		t.Errorf("solo se obtuvieron iframes: no hay fuente directa reproducible\n%s", resumenFuentes(fuentes))
	}

	// --- 5. Historial de visualización (T22) ------------------------------
	if err := a.SaveProgress(models.WatchProgress{
		AnimeID: slug, EpisodeID: epID, CurrentTime: 90, TotalDuration: 240,
	}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	guardado, err := a.GetProgress(slug)
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(guardado) != 1 {
		t.Fatalf("GetProgress devolvió %d registros; se esperaba 1", len(guardado))
	}
	if guardado[0].CurrentTime != 90 || guardado[0].TotalDuration != 240 || guardado[0].IsWatched {
		t.Errorf("round-trip del historial: %+v", guardado[0])
	}
	if err := a.SaveProgress(models.WatchProgress{
		AnimeID: slug, EpisodeID: epID, CurrentTime: 230, TotalDuration: 240,
	}); err != nil {
		t.Fatalf("SaveProgress (actualización): %v", err)
	}
	guardado, err = a.GetProgress(slug)
	if err != nil {
		t.Fatalf("GetProgress (2): %v", err)
	}
	if len(guardado) != 1 || !guardado[0].IsWatched {
		t.Errorf("el upsert no actualizó en el sitio: %+v", guardado)
	}
	t.Logf("historial: round-trip de %d registro(s) en un temporal", len(guardado))

	// --- 6. Los logs del flujo real no dejan secretos ni URLs firmadas ---
	comprobarLogsSinSecretos(t, registro)
}
