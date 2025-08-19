package main

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/parser"
)

func main() {
	src := "impl<T> S where T: Eq { func id(x: int) -> int { return x; } }"
	fmt.Println("Source:", src)
	fmt.Println("Analyzing token sequence...")

	l := lexer.New(src)
	for {
		tok := l.NextToken()
		fmt.Printf("Token: %s = '%s'\n", tok.Type, tok.Literal)
		if tok.Type == lexer.TokenEOF {
			break
		}
	}

	fmt.Println("\nParsing...")
	l = lexer.New(src)
	p := parser.NewParser(l, "test.oriz")
	prog, errs := p.Parse()

	if len(errs) > 0 {
		fmt.Println("Parse errors:")
		for _, err := range errs {
			fmt.Println(" -", err)
		}
	} else {
		fmt.Println("Parse successful!")
		fmt.Printf("Declarations: %d\n", len(prog.Declarations))
	}
}
