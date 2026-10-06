package extractors

import (
	"testing"

	"miapp/internal/models"
)

func TestSelectBestSourceTablas(t *testing.T) {
	hls := models.VideoSource{URL: "https://cdn.example.test/v.m3u8", Type: models.ResourceHLS, Server: NameJKPlayer}
	mp4 := models.VideoSource{URL: "https://cdn.example.test/v.mp4", Type: models.ResourceMP4, Server: "Mediafire"}
	iframe := models.VideoSource{URL: "https://voe.sx/e/abc", Type: models.ResourceIframe, Server: "VOE"}
	sinTipo := models.VideoSource{URL: "https://cdn.example.test/recurso", Server: "otro"}
	sinURL := models.VideoSource{URL: "", Type: models.ResourceHLS, Server: NameJKPlayer}

	casos := []struct {
		nombre  string
		fuentes []models.VideoSource
		queria  string
		ok      bool
	}{
		{"lista vacía", nil, "", false},
		{"sin ninguna URL reproducible", []models.VideoSource{sinURL}, "", false},
		{"solo HLS", []models.VideoSource{hls}, hls.URL, true},
		{"solo MP4", []models.VideoSource{mp4}, mp4.URL, true},
		{"solo iframe", []models.VideoSource{iframe}, iframe.URL, true},
		{"HLS gana a MP4", []models.VideoSource{mp4, hls}, hls.URL, true},
		{"HLS gana al iframe", []models.VideoSource{iframe, hls}, hls.URL, true},
		{"MP4 gana al iframe", []models.VideoSource{iframe, mp4}, mp4.URL, true},
		{"los tres tipos", []models.VideoSource{iframe, mp4, hls}, hls.URL, true},
		{"el iframe nunca se autoelige con MP4 al lado", []models.VideoSource{mp4, iframe, mp4}, mp4.URL, true},
		{"tipo desconocido por detrás del iframe", []models.VideoSource{sinTipo, iframe}, iframe.URL, true},
		{"tipo desconocido es el último recurso", []models.VideoSource{sinTipo}, sinTipo.URL, true},
		{"ignora las fuentes sin URL", []models.VideoSource{sinURL, mp4}, mp4.URL, true},
		{"las tres vacías no valen", []models.VideoSource{sinURL, {URL: "   "}}, "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, ok := SelectBestSource(c.fuentes)
			if ok != c.ok {
				t.Fatalf("ok = %v, quería %v", ok, c.ok)
			}
			if !c.ok {
				if got.URL != "" {
					t.Errorf("fuente = %+v, quería la cero", got)
				}
				return
			}
			if got.URL != c.queria {
				t.Errorf("URL = %q, quería %q", got.URL, c.queria)
			}
		})
	}
}

func TestSelectBestSourceDesempatePorCalidad(t *testing.T) {
	campo := func(url, quality string) models.VideoSource {
		return models.VideoSource{URL: url, Type: models.ResourceHLS, Quality: quality}
	}
	casos := []struct {
		nombre  string
		fuentes []models.VideoSource
		queria  string
	}{
		{
			"gana la mayor calidad",
			[]models.VideoSource{campo("https://cdn.example.test/720.m3u8", "720p"), campo("https://cdn.example.test/1080.m3u8", "1080p")},
			"https://cdn.example.test/1080.m3u8",
		},
		{
			"a igualdad gana la primera (orden canónico)",
			[]models.VideoSource{campo("https://cdn.example.test/a.m3u8", "720p"), campo("https://cdn.example.test/b.m3u8", "720p")},
			"https://cdn.example.test/a.m3u8",
		},
		{
			"calidad sin número no desempata",
			[]models.VideoSource{campo("https://cdn.example.test/hd.m3u8", "HD"), campo("https://cdn.example.test/480.m3u8", "480p")},
			"https://cdn.example.test/480.m3u8",
		},
		{
			"las dos sin calidad mantiene la primera",
			[]models.VideoSource{campo("https://cdn.example.test/a.m3u8", ""), campo("https://cdn.example.test/b.m3u8", "")},
			"https://cdn.example.test/a.m3u8",
		},
		{
			"la calidad solo desempata dentro del mismo rango",
			[]models.VideoSource{
				{URL: "https://cdn.example.test/1080.m3u8", Type: models.ResourceMP4, Quality: "1080p"},
				{URL: "https://cdn.example.test/hls.m3u8", Type: models.ResourceHLS, Quality: "360p"},
			},
			"https://cdn.example.test/hls.m3u8",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, ok := SelectBestSource(c.fuentes)
			if !ok {
				t.Fatal("ok = false, quería una fuente")
			}
			if got.URL != c.queria {
				t.Errorf("URL = %q, quería %q", got.URL, c.queria)
			}
		})
	}
}
