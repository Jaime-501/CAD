//go:build integration

package animeav1

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
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

// Smoke del detalle (T32): la ficha real de Bleach debe traer su catálogo
// entero en una sola petición. Si cae, cambia el dominio o el formato del
// __data.json, el test lo dirá.
//
//	go test -tags integration ./internal/scraper/animeav1/ -v
func TestDetalleRedReal(t *testing.T) {
	a := New(clientePrueba(), "")

	detalle, episodios, err := a.GetAnimeDetails(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if detalle.ID != "bleach" || detalle.Title != "Bleach" {
		t.Errorf("detalle = %+v, quería la ficha de bleach", detalle)
	}
	if detalle.TotalEpisodes < 366 {
		t.Errorf("TotalEpisodes = %d, quería >= 366", detalle.TotalEpisodes)
	}
	if len(episodios) < 366 {
		t.Errorf("episodios = %d, quería >= 366", len(episodios))
	}
	if detalle.Status == "" || detalle.Type == "" {
		t.Errorf("Status = %q Type = %q: la ficha viene incompleta", detalle.Status, detalle.Type)
	}
	vistos := make(map[string]struct{}, len(episodios))
	for _, ep := range episodios {
		if _, ya := vistos[ep.ID]; ya {
			t.Fatalf("id repetido %q en %d episodios", ep.ID, len(episodios))
		}
		vistos[ep.ID] = struct{}{}
	}
	t.Logf("Detalle: %s | %s | %s | total %d | episodios %d",
		detalle.Title, detalle.Type, detalle.Status, detalle.TotalEpisodes, len(episodios))
}

// Smoke de fuentes (T33): el episodio real de Bleach debe traer el HLS del
// reproductor propio más al menos un iframe, con idioma estructurado en
// Language, y su manifiesto debe responder 200 con Sec-Fetch-Site:
// same-origin (cabecera que exigen los segmentos y que el navegador no
// puede poner). Solo se pide el .m3u8: ni segmentos ni vídeo, y la URL se
// imprime recortada, sin ids completos.
//
//	go test -tags integration ./internal/scraper/animeav1/ -v
func TestFuentesRedReal(t *testing.T) {
	a := New(clientePrueba(), "")

	fuentes, err := a.GetVideoSources(context.Background(), "animeav1:bleach:episode:1", scraper.SourceOpts{})
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	var hls, iframe, conIdioma int
	var manifiesto models.VideoSource
	for _, f := range fuentes {
		if f.Language == models.LanguageSub || f.Language == models.LanguageDub {
			conIdioma++
		}
		if f.Type == models.ResourceHLS {
			hls++
			if manifiesto.URL == "" {
				manifiesto = f
			}
		}
		if f.Type == models.ResourceIframe {
			iframe++
		}
	}
	if hls == 0 {
		t.Fatal("ninguna fuente HLS derivada del reproductor propio")
	}
	if iframe == 0 {
		t.Errorf("ningún iframe en %d fuentes", len(fuentes))
	}
	if conIdioma == 0 {
		t.Errorf("ninguna fuente con Language informado en %d fuentes", len(fuentes))
	}
	if manifiesto.Headers[cabeceraFetch] != valorFetch {
		t.Errorf("Headers = %v, quería %q: sin ella los segmentos dan 403", manifiesto.Headers, cabeceraFetch)
	}

	resp, err := httpx.Get(context.Background(), clientePrueba(), manifiesto.URL, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "sources",
		Header:    http.Header{cabeceraFetch: []string{valorFetch}},
	})
	if err != nil {
		t.Fatalf("el manifiesto no respondió 200: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "mpegurl") {
		t.Errorf("Content-Type = %q, quería un manifiesto HLS", ct)
	}
	cabecera, err := io.ReadAll(io.LimitReader(resp.Body, 8))
	if err != nil {
		t.Fatalf("leyendo el manifiesto: %v", err)
	}
	if !strings.HasPrefix(string(cabecera), "#EXTM3U") {
		t.Errorf("el cuerpo no empieza por #EXTM3U: %q", cabecera)
	}

	t.Logf("Fuentes: %d | HLS %d | iframe %d | con idioma %d | manifiesto: %s",
		len(fuentes), hls, iframe, conIdioma, urlRecortada(manifiesto.URL))
}

// urlRecortada recorta la última posición del path (el id de 32 caracteres
// del reproductor) para no imprimir ids completos en la salida.
func urlRecortada(u string) string {
	parte := strings.Split(u, "/")
	ultimo := parte[len(parte)-1]
	if len(ultimo) > 8 {
		parte[len(parte)-1] = ultimo[:4] + "…"
	}
	return strings.Join(parte, "/")
}
