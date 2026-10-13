package app

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"miapp/internal/cache"
	"miapp/internal/models"
)

// reducirEspera acorta la cadencia y el tope de la espera de
// episodiosCompletos y los restaura al terminar el test (los defaults de
// producción son 50 ms / 60 s; aquí nada de time.Sleep largos).
func reducirEspera(t *testing.T, intervalo, maximo time.Duration) {
	t.Helper()
	anteriorIntervalo, anteriorMaximo := intervaloEsperaEpisodios, esperaMaximaEpisodios
	intervaloEsperaEpisodios, esperaMaximaEpisodios = intervalo, maximo
	t.Cleanup(func() {
		intervaloEsperaEpisodios, esperaMaximaEpisodios = anteriorIntervalo, anteriorMaximo
	})
}

// episodiosDe construye episodios numerados 1..n de kind episode.
func episodiosDe(n int) []models.Episode {
	eps := make([]models.Episode, 0, n)
	for i := 1; i <= n; i++ {
		eps = append(eps, models.Episode{
			ID:     proveedorPrincipal + ":bleach:episode:" + strconv.Itoa(i),
			Number: i,
			Kind:   models.KindEpisode,
		})
	}
	return eps
}

// ---------------------------------------------------------------------------
// 1-6: siguienteDe.
// ---------------------------------------------------------------------------

func TestSiguienteDeMenorNumeroMayor(t *testing.T) {
	// El siguiente de 5 NO es el mayor, sino el menor de los mayores.
	eps := []models.Episode{
		{Number: 1, Kind: models.KindEpisode},
		{Number: 5, Kind: models.KindEpisode},
		{Number: 9, Kind: models.KindEpisode},
		{Number: 7, Kind: models.KindEpisode},
	}
	sig, ok := siguienteDe(eps, 5)
	if !ok || sig.Number != 7 {
		t.Errorf("siguiente de 5 = (%d, %v); se esperaba (7, true)", sig.Number, ok)
	}
}

func TestSiguienteDeToleraHuecos(t *testing.T) {
	// 1, 2, 4: no hay 3, pero el siguiente de 2 sigue siendo el 4.
	eps := []models.Episode{
		{Number: 1, Kind: models.KindEpisode},
		{Number: 2, Kind: models.KindEpisode},
		{Number: 4, Kind: models.KindEpisode},
	}
	sig, ok := siguienteDe(eps, 2)
	if !ok || sig.Number != 4 {
		t.Errorf("siguiente de 2 = (%d, %v); se esperaba (4, true) aunque falte el 3", sig.Number, ok)
	}
}

func TestSiguienteDeListaDesordenada(t *testing.T) {
	eps := []models.Episode{
		{Number: 8, Kind: models.KindEpisode},
		{Number: 3, Kind: models.KindEpisode},
		{Number: 5, Kind: models.KindEpisode},
		{Number: 1, Kind: models.KindEpisode},
	}
	sig, ok := siguienteDe(eps, 2)
	if !ok || sig.Number != 3 {
		t.Errorf("siguiente de 2 = (%d, %v); el orden de entrada no debe influir: se esperaba (3, true)", sig.Number, ok)
	}
}

func TestSiguienteDeUltimoNoTieneSiguiente(t *testing.T) {
	sig, ok := siguienteDe(episodiosDe(12), 12)
	if ok {
		t.Errorf("del último episodio no hay siguiente, pero devolvió el %d", sig.Number)
	}
}

func TestSiguienteDeListaVacia(t *testing.T) {
	if _, ok := siguienteDe(nil, 1); ok {
		t.Error("con la lista vacía no hay siguiente episodio")
	}
}

func TestSiguienteDeIgnoraPeliculas(t *testing.T) {
	// Una película no tiene número episódico que perseguir: se salta y el
	// siguiente de un episodio sigue siendo el episodio siguiente.
	eps := []models.Episode{
		{ID: "jkanime:bleach:episode:1", Number: 1, Kind: models.KindEpisode},
		{ID: "jkanime:bleach:movie", Number: 0, Kind: models.KindMovie},
		{ID: "jkanime:bleach:episode:2", Number: 2, Kind: models.KindEpisode},
	}
	sig, ok := siguienteDe(eps, 1)
	if !ok || sig.ID != "jkanime:bleach:episode:2" {
		t.Errorf("siguiente de 1 = (%q, %v); se esperaba el episodio 2 (la película se ignora)", sig.ID, ok)
	}
	if _, ok := siguienteDe([]models.Episode{{Number: 0, Kind: models.KindMovie}}, -1); ok {
		t.Error("una lista solo de películas no aporta ningún episodio siguiente")
	}
}

// ---------------------------------------------------------------------------
// 7: estaEnEmision.
// ---------------------------------------------------------------------------

func TestEstaEnEmisionConEstadosReales(t *testing.T) {
	// Textos observados en los dos proveedores (puerta 0 de T46).
	casos := []struct {
		status string
		espera bool
	}{
		{"En emision", true},    // JKAnime, texto crudo
		{"En emisión", true},    // AnimeAV1
		{"  EN EMISIÓN ", true}, // mayúsculas + espacios
		{"Concluido", false},    // JKAnime
		{"Finalizado", false},   // AnimeAV1
		{"Por estrenar", false}, // JKAnime
		{"Próximamente", false}, // AnimeAV1
		{"Desconocido", false},  // AnimeAV1
		{"", false},
	}
	for _, c := range casos {
		if got := estaEnEmision(c.status); got != c.espera {
			t.Errorf("estaEnEmision(%q) = %v; se esperaba %v", c.status, got, c.espera)
		}
	}
}

// ---------------------------------------------------------------------------
// 8-12: decidirRetomar.
// ---------------------------------------------------------------------------

func TestDecidirRetomarResume(t *testing.T) {
	// Sin ver ⇒ retoma ESE episodio desde donde estaba, sin lista ni status.
	reg := models.WatchProgress{
		EpisodeID:   "jkanime:bleach:episode:3",
		CurrentTime: 123.5,
		IsWatched:   false,
	}
	info := decidirRetomar(proveedorPrincipal, "bleach", reg, models.KindEpisode, 3, nil, "")
	if info.Action != models.ResumeActionResume ||
		info.EpisodeID != reg.EpisodeID ||
		info.EpisodeNumber != 3 ||
		info.StartAt != 123.5 {
		t.Errorf("info = %+v; se esperaba resume del episodio 3 desde 123.5", info)
	}
	if info.Provider != proveedorPrincipal || info.AnimeID != "bleach" {
		t.Errorf("provider/anime = (%q, %q); se esperaba el par pedido", info.Provider, info.AnimeID)
	}
}

func TestDecidirRetomarNext(t *testing.T) {
	// Vista la 2 con lista completa ⇒ siguiente episodio desde 0.
	reg := models.WatchProgress{EpisodeID: "jkanime:bleach:episode:2", IsWatched: true}
	eps := episodiosDe(5)
	info := decidirRetomar(proveedorPrincipal, "bleach", reg, models.KindEpisode, 2, eps, "Concluido")
	if info.Action != models.ResumeActionNext ||
		info.EpisodeID != "jkanime:bleach:episode:3" ||
		info.EpisodeNumber != 3 ||
		info.StartAt != 0 {
		t.Errorf("info = %+v; se esperaba next del episodio 3 desde 0", info)
	}
}

func TestDecidirRetomarCompletedSerieFinalizada(t *testing.T) {
	// Vista la última de una serie ya no en emisión ⇒ completed (detalle).
	reg := models.WatchProgress{EpisodeID: "jkanime:bleach:episode:5", IsWatched: true}
	info := decidirRetomar(proveedorPrincipal, "bleach", reg, models.KindEpisode, 5, episodiosDe(5), "Concluido")
	if info.Action != models.ResumeActionCompleted ||
		info.EpisodeID != "" || info.EpisodeNumber != 0 || info.StartAt != 0 {
		t.Errorf("info = %+v; se esperaba completed con los campos de episodio en cero", info)
	}
}

func TestDecidirRetomarUpToDateEnEmision(t *testing.T) {
	// Vista la última pero el anime sigue en emisión ⇒ up_to_date.
	reg := models.WatchProgress{EpisodeID: "jkanime:bleach:episode:5", IsWatched: true}
	info := decidirRetomar(proveedorPrincipal, "bleach", reg, models.KindEpisode, 5, episodiosDe(5), "En emision")
	if info.Action != models.ResumeActionUpToDate ||
		info.EpisodeID != "" || info.EpisodeNumber != 0 || info.StartAt != 0 {
		t.Errorf("info = %+v; se esperaba up_to_date con los campos de episodio en cero", info)
	}
}

func TestDecidirRetomarPeliculaVistaEsCompleted(t *testing.T) {
	reg := models.WatchProgress{EpisodeID: "jkanime:bleach:movie", IsWatched: true}
	info := decidirRetomar(proveedorPrincipal, "bleach", reg, models.KindMovie, 0, nil, "En emision")
	if info.Action != models.ResumeActionCompleted {
		t.Errorf("action = %q; una película vista no tiene siguiente, se esperaba completed", info.Action)
	}
}

// ---------------------------------------------------------------------------
// 13-17: Retomar.
// ---------------------------------------------------------------------------

// guardarRegistro en el historial de un entorno ya arrancado.
func guardarRegistro(t *testing.T, e *entorno, animeID, episodeID string, current float64, updatedAt int64) {
	t.Helper()
	if err := e.n.GuardarProgreso(models.WatchProgress{
		AnimeID:       animeID,
		EpisodeID:     episodeID,
		CurrentTime:   current,
		TotalDuration: 100,
		UpdatedAt:     updatedAt,
	}); err != nil {
		t.Fatalf("GuardarProgreso(%s, %s): %v", animeID, episodeID, err)
	}
}

func TestRetomarSinRegistroDevuelveErrSinHistorial(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	ctx := context.Background()

	// Sin almacén abierto (antes de arrancar) ⇒ ErrHistorialNoDisponible.
	if _, err := e.n.Retomar(ctx, proveedorPrincipal, "bleach"); !errors.Is(err, ErrHistorialNoDisponible) {
		t.Fatalf("sin arrancar: err = %v; se esperaba ErrHistorialNoDisponible", err)
	}
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	// Almacén abierto pero sin ningún registro ⇒ ErrSinHistorial.
	if _, err := e.n.Retomar(ctx, proveedorPrincipal, "bleach"); !errors.Is(err, ErrSinHistorial) {
		t.Fatalf("vacío: err = %v; se esperaba ErrSinHistorial", err)
	}
}

func TestRetomarUsaCacheSinPeticionesNuevas(t *testing.T) {
	// Con la entrada completa ya en caché, retomar NO vuelve al proveedor.
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			t.Error("el proveedor no debe consultarse con la entrada completa en caché")
			return models.AnimeDetail{}, nil, errors.New("no debe llegarse aquí")
		},
	}
	e := nuevoEntorno(t, fake)
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	guardarRegistro(t, e, "bleach", "jkanime:bleach:episode:2", 100, 1000)

	clave := cache.Clave(proveedorPrincipal, "bleach")
	e.cache.Set(clave, models.AnimeDetail{ID: "bleach", Status: "Concluido"})
	e.cache.Append(clave, episodiosDe(5))
	e.cache.MarkComplete(clave)

	info, err := e.n.Retomar(context.Background(), proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("Retomar: %v", err)
	}
	if info.Action != models.ResumeActionNext || info.EpisodeID != "jkanime:bleach:episode:3" {
		t.Errorf("info = %+v; se esperaba next del episodio 3", info)
	}
	if ops := fake.operaciones(); len(ops) != 0 {
		t.Errorf("ops = %v; el hit completo no debe tocar al proveedor", ops)
	}
}

func TestRetomarSinCacheObtieneDetalleUnaVez(t *testing.T) {
	// Una sola llamada a StreamAnimeDetails, incluso esperando los lotes.
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
				Episodes: []models.Episode{{Number: 3, Kind: models.KindEpisode},
					{Number: 4, Kind: models.KindEpisode}}})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return models.AnimeDetail{ID: id, Status: "Concluido"}, episodiosDe(2), nil
		},
	}
	e := nuevoEntorno(t, fake)
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	guardarRegistro(t, e, "bleach", "jkanime:bleach:episode:2", 100, 1000)

	info, err := e.n.Retomar(context.Background(), proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("Retomar: %v", err)
	}
	if info.Action != models.ResumeActionNext || info.EpisodeNumber != 3 {
		t.Errorf("info = %+v; se esperaba next del episodio 3", info)
	}
	if ops := fake.operaciones(); len(ops) != 1 || ops[0] != "StreamAnimeDetails" {
		t.Errorf("ops = %v; se esperaba UNA única llamada StreamAnimeDetails", ops)
	}
}

func TestRetomarTomaElRegistroMasRecienteDelAnime(t *testing.T) {
	// LastPerAnime da el más reciente: si el 2 está visto, es next aunque
	// el registro más antiguo (1) no lo estuviera.
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return models.AnimeDetail{ID: id, Status: "Concluido"}, episodiosDe(4), nil
		},
	}
	e := nuevoEntorno(t, fake)
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	guardarRegistro(t, e, "bleach", "jkanime:bleach:episode:1", 10, 1000)
	guardarRegistro(t, e, "bleach", "jkanime:bleach:episode:2", 100, 2000)

	info, err := e.n.Retomar(context.Background(), proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("Retomar: %v", err)
	}
	if info.Action != models.ResumeActionNext || info.EpisodeID != "jkanime:bleach:episode:3" {
		t.Errorf("info = %+v; el registro más reciente manda: next del 3", info)
	}
}

func TestRetomarMismoSlugOtroProveedorNoInterfiere(t *testing.T) {
	// El slug "bleach" de animeav1 no debe confundirse con el de jkanime.
	fakeAv1 := &proveedorFalso{
		nombre: proveedorComplemento,
		detalleF: func(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
			return models.AnimeDetail{ID: id, Provider: proveedorComplemento}, nil, nil
		},
	}
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal}, fakeAv1)
	if err := e.n.Arrancar(e.emis.emitir); err != nil {
		t.Fatalf("arrancar: %v", err)
	}
	guardarRegistro(t, e, "bleach", proveedorComplemento+":bleach:movie", 100, 1000)

	info, err := e.n.Retomar(context.Background(), proveedorPrincipal, "bleach")
	if !errors.Is(err, ErrSinHistorial) {
		t.Fatalf("err = %v; el registro de otro proveedor no vale: ErrSinHistorial", err)
	}
	if info.Provider != "" || info.AnimeID != "" {
		t.Errorf("info = %+v; con error no debe venir nada", info)
	}

	// El mismo slug SÍ retoma en su proveedor (película vista ⇒ completed).
	otra, err := e.n.Retomar(context.Background(), proveedorComplemento, "bleach")
	if err != nil {
		t.Fatalf("Retomar del animeav1: %v", err)
	}
	if otra.Action != models.ResumeActionCompleted {
		t.Errorf("action = %q; la película vista del animeav1 da completed", otra.Action)
	}
}

// ---------------------------------------------------------------------------
// 18-20: SiguienteEpisodio.
// ---------------------------------------------------------------------------

func TestSiguienteEpisodioEncuentraSiguiente(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
				Episodes: []models.Episode{{ID: "jkanime:bleach:episode:3", Number: 3, Kind: models.KindEpisode},
					{ID: "jkanime:bleach:episode:4", Number: 4, Kind: models.KindEpisode}}})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return models.AnimeDetail{ID: id}, episodiosDe(2), nil
		},
	}
	e := nuevoEntorno(t, fake)

	sig, err := e.n.SiguienteEpisodio(context.Background(), proveedorPrincipal, "bleach", "jkanime:bleach:episode:2")
	if err != nil {
		t.Fatalf("SiguienteEpisodio: %v", err)
	}
	if !sig.Found || sig.EpisodeID != "jkanime:bleach:episode:3" || sig.Number != 3 {
		t.Errorf("sig = %+v; se esperaba el episodio 3", sig)
	}
}

func TestSiguienteEpisodioPeliculaNoTiene(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			t.Error("una película no debe pedir episodios")
			return models.AnimeDetail{}, nil, errors.New("no debe llegarse aquí")
		},
	}
	e := nuevoEntorno(t, fake)

	sig, err := e.n.SiguienteEpisodio(context.Background(), proveedorPrincipal, "bleach", "jkanime:bleach:movie")
	if err != nil {
		t.Fatalf("SiguienteEpisodio: %v", err)
	}
	if sig.Found || sig.EpisodeID != "" || sig.Number != 0 {
		t.Errorf("sig = %+v; una película no tiene siguiente, se esperaba Found=false", sig)
	}
}

func TestSiguienteEpisodioProveedorDesconocido(t *testing.T) {
	e := nuevoEntorno(t, &proveedorFalso{nombre: proveedorPrincipal})
	ctx := context.Background()

	_, err := e.n.SiguienteEpisodio(ctx, proveedorComplemento, "bleach", proveedorComplemento+":bleach:episode:1")
	if err == nil {
		t.Fatal("con proveedor desconocido debe devolver error")
	}
	_, esperado := e.n.Detalle(ctx, proveedorComplemento, "bleach")
	if err.Error() != esperado.Error() {
		t.Errorf("err = %v; Detalle dice %v; deben ser el mismo error", err, esperado)
	}
	if !contains(err.Error(), "proveedor desconocido") {
		t.Errorf("err = %q; debe mencionar el proveedor desconocido", err)
	}
}

// ---------------------------------------------------------------------------
// 21-25: episodiosCompletos.
// ---------------------------------------------------------------------------

func TestEpisodiosCompletosEsperaALosLotes(t *testing.T) {
	// La primera tanda (1,2) la entrega el propio StreamAnimeDetails y el
	// resto llega por onBatch DESPUÉS: sin esperar, saldrían solo 2.
	reducirEspera(t, 5*time.Millisecond, 2*time.Second)
	liberar := make(chan struct{})
	terminado := make(chan struct{})

	// bloqueante entre el primer onBatch y el resto, como en el test real de
	// adjuntarse al stream en curso: el canal no se cierra hasta DESPUÉS de
	// comprobar que aún no ha retornado con la lista parcial.
	esperando := make(chan struct{})

	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			go func() {
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:3", Number: 3, Kind: models.KindEpisode},
						{ID: "jkanime:bleach:episode:4", Number: 4, Kind: models.KindEpisode}}})
				close(esperando)
				<-liberar
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3,
					Episodes: []models.Episode{{ID: "jkanime:bleach:episode:5", Number: 5, Kind: models.KindEpisode}}})
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
				close(terminado)
			}()
			return models.AnimeDetail{ID: id}, episodiosDe(2), nil
		},
	}
	e := nuevoEntorno(t, fake)

	salida := make(chan struct {
		episodios []models.Episode
		err       error
	}, 1)
	go func() {
		_, eps, err := e.n.episodiosCompletos(context.Background(), proveedorPrincipal, "bleach")
		salida <- struct {
			episodios []models.Episode
			err       error
		}{eps, err}
	}()

	<-esperando // el lote 2 ya se emitió, pero el stream sigue incompleto
	select {
	case <-salida:
		t.Fatal("episodiosCompletos no debe retornar antes del lote Done")
	default:
	}

	close(liberar)
	select {
	case <-terminado:
	case <-time.After(2 * time.Second):
		t.Fatal("el stream no terminó tras liberarlo")
	}

	r := <-salida
	if r.err != nil {
		t.Fatalf("episodiosCompletos: %v", r.err)
	}
	if len(r.episodios) != 5 {
		t.Errorf("episodios = %d; se esperaban los 5 (2 de la primera tanda + 2 + 1 por lotes)", len(r.episodios))
	}
}

func TestEpisodiosCompletosNoDevuelveListaParcialSiElStreamFalla(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2,
				Episodes: []models.Episode{{Number: 3, Kind: models.KindEpisode}}})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true, Err: "la página 3 falló"})
			return models.AnimeDetail{ID: id}, episodiosDe(2), nil
		},
	}
	e := nuevoEntorno(t, fake)

	_, eps, err := e.n.episodiosCompletos(context.Background(), proveedorPrincipal, "bleach")
	if err == nil {
		t.Fatal("con el stream caído debe devolver error, no una lista")
	}
	if len(eps) != 0 {
		t.Errorf("episodios = %+v; nunca se devuelve la lista parcial", eps)
	}
	if !contains(err.Error(), "no se pudo completar la lista de episodios") {
		t.Errorf("err = %q; se esperaba el mensaje español de lista incompleta", err)
	}
}

func TestEpisodiosCompletosSinStreamerUsaLaListaSincrona(t *testing.T) {
	fake := &proveedorFalso{
		nombre: proveedorPrincipal,
		detalleF: func(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
			// Un proveedor sin EpisodeStreamer devuelve la lista COMPLETA
			// y síncrona (además, sin cachearse).
			return models.AnimeDetail{ID: id}, episodiosDe(3), nil
		},
	}
	e := nuevoEntorno(t, fake)

	_, eps, err := e.n.episodiosCompletos(context.Background(), proveedorPrincipal, "bleach")
	if err != nil {
		t.Fatalf("episodiosCompletos: %v", err)
	}
	if len(eps) != 3 {
		t.Errorf("episodios = %d; la lista síncrona debe venir entera (3)", len(eps))
	}
	if _, ok := e.cache.Get(cache.Clave(proveedorPrincipal, "bleach")); ok {
		t.Error("el flujo sin EpisodeStreamer no se cachea")
	}
}

func TestEpisodiosCompletosCtxCanceladoDevuelveError(t *testing.T) {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			// El lote queda incompleto a propósito: la única salida del bucle
			// de espera es el ctx, que ya está cancelado.
			return models.AnimeDetail{ID: id}, episodiosDe(1), nil
		},
	}
	e := nuevoEntorno(t, fake)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, eps, err := e.n.episodiosCompletos(ctx, proveedorPrincipal, "bleach")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; se esperaba context.Canceled", err)
	}
	if len(eps) != 0 {
		t.Errorf("episodios = %+v; con el ctx cancelado no se devuelve lista", eps)
	}
}

func TestEpisodiosCompletosTiempoMaximo(t *testing.T) {
	// Un fake que NUNCA envía el lote Done deja la entrada incompleta
	// para siempre: solo el tope de espera la desbloquea.
	reducirEspera(t, 5*time.Millisecond, 50*time.Millisecond)
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: proveedorPrincipal},
		streamF: func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			return models.AnimeDetail{ID: id}, episodiosDe(1), nil
		},
	}
	e := nuevoEntorno(t, fake)

	_, eps, err := e.n.episodiosCompletos(context.Background(), proveedorPrincipal, "bleach")
	if err == nil {
		t.Fatal("agotada la espera debe devolver error, no una lista")
	}
	if !contains(err.Error(), "no se pudo completar la lista de episodios") {
		t.Errorf("err = %q; se esperaba el mensaje español de lista incompleta", err)
	}
	if len(eps) != 0 {
		t.Errorf("episodios = %+v; nunca se devuelve la lista parcial", eps)
	}
}

// contains comprueba el texto del error en español.
func contains(s, sub string) bool { return strings.Contains(s, sub) }
