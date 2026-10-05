package jkanime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miapp/internal/models"
)

// fixtureEpisodios carga un JSON real de testdata (respuesta cruda del
// servidor, ver PROGRESO.md).
func fixtureEpisodios(t *testing.T, nombre string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatalf("no pude leer testdata/%s: %v", nombre, err)
	}
	return data
}

// paginaJSON carga un fixture de episodios y le cambia solo last_page (lo que
// usa el código para saber cuál es la última) para armar el escenario del
// test; los episodios se quedan crudos.
func paginaJSON(t *testing.T, nombre string, ultima int) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(fixtureEpisodios(t, nombre), &m); err != nil {
		t.Fatalf("el fixture %s no es JSON: %v", nombre, err)
	}
	m["last_page"] = ultima
	carros, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("no pude reescribir %s: %v", nombre, err)
	}
	return carros
}

// escenarioAJAX describe el sitio local que reproduce el contrato de
// jkanime.net para los tests de T09.
type escenarioAJAX struct {
	slug      string // slug de la serie, p. ej. "bleach"
	numericID string // id numérico del sitio, p. ej. "278"
	detalle   []byte // HTML de la página de detalle; por defecto detail.html
	// csrf entrega un token distinto por visita al detalle (por defecto el
	// sanitizado del fixture).
	csrf func(visita int) string
	// pagina responde a cada página y a cada intento sobre ella.
	pagina func(n, intento int) (int, []byte)
}

// conteoAJAX acumula lo que ha visto el servidor local: visitas al detalle,
// intentos por página y el último CSRF entregado.
type conteoAJAX struct {
	mu          sync.Mutex
	detalles    int
	intentos    map[int]int
	ultimoToken string
}

func (c *conteoAJAX) visitasDetalle() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.detalles
}

func (c *conteoAJAX) intentosDe(pagina int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.intentos[pagina]
}

// servidorAJAX monta el contrato del sitio: GET /<slug>/ devuelve el detalle
// con el CSRF vigente y POST /ajax/episodes/<id>/<n> responde según el
// escenario. Dentro del handler se comprueba el método, la ruta, el
// Content-Type, la cabecera X-Requested-With, el Referer y que el _token del
// cuerpo coincide con el último CSRF entregado — así un refresco de sesión sin
// renovar el token quedaría pillado.
func servidorAJAX(t *testing.T, esc escenarioAJAX) (*httptest.Server, *conteoAJAX) {
	t.Helper()
	if esc.pagina == nil {
		t.Fatal("el escenario necesita una función de página")
	}
	if esc.detalle == nil {
		esc.detalle = fixtureDetalle(t, "detail.html")
	}
	if esc.csrf == nil {
		esc.csrf = func(int) string { return "TOKEN_DE_PRUEBA" }
	}
	cont := &conteoAJAX{intentos: map[int]int{}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ruta := r.URL.EscapedPath()
		prefijo := "/ajax/episodes/" + esc.numericID + "/"

		switch {
		case r.Method == http.MethodGet && ruta == "/"+url.PathEscape(esc.slug)+"/":
			cont.mu.Lock()
			cont.detalles++
			visita := cont.detalles
			cont.mu.Unlock()

			token := esc.csrf(visita)
			if token == "" {
				token = "TOKEN_DE_PRUEBA"
			}
			cont.mu.Lock()
			cont.ultimoToken = token
			cont.mu.Unlock()

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(bytes.Replace(esc.detalle, []byte("TOKEN_DE_PRUEBA"), []byte(token), 1))

		case r.Method == http.MethodPost && strings.HasPrefix(ruta, prefijo):
			n, err := strconv.Atoi(strings.TrimPrefix(ruta, prefijo))
			if err != nil {
				t.Errorf("ruta de página ilegible: %q", ruta)
				http.Error(w, "ruta", http.StatusBadRequest)
				return
			}
			if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
				t.Errorf("página %d: Content-Type = %q, quería application/x-www-form-urlencoded", n, ct)
			}
			if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
				t.Errorf("página %d: X-Requested-With = %q, quería XMLHttpRequest", n, r.Header.Get("X-Requested-With"))
			}
			if !strings.HasSuffix(r.Referer(), "/"+esc.slug+"/") {
				t.Errorf("página %d: Referer = %q, quería la página de la serie", n, r.Referer())
			}
			if err := r.ParseForm(); err != nil {
				t.Errorf("página %d: no pude leer el cuerpo: %v", n, err)
			}

			cont.mu.Lock()
			cont.intentos[n]++
			intento := cont.intentos[n]
			entregado := cont.ultimoToken
			cont.mu.Unlock()

			if token := r.Form.Get("_token"); token != entregado {
				t.Errorf("página %d (intento %d): _token = %q, quería el último CSRF entregado (%q)", n, intento, token, entregado)
			}

			status, cuerpo := esc.pagina(n, intento)
			if cuerpo == nil {
				cuerpo = []byte("{}")
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			_, _ = w.Write(cuerpo)

		default:
			t.Errorf("petición inesperada: %s %s", r.Method, ruta)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, cont
}

// rango devuelve los números de episodio de desde a hasta, ambos incluidos.
func rango(desde, hasta int) []int {
	numeros := make([]int, 0, hasta-desde+1)
	for n := desde; n <= hasta; n++ {
		numeros = append(numeros, n)
	}
	return numeros
}

// comprobarSerie verifica que los episodios de una serie salen en orden, con
// el id de T05 aplicado, kind episode y título.
func comprobarSerie(t *testing.T, episodios []models.Episode, slug string, esperados []int) {
	t.Helper()
	if len(episodios) != len(esperados) {
		t.Fatalf("episodios = %d, quería %d", len(episodios), len(esperados))
	}
	for i, numero := range esperados {
		ep := episodios[i]
		if ep.Number != numero {
			t.Errorf("episodio %d: Number = %d, quería %d", i, ep.Number, numero)
		}
		querido := fmt.Sprintf("jkanime:%s:episode:%d", slug, numero)
		if ep.ID != querido {
			t.Errorf("episodio %d: ID = %q, quería %q", i, ep.ID, querido)
		}
		if ep.Kind != models.KindEpisode {
			t.Errorf("episodio %d: Kind = %q, quería %q", i, ep.Kind, models.KindEpisode)
		}
		if ep.Title == "" {
			t.Errorf("episodio %d: Title vacío", i)
		}
	}
}

// comprobarSinDuplicados falla si algún id se repite dentro de la lista.
func comprobarSinDuplicados(t *testing.T, episodios []models.Episode) {
	t.Helper()
	vistos := make(map[string]bool, len(episodios))
	for _, ep := range episodios {
		if vistos[ep.ID] {
			t.Errorf("episodio duplicado: %s", ep.ID)
		}
		vistos[ep.ID] = true
	}
}

// recojerLotes acumula lotes hasta el Done (o se rinde a los 10 s).
func recojerLotes(t *testing.T, lotes <-chan models.EpisodeBatch) []models.EpisodeBatch {
	t.Helper()
	var recibidos []models.EpisodeBatch
	for {
		select {
		case b := <-lotes:
			recibidos = append(recibidos, b)
			if b.Done {
				return recibidos
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("no llegó el lote Done; recibidos: %+v", recibidos)
			return nil
		}
	}
}

// respuestaFalsa construye una http.Response para el doerGrabador (sin red).
func respuestaFalsa(req *http.Request, status int, cuerpo []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:       io.NopCloser(bytes.NewReader(cuerpo)),
		Request:    req,
	}
}

func TestStreamLotesOrdenadosYDone(t *testing.T) {
	p1 := paginaJSON(t, "episodes_p1.json", 3)
	p2 := paginaJSON(t, "episodes_p2.json", 3)
	p3 := paginaJSON(t, "episodes_last.json", 3)
	srv, cont := servidorAJAX(t, escenarioAJAX{
		slug:      "bleach",
		numericID: "278",
		pagina: func(n, _ int) (int, []byte) {
			switch n {
			case 1:
				return http.StatusOK, p1
			case 2:
				return http.StatusOK, p2
			case 3:
				return http.StatusOK, p3
			}
			return http.StatusNotFound, nil
		},
	})
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	lotes := make(chan models.EpisodeBatch, 16)
	detalle, pagina1, err := j.StreamAnimeDetails(context.Background(), "bleach",
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}

	// El detalle sale de inmediato, con los datos del fixture.
	if detalle.ID != "bleach" || detalle.Title != "Bleach" {
		t.Errorf("detalle = %+v, quería el slug y el título del fixture", detalle)
	}
	if detalle.Provider != providerName || detalle.Type != tipoSerie {
		t.Errorf("detalle = %+v, quería provider jkanime y tipo series", detalle)
	}
	if detalle.TotalEpisodes != 366 {
		t.Errorf("TotalEpisodes = %d, quería 366", detalle.TotalEpisodes)
	}

	// La página 1 se devuelve y NO se emite como lote.
	if len(pagina1) != 16 {
		t.Fatalf("página 1 = %d episodios, quería 16 (los del fixture)", len(pagina1))
	}
	comprobarSerie(t, pagina1, "bleach", rango(1, 16))
	if primera := pagina1[0]; primera.Thumbnail != "https://cdn.jkdesa.com/assets/images/animes/video/image_thumb/jkvideo_785b35c11478409629f56d9bcfbded29.jpg" {
		t.Errorf("Thumbnail = %q, quería la miniatura absoluta del sitio", primera.Thumbnail)
	}

	recibidos := recojerLotes(t, lotes)
	if len(recibidos) != 3 {
		t.Fatalf("lotes = %d, quería 3 (páginas 2 y 3 + Done): %+v", len(recibidos), recibidos)
	}
	for i, b := range recibidos {
		if b.AnimeID != "bleach" {
			t.Errorf("lote %d: AnimeID = %q, quería bleach", i, b.AnimeID)
		}
	}

	segundo, tercero, fin := recibidos[0], recibidos[1], recibidos[2]
	if segundo.Batch != 2 || segundo.Done || segundo.Err != "" {
		t.Errorf("primer lote = %+v, quería Batch 2 sin Done ni Err", segundo)
	}
	comprobarSerie(t, segundo.Episodes, "bleach", rango(17, 32))
	if tercero.Batch != 3 || tercero.Done || tercero.Err != "" {
		t.Errorf("segundo lote = %+v, quería Batch 3 sin Done ni Err", tercero)
	}
	comprobarSerie(t, tercero.Episodes, "bleach", rango(353, 366))
	// El fixture de la última página trae todas las imágenes vacías.
	for _, ep := range tercero.Episodes {
		if ep.Thumbnail != "" {
			t.Errorf("episodio %d: Thumbnail = %q, quería vacío (el fixture no trae imagen)", ep.Number, ep.Thumbnail)
		}
	}
	if !fin.Done || fin.Err != "" || len(fin.Episodes) != 0 {
		t.Errorf("lote final = %+v, quería Done sin Err y sin episodios", fin)
	}
	if fin.Batch != 3 {
		t.Errorf("lote final: Batch = %d, quería 3 (la última página)", fin.Batch)
	}

	todos := append(append(append([]models.Episode{}, pagina1...), segundo.Episodes...), tercero.Episodes...)
	comprobarSinDuplicados(t, todos)
	if len(todos) != 46 {
		t.Errorf("episodios totales = %d, quería 46 (16+16+14)", len(todos))
	}

	// Tras el Done no debe llegar nada más.
	select {
	case b := <-lotes:
		t.Errorf("lote posterior al Done: %+v", b)
	case <-time.After(200 * time.Millisecond):
	}

	if n := cont.visitasDetalle(); n != 1 {
		t.Errorf("visitas al detalle = %d, quería 1 (sin refrescos)", n)
	}
	for _, pagina := range []int{1, 2, 3} {
		if n := cont.intentosDe(pagina); n != 1 {
			t.Errorf("intentos de la página %d = %d, quería 1", pagina, n)
		}
	}
}

func TestStream419RefrescaLaSesion(t *testing.T) {
	p1 := paginaJSON(t, "episodes_p1.json", 2)
	p2 := paginaJSON(t, "episodes_p2.json", 2)
	srv, cont := servidorAJAX(t, escenarioAJAX{
		slug:      "bleach",
		numericID: "278",
		// Cada visita al detalle entrega un CSRF distinto: si el reintento no
		// usa el último, servidorAJAX lo detecta en el propio POST.
		csrf: func(visita int) string { return fmt.Sprintf("TOKEN_%d", visita) },
		pagina: func(n, intento int) (int, []byte) {
			if n == 1 {
				return http.StatusOK, p1
			}
			if intento == 1 {
				return 419, []byte("<html>CSRF token mismatch</html>")
			}
			return http.StatusOK, p2
		},
	})
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	lotes := make(chan models.EpisodeBatch, 8)
	_, pagina1, err := j.StreamAnimeDetails(context.Background(), "bleach",
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}

	recibidos := recojerLotes(t, lotes)
	if len(recibidos) != 2 {
		t.Fatalf("lotes = %d, quería 2 (página 2 + Done): %+v", len(recibidos), recibidos)
	}
	if recibidos[0].Done || recibidos[0].Batch != 2 {
		t.Errorf("primer lote = %+v, quería la página 2 sin Done", recibidos[0])
	}
	comprobarSerie(t, recibidos[0].Episodes, "bleach", rango(17, 32))
	if recibidos[1].Err != "" {
		t.Errorf("Done con Err = %q, quería recuperación limpia", recibidos[1].Err)
	}

	todos := append(append([]models.Episode{}, pagina1...), recibidos[0].Episodes...)
	comprobarSinDuplicados(t, todos)
	if len(todos) != 32 {
		t.Errorf("episodios = %d, quería 32 (16 + 16)", len(todos))
	}

	if n := cont.visitasDetalle(); n != 2 {
		t.Errorf("visitas al detalle = %d, quería 2 (detalle + refresco)", n)
	}
	if n := cont.intentosDe(2); n != 2 {
		t.Errorf("intentos de la página 2 = %d, quería 2 (419 + reintento)", n)
	}
	if n := cont.intentosDe(1); n != 1 {
		t.Errorf("intentos de la página 1 = %d, quería 1", n)
	}
}

func TestStream419PersistenteEmiteError(t *testing.T) {
	p1 := paginaJSON(t, "episodes_p1.json", 2)
	srv, cont := servidorAJAX(t, escenarioAJAX{
		slug:      "bleach",
		numericID: "278",
		pagina: func(n, _ int) (int, []byte) {
			if n == 1 {
				return http.StatusOK, p1
			}
			return 419, []byte("<html>CSRF token mismatch</html>")
		},
	})
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	lotes := make(chan models.EpisodeBatch, 8)
	_, _, err := j.StreamAnimeDetails(context.Background(), "bleach",
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("el error de fondo no debe subir al llamador: %v", err)
	}

	recibidos := recojerLotes(t, lotes)
	if len(recibidos) != 1 {
		t.Fatalf("lotes = %d, quería solo el final con Err: %+v", len(recibidos), recibidos)
	}
	fin := recibidos[0]
	if !fin.Done {
		t.Errorf("lote = %+v, quería Done", fin)
	}
	if !strings.Contains(fin.Err, "419") {
		t.Errorf("Err = %q, quería el 419 mencionado", fin.Err)
	}
	if !strings.Contains(fin.Err, "página 2") {
		t.Errorf("Err = %q, quería la página que falló", fin.Err)
	}
	if len(fin.Episodes) != 0 {
		t.Errorf("el lote de error trae %d episodios, quería ninguno", len(fin.Episodes))
	}

	if n := cont.visitasDetalle(); n != 2 {
		t.Errorf("visitas al detalle = %d, quería 2 (detalle + un solo refresco)", n)
	}
	if n := cont.intentosDe(2); n != 2 {
		t.Errorf("intentos de la página 2 = %d, quería 2 (un reintento, no más)", n)
	}
}

func TestStreamCancelacionSinFugas(t *testing.T) {
	detalle := fixtureDetalle(t, "detail.html")
	primera := paginaJSON(t, "episodes_p1.json", 50)
	var peticiones atomic.Int32
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		peticiones.Add(1)
		switch {
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/bleach/"):
			return respuestaFalsa(req, http.StatusOK, detalle), nil
		case req.Method == http.MethodPost:
			// Cualquier página responde la misma: last_page = 50 dejaría a la
			// goroutine trabajando bastante rato si nadie la cancela.
			return respuestaFalsa(req, http.StatusOK, primera), nil
		}
		return nil, fmt.Errorf("petición inesperada: %s %s", req.Method, req.URL)
	}}
	j := New(d, "https://jkanime.test", nil)

	// Que se asienten las goroutines de los tests anteriores.
	time.Sleep(50 * time.Millisecond)
	base := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var trasCancelar []models.EpisodeBatch
	var cancelado atomic.Bool

	_, pagina1, err := j.StreamAnimeDetails(ctx, "bleach", func(b models.EpisodeBatch) {
		if cancelado.Load() {
			mu.Lock()
			trasCancelar = append(trasCancelar, b)
			mu.Unlock()
		}
	})
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if len(pagina1) != 16 {
		t.Errorf("página 1 = %d episodios, quería 16", len(pagina1))
	}

	// Se cancela nada más volver: la goroutine aún está en la pausa de 300 ms.
	cancelado.Store(true)
	cancel()

	// Sin fugas: la goroutine del stream termina y el contador de goroutines
	// vuelve al punto de partida (se comprueba corriendo con -race).
	limite := time.Now().Add(10 * time.Second)
	for runtime.NumGoroutine() > base && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > base {
		t.Errorf("goroutines = %d, quería volver a %d: la goroutine no terminó", n, base)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(trasCancelar) != 0 {
		t.Errorf("se emitieron %d lotes tras cancelar: %+v", len(trasCancelar), trasCancelar)
	}
	// Detalle + página 1: nada más se pidió.
	if n := peticiones.Load(); n != 2 {
		t.Errorf("peticiones = %d, quería 2 (detalle + página 1)", n)
	}
}

func TestGetAnimeDetailsTodasLasPaginas(t *testing.T) {
	p1 := paginaJSON(t, "episodes_p1.json", 3)
	p2 := paginaJSON(t, "episodes_p2.json", 3)
	p3 := paginaJSON(t, "episodes_last.json", 3)
	srv, cont := servidorAJAX(t, escenarioAJAX{
		slug:      "bleach",
		numericID: "278",
		pagina: func(n, _ int) (int, []byte) {
			switch n {
			case 1:
				return http.StatusOK, p1
			case 2:
				return http.StatusOK, p2
			case 3:
				return http.StatusOK, p3
			}
			return http.StatusNotFound, nil
		},
	})
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	detalle, episodios, err := j.GetAnimeDetails(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("GetAnimeDetails falló: %v", err)
	}
	if detalle.ID != "bleach" || detalle.TotalEpisodes != 366 {
		t.Errorf("detalle = %+v, quería el detalle completo del fixture", detalle)
	}
	if len(episodios) != 46 {
		t.Fatalf("episodios = %d, quería 46 (16+16+14)", len(episodios))
	}
	comprobarSerie(t, episodios[:16], "bleach", rango(1, 16))
	comprobarSerie(t, episodios[16:32], "bleach", rango(17, 32))
	comprobarSerie(t, episodios[32:], "bleach", rango(353, 366))
	comprobarSinDuplicados(t, episodios)

	if n := cont.visitasDetalle(); n != 1 {
		t.Errorf("visitas al detalle = %d, quería 1", n)
	}
	for _, pagina := range []int{1, 2, 3} {
		if n := cont.intentosDe(pagina); n != 1 {
			t.Errorf("intentos de la página %d = %d, quería 1", pagina, n)
		}
	}
	if n := cont.intentosDe(4); n != 0 {
		t.Errorf("se pidieron %d intentos de la página 4, quería 0 (last_page = 3)", n)
	}
}

func TestGetAnimeDetailsDescartaEntradasInvalidas(t *testing.T) {
	// Misma forma que el sitio, con una entrada que NewEpisodeID rechaza
	// (número negativo): esa se descarta y el resto se conserva.
	pagina := []byte(`{"current_page":1,"last_page":1,"per_page":16,"total":3,` +
		`"data":[{"id":1,"number":1,"title":"Uno","image":""},` +
		`{"id":2,"number":-1,"title":"Número negativo","image":""},` +
		`{"id":3,"number":2,"title":"Dos","image":""}]}`)
	srv, _ := servidorAJAX(t, escenarioAJAX{
		slug:      "bleach",
		numericID: "278",
		pagina: func(n, _ int) (int, []byte) {
			if n == 1 {
				return http.StatusOK, pagina
			}
			return http.StatusNotFound, nil
		},
	})
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	_, episodios, err := j.GetAnimeDetails(context.Background(), "bleach")
	if err != nil {
		t.Fatalf("una entrada inválida no debe tumbar el flujo: %v", err)
	}
	comprobarSerie(t, episodios, "bleach", []int{1, 2})
	comprobarSinDuplicados(t, episodios)
}

func TestStreamPeliculaUnaPagina(t *testing.T) {
	// La película trae una sola página (last_page = 1 en el fixture real).
	pelicula := fixtureEpisodios(t, "episodes_movie.json")
	slug := "bleach-the-diamonddust-rebellion"
	srv, cont := servidorAJAX(t, escenarioAJAX{
		slug:      slug,
		numericID: "504",
		detalle:   fixtureDetalle(t, "detail_movie.html"),
		pagina: func(n, _ int) (int, []byte) {
			if n == 1 {
				return http.StatusOK, pelicula
			}
			return http.StatusNotFound, nil
		},
	})
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	lotes := make(chan models.EpisodeBatch, 8)
	detalle, pagina1, err := j.StreamAnimeDetails(context.Background(), slug,
		func(b models.EpisodeBatch) { lotes <- b })
	if err != nil {
		t.Fatalf("StreamAnimeDetails falló: %v", err)
	}
	if detalle.Type != tipoPelicula {
		t.Errorf("Type = %q, quería %q", detalle.Type, tipoPelicula)
	}

	if len(pagina1) != 1 {
		t.Fatalf("episodios = %d, quería 1 (los del fixture de la película)", len(pagina1))
	}
	peli := pagina1[0]
	if peli.ID != "jkanime:"+slug+":movie" {
		t.Errorf("ID = %q, quería el id de película sin número", peli.ID)
	}
	if peli.Kind != models.KindMovie {
		t.Errorf("Kind = %q, quería %q", peli.Kind, models.KindMovie)
	}
	if peli.Number != 1 || peli.Title != "Bleach: The DiamondDust Rebellion" {
		t.Errorf("episodio = %+v, quería el título del fixture", peli)
	}
	if peli.Thumbnail != "https://cdn.jkdesa.com/assets/images/animes/video/image_thumb/jkvideo_3b745f046957f6b3ea1b037370023dc4.jpg" {
		t.Errorf("Thumbnail = %q, quería la miniatura absoluta", peli.Thumbnail)
	}

	// Con una sola página, el Done llega sin esperas y no se pide nada más.
	recibidos := recojerLotes(t, lotes)
	if len(recibidos) != 1 || !recibidos[0].Done || recibidos[0].Err != "" {
		t.Fatalf("lotes = %+v, quería un único Done sin Err", recibidos)
	}
	if recibidos[0].AnimeID != slug {
		t.Errorf("AnimeID = %q, quería %q", recibidos[0].AnimeID, slug)
	}
	if n := cont.visitasDetalle(); n != 1 {
		t.Errorf("visitas al detalle = %d, quería 1", n)
	}
	if n := cont.intentosDe(1); n != 1 {
		t.Errorf("intentos de la página 1 = %d, quería 1", n)
	}
	if n := cont.intentosDe(2); n != 0 {
		t.Errorf("se pidieron %d intentos de la página 2, quería 0", n)
	}
}
