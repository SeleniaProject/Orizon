package runtime

import "unsafe"

// This file provides no-op debug hooks for non-debug builds.

// debugPostAllocValidate is invoked after a block allocation to perform
// additional validation in debug builds. No-op in normal builds.
func debugPostAllocValidate(bm *BlockManager, userPtr unsafe.Pointer, size RegionSize) {}

// debugStrictCanaryCheck enforces canary validation regardless of policy
// in debug builds. No-op in normal builds.
func debugStrictCanaryCheck(bm *BlockManager, ptr unsafe.Pointer, size RegionSize) {}
