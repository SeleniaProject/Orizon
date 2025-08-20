//go:build linux.
// +build linux.

package asyncio

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestSpliceConnToConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverReady := make(chan net.Conn, 1)
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			serverReady <- c
		}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	srv := <-serverReady
	defer srv.Close()

	// Create a loopback copier goroutine: splice client->server back to client would require echo; instead write from client and splice server->discard.
	// For test, create another loopback connection as destination.
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln2.Close()
	dstReady := make(chan net.Conn, 1)
	go func() {
		d, _ := ln2.Accept()
		if d != nil {
			dstReady <- d
		}
	}()
	dstClient, _ := net.Dial("tcp", ln2.Addr().String())
	defer dstClient.Close()
	dst := <-dstReady
	defer dst.Close()

	// write data on server side source connection.
	go func() { _, _ = client.Write([]byte("hello")) }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	n, err := SpliceConnToConn(ctx, dst, srv, 5)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected 5, got %d", n)
	}
}
