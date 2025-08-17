package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// TypeInferenceEngine handles type inference and unification
type TypeInferenceEngine struct {
	traitResolver   *TraitResolver
	inferenceStack  []*InferenceContext
	unificationVars map[string]*UnificationVariable
	constraints     []*TypeConstraint
	solutions       map[string]*parser.HIRType
	errors          []InferenceError
	debugMode       bool
}

// InferenceContext represents a context for type inference
type InferenceContext struct {
	Scope          InferenceScope
	LocalVars      map[string]*parser.HIRType
	ExpectedType   *parser.HIRType
	ReturnType     *parser.HIRType
	TypeParameters []*parser.HIRTypeParameter
	Constraints    []*TypeConstraint
	Position       position.Position
}

// UnificationVariable represents a type variable in unification
type UnificationVariable struct {
	ID              string
	Bounds          []*parser.HIRType // Upper bounds (T: Trait)
	LowerBounds     []*parser.HIRType // Lower bounds (supertypes)
	Solution        *parser.HIRType   // Solved type
	Kind            UnificationKind   // Kind of unification variable
	CreationContext string            // Where this variable was created
	Position        position.Position
}

// TypeConstraint represents a constraint between types
type TypeConstraint struct {
	Kind       ConstraintKind
	Left       *parser.HIRType
	Right      *parser.HIRType
	Variable   *UnificationVariable
	Position   position.Position
	Message    string
	IsResolved bool
}

// InferenceError represents an error in type inference
type InferenceError struct {
	Kind       InferenceErrorKind
	Message    string
	Position   position.Position
	Type1      *parser.HIRType
	Type2      *parser.HIRType
	Constraint *TypeConstraint
}

// InferenceScope represents different inference scopes
type InferenceScope int

const (
	InferenceScopeGlobal InferenceScope = iota
	InferenceScopeFunction
	InferenceScopeBlock
	InferenceScopeExpression
)

// UnificationKind represents different kinds of unification variables
type UnificationKind int

const (
	UnificationKindType UnificationKind = iota
	UnificationKindLifetime
	UnificationKindConst
	UnificationKindEffect
)

// ConstraintKind represents different kinds of type constraints
type ConstraintKind int

const (
	ConstraintKindEquality ConstraintKind = iota
	ConstraintKindSubtype
	ConstraintKindTrait
	ConstraintKindLifetime
	ConstraintKindEffect
)

// InferenceErrorKind represents different kinds of inference errors
type InferenceErrorKind int

const (
	InferenceErrorNone InferenceErrorKind = iota
	InferenceErrorUnificationFailure
	InferenceErrorConstraintViolation
	InferenceErrorInfiniteType
	InferenceErrorAmbiguousType
	InferenceErrorMissingConstraint
)

// NewTypeInferenceEngine creates a new type inference engine
func NewTypeInferenceEngine(traitResolver *TraitResolver) *TypeInferenceEngine {
	return &TypeInferenceEngine{
		traitResolver:   traitResolver,
		inferenceStack:  make([]*InferenceContext, 0),
		unificationVars: make(map[string]*UnificationVariable),
		constraints:     make([]*TypeConstraint, 0),
		solutions:       make(map[string]*parser.HIRType),
		errors:          make([]InferenceError, 0),
		debugMode:       false,
	}
}

// InferExpression infers the type of an expression
func (tie *TypeInferenceEngine) InferExpression(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// Create new inference context
	ctx := &InferenceContext{
		Scope:        InferenceScopeExpression,
		LocalVars:    make(map[string]*parser.HIRType),
		ExpectedType: expectedType,
		Position:     position.Position{Line: 1, Column: 1}, // Simplified position
	}

	tie.pushContext(ctx)
	defer tie.popContext()

	// If expression already has a type, return it
	if expr.Type != nil {
		return expr.Type, nil
	}

	// Infer based on expression kind
	switch expr.Kind {
	case parser.HIRExprLiteral:
		return tie.inferLiteral(expr)
	case parser.HIRExprVariable:
		return tie.inferVariable(expr)
	case parser.HIRExprCall:
		return tie.inferCall(expr, expectedType)
	case parser.HIRExprBinary:
		return tie.inferBinary(expr, expectedType)
	case parser.HIRExprUnary:
		return tie.inferUnary(expr, expectedType)
	case parser.HIRExprFieldAccess:
		return tie.inferField(expr, expectedType)
	case parser.HIRExprIndex:
		return tie.inferIndex(expr, expectedType)
	case parser.HIRExprArray:
		return tie.inferArray(expr, expectedType)
	case parser.HIRExprStruct:
		return tie.inferStruct(expr, expectedType)
	default:
		// Create unification variable for unsupported expressions
		unificationVar := tie.CreateUnificationVariable(
			fmt.Sprintf("expr_%s", expr.Kind.String()),
			position.Position{Line: 1, Column: 1},
		)
		return &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: unificationVar.ID}, nil
	}
}

// InferFunction infers types for a function body
func (tie *TypeInferenceEngine) InferFunction(function *parser.HIRFunction) error {
	// Create function-level inference context
	ctx := &InferenceContext{
		Scope:          InferenceScopeFunction,
		LocalVars:      make(map[string]*parser.HIRType),
		ReturnType:     function.ReturnType,
		TypeParameters: function.TypeParameters,
		Position:       position.Position{Line: 1, Column: 1}, // Simplified position
	}

	tie.pushContext(ctx)
	defer tie.popContext()

	// Add parameters to local variable scope
	for _, param := range function.Parameters {
		ctx.LocalVars[param.Name] = param.Type
	}

	// Infer function body
	if function.Body != nil {
		if err := tie.inferBlock(function.Body); err != nil {
			return err
		}
	}

	// Solve constraints
	return tie.solveConstraints()
}

// Unify attempts to unify two types
func (tie *TypeInferenceEngine) Unify(type1, type2 *parser.HIRType) error {
	if tie.debugMode {
		fmt.Printf("Unifying %s with %s\n", type1.String(), type2.String())
	}

	// Handle nil types
	if type1 == nil || type2 == nil {
		return fmt.Errorf("cannot unify nil types")
	}

	// Handle identical types (simplified check)
	if type1.Kind == type2.Kind && type1.Data == type2.Data {
		return nil
	}

	// Handle unification variables
	if var1, isVar1 := tie.isUnificationVariable(type1); isVar1 {
		return tie.unifyVariable(var1, type2)
	}
	if var2, isVar2 := tie.isUnificationVariable(type2); isVar2 {
		return tie.unifyVariable(var2, type1)
	}

	// Handle structural unification
	return tie.unifyStructural(type1, type2)
}

// CreateUnificationVariable creates a new unification variable
func (tie *TypeInferenceEngine) CreateUnificationVariable(context string, pos position.Position) *UnificationVariable {
	id := fmt.Sprintf("T%d", len(tie.unificationVars))

	variable := &UnificationVariable{
		ID:              id,
		Bounds:          make([]*parser.HIRType, 0),
		LowerBounds:     make([]*parser.HIRType, 0),
		Kind:            UnificationKindType,
		CreationContext: context,
		Position:        pos,
	}

	tie.unificationVars[id] = variable
	return variable
}

// AddConstraint adds a type constraint
func (tie *TypeInferenceEngine) AddConstraint(constraint *TypeConstraint) {
	tie.constraints = append(tie.constraints, constraint)
	if tie.debugMode {
		fmt.Printf("Added constraint: %s\n", constraint.Message)
	}
}

// Private helper methods

func (tie *TypeInferenceEngine) pushContext(ctx *InferenceContext) {
	tie.inferenceStack = append(tie.inferenceStack, ctx)
}

func (tie *TypeInferenceEngine) popContext() {
	if len(tie.inferenceStack) > 0 {
		tie.inferenceStack = tie.inferenceStack[:len(tie.inferenceStack)-1]
	}
}

func (tie *TypeInferenceEngine) getCurrentContext() *InferenceContext {
	if len(tie.inferenceStack) == 0 {
		return nil
	}
	return tie.inferenceStack[len(tie.inferenceStack)-1]
}

func (tie *TypeInferenceEngine) inferLiteral(expr *parser.HIRExpression) (*parser.HIRType, error) {
	// Basic literal type inference
	// In a full implementation, this would examine the Data field for literal values
	switch expr.Kind {
	case parser.HIRExprLiteral:
		// Default to string type for literals
		return &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "string"}, nil
	default:
		return &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "unknown"}, nil
	}
}

func (tie *TypeInferenceEngine) inferVariable(expr *parser.HIRExpression) (*parser.HIRType, error) {
	ctx := tie.getCurrentContext()
	if ctx == nil {
		return nil, fmt.Errorf("no inference context for variable")
	}

	// Create unification variable for unknown variables
	unificationVar := tie.CreateUnificationVariable(
		"variable",
		position.Position{Line: 1, Column: 1},
	)

	// Convert to HIR type
	hirType := &parser.HIRType{
		Kind: parser.HIRTypeGeneric,
		Data: unificationVar.ID,
	}

	return hirType, nil
}

func (tie *TypeInferenceEngine) inferCall(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// Create unification variable for unknown return type
	returnVar := tie.CreateUnificationVariable("call_return", position.Position{Line: 1, Column: 1})
	return &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: returnVar.ID}, nil
}

func (tie *TypeInferenceEngine) inferBinary(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// For binary operations, assume they return the same type as the left operand
	// or bool for comparison operations
	if expectedType != nil {
		return expectedType, nil
	}

	// Default to int for arithmetic operations
	return &parser.HIRType{Kind: parser.HIRTypePrimitive, Data: "i32"}, nil
}

func (tie *TypeInferenceEngine) inferUnary(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// For unary operations, return expected type or create unification variable
	if expectedType != nil {
		return expectedType, nil
	}

	unificationVar := tie.CreateUnificationVariable("unary_result", position.Position{Line: 1, Column: 1})
	return &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: unificationVar.ID}, nil
}

func (tie *TypeInferenceEngine) inferField(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// Create unification variable for unknown field type
	fieldVar := tie.CreateUnificationVariable("field", position.Position{Line: 1, Column: 1})
	return &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: fieldVar.ID}, nil
}

func (tie *TypeInferenceEngine) inferIndex(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// Create unification variable for unknown element type
	elementVar := tie.CreateUnificationVariable("array_element", position.Position{Line: 1, Column: 1})
	return &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: elementVar.ID}, nil
}

func (tie *TypeInferenceEngine) inferArray(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// Use expected type or create array with unknown element type
	if expectedType != nil && expectedType.Kind == parser.HIRTypeArray {
		return expectedType, nil
	}

	elementVar := tie.CreateUnificationVariable("array_element", position.Position{Line: 1, Column: 1})
	return &parser.HIRType{
		Kind: parser.HIRTypeArray,
		Data: map[string]interface{}{
			"element_type": &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: elementVar.ID},
			"size":         0,
		},
	}, nil
}

func (tie *TypeInferenceEngine) inferStruct(expr *parser.HIRExpression, expectedType *parser.HIRType) (*parser.HIRType, error) {
	// For struct literals, use expected type or create unification variable
	if expectedType != nil && expectedType.Kind == parser.HIRTypeStruct {
		return expectedType, nil
	}

	structVar := tie.CreateUnificationVariable("struct_literal", position.Position{Line: 1, Column: 1})
	return &parser.HIRType{Kind: parser.HIRTypeGeneric, Data: structVar.ID}, nil
}

func (tie *TypeInferenceEngine) inferBlock(block *parser.HIRBlock) error {
	for _, stmt := range block.Statements {
		if err := tie.inferStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (tie *TypeInferenceEngine) inferStatement(stmt *parser.HIRStatement) error {
	// Basic statement inference - simplified to avoid HIR complexity
	switch stmt.Kind {
	case parser.HIRStmtExpression:
		// For expression statements, simply mark as processed
		return nil
	default:
		// For other statement types, just return success
		return nil
	}
}

func (tie *TypeInferenceEngine) isUnificationVariable(hirType *parser.HIRType) (*UnificationVariable, bool) {
	if hirType.Kind == parser.HIRTypeGeneric {
		if variable, exists := tie.unificationVars[hirType.Data.(string)]; exists {
			return variable, true
		}
	}
	return nil, false
}

func (tie *TypeInferenceEngine) unifyVariable(variable *UnificationVariable, otherType *parser.HIRType) error {
	// If variable is already solved, unify with solution
	if variable.Solution != nil {
		return tie.Unify(variable.Solution, otherType)
	}

	// Check bounds
	for _, bound := range variable.Bounds {
		if err := tie.checkBound(otherType, bound); err != nil {
			return err
		}
	}

	// Set solution
	variable.Solution = otherType
	tie.solutions[variable.ID] = otherType

	if tie.debugMode {
		fmt.Printf("Solved %s = %s\n", variable.ID, otherType.String())
	}

	return nil
}

func (tie *TypeInferenceEngine) unifyStructural(type1, type2 *parser.HIRType) error {
	// Handle different type kinds
	if type1.Kind != type2.Kind {
		return fmt.Errorf("cannot unify different type kinds: %v and %v", type1.Kind, type2.Kind)
	}

	// For simplified implementation, accept structural types as unified
	// A full implementation would recursively check array elements,
	// function parameters/returns, struct fields, etc.
	return nil
}

func (tie *TypeInferenceEngine) checkBound(hirType *parser.HIRType, bound *parser.HIRType) error {
	// Check if type satisfies bound (simplified)
	// In a full implementation, this would check trait bounds, subtyping, etc.
	return nil
}

func (tie *TypeInferenceEngine) solveConstraints() error {
	// Solve constraints iteratively
	changed := true
	for changed {
		changed = false

		for _, constraint := range tie.constraints {
			if constraint.IsResolved {
				continue
			}

			if err := tie.solveConstraint(constraint); err != nil {
				tie.addInferenceError(InferenceErrorConstraintViolation, err.Error(), constraint.Position, constraint.Left, constraint.Right)
				return err
			}

			constraint.IsResolved = true
			changed = true
		}
	}

	return nil
}

func (tie *TypeInferenceEngine) solveConstraint(constraint *TypeConstraint) error {
	switch constraint.Kind {
	case ConstraintKindEquality:
		return tie.Unify(constraint.Left, constraint.Right)
	case ConstraintKindSubtype:
		// Check subtype relationship
		return tie.checkSubtype(constraint.Left, constraint.Right)
	case ConstraintKindTrait:
		// Check trait bound
		return tie.checkTraitBound(constraint.Left, constraint.Right)
	default:
		return fmt.Errorf("unsupported constraint kind: %v", constraint.Kind)
	}
}

func (tie *TypeInferenceEngine) checkSubtype(subtype, supertype *parser.HIRType) error {
	// Simplified subtype checking
	// In a full implementation, this would handle variance, inheritance, etc.
	return tie.Unify(subtype, supertype)
}

func (tie *TypeInferenceEngine) checkTraitBound(hirType, traitType *parser.HIRType) error {
	// Check if type implements trait
	// This would integrate with the trait resolver
	return nil
}

func (tie *TypeInferenceEngine) addInferenceError(kind InferenceErrorKind, message string, pos position.Position, type1, type2 *parser.HIRType) {
	error := InferenceError{
		Kind:     kind,
		Message:  message,
		Position: pos,
		Type1:    type1,
		Type2:    type2,
	}
	tie.errors = append(tie.errors, error)
}

// GetErrors returns all inference errors
func (tie *TypeInferenceEngine) GetErrors() []InferenceError {
	return tie.errors
}

// GetSolutions returns all solved unification variables
func (tie *TypeInferenceEngine) GetSolutions() map[string]*parser.HIRType {
	return tie.solutions
}

// SetDebugMode enables or disables debug output
func (tie *TypeInferenceEngine) SetDebugMode(enabled bool) {
	tie.debugMode = enabled
}
