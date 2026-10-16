package animeav1

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"miapp/internal/httpx"
	"miapp/internal/logx"
	"miapp/internal/models"
	"miapp/internal/scraper"
	"miapp/internal/scraper/embeds"
)

const (
	// maxFuentesBytes limita el __data.json de un episodio (10 MB), al
	// estilo de maxBusquedaBytes y maxDetalleBytes: un cuerpo mayor se
	// rechaza sin parsearlo.
	maxFuentesBytes = 10 << 20

	// hostJugador es el único host del que se deriva un manifiesto HLS:
	// el reproductor propio del sitio (player.zilla-networks.com). El
	// resto de embeds —UPNShare, Voe, Byse, MP4Upload— salen como iframe
	// sin resolverlos (T33; pendiente anotado en PROGRESO.md).
	hostJugador = "player.zilla-networks.com"

	// Esquema, ruta del embed del reproductor (/play/<id>), ruta del
	// manifiesto derivado (/m3u8/<id>) y longitud del id que exige el
	// propio JS del player («if a.length !== 32 return», informe T30/P4).
	// /play/* exige Referer de animeav1: nunca se pide, solo se deriva.
	esquemaJugador = "https://"
	rutaJugador    = "/play/"
	rutaManifiesto = "/m3u8/"
	largoIDJugador = 32

	// cabeceraFetch y valorFetch declaran el origen de la petición de los
	// segmentos HLS: Sec-Fetch-Site es cabecera prohibida del navegador,
	// los segmentos responden 403 sin ella y la inyecta el proxy local
	// (stream/reenvio.go); el manifiesto responde 200 con ella (T30).
	cabeceraFetch = "Sec-Fetch-Site"
	valorFetch    = "same-origin"
)

// idiomasEmbeds es el recorrido determinista de las claves de embeds: el
// sitio solo publica SUB y DUB (informe T30/P4) y SUB va primero, así el
// orden de salida no depende de la iteración —aleatoria— de un mapa de Go.
var idiomasEmbeds = []string{models.LanguageSub, models.LanguageDub}

// GetVideoSources descarga el __data.json del episodio identificado por
// episodeID (p. ej. "animeav1:bleach:episode:1") y devuelve sus fuentes:
// el HLS del reproductor propio —derivado sin petición de /play/<id> a
// /m3u8/<id>— y el resto de servidores como iframe, cada una con su idioma
// estructurado en models.VideoSource.Language (SUB/DUB).
//
// Tras construir la lista, los iframes de hosts reconocidos se resuelven
// con los resolvers de embeds (T61) y se añade su mp4 directo conservando
// el iframe y el idioma.
//
// Es una sola petición, sin cookies ni CSRF (T30). downloads se ignora: son
// enlaces de descarga, no reproductores embebibles. opts.NoCache no cambia
// nada: las fuentes nunca se cachean (regla 9).
//
// Los errores HTTP llegan tal cual de httpx y un episodio fuera de rango
// —que SvelteKit contesta con HTTP 200 y un nodo de error 404— devuelve el
// mismo error tipado de "no encontrado" que el detalle (T32). Sin embeds
// reproducibles hay error descriptivo, nunca un slice vacío silencioso.
func (a *AnimeAV1) GetVideoSources(ctx context.Context, episodeID string, opts scraper.SourceOpts) ([]models.VideoSource, error) {
	proveedor, slug, kind, numero, err := models.ParseEpisodeID(episodeID)
	if err != nil {
		return nil, err
	}
	if proveedor != providerName {
		return nil, fmt.Errorf("animeav1: el id %q pertenece al proveedor %q", episodeID, proveedor)
	}
	if kind == models.KindMovie {
		// El id de película no lleva número (ParseEpisodeID devuelve 0),
		// pero la URL real del sitio es siempre /media/<slug>/1/…: así lo
		// pide el propio detalle, que trae un único episodio numerado 1.
		numero = 1
	}
	ruta := a.siteURL() + rutaDetalleMedia + url.PathEscape(slug) + "/" + strconv.Itoa(numero) + rutaDatosPagina
	resp, err := httpx.Get(ctx, a.Doer, ruta, httpx.ReqOptions{
		Provider:  providerName,
		Operation: "sources",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Un byte de más para detectar respuestas que superan el límite.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFuentesBytes+1))
	if err != nil {
		return nil, fmt.Errorf("leyendo el __data.json del episodio: %w", err)
	}
	if len(data) > maxFuentesBytes {
		return nil, fmt.Errorf("el __data.json del episodio pesa %d bytes y el máximo es %d", len(data), maxFuentesBytes)
	}
	fuentes, err := fuentesDeJSON(slug, ruta, data)
	if err != nil {
		return nil, err
	}
	return a.enriquecerConEmbeds(ctx, fuentes), nil
}

// fuentesDeJSON extrae el campo embeds del __data.json y lo convierte en
// fuentes. Sin embeds no hay con qué construirlas: error descriptivo, salvo
// que el cuerpo traiga el nodo de error 404 de SvelteKit, en cuyo caso se
// devuelve el error tipado de recurso no encontrado (mismo criterio que
// detalleDeJSON, T32).
func fuentesDeJSON(slug, ruta string, data []byte) ([]models.VideoSource, error) {
	datos, err := PageData(data)
	if err != nil {
		if paginaInexistente(data) {
			return nil, errorNoEncontrado("sources", ruta)
		}
		return nil, fmt.Errorf("parseando el __data.json del episodio de %q: %w", slug, err)
	}
	campo, existe := datos["embeds"]
	if !existe || campo == nil {
		if paginaInexistente(data) {
			return nil, errorNoEncontrado("sources", ruta)
		}
		return nil, fmt.Errorf("animeav1: el __data.json del episodio de %q no trae embeds", slug)
	}
	embeds, ok := campo.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("animeav1: embeds del episodio de %q es %T, quería un objeto con SUB/DUB", slug, campo)
	}
	fuentes, err := fuentesDeEmbeds(embeds)
	if err != nil {
		return nil, fmt.Errorf("animeav1: el episodio de %q: %w", slug, err)
	}
	return fuentes, nil
}

// grupoServidor reúne las fuentes de un mismo servidor para mantener SUB y
// DUB adyacentes y para poder mandar primero al del reproductor propio.
type grupoServidor struct {
	nombre string
	// conHLS marca al grupo con el manifiesto derivado: sale el primero,
	// igual que lo elegiría SelectBestSource (HLS > MP4 > iframe, T17).
	conHLS bool
	// primero es el orden de aparición de la primera fuente del grupo: el
	// desempate del resto de grupos, es decir, el orden del sitio.
	primero int
	fuentes []models.VideoSource
}

// fuentesDeEmbeds convierte las listas embeds.SUB y embeds.DUB en la lista
// final. Recorrido determinista (SUB antes que DUB y, dentro de cada
// idioma, el orden del sitio), deduplicación por recurso base + idioma y
// orden de salida: primero el grupo con HLS propio y después por orden de
// aparición, con SUB y DUB adyacentes dentro de cada servidor.
//
// Es un orden neutro y estable: no expresa preferencia de idioma, solo
// determinismo para el desempate de la selección automática (T17) y para
// que el resultado sea siempre el mismo. Un servidor con un solo idioma
// sale con ese solo.
func fuentesDeEmbeds(embeds map[string]any) ([]models.VideoSource, error) {
	vistas := make(map[string]struct{})
	porServidor := make(map[string]*grupoServidor)
	grupos := make([]*grupoServidor, 0, len(idiomasEmbeds))
	orden := 0

	for _, idioma := range idiomasEmbeds {
		lista, ok := embeds[idioma].([]any)
		if !ok {
			continue
		}
		for _, item := range lista {
			fuente, ok := fuenteDeEmbed(idioma, item)
			if !ok {
				continue
			}
			clave := claveFuente(fuente)
			if _, ya := vistas[clave]; ya {
				continue
			}
			vistas[clave] = struct{}{}

			grupo := porServidor[fuente.Server]
			if grupo == nil {
				grupo = &grupoServidor{nombre: fuente.Server, primero: orden}
				porServidor[fuente.Server] = grupo
				grupos = append(grupos, grupo)
			}
			if fuente.Type == models.ResourceHLS {
				grupo.conHLS = true
			}
			grupo.fuentes = append(grupo.fuentes, fuente)
			orden++
		}
	}
	if len(grupos) == 0 {
		return nil, errors.New("embeds sin ninguna entrada reproducible: se esperaban listas SUB/DUB con URLs http(s)")
	}

	sort.SliceStable(grupos, func(i, j int) bool {
		if grupos[i].conHLS != grupos[j].conHLS {
			return grupos[i].conHLS
		}
		return grupos[i].primero < grupos[j].primero
	})

	salida := make([]models.VideoSource, 0, orden)
	for _, grupo := range grupos {
		salida = append(salida, grupo.fuentes...)
	}
	return salida, nil
}

// fuenteDeEmbed convierte una entrada de embeds en models.VideoSource. La
// URL debe ser http/https con host —lo demás se descarta—; un embed del
// reproductor propio con id válido se convierte en HLS con la cabecera que
// exigen sus segmentos y cualquier otro queda como iframe con su URL tal
// cual (sin resolver: pendiente anotado en PROGRESO.md).
func fuenteDeEmbed(idioma string, item any) (models.VideoSource, bool) {
	obj, ok := item.(map[string]any)
	if !ok {
		return models.VideoSource{}, false
	}
	bruta := textoDe(obj["url"])
	if bruta == "" {
		return models.VideoSource{}, false
	}
	u, err := url.Parse(bruta)
	if err != nil || u.Host == "" {
		return models.VideoSource{}, false
	}
	if esquema := strings.ToLower(u.Scheme); esquema != "http" && esquema != "https" {
		return models.VideoSource{}, false
	}
	servidor := textoDe(obj["server"])
	if servidor == "" {
		servidor = u.Hostname()
	}
	if manifiesto, ok := manifiestoDe(u); ok {
		return models.VideoSource{
			URL:      manifiesto,
			Type:     models.ResourceHLS,
			Server:   servidor,
			Language: idioma,
			Headers:  map[string]string{cabeceraFetch: valorFetch},
		}, true
	}
	return models.VideoSource{
		URL:      bruta,
		Type:     models.ResourceIframe,
		Server:   servidor,
		Language: idioma,
	}, true
}

// manifiestoDe deriva la URL del manifiesto HLS del reproductor propio. La
// regla es la del propio JS del player (T30/P4): el manifiesto es el último
// segmento del path, siempre que el host sea el del jugador, el esquema
// https y el id exactamente 32 caracteres hexadecimales. Todo lo que no
// cumpla devuelve false y ese embed se queda como iframe: nunca se construye
// una URL a ciegas ni se pide /play/*.
func manifiestoDe(u *url.URL) (string, bool) {
	if !strings.EqualFold(u.Scheme, "https") || !strings.EqualFold(u.Host, hostJugador) {
		return "", false
	}
	if u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, rutaJugador) {
		return "", false
	}
	id := strings.TrimPrefix(u.Path, rutaJugador)
	if !idJugadorValido(id) {
		return "", false
	}
	return esquemaJugador + hostJugador + rutaManifiesto + id, true
}

// idJugadorValido comprueba la longitud y que el id sea hexadecimales. El
// id se reutiliza tal cual aparece en el path (el player también), así que
// no se normaliza su mayúsculas/minúsculas.
func idJugadorValido(id string) bool {
	if len(id) != largoIDJugador {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// claveFuente es la clave de deduplicación: el recurso base —esquema y host
// en minúsculas, path, sin query y con fragmento, el mismo criterio que
// Normalize (T16): la query solo cambia la firma temporal de un mismo vídeo
// y el fragmento de un embed es a menudo su clave de descifrado— más el
// idioma. La misma URL en SUB y en DUB son dos fuentes distintas, cada una
// con su Language (desviación respecto al "deduplica por URL" de la ficha,
// anotada en PROGRESO.md).
func claveFuente(f models.VideoSource) string {
	u, err := url.Parse(f.URL)
	if err != nil || u.Host == "" {
		return strings.ToLower(f.URL) + "\x00" + f.Language
	}
	clave := strings.ToLower(u.Scheme+"://"+u.Host) + u.Path
	if u.Fragment != "" {
		clave += "#" + u.Fragment
	}
	return clave + "\x00" + f.Language
}

// tiempoResolucionEmbeds acota cada resolución de embed (T61): es la
// petición corta al HTML del embed, no la descarga del vídeo. Un embed
// lento no puede bloquear el resto de fuentes del episodio.
const tiempoResolucionEmbeds = 8 * time.Second

// embedsResolvers devuelve los resolvers de embeds inyectados (New deja
// uno de MP4Upload con el doer del proveedor; una lista distinta permite a
// los tests inyectar un resolver falso o desactivar el enriquecimiento con
// una lista vacía).
func (a *AnimeAV1) embedsResolvers() []embeds.Resolver {
	return a.Embeds
}

// enriquecerConEmbeds añade, por cada iframe que algún resolver reconozca
// (T61), sus fuentes directas (mp4) conservando el iframe como respaldo.
// El idioma de la fuente original (SUB/DUB) se copia a la resuelta, de modo
// que SUB y DUB siguen siendo fuentes distintas. Un fallo del resolver no
// rompe el resto: se registra un aviso sin URL y se continúa. No se duplica
// un recurso ya presente (mismo recurso base + idioma, criterio de claveFuente).
func (a *AnimeAV1) enriquecerConEmbeds(ctx context.Context, fuentes []models.VideoSource) []models.VideoSource {
	resolvers := a.embedsResolvers()
	if len(resolvers) == 0 || len(fuentes) == 0 {
		return fuentes
	}
	if ctx == nil {
		ctx = context.Background()
	}
	salida := append([]models.VideoSource(nil), fuentes...)
	vistas := make(map[string]struct{}, len(salida))
	for _, f := range salida {
		vistas[claveFuente(f)] = struct{}{}
	}
	for _, base := range fuentes {
		if base.Type != models.ResourceIframe || strings.TrimSpace(base.URL) == "" {
			continue
		}
		resolver := resolverParaEmbeds(resolvers, base.URL)
		if resolver == nil {
			continue
		}
		resueltas, err := resolverConTimeout(ctx, resolver, base.URL)
		if err != nil {
			slog.Warn("animeav1: la resolución del embed falló; se conserva el iframe",
				"resolver", resolver.Name(),
				"server", base.Server,
				"err", logx.Redact(err.Error()))
			continue
		}
		for _, r := range resueltas {
			r.Language = base.Language // SUB/DUB se conservan por fuente
			if strings.TrimSpace(r.URL) == "" {
				continue
			}
			clave := claveFuente(r)
			if _, ya := vistas[clave]; ya {
				continue // mismo recurso con otra firma: no duplicar
			}
			vistas[clave] = struct{}{}
			salida = append(salida, r)
		}
	}
	return salida
}

// resolverParaEmbeds devuelve el primer resolver que reconozca la URL del
// iframe, o nil si ninguno lo hace. Comprobación barata y sin red.
func resolverParaEmbeds(resolvers []embeds.Resolver, url string) embeds.Resolver {
	for _, r := range resolvers {
		if r != nil && r.CanHandle(url) {
			return r
		}
	}
	return nil
}

// resolverConTimeout resuelve el embed con un tope corto, derivado del
// contexto del flujo: si el reproductor cancela, la petición al embed se
// corta igual. Referer vacío: el resolver no inventa la página que embebe
// (MP4Upload solo exige Referer del dominio para el CDN, no para el HTML).
func resolverConTimeout(ctx context.Context, r embeds.Resolver, url string) ([]models.VideoSource, error) {
	ctxRes, cancel := context.WithTimeout(ctx, tiempoResolucionEmbeds)
	defer cancel()
	return r.Resolve(ctxRes, url, "")
}
