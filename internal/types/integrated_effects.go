// Package types provides exception-effect integration for the Orizon compiler.
// This module integrates exception effects with the existing effect tracking system
// to provide comprehensive type-level side effect and exception control.
package types

import (
	"fmt"
	"strings"
)

// IntegratedEffect represents a unified effect that combines side effects and exceptions
type IntegratedEffect struct {
	SideEffect    *SideEffect
	ExceptionSpec *ExceptionSpec
	Context       string
	Location      SourceLocation
	Constraints   []EffectConstraint
	Metadata      map[string]interface{}
}

// NewIntegratedEffect creates a new integrated effect
func NewIntegratedEffect(sideEffect *SideEffect, exceptionSpec *ExceptionSpec) *IntegratedEffect {
	return &IntegratedEffect{
		SideEffect:    sideEffect,
		ExceptionSpec: exceptionSpec,
		Constraints:   make([]EffectConstraint, 0),
		Metadata:      make(map[string]interface{}),
	}
}

func (ie *IntegratedEffect) String() string {
	if ie.SideEffect != nil && ie.ExceptionSpec != nil {
		return fmt.Sprintf("%s+%s", ie.SideEffect.String(), ie.ExceptionSpec.String())
	} else if ie.SideEffect != nil {
		return ie.SideEffect.String()
	} else if ie.ExceptionSpec != nil {
		return ie.ExceptionSpec.String()
	}
	return "NoEffect"
}

func (ie *IntegratedEffect) IsEmpty() bool {
	return (ie.SideEffect == nil || ie.SideEffect.Kind == EffectPure) &&
		(ie.ExceptionSpec == nil || ie.ExceptionSpec.Kind == ExceptionNone)
}

func (ie *IntegratedEffect) GetSeverity() EffectLevel {
	maxSeverity := EffectLevelNone

	if ie.SideEffect != nil && ie.SideEffect.Level > maxSeverity {
		maxSeverity = ie.SideEffect.Level
	}

	if ie.ExceptionSpec != nil {
		exceptionSeverity := mapExceptionSeverityToEffectLevel(ie.ExceptionSpec.Severity)
		if exceptionSeverity > maxSeverity {
			maxSeverity = exceptionSeverity
		}
	}

	return maxSeverity
}

func mapExceptionSeverityToEffectLevel(severity ExceptionSeverity) EffectLevel {
	switch severity {
	case ExceptionSeverityInfo:
		return EffectLevelLow
	case ExceptionSeverityWarning:
		return EffectLevelMedium
	case ExceptionSeverityError:
		return EffectLevelHigh
	case ExceptionSeverityCritical:
		return EffectLevelCritical
	case ExceptionSeverityFatal:
		return EffectLevelCritical
	default:
		return EffectLevelNone
	}
}

// IntegratedEffectSet represents a collection of integrated effects
type IntegratedEffectSet struct {
	effects     map[string]*IntegratedEffect
	sideEffects *EffectSet
	exceptions  *ExceptionSet
}

func NewIntegratedEffectSet() *IntegratedEffectSet {
	return &IntegratedEffectSet{
		effects:     make(map[string]*IntegratedEffect),
		sideEffects: NewEffectSet(),
		exceptions:  NewExceptionSet(),
	}
}

func (ies *IntegratedEffectSet) Add(effect *IntegratedEffect) {
	key := effect.String()
	ies.effects[key] = effect

	if effect.SideEffect != nil {
		ies.sideEffects.Add(effect.SideEffect)
	}

	if effect.ExceptionSpec != nil {
		ies.exceptions.Add(effect.ExceptionSpec)
	}
}

func (ies *IntegratedEffectSet) Size() int {
	return len(ies.effects)
}

func (ies *IntegratedEffectSet) IsEmpty() bool {
	return len(ies.effects) == 0
}

func (ies *IntegratedEffectSet) GetSideEffects() *EffectSet {
	return ies.sideEffects
}

func (ies *IntegratedEffectSet) GetExceptions() *ExceptionSet {
	return ies.exceptions
}

func (ies *IntegratedEffectSet) String() string {
	if ies.IsEmpty() {
		return "NoEffects"
	}

	var effectStrings []string
	for _, effect := range ies.effects {
		effectStrings = append(effectStrings, effect.String())
	}

	return fmt.Sprintf("{%s}", strings.Join(effectStrings, ", "))
}

func (ies *IntegratedEffectSet) Union(other *IntegratedEffectSet) *IntegratedEffectSet {
	result := NewIntegratedEffectSet()

	for _, effect := range ies.effects {
		result.Add(effect)
	}

	for _, effect := range other.effects {
		result.Add(effect)
	}

	return result
}

func (ies *IntegratedEffectSet) Intersection(other *IntegratedEffectSet) *IntegratedEffectSet {
	result := NewIntegratedEffectSet()

	for key, effect := range ies.effects {
		if _, exists := other.effects[key]; exists {
			result.Add(effect)
		}
	}

	return result
}

func (ies *IntegratedEffectSet) Contains(effect *IntegratedEffect) bool {
	key := effect.String()
	_, exists := ies.effects[key]
	return exists
}

// IntegratedEffectSignature represents a complete effect signature including both side effects and exceptions
type IntegratedEffectSignature struct {
	FunctionName  string
	Parameters    []string
	Returns       []string
	Effects       *IntegratedEffectSet
	Requires      *IntegratedEffectSet // Precondition effects
	Ensures       *IntegratedEffectSet // Postcondition effects
	SideEffectSig *EffectSignature
	ExceptionSig  *ExceptionSignature
	Safety        ExceptionSafety
	Purity        bool
	Guarantees    []string
}

func NewIntegratedEffectSignature(name string) *IntegratedEffectSignature {
	sideEffectSig := NewEffectSignature()
	exceptionSig := NewExceptionSignature()

	return &IntegratedEffectSignature{
		FunctionName:  name,
		Parameters:    make([]string, 0),
		Returns:       make([]string, 0),
		Effects:       NewIntegratedEffectSet(),
		Requires:      NewIntegratedEffectSet(),
		Ensures:       NewIntegratedEffectSet(),
		SideEffectSig: sideEffectSig,
		ExceptionSig:  exceptionSig,
		Safety:        SafetyBasic,
		Purity:        true,
		Guarantees:    make([]string, 0),
	}
}

func (ies *IntegratedEffectSignature) String() string {
	return fmt.Sprintf("%s: effects=%s, safety=%s, pure=%v",
		ies.FunctionName, ies.Effects.String(), ies.Safety.String(), ies.Purity)
}

func (ies *IntegratedEffectSignature) AddEffect(effect *IntegratedEffect) {
	ies.Effects.Add(effect)

	if effect.SideEffect != nil {
		ies.SideEffectSig.Effects.Add(effect.SideEffect)
		if effect.SideEffect.Kind != EffectPure {
			ies.Purity = false
		}
	}

	if effect.ExceptionSpec != nil {
		ies.ExceptionSig.Throws.Add(effect.ExceptionSpec)
	}
}

func (ies *IntegratedEffectSignature) IsPure() bool {
	return ies.Purity && ies.Effects.GetSideEffects().Size() == 0 && ies.Effects.GetExceptions().IsEmpty()
}

func (ies *IntegratedEffectSignature) IsNoThrow() bool {
	return ies.Effects.GetExceptions().IsEmpty() || ies.Safety == SafetyNoThrow
}

func (ies *IntegratedEffectSignature) GetMaxSeverity() EffectLevel {
	maxSeverity := EffectLevelNone

	for _, effect := range ies.Effects.effects {
		if severity := effect.GetSeverity(); severity > maxSeverity {
			maxSeverity = severity
		}
	}

	return maxSeverity
}

// IntegratedEffectMask represents a mask for both side effects and exceptions
type IntegratedEffectMask struct {
	SideEffectMask *EffectMask
	ExceptionMask  *ExceptionMask
	Name           string
	Active         bool
	Conditions     []string
}

func NewIntegratedEffectMask(name string) *IntegratedEffectMask {
	sideEffectMask := NewEffectMask([]EffectKind{})
	sideEffectMask.Name = name

	return &IntegratedEffectMask{
		SideEffectMask: sideEffectMask,
		ExceptionMask:  NewExceptionMask(name),
		Name:           name,
		Active:         true,
		Conditions:     make([]string, 0),
	}
}

func (iem *IntegratedEffectMask) MaskEffect(effect *IntegratedEffect) bool {
	sideEffectMasked := true
	exceptionMasked := true

	if effect.SideEffect != nil && iem.SideEffectMask != nil {
		sideEffectMasked = iem.SideEffectMask.Contains(effect.SideEffect.Kind)
	}

	if effect.ExceptionSpec != nil && iem.ExceptionMask != nil {
		exceptionMasked = iem.ExceptionMask.ShouldMask(effect.ExceptionSpec)
	}

	return sideEffectMasked && exceptionMasked
}

func (iem *IntegratedEffectMask) String() string {
	var sideStr, exceptionStr string
	if iem.SideEffectMask != nil {
		sideStr = fmt.Sprintf("%v", iem.SideEffectMask.MaskedKinds)
	}
	if iem.ExceptionMask != nil {
		exceptionStr = iem.ExceptionMask.String()
	}
	return fmt.Sprintf("IntegratedMask[%s]: side=%s, exception=%s",
		iem.Name, sideStr, exceptionStr)
}

// ExceptionMask represents a mask for filtering exceptions
type ExceptionMask struct {
	Name             string
	MaskedKinds      map[ExceptionKind]bool
	MaskedSeverities map[ExceptionSeverity]bool
	AllowedTypes     map[string]bool
	Active           bool
	Temporary        bool
}

func NewExceptionMask(name string) *ExceptionMask {
	return &ExceptionMask{
		Name:             name,
		MaskedKinds:      make(map[ExceptionKind]bool),
		MaskedSeverities: make(map[ExceptionSeverity]bool),
		AllowedTypes:     make(map[string]bool),
		Active:           true,
		Temporary:        false,
	}
}

func (em *ExceptionMask) MaskKind(kind ExceptionKind) {
	em.MaskedKinds[kind] = true
}

func (em *ExceptionMask) MaskSeverity(severity ExceptionSeverity) {
	em.MaskedSeverities[severity] = true
}

func (em *ExceptionMask) AllowType(typeName string) {
	em.AllowedTypes[typeName] = true
}

func (em *ExceptionMask) ShouldMask(spec *ExceptionSpec) bool {
	if !em.Active {
		return false
	}

	// Check if kind is masked
	if em.MaskedKinds[spec.Kind] {
		return true
	}

	// Check if severity is masked
	if em.MaskedSeverities[spec.Severity] {
		return true
	}

	// Check if type is explicitly allowed
	if len(em.AllowedTypes) > 0 {
		return !em.AllowedTypes[spec.TypeName]
	}

	return false
}

func (em *ExceptionMask) String() string {
	if !em.Active {
		return fmt.Sprintf("ExceptionMask[%s:inactive]", em.Name)
	}

	var masked []string
	for kind := range em.MaskedKinds {
		masked = append(masked, kind.String())
	}

	if len(masked) == 0 {
		return fmt.Sprintf("ExceptionMask[%s:none]", em.Name)
	}

	return fmt.Sprintf("ExceptionMask[%s:%s]", em.Name, strings.Join(masked, ","))
}

// IntegratedEffectAnalyzer performs unified analysis of side effects and exceptions
type IntegratedEffectAnalyzer struct {
	EffectInferenceEngine *EffectInferenceEngine
	ExceptionAnalyzer     *ExceptionAnalyzer
	IntegratedSignatures  map[string]*IntegratedEffectSignature
	IntegratedMasks       []*IntegratedEffectMask
}

func NewIntegratedEffectAnalyzer() *IntegratedEffectAnalyzer {
	config := NewEffectInferenceConfig()
	config.EnableCaching = true
	config.EnableOptimization = true

	return &IntegratedEffectAnalyzer{
		EffectInferenceEngine: NewEffectInferenceEngine(config),
		ExceptionAnalyzer:     NewExceptionAnalyzer(),
		IntegratedSignatures:  make(map[string]*IntegratedEffectSignature),
		IntegratedMasks:       make([]*IntegratedEffectMask, 0),
	}
}

func (iea *IntegratedEffectAnalyzer) AnalyzeFunction(node ASTNode) (*IntegratedEffectSignature, error) {
	// Analyze side effects
	sideEffectSig, err := iea.EffectInferenceEngine.InferEffects(node)
	if err != nil {
		return nil, fmt.Errorf("side effect analysis failed: %w", err)
	}

	// Analyze exceptions using basic flow analysis
	exceptionSig := NewExceptionSignature()
	exceptionSig.FunctionName = getFunctionName(node)

	// For demonstration, we'll add some basic exception analysis
	// In a real implementation, this would traverse the AST and analyze exception flows
	if funcDecl, ok := node.(*FunctionDecl); ok {
		exceptionSig.FunctionName = funcDecl.Name

		// Analyze function body for potential exceptions
		exceptionSpec := iea.analyzeExceptionFlows(funcDecl)
		if exceptionSpec != nil {
			exceptionSig.Throws.Add(exceptionSpec)
		}
	}

	// Create integrated signature
	functionName := getFunctionName(node)
	integratedSig := NewIntegratedEffectSignature(functionName)
	integratedSig.SideEffectSig = sideEffectSig
	integratedSig.ExceptionSig = exceptionSig

	// Combine effects
	for _, sideEffect := range sideEffectSig.Effects.ToSlice() {
		integrated := NewIntegratedEffect(sideEffect, nil)
		integratedSig.AddEffect(integrated)
	}

	for _, exception := range exceptionSig.Throws.ToSlice() {
		integrated := NewIntegratedEffect(nil, exception)
		integratedSig.AddEffect(integrated)
	}

	// Determine overall purity and safety
	integratedSig.Purity = sideEffectSig.Pure
	integratedSig.Safety = exceptionSig.Safety

	// Store signature
	iea.IntegratedSignatures[functionName] = integratedSig

	return integratedSig, nil
}

func (iea *IntegratedEffectAnalyzer) analyzeExceptionFlows(funcDecl *FunctionDecl) *ExceptionSpec {
	// Basic exception flow analysis
	// In a real implementation, this would analyze the function body for:
	// - Array access (IndexOutOfBounds)
	// - Division operations (DivisionByZero)
	// - File operations (IOError, FileNotFound)
	// - Network operations (NetworkTimeout)
	// - etc.

	// For demonstration, return a runtime exception if function is not pure
	return NewExceptionSpec(ExceptionRuntime, ExceptionSeverityError)
}

func (iea *IntegratedEffectAnalyzer) ApplyMasks(signature *IntegratedEffectSignature) *IntegratedEffectSignature {
	if len(iea.IntegratedMasks) == 0 {
		return signature
	}

	maskedSig := NewIntegratedEffectSignature(signature.FunctionName)
	maskedSig.Parameters = signature.Parameters
	maskedSig.Returns = signature.Returns

	for _, effect := range signature.Effects.effects {
		shouldMask := false

		for _, mask := range iea.IntegratedMasks {
			if mask.Active && mask.MaskEffect(effect) {
				shouldMask = true
				break
			}
		}

		if !shouldMask {
			maskedSig.AddEffect(effect)
		}
	}

	return maskedSig
}

func (iea *IntegratedEffectAnalyzer) CheckCompatibility(caller, callee *IntegratedEffectSignature) []string {
	var issues []string

	// Check purity compatibility
	if caller.IsPure() && !callee.IsPure() {
		issues = append(issues, fmt.Sprintf("pure function %s cannot call impure function %s",
			caller.FunctionName, callee.FunctionName))
	}

	// Check exception safety compatibility
	if caller.IsNoThrow() && !callee.IsNoThrow() {
		issues = append(issues, fmt.Sprintf("no-throw function %s cannot call throwing function %s",
			caller.FunctionName, callee.FunctionName))
	}

	// Check effect level compatibility
	callerSeverity := caller.GetMaxSeverity()
	calleeSeverity := callee.GetMaxSeverity()
	if callerSeverity < calleeSeverity {
		issues = append(issues, fmt.Sprintf("function %s (severity %s) cannot call function %s (severity %s)",
			caller.FunctionName, callerSeverity.String(), callee.FunctionName, calleeSeverity.String()))
	}

	return issues
}

func (iea *IntegratedEffectAnalyzer) GetSignature(functionName string) (*IntegratedEffectSignature, bool) {
	sig, exists := iea.IntegratedSignatures[functionName]
	return sig, exists
}

func (iea *IntegratedEffectAnalyzer) AddMask(mask *IntegratedEffectMask) {
	iea.IntegratedMasks = append(iea.IntegratedMasks, mask)
}

func (iea *IntegratedEffectAnalyzer) RemoveMask(maskName string) {
	for i, mask := range iea.IntegratedMasks {
		if mask.Name == maskName {
			iea.IntegratedMasks = append(iea.IntegratedMasks[:i], iea.IntegratedMasks[i+1:]...)
			break
		}
	}
}

func (iea *IntegratedEffectAnalyzer) String() string {
	return fmt.Sprintf("IntegratedEffectAnalyzer: %d signatures, %d masks",
		len(iea.IntegratedSignatures), len(iea.IntegratedMasks))
}

// Helper function to extract function name from AST node
func getFunctionName(node ASTNode) string {
	if funcDecl, ok := node.(*FunctionDecl); ok {
		return funcDecl.Name
	}
	return "unknown"
}

// Validation functions for integrated effects

// ValidateIntegratedEffect checks if an integrated effect is valid
func ValidateIntegratedEffect(effect *IntegratedEffect) error {
	if effect == nil {
		return fmt.Errorf("integrated effect cannot be nil")
	}

	if effect.SideEffect == nil && effect.ExceptionSpec == nil {
		return fmt.Errorf("integrated effect must have at least one component")
	}

	if effect.SideEffect != nil {
		if err := ValidateSideEffect(effect.SideEffect); err != nil {
			return fmt.Errorf("invalid side effect: %w", err)
		}
	}

	if effect.ExceptionSpec != nil {
		if err := ValidateExceptionSpec(effect.ExceptionSpec); err != nil {
			return fmt.Errorf("invalid exception spec: %w", err)
		}
	}

	return nil
}

// ValidateSideEffect checks if a side effect is valid
func ValidateSideEffect(effect *SideEffect) error {
	if effect == nil {
		return fmt.Errorf("side effect cannot be nil")
	}

	if effect.Kind < EffectPure || effect.Kind > EffectCustom {
		return fmt.Errorf("invalid effect kind: %v", effect.Kind)
	}

	if effect.Level < EffectLevelNone || effect.Level > EffectLevelCritical {
		return fmt.Errorf("invalid effect level: %v", effect.Level)
	}

	return nil
}

// ValidateExceptionSpec checks if an exception specification is valid
func ValidateExceptionSpec(spec *ExceptionSpec) error {
	if spec == nil {
		return fmt.Errorf("exception spec cannot be nil")
	}

	if spec.Kind < ExceptionNone || spec.Kind > ExceptionCustom {
		return fmt.Errorf("invalid exception kind: %v", spec.Kind)
	}

	if spec.Severity < ExceptionSeverityInfo || spec.Severity > ExceptionSeverityFatal {
		return fmt.Errorf("invalid exception severity: %v", spec.Severity)
	}

	return nil
}
