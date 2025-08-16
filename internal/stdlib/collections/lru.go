package collections

// LRU is a simple fixed-capacity least-recently-used cache.
// Zero value is not ready; use NewLRU.
type LRU[K comparable, V any] struct {
	cap int
	ll  *list[K, V]
	byK map[K]*node[K, V]
}

// NewLRU creates an LRU with given positive capacity.
func NewLRU[K comparable, V any](capacity int) *LRU[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &LRU[K, V]{cap: capacity, ll: newList[K, V](), byK: make(map[K]*node[K, V], capacity)}
}

// Get returns value and ok, and marks key as most recently used on hit.
func (l *LRU[K, V]) Get(k K) (V, bool) {
	if ent, ok := l.byK[k]; ok {
		l.ll.moveToFront(ent)
		return ent.v, true
	}
	var z V
	return z, false
}

// Put inserts or updates a value and marks it as most recent.
func (l *LRU[K, V]) Put(k K, v V) {
	if ent, ok := l.byK[k]; ok {
		ent.v = v
		l.ll.moveToFront(ent)
		return
	}
	ent := &node[K, V]{k: k, v: v}
	l.byK[k] = ent
	l.ll.pushFront(ent)
	if l.ll.len > l.cap {
		tail := l.ll.back()
		if tail != nil {
			delete(l.byK, tail.k)
			l.ll.remove(tail)
		}
	}
}

// Len returns current number of entries.
func (l *LRU[K, V]) Len() int { return l.ll.len }

// list and node (minimal intrusive doubly-linked list)
type node[K comparable, V any] struct {
	prev, next *node[K, V]
	k          K
	v          V
}

type list[K comparable, V any] struct {
	head, tail *node[K, V]
	len        int
}

func newList[K comparable, V any]() *list[K, V] { return &list[K, V]{} }

func (l *list[K, V]) pushFront(n *node[K, V]) {
	n.prev = nil
	n.next = l.head
	if l.head != nil {
		l.head.prev = n
	}
	l.head = n
	if l.tail == nil {
		l.tail = n
	}
	l.len++
}

func (l *list[K, V]) moveToFront(n *node[K, V]) {
	if l.head == n {
		return
	}
	l.remove(n)
	l.pushFront(n)
}

func (l *list[K, V]) back() *node[K, V] { return l.tail }

func (l *list[K, V]) remove(n *node[K, V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		l.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		l.tail = n.prev
	}
	n.prev, n.next = nil, nil
	l.len--
}
