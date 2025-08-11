// Algorithm W implementation for Hindley-Milner type inference
// This file implements the core type inference algorithm

package types

import (
	"fmt"
)

// ====== Abstract Syntax Tree for Type Inference ======

// Expr represents an expression in the abstract syntax tree
type Expr interface {
	String() string
	Accept(visitor ExprVisitor) (*Type, error)
}

// ExprVisitor defines the visitor pattern for expression traversal
type ExprVisitor interface {
	VisitLiteral(expr *LiteralExpr) (*Type, error)
	VisitVariable(expr *VariableExpr) (*Type, error)
	VisitApplication(expr *ApplicationExpr) (*Type, error)
	VisitLambda(expr *LambdaExpr) (*Type, error)
	VisitLet(expr *LetExpr) (*Type, error)
	VisitIfElse(expr *IfElseExpr) (*Type, error)
	VisitBinaryOp(expr *BinaryOpExpr) (*Type, error)
	VisitUnaryOp(expr *UnaryOpExpr) (*Type, error)
}

// ====== Expression Types ======

// LiteralExpr represents literal values
type LiteralExpr struct {
	Value interface{}
	Type  *Type
}

func (e *LiteralExpr) String() string {
	return fmt.Sprintf("%v", e.Value)
}

func (e *LiteralExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitLiteral(e)
}

// VariableExpr represents variable references
type VariableExpr struct {
	Name string
}

func (e *VariableExpr) String() string {
	return e.Name
}

func (e *VariableExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitVariable(e)
}

// ApplicationExpr represents function application
type ApplicationExpr struct {
	Function Expr
	Argument Expr
}

func (e *ApplicationExpr) String() string {
	return fmt.Sprintf("(%s %s)", e.Function.String(), e.Argument.String())
}

func (e *ApplicationExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitApplication(e)
}

// LambdaExpr represents lambda abstraction
type LambdaExpr struct {
	Parameter string
	Body      Expr
	ParamType *Type // Optional type annotation
}

func (e *LambdaExpr) String() string {
	if e.ParamType != nil {
		return fmt.Sprintf("(λ%s:%s.%s)", e.Parameter, e.ParamType.String(), e.Body.String())
	}
	return fmt.Sprintf("(λ%s.%s)", e.Parameter, e.Body.String())
}

func (e *LambdaExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitLambda(e)
}

// LetExpr represents let-in expressions
type LetExpr struct {
	Variable   string
	Definition Expr
	Body       Expr
	VarType    *Type // Optional type annotation
}

func (e *LetExpr) String() string {
	if e.VarType != nil {
		return fmt.Sprintf("(let %s:%s = %s in %s)", e.Variable, e.VarType.String(), e.Definition.String(), e.Body.String())
	}
	return fmt.Sprintf("(let %s = %s in %s)", e.Variable, e.Definition.String(), e.Body.String())
}

func (e *LetExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitLet(e)
}

// IfElseExpr represents conditional expressions
type IfElseExpr struct {
	Condition Expr
	ThenExpr  Expr
	ElseExpr  Expr
}

func (e *IfElseExpr) String() string {
	return fmt.Sprintf("(if %s then %s else %s)", e.Condition.String(), e.ThenExpr.String(), e.ElseExpr.String())
}

func (e *IfElseExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitIfElse(e)
}

// BinaryOpExpr represents binary operations
type BinaryOpExpr struct {
	Left     Expr
	Operator string
	Right    Expr
}

func (e *BinaryOpExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Operator, e.Right.String())
}

func (e *BinaryOpExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitBinaryOp(e)
}

// UnaryOpExpr represents unary operations
type UnaryOpExpr struct {
	Operator string
	Operand  Expr
}

func (e *UnaryOpExpr) String() string {
	return fmt.Sprintf("(%s %s)", e.Operator, e.Operand.String())
}

func (e *UnaryOpExpr) Accept(visitor ExprVisitor) (*Type, error) {
	return visitor.VisitUnaryOp(e)
}

// ====== Algorithm W Implementation ======

// TypeInferenceVisitor implements Algorithm W
type TypeInferenceVisitor struct {
	engine *InferenceEngine
}

// NewTypeInferenceVisitor creates a new type inference visitor
func NewTypeInferenceVisitor(engine *InferenceEngine) *TypeInferenceVisitor {
	return &TypeInferenceVisitor{engine: engine}
}

// InferType performs type inference on an expression using Algorithm W
func (ie *InferenceEngine) InferType(expr Expr) (*Type, error) {
	ie.ClearErrors()
	ie.substitutions = make(map[string]*Type)

	visitor := NewTypeInferenceVisitor(ie)
	inferredType, err := expr.Accept(visitor)

	if err != nil {
		return nil, err
	}

	// Apply final substitutions
	finalType := ie.ApplySubstitutions(inferredType)

	if ie.config.VerboseMode {
		fmt.Printf("Inferred type for %s: %s\n", expr.String(), finalType.String())
		if len(ie.substitutions) > 0 {
			fmt.Printf("Substitutions: %v\n", ie.substitutions)
		}
	}

	return finalType, nil
}

// ====== Visitor Implementation ======

// VisitLiteral infers types for literal expressions
func (v *TypeInferenceVisitor) VisitLiteral(expr *LiteralExpr) (*Type, error) {
	if expr.Type != nil {
		return expr.Type, nil
	}

	// Infer type from literal value
	switch val := expr.Value.(type) {
	case bool:
		return TypeBool, nil
	case int:
		return TypeInt32, nil
	case int64:
		return TypeInt64, nil
	case float32:
		return TypeFloat32, nil
	case float64:
		return TypeFloat64, nil
	case string:
		return TypeString, nil
	default:
		// Check if it's int32 or rune (both are int32 in Go)
		if _, ok := val.(int32); ok {
			return TypeInt32, nil
		}
		return nil, fmt.Errorf("unknown literal type: %T (value: %v)", expr.Value, val)
	}
}

// VisitVariable infers types for variable expressions
func (v *TypeInferenceVisitor) VisitVariable(expr *VariableExpr) (*Type, error) {
	scheme, exists := v.engine.LookupVariable(expr.Name)
	if !exists {
		return nil, fmt.Errorf("undefined variable: %s", expr.Name)
	}

	// Instantiate the type scheme
	return v.engine.Instantiate(scheme), nil
}

// VisitApplication infers types for function application
func (v *TypeInferenceVisitor) VisitApplication(expr *ApplicationExpr) (*Type, error) {
	// Infer function type
	funcType, err := expr.Function.Accept(v)
	if err != nil {
		return nil, err
	}

	// Infer argument type
	argType, err := expr.Argument.Accept(v)
	if err != nil {
		return nil, err
	}

	// Create fresh type variable for result
	resultType := v.engine.FreshTypeVar()

	// Create function type: argType -> resultType
	expectedFuncType := NewFunctionType([]*Type{argType}, resultType, false, false)

	// Unify function type with expected type
	if err := v.engine.Unify(funcType, expectedFuncType); err != nil {
		return nil, fmt.Errorf("function application type error: %v", err)
	}

	// Return the result type (with substitutions applied)
	return v.engine.ApplySubstitutions(resultType), nil
}

// VisitLambda infers types for lambda expressions
func (v *TypeInferenceVisitor) VisitLambda(expr *LambdaExpr) (*Type, error) {
	// Create type for parameter
	var paramType *Type
	if expr.ParamType != nil {
		paramType = expr.ParamType
	} else {
		paramType = v.engine.FreshTypeVar()
	}

	// Create type scheme for parameter (monomorphic)
	paramScheme := &TypeScheme{
		TypeVars: []string{},
		Type:     paramType,
		Level:    v.engine.currentEnv.Level,
	}

	// Push new environment and add parameter
	v.engine.PushEnvironment()
	defer v.engine.PopEnvironment()

	v.engine.AddVariable(expr.Parameter, paramScheme)

	// Infer body type
	bodyType, err := expr.Body.Accept(v)
	if err != nil {
		return nil, err
	}

	// Create function type
	funcType := NewFunctionType([]*Type{paramType}, bodyType, false, false)

	return v.engine.ApplySubstitutions(funcType), nil
}

// VisitLet infers types for let-in expressions (with let-polymorphism)
func (v *TypeInferenceVisitor) VisitLet(expr *LetExpr) (*Type, error) {
	// Infer type of definition
	defType, err := expr.Definition.Accept(v)
	if err != nil {
		return nil, err
	}

	// Apply current substitutions
	defType = v.engine.ApplySubstitutions(defType)

	// Handle type annotation if present
	if expr.VarType != nil {
		if err := v.engine.Unify(defType, expr.VarType); err != nil {
			return nil, fmt.Errorf("type annotation mismatch for %s: %v", expr.Variable, err)
		}
		defType = v.engine.ApplySubstitutions(defType)
	}

	// Generalize the type (let-polymorphism)
	scheme := v.engine.Generalize(defType)

	// Push new environment and add variable
	v.engine.PushEnvironment()
	defer v.engine.PopEnvironment()

	v.engine.AddVariable(expr.Variable, scheme)

	// Infer body type
	bodyType, err := expr.Body.Accept(v)
	if err != nil {
		return nil, err
	}

	return v.engine.ApplySubstitutions(bodyType), nil
}

// VisitIfElse infers types for conditional expressions
func (v *TypeInferenceVisitor) VisitIfElse(expr *IfElseExpr) (*Type, error) {
	// Infer condition type
	condType, err := expr.Condition.Accept(v)
	if err != nil {
		return nil, err
	}

	// Condition must be boolean
	if err := v.engine.Unify(condType, TypeBool); err != nil {
		return nil, fmt.Errorf("condition must be boolean: %v", err)
	}

	// Infer then and else types
	thenType, err := expr.ThenExpr.Accept(v)
	if err != nil {
		return nil, err
	}

	elseType, err := expr.ElseExpr.Accept(v)
	if err != nil {
		return nil, err
	}

	// Then and else branches must have the same type
	if err := v.engine.Unify(thenType, elseType); err != nil {
		return nil, fmt.Errorf("if-else branches must have same type: %v", err)
	}

	return v.engine.ApplySubstitutions(thenType), nil
}

// VisitBinaryOp infers types for binary operations
func (v *TypeInferenceVisitor) VisitBinaryOp(expr *BinaryOpExpr) (*Type, error) {
	// Look up operator type
	opScheme, exists := v.engine.LookupVariable(expr.Operator)
	if !exists {
		return nil, fmt.Errorf("undefined operator: %s", expr.Operator)
	}

	// Instantiate operator type
	opType := v.engine.Instantiate(opScheme)

	// Infer operand types
	leftType, err := expr.Left.Accept(v)
	if err != nil {
		return nil, err
	}

	rightType, err := expr.Right.Accept(v)
	if err != nil {
		return nil, err
	}

	// Create fresh type variable for result
	resultType := v.engine.FreshTypeVar()

	// Create expected operator type: leftType -> rightType -> resultType
	expectedOpType := NewFunctionType([]*Type{leftType, rightType}, resultType, false, false)

	// Unify operator type with expected type
	if err := v.engine.Unify(opType, expectedOpType); err != nil {
		return nil, fmt.Errorf("binary operator type error for %s: %v", expr.Operator, err)
	}

	return v.engine.ApplySubstitutions(resultType), nil
}

// VisitUnaryOp infers types for unary operations
func (v *TypeInferenceVisitor) VisitUnaryOp(expr *UnaryOpExpr) (*Type, error) {
	// Look up operator type
	opScheme, exists := v.engine.LookupVariable(expr.Operator)
	if !exists {
		return nil, fmt.Errorf("undefined unary operator: %s", expr.Operator)
	}

	// Instantiate operator type
	opType := v.engine.Instantiate(opScheme)

	// Infer operand type
	operandType, err := expr.Operand.Accept(v)
	if err != nil {
		return nil, err
	}

	// Create fresh type variable for result
	resultType := v.engine.FreshTypeVar()

	// Create expected operator type: operandType -> resultType
	expectedOpType := NewFunctionType([]*Type{operandType}, resultType, false, false)

	// Unify operator type with expected type
	if err := v.engine.Unify(opType, expectedOpType); err != nil {
		return nil, fmt.Errorf("unary operator type error for %s: %v", expr.Operator, err)
	}

	return v.engine.ApplySubstitutions(resultType), nil
}

// ====== Expression Constructors ======

// NewLiteralExpr creates a new literal expression
func NewLiteralExpr(value interface{}) *LiteralExpr {
	return &LiteralExpr{Value: value, Type: nil}
}

// NewTypedLiteralExpr creates a new literal expression with explicit type
func NewTypedLiteralExpr(value interface{}, t *Type) *LiteralExpr {
	return &LiteralExpr{Value: value, Type: t}
}

// NewVariableExpr creates a new variable expression
func NewVariableExpr(name string) *VariableExpr {
	return &VariableExpr{Name: name}
}

// NewApplicationExpr creates a new function application expression
func NewApplicationExpr(function, argument Expr) *ApplicationExpr {
	return &ApplicationExpr{Function: function, Argument: argument}
}

// NewLambdaExpr creates a new lambda expression
func NewLambdaExpr(parameter string, body Expr) *LambdaExpr {
	return &LambdaExpr{Parameter: parameter, Body: body, ParamType: nil}
}

// NewTypedLambdaExpr creates a new lambda expression with type annotation
func NewTypedLambdaExpr(parameter string, paramType *Type, body Expr) *LambdaExpr {
	return &LambdaExpr{Parameter: parameter, Body: body, ParamType: paramType}
}

// NewLetExpr creates a new let-in expression
func NewLetExpr(variable string, definition, body Expr) *LetExpr {
	return &LetExpr{Variable: variable, Definition: definition, Body: body, VarType: nil}
}

// NewTypedLetExpr creates a new let-in expression with type annotation
func NewTypedLetExpr(variable string, varType *Type, definition, body Expr) *LetExpr {
	return &LetExpr{Variable: variable, Definition: definition, Body: body, VarType: varType}
}

// NewIfElseExpr creates a new conditional expression
func NewIfElseExpr(condition, thenExpr, elseExpr Expr) *IfElseExpr {
	return &IfElseExpr{Condition: condition, ThenExpr: thenExpr, ElseExpr: elseExpr}
}

// NewBinaryOpExpr creates a new binary operation expression
func NewBinaryOpExpr(left Expr, operator string, right Expr) *BinaryOpExpr {
	return &BinaryOpExpr{Left: left, Operator: operator, Right: right}
}

// NewUnaryOpExpr creates a new unary operation expression
func NewUnaryOpExpr(operator string, operand Expr) *UnaryOpExpr {
	return &UnaryOpExpr{Operator: operator, Operand: operand}
}

// ====== Utility Functions ======

// InferTypeWithContext performs type inference with a custom environment
func (ie *InferenceEngine) InferTypeWithContext(expr Expr, env *TypeEnvironment) (*Type, error) {
	oldEnv := ie.currentEnv
	ie.currentEnv = env
	defer func() { ie.currentEnv = oldEnv }()

	return ie.InferType(expr)
}

// InferTypes performs type inference on multiple expressions
func (ie *InferenceEngine) InferTypes(exprs []Expr) ([]*Type, error) {
	types := make([]*Type, len(exprs))

	for i, expr := range exprs {
		inferredType, err := ie.InferType(expr)
		if err != nil {
			return nil, fmt.Errorf("expression %d: %v", i, err)
		}
		types[i] = inferredType
	}

	return types, nil
}

// Reset resets the inference engine state
func (ie *InferenceEngine) Reset() {
	ie.nextTypeVarId = 0
	ie.substitutions = make(map[string]*Type)
	ie.constraints = []Constraint{}
	ie.currentEnv = ie.globalEnv
	ie.ClearErrors()
}

// GetSubstitutions returns the current substitution map
func (ie *InferenceEngine) GetSubstitutions() map[string]*Type {
	result := make(map[string]*Type)
	for k, v := range ie.substitutions {
		result[k] = v
	}
	return result
}
