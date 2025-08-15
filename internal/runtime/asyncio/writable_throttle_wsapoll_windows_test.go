//go:build windows
// +build windows

package asyncio

import (
    "context"
    "net"
    "os"
    "sync/atomic"
    "testing"
    "time"
)

// Ensure WSAPoll-based poller honors ORIZON_WIN_WRITABLE_INTERVAL_MS.
func TestWSAPoll_WritableThrottling_EnvInterval(t *testing.T) {
    t.Setenv("ORIZON_WIN_WSAPOLL", "1")
    t.Setenv("ORIZON_WIN_IOCP", "0")
    t.Setenv("ORIZON_WIN_PORTABLE", "0")
    t.Setenv("ORIZON_WIN_WRITABLE_INTERVAL_MS", "10")

    p := NewOSPoller()
    if p == nil {
        t.Fatal("nil poller")
    }
    if _, ok := p.(*windowsPoller); !ok {
        t.Fatalf("expected *windowsPoller, got %T (env=%v)", p, os.Environ())
    }
    if err := p.Start(context.Background()); err != nil { t.Fatal(err) }
    defer p.Stop()

    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer ln.Close()

    srvCh := make(chan net.Conn, 1)
    go func(){ c, e := ln.Accept(); if e==nil { srvCh <- c } }()

    cli, err := net.Dial("tcp", ln.Addr().String())
    if err != nil { t.Fatal(err) }
    defer cli.Close()
    srv := <-srvCh
    defer srv.Close()

    var cnt int32
    if err := p.Register(cli, []EventType{Writable}, func(ev Event){ if ev.Type==Writable { atomic.AddInt32(&cnt,1) } }); err != nil { t.Fatal(err) }

    time.Sleep(200 * time.Millisecond)

    if got := atomic.LoadInt32(&cnt); got < 8 {
        t.Fatalf("too few writable notifications with 10ms interval on WSAPoll: got=%d", got)
    }
}
