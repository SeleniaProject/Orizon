// Package types provides comprehensive tests for the effect type system.
// This module tests effect tracking, inference, masking, and composition functionality.
package types

import (
	"testing"
)

// TestEffectKind tests effect kind functionality.
func TestEffectKind(t *testing.T) {
	// Test effect kind string representation.
	tests := []struct {
		kind     EffectKind
		expected string
	}{
		{EffectPure, "Pure"},
		{EffectIO, "IO"},
		{EffectFileRead, "FileRead"},
		{EffectFileWrite, "FileWrite"},
		{EffectMemoryRead, "MemoryRead"},
		{EffectMemoryWrite, "MemoryWrite"},
		{EffectThrow, "Throw"},
		{EffectCatch, "Catch"},
	}

	for _, test := range tests {
		result := test.kind.String()
		if result != test.expected {
			t.Errorf("EffectKind.String() = %s, expected %s", result, test.expected)
		}
	}
}

// TestSideEffect tests side effect functionality.
func TestSideEffect(t *testing.T) {
	// Test side effect creation.
	effect := NewSideEffect(EffectIO, EffectLevelMedium)
	if effect.Kind != EffectIO {
		t.Errorf("NewSideEffect kind = %v, expected %v", effect.Kind, EffectIO)
	}

	if effect.Level != EffectLevelMedium {
		t.Errorf("NewSideEffect level = %v, expected %v", effect.Level, EffectLevelMedium)
	}

	// Test side effect cloning.
	clone := effect.Clone()
	if clone.Kind != effect.Kind {
		t.Errorf("Clone kind = %v, expected %v", clone.Kind, effect.Kind)
	}

	if clone.Level != effect.Level {
		t.Errorf("Clone level = %v, expected %v", clone.Level, effect.Level)
	}

	// Ensure independence.
	clone.Description = "test"

	if effect.Description == "test" {
		t.Error("Clone should be independent of original")
	}
}

// TestEffectSet tests effect set functionality.
func TestEffectSet(t *testing.T) {
	set := NewEffectSet()

	// Test empty set.
	if !set.IsEmpty() {
		t.Error("New EffectSet should be empty")
	}

	if set.Size() != 0 {
		t.Errorf("Empty set size = %d, expected 0", set.Size())
	}

	// Test adding effects.
	effect1 := NewSideEffect(EffectIO, EffectLevelLow)
	effect2 := NewSideEffect(EffectMemoryWrite, EffectLevelMedium)

	set.Add(effect1)

	if set.IsEmpty() {
		t.Error("Set should not be empty after adding effect")
	}

	if set.Size() != 1 {
		t.Errorf("Set size = %d, expected 1", set.Size())
	}

	set.Add(effect2)

	if set.Size() != 2 {
		t.Errorf("Set size = %d, expected 2", set.Size())
	}

	// Test contains.
	if !set.Contains(EffectIO) {
		t.Error("Set should contain EffectIO")
	}

	if !set.Contains(EffectMemoryWrite) {
		t.Error("Set should contain EffectMemoryWrite")
	}

	if set.Contains(EffectThrow) {
		t.Error("Set should not contain EffectThrow")
	}

	// Test get.
	retrieved, exists := set.Get(EffectIO)
	if !exists {
		t.Error("Should be able to get existing effect")
	}

	if retrieved.Kind != EffectIO {
		t.Errorf("Retrieved effect kind = %v, expected %v", retrieved.Kind, EffectIO)
	}

	// Test remove.
	set.Remove(EffectIO)

	if set.Contains(EffectIO) {
		t.Error("Set should not contain removed effect")
	}

	if set.Size() != 1 {
		t.Errorf("Set size = %d, expected 1 after removal", set.Size())
	}
}

// TestEffectSetOperations tests effect set operations.
func TestEffectSetOperations(t *testing.T) {
	set1 := NewEffectSet()
	set2 := NewEffectSet()

	effect1 := NewSideEffect(EffectIO, EffectLevelLow)
	effect2 := NewSideEffect(EffectMemoryWrite, EffectLevelMedium)
	effect3 := NewSideEffect(EffectThrow, EffectLevelHigh)

	set1.Add(effect1)
	set1.Add(effect2)

	set2.Add(effect2)
	set2.Add(effect3)

	// Test union.
	union := set1.Union(set2)
	if union.Size() != 3 {
		t.Errorf("Union size = %d, expected 3", union.Size())
	}

	if !union.Contains(EffectIO) || !union.Contains(EffectMemoryWrite) || !union.Contains(EffectThrow) {
		t.Error("Union should contain all effects from both sets")
	}

	// Test intersection.
	intersection := set1.Intersection(set2)
	if intersection.Size() != 1 {
		t.Errorf("Intersection size = %d, expected 1", intersection.Size())
	}

	if !intersection.Contains(EffectMemoryWrite) {
		t.Error("Intersection should contain common effect")
	}
}

// TestEffectSignature tests effect signature functionality.
func TestEffectSignature(t *testing.T) {
	signature := NewEffectSignature()

	// Test new signature is pure.
	if !signature.IsPure() {
		t.Error("New signature should be pure")
	}

	// Test adding effects.
	effect := NewSideEffect(EffectIO, EffectLevelMedium)
	signature.Effects.Add(effect)

	if signature.IsPure() {
		t.Error("Signature with effects should not be pure")
	}

	// Test signature cloning.
	clone := signature.Clone()
	if clone.Effects.Size() != signature.Effects.Size() {
		t.Error("Clone should have same number of effects")
	}

	// Test independence.
	newEffect := NewSideEffect(EffectThrow, EffectLevelHigh)
	clone.Effects.Add(newEffect)

	if signature.Effects.Size() == clone.Effects.Size() {
		t.Error("Clone should be independent of original")
	}
}

// TestEffectSignatureMerge tests effect signature merging.
func TestEffectSignatureMerge(t *testing.T) {
	sig1 := NewEffectSignature()
	sig2 := NewEffectSignature()

	effect1 := NewSideEffect(EffectIO, EffectLevelLow)
	effect2 := NewSideEffect(EffectMemoryWrite, EffectLevelMedium)

	sig1.Effects.Add(effect1)
	sig2.Effects.Add(effect2)

	merged := sig1.Merge(sig2)

	if merged.Effects.Size() != 2 {
		t.Errorf("Merged signature effects size = %d, expected 2", merged.Effects.Size())
	}

	if !merged.Effects.Contains(EffectIO) || !merged.Effects.Contains(EffectMemoryWrite) {
		t.Error("Merged signature should contain effects from both signatures")
	}
}

// TestEffectConstraints tests effect constraint functionality.
func TestEffectConstraints(t *testing.T) {
	signature := NewEffectSignature()

	// Test NoEffectConstraint.
	noIOConstraint := &NoEffectConstraint{
		Kinds: []EffectKind{EffectIO},
	}

	// Should pass for signature without IO.
	if err := noIOConstraint.Check(signature); err != nil {
		t.Errorf("NoEffectConstraint should pass for signature without IO: %v", err)
	}

	// Should fail for signature with IO.
	ioEffect := NewSideEffect(EffectIO, EffectLevelMedium)
	signature.Effects.Add(ioEffect)

	if err := noIOConstraint.Check(signature); err == nil {
		t.Error("NoEffectConstraint should fail for signature with IO")
	}

	// Test RequiredEffectConstraint.
	requiredIOConstraint := &RequiredEffectConstraint{
		Kinds: []EffectKind{EffectIO},
	}

	// Should pass for signature with IO.
	if err := requiredIOConstraint.Check(signature); err != nil {
		t.Errorf("RequiredEffectConstraint should pass for signature with IO: %v", err)
	}

	// Should fail for signature without required effect.
	signature.Effects.Remove(EffectIO)

	if err := requiredIOConstraint.Check(signature); err == nil {
		t.Error("RequiredEffectConstraint should fail for signature without required effect")
	}
}

// TestEffectScope tests effect scope functionality.
func TestEffectScope(t *testing.T) {
	parent := NewEffectScope("parent", nil)
	child := NewEffectScope("child", parent)

	// Test parent-child relationship.
	if child.Parent != parent {
		t.Error("Child scope should have correct parent")
	}

	if len(parent.Children) != 1 || parent.Children[0] != child {
		t.Error("Parent scope should have child in children list")
	}

	// Test adding effects.
	effect := NewSideEffect(EffectIO, EffectLevelMedium)
	child.AddEffect(effect)

	if !child.Effects.Contains(EffectIO) {
		t.Error("Scope should contain added effect")
	}

	// Test masking.
	child.MaskEffect(EffectIO)
	effective := child.GetEffectiveEffects()

	if effective.Contains(EffectIO) {
		t.Error("Masked effect should not appear in effective effects")
	}
}

// TestEffectMask tests effect masking functionality.
func TestEffectMask(t *testing.T) {
	effects := NewEffectSet()
	effect1 := NewSideEffect(EffectIO, EffectLevelMedium)
	effect2 := NewSideEffect(EffectMemoryWrite, EffectLevelLow)

	effects.Add(effect1)
	effects.Add(effect2)

	// Test masking IO effects.
	mask := NewEffectMask([]EffectKind{EffectIO})
	masked := mask.Apply(effects)

	if masked.Contains(EffectIO) {
		t.Error("Masked effects should not contain masked effect kind")
	}

	if !masked.Contains(EffectMemoryWrite) {
		t.Error("Masked effects should contain non-masked effect kind")
	}

	// Test inactive mask.
	mask.Active = false
	maskedInactive := mask.Apply(effects)

	if maskedInactive.Size() != effects.Size() {
		t.Error("Inactive mask should not change effects")
	}
}

// TestEffectComposer tests effect composition functionality.
func TestEffectComposer(t *testing.T) {
	composer := NewEffectComposer()

	sig1 := NewEffectSignature()
	sig2 := NewEffectSignature()

	effect1 := NewSideEffect(EffectIO, EffectLevelLow)
	effect2 := NewSideEffect(EffectMemoryWrite, EffectLevelMedium)

	sig1.Effects.Add(effect1)
	sig2.Effects.Add(effect2)

	signatures := []*EffectSignature{sig1, sig2}
	composed := composer.Compose(signatures)

	if composed.Effects.Size() != 2 {
		t.Errorf("Composed signature effects size = %d, expected 2", composed.Effects.Size())
	}

	if !composed.Effects.Contains(EffectIO) || !composed.Effects.Contains(EffectMemoryWrite) {
		t.Error("Composed signature should contain effects from all input signatures")
	}
}

// BenchmarkEffectSetOperations benchmarks effect set operations.
func BenchmarkEffectSetOperations(b *testing.B) {
	set := NewEffectSet()
	effects := make([]*SideEffect, 100)

	// Prepare effects.
	for i := 0; i < 100; i++ {
		kind := EffectKind(i%10 + 1)  // Cycle through first 10 effect kinds
		level := EffectLevel(i%5 + 1) // Cycle through effect levels
		effects[i] = NewSideEffect(kind, level)
	}

	b.ResetTimer()

	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			effect := effects[i%100]
			set.Add(effect)
		}
	})

	b.Run("Contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kind := EffectKind(i%10 + 1)
			set.Contains(kind)
		}
	})

	b.Run("Union", func(b *testing.B) {
		other := NewEffectSet()
		for i := 0; i < 50; i++ {
			other.Add(effects[i])
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			set.Union(other)
		}
	})
}

// BenchmarkEffectInference benchmarks effect inference performance.
func BenchmarkEffectInference(b *testing.B) {
	engine := NewEffectInferenceEngine(nil)

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
		_, err := engine.InferEffects(node)
		if err != nil {
			b.Fatalf("Effect inference failed: %v", err)
		}
	}
}
