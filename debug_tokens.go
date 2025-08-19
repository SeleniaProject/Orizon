package main

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func main() {
	src := "T: Eq {"
	fmt.Println("Source:", src)

	l := lexer.New(src)
	current := l.NextToken() // T
	peek := l.NextToken()    // :

	fmt.Printf("After T: current='%s' peek='%s'\n", current.Literal, peek.Literal)

	// Simulate parseType() consuming T
	current = peek       // T -> :
	peek = l.NextToken() // : -> Eq

	fmt.Printf("After parseType(): current='%s' peek='%s'\n", current.Literal, peek.Literal)

	// expectPeek(TokenColon)
	if peek.Type == lexer.TokenColon {
		current = peek       // : -> Eq
		peek = l.NextToken() // Eq -> {
		fmt.Printf("After expectPeek(COLON): current='%s' peek='%s'\n", current.Literal, peek.Literal)
	}

	// nextToken() before parseTraitBounds()
	current = peek       // Eq -> {
	peek = l.NextToken() // { -> EOF

	fmt.Printf("After nextToken(): current='%s' peek='%s'\n", current.Literal, peek.Literal)

	// parseTraitBounds() calls parseType() which would consume Eq
	// but Eq is already current, so parseType might not advance properly
}
