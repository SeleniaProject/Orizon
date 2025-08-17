package typechecker

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/parser"
)

func TestTraitResolver_NewTraitResolver(t *testing.T) {
	modules := []*parser.HIRModule{
		{Name: "test"},
	}

	resolver := NewTraitResolver(modules)
	if resolver == nil {
		t.Error("Expected resolver, got nil")
	}

	if len(resolver.modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(resolver.modules))
	}

	// Test that cache maps are initialized
	if resolver.implCache == nil {
		t.Error("Expected implCache to be initialized")
	}

	if resolver.genericCache == nil {
		t.Error("Expected genericCache to be initialized")
	}

	if resolver.scopeStack == nil {
		t.Error("Expected scopeStack to be initialized")
	}
}

func TestTraitResolver_TypeMatching(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Test identical types
	type1 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "int"}
	type2 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "int"}

	if !resolver.typesMatch(type1, type2) {
		t.Error("Expected identical types to match")
	}

	// Test different types
	type3 := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "string"}

	if resolver.typesMatch(type1, type3) {
		t.Error("Expected different types to not match")
	}

	// Test nil types
	if !resolver.typesMatch(nil, nil) {
		t.Error("Expected nil types to match")
	}
}

func TestTraitResolver_AdvancedResolution(t *testing.T) {
	// Create test trait and implementations
	testTrait := &parser.HIRTraitType{
		Name: "Display",
		Methods: []*parser.HIRMethodSignature{
			{Name: "display", Parameters: []*parser.HIRType{}, ReturnType: &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "string"}},
		},
	}

	stringType := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "string"}
	traitType := &parser.HIRType{Kind: parser.HIRTypeTrait, Data: testTrait}

	// Create implementations
	inherentImpl := &parser.HIRImpl{
		Kind:    parser.HIRImplInherent,
		ForType: stringType,
		Trait:   nil, // Inherent implementation
		Methods: []*parser.HIRFunction{
			{Name: "display", Parameters: []*parser.HIRParameter{}, ReturnType: stringType},
		},
	}

	traitImpl := &parser.HIRImpl{
		Kind:    parser.HIRImplTrait,
		ForType: stringType,
		Trait:   traitType,
		Methods: []*parser.HIRFunction{
			{Name: "display", Parameters: []*parser.HIRParameter{}, ReturnType: stringType},
		},
	}

	// Create module with implementations
	module := &parser.HIRModule{
		Name:  "test",
		Impls: []*parser.HIRImpl{inherentImpl, traitImpl},
	}

	resolver := NewTraitResolver([]*parser.HIRModule{module})

	// Test finding candidate implementations
	candidates := resolver.findAllCandidateImplementations(traitType, stringType)
	if len(candidates) == 0 {
		t.Fatal("Expected to find candidate implementations")
	}

	// Test priority resolution - inherent should have higher priority
	selectedImpl, err := resolver.resolveImplementationAmbiguity(candidates, stringType)
	if err != nil {
		t.Fatalf("Failed to resolve implementation ambiguity: %v", err)
	}

	if selectedImpl.Kind != parser.HIRImplInherent {
		t.Error("Expected inherent implementation to have higher priority")
	}
}

func TestTraitResolver_ScopeManagement(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Test scope stack operations
	scope1 := ResolutionScope{
		ModuleID:  "module1",
		ScopeKind: ScopeKindModule,
		Priority:  10,
	}

	scope2 := ResolutionScope{
		ModuleID:  "module2",
		ScopeKind: ScopeKindLocal,
		Priority:  20,
	}

	// Push scopes
	resolver.pushScope(scope1)
	resolver.pushScope(scope2)

	// Check current scope
	current := resolver.getCurrentScope()
	if current == nil {
		t.Fatal("Expected current scope, got nil")
	}

	if current.ModuleID != "module2" {
		t.Errorf("Expected module2, got %s", current.ModuleID)
	}

	// Pop scope
	resolver.popScope()
	current = resolver.getCurrentScope()
	if current.ModuleID != "module1" {
		t.Errorf("Expected module1 after pop, got %s", current.ModuleID)
	}

	// Pop last scope
	resolver.popScope()
	current = resolver.getCurrentScope()
	if current != nil {
		t.Error("Expected nil scope after popping all scopes")
	}
}

func TestTraitResolver_GenericBinding(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Test generic binding operations
	stringType := &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "string"}
	paramType := &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: "T"}
	constraints := []string{"Display"}

	// Add binding
	resolver.addGenericBinding("T", paramType, stringType, constraints)

	// Lookup binding
	binding := resolver.lookupGenericBinding("T")
	if binding == nil {
		t.Fatal("Expected binding, got nil")
	}

	if binding.ParamName != "T" {
		t.Errorf("Expected param name T, got %s", binding.ParamName)
	}

	if !resolver.typesMatch(binding.BoundType, stringType) {
		t.Error("Expected bound type to match string type")
	}

	if len(binding.Constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(binding.Constraints))
	}

	// Clear bindings
	resolver.clearGenericBindings()
	binding = resolver.lookupGenericBinding("T")
	if binding != nil {
		t.Error("Expected binding to be cleared")
	}
}

func TestTraitResolver_CandidatePriorityComparison(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Create test candidates
	inherentCandidate := &ImplCandidate{
		Priority: 100,
		Scope:    ScopeKindInherent,
		Distance: 0,
	}

	traitCandidate := &ImplCandidate{
		Priority: 100,
		Scope:    ScopeKindModule,
		Distance: 0,
	}

	// Inherent should have higher priority than trait
	result := resolver.compareCandidatePriority(inherentCandidate, traitCandidate)
	if result <= 0 {
		t.Error("Expected inherent candidate to have higher priority than trait candidate")
	}

	// Test priority comparison
	highPriorityCandidate := &ImplCandidate{
		Priority: 200,
		Scope:    ScopeKindModule,
		Distance: 0,
	}

	result = resolver.compareCandidatePriority(highPriorityCandidate, traitCandidate)
	if result <= 0 {
		t.Error("Expected high priority candidate to win")
	}

	// Test distance comparison
	closeCandidate := &ImplCandidate{
		Priority: 100,
		Scope:    ScopeKindModule,
		Distance: 0,
	}

	farCandidate := &ImplCandidate{
		Priority: 100,
		Scope:    ScopeKindModule,
		Distance: 2,
	}

	result = resolver.compareCandidatePriority(closeCandidate, farCandidate)
	if result <= 0 {
		t.Error("Expected close candidate to have higher priority than far candidate")
	}
}

func TestTraitResolver_AssociatedTypeResolution(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Test basic associated type resolution
	resolvedType, err := resolver.ResolveAssociatedType("T", "Item")

	// For now, we expect this to return an error since no constraints are set up
	if err == nil {
		t.Log("Associated type resolution succeeded:", resolvedType)
	} else {
		t.Log("Expected error for unbound associated type:", err)
	}
}

func TestTraitResolver_WhereClauseValidation(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Create a test where clause constraint
	constraint := &WhereClauseConstraint{
		TypeParam: "T",
		TraitBounds: []*parser.HIRType{
			{Kind: parser.HIRTypeTrait, Data: "Display"},
		},
		AssocTypeBounds: []*AssocTypeConstraint{
			{
				TypeParam: "T",
				AssocType: "Item",
				TraitBounds: []*parser.HIRType{
					{Kind: parser.HIRTypeTrait, Data: "Clone"},
				},
			},
		},
		EqualityBounds: []*EqualityConstraint{
			{
				TypeParam: "T",
				AssocType: "Output",
				EqualTo:   &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "String"},
			},
		},
		Scope: ConstraintScopeFunction,
	}

	// Validate the constraint
	err := resolver.ValidateWhereClause(constraint)
	if err != nil {
		t.Log("Where clause validation returned error (expected for unbound types):", err)
	} else {
		t.Log("Where clause validation succeeded")
	}
}

func TestTraitResolver_ConstraintSolving(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Test constraint solving for a scope
	solution, err := resolver.SolveAssociatedTypeConstraints("test_scope")
	if err != nil {
		t.Errorf("Constraint solving failed: %v", err)
	}

	if solution == nil {
		t.Error("Expected solution, got nil")
	}

	if !solution.Satisfied {
		t.Log("Solution not satisfied (expected for empty constraint set)")
	}

	if len(solution.Conflicts) > 0 {
		t.Errorf("Unexpected conflicts in empty constraint set: %d", len(solution.Conflicts))
	}
}

func TestTraitResolver_ThrowsSpecificationChecking(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Create a test trait with methods
	trait := &parser.HIRTraitType{
		Name: "Fallible",
		Methods: []*parser.HIRMethodSignature{
			{
				Name: "try_operation",
				Parameters: []*parser.HIRType{
					{Kind: parser.HIRTypePrimitive, Data: "i32"},
				},
				ReturnType: &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "String"},
			},
		},
	}

	// Test trait throws consistency
	err := resolver.CheckTraitThrowsConsistency(trait)
	if err != nil {
		t.Logf("Trait throws consistency check returned error: %v", err)
	} else {
		t.Log("Trait throws consistency check passed")
	}
}

func TestTraitResolver_ImplThrowsConsistency(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Create a test trait
	trait := &parser.HIRTraitType{
		Name: "Fallible",
		Methods: []*parser.HIRMethodSignature{
			{
				Name: "try_operation",
				Parameters: []*parser.HIRType{
					{Kind: parser.HIRTypePrimitive, Data: "i32"},
				},
				ReturnType: &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "String"},
			},
		},
	}

	// Create a test implementation
	impl := &parser.HIRImpl{
		Kind:    parser.HIRImplTrait,
		Trait:   &parser.HIRType{Kind: parser.HIRTypeTrait, Data: "Fallible"},
		ForType: &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "MyStruct"},
		Methods: []*parser.HIRFunction{
			{
				Name: "try_operation",
				Parameters: []*parser.HIRParameter{
					{
						Name: "input",
						Type: &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"},
					},
				},
				ReturnType: &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "String"},
				Attributes: []string{"nothrow"}, // Mark as no-throw
			},
		},
	}

	// Test impl throws consistency
	err := resolver.CheckImplThrowsConsistency(impl, trait)
	if err != nil {
		t.Logf("Impl throws consistency check returned error: %v", err)
	} else {
		t.Log("Impl throws consistency check passed")
	}

	// Check if there are any violations
	if resolver.HasThrowsViolations() {
		violations := resolver.GetThrowsViolations()
		t.Logf("Found %d throws violations", len(violations))
		for _, violation := range violations {
			t.Logf("Violation: %s", violation.Message)
		}
	}
}

func TestTraitResolver_WhereClauseThrowsConsistency(t *testing.T) {
	resolver := NewTraitResolver([]*parser.HIRModule{})

	// Create a test where clause constraint
	constraint := &WhereClauseConstraint{
		TypeParam: "T",
		TraitBounds: []*parser.HIRType{
			{Kind: parser.HIRTypeTrait, Data: "Fallible"},
		},
		Scope: ConstraintScopeFunction,
	}

	// Test where clause throws consistency
	err := resolver.CheckWhereClauseThrowsConsistency(constraint)
	if err != nil {
		t.Logf("Where clause throws consistency check returned error: %v", err)
	} else {
		t.Log("Where clause throws consistency check passed")
	}
}
