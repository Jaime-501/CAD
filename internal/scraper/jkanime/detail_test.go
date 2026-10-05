package jkanime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"miapp/internal/httpx"
)

// fixtureDetalle carga un HTML real de testdata (respuesta cruda del
// servidor, sanitizada; ver PROGRESO.md).
func fixtureDetalle(t *testing.T, nombre string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatalf("no pude leer testdata/%s: %v", nombre, err)
	}
	return data
}

func TestFetchDetailSerieCompleta(t *testing.T) {
	html := fixtureDetalle(t, "detail.html")
	srv, peticiones := servidorHTML(t, html, "/bleach/")
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	detalle, meta, err := j.fetchDetail(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("fetchDetail falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}

	esperado := map[string]string{
		"ID":       detalle.ID,
		"Title":    detalle.Title,
		"Provider": detalle.Provider,
		"Type":     detalle.Type,
		"Status":   detalle.Status,
	}
	if esperado["ID"] != "bleach" {
		t.Errorf("ID = %q, quería el slug", esperado["ID"])
	}
	if esperado["Title"] != "Bleach" {
		t.Errorf("Title = %q, quería %q", esperado["Title"], "Bleach")
	}
	if esperado["Provider"] != providerName {
		t.Errorf("Provider = %q, quería %q", esperado["Provider"], providerName)
	}
	if esperado["Type"] != tipoSerie {
		t.Errorf("Type = %q, quería %q (el fixture dice Serie)", esperado["Type"], tipoSerie)
	}
	if esperado["Status"] != "Concluido" {
		t.Errorf("Status = %q, quería %q", esperado["Status"], "Concluido")
	}
	if detalle.TotalEpisodes != 366 {
		t.Errorf("TotalEpisodes = %d, quería 366 (los del fixture)", detalle.TotalEpisodes)
	}
	if detalle.Cover != "https://cdn.jkdesa.com/assets/images/animes/image/bleach.jpg" {
		t.Errorf("Cover = %q, quería la portada absoluta del fixture", detalle.Cover)
	}
	if !strings.HasPrefix(detalle.Synopsis, "Kurosaki Ichigo es un estudiante de instituto de 15 años") {
		t.Errorf("Synopsis = %.60q…, quería la sinopsis del fixture", detalle.Synopsis)
	}

	esperados := []string{"Sobrenatural", "Super Poderes", "Shounen", "Comedia", "Accion"}
	if len(detalle.Genres) != len(esperados) {
		t.Fatalf("Genres = %v, quería %v", detalle.Genres, esperados)
	}
	for i, g := range esperados {
		if detalle.Genres[i] != g {
			t.Errorf("Genres[%d] = %q, quería %q", i, detalle.Genres[i], g)
		}
	}

	if meta.NumericID != "278" {
		t.Errorf("NumericID = %q, quería %q (data-anime del fixture)", meta.NumericID, "278")
	}
	if meta.CSRFToken != "TOKEN_DE_PRUEBA" {
		t.Errorf("CSRFToken = %q, quería el token sanitizado del fixture", meta.CSRFToken)
	}

	// El id numérico y el CSRF no se exponen en models (solo en detailMeta).
	carros, err := json.Marshal(detalle)
	if err != nil {
		t.Fatalf("marshaling del detalle: %v", err)
	}
	if strings.Contains(string(carros), "TOKEN_DE_PRUEBA") {
		t.Error("el CSRF se filtró al JSON de models")
	}
	if strings.Contains(string(carros), "278") {
		t.Error("el id numérico se filtró al JSON de models")
	}
}

func TestFetchDetailPeliculaNormalizaTipo(t *testing.T) {
	html := fixtureDetalle(t, "detail_movie.html")
	slug := "bleach-the-diamonddust-rebellion"
	srv, _ := servidorHTML(t, html, "/"+slug+"/")
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	detalle, meta, err := j.fetchDetail(context.Background(), slug)
	if err != nil {
		t.Fatalf("fetchDetail falló: %v", err)
	}
	if detalle.Type != tipoPelicula {
		t.Errorf("Type = %q, quería %q (el fixture dice Pelicula)", detalle.Type, tipoPelicula)
	}
	if detalle.ID != slug {
		t.Errorf("ID = %q, quería %q", detalle.ID, slug)
	}
	if detalle.Title != "Bleach: The DiamondDust Rebellion" {
		t.Errorf("Title = %q, quería el título del fixture", detalle.Title)
	}
	if detalle.TotalEpisodes != 1 {
		t.Errorf("TotalEpisodes = %d, quería 1", detalle.TotalEpisodes)
	}
	if detalle.Status != "Concluido" {
		t.Errorf("Status = %q, quería %q", detalle.Status, "Concluido")
	}
	if len(detalle.Genres) != 5 {
		t.Errorf("Genres = %v, quería los 5 géneros del fixture", detalle.Genres)
	}
	if !strings.HasSuffix(detalle.Cover, "/bleach-the-diamonddust-rebellion.jpg") {
		t.Errorf("Cover = %q, quería la portada de la película", detalle.Cover)
	}
	if meta.NumericID != "504" {
		t.Errorf("NumericID = %q, quería %q", meta.NumericID, "504")
	}
	if meta.CSRFToken != "TOKEN_DE_PRUEBA" {
		t.Errorf("CSRFToken = %q, quería el token sanitizado del fixture", meta.CSRFToken)
	}
}

func TestNormalizarTipo(t *testing.T) {
	casos := []struct{ entrada, esperado string }{
		{"Serie", tipoSerie},
		{"serie", tipoSerie},
		{"Serie TV", tipoSerie},
		{"Pelicula", tipoPelicula},
		{"Película", tipoPelicula},
		{"  película  ", tipoPelicula},
		{"Movie", tipoPelicula},
		{"OVA", tipoOVA},
		{"ova", tipoOVA},
		{"Ova 2", tipoOVA},
		{"Especial", tipoEspecial},
		{"Special", tipoEspecial},
		{"", tipoDesconocido},
		{"   ", tipoDesconocido},
		{"Documental", tipoDesconocido},
		{"Video musical", tipoDesconocido},
	}
	for _, c := range casos {
		if got := normalizarTipo(c.entrada); got != c.esperado {
			t.Errorf("normalizarTipo(%q) = %q, quería %q", c.entrada, got, c.esperado)
		}
	}
}

func TestFetchDetailSlugVacioNoPideRed(t *testing.T) {
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		t.Error("con un slug vacío no debía hacerse ninguna petición")
		return nil, errors.New("sin red (doer de prueba)")
	}}
	j := New(d, "", nil)

	for _, slug := range []string{"", "   ", "/"} {
		_, _, err := j.fetchDetail(context.Background(), slug)
		if !errors.Is(err, errSlugVacio) {
			t.Errorf("slug %q: err = %v, quería errSlugVacio", slug, err)
		}
	}
}

func TestFetchDetailErrorHTTP(t *testing.T) {
	casos := []struct {
		nombre string
		status int
		cuerpo string
		espera error
	}{
		{"404", http.StatusNotFound, "no existe", httpx.ErrNotFound},
		{"403", http.StatusForbidden, "prohibido", httpx.ErrForbidden},
		{"500", http.StatusInternalServerError, "error interno", httpx.ErrServerError},
		{
			"challenge",
			http.StatusOK,
			"<html><head><title>Just a moment...</title></head><body></body></html>",
			httpx.ErrCloudflareChallenge,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.cuerpo))
			}))
			t.Cleanup(srv.Close)
			j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

			_, _, err := j.fetchDetail(context.Background(), "bleach")
			if err == nil {
				t.Fatal("quería error")
			}
			if !errors.Is(err, c.espera) {
				t.Errorf("err = %v, quería %v", err, c.espera)
			}
		})
	}
}

func TestFetchDetailDemasiadoGrande(t *testing.T) {
	cuerpo := strings.Repeat("a", maxDetailBytes+1)
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(srv.Close)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	_, _, err := j.fetchDetail(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxDetailBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}

func TestFetchDetailCamposAusentesNoSonError(t *testing.T) {
	// Página mínima: sin ficha (.anime_data), sin portada y sin CSRF.
	html := `<!DOCTYPE html><html><head><title>x</title></head><body>
<div class="anime_info"><h3>Solo título</h3><p class="scroll">Sinopsis corta.</p></div>
</body></html>`
	srv, _ := servidorHTML(t, []byte(html), "/corto/")
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	detalle, meta, err := j.fetchDetail(context.Background(), "corto")
	if err != nil {
		t.Fatalf("una página incompleta no debe ser error: %v", err)
	}
	if detalle.Title != "Solo título" || detalle.Synopsis != "Sinopsis corta." {
		t.Errorf("detalle = %+v, quería título y sinopsis", detalle)
	}
	if detalle.Type != tipoDesconocido {
		t.Errorf("Type = %q, quería %q sin la ficha", detalle.Type, tipoDesconocido)
	}
	if detalle.TotalEpisodes != 0 || detalle.Status != "" || detalle.Cover != "" {
		t.Errorf("campos ausentes = %+v, quería los valores cero", detalle)
	}
	if detalle.Genres == nil || len(detalle.Genres) != 0 {
		t.Errorf("Genres = %#v, quería un slice vacío (JSON [], no null)", detalle.Genres)
	}
	if meta.NumericID != "" || meta.CSRFToken != "" {
		t.Errorf("meta = %+v, quería vacíos sin esos datos", meta)
	}
}
