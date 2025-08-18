package exception

import (
	"strings"
	"testing"
)

// TestX64ExceptionEmitterCreation tests creating the x64 exception emitter
func TestX64ExceptionEmitterCreation(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	if emitter == nil {
		t.Fatal("Expected non-nil x64 exception emitter")
	}

	if emitter.codeGen == nil {
		t.Error("Expected non-nil code generator")
	}

	if emitter.stackAlignment != 16 {
		t.Errorf("Expected stack alignment 16, got %d", emitter.stackAlignment)
	}

	if !emitter.enableOptimization {
		t.Error("Expected optimization to be enabled by default")
	}

	if !emitter.useUnwindInfo {
		t.Error("Expected unwind info to be enabled by default")
	}
}

// TestFunctionPrologueEpilogue tests x64 function prologue and epilogue generation
func TestFunctionPrologueEpilogue(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	prologue := emitter.EmitPrologue("test_function")
	epilogue := emitter.EmitEpilogue("test_function")

	// Check prologue components
	if !strings.Contains(prologue, "test_function:") {
		t.Error("Expected function label in prologue")
	}
	if !strings.Contains(prologue, "push rbp") {
		t.Error("Expected rbp push in prologue")
	}
	if !strings.Contains(prologue, "mov rbp, rsp") {
		t.Error("Expected rbp setup in prologue")
	}
	if !strings.Contains(prologue, "and rsp, -16") {
		t.Error("Expected stack alignment in prologue")
	}

	// Check epilogue components
	if !strings.Contains(epilogue, "mov rsp, rbp") {
		t.Error("Expected rsp restoration in epilogue")
	}
	if !strings.Contains(epilogue, "pop rbp") {
		t.Error("Expected rbp pop in epilogue")
	}
	if !strings.Contains(epilogue, "ret") {
		t.Error("Expected return instruction in epilogue")
	}
}

// TestBoundsCheckX64Generation tests x64 bounds check code generation
func TestBoundsCheckX64Generation(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	// Test with optimization enabled
	emitter.enableOptimization = true
	optimizedCode := emitter.EmitBoundsCheckX64("rax", "rbx", "testArray")

	if !strings.Contains(optimizedCode, "bounds check") {
		t.Error("Expected bounds check comment")
	}
	if !strings.Contains(optimizedCode, "cmp rax, rbx") {
		t.Error("Expected comparison instruction")
	}
	if !strings.Contains(optimizedCode, "jae") {
		t.Error("Expected unsigned comparison jump in optimized version")
	}
	if !strings.Contains(optimizedCode, "panic_bounds_check") {
		t.Error("Expected bounds check panic call")
	}

	// Test with optimization disabled
	emitter.enableOptimization = false
	standardCode := emitter.EmitBoundsCheckX64("rax", "rbx", "testArray")

	if !strings.Contains(standardCode, "test rax, rax") {
		t.Error("Expected negative check in standard version")
	}
	if !strings.Contains(standardCode, "js") {
		t.Error("Expected negative jump in standard version")
	}
	if !strings.Contains(standardCode, "jge") {
		t.Error("Expected greater-equal jump in standard version")
	}
}

// TestNullCheckX64Generation tests x64 null check code generation
func TestNullCheckX64Generation(t *testing.T) {
	emitter := NewX64ExceptionEmitter()
	emitter.codeGen.NullChecking = true // Ensure null checking is enabled

	code := emitter.EmitNullCheckX64("rcx", "testPointer")

	if !strings.Contains(code, "null pointer check") && !strings.Contains(code, "Null pointer check") {
		t.Error("Expected null pointer check comment")
	}
	if !strings.Contains(code, "test rcx, rcx") {
		t.Error("Expected test instruction for null check")
	}
	if !strings.Contains(code, "jz") {
		t.Error("Expected zero jump for null check")
	}
	if !strings.Contains(code, "panic_null_pointer") {
		t.Error("Expected null pointer panic call")
	}
}

// TestDivisionCheckX64Generation tests x64 division check code generation
func TestDivisionCheckX64Generation(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	code := emitter.EmitDivisionCheckX64("rdx", "test_division")

	if !strings.Contains(code, "division by zero check") && !strings.Contains(code, "Division by zero check") {
		t.Error("Expected division by zero check comment")
	}
	if !strings.Contains(code, "test rdx, rdx") {
		t.Error("Expected test instruction for division check")
	}
	if !strings.Contains(code, "jz") {
		t.Error("Expected zero jump for division check")
	}
	if !strings.Contains(code, "panic_division_by_zero") {
		t.Error("Expected division by zero panic call")
	}
}

// TestStackOverflowCheckGeneration tests stack overflow check generation
func TestStackOverflowCheckGeneration(t *testing.T) {
	emitter := NewX64ExceptionEmitter()
	emitter.codeGen.StackGuard = true

	code := emitter.EmitStackOverflowCheck(256)

	if !strings.Contains(code, "stack overflow check") && !strings.Contains(code, "Stack overflow check") {
		t.Error("Expected stack overflow check comment")
	}
	if !strings.Contains(code, "sub rax, 256") {
		t.Error("Expected stack space calculation")
	}
	if !strings.Contains(code, "stack_limit") {
		t.Error("Expected stack limit access")
	}
	if !strings.Contains(code, "panic_stack_overflow") {
		t.Error("Expected stack overflow panic call")
	}
}

// TestStackOverflowCheckDisabled tests that stack overflow check can be disabled
func TestStackOverflowCheckDisabled(t *testing.T) {
	emitter := NewX64ExceptionEmitter()
	emitter.codeGen.StackGuard = false

	code := emitter.EmitStackOverflowCheck(256)

	if code != "" {
		t.Error("Expected no code when stack guard is disabled")
	}
}

// TestExceptionHandlersX64Generation tests complete exception handler generation
func TestExceptionHandlersX64Generation(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	code := emitter.EmitExceptionHandlersX64()

	// Check for all expected handlers
	expectedHandlers := []string{
		"panic_bounds_check:",
		"panic_null_pointer:",
		"panic_division_by_zero:",
		"panic_stack_overflow:",
		"format_bounds_error:",
		"runtime_abort:",
	}

	for _, handler := range expectedHandlers {
		if !strings.Contains(code, handler) {
			t.Errorf("Expected handler %s not found", handler)
		}
	}

	// Check for proper function structure
	if !strings.Contains(code, "push rbp") {
		t.Error("Expected standard function prologue")
	}
	if !strings.Contains(code, "pop rbp") {
		t.Error("Expected standard function epilogue")
	}
	if !strings.Contains(code, "call runtime_abort") {
		t.Error("Expected runtime abort calls")
	}
}

// TestDataSectionGeneration tests data section generation
func TestDataSectionGeneration(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	code := emitter.EmitDataSection()

	if !strings.Contains(code, ".data") {
		t.Error("Expected .data section directive")
	}
	if !strings.Contains(code, ".align 8") {
		t.Error("Expected data alignment directive")
	}

	// Check for all expected message strings
	expectedMessages := []string{
		"bounds_check_msg:",
		"null_ptr_msg:",
		"div_zero_msg:",
		"stack_overflow_msg:",
		"assert_msg:",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(code, msg) {
			t.Errorf("Expected message %s not found", msg)
		}
	}

	// Check for format strings
	if !strings.Contains(code, "bounds_fmt_str:") {
		t.Error("Expected bounds format string")
	}
	if !strings.Contains(code, "exception_info:") {
		t.Error("Expected exception info structure")
	}
}

// TestExternDeclarations tests extern declaration generation
func TestExternDeclarations(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	code := emitter.EmitExternDeclarations()

	expectedExterns := []string{
		"extern GetStdHandle",
		"extern WriteFile",
		"extern ExitProcess",
		"extern sprintf",
		"extern strlen",
	}

	for _, extern := range expectedExterns {
		if !strings.Contains(code, extern) {
			t.Errorf("Expected extern declaration %s not found", extern)
		}
	}
}

// TestInlineExceptionChecks tests inline exception check generation
func TestInlineExceptionChecks(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	// Test bounds check
	boundsCode := emitter.EmitInlineExceptionCheck("bounds", "rax", "rbx", "myArray")
	if !strings.Contains(boundsCode, "bounds check") {
		t.Error("Expected bounds check in inline code")
	}

	// Test null check
	nullCode := emitter.EmitInlineExceptionCheck("null", "rcx", "myPtr")
	if !strings.Contains(nullCode, "null pointer check") && !strings.Contains(nullCode, "Null pointer check") {
		t.Error("Expected null check in inline code")
	}

	// Test division check
	divCode := emitter.EmitInlineExceptionCheck("division", "rdx", "myDiv")
	if !strings.Contains(divCode, "division by zero check") && !strings.Contains(divCode, "Division by zero check") {
		t.Error("Expected division check in inline code")
	}

	// Test stack check
	stackCode := emitter.EmitInlineExceptionCheck("stack", "128")
	if emitter.codeGen.StackGuard {
		if !strings.Contains(stackCode, "stack overflow check") {
			t.Error("Expected stack check in inline code")
		}
	}

	// Test unknown check type
	unknownCode := emitter.EmitInlineExceptionCheck("unknown", "arg1")
	if !strings.Contains(unknownCode, "Unknown exception check type") {
		t.Error("Expected unknown check type message")
	}
}

// TestCompleteExceptionSystem tests the complete exception system generation
func TestCompleteExceptionSystem(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	code := emitter.EmitCompleteExceptionSystem()

	// Should contain all major components
	if !strings.Contains(code, "extern") {
		t.Error("Expected extern declarations")
	}
	if !strings.Contains(code, "panic_bounds_check:") {
		t.Error("Expected exception handlers")
	}
	if !strings.Contains(code, ".data") {
		t.Error("Expected data section")
	}

	// Check for system completeness
	if !strings.Contains(code, "Complete x64 Exception Handling System") {
		t.Error("Expected system header comment")
	}
}

// TestUnwindInfoGeneration tests Windows SEH unwind info generation
func TestUnwindInfoGeneration(t *testing.T) {
	emitter := NewX64ExceptionEmitter()
	emitter.useUnwindInfo = true

	code := emitter.emitUnwindInfo()

	if !strings.Contains(code, ".pdata") {
		t.Error("Expected .pdata section")
	}
	if !strings.Contains(code, ".xdata") {
		t.Error("Expected .xdata section")
	}
	if !strings.Contains(code, "panic_bounds_check_unwind") {
		t.Error("Expected unwind info for bounds check handler")
	}
	if !strings.Contains(code, "UWOP_PUSH_NONVOL") {
		t.Error("Expected unwind operation codes")
	}
}

// TestOptimizeExceptionChecks tests x64-specific optimizations
func TestOptimizeExceptionChecks(t *testing.T) {
	emitter := NewX64ExceptionEmitter()
	emitter.enableOptimization = true

	// Test redundant test instruction removal
	input1 := "test rax, rax\n  test rax, rax\n"
	output1 := emitter.OptimizeExceptionChecks(input1)
	expected1 := "test rax, rax\n"

	if output1 != expected1 {
		t.Errorf("Expected %q, got %q", expected1, output1)
	}

	// Test comparison combination
	input2 := "cmp rax, 0\n  test rax, rax\n"
	output2 := emitter.OptimizeExceptionChecks(input2)
	expected2 := "test rax, rax\n"

	if output2 != expected2 {
		t.Errorf("Expected %q, got %q", expected2, output2)
	}

	// Test unnecessary jump removal
	input3 := "jmp .L1\n.L1:\n"
	output3 := emitter.OptimizeExceptionChecks(input3)
	expected3 := ""

	if output3 != expected3 {
		t.Errorf("Expected %q, got %q", expected3, output3)
	}
}

// TestDisabledChecksX64 tests that x64 checks can be disabled
func TestDisabledChecksX64(t *testing.T) {
	emitter := NewX64ExceptionEmitter()
	emitter.codeGen.BoundsChecking = false
	emitter.codeGen.NullChecking = false

	boundsCode := emitter.EmitBoundsCheckX64("rax", "rbx", "test")
	if boundsCode != "" {
		t.Error("Expected no bounds check code when disabled")
	}

	nullCode := emitter.EmitNullCheckX64("rcx", "test")
	if nullCode != "" {
		t.Error("Expected no null check code when disabled")
	}
}

// TestParseIntSafe tests the safe integer parsing function
func TestParseIntSafe(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
	}{
		{"32", 32},
		{"64", 64},
		{"128", 128},
		{"invalid", 0},
		{"", 0},
	}

	for _, tc := range testCases {
		result := parseIntSafe(tc.input)
		if result != tc.expected {
			t.Errorf("For input %q: expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

// Integration test for the complete x64 exception system
func TestX64ExceptionSystemIntegration(t *testing.T) {
	emitter := NewX64ExceptionEmitter()

	// Test creating a complete function with exception handling
	var completeFunction strings.Builder

	// Function prologue
	completeFunction.WriteString(emitter.EmitPrologue("test_function"))

	// Add various exception checks
	completeFunction.WriteString(emitter.EmitBoundsCheckX64("rsi", "rdi", "array"))
	completeFunction.WriteString(emitter.EmitNullCheckX64("rbx", "pointer"))
	completeFunction.WriteString(emitter.EmitDivisionCheckX64("rcx", "division"))
	completeFunction.WriteString(emitter.EmitStackOverflowCheck(64))

	// Function epilogue
	completeFunction.WriteString(emitter.EmitEpilogue("test_function"))

	// Generate complete system
	completeSystem := emitter.EmitCompleteExceptionSystem()

	functionCode := completeFunction.String()

	// Verify function structure
	if !strings.Contains(functionCode, "test_function:") {
		t.Error("Expected function label")
	}
	if !strings.Contains(functionCode, "push rbp") {
		t.Error("Expected function prologue")
	}
	if !strings.Contains(functionCode, "ret") {
		t.Error("Expected function epilogue")
	}

	// Verify exception checks are present
	checks := []string{"bounds check", "Bounds check", "null pointer check", "Null pointer check", "division by zero check", "Division by zero check"}
	for _, check := range checks {
		if strings.Contains(functionCode, check) {
			// Found at least one expected check pattern
			break
		}
		if check == checks[len(checks)-1] {
			// Reached end without finding any pattern
			t.Errorf("Expected exception checks in function code")
		}
	}

	// Verify complete system has all components
	systemComponents := []string{
		"extern", ".data", "panic_bounds_check:",
		"bounds_check_msg:", "runtime_abort:",
	}
	for _, component := range systemComponents {
		if !strings.Contains(completeSystem, component) {
			t.Errorf("Expected %s in complete system", component)
		}
	}

	// Test optimization
	if emitter.enableOptimization {
		optimized := emitter.OptimizeExceptionChecks(functionCode)
		if len(optimized) > len(functionCode) {
			t.Error("Expected optimization to reduce or maintain code size")
		}
	}
}
