package types

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

// SafeAllocator provides memory allocation with bounds checking.
type SafeAllocator struct {
	allocations map[unsafe.Pointer]*AllocationInfo
	mutex       sync.RWMutex
}

// AllocationInfo tracks allocated memory regions.
type AllocationInfo struct {
	Type      reflect.Type
	Size      uintptr
	Allocated int64
	Valid     bool
}

// UnsafeOperationError represents an unsafe operation error.
type UnsafeOperationError struct {
	Pointer   unsafe.Pointer
	Operation string
	Caller    string
	Size      uintptr
}

func (e *UnsafeOperationError) Error() string {
	return fmt.Sprintf("unsafe operation '%s' on pointer %p with size %d from %s",
		e.Operation, e.Pointer, e.Size, e.Caller)
}

// NewSafeAllocator creates a new safe allocator.
func NewSafeAllocator() *SafeAllocator {
	return &SafeAllocator{
		allocations: make(map[unsafe.Pointer]*AllocationInfo),
	}
}

// RegisterAllocation registers a memory allocation.
func (allocator *SafeAllocator) RegisterAllocation(ptr unsafe.Pointer, size uintptr, typ reflect.Type) {
	allocator.mutex.Lock()
	defer allocator.mutex.Unlock()

	allocator.allocations[ptr] = &AllocationInfo{
		Size:      size,
		Allocated: time.Now().UnixNano(),
		Type:      typ,
		Valid:     true,
	}
}

// UnregisterAllocation unregisters a memory allocation.
func (allocator *SafeAllocator) UnregisterAllocation(ptr unsafe.Pointer) {
	allocator.mutex.Lock()
	defer allocator.mutex.Unlock()

	if info, exists := allocator.allocations[ptr]; exists {
		info.Valid = false

		delete(allocator.allocations, ptr)
	}
}

// IsValidPointer checks if a pointer is valid for the given size.
func (allocator *SafeAllocator) IsValidPointer(ptr unsafe.Pointer, size uintptr) bool {
	if ptr == nil {
		return size == 0
	}

	allocator.mutex.RLock()
	defer allocator.mutex.RUnlock()

	// Additional safety checks for pointer validity
	accessStart := uintptr(ptr)

	// Check for potential integer overflow in access range calculation
	if size > 0 && accessStart > ^uintptr(0)-size {
		return false // Would overflow
	}

	accessEnd := accessStart + size

	// Check if the pointer is within any registered allocation.
	for allocPtr, info := range allocator.allocations {
		if !info.Valid {
			continue
		}

		// Calculate the range of the allocation with overflow protection
		allocStart := uintptr(allocPtr)

		// Verify allocation size is valid
		if info.Size > ^uintptr(0)-allocStart {
			continue // Invalid allocation range
		}

		allocEnd := allocStart + info.Size

		// Enhanced boundary checks
		// 1. Check if access is entirely within allocation
		// 2. Ensure no wraparound occurred in calculations
		if accessStart >= allocStart &&
			accessEnd <= allocEnd &&
			accessEnd >= accessStart && // No wraparound in access range
			allocEnd >= allocStart { // No wraparound in allocation range
			return true
		}
	}

	return false
}

// GetAllocationInfo returns information about an allocation.
func (allocator *SafeAllocator) GetAllocationInfo(ptr unsafe.Pointer) *AllocationInfo {
	allocator.mutex.RLock()
	defer allocator.mutex.RUnlock()

	if info, exists := allocator.allocations[ptr]; exists && info.Valid {
		// Return a copy to avoid race conditions.
		return &AllocationInfo{
			Size:      info.Size,
			Allocated: info.Allocated,
			Type:      info.Type,
			Valid:     info.Valid,
		}
	}

	return nil
}

// CleanupInvalidAllocations removes invalid allocations.
func (allocator *SafeAllocator) CleanupInvalidAllocations() int {
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
