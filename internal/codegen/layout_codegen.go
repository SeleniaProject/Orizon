// Package codegen provides layout-aware code generation for Orizon's data structures.
// This extends the existing code generation to handle arrays, slices, strings, and structs
// with proper memory layout considerations.
package codegen

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/layout"
	"github.com/orizon-lang/orizon/internal/lir"
)

// LayoutAwareEmitter extends the existing emitter with layout support
type LayoutAwareEmitter struct {
	Calculator  *layout.LayoutCalculator
	TypeLayouts map[string]*layout.MemoryLayout // Cache of computed layouts
}

// NewLayoutAwareEmitter creates a new layout-aware emitter
func NewLayoutAwareEmitter() *LayoutAwareEmitter {
	return &LayoutAwareEmitter{
		Calculator:  layout.NewLayoutCalculator(),
		TypeLayouts: make(map[string]*layout.MemoryLayout),
	}
}

// EmitWithLayouts generates x64 assembly with proper data structure layouts
func (lae *LayoutAwareEmitter) EmitWithLayouts(m *lir.Module) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "; module %s (layout-aware)\n", m.Name)

	// Generate data structure definitions
	layoutDefs, err := lae.generateLayoutDefinitions(m)
	if err != nil {
		return "", fmt.Errorf("failed to generate layout definitions: %w", err)
	}
	b.WriteString(layoutDefs)

	// Generate functions with layout awareness
	for _, f := range m.Functions {
		funcAsm, err := lae.emitLayoutAwareFunction(f)
		if err != nil {
			return "", fmt.Errorf("failed to emit function %s: %w", f.Name, err)
		}
		b.WriteString(funcAsm)
	}

	return b.String(), nil
}

// generateLayoutDefinitions creates assembly definitions for data structures
func (lae *LayoutAwareEmitter) generateLayoutDefinitions(m *lir.Module) (string, error) {
	var b strings.Builder

	b.WriteString("\n; Data structure layout definitions\n")

	// Standard layouts
	sliceLayout, _ := lae.Calculator.CalculateSliceLayout("generic", 8, 8)
	stringLayout := lae.Calculator.CalculateStringLayout()

	b.WriteString(fmt.Sprintf("; Slice header: %d bytes (ptr+len+cap)\n", sliceLayout.TotalSize))
	b.WriteString(fmt.Sprintf("; String header: %d bytes (ptr+len)\n", stringLayout.TotalSize))
	b.WriteString("\n")

	return b.String(), nil
}

// emitLayoutAwareFunction generates function assembly with layout considerations
func (lae *LayoutAwareEmitter) emitLayoutAwareFunction(f *lir.Function) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s:\n", f.Name))
	b.WriteString("  push rbp\n")
	b.WriteString("  mov rbp, rsp\n")

	// Analyze function for layout requirements
	layoutInfo := lae.analyzeLayoutRequirements(f)

	// Allocate stack space for local data structures
	stackSize := lae.calculateStackRequirements(layoutInfo)
	if stackSize > 0 {
		b.WriteString(fmt.Sprintf("  sub rsp, %d  ; layout stack space\n", stackSize))
	}

	// Emit basic blocks with layout awareness
	for _, bb := range f.Blocks {
		if bb.Label != "" {
			b.WriteString(fmt.Sprintf("%s:\n", bb.Label))
		}

		for _, instr := range bb.Insns {
			instrAsm, err := lae.emitLayoutAwareInstruction(instr, layoutInfo)
			if err != nil {
				return "", fmt.Errorf("failed to emit instruction: %w", err)
			}
			b.WriteString(instrAsm)
		}
	}

	// Function epilogue
	if stackSize > 0 {
		b.WriteString(fmt.Sprintf("  add rsp, %d\n", stackSize))
	}
	b.WriteString("  pop rbp\n")
	b.WriteString("  ret\n\n")

	return b.String(), nil
}

// LayoutRequirement represents layout requirements for a function
type LayoutRequirement struct {
	LocalArrays  map[string]*layout.ArrayLayout
	LocalSlices  map[string]*layout.SliceLayout
	LocalStructs map[string]*layout.StructLayout
	StackOffset  map[string]int64 // Variable to stack offset mapping
}

// analyzeLayoutRequirements examines a function for data structure usage
func (lae *LayoutAwareEmitter) analyzeLayoutRequirements(f *lir.Function) *LayoutRequirement {
	req := &LayoutRequirement{
		LocalArrays:  make(map[string]*layout.ArrayLayout),
		LocalSlices:  make(map[string]*layout.SliceLayout),
		LocalStructs: make(map[string]*layout.StructLayout),
		StackOffset:  make(map[string]int64),
	}

	// Scan instructions for layout-sensitive operations
	for _, bb := range f.Blocks {
		for _, instr := range bb.Insns {
			lae.analyzeInstructionForLayout(instr, req)
		}
	}

	return req
}

// analyzeInstructionForLayout examines an instruction for layout requirements
func (lae *LayoutAwareEmitter) analyzeInstructionForLayout(instr lir.Insn, req *LayoutRequirement) {
	switch inst := instr.(type) {
	case lir.Alloc:
		// Check if allocation needs specific layout
		if strings.Contains(inst.Name, "array") {
			// Example: alloc array_10_i32 -> array of 10 i32s
			if arrayLayout, err := lae.parseArrayAlloc(inst.Name); err == nil {
				req.LocalArrays[inst.Dst] = arrayLayout
			}
		} else if strings.Contains(inst.Name, "slice") {
			// Example: alloc slice_i32 -> slice of i32s
			if sliceLayout, err := lae.parseSliceAlloc(inst.Name); err == nil {
				req.LocalSlices[inst.Dst] = sliceLayout
			}
		}
	case lir.Load, lir.Store:
		// Track memory operations that might need layout awareness
		// Implementation depends on type information in LIR
	}
}

// calculateStackRequirements computes total stack space needed
func (lae *LayoutAwareEmitter) calculateStackRequirements(req *LayoutRequirement) int64 {
	var totalSize int64
	currentOffset := int64(0)

	// Allocate space for arrays
	for dst, arrayLayout := range req.LocalArrays {
		// Align for array
		currentOffset = alignUp(currentOffset, arrayLayout.ElementAlign)
		req.StackOffset[dst] = currentOffset
		currentOffset += arrayLayout.TotalSize
		totalSize = currentOffset
	}

	// Allocate space for slice headers
	for dst, sliceLayout := range req.LocalSlices {
		currentOffset = alignUp(currentOffset, 8) // Pointer alignment
		req.StackOffset[dst] = currentOffset
		currentOffset += sliceLayout.TotalSize
		totalSize = currentOffset
	}

	// Allocate space for struct instances
	for dst, structLayout := range req.LocalStructs {
		currentOffset = alignUp(currentOffset, structLayout.Alignment)
		req.StackOffset[dst] = currentOffset
		currentOffset += structLayout.TotalSize
		totalSize = currentOffset
	}

	// Align final stack size to 16 bytes
	return alignUp(totalSize, 16)
}

// emitLayoutAwareInstruction generates assembly for layout-sensitive instructions
func (lae *LayoutAwareEmitter) emitLayoutAwareInstruction(instr lir.Insn, req *LayoutRequirement) (string, error) {
	switch inst := instr.(type) {
	case lir.Alloc:
		return lae.emitLayoutAwareAlloc(inst, req)
	case lir.Load:
		return lae.emitLayoutAwareLoad(inst, req)
	case lir.Store:
		return lae.emitLayoutAwareStore(inst, req)
	case lir.Call:
		return lae.emitLayoutAwareCall(inst, req)
	default:
		// Fall back to basic emission for non-layout-sensitive instructions
		return lae.emitBasicInstruction(instr)
	}
}

// emitLayoutAwareAlloc generates optimized allocation code
func (lae *LayoutAwareEmitter) emitLayoutAwareAlloc(inst lir.Alloc, req *LayoutRequirement) (string, error) {
	var b strings.Builder

	if offset, hasOffset := req.StackOffset[inst.Dst]; hasOffset {
		// Stack allocation with known layout
		b.WriteString(fmt.Sprintf("  lea rax, [rbp-%d]  ; %s (%s)\n", offset, inst.Dst, inst.Name))

		// Initialize based on layout type
		if arrayLayout, isArray := req.LocalArrays[inst.Dst]; isArray {
			b.WriteString(fmt.Sprintf("  ; array initialization: %d elements of %d bytes\n",
				arrayLayout.Length, arrayLayout.ElementSize))
			// Could add zero-initialization here if needed
		} else if _, isSlice := req.LocalSlices[inst.Dst]; isSlice {
			b.WriteString("  ; slice header initialization\n")
			b.WriteString("  mov qword ptr [rax], 0     ; data ptr = null\n")
			b.WriteString("  mov qword ptr [rax+8], 0   ; len = 0\n")
			b.WriteString("  mov qword ptr [rax+16], 0  ; cap = 0\n")
		}
	} else {
		// Generic allocation
		b.WriteString(fmt.Sprintf("  ; generic alloc %s -> %s\n", inst.Name, inst.Dst))
	}

	return b.String(), nil
}

// emitLayoutAwareLoad generates optimized load instructions
func (lae *LayoutAwareEmitter) emitLayoutAwareLoad(inst lir.Load, req *LayoutRequirement) (string, error) {
	// Check if loading from a known layout structure
	if lae.isSliceAccess(inst.Addr) {
		return lae.emitSliceElementLoad(inst, req)
	} else if lae.isArrayAccess(inst.Addr) {
		return lae.emitArrayElementLoad(inst, req)
	}

	// Default load
	return fmt.Sprintf("  mov rax, qword ptr [%s]  ; %s = load %s\n", inst.Addr, inst.Dst, inst.Addr), nil
}

// emitLayoutAwareStore generates optimized store instructions
func (lae *LayoutAwareEmitter) emitLayoutAwareStore(inst lir.Store, req *LayoutRequirement) (string, error) {
	// Check if storing to a known layout structure
	if lae.isSliceAccess(inst.Addr) {
		return lae.emitSliceElementStore(inst, req)
	} else if lae.isArrayAccess(inst.Addr) {
		return lae.emitArrayElementStore(inst, req)
	}

	// Default store
	return fmt.Sprintf("  mov qword ptr [%s], %s  ; store %s, %s\n", inst.Addr, inst.Val, inst.Addr, inst.Val), nil
}

// emitLayoutAwareCall handles function calls with layout considerations
func (lae *LayoutAwareEmitter) emitLayoutAwareCall(inst lir.Call, req *LayoutRequirement) (string, error) {
	var b strings.Builder

	// Special handling for layout-related function calls
	if strings.HasPrefix(inst.Callee, "slice_") {
		return lae.emitSliceOperation(inst, req)
	} else if strings.HasPrefix(inst.Callee, "array_") {
		return lae.emitArrayOperation(inst, req)
	} else if strings.HasPrefix(inst.Callee, "string_") {
		return lae.emitStringOperation(inst, req)
	}

	// Standard function call with Win64 ABI
	b.WriteString("  sub rsp, 32  ; shadow space\n")

	// Load arguments (simplified)
	for i, arg := range inst.Args {
		if i < 4 {
			regs := []string{"rcx", "rdx", "r8", "r9"}
			b.WriteString(fmt.Sprintf("  mov %s, %s\n", regs[i], arg))
		}
	}

	b.WriteString(fmt.Sprintf("  call %s\n", inst.Callee))
	b.WriteString("  add rsp, 32\n")

	if inst.Dst != "" {
		b.WriteString(fmt.Sprintf("  mov %s, rax  ; %s = return value\n", inst.Dst, inst.Dst))
	}

	return b.String(), nil
}

// emitBasicInstruction handles non-layout-sensitive instructions
func (lae *LayoutAwareEmitter) emitBasicInstruction(instr lir.Insn) (string, error) {
	switch inst := instr.(type) {
	case lir.Mov:
		return fmt.Sprintf("  mov %s, %s\n", inst.Dst, inst.Src), nil
	case lir.Add:
		return fmt.Sprintf("  mov %s, %s\n  add %s, %s\n", inst.Dst, inst.LHS, inst.Dst, inst.RHS), nil
	case lir.Sub:
		return fmt.Sprintf("  mov %s, %s\n  sub %s, %s\n", inst.Dst, inst.LHS, inst.Dst, inst.RHS), nil
	case lir.Ret:
		if inst.Src != "" {
			return fmt.Sprintf("  mov rax, %s\n", inst.Src), nil
		}
		return "", nil
	default:
		return fmt.Sprintf("  ; unsupported instruction: %s\n", instr.Op()), nil
	}
}

// Helper functions for parsing layout information

func (lae *LayoutAwareEmitter) parseArrayAlloc(name string) (*layout.ArrayLayout, error) {
	// Parse "array_10_i32" -> length=10, element=i32
	parts := strings.Split(name, "_")
	if len(parts) < 3 || parts[0] != "array" {
		return nil, fmt.Errorf("invalid array allocation name: %s", name)
	}

	// For this example, assume length=10, elementSize=4, align=4
	return lae.Calculator.CalculateArrayLayout("i32", 4, 4, 10)
}

func (lae *LayoutAwareEmitter) parseSliceAlloc(name string) (*layout.SliceLayout, error) {
	// Parse "slice_i32" -> element=i32
	parts := strings.Split(name, "_")
	if len(parts) < 2 || parts[0] != "slice" {
		return nil, fmt.Errorf("invalid slice allocation name: %s", name)
	}

	// For this example, assume elementSize=4, align=4
	return lae.Calculator.CalculateSliceLayout("i32", 4, 4)
}

func (lae *LayoutAwareEmitter) isSliceAccess(addr string) bool {
	return strings.Contains(addr, "slice") || strings.Contains(addr, "[") && strings.Contains(addr, "+8")
}

func (lae *LayoutAwareEmitter) isArrayAccess(addr string) bool {
	return strings.Contains(addr, "array") || strings.Contains(addr, "[") && !strings.Contains(addr, "+8")
}

// Layout-specific emission functions

func (lae *LayoutAwareEmitter) emitSliceElementLoad(inst lir.Load, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; slice element load: %s = load %s\n  mov rax, qword ptr [%s]\n",
		inst.Dst, inst.Addr, inst.Addr), nil
}

func (lae *LayoutAwareEmitter) emitArrayElementLoad(inst lir.Load, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; array element load: %s = load %s\n  mov rax, qword ptr [%s]\n",
		inst.Dst, inst.Addr, inst.Addr), nil
}

func (lae *LayoutAwareEmitter) emitSliceElementStore(inst lir.Store, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; slice element store: store %s, %s\n  mov qword ptr [%s], %s\n",
		inst.Addr, inst.Val, inst.Addr, inst.Val), nil
}

func (lae *LayoutAwareEmitter) emitArrayElementStore(inst lir.Store, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; array element store: store %s, %s\n  mov qword ptr [%s], %s\n",
		inst.Addr, inst.Val, inst.Addr, inst.Val), nil
}

func (lae *LayoutAwareEmitter) emitSliceOperation(inst lir.Call, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; slice operation: %s\n  call %s\n", inst.Callee, inst.Callee), nil
}

func (lae *LayoutAwareEmitter) emitArrayOperation(inst lir.Call, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; array operation: %s\n  call %s\n", inst.Callee, inst.Callee), nil
}

func (lae *LayoutAwareEmitter) emitStringOperation(inst lir.Call, req *LayoutRequirement) (string, error) {
	return fmt.Sprintf("  ; string operation: %s\n  call %s\n", inst.Callee, inst.Callee), nil
}

// alignUp is a local utility function
func alignUp(value, alignment int64) int64 {
	if alignment <= 1 {
		return value
	}
	return (value + alignment - 1) & ^(alignment - 1)
}
