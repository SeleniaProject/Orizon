// Package position provides unified source code position tracking
// for the Orizon compiler. This system enables precise error reporting,
// debugging support, and source map generation.
package position

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Position represents a single point in source code
type Position struct {
	Filename string // Source file name
	Line     int    // 1-based line number
	Column   int    // 1-based column number
	Offset   int    // 0-based byte offset in source
}

// IsValid returns true if the position is valid
func (p Position) IsValid() bool {
	return p.Line > 0 && p.Column > 0 && p.Offset >= 0
}

// String returns a string representation of the position
func (p Position) String() string {
	if p.Filename != "" {
		return fmt.Sprintf("%s:%d:%d", filepath.Base(p.Filename), p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Before returns true if this position comes before other
func (p Position) Before(other Position) bool {
	if p.Filename != other.Filename {
		return p.Filename < other.Filename
	}
	return p.Offset < other.Offset
}

// After returns true if this position comes after other
func (p Position) After(other Position) bool {
	if p.Filename != other.Filename {
		return p.Filename > other.Filename
	}
	return p.Offset > other.Offset
}

// Span represents a range of source code between two positions
type Span struct {
	Start Position // Starting position (inclusive)
	End   Position // Ending position (exclusive)
}

// IsValid returns true if the span is valid
func (s Span) IsValid() bool {
	return s.Start.IsValid() && s.End.IsValid() &&
		s.Start.Filename == s.End.Filename &&
		s.Start.Offset <= s.End.Offset
}

// String returns a string representation of the span
func (s Span) String() string {
	if s.Start.Filename != "" {
		filename := filepath.Base(s.Start.Filename)
		if s.Start.Line == s.End.Line {
			return fmt.Sprintf("%s:%d:%d-%d", filename, s.Start.Line, s.Start.Column, s.End.Column)
		}
		return fmt.Sprintf("%s:%d:%d-%d:%d", filename, s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
	}

	if s.Start.Line == s.End.Line {
		return fmt.Sprintf("%d:%d-%d", s.Start.Line, s.Start.Column, s.End.Column)
	}
	return fmt.Sprintf("%d:%d-%d:%d", s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
}

// Contains returns true if the span contains the given position
func (s Span) Contains(pos Position) bool {
	if !s.IsValid() || !pos.IsValid() {
		return false
	}
	if s.Start.Filename != pos.Filename {
		return false
	}
	return s.Start.Offset <= pos.Offset && pos.Offset < s.End.Offset
}

// Overlaps returns true if this span overlaps with other
func (s Span) Overlaps(other Span) bool {
	if !s.IsValid() || !other.IsValid() {
		return false
	}
	if s.Start.Filename != other.Start.Filename {
		return false
	}
	return s.Start.Offset < other.End.Offset && other.Start.Offset < s.End.Offset
}

// Union returns a span that encompasses both this span and other
func (s Span) Union(other Span) Span {
	if !s.IsValid() {
		return other
	}
	if !other.IsValid() {
		return s
	}
	if s.Start.Filename != other.Start.Filename {
		return s // Cannot union spans from different files
	}

	start := s.Start
	if other.Start.Before(start) {
		start = other.Start
	}

	end := s.End
	if other.End.After(end) {
		end = other.End
	}

	return Span{Start: start, End: end}
}

// Length returns the length of the span in bytes
func (s Span) Length() int {
	if !s.IsValid() {
		return 0
	}
	return s.End.Offset - s.Start.Offset
}

// SourceFile represents a source file with content and position tracking
type SourceFile struct {
	Filename string   // File path
	Content  string   // Source code content
	Lines    []string // Lines of source code for efficient access
}

// NewSourceFile creates a new source file from content
func NewSourceFile(filename, content string) *SourceFile {
	lines := strings.Split(content, "\n")
	return &SourceFile{
		Filename: filename,
		Content:  content,
		Lines:    lines,
	}
}

// GetLine returns the specified line (1-based) or empty string if invalid
func (sf *SourceFile) GetLine(lineNum int) string {
	if lineNum < 1 || lineNum > len(sf.Lines) {
		return ""
	}
	return sf.Lines[lineNum-1]
}

// GetSpanText returns the text covered by the span
func (sf *SourceFile) GetSpanText(span Span) string {
	if !span.IsValid() || span.Start.Filename != sf.Filename {
		return ""
	}

	if span.Start.Offset >= len(sf.Content) || span.End.Offset > len(sf.Content) {
		return ""
	}

	return sf.Content[span.Start.Offset:span.End.Offset]
}

// PositionFromOffset converts a byte offset to a Position
func (sf *SourceFile) PositionFromOffset(offset int) Position {
	if offset < 0 || offset > len(sf.Content) {
		return Position{}
	}

	line := 1
	column := 1

	for i := 0; i < offset; i++ {
		if sf.Content[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}

	return Position{
		Filename: sf.Filename,
		Line:     line,
		Column:   column,
		Offset:   offset,
	}
}

// OffsetFromPosition converts a Position to a byte offset
func (sf *SourceFile) OffsetFromPosition(pos Position) int {
	if pos.Line < 1 || pos.Column < 1 {
		return -1
	}

	offset := 0
	currentLine := 1

	for i := 0; i < len(sf.Content) && currentLine < pos.Line; i++ {
		if sf.Content[i] == '\n' {
			currentLine++
		}
		offset++
	}

	// Add column offset (accounting for 1-based indexing)
	offset += pos.Column - 1

	if offset > len(sf.Content) {
		return -1
	}

	return offset
}

// SourceMap manages multiple source files and provides unified position tracking
type SourceMap struct {
	files map[string]*SourceFile // filename -> SourceFile
}

// NewSourceMap creates a new source map
func NewSourceMap() *SourceMap {
	return &SourceMap{
		files: make(map[string]*SourceFile),
	}
}

// AddFile adds a source file to the map
func (sm *SourceMap) AddFile(filename, content string) *SourceFile {
	file := NewSourceFile(filename, content)
	sm.files[filename] = file
	return file
}

// GetFile returns the source file for the given filename
func (sm *SourceMap) GetFile(filename string) *SourceFile {
	return sm.files[filename]
}

// GetSpanText returns the text covered by the span across all files
func (sm *SourceMap) GetSpanText(span Span) string {
	file := sm.GetFile(span.Start.Filename)
	if file == nil {
		return ""
	}
	return file.GetSpanText(span)
}

// GetLine returns the specified line from the appropriate file
func (sm *SourceMap) GetLine(pos Position) string {
	file := sm.GetFile(pos.Filename)
	if file == nil {
		return ""
	}
	return file.GetLine(pos.Line)
}

// GetFiles returns all registered files
func (sm *SourceMap) GetFiles() map[string]*SourceFile {
	result := make(map[string]*SourceFile)
	for k, v := range sm.files {
		result[k] = v
	}
	return result
}

// Error represents a compiler error with position information
type Error struct {
	Pos     Position // Position where the error occurred
	Message string   // Error message
	Kind    string   // Error kind (e.g., "syntax", "type", "semantic")
}

// String returns a formatted error message
func (e Error) String() string {
	return fmt.Sprintf("%s: %s: %s", e.Pos.String(), e.Kind, e.Message)
}

// Warning represents a compiler warning with position information
type Warning struct {
	Pos     Position // Position where the warning occurred
	Message string   // Warning message
	Kind    string   // Warning kind
}

// String returns a formatted warning message
func (w Warning) String() string {
	return fmt.Sprintf("%s: warning: %s: %s", w.Pos.String(), w.Kind, w.Message)
}

// Diagnostic represents a collection of errors and warnings
type Diagnostic struct {
	Errors   []Error   // List of errors
	Warnings []Warning // List of warnings
}

// NewDiagnostic creates a new diagnostic collection
func NewDiagnostic() *Diagnostic {
	return &Diagnostic{
		Errors:   make([]Error, 0),
		Warnings: make([]Warning, 0),
	}
}

// AddError adds an error to the diagnostic
func (d *Diagnostic) AddError(pos Position, kind, message string) {
	d.Errors = append(d.Errors, Error{
		Pos:     pos,
		Message: message,
		Kind:    kind,
	})
}

// AddWarning adds a warning to the diagnostic
func (d *Diagnostic) AddWarning(pos Position, kind, message string) {
	d.Warnings = append(d.Warnings, Warning{
		Pos:     pos,
		Message: message,
		Kind:    kind,
	})
}

// HasErrors returns true if there are any errors
func (d *Diagnostic) HasErrors() bool {
	return len(d.Errors) > 0
}

// HasWarnings returns true if there are any warnings
func (d *Diagnostic) HasWarnings() bool {
	return len(d.Warnings) > 0
}

// ErrorCount returns the number of errors
func (d *Diagnostic) ErrorCount() int {
	return len(d.Errors)
}

// WarningCount returns the number of warnings
func (d *Diagnostic) WarningCount() int {
	return len(d.Warnings)
}

// Clear removes all errors and warnings
func (d *Diagnostic) Clear() {
	d.Errors = d.Errors[:0]
	d.Warnings = d.Warnings[:0]
}
