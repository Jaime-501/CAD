package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"miapp/internal/models"
)

// TestMain descarta los logs: los tests no deben ensuciar la salida (el
// respaldo de un archivo corrupto avisa por slog).
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Ayudas.
// ---------------------------------------------------------------------------

// abrir monta un almacén en dir y lo cierra al terminar el test.
func abrir(t *testing.T, dir string) *DB {
	t.Helper()
	d, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%q): %v", dir, err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// progreso es un registro de ejemplo: 95 s de un episodio de 100 (visto).
func progreso(animeID, episodioID string) models.WatchProgress {
	return models.WatchProgress{
		AnimeID:       animeID,
		EpisodeID:     episodioID,
		CurrentTime:   95,
		TotalDuration: 100,
	}
}

// sinTemporales comprueba que la escritura atómica no deja temporales sueltos.
func sinTemporales(t *testing.T, dir string) {
	t.Helper()
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	for _, e := range entradas {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("quedó un temporal huérfano: %s", e.Name())
		}
	}
}

// ---------------------------------------------------------------------------
// Escritura y lectura.
// ---------------------------------------------------------------------------

func TestGuardarYLeer(t *testing.T) {
	d := abrir(t, t.TempDir())

	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:2")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	if err := d.SaveProgress(progreso("naruto", "jkanime:naruto:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	bleach, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(bleach) != 2 {
		t.Fatalf("bleach = %+v; quería sus 2 episodios", bleach)
	}
	// El orden es el de la PRIMERA guardada de cada episodio.
	if bleach[0].EpisodeID != "jkanime:bleach:episode:1" || bleach[1].EpisodeID != "jkanime:bleach:episode:2" {
		t.Errorf("orden = %+v; quería el de primera guardada", bleach)
	}
	if bleach[0].AnimeID != "bleach" || bleach[0].CurrentTime != 95 || bleach[0].TotalDuration != 100 {
		t.Errorf("registro = %+v; se perdió algún campo", bleach[0])
	}
	// 95/100 >= 0.90 → el backend lo da por visto.
	if !bleach[0].IsWatched {
		t.Error("IsWatched debe salir calculado por el backend")
	}

	if naruto, err := d.GetProgress("naruto"); err != nil || len(naruto) != 1 {
		t.Errorf("naruto = %+v (err=%v); quería su registro", naruto, err)
	}
	// animeID vacío = historial entero.
	todos, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress(\"\"): %v", err)
	}
	if len(todos) != 3 {
		t.Errorf("todos = %+v; quería los 3 registros", todos)
	}
	// Un anime sin registros no es error: lista vacía.
	vacio, err := d.GetProgress("one-piece")
	if err != nil {
		t.Fatalf("GetProgress sin registros: %v", err)
	}
	if vacio == nil || len(vacio) != 0 {
		t.Errorf("vacío = %#v; quería una lista vacía (no nil)", vacio)
	}
}

func TestGuardarActualizaSinDuplicarNiMover(t *testing.T) {
	d := abrir(t, t.TempDir())

	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("1ª SaveProgress: %v", err)
	}
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:2")); err != nil {
		t.Fatalf("2ª SaveProgress: %v", err)
	}

	// El reproductor vuelve a guardar el primer episodio más adelantado.
	segunda := progreso("bleach", "jkanime:bleach:episode:1")
	segunda.CurrentTime = 40
	segunda.TotalDuration = 100
	if err := d.SaveProgress(segunda); err != nil {
		t.Fatalf("3ª SaveProgress: %v", err)
	}

	bleach, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(bleach) != 2 {
		t.Fatalf("bleach = %+v; una guardada repetida NO debe duplicar", bleach)
	}
	if bleach[0].EpisodeID != "jkanime:bleach:episode:1" || bleach[0].CurrentTime != 40 {
		t.Errorf("primer registro = %+v; debía actualizarse en su sitio", bleach[0])
	}
	if bleach[0].IsWatched {
		t.Error("con 40/100 el episodio ya no está visto")
	}
	if bleach[1].EpisodeID != "jkanime:bleach:episode:2" {
		t.Errorf("segundo registro = %+v; no debía moverse", bleach[1])
	}
}

func TestGetProgressDevuelveCopia(t *testing.T) {
	d := abrir(t, t.TempDir())
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	primera, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	primera[0].CurrentTime = 7
	primera[0].AnimeID = "otro"
	primera = append(primera, models.WatchProgress{AnimeID: "falso"})

	segunda, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("2º GetProgress: %v", err)
	}
	if len(segunda) != 1 || segunda[0].CurrentTime != 95 || segunda[0].AnimeID != "bleach" {
		t.Errorf("segunda = %+v; mutar lo devuelto no debe tocar el almacén", segunda)
	}
}

func TestGuardarSinIdentificadoresFalla(t *testing.T) {
	d := abrir(t, t.TempDir())

	sinAnime := progreso("", "jkanime:bleach:episode:1")
	if err := d.SaveProgress(sinAnime); err == nil {
		t.Error("sin anime_id debe fallar")
	}
	sinEpisodio := progreso("bleach", "")
	if err := d.SaveProgress(sinEpisodio); err == nil {
		t.Error("sin episode_id debe fallar")
	}
	if registros, _ := d.GetProgress(""); len(registros) != 0 {
		t.Errorf("registros = %+v; no debe guardarse nada inválido", registros)
	}
}

// ---------------------------------------------------------------------------
// IsWatched: umbral y duraciones no positivas.
// ---------------------------------------------------------------------------

func TestIsWatchedSeCalculaEnElBackend(t *testing.T) {
	casos := []struct {
		nombre        string
		currentTime   float64
		totalDuration float64
		traeIsWatched bool // lo que "trae" el llamador: se ignora
		esperado      bool
	}{
		{"justo en el umbral se da por visto", 9, 10, false, true},
		{"por debajo del umbral no", 8.999, 10, true, false},
		{"episodio completado", 100, 100, false, true},
		{"sin empezar", 0, 10, true, false},
		{"TotalDuration cero nunca se da por visto", 100, 0, true, false},
		{"TotalDuration negativa nunca se da por visto", 100, -10, true, false},
		{"CurrentTime negativo no cuenta", -5, 10, true, false},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d := abrir(t, t.TempDir())
			p := models.WatchProgress{
				AnimeID:       "bleach",
				EpisodeID:     "jkanime:bleach:episode:1",
				CurrentTime:   c.currentTime,
				TotalDuration: c.totalDuration,
				IsWatched:     c.traeIsWatched,
			}
			if err := d.SaveProgress(p); err != nil {
				t.Fatalf("SaveProgress: %v", err)
			}
			registros, err := d.GetProgress("bleach")
			if err != nil {
				t.Fatalf("GetProgress: %v", err)
			}
			if len(registros) != 1 {
				t.Fatalf("registros = %+v", registros)
			}
			got := registros[0]
			if got.IsWatched != c.esperado {
				t.Errorf("IsWatched = %v (CurrentTime=%v, TotalDuration=%v); esperaba %v: lo calcula el backend, no el llamador",
					got.IsWatched, got.CurrentTime, got.TotalDuration, c.esperado)
			}
		})
	}
}

func TestEsVistoUsaElUmbralConfigurable(t *testing.T) {
	if !esVisto(UmbralIsWatched, 1) {
		t.Error("exactamente en el umbral debe contar como visto (>=)")
	}
	if esVisto(math.Nextafter(UmbralIsWatched, 0), 1) {
		t.Error("un pelín por debajo del umbral no debe contar")
	}
	if esVisto(0, 0) || esVisto(1, -1) || esVisto(100, 0) {
		t.Error("sin duración positiva nunca se da por visto")
	}
}

func TestFlotantesNoFinitosNoRompenElJSON(t *testing.T) {
	d := abrir(t, t.TempDir())

	p := progreso("bleach", "jkanime:bleach:episode:1")
	p.CurrentTime = math.NaN()
	p.TotalDuration = math.Inf(1)
	if err := d.SaveProgress(p); err != nil {
		t.Fatalf("SaveProgress con NaN/Inf: %v", err)
	}

	registros, err := d.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("registros = %+v", registros)
	}
	if registros[0].CurrentTime != 0 || registros[0].TotalDuration != 0 {
		t.Errorf("flotantes = %v/%v; NaN e Inf deben saneare a 0",
			registros[0].CurrentTime, registros[0].TotalDuration)
	}
	if registros[0].IsWatched {
		t.Error("con la duración saneada a 0 no puede estar visto")
	}
}

// ---------------------------------------------------------------------------
// Archivo corrupto y persistencia.
// ---------------------------------------------------------------------------

func TestArchivoCorruptoSeRespaldadoYEmpiezaVacio(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, nombreArchivo)
	basura := []byte("{esto no es un array de JSON, ")
	if err := os.WriteFile(ruta, basura, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	d, err := Open(dir)
	if err != nil {
		t.Fatalf("un historial corrupto NO debe impedir abrir: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	registros, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(registros) != 0 {
		t.Errorf("registros = %+v; un corrupto empieza vacío", registros)
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
	// Y el archivo ya no está ahí corrupto: al guardar se crea uno nuevo.
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress tras el respaldo: %v", err)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("ReadFile del historial nuevo: %v", err)
	}
	var nuevos []models.WatchProgress
	if err := json.Unmarshal(datos, &nuevos); err != nil {
		t.Fatalf("el historial nuevo no es JSON válido: %v (%s)", err, datos)
	}
	if len(nuevos) != 1 {
		t.Errorf("nuevos = %+v; quería el registro recién guardado", nuevos)
	}
	sinTemporales(t, dir)

	// Reabrir el ya reparado sigue funcionando sin crear otro .bak.
	segundo, err := Open(dir)
	if err != nil {
		t.Fatalf("2ª Open: %v", err)
	}
	defer segundo.Close()
	if registros, err := segundo.GetProgress(""); err != nil || len(registros) != 1 {
		t.Errorf("registros = %+v (err=%v)", registros, err)
	}
}

func TestArchivoVacioYEInexistenteEmpiezanVacios(t *testing.T) {
	t.Run("no existe", func(t *testing.T) {
		dir := t.TempDir()
		d := abrir(t, dir)
		if registros, err := d.GetProgress(""); err != nil || len(registros) != 0 {
			t.Errorf("registros = %+v (err=%v)", registros, err)
		}
		// Nada se escribe hasta que alguien guarda.
		entradas, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}
		if len(entradas) != 0 {
			t.Errorf("entradas = %v; Open no debe crear nada", entradas)
		}
	})

	t.Run("vacío", func(t *testing.T) {
		dir := t.TempDir()
		ruta := filepath.Join(dir, nombreArchivo)
		if err := os.WriteFile(ruta, nil, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		d := abrir(t, dir)
		if registros, err := d.GetProgress(""); err != nil || len(registros) != 0 {
			t.Errorf("registros = %+v (err=%v)", registros, err)
		}
		if _, err := os.Stat(ruta + sufijoRespaldo); !os.IsNotExist(err) {
			t.Errorf("un archivo vacío no tiene nada que respaldar (.bak err=%v)", err)
		}
	})
}

func TestPersistenciaTrasReabrir(t *testing.T) {
	dir := t.TempDir()

	primera, err := Open(dir)
	if err != nil {
		t.Fatalf("1ª Open: %v", err)
	}
	if err := primera.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	parcial := progreso("bleach", "jkanime:bleach:episode:2")
	parcial.CurrentTime = 40
	if err := primera.SaveProgress(parcial); err != nil {
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

	bleach, err := segunda.GetProgress("bleach")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(bleach) != 2 {
		t.Fatalf("bleach = %+v; quería los 2 registros que quedaron en disco", bleach)
	}
	if bleach[0].EpisodeID != "jkanime:bleach:episode:1" || !bleach[0].IsWatched {
		t.Errorf("primer registro = %+v; debía sobrevivir con su is_watched", bleach[0])
	}
	if bleach[1].CurrentTime != 40 || bleach[1].IsWatched {
		t.Errorf("segundo registro = %+v; debía sobrevivir con 40/100 sin ver", bleach[1])
	}
	sinTemporales(t, dir)
}

// ---------------------------------------------------------------------------
// Concurrencia (con -race).
// ---------------------------------------------------------------------------

func TestEscriturasConcurrentes(t *testing.T) {
	dir := t.TempDir()
	d := abrir(t, dir)

	const (
		goroutines   = 8
		porGoroutine = 25
		peleadores   = 4
		peleas       = 10
		duracion     = 100.0
	)

	var wg sync.WaitGroup
	// Escritoras sobre episodios DISTINTOS + lectoras entremezcladas: es el
	// flujo real (el reproductor guarda cada ~5 s mientras la UI lee).
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < porGoroutine; i++ {
				p := models.WatchProgress{
					AnimeID:       "bleach",
					EpisodeID:     fmt.Sprintf("jkanime:bleach:episode:%d-%d", g, i),
					CurrentTime:   float64(i),
					TotalDuration: duracion,
				}
				if err := d.SaveProgress(p); err != nil {
					t.Errorf("SaveProgress(g=%d,i=%d): %v", g, i, err)
					return
				}
				if _, err := d.GetProgress("bleach"); err != nil {
					t.Errorf("GetProgress(g=%d,i=%d): %v", g, i, err)
					return
				}
			}
		}(g)
	}
	// Peleas por el MISMO episodio: el mutex debe serializarlas sin
	// entrelazar escrituras en el archivo.
	for g := 0; g < peleadores; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < peleas; i++ {
				p := progreso("bleach", "jkanime:bleach:episode:1")
				p.CurrentTime = float64(g*100 + i)
				if err := d.SaveProgress(p); err != nil {
					t.Errorf("SaveProgress compartido(g=%d,i=%d): %v", g, i, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	esperados := goroutines*porGoroutine + 1 // + el episodio compartido
	todos, err := d.GetProgress("")
	if err != nil {
		t.Fatalf("GetProgress: %v", err)
	}
	if len(todos) != esperados {
		t.Fatalf("registros = %d; esperaba %d (se han perdido guardadas)", len(todos), esperados)
	}
	vistos := make(map[string]bool, len(todos))
	for _, r := range todos {
		if r.AnimeID != "bleach" || r.TotalDuration != duracion {
			t.Errorf("registro corrupto: %+v", r)
		}
		if vistos[r.EpisodeID] {
			t.Errorf("episodio duplicado: %s", r.EpisodeID)
		}
		vistos[r.EpisodeID] = true
	}
	// El archivo debe estar íntegro en disco: se reabre y vuelve a parsear.
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
	if len(enDisco) != esperados {
		t.Errorf("en disco = %d registros; esperaba %d", len(enDisco), esperados)
	}
	sinTemporales(t, dir)
}

// ---------------------------------------------------------------------------
// Ciclo de vida.
// ---------------------------------------------------------------------------

func TestCloseEsIdempotenteYLasOperacionesFallanDespues(t *testing.T) {
	dir := t.TempDir()
	d := abrir(t, dir)
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:1")); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close debe ser idempotente: %v", err)
	}
	if err := d.SaveProgress(progreso("bleach", "jkanime:bleach:episode:2")); !errors.Is(err, ErrCerrado) {
		t.Errorf("SaveProgress tras Close = %v; quería ErrCerrado", err)
	}
	if _, err := d.GetProgress("bleach"); !errors.Is(err, ErrCerrado) {
		t.Errorf("GetProgress tras Close = %v; quería ErrCerrado", err)
	}

	// Lo último guardado siguió en disco (Close no toca el archivo).
	reabierta, err := Open(dir)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	defer reabierta.Close()
	bleach, err := reabierta.GetProgress("bleach")
	if err != nil || len(bleach) != 1 {
		t.Errorf("bleach = %+v (err=%v); quería el registro previo a Close", bleach, err)
	}
}

func TestRutaPorDefectoEsConfigDelUsuarioMasMiapp(t *testing.T) {
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("os.UserConfigDir no disponible: %v", err)
	}
	donde, err := rutaPorDefecto()
	if err != nil {
		t.Fatalf("rutaPorDefecto: %v", err)
	}
	if esperado := filepath.Join(base, "miapp"); donde != esperado {
		t.Errorf("rutaPorDefecto = %q; esperaba %q", donde, esperado)
	}
}
