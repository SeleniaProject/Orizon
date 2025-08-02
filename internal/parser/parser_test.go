// Package parser tests - basic parser functionality tests
// Phase 1.2.1: 再帰下降パーサーテスト
package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// TestParserBasic tests basic parser functionality
func TestParserBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple variable declaration",
			input:    "let x = 42;",
			expected: "VariableDeclaration",
		},
		{
			name:     "Function declaration",
			input:    "func add(a: int, b: int) -> int { return a + b; }",
			expected: "FunctionDeclaration",
		},
		{
			name:     "Expression statement",
			input:    "x + y;",
			expected: "ExpressionStatement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := NewParser(l, "test.oriz")

			program, errors := p.Parse()

			if len(errors) > 0 {
				t.Errorf("Parser errors: %v", errors)
				return
			}

			if program == nil {
				t.Error("Expected program, got nil")
				return
			}

			if len(program.Declarations) == 0 {
				t.Error("Expected at least one declaration")
				return
			}

			// Basic verification that we got some kind of declaration
			decl := program.Declarations[0]
			if decl == nil {
				t.Error("Expected declaration, got nil")
			}

			t.Logf("Successfully parsed: %s", decl.String())
		})
	}
}

// TestParserErrors tests error handling
func TestParserErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError bool
	}{
		{
			name:          "Invalid function syntax",
			input:         "func 123invalid() {}",
			expectedError: true,
		},
		{
			name:          "Missing semicolon",
			input:         "let x = 42",
			expectedError: false, // semicolon is optional in some cases
		},
		{
			name:          "Unclosed brace",
			input:         "func test() { let x = 1",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := NewParser(l, "test.oriz")

			_, errors := p.Parse()

			hasErrors := len(errors) > 0
			if hasErrors != tt.expectedError {
				t.Errorf("Expected error: %v, got errors: %v", tt.expectedError, errors)
			}

			if hasErrors {
				t.Logf("Expected errors found: %v", errors)
			}
		})
	}
}

// TestExpressionParsing tests expression parsing
func TestExpressionParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Binary expression",
			input: "1 + 2 * 3;",
		},
		{
			name:  "Function call",
			input: "foo(1, 2);",
		},
		{
			name:  "Nested expressions",
			input: "(1 + 2) * (3 - 4);",
		},
		{
			name:  "Unary expression",
			input: "-42;",
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

			t.Logf("Successfully parsed expression: %s", program.Declarations[0].String())
		})
	}
}

// TestASTStructure tests AST node structure
func TestASTStructure(t *testing.T) {
	input := "func test() { let x = 42; return x; }"

	l := lexer.New(input)
	p := NewParser(l, "test.oriz")

	program, errors := p.Parse()

	if len(errors) > 0 {
		t.Fatalf("Unexpected errors: %v", errors)
	}

	// Verify program structure
	if len(program.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(program.Declarations))
	}

	// Verify function declaration
	funcDecl, ok := program.Declarations[0].(*FunctionDeclaration)
	if !ok {
		t.Fatalf("Expected FunctionDeclaration, got %T", program.Declarations[0])
	}

	if funcDecl.Name.Value != "test" {
		t.Errorf("Expected function name 'test', got '%s'", funcDecl.Name.Value)
	}

	// Verify function body
	if funcDecl.Body == nil {
		t.Fatal("Expected function body, got nil")
	}

	// Verify body has statements
	if len(funcDecl.Body.Statements) != 2 {
		t.Fatalf("Expected 2 statements in body, got %d", len(funcDecl.Body.Statements))
	}

	// Verify first statement is variable declaration
	if _, ok := funcDecl.Body.Statements[0].(*VariableDeclaration); !ok {
		t.Errorf("Expected first statement to be VariableDeclaration, got %T", funcDecl.Body.Statements[0])
	}

	// Verify second statement is return statement
	if _, ok := funcDecl.Body.Statements[1].(*ReturnStatement); !ok {
		t.Errorf("Expected second statement to be ReturnStatement, got %T", funcDecl.Body.Statements[1])
	}

	t.Log("AST structure verification passed")
}

// TestPrettyPrint tests AST pretty printing
func TestPrettyPrint(t *testing.T) {
	input := "func add(a: int, b: int) { return a + b; }"

	l := lexer.New(input)
	p := NewParser(l, "test.oriz")

	program, errors := p.Parse()

	if len(errors) > 0 {
		t.Fatalf("Unexpected errors: %v", errors)
	}

	// Test pretty printing
	output := PrettyPrint(program)
	if output == "" {
		t.Error("Expected non-empty pretty print output")
	}

	t.Logf("Pretty print output:\n%s", output)
}

// BenchmarkParser benchmarks parser performance
func BenchmarkParser(b *testing.B) {
	input := `
	func fibonacci(n: int) -> int {
		if (n <= 1) {
			return n;
		}
		return fibonacci(n - 1) + fibonacci(n - 2);
	}
	
	func main() {
		let result = fibonacci(10);
		return result;
	}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := NewParser(l, "benchmark.oriz")

		_, _ = p.Parse()
	}
}

// TestParserRecovery tests parser error recovery
func TestParserRecovery(t *testing.T) {
	// Test that parser can recover from errors and continue parsing
	input := `
	func invalid syntax here
	func valid() { return 42; }
	let x = 1;
	`

	l := lexer.New(input)
	p := NewParser(l, "test.oriz")

	program, errors := p.Parse()

	// Should have errors but still parse some valid declarations
	if len(errors) == 0 {
		t.Error("Expected parsing errors for invalid syntax")
	}

	// Should still have parsed the valid function and variable
	if len(program.Declarations) == 0 {
		t.Error("Expected some declarations to be parsed despite errors")
	}

	t.Logf("Parser recovered and found %d declarations with %d errors",
		len(program.Declarations), len(errors))
}
