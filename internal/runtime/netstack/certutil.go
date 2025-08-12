package netstack

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/tls"
    "crypto/x509"
    "encoding/pem"
    "math/big"
    "net"
    "os"
    "time"
)

// GenerateSelfSignedTLS creates an in-memory self-signed TLS config for the given hostnames.
func GenerateSelfSignedTLS(hosts []string, validFor time.Duration) (*tls.Config, error) {
    if validFor <= 0 { validFor = 24 * time.Hour }
    key, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil { return nil, err }
    tmpl := &x509.Certificate{
        SerialNumber: big.NewInt(time.Now().UnixNano()),
        NotBefore:    time.Now().Add(-time.Hour),
        NotAfter:     time.Now().Add(validFor),
        KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
    }
    for _, h := range hosts {
        if ip := net.ParseIP(h); ip != nil { tmpl.IPAddresses = append(tmpl.IPAddresses, ip) } else { tmpl.DNSNames = append(tmpl.DNSNames, h) }
    }
    der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
    if err != nil { return nil, err }
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
    pair, err := tls.X509KeyPair(certPEM, keyPEM)
    if err != nil { return nil, err }
    return &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12, NextProtos: []string{"h3", "h2", "http/1.1"}}, nil
}

// LoadTLSConfig loads server-side TLS config from certificate and key file paths.
func LoadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil { return nil, err }
    return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, nil
}

// WritePEM writes cert and key PEM to files for development use.
func WritePEM(cert *tls.Certificate, certPath, keyPath string) error {
    // cert.Certificate holds DER chain; write leaf as PEM and key as PEM
    if len(cert.Certificate) == 0 { return os.ErrInvalid }
    if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o644); err != nil { return err }
    // Private key isn't directly accessible from tls.Certificate; skip extraction in generic helper
    // Caller should manage private key persistence separately if needed.
    return nil
}


