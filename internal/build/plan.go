package build

import (
    "errors"
    "fmt"
    "sort"
)

// TargetID identifies a build target.
type TargetID string

// BuildAction is the work function to materialize a target.
// Must be deterministic and side-effect free outside of target outputs.
// The context may be cancelled for fail-fast or deadline exceeded.
type BuildAction func(ctx Context, target Target) error

// Target describes a node in the dependency graph.
type Target struct {
    ID     TargetID
    Deps   []TargetID
    Weight int // optional relative cost hint; non-positive treated as 1
    Action BuildAction
}

// Plan is a build dependency graph of targets.
type Plan struct {
    nodes map[TargetID]*Target
}

// NewPlan creates an empty build plan.
func NewPlan() *Plan { return &Plan{ nodes: make(map[TargetID]*Target) } }

// AddTarget registers a target. If already present, returns error.
func (p *Plan) AddTarget(t Target) error {
    if t.ID == "" { return errors.New("target ID is empty") }
    if t.Action == nil { return errors.New("target action is nil") }
    if _, ok := p.nodes[t.ID]; ok { return fmt.Errorf("duplicate target: %s", t.ID) }
    if t.Weight <= 0 { t.Weight = 1 }
    // normalize deps (dedupe + deterministic order)
    if len(t.Deps) > 1 {
        sort.Slice(t.Deps, func(i, j int) bool { return t.Deps[i] < t.Deps[j] })
        uniq := t.Deps[:0]
        var prev TargetID
        for i, d := range t.Deps {
            if i == 0 || d != prev { uniq = append(uniq, d); prev = d }
        }
        t.Deps = uniq
    }
    cp := t
    p.nodes[t.ID] = &cp
    return nil
}

// Get returns target by ID.
func (p *Plan) Get(id TargetID) (Target, bool) {
    n, ok := p.nodes[id]
    if !ok { return Target{}, false }
    return *n, true
}

// All returns all targets in deterministic order by ID.
func (p *Plan) All() []Target {
    ids := make([]string, 0, len(p.nodes))
    for id := range p.nodes { ids = append(ids, string(id)) }
    sort.Strings(ids)
    out := make([]Target, 0, len(ids))
    for _, s := range ids { out = append(out, *p.nodes[TargetID(s)]) }
    return out
}

// Subgraph returns the induced subgraph containing requested roots and all their transitive deps.
func (p *Plan) Subgraph(roots []TargetID) (*Plan, error) {
    out := NewPlan()
    visited := make(map[TargetID]bool)
    var dfs func(TargetID) error
    dfs = func(id TargetID) error {
        if visited[id] { return nil }
        t, ok := p.nodes[id]
        if !ok { return fmt.Errorf("unknown target: %s", id) }
        visited[id] = true
        for _, d := range t.Deps {
            if err := dfs(d); err != nil { return err }
        }
        return out.AddTarget(*t)
    }
    for _, r := range roots { if err := dfs(r); err != nil { return nil, err } }
    return out, nil
}

// Validate checks that all deps exist and there are no cycles.
func (p *Plan) Validate() error {
    // existence
    for _, t := range p.nodes {
        for _, d := range t.Deps {
            if _, ok := p.nodes[d]; !ok { return fmt.Errorf("%s depends on missing %s", t.ID, d) }
        }
    }
    // cycle via DFS colors
    const (
        white = 0
        gray  = 1
        black = 2
    )
    color := make(map[TargetID]int, len(p.nodes))
    stack := make([]TargetID, 0, len(p.nodes))
    var visit func(TargetID) error
    visit = func(id TargetID) error {
        switch color[id] {
        case gray:
            // cycle detected, build path for message
            cyc := append([]TargetID(nil), stack...)
            cyc = append(cyc, id)
            return fmt.Errorf("cycle detected: %v", cyc)
        case black:
            return nil
        }
        color[id] = gray
        stack = append(stack, id)
        t := p.nodes[id]
        for _, d := range t.Deps {
            if err := visit(d); err != nil { return err }
        }
        stack = stack[:len(stack)-1]
        color[id] = black
        return nil
    }
    // deterministic iteration
    ids := make([]string, 0, len(p.nodes))
    for id := range p.nodes { ids = append(ids, string(id)) }
    sort.Strings(ids)
    for _, s := range ids {
        if color[TargetID(s)] == white {
            if err := visit(TargetID(s)); err != nil { return err }
        }
    }
    return nil
}


