package asyncio

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// Test that ORIZON_WIN_WRITABLE_INTERVAL_MS affects Writable notification frequency.
// Note: interval is read once per process; set env BEFORE creating any poller.
func TestWritableThrottling_EnvInterval(t *testing.T) {
	t.Setenv("ORIZON_WIN_WRITABLE_INTERVAL_MS", "10")

	p := NewDefaultPoller()
	if err := p.Start(context.Background()); err != nil { t.Fatal(err) }
	defer p.Stop()

	c1, c2 := net.Pipe()
	defer c1.Close(); defer c2.Close()

	var cnt int32
	if err := p.Register(c1, []EventType{Writable}, func(ev Event){ if ev.Type==Writable { atomic.AddInt32(&cnt,1) } }); err != nil { t.Fatal(err) }

	// Observe for 200ms; with ~10ms throttle we expect at least ~8 events.
	time.Sleep(200 * time.Millisecond)
	if got := atomic.LoadInt32(&cnt); got < 8 {
		t.Fatalf("too few writable notifications with 10ms interval: got=%d", got)
	}
}
