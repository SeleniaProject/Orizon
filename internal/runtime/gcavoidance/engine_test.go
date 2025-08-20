// Package gcavoidance tests - Comprehensive test suite for GC Avoidance Phase 3.1.2
package gcavoidance

import (
	"testing"
	"time"
)

func TestEngine_Creation(t *testing.T) {
	engine := NewEngine()

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

func TestEngine_Allocation(t *testing.T) {
	engine := NewEngine()

	// Test basic allocation.
	ptr := engine.Allocate(100, "test_function")
	if ptr == 0 {
		t.Error("Allocation should succeed")
	}

	// Check statistics.
	stats := engine.GetStatistics()
	if stats["total_allocations"].(int64) != 1 {
		t.Error("Total allocations should be 1")
	}
}

func TestEngine_EnableDisable(t *testing.T) {
	engine := NewEngine()

	// Test disable.
	engine.Disable()

	if engine.enabled {
		t.Error("Engine should be disabled")
	}

	// Test enable.
	engine.Enable()

	if !engine.enabled {
		t.Error("Engine should be enabled")
	}
}

func TestLifetimeTracker_ScopeManagement(t *testing.T) {
	tracker := NewLifetimeTracker()

	// Test push scope.
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

	// Test nested scope.
	scope2 := tracker.PushScope("function2")
	if scope2.Parent != scope1 {
		t.Error("Scope2 parent should be scope1")
	}

	if len(scope1.Children) != 1 || scope1.Children[0] != scope2 {
		t.Error("Scope1 should have scope2 as child")
	}

	// Test pop scope.
	tracker.PopScope()

	if tracker.current != scope1 {
		t.Error("Current scope should be scope1 after pop")
	}

	tracker.PopScope()

	if tracker.current != nil {
		t.Error("Current scope should be nil after popping all")
	}
}

func TestLifetimeTracker_AllocationTracking(t *testing.T) {
	tracker := NewLifetimeTracker()
	scope := tracker.PushScope("test_function")

	// Test allocation tracking.
	ptr := uintptr(0x1000) // Dummy pointer
	tracker.Track(ptr, 100, StackAlloc)

	// Check allocation was recorded.
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

	if alloc.AllocType != StackAlloc {
		t.Error("Allocation type mismatch")
	}

	if alloc.Scope != scope {
		t.Error("Allocation scope mismatch")
	}

	// Check scope has allocation.
	if len(scope.Allocations) != 1 || scope.Allocations[0] != ptr {
		t.Error("Scope should track the allocation")
	}
}

func TestRefCounter_BasicOperations(t *testing.T) {
	counter := NewRefCounter()
	ptr := uintptr(0x2000) // Dummy pointer

	// Test tracking.
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

	// Test increment.
	counter.Increment(ptr)

	if entry.Count != 2 {
		t.Error("Count should be 2 after increment")
	}

	// Test decrement.
	counter.Decrement(ptr)

	if entry.Count != 1 {
		t.Error("Count should be 1 after decrement")
	}

	// Test decrement to zero.
	counter.Decrement(ptr)

	// Entry should be cleaned up.
	_, exists = counter.counters[ptr]
	if exists {
		t.Error("Entry should be cleaned up when count reaches zero")
	}
}

func TestStackManager_FrameManagement(t *testing.T) {
	manager := NewStackManager(10) // 10 frame limit

	// Test initial state.
	if manager.depth != 0 {
		t.Error("Initial depth should be 0")
	}

	if manager.current != nil {
		t.Error("Initial current frame should be nil")
	}

	// Test allocation (should create first frame).
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

	// Test pop frame.
	manager.PopFrame()

	if manager.depth != 0 {
		t.Error("Depth should be 0 after pop")
	}

	if manager.current != nil {
		t.Error("Current frame should be nil after pop")
	}
}

func TestStackManager_LargeAllocation(t *testing.T) {
	manager := NewStackManager(10)

	// Test large allocation that exceeds frame size.
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

func TestEscapeAnalyzer_Prediction(t *testing.T) {
	analyzer := NewEscapeAnalyzer()

	// Test initial prediction (should be conservative).
	if !analyzer.WillEscape("unknown_function") {
		t.Error("Should predict escape for unknown function (conservative)")
	}

	// Test recording escape events.
	analyzer.RecordEscape("test_function", false) // No escape
	analyzer.RecordEscape("test_function", false) // No escape
	analyzer.RecordEscape("test_function", true)  // Escape

	// Check pattern was created.
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

func TestEscapeAnalyzer_ConfidenceBuilding(t *testing.T) {
	analyzer := NewEscapeAnalyzer()

	// Record many non-escape events.
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

	// Should predict no escape with high confidence.
	if analyzer.WillEscape("reliable_function") {
		t.Error("Should predict no escape for well-known non-escaping function")
	}
}

func TestEngine_IntegratedWorkflow(t *testing.T) {
	engine := NewEngine()

	// Start scope.
	scope := engine.lifetimeTracker.PushScope("main_function")
	if scope == nil {
		t.Fatal("Failed to create scope")
	}

	// Perform allocations.
	ptr1 := engine.Allocate(100, "main_function")
	ptr2 := engine.Allocate(200, "main_function")

	if ptr1 == 0 || ptr2 == 0 {
		t.Error("Allocations should succeed")
	}

	// Check statistics.
	stats := engine.GetStatistics()
	if stats["total_allocations"].(int64) != 2 {
		t.Error("Should have 2 total allocations")
	}

	// End scope.
	engine.lifetimeTracker.PopScope()

	// Verify the workflow completed successfully.
	finalStats := engine.GetStatistics()
	if finalStats["total_allocations"].(int64) != 2 {
		t.Error("Total allocations should remain 2")
	}
}

func TestAllocationType_String(t *testing.T) {
	tests := []struct {
		expected  string
		allocType AllocType
	}{
		{StackAlloc, "stack"},
		{RefCountAlloc, "refcount"},
		{EscapedAlloc, "escaped"},
	}

	for _, test := range tests {
		if test.allocType.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.allocType.String())
		}
	}
}

func TestEngine_Statistics(t *testing.T) {
	engine := NewEngine()

	// Perform some operations.
	engine.Allocate(100, "test1")
	engine.Allocate(200, "test2")

	stats := engine.GetStatistics()

	// Verify required fields exist.
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

	// Test string representation.
	str := engine.String()
	if len(str) == 0 {
		t.Error("String representation should not be empty")
	}
}

func TestPerformanceBasics(t *testing.T) {
	engine := NewEngine()

	// Test performance of basic operations.
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

func TestEngine_MultiThreadedSafety(t *testing.T) {
	engine := NewEngine()

	// Test concurrent access.
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				ptr := engine.Allocate(uintptr(10+j), "concurrent_test")
				if ptr == 0 {
					t.Errorf("Goroutine %d allocation %d failed", id, j)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines.
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := engine.GetStatistics()
	if stats["total_allocations"].(int64) != 1000 {
		t.Error("Should have 1000 total allocations from concurrent access")
	}
}

func TestRefCounter_ConcurrentOperations(t *testing.T) {
	counter := NewRefCounter()
	ptr := uintptr(0x3000)

	counter.Track(ptr)

	// Test concurrent increments only (to avoid zero reaching in concurrent environment).
	done := make(chan bool, 10)

	// 10 goroutines incrementing.
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				counter.Increment(ptr)
			}
			done <- true
		}()
	}

	// Wait for all goroutines.
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still have the entry (started with 1, added 100, so count = 101).
	entry, exists := counter.counters[ptr]
	if !exists {
		t.Error("Entry should still exist after concurrent operations")
	} else if entry.Count <= 1 {
		t.Error("Count should be greater than 1 after increments")
	}
}

func TestStackManager_OverflowProtection(t *testing.T) {
	manager := NewStackManager(2) // Very small limit

	// Should succeed for first allocation (creates frame 1).
	ptr1 := manager.Allocate(8000, "function1") // Force frame creation
	if ptr1 == 0 {
		t.Error("First allocation should succeed")
	}

	// Should succeed for second allocation (creates frame 2).
	ptr2 := manager.Allocate(8000, "function2") // Force new frame
	if ptr2 == 0 {
		t.Error("Second allocation should succeed")
	}

	// Third allocation should fail due to depth limit (would need frame 3).
	ptr3 := manager.Allocate(8000, "function3")
	if ptr3 != 0 {
		t.Error("Third allocation should fail due to stack depth limit")
	}
}

func TestEscapeAnalyzer_LearningPatterns(t *testing.T) {
	analyzer := NewEscapeAnalyzer()

	// Train with specific patterns.
	functions := []string{"allocator", "constructor", "getter", "setter"}
	escapeRates := []float64{0.9, 0.7, 0.1, 0.3} // Expected escape rates

	for fi, function := range functions {
		for i := 0; i < 100; i++ {
			// Simulate escape based on expected rate.
			escaped := float64(i%100)/100.0 < escapeRates[fi]
			analyzer.RecordEscape(function, escaped)
		}
	}

	// Check learned patterns.
	for fi, function := range functions {
		pattern := analyzer.patterns[function]
		if pattern == nil {
			t.Errorf("Pattern should exist for %s", function)

			continue
		}

		// Allow for some variance due to modulo rounding.
		if pattern.EscapeRate < escapeRates[fi]-0.1 || pattern.EscapeRate > escapeRates[fi]+0.1 {
			t.Errorf("Function %s: expected escape rate ~%.1f, got %.3f",
				function, escapeRates[fi], pattern.EscapeRate)
		}

		if pattern.Confidence < 0.8 {
			t.Errorf("Function %s should have high confidence after 100 samples", function)
		}
	}
}
