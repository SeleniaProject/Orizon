package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	orifmt "github.com/orizon-lang/orizon/internal/format"
)

// orizon-fmt (minimal):
// - trims trailing spaces/tabs per line
// - ensures exactly one trailing newline
// - preserves original newline style (CRLF vs LF) when writing files
// Flags:
//
//	-w      write result to (source) file.
//	-l      list files whose formatting differs (exit 0 like gofmt).
//	-stdin  read from stdin instead of files, write formatted to stdout.
func main() {
	var (
		writeInPlace bool
		listOnly     bool
		fromStdin    bool
	)
	flag.BoolVar(&writeInPlace, "w", false, "write result to (source) file instead of stdout")
	flag.BoolVar(&listOnly, "l", false, "list files whose formatting differs from orizon-fmt output")
	flag.BoolVar(&fromStdin, "stdin", false, "read from stdin instead of files")
	flag.Parse()

	if fromStdin {
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		out := orifmt.FormatBytes(in, orifmt.Options{PreserveNewlineStyle: false})
		if _, err := os.Stdout.Write(out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
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
		// Preserve newline style on files
		out := orifmt.FormatBytes(data, orifmt.Options{PreserveNewlineStyle: true})
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

// formatBytes applies minimal formatting rules.
// If preserveNewlineStyle is true, CRLF presence in input selects CRLF in output; otherwise LF.
// keep copyStream for -stdin path performance

func copyStream(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	_, err := br.WriteTo(bw)
	return err
}
