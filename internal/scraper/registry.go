package scraper

import (
	"errors"
	"sort"
	"sync"
)

// Registry guarda providers por su Name(). Seguro para concurrencia y con
// valor cero utilizable (var r Registry).
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry crea un registro vacío.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register añade un provider bajo su Name(). Si ya existía uno con ese
// nombre, lo reemplaza (gana el último). Devuelve error si el provider es
// nil o su nombre está vacío.
func (r *Registry) Register(p Provider) error {
	if p == nil {
		return errors.New("scraper: no se puede registrar un provider nil")
	}
	if p.Name() == "" {
		return errors.New("scraper: el provider no tiene nombre")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers == nil {
		r.providers = make(map[string]Provider)
	}
	r.providers[p.Name()] = p
	return nil
}

// Get devuelve el provider con ese nombre y si existe.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	return p, ok
}

// All devuelve una copia de todos los providers ordenados por nombre.
// Modificar el slice devuelto no afecta al registro.
func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
