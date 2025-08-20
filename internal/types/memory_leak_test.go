package types

import (
	"fmt"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/allocator"
)

func TestMemoryLeakDetection(t *testing.T) {
	// Initialize test allocator.
	config := &allocator.Config{
		ArenaSize:      1024 * 1024, // 1MB
		PoolSizes:      []uintptr{64, 256, 1024, 4096},
		AlignmentSize:  8,                  // 8-byte alignment
		MemoryLimit:    1024 * 1024 * 1024, // 1GB limit
		EnableTracking: true,
	}
	alloc := allocator.NewSystemAllocator(config)

	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}

	defer ShutdownCoreTypes()

	// Force GC to establish baseline.
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats

	runtime.ReadMemStats(&m1)

	// Test string creation and destruction in a loop.
	for i := 0; i < 1000; i++ {
		data := []byte("test string for memory leak detection")
		str := NewString(data)

		// Create some operations to ensure the string is used.
		_ = str.Len()
		_ = str.IsEmpty()

		// Explicitly destroy (though it's pooled).
		str.Destroy()
	}

	// Test vector creation and destruction.
	for i := 0; i < 100; i++ {
		vec := NewVec(&TypeInfo{Size: 8})

		// Add some elements.
		for j := 0; j < 5; j++ {
			value := int64(j)
			vec.Push(unsafe.Pointer(&value))
		}

		// Access elements.
		for j := 0; j < int(vec.Len()); j++ {
			_ = vec.Get(uintptr(j))
		}

		vec.Destroy()
	}

	// Force GC and measure memory.
	runtime.GC()
	runtime.GC()
	time.Sleep(10 * time.Millisecond) // Allow GC to complete
	runtime.ReadMemStats(&m2)

	// Check for significant memory increase.
	memIncrease := m2.Alloc - m1.Alloc
	if memIncrease > 100*1024 { // More than 100KB increase
		t.Errorf("Potential memory leak detected: memory increased by %d bytes", memIncrease)
	}

	t.Logf("Memory usage - Before: %d bytes, After: %d bytes, Increase: %d bytes",
		m1.Alloc, m2.Alloc, memIncrease)
}

func TestStringPoolLeakDetection(t *testing.T) {
	config := &allocator.Config{
		ArenaSize:      1024 * 1024, // 1MB
		PoolSizes:      []uintptr{64, 256, 1024, 4096},
		AlignmentSize:  8,                  // 8-byte alignment
		MemoryLimit:    1024 * 1024 * 1024, // 1GB limit
		EnableTracking: true,
	}
	alloc := allocator.NewSystemAllocator(config)

	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}

	defer ShutdownCoreTypes()

	// Create many different strings to test pool cleanup.
	for i := 0; i < 1000; i++ {
		data := []byte(fmt.Sprintf("unique string %d", i))
		str := NewString(data)
		_ = str.Len()
	}

	// Check pool size.
	GlobalCoreTypeManager.stringPoolMu.RLock()
	poolSize := len(GlobalCoreTypeManager.stringPool)
	GlobalCoreTypeManager.stringPoolMu.RUnlock()

	if poolSize > 1000 {
		t.Errorf("String pool size is too large: %d entries", poolSize)
	}

	t.Logf("String pool contains %d entries", poolSize)
}

func TestConcurrentMemoryOperations(t *testing.T) {
	config := &allocator.Config{
		ArenaSize:      1024 * 1024, // 1MB
		PoolSizes:      []uintptr{64, 256, 1024, 4096},
		AlignmentSize:  8,                  // 8-byte alignment
		MemoryLimit:    1024 * 1024 * 1024, // 1GB limit
		EnableTracking: true,
	}
	alloc := allocator.NewSystemAllocator(config)

	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}

	defer ShutdownCoreTypes()

	const numGoroutines = 50

	const numOperations = 100

	done := make(chan bool, numGoroutines)

	// Test concurrent string operations.
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				data := []byte(fmt.Sprintf("concurrent string %d-%d", id, j))
				str := NewString(data)
				_ = str.Len()
				// Note: We don't call Destroy() here because strings are pooled.
			}
		}(i)
	}

	// Wait for all goroutines to complete.
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check that we didn't crash or leak severely.
	runtime.GC()
	runtime.GC()

	var m runtime.MemStats

	runtime.ReadMemStats(&m)
	t.Logf("After concurrent operations: %d bytes allocated", m.Alloc)
}

func BenchmarkMemoryAllocation(b *testing.B) {
	config := &allocator.Config{
		ArenaSize:      1024 * 1024, // 1MB
		PoolSizes:      []uintptr{64, 256, 1024, 4096},
		AlignmentSize:  8,                  // 8-byte alignment
		MemoryLimit:    1024 * 1024 * 1024, // 1GB limit
		EnableTracking: true,
	}
	alloc := allocator.NewSystemAllocator(config)

	err := InitializeCoreTypes(alloc)
	if err != nil {
		b.Fatalf("Failed to initialize core types: %v", err)
	}

	defer ShutdownCoreTypes()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data := []byte("benchmark string")
		str := NewString(data)
		_ = str.Len()
	}
}
