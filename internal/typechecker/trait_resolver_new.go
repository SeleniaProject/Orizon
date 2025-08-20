package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
	"github.com/orizon-lang/orizon/internal/resolver"
)

// TraitResolver handles trait resolution and associated type inference.
type TraitResolver struct {
	modules                []*parser.HIRModule
	currentScope           ScopeStack
	associatedTypeResolver *AssociatedTypeResolver
	throwsChecker          *TraitThrowsChecker
	errors                 []TraitError
}

// ResolutionScope represents a scope for trait resolution priority.
type ResolutionScope struct {
	Kind     ScopeKind
	Priority int
	Generics map[string]*GenericBinding
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

// ScopeStack manages resolution scope stack.
type ScopeStack []ResolutionScope

// Push adds a scope to the stack.
func (s *ScopeStack) Push(scope ResolutionScope) {
	*s = append(*s, scope)
}

// Pop removes the top scope from the stack.
func (s *ScopeStack) Pop() {
	if len(*s) > 0 {
		*s = (*s)[:len(*s)-1]
	}
}

// Top returns the top scope without removing it.
func (s *ScopeStack) Top() *ResolutionScope {
	if len(*s) == 0 {
		return nil
	}
	return &(*s)[len(*s)-1]
}

// GenericBinding represents a binding between a generic parameter and a concrete type.
type GenericBinding struct {
	Parameter    *parser.HIRTypeParameter
	ConcreteType *parser.HIRType
	Constraints  []*parser.HIRTraitType
	Position     position.Position
	BindingScope string
}

// TraitError represents an error in trait resolution.
type TraitError struct {
	Kind     string
	Message  string
	Position position.Position
}

// Error implements the error interface.
func (e *TraitError) Error() string { return e.Message }

// ImplCandidate represents a candidate implementation.
type ImplCandidate struct {
	Impl     *parser.HIRImpl
	Priority int
	Generics []*GenericBinding
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

// ResolveImplementation finds the implementation for a trait method.
func (tr *TraitResolver) ResolveImplementation(traitType *parser.HIRType,
	targetType *parser.HIRType, methodName string) (*parser.HIRFunction, error) {

	// Find candidates.
	candidates := tr.findAllCandidateImplementations(traitType, targetType)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no implementation found for trait %v on type %v",
			traitType, targetType)
	}

	// Resolve ambiguity.
	impl, err := tr.resolveImplementationAmbiguity(candidates, targetType)
	if err != nil {
		return nil, err
	}

	// Find method in implementation.
	for _, method := range impl.Methods {
		if method.Name == methodName {
			return method, nil
		}
	}

	return nil, fmt.Errorf("method %s not found in implementation", methodName)
}

// findAllCandidateImplementations finds all possible implementations.
func (tr *TraitResolver) findAllCandidateImplementations(traitType *parser.HIRType,
	targetType *parser.HIRType) []*ImplCandidate {

	var candidates []*ImplCandidate

	for _, module := range tr.modules {
		for _, impl := range module.Impls {
			if impl.Kind == parser.HIRImplTrait && impl.Trait != nil {
				// Check if trait matches.
				if tr.traitMatches(impl.Trait, traitType) {
					// Check if target type matches.
					if tr.typeMatches(impl.ForType, targetType) {
						candidate := &ImplCandidate{
							Impl:     impl,
							Priority: tr.calculatePriority(impl, targetType),
						}
						candidates = append(candidates, candidate)
					}
				}
			}
		}
	}

	return candidates
}

// calculatePriority calculates implementation priority.
func (tr *TraitResolver) calculatePriority(impl *parser.HIRImpl, targetType *parser.HIRType) int {
	priority := 100

	// Exact type matches get higher priority.
	if tr.isExactTypeMatch(impl.ForType, targetType) {
		priority += 50
	}

	// Generic implementations get lower priority.
	if tr.isGenericImpl(impl) {
		priority -= 25
	}

	return priority
}

// resolveImplementationAmbiguity resolves multiple candidates.
func (tr *TraitResolver) resolveImplementationAmbiguity(candidates []*ImplCandidate,
	targetType *parser.HIRType) (*parser.HIRImpl, error) {

	if len(candidates) == 1 {
		return candidates[0].Impl, nil
	}

	// Find highest priority candidate.
	var bestCandidate *ImplCandidate
	for _, candidate := range candidates {
		if bestCandidate == nil || candidate.Priority > bestCandidate.Priority {
			bestCandidate = candidate
		} else if candidate.Priority == bestCandidate.Priority {
			return nil, fmt.Errorf("ambiguous implementation: multiple candidates "+
				"with equal priority for type %v", targetType)
		}
	}

	return bestCandidate.Impl, nil
}

// traitMatches checks if traits match.
func (tr *TraitResolver) traitMatches(implTrait, targetTrait *parser.HIRType) bool {
	if implTrait.Kind != targetTrait.Kind {
		return false
	}
	return fmt.Sprintf("%v", implTrait.Data) == fmt.Sprintf("%v", targetTrait.Data)
}

// typeMatches checks if types match.
func (tr *TraitResolver) typeMatches(implType, targetType *parser.HIRType) bool {
	if implType.Kind != targetType.Kind {
		return false
	}
	return fmt.Sprintf("%v", implType.Data) == fmt.Sprintf("%v", targetType.Data)
}

// isExactTypeMatch checks for exact type match.
func (tr *TraitResolver) isExactTypeMatch(implType, targetType *parser.HIRType) bool {
	return tr.typeMatches(implType, targetType)
}

// addGenericBinding adds a generic parameter binding.
func (tr *TraitResolver) addGenericBinding(paramName string, paramType *parser.HIRType,
	boundType *parser.HIRType, constraints []string) {

	traitConstraints := make([]*parser.HIRTraitType, 0, len(constraints))
	for _, constraintName := range constraints {
		traitType := &parser.HIRTraitType{
			Name:     constraintName,
			Position: position.Position{Line: 1, Column: 1},
		}
		traitConstraints = append(traitConstraints, traitType)
	}

	binding := &GenericBinding{
		Parameter: &parser.HIRTypeParameter{
			Name:     paramName,
			Bounds:   traitConstraints,
			Position: position.Position{Line: 1, Column: 1},
		},
		ConcreteType: boundType,
		Position:     position.Position{Line: 1, Column: 1},
		BindingScope: "current",
	}

	// Add to current scope.
	if scope := tr.currentScope.Top(); scope != nil {
		if scope.Generics == nil {
			scope.Generics = make(map[string]*GenericBinding)
		}
		scope.Generics[paramName] = binding
	}
}

// ResolveAssociatedType resolves an associated type.
func (tr *TraitResolver) ResolveAssociatedType(typeParam, assocTypeName string) (*parser.HIRType, error) {
	// Create basic associated type.
	return &parser.HIRType{
		Kind: parser.HIRTypeAssociated,
		Data: map[string]interface{}{
			"type_param": typeParam,
			"assoc_type": assocTypeName,
		},
	}, nil
}

// GetWhereClauseConstraints returns where clause constraints for a scope.
func (tr *TraitResolver) GetWhereClauseConstraints(scope string) []*WhereClauseConstraint {
	return tr.associatedTypeResolver.GetWhereClauseConstraints(scope)
}

// AddWhereClauseConstraint adds a where clause constraint.
func (tr *TraitResolver) AddWhereClauseConstraint(constraint *WhereClauseConstraint) {
	tr.associatedTypeResolver.AddWhereClauseConstraint(constraint)
}

// ValidateWhereClause validates where clause constraints.
func (tr *TraitResolver) ValidateWhereClause(constraint *WhereClauseConstraint) error {
	// Create a basic resolution context.
	ctx := &resolver.ResolutionContext{
		Kind: resolver.ContextKindModule,
		Name: "where_clause_validation",
	}

	conflicts := tr.associatedTypeResolver.ValidateWhereClause(constraint, ctx)
	if len(conflicts) > 0 {
		return fmt.Errorf("where clause validation failed: %d conflicts found",
			len(conflicts))
	}

	return nil
}

// SolveAssociatedTypeConstraints solves associated type constraints.
func (tr *TraitResolver) SolveAssociatedTypeConstraints(scope string) (*ConstraintSolution, error) {
	// Get constraints for the scope.
	constraints := tr.associatedTypeResolver.whereConstraints[scope]
	if constraints == nil {
		return &ConstraintSolution{
			TypeBindings:  make(map[string]*parser.HIRType),
			AssocBindings: make(map[string]*AssociatedTypeBinding),
			TraitImpls:    make(map[string]*parser.HIRImpl),
			Satisfied:     true,
			Conflicts:     []ConstraintConflict{},
		}, nil
	}

	// Create resolution context.
	ctx := &resolver.ResolutionContext{
		Kind: resolver.ContextKindModule,
		Name: scope,
	}

	solution := tr.associatedTypeResolver.SolveConstraintSet(constraints, ctx)

	return solution, nil
}

// Exception specification checking methods.

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

// isGenericImpl checks if an implementation is generic.
func (tr *TraitResolver) isGenericImpl(impl *parser.HIRImpl) bool {
	// Check if the impl has generic parameters.
	// This would need to be determined from the impl's type parameters.
	// For now, return false as a placeholder.
	return false
}

// CheckImplementationConstraints verifies that an impl block satisfies all
// trait requirements.
func (tr *TraitResolver) CheckImplementationConstraints(impl *parser.HIRImpl) []TraitError {
	var errors []TraitError

	if impl.Trait == nil {
		// Inherent implementation - no trait constraints to check.
		return errors
	}

	// Find the trait being implemented.
	trait := tr.findTrait(impl.Trait)
	if trait == nil {
		errors = append(errors, TraitError{
			Kind:    "ImplResolution",
			Message: fmt.Sprintf("trait not found: %v", impl.Trait),
		})

		return errors
	}

	// Check that all required methods are implemented.
	for _, traitMethod := range trait.Methods {
		implMethod := tr.findMethodInImpl(impl, traitMethod.Name)
		if implMethod == nil {
			errors = append(errors, TraitError{
				Kind: "ImplResolution",
				Message: fmt.Sprintf("missing implementation for method %s",
					traitMethod.Name),
			})

			continue
		}

		// Check method signature compatibility.
		if err := tr.checkMethodSignatureCompatibility(traitMethod, implMethod); err != nil {
			errors = append(errors, TraitError{
				Kind: "ImplResolution",
				Message: fmt.Sprintf("method %s signature mismatch: %v",
					traitMethod.Name, err),
			})
		}
	}

	return errors
}

// findTrait locates a trait definition by type.
func (tr *TraitResolver) findTrait(traitType *parser.HIRType) *parser.HIRTraitType {
	if traitType.Kind != parser.HIRTypeTrait {
		return nil
	}

	// Extract trait type from HIRType.Data
	if traitData, ok := traitType.Data.(*parser.HIRTraitType); ok {
		return traitData
	}

	return nil
}

// findMethodInImpl locates a method by name within an impl block.
func (tr *TraitResolver) findMethodInImpl(impl *parser.HIRImpl, methodName string) *parser.HIRFunction {
	for _, method := range impl.Methods {
		if method.Name == methodName {
			return method
		}
	}

	return nil
}

// checkMethodSignatureCompatibility verifies trait method and impl method
// signatures match.
func (tr *TraitResolver) checkMethodSignatureCompatibility(traitMethod *parser.HIRMethodSignature,
	implMethod *parser.HIRFunction) error {

	// Check parameter count.
	if len(traitMethod.Parameters) != len(implMethod.Parameters) {
		return fmt.Errorf("parameter count mismatch: trait has %d, impl has %d",
			len(traitMethod.Parameters), len(implMethod.Parameters))
	}

	// Check parameter types.
	for i, traitParam := range traitMethod.Parameters {
		implParam := implMethod.Parameters[i]
		if !tr.typesMatch(traitParam, implParam.Type) {
			return fmt.Errorf("parameter %d type mismatch: trait expects %v, impl has %v",
				i, traitParam, implParam.Type)
		}
	}

	// Check return type.
	if !tr.typesMatch(traitMethod.ReturnType, implMethod.ReturnType) {
		return fmt.Errorf("return type mismatch: trait expects %v, impl has %v",
			traitMethod.ReturnType, implMethod.ReturnType)
	}

	return nil
}

// typesMatch checks if two HIR types are equivalent.
func (tr *TraitResolver) typesMatch(type1, type2 *parser.HIRType) bool {
	if type1 == nil && type2 == nil {
		return true
	}

	if type1 == nil || type2 == nil {
		return false
	}

	// Basic structural matching - would need more sophisticated logic for generics.
	if type1.Kind != type2.Kind {
		return false
	}

	// Compare Data fields instead of pointer addresses.
	return fmt.Sprintf("%v", type1.Data) == fmt.Sprintf("%v", type2.Data)
}

// GetErrors returns all accumulated errors
func (tr *TraitResolver) GetErrors() []TraitError {
	return tr.errors
}
