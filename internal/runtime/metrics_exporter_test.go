package runtime

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStartMetricsServer_ServesMetrics(t *testing.T) {
	collectors := map[string]MetricFunc{
		"testCollector": func() map[string]float64 {
			return map[string]float64{"requests_total": 123, "latency_ms": 4.5}
		},
	}
	addr, stop, err := StartMetricsServer(":0", collectors)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = stop(context.Background()) }()

	cli := &http.Client{Timeout: 2 * time.Second}
	resp, err := cli.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %v", resp.Status)
	}
	// Read a few lines and ensure our metric names appear
	rd := bufio.NewReader(resp.Body)
	var got string
	for i := 0; i < 5; i++ {
		line, _, err := rd.ReadLine()
		if err != nil {
			break
		}
		got += string(line) + "\n"
	}
	if !strings.Contains(got, "testCollector_requests_total") {
		t.Fatalf("missing metric name, got: %q", got)
	}
}

func TestStartMetricsTLSServer_ServesMetrics(t *testing.T) {
	// Create a self-signed pair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: bigIntOne(),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("crt: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("pair: %v", err)
	}
	srvCfg := &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS13}

	collectors := map[string]MetricFunc{"c": func() map[string]float64 { return map[string]float64{"x": 1} }}
	addr, stop, err := StartMetricsTLSServer("127.0.0.1:0", collectors, srvCfg)
	if err != nil {
		t.Fatalf("start tls: %v", err)
	}
	defer func() { _ = stop(context.Background()) }()

	// Insecure client for self-signed test cert
	cli := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}, Timeout: 2 * time.Second}
	resp, err := cli.Get("https://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %v", resp.Status)
	}
}

func TestStartMetricsServerWithAuth_RejectsWithoutToken(t *testing.T) {
	collectors := map[string]MetricFunc{"c": func() map[string]float64 { return map[string]float64{"x": 1} }}
	addr, stop, err := StartMetricsServerWithAuth("127.0.0.1:0", collectors, "secret")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = stop(context.Background()) }()
	cli := &http.Client{Timeout: 2 * time.Second}
	resp, err := cli.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %v", resp.Status)
	}
}

func TestStartMetricsServerWithAuth_AllowsWithToken(t *testing.T) {
	collectors := map[string]MetricFunc{"c": func() map[string]float64 { return map[string]float64{"x": 1} }}
	addr, stop, err := StartMetricsServerWithAuth("127.0.0.1:0", collectors, "secret")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = stop(context.Background()) }()
	req, _ := http.NewRequest("GET", "http://"+addr+"/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %v", resp.Status)
	}
}

func TestStartMetricsTLSServerWithAuth_QueryToken(t *testing.T) {
	// Generate self-signed
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), DNSNames: []string{"localhost"}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("crt: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("pair: %v", err)
	}
	srvCfg := &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS13}

	collectors := map[string]MetricFunc{"c": func() map[string]float64 { return map[string]float64{"x": 1} }}
	addr, stop, err := StartMetricsTLSServerWithAuth("127.0.0.1:0", collectors, srvCfg, "tok")
	if err != nil {
		t.Fatalf("start tls: %v", err)
	}
	defer func() { _ = stop(context.Background()) }()

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}
	cli := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	resp, err := cli.Get("https://" + addr + "/metrics?access_token=tok")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %v", resp.Status)
	}
}

// bigIntOne returns a new big.Int(1). Keep local to avoid extra imports for a single constant.
func bigIntOne() *big.Int { return big.NewInt(1) }

func TestSanitizeMetricToken(t *testing.T) {
	in := " metric name (bad)!"
	out := sanitizeMetricToken(in)
	if strings.ContainsAny(out, " !()") {
		t.Fatalf("token not sanitized: %q", out)
	}
	if out == "" {
		t.Fatalf("empty token")
	}
}
