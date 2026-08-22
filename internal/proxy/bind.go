package proxy

import (
	"fmt"
	"net"
)

// EffectiveBindAddr returns the UDP address to bind. When the configured listen
// host is loopback but the upstream login server is external, binding loopback
// prevents sendto from reaching the upstream (EADDRNOTAVAIL on macOS). In that
// case we bind all interfaces on the same port; EQ still connects via 127.0.0.1.
func EffectiveBindAddr(listen string, upstream *net.UDPAddr) (string, bool) {
	listenAddr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil || upstream == nil {
		return listen, false
	}
	if !isLoopbackIP(listenAddr.IP) || isLoopbackIP(upstream.IP) {
		return listen, false
	}
	port := listenAddr.Port
	if port == 0 {
		return "0.0.0.0:0", true
	}
	return fmt.Sprintf("0.0.0.0:%d", port), true
}

func isLoopbackIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
