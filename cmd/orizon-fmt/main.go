//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	orifmt "github.com/orizon-lang/orizon/internal/format"
)

// orizon-fmt (enhanced):
// - Basic mode: trims trailing spaces/tabs per line, ensures exactly one trailing newline
// - AST mode: provides comprehensive AST-based formatting with proper indentation
// - Diff mode: shows differences between original and formatted code
// Flags:
//
//	-w      write result to (source) file.
//	-l      list files whose formatting differs (exit 0 like gofmt).
//	-stdin  read from stdin instead of files, write formatted to stdout.
//	-ast    use AST-based formatting (enhanced mode).
//	-diff   show diff output instead of formatted code.
//	-mode   diff mode: unified (default), context, side-by-side
func main() {
	var (
		writeInPlace bool
		listOnly     bool
		fromStdin    bool
		useAST       bool
		showDiff     bool
		diffMode     string
	)
	flag.BoolVar(&writeInPlace, "w", false, "write result to (source) file instead of stdout")
	flag.BoolVar(&listOnly, "l", false, "list files whose formatting differs from orizon-fmt output")
	flag.BoolVar(&fromStdin, "stdin", false, "read from stdin instead of files")
	flag.BoolVar(&useAST, "ast", false, "use AST-based formatting (enhanced mode)")
	flag.BoolVar(&showDiff, "diff", false, "show diff output instead of formatted code")
	flag.StringVar(&diffMode, "mode", "unified", "diff mode: unified, context, side-by-side")
	flag.Parse()

	if fromStdin {
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var out []byte
		if useAST {
			// Use AST-based formatting
			options := orifmt.DefaultASTFormattingOptions()
			formatted, err := orifmt.FormatSourceWithAST(string(in), options)
			if err != nil {
				fmt.Fprintln(os.Stderr, "AST formatting error:", err)
				os.Exit(1)
			}
			out = []byte(formatted)
		} else {
			// Use basic formatting
			out = orifmt.FormatBytes(in, orifmt.Options{PreserveNewlineStyle: false})
		}

		if showDiff {
			// Show diff instead of formatted output
			var diffOptions orifmt.DiffOptions
			switch diffMode {
			case "context":
				diffOptions = orifmt.DiffOptions{
					Mode:        orifmt.DiffModeContext,
					Context:     3,
					ShowNumbers: true,
					TabWidth:    4,
				}
			case "side-by-side":
				diffOptions = orifmt.DiffOptions{
					Mode:        orifmt.DiffModeSideBySide,
					Context:     3,
					ShowNumbers: true,
					TabWidth:    4,
				}
			default: // unified
				diffOptions = orifmt.DefaultDiffOptions()
			}

			diff := orifmt.NewDiffFormatter(diffOptions)
			result := diff.GenerateDiff("stdin", string(in), string(out))

			if result.HasChanges {
				diffOutput := diff.FormatDiff("stdin", result)
				fmt.Print(diffOutput)
			}
		} else {
			if _, err := os.Stdout.Write(out); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		return
	}

	// Process files provided as args
	exitCode := 0
	for _, path := range flag.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			exitCode = 1
			continue
		}

		var out []byte
		if useAST {
			// Use AST-based formatting
			options := orifmt.DefaultASTFormattingOptions()
			formatted, err := orifmt.FormatSourceWithAST(string(data), options)
			if err != nil {
				fmt.Fprintln(os.Stderr, "AST formatting error for", path+":", err)
				exitCode = 1
				continue
			}
			out = []byte(formatted)
		} else {
			// Use basic formatting - preserve newline style on files
			out = orifmt.FormatBytes(data, orifmt.Options{PreserveNewlineStyle: true})
		}

		if showDiff {
			// Show diff output
			var diffOptions orifmt.DiffOptions
			switch diffMode {
			case "context":
				diffOptions = orifmt.DiffOptions{
					Mode:        orifmt.DiffModeContext,
					Context:     3,
					ShowNumbers: true,
					TabWidth:    4,
				}
			case "side-by-side":
				diffOptions = orifmt.DiffOptions{
					Mode:        orifmt.DiffModeSideBySide,
					Context:     3,
					ShowNumbers: true,
					TabWidth:    4,
				}
			default: // unified
				diffOptions = orifmt.DefaultDiffOptions()
			}

			diff := orifmt.NewDiffFormatter(diffOptions)
			basename := filepath.Base(path)
			result := diff.GenerateDiff(basename, string(data), string(out))

			if result.HasChanges {
				diffOutput := diff.FormatDiff(basename, result)
				fmt.Print(diffOutput)
			}
			continue
		}

		if listOnly {
			if !bytes.Equal(out, data) {
				fmt.Fprintln(os.Stdout, path)
			}
			continue
		}
		if writeInPlace {
			if !bytes.Equal(out, data) {
				if err := os.WriteFile(path, out, 0o666); err != nil {
					fmt.Fprintln(os.Stderr, err)
					exitCode = 1
				}
			}
			continue
		}
		// Default: print formatted content to stdout
		if _, err := os.Stdout.Write(out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}
