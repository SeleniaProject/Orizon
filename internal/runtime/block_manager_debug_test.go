//go:build debug.

package runtime

import (
	"testing"
	"unsafe"
)

// This test verifies that debugPostAllocValidate detects unregistered pointers.
// and panics in debug builds, helping catch allocation path inconsistencies early.
func TestDebugPostAllocValidate_PanicsOnUnregistered(t *testing.T) {
	bm := NewBlockManager(BlockPolicy{})
	var x int
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic from debugPostAllocValidate for unregistered pointer")
		}
	}()
	debugPostAllocValidate(bm, unsafe.Pointer(&x), 16)
}
