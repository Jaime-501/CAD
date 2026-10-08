package animeav1

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// TestPageDataCatalogo comprueba el desempaquetador contra el fixture real
// del catálogo y que el helper toma el nodo de página (el último), no el
// nodo auth que viene antes.
func TestPageDataCatalogo(t *testing.T) {
	datos, err := PageData(fixture(t, "catalogo_data.json"))
	if err != nil {
		t.Fatalf("PageData falló: %v", err)
	}
	if _, seColo := datos["user"]; seColo {
		t.Error("los datos traen user: el helper tomó el nodo auth, no el de página")
	}

	results, ok := datos["results"].([]any)
	if !ok {
		t.Fatalf("results = %T, quería una lista", datos["results"])
	}
	if len(results) != 11 {
		t.Fatalf("results = %d, quería 11", len(results))
	}

	primero, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("results[0] = %T, quería un objeto", results[0])
	}
	if primero["id"] != "195" {
		t.Errorf("id = %#v, quería el string \"195\" (results[].id es string)", primero["id"])
	}
	if primero["title"] != "Bleach" || primero["slug"] != "bleach" {
		t.Errorf("primero = %+v, quería title=Bleach slug=bleach", primero)
	}
	if primero["categoryId"] != float64(1) {
		t.Errorf("categoryId = %#v, quería 1", primero["categoryId"])
	}
	categoria, ok := primero["category"].(map[string]any)
	if !ok || categoria["name"] != "TV Anime" {
		t.Errorf("category = %#v, quería {name: TV Anime}", primero["category"])
	}

	pag, ok := datos["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("pagination = %T", datos["pagination"])
	}
	queria := map[string]any{"currentPage": 1.0, "recordsPerPage": 20.0, "totalPages": 1.0, "totalRecords": 11.0}
	for k, v := range queria {
		if pag[k] != v {
			t.Errorf("pagination.%s = %#v, quería %#v", k, pag[k], v)
		}
	}
}

func TestPageDataVacio(t *testing.T) {
	datos, err := PageData(fixture(t, "catalogo_vacio_data.json"))
	if err != nil {
		t.Fatalf("PageData falló: %v", err)
	}
	results, ok := datos["results"].([]any)
	if !ok {
		t.Fatalf("results = %T, quería una lista", datos["results"])
	}
	if len(results) != 0 {
		t.Errorf("results = %d, quería 0", len(results))
	}
}

func TestPageDataDetalle(t *testing.T) {
	datos, err := PageData(fixture(t, "detail_data.json"))
	if err != nil {
		t.Fatalf("PageData falló: %v", err)
	}
	media, ok := datos["media"].(map[string]any)
	if !ok {
		t.Fatalf("media = %T, quería un objeto", datos["media"])
	}
	if media["id"] != float64(195) {
		t.Errorf("media.id = %#v, quería 195 (media.id es número)", media["id"])
	}
	if media["slug"] != "bleach" || media["title"] != "Bleach" {
		t.Errorf("media = slug:%v title:%v, quería bleach/Bleach", media["slug"], media["title"])
	}
	if media["episodesCount"] != float64(366) {
		t.Errorf("episodesCount = %#v, quería 366", media["episodesCount"])
	}
	if media["status"] != float64(0) {
		t.Errorf("status = %#v, quería 0 (Finalizado)", media["status"])
	}

	episodios, ok := media["episodes"].([]any)
	if !ok {
		t.Fatalf("episodes = %T, quería una lista", media["episodes"])
	}
	if len(episodios) != 366 {
		t.Fatalf("episodes = %d, quería 366", len(episodios))
	}
	comprobarPar(t, episodios[0], float64(2892), float64(1))
	comprobarPar(t, episodios[365], float64(3257), float64(366))

	generos, ok := media["genres"].([]any)
	if !ok || len(generos) == 0 {
		t.Fatalf("genres = %#v, quería al menos un género", media["genres"])
	}
	if g, ok := generos[0].(map[string]any); !ok || g["name"] != "Acción" {
		t.Errorf("genres[0] = %#v, quería {name: Acción}", generos[0])
	}
	// aka es un mapa por locale, no un string.
	aka, ok := media["aka"].(map[string]any)
	if !ok || aka["ja-jp"] == "" {
		t.Errorf("aka = %#v, quería un mapa por locale", media["aka"])
	}
}

func TestPageDataPelicula(t *testing.T) {
	datos, err := PageData(fixture(t, "detail_movie_data.json"))
	if err != nil {
		t.Fatalf("PageData falló: %v", err)
	}
	media, ok := datos["media"].(map[string]any)
	if !ok {
		t.Fatalf("media = %T", datos["media"])
	}
	if media["slug"] != "bleach-movie-1-memories-of-nobody" {
		t.Errorf("slug = %v", media["slug"])
	}
	if media["episodesCount"] != float64(1) {
		t.Errorf("episodesCount = %#v, quería 1", media["episodesCount"])
	}
	episodios, ok := media["episodes"].([]any)
	if !ok || len(episodios) != 1 {
		t.Fatalf("episodes = %#v, quería 1 episodio", media["episodes"])
	}
	comprobarPar(t, episodios[0], float64(14703), float64(1))
}

// TestPageDataEpisodio usa episode_data.json (T30), el fixture del episodio:
// además de media trae episode, embeds y downloads en el mismo nodo.
func TestPageDataEpisodio(t *testing.T) {
	datos, err := PageData(fixture(t, "episode_data.json"))
	if err != nil {
		t.Fatalf("PageData falló: %v", err)
	}
	for _, campo := range []string{"media", "episode", "embeds", "downloads"} {
		if _, ok := datos[campo]; !ok {
			t.Errorf("falta el campo %q en la raíz", campo)
		}
	}

	episodio, ok := datos["episode"].(map[string]any)
	if !ok {
		t.Fatalf("episode = %T", datos["episode"])
	}
	if episodio["id"] != float64(2892) || episodio["number"] != float64(1) {
		t.Errorf("episode = id:%v number:%v, quería 2892/1", episodio["id"], episodio["number"])
	}
	if episodio["mediaId"] != float64(195) {
		t.Errorf("mediaId = %#v, quería 195", episodio["mediaId"])
	}
	// mirrors: void 0 en el HTML → -1 en devalue → nil.
	if episodio["mirrors"] != nil {
		t.Errorf("mirrors = %#v, quería nil (undefined)", episodio["mirrors"])
	}
	if episodio["title"] != nil {
		t.Errorf("title = %#v, quería nil (null)", episodio["title"])
	}
	if episodio["filler"] != false {
		t.Errorf("filler = %#v, quería false", episodio["filler"])
	}

	embeds, ok := datos["embeds"].(map[string]any)
	if !ok {
		t.Fatalf("embeds = %T", datos["embeds"])
	}
	dub, ok := embeds["DUB"].([]any)
	if !ok || len(dub) != 5 {
		t.Fatalf("embeds.DUB = %#v, quería 5 servidores", embeds["DUB"])
	}
	queria := []string{"HLS", "UPNShare", "Voe", "Byse", "MP4Upload"}
	for i, nombre := range queria {
		entrada, ok := dub[i].(map[string]any)
		if !ok {
			t.Fatalf("embeds.DUB[%d] = %T", i, dub[i])
		}
		if entrada["server"] != nombre {
			t.Errorf("embeds.DUB[%d].server = %v, quería %q", i, entrada["server"], nombre)
		}
		u, _ := entrada["url"].(string)
		if !strings.HasPrefix(u, "https://") {
			t.Errorf("embeds.DUB[%d].url = %q, quería una URL https", i, u)
		}
	}

	downloads, ok := datos["downloads"].(map[string]any)
	if !ok {
		t.Fatalf("downloads = %T", datos["downloads"])
	}
	sub, ok := downloads["SUB"].([]any)
	if !ok || len(sub) != 3 {
		t.Fatalf("downloads.SUB = %#v, quería 3 descargas", downloads["SUB"])
	}
}

// TestUnflattenEspeciales cubre las constantes numéricas del formato (las que
// devalue escribe como número negativo en un slot) y los literales que van
// sueltos en el array.
func TestUnflattenEspeciales(t *testing.T) {
	// Las constantes (-1…-6) solo van en los slots de un objeto/array; los
	// literales (números, cadenas, bool, null) van sueltos en el array y se
	// referencian por índice.
	plano := []any{
		map[string]any{"u": -1, "hueco": -2, "nan": -3, "inf": -4, "mInf": -5, "cero": -6, "num": 1, "cad": 2, "flag": 3, "nada": 4, "lista": 5},
		3.5,
		"hola",
		true,
		nil,
		[]any{4, 6},
		"literal",
	}
	got, err := Unflatten(plano)
	if err != nil {
		t.Fatalf("Unflatten falló: %v", err)
	}
	obj, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("raíz = %T, quería un objeto", got)
	}

	if obj["u"] != nil {
		t.Errorf("undefined = %#v, quería nil", obj["u"])
	}
	if obj["hueco"] != nil {
		t.Errorf("hueco = %#v, quería nil", obj["hueco"])
	}
	nan, ok := obj["nan"].(float64)
	if !ok || !math.IsNaN(nan) {
		t.Errorf("NaN = %#v, quería math.NaN()", obj["nan"])
	}
	if inf, ok := obj["inf"].(float64); !ok || !math.IsInf(inf, 1) {
		t.Errorf("Infinity = %#v, quería +Inf", obj["inf"])
	}
	if inf, ok := obj["mInf"].(float64); !ok || !math.IsInf(inf, -1) {
		t.Errorf("-Infinity = %#v, quería -Inf", obj["mInf"])
	}
	cero, ok := obj["cero"].(float64)
	if !ok || cero != 0 || !math.Signbit(cero) {
		t.Errorf("-0 = %#v, quería un cero negativo", obj["cero"])
	}
	if obj["num"] != 3.5 {
		t.Errorf("número literal = %#v, quería 3.5", obj["num"])
	}
	if obj["cad"] != "hola" || obj["flag"] != true || obj["nada"] != nil {
		t.Errorf("literales = %#v/%#v/%#v", obj["cad"], obj["flag"], obj["nada"])
	}
	lista, ok := obj["lista"].([]any)
	if !ok || len(lista) != 2 || lista[0] != nil || lista[1] != "literal" {
		t.Errorf("lista = %#v, quería [nil literal]", obj["lista"])
	}
}

// TestUnflattenCiclos: un objeto que referencia a un ancestro no debe
// recursar para siempre ni perder la identidad del contenedor.
func TestUnflattenCiclos(t *testing.T) {
	plano := []any{
		map[string]any{"nombre": 1, "amigo": 2},
		"raíz",
		map[string]any{"nombre": 3, "amigo": 0},
		"ciclo",
	}
	got, err := Unflatten(plano)
	if err != nil {
		t.Fatalf("Unflatten falló: %v", err)
	}
	raiz, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("raíz = %T", got)
	}
	amigo, ok := raiz["amigo"].(map[string]any)
	if !ok {
		t.Fatalf("amigo = %T", raiz["amigo"])
	}
	if amigo["nombre"] != "ciclo" {
		t.Errorf("amigo.nombre = %v, quería ciclo", amigo["nombre"])
	}
	vuelta, ok := amigo["amigo"].(map[string]any)
	if !ok {
		t.Fatalf("amigo.amigo = %T", amigo["amigo"])
	}
	if reflect.ValueOf(vuelta).Pointer() != reflect.ValueOf(raiz).Pointer() {
		t.Error("amigo.amigo debe ser el mismo objeto raíz, no una copia")
	}
	if vuelta["nombre"] != "raíz" {
		t.Errorf("vuelta.nombre = %v, quería raíz", vuelta["nombre"])
	}
}

// TestUnflattenErrores: todo lo que no sea un tipo soportado devuelve error
// (nunca panic ni un valor a medias).
func TestUnflattenErrores(t *testing.T) {
	casos := []struct {
		nombre   string
		plano    []any
		contiene string
	}{
		{"array vacío", []any{}, "vacío"},
		{"etiqueta desconocida", []any{map[string]any{"d": 1}, []any{"Date", "2020-01-01"}}, "etiqueta no soportado"},
		{"array disperso", []any{map[string]any{"d": 1}, []any{-7, 5}}, "disperso"},
		{"referencia fuera de rango", []any{map[string]any{"x": 9}}, "fuera del array"},
		{"referencia no entera", []any{map[string]any{"x": 1.5}}, "índice entero"},
		{"constante desconocida", []any{map[string]any{"x": -8}}, "constante desconocida"},
		{"campo que no es número", []any{map[string]any{"x": "texto"}}, "no es una referencia"},
		{"elemento que no es número", []any{map[string]any{"l": 1}, []any{1, "texto"}}, "no es una referencia"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			got, err := Unflatten(c.plano)
			if err == nil {
				t.Fatalf("quería error, devolvió %#v", got)
			}
			if got != nil {
				t.Errorf("valor = %#v, quería nil junto al error", got)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}

// TestPageDataTolerancia: respuestas que no son un __data.json de datos o
// que no traen nodo de página son error, sin panic.
func TestPageDataTolerancia(t *testing.T) {
	casos := []struct {
		nombre   string
		cuerpo   string
		contiene string
	}{
		{"json inválido", `{oops`, "JSON válido"},
		{"cuerpo vacío", ``, "JSON válido"},
		{"type de error", `{"type":"error","nodes":[]}`, `type = "error"`},
		{"nodes null", `{"type":"data","nodes":null}`, "nodo de página"},
		{"nodos null", `{"type":"data","nodes":[null,null]}`, "nodo de página"},
		{"nodo skip", `{"type":"data","nodes":[{"type":"skip"}]}`, "nodo de página"},
		{"nodo error", `{"type":"data","nodes":[{"type":"error"}]}`, "nodo de página"},
		{"data sin array", `{"type":"data","nodes":[{"type":"data"}]}`, "array data"},
		{"raíz no objeto", `{"type":"data","nodes":[{"type":"data","data":[42]}]}`, "no son un objeto"},
		{"referencia rota", `{"type":"data","nodes":[{"type":"data","data":[{"x":99}]}]}`, "fuera del array"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			datos, err := PageData([]byte(c.cuerpo))
			if err == nil {
				t.Fatalf("quería error, devolvió %#v", datos)
			}
			if datos != nil {
				t.Errorf("datos = %#v, quería nil junto al error", datos)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}

// comprobarPar comprueba un {id,number} de la lista de episodios.
func comprobarPar(t *testing.T, par any, id, numero float64) {
	t.Helper()
	obj, ok := par.(map[string]any)
	if !ok {
		t.Fatalf("episodio = %T, quería un objeto", par)
	}
	if obj["id"] != id || obj["number"] != numero {
		t.Errorf("episodio = {id:%v number:%v}, quería {%v %v}", obj["id"], obj["number"], id, numero)
	}
}
