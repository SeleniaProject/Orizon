package allocator

import (
	"fmt"
)

// MIRIntegration provides MIR-level integration for memory allocation
type MIRIntegration struct {
	allocatorType AllocatorKind
	config        *Config
}

// MIRInstruction represents a simplified MIR instruction for allocation
type MIRInstruction struct {
	Op       string
	Operands []string
	Result   string
	Type     string
	Size     int
}

// NewMIRIntegration creates a new MIR integration
func NewMIRIntegration(allocatorType AllocatorKind, config *Config) *MIRIntegration {
	return &MIRIntegration{
		allocatorType: allocatorType,
		config:        config,
	}
}

// GenerateAllocInstruction generates MIR instructions for memory allocation
func (mi *MIRIntegration) GenerateAllocInstruction(resultReg string, size int, alignment int) []MIRInstruction {
	var instructions []MIRInstruction

	switch mi.allocatorType {
	case SystemAllocatorKind:
		instructions = mi.generateSystemAlloc(resultReg, size, alignment)
	case ArenaAllocatorKind:
		instructions = mi.generateArenaAlloc(resultReg, size, alignment)
	case PoolAllocatorKind:
		instructions = mi.generatePoolAlloc(resultReg, size, alignment)
	default:
		// Fallback to system allocation
		instructions = mi.generateSystemAlloc(resultReg, size, alignment)
	}

	return instructions
}

// GenerateFreeInstruction generates MIR instructions for memory deallocation
func (mi *MIRIntegration) GenerateFreeInstruction(ptrReg string) []MIRInstruction {
	var instructions []MIRInstruction

	switch mi.allocatorType {
	case SystemAllocatorKind:
		instructions = mi.generateSystemFree(ptrReg)
	case ArenaAllocatorKind:
		instructions = mi.generateArenaFree(ptrReg)
	case PoolAllocatorKind:
		instructions = mi.generatePoolFree(ptrReg)
	default:
		// Fallback to system free
		instructions = mi.generateSystemFree(ptrReg)
	}

	return instructions
}

// generateSystemAlloc generates system allocation instructions
func (mi *MIRIntegration) generateSystemAlloc(resultReg string, size int, alignment int) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", size)},
			Result:   "tmp_size",
			Type:     "uintptr",
		},
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", alignment)},
			Result:   "tmp_align",
			Type:     "uintptr",
		},
		{
			Op:       "call_runtime",
			Operands: []string{"runtime_alloc", "tmp_size"},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// generateArenaAlloc generates arena allocation instructions
func (mi *MIRIntegration) generateArenaAlloc(resultReg string, size int, alignment int) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "load_global",
			Operands: []string{"global_arena"},
			Result:   "arena_ptr",
			Type:     "ptr",
		},
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", size)},
			Result:   "tmp_size",
			Type:     "uintptr",
		},
		{
			Op:       "call_method",
			Operands: []string{"arena_ptr", "alloc", "tmp_size"},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// generatePoolAlloc generates pool allocation instructions
func (mi *MIRIntegration) generatePoolAlloc(resultReg string, size int, alignment int) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "load_global",
			Operands: []string{"global_pool_allocator"},
			Result:   "pool_ptr",
			Type:     "ptr",
		},
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", size)},
			Result:   "tmp_size",
			Type:     "uintptr",
		},
		{
			Op:       "call_method",
			Operands: []string{"pool_ptr", "alloc", "tmp_size"},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// generateSystemFree generates system free instructions
func (mi *MIRIntegration) generateSystemFree(ptrReg string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "null_check",
			Operands: []string{ptrReg},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "call_runtime",
			Operands: []string{"runtime_free", ptrReg},
			Result:   "",
			Type:     "",
		},
	}
}

// generateArenaFree generates arena free instructions (no-op)
func (mi *MIRIntegration) generateArenaFree(ptrReg string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "comment",
			Operands: []string{"arena allocator - free is no-op"},
			Result:   "",
			Type:     "",
		},
	}
}

// generatePoolFree generates pool free instructions
func (mi *MIRIntegration) generatePoolFree(ptrReg string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "null_check",
			Operands: []string{ptrReg},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "load_global",
			Operands: []string{"global_pool_allocator"},
			Result:   "pool_ptr",
			Type:     "ptr",
		},
		{
			Op:       "call_method",
			Operands: []string{"pool_ptr", "free", ptrReg},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateArrayAllocInstruction generates MIR for array allocation
func (mi *MIRIntegration) GenerateArrayAllocInstruction(resultReg string, elementSize int, count string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", elementSize)},
			Result:   "element_size",
			Type:     "uintptr",
		},
		{
			Op:       "mul",
			Operands: []string{"element_size", count},
			Result:   "total_size",
			Type:     "uintptr",
		},
		{
			Op:       "call_runtime",
			Operands: []string{"runtime_alloc_array", "element_size", count},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateSliceAllocInstruction generates MIR for slice allocation
func (mi *MIRIntegration) GenerateSliceAllocInstruction(resultReg string, elementSize int, len, cap string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", elementSize)},
			Result:   "element_size",
			Type:     "uintptr",
		},
		{
			Op:       "call_runtime",
			Operands: []string{"runtime_alloc_slice", "element_size", len, cap},
			Result:   resultReg,
			Type:     "slice_header",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateStringAllocInstruction generates MIR for string allocation
func (mi *MIRIntegration) GenerateStringAllocInstruction(resultReg string, strReg string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "call_runtime",
			Operands: []string{"runtime_alloc_string", strReg},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateGCInstructions generates MIR for garbage collection
func (mi *MIRIntegration) GenerateGCInstructions() []MIRInstruction {
	if mi.allocatorType == ArenaAllocatorKind {
		return []MIRInstruction{
			{
				Op:       "call_runtime",
				Operands: []string{"runtime_force_gc"},
				Result:   "",
				Type:     "",
			},
		}
	}

	return []MIRInstruction{
		{
			Op:       "comment",
			Operands: []string{"GC not supported for this allocator type"},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateMemoryBarrierInstruction generates MIR for memory barriers
func (mi *MIRIntegration) GenerateMemoryBarrierInstruction() []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "memory_barrier",
			Operands: []string{},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateAtomicAllocInstruction generates MIR for atomic allocation
func (mi *MIRIntegration) GenerateAtomicAllocInstruction(resultReg string, size int) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "memory_barrier",
			Operands: []string{},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "load_immediate",
			Operands: []string{fmt.Sprintf("%d", size)},
			Result:   "tmp_size",
			Type:     "uintptr",
		},
		{
			Op:       "call_runtime",
			Operands: []string{"runtime_atomic_alloc", "tmp_size"},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "memory_barrier",
			Operands: []string{},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "null_check",
			Operands: []string{resultReg},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateStatsInstruction generates MIR for getting allocator statistics
func (mi *MIRIntegration) GenerateStatsInstruction(resultReg string) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "call_runtime",
			Operands: []string{"get_runtime_stats"},
			Result:   resultReg,
			Type:     "runtime_stats",
		},
	}
}

// OptimizeInstructions optimizes the generated MIR instructions
func (mi *MIRIntegration) OptimizeInstructions(instructions []MIRInstruction) []MIRInstruction {
	optimized := make([]MIRInstruction, 0, len(instructions))

	for i, instr := range instructions {
		// Skip redundant null checks
		if instr.Op == "null_check" && i > 0 {
			prevInstr := instructions[i-1]
			if prevInstr.Op == "null_check" && len(prevInstr.Operands) > 0 &&
				len(instr.Operands) > 0 && prevInstr.Operands[0] == instr.Operands[0] {
				continue // Skip redundant null check
			}
		}

		// Skip redundant memory barriers
		if instr.Op == "memory_barrier" && i > 0 {
			prevInstr := instructions[i-1]
			if prevInstr.Op == "memory_barrier" {
				continue // Skip redundant memory barrier
			}
		}

		// Combine immediate loads with same values
		if instr.Op == "load_immediate" && i > 0 {
			for j := i - 1; j >= 0; j-- {
				prevInstr := instructions[j]
				if prevInstr.Op == "load_immediate" &&
					len(prevInstr.Operands) > 0 && len(instr.Operands) > 0 &&
					prevInstr.Operands[0] == instr.Operands[0] {
					// Reuse previous register
					instr.Result = prevInstr.Result
					break
				}
			}
		}

		optimized = append(optimized, instr)
	}

	return optimized
}

// GenerateInlinedAlloc generates inlined allocation for small, fixed-size objects
func (mi *MIRIntegration) GenerateInlinedAlloc(resultReg string, size int) []MIRInstruction {
	if size > 64 || mi.allocatorType != ArenaAllocatorKind {
		// Fall back to regular allocation for large objects or non-arena allocators
		return mi.GenerateAllocInstruction(resultReg, size, int(mi.config.AlignmentSize))
	}

	// Inlined arena allocation for small objects
	return []MIRInstruction{
		{
			Op:       "load_global",
			Operands: []string{"global_arena_current"},
			Result:   "current_ptr",
			Type:     "ptr",
		},
		{
			Op:       "load_global",
			Operands: []string{"global_arena_end"},
			Result:   "end_ptr",
			Type:     "ptr",
		},
		{
			Op:       "add",
			Operands: []string{"current_ptr", fmt.Sprintf("%d", size)},
			Result:   "new_ptr",
			Type:     "ptr",
		},
		{
			Op:       "cmp",
			Operands: []string{"new_ptr", "end_ptr"},
			Result:   "cmp_result",
			Type:     "bool",
		},
		{
			Op:       "branch_if",
			Operands: []string{"cmp_result", "slow_alloc"},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "store_global",
			Operands: []string{"global_arena_current", "new_ptr"},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "move",
			Operands: []string{"current_ptr"},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "jump",
			Operands: []string{"alloc_done"},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "label",
			Operands: []string{"slow_alloc"},
			Result:   "",
			Type:     "",
		},
		{
			Op:       "call_runtime",
			Operands: []string{"arena_alloc_slow", fmt.Sprintf("%d", size)},
			Result:   resultReg,
			Type:     "ptr",
		},
		{
			Op:       "label",
			Operands: []string{"alloc_done"},
			Result:   "",
			Type:     "",
		},
	}
}

// GenerateStackAlloc generates stack allocation for temporary objects
func (mi *MIRIntegration) GenerateStackAlloc(resultReg string, size int) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "stack_alloc",
			Operands: []string{fmt.Sprintf("%d", size)},
			Result:   resultReg,
			Type:     "ptr",
		},
	}
}

// GenerateStackFree generates stack deallocation
func (mi *MIRIntegration) GenerateStackFree(ptrReg string, size int) []MIRInstruction {
	return []MIRInstruction{
		{
			Op:       "stack_free",
			Operands: []string{ptrReg, fmt.Sprintf("%d", size)},
			Result:   "",
			Type:     "",
		},
	}
}

// MIRAllocationPattern represents common allocation patterns
type MIRAllocationPattern struct {
	Name         string
	Instructions []MIRInstruction
	Condition    func(size int, usage string) bool
}

// GetOptimalPattern returns the optimal allocation pattern for given parameters
func (mi *MIRIntegration) GetOptimalPattern(size int, usage string) *MIRAllocationPattern {
	patterns := []*MIRAllocationPattern{
		{
			Name: "stack_alloc_small",
			Condition: func(s int, u string) bool {
				return s <= 64 && u == "temporary"
			},
			Instructions: mi.GenerateStackAlloc("result", size),
		},
		{
			Name: "inlined_arena_small",
			Condition: func(s int, u string) bool {
				return s <= 64 && mi.allocatorType == ArenaAllocatorKind
			},
			Instructions: mi.GenerateInlinedAlloc("result", size),
		},
		{
			Name: "pool_alloc_medium",
			Condition: func(s int, u string) bool {
				return s <= 1024 && mi.allocatorType == PoolAllocatorKind
			},
			Instructions: mi.GenerateAllocInstruction("result", size, int(mi.config.AlignmentSize)),
		},
	}

	for _, pattern := range patterns {
		if pattern.Condition(size, usage) {
			return pattern
		}
	}

	// Default pattern
	return &MIRAllocationPattern{
		Name:         "default_alloc",
		Instructions: mi.GenerateAllocInstruction("result", size, int(mi.config.AlignmentSize)),
		Condition:    nil,
	}
}

// GenerateCompleteAllocationSequence generates a complete allocation sequence with error handling
func (mi *MIRIntegration) GenerateCompleteAllocationSequence(resultReg string, size int, errorLabel string) []MIRInstruction {
	instructions := mi.GenerateAllocInstruction(resultReg, size, int(mi.config.AlignmentSize))

	// Add error handling
	instructions = append(instructions, []MIRInstruction{
		{
			Op:       "cmp_null",
			Operands: []string{resultReg},
			Result:   "is_null",
			Type:     "bool",
		},
		{
			Op:       "branch_if",
			Operands: []string{"is_null", errorLabel},
			Result:   "",
			Type:     "",
		},
	}...)

	return instructions
}
