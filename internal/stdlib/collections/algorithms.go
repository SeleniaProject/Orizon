// Advanced Algorithms and Benchmarking for Orizon Collections
package collections

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ====== Parallel Sorting Algorithms ======

type ParallelSorter[T any] struct {
	compare func(T, T) int
	workers int
}

func NewParallelSorter[T any](compare func(T, T) int) *ParallelSorter[T] {
	return &ParallelSorter[T]{
		compare: compare,
		workers: runtime.NumCPU(),
	}
}

func (ps *ParallelSorter[T]) ParallelQuickSort(data []T) {
	if len(data) < 2 {
		return
	}

	threshold := 10000 // Use parallel for arrays larger than this
	if len(data) < threshold {
		ps.quickSort(data, 0, len(data)-1)
		return
	}

	var wg sync.WaitGroup
	ps.parallelQuickSortHelper(data, 0, len(data)-1, &wg, 0)
	wg.Wait()
}

func (ps *ParallelSorter[T]) parallelQuickSortHelper(data []T, low, high int, wg *sync.WaitGroup, depth int) {
	if low < high {
		maxDepth := int(math.Log2(float64(ps.workers)))

		if depth < maxDepth && (high-low) > 1000 {
			// Parallel execution
			pivot := ps.partition(data, low, high)

			wg.Add(2)
			go func() {
				defer wg.Done()
				ps.parallelQuickSortHelper(data, low, pivot-1, wg, depth+1)
			}()
			go func() {
				defer wg.Done()
				ps.parallelQuickSortHelper(data, pivot+1, high, wg, depth+1)
			}()
		} else {
			// Sequential execution
			ps.quickSort(data, low, high)
		}
	}
}

func (ps *ParallelSorter[T]) quickSort(data []T, low, high int) {
	if low < high {
		pivot := ps.partition(data, low, high)
		ps.quickSort(data, low, pivot-1)
		ps.quickSort(data, pivot+1, high)
	}
}

func (ps *ParallelSorter[T]) partition(data []T, low, high int) int {
	pivot := data[high]
	i := low - 1

	for j := low; j < high; j++ {
		if ps.compare(data[j], pivot) <= 0 {
			i++
			data[i], data[j] = data[j], data[i]
		}
	}
	data[i+1], data[high] = data[high], data[i+1]
	return i + 1
}

func (ps *ParallelSorter[T]) ParallelMergeSort(data []T) {
	if len(data) <= 1 {
		return
	}

	temp := make([]T, len(data))
	ps.parallelMergeSortHelper(data, temp, 0, len(data)-1, 0)
}

func (ps *ParallelSorter[T]) parallelMergeSortHelper(data, temp []T, left, right, depth int) {
	if left >= right {
		return
	}

	mid := (left + right) / 2
	maxDepth := int(math.Log2(float64(ps.workers)))

	if depth < maxDepth && (right-left) > 1000 {
		// Parallel execution
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			ps.parallelMergeSortHelper(data, temp, left, mid, depth+1)
		}()
		go func() {
			defer wg.Done()
			ps.parallelMergeSortHelper(data, temp, mid+1, right, depth+1)
		}()

		wg.Wait()
	} else {
		// Sequential execution
		ps.parallelMergeSortHelper(data, temp, left, mid, depth+1)
		ps.parallelMergeSortHelper(data, temp, mid+1, right, depth+1)
	}

	ps.merge(data, temp, left, mid, right)
}

func (ps *ParallelSorter[T]) merge(data, temp []T, left, mid, right int) {
	copy(temp[left:right+1], data[left:right+1])

	i, j, k := left, mid+1, left

	for i <= mid && j <= right {
		if ps.compare(temp[i], temp[j]) <= 0 {
			data[k] = temp[i]
			i++
		} else {
			data[k] = temp[j]
			j++
		}
		k++
	}

	for i <= mid {
		data[k] = temp[i]
		i++
		k++
	}

	for j <= right {
		data[k] = temp[j]
		j++
		k++
	}
}

// ====== Advanced Tree Structures ======

// Red-Black Tree
type RBColor int

const (
	Red RBColor = iota
	Black
)

type RBNode[K comparable, V any] struct {
	key    K
	value  V
	color  RBColor
	left   *RBNode[K, V]
	right  *RBNode[K, V]
	parent *RBNode[K, V]
}

type RedBlackTree[K comparable, V any] struct {
	root    *RBNode[K, V]
	nil     *RBNode[K, V] // Sentinel node
	compare func(K, K) int
	size    int
	mu      sync.RWMutex
}

func NewRedBlackTree[K comparable, V any](compare func(K, K) int) *RedBlackTree[K, V] {
	var zeroK K
	var zeroV V

	nil := &RBNode[K, V]{
		key:   zeroK,
		value: zeroV,
		color: Black,
	}

	rbt := &RedBlackTree[K, V]{
		nil:     nil,
		root:    nil,
		compare: compare,
	}

	rbt.root = rbt.nil
	return rbt
}

func (rbt *RedBlackTree[K, V]) Insert(key K, value V) {
	rbt.mu.Lock()
	defer rbt.mu.Unlock()

	node := &RBNode[K, V]{
		key:    key,
		value:  value,
		color:  Red,
		left:   rbt.nil,
		right:  rbt.nil,
		parent: rbt.nil,
	}

	var parent *RBNode[K, V] = rbt.nil
	current := rbt.root

	// Standard BST insertion
	for current != rbt.nil {
		parent = current
		if rbt.compare(key, current.key) < 0 {
			current = current.left
		} else if rbt.compare(key, current.key) > 0 {
			current = current.right
		} else {
			// Update existing
			current.value = value
			return
		}
	}

	node.parent = parent
	if parent == rbt.nil {
		rbt.root = node
	} else if rbt.compare(key, parent.key) < 0 {
		parent.left = node
	} else {
		parent.right = node
	}

	rbt.size++
	rbt.insertFixup(node)
}

func (rbt *RedBlackTree[K, V]) insertFixup(node *RBNode[K, V]) {
	for node.parent.color == Red {
		if node.parent == node.parent.parent.left {
			uncle := node.parent.parent.right
			if uncle.color == Red {
				// Case 1: Uncle is red
				node.parent.color = Black
				uncle.color = Black
				node.parent.parent.color = Red
				node = node.parent.parent
			} else {
				if node == node.parent.right {
					// Case 2: Node is right child
					node = node.parent
					rbt.leftRotate(node)
				}
				// Case 3: Node is left child
				node.parent.color = Black
				node.parent.parent.color = Red
				rbt.rightRotate(node.parent.parent)
			}
		} else {
			// Symmetric cases
			uncle := node.parent.parent.left
			if uncle.color == Red {
				node.parent.color = Black
				uncle.color = Black
				node.parent.parent.color = Red
				node = node.parent.parent
			} else {
				if node == node.parent.left {
					node = node.parent
					rbt.rightRotate(node)
				}
				node.parent.color = Black
				node.parent.parent.color = Red
				rbt.leftRotate(node.parent.parent)
			}
		}
	}
	rbt.root.color = Black
}

func (rbt *RedBlackTree[K, V]) leftRotate(x *RBNode[K, V]) {
	y := x.right
	x.right = y.left

	if y.left != rbt.nil {
		y.left.parent = x
	}

	y.parent = x.parent
	if x.parent == rbt.nil {
		rbt.root = y
	} else if x == x.parent.left {
		x.parent.left = y
	} else {
		x.parent.right = y
	}

	y.left = x
	x.parent = y
}

func (rbt *RedBlackTree[K, V]) rightRotate(y *RBNode[K, V]) {
	x := y.left
	y.left = x.right

	if x.right != rbt.nil {
		x.right.parent = y
	}

	x.parent = y.parent
	if y.parent == rbt.nil {
		rbt.root = x
	} else if y == y.parent.right {
		y.parent.right = x
	} else {
		y.parent.left = x
	}

	x.right = y
	y.parent = x
}

func (rbt *RedBlackTree[K, V]) Search(key K) (V, bool) {
	rbt.mu.RLock()
	defer rbt.mu.RUnlock()

	current := rbt.root
	for current != rbt.nil {
		cmp := rbt.compare(key, current.key)
		if cmp == 0 {
			return current.value, true
		} else if cmp < 0 {
			current = current.left
		} else {
			current = current.right
		}
	}

	var zero V
	return zero, false
}

func (rbt *RedBlackTree[K, V]) Size() int {
	rbt.mu.RLock()
	defer rbt.mu.RUnlock()
	return rbt.size
}

// ====== Performance Benchmarking ======

type BenchmarkResult struct {
	Name           string
	Duration       time.Duration
	Operations     int64
	BytesAllocated uint64
	Allocations    uint64
	ThroughputOps  float64
}

type Benchmark struct {
	results []BenchmarkResult
	mu      sync.Mutex
}

func NewBenchmark() *Benchmark {
	return &Benchmark{
		results: make([]BenchmarkResult, 0),
	}
}

func (b *Benchmark) Run(name string, operations int64, fn func()) BenchmarkResult {
	runtime.GC() // Clean slate

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()
	fn()
	duration := time.Since(start)

	runtime.ReadMemStats(&m2)

	result := BenchmarkResult{
		Name:           name,
		Duration:       duration,
		Operations:     operations,
		BytesAllocated: m2.TotalAlloc - m1.TotalAlloc,
		Allocations:    m2.Mallocs - m1.Mallocs,
		ThroughputOps:  float64(operations) / duration.Seconds(),
	}

	b.mu.Lock()
	b.results = append(b.results, result)
	b.mu.Unlock()

	return result
}

func (b *Benchmark) RunParallel(name string, operations int64, parallelism int, fn func()) BenchmarkResult {
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()

	var wg sync.WaitGroup
	opsPerWorker := operations / int64(parallelism)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := int64(0); j < opsPerWorker; j++ {
				fn()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&m2)

	result := BenchmarkResult{
		Name:           fmt.Sprintf("%s (parallel=%d)", name, parallelism),
		Duration:       duration,
		Operations:     operations,
		BytesAllocated: m2.TotalAlloc - m1.TotalAlloc,
		Allocations:    m2.Mallocs - m1.Mallocs,
		ThroughputOps:  float64(operations) / duration.Seconds(),
	}

	b.mu.Lock()
	b.results = append(b.results, result)
	b.mu.Unlock()

	return result
}

func (b *Benchmark) Results() []BenchmarkResult {
	b.mu.Lock()
	defer b.mu.Unlock()

	results := make([]BenchmarkResult, len(b.results))
	copy(results, b.results)
	return results
}

func (b *Benchmark) PrintResults() {
	results := b.Results()

	fmt.Printf("%-40s %15s %15s %15s %15s %15s\n",
		"Benchmark", "Duration", "Operations", "Ops/sec", "B/op", "Allocs/op")
	fmt.Println(strings.Repeat("-", 120))

	for _, result := range results {
		bytesPerOp := float64(result.BytesAllocated) / float64(result.Operations)
		allocsPerOp := float64(result.Allocations) / float64(result.Operations)

		fmt.Printf("%-40s %15s %15d %15.0f %15.2f %15.2f\n",
			result.Name,
			result.Duration.Round(time.Microsecond),
			result.Operations,
			result.ThroughputOps,
			bytesPerOp,
			allocsPerOp)
	}
}

// ====== Stress Testing Framework ======

type StressTest struct {
	name       string
	duration   time.Duration
	goroutines int
	operations atomic.Int64
	errors     atomic.Int64
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewStressTest(name string, duration time.Duration, goroutines int) *StressTest {
	ctx, cancel := context.WithTimeout(context.Background(), duration)

	return &StressTest{
		name:       name,
		duration:   duration,
		goroutines: goroutines,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (st *StressTest) Run(fn func() error) {
	var wg sync.WaitGroup

	fmt.Printf("Starting stress test: %s (duration=%v, goroutines=%d)\n",
		st.name, st.duration, st.goroutines)

	for i := 0; i < st.goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-st.ctx.Done():
					return
				default:
					if err := fn(); err != nil {
						st.errors.Add(1)
					}
					st.operations.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	fmt.Printf("Stress test completed: %s\n", st.name)
	fmt.Printf("  Operations: %d\n", st.operations.Load())
	fmt.Printf("  Errors: %d\n", st.errors.Load())
	fmt.Printf("  Success rate: %.2f%%\n",
		float64(st.operations.Load()-st.errors.Load())/float64(st.operations.Load())*100)
}

// ====== Concurrent Hash Map with Advanced Features ======

type AdvancedHashMap[K comparable, V any] struct {
	segments []*hashMapSegment[K, V]
	segMask  uint32
	hash     func(K) uint32
	mu       sync.RWMutex
	size     atomic.Int64
	resizing atomic.Bool
}

type hashMapSegment[K comparable, V any] struct {
	buckets []hashMapBucket[K, V]
	mask    uint32
	mu      sync.RWMutex
}

type hashMapBucket[K comparable, V any] struct {
	entries []hashMapEntry[K, V]
	mu      sync.RWMutex
}

type hashMapEntry[K comparable, V any] struct {
	key     K
	value   V
	hash    uint32
	deleted bool
}

func NewAdvancedHashMap[K comparable, V any](hashFn func(K) uint32, initialCap int) *AdvancedHashMap[K, V] {
	segmentCount := uint32(16) // Fixed number of segments

	segments := make([]*hashMapSegment[K, V], segmentCount)
	bucketsPerSeg := uint32(initialCap) / segmentCount
	if bucketsPerSeg == 0 {
		bucketsPerSeg = 1
	}

	for i := range segments {
		buckets := make([]hashMapBucket[K, V], bucketsPerSeg)
		for j := range buckets {
			buckets[j].entries = make([]hashMapEntry[K, V], 0, 4)
		}

		segments[i] = &hashMapSegment[K, V]{
			buckets: buckets,
			mask:    bucketsPerSeg - 1,
		}
	}

	return &AdvancedHashMap[K, V]{
		segments: segments,
		segMask:  segmentCount - 1,
		hash:     hashFn,
	}
}

func (m *AdvancedHashMap[K, V]) Put(key K, value V) {
	if m == nil || m.segments == nil {
		return
	}

	hash := m.hash(key)
	segIdx := hash & m.segMask

	if segIdx >= uint32(len(m.segments)) || m.segments[segIdx] == nil {
		return
	}

	segment := m.segments[segIdx]
	bucketIdx := hash & segment.mask

	if bucketIdx >= uint32(len(segment.buckets)) {
		return
	}

	bucket := &segment.buckets[bucketIdx]

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Check if key exists
	for i := range bucket.entries {
		if !bucket.entries[i].deleted &&
			bucket.entries[i].hash == hash &&
			bucket.entries[i].key == key {
			bucket.entries[i].value = value
			return
		}
	}

	// Add new entry
	bucket.entries = append(bucket.entries, hashMapEntry[K, V]{
		key:   key,
		value: value,
		hash:  hash,
	})

	m.size.Add(1)

	// Check load factor and resize if needed
	if len(bucket.entries) > 8 && !m.resizing.Load() {
		go m.tryResize()
	}
}

func (m *AdvancedHashMap[K, V]) Get(key K) (V, bool) {
	if m == nil || m.segments == nil {
		var zero V
		return zero, false
	}

	hash := m.hash(key)
	segIdx := hash & m.segMask

	if segIdx >= uint32(len(m.segments)) || m.segments[segIdx] == nil {
		var zero V
		return zero, false
	}

	segment := m.segments[segIdx]
	bucketIdx := hash & segment.mask

	if bucketIdx >= uint32(len(segment.buckets)) {
		var zero V
		return zero, false
	}

	bucket := &segment.buckets[bucketIdx]

	bucket.mu.RLock()
	defer bucket.mu.RUnlock()

	for i := range bucket.entries {
		if !bucket.entries[i].deleted &&
			bucket.entries[i].hash == hash &&
			bucket.entries[i].key == key {
			return bucket.entries[i].value, true
		}
	}

	var zero V
	return zero, false
}

func (m *AdvancedHashMap[K, V]) Remove(key K) bool {
	if m == nil || m.segments == nil {
		return false
	}

	hash := m.hash(key)
	segIdx := hash & m.segMask

	if segIdx >= uint32(len(m.segments)) || m.segments[segIdx] == nil {
		return false
	}

	segment := m.segments[segIdx]
	bucketIdx := hash & segment.mask

	if bucketIdx >= uint32(len(segment.buckets)) {
		return false
	}

	bucket := &segment.buckets[bucketIdx]

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	for i := range bucket.entries {
		if !bucket.entries[i].deleted &&
			bucket.entries[i].hash == hash &&
			bucket.entries[i].key == key {
			bucket.entries[i].deleted = true
			m.size.Add(-1)
			return true
		}
	}

	return false
}

func (m *AdvancedHashMap[K, V]) Size() int64 {
	return m.size.Load()
}

func (m *AdvancedHashMap[K, V]) tryResize() {
	if !m.resizing.CompareAndSwap(false, true) {
		return // Already resizing
	}
	defer m.resizing.Store(false)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double the bucket count for each segment
	for _, segment := range m.segments {
		segment.mu.Lock()

		oldBuckets := segment.buckets
		newSize := len(oldBuckets) * 2
		newBuckets := make([]hashMapBucket[K, V], newSize)

		for i := range newBuckets {
			newBuckets[i].entries = make([]hashMapEntry[K, V], 0, 4)
		}

		newMask := uint32(newSize - 1)

		// Rehash all entries with proper synchronization to avoid mutex value copying
		for i := range oldBuckets {
			// Use index access to avoid copying the mutex-containing bucket struct
			bucket := &oldBuckets[i]
			bucket.mu.RLock() // Acquire read lock for safe access

			for _, entry := range bucket.entries {
				if !entry.deleted {
					newBucketIdx := entry.hash & newMask
					newBuckets[newBucketIdx].entries = append(
						newBuckets[newBucketIdx].entries, entry)
				}
			}

			bucket.mu.RUnlock() // Release read lock
		}

		segment.buckets = newBuckets
		segment.mask = newMask
		segment.mu.Unlock()
	}
}
