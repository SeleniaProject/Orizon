// Test suite for constraint handling functionality.
// This module provides comprehensive testing for the constraint system.

package types

import (
	"testing"
)

// TestSimpleConstraintSolver tests the basic functionality of the simple constraint solver.
func TestSimpleConstraintSolver(t *testing.T) {
	solver := NewSimpleConstraintSolver()

	// Test initial state.
	if solver.ConstraintCount() != 0 {
		t.Errorf("Expected 0 constraints, got %d", solver.ConstraintCount())
	}

	// Create some basic types for testing.
	intType := &Type{Kind: TypeKindInt32}
	floatType := &Type{Kind: TypeKindFloat64}

	// Test adding equality constraint.
	solver.AddEqualityConstraint(intType, intType)

	if solver.ConstraintCount() != 1 {
		t.Errorf("Expected 1 constraint after adding equality, got %d", solver.ConstraintCount())
	}

	// Test solving basic constraints.
	if err := solver.SolveBasic(); err != nil {
		t.Errorf("Failed to solve basic constraint: %v", err)
	}

	// Test subtype constraint.
	solver.AddSubtypeConstraint(intType, floatType)

	if solver.ConstraintCount() != 2 {
		t.Errorf("Expected 2 constraints after adding subtype, got %d", solver.ConstraintCount())
	}

	// Test clearing constraints.
	solver.Clear()

	if solver.ConstraintCount() != 0 {
		t.Errorf("Expected 0 constraints after clear, got %d", solver.ConstraintCount())
	}
}

// TestConstraintGenerator tests the constraint generator functionality.
func TestConstraintGenerator(t *testing.T) {
	generator := NewConstraintGenerator()

	// Test initial state.
	constraints := generator.GetConstraints()
	if len(constraints) != 0 {
		t.Errorf("Expected 0 constraints, got %d", len(constraints))
	}

	// Test fresh type variable generation.
	tv1 := generator.FreshTypeVariable()
	tv2 := generator.FreshTypeVariable()

	if tv1 == nil || tv2 == nil {
		t.Error("Fresh type variables should not be nil")
	}

	// Type variables should be different.
	if tv1 == tv2 {
		t.Error("Fresh type variables should be different instances")
	}

	// Test constraint generation (basic stub test).
	expr := "test_expression" // Placeholder expression
	resultType, resultConstraints, err := generator.GenerateConstraints(expr)
	if err != nil {
		t.Errorf("Constraint generation failed: %v", err)
	}

	if resultType == nil {
		t.Error("Result type should not be nil")
	}

	if resultConstraints == nil {
		t.Error("Result constraints should not be nil")
	}
}

// TestTypeConstraintValidation tests constraint validation functionality.
func TestTypeConstraintValidation(t *testing.T) {
	// Test unification constraints.
	intType1 := &Type{Kind: TypeKindInt32}
	intType2 := &Type{Kind: TypeKindInt32}
	floatType := &Type{Kind: TypeKindFloat64}

	solver := NewSimpleConstraintSolver()

	// Test successful unification.
	solver.AddEqualityConstraint(intType1, intType2)

	if err := solver.SolveBasic(); err != nil {
		t.Errorf("Expected successful unification of same types: %v", err)
	}

	solver.Clear()

	// Test failed unification.
	solver.AddEqualityConstraint(intType1, floatType)

	if err := solver.SolveBasic(); err == nil {
		t.Error("Expected unification to fail for different types")
	}
}

// TestNumericTypeHandling tests numeric type constraint handling.
func TestNumericTypeHandling(t *testing.T) {
	testCases := []struct {
		kind     TypeKind
		expected bool
	}{
		{TypeKindInt32, true},
		{TypeKindFloat64, true},
		{TypeKindString, false},
		{TypeKindBool, false},
		{TypeKindVoid, false},
	}

	for _, tc := range testCases {
		result := isNumericType(tc.kind)
		if result != tc.expected {
			t.Errorf("isNumericType(%s) = %v, expected %v", tc.kind.String(), result, tc.expected)
		}
	}
}

// TestConstraintSolverError tests error handling in constraint solving.
func TestConstraintSolverError(t *testing.T) {
	solver := NewSimpleConstraintSolver()

	// Test with nil types.
	solver.AddEqualityConstraint(nil, &Type{Kind: TypeKindInt32})

	if err := solver.SolveBasic(); err == nil {
		t.Error("Expected error when solving constraint with nil type")
	}

	solver.Clear()

	// Test subtype constraint with incompatible types.
	stringType := &Type{Kind: TypeKindString}
	boolType := &Type{Kind: TypeKindBool}

	solver.AddSubtypeConstraint(stringType, boolType)

	if err := solver.SolveBasic(); err == nil {
		t.Error("Expected error when solving incompatible subtype constraint")
	}
}
