package typechecker_test

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

func TestTypeInferenceEngine_Basic(t *testing.T) {
	// Basic test to verify type inference engine can be created.
	tie := NewTypeInferenceEngine(nil)

	if tie == nil {
		t.Fatal("NewTypeInferenceEngine returned nil")
	}

	if tie.unificationVars == nil {
		t.Error("Unification variables map not initialized")
	}

	if tie.constraints == nil {
		t.Error("Constraints slice not initialized")
	}

	if tie.solutions == nil {
		t.Error("Solutions map not initialized")
	}
}

func TestTypeInferenceEngine_CreateUnificationVariable(t *testing.T) {
	tie := NewTypeInferenceEngine(nil)

	pos := position.Position{Line: 1, Column: 1}
	variable := tie.CreateUnificationVariable("test_context", pos)

	if variable == nil {
		t.Fatal("CreateUnificationVariable returned nil")
	}

	if variable.ID == "" {
		t.Error("Unification variable ID is empty")
	}

	if variable.CreationContext != "test_context" {
		t.Errorf("Expected context 'test_context', got '%s'", variable.CreationContext)
	}

	if variable.Kind != UnificationKindType {
		t.Errorf("Expected kind UnificationKindType, got %v", variable.Kind)
	}

	// Verify variable is stored.
	if _, exists := tie.unificationVars[variable.ID]; !exists {
		t.Error("Unification variable not stored in engine")
	}
}

func TestTypeInferenceEngine_AddConstraint(t *testing.T) {
	tie := NewTypeInferenceEngine(nil)

	type1 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"}
	type2 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"}
	pos := position.Position{Line: 1, Column: 1}

	constraint := &TypeConstraint{
		Kind:     ConstraintKindEquality,
		Left:     type1,
		Right:    type2,
		Position: pos,
		Message:  "Test constraint",
	}

	tie.AddConstraint(constraint)

	if len(tie.constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(tie.constraints))
	}

	if tie.constraints[0] != constraint {
		t.Error("Constraint not properly stored")
	}
}

func TestTypeInferenceEngine_Unify_PrimitiveTypes(t *testing.T) {
	tie := NewTypeInferenceEngine(nil)

	// Test unifying identical primitive types.
	type1 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"}
	type2 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"}

	err := tie.Unify(type1, type2)
	if err != nil {
		t.Errorf("Unifying identical types should succeed, got error: %v", err)
	}
}

func TestTypeInferenceEngine_Unify_UnificationVariable(t *testing.T) {
	tie := NewTypeInferenceEngine(nil)

	// Create unification variable.
	pos := position.Position{Line: 1, Column: 1}
	variable := tie.CreateUnificationVariable("test_var", pos)

	// Create variable type.
	varType := &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: variable.ID}
	concreteType := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"}

	// Unify variable with concrete type.
	err := tie.Unify(varType, concreteType)
	if err != nil {
		t.Errorf("Unifying variable with concrete type should succeed, got error: %v", err)
	}

	// Check that variable was bound.
	if variable.Solution == nil {
		t.Error("Variable should be bound after unification")
	}

	if variable.Solution.Kind != parser.HIRTypePrimitive || variable.Solution.Data != "i32" {
		t.Errorf("Variable bound to wrong type: %v", variable.Solution)
	}
}

func TestTypeInferenceEngine_InferExpression_Literal(t *testing.T) {
	tie := NewTypeInferenceEngine(nil)

	// Create a literal expression.
	expr := &parser.HIRExpression{
		Kind: parser.HIRExprLiteral,
		Data: "test_literal",
	}

	inferredType, err := tie.InferExpression(expr, nil)
	if err != nil {
		t.Errorf("Inferring literal type should succeed, got error: %v", err)
	}

	if inferredType == nil {
		t.Fatal("Inferred type is nil")
	}

	if inferredType.Kind != parser.HIRTypePrimitive {
		t.Errorf("Expected primitive type for literal, got %v", inferredType.Kind)
	}
}
