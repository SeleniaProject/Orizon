//go:build windows && iocp.
// +build windows,iocp.

package asyncio

import (
	"context"
	"net"
	"testing"
	"time"
)

// Verifies that the internal zero-byte WSARecv posts and completes upon data arrival,.
// surfacing a Readable event to the callback.
func TestIOCP_ZeroByteRecvTriggersReadable(t *testing.T) {
	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	srvCh := make(chan net.Conn, 1)
	go func() {
		c, e := ln.Accept()
		if e == nil {
			srvCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvCh
	defer srv.Close()

	got := make(chan struct{}, 1)
	if err := p.Register(cli, []EventType{Readable}, func(ev Event) {
		if ev.Type == Readable {
			select {
			case got <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		t.Fatal(err)
	}

	// Send a single byte to complete the pending zero-byte recv.
	if _, err := srv.Write([]byte{0x42}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Readable after server write")
	}
}

// Verifies that the internal zero-byte WSASend posts triggers a Writable event.
// shortly after registration, even with no outbound data queued.
func TestIOCP_ZeroByteSendTriggersWritable(t *testing.T) {
	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	srvCh := make(chan net.Conn, 1)
	go func() {
		c, e := ln.Accept()
		if e == nil {
			srvCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvCh
	defer srv.Close()

	got := make(chan struct{}, 1)
	if err := p.Register(cli, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			select {
			case got <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for initial Writable event")
	}
}
