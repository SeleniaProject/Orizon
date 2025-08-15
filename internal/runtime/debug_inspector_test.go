package runtime

import (
	"encoding/json"
	"net/http"
	"testing"
)

// noopBehavior implements ActorBehavior with no-op handlers.
type noopBehavior struct{}

func (n *noopBehavior) Receive(ctx *ActorContext, msg Message) error { return nil }
func (n *noopBehavior) PreStart(ctx *ActorContext) error             { return nil }
func (n *noopBehavior) PostStop(ctx *ActorContext) error             { return nil }
func (n *noopBehavior) PreRestart(ctx *ActorContext, reason error, message *Message) error {
	return nil
}
func (n *noopBehavior) PostRestart(ctx *ActorContext, reason error) error { return nil }
func (n *noopBehavior) GetBehaviorName() string                           { return "noop" }

func TestDebugInspector_GraphAndDeadlocks(t *testing.T) {
	as, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("NewActorSystem: %v", err)
	}
	if err := as.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = as.Stop() }()

	a1, err := as.CreateActor("a1", UserActor, &noopBehavior{}, DefaultActorConfig)
	if err != nil {
		t.Fatalf("CreateActor a1: %v", err)
	}
	a2, err := as.CreateActor("a2", UserActor, &noopBehavior{}, DefaultActorConfig)
	if err != nil {
		t.Fatalf("CreateActor a2: %v", err)
	}

	// Establish mutual watch to form a cycle
	a1.Context.Watch(a2.ID)
	a2.Context.Watch(a1.ID)

	graph := as.BuildActorGraph()
	if len(graph.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(graph.Nodes))
	}
	// Expect at least two edges for watch relations (watching and watched)
	watchEdges := 0
	for _, e := range graph.Edges {
		if e.Kind == EdgeWatching || e.Kind == EdgeWatched {
			watchEdges++
		}
	}
	if watchEdges < 2 {
		t.Fatalf("expected >=2 watch edges, got %d", watchEdges)
	}

	dead := as.DetectPotentialDeadlocks()
	if len(dead) == 0 {
		t.Fatalf("expected at least one deadlock cycle")
	}
}

func TestDebugHTTP_GraphAndDeadlocks(t *testing.T) {
	as, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("NewActorSystem: %v", err)
	}
	if err := as.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = as.Stop() }()

	a1, _ := as.CreateActor("a1", UserActor, &noopBehavior{}, DefaultActorConfig)
	a2, _ := as.CreateActor("a2", UserActor, &noopBehavior{}, DefaultActorConfig)
	// Create a watch cycle
	a1.Context.Watch(a2.ID)
	a2.Context.Watch(a1.ID)

	shutdown, bound, err := StartDebugHTTPOn(as, ":0")
	if err != nil {
		t.Fatalf("StartDebugHTTPOn: %v", err)
	}
	defer func() { _ = shutdown(as.ctx) }()

	// Fetch graph
	resp, err := http.Get("http://" + bound + "/actors/graph")
	if err != nil || resp == nil || resp.Body == nil {
		t.Fatalf("GET graph: %v", err)
	}
	var gObj map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&gObj); err != nil {
		t.Fatalf("decode graph: %v", err)
	}
	_ = resp.Body.Close()
	if _, ok := gObj["nodes"]; !ok {
		t.Fatalf("graph missing nodes")
	}

	// Fetch deadlocks
	resp2, err := http.Get("http://" + bound + "/actors/deadlocks")
	if err != nil || resp2 == nil || resp2.Body == nil {
		t.Fatalf("GET deadlocks: %v", err)
	}
	var dArr []any
	if err := json.NewDecoder(resp2.Body).Decode(&dArr); err != nil {
		t.Fatalf("decode deadlocks: %v", err)
	}
	_ = resp2.Body.Close()
	if len(dArr) == 0 {
		t.Fatalf("expected non-empty deadlocks array")
	}
}

func TestDebugHTTP_Correlation(t *testing.T) {
	as, err := NewActorSystem(DefaultActorSystemConfig)
	if err != nil {
		t.Fatalf("NewActorSystem: %v", err)
	}
	if err := as.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer as.Stop()

	// Install a tracer and generate a correlated message
	as.EnableTracing(16)
	tb := &noopBehavior{}
	a1, _ := as.CreateActor("c1", UserActor, tb, DefaultActorConfig)
	a2, _ := as.CreateActor("c2", UserActor, tb, DefaultActorConfig)
	cid := NewCorrelationID()
	_ = as.SendMessageWithCorrelation(a1.ID, a2.ID, 1, "p", cid)

	shutdown, bound, err := StartDebugHTTPOn(as, ":0")
	if err != nil {
		t.Fatalf("StartDebugHTTPOn: %v", err)
	}
	defer func() { _ = shutdown(as.ctx) }()

	// Query correlation events
	resp, err := http.Get("http://" + bound + "/actors/correlation?id=" + cid + "&n=10")
	if err != nil || resp == nil || resp.Body == nil {
		t.Fatalf("GET correlation: %v", err)
	}
	var arr []any
	if err := json.NewDecoder(resp.Body).Decode(&arr); err != nil {
		t.Fatalf("decode correlation: %v", err)
	}
	_ = resp.Body.Close()
	if len(arr) == 0 {
		t.Fatalf("expected correlation events, got empty")
	}
}
