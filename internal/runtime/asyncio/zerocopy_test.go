package asyncio

import (
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

func TestZeroCopy_CopyConnToConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverReady := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		// echo loop for a single message
		buf := make([]byte, 8)
		_, _ = io.ReadFull(c, buf)
		_, _ = c.Write(buf)
	}()
	serverReady <- ln.Addr().String()

	addr := <-serverReady
	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// twin connection to act as source
	srcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srcLn.Close()

	go func() {
		sc, err := srcLn.Accept()
		if err != nil {
			return
		}
		defer sc.Close()
		_, _ = sc.Write([]byte("12345678"))
	}()

	srcConn, err := net.Dial("tcp", srcLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer srcConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := CopyConnToConn(ctx, client, srcConn); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 8)
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "12345678" {
		t.Fatalf("unexpected echo: %q", string(buf))
	}
}

func TestZeroCopy_CopyFileToConn(t *testing.T) {
	f, err := os.CreateTemp("", "zc-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString("abcdef")
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	recv := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		b := make([]byte, 6)
		_, _ = io.ReadFull(c, b)
		recv <- string(b)
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := CopyFileToConn(ctx, conn, f); err != nil {
		t.Fatal(err)
	}

	select {
	case s := <-recv:
		if s != "abcdef" {
			t.Fatalf("unexpected: %q", s)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
