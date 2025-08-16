package build

import (
	"context"
	"errors"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Context wraps context.Context to let BuildActions access shared services if needed later.
type Context interface {
	context.Context
}

// DefaultContext is a thin wrapper over context.Context.
type DefaultContext struct{ context.Context }

// Result captures execution outcome per target.
type Result struct {
	ID   TargetID
	Err  error
	Took time.Duration
}

// Stats holds simple execution statistics.
type Stats struct {
	TotalTargets int64
	Succeeded    int64
	Failed       int64
	Enqueued     int64
	Dequeued     int64
	MaxParallel  int64
}

// Executor runs a Plan with N workers honoring dependencies.
type Executor struct {
	workers int
}

// NewExecutor constructs an Executor with a given worker count (<=0 => NumCPU).
func NewExecutor(workers int) *Executor {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Executor{workers: workers}
}

// Execute runs the plan for requested roots (or whole plan if roots empty) and returns results ordered by start time.
func (e *Executor) Execute(ctx context.Context, plan *Plan, roots []TargetID) ([]Result, Stats, error) {
	if plan == nil {
		return nil, Stats{}, errors.New("nil plan")
	}
	sub := plan
	var err error
	if len(roots) > 0 {
		sub, err = plan.Subgraph(roots)
		if err != nil {
			return nil, Stats{}, err
		}
	}
	if err := sub.Validate(); err != nil {
		return nil, Stats{}, err
	}

	// Build reverse dep graph and initial indegrees
	indeg := make(map[TargetID]int, len(sub.nodes))
	rev := make(map[TargetID][]TargetID, len(sub.nodes))
	for id, t := range sub.nodes {
		for _, d := range t.Deps {
			indeg[id]++
			rev[d] = append(rev[d], id)
		}
	}
	// Ready queue (min-heap by current queue length of worker; simplified as FIFO with weight priority)
	type item struct {
		id     TargetID
		weight int
	}
	ready := make([]item, 0)
	for id, t := range sub.nodes {
		if indeg[id] == 0 {
			ready = append(ready, item{id: id, weight: t.Weight})
		}
	}
	// stable order for determinism
	sort.Slice(ready, func(i, j int) bool {
		if ready[i].weight == ready[j].weight {
			return ready[i].id < ready[j].id
		}
		return ready[i].weight > ready[j].weight
	})

	// Work channels
	type job struct{ id TargetID }
	jobs := make(chan job, len(sub.nodes))
	var wg sync.WaitGroup
	results := make([]Result, 0, len(sub.nodes))
	var resultsMu sync.Mutex
	var stats Stats
	stats.TotalTargets = int64(len(sub.nodes))

	var running int64
	// Worker function
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			t := sub.nodes[j.id]
			start := time.Now()
			err := t.Action(DefaultContext{ctx}, *t)
			took := time.Since(start)
			atomic.AddInt64(&running, -1)
			resultsMu.Lock()
			results = append(results, Result{ID: j.id, Err: err, Took: took})
			if err != nil {
				stats.Failed++
			} else {
				stats.Succeeded++
			}
			resultsMu.Unlock()
			// Notify dependents
			for _, dep := range rev[j.id] {
				left := 0
				if indeg[dep] > 0 {
					indeg[dep]--
				}
				left = indeg[dep]
				if left == 0 {
					// enqueue
					stats.Enqueued++
					jobs <- job{id: dep}
				}
			}
		}
	}

	// Start workers
	wg.Add(e.workers)
	for i := 0; i < e.workers; i++ {
		go worker()
	}

	// Submit initial ready set
	for _, it := range ready {
		stats.Enqueued++
		atomic.AddInt64(&running, 1)
		if cur := atomic.LoadInt64(&running); cur > stats.MaxParallel {
			stats.MaxParallel = cur
		}
		jobs <- job{id: it.id}
	}

	// Close jobs channel when all results are collected
	go func() {
		wg.Wait()
	}()

	// Wait for completion by watching counts
	for {
		time.Sleep(1 * time.Millisecond)
		if int(stats.Succeeded+stats.Failed) == len(sub.nodes) {
			break
		}
		// backpressure stats
		if cur := atomic.LoadInt64(&running); cur > stats.MaxParallel {
			stats.MaxParallel = cur
		}
	}
	close(jobs)

	// Deterministic order by start time already approximated by append order. As a final step, sort by ID.
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results, stats, nil
}

// ExecuteSelected runs only the selected targets, honoring dependencies among the selection.
// Dependencies not in the selection are assumed up-to-date and are not executed.
func (e *Executor) ExecuteSelected(ctx context.Context, plan *Plan, selectedIDs []TargetID) ([]Result, Stats, error) {
	if plan == nil {
		return nil, Stats{}, errors.New("nil plan")
	}
	selected := make(map[TargetID]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}
	// Build induced subgraph over selection
	indeg := make(map[TargetID]int, len(selected))
	rev := make(map[TargetID][]TargetID, len(selected))
	for id := range selected {
		indeg[id] = 0
	}
	for id := range selected {
		t := plan.nodes[id]
		if t == nil {
			return nil, Stats{}, errors.New("unknown selected target: " + string(id))
		}
		for _, d := range t.Deps {
			if selected[d] {
				indeg[id]++
				rev[d] = append(rev[d], id)
			}
		}
	}
	type job struct{ id TargetID }
	jobs := make(chan job, len(selected))
	// ready queue: nodes with indeg 0
	ready := make([]TargetID, 0)
	for id, deg := range indeg {
		if deg == 0 {
			ready = append(ready, id)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })

	var wg sync.WaitGroup
	var running int64
	results := make([]Result, 0, len(selected))
	var resultsMu sync.Mutex
	var stats Stats
	stats.TotalTargets = int64(len(selected))

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			t := plan.nodes[j.id]
			start := time.Now()
			err := t.Action(DefaultContext{ctx}, *t)
			took := time.Since(start)
			atomic.AddInt64(&running, -1)
			resultsMu.Lock()
			results = append(results, Result{ID: j.id, Err: err, Took: took})
			if err != nil {
				stats.Failed++
			} else {
				stats.Succeeded++
			}
			resultsMu.Unlock()
			for _, dep := range rev[j.id] {
				if indeg[dep] > 0 {
					indeg[dep]--
				}
				if indeg[dep] == 0 {
					stats.Enqueued++
					jobs <- job{id: dep}
				}
			}
		}
	}
	wg.Add(e.workers)
	for i := 0; i < e.workers; i++ {
		go worker()
	}
	for _, id := range ready {
		stats.Enqueued++
		atomic.AddInt64(&running, 1)
		if cur := atomic.LoadInt64(&running); cur > stats.MaxParallel {
			stats.MaxParallel = cur
		}
		jobs <- job{id: id}
	}

	// Wait for completion
	for {
		time.Sleep(1 * time.Millisecond)
		if int(stats.Succeeded+stats.Failed) == len(selected) {
			break
		}
		if cur := atomic.LoadInt64(&running); cur > stats.MaxParallel {
			stats.MaxParallel = cur
		}
	}
	close(jobs)
	wg.Wait()
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results, stats, nil
}
