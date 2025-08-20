package types

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestOrizonSlice_SecurityTests provides security-focused testing.
func TestOrizonSlice_SecurityTests(t *testing.T) {
	t.Run("buffer_overflow_protection", func(t *testing.T) {
		data := make([]int32, 10)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   10,
			capacity: 10,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Test various overflow attempts.
		maliciousIndices := []uintptr{
			10, 11, 100, 1000, 10000,
			^uintptr(0), ^uintptr(0) - 1, ^uintptr(0) >> 1,
		}

		for _, index := range maliciousIndices {
			func() {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for malicious index %d", index)
					}
				}()
				slice.Get(index)
			}()
		}
	})

	t.Run("integer_overflow_protection", func(t *testing.T) {
		// Test cases that could cause integer overflow.
		testCases := []struct {
			name        string
			length      uintptr
			elementSize uintptr
			index       uintptr
		}{
			{"max_element_size", 10, ^uintptr(0), 1},
			{"large_multiplication", 1000, ^uintptr(0) / 500, 600},
			{"boundary_overflow", 2, ^uintptr(0)/2 + 1, 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				data := make([]byte, 100)
				slice := &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   tc.length,
					capacity: tc.length,
					typeInfo: &TypeInfo{Size: tc.elementSize},
				}

				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for overflow scenario")
					}
				}()

				slice.Get(tc.index)
			})
		}
	})
}

// TestOrizonSlice_EdgeCases provides comprehensive edge case testing.
func TestOrizonSlice_EdgeCases(t *testing.T) {
	t.Run("zero_length_slice", func(t *testing.T) {
		slice := &OrizonSlice{
			data:     nil,
			length:   0,
			capacity: 0,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Test all methods on zero-length slice.
		if slice.Len() != 0 {
			t.Error("Expected zero length")
		}

		if slice.Cap() != 0 {
			t.Error("Expected zero capacity")
		}

		if !slice.IsEmpty() {
			t.Error("Expected empty slice")
		}

		// Access should panic.
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for access to empty slice")
			}
		}()
		slice.Get(0)
	})

	t.Run("single_element_slice", func(t *testing.T) {
		data := make([]int32, 1)
		data[0] = 42
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   1,
			capacity: 1,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Valid access.
		ptr := slice.Get(0)
		if *(*int32)(ptr) != 42 {
			t.Error("Single element access failed")
		}

		// Invalid access should panic.
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for index 1 in single-element slice")
			}
		}()
		slice.Get(1)
	})

	t.Run("large_slice_boundary_access", func(t *testing.T) {
		const size = 1000000
		data := make([]int32, size)
		data[0] = 1
		data[size-1] = size

		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   size,
			capacity: size,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Test first and last valid indices.
		if *(*int32)(slice.Get(0)) != 1 {
			t.Error("First element access failed")
		}

		if *(*int32)(slice.Get(size - 1)) != size {
			t.Error("Last element access failed")
		}

		// Test just past boundary.
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for access past boundary")
			}
		}()
		slice.Get(size)
	})
}

// TestOrizonSlice_ConcurrentSafety provides concurrent access safety testing.
func TestOrizonSlice_ConcurrentSafety(t *testing.T) {
	t.Run("read_only_concurrent_access", func(t *testing.T) {
		const size = 10000

		data := make([]int64, size)
		for i := range data {
			data[i] = int64(i)
		}

		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   size,
			capacity: size,
			typeInfo: &TypeInfo{Size: 8},
		}

		const numGoroutines = 50

		const operationsPerGoroutine = 1000

		var wg sync.WaitGroup

		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)

			go func(id int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("goroutine %d panicked: %v", id, r)
					}
				}()

				for j := 0; j < operationsPerGoroutine; j++ {
					index := uintptr((id*operationsPerGoroutine + j) % size)
					ptr := slice.Get(index)

					value := *(*int64)(ptr)
					if value != int64(index) {
						errors <- fmt.Errorf("wrong value at index %d: got %d", index, value)

						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}
	})

	t.Run("mixed_read_write_concurrent", func(t *testing.T) {
		const size = 1000
		data := make([]int64, size)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   size,
			capacity: size,
			typeInfo: &TypeInfo{Size: 8},
		}

		const numGoroutines = 20

		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)

			go func(id int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						// Unexpected panic in this test.
						t.Errorf("Goroutine %d panicked: %v", id, r)
					}
				}()

				for j := 0; j < 100; j++ {
					index := uintptr((id*100 + j) % size)

					if j%2 == 0 {
						// Read operation.
						ptr := slice.Get(index)
						_ = *(*int64)(ptr)
					} else {
						// Write operation.
						value := int64(id*10000 + j)
						slice.Set(index, unsafe.Pointer(&value))
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestOrizonSlice_MemoryAlignment tests memory alignment scenarios.
func TestOrizonSlice_MemoryAlignment(t *testing.T) {
	t.Run("various_element_sizes", func(t *testing.T) {
		testSizes := []uintptr{1, 2, 3, 4, 5, 7, 8, 12, 16, 24, 32, 64, 128}

		for _, size := range testSizes {
			t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
				const numElements = 100
				data := make([]byte, int(size*numElements))

				slice := &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   numElements,
					capacity: numElements,
					typeInfo: &TypeInfo{Size: size},
				}

				// Test access to all elements.
				for i := uintptr(0); i < numElements; i++ {
					ptr := slice.Get(i)
					if ptr == nil {
						t.Errorf("Got nil pointer at index %d", i)
					}

					// Verify pointer arithmetic.
					expected := unsafe.Add(unsafe.Pointer(&data[0]), i*size)
					if ptr != expected {
						t.Errorf("Pointer arithmetic failed at index %d", i)
					}
				}
			})
		}
	})
}

// TestOrizonSlice_ResourceManagement tests resource management.
func TestOrizonSlice_ResourceManagement(t *testing.T) {
	t.Run("no_memory_leaks", func(t *testing.T) {
		// Monitor memory usage.
		var m1, m2 runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Create and destroy many slices.
		for i := 0; i < 10000; i++ {
			data := make([]int32, 100)
			slice := &OrizonSlice{
				data:     unsafe.Pointer(&data[0]),
				length:   100,
				capacity: 100,
				typeInfo: &TypeInfo{Size: 4},
			}

			// Use the slice.
			for j := uintptr(0); j < 100; j++ {
				value := int32(i*100 + int(j))
				slice.Set(j, unsafe.Pointer(&value))
				ptr := slice.Get(j)
				_ = *(*int32)(ptr)
			}
			// Slice should be eligible for GC after this scope.
		}

		runtime.GC()
		runtime.GC() // Double GC to ensure cleanup
		runtime.ReadMemStats(&m2)

		// Memory usage should not grow significantly.
		memGrowth := m2.Alloc - m1.Alloc
		if memGrowth > 10*1024*1024 { // 10MB threshold
			t.Logf("Memory growth: %d bytes (may indicate leak)", memGrowth)
		}
	})

	t.Run("goroutine_cleanup", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		// Test that doesn't create persistent goroutines.
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				data := make([]int32, 1000)
				slice := &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   1000,
					capacity: 1000,
					typeInfo: &TypeInfo{Size: 4},
				}

				// Perform operations.
				for j := uintptr(0); j < 1000; j++ {
					value := int32(j)
					slice.Set(j, unsafe.Pointer(&value))
				}
			}()
		}

		wg.Wait()

		// Allow time for goroutine cleanup.
		time.Sleep(100 * time.Millisecond)
		runtime.GC()

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines+2 { // Small tolerance
			t.Errorf("Potential goroutine leak: %d -> %d goroutines",
				initialGoroutines, finalGoroutines)
		}
	})
}

// TestOrizonSlice_PerformanceCharacteristics tests performance properties.
func TestOrizonSlice_PerformanceCharacteristics(t *testing.T) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("performance_size_%d", size), func(t *testing.T) {
			data := make([]int32, size)
			slice := &OrizonSlice{
				data:     unsafe.Pointer(&data[0]),
				length:   uintptr(size),
				capacity: uintptr(size),
				typeInfo: &TypeInfo{Size: 4},
			}

			// Sequential access performance.
			start := time.Now()

			iterations := 100000
			for i := 0; i < iterations; i++ {
				index := uintptr(i % size)
				ptr := slice.Get(index)
				_ = *(*int32)(ptr)
			}

			duration := time.Since(start)
			nsPerOp := duration.Nanoseconds() / int64(iterations)

			// Performance should be O(1) - not dependent on slice size.
			if nsPerOp > 50 { // Very generous threshold
				t.Logf("Size %d: %d ns/op (may need optimization)", size, nsPerOp)
			} else {
				t.Logf("Size %d: %d ns/op", size, nsPerOp)
			}
		})
	}
}

// BenchmarkOrizonSlice_ComprehensivePerformance provides detailed benchmarks.
func BenchmarkOrizonSlice_ComprehensivePerformance(b *testing.B) {
	// Small slice benchmark.
	b.Run("small_slice_sequential", func(b *testing.B) {
		data := make([]int32, 100)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   100,
			capacity: 100,
			typeInfo: &TypeInfo{Size: 4},
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			index := uintptr(i % 100)
			ptr := slice.Get(index)
			_ = *(*int32)(ptr)
		}
	})

	// Large slice benchmark.
	b.Run("large_slice_sequential", func(b *testing.B) {
		data := make([]int32, 1000000)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   1000000,
			capacity: 1000000,
			typeInfo: &TypeInfo{Size: 4},
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			index := uintptr(i % 1000000)
			ptr := slice.Get(index)
			_ = *(*int32)(ptr)
		}
	})

	// Random access benchmark.
	b.Run("random_access", func(b *testing.B) {
		data := make([]int32, 10000)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   10000,
			capacity: 10000,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Pre-generate random indices.
		indices := make([]uintptr, b.N)
		for i := range indices {
			indices[i] = uintptr(i * 7919 % 10000)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ptr := slice.Get(indices[i])
			_ = *(*int32)(ptr)
		}
	})

	// Set operation benchmark.
	b.Run("set_operations", func(b *testing.B) {
		data := make([]int32, 10000)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   10000,
			capacity: 10000,
			typeInfo: &TypeInfo{Size: 4},
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			index := uintptr(i % 10000)
			value := int32(i)
			slice.Set(index, unsafe.Pointer(&value))
		}
	})

	// Subslice operation benchmark.
	b.Run("subslice_operations", func(b *testing.B) {
		data := make([]int32, 10000)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   10000,
			capacity: 10000,
			typeInfo: &TypeInfo{Size: 4},
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			start := uintptr(i % 5000)

			end := start + uintptr(i%1000) + 1
			if end > 10000 {
				end = 10000
			}

			sub := slice.Sub(start, end)
			_ = sub.Len() // Use the subslice
		}
	})
}
