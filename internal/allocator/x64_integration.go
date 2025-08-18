package allocator

import (
	"fmt"
	"strings"
)

// X64Integration provides x64 assembly integration for memory allocation
type X64Integration struct {
	allocatorType  AllocatorKind
	config         *Config
	mirIntegration *MIRIntegration
}

// X64Instruction represents an x64 assembly instruction
type X64Instruction struct {
	Mnemonic string
	Operands []string
	Comment  string
}

// NewX64Integration creates a new x64 integration
func NewX64Integration(allocatorType AllocatorKind, config *Config) *X64Integration {
	return &X64Integration{
		allocatorType:  allocatorType,
		config:         config,
		mirIntegration: NewMIRIntegration(allocatorType, config),
	}
}

// GenerateAllocCode generates x64 assembly for memory allocation
func (xi *X64Integration) GenerateAllocCode(sizeReg string, resultReg string) []X64Instruction {
	var instructions []X64Instruction

	switch xi.allocatorType {
	case SystemAllocatorKind:
		instructions = xi.generateSystemAllocASM(sizeReg, resultReg)
	case ArenaAllocatorKind:
		instructions = xi.generateArenaAllocASM(sizeReg, resultReg)
	case PoolAllocatorKind:
		instructions = xi.generatePoolAllocASM(sizeReg, resultReg)
	default:
		instructions = xi.generateSystemAllocASM(sizeReg, resultReg)
	}

	return instructions
}

// GenerateFreeCode generates x64 assembly for memory deallocation
func (xi *X64Integration) GenerateFreeCode(ptrReg string) []X64Instruction {
	var instructions []X64Instruction

	switch xi.allocatorType {
	case SystemAllocatorKind:
		instructions = xi.generateSystemFreeASM(ptrReg)
	case ArenaAllocatorKind:
		instructions = xi.generateArenaFreeASM(ptrReg)
	case PoolAllocatorKind:
		instructions = xi.generatePoolFreeASM(ptrReg)
	default:
		instructions = xi.generateSystemFreeASM(ptrReg)
	}

	return instructions
}

// generateSystemAllocASM generates system allocation assembly
func (xi *X64Integration) generateSystemAllocASM(sizeReg string, resultReg string) []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "push",
			Operands: []string{"rcx"},
			Comment:  "Save caller registers",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rdx"},
			Comment:  "",
		},
		{
			Mnemonic: "push",
			Operands: []string{"r8"},
			Comment:  "",
		},
		{
			Mnemonic: "push",
			Operands: []string{"r9"},
			Comment:  "",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", sizeReg},
			Comment:  "Move size to first argument register",
		},
		{
			Mnemonic: "call",
			Operands: []string{"runtime_alloc"},
			Comment:  "Call runtime allocation function",
		},
		{
			Mnemonic: "mov",
			Operands: []string{resultReg, "rax"},
			Comment:  "Move result to target register",
		},
		{
			Mnemonic: "test",
			Operands: []string{"rax", "rax"},
			Comment:  "Check if allocation succeeded",
		},
		{
			Mnemonic: "jz",
			Operands: []string{"alloc_failed"},
			Comment:  "Jump to error handler if allocation failed",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"r9"},
			Comment:  "Restore caller registers",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"r8"},
			Comment:  "",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rdx"},
			Comment:  "",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rcx"},
			Comment:  "",
		},
	}
}

// generateArenaAllocASM generates arena allocation assembly
func (xi *X64Integration) generateArenaAllocASM(sizeReg string, resultReg string) []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "mov",
			Operands: []string{"rax", "qword ptr [global_arena_current]"},
			Comment:  "Load current arena pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rdx", "qword ptr [global_arena_end]"},
			Comment:  "Load arena end pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", sizeReg},
			Comment:  "Load allocation size",
		},
		{
			Mnemonic: "add",
			Operands: []string{"rcx", fmt.Sprintf("%d", xi.config.AlignmentSize-1)},
			Comment:  "Add alignment padding",
		},
		{
			Mnemonic: "and",
			Operands: []string{"rcx", fmt.Sprintf("-%d", xi.config.AlignmentSize)},
			Comment:  "Align size to boundary",
		},
		{
			Mnemonic: "lea",
			Operands: []string{"r8", "[rax + rcx]"},
			Comment:  "Calculate new current pointer",
		},
		{
			Mnemonic: "cmp",
			Operands: []string{"r8", "rdx"},
			Comment:  "Check if allocation fits in arena",
		},
		{
			Mnemonic: "ja",
			Operands: []string{"arena_exhausted"},
			Comment:  "Jump if arena is exhausted",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"qword ptr [global_arena_current]", "r8"},
			Comment:  "Update arena current pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{resultReg, "rax"},
			Comment:  "Return allocated pointer",
		},
		{
			Mnemonic: "jmp",
			Operands: []string{"arena_alloc_done"},
			Comment:  "Jump to completion",
		},
		{
			Mnemonic: "arena_exhausted:",
			Operands: []string{},
			Comment:  "Arena exhausted error handler",
		},
		{
			Mnemonic: "xor",
			Operands: []string{resultReg, resultReg},
			Comment:  "Return null pointer",
		},
		{
			Mnemonic: "arena_alloc_done:",
			Operands: []string{},
			Comment:  "Allocation complete",
		},
	}
}

// generatePoolAllocASM generates pool allocation assembly
func (xi *X64Integration) generatePoolAllocASM(sizeReg string, resultReg string) []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "push",
			Operands: []string{"rcx"},
			Comment:  "Save caller registers",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rdx"},
			Comment:  "",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", "qword ptr [global_pool_allocator]"},
			Comment:  "Load pool allocator pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rdx", sizeReg},
			Comment:  "Load allocation size",
		},
		{
			Mnemonic: "call",
			Operands: []string{"pool_alloc_method"},
			Comment:  "Call pool allocation method",
		},
		{
			Mnemonic: "mov",
			Operands: []string{resultReg, "rax"},
			Comment:  "Move result to target register",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rdx"},
			Comment:  "Restore caller registers",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rcx"},
			Comment:  "",
		},
	}
}

// generateSystemFreeASM generates system free assembly
func (xi *X64Integration) generateSystemFreeASM(ptrReg string) []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "test",
			Operands: []string{ptrReg, ptrReg},
			Comment:  "Check if pointer is null",
		},
		{
			Mnemonic: "jz",
			Operands: []string{"free_done"},
			Comment:  "Skip if null pointer",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rcx"},
			Comment:  "Save caller registers",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", ptrReg},
			Comment:  "Move pointer to first argument register",
		},
		{
			Mnemonic: "call",
			Operands: []string{"runtime_free"},
			Comment:  "Call runtime free function",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rcx"},
			Comment:  "Restore caller registers",
		},
		{
			Mnemonic: "free_done:",
			Operands: []string{},
			Comment:  "Free operation complete",
		},
	}
}

// generateArenaFreeASM generates arena free assembly (no-op)
func (xi *X64Integration) generateArenaFreeASM(ptrReg string) []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "nop",
			Operands: []string{},
			Comment:  "Arena allocator - free is no-op",
		},
	}
}

// generatePoolFreeASM generates pool free assembly
func (xi *X64Integration) generatePoolFreeASM(ptrReg string) []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "test",
			Operands: []string{ptrReg, ptrReg},
			Comment:  "Check if pointer is null",
		},
		{
			Mnemonic: "jz",
			Operands: []string{"pool_free_done"},
			Comment:  "Skip if null pointer",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rcx"},
			Comment:  "Save caller registers",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rdx"},
			Comment:  "",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", "qword ptr [global_pool_allocator]"},
			Comment:  "Load pool allocator pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rdx", ptrReg},
			Comment:  "Load pointer to free",
		},
		{
			Mnemonic: "call",
			Operands: []string{"pool_free_method"},
			Comment:  "Call pool free method",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rdx"},
			Comment:  "Restore caller registers",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rcx"},
			Comment:  "",
		},
		{
			Mnemonic: "pool_free_done:",
			Operands: []string{},
			Comment:  "Pool free operation complete",
		},
	}
}

// GenerateInlinedArenaAlloc generates optimized inline arena allocation
func (xi *X64Integration) GenerateInlinedArenaAlloc(size int, resultReg string) []X64Instruction {
	alignedSize := ((size + int(xi.config.AlignmentSize) - 1) / int(xi.config.AlignmentSize)) * int(xi.config.AlignmentSize)

	return []X64Instruction{
		{
			Mnemonic: "mov",
			Operands: []string{"rax", "qword ptr [global_arena_current]"},
			Comment:  "Load current arena pointer",
		},
		{
			Mnemonic: "lea",
			Operands: []string{"rdx", fmt.Sprintf("[rax + %d]", alignedSize)},
			Comment:  "Calculate new current pointer",
		},
		{
			Mnemonic: "cmp",
			Operands: []string{"rdx", "qword ptr [global_arena_end]"},
			Comment:  "Check if allocation fits",
		},
		{
			Mnemonic: "ja",
			Operands: []string{"slow_arena_alloc"},
			Comment:  "Use slow path if doesn't fit",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"qword ptr [global_arena_current]", "rdx"},
			Comment:  "Update arena current pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{resultReg, "rax"},
			Comment:  "Return allocated pointer",
		},
		{
			Mnemonic: "jmp",
			Operands: []string{"inline_alloc_done"},
			Comment:  "Jump to completion",
		},
		{
			Mnemonic: "slow_arena_alloc:",
			Operands: []string{},
			Comment:  "Slow allocation path",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rcx"},
			Comment:  "Save registers for function call",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", fmt.Sprintf("%d", size)},
			Comment:  "Load size argument",
		},
		{
			Mnemonic: "call",
			Operands: []string{"arena_alloc_slow"},
			Comment:  "Call slow allocation function",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rcx"},
			Comment:  "Restore registers",
		},
		{
			Mnemonic: "mov",
			Operands: []string{resultReg, "rax"},
			Comment:  "Move result to target register",
		},
		{
			Mnemonic: "inline_alloc_done:",
			Operands: []string{},
			Comment:  "Inline allocation complete",
		},
	}
}

// GenerateMemoryBarrier generates memory barrier instructions
func (xi *X64Integration) GenerateMemoryBarrier() []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "mfence",
			Operands: []string{},
			Comment:  "Full memory barrier",
		},
	}
}

// GenerateAtomicAlloc generates atomic allocation with memory barriers
func (xi *X64Integration) GenerateAtomicAlloc(sizeReg string, resultReg string) []X64Instruction {
	instructions := []X64Instruction{
		{
			Mnemonic: "mfence",
			Operands: []string{},
			Comment:  "Memory barrier before allocation",
		},
	}

	instructions = append(instructions, xi.GenerateAllocCode(sizeReg, resultReg)...)

	instructions = append(instructions, []X64Instruction{
		{
			Mnemonic: "mfence",
			Operands: []string{},
			Comment:  "Memory barrier after allocation",
		},
	}...)

	return instructions
}

// GenerateStackAlloc generates stack allocation
func (xi *X64Integration) GenerateStackAlloc(size int, resultReg string) []X64Instruction {
	alignedSize := ((size + 15) / 16) * 16 // 16-byte align for stack

	return []X64Instruction{
		{
			Mnemonic: "sub",
			Operands: []string{"rsp", fmt.Sprintf("%d", alignedSize)},
			Comment:  "Allocate space on stack",
		},
		{
			Mnemonic: "mov",
			Operands: []string{resultReg, "rsp"},
			Comment:  "Get pointer to allocated space",
		},
	}
}

// GenerateStackFree generates stack deallocation
func (xi *X64Integration) GenerateStackFree(size int) []X64Instruction {
	alignedSize := ((size + 15) / 16) * 16 // 16-byte align for stack

	return []X64Instruction{
		{
			Mnemonic: "add",
			Operands: []string{"rsp", fmt.Sprintf("%d", alignedSize)},
			Comment:  "Free space on stack",
		},
	}
}

// GenerateErrorHandling generates error handling code for allocation failures
func (xi *X64Integration) GenerateErrorHandling() []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "alloc_failed:",
			Operands: []string{},
			Comment:  "Allocation failure handler",
		},
		{
			Mnemonic: "push",
			Operands: []string{"rcx"},
			Comment:  "Save registers",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rcx", "1"}, // OUT_OF_MEMORY exception
			Comment:  "Load out of memory exception code",
		},
		{
			Mnemonic: "call",
			Operands: []string{"runtime_panic"},
			Comment:  "Call runtime panic handler",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rcx"},
			Comment:  "Restore registers (unreachable)",
		},
		{
			Mnemonic: "ret",
			Operands: []string{},
			Comment:  "Return (unreachable)",
		},
	}
}

// GeneratePrologue generates function prologue for allocator functions
func (xi *X64Integration) GeneratePrologue() []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "push",
			Operands: []string{"rbp"},
			Comment:  "Save frame pointer",
		},
		{
			Mnemonic: "mov",
			Operands: []string{"rbp", "rsp"},
			Comment:  "Set up frame pointer",
		},
	}
}

// GenerateEpilogue generates function epilogue for allocator functions
func (xi *X64Integration) GenerateEpilogue() []X64Instruction {
	return []X64Instruction{
		{
			Mnemonic: "mov",
			Operands: []string{"rsp", "rbp"},
			Comment:  "Restore stack pointer",
		},
		{
			Mnemonic: "pop",
			Operands: []string{"rbp"},
			Comment:  "Restore frame pointer",
		},
		{
			Mnemonic: "ret",
			Operands: []string{},
			Comment:  "Return to caller",
		},
	}
}

// FormatInstructions formats x64 instructions as assembly code
func (xi *X64Integration) FormatInstructions(instructions []X64Instruction) string {
	var result strings.Builder

	for _, instr := range instructions {
		if strings.HasSuffix(instr.Mnemonic, ":") {
			// Label
			result.WriteString(instr.Mnemonic)
		} else {
			// Regular instruction
			result.WriteString(fmt.Sprintf("    %-8s", instr.Mnemonic))
			if len(instr.Operands) > 0 {
				result.WriteString(strings.Join(instr.Operands, ", "))
			}
		}

		if instr.Comment != "" {
			if !strings.HasSuffix(instr.Mnemonic, ":") {
				result.WriteString(fmt.Sprintf("  ; %s", instr.Comment))
			}
		}

		result.WriteString("\n")
	}

	return result.String()
}

// OptimizeInstructions optimizes the generated x64 instructions
func (xi *X64Integration) OptimizeInstructions(instructions []X64Instruction) []X64Instruction {
	optimized := make([]X64Instruction, 0, len(instructions))

	for i, instr := range instructions {
		// Skip redundant moves
		if instr.Mnemonic == "mov" && len(instr.Operands) == 2 &&
			instr.Operands[0] == instr.Operands[1] {
			continue
		}

		// Combine push/pop sequences
		if instr.Mnemonic == "push" && i+1 < len(instructions) &&
			instructions[i+1].Mnemonic == "pop" &&
			len(instr.Operands) > 0 && len(instructions[i+1].Operands) > 0 &&
			instr.Operands[0] == instructions[i+1].Operands[0] {
			// Skip redundant push/pop pair
			i++ // Skip next instruction too
			continue
		}

		// Remove redundant memory barriers
		if instr.Mnemonic == "mfence" && i > 0 &&
			instructions[i-1].Mnemonic == "mfence" {
			continue
		}

		optimized = append(optimized, instr)
	}

	return optimized
}

// GenerateCompleteFunction generates a complete function with prologue, body, and epilogue
func (xi *X64Integration) GenerateCompleteFunction(name string, bodyInstructions []X64Instruction) []X64Instruction {
	var complete []X64Instruction

	// Function label
	complete = append(complete, X64Instruction{
		Mnemonic: name + ":",
		Operands: []string{},
		Comment:  "",
	})

	// Prologue
	complete = append(complete, xi.GeneratePrologue()...)

	// Body
	complete = append(complete, bodyInstructions...)

	// Epilogue
	complete = append(complete, xi.GenerateEpilogue()...)

	return complete
}

// GenerateAllocatorGlobals generates global variable declarations for the allocator
func (xi *X64Integration) GenerateAllocatorGlobals() string {
	var globals strings.Builder

	globals.WriteString("; Allocator global variables\n")
	globals.WriteString(".data\n")

	switch xi.allocatorType {
	case ArenaAllocatorKind:
		globals.WriteString("global_arena_current    dq 0    ; Current arena pointer\n")
		globals.WriteString("global_arena_end        dq 0    ; Arena end pointer\n")
		globals.WriteString("global_arena_start      dq 0    ; Arena start pointer\n")
	case PoolAllocatorKind:
		globals.WriteString("global_pool_allocator   dq 0    ; Pool allocator instance\n")
	case SystemAllocatorKind:
		globals.WriteString("global_system_allocator dq 0    ; System allocator instance\n")
	}

	globals.WriteString("global_allocator        dq 0    ; Global allocator pointer\n")
	globals.WriteString("\n")

	return globals.String()
}

// GenerateAllocatorInterface generates the complete allocator interface
func (xi *X64Integration) GenerateAllocatorInterface() string {
	var code strings.Builder

	// Global variables
	code.WriteString(xi.GenerateAllocatorGlobals())

	// Code section
	code.WriteString(".text\n\n")

	// Allocation function
	allocBody := xi.GenerateAllocCode("rcx", "rax")
	allocBody = append(allocBody, xi.GenerateErrorHandling()...)
	allocFunc := xi.GenerateCompleteFunction("allocator_alloc", allocBody)
	code.WriteString(xi.FormatInstructions(allocFunc))
	code.WriteString("\n")

	// Free function
	freeBody := xi.GenerateFreeCode("rcx")
	freeFunc := xi.GenerateCompleteFunction("allocator_free", freeBody)
	code.WriteString(xi.FormatInstructions(freeFunc))
	code.WriteString("\n")

	return code.String()
}
