package types

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestOrizonSlice_ComprehensiveSafety provides exhaustive safety validation.
// covering all possible attack vectors and edge cases.
func TestOrizonSlice_ComprehensiveSafety(t *testing.T) {
	tests := []struct {
		setupSlice  func() *OrizonSlice
		testFunc    func(*testing.T, *OrizonSlice)
		name        string
		expectedMsg string
		expectPanic bool
	}{
		// === BOUNDARY VALUE TESTS ===.
		{
			name: "access at zero index in single element slice",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 1)
				data[0] = 42

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   1,
					capacity: 1,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				ptr := slice.Get(0)
				if ptr == nil {
					t.Error("Expected non-nil pointer for valid access")
				}
				if *(*int32)(ptr) != 42 {
					t.Errorf("Expected 42, got %d", *(*int32)(ptr))
				}
			},
		},
		{
			name: "access at maximum valid index",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 1000)
				data[999] = 999

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   1000,
					capacity: 1000,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				ptr := slice.Get(999)
				if ptr == nil {
					t.Error("Expected non-nil pointer for valid access")
				}
				if *(*int32)(ptr) != 999 {
					t.Errorf("Expected 999, got %d", *(*int32)(ptr))
				}
			},
		},
		{
			name: "zero-length slice access should panic",
			setupSlice: func() *OrizonSlice {
				return &OrizonSlice{
					data:     nil,
					length:   0,
					capacity: 0,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Get(0) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Index out of bounds",
		},
		{
			name: "access one past maximum index should panic",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Get(10) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Index out of bounds",
		},

		// === INTEGER OVERFLOW TESTS ===.
		{
			name: "extremely large index causing overflow",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				// Use maximum uintptr value to trigger overflow.
				slice.Get(^uintptr(0)) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Index out of bounds",
		},
		{
			name: "index near overflow boundary",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: ^uintptr(0)}, // Max element size
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Get(2) // Should trigger overflow check
			},
			expectPanic: true,
			expectedMsg: "Offset calculation would overflow",
		},

		// === MEMORY CORRUPTION PROTECTION TESTS ===.
		{
			name: "corrupted slice structure - invalid typeInfo",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: nil, // Corrupted
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Get(5) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Invalid slice: nil typeInfo",
		},
		{
			name: "corrupted slice structure - zero element size",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 0}, // Corrupted
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Get(5) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Invalid element size: 0",
		},
		{
			name: "corrupted slice structure - null data pointer",
			setupSlice: func() *OrizonSlice {
				return &OrizonSlice{
					data:     nil, // Corrupted
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Get(5) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Null slice data pointer",
		},

		// === MISALIGNED MEMORY TESTS ===.
		{
			name: "misaligned element size (not power of 2)",
			setupSlice: func() *OrizonSlice {
				data := make([]byte, 100)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   25,
					capacity: 25,
					typeInfo: &TypeInfo{Size: 3}, // Odd size
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				ptr := slice.Get(24)
				if ptr == nil {
					t.Error("Expected non-nil pointer for valid access")
				}
				// Verify pointer arithmetic worked correctly with odd size.
				dataPtr := unsafe.Pointer((*[100]byte)(slice.data))
				expected := unsafe.Add(dataPtr, 24*3)
				if ptr != expected {
					t.Error("Pointer arithmetic failed with odd element size")
				}
			},
		},
		{
			name: "very large element size",
			setupSlice: func() *OrizonSlice {
				// Simulate large struct.
				largeElementSize := uintptr(1024)
				data := make([]byte, int(largeElementSize*5))

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   5,
					capacity: 5,
					typeInfo: &TypeInfo{Size: largeElementSize},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				ptr := slice.Get(4)
				if ptr == nil {
					t.Error("Expected non-nil pointer for valid access")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := tt.setupSlice()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						panicMsg := fmt.Sprintf("%v", r)
						if tt.expectedMsg != "" && !contains(panicMsg, tt.expectedMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tt.expectedMsg, panicMsg)
						}
					} else {
						t.Error("expected panic but got none")
					}
				}()
				tt.testFunc(t, slice)
			} else {
				tt.testFunc(t, slice)
			}
		})
	}
}

// TestOrizonSlice_SetOperationComprehensive provides exhaustive testing for Set operations.
func TestOrizonSlice_SetOperationComprehensive(t *testing.T) {
	tests := []struct {
		setupSlice  func() *OrizonSlice
		testFunc    func(*testing.T, *OrizonSlice)
		name        string
		expectedMsg string
		expectPanic bool
	}{
		// === VALID SET OPERATIONS ===.
		{
			name: "set operation with valid parameters",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				value := int32(42)
				slice.Set(5, unsafe.Pointer(&value))

				// Verify the value was set correctly.
				ptr := slice.Get(5)
				if *(*int32)(ptr) != 42 {
					t.Errorf("Expected 42, got %d", *(*int32)(ptr))
				}
			},
		},
		{
			name: "set at boundary indices",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				value1 := int32(100)
				value2 := int32(200)

				// Set at first and last valid indices.
				slice.Set(0, unsafe.Pointer(&value1))
				slice.Set(9, unsafe.Pointer(&value2))

				// Verify values.
				if *(*int32)(slice.Get(0)) != 100 {
					t.Error("First element not set correctly")
				}
				if *(*int32)(slice.Get(9)) != 200 {
					t.Error("Last element not set correctly")
				}
			},
		},

		// === INVALID SET OPERATIONS ===.
		{
			name: "set with nil value pointer",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				slice.Set(5, nil) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Null value pointer",
		},
		{
			name: "set with index out of bounds",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				value := int32(42)
				slice.Set(10, unsafe.Pointer(&value)) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Index out of bounds",
		},
		{
			name: "set with corrupted slice (nil data)",
			setupSlice: func() *OrizonSlice {
				return &OrizonSlice{
					data:     nil,
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				value := int32(42)
				slice.Set(5, unsafe.Pointer(&value)) // Should panic
			},
			expectPanic: true,
			expectedMsg: "Null slice data pointer",
		},
		{
			name: "set with overflow-inducing index",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: ^uintptr(0)}, // Max size
				}
			},
			testFunc: func(t *testing.T, slice *OrizonSlice) {
				value := int32(42)
				slice.Set(2, unsafe.Pointer(&value)) // Should panic due to overflow
			},
			expectPanic: true,
			expectedMsg: "Offset calculation would overflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := tt.setupSlice()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						panicMsg := fmt.Sprintf("%v", r)
						if tt.expectedMsg != "" && !contains(panicMsg, tt.expectedMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tt.expectedMsg, panicMsg)
						}
					} else {
						t.Error("expected panic but got none")
					}
				}()
				tt.testFunc(t, slice)
			} else {
				tt.testFunc(t, slice)
			}
		})
	}
}

// TestOrizonSlice_SubsliceOperationSafety tests the Sub method comprehensively.
func TestOrizonSlice_SubsliceOperationSafety(t *testing.T) {
	tests := []struct {
		setupSlice  func() *OrizonSlice
		validateSub func(*testing.T, *OrizonSlice)
		name        string
		expectedMsg string
		start       uintptr
		end         uintptr
		expectPanic bool
	}{
		{
			name: "valid subslice operation",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)
				for i := range data {
					data[i] = int32(i)
				}

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			start: 2,
			end:   7,
			validateSub: func(t *testing.T, sub *OrizonSlice) {
				if sub.Len() != 5 {
					t.Errorf("Expected length 5, got %d", sub.Len())
				}
				// Verify first element of subslice.
				if *(*int32)(sub.Get(0)) != 2 {
					t.Errorf("Expected first element 2, got %d", *(*int32)(sub.Get(0)))
				}
				// Verify last element of subslice.
				if *(*int32)(sub.Get(4)) != 6 {
					t.Errorf("Expected last element 6, got %d", *(*int32)(sub.Get(4)))
				}
			},
		},
		{
			name: "empty subslice (start == end)",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			start: 5,
			end:   5,
			validateSub: func(t *testing.T, sub *OrizonSlice) {
				if sub.Len() != 0 {
					t.Errorf("Expected length 0, got %d", sub.Len())
				}
				if !sub.IsEmpty() {
					t.Error("Expected empty subslice")
				}
			},
		},
		{
			name: "full slice subslice",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			start: 0,
			end:   10,
			validateSub: func(t *testing.T, sub *OrizonSlice) {
				if sub.Len() != 10 {
					t.Errorf("Expected length 10, got %d", sub.Len())
				}
			},
		},
		{
			name: "invalid subslice - start > end",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			start:       7,
			end:         5,
			expectPanic: true,
			expectedMsg: "Invalid slice bounds",
		},
		{
			name: "invalid subslice - end > length",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			start:       0,
			end:         15,
			expectPanic: true,
			expectedMsg: "Invalid slice bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := tt.setupSlice()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						panicMsg := fmt.Sprintf("%v", r)
						if tt.expectedMsg != "" && !contains(panicMsg, tt.expectedMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tt.expectedMsg, panicMsg)
						}
					} else {
						t.Error("expected panic but got none")
					}
				}()
				slice.Sub(tt.start, tt.end)
			} else {
				sub := slice.Sub(tt.start, tt.end)
				if tt.validateSub != nil {
					tt.validateSub(t, sub)
				}
			}
		})
	}
}

// TestOrizonSlice_ConcurrencyStressTest provides intensive concurrent access testing.
func TestOrizonSlice_ConcurrencyStressTest(t *testing.T) {
	// Create a large slice for stress testing.
	data := make([]int64, 100000)
	for i := range data {
		data[i] = int64(i)
	}

	slice := &OrizonSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   100000,
		capacity: 100000,
		typeInfo: &TypeInfo{Size: 8},
	}

	const numGoroutines = 100

	const operationsPerGoroutine = 10000

	// Test concurrent reads.
	t.Run("concurrent_reads", func(t *testing.T) {
		var wg sync.WaitGroup

		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)

			go func(goroutineID int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("goroutine %d panicked: %v", goroutineID, r)
					}
				}()

				for j := 0; j < operationsPerGoroutine; j++ {
					index := uintptr((goroutineID*operationsPerGoroutine + j) % 100000)

					ptr := slice.Get(index)
					if ptr == nil {
						errors <- fmt.Errorf("goroutine %d got nil pointer at index %d", goroutineID, index)

						return
					}

					value := *(*int64)(ptr)
					if value != int64(index) {
						errors <- fmt.Errorf("goroutine %d got incorrect value %d at index %d", goroutineID, value, index)

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

	// Test mixed concurrent reads and writes.
	t.Run("concurrent_reads_and_writes", func(t *testing.T) {
		var wg sync.WaitGroup

		errors := make(chan error, numGoroutines)

		// Create writable copy for this test.
		writeData := make([]int64, 100000)
		copy(writeData, data)

		writeSlice := &OrizonSlice{
			data:     unsafe.Pointer(&writeData[0]),
			length:   100000,
			capacity: 100000,
			typeInfo: &TypeInfo{Size: 8},
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)

			go func(goroutineID int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("goroutine %d panicked: %v", goroutineID, r)
					}
				}()

				for j := 0; j < operationsPerGoroutine/10; j++ { // Fewer operations for mixed test
					index := uintptr((goroutineID*1000 + j) % 100000)

					// Alternate between read and write.
					if j%2 == 0 {
						// Read operation.
						ptr := writeSlice.Get(index)
						if ptr == nil {
							errors <- fmt.Errorf("goroutine %d got nil pointer at index %d", goroutineID, index)

							return
						}
					} else {
						// Write operation.
						newValue := int64(goroutineID*1000000 + j)
						writeSlice.Set(index, unsafe.Pointer(&newValue))
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
}

// TestOrizonSlice_MemoryIntegrityChecks validates memory integrity under stress.
func TestOrizonSlice_MemoryIntegrityChecks(t *testing.T) {
	t.Run("buffer_boundary_integrity", func(t *testing.T) {
		// Create slice with guard pages simulation.
		const sliceSize = 1000

		guardPattern := byte(0xDE)
		guardSize := 100

		// Allocate memory with guard regions.
		totalSize := guardSize + sliceSize*4 + guardSize // guards + data + guards
		memory := make([]byte, totalSize)

		// Fill guard regions with pattern.
		for i := 0; i < guardSize; i++ {
			memory[i] = guardPattern
			memory[totalSize-1-i] = guardPattern
		}

		// Create slice pointing to middle region.
		dataStart := guardSize
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&memory[dataStart]),
			length:   sliceSize,
			capacity: sliceSize,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Perform operations.
		for i := uintptr(0); i < sliceSize; i++ {
			value := int32(i * 2)
			slice.Set(i, unsafe.Pointer(&value))
		}

		// Verify all operations worked.
		for i := uintptr(0); i < sliceSize; i++ {
			ptr := slice.Get(i)
			if *(*int32)(ptr) != int32(i*2) {
				t.Errorf("Value mismatch at index %d", i)
			}
		}

		// Verify guard regions are intact.
		for i := 0; i < guardSize; i++ {
			if memory[i] != guardPattern {
				t.Errorf("Front guard corrupted at offset %d", i)
			}

			if memory[totalSize-1-i] != guardPattern {
				t.Errorf("Back guard corrupted at offset %d", i)
			}
		}
	})
}

// TestOrizonSlice_ResourceLeakPrevention ensures no resource leaks.
func TestOrizonSlice_ResourceLeakPrevention(t *testing.T) {
	// Monitor goroutine count.
	initialGoroutines := runtime.NumGoroutine()

	t.Run("no_goroutine_leaks", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				// Create and use slice.
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
					ptr := slice.Get(j)
					_ = *(*int32)(ptr)
				}
			}()
		}

		wg.Wait()

		// Allow garbage collection.
		runtime.GC()
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines+5 { // Allow some tolerance
			t.Errorf("Potential goroutine leak: started with %d, ended with %d",
				initialGoroutines, finalGoroutines)
		}
	})
}

// TestOrizonSlice_PerformanceRegression ensures performance doesn't degrade.
func TestOrizonSlice_PerformanceRegression(t *testing.T) {
	data := make([]int32, 100000)
	slice := &OrizonSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   100000,
		capacity: 100000,
		typeInfo: &TypeInfo{Size: 4},
	}

	t.Run("sequential_access_performance", func(t *testing.T) {
		start := time.Now()
		iterations := 1000000

		for i := 0; i < iterations; i++ {
			index := uintptr(i % 100000)
			ptr := slice.Get(index)
			_ = *(*int32)(ptr) // Force memory access
		}

		duration := time.Since(start)
		nsPerOp := duration.Nanoseconds() / int64(iterations)

		// Ensure performance is within acceptable bounds (should be < 100ns for modern systems).
		// This accounts for various factors like CPU speed, system load, and Go runtime overhead.
		if nsPerOp > 100 {
			t.Errorf("Performance regression: %d ns/op (expected < 100 ns/op)", nsPerOp)
		}

		t.Logf("Sequential access performance: %d ns/op", nsPerOp)
	})

	t.Run("random_access_performance", func(t *testing.T) {
		// Pre-generate random indices to avoid affecting timing.
		indices := make([]uintptr, 100000)
		for i := range indices {
			indices[i] = uintptr(i * 7919 % 100000) // Use prime for better distribution
		}

		start := time.Now()

		for i := 0; i < 100000; i++ {
			ptr := slice.Get(indices[i])
			_ = *(*int32)(ptr) // Force memory access
		}

		duration := time.Since(start)
		nsPerOp := duration.Nanoseconds() / 100000

		// Random access should still be fast (< 200ns for modern systems).
		// Random access is naturally slower than sequential due to cache misses.
		if nsPerOp > 200 {
			t.Errorf("Random access performance regression: %d ns/op (expected < 200 ns/op)", nsPerOp)
		}

		t.Logf("Random access performance: %d ns/op", nsPerOp)
	})
}

// TestOrizonSlice_SecurityValidation tests against security vulnerabilities.
func TestOrizonSlice_SecurityValidation(t *testing.T) {
	t.Run("no_buffer_overflow_attacks", func(t *testing.T) {
		data := make([]int32, 10)
		slice := &OrizonSlice{
			data:     unsafe.Pointer(&data[0]),
			length:   10,
			capacity: 10,
			typeInfo: &TypeInfo{Size: 4},
		}

		// Attempt various buffer overflow attacks.
		attackIndices := []uintptr{
			10, 11, 100, 1000, 10000,
			^uintptr(0), ^uintptr(0) - 1, ^uintptr(0) >> 1,
		}

		for _, attackIndex := range attackIndices {
			t.Run(fmt.Sprintf("attack_index_%d", attackIndex), func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for malicious index %d, but got none", attackIndex)
					}
				}()

				slice.Get(attackIndex)
			})
		}
	})

	t.Run("no_integer_overflow_exploits", func(t *testing.T) {
		// Test with various combinations that could cause integer overflow.
		testCases := []struct {
			length      uintptr
			elementSize uintptr
			index       uintptr
		}{
			{1, ^uintptr(0), 1},               // Max element size
			{^uintptr(0), 1, ^uintptr(0) - 1}, // Max length
			{1000, ^uintptr(0) / 500, 600},    // Overflow in multiplication
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("overflow_test_%d", i), func(t *testing.T) {
				data := make([]byte, 100) // Small allocation
				slice := &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   tc.length,
					capacity: tc.length,
					typeInfo: &TypeInfo{Size: tc.elementSize},
				}

				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for integer overflow scenario")
					}
				}()

				slice.Get(tc.index)
			})
		}
	})
}

// Helper function for string contains check using standard library.
func checkContains(s, substr string) bool {
	return strings.Contains(s, substr)
}
