// Package ast - Dead Code Elimination and Syntax Sugar Removal.
// Phase 1.3.3: Additional AST optimization passes for code cleanup and simplification
// This file extends the optimization system with dead code detection and syntax sugar removal.
package ast

// ===== Dead Code Elimination Pass =====.

// DeadCodeEliminationPass removes unreachable code and unused declarations.
// Improves code quality and reduces compilation overhead.
type DeadCodeEliminationPass struct {
	removeUnreachable bool // Whether to remove unreachable statements
	removeUnused      bool // Whether to remove unused declarations
	aggressive        bool // Whether to apply aggressive optimizations
}

// NewDeadCodeEliminationPass creates a new dead code elimination pass.
func NewDeadCodeEliminationPass() *DeadCodeEliminationPass {
	return &DeadCodeEliminationPass{
		removeUnreachable: true,
		removeUnused:      false, // Conservative by default, requires usage analysis
		aggressive:        false,
	}
}

// Name returns the name of this optimization pass.
func (dcep *DeadCodeEliminationPass) Name() string {
	return "DeadCodeElimination"
}

// ShouldApply determines if dead code elimination should be applied at the given optimization level.
func (dcep *DeadCodeEliminationPass) ShouldApply(level OptimizationLevel) bool {
	return level >= OptimizationDefault
}

// Apply performs dead code elimination on the AST.
func (dcep *DeadCodeEliminationPass) Apply(node Node) (Node, *OptimizationStats, error) {
	stats := &OptimizationStats{PassName: dcep.Name()}
	visitor := &deadCodeVisitor{
		pass:  dcep,
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

// deadCodeVisitor implements the visitor pattern for dead code elimination.
type deadCodeVisitor struct {
	pass  *DeadCodeEliminationPass
	stats *OptimizationStats
}

// New nodes support: Import/Export declarations and items.
func (dcv *deadCodeVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitExportItem(node *ExportItem) interface{} {
	dcv.stats.NodesVisited++

	return node
}

// Structural and generic-related nodes (no-op for dead code pass).
func (dcv *deadCodeVisitor) VisitStructDeclaration(node *StructDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitEnumDeclaration(node *EnumDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitTraitDeclaration(node *TraitDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitImplDeclaration(node *ImplDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitStructField(node *StructField) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitEnumVariant(node *EnumVariant) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitTraitMethod(node *TraitMethod) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitGenericParameter(node *GenericParameter) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitWherePredicate(node *WherePredicate) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitAssociatedType(node *AssociatedType) interface{} {
	dcv.stats.NodesVisited++

	return node
}

// VisitBlockStatement removes unreachable statements after return/break/continue.
func (dcv *deadCodeVisitor) VisitBlockStatement(node *BlockStatement) interface{} {
	dcv.stats.NodesVisited++

	if !dcv.pass.removeUnreachable {
		// Just traverse without removing code.
		for i, stmt := range node.Statements {
			if result := stmt.Accept(dcv); result != nil {
				if newStmt, ok := result.(Statement); ok {
					node.Statements[i] = newStmt
				}
			}
		}

		return node
	}

	newStatements := make([]Statement, 0, len(node.Statements))
	reachable := true

	for _, stmt := range node.Statements {
		if !reachable {
			// Code after return/break/continue is unreachable
			dcv.stats.DeadCodeRemoved++

			continue
		}

		// Recursively optimize the statement.
		if result := stmt.Accept(dcv); result != nil {
			if newStmt, ok := result.(Statement); ok {
				stmt = newStmt
			}
		}

		newStatements = append(newStatements, stmt)

		// Check if this statement makes subsequent code unreachable.
		if dcv.isTerminatingStatement(stmt) {
			reachable = false
		}
	}

	// Update the statements if any were removed.
	if len(newStatements) < len(node.Statements) {
		dcv.stats.NodesTransformed++
		node.Statements = newStatements
	}

	return node
}

// isTerminatingStatement checks if a statement unconditionally terminates control flow.
func (dcv *deadCodeVisitor) isTerminatingStatement(stmt Statement) bool {
	switch s := stmt.(type) {
	case *ReturnStatement:
		return true
	case *IfStatement:
		// Both branches must terminate for the if statement to be terminating.
		if s.ElseBlock != nil {
			return dcv.blockAlwaysTerminates(s.ThenBlock) && dcv.blockAlwaysTerminates(dcv.convertToBlockStatement(s.ElseBlock))
		}

		return false
	default:
		return false
	}
}

// blockAlwaysTerminates checks if a block statement always terminates.
func (dcv *deadCodeVisitor) blockAlwaysTerminates(block *BlockStatement) bool {
	for _, stmt := range block.Statements {
		if dcv.isTerminatingStatement(stmt) {
			return true
		}
	}

	return false
}

// convertToBlockStatement safely converts a Statement to BlockStatement.
func (dcv *deadCodeVisitor) convertToBlockStatement(stmt Statement) *BlockStatement {
	if block, ok := stmt.(*BlockStatement); ok {
		return block
	}
	// If it's not a block statement, we can't analyze termination properly.
	return &BlockStatement{Statements: []Statement{}}
}

// VisitIfStatement optimizes conditional statements and removes impossible branches.
func (dcv *deadCodeVisitor) VisitIfStatement(node *IfStatement) interface{} {
	dcv.stats.NodesVisited++

	// Optimize condition.
	if result := node.Condition.Accept(dcv); result != nil {
		if newCond, ok := result.(Expression); ok {
			node.Condition = newCond
		}
	}

	// Check for constant condition (after optimization).
	if lit, isLit := node.Condition.(*Literal); isLit && lit.Kind == LiteralBoolean {
		if condValue, ok := lit.Value.(bool); ok {
			dcv.stats.NodesTransformed++
			dcv.stats.DeadCodeRemoved++

			if condValue {
				// Condition is always true, replace with then branch.
				return node.ThenBlock.Accept(dcv)
			} else {
				// Condition is always false, replace with else branch or remove.
				if node.ElseBlock != nil {
					return node.ElseBlock.Accept(dcv)
				} else {
					// Return empty block.
					return &BlockStatement{Span: node.Span, Statements: []Statement{}}
				}
			}
		}
	}

	// Optimize branches.
	if result := node.ThenBlock.Accept(dcv); result != nil {
		if newThen, ok := result.(*BlockStatement); ok {
			node.ThenBlock = newThen
		}
	}

	if node.ElseBlock != nil {
		if result := node.ElseBlock.Accept(dcv); result != nil {
			if newElse, ok := result.(Statement); ok {
				node.ElseBlock = newElse
			}
		}
	}

	return node
}

// Implement visitor methods for other node types.
func (dcv *deadCodeVisitor) VisitProgram(node *Program) interface{} {
	dcv.stats.NodesVisited++
	for i, decl := range node.Declarations {
		if result := decl.Accept(dcv); result != nil {
			if newDecl, ok := result.(Declaration); ok {
				node.Declarations[i] = newDecl
			}
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitComment(node *Comment) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} {
	dcv.stats.NodesVisited++
	if node.Body != nil {
		if result := node.Body.Accept(dcv); result != nil {
			if newBody, ok := result.(*BlockStatement); ok {
				node.Body = newBody
			}
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitParameter(node *Parameter) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} {
	dcv.stats.NodesVisited++
	if node.Value != nil {
		if result := node.Value.Accept(dcv); result != nil {
			if newValue, ok := result.(Expression); ok {
				node.Value = newValue
			}
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} {
	dcv.stats.NodesVisited++
	if result := node.Expression.Accept(dcv); result != nil {
		if newExpr, ok := result.(Expression); ok {
			node.Expression = newExpr
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitReturnStatement(node *ReturnStatement) interface{} {
	dcv.stats.NodesVisited++
	if node.Value != nil {
		if result := node.Value.Accept(dcv); result != nil {
			if newValue, ok := result.(Expression); ok {
				node.Value = newValue
			}
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitWhileStatement(node *WhileStatement) interface{} {
	dcv.stats.NodesVisited++

	// Optimize condition.
	if result := node.Condition.Accept(dcv); result != nil {
		if newCond, ok := result.(Expression); ok {
			node.Condition = newCond
		}
	}

	// Check for constant false condition.
	if lit, isLit := node.Condition.(*Literal); isLit && lit.Kind == LiteralBoolean {
		if condValue, ok := lit.Value.(bool); ok && !condValue {
			// Condition is always false, remove the loop.
			dcv.stats.NodesTransformed++
			dcv.stats.DeadCodeRemoved++

			return &BlockStatement{Span: node.Span, Statements: []Statement{}}
		}
	}

	// Optimize body.
	if result := node.Body.Accept(dcv); result != nil {
		if newBody, ok := result.(*BlockStatement); ok {
			node.Body = newBody
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitIdentifier(node *Identifier) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitLiteral(node *Literal) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitBinaryExpression(node *BinaryExpression) interface{} {
	dcv.stats.NodesVisited++

	// Optimize operands.
	if result := node.Left.Accept(dcv); result != nil {
		if newLeft, ok := result.(Expression); ok {
			node.Left = newLeft
		}
	}

	if result := node.Right.Accept(dcv); result != nil {
		if newRight, ok := result.(Expression); ok {
			node.Right = newRight
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitUnaryExpression(node *UnaryExpression) interface{} {
	dcv.stats.NodesVisited++

	if result := node.Operand.Accept(dcv); result != nil {
		if newOperand, ok := result.(Expression); ok {
			node.Operand = newOperand
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitCallExpression(node *CallExpression) interface{} {
	dcv.stats.NodesVisited++

	if result := node.Function.Accept(dcv); result != nil {
		if newFunc, ok := result.(Expression); ok {
			node.Function = newFunc
		}
	}

	for i, arg := range node.Arguments {
		if result := arg.Accept(dcv); result != nil {
			if newArg, ok := result.(Expression); ok {
				node.Arguments[i] = newArg
			}
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitMemberExpression(node *MemberExpression) interface{} {
	dcv.stats.NodesVisited++

	if result := node.Object.Accept(dcv); result != nil {
		if newObj, ok := result.(Expression); ok {
			node.Object = newObj
		}
	}

	return node
}

func (dcv *deadCodeVisitor) VisitBasicType(node *BasicType) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitIdentifierType(node *IdentifierType) interface{} {
	dcv.stats.NodesVisited++

	return node
}

func (dcv *deadCodeVisitor) VisitAttribute(node *Attribute) interface{} {
	dcv.stats.NodesVisited++

	return node
}

// ===== Syntax Sugar Removal Pass =====.

// SyntaxSugarRemovalPass simplifies complex syntax constructs into basic forms.
// Reduces the complexity of subsequent compilation phases.
type SyntaxSugarRemovalPass struct {
	removeCompoundAssignment bool // Whether to expand compound assignments (+=, -=, etc.)
	simplifyControlFlow      bool // Whether to simplify complex control flow
	expandFunctionSugar      bool // Whether to expand function syntax sugar
}

// NewSyntaxSugarRemovalPass creates a new syntax sugar removal pass.
func NewSyntaxSugarRemovalPass() *SyntaxSugarRemovalPass {
	return &SyntaxSugarRemovalPass{
		removeCompoundAssignment: true,
		simplifyControlFlow:      true,
		expandFunctionSugar:      false, // May break semantics in some cases
	}
}

// Name returns the name of this optimization pass.
func (ssrp *SyntaxSugarRemovalPass) Name() string {
	return "SyntaxSugarRemoval"
}

// ShouldApply determines if syntax sugar removal should be applied at the given optimization level.
func (ssrp *SyntaxSugarRemovalPass) ShouldApply(level OptimizationLevel) bool {
	return level >= OptimizationBasic
}

// Apply performs syntax sugar removal on the AST.
func (ssrp *SyntaxSugarRemovalPass) Apply(node Node) (Node, *OptimizationStats, error) {
	stats := &OptimizationStats{PassName: ssrp.Name()}
	visitor := &syntaxSugarVisitor{
		pass:  ssrp,
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

// syntaxSugarVisitor implements the visitor pattern for syntax sugar removal.
type syntaxSugarVisitor struct {
	pass  *SyntaxSugarRemovalPass
	stats *OptimizationStats
}

// New nodes support: Import/Export declarations and items.
func (ssv *syntaxSugarVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitExportItem(node *ExportItem) interface{} {
	ssv.stats.NodesVisited++

	return node
}

// Structural and generic-related nodes (no-op for syntax sugar pass).
func (ssv *syntaxSugarVisitor) VisitStructDeclaration(node *StructDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitEnumDeclaration(node *EnumDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitTraitDeclaration(node *TraitDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitImplDeclaration(node *ImplDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitStructField(node *StructField) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitEnumVariant(node *EnumVariant) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitTraitMethod(node *TraitMethod) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitGenericParameter(node *GenericParameter) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitWherePredicate(node *WherePredicate) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitAssociatedType(node *AssociatedType) interface{} {
	ssv.stats.NodesVisited++

	return node
}

// VisitBinaryExpression simplifies certain binary expressions.
func (ssv *syntaxSugarVisitor) VisitBinaryExpression(node *BinaryExpression) interface{} {
	ssv.stats.NodesVisited++

	// Recursively optimize operands.
	if result := node.Left.Accept(ssv); result != nil {
		if newLeft, ok := result.(Expression); ok {
			node.Left = newLeft
		}
	}

	if result := node.Right.Accept(ssv); result != nil {
		if newRight, ok := result.(Expression); ok {
			node.Right = newRight
		}
	}

	// Simplify boolean expressions with constant operands.
	if ssv.pass.simplifyControlFlow {
		if simplified := ssv.simplifyBooleanExpression(node); simplified != nil {
			ssv.stats.NodesTransformed++
			ssv.stats.SyntaxSugarRemoved++

			return simplified
		}
	}

	return node
}

// simplifyBooleanExpression simplifies boolean expressions with constant operands.
func (ssv *syntaxSugarVisitor) simplifyBooleanExpression(node *BinaryExpression) Expression {
	// Simplify logical AND with constant operands.
	if node.Operator == OpAnd {
		// true && expr => expr.
		if lit, isLit := node.Left.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && value {
				return node.Right
			}
		}
		// expr && true => expr.
		if lit, isLit := node.Right.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && value {
				return node.Left
			}
		}
		// false && expr => false.
		if lit, isLit := node.Left.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && !value {
				return lit
			}
		}
		// expr && false => false.
		if lit, isLit := node.Right.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && !value {
				return lit
			}
		}
	}

	// Simplify logical OR with constant operands.
	if node.Operator == OpOr {
		// false || expr => expr.
		if lit, isLit := node.Left.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && !value {
				return node.Right
			}
		}
		// expr || false => expr.
		if lit, isLit := node.Right.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && !value {
				return node.Left
			}
		}
		// true || expr => true.
		if lit, isLit := node.Left.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && value {
				return lit
			}
		}
		// expr || true => true.
		if lit, isLit := node.Right.(*Literal); isLit && lit.Kind == LiteralBoolean {
			if value, ok := lit.Value.(bool); ok && value {
				return lit
			}
		}
	}

	// Simplify arithmetic with identity elements.
	// expr + 0 => expr, 0 + expr => expr.
	if node.Operator == OpAdd {
		if lit, isLit := node.Left.(*Literal); isLit && ssv.isZero(lit) {
			return node.Right
		}

		if lit, isLit := node.Right.(*Literal); isLit && ssv.isZero(lit) {
			return node.Left
		}
	}

	// expr * 1 => expr, 1 * expr => expr.
	if node.Operator == OpMul {
		if lit, isLit := node.Left.(*Literal); isLit && ssv.isOne(lit) {
			return node.Right
		}

		if lit, isLit := node.Right.(*Literal); isLit && ssv.isOne(lit) {
			return node.Left
		}
	}

	// expr * 0 => 0, 0 * expr => 0.
	if node.Operator == OpMul {
		if lit, isLit := node.Left.(*Literal); isLit && ssv.isZero(lit) {
			return lit
		}

		if lit, isLit := node.Right.(*Literal); isLit && ssv.isZero(lit) {
			return lit
		}
	}

	return nil
}

// isZero checks if a literal represents zero.
func (ssv *syntaxSugarVisitor) isZero(lit *Literal) bool {
	switch lit.Kind {
	case LiteralInteger:
		if value, ok := lit.Value.(int); ok {
			return value == 0
		}

		if value, ok := lit.Value.(int64); ok {
			return value == 0
		}
	case LiteralFloat:
		if value, ok := lit.Value.(float64); ok {
			return value == 0.0
		}
	}

	return false
}

// isOne checks if a literal represents one.
func (ssv *syntaxSugarVisitor) isOne(lit *Literal) bool {
	switch lit.Kind {
	case LiteralInteger:
		if value, ok := lit.Value.(int); ok {
			return value == 1
		}

		if value, ok := lit.Value.(int64); ok {
			return value == 1
		}
	case LiteralFloat:
		if value, ok := lit.Value.(float64); ok {
			return value == 1.0
		}
	}

	return false
}

// Implement visitor methods for other node types.
func (ssv *syntaxSugarVisitor) VisitProgram(node *Program) interface{} {
	ssv.stats.NodesVisited++
	for i, decl := range node.Declarations {
		if result := decl.Accept(ssv); result != nil {
			if newDecl, ok := result.(Declaration); ok {
				node.Declarations[i] = newDecl
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitComment(node *Comment) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} {
	ssv.stats.NodesVisited++
	if node.Body != nil {
		if result := node.Body.Accept(ssv); result != nil {
			if newBody, ok := result.(*BlockStatement); ok {
				node.Body = newBody
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitParameter(node *Parameter) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} {
	ssv.stats.NodesVisited++
	if node.Value != nil {
		if result := node.Value.Accept(ssv); result != nil {
			if newValue, ok := result.(Expression); ok {
				node.Value = newValue
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitBlockStatement(node *BlockStatement) interface{} {
	ssv.stats.NodesVisited++
	for i, stmt := range node.Statements {
		if result := stmt.Accept(ssv); result != nil {
			if newStmt, ok := result.(Statement); ok {
				node.Statements[i] = newStmt
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} {
	ssv.stats.NodesVisited++
	if result := node.Expression.Accept(ssv); result != nil {
		if newExpr, ok := result.(Expression); ok {
			node.Expression = newExpr
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitReturnStatement(node *ReturnStatement) interface{} {
	ssv.stats.NodesVisited++
	if node.Value != nil {
		if result := node.Value.Accept(ssv); result != nil {
			if newValue, ok := result.(Expression); ok {
				node.Value = newValue
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitIfStatement(node *IfStatement) interface{} {
	ssv.stats.NodesVisited++

	if result := node.Condition.Accept(ssv); result != nil {
		if newCond, ok := result.(Expression); ok {
			node.Condition = newCond
		}
	}

	if result := node.ThenBlock.Accept(ssv); result != nil {
		if newThen, ok := result.(*BlockStatement); ok {
			node.ThenBlock = newThen
		}
	}

	if node.ElseBlock != nil {
		if result := node.ElseBlock.Accept(ssv); result != nil {
			if newElse, ok := result.(Statement); ok {
				node.ElseBlock = newElse
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitWhileStatement(node *WhileStatement) interface{} {
	ssv.stats.NodesVisited++

	if result := node.Condition.Accept(ssv); result != nil {
		if newCond, ok := result.(Expression); ok {
			node.Condition = newCond
		}
	}

	if result := node.Body.Accept(ssv); result != nil {
		if newBody, ok := result.(*BlockStatement); ok {
			node.Body = newBody
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitIdentifier(node *Identifier) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitLiteral(node *Literal) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitUnaryExpression(node *UnaryExpression) interface{} {
	ssv.stats.NodesVisited++

	if result := node.Operand.Accept(ssv); result != nil {
		if newOperand, ok := result.(Expression); ok {
			node.Operand = newOperand
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitCallExpression(node *CallExpression) interface{} {
	ssv.stats.NodesVisited++

	if result := node.Function.Accept(ssv); result != nil {
		if newFunc, ok := result.(Expression); ok {
			node.Function = newFunc
		}
	}

	for i, arg := range node.Arguments {
		if result := arg.Accept(ssv); result != nil {
			if newArg, ok := result.(Expression); ok {
				node.Arguments[i] = newArg
			}
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitMemberExpression(node *MemberExpression) interface{} {
	ssv.stats.NodesVisited++

	if result := node.Object.Accept(ssv); result != nil {
		if newObj, ok := result.(Expression); ok {
			node.Object = newObj
		}
	}

	return node
}

func (ssv *syntaxSugarVisitor) VisitBasicType(node *BasicType) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitIdentifierType(node *IdentifierType) interface{} {
	ssv.stats.NodesVisited++

	return node
}

func (ssv *syntaxSugarVisitor) VisitAttribute(node *Attribute) interface{} {
	ssv.stats.NodesVisited++

	return node
}

// ===== Optimization Pipeline Factory Functions =====.

// CreateStandardOptimizationPipeline creates a pipeline with standard optimization passes.
func CreateStandardOptimizationPipeline() *OptimizationPipeline {
	pipeline := NewOptimizationPipeline()

	// Add passes in optimal order.
	pipeline.AddPass(NewConstantFoldingPass())
	pipeline.AddPass(NewSyntaxSugarRemovalPass())
	pipeline.AddPass(NewDeadCodeEliminationPass())

	return pipeline
}

// CreateAggressiveOptimizationPipeline creates a pipeline with aggressive optimizations.
func CreateAggressiveOptimizationPipeline() *OptimizationPipeline {
	pipeline := NewOptimizationPipeline()
	pipeline.SetOptimizationLevel(OptimizationAggressive)

	// Add multiple rounds of optimization.
	constantFolding := NewConstantFoldingPass()
	syntaxSugar := NewSyntaxSugarRemovalPass()
	deadCode := NewDeadCodeEliminationPass()

	// First round: basic cleanup.
	pipeline.AddPass(constantFolding)
	pipeline.AddPass(syntaxSugar)

	// Second round: dead code elimination after simplification.
	pipeline.AddPass(deadCode)

	// Third round: final constant folding after dead code removal.
	pipeline.AddPass(NewConstantFoldingPass())

	return pipeline
}

// CreateBasicOptimizationPipeline creates a pipeline with only safe optimizations.
func CreateBasicOptimizationPipeline() *OptimizationPipeline {
	pipeline := NewOptimizationPipeline()
	pipeline.SetOptimizationLevel(OptimizationBasic)

	// Only add conservative optimizations.
	pipeline.AddPass(NewConstantFoldingPass())

	return pipeline
}
