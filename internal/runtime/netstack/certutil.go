package netstack

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"time"
)

// GenerateSelfSignedTLS creates an in-memory self-signed TLS config for the given hostnames.
func GenerateSelfSignedTLS(hosts []string, validFor time.Duration) (*tls.Config, error) {
	if validFor <= 0 {
		validFor = 24 * time.Hour
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	// Use TLS 1.3 as a unified secure baseline and advertise common protocols
	return &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS13, NextProtos: []string{"h3", "h2", "http/1.1"}}, nil
}

// LoadTLSConfig loads server-side TLS config from certificate and key file paths.
func LoadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	// Use TLS 1.3 as a unified secure baseline for loaded certificates
	return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13}, nil
}

// WritePEM writes cert and key PEM to files for development use.
func WritePEM(cert *tls.Certificate, certPath, keyPath string) error {
	// Write leaf certificate
	if cert == nil || len(cert.Certificate) == 0 {
		return os.ErrInvalid
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o644); err != nil {
		return err
	}
	// Marshal private key if present
	switch k := cert.PrivateKey.(type) {
	case *rsa.PrivateKey:
		keyDER := x509.MarshalPKCS1PrivateKey(k)
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})
		return os.WriteFile(keyPath, keyPEM, 0o600)
	default:
		return errors.New("unsupported or missing private key for PEM export")
	}
}
