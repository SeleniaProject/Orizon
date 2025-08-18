// Package allocator provides memory allocation services for the Orizon runtime.
// This implements a minimal but functional memory allocator supporting both
// system-level allocation and arena-based allocation for bootstrap.
package allocator

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// AllocatorKind defines the type of allocator
type AllocatorKind int

const (
	SystemAllocatorKind AllocatorKind = iota
	ArenaAllocatorKind
	PoolAllocatorKind
)

// Allocator defines the interface for memory allocators
type Allocator interface {
	Alloc(size uintptr) unsafe.Pointer
	Free(ptr unsafe.Pointer)
	Realloc(ptr unsafe.Pointer, newSize uintptr) unsafe.Pointer
	TotalAllocated() uintptr
	TotalFreed() uintptr
	ActiveAllocations() int
	Stats() AllocatorStats
	Reset() // For arena allocators
}

// AllocatorStats provides allocation statistics
type AllocatorStats struct {
	TotalAllocated    uintptr
	TotalFreed        uintptr
	ActiveAllocations int
	PeakAllocations   int
	AllocationCount   uint64
	FreeCount         uint64
	BytesInUse        uintptr
	SystemMemory      uintptr
}

// GlobalAllocator provides the default allocator for the Orizon runtime
var GlobalAllocator Allocator

// Initialize sets up the global allocator
func Initialize(kind AllocatorKind, options ...Option) error {
	config := defaultConfig()
	for _, opt := range options {
		opt(config)
	}

	switch kind {
	case SystemAllocatorKind:
		GlobalAllocator = NewSystemAllocator(config)
	case ArenaAllocatorKind:
		allocator, err := NewArenaAllocator(config.ArenaSize, config)
		if err != nil {
			return fmt.Errorf("failed to create arena allocator: %v", err)
		}
		GlobalAllocator = allocator
	case PoolAllocatorKind:
		allocator, err := NewPoolAllocator(config.PoolSizes, config)
		if err != nil {
			return fmt.Errorf("failed to create pool allocator: %v", err)
		}
		GlobalAllocator = allocator
	default:
		return fmt.Errorf("unknown allocator kind: %v", kind)
	}

	return nil
}

// Configuration for allocators
type Config struct {
	EnableTracking  bool
	EnableDebug     bool
	ArenaSize       uintptr
	PoolSizes       []uintptr
	MaxAllocations  int
	MemoryLimit     uintptr
	EnableLeakCheck bool
	AlignmentSize   uintptr
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		EnableTracking:  true,
		EnableDebug:     false,
		ArenaSize:       64 * 1024 * 1024, // 64MB default arena
		PoolSizes:       []uintptr{8, 16, 32, 64, 128, 256, 512, 1024},
		MaxAllocations:  1000000,
		MemoryLimit:     1024 * 1024 * 1024, // 1GB limit
		EnableLeakCheck: true,
		AlignmentSize:   8, // 8-byte alignment
	}
}

// Option functions
func WithTracking(enabled bool) Option {
	return func(c *Config) { c.EnableTracking = enabled }
}

func WithDebug(enabled bool) Option {
	return func(c *Config) { c.EnableDebug = enabled }
}

func WithArenaSize(size uintptr) Option {
	return func(c *Config) { c.ArenaSize = size }
}

func WithPoolSizes(sizes []uintptr) Option {
	return func(c *Config) { c.PoolSizes = sizes }
}

func WithMemoryLimit(limit uintptr) Option {
	return func(c *Config) { c.MemoryLimit = limit }
}

func WithLeakCheck(enabled bool) Option {
	return func(c *Config) { c.EnableLeakCheck = enabled }
}

func WithAlignment(alignment uintptr) Option {
	return func(c *Config) { c.AlignmentSize = alignment }
}

// Allocation metadata for tracking
type AllocationInfo struct {
	Size       uintptr
	Timestamp  int64
	StackTrace []uintptr
}

// SystemAllocatorImpl implements a simple wrapper around Go's memory allocator
type SystemAllocatorImpl struct {
	mu                sync.RWMutex
	config            *Config
	totalAllocated    uintptr
	totalFreed        uintptr
	activeAllocations map[unsafe.Pointer]*AllocationInfo
	allocationCount   uint64
	freeCount         uint64
	peakAllocations   int
	allocatedSlices   map[unsafe.Pointer][]byte // Keep references to prevent GC
}

// NewSystemAllocator creates a new system allocator
func NewSystemAllocator(config *Config) *SystemAllocatorImpl {
	return &SystemAllocatorImpl{
		config:            config,
		activeAllocations: make(map[unsafe.Pointer]*AllocationInfo),
		allocatedSlices:   make(map[unsafe.Pointer][]byte),
	}
}

// Alloc allocates memory using the system allocator
func (sa *SystemAllocatorImpl) Alloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return nil
	}

	// Align size
	alignedSize := alignUp(size, sa.config.AlignmentSize)
	if alignedSize == 0 {
		return nil // Overflow or invalid size
	}

	// Check memory limit
	if sa.config.MemoryLimit > 0 {
		current := atomic.LoadUintptr(&sa.totalAllocated) - atomic.LoadUintptr(&sa.totalFreed)
		if current+alignedSize > sa.config.MemoryLimit {
			return nil // Out of memory
		}
	}

	// Allocate memory using Go slice
	slice := make([]byte, alignedSize)
	if len(slice) != int(alignedSize) || len(slice) == 0 {
		return nil
	}

	ptr := unsafe.Pointer(&slice[0])

	// Store slice to prevent GC
	sa.mu.Lock()
	sa.allocatedSlices[ptr] = slice
	sa.mu.Unlock()

	// Update statistics
	atomic.AddUintptr(&sa.totalAllocated, alignedSize)
	atomic.AddUint64(&sa.allocationCount, 1)

	// Track allocation if enabled
	if sa.config.EnableTracking {
		sa.trackAllocation(ptr, alignedSize)
	}

	return ptr
}

// Free frees memory allocated by the system allocator
func (sa *SystemAllocatorImpl) Free(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}

	var size uintptr

	// Get size from tracking if enabled
	if sa.config.EnableTracking {
		size = sa.untrackAllocation(ptr)
	}

	// Remove slice reference to allow GC
	sa.mu.Lock()
	if slice, exists := sa.allocatedSlices[ptr]; exists {
		size = uintptr(len(slice)) // Get actual size
		delete(sa.allocatedSlices, ptr)
	}
	sa.mu.Unlock()

	// Update statistics
	atomic.AddUintptr(&sa.totalFreed, size)
	atomic.AddUint64(&sa.freeCount, 1)
}

// Realloc reallocates memory
func (sa *SystemAllocatorImpl) Realloc(ptr unsafe.Pointer, newSize uintptr) unsafe.Pointer {
	if ptr == nil {
		return sa.Alloc(newSize)
	}
	if newSize == 0 {
		sa.Free(ptr)
		return nil
	}

	// Get old size from tracking
	var oldSize uintptr
	if sa.config.EnableTracking {
		sa.mu.RLock()
		if info, exists := sa.activeAllocations[ptr]; exists {
			oldSize = info.Size
		}
		sa.mu.RUnlock()
	}

	// Allocate new memory
	newPtr := sa.Alloc(newSize)
	if newPtr == nil {
		return nil
	}

	// Copy old data
	if oldSize > 0 {
		copySize := oldSize
		if newSize < oldSize {
			copySize = newSize
		}
		copyMemory(newPtr, ptr, copySize)
	}

	// Free old memory
	sa.Free(ptr)

	return newPtr
}

// TotalAllocated returns total allocated bytes
func (sa *SystemAllocatorImpl) TotalAllocated() uintptr {
	return atomic.LoadUintptr(&sa.totalAllocated)
}

// TotalFreed returns total freed bytes
func (sa *SystemAllocatorImpl) TotalFreed() uintptr {
	return atomic.LoadUintptr(&sa.totalFreed)
}

// ActiveAllocations returns the number of active allocations
func (sa *SystemAllocatorImpl) ActiveAllocations() int {
	if !sa.config.EnableTracking {
		return 0
	}

	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return len(sa.activeAllocations)
}

// Stats returns allocation statistics
func (sa *SystemAllocatorImpl) Stats() AllocatorStats {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	return AllocatorStats{
		TotalAllocated:    atomic.LoadUintptr(&sa.totalAllocated),
		TotalFreed:        atomic.LoadUintptr(&sa.totalFreed),
		ActiveAllocations: len(sa.activeAllocations),
		PeakAllocations:   sa.peakAllocations,
		AllocationCount:   atomic.LoadUint64(&sa.allocationCount),
		FreeCount:         atomic.LoadUint64(&sa.freeCount),
		BytesInUse:        atomic.LoadUintptr(&sa.totalAllocated) - atomic.LoadUintptr(&sa.totalFreed),
		SystemMemory:      getSystemMemory(),
	}
}

// Reset is a no-op for system allocator
func (sa *SystemAllocatorImpl) Reset() {
	// System allocator doesn't support reset
}

// Helper methods

func (sa *SystemAllocatorImpl) trackAllocation(ptr unsafe.Pointer, size uintptr) {
	info := &AllocationInfo{
		Size:      size,
		Timestamp: getTimestamp(),
	}

	if sa.config.EnableDebug {
		info.StackTrace = captureStackTrace()
	}

	sa.mu.Lock()
	sa.activeAllocations[ptr] = info
	if len(sa.activeAllocations) > sa.peakAllocations {
		sa.peakAllocations = len(sa.activeAllocations)
	}
	sa.mu.Unlock()
}

func (sa *SystemAllocatorImpl) untrackAllocation(ptr unsafe.Pointer) uintptr {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if info, exists := sa.activeAllocations[ptr]; exists {
		delete(sa.activeAllocations, ptr)
		return info.Size
	}
	return 0
}

// Platform-specific system allocation functions

// systemAlloc allocates memory from the system
func systemAlloc(size uintptr) unsafe.Pointer {
	// Use Go's memory allocator for bootstrap
	// In a real implementation, this would use VirtualAlloc on Windows or mmap on Linux
	if size == 0 {
		return nil
	}

	// Use runtime.malloc functionality via make and unsafe
	slice := make([]byte, size)
	if len(slice) < int(size) {
		return nil
	}

	// Keep a reference to prevent GC (in production this would be managed differently)
	runtime.KeepAlive(slice)

	return unsafe.Pointer(&slice[0])
}

// systemFree frees system memory
func systemFree(ptr unsafe.Pointer) {
	// In Go, we can't actually free memory directly
	// The GC will handle it when the slice goes out of scope
	// In a real implementation, this would use VirtualFree or munmap
}

// Utility functions

// alignUp aligns a size up to the nearest multiple of alignment
func alignUp(size, alignment uintptr) uintptr {
	return (size + alignment - 1) &^ (alignment - 1)
}

// copyMemory copies memory from src to dst
func copyMemory(dst, src unsafe.Pointer, size uintptr) {
	// Use Go's copy function
	dstSlice := (*[1 << 30]byte)(dst)[:size:size]
	srcSlice := (*[1 << 30]byte)(src)[:size:size]
	copy(dstSlice, srcSlice)
}

// getTimestamp returns current timestamp (simplified)
func getTimestamp() int64 {
	return 0 // Simplified for bootstrap
}

// captureStackTrace captures the current stack trace
func captureStackTrace() []uintptr {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	return pcs[:n]
}

// getSystemMemory returns system memory usage
func getSystemMemory() uintptr {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return uintptr(m.Sys)
}

// Memory leak detection

// CheckLeaks checks for memory leaks
func (sa *SystemAllocatorImpl) CheckLeaks() []LeakInfo {
	if !sa.config.EnableLeakCheck || !sa.config.EnableTracking {
		return nil
	}

	sa.mu.RLock()
	defer sa.mu.RUnlock()

	var leaks []LeakInfo
	for ptr, info := range sa.activeAllocations {
		leaks = append(leaks, LeakInfo{
			Pointer:    ptr,
			Size:       info.Size,
			Timestamp:  info.Timestamp,
			StackTrace: info.StackTrace,
		})
	}

	return leaks
}

// LeakInfo represents information about a memory leak
type LeakInfo struct {
	Pointer    unsafe.Pointer
	Size       uintptr
	Timestamp  int64
	StackTrace []uintptr
}

// FormatLeaks formats leak information for display
func FormatLeaks(leaks []LeakInfo) string {
	if len(leaks) == 0 {
		return "No memory leaks detected"
	}

	result := fmt.Sprintf("Detected %d memory leaks:\n", len(leaks))
	for i, leak := range leaks {
		result += fmt.Sprintf("  Leak %d: %d bytes at %p\n", i+1, leak.Size, leak.Pointer)
		if len(leak.StackTrace) > 0 {
			result += "    Stack trace:\n"
			frames := runtime.CallersFrames(leak.StackTrace)
			for {
				frame, more := frames.Next()
				result += fmt.Sprintf("      %s:%d %s\n", frame.File, frame.Line, frame.Function)
				if !more {
					break
				}
			}
		}
	}

	return result
}

// Global allocation functions for convenience

// Alloc allocates memory using the global allocator
func Alloc(size uintptr) unsafe.Pointer {
	if GlobalAllocator == nil {
		// Fall back to system allocator if not initialized
		panic("Global allocator not initialized")
	}
	return GlobalAllocator.Alloc(size)
}

// Free frees memory using the global allocator
func Free(ptr unsafe.Pointer) {
	if GlobalAllocator == nil {
		panic("Global allocator not initialized")
	}
	GlobalAllocator.Free(ptr)
}

// Realloc reallocates memory using the global allocator
func Realloc(ptr unsafe.Pointer, newSize uintptr) unsafe.Pointer {
	if GlobalAllocator == nil {
		panic("Global allocator not initialized")
	}
	return GlobalAllocator.Realloc(ptr, newSize)
}

// GetStats returns global allocator statistics
func GetStats() AllocatorStats {
	if GlobalAllocator == nil {
		return AllocatorStats{}
	}
	return GlobalAllocator.Stats()
}
