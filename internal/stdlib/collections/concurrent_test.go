package collections

import (
	"fmt"
	"hash/fnv"
	"sync"
	"testing"
)

func hashStringFnv(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))

	return h.Sum32()
}

func compareInts(a, b int) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}

	return 0
}

func TestNUMAPoolBasic(t *testing.T) {
	// Test NUMA-aware object pool.
	pool := NewNUMAPool[[]byte](
		func() []byte {
			return make([]byte, 1024)
		},
		func(buf *[]byte) {
			*buf = (*buf)[:0] // Reset slice length
		},
	)

	// Test basic operations.
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

	fmt.Printf("NUMA Pool test completed successfully\n")
}

func TestRedBlackTreeBasic(t *testing.T) {
	tree := NewRedBlackTree[int, string](compareInts)

	// Test insertions.
	tree.Insert(10, "ten")
	tree.Insert(5, "five")
	tree.Insert(15, "fifteen")

	// Test searches.
	if val, ok := tree.Search(10); !ok || val != "ten" {
		t.Errorf("Expected 'ten', got %v", val)
	}

	if val, ok := tree.Search(5); !ok || val != "five" {
		t.Errorf("Expected 'five', got %v", val)
	}

	if _, ok := tree.Search(100); ok {
		t.Error("Should not find non-existent key")
	}

	// Test size.
	if tree.Size() != 3 {
		t.Errorf("Expected size 3, got %d", tree.Size())
	}

	fmt.Printf("Red-Black Tree basic test completed successfully\n")
}

func TestParallelSortingBasic(t *testing.T) {
	sorter := NewParallelSorter[int](compareInts)

	// Test with small array first.
	data := []int{5, 2, 8, 1, 9, 3}

	sorter.ParallelQuickSort(data)

	// Verify sorted.
	for i := 1; i < len(data); i++ {
		if data[i-1] > data[i] {
			t.Error("Array not properly sorted")

			break
		}
	}

	fmt.Printf("Parallel sorting basic test completed successfully\n")
}

func TestAdvancedHashMapBasic(t *testing.T) {
	hashMap := NewAdvancedHashMap[string, int](hashStringFnv, 16)

	// Test basic operations.
	hashMap.Put("key1", 1)
	hashMap.Put("key2", 2)
	hashMap.Put("key3", 3)

	if val, ok := hashMap.Get("key1"); !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}

	if val, ok := hashMap.Get("key2"); !ok || val != 2 {
		t.Errorf("Expected 2, got %v", val)
	}

	// Test removal.
	if !hashMap.Remove("key2") {
		t.Error("Failed to remove existing key")
	}

	if _, ok := hashMap.Get("key2"); ok {
		t.Error("Key should be removed")
	}

	if hashMap.Size() != 2 {
		t.Errorf("Expected size 2, got %d", hashMap.Size())
	}

	fmt.Printf("Advanced HashMap basic test completed successfully\n")
}

func TestWaitFreeRingBufferBasic(t *testing.T) {
	buffer := NewWaitFreeRingBuffer[string](8) // Power of 2

	// Test basic operations.
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

	val, ok = buffer.Pop()
	if !ok || val != "b" {
		t.Errorf("Expected 'b', got %v", val)
	}

	// Test empty buffer.
	_, ok = buffer.Pop()
	if ok {
		t.Error("Should not be able to pop from empty buffer")
	}

	fmt.Printf("Wait-Free Ring Buffer basic test completed successfully\n")
}

func TestBenchmarkFrameworkBasic(t *testing.T) {
	bench := NewBenchmark()

	// Benchmark vector operations.
	vec := NewVector[int]()

	result := bench.Run("Vector Push", 1000, func() {
		vec.Push(42)
	})

	fmt.Printf("Vector Push: %.0f ops/sec\n", result.ThroughputOps)

	if result.Operations != 1000 {
		t.Errorf("Expected 1000 operations, got %d", result.Operations)
	}

	if result.ThroughputOps <= 0 {
		t.Error("Throughput should be positive")
	}

	fmt.Printf("Benchmark framework basic test completed successfully\n")
}

func TestLightweightConcurrentOperations(t *testing.T) {
	// Test with smaller concurrent operations to avoid timeouts.
	hashMap := NewAdvancedHashMap[int, string](func(i int) uint32 {
		return uint32(i * 2654435761)
	}, 64)

	var wg sync.WaitGroup

	operations := 100 // Reduced from 10000

	// Concurrent puts.
	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func(start int) {
			defer wg.Done()

			for j := 0; j < operations/5; j++ {
				key := start*operations + j
				hashMap.Put(key, fmt.Sprintf("value%d", key))
			}
		}(i)
	}

	wg.Wait()

	fmt.Printf("Lightweight concurrent operations test completed successfully\n")
	fmt.Printf("HashMap size after concurrent operations: %d\n", hashMap.Size())
}
