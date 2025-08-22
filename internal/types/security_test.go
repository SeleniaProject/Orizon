package types

import (
	"strings"
	"testing"
	"unsafe"
)

// TestOrizonSlice_OverflowProtection tests comprehensive overflow protection
func TestOrizonSlice_OverflowProtection(t *testing.T) {
	tests := []struct {
		name        string
		setupSlice  func() *OrizonSlice
		index       uintptr
		expectPanic bool
		panicMsg    string
	}{
		{
			name: "multiplication_overflow_max_index",
			setupSlice: func() *OrizonSlice {
				data := make([]byte, 10)
				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   ^uintptr(0), // Max length
					capacity: ^uintptr(0),
					typeInfo: &TypeInfo{Size: 2}, // Will cause overflow
				}
			},
			index:       ^uintptr(0)/2 + 1,
			expectPanic: true,
			panicMsg:    "INTEGER_OVERFLOW",
		},
		{
			name: "pointer_addition_overflow",
			setupSlice: func() *OrizonSlice {
				// Use a more realistic test scenario
				data := make([]byte, 10)
				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   1000,
					capacity: 1000,
					typeInfo: &TypeInfo{Size: ^uintptr(0)}, // Max size will cause overflow
				}
			},
			index:       2, // Small index but max size will overflow
			expectPanic: true,
			panicMsg:    "INTEGER_OVERFLOW", // This will be caught by multiplication check first
		},
		{
			name: "zero_element_size_protection",
			setupSlice: func() *OrizonSlice {
				data := make([]byte, 10)
				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 0}, // Zero size
				}
			},
			index:       5,
			expectPanic: true,
			panicMsg:    "INVALID_SIZE",
		},
		{
			name: "valid_access_no_panic",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)
				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			index:       5,
			expectPanic: false,
			panicMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := tt.setupSlice()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						panicStr := strings.ToUpper(r.(error).Error())
						if !strings.Contains(panicStr, tt.panicMsg) {
							t.Errorf("Expected panic message containing %q, got %q", tt.panicMsg, r)
						}
					} else {
						t.Error("Expected panic but none occurred")
					}
				}()
			}

			slice.Get(tt.index)
		})
	}
}

// TestOrizonVec_OverflowProtection tests vector overflow protection
func TestOrizonVec_OverflowProtection(t *testing.T) {
	tests := []struct {
		name        string
		setupVec    func() *OrizonVec
		index       uintptr
		expectPanic bool
		panicMsg    string
	}{
		{
			name: "vector_multiplication_overflow",
			setupVec: func() *OrizonVec {
				data := make([]byte, 10)
				return &OrizonVec{
					data:     unsafe.Pointer(&data[0]),
					length:   ^uintptr(0), // Max length
					capacity: ^uintptr(0),
					typeInfo: &TypeInfo{Size: 2},
				}
			},
			index:       ^uintptr(0)/2 + 1,
			expectPanic: true,
			panicMsg:    "INTEGER_OVERFLOW",
		},
		{
			name: "vector_valid_access",
			setupVec: func() *OrizonVec {
				data := make([]int32, 10)
				return &OrizonVec{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					capacity: 10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			index:       5,
			expectPanic: false,
			panicMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vec := tt.setupVec()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						panicStr := strings.ToUpper(r.(error).Error())
						if !strings.Contains(panicStr, tt.panicMsg) {
							t.Errorf("Expected panic message containing %q, got %q", tt.panicMsg, r)
						}
					} else {
						t.Error("Expected panic but none occurred")
					}
				}()
			}

			vec.Get(tt.index)
		})
	}
}

// BenchmarkOrizonSlice_SafeAccess benchmarks the performance impact of safety checks
func BenchmarkOrizonSlice_SafeAccess(b *testing.B) {
	data := make([]int32, 1000)
	slice := &OrizonSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   1000,
		capacity: 1000,
		typeInfo: &TypeInfo{Size: 4},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice.Get(uintptr(i % 1000))
	}
}
