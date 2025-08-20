package format

import (
	"bufio"
	"fmt"
	"strings"
)

// DiffMode represents the type of diff output.
type DiffMode int

const (
	DiffModeUnified    DiffMode = iota // Unified diff format (default)
	DiffModeContext                    // Context diff format
	DiffModeSideBySide                 // Side-by-side diff format
)

// DiffOptions controls diff generation.
type DiffOptions struct {
	Mode        DiffMode // Diff output format
	Context     int      // Number of context lines to show
	IgnoreSpace bool     // Ignore whitespace differences
	ShowNumbers bool     // Show line numbers
	TabWidth    int      // Tab display width
}

// DefaultDiffOptions returns default diff options.
func DefaultDiffOptions() DiffOptions {
	return DiffOptions{
		Mode:        DiffModeUnified,
		Context:     3,
		IgnoreSpace: false,
		ShowNumbers: true,
		TabWidth:    4,
	}
}

// DiffResult represents the result of a diff operation.
type DiffResult struct {
	Hunks      []Hunk
	Stats      DiffStat
	HasChanges bool
}

// Hunk represents a contiguous block of changes.
type Hunk struct {
	Header        string
	Lines         []Line
	OriginalStart int
	OriginalCount int
	ModifiedStart int
	ModifiedCount int
}

// Line represents a single line in a diff.
type Line struct {
	Content   string
	Highlight []Range
	Type      LineType
	Number    int
}

// LineType represents the type of a diff line.
type LineType int

const (
	LineTypeContext LineType = iota // Unchanged context line
	LineTypeAdded                   // Added line (+)
	LineTypeRemoved                 // Removed line (-)
)

// Range represents a range of characters to highlight.
type Range struct {
	Start int // Start position
	End   int // End position
}

// DiffStat contains statistics about changes.
type DiffStat struct {
	FilesChanged int // Number of files changed
	LinesAdded   int // Number of lines added
	LinesRemoved int // Number of lines removed
}

// DiffFormatter generates formatted diffs between source files.
type DiffFormatter struct {
	options DiffOptions
}

// NewDiffFormatter creates a new diff formatter.
func NewDiffFormatter(options DiffOptions) *DiffFormatter {
	return &DiffFormatter{options: options}
}

// GenerateDiff creates a diff between original and modified source.
func (df *DiffFormatter) GenerateDiff(filename, original, modified string) *DiffResult {
	originalLines := df.splitLines(original)
	modifiedLines := df.splitLines(modified)

	if df.options.IgnoreSpace {
		originalLines = df.normalizeWhitespace(originalLines)
		modifiedLines = df.normalizeWhitespace(modifiedLines)
	}

	// Use Myers algorithm for diff generation.
	hunks := df.generateHunks(originalLines, modifiedLines)

	result := &DiffResult{
		HasChanges: len(hunks) > 0,
		Hunks:      hunks,
		Stats:      df.calculateStats(hunks),
	}

	return result
}

// FormatDiff formats a diff result as a string.
func (df *DiffFormatter) FormatDiff(filename string, result *DiffResult) string {
	if !result.HasChanges {
		return ""
	}

	var output strings.Builder

	// Write header.
	switch df.options.Mode {
	case DiffModeUnified:
		output.WriteString(fmt.Sprintf("--- %s\t(original)\n", filename))
		output.WriteString(fmt.Sprintf("+++ %s\t(formatted)\n", filename))
	case DiffModeContext:
		output.WriteString(fmt.Sprintf("*** %s\t(original)\n", filename))
		output.WriteString(fmt.Sprintf("--- %s\t(formatted)\n", filename))
	case DiffModeSideBySide:
		output.WriteString(fmt.Sprintf("%-40s | %s\n", filename+" (original)", filename+" (formatted)"))
		output.WriteString(strings.Repeat("-", 83) + "\n")
	}

	// Write hunks.
	for _, hunk := range result.Hunks {
		df.formatHunk(&output, hunk)
	}

	return output.String()
}

// splitLines splits text into lines, preserving line endings.
func (df *DiffFormatter) splitLines(text string) []string {
	if text == "" {
		return []string{}
	}

	scanner := bufio.NewScanner(strings.NewReader(text))

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// normalizeWhitespace normalizes whitespace for comparison.
func (df *DiffFormatter) normalizeWhitespace(lines []string) []string {
	normalized := make([]string, len(lines))

	for i, line := range lines {
		// Replace tabs with spaces and trim trailing whitespace.
		expanded := strings.ReplaceAll(line, "\t", strings.Repeat(" ", df.options.TabWidth))
		normalized[i] = strings.TrimRight(expanded, " \t")
	}

	return normalized
}

// generateHunks generates diff hunks using a simplified Myers algorithm.
func (df *DiffFormatter) generateHunks(original, modified []string) []Hunk {
	changes := df.computeChanges(original, modified)
	if len(changes) == 0 {
		return []Hunk{}
	}

	var hunks []Hunk

	var currentHunk *Hunk

	context := df.options.Context

	for i, change := range changes {
		if currentHunk == nil {
			// Start new hunk.
			currentHunk = &Hunk{
				OriginalStart: max(1, change.OriginalLine-context),
				ModifiedStart: max(1, change.ModifiedLine-context),
			}

			// Add context lines before the change.
			for j := max(0, change.OriginalLine-context-1); j < change.OriginalLine-1; j++ {
				if j < len(original) {
					currentHunk.Lines = append(currentHunk.Lines, Line{
						Type:    LineTypeContext,
						Number:  j + 1,
						Content: original[j],
					})
				}
			}
		}

		// Add the change.
		switch change.Type {
		case ChangeTypeDelete:
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    LineTypeRemoved,
				Number:  change.OriginalLine,
				Content: original[change.OriginalLine-1],
			})
		case ChangeTypeInsert:
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    LineTypeAdded,
				Number:  change.ModifiedLine,
				Content: modified[change.ModifiedLine-1],
			})
		}

		// Check if we need to close this hunk.
		shouldClose := false
		if i == len(changes)-1 {
			shouldClose = true
		} else {
			nextChange := changes[i+1]

			gap := nextChange.OriginalLine - change.OriginalLine
			if gap > 2*context {
				shouldClose = true
			}
		}

		if shouldClose {
			// Add context lines after the change.
			endLine := min(len(original), change.OriginalLine+context)
			for j := change.OriginalLine; j < endLine; j++ {
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:    LineTypeContext,
					Number:  j + 1,
					Content: original[j],
				})
			}

			// Finalize hunk.
			currentHunk.OriginalCount = len(currentHunk.Lines)
			currentHunk.ModifiedCount = len(currentHunk.Lines)
			currentHunk.Header = fmt.Sprintf("@@ -%d,%d +%d,%d @@",
				currentHunk.OriginalStart, currentHunk.OriginalCount,
				currentHunk.ModifiedStart, currentHunk.ModifiedCount)

			hunks = append(hunks, *currentHunk)
			currentHunk = nil
		}
	}

	return hunks
}

// Change represents a single change in the diff.
type Change struct {
	Type         ChangeType
	OriginalLine int
	ModifiedLine int
}

// ChangeType represents the type of change.
type ChangeType int

const (
	ChangeTypeEqual ChangeType = iota
	ChangeTypeDelete
	ChangeTypeInsert
)

// computeChanges computes the changes between two slices of lines.
func (df *DiffFormatter) computeChanges(original, modified []string) []Change {
	// Simplified diff algorithm - in a real implementation, use Myers algorithm.
	var changes []Change

	i, j := 0, 0
	for i < len(original) && j < len(modified) {
		if original[i] == modified[j] {
			// Equal lines.
			i++
			j++
		} else {
			// Find the type of change.
			if j+1 < len(modified) && original[i] == modified[j+1] {
				// Insertion.
				changes = append(changes, Change{
					Type:         ChangeTypeInsert,
					OriginalLine: i + 1,
					ModifiedLine: j + 1,
				})
				j++
			} else if i+1 < len(original) && original[i+1] == modified[j] {
				// Deletion.
				changes = append(changes, Change{
					Type:         ChangeTypeDelete,
					OriginalLine: i + 1,
					ModifiedLine: j + 1,
				})
				i++
			} else {
				// Replacement (delete + insert).
				changes = append(changes, Change{
					Type:         ChangeTypeDelete,
					OriginalLine: i + 1,
					ModifiedLine: j + 1,
				})
				changes = append(changes, Change{
					Type:         ChangeTypeInsert,
					OriginalLine: i + 1,
					ModifiedLine: j + 1,
				})
				i++
				j++
			}
		}
	}

	// Handle remaining lines.
	for i < len(original) {
		changes = append(changes, Change{
			Type:         ChangeTypeDelete,
			OriginalLine: i + 1,
			ModifiedLine: len(modified) + 1,
		})
		i++
	}

	for j < len(modified) {
		changes = append(changes, Change{
			Type:         ChangeTypeInsert,
			OriginalLine: len(original) + 1,
			ModifiedLine: j + 1,
		})
		j++
	}

	return changes
}

// formatHunk formats a single hunk.
func (df *DiffFormatter) formatHunk(output *strings.Builder, hunk Hunk) {
	switch df.options.Mode {
	case DiffModeUnified:
		df.formatUnifiedHunk(output, hunk)
	case DiffModeContext:
		df.formatContextHunk(output, hunk)
	case DiffModeSideBySide:
		df.formatSideBySideHunk(output, hunk)
	}
}

// formatUnifiedHunk formats a hunk in unified diff format.
func (df *DiffFormatter) formatUnifiedHunk(output *strings.Builder, hunk Hunk) {
	output.WriteString(hunk.Header + "\n")

	for _, line := range hunk.Lines {
		var prefix string

		switch line.Type {
		case LineTypeContext:
			prefix = " "
		case LineTypeAdded:
			prefix = "+"
		case LineTypeRemoved:
			prefix = "-"
		}

		if df.options.ShowNumbers {
			output.WriteString(fmt.Sprintf("%s%4d: %s\n", prefix, line.Number, line.Content))
		} else {
			output.WriteString(fmt.Sprintf("%s%s\n", prefix, line.Content))
		}
	}
}

// formatContextHunk formats a hunk in context diff format.
func (df *DiffFormatter) formatContextHunk(output *strings.Builder, hunk Hunk) {
	output.WriteString("***************\n")
	output.WriteString(fmt.Sprintf("*** %d,%d ****\n", hunk.OriginalStart, hunk.OriginalStart+hunk.OriginalCount-1))

	// Original lines.
	for _, line := range hunk.Lines {
		if line.Type == LineTypeRemoved || line.Type == LineTypeContext {
			prefix := " "
			if line.Type == LineTypeRemoved {
				prefix = "-"
			}

			output.WriteString(fmt.Sprintf("%s %s\n", prefix, line.Content))
		}
	}

	output.WriteString(fmt.Sprintf("--- %d,%d ----\n", hunk.ModifiedStart, hunk.ModifiedStart+hunk.ModifiedCount-1))

	// Modified lines.
	for _, line := range hunk.Lines {
		if line.Type == LineTypeAdded || line.Type == LineTypeContext {
			prefix := " "
			if line.Type == LineTypeAdded {
				prefix = "+"
			}

			output.WriteString(fmt.Sprintf("%s %s\n", prefix, line.Content))
		}
	}
}

// formatSideBySideHunk formats a hunk in side-by-side format.
func (df *DiffFormatter) formatSideBySideHunk(output *strings.Builder, hunk Hunk) {
	for _, line := range hunk.Lines {
		lineNum := ""
		if df.options.ShowNumbers {
			lineNum = fmt.Sprintf("%4d: ", line.Number)
		}

		switch line.Type {
		case LineTypeContext:
			output.WriteString(fmt.Sprintf("%s%-40s | %s%-40s\n",
				lineNum, truncate(line.Content, 40-len(lineNum)),
				lineNum, truncate(line.Content, 40-len(lineNum))))
		case LineTypeRemoved:
			output.WriteString(fmt.Sprintf("%s%-40s | %s\n",
				lineNum, truncate("- "+line.Content, 40-len(lineNum)),
				strings.Repeat(" ", 41)))
		case LineTypeAdded:
			output.WriteString(fmt.Sprintf("%s%-40s | %s%-40s\n",
				strings.Repeat(" ", 41), "",
				lineNum, truncate("+ "+line.Content, 40-len(lineNum))))
		}
	}
}

// calculateStats calculates statistics for the diff.
func (df *DiffFormatter) calculateStats(hunks []Hunk) DiffStat {
	stats := DiffStat{FilesChanged: 1}

	for _, hunk := range hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case LineTypeAdded:
				stats.LinesAdded++
			case LineTypeRemoved:
				stats.LinesRemoved++
			}
		}
	}

	return stats
}

// Helper functions.
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}

// FormatWithDiff formats source and returns both formatted source and diff.
func FormatWithDiff(filename, source string, options Options, diffOptions DiffOptions) (formatted string, diff string, err error) {
	// Format using basic formatting.
	formatted = FormatText(source, options)

	// Generate diff if there are changes.
	if formatted != source {
		formatter := NewDiffFormatter(diffOptions)
		result := formatter.GenerateDiff(filename, source, formatted)
		diff = formatter.FormatDiff(filename, result)
	}

	return formatted, diff, nil
}
