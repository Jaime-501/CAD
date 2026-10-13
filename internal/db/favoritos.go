package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"miapp/internal/models"
)

// nombreFavoritos es el archivo de favoritos dentro del directorio
// inyectado: vive JUNTO a historial.json y no lo toca (cada almacén tiene
// el suyo).
const nombreFavoritos = "favoritos.json"

// Favorites es el almacén de los animes marcados con la estrella (T42).
// Cero valor no es válido: usa OpenFavorites. Seguro para concurrencia
// (un único sync.Mutex), sin goroutines y con la misma escritura atómica y
// tolerancia a corruptos que el historial. A diferencia de DB no hay Close:
// no hay ciclo de vida que gestionar (cada Add/Remove ya escribió a disco).
type Favorites struct {
	mu     sync.Mutex
	dir    string           // directorio del archivo (mismo dir que el temporal)
	ruta   string           // ruta completa de favoritos.json
	now    func() time.Time // reloj inyectable; por defecto time.Now
	lista  []models.Favorite
	indice map[string]int // Favorite.Key() → posición en lista
}

// OpenFavorites abre el almacén de favoritos del directorio indicado,
// creando el directorio si hace falta. dir vacío no tiene significado
// especial (P3: este paquete no decide rutas): devuelve ErrDirectorioVacio,
// el mismo error que Open.
//
// Un archivo que no existe (primera ejecución) o vacío deja el almacén sin
// favoritos; uno con contenido ilegible se respalda como .bak y se empieza
// vacío SIN devolver error. Solo un fallo de E/S real devuelve error.
func OpenFavorites(dir string) (*Favorites, error) {
	if dir == "" {
		return nil, ErrDirectorioVacio
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("no pude crear el directorio de favoritos %q: %w", dir, err)
	}

	f := &Favorites{
		dir:    dir,
		ruta:   filepath.Join(dir, nombreFavoritos),
		now:    time.Now,
		lista:  make([]models.Favorite, 0),
		indice: make(map[string]int),
	}
	if err := f.cargar(); err != nil {
		return nil, err
	}
	return f, nil
}

// Add guarda un favorito nuevo (true) o actualiza uno existente (false).
// Nunca es un toggle: los favoritos son idempotentes.
//
// Requiere Provider y AnimeID no vacíos (la clave es Provider + ":" +
// AnimeID). AddedAt == 0 se sella con now().Unix(); un valor distinto de cero
// se conserva (así un "Deshacer" o una importación pueden fijar la fecha). Si
// la clave ya existe se conserva el AddedAt guardado, se actualizan Title y
// CoverURL y se persiste igualmente. Si la escritura a disco falla, el
// favorito queda en memoria —el próximo Add/Remove lo reintenta— y se
// devuelve el error junto al bool de "era nuevo".
func (f *Favorites) Add(fav models.Favorite) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if fav.Provider == "" || fav.AnimeID == "" {
		return false, errors.New("el favorito necesita provider y anime_id")
	}

	k := fav.Key()
	i, existia := f.indice[k]
	if existia {
		// Misma clave: no se mueve de sitio ni cambia su fecha de alta.
		antiguo := f.lista[i]
		antiguo.Title = fav.Title
		antiguo.CoverURL = fav.CoverURL
		f.lista[i] = antiguo
	} else {
		if fav.AddedAt == 0 {
			fav.AddedAt = f.now().Unix()
		}
		f.enMemoria(fav)
	}
	if err := f.escribir(); err != nil {
		return !existia, err
	}
	return !existia, nil
}

// Remove quita el favorito indicado. true si existía (y se persiste), false
// si no —en ese caso el archivo no se reescribe—. Si la escritura falla, el
// favorito ya no está en memoria (el próximo Add lo reintenta) y se
// devuelve el error junto al true.
func (f *Favorites) Remove(provider, animeID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := models.Favorite{Provider: provider, AnimeID: animeID}.Key()
	i, ok := f.indice[k]
	if !ok {
		return false, nil
	}
	f.lista = append(f.lista[:i], f.lista[i+1:]...)
	f.reindexar()
	if err := f.escribir(); err != nil {
		return true, err
	}
	return true, nil
}

// Has dice si el anime está en favoritos (proveedor + slug).
func (f *Favorites) Has(provider, animeID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, ok := f.indice[models.Favorite{Provider: provider, AnimeID: animeID}.Key()]
	return ok
}

// List devuelve todos los favoritos ordenados por AddedAt descendente —lo
// más reciente primero— y, con empate, por Key() ascendente. El resultado
// nunca es nil y sus elementos son copias: mutarlos no toca el almacén.
func (f *Favorites) List() []models.Favorite {
	f.mu.Lock()
	defer f.mu.Unlock()

	salida := make([]models.Favorite, len(f.lista))
	copy(salida, f.lista)
	sort.Slice(salida, func(i, j int) bool {
		if salida[i].AddedAt != salida[j].AddedAt {
			return salida[i].AddedAt > salida[j].AddedAt
		}
		return salida[i].Key() < salida[j].Key()
	})
	return salida
}

// cargar lee el archivo y lo indexa. No existe o vacío → sin favoritos;
// ilegible como JSON → respaldo .bak y sin favoritos (sin error); cualquier
// otro fallo de lectura sí se devuelve.
func (f *Favorites) cargar() error {
	datos, err := os.ReadFile(f.ruta)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("no pude leer los favoritos %q: %w", f.ruta, err)
	}
	if len(datos) == 0 {
		return nil
	}

	var lista []models.Favorite
	if err := json.Unmarshal(datos, &lista); err != nil {
		f.respaldar(err)
		return nil
	}
	for _, fav := range lista {
		f.enMemoria(fav)
	}
	return nil
}

// enMemoria inserta o reemplaza un favorito conservando su posición. Llamar
// con el mutex puesto.
func (f *Favorites) enMemoria(fav models.Favorite) {
	k := fav.Key()
	if i, ok := f.indice[k]; ok {
		f.lista[i] = fav
		return
	}
	f.indice[k] = len(f.lista)
	f.lista = append(f.lista, fav)
}

// reindexar reconstruye el índice tras un borrado (las posiciones cambian).
// Llamar con el mutex puesto.
func (f *Favorites) reindexar() {
	f.indice = make(map[string]int, len(f.lista))
	for i, fav := range f.lista {
		f.indice[fav.Key()] = i
	}
}

// respaldar renombra el archivo corrupto a .bak (si ya existía otro .bak, se
// sustituye) y deja el sitio limpio. Un fallo al renombrar solo se loguea:
// los favoritos siguen empezando vacíos.
func (f *Favorites) respaldar(causa error) {
	respaldo := f.ruta + sufijoRespaldo
	if err := os.Rename(f.ruta, respaldo); err != nil {
		slog.Warn("no pude respaldar los favoritos corruptos", "ruta", f.ruta, "err", err)
		return
	}
	slog.Warn("favoritos corruptos respaldados; se empieza vacío",
		"respaldo", respaldo, "err", causa)
}

// escribir vuelca la lista entera con escritura atómica —misma lógica que
// DB.escribir, aquí sin tocar db.go—: primero a un archivo temporal DENTRO
// del mismo directorio y sync, y después rename sobre el definitivo. Si algo
// falla, el temporal se elimina y el archivo anterior queda intacto. El JSON
// va indentado (legible a mano: son pocos registros y un humano puede
// recuperarlos). Llamar con el mutex puesto.
func (f *Favorites) escribir() error {
	datos, err := json.MarshalIndent(f.lista, "", "  ")
	if err != nil {
		return fmt.Errorf("no pude serializar los favoritos: %w", err)
	}

	tmp, err := os.CreateTemp(f.dir, nombreFavoritos+".*.tmp")
	if err != nil {
		return fmt.Errorf("no pude crear el archivo temporal de favoritos: %w", err)
	}
	nombreTmp := tmp.Name()
	sustituido := false
	defer func() {
		tmp.Close() // si ya se cerró, este error no interesa
		if !sustituido {
			os.Remove(nombreTmp)
		}
	}()

	if _, err := tmp.Write(datos); err != nil {
		return fmt.Errorf("no pude escribir los favoritos: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("no pude sincronizar los favoritos: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("no pude cerrar el favorito temporal: %w", err)
	}
	if err := os.Rename(nombreTmp, f.ruta); err != nil {
		return fmt.Errorf("no pude reemplazar los favoritos: %w", err)
	}
	sustituido = true
	return nil
}
