package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// ThrowsSpecificationChecker handles exception specification consistency across traits, impls, and bounds
type ThrowsSpecificationChecker struct {
	traitResolver    *TraitResolver
	throwsViolations []ThrowsViolation
	throwsCache      map[string]*ThrowsSpecification
	constraintStack  []*ThrowsConstraint
}

// ThrowsSpecification represents exception specifications for methods
type ThrowsSpecification struct {
	ExceptionTypes []*parser.HIRType   // List of exception types that can be thrown
	IsNoThrow      bool                // Whether this method throws no exceptions
	IsPure         bool                // Whether this method is pure (no side effects)
	Constraints    []*ThrowsConstraint // Generic constraints on thrown types
	Position       position.Position   // Position in source code
}

// ThrowsConstraint represents a constraint on exception types in generics
type ThrowsConstraint struct {
	TypeParam      string                 // Generic type parameter
	BoundedTypes   []*parser.HIRType      // Types that the parameter can throw
	RequiredTraits []*parser.HIRTraitType // Traits required for exception types
	Position       position.Position      // Position in source code
}

// ThrowsViolation represents a violation of exception specification consistency
type ThrowsViolation struct {
	Type        ThrowsViolationType
	TraitMethod *parser.HIRFunction  // Method in trait definition
	ImplMethod  *parser.HIRFunction  // Method in implementation
	Expected    *ThrowsSpecification // Expected exception specification
	Actual      *ThrowsSpecification // Actual exception specification
	Constraint  *ThrowsConstraint    // Related constraint (if applicable)
	Message     string               // Detailed error message
	Position    position.Position    // Position of the violation
}

// ThrowsViolationType represents different types of throws violations
type ThrowsViolationType int

const (
	ThrowsViolationNone                  ThrowsViolationType = iota
	ThrowsViolationExtraException                            // Implementation throws more than trait allows
	ThrowsViolationMissingException                          // Implementation doesn't throw what trait requires
	ThrowsViolationIncompatibleException                     // Exception types are incompatible
	ThrowsViolationConstraintViolation                       // Generic constraint violated
	ThrowsViolationNoThrowViolation                          // NoThrow contract violated
	ThrowsViolationPurityViolation                           // Purity contract violated
)

// TraitMethodThrowsValidator validates exception specifications in trait methods
type TraitMethodThrowsValidator struct {
	checker *ThrowsSpecificationChecker
}

// ImplThrowsValidator validates that impl methods conform to trait exception specifications
type ImplThrowsValidator struct {
	checker *ThrowsSpecificationChecker
}

// WhereClauseThrowsValidator validates exception specifications in where clauses
type WhereClauseThrowsValidator struct {
	checker *ThrowsSpecificationChecker
}

// NewThrowsSpecificationChecker creates a new throws specification checker
func NewThrowsSpecificationChecker(traitResolver *TraitResolver) *ThrowsSpecificationChecker {
	return &ThrowsSpecificationChecker{
		traitResolver:    traitResolver,
		throwsViolations: make([]ThrowsViolation, 0),
		throwsCache:      make(map[string]*ThrowsSpecification),
		constraintStack:  make([]*ThrowsConstraint, 0),
	}
}

// CheckTraitThrowsConsistency checks exception specification consistency for a trait
func (tsc *ThrowsSpecificationChecker) CheckTraitThrowsConsistency(trait *parser.HIRTraitType) error {
	validator := &TraitMethodThrowsValidator{checker: tsc}

	for _, method := range trait.Methods {
		if err := validator.ValidateTraitMethod(trait, method); err != nil {
			return err
		}
	}

	return nil
}

// CheckImplThrowsConsistency checks that implementation methods conform to trait exception specifications
func (tsc *ThrowsSpecificationChecker) CheckImplThrowsConsistency(impl *parser.HIRImpl, trait *parser.HIRTraitType) error {
	validator := &ImplThrowsValidator{checker: tsc}

	for _, implMethod := range impl.Methods {
		// Find corresponding trait method
		traitMethod := tsc.findTraitMethod(trait, implMethod.Name)
		if traitMethod == nil {
			continue // Method not in trait (inherent impl)
		}

		if err := validator.ValidateImplMethod(traitMethod, implMethod, impl); err != nil {
			return err
		}
	}

	return nil
}

// CheckWhereClauseThrowsConsistency checks exception specifications in where clauses
func (tsc *ThrowsSpecificationChecker) CheckWhereClauseThrowsConsistency(whereClause *WhereClauseConstraint) error {
	validator := &WhereClauseThrowsValidator{checker: tsc}
	return validator.ValidateWhereClause(whereClause)
}

// GetThrowsSpecification extracts or computes throws specification for a method
func (tsc *ThrowsSpecificationChecker) GetThrowsSpecification(method *parser.HIRFunction) *ThrowsSpecification {
	// Create cache key
	cacheKey := fmt.Sprintf("%s::%p", method.Name, method)

	// Check cache first
	if spec, exists := tsc.throwsCache[cacheKey]; exists {
		return spec
	}

	// Extract throws specification from method
	spec := tsc.extractThrowsSpecification(method)

	// Cache the result
	tsc.throwsCache[cacheKey] = spec

	return spec
}

// ValidateTraitMethod validates exception specifications in a trait method
func (tmv *TraitMethodThrowsValidator) ValidateTraitMethod(trait *parser.HIRTraitType, method *parser.HIRMethodSignature) error {
	// Convert HIRMethodSignature to HIRFunction for consistency
	// Since HIRMethodSignature has []*HIRType for parameters, we need to convert them
	var parameters []*parser.HIRParameter
	for i, paramType := range method.Parameters {
		parameters = append(parameters, &parser.HIRParameter{
			Name: fmt.Sprintf("param%d", i), // Generate parameter names
			Type: paramType,
		})
	}

	methodFunc := &parser.HIRFunction{
		Name:           method.Name,
		Parameters:     parameters,
		ReturnType:     method.ReturnType,
		TypeParameters: method.TypeParameters,
	}

	spec := tmv.checker.GetThrowsSpecification(methodFunc)

	// Validate that all exception types are valid
	for _, exceptionType := range spec.ExceptionTypes {
		if !tmv.isValidExceptionType(exceptionType) {
			violation := ThrowsViolation{
				Type:        ThrowsViolationIncompatibleException,
				TraitMethod: methodFunc,
				Expected:    spec,
				Actual:      spec,
				Message:     fmt.Sprintf("Invalid exception type %s in trait method %s", exceptionType.String(), method.Name),
				Position:    spec.Position,
			}
			tmv.checker.throwsViolations = append(tmv.checker.throwsViolations, violation)
			return fmt.Errorf("invalid exception type in trait method: %s", exceptionType.String())
		}
	}

	return nil
}

// ValidateImplMethod validates that an implementation method conforms to trait exception specifications
func (imv *ImplThrowsValidator) ValidateImplMethod(traitMethod *parser.HIRFunction, implMethod *parser.HIRFunction, impl *parser.HIRImpl) error {
	traitSpec := imv.checker.GetThrowsSpecification(traitMethod)
	implSpec := imv.checker.GetThrowsSpecification(implMethod)

	// Check NoThrow consistency
	if traitSpec.IsNoThrow && !implSpec.IsNoThrow {
		return imv.createViolation(ThrowsViolationNoThrowViolation, traitMethod, implMethod, traitSpec, implSpec,
			fmt.Sprintf("Implementation method %s throws exceptions but trait method is marked NoThrow", implMethod.Name))
	}

	// Check purity consistency
	if traitSpec.IsPure && !implSpec.IsPure {
		return imv.createViolation(ThrowsViolationPurityViolation, traitMethod, implMethod, traitSpec, implSpec,
			fmt.Sprintf("Implementation method %s has side effects but trait method is marked pure", implMethod.Name))
	}

	// Check exception type compatibility
	if err := imv.checkExceptionTypeCompatibility(traitSpec, implSpec, traitMethod, implMethod); err != nil {
		return err
	}

	// Check generic constraints
	if err := imv.checkThrowsConstraints(traitSpec, implSpec, traitMethod, implMethod); err != nil {
		return err
	}

	return nil
}

// ValidateWhereClause validates exception specifications in where clauses
func (wcv *WhereClauseThrowsValidator) ValidateWhereClause(whereClause *WhereClauseConstraint) error {
	// Check if any trait bounds have exception specifications
	for _, traitBound := range whereClause.TraitBounds {
		if err := wcv.validateTraitBoundThrows(traitBound, whereClause); err != nil {
			return err
		}
	}

	// Check associated type bounds for exception consistency
	for _, assocBound := range whereClause.AssocTypeBounds {
		if err := wcv.validateAssocTypeBoundThrows(assocBound, whereClause); err != nil {
			return err
		}
	}

	return nil
}

// Private helper methods

func (tsc *ThrowsSpecificationChecker) extractThrowsSpecification(method *parser.HIRFunction) *ThrowsSpecification {
	spec := &ThrowsSpecification{
		ExceptionTypes: make([]*parser.HIRType, 0),
		IsNoThrow:      false,
		IsPure:         false,
		Constraints:    make([]*ThrowsConstraint, 0),
	}

	// Extract throws information from method attributes
	for _, attr := range method.Attributes {
		switch attr {
		case "nothrow":
			spec.IsNoThrow = true
		case "pure":
			spec.IsPure = true
		}
	}

	// For now, we assume methods can throw any exception unless marked nothrow
	// In a full implementation, this would analyze the method body and function signature
	if !spec.IsNoThrow {
		// Add a generic exception type if not explicitly no-throw
		genericException := &parser.HIRType{
			Kind: parser.HIRTypeStruct,
			Data: "Exception",
		}
		spec.ExceptionTypes = append(spec.ExceptionTypes, genericException)
	}

	return spec
}

func (tsc *ThrowsSpecificationChecker) findTraitMethod(trait *parser.HIRTraitType, methodName string) *parser.HIRFunction {
	for _, method := range trait.Methods {
		if method.Name == methodName {
			// Convert HIRMethodSignature to HIRFunction
			var parameters []*parser.HIRParameter
			for i, paramType := range method.Parameters {
				parameters = append(parameters, &parser.HIRParameter{
					Name: fmt.Sprintf("param%d", i),
					Type: paramType,
				})
			}

			return &parser.HIRFunction{
				Name:           method.Name,
				Parameters:     parameters,
				ReturnType:     method.ReturnType,
				TypeParameters: method.TypeParameters,
			}
		}
	}
	return nil
}

func (tmv *TraitMethodThrowsValidator) isValidExceptionType(exceptionType *parser.HIRType) bool {
	// Check if the type is a valid exception type
	// This would integrate with the type system to verify:
	// 1. Type implements the Exception trait (if such exists)
	// 2. Type is a valid throwable type
	// For now, allow all types
	return true
}

func (imv *ImplThrowsValidator) createViolation(violationType ThrowsViolationType, traitMethod, implMethod *parser.HIRFunction,
	traitSpec, implSpec *ThrowsSpecification, message string) error {

	violation := ThrowsViolation{
		Type:        violationType,
		TraitMethod: traitMethod,
		ImplMethod:  implMethod,
		Expected:    traitSpec,
		Actual:      implSpec,
		Message:     message,
		Position:    implSpec.Position,
	}

	imv.checker.throwsViolations = append(imv.checker.throwsViolations, violation)
	return fmt.Errorf("throws specification violation: %s", message)
}

func (imv *ImplThrowsValidator) checkExceptionTypeCompatibility(traitSpec, implSpec *ThrowsSpecification,
	traitMethod, implMethod *parser.HIRFunction) error {

	// Implementation can throw a subset of what trait allows (covariance)
	for _, implException := range implSpec.ExceptionTypes {
		if !imv.isExceptionTypeAllowed(implException, traitSpec.ExceptionTypes) {
			return imv.createViolation(ThrowsViolationExtraException, traitMethod, implMethod, traitSpec, implSpec,
				fmt.Sprintf("Implementation throws %s which is not allowed by trait specification", implException.String()))
		}
	}

	return nil
}

func (imv *ImplThrowsValidator) isExceptionTypeAllowed(exceptionType *parser.HIRType, allowedTypes []*parser.HIRType) bool {
	for _, allowedType := range allowedTypes {
		if imv.checker.traitResolver.typesMatch(exceptionType, allowedType) {
			return true
		}
		// Check if exceptionType is a subtype of allowedType
		if imv.isSubtypeOf(exceptionType, allowedType) {
			return true
		}
	}
	return false
}

func (imv *ImplThrowsValidator) isSubtypeOf(subtype, supertype *parser.HIRType) bool {
	// Simplified subtype checking
	// In a full implementation, this would check inheritance hierarchy
	return imv.checker.traitResolver.typesMatch(subtype, supertype)
}

func (imv *ImplThrowsValidator) checkThrowsConstraints(traitSpec, implSpec *ThrowsSpecification,
	traitMethod, implMethod *parser.HIRFunction) error {

	// Check that implementation satisfies all throws constraints from trait
	for _, traitConstraint := range traitSpec.Constraints {
		if !imv.implSatisfiesThrowsConstraint(traitConstraint, implSpec) {
			return imv.createViolation(ThrowsViolationConstraintViolation, traitMethod, implMethod, traitSpec, implSpec,
				fmt.Sprintf("Implementation does not satisfy throws constraint for type parameter %s", traitConstraint.TypeParam))
		}
	}

	return nil
}

func (imv *ImplThrowsValidator) implSatisfiesThrowsConstraint(constraint *ThrowsConstraint, implSpec *ThrowsSpecification) bool {
	// Check if implementation's exception types satisfy the constraint
	// This is a simplified check - full implementation would involve complex constraint solving

	// If constraint requires no exceptions for this type parameter, check that impl doesn't throw
	if len(constraint.BoundedTypes) == 0 {
		return len(implSpec.ExceptionTypes) == 0
	}

	// Check that all thrown types are within the bounded types
	for _, thrownType := range implSpec.ExceptionTypes {
		allowed := false
		for _, boundedType := range constraint.BoundedTypes {
			if imv.checker.traitResolver.typesMatch(thrownType, boundedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

func (wcv *WhereClauseThrowsValidator) validateTraitBoundThrows(traitBound *parser.HIRType, whereClause *WhereClauseConstraint) error {
	// Look up the trait and check its methods for exception specifications
	trait := wcv.findTraitByType(traitBound)
	if trait == nil {
		return nil // Not a trait bound
	}

	// Validate that any exception specifications in the trait are consistent
	return wcv.checker.CheckTraitThrowsConsistency(trait)
}

func (wcv *WhereClauseThrowsValidator) validateAssocTypeBoundThrows(assocBound *AssocTypeConstraint, whereClause *WhereClauseConstraint) error {
	// Check if associated type bounds involve exception specifications
	for _, traitBound := range assocBound.TraitBounds {
		if err := wcv.validateTraitBoundThrows(traitBound, whereClause); err != nil {
			return err
		}
	}
	return nil
}

func (wcv *WhereClauseThrowsValidator) findTraitByType(traitType *parser.HIRType) *parser.HIRTraitType {
	// This would integrate with the trait resolver to find trait definitions
	// For now, return nil as a placeholder
	return nil
}

// GetViolations returns all accumulated throws violations
func (tsc *ThrowsSpecificationChecker) GetViolations() []ThrowsViolation {
	return tsc.throwsViolations
}

// ClearViolations clears all accumulated violations
func (tsc *ThrowsSpecificationChecker) ClearViolations() {
	tsc.throwsViolations = make([]ThrowsViolation, 0)
}

// HasViolations checks if there are any throws violations
func (tsc *ThrowsSpecificationChecker) HasViolations() bool {
	return len(tsc.throwsViolations) > 0
}
