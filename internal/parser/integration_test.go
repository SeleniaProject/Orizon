// Package parser integration test.
// Phase 1.2.1: パーサー統合テスト
package parser

import (
	"os"
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// TestParserIntegration tests parser with real code files.
func TestParserIntegration(t *testing.T) {
	// Test with example files.
	testFiles := []string{
		"../../examples/hello.oriz",
		"../../examples/simple.oriz",
	}

	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Skipf("Test file %s does not exist", file)

				return
			}

			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", file, err)
			}

			l := lexer.New(string(content))
			p := NewParser(l, file)

			program, errors := p.Parse()

			if len(errors) > 0 {
				t.Logf("Parsing errors for %s: %v", file, errors)
			}

			if program == nil {
				t.Fatalf("Failed to parse %s", file)
			}

			t.Logf("Successfully parsed %s with %d declarations", file, len(program.Declarations))

			// Test pretty printing.
			output := PrettyPrint(program)
			if output == "" {
				t.Error("Pretty print produced empty output")
			}
		})
	}
}

// TestComplexExpressions tests complex expression parsing.
func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Operator precedence",
			input: "1 + 2 * 3 + 4;",
		},
		{
			name:  "Function calls with nested expressions",
			input: "foo(bar(1 + 2), baz(3 * 4));",
		},
		{
			name:  "Mixed operators",
			input: "a + b * c - d / e;",
		},
		{
			name:  "Parenthesized expressions",
			input: "(a + b) * (c - d);",
		},
		{
			name:  "Unary with binary",
			input: "-a + b * -c;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := NewParser(l, "test.oriz")

			program, errors := p.Parse()

			if len(errors) > 0 {
				t.Errorf("Unexpected errors: %v", errors)

				return
			}

			if len(program.Declarations) == 0 {
				t.Error("Expected at least one statement")

				return
			}

			// Verify we got an expression statement.
			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Errorf("Expected ExpressionStatement, got %T", program.Declarations[0])

				return
			}

			if stmt.Expression == nil {
				t.Error("Expected expression, got nil")

				return
			}

			t.Logf("Successfully parsed complex expression: %s", stmt.Expression.String())
		})
	}
}

// TestLanguageFeatures tests various language features.
func TestLanguageFeatures(t *testing.T) {
	input := `
	// Function with parameters and return type.
	func factorial(n: int) -> int {
		if (n <= 1) {
			return 1;
		}
		return n * factorial(n - 1);
	}

	// Variable declarations.
	let pi = 3.14159;
	var counter = 0;
	const MAX_SIZE = 100;

	// Main function.
	func main() {
		let result = factorial(5);
		counter = counter + 1;
		return result;
	}
	`

	l := lexer.New(input)
	p := NewParser(l, "features.oriz")

	program, errors := p.Parse()

	if len(errors) > 0 {
		t.Logf("Parsing errors: %v", errors)
	}

	if program == nil {
		t.Fatal("Failed to parse program")
	}

	// Expect multiple declarations.
	expectedCount := 4 // factorial, pi, counter, main
	if len(program.Declarations) < expectedCount {
		t.Errorf("Expected at least %d declarations, got %d", expectedCount, len(program.Declarations))
	}

	// Verify types of declarations.
	for i, decl := range program.Declarations {
		switch d := decl.(type) {
		case *FunctionDeclaration:
			t.Logf("Declaration %d: Function %s", i, d.Name.Value)
		case *VariableDeclaration:
			t.Logf("Declaration %d: Variable %s", i, d.Name.Value)
		case *ExpressionStatement:
			t.Logf("Declaration %d: Expression statement", i)
		default:
			t.Logf("Declaration %d: Unknown type %T", i, d)
		}
	}

	// Test pretty printing.
	output := PrettyPrint(program)
	t.Logf("Pretty printed AST:\n%s", output)
}

// TestErrorTolerance tests parser error tolerance.
func TestErrorTolerance(t *testing.T) {
	input := `
	func good1() { return 1; }
	
	func bad syntax here invalid
	
	func good2() { return 2; }
	
	let invalid = ;
	
	let valid = 42;
	`

	l := lexer.New(input)
	p := NewParser(l, "error_test.oriz")

	program, errors := p.Parse()

	// Should have errors but still parse valid parts.
	if len(errors) == 0 {
		t.Error("Expected some parsing errors")
	}

	// Should still have parsed some valid declarations.
	if len(program.Declarations) == 0 {
		t.Error("Expected some valid declarations to be parsed")
	}

	validCount := 0

	for _, decl := range program.Declarations {
		if decl != nil {
			validCount++
		}
	}

	t.Logf("Parsed %d valid declarations with %d errors", validCount, len(errors))

	// Should have parsed at least the good functions and valid variable.
	if validCount < 3 {
		t.Errorf("Expected at least 3 valid declarations, got %d", validCount)
	}
}
