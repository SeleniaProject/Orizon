// Package parser tests - Phase 1.2.2: Pratt Parser Integration Tests
// Comprehensive test suite for enhanced operator precedence and associativity
package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// TestOperatorPrecedence tests complete operator precedence hierarchy
func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic arithmetic precedence",
			input:    "1 + 2 * 3;",
			expected: "(1 + (2 * 3))",
		},
		{
			name:     "Power operator precedence",
			input:    "2 * 3 ** 4;",
			expected: "(2 * (3 ** 4))",
		},
		{
			name:     "Power right associativity",
			input:    "2 ** 3 ** 4;",
			expected: "(2 ** (3 ** 4))",
		},
		{
			name:     "Comparison and arithmetic",
			input:    "1 + 2 < 3 * 4;",
			expected: "((1 + 2) < (3 * 4))",
		},
		{
			name:     "Logical operators",
			input:    "a && b || c;",
			expected: "((a && b) || c)",
		},
		{
			name:     "Bitwise operators",
			input:    "a | b & c ^ d;",
			expected: "(a | ((b & c) ^ d))",
		},
		{
			name:     "Shift operators",
			input:    "a + b << c * d;",
			expected: "((a + b) << (c * d))",
		},
		{
			name:     "Mixed precedence complex",
			input:    "a + b * c << d & e | f;",
			expected: "((((a + (b * c)) << d) & e) | f)",
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

			if len(program.Declarations) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(program.Declarations))
			}

			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Fatalf("Expected ExpressionStatement, got %T", program.Declarations[0])
			}

			actual := stmt.Expression.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestAssignmentOperators tests assignment operator precedence and associativity
func TestAssignmentOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple assignment",
			input:    "a = b + c;",
			expected: "(a = (b + c))",
		},
		{
			name:     "Right associative assignment",
			input:    "a = b = c;",
			expected: "(a = (b = c))",
		},
		{
			name:     "Compound assignment",
			input:    "a += b * c;",
			expected: "(a += (b * c))",
		},
		{
			name:     "Multiple compound assignments",
			input:    "a += b -= c;",
			expected: "(a += (b -= c))",
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

			if len(program.Declarations) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(program.Declarations))
			}

			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Fatalf("Expected ExpressionStatement, got %T", program.Declarations[0])
			}

			actual := stmt.Expression.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestUnaryOperators tests unary operators with proper precedence
func TestUnaryOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unary minus",
			input:    "-a + b;",
			expected: "((-a) + b)",
		},
		{
			name:     "Unary not",
			input:    "!a && b;",
			expected: "((!a) && b)",
		},
		{
			name:     "Bitwise not",
			input:    "~a | b;",
			expected: "((~a) | b)",
		},
		{
			name:     "Multiple unary operators",
			input:    "!!a;",
			expected: "(!(!a))",
		},
		{
			name:     "Unary with power",
			input:    "-a ** b;",
			expected: "(-(a ** b))",
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

			if len(program.Declarations) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(program.Declarations))
			}

			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Fatalf("Expected ExpressionStatement, got %T", program.Declarations[0])
			}

			actual := stmt.Expression.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestTernaryOperator tests ternary conditional operator
func TestTernaryOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple ternary",
			input:    "a ? b : c;",
			expected: "(a ? b : c)",
		},
		{
			name:     "Ternary with precedence",
			input:    "a + b ? c * d : e - f;",
			expected: "((a + b) ? (c * d) : (e - f))",
		},
		{
			name:     "Right associative ternary",
			input:    "a ? b : c ? d : e;",
			expected: "(a ? b : (c ? d : e))",
		},
		{
			name:     "Ternary with assignment",
			input:    "a = b ? c : d;",
			expected: "(a = (b ? c : d))",
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

			if len(program.Declarations) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(program.Declarations))
			}

			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Fatalf("Expected ExpressionStatement, got %T", program.Declarations[0])
			}

			actual := stmt.Expression.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestPrattComplexExpressions tests complex expression combinations
func TestPrattComplexExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Function call with operators",
			input:    "foo(a + b) * c;",
			expected: "(foo(...) * c)",
		},
		{
			name:     "Member access with operators",
			input:    "obj.field + other.value;",
			expected: "((obj . field) + (other . value))",
		},
		{
			name:     "Array index with operators",
			input:    "arr[i + 1] * factor;",
			expected: "(arr(...) * factor)",
		},
		{
			name:     "Mixed access patterns",
			input:    "obj.method(arg)[index].field;",
			expected: "((obj . method)(...)(...) . field)",
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

			if len(program.Declarations) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(program.Declarations))
			}

			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Fatalf("Expected ExpressionStatement, got %T", program.Declarations[0])
			}

			actual := stmt.Expression.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestAssociativityEdgeCases tests edge cases for associativity
func TestAssociativityEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Power tower right associative",
			input:    "2 ** 3 ** 4 ** 5;",
			expected: "(2 ** (3 ** (4 ** 5)))",
		},
		{
			name:     "Assignment chain right associative",
			input:    "a = b = c = d;",
			expected: "(a = (b = (c = d)))",
		},
		{
			name:     "Mixed assignment types",
			input:    "a += b *= c;",
			expected: "(a += (b *= c))",
		},
		{
			name:     "Left associative arithmetic",
			input:    "a - b - c - d;",
			expected: "(((a - b) - c) - d)",
		},
		{
			name:     "Left associative division",
			input:    "a / b / c / d;",
			expected: "(((a / b) / c) / d)",
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

			if len(program.Declarations) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(program.Declarations))
			}

			stmt, ok := program.Declarations[0].(*ExpressionStatement)
			if !ok {
				t.Fatalf("Expected ExpressionStatement, got %T", program.Declarations[0])
			}

			actual := stmt.Expression.String()
			if actual != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// BenchmarkPrattParser benchmarks the enhanced Pratt parser performance
func BenchmarkPrattParser(b *testing.B) {
	complexExpr := `
	a = b + c * d ** e - f / g % h << i & j | k ^ l && m || n ? o : p += q -= r *= s /= t %= u;
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(complexExpr)
		p := NewParser(l, "benchmark.oriz")
		_, _ = p.Parse()
	}
}

// TestPrattParserErrorRecovery tests error recovery in complex expressions
func TestPrattParserErrorRecovery(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectErrors  bool
		minDeclsCount int
	}{
		{
			name:          "Missing operand",
			input:         "a + ;",
			expectErrors:  true,
			minDeclsCount: 1,
		},
		{
			name:          "Invalid operator sequence",
			input:         "a @ b;", // @ is not a valid operator
			expectErrors:  true,
			minDeclsCount: 1,
		},
		{
			name:          "Unmatched parentheses",
			input:         "a + (b * c;",
			expectErrors:  true,
			minDeclsCount: 1,
		},
		{
			name:          "Recovery continues parsing",
			input:         "a + ; b = c;",
			expectErrors:  true,
			minDeclsCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := NewParser(l, "test.oriz")

			program, errors := p.Parse()

			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErrors {
				t.Errorf("Expected errors: %v, got errors: %v", tt.expectErrors, errors)
			}

			if len(program.Declarations) < tt.minDeclsCount {
				t.Errorf("Expected at least %d declarations, got %d", tt.minDeclsCount, len(program.Declarations))
			}

			if hasErrors {
				t.Logf("Expected errors found: %v", errors)
			}
		})
	}
}
