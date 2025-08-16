package concurrency

import (
	"sync"
)

// RaceKind classifies a detected race.
type RaceKind string

const (
	RaceWriteWrite RaceKind = "write-write"
	RaceWriteRead  RaceKind = "write-read"
	RaceReadWrite  RaceKind = "read-write"
)

// Race describes a single data race between goroutines on a variable address.
type Race struct {
	VarAddr uintptr
	Thread1 int64
	Thread2 int64
	Kind    RaceKind
}

// accessKind is a bitset for read/write kinds.
type accessKind uint8

const (
	accessRead  accessKind = 1 << 0
	accessWrite accessKind = 1 << 1
)

// raceVar stores per-variable access information and candidate lockset.
type raceVar struct {
	seenByThread map[int64]accessKind // thread -> last observed access kind(s)
	lockset      map[int64]struct{}   // intersection of locks held across all accesses
	reportedEdge map[string]struct{}  // to deduplicate reports per (t1,t2,kind)
}

// RaceDetector implements a lightweight dynamic race detector using a lockset
// analysis similar to Eraser. It is designed for use in tests and requires
// cooperative instrumentation at read/write and lock/unlock points.
type RaceDetector struct {
	mu        sync.Mutex
	locksHeld map[int64]map[int64]struct{} // thread -> set of lock IDs held
	vars      map[uintptr]*raceVar         // address -> state
	races     []Race                       // recorded races
}

// NewRaceDetector creates a new detector instance.
func NewRaceDetector() *RaceDetector {
	return &RaceDetector{
		locksHeld: make(map[int64]map[int64]struct{}),
		vars:      make(map[uintptr]*raceVar),
	}
}

// OnLock records that thread gid acquired a lock with logical ID lid.
func (d *RaceDetector) OnLock(gid, lid int64) {
	d.mu.Lock()
	set := d.locksHeld[gid]
	if set == nil {
		set = make(map[int64]struct{})
		d.locksHeld[gid] = set
	}
	set[lid] = struct{}{}
	d.mu.Unlock()
}

// OnUnlock records that thread gid released a lock with logical ID lid.
func (d *RaceDetector) OnUnlock(gid, lid int64) {
	d.mu.Lock()
	if set := d.locksHeld[gid]; set != nil {
		delete(set, lid)
	}
	d.mu.Unlock()
}

// Read marks a read access to the variable at address addr by thread gid.
func (d *RaceDetector) Read(gid int64, addr uintptr) {
	d.onAccess(gid, addr, accessRead)
}

// Write marks a write access to the variable at address addr by thread gid.
func (d *RaceDetector) Write(gid int64, addr uintptr) {
	d.onAccess(gid, addr, accessWrite)
}

// Races returns a snapshot of all detected races.
func (d *RaceDetector) Races() []Race {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]Race, len(d.races))
	copy(out, d.races)
	return out
}

// HasRace reports whether any race has been detected.
func (d *RaceDetector) HasRace() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.races) > 0
}

func (d *RaceDetector) onAccess(gid int64, addr uintptr, kind accessKind) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get current thread lockset snapshot
	curLocks := d.copyLocks(d.locksHeld[gid])

	st := d.vars[addr]
	if st == nil {
		st = &raceVar{
			seenByThread: make(map[int64]accessKind),
			reportedEdge: make(map[string]struct{}),
		}
		// Initialize candidate lockset to locks currently held
		st.lockset = curLocks
		d.vars[addr] = st
	}

	// Check for conflicting prior accesses from other threads without common locks
	for other, okind := range st.seenByThread {
		if other == gid {
			continue
		}
		if conflicts(kind, okind) {
			if isEmpty(intersect(st.lockset, curLocks)) {
				// Report race once per (t1,t2,kind) edge
				t1, t2 := orderPair(gid, other)
				var rk RaceKind
				if kind&accessWrite != 0 && okind&accessWrite != 0 {
					rk = RaceWriteWrite
				} else if kind&accessWrite != 0 {
					rk = RaceWriteRead
				} else {
					rk = RaceReadWrite
				}
				key := edgeKey(addr, t1, t2, rk)
				if _, dup := st.reportedEdge[key]; !dup {
					st.reportedEdge[key] = struct{}{}
					d.races = append(d.races, Race{VarAddr: addr, Thread1: t1, Thread2: t2, Kind: rk})
				}
			}
		}
	}

	// Update candidate lockset by intersecting with current locks
	st.lockset = intersect(st.lockset, curLocks)
	// Record this access kind for the thread
	st.seenByThread[gid] = st.seenByThread[gid] | kind
}

func conflicts(a, b accessKind) bool {
	// A race requires at least one write across different threads
	return (a&accessWrite) != 0 || (b&accessWrite) != 0
}

func (d *RaceDetector) copyLocks(src map[int64]struct{}) map[int64]struct{} {
	if src == nil {
		return make(map[int64]struct{})
	}
	out := make(map[int64]struct{}, len(src))
	for k := range src {
		out[k] = struct{}{}
	}
	return out
}

func intersect(a, b map[int64]struct{}) map[int64]struct{} {
	// Choose smaller to iterate
	if len(a) > len(b) {
		return intersect(b, a)
	}
	out := make(map[int64]struct{})
	for k := range a {
		if _, ok := b[k]; ok {
			out[k] = struct{}{}
		}
	}
	return out
}

func isEmpty(m map[int64]struct{}) bool { return len(m) == 0 }

func orderPair(a, b int64) (int64, int64) {
	if a < b {
		return a, b
	}
	return b, a
}

func edgeKey(addr uintptr, t1, t2 int64, kind RaceKind) string {
	// Deterministic key for deduplication
	return string(kind) + ":" + itoa64(int64(addr)) + ":" + itoa64(t1) + "/" + itoa64(t2)
}

// itoa64 is a minimal int64 to base-10 string formatter to avoid allocations from fmt.
func itoa64(n int64) string {
	// Buffer sized for int64
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TrackedMutex is a mutex that reports lock/unlock to a RaceDetector.
type TrackedMutex struct {
	mu  sync.Mutex
	id  int64
	det *RaceDetector
}

// NewTrackedMutex creates a mutex with a stable logical ID associated with the detector.
func NewTrackedMutex(id int64, det *RaceDetector) *TrackedMutex {
	return &TrackedMutex{id: id, det: det}
}

// Lock records the attempt by gid and acquires the underlying mutex.
func (m *TrackedMutex) Lock(gid int64) {
	if m.det != nil {
		m.det.OnLock(gid, m.id)
	}
	m.mu.Lock()
}

// Unlock releases the mutex and updates the detector state.
func (m *TrackedMutex) Unlock(gid int64) {
	m.mu.Unlock()
	if m.det != nil {
		m.det.OnUnlock(gid, m.id)
	}
}
