package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	orifmt "github.com/orizon-lang/orizon/internal/format"
)

// orizon-fmt (enhanced):
// - Basic formatting: trims trailing spaces/tabs per line, ensures exactly one trailing newline
// - AST-based formatting: comprehensive code formatting using parsed AST
// - Diff mode: shows differences between original and formatted code
// - Preserves original newline style (CRLF vs LF) when writing files
//
// Flags:
//   -w         write result to (source) file.
//   -l         list files whose formatting differs (exit 0 like gofmt).
//   -d         display diffs instead of rewriting files.
//   -stdin     read from stdin instead of files, write formatted to stdout.
//   -ast       use AST-based formatting (experimental).
//   -indent    indentation size for AST formatting (default 4).
//   -tabs      use tabs instead of spaces for indentation.
//   -maxline   maximum line length for AST formatting (default 100).

func main() {
	var (
		writeInPlace  bool
		listOnly      bool
		showDiff      bool
		fromStdin     bool
		useAST        bool
		indentSize    int
		useTabs       bool
		maxLineLength int
		contextLines  int
	)

	flag.BoolVar(&writeInPlace, "w", false, "write result to (source) file instead of stdout")
	flag.BoolVar(&listOnly, "l", false, "list files whose formatting differs from orizon-fmt output")
	flag.BoolVar(&showDiff, "d", false, "display diffs instead of rewriting files")
	flag.BoolVar(&fromStdin, "stdin", false, "read from stdin instead of files")
	flag.BoolVar(&useAST, "ast", false, "use AST-based formatting (experimental)")
	flag.IntVar(&indentSize, "indent", 4, "indentation size for AST formatting")
	flag.BoolVar(&useTabs, "tabs", false, "use tabs instead of spaces for indentation")
	flag.IntVar(&maxLineLength, "maxline", 100, "maximum line length for AST formatting")
	flag.IntVar(&contextLines, "context", 3, "number of context lines in diff output")

	flag.Parse()

	// Create formatting options
	basicOptions := orifmt.Options{
		PreserveNewlineStyle: !fromStdin, // Don't preserve for stdin
	}

	astOptions := orifmt.ASTFormattingOptions{
		IndentSize:                   indentSize,
		PreferTabs:                   useTabs,
		MaxLineLength:                maxLineLength,
		AlignFields:                  true,
		SpaceAroundOperators:         true,
		TrailingComma:                true,
		EmptyLineBetweenDeclarations: true,
	}

	diffOptions := orifmt.DiffOptions{
		Mode:        orifmt.DiffModeUnified,
		Context:     contextLines,
		IgnoreSpace: false,
		ShowNumbers: true,
		TabWidth:    4,
	}

	if fromStdin {
		handleStdin(basicOptions, astOptions, useAST)
		return
	}

	// Process files provided as args
	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No input files specified")
		os.Exit(1)
	}

	exitCode := 0
	for _, path := range files {
		if err := processFile(path, basicOptions, astOptions, diffOptions,
			writeInPlace, listOnly, showDiff, useAST); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", path, err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

// handleStdin processes input from stdin
func handleStdin(basicOptions orifmt.Options, astOptions orifmt.ASTFormattingOptions, useAST bool) {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
		os.Exit(1)
	}

	var output []byte

	if useAST {
		formatted, err := orifmt.FormatSourceWithAST(string(input), astOptions)
		if err != nil {
			// Fallback to basic formatting on AST parsing errors
			output = orifmt.FormatBytes(input, basicOptions)
		} else {
			output = []byte(formatted)
		}
	} else {
		output = orifmt.FormatBytes(input, basicOptions)
	}

	if _, err := os.Stdout.Write(output); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing output:", err)
		os.Exit(1)
	}
}

// processFile processes a single file
func processFile(path string, basicOptions orifmt.Options, astOptions orifmt.ASTFormattingOptions,
	diffOptions orifmt.DiffOptions, writeInPlace, listOnly, showDiff, useAST bool) error {

	// Check if file is a valid Orizon source file
	if !isOrizonFile(path) {
		return fmt.Errorf("not an Orizon source file: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	original := string(data)
	var formatted string

	if useAST {
		// Try AST-based formatting first
		astFormatted, err := orifmt.FormatSourceWithAST(original, astOptions)
		if err != nil {
			// Fallback to basic formatting on AST parsing errors
			basicFormatted := orifmt.FormatBytes(data, basicOptions)
			formatted = string(basicFormatted)

			if !listOnly && !showDiff {
				fmt.Fprintf(os.Stderr, "Warning: AST parsing failed for %s, using basic formatting: %v\n", path, err)
			}
		} else {
			formatted = astFormatted
		}
	} else {
		// Use basic formatting
		basicFormatted := orifmt.FormatBytes(data, basicOptions)
		formatted = string(basicFormatted)
	}

	// Check if there are changes
	hasChanges := formatted != original

	if listOnly {
		if hasChanges {
			fmt.Println(path)
		}
		return nil
	}

	if showDiff {
		if hasChanges {
			diffFormatter := orifmt.NewDiffFormatter(diffOptions)
			result := diffFormatter.GenerateDiff(path, original, formatted)
			diff := diffFormatter.FormatDiff(path, result)
			fmt.Print(diff)
		}
		return nil
	}

	if writeInPlace {
		if hasChanges {
			return os.WriteFile(path, []byte(formatted), 0666)
		}
		return nil
	}

	// Default: print formatted content to stdout
	fmt.Print(formatted)
	return nil
}

// isOrizonFile checks if a file is a valid Orizon source file
func isOrizonFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".oriz" || ext == ".orizon"
}

// Additional utility functions for advanced formatting

// formatDirectory recursively formats all Orizon files in a directory
func formatDirectory(dir string, options orifmt.Options, recursive bool) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !recursive && path != dir {
				return filepath.SkipDir
			}
			return nil
		}

		if isOrizonFile(path) {
			// Process file...
			return nil
		}

		return nil
	})
}

// validateSyntax checks if the formatted code is syntactically valid
func validateSyntax(source string) error {
	// This would integrate with the parser to validate syntax
	// For now, just return nil
	return nil
}

// formatWithBackup creates a backup of the original file before formatting
func formatWithBackup(path string, formatted []byte) error {
	backupPath := path + ".backup"

	// Read original file
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Create backup
	if err := os.WriteFile(backupPath, original, 0666); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Write formatted content
	if err := os.WriteFile(path, formatted, 0666); err != nil {
		// Restore from backup on failure
		os.WriteFile(path, original, 0666)
		os.Remove(backupPath)
		return fmt.Errorf("failed to write formatted file: %v", err)
	}

	return nil
}

// Statistics tracking for batch operations
type FormatStats struct {
	FilesProcessed int
	FilesChanged   int
	LinesAdded     int
	LinesRemoved   int
	Errors         int
}

// String returns a formatted string representation of the stats
func (fs FormatStats) String() string {
	return fmt.Sprintf("Processed: %d files, Changed: %d files, +%d -%d lines, Errors: %d",
		fs.FilesProcessed, fs.FilesChanged, fs.LinesAdded, fs.LinesRemoved, fs.Errors)
}

// copyStream keeps the stream copying functionality for backward compatibility
func copyStream(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	_, err := br.WriteTo(bw)
	return err
}
