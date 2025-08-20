// Package types provides comprehensive tests for the integrated effect system.
// This module tests the integration of side effects and exception tracking.
// for comprehensive type-level safety analysis.
package types

import (
	"testing"
)

// TestIntegratedEffect tests basic integrated effect functionality.
func TestIntegratedEffect(t *testing.T) {
	// Create side effect.
	sideEffect := NewSideEffect(EffectIO, EffectLevelMedium)
	sideEffect.Description = "File I/O operation"

	// Create exception spec.
	exceptionSpec := NewExceptionSpec(ExceptionIOError, ExceptionSeverityError)
	exceptionSpec.Message = "File access failed"

	// Create integrated effect.
	integrated := NewIntegratedEffect(sideEffect, exceptionSpec)

	// Test basic properties.
	if integrated.IsEmpty() {
		t.Error("integrated effect should not be empty")
	}

	// Test severity mapping.
	severity := integrated.GetSeverity()
	if severity != EffectLevelHigh { // IOError maps to EffectLevelHigh
		t.Errorf("expected severity %v, got %v", EffectLevelHigh, severity)
	}

	// Test string representation.
	str := integrated.String()
	expected := "IO[Medium]+IOError[Error]"

	if str != expected {
		t.Errorf("expected string %s, got %s", expected, str)
	}
}

// TestIntegratedEffectSet tests integrated effect set operations.
func TestIntegratedEffectSet(t *testing.T) {
	set := NewIntegratedEffectSet()

	// Test empty set.
	if !set.IsEmpty() {
		t.Error("new set should be empty")
	}

	if set.Size() != 0 {
		t.Error("new set should have size 0")
	}

	// Add effects.
	effect1 := NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelMedium),
		NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
	)

	effect2 := NewIntegratedEffect(
		NewSideEffect(EffectMemoryAlloc, EffectLevelLow),
		nil,
	)

	effect3 := NewIntegratedEffect(
		nil,
		NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError),
	)

	set.Add(effect1)
	set.Add(effect2)
	set.Add(effect3)

	// Test set properties.
	if set.IsEmpty() {
		t.Error("set should not be empty after adding effects")
	}

	if set.Size() != 3 {
		t.Errorf("expected size 3, got %d", set.Size())
	}

	// Test contains.
	if !set.Contains(effect1) {
		t.Error("set should contain effect1")
	}

	// Test side effects extraction.
	sideEffects := set.GetSideEffects()
	if sideEffects.Size() != 2 { // effect1 and effect2 have side effects
		t.Errorf("expected 2 side effects, got %d", sideEffects.Size())
	}

	// Test exceptions extraction.
	exceptions := set.GetExceptions()
	if exceptions.Size() != 2 { // effect1 and effect3 have exceptions
		t.Errorf("expected 2 exceptions, got %d", exceptions.Size())
	}
}

// TestIntegratedEffectSignature tests integrated effect signatures.
func TestIntegratedEffectSignature(t *testing.T) {
	signature := NewIntegratedEffectSignature("testFunction")

	// Test initial state.
	if !signature.IsPure() {
		t.Error("new signature should be pure initially")
	}

	if !signature.IsNoThrow() {
		t.Error("new signature should be no-throw initially")
	}

	// Add impure effect.
	impureEffect := NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelMedium),
		nil,
	)
	signature.AddEffect(impureEffect)

	if signature.IsPure() {
		t.Error("signature should not be pure after adding I/O effect")
	}

	// Add throwing effect.
	throwingEffect := NewIntegratedEffect(
		nil,
		NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
	)
	signature.AddEffect(throwingEffect)

	if signature.IsNoThrow() {
		t.Error("signature should not be no-throw after adding exception")
	}

	// Test severity.
	maxSeverity := signature.GetMaxSeverity()
	if maxSeverity != EffectLevelHigh {
		t.Errorf("expected max severity %v, got %v", EffectLevelHigh, maxSeverity)
	}
}

// TestExceptionMask tests exception masking functionality.
func TestExceptionMask(t *testing.T) {
	mask := NewExceptionMask("testMask")

	// Create exception spec.
	spec := NewExceptionSpec(ExceptionIOError, ExceptionSeverityError)
	spec.TypeName = "IOErrorType"

	// Test initial state - should not mask.
	if mask.ShouldMask(spec) {
		t.Error("new mask should not mask anything initially")
	}

	// Mask by kind.
	mask.MaskKind(ExceptionIOError)

	if !mask.ShouldMask(spec) {
		t.Error("mask should mask IOError after masking kind")
	}

	// Reset and mask by severity.
	mask = NewExceptionMask("testMask2")
	mask.MaskSeverity(ExceptionSeverityError)

	if !mask.ShouldMask(spec) {
		t.Error("mask should mask Error severity after masking severity")
	}

	// Reset and test allowed types.
	mask = NewExceptionMask("testMask3")
	mask.AllowType("AllowedType")

	if !mask.ShouldMask(spec) { // spec has IOErrorType, not AllowedType
		t.Error("mask should mask unallowed type when specific types are allowed")
	}

	// Deactivate mask.
	mask.Active = false
	if mask.ShouldMask(spec) {
		t.Error("inactive mask should not mask anything")
	}
}

// TestIntegratedEffectMask tests integrated effect masking.
func TestIntegratedEffectMask(t *testing.T) {
	mask := NewIntegratedEffectMask("testMask")

	// Create integrated effect.
	effect := NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelMedium),
		NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
	)

	// Test initial state - should not mask.
	if mask.MaskEffect(effect) {
		t.Error("new mask should not mask effect initially")
	}

	// Configure masks.
	mask.SideEffectMask.MaskedKinds = []EffectKind{
		EffectIO,
	}
	mask.ExceptionMask.MaskKind(ExceptionIOError)

	// Test masking.
	if !mask.MaskEffect(effect) {
		t.Error("mask should mask effect when both components are masked")
	}

	// Test partial masking.
	mask.ExceptionMask = NewExceptionMask("empty")
	if mask.MaskEffect(effect) {
		t.Error("mask should not mask effect when only side effect is masked")
	}
}

// TestIntegratedEffectAnalyzer tests the integrated analyzer.
func TestIntegratedEffectAnalyzer(t *testing.T) {
	analyzer := NewIntegratedEffectAnalyzer()

	// Test initial state.
	if analyzer.String() != "IntegratedEffectAnalyzer: 0 signatures, 0 masks" {
		t.Error("new analyzer should have 0 signatures and 0 masks")
	}

	// Test mask management.
	mask := NewIntegratedEffectMask("testMask")
	analyzer.AddMask(mask)

	if len(analyzer.IntegratedMasks) != 1 {
		t.Error("analyzer should have 1 mask after adding")
	}

	analyzer.RemoveMask("testMask")

	if len(analyzer.IntegratedMasks) != 0 {
		t.Error("analyzer should have 0 masks after removing")
	}
}

// TestCompatibilityChecking tests compatibility between signatures.
func TestCompatibilityChecking(t *testing.T) {
	analyzer := NewIntegratedEffectAnalyzer()

	// Create pure function signature.
	pureFunc := NewIntegratedEffectSignature("pureFunction")
	pureFunc.Purity = true

	// Create impure function signature.
	impureFunc := NewIntegratedEffectSignature("impureFunction")
	impureFunc.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelMedium),
		nil,
	))

	// Create no-throw function signature.
	noThrowFunc := NewIntegratedEffectSignature("noThrowFunction")
	noThrowFunc.Safety = SafetyNoThrow

	// Create throwing function signature.
	throwingFunc := NewIntegratedEffectSignature("throwingFunction")
	throwingFunc.AddEffect(NewIntegratedEffect(
		nil,
		NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
	))

	// Test purity compatibility.
	issues := analyzer.CheckCompatibility(pureFunc, impureFunc)
	if len(issues) == 0 {
		t.Error("pure function should not be compatible with impure function")
	}

	// Test exception compatibility.
	issues = analyzer.CheckCompatibility(noThrowFunc, throwingFunc)
	if len(issues) == 0 {
		t.Error("no-throw function should not be compatible with throwing function")
	}

	// Test compatible case.
	issues = analyzer.CheckCompatibility(impureFunc, pureFunc)
	if len(issues) != 0 {
		t.Error("impure function should be compatible with pure function")
	}
}

// TestSeverityMapping tests exception severity to effect level mapping.
func TestSeverityMapping(t *testing.T) {
	tests := []struct {
		severity ExceptionSeverity
		expected EffectLevel
	}{
		{ExceptionSeverityInfo, EffectLevelLow},
		{ExceptionSeverityWarning, EffectLevelMedium},
		{ExceptionSeverityError, EffectLevelHigh},
		{ExceptionSeverityCritical, EffectLevelCritical},
		{ExceptionSeverityFatal, EffectLevelCritical},
	}

	for _, test := range tests {
		actual := mapExceptionSeverityToEffectLevel(test.severity)
		if actual != test.expected {
			t.Errorf("severity %v should map to %v, got %v",
				test.severity, test.expected, actual)
		}
	}
}

// TestValidation tests validation functions.
func TestValidation(t *testing.T) {
	// Test valid integrated effect.
	validEffect := NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelMedium),
		NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
	)

	if err := ValidateIntegratedEffect(validEffect); err != nil {
		t.Errorf("valid effect should pass validation: %v", err)
	}

	// Test nil integrated effect.
	if err := ValidateIntegratedEffect(nil); err == nil {
		t.Error("nil effect should fail validation")
	}

	// Test empty integrated effect.
	emptyEffect := NewIntegratedEffect(nil, nil)
	if err := ValidateIntegratedEffect(emptyEffect); err == nil {
		t.Error("empty effect should fail validation")
	}

	// Test invalid side effect.
	invalidSideEffect := &SideEffect{
		Kind:  EffectKind(999), // Invalid kind
		Level: EffectLevelMedium,
	}

	invalidIntegrated := NewIntegratedEffect(invalidSideEffect, nil)
	if err := ValidateIntegratedEffect(invalidIntegrated); err == nil {
		t.Error("effect with invalid side effect should fail validation")
	}

	// Test invalid exception spec.
	invalidExceptionSpec := &ExceptionSpec{
		Kind:     ExceptionKind(999), // Invalid kind
		Severity: ExceptionSeverityError,
	}

	invalidIntegrated2 := NewIntegratedEffect(nil, invalidExceptionSpec)
	if err := ValidateIntegratedEffect(invalidIntegrated2); err == nil {
		t.Error("effect with invalid exception spec should fail validation")
	}
}

// TestRealWorldScenarios tests realistic usage patterns.
func TestRealWorldScenarios(t *testing.T) {
	analyzer := NewIntegratedEffectAnalyzer()

	// Scenario 1: File processing function.
	fileProcessor := NewIntegratedEffectSignature("processFile")
	fileProcessor.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectFileRead, EffectLevelMedium),
		NewExceptionSpec(ExceptionFileNotFound, ExceptionSeverityError),
	))
	fileProcessor.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectFileWrite, EffectLevelMedium),
		NewExceptionSpec(ExceptionPermissionDenied, ExceptionSeverityError),
	))

	// Scenario 2: Network client function.
	networkClient := NewIntegratedEffectSignature("networkRequest")
	networkClient.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectNetworkRead, EffectLevelHigh),
		NewExceptionSpec(ExceptionNetworkTimeout, ExceptionSeverityWarning),
	))
	networkClient.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectNetworkWrite, EffectLevelHigh),
		NewExceptionSpec(ExceptionConnectionFailed, ExceptionSeverityError),
	))

	// Scenario 3: Database transaction.
	dbTransaction := NewIntegratedEffectSignature("dbTransaction")
	dbTransaction.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelHigh),
		NewExceptionSpec(ExceptionDeadlock, ExceptionSeverityError),
	))

	dbTransaction.Safety = SafetyStrong

	// Test signatures.
	if fileProcessor.IsPure() {
		t.Error("file processor should not be pure")
	}

	if networkClient.IsNoThrow() {
		t.Error("network client should be throwing")
	}

	if dbTransaction.GetMaxSeverity() != EffectLevelHigh {
		t.Error("db transaction should have high severity")
	}

	// Test compatibility.
	issues := analyzer.CheckCompatibility(dbTransaction, networkClient)
	if len(issues) != 0 {
		t.Error("db transaction should be compatible with network client")
	}

	// Store signatures.
	analyzer.IntegratedSignatures["processFile"] = fileProcessor
	analyzer.IntegratedSignatures["networkRequest"] = networkClient
	analyzer.IntegratedSignatures["dbTransaction"] = dbTransaction

	// Test retrieval.
	if sig, exists := analyzer.GetSignature("processFile"); !exists || sig != fileProcessor {
		t.Error("should be able to retrieve stored signature")
	}
}

// TestEffectMasking tests effect masking in realistic scenarios.
func TestEffectMasking(t *testing.T) {
	analyzer := NewIntegratedEffectAnalyzer()

	// Create function with multiple effects.
	complexFunc := NewIntegratedEffectSignature("complexFunction")
	complexFunc.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectIO, EffectLevelMedium),
		NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
	))
	complexFunc.AddEffect(NewIntegratedEffect(
		NewSideEffect(EffectMemoryAlloc, EffectLevelLow),
		nil,
	))
	complexFunc.AddEffect(NewIntegratedEffect(
		nil,
		NewExceptionSpec(ExceptionNullPointer, ExceptionSeverityError),
	))

	// Create mask to hide I/O operations
	ioMask := NewIntegratedEffectMask("hiddenIO")
	ioMask.SideEffectMask.MaskedKinds = []EffectKind{
		EffectIO,
	}
	ioMask.ExceptionMask.MaskKind(ExceptionIOError)

	analyzer.AddMask(ioMask)

	// Apply masking.
	maskedFunc := analyzer.ApplyMasks(complexFunc)

	// Check that I/O effects are masked
	if maskedFunc.Effects.Size() >= complexFunc.Effects.Size() {
		t.Error("masked function should have fewer effects than original")
	}

	// Verify specific effects are masked.
	hasIOEffect := false

	for _, effect := range maskedFunc.Effects.effects {
		if effect.SideEffect != nil && effect.SideEffect.Kind == EffectIO {
			hasIOEffect = true

			break
		}
	}

	if hasIOEffect {
		t.Error("I/O effects should be masked")
	}
}

// BenchmarkIntegratedEffectOperations benchmarks integrated effect operations.
func BenchmarkIntegratedEffectOperations(b *testing.B) {
	// Benchmark effect creation.
	b.Run("CreateIntegratedEffect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewIntegratedEffect(
				NewSideEffect(EffectIO, EffectLevelMedium),
				NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
			)
		}
	})

	// Benchmark set operations.
	b.Run("EffectSetOperations", func(b *testing.B) {
		set := NewIntegratedEffectSet()
		effect := NewIntegratedEffect(
			NewSideEffect(EffectIO, EffectLevelMedium),
			NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
		)

		for i := 0; i < b.N; i++ {
			set.Add(effect)
			set.Contains(effect)
		}
	})

	// Benchmark signature operations.
	b.Run("SignatureOperations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sig := NewIntegratedEffectSignature("testFunc")
			sig.AddEffect(NewIntegratedEffect(
				NewSideEffect(EffectIO, EffectLevelMedium),
				NewExceptionSpec(ExceptionIOError, ExceptionSeverityError),
			))
			sig.IsPure()
			sig.IsNoThrow()
			sig.GetMaxSeverity()
		}
	})

	// Benchmark compatibility checking.
	b.Run("CompatibilityChecking", func(b *testing.B) {
		analyzer := NewIntegratedEffectAnalyzer()
		sig1 := NewIntegratedEffectSignature("func1")
		sig2 := NewIntegratedEffectSignature("func2")
		sig2.AddEffect(NewIntegratedEffect(
			NewSideEffect(EffectIO, EffectLevelMedium),
			nil,
		))

		for i := 0; i < b.N; i++ {
			analyzer.CheckCompatibility(sig1, sig2)
		}
	})
}
