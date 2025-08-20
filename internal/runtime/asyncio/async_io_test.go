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

// Stress: rapid register/deregister across many connections should not deadlock or leak events after deregister.
func testPoller_Stress_RapidRegDereg(t *testing.T, makePoller func() Poller) {
	const n = 64
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// acceptor.
	srvCh := make(chan net.Conn, n)
	go func() {
		for i := 0; i < n; i++ {
			c, er := ln.Accept()
			if er == nil {
				srvCh <- c
			} else {
				return
			}
		}
	}()

	// dials.
	clients := make([]net.Conn, 0, n)
	for i := 0; i < n; i++ {
		c, er := net.Dial("tcp", ln.Addr().String())
		if er != nil {
			t.Fatal(er)
		}
		clients = append(clients, c)
	}
	defer func() {
		for _, c := range clients {
			_ = c.Close()
		}
	}()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	evCh := make(chan Event, n*2)
	srvs := make([]net.Conn, 0, n)
	for i := 0; i < n; i++ {
		srvs = append(srvs, <-srvCh)
	}
	defer func() {
		for _, s := range srvs {
			_ = s.Close()
		}
	}()

	// Rapid register/deregister
	for i := 0; i < n; i++ {
		s := srvs[i]
		if err := p.Register(s, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
			t.Fatal(err)
		}
		_ = p.Deregister(s)
	}

	// Small observation window: expect no events.
	if _, ok := waitEvent(t, evCh, 250*time.Millisecond); ok {
		t.Fatalf("unexpected event observed in rapid reg/dereg stress")
	}
}

func TestDefaultPoller_Stress_RapidRegDereg(t *testing.T) {
	testPoller_Stress_RapidRegDereg(t, NewDefaultPoller)
}

func TestOSPoller_Stress_RapidRegDereg(t *testing.T) {
	testPoller_Stress_RapidRegDereg(t, NewOSPoller)
}

// Benchmark: measure writable events over time to observe throttling effectiveness.
func benchmarkPoller_WritableRate(b *testing.B, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	srvConnCh := make(chan net.Conn, 1)
	go func() {
		if c, er := ln.Accept(); er == nil {
			srvConnCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		b.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		b.Fatal(err)
	}
	defer p.Stop()

	var count int64
	if err := p.Register(cli, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			atomic.AddInt64(&count, 1)
		}
	}); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// single iteration observes events for ~10ms.
		time.Sleep(10 * time.Millisecond)
	}
	b.StopTimer()
	_ = atomic.LoadInt64(&count) // value recorded by benchmark output
}

func BenchmarkDefaultPoller_WritableRate(b *testing.B) {
	benchmarkPoller_WritableRate(b, NewDefaultPoller)
}
func BenchmarkOSPoller_WritableRate(b *testing.B) { benchmarkPoller_WritableRate(b, NewOSPoller) }

func testPollerReadableAndDeregister(t *testing.T, makePoller func() Poller) {
	// Setup TCP loopback pair.
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

	// Trigger readability by writing from client.
	if _, err := cli.Write([]byte("hello")); err != nil {
		t.Fatalf("client write: %v", err)
	}

	// Expect a Readable event.
	if ev, ok := waitEvent(t, evCh, 2*time.Second); !ok || ev.Type != Readable {
		t.Fatalf("expected Readable event, got: %+v (ok=%v)", ev, ok)
	}

	// Deregister and ensure no further events are delivered.
	if err := p.Deregister(srv); err != nil {
		t.Fatalf("deregister: %v", err)
	}

	// Drain any events that might have been queued just before deregistration.
	drainDeadline := time.Now().Add(100 * time.Millisecond)
	for {
		select {
		case <-evCh:
			// keep draining.
		default:
			// small wait loop to catch stragglers within the drain window.
			if time.Now().After(drainDeadline) {
				goto drained
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
drained:

	// Try to trigger more events after deregister.
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

// Writable throttling should limit idle notifications to ~20Hz (>=50ms spacing).
func testPollerWritableThrottled(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	var count int32
	if err := p.Register(cli, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			atomic.AddInt32(&count, 1)
		}
	}); err != nil {
		t.Fatal(err)
	}

	// Observe for 200ms; expect at most ~5 notifications if throttled at 50ms.
	time.Sleep(200 * time.Millisecond)
	c := atomic.LoadInt32(&count)
	if c > 6 { // allow slight jitter
		t.Fatalf("writable throttling too frequent: got=%d in 200ms", c)
	}
}

func TestDefaultPoller_Writable_Throttled(t *testing.T) {
	testPollerWritableThrottled(t, NewDefaultPoller)
}

func TestOSPoller_Writable_Throttled(t *testing.T) {
	testPollerWritableThrottled(t, NewOSPoller)
}

// Register should be idempotent: re-registering the same connection should update the handler/kinds.
func testPollerRegisterIdempotent(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	var firstCount int32
	var secondCount int32
	// First registration.
	if err := p.Register(srv, []EventType{Readable}, func(ev Event) {
		if ev.Type == Readable {
			atomic.AddInt32(&firstCount, 1)
		}
	}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	// Second registration on same conn; should update handler and not error.
	if err := p.Register(srv, []EventType{Readable}, func(ev Event) {
		if ev.Type == Readable {
			atomic.AddInt32(&secondCount, 1)
		}
	}); err != nil {
		t.Fatalf("second register (idempotent) failed: %v", err)
	}

	// Trigger readability.
	if _, err := cli.Write([]byte("X")); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&secondCount) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if c := atomic.LoadInt32(&secondCount); c == 0 {
		t.Fatalf("expected second handler to receive Readable, got=%d", c)
	}
	if c := atomic.LoadInt32(&firstCount); c != 0 {
		t.Fatalf("first handler should not be invoked after re-register, got=%d", c)
	}
}

func TestDefaultPoller_Register_Idempotent(t *testing.T) {
	testPollerRegisterIdempotent(t, NewDefaultPoller)
}

func TestOSPoller_Register_Idempotent(t *testing.T) {
	testPollerRegisterIdempotent(t, NewOSPoller)
}

// Register idempotency should also update event kinds (e.g., Readable -> Writable).
func testPollerRegisterIdempotent_UpdateKinds(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	srvConnCh := make(chan net.Conn, 1)
	go func() {
		if c, er := ln.Accept(); er == nil {
			srvConnCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	gotReadable := atomic.Bool{}
	gotWritable := atomic.Bool{}

	// First: Readable only.
	if err := p.Register(srv, []EventType{Readable}, func(ev Event) {
		if ev.Type == Readable {
			gotReadable.Store(true)
		}
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Immediately re-register: Writable only.
	if err := p.Register(srv, []EventType{Writable}, func(ev Event) {
		if ev.Type == Writable {
			gotWritable.Store(true)
		}
	}); err != nil {
		t.Fatalf("reregister(writable): %v", err)
	}

	// Wait a bit to allow writable probe.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && !gotWritable.Load() {
		time.Sleep(5 * time.Millisecond)
	}
	if !gotWritable.Load() {
		t.Fatalf("expected Writable after updating kinds")
	}
	if gotReadable.Load() {
		t.Fatalf("Readable should not be delivered after kinds update")
	}
}

func TestDefaultPoller_Register_Idempotent_UpdateKinds(t *testing.T) {
	testPollerRegisterIdempotent_UpdateKinds(t, NewDefaultPoller)
}

func TestOSPoller_Register_Idempotent_UpdateKinds(t *testing.T) {
	testPollerRegisterIdempotent_UpdateKinds(t, NewOSPoller)
}

// Deregister robustness: calling Deregister twice should be safe and not deliver further events.
func testPollerDeregisterIdempotent(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	evCh := make(chan Event, 8)
	if err := p.Register(srv, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
		t.Fatalf("register: %v", err)
	}
	// trigger an event then deregister twice.
	_, _ = cli.Write([]byte("ping"))
	_ = p.Deregister(srv)
	_ = p.Deregister(srv)

	// ensure no further events after small grace period.
	if _, ok := waitEvent(t, evCh, 200*time.Millisecond); ok {
		t.Fatalf("unexpected event after double deregister")
	}
}

func TestDefaultPoller_Deregister_Idempotent(t *testing.T) {
	testPollerDeregisterIdempotent(t, NewDefaultPoller)
}

func TestOSPoller_Deregister_Idempotent(t *testing.T) {
	testPollerDeregisterIdempotent(t, NewOSPoller)
}

// Deregister after closing the connection should be safe.
func testPollerDeregisterAfterClose(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	srv := <-srvConnCh

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	if err := p.Register(srv, []EventType{Readable}, func(ev Event) {}); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Close both sides, then Deregister should not hang or panic.
	_ = cli.Close()
	_ = srv.Close()
	_ = p.Deregister(srv)
}

func TestDefaultPoller_Deregister_AfterClose(t *testing.T) {
	testPollerDeregisterAfterClose(t, NewDefaultPoller)
}

func TestOSPoller_Deregister_AfterClose(t *testing.T) {
	testPollerDeregisterAfterClose(t, NewOSPoller)
}

// After Deregister, even with subsequent traffic, no events should be delivered.
func testPoller_NoEvents_AfterDeregister_UnderTraffic(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	defer cli.Close()
	srv := <-srvConnCh
	defer srv.Close()

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	evCh := make(chan Event, 32)
	if err := p.Register(srv, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Deregister immediately.
	if err := p.Deregister(srv); err != nil {
		t.Fatalf("deregister: %v", err)
	}

	// Flood traffic after deregister.
	for i := 0; i < 50; i++ {
		_, _ = cli.Write([]byte("spam"))
		time.Sleep(2 * time.Millisecond)
	}

	// Ensure no events within the observation window.
	if _, ok := waitEvent(t, evCh, 300*time.Millisecond); ok {
		t.Fatalf("unexpected event after deregister under traffic")
	}
}

func TestDefaultPoller_NoEvents_AfterDeregister_UnderTraffic(t *testing.T) {
	testPoller_NoEvents_AfterDeregister_UnderTraffic(t, NewDefaultPoller)
}

func TestOSPoller_NoEvents_AfterDeregister_UnderTraffic(t *testing.T) {
	testPoller_NoEvents_AfterDeregister_UnderTraffic(t, NewOSPoller)
}

// Race test: Deregister concurrently with incoming data should not yield stray events.
func testPoller_NoStrayEvents_OnDeregisterRace(t *testing.T, makePoller func() Poller) {
	const iters = 10
	for n := 0; n < iters; n++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		srvConnCh := make(chan net.Conn, 1)
		go func() {
			if c, er := ln.Accept(); er == nil {
				srvConnCh <- c
			}
		}()

		cli, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		srv := <-srvConnCh

		p := makePoller()
		if err := p.Start(context.Background()); err != nil {
			t.Fatal(err)
		}

		evCh := make(chan Event, 16)
		if err := p.Register(srv, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
			t.Fatalf("register: %v", err)
		}

		// writer goroutine floods a bit.
		doneW := make(chan struct{})
		go func() {
			defer close(doneW)
			for i := 0; i < 20; i++ {
				_, _ = cli.Write([]byte("x"))
				time.Sleep(1 * time.Millisecond)
			}
		}()

		// race: deregister asap.
		_ = p.Deregister(srv)

		// observation window: should not receive any events.
		if _, ok := waitEvent(t, evCh, 200*time.Millisecond); ok {
			t.Fatalf("stray event observed on iteration %d", n)
		}

		// cleanup.
		<-doneW
		_ = cli.Close()
		_ = srv.Close()
		_ = p.Stop()
		_ = ln.Close()
	}
}

func TestDefaultPoller_NoStrayEvents_OnDeregisterRace(t *testing.T) {
	testPoller_NoStrayEvents_OnDeregisterRace(t, NewDefaultPoller)
}

func TestOSPoller_NoStrayEvents_OnDeregisterRace(t *testing.T) {
	testPoller_NoStrayEvents_OnDeregisterRace(t, NewOSPoller)
}

// After Deregister, even if EOF arrives immediately, no events must be delivered.
func testPoller_NoEvents_OnEOF_AfterDeregister(t *testing.T, makePoller func() Poller) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	srvConnCh := make(chan net.Conn, 1)
	go func() {
		if c, er := ln.Accept(); er == nil {
			srvConnCh <- c
		}
	}()

	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	srv := <-srvConnCh

	p := makePoller()
	if err := p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	evCh := make(chan Event, 8)
	if err := p.Register(srv, []EventType{Readable, Writable}, func(ev Event) { evCh <- ev }); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Deregister then close the peer immediately to cause EOF.
	_ = p.Deregister(srv)
	_ = cli.Close()

	// No events should be observed after Deregister.
	if _, ok := waitEvent(t, evCh, 200*time.Millisecond); ok {
		t.Fatalf("unexpected event after deregister with immediate EOF")
	}

	_ = srv.Close()
	_ = p.Stop()
}

func TestDefaultPoller_NoEvents_OnEOF_AfterDeregister(t *testing.T) {
	testPoller_NoEvents_OnEOF_AfterDeregister(t, NewDefaultPoller)
}

func TestOSPoller_NoEvents_OnEOF_AfterDeregister(t *testing.T) {
	testPoller_NoEvents_OnEOF_AfterDeregister(t, NewOSPoller)
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
		// ok.
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for readability")
	}
	<-done
}
