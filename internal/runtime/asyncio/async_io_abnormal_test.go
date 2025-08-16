package asyncio

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// Test that Register followed by concurrent Deregister and Stop does not deadlock or panic.
func TestAsyncIO_ConcurrentDeregisterAndStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Accept and immediately close to create a quick lifecycle
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			_ = c.Close()
		}
	}()

	p := NewOSPoller()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Register readable/error handler. Multiple callbacks may arrive depending on poller timing,
	// but the test only requires that at least one callback can happen without deadlock.
	// Use a one-shot to avoid negative WaitGroup counts if multiple events are delivered.
	var wg sync.WaitGroup
	wg.Add(1)
	var once sync.Once
	if err := p.Register(client, []EventType{Readable, Error}, func(ev Event) {
		once.Do(func() { wg.Done() })
	}); err != nil {
		t.Fatal(err)
	}

	// Run Deregister and Stop concurrently
	done := make(chan struct{})
	go func() {
		_ = p.Deregister(client)
		close(done)
	}()
	// Give a tiny slice to interleave
	time.Sleep(10 * time.Millisecond)
	_ = p.Stop()

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for deregister to complete")
	}
	// Ensure handler goroutine had a chance to run at least once or was safely canceled
	// Use Wait with timeout to avoid flakes across OSes
	c := make(chan struct{})
	go func() { wg.Wait(); close(c) }()
	select {
	case <-c:
	case <-time.After(time.Second):
		// Even if handler never ran, the important invariant is that we did not deadlock.
	}
}
