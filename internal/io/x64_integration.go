// Package io provides x64 assembly code generation for I/O operations
// in the Orizon compiler. This enables direct machine code generation
// for file I/O, console I/O, and threading primitives.
package io

import (
	"fmt"
	"strings"
)

// X64IOIntegration provides x64 assembly code generation for I/O operations
type X64IOIntegration struct {
	labelCounter int
}

// NewX64IOIntegration creates a new x64 I/O integration
func NewX64IOIntegration() *X64IOIntegration {
	return &X64IOIntegration{
		labelCounter: 0,
	}
}

// generateLabel generates a unique label for assembly code
func (x64 *X64IOIntegration) generateLabel(prefix string) string {
	x64.labelCounter++
	return fmt.Sprintf("%s_%d", prefix, x64.labelCounter)
}

// File I/O x64 Operations

// GenerateFileOpenAsm generates x64 assembly for opening a file
func (x64 *X64IOIntegration) GenerateFileOpenAsm(pathReg, modeReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; File open operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: path in %s, mode in %s\n", pathReg, modeReg))
	asm.WriteString(fmt.Sprintf("    ; Result: handle in %s\n", resultReg))

	// Windows system call for CreateFileA
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; lpFileName (path)\n", pathReg))
	asm.WriteString(fmt.Sprintf("    mov rdx, %s        ; dwDesiredAccess (mode converted)\n", modeReg))
	asm.WriteString(fmt.Sprintf("    mov r8, 3          ; dwShareMode (FILE_SHARE_READ | FILE_SHARE_WRITE)\n"))
	asm.WriteString(fmt.Sprintf("    mov r9, 0          ; lpSecurityAttributes (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    push 0             ; hTemplateFile (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    push 128          ; dwFlagsAndAttributes (FILE_ATTRIBUTE_NORMAL)\n"))
	asm.WriteString(fmt.Sprintf("    push 3             ; dwCreationDisposition (OPEN_EXISTING)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call CreateFileA   ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 56        ; Clean up stack (32 + 24)\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store file handle\n", resultReg))

	return asm.String()
}

// GenerateFileCloseAsm generates x64 assembly for closing a file
func (x64 *X64IOIntegration) GenerateFileCloseAsm(handleReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; File close operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameter: handle in %s\n", handleReg))
	asm.WriteString(fmt.Sprintf("    ; Result: success code in %s\n", resultReg))

	// Windows system call for CloseHandle
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; hObject (file handle)\n", handleReg))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call CloseHandle   ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store result (0 = failure, non-zero = success)\n", resultReg))

	return asm.String()
}

// GenerateFileReadAsm generates x64 assembly for reading from a file
func (x64 *X64IOIntegration) GenerateFileReadAsm(handleReg, bufferReg, sizeReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; File read operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: handle in %s, buffer in %s, size in %s\n", handleReg, bufferReg, sizeReg))
	asm.WriteString(fmt.Sprintf("    ; Result: bytes read in %s\n", resultReg))

	// Allocate space for bytesRead variable on stack
	asm.WriteString(fmt.Sprintf("    sub rsp, 8          ; Space for bytesRead\n"))

	// Windows system call for ReadFile
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; hFile (file handle)\n", handleReg))
	asm.WriteString(fmt.Sprintf("    mov rdx, %s        ; lpBuffer\n", bufferReg))
	asm.WriteString(fmt.Sprintf("    mov r8, %s         ; nNumberOfBytesToRead\n", sizeReg))
	asm.WriteString(fmt.Sprintf("    lea r9, [rsp]      ; lpNumberOfBytesRead (pointer to stack)\n"))
	asm.WriteString(fmt.Sprintf("    push 0             ; lpOverlapped (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call ReadFile      ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 40        ; Clean up stack (32 + 8)\n"))

	// Get bytes read from stack
	asm.WriteString(fmt.Sprintf("    mov %s, [rsp]      ; Load bytes read\n", resultReg))
	asm.WriteString(fmt.Sprintf("    add rsp, 8          ; Clean up bytesRead space\n"))

	return asm.String()
}

// GenerateFileWriteAsm generates x64 assembly for writing to a file
func (x64 *X64IOIntegration) GenerateFileWriteAsm(handleReg, bufferReg, sizeReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; File write operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: handle in %s, buffer in %s, size in %s\n", handleReg, bufferReg, sizeReg))
	asm.WriteString(fmt.Sprintf("    ; Result: bytes written in %s\n", resultReg))

	// Allocate space for bytesWritten variable on stack
	asm.WriteString(fmt.Sprintf("    sub rsp, 8          ; Space for bytesWritten\n"))

	// Windows system call for WriteFile
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; hFile (file handle)\n", handleReg))
	asm.WriteString(fmt.Sprintf("    mov rdx, %s        ; lpBuffer\n", bufferReg))
	asm.WriteString(fmt.Sprintf("    mov r8, %s         ; nNumberOfBytesToWrite\n", sizeReg))
	asm.WriteString(fmt.Sprintf("    lea r9, [rsp]      ; lpNumberOfBytesWritten (pointer to stack)\n"))
	asm.WriteString(fmt.Sprintf("    push 0             ; lpOverlapped (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call WriteFile     ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 40        ; Clean up stack (32 + 8)\n"))

	// Get bytes written from stack
	asm.WriteString(fmt.Sprintf("    mov %s, [rsp]      ; Load bytes written\n", resultReg))
	asm.WriteString(fmt.Sprintf("    add rsp, 8          ; Clean up bytesWritten space\n"))

	return asm.String()
}

// Console I/O x64 Operations

// GenerateConsoleWriteAsm generates x64 assembly for writing to console
func (x64 *X64IOIntegration) GenerateConsoleWriteAsm(streamReg, bufferReg, sizeReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Console write operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: stream in %s, buffer in %s, size in %s\n", streamReg, bufferReg, sizeReg))
	asm.WriteString(fmt.Sprintf("    ; Result: bytes written in %s\n", resultReg))

	// Get console handle based on stream (stdout = -11, stderr = -12)
	stdoutLabel := x64.generateLabel("stdout")
	endLabel := x64.generateLabel("write_end")

	asm.WriteString(fmt.Sprintf("    cmp %s, 1          ; Check if stdout (1)\n", streamReg))
	asm.WriteString(fmt.Sprintf("    je %s              ; Jump to stdout handling\n", stdoutLabel))
	asm.WriteString(fmt.Sprintf("    mov rcx, -12       ; stderr handle\n"))
	asm.WriteString(fmt.Sprintf("    jmp %s             ; Skip stdout handling\n", endLabel))
	asm.WriteString(fmt.Sprintf("%s:\n", stdoutLabel))
	asm.WriteString(fmt.Sprintf("    mov rcx, -11       ; stdout handle\n"))
	asm.WriteString(fmt.Sprintf("%s:\n", endLabel))

	// Get standard handle
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call GetStdHandle  ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov rcx, rax       ; Console handle in rcx\n"))

	// Allocate space for bytesWritten
	asm.WriteString(fmt.Sprintf("    sub rsp, 8          ; Space for bytesWritten\n"))

	// WriteConsole call
	asm.WriteString(fmt.Sprintf("    mov rdx, %s        ; lpBuffer\n", bufferReg))
	asm.WriteString(fmt.Sprintf("    mov r8, %s         ; nNumberOfCharsToWrite\n", sizeReg))
	asm.WriteString(fmt.Sprintf("    lea r9, [rsp]      ; lpNumberOfCharsWritten\n"))
	asm.WriteString(fmt.Sprintf("    push 0             ; lpReserved (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call WriteConsoleA ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 40        ; Clean up stack\n"))

	// Get bytes written
	asm.WriteString(fmt.Sprintf("    mov %s, [rsp]      ; Load bytes written\n", resultReg))
	asm.WriteString(fmt.Sprintf("    add rsp, 8          ; Clean up bytesWritten space\n"))

	return asm.String()
}

// GenerateConsoleReadAsm generates x64 assembly for reading from console
func (x64 *X64IOIntegration) GenerateConsoleReadAsm(bufferReg, sizeReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Console read operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: buffer in %s, size in %s\n", bufferReg, sizeReg))
	asm.WriteString(fmt.Sprintf("    ; Result: bytes read in %s\n", resultReg))

	// Get stdin handle
	asm.WriteString(fmt.Sprintf("    mov rcx, -10       ; stdin handle\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call GetStdHandle  ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov rcx, rax       ; Console handle in rcx\n"))

	// Allocate space for bytesRead
	asm.WriteString(fmt.Sprintf("    sub rsp, 8          ; Space for bytesRead\n"))

	// ReadConsole call
	asm.WriteString(fmt.Sprintf("    mov rdx, %s        ; lpBuffer\n", bufferReg))
	asm.WriteString(fmt.Sprintf("    mov r8, %s         ; nNumberOfCharsToRead\n", sizeReg))
	asm.WriteString(fmt.Sprintf("    lea r9, [rsp]      ; lpNumberOfCharsRead\n"))
	asm.WriteString(fmt.Sprintf("    push 0             ; pInputControl (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call ReadConsoleA  ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 40        ; Clean up stack\n"))

	// Get bytes read
	asm.WriteString(fmt.Sprintf("    mov %s, [rsp]      ; Load bytes read\n", resultReg))
	asm.WriteString(fmt.Sprintf("    add rsp, 8          ; Clean up bytesRead space\n"))

	return asm.String()
}

// Threading x64 Operations

// GenerateThreadCreateAsm generates x64 assembly for creating a thread
func (x64 *X64IOIntegration) GenerateThreadCreateAsm(functionReg, dataReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Thread create operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: function in %s, data in %s\n", functionReg, dataReg))
	asm.WriteString(fmt.Sprintf("    ; Result: thread handle in %s\n", resultReg))

	// Allocate space for thread ID
	asm.WriteString(fmt.Sprintf("    sub rsp, 8          ; Space for thread ID\n"))

	// CreateThread call
	asm.WriteString(fmt.Sprintf("    mov rcx, 0         ; lpThreadAttributes (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    mov rdx, 0         ; dwStackSize (default)\n"))
	asm.WriteString(fmt.Sprintf("    mov r8, %s         ; lpStartAddress (function)\n", functionReg))
	asm.WriteString(fmt.Sprintf("    mov r9, %s         ; lpParameter (data)\n", dataReg))
	asm.WriteString(fmt.Sprintf("    push 0             ; dwCreationFlags (0 = run immediately)\n"))
	asm.WriteString(fmt.Sprintf("    lea rax, [rsp+8]   ; lpThreadId (pointer to stack space)\n"))
	asm.WriteString(fmt.Sprintf("    push rax           ; Push thread ID pointer\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call CreateThread  ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 48        ; Clean up stack (32 + 16)\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store thread handle\n", resultReg))
	asm.WriteString(fmt.Sprintf("    add rsp, 8          ; Clean up thread ID space\n"))

	return asm.String()
}

// GenerateThreadJoinAsm generates x64 assembly for joining a thread
func (x64 *X64IOIntegration) GenerateThreadJoinAsm(handleReg, timeoutReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Thread join operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameters: handle in %s, timeout in %s\n", handleReg, timeoutReg))
	asm.WriteString(fmt.Sprintf("    ; Result: wait result in %s\n", resultReg))

	// WaitForSingleObject call
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; hHandle (thread handle)\n", handleReg))
	asm.WriteString(fmt.Sprintf("    mov rdx, %s        ; dwMilliseconds (timeout)\n", timeoutReg))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call WaitForSingleObject ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store wait result\n", resultReg))

	return asm.String()
}

// Synchronization x64 Operations

// GenerateMutexCreateAsm generates x64 assembly for creating a mutex
func (x64 *X64IOIntegration) GenerateMutexCreateAsm(resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Mutex create operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Result: mutex handle in %s\n", resultReg))

	// CreateMutex call
	asm.WriteString(fmt.Sprintf("    mov rcx, 0         ; lpMutexAttributes (NULL)\n"))
	asm.WriteString(fmt.Sprintf("    mov rdx, 0         ; bInitialOwner (FALSE)\n"))
	asm.WriteString(fmt.Sprintf("    mov r8, 0          ; lpName (NULL - unnamed mutex)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call CreateMutexA  ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store mutex handle\n", resultReg))

	return asm.String()
}

// GenerateMutexLockAsm generates x64 assembly for locking a mutex
func (x64 *X64IOIntegration) GenerateMutexLockAsm(mutexReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Mutex lock operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameter: mutex handle in %s\n", mutexReg))
	asm.WriteString(fmt.Sprintf("    ; Result: wait result in %s\n", resultReg))

	// WaitForSingleObject call (infinite timeout for lock)
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; hHandle (mutex handle)\n", mutexReg))
	asm.WriteString(fmt.Sprintf("    mov rdx, 0xFFFFFFFF ; dwMilliseconds (INFINITE)\n"))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call WaitForSingleObject ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store wait result\n", resultReg))

	return asm.String()
}

// GenerateMutexUnlockAsm generates x64 assembly for unlocking a mutex
func (x64 *X64IOIntegration) GenerateMutexUnlockAsm(mutexReg, resultReg string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Mutex unlock operation\n"))
	asm.WriteString(fmt.Sprintf("    ; Parameter: mutex handle in %s\n", mutexReg))
	asm.WriteString(fmt.Sprintf("    ; Result: release result in %s\n", resultReg))

	// ReleaseMutex call
	asm.WriteString(fmt.Sprintf("    mov rcx, %s        ; hMutex (mutex handle)\n", mutexReg))
	asm.WriteString(fmt.Sprintf("    sub rsp, 32        ; Shadow space\n"))
	asm.WriteString(fmt.Sprintf("    call ReleaseMutex  ; Windows API call\n"))
	asm.WriteString(fmt.Sprintf("    add rsp, 32        ; Clean up stack\n"))
	asm.WriteString(fmt.Sprintf("    mov %s, rax        ; Store release result\n", resultReg))

	return asm.String()
}

// Utility Functions

// GenerateFunctionPrologue generates standard function prologue
func (x64 *X64IOIntegration) GenerateFunctionPrologue(functionName string, stackSize int) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("; Function: %s\n", functionName))
	asm.WriteString(fmt.Sprintf("%s:\n", functionName))
	asm.WriteString(fmt.Sprintf("    push rbp           ; Save base pointer\n"))
	asm.WriteString(fmt.Sprintf("    mov rbp, rsp       ; Set up base pointer\n"))
	if stackSize > 0 {
		asm.WriteString(fmt.Sprintf("    sub rsp, %d        ; Allocate stack space\n", stackSize))
	}

	return asm.String()
}

// GenerateFunctionEpilogue generates standard function epilogue
func (x64 *X64IOIntegration) GenerateFunctionEpilogue(stackSize int) string {
	var asm strings.Builder

	if stackSize > 0 {
		asm.WriteString(fmt.Sprintf("    add rsp, %d        ; Deallocate stack space\n", stackSize))
	}
	asm.WriteString(fmt.Sprintf("    pop rbp            ; Restore base pointer\n"))
	asm.WriteString(fmt.Sprintf("    ret                ; Return to caller\n"))

	return asm.String()
}

// GenerateErrorHandling generates error handling code
func (x64 *X64IOIntegration) GenerateErrorHandling(errorLabel string) string {
	var asm strings.Builder

	asm.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	asm.WriteString(fmt.Sprintf("    ; Error handling\n"))
	asm.WriteString(fmt.Sprintf("    mov rax, -1        ; Error return value\n"))
	asm.WriteString(fmt.Sprintf("    ret                ; Return with error\n"))

	return asm.String()
}

// GetRequiredExternals returns the list of external Windows API functions needed
func (x64 *X64IOIntegration) GetRequiredExternals() []string {
	return []string{
		"CreateFileA",
		"CloseHandle",
		"ReadFile",
		"WriteFile",
		"GetStdHandle",
		"WriteConsoleA",
		"ReadConsoleA",
		"CreateThread",
		"WaitForSingleObject",
		"CreateMutexA",
		"ReleaseMutex",
	}
}

// GenerateExternalDeclarations generates extern declarations for Windows API
func (x64 *X64IOIntegration) GenerateExternalDeclarations() string {
	var asm strings.Builder

	asm.WriteString("; External Windows API declarations\n")
	for _, external := range x64.GetRequiredExternals() {
		asm.WriteString(fmt.Sprintf("extern %s\n", external))
	}
	asm.WriteString("\n")

	return asm.String()
}
