package hir

import "testing"

// TestCreateFunctionType tests basic function type creation
func TestCreateFunctionType(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}

	signature := FunctionSignature{
		Name: "add",
		Parameters: []Parameter{
			{Name: "a", Type: intType},
			{Name: "b", Type: intType},
		},
		ReturnType: intType,
	}

	funcType := CreateFunctionType(signature)

	if funcType.Kind != TypeKindFunction {
		t.Errorf("Expected function kind, got %v", funcType.Kind)
	}
	if funcType.Name != "add" {
		t.Errorf("Expected name 'add', got %s", funcType.Name)
	}
	if funcType.Size != 8 { // Function pointer size
		t.Errorf("Expected size 8, got %d", funcType.Size)
	}
	if len(funcType.Parameters) != 3 { // 2 params + return type
		t.Errorf("Expected 3 parameters, got %d", len(funcType.Parameters))
	}

	// Check parameter types
	if !funcType.Parameters[0].Equals(intType) {
		t.Error("First parameter type mismatch")
	}
	if !funcType.Parameters[1].Equals(intType) {
		t.Error("Second parameter type mismatch")
	}
	if !funcType.Parameters[2].Equals(intType) { // Return type
		t.Error("Return type mismatch")
	}
}

// TestIsCompatibleSignature tests function signature compatibility
func TestIsCompatibleSignature(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	sig1 := FunctionSignature{
		Parameters: []Parameter{
			{Name: "a", Type: intType},
		},
		ReturnType: intType,
	}

	sig2 := FunctionSignature{
		Parameters: []Parameter{
			{Name: "a", Type: intType},
		},
		ReturnType: intType,
	}

	sig3 := FunctionSignature{
		Parameters: []Parameter{
			{Name: "a", Type: floatType}, // Different parameter type
		},
		ReturnType: intType,
	}

	if !IsCompatibleSignature(sig1, sig2) {
		t.Error("Identical signatures should be compatible")
	}

	// int can convert to float, so sig1 should be compatible with sig3
	if !IsCompatibleSignature(sig1, sig3) {
		t.Error("Compatible signatures should be accepted")
	}
}

// TestClosureLayout tests closure memory layout calculation
func TestClosureLayout(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	env := ClosureEnvironment{
		CapturedVars: []CapturedVariable{
			{Name: "x", Type: intType, Mode: CaptureByValue},
			{Name: "y", Type: floatType, Mode: CaptureByValue},
		},
		CaptureMode: CaptureByValue,
	}

	layout := CalculateClosureLayout(env)

	if layout.FunctionPtr != 0 {
		t.Errorf("Expected function pointer at offset 0, got %d", layout.FunctionPtr)
	}
	if layout.Environment != 8 {
		t.Errorf("Expected environment at offset 8, got %d", layout.Environment)
	}
	if len(layout.CaptureLayout) != 2 {
		t.Errorf("Expected 2 captured variables, got %d", len(layout.CaptureLayout))
	}

	// Total size should be: function pointer (8) + int (4) + float (4) + padding
	expectedSize := int64(16) // 8 + 4 + 4 aligned to 8 bytes
	if layout.TotalSize != expectedSize {
		t.Errorf("Expected total size %d, got %d", expectedSize, layout.TotalSize)
	}
}

// TestCreateClosureType tests closure type creation
func TestCreateClosureType(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}

	signature := FunctionSignature{
		Name: "lambda",
		Parameters: []Parameter{
			{Name: "a", Type: intType},
		},
		ReturnType: intType,
	}

	env := ClosureEnvironment{
		CapturedVars: []CapturedVariable{
			{Name: "captured", Type: intType, Mode: CaptureByValue},
		},
	}

	closureType := CreateClosureType(signature, env)

	if closureType.Kind != TypeKindFunction {
		t.Errorf("Expected function kind, got %v", closureType.Kind)
	}
	if closureType.Name != "closure<lambda>" {
		t.Errorf("Expected name 'closure<lambda>', got %s", closureType.Name)
	}
	if closureType.Size <= 8 { // Should be larger than a simple function pointer
		t.Errorf("Expected closure size > 8, got %d", closureType.Size)
	}
	if closureType.Properties.Copyable {
		t.Error("Closures should not be copyable by default")
	}
}

// TestPartialApplication tests partial function application
func TestPartialApplication(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}

	// Create a function that takes 3 parameters
	signature := FunctionSignature{
		Name: "add3",
		Parameters: []Parameter{
			{Name: "a", Type: intType},
			{Name: "b", Type: intType},
			{Name: "c", Type: intType},
		},
		ReturnType: intType,
	}

	// Create mock expressions for applied arguments
	arg1 := &HIRLiteral{Type: intType}
	appliedArgs := []HIRExpression{arg1}

	partialApp := CreatePartialApplication(signature, appliedArgs)

	if len(partialApp.RemainingParams) != 2 {
		t.Errorf("Expected 2 remaining parameters, got %d", len(partialApp.RemainingParams))
	}
	if partialApp.OriginalFunc.Name != "add3" {
		t.Errorf("Expected original function name 'add3', got %s", partialApp.OriginalFunc.Name)
	}
	if partialApp.ResultType.Kind != TypeKindFunction {
		t.Errorf("Expected result type to be function, got %v", partialApp.ResultType.Kind)
	}
}

// TestCanPartiallyApply tests partial application validation
func TestCanPartiallyApply(t *testing.T) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	signature := FunctionSignature{
		Parameters: []Parameter{
			{Name: "a", Type: intType},
			{Name: "b", Type: floatType},
		},
		ReturnType: intType,
	}

	// Valid partial application
	arg1 := &HIRLiteral{Type: intType}
	args1 := []HIRExpression{arg1}

	if !CanPartiallyApply(signature, args1) {
		t.Error("Should be able to partially apply with compatible argument")
	}

	// Too many arguments
	arg2 := &HIRLiteral{Type: floatType}
	arg3 := &HIRLiteral{Type: intType}
	args2 := []HIRExpression{arg1, arg2, arg3}

	if CanPartiallyApply(signature, args2) {
		t.Error("Should not be able to apply more arguments than parameters")
	}

	// Wrong argument type (assuming no conversion from float to int for this test)
	argWrong := &HIRLiteral{Type: floatType}
	argsWrong := []HIRExpression{argWrong} // float where int expected

	// This should still work because float can convert to int in our type system
	if !CanPartiallyApply(signature, argsWrong) {
		t.Error("Should be able to partially apply with convertible argument")
	}
}
