package types

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// UnsafeOperationGuard provides protection for unsafe operations.
type UnsafeOperationGuard struct {
	auditor   *UnsafeAuditor
	allocator *safeAllocator
	mutex     sync.RWMutex
	enabled   bool
}

// UnsafeAuditor tracks and validates unsafe operations.
type UnsafeAuditor struct {
	operations []UnsafeOperation
	mutex      sync.RWMutex
}

// UnsafeOperation represents a tracked unsafe operation.
type UnsafeOperation struct {
	Source    unsafe.Pointer
	Target    unsafe.Pointer
	Type      string
	Caller    string
	Size      uintptr
	Timestamp int64
	Validated bool
}

// SafeAllocator provides memory allocation with bounds checking.
type safeAllocator struct {
	allocations map[unsafe.Pointer]*allocationInfo
	mutex       sync.RWMutex
}

// allocationInfo tracks allocated memory regions.
type allocationInfo struct {
	Type      reflect.Type
	Size      uintptr
	Allocated int64
	Valid     bool
}

// NewUnsafeOperationGuard creates a new unsafe operation guard.
func NewUnsafeOperationGuard() *UnsafeOperationGuard {
	return &UnsafeOperationGuard{
		enabled:   true,
		auditor:   NewUnsafeAuditor(),
		allocator: newSafeAllocator(),
	}
}

// NewUnsafeAuditor creates a new unsafe auditor.
func NewUnsafeAuditor() *UnsafeAuditor {
	return &UnsafeAuditor{
		operations: make([]UnsafeOperation, 0),
	}
}

// newSafeAllocator creates a new safe allocator.
func newSafeAllocator() *safeAllocator {
	return &safeAllocator{
		allocations: make(map[unsafe.Pointer]*allocationInfo),
	}
}

// ValidatePointer validates that a pointer operation is safe.
func (guard *UnsafeOperationGuard) ValidatePointer(ptr unsafe.Pointer, size uintptr, operation string) error {
	if !guard.enabled {
		return nil
	}

	guard.mutex.RLock()
	defer guard.mutex.RUnlock()

	// Check if pointer is in valid allocation.
	if !guard.allocator.IsValidPointer(ptr, size) {
		pc, _, _, _ := runtime.Caller(1)
		caller := runtime.FuncForPC(pc).Name()

		// Log the unsafe operation.
		guard.auditor.LogOperation(UnsafeOperation{
			Type:      operation,
			Source:    ptr,
			Size:      size,
			Timestamp: time.Now().UnixNano(),
			Caller:    caller,
			Validated: false,
		})

		return &unsafeOperationError{
			Operation: operation,
			Pointer:   ptr,
			Size:      size,
			Caller:    caller,
		}
	}

	// Log successful validation.
	pc, _, _, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	guard.auditor.LogOperation(UnsafeOperation{
		Type:      operation,
		Source:    ptr,
		Size:      size,
		Timestamp: time.Now().UnixNano(),
		Caller:    caller,
		Validated: true,
	})

	return nil
}

// unsafeOperationError represents an unsafe operation error.
type unsafeOperationError struct {
	Pointer   unsafe.Pointer
	Operation string
	Caller    string
	Size      uintptr
}

func (e *unsafeOperationError) Error() string {
	return fmt.Sprintf("unsafe operation '%s' on pointer %p with size %d from %s",
		e.Operation, e.Pointer, e.Size, e.Caller)
}

// SafePointerCast performs a validated pointer cast.
func (guard *UnsafeOperationGuard) SafePointerCast(ptr unsafe.Pointer, targetSize uintptr) (unsafe.Pointer, error) {
	if err := guard.ValidatePointer(ptr, targetSize, "pointer_cast"); err != nil {
		return nil, err
	}

	return ptr, nil
}

// SafeSliceFromPointer creates a slice from a pointer with bounds checking.
func (guard *UnsafeOperationGuard) SafeSliceFromPointer(
	ptr unsafe.Pointer, length, capacity uintptr, elementSize uintptr,
) (unsafe.Pointer, error) {
	totalSize := capacity * elementSize
	if err := guard.ValidatePointer(ptr, totalSize, "slice_from_pointer"); err != nil {
		return nil, err
	}

	// Additional validation: ensure length <= capacity.
	if length > capacity {
		return nil, &unsafeOperationError{
			Operation: "slice_from_pointer",
			Pointer:   ptr,
			Size:      totalSize,
			Caller:    "length > capacity",
		}
	}

	return ptr, nil
}

// LogOperation logs an unsafe operation for auditing.
func (auditor *UnsafeAuditor) LogOperation(op UnsafeOperation) {
	auditor.mutex.Lock()
	defer auditor.mutex.Unlock()

	auditor.operations = append(auditor.operations, op)

	// Limit audit log size to prevent memory leaks.
	if len(auditor.operations) > 10000 {
		// Keep only the latest 5000 operations.
		copy(auditor.operations, auditor.operations[5000:])
		auditor.operations = auditor.operations[:5000]
	}
}

// GetOperations returns a copy of all logged operations.
func (auditor *UnsafeAuditor) GetOperations() []UnsafeOperation {
	auditor.mutex.RLock()
	defer auditor.mutex.RUnlock()

	operations := make([]UnsafeOperation, len(auditor.operations))
	copy(operations, auditor.operations)

	return operations
}

// GetFailedOperations returns operations that failed validation.
func (auditor *UnsafeAuditor) GetFailedOperations() []UnsafeOperation {
	auditor.mutex.RLock()
	defer auditor.mutex.RUnlock()

	var failed []UnsafeOperation

	for _, op := range auditor.operations {
		if !op.Validated {
			failed = append(failed, op)
		}
	}

	return failed
}

// RegisterAllocation registers a memory allocation.
func (allocator *safeAllocator) RegisterAllocation(ptr unsafe.Pointer, size uintptr, typ reflect.Type) {
	allocator.mutex.Lock()
	defer allocator.mutex.Unlock()

	allocator.allocations[ptr] = &allocationInfo{
		Size:      size,
		Allocated: time.Now().UnixNano(),
		Type:      typ,
		Valid:     true,
	}
}

// UnregisterAllocation unregisters a memory allocation.
func (allocator *safeAllocator) UnregisterAllocation(ptr unsafe.Pointer) {
	allocator.mutex.Lock()
	defer allocator.mutex.Unlock()

	if info, exists := allocator.allocations[ptr]; exists {
		info.Valid = false

		delete(allocator.allocations, ptr)
	}
}

// IsValidPointer checks if a pointer is valid for the given size.
func (allocator *safeAllocator) IsValidPointer(ptr unsafe.Pointer, size uintptr) bool {
	if ptr == nil {
		return size == 0
	}

	allocator.mutex.RLock()
	defer allocator.mutex.RUnlock()

	// Check if the pointer is within any registered allocation.
	for allocPtr, info := range allocator.allocations {
		if !info.Valid {
			continue
		}

		// Calculate the range of the allocation.
		allocStart := uintptr(allocPtr)
		allocEnd := allocStart + info.Size

		// Calculate the range of the requested access.
		accessStart := uintptr(ptr)
		accessEnd := accessStart + size

		// Check if the access is entirely within the allocation.
		if accessStart >= allocStart && accessEnd <= allocEnd {
			return true
		}
	}

	return false
}

// GetAllocationInfo returns information about an allocation.
func (allocator *safeAllocator) GetAllocationInfo(ptr unsafe.Pointer) *allocationInfo {
	allocator.mutex.RLock()
	defer allocator.mutex.RUnlock()

	if info, exists := allocator.allocations[ptr]; exists && info.Valid {
		// Return a copy to avoid race conditions.
		return &allocationInfo{
			Size:      info.Size,
			Allocated: info.Allocated,
			Type:      info.Type,
			Valid:     info.Valid,
		}
	}

	return nil
}

// CleanupInvalidAllocations removes invalid allocations.
func (allocator *safeAllocator) CleanupInvalidAllocations() int {
	allocator.mutex.Lock()
	defer allocator.mutex.Unlock()

	cleaned := 0

	for ptr, info := range allocator.allocations {
		if !info.Valid {
			delete(allocator.allocations, ptr)

			cleaned++
		}
	}

	return cleaned
}
