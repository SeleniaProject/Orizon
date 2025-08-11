// Specific optimization passes implementation
// This file implements the three core optimization passes:
// 1. Constant Folding Pass
// 2. Dead Code Detection Pass
// 3. Syntax Sugar Removal Pass

package parser

import (
	"fmt"
	"strconv"
)

// ====== Constant Folding Pass ======

// ConstantFoldingPass performs compile-time evaluation of constant expressions
type ConstantFoldingPass struct {
	metrics OptimizationMetrics
}

// NewConstantFoldingPass creates a new constant folding optimization pass
func NewConstantFoldingPass() *ConstantFoldingPass {
	return &ConstantFoldingPass{
		metrics: OptimizationMetrics{
			PassName:           "ConstantFolding",
			NodesProcessed:     0,
			NodesOptimized:     0,
			ConstantsFolded:    0,
			DeadCodeRemoved:    0,
			SyntaxSugarRemoved: 0,
			EstimatedSpeedup:   0.0,
		},
	}
}

// GetName returns the name of this optimization pass
func (cfp *ConstantFoldingPass) GetName() string {
	return "ConstantFolding"
}

// Apply applies constant folding to an AST node
func (cfp *ConstantFoldingPass) Apply(node Node) (Node, bool) {
	cfp.metrics.NodesProcessed++

	switch n := node.(type) {
	case *BinaryExpression:
		return cfp.foldBinaryExpression(n)
	case *UnaryExpression:
		return cfp.foldUnaryExpression(n)
	case *Literal:
		return cfp.normalizeLiteral(n)
	default:
		return node, false
	}
}

// GetMetrics returns optimization statistics
func (cfp *ConstantFoldingPass) GetMetrics() OptimizationMetrics {
	if cfp.metrics.NodesProcessed > 0 {
		cfp.metrics.EstimatedSpeedup = float64(cfp.metrics.ConstantsFolded) / float64(cfp.metrics.NodesProcessed) * 10.0
	}
	return cfp.metrics
}

// foldBinaryExpression performs constant folding on binary expressions
func (cfp *ConstantFoldingPass) foldBinaryExpression(bin *BinaryExpression) (Node, bool) {
	// Check if both operands are literals
	leftLit, leftIsLit := bin.Left.(*Literal)
	rightLit, rightIsLit := bin.Right.(*Literal)

	if !leftIsLit || !rightIsLit {
		return bin, false
	}

	// Attempt to fold based on operator and operand types
	switch bin.Operator.Value {
	case "+":
		return cfp.foldAddition(bin.Span, leftLit, rightLit)
	case "-":
		return cfp.foldSubtraction(bin.Span, leftLit, rightLit)
	case "*":
		return cfp.foldMultiplication(bin.Span, leftLit, rightLit)
	case "/":
		return cfp.foldDivision(bin.Span, leftLit, rightLit)
	case "%":
		return cfp.foldModulo(bin.Span, leftLit, rightLit)
	case "==":
		return cfp.foldEquality(bin.Span, leftLit, rightLit)
	case "!=":
		return cfp.foldInequality(bin.Span, leftLit, rightLit)
	case "<", ">", "<=", ">=":
		return cfp.foldComparison(bin.Span, bin.Operator.Value, leftLit, rightLit)
	case "&&":
		return cfp.foldLogicalAnd(bin.Span, leftLit, rightLit)
	case "||":
		return cfp.foldLogicalOr(bin.Span, leftLit, rightLit)
	}

	return bin, false
}

// foldUnaryExpression performs constant folding on unary expressions
func (cfp *ConstantFoldingPass) foldUnaryExpression(unary *UnaryExpression) (Node, bool) {
	lit, isLit := unary.Operand.(*Literal)
	if !isLit {
		return unary, false
	}

	switch unary.Operator.Value {
	case "-":
		return cfp.foldNegation(unary.Span, lit)
	case "!":
		return cfp.foldLogicalNot(unary.Span, lit)
	}

	return unary, false
}

// normalizeLiteral ensures literals are in canonical form
func (cfp *ConstantFoldingPass) normalizeLiteral(lit *Literal) (Node, bool) {
	// Normalize integer literals (remove leading zeros, etc.)
	if lit.Kind == LiteralInteger {
		if strVal, ok := lit.Value.(string); ok {
			if intVal, err := strconv.Atoi(strVal); err == nil {
				normalizedStr := fmt.Sprintf("%d", intVal)
				if normalizedStr != strVal {
					cfp.metrics.NodesOptimized++
					return &Literal{
						Span:  lit.Span,
						Kind:  LiteralInteger,
						Value: normalizedStr,
					}, true
				}
			}
		}
	}

	return lit, false
}

// Specific folding operations
func (cfp *ConstantFoldingPass) foldAddition(span Span, left, right *Literal) (Node, bool) {
	// Integer addition
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		leftVal, leftErr := strconv.Atoi(left.Value.(string))
		rightVal, rightErr := strconv.Atoi(right.Value.(string))
		if leftErr == nil && rightErr == nil {
			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralInteger,
				Value: fmt.Sprintf("%d", leftVal+rightVal),
			}, true
		}
	}

	// String concatenation
	if left.Kind == LiteralString && right.Kind == LiteralString {
		cfp.metrics.ConstantsFolded++
		cfp.metrics.NodesOptimized++
		return &Literal{
			Span:  span,
			Kind:  LiteralString,
			Value: left.Value.(string) + right.Value.(string),
		}, true
	}

	return nil, false
}

func (cfp *ConstantFoldingPass) foldSubtraction(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		leftVal, leftErr := strconv.Atoi(left.Value.(string))
		rightVal, rightErr := strconv.Atoi(right.Value.(string))
		if leftErr == nil && rightErr == nil {
			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralInteger,
				Value: fmt.Sprintf("%d", leftVal-rightVal),
			}, true
		}
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldMultiplication(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		leftVal, leftErr := strconv.Atoi(left.Value.(string))
		rightVal, rightErr := strconv.Atoi(right.Value.(string))
		if leftErr == nil && rightErr == nil {
			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralInteger,
				Value: fmt.Sprintf("%d", leftVal*rightVal),
			}, true
		}
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldDivision(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		leftVal, leftErr := strconv.Atoi(left.Value.(string))
		rightVal, rightErr := strconv.Atoi(right.Value.(string))
		if leftErr == nil && rightErr == nil && rightVal != 0 {
			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralInteger,
				Value: fmt.Sprintf("%d", leftVal/rightVal),
			}, true
		}
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldModulo(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		leftVal, leftErr := strconv.Atoi(left.Value.(string))
		rightVal, rightErr := strconv.Atoi(right.Value.(string))
		if leftErr == nil && rightErr == nil && rightVal != 0 {
			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralInteger,
				Value: fmt.Sprintf("%d", leftVal%rightVal),
			}, true
		}
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldEquality(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == right.Kind {
		result := false
		switch left.Kind {
		case LiteralInteger:
			leftVal, _ := strconv.Atoi(left.Value.(string))
			rightVal, _ := strconv.Atoi(right.Value.(string))
			result = leftVal == rightVal
		case LiteralString:
			result = left.Value.(string) == right.Value.(string)
		case LiteralBool:
			result = left.Value.(bool) == right.Value.(bool)
		default:
			return nil, false
		}

		cfp.metrics.ConstantsFolded++
		cfp.metrics.NodesOptimized++
		return &Literal{
			Span:  span,
			Kind:  LiteralBool,
			Value: result,
		}, true
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldInequality(span Span, left, right *Literal) (Node, bool) {
	if result, changed := cfp.foldEquality(span, left, right); changed {
		resultLit := result.(*Literal)
		return &Literal{
			Span:  span,
			Kind:  LiteralBool,
			Value: !resultLit.Value.(bool),
		}, true
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldComparison(span Span, op string, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		leftVal, leftErr := strconv.Atoi(left.Value.(string))
		rightVal, rightErr := strconv.Atoi(right.Value.(string))
		if leftErr == nil && rightErr == nil {
			var result bool
			switch op {
			case "<":
				result = leftVal < rightVal
			case ">":
				result = leftVal > rightVal
			case "<=":
				result = leftVal <= rightVal
			case ">=":
				result = leftVal >= rightVal
			}

			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralBool,
				Value: result,
			}, true
		}
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldLogicalAnd(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralBool && right.Kind == LiteralBool {
		cfp.metrics.ConstantsFolded++
		cfp.metrics.NodesOptimized++
		return &Literal{
			Span:  span,
			Kind:  LiteralBool,
			Value: left.Value.(bool) && right.Value.(bool),
		}, true
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldLogicalOr(span Span, left, right *Literal) (Node, bool) {
	if left.Kind == LiteralBool && right.Kind == LiteralBool {
		cfp.metrics.ConstantsFolded++
		cfp.metrics.NodesOptimized++
		return &Literal{
			Span:  span,
			Kind:  LiteralBool,
			Value: left.Value.(bool) || right.Value.(bool),
		}, true
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldNegation(span Span, operand *Literal) (Node, bool) {
	if operand.Kind == LiteralInteger {
		if val, err := strconv.Atoi(operand.Value.(string)); err == nil {
			cfp.metrics.ConstantsFolded++
			cfp.metrics.NodesOptimized++
			return &Literal{
				Span:  span,
				Kind:  LiteralInteger,
				Value: fmt.Sprintf("%d", -val),
			}, true
		}
	}
	return nil, false
}

func (cfp *ConstantFoldingPass) foldLogicalNot(span Span, operand *Literal) (Node, bool) {
	if operand.Kind == LiteralBool {
		cfp.metrics.ConstantsFolded++
		cfp.metrics.NodesOptimized++
		return &Literal{
			Span:  span,
			Kind:  LiteralBool,
			Value: !operand.Value.(bool),
		}, true
	}
	return nil, false
}

// ====== Dead Code Detection Pass ======

// DeadCodeDetectionPass identifies and removes unreachable code
type DeadCodeDetectionPass struct {
	metrics OptimizationMetrics
}

// NewDeadCodeDetectionPass creates a new dead code detection pass
func NewDeadCodeDetectionPass() *DeadCodeDetectionPass {
	return &DeadCodeDetectionPass{
		metrics: OptimizationMetrics{
			PassName:           "DeadCodeDetection",
			NodesProcessed:     0,
			NodesOptimized:     0,
			ConstantsFolded:    0,
			DeadCodeRemoved:    0,
			SyntaxSugarRemoved: 0,
			EstimatedSpeedup:   0.0,
		},
	}
}

// GetName returns the name of this optimization pass
func (dcdp *DeadCodeDetectionPass) GetName() string {
	return "DeadCodeDetection"
}

// Apply applies dead code detection to an AST node
func (dcdp *DeadCodeDetectionPass) Apply(node Node) (Node, bool) {
	dcdp.metrics.NodesProcessed++

	switch n := node.(type) {
	case *IfStatement:
		return dcdp.eliminateDeadBranches(n)
	case *BlockStatement:
		return dcdp.eliminateUnreachableStatements(n)
	default:
		return node, false
	}
}

// GetMetrics returns optimization statistics
func (dcdp *DeadCodeDetectionPass) GetMetrics() OptimizationMetrics {
	if dcdp.metrics.NodesProcessed > 0 {
		dcdp.metrics.EstimatedSpeedup = float64(dcdp.metrics.DeadCodeRemoved) / float64(dcdp.metrics.NodesProcessed) * 5.0
	}
	return dcdp.metrics
}

// eliminateDeadBranches removes unreachable branches in if statements
func (dcdp *DeadCodeDetectionPass) eliminateDeadBranches(ifStmt *IfStatement) (Node, bool) {
	// Check if condition is a constant boolean
	if condLit, isLit := ifStmt.Condition.(*Literal); isLit && condLit.Kind == LiteralBool {
		dcdp.metrics.DeadCodeRemoved++
		dcdp.metrics.NodesOptimized++

		if condLit.Value.(bool) {
			// Condition is always true, return then body
			return ifStmt.ThenStmt, true
		} else {
			// Condition is always false, return else body or nil
			if ifStmt.ElseStmt != nil {
				return ifStmt.ElseStmt, true
			}
			return nil, true // entire if statement is dead
		}
	}

	return ifStmt, false
}

// eliminateUnreachableStatements removes statements after return/break/continue
func (dcdp *DeadCodeDetectionPass) eliminateUnreachableStatements(block *BlockStatement) (Node, bool) {
	newStatements := make([]Statement, 0, len(block.Statements))
	changed := false
	foundTerminator := false

	for _, stmt := range block.Statements {
		if foundTerminator {
			// This statement is unreachable
			dcdp.metrics.DeadCodeRemoved++
			changed = true
			continue
		}

		newStatements = append(newStatements, stmt)

		// Check if this statement terminates control flow
		if dcdp.isTerminatorStatement(stmt) {
			foundTerminator = true
		}
	}

	if changed {
		dcdp.metrics.NodesOptimized++
		return &BlockStatement{
			Span:       block.Span,
			Statements: newStatements,
		}, true
	}

	return block, false
}

// isTerminatorStatement checks if a statement terminates control flow
func (dcdp *DeadCodeDetectionPass) isTerminatorStatement(stmt Statement) bool {
	switch stmt.(type) {
	case *ReturnStatement:
		return true
	// Add more terminator types as needed (break, continue, etc.)
	default:
		return false
	}
}

// ====== Syntax Sugar Removal Pass ======

// SyntaxSugarRemovalPass desugars high-level constructs to basic forms
type SyntaxSugarRemovalPass struct {
	metrics OptimizationMetrics
}

// NewSyntaxSugarRemovalPass creates a new syntax sugar removal pass
func NewSyntaxSugarRemovalPass() *SyntaxSugarRemovalPass {
	return &SyntaxSugarRemovalPass{
		metrics: OptimizationMetrics{
			PassName:           "SyntaxSugarRemoval",
			NodesProcessed:     0,
			NodesOptimized:     0,
			ConstantsFolded:    0,
			DeadCodeRemoved:    0,
			SyntaxSugarRemoved: 0,
			EstimatedSpeedup:   0.0,
		},
	}
}

// GetName returns the name of this optimization pass
func (ssrp *SyntaxSugarRemovalPass) GetName() string {
	return "SyntaxSugarRemoval"
}

// Apply applies syntax sugar removal to an AST node
func (ssrp *SyntaxSugarRemovalPass) Apply(node Node) (Node, bool) {
	ssrp.metrics.NodesProcessed++

	switch n := node.(type) {
	case *AssignmentExpression:
		return ssrp.desugarCompoundAssignment(n)
	default:
		return node, false
	}
}

// GetMetrics returns optimization statistics
func (ssrp *SyntaxSugarRemovalPass) GetMetrics() OptimizationMetrics {
	if ssrp.metrics.NodesProcessed > 0 {
		ssrp.metrics.EstimatedSpeedup = float64(ssrp.metrics.SyntaxSugarRemoved) / float64(ssrp.metrics.NodesProcessed) * 2.0
	}
	return ssrp.metrics
}

// desugarCompoundAssignment converts += to = and +
func (ssrp *SyntaxSugarRemovalPass) desugarCompoundAssignment(assign *AssignmentExpression) (Node, bool) {
	// Check if this is a compound assignment (+=, -=, *=, /=)
	switch assign.Operator.Value {
	case "+=", "-=", "*=", "/=", "%=":
		ssrp.metrics.SyntaxSugarRemoved++
		ssrp.metrics.NodesOptimized++

		// Convert a += b to a = a + b
		baseOp := assign.Operator.Value[:1] // Remove the '='

		return &AssignmentExpression{
			Span: assign.Span,
			Left: assign.Left,
			Operator: &Operator{
				Span:  assign.Operator.Span,
				Value: "=",
				Kind:  assign.Operator.Kind,
			},
			Right: &BinaryExpression{
				Span: assign.Span,
				Left: assign.Left,
				Operator: &Operator{
					Span:  assign.Operator.Span,
					Value: baseOp,
					Kind:  BinaryOp,
				},
				Right: assign.Right,
			},
		}, true
	}

	return assign, false
}
