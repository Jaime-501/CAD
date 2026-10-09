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
package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"

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

// DB es el almacén del historial. Cero valor no es válido: usa Open. Es
// segura para concurrencia (un único sync.Mutex) y no crea goroutines.
type DB struct {
	mu      sync.Mutex
	dir     string // directorio del archivo (mismo dir que el temporal)
	ruta    string // ruta completa del historial
	cerrado bool

	// registros es el historial en memoria, en orden de PRIMERA guardada;
	// indice ubica cada (anime, episodio) dentro de ese orden. No hay
	// borrado: las posiciones son estables.
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

// fino devuelve 0 para NaN e infinitos (positivos o negativos): JSON no los
// puede serializar y el <video> los produce con facilidad.
func fino(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}
