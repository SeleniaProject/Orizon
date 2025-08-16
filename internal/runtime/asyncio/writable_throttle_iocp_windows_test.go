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

// Ensure IOCP-based poller honors ORIZON_WIN_WRITABLE_INTERVAL_MS.
func TestIOCP_WritableThrottling_EnvInterval(t *testing.T) {
	t.Setenv("ORIZON_WIN_WRITABLE_INTERVAL_MS", "10")

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

	var cnt int32
	if err := p.Register(cli, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			atomic.AddInt32(&cnt, 1)
		}
	}); err != nil {
		t.Fatal(err)
	}

	// Observe for 200ms; with ~10ms throttle we expect multiple events.
	// IOCP timer jitter can be significant on shared CI.
	time.Sleep(200 * time.Millisecond)

	if got := atomic.LoadInt32(&cnt); got < 4 {
		t.Fatalf("too few writable notifications with 10ms interval on IOCP: got=%d", got)
	}
}
