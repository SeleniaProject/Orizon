package types

import (
	"testing"
	"unsafe"
)

func TestSafeErrorHandler_SafeCall(t *testing.T) {
	handler := NewSafeErrorHandler()

	// Test normal function execution
	err := handler.SafeCall(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error for normal execution, got: %v", err)
	}

	if handler.HasPanicOccurred() {
		t.Error("Expected no panic for normal execution")
	}
}

func TestSafeErrorHandler_PanicRecovery(t *testing.T) {
	handler := NewSafeErrorHandler()

	// Test panic recovery
	err := handler.SafeCall(func() error {
		panic("test panic")
	})

	if err == nil {
		t.Error("Expected error after panic recovery")
	}

	if !handler.HasPanicOccurred() {
		t.Error("Expected panic to be detected")
	}

	lastErr := handler.GetLastError()
	if lastErr == nil {
		t.Error("Expected last error to be set")
	}

	if lastErr.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestSafeUnwrap(t *testing.T) {
	// Test successful unwrap
	opt := Some(42)
	value, err := SafeUnwrap(opt)
	if err != nil {
		t.Errorf("Expected no error for Some value, got: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected value 42, got: %v", value)
	}

	// Test failed unwrap
	noneOpt := None[int]()
	_, err = SafeUnwrap(noneOpt)
	if err == nil {
		t.Error("Expected error for None value")
	}
}

func TestSafeUnwrapResult(t *testing.T) {
	// Test successful unwrap
	okResult := Ok[int, string](42)
	value, err := SafeUnwrapResult(okResult)
	if err != nil {
		t.Errorf("Expected no error for Ok result, got: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected value 42, got: %v", value)
	}

	// Test failed unwrap
	errResult := Err[int, string]("test error")
	_, err = SafeUnwrapResult(errResult)
	if err == nil {
		t.Error("Expected error for Err result")
	}
}

func TestSafeSliceAccess(t *testing.T) {
	// Create a test type info
	typeInfo := &CoreTypeInfo{
		Size: 4, // int32
	}

	// Create test data
	data := make([]int32, 5)
	for i := range data {
		data[i] = int32(i * 10)
	}

	slice := CoreSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   int64(len(data)),
		capacity: int64(len(data)),
		typeInfo: typeInfo,
	}

	// Test valid access
	ptr, err := SafeSliceAccess(slice, 2)
	if err != nil {
		t.Errorf("Expected no error for valid access, got: %v", err)
	}
	if ptr == nil {
		t.Error("Expected non-nil pointer")
	}

	// Test out of bounds access
	_, err = SafeSliceAccess(slice, 10)
	if err == nil {
		t.Error("Expected error for out of bounds access")
	}

	// Test negative index
	_, err = SafeSliceAccess(slice, -1)
	if err == nil {
		t.Error("Expected error for negative index")
	}
}

func TestSafeSliceSubslice(t *testing.T) {
	// Create a test type info
	typeInfo := &CoreTypeInfo{
		Size: 4, // int32
	}

	// Create test data
	data := make([]int32, 10)

	slice := CoreSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   int64(len(data)),
		capacity: int64(len(data)),
		typeInfo: typeInfo,
	}

	// Test valid subslice
	subslice, err := SafeSliceSubslice(slice, 2, 5)
	if err != nil {
		t.Errorf("Expected no error for valid subslice, got: %v", err)
	}
	if subslice.length != 3 {
		t.Errorf("Expected subslice length 3, got: %d", subslice.length)
	}

	// Test invalid bounds
	_, err = SafeSliceSubslice(slice, 5, 2)
	if err == nil {
		t.Error("Expected error for invalid bounds (start > end)")
	}

	// Test out of bounds
	_, err = SafeSliceSubslice(slice, 0, 20)
	if err == nil {
		t.Error("Expected error for out of bounds subslice")
	}
}

func TestSafeGetCoreTypeManager(t *testing.T) {
	// Save original manager
	originalManager := globalCoreManager
	defer func() {
		globalCoreManager = originalManager
	}()

	// Test with nil manager
	globalCoreManager = nil
	_, err := SafeGetCoreTypeManager()
	if err == nil {
		t.Error("Expected error when core type manager is nil")
	}

	// Test with valid manager
	globalCoreManager = &CoreTypeManager{}
	manager, err := SafeGetCoreTypeManager()
	if err != nil {
		t.Errorf("Expected no error with valid manager, got: %v", err)
	}
	if manager == nil {
		t.Error("Expected non-nil manager")
	}
}

func BenchmarkSafeCall(b *testing.B) {
	handler := NewSafeErrorHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.SafeCall(func() error {
			return nil
		})
	}
}

func BenchmarkSafeSliceAccess(b *testing.B) {
	typeInfo := &CoreTypeInfo{Size: 4}
	data := make([]int32, 1000)
	slice := CoreSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   int64(len(data)),
		capacity: int64(len(data)),
		typeInfo: typeInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SafeSliceAccess(slice, int64(i%1000))
	}
}
