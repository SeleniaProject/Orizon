// Package types provides comprehensive tests for the exception effect system.
// This module tests exception tracking, try-catch typing, and exception safety.
package types

import (
	"testing"
)

// TestExceptionKind tests exception kind functionality.
func TestExceptionKind(t *testing.T) {
	// Test exception kind string representation.
	tests := []struct {
		kind     ExceptionKind
		expected string
	}{
		{ExceptionNone, "None"},
		{ExceptionRuntime, "Runtime"},
		{ExceptionNullPointer, "NullPointer"},
		{ExceptionIndexOutOfBounds, "IndexOutOfBounds"},
		{ExceptionDivisionByZero, "DivisionByZero"},
		{ExceptionIOError, "IOError"},
		{ExceptionFileNotFound, "FileNotFound"},
		{ExceptionNetworkTimeout, "NetworkTimeout"},
		{ExceptionDeadlock, "Deadlock"},
		{ExceptionSystemError, "SystemError"},
	}

	for _, test := range tests {
		result := test.kind.String()
		if result != test.expected {
			t.Errorf("ExceptionKind.String() = %s, expected %s", result, test.expected)
		}
	}
}

// TestExceptionSeverity tests exception severity functionality.
func TestExceptionSeverity(t *testing.T) {
	// Test severity string representation.
	tests := []struct {
		severity ExceptionSeverity

		expected string
	}{
		{ExceptionSeverityInfo, "Info"},
		{ExceptionSeverityWarning, "Warning"},
		{ExceptionSeverityError, "Error"},
		{ExceptionSeverityCritical, "Critical"},
		{ExceptionSeverityFatal, "Fatal"},
	}

	for _, test := range tests {
		result := test.severity.String()
		if result != test.expected {
			t.Errorf("ExceptionSeverity.String() = %s, expected %s", result, test.expected)
		}
	}
}

// TestExceptionSpec tests exception specification functionality.
func TestExceptionSpec(t *testing.T) {
	// Test exception spec creation.
	spec := NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError)
	if spec.Kind != ExceptionNullPointer {
		t.Errorf("NewExceptionSpec kind = %v, expected %v", spec.Kind, ExceptionNullPointer)
	}

	if spec.Severity != ExceptionSeverityError {
		t.Errorf("NewExceptionSpec severity = %v, expected %v", spec.Severity, ExceptionSeverityError)
	}

	// Test string representation.
	expected := "NullPointer[Error]"
	if spec.String() != expected {
		t.Errorf("ExceptionSpec.String() = %s, expected %s", spec.String(), expected)
	}

	// Test subtype relationships.
	parent := NewExceptionSpec(ExceptionRuntime, ExceptionSeverityError)
	child := NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError)

	parent.AddChild(child)

	if !child.IsSubtypeOf(parent) {
		t.Error("Child should be subtype of parent")
	}

	if parent.IsSubtypeOf(child) {
		t.Error("Parent should not be subtype of child")
	}
}

// TestExceptionSet tests exception set functionality.
func TestExceptionSet(t *testing.T) {
	set := NewExceptionSet()

	// Test empty set.
	if !set.IsEmpty() {
		t.Error("New ExceptionSet should be empty")
	}

	if set.Size() != 0 {
		t.Errorf("Empty set size = %d, expected 0", set.Size())
	}

	// Test adding exceptions.
	spec1 := NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError)
	spec2 := NewExceptionSpec(ExceptionIOError, ExceptionSeverityCritical)

	set.Add(spec1)

	if set.IsEmpty() {
		t.Error("Set should not be empty after adding exception")
	}

	if set.Size() != 1 {
		t.Errorf("Set size = %d, expected 1", set.Size())
	}

	set.Add(spec2)

	if set.Size() != 2 {
		t.Errorf("Set size = %d, expected 2", set.Size())
	}

	// Test contains.
	if !set.Contains(ExceptionNullPointer) {
		t.Error("Set should contain ExceptionNullPointer")
	}

	if !set.Contains(ExceptionIOError) {
		t.Error("Set should contain ExceptionIOError")
	}

	if set.Contains(ExceptionDeadlock) {
		t.Error("Set should not contain ExceptionDeadlock")
	}

	// Test get.
	retrieved, exists := set.Get(ExceptionNullPointer)
	if !exists {
		t.Error("Should be able to get existing exception")
	}

	if retrieved.Kind != ExceptionNullPointer {
		t.Errorf("Retrieved exception kind = %v, expected %v", retrieved.Kind, ExceptionNullPointer)
	}

	// Test remove.
	set.Remove(ExceptionNullPointer)

	if set.Contains(ExceptionNullPointer) {
		t.Error("Set should not contain removed exception")
	}

	if set.Size() != 1 {
		t.Errorf("Set size = %d, expected 1 after removal", set.Size())
	}
}

// TestExceptionSetOperations tests exception set operations.
func TestExceptionSetOperations(t *testing.T) {
	set1 := NewExceptionSet()
	set2 := NewExceptionSet()

	spec1 := NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError)
	spec2 := NewExceptionSpec(ExceptionIOError, ExceptionSeverityCritical)
	spec3 := NewExceptionSpec(ExceptionDeadlock, ExceptionSeverityFatal)

	set1.Add(spec1)
	set1.Add(spec2)

	set2.Add(spec2)
	set2.Add(spec3)

	// Test union.
	union := set1.Union(set2)
	if union.Size() != 3 {
		t.Errorf("Union size = %d, expected 3", union.Size())
	}

	if !union.Contains(ExceptionNullPointer) || !union.Contains(ExceptionIOError) || !union.Contains(ExceptionDeadlock) {
		t.Error("Union should contain all exceptions from both sets")
	}

	// Test intersection.
	intersection := set1.Intersection(set2)
	if intersection.Size() != 1 {
		t.Errorf("Intersection size = %d, expected 1", intersection.Size())
	}

	if !intersection.Contains(ExceptionIOError) {
		t.Error("Intersection should contain common exception")
	}

	// Test subtraction.
	subtract := set1.Subtract(set2)
	if subtract.Size() != 1 {
		t.Errorf("Subtract size = %d, expected 1", subtract.Size())
	}

	if !subtract.Contains(ExceptionNullPointer) {
		t.Error("Subtract should contain unique exception from first set")
	}
}

// TestTryBlock tests try block functionality.
func TestTryBlock(t *testing.T) {
	// Create a simple expression for testing.
	body := &Expression{
		BaseNode: BaseNode{
			Location: &SourceLocation{
				File:   "test.oriz",
				Line:   1,
				Column: 1,
			},
		},
		Value: "test",
	}

	tryBlock := NewTryBlock(body)

	if tryBlock.Body != body {
		t.Error("TryBlock should have correct body")
	}

	if len(tryBlock.CatchBlocks) != 0 {
		t.Error("New TryBlock should have no catch blocks")
	}

	// Test adding catch block.
	exceptionTypes := []*ExceptionSpec{
		NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError),
	}
	catchBlock := NewCatchBlock(exceptionTypes, "e", body)
	tryBlock.AddCatchBlock(catchBlock)

	if len(tryBlock.CatchBlocks) != 1 {
		t.Errorf("TryBlock should have 1 catch block, got %d", len(tryBlock.CatchBlocks))
	}

	// Test finally block.
	finallyBlock := NewFinallyBlock(body)
	tryBlock.SetFinallyBlock(finallyBlock)

	if tryBlock.FinallyBlock != finallyBlock {
		t.Error("TryBlock should have correct finally block")
	}
}

// TestCatchBlock tests catch block functionality.
func TestCatchBlock(t *testing.T) {
	body := &Expression{
		BaseNode: BaseNode{
			Location: &SourceLocation{
				File:   "test.oriz",
				Line:   1,
				Column: 1,
			},
		},
		Value: "test",
	}

	// Create exception hierarchy.
	runtime := NewExceptionSpec(ExceptionRuntime, ExceptionSeverityError)
	nullPointer := NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError)
	runtime.AddChild(nullPointer)

	// Test catch block that handles runtime exceptions.
	exceptionTypes := []*ExceptionSpec{runtime}
	catchBlock := NewCatchBlock(exceptionTypes, "e", body)

	// Should handle runtime exception.
	if !catchBlock.CanHandle(runtime) {
		t.Error("Catch block should handle runtime exception")
	}

	// Should handle null pointer exception (subtype of runtime).
	if !catchBlock.CanHandle(nullPointer) {
		t.Error("Catch block should handle null pointer exception")
	}

	// Should not handle unrelated exception.
	ioError := NewExceptionSpec(ExceptionIOError, ExceptionSeverityError)
	if catchBlock.CanHandle(ioError) {
		t.Error("Catch block should not handle IO error")
	}
}

// TestExceptionSignature tests exception signature functionality.
func TestExceptionSignature(t *testing.T) {
	signature := NewExceptionSignature()

	// Test new signature.
	if !signature.Throws.IsEmpty() {
		t.Error("New signature should have no thrown exceptions")
	}

	if signature.Safety != SafetyBasic {
		t.Errorf("New signature safety = %v, expected %v", signature.Safety, SafetyBasic)
	}

	// Test adding exceptions.
	spec := NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError)
	signature.Throws.Add(spec)

	if signature.Throws.IsEmpty() {
		t.Error("Signature should have thrown exceptions after adding")
	}

	// Test safety levels.
	signature.Safety = SafetyNoThrow
	if signature.Safety.String() != "NoThrow" {
		t.Errorf("Safety string = %s, expected NoThrow", signature.Safety.String())
	}
}

// TestExceptionAnalyzer tests exception analyzer functionality.
func TestExceptionAnalyzer(t *testing.T) {
	analyzer := NewExceptionAnalyzer()

	// Create a simple expression for testing.
	node := &Expression{
		BaseNode: BaseNode{
			Location: &SourceLocation{
				File:   "test.oriz",
				Line:   1,
				Column: 1,
			},
		},
		Value: "test",
	}

	signature, err := analyzer.AnalyzeExceptions(node)
	if err != nil {
		t.Fatalf("Exception analysis failed: %v", err)
	}

	if signature == nil {
		t.Error("Exception analysis should return a signature")
	}

	// Check statistics.
	stats := analyzer.statistics.GetStats()
	if stats.TotalAnalyses == 0 {
		t.Error("Statistics should show analysis was performed")
	}
}

// TestExceptionRecovery tests exception recovery functionality.
func TestExceptionRecovery(t *testing.T) {
	// Test recovery string representation.
	tests := []struct {
		recovery ExceptionRecovery
		expected string
	}{
		{RecoveryNone, "None"},
		{RecoveryRetry, "Retry"},
		{RecoveryFallback, "Fallback"},
		{RecoveryPropagate, "Propagate"},
		{RecoveryTerminate, "Terminate"},
		{RecoveryIgnore, "Ignore"},
		{RecoveryLog, "Log"},
		{RecoveryCustom, "Custom"},
	}

	for _, test := range tests {
		result := test.recovery.String()
		if result != test.expected {
			t.Errorf("ExceptionRecovery.String() = %s, expected %s", result, test.expected)
		}
	}
}

// BenchmarkExceptionSetOperations benchmarks exception set operations.
func BenchmarkExceptionSetOperations(b *testing.B) {
	set := NewExceptionSet()
	specs := make([]*ExceptionSpec, 100)

	// Prepare exception specs.
	for i := 0; i < 100; i++ {
		kind := ExceptionKind(i%20 + 1)      // Cycle through first 20 exception kinds
		severity := ExceptionSeverity(i % 5) // Cycle through severities
		specs[i] = NewExceptionSpec(kind, severity)
	}

	b.ResetTimer()

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			spec := specs[i%100]
			set.Add(spec)
		}
	})

	b.Run("Contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kind := ExceptionKind(i%20 + 1)
			set.Contains(kind)
		}
	})

	b.Run("Union", func(b *testing.B) {
		other := NewExceptionSet()
		for i := 0; i < 50; i++ {
			other.Add(specs[i])
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			set.Union(other)
		}
	})
}

// BenchmarkExceptionAnalysis benchmarks exception analysis performance.
func BenchmarkExceptionAnalysis(b *testing.B) {
	analyzer := NewExceptionAnalyzer()

	// Create a simple AST node for testing.
	node := &Expression{
		BaseNode: BaseNode{
			Location: &SourceLocation{
				File:   "test.oriz",
				Line:   1,
				Column: 1,
			},
		},
		Value: "test",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeExceptions(node)
		if err != nil {
			b.Fatalf("Exception analysis failed: %v", err)
		}
	}
}
