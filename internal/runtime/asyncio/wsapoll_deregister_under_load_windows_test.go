//go:build windows.
// +build windows.

package asyncio

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// Ensure no events are delivered after Deregister, even while traffic is ongoing, for WSAPoll-based poller.
func TestWSAPoll_NoEventsAfterDeregister_UnderTraffic(t *testing.T) {
	t.Setenv("ORIZON_WIN_WSAPOLL", "1")
	t.Setenv("ORIZON_WIN_IOCP", "0")
	t.Setenv("ORIZON_WIN_PORTABLE", "0")

	p := NewOSPoller()
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
	srv := <-srvCh

	var got int32
	if err := p.Register(cli, []EventType{Readable, Writable, Error}, func(ev Event) { atomic.AddInt32(&got, 1) }); err != nil {
		t.Fatal(err)
	}

	// Start traffic generator on server side.
	stop := make(chan struct{})
	go func() {
		payload := make([]byte, 1024)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_, _ = srv.Write(payload)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Let events flow briefly.
	time.Sleep(50 * time.Millisecond)

	// Deregister and ensure no further events are observed.
	if err := p.Deregister(cli); err != nil {
		t.Fatal(err)
	}

	before := atomic.LoadInt32(&got)
	time.Sleep(200 * time.Millisecond)
	after := atomic.LoadInt32(&got)

	close(stop)
	_ = cli.Close()
	_ = srv.Close()

	if after > before {
		t.Fatalf("unexpected events after deregister: before=%d after=%d", before, after)
	}
}
