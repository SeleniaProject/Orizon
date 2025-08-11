package asyncio

import (
    "context"
    "net"
    "testing"
    "time"
)

func TestAsyncIO_Echo_Ready(t *testing.T) {
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer ln.Close()

    done := make(chan struct{}, 1)
    go func(){
        c, err := ln.Accept()
        if err != nil { return }
        defer c.Close()
        buf := make([]byte, 4)
        _, _ = c.Read(buf)
        _, _ = c.Write(buf)
        done <- struct{}{}
    }()

    client, err := net.Dial("tcp", ln.Addr().String())
    if err != nil { t.Fatal(err) }
    defer client.Close()

    p := NewDefaultPoller()
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    if err := p.Start(ctx); err != nil { t.Fatal(err) }
    defer p.Stop()

    gotReadable := make(chan struct{}, 1)
    if err := p.Register(client, []EventType{Readable, Writable}, func(ev Event){
        if ev.Type == Writable {
            _, _ = client.Write([]byte("ping"))
        }
        if ev.Type == Readable {
            gotReadable <- struct{}{}
        }
    }); err != nil { t.Fatal(err) }

    select {
    case <-gotReadable:
        // ok
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for readability")
    }
    <-done
}


