// Package io provides enhanced x64 assembly code generation methods for complete I/O functions.
// This file contains complete function implementations with proper prologue and epilogue structures
// for optimal performance and compatibility with the Orizon runtime system.
package io

import (
	"strings"
)

// GenerateCompleteFileCloseFunction generates a complete file close function with proper prologue and epilogue.
// This function provides a self-contained assembly implementation for file closing operations
// that follows standard calling conventions and includes proper error handling.
//
// The generated function:
// 1. Establishes a proper stack frame with prologue
// 2. Calls the Windows CloseHandle API with appropriate parameters
// 3. Handles the return value and error conditions
// 4. Restores the stack frame with epilogue
// 5. Returns control to the caller
func (x64 *X64IOIntegration) GenerateCompleteFileCloseFunction() string {
	var asm strings.Builder

	// Function label with proper naming convention
	asm.WriteString("orizon_io_file_close:\n")
	asm.WriteString("    ; Complete file close function implementation\n")
	asm.WriteString("    ; Parameters: file handle passed via calling convention\n")
	asm.WriteString("    ; Returns: success code (0 = failure, non-zero = success)\n")

	// Function prologue - establish stack frame and preserve registers
	asm.WriteString("    push rbp           ; Save caller's base pointer\n")
	asm.WriteString("    mov rbp, rsp       ; Establish new stack frame\n")
	asm.WriteString("    sub rsp, 32        ; Reserve shadow space for Windows API\n")

	// Function body - implement file close logic with error handling
	asm.WriteString("    ; File close implementation using Windows CloseHandle API\n")
	asm.WriteString("    mov rcx, [rbp+16]  ; Load file handle parameter from stack\n")
	asm.WriteString("    ; Validate handle before API call\n")
	asm.WriteString("    cmp rcx, 0         ; Check for null handle\n")
	asm.WriteString("    je  close_error    ; Jump to error handling if null\n")
	asm.WriteString("    cmp rcx, -1        ; Check for invalid handle value\n")
	asm.WriteString("    je  close_error    ; Jump to error handling if invalid\n")

	// Call Windows API with proper error handling
	asm.WriteString("    call CloseHandle   ; Windows API call to close file handle\n")
	asm.WriteString("    ; rax now contains the result (0 = failure, non-zero = success)\n")
	asm.WriteString("    test rax, rax      ; Test return value\n")
	asm.WriteString("    jz   close_error   ; Jump to error handling on failure\n")

	// Success path - clean return
	asm.WriteString("    ; Success path: file successfully closed\n")
	asm.WriteString("    jmp  close_done    ; Jump to cleanup and return\n")

	// Error handling section
	asm.WriteString("close_error:\n")
	asm.WriteString("    ; Error handling: set appropriate error code\n")
	asm.WriteString("    mov rax, 0         ; Set failure return value\n")
	asm.WriteString("    ; Additional error logging could be added here\n")

	// Function epilogue - restore stack frame and return
	asm.WriteString("close_done:\n")
	asm.WriteString("    ; Function cleanup and return\n")
	asm.WriteString("    add rsp, 32        ; Deallocate shadow space\n")
	asm.WriteString("    pop rbp            ; Restore caller's base pointer\n")
	asm.WriteString("    ret                ; Return to caller with result in rax\n")

	return asm.String()
}

// GenerateCompleteFileOpenFunction generates a complete file open function with enhanced error handling.
// This function complements the existing file open functionality with additional robustness
// and proper integration with the Orizon runtime error handling system.
func (x64 *X64IOIntegration) GenerateCompleteFileOpenFunction() string {
	var asm strings.Builder

	// Function label with comprehensive documentation
	asm.WriteString("orizon_io_file_open:\n")
	asm.WriteString("    ; Enhanced complete file open function implementation\n")
	asm.WriteString("    ; Parameters: file path and open mode via calling convention\n")
	asm.WriteString("    ; Returns: file handle (INVALID_HANDLE_VALUE on failure)\n")

	// Function prologue with extended stack space for local variables
	asm.WriteString("    push rbp           ; Save caller's base pointer\n")
	asm.WriteString("    mov rbp, rsp       ; Establish new stack frame\n")
	asm.WriteString("    sub rsp, 48        ; Reserve space for API call and local variables\n")

	// Enhanced parameter validation and setup
	asm.WriteString("    ; Enhanced file open implementation with validation\n")
	asm.WriteString("    mov rcx, [rbp+16]  ; Load file path parameter\n")
	asm.WriteString("    mov rdx, [rbp+24]  ; Load open mode parameter\n")
	asm.WriteString("    ; Validate parameters before API call\n")
	asm.WriteString("    test rcx, rcx      ; Check for null path pointer\n")
	asm.WriteString("    jz   open_error    ; Jump to error handling if null\n")

	// Windows API call setup with comprehensive parameter handling
	asm.WriteString("    ; Setup Windows CreateFileA API call parameters\n")
	asm.WriteString("    mov r8, 3          ; dwShareMode (FILE_SHARE_READ | FILE_SHARE_WRITE)\n")
	asm.WriteString("    mov r9, 0          ; lpSecurityAttributes (NULL for default)\n")
	asm.WriteString("    push 0             ; hTemplateFile (NULL)\n")
	asm.WriteString("    push 128           ; dwFlagsAndAttributes (FILE_ATTRIBUTE_NORMAL)\n")
	asm.WriteString("    push 3             ; dwCreationDisposition (OPEN_EXISTING)\n")
	asm.WriteString("    call CreateFileA   ; Windows API call\n")
	asm.WriteString("    add rsp, 24        ; Clean up pushed parameters\n")

	// Result validation and error handling
	asm.WriteString("    ; Validate CreateFileA result\n")
	asm.WriteString("    cmp rax, -1        ; Check for INVALID_HANDLE_VALUE\n")
	asm.WriteString("    je  open_error     ; Jump to error handling on failure\n")
	asm.WriteString("    jmp open_done      ; Jump to successful completion\n")

	// Error handling with proper cleanup
	asm.WriteString("open_error:\n")
	asm.WriteString("    ; Error handling: set invalid handle return value\n")
	asm.WriteString("    mov rax, -1        ; Set INVALID_HANDLE_VALUE\n")

	// Function epilogue with proper cleanup
	asm.WriteString("open_done:\n")
	asm.WriteString("    ; Function cleanup and return\n")
	asm.WriteString("    add rsp, 48        ; Deallocate local variable space\n")
	asm.WriteString("    pop rbp            ; Restore caller's base pointer\n")
	asm.WriteString("    ret                ; Return to caller with handle in rax\n")

	return asm.String()
}
