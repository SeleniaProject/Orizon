// AST optimization passes for Orizon language.
// This file implements compile-time optimizations at the syntax tree level:.
// 1. Constant folding - evaluate constant expressions at compile time
// 2. Dead code detection - identify unreachable code segments
// 3. Syntax sugar removal - desugar high-level constructs to basic forms

package parser

import (
	"fmt"
	"strconv"
)

// ====== Optimization Pass Infrastructure ======.

// OptimizationPass represents a single optimization transformation.
type OptimizationPass interface {
	// GetName returns the name of this optimization pass.
	GetName() string

	// Apply applies the optimization to an AST node.
	Apply(node Node) (Node, bool) // returns (optimized_node, changed)

	// GetMetrics returns optimization statistics.
	GetMetrics() OptimizationMetrics
}

// OptimizationMetrics tracks the effectiveness of optimizations.
type OptimizationMetrics struct {
	PassName           string
	NodesProcessed     int
	NodesOptimized     int
	ConstantsFolded    int
	DeadCodeRemoved    int
	SyntaxSugarRemoved int
	EstimatedSpeedup   float64 // percentage improvement estimate
}

// OptimizationContext provides context for optimization passes.
type OptimizationContext struct {
	ConstantValues map[string]interface{} // known constant values
	DeadCodePaths  []Span                 // identified dead code locations
	OptLevel       int                    // optimization level (0-3)
	DebugMode      bool                   // preserve debug information
}

// OptimizationEngine orchestrates multiple optimization passes.
type OptimizationEngine struct {
	context *OptimizationContext
	passes  []OptimizationPass
	metrics []OptimizationMetrics
}

// NewOptimizationEngine creates a new optimization engine.
func NewOptimizationEngine(optLevel int) *OptimizationEngine {
	engine := &OptimizationEngine{
		context: &OptimizationContext{
			ConstantValues: make(map[string]interface{}),
			DeadCodePaths:  make([]Span, 0),
			OptLevel:       optLevel,
			DebugMode:      optLevel == 0,
		},
		passes:  make([]OptimizationPass, 0),
		metrics: make([]OptimizationMetrics, 0),
	}

	// Register optimization passes based on level.
	if optLevel >= 1 {
		engine.RegisterPass(NewConstantFoldingPass())
	}

	if optLevel >= 2 {
		engine.RegisterPass(NewDeadCodeDetectionPass())
	}

	if optLevel >= 3 {
		engine.RegisterPass(NewSyntaxSugarRemovalPass())
	}

	return engine
}

// RegisterPass adds an optimization pass to the engine.
func (oe *OptimizationEngine) RegisterPass(pass OptimizationPass) {
	oe.passes = append(oe.passes, pass)
}

// OptimizeProgram applies all optimization passes to a program.
func (oe *OptimizationEngine) OptimizeProgram(program *Program) (*Program, []OptimizationMetrics) {
	optimizedProgram := program

	var changed bool

	// Apply each pass until no changes occur.
	for iteration := 0; iteration < 10; iteration++ { // max 10 iterations
		totalChanges := false

		for _, pass := range oe.passes {
			optimizedProgram, changed = oe.applyPassToProgram(pass, optimizedProgram)
			if changed {
				totalChanges = true
			}

			oe.metrics = append(oe.metrics, pass.GetMetrics())
		}

		if !totalChanges {
			break // convergence reached
		}
	}

	return optimizedProgram, oe.metrics
}

// applyPassToProgram applies a single pass to a program.
func (oe *OptimizationEngine) applyPassToProgram(pass OptimizationPass, program *Program) (*Program, bool) {
	optimizer := &ASTOptimizer{
		pass:    pass,
		context: oe.context,
		changed: false,
	}

	result := program.Accept(optimizer)

	return result.(*Program), optimizer.changed
}

// ====== AST Optimizer Visitor ======.

// ASTOptimizer implements the Visitor pattern for AST optimization.
type ASTOptimizer struct {
	pass        OptimizationPass
	currentPass OptimizationPass
	context     *OptimizationContext
	changed     bool
}

// NewASTOptimizer creates a new AST optimizer with the given context and pass.
func NewASTOptimizer(context *OptimizationContext, pass OptimizationPass) *ASTOptimizer {
	return &ASTOptimizer{
		pass:        pass,
		currentPass: pass,
		context:     context,
		changed:     false,
	}
}

// VisitProgram optimizes a program node.
func (ao *ASTOptimizer) VisitProgram(program *Program) interface{} {
	optimizedDecls := make([]Declaration, 0, len(program.Declarations))

	for _, decl := range program.Declarations {
		if optimizedDecl := ao.optimizeNode(decl); optimizedDecl != nil {
			optimizedDecls = append(optimizedDecls, optimizedDecl.(Declaration))
		}
	}

	return &Program{
		Span:         program.Span,
		Declarations: optimizedDecls,
	}
}

// VisitFunctionDeclaration optimizes a function declaration.
func (ao *ASTOptimizer) VisitFunctionDeclaration(fn *FunctionDeclaration) interface{} {
	// Optimize function body.
	optimizedBody := ao.optimizeNode(fn.Body)

	// Check if function body became empty (dead code elimination).
	if optimizedBody == nil {
		return nil // entire function is dead code
	}

	return &FunctionDeclaration{
		Span:       fn.Span,
		Name:       fn.Name,
		Parameters: fn.Parameters, // parameters don't need optimization
		ReturnType: fn.ReturnType,
		Body:       optimizedBody.(*BlockStatement),
		IsPublic:   fn.IsPublic,
		IsAsync:    fn.IsAsync,
		Generics:   fn.Generics,
	}
}

// VisitStructDeclaration handles struct declarations (no optimization on fields yet).
func (ao *ASTOptimizer) VisitStructDeclaration(sd *StructDeclaration) interface{} {
	// Fields/types are not optimized at AST level here; just return as-is
	return sd
}

// VisitEnumDeclaration handles enum declarations (no optimization on variants yet).
func (ao *ASTOptimizer) VisitEnumDeclaration(ed *EnumDeclaration) interface{} {
	return ed
}

// VisitTraitDeclaration handles trait declarations.
func (ao *ASTOptimizer) VisitTraitDeclaration(td *TraitDeclaration) interface{} {
	// Method signatures don't have bodies to optimize here.
	return td
}

// VisitImplBlock optimizes functions inside impl blocks.
func (ao *ASTOptimizer) VisitImplBlock(ib *ImplBlock) interface{} {
	optimizedItems := make([]*FunctionDeclaration, 0, len(ib.Items))

	for _, fn := range ib.Items {
		if opt := ao.optimizeNode(fn); opt != nil {
			optimizedItems = append(optimizedItems, opt.(*FunctionDeclaration))
		}
	}

	return &ImplBlock{Span: ib.Span, Trait: ib.Trait, ForType: ib.ForType, Items: optimizedItems}
}

// VisitImportDeclaration passes through imports.
func (ao *ASTOptimizer) VisitImportDeclaration(id *ImportDeclaration) interface{} { return id }

// VisitExportDeclaration passes through exports.
func (ao *ASTOptimizer) VisitExportDeclaration(ed *ExportDeclaration) interface{} { return ed }

// VisitVariableDeclaration optimizes a variable declaration.
func (ao *ASTOptimizer) VisitVariableDeclaration(vardecl *VariableDeclaration) interface{} {
	// Optimize initializer if present.
	var optimizedInit Expression

	if vardecl.Initializer != nil {
		if opt := ao.optimizeNode(vardecl.Initializer); opt != nil {
			optimizedInit = opt.(Expression)

			// Register constant value if this is a constant declaration.
			if !vardecl.IsMutable && ao.isConstantExpression(optimizedInit) {
				constValue := ao.evaluateConstant(optimizedInit)
				ao.context.ConstantValues[vardecl.Name.Value] = constValue
			}
		}
	}

	return &VariableDeclaration{
		Span:        vardecl.Span,
		Name:        vardecl.Name,
		TypeSpec:    vardecl.TypeSpec,
		Initializer: optimizedInit,
		IsMutable:   vardecl.IsMutable,
		IsPublic:    vardecl.IsPublic,
	}
}

// VisitBlockStatement optimizes a block statement.
func (ao *ASTOptimizer) VisitBlockStatement(block *BlockStatement) interface{} {
	optimizedStmts := make([]Statement, 0, len(block.Statements))

	for _, stmt := range block.Statements {
		if optimizedStmt := ao.optimizeNode(stmt); optimizedStmt != nil {
			optimizedStmts = append(optimizedStmts, optimizedStmt.(Statement))
		}
	}

	// If block becomes empty, return nil (dead code).
	if len(optimizedStmts) == 0 && len(block.Statements) > 0 {
		ao.changed = true

		return nil
	}

	return &BlockStatement{
		Span:       block.Span,
		Statements: optimizedStmts,
	}
}

// VisitBinaryExpression optimizes binary expressions (constant folding target).
func (ao *ASTOptimizer) VisitBinaryExpression(bin *BinaryExpression) interface{} {
	// Optimize operands first.
	leftOpt := ao.optimizeNode(bin.Left)
	rightOpt := ao.optimizeNode(bin.Right)

	if leftOpt == nil || rightOpt == nil {
		return nil // operand became invalid
	}

	left := leftOpt.(Expression)
	right := rightOpt.(Expression)

	// Apply constant folding if both operands are constants.
	if ao.isConstantExpression(left) && ao.isConstantExpression(right) {
		if folded := ao.foldConstantBinaryExpr(bin.Operator, left, right); folded != nil {
			ao.changed = true

			return folded
		}
	}

	return &BinaryExpression{
		Span:     bin.Span,
		Left:     left,
		Operator: bin.Operator,
		Right:    right,
	}
}

// VisitLiteral handles literal optimization.
func (ao *ASTOptimizer) VisitLiteral(lit *Literal) interface{} {
	// Literals are already optimized.
	return lit
}

// VisitIdentifier handles identifier optimization.
func (ao *ASTOptimizer) VisitIdentifier(id *Identifier) interface{} {
	// Check if identifier refers to a known constant.
	if constValue, exists := ao.context.ConstantValues[id.Value]; exists {
		// Replace identifier with constant literal.
		ao.changed = true

		return ao.createLiteralFromValue(id.Span, constValue)
	}

	return id
}

// VisitReferenceType provides a default passthrough for reference types.
func (ao *ASTOptimizer) VisitReferenceType(rt *ReferenceType) interface{} {
	return rt
}

// VisitPointerType provides a default passthrough for pointer types.
func (ao *ASTOptimizer) VisitPointerType(pt *PointerType) interface{} {
	return pt
}

// VisitCallExpression optimizes function calls.
func (ao *ASTOptimizer) VisitCallExpression(call *CallExpression) interface{} {
	// Optimize function and arguments.
	optimizedFunc := ao.optimizeNode(call.Function)
	optimizedArgs := make([]Expression, 0, len(call.Arguments))

	for _, arg := range call.Arguments {
		if optArg := ao.optimizeNode(arg); optArg != nil {
			optimizedArgs = append(optimizedArgs, optArg.(Expression))
		}
	}

	if optimizedFunc == nil {
		return nil
	}

	return &CallExpression{
		Span:      call.Span,
		Function:  optimizedFunc.(Expression),
		Arguments: optimizedArgs,
	}
}

// VisitIfStatement optimizes conditional statements (dead code detection target).
func (ao *ASTOptimizer) VisitIfStatement(ifStmt *IfStatement) interface{} {
	// Optimize condition.
	optimizedCond := ao.optimizeNode(ifStmt.Condition)
	if optimizedCond == nil {
		return nil
	}

	condition := optimizedCond.(Expression)

	// Check if condition is a constant.
	if ao.isConstantExpression(condition) {
		condValue := ao.evaluateConstant(condition)
		if boolValue, ok := condValue.(bool); ok {
			ao.changed = true
			if boolValue {
				// Condition is always true, return then branch.
				return ao.optimizeNode(ifStmt.ThenStmt)
			} else {
				// Condition is always false, return else branch (if any).
				if ifStmt.ElseStmt != nil {
					return ao.optimizeNode(ifStmt.ElseStmt)
				}

				return nil // entire if statement is dead code
			}
		}
	}

	// Optimize branches.
	optimizedThen := ao.optimizeNode(ifStmt.ThenStmt)

	var optimizedElse Statement

	if ifStmt.ElseStmt != nil {
		if opt := ao.optimizeNode(ifStmt.ElseStmt); opt != nil {
			optimizedElse = opt.(Statement)
		}
	}

	if optimizedThen == nil {
		return optimizedElse // then branch is dead
	}

	return &IfStatement{
		Span:      ifStmt.Span,
		Condition: condition,
		ThenStmt:  optimizedThen.(Statement),
		ElseStmt:  optimizedElse,
	}
}

// optimizeNode applies the current pass to any AST node.
func (ao *ASTOptimizer) optimizeNode(node Node) Node {
	if node == nil {
		return nil
	}

	// First apply pass-specific optimization.
	if optimized, changed := ao.pass.Apply(node); changed {
		ao.changed = true

		if optimized == nil {
			return nil
		}

		node = optimized
	}

	// Then recursively optimize children via visitor pattern.
	if optimized := node.Accept(ao); optimized != nil {
		return optimized.(Node)
	}

	return nil
}

// ====== Utility Methods ======.

// isConstantExpression checks if an expression represents a compile-time constant.
func (ao *ASTOptimizer) isConstantExpression(expr Expression) bool {
	switch e := expr.(type) {
	case *Literal:
		return true
	case *Identifier:
		_, exists := ao.context.ConstantValues[e.Value]

		return exists
	case *BinaryExpression:
		return ao.isConstantExpression(e.Left) && ao.isConstantExpression(e.Right)
	default:
		return false
	}
}

// evaluateConstant evaluates a constant expression to its value.
func (ao *ASTOptimizer) evaluateConstant(expr Expression) interface{} {
	switch e := expr.(type) {
	case *Literal:
		switch e.Kind {
		case LiteralInteger:
			if value, err := strconv.Atoi(e.Value.(string)); err == nil {
				return value
			}
		case LiteralFloat:
			if value, err := strconv.ParseFloat(e.Value.(string), 64); err == nil {
				return value
			}
		case LiteralString:
			return e.Value.(string)
		case LiteralBool:
			return e.Value.(bool)
		}
	case *Identifier:
		if value, exists := ao.context.ConstantValues[e.Value]; exists {
			return value
		}
	}

	return nil
}

// foldConstantBinaryExpr performs constant folding on binary expressions.
func (ao *ASTOptimizer) foldConstantBinaryExpr(op *Operator, left, right Expression) Expression {
	leftVal := ao.evaluateConstant(left)
	rightVal := ao.evaluateConstant(right)

	if leftVal == nil || rightVal == nil {
		return nil
	}

	// Handle integer arithmetic.
	if leftInt, ok1 := leftVal.(int); ok1 {
		if rightInt, ok2 := rightVal.(int); ok2 {
			switch op.Value {
			case "+":
				return ao.createLiteralFromValue(op.Span, leftInt+rightInt)
			case "-":
				return ao.createLiteralFromValue(op.Span, leftInt-rightInt)
			case "*":
				return ao.createLiteralFromValue(op.Span, leftInt*rightInt)
			case "/":
				if rightInt != 0 {
					return ao.createLiteralFromValue(op.Span, leftInt/rightInt)
				}
			case "%":
				if rightInt != 0 {
					return ao.createLiteralFromValue(op.Span, leftInt%rightInt)
				}
			case "==":
				return ao.createLiteralFromValue(op.Span, leftInt == rightInt)
			case "!=":
				return ao.createLiteralFromValue(op.Span, leftInt != rightInt)
			case "<":
				return ao.createLiteralFromValue(op.Span, leftInt < rightInt)
			case ">":
				return ao.createLiteralFromValue(op.Span, leftInt > rightInt)
			case "<=":
				return ao.createLiteralFromValue(op.Span, leftInt <= rightInt)
			case ">=":
				return ao.createLiteralFromValue(op.Span, leftInt >= rightInt)
			}
		}
	}

	// Handle boolean operations.
	if leftBool, ok1 := leftVal.(bool); ok1 {
		if rightBool, ok2 := rightVal.(bool); ok2 {
			switch op.Value {
			case "&&":
				return ao.createLiteralFromValue(op.Span, leftBool && rightBool)
			case "||":
				return ao.createLiteralFromValue(op.Span, leftBool || rightBool)
			case "==":
				return ao.createLiteralFromValue(op.Span, leftBool == rightBool)
			case "!=":
				return ao.createLiteralFromValue(op.Span, leftBool != rightBool)
			}
		}
	}

	// Handle string concatenation.
	if leftStr, ok1 := leftVal.(string); ok1 {
		if rightStr, ok2 := rightVal.(string); ok2 {
			switch op.Value {
			case "+":
				return ao.createLiteralFromValue(op.Span, leftStr+rightStr)
			case "==":
				return ao.createLiteralFromValue(op.Span, leftStr == rightStr)
			case "!=":
				return ao.createLiteralFromValue(op.Span, leftStr != rightStr)
			}
		}
	}

	return nil
}

// createLiteralFromValue creates a literal AST node from a Go value.
func (ao *ASTOptimizer) createLiteralFromValue(span Span, value interface{}) *Literal {
	switch v := value.(type) {
	case int:
		return &Literal{
			Span:  span,
			Kind:  LiteralInteger,
			Value: fmt.Sprintf("%d", v),
		}
	case float64:
		return &Literal{
			Span:  span,
			Kind:  LiteralFloat,
			Value: fmt.Sprintf("%g", v),
		}
	case string:
		return &Literal{
			Span:  span,
			Kind:  LiteralString,
			Value: v,
		}
	case bool:
		return &Literal{
			Span:  span,
			Kind:  LiteralBool,
			Value: v,
		}
	default:
		return nil
	}
}

// Implement remaining visitor methods (delegating to optimizeNode).
func (ao *ASTOptimizer) VisitParameter(param *Parameter) interface{} {
	return param // parameters don't need optimization
}

func (ao *ASTOptimizer) VisitReturnStatement(ret *ReturnStatement) interface{} {
	if ret.Value != nil {
		if optimized := ao.optimizeNode(ret.Value); optimized != nil {
			return &ReturnStatement{
				Span:  ret.Span,
				Value: optimized.(Expression),
			}
		}
	}

	return ret
}

func (ao *ASTOptimizer) VisitExpressionStatement(exprStmt *ExpressionStatement) interface{} {
	if optimized := ao.optimizeNode(exprStmt.Expression); optimized != nil {
		return &ExpressionStatement{
			Span:       exprStmt.Span,
			Expression: optimized.(Expression),
		}
	}

	return nil
}

func (ao *ASTOptimizer) VisitAssignmentExpression(assign *AssignmentExpression) interface{} {
	optimizedTarget := ao.optimizeNode(assign.Left)
	optimizedValue := ao.optimizeNode(assign.Right)

	if optimizedTarget != nil && optimizedValue != nil {
		return &AssignmentExpression{
			Span:     assign.Span,
			Left:     optimizedTarget.(Expression),
			Operator: assign.Operator,
			Right:    optimizedValue.(Expression),
		}
	}

	return nil
}

func (ao *ASTOptimizer) VisitUnaryExpression(unary *UnaryExpression) interface{} {
	if optimized := ao.optimizeNode(unary.Operand); optimized != nil {
		return &UnaryExpression{
			Span:     unary.Span,
			Operator: unary.Operator,
			Operand:  optimized.(Expression),
		}
	}

	return nil
}

func (ao *ASTOptimizer) VisitIndexExpression(idx *IndexExpression) interface{} {
	optimizedObject := ao.optimizeNode(idx.Object)
	optimizedIndex := ao.optimizeNode(idx.Index)

	if optimizedObject != nil && optimizedIndex != nil {
		return &IndexExpression{
			Span:   idx.Span,
			Object: optimizedObject.(Expression),
			Index:  optimizedIndex.(Expression),
		}
	}

	return nil
}

func (ao *ASTOptimizer) VisitMemberExpression(member *MemberExpression) interface{} {
	if optimized := ao.optimizeNode(member.Object); optimized != nil {
		return &MemberExpression{
			Span:     member.Span,
			Object:   optimized.(Expression),
			Member:   member.Member,
			IsMethod: member.IsMethod,
		}
	}

	return nil
}

func (ao *ASTOptimizer) VisitArrayExpression(arr *ArrayExpression) interface{} {
	optimizedElements := make([]Expression, 0, len(arr.Elements))

	for _, elem := range arr.Elements {
		if opt := ao.optimizeNode(elem); opt != nil {
			optimizedElements = append(optimizedElements, opt.(Expression))
		}
	}

	return &ArrayExpression{
		Span:     arr.Span,
		Elements: optimizedElements,
	}
}

// Add missing Visitor methods to complete the interface.
func (ao *ASTOptimizer) VisitStructExpression(se *StructExpression) interface{} {
	// For now, just pass through struct expressions.
	return se
}

func (ao *ASTOptimizer) VisitForStatement(fs *ForStatement) interface{} {
	// For now, just pass through for statements.
	return fs
}

func (ao *ASTOptimizer) VisitForInStatement(fis *ForInStatement) interface{} {
	// For now, just pass through for-in statements.
	return fis
}

func (ao *ASTOptimizer) VisitRangeExpression(re *RangeExpression) interface{} {
	// For now, just pass through range expressions.
	return re
}

func (ao *ASTOptimizer) VisitBreakStatement(bs *BreakStatement) interface{} {
	return bs
}

func (ao *ASTOptimizer) VisitContinueStatement(cs *ContinueStatement) interface{} {
	return cs
}

func (ao *ASTOptimizer) VisitMatchStatement(ms *MatchStatement) interface{} {
	// For now, just pass through match statements.
	return ms
}

func (ao *ASTOptimizer) VisitStructType(st *StructType) interface{} {
	return st
}

func (ao *ASTOptimizer) VisitEnumType(et *EnumType) interface{} {
	return et
}

func (ao *ASTOptimizer) VisitTraitType(tt *TraitType) interface{} {
	return tt
}

func (ao *ASTOptimizer) VisitWhileStatement(ws *WhileStatement) interface{} {
	optimizedCondition := ao.optimizeNode(ws.Condition)
	optimizedBody := ao.optimizeNode(ws.Body)

	if optimizedCondition != nil && optimizedBody != nil {
		// Check for constant false condition (dead loop).
		if ao.isConstantExpression(optimizedCondition.(Expression)) {
			condValue := ao.evaluateConstant(optimizedCondition.(Expression))
			if boolVal, ok := condValue.(bool); ok && !boolVal {
				ao.changed = true

				return nil // Dead loop - condition is always false
			}
		}

		return &WhileStatement{
			Span:      ws.Span,
			Condition: optimizedCondition.(Expression),
			Body:      optimizedBody.(Statement),
		}
	}

	return ws
}

func (ao *ASTOptimizer) VisitTernaryExpression(te *TernaryExpression) interface{} {
	optimizedCondition := ao.optimizeNode(te.Condition)
	optimizedTrueExpr := ao.optimizeNode(te.TrueExpr)
	optimizedFalseExpr := ao.optimizeNode(te.FalseExpr)

	if optimizedCondition != nil && optimizedTrueExpr != nil && optimizedFalseExpr != nil {
		// Check if condition is constant.
		if ao.isConstantExpression(optimizedCondition.(Expression)) {
			condValue := ao.evaluateConstant(optimizedCondition.(Expression))
			if boolVal, ok := condValue.(bool); ok {
				ao.changed = true

				if boolVal {
					return optimizedTrueExpr
				} else {
					return optimizedFalseExpr
				}
			}
		}

		return &TernaryExpression{
			Span:      te.Span,
			Condition: optimizedCondition.(Expression),
			TrueExpr:  optimizedTrueExpr.(Expression),
			FalseExpr: optimizedFalseExpr.(Expression),
		}
	}

	return te
}

func (ao *ASTOptimizer) VisitRefinementTypeExpression(rte *RefinementTypeExpression) interface{} {
	// Optimize the constraint expression.
	optimizedConstraint := ao.optimizeNode(rte.Predicate)
	if optimizedConstraint != nil {
		return &RefinementTypeExpression{
			Span:      rte.Span,
			Variable:  rte.Variable,
			BaseType:  rte.BaseType,
			Predicate: optimizedConstraint.(Expression),
		}
	}

	return rte
}

func (ao *ASTOptimizer) VisitGenericType(gt *GenericType) interface{} {
	return gt
}

func (ao *ASTOptimizer) VisitOperator(op *Operator) interface{} {
	return op
}

func (ao *ASTOptimizer) VisitMacroParameter(mp *MacroParameter) interface{} {
	return mp
}

func (ao *ASTOptimizer) VisitMacroConstraint(mc *MacroConstraint) interface{} {
	return mc
}

func (ao *ASTOptimizer) VisitMacroBody(mb *MacroBody) interface{} {
	return mb
}

func (ao *ASTOptimizer) VisitMacroTemplate(mt *MacroTemplate) interface{} {
	return mt
}

func (ao *ASTOptimizer) VisitMacroPattern(mp *MacroPattern) interface{} {
	return mp
}

func (ao *ASTOptimizer) VisitMacroPatternElement(mpe *MacroPatternElement) interface{} {
	return mpe
}

func (ao *ASTOptimizer) VisitMacroArgument(ma *MacroArgument) interface{} {
	return ma
}

func (ao *ASTOptimizer) VisitMacroContext(mc *MacroContext) interface{} {
	return mc
}

// TODO: Re-enable when dependent types are properly implemented.
/*
func (ao *ASTOptimizer) VisitDependentParameter(dp *DependentParameter) interface{} {
	return dp
}
*/

// Handle type-related visitors (no optimization needed for types).
func (ao *ASTOptimizer) VisitBasicType(bt *BasicType) interface{}       { return bt }
func (ao *ASTOptimizer) VisitArrayType(at *ArrayType) interface{}       { return at }
func (ao *ASTOptimizer) VisitFunctionType(ft *FunctionType) interface{} { return ft }

// Handle macro-related visitors (optimization happens after macro expansion).
func (ao *ASTOptimizer) VisitMacroDefinition(md *MacroDefinition) interface{} { return md }
func (ao *ASTOptimizer) VisitMacroInvocation(mi *MacroInvocation) interface{} { return mi }

// TODO: Re-enable when dependent type system is properly implemented.
/*
// Handle dependent type visitors (pass through for now).
func (ao *ASTOptimizer) VisitDependentFunctionType(dft *DependentFunctionType) interface{} {
	return dft
}
func (ao *ASTOptimizer) VisitRefinementType(rt *RefinementType) interface{}  { return rt }
func (ao *ASTOptimizer) VisitSizedArrayType(sat *SizedArrayType) interface{} { return sat }
func (ao *ASTOptimizer) VisitIndexType(it *IndexType) interface{}            { return it }
func (ao *ASTOptimizer) VisitProofType(pt *ProofType) interface{}            { return pt }
*/

// Pattern matching visitor methods (pass through for now).
func (ao *ASTOptimizer) VisitLiteralPattern(lp *LiteralPattern) interface{}         { return lp }
func (ao *ASTOptimizer) VisitVariablePattern(vp *VariablePattern) interface{}       { return vp }
func (ao *ASTOptimizer) VisitConstructorPattern(cp *ConstructorPattern) interface{} { return cp }
func (ao *ASTOptimizer) VisitGuardPattern(gp *GuardPattern) interface{}             { return gp }
func (ao *ASTOptimizer) VisitWildcardPattern(wp *WildcardPattern) interface{}       { return wp }
func (ao *ASTOptimizer) VisitMatchArm(ma *MatchArm) interface{}                     { return ma }

// Generics and where-clause visitor methods.
func (ao *ASTOptimizer) VisitGenericParameter(gp *GenericParameter) interface{} { return gp }
func (ao *ASTOptimizer) VisitWherePredicate(wp *WherePredicate) interface{}     { return wp }
func (ao *ASTOptimizer) VisitAssociatedType(at *AssociatedType) interface{}     { return at }
