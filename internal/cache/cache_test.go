package cache

import (
	"sync"
	"testing"
	"time"

	"miapp/internal/models"
)

// ---------------------------------------------------------------------------
// Reloj falso: los tests avanzan el tiempo sin time.Sleep.
// ---------------------------------------------------------------------------

type relojFalso struct {
	mu sync.Mutex
	t  time.Time
}

func nuevoReloj() *relojFalso {
	return &relojFalso{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (r *relojFalso) ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.t
}

func (r *relojFalso) avanzar(d time.Duration) {
	r.mu.Lock()
	r.t = r.t.Add(d)
	r.mu.Unlock()
}

func hayEntradas(c *Cache) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entradas)
}

func esperarHasta(t *testing.T, que string, cond func() bool) {
	t.Helper()
	limite := time.Now().Add(2 * time.Second)
	for time.Now().Before(limite) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timeout esperando: %s", que)
}

// ---------------------------------------------------------------------------
// Copias, deduplicación y semántica de los métodos.
// ---------------------------------------------------------------------------

func TestGetDevuelveCopias(t *testing.T) {
	c := New(time.Minute)
	clave := Clave("jkanime", "bleach")

	detalle := models.AnimeDetail{ID: "bleach", Title: "Bleach", Genres: []string{"accion", "aventura"}}
	c.Set(clave, detalle)
	eps := []models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1, Title: "El inicio"}}
	c.Append(clave, eps)

	// Ni lo que devuelvo ni lo que entró por parámetro deben compartir
	// memoria con la caché.
	detalle.Genres[0] = "mutado-sin-cache"
	eps[0].Title = "mutado-sin-cache"

	e, ok := c.Get(clave)
	if !ok {
		t.Fatal("la entrada debía existir")
	}
	if len(e.Episodes) != 1 || e.Episodes[0].Title != "El inicio" {
		t.Errorf("episodios = %+v; el título debía ser el original", e.Episodes)
	}
	if len(e.Detail.Genres) != 2 || e.Detail.Genres[0] != "accion" {
		t.Errorf("géneros = %v; debían conservarse", e.Detail.Genres)
	}

	// Mutar lo devuelto por Get tampoco debe tocar la caché.
	e.Episodes[0].Title = "mutado-en-get"
	e.Episodes = append(e.Episodes, models.Episode{ID: "otro", Number: 2})
	e.Detail.Genres[1] = "mutado-en-get"

	deNueva, ok := c.Get(clave)
	if !ok {
		t.Fatal("la entrada seguía debiendo existir")
	}
	if len(deNueva.Episodes) != 1 || deNueva.Episodes[0].Title != "El inicio" {
		t.Errorf("episodios = %+v; Get no debe compartir memoria con la caché", deNueva.Episodes)
	}
	if deNueva.Detail.Genres[1] != "aventura" {
		t.Errorf("géneros = %v; Get no debe compartir memoria con la caché", deNueva.Detail.Genres)
	}
}

func TestAppendDeduplicaPorID(t *testing.T) {
	c := New(time.Minute)
	clave := Clave("jkanime", "bleach")

	c.Append(clave, []models.Episode{
		{ID: "jkanime:bleach:episode:1", Number: 1},
		{ID: "jkanime:bleach:episode:2", Number: 2},
	})
	c.Append(clave, []models.Episode{
		{ID: "jkanime:bleach:episode:2", Number: 2},
		{ID: "jkanime:bleach:episode:3", Number: 3},
	})
	c.Append(clave, []models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}})

	e, ok := c.Get(clave)
	if !ok {
		t.Fatal("la entrada debía existir")
	}
	if len(e.Episodes) != 3 {
		t.Fatalf("episodios = %+v; la deduplicación por ID debe dejar 3", e.Episodes)
	}
	for i, ep := range e.Episodes {
		if ep.Number != i+1 {
			t.Errorf("episodio %d = %+v", i, ep)
		}
	}

	// Una entrada sin ID no hay con qué deduplicarla: se añade siempre.
	c.Append(clave, []models.Episode{{Number: 99}})
	c.Append(clave, []models.Episode{{Number: 99}})
	sinID, _ := c.Get(clave)
	if len(sinID.Episodes) != 5 {
		t.Errorf("episodios = %+v; los que no traen ID se añaden siempre", sinID.Episodes)
	}

	// Append de lista vacía no hace nada (ni crea entradas).
	c.Append(Clave("jkanime", "naruto"), nil)
	if hayEntradas(c) != 1 {
		t.Errorf("entradas = %d; Append vacío no debe crear ninguna", hayEntradas(c))
	}
}

func TestSetNoPisaEpisodiosNiComplete(t *testing.T) {
	c := New(time.Minute)
	clave := Clave("jkanime", "bleach")

	c.Append(clave, []models.Episode{{ID: "jkanime:bleach:episode:17", Number: 17}})
	c.MarkComplete(clave)
	// El detalle puede llegar DESPUÉS de los lotes: Set solo lo guarda.
	c.Set(clave, models.AnimeDetail{ID: "bleach", Title: "Bleach"})

	e, ok := c.Get(clave)
	if !ok {
		t.Fatal("la entrada debía existir")
	}
	if !e.Complete {
		t.Error("Set no debe tocar Complete")
	}
	if len(e.Episodes) != 1 || e.Episodes[0].Number != 17 {
		t.Errorf("episodios = %+v; Set no debe tocar los episodios", e.Episodes)
	}
	if e.Detail.Title != "Bleach" {
		t.Errorf("detalle = %+v", e.Detail)
	}
}

func TestMarkCompleteCreaLaEntradaYDeleteLaBorra(t *testing.T) {
	c := New(time.Minute)
	clave := Clave("jkanime", "bleach")

	if e, ok := c.Get(clave); ok {
		t.Fatalf("no debía haber nada: %+v", e)
	}
	c.MarkComplete(clave)
	e, ok := c.Get(clave)
	if !ok || !e.Complete {
		t.Fatalf("MarkComplete debe crear la entrada ya completa: %+v (ok=%v)", e, ok)
	}

	c.Delete(clave)
	if _, ok := c.Get(clave); ok {
		t.Error("Delete debe eliminar la entrada")
	}
	// Borrar algo que no existe no revienta.
	c.Delete(clave)
}

func TestClaveSeparaProveedores(t *testing.T) {
	if Clave("jkanime", "bleach") != "jkanime:bleach" {
		t.Errorf("clave = %q", Clave("jkanime", "bleach"))
	}
	if Clave("jkanime", "bleach") == Clave("animeflv", "bleach") {
		t.Error("dos proveedores no deben compartir entrada")
	}
}

// ---------------------------------------------------------------------------
// Caducidad (reloj inyectado, sin time.Sleep) y limpieza.
// ---------------------------------------------------------------------------

func TestExpiracionPorTTLBorraEnLectura(t *testing.T) {
	reloj := nuevoReloj()
	c := New(time.Minute, ConReloj(reloj.ahora))
	clave := Clave("jkanime", "bleach")
	c.Set(clave, models.AnimeDetail{ID: "bleach"})
	c.Append(clave, []models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}})

	reloj.avanzar(59 * time.Second)
	if _, ok := c.Get(clave); !ok {
		t.Fatal("aún no debía caducar")
	}
	// Cada ESCRITURA renueva el TTL (por deslizamiento); Get no lo renueva.
	c.Append(clave, []models.Episode{{ID: "jkanime:bleach:episode:2", Number: 2}})
	reloj.avanzar(30 * time.Second)
	if _, ok := c.Get(clave); !ok {
		t.Fatal("Append renovó el TTL: aún no debía caducar")
	}

	reloj.avanzar(61 * time.Second)
	if _, ok := c.Get(clave); ok {
		t.Fatal("con el TTL vencido el Get debe fallar")
	}
	if n := hayEntradas(c); n != 0 {
		t.Errorf("entradas = %d; Get debe limpiar la caducada (limpieza en lectura)", n)
	}
}

func TestPodarEliminaSoloLasCaducadas(t *testing.T) {
	reloj := nuevoReloj()
	c := New(time.Minute, ConReloj(reloj.ahora))
	vieja := Clave("jkanime", "bleach")
	nueva := Clave("jkanime", "naruto")

	c.Set(vieja, models.AnimeDetail{ID: "bleach"})
	reloj.avanzar(40 * time.Second)
	c.Set(nueva, models.AnimeDetail{ID: "naruto"})
	reloj.avanzar(30 * time.Second) // vieja: 70 s (vencida), nueva: 30 s

	if quitas := c.podar(); quitas != 1 {
		t.Errorf("podar quitó %d; solo debía caducar la vieja", quitas)
	}
	if _, ok := c.Get(vieja); ok {
		t.Error("la entrada vieja debía estar podada")
	}
	if _, ok := c.Get(nueva); !ok {
		t.Error("la entrada nueva debe seguir")
	}
}

func TestTTLCeroNoCaduca(t *testing.T) {
	reloj := nuevoReloj()
	c := New(0, ConReloj(reloj.ahora))
	clave := Clave("jkanime", "bleach")
	c.Set(clave, models.AnimeDetail{ID: "bleach"})

	reloj.avanzar(1000 * time.Hour)
	if _, ok := c.Get(clave); !ok {
		t.Fatal("sin TTL la entrada no caduca")
	}
	if quitas := c.podar(); quitas != 0 {
		t.Errorf("podar quitó %d; sin TTL no hay nada que podar", quitas)
	}

	// Y no se arranca ninguna rutina de limpieza.
	c.Arrancar()
	c.Cerrar()
}

func TestLimpiezaPeriodicaSeDetieneEnCerrar(t *testing.T) {
	reloj := nuevoReloj()
	c := New(30*time.Second, ConReloj(reloj.ahora), ConIntervalo(5*time.Millisecond))
	clave := Clave("jkanime", "bleach")

	c.Arrancar()
	c.Set(clave, models.AnimeDetail{ID: "bleach"})
	reloj.avanzar(time.Minute)

	// La rutina poda sola...
	esperarHasta(t, "la rutina periódica poda la entrada", func() bool {
		return hayEntradas(c) == 0
	})

	c.Cerrar()

	// ...y tras Cerrar ya no poda: esta caducada se queda hasta que alguien
	// la lea (prueba de que la goroutine ha terminado, no de que no expira).
	otra := Clave("jkanime", "naruto")
	c.Set(otra, models.AnimeDetail{ID: "naruto"})
	reloj.avanzar(time.Minute)
	time.Sleep(50 * time.Millisecond)

	if n := hayEntradas(c); n != 1 {
		t.Errorf("entradas = %d; tras Cerrar la limpieza debe estar detenida", n)
	}
	// Idempotente: llamar otra vez no revienta.
	c.Cerrar()
}
