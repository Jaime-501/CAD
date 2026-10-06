package stream

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"

	"miapp/internal/httpx"
)

// ErrDestinoBloqueado es la causa sentinela de la anti-SSRF: el destino (o
// un redirect hacia él) resuelve a una IP que no es pública. El error del
// dialer la envuelve, así que sirve para errors.Is en el manejo de errores.
var ErrDestinoBloqueado = errors.New("destino bloqueado por la anti-SSRF")

// timeouts del cliente propio del proxy. No hay Timeout global: el
// streaming de un vídeo dura minutos y se cancela por contexto (si el
// reproductor cierra la conexión, se cancela la petición upstream).
const (
	timeoutConexion  = 15 * time.Second
	timeoutCabezales = 30 * time.Second
)

// ipPublica informa si una IP ya resuelta es apta como destino: solo
// direcciones globales. Se rechazan loopback, privadas, link-local, no
// especificadas y multicast, en IPv4 e IPv6 (y sus variantes mapeadas).
func ipPublica(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	// 0.0.0.0/8 ("esta red") solo lo cubre IsUnspecified en su primera IP.
	if v4 := ip.To4(); v4 != nil && v4[0] == 0 {
		return false
	}
	return true
}

// controlRed construye la función net.Dialer.Control del cliente interno.
// Se invoca con la dirección ya resuelta, antes de usar la conexión, y lo
// mismo en cada redirect (el transporte resuelve de nuevo cada salto). Con
// permitir=true devuelve nil y el control queda anulado: es la opción
// PermitirRedPrivada, exclusiva de los tests.
func controlRed(permitir bool) func(network, address string, _ syscall.RawConn) error {
	if permitir {
		return nil
	}
	return func(network, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			host = address
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("%w: %q no es una IP", ErrDestinoBloqueado, host)
		}
		if !ipPublica(ip) {
			return fmt.Errorf("%w: %s no es pública", ErrDestinoBloqueado, ip)
		}
		return nil
	}
}

// nuevoCliente construye el http.Client del proxy: su propio transporte con
// el Dialer que lleva el control de red. Proxy de entorno desactivado a
// propósito: si se usara, el control validaría la IP del proxy y no la del
// destino.
func nuevoCliente(permitirRedPrivada bool) *http.Client {
	base, _ := httpx.NewTransport().(*http.Transport)
	if base == nil {
		if def, ok := http.DefaultTransport.(*http.Transport); ok {
			base = def
		} else {
			base = &http.Transport{}
		}
	}
	seguro := base.Clone()
	seguro.Proxy = nil
	seguro.ResponseHeaderTimeout = timeoutCabezales
	seguro.DialContext = (&net.Dialer{
		Timeout:   timeoutConexion,
		KeepAlive: 30 * time.Second,
		Control:   controlRed(permitirRedPrivada),
	}).DialContext
	return &http.Client{Transport: seguro}
}
