package concurrency

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// DeadlockDetector observes lock waits/holds and detects wait-for cycles.
type DeadlockDetector struct {
	mu     sync.Mutex
	holds  map[int64]map[int64]struct{} // goroutine -> set of locks held
	waits  map[int64]int64              // goroutine -> lock waiting for
	locked atomic.Uint32
}

// NewDeadlockDetector creates a new detector.
func NewDeadlockDetector() *DeadlockDetector {
	return &DeadlockDetector{holds: make(map[int64]map[int64]struct{}), waits: make(map[int64]int64)}
}

// OnLockAttempt records that goroutine g is waiting for lock l.
func (d *DeadlockDetector) OnLockAttempt(g, l int64) {
	d.mu.Lock()
	d.waits[g] = l
	d.mu.Unlock()
}

// OnLockAcquired records that g acquired l.
func (d *DeadlockDetector) OnLockAcquired(g, l int64) {
	d.mu.Lock()
	delete(d.waits, g)
	set := d.holds[g]
	if set == nil {
		set = make(map[int64]struct{})
		d.holds[g] = set
	}
	set[l] = struct{}{}
	d.mu.Unlock()
}

// OnUnlock records that g released l.
func (d *DeadlockDetector) OnUnlock(g, l int64) {
	d.mu.Lock()
	if set := d.holds[g]; set != nil {
		delete(set, l)
	}
	d.mu.Unlock()
}

// Check detects a cycle using a simple DFS over wait-for graph.
func (d *DeadlockDetector) Check() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Build wait-for graph: g -> h if g waits for lock l held by h
	waitFor := make(map[int64][]int64)
	// Build reverse index: lock -> owners
	owners := make(map[int64][]int64)
	for g, set := range d.holds {
		for l := range set {
			owners[l] = append(owners[l], g)
		}
	}
	for g, l := range d.waits {
		for _, h := range owners[l] {
			waitFor[g] = append(waitFor[g], h)
		}
	}
	// DFS cycle detection
	const visiting = 1
	const visited = 2
	state := make(map[int64]int)
	var dfs func(int64) bool
	dfs = func(u int64) bool {
		state[u] = visiting
		for _, v := range waitFor[u] {
			if state[v] == visiting {
				return true
			}
			if state[v] == 0 && dfs(v) {
				return true
			}
		}
		state[u] = visited
		return false
	}
	for g := range waitFor {
		if state[g] == 0 && dfs(g) {
			return true
		}
	}
	return false
}

// MonitoredMutex wraps sync.Mutex and reports to the detector.
type MonitoredMutex struct {
	mu sync.Mutex
	id int64
	d  *DeadlockDetector
}

// NewMonitoredMutex creates a monitored mutex with unique id.
func NewMonitoredMutex(id int64, d *DeadlockDetector) *MonitoredMutex {
	return &MonitoredMutex{id: id, d: d}
}

// Lock reports attempts and acquisitions.
func (m *MonitoredMutex) Lock(gid int64) {
	if m.d != nil {
		m.d.OnLockAttempt(gid, m.id)
	}
	m.mu.Lock()
	if m.d != nil {
		m.d.OnLockAcquired(gid, m.id)
	}
}

// Unlock reports releases.
func (m *MonitoredMutex) Unlock(gid int64) {
	m.mu.Unlock()
	if m.d != nil {
		m.d.OnUnlock(gid, m.id)
	}
}

// WaitUntilDeadlock blocks until a deadlock is detected or timeout.
func (d *DeadlockDetector) WaitUntilDeadlock(timeout time.Duration, poll time.Duration) error {
	if poll <= 0 {
		poll = 5 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if d.Check() {
			return nil
		}
		time.Sleep(poll)
	}
	return errors.New("no deadlock detected before timeout")
}
