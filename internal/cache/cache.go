// Package cache implementa la caché en memoria con TTL de T21: guarda el
// detalle de un anime y sus episodios acumulados mientras —y después de—
// se descargan, para que navegar entre detalle y reproductor no vuelva a
// golpear el scraper (y no arriesgue un baneo).
//
// Qué se cachea: SOLO el flujo con scraper.EpisodeStreamer (detalle +
// episodios por lotes). Qué NO se cachea nunca: errores, streams terminados
// con Err ni GetVideoSources (los enlaces de vídeo son temporales, regla 9).
//
// Clave: Clave(proveedor, animeID). Estado: Entrada{Detail, Episodes,
// Complete}. Get siempre devuelve COPIAS: jamás se comparte memoria con la
// caché (ni los episodios ni los géneros del detalle).
//
// Caducidad por deslizamiento: cada escritura (Set/Append/MarkComplete)
// renueva el TTL. La limpieza de caducadas ocurre en la propia lectura (Get)
// y, si ttl > 0, también en una rutina periódica que se arranca con Arrancar
// y se detiene con Cerrar — el núcleo la arranca en su arrancar y la para en
// cerrar, así que no quedan goroutines huérfanas.
//
// La caché es segura para concurrencia (sync.RWMutex) y por sí sola no crea
// goroutines.
package cache

import (
	"sync"
	"time"

	"miapp/internal/models"
)

// Option es una opción de configuración de New.
type Option func(*Cache)

// ConReloj inyecta el reloj de la caché: los tests avanzan el tiempo sin
// time.Sleep. Un nil se ignora (se conserva time.Now).
func ConReloj(ahora func() time.Time) Option {
	return func(c *Cache) {
		if ahora != nil {
			c.ahora = ahora
		}
	}
}

// ConIntervalo cambia la cadencia de la limpieza periódica (por defecto la
// mitad del TTL, acotada a 1 s..60 s). Con 0 o menos no hay rutina: la
// limpieza queda solo en la lectura.
func ConIntervalo(d time.Duration) Option {
	return func(c *Cache) { c.intervalo = d }
}

// Entrada es lo que guarda la caché por anime: el detalle, los episodios
// acumulados (deduplicados por Episode.ID) y si el stream ya terminó.
type Entrada struct {
	Detail   models.AnimeDetail
	Episodes []models.Episode
	Complete bool
}

// viva es una entrada con su fecha de caducidad (cero = no caduca).
type viva struct {
	e      Entrada
	expira time.Time
}

// Cache es la caché de detalle+episodios. Cero valor no es válido: usa New.
type Cache struct {
	mu        sync.RWMutex
	ttl       time.Duration
	ahora     func() time.Time
	intervalo time.Duration
	entradas  map[string]viva

	// La rutina periódica: solo existe entre Arrancar y Cerrar.
	corriendo bool
	stop      chan struct{}
	done      chan struct{}
}

// New crea una caché con el TTL indicado. ttl <= 0 significa "no caduca".
// La limpieza de caducadas ocurre al leer (Get) y, si ttl > 0, también en la
// rutina periódica, que hay que arrancar con Arrancar y detener con Cerrar.
func New(ttl time.Duration, opts ...Option) *Cache {
	c := &Cache{
		ttl:       ttl,
		ahora:     time.Now,
		intervalo: intervaloPara(ttl),
		entradas:  make(map[string]viva),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Clave arma la clave de una entrada: proveedor + identificador del anime
// (p. ej. "jkanime:bleach"). El prefijo del proveedor evita que dos sitios
// compartan la misma entrada.
func Clave(proveedor, animeID string) string {
	return proveedor + ":" + animeID
}

// Get devuelve la entrada si existe y no ha caducado. El resultado es una
// COPIA: mutarlo no toca la caché. Una entrada caducada se elimina ahí mismo
// (limpieza en lectura) y se devuelve como ausente.
func (c *Cache) Get(clave string) (Entrada, bool) {
	c.mu.RLock()
	v, ok := c.entradas[clave]
	if !ok {
		c.mu.RUnlock()
		return Entrada{}, false
	}
	if c.caducada(v) {
		c.mu.RUnlock()
		c.Delete(clave)
		return Entrada{}, false
	}
	e := copiaEntrada(v.e)
	c.mu.RUnlock()
	return e, true
}

// Set guarda el detalle del anime, creando la entrada si no existe. NO toca
// los episodios acumulados ni Complete: esos campos solo los mueven Append y
// MarkComplete. Así el detalle puede llegar cuando llegue —incluso después de
// que un lote se haya emitido— sin pisar lo que ya se acumuló.
func (c *Cache) Set(clave string, det models.AnimeDetail) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ahora := c.ahora()
	v := c.recuperar(clave, ahora)
	v.e.Detail = copiaDetalle(det)
	v.expira = c.caducidad(ahora)
	c.entradas[clave] = v
}

// Append acumula episodios deduplicando por Episode.ID (una entrada sin ID
// se añade siempre, no hay con qué deduplicarla) y renueva el TTL. Si la
// entrada no existe la crea. No modifica Detail ni Complete.
func (c *Cache) Append(clave string, eps []models.Episode) {
	if len(eps) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	ahora := c.ahora()
	v := c.recuperar(clave, ahora)
	vistos := make(map[string]struct{}, len(v.e.Episodes)+len(eps))
	for _, ep := range v.e.Episodes {
		if ep.ID != "" {
			vistos[ep.ID] = struct{}{}
		}
	}
	for _, ep := range eps {
		if ep.ID != "" {
			if _, ya := vistos[ep.ID]; ya {
				continue
			}
			vistos[ep.ID] = struct{}{}
		}
		v.e.Episodes = append(v.e.Episodes, ep)
	}
	v.expira = c.caducidad(ahora)
	c.entradas[clave] = v
}

// MarkComplete marca la entrada como terminada (el stream llegó a su lote
// final sin Err). Crea la entrada si no existe: el Done puede llegar antes
// que el detalle. No cambia ni Detail ni los episodios.
func (c *Cache) MarkComplete(clave string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ahora := c.ahora()
	v := c.recuperar(clave, ahora)
	v.e.Complete = true
	v.expira = c.caducidad(ahora)
	c.entradas[clave] = v
}

// Delete elimina la entrada (existente o no). El núcleo lo usa para las
// entradas huérfanas: un stream cancelado o terminado con Err no debe dejar
// nada esperándolo.
func (c *Cache) Delete(clave string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entradas, clave)
}

// Arrancar pone en marcha la limpieza periódica de caducadas. Idempotente.
// Con ttl <= 0 (o intervalo <= 0) no hay nada que limpiar y no arranca
// ninguna goroutine. El núcleo la llama en su arrancar.
func (c *Cache) Arrancar() {
	if c.ttl <= 0 || c.intervalo <= 0 {
		return
	}
	c.mu.Lock()
	if c.corriendo {
		c.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	c.corriendo = true
	c.stop, c.done = stop, done
	intervalo := c.intervalo
	c.mu.Unlock()

	go c.bucleLimpieza(intervalo, stop, done)
}

// Cerrar detiene la rutina iniciada por Arrancar y ESPERA a que termine
// (ninguna goroutine sobrevive al shutdown). Idempotente: sin rutina no hace
// nada. El núcleo la llama en su cerrar.
func (c *Cache) Cerrar() {
	c.mu.Lock()
	if !c.corriendo {
		c.mu.Unlock()
		return
	}
	stop, done := c.stop, c.done
	c.corriendo = false
	c.stop, c.done = nil, nil
	c.mu.Unlock()

	close(stop)
	<-done
}

// bucleLimpieza poda caducadas hasta que se le ordena parar. Al salir cierra
// done, que es lo que espera Cerrar.
func (c *Cache) bucleLimpieza(intervalo time.Duration, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			c.podar()
		}
	}
}

// podar elimina todas las entradas caducadas y devuelve cuántas quita. Usa
// un solo tic de reloj para toda la pasada.
func (c *Cache) podar() int {
	ahora := c.ahora()

	c.mu.Lock()
	defer c.mu.Unlock()

	quitas := 0
	for clave, v := range c.entradas {
		if c.expira(v, ahora) {
			delete(c.entradas, clave)
			quitas++
		}
	}
	return quitas
}

// caducada dice si una entrada ya venció según el reloj inyectado.
func (c *Cache) caducada(v viva) bool {
	return c.expira(v, c.ahora())
}

// expira es el criterio de caducidad compartido por Get y podar: sin TTL no
// caduca nunca, y una fecha cero (nunca escrita) tampoco.
func (c *Cache) expira(v viva, ahora time.Time) bool {
	if c.ttl <= 0 || v.expira.IsZero() {
		return false
	}
	return !ahora.Before(v.expira)
}

// recuperar devuelve la entrada viva de la clave para escribir sobre ella: la
// crea vacía si no existe y también si ya caducó (una entrada vencida no se
// revive con datos viejos). Llamar con el mu puesto y con un solo tic de
// reloj por operación.
func (c *Cache) recuperar(clave string, ahora time.Time) viva {
	v, ok := c.entradas[clave]
	if !ok || c.expira(v, ahora) {
		return viva{}
	}
	return v
}

// caducidad es la nueva fecha de caducidad para una escritura (TTL por
// deslizamiento). Con ttl <= 0 devuelve la fecha cero: no caduca.
func (c *Cache) caducidad(ahora time.Time) time.Time {
	if c.ttl <= 0 {
		return time.Time{}
	}
	return ahora.Add(c.ttl)
}

// intervaloPara calcula la cadencia por defecto de la limpieza periódica:
// la mitad del TTL, acotada entre 1 s y 60 s. ttl <= 0 → 0 (sin rutina).
func intervaloPara(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return 0
	}
	i := ttl / 2
	if i < time.Second {
		return time.Second
	}
	if i > time.Minute {
		return time.Minute
	}
	return i
}

// copiaEntrada copia la entrada entera: los episodios a su propio slice y el
// detalle con sus géneros (los únicos slices que trae). Así Get nunca devuelve
// memoria que comparta con la caché.
func copiaEntrada(e Entrada) Entrada {
	c := Entrada{
		Detail:   copiaDetalle(e.Detail),
		Complete: e.Complete,
	}
	if e.Episodes != nil {
		c.Episodes = make([]models.Episode, len(e.Episodes))
		copy(c.Episodes, e.Episodes)
	}
	return c
}

// copiaDetalle copia los géneros del detalle (el resto de campos son valores).
func copiaDetalle(d models.AnimeDetail) models.AnimeDetail {
	if d.Genres != nil {
		g := make([]string, len(d.Genres))
		copy(g, d.Genres)
		d.Genres = g
	}
	return d
}
