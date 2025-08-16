package main

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/types"
)

func main() {
	// Test basic parsing
	parser := types.NewIndexExpressionParser("i")
	expr, err := parser.ParseIndexExpression()
	if err != nil {
		fmt.Printf("Error parsing 'i': %v\n", err)
	} else {
		fmt.Printf("Successfully parsed 'i': %T\n", expr)
	}

	// Test number parsing
	parser = types.NewIndexExpressionParser("42")
	expr, err = parser.ParseIndexExpression()
	if err != nil {
		fmt.Printf("Error parsing '42': %v\n", err)
	} else {
		fmt.Printf("Successfully parsed '42': %T\n", expr)
	}

	// Test binary expression
	parser = types.NewIndexExpressionParser("i + 1")
	expr, err = parser.ParseIndexExpression()
	if err != nil {
		fmt.Printf("Error parsing 'i + 1': %v\n", err)
	} else {
		fmt.Printf("Successfully parsed 'i + 1': %T\n", expr)
	}

	// Test len expression
	parser = types.NewIndexExpressionParser("len(arr)")
	expr, err = parser.ParseIndexExpression()
	if err != nil {
		fmt.Printf("Error parsing 'len(arr)': %v\n", err)
	} else {
		fmt.Printf("Successfully parsed 'len(arr)': %T\n", expr)
	}
}
