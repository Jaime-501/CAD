package models

import (
	"encoding/json"
	"reflect"
	"sort"
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
	// T40 añadió cinco campos a WatchProgress (sin omitempty): el JSON de un
	// registro con los nuevos vacíos los incluye en cero.
	want := `{"anime_id":"bleach","episode_id":"jkanime:bleach:episode:12",` +
		`"current_time":734.25,"total_duration":1420.5,"is_watched":true,` +
		`"provider":"","anime_title":"","cover_url":"","episode_number":0,"updated_at":0}`
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

// clavesJSON devuelve ordenadas las claves del objeto JSON que produce v.
func clavesJSON(t *testing.T, v any) []string {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal(%+v): %v", v, err)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(raw, &objeto); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", raw, err)
	}
	claves := make([]string, 0, len(objeto))
	for clave := range objeto {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	return claves
}

// TestWatchProgressJSONAntiguoSinCamposNuevos (T40): un historial.json escrito
// antes de ampliar el modelo sigue cargando; los campos nuevos quedan en cero
// y ProviderName() se infiere del prefijo de EpisodeID.
func TestWatchProgressJSONAntiguoSinCamposNuevos(t *testing.T) {
	crudo := `{"anime_id":"bleach","episode_id":"jkanime:bleach:episode:1","current_time":10,"total_duration":1400,"is_watched":false}`

	var p WatchProgress
	if err := json.Unmarshal([]byte(crudo), &p); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", crudo, err)
	}

	if p.AnimeID != "bleach" || p.EpisodeID != "jkanime:bleach:episode:1" {
		t.Errorf("los campos antiguos cambian: %+v", p)
	}
	if p.UpdatedAt != 0 {
		t.Errorf("UpdatedAt = %d, quería 0 (registro antiguo sin fecha)", p.UpdatedAt)
	}
	if p.Provider != "" {
		t.Errorf("Provider = %q, quería vacío", p.Provider)
	}
	if got := p.ProviderName(); got != "jkanime" {
		t.Errorf("ProviderName() = %q, quería %q", got, "jkanime")
	}
}

// TestWatchProgressJSONRoundTripCamposNuevos (T40): los cinco campos nuevos
// sobreviven intactos a la ida y vuelta por JSON.
func TestWatchProgressJSONRoundTripCamposNuevos(t *testing.T) {
	in := WatchProgress{
		AnimeID:       "bleach",
		EpisodeID:     "animeav1:bleach:episode:12",
		CurrentTime:   734.25,
		TotalDuration: 1420.5,
		IsWatched:     true,
		Provider:      "animeav1",
		AnimeTitle:    "Bleach",
		CoverURL:      "https://cdn.example.test/bleach.jpg",
		EpisodeNumber: 12,
		UpdatedAt:     1728000000,
	}

	var out WatchProgress
	roundTrip(t, &in, &out)

	if out.Provider != in.Provider || out.AnimeTitle != in.AnimeTitle ||
		out.CoverURL != in.CoverURL || out.EpisodeNumber != in.EpisodeNumber ||
		out.UpdatedAt != in.UpdatedAt {
		t.Errorf("los campos nuevos no sobreviven:\n entran: %#v\n salen: %#v", in, out)
	}
}

// TestWatchProgressProviderName (T40): tabla de la inferencia del proveedor.
func TestWatchProgressProviderName(t *testing.T) {
	casos := []struct {
		nombre string
		p      WatchProgress
		want   string
	}{
		{
			nombre: "Provider gana sobre el prefijo",
			p:      WatchProgress{Provider: "animeav1", EpisodeID: "jkanime:bleach:episode:1"},
			want:   "animeav1",
		},
		{
			nombre: "sin Provider usa el prefijo de EpisodeID",
			p:      WatchProgress{EpisodeID: "jkanime:bleach:episode:1"},
			want:   "jkanime",
		},
		{
			nombre: "sin EpisodeID ni Provider",
			p:      WatchProgress{},
			want:   "",
		},
		{
			nombre: "EpisodeID sin dos puntos",
			p:      WatchProgress{EpisodeID: "bleach-1"},
			want:   "",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := c.p.ProviderName(); got != c.want {
				t.Errorf("ProviderName() = %q, quería %q", got, c.want)
			}
		})
	}
}

// TestWatchProgressTagsJSON (T40): el JSON lleva exactamente las claves
// antiguas más las cinco nuevas (sin omitempty).
func TestWatchProgressTagsJSON(t *testing.T) {
	p := WatchProgress{
		AnimeID:       "bleach",
		EpisodeID:     "jkanime:bleach:episode:12",
		CurrentTime:   734.25,
		TotalDuration: 1420.5,
		IsWatched:     true,
		Provider:      "jkanime",
		AnimeTitle:    "Bleach",
		CoverURL:      "https://cdn.example.test/bleach.jpg",
		EpisodeNumber: 12,
		UpdatedAt:     1728000000,
	}

	want := []string{
		"anime_id", "anime_title", "cover_url", "current_time",
		"episode_id", "episode_number", "is_watched", "provider",
		"total_duration", "updated_at",
	}
	if got := clavesJSON(t, p); !reflect.DeepEqual(got, want) {
		t.Errorf("claves del JSON = %v, quería %v", got, want)
	}
}
