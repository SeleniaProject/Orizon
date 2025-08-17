package main

import (
	"log"
	"os"

	"github.com/orizon-lang/orizon/internal/tools/lsp"
)

// orizon-lsp: LSP server entrypoint speaking stdio JSON-RPC.
func main() {
	if err := lsp.RunStdio(); err != nil {
		log.Println("orizon-lsp error:", err)
		os.Exit(1)
	}
}
