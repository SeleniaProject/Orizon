// Package layout provides memory layout definitions for Orizon's core data structures.
// This package defines the runtime representation of arrays, slices, strings, and other
// complex data types that require specific memory layouts for efficient code generation.
package layout

import (
	"fmt"
	"unsafe"
)

// LayoutKind represents different types of memory layouts
type LayoutKind int

const (
	LayoutArray LayoutKind = iota
	LayoutSlice
	LayoutString
	LayoutStruct
	LayoutUnion
	LayoutPointer
	LayoutReference
)

// MemoryLayout defines the memory layout of a data type
type MemoryLayout struct {
	Kind        LayoutKind
	Size        int64  // Total size in bytes
	Alignment   int64  // Required alignment in bytes
	ElementSize int64  // Size of individual elements (for arrays/slices)
	Metadata    []byte // Additional layout metadata
}

// ArrayLayout represents the memory layout of a fixed-size array
type ArrayLayout struct {
	ElementType  string  // Type name of elements
	ElementSize  int64   // Size of each element in bytes
	ElementAlign int64   // Alignment requirement of elements
	Length       int64   // Number of elements
	TotalSize    int64   // Total array size (Length * ElementSize)
	PaddingMap   []int64 // Padding offsets for alignment
}

// SliceLayout represents the memory layout of a dynamic slice
// Layout: [ptr: 8 bytes][len: 8 bytes][cap: 8 bytes] = 24 bytes total
type SliceLayout struct {
	ElementType  string // Type name of elements
	ElementSize  int64  // Size of each element in bytes
	ElementAlign int64  // Alignment requirement of elements
	PtrOffset    int64  // Offset of data pointer (0)
	LenOffset    int64  // Offset of length field (8)
	CapOffset    int64  // Offset of capacity field (16)
	TotalSize    int64  // Total slice header size (24)
}

// StringLayout represents the memory layout of a string
// Layout: [ptr: 8 bytes][len: 8 bytes] = 16 bytes total
type StringLayout struct {
	PtrOffset int64 // Offset of data pointer (0)
	LenOffset int64 // Offset of length field (8)
	TotalSize int64 // Total string header size (16)
}

// StructLayout represents the memory layout of a struct
type StructLayout struct {
	Name       string        // Struct name
	Fields     []FieldInfo   // Field information
	TotalSize  int64         // Total struct size including padding
	Alignment  int64         // Required alignment
	PaddingMap []PaddingInfo // Padding information
}

// FieldInfo contains information about a struct field
type FieldInfo struct {
	Name      string // Field name
	Type      string // Field type name
	Offset    int64  // Offset from struct start
	Size      int64  // Size of the field
	Alignment int64  // Required alignment
}

// PaddingInfo represents padding bytes inserted for alignment
type PaddingInfo struct {
	Offset int64  // Offset where padding starts
	Size   int64  // Number of padding bytes
	Reason string // Reason for padding (e.g., "field alignment", "struct alignment")
}

// LayoutCalculator provides methods to calculate memory layouts
type LayoutCalculator struct {
	TargetPointerSize int64 // Size of pointers on target architecture (8 for x64)
	MaxAlignment      int64 // Maximum alignment supported by target
}

// NewLayoutCalculator creates a new layout calculator for x64 architecture
func NewLayoutCalculator() *LayoutCalculator {
	return &LayoutCalculator{
		TargetPointerSize: 8,
		MaxAlignment:      16, // SSE alignment
	}
}

// CalculateArrayLayout calculates the memory layout for a fixed-size array
func (lc *LayoutCalculator) CalculateArrayLayout(elementType string, elementSize, elementAlign, length int64) (*ArrayLayout, error) {
	if length < 0 {
		return nil, fmt.Errorf("array length cannot be negative: %d", length)
	}
	if elementSize <= 0 {
		return nil, fmt.Errorf("element size must be positive: %d", elementSize)
	}
	if elementAlign <= 0 {
		elementAlign = 1
	}

	// Ensure element alignment is power of 2
	if !isPowerOfTwo(elementAlign) {
		return nil, fmt.Errorf("element alignment must be power of 2: %d", elementAlign)
	}

	// Calculate total size with proper alignment
	totalSize := length * elementSize

	// Add padding for overall array alignment if needed
	if totalSize%elementAlign != 0 {
		totalSize = alignUp(totalSize, elementAlign)
	}

	layout := &ArrayLayout{
		ElementType:  elementType,
		ElementSize:  elementSize,
		ElementAlign: elementAlign,
		Length:       length,
		TotalSize:    totalSize,
		PaddingMap:   []int64{}, // Arrays typically don't need internal padding
	}

	return layout, nil
}

// CalculateSliceLayout calculates the memory layout for a dynamic slice
func (lc *LayoutCalculator) CalculateSliceLayout(elementType string, elementSize, elementAlign int64) (*SliceLayout, error) {
	if elementSize <= 0 {
		return nil, fmt.Errorf("element size must be positive: %d", elementSize)
	}
	if elementAlign <= 0 {
		elementAlign = 1
	}

	layout := &SliceLayout{
		ElementType:  elementType,
		ElementSize:  elementSize,
		ElementAlign: elementAlign,
		PtrOffset:    0,
		LenOffset:    8,
		CapOffset:    16,
		TotalSize:    24, // 3 * 8 bytes (ptr, len, cap)
	}

	return layout, nil
}

// CalculateStringLayout calculates the memory layout for a string
func (lc *LayoutCalculator) CalculateStringLayout() *StringLayout {
	return &StringLayout{
		PtrOffset: 0,
		LenOffset: 8,
		TotalSize: 16, // 2 * 8 bytes (ptr, len)
	}
}

// CalculateStructLayout calculates the memory layout for a struct
func (lc *LayoutCalculator) CalculateStructLayout(name string, fields []FieldInfo) (*StructLayout, error) {
	if len(fields) == 0 {
		return &StructLayout{
			Name:       name,
			Fields:     []FieldInfo{},
			TotalSize:  0,
			Alignment:  1,
			PaddingMap: []PaddingInfo{},
		}, nil
	}

	var padding []PaddingInfo
	var layoutFields []FieldInfo
	currentOffset := int64(0)
	maxAlignment := int64(1)

	for _, field := range fields {
		if field.Size <= 0 {
			return nil, fmt.Errorf("field %s has invalid size: %d", field.Name, field.Size)
		}
		if field.Alignment <= 0 {
			field.Alignment = 1
		}

		// Track maximum alignment requirement
		if field.Alignment > maxAlignment {
			maxAlignment = field.Alignment
		}

		// Add padding for field alignment
		alignedOffset := alignUp(currentOffset, field.Alignment)
		if alignedOffset > currentOffset {
			padding = append(padding, PaddingInfo{
				Offset: currentOffset,
				Size:   alignedOffset - currentOffset,
				Reason: fmt.Sprintf("alignment for field %s", field.Name),
			})
		}

		// Create field info with calculated offset
		layoutField := FieldInfo{
			Name:      field.Name,
			Type:      field.Type,
			Offset:    alignedOffset,
			Size:      field.Size,
			Alignment: field.Alignment,
		}
		layoutFields = append(layoutFields, layoutField)

		currentOffset = alignedOffset + field.Size
	}

	// Add final padding for struct alignment
	totalSize := alignUp(currentOffset, maxAlignment)
	if totalSize > currentOffset {
		padding = append(padding, PaddingInfo{
			Offset: currentOffset,
			Size:   totalSize - currentOffset,
			Reason: "struct alignment",
		})
	}

	layout := &StructLayout{
		Name:       name,
		Fields:     layoutFields,
		TotalSize:  totalSize,
		Alignment:  maxAlignment,
		PaddingMap: padding,
	}

	return layout, nil
}

// GetMemoryLayout returns a generic MemoryLayout for any data type
func (lc *LayoutCalculator) GetMemoryLayout(kind LayoutKind, params map[string]interface{}) (*MemoryLayout, error) {
	switch kind {
	case LayoutArray:
		elementType, _ := params["elementType"].(string)
		elementSize, _ := params["elementSize"].(int64)
		elementAlign, _ := params["elementAlign"].(int64)
		length, _ := params["length"].(int64)

		arrayLayout, err := lc.CalculateArrayLayout(elementType, elementSize, elementAlign, length)
		if err != nil {
			return nil, err
		}

		return &MemoryLayout{
			Kind:        LayoutArray,
			Size:        arrayLayout.TotalSize,
			Alignment:   arrayLayout.ElementAlign,
			ElementSize: arrayLayout.ElementSize,
		}, nil

	case LayoutSlice:
		elementType, _ := params["elementType"].(string)
		elementSize, _ := params["elementSize"].(int64)
		elementAlign, _ := params["elementAlign"].(int64)

		sliceLayout, err := lc.CalculateSliceLayout(elementType, elementSize, elementAlign)
		if err != nil {
			return nil, err
		}

		return &MemoryLayout{
			Kind:        LayoutSlice,
			Size:        sliceLayout.TotalSize,
			Alignment:   8, // Pointer alignment
			ElementSize: sliceLayout.ElementSize,
		}, nil

	case LayoutString:
		stringLayout := lc.CalculateStringLayout()
		return &MemoryLayout{
			Kind:        LayoutString,
			Size:        stringLayout.TotalSize,
			Alignment:   8, // Pointer alignment
			ElementSize: 1, // Bytes
		}, nil

	case LayoutPointer, LayoutReference:
		return &MemoryLayout{
			Kind:        kind,
			Size:        lc.TargetPointerSize,
			Alignment:   lc.TargetPointerSize,
			ElementSize: 0,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported layout kind: %v", kind)
	}
}

// Utility functions

// isPowerOfTwo checks if a number is a power of 2
func isPowerOfTwo(n int64) bool {
	return n > 0 && (n&(n-1)) == 0
}

// alignUp rounds up to the next multiple of alignment
func alignUp(value, alignment int64) int64 {
	if alignment <= 1 {
		return value
	}
	return (value + alignment - 1) & ^(alignment - 1)
}

// ABI-specific functions

// GetFieldOffset returns the byte offset of a field within a struct
func (sl *StructLayout) GetFieldOffset(fieldName string) (int64, bool) {
	for _, field := range sl.Fields {
		if field.Name == fieldName {
			return field.Offset, true
		}
	}
	return 0, false
}

// GetPaddingBytes returns the total number of padding bytes in the struct
func (sl *StructLayout) GetPaddingBytes() int64 {
	var total int64
	for _, pad := range sl.PaddingMap {
		total += pad.Size
	}
	return total
}

// GetEfficiencyRatio returns the ratio of useful data to total size
func (sl *StructLayout) GetEfficiencyRatio() float64 {
	if sl.TotalSize == 0 {
		return 1.0
	}

	var usefulBytes int64
	for _, field := range sl.Fields {
		usefulBytes += field.Size
	}

	return float64(usefulBytes) / float64(sl.TotalSize)
}

// String representations for debugging

func (al *ArrayLayout) String() string {
	return fmt.Sprintf("Array[%s; %d] (element: %d bytes, total: %d bytes, align: %d)",
		al.ElementType, al.Length, al.ElementSize, al.TotalSize, al.ElementAlign)
}

func (sl *SliceLayout) String() string {
	return fmt.Sprintf("Slice<%s> (element: %d bytes, header: %d bytes, align: %d)",
		sl.ElementType, sl.ElementSize, sl.TotalSize, sl.ElementAlign)
}

func (stl *StringLayout) String() string {
	return fmt.Sprintf("String (header: %d bytes)", stl.TotalSize)
}

func (sl *StructLayout) String() string {
	return fmt.Sprintf("Struct %s (%d fields, %d bytes, %d padding, %.1f%% efficiency)",
		sl.Name, len(sl.Fields), sl.TotalSize, sl.GetPaddingBytes(), sl.GetEfficiencyRatio()*100)
}

// Runtime type information helpers

// GetArrayElementAddress calculates the address of an array element
func (al *ArrayLayout) GetArrayElementAddress(baseAddr uintptr, index int64) uintptr {
	if index < 0 || index >= al.Length {
		return 0 // Invalid index
	}
	return baseAddr + uintptr(index*al.ElementSize)
}

// GetSliceElementAddress calculates the address of a slice element
func (sl *SliceLayout) GetSliceElementAddress(sliceHeaderAddr uintptr, index int64) uintptr {
	// Read the data pointer from the slice header
	dataPtr := *(*uintptr)(unsafe.Pointer(sliceHeaderAddr + uintptr(sl.PtrOffset)))
	return dataPtr + uintptr(index*sl.ElementSize)
}

// GetStringByteAddress calculates the address of a string byte
func (stl *StringLayout) GetStringByteAddress(stringHeaderAddr uintptr, index int64) uintptr {
	// Read the data pointer from the string header
	dataPtr := *(*uintptr)(unsafe.Pointer(stringHeaderAddr + uintptr(stl.PtrOffset)))
	return dataPtr + uintptr(index)
}
