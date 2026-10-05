// Package jkanime reúne el scraper de JKAnime: búsqueda, detalle,
// episodios y extractores de vídeo.
package jkanime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"miapp/internal/httpx"
)

// DefaultExceptionsURL es el mapa remoto de excepciones publicado en el
// repositorio Zhuchii/ani-es: un JSON plano string→string donde la clave es
// la URL de jkanime sin barra final (slug roto) y el valor la URL correcta
// con barra final.
const DefaultExceptionsURL = "https://raw.githubusercontent.com/Zhuchii/ani-es/refs/heads/main/excepciones.json"

// exceptionsTimeout limita la descarga del mapa: aunque el doer no tenga
// timeout propio, la carga nunca dura más que esto.
const exceptionsTimeout = 5 * time.Second

// maxExceptionsBytes es el tamaño máximo aceptado del mapa (1 MB). Una
// respuesta mayor se descarta entera, sin siquiera intentar parsearla.
const maxExceptionsBytes = 1 << 20

// ResolvedAnimeURL es el resultado de Resolve: la URL utilizable y si salió
// del mapa de excepciones.
type ResolvedAnimeURL struct {
	URL           string `json:"url"`
	UsedException bool   `json:"used_exception"`
}

// ExceptionResolver descarga el mapa remoto de excepciones con
// httpx.HTTPDoer y lo conserva en memoria durante la ejecución (protegido
// con mutex). El valor cero es utilizable: con URL vacía usa
// DefaultExceptionsURL y, si Doer es nil, no se toca la red y se sigue sin
// excepciones. Resolve nunca devuelve error: si el recurso remoto falla se
// continúa sin excepciones y el siguiente Resolve reintenta la descarga.
type ExceptionResolver struct {
	// URL del mapa remoto; vacía → DefaultExceptionsURL.
	URL string
	// Doer con el que se descarga el mapa (httpx.HTTPDoer; *http.Client lo
	// cumple).
	Doer httpx.HTTPDoer

	mu      sync.Mutex
	loaded  bool
	entries map[string]string
}

// Resolve devuelve la URL definitiva de slug. La coincidencia con el mapa es
// exacta tras normalizar (minúsculas y sin barra final): si hay entrada, se
// devuelve la URL correcta del mapa con UsedException=true; si no, el slug
// tal cual con UsedException=false. No se deduce ninguna regla de truncado:
// solo se consulta el mapa.
func (r *ExceptionResolver) Resolve(ctx context.Context, slug string) ResolvedAnimeURL {
	if fixed, ok := r.load(ctx)[normalizeExceptionKey(slug)]; ok {
		return ResolvedAnimeURL{URL: fixed, UsedException: true}
	}
	return ResolvedAnimeURL{URL: slug}
}

// load devuelve el mapa ya cargado, descargándolo si hace falta. La descarga
// ocurre bajo el mutex para que varios Resolve concurrentes solo disparen una
// petición; un fallo no se cachea (se reintenta en la siguiente llamada).
func (r *ExceptionResolver) load(ctx context.Context) map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loaded {
		return r.entries
	}
	entries, err := r.fetch(ctx)
	if err != nil {
		slog.Warn("jkanime: no se pudo cargar el mapa de excepciones; se continúa sin excepciones", "err", err)
		return nil
	}
	r.entries = entries
	r.loaded = true
	return entries
}

// fetch descarga y parsea el mapa con un timeout corto. El tamaño de la
// respuesta está limitado a maxExceptionsBytes y el esquema se valida
// entrada a entrada.
func (r *ExceptionResolver) fetch(ctx context.Context) (map[string]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, exceptionsTimeout)
	defer cancel()
	source := r.URL
	if source == "" {
		source = DefaultExceptionsURL
	}
	resp, err := httpx.Get(ctx, r.Doer, source, httpx.ReqOptions{
		Provider:  "jkanime",
		Operation: "exceptions",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxExceptionsBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leyendo el mapa de excepciones: %w", err)
	}
	if len(data) > maxExceptionsBytes {
		return nil, fmt.Errorf("el mapa de excepciones pesa %d bytes y el máximo es %d", len(data), maxExceptionsBytes)
	}
	return parseExceptions(data)
}

// parseExceptions valida el esquema: un objeto plano string→string cuyas
// claves y valores sean URLs http/https con clave no vacía. Las entradas
// inválidas se descartan sin invalidar el resto del mapa.
func parseExceptions(data []byte) (map[string]string, error) {
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("el mapa de excepciones no es JSON válido: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("el mapa de excepciones no es un objeto JSON")
	}
	entries := make(map[string]string, len(raw))
	for k, v := range raw {
		key := normalizeExceptionKey(k)
		value := strings.TrimSpace(v)
		if key == "" || !isHTTPURL(key) || !isHTTPURL(value) {
			continue
		}
		entries[key] = value
	}
	return entries, nil
}

// normalizeExceptionKey normaliza una URL para la coincidencia exacta: sin
// espacios, en minúsculas y sin barra final. Se aplica igual a las claves
// del mapa y a la URL entrante.
func normalizeExceptionKey(s string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(s), "/"))
}

// isHTTPURL comprueba que la cadena sea una URL absoluta http o https.
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	return scheme == "http" || scheme == "https"
}
