package typechecker

import (
	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// InferenceContext represents a context for type inference.
type InferenceContext struct {
	LocalVars      map[string]*parser.HIRType
	ExpectedType   *parser.HIRType
	ReturnType     *parser.HIRType
	TypeParameters []*parser.HIRTypeParameter
	Constraints    []*TypeConstraint
	Position       position.Position
	Scope          InferenceScope
}

// UnificationVariable represents a type variable in unification.
type UnificationVariable struct {
	Solution        *parser.HIRType
	ID              string
	CreationContext string
	Bounds          []*parser.HIRType
	LowerBounds     []*parser.HIRType
	Position        position.Position
	Kind            UnificationKind
}

// TypeConstraint represents a constraint between types.
type TypeConstraint struct {
	Left       *parser.HIRType
	Right      *parser.HIRType
	Variable   *UnificationVariable
	Message    string
	Position   position.Position
	Kind       ConstraintKind
	IsResolved bool
}

// InferenceError represents an error in type inference.
type InferenceError struct {
	Type1      *parser.HIRType
	Type2      *parser.HIRType
	Constraint *TypeConstraint
	Message    string
	Position   position.Position
	Kind       InferenceErrorKind
}

// InferenceScope represents different inference scopes.
type InferenceScope int

const (
	InferenceScopeGlobal InferenceScope = iota
	InferenceScopeFunction
	InferenceScopeBlock
	InferenceScopeExpression
)

// UnificationKind represents different kinds of unification variables.
type UnificationKind int

const (
	UnificationKindType UnificationKind = iota
	UnificationKindLifetime
	UnificationKindConst
	UnificationKindEffect
)

// ConstraintKind represents different kinds of type constraints.
type ConstraintKind int

const (
	ConstraintKindEquality ConstraintKind = iota
	ConstraintKindSubtype
	ConstraintKindTrait
	ConstraintKindLifetime
	ConstraintKindEffect
)

// InferenceErrorKind represents different kinds of inference errors.
type InferenceErrorKind int

const (
	InferenceErrorNone InferenceErrorKind = iota
	InferenceErrorUnificationFailure
	InferenceErrorConstraintViolation
	InferenceErrorInfiniteType
	InferenceErrorAmbiguousType
	InferenceErrorMissingConstraint
)
