package collections

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// ====== Hash Functions for Testing ======

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func hashInt(i int) uint32 {
	return uint32(i * 2654435761) // Knuth's multiplicative hash
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

func compareString(a, b string) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// ====== Concurrent Data Structure Tests ======

func TestLockFreeQueue(t *testing.T) {
	queue := NewLockFreeQueue[int]()

	// Test basic operations
	queue.Enqueue(1)
	queue.Enqueue(2)

	val, ok := queue.Dequeue()
	if !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}

	val, ok = queue.Dequeue()
	if !ok || val != 2 {
		t.Errorf("Expected 2, got %v", val)
	}

	// Test concurrent operations
	var wg sync.WaitGroup
	producers := 2
	consumers := 2
	itemsPerProducer := 10

	consumed := make([]int, 0, producers*itemsPerProducer)
	var mu sync.Mutex
	done := make(chan struct{})

	// Start producers
	producerWg := sync.WaitGroup{}
	for i := 0; i < producers; i++ {
		producerWg.Add(1)
		go func(start int) {
			defer producerWg.Done()
			for j := 0; j < itemsPerProducer; j++ {
				queue.Enqueue(start*itemsPerProducer + j)
			}
		}(i)
	}

	// Start consumers
	for i := 0; i < consumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					if val, ok := queue.Dequeue(); ok {
						mu.Lock()
						consumed = append(consumed, val)
						mu.Unlock()
					} else {
						time.Sleep(time.Nanosecond)
					}
				}
			}
		}()
	}

	// Wait for producers to finish, then signal consumers to stop
	go func() {
		producerWg.Wait()
		// Give consumers a bit more time to drain the queue
		for {
			mu.Lock()
			count := len(consumed)
			mu.Unlock()
			if count >= producers*itemsPerProducer {
				break
			}
			time.Sleep(time.Microsecond)
		}
		close(done)
	}()

	wg.Wait()

	if len(consumed) != producers*itemsPerProducer {
		t.Errorf("Expected %d items, got %d", producers*itemsPerProducer, len(consumed))
	}
}

func TestWaitFreeRingBuffer(t *testing.T) {
	buffer := NewWaitFreeRingBuffer[string](8) // Power of 2

	// Test basic operations
	if !buffer.Push("a") {
		t.Error("Failed to push to empty buffer")
	}

	if !buffer.Push("b") {
		t.Error("Failed to push second item")
	}

	val, ok := buffer.Pop()
	if !ok || val != "a" {
		t.Errorf("Expected 'a', got %v", val)
	}

	// Test overflow (buffer has "b" already, so we can add 6 more)
	for i := 0; i < 6; i++ {
		if !buffer.Push(fmt.Sprintf("item%d", i)) {
			t.Errorf("Failed to push item %d", i)
		}
	}

	// Buffer should be full now (has 7 items total)
	if buffer.Push("overflow") {
		t.Error("Buffer should be full")
	}

	// Test concurrent access
	var wg sync.WaitGroup
	pushCount := 0
	popCount := 0
	var mu sync.Mutex

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			for !buffer.Push(fmt.Sprintf("item%d", i)) {
				time.Sleep(time.Microsecond)
			}
			mu.Lock()
			pushCount++
			mu.Unlock()
		}
	}()

	// Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if _, ok := buffer.Pop(); ok {
				mu.Lock()
				popCount++
				if popCount >= 1000 {
					mu.Unlock()
					return
				}
				mu.Unlock()
			} else {
				time.Sleep(time.Microsecond)
			}
		}
	}()

	wg.Wait()

	if pushCount != 1000 || popCount != 1000 {
		t.Errorf("Expected 1000 push/pop operations, got %d/%d", pushCount, popCount)
	}
}

func TestRedBlackTree(t *testing.T) {
	tree := NewRedBlackTree[int, string](compareInt)

	// Test insertions
	tree.Insert(10, "ten")
	tree.Insert(5, "five")
	tree.Insert(15, "fifteen")
	tree.Insert(3, "three")
	tree.Insert(7, "seven")
	tree.Insert(12, "twelve")
	tree.Insert(18, "eighteen")

	// Test searches
	if val, ok := tree.Search(10); !ok || val != "ten" {
		t.Errorf("Expected 'ten', got %v", val)
	}

	if val, ok := tree.Search(7); !ok || val != "seven" {
		t.Errorf("Expected 'seven', got %v", val)
	}

	if _, ok := tree.Search(100); ok {
		t.Error("Should not find non-existent key")
	}

	// Test size
	if tree.Size() != 7 {
		t.Errorf("Expected size 7, got %d", tree.Size())
	}

	// Test concurrent operations
	var wg sync.WaitGroup
	operations := 1000

	// Concurrent inserts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < operations/10; j++ {
				key := start*operations + j
				tree.Insert(key, fmt.Sprintf("value%d", key))
			}
		}(i)
	}

	// Concurrent searches
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				tree.Search(j)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentSkipList(t *testing.T) {
	skipList := NewConcurrentSkipList[int, string](compareInt)

	// Test basic operations
	skipList.Insert(10, "ten")
	skipList.Insert(5, "five")
	skipList.Insert(15, "fifteen")

	if val, ok := skipList.Search(10); !ok || val != "ten" {
		t.Errorf("Expected 'ten', got %v", val)
	}

	if _, ok := skipList.Search(100); ok {
		t.Error("Should not find non-existent key")
	}

	// Test concurrent operations
	var wg sync.WaitGroup
	items := 10000

	// Concurrent inserts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < items/10; j++ {
				key := start*(items/10) + j
				skipList.Insert(key, fmt.Sprintf("value%d", key))
			}
		}(i)
	}

	wg.Wait()

	// Verify all insertions
	missing := 0
	for i := 0; i < items; i++ {
		if _, ok := skipList.Search(i); !ok {
			missing++
		}
	}

	if missing > 0 {
		t.Errorf("Missing %d items from skip list", missing)
	}
}

func TestParallelSorting(t *testing.T) {
	sorter := NewParallelSorter[int](compareInt)

	// Test parallel quicksort
	data := make([]int, 100000)
	for i := range data {
		data[i] = rand.Int()
	}

	original := make([]int, len(data))
	copy(original, data)

	start := time.Now()
	sorter.ParallelQuickSort(data)
	duration := time.Since(start)

	// Verify sorted
	for i := 1; i < len(data); i++ {
		if data[i-1] > data[i] {
			t.Error("Array not properly sorted")
			break
		}
	}

	fmt.Printf("Parallel QuickSort of %d elements took %v\n", len(data), duration)

	// Test parallel mergesort
	copy(data, original)

	start = time.Now()
	sorter.ParallelMergeSort(data)
	duration = time.Since(start)

	// Verify sorted
	for i := 1; i < len(data); i++ {
		if data[i-1] > data[i] {
			t.Error("Array not properly sorted by merge sort")
			break
		}
	}

	fmt.Printf("Parallel MergeSort of %d elements took %v\n", len(data), duration)
}

func TestAdvancedHashMap(t *testing.T) {
	hashMap := NewAdvancedHashMap[string, int](hashString, 16)

	// Test basic operations
	hashMap.Put("key1", 1)
	hashMap.Put("key2", 2)
	hashMap.Put("key3", 3)

	if val, ok := hashMap.Get("key1"); !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}

	if val, ok := hashMap.Get("key2"); !ok || val != 2 {
		t.Errorf("Expected 2, got %v", val)
	}

	// Test removal
	if !hashMap.Remove("key2") {
		t.Error("Failed to remove existing key")
	}

	if _, ok := hashMap.Get("key2"); ok {
		t.Error("Key should be removed")
	}

	if hashMap.Size() != 2 {
		t.Errorf("Expected size 2, got %d", hashMap.Size())
	}

	// Test concurrent operations
	var wg sync.WaitGroup
	operations := 10000

	// Concurrent puts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < operations/10; j++ {
				key := fmt.Sprintf("key%d", start*operations+j)
				hashMap.Put(key, start*operations+j)
			}
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key%d", j)
				hashMap.Get(key)
			}
		}()
	}

	wg.Wait()

	fmt.Printf("Advanced HashMap size after concurrent operations: %d\n", hashMap.Size())
}

// ====== Benchmark Tests ======

func TestBenchmarkFramework(t *testing.T) {
	bench := NewBenchmark()

	// Benchmark vector operations
	vec := NewVector[int]()

	result := bench.Run("Vector Push", 100000, func() {
		vec.Push(42)
	})

	fmt.Printf("Vector Push: %v ops/sec, %v ns/op\n",
		result.ThroughputOps,
		result.Duration.Nanoseconds()/result.Operations)

	// Benchmark parallel operations
	result = bench.RunParallel("Parallel HashMap Put", 100000, 8, func() {
		hashMap := NewAdvancedHashMap[int, string](hashInt, 1024)
		key := rand.Int()
		hashMap.Put(key, fmt.Sprintf("value%d", key))
	})

	fmt.Printf("Parallel HashMap Put: %v ops/sec\n", result.ThroughputOps)

	bench.PrintResults()
}

func TestStressTest(t *testing.T) {
	// Create a stress test for concurrent hash map
	hashMap := NewAdvancedHashMap[string, int](hashString, 64)

	stressTest := NewStressTest("HashMap Stress", 2*time.Second, 20)

	stressTest.Run(func() error {
		key := fmt.Sprintf("key%d", rand.Int()%10000)

		switch rand.Int() % 3 {
		case 0: // Put
			hashMap.Put(key, rand.Int())
		case 1: // Get
			hashMap.Get(key)
		case 2: // Remove
			hashMap.Remove(key)
		}

		return nil
	})
}

func TestNUMAPool(t *testing.T) {
	// Test NUMA-aware object pool
	pool := NewNUMAPool[[]byte](
		func() []byte {
			return make([]byte, 1024)
		},
		func(buf *[]byte) {
			*buf = (*buf)[:0] // Reset slice length
		},
	)

	// Test basic operations
	buf1 := pool.Get()
	if len(buf1) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buf1))
	}

	buf1 = append(buf1[:0], []byte("test data")...)
	pool.Put(buf1)

	buf2 := pool.Get()
	if len(buf2) != 0 { // Should be reset
		t.Errorf("Expected reset buffer, got length %d", len(buf2))
	}

	// Test concurrent operations
	var wg sync.WaitGroup
	operations := 10000

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operations/10; j++ {
				buf := pool.Get()
				buf = append(buf[:0], []byte("test")...)
				pool.Put(buf)
			}
		}()
	}

	wg.Wait()
}

// ====== Performance Comparison Tests ======

func BenchmarkDataStructureComparison(b *testing.B) {
	// Compare different map implementations
	stdMap := make(map[string]int)
	var stdMapMu sync.RWMutex

	advancedMap := NewAdvancedHashMap[string, int](hashString, 1024)

	keys := make([]string, 10000)
	for i := range keys {
		keys[i] = fmt.Sprintf("key%d", i)
	}

	b.Run("StdMap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			stdMapMu.Lock()
			stdMap[key] = i
			stdMapMu.Unlock()
		}
	})

	b.Run("AdvancedHashMap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			advancedMap.Put(key, i)
		}
	})
}

func BenchmarkConcurrentOperations(b *testing.B) {
	hashMap := NewAdvancedHashMap[int, string](hashInt, 1024)

	// Populate with initial data
	for i := 0; i < 1000; i++ {
		hashMap.Put(i, fmt.Sprintf("value%d", i))
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Int() % 1000

			switch rand.Int() % 3 {
			case 0:
				hashMap.Put(key, fmt.Sprintf("value%d", key))
			case 1:
				hashMap.Get(key)
			case 2:
				hashMap.Remove(key)
			}
		}
	})
}

// ====== Memory and Performance Profiling ======

func TestMemoryUsage(t *testing.T) {
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create large data structures
	vec := NewVector[int]()
	for i := 0; i < 100000; i++ {
		vec.Push(i)
	}

	hashMap := NewAdvancedHashMap[int, string](hashInt, 1024)
	for i := 0; i < 50000; i++ {
		hashMap.Put(i, fmt.Sprintf("value%d", i))
	}

	runtime.ReadMemStats(&m2)

	allocated := m2.TotalAlloc - m1.TotalAlloc
	fmt.Printf("Memory allocated for data structures: %d bytes\n", allocated)
	fmt.Printf("Vector size: %d elements\n", vec.Len())
	fmt.Printf("HashMap size: %d elements\n", hashMap.Size())
}
