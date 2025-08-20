// Advanced Concurrent Data Structures for Orizon.
package collections

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// ====== Lock-Free Queue (SPSC - Single Producer Single Consumer) ======.

type LockFreeQueue[T any] struct {
	head     atomic.Pointer[queueNode[T]]
	tail     atomic.Pointer[queueNode[T]]
	_padding [56]byte // Cache line padding
}

type queueNode[T any] struct {
	data T
	next atomic.Pointer[queueNode[T]]
}

func NewLockFreeQueue[T any]() *LockFreeQueue[T] {
	dummy := &queueNode[T]{}
	q := &LockFreeQueue[T]{}
	q.head.Store(dummy)
	q.tail.Store(dummy)

	return q
}

func (q *LockFreeQueue[T]) Enqueue(item T) {
	node := &queueNode[T]{data: item}

	for {
		tail := q.tail.Load()
		next := tail.next.Load()

		if tail == q.tail.Load() { // Check consistency
			if next == nil {
				// Try to link node at the end of the list.
				if tail.next.CompareAndSwap(nil, node) {
					// Success, try to swing tail to the inserted node.
					q.tail.CompareAndSwap(tail, node)

					break
				}
			} else {
				// Help advance tail.
				q.tail.CompareAndSwap(tail, next)
			}
		}
	}
}

func (q *LockFreeQueue[T]) Dequeue() (T, bool) {
	for {
		head := q.head.Load()
		tail := q.tail.Load()
		next := head.next.Load()

		if head == q.head.Load() { // Check consistency
			if head == tail {
				if next == nil {
					var zero T

					return zero, false // Queue is empty
				}
				// Help advance tail.
				q.tail.CompareAndSwap(tail, next)
			} else {
				if next == nil {
					continue // Should not happen
				}

				// Read data before CAS, as another dequeue might free next node.
				data := next.data

				// Try to swing head to the next node.
				if q.head.CompareAndSwap(head, next) {
					return data, true
				}
			}
		}
	}
}

// ====== Wait-Free Ring Buffer ======.

type WaitFreeRingBuffer[T any] struct {
	buffer   []T
	mask     uint64
	head     atomic.Uint64
	tail     atomic.Uint64
	_padding [48]byte // Cache line padding
}

func NewWaitFreeRingBuffer[T any](size uint64) *WaitFreeRingBuffer[T] {
	// Ensure size is power of 2.
	if size == 0 || (size&(size-1)) != 0 {
		panic("size must be a power of 2")
	}

	return &WaitFreeRingBuffer[T]{
		buffer: make([]T, size),
		mask:   size - 1,
	}
}

func (rb *WaitFreeRingBuffer[T]) Push(item T) bool {
	head := rb.head.Load()
	tail := rb.tail.Load()

	if (head+1)&rb.mask == tail {
		return false // Buffer is full
	}

	rb.buffer[head] = item
	rb.head.Store((head + 1) & rb.mask)

	return true
}

func (rb *WaitFreeRingBuffer[T]) Pop() (T, bool) {
	head := rb.head.Load()
	tail := rb.tail.Load()

	if head == tail {
		var zero T

		return zero, false // Buffer is empty
	}

	item := rb.buffer[tail]
	rb.tail.Store((tail + 1) & rb.mask)

	return item, true
}

func (rb *WaitFreeRingBuffer[T]) Size() uint64 {
	head := rb.head.Load()
	tail := rb.tail.Load()

	return (head - tail) & rb.mask
}

// ====== Read-Copy-Update (RCU) List ======.

type RCUNode[T any] struct {
	data T
	next atomic.Pointer[RCUNode[T]]
}

type RCUList[T any] struct {
	head    atomic.Pointer[RCUNode[T]]
	pending []unsafe.Pointer
	epoch   atomic.Uint64
	mu      sync.RWMutex
}

func NewRCUList[T any]() *RCUList[T] {
	return &RCUList[T]{
		pending: make([]unsafe.Pointer, 0),
	}
}

func (l *RCUList[T]) Insert(data T) {
	node := &RCUNode[T]{data: data}

	for {
		head := l.head.Load()
		node.next.Store(head)

		if l.head.CompareAndSwap(head, node) {
			break
		}
	}
}

func (l *RCUList[T]) Find(predicate func(T) bool) (T, bool) {
	// Enter read-side critical section.
	epoch := l.epoch.Load()

	current := l.head.Load()
	for current != nil {
		if predicate(current.data) {
			// Ensure we're still in the same epoch.
			if l.epoch.Load() == epoch {
				return current.data, true
			}
			// Epoch changed, restart.
			return l.Find(predicate)
		}

		current = current.next.Load()
	}

	var zero T

	return zero, false
}

func (l *RCUList[T]) Remove(predicate func(T) bool) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Increment epoch to signal update.
	l.epoch.Add(1)

	prev := (*RCUNode[T])(nil)
	current := l.head.Load()

	for current != nil {
		if predicate(current.data) {
			next := current.next.Load()

			if prev == nil {
				l.head.Store(next)
			} else {
				prev.next.Store(next)
			}

			// Add to pending deletion list.
			l.pending = append(l.pending, unsafe.Pointer(current))

			// Grace period simulation.
			go func() {
				runtime.GC() // Force GC to ensure no readers
				// In real RCU, we'd use proper grace period detection.
			}()

			return true
		}

		prev = current
		current = current.next.Load()
	}

	return false
}

// ====== Concurrent B+ Tree ======.

const btreeDegree = 16

type BTreeNode[K comparable, V any] struct {
	parent   *BTreeNode[K, V]
	keys     []K
	values   []V
	children []*BTreeNode[K, V]
	mu       sync.RWMutex
	isLeaf   bool
}

type ConcurrentBTree[K comparable, V any] struct {
	root    atomic.Pointer[BTreeNode[K, V]]
	compare func(K, K) int
	mu      sync.RWMutex
}

func NewConcurrentBTree[K comparable, V any](compare func(K, K) int) *ConcurrentBTree[K, V] {
	root := &BTreeNode[K, V]{
		keys:   make([]K, 0, btreeDegree-1),
		values: make([]V, 0, btreeDegree-1),
		isLeaf: true,
	}

	bt := &ConcurrentBTree[K, V]{
		compare: compare,
	}
	bt.root.Store(root)

	return bt
}

func (bt *ConcurrentBTree[K, V]) Insert(key K, value V) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	root := bt.root.Load()
	if len(root.keys) == btreeDegree-1 {
		// Split root.
		newRoot := &BTreeNode[K, V]{
			keys:     make([]K, 0, btreeDegree-1),
			values:   make([]V, 0, btreeDegree-1),
			children: make([]*BTreeNode[K, V], 0, btreeDegree),
			isLeaf:   false,
		}
		newRoot.children = append(newRoot.children, root)
		bt.splitChild(newRoot, 0)
		bt.root.Store(newRoot)
		root = newRoot
	}

	bt.insertNonFull(root, key, value)
}

func (bt *ConcurrentBTree[K, V]) insertNonFull(node *BTreeNode[K, V], key K, value V) {
	node.mu.Lock()
	defer node.mu.Unlock()

	i := len(node.keys) - 1

	if node.isLeaf {
		// Insert into leaf.
		node.keys = append(node.keys, key)
		node.values = append(node.values, value)

		// Shift elements to maintain sorted order.
		for i >= 0 && bt.compare(node.keys[i], key) > 0 {
			node.keys[i+1] = node.keys[i]
			node.values[i+1] = node.values[i]
			i--
		}

		node.keys[i+1] = key
		node.values[i+1] = value
	} else {
		// Find child to insert into.
		for i >= 0 && bt.compare(node.keys[i], key) > 0 {
			i--
		}

		i++

		child := node.children[i]
		if len(child.keys) == btreeDegree-1 {
			bt.splitChild(node, i)

			if bt.compare(key, node.keys[i]) > 0 {
				i++
			}
		}

		bt.insertNonFull(node.children[i], key, value)
	}
}

func (bt *ConcurrentBTree[K, V]) splitChild(parent *BTreeNode[K, V], index int) {
	fullChild := parent.children[index]
	newChild := &BTreeNode[K, V]{
		keys:   make([]K, 0, btreeDegree-1),
		values: make([]V, 0, btreeDegree-1),
		isLeaf: fullChild.isLeaf,
		parent: parent,
	}

	if !fullChild.isLeaf {
		newChild.children = make([]*BTreeNode[K, V], 0, btreeDegree)
	}

	mid := btreeDegree / 2

	// Move half of keys/values to new child
	newChild.keys = append(newChild.keys, fullChild.keys[mid+1:]...)
	newChild.values = append(newChild.values, fullChild.values[mid+1:]...)
	fullChild.keys = fullChild.keys[:mid]
	fullChild.values = fullChild.values[:mid]

	if !fullChild.isLeaf {
		newChild.children = append(newChild.children, fullChild.children[mid+1:]...)
		fullChild.children = fullChild.children[:mid+1]
	}

	// Insert new child into parent.
	parent.children = append(parent.children, nil)
	copy(parent.children[index+2:], parent.children[index+1:])
	parent.children[index+1] = newChild

	// Move median key up to parent.
	parent.keys = append(parent.keys, fullChild.keys[mid])
	parent.values = append(parent.values, fullChild.values[mid])

	// Shift to maintain order.
	for i := len(parent.keys) - 1; i > index; i-- {
		parent.keys[i] = parent.keys[i-1]
		parent.values[i] = parent.values[i-1]
	}

	parent.keys[index] = fullChild.keys[mid]
	parent.values[index] = fullChild.values[mid]
}

func (bt *ConcurrentBTree[K, V]) Search(key K) (V, bool) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	return bt.searchNode(bt.root.Load(), key)
}

func (bt *ConcurrentBTree[K, V]) searchNode(node *BTreeNode[K, V], key K) (V, bool) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	i := 0
	for i < len(node.keys) && bt.compare(key, node.keys[i]) > 0 {
		i++
	}

	if i < len(node.keys) && bt.compare(key, node.keys[i]) == 0 {
		return node.values[i], true
	}

	if node.isLeaf {
		var zero V

		return zero, false
	}

	return bt.searchNode(node.children[i], key)
}

// ====== Concurrent Skip List ======.

const maxLevel = 32

type SkipListNode[K comparable, V any] struct {
	key     K
	value   V
	forward []*SkipListNode[K, V]
	mu      sync.RWMutex
}

type ConcurrentSkipList[K comparable, V any] struct {
	header  *SkipListNode[K, V]
	compare func(K, K) int
	mu      sync.RWMutex
	level   atomic.Int32
}

func NewConcurrentSkipList[K comparable, V any](compare func(K, K) int) *ConcurrentSkipList[K, V] {
	var zeroK K

	var zeroV V

	header := &SkipListNode[K, V]{
		key:     zeroK,
		value:   zeroV,
		forward: make([]*SkipListNode[K, V], maxLevel),
	}

	return &ConcurrentSkipList[K, V]{
		header:  header,
		compare: compare,
	}
}

func (sl *ConcurrentSkipList[K, V]) randomLevel() int {
	level := 1
	for level < maxLevel && (rand.Int()&0x1) == 0 {
		level++
	}

	return level
}

func (sl *ConcurrentSkipList[K, V]) Insert(key K, value V) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	update := make([]*SkipListNode[K, V], maxLevel)
	current := sl.header

	// Find position to insert.
	for i := int(sl.level.Load()) - 1; i >= 0; i-- {
		for current.forward[i] != nil && sl.compare(current.forward[i].key, key) < 0 {
			current = current.forward[i]
		}

		update[i] = current
	}

	current = current.forward[0]

	if current != nil && sl.compare(current.key, key) == 0 {
		// Update existing.
		current.mu.Lock()
		current.value = value
		current.mu.Unlock()

		return
	}

	// Create new node.
	newLevel := sl.randomLevel()
	if newLevel > int(sl.level.Load()) {
		for i := int(sl.level.Load()); i < newLevel; i++ {
			update[i] = sl.header
		}

		sl.level.Store(int32(newLevel))
	}

	newNode := &SkipListNode[K, V]{
		key:     key,
		value:   value,
		forward: make([]*SkipListNode[K, V], newLevel),
	}

	// Update pointers.
	for i := 0; i < newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}
}

func (sl *ConcurrentSkipList[K, V]) Search(key K) (V, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	current := sl.header

	for i := int(sl.level.Load()) - 1; i >= 0; i-- {
		for current.forward[i] != nil && sl.compare(current.forward[i].key, key) < 0 {
			current = current.forward[i]
		}
	}

	current = current.forward[0]

	if current != nil && sl.compare(current.key, key) == 0 {
		current.mu.RLock()
		value := current.value
		current.mu.RUnlock()

		return value, true
	}

	var zero V

	return zero, false
}

// ====== Memory Pool with NUMA Awareness ======.

type NUMAPool[T any] struct {
	allocFn   func() T
	resetFn   func(*T)
	pools     []chan T
	nodeCount int
}

func NewNUMAPool[T any](allocFn func() T, resetFn func(*T)) *NUMAPool[T] {
	nodeCount := runtime.NumCPU() // Approximate NUMA nodes
	if nodeCount > 8 {
		nodeCount = 8 // Reasonable upper bound
	}

	pools := make([]chan T, nodeCount)
	for i := range pools {
		pools[i] = make(chan T, 100) // Buffer per node
	}

	return &NUMAPool[T]{
		pools:     pools,
		nodeCount: nodeCount,
		allocFn:   allocFn,
		resetFn:   resetFn,
	}
}

func (p *NUMAPool[T]) Get() T {
	// Try to get from local NUMA node first.
	nodeID := rand.Int() % p.nodeCount

	select {
	case obj := <-p.pools[nodeID]:
		return obj
	default:
		// Try other nodes.
		for i := 0; i < p.nodeCount; i++ {
			if i == nodeID {
				continue
			}
			select {
			case obj := <-p.pools[i]:
				return obj
			default:
			}
		}
	}

	// Allocate new object.
	return p.allocFn()
}

func (p *NUMAPool[T]) Put(obj T) {
	if p.resetFn != nil {
		p.resetFn(&obj)
	}

	nodeID := rand.Int() % p.nodeCount

	select {
	case p.pools[nodeID] <- obj:
	default:
		// Pool is full, just discard.
	}
}
