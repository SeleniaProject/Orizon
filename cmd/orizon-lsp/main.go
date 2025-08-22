package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/orizon-lang/orizon/internal/cli"
	"github.com/orizon-lang/orizon/internal/tools/lsp"
)

// orizon-lsp: LSP server entrypoint speaking stdio JSON-RPC.
func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help message")
		jsonOutput  = flag.Bool("json", false, "Output version in JSON format")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Orizon Language Server Protocol (LSP) implementation.\n")
		fmt.Fprintf(os.Stderr, "Communicates via stdin/stdout using JSON-RPC protocol.\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s                 # Start LSP server\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --version       # Show version info\n", os.Args[0])
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		cli.PrintVersion("Orizon LSP Server", *jsonOutput)
		os.Exit(0)
	}

	if err := lsp.RunStdio(); err != nil {
		log.Printf("orizon-lsp error: %v", err)
		os.Exit(1)
	}
}
