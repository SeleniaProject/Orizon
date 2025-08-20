package hir

import (
	"strings"
	"testing"
)

// =============================================================================.
// Type Inference Engine Tests.
// =============================================================================.

func TestTypeInferenceEngine_Creation(t *testing.T) {
	engine := NewTypeInferenceEngine()

	if engine == nil {
		t.Fatal("Failed to create type inference engine")
	}

	if engine.typeVarCounter != 0 {
		t.Errorf("Expected initial type variable counter to be 0, got %d", engine.typeVarCounter)
	}

	if len(engine.constraints) != 0 {
		t.Errorf("Expected empty constraints list, got %d items", len(engine.constraints))
	}

	if len(engine.errors) != 0 {
		t.Errorf("Expected empty errors list, got %d items", len(engine.errors))
	}
}

func TestTypeInferenceEngine_FreshTypeVariable(t *testing.T) {
	engine := NewTypeInferenceEngine()

	// Generate first type variable.
	var1 := engine.freshTypeVariable()
	if var1.Kind != TypeKindVariable {
		t.Errorf("Expected TypeKindVariable, got %v", var1.Kind)
	}

	if var1.Name != "'t1" {
		t.Errorf("Expected name 't1', got %s", var1.Name)
	}

	if var1.VariableID == nil || *var1.VariableID != 1 {
		t.Errorf("Expected variable ID 1, got %v", var1.VariableID)
	}

	// Generate second type variable.
	var2 := engine.freshTypeVariable()
	if var2.Name != "'t2" {
		t.Errorf("Expected name 't2', got %s", var2.Name)
	}

	if var2.VariableID == nil || *var2.VariableID != 2 {
		t.Errorf("Expected variable ID 2, got %v", var2.VariableID)
	}
}

func TestTypeInferenceEngine_InferLiteral(t *testing.T) {
	engine := NewTypeInferenceEngine()

	// Create integer literal.
	intLiteral := &HIRLiteral{
		Value: 42,
		Type: TypeInfo{
			Kind: TypeKindInteger,
			Name: "int",
		},
	}

	result, err := engine.inferLiteral(intLiteral)
	if err != nil {
		t.Fatalf("Failed to infer literal type: %v", err)
	}

	if result.Kind != TypeKindInteger {
		t.Errorf("Expected TypeKindInteger, got %v", result.Kind)
	}

	if result.Name != "int" {
		t.Errorf("Expected name 'int', got %s", result.Name)
	}
}

// =============================================================================.
// Substitution Tests.
// =============================================================================.

func TestSubstitution_Creation(t *testing.T) {
	subst := NewSubstitution()

	if subst == nil {
		t.Fatal("Failed to create substitution")
	}

	if len(subst.mappings) != 0 {
		t.Errorf("Expected empty mappings, got %d items", len(subst.mappings))
	}
}

func TestSubstitution_AddAndApply(t *testing.T) {
	subst := NewSubstitution()

	// Create a type variable.
	varID := 1
	variable := TypeInfo{
		Kind:       TypeKindVariable,
		Name:       "'t1",
		VariableID: &varID,
	}

	// Create a concrete type.
	concrete := TypeInfo{
		Kind: TypeKindInteger,
		Name: "int",
	}

	// Add substitution.
	subst.Add(varID, concrete)

	// Apply substitution to the variable.
	result := subst.Apply(variable)

	if result.Kind != TypeKindInteger {
		t.Errorf("Expected TypeKindInteger after substitution, got %v", result.Kind)
	}

	if result.Name != "int" {
		t.Errorf("Expected name 'int' after substitution, got %s", result.Name)
	}

	// Apply substitution to non-variable type should return unchanged.
	result = subst.Apply(concrete)
	if result.Kind != TypeKindInteger || result.Name != "int" {
		t.Error("Expected concrete type to remain unchanged")
	}
}

// =============================================================================.
// Unification Tests.
// =============================================================================.

func TestUnification_SameTypes(t *testing.T) {
	engine := NewTypeInferenceEngine()

	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}

	err := engine.unify(intType, intType)
	if err != nil {
		t.Errorf("Expected unification of same types to succeed, got error: %v", err)
	}
}

func TestUnification_DifferentPrimitives(t *testing.T) {
	engine := NewTypeInferenceEngine()

	intType := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	stringType := TypeInfo{Kind: TypeKindString, Name: "string"}

	err := engine.unify(intType, stringType)
	if err == nil {
		t.Error("Expected unification of different primitive types to fail")
	}
}

func TestUnification_VariableWithConcrete(t *testing.T) {
	engine := NewTypeInferenceEngine()

	variable := engine.freshTypeVariable()
	concrete := TypeInfo{Kind: TypeKindInteger, Name: "int"}

	err := engine.unify(variable, concrete)
	if err != nil {
		t.Errorf("Expected variable-concrete unification to succeed, got error: %v", err)
	}

	// Check that substitution was added.
	result := engine.substitutions.Apply(variable)
	if result.Kind != TypeKindInteger {
		t.Error("Expected variable to be substituted with concrete type")
	}
}

// =============================================================================.
// Integration Tests.
// =============================================================================.

func TestInferenceIntegration_SimpleExpression(t *testing.T) {
	engine := NewTypeInferenceEngine()

	// Create a simple integer literal.
	literal := &HIRLiteral{
		Value: 42,
		Type: TypeInfo{
			Kind: TypeKindInteger,
			Name: "int",
		},
	}

	result, err := engine.InferType(literal)
	if err != nil {
		t.Fatalf("Failed to infer type for literal: %v", err)
	}

	if result.Kind != TypeKindInteger {
		t.Errorf("Expected TypeKindInteger, got %v", result.Kind)
	}
}

func TestInferenceIntegration_ErrorCollection(t *testing.T) {
	engine := NewTypeInferenceEngine()

	// Add some errors manually to test error formatting.
	engine.addError(ErrorTypeMismatch, "Type mismatch", "test.oriz:5", nil, nil, "test context")
	engine.addError(ErrorUndefinedVariable, "Undefined variable", "test.oriz:10", nil, nil, "")

	errors := engine.formatErrors()
	if errors == "" {
		t.Error("Expected formatted errors to be non-empty")
	}

	expectedSubstrings := []string{"Type mismatch", "Undefined variable"}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(errors, substr) {
			t.Errorf("Expected error message to contain '%s'", substr)
		}
	}
}

// =============================================================================.
// Performance Tests.
// =============================================================================.

func BenchmarkFreshTypeVariable(b *testing.B) {
	engine := NewTypeInferenceEngine()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.freshTypeVariable()
	}
}

func BenchmarkSubstitutionApply(b *testing.B) {
	engine := NewTypeInferenceEngine()
	subst := NewSubstitution()

	// Create a complex type with multiple nested variables.
	var1 := engine.freshTypeVariable()
	var2 := engine.freshTypeVariable()

	complexType := TypeInfo{
		Kind:       TypeKindFunction,
		Parameters: []TypeInfo{var1, var2},
	}

	// Add substitutions.
	subst.Add(*var1.VariableID, TypeInfo{Kind: TypeKindInteger, Name: "int"})
	subst.Add(*var2.VariableID, TypeInfo{Kind: TypeKindString, Name: "string"})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		subst.Apply(complexType)
	}
}

func BenchmarkUnification(b *testing.B) {
	engine := NewTypeInferenceEngine()

	type1 := TypeInfo{Kind: TypeKindInteger, Name: "int"}
	type2 := TypeInfo{Kind: TypeKindInteger, Name: "int"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.clearState()
		engine.unify(type1, type2)
	}
}
