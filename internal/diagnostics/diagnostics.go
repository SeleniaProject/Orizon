// Package diagnostics provides comprehensive error diagnosis, warnings,
// and code suggestions for the Orizon compiler. This system provides
// detailed analysis of compilation errors with context and fix suggestions.
package diagnostics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orizon-lang/orizon/internal/position"
)

// DiagnosticLevel represents the severity level of a diagnostic
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

// DiagnosticCategory represents the category of diagnostic
type DiagnosticCategory int

const (
	// Syntax and parsing errors
	CategorySyntax DiagnosticCategory = iota
	CategoryParsing

	// Type system errors
	CategoryTypeError
	CategoryTypeInference
	CategoryGenericError

	// Semantic analysis
	CategoryUndefinedVariable
	CategoryUndefinedFunction
	CategoryUndefinedType
	CategoryRedefinition
	CategoryUnusedVariable
	CategoryUnusedFunction
	CategoryUnusedImport

	// Control flow analysis
	CategoryUnreachableCode
	CategoryMissingReturn
	CategoryInfiniteLoop
	CategoryDeadCode

	// Memory and ownership
	CategoryMemoryLeak
	CategoryUseAfterFree
	CategoryDanglingPointer
	CategoryOwnershipViolation

	// Performance warnings
	CategoryPerformance
	CategoryOptimization

	// Style and convention
	CategoryStyle
	CategoryNaming
	CategoryDocumentation

	// Security warnings
	CategorySecurity
	CategoryBufferOverflow
	CategoryNullPointerDeref
)

func (dc DiagnosticCategory) String() string {
	switch dc {
	case CategorySyntax:
		return "syntax"
	case CategoryParsing:
		return "parsing"
	case CategoryTypeError:
		return "type-error"
	case CategoryTypeInference:
		return "type-inference"
	case CategoryGenericError:
		return "generic-error"
	case CategoryUndefinedVariable:
		return "undefined-variable"
	case CategoryUndefinedFunction:
		return "undefined-function"
	case CategoryUndefinedType:
		return "undefined-type"
	case CategoryRedefinition:
		return "redefinition"
	case CategoryUnusedVariable:
		return "unused-variable"
	case CategoryUnusedFunction:
		return "unused-function"
	case CategoryUnusedImport:
		return "unused-import"
	case CategoryUnreachableCode:
		return "unreachable-code"
	case CategoryMissingReturn:
		return "missing-return"
	case CategoryInfiniteLoop:
		return "infinite-loop"
	case CategoryDeadCode:
		return "dead-code"
	case CategoryMemoryLeak:
		return "memory-leak"
	case CategoryUseAfterFree:
		return "use-after-free"
	case CategoryDanglingPointer:
		return "dangling-pointer"
	case CategoryOwnershipViolation:
		return "ownership-violation"
	case CategoryPerformance:
		return "performance"
	case CategoryOptimization:
		return "optimization"
	case CategoryStyle:
		return "style"
	case CategoryNaming:
		return "naming"
	case CategoryDocumentation:
		return "documentation"
	case CategorySecurity:
		return "security"
	case CategoryBufferOverflow:
		return "buffer-overflow"
	case CategoryNullPointerDeref:
		return "null-pointer-deref"
	default:
		return "unknown"
	}
}

// FixSuggestion represents a suggested fix for a diagnostic
type FixSuggestion struct {
	Description string
	Replacement string
	Span        position.Span
	Automatic   bool // Whether this fix can be applied automatically
}

// RelatedInformation provides additional context for a diagnostic
type RelatedInformation struct {
	Message  string
	Location position.Span
}

// Diagnostic represents a comprehensive diagnostic message
type Diagnostic struct {
	ID       string
	Level    DiagnosticLevel
	Category DiagnosticCategory
	Message  string
	Span     position.Span
	Code     string // Error code like "E001", "W042"

	// Context information
	Context     []string      // Lines of source code around the error
	ContextSpan position.Span // Span for the context

	// Additional information
	Explanation    string               // Detailed explanation
	Examples       []string             // Code examples
	RelatedInfo    []RelatedInformation // Related information from other locations
	FixSuggestions []FixSuggestion      // Suggested fixes

	// Help and documentation
	HelpURL string   // URL to documentation
	SeeAlso []string // Related error codes or concepts

	// Internal tracking
	SourceFile string
	StackTrace []string // Internal compiler stack trace
}

// DiagnosticManager manages all diagnostics for a compilation session
type DiagnosticManager struct {
	diagnostics  []Diagnostic
	errorCount   int
	warningCount int
	maxErrors    int
	maxWarnings  int
	sourceCache  map[string][]string // Cache of source file lines
	suppressions map[string]bool     // Suppressed diagnostic categories
}

// NewDiagnosticManager creates a new diagnostic manager
func NewDiagnosticManager() *DiagnosticManager {
	return &DiagnosticManager{
		diagnostics:  make([]Diagnostic, 0),
		maxErrors:    100,
		maxWarnings:  1000,
		sourceCache:  make(map[string][]string),
		suppressions: make(map[string]bool),
	}
}

// SetErrorLimit sets the maximum number of errors before compilation stops
func (dm *DiagnosticManager) SetErrorLimit(limit int) {
	dm.maxErrors = limit
}

// SetWarningLimit sets the maximum number of warnings to report
func (dm *DiagnosticManager) SetWarningLimit(limit int) {
	dm.maxWarnings = limit
}

// SuppressCategory suppresses all diagnostics of a specific category
func (dm *DiagnosticManager) SuppressCategory(category DiagnosticCategory) {
	dm.suppressions[category.String()] = true
}

// AddDiagnostic adds a new diagnostic to the manager
func (dm *DiagnosticManager) AddDiagnostic(diagnostic Diagnostic) {
	// Check if this category is suppressed
	if dm.suppressions[diagnostic.Category.String()] {
		return
	}

	// Check limits
	if diagnostic.Level == DiagnosticError && dm.errorCount >= dm.maxErrors {
		return
	}
	if diagnostic.Level == DiagnosticWarning && dm.warningCount >= dm.maxWarnings {
		return
	}

	// Update counts
	switch diagnostic.Level {
	case DiagnosticError:
		dm.errorCount++
	case DiagnosticWarning:
		dm.warningCount++
	}

	// Enhance diagnostic with context
	dm.enhanceDiagnostic(&diagnostic)

	dm.diagnostics = append(dm.diagnostics, diagnostic)
}

// enhanceDiagnostic adds context information to a diagnostic
func (dm *DiagnosticManager) enhanceDiagnostic(d *Diagnostic) {
	// Add source context
	if d.SourceFile != "" {
		lines := dm.getSourceLines(d.SourceFile)
		if lines != nil {
			d.Context = dm.extractContext(lines, d.Span)
		}
	}

	// Generate automatic fix suggestions based on category
	d.FixSuggestions = dm.generateFixSuggestions(d)

	// Add help information
	d.HelpURL = dm.generateHelpURL(d)
	d.SeeAlso = dm.generateSeeAlso(d)
}

// getSourceLines retrieves and caches source file lines
func (dm *DiagnosticManager) getSourceLines(filename string) []string {
	if lines, exists := dm.sourceCache[filename]; exists {
		return lines
	}

	// In a real implementation, read the file
	// For now, return nil
	return nil
}

// extractContext extracts source code context around a span
func (dm *DiagnosticManager) extractContext(lines []string, span position.Span) []string {
	if len(lines) == 0 {
		return nil
	}

	startLine := max(0, span.Start.Line-3)
	endLine := min(len(lines)-1, span.End.Line+3)

	context := make([]string, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		if i < len(lines) {
			context = append(context, lines[i])
		}
	}

	return context
}

// generateFixSuggestions generates automatic fix suggestions
func (dm *DiagnosticManager) generateFixSuggestions(d *Diagnostic) []FixSuggestion {
	suggestions := make([]FixSuggestion, 0)

	switch d.Category {
	case CategoryUndefinedVariable:
		// Suggest similar variable names
		suggestions = append(suggestions, FixSuggestion{
			Description: "Did you mean to use a similar variable?",
			Automatic:   false,
		})

	case CategoryUnusedVariable:
		// Suggest removing or prefixing with underscore
		suggestions = append(suggestions, FixSuggestion{
			Description: "Remove unused variable",
			Automatic:   true,
		})
		suggestions = append(suggestions, FixSuggestion{
			Description: "Prefix with underscore to indicate intentionally unused",
			Automatic:   true,
		})

	case CategoryTypeError:
		// Suggest type conversions
		suggestions = append(suggestions, FixSuggestion{
			Description: "Add explicit type conversion",
			Automatic:   false,
		})

	case CategoryMissingReturn:
		// Suggest adding return statement
		suggestions = append(suggestions, FixSuggestion{
			Description: "Add return statement",
			Replacement: "return <value>;",
			Automatic:   false,
		})
	}

	return suggestions
}

// generateHelpURL generates help documentation URL
func (dm *DiagnosticManager) generateHelpURL(d *Diagnostic) string {
	baseURL := "https://docs.orizon-lang.org/diagnostics/"
	return baseURL + d.Code
}

// generateSeeAlso generates related concepts
func (dm *DiagnosticManager) generateSeeAlso(d *Diagnostic) []string {
	switch d.Category {
	case CategoryTypeError:
		return []string{"type-system", "type-inference", "generics"}
	case CategoryUndefinedVariable:
		return []string{"scoping", "variable-declaration", "imports"}
	case CategoryMemoryLeak:
		return []string{"ownership", "resource-management", "RAII"}
	default:
		return nil
	}
}

// GetDiagnostics returns all diagnostics
func (dm *DiagnosticManager) GetDiagnostics() []Diagnostic {
	return dm.diagnostics
}

// GetErrorCount returns the number of errors
func (dm *DiagnosticManager) GetErrorCount() int {
	return dm.errorCount
}

// GetWarningCount returns the number of warnings
func (dm *DiagnosticManager) GetWarningCount() int {
	return dm.warningCount
}

// HasErrors returns true if there are any errors
func (dm *DiagnosticManager) HasErrors() bool {
	return dm.errorCount > 0
}

// SortDiagnostics sorts diagnostics by location and severity
func (dm *DiagnosticManager) SortDiagnostics() {
	sort.Slice(dm.diagnostics, func(i, j int) bool {
		a, b := dm.diagnostics[i], dm.diagnostics[j]

		// Sort by file first
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}

		// Then by line
		if a.Span.Start.Line != b.Span.Start.Line {
			return a.Span.Start.Line < b.Span.Start.Line
		}

		// Then by column
		if a.Span.Start.Column != b.Span.Start.Column {
			return a.Span.Start.Column < b.Span.Start.Column
		}

		// Finally by severity (errors first)
		return a.Level < b.Level
	})
}

// FormatDiagnostic formats a diagnostic for display
func (dm *DiagnosticManager) FormatDiagnostic(d Diagnostic, colorize bool) string {
	var result strings.Builder

	// Header line: level, location, message
	if colorize {
		result.WriteString(dm.colorizeLevel(d.Level))
	}
	result.WriteString(d.Level.String())
	if d.Code != "" {
		result.WriteString("[" + d.Code + "]")
	}
	result.WriteString(": " + d.Message)

	// Location information
	result.WriteString("\n  --> " + d.SourceFile)
	result.WriteString(fmt.Sprintf(":%d:%d", d.Span.Start.Line, d.Span.Start.Column))

	// Source context
	if len(d.Context) > 0 {
		result.WriteString("\n")
		for i, line := range d.Context {
			lineNum := d.Span.Start.Line - len(d.Context)/2 + i
			result.WriteString(fmt.Sprintf("%4d | %s\n", lineNum, line))

			// Add pointer to the error location
			if lineNum == d.Span.Start.Line {
				spaces := strings.Repeat(" ", 7+d.Span.Start.Column)
				pointer := strings.Repeat("^", max(1, d.Span.End.Column-d.Span.Start.Column))
				result.WriteString(spaces + pointer + "\n")
			}
		}
	}

	// Explanation
	if d.Explanation != "" {
		result.WriteString("\nExplanation:\n")
		result.WriteString("  " + d.Explanation + "\n")
	}

	// Fix suggestions
	if len(d.FixSuggestions) > 0 {
		result.WriteString("\nSuggested fixes:\n")
		for _, fix := range d.FixSuggestions {
			result.WriteString("  - " + fix.Description)
			if fix.Automatic {
				result.WriteString(" (automatic)")
			}
			result.WriteString("\n")
		}
	}

	// Related information
	if len(d.RelatedInfo) > 0 {
		result.WriteString("\nRelated:\n")
		for _, info := range d.RelatedInfo {
			result.WriteString(fmt.Sprintf("  %s:%d:%d: %s\n",
				info.Location.Start.Filename,
				info.Location.Start.Line,
				info.Location.Start.Column,
				info.Message))
		}
	}

	// Help information
	if d.HelpURL != "" {
		result.WriteString("\nFor more information: " + d.HelpURL + "\n")
	}

	return result.String()
}

// colorizeLevel adds color codes for terminal display
func (dm *DiagnosticManager) colorizeLevel(level DiagnosticLevel) string {
	switch level {
	case DiagnosticError:
		return "\033[31m" // Red
	case DiagnosticWarning:
		return "\033[33m" // Yellow
	case DiagnosticInfo:
		return "\033[34m" // Blue
	case DiagnosticHint:
		return "\033[90m" // Gray
	default:
		return ""
	}
}

// FormatSummary formats a summary of all diagnostics
func (dm *DiagnosticManager) FormatSummary() string {
	if len(dm.diagnostics) == 0 {
		return "No diagnostics."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d error(s) and %d warning(s).",
		dm.errorCount, dm.warningCount))

	// Group by category
	categoryCount := make(map[DiagnosticCategory]int)
	for _, d := range dm.diagnostics {
		categoryCount[d.Category]++
	}

	if len(categoryCount) > 0 {
		result.WriteString("\n\nBreakdown by category:")
		for category, count := range categoryCount {
			result.WriteString(fmt.Sprintf("\n  %s: %d", category.String(), count))
		}
	}

	return result.String()
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetDiagnosticsByLevel returns diagnostics filtered by level
func (dm *DiagnosticManager) GetDiagnosticsByLevel(level DiagnosticLevel) []Diagnostic {
	var filtered []Diagnostic
	for _, diag := range dm.diagnostics {
		if diag.Level == level {
			filtered = append(filtered, diag)
		}
	}
	return filtered
}

// GetDiagnosticsByCategory returns diagnostics filtered by category
func (dm *DiagnosticManager) GetDiagnosticsByCategory(category DiagnosticCategory) []Diagnostic {
	var filtered []Diagnostic
	for _, diag := range dm.diagnostics {
		if diag.Category == category {
			filtered = append(filtered, diag)
		}
	}
	return filtered
}

// GetDiagnosticSummary returns a summary of diagnostics
func (dm *DiagnosticManager) GetDiagnosticSummary() DiagnosticSummary {
	summary := DiagnosticSummary{
		TotalCount:   len(dm.diagnostics),
		ErrorCount:   dm.GetErrorCount(),
		WarningCount: dm.GetWarningCount(),
	}

	// Count info and hint diagnostics
	for _, diag := range dm.diagnostics {
		switch diag.Level {
		case DiagnosticInfo:
			summary.InfoCount++
		case DiagnosticHint:
			summary.HintCount++
		}
	}

	return summary
}

// DiagnosticSummary represents a summary of diagnostics
type DiagnosticSummary struct {
	TotalCount   int
	ErrorCount   int
	WarningCount int
	InfoCount    int
	HintCount    int
}
