package typechecker

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/parser"
	"github.com/orizon-lang/orizon/internal/position"
)

// TraitResolver handles trait resolution and implementation checking
type TraitResolver struct {
	modules []*parser.HIRModule
	errors  []TraitError
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
	return &TraitResolver{
		modules: modules,
		errors:  []TraitError{},
	}
}

// ResolveImplementation finds the appropriate implementation for a trait method call
func (tr *TraitResolver) ResolveImplementation(traitType *parser.HIRType, targetType *parser.HIRType, methodName string) (*parser.HIRFunction, error) {
	// Find the trait definition
	trait := tr.findTrait(traitType)
	if trait == nil {
		return nil, fmt.Errorf("trait not found: %v", traitType)
	}

	// Find implementation for the target type
	impl := tr.findImplementation(trait, targetType)
	if impl == nil {
		return nil, fmt.Errorf("no implementation found for trait %v on type %v", traitType, targetType)
	}

	// Find the method in the implementation
	method := tr.findMethodInImpl(impl, methodName)
	if method == nil {
		return nil, fmt.Errorf("method %s not found in implementation", methodName)
	}

	return method, nil
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
