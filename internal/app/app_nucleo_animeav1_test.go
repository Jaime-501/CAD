package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"miapp/internal/cache"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/jkanime/extractors"
	"miapp/internal/stream"
)

// animeAV1DePrueba monta un doble con el nombre real del proveedor para
// recorrer el núcleo igual que lo haría animeav1 (EpisodeStreamer + fuentes
// con idioma y cabeceras, T33).
func animeAV1DePrueba(fuentes func(context.Context, string, scraper.SourceOpts) ([]models.VideoSource, error),
	stream func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error)) *streamerFalso {
	fake := &streamerFalso{
		proveedorFalso: proveedorFalso{nombre: "animeav1", fuentesF: fuentes},
		streamF:        stream,
	}
	return fake
}

// TestAnimeAV1PuntaAPuntaDetalleLotesYCaché recorre AnimeAV1 a través del
// núcleo con dobles: detalle por lotes (T32/T33: el primero vuelve por valor
// y el resto sale por episodes:update), eventos, y el hit de caché de T21 en
// la segunda llamada (el proveedor NO se vuelve a tocar).
func TestAnimeAV1PuntaAPuntaDetalleLotesYCaché(t *testing.T) {
	detalle := models.AnimeDetail{
		ID: "bleach", Title: "Bleach", Provider: "animeav1", Type: "tv-anime", TotalEpisodes: 366,
	}
	primeraTanda := []models.Episode{
		{ID: "animeav1:bleach:episode:1", Number: 1},
		{ID: "animeav1:bleach:episode:2", Number: 2},
	}
	fake := animeAV1DePrueba(nil,
		func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{
				AnimeID: id, Batch: 2,
				Episodes: []models.Episode{
					{ID: "animeav1:bleach:episode:51", Number: 51},
					{ID: "animeav1:bleach:episode:52", Number: 52},
				},
			})
			onBatch(models.EpisodeBatch{AnimeID: id, Batch: 2, Done: true})
			return detalle, primeraTanda, nil
		})
	e := nuevoEntorno(t, fake)
	ctx := context.Background()

	primera, err := e.n.Detalle(ctx, "animeav1", "bleach")
	if err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if primera.Detail.Provider != "animeav1" || primera.Detail.Type != "tv-anime" || primera.Detail.TotalEpisodes != 366 {
		t.Errorf("detalle = %+v", primera.Detail)
	}
	if len(primera.Episodes) != 2 || primera.Episodes[0].ID != "animeav1:bleach:episode:1" {
		t.Errorf("primera tanda = %+v", primera.Episodes)
	}
	lotes := lotesDe(t, e.emis.lista(), "bleach")
	if len(lotes) != 2 || !lotes[1].Done {
		t.Errorf("lotes = %+v; quería el lote 2 y el Done", lotes)
	}
	if c := completadosDe(t, e.emis.lista(), "bleach"); len(c) != 1 || c[0].Error != "" {
		t.Errorf("completados = %+v", c)
	}

	// Segunda llamada: hit de caché completo (T21), sin volver al proveedor
	// y con todos los episodios ordenados por número.
	segunda, err := e.n.Detalle(ctx, "animeav1", "bleach")
	if err != nil {
		t.Fatalf("segunda detalle: %v", err)
	}
	if ops := fake.operaciones(); len(ops) != 1 || ops[0] != "StreamAnimeDetails" {
		t.Errorf("ops = %v; la caché debe evitar un segundo stream", ops)
	}
	if len(segunda.Episodes) != 4 {
		t.Errorf("episodios = %+v; debía venir la tanda 1 + el lote 2 (4)", segunda.Episodes)
	}
	for i, ep := range segunda.Episodes {
		if i > 0 && ep.Number < segunda.Episodes[i-1].Number {
			t.Errorf("los episodios no salen ordenados: %+v", segunda.Episodes)
			break
		}
	}
	if c := completadosDe(t, e.emis.lista(), "bleach"); len(c) != 2 || c[1].Error != "" {
		t.Errorf("el hit debe emitir otro episodes:complete: %+v", c)
	}
}

// TestAnimeAV1CancelacionDelCtxDetieneLosLotes: cancelar el ctx del llamado
// (como hace el shutdown o el cambio de anime) corta los lotes siguientes,
// ni siquiera el Done, y no deja la goroutine huérfana.
func TestAnimeAV1CancelacionDelCtxDetieneLosLotes(t *testing.T) {
	terminado := make(chan struct{})
	fake := animeAV1DePrueba(nil,
		func(ctx context.Context, id string, onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
			onBatch(models.EpisodeBatch{
				AnimeID: id, Batch: 2,
				Episodes: []models.Episode{{ID: "animeav1:bleach:episode:51", Number: 51}},
			})
			go func() {
				defer close(terminado)
				<-ctx.Done()
				onBatch(models.EpisodeBatch{
					AnimeID: id, Batch: 3,
					Episodes: []models.Episode{{ID: "animeav1:bleach:episode:101", Number: 101}},
				})
				onBatch(models.EpisodeBatch{AnimeID: id, Batch: 3, Done: true})
			}()
			return models.AnimeDetail{ID: id, Provider: "animeav1"},
				[]models.Episode{{ID: "animeav1:bleach:episode:1", Number: 1}}, nil
		})
	e := nuevoEntorno(t, fake)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := e.n.Detalle(ctx, "animeav1", "bleach"); err != nil {
		t.Fatalf("detalle: %v", err)
	}
	e.emis.esperar(t, 1) // el lote 2 ya llegó

	cancel()
	select {
	case <-terminado:
	case <-time.After(2 * time.Second):
		t.Fatal("el stream de animeav1 no terminó tras cancelarse")
	}

	lotes := lotesDe(t, e.emis.lista(), "bleach")
	if len(lotes) != 1 || lotes[0].Batch != 2 {
		t.Errorf("lotes = %+v; tras cancelar solo debe llegar el primero", lotes)
	}
	if c := completadosDe(t, e.emis.lista(), "bleach"); len(c) != 0 {
		t.Errorf("un stream cancelado no emite episodes:complete: %+v", c)
	}
}

// TestAnimeAV1CabecerasSoloVivenEnElProxy es la prueba del riesgo 1 de T30:
// las fuentes HLS de AnimeAV1 traen Headers (Sec-Fetch-Site: same-origin, que
// los segmentos exigen con 403 si falta). Aquí se comprueba con el proxy REAL
// que (a) el origen SÍ recibe la cabecera a través del proxy y (b) la fuente
// que cruza el puente de Wails hacia el frontend sale SIN Headers ni Referer.
func TestAnimeAV1CabecerasSoloVivenEnElProxy(t *testing.T) {
	// Origen falsa del "player": registra las cabeceras que llegan.
	var mu sync.Mutex
	var recibidas http.Header
	origen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		recibidas = r.Header.Clone()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = w.Write([]byte("#EXTM3U\n#EXT-X-ENDLIST\n"))
	}))
	t.Cleanup(origen.Close)

	// Proxy real escuchando, como en producción (T19). El httptest vive en
	// 127.0.0.1, así que hace falta PermitirRedPrivada (solo en el test).
	px := stream.NewProxy(nil, stream.PermitirRedPrivada())
	base, err := px.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	t.Cleanup(func() { _ = px.Cerrar() })

	// Fuentes con la forma que devuelve animeav1 en T33: HLS derivado con su
	// cabecera + un iframe sin resolver, ambos con Language (SUB/DUB).
	fake := animeAV1DePrueba(
		func(context.Context, string, scraper.SourceOpts) ([]models.VideoSource, error) {
			return []models.VideoSource{
				{
					URL:      origen.URL + "/m3u8/65f3aa11bb22cc33dd44ee55ff660077",
					Type:     models.ResourceHLS,
					Server:   "HLS",
					Language: models.LanguageSub,
					Headers:  map[string]string{"Sec-Fetch-Site": "same-origin"},
				},
				{
					URL:      "https://voe.sx/e/abc123",
					Type:     models.ResourceIframe,
					Server:   "VOE",
					Language: models.LanguageDub,
				},
			}, nil
		}, nil)

	reg := scraper.NewRegistry()
	if err := reg.Register(fake); err != nil {
		t.Fatalf("registro: %v", err)
	}
	n := NuevoNucleo(reg, px, extractors.SelectBestSource, nuevoEmisorFalso().emitir,
		cache.New(ttlCacheDetalle), t.TempDir())
	t.Cleanup(func() { _ = n.Cerrar() })

	salida, err := n.Fuentes(context.Background(), "animeav1", "animeav1:bleach:episode:1", false)
	if err != nil {
		t.Fatalf("fuentes: %v", err)
	}
	if len(salida) != 2 {
		t.Fatalf("salida = %+v; quería HLS e iframe", salida)
	}

	hls := salida[0]
	if hls.Type != models.ResourceHLS {
		t.Fatalf("la primera (mejor) debe ser la HLS: %+v", hls)
	}
	// (b) lo que ve el frontend: sin cabeceras ni referer...
	if hls.Headers != nil || hls.Referer != "" {
		t.Errorf("la fuente hacia el frontend arrastra cabeceras: %+v", hls)
	}
	// ...pero SÍ con el idioma, que es lo que pinta la etiqueta (T34).
	if hls.Language != models.LanguageSub {
		t.Errorf("Language = %q; la etiqueta del reproductor lo necesita", hls.Language)
	}
	if !strings.HasPrefix(hls.URL, base+"/proxy?") {
		t.Errorf("la HLS no apunta al proxy: %q (base %q)", hls.URL, base)
	}
	puente, err := json.Marshal(hls)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(puente), "Sec-Fetch-Site") {
		t.Errorf("el JSON que cruza el puente de Wails contiene la cabecera: %s", puente)
	}
	// El iframe no pasa por el proxy y conserva lo suyo (incluido el idioma).
	if iframe := salida[1]; iframe.Type != models.ResourceIframe ||
		iframe.URL != "https://voe.sx/e/abc123" || iframe.Language != models.LanguageDub {
		t.Errorf("el iframe se tocó: %+v", iframe)
	}

	// (a) petición REAL a través del proxy: el origen debe recibir la
	// cabecera Sec-Fetch-Site que guardó la sesión.
	resp, err := http.Get(hls.URL)
	if err != nil {
		t.Fatalf("GET a través del proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("el proxy respondió %d", resp.StatusCode)
	}
	mu.Lock()
	originRecibidas := recibidas
	mu.Unlock()
	if originRecibidas.Get("Sec-Fetch-Site") != "same-origin" {
		t.Errorf("el origen no recibió Sec-Fetch-Site: %v", originRecibidas)
	}
	if originRecibidas.Get("Referer") != "" {
		t.Errorf("AnimeAV1 no declara Referer y no debe inventarse: %v", originRecibidas)
	}
}

// TestMP4UploadRefererSoloViveEnElProxy es la prueba del requisito 5 de T61:
// la fuente mp4 resuelta de MP4Upload lleva un Referer (la URL del embed) que
// EXIGE el CDN y va en un host con puerto NO estándar. Con el proxy REAL se
// comprueba que (a) la fuente que cruza a Wails sale sin Referer ni Headers,
// (b) el origen SÍ recibe el Referer (no 403/000) y (c) el Range viaja y el
// origen responde 206. Si el Referer no se guardara en la sesión, MP4Upload
// devolvería 403 y el mp4 no se vería.
func TestMP4UploadRefererSoloViveEnElProxy(t *testing.T) {
	var mu sync.Mutex
	var recibidas http.Header
	var metodo, rango string
	origen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		recibidas = r.Header.Clone()
		metodo, rango = r.Method, r.Header.Get("Range")
		mu.Unlock()
		w.Header().Set("Content-Type", "video/mp4")
		if r.Header.Get("Range") == "bytes=0-0" {
			w.Header().Set("Content-Range", "bytes 0-0/1024")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("M"))
			return
		}
		_, _ = w.Write([]byte("MP4_DE_PRUEBA"))
	}))
	t.Cleanup(origen.Close)

	// Proxy real como en producción (T19). El CDN falso vive en el 127.0.0.1
	// del httptest con un puerto NO estándar: hace falta PermitirRedPrivada
	// (solo aquí) y sirve además para probar que el puerto se conserva.
	px := stream.NewProxy(nil, stream.PermitirRedPrivada())
	base, err := px.Escuchar()
	if err != nil {
		t.Fatalf("Escuchar: %v", err)
	}
	t.Cleanup(func() { _ = px.Cerrar() })

	const embedURL = "https://www.mp4upload.com/embed-i8l69rmmcvhi.html"
	cdnURL := origen.URL + "/d/TOKEN_DE_PRUEBA/video.mp4"
	fake := animeAV1DePrueba(
		func(context.Context, string, scraper.SourceOpts) ([]models.VideoSource, error) {
			return []models.VideoSource{{
				URL:      cdnURL,
				Type:     models.ResourceMP4,
				Server:   "MP4Upload",
				Language: models.LanguageSub,
				Referer:  embedURL,
				Headers:  map[string]string{"Sec-Fetch-Site": "same-origin"},
			}}, nil
		}, nil)

	reg := scraper.NewRegistry()
	if err := reg.Register(fake); err != nil {
		t.Fatalf("registro: %v", err)
	}
	n := NuevoNucleo(reg, px, extractors.SelectBestSource, nuevoEmisorFalso().emitir,
		cache.New(ttlCacheDetalle), t.TempDir())
	t.Cleanup(func() { _ = n.Cerrar() })

	salida, err := n.Fuentes(context.Background(), "animeav1", "animeav1:bleach:episode:1", false)
	if err != nil {
		t.Fatalf("fuentes: %v", err)
	}
	if len(salida) != 1 || salida[0].Type != models.ResourceMP4 {
		t.Fatalf("salida = %+v; quería el mp4 resuelto", salida)
	}
	mp4 := salida[0]

	// (a) Lo que ve el frontend: proxificado, sin cabeceras ni referer...
	if mp4.Referer != "" || mp4.Headers != nil {
		t.Errorf("la fuente hacia el frontend arrastra cabeceras: %+v", mp4)
	}
	if !strings.HasPrefix(mp4.URL, base+"/proxy?") {
		t.Fatalf("el mp4 no apunta al proxy: %q (base %q)", mp4.URL, base)
	}
	puente, err := json.Marshal(mp4)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, prohibido := range []string{"Sec-Fetch-Site", "Referer", "mp4upload.com/embed-"} {
		if strings.Contains(string(puente), prohibido) {
			t.Errorf("el JSON que cruza el puente de Wails contiene %q: %s", prohibido, puente)
		}
	}

	// (b) petición REAL con Range a través del proxy: el origen debe recibir
	// el Referer de la sesión y responder 206.
	req, err := http.NewRequest(http.MethodGet, mp4.URL, nil)
	if err != nil {
		t.Fatalf("preparando la petición: %v", err)
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET a través del proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("el proxy respondió %d para el mp4 con Range; se esperaba 206 (¿403 por Referer perdido?)",
			resp.StatusCode)
	}
	if cr := resp.Header.Get("Content-Range"); !strings.HasPrefix(cr, "bytes 0-0/") {
		t.Errorf("Content-Range del proxy = %q; se esperaba bytes 0-0/<tamaño>", cr)
	}

	mu.Lock()
	cabeceras, m, rg := recibidas, metodo, rango
	mu.Unlock()
	if m != http.MethodGet || rg != "bytes=0-0" {
		t.Errorf("el origen vio %s con Range %q; se esperaba GET con bytes=0-0", m, rg)
	}
	if cabeceras.Get("Referer") != embedURL {
		t.Errorf("el origen no recibió el Referer del embed: %v (quería %q)", cabeceras.Get("Referer"), embedURL)
	}
	if cabeceras.Get("Sec-Fetch-Site") != "same-origin" {
		t.Errorf("el origen no recibió Sec-Fetch-Site: %v", cabeceras.Get("Sec-Fetch-Site"))
	}
}
