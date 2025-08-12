// Package stm implements a simple Software Transactional Memory runtime
// providing optimistic concurrency with versioned TVars and transactional
// read/write/commit with validation and automatic retry.
package stm

import (
	"errors"
	"math/rand"
	"sync/atomic"
	"time"
)

// Seed the default math/rand generator to avoid deterministic jitter across runs.
// The package-level default generator in math/rand is safe for concurrent use.
func init() {
	rand.Seed(time.Now().UnixNano())
}

// TVar is a transactional variable holding a value of type T with a version.
type TVar[T any] struct {
	ver uint64       // monotonically increasing version
	val atomic.Value // stores T
}

// NewTVar creates a new TVar with the given initial value.
func NewTVar[T any](v T) *TVar[T] {
	tv := &TVar[T]{}
	tv.val.Store(v)
	// Version must start as even (unlocked). Even numbers indicate no writer holds the lock.
	atomic.StoreUint64(&tv.ver, 0)
	return tv
}

// Txn represents an active transaction context.
type Txn[T any] struct {
	// readSet records TVar version and snapshot value at first read
	readSet map[*TVar[T]]readEntry[T]
	// writeSet records intended new values (overrides reads)
	writeSet map[*TVar[T]]T
}

type readEntry[T any] struct {
	ver uint64
	val T
}

// Begin starts a new transaction.
func Begin[T any]() *Txn[T] {
	return &Txn[T]{
		readSet:  make(map[*TVar[T]]readEntry[T]),
		writeSet: make(map[*TVar[T]]T),
	}
}

// Read reads a value from a TVar under the transaction.
func (tx *Txn[T]) Read(tv *TVar[T]) T {
	if v, ok := tx.writeSet[tv]; ok {
		return v
	}
	if e, ok := tx.readSet[tv]; ok {
		return e.val
	}
	// Read with validation: ensure version is even and unchanged after reading value
	for {
		ver1 := atomic.LoadUint64(&tv.ver)
		if ver1&1 == 1 { // writer holds lock
			time.Sleep(time.Microsecond)
			continue
		}
		vAny := tv.val.Load()
		ver2 := atomic.LoadUint64(&tv.ver)
		if ver1 == ver2 && (ver2&1) == 0 {
			val := vAny.(T)
			tx.readSet[tv] = readEntry[T]{ver: ver2, val: val}
			return val
		}
		// changed concurrently; retry
	}
}

// Write writes a value to a TVar under the transaction.
func (tx *Txn[T]) Write(tv *TVar[T], val T) {
	tx.writeSet[tv] = val
}

// ErrConflict indicates validation failed due to concurrent modification.
var ErrConflict = errors.New("stm: conflict")

// Commit validates and applies the transaction. On conflict it returns ErrConflict.
func (tx *Txn[T]) Commit() error {
	// Validate read set (excluding those we will write, which we validate during write lock)
	for tv, e := range tx.readSet {
		if _, willWrite := tx.writeSet[tv]; willWrite {
			continue
		}
		v := atomic.LoadUint64(&tv.ver)
		if v != e.ver || (v&1) == 1 { // changed or locked
			return ErrConflict
		}
	}
	// Apply writes
	for tv, newVal := range tx.writeSet {
		// expected even version from read set if present; else load current and ensure even
		exp := func() uint64 {
			if e, ok := tx.readSet[tv]; ok {
				return e.ver
			}
			return atomic.LoadUint64(&tv.ver)
		}()
		if exp&1 == 1 { // locked
			return ErrConflict
		}
		// Lock variable by moving to odd version
		if !atomic.CompareAndSwapUint64(&tv.ver, exp, exp+1) {
			return ErrConflict
		}
		// Write value while locked
		tv.val.Store(newVal)
		// Unlock by setting to next even version
		atomic.StoreUint64(&tv.ver, exp+2)
	}
	return nil
}

// Run executes f in a transactional loop with retry/backoff on conflict.
// maxRetries <= 0 means default retries.
func Run[T any](maxRetries int, f func(tx *Txn[T]) error) error {
	unlimited := maxRetries <= 0
	base := 50 * time.Microsecond
	maxSleep := 10 * time.Millisecond // cap sleep to avoid hot-looping under heavy contention
	attempt := func() error {
		tx := Begin[T]()
		if err := f(tx); err != nil {
			return err
		}
		return tx.Commit()
	}
	if unlimited {
		for i := 0; ; i++ {
			if err := attempt(); err == nil {
				return nil
			}
			// Small exponential backoff with microsecond jitter up to ~1ms
			step := i
			if step > 4 {
				step = 4
			}
			sleep := base << step
			if sleep > maxSleep {
				sleep = maxSleep
			}
			jitter := time.Duration(rand.Intn(200)) * time.Microsecond
			d := sleep + jitter
			if d > maxSleep {
				d = maxSleep
			}
			time.Sleep(d)
		}
	}
	for i := 0; i < maxRetries; i++ {
		if err := attempt(); err == nil {
			return nil
		}
		step := i
		if step > 4 {
			step = 4
		}
		sleep := base << step
		if sleep > maxSleep {
			sleep = maxSleep
		}
		jitter := time.Duration(rand.Intn(200)) * time.Microsecond
		d := sleep + jitter
		if d > maxSleep {
			d = maxSleep
		}
		time.Sleep(d)
	}
	return ErrConflict
}
