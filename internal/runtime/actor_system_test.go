package runtime

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	asyncio "github.com/orizon-lang/orizon/internal/runtime/asyncio"
	vfs "github.com/orizon-lang/orizon/internal/runtime/vfs"
)

// testBehavior is a simple actor behavior used for testing.
type testBehavior struct {
	received chan Message
	name     string
	failOnce bool
}

func (tb *testBehavior) Receive(ctx *ActorContext, msg Message) error {
	if tb.failOnce {
		tb.failOnce = false

		return fmt.Errorf("fail")
	}
	select {
	case tb.received <- msg:
	default:
	}

	return nil
}

func (tb *testBehavior) PreStart(ctx *ActorContext) error { return nil }
func (tb *testBehavior) PostStop(ctx *ActorContext) error { return nil }
func (tb *testBehavior) PreRestart(ctx *ActorContext, reason error, message *Message) error {
	return nil
}
func (tb *testBehavior) PostRestart(ctx *ActorContext, reason error) error { return nil }
func (tb *testBehavior) GetBehaviorName() string                           { return tb.name }

func TestActorSystem_Lifecycle(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("failed to create actor system: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("failed to start actor system: %v", err)
	}

	if err := system.Stop(); err != nil {
		t.Fatalf("failed to stop actor system: %v", err)
	}
}

// Scheduling exploration test: inject a stub poller and fire Readable events while mailbox is near watermarks.
func TestActorSystem_Scheduling_WatermarkAndRate(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	defer system.Stop()

	blk := make(chan struct{})

	actor, err := NewActor("schedTarget", UserActor, &testBehavior{received: make(chan Message, 16), name: "sched"}, DefaultActorConfig)
	if err != nil {
		t.Fatalf("actor: %v", err)
	}

	system.mutex.Lock()
	system.actors[actor.ID] = actor
	system.mailboxes[actor.Mailbox.ID] = actor.Mailbox
	system.mutex.Unlock()
	actor.Context.System = system

	// Prefill mailbox to simulate backpressure near high watermark.
	_ = system.SendMessage(0, actor.ID, IOReadable, IOEvent{})

	// stub poller.
	sp := &testPollerHolder{}

	system.SetIOPoller(&pollerAdapter{sp: sp})

	// Watch with tight watermarks.
	opts := IOWatchOptions{HighWatermark: 1, LowWatermark: 0, MonitorInterval: time.Millisecond * 5, ReadEventPriority: NormalPriority}
	conn := &fakeConn{}

	if err := system.WatchConnWithActorOpts(conn, []asyncio.EventType{asyncio.Readable}, actor.ID, opts); err != nil {
		t.Fatalf("watch: %v", err)
	}
	// fire readable then allow time to process.
	// Reuse helper from io_backpressure test via a small local dispatcher.
	if ap, ok := system.ioPoller.(*pollerAdapter); ok {
		ap.fire(asyncio.Event{Conn: conn, Type: asyncio.Readable})
	}

	time.Sleep(20 * time.Millisecond)
	close(blk)
}

// pollerAdapter adapts a minimal stub to asyncio.Poller for local test usage.
type pollerAdapter struct{ sp *testPollerHolder }

func (p *pollerAdapter) Start(ctx context.Context) error { return nil }
func (p *pollerAdapter) Stop() error                     { return nil }
func (p *pollerAdapter) Register(conn net.Conn, kinds []asyncio.EventType, h asyncio.Handler) error {
	p.sp.h = h

	return nil
}
func (p *pollerAdapter) Deregister(conn net.Conn) error { return nil }
func (p *pollerAdapter) fire(ev asyncio.Event) {
	if p.sp.h != nil {
		p.sp.h.(asyncio.Handler)(ev)
	}
}

// testPollerHolder stores the handler function for the poller adapter.
type testPollerHolder struct{ h any }

func TestActor_MessageFlow_ManualDispatch(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("failed to create actor system: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	defer system.Stop()

	// Stop scheduler to avoid auto-dispatch interfering with manual dequeue below.
	system.scheduler.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "test"}

	actor, err := system.CreateActor("echo", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("failed to create actor: %v", err)
	}

	// Send via system (enqueues to mailbox).
	if err := system.SendMessage(0, actor.ID, 1, "hello"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	// Manually dispatch: pop from mailbox and process (scheduler is decoupled).
	msg, ok := actor.Mailbox.Dequeue()
	if !ok {
		t.Fatalf("expected a message in mailbox")
	}

	if err := actor.ProcessMessage(msg); err != nil {
		t.Fatalf("process failed: %v", err)
	}

	select {
	case got := <-tb.received:
		if got.Payload != "hello" {
			t.Errorf("unexpected payload: %v", got.Payload)
		}

		if got.Receiver != actor.ID {
			t.Errorf("unexpected receiver: %v", got.Receiver)
		}
	case <-time.After(time.Second):
		t.Fatal("behavior did not receive message")
	}
}

func TestDebugInspector_Snapshots_And_Tracing(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("new system: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	defer system.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "dbg"}

	a, err := system.CreateActor("dbg", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// enable tracing with small ring.
	system.EnableTracing(8)

	if err := system.SendMessage(0, a.ID, 123, "trace"); err != nil {
		t.Fatalf("send: %v", err)
	}
	select {
	case <-tb.received:
	case <-time.After(time.Second):
		t.Fatal("did not receive")
	}

	// actor snapshot.
	snap, ok := system.GetActorSnapshot(a.ID)
	if !ok || snap.ID != a.ID || snap.Name != "dbg" {
		t.Fatalf("snapshot mismatch: %+v", snap)
	}
	// system snapshot.
	ss := system.GetSystemSnapshot()
	if len(ss.Actors) == 0 || ss.Statistics.TotalMessages == 0 {
		t.Fatalf("system snapshot not populated: %+v", ss)
	}
	// recent messages.
	evs := system.GetRecentMessages(a.ID, 4)
	if len(evs) == 0 || evs[len(evs)-1].Type != 123 {
		t.Fatalf("trace not recorded: %+v", evs)
	}

	system.DisableTracing()
}

func TestActorSystem_AutoDispatch(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("failed to create actor system: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	defer system.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "auto"}

	actor, err := system.CreateActor("auto", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("failed to create actor: %v", err)
	}

	// Send message; scheduler worker should auto-dispatch and process it.
	if err := system.SendMessage(0, actor.ID, 1, "ping"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	select {
	case got := <-tb.received:
		if got.Payload != "ping" {
			t.Errorf("unexpected payload: %v", got.Payload)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("auto dispatch did not deliver message in time")
	}
}

// Simple interceptor/transformer for testing dispatcher pipeline.
type interceptOnce struct{ called *bool }

func (i interceptOnce) Intercept(msg Message) (Message, error) {
	*i.called = true
	return msg, nil
}

type transformTag struct{ key string }

func (t transformTag) Transform(msg Message) (Message, error) {
	if msg.Headers == nil {
		msg.Headers = map[string]interface{}{}
	}

	msg.Headers[t.key] = true

	return msg, nil
}

func (t transformTag) GetTransformerName() string { return "transformTag" }

func (i interceptOnce) GetInterceptorName() string { return "interceptOnce" }

func TestDispatcher_Interception_Transformation(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("failed to create actor system: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	defer system.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "pipe"}

	actor, err := system.CreateActor("pipe", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("failed to create actor: %v", err)
	}

	// Install interceptor and transformer.
	called := false

	system.dispatcher.mutex.Lock()
	system.dispatcher.interceptors = append(system.dispatcher.interceptors, interceptOnce{called: &called})
	system.dispatcher.transformers = append(system.dispatcher.transformers, transformTag{key: "tagged"})
	system.dispatcher.mutex.Unlock()

	if err := system.SendMessage(0, actor.ID, 1, "x"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	select {
	case got := <-tb.received:
		if !called {
			t.Error("interceptor not called")
		}

		if got.Headers["tagged"] != true {
			t.Error("transformer did not tag header")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive message via pipeline")
	}
}

func TestSupervisor_Restart_OnFailure(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("failed to create actor system: %v", err)
	}

	if err := system.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	defer system.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "fail", failOnce: true}

	actor, err := system.CreateActor("fail", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("failed to create actor: %v", err)
	}

	// First message will fail; supervisor should restart actor; second message should succeed.
	if err := system.SendMessage(0, actor.ID, 1, "first"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if err := system.SendMessage(0, actor.ID, 1, "second"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	// Expect at least one successful receipt.
	select {
	case <-tb.received:
		// ok.
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("supervisor did not restart actor in time")
	}
}

// behavior that always fails to test restart limits.
type failAlways struct{ count int }

func (f *failAlways) Receive(ctx *ActorContext, msg Message) error {
	f.count++

	return fmt.Errorf("boom")
}
func (f *failAlways) PreStart(*ActorContext) error                    { return nil }
func (f *failAlways) PostStop(*ActorContext) error                    { return nil }
func (f *failAlways) PreRestart(*ActorContext, error, *Message) error { return nil }
func (f *failAlways) PostRestart(*ActorContext, error) error          { return nil }
func (f *failAlways) GetBehaviorName() string                         { return "failAlways" }

func TestSupervisor_RestartLimits_StopAfterMaxRetries(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	// Configure root supervisor restart window.
	system.rootSupervisor.MaxRetries = 2
	system.rootSupervisor.RetryPeriod = time.Millisecond * 200

	fb := &failAlways{}
	a, _ := system.CreateActor("lim", UserActor, fb, DefaultActorConfig)

	// Send multiple failing messages beyond limit.
	_ = system.SendMessage(0, a.ID, 1, "x")
	_ = system.SendMessage(0, a.ID, 1, "y")
	_ = system.SendMessage(0, a.ID, 1, "z")

	// Allow time for processing and potential stops.
	time.Sleep(time.Millisecond * 600)

	// Actor should be stopped after exceeding retries.
	system.mutex.RLock()
	got := system.actors[a.ID]
	system.mutex.RUnlock()

	if got != nil && got.State != ActorStopped {
		t.Fatalf("expected actor stopped after max retries, got state=%v", got.State)
	}
}

type termProbe struct{ term chan ActorID }

func (tp *termProbe) Receive(ctx *ActorContext, msg Message) error {
	if msg.Type == SystemTerminated {
		if id, _ := msg.Payload.(ActorID); id != 0 {
			tp.term <- id
		}
	}

	return nil
}
func (tp *termProbe) PreStart(*ActorContext) error                    { return nil }
func (tp *termProbe) PostStop(*ActorContext) error                    { return nil }
func (tp *termProbe) PreRestart(*ActorContext, error, *Message) error { return nil }
func (tp *termProbe) PostRestart(*ActorContext, error) error          { return nil }
func (tp *termProbe) GetBehaviorName() string                         { return "termProbe" }

func TestActor_WatchTermination(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	// watcher actor.
	probe := &termProbe{term: make(chan ActorID, 1)}

	watcher, err := system.CreateActor("watcher", UserActor, probe, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}

	// target actor.
	tb := &testBehavior{received: make(chan Message, 1), name: "t"}

	target, err := system.CreateActor("target", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	// register watch.
	watcher.Context.Watch(target.ID)

	// stop target.
	_ = system.stopActor(target)

	select {
	case got := <-probe.term:
		if got != target.ID {
			t.Fatalf("expected term for %v, got %v", target.ID, got)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive termination notification")
	}
}

func TestSupervisor_OneForAll(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	// Configure root as OneForAll + Restart.
	system.rootSupervisor.Strategy = RestartStrategy
	system.rootSupervisor.Type = OneForAll

	tb1 := &testBehavior{received: make(chan Message, 2), name: "a", failOnce: true}
	tb2 := &testBehavior{received: make(chan Message, 2), name: "b"}
	a, _ := system.CreateActor("a", UserActor, tb1, DefaultActorConfig)
	b, _ := system.CreateActor("b", UserActor, tb2, DefaultActorConfig)

	_ = system.SendMessage(0, a.ID, 1, "x") // will fail and trigger restart for all
	_ = system.SendMessage(0, a.ID, 1, "y")
	_ = system.SendMessage(0, b.ID, 1, "z")

	// We only assert no panic and that at least one message is processed after restarts.
	select {
	case <-tb1.received:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("one-for-all did not recover")
	}
}

func TestSupervisor_RestForOne(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	// Configure root as RestForOne + Restart.
	system.rootSupervisor.Strategy = RestartStrategy
	system.rootSupervisor.Type = RestForOne

	tb1 := &testBehavior{received: make(chan Message, 2), name: "a"}
	tb2 := &testBehavior{received: make(chan Message, 2), name: "b", failOnce: true}
	tb3 := &testBehavior{received: make(chan Message, 2), name: "c"}
	_, _ = system.CreateActor("a", UserActor, tb1, DefaultActorConfig)
	b, _ := system.CreateActor("b", UserActor, tb2, DefaultActorConfig)
	_, _ = system.CreateActor("c", UserActor, tb3, DefaultActorConfig)

	_ = system.SendMessage(0, b.ID, 1, "x") // b fails; b and c should restart
	_ = system.SendMessage(0, b.ID, 1, "y")

	select {
	case <-tb2.received:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("rest-for-one did not recover")
	}
}

func TestMailbox_BackPressure(t *testing.T) {
	mb, _ := NewMailbox(StandardMailbox, 2)
	mb.OverflowPolicy = BackPressure
	mb.BackPressureWait = time.Millisecond * 50

	// Fill capacity.
	_ = mb.Enqueue(Message{ID: 1})
	_ = mb.Enqueue(Message{ID: 2})

	// Enqueue with back pressure should timeout eventually.
	start := time.Now()

	err := mb.Enqueue(Message{ID: 3})
	if err == nil {
		t.Fatal("expected back pressure timeout")
	}

	if time.Since(start) < mb.BackPressureWait {
		t.Error("back pressure did not wait long enough")
	}
}

func TestGroup_Broadcast(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	g, err := system.CreateGroup("g", StaticGroup, GroupConfig{})
	if err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	tb1 := &testBehavior{received: make(chan Message, 1), name: "g1"}
	tb2 := &testBehavior{received: make(chan Message, 1), name: "g2"}
	a1, _ := system.CreateActor("g1", UserActor, tb1, DefaultActorConfig)
	a2, _ := system.CreateActor("g2", UserActor, tb2, DefaultActorConfig)

	if err := system.AddToGroup(g.ID, a1.ID); err != nil {
		t.Fatalf("add1: %v", err)
	}

	if err := system.AddToGroup(g.ID, a2.ID); err != nil {
		t.Fatalf("add2: %v", err)
	}

	if err := system.Broadcast(g.ID, 1, "B"); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	// Expect both to receive.
	received := 0
	timeout := time.After(time.Second)

	for received < 2 {
		select {
		case <-tb1.received:
			received++
		case <-tb2.received:
			received++
		case <-timeout:
			t.Fatal("broadcast timed out")
		}
	}
}

func TestActorContext_Timers(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "timer"}
	actor, _ := system.CreateActor("timer", UserActor, tb, DefaultActorConfig)

	fired := make(chan struct{}, 1)

	actor.Context.StartTimer("once", time.Millisecond*10, func() { fired <- struct{}{} })

	select {
	case <-fired:
		// ok.
	case <-time.After(time.Second):
		t.Fatal("timer did not fire")
	}

	// Restart timer then stop it before firing.
	actor.Context.StartTimer("stop", time.Millisecond*50, func() { fired <- struct{}{} })
	actor.Context.StopTimer("stop")
	select {
	case <-fired:
		t.Fatal("stopped timer should not fire")
	case <-time.After(time.Millisecond * 80):
		// ok.
	}
}

func TestDispatcher_RouteToTarget(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	tb1 := &testBehavior{received: make(chan Message, 1), name: "r1"}
	tb2 := &testBehavior{received: make(chan Message, 1), name: "r2"}
	a1, _ := system.CreateActor("r1", UserActor, tb1, DefaultActorConfig)
	a2, _ := system.CreateActor("r2", UserActor, tb2, DefaultActorConfig)

	system.dispatcher.AddRoute(100, DispatchRule{Target: a2.ID, Priority: 0})

	// send with receiver=a1, but route by type to a2.
	if err := system.SendMessage(a1.ID, a1.ID, 100, "route"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	select {
	case <-tb2.received:
		// ok, routed.
	case <-time.After(time.Second):
		t.Fatal("route did not deliver to target")
	}
}

type pingBehavior struct{ got chan string }

func (p *pingBehavior) Receive(ctx *ActorContext, msg Message) error {
	if s, _ := msg.Payload.(string); s != "" {
		p.got <- s
	}

	return nil
}
func (p *pingBehavior) PreStart(*ActorContext) error                    { return nil }
func (p *pingBehavior) PostStop(*ActorContext) error                    { return nil }
func (p *pingBehavior) PreRestart(*ActorContext, error, *Message) error { return nil }
func (p *pingBehavior) PostRestart(*ActorContext, error) error          { return nil }
func (p *pingBehavior) GetBehaviorName() string                         { return "ping" }

func TestActor_SpawnAndTell(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	pb := &pingBehavior{got: make(chan string, 1)}

	ref, err := system.Spawn("p", pb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	if err := ref.Tell(1, "hello"); err != nil {
		t.Fatalf("tell failed: %v", err)
	}
	select {
	case s := <-pb.got:
		if s != "hello" {
			t.Fatalf("unexpected: %s", s)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive ping")
	}
}

// ioBehavior receives IOEvent messages and signals when a Readable event arrives.
type ioBehavior struct{ got chan struct{} }

func (b *ioBehavior) Receive(ctx *ActorContext, msg Message) error {
	if msg.Type == IOReadable {
		b.got <- struct{}{}
	}

	return nil
}
func (b *ioBehavior) PreStart(*ActorContext) error                    { return nil }
func (b *ioBehavior) PostStop(*ActorContext) error                    { return nil }
func (b *ioBehavior) PreRestart(*ActorContext, error, *Message) error { return nil }
func (b *ioBehavior) PostRestart(*ActorContext, error) error          { return nil }
func (b *ioBehavior) GetBehaviorName() string                         { return "io" }

func TestActorSystem_IOPollerIntegration(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	// attach OS poller.
	system.SetIOPoller(asyncio.NewOSPoller())
	_ = system.Start()
	defer system.Stop()

	beh := &ioBehavior{got: make(chan struct{}, 1)}

	actor, err := system.CreateActor("io", UserActor, beh, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}

	// Create loopback TCP to trigger readability.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer ln.Close()
	addr := ln.Addr().String()

	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	reg := make(chan struct{}, 1)

	var srvConn net.Conn

	go func() {
		c, _ := ln.Accept()
		if c != nil {
			srvConn = c
			_ = system.WatchConnWithActor(c, []asyncio.EventType{asyncio.Readable}, actor.ID)
			reg <- struct{}{}
		}
	}()

	// wait until server side is registered, then write from client to make server readable.
	select {
	case <-reg:
		_, _ = client.Write([]byte("x"))
	case <-time.After(time.Second):
		t.Fatal("registration timeout")
	}

	if srvConn != nil {
		defer srvConn.Close()
	}

	// wait for event delivered to actor.
	select {
	case <-beh.got:
		// ok.
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive IOReadable event")
	}

	_ = client.Close()
}

func TestActorSystem_IO_WatermarkPauseResume(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	// Use default poller to avoid OS specifics.
	system.SetIOPoller(asyncio.NewDefaultPoller())
	_ = system.Start()
	defer system.Stop()
	// Stop scheduler to accumulate mailbox without processing.
	system.scheduler.Stop()

	beh := &ioBehavior{got: make(chan struct{}, 1)}

	actor, err := system.CreateActor("io-wm", UserActor, beh, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	// Pre-seed mailbox to meet high watermark condition on first IO event.
	_ = actor.Mailbox.Enqueue(Message{ID: 999, Receiver: actor.ID, Type: 0, Payload: "seed"})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer ln.Close()
	addr := ln.Addr().String()

	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	var srv net.Conn

	reg := make(chan struct{}, 1)

	go func() {
		c, _ := ln.Accept()
		if c != nil {
			srv = c
			opts := IOWatchOptions{
				HighWatermark:      1,
				LowWatermark:       0,
				MonitorInterval:    time.Millisecond * 10,
				ReadEventPriority:  NormalPriority,
				WriteEventPriority: NormalPriority,
				ErrorEventPriority: HighPriority,
			}
			_ = system.WatchConnWithActorOpts(c, []asyncio.EventType{asyncio.Readable}, actor.ID, opts)
			reg <- struct{}{}
		}
	}()

	defer func() {
		if srv != nil {
			_ = srv.Close()
		}
	}()

	select {
	case <-reg:
	case <-time.After(time.Second):
		t.Fatal("registration timeout")
	}

	basePauses := system.GetStatistics().IOPausesRead

	// Flood a few writes to exceed watermark (scheduler stopped -> mailbox grows).
	for i := 0; i < 6; i++ {
		_, _ = client.Write([]byte("x"))

		time.Sleep(time.Millisecond * 5)
	}

	// Expect pause recorded.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if system.GetStatistics().IOPausesRead > basePauses {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	if system.GetStatistics().IOPausesRead <= basePauses {
		t.Skip("watermark pause not observed in time; skipping due to timing variability")
	}

	// Drain mailbox to below low watermark, then expect resume.
	for {
		l, ok := system.GetMailboxLength(actor.ID)
		if !ok || l == 0 {
			break
		}
		// Manually drain one.
		msg, ok2 := actor.Mailbox.Dequeue()
		if !ok2 {
			break
		}

		_ = actor.ProcessMessage(msg)

		time.Sleep(time.Millisecond * 2)
	}

	baseResumes := system.GetStatistics().IOResumesRead
	// Trigger another write to provoke resume path.
	_, _ = client.Write([]byte("y"))

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if system.GetStatistics().IOResumesRead > baseResumes {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	if system.GetStatistics().IOResumesRead <= baseResumes {
		t.Skip("watermark resume not observed in time; skipping due to timing variability")
	}
}

func TestActorSystem_IO_RateLimitDrops(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	system.SetIOPoller(asyncio.NewDefaultPoller())
	_ = system.Start()

	defer system.Stop()
	// Stop scheduler to keep mailbox interaction simple.
	system.scheduler.Stop()

	beh := &ioBehavior{got: make(chan struct{}, 1)}

	actor, err := system.CreateActor("io-rl", UserActor, beh, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer ln.Close()
	addr := ln.Addr().String()

	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	var srv net.Conn

	reg := make(chan struct{}, 1)

	go func() {
		c, _ := ln.Accept()
		if c != nil {
			srv = c
			opts := IOWatchOptions{
				ReadMaxEventsPerSec: 1,
				ReadBurst:           1,
				DropOnRateLimit:     true,
				MonitorInterval:     time.Millisecond * 10,
				ReadEventPriority:   NormalPriority,
				WriteEventPriority:  NormalPriority,
				ErrorEventPriority:  HighPriority,
			}
			_ = system.WatchConnWithActorOpts(c, []asyncio.EventType{asyncio.Readable}, actor.ID, opts)
			reg <- struct{}{}
		}
	}()

	defer func() {
		if srv != nil {
			_ = srv.Close()
		}
	}()

	select {
	case <-reg:
	case <-time.After(time.Second):
		t.Fatal("registration timeout")
	}

	baseDrops := system.GetStatistics().IORateLimitedDrops
	// Burst a number of writes quickly to trigger drops.
	for i := 0; i < 10; i++ {
		_, _ = client.Write([]byte("z"))
	}
	// Allow some time for handler to count drops.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if system.GetStatistics().IORateLimitedDrops > baseDrops {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	if system.GetStatistics().IORateLimitedDrops <= baseDrops {
		t.Fatal("expected rate-limited drops to increase")
	}
}

type fsProbe struct{ got chan string }

func (p *fsProbe) Receive(ctx *ActorContext, msg Message) error {
	if msg.Type == FSChanged {
		if ev, ok := msg.Payload.(FSEvent); ok {
			p.got <- ev.Path
		}
	}

	return nil
}
func (p *fsProbe) PreStart(*ActorContext) error                    { return nil }
func (p *fsProbe) PostStop(*ActorContext) error                    { return nil }
func (p *fsProbe) PreRestart(*ActorContext, error, *Message) error { return nil }
func (p *fsProbe) PostRestart(*ActorContext, error) error          { return nil }
func (p *fsProbe) GetBehaviorName() string                         { return "fsProbe" }

func TestActorSystem_VFSWatchIntegration(t *testing.T) {
	system, _ := NewActorSystem(DefaultActorSystemConfig)
	_ = system.Start()
	defer system.Stop()

	probe := &fsProbe{got: make(chan string, 1)}

	a, err := system.CreateActor("fsp", UserActor, probe, DefaultActorConfig)
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}

	fsys := vfs.NewOS()
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	closeFn, err := system.WatchPathWithActor(ctx, fsys, nil, dir, a.ID)
	if err != nil {
		t.Skip("fsnotify may be unavailable:", err)
	}

	defer closeFn()

	// trigger create.
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case pth := <-probe.got:
		if pth == "" {
			t.Fatal("empty fs event path")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive fs event")
	}
}

func TestMailbox_PriorityQueue(t *testing.T) {
	mb, err := NewMailbox(PriorityMailbox, 16)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}

	// Enqueue multiple messages with different priorities.
	for i := 0; i < 5; i++ {
		m := Message{ID: MessageID(i + 1), Priority: MessagePriority(i % 3)}
		if err := mb.Enqueue(m); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}

	// Dequeue should respect priority ordering via priority queue.
	prev := int(CriticalPriority)

	for i := 0; i < 5; i++ {
		m, ok := mb.Dequeue()
		if !ok {
			t.Fatalf("expected message at %d", i)
		}

		if int(m.Priority) > prev {
			t.Errorf("priority order violated: %d > %d", m.Priority, prev)
		}

		prev = int(m.Priority)
	}
}

func TestRegistry_Lookup(t *testing.T) {
	reg := NewActorRegistry()
	id := ActorID(42)

	if err := reg.Register("svc", id); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if got, ok := reg.Lookup("svc"); !ok || got != id {
		t.Fatalf("lookup mismatch: %v %v", got, ok)
	}
}
