// Advanced diagnostic system for Orizon compiler.
// Provides comprehensive error reporting, warnings, and static analysis.

package diagnostic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orizon-lang/orizon/internal/position"
)

// DiagnosticLevel represents the severity level of a diagnostic message.
type DiagnosticLevel int

const (
	DiagnosticError DiagnosticLevel = iota
	DiagnosticWarning
	DiagnosticInfo
	DiagnosticHint
)

func (dl DiagnosticLevel) String() string {
	switch dl {
	case DiagnosticError:
		return "error"
	case DiagnosticWarning:
		return "warning"
	case DiagnosticInfo:
		return "info"
	case DiagnosticHint:
		return "hint"
	default:
		return "unknown"
	}
}

// DiagnosticCategory represents the category of diagnostic.
type DiagnosticCategory int

const (
	DiagnosticSyntax DiagnosticCategory = iota
	DiagnosticType
	DiagnosticSemantic
	DiagnosticPerformance
	DiagnosticStyle
	DiagnosticSecurity
)

func (dc DiagnosticCategory) String() string {
	switch dc {
	case DiagnosticSyntax:
		return "syntax"
	case DiagnosticType:
		return "type"
	case DiagnosticSemantic:
		return "semantic"
	case DiagnosticPerformance:
		return "performance"
	case DiagnosticStyle:
		return "style"
	case DiagnosticSecurity:
		return "security"
	default:
		return "unknown"
	}
}

// Diagnostic represents a single diagnostic message.
type Diagnostic struct {
	Code        string
	Title       string
	Message     string
	Suggestions []Suggestion
	RelatedInfo []RelatedInformation
	Tags        []string
	Span        position.Span
	Level       DiagnosticLevel
	Category    DiagnosticCategory
}

// Suggestion represents a suggested fix for a diagnostic.
type Suggestion struct {
	Title       string
	Description string
	Edits       []TextEdit
}

// TextEdit represents a text replacement.
type TextEdit struct {
	NewText     string
	Description string
	Span        position.Span
}

// RelatedInformation provides additional context for a diagnostic.
type RelatedInformation struct {
	Message string
	Span    position.Span
}

// DiagnosticBuilder helps construct diagnostic messages with fluent API.
type DiagnosticBuilder struct {
	diagnostic *Diagnostic
}

// NewDiagnostic creates a new diagnostic builder.
func NewDiagnostic() *DiagnosticBuilder {
	return &DiagnosticBuilder{
		diagnostic: &Diagnostic{
			Suggestions: make([]Suggestion, 0),
			RelatedInfo: make([]RelatedInformation, 0),
			Tags:        make([]string, 0),
		},
	}
}

func (db *DiagnosticBuilder) Error() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticError

	return db
}

func (db *DiagnosticBuilder) Warning() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticWarning

	return db
}

func (db *DiagnosticBuilder) Info() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticInfo

	return db
}

func (db *DiagnosticBuilder) Hint() *DiagnosticBuilder {
	db.diagnostic.Level = DiagnosticHint

	return db
}

func (db *DiagnosticBuilder) Syntax() *DiagnosticBuilder {
	db.diagnostic.Category = DiagnosticSyntax

	return db
}

func (db *DiagnosticBuilder) Type() *DiagnosticBuilder {
	db.diagnostic.Category = DiagnosticType

	return db
}

func (db *DiagnosticBuilder) Semantic() *DiagnosticBuilder {
	db.diagnostic.Category = DiagnosticSemantic

	return db
}

func (db *DiagnosticBuilder) Performance() *DiagnosticBuilder {
	db.diagnostic.Category = DiagnosticPerformance

	return db
}

func (db *DiagnosticBuilder) Style() *DiagnosticBuilder {
	db.diagnostic.Category = DiagnosticStyle

	return db
}

func (db *DiagnosticBuilder) Security() *DiagnosticBuilder {
	db.diagnostic.Category = DiagnosticSecurity

	return db
}

func (db *DiagnosticBuilder) Code(code string) *DiagnosticBuilder {
	db.diagnostic.Code = code

	return db
}

func (db *DiagnosticBuilder) Title(title string) *DiagnosticBuilder {
	db.diagnostic.Title = title

	return db
}

func (db *DiagnosticBuilder) Message(message string) *DiagnosticBuilder {
	db.diagnostic.Message = message

	return db
}

func (db *DiagnosticBuilder) Span(span position.Span) *DiagnosticBuilder {
	db.diagnostic.Span = span

	return db
}

func (db *DiagnosticBuilder) Suggest(title, description string, edits ...TextEdit) *DiagnosticBuilder {
	suggestion := Suggestion{
		Title:       title,
		Description: description,
		Edits:       edits,
	}
	db.diagnostic.Suggestions = append(db.diagnostic.Suggestions, suggestion)

	return db
}

func (db *DiagnosticBuilder) Related(span position.Span, message string) *DiagnosticBuilder {
	related := RelatedInformation{
		Span:    span,
		Message: message,
	}
	db.diagnostic.RelatedInfo = append(db.diagnostic.RelatedInfo, related)

	return db
}

func (db *DiagnosticBuilder) Tag(tag string) *DiagnosticBuilder {
	db.diagnostic.Tags = append(db.diagnostic.Tags, tag)

	return db
}

func (db *DiagnosticBuilder) Build() *Diagnostic {
	return db.diagnostic
}

// DiagnosticEngine manages the collection and processing of diagnostics.
type DiagnosticEngine struct {
	diagnostics []Diagnostic
	config      DiagnosticConfig
}

// DiagnosticConfig controls diagnostic behavior.
type DiagnosticConfig struct {
	IgnoreCategories  []DiagnosticCategory
	IgnoreCodes       []string
	MaxErrors         int
	WarningsAsErrors  bool
	VerboseOutput     bool
	ShowSuggestions   bool
	ShowRelatedInfo   bool
	EnablePerformance bool
	EnableStyle       bool
	EnableSecurity    bool
}

// NewDiagnosticEngine creates a new diagnostic engine.
func NewDiagnosticEngine(config DiagnosticConfig) *DiagnosticEngine {
	return &DiagnosticEngine{
		diagnostics: make([]Diagnostic, 0),
		config:      config,
	}
}

// AddDiagnostic adds a diagnostic to the engine.
func (de *DiagnosticEngine) AddDiagnostic(diagnostic *Diagnostic) {
	// Check if diagnostic should be ignored.
	if de.shouldIgnore(diagnostic) {
		return
	}

	// Convert warnings to errors if configured.
	if de.config.WarningsAsErrors && diagnostic.Level == DiagnosticWarning {
		diagnostic.Level = DiagnosticError
	}

	de.diagnostics = append(de.diagnostics, *diagnostic)

	// Stop adding diagnostics if max errors reached.
	if len(de.GetErrors()) >= de.config.MaxErrors {
		// Add a special diagnostic indicating truncation.
		truncationDiag := NewDiagnostic().
			Error().
			Code("E0001").
			Title("Too many errors").
			Message(fmt.Sprintf("Stopping after %d errors", de.config.MaxErrors)).
			Build()
		de.diagnostics = append(de.diagnostics, *truncationDiag)
	}
}

// shouldIgnore checks if a diagnostic should be ignored based on config.
func (de *DiagnosticEngine) shouldIgnore(diagnostic *Diagnostic) bool {
	// Check ignored categories.
	for _, cat := range de.config.IgnoreCategories {
		if diagnostic.Category == cat {
			return true
		}
	}

	// Check ignored codes.
	for _, code := range de.config.IgnoreCodes {
		if diagnostic.Code == code {
			return true
		}
	}

	// Check if category is disabled.
	switch diagnostic.Category {
	case DiagnosticPerformance:
		return !de.config.EnablePerformance
	case DiagnosticStyle:
		return !de.config.EnableStyle
	case DiagnosticSecurity:
		return !de.config.EnableSecurity
	}

	return false
}

// GetDiagnostics returns all diagnostics.
func (de *DiagnosticEngine) GetDiagnostics() []Diagnostic {
	return de.diagnostics
}

// GetErrors returns only error-level diagnostics.
func (de *DiagnosticEngine) GetErrors() []Diagnostic {
	errors := make([]Diagnostic, 0)

	for _, diag := range de.diagnostics {
		if diag.Level == DiagnosticError {
			errors = append(errors, diag)
		}
	}

	return errors
}

// GetWarnings returns only warning-level diagnostics.
func (de *DiagnosticEngine) GetWarnings() []Diagnostic {
	warnings := make([]Diagnostic, 0)

	for _, diag := range de.diagnostics {
		if diag.Level == DiagnosticWarning {
			warnings = append(warnings, diag)
		}
	}

	return warnings
}

// HasErrors returns true if there are any errors.
func (de *DiagnosticEngine) HasErrors() bool {
	return len(de.GetErrors()) > 0
}

// Clear removes all diagnostics.
func (de *DiagnosticEngine) Clear() {
	de.diagnostics = de.diagnostics[:0]
}

// SortDiagnostics sorts diagnostics by position and severity.
func (de *DiagnosticEngine) SortDiagnostics() {
	sort.Slice(de.diagnostics, func(i, j int) bool {
		a, b := de.diagnostics[i], de.diagnostics[j]

		// First by file, then by line, then by column.
		if a.Span.Start.Filename != b.Span.Start.Filename {
			return a.Span.Start.Filename < b.Span.Start.Filename
		}

		if a.Span.Start.Line != b.Span.Start.Line {
			return a.Span.Start.Line < b.Span.Start.Line
		}

		if a.Span.Start.Column != b.Span.Start.Column {
			return a.Span.Start.Column < b.Span.Start.Column
		}

		// Then by severity (errors first).
		return a.Level < b.Level
	})
}

// FormatDiagnostics returns a formatted string representation of all diagnostics.
func (de *DiagnosticEngine) FormatDiagnostics() string {
	if len(de.diagnostics) == 0 {
		return ""
	}

	de.SortDiagnostics()

	var result strings.Builder

	for i, diag := range de.diagnostics {
		if i > 0 {
			result.WriteString("\n")
		}

		result.WriteString(de.formatSingleDiagnostic(&diag))
	}

	// Add summary.
	result.WriteString(de.formatSummary())

	return result.String()
}

// formatSingleDiagnostic formats a single diagnostic.
func (de *DiagnosticEngine) formatSingleDiagnostic(diag *Diagnostic) string {
	var result strings.Builder

	// Main diagnostic line.
	result.WriteString(fmt.Sprintf("%s:%d:%d: %s[%s]: %s\n",
		diag.Span.Start.Filename,
		diag.Span.Start.Line,
		diag.Span.Start.Column,
		diag.Level.String(),
		diag.Code,
		diag.Title,
	))

	// Message.
	if diag.Message != "" {
		result.WriteString(fmt.Sprintf("  %s\n", diag.Message))
	}

	// Show suggestions if enabled.
	if de.config.ShowSuggestions && len(diag.Suggestions) > 0 {
		result.WriteString("  Suggestions:\n")

		for _, suggestion := range diag.Suggestions {
			result.WriteString(fmt.Sprintf("    - %s: %s\n", suggestion.Title, suggestion.Description))
		}
	}

	// Show related info if enabled.
	if de.config.ShowRelatedInfo && len(diag.RelatedInfo) > 0 {
		result.WriteString("  Related:\n")

		for _, related := range diag.RelatedInfo {
			result.WriteString(fmt.Sprintf("    %s:%d:%d: %s\n",
				related.Span.Start.Filename,
				related.Span.Start.Line,
				related.Span.Start.Column,
				related.Message,
			))
		}
	}

	return result.String()
}

// formatSummary formats a summary of all diagnostics.
func (de *DiagnosticEngine) formatSummary() string {
	errorCount := len(de.GetErrors())
	warningCount := len(de.GetWarnings())

	if errorCount == 0 && warningCount == 0 {
		return "\n✅ No issues found."
	}

	var parts []string
	if errorCount > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", errorCount))
	}

	if warningCount > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", warningCount))
	}

	return fmt.Sprintf("\n📊 Found %s.", strings.Join(parts, ", "))
}

// CommonDiagnostics provides factory functions for common diagnostic patterns.
type CommonDiagnostics struct{}

// UnexpectedToken creates a diagnostic for unexpected token errors.
func (cd *CommonDiagnostics) UnexpectedToken(span position.Span, expected, actual string) *Diagnostic {
	return NewDiagnostic().
		Error().
		Syntax().
		Code("E1001").
		Title("Unexpected token").
		Message(fmt.Sprintf("Expected '%s', found '%s'", expected, actual)).
		Span(span).
		Build()
}

// UndefinedVariable creates a diagnostic for undefined variable errors.
func (cd *CommonDiagnostics) UndefinedVariable(span position.Span, name string) *Diagnostic {
	return NewDiagnostic().
		Error().
		Semantic().
		Code("E2001").
		Title("Undefined variable").
		Message(fmt.Sprintf("Variable '%s' is not defined", name)).
		Span(span).
		Build()
}

// TypeMismatch creates a diagnostic for type mismatch errors.
func (cd *CommonDiagnostics) TypeMismatch(span position.Span, expected, actual string) *Diagnostic {
	return NewDiagnostic().
		Error().
		Type().
		Code("E3001").
		Title("Type mismatch").
		Message(fmt.Sprintf("Expected type '%s', found '%s'", expected, actual)).
		Span(span).
		Build()
}

// UnusedVariable creates a diagnostic for unused variable warnings.
func (cd *CommonDiagnostics) UnusedVariable(span position.Span, name string) *Diagnostic {
	return NewDiagnostic().
		Warning().
		Style().
		Code("W4001").
		Title("Unused variable").
		Message(fmt.Sprintf("Variable '%s' is declared but never used", name)).
		Span(span).
		Suggest("Remove variable", "Remove the unused variable declaration").
		Tag("unused").
		Build()
}

// DeadCode creates a diagnostic for dead code warnings.
func (cd *CommonDiagnostics) DeadCode(span position.Span) *Diagnostic {
	return NewDiagnostic().
		Warning().
		Semantic().
		Code("W5001").
		Title("Dead code").
		Message("This code will never be executed").
		Span(span).
		Suggest("Remove dead code", "Remove the unreachable code").
		Tag("dead-code").
		Build()
}

// PerformanceIssue creates a diagnostic for performance issues.
func (cd *CommonDiagnostics) PerformanceIssue(span position.Span, issue, suggestion string) *Diagnostic {
	return NewDiagnostic().
		Warning().
		Performance().
		Code("W6001").
		Title("Performance issue").
		Message(issue).
		Span(span).
		Suggest("Optimize", suggestion).
		Tag("performance").
		Build()
}

// Global instance for convenience.
var Common = &CommonDiagnostics{}
