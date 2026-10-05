package scraper

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"miapp/internal/models"
)

// fakeProvider es un provider falso para probar el registro.
type fakeProvider struct {
	name     string
	detalles bool
}

var (
	_ Provider        = (*fakeProvider)(nil)
	_ EpisodeStreamer = (*fakeProvider)(nil)
)

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Search(ctx context.Context, query string) ([]models.Anime, error) {
	return []models.Anime{{ID: query, Title: query, Provider: f.name}}, nil
}

func (f *fakeProvider) GetAnimeDetails(ctx context.Context, id string) (models.AnimeDetail, []models.Episode, error) {
	if !f.detalles {
		return models.AnimeDetail{}, nil, ErrNoDetails
	}
	return models.AnimeDetail{ID: id, Provider: f.name}, nil, nil
}

func (f *fakeProvider) GetVideoSources(ctx context.Context, episodeID string, opts SourceOpts) ([]models.VideoSource, error) {
	return []models.VideoSource{{
		URL:     "https://cdn.example.test/" + episodeID + ".m3u8",
		Type:    models.ResourceHLS,
		Quality: "720p",
		Server:  f.name,
	}}, nil
}

func (f *fakeProvider) StreamAnimeDetails(ctx context.Context, id string,
	onBatch func(models.EpisodeBatch)) (models.AnimeDetail, []models.Episode, error) {
	if !f.detalles {
		return models.AnimeDetail{}, nil, ErrNoDetails
	}
	episodes := []models.Episode{{ID: "e1", Number: 1, Kind: models.KindEpisode}}
	if onBatch != nil {
		onBatch(models.EpisodeBatch{AnimeID: id, Batch: 1, Episodes: episodes})
	}
	return models.AnimeDetail{ID: id, Provider: f.name}, episodes, nil
}

func TestRegistryRegisterYGet(t *testing.T) {
	r := NewRegistry()
	jk := &fakeProvider{name: "jkanime", detalles: true}
	fv := &fakeProvider{name: "animeflv"}

	if err := r.Register(jk); err != nil {
		t.Fatalf("Register(jkanime): %v", err)
	}
	if err := r.Register(fv); err != nil {
		t.Fatalf("Register(animeflv): %v", err)
	}

	got, ok := r.Get("jkanime")
	if !ok {
		t.Fatal("Get(jkanime) no encontró el provider")
	}
	if got != jk {
		t.Errorf("Get(jkanime) = %v, quería el provider registrado", got)
	}

	if _, ok := r.Get("desconocido"); ok {
		t.Error("Get(desconocido) encontró algo con el registro vacío de ese nombre")
	}
}

func TestRegistryReemplazaDuplicado(t *testing.T) {
	r := NewRegistry()
	primero := &fakeProvider{name: "jkanime"}
	segundo := &fakeProvider{name: "jkanime", detalles: true}

	if err := r.Register(primero); err != nil {
		t.Fatalf("Register(primero): %v", err)
	}
	if err := r.Register(segundo); err != nil {
		t.Fatalf("Register(segundo): %v", err)
	}

	got, ok := r.Get("jkanime")
	if !ok || got != segundo {
		t.Errorf("Get(jkanime) = %v, quería el último registrado", got)
	}
	if len(r.All()) != 1 {
		t.Errorf("All() tiene %d providers, quería 1 (sin duplicados)", len(r.All()))
	}
}

func TestRegistryRegistroInvalido(t *testing.T) {
	r := NewRegistry()

	if err := r.Register(nil); err == nil {
		t.Error("Register(nil) no devolvió error")
	}
	if err := r.Register(&fakeProvider{}); err == nil {
		t.Error("Register con nombre vacío no devolvió error")
	}
	if len(r.All()) != 0 {
		t.Errorf("All() = %d providers, quería 0 tras registros fallidos", len(r.All()))
	}
}

func TestRegistryCeroValorUtilizable(t *testing.T) {
	var r Registry
	if err := r.Register(&fakeProvider{name: "jkanime"}); err != nil {
		t.Fatalf("Register sobre valor cero: %v", err)
	}
	if _, ok := r.Get("jkanime"); !ok {
		t.Error("Get(jkanime) sobre valor cero no encontró el provider")
	}
}

func TestRegistryAllOrdenadoYSnapshot(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"zeta", "alfa", "mega"} {
		if err := r.Register(&fakeProvider{name: name}); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All() = %d providers, quería 3", len(all))
	}
	want := []string{"alfa", "mega", "zeta"}
	for i, p := range all {
		if p.Name() != want[i] {
			t.Errorf("All()[%d] = %q, quería %q (orden por nombre)", i, p.Name(), want[i])
		}
	}

	// El slice devuelto es una copia: tocarlo no rompe el registro.
	all[0] = &fakeProvider{name: "intruso"}
	all = append(all, &fakeProvider{name: "intruso"})
	if len(r.All()) != 3 {
		t.Errorf("All() = %d providers tras mutar la copia, quería 3", len(r.All()))
	}
	if p, ok := r.Get("alfa"); !ok || p.Name() != "alfa" {
		t.Errorf("Get(alfa) = %v, %v; el registro se vio afectado por la copia", p, ok)
	}
}

func TestRegistryConcurrente(t *testing.T) {
	r := NewRegistry()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 3)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			name := "provider-" + strconv.Itoa(i)
			if err := r.Register(&fakeProvider{name: name}); err != nil {
				t.Errorf("Register(%s): %v", name, err)
			}
		}(i)
		go func() {
			defer wg.Done()
			r.Get("jkanime")
		}()
		go func() {
			defer wg.Done()
			r.All()
		}()
	}
	wg.Wait()

	if got := len(r.All()); got != n {
		t.Errorf("All() = %d providers, quería %d", got, n)
	}
}

func TestErrNoDetails(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeProvider{name: "animeflv"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	p, _ := r.Get("animeflv")
	_, _, err := p.GetAnimeDetails(context.Background(), "bleach")
	if !errors.Is(err, ErrNoDetails) {
		t.Errorf("error = %v, quería ErrNoDetails", err)
	}
}
