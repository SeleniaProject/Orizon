package types

import (
	"fmt"
	"unsafe"
)

// Fixed versions of safe error handling functions to resolve compilation errors.

// SafeUnwrap safely unwraps an OrizonOption value.
func SafeUnwrap(opt *OrizonOption) (unsafe.Pointer, error) {
	if !opt.IsSome() {
		return nil, fmt.Errorf("attempted to unwrap None value")
	}

	return opt.Unwrap(), nil
}

// SafeUnwrapResult safely unwraps an OrizonResult value.
func SafeUnwrapResult(result *OrizonResult) (unsafe.Pointer, error) {
	if !result.IsOk() {
		return nil, fmt.Errorf("attempted to unwrap error result")
	}

	return result.Unwrap(), nil
}

// SafeSliceAccess safely accesses a slice element with bounds checking.
func SafeSliceAccess(slice *OrizonSlice, index int64) (unsafe.Pointer, error) {
	// Bounds check.
	if index < 0 || index >= int64(slice.length) {
		return nil, fmt.Errorf("index %d out of bounds for slice of length %d", index, slice.length)
	}

	// Type info check.
	if slice.typeInfo == nil {
		return nil, fmt.Errorf("invalid slice: nil typeInfo")
	}

	// Check if slice.typeInfo.Size is valid
	elementSize := slice.typeInfo.Size
	if elementSize == 0 {
		return nil, fmt.Errorf("invalid element size: 0")
	}

	// Calculate the offset with overflow check.
	offset := uintptr(index) * elementSize
	if offset/elementSize != uintptr(index) {
		return nil, fmt.Errorf("offset calculation would overflow")
	}

	// Additional bounds check to prevent buffer overflow.
	totalSize := slice.length * elementSize
	if offset+elementSize > totalSize {
		return nil, fmt.Errorf("element access would exceed slice bounds")
	}

	// Use unsafe.Add for race detector compatibility
	return unsafe.Add(slice.data, offset), nil
}

// SafeSliceSet safely sets a slice element with bounds checking.
func SafeSliceSet(slice *OrizonSlice, index int64, value unsafe.Pointer) error {
	// Bounds check.
	if index < 0 || index >= int64(slice.length) {
		return fmt.Errorf("index %d out of bounds for slice of length %d", index, slice.length)
	}

	// Type info check.
	if slice.typeInfo == nil {
		return fmt.Errorf("invalid slice: nil typeInfo")
	}

	// Check if slice.typeInfo.Size is valid
	elementSize := slice.typeInfo.Size
	if elementSize == 0 {
		return fmt.Errorf("invalid element size: 0")
	}

	// Calculate the offset with overflow check.
	offset := uintptr(index) * elementSize
	if offset/elementSize != uintptr(index) {
		return fmt.Errorf("offset calculation would overflow")
	}

	// Additional bounds check to prevent buffer overflow.
	totalSize := slice.length * elementSize
	if offset+elementSize > totalSize {
		return fmt.Errorf("element access would exceed slice bounds")
	}

	// Validate source value pointer.
	if value == nil {
		return fmt.Errorf("cannot set nil value")
	}

	// Use unsafe.Add for race detector compatibility
	dest := unsafe.Add(slice.data, offset)
	copyBytes(dest, value, elementSize)

	return nil
}

// SafeSliceSubslice safely creates a subslice with bounds checking.
func SafeSliceSubslice(slice *OrizonSlice, start, end int64) (*OrizonSlice, error) {
	if start < 0 || end < start || end > int64(slice.length) {
		return nil, fmt.Errorf("invalid slice bounds [%d:%d] for slice of length %d", start, end, slice.length)
	}

	if slice.typeInfo == nil {
		return nil, fmt.Errorf("invalid slice: nil typeInfo")
	}

	elementSize := slice.typeInfo.Size
	if elementSize == 0 {
		return nil, fmt.Errorf("invalid element size: 0")
	}

	// Calculate the offset with overflow check.
	offset := uintptr(start) * elementSize
	if offset/elementSize != uintptr(start) {
		return nil, fmt.Errorf("offset calculation would overflow")
	}

	var newData unsafe.Pointer

	newLength := uintptr(end - start)

	if newLength == 0 {
		newData = nil
	} else {
		newData = unsafe.Add(slice.data, offset)
	}

	return &OrizonSlice{
		data:     newData,
		length:   newLength,
		capacity: slice.capacity - uintptr(start),
		typeInfo: slice.typeInfo,
	}, nil
}

// GetCoreManager returns the global core manager (placeholder implementation).
func GetCoreManager() (*OrizonSlice, error) {
	// For now, return nil as we don't have a global core manager implemented.
	return nil, fmt.Errorf("global core manager not implemented")
}
