package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// TraitResolver handles trait resolution and associated type inference.
type TraitResolver struct {
	modules                []*parser.HIRModule
	associatedTypeResolver *AssociatedTypeResolver
	throwsChecker          *ThrowsSpecificationChecker
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

// NewTraitResolver creates a new trait resolver.
func NewTraitResolver(modules []*parser.HIRModule) *TraitResolver {
	return &TraitResolver{
		modules:                modules,
		associatedTypeResolver: NewAssociatedTypeResolver(),
		throwsChecker:          NewThrowsSpecificationChecker(nil),
		errors:                 make([]TraitError, 0),
	}
}

// GetErrors returns all accumulated errors.
func (tr *TraitResolver) GetErrors() []TraitError {
	return tr.errors
}

// GetWhereClauseConstraints returns where clause constraints for a scope.
func (tr *TraitResolver) GetWhereClauseConstraints(scope string) []*WhereClauseConstraint {
	return tr.associatedTypeResolver.GetWhereClauseConstraints(scope)
}

// AddWhereClauseConstraint adds a where clause constraint.
func (tr *TraitResolver) AddWhereClauseConstraint(constraint *WhereClauseConstraint) {
	tr.associatedTypeResolver.AddWhereClauseConstraint(constraint)
}

// CheckTraitThrowsConsistency checks exception specification consistency for a trait.
func (tr *TraitResolver) CheckTraitThrowsConsistency(trait *parser.HIRTraitType) error {
	return tr.throwsChecker.CheckTraitThrowsConsistency(trait)
}

// CheckImplThrowsConsistency checks that implementation methods conform to trait
// exception specifications.
func (tr *TraitResolver) CheckImplThrowsConsistency(impl *parser.HIRImpl,
	trait *parser.HIRTraitType) error {
	return tr.throwsChecker.CheckImplThrowsConsistency(impl, trait)
}

// CheckWhereClauseThrowsConsistency checks exception specifications in where clauses.
func (tr *TraitResolver) CheckWhereClauseThrowsConsistency(whereClause *WhereClauseConstraint) error {
	return tr.throwsChecker.CheckWhereClauseThrowsConsistency(whereClause)
}

// GetThrowsViolations returns all accumulated throws violations.
func (tr *TraitResolver) GetThrowsViolations() []ThrowsViolation {
	return tr.throwsChecker.GetViolations()
}

// HasThrowsViolations checks if there are any throws violations.
func (tr *TraitResolver) HasThrowsViolations() bool {
	return tr.throwsChecker.HasViolations()
}

// typesMatch checks if two HIR types are equivalent.
func (tr *TraitResolver) typesMatch(type1, type2 *parser.HIRType) bool {
	if type1 == nil && type2 == nil {
		return true
	}

	if type1 == nil || type2 == nil {
		return false
	}

	// Basic structural matching.
	if type1.Kind != type2.Kind {
		return false
	}

	// Compare Data fields.
	return fmt.Sprintf("%v", type1.Data) == fmt.Sprintf("%v", type2.Data)
}
