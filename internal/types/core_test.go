package types

import (
	"testing"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/allocator"
)

func getTestConfig() *allocator.Config {
	return &allocator.Config{
		ArenaSize:      1024 * 1024, // 1MB
		PoolSizes:      []uintptr{64, 256, 1024, 4096},
		AlignmentSize:  8,                  // 8-byte alignment
		MemoryLimit:    1024 * 1024 * 1024, // 1GB limit
		EnableTracking: true,
	}
}

func TestCoreTypeInitialization(t *testing.T) {
	// Test core type manager initialization
	alloc := allocator.NewSystemAllocator(getTestConfig())

	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}

	if GlobalCoreTypeManager == nil {
		t.Fatal("Global core type manager should be initialized")
	}

	// Clean up
	ShutdownCoreTypes()

	if GlobalCoreTypeManager != nil {
		t.Error("Global core type manager should be nil after shutdown")
	}
}

func TestOption(t *testing.T) {
	// Initialize core types
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	// Test Some option
	value := int64(42)
	someOption := NewSome(unsafe.Pointer(&value), TypeInfoInt64)

	if !someOption.IsSome() {
		t.Error("Some option should return true for IsSome()")
	}

	if someOption.IsNone() {
		t.Error("Some option should return false for IsNone()")
	}

	unwrapped := someOption.Unwrap()
	if *(*int64)(unwrapped) != 42 {
		t.Errorf("Expected unwrapped value to be 42, got %d", *(*int64)(unwrapped))
	}

	// Test None option
	noneOption := NewNone(TypeInfoInt64)

	if noneOption.IsSome() {
		t.Error("None option should return false for IsSome()")
	}

	if !noneOption.IsNone() {
		t.Error("None option should return true for IsNone()")
	}

	// Test UnwrapOr
	defaultValue := int64(100)
	result := noneOption.UnwrapOr(unsafe.Pointer(&defaultValue))
	if *(*int64)(result) != 100 {
		t.Errorf("Expected default value 100, got %d", *(*int64)(result))
	}

	// Test Map
	doubleFunc := func(ptr unsafe.Pointer) unsafe.Pointer {
		val := *(*int64)(ptr)
		doubled := val * 2
		return unsafe.Pointer(&doubled)
	}

	mappedSome := someOption.Map(doubleFunc)
	if !mappedSome.IsSome() {
		t.Error("Mapped Some should still be Some")
	}

	mappedNone := noneOption.Map(doubleFunc)
	if !mappedNone.IsNone() {
		t.Error("Mapped None should still be None")
	}
}

func TestResult(t *testing.T) {
	// Initialize core types
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	// Test Ok result
	value := int64(42)
	okResult := NewOk(unsafe.Pointer(&value), TypeInfoInt64, TypeInfoInt32)

	if !okResult.IsOk() {
		t.Error("Ok result should return true for IsOk()")
	}

	if okResult.IsErr() {
		t.Error("Ok result should return false for IsErr()")
	}

	unwrapped := okResult.Unwrap()
	if *(*int64)(unwrapped) != 42 {
		t.Errorf("Expected unwrapped value to be 42, got %d", *(*int64)(unwrapped))
	}

	// Test Err result
	errorValue := int32(-1)
	errResult := NewErr(unsafe.Pointer(&errorValue), TypeInfoInt64, TypeInfoInt32)

	if errResult.IsOk() {
		t.Error("Err result should return false for IsOk()")
	}

	if !errResult.IsErr() {
		t.Error("Err result should return true for IsErr()")
	}

	unwrappedErr := errResult.UnwrapErr()
	if *(*int32)(unwrappedErr) != -1 {
		t.Errorf("Expected unwrapped error to be -1, got %d", *(*int32)(unwrappedErr))
	}

	// Test UnwrapOr
	defaultValue := int64(100)
	result := errResult.UnwrapOr(unsafe.Pointer(&defaultValue))
	if *(*int64)(result) != 100 {
		t.Errorf("Expected default value 100, got %d", *(*int64)(result))
	}

	// Test Map
	doubleFunc := func(ptr unsafe.Pointer) unsafe.Pointer {
		val := *(*int64)(ptr)
		doubled := val * 2
		return unsafe.Pointer(&doubled)
	}

	mappedOk := okResult.Map(doubleFunc)
	if !mappedOk.IsOk() {
		t.Error("Mapped Ok should still be Ok")
	}

	mappedErr := errResult.Map(doubleFunc)
	if !mappedErr.IsErr() {
		t.Error("Mapped Err should still be Err")
	}
}

func TestSlice(t *testing.T) {
	// Initialize core types
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	// Create test array
	array := [5]int64{1, 2, 3, 4, 5}
	slice := NewSliceFromArray(unsafe.Pointer(&array[0]), 5, TypeInfoInt64)

	// Test basic properties
	if slice.Len() != 5 {
		t.Errorf("Expected slice length 5, got %d", slice.Len())
	}

	if slice.Cap() != 5 {
		t.Errorf("Expected slice capacity 5, got %d", slice.Cap())
	}

	if slice.IsEmpty() {
		t.Error("Slice should not be empty")
	}

	// Test element access
	for i := uintptr(0); i < 5; i++ {
		element := slice.Get(i)
		value := *(*int64)(element)
		expected := int64(i + 1)
		if value != expected {
			t.Errorf("Expected element %d to be %d, got %d", i, expected, value)
		}
	}

	// Test element modification
	newValueArray := [1]int64{99}
	slice.Set(0, unsafe.Pointer(&newValueArray[0]))
	firstElement := slice.Get(0)
	if *(*int64)(firstElement) != 99 {
		t.Errorf("Expected first element to be 99, got %d", *(*int64)(firstElement))
	}

	// Test sub-slice
	subSlice := slice.Sub(1, 4)
	if subSlice.Len() != 3 {
		t.Errorf("Expected sub-slice length 3, got %d", subSlice.Len())
	}

	firstSubElement := subSlice.Get(0)
	if *(*int64)(firstSubElement) != 2 {
		t.Errorf("Expected first sub-slice element to be 2, got %d", *(*int64)(firstSubElement))
	}
}

func TestString(t *testing.T) {
	// Initialize core types
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	// Test allocator directly first
	testPtr := alloc.Alloc(100)
	if testPtr == nil {
		t.Fatal("Allocator returned nil for test allocation")
	}
	alloc.Free(testPtr)

	// Test string creation
	data := []byte("Hello, Orizon!")
	str := NewString(data)

	if str.Len() != uintptr(len(data)) {
		t.Errorf("Expected string length %d, got %d", len(data), str.Len())
	}

	if str.IsEmpty() {
		t.Error("String should not be empty")
	}

	// Test string content
	goStr := str.AsGoString()
	if goStr != "Hello, Orizon!" {
		t.Errorf("Expected string 'Hello, Orizon!', got '%s'", goStr)
	}

	// Test empty string
	emptyStr := NewString([]byte{})
	if !emptyStr.IsEmpty() {
		t.Error("Empty string should be empty")
	}

	if emptyStr.Len() != 0 {
		t.Errorf("Expected empty string length 0, got %d", emptyStr.Len())
	}

	// Test string equality
	str2 := NewString([]byte("Hello, Orizon!"))
	if !str.Equals(str2) {
		t.Error("Identical strings should be equal")
	}

	str3 := NewString([]byte("Different"))
	if str.Equals(str3) {
		t.Error("Different strings should not be equal")
	}

	// Test string comparison
	if str.Compare(str2) != 0 {
		t.Error("Identical strings should compare equal")
	}

	if str.Compare(str3) <= 0 {
		t.Error("'Hello, Orizon!' should be greater than 'Different'")
	}

	// Test string concatenation
	str4 := NewString([]byte(" World"))
	concat := str.Concat(str4)
	expectedConcat := "Hello, Orizon! World"
	if concat.AsGoString() != expectedConcat {
		t.Errorf("Expected concatenated string '%s', got '%s'", expectedConcat, concat.AsGoString())
	}

	// Test string pooling
	str5 := NewString([]byte("Hello, Orizon!"))
	if str != str5 {
		t.Error("Identical strings should be the same instance (pooled)")
	}
}

func TestVec(t *testing.T) {
	// Initialize core types
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	// Test vector creation
	vec := NewVec(TypeInfoInt64)

	if vec.Len() != 0 {
		t.Errorf("Expected new vector length 0, got %d", vec.Len())
	}

	if !vec.IsEmpty() {
		t.Error("New vector should be empty")
	}

	// Test vector with capacity
	vecWithCap := NewVecWithCapacity(10, TypeInfoInt64)
	if vecWithCap.Cap() != 10 {
		t.Errorf("Expected vector capacity 10, got %d", vecWithCap.Cap())
	}

	if vecWithCap.Len() != 0 {
		t.Errorf("Expected vector with capacity length 0, got %d", vecWithCap.Len())
	}

	// Test push operations
	for i := int64(0); i < 5; i++ {
		vec.Push(unsafe.Pointer(&i))
	}

	if vec.Len() != 5 {
		t.Errorf("Expected vector length 5 after pushes, got %d", vec.Len())
	}

	if vec.IsEmpty() {
		t.Error("Vector should not be empty after pushes")
	}

	// Test element access
	for i := uintptr(0); i < 5; i++ {
		element := vec.Get(i)
		value := *(*int64)(element)
		expected := int64(i)
		if value != expected {
			t.Errorf("Expected element %d to be %d, got %d", i, expected, value)
		}
	}

	// Test element modification
	newValue := int64(99)
	vec.Set(0, unsafe.Pointer(&newValue))
	firstElement := vec.Get(0)
	if *(*int64)(firstElement) != 99 {
		t.Errorf("Expected first element to be 99, got %d", *(*int64)(firstElement))
	}

	// Test pop operation
	lastElement := vec.Pop()
	lastValue := *(*int64)(lastElement)
	if lastValue != 4 {
		t.Errorf("Expected popped value to be 4, got %d", lastValue)
	}

	if vec.Len() != 4 {
		t.Errorf("Expected vector length 4 after pop, got %d", vec.Len())
	}

	// Test clear
	vec.Clear()
	if vec.Len() != 0 {
		t.Errorf("Expected vector length 0 after clear, got %d", vec.Len())
	}

	if !vec.IsEmpty() {
		t.Error("Vector should be empty after clear")
	}

	// Test as slice
	vec.Push(unsafe.Pointer(&newValue))
	slice := vec.AsSlice()
	if slice.Len() != vec.Len() {
		t.Errorf("Expected slice length %d, got %d", vec.Len(), slice.Len())
	}

	// Clean up
	vec.Destroy()
	vecWithCap.Destroy()
}

func TestTypeInfo(t *testing.T) {
	// Test primitive type info
	if TypeInfoInt64.Size != 8 {
		t.Errorf("Expected int64 size 8, got %d", TypeInfoInt64.Size)
	}

	if TypeInfoInt64.Alignment != 8 {
		t.Errorf("Expected int64 alignment 8, got %d", TypeInfoInt64.Alignment)
	}

	if TypeInfoInt64.Name != "i64" {
		t.Errorf("Expected int64 name 'i64', got '%s'", TypeInfoInt64.Name)
	}

	if !TypeInfoInt64.IsPrimitive {
		t.Error("int64 should be marked as primitive")
	}

	if TypeInfoInt64.IsPointer {
		t.Error("int64 should not be marked as pointer")
	}

	// Test pointer type info
	if TypeInfoPtr.Size != 8 {
		t.Errorf("Expected pointer size 8, got %d", TypeInfoPtr.Size)
	}

	if !TypeInfoPtr.IsPointer {
		t.Error("Pointer type should be marked as pointer")
	}

	if TypeInfoPtr.IsPrimitive {
		t.Error("Pointer type should not be marked as primitive")
	}
}

func TestMemoryManagement(t *testing.T) {
	// Initialize core types
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	// Test string memory management
	str := NewString([]byte("Test string for memory management"))
	str.Destroy()

	// Test vector memory management
	vec := NewVecWithCapacity(100, TypeInfoInt64)
	for i := int64(0); i < 50; i++ {
		value := i // Create a copy to get a different address
		vec.Push(unsafe.Pointer(&value))
	}
	vec.Destroy()
}

func BenchmarkStringCreation(b *testing.B) {
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		b.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	data := []byte("Benchmark string creation")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str := NewString(data)
		_ = str
	}
}

func BenchmarkVecOperations(b *testing.B) {
	alloc := allocator.NewSystemAllocator(getTestConfig())
	err := InitializeCoreTypes(alloc)
	if err != nil {
		b.Fatalf("Failed to initialize core types: %v", err)
	}
	defer ShutdownCoreTypes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vec := NewVecWithCapacity(100, TypeInfoInt64)

		for j := int64(0); j < 100; j++ {
			vec.Push(unsafe.Pointer(&j))
		}

		vec.Destroy()
	}
}
