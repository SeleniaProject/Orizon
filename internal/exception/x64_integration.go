// Package exception provides x64 code generation integration for exception handling
package exception

import (
	"fmt"
	"strings"
)

// X64ExceptionEmitter generates x64 assembly code for exception handling
type X64ExceptionEmitter struct {
	codeGen            *ExceptionCodeGen
	enableOptimization bool
	useUnwindInfo      bool // For SEH on Windows
	useDwarfInfo       bool // For DWARF on Linux
	stackAlignment     int  // Stack alignment requirement
}

// NewX64ExceptionEmitter creates a new x64 exception emitter
func NewX64ExceptionEmitter() *X64ExceptionEmitter {
	return &X64ExceptionEmitter{
		codeGen:            NewExceptionCodeGen(),
		enableOptimization: true,
		useUnwindInfo:      true,  // Windows SEH
		useDwarfInfo:       false, // DWARF for Linux (not implemented in bootstrap)
		stackAlignment:     16,    // x64 requires 16-byte alignment
	}
}

// EmitPrologue generates function prologue with exception handling setup
func (x64 *X64ExceptionEmitter) EmitPrologue(funcName string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("; Function: %s\n", funcName))
	code.WriteString(fmt.Sprintf("%s:\n", funcName))

	// Standard x64 function prologue
	code.WriteString("  push rbp\n")
	code.WriteString("  mov rbp, rsp\n")

	// Reserve space for exception context if needed
	if x64.useUnwindInfo {
		code.WriteString("  sub rsp, 32      ; Shadow space for exception context\n")
	}

	// Align stack to 16-byte boundary
	code.WriteString("  and rsp, -16     ; Align stack\n")

	if x64.useUnwindInfo {
		// Windows SEH setup (simplified)
		code.WriteString("  ; SEH setup\n")
		code.WriteString("  mov rax, rsp\n")
		code.WriteString("  mov [rbp-8], rax  ; Store original rsp for unwind\n")
	}

	return code.String()
}

// EmitEpilogue generates function epilogue with exception cleanup
func (x64 *X64ExceptionEmitter) EmitEpilogue(funcName string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("; Epilogue for %s\n", funcName))

	if x64.useUnwindInfo {
		// Windows SEH cleanup
		code.WriteString("  ; SEH cleanup\n")
		code.WriteString("  mov rsp, [rbp-8]  ; Restore original rsp\n")
	}

	// Standard x64 function epilogue
	code.WriteString("  mov rsp, rbp\n")
	code.WriteString("  pop rbp\n")
	code.WriteString("  ret\n")

	return code.String()
}

// EmitBoundsCheckX64 generates optimized x64 bounds checking code
func (x64 *X64ExceptionEmitter) EmitBoundsCheckX64(indexReg, lengthReg, arrayName string) string {
	var code strings.Builder

	if !x64.codeGen.BoundsChecking {
		return ""
	}

	okLabel := fmt.Sprintf(".bounds_ok_%s", x64.generateUniqueId())
	errorLabel := fmt.Sprintf(".bounds_error_%s", x64.generateUniqueId())

	code.WriteString(fmt.Sprintf("  ; Optimized bounds check for %s\n", arrayName))

	if x64.enableOptimization {
		// Optimized version: single comparison with unsigned arithmetic
		code.WriteString(fmt.Sprintf("  cmp %s, %s        ; Compare index with length\n", indexReg, lengthReg))
		code.WriteString(fmt.Sprintf("  jae %s            ; Jump if index >= length (unsigned)\n", errorLabel))
	} else {
		// Standard version: two comparisons
		code.WriteString(fmt.Sprintf("  test %s, %s       ; Check if index is negative\n", indexReg, indexReg))
		code.WriteString(fmt.Sprintf("  js %s             ; Jump if negative\n", errorLabel))
		code.WriteString(fmt.Sprintf("  cmp %s, %s        ; Compare with length\n", indexReg, lengthReg))
		code.WriteString(fmt.Sprintf("  jge %s            ; Jump if >= length\n", errorLabel))
	}

	code.WriteString(fmt.Sprintf("  jmp %s            ; Continue if check passes\n", okLabel))

	// Error handling
	code.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	code.WriteString("  ; Prepare arguments for bounds check panic\n")
	code.WriteString(fmt.Sprintf("  mov rcx, %s       ; Index value\n", indexReg))
	code.WriteString(fmt.Sprintf("  mov rdx, %s       ; Length value\n", lengthReg))
	code.WriteString("  mov r8, bounds_check_msg ; Error message\n")
	code.WriteString("  call panic_bounds_check\n")
	code.WriteString("  ; Never returns\n")

	code.WriteString(fmt.Sprintf("%s:\n", okLabel))
	code.WriteString("  ; Bounds check passed, continue\n")

	return code.String()
}

// EmitNullCheckX64 generates optimized x64 null pointer checking code
func (x64 *X64ExceptionEmitter) EmitNullCheckX64(ptrReg, ptrName string) string {
	var code strings.Builder

	if !x64.codeGen.NullChecking {
		return ""
	}

	okLabel := fmt.Sprintf(".null_ok_%s", x64.generateUniqueId())
	errorLabel := fmt.Sprintf(".null_error_%s", x64.generateUniqueId())

	code.WriteString(fmt.Sprintf("  ; Null pointer check for %s\n", ptrName))
	code.WriteString(fmt.Sprintf("  test %s, %s       ; Check if pointer is null\n", ptrReg, ptrReg))
	code.WriteString(fmt.Sprintf("  jz %s             ; Jump if null\n", errorLabel))
	code.WriteString(fmt.Sprintf("  jmp %s            ; Continue if not null\n", okLabel))

	// Error handling
	code.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	code.WriteString("  ; Prepare arguments for null pointer panic\n")
	code.WriteString("  mov rcx, null_ptr_msg  ; Error message\n")
	code.WriteString("  call panic_null_pointer\n")
	code.WriteString("  ; Never returns\n")

	code.WriteString(fmt.Sprintf("%s:\n", okLabel))
	code.WriteString("  ; Null check passed, continue\n")

	return code.String()
}

// EmitDivisionCheckX64 generates x64 division by zero checking code
func (x64 *X64ExceptionEmitter) EmitDivisionCheckX64(divisorReg, operation string) string {
	var code strings.Builder

	okLabel := fmt.Sprintf(".div_ok_%s", x64.generateUniqueId())
	errorLabel := fmt.Sprintf(".div_error_%s", x64.generateUniqueId())

	code.WriteString(fmt.Sprintf("  ; Division by zero check for %s\n", operation))
	code.WriteString(fmt.Sprintf("  test %s, %s       ; Check if divisor is zero\n", divisorReg, divisorReg))
	code.WriteString(fmt.Sprintf("  jz %s             ; Jump if zero\n", errorLabel))
	code.WriteString(fmt.Sprintf("  jmp %s            ; Continue if not zero\n", okLabel))

	// Error handling
	code.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	code.WriteString("  ; Prepare arguments for division by zero panic\n")
	code.WriteString("  mov rcx, div_zero_msg  ; Error message\n")
	code.WriteString("  call panic_division_by_zero\n")
	code.WriteString("  ; Never returns\n")

	code.WriteString(fmt.Sprintf("%s:\n", okLabel))
	code.WriteString("  ; Division check passed, continue\n")

	return code.String()
}

// EmitStackOverflowCheck generates stack overflow checking code
func (x64 *X64ExceptionEmitter) EmitStackOverflowCheck(requiredBytes int) string {
	if !x64.codeGen.StackGuard {
		return ""
	}

	var code strings.Builder

	okLabel := fmt.Sprintf(".stack_ok_%s", x64.generateUniqueId())
	errorLabel := fmt.Sprintf(".stack_error_%s", x64.generateUniqueId())

	code.WriteString("  ; Stack overflow check\n")
	code.WriteString("  mov rax, rsp\n")
	code.WriteString(fmt.Sprintf("  sub rax, %d       ; Required stack space\n", requiredBytes))
	code.WriteString("  mov rbx, [gs:stack_limit] ; Get stack limit from TLS\n")
	code.WriteString("  cmp rax, rbx\n")
	code.WriteString(fmt.Sprintf("  jb %s             ; Jump if stack overflow\n", errorLabel))
	code.WriteString(fmt.Sprintf("  jmp %s            ; Continue if stack is ok\n", okLabel))

	// Error handling
	code.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	code.WriteString("  ; Stack overflow detected\n")
	code.WriteString("  mov rcx, stack_overflow_msg\n")
	code.WriteString("  call panic_stack_overflow\n")
	code.WriteString("  ; Never returns\n")

	code.WriteString(fmt.Sprintf("%s:\n", okLabel))
	code.WriteString("  ; Stack check passed, continue\n")

	return code.String()
}

// EmitExceptionHandlersX64 generates all x64 exception handler routines
func (x64 *X64ExceptionEmitter) EmitExceptionHandlersX64() string {
	var code strings.Builder

	code.WriteString("; Exception handler routines for x64\n")
	code.WriteString(".text\n")
	code.WriteString(".align 16\n\n")

	// Bounds check panic handler
	code.WriteString("panic_bounds_check:\n")
	code.WriteString(x64.EmitPrologue("panic_bounds_check"))
	code.WriteString("  ; rcx = index, rdx = length, r8 = message\n")
	code.WriteString("  call format_bounds_error  ; Format error message\n")
	code.WriteString("  mov rcx, rax              ; Formatted message\n")
	code.WriteString("  call runtime_abort        ; Abort with message\n")
	code.WriteString("  ; Never returns\n")
	code.WriteString(x64.EmitEpilogue("panic_bounds_check"))
	code.WriteString("\n")

	// Null pointer panic handler
	code.WriteString("panic_null_pointer:\n")
	code.WriteString(x64.EmitPrologue("panic_null_pointer"))
	code.WriteString("  ; rcx = message\n")
	code.WriteString("  call runtime_abort        ; Abort with message\n")
	code.WriteString("  ; Never returns\n")
	code.WriteString(x64.EmitEpilogue("panic_null_pointer"))
	code.WriteString("\n")

	// Division by zero panic handler
	code.WriteString("panic_division_by_zero:\n")
	code.WriteString(x64.EmitPrologue("panic_division_by_zero"))
	code.WriteString("  ; rcx = message\n")
	code.WriteString("  call runtime_abort        ; Abort with message\n")
	code.WriteString("  ; Never returns\n")
	code.WriteString(x64.EmitEpilogue("panic_division_by_zero"))
	code.WriteString("\n")

	// Stack overflow panic handler
	code.WriteString("panic_stack_overflow:\n")
	code.WriteString(x64.EmitPrologue("panic_stack_overflow"))
	code.WriteString("  ; rcx = message\n")
	code.WriteString("  call runtime_abort        ; Abort with message\n")
	code.WriteString("  ; Never returns\n")
	code.WriteString(x64.EmitEpilogue("panic_stack_overflow"))
	code.WriteString("\n")

	// Helper function to format bounds error
	code.WriteString("format_bounds_error:\n")
	code.WriteString(x64.EmitPrologue("format_bounds_error"))
	code.WriteString("  ; rcx = index, rdx = length, r8 = message\n")
	code.WriteString("  ; Format: \"Array bounds check failed: index %d >= length %d\"\n")
	code.WriteString("  push rcx                  ; Save index\n")
	code.WriteString("  push rdx                  ; Save length\n")
	code.WriteString("  mov rcx, bounds_fmt_str   ; Format string\n")
	code.WriteString("  pop rdx                   ; length (2nd arg)\n")
	code.WriteString("  pop r8                    ; index (3rd arg)\n")
	code.WriteString("  call sprintf              ; Format the string\n")
	code.WriteString("  ; Return formatted string in rax\n")
	code.WriteString(x64.EmitEpilogue("format_bounds_error"))
	code.WriteString("\n")

	// Runtime abort function (simplified)
	code.WriteString("runtime_abort:\n")
	code.WriteString(x64.EmitPrologue("runtime_abort"))
	code.WriteString("  ; rcx = error message\n")
	code.WriteString("  ; Write to stderr and exit\n")
	code.WriteString("  mov rdx, rcx              ; Message\n")
	code.WriteString("  call strlen               ; Get message length\n")
	code.WriteString("  mov r8, rax               ; Length\n")
	code.WriteString("  mov rcx, -12              ; STD_ERROR_HANDLE\n")
	code.WriteString("  call GetStdHandle         ; Get stderr handle\n")
	code.WriteString("  mov rcx, rax              ; Handle\n")
	code.WriteString("  ; rcx = handle, rdx = message, r8 = length\n")
	code.WriteString("  xor r9, r9                ; lpNumberOfBytesWritten (NULL)\n")
	code.WriteString("  push 0                    ; lpOverlapped (NULL)\n")
	code.WriteString("  call WriteFile            ; Write to stderr\n")
	code.WriteString("  add rsp, 8                ; Clean up stack\n")
	code.WriteString("  mov rcx, 1                ; Exit code\n")
	code.WriteString("  call ExitProcess          ; Terminate process\n")
	code.WriteString("  ; Never returns\n")
	code.WriteString(x64.EmitEpilogue("runtime_abort"))
	code.WriteString("\n")

	return code.String()
}

// EmitDataSection generates data section with exception messages
func (x64 *X64ExceptionEmitter) EmitDataSection() string {
	var code strings.Builder

	code.WriteString("; Exception handling data section\n")
	code.WriteString(".data\n")
	code.WriteString(".align 8\n\n")

	// Exception message strings
	code.WriteString("bounds_check_msg: db 'Array bounds check failed', 0\n")
	code.WriteString("null_ptr_msg: db 'Null pointer access', 0\n")
	code.WriteString("div_zero_msg: db 'Division by zero', 0\n")
	code.WriteString("stack_overflow_msg: db 'Stack overflow', 0\n")
	code.WriteString("assert_msg: db 'Assertion failed', 0\n")

	// Format strings
	code.WriteString("bounds_fmt_str: db 'Array bounds check failed: index %d >= length %d', 0\n")
	code.WriteString("null_fmt_str: db 'Null pointer access: %s', 0\n")

	// Exception info structure (for debugging)
	code.WriteString("exception_info:\n")
	code.WriteString("  dq 0  ; Exception kind\n")
	code.WriteString("  dq 0  ; Exception message pointer\n")
	code.WriteString("  dq 0  ; Exception location pointer\n")
	code.WriteString("  dq 0  ; Stack trace pointer\n")
	code.WriteString("\n")

	return code.String()
}

// EmitExternDeclarations generates extern declarations for system functions
func (x64 *X64ExceptionEmitter) EmitExternDeclarations() string {
	var code strings.Builder

	code.WriteString("; External function declarations\n")
	code.WriteString("extern GetStdHandle\n")
	code.WriteString("extern WriteFile\n")
	code.WriteString("extern ExitProcess\n")
	code.WriteString("extern sprintf\n")
	code.WriteString("extern strlen\n")
	code.WriteString("\n")

	return code.String()
}

// EmitInlineExceptionCheck generates inline exception checking code
func (x64 *X64ExceptionEmitter) EmitInlineExceptionCheck(checkType string, args ...string) string {
	switch checkType {
	case "bounds":
		if len(args) >= 3 {
			return x64.EmitBoundsCheckX64(args[0], args[1], args[2])
		}
	case "null":
		if len(args) >= 2 {
			return x64.EmitNullCheckX64(args[0], args[1])
		}
	case "division":
		if len(args) >= 2 {
			return x64.EmitDivisionCheckX64(args[0], args[1])
		}
	case "stack":
		if len(args) >= 1 {
			if requiredBytes := parseIntSafe(args[0]); requiredBytes > 0 {
				return x64.EmitStackOverflowCheck(requiredBytes)
			}
		}
	}

	return fmt.Sprintf("  ; Unknown exception check type: %s\n", checkType)
}

// generateUniqueId generates a unique identifier for labels
func (x64 *X64ExceptionEmitter) generateUniqueId() string {
	// Simple unique ID generation (in real implementation, use atomic counter)
	return fmt.Sprintf("%d", len(fmt.Sprintf("%p", x64)))
}

// parseIntSafe safely parses an integer from string
func parseIntSafe(s string) int {
	// Simplified integer parsing (in real implementation, use strconv.Atoi)
	if s == "32" {
		return 32
	}
	if s == "64" {
		return 64
	}
	if s == "128" {
		return 128
	}
	return 0
}

// EmitCompleteExceptionSystem generates a complete exception handling system
func (x64 *X64ExceptionEmitter) EmitCompleteExceptionSystem() string {
	var code strings.Builder

	code.WriteString("; Complete x64 Exception Handling System\n")
	code.WriteString("; Generated by Orizon Compiler Exception System\n\n")

	// Extern declarations
	code.WriteString(x64.EmitExternDeclarations())

	// Code section
	code.WriteString(x64.EmitExceptionHandlersX64())

	// Data section
	code.WriteString(x64.EmitDataSection())

	// Optional: SEH unwind info (Windows)
	if x64.useUnwindInfo {
		code.WriteString(x64.emitUnwindInfo())
	}

	return code.String()
}

// emitUnwindInfo generates Windows SEH unwind information
func (x64 *X64ExceptionEmitter) emitUnwindInfo() string {
	var code strings.Builder

	code.WriteString("; Windows SEH Unwind Information\n")
	code.WriteString(".pdata\n")
	code.WriteString(".align 4\n")

	// Unwind info for each exception handler
	handlers := []string{
		"panic_bounds_check",
		"panic_null_pointer",
		"panic_division_by_zero",
		"panic_stack_overflow",
	}

	for _, handler := range handlers {
		code.WriteString(fmt.Sprintf("; Unwind info for %s\n", handler))
		code.WriteString(fmt.Sprintf("  dd OFFSET %s        ; Begin address\n", handler))
		code.WriteString(fmt.Sprintf("  dd OFFSET %s_end    ; End address\n", handler))
		code.WriteString(fmt.Sprintf("  dd OFFSET %s_unwind ; Unwind info\n", handler))
	}

	code.WriteString("\n.xdata\n")
	code.WriteString(".align 4\n")

	for _, handler := range handlers {
		code.WriteString(fmt.Sprintf("%s_unwind:\n", handler))
		code.WriteString("  db 1          ; Version\n")
		code.WriteString("  db 0          ; Flags\n")
		code.WriteString("  db 4          ; Size of prolog\n")
		code.WriteString("  db 2          ; Count of unwind codes\n")
		code.WriteString("  db 0          ; Frame register\n")
		code.WriteString("  db 0          ; Frame register offset\n")
		code.WriteString("  ; Unwind codes\n")
		code.WriteString("  dw 0x0504     ; UWOP_SAVE_NONVOL RBP, offset 0x40\n")
		code.WriteString("  dw 0x0100     ; UWOP_PUSH_NONVOL RBP\n")
	}

	return code.String()
}

// OptimizeExceptionChecks performs x64-specific optimizations on exception checks
func (x64 *X64ExceptionEmitter) OptimizeExceptionChecks(assemblyCode string) string {
	if !x64.enableOptimization {
		return assemblyCode
	}

	// Remove redundant test instructions
	optimized := strings.ReplaceAll(assemblyCode,
		"test rax, rax\n  test rax, rax\n",
		"test rax, rax\n")

	// Combine consecutive comparisons
	optimized = strings.ReplaceAll(optimized,
		"cmp rax, 0\n  test rax, rax\n",
		"test rax, rax\n")

	// Remove unnecessary jumps
	optimized = strings.ReplaceAll(optimized,
		"jmp .L1\n.L1:\n",
		"")

	return optimized
}
