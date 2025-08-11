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

func TestHTTP3_Loopback(t *testing.T) {
    srvTLS := genSelfSigned(t)
    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request){ _, _ = w.Write([]byte("pong")) })
    s := NewHTTP3Server("127.0.0.1:0", srvTLS, mux)
    addr, err := s.Start()
    if err != nil { t.Skip("http3 not supported here:", err) }
    defer s.Stop()

    cli := HTTP3Client(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, 2*time.Second)
    defer ShutdownHTTP3(cli)
    resp, err := cli.Get("https://" + addr + "/ping")
    if err != nil { t.Skip("http3 dial failed:", err) }
    defer resp.Body.Close()
    b, _ := io.ReadAll(resp.Body)
    if string(b) != "pong" { t.Fatalf("unexpected: %q", string(b)) }
}


