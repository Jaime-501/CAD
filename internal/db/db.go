// Package db persiste el historial de visualización (T22) en un archivo
// JSON dentro de un directorio inyectable: lo elige quien llama —en la app,
// app.go decide la ruta— y en los tests es t.TempDir(). Este paquete NO
// consulta el sistema (P3: portabilidad Android, ver portabilidad_test).
// Abrir el almacén es cosa del núcleo de la app (arrancar en startup,
// cerrar en shutdown).
//
// Qué guarda: registros models.WatchProgress (un episodio dentro de un
// anime), con IsWatched CALCULADO AQUÍ —no confíes en el que traiga el
// llamador—: CurrentTime/TotalDuration >= UmbralIsWatched, ignorando los
// episodios con TotalDuration <= 0.
//
// Escritura: atómica (archivo temporal en el MISMO directorio + rename),
// así que un lector nunca ve el archivo a medias. Cada SaveProgress escribe
// a disco de inmediato —el reproductor lo llama cada ~5 s— bajo un único
// mutex: no hay goroutines, ni búferes pendientes ni time.Sleep.
//
// Tolerancia: un archivo ilegible como JSON no rompe nada: se respalda como
// .bak y se empieza con el historial vacío.
//
// T41: cada guardada sella UpdatedAt (si el registro viene sin fecha) y
// rellena Provider; LastPerAnime da el último registro por anime ordenado
// por recencia —lo que pinta la barra lateral— y DeleteAnime /
// DeleteProvider / Clear borran registros con la misma escritura atómica.
package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"miapp/internal/models"
)

const (
	// UmbralIsWatched es la fracción mínima de un episodio que hay que tener
	// vista para darlo por visto: CurrentTime/TotalDuration >= 0.90. Es el
	// único punto donde se decide ese criterio (constante configurable).
	UmbralIsWatched = 0.90

	// nombreArchivo es el historial dentro del directorio inyectado.
	nombreArchivo = "historial.json"
	// sufijoRespaldo se añade al nombre para el archivo corrupto: el
	// contenido ilegible se conserva (alguien puede querer recuperarlo a
	// mano) y el sitio queda limpio para empezar de cero.
	sufijoRespaldo = ".bak"
)

// ErrCerrado es lo que devuelven SaveProgress/GetProgress después de Close.
var ErrCerrado = errors.New("el historial de visualización está cerrado")

// ErrDirectorioVacio es lo que devuelve Open cuando el directorio está
// vacío: decidir la ruta por defecto es cosa de quien llama (P3: app.go),
// nunca de este paquete.
var ErrDirectorioVacio = errors.New("el historial necesita un directorio no vacío")

// clave identifica un registro: un episodio dentro de un anime. Dos animes
// distintos no pueden pisarse aunque compartan episodio.
type clave struct {
	anime    string
	episodio string
}

// claveAnime identifica un anime del historial: su proveedor —con
// ProviderName(), que cubre los registros antiguos guardados sin el campo
// provider— más el slug (el slug se comparte entre proveedores).
type claveAnime struct {
	proveedor string
	anime     string
}

// DB es el almacén del historial. Cero valor no es válido: usa Open. Es
// segura para concurrencia (un único sync.Mutex) y no crea goroutines.
type DB struct {
	mu      sync.Mutex
	dir     string // directorio del archivo (mismo dir que el temporal)
	ruta    string // ruta completa del historial
	cerrado bool
	now     func() time.Time // reloj inyectable; por defecto time.Now (T41)

	// registros es el historial en memoria, en orden de PRIMERA guardada;
	// indice ubica cada (anime, episodio) dentro de ese orden. Tras un
	// borrado las posiciones se rehacen (reindexar).
	registros []models.WatchProgress
	indice    map[clave]int
}

// Open abre el historial del directorio indicado, creando el directorio si
// hace falta. dir vacío no tiene significado especial (P3: este paquete no
// decide rutas): devuelve ErrDirectorioVacio y quien llama resuelve la suya.
//
// Un archivo que no existe (primera ejecución) o vacío deja el almacén sin
// registros; uno con contenido ilegible se respalda como .bak y se empieza
// vacío SIN devolver error: un historial corrupto nunca hace fallar el
// arranque. Solo un fallo de E/S real (sin permisos, directorio
// inexistente...) devuelve error.
func Open(dir string) (*DB, error) {
	if dir == "" {
		return nil, ErrDirectorioVacio
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("no pude crear el directorio del historial %q: %w", dir, err)
	}

	d := &DB{
		dir:    dir,
		ruta:   filepath.Join(dir, nombreArchivo),
		now:    time.Now,
		indice: make(map[clave]int),
	}
	if err := d.cargar(); err != nil {
		return nil, err
	}
	return d, nil
}

// SaveProgress guarda (o actualiza) el progreso de UN episodio y lo
// persiste a disco en el momento. El reproductor lo llamará cada ~5 s: la
// operación es un marshal de JSON pequeño + rename atómico, todo bajo el
// mutex del almacén — dos guardadas nunca se entrelazan y GetProgress
// espera como mucho a esa escritura.
//
// IsWatched lo calcula aquí el backend: el valor que traiga el llamador se
// ignora. Si el registro sale idéntico al guardado no se reescribe el
// archivo (guardar en pausa no gasta I/O). Si la escritura falla, el dato
// se conserva en memoria —el próximo guardado lo reintentará— y se devuelve
// el error.
//
// T41: además del sello de tiempo —si UpdatedAt viene 0 se pone
// now().Unix(); si trae otra fecha se conserva, para que un "Deshacer"
// restaure el orden original— y del proveedor —si Provider viene vacío se
// rellena con ProviderName(), el prefijo de EpisodeID—. Solo al guardar: un
// archivo antiguo cargado conserva sus campos en cero.
func (d *DB) SaveProgress(progress models.WatchProgress) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cerrado {
		return ErrCerrado
	}
	if progress.AnimeID == "" || progress.EpisodeID == "" {
		return errors.New("el progreso necesita anime_id y episode_id")
	}
	progress = normalizar(progress)
	if progress.UpdatedAt == 0 {
		progress.UpdatedAt = d.now().Unix()
	}
	if progress.Provider == "" {
		progress.Provider = progress.ProviderName()
	}

	k := clave{anime: progress.AnimeID, episodio: progress.EpisodeID}
	if i, ok := d.indice[k]; ok && d.registros[i] == progress {
		return nil
	}
	d.guardarEnMemoria(progress)
	return d.escribir()
}

// GetProgress devuelve los registros de un anime en el orden en que se
// guardaron por primera vez (guardar otra vez actualiza en su sitio, sin
// moverlo). animeID vacío devuelve el historial entero. El resultado es una
// copia: mutarlo no toca el almacén.
func (d *DB) GetProgress(animeID string) ([]models.WatchProgress, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cerrado {
		return nil, ErrCerrado
	}
	salida := make([]models.WatchProgress, 0, len(d.registros))
	for _, r := range d.registros {
		if animeID == "" || r.AnimeID == animeID {
			salida = append(salida, r)
		}
	}
	return salida, nil
}

// LastPerAnime devuelve el ÚLTIMO registro de cada anime (T41), que es lo
// que pinta la barra lateral: un registro por (proveedor, anime_id) —el de
// mayor UpdatedAt; con empate, el de mayor EpisodeNumber y después el de
// EpisodeID descendente lexicográfico— y el conjunto ordenado por UpdatedAt
// descendente, con empate por proveedor y anime_id ascendentes. Los
// registros antiguos sin fecha (UpdatedAt == 0) quedan al final.
//
// El proveedor se compara con ProviderName(), que cubre los registros
// guardados antes de T41 sin el campo provider. El resultado nunca es nil
// (lista vacía sin registros) y sus elementos son copias: mutarlos no toca
// el almacén. Cerrado ⇒ lista vacía (sin error: la firma no lo admite; el
// llamador comprueba antes con GetProgress si necesita distinguirlo).
func (d *DB) LastPerAnime() []models.WatchProgress {
	d.mu.Lock()
	defer d.mu.Unlock()

	salida := make([]models.WatchProgress, 0, len(d.indice))
	if d.cerrado {
		return salida
	}

	ganadores := make(map[claveAnime]int, len(d.indice))
	for i, r := range d.registros {
		k := claveAnime{proveedor: r.ProviderName(), anime: r.AnimeID}
		if j, ok := ganadores[k]; ok && !mejorQue(r, d.registros[j]) {
			continue
		}
		ganadores[k] = i
	}
	for _, i := range ganadores {
		salida = append(salida, d.registros[i]) // copia: WatchProgress es todo valor
	}
	sort.Slice(salida, func(i, j int) bool {
		a, b := salida[i], salida[j]
		if a.UpdatedAt != b.UpdatedAt {
			return a.UpdatedAt > b.UpdatedAt
		}
		if pa, pb := a.ProviderName(), b.ProviderName(); pa != pb {
			return pa < pb
		}
		return a.AnimeID < b.AnimeID
	})
	return salida
}

// DeleteAnime borra TODOS los registros de un anime (todos sus episodios) de
// un proveedor y devuelve cuántos han sido (T41). El proveedor se compara
// con ProviderName(), así que también alcanza a los registros antiguos sin
// el campo provider. No existe ⇒ 0 sin error. Cerrado ⇒ ErrCerrado.
func (d *DB) DeleteAnime(provider, animeID string) (int, error) {
	return d.borrar(func(r models.WatchProgress) bool {
		return r.AnimeID == animeID && r.ProviderName() == provider
	})
}

// DeleteProvider borra todos los registros de un proveedor y devuelve cuántos
// han sido (T41). Mismas reglas que DeleteAnime (ProviderName, 0 sin error,
// ErrCerrado).
func (d *DB) DeleteProvider(provider string) (int, error) {
	return d.borrar(func(r models.WatchProgress) bool {
		return r.ProviderName() == provider
	})
}

// Clear borra el historial entero y devuelve cuántos registros había (T41).
// Ya vacío ⇒ 0 sin error. Cerrado ⇒ ErrCerrado.
func (d *DB) Clear() (int, error) {
	return d.borrar(func(models.WatchProgress) bool { return true })
}

// Close deja el almacén cerrado: las llamadas posteriores devuelven
// ErrCerrado. No hay nada en búfer que volcar (cada SaveProgress ya escribió
// a disco), así que esto solo evita que un guardado tardío reabra el archivo
// después del shutdown. Idempotente.
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cerrado = true
	return nil
}

// cargar lee el archivo del historial y lo indexa. No existe o vacío → sin
// registros; ilegible como JSON → respaldo .bak y sin registros (sin error);
// cualquier otro fallo de lectura sí se devuelve.
func (d *DB) cargar() error {
	datos, err := os.ReadFile(d.ruta)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("no pude leer el historial %q: %w", d.ruta, err)
	}
	if len(datos) == 0 {
		return nil
	}

	var registros []models.WatchProgress
	if err := json.Unmarshal(datos, &registros); err != nil {
		d.respaldar(err)
		return nil
	}
	for _, r := range registros {
		// Al cargar se re-aplica el mismo normalizado que al guardar: los
		// is_watched de un archivo viejo o editado a mano quedan al día.
		d.guardarEnMemoria(normalizar(r))
	}
	return nil
}

// guardarEnMemoria inserta o reemplaza un registro, conservando el orden de
// primera guardada. Llamar con el mutex puesto.
func (d *DB) guardarEnMemoria(progress models.WatchProgress) {
	k := clave{anime: progress.AnimeID, episodio: progress.EpisodeID}
	if i, ok := d.indice[k]; ok {
		d.registros[i] = progress
		return
	}
	d.indice[k] = len(d.registros)
	d.registros = append(d.registros, progress)
}

// borrar quita los registros que cumplan el predicado, conserva el orden de
// los que quedan, lo persiste con la MISMA escritura atómica que SaveProgress
// y devuelve el número de borrados. Si no se borra nada no se reescribe el
// archivo; si la escritura falla, los borrados ya no están en memoria —el
// próximo guardado los persistirá— y se devuelve el error junto al conteo.
func (d *DB) borrar(predicado func(models.WatchProgress) bool) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cerrado {
		return 0, ErrCerrado
	}
	restantes := make([]models.WatchProgress, 0, len(d.registros))
	borrados := 0
	for _, r := range d.registros {
		if predicado(r) {
			borrados++
			continue
		}
		restantes = append(restantes, r)
	}
	if borrados == 0 {
		return 0, nil
	}
	d.registros = restantes
	d.reindexar()
	if err := d.escribir(); err != nil {
		return borrados, err
	}
	return borrados, nil
}

// reindexar reconstruye el índice tras un borrado (las posiciones dentro del
// orden de primera guardada cambian). Llamar con el mutex puesto.
func (d *DB) reindexar() {
	d.indice = make(map[clave]int, len(d.registros))
	for i, r := range d.registros {
		d.indice[clave{anime: r.AnimeID, episodio: r.EpisodeID}] = i
	}
}

// respaldar renombra el archivo corrupto a .bak (si ya existía otro .bak, se
// sustituye: también era basura) y deja el sitio limpio. Un fallo al
// renombrar solo se loguea: el historial sigue empezando vacío.
func (d *DB) respaldar(causa error) {
	respaldo := d.ruta + sufijoRespaldo
	if err := os.Rename(d.ruta, respaldo); err != nil {
		slog.Warn("no pude respaldar el historial corrupto", "ruta", d.ruta, "err", err)
		return
	}
	slog.Warn("historial corrupto respaldado; se empieza vacío",
		"respaldo", respaldo, "err", causa)
}

// escribir vuelca el historial entero con escritura atómica: primero a un
// archivo temporal DENTRO del mismo directorio (para que el rename sea
// entre rutas del mismo sistema de ficheros) y sync, y después rename sobre
// el definitivo. Si algo falla, el temporal se elimina y el archivo
// anterior queda intacto. Llamar con el mutex puesto.
func (d *DB) escribir() error {
	datos, err := json.Marshal(d.registros)
	if err != nil {
		return fmt.Errorf("no pude serializar el historial: %w", err)
	}

	tmp, err := os.CreateTemp(d.dir, nombreArchivo+".*.tmp")
	if err != nil {
		return fmt.Errorf("no pude crear el archivo temporal del historial: %w", err)
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
		return fmt.Errorf("no pude escribir el historial: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("no pude sincronizar el historial: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("no pude cerrar el historial temporal: %w", err)
	}
	if err := os.Rename(nombreTmp, d.ruta); err != nil {
		return fmt.Errorf("no pude reemplazar el historial: %w", err)
	}
	sustituido = true
	return nil
}

// normalizar prepara un registro para guardarlo: sanea los flotantes no
// finitos —el <video> puede entregar NaN o Infinity, que JSON no sabe
// escribir— y calcula IsWatched en el backend.
func normalizar(progress models.WatchProgress) models.WatchProgress {
	progress.CurrentTime = fino(progress.CurrentTime)
	progress.TotalDuration = fino(progress.TotalDuration)
	progress.IsWatched = esVisto(progress.CurrentTime, progress.TotalDuration)
	return progress
}

// esVisto aplica el umbral de T22: CurrentTime/TotalDuration >=
// UmbralIsWatched. Un episodio sin duración útil (TotalDuration <= 0, que
// además ya está saneado a 0) NUNCA se da por visto.
func esVisto(currentTime, totalDuration float64) bool {
	if totalDuration <= 0 {
		return false
	}
	return currentTime/totalDuration >= UmbralIsWatched
}

// mejorQue decide si a gana a b para ser el último registro de su anime
// (T41): UpdatedAt mayor; con empate, EpisodeNumber mayor; a igual número,
// EpisodeID descendente lexicográfico.
func mejorQue(a, b models.WatchProgress) bool {
	if a.UpdatedAt != b.UpdatedAt {
		return a.UpdatedAt > b.UpdatedAt
	}
	if a.EpisodeNumber != b.EpisodeNumber {
		return a.EpisodeNumber > b.EpisodeNumber
	}
	return a.EpisodeID > b.EpisodeID
}

// fino devuelve 0 para NaN e infinitos (positivos o negativos): JSON no los
// puede serializar y el <video> los produce con facilidad.
func fino(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}
