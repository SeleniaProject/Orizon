package ast

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/position"
)

// createOptimizationTestSpan creates a basic position span for optimization testing
func createOptimizationTestSpan(line, col int) position.Span {
	return position.Span{
		Start: position.Position{Filename: "test.oriz", Line: line, Column: col},
		End:   position.Position{Filename: "test.oriz", Line: line, Column: col + 1},
	}
}

// TestOptimizationPipeline tests the basic optimization pipeline functionality
func TestOptimizationPipeline(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	// Create a simple program with optimization opportunities
	left := &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"}
	right := &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"}
	expr := &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: OpAdd,
		Right:    right,
	}

	stmt := &ExpressionStatement{
		Span:       span,
		Expression: expr,
	}

	block := &BlockStatement{
		Span:       span,
		Statements: []Statement{stmt},
	}

	fn := &FunctionDeclaration{
		Span: span,
		Name: &Identifier{Span: span, Value: "test"},
		Body: block,
	}

	program := &Program{
		Span:         span,
		Declarations: []Declaration{fn},
	}

	// Create and run optimization pipeline
	pipeline := CreateStandardOptimizationPipeline()

	optimized, stats, err := pipeline.Optimize(program)
	if err != nil {
		t.Errorf("Optimization failed: %v", err)
	}

	if optimized == nil {
		t.Error("Optimization returned nil")
	}

	if stats == nil {
		t.Error("Optimization stats not returned")
	}

	if stats.NodesVisited == 0 {
		t.Error("No nodes were visited during optimization")
	}
}

// TestConstantFoldingPass tests the constant folding optimization pass
func TestConstantFoldingPass(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	tests := []struct {
		name     string
		left     *Literal
		operator Operator
		right    *Literal
		expected interface{}
	}{
		{
			name:     "Integer addition",
			left:     &Literal{Span: span, Kind: LiteralInteger, Value: 3, Raw: "3"},
			operator: OpAdd,
			right:    &Literal{Span: span, Kind: LiteralInteger, Value: 4, Raw: "4"},
			expected: int64(7),
		},
		{
			name:     "Integer subtraction",
			left:     &Literal{Span: span, Kind: LiteralInteger, Value: 10, Raw: "10"},
			operator: OpSub,
			right:    &Literal{Span: span, Kind: LiteralInteger, Value: 3, Raw: "3"},
			expected: int64(7),
		},
		{
			name:     "Integer multiplication",
			left:     &Literal{Span: span, Kind: LiteralInteger, Value: 5, Raw: "5"},
			operator: OpMul,
			right:    &Literal{Span: span, Kind: LiteralInteger, Value: 6, Raw: "6"},
			expected: int64(30),
		},
		{
			name:     "Boolean AND true",
			left:     &Literal{Span: span, Kind: LiteralBoolean, Value: true, Raw: "true"},
			operator: OpAnd,
			right:    &Literal{Span: span, Kind: LiteralBoolean, Value: true, Raw: "true"},
			expected: true,
		},
		{
			name:     "Boolean AND false",
			left:     &Literal{Span: span, Kind: LiteralBoolean, Value: true, Raw: "true"},
			operator: OpAnd,
			right:    &Literal{Span: span, Kind: LiteralBoolean, Value: false, Raw: "false"},
			expected: false,
		},
		{
			name:     "String concatenation",
			left:     &Literal{Span: span, Kind: LiteralString, Value: "Hello", Raw: "\"Hello\""},
			operator: OpAdd,
			right:    &Literal{Span: span, Kind: LiteralString, Value: " World", Raw: "\" World\""},
			expected: "Hello World",
		},
	}

	pass := NewConstantFoldingPass()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expr := &BinaryExpression{
				Span:     span,
				Left:     test.left,
				Operator: test.operator,
				Right:    test.right,
			}

			result, stats, err := pass.Apply(expr)
			if err != nil {
				t.Errorf("Constant folding failed: %v", err)
			}

			if stats.ConstantsFolded == 0 {
				t.Error("Expected constant to be folded")
			}

			// Check if result is a literal with the expected value
			if lit, ok := result.(*Literal); ok {
				if lit.Value != test.expected {
					t.Errorf("Expected %v, got %v", test.expected, lit.Value)
				}
			} else {
				t.Error("Expected literal result from constant folding")
			}
		})
	}
}

// TestDeadCodeEliminationPass tests the dead code elimination optimization pass
func TestDeadCodeEliminationPass(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	// Create a block with unreachable code after return
	returnStmt := &ReturnStatement{
		Span:  span,
		Value: &Literal{Span: span, Kind: LiteralInteger, Value: 42, Raw: "42"},
	}

	deadStmt := &ExpressionStatement{
		Span:       span,
		Expression: &Literal{Span: span, Kind: LiteralInteger, Value: 99, Raw: "99"},
	}

	block := &BlockStatement{
		Span:       span,
		Statements: []Statement{returnStmt, deadStmt},
	}

	pass := NewDeadCodeEliminationPass()

	result, stats, err := pass.Apply(block)
	if err != nil {
		t.Errorf("Dead code elimination failed: %v", err)
	}

	if stats.DeadCodeRemoved == 0 {
		t.Error("Expected dead code to be removed")
	}

	// Check if unreachable statement was removed
	if resultBlock, ok := result.(*BlockStatement); ok {
		if len(resultBlock.Statements) != 1 {
			t.Errorf("Expected 1 statement after dead code removal, got %d", len(resultBlock.Statements))
		}
	} else {
		t.Error("Expected block statement result")
	}
}

// TestDeadCodeIfElimination tests dead code elimination in if statements
func TestDeadCodeIfElimination(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	// Create if statement with constant true condition
	trueCond := &Literal{Span: span, Kind: LiteralBoolean, Value: true, Raw: "true"}
	thenBlock := &BlockStatement{
		Span: span,
		Statements: []Statement{
			&ExpressionStatement{
				Span:       span,
				Expression: &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"},
			},
		},
	}
	elseBlock := &BlockStatement{
		Span: span,
		Statements: []Statement{
			&ExpressionStatement{
				Span:       span,
				Expression: &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"},
			},
		},
	}

	ifStmt := &IfStatement{
		Span:      span,
		Condition: trueCond,
		ThenBlock: thenBlock,
		ElseBlock: elseBlock,
	}

	pass := NewDeadCodeEliminationPass()

	result, stats, err := pass.Apply(ifStmt)
	if err != nil {
		t.Errorf("Dead code elimination failed: %v", err)
	}

	if stats.DeadCodeRemoved == 0 {
		t.Error("Expected dead code to be removed")
	}

	// Result should be the then block since condition is always true
	if resultBlock, ok := result.(*BlockStatement); ok {
		if len(resultBlock.Statements) != 1 {
			t.Errorf("Expected 1 statement in optimized block, got %d", len(resultBlock.Statements))
		}
	} else {
		t.Error("Expected block statement result from if optimization")
	}
}

// TestSyntaxSugarRemovalPass tests the syntax sugar removal optimization pass
func TestSyntaxSugarRemovalPass(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	tests := []struct {
		name     string
		expr     *BinaryExpression
		expected bool // Whether transformation should occur
	}{
		{
			name: "Boolean AND with true",
			expr: &BinaryExpression{
				Span:     span,
				Left:     &Literal{Span: span, Kind: LiteralBoolean, Value: true, Raw: "true"},
				Operator: OpAnd,
				Right:    &Identifier{Span: span, Value: "x"},
			},
			expected: true,
		},
		{
			name: "Boolean OR with false",
			expr: &BinaryExpression{
				Span:     span,
				Left:     &Literal{Span: span, Kind: LiteralBoolean, Value: false, Raw: "false"},
				Operator: OpOr,
				Right:    &Identifier{Span: span, Value: "x"},
			},
			expected: true,
		},
		{
			name: "Addition with zero",
			expr: &BinaryExpression{
				Span:     span,
				Left:     &Identifier{Span: span, Value: "x"},
				Operator: OpAdd,
				Right:    &Literal{Span: span, Kind: LiteralInteger, Value: 0, Raw: "0"},
			},
			expected: true,
		},
		{
			name: "Multiplication with one",
			expr: &BinaryExpression{
				Span:     span,
				Left:     &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"},
				Operator: OpMul,
				Right:    &Identifier{Span: span, Value: "x"},
			},
			expected: true,
		},
		{
			name: "Multiplication with zero",
			expr: &BinaryExpression{
				Span:     span,
				Left:     &Literal{Span: span, Kind: LiteralInteger, Value: 0, Raw: "0"},
				Operator: OpMul,
				Right:    &Identifier{Span: span, Value: "x"},
			},
			expected: true,
		},
	}

	pass := NewSyntaxSugarRemovalPass()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, stats, err := pass.Apply(test.expr)
			if err != nil {
				t.Errorf("Syntax sugar removal failed: %v", err)
			}

			if test.expected && stats.SyntaxSugarRemoved == 0 {
				t.Error("Expected syntax sugar to be removed")
			}

			if result == nil {
				t.Error("Expected non-nil result from syntax sugar removal")
			}
		})
	}
}

// TestOptimizationLevels tests different optimization levels
func TestOptimizationLevels(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	// Create a simple expression that can be optimized
	expr := &BinaryExpression{
		Span:     span,
		Left:     &Literal{Span: span, Kind: LiteralInteger, Value: 5, Raw: "5"},
		Operator: OpAdd,
		Right:    &Literal{Span: span, Kind: LiteralInteger, Value: 3, Raw: "3"},
	}

	tests := []struct {
		name  string
		level OptimizationLevel
	}{
		{"None", OptimizationNone},
		{"Basic", OptimizationBasic},
		{"Default", OptimizationDefault},
		{"Aggressive", OptimizationAggressive},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pipeline := NewOptimizationPipeline()
			pipeline.SetOptimizationLevel(test.level)
			pipeline.AddPass(NewConstantFoldingPass())
			pipeline.AddPass(NewDeadCodeEliminationPass())
			pipeline.AddPass(NewSyntaxSugarRemovalPass())

			result, stats, err := pipeline.Optimize(expr)
			if err != nil {
				t.Errorf("Optimization failed: %v", err)
			}

			if result == nil {
				t.Error("Expected non-nil result")
			}

			if stats == nil {
				t.Error("Expected optimization stats")
			}

			// At basic level and above, constant folding should occur
			if test.level >= OptimizationBasic {
				if stats.ConstantsFolded == 0 {
					t.Error("Expected constant folding at basic optimization level")
				}
			}
		})
	}
}

// TestAggressiveOptimizationPipeline tests the aggressive optimization pipeline
func TestAggressiveOptimizationPipeline(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	// Create a complex expression with multiple optimization opportunities
	innerExpr := &BinaryExpression{
		Span:     span,
		Left:     &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"},
		Operator: OpMul,
		Right:    &Literal{Span: span, Kind: LiteralInteger, Value: 3, Raw: "3"},
	}

	outerExpr := &BinaryExpression{
		Span:     span,
		Left:     innerExpr,
		Operator: OpAdd,
		Right:    &Literal{Span: span, Kind: LiteralInteger, Value: 0, Raw: "0"},
	}

	pipeline := CreateAggressiveOptimizationPipeline()

	result, stats, err := pipeline.Optimize(outerExpr)
	if err != nil {
		t.Errorf("Aggressive optimization failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if stats == nil {
		t.Error("Expected optimization stats")
	}

	// Should have folded constants
	if stats.ConstantsFolded == 0 {
		t.Error("Expected constants to be folded")
	}

	// Syntax sugar removal may or may not occur depending on optimization order
	// The important thing is that we get the correct final result

	// Final result should be a single literal with value 6
	if lit, ok := result.(*Literal); ok {
		var value int64
		switch v := lit.Value.(type) {
		case int:
			value = int64(v)
		case int64:
			value = v
		default:
			t.Errorf("Expected integer value, got %T", lit.Value)
			return
		}
		if value != 6 {
			t.Errorf("Expected final value 6, got %d", value)
		}
	} else {
		t.Error("Expected final result to be a literal")
	}
}

// TestOptimizationStats tests optimization statistics tracking
func TestOptimizationStats(t *testing.T) {
	span := createOptimizationTestSpan(1, 1)

	// Create a program with various optimization opportunities
	expr1 := &BinaryExpression{
		Span:     span,
		Left:     &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"},
		Operator: OpAdd,
		Right:    &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"},
	}

	expr2 := &BinaryExpression{
		Span:     span,
		Left:     &Literal{Span: span, Kind: LiteralInteger, Value: 3, Raw: "3"},
		Operator: OpMul,
		Right:    &Literal{Span: span, Kind: LiteralInteger, Value: 0, Raw: "0"},
	}

	block := &BlockStatement{
		Span: span,
		Statements: []Statement{
			&ExpressionStatement{Span: span, Expression: expr1},
			&ExpressionStatement{Span: span, Expression: expr2},
		},
	}

	pipeline := CreateStandardOptimizationPipeline()
	pipeline.SetStatsEnabled(true)

	result, stats, err := pipeline.Optimize(block)
	if err != nil {
		t.Errorf("Optimization failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if stats == nil {
		t.Error("Expected optimization stats")
	}

	// Verify stats collection
	if stats.NodesVisited == 0 {
		t.Error("Expected nodes to be visited")
	}

	if stats.PassName != "Global" {
		t.Errorf("Expected global stats, got %s", stats.PassName)
	}

	// Test stats string representation
	statsStr := stats.String()
	if statsStr == "" {
		t.Error("Expected non-empty stats string")
	}
}
