package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/resolver"
)

// AssociatedTypeResolver handles resolution of associated types and where clauses
type AssociatedTypeResolver struct {
	tr                *TraitResolver
	associatedTypes   map[string]*AssociatedTypeBinding   // trait.name -> associated type bindings
	whereConstraints  map[string][]*WhereClauseConstraint // scope -> where clause constraints
	constraintCache   map[string]*ConstraintSolution      // cache for solved constraints
	activeConstraints []*ActiveConstraint                 // currently active constraint stack
}

// AssociatedTypeBinding represents a binding for a trait's associated type
type AssociatedTypeBinding struct {
	TraitName    string
	TypeName     string
	BoundType    *parser.HIRType
	Constraints  []*parser.HIRType
	DefaultType  *parser.HIRType
	IsProjection bool // true if this is a type projection (T::AssocType)
}

// WhereClauseConstraint represents a where clause constraint
type WhereClauseConstraint struct {
	TypeParam       string                 // Type parameter being constrained
	TraitBounds     []*parser.HIRType      // Trait bounds (T: Display + Clone)
	AssocTypeBounds []*AssocTypeConstraint // Associated type bounds (T::Item: Display)
	EqualityBounds  []*EqualityConstraint  // Equality bounds (T::Item = String)
	Scope           ConstraintScope        // Scope where this constraint applies
}

// AssocTypeConstraint represents an associated type constraint (T::Item: Display)
type AssocTypeConstraint struct {
	TypeParam   string
	AssocType   string
	TraitBounds []*parser.HIRType
}

// EqualityConstraint represents an equality constraint (T::Item = String)
type EqualityConstraint struct {
	TypeParam string
	AssocType string
	EqualTo   *parser.HIRType
}

// ConstraintScope represents the scope of a constraint
type ConstraintScope int

const (
	ConstraintScopeGlobal ConstraintScope = iota
	ConstraintScopeImpl
	ConstraintScopeFunction
	ConstraintScopeLocal
)

// ActiveConstraint represents a constraint currently being processed
type ActiveConstraint struct {
	Constraint *WhereClauseConstraint
	Depth      int
	Context    string
}

// ConstraintSolution represents a solved constraint set
type ConstraintSolution struct {
	TypeBindings  map[string]*parser.HIRType        // Type parameter -> concrete type
	AssocBindings map[string]*AssociatedTypeBinding // Associated type -> binding
	TraitImpls    map[string]*parser.HIRImpl        // Required trait implementations
	Satisfied     bool
	Conflicts     []ConstraintConflict
}

// ConstraintConflict represents a conflict between constraints
type ConstraintConflict struct {
	Constraint1 *WhereClauseConstraint
	Constraint2 *WhereClauseConstraint
	Reason      string
}

// NewAssociatedTypeResolver creates a new associated type resolver
func NewAssociatedTypeResolver(tr *TraitResolver) *AssociatedTypeResolver {
	return &AssociatedTypeResolver{
		tr:                tr,
		associatedTypes:   make(map[string]*AssociatedTypeBinding),
		whereConstraints:  make(map[string][]*WhereClauseConstraint),
		constraintCache:   make(map[string]*ConstraintSolution),
		activeConstraints: []*ActiveConstraint{},
	}
}

// ResolveAssociatedType resolves an associated type projection (T::AssocType)
func (atr *AssociatedTypeResolver) ResolveAssociatedType(typeParam string, assocTypeName string, context *resolver.ResolutionContext) (*parser.HIRType, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("%s::%s", typeParam, assocTypeName)

	// Check cache first
	if binding, exists := atr.associatedTypes[cacheKey]; exists {
		if binding.BoundType != nil {
			return binding.BoundType, nil
		}
	}

	// Find trait bounds for the type parameter
	traitBounds := atr.findTraitBoundsForTypeParam(typeParam, context)
	if len(traitBounds) == 0 {
		return nil, fmt.Errorf("no trait bounds found for type parameter %s", typeParam)
	}

	// Look for the associated type in trait bounds
	for _, traitBound := range traitBounds {
		trait := atr.tr.findTrait(traitBound)
		if trait == nil {
			continue
		}

		// Find the associated type in this trait
		for _, assocType := range trait.AssociatedTypes {
			if assocType.Name == assocTypeName {
				// Check if we have a concrete binding for this associated type
				binding := atr.findAssociatedTypeBinding(trait.Name, assocTypeName, context)
				if binding != nil && binding.BoundType != nil {
					// Cache the result
					atr.associatedTypes[cacheKey] = binding
					return binding.BoundType, nil
				}

				// Create projection type if no concrete binding exists
				projection := atr.createTypeProjection(typeParam, assocTypeName, trait)
				atr.associatedTypes[cacheKey] = &AssociatedTypeBinding{
					TraitName:    trait.Name,
					TypeName:     assocTypeName,
					BoundType:    projection,
					Constraints:  assocType.Bounds,
					IsProjection: true,
				}
				return projection, nil
			}
		}
	}

	return nil, fmt.Errorf("associated type %s not found in any trait bound for %s", assocTypeName, typeParam)
}

// ValidateWhereClause validates a where clause and its constraints
func (atr *AssociatedTypeResolver) ValidateWhereClause(constraint *WhereClauseConstraint, context *resolver.ResolutionContext) []ConstraintConflict {
	var conflicts []ConstraintConflict

	// Add to active constraints for cycle detection
	active := &ActiveConstraint{
		Constraint: constraint,
		Depth:      len(atr.activeConstraints),
		Context:    fmt.Sprintf("resolver.ResolutionContext{%v}", context),
	}
	atr.activeConstraints = append(atr.activeConstraints, active)
	defer func() {
		atr.activeConstraints = atr.activeConstraints[:len(atr.activeConstraints)-1]
	}()

	// Check for cycles
	if atr.detectConstraintCycle(constraint) {
		conflicts = append(conflicts, ConstraintConflict{
			Constraint1: constraint,
			Constraint2: constraint,
			Reason:      "cyclic constraint dependency",
		})
		return conflicts
	}

	// Validate trait bounds
	for _, traitBound := range constraint.TraitBounds {
		if err := atr.validateTraitBound(constraint.TypeParam, traitBound, context); err != nil {
			conflicts = append(conflicts, ConstraintConflict{
				Constraint1: constraint,
				Reason:      fmt.Sprintf("trait bound validation failed: %v", err),
			})
		}
	}

	// Validate associated type bounds
	for _, assocBound := range constraint.AssocTypeBounds {
		if err := atr.validateAssociatedTypeBound(assocBound, context); err != nil {
			conflicts = append(conflicts, ConstraintConflict{
				Constraint1: constraint,
				Reason:      fmt.Sprintf("associated type bound validation failed: %v", err),
			})
		}
	}

	// Validate equality bounds
	for _, eqBound := range constraint.EqualityBounds {
		if err := atr.validateEqualityBound(eqBound, context); err != nil {
			conflicts = append(conflicts, ConstraintConflict{
				Constraint1: constraint,
				Reason:      fmt.Sprintf("equality bound validation failed: %v", err),
			})
		}
	}

	return conflicts
}

// SolveConstraintSet solves a set of where clause constraints
func (atr *AssociatedTypeResolver) SolveConstraintSet(constraints []*WhereClauseConstraint, context *resolver.ResolutionContext) *ConstraintSolution {
	// Create cache key for constraint set
	cacheKey := atr.createConstraintSetKey(constraints)

	// Check cache first
	if solution, exists := atr.constraintCache[cacheKey]; exists {
		return solution
	}

	solution := &ConstraintSolution{
		TypeBindings:  make(map[string]*parser.HIRType),
		AssocBindings: make(map[string]*AssociatedTypeBinding),
		TraitImpls:    make(map[string]*parser.HIRImpl),
		Satisfied:     true,
		Conflicts:     []ConstraintConflict{},
	}

	// Solve each constraint
	for _, constraint := range constraints {
		conflicts := atr.ValidateWhereClause(constraint, context)
		solution.Conflicts = append(solution.Conflicts, conflicts...)

		if len(conflicts) > 0 {
			solution.Satisfied = false
		}

		// Extract type bindings from constraint
		atr.extractTypeBindings(constraint, solution, context)
	}

	// Check for conflicting bindings
	atr.checkBindingConflicts(solution)

	// Cache the solution
	atr.constraintCache[cacheKey] = solution

	return solution
}

// Helper methods

// findTraitBoundsForTypeParam finds all trait bounds for a type parameter
func (atr *AssociatedTypeResolver) findTraitBoundsForTypeParam(typeParam string, context *resolver.ResolutionContext) []*parser.HIRType {
	var bounds []*parser.HIRType

	// Look in current scope constraints
	for scope, constraints := range atr.whereConstraints {
		if atr.isInScope(scope, context) {
			for _, constraint := range constraints {
				if constraint.TypeParam == typeParam {
					bounds = append(bounds, constraint.TraitBounds...)
				}
			}
		}
	}

	return bounds
}

// findAssociatedTypeBinding finds a concrete binding for an associated type
func (atr *AssociatedTypeResolver) findAssociatedTypeBinding(traitName, assocTypeName string, context *resolver.ResolutionContext) *AssociatedTypeBinding {
	key := fmt.Sprintf("%s::%s", traitName, assocTypeName)
	return atr.associatedTypes[key]
}

// createTypeProjection creates a type projection for an unresolved associated type
func (atr *AssociatedTypeResolver) createTypeProjection(typeParam, assocTypeName string, trait *parser.HIRTraitType) *parser.HIRType {
	// This would create a proper type projection in the HIR
	// For now, return a generic type representing the projection
	return &parser.HIRType{
		Kind: parser.HIRTypeGeneric,
		Data: &parser.HIRGenericType{
			Name: fmt.Sprintf("%s::%s", typeParam, assocTypeName),
		},
	}
}

// detectConstraintCycle detects cycles in constraint dependencies
func (atr *AssociatedTypeResolver) detectConstraintCycle(constraint *WhereClauseConstraint) bool {
	// Simple cycle detection by checking if the same constraint is already active
	for _, active := range atr.activeConstraints {
		if atr.constraintsEqual(active.Constraint, constraint) {
			return true
		}
	}
	return false
}

// validateTraitBound validates that a trait bound is well-formed
func (atr *AssociatedTypeResolver) validateTraitBound(typeParam string, traitBound *parser.HIRType, context *resolver.ResolutionContext) error {
	trait := atr.tr.findTrait(traitBound)
	if trait == nil {
		return fmt.Errorf("trait not found: %v", traitBound)
	}

	// Check that the trait is applicable to the type parameter
	// This would involve more complex trait resolution logic
	return nil
}

// validateAssociatedTypeBound validates an associated type bound
func (atr *AssociatedTypeResolver) validateAssociatedTypeBound(assocBound *AssocTypeConstraint, context *resolver.ResolutionContext) error {
	// Resolve the associated type
	_, err := atr.ResolveAssociatedType(assocBound.TypeParam, assocBound.AssocType, context)
	if err != nil {
		return err
	}

	// Validate the trait bounds on the associated type
	for _, traitBound := range assocBound.TraitBounds {
		if err := atr.validateTraitBound(assocBound.AssocType, traitBound, context); err != nil {
			return err
		}
	}

	return nil
}

// validateEqualityBound validates an equality bound
func (atr *AssociatedTypeResolver) validateEqualityBound(eqBound *EqualityConstraint, context *resolver.ResolutionContext) error {
	// Resolve the associated type
	assocType, err := atr.ResolveAssociatedType(eqBound.TypeParam, eqBound.AssocType, context)
	if err != nil {
		return err
	}

	// Check if the types are compatible
	if !atr.tr.typeCompatible(assocType, eqBound.EqualTo) {
		return fmt.Errorf("type mismatch: %v != %v", assocType, eqBound.EqualTo)
	}

	return nil
}

// extractTypeBindings extracts type bindings from a constraint
func (atr *AssociatedTypeResolver) extractTypeBindings(constraint *WhereClauseConstraint, solution *ConstraintSolution, context *resolver.ResolutionContext) {
	// Extract equality bindings
	for _, eqBound := range constraint.EqualityBounds {
		key := fmt.Sprintf("%s::%s", eqBound.TypeParam, eqBound.AssocType)
		binding := &AssociatedTypeBinding{
			TypeName:     eqBound.AssocType,
			BoundType:    eqBound.EqualTo,
			IsProjection: false,
		}
		solution.AssocBindings[key] = binding
	}
}

// checkBindingConflicts checks for conflicting type bindings
func (atr *AssociatedTypeResolver) checkBindingConflicts(solution *ConstraintSolution) {
	// Check for conflicting associated type bindings
	seen := make(map[string]*AssociatedTypeBinding)
	for key, binding := range solution.AssocBindings {
		if existing, exists := seen[key]; exists {
			if !atr.tr.typesMatch(existing.BoundType, binding.BoundType) {
				solution.Conflicts = append(solution.Conflicts, ConstraintConflict{
					Reason: fmt.Sprintf("conflicting bindings for %s: %v vs %v", key, existing.BoundType, binding.BoundType),
				})
				solution.Satisfied = false
			}
		} else {
			seen[key] = binding
		}
	}
}

// createConstraintSetKey creates a cache key for a constraint set
func (atr *AssociatedTypeResolver) createConstraintSetKey(constraints []*WhereClauseConstraint) string {
	// Create a deterministic key from constraints
	var parts []string
	for _, constraint := range constraints {
		parts = append(parts, constraint.TypeParam)
		for _, bound := range constraint.TraitBounds {
			parts = append(parts, fmt.Sprintf("trait:%v", bound))
		}
		for _, assocBound := range constraint.AssocTypeBounds {
			parts = append(parts, fmt.Sprintf("assoc:%s::%s", assocBound.TypeParam, assocBound.AssocType))
		}
		for _, eqBound := range constraint.EqualityBounds {
			parts = append(parts, fmt.Sprintf("eq:%s::%s=%v", eqBound.TypeParam, eqBound.AssocType, eqBound.EqualTo))
		}
	}
	return fmt.Sprintf("constraints[%s]", fmt.Sprintf("%v", parts))
}

// isInScope checks if a scope applies to the current resolution context
func (atr *AssociatedTypeResolver) isInScope(scope string, context *resolver.ResolutionContext) bool {
	// Simplified scope checking - in a full implementation this would be more sophisticated
	return true
}

// constraintsEqual checks if two constraints are equal
func (atr *AssociatedTypeResolver) constraintsEqual(c1, c2 *WhereClauseConstraint) bool {
	return c1.TypeParam == c2.TypeParam && len(c1.TraitBounds) == len(c2.TraitBounds)
}
