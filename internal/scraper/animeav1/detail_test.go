package animeav1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// cuerpoSintetico arma un __data.json devalue a mano. Es el único caso del
// paquete con payload artificial: hace falta porque ningún fixture real trae
// un status fuera del catálogo 0/1/2 (se anota como desviación en
// PROGRESO.md, regla 11). Respeta el formato real: las cadenas van sueltas
// en el array plano y los objetos solo guardan referencias por índice.
//
//	claveStatus  `"status":4,` cuando el campo existe (vacío = ausente)
//	valorStatus  literal de la posición 4 del array: "7", "null", "0.5"…
//
// El resto de slots: title "Prueba", slug "mi-slug", episodesCount 0 (0
// episodios), genres [], category tv-anime, episodes [] e id 123.
func cuerpoSintetico(claveStatus, valorStatus string) string {
	return fmt.Sprintf(`{"type":"data","nodes":[{"type":"data","data":[
 {"media":1},
 {"title":2,"slug":3,%s"episodesCount":5,"genres":6,"category":7,"episodes":8,"synopsis":9,"id":10},
 "Prueba","mi-slug",%s,0,[],{"id":11,"name":12,"slug":13,"malId":14},[],null,123,1,"TV Anime","tv-anime","TV"]}]}`,
		claveStatus, valorStatus)
}

func TestDetalleRutaUserAgentYSlug(t *testing.T) {
	var ruta, ua string
	var peticiones int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones++
		ruta = r.URL.EscapedPath()
		ua = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "detail_data.json"))
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	// El id llega como lo pinta la UI (a veces con barra o espacios): se
	// recorta antes de montar la ruta.
	_, episodios, err := a.GetAnimeDetails(context.Background(), "  /bleach/  ")
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if peticiones != 1 {
		t.Errorf("peticiones = %d, quería 1", peticiones)
	}
	if ruta != "/media/bleach/__data.json" {
		t.Errorf("ruta pedida = %q, quería /media/bleach/__data.json", ruta)
	}
	if ua != httpx.DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería el de httpx", ua)
	}
	if len(episodios) != 366 {
		t.Errorf("episodios = %d, quería 366", len(episodios))
	}
}

func TestDetalleBleach(t *testing.T) {
	srv, peticiones := servidorJSON(t, fixture(t, "detail_data.json"), "/media/bleach/__data.json")
	a := New(clientePrueba(), srv.URL)

	detalle, episodios, err := a.GetAnimeDetails(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1 (una sola petición para detalle y episodios)", n)
	}
	if detalle.ID != "bleach" || detalle.Title != "Bleach" || detalle.Provider != providerName {
		t.Errorf("detalle = %+v, quería id=bleach title=Bleach provider=animeav1", detalle)
	}
	if detalle.Type != "tv-anime" || detalle.Status != "Finalizado" {
		t.Errorf("Type = %q Status = %q, quería tv-anime / Finalizado", detalle.Type, detalle.Status)
	}
	if detalle.TotalEpisodes != 366 {
		t.Errorf("TotalEpisodes = %d, quería 366 (episodesCount del sitio)", detalle.TotalEpisodes)
	}
	if detalle.Cover != "https://cdn.animeav1.com/covers/195.jpg" {
		t.Errorf("Cover = %q, quería la portada del CDN", detalle.Cover)
	}
	if !strings.HasPrefix(detalle.Synopsis, "Ichigo Kurosaki") {
		t.Errorf("Synopsis = %.40q, quería la sinopsis real", detalle.Synopsis)
	}
	if queria := []string{"Acción", "Aventura", "Sobrenatural", "Shounen"}; !slices.Equal(detalle.Genres, queria) {
		t.Errorf("Genres = %v, quería %v", detalle.Genres, queria)
	}

	if len(episodios) != 366 {
		t.Fatalf("episodios = %d, quería los 366 del detalle en una sola petición", len(episodios))
	}
	primero, ultimo := episodios[0], episodios[len(episodios)-1]
	if primero.ID != "animeav1:bleach:episode:1" || primero.Number != 1 || primero.Kind != models.KindEpisode {
		t.Errorf("primer episodio = %+v, quería episode:1", primero)
	}
	if primero.Title != "Bleach - 1" {
		t.Errorf("título = %q, quería el patrón de JKAnime", primero.Title)
	}
	if queria := "https://cdn.animeav1.com/screenshots/195/1.jpg"; primero.Thumbnail != queria {
		t.Errorf("Thumbnail = %q, quería %q", primero.Thumbnail, queria)
	}
	if ultimo.ID != "animeav1:bleach:episode:366" || ultimo.Number != 366 {
		t.Errorf("último episodio = %+v, quería episode:366", ultimo)
	}
	comprobarNumeracion(t, episodios, 1, 366)
}

func TestDetallePeliculaDeUnEpisodio(t *testing.T) {
	slug := "bleach-movie-1-memories-of-nobody"
	srv, _ := servidorJSON(t, fixture(t, "detail_movie_data.json"), "/media/"+slug+"/__data.json")
	a := New(clientePrueba(), srv.URL)

	detalle, episodios, err := a.GetAnimeDetails(context.Background(), slug)
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if detalle.Type != "pelicula" || detalle.TotalEpisodes != 1 {
		t.Errorf("Type = %q TotalEpisodes = %d, quería pelicula / 1", detalle.Type, detalle.TotalEpisodes)
	}
	if len(episodios) != 1 {
		t.Fatalf("episodios = %d, quería 1", len(episodios))
	}
	peli := episodios[0]
	if peli.Kind != models.KindMovie {
		t.Errorf("Kind = %q, quería %q", peli.Kind, models.KindMovie)
	}
	if peli.ID != "animeav1:"+slug+":movie" {
		t.Errorf("ID = %q, quería el id de película sin número", peli.ID)
	}
	// Number 0 para KindMovie: así la tarjeta pinta "Película" y no "Ep. 1"
	// (contrato de models.Episode; el número del sitio sigue en el thumbnail).
	if peli.Number != 0 {
		t.Errorf("Number = %d, quería 0 para una película", peli.Number)
	}
	if queria := "https://cdn.animeav1.com/screenshots/905/1.jpg"; peli.Thumbnail != queria {
		t.Errorf("Thumbnail = %q, quería %q (el sitio numera su película desde 1)", peli.Thumbnail, queria)
	}
	if peli.Title != "Bleach Movie 1: Memories of Nobody" {
		t.Errorf("Title = %q, quería el título de la película", peli.Title)
	}
}

func TestDetallePeliculaDeVariosEpisodios(t *testing.T) {
	// El sitio sí tiene películas con más de un episodio: el id movie no
	// admite número, así que esos casos son episodios normales.
	slug := "girls-band-cry-movie"
	srv, _ := servidorJSON(t, fixture(t, "detail_girls-band-cry-movie_data.json"), "/media/"+slug+"/__data.json")
	a := New(clientePrueba(), srv.URL)

	detalle, episodios, err := a.GetAnimeDetails(context.Background(), slug)
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if detalle.Type != "pelicula" || detalle.TotalEpisodes != 2 {
		t.Errorf("Type = %q TotalEpisodes = %d, quería pelicula / 2", detalle.Type, detalle.TotalEpisodes)
	}
	if len(episodios) != 2 {
		t.Fatalf("episodios = %d, quería 2", len(episodios))
	}
	comprobarNumeracion(t, episodios, 1, 2)
	for i, ep := range episodios {
		if ep.Kind != models.KindEpisode {
			t.Errorf("episodio %d: Kind = %q, quería %q (colisión de ids movie)", ep.Number, ep.Kind, models.KindEpisode)
		}
		if esperado := fmt.Sprintf("animeav1:%s:episode:%d", slug, i+1); ep.ID != esperado {
			t.Errorf("episodio %d: ID = %q, quería %q", i+1, ep.ID, esperado)
		}
	}
}

func TestDetalleEstadosYTipos(t *testing.T) {
	casos := []struct {
		nombre, archivo, slug, estado, tipo string
		total, episodios                    int
		kind                                models.EpisodeKind
	}{
		{"status 1 pinta Próximamente", "detail_one-piece_data.json", "one-piece", "Próximamente", "tv-anime", 1180, 1180, models.KindEpisode},
		{"status 2 pinta En emisión", "detail_battle-spirits-re-zekkai-no-kuu_data.json", "battle-spirits-re-zekkai-no-kuu", "En emisión", "tv-anime", 1, 1, models.KindEpisode},
		{"status 0 pinta Finalizado en OVA", "detail_ajin-ova_data.json", "ajin-ova", "Finalizado", "ova", 3, 3, models.KindEpisode},
		{"especial", "detail_boku-no-hero-academia-more_data.json", "boku-no-hero-academia-more", "Finalizado", "especial", 1, 1, models.KindSpecial},
		{"slug con episode-0 no es el número", "detail_one-piece-film-strong-world-episode-0_data.json", "one-piece-film-strong-world-episode-0", "Finalizado", "ova", 1, 1, models.KindEpisode},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, _ := servidorJSON(t, fixture(t, c.archivo), "/media/"+c.slug+"/__data.json")
			a := New(clientePrueba(), srv.URL)

			detalle, episodios, err := a.GetAnimeDetails(context.Background(), c.slug)
			if err != nil {
				t.Fatalf("GetAnimeDetails falló: %v", err)
			}
			if detalle.Status != c.estado {
				t.Errorf("Status = %q, quería %q", detalle.Status, c.estado)
			}
			if detalle.Type != c.tipo {
				t.Errorf("Type = %q, quería %q", detalle.Type, c.tipo)
			}
			if detalle.TotalEpisodes != c.total {
				t.Errorf("TotalEpisodes = %d, quería %d", detalle.TotalEpisodes, c.total)
			}
			if len(episodios) != c.episodios {
				t.Fatalf("episodios = %d, quería %d", len(episodios), c.episodios)
			}
			if episodios[0].Kind != c.kind {
				t.Errorf("Kind = %q, quería %q", episodios[0].Kind, c.kind)
			}
			if queria := fmt.Sprintf("animeav1:%s:%s:1", c.slug, c.kind); episodios[0].ID != queria {
				t.Errorf("ID = %q, quería %q", episodios[0].ID, queria)
			}
			if episodios[0].Thumbnail == "" {
				t.Errorf("Thumbnail vacío: quería la miniatura del CDN")
			}
		})
	}
}

func TestDetalleEstadoDesconocido(t *testing.T) {
	casos := []struct {
		nombre, clave, valor string
	}{
		{"número fuera del catálogo", `"status":4,`, "7"},
		{"status null", `"status":4,`, "null"},
		{"status con fracción", `"status":4,`, "0.5"},
		{"campo ausente", "", "7"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cuerpo := []byte(cuerpoSintetico(c.clave, c.valor))
			srv, _ := servidorJSON(t, cuerpo, "/media/mi-slug/__data.json")
			a := New(clientePrueba(), srv.URL)

			detalle, episodios, err := a.GetAnimeDetails(context.Background(), "mi-slug")
			if err != nil {
				t.Fatalf("un status ilegible no debe ser error: %v", err)
			}
			if detalle.Status != estadoDesconocido {
				t.Errorf("Status = %q, quería %q", detalle.Status, estadoDesconocido)
			}
			if detalle.ID != "mi-slug" || detalle.Title != "Prueba" || detalle.Type != "tv-anime" {
				t.Errorf("detalle = %+v, quería la ficha sintética leída entera", detalle)
			}
			if detalle.Cover != "https://cdn.animeav1.com/covers/123.jpg" {
				t.Errorf("Cover = %q, quería el id numérico del sitio", detalle.Cover)
			}
			if detalle.Genres == nil || len(detalle.Genres) != 0 {
				t.Errorf("Genres = %#v, quería un slice vacío no nil", detalle.Genres)
			}
			if len(episodios) != 0 {
				t.Errorf("episodios = %d, quería 0", len(episodios))
			}
		})
	}
}

func TestDetallePaginaInexistente(t *testing.T) {
	srv, _ := servidorJSON(t, fixture(t, "detail_404_data.json"), "/media/no-existe/__data.json")
	a := New(clientePrueba(), srv.URL)

	_, _, err := a.GetAnimeDetails(context.Background(), "no-existe")
	if err == nil {
		t.Fatal("quería error con el nodo 404 de SvelteKit")
	}
	if !errors.Is(err, httpx.ErrNotFound) {
		t.Errorf("err = %v, quería httpx.ErrNotFound (errors.Is)", err)
	}
	var se *httpx.ScraperError
	if !errors.As(err, &se) {
		t.Fatalf("err = %T, quería un *httpx.ScraperError", err)
	}
	if se.StatusCode != http.StatusNotFound || se.Operation != "detail" || se.Provider != providerName {
		t.Errorf("ScraperError = %+v, quería el 404 de detail/animeav1", se)
	}
}

func TestDetalleSinMedia(t *testing.T) {
	srv, _ := servidorJSON(t, []byte(`{"type":"data","nodes":[{"type":"data","data":[{}]}]}`), "/media/otra/__data.json")
	a := New(clientePrueba(), srv.URL)

	_, _, err := a.GetAnimeDetails(context.Background(), "otra")
	if err == nil {
		t.Fatal("quería error sin el objeto media")
	}
	if !strings.Contains(err.Error(), "no trae media") {
		t.Errorf("err = %v, quería que mencionara el campo que falta", err)
	}
	if errors.Is(err, httpx.ErrNotFound) {
		t.Error("un 200 sin media no es un recurso no encontrado")
	}
}

func TestDetalleRespuestaInesperada(t *testing.T) {
	casos := []struct {
		nombre, cuerpo, contiene string
	}{
		{"json cortado", `{"type":"data","nodes":`, "parseando el __data.json del detalle"},
		{"type de error", `{"type":"error","nodes":[]}`, `type = "error"`},
		{"sin nodo de página", `{"type":"data","nodes":[null]}`, "nodo de página"},
		{"raíz que no es objeto", `{"type":"data","nodes":[{"type":"data","data":[1,2]}]}`, "no son un objeto"},
		{"referencia fuera de rango", `{"type":"data","nodes":[{"type":"data","data":[{"media":99}]}]}`, "fuera del array"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv, _ := servidorJSON(t, []byte(c.cuerpo), "/media/bleach/__data.json")
			a := New(clientePrueba(), srv.URL)

			_, _, err := a.GetAnimeDetails(context.Background(), "bleach")
			if err == nil {
				t.Fatalf("quería error con %s", c.nombre)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}

func TestDetalleSlugVacioNoPideRed(t *testing.T) {
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		t.Error("GetAnimeDetails no debía hacer ninguna petición")
		return nil, errors.New("sin red (doer de prueba)")
	}}
	a := New(d, DefaultBaseURL)

	for _, id := range []string{"", "   ", "/", " // "} {
		_, _, err := a.GetAnimeDetails(context.Background(), id)
		if !errors.Is(err, errSlugVacio) {
			t.Errorf("id %q: err = %v, quería errSlugVacio", id, err)
		}
	}
}

func TestDetalleSinDoer(t *testing.T) {
	a := New(nil, "")
	_, _, err := a.GetAnimeDetails(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error sin doer")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer", err)
	}
}

func TestDetalleCtxCancelado(t *testing.T) {
	var peticiones int
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		peticiones++
		<-req.Context().Done()
		return nil, req.Context().Err()
	}}
	a := New(d, DefaultBaseURL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := a.GetAnimeDetails(ctx, "bleach")
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería context.Canceled", err)
	}
	if peticiones != 1 {
		t.Errorf("peticiones = %d, quería 1 (la petición se lanza y cancela)", peticiones)
	}
}

func TestDetalleCuerpoDemasiadoGrande(t *testing.T) {
	cuerpo := strings.Repeat("a", maxDetalleBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	_, _, err := a.GetAnimeDetails(context.Background(), "bleach")
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxDetalleBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}

// TestDetalleNumeracionConHueco cubre la "numeración rara" del muestreo:
// Gintama Season 1 declara 201 episodios pero la lista salta del 1 al 3 (el
// 2 no existe en el sitio) y termina en el 201. La lista manda para los
// episodios y episodesCount para el total de la ficha.
func TestDetalleNumeracionConHueco(t *testing.T) {
	srv, _ := servidorJSON(t, fixture(t, "detail_gintama-season-1_data.json"), "/media/gintama-season-1/__data.json")
	a := New(clientePrueba(), srv.URL)

	detalle, episodios, err := a.GetAnimeDetails(context.Background(), "gintama-season-1")
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if detalle.TotalEpisodes != 201 {
		t.Errorf("TotalEpisodes = %d, quería 201 (el que declara el sitio)", detalle.TotalEpisodes)
	}
	if len(episodios) != 200 {
		t.Fatalf("episodios = %d, quería 200 (los de la lista)", len(episodios))
	}
	if episodios[0].Number != 1 || episodios[1].Number != 3 {
		t.Errorf("primeros números = %d, %d; quería 1 y 3 (el 2 no existe)", episodios[0].Number, episodios[1].Number)
	}
	if episodios[1].Title != "Gintama - 3" {
		t.Errorf("título = %q, quería el generado con el número real", episodios[1].Title)
	}
	if ultimo := episodios[len(episodios)-1]; ultimo.Number != 201 {
		t.Errorf("último número = %d, quería 201", ultimo.Number)
	}
	comprobarNumeracion(t, episodios, 1, 201)
}

// comprobarNumeracion exige números enteros crecientes dentro del rango (con
// los huecos que haya por el medio), sin repetir ningún id de episodio.
func comprobarNumeracion(t *testing.T, episodios []models.Episode, desde, hasta int) {
	t.Helper()
	vistos := make(map[string]struct{}, len(episodios))
	anterior := 0
	for i, ep := range episodios {
		if _, ya := vistos[ep.ID]; ya {
			t.Errorf("episodio %d: id repetido %q", i, ep.ID)
		}
		vistos[ep.ID] = struct{}{}
		if ep.ID == "" || ep.Kind == "" {
			t.Errorf("episodio %d incompleto: %+v", i, ep)
		}
		if ep.Number < desde || ep.Number > hasta {
			t.Errorf("episodio %d: número %d fuera de [%d, %d]", i, ep.Number, desde, hasta)
		}
		if ep.Number <= anterior {
			t.Errorf("episodio %d: número %d tras %d, quería creciente", i, ep.Number, anterior)
		}
		anterior = ep.Number
	}
}
