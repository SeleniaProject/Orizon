package runtime

import (
	"sync"
	"time"
)

// DebugActorSnapshot provides a read-only view of an actor for diagnostics.
type DebugActorSnapshot struct {
	CreateTime    time.Time
	LastHeartbeat time.Time
	Name          string
	Children      []ActorID
	Statistics    ActorStatistics
	ID            ActorID
	Type          ActorType
	State         ActorState
	MailboxLength int
	SupervisorID  SupervisorID
}

// DebugSystemSnapshot aggregates system-wide diagnostics.
type DebugSystemSnapshot struct {
	Time           time.Time
	Actors         []DebugActorSnapshot
	Supervisors    []SupervisorID
	Groups         []ActorGroupID
	SchedulerQueue []int64
	Statistics     ActorSystemStatistics
}

// GetActorSnapshot returns a diagnostic snapshot of a single actor.
func (as *ActorSystem) GetActorSnapshot(aid ActorID) (DebugActorSnapshot, bool) {
	as.mutex.RLock()
	actor := as.actors[aid]
	as.mutex.RUnlock()

	if actor == nil {
		return DebugActorSnapshot{}, false
	}

	actor.mutex.RLock()

	mbLen := 0
	if actor.Mailbox != nil {
		mbLen = actor.Mailbox.Len()
	}

	ch := make([]ActorID, 0, len(actor.Children))
	for id := range actor.Children {
		ch = append(ch, id)
	}

	sup := SupervisorID(0)
	if actor.Supervisor != nil {
		sup = actor.Supervisor.ID
	}

	snap := DebugActorSnapshot{
		ID:            actor.ID,
		Name:          actor.Name,
		Type:          actor.Type,
		State:         actor.State,
		MailboxLength: mbLen,
		SupervisorID:  sup,
		Children:      ch,
		Statistics:    actor.Statistics,
		CreateTime:    actor.CreateTime,
		LastHeartbeat: actor.LastHeartbeat,
	}
	actor.mutex.RUnlock()

	return snap, true
}

// GetSystemSnapshot returns a diagnostic snapshot of the whole system.
func (as *ActorSystem) GetSystemSnapshot() DebugSystemSnapshot {
	as.mutex.RLock()

	actors := make([]*Actor, 0, len(as.actors))
	for _, a := range as.actors {
		actors = append(actors, a)
	}

	sups := make([]SupervisorID, 0, len(as.supervisors))
	for id := range as.supervisors {
		sups = append(sups, id)
	}

	grps := make([]ActorGroupID, 0, len(as.groups))
	for id := range as.groups {
		grps = append(grps, id)
	}

	stats := as.statistics
	as.mutex.RUnlock()

	out := DebugSystemSnapshot{
		Time:           time.Now(),
		Statistics:     stats,
		Actors:         make([]DebugActorSnapshot, 0, len(actors)),
		Supervisors:    sups,
		Groups:         grps,
		SchedulerQueue: as.scheduler.GetQueueLengths(),
	}

	for _, a := range actors {
		if s, ok := as.GetActorSnapshot(a.ID); ok {
			out.Actors = append(out.Actors, s)
		}
	}

	return out
}

// TraceEvent captures a single message transfer for diagnostics.
type TraceEvent struct {
	Time          time.Time
	CorrelationID string
	Sender        ActorID
	Receiver      ActorID
	Priority      MessagePriority
	MessageID     MessageID
	Type          MessageType
}

// MessageTracer records recent message events per actor in ring buffers.
type MessageTracer struct {
	perActor       map[ActorID]*traceRing
	perCorrelation map[string]*traceRing
	perSize        int
	mu             sync.RWMutex
}

type traceRing struct {
	buf   []TraceEvent
	head  int
	count int
	mu    sync.Mutex
}

func newTraceRing(size int) *traceRing {
	if size <= 0 {
		size = 128
	}

	return &traceRing{buf: make([]TraceEvent, size)}
}

func (r *traceRing) add(ev TraceEvent) {
	r.mu.Lock()
	r.buf[r.head] = ev
	r.head = (r.head + 1) % len(r.buf)

	if r.count < len(r.buf) {
		r.count++
	}
	r.mu.Unlock()
}

func (r *traceRing) lastN(n int) []TraceEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	if n <= 0 || n > r.count {
		n = r.count
	}

	out := make([]TraceEvent, n)

	idx := r.head
	for i := 0; i < n; i++ {
		idx = (idx - 1 + len(r.buf)) % len(r.buf)
		out[n-1-i] = r.buf[idx]
	}

	return out
}

// newMessageTracer constructs a tracer with per-actor ring buffer size.
func newMessageTracer(size int) *MessageTracer {
	return &MessageTracer{perActor: make(map[ActorID]*traceRing), perSize: size, perCorrelation: make(map[string]*traceRing)}
}

// record stores an event for both sender and receiver.
func (t *MessageTracer) record(ev TraceEvent) {
	t.mu.RLock()
	sring := t.perActor[ev.Sender]
	rring := t.perActor[ev.Receiver]

	cring := (*traceRing)(nil)
	if ev.CorrelationID != "" {
		cring = t.perCorrelation[ev.CorrelationID]
	}
	t.mu.RUnlock()

	if sring == nil {
		sring = newTraceRing(t.perSize)
		t.mu.Lock()
		if _, ok := t.perActor[ev.Sender]; !ok {
			t.perActor[ev.Sender] = sring
		}
		t.mu.Unlock()
	}

	if rring == nil {
		rring = newTraceRing(t.perSize)
		t.mu.Lock()
		if _, ok := t.perActor[ev.Receiver]; !ok {
			t.perActor[ev.Receiver] = rring
		}
		t.mu.Unlock()
	}

	if ev.CorrelationID != "" && cring == nil {
		cring = newTraceRing(t.perSize)
		t.mu.Lock()
		if _, ok := t.perCorrelation[ev.CorrelationID]; !ok {
			t.perCorrelation[ev.CorrelationID] = cring
		}
		t.mu.Unlock()
	}

	sring.add(ev)

	if ev.Sender != ev.Receiver {
		rring.add(ev)
	}

	if ev.CorrelationID != "" {
		cring.add(ev)
	}
}

// getLastN returns last n events involving the actor id.
func (t *MessageTracer) getLastN(aid ActorID, n int) []TraceEvent {
	t.mu.RLock()
	r := t.perActor[aid]
	t.mu.RUnlock()

	if r == nil {
		return nil
	}

	return r.lastN(n)
}

// getByCorrelation returns last n events associated with the correlation id.
func (t *MessageTracer) getByCorrelation(corrID string, n int) []TraceEvent {
	if corrID == "" {
		return nil
	}

	t.mu.RLock()
	r := t.perCorrelation[corrID]
	t.mu.RUnlock()

	if r == nil {
		return nil
	}

	return r.lastN(n)
}

// EnableTracing turns on message tracing (thread-safe). If already enabled, it resets buffer size.
func (as *ActorSystem) EnableTracing(bufferPerActor int) {
	as.mutex.Lock()
	as.tracer = newMessageTracer(bufferPerActor)
	as.config.EnableTracing = true
	as.mutex.Unlock()
}

// DisableTracing turns off message tracing.
func (as *ActorSystem) DisableTracing() {
	as.mutex.Lock()
	as.tracer = nil
	as.config.EnableTracing = false
	as.mutex.Unlock()
}

// GetRecentMessages returns up to n recent trace events for an actor.
func (as *ActorSystem) GetRecentMessages(aid ActorID, n int) []TraceEvent {
	as.mutex.RLock()
	tracer := as.tracer
	as.mutex.RUnlock()

	if tracer == nil {
		return nil
	}

	return tracer.getLastN(aid, n)
}

// traceMessage records a trace event if tracing is enabled.
func (as *ActorSystem) traceMessage(sender, receiver ActorID, mt MessageType, pr MessagePriority, corrID string, mid MessageID) {
	as.mutex.RLock()
	tracer := as.tracer
	as.mutex.RUnlock()

	if tracer == nil {
		return
	}

	tracer.record(TraceEvent{Time: time.Now(), Sender: sender, Receiver: receiver, Type: mt, Priority: pr, CorrelationID: corrID, MessageID: mid})
}

// GetCorrelationEvents returns up to n recent events for a correlation id.
func (as *ActorSystem) GetCorrelationEvents(correlationID string, n int) []TraceEvent {
	as.mutex.RLock()
	tracer := as.tracer
	as.mutex.RUnlock()

	if tracer == nil {
		return nil
	}

	return tracer.getByCorrelation(correlationID, n)
}

// DebugActorGraph represents the actor relationship graph for diagnostics.
type DebugActorGraph struct {
	GeneratedAt time.Time             `json:"generatedAt"`
	Nodes       []DebugActorGraphNode `json:"nodes"`
	Edges       []DebugActorGraphEdge `json:"edges"`
}

// DebugActorGraphNode describes a single actor node and its attributes.
type DebugActorGraphNode struct {
	CreateTime    time.Time       `json:"createTime"`
	LastHeartbeat time.Time       `json:"lastHeartbeat"`
	Name          string          `json:"name"`
	GroupIDs      []ActorGroupID  `json:"groupIds"`
	Statistics    ActorStatistics `json:"statistics"`
	ID            ActorID         `json:"id"`
	Type          ActorType       `json:"type"`
	State         ActorState      `json:"state"`
	MailboxLength int             `json:"mailboxLength"`
	SupervisorID  SupervisorID    `json:"supervisorId"`
}

// DebugActorGraphEdgeKind classifies edge semantics.
type DebugActorGraphEdgeKind string

const (
	EdgeSupervises DebugActorGraphEdgeKind = "supervises"
	EdgeWatched    DebugActorGraphEdgeKind = "watched"      // from -> watched target
	EdgeWatching   DebugActorGraphEdgeKind = "watching"     // from -> is watching target
	EdgeGroup      DebugActorGraphEdgeKind = "group-member" // group context represented on node GroupIDs
)

// DebugActorGraphEdge encodes a directed relationship between actors.
type DebugActorGraphEdge struct {
	Kind DebugActorGraphEdgeKind `json:"kind"`
	From ActorID                 `json:"from"`
	To   ActorID                 `json:"to"`
}

// BuildActorGraph constructs a graph snapshot of actors, supervision, watch relations, and groups.
func (as *ActorSystem) BuildActorGraph() DebugActorGraph {
	// Snapshot actors and supervisors/groups under read locks
	as.mutex.RLock()

	actors := make([]*Actor, 0, len(as.actors))
	for _, a := range as.actors {
		actors = append(actors, a)
	}

	groups := make(map[ActorID][]ActorGroupID)
	for gid, grp := range as.groups {
		for aid := range grp.Members {
			groups[aid] = append(groups[aid], gid)
		}
	}
	as.mutex.RUnlock()

	out := DebugActorGraph{GeneratedAt: time.Now(), Nodes: make([]DebugActorGraphNode, 0, len(actors))}
	// Build nodes.
	for _, a := range actors {
		a.mutex.RLock()

		mbLen := 0
		if a.Mailbox != nil {
			mbLen = a.Mailbox.Len()
		}

		sup := SupervisorID(0)
		if a.Supervisor != nil {
			sup = a.Supervisor.ID
		}

		node := DebugActorGraphNode{
			ID:            a.ID,
			Name:          a.Name,
			Type:          a.Type,
			State:         a.State,
			MailboxLength: mbLen,
			SupervisorID:  sup,
			GroupIDs:      groups[a.ID],
			Statistics:    a.Statistics,
			CreateTime:    a.CreateTime,
			LastHeartbeat: a.LastHeartbeat,
		}
		out.Nodes = append(out.Nodes, node)
		a.mutex.RUnlock()
	}
	// Build edges for supervision and watch relations.
	edges := make([]DebugActorGraphEdge, 0)

	for _, a := range actors {
		a.mutex.RLock()
		if a.Supervisor != nil {
			// parent supervision edge from supervisor to actor.
			edges = append(edges, DebugActorGraphEdge{From: ActorID(a.Supervisor.ID), To: a.ID, Kind: EdgeSupervises})
		}
		// watching edges: from actor to targets it watches.
		if a.Context != nil && a.Context.Watched != nil {
			for target := range a.Context.Watched {
				edges = append(edges, DebugActorGraphEdge{From: a.ID, To: target, Kind: EdgeWatching})
				// For convenience also add reverse semantic label.
				edges = append(edges, DebugActorGraphEdge{From: target, To: a.ID, Kind: EdgeWatched})
			}
		}
		a.mutex.RUnlock()
	}

	out.Edges = edges

	return out
}

// DebugDeadlockReport describes a potential deadlock cycle with involved actors and rationale.
type DebugDeadlockReport struct {
	Kind       string    `json:"kind"`       // e.g., "watch-cycle"
	ActorIDs   []ActorID `json:"actorIds"`   // involved actor ids in cycle order
	ActorNames []string  `json:"actorNames"` // names aligned with ids
	States     []int     `json:"states"`     // ActorState enum values aligned with ids
	Size       int       `json:"size"`       // cycle size
}

// DetectPotentialDeadlocks scans known dependency relations and reports cycles.
// Current implementation analyzes the directed graph induced by watch relations.
// (A -> B when A watches B). Strongly connected components of size >= 2 (or self-loop)
// are reported as potential deadlocks. States are included to help severity assessment.
func (as *ActorSystem) DetectPotentialDeadlocks() []DebugDeadlockReport {
	// Build adjacency on watches.
	as.mutex.RLock()

	actors := make(map[ActorID]*Actor, len(as.actors))
	for id, a := range as.actors {
		actors[id] = a
	}
	as.mutex.RUnlock()

	adj := make(map[ActorID][]ActorID, len(actors))

	for id, a := range actors {
		a.mutex.RLock()
		if a.Context != nil && a.Context.Watched != nil {
			for target := range a.Context.Watched {
				adj[id] = append(adj[id], target)
			}
		}
		a.mutex.RUnlock()

		if _, ok := adj[id]; !ok {
			adj[id] = nil
		}
	}
	// Tarjan SCC.
	index := 0

	type frame struct {
		index int
		low   int
		onstk bool
	}

	frames := make(map[ActorID]*frame, len(adj))
	stack := make([]ActorID, 0, len(adj))

	var reports []DebugDeadlockReport

	var strongconnect func(v ActorID)
	strongconnect = func(v ActorID) {
		f := &frame{index: index, low: index, onstk: true}
		frames[v] = f
		index++

		stack = append(stack, v)

		for _, w := range adj[v] {
			fw, ok := frames[w]
			if !ok {
				strongconnect(w)

				if fw = frames[w]; fw != nil && fw.low < f.low {
					f.low = fw.low
				}
			} else if fw.onstk {
				if fw.index < f.low {
					f.low = fw.index
				}
			}
		}

		if f.low == f.index {
			// root of SCC.
			comp := make([]ActorID, 0)

			for {
				if len(stack) == 0 {
					break
				}

				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				fw := frames[w]
				if fw != nil {
					fw.onstk = false
				}

				comp = append(comp, w)

				if w == v {
					break
				}
			}
			// Determine cycle condition: size>=2 or self-loop.
			isCycle := false
			if len(comp) >= 2 {
				isCycle = true
			} else if len(comp) == 1 {
				// self-loop.
				u := comp[0]
				for _, t := range adj[u] {
					if t == u {
						isCycle = true

						break
					}
				}
			}

			if isCycle {
				names := make([]string, len(comp))
				states := make([]int, len(comp))

				for i, id := range comp {
					if a := actors[id]; a != nil {
						a.mutex.RLock()
						names[i] = a.Name
						states[i] = int(a.State)
						a.mutex.RUnlock()
					}
				}

				reports = append(reports, DebugDeadlockReport{Kind: "watch-cycle", ActorIDs: comp, ActorNames: names, States: states, Size: len(comp)})
			}
		}
	}

	for v := range adj {
		if _, seen := frames[v]; !seen {
			strongconnect(v)
		}
	}

	return reports
}
