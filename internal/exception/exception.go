// Package exception provides minimal exception and panic handling for the Orizon compiler.
// This implements a basic abort strategy for exception handling during the bootstrap phase.
package exception

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// ExceptionKind represents different types of exceptions
type ExceptionKind int

const (
	ExceptionPanic ExceptionKind = iota
	ExceptionAssert
	ExceptionBoundsCheck
	ExceptionNullPointer
	ExceptionDivisionByZero
	ExceptionStackOverflow
	ExceptionOutOfMemory
	ExceptionUser // User-defined exceptions
)

// Exception represents a runtime exception
type Exception struct {
	Kind           ExceptionKind
	Message        string
	Location       string       // Source location where exception occurred
	StackTrace     []StackFrame // Stack trace at exception point
	InnerException *Exception   // Nested exception (if any)
}

// StackFrame represents a single frame in the call stack
type StackFrame struct {
	Function string  // Function name
	File     string  // Source file
	Line     int     // Line number
	PC       uintptr // Program counter
}

// ExceptionHandler defines the interface for handling exceptions
type ExceptionHandler interface {
	HandleException(exception *Exception) bool // Returns true if handled
}

// AbortHandler implements the abort strategy for exception handling
type AbortHandler struct {
	ShowStackTrace bool
	LogToFile      bool
	LogFile        string
}

// HandleException implements the abort strategy
func (ah *AbortHandler) HandleException(exception *Exception) bool {
	fmt.Fprintf(os.Stderr, "FATAL: %s\n", ah.formatException(exception))

	if ah.ShowStackTrace && len(exception.StackTrace) > 0 {
		fmt.Fprintf(os.Stderr, "\nStack trace:\n")
		for i, frame := range exception.StackTrace {
			fmt.Fprintf(os.Stderr, "  %d: %s at %s:%d\n", i, frame.Function, frame.File, frame.Line)
		}
	}

	if ah.LogToFile && ah.LogFile != "" {
		ah.logToFile(exception)
	}

	// Abort strategy: immediately terminate
	os.Exit(1)
	return true // Never reached, but satisfies interface
}

// formatException creates a human-readable exception message
func (ah *AbortHandler) formatException(exception *Exception) string {
	var b strings.Builder

	// Exception kind and message
	b.WriteString(fmt.Sprintf("[%s] %s", ah.kindToString(exception.Kind), exception.Message))

	// Location information
	if exception.Location != "" {
		b.WriteString(fmt.Sprintf(" at %s", exception.Location))
	}

	// Nested exception
	if exception.InnerException != nil {
		b.WriteString(fmt.Sprintf("\nCaused by: %s", ah.formatException(exception.InnerException)))
	}

	return b.String()
}

// kindToString converts exception kind to string
func (ah *AbortHandler) kindToString(kind ExceptionKind) string {
	switch kind {
	case ExceptionPanic:
		return "PANIC"
	case ExceptionAssert:
		return "ASSERTION_FAILED"
	case ExceptionBoundsCheck:
		return "BOUNDS_CHECK"
	case ExceptionNullPointer:
		return "NULL_POINTER"
	case ExceptionDivisionByZero:
		return "DIVISION_BY_ZERO"
	case ExceptionStackOverflow:
		return "STACK_OVERFLOW"
	case ExceptionOutOfMemory:
		return "OUT_OF_MEMORY"
	case ExceptionUser:
		return "USER_EXCEPTION"
	default:
		return "UNKNOWN"
	}
}

// logToFile logs the exception to a file
func (ah *AbortHandler) logToFile(exception *Exception) {
	file, err := os.OpenFile(ah.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "[%s] %s\n", getCurrentTimestamp(), ah.formatException(exception))
}

// Runtime exception handling

var (
	currentHandler ExceptionHandler = &AbortHandler{
		ShowStackTrace: true,
		LogToFile:      false,
	}
)

// SetExceptionHandler sets the global exception handler
func SetExceptionHandler(handler ExceptionHandler) {
	currentHandler = handler
}

// Panic raises a panic exception with the given message
func Panic(message string) {
	exception := &Exception{
		Kind:       ExceptionPanic,
		Message:    message,
		Location:   getCallerLocation(),
		StackTrace: captureStackTrace(),
	}

	currentHandler.HandleException(exception)
}

// Assert checks a condition and panics if it's false
func Assert(condition bool, message string) {
	if !condition {
		exception := &Exception{
			Kind:       ExceptionAssert,
			Message:    message,
			Location:   getCallerLocation(),
			StackTrace: captureStackTrace(),
		}

		currentHandler.HandleException(exception)
	}
}

// CheckBounds performs bounds checking and panics on violation
func CheckBounds(index, length int, arrayName string) {
	if index < 0 || index >= length {
		exception := &Exception{
			Kind:       ExceptionBoundsCheck,
			Message:    fmt.Sprintf("Index %d out of bounds for %s[%d]", index, arrayName, length),
			Location:   getCallerLocation(),
			StackTrace: captureStackTrace(),
		}

		currentHandler.HandleException(exception)
	}
}

// CheckNullPointer checks for null pointer and panics if null
func CheckNullPointer(ptr interface{}, name string) {
	if ptr == nil {
		exception := &Exception{
			Kind:       ExceptionNullPointer,
			Message:    fmt.Sprintf("Null pointer access: %s", name),
			Location:   getCallerLocation(),
			StackTrace: captureStackTrace(),
		}

		currentHandler.HandleException(exception)
	}
}

// CheckDivisionByZero checks for division by zero
func CheckDivisionByZero(divisor interface{}, operation string) {
	isZero := false

	switch v := divisor.(type) {
	case int:
		isZero = v == 0
	case int32:
		isZero = v == 0
	case int64:
		isZero = v == 0
	case float32:
		isZero = v == 0.0
	case float64:
		isZero = v == 0.0
	}

	if isZero {
		exception := &Exception{
			Kind:       ExceptionDivisionByZero,
			Message:    fmt.Sprintf("Division by zero in %s", operation),
			Location:   getCallerLocation(),
			StackTrace: captureStackTrace(),
		}

		currentHandler.HandleException(exception)
	}
}

// ThrowUserException throws a user-defined exception
func ThrowUserException(message string, innerException *Exception) {
	exception := &Exception{
		Kind:           ExceptionUser,
		Message:        message,
		Location:       getCallerLocation(),
		StackTrace:     captureStackTrace(),
		InnerException: innerException,
	}

	currentHandler.HandleException(exception)
}

// Utility functions for stack tracing and location

// getCallerLocation returns the location of the caller
func getCallerLocation() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}

	// Extract just the filename, not the full path
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return fmt.Sprintf("%s:%d", file, line)
}

// captureStackTrace captures the current stack trace
func captureStackTrace() []StackFrame {
	const maxFrames = 32
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(3, pcs) // Skip 3 frames (Callers, captureStackTrace, exception func)

	frames := make([]StackFrame, 0, n)
	for i := 0; i < n; i++ {
		pc := pcs[i]
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		file, line := fn.FileLine(pc)

		// Extract just the function name
		name := fn.Name()
		if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
			name = name[lastDot+1:]
		}

		// Extract just the filename
		if lastSlash := strings.LastIndex(file, "/"); lastSlash >= 0 {
			file = file[lastSlash+1:]
		}

		frames = append(frames, StackFrame{
			Function: name,
			File:     file,
			Line:     line,
			PC:       pc,
		})
	}

	return frames
}

// getCurrentTimestamp returns current timestamp for logging
func getCurrentTimestamp() string {
	// Simple timestamp implementation for bootstrap
	return "TIMESTAMP"
}

// Code generation support for exception handling

// ExceptionCodeGen generates assembly code for exception handling
type ExceptionCodeGen struct {
	UseTraps       bool // Use CPU traps for faster checking
	BoundsChecking bool // Enable bounds checking
	NullChecking   bool // Enable null pointer checking
	StackGuard     bool // Enable stack overflow protection
}

// NewExceptionCodeGen creates a new code generator for exceptions
func NewExceptionCodeGen() *ExceptionCodeGen {
	return &ExceptionCodeGen{
		UseTraps:       false, // Safe default for bootstrap
		BoundsChecking: true,
		NullChecking:   true,
		StackGuard:     false, // Complex to implement
	}
}

// EmitBoundsCheck generates assembly for bounds checking
func (ecg *ExceptionCodeGen) EmitBoundsCheck(index, length, arrayName string) string {
	if !ecg.BoundsChecking {
		return ""
	}

	var code strings.Builder
	code.WriteString(fmt.Sprintf("  ; bounds check for %s\n", arrayName))
	code.WriteString(fmt.Sprintf("  cmp %s, 0\n", index))
	code.WriteString("  jl .bounds_error\n")
	code.WriteString(fmt.Sprintf("  cmp %s, %s\n", index, length))
	code.WriteString("  jge .bounds_error\n")
	code.WriteString("  jmp .bounds_ok\n")
	code.WriteString(".bounds_error:\n")
	code.WriteString("  call panic_bounds_check\n")
	code.WriteString(".bounds_ok:\n")

	return code.String()
}

// EmitNullCheck generates assembly for null pointer checking
func (ecg *ExceptionCodeGen) EmitNullCheck(pointer, name string) string {
	if !ecg.NullChecking {
		return ""
	}

	var code strings.Builder
	code.WriteString(fmt.Sprintf("  ; null check for %s\n", name))
	code.WriteString(fmt.Sprintf("  test %s, %s\n", pointer, pointer))
	code.WriteString("  jz .null_error\n")
	code.WriteString("  jmp .null_ok\n")
	code.WriteString(".null_error:\n")
	code.WriteString("  call panic_null_pointer\n")
	code.WriteString(".null_ok:\n")

	return code.String()
}

// EmitDivisionCheck generates assembly for division by zero checking
func (ecg *ExceptionCodeGen) EmitDivisionCheck(divisor string) string {
	var code strings.Builder
	code.WriteString("  ; division by zero check\n")
	code.WriteString(fmt.Sprintf("  test %s, %s\n", divisor, divisor))
	code.WriteString("  jz .div_zero_error\n")
	code.WriteString("  jmp .div_ok\n")
	code.WriteString(".div_zero_error:\n")
	code.WriteString("  call panic_division_by_zero\n")
	code.WriteString(".div_ok:\n")

	return code.String()
}

// EmitPanicHandler generates assembly for panic handling
func (ecg *ExceptionCodeGen) EmitPanicHandler() string {
	var code strings.Builder

	code.WriteString("; Exception handler routines\n")
	code.WriteString("panic_bounds_check:\n")
	code.WriteString("  mov rcx, bounds_check_msg\n")
	code.WriteString("  call panic_with_message\n")
	code.WriteString("  ret\n\n")

	code.WriteString("panic_null_pointer:\n")
	code.WriteString("  mov rcx, null_pointer_msg\n")
	code.WriteString("  call panic_with_message\n")
	code.WriteString("  ret\n\n")

	code.WriteString("panic_division_by_zero:\n")
	code.WriteString("  mov rcx, div_zero_msg\n")
	code.WriteString("  call panic_with_message\n")
	code.WriteString("  ret\n\n")

	code.WriteString("panic_with_message:\n")
	code.WriteString("  ; rcx contains message pointer\n")
	code.WriteString("  ; Call runtime panic function\n")
	code.WriteString("  call runtime_panic\n")
	code.WriteString("  ; Never returns\n\n")

	// String constants
	code.WriteString("; Exception message constants\n")
	code.WriteString("bounds_check_msg: db 'Array bounds check failed', 0\n")
	code.WriteString("null_pointer_msg: db 'Null pointer access', 0\n")
	code.WriteString("div_zero_msg: db 'Division by zero', 0\n\n")

	return code.String()
}

// TryRecovery implements a basic try-catch mechanism (minimal)
type TryRecovery struct {
	handlers map[ExceptionKind]func(*Exception) bool
}

// NewTryRecovery creates a new recovery mechanism
func NewTryRecovery() *TryRecovery {
	return &TryRecovery{
		handlers: make(map[ExceptionKind]func(*Exception) bool),
	}
}

// AddHandler adds an exception handler for a specific kind
func (tr *TryRecovery) AddHandler(kind ExceptionKind, handler func(*Exception) bool) {
	tr.handlers[kind] = handler
}

// Try executes a function with exception recovery
func (tr *TryRecovery) Try(fn func()) (recovered bool) {
	// In a full implementation, this would use setjmp/longjmp or similar
	// For bootstrap, we use a simple defer/recover pattern
	defer func() {
		if r := recover(); r != nil {
			// Convert Go panic to Orizon exception
			exception := &Exception{
				Kind:       ExceptionPanic,
				Message:    fmt.Sprintf("%v", r),
				Location:   getCallerLocation(),
				StackTrace: captureStackTrace(),
			}

			// Try to handle with registered handlers
			if handler, exists := tr.handlers[exception.Kind]; exists {
				recovered = handler(exception)
			} else {
				// No handler found, re-panic
				currentHandler.HandleException(exception)
			}
		}
	}()

	fn()
	return false // No exception occurred
}
