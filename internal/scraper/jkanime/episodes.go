package jkanime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"miapp/internal/httpx"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

const (
	// maxEpisodesBytes limita el JSON de una página de episodios (10 MB),
	// igual que maxSearchBytes y maxDetailBytes.
	maxEpisodesBytes = 10 << 20

	// pausaPaginas es la espera entre peticiones de la goroutine de fondo
	// (~300 ms), para no golpear el endpoint AJAX de episodios.
	pausaPaginas = 300 * time.Millisecond
)

var _ scraper.EpisodeStreamer = (*JkAnime)(nil)

// episodiosPagina es la respuesta JSON de POST /ajax/episodes/<id>/<página>:
// una paginación de Laravel con 16 episodios por página en el sitio real.
// last_page es el campo que dice cuál es la última.
type episodiosPagina struct {
	CurrentPage int            `json:"current_page"`
	LastPage    int            `json:"last_page"`
	PerPage     int            `json:"per_page"`
	Total       int            `json:"total"`
	Data        []episodioJSON `json:"data"`
}

// episodioJSON es un episodio dentro de esa respuesta. El sitio enlaza cada
// uno a <base>/<slug>/<number>/; image es solo el nombre de archivo de la
// miniatura (sin dominio) y timestamp no se usa.
type episodioJSON struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Image  string `json:"image"`
}

// sesionEpisodios guarda el estado de un stream de episodios: el detalle ya
// descargado, el CSRF vigente (se refresca en el 419) y los ids ya emitidos,
// para no repetir ningún episodio dentro del mismo stream.
type sesionEpisodios struct {
	j       *JkAnime
	slug    string
	detalle models.AnimeDetail
	meta    detailMeta
	tipo    models.EpisodeKind
	vistos  map[string]struct{}
	// ultimaPagina sale de la respuesta de la página 1 (last_page).
	ultimaPagina int
	// episodios acumula todos los convertidos: es lo que devuelve
	// GetAnimeDetails. StreamAnimeDetails no lo lee (devuelve la página 1).
	episodios []models.Episode
}

// StreamAnimeDetails implementa scraper.EpisodeStreamer: entrega el detalle y
// la primera página de episodios de inmediato y descarga el resto (páginas
// 2..N) en una goroutine, emitiendo un lote por página con onBatch y, al
// final, el lote con Done: true (con Err vacío o con el fallo que lo paró).
//
// Si ctx se cancela, la goroutine termina sin emitir nada más —ni siquiera el
// lote final— y sin fugas. Un 419 (sesión caducada) refresca la página de
// detalle (cookie y CSRF nuevos) y reintenta la página fallida una sola vez;
// si vuelve a fallar, el lote final lleva un Err descriptivo.
func (j *JkAnime) StreamAnimeDetails(ctx context.Context, id string,
	onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
	s, primera, err := j.iniciarStream(ctx, id)
	if err != nil {
		return models.AnimeDetail{}, nil, err
	}
	go func() {
		_ = s.restantes(ctx, onBatch)
	}()
	return s.detalle, primera, nil
}

// getAnimeDetails es el wrapper de Provider.GetAnimeDetails: el mismo flujo
// que StreamAnimeDetails pero en la propia goroutine y sin emitir lotes
// (onBatch nil). Devuelve el detalle y TODOS los episodios cuando termina; si
// algo falla a mitad, devuelve el error sin detalle.
func (j *JkAnime) getAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	s, _, err := j.iniciarStream(ctx, id)
	if err != nil {
		return models.AnimeDetail{}, nil, err
	}
	if err := s.restantes(ctx, nil); err != nil {
		return models.AnimeDetail{}, nil, err
	}
	return s.detalle, s.episodios, nil
}

// iniciarStream descarga la página de detalle (GET <base>/<slug>/) y pide la
// primera página de episodios al AJAX. Un slug vacío no toca la red; los
// fallos HTTP llegan tal cual de httpx. Si el detalle no trae id numérico o
// CSRF no se puede pedir ningún episodio, y eso es un error explícito.
func (j *JkAnime) iniciarStream(ctx context.Context, id string) (*sesionEpisodios, []models.Episode, error) {
	slug := strings.Trim(strings.TrimSpace(id), "/")
	if slug == "" {
		return nil, nil, errSlugVacio
	}
	detalle, meta, err := j.fetchDetail(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	if meta.NumericID == "" {
		return nil, nil, fmt.Errorf("jkanime: la página de detalle de %q no trae id numérico", slug)
	}
	if meta.CSRFToken == "" {
		return nil, nil, fmt.Errorf("jkanime: la página de detalle de %q no trae token CSRF", slug)
	}
	s := &sesionEpisodios{
		j:       j,
		slug:    slug,
		detalle: detalle,
		meta:    meta,
		tipo:    tipoEpisodioDe(detalle),
		vistos:  make(map[string]struct{}),
	}
	pag, err := s.post(ctx, 1)
	if err != nil {
		return nil, nil, err
	}
	primera := s.convertir(pag.Data)
	s.episodios = append(s.episodios, primera...)
	s.ultimaPagina = pag.LastPage
	return s, primera, nil
}

// restantes recorre las páginas 2..N con la pausa de pausaPaginas entre
// peticiones, emitiendo un lote por página y el lote final Done. Si ctx se
// cancela (durante la espera o la petición) devuelve ctx.Err() sin emitir
// nada más; si una página falla, emite el lote final con Err y devuelve el
// error. onBatch nil (GetAnimeDetails) no emite nada y solo acumula.
func (s *sesionEpisodios) restantes(ctx context.Context, onBatch func(models.EpisodeBatch)) error {
	for pagina := 2; pagina <= s.ultimaPagina; pagina++ {
		if !esperar(ctx, pausaPaginas) {
			return ctx.Err()
		}
		pag, err := s.post(ctx, pagina)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.emitir(onBatch, pagina, nil, true, err.Error())
			return err
		}
		nuevos := s.convertir(pag.Data)
		s.episodios = append(s.episodios, nuevos...)
		s.emitir(onBatch, pagina, nuevos, false, "")
	}
	s.emitir(onBatch, s.ultimaPagina, nil, true, "")
	return nil
}

// post pide una página de episodios. Si el sitio responde 419 (sesión
// caducada o CSRF antiguo), refresca la página de detalle —cookie y token
// nuevos— y reintenta UNA sola vez; si vuelve a fallar, devuelve el error ya
// envuelto con la página que falló.
func (s *sesionEpisodios) post(ctx context.Context, pagina int) (episodiosPagina, error) {
	pag, err := s.j.postEpisodios(ctx, s.slug, s.meta, pagina)
	if err == nil || !errors.Is(err, httpx.ErrSessionExpired) || ctx.Err() != nil {
		return pag, err
	}
	_, fresca, ferr := s.j.fetchDetail(ctx, s.slug)
	if ferr != nil {
		return episodiosPagina{}, fmt.Errorf("jkanime: no pude refrescar la sesión tras el 419 de la página %d: %w", pagina, ferr)
	}
	if fresca.NumericID == "" || fresca.CSRFToken == "" {
		return episodiosPagina{}, fmt.Errorf("jkanime: el detalle refrescado de %q no trae id numérico ni CSRF (419 de la página %d)", s.slug, pagina)
	}
	s.meta = fresca
	pag, err = s.j.postEpisodios(ctx, s.slug, s.meta, pagina)
	if err != nil {
		return episodiosPagina{}, fmt.Errorf("jkanime: la página %d falló de nuevo tras refrescar la sesión: %w", pagina, err)
	}
	return pag, nil
}

// emitir entrega un lote al llamador si hay callback. Batch es la página que
// representa (1 es la que ya se devolvió; no se emite) y Done marca el final,
// con o sin Err.
func (s *sesionEpisodios) emitir(onBatch func(models.EpisodeBatch), pagina int,
	episodios []models.Episode, done bool, fallo string) {
	if onBatch == nil {
		return
	}
	onBatch(models.EpisodeBatch{
		AnimeID:  s.detalle.ID,
		Batch:    pagina,
		Episodes: episodios,
		Done:     done,
		Err:      fallo,
	})
}

// convertir traduce una página del JSON a modelos.Episode aplicando
// models.NewEpisodeID (T05): una entrada cuyo id no se puede construir (p.
// ej. número negativo) se descarta en lugar de tumbar el stream, y una
// entrada ya vista dentro de este stream no se vuelve a emitir.
func (s *sesionEpisodios) convertir(datos []episodioJSON) []models.Episode {
	episodios := make([]models.Episode, 0, len(datos))
	for _, d := range datos {
		id, err := models.NewEpisodeID(providerName, s.slug, s.tipo, d.Number)
		if err != nil {
			continue
		}
		if _, ya := s.vistos[id]; ya {
			continue
		}
		s.vistos[id] = struct{}{}
		episodios = append(episodios, models.Episode{
			ID:        id,
			Number:    d.Number,
			Title:     strings.TrimSpace(d.Title),
			Thumbnail: miniatura(s.detalle.Cover, d.Image),
			Kind:      s.tipo,
		})
	}
	return episodios
}

// postEpisodios hace el POST real del endpoint AJAX:
// cuerpo application/x-www-form-urlencoded con _token=<CSRF>, cabecera
// X-Requested-With (la manda jQuery) y Referer de la serie; las cookies viajan
// en el jar del doer. El método es obligatorio (un GET responde 405) y sin
// cookie de sesión o con token malo responde 419 (verificado en vivo).
func (j *JkAnime) postEpisodios(ctx context.Context, slug string, meta detailMeta, pagina int) (episodiosPagina, error) {
	cuerpo := url.Values{"_token": {meta.CSRFToken}}.Encode()
	resp, err := httpx.Post(ctx, j.Doer,
		j.siteURL()+"/ajax/episodes/"+meta.NumericID+"/"+strconv.Itoa(pagina),
		[]byte(cuerpo),
		httpx.ReqOptions{
			Provider:  providerName,
			Operation: "episodes",
			Referer:   j.siteURL() + "/" + url.PathEscape(slug) + "/",
			Header: http.Header{
				"Content-Type":     {"application/x-www-form-urlencoded; charset=UTF-8"},
				"X-Requested-With": {"XMLHttpRequest"},
			},
		})
	if err != nil {
		return episodiosPagina{}, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxEpisodesBytes+1))
	if err != nil {
		return episodiosPagina{}, err
	}
	if len(data) > maxEpisodesBytes {
		return episodiosPagina{}, fmt.Errorf("la página %d de episodios pesa %d bytes y el máximo es %d", pagina, len(data), maxEpisodesBytes)
	}
	var pag episodiosPagina
	if err := json.Unmarshal(data, &pag); err != nil {
		return episodiosPagina{}, fmt.Errorf("jkanime: la página %d de episodios no es JSON válido: %w", pagina, err)
	}
	return pag, nil
}

// tipoEpisodioDe decide el kind de cada episodio a partir del tipo del anime:
// las películas son movie (el id no lleva número) y el resto, episodios
// numerados. El endpoint AJAX no distingue especiales: el sitio enlaza igual
// a <slug>/<n>/.
func tipoEpisodioDe(detalle models.AnimeDetail) models.EpisodeKind {
	if detalle.Type == tipoPelicula {
		return models.KindMovie
	}
	return models.KindEpisode
}

// miniatura construye la URL de la miniatura como hace la web: el nombre de
// imagen del JSON se antepone al CDN de miniaturas, que se deriva de la
// portada del detalle sustituyendo /animes/image/ por
// /animes/video/image_thumb/. Sin imagen (o demasiado corta, que es el
// criterio del sitio) no hay miniatura.
func miniatura(cover, imagen string) string {
	imagen = strings.TrimSpace(imagen)
	if len(imagen) <= 10 {
		return ""
	}
	const patron = "/animes/image/"
	i := strings.Index(cover, patron)
	if i < 0 {
		return ""
	}
	return cover[:i] + "/animes/video/image_thumb/" + imagen
}

// esperar espera d sin saltarse la cancelación: devuelve false si el contexto
// se termina antes de tiempo.
func esperar(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
