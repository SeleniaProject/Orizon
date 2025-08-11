// Constraint generator for the Orizon type system
// This module generates type constraints from HIR nodes for type inference

package types

import (
	"fmt"
)

// ConstraintGenerator generates type constraints from HIR expressions
// This is used during type inference to build constraint systems
type ConstraintGenerator struct {
	constraints []*TypeInferenceConstraint
	nextTypeVar int
}

// TypeInferenceConstraint represents a constraint between two types during inference
type TypeInferenceConstraint struct {
	Left  *Type
	Right *Type
	Kind  TypeConstraintKind
}

// TypeConstraintKind represents the kind of type constraint
type TypeConstraintKind int

const (
	TypeConstraintEqual TypeConstraintKind = iota // Left = Right
	TypeConstraintSub                             // Left <: Right (subtype)
	TypeConstraintSuper                           // Left :> Right (supertype)
)

// NewConstraintGenerator creates a new constraint generator
func NewConstraintGenerator() *ConstraintGenerator {
	return &ConstraintGenerator{
		constraints: make([]*TypeInferenceConstraint, 0),
		nextTypeVar: 0,
	}
}

// GenerateConstraints generates constraints from an expression
// This is the main entry point for constraint generation
func (cg *ConstraintGenerator) GenerateConstraints(expr interface{}) (*Type, []*TypeInferenceConstraint, error) {
	// TODO: Implement constraint generation for different expression types
	// This is a stub implementation that will be expanded as needed
	voidType := &Type{Kind: TypeKindVoid}
	return voidType, cg.constraints, nil
}

// FreshTypeVariable creates a fresh type variable for constraint generation
func (cg *ConstraintGenerator) FreshTypeVariable() *Type {
	tv := NewTypeVar(cg.nextTypeVar, fmt.Sprintf("t%d", cg.nextTypeVar), nil)
	cg.nextTypeVar++
	return tv
}

// AddConstraint adds a constraint to the constraint set
func (cg *ConstraintGenerator) AddConstraint(constraint *TypeInferenceConstraint) {
	cg.constraints = append(cg.constraints, constraint)
}

// GetConstraints returns all generated constraints
func (cg *ConstraintGenerator) GetConstraints() []*TypeInferenceConstraint {
	return cg.constraints
}

// Reset clears all constraints and resets the generator state
func (cg *ConstraintGenerator) Reset() {
	cg.constraints = cg.constraints[:0]
	cg.nextTypeVar = 0
}
