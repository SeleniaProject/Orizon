package build

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"
)

func TestPlan_ValidateCycle(t *testing.T) {
    p := NewPlan()
    if err := p.AddTarget(Target{ ID: "A", Deps: []TargetID{"B"}, Action: func(ctx Context, t Target) error { return nil } }); err != nil { t.Fatal(err) }
    if err := p.AddTarget(Target{ ID: "B", Deps: []TargetID{"A"}, Action: func(ctx Context, t Target) error { return nil } }); err != nil { t.Fatal(err) }
    if err := p.Validate(); err == nil { t.Fatalf("expected cycle error") }
}

func TestExecutor_BasicDAG(t *testing.T) {
    p := NewPlan()
    var order int64
    mk := func(id string, deps ...TargetID) Target {
        return Target{ ID: TargetID(id), Deps: deps, Action: func(ctx Context, t Target) error { time.Sleep(10 * time.Millisecond); atomic.AddInt64(&order, 1); return nil } }
    }
    if err := p.AddTarget(mk("A")); err != nil { t.Fatal(err) }
    if err := p.AddTarget(mk("B", "A")); err != nil { t.Fatal(err) }
    if err := p.AddTarget(mk("C", "A")); err != nil { t.Fatal(err) }
    if err := p.AddTarget(mk("D", "B", "C")); err != nil { t.Fatal(err) }
    if err := p.Validate(); err != nil { t.Fatal(err) }

    ex := NewExecutor(4)
    res, stats, err := ex.Execute(context.Background(), p, nil)
    if err != nil { t.Fatalf("execute failed: %v", err) }
    if len(res) != 4 { t.Fatalf("unexpected results: %d", len(res)) }
    if stats.Failed != 0 || stats.Succeeded != 4 { t.Fatalf("bad stats: %+v", stats) }
}

func TestExecutor_FailurePropagation(t *testing.T) {
    p := NewPlan()
    mkOK := func(id string, deps ...TargetID) Target {
        return Target{ ID: TargetID(id), Deps: deps, Action: func(ctx Context, t Target) error { return nil } }
    }
    mkFail := func(id string, deps ...TargetID) Target {
        return Target{ ID: TargetID(id), Deps: deps, Action: func(ctx Context, t Target) error { return errors.New("fail") } }
    }
    if err := p.AddTarget(mkOK("A")); err != nil { t.Fatal(err) }
    if err := p.AddTarget(mkFail("B", "A")); err != nil { t.Fatal(err) }
    if err := p.AddTarget(mkOK("C", "B")); err != nil { t.Fatal(err) }
    if err := p.Validate(); err != nil { t.Fatal(err) }

    ex := NewExecutor(2)
    res, stats, err := ex.Execute(context.Background(), p, nil)
    if err != nil { t.Fatalf("execute failed: %v", err) }
    if len(res) != 3 { t.Fatalf("unexpected results: %d", len(res)) }
    if stats.Failed == 0 { t.Fatalf("expected failure count > 0") }
}


