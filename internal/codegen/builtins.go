package codegen

import (
	"strings"
)

// BuiltinFunctions contains the list of built-in functions available in Orizon
var BuiltinFunctions = map[string]BuiltinFunction{
	"print": {
		Name:       "print",
		ReturnType: "void",
		Parameters: []BuiltinParameter{
			{Name: "message", Type: "string"},
		},
		AssemblyName: "orizon_print",
	},
	"println": {
		Name:       "println",
		ReturnType: "void",
		Parameters: []BuiltinParameter{
			{Name: "message", Type: "string"},
		},
		AssemblyName: "orizon_println",
	},
	"exit": {
		Name:       "exit",
		ReturnType: "void",
		Parameters: []BuiltinParameter{
			{Name: "code", Type: "int"},
		},
		AssemblyName: "orizon_exit",
	},
}

// BuiltinFunction represents a built-in function definition
type BuiltinFunction struct {
	Name         string
	ReturnType   string
	Parameters   []BuiltinParameter
	AssemblyName string
}

// BuiltinParameter represents a parameter of a built-in function
type BuiltinParameter struct {
	Name string
	Type string
}

// IsBuiltinFunction checks if a function name is a built-in function
func IsBuiltinFunction(name string) bool {
	_, exists := BuiltinFunctions[name]
	return exists
}

// GetBuiltinFunction returns the built-in function definition
func GetBuiltinFunction(name string) (BuiltinFunction, bool) {
	fn, exists := BuiltinFunctions[name]
	return fn, exists
}

// GenerateBuiltinAssembly generates assembly code for all built-in functions
func GenerateBuiltinAssembly() string {
	var b strings.Builder

	b.WriteString("; Built-in functions for Orizon\n")
	b.WriteString("; Windows x64 calling convention\n\n")

	// print function
	b.WriteString("orizon_print:\n")
	b.WriteString("  push rbp\n")
	b.WriteString("  mov rbp, rsp\n")
	b.WriteString("  sub rsp, 32        ; Shadow space\n")
	b.WriteString("  ; rcx contains string pointer\n")
	b.WriteString("  ; For now, we'll use Windows WriteConsoleA\n")
	b.WriteString("  ; Get string length first\n")
	b.WriteString("  mov rax, rcx       ; string pointer\n")
	b.WriteString("  mov rdx, 0         ; counter\n")
	b.WriteString(".count_loop:\n")
	b.WriteString("  cmp byte ptr [rax + rdx], 0\n")
	b.WriteString("  je .count_done\n")
	b.WriteString("  inc rdx\n")
	b.WriteString("  jmp .count_loop\n")
	b.WriteString(".count_done:\n")
	b.WriteString("  ; rdx now contains string length\n")
	b.WriteString("  ; Call WriteConsoleA (simplified - would need proper Windows API setup)\n")
	b.WriteString("  ; For diagnostic purposes, just return\n")
	b.WriteString("  add rsp, 32\n")
	b.WriteString("  mov rsp, rbp\n")
	b.WriteString("  pop rbp\n")
	b.WriteString("  ret\n\n")

	// println function
	b.WriteString("orizon_println:\n")
	b.WriteString("  push rbp\n")
	b.WriteString("  mov rbp, rsp\n")
	b.WriteString("  sub rsp, 32\n")
	b.WriteString("  call orizon_print\n")
	b.WriteString("  ; Print newline\n")
	b.WriteString("  mov rcx, newline_str\n")
	b.WriteString("  call orizon_print\n")
	b.WriteString("  add rsp, 32\n")
	b.WriteString("  mov rsp, rbp\n")
	b.WriteString("  pop rbp\n")
	b.WriteString("  ret\n\n")

	// exit function
	b.WriteString("orizon_exit:\n")
	b.WriteString("  push rbp\n")
	b.WriteString("  mov rbp, rsp\n")
	b.WriteString("  ; rcx contains exit code\n")
	b.WriteString("  ; For Windows, call ExitProcess\n")
	b.WriteString("  ; Simplified - just halt\n")
	b.WriteString("  mov rax, rcx\n")
	b.WriteString("  mov rsp, rbp\n")
	b.WriteString("  pop rbp\n")
	b.WriteString("  ret\n\n")

	// Data section
	b.WriteString("section .data\n")
	b.WriteString("newline_str db 10, 0  ; newline + null terminator\n\n")

	return b.String()
}
