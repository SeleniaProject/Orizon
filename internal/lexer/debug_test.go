// Simple test for error recovery functionality.
package lexer

import (
	"fmt"
	"testing"
)

func TestErrorRecovery_MultipleErrors(t *testing.T) {
	input := `"unterminated 123abc invalid-id`
	lexer := NewWithFilename(input, "test.oriz").WithErrorRecovery()

	fmt.Printf("Testing input: %q\n", input)

	tokenCount := 0
	for tokenCount < 10 { // Safety limit
		token := lexer.RecoverableNextToken()
		fmt.Printf("Token %d: Type=%v, Literal=%q\n", tokenCount, token.Type, token.Literal)

		errors := lexer.GetErrors()
		fmt.Printf("  Current errors: %d\n", len(errors))

		for i, err := range errors {
			fmt.Printf("  Error %d: %s\n", i, lexer.FormatError(err))
		}

		tokenCount++

		if token.Type == TokenEOF {
			break
		}
	}
}
