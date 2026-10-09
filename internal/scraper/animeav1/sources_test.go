package animeav1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

// fuentesBleach1 es la salida esperada de bleach/1 con el fixture real:
// recorrido SUB → DUB dentro de cada servidor, el HLS propio primero y ni
// una sola URL de downloads (Mega/1Fichier/MP4Upload de descarga).
var fuentesBleach1 = []struct {
	servidor string
	url      string
	idioma   string
	tipo     models.ResourceType
}{
	{"HLS", "https://player.zilla-networks.com/m3u8/65f33e9a44ff7767faf60d1dae9eb1e6", models.LanguageSub, models.ResourceHLS},
	{"HLS", "https://player.zilla-networks.com/m3u8/62f0bdd0728f76b50d49348a17baea3a", models.LanguageDub, models.ResourceHLS},
	{"UPNShare", "https://animeav1.uns.bio/#t1tib8", models.LanguageSub, models.ResourceIframe},
	{"UPNShare", "https://animeav1.uns.bio/#vyveci", models.LanguageDub, models.ResourceIframe},
	{"Voe", "https://voe.sx/e/mhuxdfwigwqp", models.LanguageSub, models.ResourceIframe},
	{"Voe", "https://voe.sx/e/mwpqlc2nfdwp", models.LanguageDub, models.ResourceIframe},
	{"Byse", "https://byselapuix.com/e/38ajw9wez0hk", models.LanguageSub, models.ResourceIframe},
	{"Byse", "https://byselapuix.com/e/efptdd773cdu", models.LanguageDub, models.ResourceIframe},
	{"MP4Upload", "https://www.mp4upload.com/embed-wuf5r9fzzo9q.html", models.LanguageSub, models.ResourceIframe},
	{"MP4Upload", "https://www.mp4upload.com/embed-i8l69rmmcvhi.html", models.LanguageDub, models.ResourceIframe},
}

// urlJugador es la URL del embed del reproductor propio en el fixture de
// la película (la que se muta en los casos de validación).
const urlJugador = "https://player.zilla-networks.com/play/faaf7b08cc8d70e376b9007f20f74554"

// fuentesConCuerpo sirve cuerpo en ruta y pide las fuentes de episodeID.
func fuentesConCuerpo(t *testing.T, episodeID, ruta string, cuerpo []byte) ([]models.VideoSource, error) {
	t.Helper()
	srv, _ := servidorJSON(t, cuerpo, ruta)
	a := New(clientePrueba(), srv.URL)
	return a.GetVideoSources(context.Background(), episodeID, scraper.SourceOpts{})
}

// fuentesDeFixture es fuentesConCuerpo con un JSON real de testdata.
func fuentesDeFixture(t *testing.T, nombre, episodeID, ruta string) []models.VideoSource {
	t.Helper()
	fuentes, err := fuentesConCuerpo(t, episodeID, ruta, fixture(t, nombre))
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	return fuentes
}

// cuerpoModificado cambia un literal de un fixture real (payload
// artificial: ningún episodio real trae URLs inválidas ni repetidas; se
// anota como desviación en PROGRESO.md, regla 11).
func cuerpoModificado(t *testing.T, base, vieja, nueva string) []byte {
	t.Helper()
	cruso := string(fixture(t, base))
	if !strings.Contains(cruso, vieja) {
		t.Fatalf("el fixture %s no contiene %q", base, vieja)
	}
	return []byte(strings.Replace(cruso, vieja, nueva, 1))
}

func TestFuentesRutaUserAgentYUnaPeticion(t *testing.T) {
	var ruta, ua string
	var peticiones int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones++
		ruta = r.URL.EscapedPath()
		ua = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "episode_data.json"))
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	fuentes, err := a.GetVideoSources(context.Background(), "animeav1:bleach:episode:1", scraper.SourceOpts{})
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	if peticiones != 1 {
		t.Errorf("peticiones = %d, quería 1 (una sola)", peticiones)
	}
	if ruta != "/media/bleach/1/__data.json" {
		t.Errorf("ruta pedida = %q, quería /media/bleach/1/__data.json", ruta)
	}
	if ua != httpx.DefaultUserAgent {
		t.Errorf("User-Agent = %q, quería el de httpx", ua)
	}
	if len(fuentes) != len(fuentesBleach1) {
		t.Fatalf("fuentes = %d, quería %d (5 por idioma, sin downloads)", len(fuentes), len(fuentesBleach1))
	}
	for i, want := range fuentesBleach1 {
		got := fuentes[i]
		if got.Server != want.servidor || got.URL != want.url || got.Language != want.idioma || got.Type != want.tipo {
			t.Errorf("fuente %d = {%s %s %s %s}, quería {%s %s %s %s}",
				i, got.Server, got.Language, got.Type, got.URL,
				want.servidor, want.idioma, want.tipo, want.url)
		}
	}
}

// TestFuentesUsaElNumeroDelID comprueba que episodio y especial piden el
// número que traen el id (la película, que no lo lleva, tiene su test
// aparte: TestFuentesPeliculaPideElUno).
func TestFuentesUsaElNumeroDelID(t *testing.T) {
	casos := []struct {
		nombre, episodeID, ruta string
	}{
		{"episodio", "animeav1:bleach:episode:7", "/media/bleach/7/__data.json"},
		{"especial", "animeav1:bleach:special:2", "/media/bleach/2/__data.json"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := fuentesConCuerpo(t, c.episodeID, c.ruta, fixture(t, "episode_data.json"))
			if err != nil {
				t.Fatalf("GetVideoSources falló: %v", err)
			}
		})
	}
}

// TestFuentesPeliculaPideElUno: el id de película no lleva número
// (models.ParseEpisodeID devuelve 0) pero la URL real del sitio es
// /media/<slug>/1/__data.json — servidorJSON falla si se pide otra cosa.
func TestFuentesPeliculaPideElUno(t *testing.T) {
	srv, peticiones := servidorJSON(t, fixture(t, "episode_movie_data.json"),
		"/media/bleach-movie-1-memories-of-nobody/1/__data.json")
	a := New(clientePrueba(), srv.URL)

	fuentes, err := a.GetVideoSources(context.Background(),
		"animeav1:bleach-movie-1-memories-of-nobody:movie", scraper.SourceOpts{})
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Errorf("peticiones = %d, quería 1", n)
	}

	// La película solo trae SUB en el sitio: 5 fuentes, ninguna con DUB.
	if len(fuentes) != 5 {
		t.Fatalf("fuentes = %d, quería 5 (las del embed de la película)", len(fuentes))
	}
	if fuentes[0].URL != "https://player.zilla-networks.com/m3u8/faaf7b08cc8d70e376b9007f20f74554" ||
		fuentes[0].Type != models.ResourceHLS || fuentes[0].Language != models.LanguageSub {
		t.Errorf("primera fuente = %+v, quería el HLS SUB de la película", fuentes[0])
	}
	ordenServidores := []string{}
	for _, f := range fuentes {
		if f.Language != models.LanguageSub {
			t.Errorf("idioma = %q, quería SUB (la película no trae DUB)", f.Language)
		}
		ordenServidores = append(ordenServidores, f.Server)
	}
	queria := "HLS,UPNShare,Voe,Byse,MP4Upload"
	if got := strings.Join(ordenServidores, ","); got != queria {
		t.Errorf("servidores = %q, quería %q", got, queria)
	}
}

// TestFuentesHlsDerivado valida la URL exacta del manifiesto y su cabecera
// (los segmentos responden 403 sin Sec-Fetch-Site y el navegador no puede
// ponerla: la inyecta el proxy local).
func TestFuentesHlsDerivado(t *testing.T) {
	fuentes := fuentesDeFixture(t, "episode_data.json", "animeav1:bleach:episode:1", "/media/bleach/1/__data.json")

	hls := 0
	for _, f := range fuentes {
		if f.Type != models.ResourceHLS {
			continue
		}
		hls++
		if f.Headers[cabeceraFetch] != valorFetch {
			t.Errorf("Headers[%q] = %q, quería %q", cabeceraFetch, f.Headers[cabeceraFetch], valorFetch)
		}
		if len(f.Headers) != 1 {
			t.Errorf("Headers = %v, quería solo %q", f.Headers, cabeceraFetch)
		}
		if f.Referer != "" {
			t.Errorf("Referer = %q, quería vacío (el manifiesto no exige Referer)", f.Referer)
		}
		if f.Quality != "" || f.Subtitles != nil {
			t.Errorf("fuente con campos que deberían venir vacíos: %+v", f)
		}
	}
	if hls != 2 {
		t.Errorf("fuentes HLS = %d, quería 2 (SUB y DUB)", hls)
	}
	if fuentes[0].URL != "https://player.zilla-networks.com/m3u8/65f33e9a44ff7767faf60d1dae9eb1e6" {
		t.Errorf("primera fuente = %q, quería el manifiesto derivado del embed SUB", fuentes[0].URL)
	}
}

// TestFuentesParIdiomaPorServidor exige el par exacto (SUB, DUB) dentro de
// cada servidor —no solo que estén adyacentes— y el orden de servidores
// (HLS primero, después el del sitio) sin ningún resto de idioma.
func TestFuentesParIdiomaPorServidor(t *testing.T) {
	fuentes := fuentesDeFixture(t, "episode_data.json", "animeav1:bleach:episode:1", "/media/bleach/1/__data.json")

	ordenServidores := []string{}
	idiomas := make(map[string][]string)
	for _, f := range fuentes {
		if f.Language != models.LanguageSub && f.Language != models.LanguageDub {
			t.Fatalf("idioma inesperado %q en la fuente %+v", f.Language, f)
		}
		if _, ya := idiomas[f.Server]; !ya {
			ordenServidores = append(ordenServidores, f.Server)
		}
		idiomas[f.Server] = append(idiomas[f.Server], f.Language)
	}

	if got, queria := strings.Join(ordenServidores, ","), "HLS,UPNShare,Voe,Byse,MP4Upload"; got != queria {
		t.Errorf("orden de servidores = %q, quería %q", got, queria)
	}
	if fuentes[0].Type != models.ResourceHLS {
		t.Errorf("la primera fuente debe ser la del HLS propio, es %+v", fuentes[0])
	}
	for _, servidor := range ordenServidores {
		par := idiomas[servidor]
		if len(par) != 2 || par[0] != models.LanguageSub || par[1] != models.LanguageDub {
			t.Errorf("servidor %q: idiomas %v, quería el par exacto [SUB DUB]", servidor, par)
		}
	}
}

// TestFuentesIframesYDownloadsIgnorados: los embeds que no son del
// reproductor propio salen como iframe con su URL intacta, sin cabeceras
// ni Referer por verificar, y los downloads (Mega, 1Fichier, MP4Upload de
// descarga) no producen ninguna fuente.
func TestFuentesIframesYDownloadsIgnorados(t *testing.T) {
	fuentes := fuentesDeFixture(t, "episode_data.json", "animeav1:bleach:episode:1", "/media/bleach/1/__data.json")

	descargas := []string{
		"https://mega.nz/file/",
		"https://1fichier.com/?",
		"https://www.mp4upload.com/wuf5r9fzzo9q",
		"https://www.mp4upload.com/i8l69rmmcvhi",
	}
	iframes := 0
	for _, f := range fuentes {
		for _, d := range descargas {
			if strings.Contains(f.URL, d) {
				t.Errorf("la fuente %q viene de downloads: los enlaces de descarga se ignoran", f.URL)
			}
		}
		u, err := url.Parse(f.URL)
		if err != nil || u.Host == "" {
			t.Errorf("URL no válida: %q", f.URL)
			continue
		}
		if f.Type == models.ResourceIframe {
			iframes++
			if f.Headers != nil {
				t.Errorf("iframe con Headers %v: sin petición propia no hay cabeceras que poner", f.Headers)
			}
			if f.Referer != "" {
				t.Errorf("iframe con Referer %q: no está verificado que lo exija", f.Referer)
			}
		}
	}
	if iframes != 8 {
		t.Errorf("iframes = %d, quería 8 (4 servidores × 2 idiomas)", iframes)
	}
}

// TestFuentesEmbedNoDerivaEsIframe: todo lo que no cumpla la regla del
// reproductor (host, esquema, path, id de 32 hex) se queda como iframe con
// la URL del embed; nunca se construye una URL a ciegas.
func TestFuentesEmbedNoDerivaEsIframe(t *testing.T) {
	casos := []struct {
		nombre, vieja, nueva string
	}{
		{"id de 31", "faaf7b08cc8d70e376b9007f20f74554", "faaf7b08cc8d70e376b9007f20f7455"},
		{"id de 33", "faaf7b08cc8d70e376b9007f20f74554", "faaf7b08cc8d70e376b9007f20f74554a"},
		{"id no hexadecimal", "faaf7b08cc8d70e376b9007f20f74554", "faaf7b08cc8d70e376b9007f20f7455z"},
		{"host distinto", "https://player.zilla-networks.com/play/", "https://player2.zilla-networks.com/play/"},
		{"esquema http", "https://player.zilla-networks.com/play/", "http://player.zilla-networks.com/play/"},
		{"path distinto", "https://player.zilla-networks.com/play/", "https://player.zilla-networks.com/repro/"},
		{"con query", "faaf7b08cc8d70e376b9007f20f74554", "faaf7b08cc8d70e376b9007f20f74554?x=1"},
		{"con fragmento", "faaf7b08cc8d70e376b9007f20f74554", "faaf7b08cc8d70e376b9007f20f74554#frag"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cuerpo := cuerpoModificado(t, "episode_movie_data.json", c.vieja, c.nueva)
			fuentes, err := fuentesConCuerpo(t, "animeav1:bleach-movie-1-memories-of-nobody:movie",
				"/media/bleach-movie-1-memories-of-nobody/1/__data.json", cuerpo)
			if err != nil {
				t.Fatalf("GetVideoSources falló: %v", err)
			}
			if len(fuentes) != 5 {
				t.Fatalf("fuentes = %d, quería 5", len(fuentes))
			}
			queria := strings.Replace(urlJugador, c.vieja, c.nueva, 1)
			primera := fuentes[0]
			if primera.Type != models.ResourceIframe {
				t.Errorf("tipo = %s, quería iframe", primera.Type)
			}
			if primera.URL != queria {
				t.Errorf("URL = %q, quería el embed tal cual %q", primera.URL, queria)
			}
			if primera.Headers != nil {
				t.Errorf("Headers = %v, quería vacío en un iframe", primera.Headers)
			}
			if primera.Server != "HLS" {
				t.Errorf("Server = %q, quería el nombre del servidor del embed", primera.Server)
			}
		})
	}

	t.Run("id válido deriva el manifiesto", func(t *testing.T) {
		fuentes := fuentesDeFixture(t, "episode_movie_data.json",
			"animeav1:bleach-movie-1-memories-of-nobody:movie",
			"/media/bleach-movie-1-memories-of-nobody/1/__data.json")
		if fuentes[0].Type != models.ResourceHLS {
			t.Errorf("tipo = %s, quería hls", fuentes[0].Type)
		}
		if fuentes[0].URL != "https://player.zilla-networks.com/m3u8/faaf7b08cc8d70e376b9007f20f74554" {
			t.Errorf("URL = %q, quería el manifiesto derivado", fuentes[0].URL)
		}
	})
}

// TestFuentesDedupConservaElFragmento: dos URLs que solo difieren en el
// fragmento son dos recursos distintos y ambas sobreviven al dedup (el
// fragmento de un embed es a menudo su clave de descifrado).
func TestFuentesDedupConservaElFragmento(t *testing.T) {
	vieja := "https://voe.sx/e/mhuxdfwigwqp"
	nueva := "https://animeav1.uns.bio/#t1tib8-sub"
	cuerpo := cuerpoModificado(t, "episode_data.json", vieja, nueva)

	fuentes, err := fuentesConCuerpo(t, "animeav1:bleach:episode:1", "/media/bleach/1/__data.json", cuerpo)
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	if len(fuentes) != len(fuentesBleach1) {
		t.Fatalf("fuentes = %d, quería %d: el fragmento debe distinguir los recursos", len(fuentes), len(fuentesBleach1))
	}
	conOrigen := 0
	for _, f := range fuentes {
		if strings.HasPrefix(f.URL, "https://animeav1.uns.bio/") {
			conOrigen++
		}
	}
	if conOrigen != 3 {
		t.Errorf("fuentes en animeav1.uns.bio = %d, quería 3 (las dos del embed y la nueva)", conOrigen)
	}
}

// TestFuentesDedupConservaElIdioma: la misma URL publicada en SUB y en DUB
// son dos fuentes, cada una con su Language (desviación respecto al
// "deduplica por URL" de la ficha).
func TestFuentesDedupConservaElIdioma(t *testing.T) {
	vieja := "https://animeav1.uns.bio/#t1tib8"
	nueva := "https://animeav1.uns.bio/#vyveci"
	cuerpo := cuerpoModificado(t, "episode_data.json", vieja, nueva)

	fuentes, err := fuentesConCuerpo(t, "animeav1:bleach:episode:1", "/media/bleach/1/__data.json", cuerpo)
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	if len(fuentes) != len(fuentesBleach1) {
		t.Fatalf("fuentes = %d, quería %d: la misma URL con dos idiomas no se colapsa", len(fuentes), len(fuentesBleach1))
	}
	idiomas := []string{}
	for _, f := range fuentes {
		if f.URL == nueva {
			idiomas = append(idiomas, f.Language)
		}
	}
	if strings.Join(idiomas, ",") != "SUB,DUB" {
		t.Errorf("idiomas de la URL repetida = %v, quería [SUB DUB]", idiomas)
	}
}

// TestFuentesDedupMismaUrlMismoIdioma: dentro de un mismo idioma la URL
// repetida sí se colapsa (una sola fuente, conservando su idioma).
func TestFuentesDedupMismaUrlMismoIdioma(t *testing.T) {
	vieja := "https://byselapuix.com/e/38ajw9wez0hk"
	nueva := "https://voe.sx/e/mhuxdfwigwqp"
	cuerpo := cuerpoModificado(t, "episode_data.json", vieja, nueva)

	fuentes, err := fuentesConCuerpo(t, "animeav1:bleach:episode:1", "/media/bleach/1/__data.json", cuerpo)
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	if len(fuentes) != len(fuentesBleach1)-1 {
		t.Fatalf("fuentes = %d, quería %d: la URL repetida dentro de SUB se colapsa", len(fuentes), len(fuentesBleach1)-1)
	}
	repeticiones := 0
	for _, f := range fuentes {
		if f.URL == nueva {
			repeticiones++
			if f.Language != models.LanguageSub {
				t.Errorf("idioma = %q, quería SUB (la fuente conserva su idioma)", f.Language)
			}
		}
	}
	if repeticiones != 1 {
		t.Errorf("repeticiones de la URL = %d, quería 1", repeticiones)
	}
}

// TestFuentesSinEmbeds: sin un embed utilizable hay error descriptivo,
// nunca un slice vacío silencioso (payloads devalue artificiales).
func TestFuentesSinEmbeds(t *testing.T) {
	casos := []struct {
		nombre, cuerpo, contiene string
	}{
		{"sin el campo", `{"type":"data","nodes":[{"type":"data","data":[{}]}]}`, "no trae embeds"},
		{"embeds null", `{"type":"data","nodes":[{"type":"data","data":[{"embeds":1},null]}]}`, "no trae embeds"},
		{"embeds que no es objeto", `{"type":"data","nodes":[{"type":"data","data":[{"embeds":1},"x"]}]}`, "quería un objeto con SUB/DUB"},
		{"listas vacías", `{"type":"data","nodes":[{"type":"data","data":[{"embeds":1},{"SUB":2},[]]}]}`, "sin ninguna entrada reproducible"},
		{"idioma desconocido", `{"type":"data","nodes":[{"type":"data","data":[{"embeds":1},{"LAT":2},[]]}]}`, "sin ninguna entrada reproducible"},
		{"entrada sin url", `{"type":"data","nodes":[{"type":"data","data":[{"embeds":1},{"SUB":2},[3],{"server":4},"Byse"]}]}`, "sin ninguna entrada reproducible"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			fuentes, err := fuentesConCuerpo(t, "animeav1:mi-slug:episode:1",
				"/media/mi-slug/1/__data.json", []byte(c.cuerpo))
			if err == nil {
				t.Fatal("quería un error descriptivo, no un éxito silencioso")
			}
			if fuentes != nil {
				t.Errorf("fuentes = %#v, quería nil", fuentes)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}

// TestFuentesEntradaMalFormada: una entrada sin URL se descarta sin
// tumbar la buena de al lado (payload devalue artificial).
func TestFuentesEntradaMalFormada(t *testing.T) {
	cuerpo := `{"type":"data","nodes":[{"type":"data","data":[` +
		`{"embeds":1},{"SUB":2},[3,5],{"server":4},"Byse",` +
		`{"server":6,"url":7},"Voe","https://voe.sx/e/ok"]}]}`

	fuentes, err := fuentesConCuerpo(t, "animeav1:mi-slug:episode:1", "/media/mi-slug/1/__data.json", []byte(cuerpo))
	if err != nil {
		t.Fatalf("GetVideoSources falló: %v", err)
	}
	if len(fuentes) != 1 {
		t.Fatalf("fuentes = %d, quería 1 (solo la entrada con URL)", len(fuentes))
	}
	f := fuentes[0]
	if f.URL != "https://voe.sx/e/ok" || f.Server != "Voe" || f.Language != models.LanguageSub {
		t.Errorf("fuente = %+v, quería la entrada válida con su idioma", f)
	}
}

// TestFuentesPaginaInexistente: un episodio fuera de rango responde HTTP
// 200 con el nodo de error 404 de SvelteKit y devuelve el mismo error
// tipado que el detalle (T32).
func TestFuentesPaginaInexistente(t *testing.T) {
	fuentes, err := fuentesConCuerpo(t, "animeav1:bleach:episode:367",
		"/media/bleach/367/__data.json", fixture(t, "episode_404_data.json"))
	if fuentes != nil {
		t.Errorf("fuentes = %#v, quería nil", fuentes)
	}
	if !errors.Is(err, httpx.ErrNotFound) {
		t.Fatalf("err = %v, quería httpx.ErrNotFound", err)
	}
	var scr *httpx.ScraperError
	if !errors.As(err, &scr) || scr.StatusCode != http.StatusNotFound || scr.Operation != "sources" {
		t.Errorf("err = %#v, quería un ScraperError 404 de sources", err)
	}
}

func TestFuentesErrorHTTP(t *testing.T) {
	casos := []struct {
		nombre string
		status int
		queria error
	}{
		{"404", http.StatusNotFound, httpx.ErrNotFound},
		{"500", http.StatusInternalServerError, httpx.ErrServerError},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "error", c.status)
			}))
			t.Cleanup(srv.Close)
			a := New(clientePrueba(), srv.URL)

			_, err := a.GetVideoSources(context.Background(), "animeav1:bleach:episode:1", scraper.SourceOpts{})
			if err == nil {
				t.Fatalf("quería error con un %d", c.status)
			}
			if !errors.Is(err, c.queria) {
				t.Errorf("err = %v, quería %v", err, c.queria)
			}
		})
	}
}

func TestFuentesIdInvalidoNoPideRed(t *testing.T) {
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		t.Error("GetVideoSources no debía hacer ninguna petición")
		return nil, errors.New("sin red (doer de prueba)")
	}}
	a := New(d, "")

	casos := []struct {
		nombre, episodeID string
		contiene          string
	}{
		{"id vacío", "", "id vacío"},
		{"número no canónico", "animeav1:bleach:episode:01", "número de episodio no válido"},
		{"tipo desconocido", "animeav1:bleach:ova:1", "tipo de episodio desconocido"},
		{"otro proveedor", "jkanime:bleach:episode:1", "pertenece al proveedor"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			fuentes, err := a.GetVideoSources(context.Background(), c.episodeID, scraper.SourceOpts{})
			if err == nil {
				t.Fatal("quería error con un id inválido")
			}
			if fuentes != nil {
				t.Errorf("fuentes = %#v, quería nil", fuentes)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}

func TestFuentesSinDoer(t *testing.T) {
	a := New(nil, "")
	_, err := a.GetVideoSources(context.Background(), "animeav1:bleach:episode:1", scraper.SourceOpts{})
	if err == nil {
		t.Fatal("quería error sin doer")
	}
	if !errors.Is(err, httpx.ErrNilDoer) {
		t.Errorf("err = %v, quería httpx.ErrNilDoer", err)
	}
}

func TestFuentesCtxCancelado(t *testing.T) {
	var peticiones int
	d := &doerGrabador{fn: func(req *http.Request) (*http.Response, error) {
		peticiones++
		<-req.Context().Done()
		return nil, req.Context().Err()
	}}
	a := New(d, DefaultBaseURL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fuentes, err := a.GetVideoSources(ctx, "animeav1:bleach:episode:1", scraper.SourceOpts{})
	if err == nil {
		t.Fatal("quería error con el contexto cancelado")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, quería context.Canceled", err)
	}
	if peticiones != 1 {
		t.Errorf("peticiones = %d, quería 1 (la petición se lanza y cancela)", peticiones)
	}
	if fuentes != nil {
		t.Errorf("fuentes = %#v, quería nil", fuentes)
	}
}

func TestFuentesCuerpoDemasiadoGrande(t *testing.T) {
	cuerpo := strings.Repeat("a", maxFuentesBytes+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, cuerpo)
	}))
	t.Cleanup(srv.Close)
	a := New(clientePrueba(), srv.URL)

	_, err := a.GetVideoSources(context.Background(), "animeav1:bleach:episode:1", scraper.SourceOpts{})
	if err == nil {
		t.Fatal("quería error con un cuerpo mayor que maxFuentesBytes")
	}
	if !strings.Contains(err.Error(), "máximo") {
		t.Errorf("err = %v, quería el límite de tamaño mencionado", err)
	}
}

func TestFuentesRespuestaInesperada(t *testing.T) {
	casos := []struct {
		nombre, cuerpo, contiene string
	}{
		{"type de error", `{"type":"error"}`, "type = \"error\""},
		{"sin nodo de página", `{"type":"data","nodes":[null]}`, "nodo de página"},
		{"JSON cortado", `{"type":"data","nodes":`, "JSON válido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			fuentes, err := fuentesConCuerpo(t, "animeav1:mi-slug:episode:1",
				"/media/mi-slug/1/__data.json", []byte(c.cuerpo))
			if err == nil {
				t.Fatalf("quería error con %s", c.nombre)
			}
			if fuentes != nil {
				t.Errorf("fuentes = %#v, quería nil cuando falla", fuentes)
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("err = %v, quería que mencionara %q", err, c.contiene)
			}
		})
	}
}
