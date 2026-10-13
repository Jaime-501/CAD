package db

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Ayuda: escribir a mano un historial "antiguo" (campos de T41 en cero),
// que es la única forma de tener registros sin fecha o sin provider: Save
// siempre los rellena.
// ---------------------------------------------------------------------------

// escribirHistorialViejo deja en el directorio un historial.json con los
// registros indicados, tal como lo escribiría la app anterior a T41.
func escribirHistorialViejo(t *testing.T, dir, contenido string) string {
	t.Helper()
	ruta := filepath.Join(dir, nombreArchivo)
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", ruta, err)
	}
	return ruta
}

// ---------------------------------------------------------------------------
// Save: sello de tiempo y proveedor.
// ---------------------------------------------------------------------------

func TestSaveAsignaUpdatedAtConRelojInyectado(t *testing.T) {
	d := abrir(t, t.TempDir())
	fija := time.Unix(1_700_000_000, 0)
	d.now = func() time.Time { return fija }

	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	registros, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("registros = %+v; quería 1", registros)
	}
	if got := registros[0].UpdatedAt; got != fija.Unix() {
		t.Errorf("UpdatedAt = %d; con el reloj inyectado debía ser %d", got, fija.Unix())
	}
}

func TestSaveConservaUpdatedAtNoCero(t *testing.T) {
	d := abrir(t, t.TempDir())
	d.now = func() time.Time { return time.Unix(99, 0) }

	p := progreso("bleach", "jkanime:bleach:episode:1")
	p.UpdatedAt = 12345 // una fecha que ya trae el llamador (p. ej. Deshacer)
	if err := d.SaveProgress(p); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	registros, err := d.GetProgress("bleach")
	if err != nil || len(registros) != 1 {
		t.Fatalf("registros = %+v (err=%v)", registros, err)
	}
	if got := registros[0].UpdatedAt; got != 12345 {
		t.Errorf("UpdatedAt = %d; una fecha no cero debe conservarse (el reloj marca 99)", got)
	}

	// Una guardada idéntica no debe tocar la fecha (ni reescribir el archivo).
	if err := d.SaveProgress(p); err != nil {
		t.Fatalf("2ª SaveProgress: %v", err)
	}
	if got, err := d.GetProgress("bleach"); err != nil || len(got) != 1 || got[0].UpdatedAt != 12345 {
		t.Errorf("tras repetir la guardada = %+v (err=%v); la fecha no debe cambiar", got, err)
	}
}

func TestSaveNormalizaProviderDesdeEpisodeID(t *testing.T) {
	d := abrir(t, t.TempDir())

	// Sin provider: se rellena con el prefijo del episode_id.
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress (sin provider): %v", err)
	}
	// Con provider: se conserva el que trae el llamador.
	conProvider := progreso("bleach", "jkanime:bleach:episode:2")
	conProvider.Provider = "proveedor-dicho"
	if err := d.SaveProgress(conProvider); err != nil {
		t.Fatalf("SaveProgress (con provider): %v", err)
	}
	// Sin provider y sin ":" en el episode_id no hay nada que derivar.
	if err := d.SaveProgress(progreso("naruto", "sinprefijo")); err != nil {
		t.Fatalf("SaveProgress (sin prefijo): %v", err)
	}

	todos, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	porEpisodio := make(map[string]string, len(todos))
	for _, r := range todos {
		porEpisodio[r.EpisodeID] = r.Provider
	}
	if got := porEpisodio["jkanime:bleach:episode:1"]; got != "jkanime" {
		t.Errorf("provider = %q; debía salir del prefijo del episode_id", got)
	}
	if got := porEpisodio["jkanime:bleach:episode:2"]; got != "proveedor-dicho" {
		t.Errorf("provider = %q; el que trae el llamador se conserva", got)
	}
	if got := porEpisodio["sinprefijo"]; got != "" {
		t.Errorf("provider = %q; sin prefijo no hay nada que rellenar", got)
	}
}

func TestSaveUpdatedAtPersisteTrasReabrir(t *testing.T) {
	dir := t.TempDir()
	primera, err := Open(dir)
	if err != nil {
		t.Fatalf("1ª Open: %v", err)
	}
	fija := time.Unix(1_700_000_000, 0)
	primera.now = func() time.Time { return fija }
	if err := primera.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	if err := primera.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	segunda, err := Open(dir)
	if err != nil {
		t.Fatalf("2ª Open: %v", err)
	}
	t.Cleanup(func() { _ = segunda.Close() })
	registros, err := segunda.GetProgress("bleach")
	if err != nil || len(registros) != 1 {
		t.Fatalf("registros = %+v (err=%v)", registros, err)
	}
	if got := registros[0].UpdatedAt; got != fija.Unix() {
		t.Errorf("UpdatedAt tras reabrir = %d; el sello debía persistir (%d)", got, fija.Unix())
	}
	if got := registros[0].Provider; got != "jkanime" {
		t.Errorf("provider tras reabrir = %q; debía persistir", got)
	}
}

// ---------------------------------------------------------------------------
// LastPerAnime: el último registro por anime.
// ---------------------------------------------------------------------------

func TestLastPerAnimeUnoPorAnimeYProveedor(t *testing.T) {
	d := abrir(t, t.TempDir())

	// El mismo slug en dos proveedores son DOS entradas.
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress jkanime: %v", err)
	}
	if err := d.SaveProgress(progreso("bleach", "animeav1:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress animeav1: %v", err)
	}
	// Y un segundo episodio del mismo anime NO añade otra entrada.
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:2")); err != nil {
		t.Fatalf("SaveProgress 2º episodio: %v", err)
	}

	ultimos := d.LastPerAnime()
	if len(ultimos) != 2 {
		t.Fatalf("LastPerAnime = %+v; quería 2 entradas (una por proveedor)", ultimos)
	}
	proveedores := make(map[string]string, len(ultimos))
	for _, r := range ultimos {
		if r.AnimeID != "bleach" {
			t.Errorf("registro = %+v; todos son bleach", r)
		}
		proveedores[r.ProviderName()] = r.EpisodeID
	}
	if proveedores["jkanime"] != "jkanime:bleach:episode:2" {
		t.Errorf("jkanime = %q; debía ser su episodio más avanzado", proveedores["jkanime"])
	}
	if proveedores["animeav1"] != "animeav1:bleach:episode:1" {
		t.Errorf("animeav1 = %q", proveedores["animeav1"])
	}
}

func TestLastPerAnimeOrdenPorRecencia(t *testing.T) {
	d := abrir(t, t.TempDir())

	// Se guardan en orden contrario a como deben salir.
	for _, p := range []struct {
		anime, episodio string
		updatedAt       int64
	}{
		{"naruto", "jkanime:naruto:episode:1", 100},
		{"bleach", "jkanime:bleach:episode:1", 300},
		{"one-piece", "jkanime:one-piece:episode:1", 200},
	} {
		reg := progreso(p.anime, p.episodio)
		reg.UpdatedAt = p.updatedAt
		if err := d.SaveProgress(reg); err != nil {
			t.Fatalf("SaveProgress(%s): %v", p.anime, err)
		}
	}

	ultimos := d.LastPerAnime()
	orden := make([]string, len(ultimos))
	for i, r := range ultimos {
		orden[i] = r.AnimeID
	}
	esperado := []string{"bleach", "one-piece", "naruto"}
	for i := range esperado {
		if orden[i] != esperado[i] {
			t.Fatalf("orden = %v; quería %v (UpdatedAt descendente)", orden, esperado)
		}
	}

	// Lo devuelto son copias: mutarlo no toca el almacén.
	ultimos[0].AnimeID = "borroso"
	if trasMutar := d.LastPerAnime(); trasMutar[0].AnimeID != "bleach" {
		t.Errorf("anime = %q; mutar lo devuelto no debe tocar el almacén", trasMutar[0].AnimeID)
	}
}

func TestLastPerAnimeDesempate(t *testing.T) {
	d := abrir(t, t.TempDir())
	mismoInstante := int64(1_700_000_000)

	// bleach: 3 episodios con la MISMA fecha → gana el de mayor número.
	for _, n := range []int{1, 3, 2} {
		p := progreso("bleach", fmt.Sprintf("jkanime:bleach:episode:%d", n))
		p.EpisodeNumber = n
		p.UpdatedAt = mismoInstante
		if err := d.SaveProgress(p); err != nil {
			t.Fatalf("SaveProgress(bleach ep %d): %v", n, err)
		}
	}
	// naruto: mismo número (5) y misma fecha → gana el EpisodeID descendente.
	normal := progreso("naruto", "jkanime:naruto:episode:5")
	normal.EpisodeNumber = 5
	normal.UpdatedAt = mismoInstante
	if err := d.SaveProgress(normal); err != nil {
		t.Fatalf("SaveProgress(naruto ep): %v", err)
	}
	especial := progreso("naruto", "jkanime:naruto:special:5")
	especial.EpisodeNumber = 5
	especial.UpdatedAt = mismoInstante
	if err := d.SaveProgress(especial); err != nil {
		t.Fatalf("SaveProgress(naruto special): %v", err)
	}
	// zeta (animeav1): misma fecha que todos → orden final por proveedor asc.
	zeta := progreso("zeta", "animeav1:zeta:episode:1")
	zeta.EpisodeNumber = 1
	zeta.UpdatedAt = mismoInstante
	if err := d.SaveProgress(zeta); err != nil {
		t.Fatalf("SaveProgress(zeta): %v", err)
	}

	ultimos := d.LastPerAnime()
	esperado := []struct {
		proveedor, anime, episodio string
	}{
		{"animeav1", "zeta", "animeav1:zeta:episode:1"},
		{"jkanime", "bleach", "jkanime:bleach:episode:3"},
		{"jkanime", "naruto", "jkanime:naruto:special:5"},
	}
	if len(ultimos) != len(esperado) {
		t.Fatalf("LastPerAnime = %+v; quería %d entradas", ultimos, len(esperado))
	}
	for i, e := range esperado {
		got := ultimos[i]
		if got.ProviderName() != e.proveedor || got.AnimeID != e.anime || got.EpisodeID != e.episodio {
			t.Errorf("entrada %d = (%s, %s, %s); quería (%s, %s, %s)",
				i, got.ProviderName(), got.AnimeID, got.EpisodeID, e.proveedor, e.anime, e.episodio)
		}
	}
}

func TestLastPerAnimeRegistrosAntiguosSinFechaAlFinal(t *testing.T) {
	dir := t.TempDir()
	// El "nuevo" va el PRIMERO del archivo para que el orden salga del sort,
	// no del orden del fichero. El "viejo" es un registro anterior a T41
	// (updated_at 0 y sin provider: solo su episode_id lleva el prefijo).
	escribirHistorialViejo(t, dir, `[`+
		`{"anime_id":"nuevo","episode_id":"jkanime:nuevo:episode:1","current_time":95,"total_duration":100,"is_watched":true,"provider":"jkanime","anime_title":"","cover_url":"","episode_number":1,"updated_at":100},`+
		`{"anime_id":"viejo","episode_id":"jkanime:viejo:episode:1","current_time":95,"total_duration":100,"is_watched":true,"provider":"","anime_title":"","cover_url":"","episode_number":0,"updated_at":0}`+
		`]`)
	d := abrir(t, dir)

	ultimos := d.LastPerAnime()
	if len(ultimos) != 2 {
		t.Fatalf("LastPerAnime = %+v; quería 2 entradas", ultimos)
	}
	if ultimos[0].AnimeID != "nuevo" || ultimos[1].AnimeID != "viejo" {
		t.Errorf("orden = (%s, %s); los sin fecha (UpdatedAt == 0) van al final",
			ultimos[0].AnimeID, ultimos[1].AnimeID)
	}
	if ultimos[1].UpdatedAt != 0 {
		t.Errorf("UpdatedAt = %d; el registro antiguo no debe inventar fecha", ultimos[1].UpdatedAt)
	}
	if got := ultimos[1].ProviderName(); got != "jkanime" {
		t.Errorf("provider = %q; el prefijo del episode_id cubre a los antiguos", got)
	}
}

func TestLastPerAnimeVacioEsSliceNoNil(t *testing.T) {
	d := abrir(t, t.TempDir())

	ultimos := d.LastPerAnime()
	if ultimos == nil {
		t.Fatal("LastPerAnime sin registros no debe devolver nil")
	}
	if len(ultimos) != 0 {
		t.Errorf("LastPerAnime = %+v; quería vacío", ultimos)
	}

	// Cerrado: la firma no admite error, devuelve lista vacía (no nil).
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	cerrado := d.LastPerAnime()
	if cerrado == nil || len(cerrado) != 0 {
		t.Errorf("LastPerAnime cerrado = %#v; quería lista vacía no nil", cerrado)
	}
}

// ---------------------------------------------------------------------------
// Borrados.
// ---------------------------------------------------------------------------

func TestDeleteAnimeBorraTodosLosEpisodiosYDevuelveCuenta(t *testing.T) {
	d := abrir(t, t.TempDir())
	for _, e := range []int{1, 2, 3} {
		if err := d.SaveProgress(progreso("bleach", fmt.Sprintf("jkanime:bleach:episode:%d", e))); err != nil {
			t.Fatalf("SaveProgress(bleach ep %d): %v", e, err)
		}
	}
	if err := d.SaveProgress(progreso("naruto", "jkanime:naruto:episode:1")); err != nil {
		t.Fatalf("SaveProgress(naruto): %v", err)
	}

	borrados, err := d.DeleteAnime("jkanime", "bleach")
	if err != nil {
		t.Fatalf("DeleteAnime: %v", err)
	}
	if borrados != 3 {
		t.Errorf("borrados = %d; bleach tenía 3 episodios", borrados)
	}

	bleach, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress(bleach): %v", err)
	}
	if len(bleach) != 0 {
		t.Errorf("bleach = %+v; debía quedar sin registros", bleach)
	}
	naruto, err := d.GetProgress("naruto")
	if err != nil || len(naruto) != 1 {
		t.Errorf("naruto = %+v (err=%v); no debe tocarse", naruto, err)
	}
}

func TestDeleteAnimeNoAfectaOtroProveedorConMismoSlug(t *testing.T) {
	d := abrir(t, t.TempDir())
	for _, e := range []int{1, 2} {
		if err := d.SaveProgress(progreso("bleach", fmt.Sprintf("jkanime:bleach:episode:%d", e))); err != nil {
			t.Fatalf("SaveProgress jkanime: %v", err)
		}
	}
	if err := d.SaveProgress(progreso("bleach", "animeav1:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress animeav1: %v", err)
	}

	borrados, err := d.DeleteAnime("jkanime", "bleach")
	if err != nil {
		t.Fatalf("DeleteAnime: %v", err)
	}
	if borrados != 2 {
		t.Errorf("borrados = %d; solo los de jkanime (2)", borrados)
	}

	todos, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(todos) != 1 || todos[0].ProviderName() != "animeav1" {
		t.Errorf("restantes = %+v; el mismo slug de animeav1 no debe borrarse", todos)
	}
}

func TestDeleteAnimeInexistenteDevuelveCeroSinError(t *testing.T) {
	t.Run("con registros no reescribe el archivo", func(t *testing.T) {
		dir := t.TempDir()
		d := abrir(t, dir)
		if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
			t.Fatalf("SaveProgress: %v", err)
		}
		ruta := filepath.Join(dir, nombreArchivo)
		antes, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}

		borrados, err := d.DeleteAnime("jkanime", "one-piece")
		if err != nil {
			t.Fatalf("DeleteAnime inexistente: %v", err)
		}
		if borrados != 0 {
			t.Errorf("borrados = %d; no existe tal anime", borrados)
		}
		despues, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatalf("ReadFile tras el borrado: %v", err)
		}
		if !bytes.Equal(antes, despues) {
			t.Error("el archivo cambió con 0 borrados: no debe reescribirse")
		}
	})

	t.Run("sin registros no crea el archivo", func(t *testing.T) {
		dir := t.TempDir()
		d := abrir(t, dir)

		borrados, err := d.DeleteAnime("jkanime", "bleach")
		if err != nil || borrados != 0 {
			t.Fatalf("DeleteAnime sobre vacío = %d, %v; quería 0 sin error", borrados, err)
		}
		entradas, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}
		if len(entradas) != 0 {
			t.Errorf("entradas = %v; un borrado sin cambios no debe escribir nada", entradas)
		}
	})
}

func TestDeleteProveedor(t *testing.T) {
	dir := t.TempDir()
	// Un registro anterior a T41, sin provider en el JSON: lo borra el
	// prefijo del episode_id (ProviderName).
	escribirHistorialViejo(t, dir, `[`+
		`{"anime_id":"legacy","episode_id":"jkanime:legacy:episode:1","current_time":10,"total_duration":100,"is_watched":false,"provider":"","anime_title":"","cover_url":"","episode_number":0,"updated_at":0}`+
		`]`)
	d := abrir(t, dir)

	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress jkanime:bleach: %v", err)
	}
	if err := d.SaveProgress(progreso("bleach", "animeav1:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress animeav1:bleach: %v", err)
	}
	if err := d.SaveProgress(progreso("naruto", "jkanime:naruto:episode:1")); err != nil {
		t.Fatalf("SaveProgress jkanime:naruto: %v", err)
	}

	borrados, err := d.DeleteProvider("jkanime")
	if err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	if borrados != 3 {
		t.Errorf("borrados = %d; jkanime tenía 3 (legacy + bleach + naruto)", borrados)
	}
	restantes, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(restantes) != 1 || restantes[0].ProviderName() != "animeav1" {
		t.Errorf("restantes = %+v; solo debe quedar animeav1", restantes)
	}

	// Un proveedor que no existe no es error.
	nada, err := d.DeleteProvider("desconocido")
	if err != nil || nada != 0 {
		t.Errorf("DeleteProvider(desconocido) = %d, %v; quería 0 sin error", nada, err)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	d := abrir(t, dir)
	for _, p := range []struct {
		anime, episodio string
	}{
		{"bleach", "jkanime:bleach:episode:1"},
		{"bleach", "animeav1:bleach:episode:1"},
		{"naruto", "jkanime:naruto:episode:1"},
	} {
		if err := d.SaveProgress(progreso(p.anime, p.episodio)); err != nil {
			t.Fatalf("SaveProgress(%s): %v", p.episodio, err)
		}
	}

	borrados, err := d.Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if borrados != 3 {
		t.Errorf("borrados = %d; Clear debía contar los 3 registros", borrados)
	}
	todos, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("historial = %+v; Clear debe vaciarlo entero", todos)
	}
	if ultimos := d.LastPerAnime(); len(ultimos) != 0 {
		t.Errorf("LastPerAnime tras Clear = %+v; quería vacío", ultimos)
	}
	// Repetir sobre lo ya vacío: 0 sin error.
	otro, err := d.Clear()
	if err != nil || otro != 0 {
		t.Errorf("Clear repetido = %d, %v; quería 0 sin error", otro, err)
	}

	// Y queda vacío en disco también.
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reabierta, err := Open(dir)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	defer reabierta.Close()
	if enDisco, err := reabierta.GetProgress(""); err != nil || len(enDisco) != 0 {
		t.Errorf("en disco = %+v (err=%v); Clear debe persistir", enDisco, err)
	}
	sinTemporales(t, dir)
}

func TestBorradosPersistenAlReabrir(t *testing.T) {
	dir := t.TempDir()
	primera, err := Open(dir)
	if err != nil {
		t.Fatalf("1ª Open: %v", err)
	}
	for _, p := range []struct {
		anime, episodio string
	}{
		{"bleach", "jkanime:bleach:episode:1"},
		{"bleach", "jkanime:bleach:episode:2"},
		{"bleach", "animeav1:bleach:episode:1"},
		{"naruto", "jkanime:naruto:episode:1"},
	} {
		if err := primera.SaveProgress(progreso(p.anime, p.episodio)); err != nil {
			t.Fatalf("SaveProgress(%s): %v", p.episodio, err)
		}
	}
	if n, err := primera.DeleteAnime("jkanime", "bleach"); err != nil || n != 2 {
		t.Fatalf("DeleteAnime = %d, %v; quería 2", n, err)
	}
	if n, err := primera.DeleteProvider("animeav1"); err != nil || n != 1 {
		t.Fatalf("DeleteProvider = %d, %v; quería 1", n, err)
	}
	if err := primera.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	segunda, err := Open(dir)
	if err != nil {
		t.Fatalf("2ª Open: %v", err)
	}
	quedan, err := segunda.GetProgress("")
	if err != nil || len(quedan) != 1 || quedan[0].AnimeID != "naruto" {
		t.Fatalf("en disco = %+v (err=%v); solo debía quedar naruto", quedan, err)
	}
	if n, err := segunda.Clear(); err != nil || n != 1 {
		t.Fatalf("Clear = %d, %v; quería 1", n, err)
	}
	if err := segunda.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	tercera, err := Open(dir)
	if err != nil {
		t.Fatalf("3ª Open: %v", err)
	}
	defer tercera.Close()
	if enDisco, err := tercera.GetProgress(""); err != nil || len(enDisco) != 0 {
		t.Errorf("en disco = %+v (err=%v); Clear debía dejar el historial vacío", enDisco, err)
	}
	sinTemporales(t, dir)
}

// TestBorradoConcurrenteConSave: el reproductor guarda mientras la UI borra
// (y lee el historial). Con -race no debe aparecer ninguna carrera y el
// archivo debe quedar íntegro.
func TestBorradoConcurrenteConSave(t *testing.T) {
	dir := t.TempDir()
	d := abrir(t, dir)

	// naruto: 2 registros que las borradoras quitarán. Nadie los re-crea, así
	// que la suma de lo que devuelven es exactamente 2.
	for _, e := range []int{1, 2} {
		if err := d.SaveProgress(progreso("naruto", fmt.Sprintf("jkanime:naruto:episode:%d", e))); err != nil {
			t.Fatalf("SaveProgress(naruto): %v", err)
		}
	}

	const (
		guardadoras   = 4
		porGuardadora = 20
		borradoras    = 3
		lectoras      = 2
	)
	var (
		wg       sync.WaitGroup
		borrados atomic.Int64
	)

	for g := 0; g < guardadoras; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < porGuardadora; i++ {
				p := progreso("bleach", fmt.Sprintf("jkanime:bleach:episode:%d-%d", g, i))
				if err := d.SaveProgress(p); err != nil {
					t.Errorf("SaveProgress(g=%d,i=%d): %v", g, i, err)
					return
				}
			}
		}(g)
	}
	for b := 0; b < borradoras; b++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := d.DeleteAnime("jkanime", "naruto")
			if err != nil {
				t.Errorf("DeleteAnime: %v", err)
				return
			}
			borrados.Add(int64(n))
		}()
	}
	for l := 0; l < lectoras; l++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if u := d.LastPerAnime(); u == nil {
				t.Error("LastPerAnime no debe devolver nil")
			}
		}()
	}
	wg.Wait()

	if total := borrados.Load(); total != 2 {
		t.Errorf("suma de borrados = %d; naruto tenía 2 registros y nadie los re-crea", total)
	}
	bleach, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress(bleach): %v", err)
	}
	if len(bleach) != guardadoras*porGuardadora {
		t.Errorf("bleach = %d registros; las borradas no deben tocarlo", len(bleach))
	}
	if naruto, err := d.GetProgress("naruto"); err != nil || len(naruto) != 0 {
		t.Errorf("naruto = %+v (err=%v); debía quedar vacío", naruto, err)
	}

	// El archivo en disco está íntegro: se reabre y vuelve a parsear.
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reabierta, err := Open(dir)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	defer reabierta.Close()
	enDisco, err := reabierta.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress tras reabrir: %v", err)
	}
	if len(enDisco) != guardadoras*porGuardadora {
		t.Errorf("en disco = %d registros; esperaba %d", len(enDisco), guardadoras*porGuardadora)
	}
	sinTemporales(t, dir)
}
