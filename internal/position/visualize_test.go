package position

import (
	"strings"
	"testing"
)

// TestSpanHighlighterBasic tests basic span highlighting functionality.
func TestSpanHighlighterBasic(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n\treturn 0\n}"
	sourceMap.AddFile("test.oriz", content)

	highlighter := NewSpanHighlighter(sourceMap)

	// Test single line highlighting.
	span := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 1, Offset: 0},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
	}

	result := highlighter.HighlightSpan(span)

	expectedStrings := []string{
		"File: test.oriz",
		"Span: test.oriz:1:1-5",
		"func main() {",
		"^^^^",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("HighlightSpan result should contain '%s', got:\n%s", expected, result)
		}
	}
}

// TestSpanHighlighterMultiLine tests multi-line span highlighting.
func TestSpanHighlighterMultiLine(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n\treturn 0\n}"
	sourceMap.AddFile("test.oriz", content)

	highlighter := NewSpanHighlighter(sourceMap)

	// Test multi-line highlighting.
	span := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 6, Offset: 5},
		End:   Position{Filename: "test.oriz", Line: 2, Column: 10, Offset: 22},
	}

	result := highlighter.HighlightSpan(span)

	// Check basic structure.
	if !strings.Contains(result, "File: test.oriz") {
		t.Error("Result should contain filename")
	}

	if !strings.Contains(result, "main() {") {
		t.Error("Result should contain first line content")
	}

	if !strings.Contains(result, "println") {
		t.Error("Result should contain second line content")
	}
}

// TestSpanHighlighterInvalidSpan tests handling of invalid spans.
func TestSpanHighlighterInvalidSpan(t *testing.T) {
	sourceMap := NewSourceMap()
	highlighter := NewSpanHighlighter(sourceMap)

	// Test invalid span.
	invalidSpan := Span{}
	result := highlighter.HighlightSpan(invalidSpan)

	if result != "Invalid span" {
		t.Errorf("Expected 'Invalid span', got '%s'", result)
	}
}

// TestSpanHighlighterMissingFile tests handling of missing files.
func TestSpanHighlighterMissingFile(t *testing.T) {
	sourceMap := NewSourceMap()
	highlighter := NewSpanHighlighter(sourceMap)

	// Test span for missing file.
	span := Span{
		Start: Position{Filename: "missing.oriz", Line: 1, Column: 1, Offset: 0},
		End:   Position{Filename: "missing.oriz", Line: 1, Column: 5, Offset: 4},
	}

	result := highlighter.HighlightSpan(span)
	expected := "File not found: missing.oriz"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestHighlightMultipleSpans tests highlighting of multiple spans.
func TestHighlightMultipleSpans(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	sourceMap.AddFile("test.oriz", content)

	highlighter := NewSpanHighlighter(sourceMap)

	spans := []Span{
		{
			Start: Position{Filename: "test.oriz", Line: 1, Column: 1, Offset: 0},
			End:   Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
		},
		{
			Start: Position{Filename: "test.oriz", Line: 2, Column: 2, Offset: 15},
			End:   Position{Filename: "test.oriz", Line: 2, Column: 9, Offset: 22},
		},
	}

	result := highlighter.HighlightMultipleSpans(spans)

	expectedStrings := []string{
		"Multiple Span Highlighting",
		"Span 1:",
		"Span 2:",
		"func",
		"println",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("HighlightMultipleSpans result should contain '%s'", expected)
		}
	}
}

// TestHighlightMultipleSpansEmpty tests empty span list handling.
func TestHighlightMultipleSpansEmpty(t *testing.T) {
	sourceMap := NewSourceMap()
	highlighter := NewSpanHighlighter(sourceMap)

	result := highlighter.HighlightMultipleSpans([]Span{})
	expected := "No spans to highlight"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestErrorVisualizer tests error visualization functionality.
func TestErrorVisualizer(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	sourceMap.AddFile("test.oriz", content)

	visualizer := NewErrorVisualizer(sourceMap)

	err := Error{
		Pos: Position{
			Filename: "test.oriz",
			Line:     1,
			Column:   5,
			Offset:   4,
		},
		Message: "unexpected token",
		Kind:    "syntax",
	}

	result := visualizer.VisualizeError(err)

	expectedStrings := []string{
		"Error: test.oriz:1:5: syntax: unexpected token",
		"func main() {",
		"^",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("VisualizeError result should contain '%s', got:\n%s", expected, result)
		}
	}
}

// TestVisualizeDiagnosticWithErrorsAndWarnings tests complete diagnostic visualization.
func TestVisualizeDiagnosticWithErrorsAndWarnings(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	sourceMap.AddFile("test.oriz", content)

	visualizer := NewErrorVisualizer(sourceMap)
	diag := NewDiagnostic()

	pos1 := Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4}
	pos2 := Position{Filename: "test.oriz", Line: 2, Column: 2, Offset: 15}

	diag.AddError(pos1, "syntax", "unexpected token")
	diag.AddWarning(pos2, "unused", "variable not used")

	result := visualizer.VisualizeDiagnostic(diag)

	expectedStrings := []string{
		"Diagnostic Report",
		"Errors (1):",
		"Error 1:",
		"unexpected token",
		"Warnings (1):",
		"Warning 1:",
		"variable not used",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("VisualizeDiagnostic result should contain '%s'", expected)
		}
	}
}

// TestVisualizeDiagnosticEmpty tests empty diagnostic visualization.
func TestVisualizeDiagnosticEmpty(t *testing.T) {
	sourceMap := NewSourceMap()
	visualizer := NewErrorVisualizer(sourceMap)
	diag := NewDiagnostic()

	result := visualizer.VisualizeDiagnostic(diag)

	if !strings.Contains(result, "No errors or warnings") {
		t.Error("Empty diagnostic should show 'No errors or warnings'")
	}
}

// TestVisualizeDiagnosticOnlyErrors tests diagnostic with only errors.
func TestVisualizeDiagnosticOnlyErrors(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	sourceMap.AddFile("test.oriz", content)

	visualizer := NewErrorVisualizer(sourceMap)
	diag := NewDiagnostic()

	pos := Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4}
	diag.AddError(pos, "syntax", "unexpected token")

	result := visualizer.VisualizeDiagnostic(diag)

	expectedStrings := []string{
		"Diagnostic Report",
		"Errors (1):",
		"Error 1:",
		"unexpected token",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("VisualizeDiagnostic result should contain '%s'", expected)
		}
	}

	// Should not contain warnings section.
	if strings.Contains(result, "Warnings") {
		t.Error("Result should not contain warnings section when there are no warnings")
	}
}

// TestVisualizeDiagnosticOnlyWarnings tests diagnostic with only warnings.
func TestVisualizeDiagnosticOnlyWarnings(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	sourceMap.AddFile("test.oriz", content)

	visualizer := NewErrorVisualizer(sourceMap)
	diag := NewDiagnostic()

	pos := Position{Filename: "test.oriz", Line: 2, Column: 2, Offset: 15}
	diag.AddWarning(pos, "unused", "variable not used")

	result := visualizer.VisualizeDiagnostic(diag)

	expectedStrings := []string{
		"Diagnostic Report",
		"Warnings (1):",
		"Warning 1:",
		"variable not used",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("VisualizeDiagnostic result should contain '%s'", expected)
		}
	}

	// Should not contain errors section.
	if strings.Contains(result, "Errors") {
		t.Error("Result should not contain errors section when there are no errors")
	}
}

// TestCodeInspectorBasic tests basic code inspection functionality.
func TestCodeInspectorBasic(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n\treturn 0\n}"
	sourceMap.AddFile("test.oriz", content)

	inspector := NewCodeInspector(sourceMap)

	result := inspector.InspectFile("test.oriz")

	expectedStrings := []string{
		"File Inspection: test.oriz",
		"Total lines: 4", // Updated: strings.Split creates 4 elements for 3-line content
		"func main() {",
		"println",
		"return 0",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("InspectFile result should contain '%s'", expected)
		}
	}
}

// TestCodeInspectorMissingFile tests handling of missing files.
func TestCodeInspectorMissingFile(t *testing.T) {
	sourceMap := NewSourceMap()
	inspector := NewCodeInspector(sourceMap)

	result := inspector.InspectFile("missing.oriz")
	expected := "File not found: missing.oriz"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestCodeInspectorWithPositionMarkers tests position marker display.
func TestCodeInspectorWithPositionMarkers(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello\")\n}"
	sourceMap.AddFile("test.oriz", content)

	inspector := NewCodeInspector(sourceMap)
	result := inspector.InspectFile("test.oriz")

	// Should show line numbers and character positions for first few lines.
	if !strings.Contains(result, "func main() {") {
		t.Error("Should contain source content")
	}

	if !strings.Contains(result, "println") {
		t.Error("Should contain println call")
	}
}

// TestGetFileStatisticsComplete tests comprehensive file statistics.
func TestGetFileStatisticsComplete(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n\n\treturn 0\n}"
	sourceMap.AddFile("test.oriz", content)

	inspector := NewCodeInspector(sourceMap)

	stats := inspector.GetFileStatistics("test.oriz")

	expectedKeys := []string{
		"filename", "total_lines", "total_chars", "empty_lines",
		"max_line_len", "min_line_len", "avg_line_len",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Statistics should contain key '%s'", key)
		}
	}

	// Check specific values.
	if stats["filename"] != "test.oriz" {
		t.Errorf("Expected filename 'test.oriz', got '%v'", stats["filename"])
	}

	if stats["total_lines"] != 5 { // Updated: content has 5 lines when split
		t.Errorf("Expected 5 lines, got %v", stats["total_lines"])
	}

	if stats["empty_lines"] != 1 {
		t.Errorf("Expected 1 empty line, got %v", stats["empty_lines"])
	}

	if stats["max_line_len"].(int) <= 0 {
		t.Error("Max line length should be positive")
	}

	if stats["min_line_len"].(int) < 0 {
		t.Error("Min line length should be non-negative")
	}

	if avgLen, ok := stats["avg_line_len"].(float64); !ok || avgLen < 0 {
		t.Error("Average line length should be a positive float")
	}
}

// TestGetFileStatisticsMissingFile tests statistics for missing files.
func TestGetFileStatisticsMissingFile(t *testing.T) {
	sourceMap := NewSourceMap()
	inspector := NewCodeInspector(sourceMap)

	stats := inspector.GetFileStatistics("missing.oriz")

	if errorMsg, exists := stats["error"]; !exists {
		t.Error("Statistics for missing file should contain error")
	} else if errorMsg != "File not found: missing.oriz" {
		t.Errorf("Expected error message about missing file, got '%v'", errorMsg)
	}
}

// TestGetFileStatisticsEmptyFile tests statistics for empty files.
func TestGetFileStatisticsEmptyFile(t *testing.T) {
	sourceMap := NewSourceMap()
	sourceMap.AddFile("empty.oriz", "")

	inspector := NewCodeInspector(sourceMap)
	stats := inspector.GetFileStatistics("empty.oriz")

	if stats["total_lines"] != 1 {
		t.Errorf("Empty file should have 1 line, got %v", stats["total_lines"])
	}

	if stats["total_chars"] != 0 {
		t.Errorf("Empty file should have 0 characters, got %v", stats["total_chars"])
	}

	if stats["empty_lines"] != 1 {
		t.Errorf("Empty file should have 1 empty line, got %v", stats["empty_lines"])
	}
}

// TestGetFileStatisticsSingleLine tests statistics for single-line files.
func TestGetFileStatisticsSingleLine(t *testing.T) {
	sourceMap := NewSourceMap()
	sourceMap.AddFile("single.oriz", "func main() {}")

	inspector := NewCodeInspector(sourceMap)
	stats := inspector.GetFileStatistics("single.oriz")

	if stats["total_lines"] != 1 {
		t.Errorf("Single line file should have 1 line, got %v", stats["total_lines"])
	}

	if stats["empty_lines"] != 0 {
		t.Errorf("Single line file should have 0 empty lines, got %v", stats["empty_lines"])
	}

	if stats["max_line_len"] != stats["min_line_len"] {
		t.Error("In single line file, max and min line length should be equal")
	}
}

// TestHelperFunctions tests the utility functions.
func TestHelperFunctions(t *testing.T) {
	// Test min function with various inputs.
	testCases := []struct {
		a, b, expected int
	}{
		{5, 3, 3},
		{2, 7, 2},
		{10, 10, 10},
		{-5, 3, -5},
		{0, 0, 0},
	}

	for _, tc := range testCases {
		if result := min(tc.a, tc.b); result != tc.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
		}
	}

	// Test max function with various inputs.
	maxTestCases := []struct {
		a, b, expected int
	}{
		{5, 3, 5},
		{2, 7, 7},
		{10, 10, 10},
		{-5, 3, 3},
		{0, 0, 0},
	}

	for _, tc := range maxTestCases {
		if result := max(tc.a, tc.b); result != tc.expected {
			t.Errorf("max(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
		}
	}
}

// TestSpanHighlighterEdgeCases tests edge cases in span highlighting.
func TestSpanHighlighterEdgeCases(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "a\n\nb"
	sourceMap.AddFile("edge.oriz", content)

	highlighter := NewSpanHighlighter(sourceMap)

	// Test span at very end of file.
	span := Span{
		Start: Position{Filename: "edge.oriz", Line: 3, Column: 1, Offset: 3},
		End:   Position{Filename: "edge.oriz", Line: 3, Column: 2, Offset: 4},
	}

	result := highlighter.HighlightSpan(span)
	if !strings.Contains(result, "File: edge.oriz") {
		t.Error("Should handle span at end of file")
	}

	// Test span on empty line.
	emptyLineSpan := Span{
		Start: Position{Filename: "edge.oriz", Line: 2, Column: 1, Offset: 2},
		End:   Position{Filename: "edge.oriz", Line: 2, Column: 1, Offset: 2},
	}

	result = highlighter.HighlightSpan(emptyLineSpan)
	if !strings.Contains(result, "File: edge.oriz") {
		t.Error("Should handle span on empty line")
	}
}

// TestErrorVisualizerEdgeCases tests error visualizer edge cases.
func TestErrorVisualizerEdgeCases(t *testing.T) {
	sourceMap := NewSourceMap()
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	sourceMap.AddFile("test.oriz", content)

	visualizer := NewErrorVisualizer(sourceMap)

	// Test error at end of line.
	err := Error{
		Pos: Position{
			Filename: "test.oriz",
			Line:     1,
			Column:   13,
			Offset:   12,
		},
		Message: "expected semicolon",
		Kind:    "syntax",
	}

	result := visualizer.VisualizeError(err)
	if !strings.Contains(result, "expected semicolon") {
		t.Error("Should handle error at end of line")
	}

	// Test error with very long message.
	longErr := Error{
		Pos: Position{
			Filename: "test.oriz",
			Line:     1,
			Column:   1,
			Offset:   0,
		},
		Message: "this is a very long error message that should still be displayed correctly without causing any formatting issues",
		Kind:    "semantic",
	}

	result = visualizer.VisualizeError(longErr)
	if !strings.Contains(result, "very long error message") {
		t.Error("Should handle long error messages")
	}
}
