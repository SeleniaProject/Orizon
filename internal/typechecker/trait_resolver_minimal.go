package typechecker

import (
	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// TraitResolver handles trait resolution and associated type inference.
type TraitResolver struct {
	modules                []*parser.HIRModule
	currentScope           ScopeStack
	associatedTypeResolver *AssociatedTypeResolver
	throwsChecker          *TraitThrowsChecker
	errors                 []TraitError
}

// TraitError represents an error in trait resolution.
type TraitError struct {
	Kind     string
	Message  string
	Position position.Position
}

// Error implements the error interface.
func (e *TraitError) Error() string { return e.Message }

// TraitThrowsChecker represents a throws specification checker.
type TraitThrowsChecker struct {
	violations []ThrowsViolation
}

// NewTraitThrowsChecker creates a new trait throws checker.
func NewTraitThrowsChecker() *TraitThrowsChecker {
	return &TraitThrowsChecker{
		violations: make([]ThrowsViolation, 0),
	}
}

// CheckTraitThrowsConsistency checks trait throws consistency.
func (c *TraitThrowsChecker) CheckTraitThrowsConsistency(_ *parser.HIRTraitType) error {
	return nil
}

// CheckImplThrowsConsistency checks implementation throws consistency.
func (c *TraitThrowsChecker) CheckImplThrowsConsistency(_ *parser.HIRImpl, _ *parser.HIRTraitType) error {
	return nil
}

// CheckWhereClauseThrowsConsistency checks where clause throws consistency.
func (c *TraitThrowsChecker) CheckWhereClauseThrowsConsistency(_ *WhereClauseConstraint) error {
	return nil
}

// GetViolations returns all violations.
func (c *TraitThrowsChecker) GetViolations() []ThrowsViolation {
	return c.violations
}

// HasViolations returns true if there are violations.
func (c *TraitThrowsChecker) HasViolations() bool {
	return len(c.violations) > 0
}

// NewTraitResolver creates a new trait resolver.
func NewTraitResolver(modules []*parser.HIRModule) *TraitResolver {
	return &TraitResolver{
		modules:                modules,
		currentScope:           make(ScopeStack, 0),
		associatedTypeResolver: NewAssociatedTypeResolver(),
		throwsChecker:          NewTraitThrowsChecker(),
		errors:                 make([]TraitError, 0),
	}
}

// GetErrors returns all accumulated errors
func (tr *TraitResolver) GetErrors() []TraitError {
	return tr.errors
}

// ScopeStack manages resolution scope stack.
type ScopeStack []ResolutionScope

// ResolutionScope represents a scope for trait resolution priority.
type ResolutionScope struct {
	Kind     ScopeKind
	Priority int
	Name     string
}

// ScopeKind represents different kinds of resolution scopes.
type ScopeKind int

const (
	ScopeKindModule ScopeKind = iota
	ScopeKindTrait
	ScopeKindImpl
	ScopeKindFunction
	ScopeKindBlock
)
