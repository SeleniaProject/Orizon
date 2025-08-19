// Enhanced Collections Module for Orizon Standard Library
// Provides high-performance, memory-safe collection types

package collections

import (
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

// Type aliases for clarity
type usize = uint64

// ====== Vector Implementation ======

// Vector is a generic growable array type
type Vector[T any] struct {
	data []T
	mu   sync.RWMutex
}

// NewVector creates a new vector with optional initial capacity
func NewVector[T any](capacity ...int) *Vector[T] {
	cap := 0
	if len(capacity) > 0 {
		cap = capacity[0]
	}
	return &Vector[T]{
		data: make([]T, 0, cap),
	}
}

// Push adds an element to the end of the vector
func (v *Vector[T]) Push(elem T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data = append(v.data, elem)
}

// Pop removes and returns the last element
func (v *Vector[T]) Pop() (T, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()

	var zero T
	if len(v.data) == 0 {
		return zero, false
	}

	elem := v.data[len(v.data)-1]
	v.data = v.data[:len(v.data)-1]
	return elem, true
}

// Get returns the element at the given index
func (v *Vector[T]) Get(index int) (T, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var zero T
	if index < 0 || index >= len(v.data) {
		return zero, false
	}
	return v.data[index], true
}

// Set sets the element at the given index
func (v *Vector[T]) Set(index int, elem T) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if index < 0 || index >= len(v.data) {
		return false
	}
	v.data[index] = elem
	return true
}

// Len returns the number of elements
func (v *Vector[T]) Len() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.data)
}

// Capacity returns the current capacity
func (v *Vector[T]) Capacity() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return cap(v.data)
}

// Reserve ensures minimum capacity
func (v *Vector[T]) Reserve(additional int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	newCap := len(v.data) + additional
	if cap(v.data) < newCap {
		newData := make([]T, len(v.data), newCap)
		copy(newData, v.data)
		v.data = newData
	}
}

// Iter returns a channel for iteration
func (v *Vector[T]) Iter() <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		v.mu.RLock()
		defer v.mu.RUnlock()
		for _, elem := range v.data {
			ch <- elem
		}
	}()
	return ch
}

// ====== HashMap Implementation ======

type HashMapEntry[K comparable, V any] struct {
	key   K
	value V
	hash  uint64
}

// HashMap is a thread-safe hash map with Robin Hood hashing
type HashMap[K comparable, V any] struct {
	buckets  []HashMapEntry[K, V]
	len      uint64
	capacity uint64
	maxLoad  float64
	mu       sync.RWMutex
}

// NewHashMap creates a new hash map
func NewHashMap[K comparable, V any](capacity ...int) *HashMap[K, V] {
	cap := 16
	if len(capacity) > 0 && capacity[0] > 0 {
		cap = capacity[0]
	}

	// Ensure capacity is power of 2
	for cap&(cap-1) != 0 {
		cap++
	}

	return &HashMap[K, V]{
		buckets:  make([]HashMapEntry[K, V], cap),
		capacity: uint64(cap),
		maxLoad:  0.75,
	}
}

// hashKey computes hash for a key
func (h *HashMap[K, V]) hashKey(key K) uint64 {
	hasher := fnv.New64a()
	hasher.Write([]byte(fmt.Sprintf("%v", key)))
	return hasher.Sum64()
}

// Insert inserts or updates a key-value pair
func (h *HashMap[K, V]) Insert(key K, value V) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if float64(h.len)/float64(h.capacity) > h.maxLoad {
		h.resize()
	}

	hash := h.hashKey(key)
	index := hash & (h.capacity - 1)

	for {
		entry := &h.buckets[index]
		var zeroKey K

		if entry.key == zeroKey { // Empty slot
			entry.key = key
			entry.value = value
			entry.hash = hash
			h.len++
			return
		}

		if entry.key == key { // Update existing
			entry.value = value
			return
		}

		index = (index + 1) & (h.capacity - 1)
	}
}

// Get retrieves a value by key
func (h *HashMap[K, V]) Get(key K) (V, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hash := h.hashKey(key)
	index := hash & (h.capacity - 1)

	for {
		entry := &h.buckets[index]
		var zeroKey K
		var zeroValue V

		if entry.key == zeroKey {
			return zeroValue, false
		}

		if entry.key == key {
			return entry.value, true
		}

		index = (index + 1) & (h.capacity - 1)
	}
}

// Remove removes a key-value pair
func (h *HashMap[K, V]) Remove(key K) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	hash := h.hashKey(key)
	index := hash & (h.capacity - 1)

	for {
		entry := &h.buckets[index]
		var zeroKey K

		if entry.key == zeroKey {
			return false
		}

		if entry.key == key {
			*entry = HashMapEntry[K, V]{} // Zero out
			h.len--
			// Compact following entries
			h.compactFrom(index)
			return true
		}

		index = (index + 1) & (h.capacity - 1)
	}
}

// Len returns the number of elements
func (h *HashMap[K, V]) Len() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.len
}

// resize doubles the capacity
func (h *HashMap[K, V]) resize() {
	oldBuckets := h.buckets
	h.capacity *= 2
	h.buckets = make([]HashMapEntry[K, V], h.capacity)
	h.len = 0

	for _, entry := range oldBuckets {
		var zeroKey K
		if entry.key != zeroKey {
			h.Insert(entry.key, entry.value)
		}
	}
}

// compactFrom compacts entries after removal
func (h *HashMap[K, V]) compactFrom(start uint64) {
	for i := (start + 1) & (h.capacity - 1); ; i = (i + 1) & (h.capacity - 1) {
		entry := &h.buckets[i]
		var zeroKey K

		if entry.key == zeroKey {
			break
		}

		// Find new position
		newIndex := entry.hash & (h.capacity - 1)
		h.buckets[newIndex] = *entry
		*entry = HashMapEntry[K, V]{}
	}
}

// ====== HashSet Implementation ======

// HashSet is a set implementation using HashMap
type HashSet[T comparable] struct {
	inner *HashMap[T, struct{}]
}

// NewHashSet creates a new hash set
func NewHashSet[T comparable](capacity ...int) *HashSet[T] {
	return &HashSet[T]{
		inner: NewHashMap[T, struct{}](capacity...),
	}
}

// Insert adds an element to the set
func (s *HashSet[T]) Insert(elem T) {
	s.inner.Insert(elem, struct{}{})
}

// Contains checks if an element exists
func (s *HashSet[T]) Contains(elem T) bool {
	_, exists := s.inner.Get(elem)
	return exists
}

// Remove removes an element from the set
func (s *HashSet[T]) Remove(elem T) bool {
	return s.inner.Remove(elem)
}

// Len returns the number of elements
func (s *HashSet[T]) Len() uint64 {
	return s.inner.Len()
}

// ====== Deque Implementation ======

// Deque is a double-ended queue
type Deque[T any] struct {
	data  []T
	head  int
	tail  int
	count int
	mu    sync.RWMutex
}

// NewDeque creates a new deque
func NewDeque[T any](capacity ...int) *Deque[T] {
	cap := 16
	if len(capacity) > 0 && capacity[0] > 0 {
		cap = capacity[0]
	}

	return &Deque[T]{
		data: make([]T, cap),
	}
}

// PushFront adds element to front
func (d *Deque[T]) PushFront(elem T) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.count == len(d.data) {
		d.resize()
	}

	d.head = (d.head - 1 + len(d.data)) % len(d.data)
	d.data[d.head] = elem
	d.count++
}

// PushBack adds element to back
func (d *Deque[T]) PushBack(elem T) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.count == len(d.data) {
		d.resize()
	}

	d.data[d.tail] = elem
	d.tail = (d.tail + 1) % len(d.data)
	d.count++
}

// PopFront removes element from front
func (d *Deque[T]) PopFront() (T, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var zero T
	if d.count == 0 {
		return zero, false
	}

	elem := d.data[d.head]
	d.head = (d.head + 1) % len(d.data)
	d.count--
	return elem, true
}

// PopBack removes element from back
func (d *Deque[T]) PopBack() (T, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var zero T
	if d.count == 0 {
		return zero, false
	}

	d.tail = (d.tail - 1 + len(d.data)) % len(d.data)
	elem := d.data[d.tail]
	d.count--
	return elem, true
}

// Len returns the number of elements
func (d *Deque[T]) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.count
}

// resize doubles the capacity
func (d *Deque[T]) resize() {
	newData := make([]T, len(d.data)*2)

	for i := 0; i < d.count; i++ {
		newData[i] = d.data[(d.head+i)%len(d.data)]
	}

	d.data = newData
	d.head = 0
	d.tail = d.count
}

// ====== Priority Queue Implementation ======

// PriorityQueueItem represents an item in the priority queue
type PriorityQueueItem[T any] struct {
	Value    T
	Priority int64
	Index    int
}

// PriorityQueue is a thread-safe priority queue using a binary heap
type PriorityQueue[T any] struct {
	items []*PriorityQueueItem[T]
	mu    sync.RWMutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue[T any]() *PriorityQueue[T] {
	return &PriorityQueue[T]{
		items: make([]*PriorityQueueItem[T], 0),
	}
}

// Push adds an item with priority
func (pq *PriorityQueue[T]) Push(value T, priority int64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &PriorityQueueItem[T]{
		Value:    value,
		Priority: priority,
		Index:    len(pq.items),
	}

	pq.items = append(pq.items, item)
	pq.heapifyUp(len(pq.items) - 1)
}

// Pop removes and returns the highest priority item
func (pq *PriorityQueue[T]) Pop() (T, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	var zero T
	if len(pq.items) == 0 {
		return zero, false
	}

	item := pq.items[0]
	last := len(pq.items) - 1
	pq.items[0] = pq.items[last]
	pq.items = pq.items[:last]

	if len(pq.items) > 0 {
		pq.items[0].Index = 0
		pq.heapifyDown(0)
	}

	return item.Value, true
}

// Peek returns the highest priority item without removing it
func (pq *PriorityQueue[T]) Peek() (T, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	var zero T
	if len(pq.items) == 0 {
		return zero, false
	}

	return pq.items[0].Value, true
}

// Len returns the number of items
func (pq *PriorityQueue[T]) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

// heapifyUp moves item up to maintain heap property
func (pq *PriorityQueue[T]) heapifyUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if pq.items[index].Priority <= pq.items[parent].Priority {
			break
		}

		pq.items[index], pq.items[parent] = pq.items[parent], pq.items[index]
		pq.items[index].Index = index
		pq.items[parent].Index = parent
		index = parent
	}
}

// heapifyDown moves item down to maintain heap property
func (pq *PriorityQueue[T]) heapifyDown(index int) {
	for {
		largest := index
		left := 2*index + 1
		right := 2*index + 2

		if left < len(pq.items) && pq.items[left].Priority > pq.items[largest].Priority {
			largest = left
		}

		if right < len(pq.items) && pq.items[right].Priority > pq.items[largest].Priority {
			largest = right
		}

		if largest == index {
			break
		}

		pq.items[index], pq.items[largest] = pq.items[largest], pq.items[index]
		pq.items[index].Index = index
		pq.items[largest].Index = largest
		index = largest
	}
}

// ====== Lock-Free Structures ======

// LockFreeStack is a lock-free stack using atomic operations
type LockFreeStack[T any] struct {
	head atomic.Pointer[StackNode[T]]
}

// StackNode represents a node in the lock-free stack
type StackNode[T any] struct {
	Value T
	Next  *StackNode[T]
}

// NewLockFreeStack creates a new lock-free stack
func NewLockFreeStack[T any]() *LockFreeStack[T] {
	return &LockFreeStack[T]{}
}

// Push adds an element to the stack
func (s *LockFreeStack[T]) Push(value T) {
	node := &StackNode[T]{Value: value}

	for {
		head := s.head.Load()
		node.Next = head
		if s.head.CompareAndSwap(head, node) {
			break
		}
	}
}

// Pop removes and returns the top element
func (s *LockFreeStack[T]) Pop() (T, bool) {
	for {
		head := s.head.Load()
		if head == nil {
			var zero T
			return zero, false
		}

		if s.head.CompareAndSwap(head, head.Next) {
			return head.Value, true
		}
	}
}

// IsEmpty checks if the stack is empty
func (s *LockFreeStack[T]) IsEmpty() bool {
	return s.head.Load() == nil
}

// ====== Atomic Counter ======

// AtomicCounter is a thread-safe counter
type AtomicCounter struct {
	value int64
}

// NewAtomicCounter creates a new atomic counter
func NewAtomicCounter(initial ...int64) *AtomicCounter {
	val := int64(0)
	if len(initial) > 0 {
		val = initial[0]
	}
	return &AtomicCounter{value: val}
}

// Inc increments the counter and returns the new value
func (c *AtomicCounter) Inc() int64 {
	return atomic.AddInt64(&c.value, 1)
}

// Dec decrements the counter and returns the new value
func (c *AtomicCounter) Dec() int64 {
	return atomic.AddInt64(&c.value, -1)
}

// Add adds a value to the counter and returns the new value
func (c *AtomicCounter) Add(delta int64) int64 {
	return atomic.AddInt64(&c.value, delta)
}

// Get returns the current value
func (c *AtomicCounter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// Set sets the counter to a specific value
func (c *AtomicCounter) Set(value int64) {
	atomic.StoreInt64(&c.value, value)
}

// CompareAndSwap atomically compares and swaps the value
func (c *AtomicCounter) CompareAndSwap(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&c.value, old, new)
}
