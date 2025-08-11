package runtime

import (
    "fmt"
    "testing"
    "time"
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

func TestActor_MessageFlow_ManualDispatch(t *testing.T) {
	system, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("failed to create actor system: %v", err)
	}
	if err := system.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer system.Stop()

	tb := &testBehavior{received: make(chan Message, 1), name: "test"}
	actor, err := system.CreateActor("echo", UserActor, tb, DefaultActorConfig)
	if err != nil {
		t.Fatalf("failed to create actor: %v", err)
	}

	// Send via system (enqueues to mailbox)
	if err := system.SendMessage(0, actor.ID, 1, "hello"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	// Manually dispatch: pop from mailbox and process (scheduler is decoupled)
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

func TestActorSystem_AutoDispatch(t *testing.T) {
    system, err := NewActorSystem(DefaultActorSystemConfig)
    if err != nil { t.Fatalf("failed to create actor system: %v", err) }
    if err := system.Start(); err != nil { t.Fatalf("failed to start: %v", err) }
    defer system.Stop()

    tb := &testBehavior{received: make(chan Message, 1), name: "auto"}
    actor, err := system.CreateActor("auto", UserActor, tb, DefaultActorConfig)
    if err != nil { t.Fatalf("failed to create actor: %v", err) }

    // Send message; scheduler worker should auto-dispatch and process it
    if err := system.SendMessage(0, actor.ID, 1, "ping"); err != nil {
        t.Fatalf("send failed: %v", err)
    }

    select {
    case got := <-tb.received:
        if got.Payload != "ping" { t.Errorf("unexpected payload: %v", got.Payload) }
    case <-time.After(1500 * time.Millisecond):
        t.Fatal("auto dispatch did not deliver message in time")
    }
}

// Simple interceptor/transformer for testing dispatcher pipeline
type interceptOnce struct{ called *bool }
func (i interceptOnce) Intercept(msg Message) (Message, error) { *i.called = true; return msg, nil }

type transformTag struct{ key string }
func (t transformTag) Transform(msg Message) (Message, error) {
    if msg.Headers == nil { msg.Headers = map[string]interface{}{} }
    msg.Headers[t.key] = true
    return msg, nil
}

func (t transformTag) GetTransformerName() string { return "transformTag" }

func (i interceptOnce) GetInterceptorName() string { return "interceptOnce" }

func TestDispatcher_Interception_Transformation(t *testing.T) {
    system, err := NewActorSystem(DefaultActorSystemConfig)
    if err != nil { t.Fatalf("failed to create actor system: %v", err) }
    if err := system.Start(); err != nil { t.Fatalf("failed to start: %v", err) }
    defer system.Stop()

    tb := &testBehavior{received: make(chan Message, 1), name: "pipe"}
    actor, err := system.CreateActor("pipe", UserActor, tb, DefaultActorConfig)
    if err != nil { t.Fatalf("failed to create actor: %v", err) }

    // Install interceptor and transformer
    called := false
    system.dispatcher.mutex.Lock()
    system.dispatcher.interceptors = append(system.dispatcher.interceptors, interceptOnce{called: &called})
    system.dispatcher.transformers = append(system.dispatcher.transformers, transformTag{key: "tagged"})
    system.dispatcher.mutex.Unlock()

    if err := system.SendMessage(0, actor.ID, 1, "x"); err != nil { t.Fatalf("send failed: %v", err) }

    select {
    case got := <-tb.received:
        if !called { t.Error("interceptor not called") }
        if got.Headers["tagged"] != true { t.Error("transformer did not tag header") }
    case <-time.After(time.Second):
        t.Fatal("did not receive message via pipeline")
    }
}

func TestSupervisor_Restart_OnFailure(t *testing.T) {
    system, err := NewActorSystem(DefaultActorSystemConfig)
    if err != nil { t.Fatalf("failed to create actor system: %v", err) }
    if err := system.Start(); err != nil { t.Fatalf("failed to start: %v", err) }
    defer system.Stop()

    tb := &testBehavior{received: make(chan Message, 1), name: "fail", failOnce: true}
    actor, err := system.CreateActor("fail", UserActor, tb, DefaultActorConfig)
    if err != nil { t.Fatalf("failed to create actor: %v", err) }

    // First message will fail; supervisor should restart actor; second message should succeed
    if err := system.SendMessage(0, actor.ID, 1, "first"); err != nil { t.Fatalf("send failed: %v", err) }
    if err := system.SendMessage(0, actor.ID, 1, "second"); err != nil { t.Fatalf("send failed: %v", err) }

    // Expect at least one successful receipt
    select {
    case <-tb.received:
        // ok
    case <-time.After(1500 * time.Millisecond):
        t.Fatal("supervisor did not restart actor in time")
    }
}

// behavior that always fails to test restart limits
type failAlways struct{ count int }
func (f *failAlways) Receive(ctx *ActorContext, msg Message) error { f.count++; return fmt.Errorf("boom") }
func (f *failAlways) PreStart(*ActorContext) error { return nil }
func (f *failAlways) PostStop(*ActorContext) error { return nil }
func (f *failAlways) PreRestart(*ActorContext, error, *Message) error { return nil }
func (f *failAlways) PostRestart(*ActorContext, error) error { return nil }
func (f *failAlways) GetBehaviorName() string { return "failAlways" }

func TestSupervisor_RestartLimits_StopAfterMaxRetries(t *testing.T) {
    system, _ := NewActorSystem(DefaultActorSystemConfig)
    _ = system.Start()
    defer system.Stop()

    // Configure root supervisor restart window
    system.rootSupervisor.MaxRetries = 2
    system.rootSupervisor.RetryPeriod = time.Millisecond * 200

    fb := &failAlways{}
    a, _ := system.CreateActor("lim", UserActor, fb, DefaultActorConfig)

    // Send multiple failing messages beyond limit
    _ = system.SendMessage(0, a.ID, 1, "x")
    _ = system.SendMessage(0, a.ID, 1, "y")
    _ = system.SendMessage(0, a.ID, 1, "z")

    // Allow time for processing and potential stops
    time.Sleep(time.Millisecond * 600)

    // Actor should be stopped after exceeding retries
    system.mutex.RLock()
    got := system.actors[a.ID]
    system.mutex.RUnlock()
    if got != nil && got.State != ActorStopped {
        t.Fatalf("expected actor stopped after max retries, got state=%v", got.State)
    }
}

func TestSupervisor_OneForAll(t *testing.T) {
    system, _ := NewActorSystem(DefaultActorSystemConfig)
    _ = system.Start()
    defer system.Stop()

    // Configure root as OneForAll + Restart
    system.rootSupervisor.Strategy = RestartStrategy
    system.rootSupervisor.Type = OneForAll

    tb1 := &testBehavior{received: make(chan Message, 2), name: "a", failOnce: true}
    tb2 := &testBehavior{received: make(chan Message, 2), name: "b"}
    a, _ := system.CreateActor("a", UserActor, tb1, DefaultActorConfig)
    b, _ := system.CreateActor("b", UserActor, tb2, DefaultActorConfig)

    _ = system.SendMessage(0, a.ID, 1, "x") // will fail and trigger restart for all
    _ = system.SendMessage(0, a.ID, 1, "y")
    _ = system.SendMessage(0, b.ID, 1, "z")

    // We only assert no panic and that at least one message is processed after restarts
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

    // Configure root as RestForOne + Restart
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

    // Fill capacity
    _ = mb.Enqueue(Message{ID: 1})
    _ = mb.Enqueue(Message{ID: 2})

    // Enqueue with back pressure should timeout eventually
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
    if err != nil { t.Fatalf("create group failed: %v", err) }

    tb1 := &testBehavior{received: make(chan Message, 1), name: "g1"}
    tb2 := &testBehavior{received: make(chan Message, 1), name: "g2"}
    a1, _ := system.CreateActor("g1", UserActor, tb1, DefaultActorConfig)
    a2, _ := system.CreateActor("g2", UserActor, tb2, DefaultActorConfig)
    if err := system.AddToGroup(g.ID, a1.ID); err != nil { t.Fatalf("add1: %v", err) }
    if err := system.AddToGroup(g.ID, a2.ID); err != nil { t.Fatalf("add2: %v", err) }

    if err := system.Broadcast(g.ID, 1, "B"); err != nil { t.Fatalf("broadcast: %v", err) }

    // Expect both to receive
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
    actor.Context.StartTimer("once", time.Millisecond*10, func(){ fired <- struct{}{} })

    select {
    case <-fired:
        // ok
    case <-time.After(time.Second):
        t.Fatal("timer did not fire")
    }

    // Restart timer then stop it before firing
    actor.Context.StartTimer("stop", time.Millisecond*50, func(){ fired <- struct{}{} })
    actor.Context.StopTimer("stop")
    select {
    case <-fired:
        t.Fatal("stopped timer should not fire")
    case <-time.After(time.Millisecond*80):
        // ok
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

    // send with receiver=a1, but route by type to a2
    if err := system.SendMessage(a1.ID, a1.ID, 100, "route"); err != nil { t.Fatalf("send failed: %v", err) }

    select {
    case <-tb2.received:
        // ok, routed
    case <-time.After(time.Second):
        t.Fatal("route did not deliver to target")
    }
}

type pingBehavior struct{ got chan string }
func (p *pingBehavior) Receive(ctx *ActorContext, msg Message) error {
    if s, _ := msg.Payload.(string); s != "" { p.got <- s }
    return nil
}
func (p *pingBehavior) PreStart(*ActorContext) error { return nil }
func (p *pingBehavior) PostStop(*ActorContext) error { return nil }
func (p *pingBehavior) PreRestart(*ActorContext, error, *Message) error { return nil }
func (p *pingBehavior) PostRestart(*ActorContext, error) error { return nil }
func (p *pingBehavior) GetBehaviorName() string { return "ping" }

func TestActor_SpawnAndTell(t *testing.T) {
    system, _ := NewActorSystem(DefaultActorSystemConfig)
    _ = system.Start()
    defer system.Stop()

    pb := &pingBehavior{got: make(chan string, 1)}
    ref, err := system.Spawn("p", pb, DefaultActorConfig)
    if err != nil { t.Fatalf("spawn failed: %v", err) }

    if err := ref.Tell(1, "hello"); err != nil { t.Fatalf("tell failed: %v", err) }
    select {
    case s := <-pb.got:
        if s != "hello" { t.Fatalf("unexpected: %s", s) }
    case <-time.After(time.Second):
        t.Fatal("did not receive ping")
    }
}

func TestMailbox_PriorityQueue(t *testing.T) {
	mb, err := NewMailbox(PriorityMailbox, 16)
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}

	// Enqueue multiple messages with different priorities
	for i := 0; i < 5; i++ {
		m := Message{ID: MessageID(i + 1), Priority: MessagePriority(i % 3)}
		if err := mb.Enqueue(m); err != nil {
			t.Fatalf("enqueue failed: %v", err)
		}
	}

	// Dequeue should respect priority ordering via priority queue
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
