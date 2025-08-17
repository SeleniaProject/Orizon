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
