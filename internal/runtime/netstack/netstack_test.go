package netstack

import (
    "crypto/tls"
    "io"
    "net"
    "testing"
    "time"
)

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
    n, err := cli.WriteTo(msg, srv.conn.LocalAddr().(*net.UDPAddr))
    if err != nil || n != len(msg) { t.Fatalf("write: %v n=%d", err, n) }
    buf := make([]byte, 16)
    _ = srv.conn.SetReadDeadline(time.Now().Add(time.Second))
    n, addr, err := srv.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    if string(buf[:n]) != "hi" { t.Fatalf("got %q", string(buf[:n])) }
    // reply
    _, _ = srv.WriteTo([]byte("ok"), addr)
    _ = cli.conn.SetReadDeadline(time.Now().Add(time.Second))
    n, _, err = cli.conn.ReadFromUDP(buf)
    if err != nil { t.Fatal(err) }
    if string(buf[:n]) != "ok" { t.Fatalf("got %q", string(buf[:n])) }
}

func TestTLS_Wrap(t *testing.T) {
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer ln.Close()
    // self-signed insecure config for local test use only
    cfg := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
    tln := TLSServer(ln, cfg)
    go func(){
        c, _ := tln.Accept()
        if c != nil { _ = c.Close() }
    }()
    c, err := TLSDial("tcp", ln.Addr().String(), cfg)
    if err != nil { t.Fatal(err) }
    _ = c.Close()
}


