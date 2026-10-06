package stream

import (
	"crypto/rand"
	"sync"
	"time"
)

const (
	// TTLDefecto es la vida de una sesión desde su creación o desde su
	// último acceso (caducidad por deslizamiento): basta con que aguante
	// el hueco entre dos fragmentos seguidos mientras se reproduce.
	TTLDefecto = 90 * time.Minute
	// LimiteDefecto es el número máximo de sesiones vivas. Al superarlo se
	// purgan las caducadas y, si siguen llenas, la que antes caduca.
	LimiteDefecto = 128
)

// sesion guarda las cabeceras que el proxy inyecta al reenviar: son las del
// VideoSource registrado, nunca viajan en la URL. Tras crearla no se muta,
// así que varios handlers pueden leerla sin bloquear.
type sesion struct {
	referer string
	headers map[string]string
	expira  time.Time
}

// sesiones es el almacén en memoria de las sesiones, con TTL y límite.
type sesiones struct {
	mu     sync.Mutex
	ttl    time.Duration
	limite int
	vivas  map[string]*sesion
}

func nuevasSesiones(ttl time.Duration, limite int) *sesiones {
	if ttl <= 0 {
		ttl = TTLDefecto
	}
	if limite <= 0 {
		limite = LimiteDefecto
	}
	return &sesiones{ttl: ttl, limite: limite, vivas: map[string]*sesion{}}
}

// crear da de alta una sesión con las cabeceras indicadas y devuelve su id.
func (s *sesiones) crear(referer string, headers map[string]string) string {
	copia := make(map[string]string, len(headers))
	for k, v := range headers {
		copia[k] = v
	}
	id := nuevoID()

	s.mu.Lock()
	defer s.mu.Unlock()
	ahora := time.Now()
	s.purgar(ahora)
	for len(s.vivas) >= s.limite {
		if !s.evol() {
			break
		}
	}
	s.vivas[id] = &sesion{referer: referer, headers: copia, expira: ahora.Add(s.ttl)}
	return id
}

// obtener devuelve la sesión viva (renovando su caducidad) o nil si no
// existe o ha caducado.
func (s *sesiones) obtener(id string) *sesion {
	if id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ses, ok := s.vivas[id]
	if !ok {
		return nil
	}
	ahora := time.Now()
	if !ses.expira.After(ahora) {
		delete(s.vivas, id)
		return nil
	}
	ses.expira = ahora.Add(s.ttl)
	return ses
}

// purgar elimina las sesiones caducadas.
func (s *sesiones) purgar(ahora time.Time) {
	for id, ses := range s.vivas {
		if !ses.expira.After(ahora) {
			delete(s.vivas, id)
		}
	}
}

// evol elimina la sesión que antes caduca y devuelve false si no queda
// ninguna (para no enciclar al creador).
func (s *sesiones) evol() bool {
	var id string
	var cuando time.Time
	for k, ses := range s.vivas {
		if id == "" || ses.expira.Before(cuando) {
			id, cuando = k, ses.expira
		}
	}
	if id == "" {
		return false
	}
	delete(s.vivas, id)
	return true
}

// nueva devuelve un id de sesión aleatorio y no adivinable.
func nuevoID() string {
	return rand.Text()
}
