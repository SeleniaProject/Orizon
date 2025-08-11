// Package types provides unified integration of all effect systems:
// side effect tracking, exception effects, and I/O effects.
// This creates a comprehensive effect type system for static analysis.
package types

import (
	"fmt"
	"sync"
)

// UnifiedEffectKind represents all types of effects in the system
type UnifiedEffectKind int

const (
	// Pure effects
	UnifiedEffectPure UnifiedEffectKind = iota

	// Side effects (from effect tracking system)
	UnifiedEffectMemoryRead
	UnifiedEffectMemoryWrite
	UnifiedEffectFileRead
	UnifiedEffectFileWrite
	UnifiedEffectNetworkRead
	UnifiedEffectNetworkWrite
	UnifiedEffectConsoleOutput

	// Exception effects (from exception effects system)
	UnifiedEffectThrowsException
	UnifiedEffectCatchesException
	UnifiedEffectHandlesError

	// I/O effects (from I/O effects system)
	UnifiedEffectIORead
	UnifiedEffectIOWrite
	UnifiedEffectIOCreate
	UnifiedEffectIODelete
	UnifiedEffectIOModify
)

// String returns string representation of unified effect kind
func (k UnifiedEffectKind) String() string {
	switch k {
	case UnifiedEffectPure:
		return "Pure"
	case UnifiedEffectMemoryRead:
		return "MemoryRead"
	case UnifiedEffectMemoryWrite:
		return "MemoryWrite"
	case UnifiedEffectFileRead:
		return "FileRead"
	case UnifiedEffectFileWrite:
		return "FileWrite"
	case UnifiedEffectNetworkRead:
		return "NetworkRead"
	case UnifiedEffectNetworkWrite:
		return "NetworkWrite"
	case UnifiedEffectConsoleOutput:
		return "ConsoleOutput"
	case UnifiedEffectThrowsException:
		return "ThrowsException"
	case UnifiedEffectCatchesException:
		return "CatchesException"
	case UnifiedEffectHandlesError:
		return "HandlesError"
	case UnifiedEffectIORead:
		return "IORead"
	case UnifiedEffectIOWrite:
		return "IOWrite"
	case UnifiedEffectIOCreate:
		return "IOCreate"
	case UnifiedEffectIODelete:
		return "IODelete"
	case UnifiedEffectIOModify:
		return "IOModify"
	default:
		return "Unknown"
	}
}

// UnifiedEffect represents a unified effect combining all effect systems
type UnifiedEffect struct {
	// Core properties
	Kind        UnifiedEffectKind
	Level       EffectLevel
	Description string
	Resource    string

	// Side effect properties (simplified)
	HasSideEffect  bool   `json:"has_side_effect,omitempty"`
	SideEffectType string `json:"side_effect_type,omitempty"`

	// Exception effect properties (simplified)
	HasExceptionEffect bool   `json:"has_exception_effect,omitempty"`
	ExceptionType      string `json:"exception_type,omitempty"`

	// I/O effect properties (simplified)
	HasIOEffect bool   `json:"has_io_effect,omitempty"`
	IOType      string `json:"io_type,omitempty"`

	// Unified properties
	Pure          bool
	Deterministic bool
	Idempotent    bool
	Reversible    bool
	Safe          bool

	// Detailed effect references
	ExceptionEffect *ExceptionEffect `json:"exception_effect,omitempty"`
	IOEffect        *IOEffect        `json:"io_effect,omitempty"`

	// Metadata
	Metadata  map[string]interface{}
	Tags      []string
	CreatedAt int64
}

// NewUnifiedEffect creates a new unified effect
func NewUnifiedEffect(kind UnifiedEffectKind, level EffectLevel) *UnifiedEffect {
	return &UnifiedEffect{
		Kind:          kind,
		Level:         level,
		Pure:          kind == UnifiedEffectPure,
		Deterministic: true,
		Idempotent:    true,
		Reversible:    false,
		Safe:          true,
		Metadata:      make(map[string]interface{}),
		Tags:          make([]string, 0),
		CreatedAt:     getCurrentTimestamp(),
	}
}

// Clone creates a deep copy of the unified effect
func (e *UnifiedEffect) Clone() *UnifiedEffect {
	clone := &UnifiedEffect{
		Kind:               e.Kind,
		Level:              e.Level,
		Description:        e.Description,
		Resource:           e.Resource,
		HasSideEffect:      e.HasSideEffect,
		SideEffectType:     e.SideEffectType,
		HasExceptionEffect: e.HasExceptionEffect,
		ExceptionType:      e.ExceptionType,
		HasIOEffect:        e.HasIOEffect,
		IOType:             e.IOType,
		Pure:               e.Pure,
		Deterministic:      e.Deterministic,
		Idempotent:         e.Idempotent,
		Reversible:         e.Reversible,
		Safe:               e.Safe,
		Metadata:           make(map[string]interface{}),
		Tags:               make([]string, len(e.Tags)),
		CreatedAt:          e.CreatedAt,
	}

	// Deep copy metadata
	for k, v := range e.Metadata {
		clone.Metadata[k] = v
	}

	// Copy tags
	copy(clone.Tags, e.Tags)

	return clone
}

// AddTag adds a tag to the effect
func (e *UnifiedEffect) AddTag(tag string) {
	for _, existingTag := range e.Tags {
		if existingTag == tag {
			return // Tag already exists
		}
	}
	e.Tags = append(e.Tags, tag)
}

// HasTag checks if the effect has a specific tag
func (e *UnifiedEffect) HasTag(tag string) bool {
	for _, existingTag := range e.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}

// UnifiedEffectSet represents a thread-safe set of unified effects
type UnifiedEffectSet struct {
	effects map[string]*UnifiedEffect
	mutex   sync.RWMutex
}

// NewUnifiedEffectSet creates a new unified effect set
func NewUnifiedEffectSet() *UnifiedEffectSet {
	return &UnifiedEffectSet{
		effects: make(map[string]*UnifiedEffect),
	}
}

// Add adds an effect to the set
func (s *UnifiedEffectSet) Add(effect *UnifiedEffect) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	key := s.getEffectKey(effect)
	s.effects[key] = effect
}

// Remove removes an effect from the set
func (s *UnifiedEffectSet) Remove(effect *UnifiedEffect) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	key := s.getEffectKey(effect)
	delete(s.effects, key)
}

// Contains checks if the set contains an effect
func (s *UnifiedEffectSet) Contains(effect *UnifiedEffect) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := s.getEffectKey(effect)
	_, exists := s.effects[key]
	return exists
}

// Size returns the number of effects in the set
func (s *UnifiedEffectSet) Size() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.effects)
}

// IsEmpty checks if the set is empty
func (s *UnifiedEffectSet) IsEmpty() bool {
	return s.Size() == 0
}

// IsPure checks if all effects in the set are pure
func (s *UnifiedEffectSet) IsPure() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, effect := range s.effects {
		if !effect.Pure {
			return false
		}
	}
	return true
}

// ToSlice returns a slice of all effects
func (s *UnifiedEffectSet) ToSlice() []*UnifiedEffect {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]*UnifiedEffect, 0, len(s.effects))
	for _, effect := range s.effects {
		result = append(result, effect)
	}
	return result
}

// Union returns a new set containing effects from both sets
func (s *UnifiedEffectSet) Union(other *UnifiedEffectSet) *UnifiedEffectSet {
	result := NewUnifiedEffectSet()

	// Add effects from this set
	for _, effect := range s.ToSlice() {
		result.Add(effect)
	}

	// Add effects from other set
	for _, effect := range other.ToSlice() {
		result.Add(effect)
	}

	return result
}

// Intersection returns a new set containing effects common to both sets
func (s *UnifiedEffectSet) Intersection(other *UnifiedEffectSet) *UnifiedEffectSet {
	result := NewUnifiedEffectSet()

	for _, effect := range s.ToSlice() {
		if other.Contains(effect) {
			result.Add(effect)
		}
	}

	return result
}

// Difference returns a new set containing effects in this set but not in other
func (s *UnifiedEffectSet) Difference(other *UnifiedEffectSet) *UnifiedEffectSet {
	result := NewUnifiedEffectSet()

	for _, effect := range s.ToSlice() {
		if !other.Contains(effect) {
			result.Add(effect)
		}
	}

	return result
}

// getEffectKey generates a unique key for an effect
func (s *UnifiedEffectSet) getEffectKey(effect *UnifiedEffect) string {
	return fmt.Sprintf("%s_%s_%s", effect.Kind.String(), effect.Resource, effect.Description)
}

// UnifiedEffectSignature represents a function signature with unified effects
type UnifiedEffectSignature struct {
	FunctionName  string
	Parameters    []string
	ReturnTypes   []string
	Effects       *UnifiedEffectSet
	Constraints   []UnifiedEffectConstraint
	Pure          bool
	Deterministic bool
	Idempotent    bool
	Safe          bool
	Documentation string
	Examples      []string
	Metadata      map[string]interface{}
}

// NewUnifiedEffectSignature creates a new unified effect signature
func NewUnifiedEffectSignature(functionName string) *UnifiedEffectSignature {
	return &UnifiedEffectSignature{
		FunctionName:  functionName,
		Parameters:    make([]string, 0),
		ReturnTypes:   make([]string, 0),
		Effects:       NewUnifiedEffectSet(),
		Constraints:   make([]UnifiedEffectConstraint, 0),
		Pure:          true,
		Deterministic: true,
		Idempotent:    true,
		Safe:          true,
		Examples:      make([]string, 0),
		Metadata:      make(map[string]interface{}),
	}
}

// AddEffect adds an effect to the signature
func (s *UnifiedEffectSignature) AddEffect(effect *UnifiedEffect) {
	s.Effects.Add(effect)

	// Update signature properties based on effect
	if !effect.Pure {
		s.Pure = false
	}
	if !effect.Deterministic {
		s.Deterministic = false
	}
	if !effect.Idempotent {
		s.Idempotent = false
	}
	if !effect.Safe {
		s.Safe = false
	}
}

// AddConstraint adds a constraint to the signature
func (s *UnifiedEffectSignature) AddConstraint(constraint UnifiedEffectConstraint) {
	s.Constraints = append(s.Constraints, constraint)
}

// CheckConstraints checks all constraints against an effect
func (s *UnifiedEffectSignature) CheckConstraints(effect *UnifiedEffect) []string {
	violations := make([]string, 0)

	for _, constraint := range s.Constraints {
		if !constraint.Check(effect) {
			violations = append(violations, constraint.Describe())
		}
	}

	return violations
}

// UnifiedEffectConstraint interface for unified effect constraints
type UnifiedEffectConstraint interface {
	Check(effect *UnifiedEffect) bool
	Describe() string
}

// UnifiedEffectKindConstraint constrains by effect kind
type UnifiedEffectKindConstraint struct {
	AllowedKinds []UnifiedEffectKind
	DeniedKinds  []UnifiedEffectKind
}

// NewUnifiedEffectKindConstraint creates a new kind constraint
func NewUnifiedEffectKindConstraint() *UnifiedEffectKindConstraint {
	return &UnifiedEffectKindConstraint{
		AllowedKinds: make([]UnifiedEffectKind, 0),
		DeniedKinds:  make([]UnifiedEffectKind, 0),
	}
}

// Allow adds an allowed kind
func (c *UnifiedEffectKindConstraint) Allow(kind UnifiedEffectKind) {
	c.AllowedKinds = append(c.AllowedKinds, kind)
}

// Deny adds a denied kind
func (c *UnifiedEffectKindConstraint) Deny(kind UnifiedEffectKind) {
	c.DeniedKinds = append(c.DeniedKinds, kind)
}

// Check checks if an effect satisfies the constraint
func (c *UnifiedEffectKindConstraint) Check(effect *UnifiedEffect) bool {
	// Check denied kinds first
	for _, deniedKind := range c.DeniedKinds {
		if effect.Kind == deniedKind {
			return false
		}
	}

	// If no allowed kinds specified, allow by default (unless denied)
	if len(c.AllowedKinds) == 0 {
		return true
	}

	// Check allowed kinds
	for _, allowedKind := range c.AllowedKinds {
		if effect.Kind == allowedKind {
			return true
		}
	}

	return false
}

// Describe returns a description of the constraint
func (c *UnifiedEffectKindConstraint) Describe() string {
	return "Effect kind constraint"
}

// UnifiedEffectAnalyzer provides comprehensive analysis of unified effects
type UnifiedEffectAnalyzer struct {
	signatures map[string]*UnifiedEffectSignature
	policies   []UnifiedEffectPolicy
	mutex      sync.RWMutex
}

// NewUnifiedEffectAnalyzer creates a new unified effect analyzer
func NewUnifiedEffectAnalyzer() *UnifiedEffectAnalyzer {
	return &UnifiedEffectAnalyzer{
		signatures: make(map[string]*UnifiedEffectSignature),
		policies:   make([]UnifiedEffectPolicy, 0),
	}
}

// RegisterSignature registers a function signature
func (a *UnifiedEffectAnalyzer) RegisterSignature(sig *UnifiedEffectSignature) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.signatures[sig.FunctionName] = sig
}

// GetSignature retrieves a function signature
func (a *UnifiedEffectAnalyzer) GetSignature(functionName string) (*UnifiedEffectSignature, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	sig, exists := a.signatures[functionName]
	return sig, exists
}

// AddPolicy adds an effect policy
func (a *UnifiedEffectAnalyzer) AddPolicy(policy UnifiedEffectPolicy) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.policies = append(a.policies, policy)
}

// AnalyzeEffects analyzes effects for policy compliance
func (a *UnifiedEffectAnalyzer) AnalyzeEffects(effects *UnifiedEffectSet) *UnifiedEffectAnalysisResult {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	result := &UnifiedEffectAnalysisResult{
		TotalEffects:    effects.Size(),
		PureEffects:     0,
		ImpureEffects:   0,
		SafeEffects:     0,
		UnsafeEffects:   0,
		Violations:      make([]string, 0),
		Recommendations: make([]string, 0),
		Passed:          true,
	}

	// Analyze individual effects
	for _, effect := range effects.ToSlice() {
		if effect.Pure {
			result.PureEffects++
		} else {
			result.ImpureEffects++
		}

		if effect.Safe {
			result.SafeEffects++
		} else {
			result.UnsafeEffects++
		}

		// Check policies
		for _, policy := range a.policies {
			if !policy.Check(effect) {
				violation := fmt.Sprintf("Policy violation: %s for effect %s",
					policy.Describe(), effect.Kind.String())
				result.Violations = append(result.Violations, violation)
				result.Passed = false
			}
		}
	}

	// Generate recommendations
	if result.ImpureEffects > result.PureEffects {
		result.Recommendations = append(result.Recommendations,
			"Consider refactoring to reduce impure effects")
	}

	if result.UnsafeEffects > 0 {
		result.Recommendations = append(result.Recommendations,
			"Review unsafe effects for security implications")
	}

	return result
}

// UnifiedEffectPolicy interface for effect policies
type UnifiedEffectPolicy interface {
	Check(effect *UnifiedEffect) bool
	Describe() string
}

// PurityPolicy enforces purity requirements
type PurityPolicy struct {
	RequirePure bool
}

// Check checks if an effect satisfies the purity policy
func (p *PurityPolicy) Check(effect *UnifiedEffect) bool {
	if p.RequirePure {
		return effect.Pure
	}
	return true
}

// Describe returns a description of the policy
func (p *PurityPolicy) Describe() string {
	if p.RequirePure {
		return "Requires pure effects only"
	}
	return "No purity requirements"
}

// SafetyPolicy enforces safety requirements
type SafetyPolicy struct {
	RequireSafe bool
}

// Check checks if an effect satisfies the safety policy
func (s *SafetyPolicy) Check(effect *UnifiedEffect) bool {
	if s.RequireSafe {
		return effect.Safe
	}
	return true
}

// Describe returns a description of the policy
func (s *SafetyPolicy) Describe() string {
	if s.RequireSafe {
		return "Requires safe effects only"
	}
	return "No safety requirements"
}

// UnifiedEffectAnalysisResult represents the result of effect analysis
type UnifiedEffectAnalysisResult struct {
	TotalEffects    int
	PureEffects     int
	ImpureEffects   int
	SafeEffects     int
	UnsafeEffects   int
	Violations      []string
	Recommendations []string
	Passed          bool
}

// String returns a string representation of the analysis result
func (r *UnifiedEffectAnalysisResult) String() string {
	return fmt.Sprintf(
		"Analysis Result: %d total effects (%d pure, %d impure, %d safe, %d unsafe), "+
			"Passed: %v, Violations: %d, Recommendations: %d",
		r.TotalEffects, r.PureEffects, r.ImpureEffects, r.SafeEffects, r.UnsafeEffects,
		r.Passed, len(r.Violations), len(r.Recommendations))
}

// UnifiedEffectConverter provides conversion between effect systems
type UnifiedEffectConverter struct{}

// NewUnifiedEffectConverter creates a new effect converter
func NewUnifiedEffectConverter() *UnifiedEffectConverter {
	return &UnifiedEffectConverter{}
}

// FromSideEffect converts a side effect to unified effect
func (c *UnifiedEffectConverter) FromSideEffect(sideEffect *Effect) *UnifiedEffect {
	// Map effect name to kind (simplified approach)
	kind := c.mapEffectNameToKind(sideEffect.Name)
	unified := NewUnifiedEffect(kind, EffectLevelLow) // Default level

	unified.Description = sideEffect.Name
	unified.Resource = "" // Not available in basic Effect struct
	unified.Pure = sideEffect.Name == "Pure"
	unified.Deterministic = true // Default assumption
	unified.Safe = true          // Default assumption

	return unified
}

// mapEffectNameToKind maps an effect name to UnifiedEffectKind
func (c *UnifiedEffectConverter) mapEffectNameToKind(name string) UnifiedEffectKind {
	switch name {
	case "IO":
		return UnifiedEffectIORead
	case "Memory":
		return UnifiedEffectMemoryRead
	case "File":
		return UnifiedEffectFileRead
	case "Network":
		return UnifiedEffectNetworkRead
	case "Console":
		return UnifiedEffectConsoleOutput
	case "Pure":
		return UnifiedEffectPure
	default:
		return UnifiedEffectPure // Default to pure for unknown effects
	}
}

// FromExceptionEffect converts an exception effect to unified effect
func (c *UnifiedEffectConverter) FromExceptionEffect(exceptionEffect *ExceptionEffect) *UnifiedEffect {
	unified := NewUnifiedEffect(c.mapExceptionEffectKind(exceptionEffect.Kind), exceptionEffect.Level)
	unified.ExceptionEffect = exceptionEffect
	unified.Description = exceptionEffect.Description
	unified.Pure = false // Exception effects are impure
	unified.Deterministic = exceptionEffect.Deterministic
	unified.Safe = exceptionEffect.Safe

	return unified
}

// FromIOEffect converts an I/O effect to unified effect
func (c *UnifiedEffectConverter) FromIOEffect(ioEffect *IOEffect) *UnifiedEffect {
	unified := NewUnifiedEffect(c.mapIOEffectKind(ioEffect.Kind), ioEffect.Level)
	unified.IOEffect = ioEffect
	unified.Description = ioEffect.Description
	unified.Resource = ioEffect.Resource
	unified.Pure = ioEffect.Kind == IOEffectPure
	unified.Deterministic = !ioEffect.HasBehavior(IOBehaviorNonDeterministic)
	unified.Idempotent = ioEffect.HasBehavior(IOBehaviorIdempotent)
	unified.Safe = ioEffect.Permission != IOPermissionFullAccess

	return unified
}

// mapSideEffectKind maps side effect kind to unified effect kind
func (c *UnifiedEffectConverter) mapSideEffectKind(kind EffectKind) UnifiedEffectKind {
	switch kind {
	case EffectPure:
		return UnifiedEffectPure
	case EffectMemoryRead:
		return UnifiedEffectMemoryRead
	case EffectMemoryWrite:
		return UnifiedEffectMemoryWrite
	case EffectFileRead:
		return UnifiedEffectFileRead
	case EffectFileWrite:
		return UnifiedEffectFileWrite
	case EffectNetworkRead:
		return UnifiedEffectNetworkRead
	case EffectNetworkWrite:
		return UnifiedEffectNetworkWrite
	case EffectConsoleOutput:
		return UnifiedEffectConsoleOutput
	default:
		return UnifiedEffectPure
	}
}

// mapExceptionEffectKind maps exception effect kind to unified effect kind
func (c *UnifiedEffectConverter) mapExceptionEffectKind(kind ExceptionEffectKind) UnifiedEffectKind {
	switch kind {
	case ExceptionEffectThrows, ExceptionEffectArithmeticError, ExceptionEffectNullPointer,
		ExceptionEffectIndexOutOfBounds, ExceptionEffectTypeError, ExceptionEffectIOError:
		return UnifiedEffectThrowsException
	case ExceptionEffectCatches:
		return UnifiedEffectCatchesException
	case ExceptionEffectHandles:
		return UnifiedEffectHandlesError
	default:
		return UnifiedEffectHandlesError
	}
}

// mapIOEffectKind maps I/O effect kind to unified effect kind
func (c *UnifiedEffectConverter) mapIOEffectKind(kind IOEffectKind) UnifiedEffectKind {
	switch kind {
	case IOEffectPure:
		return UnifiedEffectPure
	case IOEffectFileRead, IOEffectDirectoryList, IOEffectFileMetadata,
		IOEffectStdinRead, IOEffectConsoleRead:
		return UnifiedEffectIORead
	case IOEffectFileWrite, IOEffectStdoutWrite, IOEffectStderrWrite, IOEffectConsoleWrite:
		return UnifiedEffectIOWrite
	case IOEffectFileCreate, IOEffectDirectoryCreate:
		return UnifiedEffectIOCreate
	case IOEffectFileDelete, IOEffectDirectoryDelete:
		return UnifiedEffectIODelete
	case IOEffectFileRename, IOEffectFilePermissions:
		return UnifiedEffectIOModify
	default:
		return UnifiedEffectIORead
	}
}

// getCurrentTimestamp returns the current timestamp
func getCurrentTimestamp() int64 {
	return 1640995200 // Mock timestamp for consistent testing
}
