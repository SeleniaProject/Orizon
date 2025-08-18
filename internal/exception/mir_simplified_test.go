package exception

import (
	"strings"
	"testing"
)

// TestSimplifiedMIRIntegrationCreation tests creating the simplified MIR integration
func TestSimplifiedMIRIntegrationCreation(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	if integration == nil {
		t.Fatal("Expected non-nil simplified MIR exception integration")
	}

	if integration.codeGen == nil {
		t.Error("Expected non-nil code generator")
	}

	if integration.exceptionInfos == nil {
		t.Error("Expected non-nil exception infos map")
	}

	if integration.labelCounter != 0 {
		t.Error("Expected label counter to start at 0")
	}
}

// TestSimplifiedLabelGeneration tests unique label generation
func TestSimplifiedLabelGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	label1 := integration.nextLabel("test")
	label2 := integration.nextLabel("test")
	label3 := integration.nextLabel("other")

	expected := []string{"test_1", "test_2", "other_3"}
	actual := []string{label1, label2, label3}

	for i, exp := range expected {
		if actual[i] != exp {
			t.Errorf("Expected label %s, got %s", exp, actual[i])
		}
	}
}

// TestSimplifiedBoundsCheckMIRGeneration tests bounds check MIR generation
func TestSimplifiedBoundsCheckMIRGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	mirCode := integration.GenerateBoundsCheckMIR("index_var", "length_var", "myArray")

	// Check for basic MIR structure
	if !strings.Contains(mirCode, "Bounds check") {
		t.Error("Expected bounds check comment")
	}
	if !strings.Contains(mirCode, "cmp.slt") {
		t.Error("Expected signed less than comparison")
	}
	if !strings.Contains(mirCode, "cmp.sge") {
		t.Error("Expected signed greater equal comparison")
	}
	if !strings.Contains(mirCode, "brcond") {
		t.Error("Expected conditional branch")
	}
	if !strings.Contains(mirCode, "panic_bounds_check") {
		t.Error("Expected bounds check panic call")
	}
	if !strings.Contains(mirCode, "index_var") {
		t.Error("Expected index variable in generated code")
	}
	if !strings.Contains(mirCode, "length_var") {
		t.Error("Expected length variable in generated code")
	}
	if !strings.Contains(mirCode, "myArray") {
		t.Error("Expected array name in generated code")
	}

	// Check exception info was recorded
	if len(integration.exceptionInfos) == 0 {
		t.Error("Expected exception info to be recorded")
	}
}

// TestSimplifiedNullCheckMIRGeneration tests null check MIR generation
func TestSimplifiedNullCheckMIRGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	mirCode := integration.GenerateNullCheckMIR("ptr_var", "myPointer")

	// Check for null check components
	if !strings.Contains(mirCode, "Null pointer check") {
		t.Error("Expected null pointer check comment")
	}
	if !strings.Contains(mirCode, "cmp.eq") {
		t.Error("Expected equality comparison")
	}
	if !strings.Contains(mirCode, "brcond") {
		t.Error("Expected conditional branch")
	}
	if !strings.Contains(mirCode, "panic_null_pointer") {
		t.Error("Expected null pointer panic call")
	}
	if !strings.Contains(mirCode, "ptr_var") {
		t.Error("Expected pointer variable in generated code")
	}
	if !strings.Contains(mirCode, "myPointer") {
		t.Error("Expected pointer name in generated code")
	}

	// Check exception info
	found := false
	for _, info := range integration.exceptionInfos {
		if info.Kind == ExceptionNullPointer {
			found = true
			if !strings.Contains(info.Message, "myPointer") {
				t.Error("Expected pointer name in exception message")
			}
		}
	}

	if !found {
		t.Error("Expected null pointer exception info")
	}
}

// TestSimplifiedDivisionCheckMIRGeneration tests division check MIR generation
func TestSimplifiedDivisionCheckMIRGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	mirCode := integration.GenerateDivisionCheckMIR("divisor_var", "test_division")

	// Check for division check components
	if !strings.Contains(mirCode, "Division by zero check") {
		t.Error("Expected division by zero check comment")
	}
	if !strings.Contains(mirCode, "cmp.eq") {
		t.Error("Expected equality comparison")
	}
	if !strings.Contains(mirCode, "panic_division_by_zero") {
		t.Error("Expected division by zero panic call")
	}
	if !strings.Contains(mirCode, "divisor_var") {
		t.Error("Expected divisor variable in generated code")
	}
	if !strings.Contains(mirCode, "test_division") {
		t.Error("Expected operation name in generated code")
	}

	// Check exception info
	found := false
	for _, info := range integration.exceptionInfos {
		if info.Kind == ExceptionDivisionByZero {
			found = true
			if !strings.Contains(info.Message, "test_division") {
				t.Error("Expected operation name in exception message")
			}
		}
	}

	if !found {
		t.Error("Expected division by zero exception info")
	}
}

// TestSimplifiedAssertMIRGeneration tests assertion MIR generation
func TestSimplifiedAssertMIRGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	mirCode := integration.GenerateAssertMIR("condition_var", "Test assertion failed")

	// Check for assertion components
	if !strings.Contains(mirCode, "Assertion check") {
		t.Error("Expected assertion check comment")
	}
	if !strings.Contains(mirCode, "cmp.eq") {
		t.Error("Expected equality comparison")
	}
	if !strings.Contains(mirCode, "panic_assert") {
		t.Error("Expected assertion panic call")
	}
	if !strings.Contains(mirCode, "condition_var") {
		t.Error("Expected condition variable in generated code")
	}
	if !strings.Contains(mirCode, "Test assertion failed") {
		t.Error("Expected assertion message in generated code")
	}

	// Check exception info
	found := false
	for _, info := range integration.exceptionInfos {
		if info.Kind == ExceptionAssert {
			found = true
			if info.Message != "Test assertion failed" {
				t.Errorf("Expected 'Test assertion failed', got %s", info.Message)
			}
		}
	}

	if !found {
		t.Error("Expected assertion exception info")
	}
}

// TestExceptionHandlersMIRGeneration tests exception handlers MIR generation
func TestExceptionHandlersMIRGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	mirCode := integration.GenerateExceptionHandlersMIR()

	// Check for all expected handlers
	expectedHandlers := []string{
		"func panic_bounds_check",
		"func panic_null_pointer",
		"func panic_division_by_zero",
		"func panic_assert",
		"func runtime_abort",
	}

	for _, handler := range expectedHandlers {
		if !strings.Contains(mirCode, handler) {
			t.Errorf("Expected handler %s not found", handler)
		}
	}

	// Check for proper MIR function structure
	if !strings.Contains(mirCode, "entry:") {
		t.Error("Expected entry block labels")
	}
	if !strings.Contains(mirCode, "ret") {
		t.Error("Expected return statements")
	}
	if !strings.Contains(mirCode, "call runtime_abort") {
		t.Error("Expected runtime abort calls")
	}
	if !strings.Contains(mirCode, "alloca") {
		t.Error("Expected stack allocation for messages")
	}
}

// TestMIROptimization tests MIR exception check optimization
func TestMIROptimization(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()
	integration.codeGen.UseTraps = true

	input := "temp_cmp1 = cmp.slt index, 0\nother_instruction"
	output := integration.OptimizeExceptionChecksMIR(input)

	// Should add optimization comment
	if !strings.Contains(output, "optimized") {
		t.Error("Expected optimization comment to be added")
	}

	// Test redundant check removal
	redundantInput := "temp_null_cmp = cmp.eq ptr, 0\n  temp_null_cmp2 = cmp.eq ptr, 0"
	redundantOutput := integration.OptimizeExceptionChecksMIR(redundantInput)

	// Should remove the redundant check
	if strings.Count(redundantOutput, "temp_null_cmp2") > 0 {
		t.Error("Expected redundant null check to be removed")
	}
}

// TestExceptionMetadataMIRGeneration tests exception metadata generation
func TestExceptionMetadataMIRGeneration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	// Add some exception infos
	integration.exceptionInfos["check1"] = &ExceptionInfo{
		Kind:         ExceptionBoundsCheck,
		HandlerLabel: "handler1",
	}
	integration.exceptionInfos["check2"] = &ExceptionInfo{
		Kind:         ExceptionNullPointer,
		HandlerLabel: "handler2",
	}
	integration.exceptionInfos["check3"] = &ExceptionInfo{
		Kind:         ExceptionBoundsCheck,
		HandlerLabel: "handler3",
	}

	metadata := integration.GenerateExceptionMetadataMIR()

	// Check exception count
	if count, ok := metadata["exception_count"].(int); !ok || count != 3 {
		t.Errorf("Expected exception count 3, got %v", metadata["exception_count"])
	}

	// Check exception types
	if types, ok := metadata["exception_types"].(map[string]int); ok {
		if types["bounds_check"] != 2 {
			t.Errorf("Expected 2 bounds checks, got %d", types["bounds_check"])
		}
		if types["null_pointer"] != 1 {
			t.Errorf("Expected 1 null pointer check, got %d", types["null_pointer"])
		}
	} else {
		t.Error("Expected exception_types to be map[string]int")
	}

	// Check format indicator
	if format, ok := metadata["mir_format"].(string); !ok || format != "simplified" {
		t.Errorf("Expected mir_format 'simplified', got %v", metadata["mir_format"])
	}
}

// TestCompleteMIRExceptionSystem tests complete MIR exception system generation
func TestCompleteMIRExceptionSystem(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	completeSystem := integration.GenerateCompleteMIRExceptionSystem()

	// Should contain system header
	if !strings.Contains(completeSystem, "Complete MIR Exception Handling System") {
		t.Error("Expected system header comment")
	}

	// Should contain example function
	if !strings.Contains(completeSystem, "func example_with_exceptions") {
		t.Error("Expected example function")
	}

	// Should contain bounds and null checks
	if !strings.Contains(completeSystem, "Bounds check") {
		t.Error("Expected bounds check in example")
	}
	if !strings.Contains(completeSystem, "Null pointer check") {
		t.Error("Expected null check in example")
	}

	// Should contain exception handlers
	if !strings.Contains(completeSystem, "func panic_bounds_check") {
		t.Error("Expected bounds check handler")
	}
	if !strings.Contains(completeSystem, "func runtime_abort") {
		t.Error("Expected runtime abort function")
	}

	// Should contain proper MIR syntax
	if !strings.Contains(completeSystem, "entry:") {
		t.Error("Expected MIR entry blocks")
	}
	if !strings.Contains(completeSystem, "ret") {
		t.Error("Expected MIR return statements")
	}
	if !strings.Contains(completeSystem, "load") {
		t.Error("Expected MIR load instruction")
	}
}

// Integration test for the complete simplified MIR exception system
func TestSimplifiedMIRExceptionSystemIntegration(t *testing.T) {
	integration := NewSimplifiedMIRIntegration()

	// Generate various exception checks
	boundsCheck := integration.GenerateBoundsCheckMIR("i", "len", "array")
	nullCheck := integration.GenerateNullCheckMIR("ptr", "pointer")
	divCheck := integration.GenerateDivisionCheckMIR("div", "calc")
	assertCheck := integration.GenerateAssertMIR("cond", "Invariant")

	// Verify each check generated proper MIR
	checks := []struct {
		code string
		name string
	}{
		{boundsCheck, "bounds check"},
		{nullCheck, "null check"},
		{divCheck, "division check"},
		{assertCheck, "assertion check"},
	}

	for _, check := range checks {
		if !strings.Contains(check.code, "cmp.") {
			t.Errorf("Expected comparison instruction in %s", check.name)
		}
		if !strings.Contains(check.code, "brcond") {
			t.Errorf("Expected conditional branch in %s", check.name)
		}
		if !strings.Contains(check.code, "call panic_") {
			t.Errorf("Expected panic call in %s", check.name)
		}
	}

	// Verify exception infos were recorded
	if len(integration.exceptionInfos) != 4 {
		t.Errorf("Expected 4 exception infos, got %d", len(integration.exceptionInfos))
	}

	// Generate handlers and complete system
	handlers := integration.GenerateExceptionHandlersMIR()
	complete := integration.GenerateCompleteMIRExceptionSystem()

	if handlers == "" {
		t.Error("Expected non-empty exception handlers")
	}
	if complete == "" {
		t.Error("Expected non-empty complete system")
	}

	// Generate and verify metadata
	metadata := integration.GenerateExceptionMetadataMIR()
	if metadata == nil {
		t.Error("Expected non-nil metadata")
	}

	// Verify all exception types are represented
	if types, ok := metadata["exception_types"].(map[string]int); ok {
		expectedTypes := []string{"bounds_check", "null_pointer", "division_by_zero", "assertion"}
		for _, expectedType := range expectedTypes {
			if types[expectedType] == 0 {
				t.Errorf("Expected %s exception type to be present", expectedType)
			}
		}
	}

	// Test optimization
	originalBounds := boundsCheck
	optimized := integration.OptimizeExceptionChecksMIR(originalBounds)
	if len(optimized) == 0 {
		t.Error("Expected non-empty optimized code")
	}
}
