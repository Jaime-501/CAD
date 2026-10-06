package stream

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
)

// TestIPPublica: la anti-SSRF solo acepta IPs globales. Se rechazan
// loopback, privadas, link-local, no especificadas y multicast, en IPv4 e
// IPv6 (incluidas las variantes mapeadas).
func TestIPPublica(t *testing.T) {
	casos := []struct {
		ip     string
		queria bool
	}{
		// loopback
		{"127.0.0.1", false}, {"127.1.2.3", false}, {"::1", false},
		{"::ffff:127.0.0.1", false},
		// privadas
		{"10.0.0.1", false}, {"172.16.0.1", false}, {"172.31.255.255", false},
		{"192.168.1.1", false}, {"192.168.0.0", false},
		{"fc00::1", false}, {"fd00::1", false},
		// link-local
		{"169.254.1.1", false}, {"fe80::1", false}, {"::ffff:169.254.0.1", false},
		// no especificadas
		{"0.0.0.0", false}, {"0.1.2.3", false}, {"::", false},
		// multicast
		{"224.0.0.1", false}, {"239.255.255.250", false}, {"ff02::1", false},
		// globales: se admiten
		{"1.1.1.1", true}, {"8.8.8.8", true}, {"93.184.216.34", true},
		{"172.32.0.1", true}, {"11.0.0.1", true},
		{"2606:4700:4700::1111", true},
	}
	for _, c := range casos {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("IP de prueba no interpretable: %q", c.ip)
		}
		if got := ipPublica(ip); got != c.queria {
			t.Errorf("ipPublica(%s) = %v, quería %v", c.ip, got, c.queria)
		}
	}
	if ipPublica(nil) {
		t.Error("ipPublica(nil) debía ser false")
	}
}

// TestControlRedPorDefectoYPermitirRedPrivada: el valor por defecto es el
// restrictivo (Control activo que rechaza lo no público) y la opción
// PermitirRedPrivada anula SOLO ese control.
func TestControlRedPorDefectoYPermitirRedPrivada(t *testing.T) {
	control := controlRed(false)
	if control == nil {
		t.Fatal("con la opción por defecto el Control debe estar activo")
	}
	bloqueadas := []string{
		"127.0.0.1:80", "[::1]:443", "10.0.0.5:8080", "192.168.1.10:443",
		"169.254.169.254:80", "0.0.0.0:80", "224.0.0.1:80",
	}
	for _, dir := range bloqueadas {
		err := control("tcp", dir, nil)
		if err == nil {
			t.Errorf("el control no bloqueó %s", dir)
			continue
		}
		if !errors.Is(err, ErrDestinoBloqueado) {
			t.Errorf("error para %s = %v, debía envolver ErrDestinoBloqueado", dir, err)
		}
	}
	if err := control("tcp", "1.1.1.1:443", nil); err != nil {
		t.Errorf("el control rechazó una IP pública: %v", err)
	}

	if permiso := controlRed(true); permiso != nil {
		t.Error("PermitirRedPrivada debía anular el Control (nil)")
	}
}

// TestNuevoClienteAplicaElControl: el cliente que construye el proxy lleva
// ese Control en su dialer, así que no llega a conectar con un destino
// privado (y con la opción solo se desactiva el control, el dialer sigue).
func TestNuevoClienteAplicaElControl(t *testing.T) {
	restrictivo, ok := nuevoCliente(false).Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transporte restrictivo = %T, esperaba *http.Transport", nuevoCliente(false).Transport)
	}
	if restrictivo.DialContext == nil {
		t.Fatal("el cliente restrictivo no tiene DialContext con control")
	}
	for _, dir := range []string{"127.0.0.1:9", "[::1]:9", "10.1.2.3:80"} {
		conn, err := restrictivo.DialContext(context.Background(), "tcp", dir)
		if conn != nil {
			_ = conn.Close()
			t.Errorf("el dial a %s devolvió conexión: debía bloquearse", dir)
		}
		if !errors.Is(err, ErrDestinoBloqueado) {
			t.Errorf("dial a %s = %v, debía bloquearse con ErrDestinoBloqueado", dir, err)
		}
	}

	permisivo, ok := nuevoCliente(true).Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transporte permisivo = %T, esperaba *http.Transport", nuevoCliente(true).Transport)
	}
	if permisivo.DialContext == nil {
		t.Fatal("el cliente permisivo perdió el DialContext")
	}
	if restrictivo == permisivo {
		t.Fatal("cada cliente debe tener su propio transporte")
	}
}
