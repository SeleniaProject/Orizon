package io

import (
	"strings"
	"testing"

	"github.com/orizon-lang/orizon/internal/mir"
)

// TestMIRIntegration tests MIR code generation for I/O operations
func TestMIRIntegration(t *testing.T) {
	// Create a mock MIR module.
	module := &mir.Module{
		Name:      "test_io",
		Functions: []*mir.Function{},
	}

	// Create MIR integration.
	integration := NewIOMIRIntegration(module)
	if integration == nil {
		t.Fatal("Failed to create MIR integration")
	}

	// Test file operation function generation.
	fileOpenFunc := integration.GenerateFileOpenFunction()
	if fileOpenFunc == nil {
		t.Error("Failed to generate file open function")
	} else {
		if fileOpenFunc.Name != "orizon_io_file_open" {
			t.Errorf("Expected function name 'orizon_io_file_open', got '%s'", fileOpenFunc.Name)
		}
		if len(fileOpenFunc.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(fileOpenFunc.Parameters))
		}
	}

	fileCloseFunc := integration.GenerateFileCloseFunction()
	if fileCloseFunc == nil {
		t.Error("Failed to generate file close function")
	} else {
		if fileCloseFunc.Name != "orizon_io_file_close" {
			t.Errorf("Expected function name 'orizon_io_file_close', got '%s'", fileCloseFunc.Name)
		}
	}

	fileReadFunc := integration.GenerateFileReadFunction()
	if fileReadFunc == nil {
		t.Error("Failed to generate file read function")
	} else {
		if fileReadFunc.Name != "orizon_io_file_read" {
			t.Errorf("Expected function name 'orizon_io_file_read', got '%s'", fileReadFunc.Name)
		}
	}

	fileWriteFunc := integration.GenerateFileWriteFunction()
	if fileWriteFunc == nil {
		t.Error("Failed to generate file write function")
	} else {
		if fileWriteFunc.Name != "orizon_io_file_write" {
			t.Errorf("Expected function name 'orizon_io_file_write', got '%s'", fileWriteFunc.Name)
		}
	}
}

// TestX64Integration tests x64 assembly generation for I/O operations
func TestX64Integration(t *testing.T) {
	// Create x64 integration.
	integration := NewX64IOIntegration()
	if integration == nil {
		t.Fatal("Failed to create x64 integration")
	}

	// Test complete file open function generation.
	fileOpenAsm := integration.GenerateCompleteFileOpenFunction()
	if fileOpenAsm == "" {
		t.Error("Failed to generate complete file open function")
	} else {
		// Verify assembly contains basic structure.
		if !strings.Contains(fileOpenAsm, "push rbp") {
			t.Error("File open assembly should contain function prologue")
		}
		if !strings.Contains(fileOpenAsm, "ret") {
			t.Error("File open assembly should contain return instruction")
		}
	}

	// Test basic file operation assembly generation with enhanced validation.
	fileCloseAsm := integration.GenerateCompleteFileCloseFunction()
	if fileCloseAsm == "" {
		t.Error("Failed to generate complete file close function")
	} else {
		// Verify assembly contains proper function structure.
		requiredElements := []string{
			"push rbp",     // Function prologue
			"mov rbp, rsp", // Stack frame setup
			"CloseHandle",  // Windows API call
			"pop rbp",      // Function epilogue restoration
			"ret",          // Return instruction
		}

		for _, element := range requiredElements {
			if !strings.Contains(fileCloseAsm, element) {
				t.Errorf("File close assembly should contain '%s'", element)
			}
		}
	}

	fileReadAsm := integration.GenerateFileReadAsm("rcx", "rdx", "r8", "rax")
	if fileReadAsm == "" {
		t.Error("Failed to generate file read assembly")
	} else {
		if !strings.Contains(fileReadAsm, "call") {
			t.Error("File read assembly should contain system calls")
		}
	}

	fileWriteAsm := integration.GenerateFileWriteAsm("rcx", "rdx", "r8", "rax")
	if fileWriteAsm == "" {
		t.Error("Failed to generate file write assembly")
	} else {
		if !strings.Contains(fileWriteAsm, "call") {
			t.Error("File write assembly should contain system calls")
		}
	}
}

// TestConsoleIntegration tests console I/O integration
func TestConsoleIntegration(t *testing.T) {
	// Test MIR integration for console operations.
	module := &mir.Module{
		Name:      "test_console",
		Functions: []*mir.Function{},
	}

	mirIntegration := NewIOMIRIntegration(module)
	if mirIntegration == nil {
		t.Fatal("Failed to create MIR integration")
	}

	consoleWriteFunc := mirIntegration.GenerateConsoleWriteFunction()
	if consoleWriteFunc == nil {
		t.Error("Failed to generate console write function")
	} else {
		if consoleWriteFunc.Name != "orizon_io_console_write" {
			t.Errorf("Expected function name 'orizon_io_console_write', got '%s'", consoleWriteFunc.Name)
		}
	}

	consoleReadFunc := mirIntegration.GenerateConsoleReadFunction()
	if consoleReadFunc == nil {
		t.Error("Failed to generate console read function")
	} else {
		if consoleReadFunc.Name != "orizon_io_console_read" {
			t.Errorf("Expected function name 'orizon_io_console_read', got '%s'", consoleReadFunc.Name)
		}
	}

	// Test x64 integration for console operations.
	x64Integration := NewX64IOIntegration()
	if x64Integration == nil {
		t.Fatal("Failed to create x64 integration")
	}

	consoleWriteAsm := x64Integration.GenerateConsoleWriteAsm("rcx", "rdx", "r8", "rax")
	if consoleWriteAsm == "" {
		t.Error("Failed to generate console write assembly")
	} else {
		// Console operations should use Windows API.
		if !strings.Contains(consoleWriteAsm, "WriteConsole") && !strings.Contains(consoleWriteAsm, "WriteFile") {
			t.Error("Console write assembly should contain Windows API calls")
		}
	}

	consoleReadAsm := x64Integration.GenerateConsoleReadAsm("rcx", "rdx", "rax")
	if consoleReadAsm == "" {
		t.Error("Failed to generate console read assembly")
	} else {
		if !strings.Contains(consoleReadAsm, "ReadConsole") && !strings.Contains(consoleReadAsm, "ReadFile") {
			t.Error("Console read assembly should contain Windows API calls")
		}
	}
}

// TestThreadingIntegration tests threading integration.
func TestThreadingIntegration(t *testing.T) {
	// Test MIR integration for threading operations.
	module := &mir.Module{
		Name:      "test_threading",
		Functions: []*mir.Function{},
	}

	mirIntegration := NewIOMIRIntegration(module)
	if mirIntegration == nil {
		t.Fatal("Failed to create MIR integration")
	}

	threadCreateFunc := mirIntegration.GenerateThreadCreateFunction()
	if threadCreateFunc == nil {
		t.Error("Failed to generate thread create function")
	} else {
		if threadCreateFunc.Name != "orizon_io_thread_create" {
			t.Errorf("Expected function name 'orizon_io_thread_create', got '%s'", threadCreateFunc.Name)
		}
	}

	mutexLockFunc := mirIntegration.GenerateMutexLockFunction()
	if mutexLockFunc == nil {
		t.Error("Failed to generate mutex lock function")
	} else {
		if mutexLockFunc.Name != "orizon_io_mutex_lock" {
			t.Errorf("Expected function name 'orizon_io_mutex_lock', got '%s'", mutexLockFunc.Name)
		}
	}

	// Test x64 integration for threading operations.
	x64Integration := NewX64IOIntegration()
	if x64Integration == nil {
		t.Fatal("Failed to create x64 integration")
	}

	threadCreateAsm := x64Integration.GenerateThreadCreateAsm("rcx", "rdx", "rax")
	if threadCreateAsm == "" {
		t.Error("Failed to generate thread create assembly")
	} else {
		// Threading operations should use Windows threading API.
		if !strings.Contains(threadCreateAsm, "CreateThread") {
			t.Error("Thread create assembly should contain CreateThread API call")
		}
	}

	mutexLockAsm := x64Integration.GenerateMutexLockAsm("rcx", "rax")
	if mutexLockAsm == "" {
		t.Error("Failed to generate mutex lock assembly")
	} else {
		if !strings.Contains(mutexLockAsm, "WaitForSingleObject") && !strings.Contains(mutexLockAsm, "EnterCriticalSection") {
			t.Error("Mutex lock assembly should contain synchronization API calls")
		}
	}
}

// TestIntegrationConsistency tests consistency between MIR and x64 generations.
func TestIntegrationConsistency(t *testing.T) {
	// Create integrations.
	module := &mir.Module{
		Name:      "test_consistency",
		Functions: []*mir.Function{},
	}

	mirIntegration := NewIOMIRIntegration(module)
	x64Integration := NewX64IOIntegration()

	if mirIntegration == nil {
		t.Fatal("Failed to create MIR integration")
	}
	if x64Integration == nil {
		t.Fatal("Failed to create x64 integration")
	}

	// Test that both integrations can generate code for the same operations.
	mirOperations := []struct {
		name   string
		mirGen func() *mir.Function
	}{
		{"file_open", mirIntegration.GenerateFileOpenFunction},
		{"file_close", mirIntegration.GenerateFileCloseFunction},
		{"file_read", mirIntegration.GenerateFileReadFunction},
		{"file_write", mirIntegration.GenerateFileWriteFunction},
		{"console_write", mirIntegration.GenerateConsoleWriteFunction},
		{"console_read", mirIntegration.GenerateConsoleReadFunction},
		{"thread_create", mirIntegration.GenerateThreadCreateFunction},
		{"mutex_lock", mirIntegration.GenerateMutexLockFunction},
	}

	x64Operations := []struct {
		name   string
		x64Gen func() string
	}{
		{"file_open", func() string { return x64Integration.GenerateFileOpenAsm("rcx", "rdx", "rax") }},
		{"file_close", func() string { return x64Integration.GenerateFileCloseAsm("rcx", "rax") }},
		{"file_read", func() string { return x64Integration.GenerateFileReadAsm("rcx", "rdx", "r8", "rax") }},
		{"file_write", func() string { return x64Integration.GenerateFileWriteAsm("rcx", "rdx", "r8", "rax") }},
		{"console_write", func() string { return x64Integration.GenerateConsoleWriteAsm("rcx", "rdx", "r8", "rax") }},
		{"console_read", func() string { return x64Integration.GenerateConsoleReadAsm("rcx", "rdx", "rax") }},
		{"thread_create", func() string { return x64Integration.GenerateThreadCreateAsm("rcx", "rdx", "rax") }},
		{"mutex_lock", func() string { return x64Integration.GenerateMutexLockAsm("rcx", "rax") }},
	}

	// Verify both have the same number of operations.
	if len(mirOperations) != len(x64Operations) {
		t.Errorf("MIR and x64 should support the same number of operations: MIR=%d, x64=%d",
			len(mirOperations), len(x64Operations))
	}

	// Verify both have implementations for each operation.
	for i, mirOp := range mirOperations {
		if i >= len(x64Operations) {
			t.Errorf("Missing x64 implementation for %s", mirOp.name)
			continue
		}

		x64Op := x64Operations[i]
		if mirOp.name != x64Op.name {
			t.Errorf("Operation mismatch: MIR=%s, x64=%s", mirOp.name, x64Op.name)
		}

		// Test MIR generation.
		mirFunc := mirOp.mirGen()
		if mirFunc == nil {
			t.Errorf("Missing MIR implementation for %s", mirOp.name)
		}

		// Test x64 generation.
		x64Asm := x64Op.x64Gen()
		if x64Asm == "" {
			t.Errorf("Missing x64 implementation for %s", x64Op.name)
		}

		// Both should succeed.
		if mirFunc == nil || x64Asm == "" {
			t.Errorf("Incomplete implementation for %s: MIR=%v, x64=%v",
				mirOp.name, mirFunc != nil, x64Asm != "")
		}
	}
}

// TestMIRFunctionStructure tests the structure of generated MIR functions.
func TestMIRFunctionStructure(t *testing.T) {
	module := &mir.Module{
		Name:      "test_structure",
		Functions: []*mir.Function{},
	}

	integration := NewIOMIRIntegration(module)
	if integration == nil {
		t.Fatal("Failed to create MIR integration")
	}

	// Test file open function structure.
	fileOpenFunc := integration.GenerateFileOpenFunction()
	if fileOpenFunc == nil {
		t.Fatal("Failed to generate file open function")
	}

	// Verify function has correct name.
	if fileOpenFunc.Name == "" {
		t.Error("Function should have a name")
	}

	// Verify function has parameters.
	if len(fileOpenFunc.Parameters) == 0 {
		t.Error("File open function should have parameters")
	}

	// Verify function has blocks.
	if len(fileOpenFunc.Blocks) == 0 {
		t.Error("Function should have basic blocks")
	}

	// Verify entry block exists.
	entryBlock := fileOpenFunc.Blocks[0]
	if entryBlock.Name != "entry" {
		t.Error("First block should be entry block")
	}

	// Verify block has instructions.
	if len(entryBlock.Instr) == 0 {
		t.Error("Entry block should have instructions")
	}
}

// TestX64AssemblyStructure tests the structure of generated x64 assembly.
func TestX64AssemblyStructure(t *testing.T) {
	integration := NewX64IOIntegration()
	if integration == nil {
		t.Fatal("Failed to create x64 integration")
	}

	// Test complete file open function structure.
	fileOpenAsm := integration.GenerateCompleteFileOpenFunction()
	if fileOpenAsm == "" {
		t.Fatal("Failed to generate complete file open function")
	}

	// Verify assembly contains required elements.
	requiredElements := []string{
		"orizon_io_file_open:", // Function label
		"push rbp",             // Function prologue
		"mov rbp, rsp",         // Stack frame setup
		"pop rbp",              // Function epilogue
		"ret",                  // Return instruction
	}

	for _, element := range requiredElements {
		if !strings.Contains(fileOpenAsm, element) {
			t.Errorf("Assembly should contain '%s'", element)
		}
	}

	// Verify Windows API usage.
	windowsAPIs := []string{
		"CreateFile",
		"GetLastError",
	}

	foundAPI := false
	for _, api := range windowsAPIs {
		if strings.Contains(fileOpenAsm, api) {
			foundAPI = true
			break
		}
	}

	if !foundAPI {
		t.Error("Assembly should contain Windows API calls")
	}
}
