//go:build windows && iocp
// +build windows,iocp

package asyncio

import (
	"context"
	"net"
	"testing"
	"time"
)

// BenchmarkIOCPPoller_SingleConn measures the overhead of IOCP poller delivering
// a small number of Readable events on a single TCP connection.
func BenchmarkIOCPPoller_SingleConn(b *testing.B) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverReady := make(chan net.Conn, 1)
	go func() {
		c, er := ln.Accept()
		if er == nil {
			serverReady <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		b.Fatalf("dial: %v", err)
	}
	defer cli.Close()

	srv := <-serverReady
	defer srv.Close()

	p := NewIOCPPoller()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Start(ctx); err != nil {
		b.Fatalf("start: %v", err)
	}
	defer p.Stop()

	ready := make(chan struct{}, 1)
	if err := p.Register(srv, []EventType{Readable}, func(ev Event) {
		if ev.Type == Readable {
			select {
			case ready <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		b.Fatalf("register: %v", err)
	}

	// Warmup
	_, _ = cli.Write([]byte{1})
	select {
	case <-ready:
	case <-time.After(time.Second):
		b.Fatalf("warmup timeout")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clear any pending
		select {
		case <-ready:
		default:
		}
		b.StartTimer()
		_, _ = cli.Write([]byte{1})
		select {
		case <-ready:
			// ok
		case <-time.After(2 * time.Second):
			b.Fatalf("timeout at iter=%d", i)
		}
	}
}

// BenchmarkIOCPPoller_MultiConn exercises multiple concurrent connections delivering
// small Readable events to evaluate scalability.
func BenchmarkIOCPPoller_MultiConn(b *testing.B) {
	const numConns = 32
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srvCh := make(chan net.Conn, numConns)
	go func() {
		for i := 0; i < numConns; i++ {
			c, er := ln.Accept()
			if er == nil {
				srvCh <- c
			}
		}
	}()

	clients := make([]net.Conn, 0, numConns)
	for i := 0; i < numConns; i++ {
		c, er := net.Dial("tcp", ln.Addr().String())
		if er != nil {
			b.Fatalf("dial: %v", er)
		}
		clients = append(clients, c)
	}
	defer func() {
		for _, c := range clients {
			_ = c.Close()
		}
	}()

	servers := make([]net.Conn, 0, numConns)
	for i := 0; i < numConns; i++ {
		servers = append(servers, <-srvCh)
	}
	defer func() {
		for _, s := range servers {
			_ = s.Close()
		}
	}()

	p := NewIOCPPoller()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Start(ctx); err != nil {
		b.Fatalf("start: %v", err)
	}
	defer p.Stop()

	ready := make([]chan struct{}, numConns)
	for i, s := range servers {
		ready[i] = make(chan struct{}, 1)
		if err := p.Register(s, []EventType{Readable}, func(ev Event) {
			if ev.Type == Readable {
				select {
				case ready[i] <- struct{}{}:
				default:
				}
			}
		}); err != nil {
			b.Fatalf("register: %v", err)
		}
	}

	// Warmup
	for i, c := range clients {
		_, _ = c.Write([]byte{1})
		select {
		case <-ready[i]:
		case <-time.After(2 * time.Second):
			b.Fatalf("warmup timeout %d", i)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % numConns
		b.StopTimer()
		// drain
		select {
		case <-ready[idx]:
		default:
		}
		b.StartTimer()
		_, _ = clients[idx].Write([]byte{1})
		select {
		case <-ready[idx]:
		case <-time.After(2 * time.Second):
			b.Fatalf("timeout idx=%d", idx)
		}
	}
}

// BenchmarkIOCPPoller_Writable measures periodic writable notifications overhead.
func BenchmarkIOCPPoller_Writable(b *testing.B) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
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
		b.Fatalf("dial: %v", err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := NewIOCPPoller()
	if err := p.Start(context.Background()); err != nil {
		b.Fatalf("start: %v", err)
	}
	defer p.Stop()

	tick := make(chan struct{}, 1)
	if err := p.Register(cli, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			select {
			case tick <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		b.Fatalf("register: %v", err)
	}

	// Wait one tick
	select {
	case <-tick:
	case <-time.After(2 * time.Second):
		b.Fatalf("warmup timeout")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case <-tick:
		case <-time.After(2 * time.Second):
			b.Fatalf("timeout at %d", i)
		}
	}
}
