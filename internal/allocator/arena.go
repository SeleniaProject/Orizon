package allocator

import (
	"fmt"
	"sync"
	"unsafe"
)

// ArenaAllocatorImpl implements an arena-based allocator.
type ArenaAllocatorImpl struct {
	config         *Config
	buffer         []byte
	current        uintptr
	size           uintptr
	allocations    uint64
	totalAllocated uintptr
	peakUsage      uintptr
	mu             sync.RWMutex
}

// NewArenaAllocator creates a new arena allocator.
func NewArenaAllocator(size uintptr, config *Config) (*ArenaAllocatorImpl, error) {
	if size == 0 {
		return nil, fmt.Errorf("arena size must be greater than 0")
	}

	// Allocate the arena buffer.
	buffer := make([]byte, size)
	if len(buffer) == 0 {
		return nil, fmt.Errorf("failed to allocate arena buffer")
	}

	return &ArenaAllocatorImpl{
		config:  config,
		buffer:  buffer,
		current: 0,
		size:    size,
	}, nil
}

// Alloc allocates memory from the arena.
func (aa *ArenaAllocatorImpl) Alloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}

	// Align size.
	alignedSize := alignUp(size, aa.config.AlignmentSize)

	aa.mu.Lock()
	defer aa.mu.Unlock()

	// Check if we have enough space.
	if aa.current+alignedSize > aa.size {
		return nil // Out of arena space
	}

	// Get pointer to current position.
	ptr := unsafe.Pointer(&aa.buffer[aa.current])

	// Update current position.
	aa.current += alignedSize
	aa.allocations++
	aa.totalAllocated += alignedSize

	// Track peak usage.
	if aa.current > aa.peakUsage {
		aa.peakUsage = aa.current
	}

	return ptr
}

// Free is a no-op for arena allocator (can't free individual allocations).
func (aa *ArenaAllocatorImpl) Free(ptr unsafe.Pointer) {
	// Arena allocator doesn't support individual free operations.
	// Memory is only freed when the arena is reset.
}

// Realloc reallocates memory (limited implementation).
func (aa *ArenaAllocatorImpl) Realloc(ptr unsafe.Pointer, newSize uintptr) unsafe.Pointer {
	if ptr == nil {
		return aa.Alloc(newSize)
	}

	if newSize == 0 {
		aa.Free(ptr)

		return nil
	}

	// For arena allocator, we can only grow the last allocation efficiently.
	newPtr := aa.Alloc(newSize)
	if newPtr == nil {
		return nil
	}

	// We can't determine the old size without tracking, so we assume a reasonable copy size.
	// In a real implementation, we would track allocation sizes.
	copySize := newSize
	if copySize > 1024 {
		copySize = 1024 // Limit copy to avoid excessive copying
	}

	copyMemory(newPtr, ptr, copySize)

	// Note: We can't free the old memory in an arena allocator.
	return newPtr
}

// TotalAllocated returns total allocated bytes.
func (aa *ArenaAllocatorImpl) TotalAllocated() uintptr {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.totalAllocated
}

// TotalFreed returns total freed bytes (always 0 for arena).
func (aa *ArenaAllocatorImpl) TotalFreed() uintptr {
	return 0 // Arena allocator doesn't track individual frees
}

// ActiveAllocations returns the number of active allocations.
func (aa *ArenaAllocatorImpl) ActiveAllocations() int {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return int(aa.allocations)
}

// Stats returns allocation statistics.
func (aa *ArenaAllocatorImpl) Stats() AllocatorStats {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return AllocatorStats{
		TotalAllocated:    aa.totalAllocated,
		TotalFreed:        0,
		ActiveAllocations: int(aa.allocations),
		PeakAllocations:   int(aa.allocations), // Arena doesn't track peak allocations
		AllocationCount:   aa.allocations,
		FreeCount:         0,
		BytesInUse:        aa.current,
		SystemMemory:      aa.size,
	}
}

// Reset resets the arena, freeing all memory.
func (aa *ArenaAllocatorImpl) Reset() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	aa.current = 0
	aa.allocations = 0
	aa.totalAllocated = 0
}

// Additional arena-specific methods.

// Available returns the amount of available space in the arena.
func (aa *ArenaAllocatorImpl) Available() uintptr {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.size - aa.current
}

// Used returns the amount of used space in the arena.
func (aa *ArenaAllocatorImpl) Used() uintptr {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.current
}

// Size returns the total size of the arena.
func (aa *ArenaAllocatorImpl) Size() uintptr {
	return aa.size
}

// PeakUsage returns the peak memory usage.
func (aa *ArenaAllocatorImpl) PeakUsage() uintptr {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.peakUsage
}

// Fragmentation returns fragmentation ratio (always 0 for arena).
func (aa *ArenaAllocatorImpl) Fragmentation() float64 {
	return 0.0 // Arena allocator has no fragmentation
}

// CanAlloc checks if an allocation of the given size would succeed.
func (aa *ArenaAllocatorImpl) CanAlloc(size uintptr) bool {
	alignedSize := alignUp(size, aa.config.AlignmentSize)
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.current+alignedSize <= aa.size
}

// AllocAligned allocates memory with specific alignment.
func (aa *ArenaAllocatorImpl) AllocAligned(size, alignment uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}

	aa.mu.Lock()
	defer aa.mu.Unlock()

	// Align current position to the requested alignment.
	alignedCurrent := alignUp(aa.current, alignment)
	alignedSize := alignUp(size, aa.config.AlignmentSize)

	// Check if we have enough space.
	if alignedCurrent+alignedSize > aa.size {
		return nil // Out of arena space
	}

	// Get pointer to aligned position.
	ptr := unsafe.Pointer(&aa.buffer[alignedCurrent])

	// Update current position.
	aa.current = alignedCurrent + alignedSize
	aa.allocations++
	aa.totalAllocated += alignedSize

	// Track peak usage.
	if aa.current > aa.peakUsage {
		aa.peakUsage = aa.current
	}

	return ptr
}

// SaveState saves the current state of the arena.
func (aa *ArenaAllocatorImpl) SaveState() ArenaState {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return ArenaState{
		Current:     aa.current,
		Allocations: aa.allocations,
	}
}

// RestoreState restores the arena to a previous state.
func (aa *ArenaAllocatorImpl) RestoreState(state ArenaState) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	if state.Current <= aa.size {
		aa.current = state.Current
		aa.allocations = state.Allocations
	}
}

// ArenaState represents the state of an arena allocator.
type ArenaState struct {
	Current     uintptr
	Allocations uint64
}

// SubArena creates a sub-arena from the current arena.
func (aa *ArenaAllocatorImpl) SubArena(size uintptr) (*ArenaAllocatorImpl, error) {
	alignedSize := alignUp(size, aa.config.AlignmentSize)

	// Allocate space for the sub-arena.
	ptr := aa.Alloc(alignedSize)
	if ptr == nil {
		return nil, fmt.Errorf("insufficient space for sub-arena")
	}

	// Create a new arena using the allocated space.
	buffer := (*[1 << 30]byte)(ptr)[:size:size]

	return &ArenaAllocatorImpl{
		config:  aa.config,
		buffer:  buffer,
		current: 0,
		size:    size,
	}, nil
}

// AllocString allocates memory for a string and copies the content.
func (aa *ArenaAllocatorImpl) AllocString(s string) unsafe.Pointer {
	if len(s) == 0 {
		return nil
	}

	ptr := aa.Alloc(uintptr(len(s)))
	if ptr == nil {
		return nil
	}

	// Copy string content.
	dst := (*[1 << 30]byte)(ptr)[:len(s):len(s)]
	copy(dst, []byte(s))

	return ptr
}

// AllocSlice allocates memory for a slice of the given element type and count.
func (aa *ArenaAllocatorImpl) AllocSlice(elementSize uintptr, count int) unsafe.Pointer {
	if count == 0 {
		return nil
	}

	totalSize := elementSize * uintptr(count)

	return aa.Alloc(totalSize)
}

// Clone creates a copy of the arena with the same content.
func (aa *ArenaAllocatorImpl) Clone() (*ArenaAllocatorImpl, error) {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	// Create new arena with same size.
	newArena, err := NewArenaAllocator(aa.size, aa.config)
	if err != nil {
		return nil, err
	}

	// Copy used content.
	if aa.current > 0 {
		copy(newArena.buffer[:aa.current], aa.buffer[:aa.current])
		newArena.current = aa.current
		newArena.allocations = aa.allocations
		newArena.totalAllocated = aa.totalAllocated
		newArena.peakUsage = aa.peakUsage
	}

	return newArena, nil
}
