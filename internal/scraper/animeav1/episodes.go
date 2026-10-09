package animeav1

import (
	"context"

	"miapp/internal/models"
	"miapp/internal/scraper"
)

// tamLote es el número de episodios que viaja en cada models.EpisodeBatch:
// 50, el mismo tamaño que pinta el sitio en su desplegable de rangos
// ("1 - 50", "51 - 100"…) y el que espera el E2E de T35. Los episodios
// llegan todos en la respuesta del detalle, así que el lote no descarga
// nada: solo reparte la lista y respeta el ctx entre lotes.
const tamLote = 50

var _ scraper.EpisodeStreamer = (*AnimeAV1)(nil)

// StreamAnimeDetails entrega el detalle y los primeros tamLote episodios de
// inmediato (esa primera tanda NO se emite como lote, igual que la primera
// página de JKAnime) y el resto en una goroutine, un lote de 50 en lote,
// con el último marcado como Done y siempre con AnimeID = slug del detalle.
//
// Si onBatch es nil no se lanza ninguna goroutine. Si ctx se cancela —antes
// o entre lotes— la goroutine termina sin emitir nada más, ni siquiera el
// lote Done, y sin quedar huérfana; los datos ya están en memoria, así que
// no hay espera artificial entre lotes.
func (a *AnimeAV1) StreamAnimeDetails(ctx context.Context, id string,
	onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
	detalle, episodios, err := a.detalleYEpisodios(ctx, id)
	if err != nil {
		return models.AnimeDetail{}, nil, err
	}
	corte := min(tamLote, len(episodios))
	primera := episodios[:corte:corte]
	if onBatch == nil {
		return detalle, primera, nil
	}
	go emitirLotes(ctx, detalle.ID, episodios[corte:], onBatch)
	return detalle, primera, nil
}

// emitirLotes reparte resto en lotes de tamLote (Batch 2 en adelante: el 1
// es el que ya se devolvió) y cierra el stream con Done en el último.
//
// Una lista sin resto —una serie de 50 episodios o menos, o una ficha sin
// episodios— emite un único lote Done sin episodios, que es lo que marca la
// entrada como completa y dispara episodes:complete. Cada lote se comprueba
// contra ctx justo antes de emitirlo: cancelar durante un onBatch corta el
// resto sin emitir nada más.
func emitirLotes(ctx context.Context, animeID string, resto []models.Episode,
	onBatch func(models.EpisodeBatch)) {
	if len(resto) == 0 {
		if ctx.Err() != nil {
			return
		}
		onBatch(models.EpisodeBatch{AnimeID: animeID, Batch: 1, Done: true})
		return
	}
	for i := 0; i < len(resto); i += tamLote {
		if ctx.Err() != nil {
			return
		}
		fin := min(i+tamLote, len(resto))
		onBatch(models.EpisodeBatch{
			AnimeID:  animeID,
			Batch:    2 + i/tamLote,
			Episodes: resto[i:fin],
			Done:     fin == len(resto),
		})
	}
}
