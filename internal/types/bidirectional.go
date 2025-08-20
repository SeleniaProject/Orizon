// Package types implements Phase 2.2.3 bidirectional type checking for the Orizon compiler.
// This system combines type synthesis and type checking for more efficient and accurate type inference.
package types

import (
	"errors"
	"fmt"
	"strings"
)

// BinaryExpr represents binary expressions for bidirectional type checking.
type BinaryExpr struct {
	Left     Expr
	Right    Expr
	Operator string
}

func (e *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Operator, e.Right.String())
}

func (e *BinaryExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// This would be implemented by the visitor pattern.
	return nil, fmt.Errorf("binary expression visitor not implemented")
}

// IfExpr represents conditional expressions for bidirectional type checking.
type IfExpr struct {
	Condition  Expr
	ThenBranch Expr
	ElseBranch Expr
}

func (e *IfExpr) String() string {
	return fmt.Sprintf("(if %s then %s else %s)", e.Condition.String(), e.ThenBranch.String(), e.ElseBranch.String())
}

func (e *IfExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// This would be implemented by the visitor pattern.
	return nil, fmt.Errorf("if expression visitor not implemented")
}

// AnnotatedExpr represents expressions with type annotations.
type AnnotatedExpr struct {
	Expression Expr
	Annotation *Type
}

func (e *AnnotatedExpr) String() string {
	return fmt.Sprintf("(%s : %s)", e.Expression.String(), e.Annotation.String())
}

func (e *AnnotatedExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// This would be implemented by the visitor pattern.
	return nil, fmt.Errorf("annotated expression visitor not implemented")
}

// BidirectionalMode represents the mode of bidirectional type checking.
type BidirectionalMode int

const (
	// SynthesisMode generates types from expressions (⇒).
	SynthesisMode BidirectionalMode = iota

	// CheckMode verifies expressions against expected types (⇐).
	CheckMode
)

// String returns a string representation of the bidirectional mode.
func (bm BidirectionalMode) String() string {
	switch bm {
	case SynthesisMode:
		return "synthesis"
	case CheckMode:
		return "check"
	default:
		return "unknown"
	}
}

// TypeCheckingContext represents the context for bidirectional type checking.
type TypeCheckingContext struct {
	expectedType *Type
	environment  *TypeEnvironment
	inference    *InferenceEngine
	errors       []BidirectionalError
	position     SourceLocation
	mode         BidirectionalMode
}

// BidirectionalError represents errors in bidirectional type checking.
type BidirectionalError struct {
	ExpectedType *Type
	ActualType   *Type
	Message      string
	Context      string
	Suggestion   string
	Location     SourceLocation
}

// String returns a string representation of the bidirectional error.
func (be *BidirectionalError) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Type error at %s:%d:%d: %s",
		be.Location.File, be.Location.Line, be.Location.Column, be.Message))

	if be.ExpectedType != nil && be.ActualType != nil {
		sb.WriteString(fmt.Sprintf("\n  Expected: %s", be.ExpectedType.String()))
		sb.WriteString(fmt.Sprintf("\n  Actual: %s", be.ActualType.String()))
	}

	if be.Context != "" {
		sb.WriteString(fmt.Sprintf("\n  Context: %s", be.Context))
	}

	if be.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\n  Suggestion: %s", be.Suggestion))
	}

	return sb.String()
}

// BidirectionalChecker implements the bidirectional type checking algorithm.
type BidirectionalChecker struct {
	inference   *InferenceEngine
	environment *TypeEnvironment
	context     *TypeCheckingContext
	verbose     bool
}

// NewBidirectionalChecker creates a new bidirectional type checker.
func NewBidirectionalChecker(inference *InferenceEngine) *BidirectionalChecker {
	return &BidirectionalChecker{
		inference:   inference,
		environment: inference.currentEnv,
		context: &TypeCheckingContext{
			mode:        SynthesisMode,
			environment: inference.currentEnv,
			inference:   inference,
			errors:      make([]BidirectionalError, 0),
		},
		verbose: false,
	}
}

// SetVerbose enables or disables verbose output.
func (bc *BidirectionalChecker) SetVerbose(verbose bool) {
	bc.verbose = verbose
}

// GetErrors returns all type checking errors.
func (bc *BidirectionalChecker) GetErrors() []BidirectionalError {
	return bc.context.errors
}

// ClearErrors clears all accumulated errors.
func (bc *BidirectionalChecker) ClearErrors() {
	bc.context.errors = make([]BidirectionalError, 0)
}

// CheckExpression performs bidirectional type checking on an expression.
func (bc *BidirectionalChecker) CheckExpression(expr Expr, expectedType *Type) (*Type, error) {
	if bc.verbose {
		fmt.Printf("Checking expression against type: %s\n", expectedType.String())
	}

	// Set check mode with expected type.
	oldMode := bc.context.mode
	oldExpected := bc.context.expectedType

	bc.context.mode = CheckMode
	bc.context.expectedType = expectedType

	defer func() {
		bc.context.mode = oldMode
		bc.context.expectedType = oldExpected
	}()

	return bc.checkExpressionInternal(expr)
}

// SynthesizeType performs type synthesis on an expression.
func (bc *BidirectionalChecker) SynthesizeType(expr Expr) (*Type, error) {
	if bc.verbose {
		fmt.Printf("Synthesizing type for expression\n")
	}

	// Set synthesis mode.
	oldMode := bc.context.mode
	oldExpected := bc.context.expectedType

	bc.context.mode = SynthesisMode
	bc.context.expectedType = nil

	defer func() {
		bc.context.mode = oldMode
		bc.context.expectedType = oldExpected
	}()

	return bc.synthesizeTypeInternal(expr)
}

// checkExpressionInternal implements the checking judgment (Γ ⊢ e ⇐ A).
func (bc *BidirectionalChecker) checkExpressionInternal(expr Expr) (*Type, error) {
	switch e := expr.(type) {
	case *LambdaExpr:
		return bc.checkLambda(e)
	case *AnnotatedExpr:
		return bc.checkAnnotated(e)
	case *IfExpr:
		return bc.checkIf(e)
	default:
		// For other expressions, synthesize and check subsumption.
		synthesizedType, err := bc.synthesizeTypeInternal(expr)
		if err != nil {
			return nil, err
		}

		// Check if synthesized type is compatible with expected type.
		if bc.context.expectedType != nil {
			if err := bc.checkSubsumption(synthesizedType, bc.context.expectedType, expr); err != nil {
				return nil, err
			}
		}

		return synthesizedType, nil
	}
}

// synthesizeTypeInternal implements the synthesis judgment (Γ ⊢ e ⇒ A).
func (bc *BidirectionalChecker) synthesizeTypeInternal(expr Expr) (*Type, error) {
	switch e := expr.(type) {
	case *LiteralExpr:
		return bc.synthesizeLiteral(e)
	case *VariableExpr:
		return bc.synthesizeVariable(e)
	case *ApplicationExpr:
		return bc.synthesizeApplication(e)
	case *AnnotatedExpr:
		return bc.synthesizeAnnotated(e)
	case *BinaryExpr:
		return bc.synthesizeBinary(e)
	default:
		return nil, fmt.Errorf("cannot synthesize type for expression: %T", expr)
	}
}

// checkLambda checks lambda expressions against function types.
func (bc *BidirectionalChecker) checkLambda(lambda *LambdaExpr) (*Type, error) {
	if bc.context.expectedType == nil {
		return nil, fmt.Errorf("cannot check lambda without expected type")
	}

	if bc.context.expectedType.Kind != TypeKindFunction {
		return nil, bc.addError("lambda expression", bc.context.expectedType, nil,
			"Expected function type for lambda expression")
	}

	funcType := bc.context.expectedType.Data.(*FunctionType)

	// For simplicity, handle single parameter lambdas.
	if len(funcType.Parameters) != 1 {
		return nil, bc.addError("lambda parameters", nil, nil,
			fmt.Sprintf("Expected 1 parameter, got function with %d parameters",
				len(funcType.Parameters)))
	}

	// Extend environment with parameter type.
	newEnv := &TypeEnvironment{
		Variables: make(map[string]*TypeScheme),
		Parent:    bc.context.environment,
		Level:     bc.context.environment.Level + 1,
	}

	paramType := funcType.Parameters[0]
	newEnv.Variables[lambda.Parameter] = &TypeScheme{
		TypeVars: []string{},
		Type:     paramType,
		Level:    newEnv.Level,
	}

	// Check body against expected return type.
	oldEnv := bc.context.environment
	bc.context.environment = newEnv

	defer func() {
		bc.context.environment = oldEnv
	}()

	bodyType, err := bc.CheckExpression(lambda.Body, funcType.ReturnType)
	if err != nil {
		return nil, err
	}

	if bc.verbose {
		fmt.Printf("Lambda body type: %s, expected: %s\n",
			bodyType.String(), funcType.ReturnType.String())
	}

	return bc.context.expectedType, nil
}

// checkAnnotated checks annotated expressions.
func (bc *BidirectionalChecker) checkAnnotated(annotated *AnnotatedExpr) (*Type, error) {
	// Get the type annotation.
	annotationType := annotated.Annotation

	// Check if annotation matches expected type.
	if bc.context.expectedType != nil {
		if err := bc.checkSubsumption(annotationType, bc.context.expectedType, annotated); err != nil {
			return nil, err
		}
	}

	// Check expression against annotated type.
	return bc.CheckExpression(annotated.Expression, annotationType)
}

// checkIf checks conditional expressions.
func (bc *BidirectionalChecker) checkIf(ifExpr *IfExpr) (*Type, error) {
	// Synthesize condition type and check it's boolean.
	condType, err := bc.SynthesizeType(ifExpr.Condition)
	if err != nil {
		return nil, err
	}

	if !condType.Equals(TypeBool) {
		return nil, bc.addError("if condition", TypeBool, condType,
			"Condition must be boolean")
	}

	// Check both branches against expected type.
	thenType, err := bc.CheckExpression(ifExpr.ThenBranch, bc.context.expectedType)
	if err != nil {
		return nil, err
	}

	elseType, err := bc.CheckExpression(ifExpr.ElseBranch, bc.context.expectedType)
	if err != nil {
		return nil, err
	}

	// Verify both branches have compatible types.
	if !thenType.Equals(elseType) {
		return nil, bc.addError("if branches", thenType, elseType,
			"Both branches must have the same type")
	}

	return thenType, nil
}

// synthesizeLiteral synthesizes types for literal expressions.
func (bc *BidirectionalChecker) synthesizeLiteral(literal *LiteralExpr) (*Type, error) {
	switch literal.Value.(type) {
	case int32:
		return TypeInt32, nil
	case int64:
		return TypeInt64, nil
	case float32:
		return TypeFloat32, nil
	case float64:
		return TypeFloat64, nil
	case string:
		return TypeString, nil
	case bool:
		return TypeBool, nil
	default:
		return nil, fmt.Errorf("unsupported literal type: %T", literal.Value)
	}
}

// synthesizeVariable synthesizes types for variable expressions.
func (bc *BidirectionalChecker) synthesizeVariable(variable *VariableExpr) (*Type, error) {
	scheme, exists := bc.inference.LookupVariable(variable.Name)
	if !exists {
		return nil, bc.addError("variable lookup", nil, nil,
			fmt.Sprintf("Undefined variable: %s", variable.Name))
	}

	return bc.inference.Instantiate(scheme), nil
}

// synthesizeApplication synthesizes types for function applications.
func (bc *BidirectionalChecker) synthesizeApplication(app *ApplicationExpr) (*Type, error) {
	// Synthesize function type.
	funcType, err := bc.SynthesizeType(app.Function)
	if err != nil {
		return nil, err
	}

	if funcType.Kind != TypeKindFunction {
		return nil, bc.addError("function application", nil, funcType,
			"Cannot apply non-function type")
	}

	funcData := funcType.Data.(*FunctionType)
	if len(funcData.Parameters) == 0 {
		return nil, bc.addError("function application", nil, nil,
			"Cannot apply function with no parameters")
	}

	// Check argument against parameter type.
	expectedParamType := funcData.Parameters[0]

	_, err = bc.CheckExpression(app.Argument, expectedParamType)
	if err != nil {
		return nil, err
	}

	// Return the return type (for single parameter functions).
	return funcData.ReturnType, nil
}

// synthesizeAnnotated synthesizes types for annotated expressions.
func (bc *BidirectionalChecker) synthesizeAnnotated(annotated *AnnotatedExpr) (*Type, error) {
	// Get the type annotation.
	annotationType := annotated.Annotation

	// Check expression against annotated type.
	_, err := bc.CheckExpression(annotated.Expression, annotationType)
	if err != nil {
		return nil, err
	}

	return annotationType, nil
}

// synthesizeBinary synthesizes types for binary expressions.
func (bc *BidirectionalChecker) synthesizeBinary(binary *BinaryExpr) (*Type, error) {
	// Synthesize operand types.
	leftType, err := bc.SynthesizeType(binary.Left)
	if err != nil {
		return nil, err
	}

	rightType, err := bc.SynthesizeType(binary.Right)
	if err != nil {
		return nil, err
	}

	// Determine result type based on operator.
	return bc.getBinaryOperatorResultType(binary.Operator, leftType, rightType)
}

// checkSubsumption checks if one type is a subtype of another.
func (bc *BidirectionalChecker) checkSubsumption(actual, expected *Type, expr Expr) error {
	if actual.Equals(expected) {
		return nil
	}

	// For now, use structural subtyping for functions.
	if actual.Kind == TypeKindFunction && expected.Kind == TypeKindFunction {
		actualFunc := actual.Data.(*FunctionType)
		expectedFunc := expected.Data.(*FunctionType)

		// Check contravariance in parameters and covariance in return type.
		if len(actualFunc.Parameters) != len(expectedFunc.Parameters) {
			return bc.addError("function subtyping", expected, actual,
				"Parameter count mismatch")
		}

		// Parameters are contravariant.
		for i, actualParam := range actualFunc.Parameters {
			expectedParam := expectedFunc.Parameters[i]
			if err := bc.checkSubsumption(expectedParam, actualParam, expr); err != nil {
				return err
			}
		}

		// Return type is covariant.
		return bc.checkSubsumption(actualFunc.ReturnType, expectedFunc.ReturnType, expr)
	}

	return bc.addError("type compatibility", expected, actual,
		"Types are not compatible")
}

// getBinaryOperatorResultType determines the result type of binary operations.
func (bc *BidirectionalChecker) getBinaryOperatorResultType(op string, left, right *Type) (*Type, error) {
	switch op {
	case "+", "-", "*", "/":
		// Arithmetic operators.
		if left.Equals(right) && (left.Equals(TypeInt32) || left.Equals(TypeInt64) ||
			left.Equals(TypeFloat32) || left.Equals(TypeFloat64)) {
			return left, nil
		}

		return nil, bc.addError("arithmetic operation", left, right,
			"Operands must be of the same numeric type")

	case "==", "!=", "<", ">", "<=", ">=":
		// Comparison operators.
		if left.Equals(right) {
			return TypeBool, nil
		}

		return nil, bc.addError("comparison operation", left, right,
			"Operands must be of the same type")

	case "&&", "||":
		// Logical operators.
		if left.Equals(TypeBool) && right.Equals(TypeBool) {
			return TypeBool, nil
		}

		return nil, bc.addError("logical operation", TypeBool, nil,
			"Operands must be boolean")

	default:
		return nil, bc.addError("binary operation", nil, nil,
			fmt.Sprintf("Unknown binary operator: %s", op))
	}
}

// parseTypeAnnotation parses a type annotation string into a Type.
func (bc *BidirectionalChecker) parseTypeAnnotation(annotation string) (*Type, error) {
	// Simple type annotation parsing.
	switch annotation {
	case "Int32":
		return TypeInt32, nil
	case "Int64":
		return TypeInt64, nil
	case "Float32":
		return TypeFloat32, nil
	case "Float64":
		return TypeFloat64, nil
	case "String":
		return TypeString, nil
	case "Bool":
		return TypeBool, nil
	default:
		// For more complex types, you would need a proper parser.
		return nil, fmt.Errorf("unsupported type annotation: %s", annotation)
	}
}

// addError adds a bidirectional error to the context.
func (bc *BidirectionalChecker) addError(context string, expected, actual *Type, message string) error {
	error := BidirectionalError{
		Message:      message,
		Location:     bc.context.position,
		ExpectedType: expected,
		ActualType:   actual,
		Context:      context,
	}

	bc.context.errors = append(bc.context.errors, error)

	return errors.New(error.String())
}

// BidirectionalInference combines bidirectional checking with constraint-based inference.
type BidirectionalInference struct {
	checker           *BidirectionalChecker
	constraintChecker *ConstraintBasedInference
	inference         *InferenceEngine
}

// NewBidirectionalInference creates a new bidirectional inference system.
func NewBidirectionalInference(inference *InferenceEngine) *BidirectionalInference {
	return &BidirectionalInference{
		checker:           NewBidirectionalChecker(inference),
		constraintChecker: NewConstraintBasedInference(inference),
		inference:         inference,
	}
}

// InferWithBidirectional performs type inference using bidirectional checking.
func (bi *BidirectionalInference) InferWithBidirectional(expr Expr, expectedType *Type) (*Type, error) {
	if expectedType != nil {
		// Use checking mode when expected type is provided.
		return bi.checker.CheckExpression(expr, expectedType)
	} else {
		// Use synthesis mode when no expected type is provided.
		return bi.checker.SynthesizeType(expr)
	}
}

// InferWithFallback tries bidirectional inference first, then falls back to constraint-based.
func (bi *BidirectionalInference) InferWithFallback(expr Expr, expectedType *Type) (*Type, error) {
	// Try bidirectional first.
	result, err := bi.InferWithBidirectional(expr, expectedType)
	if err == nil {
		return result, nil
	}

	// Fall back to constraint-based inference.
	if expectedType != nil {
		// Use constraint-based with expected type information.
		return bi.constraintChecker.InferTypeWithConstraints(expr)
	} else {
		// Use regular constraint-based inference.
		return bi.constraintChecker.InferTypeWithConstraints(expr)
	}
}

// SetVerbose enables or disables verbose output.
func (bi *BidirectionalInference) SetVerbose(verbose bool) {
	bi.checker.SetVerbose(verbose)
	bi.constraintChecker.SetVerbose(verbose)
}

// GetErrors returns all accumulated errors.
func (bi *BidirectionalInference) GetErrors() []BidirectionalError {
	return bi.checker.GetErrors()
}
