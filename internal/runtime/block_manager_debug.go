//go:build debug

package runtime

import (
	"fmt"
	"unsafe"
)

// In debug builds, enforce strict validation after allocation and before free.

func debugPostAllocValidate(bm *BlockManager, userPtr unsafe.Pointer, size RegionSize) {
	if userPtr == nil || size == 0 {
		panic("debug: invalid allocation state")
	}
	// Verify header presence and map registration
	bm.mutex.RLock()
	hdr, ok := bm.blockMap[userPtr]
	bm.mutex.RUnlock()
	if !ok || hdr == nil {
		panic(fmt.Sprintf("debug: header not registered for %p", userPtr))
	}
	if hdr.Magic != BlockMagicValue {
		panic("debug: magic mismatch after alloc")
	}
	if hdr.Size == 0 || RegionSize(hdr.Size) < size {
		panic("debug: header size inconsistent after alloc")
	}
}

func debugStrictCanaryCheck(bm *BlockManager, ptr unsafe.Pointer, size RegionSize) {
	if !bm.policy.EnableCanaries {
		// Force canary write/validate in debug regardless of policy
		bm.writeCanaries(ptr, size)
	}
	if !bm.validateCanaries(ptr, size) {
		panic("debug: canary validation failed")
	}
}
