package runtime

import (
	"fmt"
	"os"
	"unsafe"
)

// Built-in functions for Orizon runtime
// These functions are compiled into the runtime and can be called from Orizon code

//go:linkname OrizonPrint orizon.print
func OrizonPrint(s string) {
	fmt.Print(s)
}

//go:linkname OrizonPrintln orizon.println
func OrizonPrintln(s string) {
	fmt.Println(s)
}

//go:linkname OrizonExit orizon.exit
func OrizonExit(code int) {
	os.Exit(code)
}

// Use unsafe to satisfy import requirement
var _ = unsafe.Pointer(nil)

// Assembly stubs for built-in functions
// These will be linked with the generated assembly

var BuiltinFunctions = map[string]string{
	"print": `
orizon_print:
	push rbp
	mov rbp, rsp
	; Windows x64 calling convention: rcx = first argument (string pointer)
	; For now, we'll use a simple syscall to write to stdout
	; This is a placeholder - in a real implementation, we'd link with C runtime
	mov rax, 1          ; sys_write
	mov rdi, 1          ; stdout
	mov rsi, rcx        ; string pointer
	mov rdx, [rcx-8]    ; string length (assuming it's stored before the string)
	syscall
	mov rsp, rbp
	pop rbp
	ret
`,
	"println": `
orizon_println:
	push rbp
	mov rbp, rsp
	; Call print first
	call orizon_print
	; Then print newline
	mov rax, 1          ; sys_write
	mov rdi, 1          ; stdout
	lea rsi, [newline]  ; newline character
	mov rdx, 1          ; length
	syscall
	mov rsp, rbp
	pop rbp
	ret

newline:
	db 10               ; ASCII newline
`,
	"exit": `
orizon_exit:
	push rbp
	mov rbp, rsp
	mov rax, 60         ; sys_exit
	mov rdi, rcx        ; exit code
	syscall
`,
}
