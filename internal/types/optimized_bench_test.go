package types

import (
	"testing"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/allocator"
)

// BenchmarkOptimizedVecOperations benchmarks optimized vector operations.
func BenchmarkOptimizedVecOperations(b *testing.B) {
	intType := &TypeInfo{Size: 8} // Simple int type
	vec := NewVec(intType)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		val := i
		vec.Push(unsafe.Pointer(&val))
		if i%100 == 0 {
			vec = NewVec(intType) // Reset to test growth patterns
		}
	}
}

// BenchmarkOptimizedStringCreation benchmarks optimized string creation with pooling.
func BenchmarkOptimizedStringCreation(b *testing.B) {
	// Initialize core type system for benchmarks
	alloc := allocator.NewSystemAllocator(&allocator.Config{
		AlignmentSize:  8,
		EnableTracking: false,
	})
	err := InitializeCoreTypes(alloc)
	if err != nil {
		b.Fatalf("Failed to initialize core types: %v", err)
	}

	testStr := []byte("test string")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str := NewString(testStr)
		_ = str
	}
}

// BenchmarkOptimizedStringCreationLarge benchmarks optimized large string creation.
func BenchmarkOptimizedStringCreationLarge(b *testing.B) {
	// Initialize core type system for benchmarks
	alloc := allocator.NewSystemAllocator(&allocator.Config{
		AlignmentSize:  8,
		EnableTracking: false,
	})
	err := InitializeCoreTypes(alloc)
	if err != nil {
		b.Fatalf("Failed to initialize core types: %v", err)
	}

	largeStr := make([]byte, 2048)
	for i := range largeStr {
		largeStr[i] = byte('a' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str := NewString(largeStr)
		_ = str
	}
}
