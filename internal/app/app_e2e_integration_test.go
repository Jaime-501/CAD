//go:build integration

package app

import (
	"context"
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
	n := NuevoNucleoPorDefecto(t.TempDir())
	emis := nuevoEmisorFalso()
	if err := n.Arrancar(emis.emitir); err != nil {
		t.Fatalf("arrancando el núcleo: %v", err)
	}
	t.Cleanup(func() { _ = n.Cerrar() })

	// --- 1. Búsqueda ------------------------------------------------------
	resultados, err := n.Buscar(context.Background(), "bleach")
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
	detalle, err := n.Detalle(context.Background(), proveedorPrincipal, slug)
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
	fuentes, err := n.Fuentes(context.Background(), proveedorPrincipal, epID, false)
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
	if err := n.GuardarProgreso(models.WatchProgress{
		AnimeID: slug, EpisodeID: epID, CurrentTime: 90, TotalDuration: 240,
	}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	guardado, err := n.VerProgreso(slug)
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(guardado) != 1 {
		t.Fatalf("GetProgress devolvió %d registros; se esperaba 1", len(guardado))
	}
	if guardado[0].CurrentTime != 90 || guardado[0].TotalDuration != 240 || guardado[0].IsWatched {
		t.Errorf("round-trip del historial: %+v", guardado[0])
	}
	if err := n.GuardarProgreso(models.WatchProgress{
		AnimeID: slug, EpisodeID: epID, CurrentTime: 230, TotalDuration: 240,
	}); err != nil {
		t.Fatalf("SaveProgress (actualización): %v", err)
	}
	guardado, err = n.VerProgreso(slug)
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

// validarURLRecortada es validarURL sin volcar la URL cruda en el fallo: el
// E2E real no imprime URLs con query ni ids completos.
func validarURLRecortada(t *testing.T, contexto, raw string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Errorf("%s: la URL no es parseable (%s): %v", contexto, recortarURL(raw), err)
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		t.Errorf("%s: el esquema de %q no es http/https", contexto, recortarURL(raw))
	}
	if u.Host == "" {
		t.Errorf("%s: %q no tiene host", contexto, recortarURL(raw))
	}
}

// TestE2EAnimeAV1RedReal encadena la misma cadena que el offline de
// AnimeAV1 —búsqueda combinada, detalle por lotes, fuentes y proxy— contra
// la red real: descarga el manifiesto HLS del reproductor propio y UN
// segmento con Range: bytes=0-0 a través del proxy (la reproducción real
// que T35 deja anotada como prueba manual con el .exe). Tolerante a
// cambios de contenido (T28): lo que depende del sitio se anota con
// t.Errorf/t.Log y solo es fatal lo que rompe la cadena; nada de vídeo
// entero (regla 13) y ninguna URL firmada completa en la salida.
func TestE2EAnimeAV1RedReal(t *testing.T) {
	registro := capturarLogs(t)
	n := NuevoNucleoPorDefecto(t.TempDir())
	emis := nuevoEmisorFalso()
	if err := n.Arrancar(emis.emitir); err != nil {
		t.Fatalf("arrancando el núcleo: %v", err)
	}
	t.Cleanup(func() { _ = n.Cerrar() })

	// --- 1. Búsqueda combinada (T34) --------------------------------------
	resultados, err := n.Buscar(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("SearchAnime contra la red real: %v", err)
	}
	if len(resultados) == 0 {
		t.Fatal("la búsqueda no devolvió ningún resultado")
	}
	if resultados[0].Provider != proveedorPrincipal {
		t.Errorf("el primer resultado es de %q; se esperaba %q (JKAnime encabeza la lista combinada)",
			resultados[0].Provider, proveedorPrincipal)
	}
	av1, hayBleach := 0, false
	for _, r := range resultados {
		if r.ID == "" || r.Title == "" {
			t.Errorf("resultado incompleto: %+v", r)
			continue
		}
		if r.CoverURL == "" {
			t.Logf("resultado %q (%s) sin portada", r.ID, r.Provider)
		} else {
			validarURLRecortada(t, "portada de "+r.Provider+":"+r.ID, r.CoverURL)
		}
		if r.Provider == proveedorComplemento {
			av1++
			if r.ID == "bleach" {
				hayBleach = true
			}
		}
	}
	if av1 == 0 {
		t.Error("la búsqueda no devolvió NINGÚN resultado de animeav1: su búsqueda está caída o cambió")
	} else if !hayBleach {
		t.Errorf("ningún resultado de animeav1 con slug bleach entre los %d; se continúa con bleach (fijo desde T32/T33)", av1)
	}
	slug := "bleach"
	t.Logf("búsqueda: %d resultados (%d de animeav1); se sigue %q", len(resultados), av1, slug)

	// --- 2. Detalle con streaming y eventos -------------------------------
	detalle, err := n.Detalle(context.Background(), proveedorComplemento, slug)
	if err != nil {
		t.Fatalf("GetAnimeDetails(animeav1): %v", err)
	}
	if detalle.Detail.ID == "" || detalle.Detail.Title == "" || detalle.Detail.Provider != proveedorComplemento {
		t.Errorf("detalle incompleto: %+v", detalle.Detail)
	}
	if detalle.Detail.Cover != "" {
		validarURLRecortada(t, "portada del detalle", detalle.Detail.Cover)
	}
	if len(detalle.Episodes) == 0 {
		t.Fatal("la primera tanda de episodios llegó vacía: sin episodios no hay E2E")
	}
	for _, ep := range detalle.Episodes {
		validarEpisodioDe(t, "primera tanda", ep, proveedorComplemento)
	}
	if detalle.Detail.TotalEpisodes < 366 {
		t.Errorf("TotalEpisodes = %d; bleach declaraba >= 366 (T32)", detalle.Detail.TotalEpisodes)
	}
	if len(detalle.Episodes) != 50 {
		t.Errorf("la primera tanda trae %d episodios; se esperaban 50 (tamLote de episodes.go)", len(detalle.Episodes))
	}
	t.Logf("detalle %q (%s, %s, %d declarados); %d episodios en la primera tanda",
		detalle.Detail.Title, detalle.Detail.Type, detalle.Detail.Status,
		detalle.Detail.TotalEpisodes, len(detalle.Episodes))

	if esperarCompleto(t, emis, slug, 90*time.Second) {
		lotes := lotesDe(t, emis.lista(), slug)
		conEpisodios, final := 0, false
		for i, l := range lotes {
			if l.AnimeID != slug {
				t.Errorf("lote %d con AnimeID %q; se esperaba %q", l.Batch, l.AnimeID, slug)
			}
			if l.Batch != i+2 {
				t.Errorf("lote con Batch %d en la posición %d; se esperaba %d", l.Batch, i, i+2)
			}
			if len(l.Episodes) > 50 {
				t.Errorf("lote %d trae %d episodios; el máximo por lote es 50", l.Batch, len(l.Episodes))
			}
			if len(l.Episodes) > 0 {
				conEpisodios++
				for _, ep := range l.Episodes {
					validarEpisodioDe(t, "lote emitido", ep, proveedorComplemento)
				}
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
	fuentes, err := n.Fuentes(context.Background(), proveedorComplemento, epID, false)
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
	hls := 0
	for _, f := range fuentes {
		if f.Server == "" {
			t.Errorf("fuente sin etiqueta de servidor: %+v", f)
		}
		validarURLRecortada(t, "fuente "+f.Server, f.URL)
		validarTipo(t, "fuente "+f.Server, f.Type)
		u, parseErr := url.Parse(f.URL)
		enProxy := parseErr == nil && u.Path == "/proxy" && u.Host == hostDe(baseProxy)
		if f.Type == models.ResourceHLS || f.Type == models.ResourceMP4 {
			if !enProxy {
				t.Errorf("la fuente %s (%s) no pasó por el proxy", f.Server, f.Type)
			}
		} else if enProxy {
			t.Errorf("la fuente %s (%s) no debía pasar por el proxy", f.Server, f.Type)
		}
		if f.Type == models.ResourceHLS {
			hls++
		}
	}
	if hls == 0 {
		t.Fatalf("ninguna fuente HLS en %d fuentes: no hay manifiesto ni segmento que probar\n%s",
			len(fuentes), resumenFuentes(fuentes))
	}
	mejor, ok := extractors.SelectBestSource(fuentes)
	if !ok {
		t.Fatal("SelectBestSource no eligió ninguna fuente con la lista devuelta")
	}
	if mejor.URL != fuentes[0].URL {
		t.Errorf("SelectBestSource eligió la posición %d de %d; se esperaba la primera",
			indiceDe(fuentes, mejor)+1, len(fuentes))
	}
	if mejor.Type != models.ResourceHLS {
		t.Fatalf("la mejor fuente es %s (%s); sin HLS no hay manifiesto que probar\n%s",
			mejor.Type, mejor.Server, resumenFuentes(fuentes))
	}
	t.Logf("fuentes: %d; mejor = %s (%s, %s) vía proxy\n%s",
		len(fuentes), mejor.Type, mejor.Server, mejor.Language, resumenFuentes(fuentes))

	// --- 4. Manifiesto real y UN segmento con Range a través del proxy -----
	cliente := &http.Client{Timeout: 45 * time.Second}
	resp, err := cliente.Get(mejor.URL)
	if err != nil {
		t.Fatalf("GET del manifiesto real a través del proxy: %v", recortarError(err, mejor.URL))
	}
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err != nil {
		t.Fatalf("leyendo el manifiesto real: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("el proxy respondió %d para el manifiesto real", resp.StatusCode)
	}
	if !strings.HasPrefix(string(cuerpo), "#EXTM3U") {
		t.Fatalf("el proxy no devolvió un manifiesto HLS: %.120q", cuerpo)
	}
	t.Logf("proxy: manifiesto real de %d bytes (%s); el segmento se pide con Range bytes=0-0",
		len(cuerpo), resp.Header.Get("Content-Type"))

	// Un manifiesto maestro trae sub-playlists: subo un nivel como mucho dos
	// veces hasta dar con la URI de un segmento.
	uri := primeraURIDelManifiesto(string(cuerpo))
	for nivel := 0; uri != ""; nivel++ {
		destino := destinoProxy(t, uri)
		du, parseErr := url.Parse(destino)
		if parseErr != nil {
			t.Errorf("URI del manifiesto ilegible (%s): %v", recortarURL(destino), parseErr)
			uri = ""
			break
		}
		if nivel >= 2 || !strings.HasSuffix(strings.ToLower(du.Path), ".m3u8") {
			break // ya apunta a un segmento (o se agotaron los niveles)
		}
		sub, err := cliente.Get(uri)
		if err != nil {
			t.Errorf("GET del sub-manifiesto real: %v", recortarError(err, uri))
			uri = ""
			break
		}
		subCuerpo, err := io.ReadAll(io.LimitReader(sub.Body, 1<<20))
		sub.Body.Close()
		if err != nil || sub.StatusCode != http.StatusOK || !strings.HasPrefix(string(subCuerpo), "#EXTM3U") {
			t.Errorf("el sub-manifiesto real no es un HLS legible (status %d, %d bytes)",
				sub.StatusCode, len(subCuerpo))
			uri = ""
			break
		}
		t.Logf("sub-manifiesto real de %d bytes", len(subCuerpo))
		uri = primeraURIDelManifiesto(string(subCuerpo))
	}
	if uri == "" {
		t.Error("el manifiesto no trae ninguna URI de segmento utilizable")
	} else {
		validarURLRecortada(t, "segmento real", uri)
		reqRango, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			t.Fatalf("preparando la petición con Range: %v", err)
		}
		reqRango.Header.Set("Range", "bytes=0-0")
		respRango, err := cliente.Do(reqRango)
		if err != nil {
			t.Fatalf("GET del segmento real a través del proxy: %v", recortarError(err, uri))
		}
		leido, err := io.ReadAll(io.LimitReader(respRango.Body, 64<<10))
		respRango.Body.Close()
		if err != nil {
			t.Fatalf("leyendo el segmento real: %v", err)
		}
		if respRango.StatusCode != http.StatusOK && respRango.StatusCode != http.StatusPartialContent {
			t.Errorf("el proxy respondió %d para el segmento real con Range; se esperaba 200 o 206",
				respRango.StatusCode)
		}
		if len(leido) == 0 {
			t.Error("el segmento real llegó vacío")
		}
		cr := respRango.Header.Get("Content-Range")
		if respRango.StatusCode == http.StatusPartialContent && cr == "" {
			t.Error("206 sin cabecera Content-Range")
		}
		t.Logf("segmento real: %d bytes leídos con Range bytes=0-0 → %d (Content-Range: %s)",
			len(leido), respRango.StatusCode, cr)
	}

	// --- 5. Los logs del flujo real no dejan secretos ni URLs firmadas ---
	comprobarLogsSinSecretos(t, registro)
}
