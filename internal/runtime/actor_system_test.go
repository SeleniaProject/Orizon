package runtime

import (
	"testing"
	"time"
)

// testBehavior is a simple actor behavior used for testing.
type testBehavior struct {
	received chan Message
	name     string
}

func (tb *testBehavior) Receive(ctx *ActorContext, msg Message) error {
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
