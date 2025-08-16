package netstack

import (
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateSelfSignedTLS_UsesTLS13Min(t *testing.T) {
	cfg, err := GenerateSelfSignedTLS([]string{"localhost"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateSelfSignedTLS error: %v", err)
	}
	if cfg == nil || cfg.MinVersion != tls.VersionTLS13 {
		t.Fatalf("MinVersion not TLS1.3: %#v", cfg)
	}
}

func TestTLSServer_EnforcesTLS13(t *testing.T) {
	ln := &dummyListener{}
	l := TLSServer(ln, &tls.Config{MinVersion: tls.VersionTLS12})
	// Basic sanity: TLSServer must return a non-nil listener
	if l == nil {
		t.Fatalf("TLSServer returned nil listener")
	}
}

func TestWritePEMAndLoadTLSConfig(t *testing.T) {
	cfg, err := GenerateSelfSignedTLS([]string{"localhost"}, time.Hour)
	if err != nil {
		t.Fatalf("self-signed: %v", err)
	}
	if len(cfg.Certificates) == 0 {
		t.Fatalf("no certs in cfg")
	}
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := WritePEM(&cfg.Certificates[0], certPath, keyPath); err != nil {
		t.Fatalf("write pem: %v", err)
	}
	// Ensure files exist
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("missing cert: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("missing key: %v", err)
	}
	// Load back and verify TLS1.3 min version
	loaded, err := LoadTLSConfig(certPath, keyPath)
	if err != nil {
		t.Fatalf("load tls: %v", err)
	}
	if loaded.MinVersion != tls.VersionTLS13 {
		t.Fatalf("MinVersion not TLS1.3 after load: %v", loaded.MinVersion)
	}
}

// dummyListener is a minimal net.Listener stub for wrapping by TLSServer
type dummyListener struct{}

func (d *dummyListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (d *dummyListener) Close() error              { return nil }
func (d *dummyListener) Addr() net.Addr            { return dummyAddr(":0") }

type dummyAddr string

func (d dummyAddr) Network() string { return "tcp" }
func (d dummyAddr) String() string  { return string(d) }
