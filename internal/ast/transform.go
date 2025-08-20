// Package ast - AST transformation infrastructure.
// Phase 1.3.1: AST変換インフラ - Comprehensive transformation system for AST manipulation
// This file provides infrastructure for safe, composable AST transformations.
// with support for validation, optimization, and error recovery.
package ast

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/position"
)

// TransformationError represents an error that occurred during AST transformation.
type TransformationError struct {
	Message string
	Span    position.Span
}

// NewTransformationError creates a new transformation error.
func NewTransformationError(message string, span position.Span) *TransformationError {
	return &TransformationError{
		Message: message,
		Span:    span,
	}
}

// Error implements the error interface.
func (te *TransformationError) Error() string {
	return fmt.Sprintf("transformation error at %s: %s", te.Span.String(), te.Message)
}

// Transformer defines the interface for AST transformations.
// Transformers can modify, replace, or remove AST nodes while maintaining tree integrity.
type Transformer interface {
	// Transform applies the transformation to a node and returns the result.
	Transform(node Node) (Node, error)
}

// TransformationPipeline represents a sequence of transformations to apply to an AST.
type TransformationPipeline struct {
	transformers []Transformer
	stopOnError  bool
}

// NewTransformationPipeline creates a new transformation pipeline.
func NewTransformationPipeline() *TransformationPipeline {
	return &TransformationPipeline{
		transformers: make([]Transformer, 0),
		stopOnError:  true,
	}
}

// AddTransformer adds a transformer to the pipeline.
func (tp *TransformationPipeline) AddTransformer(transformer Transformer) {
	tp.transformers = append(tp.transformers, transformer)
}

// Transform applies all transformations in the pipeline to the given node.
func (tp *TransformationPipeline) Transform(node Node) (Node, error) {
	current := node

	var err error

	for _, transformer := range tp.transformers {
		current, err = transformer.Transform(current)
		if err != nil && tp.stopOnError {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
	}

	return current, err
}

// SetStopOnError configures whether the pipeline should stop on first error.
func (tp *TransformationPipeline) SetStopOnError(stop bool) {
	tp.stopOnError = stop
}

// ConstantFoldingTransformer performs constant folding optimization.
type ConstantFoldingTransformer struct{}

// Transform implements the Transformer interface for constant folding.
func (cft *ConstantFoldingTransformer) Transform(node Node) (Node, error) {
	switch n := node.(type) {
	case *BinaryExpression:
		return cft.foldBinaryExpression(n)
	case *UnaryExpression:
		return cft.foldUnaryExpression(n)
	default:
		return node, nil
	}
}

// foldBinaryExpression attempts to fold binary expressions with constant operands.
func (cft *ConstantFoldingTransformer) foldBinaryExpression(expr *BinaryExpression) (Node, error) {
	// Recursively fold operands first.
	left, err := cft.Transform(expr.Left)
	if err != nil {
		return expr, err
	}

	right, err := cft.Transform(expr.Right)
	if err != nil {
		return expr, err
	}

	// Check if both operands are literals.
	leftLit, leftIsLit := left.(*Literal)
	rightLit, rightIsLit := right.(*Literal)

	if !leftIsLit || !rightIsLit {
		// Return expression with potentially folded operands.
		return &BinaryExpression{
			Span:     expr.Span,
			Left:     left.(Expression),
			Operator: expr.Operator,
			Right:    right.(Expression),
		}, nil
	}

	// Perform constant folding based on operator and operand types.
	result, err := cft.evaluateBinaryOperation(leftLit, expr.Operator, rightLit, expr.Span)
	if err != nil {
		// Return original expression if folding fails.
		return expr, nil
	}

	return result, nil
}

// evaluateBinaryOperation evaluates a binary operation on two literals.
func (cft *ConstantFoldingTransformer) evaluateBinaryOperation(left *Literal, op Operator, right *Literal, span position.Span) (*Literal, error) {
	// Integer operations.
	if left.Kind == LiteralInteger && right.Kind == LiteralInteger {
		// Convert to int64 to handle both int and int64.
		var l, r int64
		switch v := left.Value.(type) {
		case int:
			l = int64(v)
		case int64:
			l = v
		default:
			return nil, fmt.Errorf("invalid integer type: %T", left.Value)
		}

		switch v := right.Value.(type) {
		case int:
			r = int64(v)
		case int64:
			r = v
		default:
			return nil, fmt.Errorf("invalid integer type: %T", right.Value)
		}

		switch op {
		case OpAdd:
			return &Literal{Span: span, Kind: LiteralInteger, Value: l + r, Raw: fmt.Sprintf("%d", l+r)}, nil
		case OpSub:
			return &Literal{Span: span, Kind: LiteralInteger, Value: l - r, Raw: fmt.Sprintf("%d", l-r)}, nil
		case OpMul:
			return &Literal{Span: span, Kind: LiteralInteger, Value: l * r, Raw: fmt.Sprintf("%d", l*r)}, nil
		case OpDiv:
			if r == 0 {
				return nil, NewTransformationError("division by zero", span)
			}

			return &Literal{Span: span, Kind: LiteralInteger, Value: l / r, Raw: fmt.Sprintf("%d", l/r)}, nil
		case OpMod:
			if r == 0 {
				return nil, NewTransformationError("modulo by zero", span)
			}

			return &Literal{Span: span, Kind: LiteralInteger, Value: l % r, Raw: fmt.Sprintf("%d", l%r)}, nil
		case OpEq:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l == r, Raw: fmt.Sprintf("%t", l == r)}, nil
		case OpNe:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l != r, Raw: fmt.Sprintf("%t", l != r)}, nil
		case OpLt:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l < r, Raw: fmt.Sprintf("%t", l < r)}, nil
		case OpLe:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l <= r, Raw: fmt.Sprintf("%t", l <= r)}, nil
		case OpGt:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l > r, Raw: fmt.Sprintf("%t", l > r)}, nil
		case OpGe:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l >= r, Raw: fmt.Sprintf("%t", l >= r)}, nil
		}
	}

	// Boolean operations.
	if left.Kind == LiteralBoolean && right.Kind == LiteralBoolean {
		l, r := left.Value.(bool), right.Value.(bool)

		switch op {
		case OpAnd:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l && r, Raw: fmt.Sprintf("%t", l && r)}, nil
		case OpOr:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l || r, Raw: fmt.Sprintf("%t", l || r)}, nil
		case OpEq:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l == r, Raw: fmt.Sprintf("%t", l == r)}, nil
		case OpNe:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l != r, Raw: fmt.Sprintf("%t", l != r)}, nil
		}
	}

	// String operations.
	if left.Kind == LiteralString && right.Kind == LiteralString {
		l, r := left.Value.(string), right.Value.(string)

		switch op {
		case OpAdd: // String concatenation
			result := l + r

			return &Literal{Span: span, Kind: LiteralString, Value: result, Raw: fmt.Sprintf("\"%s\"", result)}, nil
		case OpEq:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l == r, Raw: fmt.Sprintf("%t", l == r)}, nil
		case OpNe:
			return &Literal{Span: span, Kind: LiteralBoolean, Value: l != r, Raw: fmt.Sprintf("%t", l != r)}, nil
		}
	}

	return nil, NewTransformationError("unsupported constant folding operation", span)
}

// foldUnaryExpression attempts to fold unary expressions with constant operands.
func (cft *ConstantFoldingTransformer) foldUnaryExpression(expr *UnaryExpression) (Node, error) {
	// Recursively fold operand first.
	operand, err := cft.Transform(expr.Operand)
	if err != nil {
		return expr, err
	}

	// Check if operand is a literal.
	literal, isLit := operand.(*Literal)
	if !isLit {
		return &UnaryExpression{
			Span:     expr.Span,
			Operator: expr.Operator,
			Operand:  operand.(Expression),
		}, nil
	}

	// Perform constant folding based on operator and operand type.
	switch expr.Operator {
	case OpNot:
		if literal.Kind == LiteralBoolean {
			value := !literal.Value.(bool)

			return &Literal{
				Span:  expr.Span,
				Kind:  LiteralBoolean,
				Value: value,
				Raw:   fmt.Sprintf("%t", value),
			}, nil
		}
	case OpSub: // Unary minus
		if literal.Kind == LiteralInteger {
			value := -literal.Value.(int64)

			return &Literal{
				Span:  expr.Span,
				Kind:  LiteralInteger,
				Value: value,
				Raw:   fmt.Sprintf("%d", value),
			}, nil
		}

		if literal.Kind == LiteralFloat {
			value := -literal.Value.(float64)

			return &Literal{
				Span:  expr.Span,
				Kind:  LiteralFloat,
				Value: value,
				Raw:   fmt.Sprintf("%f", value),
			}, nil
		}
	}

	return expr, nil
}

// DeadCodeEliminationTransformer removes dead code from the AST.
type DeadCodeEliminationTransformer struct{}

// Transform implements the Transformer interface for dead code elimination.
func (dcet *DeadCodeEliminationTransformer) Transform(node Node) (Node, error) {
	switch n := node.(type) {
	case *IfStatement:
		return dcet.eliminateDeadIf(n)
	case *WhileStatement:
		return dcet.eliminateDeadWhile(n)
	case *BlockStatement:
		return dcet.eliminateDeadBlock(n)
	default:
		return node, nil
	}
}

// eliminateDeadIf removes dead branches from if statements.
func (dcet *DeadCodeEliminationTransformer) eliminateDeadIf(ifStmt *IfStatement) (Node, error) {
	// Check if condition is a constant.
	if literal, ok := ifStmt.Condition.(*Literal); ok && literal.Kind == LiteralBoolean {
		if literal.Value.(bool) {
			// Condition is always true, return then block.
			return ifStmt.ThenBlock, nil
		} else {
			// Condition is always false, return else block (or empty block if no else).
			if ifStmt.ElseBlock != nil {
				return ifStmt.ElseBlock, nil
			}

			return &BlockStatement{Span: ifStmt.Span, Statements: []Statement{}}, nil
		}
	}

	return ifStmt, nil
}

// eliminateDeadWhile removes dead while loops.
func (dcet *DeadCodeEliminationTransformer) eliminateDeadWhile(whileStmt *WhileStatement) (Node, error) {
	// Check if condition is a constant false.
	if literal, ok := whileStmt.Condition.(*Literal); ok && literal.Kind == LiteralBoolean {
		if !literal.Value.(bool) {
			// Condition is always false, remove the loop.
			return &BlockStatement{Span: whileStmt.Span, Statements: []Statement{}}, nil
		}
	}

	return whileStmt, nil
}

// eliminateDeadBlock removes unreachable statements after return/break/continue.
func (dcet *DeadCodeEliminationTransformer) eliminateDeadBlock(block *BlockStatement) (Node, error) {
	if len(block.Statements) == 0 {
		return block, nil
	}

	// Find first return statement and remove everything after it.
	newStatements := make([]Statement, 0, len(block.Statements))
	for _, stmt := range block.Statements {
		newStatements = append(newStatements, stmt)

		if _, isReturn := stmt.(*ReturnStatement); isReturn {
			break // Stop processing after return
		}
	}

	return &BlockStatement{
		Span:       block.Span,
		Statements: newStatements,
	}, nil
}

// ValidatorTransformer validates AST structure and reports errors.
type ValidatorTransformer struct{}

// Transform implements the Transformer interface for AST validation.
func (vt *ValidatorTransformer) Transform(node Node) (Node, error) {
	switch n := node.(type) {
	case *Program:
		return vt.validateProgram(n)
	case *FunctionDeclaration:
		return vt.validateFunctionDeclaration(n)
	case *VariableDeclaration:
		return vt.validateVariableDeclaration(n)
	default:
		return node, nil
	}
}

// validateProgram validates a program node.
func (vt *ValidatorTransformer) validateProgram(program *Program) (Node, error) {
	if program.Declarations == nil {
		return program, NewTransformationError("program declarations cannot be nil", program.Span)
	}

	return program, nil
}

// validateFunctionDeclaration validates a function declaration.
func (vt *ValidatorTransformer) validateFunctionDeclaration(funcDecl *FunctionDeclaration) (Node, error) {
	if funcDecl.Name == nil {
		return funcDecl, NewTransformationError("function declaration must have a name", funcDecl.Span)
	}

	if funcDecl.Name.Value == "" {
		return funcDecl, NewTransformationError("function name cannot be empty", funcDecl.Span)
	}

	if funcDecl.Body == nil {
		return funcDecl, NewTransformationError("function declaration must have a body", funcDecl.Span)
	}

	return funcDecl, nil
}

// validateVariableDeclaration validates a variable declaration.
func (vt *ValidatorTransformer) validateVariableDeclaration(varDecl *VariableDeclaration) (Node, error) {
	if varDecl.Name == nil {
		return varDecl, NewTransformationError("variable declaration must have a name", varDecl.Span)
	}

	if varDecl.Name.Value == "" {
		return varDecl, NewTransformationError("variable name cannot be empty", varDecl.Span)
	}

	return varDecl, nil
}

// ASTBuilder provides a fluent interface for building AST nodes.
type ASTBuilder struct {
	defaultSpan position.Span
}

// NewASTBuilder creates a new AST builder with zero span.
func NewASTBuilder() *ASTBuilder {
	return &ASTBuilder{
		defaultSpan: position.Span{},
	}
}

// NewASTBuilderWithSpan creates a new AST builder with the specified default span.
func NewASTBuilderWithSpan(span position.Span) *ASTBuilder {
	return &ASTBuilder{
		defaultSpan: span,
	}
}

// ProgramBuilder provides fluent interface for building programs.
type ProgramBuilder struct {
	builder      *ASTBuilder
	declarations []Declaration
	comments     []Comment
	span         position.Span
}

// Program starts building a program.
func (ab *ASTBuilder) Program() *ProgramBuilder {
	return &ProgramBuilder{
		builder:      ab,
		span:         ab.defaultSpan,
		declarations: make([]Declaration, 0),
		comments:     make([]Comment, 0),
	}
}

// AddFunction adds a function declaration to the program.
func (pb *ProgramBuilder) AddFunction(name string, params []*Parameter, returnType Type) *FunctionBuilder {
	return &FunctionBuilder{
		programBuilder: pb,
		span:           pb.builder.defaultSpan,
		name:           name,
		parameters:     params,
		returnType:     returnType,
		statements:     make([]Statement, 0),
		isExported:     false,
	}
}

// Build creates the program.
func (pb *ProgramBuilder) Build() *Program {
	return &Program{
		Span:         pb.span,
		Declarations: pb.declarations,
		Comments:     pb.comments,
	}
}

// FunctionBuilder provides fluent interface for building functions.
type FunctionBuilder struct {
	returnType     Type
	programBuilder *ProgramBuilder
	name           string
	parameters     []*Parameter
	statements     []Statement
	attributes     []Attribute
	span           position.Span
	isExported     bool
}

// AddStatement adds a statement to the function body.
func (fb *FunctionBuilder) AddStatement(stmt Statement) *FunctionBuilder {
	fb.statements = append(fb.statements, stmt)

	return fb
}

// Build creates the function and adds it to the parent program.
func (fb *FunctionBuilder) Build() *ProgramBuilder {
	function := &FunctionDeclaration{
		Span:       fb.span,
		Name:       &Identifier{Span: fb.span, Value: fb.name},
		Parameters: fb.parameters,
		ReturnType: fb.returnType,
		Body: &BlockStatement{
			Span:       fb.span,
			Statements: fb.statements,
		},
		Attributes: fb.attributes,
		IsExported: fb.isExported,
		Comments:   make([]Comment, 0),
	}

	fb.programBuilder.declarations = append(fb.programBuilder.declarations, function)

	return fb.programBuilder
}

// ExpressionBuilder provides fluent interface for building expressions.
type ExpressionBuilder struct {
	builder *ASTBuilder
	expr    Expression
}

// Binary creates a binary expression.
func (ab *ASTBuilder) Binary(left Expression, op Operator, right Expression) *ExpressionBuilder {
	return &ExpressionBuilder{
		builder: ab,
		expr: &BinaryExpression{
			Span:     ab.defaultSpan,
			Left:     left,
			Operator: op,
			Right:    right,
		},
	}
}

// Call creates a call expression.
func (ab *ASTBuilder) Call(function Expression, args ...Expression) *ExpressionBuilder {
	return &ExpressionBuilder{
		builder: ab,
		expr: &CallExpression{
			Span:      ab.defaultSpan,
			Function:  function,
			Arguments: args,
		},
	}
}

// Identifier creates an identifier expression.
func (ab *ASTBuilder) Identifier(name string) *ExpressionBuilder {
	return &ExpressionBuilder{
		builder: ab,
		expr: &Identifier{
			Span:  ab.defaultSpan,
			Value: name,
		},
	}
}

// Literal creates a literal expression.
func (ab *ASTBuilder) Literal(value interface{}) *ExpressionBuilder {
	var kind LiteralKind

	var raw string

	switch v := value.(type) {
	case int:
		kind = LiteralInteger
		value = int64(v)
		raw = fmt.Sprintf("%d", v)
	case int64:
		kind = LiteralInteger
		raw = fmt.Sprintf("%d", v)
	case float64:
		kind = LiteralFloat
		raw = fmt.Sprintf("%f", v)
	case string:
		kind = LiteralString
		raw = fmt.Sprintf("\"%s\"", v)
	case bool:
		kind = LiteralBoolean
		raw = fmt.Sprintf("%t", v)
	default:
		kind = LiteralNull
		raw = "null"
		value = nil
	}

	return &ExpressionBuilder{
		builder: ab,
		expr: &Literal{
			Span:  ab.defaultSpan,
			Kind:  kind,
			Value: value,
			Raw:   raw,
		},
	}
}

// Build returns the built expression.
func (eb *ExpressionBuilder) Build() Expression {
	return eb.expr
}

// StatementBuilder provides fluent interface for building statements.
type StatementBuilder struct {
	builder *ASTBuilder
	stmt    Statement
}

// Return creates a return statement.
func (ab *ASTBuilder) Return(value Expression) *StatementBuilder {
	return &StatementBuilder{
		builder: ab,
		stmt: &ReturnStatement{
			Span:  ab.defaultSpan,
			Value: value,
		},
	}
}

// Build returns the built statement.
func (sb *StatementBuilder) Build() Statement {
	return sb.stmt
}

// VariableBuilder provides fluent interface for building variable declarations.
type VariableBuilder struct {
	builder *ASTBuilder
	varDecl *VariableDeclaration
}

// Variable creates a variable declaration.
func (ab *ASTBuilder) Variable(name string, varType Type, value Expression, kind VarKind) *VariableBuilder {
	return &VariableBuilder{
		builder: ab,
		varDecl: &VariableDeclaration{
			Span:  ab.defaultSpan,
			Name:  &Identifier{Span: ab.defaultSpan, Value: name},
			Type:  varType,
			Value: value,
			Kind:  kind,
		},
	}
}

// Build returns the built variable declaration.
func (vb *VariableBuilder) Build() *VariableDeclaration {
	return vb.varDecl
}

// TypeBuilder provides fluent interface for building types.
type TypeBuilder struct {
	builder *ASTBuilder
	typ     Type
}

// Type creates a type from string (basic types).
func (ab *ASTBuilder) Type(typeName string) Type {
	var kind BasicKind

	switch typeName {
	case "int":
		kind = BasicInt
	case "float":
		kind = BasicFloat
	case "string":
		kind = BasicString
	case "bool":
		kind = BasicBool
	case "char":
		kind = BasicChar
	case "void":
		kind = BasicVoid
	default:
		// Return identifier type for custom types.
		return &IdentifierType{
			Span: ab.defaultSpan,
			Name: &Identifier{
				Span:  ab.defaultSpan,
				Value: typeName,
			},
		}
	}

	return &BasicType{
		Span: ab.defaultSpan,
		Kind: kind,
	}
}
