package concurrency

import (
	"runtime"
	"sync/atomic"
)

// MPMCQueue is a bounded multi-producer multi-consumer lock-free ring buffer
// based on Dmitry Vyukov's algorithm using per-slot sequence numbers.
type MPMCQueue[T any] struct {
	_pad0   [64]byte
	mask    uint64
	_pad1   [64]byte
	enqueue uint64
	_pad2   [64]byte
	dequeue uint64
	_pad3   [64]byte
	cells   []cell[T]
}

type cell[T any] struct {
	seq  uint64
	_pad [56]byte // cache line padding (approx)
	val  T
}

// NewMPMCQueue creates a new queue with the given capacity (must be power of two; rounded up if not).
func NewMPMCQueue[T any](capacity uint64) *MPMCQueue[T] {
	if capacity < 2 {
		capacity = 2
	}
	// round up to power of two
	capPow2 := uint64(1)
	for capPow2 < capacity {
		capPow2 <<= 1
	}
	q := &MPMCQueue[T]{
		mask:  capPow2 - 1,
		cells: make([]cell[T], capPow2),
	}
	for i := range q.cells {
		q.cells[i].seq = uint64(i)
	}
	return q
}

// Enqueue tries to push v; returns false if the queue is full.
func (q *MPMCQueue[T]) Enqueue(v T) bool {
	for {
		pos := atomic.LoadUint64(&q.enqueue)
		c := &q.cells[pos&q.mask]
		seq := atomic.LoadUint64(&c.seq)
		dif := int64(seq) - int64(pos)
		if dif == 0 {
			if atomic.CompareAndSwapUint64(&q.enqueue, pos, pos+1) {
				c.val = v
				atomic.StoreUint64(&c.seq, pos+1)
				return true
			}
		} else if dif < 0 {
			return false // full
		} else {
			runtime.Gosched()
		}
	}
}

// Dequeue tries to pop into out; returns false if the queue is empty.
func (q *MPMCQueue[T]) Dequeue(out *T) bool {
	for {
		pos := atomic.LoadUint64(&q.dequeue)
		c := &q.cells[pos&q.mask]
		seq := atomic.LoadUint64(&c.seq)
		dif := int64(seq) - int64(pos+1)
		if dif == 0 {
			if atomic.CompareAndSwapUint64(&q.dequeue, pos, pos+1) {
				*out = c.val
				atomic.StoreUint64(&c.seq, pos+q.mask+1)
				return true
			}
		} else if dif < 0 {
			return false // empty
		} else {
			runtime.Gosched()
		}
	}
}
