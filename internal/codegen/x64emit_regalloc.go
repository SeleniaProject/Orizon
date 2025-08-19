// Package codegen provides enhanced x64 code generation with full register allocation.
// This replaces the naive stack-slot-only approach with proper register allocation
// using the regalloc package for optimal register utilization.
package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/orizon-lang/orizon/internal/codegen/regalloc"
	"github.com/orizon-lang/orizon/internal/lir"
)

const scratchXMMRegAlloc = "xmm7" // スタック上の浮動小数引数退避に利用（非callee-saved、テストもこれを前提）

// EmitX64WithRegisterAllocation emits optimized x64 assembly using full register allocation
func EmitX64WithRegisterAllocation(m *lir.Module) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "; module %s (with register allocation)\n", m.Name)

	for _, f := range m.Functions {
		asm, err := emitFuncWithRegAlloc(&b, f)
		if err != nil {
			return "", fmt.Errorf("failed to emit function %s: %w", f.Name, err)
		}
		b.WriteString(asm)
	}

	return b.String(), nil
}

// emitFuncWithRegAlloc generates assembly for a function using register allocation
func emitFuncWithRegAlloc(b *strings.Builder, f *lir.Function) (string, error) {
	var funcBuilder strings.Builder

	// Perform register allocation
	allocator := regalloc.NewRegisterAllocator(f)
	if err := allocator.AllocateRegisters(); err != nil {
		return "", fmt.Errorf("register allocation failed: %w", err)
	}

	// Calculate frame size including spill slots
	spillSlots := allocator.GetTotalSpillSlots()
	frameSize := int64(spillSlots * 8) // Each spill slot is 8 bytes

	// Align frame to 16 bytes for better call alignment
	if rem := frameSize % 16; rem != 0 {
		frameSize += 16 - rem
	}

	// Function prologue
	funcBuilder.WriteString(fmt.Sprintf("%s:\n", f.Name))
	funcBuilder.WriteString("  push rbp\n")
	funcBuilder.WriteString("  mov rbp, rsp\n")

	// Save callee-saved registers that were allocated
	savedRegs := getSavedRegisters(allocator)
	for _, reg := range savedRegs {
		funcBuilder.WriteString(fmt.Sprintf("  push %s\n", reg))
		frameSize += 8 // Account for saved register
	}

	if frameSize > 0 {
		funcBuilder.WriteString(fmt.Sprintf("  sub rsp, %d\n", frameSize))
	}

	// Emit blocks with register-allocated instructions
	for _, bb := range f.Blocks {
		if bb.Label != "" {
			funcBuilder.WriteString(fmt.Sprintf("%s:\n", bb.Label))
		}

		for _, instr := range bb.Insns {
			instrAsm, err := emitInstructionWithRegAlloc(instr, allocator)
			if err != nil {
				return "", fmt.Errorf("failed to emit instruction %v: %w", instr, err)
			}
			funcBuilder.WriteString(instrAsm)
		}
	}

	// Function epilogue
	if frameSize > 0 {
		funcBuilder.WriteString(fmt.Sprintf("  add rsp, %d\n", frameSize))
	}

	// Restore callee-saved registers in reverse order
	for i := len(savedRegs) - 1; i >= 0; i-- {
		funcBuilder.WriteString(fmt.Sprintf("  pop %s\n", savedRegs[i]))
	}

	funcBuilder.WriteString("  pop rbp\n")
	funcBuilder.WriteString("  ret\n\n")

	// Add allocation debug information as comments
	funcBuilder.WriteString("; Register Allocation Summary:\n")
	allocSummary := allocator.PrintAllocationResults()
	for _, line := range strings.Split(allocSummary, "\n") {
		if line != "" {
			funcBuilder.WriteString(fmt.Sprintf("; %s\n", line))
		}
	}
	funcBuilder.WriteString("\n")

	return funcBuilder.String(), nil
}

// getSavedRegisters returns the list of callee-saved registers that were allocated
func getSavedRegisters(allocator *regalloc.RegisterAllocator) []string {
	usedCalleeSaved := make(map[string]bool)

	// Check all allocations for callee-saved registers
	for virtualReg := range allocator.GetSpillSlots() {
		if alloc, exists := allocator.GetAllocation(virtualReg); exists {
			if alloc.Type == regalloc.AllocRegister && alloc.Register.CalleeSaved {
				usedCalleeSaved[alloc.Register.Name] = true
			}
		}
	}

	// Return sorted list for deterministic output
	var regs []string
	for reg := range usedCalleeSaved {
		regs = append(regs, reg)
	}
	sort.Strings(regs)

	return regs
}

// emitInstructionWithRegAlloc generates assembly for a single instruction using register allocation
func emitInstructionWithRegAlloc(instr lir.Insn, allocator *regalloc.RegisterAllocator) (string, error) {
	switch inst := instr.(type) {
	case lir.Mov:
		return emitMov(inst, allocator)
	case lir.Add:
		return emitBinaryOp(inst.Dst, inst.LHS, inst.RHS, "add", allocator)
	case lir.Sub:
		return emitBinaryOp(inst.Dst, inst.LHS, inst.RHS, "sub", allocator)
	case lir.Mul:
		return emitBinaryOp(inst.Dst, inst.LHS, inst.RHS, "imul", allocator)
	case lir.Div:
		return emitDiv(inst, allocator)
	case lir.Load:
		return emitLoad(inst, allocator)
	case lir.Store:
		return emitStore(inst, allocator)
	case lir.Cmp:
		return emitCmp(inst, allocator)
	case lir.Br:
		return fmt.Sprintf("  jmp %s\n", inst.Target), nil
	case lir.BrCond:
		return emitBrCond(inst, allocator)
	case lir.Call:
		return emitCall(inst, allocator)
	case lir.Ret:
		return emitRet(inst, allocator)
	case lir.Alloc:
		// Alloca is handled during register allocation - just emit a comment
		return fmt.Sprintf("  ; alloca %s -> %s\n", inst.Name, inst.Dst), nil
	default:
		// Unknown instruction - emit as comment
		if s, ok := any(instr).(fmt.Stringer); ok {
			return fmt.Sprintf("  ; unknown: %s\n", s.String()), nil
		}
		return fmt.Sprintf("  ; unknown op %s\n", instr.Op()), nil
	}
}

// emitMov generates a move instruction with register allocation
func emitMov(inst lir.Mov, allocator *regalloc.RegisterAllocator) (string, error) {
	src := resolveLocation(inst.Src, allocator)
	dst := resolveLocation(inst.Dst, allocator)

	if src == dst {
		return "  ; nop (src == dst)\n", nil
	}

	// Generate appropriate move instruction
	if isMemoryLocation(src) && isMemoryLocation(dst) {
		// Memory to memory - use scratch register
		return fmt.Sprintf("  mov rax, %s\n  mov %s, rax\n", src, dst), nil
	} else {
		return fmt.Sprintf("  mov %s, %s\n", dst, src), nil
	}
}

// emitBinaryOp generates binary arithmetic operations with register allocation
func emitBinaryOp(dst, lhs, rhs, op string, allocator *regalloc.RegisterAllocator) (string, error) {
	dstLoc := resolveLocation(dst, allocator)
	lhsLoc := resolveLocation(lhs, allocator)
	rhsLoc := resolveLocation(rhs, allocator)

	var result strings.Builder

	// Load LHS into destination or temporary register
	if dstLoc != lhsLoc {
		if isMemoryLocation(lhsLoc) && isMemoryLocation(dstLoc) {
			// Both memory - use scratch register
			result.WriteString(fmt.Sprintf("  mov rax, %s\n", lhsLoc))
			result.WriteString(fmt.Sprintf("  %s rax, %s\n", op, rhsLoc))
			result.WriteString(fmt.Sprintf("  mov %s, rax\n", dstLoc))
		} else {
			result.WriteString(fmt.Sprintf("  mov %s, %s\n", dstLoc, lhsLoc))
			result.WriteString(fmt.Sprintf("  %s %s, %s\n", op, dstLoc, rhsLoc))
		}
	} else {
		// Destination already has LHS
		result.WriteString(fmt.Sprintf("  %s %s, %s\n", op, dstLoc, rhsLoc))
	}

	return result.String(), nil
}

// emitDiv generates division instruction with special handling for x64 requirements
func emitDiv(inst lir.Div, allocator *regalloc.RegisterAllocator) (string, error) {
	dstLoc := resolveLocation(inst.Dst, allocator)
	lhsLoc := resolveLocation(inst.LHS, allocator)
	rhsLoc := resolveLocation(inst.RHS, allocator)

	var result strings.Builder

	// Division requires RAX as dividend and RDX for remainder
	result.WriteString(fmt.Sprintf("  mov rax, %s\n", lhsLoc))
	result.WriteString("  cqo\n") // Sign-extend RAX to RDX:RAX

	if rhsLoc == "rdx" {
		// Divisor is in RDX which we need for remainder - use scratch
		result.WriteString("  mov r10, rdx\n")
		result.WriteString("  idiv r10\n")
	} else {
		result.WriteString(fmt.Sprintf("  idiv %s\n", rhsLoc))
	}

	// Move result from RAX to destination
	if dstLoc != "rax" {
		result.WriteString(fmt.Sprintf("  mov %s, rax\n", dstLoc))
	}

	return result.String(), nil
}

// emitLoad generates load instruction with register allocation
func emitLoad(inst lir.Load, allocator *regalloc.RegisterAllocator) (string, error) {
	dstLoc := resolveLocation(inst.Dst, allocator)
	addrLoc := resolveLocation(inst.Addr, allocator)

	// Handle different addressing modes
	if isImmediate(inst.Addr) {
		return fmt.Sprintf("  mov %s, %s\n", dstLoc, inst.Addr), nil
	} else if isMemoryLocation(addrLoc) {
		// Address is in memory - load address first, then dereference
		return fmt.Sprintf("  mov rax, %s\n  mov %s, qword ptr [rax]\n", addrLoc, dstLoc), nil
	} else {
		// Address is in register - direct dereference
		return fmt.Sprintf("  mov %s, qword ptr [%s]\n", dstLoc, addrLoc), nil
	}
}

// emitStore generates store instruction with register allocation
func emitStore(inst lir.Store, allocator *regalloc.RegisterAllocator) (string, error) {
	addrLoc := resolveLocation(inst.Addr, allocator)
	valLoc := resolveLocation(inst.Val, allocator)

	if isMemoryLocation(addrLoc) {
		// Address is in memory - load address first
		if isMemoryLocation(valLoc) {
			// Both in memory - use two scratch registers
			return fmt.Sprintf("  mov rax, %s\n  mov r10, %s\n  mov qword ptr [rax], r10\n", addrLoc, valLoc), nil
		} else {
			return fmt.Sprintf("  mov rax, %s\n  mov qword ptr [rax], %s\n", addrLoc, valLoc), nil
		}
	} else {
		// Address is in register
		return fmt.Sprintf("  mov qword ptr [%s], %s\n", addrLoc, valLoc), nil
	}
}

// emitCmp generates comparison instruction with register allocation
func emitCmp(inst lir.Cmp, allocator *regalloc.RegisterAllocator) (string, error) {
	dstLoc := resolveLocation(inst.Dst, allocator)
	lhsLoc := resolveLocation(inst.LHS, allocator)
	rhsLoc := resolveLocation(inst.RHS, allocator)

	var result strings.Builder

	// Perform comparison
	if isMemoryLocation(lhsLoc) && isMemoryLocation(rhsLoc) {
		// Both in memory - load one to register
		result.WriteString(fmt.Sprintf("  mov rax, %s\n", lhsLoc))
		result.WriteString(fmt.Sprintf("  cmp rax, %s\n", rhsLoc))
	} else {
		result.WriteString(fmt.Sprintf("  cmp %s, %s\n", lhsLoc, rhsLoc))
	}

	// Map predicate to setcc instruction
	setcc := mapCmpToSetccRegAlloc(inst.Pred)
	result.WriteString(fmt.Sprintf("  %s al\n", setcc))
	result.WriteString("  movzx rax, al\n")

	// Move result to destination if not RAX
	if dstLoc != "rax" {
		result.WriteString(fmt.Sprintf("  mov %s, rax\n", dstLoc))
	}

	return result.String(), nil
}

// emitBrCond generates conditional branch with register allocation
func emitBrCond(inst lir.BrCond, allocator *regalloc.RegisterAllocator) (string, error) {
	condLoc := resolveLocation(inst.Cond, allocator)

	var result strings.Builder

	// Test condition (0 = false, non-zero = true)
	if condLoc == "rax" {
		result.WriteString("  test rax, rax\n")
	} else {
		result.WriteString(fmt.Sprintf("  cmp %s, 0\n", condLoc))
	}

	result.WriteString(fmt.Sprintf("  jnz %s\n", inst.True))
	result.WriteString(fmt.Sprintf("  jmp %s\n", inst.False))

	return result.String(), nil
}

// emitCall generates function call with register allocation and Win64 ABI
func emitCall(inst lir.Call, allocator *regalloc.RegisterAllocator) (string, error) {
	var result strings.Builder

	// Win64 calling convention: RCX, RDX, R8, R9 for integer args
	// XMM0-XMM3 for floating point args
	gprRegs := []string{"rcx", "rdx", "r8", "r9"}
	xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3"}

	// Calculate stack space needed for arguments
	stackArgs := 0
	if len(inst.Args) > 4 {
		stackArgs = len(inst.Args) - 4
	}

	// Reserve shadow space (32 bytes) + stack arguments, aligned to 16 bytes
	reserve := int64(32 + stackArgs*8)
	if rem := reserve % 16; rem != 0 {
		reserve += 16 - rem
	}

	if reserve > 0 {
		result.WriteString(fmt.Sprintf("  sub rsp, %d\n", reserve))
	}

	// Place stack arguments (beyond 4th argument)
	for i := 4; i < len(inst.Args); i++ {
		offset := 32 + (i-4)*8
		argLoc := resolveLocation(inst.Args[i], allocator)

		cls := ""
		if i < len(inst.ArgClasses) {
			cls = inst.ArgClasses[i]
		}

		if cls == "f32" || cls == "f64" {
			// Floating point stack argument
			if isMemoryLocation(argLoc) {
				result.WriteString(fmt.Sprintf("  mov rax, %s\n", argLoc))
				result.WriteString(fmt.Sprintf("  movq %s, rax\n", scratchXMMRegAlloc))
			} else {
				result.WriteString(fmt.Sprintf("  movq %s, %s\n", scratchXMMRegAlloc, argLoc))
			}

			if cls == "f32" {
				result.WriteString(fmt.Sprintf("  movss dword ptr [rsp+%d], %s\n", offset, scratchXMMRegAlloc))
			} else {
				result.WriteString(fmt.Sprintf("  movsd qword ptr [rsp+%d], %s\n", offset, scratchXMMRegAlloc))
			}
		} else {
			// Integer stack argument
			result.WriteString(fmt.Sprintf("  mov qword ptr [rsp+%d], %s\n", offset, argLoc))
		}
	}

	// Load first 4 arguments into registers
	gprIndex := 0
	xmmIndex := 0

	for i := 0; i < len(inst.Args) && i < 4; i++ {
		argLoc := resolveLocation(inst.Args[i], allocator)

		cls := ""
		if i < len(inst.ArgClasses) {
			cls = inst.ArgClasses[i]
		}

		if cls == "f32" || cls == "f64" {
			// Floating point argument
			if xmmIndex < len(xmmRegs) {
				targetReg := xmmRegs[xmmIndex]
				if isMemoryLocation(argLoc) {
					result.WriteString(fmt.Sprintf("  mov rax, %s\n", argLoc))
					result.WriteString(fmt.Sprintf("  movq %s, rax\n", targetReg))
				} else {
					result.WriteString(fmt.Sprintf("  movq %s, %s\n", targetReg, argLoc))
				}
				xmmIndex++
			}
		} else {
			// Integer argument
			if gprIndex < len(gprRegs) {
				targetReg := gprRegs[gprIndex]
				if argLoc != targetReg {
					result.WriteString(fmt.Sprintf("  mov %s, %s\n", targetReg, argLoc))
				}
				gprIndex++
			}
		}
	}

	// Perform the call
	result.WriteString(fmt.Sprintf("  call %s\n", inst.Callee))

	// Restore stack pointer
	if reserve > 0 {
		result.WriteString(fmt.Sprintf("  add rsp, %d\n", reserve))
	}

	// Move return value to destination if needed
	if inst.Dst != "" {
		dstLoc := resolveLocation(inst.Dst, allocator)

		if inst.RetClass == "f32" || inst.RetClass == "f64" {
			// Floating point return value in XMM0
			if dstLoc != "xmm0" {
				result.WriteString(fmt.Sprintf("  movq %s, xmm0\n", dstLoc))
			}
		} else {
			// Integer return value in RAX
			if dstLoc != "rax" {
				result.WriteString(fmt.Sprintf("  mov %s, rax\n", dstLoc))
			}
		}
	}

	return result.String(), nil
}

// emitRet generates return instruction with register allocation
func emitRet(inst lir.Ret, allocator *regalloc.RegisterAllocator) (string, error) {
	if inst.Src != "" {
		srcLoc := resolveLocation(inst.Src, allocator)
		if srcLoc != "rax" {
			return fmt.Sprintf("  mov rax, %s\n", srcLoc), nil
		}
	}
	return "", nil // Return handled by function epilogue
}

// resolveLocation converts a virtual register or value to its allocated location
func resolveLocation(operand string, allocator *regalloc.RegisterAllocator) string {
	if operand == "" {
		return ""
	}

	// Check if it's a virtual register that has been allocated
	if strings.HasPrefix(operand, "%") {
		if alloc, exists := allocator.GetAllocation(operand); exists {
			switch alloc.Type {
			case regalloc.AllocRegister:
				return alloc.Register.Name
			case regalloc.AllocSpill:
				return fmt.Sprintf("qword ptr [rbp-%d]", alloc.SpillSlot)
			}
		}
		// Fallback for unallocated virtual registers
		return fmt.Sprintf("qword ptr [rbp-8] ; unallocated %s", operand)
	}

	// Physical register or immediate value
	return operand
}

// isMemoryLocation checks if a location string represents a memory reference
func isMemoryLocation(loc string) bool {
	return strings.Contains(loc, "[") && strings.Contains(loc, "]")
}

// isImmediate checks if an operand is an immediate value
func isImmediate(operand string) bool {
	_, err := strconv.ParseInt(operand, 10, 64)
	return err == nil
}

// mapCmpToSetccRegAlloc maps LIR comparison predicates to x64 setcc instructions
func mapCmpToSetccRegAlloc(pred string) string {
	switch pred {
	case "eq":
		return "sete"
	case "ne":
		return "setne"
	case "slt":
		return "setl"
	case "sle":
		return "setle"
	case "sgt":
		return "setg"
	case "sge":
		return "setge"
	case "ult":
		return "setb"
	case "ule":
		return "setbe"
	case "ugt":
		return "seta"
	case "uge":
		return "setae"
	default:
		return "sete" // Default fallback
	}
}
