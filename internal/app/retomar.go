// Retomar y siguiente episodio (T46): la lógica que decide qué hace el clic
// sobre una tarjeta del historial y cuál es el episodio siguiente a uno dado.
// El frontend no calcula nada de esto (v1.2, tabla de acciones del plan).
//
// La lista COMPLETA de episodios se obtiene —decisión de diseño cerrada en la
// ficha— llamando a Detalle y esperando a que la entrada de caché quede
// completa: los efectos de Detalle (cancelar el stream de otro anime y emitir
// episodes:*) son aceptables aquí porque el anime por el que se pregunta es
// siempre el que la UI va a abrir a continuación. Nunca se devuelve una lista
// parcial: con una lista a medias, "no hay siguiente" sería un falso
// up_to_date/completed.
package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"miapp/internal/cache"
	"miapp/internal/models"
	"miapp/internal/scraper"
)

// ErrSinHistorial es lo que ve la UI cuando se pide retomar (o buscar el
// siguiente episodio en el historial) de un anime que no tiene NINGÚN
// registro para ese (proveedor, anime). No es un fallo de red ni del
// almacén: no hay nada que retomar.
var ErrSinHistorial = errors.New("No hay registros de ese anime en el historial")

var (
	// intervaloEsperaEpisodios es la cadencia con la que episodiosCompletos
	// vuelve a mirar la caché a la espera de que el stream complete la
	// lista. Es una variable para que los tests la reduzcan.
	intervaloEsperaEpisodios = 50 * time.Millisecond
	// esperaMaximaEpisodios acota esa espera: pasado el tiempo se devuelve
	// un error en español. Solo gobierna la ESPERA: el contexto que recibe
	// Detalle es el del llamador y no se deriva con timeout, porque el
	// contexto del stream nace de él y cancelarlo mataría los lotes que el
	// frontend necesita después.
	esperaMaximaEpisodios = 60 * time.Second
)

// errListaEpisodios es lo que se devuelve cuando la lista completa de
// episodios no pudo completarse (el stream falló, otro anime lo canceló, la
// entrada caducó o se agotó la espera). El mensaje es el que ve la UI; nunca
// se decide completed/up_to_date ni se devuelve la lista parcial con este
// error.
var errListaEpisodios = errors.New("no se pudo completar la lista de episodios")

// Retomar decide qué hace el clic sobre una tarjeta del historial (T46):
// reproduce el episodio guardado desde donde estaba (resume), salta al
// siguiente (next) o abre el detalle porque no hay siguiente (completed si el
// anime terminó o es película, up_to_date si sigue en emisión). El "siguiente"
// es el episodio de menor número estrictamente mayor que el actual (tolera
// huecos) y el número actual sale de ParseEpisodeID (canónico; película ⇒ 0),
// nunca del campo EpisodeNumber del registro.
//
// Los dos primeros casos (aún no visto y película vista) se resuelven SIN
// pedir episodios; solo una serie vista necesita la lista completa.
func (n *Nucleo) Retomar(ctx context.Context, provider, animeID string) (models.ResumeInfo, error) {
	provider = strings.TrimSpace(provider)
	animeID = strings.TrimSpace(animeID)

	h := n.historialAbierto()
	if h == nil {
		return models.ResumeInfo{}, ErrHistorialNoDisponible
	}
	// LastPerAnime (T41) ya devuelve un único registro por (proveedor,
	// anime): se busca el de ESTE par; el mismo slug de otro proveedor no
	// interfiere.
	var registro models.WatchProgress
	encontrado := false
	for _, r := range h.LastPerAnime() {
		if r.AnimeID == animeID && r.ProviderName() == provider {
			registro, encontrado = r, true
			break
		}
	}
	if !encontrado {
		return models.ResumeInfo{}, ErrSinHistorial
	}

	_, _, kind, numero, err := models.ParseEpisodeID(registro.EpisodeID)
	if err != nil {
		return models.ResumeInfo{}, err
	}
	if !registro.IsWatched || kind == models.KindMovie {
		// Ni el episodio a retomar ni el "siguiente" hacen falta aquí:
		// NO se pide el detalle.
		return decidirRetomar(provider, animeID, registro, kind, numero, nil, ""), nil
	}

	detalle, episodios, err := n.episodiosCompletos(ctx, provider, animeID)
	if err != nil {
		return models.ResumeInfo{}, err
	}
	return decidirRetomar(provider, animeID, registro, kind, numero, episodios, detalle.Status), nil
}

// SiguienteEpisodio devuelve el episodio siguiente al dado (T46/T55): el de
// menor número estrictamente mayor, tolerando huecos. Una película no tiene
// siguiente (Found=false, sin pedir episodios). Si la lista completa no pudo
// completarse se devuelve el error —la UI oculta el botón "Siguiente"— y un
// proveedor desconocido da el mismo error que Detalle.
func (n *Nucleo) SiguienteEpisodio(ctx context.Context, provider, animeID, episodeID string) (models.NextEpisode, error) {
	provider = strings.TrimSpace(provider)
	animeID = strings.TrimSpace(animeID)
	episodeID = strings.TrimSpace(episodeID)

	_, _, kind, numero, err := models.ParseEpisodeID(episodeID)
	if err != nil {
		return models.NextEpisode{}, err
	}
	if kind == models.KindMovie {
		return models.NextEpisode{Found: false}, nil
	}

	_, episodios, err := n.episodiosCompletos(ctx, provider, animeID)
	if err != nil {
		return models.NextEpisode{}, err
	}
	siguiente, ok := siguienteDe(episodios, numero)
	if !ok {
		return models.NextEpisode{Found: false}, nil
	}
	return models.NextEpisode{Found: true, EpisodeID: siguiente.ID, Number: siguiente.Number}, nil
}

// episodiosCompletos devuelve el detalle y la lista COMPLETA de episodios.
// Nunca devuelve una lista parcial.
//
//  1. proveedor y animeID se normalizan y la clave es la misma que usa la
//     caché de detalle (cache.Clave);
//  2. se llama a Detalle con el ctx RECIBIDO tal cual: el contexto del stream
//     nace del que se pasa a Detalle (iniciarStream) y cancelarlo mataría los
//     lotes que el frontend necesita después;
//  3. un proveedor sin scraper.EpisodeStreamer ya devolvió la lista completa
//     y síncrona (no se cachea), así que no hay nada que esperar;
//  4. con EpisodeStreamer la única señal de "terminó" es la CACHÉ (la entrada
//     la escriben Detalle y lote; no se leen st.fallo ni otros internos del
//     stream): se comprueba antes del primer tic y luego cada
//     intervaloEsperaEpisodios hasta que la entrada esté Complete, falte
//     (stream fallido, cancelado por otro anime o caducado), el ctx se
//     cancele o se agote esperaMaximaEpisodios.
func (n *Nucleo) episodiosCompletos(ctx context.Context, proveedor, animeID string) (models.AnimeDetail, []models.Episode, error) {
	proveedor = strings.TrimSpace(proveedor)
	animeID = strings.TrimSpace(animeID)
	clave := cache.Clave(proveedor, animeID)

	res, err := n.Detalle(ctx, proveedor, animeID)
	if err != nil {
		return models.AnimeDetail{}, nil, err
	}

	p, err := n.proveedor(proveedor)
	if err != nil {
		return models.AnimeDetail{}, nil, err
	}
	if _, ok := p.(scraper.EpisodeStreamer); !ok {
		return res.Detail, res.Episodes, nil
	}

	limite := time.NewTimer(esperaMaximaEpisodios)
	defer limite.Stop()
	for {
		e, ok := n.cacheLocal.Get(clave)
		if !ok {
			// Detalle ya había escrito la entrada (Set + Append) al
			// devolver sin error: si ya no está, el stream falló, otro
			// anime la borró o caducó. Jamás se devuelve la parcial.
			return models.AnimeDetail{}, nil, errListaEpisodios
		}
		if e.Complete {
			// Get devuelve copias: la caché y el llamador no comparten
			// memoria (ni los episodios ni los géneros del detalle).
			return e.Detail, e.Episodes, nil
		}
		select {
		case <-ctx.Done():
			return models.AnimeDetail{}, nil, ctx.Err()
		case <-limite.C:
			return models.AnimeDetail{}, nil, errListaEpisodios
		case <-time.After(intervaloEsperaEpisodios):
		}
	}
}

// decidirRetomar aplica la tabla de acciones del plan v1.2 sobre el último
// registro de un anime. kind y numero vienen de ParseEpisodeID del
// EpisodeID del registro; episodios y status solo se consultan cuando el
// registro está visto y el episodio no es una película (el llamador pasa en
// esos casos la lista COMPLETA que devolvió episodiosCompletos).
func decidirRetomar(proveedor, animeID string, registro models.WatchProgress, kind models.EpisodeKind, numero int, episodios []models.Episode, status string) models.ResumeInfo {
	info := models.ResumeInfo{Provider: proveedor, AnimeID: animeID}

	if !registro.IsWatched {
		// Reproduce ese episodio desde donde estaba (sin pedir episodios).
		info.Action = models.ResumeActionResume
		info.EpisodeID = registro.EpisodeID
		info.EpisodeNumber = numero
		info.StartAt = registro.CurrentTime
		return info
	}
	if kind == models.KindMovie {
		// Una película vista no tiene "siguiente": abre el detalle.
		info.Action = models.ResumeActionCompleted
		return info
	}
	if siguiente, ok := siguienteDe(episodios, numero); ok {
		// Reproduce el siguiente desde 0 (StartAt en cero).
		info.Action = models.ResumeActionNext
		info.EpisodeID = siguiente.ID
		info.EpisodeNumber = siguiente.Number
		return info
	}
	if estaEnEmision(status) {
		info.Action = models.ResumeActionUpToDate
	} else {
		info.Action = models.ResumeActionCompleted
	}
	return info
}

// siguienteDe devuelve el episodio de MENOR número estrictamente mayor que
// actual: no depende del orden de entrada, no asume números consecutivos
// (tolera huecos) e ignora las películas (kind movie), que no forman parte
// del progreso episódico.
func siguienteDe(episodes []models.Episode, actual int) (models.Episode, bool) {
	var mejor models.Episode
	hay := false
	for _, ep := range episodes {
		if ep.Kind == models.KindMovie || ep.Number <= actual {
			continue
		}
		if !hay || ep.Number < mejor.Number {
			mejor, hay = ep, true
		}
	}
	return mejor, hay
}

// estaEnEmision normaliza el status del detalle (minúsculas, sin acentos, sin
// espacios sobrantes) y devuelve true SOLO si el resultado es exactamente
// "en emision": cubre "En emision" (JKAnime, texto crudo del sitio) y
// "En emisión" (AnimeAV1, mapa del status numérico). Todo lo demás —
// Concluido, Finalizado, Por estrenar, Próximamente, Desconocido, vacío— es
// false. No se reconocen valores no observados en los dos proveedores.
func estaEnEmision(status string) bool {
	return quitarTildes.Replace(strings.ToLower(strings.TrimSpace(status))) == "en emision"
}

// quitarTildes acerca el texto a lo que teclea el usuario y a lo que dejan
// los dos scrapers al normalizar (JKAnime no acentúa "emision"; AnimeAV1 sí).
var quitarTildes = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o",
	"ú", "u", "ü", "u", "ñ", "n",
)
