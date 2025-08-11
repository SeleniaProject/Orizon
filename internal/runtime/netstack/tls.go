package netstack

import (
	"crypto/tls"
	"net"
)

// TLSDial wraps tls.Dial with a minimal config if none provided.
func TLSDial(network, addr string, cfg *tls.Config) (net.Conn, error) {
	if cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return tls.Dial(network, addr, cfg)
}

// TLSServer wraps a net.Listener with TLS.
func TLSServer(ln net.Listener, cfg *tls.Config) net.Listener {
	if cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return tls.NewListener(ln, cfg)
}
