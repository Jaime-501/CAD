package models

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidEpisodeID señala un id de episodio con formato inválido.
// Se puede comprobar con errors.Is.
var ErrInvalidEpisodeID = errors.New("id de episodio inválido")

// NewEpisodeID construye el id estable de un episodio, definido antes de
// instanciarlo:
//
//	jkanime:<slug>:episode:<number>
//	jkanime:<slug>:movie
//	jkanime:<slug>:special:<number>
//
// Para KindMovie el number no se usa (nunca se fuerza :1): la película se
// identifica como "movie".
func NewEpisodeID(provider, slug string, kind EpisodeKind, number int) (string, error) {
	if provider == "" {
		return "", fmt.Errorf("%w: proveedor vacío", ErrInvalidEpisodeID)
	}
	if strings.ContainsRune(provider, ':') {
		return "", fmt.Errorf("%w: el proveedor %q no puede contener ':'", ErrInvalidEpisodeID, provider)
	}
	if slug == "" {
		return "", fmt.Errorf("%w: slug vacío", ErrInvalidEpisodeID)
	}
	if strings.ContainsRune(slug, ':') {
		return "", fmt.Errorf("%w: el slug %q no puede contener ':'", ErrInvalidEpisodeID, slug)
	}
	if number < 0 {
		return "", fmt.Errorf("%w: número de episodio negativo (%d)", ErrInvalidEpisodeID, number)
	}

	switch kind {
	case KindEpisode, KindSpecial:
		return fmt.Sprintf("%s:%s:%s:%d", provider, slug, kind, number), nil
	case KindMovie:
		return fmt.Sprintf("%s:%s:%s", provider, slug, kind), nil
	default:
		return "", fmt.Errorf("%w: tipo de episodio desconocido %q", ErrInvalidEpisodeID, kind)
	}
}

// ParseEpisodeID descompone un id generado por NewEpisodeID. Devuelve
// ErrInvalidEpisodeID si el formato no es válido o el número no es canónico.
func ParseEpisodeID(id string) (provider, slug string, kind EpisodeKind, number int, err error) {
	if id == "" {
		return "", "", "", 0, fmt.Errorf("%w: id vacío", ErrInvalidEpisodeID)
	}

	parts := strings.Split(id, ":")
	switch len(parts) {
	case 3:
		// Solo la película no lleva número: <provider>:<slug>:movie
		if parts[2] != string(KindMovie) {
			return "", "", "", 0, fmt.Errorf("%w: %q no es un tipo válido sin número", ErrInvalidEpisodeID, parts[2])
		}
		provider, slug, kind = parts[0], parts[1], KindMovie
	case 4:
		kind = EpisodeKind(parts[2])
		if kind != KindEpisode && kind != KindSpecial {
			return "", "", "", 0, fmt.Errorf("%w: tipo de episodio desconocido %q", ErrInvalidEpisodeID, parts[2])
		}
		n, convErr := strconv.Atoi(parts[3])
		if convErr != nil || n < 0 || strconv.Itoa(n) != parts[3] {
			return "", "", "", 0, fmt.Errorf("%w: número de episodio no válido %q", ErrInvalidEpisodeID, parts[3])
		}
		provider, slug, number = parts[0], parts[1], n
	default:
		return "", "", "", 0, fmt.Errorf("%w: se esperaban 3 o 4 partes, hay %d", ErrInvalidEpisodeID, len(parts))
	}

	if provider == "" {
		return "", "", "", 0, fmt.Errorf("%w: proveedor vacío", ErrInvalidEpisodeID)
	}
	if slug == "" {
		return "", "", "", 0, fmt.Errorf("%w: slug vacío", ErrInvalidEpisodeID)
	}
	return provider, slug, kind, number, nil
}
