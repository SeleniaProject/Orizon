package types

import (
	"fmt"
	"testing"
)

func TestSimpleIndexParser(t *testing.T) {
	// Debug the parser step by step.
	t.Run("SingleCharacter", func(t *testing.T) {
		parser := NewIndexExpressionParser("i")
		fmt.Printf("Input: %q\n", parser.input)
		fmt.Printf("Initial position: %d\n", parser.position)
		fmt.Printf("Initial current: %c (%d)\n", parser.current, parser.current)

		expr, err := parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		if varExpr, ok := expr.(*VariableIndexExpr); !ok {
			t.Errorf("Expected VariableIndexExpr, got %T", expr)
		} else {
			fmt.Printf("Parsed variable: %s\n", varExpr.Name)
		}
	})

	t.Run("TwoDigitNumber", func(t *testing.T) {
		parser := NewIndexExpressionParser("42")
		fmt.Printf("Input: %q\n", parser.input)
		fmt.Printf("Initial position: %d\n", parser.position)
		fmt.Printf("Initial current: %c (%d)\n", parser.current, parser.current)

		expr, err := parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		if constExpr, ok := expr.(*ConstantIndexExpr); !ok {
			t.Errorf("Expected ConstantIndexExpr, got %T", expr)
		} else {
			fmt.Printf("Parsed constant: %d\n", constExpr.Value)
		}
	})
}
