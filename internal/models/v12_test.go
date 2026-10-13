package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestFavoriteKey (T40): la clave es proveedor + ":" + anime_id; el mismo
// slug con otro proveedor es otra clave.
func TestFavoriteKey(t *testing.T) {
	if got := (Favorite{Provider: "animeav1", AnimeID: "bleach"}).Key(); got != "animeav1:bleach" {
		t.Errorf("Key() = %q, quería %q", got, "animeav1:bleach")
	}
	otro := Favorite{Provider: "jkanime", AnimeID: "bleach"}.Key()
	mismo := Favorite{Provider: "animeav1", AnimeID: "bleach"}.Key()
	if otro == mismo {
		t.Errorf("claves iguales para proveedores distintos: %q", otro)
	}
}

// TestFavoriteJSONTags (T40): etiquetas exactas de Favorite.
func TestFavoriteJSONTags(t *testing.T) {
	f := Favorite{
		Provider: "animeav1",
		AnimeID:  "bleach",
		Title:    "Bleach",
		CoverURL: "https://cdn.example.test/bleach.jpg",
		AddedAt:  1728000000,
	}

	want := []string{"added_at", "anime_id", "cover_url", "provider", "title"}
	if got := clavesJSON(t, f); !reflect.DeepEqual(got, want) {
		t.Errorf("claves del JSON = %v, quería %v", got, want)
	}
}

// TestHomeSectionJSONTags (T40): etiquetas exactas de HomeSection; un Items
// con un Anime serializa su cover_url.
func TestHomeSectionJSONTags(t *testing.T) {
	s := HomeSection{
		ID:    "emision",
		Title: "En emisión",
		Items: []Anime{
			{ID: "bleach", Title: "Bleach", CoverURL: "https://cdn.example.test/bleach.jpg", Provider: "jkanime"},
		},
	}

	want := []string{"id", "items", "title"}
	if got := clavesJSON(t, s); !reflect.DeepEqual(got, want) {
		t.Errorf("claves del JSON = %v, quería %v", got, want)
	}

	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal(%+v): %v", s, err)
	}
	if !strings.Contains(string(raw), `"cover_url":"https://cdn.example.test/bleach.jpg"`) {
		t.Errorf("el item no serializa cover_url: %s", raw)
	}

	var vuelta HomeSection
	roundTrip(t, &s, &vuelta)
}

// TestResumeInfoJSONTags (T40): etiquetas exactas de ResumeInfo.
func TestResumeInfoJSONTags(t *testing.T) {
	r := ResumeInfo{
		Provider:      "jkanime",
		AnimeID:       "bleach",
		Action:        ResumeActionResume,
		EpisodeID:     "jkanime:bleach:episode:12",
		EpisodeNumber: 12,
		StartAt:       734.25,
	}

	want := []string{"action", "anime_id", "episode_id", "episode_number", "provider", "start_at"}
	if got := clavesJSON(t, r); !reflect.DeepEqual(got, want) {
		t.Errorf("claves del JSON = %v, quería %v", got, want)
	}
}

// TestNextEpisodeJSONTags (T40): etiquetas exactas de NextEpisode.
func TestNextEpisodeJSONTags(t *testing.T) {
	n := NextEpisode{Found: true, EpisodeID: "jkanime:bleach:episode:13", Number: 13}

	want := []string{"episode_id", "found", "number"}
	if got := clavesJSON(t, n); !reflect.DeepEqual(got, want) {
		t.Errorf("claves del JSON = %v, quería %v", got, want)
	}
}

// TestResumeActionConstantes (T40): los cuatro literales de la tabla "retomar".
func TestResumeActionConstantes(t *testing.T) {
	casos := map[string]string{
		ResumeActionResume:    "resume",
		ResumeActionNext:      "next",
		ResumeActionCompleted: "completed",
		ResumeActionUpToDate:  "up_to_date",
	}
	if len(casos) != 4 {
		t.Errorf("constantes = %d, quería 4", len(casos))
	}
	for got, want := range casos {
		if got != want {
			t.Errorf("constante = %q, quería %q", got, want)
		}
	}
}
