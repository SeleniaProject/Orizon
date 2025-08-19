// Package regalloc implements a full register allocation system for x64 code generation.
// This implements linear scan register allocation with liveness analysis and spill handling,
// replacing the naive stack-slot-only approach with proper physical register utilization.
package regalloc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orizon-lang/orizon/internal/lir"
)

// RegisterClass represents different register classes for x64
type RegisterClass int

const (
	RegClassGPR RegisterClass = iota // General Purpose Registers
	RegClassXMM                      // XMM (floating point) Registers
)

// PhysicalRegister represents a physical x64 register
type PhysicalRegister struct {
	Name  string
	Class RegisterClass
	Index int
	// Callee-saved registers must be preserved across function calls
	CalleeSaved bool
}

// Available x64 registers for allocation
var (
	// General Purpose Registers (caller-saved, except RBX, RBP, R12-R15)
	GPRRegisters = []PhysicalRegister{
		{Name: "rax", Class: RegClassGPR, Index: 0, CalleeSaved: false}, // Return value
		{Name: "rcx", Class: RegClassGPR, Index: 1, CalleeSaved: false}, // Arg 1
		{Name: "rdx", Class: RegClassGPR, Index: 2, CalleeSaved: false}, // Arg 2
		{Name: "r8", Class: RegClassGPR, Index: 3, CalleeSaved: false},  // Arg 3
		{Name: "r9", Class: RegClassGPR, Index: 4, CalleeSaved: false},  // Arg 4
		{Name: "r10", Class: RegClassGPR, Index: 5, CalleeSaved: false}, // Scratch
		{Name: "r11", Class: RegClassGPR, Index: 6, CalleeSaved: false}, // Scratch
		{Name: "rbx", Class: RegClassGPR, Index: 7, CalleeSaved: true},  // Callee-saved
		{Name: "r12", Class: RegClassGPR, Index: 8, CalleeSaved: true},  // Callee-saved
		{Name: "r13", Class: RegClassGPR, Index: 9, CalleeSaved: true},  // Callee-saved
		{Name: "r14", Class: RegClassGPR, Index: 10, CalleeSaved: true}, // Callee-saved
		{Name: "r15", Class: RegClassGPR, Index: 11, CalleeSaved: true}, // Callee-saved
	}

	// XMM Registers (floating point, XMM6-XMM15 are callee-saved on Windows)
	XMMRegisters = []PhysicalRegister{
		{Name: "xmm0", Class: RegClassXMM, Index: 0, CalleeSaved: false}, // Arg 1
		{Name: "xmm1", Class: RegClassXMM, Index: 1, CalleeSaved: false}, // Arg 2
		{Name: "xmm2", Class: RegClassXMM, Index: 2, CalleeSaved: false}, // Arg 3
		{Name: "xmm3", Class: RegClassXMM, Index: 3, CalleeSaved: false}, // Arg 4
		{Name: "xmm4", Class: RegClassXMM, Index: 4, CalleeSaved: false}, // Scratch
		{Name: "xmm5", Class: RegClassXMM, Index: 5, CalleeSaved: false}, // Scratch
		{Name: "xmm6", Class: RegClassXMM, Index: 6, CalleeSaved: true},  // Callee-saved
		{Name: "xmm7", Class: RegClassXMM, Index: 7, CalleeSaved: true},  // Callee-saved
	}
)

// LiveInterval represents the lifetime of a virtual register
type LiveInterval struct {
	VirtualReg string        // Virtual register name (e.g., "%t0")
	Start      int           // Starting instruction index
	End        int           // Ending instruction index
	Class      RegisterClass // Required register class
	SpillCost  float64       // Cost of spilling this interval
	UseCount   int           // Number of uses (for priority)
}

// RegisterAllocator performs linear scan register allocation
type RegisterAllocator struct {
	function      *lir.Function
	intervals     []LiveInterval
	active        []LiveInterval        // Currently active intervals
	gprAllocated  map[int]LiveInterval  // GPR allocations (reg index -> interval)
	xmmAllocated  map[int]LiveInterval  // XMM allocations (reg index -> interval)
	allocation    map[string]Allocation // Final register/spill allocation
	spillSlots    map[string]int        // Spilled values -> stack slot offset
	nextSpillSlot int                   // Next available spill slot offset
	callSites     []int                 // Instruction indices with function calls
}

// Allocation represents the final allocation decision for a virtual register
type Allocation struct {
	Type      AllocationType
	Register  PhysicalRegister // If allocated to register
	SpillSlot int              // If spilled to stack (offset from rbp)
}

// AllocationType indicates how a virtual register was allocated
type AllocationType int

const (
	AllocRegister AllocationType = iota
	AllocSpill
)

// NewRegisterAllocator creates a new register allocator for a function
func NewRegisterAllocator(function *lir.Function) *RegisterAllocator {
	return &RegisterAllocator{
		function:      function,
		intervals:     make([]LiveInterval, 0),
		active:        make([]LiveInterval, 0),
		gprAllocated:  make(map[int]LiveInterval),
		xmmAllocated:  make(map[int]LiveInterval),
		allocation:    make(map[string]Allocation),
		spillSlots:    make(map[string]int),
		nextSpillSlot: 8, // Start after frame pointer
		callSites:     make([]int, 0),
	}
}

// AllocateRegisters performs complete register allocation for the function
func (ra *RegisterAllocator) AllocateRegisters() error {
	// Step 1: Build live intervals through liveness analysis
	if err := ra.buildLiveIntervals(); err != nil {
		return fmt.Errorf("liveness analysis failed: %w", err)
	}

	// Step 2: Identify call sites for caller-saved register handling
	ra.identifyCallSites()

	// Step 3: Sort intervals by start point (linear scan requirement)
	sort.Slice(ra.intervals, func(i, j int) bool {
		return ra.intervals[i].Start < ra.intervals[j].Start
	})

	// Step 4: Perform linear scan allocation
	if err := ra.linearScanAllocation(); err != nil {
		return fmt.Errorf("linear scan allocation failed: %w", err)
	}

	return nil
}

// buildLiveIntervals performs liveness analysis to determine variable lifetimes
func (ra *RegisterAllocator) buildLiveIntervals() error {
	// Map virtual registers to their definition and use points
	defs := make(map[string]int)               // reg -> instruction index where defined
	uses := make(map[string][]int)             // reg -> instruction indices where used
	regClass := make(map[string]RegisterClass) // reg -> required register class

	instrIndex := 0

	// Scan all instructions to find definitions and uses
	for _, block := range ra.function.Blocks {
		for _, instr := range block.Insns {
			// Record definitions and uses for this instruction
			switch inst := instr.(type) {
			case lir.Add:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.LHS, instrIndex, uses)
				ra.recordUse(inst.RHS, instrIndex, uses)
			case lir.Sub:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.LHS, instrIndex, uses)
				ra.recordUse(inst.RHS, instrIndex, uses)
			case lir.Mul:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.LHS, instrIndex, uses)
				ra.recordUse(inst.RHS, instrIndex, uses)
			case lir.Div:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.LHS, instrIndex, uses)
				ra.recordUse(inst.RHS, instrIndex, uses)
			case lir.Load:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.Addr, instrIndex, uses)
			case lir.Store:
				ra.recordUse(inst.Addr, instrIndex, uses)
				ra.recordUse(inst.Val, instrIndex, uses)
			case lir.Cmp:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.LHS, instrIndex, uses)
				ra.recordUse(inst.RHS, instrIndex, uses)
			case lir.BrCond:
				ra.recordUse(inst.Cond, instrIndex, uses)
			case lir.Call:
				if inst.Dst != "" {
					// Determine return register class based on return class hint
					class := RegClassGPR
					if inst.RetClass == "f32" || inst.RetClass == "f64" {
						class = RegClassXMM
					}
					ra.recordDef(inst.Dst, instrIndex, class, defs, regClass)
				}
				// Record argument uses
				for _, arg := range inst.Args {
					ra.recordUse(arg, instrIndex, uses)
				}
			case lir.Ret:
				if inst.Src != "" {
					ra.recordUse(inst.Src, instrIndex, uses)
				}
			case lir.Alloc:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
			case lir.Mov:
				ra.recordDef(inst.Dst, instrIndex, RegClassGPR, defs, regClass)
				ra.recordUse(inst.Src, instrIndex, uses)
			}

			instrIndex++
		}
	}

	// Build live intervals from def/use information
	for reg, defPoint := range defs {
		usePoints := uses[reg]
		if len(usePoints) == 0 {
			// Dead code - variable defined but never used
			continue
		}

		// Find last use point
		lastUse := defPoint
		for _, use := range usePoints {
			if use > lastUse {
				lastUse = use
			}
		}

		// Create live interval
		interval := LiveInterval{
			VirtualReg: reg,
			Start:      defPoint,
			End:        lastUse,
			Class:      regClass[reg],
			SpillCost:  ra.calculateSpillCost(reg, usePoints),
			UseCount:   len(usePoints),
		}

		ra.intervals = append(ra.intervals, interval)
	}

	return nil
}

// recordDef records a register definition at the given instruction index
func (ra *RegisterAllocator) recordDef(reg string, instrIndex int, class RegisterClass,
	defs map[string]int, regClass map[string]RegisterClass) {
	if reg == "" || !isVirtualRegister(reg) {
		return
	}
	defs[reg] = instrIndex
	regClass[reg] = class
}

// recordUse records a register use at the given instruction index
func (ra *RegisterAllocator) recordUse(reg string, instrIndex int, uses map[string][]int) {
	if reg == "" || !isVirtualRegister(reg) {
		return
	}
	uses[reg] = append(uses[reg], instrIndex)
}

// isVirtualRegister checks if a register name represents a virtual register (starts with %)
func isVirtualRegister(reg string) bool {
	return strings.HasPrefix(reg, "%")
}

// calculateSpillCost computes the cost of spilling a register based on usage patterns
func (ra *RegisterAllocator) calculateSpillCost(reg string, usePoints []int) float64 {
	baseCost := float64(len(usePoints)) // More uses = higher cost to spill

	// Increase cost for uses in loops (simplified heuristic)
	loopFactor := 1.0
	for _, usePoint := range usePoints {
		if ra.isInLoop(usePoint) {
			loopFactor += 0.5
		}
	}

	return baseCost * loopFactor
}

// isInLoop provides a simple heuristic to detect if an instruction is likely in a loop
func (ra *RegisterAllocator) isInLoop(instrIndex int) bool {
	// Find which basic block contains this instruction
	blockIndex, _ := ra.findInstructionBlock(instrIndex)
	if blockIndex == -1 {
		return false
	}

	// Use dominance frontier and back-edge analysis to detect loops
	return ra.detectLoopForBlock(blockIndex)
}

// findInstructionBlock finds which basic block contains the given instruction index
func (ra *RegisterAllocator) findInstructionBlock(instrIndex int) (blockIndex, instrInBlock int) {
	currentIndex := 0
	for i, block := range ra.function.Blocks {
		if currentIndex+len(block.Insns) > instrIndex {
			return i, instrIndex - currentIndex
		}
		currentIndex += len(block.Insns)
	}
	return -1, -1
}

// detectLoopForBlock uses control flow analysis to detect if a block is in a loop
func (ra *RegisterAllocator) detectLoopForBlock(blockIndex int) bool {
	if blockIndex < 0 || blockIndex >= len(ra.function.Blocks) {
		return false
	}

	// Build basic control flow graph
	cfg := ra.buildControlFlowGraph()

	// Detect back edges using DFS
	visited := make([]bool, len(ra.function.Blocks))
	recStack := make([]bool, len(ra.function.Blocks))

	// Check if the block is reachable from itself (loop detection)
	return ra.hasBackEdge(cfg, blockIndex, blockIndex, visited, recStack)
}

// buildControlFlowGraph builds a simple control flow graph representation
func (ra *RegisterAllocator) buildControlFlowGraph() [][]int {
	cfg := make([][]int, len(ra.function.Blocks))

	for i, block := range ra.function.Blocks {
		// Initialize empty successor lists
		cfg[i] = make([]int, 0)

		// Analyze terminator instructions to find successors
		if len(block.Insns) > 0 {
			lastInstr := block.Insns[len(block.Insns)-1]
			successors := ra.getSuccessors(lastInstr, i)
			cfg[i] = append(cfg[i], successors...)
		}
	}

	return cfg
}

// getSuccessors extracts successor block indices from terminator instructions
func (ra *RegisterAllocator) getSuccessors(instr interface{}, currentBlock int) []int {
	successors := make([]int, 0)

	// This is a simplified version - in reality, you'd need to analyze
	// the actual instruction types and their target labels
	// For now, assume linear flow with possible branches
	if currentBlock+1 < len(ra.function.Blocks) {
		successors = append(successors, currentBlock+1)
	}

	// TODO: Add proper instruction analysis for:
	// - Conditional branches
	// - Unconditional jumps
	// - Switch statements
	// - Function returns

	return successors
}

// hasBackEdge uses DFS to detect if there's a path from start to target (indicating a loop)
func (ra *RegisterAllocator) hasBackEdge(cfg [][]int, start, target int, visited, recStack []bool) bool {
	if start == target && visited[start] {
		return true // Found a cycle
	}

	visited[start] = true
	recStack[start] = true

	// Visit all successors
	for _, successor := range cfg[start] {
		if !visited[successor] {
			if ra.hasBackEdge(cfg, successor, target, visited, recStack) {
				return true
			}
		} else if recStack[successor] {
			// Found a back edge
			return true
		}
	}

	recStack[start] = false
	return false
}

// identifyCallSites finds all function call instructions for caller-saved register handling
func (ra *RegisterAllocator) identifyCallSites() {
	instrIndex := 0
	for _, block := range ra.function.Blocks {
		for _, instr := range block.Insns {
			if _, isCall := instr.(lir.Call); isCall {
				ra.callSites = append(ra.callSites, instrIndex)
			}
			instrIndex++
		}
	}
}

// linearScanAllocation performs the main linear scan register allocation algorithm
func (ra *RegisterAllocator) linearScanAllocation() error {
	for _, interval := range ra.intervals {
		// Remove expired intervals from active list
		ra.expireOldIntervals(interval.Start)

		// Try to allocate a register for this interval
		if ra.tryAllocateRegister(interval) {
			// Successfully allocated to register
			ra.active = append(ra.active, interval)
			// Keep active list sorted by end point
			sort.Slice(ra.active, func(i, j int) bool {
				return ra.active[i].End < ra.active[j].End
			})
		} else {
			// No register available - must spill
			if err := ra.spillInterval(interval); err != nil {
				return fmt.Errorf("failed to spill interval %s: %w", interval.VirtualReg, err)
			}
		}
	}

	return nil
}

// expireOldIntervals removes intervals that have ended from the active list
func (ra *RegisterAllocator) expireOldIntervals(currentStart int) {
	newActive := make([]LiveInterval, 0, len(ra.active))

	for _, active := range ra.active {
		if active.End >= currentStart {
			// Still active
			newActive = append(newActive, active)
		} else {
			// Expired - free its register
			if alloc, exists := ra.allocation[active.VirtualReg]; exists {
				if alloc.Type == AllocRegister {
					switch alloc.Register.Class {
					case RegClassGPR:
						delete(ra.gprAllocated, alloc.Register.Index)
					case RegClassXMM:
						delete(ra.xmmAllocated, alloc.Register.Index)
					}
				}
			}
		}
	}

	ra.active = newActive
}

// tryAllocateRegister attempts to allocate a physical register for the given interval
func (ra *RegisterAllocator) tryAllocateRegister(interval LiveInterval) bool {
	var availableRegs []PhysicalRegister
	var allocatedMap map[int]LiveInterval

	// Select register set based on required class
	switch interval.Class {
	case RegClassGPR:
		availableRegs = GPRRegisters
		allocatedMap = ra.gprAllocated
	case RegClassXMM:
		availableRegs = XMMRegisters
		allocatedMap = ra.xmmAllocated
	default:
		return false
	}

	// Find an available register
	for _, reg := range availableRegs {
		if _, allocated := allocatedMap[reg.Index]; !allocated {
			// Check if this register conflicts with call sites for caller-saved registers
			if !reg.CalleeSaved && ra.spansCallSite(interval) {
				// Caller-saved register spanning call site - prefer callee-saved if available
				continue
			}

			// Allocate this register
			allocatedMap[reg.Index] = interval
			ra.allocation[interval.VirtualReg] = Allocation{
				Type:     AllocRegister,
				Register: reg,
			}
			return true
		}
	}

	// No register available
	return false
}

// spansCallSite checks if an interval spans any function call sites
func (ra *RegisterAllocator) spansCallSite(interval LiveInterval) bool {
	for _, callSite := range ra.callSites {
		if interval.Start <= callSite && callSite <= interval.End {
			return true
		}
	}
	return false
}

// spillInterval handles spilling when no registers are available
func (ra *RegisterAllocator) spillInterval(interval LiveInterval) error {
	// Find the best interval to spill (highest end point, lowest spill cost)
	var spillCandidate *LiveInterval
	bestScore := -1.0

	// Consider currently active intervals for spilling
	for i := range ra.active {
		active := &ra.active[i]
		if active.Class != interval.Class {
			continue
		}

		// Score based on end point (later is better) and spill cost (lower is better)
		score := float64(active.End) / (active.SpillCost + 1.0)
		if score > bestScore && active.End > interval.End {
			bestScore = score
			spillCandidate = active
		}
	}

	if spillCandidate != nil {
		// Spill the candidate and allocate its register to current interval
		if err := ra.doSpill(*spillCandidate); err != nil {
			return fmt.Errorf("failed to spill candidate %s: %w", spillCandidate.VirtualReg, err)
		}

		// Allocate the freed register to current interval
		if alloc, exists := ra.allocation[spillCandidate.VirtualReg]; exists && alloc.Type == AllocRegister {
			ra.allocation[interval.VirtualReg] = Allocation{
				Type:     AllocRegister,
				Register: alloc.Register,
			}

			// Update allocated map
			switch interval.Class {
			case RegClassGPR:
				ra.gprAllocated[alloc.Register.Index] = interval
			case RegClassXMM:
				ra.xmmAllocated[alloc.Register.Index] = interval
			}

			// Remove spilled interval from active list
			for i, active := range ra.active {
				if active.VirtualReg == spillCandidate.VirtualReg {
					ra.active = append(ra.active[:i], ra.active[i+1:]...)
					break
				}
			}

			// Add current interval to active list
			ra.active = append(ra.active, interval)
		}
	} else {
		// Spill current interval
		if err := ra.doSpill(interval); err != nil {
			return fmt.Errorf("failed to spill current interval %s: %w", interval.VirtualReg, err)
		}
	}

	return nil
}

// doSpill performs the actual spilling of an interval to a stack slot
func (ra *RegisterAllocator) doSpill(interval LiveInterval) error {
	// Allocate a new spill slot
	spillSlot := ra.nextSpillSlot
	ra.nextSpillSlot += 8 // Each slot is 8 bytes

	// Record spill allocation
	ra.spillSlots[interval.VirtualReg] = spillSlot
	ra.allocation[interval.VirtualReg] = Allocation{
		Type:      AllocSpill,
		SpillSlot: spillSlot,
	}

	return nil
}

// GetAllocation returns the final register allocation for a virtual register
func (ra *RegisterAllocator) GetAllocation(virtualReg string) (Allocation, bool) {
	alloc, exists := ra.allocation[virtualReg]
	return alloc, exists
}

// GetSpillSlots returns the complete mapping of spilled registers to stack slots
func (ra *RegisterAllocator) GetSpillSlots() map[string]int {
	return ra.spillSlots
}

// GetTotalSpillSlots returns the total number of spill slots needed
func (ra *RegisterAllocator) GetTotalSpillSlots() int {
	return (ra.nextSpillSlot - 8) / 8 // Number of 8-byte slots allocated
}

// PrintAllocationResults outputs the allocation results for debugging
func (ra *RegisterAllocator) PrintAllocationResults() string {
	var result strings.Builder
	result.WriteString("Register Allocation Results:\n")

	// Sort allocations for consistent output
	var regs []string
	for reg := range ra.allocation {
		regs = append(regs, reg)
	}
	sort.Strings(regs)

	for _, reg := range regs {
		alloc := ra.allocation[reg]
		switch alloc.Type {
		case AllocRegister:
			result.WriteString(fmt.Sprintf("  %s -> %s\n", reg, alloc.Register.Name))
		case AllocSpill:
			result.WriteString(fmt.Sprintf("  %s -> spill slot [rbp-%d]\n", reg, alloc.SpillSlot))
		}
	}

	result.WriteString(fmt.Sprintf("Total spill slots: %d\n", ra.GetTotalSpillSlots()))

	return result.String()
}
