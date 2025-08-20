// Package diagnostics - Diagnostic builder for easy creation of comprehensive diagnostics.
package diagnostics

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/position"
)

// DiagnosticBuilder provides a fluent interface for building diagnostics.
type DiagnosticBuilder struct {
	diagnostic Diagnostic
}

// NewDiagnosticBuilder creates a new diagnostic builder.
func NewDiagnosticBuilder() *DiagnosticBuilder {
	return &DiagnosticBuilder{
		diagnostic: Diagnostic{
			FixSuggestions: make([]FixSuggestion, 0),
			RelatedInfo:    make([]RelatedInformation, 0),
			Context:        make([]string, 0),
			Examples:       make([]string, 0),
			SeeAlso:        make([]string, 0),
			StackTrace:     make([]string, 0),
		},
	}
}

// Error creates an error-level diagnostic.
func (db *DiagnosticBuilder) Error() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticError

	return db
}

// Warning creates a warning-level diagnostic.
func (db *DiagnosticBuilder) Warning() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticWarning

	return db
}

// Info creates an info-level diagnostic.
func (db *DiagnosticBuilder) Info() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticInfo

	return db
}

// Hint creates a hint-level diagnostic.
func (db *DiagnosticBuilder) Hint() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticHint

	return db
}

// WithID sets the diagnostic ID.
func (db *DiagnosticBuilder) WithID(id string) *DiagnosticBuilder {
	db.diagnostic.ID = id

	return db
}

// WithCode sets the error code.
func (db *DiagnosticBuilder) WithCode(code string) *DiagnosticBuilder {
	db.diagnostic.Code = code

	return db
}

// WithCategory sets the diagnostic category.
func (db *DiagnosticBuilder) WithCategory(category DiagnosticCategory) *DiagnosticBuilder {
	db.diagnostic.Category = category

	return db
}

// WithMessage sets the main diagnostic message.
func (db *DiagnosticBuilder) WithMessage(message string) *DiagnosticBuilder {
	db.diagnostic.Message = message

	return db
}

// WithMessagef sets the main diagnostic message with formatting.
func (db *DiagnosticBuilder) WithMessagef(format string, args ...interface{}) *DiagnosticBuilder {
	db.diagnostic.Message = fmt.Sprintf(format, args...)

	return db
}

// WithSpan sets the source location span.
func (db *DiagnosticBuilder) WithSpan(span position.Span) *DiagnosticBuilder {
	db.diagnostic.Span = span

	return db
}

// WithSourceFile sets the source file.
func (db *DiagnosticBuilder) WithSourceFile(filename string) *DiagnosticBuilder {
	db.diagnostic.SourceFile = filename

	return db
}

// WithExplanation sets a detailed explanation.
func (db *DiagnosticBuilder) WithExplanation(explanation string) *DiagnosticBuilder {
	db.diagnostic.Explanation = explanation

	return db
}

// WithExplanationf sets a detailed explanation with formatting.
func (db *DiagnosticBuilder) WithExplanationf(format string, args ...interface{}) *DiagnosticBuilder {
	db.diagnostic.Explanation = fmt.Sprintf(format, args...)

	return db
}

// AddExample adds a code example.
func (db *DiagnosticBuilder) AddExample(example string) *DiagnosticBuilder {
	db.diagnostic.Examples = append(db.diagnostic.Examples, example)

	return db
}

// AddExamplef adds a formatted code example.
func (db *DiagnosticBuilder) AddExamplef(format string, args ...interface{}) *DiagnosticBuilder {
	example := fmt.Sprintf(format, args...)
	db.diagnostic.Examples = append(db.diagnostic.Examples, example)

	return db
}

// AddFixSuggestion adds a fix suggestion.
func (db *DiagnosticBuilder) AddFixSuggestion(description, replacement string, span position.Span, automatic bool) *DiagnosticBuilder {
	fix := FixSuggestion{
		Description: description,
		Replacement: replacement,
		Span:        span,
		Automatic:   automatic,
	}
	db.diagnostic.FixSuggestions = append(db.diagnostic.FixSuggestions, fix)

	return db
}

// AddAutomaticFix adds an automatic fix suggestion.
func (db *DiagnosticBuilder) AddAutomaticFix(description, replacement string, span position.Span) *DiagnosticBuilder {
	return db.AddFixSuggestion(description, replacement, span, true)
}

// AddManualFix adds a manual fix suggestion.
func (db *DiagnosticBuilder) AddManualFix(description string) *DiagnosticBuilder {
	return db.AddFixSuggestion(description, "", position.Span{}, false)
}

// AddRelatedInfo adds related information from another location.
func (db *DiagnosticBuilder) AddRelatedInfo(message string, location position.Span) *DiagnosticBuilder {
	info := RelatedInformation{
		Message:  message,
		Location: location,
	}
	db.diagnostic.RelatedInfo = append(db.diagnostic.RelatedInfo, info)

	return db
}

// AddRelatedInfof adds formatted related information.
func (db *DiagnosticBuilder) AddRelatedInfof(location position.Span, format string, args ...interface{}) *DiagnosticBuilder {
	message := fmt.Sprintf(format, args...)

	return db.AddRelatedInfo(message, location)
}

// WithHelpURL sets the help documentation URL.
func (db *DiagnosticBuilder) WithHelpURL(url string) *DiagnosticBuilder {
	db.diagnostic.HelpURL = url

	return db
}

// AddSeeAlso adds a related concept or error code.
func (db *DiagnosticBuilder) AddSeeAlso(concept string) *DiagnosticBuilder {
	db.diagnostic.SeeAlso = append(db.diagnostic.SeeAlso, concept)

	return db
}

// WithStackTrace adds internal compiler stack trace.
func (db *DiagnosticBuilder) WithStackTrace(trace []string) *DiagnosticBuilder {
	db.diagnostic.StackTrace = trace

	return db
}

// Build returns the constructed diagnostic.
func (db *DiagnosticBuilder) Build() Diagnostic {
	return db.diagnostic
}

// Predefined diagnostic builders for common scenarios.

// UndefinedVariableError creates a diagnostic for undefined variables.
func UndefinedVariableError(name string, span position.Span, suggestions []string) Diagnostic {
	builder := NewDiagnosticBuilder().
		Error().
		WithCode("E001").
		WithCategory(CategoryUndefinedVariable).
		WithMessagef("undefined variable '%s'", name).
		WithSpan(span).
		WithExplanationf("The variable '%s' is used but has not been declared in the current scope.", name)

	// Add suggestions for similar variable names.
	if len(suggestions) > 0 {
		builder.AddManualFix(fmt.Sprintf("Did you mean one of: %s?", strings.Join(suggestions, ", ")))
	}

	builder.AddExample("let x = 42; // Declare variable before use").
		AddSeeAlso("variable-declaration").
		AddSeeAlso("scoping-rules")

	return builder.Build()
}

// TypeMismatchError creates a diagnostic for type mismatches.
func TypeMismatchError(expected, actual string, span position.Span) Diagnostic {
	return NewDiagnosticBuilder().
		Error().
		WithCode("E002").
		WithCategory(CategoryTypeError).
		WithMessagef("type mismatch: expected '%s', found '%s'", expected, actual).
		WithSpan(span).
		WithExplanationf("The expression has type '%s' but a value of type '%s' is expected in this context.", actual, expected).
		AddExample(fmt.Sprintf("let x: %s = value as %s; // Explicit type conversion", expected, expected)).
		AddManualFix("Add explicit type conversion").
		AddManualFix("Change the expected type").
		AddSeeAlso("type-system").
		AddSeeAlso("type-conversion").
		Build()
}

// UnusedVariableWarning creates a diagnostic for unused variables.
func UnusedVariableWarning(name string, span position.Span) Diagnostic {
	builder := NewDiagnosticBuilder().
		Warning().
		WithCode("W001").
		WithCategory(CategoryUnusedVariable).
		WithMessagef("unused variable '%s'", name).
		WithSpan(span).
		WithExplanationf("The variable '%s' is declared but never used.", name)

	// Add automatic fixes.
	builder.AddAutomaticFix("Remove unused variable", "", span).
		AddAutomaticFix(fmt.Sprintf("Prefix with underscore: '_%s'", name), "_"+name, span)

	builder.AddExample("let _unused = value; // Prefix with underscore to indicate intentionally unused").
		AddSeeAlso("unused-variables").
		AddSeeAlso("code-style")

	return builder.Build()
}

// MissingReturnError creates a diagnostic for missing return statements.
func MissingReturnError(functionName string, span position.Span, returnType string) Diagnostic {
	return NewDiagnosticBuilder().
		Error().
		WithCode("E003").
		WithCategory(CategoryMissingReturn).
		WithMessagef("function '%s' must return a value of type '%s'", functionName, returnType).
		WithSpan(span).
		WithExplanationf("The function '%s' is declared to return '%s' but some code paths do not return a value.", functionName, returnType).
		AddExample(fmt.Sprintf("return default_value; // Return a value of type '%s'", returnType)).
		AddManualFix("Add return statement to all code paths").
		AddSeeAlso("function-returns").
		AddSeeAlso("control-flow").
		Build()
}

// DeadCodeWarning creates a diagnostic for unreachable code.
func DeadCodeWarning(span position.Span) Diagnostic {
	return NewDiagnosticBuilder().
		Warning().
		WithCode("W002").
		WithCategory(CategoryUnreachableCode).
		WithMessage("unreachable code").
		WithSpan(span).
		WithExplanation("This code will never be executed because it follows an unconditional return, break, or continue statement.").
		AddAutomaticFix("Remove unreachable code", "", span).
		AddManualFix("Restructure control flow to make code reachable").
		AddSeeAlso("control-flow").
		AddSeeAlso("dead-code-elimination").
		Build()
}

// MemoryLeakWarning creates a diagnostic for potential memory leaks.
func MemoryLeakWarning(resource string, span position.Span) Diagnostic {
	return NewDiagnosticBuilder().
		Warning().
		WithCode("W003").
		WithCategory(CategoryMemoryLeak).
		WithMessagef("potential memory leak: '%s' may not be properly released", resource).
		WithSpan(span).
		WithExplanationf("The resource '%s' is allocated but may not be released in all code paths.", resource).
		AddExample("defer resource.close(); // Ensure resource is released").
		AddManualFix("Add explicit resource cleanup").
		AddManualFix("Use RAII pattern").
		AddSeeAlso("resource-management").
		AddSeeAlso("memory-safety").
		Build()
}

// PerformanceWarning creates a diagnostic for performance issues.
func PerformanceWarning(issue string, span position.Span, suggestion string) Diagnostic {
	return NewDiagnosticBuilder().
		Warning().
		WithCode("W004").
		WithCategory(CategoryPerformance).
		WithMessagef("performance issue: %s", issue).
		WithSpan(span).
		WithExplanation("This code pattern may have performance implications.").
		AddManualFix(suggestion).
		AddSeeAlso("performance-optimization").
		AddSeeAlso("best-practices").
		Build()
}

// SecurityWarning creates a diagnostic for security issues.
func SecurityWarning(issue string, span position.Span) Diagnostic {
	return NewDiagnosticBuilder().
		Warning().
		WithCode("S001").
		WithCategory(CategorySecurity).
		WithMessagef("security warning: %s", issue).
		WithSpan(span).
		WithExplanation("This code pattern may have security implications.").
		AddManualFix("Review and validate input").
		AddManualFix("Use secure alternatives").
		AddSeeAlso("security-guidelines").
		AddSeeAlso("safe-coding").
		Build()
}

// StyleWarning creates a diagnostic for style violations.
func StyleWarning(issue string, span position.Span, suggestion string) Diagnostic {
	return NewDiagnosticBuilder().
		Warning().
		WithCode("S002").
		WithCategory(CategoryStyle).
		WithMessagef("style: %s", issue).
		WithSpan(span).
		WithExplanation("This code does not follow the recommended style guidelines.").
		AddManualFix(suggestion).
		AddSeeAlso("style-guide").
		AddSeeAlso("code-formatting").
		Build()
}

// GenericInstantiationError creates a diagnostic for generic instantiation errors.
func GenericInstantiationError(typeName string, typeArgs []string, span position.Span) Diagnostic {
	return NewDiagnosticBuilder().
		Error().
		WithCode("E004").
		WithCategory(CategoryGenericError).
		WithMessagef("cannot instantiate generic type '%s' with arguments [%s]", typeName, strings.Join(typeArgs, ", ")).
		WithSpan(span).
		WithExplanationf("The generic type '%s' cannot be instantiated with the provided type arguments.", typeName).
		AddExample(fmt.Sprintf("%s<CorrectType> // Use compatible type arguments", typeName)).
		AddManualFix("Check type constraints").
		AddManualFix("Provide correct type arguments").
		AddSeeAlso("generics").
		AddSeeAlso("type-constraints").
		Build()
}

// CircularDependencyError creates a diagnostic for circular dependencies.
func CircularDependencyError(cycle []string, span position.Span) Diagnostic {
	cycleStr := strings.Join(cycle, " -> ")

	return NewDiagnosticBuilder().
		Error().
		WithCode("E005").
		WithCategory(CategoryRedefinition).
		WithMessagef("circular dependency detected: %s", cycleStr).
		WithSpan(span).
		WithExplanationf("A circular dependency was detected in the dependency chain: %s", cycleStr).
		AddManualFix("Break the circular dependency by restructuring the code").
		AddManualFix("Use forward declarations").
		AddSeeAlso("circular-dependencies").
		AddSeeAlso("module-system").
		Build()
}
