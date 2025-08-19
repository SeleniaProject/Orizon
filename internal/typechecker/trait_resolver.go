package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
	"github.com/orizon-lang/orizon/internal/resolver"
)

// TraitResolver handles trait resolution and implementation checking
type TraitResolver struct {
	modules []*parser.HIRModule
	errors  []TraitError

	// Core data structures
	traits          []*parser.HIRTraitType
	implementations []*parser.HIRImpl

	// Advanced resolution system
	implCache    map[string][]*parser.HIRImpl
	scopeStack   []ResolutionScope
	genericCache map[string]*GenericBinding

	// Associated type resolution
	// Associated type resolution
	associatedTypeResolver *AssociatedTypeResolver

	// Exception specification checking
	throwsChecker *ThrowsSpecificationChecker
}

// ResolutionScope represents a scope for trait resolution priority
type ResolutionScope struct {
	ModuleID  string
	ScopeKind ScopeKind
	Priority  int
	Impls     []*parser.HIRImpl
}

// ScopeKind represents the kind of resolution scope
type ScopeKind int

const (
	ScopeKindGlobal ScopeKind = iota
	ScopeKindModule
	ScopeKindLocal
	ScopeKindInherent
)

// GenericBinding represents a binding for generic type parameters
type GenericBinding struct {
	ParamName   string
	BoundType   *parser.HIRType
	Constraints []*parser.HIRTraitType
}

// ImplCandidate represents a candidate implementation with priority
type ImplCandidate struct {
	Impl     *parser.HIRImpl
	Priority int
	Scope    ScopeKind
	Distance int // Distance from the call site
}

// TraitError represents trait resolution errors
type TraitError struct {
	Kind    string
	Message string
	Span    position.Span
}

func (e TraitError) Error() string { return e.Message }

// NewTraitResolver creates a new trait resolver
func NewTraitResolver(modules []*parser.HIRModule) *TraitResolver {
	tr := &TraitResolver{
		modules:      modules,
		errors:       []TraitError{},
		implCache:    make(map[string][]*parser.HIRImpl),
		scopeStack:   []ResolutionScope{},
		genericCache: make(map[string]*GenericBinding),
	}

	// Initialize the implementation cache
	tr.initImplCache()

	// Initialize associated type resolver
	tr.associatedTypeResolver = NewAssociatedTypeResolver(tr)

	// Initialize throws specification checker
	tr.throwsChecker = NewThrowsSpecificationChecker(tr)

	return tr
}

// initImplCache builds the implementation cache for efficient lookup
func (tr *TraitResolver) initImplCache() {
	for _, module := range tr.modules {
		for _, impl := range module.Impls {
			// Create cache key based on trait name or "inherent" for inherent impls
			var key string
			if impl.Trait != nil {
				if traitData, ok := impl.Trait.Data.(*parser.HIRTraitType); ok {
					key = traitData.Name
				}
			} else {
				key = "inherent"
			}

			tr.implCache[key] = append(tr.implCache[key], impl)
		}
	}
}

// ResolveImplementation finds the appropriate implementation for a trait method call
func (tr *TraitResolver) ResolveImplementation(traitType *parser.HIRType, targetType *parser.HIRType, methodName string) (*parser.HIRFunction, error) {
	// Find all candidate implementations
	candidates := tr.findAllCandidateImplementations(traitType, targetType)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no implementation found for trait %v on type %v", traitType, targetType)
	}

	// Resolve ambiguity using priority rules
	selectedImpl, err := tr.resolveImplementationAmbiguity(candidates, targetType)
	if err != nil {
		return nil, err
	}

	// Find the method in the selected implementation
	method := tr.findMethodInImpl(selectedImpl, methodName)
	if method == nil {
		return nil, fmt.Errorf("method %s not found in implementation", methodName)
	}

	return method, nil
}

// findAllCandidateImplementations finds all possible implementations for a trait and type
func (tr *TraitResolver) findAllCandidateImplementations(traitType *parser.HIRType, targetType *parser.HIRType) []*ImplCandidate {
	var candidates []*ImplCandidate

	// Get trait name for cache lookup
	var traitName string
	if traitType != nil {
		if trait := tr.findTrait(traitType); trait != nil {
			traitName = trait.Name
		}
	}

	// Look for trait implementations
	if traitName != "" {
		if impls, exists := tr.implCache[traitName]; exists {
			for _, impl := range impls {
				if tr.typeCompatible(impl.ForType, targetType) {
					candidate := &ImplCandidate{
						Impl:     impl,
						Priority: tr.calculateImplPriority(impl, targetType),
						Scope:    ScopeKindModule, // Default scope for trait impls
						Distance: tr.calculateDistance(impl, targetType),
					}
					candidates = append(candidates, candidate)
				}
			}
		}
	}

	// Look for inherent implementations (always have higher priority)
	if inherentImpls, exists := tr.implCache["inherent"]; exists {
		for _, impl := range inherentImpls {
			if tr.typeCompatible(impl.ForType, targetType) {
				candidate := &ImplCandidate{
					Impl:     impl,
					Priority: tr.calculateImplPriority(impl, targetType) + 1000, // Boost inherent impl priority
					Scope:    ScopeKindInherent,
					Distance: tr.calculateDistance(impl, targetType),
				}
				candidates = append(candidates, candidate)
			}
		}
	}

	return candidates
}

// resolveImplementationAmbiguity resolves ambiguity when multiple implementations are found
func (tr *TraitResolver) resolveImplementationAmbiguity(candidates []*ImplCandidate, targetType *parser.HIRType) (*parser.HIRImpl, error) {
	if len(candidates) == 1 {
		return candidates[0].Impl, nil
	}

	// Sort candidates by priority (higher priority first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if tr.compareCandidatePriority(candidates[i], candidates[j]) < 0 {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Check if the highest priority candidate is unambiguous
	highest := candidates[0]
	if len(candidates) > 1 && tr.compareCandidatePriority(highest, candidates[1]) == 0 {
		// Ambiguous - multiple candidates with same priority
		return nil, fmt.Errorf("ambiguous implementation: multiple candidates with equal priority for type %v", targetType)
	}

	return highest.Impl, nil
}

// compareCandidatePriority compares two implementation candidates
// Returns: 1 if a > b, -1 if a < b, 0 if equal
func (tr *TraitResolver) compareCandidatePriority(a, b *ImplCandidate) int {
	// First compare by scope priority (inherent > trait)
	if a.Scope != b.Scope {
		if a.Scope == ScopeKindInherent && b.Scope != ScopeKindInherent {
			return 1
		}
		if b.Scope == ScopeKindInherent && a.Scope != ScopeKindInherent {
			return -1
		}
	}

	// Compare by priority value
	if a.Priority != b.Priority {
		if a.Priority > b.Priority {
			return 1
		}
		return -1
	}

	// Compare by distance (closer is better)
	if a.Distance != b.Distance {
		if a.Distance < b.Distance {
			return 1
		}
		return -1
	}

	// Equal priority
	return 0
}

// calculateImplPriority calculates the priority of an implementation
func (tr *TraitResolver) calculateImplPriority(impl *parser.HIRImpl, targetType *parser.HIRType) int {
	priority := 0

	// Exact type match gets highest priority
	if tr.typesMatch(impl.ForType, targetType) {
		priority += 100
	}

	// Generic implementations get lower priority
	if tr.isGenericImpl(impl) {
		priority -= 10
	}

	// Local implementations get higher priority
	// (This would need module/scope context to implement fully)

	return priority
}

// calculateDistance calculates the "distance" between an implementation and target type
func (tr *TraitResolver) calculateDistance(impl *parser.HIRImpl, targetType *parser.HIRType) int {
	// Simple heuristic - exact match has distance 0
	if tr.typesMatch(impl.ForType, targetType) {
		return 0
	}

	// Generic implementations have higher distance
	if tr.isGenericImpl(impl) {
		return 2
	}

	// Default distance for compatible but not exact matches
	return 1
}

// typeCompatible checks if an impl type is compatible with the target type
func (tr *TraitResolver) typeCompatible(implType, targetType *parser.HIRType) bool {
	// For now, use the existing typesMatch function
	// In a full implementation, this would include generic parameter substitution
	return tr.typesMatch(implType, targetType) || tr.canUnifyTypes(implType, targetType)
}

// canUnifyTypes checks if two types can be unified (for generic types)
func (tr *TraitResolver) canUnifyTypes(type1, type2 *parser.HIRType) bool {
	// Simplified unification - in a full implementation this would handle:
	// - Generic type parameter substitution
	// - Variance rules
	// - Constraint checking

	if type1 == nil || type2 == nil {
		return false
	}

	// Same type kinds can potentially unify
	return type1.Kind == type2.Kind
}

// Scope management methods
func (tr *TraitResolver) pushScope(scope ResolutionScope) {
	tr.scopeStack = append(tr.scopeStack, scope)
}

func (tr *TraitResolver) popScope() {
	if len(tr.scopeStack) > 0 {
		tr.scopeStack = tr.scopeStack[:len(tr.scopeStack)-1]
	}
}

func (tr *TraitResolver) getCurrentScope() *ResolutionScope {
	if len(tr.scopeStack) == 0 {
		return nil
	}
	return &tr.scopeStack[len(tr.scopeStack)-1]
}

func (tr *TraitResolver) enterScope(kind ScopeKind) {
	scope := ResolutionScope{
		ScopeKind: kind,
		Priority:  len(tr.scopeStack) + 1, // Higher values for deeper scopes
	}
	tr.pushScope(scope)
}

func (tr *TraitResolver) exitScope() {
	tr.popScope()
}

// Generic binding methods
func (tr *TraitResolver) addGenericBinding(paramName string, paramType *parser.HIRType, boundType *parser.HIRType, constraints []string) {
	// Convert string constraints to HIRTraitType constraints
	var traitConstraints []*parser.HIRTraitType
	for _, constraint := range constraints {
		traitConstraints = append(traitConstraints, &parser.HIRTraitType{
			Name: constraint,
		})
	}

	binding := &GenericBinding{
		ParamName:   paramName,
		BoundType:   boundType,
		Constraints: traitConstraints,
	}
	tr.genericCache[paramName] = binding
}

func (tr *TraitResolver) lookupGenericBinding(paramName string) *GenericBinding {
	if binding, exists := tr.genericCache[paramName]; exists {
		return binding
	}
	return nil
}

func (tr *TraitResolver) clearGenericBindings() {
	tr.genericCache = make(map[string]*GenericBinding)
}

func (tr *TraitResolver) bindGeneric(name string, param *parser.HIRType, concrete *parser.HIRType) {
	binding := &GenericBinding{
		ParamName: name,
		BoundType: concrete,
	}
	tr.genericCache[name] = binding
}

// Associated type resolution methods

// ResolveAssociatedType resolves an associated type for a trait implementation
func (tr *TraitResolver) ResolveAssociatedType(typeParam string, assocTypeName string) (*parser.HIRType, error) {
	// Create a basic resolution context
	ctx := &resolver.ResolutionContext{
		Kind: resolver.ContextKindModule,
		Name: "associated_type_resolution",
	}
	return tr.associatedTypeResolver.ResolveAssociatedType(typeParam, assocTypeName, ctx)
}

// ValidateWhereClause validates where clause constraints
func (tr *TraitResolver) ValidateWhereClause(constraint *WhereClauseConstraint) error {
	// Create a basic resolution context
	ctx := &resolver.ResolutionContext{
		Kind: resolver.ContextKindModule,
		Name: "where_clause_validation",
	}
	conflicts := tr.associatedTypeResolver.ValidateWhereClause(constraint, ctx)
	if len(conflicts) > 0 {
		return fmt.Errorf("where clause validation failed: %d conflicts found", len(conflicts))
	}
	return nil
}

// SolveAssociatedTypeConstraints solves associated type constraints
func (tr *TraitResolver) SolveAssociatedTypeConstraints(scope string) (*ConstraintSolution, error) {
	// Get constraints for the scope
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

	// Create resolution context
	ctx := &resolver.ResolutionContext{
		Kind: resolver.ContextKindModule,
		Name: scope,
	}

	solution := tr.associatedTypeResolver.SolveConstraintSet(constraints, ctx)
	return solution, nil
}

// Exception specification checking methods

// CheckTraitThrowsConsistency checks exception specification consistency for a trait
func (tr *TraitResolver) CheckTraitThrowsConsistency(trait *parser.HIRTraitType) error {
	return tr.throwsChecker.CheckTraitThrowsConsistency(trait)
}

// CheckImplThrowsConsistency checks that implementation methods conform to trait exception specifications
func (tr *TraitResolver) CheckImplThrowsConsistency(impl *parser.HIRImpl, trait *parser.HIRTraitType) error {
	return tr.throwsChecker.CheckImplThrowsConsistency(impl, trait)
}

// CheckWhereClauseThrowsConsistency checks exception specifications in where clauses
func (tr *TraitResolver) CheckWhereClauseThrowsConsistency(whereClause *WhereClauseConstraint) error {
	return tr.throwsChecker.CheckWhereClauseThrowsConsistency(whereClause)
}

// GetThrowsViolations returns all accumulated throws violations
func (tr *TraitResolver) GetThrowsViolations() []ThrowsViolation {
	return tr.throwsChecker.GetViolations()
}

// HasThrowsViolations checks if there are any throws violations
func (tr *TraitResolver) HasThrowsViolations() bool {
	return tr.throwsChecker.HasViolations()
}

// isGenericImpl checks if an implementation is generic
func (tr *TraitResolver) isGenericImpl(impl *parser.HIRImpl) bool {
	// Check if the impl has generic parameters
	// This would need to be determined from the impl's type parameters
	// For now, return false as a placeholder
	return false
}

// CheckImplementationConstraints verifies that an impl block satisfies all trait requirements
func (tr *TraitResolver) CheckImplementationConstraints(impl *parser.HIRImpl) []TraitError {
	var errors []TraitError

	if impl.Trait == nil {
		// Inherent implementation - no trait constraints to check
		return errors
	}

	// Find the trait being implemented
	trait := tr.findTrait(impl.Trait)
	if trait == nil {
		errors = append(errors, TraitError{
			Kind:    "ImplResolution",
			Message: fmt.Sprintf("trait not found: %v", impl.Trait),
		})
		return errors
	}

	// Check that all required methods are implemented
	for _, traitMethod := range trait.Methods {
		implMethod := tr.findMethodInImpl(impl, traitMethod.Name)
		if implMethod == nil {
			errors = append(errors, TraitError{
				Kind:    "ImplResolution",
				Message: fmt.Sprintf("missing implementation for method %s", traitMethod.Name),
			})
			continue
		}

		// Check method signature compatibility
		if err := tr.checkMethodSignatureCompatibility(traitMethod, implMethod); err != nil {
			errors = append(errors, TraitError{
				Kind:    "ImplResolution",
				Message: fmt.Sprintf("method %s signature mismatch: %v", traitMethod.Name, err),
			})
		}
	}

	return errors
}

// findTrait locates a trait definition by type
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

// findImplementation locates an impl block for a trait and target type
func (tr *TraitResolver) findImplementation(trait *parser.HIRTraitType, targetType *parser.HIRType) *parser.HIRImpl {
	for _, module := range tr.modules {
		for _, impl := range module.Impls {
			if impl.Trait != nil && impl.Kind == parser.HIRImplTrait {
				if tr.typesMatch(impl.ForType, targetType) && tr.traitTypesMatch(impl.Trait, trait) {
					return impl
				}
			}
		}
	}
	return nil
}

// findMethodInImpl locates a method by name within an impl block
func (tr *TraitResolver) findMethodInImpl(impl *parser.HIRImpl, methodName string) *parser.HIRFunction {
	for _, method := range impl.Methods {
		if method.Name == methodName {
			return method
		}
	}
	return nil
}

// checkMethodSignatureCompatibility verifies trait method and impl method signatures match
func (tr *TraitResolver) checkMethodSignatureCompatibility(traitMethod *parser.HIRMethodSignature, implMethod *parser.HIRFunction) error {
	// Check parameter count
	if len(traitMethod.Parameters) != len(implMethod.Parameters) {
		return fmt.Errorf("parameter count mismatch: trait has %d, impl has %d",
			len(traitMethod.Parameters), len(implMethod.Parameters))
	}

	// Check parameter types
	for i, traitParam := range traitMethod.Parameters {
		implParam := implMethod.Parameters[i]
		if !tr.typesMatch(traitParam, implParam.Type) {
			return fmt.Errorf("parameter %d type mismatch: trait expects %v, impl has %v",
				i, traitParam, implParam.Type)
		}
	}

	// Check return type
	if !tr.typesMatch(traitMethod.ReturnType, implMethod.ReturnType) {
		return fmt.Errorf("return type mismatch: trait expects %v, impl has %v",
			traitMethod.ReturnType, implMethod.ReturnType)
	}

	return nil
}

// typesMatch checks if two HIR types are equivalent
func (tr *TraitResolver) typesMatch(type1, type2 *parser.HIRType) bool {
	if type1 == nil && type2 == nil {
		return true
	}
	if type1 == nil || type2 == nil {
		return false
	}

	// Basic structural matching - would need more sophisticated logic for generics
	if type1.Kind != type2.Kind {
		return false
	}

	// Compare Data fields instead of pointer addresses
	return fmt.Sprintf("%v", type1.Data) == fmt.Sprintf("%v", type2.Data)
}

// traitTypesMatch checks if trait types match
func (tr *TraitResolver) traitTypesMatch(implTrait *parser.HIRType, targetTrait *parser.HIRTraitType) bool {
	if implTrait.Kind != parser.HIRTypeTrait {
		return false
	}

	// Extract trait type from implTrait.Data
	if implTraitData, ok := implTrait.Data.(*parser.HIRTraitType); ok {
		return implTraitData.Name == targetTrait.Name
	}

	return false
} // GetErrors returns all accumulated errors
func (tr *TraitResolver) GetErrors() []TraitError {
	return tr.errors
}
