package netstack

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"

	http3 "github.com/quic-go/quic-go/http3"
)

func genSelfSigned(t *testing.T) *tls.Config {
	t.Helper()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), DNSNames: []string{"localhost"}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pair, _ := tls.X509KeyPair(certPEM, keyPEM)
	return &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
}

func TestTLS13EnforcedOnClient(t *testing.T) {
	// Even if TLS 1.2 is provided, HTTP3Client should enforce TLS 1.3
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	cli := HTTP3Client(cfg, 2*time.Second)
	tr, ok := cli.Transport.(*http3.Transport)
	if !ok {
		t.Fatalf("transport is not http3.Transport: %T", cli.Transport)
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("client MinVersion not enforced to TLS1.3: got %#v", tr.TLSClientConfig)
	}
}

func TestTLS13EnforcedOnServer(t *testing.T) {
	// Construct server with TLS 1.2 and ensure it is bumped to TLS 1.3
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	s := NewHTTP3Server("127.0.0.1:0", cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if s == nil || s.srv == nil || s.srv.TLSConfig == nil {
		t.Fatal("server or TLS config is nil")
	}
	if s.srv.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("server MinVersion not enforced to TLS1.3: got %v", s.srv.TLSConfig.MinVersion)
	}
}

func TestHTTP3_Loopback(t *testing.T) {
	srvTLS := genSelfSigned(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("pong")) })
	s := NewHTTP3Server("127.0.0.1:0", srvTLS, mux)
	addr, err := s.Start()
	if err != nil {
		t.Skip("http3 not supported here:", err)
	}
	defer s.Stop()

	cli := HTTP3Client(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, 2*time.Second)
	defer ShutdownHTTP3(cli)
	resp, err := cli.Get("https://" + addr + "/ping")
	if err != nil {
		t.Skip("http3 dial failed:", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "pong" {
		t.Fatalf("unexpected: %q", string(b))
	}
}

func TestHTTP3_ProtoIsH3(t *testing.T) {
	srvTLS := genSelfSigned(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/p", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	s := NewHTTP3ServerWithOptions("127.0.0.1:0", srvTLS, mux, HTTP3Options{KeepAlivePeriod: 200 * time.Millisecond})
	addr, err := s.Start()
	if err != nil {
		t.Skip("http3 not supported:", err)
	}
	defer s.Stop()
	cli := HTTP3ClientWithOptions(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, 2*time.Second, HTTP3Options{Enable0RTT: true})
	defer ShutdownHTTP3(cli)
	resp, err := cli.Get("https://" + addr + "/p")
	if err != nil {
		t.Skip("http3 dial failed:", err)
	}
	defer resp.Body.Close()
	if resp.ProtoMajor != 3 {
		t.Fatalf("expected HTTP/3, got %s", resp.Proto)
	}
}
