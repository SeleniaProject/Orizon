package exception

import (
	"os"
	"strings"
	"testing"
)

// TestExceptionCreation tests basic exception creation
func TestExceptionCreation(t *testing.T) {
	exception := &Exception{
		Kind:     ExceptionPanic,
		Message:  "Test panic",
		Location: "test.go:10",
	}

	if exception.Kind != ExceptionPanic {
		t.Errorf("Expected ExceptionPanic, got %v", exception.Kind)
	}

	if exception.Message != "Test panic" {
		t.Errorf("Expected 'Test panic', got %s", exception.Message)
	}

	if exception.Location != "test.go:10" {
		t.Errorf("Expected 'test.go:10', got %s", exception.Location)
	}
}

// TestAbortHandler tests the abort handler formatting
func TestAbortHandler(t *testing.T) {
	handler := &AbortHandler{
		ShowStackTrace: false,
		LogToFile:      false,
	}

	exception := &Exception{
		Kind:     ExceptionBoundsCheck,
		Message:  "Index out of bounds",
		Location: "array.go:25",
	}

	formatted := handler.formatException(exception)
	expected := "[BOUNDS_CHECK] Index out of bounds at array.go:25"

	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

// TestNestedExceptions tests nested exception handling
func TestNestedExceptions(t *testing.T) {
	handler := &AbortHandler{
		ShowStackTrace: false,
		LogToFile:      false,
	}

	innerException := &Exception{
		Kind:     ExceptionDivisionByZero,
		Message:  "Division by zero",
		Location: "math.go:15",
	}

	outerException := &Exception{
		Kind:           ExceptionUser,
		Message:        "Calculation failed",
		Location:       "calc.go:42",
		InnerException: innerException,
	}

	formatted := handler.formatException(outerException)

	if !strings.Contains(formatted, "Calculation failed") {
		t.Error("Expected outer exception message")
	}

	if !strings.Contains(formatted, "Caused by:") {
		t.Error("Expected nested exception indicator")
	}

	if !strings.Contains(formatted, "Division by zero") {
		t.Error("Expected inner exception message")
	}
}

// TestExceptionKindToString tests exception kind string conversion
func TestExceptionKindToString(t *testing.T) {
	handler := &AbortHandler{}

	testCases := []struct {
		kind     ExceptionKind
		expected string
	}{
		{ExceptionPanic, "PANIC"},
		{ExceptionAssert, "ASSERTION_FAILED"},
		{ExceptionBoundsCheck, "BOUNDS_CHECK"},
		{ExceptionNullPointer, "NULL_POINTER"},
		{ExceptionDivisionByZero, "DIVISION_BY_ZERO"},
		{ExceptionStackOverflow, "STACK_OVERFLOW"},
		{ExceptionOutOfMemory, "OUT_OF_MEMORY"},
		{ExceptionUser, "USER_EXCEPTION"},
	}

	for _, tc := range testCases {
		result := handler.kindToString(tc.kind)
		if result != tc.expected {
			t.Errorf("Expected %s for %v, got %s", tc.expected, tc.kind, result)
		}
	}
}

// TestSetExceptionHandler tests setting global exception handler
func TestSetExceptionHandler(t *testing.T) {
	originalHandler := currentHandler
	defer func() {
		currentHandler = originalHandler
	}()

	newHandler := &AbortHandler{
		ShowStackTrace: false,
		LogToFile:      true,
		LogFile:        "test.log",
	}

	SetExceptionHandler(newHandler)

	if currentHandler != newHandler {
		t.Error("Expected handler to be set")
	}
}

// TestAssert tests assertion functionality
func TestAssert(t *testing.T) {
	// Test successful assertion (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Error("Assert should not panic on true condition")
		}
	}()

	// This should not trigger anything
	Assert(true, "This should pass")
}

// TestBoundsChecking tests bounds checking functionality
func TestBoundsChecking(t *testing.T) {
	// Test valid bounds (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Error("CheckBounds should not panic on valid bounds")
		}
	}()

	CheckBounds(5, 10, "testArray")
}

// TestNullPointerCheck tests null pointer checking
func TestNullPointerCheck(t *testing.T) {
	// Test non-null pointer (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Error("CheckNullPointer should not panic on non-null pointer")
		}
	}()

	value := 42
	CheckNullPointer(&value, "testPointer")
}

// TestDivisionByZeroCheck tests division by zero checking
func TestDivisionByZeroCheck(t *testing.T) {
	// Test non-zero divisor (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Error("CheckDivisionByZero should not panic on non-zero divisor")
		}
	}()

	CheckDivisionByZero(5, "test division")
	CheckDivisionByZero(5.5, "test float division")
}

// TestStackTraceCapture tests stack trace capturing
func TestStackTraceCapture(t *testing.T) {
	stackTrace := captureStackTrace()

	if len(stackTrace) == 0 {
		t.Error("Expected non-empty stack trace")
	}

	// Check that we have reasonable stack frame information
	for i, frame := range stackTrace {
		if frame.Function == "" {
			t.Errorf("Frame %d has empty function name", i)
		}

		if frame.File == "" {
			t.Errorf("Frame %d has empty file name", i)
		}

		if frame.Line <= 0 {
			t.Errorf("Frame %d has invalid line number: %d", i, frame.Line)
		}
	}
}

// TestCallerLocation tests caller location functionality
func TestCallerLocation(t *testing.T) {
	location := getCallerLocation()

	if location == "unknown" {
		t.Error("Expected valid caller location")
	}

	if !strings.Contains(location, ":") {
		t.Error("Expected location to contain line number")
	}

	if !strings.Contains(location, ".go") {
		t.Error("Expected location to contain Go file")
	}
}

// MockHandler for testing exception handling without aborting
type MockHandler struct {
	LastException *Exception
	HandleCount   int
}

func (mh *MockHandler) HandleException(exception *Exception) bool {
	mh.LastException = exception
	mh.HandleCount++
	return true // Don't actually abort
}

// TestCustomExceptionHandler tests custom exception handling
func TestCustomExceptionHandler(t *testing.T) {
	originalHandler := currentHandler
	defer func() {
		currentHandler = originalHandler
	}()

	mockHandler := &MockHandler{}
	SetExceptionHandler(mockHandler)

	// Create a test exception and verify it's handled
	testException := &Exception{
		Kind:     ExceptionUser,
		Message:  "Test exception",
		Location: "test.go:1",
	}

	currentHandler.HandleException(testException)

	if mockHandler.HandleCount != 1 {
		t.Errorf("Expected 1 handled exception, got %d", mockHandler.HandleCount)
	}

	if mockHandler.LastException != testException {
		t.Error("Expected last exception to match test exception")
	}
}

// TestTryRecovery tests the basic try-recovery mechanism
func TestTryRecovery(t *testing.T) {
	recovery := NewTryRecovery()

	handlerCalled := false
	recovery.AddHandler(ExceptionPanic, func(e *Exception) bool {
		handlerCalled = true
		return true // Recovered
	})

	// Test successful recovery
	recovered := recovery.Try(func() {
		// Simulate an exception by panicking
		panic("test panic")
	})

	if !recovered {
		t.Error("Expected recovery to succeed")
	}

	if !handlerCalled {
		t.Error("Expected exception handler to be called")
	}
}

// TestExceptionCodeGen tests code generation for exception handling
func TestExceptionCodeGen(t *testing.T) {
	codeGen := NewExceptionCodeGen()

	// Test bounds check code generation
	boundsCode := codeGen.EmitBoundsCheck("rax", "rbx", "myArray")
	if !strings.Contains(boundsCode, "bounds check") {
		t.Error("Expected bounds check comment in generated code")
	}
	if !strings.Contains(boundsCode, "cmp") {
		t.Error("Expected comparison instruction in bounds check")
	}
	if !strings.Contains(boundsCode, "jl") || !strings.Contains(boundsCode, "jge") {
		t.Error("Expected conditional jumps in bounds check")
	}

	// Test null check code generation
	nullCode := codeGen.EmitNullCheck("rcx", "myPointer")
	if !strings.Contains(nullCode, "null check") {
		t.Error("Expected null check comment in generated code")
	}
	if !strings.Contains(nullCode, "test") {
		t.Error("Expected test instruction in null check")
	}
	if !strings.Contains(nullCode, "jz") {
		t.Error("Expected zero jump in null check")
	}

	// Test division check code generation
	divCode := codeGen.EmitDivisionCheck("rdx")
	if !strings.Contains(divCode, "division by zero") {
		t.Error("Expected division check comment in generated code")
	}
	if !strings.Contains(divCode, "test") {
		t.Error("Expected test instruction in division check")
	}

	// Test panic handler code generation
	panicCode := codeGen.EmitPanicHandler()
	if !strings.Contains(panicCode, "panic_bounds_check") {
		t.Error("Expected bounds check panic handler")
	}
	if !strings.Contains(panicCode, "panic_null_pointer") {
		t.Error("Expected null pointer panic handler")
	}
	if !strings.Contains(panicCode, "panic_division_by_zero") {
		t.Error("Expected division by zero panic handler")
	}
	if !strings.Contains(panicCode, "bounds_check_msg") {
		t.Error("Expected bounds check message constant")
	}
}

// TestDisabledChecks tests that checks can be disabled
func TestDisabledChecks(t *testing.T) {
	codeGen := &ExceptionCodeGen{
		BoundsChecking: false,
		NullChecking:   false,
	}

	boundsCode := codeGen.EmitBoundsCheck("rax", "rbx", "test")
	if boundsCode != "" {
		t.Error("Expected no bounds check code when disabled")
	}

	nullCode := codeGen.EmitNullCheck("rcx", "test")
	if nullCode != "" {
		t.Error("Expected no null check code when disabled")
	}
}

// Integration test that verifies the complete exception system
func TestExceptionSystemIntegration(t *testing.T) {
	// Create a test log file in temp directory
	tmpFile := "/tmp/test_exception.log"
	defer os.Remove(tmpFile)

	// Set up logging handler
	originalHandler := currentHandler
	defer func() {
		currentHandler = originalHandler
	}()

	logHandler := &AbortHandler{
		ShowStackTrace: true,
		LogToFile:      true,
		LogFile:        tmpFile,
	}

	SetExceptionHandler(logHandler)

	// Test exception creation with stack trace
	exception := &Exception{
		Kind:       ExceptionBoundsCheck,
		Message:    "Integration test exception",
		Location:   "integration_test.go:1",
		StackTrace: captureStackTrace(),
	}

	// Verify stack trace was captured
	if len(exception.StackTrace) == 0 {
		t.Error("Expected stack trace to be captured")
	}

	// Verify the first frame has reasonable information
	if len(exception.StackTrace) > 0 {
		frame := exception.StackTrace[0]
		if frame.Function == "" {
			t.Error("Expected function name in stack frame")
		}
		if frame.File == "" {
			t.Error("Expected file name in stack frame")
		}
		if frame.Line <= 0 {
			t.Error("Expected valid line number in stack frame")
		}
	}

	// Test code generation produces valid assembly
	codeGen := NewExceptionCodeGen()
	allCode := codeGen.EmitBoundsCheck("rax", "10", "testArray") +
		codeGen.EmitNullCheck("rbx", "testPtr") +
		codeGen.EmitDivisionCheck("rcx") +
		codeGen.EmitPanicHandler()

	// Verify the generated code has all necessary components
	requiredComponents := []string{
		"bounds check", "null check", "division by zero",
		"panic_bounds_check", "panic_null_pointer", "panic_division_by_zero",
		"cmp", "test", "jl", "jge", "jz",
		"bounds_check_msg", "null_pointer_msg", "div_zero_msg",
	}

	for _, component := range requiredComponents {
		if !strings.Contains(allCode, component) {
			t.Errorf("Generated code missing required component: %s", component)
		}
	}
}
