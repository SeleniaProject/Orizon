package netstack

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/tls"
    "crypto/x509"
    "encoding/pem"
    "io"
    "math/big"
    "net"
    "testing"
    "time"
)

func selfSignedPair(t *testing.T) (*tls.Config, *tls.Config) {
    t.Helper()
    // generate a simple self-signed certificate for localhost testing
    priv, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil { t.Fatalf("key gen: %v", err) }
    tmpl := &x509.Certificate{
        SerialNumber: big.NewInt(1),
        NotBefore:    time.Now().Add(-time.Hour),
        NotAfter:     time.Now().Add(24 * time.Hour),
        KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
        DNSNames:     []string{"localhost"},
    }
    der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
    if err != nil { t.Fatalf("crt gen: %v", err) }
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
    pair, err := tls.X509KeyPair(certPEM, keyPEM)
    if err != nil { t.Fatalf("pair: %v", err) }
    serverCfg := &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
    clientCfg := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
    return serverCfg, clientCfg
}

func TestTCP_Echo(t *testing.T) {
    srv := NewTCPServer("127.0.0.1:0")
    err := srv.Start(nil, func(c net.Conn){
        defer c.Close()
        buf := make([]byte, 4)
        _, _ = io.ReadFull(c, buf)
        _, _ = c.Write(buf)
    })
    if err != nil { t.Fatal(err) }
    defer srv.Stop()
    addr := srv.ln.Addr().String()
    c, err := DialTCP(addr, time.Second)
    if err != nil { t.Fatal(err) }
    defer c.Close()
    _, _ = c.Write([]byte("ping"))
    buf := make([]byte, 4)
    if _, err := io.ReadFull(c, buf); err != nil { t.Fatal(err) }
    if string(buf) != "ping" { t.Fatalf("got %q", string(buf)) }
}

func TestUDP_Roundtrip(t *testing.T) {
    srv, err := ListenUDP("127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer srv.Close()
    cli, err := DialUDP(srv.conn.LocalAddr().String())
    if err != nil { t.Fatal(err) }
    defer cli.Close()
    msg := []byte("hi")
    // since cli is connected, use Write (not WriteTo)
    n, err := cli.Write(msg)
    if err != nil || n != len(msg) { t.Fatalf("write: %v n=%d", err, n) }
    buf := make([]byte, 16)
    _ = srv.conn.SetReadDeadline(time.Now().Add(time.Second))
    n, addr, err := srv.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    if string(buf[:n]) != "hi" { t.Fatalf("got %q", string(buf[:n])) }
    // reply
    _, _ = srv.WriteTo([]byte("ok"), addr)
    _ = cli.conn.SetReadDeadline(time.Now().Add(time.Second))
    n, err = cli.Read(buf)
    if err != nil { t.Fatal(err) }
    if string(buf[:n]) != "ok" { t.Fatalf("got %q", string(buf[:n])) }
}

func TestTLS_Wrap(t *testing.T) {
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer ln.Close()
    srvCfg, cliCfg := selfSignedPair(t)
    tln := TLSServer(ln, srvCfg)
    go func(){
        c, _ := tln.Accept()
        if c != nil { _ = c.Close() }
    }()
    c, err := TLSDial("tcp", ln.Addr().String(), cliCfg)
    if err != nil { t.Fatal(err) }
    _ = c.Close()
}


