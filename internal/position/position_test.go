package position

import (
	"testing"
)

func TestPosition(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		pos      Position
		isValid  bool
	}{
		{
			name: "Valid position with filename",
			pos: Position{
				Filename: "test.oriz",
				Line:     10,
				Column:   5,
				Offset:   100,
			},
			isValid:  true,
			expected: "test.oriz:10:5",
		},
		{
			name: "Valid position without filename",
			pos: Position{
				Line:   1,
				Column: 1,
				Offset: 0,
			},
			isValid:  true,
			expected: "1:1",
		},
		{
			name: "Invalid position - zero line",
			pos: Position{
				Line:   0,
				Column: 1,
				Offset: 0,
			},
			isValid: false,
		},
		{
			name: "Invalid position - zero column",
			pos: Position{
				Line:   1,
				Column: 0,
				Offset: 0,
			},
			isValid: false,
		},
		{
			name: "Invalid position - negative offset",
			pos: Position{
				Line:   1,
				Column: 1,
				Offset: -1,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pos.IsValid(); got != tt.isValid {
				t.Errorf("Position.IsValid() = %v, want %v", got, tt.isValid)
			}

			if tt.isValid {
				if got := tt.pos.String(); got != tt.expected {
					t.Errorf("Position.String() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestPositionComparison(t *testing.T) {
	pos1 := Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4}
	pos2 := Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9}
	pos3 := Position{Filename: "other.oriz", Line: 1, Column: 1, Offset: 0}

	if !pos1.Before(pos2) {
		t.Error("pos1 should be before pos2")
	}

	if !pos2.After(pos1) {
		t.Error("pos2 should be after pos1")
	}

	if !pos3.Before(pos1) {
		t.Error("pos3 should be before pos1 (different filename)")
	}
}

func TestSpan(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		span     Span
		length   int
		isValid  bool
	}{
		{
			name: "Valid span same line",
			span: Span{
				Start: Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
				End:   Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
			},
			isValid:  true,
			expected: "test.oriz:1:5-10",
			length:   5,
		},
		{
			name: "Valid span multiple lines",
			span: Span{
				Start: Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
				End:   Position{Filename: "test.oriz", Line: 3, Column: 2, Offset: 20},
			},
			isValid:  true,
			expected: "test.oriz:1:5-3:2",
			length:   16,
		},
		{
			name: "Invalid span - different files",
			span: Span{
				Start: Position{Filename: "test1.oriz", Line: 1, Column: 1, Offset: 0},
				End:   Position{Filename: "test2.oriz", Line: 1, Column: 5, Offset: 4},
			},
			isValid: false,
		},
		{
			name: "Invalid span - end before start",
			span: Span{
				Start: Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
				End:   Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.span.IsValid(); got != tt.isValid {
				t.Errorf("Span.IsValid() = %v, want %v", got, tt.isValid)
			}

			if tt.isValid {
				if got := tt.span.String(); got != tt.expected {
					t.Errorf("Span.String() = %v, want %v", got, tt.expected)
				}

				if got := tt.span.Length(); got != tt.length {
					t.Errorf("Span.Length() = %v, want %v", got, tt.length)
				}
			}
		})
	}
}

func TestSpanContains(t *testing.T) {
	span := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
	}

	tests := []struct {
		name     string
		pos      Position
		contains bool
	}{
		{
			name:     "Position at start",
			pos:      Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
			contains: true,
		},
		{
			name:     "Position in middle",
			pos:      Position{Filename: "test.oriz", Line: 1, Column: 7, Offset: 6},
			contains: true,
		},
		{
			name:     "Position at end (exclusive)",
			pos:      Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
			contains: false,
		},
		{
			name:     "Position before span",
			pos:      Position{Filename: "test.oriz", Line: 1, Column: 1, Offset: 0},
			contains: false,
		},
		{
			name:     "Position after span",
			pos:      Position{Filename: "test.oriz", Line: 1, Column: 15, Offset: 14},
			contains: false,
		},
		{
			name:     "Position in different file",
			pos:      Position{Filename: "other.oriz", Line: 1, Column: 7, Offset: 6},
			contains: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := span.Contains(tt.pos); got != tt.contains {
				t.Errorf("Span.Contains() = %v, want %v", got, tt.contains)
			}
		})
	}
}

func TestSpanOverlaps(t *testing.T) {
	span1 := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
	}

	tests := []struct {
		name     string
		span2    Span
		overlaps bool
	}{
		{
			name: "Overlapping spans",
			span2: Span{
				Start: Position{Filename: "test.oriz", Line: 1, Column: 8, Offset: 7},
				End:   Position{Filename: "test.oriz", Line: 1, Column: 15, Offset: 14},
			},
			overlaps: true,
		},
		{
			name: "Adjacent spans (no overlap)",
			span2: Span{
				Start: Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
				End:   Position{Filename: "test.oriz", Line: 1, Column: 15, Offset: 14},
			},
			overlaps: false,
		},
		{
			name: "Separate spans",
			span2: Span{
				Start: Position{Filename: "test.oriz", Line: 1, Column: 20, Offset: 19},
				End:   Position{Filename: "test.oriz", Line: 1, Column: 25, Offset: 24},
			},
			overlaps: false,
		},
		{
			name: "Spans in different files",
			span2: Span{
				Start: Position{Filename: "other.oriz", Line: 1, Column: 5, Offset: 4},
				End:   Position{Filename: "other.oriz", Line: 1, Column: 10, Offset: 9},
			},
			overlaps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := span1.Overlaps(tt.span2); got != tt.overlaps {
				t.Errorf("Span.Overlaps() = %v, want %v", got, tt.overlaps)
			}
		})
	}
}

func TestSpanUnion(t *testing.T) {
	span1 := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 10, Offset: 9},
	}

	span2 := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 8, Offset: 7},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 15, Offset: 14},
	}

	union := span1.Union(span2)
	expected := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 15, Offset: 14},
	}

	if union != expected {
		t.Errorf("Span.Union() = %v, want %v", union, expected)
	}
}

func TestSourceFile(t *testing.T) {
	content := "func main() {\n\tprintln(\"Hello, World!\")\n}"
	file := NewSourceFile("test.oriz", content)

	if file.Filename != "test.oriz" {
		t.Errorf("SourceFile.Filename = %v, want %v", file.Filename, "test.oriz")
	}

	if file.Content != content {
		t.Errorf("SourceFile.Content = %v, want %v", file.Content, content)
	}

	expectedLines := []string{
		"func main() {",
		"\tprintln(\"Hello, World!\")",
		"}",
	}

	if len(file.Lines) != len(expectedLines) {
		t.Errorf("SourceFile.Lines length = %v, want %v", len(file.Lines), len(expectedLines))
	}

	for i, line := range expectedLines {
		if file.GetLine(i+1) != line {
			t.Errorf("SourceFile.GetLine(%d) = %v, want %v", i+1, file.GetLine(i+1), line)
		}
	}
}

func TestSourceFilePositionConversion(t *testing.T) {
	content := "func main() {\n\tprintln(\"Hello\")\n}"
	file := NewSourceFile("test.oriz", content)

	tests := []struct {
		name     string
		expected Position
		offset   int
	}{
		{
			name:   "Start of file",
			offset: 0,
			expected: Position{
				Filename: "test.oriz",
				Line:     1,
				Column:   1,
				Offset:   0,
			},
		},
		{
			name:   "Start of second line",
			offset: 14, // After "func main() {\n"
			expected: Position{
				Filename: "test.oriz",
				Line:     2,
				Column:   1,
				Offset:   14,
			},
		},
		{
			name:   "Middle of second line",
			offset: 16, // At 'p' in "println"
			expected: Position{
				Filename: "test.oriz",
				Line:     2,
				Column:   3,
				Offset:   16,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := file.PositionFromOffset(tt.offset)
			if pos != tt.expected {
				t.Errorf("PositionFromOffset(%d) = %v, want %v", tt.offset, pos, tt.expected)
			}

			// Test round trip conversion.
			offset := file.OffsetFromPosition(pos)
			if offset != tt.offset {
				t.Errorf("OffsetFromPosition(%v) = %d, want %d", pos, offset, tt.offset)
			}
		})
	}
}

func TestSourceFileGetSpanText(t *testing.T) {
	content := "func main() {\n\tprintln(\"Hello\")\n}"
	file := NewSourceFile("test.oriz", content)

	span := Span{
		Start: Position{Filename: "test.oriz", Line: 1, Column: 1, Offset: 0},
		End:   Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4},
	}

	text := file.GetSpanText(span)
	expected := "func"

	if text != expected {
		t.Errorf("GetSpanText() = %v, want %v", text, expected)
	}
}

func TestSourceMap(t *testing.T) {
	sm := NewSourceMap()

	content1 := "func main() {}"
	content2 := "type User struct {}"

	file1 := sm.AddFile("main.oriz", content1)
	file2 := sm.AddFile("user.oriz", content2)

	// Test file retrieval.
	if got := sm.GetFile("main.oriz"); got != file1 {
		t.Error("GetFile returned wrong file")
	}

	if got := sm.GetFile("user.oriz"); got != file2 {
		t.Error("GetFile returned wrong file")
	}

	// Test span text retrieval.
	span := Span{
		Start: Position{Filename: "main.oriz", Line: 1, Column: 1, Offset: 0},
		End:   Position{Filename: "main.oriz", Line: 1, Column: 5, Offset: 4},
	}

	text := sm.GetSpanText(span)
	expected := "func"

	if text != expected {
		t.Errorf("GetSpanText() = %v, want %v", text, expected)
	}

	// Test line retrieval.
	pos := Position{Filename: "user.oriz", Line: 1, Column: 1, Offset: 0}
	line := sm.GetLine(pos)
	expected = "type User struct {}"

	if line != expected {
		t.Errorf("GetLine() = %v, want %v", line, expected)
	}

	// Test file listing.
	files := sm.GetFiles()
	if len(files) != 2 {
		t.Errorf("GetFiles() returned %d files, want 2", len(files))
	}
}

func TestDiagnostic(t *testing.T) {
	diag := NewDiagnostic()

	pos1 := Position{Filename: "test.oriz", Line: 1, Column: 5, Offset: 4}
	pos2 := Position{Filename: "test.oriz", Line: 2, Column: 10, Offset: 20}

	// Test initial state.
	if diag.HasErrors() {
		t.Error("New diagnostic should not have errors")
	}

	if diag.HasWarnings() {
		t.Error("New diagnostic should not have warnings")
	}

	// Add error.
	diag.AddError(pos1, "syntax", "unexpected token")

	if !diag.HasErrors() {
		t.Error("Diagnostic should have errors after adding one")
	}

	if diag.ErrorCount() != 1 {
		t.Errorf("ErrorCount() = %d, want 1", diag.ErrorCount())
	}

	// Add warning.
	diag.AddWarning(pos2, "unused", "variable not used")

	if !diag.HasWarnings() {
		t.Error("Diagnostic should have warnings after adding one")
	}

	if diag.WarningCount() != 1 {
		t.Errorf("WarningCount() = %d, want 1", diag.WarningCount())
	}

	// Test error formatting.
	error := diag.Errors[0]
	expected := "test.oriz:1:5: syntax: unexpected token"

	if error.String() != expected {
		t.Errorf("Error.String() = %v, want %v", error.String(), expected)
	}

	// Test warning formatting.
	warning := diag.Warnings[0]
	expected = "test.oriz:2:10: warning: unused: variable not used"

	if warning.String() != expected {
		t.Errorf("Warning.String() = %v, want %v", warning.String(), expected)
	}

	// Test clear.
	diag.Clear()

	if diag.HasErrors() || diag.HasWarnings() {
		t.Error("Diagnostic should be clear after Clear()")
	}

	if diag.ErrorCount() != 0 || diag.WarningCount() != 0 {
		t.Error("Counts should be zero after Clear()")
	}
}

func TestInvalidPositions(t *testing.T) {
	// Test invalid position handling.
	invalidPos := Position{Line: 0, Column: 1, Offset: 0}
	if invalidPos.IsValid() {
		t.Error("Invalid position should not be valid")
	}

	// Test invalid span handling.
	invalidSpan := Span{
		Start: invalidPos,
		End:   Position{Line: 1, Column: 1, Offset: 0},
	}
	if invalidSpan.IsValid() {
		t.Error("Invalid span should not be valid")
	}

	// Test operations on invalid spans.
	if invalidSpan.Length() != 0 {
		t.Error("Invalid span length should be 0")
	}

	validPos := Position{Line: 1, Column: 1, Offset: 0}
	if invalidSpan.Contains(validPos) {
		t.Error("Invalid span should not contain any position")
	}
}
