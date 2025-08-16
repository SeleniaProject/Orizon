package netstack

import (
	"crypto/tls"
	"net"
	"strings"
)

// TLSDial wraps tls.Dial with a minimal config if none provided.
func TLSDial(network, addr string, cfg *tls.Config) (net.Conn, error) {
	// Strengthen defaults: prefer TLS 1.3 by default and ensure ServerName is set.
	if cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS13}
	} else if cfg.MinVersion == 0 || cfg.MinVersion < tls.VersionTLS13 {
		// Enforce at least TLS 1.3 when not explicitly set higher.
		cfg = cfg.Clone()
		cfg.MinVersion = tls.VersionTLS13
	}
	if cfg.ServerName == "" {
		// Derive SNI from addr if host:port format is provided.
		host := addr
		if idx := strings.LastIndexByte(addr, ':'); idx > 0 {
			host = addr[:idx]
		}
		// IPv6 literals in brackets like [::1]:443 → strip brackets
		host = strings.TrimPrefix(host, "[")
		host = strings.TrimSuffix(host, "]")
		if host != "" {
			if cfg == nil {
				cfg = &tls.Config{}
			} else {
				cfg = cfg.Clone()
			}
			cfg.ServerName = host
		}
	}
	return tls.Dial(network, addr, cfg)
}

// TLSServer wraps a net.Listener with TLS.
func TLSServer(ln net.Listener, cfg *tls.Config) net.Listener {
	// Strengthen server defaults to TLS 1.3 minimum when not specified.
	if cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS13}
	} else if cfg.MinVersion == 0 || cfg.MinVersion < tls.VersionTLS13 {
		c := cfg.Clone()
		c.MinVersion = tls.VersionTLS13
		cfg = c
	}
	return tls.NewListener(ln, cfg)
}
