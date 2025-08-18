package exception

import (
	"strings"
	"testing"

	"github.com/OrizonProject/Orizon/internal/mir"
)

// TestMIRExceptionIntegrationCreation tests creating the MIR integration
func TestMIRExceptionIntegrationCreation(t *testing.T) {
	integration := NewMIRExceptionIntegration()

	if integration == nil {
		t.Fatal("Expected non-nil MIR exception integration")
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

// TestLabelGeneration tests unique label generation
func TestLabelGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()

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

// TestBoundsCheckMIRGeneration tests bounds check MIR generation
func TestBoundsCheckMIRGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()
	block := &mir.BasicBlock{
		Label:        "test_block",
		Instructions: []*mir.Instruction{},
	}

	initialInstructionCount := len(block.Instructions)
	integration.AddBoundsCheckToMIR(block, "index_var", "length_var", "myArray")

	// Should add multiple instructions for bounds checking
	if len(block.Instructions) <= initialInstructionCount {
		t.Error("Expected instructions to be added to block")
	}

	// Check that we have comparison instructions
	hasComparison := false
	hasBranch := false
	hasCall := false

	for _, inst := range block.Instructions {
		switch inst.Type {
		case mir.InstCmp:
			hasComparison = true
		case mir.InstBranchCond:
			hasBranch = true
		case mir.InstCall:
			hasCall = true
			if inst.Arg1 != "panic_bounds_check" {
				t.Errorf("Expected panic_bounds_check call, got %s", inst.Arg1)
			}
		}
	}

	if !hasComparison {
		t.Error("Expected comparison instruction for bounds check")
	}
	if !hasBranch {
		t.Error("Expected branch instruction for bounds check")
	}
	if !hasCall {
		t.Error("Expected call instruction for bounds check")
	}

	// Check that exception info was recorded
	if len(integration.exceptionInfos) == 0 {
		t.Error("Expected exception info to be recorded")
	}
}

// TestNullCheckMIRGeneration tests null check MIR generation
func TestNullCheckMIRGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()
	block := &mir.BasicBlock{
		Label:        "test_block",
		Instructions: []*mir.Instruction{},
	}

	integration.AddNullCheckToMIR(block, "ptr_var", "myPointer")

	// Verify null check instructions
	hasNullCheck := false
	for _, inst := range block.Instructions {
		if inst.Type == mir.InstCall && inst.Arg1 == "panic_null_pointer" {
			hasNullCheck = true
			break
		}
	}

	if !hasNullCheck {
		t.Error("Expected null pointer panic call")
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

// TestDivisionCheckMIRGeneration tests division check MIR generation
func TestDivisionCheckMIRGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()
	block := &mir.BasicBlock{
		Label:        "test_block",
		Instructions: []*mir.Instruction{},
	}

	integration.AddDivisionCheckToMIR(block, "divisor_var", "test_division")

	// Verify division check instructions
	hasDivCheck := false
	for _, inst := range block.Instructions {
		if inst.Type == mir.InstCall && inst.Arg1 == "panic_division_by_zero" {
			hasDivCheck = true
			break
		}
	}

	if !hasDivCheck {
		t.Error("Expected division by zero panic call")
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

// TestAssertMIRGeneration tests assertion MIR generation
func TestAssertMIRGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()
	block := &mir.BasicBlock{
		Label:        "test_block",
		Instructions: []*mir.Instruction{},
	}

	integration.AddAssertToMIR(block, "condition_var", "Test assertion failed")

	// Verify assertion instructions
	hasAssert := false
	for _, inst := range block.Instructions {
		if inst.Type == mir.InstCall && inst.Arg1 == "panic_assert" {
			hasAssert = true
			break
		}
	}

	if !hasAssert {
		t.Error("Expected assertion panic call")
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

// TestExceptionHandlerGeneration tests exception handler MIR generation
func TestExceptionHandlerGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()
	function := integration.GenerateExceptionHandlers()

	if function == nil {
		t.Fatal("Expected non-nil exception handler function")
	}

	if function.Name != "exception_handlers" {
		t.Errorf("Expected function name 'exception_handlers', got %s", function.Name)
	}

	if len(function.Blocks) == 0 {
		t.Error("Expected non-empty blocks in exception handler function")
	}

	// Check for specific handler blocks
	expectedHandlers := []string{
		"panic_bounds_check",
		"panic_null_pointer",
		"panic_division_by_zero",
		"panic_assert",
	}

	found := make(map[string]bool)
	for _, block := range function.Blocks {
		for _, handler := range expectedHandlers {
			if block.Label == handler {
				found[handler] = true
			}
		}
	}

	for _, handler := range expectedHandlers {
		if !found[handler] {
			t.Errorf("Expected handler block %s not found", handler)
		}
	}
}

// TestRedundantBoundsCheckRemoval tests optimization of redundant bounds checks
func TestRedundantBoundsCheckRemoval(t *testing.T) {
	integration := NewMIRExceptionIntegration()
	block := &mir.BasicBlock{
		Label: "test_block",
		Instructions: []*mir.Instruction{
			{
				Type:  mir.InstCmp,
				Dest:  "cmp1",
				Arg1:  "index",
				Arg2:  "length",
				Label: "bounds_check_1",
			},
			{
				Type:  mir.InstCmp,
				Dest:  "cmp2",
				Arg1:  "index",
				Arg2:  "length",
				Label: "bounds_check_2", // Same check, should be removed
			},
			{
				Type: mir.InstAdd,
				Dest: "result",
				Arg1: "a",
				Arg2: "b",
			},
		},
	}

	originalCount := len(block.Instructions)
	integration.removeRedundantBoundsChecks(block)

	if len(block.Instructions) >= originalCount {
		t.Error("Expected redundant bounds checks to be removed")
	}

	// Verify the non-bounds-check instruction is still there
	hasAddInst := false
	for _, inst := range block.Instructions {
		if inst.Type == mir.InstAdd {
			hasAddInst = true
		}
	}

	if !hasAddInst {
		t.Error("Expected non-bounds-check instruction to remain")
	}
}

// TestExceptionCheckOptimization tests exception check optimization
func TestExceptionCheckOptimization(t *testing.T) {
	integration := NewMIRExceptionIntegration()

	// Create a function with multiple blocks containing various checks
	function := &mir.Function{
		Name: "test_function",
		Blocks: []*mir.BasicBlock{
			{
				Label: "block1",
				Instructions: []*mir.Instruction{
					{
						Type:  mir.InstCmp,
						Label: "bounds_check_1",
						Arg1:  "index1",
						Arg2:  "length1",
					},
					{
						Type:  mir.InstCmp,
						Label: "null_check_1",
						Arg1:  "ptr1",
						Arg2:  "0",
					},
					{
						Type:  mir.InstCmp,
						Label: "bounds_check_2",
						Arg1:  "index1", // Same as bounds_check_1
						Arg2:  "length1",
					},
				},
			},
		},
	}

	originalInstructionCount := len(function.Blocks[0].Instructions)
	integration.OptimizeExceptionChecks(function)

	// Should have removed redundant bounds check
	if len(function.Blocks[0].Instructions) >= originalInstructionCount {
		t.Error("Expected optimization to reduce instruction count")
	}
}

// TestExceptionMetadataGeneration tests exception metadata generation
func TestExceptionMetadataGeneration(t *testing.T) {
	integration := NewMIRExceptionIntegration()

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

	metadata := integration.GenerateExceptionMetadata()

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

	// Check check labels
	if labels, ok := metadata["check_labels"].([]string); ok {
		if len(labels) != 3 {
			t.Errorf("Expected 3 check labels, got %d", len(labels))
		}
	} else {
		t.Error("Expected check_labels to be []string")
	}

	// Check handler labels
	if handlers, ok := metadata["handler_labels"].([]string); ok {
		if len(handlers) != 3 {
			t.Errorf("Expected 3 handler labels, got %d", len(handlers))
		}
	} else {
		t.Error("Expected handler_labels to be []string")
	}
}

// TestIsExceptionCheck tests exception check detection
func TestIsExceptionCheck(t *testing.T) {
	integration := NewMIRExceptionIntegration()

	testCases := []struct {
		instruction *mir.Instruction
		expected    bool
		description string
	}{
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "bounds_check_1"},
			true,
			"bounds check",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "null_check_1"},
			true,
			"null check",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "div_check_1"},
			true,
			"division check",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "assert_check_1"},
			true,
			"assertion check",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "normal_comparison"},
			false,
			"normal comparison",
		},
		{
			&mir.Instruction{Type: mir.InstAdd, Label: "bounds_check_1"},
			false,
			"non-comparison with exception label",
		},
	}

	for _, tc := range testCases {
		result := integration.isExceptionCheck(tc.instruction)
		if result != tc.expected {
			t.Errorf("For %s: expected %v, got %v", tc.description, tc.expected, result)
		}
	}
}

// TestGetCheckCost tests exception check cost calculation
func TestGetCheckCost(t *testing.T) {
	integration := NewMIRExceptionIntegration()

	testCases := []struct {
		instruction *mir.Instruction
		expected    int
		description string
	}{
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "null_check_1"},
			1,
			"null check (cheapest)",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "div_check_1"},
			2,
			"division check",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "assert_check_1"},
			3,
			"assertion check",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "bounds_check_1"},
			4,
			"bounds check (most expensive)",
		},
		{
			&mir.Instruction{Type: mir.InstCmp, Label: "unknown_check"},
			5,
			"unknown check",
		},
	}

	for _, tc := range testCases {
		result := integration.getCheckCost(tc.instruction)
		if result != tc.expected {
			t.Errorf("For %s: expected cost %d, got %d", tc.description, tc.expected, result)
		}
	}
}

// Integration test for the complete MIR exception system
func TestMIRExceptionSystemIntegration(t *testing.T) {
	integration := NewMIRExceptionIntegration()

	// Create a complex function with multiple exception checks
	function := &mir.Function{
		Name:   "complex_function",
		Params: []string{"param1", "param2"},
		Blocks: []*mir.BasicBlock{
			{
				Label:        "entry",
				Instructions: []*mir.Instruction{},
			},
			{
				Label:        "loop",
				Instructions: []*mir.Instruction{},
			},
		},
	}

	// Add various exception checks to blocks
	integration.AddBoundsCheckToMIR(function.Blocks[0], "index", "length", "array1")
	integration.AddNullCheckToMIR(function.Blocks[0], "pointer", "ptr1")
	integration.AddDivisionCheckToMIR(function.Blocks[1], "divisor", "division_op")
	integration.AddAssertToMIR(function.Blocks[1], "condition", "Loop invariant")

	// Verify exception infos were recorded
	if len(integration.exceptionInfos) != 4 {
		t.Errorf("Expected 4 exception infos, got %d", len(integration.exceptionInfos))
	}

	// Optimize the function
	integration.OptimizeExceptionChecks(function)

	// Generate exception handlers
	handlers := integration.GenerateExceptionHandlers()
	if handlers == nil {
		t.Error("Expected non-nil exception handlers")
	}

	// Generate metadata
	metadata := integration.GenerateExceptionMetadata()
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

	// Verify that the system can handle complex scenarios
	totalInstructions := 0
	for _, block := range function.Blocks {
		totalInstructions += len(block.Instructions)
	}

	if totalInstructions == 0 {
		t.Error("Expected function to have instructions after adding exception checks")
	}
}
