// Simple constraint handling for basic type operations
// This module provides simplified constraint operations for common cases

package types

import (
	"fmt"
)

// SimpleConstraintSolver provides simplified constraint solving for basic type operations
type SimpleConstraintSolver struct {
	constraints []*ExtendedConstraint
}

// NewSimpleConstraintSolver creates a new simple constraint solver
func NewSimpleConstraintSolver() *SimpleConstraintSolver {
	return &SimpleConstraintSolver{
		constraints: make([]*ExtendedConstraint, 0),
	}
}

// AddEqualityConstraint adds a simple equality constraint
func (scs *SimpleConstraintSolver) AddEqualityConstraint(left, right *Type) {
	constraint := &ExtendedConstraint{
		Kind:  ExtendedUnificationConstraint,
		Left:  left,
		Right: right,
	}
	scs.constraints = append(scs.constraints, constraint)
}

// AddSubtypeConstraint adds a simple subtype constraint
func (scs *SimpleConstraintSolver) AddSubtypeConstraint(subtype, supertype *Type) {
	constraint := &ExtendedConstraint{
		Kind:  ExtendedSubtypeConstraint,
		Left:  subtype,
		Right: supertype,
	}
	scs.constraints = append(scs.constraints, constraint)
}

// SolveBasic performs basic constraint solving for simple cases
func (scs *SimpleConstraintSolver) SolveBasic() error {
	for _, constraint := range scs.constraints {
		if err := scs.solveConstraint(constraint); err != nil {
			return fmt.Errorf("failed to solve constraint %s: %w", constraint.String(), err)
		}
	}
	return nil
}

// solveConstraint solves a single constraint
func (scs *SimpleConstraintSolver) solveConstraint(constraint *ExtendedConstraint) error {
	switch constraint.Kind {
	case ExtendedUnificationConstraint:
		return scs.solveUnification(constraint.Left, constraint.Right)
	case ExtendedSubtypeConstraint:
		return scs.solveSubtype(constraint.Left, constraint.Right)
	default:
		return fmt.Errorf("unsupported constraint kind: %s", constraint.Kind.String())
	}
}

// solveUnification solves a unification constraint
func (scs *SimpleConstraintSolver) solveUnification(left, right *Type) error {
	if left == nil || right == nil {
		return fmt.Errorf("cannot unify with nil type")
	}

	// Simple unification - check kind equality
	if left.Kind != right.Kind {
		return fmt.Errorf("cannot unify types of different kinds: %s and %s",
			left.Kind.String(), right.Kind.String())
	}

	return nil
}

// solveSubtype solves a subtype constraint
func (scs *SimpleConstraintSolver) solveSubtype(subtype, supertype *Type) error {
	if subtype == nil || supertype == nil {
		return fmt.Errorf("cannot check subtyping with nil type")
	}

	// Simple subtyping - for now, allow same types and upcast primitives
	if subtype.Kind == supertype.Kind {
		return nil
	}

	// Allow some basic numeric upcasting
	if isNumericType(subtype.Kind) && isNumericType(supertype.Kind) {
		return nil
	}

	return fmt.Errorf("type %s is not a subtype of %s",
		subtype.Kind.String(), supertype.Kind.String())
}

// isNumericType checks if a type kind represents a numeric type
func isNumericType(kind TypeKind) bool {
	switch kind {
	case TypeKindInt8, TypeKindInt16, TypeKindInt32, TypeKindInt64,
		TypeKindUint8, TypeKindUint16, TypeKindUint32, TypeKindUint64,
		TypeKindFloat32, TypeKindFloat64:
		return true
	default:
		return false
	}
}

// Clear clears all constraints from the solver
func (scs *SimpleConstraintSolver) Clear() {
	scs.constraints = scs.constraints[:0]
}

// ConstraintCount returns the number of constraints currently held
func (scs *SimpleConstraintSolver) ConstraintCount() int {
	return len(scs.constraints)
}
