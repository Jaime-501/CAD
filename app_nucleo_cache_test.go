package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"miapp/internal/cache"
	"miapp/internal/models"
)

// relojFalso avanza el tiempo sin time.Sleep (para probar la expiración de
// la caché desde el núcleo).
type relojFalso struct {
	mu sync.Mutex
	t  time.Time
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

// TestDetalleSegundaLlamadaAlMismoIDNoVuelveAlProveedor comprueba el hit
// completo: la 2ª llamada devuelve lo cacheado —detalle + todos los
// episodios— sin tocar al proveedor y emite episodes:complete de inmediato.
func TestDetalleSegundaLlamadaAlMismoIDNoVuelveAlProveedor(t *testing.T) {
	detalle := models.AnimeDetail{ID: "bleach", Title: "Bleach", Provider: "jkanime"}
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
				Episodes: []models.Episode{{ID: "jkanime:bleach:episode:17", Number: 17}}})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return detalle, []models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}}, nil
		},
	}
	e := nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), fake)
	ctx := context.Background()

	primera, err := e.n.detalle(ctx, "jkanime", "bleach")
	if err != nil {
		t.Fatalf("1ª llamada: %v", err)
	}
	if primera.Detail.ID != "bleach" || len(primera.Episodes) != 1 {
		t.Fatalf("1ª llamada = %+v; quería el detalle y la página 1", primera)
	}

	segunda, err := e.n.detalle(ctx, "jkanime", "bleach")
	if err != nil {
		t.Fatalf("2ª llamada: %v", err)
	}
	if ops := fake.operaciones(); len(ops) != 1 || ops[0] != "StreamAnimeDetails" {
		t.Errorf("ops = %v; la 2ª llamada no debe volver al proveedor", ops)
	}
	if segunda.Detail.ID != "bleach" {
		t.Errorf("detalle = %+v", segunda.Detail)
	}
	if len(segunda.Episodes) != 2 || segunda.Episodes[0].Number != 1 || segunda.Episodes[1].Number != 17 {
		t.Errorf("episodios = %+v; debía venir TODO lo acumulado, ordenado", segunda.Episodes)
	}
	// El hit cierra de inmediato: el del stream + el del hit.
	completados := completadosDe(t, e.emis.lista(), "bleach")
	if len(completados) != 2 || completados[1].Error != "" {
		t.Errorf("completados = %+v; el hit debe emitir episodes:complete sin error", completados)
	}
}

// TestDetalleHitAPartidoSeAdjuntaAlStreamEnCurso es la regla central de T21:
// a mitad de paginación la 2ª llamada se adjunta al stream vivo —sin
// cancelarlo ni lanzar otro— y sigue recibiendo sus lotes.
func TestDetalleHitAPartidoSeAdjuntaAlStreamEnCurso(t *testing.T) {
	liberar := make(chan struct{})
	terminado := make(chan struct{})

	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			go func() {
				defer close(terminado)
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:17", Number: 17}}})
				<-liberar
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:18", Number: 18}}})
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
			}()
			return models.AnimeDetail{ID: id, Title: "Bleach"},
				[]models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}}, nil
		},
	}
	e := nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), fake)
	ctx := context.Background()

	if _, err := e.n.detalle(ctx, "jkanime", "bleach"); err != nil {
		t.Fatalf("1ª llamada: %v", err)
	}
	e.emis.esperar(t, 1) // el primer lote ya está en la caché y en el emisor

	parcial, err := e.n.detalle(ctx, "jkanime", "bleach")
	if err != nil {
		t.Fatalf("2ª llamada: %v", err)
	}
	if ops := fake.operaciones(); len(ops) != 1 {
		t.Errorf("ops = %v; no debe lanzarse un segundo stream para el mismo anime", ops)
	}
	if len(parcial.Episodes) != 2 {
		t.Errorf("episodios = %+v; debía venir lo acumulado (página 1 + lote 2)", parcial.Episodes)
	}
	if c := completadosDe(t, e.emis.lista(), "bleach"); len(c) != 0 {
		t.Errorf("un stream en curso no debe cerrar: %+v", c)
	}

	close(liberar)
	select {
	case <-terminado:
	case <-time.After(2 * time.Second):
		t.Fatal("el stream no terminó tras liberarlo")
	}

	// Los lotes siguientes salen del MISMO stream (una sola llamada al provider).
	eventos := e.emis.lista()
	if lotes := lotesDe(t, eventos, "bleach"); len(lotes) != 3 {
		t.Errorf("lotes = %+v; debían ser los 3 del único stream", lotes)
	}
	if c := completadosDe(t, eventos, "bleach"); len(c) != 1 || c[0].Error != "" {
		t.Errorf("completados = %+v; el stream debe cerrar una vez y sin error", c)
	}
	ent, ok := e.cache.Get(cache.Clave("jkanime", "bleach"))
	if !ok || !ent.Complete || len(ent.Episodes) != 3 {
		t.Errorf("entrada = %+v (ok=%v); debía quedar completa con los 3 episodios", ent, ok)
	}
}

// TestCambioDeAnimeBorraLaEntradaIncompleta: al cambiar a un anime distinto
// se cancela el stream anterior y su entrada incompleta NO sobrevive.
func TestCambioDeAnimeBorraLaEntradaIncompleta(t *testing.T) {
	liberar := make(chan struct{})
	terminado := make(chan struct{})

	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			if id == "anime-a" {
				go func() {
					defer close(terminado)
					onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
						Episodes: []models.Episode{{ID: "jkanime:anime-a:episode:2", Number: 2}}})
					<-liberar
					onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
				}()
			}
			return models.AnimeDetail{ID: id},
				[]models.Episode{{ID: "jkanime:" + id + ":episode:1", Number: 1}}, nil
		},
	}
	e := nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), fake)
	ctx := context.Background()
	claveA := cache.Clave("jkanime", "anime-a")
	claveB := cache.Clave("jkanime", "anime-b")

	if _, err := e.n.detalle(ctx, "jkanime", "anime-a"); err != nil {
		t.Fatalf("detalle anime-a: %v", err)
	}
	e.emis.esperar(t, 1)
	if _, ok := e.cache.Get(claveA); !ok {
		t.Fatal("la entrada de anime-a debía existir mientras streama")
	}

	if _, err := e.n.detalle(ctx, "jkanime", "anime-b"); err != nil {
		t.Fatalf("detalle anime-b: %v", err)
	}
	if _, ok := e.cache.Get(claveA); ok {
		t.Error("la entrada incompleta de anime-a debe eliminarse al cambiar de anime")
	}
	if _, ok := e.cache.Get(claveB); !ok {
		t.Error("la entrada de anime-b debe quedar")
	}

	close(liberar)
	select {
	case <-terminado:
	case <-time.After(2 * time.Second):
		t.Fatal("el stream de anime-a no terminó tras cancelarse")
	}
	// Su lote final llega con el ctx cancelado: no se cachea ni se emite.
	if _, ok := e.cache.Get(claveA); ok {
		t.Error("el lote tardío de un stream cancelado no debe recrear la entrada")
	}
	if c := completadosDe(t, e.emis.lista(), "anime-a"); len(c) != 0 {
		t.Errorf("un stream cancelado no emite episodes:complete: %+v", c)
	}
}

// TestErroresNoSeCachean cubre los dos caminos de error: lote final con Err
// y un StreamAnimeDetails que devuelve error tras haber emitido lotes.
func TestErroresNoSeCachean(t *testing.T) {
	clave := cache.Clave("jkanime", "bleach")

	t.Run("lote final con Err", func(t *testing.T) {
		fake := &streamerFalso{
			proveedorFalso: proveedorFalso{nombre: "jkanime"},
			streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:17", Number: 17}}})
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true, Err: "la página 3 falló"})
				return models.AnimeDetail{ID: id},
					[]models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}}, nil
			},
		}
		e := nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), fake)

		if _, err := e.n.detalle(context.Background(), "jkanime", "bleach"); err != nil {
			t.Fatalf("el fallo de una página no tumba la llamada: %v", err)
		}
		if _, ok := e.cache.Get(clave); ok {
			t.Error("un stream terminado con Err no se cachea")
		}
		c := completadosDe(t, e.emis.lista(), "bleach")
		if len(c) != 1 || c[0].Error != "la página 3 falló" {
			t.Errorf("completados = %+v; el error debe llegar al frontend como texto", c)
		}
		// Sin entrada cacheada, la 2ª llamada vuelve al proveedor.
		if _, err := e.n.detalle(context.Background(), "jkanime", "bleach"); err != nil {
			t.Fatalf("2ª llamada: %v", err)
		}
		if ops := fake.operaciones(); len(ops) != 2 {
			t.Errorf("ops = %v; sin caché debe volver a pedirlo", ops)
		}
	})

	t.Run("StreamAnimeDetails devuelve error", func(t *testing.T) {
		fake := &streamerFalso{
			proveedorFalso: proveedorFalso{nombre: "jkanime"},
			streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
				// Emite un lote (que crea la entrada) y luego falla.
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:17", Number: 17}}})
				return models.AnimeDetail{}, nil, errors.New("jkanime caído")
			},
		}
		e := nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), fake)

		if _, err := e.n.detalle(context.Background(), "jkanime", "bleach"); err == nil {
			t.Fatal("con el proveedor caído debe devolver error")
		}
		if _, ok := e.cache.Get(clave); ok {
			t.Error("un stream con error no deja entrada en la caché")
		}
	})
}

// TestDetalleExpiradoVuelveAlProveedor: la entrada caducada (TTL) no sirve
// como hit y el proveedor se vuelve a llamar.
func TestDetalleExpiradoVuelveAlProveedor(t *testing.T) {
	reloj := &relojFalso{t: time.Now()}
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return models.AnimeDetail{ID: id}, []models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}}, nil
		},
	}
	c := cache.New(ttlCacheDetalle, cache.ConReloj(reloj.ahora))
	e := nuevoEntornoConCache(t, c, fake)
	ctx := context.Background()

	if _, err := e.n.detalle(ctx, "jkanime", "bleach"); err != nil {
		t.Fatalf("1ª llamada: %v", err)
	}
	if _, err := e.n.detalle(ctx, "jkanime", "bleach"); err != nil {
		t.Fatalf("2ª llamada: %v", err)
	}
	if ops := fake.operaciones(); len(ops) != 1 {
		t.Fatalf("ops = %v; antes de caducar debe ser hit", ops)
	}

	reloj.avanzar(ttlCacheDetalle + time.Second)

	if _, err := e.n.detalle(ctx, "jkanime", "bleach"); err != nil {
		t.Fatalf("3ª llamada: %v", err)
	}
	if ops := fake.operaciones(); len(ops) != 2 {
		t.Errorf("ops = %v; caducada la entrada debe volver al proveedor", ops)
	}
}

// TestCerrarBorraLaEntradaIncompletaDelStream: al parar la app se cancela el
// stream activo y su entrada incompleta no queda esperando a nadie.
func TestCerrarBorraLaEntradaIncompletaDelStream(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "jkanime"},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			go func() {
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:17", Number: 17}}})
				<-ctx.Done()
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
			}()
			return models.AnimeDetail{ID: id},
				[]models.Episode{{ID: "jkanime:bleach:episode:1", Number: 1}}, nil
		},
	}
	e := nuevoEntornoConCache(t, cache.New(ttlCacheDetalle), fake)
	clave := cache.Clave("jkanime", "bleach")

	if err := e.n.arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	if _, err := e.n.detalle(context.Background(), "jkanime", "bleach"); err != nil {
		t.Fatalf("detalle: %v", err)
	}
	e.emis.esperar(t, 1)
	if _, ok := e.cache.Get(clave); !ok {
		t.Fatal("la entrada debía existir mientras streama")
	}

	if err := e.n.cerrar(); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
	if _, ok := e.cache.Get(clave); ok {
		t.Error("shutdown debe eliminar la entrada incompleta del stream cancelado")
	}
}
