package jkanime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// servidorHome monta el servidor local con las dos rutas del Home:
// /directorio (exigiendo estado=emision, como la página 1 que pide Home) y
// /. Un HTML nil responde 500, para simular el fallo de esa sección.
func servidorHome(t *testing.T, emisionHTML, portadaHTML []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		servir := func(html []byte) {
			if html == nil {
				http.Error(w, "error interno", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(html)
		}
		switch r.URL.Path {
		case "/directorio":
			if r.URL.Query().Get("estado") != "emision" {
				t.Errorf("directorio pedido sin estado=emision: %q", r.URL.RawQuery)
			}
			servir(emisionHTML)
		case "/":
			servir(portadaHTML)
		default:
			t.Errorf("ruta inesperada del Home: %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &peticiones
}

// fixturesHome carga los dos HTML reales de T39 que usa Home.
func fixturesHome(t *testing.T) (emision, portada []byte) {
	t.Helper()
	return fixtureBusqueda(t, "home_directorio_emision.html"),
		fixtureBusqueda(t, "home_portada.html")
}

// emisionDePrueba arma una página de directorio mínima con el mismo patrón
// que el fixture real: var animes = {…} dentro de un <script> (el primer
// <script> es un señuelo sin el marcador). Es HTML artificial: cubre casos
// que el sitio no ofrece (dedup, excepciones, sin JSON); ver PROGRESO.md.
func emisionDePrueba(t *testing.T, items ...directorioItem) []byte {
	t.Helper()
	datos, err := json.Marshal(directorioPagina{Data: items})
	if err != nil {
		t.Fatalf("serializando el directorio de prueba: %v", err)
	}
	return []byte("<html><body><script>var mode = 1;</script><script>" +
		varAnimes + string(datos) + ";</script></body></html>")
}

// portadaDePrueba arma una página de portada mínima con el patrón real de
// tarjeta (div.p-3.d-flex > div.custom_thumb_home). HTML artificial, igual
// que emisionDePrueba.
func portadaDePrueba(t *testing.T, items ...directorioItem) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString("<html><body>")
	for _, it := range items {
		fmt.Fprintf(&b,
			`<div class="p-3 d-flex"><div class="custom_thumb_home"><a href="%s"><img src="%s" class="img-fluid rounded-start" alt="%s"></a></div><div class="card-body-home"><h5 class="card-title"><a href="%s">%s</a><span></span></h5></div></div>`,
			it.Slug, it.Image, it.Title, it.Slug, it.Title)
	}
	b.WriteString("</body></html>")
	return []byte(b.String())
}

// buscarSeccion devuelve la sección por su id.
func buscarSeccion(t *testing.T, secciones []models.HomeSection, id string) models.HomeSection {
	t.Helper()
	for _, s := range secciones {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no encontré la sección %q; secciones: %v", idsDe(secciones), idsDe(secciones))
	return models.HomeSection{}
}

func idsDe(secciones []models.HomeSection) []string {
	out := make([]string, 0, len(secciones))
	for _, s := range secciones {
		out = append(out, s.ID)
	}
	return out
}

func TestHomeParseaSeccionEmision(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería [emision recientes]", idsDe(got))
	}

	s := buscarSeccion(t, got, "emision")
	if s.Title != "En emisión" {
		t.Errorf("Title = %q, quería %q", s.Title, "En emisión")
	}
	if len(s.Items) != 30 {
		t.Fatalf("items = %d, quería 30 (los del fixture)", len(s.Items))
	}
	if s.Items[0].ID != "hyouken-no-majutsushi-ga-sekai-wo-suberu-ii" {
		t.Errorf("primer item = %q, quería hyouken-no-majutsushi-ga-sekai-wo-suberu-ii", s.Items[0].ID)
	}
	ultimo := s.Items[len(s.Items)-1].ID
	if ultimo != "hotel-inhumans-2nd-season" {
		t.Errorf("último item = %q, quería hotel-inhumans-2nd-season", ultimo)
	}
	for i, a := range s.Items {
		if a.ID == "" || a.Title == "" {
			t.Errorf("item %d incompleto: %+v", i, a)
		}
		if !strings.HasPrefix(a.CoverURL, "https://") {
			t.Errorf("item %d: CoverURL = %q, quería una URL absoluta", i, a.CoverURL)
		}
		if a.Provider != providerName {
			t.Errorf("item %d: Provider = %q, quería %q", i, a.Provider, providerName)
		}
	}
}

func TestHomeParseaSeccionRecientes(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería [emision recientes]", idsDe(got))
	}

	s := buscarSeccion(t, got, "recientes")
	if s.Title != "Animes recientes" {
		t.Errorf("Title = %q, quería %q", s.Title, "Animes recientes")
	}
	if len(s.Items) != 32 {
		t.Fatalf("items = %d, quería 32 (las tarjetas del fixture)", len(s.Items))
	}
	if s.Items[0].ID != "holy-knight" {
		t.Errorf("primer item = %q, quería holy-knight", s.Items[0].ID)
	}
	ultimo := s.Items[len(s.Items)-1].ID
	if ultimo != "dark-summoner-to-dekiteiru" {
		t.Errorf("último item = %q, quería dark-summoner-to-dekiteiru", ultimo)
	}

	var holy *models.Anime
	for i := range s.Items {
		if s.Items[i].ID == "holy-knight" {
			holy = &s.Items[i]
		}
		// En las 32 muestras del fixture la portada es el slug del id.
		if sufijo := "/" + s.Items[i].ID + ".jpg"; !strings.HasSuffix(s.Items[i].CoverURL, sufijo) {
			t.Errorf("item %q: CoverURL = %q, quería terminar en %s", s.Items[i].ID, s.Items[i].CoverURL, sufijo)
		}
	}
	if holy == nil {
		t.Fatal("no encontré holy-knight entre los 32 items")
	}
	if holy.Title != "Holy Knight" {
		t.Errorf("Title de holy-knight = %q, quería %q", holy.Title, "Holy Knight")
	}
	if holy.CoverURL != "https://cdn.jkdesa.com/assets/images/animes/image/holy-knight.jpg" {
		t.Errorf("CoverURL de holy-knight = %q, quería la portada del fixture", holy.CoverURL)
	}
	if holy.Provider != providerName {
		t.Errorf("Provider = %q, quería %q", holy.Provider, providerName)
	}
}

func TestHomeSlugsTienenFormatoDeBusqueda(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería 2", idsDe(got))
	}
	// El mismo formato que Anime.ID de la búsqueda: sin barra, sin esquema,
	// sin ":" ni espacios (no se exige más porque hay slugs con "---").
	prohibidos := []string{"/", ":", " "}
	for _, s := range got {
		for i, a := range s.Items {
			if a.ID == "" {
				t.Errorf("sección %s item %d: id vacío", s.ID, i)
			}
			for _, p := range prohibidos {
				if strings.Contains(a.ID, p) {
					t.Errorf("sección %s item %d: el id %q contiene %q", s.ID, i, a.ID, p)
				}
			}
		}
	}
}

func TestHomeDeduplicaPorSlug(t *testing.T) {
	const (
		uno    = "https://jkanime.net/uno/"
		dos    = "https://jkanime.net/dos/"
		imagen = "https://cdn.jkdesa.com/assets/images/animes/image/"
	)
	emision := emisionDePrueba(t,
		directorioItem{Slug: "uno", Title: "Uno", Image: imagen + "uno.jpg"},
		directorioItem{Slug: "dos", Title: "Dos", Image: imagen + "dos.jpg"},
		directorioItem{Slug: "uno", Title: "Uno repetido", Image: imagen + "uno.jpg"},
	)
	portada := portadaDePrueba(t,
		directorioItem{Slug: uno, Title: "Uno", Image: imagen + "uno.jpg"},
		directorioItem{Slug: dos, Title: "Dos", Image: imagen + "dos.jpg"},
		directorioItem{Slug: uno, Title: "Uno repetido", Image: imagen + "uno.jpg"},
	)
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}

	em := buscarSeccion(t, got, "emision")
	if len(em.Items) != 2 {
		t.Fatalf("emision: items = %d, quería 2 tras el dedup", len(em.Items))
	}
	if em.Items[0].ID != "uno" || em.Items[1].ID != "dos" {
		t.Errorf("emision: ids = [%s %s], quería preservar el orden [uno dos]", em.Items[0].ID, em.Items[1].ID)
	}
	if em.Items[0].Title != "Uno" {
		t.Errorf("emision: Title = %q, quería el de la primera aparición", em.Items[0].Title)
	}

	re := buscarSeccion(t, got, "recientes")
	if len(re.Items) != 2 {
		t.Fatalf("recientes: items = %d, quería 2 tras el dedup", len(re.Items))
	}
	if re.Items[0].ID != "uno" || re.Items[1].ID != "dos" {
		t.Errorf("recientes: ids = [%s %s], quería preservar el orden [uno dos]", re.Items[0].ID, re.Items[1].ID)
	}
}

func TestHomeAplicaExcepciones(t *testing.T) {
	// Claves reales de testdata/excepciones.json (la misma pareja que usa
	// TestSearchAplicaExcepciones): la URL rota y la correcta.
	const (
		rota      = "https://jkanime.net/class-de-2-banme-ni-kawaii-onnanoko-to-tomodachi-n"
		correcta  = "https://jkanime.net/class-de-2-banme-ni-kawaii-onnanoko-to-tomodachi-ni-natta/"
		slugBueno = "class-de-2-banme-ni-kawaii-onnanoko-to-tomodachi-ni-natta"
		imagen    = "https://cdn.jkdesa.com/assets/images/animes/image/x.jpg"
	)
	srvMapa, peticionesMapa := servidorJSON(t, fixtureExcepciones(t))

	emision := emisionDePrueba(t,
		directorioItem{Slug: rota, Title: "Class...", Image: imagen},
		directorioItem{Slug: "one-piece", Title: "One Piece", Image: imagen},
	)
	portada := portadaDePrueba(t,
		directorioItem{Slug: rota + "/", Title: "Class...", Image: imagen},
		directorioItem{Slug: "https://jkanime.net/one-piece/", Title: "One Piece", Image: imagen},
	)
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{URL: srvMapa.URL, Doer: clientePrueba()})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}

	em := buscarSeccion(t, got, "emision")
	if em.Items[0].ID != slugBueno {
		t.Errorf("emision: id = %q, quería el slug corregido por el mapa %q", em.Items[0].ID, slugBueno)
	}
	if em.Items[1].ID != "one-piece" {
		t.Errorf("emision: id = %q, quería el slug original (no está en el mapa)", em.Items[1].ID)
	}

	re := buscarSeccion(t, got, "recientes")
	if re.Items[0].ID != slugBueno {
		t.Errorf("recientes: id = %q, quería el slug corregido por el mapa %q", re.Items[0].ID, slugBueno)
	}
	if re.Items[1].ID != "one-piece" {
		t.Errorf("recientes: id = %q, quería el slug original (no está en el mapa)", re.Items[1].ID)
	}

	// El mapa se descarga una sola vez aunque lo consulten items de las
	// dos secciones (caché en memoria del ExceptionResolver, T06).
	if n := peticionesMapa.Load(); n != 1 {
		t.Errorf("el mapa de excepciones se descargó %d veces, quería 1", n)
	}
}

func TestHomeUnaSeccionFallaLaOtraSirve(t *testing.T) {
	_, portada := fixturesHome(t)
	srv, _ := servidorHome(t, nil, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("un fallo en emision no debe romper el Home: %v", err)
	}
	if len(got) != 1 || got[0].ID != "recientes" {
		t.Fatalf("secciones = %v, quería solo [recientes]", idsDe(got))
	}
	if len(got[0].Items) != 32 {
		t.Errorf("recientes: items = %d, quería 32", len(got[0].Items))
	}
}

func TestHomeTodasFallanDevuelveError(t *testing.T) {
	srv, _ := servidorHome(t, nil, nil)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err == nil {
		t.Fatal("quería error cuando fallan las dos secciones")
	}
	if !errors.Is(err, httpx.ErrServerError) {
		t.Errorf("err = %v, quería el primer error (httpx.ErrServerError) envuelto", err)
	}
	if len(got) != 0 {
		t.Errorf("secciones = %v, quería vacío", idsDe(got))
	}
}

func TestHomeSeccionVaciaSeOmite(t *testing.T) {
	emision, _ := fixturesHome(t)
	// Portada artificial sin ninguna tarjeta .custom_thumb_home.
	portada := []byte("<html><body><h4>Animes recientes</h4></body></html>")
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 1 || got[0].ID != "emision" {
		t.Fatalf("secciones = %v, quería solo [emision]", idsDe(got))
	}
	if len(got[0].Items) != 30 {
		t.Errorf("emision: items = %d, quería 30", len(got[0].Items))
	}
}

func TestHomeContextoCancelado(t *testing.T) {
	var peticiones atomic.Int32
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		peticiones.Add(1)
		return nil, errors.New("no debía hacerse ninguna petición")
	}}
	j := New(d, "https://jkanime.net", &ExceptionResolver{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := j.Home(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería context.Canceled", err)
	}
	if n := peticiones.Load(); n != 0 {
		t.Errorf("peticiones = %d, quería 0 tras cancelar", n)
	}
}

func TestHomeNoSaleDeLaRedEnTests(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	hostPrueba := strings.TrimPrefix(srv.URL, "http://")

	var ajenas atomic.Int32
	cliente := clientePrueba()
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != hostPrueba {
			ajenas.Add(1)
			return nil, fmt.Errorf("petición fuera del servidor de pruebas: %s", req.URL.Redacted())
		}
		return cliente.Do(req)
	}}
	j := New(d, srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló con el doer local: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería 2", idsDe(got))
	}
	if n := ajenas.Load(); n != 0 {
		t.Errorf("peticiones a hosts ajenos = %d, quería 0 (el doer inyectado es el único camino)", n)
	}
}

func TestHomeEmisionSinJSONDevuelveError(t *testing.T) {
	// Directorio artificial sin el script var animes (el caso no se ofrece
	// en el sitio real): la sección emision falla y recientes sigue.
	emision := []byte("<html><body><p>directorio sin listado embebido</p></body></html>")
	_, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	j := New(clientePrueba(), srv.URL, &ExceptionResolver{})

	got, err := j.Home(context.Background())
	if err != nil {
		t.Fatalf("un fallo de emision no debe romper el Home: %v", err)
	}
	if len(got) != 1 || got[0].ID != "recientes" {
		t.Fatalf("secciones = %v, quería solo [recientes]", idsDe(got))
	}
	if len(got[0].Items) != 32 {
		t.Errorf("recientes: items = %d, quería 32", len(got[0].Items))
	}
}
