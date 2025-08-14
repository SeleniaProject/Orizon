//go:build windows && iocp
// +build windows,iocp

package asyncio

import (
    "context"
    "net"
    "sync/atomic"
    "testing"
    "time"
)

// waitEventIOCP waits for an event or times out.
func waitEventIOCP(t *testing.T, ch <-chan Event, d time.Duration) (Event, bool) {
    t.Helper()
    select {
    case ev := <-ch:
        return ev, true
    case <-time.After(d):
        return Event{}, false
    }
}

func testIOCPPollerReadableAndDeregister(t *testing.T) {
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
        t.Fatalf("poller start: %v", err)
    }
    defer p.Stop()

    evCh := make(chan Event, 8)
    if err := p.Register(srv, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
        t.Fatalf("register: %v", err)
    }

    if _, err := cli.Write([]byte("hello")); err != nil {
        t.Fatalf("client write: %v", err)
    }

    if ev, ok := waitEventIOCP(t, evCh, 2*time.Second); !ok || ev.Type != Readable {
        t.Fatalf("expected Readable event, got: %+v (ok=%v)", ev, ok)
    }

    if err := p.Deregister(srv); err != nil {
        t.Fatalf("deregister: %v", err)
    }
    if _, ok := waitEventIOCP(t, evCh, 250*time.Millisecond); ok {
        t.Fatalf("unexpected event after deregister")
    }
}

func testIOCPPollerErrorOnClose(t *testing.T) {
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

    _ = cli.Close()

    deadline := time.Now().Add(2 * time.Second)
    for time.Now().Before(deadline) && !gotError.Load() {
        time.Sleep(10 * time.Millisecond)
    }
    if !gotError.Load() {
        t.Fatalf("expected Error event on close, but none observed")
    }
}

func TestIOCPPoller_Readable_And_Deregister(t *testing.T) {
    testIOCPPollerReadableAndDeregister(t)
}

func TestIOCPPoller_ErrorOnClose(t *testing.T) {
    testIOCPPollerErrorOnClose(t)
}

func TestIOCPPoller_Echo_Ready(t *testing.T) {
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

    p := NewIOCPPoller()
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
	testPollerReadableAndDeregister(t, NewIOCPPoller)
}

func TestIOCPPoller_ErrorOnClose(t *testing.T) {
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
