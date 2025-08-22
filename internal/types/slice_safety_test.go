package types

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/errors"
)

func TestOrizonSlice_SafetyValidation(t *testing.T) {
	tests := []struct {
		setupSlice  func() *OrizonSlice
		name        string
		expectedMsg string
		index       uintptr
		expectPanic bool
	}{
		{
			name: "valid access within bounds",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			index:       5,
			expectPanic: false,
		},
		{
			name: "index out of bounds",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			index:       15,
			expectPanic: true,
			expectedMsg: "Index 15 out of bounds for length 10",
		},
		{
			name: "nil data pointer",
			setupSlice: func() *OrizonSlice {
				return &OrizonSlice{
					data:     nil,
					length:   10,
					typeInfo: &TypeInfo{Size: 4},
				}
			},
			index:       5,
			expectPanic: true,
			expectedMsg: "Null pointer dereference in slice.data",
		},
		{
			name: "nil typeInfo",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					typeInfo: nil,
				}
			},
			index:       5,
			expectPanic: true,
			expectedMsg: "Null pointer dereference in slice.typeInfo",
		},
		{
			name: "zero element size",
			setupSlice: func() *OrizonSlice {
				data := make([]int32, 10)

				return &OrizonSlice{
					data:     unsafe.Pointer(&data[0]),
					length:   10,
					typeInfo: &TypeInfo{Size: 0},
				}
			},
			index:       5,
			expectPanic: true,
			expectedMsg: "Invalid size 0 in slice element size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := tt.setupSlice()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						var panicMsg string
						// Handle both string and StandardError types for better error compatibility
						if errObj, ok := r.(*errors.StandardError); ok {
							panicMsg = errObj.Error()
						} else if str, ok := r.(string); ok {
							panicMsg = str
						} else {
							panicMsg = fmt.Sprintf("%v", r)
						}
						if tt.expectedMsg != "" && !strings.Contains(panicMsg, tt.expectedMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tt.expectedMsg, panicMsg)
						}
					} else {
						t.Errorf("expected panic but got none")
					}
				}()
				slice.Get(tt.index)
				t.Errorf("expected panic but function returned normally")
			} else {
				ptr := slice.Get(tt.index)
				if ptr == nil {
					t.Errorf("expected non-nil pointer for valid access")
				}
			}
		})
	}
}

func TestOrizonSlice_SetSafetyValidation(t *testing.T) {
	data := make([]int32, 10)
	slice := &OrizonSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   10,
		typeInfo: &TypeInfo{Size: 4},
	}

	// Test nil value pointer.
	defer func() {
		if r := recover(); r != nil {
			var panicMsg string
			// Handle both string and StandardError types for better error compatibility
			if errObj, ok := r.(*errors.StandardError); ok {
				panicMsg = errObj.Error()
			} else if str, ok := r.(string); ok {
				panicMsg = str
			} else {
				panicMsg = fmt.Sprintf("%v", r)
			}
			if !strings.Contains(panicMsg, "Null value pointer") {
				t.Errorf("expected panic message to contain 'Null value pointer', got %q", panicMsg)
			}
		} else {
			t.Errorf("expected panic for nil value pointer")
		}
	}()
	slice.Set(5, nil)
}

func TestOrizonSlice_ConcurrentAccess(t *testing.T) {
	// Test for race conditions in concurrent access.
	data := make([]int32, 1000)
	slice := &OrizonSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   1000,
		typeInfo: &TypeInfo{Size: 4},
	}

	const numGoroutines = 100

	const numOperations = 1000

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				index := uintptr(j % 1000)
				_ = slice.Get(index)
			}
		}(i)
	}

	// Wait for all goroutines to complete.
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func BenchmarkOrizonSlice_SafeGet(b *testing.B) {
	data := make([]int32, 10000)
	slice := &OrizonSlice{
		data:     unsafe.Pointer(&data[0]),
		length:   10000,
		typeInfo: &TypeInfo{Size: 4},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		index := uintptr(i % 10000)
		_ = slice.Get(index)
	}
}
