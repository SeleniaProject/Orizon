package layout

import (
	"testing"
	"unsafe"
)

func TestLayoutCalculator(t *testing.T) {
	lc := NewLayoutCalculator()

	if lc.TargetPointerSize != 8 {
		t.Errorf("Expected pointer size 8, got %d", lc.TargetPointerSize)
	}

	if lc.MaxAlignment != 16 {
		t.Errorf("Expected max alignment 16, got %d", lc.MaxAlignment)
	}
}

func TestArrayLayout(t *testing.T) {
	lc := NewLayoutCalculator()

	tests := []struct {
		name         string
		elementType  string
		elementSize  int64
		elementAlign int64
		length       int64
		expectedSize int64
		shouldError  bool
	}{
		{
			name:         "int32_array",
			elementType:  "i32",
			elementSize:  4,
			elementAlign: 4,
			length:       10,
			expectedSize: 40,
			shouldError:  false,
		},
		{
			name:         "int64_array",
			elementType:  "i64",
			elementSize:  8,
			elementAlign: 8,
			length:       5,
			expectedSize: 40,
			shouldError:  false,
		},
		{
			name:         "char_array",
			elementType:  "char",
			elementSize:  1,
			elementAlign: 1,
			length:       16,
			expectedSize: 16,
			shouldError:  false,
		},
		{
			name:         "zero_length_array",
			elementType:  "i32",
			elementSize:  4,
			elementAlign: 4,
			length:       0,
			expectedSize: 0,
			shouldError:  false,
		},
		{
			name:         "negative_length",
			elementType:  "i32",
			elementSize:  4,
			elementAlign: 4,
			length:       -1,
			expectedSize: 0,
			shouldError:  true,
		},
		{
			name:         "invalid_element_size",
			elementType:  "void",
			elementSize:  0,
			elementAlign: 1,
			length:       5,
			expectedSize: 0,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := lc.CalculateArrayLayout(tt.elementType, tt.elementSize, tt.elementAlign, tt.length)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got none", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error for test %s: %v", tt.name, err)

				return
			}

			if layout.TotalSize != tt.expectedSize {
				t.Errorf("Expected size %d, got %d for test %s", tt.expectedSize, layout.TotalSize, tt.name)
			}

			if layout.ElementType != tt.elementType {
				t.Errorf("Expected element type %s, got %s", tt.elementType, layout.ElementType)
			}

			if layout.Length != tt.length {
				t.Errorf("Expected length %d, got %d", tt.length, layout.Length)
			}
		})
	}
}

func TestSliceLayout(t *testing.T) {
	lc := NewLayoutCalculator()

	tests := []struct {
		name         string
		elementType  string
		elementSize  int64
		elementAlign int64
		shouldError  bool
	}{
		{
			name:         "int32_slice",
			elementType:  "i32",
			elementSize:  4,
			elementAlign: 4,
			shouldError:  false,
		},
		{
			name:         "int64_slice",
			elementType:  "i64",
			elementSize:  8,
			elementAlign: 8,
			shouldError:  false,
		},
		{
			name:         "char_slice",
			elementType:  "char",
			elementSize:  1,
			elementAlign: 1,
			shouldError:  false,
		},
		{
			name:         "invalid_element_size",
			elementType:  "void",
			elementSize:  0,
			elementAlign: 1,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := lc.CalculateSliceLayout(tt.elementType, tt.elementSize, tt.elementAlign)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got none", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error for test %s: %v", tt.name, err)

				return
			}

			// All slices should have the same header size.
			expectedSize := int64(24) // ptr(8) + len(8) + cap(8)
			if layout.TotalSize != expectedSize {
				t.Errorf("Expected slice header size %d, got %d", expectedSize, layout.TotalSize)
			}

			// Check field offsets.
			if layout.PtrOffset != 0 {
				t.Errorf("Expected ptr offset 0, got %d", layout.PtrOffset)
			}

			if layout.LenOffset != 8 {
				t.Errorf("Expected len offset 8, got %d", layout.LenOffset)
			}

			if layout.CapOffset != 16 {
				t.Errorf("Expected cap offset 16, got %d", layout.CapOffset)
			}

			if layout.ElementType != tt.elementType {
				t.Errorf("Expected element type %s, got %s", tt.elementType, layout.ElementType)
			}
		})
	}
}

func TestStringLayout(t *testing.T) {
	lc := NewLayoutCalculator()
	layout := lc.CalculateStringLayout()

	expectedSize := int64(16) // ptr(8) + len(8)
	if layout.TotalSize != expectedSize {
		t.Errorf("Expected string header size %d, got %d", expectedSize, layout.TotalSize)
	}

	if layout.PtrOffset != 0 {
		t.Errorf("Expected ptr offset 0, got %d", layout.PtrOffset)
	}

	if layout.LenOffset != 8 {
		t.Errorf("Expected len offset 8, got %d", layout.LenOffset)
	}
}

func TestStructLayout(t *testing.T) {
	lc := NewLayoutCalculator()

	tests := []struct {
		name          string
		fields        []FieldInfo
		expectedSize  int64
		expectedAlign int64
		expectError   bool
	}{
		{
			name: "simple_struct",
			fields: []FieldInfo{
				{Name: "a", Type: "i32", Size: 4, Alignment: 4},
				{Name: "b", Type: "i32", Size: 4, Alignment: 4},
			},
			expectedSize:  8,
			expectedAlign: 4,
			expectError:   false,
		},
		{
			name: "mixed_alignment_struct",
			fields: []FieldInfo{
				{Name: "a", Type: "i8", Size: 1, Alignment: 1},
				{Name: "b", Type: "i32", Size: 4, Alignment: 4},
				{Name: "c", Type: "i8", Size: 1, Alignment: 1},
			},
			expectedSize:  12, // 1 + 3(pad) + 4 + 1 + 3(pad) = 12
			expectedAlign: 4,
			expectError:   false,
		},
		{
			name: "pointer_struct",
			fields: []FieldInfo{
				{Name: "ptr", Type: "*i32", Size: 8, Alignment: 8},
				{Name: "len", Type: "i64", Size: 8, Alignment: 8},
			},
			expectedSize:  16,
			expectedAlign: 8,
			expectError:   false,
		},
		{
			name:          "empty_struct",
			fields:        []FieldInfo{},
			expectedSize:  0,
			expectedAlign: 1,
			expectError:   false,
		},
		{
			name: "invalid_field_size",
			fields: []FieldInfo{
				{Name: "invalid", Type: "void", Size: 0, Alignment: 1},
			},
			expectedSize:  0,
			expectedAlign: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := lc.CalculateStructLayout(tt.name, tt.fields)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got none", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error for test %s: %v", tt.name, err)

				return
			}

			if layout.TotalSize != tt.expectedSize {
				t.Errorf("Expected size %d, got %d for test %s", tt.expectedSize, layout.TotalSize, tt.name)
			}

			if layout.Alignment != tt.expectedAlign {
				t.Errorf("Expected alignment %d, got %d for test %s", tt.expectedAlign, layout.Alignment, tt.name)
			}

			// Verify field offsets are correctly aligned.
			for i, field := range layout.Fields {
				expectedOffset := alignUp(getPreviousFieldsSize(layout.Fields, i), field.Alignment)
				if field.Offset != expectedOffset {
					t.Errorf("Field %s: expected offset %d, got %d", field.Name, expectedOffset, field.Offset)
				}
			}
		})
	}
}

func TestMemoryLayout(t *testing.T) {
	lc := NewLayoutCalculator()

	tests := []struct {
		params   map[string]interface{}
		expected *MemoryLayout
		name     string
		kind     LayoutKind
		hasError bool
	}{
		{
			name: "array_layout",
			kind: LayoutArray,
			params: map[string]interface{}{
				"elementType":  "i32",
				"elementSize":  int64(4),
				"elementAlign": int64(4),
				"length":       int64(10),
			},
			expected: &MemoryLayout{
				Kind:        LayoutArray,
				Size:        40,
				Alignment:   4,
				ElementSize: 4,
			},
			hasError: false,
		},
		{
			name: "slice_layout",
			kind: LayoutSlice,
			params: map[string]interface{}{
				"elementType":  "i64",
				"elementSize":  int64(8),
				"elementAlign": int64(8),
			},
			expected: &MemoryLayout{
				Kind:        LayoutSlice,
				Size:        24,
				Alignment:   8,
				ElementSize: 8,
			},
			hasError: false,
		},
		{
			name:   "string_layout",
			kind:   LayoutString,
			params: map[string]interface{}{},
			expected: &MemoryLayout{
				Kind:        LayoutString,
				Size:        16,
				Alignment:   8,
				ElementSize: 1,
			},
			hasError: false,
		},
		{
			name:   "pointer_layout",
			kind:   LayoutPointer,
			params: map[string]interface{}{},
			expected: &MemoryLayout{
				Kind:        LayoutPointer,
				Size:        8,
				Alignment:   8,
				ElementSize: 0,
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := lc.GetMemoryLayout(tt.kind, tt.params)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for test %s, but got none", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error for test %s: %v", tt.name, err)

				return
			}

			if layout.Kind != tt.expected.Kind {
				t.Errorf("Expected kind %v, got %v", tt.expected.Kind, layout.Kind)
			}

			if layout.Size != tt.expected.Size {
				t.Errorf("Expected size %d, got %d", tt.expected.Size, layout.Size)
			}

			if layout.Alignment != tt.expected.Alignment {
				t.Errorf("Expected alignment %d, got %d", tt.expected.Alignment, layout.Alignment)
			}

			if layout.ElementSize != tt.expected.ElementSize {
				t.Errorf("Expected element size %d, got %d", tt.expected.ElementSize, layout.ElementSize)
			}
		})
	}
}

func TestStructLayoutUtilities(t *testing.T) {
	lc := NewLayoutCalculator()

	fields := []FieldInfo{
		{Name: "a", Type: "i8", Size: 1, Alignment: 1},
		{Name: "b", Type: "i32", Size: 4, Alignment: 4},
		{Name: "c", Type: "i8", Size: 1, Alignment: 1},
	}

	layout, err := lc.CalculateStructLayout("TestStruct", fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test GetFieldOffset.
	offsetA, foundA := layout.GetFieldOffset("a")
	if !foundA || offsetA != 0 {
		t.Errorf("Expected field 'a' at offset 0, got %d (found: %v)", offsetA, foundA)
	}

	offsetB, foundB := layout.GetFieldOffset("b")
	if !foundB || offsetB != 4 {
		t.Errorf("Expected field 'b' at offset 4, got %d (found: %v)", offsetB, foundB)
	}

	offsetC, foundC := layout.GetFieldOffset("c")
	if !foundC || offsetC != 8 {
		t.Errorf("Expected field 'c' at offset 8, got %d (found: %v)", offsetC, foundC)
	}

	// Test non-existent field.
	_, foundD := layout.GetFieldOffset("d")
	if foundD {
		t.Error("Should not find non-existent field 'd'")
	}

	// Test GetPaddingBytes.
	paddingBytes := layout.GetPaddingBytes()
	expectedPadding := int64(6) // 3 bytes after 'a', 3 bytes after 'c'

	if paddingBytes != expectedPadding {
		t.Errorf("Expected %d padding bytes, got %d", expectedPadding, paddingBytes)
	}

	// Test GetEfficiencyRatio.
	ratio := layout.GetEfficiencyRatio()
	expectedRatio := float64(6) / float64(12) // 6 useful bytes out of 12 total

	if ratio != expectedRatio {
		t.Errorf("Expected efficiency ratio %.2f, got %.2f", expectedRatio, ratio)
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test isPowerOfTwo.
	tests := []struct {
		input    int64
		expected bool
	}{
		{1, true},
		{2, true},
		{4, true},
		{8, true},
		{16, true},
		{3, false},
		{5, false},
		{7, false},
		{0, false},
		{-1, false},
	}

	for _, tt := range tests {
		result := isPowerOfTwo(tt.input)
		if result != tt.expected {
			t.Errorf("isPowerOfTwo(%d): expected %v, got %v", tt.input, tt.expected, result)
		}
	}

	// Test alignUp.
	alignTests := []struct {
		value     int64
		alignment int64
		expected  int64
	}{
		{1, 1, 1},
		{1, 2, 2},
		{1, 4, 4},
		{5, 4, 8},
		{8, 4, 8},
		{9, 4, 12},
		{0, 8, 0},
	}

	for _, tt := range alignTests {
		result := alignUp(tt.value, tt.alignment)
		if result != tt.expected {
			t.Errorf("alignUp(%d, %d): expected %d, got %d", tt.value, tt.alignment, tt.expected, result)
		}
	}
}

func TestStringRepresentations(t *testing.T) {
	lc := NewLayoutCalculator()

	// Test ArrayLayout.String()
	arrayLayout, _ := lc.CalculateArrayLayout("i32", 4, 4, 10)
	arrayStr := arrayLayout.String()
	expected := "Array[i32; 10] (element: 4 bytes, total: 40 bytes, align: 4)"

	if arrayStr != expected {
		t.Errorf("ArrayLayout.String(): expected %q, got %q", expected, arrayStr)
	}

	// Test SliceLayout.String()
	sliceLayout, _ := lc.CalculateSliceLayout("i64", 8, 8)
	sliceStr := sliceLayout.String()
	expected = "Slice<i64> (element: 8 bytes, header: 24 bytes, align: 8)"

	if sliceStr != expected {
		t.Errorf("SliceLayout.String(): expected %q, got %q", expected, sliceStr)
	}

	// Test StringLayout.String()
	stringLayout := lc.CalculateStringLayout()
	stringStr := stringLayout.String()
	expected = "String (header: 16 bytes)"

	if stringStr != expected {
		t.Errorf("StringLayout.String(): expected %q, got %q", expected, stringStr)
	}
}

func TestRuntimeAddressCalculation(t *testing.T) {
	lc := NewLayoutCalculator()

	// Test array element address calculation.
	arrayLayout, _ := lc.CalculateArrayLayout("i32", 4, 4, 10)
	baseAddr := uintptr(0x1000)

	// Valid indices.
	addr0 := arrayLayout.GetArrayElementAddress(baseAddr, 0)
	if addr0 != baseAddr {
		t.Errorf("Expected element 0 at 0x%x, got 0x%x", baseAddr, addr0)
	}

	addr5 := arrayLayout.GetArrayElementAddress(baseAddr, 5)
	expected := baseAddr + uintptr(5*4)

	if addr5 != expected {
		t.Errorf("Expected element 5 at 0x%x, got 0x%x", expected, addr5)
	}

	// Invalid indices.
	addrNeg := arrayLayout.GetArrayElementAddress(baseAddr, -1)
	if addrNeg != 0 {
		t.Errorf("Expected 0 for negative index, got 0x%x", addrNeg)
	}

	addrOOB := arrayLayout.GetArrayElementAddress(baseAddr, 10)
	if addrOOB != 0 {
		t.Errorf("Expected 0 for out-of-bounds index, got 0x%x", addrOOB)
	}
}

// Helper function for struct layout tests.
func getPreviousFieldsSize(fields []FieldInfo, currentIndex int) int64 {
	if currentIndex == 0 {
		return 0
	}

	var size int64

	for i := 0; i < currentIndex; i++ {
		fieldEnd := fields[i].Offset + fields[i].Size
		if fieldEnd > size {
			size = fieldEnd
		}
	}

	return size
}

func BenchmarkArrayLayoutCalculation(b *testing.B) {
	lc := NewLayoutCalculator()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lc.CalculateArrayLayout("i32", 4, 4, 1000)
	}
}

func BenchmarkStructLayoutCalculation(b *testing.B) {
	lc := NewLayoutCalculator()

	fields := []FieldInfo{
		{Name: "a", Type: "i8", Size: 1, Alignment: 1},
		{Name: "b", Type: "i32", Size: 4, Alignment: 4},
		{Name: "c", Type: "i64", Size: 8, Alignment: 8},
		{Name: "d", Type: "i16", Size: 2, Alignment: 2},
		{Name: "e", Type: "i8", Size: 1, Alignment: 1},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = lc.CalculateStructLayout("BenchStruct", fields)
	}
}

// Integration test with actual memory representation.
func TestLayoutIntegration(t *testing.T) {
	lc := NewLayoutCalculator()

	// Create a struct that matches Go's struct layout for comparison.
	type TestStruct struct {
		B int32
		A int8
		C int8
	}

	fields := []FieldInfo{
		{Name: "A", Type: "i8", Size: 1, Alignment: 1},
		{Name: "B", Type: "i32", Size: 4, Alignment: 4},
		{Name: "C", Type: "i8", Size: 1, Alignment: 1},
	}

	layout, err := lc.CalculateStructLayout("TestStruct", fields)
	if err != nil {
		t.Fatalf("Failed to calculate layout: %v", err)
	}

	// Compare with Go's actual struct size.
	goSize := unsafe.Sizeof(TestStruct{})
	if layout.TotalSize != int64(goSize) {
		t.Errorf("Layout size %d doesn't match Go struct size %d", layout.TotalSize, goSize)
	}

	// Check field offsets match Go's offsets.
	var ts TestStruct
	offsetA := unsafe.Offsetof(ts.A)
	offsetB := unsafe.Offsetof(ts.B)
	offsetC := unsafe.Offsetof(ts.C)

	layoutOffsetA, _ := layout.GetFieldOffset("A")
	layoutOffsetB, _ := layout.GetFieldOffset("B")
	layoutOffsetC, _ := layout.GetFieldOffset("C")

	if layoutOffsetA != int64(offsetA) {
		t.Errorf("Field A offset: expected %d, got %d", offsetA, layoutOffsetA)
	}

	if layoutOffsetB != int64(offsetB) {
		t.Errorf("Field B offset: expected %d, got %d", offsetB, layoutOffsetB)
	}

	if layoutOffsetC != int64(offsetC) {
		t.Errorf("Field C offset: expected %d, got %d", offsetC, layoutOffsetC)
	}
}
