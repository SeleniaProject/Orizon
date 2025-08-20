package format

import (
	"bytes"
	"strings"
)

// Options controls formatting style.
type Options struct {
	// PreserveNewlineStyle: when true, CRLF in input keeps CRLF in output; else LF.
	PreserveNewlineStyle bool
}

// DefaultOptions returns sane defaults.
func DefaultOptions() Options {
	return Options{PreserveNewlineStyle: true}
}

// FormatBytes formats source bytes and returns formatted bytes.
func FormatBytes(in []byte, opts Options) []byte {
	return []byte(FormatText(string(in), opts))
}

// FormatText applies minimal, safe formatting:.
// - trims trailing spaces/tabs on each line
// - ensures exactly one trailing newline.
// - preserves CRLF vs LF depending on options and input.
func FormatText(text string, opts Options) string {
	// Decide newline style.
	useCRLF := opts.PreserveNewlineStyle && strings.Contains(text, "\r\n")

	// Normalize to \n for processing.
	norm := strings.ReplaceAll(text, "\r\n", "\n")
	norm = strings.ReplaceAll(norm, "\r", "\n")

	if norm == "" {
		if useCRLF {
			return "\r\n"
		}

		return "\n"
	}

	lines := strings.Split(norm, "\n")
	// Drop final empty due to trailing newline; we'll re-add exactly one later.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	sep := "\n"
	if useCRLF {
		sep = "\r\n"
	}

	var buf bytes.Buffer

	for i, ln := range lines {
		if i > 0 {
			buf.WriteString(sep)
		}

		buf.WriteString(ln)
	}

	buf.WriteString(sep)

	return buf.String()
}
