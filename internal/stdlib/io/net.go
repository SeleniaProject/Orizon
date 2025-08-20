package io

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/orizon-lang/orizon/internal/runtime/netstack"
)

// DialTCP dials TCP with timeout.
func DialTCP(addr string, timeout time.Duration) (net.Conn, error) {
	return netstack.DialTCP(addr, timeout)
}

// TCPServer is re-exported for convenience.
type TCPServer = netstack.TCPServer

// NewTCPServer creates a server.
func NewTCPServer(addr string) *TCPServer { return netstack.NewTCPServer(addr) }

// TLSDial dials TLS with a given config.
func TLSDial(network, addr string, cfg *tls.Config) (net.Conn, error) {
	return netstack.TLSDial(network, addr, cfg)
}

// TLSServer wraps a listener with TLS server settings.
func TLSServer(ln net.Listener, cfg *tls.Config) net.Listener { return netstack.TLSServer(ln, cfg) }

// UDP exports.
type UDPEndpoint = netstack.UDPEndpoint

func ListenUDP(addr string) (*UDPEndpoint, error) { return netstack.ListenUDP(addr) }
func DialUDP(addr string) (*UDPEndpoint, error)   { return netstack.DialUDP(addr) }

// TCPMetrics exposes runtime metrics.
func TCPMetrics() map[string]uint64 { return netstack.TCPMetrics() }

// HTTP/3 exports.
type (
	HTTP3Server  = netstack.HTTP3Server
	HTTP3Options = netstack.HTTP3Options
)

func NewHTTP3Server(addr string, tlsCfg *tls.Config, h http.Handler) *HTTP3Server {
	return netstack.NewHTTP3Server(addr, tlsCfg, h)
}

func NewHTTP3ServerWithOptions(addr string, tlsCfg *tls.Config, h http.Handler, opts HTTP3Options) *HTTP3Server {
	return netstack.NewHTTP3ServerWithOptions(addr, tlsCfg, h, opts)
}

func HTTP3Client(tlsCfg *tls.Config, timeout time.Duration) *http.Client {
	return netstack.HTTP3Client(tlsCfg, timeout)
}

func HTTP3ClientWithOptions(tlsCfg *tls.Config, timeout time.Duration, opts HTTP3Options) *http.Client {
	return netstack.HTTP3ClientWithOptions(tlsCfg, timeout, opts)
}
func ShutdownHTTP3(c *http.Client) { netstack.ShutdownHTTP3(c) }

// Certificate utilities (dev/test convenience).
func GenerateSelfSignedTLS(hosts []string, validFor time.Duration) (*tls.Config, error) {
	return netstack.GenerateSelfSignedTLS(hosts, validFor)
}

func LoadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	return netstack.LoadTLSConfig(certFile, keyFile)
}
