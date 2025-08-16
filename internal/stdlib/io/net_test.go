package io

import (
	"crypto/tls"
	stdio "io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestTCP_Echo(t *testing.T) {
	srv := NewTCPServer("127.0.0.1:0")
	err := srv.Start(nil, func(c net.Conn) {
		defer c.Close()
		buf := make([]byte, 4)
		_, _ = stdio.ReadFull(c, buf)
		_, _ = c.Write(buf)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Stop() })
	addr := srv.Addr()
	c, err := DialTCP(addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, _ = c.Write([]byte("ping"))
	buf := make([]byte, 4)
	if _, err := stdio.ReadFull(c, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "ping" {
		t.Fatalf("got %q", string(buf))
	}
}

func TestUDP_Roundtrip(t *testing.T) {
	srv, err := ListenUDP("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	cli, err := DialUDP(srv.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	msg := []byte("hi")
	n, err := cli.Write(msg)
	if err != nil || n != len(msg) {
		t.Fatalf("write: %v n=%d", err, n)
	}
	buf := make([]byte, 16)
	_ = srv.SetReadDeadline(time.Now().Add(time.Second))
	n, addr, err := srv.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "hi" {
		t.Fatalf("got %q", string(buf[:n]))
	}
	_, _ = srv.WriteTo([]byte("ok"), addr)
	_ = cli.SetReadDeadline(time.Now().Add(time.Second))
	n, err = cli.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "ok" {
		t.Fatalf("got %q", string(buf[:n]))
	}
}

func TestTLS_Wrap(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	cfg, err := GenerateSelfSignedTLS([]string{"localhost"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tln := TLSServer(ln, cfg)
	go func() {
		c, _ := tln.Accept()
		if c != nil {
			if tc, ok := c.(*tls.Conn); ok {
				_ = tc.Handshake()
			}
			_ = c.Close()
		}
	}()
	client := cfg.Clone()
	client.InsecureSkipVerify = true
	c, err := TLSDial("tcp", ln.Addr().String(), client)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
}

func TestHTTP3_ServerClient(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	cfg, err := GenerateSelfSignedTLS([]string{"localhost"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	s := NewHTTP3Server("127.0.0.1:0", cfg, h)
	addr, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop() })
	// Use an insecure client for self-signed local testing
	cli := HTTP3Client(&tls.Config{InsecureSkipVerify: true}, 3*time.Second)
	resp, err := cli.Get("https://" + addr + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
