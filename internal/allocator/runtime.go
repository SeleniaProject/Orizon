package allocator

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Runtime provides the runtime memory management interface for Orizon
type Runtime struct {
	allocator   Allocator
	gcEnabled   bool
	gcThreshold uintptr
	lastGCTime  int64
	gcStats     GCStats
	stringPool  *StringPool
	slicePool   *SlicePool
	mu          sync.RWMutex
}

// GCStats provides garbage collection statistics
type GCStats struct {
	Collections    uint64
	TotalFreed     uintptr
	LastCollection int64
	AverageTime    int64
	MaxPauseTime   int64
}

// StringPool manages string allocations
type StringPool struct {
	mu      sync.RWMutex
	strings map[string]unsafe.Pointer
	stats   StringPoolStats
}

// StringPoolStats provides string pool statistics
type StringPoolStats struct {
	Hits        uint64
	Misses      uint64
	TotalSize   uintptr
	StringCount int
}

// SlicePool manages slice header allocations
type SlicePool struct {
	mu    sync.RWMutex
	pool  []SliceHeader
	stats SlicePoolStats
}

// SliceHeader represents a slice header for runtime
type SliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// SlicePoolStats provides slice pool statistics
type SlicePoolStats struct {
	Allocated uint64
	Reused    uint64
	TotalSize uintptr
}

// Global runtime instance
var GlobalRuntime *Runtime

// InitializeRuntime initializes the global runtime with the specified allocator
func InitializeRuntime(allocator Allocator, options ...RuntimeOption) error {
	if allocator == nil {
		return fmt.Errorf("allocator cannot be nil")
	}

	runtime := &Runtime{
		allocator:   allocator,
		gcEnabled:   true,
		gcThreshold: 32 * 1024 * 1024, // 32MB default GC threshold
		stringPool:  NewStringPool(),
		slicePool:   NewSlicePool(),
	}

	// Apply options
	for _, opt := range options {
		opt(runtime)
	}

	GlobalRuntime = runtime
	return nil
}

// RuntimeOption configures the runtime
type RuntimeOption func(*Runtime)

// WithGC enables/disables garbage collection
func WithGC(enabled bool) RuntimeOption {
	return func(r *Runtime) { r.gcEnabled = enabled }
}

// WithGCThreshold sets the GC threshold
func WithGCThreshold(threshold uintptr) RuntimeOption {
	return func(r *Runtime) { r.gcThreshold = threshold }
}

// Memory allocation functions for Orizon runtime

// AllocObject allocates memory for an object
func (r *Runtime) AllocObject(size uintptr) unsafe.Pointer {
	ptr := r.allocator.Alloc(size)

	// Check if GC should run
	if r.gcEnabled && r.shouldRunGC() {
		go r.runGC()
	}

	return ptr
}

// AllocArray allocates memory for an array
func (r *Runtime) AllocArray(elementSize uintptr, count int) unsafe.Pointer {
	if count <= 0 {
		return nil
	}

	totalSize := elementSize * uintptr(count)
	return r.AllocObject(totalSize)
}

// AllocSlice allocates memory for a slice
func (r *Runtime) AllocSlice(elementSize uintptr, len, cap int) *SliceHeader {
	if cap <= 0 {
		return &SliceHeader{Data: nil, Len: 0, Cap: 0}
	}

	totalSize := elementSize * uintptr(cap)
	data := r.AllocObject(totalSize)

	if len > cap {
		len = cap
	}

	// Try to reuse slice header from pool
	header := r.slicePool.Get()
	if header == nil {
		header = &SliceHeader{}
	}

	header.Data = data
	header.Len = len
	header.Cap = cap

	return header
}

// AllocString allocates memory for a string
func (r *Runtime) AllocString(s string) unsafe.Pointer {
	if len(s) == 0 {
		return nil
	}

	// Check string pool first
	if ptr := r.stringPool.Get(s); ptr != nil {
		return ptr
	}

	// Allocate new string
	ptr := r.AllocObject(uintptr(len(s)))
	if ptr != nil {
		// Copy string content
		dst := (*[1 << 30]byte)(ptr)[:len(s):len(s)]
		copy(dst, []byte(s))

		// Add to string pool
		r.stringPool.Put(s, ptr)
	}

	return ptr
}

// FreeObject frees memory for an object
func (r *Runtime) FreeObject(ptr unsafe.Pointer) {
	if ptr != nil {
		r.allocator.Free(ptr)
	}
}

// FreeSlice frees memory for a slice
func (r *Runtime) FreeSlice(header *SliceHeader) {
	if header == nil {
		return
	}

	if header.Data != nil {
		r.FreeObject(header.Data)
	}

	// Return header to pool
	r.slicePool.Put(header)
}

// ReallocSlice reallocates a slice with new capacity
func (r *Runtime) ReallocSlice(header *SliceHeader, elementSize uintptr, newCap int) *SliceHeader {
	if header == nil {
		return r.AllocSlice(elementSize, 0, newCap)
	}

	if newCap <= header.Cap {
		// Just adjust the capacity
		header.Cap = newCap
		if header.Len > newCap {
			header.Len = newCap
		}
		return header
	}

	// Need to reallocate
	newSize := elementSize * uintptr(newCap)

	newData := r.allocator.Realloc(header.Data, newSize)
	if newData == nil {
		return nil
	}

	header.Data = newData
	header.Cap = newCap

	return header
}

// GC functions

// shouldRunGC determines if GC should run
func (r *Runtime) shouldRunGC() bool {
	if !r.gcEnabled {
		return false
	}

	stats := r.allocator.Stats()
	return stats.BytesInUse > r.gcThreshold
}

// runGC runs garbage collection
func (r *Runtime) runGC() {
	// Don't hold the main lock during GC to avoid deadlock
	startTime := getTimestamp()

	// Simple GC: just reset arena allocators
	// In a real implementation, this would mark and sweep
	if arena, ok := r.allocator.(*ArenaAllocatorImpl); ok {
		arena.Reset()
	}

	endTime := getTimestamp()

	// Update GC stats with lock
	r.mu.Lock()
	r.gcStats.Collections++
	r.gcStats.LastCollection = endTime

	pauseTime := endTime - startTime
	if pauseTime > r.gcStats.MaxPauseTime {
		r.gcStats.MaxPauseTime = pauseTime
	}

	// Update average time
	if r.gcStats.Collections == 1 {
		r.gcStats.AverageTime = pauseTime
	} else {
		r.gcStats.AverageTime = (r.gcStats.AverageTime + pauseTime) / 2
	}
	r.mu.Unlock()
}

// ForceGC forces garbage collection
func (r *Runtime) ForceGC() {
	r.runGC()
}

// GetGCStats returns GC statistics
func (r *Runtime) GetGCStats() GCStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.gcStats
}

// String pool implementation

// NewStringPool creates a new string pool
func NewStringPool() *StringPool {
	return &StringPool{
		strings: make(map[string]unsafe.Pointer),
	}
}

// Get retrieves a string from the pool
func (sp *StringPool) Get(s string) unsafe.Pointer {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if ptr, exists := sp.strings[s]; exists {
		sp.stats.Hits++
		return ptr
	}

	sp.stats.Misses++
	return nil
}

// Put adds a string to the pool
func (sp *StringPool) Put(s string, ptr unsafe.Pointer) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if _, exists := sp.strings[s]; !exists {
		sp.strings[s] = ptr
		sp.stats.TotalSize += uintptr(len(s))
		sp.stats.StringCount++
	}
}

// GetStats returns string pool statistics
func (sp *StringPool) GetStats() StringPoolStats {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.stats
}

// Clear clears the string pool
func (sp *StringPool) Clear() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.strings = make(map[string]unsafe.Pointer)
	sp.stats = StringPoolStats{}
}

// Slice pool implementation

// NewSlicePool creates a new slice pool
func NewSlicePool() *SlicePool {
	return &SlicePool{
		pool: make([]SliceHeader, 0, 100),
	}
}

// Get retrieves a slice header from the pool
func (slp *SlicePool) Get() *SliceHeader {
	slp.mu.Lock()
	defer slp.mu.Unlock()

	if len(slp.pool) > 0 {
		header := slp.pool[len(slp.pool)-1]
		slp.pool = slp.pool[:len(slp.pool)-1]
		slp.stats.Reused++
		return &header
	}

	slp.stats.Allocated++
	return nil // Caller will allocate new one
}

// Put returns a slice header to the pool
func (slp *SlicePool) Put(header *SliceHeader) {
	if header == nil {
		return
	}

	slp.mu.Lock()
	defer slp.mu.Unlock()

	// Reset the header
	header.Data = nil
	header.Len = 0
	header.Cap = 0

	// Add to pool if there's space
	if len(slp.pool) < cap(slp.pool) {
		slp.pool = append(slp.pool, *header)
	}
}

// GetStats returns slice pool statistics
func (slp *SlicePool) GetStats() SlicePoolStats {
	slp.mu.RLock()
	defer slp.mu.RUnlock()
	return slp.stats
}

// Global convenience functions

// RuntimeAlloc allocates memory using the global runtime
func RuntimeAlloc(size uintptr) unsafe.Pointer {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	return GlobalRuntime.AllocObject(size)
}

// RuntimeFree frees memory using the global runtime
func RuntimeFree(ptr unsafe.Pointer) {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	GlobalRuntime.FreeObject(ptr)
}

// RuntimeAllocArray allocates an array using the global runtime
func RuntimeAllocArray(elementSize uintptr, count int) unsafe.Pointer {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	return GlobalRuntime.AllocArray(elementSize, count)
}

// RuntimeAllocSlice allocates a slice using the global runtime
func RuntimeAllocSlice(elementSize uintptr, len, cap int) *SliceHeader {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	return GlobalRuntime.AllocSlice(elementSize, len, cap)
}

// RuntimeAllocString allocates a string using the global runtime
func RuntimeAllocString(s string) unsafe.Pointer {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	return GlobalRuntime.AllocString(s)
}

// RuntimeFreeSlice frees a slice using the global runtime
func RuntimeFreeSlice(header *SliceHeader) {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	GlobalRuntime.FreeSlice(header)
}

// RuntimeForceGC forces garbage collection using the global runtime
func RuntimeForceGC() {
	if GlobalRuntime == nil {
		panic("Runtime not initialized")
	}
	GlobalRuntime.ForceGC()
}

// GetRuntimeStats returns comprehensive runtime statistics
func GetRuntimeStats() RuntimeStats {
	if GlobalRuntime == nil {
		return RuntimeStats{}
	}

	allocStats := GlobalRuntime.allocator.Stats()
	gcStats := GlobalRuntime.GetGCStats()
	stringStats := GlobalRuntime.stringPool.GetStats()
	sliceStats := GlobalRuntime.slicePool.GetStats()

	return RuntimeStats{
		Allocator:  allocStats,
		GC:         gcStats,
		StringPool: stringStats,
		SlicePool:  sliceStats,
	}
}

// RuntimeStats provides comprehensive runtime statistics
type RuntimeStats struct {
	Allocator  AllocatorStats
	GC         GCStats
	StringPool StringPoolStats
	SlicePool  SlicePoolStats
}

// Memory layout helper functions for runtime

// GetObjectSize returns the size of an object at the given pointer
func GetObjectSize(ptr unsafe.Pointer) uintptr {
	// In a real implementation, this would read object metadata
	// For simplicity, we return a placeholder
	return 0
}

// GetObjectType returns the type of an object at the given pointer
func GetObjectType(ptr unsafe.Pointer) int {
	// In a real implementation, this would read type metadata
	// For simplicity, we return a placeholder
	return 0
}

// IsValidPointer checks if a pointer is valid
func IsValidPointer(ptr unsafe.Pointer) bool {
	if ptr == nil {
		return false
	}

	// Basic sanity checks
	addr := uintptr(ptr)

	// Check if aligned
	if addr%8 != 0 {
		return false
	}

	// Check if in reasonable range (simplified)
	if addr < 0x1000 || addr > 0x7FFFFFFFFFFF {
		return false
	}

	return true
}

// Memory barriers and atomic operations

var memoryBarrier uint64

// MemoryBarrier provides a memory barrier
func MemoryBarrier() {
	atomic.AddUint64(&memoryBarrier, 1)
}

// AtomicLoadPointer atomically loads a pointer
func AtomicLoadPointer(addr *unsafe.Pointer) unsafe.Pointer {
	return atomic.LoadPointer(addr)
}

// AtomicStorePointer atomically stores a pointer
func AtomicStorePointer(addr *unsafe.Pointer, val unsafe.Pointer) {
	atomic.StorePointer(addr, val)
}

// AtomicCompareAndSwapPointer atomically compares and swaps a pointer
func AtomicCompareAndSwapPointer(addr *unsafe.Pointer, old, new unsafe.Pointer) bool {
	return atomic.CompareAndSwapPointer(addr, old, new)
}

// Shutdown gracefully shuts down the runtime
func (r *Runtime) Shutdown() error {
	r.mu.Lock()
	gcEnabled := r.gcEnabled
	r.mu.Unlock()

	// Run final GC without holding lock
	if gcEnabled {
		r.runGC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear pools
	r.stringPool.Clear()

	// Reset allocator if possible
	r.allocator.Reset()

	return nil
}

// ShutdownRuntime shuts down the global runtime
func ShutdownRuntime() error {
	if GlobalRuntime == nil {
		return nil
	}

	return GlobalRuntime.Shutdown()
}
