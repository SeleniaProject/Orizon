package main

import (
	"log"

	"github.com/orizon-lang/orizon/internal/tools/lsp"
)

func main() {
	if err := lsp.RunStdio(); err != nil {
		log.Fatal(err)
	}
}
