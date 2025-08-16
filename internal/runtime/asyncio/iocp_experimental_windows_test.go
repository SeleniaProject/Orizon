//go:build windows && iocp
// +build windows,iocp

package asyncio

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestIOCPPoller_Readable_And_Deregister(t *testing.T) {
	// Reuse generic helpers to ensure parity with other pollers.
	testPollerReadableAndDeregister(t, NewIOCPPoller)
}

func TestIOCPPoller_ErrorOnClose(t *testing.T) {
	// Reuse generic helpers to ensure consistent error signaling.
	testPollerErrorOnClose(t, NewIOCPPoller)
}

func TestIOCPPoller_Writable_Throttled(t *testing.T) {
	// Reuse helper by registering Writable and counting events
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srvConnCh := make(chan net.Conn, 1)
	go func() {
		c, er := ln.Accept()
		if er == nil {
			srvConnCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer cli.Close()

	srv := <-srvConnCh
	defer srv.Close()

	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer p.Stop()

	var count int
	done := make(chan struct{}, 1)
	if err := p.Register(cli, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			count++
			if count == 1 {
				done <- struct{}{}
			}
		}
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	select {
	case <-done:
		// ok got first writable
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for writable event")
	}

	// Deregister and ensure no more events accrue shortly after
	if err := p.Deregister(cli); err != nil {
		t.Fatalf("deregister: %v", err)
	}
	before := count
	time.Sleep(200 * time.Millisecond)
	if count > before {
		t.Fatalf("unexpected writable events after deregister: before=%d after=%d", before, count)
	}
}

// Additional IOCP tests to strengthen behavior under races and EOF edges.

func TestIOCPPoller_RegisterDeregisterRace(t *testing.T) {
	// This test exercises rapid register/deregister cycles to surface races.
	// It does not assert specific events, only that no panics occur and no
	// events are delivered after deregistration.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Accept loop
	srvConnCh := make(chan net.Conn, 32)
	go func() {
		for {
			c, er := ln.Accept()
			if er != nil {
				return
			}
			srvConnCh <- c
		}
	}()

	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer p.Stop()

	for i := 0; i < 50; i++ {
		cli, er := net.Dial("tcp", ln.Addr().String())
		if er != nil {
			t.Fatalf("dial: %v", er)
		}
		srv := <-srvConnCh
		// Register readable; immediately deregister
		gotPost := make(chan struct{}, 1)
		if er := p.Register(cli, []EventType{Readable, Writable}, func(ev Event) {
			// If we get events after deregister, flag
			select {
			case gotPost <- struct{}{}:
			default:
			}
		}); er != nil {
			// Allow already registered in pathological cases, but continue
		}
		// Deregister quickly
		_ = p.Deregister(cli)
		// Close client and server
		_ = cli.Close()
		_ = srv.Close()
		// Ensure no events arrive post deregistration in a short window
		select {
		case <-gotPost:
			t.Fatalf("unexpected event after deregister at iter %d", i)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func TestIOCPPoller_CancelZeroByteRecvOnDeregister(t *testing.T) {
	// When a socket is deregistered, pending zero-byte receives should be canceled
	// and no further events delivered.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srvConnCh := make(chan net.Conn, 1)
	go func() {
		c, er := ln.Accept()
		if er == nil {
			srvConnCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	srv := <-srvConnCh

	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer p.Stop()

	got := make(chan Event, 8)
	if err := p.Register(cli, []EventType{Readable}, func(ev Event) { got <- ev }); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Immediately deregister
	if err := p.Deregister(cli); err != nil {
		t.Fatalf("deregister: %v", err)
	}
	// Close both ends to ensure any pending I/O is canceled quickly
	_ = cli.Close()
	_ = srv.Close()

	// Ensure no events delivered after deregister
	select {
	case ev := <-got:
		t.Fatalf("unexpected event after deregister: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIOCPPoller_ReadEOFBoundary(t *testing.T) {
	// Ensure EOF is reported when peer closes without sending additional data.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srvConnCh := make(chan net.Conn, 1)
	go func() {
		c, er := ln.Accept()
		if er == nil {
			// Close immediately to trigger EOF on client
			_ = c.Close()
			srvConnCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer cli.Close()
	<-srvConnCh // ensure server side accepted and closed

	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer p.Stop()

	got := make(chan Event, 1)
	if err := p.Register(cli, []EventType{Readable}, func(ev Event) { got <- ev }); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Expect an Error with EOF in a short time window
	select {
	case ev := <-got:
		if ev.Type != Error {
			t.Fatalf("expected Error event, got %v", ev.Type)
		}
		if ev.Err == nil {
			t.Fatalf("expected EOF error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for EOF event")
	}
}
