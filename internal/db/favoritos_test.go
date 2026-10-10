package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miapp/internal/models"
)

// ---------------------------------------------------------------------------
// Ayudas.
// ---------------------------------------------------------------------------

// abrirFavoritos monta un almacén de favoritos en dir (no tiene Close: no
// hay nada que cerrar).
func abrirFavoritos(t *testing.T, dir string) *Favorites {
	t.Helper()
	f, err := OpenFavorites(dir)
	if err != nil {
		t.Fatalf("OpenFavorites(%q): %v", dir, err)
	}
	return f
}

// ---------------------------------------------------------------------------
// Open.
// ---------------------------------------------------------------------------

func TestOpenFavoritosDirectorioVacio(t *testing.T) {
	if _, err := OpenFavorites(""); !errors.Is(err, ErrDirectorioVacio) {
		t.Errorf(`OpenFavorites("") = %v; quería ErrDirectorioVacio (P3: este paquete no decide rutas)`, err)
	}

	t.Run("crea el directorio si falta", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "anidado", "miapp")
		f := abrirFavoritos(t, dir)
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("el directorio no se creó (MkdirAll): %v", err)
		}
		if lista := f.List(); lista == nil || len(lista) != 0 {
			t.Errorf("List = %#v; quería lista vacía no nil", lista)
		}
	})
}

func TestFavoritosInicialVacio(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)

	lista := f.List()
	if lista == nil {
		t.Fatal("List sin favoritos no debe devolver nil")
	}
	if len(lista) != 0 {
		t.Errorf("List = %+v; quería vacío", lista)
	}
	if f.Has("jkanime", "bleach") {
		t.Error("Has sobre almacén vacío debe ser false")
	}

	// Nada se escribe hasta que alguien añade (mismo criterio que Open del
	// historial: no se crea un archivo vacío por deporte).
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entradas) != 0 {
		t.Errorf("entradas = %v; OpenFavorites no debe crear nada", entradas)
	}
}

// ---------------------------------------------------------------------------
// Add.
// ---------------------------------------------------------------------------

func TestAddNuevoDevuelveTrueYPersiste(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)
	fija := time.Unix(1_700_000_000, 0)
	f.now = func() time.Time { return fija }

	nuevo, err := f.Add(models.Favorite{
		Provider: "jkanime",
		AnimeID:  "bleach",
		Title:    "Bleach",
		CoverURL: "https://cdn.example/bleach.jpg",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !nuevo {
		t.Error("Add de un favorito nuevo debe devolver true")
	}
	if !f.Has("jkanime", "bleach") {
		t.Error("Has tras Add debe ser true")
	}

	lista := f.List()
	if len(lista) != 1 {
		t.Fatalf("List = %+v; quería 1 favorito", lista)
	}
	if lista[0].AddedAt != fija.Unix() {
		t.Errorf("AddedAt = %d; con AddedAt 0 debía sellarse con el reloj (%d)",
			lista[0].AddedAt, fija.Unix())
	}
	if lista[0].Title != "Bleach" || lista[0].CoverURL != "https://cdn.example/bleach.jpg" {
		t.Errorf("favorito = %+v; se perdió algún campo", lista[0])
	}

	// Persiste: el archivo existe, es JSON válido, indentado y con el
	// favorito dentro.
	datos, err := os.ReadFile(filepath.Join(dir, nombreFavoritos))
	if err != nil {
		t.Fatalf("ReadFile(favoritos.json): %v", err)
	}
	if !bytes.Contains(datos, []byte("\n    \"provider\"")) {
		t.Errorf("el JSON debe ir indentado: %s", datos)
	}
	var enDisco []models.Favorite
	if err := json.Unmarshal(datos, &enDisco); err != nil {
		t.Fatalf("favoritos.json no es JSON válido: %v (%s)", err, datos)
	}
	if len(enDisco) != 1 || enDisco[0].Key() != "jkanime:bleach" || enDisco[0].AddedAt != fija.Unix() {
		t.Errorf("en disco = %+v; quería el favorito recién guardado", enDisco)
	}
	sinTemporales(t, dir)
}

func TestAddExistenteDevuelveFalseConservaAddedAtActualizaTitulo(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)

	original := models.Favorite{
		Provider: "jkanime",
		AnimeID:  "bleach",
		Title:    "Título viejo",
		CoverURL: "https://cdn.example/vieja.jpg",
		AddedAt:  500,
	}
	if nuevo, err := f.Add(original); err != nil || !nuevo {
		t.Fatalf("1ª Add = %v, %v; quería true sin error", nuevo, err)
	}

	// El usuario vuelve a puntuar el mismo anime desde otro sitio: misma
	// clave, ficha nueva y otra fecha de alta.
	if nuevo, err := f.Add(models.Favorite{
		Provider: "jkanime",
		AnimeID:  "bleach",
		Title:    "Título nuevo",
		CoverURL: "https://cdn.example/nueva.jpg",
		AddedAt:  999,
	}); err != nil {
		t.Fatalf("2ª Add: %v", err)
	} else if nuevo {
		t.Error("Add sobre un favorito existente debe devolver false (nunca toggle)")
	}

	lista := f.List()
	if len(lista) != 1 {
		t.Fatalf("List = %+v; la 2ª Add no debe duplicar", lista)
	}
	if lista[0].AddedAt != 500 {
		t.Errorf("AddedAt = %d; el existente debe conservar su fecha de alta (500)", lista[0].AddedAt)
	}
	if lista[0].Title != "Título nuevo" || lista[0].CoverURL != "https://cdn.example/nueva.jpg" {
		t.Errorf("ficha = %+v; Title y CoverURL deben actualizarse", lista[0])
	}

	// Y los cambios se persisten: al reabrir siguen ahí.
	reabierta := abrirFavoritos(t, dir)
	enDisco := reabierta.List()
	if len(enDisco) != 1 || enDisco[0].AddedAt != 500 ||
		enDisco[0].Title != "Título nuevo" || enDisco[0].CoverURL != "https://cdn.example/nueva.jpg" {
		t.Errorf("en disco = %+v; la actualización debía persistir", enDisco)
	}
}

func TestAddMismoSlugDistintoProveedorSonDistintos(t *testing.T) {
	f := abrirFavoritos(t, t.TempDir())

	if nuevo, err := f.Add(models.Favorite{Provider: "jkanime", AnimeID: "bleach", Title: "Bleach (JK)"}); err != nil || !nuevo {
		t.Fatalf("Add jkanime = %v, %v", nuevo, err)
	}
	if nuevo, err := f.Add(models.Favorite{Provider: "animeav1", AnimeID: "bleach", Title: "Bleach (AV1)"}); err != nil || !nuevo {
		t.Fatalf("Add animeav1 = %v, %v", nuevo, err)
	}

	lista := f.List()
	if len(lista) != 2 {
		t.Fatalf("List = %+v; el mismo slug en dos proveedores son DOS favoritos", lista)
	}
	if !f.Has("jkanime", "bleach") || !f.Has("animeav1", "bleach") {
		t.Errorf("Has = %v/%v; ambos deben estar", f.Has("jkanime", "bleach"), f.Has("animeav1", "bleach"))
	}

	// Quitar uno no toca al otro.
	if ok, err := f.Remove("jkanime", "bleach"); err != nil || !ok {
		t.Fatalf("Remove(jkanime) = %v, %v", ok, err)
	}
	if !f.Has("animeav1", "bleach") {
		t.Error("Remove de un proveedor no debe borrar el mismo slug del otro")
	}
}

func TestAddRechazaProveedorOIDVacio(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)

	casos := []struct {
		nombre string
		fav    models.Favorite
	}{
		{"sin provider", models.Favorite{Provider: "", AnimeID: "bleach", Title: "Bleach"}},
		{"sin anime_id", models.Favorite{Provider: "jkanime", AnimeID: "", Title: "Nada"}},
		{"sin ninguno", models.Favorite{}},
	}
	for _, c := range casos {
		if _, err := f.Add(c.fav); err == nil {
			t.Errorf("Add(%s) no debe dejar pasar la clave vacía", c.nombre)
		} else if !strings.Contains(err.Error(), "provider") || !strings.Contains(err.Error(), "anime_id") {
			t.Errorf("Add(%s) = %q; el mensaje debe nombrar los campos que faltan", c.nombre, err)
		}
	}

	if lista := f.List(); len(lista) != 0 {
		t.Errorf("List = %+v; no debe guardarse nada inválido", lista)
	}
	// Y no se crea ningún archivo.
	if _, err := os.Stat(filepath.Join(dir, nombreFavoritos)); !os.IsNotExist(err) {
		t.Errorf("favoritos.json no debe existir tras un Add inválido (err=%v)", err)
	}
}

// ---------------------------------------------------------------------------
// Remove y Has.
// ---------------------------------------------------------------------------

func TestRemoveExistenteYInexistente(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)
	for _, fav := range []models.Favorite{
		{Provider: "jkanime", AnimeID: "bleach", AddedAt: 100},
		{Provider: "jkanime", AnimeID: "naruto", AddedAt: 200},
	} {
		if nuevo, err := f.Add(fav); err != nil || !nuevo {
			t.Fatalf("Add(%s) = %v, %v", fav.AnimeID, nuevo, err)
		}
	}

	// Existe ⇒ true y desaparece.
	if ok, err := f.Remove("jkanime", "bleach"); err != nil {
		t.Fatalf("Remove(existente): %v", err)
	} else if !ok {
		t.Fatal("Remove de un favorito existente debe devolver true")
	}
	if f.Has("jkanime", "bleach") {
		t.Error("Has tras Remove debe ser false")
	}
	if lista := f.List(); len(lista) != 1 || lista[0].AnimeID != "naruto" {
		t.Errorf("List = %+v; solo debe quedar naruto", lista)
	}

	// No existen ⇒ false SIN reescribir el archivo.
	antes, err := os.ReadFile(filepath.Join(dir, nombreFavoritos))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, caso := range []struct{ proveedor, anime string }{
		{"jkanime", "bleach"},    // el que acaba de salir
		{"jkanime", "one-piece"}, // que nunca estuvo
		{"animeav1", "naruto"},   // mismo slug, otro proveedor
	} {
		if ok, err := f.Remove(caso.proveedor, caso.anime); err != nil {
			t.Fatalf("Remove(%s/%s): %v", caso.proveedor, caso.anime, err)
		} else if ok {
			t.Errorf("Remove(%s/%s) = true; no existe tal favorito", caso.proveedor, caso.anime)
		}
	}
	despues, err := os.ReadFile(filepath.Join(dir, nombreFavoritos))
	if err != nil {
		t.Fatalf("ReadFile tras los Remove fallidos: %v", err)
	}
	if !bytes.Equal(antes, despues) {
		t.Error("el archivo cambió con Remove inexistente: no debe reescribirse")
	}

	// Y el que quedó sigue intacto en disco.
	reabierta := abrirFavoritos(t, dir)
	if lista := reabierta.List(); len(lista) != 1 || lista[0].Key() != "jkanime:naruto" {
		t.Errorf("en disco = %+v; solo debía quedar naruto", lista)
	}
}

func TestHas(t *testing.T) {
	f := abrirFavoritos(t, t.TempDir())

	if f.Has("jkanime", "bleach") {
		t.Error("Has sobre vacío debe ser false")
	}
	if _, err := f.Add(models.Favorite{Provider: "jkanime", AnimeID: "bleach", AddedAt: 1}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !f.Has("jkanime", "bleach") {
		t.Error("Has tras Add debe ser true")
	}
	if f.Has("animeav1", "bleach") {
		t.Error("el mismo slug de otro proveedor NO está en favoritos")
	}
	if f.Has("jkanime", "naruto") {
		t.Error("otro slug del mismo proveedor NO está en favoritos")
	}
	if ok, err := f.Remove("jkanime", "bleach"); err != nil || !ok {
		t.Fatalf("Remove = %v, %v", ok, err)
	}
	if f.Has("jkanime", "bleach") {
		t.Error("Has tras Remove debe ser false")
	}
}

// ---------------------------------------------------------------------------
// List.
// ---------------------------------------------------------------------------

func TestListOrdenAddedAtDescDesempateClave(t *testing.T) {
	f := abrirFavoritos(t, t.TempDir())

	// Se añaden en orden "desordenado" a propósito.
	anadidos := []models.Favorite{
		{Provider: "jkanime", AnimeID: "zeta", AddedAt: 100},
		{Provider: "animeav1", AnimeID: "alpha", AddedAt: 100},
		{Provider: "jkanime", AnimeID: "beta", AddedAt: 200},
		{Provider: "jkanime", AnimeID: "alpha", AddedAt: 100},
	}
	for _, fav := range anadidos {
		if nuevo, err := f.Add(fav); err != nil || !nuevo {
			t.Fatalf("Add(%s) = %v, %v", fav.Key(), nuevo, err)
		}
	}

	esperado := []string{
		"jkanime:beta",   // AddedAt 200: lo más reciente primero
		"animeav1:alpha", // AddedAt 100, empate → Key() ascendente
		"jkanime:alpha",
		"jkanime:zeta",
	}
	lista := f.List()
	if len(lista) != len(esperado) {
		t.Fatalf("List = %+v; quería %d favoritos", lista, len(esperado))
	}
	for i, k := range esperado {
		if got := lista[i].Key(); got != k {
			t.Errorf("List[%d] = %s; quería %s (AddedAt desc, empate por Key asc)", i, got, k)
		}
	}

	// Son copias: mutar lo devuelto no toca el almacén.
	lista[0].Title = "borroso"
	if trasMutar := f.List(); trasMutar[0].Title == "borroso" {
		t.Errorf("Title = %q; mutar lo devuelto no debe tocar el almacén", trasMutar[0].Title)
	}
}

// ---------------------------------------------------------------------------
// Persistencia, corruptos y escritura atómica.
// ---------------------------------------------------------------------------

func TestFavoritosPersistenTrasReabrir(t *testing.T) {
	dir := t.TempDir()
	primera := abrirFavoritos(t, dir)
	for _, fav := range []models.Favorite{
		{Provider: "jkanime", AnimeID: "bleach", AddedAt: 100},
		{Provider: "animeav1", AnimeID: "bleach", AddedAt: 200},
		{Provider: "jkanime", AnimeID: "naruto", AddedAt: 300},
	} {
		if nuevo, err := primera.Add(fav); err != nil || !nuevo {
			t.Fatalf("Add(%s) = %v, %v", fav.Key(), nuevo, err)
		}
	}
	if ok, err := primera.Remove("jkanime", "bleach"); err != nil || !ok {
		t.Fatalf("Remove = %v, %v", ok, err)
	}

	segunda := abrirFavoritos(t, dir)
	lista := segunda.List()
	if len(lista) != 2 {
		t.Fatalf("List tras reabrir = %+v; quería los 2 favoritos de disco", lista)
	}
	if lista[0].Key() != "jkanime:naruto" || lista[0].AddedAt != 300 {
		t.Errorf("List[0] = %+v; AddedAt debe sobrevivir el reinicio", lista[0])
	}
	if lista[1].Key() != "animeav1:bleach" || lista[1].AddedAt != 200 {
		t.Errorf("List[1] = %+v", lista[1])
	}
	if segunda.Has("jkanime", "bleach") {
		t.Error("el favorito borrado no debe resucitar al reabrir")
	}
	sinTemporales(t, dir)
}

func TestFavoritosArchivoCorruptoSeRenombraABakYEmpiezaVacio(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, nombreFavoritos)
	basura := []byte("{esto no es un array de favoritos, ")
	if err := os.WriteFile(ruta, basura, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	f, err := OpenFavorites(dir)
	if err != nil {
		t.Fatalf("unos favoritos corruptos NO deben impedir abrir: %v", err)
	}
	if lista := f.List(); lista == nil || len(lista) != 0 {
		t.Errorf("List = %#v; un corrupto empieza vacío (no nil)", lista)
	}

	// El contenido ilegible se conserva en .bak (no se pierde nada).
	respaldo := ruta + sufijoRespaldo
	leido, err := os.ReadFile(respaldo)
	if err != nil {
		t.Fatalf("el respaldo .bak no existe: %v", err)
	}
	if !bytes.Equal(leido, basura) {
		t.Errorf(".bak = %q; quería la basura original (%q)", leido, basura)
	}

	// Y al añadir se crea uno nuevo válido.
	if nuevo, err := f.Add(models.Favorite{Provider: "jkanime", AnimeID: "bleach", AddedAt: 1}); err != nil || !nuevo {
		t.Fatalf("Add tras el respaldo = %v, %v", nuevo, err)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("ReadFile del favoritos nuevo: %v", err)
	}
	var nuevos []models.Favorite
	if err := json.Unmarshal(datos, &nuevos); err != nil {
		t.Fatalf("el favoritos nuevo no es JSON válido: %v (%s)", err, datos)
	}
	if len(nuevos) != 1 || nuevos[0].Key() != "jkanime:bleach" {
		t.Errorf("nuevos = %+v; quería el favorito recién guardado", nuevos)
	}
	sinTemporales(t, dir)

	// Reabrir el ya reparado sigue funcionando sin crear otro .bak.
	segunda, err := OpenFavorites(dir)
	if err != nil {
		t.Fatalf("2ª OpenFavorites: %v", err)
	}
	if lista := segunda.List(); len(lista) != 1 {
		t.Errorf("List = %+v; quería el favorito guardado tras la reparación", lista)
	}
}

func TestFavoritosEscrituraAtomicaNoDejaTemporales(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)

	// Unas cuantas vueltas de Add/Remove: cada una escribe a disco.
	for i := 0; i < 5; i++ {
		if _, err := f.Add(models.Favorite{Provider: "jkanime", AnimeID: fmt.Sprintf("anime-%d", i), AddedAt: int64(100 + i)}); err != nil {
			t.Fatalf("Add(anime-%d): %v", i, err)
		}
	}
	for i := 0; i < 5; i += 2 {
		if ok, err := f.Remove("jkanime", fmt.Sprintf("anime-%d", i)); err != nil || !ok {
			t.Fatalf("Remove(anime-%d) = %v, %v", i, ok, err)
		}
	}
	// Un Remove inexistente tampoco deja basura.
	if ok, err := f.Remove("jkanime", "nunca-existio"); err != nil || ok {
		t.Fatalf("Remove(inexistente) = %v, %v", ok, err)
	}
	sinTemporales(t, dir)

	// Y el archivo final está íntegro: 2 favoritos (de los 5, se borraron
	// los tres pares), en el orden de AddedAt.
	datos, err := os.ReadFile(filepath.Join(dir, nombreFavoritos))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var enDisco []models.Favorite
	if err := json.Unmarshal(datos, &enDisco); err != nil {
		t.Fatalf("favoritos.json no parsea: %v (%s)", err, datos)
	}
	if len(enDisco) != 2 || enDisco[0].AnimeID != "anime-1" || enDisco[1].AnimeID != "anime-3" {
		t.Errorf("en disco = %+v; quería los 2 favoritos vivos (anime-1 y anime-3)", enDisco)
	}
}

// ---------------------------------------------------------------------------
// Concurrencia (con -race).
// ---------------------------------------------------------------------------

// TestFavoritosConcurrentes: la UI añade y quita estrellas mientras se lee
// la pestaña de favoritos. Con -race no debe aparecer ninguna carrera y el
// archivo debe quedar íntegro.
func TestFavoritosConcurrentes(t *testing.T) {
	dir := t.TempDir()
	f := abrirFavoritos(t, dir)

	const borrables = 12
	for i := 0; i < borrables; i++ {
		if _, err := f.Add(models.Favorite{Provider: "jkanime", AnimeID: fmt.Sprintf("borra-%02d", i), AddedAt: int64(1000 + i)}); err != nil {
			t.Fatalf("Add(borra-%02d): %v", i, err)
		}
	}

	const (
		anadidoras   = 4
		porAnadidora = 15
		peleadores   = 3
		peleas       = 10
		borradoras   = 3
		lectoras     = 2
	)
	var (
		wg         sync.WaitGroup
		eliminados atomic.Int64
	)

	// Añaden claves DISTINTAS: al final debe estar cada una exactamente una vez.
	for g := 0; g < anadidoras; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < porAnadidora; i++ {
				fav := models.Favorite{
					Provider: "animeav1",
					AnimeID:  fmt.Sprintf("g%d-%d", g, i),
					Title:    fmt.Sprintf("G%d %d", g, i),
				}
				nuevo, err := f.Add(fav)
				if err != nil {
					t.Errorf("Add(%s): %v", fav.Key(), err)
					return
				}
				if !nuevo {
					t.Errorf("Add(%s) = false; debía ser una clave nueva", fav.Key())
					return
				}
				if !f.Has(fav.Provider, fav.AnimeID) {
					t.Errorf("Has(%s) tras su Add debe ser true", fav.Key())
					return
				}
			}
		}(g)
	}
	// Peleas por la MISMA clave: el mutex debe serializarlas sin duplicarla.
	for g := 0; g < peleadores; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < peleas; i++ {
				fav := models.Favorite{
					Provider: "jkanime",
					AnimeID:  "pelea",
					Title:    fmt.Sprintf("pelea-%d-%d", g, i),
					AddedAt:  int64(g*100 + i + 1),
				}
				if _, err := f.Add(fav); err != nil {
					t.Errorf("Add(pelea g=%d,i=%d): %v", g, i, err)
					return
				}
			}
		}(g)
	}
	// Tres borrando la misma lista de favoritos: la suma de "true" es exacta.
	for b := 0; b < borradoras; b++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < borrables; i++ {
				ok, err := f.Remove("jkanime", fmt.Sprintf("borra-%02d", i))
				if err != nil {
					t.Errorf("Remove(borra-%02d): %v", i, err)
					return
				}
				if ok {
					eliminados.Add(1)
				}
			}
		}()
	}
	// Lectoras entremezcladas (la pestaña de favoritos se repinta sola).
	for l := 0; l < lectoras; l++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				if f.List() == nil {
					t.Error("List no debe devolver nil")
					return
				}
				f.Has("jkanime", "pelea")
			}
		}()
	}
	wg.Wait()

	if total := eliminados.Load(); total != borrables {
		t.Errorf("suma de borrados = %d; los %d borrables existían y nadie los re-crea", total, borrables)
	}
	for i := 0; i < borrables; i++ {
		if f.Has("jkanime", fmt.Sprintf("borra-%02d", i)) {
			t.Errorf("borra-%02d sigue en favoritos", i)
			break
		}
	}
	if !f.Has("jkanime", "pelea") {
		t.Error("la clave peleada debía sobrevivir exactamente una vez")
	}
	esperados := anadidoras*porAnadidora + 1 // + la clave peleada
	lista := f.List()
	if len(lista) != esperados {
		t.Fatalf("List = %d favoritos; esperaba %d (se han perdido Add)", len(lista), esperados)
	}

	// El archivo en disco está íntegro: se reabre y vuelve a parsear.
	reabierta := abrirFavoritos(t, dir)
	if enDisco := reabierta.List(); len(enDisco) != esperados {
		t.Errorf("en disco = %d favoritos; esperaba %d", len(enDisco), esperados)
	}
	sinTemporales(t, dir)
}

// ---------------------------------------------------------------------------
// Convivencia con el historial en el mismo directorio.
// ---------------------------------------------------------------------------

func TestFavoritosYHistorialConviveEnMismoDirectorio(t *testing.T) {
	dir := t.TempDir()
	historial := abrir(t, dir)
	favoritos := abrirFavoritos(t, dir)

	// Escribir en uno no toca al otro (archivos distintos, mismo dir).
	if err := historial.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	if nuevo, err := favoritos.Add(models.Favorite{Provider: "jkanime", AnimeID: "bleach", Title: "Bleach", AddedAt: 100}); err != nil || !nuevo {
		t.Fatalf("Add = %v, %v", nuevo, err)
	}

	comprueba := func(etapa string) {
		t.Helper()
		registros, err := historial.GetProgress("bleach")
		if err != nil || len(registros) != 1 || registros[0].EpisodeID != "jkanime:bleach:episode:1" {
			t.Fatalf("%s: historial = %+v (err=%v); el favorito no debe tocarlo", etapa, registros, err)
		}
		if !favoritos.Has("jkanime", "bleach") {
			t.Fatalf("%s: el favorito no debe perderse por el historial", etapa)
		}
	}
	comprueba("en memoria")

	// Cada archivo lleva su propio JSON.
	histDatos, err := os.ReadFile(filepath.Join(dir, nombreArchivo))
	if err != nil {
		t.Fatalf("ReadFile(historial.json): %v", err)
	}
	var histJSON []models.WatchProgress
	if err := json.Unmarshal(histDatos, &histJSON); err != nil || len(histJSON) != 1 {
		t.Fatalf("historial.json = %s (err=%v); debía tener su único registro", histDatos, err)
	}
	favDatos, err := os.ReadFile(filepath.Join(dir, nombreFavoritos))
	if err != nil {
		t.Fatalf("ReadFile(favoritos.json): %v", err)
	}
	var favJSON []models.Favorite
	if err := json.Unmarshal(favDatos, &favJSON); err != nil || len(favJSON) != 1 {
		t.Fatalf("favoritos.json = %s (err=%v); debía tener su único favorito", favDatos, err)
	}

	// Quitar el favorito no deja el historial sin su registro.
	if ok, err := favoritos.Remove("jkanime", "bleach"); err != nil || !ok {
		t.Fatalf("Remove = %v, %v", ok, err)
	}
	if registros, err := historial.GetProgress("bleach"); err != nil || len(registros) != 1 {
		t.Errorf("historial tras Remove = %+v (err=%v); no debe tocarse", registros, err)
	}

	// Y al reabrir ambos, cada uno sigue con lo suyo.
	historial2 := abrir(t, dir)
	favoritos2 := abrirFavoritos(t, dir)
	if registros, err := historial2.GetProgress("bleach"); err != nil || len(registros) != 1 {
		t.Errorf("historial tras reabrir = %+v (err=%v)", registros, err)
	}
	if lista := favoritos2.List(); len(lista) != 0 {
		t.Errorf("favoritos tras reabrir = %+v; debía seguir vacío", lista)
	}
	sinTemporales(t, dir)
}
