package collections

// PriorityQueue is a generic min-heap priority queue.
// Use Less to define ordering.
type PriorityQueue[T any] struct {
	data []T
	less func(a, b T) bool // return true if a has higher priority than b (min-heap: a < b)
}

func NewPriorityQueue[T any](less func(a, b T) bool) *PriorityQueue[T] {
	if less == nil {
		panic("less function required")
	}
	return &PriorityQueue[T]{less: less}
}

func (pq *PriorityQueue[T]) Len() int      { return len(pq.data) }
func (pq *PriorityQueue[T]) IsEmpty() bool { return len(pq.data) == 0 }

func (pq *PriorityQueue[T]) Push(x T) {
	pq.data = append(pq.data, x)
	pq.up(len(pq.data) - 1)
}

func (pq *PriorityQueue[T]) Peek() (T, bool) {
	if len(pq.data) == 0 {
		var z T
		return z, false
	}
	return pq.data[0], true
}

func (pq *PriorityQueue[T]) Pop() (T, bool) {
	if len(pq.data) == 0 {
		var z T
		return z, false
	}
	top := pq.data[0]
	last := pq.data[len(pq.data)-1]
	pq.data = pq.data[:len(pq.data)-1]
	if len(pq.data) > 0 {
		pq.data[0] = last
		pq.down(0)
	}
	return top, true
}

func (pq *PriorityQueue[T]) up(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if !pq.less(pq.data[i], pq.data[p]) {
			break
		}
		pq.data[i], pq.data[p] = pq.data[p], pq.data[i]
		i = p
	}
}

func (pq *PriorityQueue[T]) down(i int) {
	n := len(pq.data)
	for {
		l := 2*i + 1
		r := l + 1
		smallest := i
		if l < n && pq.less(pq.data[l], pq.data[smallest]) {
			smallest = l
		}
		if r < n && pq.less(pq.data[r], pq.data[smallest]) {
			smallest = r
		}
		if smallest == i {
			return
		}
		pq.data[i], pq.data[smallest] = pq.data[smallest], pq.data[i]
		i = smallest
	}
}
