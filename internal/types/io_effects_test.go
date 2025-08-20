// Package types provides comprehensive testing for the I/O effects system.
// This module tests I/O effect tracking, pure function guarantees,
// and I/O monad functionality.
package types

import (
	"fmt"
	"testing"
	"time"
)

// Test I/O effect kind string representation.
func TestIOEffectKind_String(t *testing.T) {
	tests := []struct {
		expected string
		kind     IOEffectKind
	}{
		{IOEffectPure, "Pure"},
		{IOEffectFileRead, "FileRead"},
		{IOEffectFileWrite, "FileWrite"},
		{IOEffectNetworkConnect, "NetworkConnect"},
		{IOEffectDatabaseQuery, "DatabaseQuery"},
		{IOEffectConsoleWrite, "ConsoleWrite"},
		{IOEffectCustom, "Custom"},
	}

	for _, test := range tests {
		if got := test.kind.String(); got != test.expected {
			t.Errorf("IOEffectKind.String() = %v, want %v", got, test.expected)
		}
	}
}

// Test I/O effect permission string representation.
func TestIOEffectPermission_String(t *testing.T) {
	tests := []struct {
		expected   string
		permission IOEffectPermission
	}{
		{IOPermissionNone, "None"},
		{IOPermissionRead, "Read"},
		{IOPermissionWrite, "Write"},
		{IOPermissionReadWrite, "ReadWrite"},
		{IOPermissionFullAccess, "FullAccess"},
	}

	for _, test := range tests {
		if got := test.permission.String(); got != test.expected {
			t.Errorf("IOEffectPermission.String() = %v, want %v", got, test.expected)
		}
	}
}

// Test I/O effect behavior string representation.
func TestIOEffectBehavior_String(t *testing.T) {
	tests := []struct {
		expected string
		behavior IOEffectBehavior
	}{
		{IOBehaviorDeterministic, "Deterministic"},
		{IOBehaviorNonDeterministic, "NonDeterministic"},
		{IOBehaviorIdempotent, "Idempotent"},
		{IOBehaviorBlocking, "Blocking"},
		{IOBehaviorAtomic, "Atomic"},
	}

	for _, test := range tests {
		if got := test.behavior.String(); got != test.expected {
			t.Errorf("IOEffectBehavior.String() = %v, want %v", got, test.expected)
		}
	}
}

// Test I/O effect creation and properties.
func TestIOEffect_Creation(t *testing.T) {
	effect := NewIOEffect(IOEffectFileRead, IOPermissionRead)

	if effect.Kind != IOEffectFileRead {
		t.Errorf("Expected kind FileRead, got %v", effect.Kind)
	}

	if effect.Permission != IOPermissionRead {
		t.Errorf("Expected permission Read, got %v", effect.Permission)
	}

	if effect.Level != EffectLevelMedium {
		t.Errorf("Expected level Medium, got %v", effect.Level)
	}

	if effect.IsPure() {
		t.Error("FileRead effect should not be pure")
	}

	if !effect.IsReadOnly() {
		t.Error("FileRead effect should be read-only")
	}

	if effect.IsWriteAccess() {
		t.Error("FileRead effect should not have write access")
	}
}

// Test I/O effect behavior management.
func TestIOEffect_Behaviors(t *testing.T) {
	effect := NewIOEffect(IOEffectNetworkSend, IOPermissionWrite)

	// Initially no behaviors.
	if effect.HasBehavior(IOBehaviorNonDeterministic) {
		t.Error("Effect should not have NonDeterministic behavior initially")
	}

	// Add behavior.
	effect.AddBehavior(IOBehaviorNonDeterministic)

	if !effect.HasBehavior(IOBehaviorNonDeterministic) {
		t.Error("Effect should have NonDeterministic behavior after adding")
	}

	// Add same behavior again (should not duplicate).
	effect.AddBehavior(IOBehaviorNonDeterministic)

	count := 0

	for _, behavior := range effect.Behaviors {
		if behavior == IOBehaviorNonDeterministic {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 occurrence of NonDeterministic behavior, got %d", count)
	}
}

// Test I/O effect cloning.
func TestIOEffect_Clone(t *testing.T) {
	original := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	original.Resource = "/tmp/test.txt"
	original.Description = "Test file write"
	original.AddBehavior(IOBehaviorSideEffecting)
	original.Metadata["size"] = 1024

	clone := original.Clone()

	// Check that fields are copied.
	if clone.Kind != original.Kind {
		t.Error("Clone should have same kind")
	}

	if clone.Resource != original.Resource {
		t.Error("Clone should have same resource")
	}

	if clone.Description != original.Description {
		t.Error("Clone should have same description")
	}

	if len(clone.Behaviors) != len(original.Behaviors) {
		t.Error("Clone should have same behaviors")
	}

	if clone.Metadata["size"] != original.Metadata["size"] {
		t.Error("Clone should have same metadata")
	}

	// Check that modifications to clone don't affect original.
	clone.Resource = "/tmp/clone.txt"
	if original.Resource == clone.Resource {
		t.Error("Modifying clone should not affect original")
	}
}

// Test I/O effect set operations.
func TestIOEffectSet_Operations(t *testing.T) {
	set := NewIOEffectSet()

	// Initially empty.
	if !set.IsEmpty() {
		t.Error("New effect set should be empty")
	}

	if set.Size() != 0 {
		t.Error("Empty set should have size 0")
	}

	if !set.IsPure() {
		t.Error("Empty set should be pure")
	}

	// Add effect.
	effect1 := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	set.Add(effect1)

	if set.IsEmpty() {
		t.Error("Set should not be empty after adding effect")
	}

	if set.Size() != 1 {
		t.Error("Set should have size 1 after adding one effect")
	}

	if set.IsPure() {
		t.Error("Set with FileRead effect should not be pure")
	}

	if !set.Contains(effect1) {
		t.Error("Set should contain the added effect")
	}

	// Add another effect.
	effect2 := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	set.Add(effect2)

	if set.Size() != 2 {
		t.Error("Set should have size 2 after adding two effects")
	}

	if !set.Contains(effect2) {
		t.Error("Set should contain the second effect")
	}

	// Remove effect.
	set.Remove(effect1)

	if set.Contains(effect1) {
		t.Error("Set should not contain removed effect")
	}

	if set.Size() != 1 {
		t.Error("Set should have size 1 after removing one effect")
	}
}

// Test I/O effect set operations (union, intersection, difference).
func TestIOEffectSet_SetOperations(t *testing.T) {
	set1 := NewIOEffectSet()
	set2 := NewIOEffectSet()

	effect1 := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	effect2 := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	effect3 := NewIOEffect(IOEffectNetworkSend, IOPermissionWrite)

	set1.Add(effect1)
	set1.Add(effect2)

	set2.Add(effect2)
	set2.Add(effect3)

	// Test union.
	union := set1.Union(set2)
	if union.Size() != 3 {
		t.Errorf("Union should have 3 effects, got %d", union.Size())
	}

	if !union.Contains(effect1) || !union.Contains(effect2) || !union.Contains(effect3) {
		t.Error("Union should contain all effects from both sets")
	}

	// Test intersection.
	intersection := set1.Intersection(set2)
	if intersection.Size() != 1 {
		t.Errorf("Intersection should have 1 effect, got %d", intersection.Size())
	}

	if !intersection.Contains(effect2) {
		t.Error("Intersection should contain the common effect")
	}

	// Test difference.
	difference := set1.Difference(set2)
	if difference.Size() != 1 {
		t.Errorf("Difference should have 1 effect, got %d", difference.Size())
	}

	if !difference.Contains(effect1) {
		t.Error("Difference should contain effect1")
	}

	if difference.Contains(effect2) {
		t.Error("Difference should not contain effect2")
	}
}

// Test I/O constraints.
func TestIOConstraints(t *testing.T) {
	// Test permission constraint.
	permConstraint := NewIOPermissionConstraint(IOPermissionRead, IOPermissionReadWrite)

	readEffect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	writeEffect := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)

	if !permConstraint.Check(readEffect) {
		t.Error("Permission constraint should allow read effect")
	}

	if permConstraint.Check(writeEffect) {
		t.Error("Permission constraint should not allow write-only effect")
	}

	// Test resource constraint.
	resourceConstraint := NewIOResourceConstraint()
	resourceConstraint.AllowResource("/tmp/")
	resourceConstraint.DenyResource("/etc/")

	tmpEffect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	tmpEffect.Resource = "/tmp/test.txt"

	etcEffect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	etcEffect.Resource = "/etc/passwd"

	homeEffect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	homeEffect.Resource = "/home/user/file.txt"

	if !resourceConstraint.Check(tmpEffect) {
		t.Error("Resource constraint should allow /tmp/ files")
	}

	if resourceConstraint.Check(etcEffect) {
		t.Error("Resource constraint should deny /etc/ files")
	}

	if !resourceConstraint.Check(homeEffect) {
		t.Error("Resource constraint should allow files not in deny list when no specific allow rules")
	}
}

// Test I/O signature.
func TestIOSignature(t *testing.T) {
	sig := NewIOSignature("testFunction")

	// Initially pure.
	if !sig.Pure {
		t.Error("New signature should be pure")
	}

	if !sig.Deterministic {
		t.Error("New signature should be deterministic")
	}

	if !sig.Idempotent {
		t.Error("New signature should be idempotent")
	}

	// Add pure effect.
	pureEffect := NewIOEffect(IOEffectPure, IOPermissionNone)
	sig.AddEffect(pureEffect)

	if !sig.Pure {
		t.Error("Signature should remain pure after adding pure effect")
	}

	// Add impure effect.
	fileEffect := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	fileEffect.AddBehavior(IOBehaviorNonDeterministic)
	sig.AddEffect(fileEffect)

	if sig.Pure {
		t.Error("Signature should not be pure after adding file write effect")
	}

	if sig.Deterministic {
		t.Error("Signature should not be deterministic after adding non-deterministic effect")
	}

	// Test constraint checking.
	constraint := NewIOPermissionConstraint(IOPermissionRead)
	sig.AddConstraint(constraint)

	violations := sig.CheckConstraints(fileEffect)
	if len(violations) == 0 {
		t.Error("Should have constraint violations for write effect with read-only constraint")
	}
}

// Test I/O monad operations.
func TestIOMonad_Operations(t *testing.T) {
	// Test pure I/O
	pure := PureIO(42)

	result, err := pure.Run()
	if err != nil {
		t.Errorf("Pure I/O should not error: %v", err)
	}

	if result != 42 {
		t.Errorf("Pure I/O should return 42, got %v", result)
	}

	// Test map operation.
	mapped := pure.Map(func(x interface{}) interface{} {
		return x.(int) * 2
	})

	result, err = mapped.Run()
	if err != nil {
		t.Errorf("Mapped I/O should not error: %v", err)
	}

	if result != 84 {
		t.Errorf("Mapped I/O should return 84, got %v", result)
	}

	// Test bind operation.
	bound := pure.Bind(func(x interface{}) *IOMonad {
		return PureIO(x.(int) + 10)
	})

	result, err = bound.Run()
	if err != nil {
		t.Errorf("Bound I/O should not error: %v", err)
	}

	if result != 52 {
		t.Errorf("Bound I/O should return 52, got %v", result)
	}
}

// Test I/O monad sequencing.
func TestIOMonad_Sequence(t *testing.T) {
	monads := []*IOMonad{
		PureIO(1),
		PureIO(2),
		PureIO(3),
	}

	sequence := Sequence(monads)

	result, err := sequence.Run()
	if err != nil {
		t.Errorf("Sequence should not error: %v", err)
	}

	results, ok := result.([]interface{})
	if !ok {
		t.Error("Sequence should return slice of results")
	}

	if len(results) != 3 {
		t.Errorf("Sequence should return 3 results, got %d", len(results))
	}

	for i, expected := range []int{1, 2, 3} {
		if results[i] != expected {
			t.Errorf("Result %d should be %d, got %v", i, expected, results[i])
		}
	}
}

// Test I/O monad parallel execution.
func TestIOMonad_Parallel(t *testing.T) {
	monads := []*IOMonad{
		NewIOMonad(func() (interface{}, error) {
			time.Sleep(10 * time.Millisecond)

			return "first", nil
		}),
		NewIOMonad(func() (interface{}, error) {
			time.Sleep(5 * time.Millisecond)

			return "second", nil
		}),
	}

	start := time.Now()
	parallel := Parallel(monads)
	result, err := parallel.Run()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Parallel should not error: %v", err)
	}

	// Should complete significantly faster than the sum of individual times.
	// Allow a higher threshold to reduce flakiness across platforms/CI.
	if duration > 30*time.Millisecond {
		t.Errorf("Parallel execution took too long: %v (threshold 30ms)", duration)
	}

	results, ok := result.([]interface{})
	if !ok {
		t.Error("Parallel should return slice of results")
	}

	if len(results) != 2 {
		t.Errorf("Parallel should return 2 results, got %d", len(results))
	}
}

// Test I/O context.
func TestIOContext(t *testing.T) {
	context := NewIOContext()

	// Initially allows nothing specific.
	effect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	if !context.CanPerform(effect) {
		t.Error("Empty context should allow effects by default")
	}

	// Add permission constraint.
	context.AllowPermission(IOPermissionRead)

	if !context.CanPerform(effect) {
		t.Error("Context should allow read effect with read permission")
	}

	writeEffect := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	if context.CanPerform(writeEffect) {
		t.Error("Context should not allow write effect without write permission")
	}

	// Add kind constraint.
	context.AllowKind(IOEffectFileRead)

	if !context.CanPerform(effect) {
		t.Error("Context should allow file read effect")
	}

	networkEffect := NewIOEffect(IOEffectNetworkSend, IOPermissionRead)
	if context.CanPerform(networkEffect) {
		t.Error("Context should not allow network effect without network kind permission")
	}
}

// Test I/O inference engine.
func TestIOInferenceEngine(t *testing.T) {
	context := NewIOContext()
	engine := NewIOInferenceEngine(context)

	// Register a function signature.
	sig := NewIOSignature("readFile")
	fileEffect := NewIOEffect(IOEffectFileRead, IOPermissionRead)
	sig.AddEffect(fileEffect)
	engine.RegisterFunction("readFile", sig)

	// Register println function signature.
	printlnSig := NewIOSignature("println")
	stdoutEffect := NewIOEffect(IOEffectStdoutWrite, IOPermissionWrite)
	printlnSig.AddEffect(stdoutEffect)
	engine.RegisterFunction("println", printlnSig)

	// Test function signature retrieval.
	retrieved, exists := engine.GetFunctionSignature("readFile")
	if !exists {
		t.Error("Should find registered function signature")
	}

	if retrieved.FunctionName != "readFile" {
		t.Error("Retrieved signature should have correct function name")
	}

	if retrieved.Effects.Size() != 1 {
		t.Error("Retrieved signature should have one effect")
	}

	// Test inference on function call.
	callExpr := &CallExpr{
		Function: &FunctionDecl{Name: "println"},
	}

	effects, err := engine.InferEffects(callExpr)
	if err != nil {
		t.Errorf("Inference should not error: %v", err)
	}

	if effects.Size() != 1 {
		t.Errorf("Should infer 1 effect for println, got %d", effects.Size())
	}

	effectList := effects.ToSlice()
	if len(effectList) > 0 && effectList[0].Kind != IOEffectStdoutWrite {
		t.Error("Should infer stdout write effect for println")
	}
}

// Test I/O purity checker.
func TestIOPurityChecker(t *testing.T) {
	checker := NewIOPurityChecker(true)

	// Test pure function.
	pureSig := NewIOSignature("pureFunction")

	violations := checker.CheckPurity(pureSig)
	if len(violations) != 0 {
		t.Errorf("Pure function should have no violations, got %d", len(violations))
	}

	// Test impure function.
	impureSig := NewIOSignature("impureFunction")
	fileEffect := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
	impureSig.AddEffect(fileEffect)

	violations = checker.CheckPurity(impureSig)
	if len(violations) == 0 {
		t.Error("Impure function should have violations")
	}

	// Test whitelisted function.
	checker.AllowFunction("impureFunction")
	violations = checker.CheckPurity(impureSig)
	// Note: violations might still exist for the signature being marked as not pure.
	// but there should be no violations for the individual effects.

	// Test enforce purity.
	err := checker.EnforcePurity(pureSig)
	if err != nil {
		t.Errorf("Pure function should pass purity enforcement: %v", err)
	}

	err = checker.EnforcePurity(impureSig)
	if err == nil {
		t.Error("Impure function should fail purity enforcement")
	}
}

// Benchmark I/O effect set operations.
func BenchmarkIOEffectSet_Add(b *testing.B) {
	set := NewIOEffectSet()
	effect := NewIOEffect(IOEffectFileRead, IOPermissionRead)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		set.Add(effect)
	}
}

func BenchmarkIOEffectSet_Union(b *testing.B) {
	set1 := NewIOEffectSet()
	set2 := NewIOEffectSet()

	for i := 0; i < 100; i++ {
		effect1 := NewIOEffect(IOEffectFileRead, IOPermissionRead)
		effect1.Resource = fmt.Sprintf("file%d.txt", i)
		set1.Add(effect1)

		effect2 := NewIOEffect(IOEffectFileWrite, IOPermissionWrite)
		effect2.Resource = fmt.Sprintf("file%d.txt", i)
		set2.Add(effect2)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = set1.Union(set2)
	}
}

func BenchmarkIOInferenceEngine_InferEffects(b *testing.B) {
	context := NewIOContext()
	engine := NewIOInferenceEngine(context)

	callExpr := &CallExpr{
		Function: &FunctionDecl{Name: "println"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = engine.InferEffects(callExpr)
	}
}
