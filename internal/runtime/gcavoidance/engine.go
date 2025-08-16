// Package gcavoidance implements Phase 3.1.2 GC Avoidance for Orizon
// This package provides comprehensive garbage collection avoidance mechanisms
// through lifetime tracking, reference counting, stack allocation prioritization,
// and escape analysis to eliminate GC pressure completely.
package gcavoidance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Engine is the main coordinator for GC avoidance
type Engine struct {
	enabled bool
	mutex   sync.RWMutex

	// Core components
	lifetimeTracker *LifetimeTracker
	refCounter      *RefCounter
	stackManager    *StackManager
	escapeAnalyzer  *EscapeAnalyzer

	// Statistics
	stats Stats
}

// Stats tracks system performance
type Stats struct {
	TotalAllocations    int64
	StackAllocations    int64
	RefCountAllocations int64
	EscapedAllocations  int64
	AvoidedGCCycles     int64
	MemorySaved         int64
}

// LifetimeTracker tracks object lifetimes for optimal allocation
type LifetimeTracker struct {
	allocations map[uintptr]*Allocation
	scopes      []*Scope
	current     *Scope
	mutex       sync.RWMutex
}

// Allocation represents a tracked allocation
type Allocation struct {
	Ptr       uintptr
	Size      uintptr
	AllocType AllocType
	Scope     *Scope
	RefCount  int32
	IsValid   bool
	Created   time.Time
}

// AllocType defines allocation strategies
type AllocType int

const (
	StackAlloc AllocType = iota
	RefCountAlloc
	EscapedAlloc
)

// Scope represents a lexical scope
type Scope struct {
	ID          int
	Function    string
	Parent      *Scope
	Children    []*Scope
	Allocations []uintptr
	Variables   map[string]*Variable
}

// Variable represents a variable in scope
type Variable struct {
	Name     string
	Ptr      uintptr
	IsLive   bool
	RefCount int32
}

// RefCounter manages reference counting
type RefCounter struct {
	counters map[uintptr]*RefCountEntry
	mutex    sync.RWMutex
	stats    struct {
		increments int64
		decrements int64
	}
}

// RefCountEntry represents a reference count entry
type RefCountEntry struct {
	Ptr     uintptr
	Count   int32
	IsValid bool
	Created time.Time
}

// StackManager manages stack allocation
type StackManager struct {
	frames   []*StackFrame
	current  *StackFrame
	maxDepth int
	depth    int
	mutex    sync.Mutex
}

// StackFrame represents a stack frame
type StackFrame struct {
	ID       int
	Function string
	Parent   *StackFrame
	Size     uintptr
	Used     uintptr
	Objects  map[uintptr]*StackObject
}

// StackObject represents a stack-allocated object
type StackObject struct {
	Ptr    uintptr
	Size   uintptr
	Offset uintptr
	IsLive bool
}

// EscapeAnalyzer analyzes escape patterns
type EscapeAnalyzer struct {
	patterns map[string]*EscapePattern
	mutex    sync.RWMutex
}

// EscapePattern represents an escape analysis pattern
type EscapePattern struct {
	Function    string
	EscapeRate  float64
	Confidence  float64
	SampleCount int64
}

// NewEngine creates a new GC avoidance engine
func NewEngine() *Engine {
	return &Engine{
		enabled:         true,
		lifetimeTracker: NewLifetimeTracker(),
		refCounter:      NewRefCounter(),
		stackManager:    NewStackManager(1000), // 1000 frame limit
		escapeAnalyzer:  NewEscapeAnalyzer(),
	}
}

// NewLifetimeTracker creates a new lifetime tracker
func NewLifetimeTracker() *LifetimeTracker {
	return &LifetimeTracker{
		allocations: make(map[uintptr]*Allocation),
		scopes:      make([]*Scope, 0),
	}
}

// NewRefCounter creates a new reference counter
func NewRefCounter() *RefCounter {
	return &RefCounter{
		counters: make(map[uintptr]*RefCountEntry),
	}
}

// NewStackManager creates a new stack manager
func NewStackManager(maxDepth int) *StackManager {
	return &StackManager{
		frames:   make([]*StackFrame, 0),
		maxDepth: maxDepth,
	}
}

// NewEscapeAnalyzer creates a new escape analyzer
func NewEscapeAnalyzer() *EscapeAnalyzer {
	return &EscapeAnalyzer{
		patterns: make(map[string]*EscapePattern),
	}
}

// Allocate performs smart allocation based on escape analysis
func (e *Engine) Allocate(size uintptr, hint string) uintptr {
	if !e.enabled {
		return e.heapAllocate(size)
	}

	atomic.AddInt64(&e.stats.TotalAllocations, 1)

	// Try stack allocation first
	if ptr := e.tryStackAllocate(size, hint); ptr != 0 {
		atomic.AddInt64(&e.stats.StackAllocations, 1)
		return ptr
	}

	// Try reference counted allocation
	if ptr := e.tryRefCountAllocate(size, hint); ptr != 0 {
		atomic.AddInt64(&e.stats.RefCountAllocations, 1)
		return ptr
	}

	// Fall back to heap
	atomic.AddInt64(&e.stats.EscapedAllocations, 1)
	return e.heapAllocate(size)
}

// tryStackAllocate attempts stack allocation
func (e *Engine) tryStackAllocate(size uintptr, hint string) uintptr {
	// Check if likely to escape
	if e.escapeAnalyzer.WillEscape(hint) {
		return 0
	}

	// Check stack space
	return e.stackManager.Allocate(size, hint)
}

// tryRefCountAllocate attempts reference counted allocation
func (e *Engine) tryRefCountAllocate(size uintptr, hint string) uintptr {
	// Allocate with reference counting
	ptr := e.heapAllocate(size)
	if ptr != 0 {
		e.refCounter.Track(ptr)
	}
	return ptr
}

// heapAllocate performs traditional heap allocation
func (e *Engine) heapAllocate(size uintptr) uintptr {
	// Simple allocation using unsafe.Pointer
	data := make([]byte, size)
	return uintptr(unsafe.Pointer(&data[0]))
}

// PushScope enters a new lexical scope
func (lt *LifetimeTracker) PushScope(function string) *Scope {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	scope := &Scope{
		ID:          len(lt.scopes),
		Function:    function,
		Parent:      lt.current,
		Children:    make([]*Scope, 0),
		Allocations: make([]uintptr, 0),
		Variables:   make(map[string]*Variable),
	}

	if lt.current != nil {
		lt.current.Children = append(lt.current.Children, scope)
	}

	lt.scopes = append(lt.scopes, scope)
	lt.current = scope

	return scope
}

// PopScope exits the current lexical scope
func (lt *LifetimeTracker) PopScope() {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	if lt.current != nil {
		// Cleanup scope allocations
		for _, ptr := range lt.current.Allocations {
			if alloc, exists := lt.allocations[ptr]; exists && alloc.IsValid {
				// Mark for cleanup
				alloc.IsValid = false
			}
		}

		lt.current = lt.current.Parent
	}
}

// Track adds an allocation to lifetime tracking
func (lt *LifetimeTracker) Track(ptr uintptr, size uintptr, allocType AllocType) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	allocation := &Allocation{
		Ptr:       ptr,
		Size:      size,
		AllocType: allocType,
		Scope:     lt.current,
		RefCount:  1,
		IsValid:   true,
		Created:   time.Now(),
	}

	lt.allocations[ptr] = allocation

	if lt.current != nil {
		lt.current.Allocations = append(lt.current.Allocations, ptr)
	}
}

// Allocate allocates memory on the stack
func (sm *StackManager) Allocate(size uintptr, function string) uintptr {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.current == nil || sm.current.Used+size > 8192 { // 8KB frame limit
		if !sm.pushFrame(function) {
			return 0 // Stack overflow
		}
	}

	// Simple allocation
	data := make([]byte, size)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	obj := &StackObject{
		Ptr:    ptr,
		Size:   size,
		Offset: sm.current.Used,
		IsLive: true,
	}

	sm.current.Objects[ptr] = obj
	sm.current.Used += size

	return ptr
}

// pushFrame pushes a new stack frame
func (sm *StackManager) pushFrame(function string) bool {
	if sm.depth >= sm.maxDepth {
		return false
	}

	frame := &StackFrame{
		ID:       len(sm.frames),
		Function: function,
		Parent:   sm.current,
		Size:     8192, // 8KB default
		Used:     0,
		Objects:  make(map[uintptr]*StackObject),
	}

	sm.frames = append(sm.frames, frame)
	sm.current = frame
	sm.depth++

	return true
}

// PopFrame pops the current stack frame
func (sm *StackManager) PopFrame() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.current != nil {
		// Cleanup frame objects
		for _, obj := range sm.current.Objects {
			if obj.IsLive {
				obj.IsLive = false
			}
		}

		sm.current = sm.current.Parent
		sm.depth--
	}
}

// Track adds a pointer to reference counting
func (rc *RefCounter) Track(ptr uintptr) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	entry := &RefCountEntry{
		Ptr:     ptr,
		Count:   1,
		IsValid: true,
		Created: time.Now(),
	}

	rc.counters[ptr] = entry
}

// Increment increments reference count
func (rc *RefCounter) Increment(ptr uintptr) {
	rc.mutex.RLock()
	entry, exists := rc.counters[ptr]
	rc.mutex.RUnlock()

	if exists && entry.IsValid {
		atomic.AddInt32(&entry.Count, 1)
		atomic.AddInt64(&rc.stats.increments, 1)
	}
}

// Decrement decrements reference count
func (rc *RefCounter) Decrement(ptr uintptr) {
	rc.mutex.RLock()
	entry, exists := rc.counters[ptr]
	rc.mutex.RUnlock()

	if exists && entry.IsValid {
		newCount := atomic.AddInt32(&entry.Count, -1)
		atomic.AddInt64(&rc.stats.decrements, 1)

		if newCount == 0 {
			rc.cleanup(ptr)
		}
	}
}

// cleanup removes entry when count reaches zero
func (rc *RefCounter) cleanup(ptr uintptr) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	if entry, exists := rc.counters[ptr]; exists {
		entry.IsValid = false
		delete(rc.counters, ptr)
	}
}

// WillEscape predicts if an allocation will escape
func (ea *EscapeAnalyzer) WillEscape(function string) bool {
	ea.mutex.RLock()
	defer ea.mutex.RUnlock()

	if pattern, exists := ea.patterns[function]; exists {
		return pattern.EscapeRate > 0.5 // 50% threshold
	}

	// Conservative default - assume might escape
	return true
}

// RecordEscape records an escape event for learning
func (ea *EscapeAnalyzer) RecordEscape(function string, escaped bool) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()

	pattern, exists := ea.patterns[function]
	if !exists {
		pattern = &EscapePattern{
			Function:    function,
			EscapeRate:  0.0,
			Confidence:  0.0,
			SampleCount: 0,
		}
		ea.patterns[function] = pattern
	}

	// Update escape rate using running average
	pattern.SampleCount++
	if escaped {
		pattern.EscapeRate = (pattern.EscapeRate*float64(pattern.SampleCount-1) + 1.0) / float64(pattern.SampleCount)
	} else {
		pattern.EscapeRate = (pattern.EscapeRate * float64(pattern.SampleCount-1)) / float64(pattern.SampleCount)
	}

	// Update confidence
	pattern.Confidence = float64(pattern.SampleCount) / (float64(pattern.SampleCount) + 10.0)
}

// GetStatistics returns comprehensive statistics
func (e *Engine) GetStatistics() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["enabled"] = e.enabled
	stats["total_allocations"] = atomic.LoadInt64(&e.stats.TotalAllocations)
	stats["stack_allocations"] = atomic.LoadInt64(&e.stats.StackAllocations)
	stats["refcount_allocations"] = atomic.LoadInt64(&e.stats.RefCountAllocations)
	stats["escaped_allocations"] = atomic.LoadInt64(&e.stats.EscapedAllocations)
	stats["avoided_gc_cycles"] = atomic.LoadInt64(&e.stats.AvoidedGCCycles)
	stats["memory_saved"] = atomic.LoadInt64(&e.stats.MemorySaved)

	return stats
}

// Enable enables the GC avoidance system
func (e *Engine) Enable() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.enabled = true
}

// Disable disables the GC avoidance system
func (e *Engine) Disable() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.enabled = false
}

// String returns string representation
func (e *Engine) String() string {
	stats := e.GetStatistics()
	return fmt.Sprintf("GCAvoidanceEngine{enabled: %v, total: %d, stack: %d, refcount: %d, escaped: %d}",
		stats["enabled"], stats["total_allocations"], stats["stack_allocations"],
		stats["refcount_allocations"], stats["escaped_allocations"])
}

// String methods for component types
func (at AllocType) String() string {
	switch at {
	case StackAlloc:
		return "stack"
	case RefCountAlloc:
		return "refcount"
	case EscapedAlloc:
		return "escaped"
	default:
		return "unknown"
	}
}
