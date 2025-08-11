// gc_avoidance_clean_test.go - Tests for Clean GC Avoidance Implementation
package runtime

import (
	"testing"
	"time"
)

func TestGCAvoidanceEngine_Creation(t *testing.T) {
	engine := NewGCAvoidanceEngine()

	if engine == nil {
		t.Fatal("Failed to create GC avoidance engine")
	}

	if !engine.enabled {
		t.Error("Engine should be enabled by default")
	}

	if engine.lifetimeTracker == nil {
		t.Error("Lifetime tracker should be initialized")
	}

	if engine.refCounter == nil {
		t.Error("Reference counter should be initialized")
	}

	if engine.stackManager == nil {
		t.Error("Stack manager should be initialized")
	}

	if engine.escapeAnalyzer == nil {
		t.Error("Escape analyzer should be initialized")
	}
}

func TestGCAvoidanceEngine_Allocation(t *testing.T) {
	engine := NewGCAvoidanceEngine()

	// Test basic allocation
	ptr := engine.Allocate(100, "test_function")
	if ptr == 0 {
		t.Error("Allocation should succeed")
	}

	// Check statistics
	stats := engine.GetStatistics()
	if stats["total_allocations"].(int64) != 1 {
		t.Error("Total allocations should be 1")
	}
}

func TestGCAvoidanceEngine_EnableDisable(t *testing.T) {
	engine := NewGCAvoidanceEngine()

	// Test disable
	engine.Disable()
	if engine.enabled {
		t.Error("Engine should be disabled")
	}

	// Test enable
	engine.Enable()
	if !engine.enabled {
		t.Error("Engine should be enabled")
	}
}

func TestCleanLifetimeTracker_ScopeManagement(t *testing.T) {
	tracker := NewCleanLifetimeTracker()

	// Test push scope
	scope1 := tracker.PushScope("function1")
	if scope1 == nil {
		t.Fatal("Failed to create scope")
	}

	if scope1.Function != "function1" {
		t.Error("Scope function name mismatch")
	}

	if tracker.current != scope1 {
		t.Error("Current scope should be scope1")
	}

	// Test nested scope
	scope2 := tracker.PushScope("function2")
	if scope2.Parent != scope1 {
		t.Error("Scope2 parent should be scope1")
	}

	if len(scope1.Children) != 1 || scope1.Children[0] != scope2 {
		t.Error("Scope1 should have scope2 as child")
	}

	// Test pop scope
	tracker.PopScope()
	if tracker.current != scope1 {
		t.Error("Current scope should be scope1 after pop")
	}

	tracker.PopScope()
	if tracker.current != nil {
		t.Error("Current scope should be nil after popping all")
	}
}

func TestCleanLifetimeTracker_AllocationTracking(t *testing.T) {
	tracker := NewCleanLifetimeTracker()
	scope := tracker.PushScope("test_function")

	// Test allocation tracking
	ptr := uintptr(0x1000) // Dummy pointer
	tracker.Track(ptr, 100, CleanStackAlloc)

	// Check allocation was recorded
	if len(tracker.allocations) != 1 {
		t.Error("Should have 1 tracked allocation")
	}

	alloc, exists := tracker.allocations[ptr]
	if !exists {
		t.Fatal("Allocation should exist")
	}

	if alloc.Size != 100 {
		t.Error("Allocation size mismatch")
	}

	if alloc.AllocType != CleanStackAlloc {
		t.Error("Allocation type mismatch")
	}

	if alloc.Scope != scope {
		t.Error("Allocation scope mismatch")
	}

	// Check scope has allocation
	if len(scope.Allocations) != 1 || scope.Allocations[0] != ptr {
		t.Error("Scope should track the allocation")
	}
}

func TestCleanRefCounter_BasicOperations(t *testing.T) {
	counter := NewCleanRefCounter()
	ptr := uintptr(0x2000) // Dummy pointer

	// Test tracking
	counter.Track(ptr)

	if len(counter.counters) != 1 {
		t.Error("Should have 1 tracked pointer")
	}

	entry, exists := counter.counters[ptr]
	if !exists {
		t.Fatal("Entry should exist")
	}

	if entry.Count != 1 {
		t.Error("Initial count should be 1")
	}

	// Test increment
	counter.Increment(ptr)
	if entry.Count != 2 {
		t.Error("Count should be 2 after increment")
	}

	// Test decrement
	counter.Decrement(ptr)
	if entry.Count != 1 {
		t.Error("Count should be 1 after decrement")
	}

	// Test decrement to zero
	counter.Decrement(ptr)

	// Entry should be cleaned up
	_, exists = counter.counters[ptr]
	if exists {
		t.Error("Entry should be cleaned up when count reaches zero")
	}
}

func TestCleanStackManager_FrameManagement(t *testing.T) {
	manager := NewCleanStackManager(10) // 10 frame limit

	// Test initial state
	if manager.depth != 0 {
		t.Error("Initial depth should be 0")
	}

	if manager.current != nil {
		t.Error("Initial current frame should be nil")
	}

	// Test allocation (should create first frame)
	ptr := manager.Allocate(100, "test_function")
	if ptr == 0 {
		t.Error("Allocation should succeed")
	}

	if manager.depth != 1 {
		t.Error("Depth should be 1 after first allocation")
	}

	if manager.current == nil {
		t.Fatal("Current frame should exist")
	}

	if manager.current.Function != "test_function" {
		t.Error("Frame function name mismatch")
	}

	if manager.current.Used != 100 {
		t.Error("Frame used space should be 100")
	}

	// Test pop frame
	manager.PopFrame()
	if manager.depth != 0 {
		t.Error("Depth should be 0 after pop")
	}

	if manager.current != nil {
		t.Error("Current frame should be nil after pop")
	}
}

func TestCleanStackManager_LargeAllocation(t *testing.T) {
	manager := NewCleanStackManager(10)

	// Test large allocation that exceeds frame size
	ptr1 := manager.Allocate(4000, "function1") // Within 8KB limit
	if ptr1 == 0 {
		t.Error("First allocation should succeed")
	}

	ptr2 := manager.Allocate(5000, "function2") // Should create new frame
	if ptr2 == 0 {
		t.Error("Second allocation should succeed")
	}

	if manager.depth != 2 {
		t.Error("Should have 2 frames after large allocations")
	}
}

func TestCleanEscapeAnalyzer_Prediction(t *testing.T) {
	analyzer := NewCleanEscapeAnalyzer()

	// Test initial prediction (should be conservative)
	if !analyzer.WillEscape("unknown_function") {
		t.Error("Should predict escape for unknown function (conservative)")
	}

	// Test recording escape events
	analyzer.RecordEscape("test_function", false) // No escape
	analyzer.RecordEscape("test_function", false) // No escape
	analyzer.RecordEscape("test_function", true)  // Escape

	// Check pattern was created
	if len(analyzer.patterns) != 1 {
		t.Error("Should have 1 pattern recorded")
	}

	pattern, exists := analyzer.patterns["test_function"]
	if !exists {
		t.Fatal("Pattern should exist for test_function")
	}

	if pattern.SampleCount != 3 {
		t.Error("Sample count should be 3")
	}

	expectedRate := 1.0 / 3.0 // 1 escape out of 3 samples
	if pattern.EscapeRate < expectedRate-0.01 || pattern.EscapeRate > expectedRate+0.01 {
		t.Errorf("Escape rate should be approximately %.3f, got %.3f", expectedRate, pattern.EscapeRate)
	}
}

func TestCleanEscapeAnalyzer_ConfidenceBuilding(t *testing.T) {
	analyzer := NewCleanEscapeAnalyzer()

	// Record many non-escape events
	for i := 0; i < 20; i++ {
		analyzer.RecordEscape("reliable_function", false)
	}

	pattern := analyzer.patterns["reliable_function"]
	if pattern.EscapeRate != 0.0 {
		t.Error("Escape rate should be 0.0 for non-escaping function")
	}

	if pattern.Confidence < 0.5 {
		t.Error("Confidence should be reasonably high after many samples")
	}

	// Should predict no escape with high confidence
	if analyzer.WillEscape("reliable_function") {
		t.Error("Should predict no escape for well-known non-escaping function")
	}
}

func TestGCAvoidanceEngine_IntegratedWorkflow(t *testing.T) {
	engine := NewGCAvoidanceEngine()

	// Start scope
	scope := engine.lifetimeTracker.PushScope("main_function")
	if scope == nil {
		t.Fatal("Failed to create scope")
	}

	// Perform allocations
	ptr1 := engine.Allocate(100, "main_function")
	ptr2 := engine.Allocate(200, "main_function")

	if ptr1 == 0 || ptr2 == 0 {
		t.Error("Allocations should succeed")
	}

	// Check statistics
	stats := engine.GetStatistics()
	if stats["total_allocations"].(int64) != 2 {
		t.Error("Should have 2 total allocations")
	}

	// End scope
	engine.lifetimeTracker.PopScope()

	// Verify the workflow completed successfully
	finalStats := engine.GetStatistics()
	if finalStats["total_allocations"].(int64) != 2 {
		t.Error("Total allocations should remain 2")
	}
}

func TestAllocationType_String(t *testing.T) {
	tests := []struct {
		allocType CleanAllocType
		expected  string
	}{
		{CleanStackAlloc, "stack"},
		{CleanRefCountAlloc, "refcount"},
		{CleanEscapedAlloc, "escaped"},
	}

	for _, test := range tests {
		if test.allocType.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.allocType.String())
		}
	}
}

func TestGCAvoidanceEngine_Statistics(t *testing.T) {
	engine := NewGCAvoidanceEngine()

	// Perform some operations
	engine.Allocate(100, "test1")
	engine.Allocate(200, "test2")

	stats := engine.GetStatistics()

	// Verify required fields exist
	requiredFields := []string{
		"enabled", "total_allocations", "stack_allocations",
		"refcount_allocations", "escaped_allocations",
		"avoided_gc_cycles", "memory_saved",
	}

	for _, field := range requiredFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Statistics should include field: %s", field)
		}
	}

	// Test string representation
	str := engine.String()
	if len(str) == 0 {
		t.Error("String representation should not be empty")
	}
}

func TestPerformanceBasics(t *testing.T) {
	engine := NewGCAvoidanceEngine()

	// Test performance of basic operations
	start := time.Now()

	for i := 0; i < 1000; i++ {
		ptr := engine.Allocate(100, "perf_test")
		if ptr == 0 {
			t.Errorf("Allocation %d failed", i)
		}
	}

	duration := time.Since(start)
	if duration > time.Second {
		t.Errorf("1000 allocations took too long: %v", duration)
	}

	stats := engine.GetStatistics()
	if stats["total_allocations"].(int64) != 1000 {
		t.Error("Should have 1000 total allocations")
	}
}
