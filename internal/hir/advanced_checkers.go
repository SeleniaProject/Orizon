package hir

import (
	"fmt"
)

// =============================================================================.
// Advanced Type Checker Infrastructure.
// =============================================================================.

// AdvancedTypeChecker provides comprehensive type checking for advanced type systems.
type AdvancedTypeChecker struct {
	RankNChecker      *RankNTypeChecker
	DependentChecker  *DependentTypeChecker
	EffectChecker     *AdvancedEffectChecker
	LinearChecker     *LinearTypeChecker
	RefinementChecker *RefinementTypeChecker
	CapabilityChecker *CapabilityChecker
	UnificationEngine *AdvancedUnificationEngine
	InferenceEngine   *AdvancedInferenceEngine
	ProofEngine       *ProofEngine
	Environment       *AdvancedTypeEnvironment
	Context           *AdvancedTypeContext
}

// TypeCheckResult represents the result of advanced type checking.
type TypeCheckResult struct {
	Type             TypeInfo
	Constraints      []TypeConstraint
	ProofObligations []ProofObligation
	Substitutions    map[TypeVariable]TypeInfo
	EffectSet        []AdvancedEffect
	RegionSet        []RegionInfo
	Capabilities     []Capability
	Errors           []TypeError
	Warnings         []TypeWarning
	Success          bool
}

// TypeError represents a type checking error.
type TypeError struct {
	Message     string
	Context     string
	Suggestions []string
	Location    SourceLocation
	Severity    ErrorSeverity
	Code        ErrorCode
}

// TypeWarning represents a type checking warning.
type TypeWarning struct {
	Message  string
	Context  string
	Location SourceLocation
	Code     AdvancedWarningCode
}

// ErrorSeverity represents the severity of type errors.
type ErrorSeverity int

const (
	SeverityError ErrorSeverity = iota
	SeverityWarning
	SeverityInfo
	SeverityHint
)

// ErrorCode represents specific type error codes.
type ErrorCode int

const (
	ErrorTypeKindMismatch ErrorCode = iota
	ErrorRankMismatch
	ErrorDependencyMismatch
	ErrorEffectMismatch
	ErrorLinearityViolation
	ErrorRefinementFailure
	ErrorCapabilityInsufficient
	ErrorProofObligationUnsatisfied
	ErrorConstraintUnsatisfiable
	ErrorAdvancedUnificationFailure
)

// AdvancedWarningCode represents specific advanced type warning codes.
type AdvancedWarningCode int

const (
	AdvancedWarningUnusedVariable AdvancedWarningCode = iota
	AdvancedWarningRedundantConstraint
	AdvancedWarningEffectNeverUsed
	AdvancedWarningLinearResourceNotConsumed
	AdvancedWarningRefinementWeakened
	AdvancedWarningCapabilityOverpermissive
)

// SourceLocation represents a location in source code.
type SourceLocation struct {
	File   string
	Line   int
	Column int
	Offset int
}

// NewAdvancedTypeChecker creates a fully-featured advanced type checker.
func NewAdvancedTypeChecker() *AdvancedTypeChecker {
	env := NewAdvancedTypeEnvironment()
	ctx := NewAdvancedTypeContext()

	return &AdvancedTypeChecker{
		RankNChecker:      NewRankNTypeChecker(env, ctx),
		DependentChecker:  NewDependentTypeChecker(env, ctx),
		EffectChecker:     NewAdvancedEffectChecker(env, ctx),
		LinearChecker:     NewLinearTypeChecker(env, ctx),
		RefinementChecker: NewRefinementTypeChecker(env, ctx),
		CapabilityChecker: NewCapabilityChecker(env, ctx),
		UnificationEngine: NewAdvancedUnificationEngine(),
		InferenceEngine:   NewAdvancedInferenceEngine(),
		ProofEngine:       NewProofEngine(),
		Environment:       env,
		Context:           ctx,
	}
}

// CheckAdvancedType performs comprehensive type checking for advanced type systems.
func (atc *AdvancedTypeChecker) CheckAdvancedType(expr HIRExpression, expectedType TypeInfo) (*TypeCheckResult, error) {
	result := &TypeCheckResult{
		Type:             expectedType,
		Constraints:      []TypeConstraint{},
		ProofObligations: []ProofObligation{},
		Substitutions:    make(map[TypeVariable]TypeInfo),
		EffectSet:        []AdvancedEffect{},
		RegionSet:        []RegionInfo{},
		Capabilities:     []Capability{},
		Errors:           []TypeError{},
		Warnings:         []TypeWarning{},
		Success:          true,
	}

	// Phase 1: Determine expression type through inference.
	inferredType, err := atc.InferenceEngine.InferType(expr, atc.Environment)
	if err != nil {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Type inference failed: %v", err),
			Severity: SeverityError,
			Code:     ErrorAdvancedUnificationFailure,
		})
		result.Success = false

		return result, err
	}

	// Phase 2: Check if inferred type is advanced.
	if advType, isAdvanced := IsAdvancedType(inferredType); isAdvanced {
		return atc.checkAdvancedTypeExpression(expr, advType, expectedType, result)
	}

	// Phase 3: Check if expected type is advanced.
	if advExpected, isAdvancedExpected := IsAdvancedType(expectedType); isAdvancedExpected {
		return atc.checkRegularWithAdvanced(expr, inferredType, advExpected, result)
	}

	// Phase 4: Both types are regular - use standard unification.
	unifyResult, err := atc.UnificationEngine.Unify(inferredType, expectedType)
	if err != nil || !unifyResult.Success {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Type mismatch: expected %v, found %v", expectedType, inferredType),
			Severity: SeverityError,
			Code:     ErrorTypeKindMismatch,
		})
		result.Success = false
	}

	return result, nil
}

// checkAdvancedTypeExpression handles expressions with advanced types.
func (atc *AdvancedTypeChecker) checkAdvancedTypeExpression(expr HIRExpression, advType AdvancedTypeInfo, expectedType TypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	switch advType.GetAdvancedKind() {
	case AdvancedTypeRankN:
		return atc.checkRankNExpression(expr, advType.(*RankNType), expectedType, result)
	case AdvancedTypeDependent:
		return atc.checkDependentExpression(expr, advType.(*DependentType), expectedType, result)
	case AdvancedTypeEffect:
		return atc.checkEffectExpression(expr, advType.(*AdvancedEffectType), expectedType, result)
	case AdvancedTypeLinear:
		return atc.checkLinearExpression(expr, advType.(*LinearType), expectedType, result)
	case AdvancedTypeRefinement:
		return atc.checkRefinementExpression(expr, advType.(*RefinementType), expectedType, result)
	default:
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Unsupported advanced type kind: %v", advType.GetAdvancedKind()),
			Severity: SeverityError,
			Code:     ErrorTypeKindMismatch,
		})
		result.Success = false

		return result, fmt.Errorf("unsupported advanced type kind: %v", advType.GetAdvancedKind())
	}
}

// checkRankNExpression handles Rank-N polymorphic type checking.
func (atc *AdvancedTypeChecker) checkRankNExpression(expr HIRExpression, rankNType *RankNType, expectedType TypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	// Enter new scope for type quantifiers.
	atc.Context.EnterScope()
	defer atc.Context.ExitScope()

	// Bind quantified type variables.
	for _, quantifier := range rankNType.Quantifiers {
		skolem := atc.Context.SkolemGenerator.GenerateSkolem()
		skolemType := TypeInfo{
			Kind: TypeKindSkolem,
			Name: skolem,
		}
		atc.Environment.BindTypeVariable(quantifier.Variable, skolemType)
	}

	// Check the body type with quantifiers bound.
	bodyResult, err := atc.CheckAdvancedType(expr, rankNType.Body)
	if err != nil {
		result.Errors = append(result.Errors, bodyResult.Errors...)
		result.Success = false

		return result, err
	}

	// Validate rank constraints.
	for _, constraint := range rankNType.Constraints {
		if !atc.RankNChecker.ValidateRankConstraint(constraint, rankNType) {
			result.Errors = append(result.Errors, TypeError{
				Message:  fmt.Sprintf("Rank constraint violation: %v", constraint),
				Severity: SeverityError,
				Code:     ErrorRankMismatch,
			})
			result.Success = false
		}
	}

	// Merge results.
	result.Constraints = append(result.Constraints, bodyResult.Constraints...)
	result.ProofObligations = append(result.ProofObligations, bodyResult.ProofObligations...)

	return result, nil
}

// checkDependentExpression handles dependent type checking.
func (atc *AdvancedTypeChecker) checkDependentExpression(expr HIRExpression, depType *DependentType, expectedType TypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	// Validate value dependencies.
	dependencyResult := atc.DependentChecker.CheckValueDependency(depType.Dependency, expr)
	if !dependencyResult.Valid {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Dependency validation failed: %v", dependencyResult.Error),
			Severity: SeverityError,
			Code:     ErrorDependencyMismatch,
		})
		result.Success = false
	}

	// Check constructor validity.
	constructorResult := atc.DependentChecker.ValidateConstructor(depType.Constructor, depType.Universe)
	if !constructorResult.Valid {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Constructor validation failed: %v", constructorResult.Error),
			Severity: SeverityError,
			Code:     ErrorDependencyMismatch,
		})
		result.Success = false
	}

	// Validate universe level.
	if !atc.DependentChecker.ValidateUniverseLevel(depType.Universe, expr) {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Universe level validation failed for level %d", depType.Universe),
			Severity: SeverityError,
			Code:     ErrorDependencyMismatch,
		})
		result.Success = false
	}

	// Check index constraints.
	for i, index := range depType.Indices {
		if !atc.DependentChecker.ValidateIndex(index, i, depType) {
			result.Errors = append(result.Errors, TypeError{
				Message:  fmt.Sprintf("Index validation failed at position %d", i),
				Severity: SeverityError,
				Code:     ErrorDependencyMismatch,
			})
			result.Success = false
		}
	}

	return result, nil
}

// checkEffectExpression handles effect type checking.
func (atc *AdvancedTypeChecker) checkEffectExpression(expr HIRExpression, effectType *AdvancedEffectType, expectedType TypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	// Collect effects from expression.
	effectSet := atc.EffectChecker.CollectEffects(expr)
	result.EffectSet = effectSet

	// Check effect compatibility.
	for _, effect := range effectType.Effects {
		if !atc.EffectChecker.IsEffectAllowed(effect, effectSet) {
			result.Errors = append(result.Errors, TypeError{
				Message:  fmt.Sprintf("Effect %s not allowed in current context", effect.Name),
				Severity: SeverityError,
				Code:     ErrorEffectMismatch,
			})
			result.Success = false
		}

		// Validate effect operations.
		for _, operation := range effect.Operations {
			if !atc.EffectChecker.ValidateOperation(operation, effect) {
				result.Errors = append(result.Errors, TypeError{
					Message:  fmt.Sprintf("Invalid operation %s for effect %s", operation.Name, effect.Name),
					Severity: SeverityError,
					Code:     ErrorEffectMismatch,
				})
				result.Success = false
			}
		}
	}

	// Check purity requirements.
	if effectType.Purity == PurityPure && len(effectSet) > 0 {
		result.Errors = append(result.Errors, TypeError{
			Message:  "Pure function cannot have effects",
			Severity: SeverityError,
			Code:     ErrorEffectMismatch,
		})
		result.Success = false
	}

	// Validate region access.
	if !atc.EffectChecker.ValidateRegionAccess(effectType.Region, expr) {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Invalid region access for %s", effectType.Region.Name),
			Severity: SeverityError,
			Code:     ErrorEffectMismatch,
		})
		result.Success = false
	}

	// Check effect handlers.
	for _, handler := range effectType.Handlers {
		if !atc.EffectChecker.ValidateHandler(handler, effectType) {
			result.Errors = append(result.Errors, TypeError{
				Message:  fmt.Sprintf("Invalid effect handler for %s", handler.EffectName),
				Severity: SeverityError,
				Code:     ErrorEffectMismatch,
			})
			result.Success = false
		}
	}

	return result, nil
}

// checkLinearExpression handles linear type checking.
func (atc *AdvancedTypeChecker) checkLinearExpression(expr HIRExpression, linearType *LinearType, expectedType TypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	// Track resource usage.
	usageResult := atc.LinearChecker.TrackResourceUsage(expr, linearType)

	// Check multiplicity constraints.
	if usageResult.ActualUsage < linearType.Multiplicity.Min {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Resource used %d times, minimum required: %d", usageResult.ActualUsage, linearType.Multiplicity.Min),
			Severity: SeverityError,
			Code:     ErrorLinearityViolation,
		})
		result.Success = false
	}

	if usageResult.ActualUsage > linearType.Multiplicity.Max {
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Resource used %d times, maximum allowed: %d", usageResult.ActualUsage, linearType.Multiplicity.Max),
			Severity: SeverityError,
			Code:     ErrorLinearityViolation,
		})
		result.Success = false
	}

	// Validate linear constraints.
	for _, constraint := range linearType.Constraints {
		if !atc.LinearChecker.ValidateLinearConstraint(constraint, linearType, expr) {
			result.Errors = append(result.Errors, TypeError{
				Message:  fmt.Sprintf("Linear constraint violation: %s", constraint.Message),
				Severity: SeverityError,
				Code:     ErrorLinearityViolation,
			})
			result.Success = false
		}
	}

	// Check access permissions.
	requiredAccess := atc.LinearChecker.DetermineRequiredAccess(expr)
	if !atc.LinearChecker.HasSufficientAccess(linearType.Region.Access, requiredAccess) {
		result.Errors = append(result.Errors, TypeError{
			Message:  "Insufficient access permissions for linear resource",
			Severity: SeverityError,
			Code:     ErrorLinearityViolation,
		})
		result.Success = false
	}

	// Validate lifetime constraints.
	if !atc.LinearChecker.ValidateLifetime(linearType.Region.Lifetime, expr) {
		result.Errors = append(result.Errors, TypeError{
			Message:  "Lifetime constraint violation for linear resource",
			Severity: SeverityError,
			Code:     ErrorLinearityViolation,
		})
		result.Success = false
	}

	return result, nil
}

// checkRefinementExpression handles refinement type checking.
func (atc *AdvancedTypeChecker) checkRefinementExpression(expr HIRExpression, refinementType *RefinementType, expectedType TypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	// First check base type.
	baseResult, err := atc.CheckAdvancedType(expr, refinementType.BaseType)
	if err != nil {
		result.Errors = append(result.Errors, baseResult.Errors...)
		result.Success = false

		return result, err
	}

	// Validate refinements.
	for i, refinement := range refinementType.Refinements {
		if !atc.RefinementChecker.ValidateRefinement(refinement, expr, refinementType.Context) {
			result.Errors = append(result.Errors, TypeError{
				Message:  fmt.Sprintf("Refinement validation failed at index %d for variable %s", i, refinement.Variable),
				Severity: SeverityError,
				Code:     ErrorRefinementFailure,
			})
			result.Success = false
		}
	}

	// Handle proof obligations.
	if refinementType.Proof.Status == ProofPending {
		proofResult := atc.ProofEngine.DischargeObligation(refinementType.Proof, refinementType.Context)
		if !proofResult.Success {
			result.ProofObligations = append(result.ProofObligations, refinementType.Proof)

			if proofResult.RequiredProof {
				result.Errors = append(result.Errors, TypeError{
					Message:  fmt.Sprintf("Proof obligation not discharged: %v", proofResult.Error),
					Severity: SeverityError,
					Code:     ErrorProofObligationUnsatisfied,
				})
				result.Success = false
			} else {
				result.Warnings = append(result.Warnings, TypeWarning{
					Message: fmt.Sprintf("Proof obligation deferred: %v", proofResult.Error),
					Code:    AdvancedWarningCode(0), // Generic warning
				})
			}
		}
	}

	// Validate refinement context.
	if !atc.RefinementChecker.ValidateContext(refinementType.Context, expr) {
		result.Errors = append(result.Errors, TypeError{
			Message:  "Refinement context validation failed",
			Severity: SeverityError,
			Code:     ErrorRefinementFailure,
		})
		result.Success = false
	}

	return result, nil
}

// checkRegularWithAdvanced handles cases where regular type meets advanced expected type.
func (atc *AdvancedTypeChecker) checkRegularWithAdvanced(expr HIRExpression, regularType TypeInfo, advExpected AdvancedTypeInfo, result *TypeCheckResult) (*TypeCheckResult, error) {
	// Most regular types cannot be coerced to advanced types.
	result.Errors = append(result.Errors, TypeError{
		Message:  fmt.Sprintf("Cannot coerce regular type %v to advanced type %v", regularType, advExpected.GetAdvancedKind()),
		Severity: SeverityError,
		Code:     ErrorTypeKindMismatch,
	})
	result.Success = false

	return result, nil
}

// ValidateTypeWellFormedness checks if an advanced type is well-formed.
func (atc *AdvancedTypeChecker) ValidateTypeWellFormedness(advType AdvancedTypeInfo) (*TypeValidationResult, error) {
	result := &TypeValidationResult{
		Valid:    true,
		Errors:   []TypeError{},
		Warnings: []TypeWarning{},
	}

	switch advType.GetAdvancedKind() {
	case AdvancedTypeRankN:
		return atc.RankNChecker.ValidateWellFormedness(advType.(*RankNType))
	case AdvancedTypeDependent:
		return atc.DependentChecker.ValidateWellFormedness(advType.(*DependentType))
	case AdvancedTypeEffect:
		return atc.EffectChecker.ValidateWellFormedness(advType.(*AdvancedEffectType))
	case AdvancedTypeLinear:
		return atc.LinearChecker.ValidateWellFormedness(advType.(*LinearType))
	case AdvancedTypeRefinement:
		return atc.RefinementChecker.ValidateWellFormedness(advType.(*RefinementType))
	default:
		result.Valid = false
		result.Errors = append(result.Errors, TypeError{
			Message:  fmt.Sprintf("Unknown advanced type kind: %v", advType.GetAdvancedKind()),
			Severity: SeverityError,
			Code:     ErrorTypeKindMismatch,
		})
	}

	return result, nil
}

// TypeValidationResult represents the result of type validation.
type TypeValidationResult struct {
	Errors   []TypeError
	Warnings []TypeWarning
	Valid    bool
}
