package collections

// Deque is a generic double-ended queue implemented as a ring buffer.
// Zero value is ready to use.
type Deque[T any] struct {
	buf        []T
	head, tail int // head points to the first element, tail to the position to write next at back
	size       int
}

// Len returns the number of elements.
func (d *Deque[T]) Len() int { return d.size }

// IsEmpty reports whether the deque has no elements.
func (d *Deque[T]) IsEmpty() bool { return d.size == 0 }

// Clear removes all elements while keeping capacity.
func (d *Deque[T]) Clear() {
	// Do not zero elements to keep it fast; allow GC by zeroing only visible range if needed later.
	d.head, d.tail, d.size = 0, 0, 0
}

// PushBack appends v at the back.
func (d *Deque[T]) PushBack(v T) {
	d.growIfFull()
	if len(d.buf) == 0 {
		d.buf = make([]T, 8)
	}
	d.buf[d.tail] = v
	d.tail = (d.tail + 1) & (len(d.buf) - 1)
	d.size++
}

// PushFront inserts v at the front.
func (d *Deque[T]) PushFront(v T) {
	d.growIfFull()
	if len(d.buf) == 0 {
		d.buf = make([]T, 8)
	}
	d.head = (d.head - 1) & (len(d.buf) - 1)
	d.buf[d.head] = v
	d.size++
}

// PopBack removes and returns the back element. ok=false when empty.
func (d *Deque[T]) PopBack() (out T, ok bool) {
	if d.size == 0 {
		var z T
		return z, false
	}
	d.tail = (d.tail - 1) & (len(d.buf) - 1)
	out = d.buf[d.tail]
	var z T
	d.buf[d.tail] = z
	d.size--
	return out, true
}

// PopFront removes and returns the front element. ok=false when empty.
func (d *Deque[T]) PopFront() (out T, ok bool) {
	if d.size == 0 {
		var z T
		return z, false
	}
	out = d.buf[d.head]
	var z T
	d.buf[d.head] = z
	d.head = (d.head + 1) & (len(d.buf) - 1)
	d.size--
	return out, true
}

// Front returns the front element without removing it. ok=false when empty.
func (d *Deque[T]) Front() (T, bool) {
	if d.size == 0 {
		var z T
		return z, false
	}
	return d.buf[d.head], true
}

// Back returns the back element without removing it. ok=false when empty.
func (d *Deque[T]) Back() (T, bool) {
	if d.size == 0 {
		var z T
		return z, false
	}
	idx := (d.tail - 1) & (len(d.buf) - 1)
	return d.buf[idx], true
}

// growIfFull ensures there's room for at least one more element.
func (d *Deque[T]) growIfFull() {
	if d.size == 0 && len(d.buf) == 0 {
		return
	}
	if d.size < len(d.buf) {
		return
	}
	newCap := 8
	if len(d.buf) > 0 {
		newCap = len(d.buf) << 1
	}
	nb := make([]T, newCap)
	// copy existing elements in order
	if d.size > 0 {
		if d.head < d.tail {
			copy(nb, d.buf[d.head:d.tail])
		} else {
			n := copy(nb, d.buf[d.head:])
			copy(nb[n:], d.buf[:d.tail])
		}
	}
	d.buf = nb
	d.head = 0
	d.tail = d.size
}

// ForEach iterates over elements from front to back.
func (d *Deque[T]) ForEach(fn func(T)) {
	if d.size == 0 {
		return
	}
	if d.head < d.tail {
		for i := d.head; i < d.tail; i++ {
			fn(d.buf[i])
		}
		return
	}
	for i := d.head; i < len(d.buf); i++ {
		fn(d.buf[i])
	}
	for i := 0; i < d.tail; i++ {
		fn(d.buf[i])
	}
}
