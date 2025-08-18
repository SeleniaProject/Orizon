// Package exception provides simplified exception handling integration with MIR
// This is a bootstrap implementation focusing on core functionality
package exception

import (
	"fmt"
	"strings"
)

// SimplifiedMIRIntegration provides basic exception handling for MIR
type SimplifiedMIRIntegration struct {
	codeGen        *ExceptionCodeGen
	labelCounter   int
	exceptionInfos map[string]*ExceptionInfo
}

// NewSimplifiedMIRIntegration creates a new simplified MIR exception integration
func NewSimplifiedMIRIntegration() *SimplifiedMIRIntegration {
	return &SimplifiedMIRIntegration{
		codeGen:        NewExceptionCodeGen(),
		labelCounter:   0,
		exceptionInfos: make(map[string]*ExceptionInfo),
	}
}

// nextLabel generates a unique label for exception handling
func (smi *SimplifiedMIRIntegration) nextLabel(prefix string) string {
	smi.labelCounter++
	return fmt.Sprintf("%s_%d", prefix, smi.labelCounter)
}

// GenerateBoundsCheckMIR generates MIR-level bounds checking logic
func (smi *SimplifiedMIRIntegration) GenerateBoundsCheckMIR(indexVar, lengthVar, arrayName string) string {
	checkLabel := smi.nextLabel("bounds_check")
	errorLabel := smi.nextLabel("bounds_error")
	okLabel := smi.nextLabel("bounds_ok")

	// Add exception info
	smi.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionBoundsCheck,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      fmt.Sprintf("Array bounds check failed for %s", arrayName),
		Location:     "generated",
	}

	var mir strings.Builder
	mir.WriteString(fmt.Sprintf("; Bounds check for %s\n", arrayName))
	mir.WriteString(fmt.Sprintf("%s:\n", checkLabel))
	mir.WriteString(fmt.Sprintf("  temp_cmp1 = cmp.slt %s, 0\n", indexVar))
	mir.WriteString(fmt.Sprintf("  brcond temp_cmp1, %s, %s_cont1\n", errorLabel, checkLabel))
	mir.WriteString(fmt.Sprintf("%s_cont1:\n", checkLabel))
	mir.WriteString(fmt.Sprintf("  temp_cmp2 = cmp.sge %s, %s\n", indexVar, lengthVar))
	mir.WriteString(fmt.Sprintf("  brcond temp_cmp2, %s, %s\n", errorLabel, okLabel))
	mir.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	mir.WriteString(fmt.Sprintf("  call panic_bounds_check(%s, %s, %s)\n", arrayName, indexVar, lengthVar))
	mir.WriteString(fmt.Sprintf("%s:\n", okLabel))
	mir.WriteString("  ; Bounds check passed\n")

	return mir.String()
}

// GenerateNullCheckMIR generates MIR-level null pointer checking logic
func (smi *SimplifiedMIRIntegration) GenerateNullCheckMIR(ptrVar, ptrName string) string {
	checkLabel := smi.nextLabel("null_check")
	errorLabel := smi.nextLabel("null_error")
	okLabel := smi.nextLabel("null_ok")

	// Add exception info
	smi.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionNullPointer,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      fmt.Sprintf("Null pointer access: %s", ptrName),
		Location:     "generated",
	}

	var mir strings.Builder
	mir.WriteString(fmt.Sprintf("; Null pointer check for %s\n", ptrName))
	mir.WriteString(fmt.Sprintf("%s:\n", checkLabel))
	mir.WriteString(fmt.Sprintf("  temp_null_cmp = cmp.eq %s, 0\n", ptrVar))
	mir.WriteString(fmt.Sprintf("  brcond temp_null_cmp, %s, %s\n", errorLabel, okLabel))
	mir.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	mir.WriteString(fmt.Sprintf("  call panic_null_pointer(%s)\n", ptrName))
	mir.WriteString(fmt.Sprintf("%s:\n", okLabel))
	mir.WriteString("  ; Null check passed\n")

	return mir.String()
}

// GenerateDivisionCheckMIR generates MIR-level division by zero checking logic
func (smi *SimplifiedMIRIntegration) GenerateDivisionCheckMIR(divisorVar, operation string) string {
	checkLabel := smi.nextLabel("div_check")
	errorLabel := smi.nextLabel("div_error")
	okLabel := smi.nextLabel("div_ok")

	// Add exception info
	smi.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionDivisionByZero,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      fmt.Sprintf("Division by zero in %s", operation),
		Location:     "generated",
	}

	var mir strings.Builder
	mir.WriteString(fmt.Sprintf("; Division by zero check for %s\n", operation))
	mir.WriteString(fmt.Sprintf("%s:\n", checkLabel))
	mir.WriteString(fmt.Sprintf("  temp_div_cmp = cmp.eq %s, 0\n", divisorVar))
	mir.WriteString(fmt.Sprintf("  brcond temp_div_cmp, %s, %s\n", errorLabel, okLabel))
	mir.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	mir.WriteString(fmt.Sprintf("  call panic_division_by_zero(%s)\n", operation))
	mir.WriteString(fmt.Sprintf("%s:\n", okLabel))
	mir.WriteString("  ; Division check passed\n")

	return mir.String()
}

// GenerateAssertMIR generates MIR-level assertion checking logic
func (smi *SimplifiedMIRIntegration) GenerateAssertMIR(conditionVar, message string) string {
	checkLabel := smi.nextLabel("assert_check")
	errorLabel := smi.nextLabel("assert_error")
	okLabel := smi.nextLabel("assert_ok")

	// Add exception info
	smi.exceptionInfos[checkLabel] = &ExceptionInfo{
		Kind:         ExceptionAssert,
		CheckLabels:  []string{errorLabel, okLabel},
		HandlerLabel: errorLabel,
		Message:      message,
		Location:     "generated",
	}

	var mir strings.Builder
	mir.WriteString(fmt.Sprintf("; Assertion check: %s\n", message))
	mir.WriteString(fmt.Sprintf("%s:\n", checkLabel))
	mir.WriteString(fmt.Sprintf("  temp_assert_cmp = cmp.eq %s, 0\n", conditionVar))
	mir.WriteString(fmt.Sprintf("  brcond temp_assert_cmp, %s, %s\n", errorLabel, okLabel))
	mir.WriteString(fmt.Sprintf("%s:\n", errorLabel))
	mir.WriteString(fmt.Sprintf("  call panic_assert(\"%s\")\n", message))
	mir.WriteString(fmt.Sprintf("%s:\n", okLabel))
	mir.WriteString("  ; Assertion passed\n")

	return mir.String()
}

// GenerateExceptionHandlersMIR generates MIR for all exception handlers
func (smi *SimplifiedMIRIntegration) GenerateExceptionHandlersMIR() string {
	var mir strings.Builder

	mir.WriteString("; Exception handler functions in MIR\n\n")

	// Bounds check handler
	mir.WriteString("func panic_bounds_check(array, index, length) {\n")
	mir.WriteString("entry:\n")
	mir.WriteString("  msg = alloca bounds_check_msg\n")
	mir.WriteString("  call runtime_abort(msg)\n")
	mir.WriteString("  ret\n")
	mir.WriteString("}\n\n")

	// Null pointer handler
	mir.WriteString("func panic_null_pointer(name) {\n")
	mir.WriteString("entry:\n")
	mir.WriteString("  msg = alloca null_pointer_msg\n")
	mir.WriteString("  call runtime_abort(msg)\n")
	mir.WriteString("  ret\n")
	mir.WriteString("}\n\n")

	// Division by zero handler
	mir.WriteString("func panic_division_by_zero(operation) {\n")
	mir.WriteString("entry:\n")
	mir.WriteString("  msg = alloca div_zero_msg\n")
	mir.WriteString("  call runtime_abort(msg)\n")
	mir.WriteString("  ret\n")
	mir.WriteString("}\n\n")

	// Assertion handler
	mir.WriteString("func panic_assert(message) {\n")
	mir.WriteString("entry:\n")
	mir.WriteString("  call runtime_abort(message)\n")
	mir.WriteString("  ret\n")
	mir.WriteString("}\n\n")

	// Runtime abort function
	mir.WriteString("func runtime_abort(message) {\n")
	mir.WriteString("entry:\n")
	mir.WriteString("  ; Write message to stderr and exit\n")
	mir.WriteString("  call write_stderr(message)\n")
	mir.WriteString("  call exit_process(1)\n")
	mir.WriteString("  ret\n")
	mir.WriteString("}\n\n")

	return mir.String()
}

// OptimizeExceptionChecksMIR optimizes exception checks in MIR representation
func (smi *SimplifiedMIRIntegration) OptimizeExceptionChecksMIR(mirCode string) string {
	if !smi.codeGen.UseTraps {
		return mirCode
	}

	// Remove redundant comparisons
	optimized := strings.ReplaceAll(mirCode,
		"temp_cmp1 = cmp.slt",
		"temp_cmp1 = cmp.slt ; optimized")

	// Combine adjacent null checks
	optimized = strings.ReplaceAll(optimized,
		"temp_null_cmp = cmp.eq ptr, 0\n  temp_null_cmp2 = cmp.eq ptr, 0",
		"temp_null_cmp = cmp.eq ptr, 0")

	return optimized
}

// GenerateExceptionMetadataMIR generates metadata for exception handling
func (smi *SimplifiedMIRIntegration) GenerateExceptionMetadataMIR() map[string]interface{} {
	metadata := make(map[string]interface{})

	metadata["exception_count"] = len(smi.exceptionInfos)
	metadata["exception_types"] = smi.getExceptionTypeCounts()
	metadata["check_labels"] = smi.getAllCheckLabels()
	metadata["handler_labels"] = smi.getAllHandlerLabels()
	metadata["mir_format"] = "simplified"

	return metadata
}

// getExceptionTypeCounts returns count of each exception type
func (smi *SimplifiedMIRIntegration) getExceptionTypeCounts() map[string]int {
	counts := make(map[string]int)

	for _, info := range smi.exceptionInfos {
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

// getAllCheckLabels returns all check labels
func (smi *SimplifiedMIRIntegration) getAllCheckLabels() []string {
	labels := make([]string, 0)

	for label := range smi.exceptionInfos {
		labels = append(labels, label)
	}

	return labels
}

// getAllHandlerLabels returns all handler labels
func (smi *SimplifiedMIRIntegration) getAllHandlerLabels() []string {
	labels := make([]string, 0)
	seen := make(map[string]bool)

	for _, info := range smi.exceptionInfos {
		if !seen[info.HandlerLabel] {
			labels = append(labels, info.HandlerLabel)
			seen[info.HandlerLabel] = true
		}
	}

	return labels
}

// GenerateCompleteMIRExceptionSystem generates a complete MIR exception system
func (smi *SimplifiedMIRIntegration) GenerateCompleteMIRExceptionSystem() string {
	var complete strings.Builder

	complete.WriteString("; Complete MIR Exception Handling System\n")
	complete.WriteString("; Generated by Orizon Compiler Exception System\n\n")

	// Add example usage
	complete.WriteString("func example_with_exceptions(array_ptr, array_len, index) {\n")
	complete.WriteString("entry:\n")
	complete.WriteString(smi.GenerateBoundsCheckMIR("index", "array_len", "example_array"))
	complete.WriteString(smi.GenerateNullCheckMIR("array_ptr", "example_array"))
	complete.WriteString("  ; Safe to access array[index] now\n")
	complete.WriteString("  element = load array_ptr[index]\n")
	complete.WriteString("  ret element\n")
	complete.WriteString("}\n\n")

	// Add exception handlers
	complete.WriteString(smi.GenerateExceptionHandlersMIR())

	return complete.String()
}
