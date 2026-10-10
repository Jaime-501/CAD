package animeav1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
)

// servidorHome monta el servidor local con las dos rutas del Home:
// /catalogo/__data.json (exigiendo status=emision, como la página 1 que pide
// Home) y /__data.json. Un body nil responde 500, para simular el fallo de
// esa sección.
func servidorHome(t *testing.T, emisionJSON, portadaJSON []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var peticiones atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		servir := func(body []byte) {
			if body == nil {
				http.Error(w, "error interno", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		}
		switch r.URL.Path {
		case "/catalogo/__data.json":
			if r.URL.Query().Get("status") != "emision" {
				t.Errorf("catálogo del Home pedido sin status=emision: %q", r.URL.RawQuery)
			}
			servir(emisionJSON)
		case "/__data.json":
			servir(portadaJSON)
		default:
			t.Errorf("ruta inesperada del Home: %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &peticiones
}

// fixturesHome carga los dos JSON reales de T39 que usa Home.
func fixturesHome(t *testing.T) (emision, portada []byte) {
	t.Helper()
	return fixture(t, "home_catalogo_emision_data.json"),
		fixture(t, "home_portada_data.json")
}

// payloadDePrueba envuelve un array plano devalue en el cuerpo de un
// __data.json con nodo de página. Es JSON artificial: los literales van como
// elementos sueltos del array plano y los objetos solo referencias por
// índice (así trabaja devalue); cubre casos que el sitio no ofrece (dedup,
// sección vacía). Ver PROGRESO.md.
func payloadDePrueba(t *testing.T, plano []any) []byte {
	t.Helper()
	cuerpo := map[string]any{
		"type":  "data",
		"nodes": []any{map[string]any{"type": "data", "data": plano}},
	}
	data, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatalf("serializando el payload de prueba: %v", err)
	}
	return data
}

// buscarSeccion devuelve la sección por su id.
func buscarSeccion(t *testing.T, secciones []models.HomeSection, id string) models.HomeSection {
	t.Helper()
	for _, s := range secciones {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no encontré la sección %q; secciones: %v", id, idsDe(secciones))
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
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
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
	if len(s.Items) != 20 {
		t.Fatalf("items = %d, quería 20 (los del fixture)", len(s.Items))
	}
	if s.Items[0].ID != "battle-spirits-re-zekkai-no-kuu" {
		t.Errorf("primer item = %q, quería battle-spirits-re-zekkai-no-kuu", s.Items[0].ID)
	}
	ultimo := s.Items[len(s.Items)-1].ID
	if ultimo != "marronnier-oukoku-no-shichinin-no-kishi" {
		t.Errorf("último item = %q, quería marronnier-oukoku-no-shichinin-no-kishi", ultimo)
	}
	for i, item := range s.Items {
		if item.ID == "" || item.Title == "" {
			t.Errorf("item %d incompleto: %+v", i, item)
		}
		if item.Provider != providerName {
			t.Errorf("item %d: Provider = %q, quería %q", i, item.Provider, providerName)
		}
	}
}

func TestHomeParseaSeccionRecientes(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería [emision recientes]", idsDe(got))
	}

	s := buscarSeccion(t, got, "recientes")
	if s.Title != "Recién agregados" {
		t.Errorf("Title = %q, quería %q", s.Title, "Recién agregados")
	}
	if len(s.Items) != 20 {
		t.Fatalf("items = %d, quería 20 (los del fixture)", len(s.Items))
	}
	if s.Items[0].ID != "hotaru-no-yomeiri" {
		t.Errorf("primer item = %q, quería hotaru-no-yomeiri", s.Items[0].ID)
	}
	ultimo := s.Items[len(s.Items)-1].ID
	if ultimo != "mouse-cursor-de-genjitsu-wo-sousa-dekiru-you-ni-natta-node-onna-no-ko-wo-ippai-click-shimaasu" {
		t.Errorf("último item = %q, quería mouse-cursor-de-genjitsu-…-click-shimaasu", ultimo)
	}
	primero := s.Items[0]
	if primero.Title != "Hotaru no Yomeiri" {
		t.Errorf("Title de hotaru-no-yomeiri = %q, quería %q", primero.Title, "Hotaru no Yomeiri")
	}
	if primero.Provider != providerName {
		t.Errorf("Provider = %q, quería %q", primero.Provider, providerName)
	}
}

func TestHomeSlugsUsablesEnDetalle(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería 2", idsDe(got))
	}
	// El mismo formato que Anime.ID de la búsqueda: sin barra, sin ":" ni
	// espacios (no se exige más porque hay slugs con "---").
	prohibidos := []string{"/", ":", " "}
	for _, s := range got {
		for i, item := range s.Items {
			if item.ID == "" {
				t.Errorf("sección %s item %d: id vacío", s.ID, i)
			}
			for _, p := range prohibidos {
				if strings.Contains(item.ID, p) {
					t.Errorf("sección %s item %d: el id %q contiene %q", s.ID, i, item.ID, p)
				}
			}
		}
	}
}

func TestHomePortadaUsaCDN(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("secciones = %v, quería 2", idsDe(got))
	}
	const prefijo = "https://cdn.animeav1.com/covers/"
	for _, s := range got {
		for i, item := range s.Items {
			if !strings.HasPrefix(item.CoverURL, prefijo) {
				t.Errorf("sección %s item %d (%s): CoverURL = %q, quería el CDN %q",
					s.ID, i, item.ID, item.CoverURL, prefijo)
			}
			if !strings.HasSuffix(item.CoverURL, ".jpg") {
				t.Errorf("sección %s item %d (%s): CoverURL = %q, quería .jpg",
					s.ID, i, item.ID, item.CoverURL)
			}
		}
	}
}

func TestHomeIdNumericoSinNotacionCientifica(t *testing.T) {
	emision, portada := fixturesHome(t)
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	s := buscarSeccion(t, got, "recientes")
	if len(s.Items) != 20 {
		t.Fatalf("items = %d, quería 20", len(s.Items))
	}
	for _, item := range s.Items {
		if strings.Contains(item.CoverURL, "e+") {
			t.Errorf("portada %q: notación científica en el id numérico", item.CoverURL)
		}
		base := item.CoverURL
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if strings.Contains(strings.TrimSuffix(base, ".jpg"), ".") {
			t.Errorf("portada %q: el id numérico no debe serializarse con %q", item.CoverURL, base)
		}
	}
}

func TestHomeEmisionFiltroNoAplicadoEsError(t *testing.T) {
	// home_catalogo_data.json es real y trae filters.status = null (sin
	// filtro): si la ruta de status=emision lo sirviera, el sitio ya no
	// reconocería el slug y la sección emision debe fallar por la guarda.
	_, portada := fixturesHome(t)
	emision := fixture(t, "home_catalogo_data.json")
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("un fallo de emision no debe romper el Home: %v", err)
	}
	if len(got) != 1 || got[0].ID != "recientes" {
		t.Fatalf("secciones = %v, quería solo [recientes]", idsDe(got))
	}
	if len(got[0].Items) != 20 {
		t.Errorf("recientes: items = %d, quería 20", len(got[0].Items))
	}
}

func TestHomeDeduplicaPorSlug(t *testing.T) {
	// Payloads artificiales derivados del formato real (ver payloadDePrueba):
	// un item repetido en cada sección.
	emision := payloadDePrueba(t, []any{
		map[string]any{"results": 1, "filters": 2}, // 0 raíz
		[]any{3, 7, 11},                                // 1 results
		map[string]any{"status": 13},                   // 2 filters (status 2)
		map[string]any{"slug": 4, "title": 5, "id": 6}, // 3 item uno
		"uno", // 4
		"Uno", // 5
		"1",   // 6
		map[string]any{"slug": 8, "title": 9, "id": 10}, // 7 item dos
		"dos", // 8
		"Dos", // 9
		"2",   // 10
		map[string]any{"slug": 4, "title": 12, "id": 6}, // 11 item uno repetido
		"Uno repetido", // 12
		2,              // 13 status numérico
	})
	portada := payloadDePrueba(t, []any{
		map[string]any{"latestMedia": 1},               // 0 raíz
		[]any{2, 6, 10},                                // 1 latestMedia
		map[string]any{"slug": 3, "title": 4, "id": 5}, // 2 item uno (id numérico)
		"uno", // 3
		"Uno", // 4
		1,     // 5
		map[string]any{"slug": 7, "title": 8, "id": 9}, // 6 item dos
		"dos", // 7
		"Dos", // 8
		2,     // 9
		map[string]any{"slug": 3, "title": 11, "id": 5}, // 10 item uno repetido
		"Uno repetido", // 11
	})
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}

	em := buscarSeccion(t, got, "emision")
	if len(em.Items) != 2 {
		t.Fatalf("emision: items = %d, quería 2 tras el dedup", len(em.Items))
	}
	if em.Items[0].ID != "uno" || em.Items[1].ID != "dos" {
		t.Errorf("emision: ids = [%s %s], quería preservar el orden [uno dos]",
			em.Items[0].ID, em.Items[1].ID)
	}
	if em.Items[0].Title != "Uno" {
		t.Errorf("emision: Title = %q, quería el de la primera aparición", em.Items[0].Title)
	}

	re := buscarSeccion(t, got, "recientes")
	if len(re.Items) != 2 {
		t.Fatalf("recientes: items = %d, quería 2 tras el dedup", len(re.Items))
	}
	if re.Items[0].ID != "uno" || re.Items[1].ID != "dos" {
		t.Errorf("recientes: ids = [%s %s], quería preservar el orden [uno dos]",
			re.Items[0].ID, re.Items[1].ID)
	}
	if re.Items[0].CoverURL != "https://cdn.animeav1.com/covers/1.jpg" {
		t.Errorf("recientes: CoverURL = %q, quería covers/1.jpg desde el id numérico", re.Items[0].CoverURL)
	}
}

func TestHomeUnaSeccionFallaLaOtraSirve(t *testing.T) {
	_, portada := fixturesHome(t)
	srv, _ := servidorHome(t, nil, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("un fallo en emision no debe romper el Home: %v", err)
	}
	if len(got) != 1 || got[0].ID != "recientes" {
		t.Fatalf("secciones = %v, quería solo [recientes]", idsDe(got))
	}
	if len(got[0].Items) != 20 {
		t.Errorf("recientes: items = %d, quería 20", len(got[0].Items))
	}
}

func TestHomeTodasFallanDevuelveError(t *testing.T) {
	srv, _ := servidorHome(t, nil, nil)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
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
	// Portada artificial con latestMedia vacío (el sitio siempre trae 20).
	portada := payloadDePrueba(t, []any{
		map[string]any{"latestMedia": 1}, // 0 raíz
		[]any{},                          // 1 latestMedia vacío
	})
	srv, _ := servidorHome(t, emision, portada)
	a := New(clientePrueba(), srv.URL)

	got, err := a.Home(context.Background())
	if err != nil {
		t.Fatalf("Home falló: %v", err)
	}
	if len(got) != 1 || got[0].ID != "emision" {
		t.Fatalf("secciones = %v, quería solo [emision]", idsDe(got))
	}
	if len(got[0].Items) != 20 {
		t.Errorf("emision: items = %d, quería 20", len(got[0].Items))
	}
}

func TestHomeContextoCancelado(t *testing.T) {
	var peticiones atomic.Int32
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		peticiones.Add(1)
		return nil, errors.New("no debía hacerse ninguna petición")
	}}
	a := New(d, DefaultBaseURL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := a.Home(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería context.Canceled", err)
	}
	if n := peticiones.Load(); n != 0 {
		t.Errorf("peticiones = %d, quería 0 tras cancelar", n)
	}
}
