package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// roundTrip serializa a JSON y vuelve a leer, comprobando que el valor
// sobrevive intacto (ida y vuelta).
func roundTrip(t *testing.T, in, out any) {
	t.Helper()

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("json.Marshal(%+v): %v", in, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", raw, err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("ida y vuelta altera el valor:\n entran: %#v\n salen: %#v\n json:  %s", in, out, raw)
	}
}

func TestAnimeIdaYVuelta(t *testing.T) {
	in := Anime{
		ID:       "bleach",
		Title:    "Bleach",
		CoverURL: "https://example.test/bleach.jpg",
		Provider: "jkanime",
	}

	var out Anime
	roundTrip(t, &in, &out)

	raw, _ := json.Marshal(in)
	for _, key := range []string{`"id"`, `"title"`, `"cover_url"`, `"provider"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("falta la etiqueta %s en %s", key, raw)
		}
	}
}

func TestAnimeDetailIdaYVuelta(t *testing.T) {
	in := AnimeDetail{
		ID:            "bleach",
		Title:         "Bleach",
		Provider:      "jkanime",
		Type:          "series",
		Status:        "finalizado",
		Genres:        []string{"accion", "sobrenatural"},
		Synopsis:      "Ichigo Kurosaki obtiene poderes de Shinigami.",
		Cover:         "https://example.test/bleach.jpg",
		TotalEpisodes: 366,
	}

	var out AnimeDetail
	roundTrip(t, &in, &out)

	raw, _ := json.Marshal(in)
	for _, key := range []string{`"type"`, `"status"`, `"genres"`, `"synopsis"`, `"cover"`, `"total_episodes"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("falta la etiqueta %s en %s", key, raw)
		}
	}
}

func TestEpisodeIdaYVuelta(t *testing.T) {
	casos := []struct {
		nombre string
		in     Episode
	}{
		{
			nombre: "serie",
			in: Episode{
				ID:        "jkanime:bleach:episode:1",
				Number:    1,
				Title:     "Episodio 1",
				Thumbnail: "https://example.test/1.jpg",
				Kind:      KindEpisode,
			},
		},
		{
			nombre: "pelicula",
			in: Episode{
				ID:    "jkanime:your-name:movie",
				Title: "Your Name",
				Kind:  KindMovie,
			},
		},
		{
			nombre: "especial",
			in: Episode{
				ID:     "jkanime:bleach:special:2",
				Number: 2,
				Title:  "Especial 2",
				Kind:   KindSpecial,
			},
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var out Episode
			roundTrip(t, &c.in, &out)

			raw, _ := json.Marshal(c.in)
			for _, key := range []string{`"id"`, `"number"`, `"title"`, `"thumbnail"`, `"kind"`} {
				if !strings.Contains(string(raw), key) {
					t.Errorf("falta la etiqueta %s en %s", key, raw)
				}
			}
			if got, want := out.Kind, c.in.Kind; got != want {
				t.Errorf("Kind = %q, quería %q", got, want)
			}
		})
	}
}

func TestResourceTypeValores(t *testing.T) {
	casos := map[ResourceType]string{
		ResourceHLS:    "hls",
		ResourceMP4:    "mp4",
		ResourceIframe: "iframe",
	}
	for got, want := range casos {
		if string(got) != want {
			t.Errorf("ResourceType = %q, quería %q", got, want)
		}
	}
}

func TestVideoSourceIdaYVuelta(t *testing.T) {
	in := VideoSource{
		URL:     "https://cdn.example.test/bleach-1.m3u8",
		Type:    ResourceHLS,
		Quality: "720p",
		Server:  "jkplayer",
		Headers: map[string]string{"Referer": "https://jkanime.net/"},
		Referer: "https://jkanime.net/",
		Subtitles: []Subtitle{
			{URL: "https://cdn.example.test/es.vtt", Lang: "es"},
			{URL: "https://cdn.example.test/en.vtt", Lang: "en"},
		},
	}

	var out VideoSource
	roundTrip(t, &in, &out)

	if len(out.Subtitles) != 2 || out.Subtitles[0].Lang != "es" {
		t.Errorf("Subtitles no sobrevive a la ida y vuelta: %+v", out.Subtitles)
	}
}

func TestVideoSourceOmiteSubtitlesVacios(t *testing.T) {
	sin := VideoSource{
		URL:     "https://cdn.example.test/video.mp4",
		Type:    ResourceMP4,
		Quality: "1080p",
		Server:  "mega",
		Headers: map[string]string{"User-Agent": "prueba"},
		Referer: "https://jkanime.net/",
	}

	raw, err := json.Marshal(sin)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Etiquetas exactas: si alguna cambia, el frontend deja de leer el campo.
	want := `{"url":"https://cdn.example.test/video.mp4","type":"mp4","quality":"1080p",` +
		`"server":"mega","headers":{"User-Agent":"prueba"},"referer":"https://jkanime.net/"}`
	if string(raw) != want {
		t.Errorf("JSON =\n %s\nquería\n %s", raw, want)
	}
	if strings.Contains(string(raw), "subtitles") {
		t.Errorf("Subtitles vacío no debería serializarse: %s", raw)
	}

	// Sin subtítulos la ida y vuelta sigue siendo exacta (nil -> ausente -> nil).
	var back VideoSource
	roundTrip(t, &sin, &back)

	// Con subtítulos sí aparece la etiqueta.
	con := VideoSource{URL: "https://cdn.example.test/v.m3u8", Subtitles: []Subtitle{{URL: "https://cdn.example.test/es.vtt", Lang: "es"}}}
	rawCon, err := json.Marshal(con)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(rawCon), `"subtitles":[{"url":"https://cdn.example.test/es.vtt","lang":"es"}]`) {
		t.Errorf("faltan los subtítulos en %s", rawCon)
	}
}

// TestVideoSourceIdioma cubre el campo estructurado Language (T33): las
// constantes exportadas, la etiqueta exacta del JSON y su omisión cuando la
// fuente no tiene idioma (JKAnime y cualquier proveedor que no lo declare).
func TestVideoSourceIdioma(t *testing.T) {
	if LanguageSub != "SUB" || LanguageDub != "DUB" {
		t.Errorf("LanguageSub/LanguageDub = %q/%q, quería SUB/DUB", LanguageSub, LanguageDub)
	}

	con := VideoSource{
		URL:      "https://player.zilla-networks.com/m3u8/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Type:     ResourceHLS,
		Server:   "HLS",
		Language: LanguageSub,
		Headers:  map[string]string{"Sec-Fetch-Site": "same-origin"},
	}
	var vuelta VideoSource
	roundTrip(t, &con, &vuelta)
	if vuelta.Language != LanguageSub {
		t.Errorf("Language = %q, quería %q", vuelta.Language, LanguageSub)
	}

	raw, err := json.Marshal(con)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	// El campo va tras server y con su propia etiqueta: el frontend lo lee
	// como source.language (por eso T34 debe regenerar wailsjs).
	if !strings.Contains(string(raw), `"server":"HLS","language":"SUB","headers"`) {
		t.Errorf("JSON = %s, quería language entre server y headers", raw)
	}

	// Sin idioma la clave desaparece: la ida y vuelta sigue siendo exacta.
	sin := VideoSource{URL: "https://cdn.example.test/v.m3u8", Type: ResourceHLS, Server: "jkplayer"}
	rawSin, err := json.Marshal(sin)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(rawSin), "language") {
		t.Errorf("Language vacío no debería serializarse: %s", rawSin)
	}
	var vueltaSin VideoSource
	roundTrip(t, &sin, &vueltaSin)
	if vueltaSin.Language != "" {
		t.Errorf("Language = %q, quería vacío", vueltaSin.Language)
	}
}

func TestWatchProgressIdaYVuelta(t *testing.T) {
	in := WatchProgress{
		AnimeID:       "bleach",
		EpisodeID:     "jkanime:bleach:episode:12",
		CurrentTime:   734.25,
		TotalDuration: 1420.5,
		IsWatched:     true,
	}

	var out WatchProgress
	roundTrip(t, &in, &out)

	raw, _ := json.Marshal(in)
	want := `{"anime_id":"bleach","episode_id":"jkanime:bleach:episode:12",` +
		`"current_time":734.25,"total_duration":1420.5,"is_watched":true}`
	if string(raw) != want {
		t.Errorf("JSON = %s, quería %s", raw, want)
	}
}

func TestEpisodeBatchIdaYVuelta(t *testing.T) {
	in := EpisodeBatch{
		AnimeID: "bleach",
		Batch:   2,
		Episodes: []Episode{
			{ID: "jkanime:bleach:episode:13", Number: 13, Title: "Episodio 13", Kind: KindEpisode},
			{ID: "jkanime:bleach:episode:14", Number: 14, Title: "Episodio 14", Kind: KindEpisode},
		},
		Done: true,
		Err:  "",
	}

	var out EpisodeBatch
	roundTrip(t, &in, &out)

	raw, _ := json.Marshal(in)
	for _, key := range []string{`"anime_id"`, `"batch"`, `"episodes"`, `"done"`, `"err"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("falta la etiqueta %s en %s", key, raw)
		}
	}
	if out.AnimeID != in.AnimeID || out.Batch != in.Batch || !out.Done {
		t.Errorf("EpisodeBatch no sobrevive: %+v", out)
	}
}
