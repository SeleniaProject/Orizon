//go:build windows && iocp
// +build windows,iocp

package asyncio

import (
    "context"
    "net"
    "syscall"
    "testing"
    "time"

    "golang.org/x/sys/windows"
)

// Explicitly cancel the pending zero-byte WSARecv using CancelIoEx and
// verify that no spurious events are delivered by the poller.
func TestIOCP_CancelIoEx_NoEventDelivered(t *testing.T) {
    p := NewIOCPPoller()
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

    got := make(chan struct{}, 1)
    if err := p.Register(cli, []EventType{Readable}, func(ev Event){ select{case got<-struct{}{}: default:{}} }); err != nil { t.Fatal(err) }

    // Extract socket handle
    type sc interface{ SyscallConn() (syscall.RawConn, error) }
    rc, err := cli.(sc).SyscallConn()
    if err != nil { t.Fatal(err) }
    var s uintptr
    if err := rc.Control(func(fd uintptr){ s = fd }); err != nil { t.Fatal(err) }
    sh := windows.Handle(s)

    // Give a tiny window to ensure zero-byte recv was posted
    time.Sleep(10 * time.Millisecond)

    // Cancel any pending I/O on the socket
    if err := windows.CancelIoEx(sh, nil); err != nil {
        // ERROR_NOT_FOUND means nothing pending yet; allow a retry once
        if errno, ok := err.(syscall.Errno); ok && errno == syscall.ERROR_NOT_FOUND {
            time.Sleep(10 * time.Millisecond)
            _ = windows.CancelIoEx(sh, nil)
        } else {
            t.Fatalf("CancelIoEx failed: %v", err)
        }
    }

    // Ensure no events delivered in a short window
    select{
    case <-got:
        t.Fatal("unexpected event delivered after CancelIoEx")
    case <-time.After(150 * time.Millisecond):
    }

    _ = p.Deregister(cli)
}
