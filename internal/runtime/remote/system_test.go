package remote

import (
    "testing"
    rt "github.com/orizon-lang/orizon/internal/runtime"
)

type echoBehavior struct{ got chan []byte }
func (e *echoBehavior) Receive(ctx *rt.ActorContext, msg rt.Message) error {
    if b, _ := msg.Payload.([]byte); len(b) > 0 {
        e.got <- b
    }
    return nil
}
func (e *echoBehavior) PreStart(*rt.ActorContext) error { return nil }
func (e *echoBehavior) PostStop(*rt.ActorContext) error { return nil }
func (e *echoBehavior) PreRestart(*rt.ActorContext, error, *rt.Message) error { return nil }
func (e *echoBehavior) PostRestart(*rt.ActorContext, error) error { return nil }
func (e *echoBehavior) GetBehaviorName() string { return "echo" }

// adapter implements LocalDispatcher and NameResolver for wiring.
type adapter struct{ sys *rt.ActorSystem }
func (a adapter) SendMessage(sid uint64, rid uint64, mt uint32, p interface{}) error { return a.sys.SendMessage(rt.ActorID(sid), rt.ActorID(rid), rt.MessageType(mt), p) }
func (a adapter) LookupActorID(name string) (uint64, bool) {
    // use registry
    id, ok := a.sys.registry.Lookup(name)
    return uint64(id), ok
}

type regAdapter struct{ sys *rt.ActorSystem }
func (r regAdapter) Lookup(name string) (uint64, bool) { id, ok := r.sys.registry.Lookup(name); return uint64(id), ok }

func TestRemote_InMemory_SendByName(t *testing.T) {
    // local node A
    a, _ := rt.NewActorSystem(rt.DefaultActorSystemConfig)
    _ = a.Start()
    defer a.Stop()
    eb := &echoBehavior{got: make(chan []byte, 1)}
    _, err := a.CreateActor("svc", rt.UserActor, eb, rt.DefaultActorConfig)
    if err != nil { t.Fatalf("create: %v", err) }

    // remote system for A
    rsA := &RemoteSystem{Trans: &InMemoryTransport{}, Default: JSONCodec{}, Local: adapter{a}, Resolver: regAdapter{a}}
    if err := rsA.Start("A", "A"); err != nil { t.Fatalf("rsA start: %v", err) }
    defer rsA.Stop()

    // remote client B (no local system needed for sending in this test)
    rsB := &RemoteSystem{Trans: &InMemoryTransport{}, Default: JSONCodec{}, Local: adapter{a}, Resolver: regAdapter{a}}
    if err := rsB.Start("B", "B"); err != nil { t.Fatalf("rsB start: %v", err) }
    defer rsB.Stop()

    // send to A.svc from B
    if err := rsB.Send("A", "svc", 1, []byte("ping")); err != nil { t.Fatalf("send: %v", err) }

    // expect payload delivery into echo behavior
    select {
    case b := <-eb.got:
        if string(b) != "ping" { t.Fatalf("unexpected payload: %q", string(b)) }
    }
}


