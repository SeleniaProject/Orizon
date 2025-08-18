package allocator

import (
	"fmt"
	"sync"
	"unsafe"
)

// PoolAllocatorImpl implements a pool-based allocator for fixed-size objects
type PoolAllocatorImpl struct {
	mu       sync.RWMutex
	config   *Config
	pools    map[uintptr]*Pool
	fallback Allocator
	stats    PoolStats
}

// Pool represents a memory pool for objects of a specific size
type Pool struct {
	mu        sync.Mutex
	size      uintptr
	chunks    [][]byte
	freeList  []unsafe.Pointer
	chunkSize uintptr
	allocated uint64
	freed     uint64
}

// PoolStats provides statistics for pool allocator
type PoolStats struct {
	TotalPools      int
	TotalChunks     int
	TotalAllocated  uintptr
	TotalFreed      uintptr
	AllocationCount uint64
	FreeCount       uint64
	HitRate         float64
	MissRate        float64
}

// NewPoolAllocator creates a new pool allocator
func NewPoolAllocator(poolSizes []uintptr, config *Config) (*PoolAllocatorImpl, error) {
	if len(poolSizes) == 0 {
		return nil, fmt.Errorf("pool sizes cannot be empty")
	}

	pools := make(map[uintptr]*Pool)

	// Create pools for each size
	for _, size := range poolSizes {
		alignedSize := alignUp(size, config.AlignmentSize)
		pools[alignedSize] = &Pool{
			size:      alignedSize,
			chunkSize: 64 * 1024, // 64KB chunks
			freeList:  make([]unsafe.Pointer, 0),
		}
	}

	// Create fallback allocator (system allocator)
	fallback := NewSystemAllocator(config)

	return &PoolAllocatorImpl{
		config:   config,
		pools:    pools,
		fallback: fallback,
	}, nil
}

// Alloc allocates memory from the appropriate pool
func (pa *PoolAllocatorImpl) Alloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}

	alignedSize := alignUp(size, pa.config.AlignmentSize)

	// Find the best-fit pool
	poolSize := pa.findBestPool(alignedSize)
	if poolSize == 0 {
		// No suitable pool, use fallback allocator
		pa.mu.Lock()
		pa.stats.MissRate++
		pa.mu.Unlock()
		return pa.fallback.Alloc(size)
	}

	// Get from pool
	pa.mu.RLock()
	pool, exists := pa.pools[poolSize]
	pa.mu.RUnlock()

	if !exists {
		// Should not happen, but fallback to system allocator
		return pa.fallback.Alloc(size)
	}

	ptr := pool.alloc()
	if ptr != nil {
		pa.mu.Lock()
		pa.stats.HitRate++
		pa.stats.AllocationCount++
		pa.stats.TotalAllocated += poolSize
		pa.mu.Unlock()
	}

	return ptr
}

// Free frees memory back to the appropriate pool
func (pa *PoolAllocatorImpl) Free(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}

	// Try to find which pool this pointer belongs to
	poolSize := pa.findPoolForPointer(ptr)
	if poolSize == 0 {
		// Not from our pools, use fallback
		pa.fallback.Free(ptr)
		return
	}

	pa.mu.RLock()
	pool, exists := pa.pools[poolSize]
	pa.mu.RUnlock()

	if !exists {
		// Should not happen
		pa.fallback.Free(ptr)
		return
	}

	pool.free(ptr)

	pa.mu.Lock()
	pa.stats.FreeCount++
	pa.stats.TotalFreed += poolSize
	pa.mu.Unlock()
}

// Realloc reallocates memory
func (pa *PoolAllocatorImpl) Realloc(ptr unsafe.Pointer, newSize uintptr) unsafe.Pointer {
	if ptr == nil {
		return pa.Alloc(newSize)
	}
	if newSize == 0 {
		pa.Free(ptr)
		return nil
	}

	// Get old pool size
	oldPoolSize := pa.findPoolForPointer(ptr)
	newAlignedSize := alignUp(newSize, pa.config.AlignmentSize)
	newPoolSize := pa.findBestPool(newAlignedSize)

	// If same pool size, just return the same pointer
	if oldPoolSize != 0 && oldPoolSize == newPoolSize {
		return ptr
	}

	// Allocate new memory
	newPtr := pa.Alloc(newSize)
	if newPtr == nil {
		return nil
	}

	// Copy old data
	copySize := oldPoolSize
	if newSize < oldPoolSize {
		copySize = newSize
	}
	if copySize > 0 {
		copyMemory(newPtr, ptr, copySize)
	}

	// Free old memory
	pa.Free(ptr)

	return newPtr
}

// TotalAllocated returns total allocated bytes
func (pa *PoolAllocatorImpl) TotalAllocated() uintptr {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	return pa.stats.TotalAllocated + pa.fallback.TotalAllocated()
}

// TotalFreed returns total freed bytes
func (pa *PoolAllocatorImpl) TotalFreed() uintptr {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	return pa.stats.TotalFreed + pa.fallback.TotalFreed()
}

// ActiveAllocations returns the number of active allocations
func (pa *PoolAllocatorImpl) ActiveAllocations() int {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	count := int(pa.stats.AllocationCount - pa.stats.FreeCount)
	return count + pa.fallback.ActiveAllocations()
}

// Stats returns allocation statistics
func (pa *PoolAllocatorImpl) Stats() AllocatorStats {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	fallbackStats := pa.fallback.Stats()

	return AllocatorStats{
		TotalAllocated:    pa.stats.TotalAllocated + fallbackStats.TotalAllocated,
		TotalFreed:        pa.stats.TotalFreed + fallbackStats.TotalFreed,
		ActiveAllocations: int(pa.stats.AllocationCount-pa.stats.FreeCount) + fallbackStats.ActiveAllocations,
		PeakAllocations:   fallbackStats.PeakAllocations, // Use fallback's peak
		AllocationCount:   pa.stats.AllocationCount + fallbackStats.AllocationCount,
		FreeCount:         pa.stats.FreeCount + fallbackStats.FreeCount,
		BytesInUse:        (pa.stats.TotalAllocated - pa.stats.TotalFreed) + fallbackStats.BytesInUse,
		SystemMemory:      fallbackStats.SystemMemory,
	}
}

// Reset resets all pools
func (pa *PoolAllocatorImpl) Reset() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	for _, pool := range pa.pools {
		pool.reset()
	}

	pa.stats = PoolStats{}
	pa.fallback.Reset()
}

// Helper methods

// findBestPool finds the smallest pool that can accommodate the size
func (pa *PoolAllocatorImpl) findBestPool(size uintptr) uintptr {
	var bestSize uintptr = 0

	pa.mu.RLock()
	defer pa.mu.RUnlock()

	for poolSize := range pa.pools {
		if poolSize >= size {
			if bestSize == 0 || poolSize < bestSize {
				bestSize = poolSize
			}
		}
	}

	return bestSize
}

// findPoolForPointer finds which pool a pointer belongs to (simplified)
func (pa *PoolAllocatorImpl) findPoolForPointer(ptr unsafe.Pointer) uintptr {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	// In a real implementation, we would track which chunk each pointer comes from
	// For simplicity, we'll try to guess based on common sizes
	// This is not reliable and a real implementation would need better tracking

	for poolSize, pool := range pa.pools {
		if pool.containsPointer(ptr) {
			return poolSize
		}
	}

	return 0 // Not found in any pool
}

// Pool methods

// alloc allocates from this pool
func (p *Pool) alloc() unsafe.Pointer {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check free list first
	if len(p.freeList) > 0 {
		ptr := p.freeList[len(p.freeList)-1]
		p.freeList = p.freeList[:len(p.freeList)-1]
		p.allocated++
		return ptr
	}

	// Need to allocate new chunk
	if err := p.allocateChunk(); err != nil {
		return nil
	}

	// Try again from free list
	if len(p.freeList) > 0 {
		ptr := p.freeList[len(p.freeList)-1]
		p.freeList = p.freeList[:len(p.freeList)-1]
		p.allocated++
		return ptr
	}

	return nil
}

// free returns memory to this pool
func (p *Pool) free(ptr unsafe.Pointer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.freeList = append(p.freeList, ptr)
	p.freed++
}

// allocateChunk allocates a new chunk for this pool
func (p *Pool) allocateChunk() error {
	// Calculate how many objects fit in a chunk
	objectsPerChunk := p.chunkSize / p.size
	if objectsPerChunk == 0 {
		objectsPerChunk = 1
	}

	actualChunkSize := objectsPerChunk * p.size

	// Allocate chunk
	chunk := make([]byte, actualChunkSize)
	if len(chunk) == 0 {
		return fmt.Errorf("failed to allocate chunk")
	}

	// Add chunk to list
	p.chunks = append(p.chunks, chunk)

	// Add all objects in chunk to free list
	for i := uintptr(0); i < objectsPerChunk; i++ {
		ptr := unsafe.Pointer(&chunk[i*p.size])
		p.freeList = append(p.freeList, ptr)
	}

	return nil
}

// containsPointer checks if a pointer belongs to this pool (simplified)
func (p *Pool) containsPointer(ptr unsafe.Pointer) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	ptrAddr := uintptr(ptr)

	for _, chunk := range p.chunks {
		chunkStart := uintptr(unsafe.Pointer(&chunk[0]))
		chunkEnd := chunkStart + uintptr(len(chunk))

		if ptrAddr >= chunkStart && ptrAddr < chunkEnd {
			// Check if it's aligned to object boundaries
			offset := ptrAddr - chunkStart
			if offset%p.size == 0 {
				return true
			}
		}
	}

	return false
}

// reset resets this pool
func (p *Pool) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.chunks = nil
	p.freeList = p.freeList[:0]
	p.allocated = 0
	p.freed = 0
}

// GetPoolStats returns statistics for this pool allocator
func (pa *PoolAllocatorImpl) GetPoolStats() PoolStats {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	stats := pa.stats
	stats.TotalPools = len(pa.pools)

	totalChunks := 0
	for _, pool := range pa.pools {
		pool.mu.Lock()
		totalChunks += len(pool.chunks)
		pool.mu.Unlock()
	}
	stats.TotalChunks = totalChunks

	// Calculate hit/miss rates
	total := stats.HitRate + stats.MissRate
	if total > 0 {
		stats.HitRate = stats.HitRate / total * 100
		stats.MissRate = stats.MissRate / total * 100
	}

	return stats
}

// GetPoolInfo returns information about individual pools
func (pa *PoolAllocatorImpl) GetPoolInfo() []PoolInfo {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	var info []PoolInfo
	for size, pool := range pa.pools {
		pool.mu.Lock()
		poolInfo := PoolInfo{
			Size:          size,
			ChunkCount:    len(pool.chunks),
			FreeObjects:   len(pool.freeList),
			Allocated:     pool.allocated,
			Freed:         pool.freed,
			ActiveObjects: pool.allocated - pool.freed,
		}
		pool.mu.Unlock()
		info = append(info, poolInfo)
	}

	return info
}

// PoolInfo provides information about a specific pool
type PoolInfo struct {
	Size          uintptr
	ChunkCount    int
	FreeObjects   int
	Allocated     uint64
	Freed         uint64
	ActiveObjects uint64
}

// AddPool adds a new pool of the specified size
func (pa *PoolAllocatorImpl) AddPool(size uintptr) error {
	alignedSize := alignUp(size, pa.config.AlignmentSize)

	pa.mu.Lock()
	defer pa.mu.Unlock()

	if _, exists := pa.pools[alignedSize]; exists {
		return fmt.Errorf("pool of size %d already exists", alignedSize)
	}

	pa.pools[alignedSize] = &Pool{
		size:      alignedSize,
		chunkSize: 64 * 1024, // 64KB chunks
		freeList:  make([]unsafe.Pointer, 0),
	}

	return nil
}

// RemovePool removes a pool of the specified size
func (pa *PoolAllocatorImpl) RemovePool(size uintptr) error {
	alignedSize := alignUp(size, pa.config.AlignmentSize)

	pa.mu.Lock()
	defer pa.mu.Unlock()

	pool, exists := pa.pools[alignedSize]
	if !exists {
		return fmt.Errorf("pool of size %d does not exist", alignedSize)
	}

	// Reset the pool before removing
	pool.reset()
	delete(pa.pools, alignedSize)

	return nil
}

// OptimizePools optimizes pool sizes based on usage patterns
func (pa *PoolAllocatorImpl) OptimizePools() {
	// This would analyze allocation patterns and adjust pool sizes
	// For simplicity, this is left as a placeholder
}

// Defragment attempts to defragment the pools
func (pa *PoolAllocatorImpl) Defragment() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	for _, pool := range pa.pools {
		pool.mu.Lock()
		// Defragmentation logic would go here
		// For simplicity, we'll just compact the free list
		if len(pool.freeList) > 1000 {
			// Keep only the last 500 free objects to reduce memory usage
			pool.freeList = pool.freeList[len(pool.freeList)-500:]
		}
		pool.mu.Unlock()
	}
}
