package runtime

import (
	"testing"
	"time"
	"unsafe"
)

func TestGCAvoidanceBasics(t *testing.T) {
	// Test basic GC avoidance system initialization
	config := GCAvoidanceConfig{
		EnableLifetimeTracking:  true,
		EnableRefCounting:       true,
		EnableStackOptimization: true,
		EnableEscapeAnalysis:    true,
		MaxStackDepth:           100,
		RefCountThreshold:       10,
		CycleDetectionInterval:  time.Second * 5,
	}

	system := NewGCAvoidanceSystem(config)
	if system == nil {
		t.Fatal("Failed to create GC avoidance system")
	}

	if !system.enabled {
		t.Error("System should be enabled by default")
	}

	// Test statistics
	stats := system.GetStatistics()
	if stats == nil {
		t.Error("Statistics should not be nil")
	}

	// Test string representation
	str := system.String()
	if str == "" {
		t.Error("String representation should not be empty")
	}

	t.Logf("GC Avoidance System: %s", str)
}

func TestLifetimeTracker(t *testing.T) {
	tracker := NewLifetimeTracker()
	if tracker == nil {
		t.Fatal("Failed to create lifetime tracker")
	}

	// Test scope management
	scope1 := tracker.EnterScope("main")
	if scope1 == nil {
		t.Fatal("Failed to enter scope")
	}
	if scope1.name != "main" {
		t.Errorf("Expected scope name 'main', got '%s'", scope1.name)
	}

	// Test allocation tracking
	data := make([]byte, 64)
	ptr := uintptr(unsafe.Pointer(&data[0]))
	trackingData := tracker.TrackAllocation(ptr, 64, "test_var")

	if trackingData == nil {
		t.Fatal("Failed to track allocation")
	}
	if trackingData.ptr != ptr {
		t.Error("Pointer mismatch in tracking data")
	}
	if trackingData.size != 64 {
		t.Error("Size mismatch in tracking data")
	}

	// Test nested scope
	scope2 := tracker.EnterScope("function")
	if scope2.parent != scope1 {
		t.Error("Parent scope not set correctly")
	}

	// Exit scopes
	exitedScope2 := tracker.ExitScope()
	if exitedScope2 != scope2 {
		t.Error("Wrong scope exited")
	}

	exitedScope1 := tracker.ExitScope()
	if exitedScope1 != scope1 {
		t.Error("Wrong scope exited")
	}
}

func TestOptimizedRefCountManager(t *testing.T) {
	rcm := NewOptimizedRefCountManager()
	if rcm == nil {
		t.Fatal("Failed to create ref count manager")
	}

	// Test reference counting
	data := make([]byte, 32)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	// Increment
	rcm.Increment(ptr)
	if counter, exists := rcm.counters[ptr]; !exists {
		t.Fatal("Counter not created")
	} else {
		if counter.strongCount != 1 {
			t.Errorf("Expected count 1, got %d", counter.strongCount)
		}
	}

	// Multiple increments
	rcm.Increment(ptr)
	rcm.Increment(ptr)
	if counter, exists := rcm.counters[ptr]; exists {
		if counter.strongCount != 3 {
			t.Errorf("Expected count 3, got %d", counter.strongCount)
		}
	}

	// Decrements
	rcm.Decrement(ptr)
	rcm.Decrement(ptr)
	if counter, exists := rcm.counters[ptr]; exists {
		if counter.strongCount != 1 {
			t.Errorf("Expected count 1, got %d", counter.strongCount)
		}
	}

	// Final decrement should remove counter
	rcm.Decrement(ptr)

	// Give time for cleanup
	time.Sleep(10 * time.Millisecond)

	if _, exists := rcm.counters[ptr]; exists {
		t.Error("Counter should be removed when count reaches zero")
	}
}

func TestStackManager(t *testing.T) {
	sm := NewStackManager(50)
	if sm == nil {
		t.Fatal("Failed to create stack manager")
	}

	// Test frame management
	frame := sm.PushFrame("test_function")
	if frame == nil {
		t.Fatal("Failed to push frame")
	}
	if frame.name != "test_function" {
		t.Errorf("Expected frame name 'test_function', got '%s'", frame.name)
	}

	// Test stack allocation
	ptr := sm.AllocateOnStack(128, "test_var")
	if ptr == 0 {
		t.Fatal("Failed to allocate on stack")
	}

	if alloc, exists := frame.allocations[ptr]; !exists {
		t.Error("Stack allocation not tracked")
	} else {
		if alloc.size != 128 {
			t.Errorf("Expected size 128, got %d", alloc.size)
		}
		if !alloc.isLive {
			t.Error("Allocation should be live")
		}
	}

	// Test frame cleanup
	sm.PopFrame()
	if sm.currentFrame != nil {
		t.Error("Current frame should be nil after pop")
	}
}

func TestEscapeAnalyzer(t *testing.T) {
	ea := NewEscapeAnalyzer()
	if ea == nil {
		t.Fatal("Failed to create escape analyzer")
	}

	// Test escape analysis
	data := make([]byte, 64)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	result := ea.AnalyzeEscape(ptr, "test_context")
	if result == nil {
		t.Fatal("Failed to analyze escape")
	}

	if result.ptr != ptr {
		t.Error("Pointer mismatch in escape result")
	}

	if result.confidence <= 0 || result.confidence > 1 {
		t.Errorf("Invalid confidence value: %f", result.confidence)
	}

	// Check that result is stored
	if stored, exists := ea.escapeResults[ptr]; !exists {
		t.Error("Escape result not stored")
	} else {
		if stored != result {
			t.Error("Stored result mismatch")
		}
	}
}

func TestIntegration(t *testing.T) {
	// Test integration of all components
	config := GCAvoidanceConfig{
		EnableLifetimeTracking:  true,
		EnableRefCounting:       true,
		EnableStackOptimization: true,
		EnableEscapeAnalysis:    true,
		MaxStackDepth:           20,
		RefCountThreshold:       5,
		CycleDetectionInterval:  time.Second,
	}

	system := NewGCAvoidanceSystem(config)

	// Simulate a function call with various allocations
	system.lifetimeTracker.EnterScope("integration_test")
	frame := system.stackManager.PushFrame("integration_test")

	if frame == nil {
		t.Fatal("Failed to push frame")
	}

	// Stack allocation
	stackPtr := system.stackManager.AllocateOnStack(64, "stack_var")
	if stackPtr != 0 {
		t.Logf("Stack allocation successful: %p", unsafe.Pointer(stackPtr))
	}

	// Tracked allocation
	data := make([]byte, 128)
	heapPtr := uintptr(unsafe.Pointer(&data[0]))
	trackingData := system.lifetimeTracker.TrackAllocation(heapPtr, 128, "heap_var")
	if trackingData != nil {
		t.Logf("Heap allocation tracked: %p", unsafe.Pointer(heapPtr))
	}

	// Reference counting
	system.refCountManager.Increment(heapPtr)
	system.refCountManager.Increment(heapPtr)

	// Escape analysis
	escapeResult := system.escapeAnalyzer.AnalyzeEscape(heapPtr, "integration")
	if escapeResult != nil {
		t.Logf("Escape analysis: escaped=%v, reason=%s",
			escapeResult.escaped, escapeResult.reason)
	}

	// Cleanup
	system.refCountManager.Decrement(heapPtr)
	system.refCountManager.Decrement(heapPtr)
	system.stackManager.PopFrame()
	system.lifetimeTracker.ExitScope()

	// Check final statistics
	stats := system.GetStatistics()
	t.Logf("Final statistics: %+v", stats)
}

// Benchmark tests
func BenchmarkGCAvoidanceSystem(b *testing.B) {
	config := GCAvoidanceConfig{
		EnableLifetimeTracking:  true,
		EnableRefCounting:       true,
		EnableStackOptimization: true,
		EnableEscapeAnalysis:    true,
		MaxStackDepth:           100,
		RefCountThreshold:       10,
		CycleDetectionInterval:  time.Second * 5,
	}

	system := NewGCAvoidanceSystem(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		system.lifetimeTracker.EnterScope("bench")
		frame := system.stackManager.PushFrame("bench")

		if frame != nil {
			ptr := system.stackManager.AllocateOnStack(64, "var")
			if ptr != 0 {
				system.refCountManager.Increment(ptr)
				system.refCountManager.Decrement(ptr)
			}
			system.stackManager.PopFrame()
		}
		system.lifetimeTracker.ExitScope()
	}
}

func BenchmarkRefCounting(b *testing.B) {
	rcm := NewOptimizedRefCountManager()
	data := make([]byte, 32)
	ptr := uintptr(unsafe.Pointer(&data[0]))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rcm.Increment(ptr)
		rcm.Decrement(ptr)
	}
}
