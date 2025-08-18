// Package runtime provides memory allocation operations within regions.
// This module implements allocate/deallocate operations, block management,
// and memory layout optimization for region-based memory management.
package runtime

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// AllocationError represents allocation-related errors
type AllocationError struct {
	Message   string
	Code      ErrorCode
	Region    *Region
	Size      RegionSize
	Alignment RegionAlignment
}

// ErrorCode represents allocation error types
type ErrorCode int

const (
	ErrorOutOfMemory      ErrorCode = iota // Out of memory
	ErrorInvalidSize                       // Invalid allocation size
	ErrorInvalidAlign                      // Invalid alignment
	ErrorRegionCorrupt                     // Region corruption
	ErrorDoubleAlloc                       // Double allocation
	ErrorDoubleFree                        // Double free
	ErrorUseAfterFree                      // Use after free
	ErrorHeapOverflow                      // Heap overflow
	ErrorStackOverflow                     // Stack overflow
	ErrorPermissionDenied                  // Permission denied
)

// String returns string representation of allocation error
func (ae *AllocationError) String() string {
	return fmt.Sprintf("AllocationError[%s]: %s (region=%d, size=%d, align=%d)",
		ae.Code.String(), ae.Message, ae.Region.Header.ID, ae.Size, ae.Alignment)
}

// Error implements error interface
func (ae *AllocationError) Error() string {
	return ae.String()
}

// String returns string representation of error code
func (ec ErrorCode) String() string {
	switch ec {
	case ErrorOutOfMemory:
		return "OutOfMemory"
	case ErrorInvalidSize:
		return "InvalidSize"
	case ErrorInvalidAlign:
		return "InvalidAlign"
	case ErrorRegionCorrupt:
		return "RegionCorrupt"
	case ErrorDoubleAlloc:
		return "DoubleAlloc"
	case ErrorDoubleFree:
		return "DoubleFree"
	case ErrorUseAfterFree:
		return "UseAfterFree"
	case ErrorHeapOverflow:
		return "HeapOverflow"
	case ErrorStackOverflow:
		return "StackOverflow"
	case ErrorPermissionDenied:
		return "PermissionDenied"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ec))
	}
}

// Allocate allocates memory block within the region
func (r *Region) Allocate(size RegionSize, alignment RegionAlignment, typeInfo *TypeInfo) (unsafe.Pointer, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// Validate region state
	if atomic.LoadUint32(&r.Header.State) != RegionActive {
		return nil, &AllocationError{
			Message:   "region not active",
			Code:      ErrorRegionCorrupt,
			Region:    r,
			Size:      size,
			Alignment: alignment,
		}
	}

	// Validate allocation parameters
	if size == 0 {
		return nil, &AllocationError{
			Message:   "zero size allocation",
			Code:      ErrorInvalidSize,
			Region:    r,
			Size:      size,
			Alignment: alignment,
		}
	}

	if alignment == 0 {
		alignment = r.Header.Alignment
	}

	if !isPowerOfTwo(int64(alignment)) {
		return nil, &AllocationError{
			Message:   "alignment must be power of two",
			Code:      ErrorInvalidAlign,
			Region:    r,
			Size:      size,
			Alignment: alignment,
		}
	}

	// Check policy constraints
	if r.Policy != nil {
		if r.Header.AllocCount >= r.Policy.MaxAllocations {
			return nil, &AllocationError{
				Message:   "allocation limit exceeded",
				Code:      ErrorOutOfMemory,
				Region:    r,
				Size:      size,
				Alignment: alignment,
			}
		}

		if r.Header.Used+size > r.Policy.MaxMemoryUsage {
			return nil, &AllocationError{
				Message:   "memory usage limit exceeded",
				Code:      ErrorOutOfMemory,
				Region:    r,
				Size:      size,
				Alignment: alignment,
			}
		}
	}

	// Find suitable free block
	block, err := r.findFreeBlock(size, alignment)
	if err != nil {
		// Try compaction if enabled
		if r.Policy != nil && r.Policy.CompactionPolicy.Enabled {
			err = r.compact()
			if err != nil {
				return nil, err
			}

			// Retry allocation after compaction
			block, err = r.findFreeBlock(size, alignment)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Allocate from free block
	ptr, allocBlock, err := r.allocateFromBlock(block, size, alignment, typeInfo)
	if err != nil {
		return nil, err
	}

	// Update region statistics
	r.Header.AllocCount++
	r.Header.Used += size
	r.Header.Free -= size
	r.Header.LastAccess = getCurrentTimestamp()

	r.Stats.TotalAllocations++
	r.Stats.TotalBytesAllocated += uint64(size)
	if r.Header.Used > RegionSize(r.Stats.PeakUsage) {
		r.Stats.PeakUsage = uint64(r.Header.Used)
	}

	// Update fragmentation ratio
	r.updateFragmentationRatio()

	// Notify observers
	for _, observer := range r.Observers {
		observer.OnAllocation(r, allocBlock)
	}

	return ptr, nil
}

// findFreeBlock finds a suitable free block using allocation strategy
func (r *Region) findFreeBlock(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	strategy := BestFit
	if r.Policy != nil {
		strategy = r.Policy.AllocationStrategy
	}

	switch strategy {
	case FirstFit:
		return r.findFirstFit(size, alignment)
	case BestFit:
		return r.findBestFit(size, alignment)
	case WorstFit:
		return r.findWorstFit(size, alignment)
	case NextFit:
		return r.findNextFit(size, alignment)
	case QuickFit:
		return r.findQuickFit(size, alignment)
	case BuddySystem:
		return r.findBuddyFit(size, alignment)
	default:
		return r.findBestFit(size, alignment)
	}
}

// findFirstFit implements first fit allocation strategy
func (r *Region) findFirstFit(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	current := r.Header.FreeList

	for current != nil {
		alignedOffset := alignUp(int64(current.Offset), int64(alignment))
		alignedSize := RegionSize(alignedOffset-int64(current.Offset)) + size

		if alignedSize <= current.Size {
			return current, nil
		}

		current = current.Next
	}

	return nil, &AllocationError{
		Message:   "no suitable free block found",
		Code:      ErrorOutOfMemory,
		Region:    r,
		Size:      size,
		Alignment: alignment,
	}
}

// findBestFit implements best fit allocation strategy
func (r *Region) findBestFit(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	var bestBlock *FreeBlock
	var bestWaste RegionSize = RegionSize(^uint64(0)) // Max value

	current := r.Header.FreeList

	for current != nil {
		alignedOffset := alignUp(int64(current.Offset), int64(alignment))
		alignedSize := RegionSize(alignedOffset-int64(current.Offset)) + size

		if alignedSize <= current.Size {
			waste := current.Size - alignedSize
			if waste < bestWaste {
				bestWaste = waste
				bestBlock = current
			}
		}

		current = current.Next
	}

	if bestBlock == nil {
		return nil, &AllocationError{
			Message:   "no suitable free block found",
			Code:      ErrorOutOfMemory,
			Region:    r,
			Size:      size,
			Alignment: alignment,
		}
	}

	return bestBlock, nil
}

// findWorstFit implements worst fit allocation strategy
func (r *Region) findWorstFit(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	var worstBlock *FreeBlock
	var worstWaste RegionSize = 0

	current := r.Header.FreeList

	for current != nil {
		alignedOffset := alignUp(int64(current.Offset), int64(alignment))
		alignedSize := RegionSize(alignedOffset-int64(current.Offset)) + size

		if alignedSize <= current.Size {
			waste := current.Size - alignedSize
			if waste > worstWaste {
				worstWaste = waste
				worstBlock = current
			}
		}

		current = current.Next
	}

	if worstBlock == nil {
		return nil, &AllocationError{
			Message:   "no suitable free block found",
			Code:      ErrorOutOfMemory,
			Region:    r,
			Size:      size,
			Alignment: alignment,
		}
	}

	return worstBlock, nil
}

// findNextFit implements next fit allocation strategy
func (r *Region) findNextFit(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	// Start from last allocation point (simplified - using first fit for now)
	return r.findFirstFit(size, alignment)
}

// findQuickFit implements quick fit allocation strategy
func (r *Region) findQuickFit(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	// Quick fit uses segregated free lists for common sizes
	// Simplified implementation using best fit
	return r.findBestFit(size, alignment)
}

// findBuddyFit implements buddy system allocation strategy
func (r *Region) findBuddyFit(size RegionSize, alignment RegionAlignment) (*FreeBlock, error) {
	// Buddy system implementation would require more complex data structures
	// Simplified implementation using best fit
	return r.findBestFit(size, alignment)
}

// allocateFromBlock allocates memory from a free block
func (r *Region) allocateFromBlock(freeBlock *FreeBlock, size RegionSize, alignment RegionAlignment, typeInfo *TypeInfo) (unsafe.Pointer, *AllocBlock, error) {
	// Calculate aligned offset
	alignedOffset := alignUp(freeBlock.Offset, uintptr(alignment))
	alignmentPadding := RegionSize(alignedOffset - freeBlock.Offset)
	totalSize := alignmentPadding + size

	if totalSize > freeBlock.Size {
		return nil, nil, &AllocationError{
			Message:   "block too small after alignment",
			Code:      ErrorOutOfMemory,
			Region:    r,
			Size:      size,
			Alignment: alignment,
		}
	}

	// Create allocation block
	allocBlock := &AllocBlock{
		Size:       size,
		Offset:     alignedOffset,
		Alignment:  alignment,
		TypeInfo:   typeInfo,
		StackTrace: getStackTrace(),
		Timestamp:  getCurrentTimestamp(),
		Next:       r.Header.AllocList,
		Prev:       nil,
	}

	// Link into allocation list
	if r.Header.AllocList != nil {
		r.Header.AllocList.Prev = allocBlock
	}
	r.Header.AllocList = allocBlock

	// Update free block
	remainingSize := freeBlock.Size - totalSize
	if remainingSize > 0 {
		// Split block
		newFreeBlock := &FreeBlock{
			Size:   remainingSize,
			Offset: alignedOffset + uintptr(size),
			Next:   freeBlock.Next,
			Prev:   freeBlock.Prev,
		}

		// Update linked list
		if freeBlock.Prev != nil {
			freeBlock.Prev.Next = newFreeBlock
		} else {
			r.Header.FreeList = newFreeBlock
		}

		if freeBlock.Next != nil {
			freeBlock.Next.Prev = newFreeBlock
		}
	} else {
		// Remove entire block
		if freeBlock.Prev != nil {
			freeBlock.Prev.Next = freeBlock.Next
		} else {
			r.Header.FreeList = freeBlock.Next
		}

		if freeBlock.Next != nil {
			freeBlock.Next.Prev = freeBlock.Prev
		}
	}

	// Calculate memory address
	ptr := unsafe.Pointer(uintptr(r.Data) + alignedOffset)

	// Initialize memory if required by security policy
	if r.Policy != nil && r.Policy.SecurityPolicy.EnableZeroOnFree {
		clearMemory(ptr, int(size))
	}

	return ptr, allocBlock, nil
}

// Deallocate frees a previously allocated memory block
func (r *Region) Deallocate(ptr unsafe.Pointer) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// Validate region state
	if atomic.LoadUint32(&r.Header.State) != RegionActive {
		return &AllocationError{
			Message: "region not active",
			Code:    ErrorRegionCorrupt,
			Region:  r,
		}
	}

	// Find allocation block
	allocBlock := r.findAllocBlock(ptr)
	if allocBlock == nil {
		return &AllocationError{
			Message: "invalid pointer or double free",
			Code:    ErrorDoubleFree,
			Region:  r,
		}
	}

	// Remove from allocation list
	if allocBlock.Prev != nil {
		allocBlock.Prev.Next = allocBlock.Next
	} else {
		r.Header.AllocList = allocBlock.Next
	}

	if allocBlock.Next != nil {
		allocBlock.Next.Prev = allocBlock.Prev
	}

	// Zero memory if required by security policy
	if r.Policy != nil && r.Policy.SecurityPolicy.EnableZeroOnFree {
		clearMemory(ptr, int(allocBlock.Size))
	}

	// Create free block
	freeBlock := &FreeBlock{
		Size:   allocBlock.Size,
		Offset: allocBlock.Offset,
		Next:   nil,
		Prev:   nil,
	}

	// Insert into free list and coalesce
	r.insertFreeBlock(freeBlock)
	r.coalesceFreeBlocks(freeBlock)

	// Update statistics
	r.Header.FreeCount++
	r.Header.Used -= allocBlock.Size
	r.Header.Free += allocBlock.Size
	r.Header.LastAccess = getCurrentTimestamp()

	r.Stats.TotalDeallocations++
	r.Stats.TotalBytesFreed += uint64(allocBlock.Size)

	// Update fragmentation ratio
	r.updateFragmentationRatio()

	// Notify observers
	for _, observer := range r.Observers {
		observer.OnDeallocation(r, allocBlock)
	}

	return nil
}

// findAllocBlock finds an allocation block by pointer
func (r *Region) findAllocBlock(ptr unsafe.Pointer) *AllocBlock {
	offset := uintptr(ptr) - uintptr(r.Data)

	current := r.Header.AllocList
	for current != nil {
		if current.Offset == offset {
			return current
		}
		current = current.Next
	}

	return nil
}

// insertFreeBlock inserts a free block into the free list (sorted by offset)
func (r *Region) insertFreeBlock(block *FreeBlock) {
	if r.Header.FreeList == nil {
		r.Header.FreeList = block
		return
	}

	// Find insertion point
	current := r.Header.FreeList
	var prev *FreeBlock

	for current != nil && current.Offset < block.Offset {
		prev = current
		current = current.Next
	}

	// Insert block
	block.Next = current
	block.Prev = prev

	if prev != nil {
		prev.Next = block
	} else {
		r.Header.FreeList = block
	}

	if current != nil {
		current.Prev = block
	}
}

// coalesceFreeBlocks coalesces adjacent free blocks
func (r *Region) coalesceFreeBlocks(block *FreeBlock) {
	// Coalesce with next block
	if block.Next != nil && block.Offset+uintptr(block.Size) == block.Next.Offset {
		next := block.Next
		block.Size += next.Size
		block.Next = next.Next
		if next.Next != nil {
			next.Next.Prev = block
		}
		next.Coalesced = true
	}

	// Coalesce with previous block
	if block.Prev != nil && block.Prev.Offset+uintptr(block.Prev.Size) == block.Offset {
		prev := block.Prev
		prev.Size += block.Size
		prev.Next = block.Next
		if block.Next != nil {
			block.Next.Prev = prev
		}
		block.Coalesced = true
	}
}

// compact performs memory compaction
func (r *Region) compact() error {
	if r.Policy == nil || !r.Policy.CompactionPolicy.Enabled {
		return fmt.Errorf("compaction disabled")
	}

	// Calculate current fragmentation ratio
	fragRatio := r.calculateFragmentationRatio()
	if fragRatio < r.Policy.CompactionPolicy.ThresholdRatio {
		return nil // No compaction needed
	}

	// Count free blocks
	freeBlockCount := r.countFreeBlocks()
	if freeBlockCount < r.Policy.CompactionPolicy.MinFreeBlocks {
		return nil // Not enough fragmentation
	}

	startTime := getCurrentTimestamp()
	beforeStats := *r.Stats

	// Perform compaction based on strategy
	var err error
	switch r.Policy.CompactionPolicy.Strategy {
	case MarkAndSweep:
		err = r.markAndSweepCompact()
	case CopyingGC:
		err = r.copyingCompact()
	case GenerationalGC:
		err = r.generationalCompact()
	case IncrementalGC:
		err = r.incrementalCompact()
	case ConcurrentGC:
		err = r.concurrentCompact()
	default:
		err = r.markAndSweepCompact()
	}

	endTime := getCurrentTimestamp()
	compactionTime := endTime - startTime

	if err != nil {
		return err
	}

	// Check time limit
	if compactionTime > r.Policy.CompactionPolicy.MaxCompactionTime {
		return fmt.Errorf("compaction exceeded time limit")
	}

	// Update statistics
	r.Stats.LastCompaction = endTime
	r.Stats.CompactionCount++

	// Notify observers
	for _, observer := range r.Observers {
		observer.OnCompaction(r, &beforeStats, r.Stats)
	}

	return nil
}

// markAndSweepCompact implements mark and sweep compaction
func (r *Region) markAndSweepCompact() error {
	// Mark phase: mark all reachable allocations
	// Sweep phase: move marked allocations to beginning of region
	// This is a simplified implementation

	// Coalesce all free blocks first
	current := r.Header.FreeList
	for current != nil {
		r.coalesceFreeBlocks(current)
		current = current.Next
	}

	return nil
}

// copyingCompact implements copying compaction
func (r *Region) copyingCompact() error {
	// Copy all live objects to new location
	// This requires updating all pointers (complex)
	return r.markAndSweepCompact() // Fallback to mark and sweep
}

// generationalCompact implements generational compaction
func (r *Region) generationalCompact() error {
	// Compact based on object generations
	return r.markAndSweepCompact() // Fallback to mark and sweep
}

// incrementalCompact implements incremental compaction
func (r *Region) incrementalCompact() error {
	// Compact in small increments
	return r.markAndSweepCompact() // Fallback to mark and sweep
}

// concurrentCompact implements concurrent compaction
func (r *Region) concurrentCompact() error {
	// Compact concurrently with allocation
	return r.markAndSweepCompact() // Fallback to mark and sweep
}

// Helper functions
// clearMemory zeros out memory block
func clearMemory(ptr unsafe.Pointer, size int) {
	// Clear memory by setting all bytes to zero
	slice := (*[1 << 30]byte)(ptr)[:size:size]
	for i := range slice {
		slice[i] = 0
	}
}

// getStackTrace captures current stack trace
func getStackTrace() []uintptr {
	// Mock implementation - in real code, use runtime.Callers()
	return []uintptr{0x1000, 0x2000, 0x3000}
}

// updateFragmentationRatio updates the fragmentation ratio
func (r *Region) updateFragmentationRatio() {
	r.Stats.FragmentationRatio = r.calculateFragmentationRatio()
}

// calculateFragmentationRatio calculates current fragmentation ratio
func (r *Region) calculateFragmentationRatio() float64 {
	if r.Header.Free == 0 {
		return 0.0
	}

	freeBlockCount := r.countFreeBlocks()
	if freeBlockCount <= 1 {
		return 0.0
	}

	// Calculate external fragmentation
	largestFreeBlock := r.findLargestFreeBlock()
	if largestFreeBlock == 0 {
		return 1.0
	}

	return 1.0 - float64(largestFreeBlock)/float64(r.Header.Free)
}

// countFreeBlocks counts the number of free blocks
func (r *Region) countFreeBlocks() int {
	count := 0
	current := r.Header.FreeList
	for current != nil {
		count++
		current = current.Next
	}
	return count
}

// findLargestFreeBlock finds the size of the largest free block
func (r *Region) findLargestFreeBlock() RegionSize {
	var largest RegionSize = 0
	current := r.Header.FreeList

	for current != nil {
		if current.Size > largest {
			largest = current.Size
		}
		current = current.Next
	}

	return largest
}
