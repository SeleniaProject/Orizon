// Dependent type checking and inference engine for Orizon language
// This file implements type checking, constraint solving, and type inference
// for the dependent type system.

package parser

import (
	"fmt"
	"strings"
)

// ====== Type Environment ======

// TypeEnvironment manages type bindings and scopes for dependent types
type TypeEnvironment struct {
	parent   *TypeEnvironment
	bindings map[string]DependentType
	types    map[string]DependentType
}

// NewTypeEnvironment creates a new type environment
func NewTypeEnvironment(parent *TypeEnvironment) *TypeEnvironment {
	return &TypeEnvironment{
		parent:   parent,
		bindings: make(map[string]DependentType),
		types:    make(map[string]DependentType),
	}
}

// LookupType searches for a type binding in the environment
func (env *TypeEnvironment) LookupType(name string) (DependentType, bool) {
	if typ, exists := env.bindings[name]; exists {
		return typ, true
	}
	if env.parent != nil {
		return env.parent.LookupType(name)
	}
	return nil, false
}

// BindType adds a type binding to the environment
func (env *TypeEnvironment) BindType(name string, typ DependentType) {
	env.bindings[name] = typ
}

// EnterScope creates a new nested environment
func (env *TypeEnvironment) EnterScope() *TypeEnvironment {
	return NewTypeEnvironment(env)
}

// ExitScope returns the parent environment
func (env *TypeEnvironment) ExitScope() *TypeEnvironment {
	return env.parent
}

// ====== Dependent Type Checker ======

// DependentTypeChecker performs type checking for dependent types
type DependentTypeChecker struct {
	environment *TypeEnvironment
	errors      []DependentTypeError
}

// DependentTypeError represents a type error during dependent type checking
type DependentTypeError struct {
	Span    Span
	Message string
	Kind    TypeErrorKind
}

// TypeErrorKind represents different kinds of type errors
type TypeErrorKind int

const (
	TypeErrorMismatch TypeErrorKind = iota
	TypeErrorConstraintViolation
	TypeErrorUnresolvable
	TypeErrorInconsistent
	TypeErrorAmbiguous
)

// String returns a string representation of the type error
func (te DependentTypeError) String() string {
	return fmt.Sprintf("Type error at %s: %s", te.Span.String(), te.Message)
}

// NewDependentTypeChecker creates a new dependent type checker
func NewDependentTypeChecker() *DependentTypeChecker {
	return &DependentTypeChecker{
		environment: NewTypeEnvironment(nil),
		errors:      make([]DependentTypeError, 0),
	}
}

// CheckProgram type checks an entire program
func (tc *DependentTypeChecker) CheckProgram(program *Program) []DependentTypeError {
	tc.errors = make([]DependentTypeError, 0)

	// Check all declarations
	for _, decl := range program.Declarations {
		tc.CheckDeclaration(decl)
	}

	return tc.errors
}

// CheckDeclaration type checks a declaration
func (tc *DependentTypeChecker) CheckDeclaration(decl Declaration) DependentType {
	switch d := decl.(type) {
	case *FunctionDeclaration:
		return tc.CheckFunctionDeclaration(d)
	case *VariableDeclaration:
		return tc.CheckVariableDeclaration(d)
	default:
		tc.addError(d.GetSpan(), "Unknown declaration type")
		return CreateBasicDependentType(d.GetSpan(), "Error")
	}
}

// CheckFunctionDeclaration type checks a function declaration
func (tc *DependentTypeChecker) CheckFunctionDeclaration(fn *FunctionDeclaration) DependentType {
	// Enter new scope for function
	tc.environment = tc.environment.EnterScope()
	defer func() {
		tc.environment = tc.environment.ExitScope()
	}()

	// Check parameter types and add to environment
	paramTypes := make([]DependentType, len(fn.Parameters))
	for i, param := range fn.Parameters {
		paramType := tc.CheckType(param.TypeSpec)
		paramTypes[i] = paramType
		tc.environment.BindType(param.Name.Value, paramType)
		// Validate parameter type constraints that are statically false
		tc.validateConstraints(paramType, param.Span)
	}

	// Check return type
	var returnType DependentType
	if fn.ReturnType != nil {
		returnType = tc.CheckType(fn.ReturnType)
	} else {
		returnType = CreateBasicDependentType(fn.Span, "Unit")
	}
	// Validate return type constraints
	tc.validateConstraints(returnType, fn.Span)

	// Check function body
	if fn.Body != nil {
		bodyType := tc.CheckStatement(fn.Body)

		// Verify return type consistency
		if !returnType.IsEquivalent(bodyType) {
			tc.addError(fn.Span,
				fmt.Sprintf("Function body type %v does not match declared return type %v",
					bodyType, returnType))
		}
	}

	// Create function type
	functionType := &DependentFunctionType{
		Span:       fn.Span,
		Parameters: make([]*DependentParameter, len(fn.Parameters)),
		ReturnType: returnType,
	}

	for i, param := range fn.Parameters {
		functionType.Parameters[i] = &DependentParameter{
			Name: &Identifier{Value: param.Name.Value},
			Type: paramTypes[i],
		}
	} // Bind function name to its type
	tc.environment.BindType(fn.Name.Value, functionType)

	return functionType
}

// CheckVariableDeclaration type checks a variable declaration
func (tc *DependentTypeChecker) CheckVariableDeclaration(vardecl *VariableDeclaration) DependentType {
	var declaredType DependentType

	// Check declared type if present
	if vardecl.TypeSpec != nil {
		declaredType = tc.CheckType(vardecl.TypeSpec)
	}

	// Check initializer if present
	var initType DependentType
	if vardecl.Initializer != nil {
		initType = tc.CheckExpression(vardecl.Initializer)
	}

	// Determine final type
	var finalType DependentType
	if declaredType != nil && initType != nil {
		// Both type and initializer present - check compatibility
		if declaredType.IsEquivalent(initType) {
			finalType = declaredType
		} else {
			tc.addError(vardecl.Span,
				fmt.Sprintf("Variable type %v does not match initializer type %v",
					declaredType, initType))
			finalType = declaredType // Use declared type despite mismatch
		}
	} else if declaredType != nil {
		finalType = declaredType
	} else if initType != nil {
		finalType = initType
	} else {
		tc.addError(vardecl.Span, "Variable declaration must have either type annotation or initializer")
		finalType = CreateBasicDependentType(vardecl.Span, "Error")
	}

	// Validate final type constraints that are statically false
	tc.validateConstraints(finalType, vardecl.Span)

	// Bind variable to its type
	tc.environment.BindType(vardecl.Name.Value, finalType)

	return finalType
}

// validateConstraints reports an error when all information available allows
// a constraint to be determined as false (statically unsatisfiable).
func (tc *DependentTypeChecker) validateConstraints(depType DependentType, span Span) {
	if depType == nil {
		return
	}
	constraints := depType.GetConstraints()
	if len(constraints) == 0 {
		return
	}
	for _, c := range constraints {
		ok, err := evalBool(c.Predicate, map[string]interface{}{})
		if err == nil && !ok {
			tc.errors = append(tc.errors, DependentTypeError{
				Span:    span,
				Message: fmt.Sprintf("Unsatisfiable type constraint: %s", c.Predicate.String()),
				Kind:    TypeErrorConstraintViolation,
			})
		}
	}
}

// CheckStatement type checks a statement
func (tc *DependentTypeChecker) CheckStatement(stmt Statement) DependentType {
	switch s := stmt.(type) {
	case *BlockStatement:
		return tc.CheckBlockStatement(s)
	case *ReturnStatement:
		return tc.CheckReturnStatement(s)
	case *ExpressionStatement:
		return tc.CheckExpression(s.Expression)
	default:
		tc.addError(s.GetSpan(), "Unknown statement type")
		return CreateBasicDependentType(s.GetSpan(), "Error")
	}
}

// CheckBlockStatement type checks a block statement
func (tc *DependentTypeChecker) CheckBlockStatement(block *BlockStatement) DependentType {
	// Enter new scope for block
	tc.environment = tc.environment.EnterScope()
	defer func() {
		tc.environment = tc.environment.ExitScope()
	}()

	var lastType DependentType = CreateBasicDependentType(block.Span, "Unit")

	// Check all statements in block
	for _, stmt := range block.Statements {
		lastType = tc.CheckStatement(stmt)
	}

	return lastType
}

// CheckReturnStatement type checks a return statement
func (tc *DependentTypeChecker) CheckReturnStatement(ret *ReturnStatement) DependentType {
	if ret.Value != nil {
		return tc.CheckExpression(ret.Value)
	}
	return CreateBasicDependentType(ret.Span, "Unit") // Unit type for void returns
}

// CheckExpression type checks an expression
func (tc *DependentTypeChecker) CheckExpression(expr Expression) DependentType {
	switch e := expr.(type) {
	case *Identifier:
		return tc.CheckIdentifier(e)
	case *Literal:
		return tc.CheckLiteral(e)
	case *BinaryExpression:
		return tc.CheckBinaryExpression(e)
	case *CallExpression:
		return tc.CheckCallExpression(e)
	case *IndexExpression:
		return tc.CheckIndexExpression(e)
	default:
		tc.addError(e.GetSpan(), "Unknown expression type")
		return CreateBasicDependentType(e.GetSpan(), "Error")
	}
}

// CheckIdentifier type checks an identifier
func (tc *DependentTypeChecker) CheckIdentifier(id *Identifier) DependentType {
	if typ, exists := tc.environment.LookupType(id.Value); exists {
		return typ
	}

	tc.addError(id.Span, fmt.Sprintf("Undefined identifier: %s", id.Value))
	return CreateBasicDependentType(id.Span, "Error")
}

// CheckLiteral type checks a literal
func (tc *DependentTypeChecker) CheckLiteral(lit *Literal) DependentType {
	switch lit.Kind {
	case LiteralInteger:
		// Create refinement type for integer literals
		var litValue string
		switch v := lit.Value.(type) {
		case string:
			litValue = v
		case int64:
			litValue = fmt.Sprintf("%d", v)
		case int:
			litValue = fmt.Sprintf("%d", v)
		default:
			litValue = fmt.Sprintf("%v", v)
		}
		return &RefinementType{
			Span:     lit.Span,
			BaseType: CreateBasicDependentType(lit.Span, "Int"),
			Variable: &Identifier{Value: "x"},
			Predicate: &BinaryExpression{
				Left:     &Identifier{Value: "x"},
				Operator: &Operator{Value: "=="},
				Right:    &Literal{Kind: LiteralInteger, Value: litValue},
			},
		}
	case LiteralFloat:
		return CreateBasicDependentType(lit.Span, "Float")
	case LiteralString:
		return CreateBasicDependentType(lit.Span, "String")
	case LiteralBool:
		return CreateBasicDependentType(lit.Span, "Bool")
	default:
		tc.addError(lit.Span, "Unknown literal type")
		return CreateBasicDependentType(lit.Span, "Error")
	}
}

// CheckBinaryExpression type checks a binary expression
func (tc *DependentTypeChecker) CheckBinaryExpression(bin *BinaryExpression) DependentType {
	leftType := tc.CheckExpression(bin.Left)
	rightType := tc.CheckExpression(bin.Right)

	// Check operator compatibility and determine result type
	return tc.CheckBinaryOperator(bin.Operator, leftType, rightType)
}

// CheckBinaryOperator determines the result type of a binary operation
func (tc *DependentTypeChecker) CheckBinaryOperator(op *Operator, leftType, rightType DependentType) DependentType {
	switch op.Kind {
	case BinaryOp:
		// Simplified handling for binary operations
		switch op.Value {
		case "+", "-", "*", "/":
			// Arithmetic operations return numeric types
			return CreateBasicDependentType(op.Span, "Int")
		case "==", "!=":
			return CreateBasicDependentType(op.Span, "Bool")
		case "<", ">", "<=", ">=":
			return CreateBasicDependentType(op.Span, "Bool")
		default:
			tc.addError(op.Span, fmt.Sprintf("Unknown binary operator: %s", op.Value))
			return CreateBasicDependentType(op.Span, "Error")
		}
	default:
		tc.addError(op.Span, fmt.Sprintf("Unknown operator kind: %v", op.Kind))
		return CreateBasicDependentType(op.Span, "Error")
	}
}

// CheckCallExpression type checks a function call
func (tc *DependentTypeChecker) CheckCallExpression(call *CallExpression) DependentType {
	// Check function type
	funcType := tc.CheckExpression(call.Function)

	// Check if it's a function type
	if depFunc, ok := funcType.(*DependentFunctionType); ok {
		// Check argument count
		if len(call.Arguments) != len(depFunc.Parameters) {
			tc.addError(call.Span,
				fmt.Sprintf("Function expects %d arguments, got %d",
					len(depFunc.Parameters), len(call.Arguments)))
			return CreateBasicDependentType(call.Span, "Error")
		}

		// Check argument types
		for i, arg := range call.Arguments {
			argType := tc.CheckExpression(arg)
			paramType := depFunc.Parameters[i].Type

			if !argType.IsEquivalent(paramType) {
				tc.addError(arg.GetSpan(),
					fmt.Sprintf("Argument %d type %v does not match parameter type %v",
						i+1, argType, paramType))
			}
		}

		return depFunc.ReturnType
	}

	tc.addError(call.Function.GetSpan(), "Expression is not callable")
	return CreateBasicDependentType(call.Span, "Error")
}

// CheckIndexExpression type checks an array/map index expression
func (tc *DependentTypeChecker) CheckIndexExpression(idx *IndexExpression) DependentType {
	// Get the object being indexed (assuming Object field exists)
	var objectExpr Expression
	if idx.Object != nil {
		objectExpr = idx.Object
	} else {
		tc.addError(idx.Span, "Index expression missing object")
		return CreateBasicDependentType(idx.Span, "Error")
	}

	arrayType := tc.CheckExpression(objectExpr)
	indexType := tc.CheckExpression(idx.Index)

	// Check if it's a sized array type
	if sizedArray, ok := arrayType.(*SizedArrayType); ok {
		// Verify index type is appropriate
		if basicIndex, ok := indexType.(*BasicType); ok && basicIndex.Name == "Int" {
			// Add bounds checking constraint: 0 <= index < array.length
			tc.addBoundsCheckingConstraint(idx, sizedArray)

			// Convert Type to DependentType
			if depElemType, ok := sizedArray.ElementType.(DependentType); ok {
				return depElemType
			} else {
				// Convert BasicType to DependentType
				if basicElem, ok := sizedArray.ElementType.(*BasicType); ok {
					return basicElem // BasicType now implements DependentType
				}
				return CreateBasicDependentType(idx.Span, "Error")
			}
		} else {
			tc.addError(idx.Index.GetSpan(), "Array index must be integer type")
			return CreateBasicDependentType(idx.Span, "Error")
		}
	}

	// Handle basic array types
	if basicType, ok := arrayType.(*BasicType); ok && strings.HasPrefix(basicType.Name, "Array") {
		// Simplified array handling
		return CreateBasicDependentType(idx.Span, "Int") // Simplified element type
	}

	tc.addError(objectExpr.GetSpan(),
		fmt.Sprintf("Type %v is not indexable", arrayType))
	return CreateBasicDependentType(idx.Span, "Error")
}

// CheckType type checks a type annotation
func (tc *DependentTypeChecker) CheckType(typ Type) DependentType {
	switch t := typ.(type) {
	case *BasicType:
		return t // Already implements DependentType
	case *ArrayType:
		elemType := tc.CheckType(t.ElementType)
		// Convert to sized array with unknown size
		return &SizedArrayType{
			Span:        t.Span,
			ElementType: elemType,
			Size:        &Identifier{Value: "_"}, // Unknown size
		}
	default:
		tc.addError(t.GetSpan(), "Unknown type")
		return CreateBasicDependentType(t.GetSpan(), "Error")
	}
}

// addError adds a type error to the checker
func (tc *DependentTypeChecker) addError(span Span, message string) {
	tc.errors = append(tc.errors, DependentTypeError{
		Span:    span,
		Message: message,
		Kind:    TypeErrorMismatch,
	})
}

// ====== Constraint Solver ======

// ConstraintSolver solves type constraints
type ConstraintSolver struct {
	constraints   []TypeConstraint
	substitutions map[string]DependentType
}

// NewConstraintSolver creates a new constraint solver
func NewConstraintSolver() *ConstraintSolver {
	return &ConstraintSolver{
		constraints:   make([]TypeConstraint, 0),
		substitutions: make(map[string]DependentType),
	}
}

// AddConstraint adds a constraint to solve
func (cs *ConstraintSolver) AddConstraint(constraint TypeConstraint) {
	cs.constraints = append(cs.constraints, constraint)
}

// Solve attempts to solve all constraints
func (cs *ConstraintSolver) Solve() (map[string]DependentType, []DependentTypeError) {
	errors := make([]DependentTypeError, 0)

	// Simple constraint solving - can be extended
	for _, constraint := range cs.constraints {
		switch constraint.Kind {
		case ConstraintEquality:
			// Handle equality constraints
			if constraint.Predicate != nil {
				// Extract variable bindings from equality expressions
				cs.solveEqualityConstraint(constraint)
			}
		case ConstraintRange:
			// Handle range constraints
			cs.solveRangeConstraint(constraint)
		case ConstraintPredicate:
			// Handle predicate constraints
			cs.solvePredicateConstraint(constraint)
		}
	}

	return cs.substitutions, errors
}

// solveEqualityConstraint solves equality constraints
func (cs *ConstraintSolver) solveEqualityConstraint(constraint TypeConstraint) {
	// Simplified equality constraint solving
	// In a full implementation, this would use unification
}

// solveRangeConstraint solves range constraints
func (cs *ConstraintSolver) solveRangeConstraint(constraint TypeConstraint) {
	// Handle range constraints like x ∈ [0, 10]
}

// solvePredicateConstraint solves predicate constraints
func (cs *ConstraintSolver) solvePredicateConstraint(constraint TypeConstraint) {
	// Handle predicate constraints like x > 0
}

// ====== Type Inference Engine ======

// TypeInferenceEngine performs type inference
type TypeInferenceEngine struct {
	checker *DependentTypeChecker
	solver  *ConstraintSolver
}

// NewTypeInferenceEngine creates a new type inference engine
func NewTypeInferenceEngine() *TypeInferenceEngine {
	return &TypeInferenceEngine{
		checker: NewDependentTypeChecker(),
		solver:  NewConstraintSolver(),
	}
}

// InferTypes infers types for a program
func (tie *TypeInferenceEngine) InferTypes(program *Program) (map[string]DependentType, []DependentTypeError) {
	// First pass: collect constraints
	errors := tie.checker.CheckProgram(program)

	// Second pass: solve constraints
	substitutions, solverErrors := tie.solver.Solve()
	errors = append(errors, solverErrors...)

	return substitutions, errors
}

// addBoundsCheckingConstraint adds bounds checking constraints for array access
func (tc *DependentTypeChecker) addBoundsCheckingConstraint(idx *IndexExpression, arrayType *SizedArrayType) {
	// Create constraint: 0 <= index < arrayLength
	// For now, this is a simplified implementation that logs the constraint
	// In a full implementation, this would integrate with the constraint solver

	indexSpan := idx.Index.GetSpan()

	// Log the bounds checking requirement
	tc.addError(indexSpan, fmt.Sprintf(
		"Bounds check required: index must be in range [0, size) for array access where size = %s",
		arrayType.Size.String(),
	))

	// In a real implementation, you would:
	// 1. Create symbolic constraints for 0 <= index < arrayType.Size
	// 2. Add these to a constraint solver
	// 3. Verify the constraints can be statically proven or insert runtime checks
	// 4. Generate refinement types with the appropriate bounds information

	// Validate that the size expression exists
	if arrayType.Size == nil {
		tc.addError(indexSpan, "Array size expression is required for bounds checking")
	}
}
