// Package exception provides integration between the exception handling system.
// and the MIR (Mid-level Intermediate Representation) for code generation.
package exception

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/mir"
)

// ExceptionKind represents different types of exceptions.
type ExceptionKind int

const (
	ExceptionBoundsCheck ExceptionKind = iota
	ExceptionNullPointer
	ExceptionDivisionByZero
	ExceptionAssert
	ExceptionUserDefined
)

// ExceptionCodeGen handles code generation for exceptions.
type ExceptionCodeGen struct {
	labelCounter int
}

// NewExceptionCodeGen creates a new exception code generator.
func NewExceptionCodeGen() *ExceptionCodeGen {
	return &ExceptionCodeGen{
		labelCounter: 0,
	}
}

// MIRExceptionIntegration provides exception handling integration with MIR.
type MIRExceptionIntegration struct {
	codeGen        *ExceptionCodeGen
	exceptionInfos map[string]*ExceptionInfo
	labelCounter   int
}

// ExceptionInfo stores information about exceptions at the MIR level.
type ExceptionInfo struct {
	HandlerLabel string
	Message      string
	Location     string
	CheckLabels  []string
	Kind         ExceptionKind
}

// NewMIRExceptionIntegration creates a new MIR exception integration.
func NewMIRExceptionIntegration() *MIRExceptionIntegration {
	return &MIRExceptionIntegration{
		codeGen:        NewExceptionCodeGen(),
		labelCounter:   0,
		exceptionInfos: make(map[string]*ExceptionInfo),
	}
}

// nextLabel generates a unique label for exception handling.
func (mei *MIRExceptionIntegration) nextLabel(prefix string) string {
	mei.labelCounter++

	return fmt.Sprintf("%s_%d", prefix, mei.labelCounter)
}

// AddBoundsCheckToMIR adds bounds checking to a MIR basic block.
func (mei *MIRExceptionIntegration) AddBoundsCheckToMIR(block *mir.BasicBlock, indexVar, lengthVar, arrayName string) {
	checkLabel := mei.nextLabel("bounds_check")
	errorLabel := mei.nextLabel("bounds_error")
	okLabel := mei.nextLabel("bounds_ok")

	// Add exception info.
	mei.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionBoundsCheck,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      fmt.Sprintf("Array bounds check failed for %s", arrayName),
		Location:     "generated", // Would be filled from source info
	}

	// Create bounds check instructions using MIR structures.
	// Compare index < 0.
	cmpInst1 := &mir.Cmp{
		Dst:  "temp_cmp1",
		Pred: mir.CmpSLT,
		LHS:  mir.Value{Kind: mir.ValRef, Ref: indexVar},
		RHS:  mir.Value{Kind: mir.ValConstInt, Int64: 0},
	}

	// Conditional branch if less than 0.
	branchInst1 := &mir.CondBr{
		Cond:  mir.Value{Kind: mir.ValRef, Ref: "temp_cmp1"},
		True:  errorLabel,
		False: okLabel + "_continue1",
	}

	// Compare index >= length.
	cmpInst2 := &mir.Cmp{
		Dst:  "temp_cmp2",
		Pred: mir.CmpSGE,
		LHS:  mir.Value{Kind: mir.ValRef, Ref: indexVar},
		RHS:  mir.Value{Kind: mir.ValRef, Ref: lengthVar},
	}

	// Conditional branch if greater or equal.
	branchInst2 := &mir.CondBr{
		Cond:  mir.Value{Kind: mir.ValRef, Ref: "temp_cmp2"},
		True:  errorLabel,
		False: okLabel,
	}

	// Jump to ok label.
	jumpOkInst := &mir.Br{Target: okLabel}

	// Exception handling call.
	panicInst := &mir.Call{
		Dst:    "",
		Callee: "panic_bounds_check",
		Args: []mir.Value{
			{Kind: mir.ValRef, Ref: arrayName},
			{Kind: mir.ValRef, Ref: indexVar},
			{Kind: mir.ValRef, Ref: lengthVar},
		},
	}

	// Add instructions to the block.
	block.Instr = append(block.Instr,
		cmpInst1, branchInst1,
		cmpInst2, branchInst2,
		jumpOkInst, panicInst)
}

// AddNullCheckToMIR adds null pointer checking to a MIR basic block.
func (mei *MIRExceptionIntegration) AddNullCheckToMIR(block *mir.BasicBlock, ptrVar, ptrName string) {
	checkLabel := mei.nextLabel("null_check")
	errorLabel := mei.nextLabel("null_error")
	okLabel := mei.nextLabel("null_ok")

	// Add exception info.
	mei.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionNullPointer,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      fmt.Sprintf("Null pointer access: %s", ptrName),
		Location:     "generated",
	}

	// Create null check instructions.
	cmpInst := &mir.Cmp{
		Dst:  "temp_null_cmp",
		Pred: mir.CmpEQ,
		LHS:  mir.Value{Kind: mir.ValRef, Ref: ptrVar},
		RHS:  mir.Value{Kind: mir.ValConstInt, Int64: 0},
	}

	branchInst := &mir.CondBr{
		Cond:  mir.Value{Kind: mir.ValRef, Ref: "temp_null_cmp"},
		True:  errorLabel,
		False: okLabel,
	}

	jumpOkInst := &mir.Br{
		Target: okLabel,
	}

	panicInst := &mir.Call{
		Dst:    "",
		Callee: "panic_null_pointer",
		Args:   []mir.Value{{Kind: mir.ValRef, Ref: ptrName}},
	}

	// Add instructions to the block.
	block.Instr = append(block.Instr,
		cmpInst, branchInst, jumpOkInst, panicInst)
}

// AddDivisionCheckToMIR adds division by zero checking to a MIR basic block.
func (mei *MIRExceptionIntegration) AddDivisionCheckToMIR(block *mir.BasicBlock, divisorVar, operation string) {
	checkLabel := mei.nextLabel("div_check")
	errorLabel := mei.nextLabel("div_error")
	okLabel := mei.nextLabel("div_ok")

	// Add exception info.
	mei.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionDivisionByZero,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      fmt.Sprintf("Division by zero in %s", operation),
		Location:     "generated",
	}

	// Create division check instructions.
	cmpInst := &mir.Cmp{
		Dst:  "temp_div_cmp",
		Pred: mir.CmpEQ,
		LHS:  mir.Value{Kind: mir.ValRef, Ref: divisorVar},
		RHS:  mir.Value{Kind: mir.ValConstInt, Int64: 0},
	}

	branchInst := &mir.CondBr{
		Cond:  mir.Value{Kind: mir.ValRef, Ref: "temp_div_cmp"},
		True:  errorLabel,
		False: okLabel,
	}

	jumpOkInst := &mir.Br{
		Target: okLabel,
	}

	panicInst := &mir.Call{
		Dst:    "",
		Callee: "panic_division_by_zero",
		Args:   []mir.Value{{Kind: mir.ValRef, Ref: operation}},
	}

	// Add instructions to the block.
	block.Instr = append(block.Instr,
		cmpInst, branchInst, jumpOkInst, panicInst)
}

// AddArrayBoundsCheckToMIR adds array bounds checking to a MIR basic block.
func (mei *MIRExceptionIntegration) AddArrayBoundsCheckToMIR(block *mir.BasicBlock, arrayVar, indexVar, lengthVar string) {
	// Implementation for array bounds checking.
}

// AddAssertToMIR adds assertion checking to a MIR basic block.
func (mei *MIRExceptionIntegration) AddAssertToMIR(block *mir.BasicBlock, conditionVar, message string) {
	checkLabel := mei.nextLabel("assert_check")
	errorLabel := mei.nextLabel("assert_error")
	okLabel := mei.nextLabel("assert_ok")

	// Add exception info.
	mei.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionAssert,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      message,
		Location:     "generated",
	}

	// Create assertion check instructions.
	// Branch if condition is false (0).
	cmpInst := &mir.Cmp{
		Dst:  "temp_assert_cmp",
		Pred: mir.CmpEQ,
		LHS:  mir.Value{Kind: mir.ValRef, Ref: conditionVar},
		RHS:  mir.Value{Kind: mir.ValConstInt, Int64: 0},
	}

	branchInst := &mir.CondBr{
		Cond:  mir.Value{Kind: mir.ValRef, Ref: "temp_assert_cmp"},
		True:  errorLabel,
		False: okLabel,
	}

	jumpOkInst := &mir.Br{
		Target: okLabel,
	}

	panicInst := &mir.Call{
		Dst:    "",
		Callee: "panic_assert",
		Args:   []mir.Value{{Kind: mir.ValConstString, StrVal: message}},
	}

	// Add instructions to the block.
	block.Instr = append(block.Instr,
		cmpInst, branchInst, jumpOkInst, panicInst)
}

// GenerateExceptionHandlers generates MIR for all exception handlers.
func (mei *MIRExceptionIntegration) GenerateExceptionHandlers() *mir.Function {
	function := &mir.Function{
		Name:       "exception_handlers",
		Parameters: []mir.Value{},
		Blocks:     []*mir.BasicBlock{},
	}

	// Create main handler block.
	mainBlock := &mir.BasicBlock{
		Name:  "exception_handlers",
		Instr: []mir.Instr{},
	}

	// Bounds check handler.
	boundsBlock := &mir.BasicBlock{
		Name: "panic_bounds_check",
		Instr: []mir.Instr{
			&mir.Call{
				Dst:    "",
				Callee: "runtime_panic",
				Args: []mir.Value{
					{Kind: mir.ValConstString, StrVal: "bounds_check_msg"},
				},
			},
			&mir.Ret{Val: nil},
		},
	}

	// Null pointer handler.
	nullBlock := &mir.BasicBlock{
		Name: "panic_null_pointer",
		Instr: []mir.Instr{
			&mir.Call{
				Dst:    "",
				Callee: "runtime_panic",
				Args: []mir.Value{
					{Kind: mir.ValConstString, StrVal: "null_pointer_msg"},
				},
			},
			&mir.Ret{Val: nil},
		},
	}

	// Division by zero handler.
	divBlock := &mir.BasicBlock{
		Name: "panic_division_by_zero",
		Instr: []mir.Instr{
			&mir.Call{
				Dst:    "",
				Callee: "runtime_panic",
				Args: []mir.Value{
					{Kind: mir.ValConstString, StrVal: "div_zero_msg"},
				},
			},
			&mir.Ret{Val: nil},
		},
	}

	// Assertion handler.
	assertBlock := &mir.BasicBlock{
		Name: "panic_assert",
		Instr: []mir.Instr{
			&mir.Call{
				Dst:    "",
				Callee: "runtime_panic",
				Args: []mir.Value{
					{Kind: mir.ValConstString, StrVal: "assert_msg"},
				},
			},
			&mir.Ret{Val: nil},
		},
	}

	function.Blocks = append(function.Blocks, mainBlock, boundsBlock, nullBlock, divBlock, assertBlock)

	return function
}

// OptimizeExceptionChecks optimizes exception checks in MIR.
func (mei *MIRExceptionIntegration) OptimizeExceptionChecks(function *mir.Function) {
	for _, block := range function.Blocks {
		mei.optimizeBlockExceptionChecks(block)
	}
}

// optimizeBlockExceptionChecks optimizes exception checks within a single block.
func (mei *MIRExceptionIntegration) optimizeBlockExceptionChecks(block *mir.BasicBlock) {
	// Remove redundant bounds checks.
	mei.removeRedundantBoundsChecks(block)

	// Combine adjacent null checks.
	mei.combineAdjacentNullChecks(block)

	// Optimize exception check ordering.
	mei.optimizeCheckOrdering(block)
}

// removeRedundantBoundsChecks removes redundant bounds checks in a block.
func (mei *MIRExceptionIntegration) removeRedundantBoundsChecks(block *mir.BasicBlock) {
	seen := make(map[string]bool)
	optimized := make([]mir.Instr, 0, len(block.Instr))

	for _, inst := range block.Instr {
		// Check if this is a bounds check (Cmp instruction).
		if cmpInst, ok := inst.(*mir.Cmp); ok {
			// Use destination variable as key for redundancy check.
			key := fmt.Sprintf("bounds_%s_%s", cmpInst.LHS.String(), cmpInst.RHS.String())
			if seen[key] {
				// Skip redundant check.
				continue
			}

			seen[key] = true
		}

		optimized = append(optimized, inst)
	}

	block.Instr = optimized
}

// combineAdjacentNullChecks combines adjacent null checks for the same variable.
func (mei *MIRExceptionIntegration) combineAdjacentNullChecks(block *mir.BasicBlock) {
	optimized := make([]mir.Instr, 0, len(block.Instr))
	i := 0

	for i < len(block.Instr) {
		inst := block.Instr[i]

		// Check if this is a null check (Cmp instruction).
		if cmpInst, ok := inst.(*mir.Cmp); ok && cmpInst.Dst != "" {
			// Look ahead for duplicate null checks.
			j := i + 1
			for j < len(block.Instr) {
				if nextCmp, ok := block.Instr[j].(*mir.Cmp); ok &&
					nextCmp.LHS.String() == cmpInst.LHS.String() {
					// Skip duplicate.
					j++

					continue
				}

				break
			}

			optimized = append(optimized, inst)
			i = j
		} else {
			optimized = append(optimized, inst)
			i++
		}
	}

	block.Instr = optimized
}

// optimizeCheckOrdering reorders exception checks for better performance.
func (mei *MIRExceptionIntegration) optimizeCheckOrdering(block *mir.BasicBlock) {
	// This is a simple heuristic: put cheaper checks first.
	// Null checks are generally cheaper than bounds checks.
	checkInsts := make([]mir.Instr, 0)
	otherInsts := make([]mir.Instr, 0)

	for _, inst := range block.Instr {
		if mei.isExceptionCheck(inst) {
			checkInsts = append(checkInsts, inst)
		} else {
			otherInsts = append(otherInsts, inst)
		}
	}

	// Sort checks by cost (null checks first, then bounds checks).
	for i := 0; i < len(checkInsts); i++ {
		for j := i + 1; j < len(checkInsts); j++ {
			if mei.getCheckCost(checkInsts[j]) < mei.getCheckCost(checkInsts[i]) {
				checkInsts[i], checkInsts[j] = checkInsts[j], checkInsts[i]
			}
		}
	}

	// Rebuild instructions with optimized order.
	optimized := make([]mir.Instr, 0, len(block.Instr))
	optimized = append(optimized, checkInsts...)
	optimized = append(optimized, otherInsts...)

	block.Instr = optimized
}

// isExceptionCheck returns true if the instruction is an exception check.
func (mei *MIRExceptionIntegration) isExceptionCheck(inst mir.Instr) bool {
	// Check if this is a Cmp instruction (used for exception checks).
	if _, ok := inst.(*mir.Cmp); ok {
		return true
	}

	return false
}

// getCheckCost returns the relative cost of an exception check.
func (mei *MIRExceptionIntegration) getCheckCost(inst mir.Instr) int {
	// For now, assume all checks have same cost.
	// In a real implementation, we'd analyze the specific check type.
	return 1
}

// GenerateExceptionMetadata generates metadata for exception handling.
func (mei *MIRExceptionIntegration) GenerateExceptionMetadata() map[string]interface{} {
	metadata := make(map[string]interface{})

	metadata["exception_count"] = len(mei.exceptionInfos)
	metadata["exception_types"] = mei.getExceptionTypeCounts()
	metadata["check_labels"] = mei.getAllCheckLabels()
	metadata["handler_labels"] = mei.getAllHandlerLabels()

	return metadata
}

// getExceptionTypeCounts returns count of each exception type.
func (mei *MIRExceptionIntegration) getExceptionTypeCounts() map[string]int {
	counts := make(map[string]int)

	for _, info := range mei.exceptionInfos {
		switch info.Kind {
		case ExceptionBoundsCheck:
			counts["bounds_check"]++
		case ExceptionNullPointer:
			counts["null_pointer"]++
		case ExceptionDivisionByZero:
			counts["division_by_zero"]++
		case ExceptionAssert:
			counts["assertion"]++
		default:
			counts["other"]++
		}
	}

	return counts
}

// getAllCheckLabels returns all check labels.
func (mei *MIRExceptionIntegration) getAllCheckLabels() []string {
	labels := make([]string, 0)

	for label := range mei.exceptionInfos {
		labels = append(labels, label)
	}

	return labels
}

// getAllHandlerLabels returns all handler labels.
func (mei *MIRExceptionIntegration) getAllHandlerLabels() []string {
	labels := make([]string, 0)
	seen := make(map[string]bool)

	for _, info := range mei.exceptionInfos {
		if !seen[info.HandlerLabel] {
			labels = append(labels, info.HandlerLabel)
			seen[info.HandlerLabel] = true
		}
	}

	return labels
}
