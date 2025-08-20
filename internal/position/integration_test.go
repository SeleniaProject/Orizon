package position

import (
	"strings"
	"testing"
)

// TestPositionIntegrationWithAST tests integration between position system and AST.
func TestPositionIntegrationWithAST(t *testing.T) {
	// Create a source map for integration testing.
	sourceMap := NewSourceMap()
	content := `func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}`
	file := sourceMap.AddFile("fibonacci.oriz", content)

	// Test position calculations for various parts of the code.
	tests := []struct {
		name     string
		line     int
		column   int
		expected string
	}{
		{"function keyword", 1, 1, "func"},
		{"function name", 1, 6, "fibonacci"},
		{"parameter", 1, 16, "n"}, // Updated column position
		{"if keyword", 2, 2, "if"},
		{"return keyword", 3, 3, "return"},
		{"recursive call", 5, 9, "fibonacci"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pos := Position{
				Filename: "fibonacci.oriz",
				Line:     test.line,
				Column:   test.column,
				Offset:   file.OffsetFromPosition(Position{Filename: "fibonacci.oriz", Line: test.line, Column: test.column}),
			}

			// Create a span for the expected token.
			span := Span{
				Start: pos,
				End: Position{
					Filename: "fibonacci.oriz",
					Line:     test.line,
					Column:   test.column + len(test.expected),
					Offset:   pos.Offset + len(test.expected),
				},
			}

			// Verify span text matches expected.
			spanText := sourceMap.GetSpanText(span)
			if spanText != test.expected {
				t.Errorf("Expected span text '%s', got '%s'", test.expected, spanText)
			}

			// Test span highlighting.
			highlighter := NewSpanHighlighter(sourceMap)
			result := highlighter.HighlightSpan(span)

			if !strings.Contains(result, test.expected) {
				t.Errorf("Highlight should contain '%s'", test.expected)
			}
		})
	}
}

// TestDiagnosticIntegration tests diagnostic reporting with source context.
func TestDiagnosticIntegration(t *testing.T) {
	sourceMap := NewSourceMap()
	content := `func main() {
	let x = 10
	let y = x + 
	println(x, y)
}`
	sourceMap.AddFile("syntax_error.oriz", content)

	// Create diagnostic with multiple errors and warnings.
	diag := NewDiagnostic()

	// Syntax error: incomplete expression.
	syntaxErrorPos := Position{
		Filename: "syntax_error.oriz",
		Line:     3,
		Column:   13,
		Offset:   27,
	}
	diag.AddError(syntaxErrorPos, "syntax", "unexpected end of line")

	// Warning: unused variable.
	warningPos := Position{
		Filename: "syntax_error.oriz",
		Line:     4,
		Column:   11,
		Offset:   40,
	}
	diag.AddWarning(warningPos, "unused", "variable 'y' may be uninitialized")

	// Test diagnostic visualization.
	visualizer := NewErrorVisualizer(sourceMap)
	result := visualizer.VisualizeDiagnostic(diag)

	expectedElements := []string{
		"Diagnostic Report",
		"Errors (1):",
		"unexpected end of line",
		"Warnings (1):",
		"variable 'y' may be uninitialized",
		"let y = x +",
		"println(x, y)",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Diagnostic should contain '%s'", expected)
		}
	}

	// Test individual error visualization.
	errorResult := visualizer.VisualizeError(diag.Errors[0])
	if !strings.Contains(errorResult, "let y = x +") {
		t.Error("Error visualization should show source line")
	}
}

// TestSourceMapMultipleFiles tests source map with multiple files.
func TestSourceMapMultipleFiles(t *testing.T) {
	sourceMap := NewSourceMap()

	// Add multiple source files.
	mainContent := `import "utils"

func main() {
	utils.hello()
}`
	sourceMap.AddFile("main.oriz", mainContent)

	utilsContent := `func hello() {
	println("Hello, World!")
}`
	sourceMap.AddFile("utils.oriz", utilsContent)

	// Test cross-file position handling.
	mainPos := Position{Filename: "main.oriz", Line: 4, Column: 2, Offset: 30}
	utilsPos := Position{Filename: "utils.oriz", Line: 2, Column: 2, Offset: 15}

	// Test that positions from different files are handled correctly.
	if !mainPos.Before(utilsPos) {
		// Different files should be compared by filename.
		if mainPos.Filename >= utilsPos.Filename {
			t.Error("Position comparison across files should work by filename")
		}
	}

	// Test span operations across files.
	mainSpan := Span{
		Start: Position{Filename: "main.oriz", Line: 1, Column: 1, Offset: 0},
		End:   Position{Filename: "main.oriz", Line: 1, Column: 7, Offset: 6},
	}

	utilsSpan := Span{
		Start: Position{Filename: "utils.oriz", Line: 1, Column: 1, Offset: 0},
		End:   Position{Filename: "utils.oriz", Line: 1, Column: 5, Offset: 4},
	}

	// Spans from different files should not overlap.
	if mainSpan.Overlaps(utilsSpan) {
		t.Error("Spans from different files should not overlap")
	}

	// Test source text retrieval.
	mainText := sourceMap.GetSpanText(mainSpan)
	if mainText != "import" {
		t.Errorf("Expected 'import', got '%s'", mainText)
	}

	utilsText := sourceMap.GetSpanText(utilsSpan)
	if utilsText != "func" {
		t.Errorf("Expected 'func', got '%s'", utilsText)
	}
}

// TestCodeInspectorIntegration tests code inspector with realistic code.
func TestCodeInspectorIntegration(t *testing.T) {
	sourceMap := NewSourceMap()
	content := `// Fibonacci implementation with memoization
type Memo = map[int]int

func fibonacci(n int, memo Memo) int {
	if n <= 1 {
		return n
	}
	
	if val, exists := memo[n]; exists {
		return val
	}
	
	result := fibonacci(n-1, memo) + fibonacci(n-2, memo)
	memo[n] = result
	return result
}

func main() {
	memo := make(Memo)
	for i := 0; i < 10; i++ {
		println(fibonacci(i, memo))
	}
}`

	sourceMap.AddFile("fibonacci.oriz", content)

	inspector := NewCodeInspector(sourceMap)

	// Test file inspection.
	inspection := inspector.InspectFile("fibonacci.oriz")

	expectedContent := []string{
		"File Inspection: fibonacci.oriz",
		"Fibonacci implementation with memoization",
		"type Memo = map[int]int",
		"func fibonacci",
		"func main",
		"for i := 0",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(inspection, expected) {
			t.Errorf("Inspection should contain '%s'", expected)
		}
	}

	// Test file statistics.
	stats := inspector.GetFileStatistics("fibonacci.oriz")

	// Verify basic statistics.
	if stats["filename"] != "fibonacci.oriz" {
		t.Errorf("Expected filename 'fibonacci.oriz', got '%v'", stats["filename"])
	}

	totalLines := stats["total_lines"].(int)
	if totalLines < 20 {
		t.Errorf("Expected at least 20 lines, got %d", totalLines)
	}

	totalChars := stats["total_chars"].(int)
	if totalChars < 300 { // Adjusted expectation
		t.Errorf("Expected at least 300 characters, got %d", totalChars)
	}

	// Should have some empty lines (from formatting).
	emptyLines := stats["empty_lines"].(int)
	if emptyLines < 1 {
		t.Errorf("Expected at least 1 empty line, got %d", emptyLines)
	}
}

// TestVisualizationPerformance tests performance of visualization tools.
func TestVisualizationPerformance(t *testing.T) {
	sourceMap := NewSourceMap()

	// Create a large source file for performance testing.
	var contentBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		contentBuilder.WriteString("func function")
		contentBuilder.WriteString("X") // Simplified for testing
		contentBuilder.WriteString("() {\n\treturn 42\n}\n")
	}

	content := contentBuilder.String()

	sourceMap.AddFile("large.oriz", content)

	// Test that visualization completes in reasonable time.
	highlighter := NewSpanHighlighter(sourceMap)

	// Create spans throughout the file.
	spans := make([]Span, 100)

	for i := 0; i < 100; i++ {
		line := (i * 3) + 1 // Every function declaration line
		spans[i] = Span{
			Start: Position{Filename: "large.oriz", Line: line, Column: 1, Offset: 0},
			End:   Position{Filename: "large.oriz", Line: line, Column: 5, Offset: 4},
		}
	}

	// This should complete without timeout.
	result := highlighter.HighlightMultipleSpans(spans)

	if !strings.Contains(result, "Multiple Span Highlighting") {
		t.Error("Performance test should produce valid output")
	}

	if !strings.Contains(result, "func") {
		t.Error("Performance test should highlight function keywords")
	}

	// Test code inspection performance.
	inspector := NewCodeInspector(sourceMap)
	stats := inspector.GetFileStatistics("large.oriz")

	if stats["total_lines"].(int) < 3000 {
		t.Error("Large file should have many lines")
	}
}

// TestErrorRecoveryWithPosition tests error recovery scenarios.
func TestErrorRecoveryWithPosition(t *testing.T) {
	sourceMap := NewSourceMap()
	content := `func problematic() {
	let x = 10 +
	let y = 20 *
	let z = x + y
	return z
}`
	sourceMap.AddFile("errors.oriz", content)

	// Simulate multiple parse errors.
	diag := NewDiagnostic()

	// First error: incomplete expression.
	diag.AddError(
		Position{Filename: "errors.oriz", Line: 2, Column: 13, Offset: 25},
		"syntax",
		"expected expression after '+'",
	)

	// Second error: incomplete expression.
	diag.AddError(
		Position{Filename: "errors.oriz", Line: 3, Column: 13, Offset: 40},
		"syntax",
		"expected expression after '*'",
	)

	// Warning: variables may be uninitialized.
	diag.AddWarning(
		Position{Filename: "errors.oriz", Line: 4, Column: 10, Offset: 55},
		"semantic",
		"variables 'x' and 'y' may not be properly initialized",
	)

	// Test that diagnostic handles multiple issues well.
	visualizer := NewErrorVisualizer(sourceMap)
	result := visualizer.VisualizeDiagnostic(diag)

	expectedElements := []string{
		"Errors (2):",
		"expected expression after '+'",
		"expected expression after '*'",
		"Warnings (1):",
		"may not be properly initialized",
		"let x = 10 +",
		"let y = 20 *",
		"let z = x + y",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(result, expected) {
			t.Errorf("Error recovery test should contain '%s'", expected)
		}
	}

	// Verify that all source lines are shown.
	lines := sourceMap.GetFiles()["errors.oriz"].Lines
	for i, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(result, strings.TrimSpace(line)) {
			t.Errorf("Line %d should be visible in error output: '%s'", i+1, line)
		}
	}
}
