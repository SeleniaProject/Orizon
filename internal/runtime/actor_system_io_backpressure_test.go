package runtime

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	asyncio "github.com/orizon-lang/orizon/internal/runtime/asyncio"
)

// stubPoller captures registrations and allows manual event delivery.
type stubPoller struct {
	h          atomic.Value // stores asyncio.Handler
	deregCount int32
}

func (s *stubPoller) Start(ctx context.Context) error { return nil }
func (s *stubPoller) Stop() error                     { return nil }
func (s *stubPoller) Register(conn net.Conn, kinds []asyncio.EventType, h asyncio.Handler) error {
	s.h.Store(h)

	return nil
}

func (s *stubPoller) Deregister(conn net.Conn) error {
	atomic.AddInt32(&s.deregCount, 1)

	return nil
}

func (s *stubPoller) fire(ev asyncio.Event) {
	if v := s.h.Load(); v != nil {
		v.(asyncio.Handler)(ev)
	}
}

// fakeConn is a minimal net.Conn for tests.
type fakeConn struct{}

func (f *fakeConn) Read(b []byte) (int, error)         { return 0, nil }
func (f *fakeConn) Write(b []byte) (int, error)        { return len(b), nil }
func (f *fakeConn) Close() error                       { return nil }
func (f *fakeConn) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (f *fakeConn) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (f *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

// slowBehavior blocks on a channel to keep the mailbox from draining quickly.
type slowBehavior struct{ blockCh chan struct{} }

func (b *slowBehavior) Receive(ctx *ActorContext, msg Message) error {
	<-b.blockCh

	return nil
}
func (b *slowBehavior) PreStart(ctx *ActorContext) error { return nil }
func (b *slowBehavior) PostStop(ctx *ActorContext) error { return nil }
func (b *slowBehavior) PreRestart(ctx *ActorContext, reason error, message *Message) error {
	return nil
}
func (b *slowBehavior) PostRestart(ctx *ActorContext, reason error) error { return nil }
func (b *slowBehavior) GetBehaviorName() string                           { return "slow" }

func TestIOWatch_WatermarkPause(t *testing.T) {
	sys, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("system: %v", err)
	}

	if err := sys.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	defer sys.Stop()

	// Stop scheduler to keep the prefilled mailbox from draining before the watermark check.
	sys.scheduler.Stop()

	blk := make(chan struct{})

	actor, err := NewActor("ioTarget", UserActor, &slowBehavior{blockCh: blk}, DefaultActorConfig)
	if err != nil {
		t.Fatalf("actor: %v", err)
	}

	sys.mutex.Lock()
	sys.actors[actor.ID] = actor
	sys.mailboxes[actor.Mailbox.ID] = actor.Mailbox
	sys.mutex.Unlock()
	actor.Context.System = sys

	// Pre-fill mailbox with one message so that length >= 1.
	if err := sys.SendMessage(0, actor.ID, IOReadable, IOEvent{}); err != nil {
		t.Fatalf("prefill: %v", err)
	}

	sp := &stubPoller{}
	sys.SetIOPoller(sp)

	opts := IOWatchOptions{
		HighWatermark:     1,
		LowWatermark:      0,
		MonitorInterval:   time.Millisecond * 5,
		ReadEventPriority: NormalPriority,
	}
	conn := &fakeConn{}

	if err := sys.WatchConnWithActorOpts(conn, []asyncio.EventType{asyncio.Readable}, actor.ID, opts); err != nil {
		t.Fatalf("watch: %v", err)
	}

	// Fire a readable event; with mailbox len >= HighWatermark, it should pause (Deregister once).
	sp.fire(asyncio.Event{Conn: conn, Type: asyncio.Readable})

	// Allow a brief moment for handler to run.
	time.Sleep(20 * time.Millisecond)

	if atomic.LoadInt32(&sp.deregCount) == 0 {
		t.Fatalf("expected Deregister to be called at least once")
	}

	// Unblock actor to drain and avoid leaks.
	close(blk)
}
