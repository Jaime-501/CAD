package animeav1

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"miapp/internal/models"
	"miapp/internal/scraper/embeds"
)

// resolverEmbedsFalso implementa embeds.Resolver sin red (T61).
type resolverEmbedsFalso struct {
	nombre   string
	fn       func(string) bool
	resuelve func(context.Context, string, string) ([]models.VideoSource, error)
	llamadas int
}

var _ embeds.Resolver = (*resolverEmbedsFalso)(nil)

func (r *resolverEmbedsFalso) Name() string { return r.nombre }

func (r *resolverEmbedsFalso) CanHandle(url string) bool {
	if r.fn == nil {
		return true
	}
	return r.fn(url)
}

func (r *resolverEmbedsFalso) Resolve(ctx context.Context, url, referer string) ([]models.VideoSource, error) {
	r.llamadas++
	if r.resuelve == nil {
		return nil, errors.New("resolver falso sin medios")
	}
	return r.resuelve(ctx, url, referer)
}

func iframeEmbed(url, idioma string) models.VideoSource {
	return models.VideoSource{URL: url, Type: models.ResourceIframe, Server: "MP4Upload", Language: idioma}
}

func mp4Embed(url, idioma string) models.VideoSource {
	return models.VideoSource{URL: url, Type: models.ResourceMP4, Server: "MP4Upload", Language: idioma}
}

// TestEnriquecerEmbedsConservaElIdioma: sobre embeds SUB y DUB, cada mp4
// resuelto hereda el Language de su iframe, así SUB y DUB siguen siendo
// fuentes distintas aunque el resolver devuelva la misma URL (requisito 3).
func TestEnriquecerEmbedsConservaElIdioma(t *testing.T) {
	embedURL := "https://www.mp4upload.com/embed-aaaaaaaaaaaa.html"
	res := &resolverEmbedsFalso{
		nombre: "MP4Upload",
		resuelve: func(context.Context, string, string) ([]models.VideoSource, error) {
			// El resolver no conoce el idioma: lo pone el proveedor.
			return []models.VideoSource{{URL: "https://a2.mp4upload.com:183/d/tok/video.mp4", Type: models.ResourceMP4, Server: "MP4Upload", Referer: embedURL}}, nil
		},
	}
	a := &AnimeAV1{Embeds: []embeds.Resolver{res}}

	fuentes := a.enriquecerConEmbeds(context.Background(), []models.VideoSource{
		iframeEmbed(embedURL, models.LanguageSub),
		iframeEmbed(embedURL, models.LanguageDub),
	})

	if len(fuentes) != 4 {
		t.Fatalf("fuentes = %d, quería 4 (2 iframes + 2 mp4)", len(fuentes))
	}
	idiomas := map[string]string{}
	for _, f := range fuentes {
		if f.Type == models.ResourceMP4 {
			idiomas[f.Language] = f.URL
		}
	}
	if len(idiomas) != 2 || idiomas[models.LanguageSub] == "" || idiomas[models.LanguageDub] == "" {
		t.Errorf("idiomas de los mp4 = %v, quería uno SUB y otro DUB", idiomas)
	}
	// El Referer del embed lo aporta el resolver (lo verificará el proxy).
	for _, f := range fuentes {
		if f.Type == models.ResourceMP4 && f.Referer != embedURL {
			t.Errorf("Referer del mp4 = %q, quería el embed %q", f.Referer, embedURL)
		}
	}
}

// TestEnriquecerEmbedsFallaConservaIframes: un fallo del resolver no rompe
// el resto, se registra un aviso sin la URL del CDN y los iframes siguen.
func TestEnriquecerEmbedsFallaConservaIframes(t *testing.T) {
	previo := slog.Default()
	defer slog.SetDefault(previo)
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	res := &resolverEmbedsFalso{
		nombre: "MP4Upload",
		resuelve: func(context.Context, string, string) ([]models.VideoSource, error) {
			return nil, errors.New("httpx: MP4Upload/mp4upload https://www.mp4upload.com/embed-bbbbbbbbbbbb.html: status 404")
		},
	}
	a := &AnimeAV1{Embeds: []embeds.Resolver{res}}

	entrada := []models.VideoSource{
		{URL: "https://player.zilla-networks.com/m3u8/abc", Type: models.ResourceHLS, Server: "HLS", Language: models.LanguageSub},
		iframeEmbed("https://www.mp4upload.com/embed-bbbbbbbbbbbb.html", models.LanguageSub),
	}
	fuentes := a.enriquecerConEmbeds(context.Background(), entrada)

	if len(fuentes) != 2 {
		t.Fatalf("fuentes = %d, quería 2 (sin añadir nada)", len(fuentes))
	}
	log := buf.String()
	if !strings.Contains(log, "resolución del embed falló") {
		t.Errorf("no se registró el aviso de fallo: %q", log)
	}
	if strings.Contains(log, "/d/") || strings.Contains(log, "a2.mp4upload.com") {
		t.Errorf("el aviso no debe filtrar la URL del CDN resuelto: %q", log)
	}
}

// TestEnriquecerEmbedsDedup: un mp4 ya presente (mismo recurso + idioma) no
// se añade dos veces.
func TestEnriquecerEmbedsDedup(t *testing.T) {
	mp4 := mp4Embed("https://a2.mp4upload.com:183/d/tok/video.mp4", models.LanguageSub)
	res := &resolverEmbedsFalso{
		nombre: "MP4Upload",
		resuelve: func(context.Context, string, string) ([]models.VideoSource, error) {
			return []models.VideoSource{mp4}, nil
		},
	}
	a := &AnimeAV1{Embeds: []embeds.Resolver{res}}

	fuentes := a.enriquecerConEmbeds(context.Background(), []models.VideoSource{
		mp4,
		iframeEmbed("https://www.mp4upload.com/embed-aaaaaaaaaaaa.html", models.LanguageSub),
	})
	if len(fuentes) != 2 {
		t.Errorf("fuentes = %d, quería 2 (mp4 ya presente)", len(fuentes))
	}
}

// TestEnriquecerEmbedsCtxCancelado: el contexto del flujo se respeta.
func TestEnriquecerEmbedsCtxCancelado(t *testing.T) {
	res := &resolverEmbedsFalso{
		nombre: "MP4Upload",
		resuelve: func(ctx context.Context, _ string, _ string) ([]models.VideoSource, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return []models.VideoSource{{URL: "https://a2.mp4upload.com:183/d/tok/video.mp4", Type: models.ResourceMP4}}, nil
		},
	}
	a := &AnimeAV1{Embeds: []embeds.Resolver{res}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fuentes := a.enriquecerConEmbeds(ctx, []models.VideoSource{iframeEmbed("https://www.mp4upload.com/embed-aaaaaaaaaaaa.html", models.LanguageSub)})
	if len(fuentes) != 1 {
		t.Errorf("fuentes = %d, quería 1 con el ctx cancelado", len(fuentes))
	}
}

// TestNuevoAnimeAV1InyectaResolverPorDefecto: New deja un resolver de
// MP4Upload.
func TestNuevoAnimeAV1InyectaResolverPorDefecto(t *testing.T) {
	a := New(nil, "")
	if len(a.Embeds) != 1 {
		t.Fatalf("Embeds = %d, quería 1 por defecto", len(a.Embeds))
	}
	if !a.Embeds[0].CanHandle("https://www.mp4upload.com/embed-aaaaaaaaaaaa.html") {
		t.Errorf("el resolver por defecto no reconoce el embed de MP4Upload")
	}
}
