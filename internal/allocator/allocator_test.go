package allocator

import (
	"testing"
	"unsafe"
)

// TestSystemAllocator tests the system allocator implementation
func TestSystemAllocator(t *testing.T) {
	config := defaultConfig()
	allocator := NewSystemAllocator(config)

	t.Run("BasicAllocation", func(t *testing.T) {
		ptr := allocator.Alloc(1024)
		if ptr == nil {
			t.Fatal("Allocation failed")
		}

		// Write to memory to ensure it's valid
		data := (*[1024]byte)(ptr)
		for i := 0; i < 1024; i++ {
			data[i] = byte(i % 256)
		}

		// Verify data
		for i := 0; i < 1024; i++ {
			if data[i] != byte(i%256) {
				t.Errorf("Data corruption at index %d", i)
			}
		}

		allocator.Free(ptr)
	})

	t.Run("ZeroAllocation", func(t *testing.T) {
		ptr := allocator.Alloc(0)
		if ptr != nil {
			t.Error("Zero allocation should return nil")
		}
	})

	t.Run("Reallocation", func(t *testing.T) {
		ptr := allocator.Alloc(512)
		if ptr == nil {
			t.Fatal("Initial allocation failed")
		}

		// Write test data
		data := (*[512]byte)(ptr)
		for i := 0; i < 512; i++ {
			data[i] = byte(i % 256)
		}

		// Reallocate to larger size
		newPtr := allocator.Realloc(ptr, 1024)
		if newPtr == nil {
			t.Fatal("Reallocation failed")
		}

		// Verify original data is preserved
		newData := (*[1024]byte)(newPtr)
		for i := 0; i < 512; i++ {
			if newData[i] != byte(i%256) {
				t.Errorf("Data corruption after realloc at index %d", i)
			}
		}

		allocator.Free(newPtr)
	})

	t.Run("Statistics", func(t *testing.T) {
		initialStats := allocator.Stats()

		ptrs := make([]unsafe.Pointer, 10)
		for i := range ptrs {
			ptrs[i] = allocator.Alloc(128)
			if ptrs[i] == nil {
				t.Fatalf("Allocation %d failed", i)
			}
		}

		midStats := allocator.Stats()
		if midStats.AllocationCount <= initialStats.AllocationCount {
			t.Error("Allocation count not updated")
		}

		for _, ptr := range ptrs {
			allocator.Free(ptr)
		}

		finalStats := allocator.Stats()
		if finalStats.FreeCount <= midStats.FreeCount {
			t.Error("Free count not updated")
		}
	})
}

// TestArenaAllocator tests the arena allocator implementation
func TestArenaAllocator(t *testing.T) {
	config := defaultConfig()
	allocator, err := NewArenaAllocator(64*1024, config)
	if err != nil {
		t.Fatalf("Failed to create arena allocator: %v", err)
	}

	t.Run("BasicAllocation", func(t *testing.T) {
		ptr := allocator.Alloc(1024)
		if ptr == nil {
			t.Fatal("Allocation failed")
		}

		// Write to memory
		data := (*[1024]byte)(ptr)
		for i := 0; i < 1024; i++ {
			data[i] = byte(i % 256)
		}

		// Verify data
		for i := 0; i < 1024; i++ {
			if data[i] != byte(i%256) {
				t.Errorf("Data corruption at index %d", i)
			}
		}
	})

	t.Run("ExhaustArena", func(t *testing.T) {
		allocator.Reset()

		// Allocate until exhausted
		var ptrs []unsafe.Pointer
		for {
			ptr := allocator.Alloc(1024)
			if ptr == nil {
				break
			}
			ptrs = append(ptrs, ptr)
		}

		if len(ptrs) == 0 {
			t.Error("Should have allocated at least one block")
		}

		// Verify we can't allocate more
		ptr := allocator.Alloc(1)
		if ptr != nil {
			t.Error("Should not be able to allocate from exhausted arena")
		}
	})

	t.Run("Reset", func(t *testing.T) {
		allocator.Reset()

		// Allocate some memory
		ptr1 := allocator.Alloc(1024)
		if ptr1 == nil {
			t.Fatal("Allocation failed")
		}

		usedBefore := allocator.Used()
		if usedBefore == 0 {
			t.Error("Used memory should be greater than 0")
		}

		// Reset arena
		allocator.Reset()

		usedAfter := allocator.Used()
		if usedAfter != 0 {
			t.Error("Used memory should be 0 after reset")
		}

		// Should be able to allocate again
		ptr2 := allocator.Alloc(1024)
		if ptr2 == nil {
			t.Fatal("Allocation failed after reset")
		}
	})

	t.Run("AlignedAllocation", func(t *testing.T) {
		allocator.Reset()

		ptr := allocator.AllocAligned(100, 32)
		if ptr == nil {
			t.Fatal("Aligned allocation failed")
		}

		// Check alignment
		addr := uintptr(ptr)
		if addr%32 != 0 {
			t.Errorf("Memory not aligned to 32 bytes: %x", addr)
		}
	})

	t.Run("SubArena", func(t *testing.T) {
		allocator.Reset()

		subArena, err := allocator.SubArena(8192)
		if err != nil {
			t.Fatalf("Failed to create sub-arena: %v", err)
		}

		// Allocate from sub-arena
		ptr := subArena.Alloc(1024)
		if ptr == nil {
			t.Fatal("Sub-arena allocation failed")
		}

		// Check that parent arena usage increased
		if allocator.Used() == 0 {
			t.Error("Parent arena should show usage")
		}
	})
}

// TestPoolAllocator tests the pool allocator implementation
func TestPoolAllocator(t *testing.T) {
	config := defaultConfig()
	poolSizes := []uintptr{8, 16, 32, 64, 128, 256, 512, 1024}
	allocator, err := NewPoolAllocator(poolSizes, config)
	if err != nil {
		t.Fatalf("Failed to create pool allocator: %v", err)
	}

	t.Run("PoolAllocation", func(t *testing.T) {
		// Allocate sizes that match pools
		for _, size := range poolSizes {
			ptr := allocator.Alloc(size)
			if ptr == nil {
				t.Errorf("Pool allocation failed for size %d", size)
				continue
			}

			// Write to memory
			data := (*[1024]byte)(ptr)[:size:size]
			for i := range data {
				data[i] = byte(i % 256)
			}

			// Verify data
			for i := range data {
				if data[i] != byte(i%256) {
					t.Errorf("Data corruption at index %d for size %d", i, size)
				}
			}

			allocator.Free(ptr)
		}
	})

	t.Run("FallbackAllocation", func(t *testing.T) {
		// Allocate size larger than any pool
		largeSize := uintptr(2048)
		ptr := allocator.Alloc(largeSize)
		if ptr == nil {
			t.Error("Fallback allocation failed")
		}

		allocator.Free(ptr)
	})

	t.Run("PoolReuse", func(t *testing.T) {
		size := uintptr(64)

		// Allocate and free multiple times
		var ptrs []unsafe.Pointer
		for i := 0; i < 10; i++ {
			ptr := allocator.Alloc(size)
			if ptr == nil {
				t.Errorf("Allocation %d failed", i)
				continue
			}
			ptrs = append(ptrs, ptr)
		}

		// Free all
		for _, ptr := range ptrs {
			allocator.Free(ptr)
		}

		// Allocate again - should reuse from pool
		for i := 0; i < 10; i++ {
			ptr := allocator.Alloc(size)
			if ptr == nil {
				t.Errorf("Reallocation %d failed", i)
			}
		}
	})

	t.Run("AddRemovePool", func(t *testing.T) {
		// Add new pool
		newSize := uintptr(2048)
		err := allocator.AddPool(newSize)
		if err != nil {
			t.Errorf("Failed to add pool: %v", err)
		}

		// Allocate from new pool
		ptr := allocator.Alloc(newSize)
		if ptr == nil {
			t.Error("Allocation from new pool failed")
		}
		allocator.Free(ptr)

		// Remove pool
		err = allocator.RemovePool(newSize)
		if err != nil {
			t.Errorf("Failed to remove pool: %v", err)
		}
	})
}

// TestRuntime tests the runtime memory management
func TestRuntime(t *testing.T) {
	config := defaultConfig()
	allocator := NewSystemAllocator(config)

	err := InitializeRuntime(allocator, WithGC(false))
	if err != nil {
		t.Fatalf("Failed to initialize runtime: %v", err)
	}
	defer ShutdownRuntime()

	t.Run("ObjectAllocation", func(t *testing.T) {
		ptr := RuntimeAlloc(1024)
		if ptr == nil {
			t.Fatal("Runtime allocation failed")
		}

		// Write to memory
		data := (*[1024]byte)(ptr)
		for i := 0; i < 1024; i++ {
			data[i] = byte(i % 256)
		}

		// Verify data
		for i := 0; i < 1024; i++ {
			if data[i] != byte(i%256) {
				t.Errorf("Data corruption at index %d", i)
			}
		}

		RuntimeFree(ptr)
	})

	t.Run("ArrayAllocation", func(t *testing.T) {
		elementSize := uintptr(8)
		count := 100

		ptr := RuntimeAllocArray(elementSize, count)
		if ptr == nil {
			t.Fatal("Array allocation failed")
		}

		// Access array elements
		array := (*[100]uint64)(ptr)
		for i := 0; i < count; i++ {
			array[i] = uint64(i * 2)
		}

		// Verify values
		for i := 0; i < count; i++ {
			if array[i] != uint64(i*2) {
				t.Errorf("Array element %d corrupted", i)
			}
		}

		RuntimeFree(ptr)
	})

	t.Run("SliceAllocation", func(t *testing.T) {
		elementSize := uintptr(4)
		len := 50
		cap := 100

		header := RuntimeAllocSlice(elementSize, len, cap)
		if header == nil {
			t.Fatal("Slice allocation failed")
		}

		if header.Len != len {
			t.Errorf("Slice length mismatch: got %d, want %d", header.Len, len)
		}

		if header.Cap != cap {
			t.Errorf("Slice capacity mismatch: got %d, want %d", header.Cap, cap)
		}

		if header.Data == nil {
			t.Error("Slice data is nil")
		}

		// Access slice elements
		slice := (*[100]uint32)(header.Data)
		for i := 0; i < len; i++ {
			slice[i] = uint32(i * 3)
		}

		// Verify values
		for i := 0; i < len; i++ {
			if slice[i] != uint32(i*3) {
				t.Errorf("Slice element %d corrupted", i)
			}
		}

		RuntimeFreeSlice(header)
	})

	t.Run("StringAllocation", func(t *testing.T) {
		testString := "Hello, Orizon Runtime!"

		ptr := RuntimeAllocString(testString)
		if ptr == nil {
			t.Fatal("String allocation failed")
		}

		// Verify string content
		data := (*[100]byte)(ptr)[:len(testString):len(testString)]
		resultString := string(data)

		if resultString != testString {
			t.Errorf("String content mismatch: got %q, want %q", resultString, testString)
		}

		// Allocate same string again (should hit string pool)
		ptr2 := RuntimeAllocString(testString)
		if ptr2 == nil {
			t.Fatal("Second string allocation failed")
		}

		// Check string pool stats
		stats := GetRuntimeStats()
		if stats.StringPool.StringCount == 0 {
			t.Error("String pool should contain at least one string")
		}
	})

	t.Run("Statistics", func(t *testing.T) {
		stats := GetRuntimeStats()

		if stats.Allocator.AllocationCount == 0 {
			t.Error("Should have some allocations")
		}

		// Allocate something to change stats
		ptr := RuntimeAlloc(256)
		RuntimeFree(ptr)

		newStats := GetRuntimeStats()
		if newStats.Allocator.AllocationCount <= stats.Allocator.AllocationCount {
			t.Error("Allocation count should have increased")
		}
	})
}

// TestAlignment tests memory alignment
func TestAlignment(t *testing.T) {
	config := defaultConfig()
	config.AlignmentSize = 16

	allocator := NewSystemAllocator(config)

	t.Run("AlignmentCheck", func(t *testing.T) {
		sizes := []uintptr{1, 7, 15, 16, 17, 31, 32, 63, 64}

		for _, size := range sizes {
			ptr := allocator.Alloc(size)
			if ptr == nil {
				t.Errorf("Allocation failed for size %d", size)
				continue
			}

			addr := uintptr(ptr)
			if addr%16 != 0 {
				t.Errorf("Memory not aligned for size %d: address %x", size, addr)
			}

			allocator.Free(ptr)
		}
	})
}

// TestMemoryLimits tests memory limits
func TestMemoryLimits(t *testing.T) {
	config := defaultConfig()
	config.MemoryLimit = 4096 // 4KB limit

	allocator := NewSystemAllocator(config)

	t.Run("MemoryLimit", func(t *testing.T) {
		// Allocate within limit
		ptr1 := allocator.Alloc(2048)
		if ptr1 == nil {
			t.Fatal("Allocation within limit failed")
		}

		// Try to allocate beyond limit
		ptr2 := allocator.Alloc(3072)
		if ptr2 != nil {
			t.Error("Allocation beyond limit should fail")
			allocator.Free(ptr2)
		}

		allocator.Free(ptr1)

		// Should be able to allocate again after freeing
		ptr3 := allocator.Alloc(3072)
		if ptr3 == nil {
			t.Error("Allocation should succeed after freeing memory")
		}

		allocator.Free(ptr3)
	})
}

// TestLeakDetection tests memory leak detection
func TestLeakDetection(t *testing.T) {
	config := defaultConfig()
	config.EnableLeakCheck = true
	config.EnableTracking = true

	allocator := NewSystemAllocator(config)

	t.Run("LeakDetection", func(t *testing.T) {
		// Allocate without freeing
		ptr1 := allocator.Alloc(1024)
		ptr2 := allocator.Alloc(2048)

		if ptr1 == nil || ptr2 == nil {
			t.Fatal("Allocations failed")
		}

		// Check for leaks
		leaks := allocator.CheckLeaks()
		if len(leaks) != 2 {
			t.Errorf("Expected 2 leaks, got %d", len(leaks))
		}

		// Free one allocation
		allocator.Free(ptr1)

		leaks = allocator.CheckLeaks()
		if len(leaks) != 1 {
			t.Errorf("Expected 1 leak after freeing, got %d", len(leaks))
		}

		// Free remaining allocation
		allocator.Free(ptr2)

		leaks = allocator.CheckLeaks()
		if len(leaks) != 0 {
			t.Errorf("Expected 0 leaks after freeing all, got %d", len(leaks))
		}
	})
}

// TestConcurrency tests thread safety
func TestConcurrency(t *testing.T) {
	config := defaultConfig()
	allocator := NewSystemAllocator(config)

	t.Run("ConcurrentAllocations", func(t *testing.T) {
		const numGoroutines = 10
		const allocsPerGoroutine = 100

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()

				var ptrs []unsafe.Pointer

				// Allocate
				for j := 0; j < allocsPerGoroutine; j++ {
					ptr := allocator.Alloc(256)
					if ptr != nil {
						ptrs = append(ptrs, ptr)
					}
				}

				// Free
				for _, ptr := range ptrs {
					allocator.Free(ptr)
				}
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check stats
		stats := allocator.Stats()
		expectedAllocs := uint64(numGoroutines * allocsPerGoroutine)

		if stats.AllocationCount < expectedAllocs {
			t.Errorf("Expected at least %d allocations, got %d",
				expectedAllocs, stats.AllocationCount)
		}
	})
}

// BenchmarkAllocators benchmarks different allocator types
func BenchmarkSystemAllocator(b *testing.B) {
	config := defaultConfig()
	config.EnableTracking = false // Disable for performance
	allocator := NewSystemAllocator(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ptr := allocator.Alloc(256)
			if ptr != nil {
				allocator.Free(ptr)
			}
		}
	})
}

func BenchmarkArenaAllocator(b *testing.B) {
	config := defaultConfig()
	allocator, _ := NewArenaAllocator(1024*1024, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%1000 == 0 {
			allocator.Reset() // Reset periodically to avoid exhaustion
		}
		allocator.Alloc(256)
	}
}

func BenchmarkPoolAllocator(b *testing.B) {
	config := defaultConfig()
	poolSizes := []uintptr{64, 128, 256, 512, 1024}
	allocator, _ := NewPoolAllocator(poolSizes, config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ptr := allocator.Alloc(256)
			if ptr != nil {
				allocator.Free(ptr)
			}
		}
	})
}

// TestInitialization tests allocator initialization
func TestInitialization(t *testing.T) {
	t.Run("SystemAllocatorInit", func(t *testing.T) {
		err := Initialize(SystemAllocatorKind)
		if err != nil {
			t.Errorf("System allocator initialization failed: %v", err)
		}

		if GlobalAllocator == nil {
			t.Error("Global allocator not set")
		}
	})

	t.Run("ArenaAllocatorInit", func(t *testing.T) {
		err := Initialize(ArenaAllocatorKind, WithArenaSize(32*1024))
		if err != nil {
			t.Errorf("Arena allocator initialization failed: %v", err)
		}

		if GlobalAllocator == nil {
			t.Error("Global allocator not set")
		}
	})

	t.Run("PoolAllocatorInit", func(t *testing.T) {
		poolSizes := []uintptr{8, 16, 32, 64, 128}
		err := Initialize(PoolAllocatorKind, WithPoolSizes(poolSizes))
		if err != nil {
			t.Errorf("Pool allocator initialization failed: %v", err)
		}

		if GlobalAllocator == nil {
			t.Error("Global allocator not set")
		}
	})

	t.Run("InvalidAllocatorKind", func(t *testing.T) {
		err := Initialize(AllocatorKind(999))
		if err == nil {
			t.Error("Invalid allocator kind should return error")
		}
	})
}
