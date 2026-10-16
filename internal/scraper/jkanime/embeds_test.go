package jkanime

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"miapp/internal/models"
	"miapp/internal/scraper/embeds"
	"miapp/internal/scraper/jkanime/extractors"
)

// resolverFalso implementa embeds.Resolver sin red: decide con fn y
// resuelve con resuelve (T61).
type resolverFalso struct {
	nombre    string
	fn        func(string) bool
	resuelve  func(context.Context, string, string) ([]models.VideoSource, error)
	llamadas  int
	refereres []string
}

var _ embeds.Resolver = (*resolverFalso)(nil)

func (r *resolverFalso) Name() string { return r.nombre }

func (r *resolverFalso) CanHandle(url string) bool {
	if r.fn == nil {
		return true
	}
	return r.fn(url)
}

func (r *resolverFalso) Resolve(ctx context.Context, url, referer string) ([]models.VideoSource, error) {
	r.llamadas++
	r.refereres = append(r.refereres, referer)
	if r.resuelve == nil {
		return nil, errors.New("resolver falso sin medios")
	}
	return r.resuelve(ctx, url, referer)
}

// mp4De construye la fuente mp4 con el token en el path (forma real del
// CDN de MP4Upload).
func mp4De(token string) models.VideoSource {
	return models.VideoSource{
		URL:     "https://a2.mp4upload.com:183/d/" + token + "/video.mp4",
		Type:    models.ResourceMP4,
		Server:  "MP4Upload",
		Referer: "https://www.mp4upload.com/embed-" + token + ".html",
	}
}

func iframeMP4(token string) models.VideoSource {
	return models.VideoSource{
		URL:    "https://www.mp4upload.com/embed-" + token + ".html",
		Type:   models.ResourceIframe,
		Server: "MP4Upload",
	}
}

// TestEnriquecerConEmbedsAñadeMP4ConservaIframe: por cada iframe reconocido
// se añade su mp4 directo y el iframe se conserva como respaldo; el resto
// de fuentes no se toca y SelectBestSource prefiere el mp4 al iframe.
func TestEnriquecerConEmbedsAñadeMP4ConservaIframe(t *testing.T) {
	iframe := iframeMP4("aaaaaaaaaaaa")
	res := &resolverFalso{
		nombre: "MP4Upload",
		resuelve: func(context.Context, string, string) ([]models.VideoSource, error) {
			return []models.VideoSource{mp4De("aaaaaaaaaaaa")}, nil
		},
	}
	j := &JkAnime{Embeds: []embeds.Resolver{res}}

	// Sin T61 el iframe era la única (y por tanto "mejor") fuente.
	antes, ok := extractors.SelectBestSource([]models.VideoSource{iframe})
	if !ok || antes.Type != models.ResourceIframe {
		t.Fatalf("base = %+v (ok=%v), esperaba solo el iframe", antes, ok)
	}

	fuentes := j.enriquecerConEmbeds(context.Background(), []models.VideoSource{iframe})

	if len(fuentes) != 2 {
		t.Fatalf("fuentes = %d, quería 2 (iframe + mp4)", len(fuentes))
	}
	if fuentes[0].Type != models.ResourceIframe || fuentes[1].Type != models.ResourceMP4 {
		t.Errorf("tipos = %s,%s, quería iframe,mp4", fuentes[0].Type, fuentes[1].Type)
	}
	if res.llamadas != 1 {
		t.Errorf("llamadas = %d, quería 1 (solo el iframe)", res.llamadas)
	}
	if mejor, ok := extractors.SelectBestSource(fuentes); !ok || mejor.Type != models.ResourceMP4 {
		t.Errorf("SelectBestSource = %+v (ok=%v), quería el mp4 resuelto", mejor, ok)
	}
}

// TestEnriquecerEmbedsNoReconoceNada: un iframe que ningún resolver
// reconoce no dispara ninguna resolución ni añade fuentes.
func TestEnriquecerEmbedsNoReconoceNada(t *testing.T) {
	res := &resolverFalso{nombre: "MP4Upload", fn: func(string) bool { return false }}
	j := &JkAnime{Embeds: []embeds.Resolver{res}}

	fuentes := j.enriquecerConEmbeds(context.Background(), []models.VideoSource{iframeMP4("aaaaaaaaaaaa")})
	if len(fuentes) != 1 || res.llamadas != 0 {
		t.Errorf("fuentes = %d, llamadas = %d; quería 1 y 0", len(fuentes), res.llamadas)
	}
}

// TestEnriquecerEmbedsFallaConservaIframeYAvisa: un Resolve que falla no
// rompe el resto, se registra un aviso y el iframe sobrevive (requisito 1).
// El fallo llega al pedir el EMBED, no el vídeo: ningún token del CDN
// (/d/<token>/) puede acabar en el log.
func TestEnriquecerEmbedsFallaConservaIframeYAvisa(t *testing.T) {
	previo := slog.Default()
	defer slog.SetDefault(previo)
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	iframe := iframeMP4("bbbbbbbbbbbb")
	res := &resolverFalso{
		nombre: "MP4Upload",
		resuelve: func(context.Context, string, string) ([]models.VideoSource, error) {
			return nil, errors.New("httpx: MP4Upload/mp4upload https://www.mp4upload.com/embed-bbbbbbbbbbbb.html: status 404")
		},
	}
	j := &JkAnime{Embeds: []embeds.Resolver{res}}

	fuentes := j.enriquecerConEmbeds(context.Background(), []models.VideoSource{iframe})

	if len(fuentes) != 1 || fuentes[0].Type != models.ResourceIframe {
		t.Fatalf("fuentes = %+v, quería solo el iframe conservado", fuentes)
	}
	log := buf.String()
	if !strings.Contains(log, "resolución del embed falló") {
		t.Errorf("no se registró el aviso de fallo: %q", log)
	}
	if strings.Contains(log, "/d/") || strings.Contains(log, "a2.mp4upload.com") {
		t.Errorf("el aviso no debe filtrar la URL del CDN resuelto: %q", log)
	}
}

// TestEnriquecerEmbedsDedup: un mp4 que ya estaba en la lista no se añade
// dos veces (requisito 2, dedup).
func TestEnriquecerEmbedsDedup(t *testing.T) {
	mp4 := mp4De("cccccccccccc")
	res := &resolverFalso{
		nombre: "MP4Upload",
		resuelve: func(context.Context, string, string) ([]models.VideoSource, error) {
			return []models.VideoSource{mp4}, nil
		},
	}
	j := &JkAnime{Embeds: []embeds.Resolver{res}}

	fuentes := j.enriquecerConEmbeds(context.Background(), []models.VideoSource{mp4, iframeMP4("cccccccccccc")})
	if len(fuentes) != 2 {
		t.Errorf("fuentes = %d, quería 2 (mp4 ya presente, no duplicado)", len(fuentes))
	}
}

// TestEnriquecerEmbedsCtxCancelado: el contexto del flujo se respeta; con
// el contexto cancelado el resolver falla y el iframe se conserva.
func TestEnriquecerEmbedsCtxCancelado(t *testing.T) {
	res := &resolverFalso{
		nombre: "MP4Upload",
		resuelve: func(ctx context.Context, _ string, _ string) ([]models.VideoSource, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return []models.VideoSource{mp4De("dddddddddddd")}, nil
		},
	}
	j := &JkAnime{Embeds: []embeds.Resolver{res}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fuentes := j.enriquecerConEmbeds(ctx, []models.VideoSource{iframeMP4("dddddddddddd")})
	if len(fuentes) != 1 || fuentes[0].Type != models.ResourceIframe {
		t.Errorf("fuentes = %+v, quería solo el iframe con el ctx cancelado", fuentes)
	}
}

// TestEnriquecerEmbedsSinResolvers: sin resolvers inyectados no se toca
// nada (ni se sale a la red).
func TestEnriquecerEmbedsSinResolvers(t *testing.T) {
	j := &JkAnime{Embeds: nil}
	fuentes := j.enriquecerConEmbeds(context.Background(), []models.VideoSource{iframeMP4("eeeeeeeeeeee")})
	if len(fuentes) != 1 {
		t.Errorf("fuentes = %d, quería 1 sin resolvers", len(fuentes))
	}
}

// TestNuevoJkAnimeInyectaResolverPorDefecto: New deja un resolver de
// MP4Upload (para que el flujo real resuelva sin cableado extra).
func TestNuevoJkAnimeInyectaResolverPorDefecto(t *testing.T) {
	j := New(nil, "", nil)
	if len(j.Embeds) != 1 {
		t.Fatalf("Embeds = %d, quería 1 por defecto", len(j.Embeds))
	}
	if !j.Embeds[0].CanHandle("https://www.mp4upload.com/embed-aaaaaaaaaaaa.html") {
		t.Errorf("el resolver por defecto no reconoce el embed de MP4Upload")
	}
}
