package models

import (
	"errors"
	"testing"
)

func TestNewEpisodeID(t *testing.T) {
	casos := []struct {
		nombre   string
		provider string
		slug     string
		kind     EpisodeKind
		number   int
		queria   string
	}{
		{"serie", "jkanime", "bleach", KindEpisode, 5, "jkanime:bleach:episode:5"},
		{"serie en el episodio 0", "jkanime", "bleach", KindEpisode, 0, "jkanime:bleach:episode:0"},
		{"especial", "jkanime", "naruto", KindSpecial, 2, "jkanime:naruto:special:2"},
		{"pelicula", "jkanime", "your-name", KindMovie, 0, "jkanime:your-name:movie"},
		{"pelicula ignora el numero", "jkanime", "your-name", KindMovie, 1, "jkanime:your-name:movie"},
		{"otro proveedor", "animeflv", "bleach", KindEpisode, 1, "animeflv:bleach:episode:1"},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, err := NewEpisodeID(c.provider, c.slug, c.kind, c.number)
			if err != nil {
				t.Fatalf("NewEpisodeID(%q,%q,%q,%d): %v", c.provider, c.slug, c.kind, c.number, err)
			}
			if got != c.queria {
				t.Errorf("id = %q, quería %q", got, c.queria)
			}
		})
	}
}

func TestNewEpisodeIDInvalido(t *testing.T) {
	casos := []struct {
		nombre   string
		provider string
		slug     string
		kind     EpisodeKind
		number   int
	}{
		{"proveedor vacío", "", "bleach", KindEpisode, 1},
		{"proveedor con dos puntos", "jk:anime", "bleach", KindEpisode, 1},
		{"slug vacío", "jkanime", "", KindEpisode, 1},
		{"slug con dos puntos", "jkanime", "bleach:tybur", KindEpisode, 1},
		{"tipo desconocido", "jkanime", "bleach", EpisodeKind("ova"), 1},
		{"tipo vacío", "jkanime", "bleach", EpisodeKind(""), 1},
		{"numero negativo", "jkanime", "bleach", KindEpisode, -1},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, err := NewEpisodeID(c.provider, c.slug, c.kind, c.number)
			if err == nil {
				t.Fatalf("NewEpisodeID(%q,%q,%q,%d) no falló; devolvió %q", c.provider, c.slug, c.kind, c.number, got)
			}
			if got != "" {
				t.Errorf("id = %q, quería vacío en error", got)
			}
			if !errors.Is(err, ErrInvalidEpisodeID) {
				t.Errorf("error = %v, quería que envolviera ErrInvalidEpisodeID", err)
			}
		})
	}
}

func TestParseEpisodeID(t *testing.T) {
	casos := []struct {
		nombre   string
		id       string
		provider string
		slug     string
		kind     EpisodeKind
		number   int
	}{
		{"serie", "jkanime:bleach:episode:5", "jkanime", "bleach", KindEpisode, 5},
		{"episodio 0", "jkanime:bleach:episode:0", "jkanime", "bleach", KindEpisode, 0},
		{"especial", "jkanime:naruto:special:2", "jkanime", "naruto", KindSpecial, 2},
		{"pelicula", "jkanime:your-name:movie", "jkanime", "your-name", KindMovie, 0},
		{"slug con guiones", "jkanime:fullmetal-alchemist-brotherhood:episode:64", "jkanime", "fullmetal-alchemist-brotherhood", KindEpisode, 64},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			provider, slug, kind, number, err := ParseEpisodeID(c.id)
			if err != nil {
				t.Fatalf("ParseEpisodeID(%q): %v", c.id, err)
			}
			if provider != c.provider || slug != c.slug || kind != c.kind || number != c.number {
				t.Errorf("devolvió (%q,%q,%q,%d), quería (%q,%q,%q,%d)",
					provider, slug, kind, number, c.provider, c.slug, c.kind, c.number)
			}
		})
	}
}

func TestParseEpisodeIDInvalido(t *testing.T) {
	casos := []struct {
		nombre string
		id     string
	}{
		{"vacío", ""},
		{"sin separadores", "jkanime"},
		{"dos partes", "jkanime:bleach"},
		{"sin número", "jkanime:bleach:episode"},
		{"cinco partes", "jkanime:bleach:episode:1:extra"},
		{"slug vacío", "jkanime::episode:1"},
		{"proveedor vacío", ":bleach:episode:1"},
		{"tipo desconocido", "jkanime:bleach:ova:1"},
		{"tipo en mayúsculas", "jkanime:bleach:EPISODE:1"},
		{"pelicula con número", "jkanime:your-name:movie:1"},
		{"numero no numérico", "jkanime:bleach:episode:abc"},
		{"numero negativo", "jkanime:bleach:episode:-1"},
		{"numero no canónico", "jkanime:bleach:episode:01"},
		{"numero con signo", "jkanime:bleach:episode:+1"},
		{"tipo sin número pero no pelicula", "jkanime:bleach:special"},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			provider, slug, kind, number, err := ParseEpisodeID(c.id)
			if err == nil {
				t.Fatalf("ParseEpisodeID(%q) no falló; devolvió (%q,%q,%q,%d)", c.id, provider, slug, kind, number)
			}
			if !errors.Is(err, ErrInvalidEpisodeID) {
				t.Errorf("error = %v, quería que envolviera ErrInvalidEpisodeID", err)
			}
		})
	}
}

func TestEpisodeIDIdaYVuelta(t *testing.T) {
	casos := []struct {
		nombre   string
		provider string
		slug     string
		kind     EpisodeKind
		number   int
	}{
		{"serie", "jkanime", "bleach", KindEpisode, 137},
		{"especial", "jkanime", "bleach", KindSpecial, 0},
		{"pelicula", "jkanime", "your-name", KindMovie, 0},
		{"otro proveedor", "animeflv", "naruto-shippuden", KindEpisode, 1},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			id, err := NewEpisodeID(c.provider, c.slug, c.kind, c.number)
			if err != nil {
				t.Fatalf("NewEpisodeID: %v", err)
			}
			provider, slug, kind, number, err := ParseEpisodeID(id)
			if err != nil {
				t.Fatalf("ParseEpisodeID(%q): %v", id, err)
			}
			if provider != c.provider || slug != c.slug || kind != c.kind || number != c.number {
				t.Errorf("Parse(New(x)) = (%q,%q,%q,%d), quería (%q,%q,%q,%d)",
					provider, slug, kind, number, c.provider, c.slug, c.kind, c.number)
			}

			// Y de vuelta: el id reconstruido es idéntico al original.
			again, err := NewEpisodeID(provider, slug, kind, number)
			if err != nil {
				t.Fatalf("NewEpisodeID (reconstruido): %v", err)
			}
			if again != id {
				t.Errorf("id reconstruido = %q, quería %q", again, id)
			}
		})
	}
}
