package animeav1

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"runtime"
	"testing"
	"time"

	"miapp/internal/models"
)

// respuestaFalsa construye una http.Response para el doerGrabador (sin red).
func respuestaFalsa(req *http.Request, status int, cuerpo []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:       io.NopCloser(bytes.NewReader(cuerpo)),
		Request:    req,
	}
}

// doerDetalle responde cualquier petición con el cuerpo dado: sirve para los
// tests que miden goroutines, donde un httptest.Server añadiría las suyas.
func doerDetalle(cuerpo []byte) *doerGrabador {
	return &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		return respuestaFalsa(req, http.StatusOK, cuerpo), nil
	}}
}

// esperarLotes recibe exactamente n lotes (o falla por tiempo).
func esperarLotes(t *testing.T, lote <-chan models.EpisodeBatch, n int) []models.EpisodeBatch {
	t.Helper()
	recibidos := make([]models.EpisodeBatch, 0, n)
	limite := time.After(10 * time.Second)
	for len(recibidos) < n {
		select {
		case b := <-lote:
			recibidos = append(recibidos, b)
		case <-limite:
			t.Fatalf("solo llegaron %d lotes de %d: %+v", len(recibidos), n, recibidos)
		}
	}
	return recibidos
}

// recojerLotes espera hasta el lote Done (o falla por tiempo).
func recojerLotes(t *testing.T, lote <-chan models.EpisodeBatch) []models.EpisodeBatch {
	t.Helper()
	recibidos := make([]models.EpisodeBatch, 0, 8)
	limite := time.After(10 * time.Second)
	for {
		select {
		case b := <-lote:
			recibidos = append(recibidos, b)
			if b.Done {
				return recibidos
			}
		case <-limite:
			t.Fatalf("se acabó el tiempo esperando el lote Done: %+v", recibidos)
		}
	}
}

// comprobarLote exige el encabezado común de todos los lotes.
func comprobarLote(t *testing.T, b models.EpisodeBatch, animeID string, batch int) {
	t.Helper()
	if b.AnimeID != animeID {
		t.Errorf("lote %d: AnimeID = %q, quería %q (el id del detalle)", b.Batch, b.AnimeID, animeID)
	}
	if b.Batch != batch {
		t.Errorf("Batch = %d, quería %d", b.Batch, batch)
	}
	if b.Err != "" {
		t.Errorf("lote %d: Err = %q, quería transmisión limpia", b.Batch, b.Err)
	}
}

// TestStreamLotesDe50 reparte los 366 episodios de Bleach tal como pide T32:
// los primeros 50 vuelven con el detalle (sin emitirse) y el resto llega en
// lotes de 50 con el último marcado como Done (7 lotes: 6×50 + 16).
func TestStreamLotesDe50(t *testing.T) {
	srv, peticiones := servidorJSON(t, fixture(t, "detail_data.json"), "/media/bleach/__data.json")
	a := New(clientePrueba(), srv.URL)
	lotes := make(chan models.EpisodeBatch, 16)

	detalle, primera, err := a.StreamAnimeDetails(context.Background(), "bleach",
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if detalle.ID != "bleach" || detalle.TotalEpisodes != 366 {
		t.Errorf("detalle = %+v, quería bleach con 366 episodios", detalle)
	}
	if len(primera) != 50 {
		t.Fatalf("primera tanda = %d episodios, quería 50 (los que devuelve el sitio de un tirón)", len(primera))
	}
	comprobarNumeracion(t, primera, 1, 50)

	recibidos := recojerLotes(t, lotes)
	queriaTamanos := []int{50, 50, 50, 50, 50, 50, 16}
	if len(recibidos) != len(queriaTamanos) {
		t.Fatalf("lotes = %d, quería %d (366 - 50 repartidos de 50)", len(recibidos), len(queriaTamanos))
	}
	todos := append([]models.Episode{}, primera...)
	for i, b := range recibidos {
		comprobarLote(t, b, "bleach", i+2)
		if queria := queriaTamanos[i]; len(b.Episodes) != queria {
			t.Errorf("lote %d: %d episodios, quería %d", b.Batch, len(b.Episodes), queria)
		}
		if esperado := i == len(recibidos)-1; b.Done != esperado {
			t.Errorf("lote %d: Done = %v, quería %v", b.Batch, b.Done, esperado)
		}
		todos = append(todos, b.Episodes...)
	}
	if len(todos) != 366 {
		t.Errorf("episodios totales = %d, quería 366 (50 + 6×50 + 16)", len(todos))
	}
	comprobarNumeracion(t, todos, 1, 366)

	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1 (los lotes salen de la memoria)", n)
	}
	select {
	case b := <-lotes:
		t.Errorf("lote posterior al Done: %+v", b)
	case <-time.After(200 * time.Millisecond):
	}
}

// TestStreamPeliculaDeUnEpisodio: cuando todo cabe en la primera tanda no
// hay resto que emitir, así que llega un único lote Done vacío (la entrada
// que cierra episodes:complete).
func TestStreamPeliculaDeUnEpisodio(t *testing.T) {
	slug := "bleach-movie-1-memories-of-nobody"
	srv, _ := servidorJSON(t, fixture(t, "detail_movie_data.json"), "/media/"+slug+"/__data.json")
	a := New(clientePrueba(), srv.URL)
	lotes := make(chan models.EpisodeBatch, 4)

	_, primera, err := a.StreamAnimeDetails(context.Background(), slug,
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if len(primera) != 1 || primera[0].Kind != models.KindMovie {
		t.Fatalf("primera tanda = %+v, quería la película", primera)
	}

	recibidos := recojerLotes(t, lotes)
	if len(recibidos) != 1 {
		t.Fatalf("lotes = %d, quería 1 solo con Done: %+v", len(recibidos), recibidos)
	}
	unico := recibidos[0]
	comprobarLote(t, unico, slug, 1)
	if !unico.Done || len(unico.Episodes) != 0 {
		t.Errorf("lote = %+v, quería Done vacío", unico)
	}
}

// TestStreamFichaSinEpisodios: un detalle sin ningún episodio también cierra
// el stream con un lote Done vacío, sin devolver el detalle "colgado".
func TestStreamFichaSinEpisodios(t *testing.T) {
	cuerpo := []byte(cuerpoSintetico(`"status":4,`, "0"))
	a := New(doerDetalle(cuerpo), DefaultBaseURL)
	lotes := make(chan models.EpisodeBatch, 4)

	detalle, primera, err := a.StreamAnimeDetails(context.Background(), "mi-slug",
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if detalle.ID != "mi-slug" || detalle.TotalEpisodes != 0 {
		t.Errorf("detalle = %+v, quería la ficha sin episodios", detalle)
	}
	if len(primera) != 0 {
		t.Errorf("primera tanda = %d episodios, quería 0", len(primera))
	}

	recibidos := recojerLotes(t, lotes)
	if len(recibidos) != 1 {
		t.Fatalf("lotes = %d, quería 1 solo con Done: %+v", len(recibidos), recibidos)
	}
	comprobarLote(t, recibidos[0], "mi-slug", 1)
	if !recibidos[0].Done || len(recibidos[0].Episodes) != 0 {
		t.Errorf("lote = %+v, quería Done vacío", recibidos[0])
	}
}

// TestStreamSinCallbackNoLanzaGoroutine: sin callback la función devuelve la
// primera tanda y no arranca nada en segundo plano.
func TestStreamSinCallbackNoLanzaGoroutine(t *testing.T) {
	a := New(doerDetalle(fixture(t, "detail_data.json")), DefaultBaseURL)

	// Que se asienten las goroutines de los tests anteriores.
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()

	_, primera, err := a.StreamAnimeDetails(context.Background(), "bleach", nil)
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if len(primera) != 50 {
		t.Errorf("primera tanda = %d episodios, quería 50", len(primera))
	}

	limite := time.Now().Add(10 * time.Second)
	for runtime.NumGoroutine() > base && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > base {
		t.Errorf("goroutines = %d, quería no pasar de %d: sin callback no debe haber emisión", n, base)
	}
}

// TestStreamCancelacionSinFugas cancela el contexto a mitad del reparto (en
// el lote 3): los lotes ya emitidos se quedan emitidos y de ahí en adelante
// no llega nada, ni siquiera el Done. La goroutine termina sin fugas.
func TestStreamCancelacionSinFugas(t *testing.T) {
	a := New(doerDetalle(fixture(t, "detail_data.json")), DefaultBaseURL)

	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lotes := make(chan models.EpisodeBatch, 16)

	_, primera, err := a.StreamAnimeDetails(ctx, "bleach", func(b models.EpisodeBatch) {
		lotes <- b
		if b.Batch == 3 {
			cancel()
		}
	})
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if len(primera) != 50 {
		t.Fatalf("primera tanda = %d episodios, quería 50", len(primera))
	}

	recibidos := esperarLotes(t, lotes, 2)
	for i, b := range recibidos {
		comprobarLote(t, b, "bleach", i+2)
		if b.Done {
			t.Errorf("lote %d llegó con Done tras cancelar", b.Batch)
		}
		if len(b.Episodes) != 50 {
			t.Errorf("lote %d: %d episodios, quería 50", b.Batch, len(b.Episodes))
		}
	}
	todos := append(append([]models.Episode{}, primera...), recibidos[0].Episodes...)
	todos = append(todos, recibidos[1].Episodes...)
	if len(todos) != 150 {
		t.Errorf("episodios = %d, quería 150 (50 + 50 + 50)", len(todos))
	}
	comprobarNumeracion(t, todos, 1, 150)

	select {
	case b := <-lotes:
		t.Errorf("lote posterior a la cancelación: %+v", b)
	case <-time.After(300 * time.Millisecond):
	}

	limite := time.Now().Add(10 * time.Second)
	for runtime.NumGoroutine() > base && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > base {
		t.Errorf("goroutines = %d, quería volver a %d: la goroutine no terminó", n, base)
	}
}
