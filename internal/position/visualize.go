// Package position provides unified source code position tracking.
// for the Orizon compiler. This system enables precise error reporting,
// debugging support, and source map generation.
//
// This file contains visualization tools for debugging and development assistance.
package position

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// SpanHighlighter provides tools for highlighting source code spans.
type SpanHighlighter struct {
	sourceMap *SourceMap
}

// NewSpanHighlighter creates a new span highlighter.
func NewSpanHighlighter(sourceMap *SourceMap) *SpanHighlighter {
	return &SpanHighlighter{
		sourceMap: sourceMap,
	}
}

// HighlightSpan returns a string representation of the source code.
// with the specified span highlighted using ASCII art.
func (sh *SpanHighlighter) HighlightSpan(span Span) string {
	if !span.IsValid() {
		return "Invalid span"
	}

	file := sh.sourceMap.GetFile(span.Start.Filename)
	if file == nil {
		return fmt.Sprintf("File not found: %s", span.Start.Filename)
	}

	var result strings.Builder

	result.WriteString(fmt.Sprintf("File: %s\n", span.Start.Filename))
	result.WriteString(fmt.Sprintf("Span: %s\n", span.String()))
	result.WriteString("\n")

	// Calculate range of lines to show (with context).
	startLine := max(1, span.Start.Line-2)
	endLine := min(len(file.Lines), span.End.Line+2)

	// Add line numbers and content.
	for lineNum := startLine; lineNum <= endLine; lineNum++ {
		line := file.GetLine(lineNum)
		result.WriteString(fmt.Sprintf("%4d | %s\n", lineNum, line))

		// Add highlighting for the current span.
		if lineNum >= span.Start.Line && lineNum <= span.End.Line {
			sh.addHighlighting(&result, lineNum, line, span)
		}
	}

	return result.String()
}

// addHighlighting adds ASCII highlighting under the relevant part of the line.
func (sh *SpanHighlighter) addHighlighting(result *strings.Builder, lineNum int, line string, span Span) {
	result.WriteString("     | ")

	if lineNum == span.Start.Line && lineNum == span.End.Line {
		// Single line span.
		sh.addSingleLineHighlight(result, line, span.Start.Column, span.End.Column)
	} else if lineNum == span.Start.Line {
		// Start of multi-line span.
		sh.addSingleLineHighlight(result, line, span.Start.Column, utf8.RuneCountInString(line)+1)
	} else if lineNum == span.End.Line {
		// End of multi-line span.
		sh.addSingleLineHighlight(result, line, 1, span.End.Column)
	} else {
		// Middle of multi-line span.
		sh.addSingleLineHighlight(result, line, 1, utf8.RuneCountInString(line)+1)
	}

	result.WriteString("\n")
}

// addSingleLineHighlight adds highlighting for a single line between given columns.
func (sh *SpanHighlighter) addSingleLineHighlight(result *strings.Builder, line string, startCol, endCol int) {
	runes := []rune(line)

	// Add spaces before the highlight.
	for i := 1; i < startCol; i++ {
		if i <= len(runes) && runes[i-1] == '\t' {
			result.WriteString("\t")
		} else {
			result.WriteString(" ")
		}
	}

	// Add the highlight.
	highlightLen := endCol - startCol
	if highlightLen > 0 {
		result.WriteString(strings.Repeat("^", min(highlightLen, len(runes)-startCol+1)))
	}
}

// HighlightMultipleSpans highlights multiple spans in the source code.
func (sh *SpanHighlighter) HighlightMultipleSpans(spans []Span) string {
	if len(spans) == 0 {
		return "No spans to highlight"
	}

	var result strings.Builder

	result.WriteString("Multiple Span Highlighting\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")

	for i, span := range spans {
		result.WriteString(fmt.Sprintf("Span %d:\n", i+1))
		result.WriteString(sh.HighlightSpan(span))
		result.WriteString("\n")
	}

	return result.String()
}

// ErrorVisualizer provides visualization for compiler errors.
type ErrorVisualizer struct {
	highlighter *SpanHighlighter
}

// NewErrorVisualizer creates a new error visualizer.
func NewErrorVisualizer(sourceMap *SourceMap) *ErrorVisualizer {
	return &ErrorVisualizer{
		highlighter: NewSpanHighlighter(sourceMap),
	}
}

// VisualizeError creates a visual representation of an error.
func (ev *ErrorVisualizer) VisualizeError(err Error) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Error: %s\n", err.String()))
	result.WriteString(strings.Repeat("-", 50) + "\n")

	// Create a single-character span at the error position.
	span := Span{
		Start: err.Pos,
		End: Position{
			Filename: err.Pos.Filename,
			Line:     err.Pos.Line,
			Column:   err.Pos.Column + 1,
			Offset:   err.Pos.Offset + 1,
		},
	}

	result.WriteString(ev.highlighter.HighlightSpan(span))

	return result.String()
}

// VisualizeDiagnostic creates a visual representation of a diagnostic.
func (ev *ErrorVisualizer) VisualizeDiagnostic(diag *Diagnostic) string {
	var result strings.Builder

	result.WriteString("Diagnostic Report\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")

	if diag.HasErrors() {
		result.WriteString(fmt.Sprintf("Errors (%d):\n", diag.ErrorCount()))
		result.WriteString(strings.Repeat("-", 20) + "\n")

		for i, err := range diag.Errors {
			result.WriteString(fmt.Sprintf("\nError %d:\n", i+1))
			result.WriteString(ev.VisualizeError(err))
		}
	}

	if diag.HasWarnings() {
		result.WriteString(fmt.Sprintf("\nWarnings (%d):\n", diag.WarningCount()))
		result.WriteString(strings.Repeat("-", 20) + "\n")

		for i, warning := range diag.Warnings {
			result.WriteString(fmt.Sprintf("\nWarning %d: %s\n", i+1, warning.String()))

			// Create a single-character span at the warning position.
			span := Span{
				Start: warning.Pos,
				End: Position{
					Filename: warning.Pos.Filename,
					Line:     warning.Pos.Line,
					Column:   warning.Pos.Column + 1,
					Offset:   warning.Pos.Offset + 1,
				},
			}
			result.WriteString(ev.highlighter.HighlightSpan(span))
		}
	}

	if !diag.HasErrors() && !diag.HasWarnings() {
		result.WriteString("No errors or warnings.\n")
	}

	return result.String()
}

// CodeInspector provides detailed code inspection tools.
type CodeInspector struct {
	sourceMap *SourceMap
}

// NewCodeInspector creates a new code inspector.
func NewCodeInspector(sourceMap *SourceMap) *CodeInspector {
	return &CodeInspector{
		sourceMap: sourceMap,
	}
}

// InspectFile provides a detailed view of a source file.
func (ci *CodeInspector) InspectFile(filename string) string {
	file := ci.sourceMap.GetFile(filename)
	if file == nil {
		return fmt.Sprintf("File not found: %s", filename)
	}

	var result strings.Builder

	result.WriteString(fmt.Sprintf("File Inspection: %s\n", filename))
	result.WriteString(strings.Repeat("=", 50) + "\n")
	result.WriteString(fmt.Sprintf("Total lines: %d\n", len(file.Lines)))
	result.WriteString(fmt.Sprintf("Total characters: %d\n", len(file.Content)))
	result.WriteString("\n")

	// Show line-by-line breakdown.
	for i, line := range file.Lines {
		lineNum := i + 1
		result.WriteString(fmt.Sprintf("%4d | %s\n", lineNum, line))

		// Show character positions for first few lines as example.
		if lineNum <= 3 && len(line) > 0 {
			result.WriteString("     | ")

			for j := 0; j < len(line) && j < 50; j++ {
				if j%10 == 0 {
					result.WriteString(fmt.Sprintf("%d", j/10))
				} else {
					result.WriteString(" ")
				}
			}

			result.WriteString("\n")
			result.WriteString("     | ")

			for j := 0; j < len(line) && j < 50; j++ {
				result.WriteString(fmt.Sprintf("%d", j%10))
			}

			result.WriteString("\n")
		}
	}

	return result.String()
}

// GetFileStatistics returns statistics about a source file.
func (ci *CodeInspector) GetFileStatistics(filename string) map[string]interface{} {
	file := ci.sourceMap.GetFile(filename)
	if file == nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("File not found: %s", filename),
		}
	}

	stats := map[string]interface{}{
		"filename":     filename,
		"total_lines":  len(file.Lines),
		"total_chars":  len(file.Content),
		"empty_lines":  0,
		"max_line_len": 0,
		"min_line_len": -1,
		"avg_line_len": 0.0,
	}

	totalLineLen := 0
	emptyLines := 0
	maxLineLen := 0
	minLineLen := -1

	for _, line := range file.Lines {
		lineLen := len(line)
		totalLineLen += lineLen

		if lineLen == 0 {
			emptyLines++
		}

		if lineLen > maxLineLen {
			maxLineLen = lineLen
		}

		if minLineLen == -1 || lineLen < minLineLen {
			minLineLen = lineLen
		}
	}

	stats["empty_lines"] = emptyLines
	stats["max_line_len"] = maxLineLen
	stats["min_line_len"] = minLineLen

	if len(file.Lines) > 0 {
		stats["avg_line_len"] = float64(totalLineLen) / float64(len(file.Lines))
	}

	return stats
}

// Helper functions.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
