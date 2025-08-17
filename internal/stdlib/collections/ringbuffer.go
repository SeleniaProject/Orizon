package collections

// RingBuffer is a fixed-capacity ring that overwrites the oldest entries when full.
// Zero value is not ready; use NewRingBuffer.
type RingBuffer[T any] struct {
	buf  []T
	head int // index of next element to read
	size int // number of elements stored (<= cap)
}

// NewRingBuffer creates a ring with capacity cap (>=1).
func NewRingBuffer[T any](cap int) *RingBuffer[T] {
	if cap < 1 {
		cap = 1
	}
	return &RingBuffer[T]{buf: make([]T, cap)}
}

// Cap returns the capacity of the ring.
func (r *RingBuffer[T]) Cap() int { return len(r.buf) }

// Len returns the number of stored elements.
func (r *RingBuffer[T]) Len() int { return r.size }

// IsEmpty reports whether the buffer is empty.
func (r *RingBuffer[T]) IsEmpty() bool { return r.size == 0 }

// Push appends v, overwriting the oldest element when full.
func (r *RingBuffer[T]) Push(v T) {
	if r.size < len(r.buf) {
		idx := (r.head + r.size) % len(r.buf)
		r.buf[idx] = v
		r.size++
		return
	}
	// overwrite oldest at head and advance head
	r.buf[r.head] = v
	r.head = (r.head + 1) % len(r.buf)
}

// Pop removes and returns the oldest element. ok=false when empty.
func (r *RingBuffer[T]) Pop() (out T, ok bool) {
	if r.size == 0 {
		var z T
		return z, false
	}
	out = r.buf[r.head]
	var z T
	r.buf[r.head] = z
	r.head = (r.head + 1) % len(r.buf)
	r.size--
	return out, true
}

// Peek returns the oldest element without removing it. ok=false when empty.
func (r *RingBuffer[T]) Peek() (T, bool) {
	if r.size == 0 {
		var z T
		return z, false
	}
	return r.buf[r.head], true
}
