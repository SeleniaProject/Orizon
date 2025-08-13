package asyncio

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// waitEvent waits for an event or times out.
func waitEvent(t *testing.T, ch <-chan Event, d time.Duration) (Event, bool) {
	t.Helper()
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return Event{}, false
	}
}

func testPollerReadableAndDeregister(t *testing.T, makePoller func() Poller) {
	// Setup TCP loopback pair
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

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("poller start: %v", err)
	}
	defer p.Stop()

	evCh := make(chan Event, 8)
	if err := p.Register(srv, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Trigger readability by writing from client
	if _, err := cli.Write([]byte("hello")); err != nil {
		t.Fatalf("client write: %v", err)
	}

	// Expect a Readable event
	if ev, ok := waitEvent(t, evCh, 2*time.Second); !ok || ev.Type != Readable {
		t.Fatalf("expected Readable event, got: %+v (ok=%v)", ev, ok)
	}

	// Deregister and ensure no further events are delivered
	if err := p.Deregister(srv); err != nil {
		t.Fatalf("deregister: %v", err)
	}

	// Drain any events that might have been queued just before deregistration
	drainDeadline := time.Now().Add(100 * time.Millisecond)
	for {
		select {
		case <-evCh:
			// keep draining
		default:
			// small wait loop to catch stragglers within the drain window
			if time.Now().After(drainDeadline) {
				goto drained
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
drained:

	// Try to trigger more events after deregister
	_, _ = cli.Write([]byte("more"))

	if _, ok := waitEvent(t, evCh, 250*time.Millisecond); ok {
		t.Fatalf("unexpected event after deregister")
	}
}

func testPollerErrorOnClose(t *testing.T, makePoller func() Poller) {
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

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("poller start: %v", err)
	}
	defer p.Stop()

	var gotError atomic.Bool
	if err := p.Register(srv, []EventType{Readable}, func(ev Event) {
		if ev.Type == Error {
			gotError.Store(true)
		}
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Close the client side; server should observe an error/hangup eventually
	_ = cli.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !gotError.Load() {
		time.Sleep(10 * time.Millisecond)
	}
	if !gotError.Load() {
		t.Fatalf("expected Error event on close, but none observed")
	}
}

func TestDefaultPoller_Readable_And_Deregister(t *testing.T) {
	testPollerReadableAndDeregister(t, NewDefaultPoller)
}

func TestDefaultPoller_ErrorOnClose(t *testing.T) {
	testPollerErrorOnClose(t, NewDefaultPoller)
}

func TestOSPoller_Readable_And_Deregister(t *testing.T) {
	testPollerReadableAndDeregister(t, NewOSPoller)
}

func TestOSPoller_ErrorOnClose(t *testing.T) {
	testPollerErrorOnClose(t, NewOSPoller)
}

func TestAsyncIO_Echo_Ready(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan struct{}, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4)
		_, _ = c.Read(buf)
		_, _ = c.Write(buf)
		done <- struct{}{}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	p := NewOSPoller()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	gotReadable := make(chan struct{}, 1)
	if err := p.Register(client, []EventType{Readable, Writable}, func(ev Event) {
		if ev.Type == Writable {
			_, _ = client.Write([]byte("ping"))
		}
		if ev.Type == Readable {
			gotReadable <- struct{}{}
		}
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-gotReadable:
		// ok
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for readability")
	}
	<-done
}
