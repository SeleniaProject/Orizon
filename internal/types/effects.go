// Package types provides effect type system for static side effect tracking and control.
// This module implements comprehensive effect tracking, inference, masking, and composition
// to enable safe and predictable handling of side effects in the Orizon language.
package types

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// EffectKind represents different categories of side effects
type EffectKind int

const (
	// Pure indicates no side effects
	EffectPure EffectKind = iota
	// I/O operations
	EffectIO
	EffectFileRead
	EffectFileWrite
	EffectNetworkRead
	EffectNetworkWrite
	EffectConsoleRead
	EffectConsoleWrite
	EffectConsoleOutput // Alias for console output operations
	// Memory operations
	EffectMemoryRead
	EffectMemoryWrite
	EffectMemoryAlloc
	EffectMemoryFree
	// Exception operations
	EffectThrow
	EffectCatch
	// Concurrency operations
	EffectSpawn
	EffectJoin
	EffectSync
	EffectAsync
	// State operations
	EffectStateRead
	EffectStateWrite
	EffectGlobalRead
	EffectGlobalWrite
	// System operations
	EffectSystemCall
	EffectEnvironment
	EffectRandom
	EffectTime
	// Custom effects
	EffectCustom
)

// String returns the string representation of an EffectKind
func (ek EffectKind) String() string {
	switch ek {
	case EffectPure:
		return "Pure"
	case EffectIO:
		return "IO"
	case EffectFileRead:
		return "FileRead"
	case EffectFileWrite:
		return "FileWrite"
	case EffectNetworkRead:
		return "NetworkRead"
	case EffectNetworkWrite:
		return "NetworkWrite"
	case EffectConsoleRead:
		return "ConsoleRead"
	case EffectConsoleWrite:
		return "ConsoleWrite"
	case EffectMemoryRead:
		return "MemoryRead"
	case EffectMemoryWrite:
		return "MemoryWrite"
	case EffectMemoryAlloc:
		return "MemoryAlloc"
	case EffectMemoryFree:
		return "MemoryFree"
	case EffectThrow:
		return "Throw"
	case EffectCatch:
		return "Catch"
	case EffectSpawn:
		return "Spawn"
	case EffectJoin:
		return "Join"
	case EffectSync:
		return "Sync"
	case EffectAsync:
		return "Async"
	case EffectStateRead:
		return "StateRead"
	case EffectStateWrite:
		return "StateWrite"
	case EffectGlobalRead:
		return "GlobalRead"
	case EffectGlobalWrite:
		return "GlobalWrite"
	case EffectSystemCall:
		return "SystemCall"
	case EffectEnvironment:
		return "Environment"
	case EffectRandom:
		return "Random"
	case EffectTime:
		return "Time"
	case EffectCustom:
		return "Custom"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ek))
	}
}

// EffectLevel represents the severity/impact level of an effect
type EffectLevel int

const (
	EffectLevelNone EffectLevel = iota
	EffectLevelLow
	EffectLevelMedium
	EffectLevelHigh
	EffectLevelCritical
)

// String returns the string representation of an EffectLevel
func (el EffectLevel) String() string {
	switch el {
	case EffectLevelNone:
		return "None"
	case EffectLevelLow:
		return "Low"
	case EffectLevelMedium:
		return "Medium"
	case EffectLevelHigh:
		return "High"
	case EffectLevelCritical:
		return "Critical"
	default:
		return fmt.Sprintf("Unknown(%d)", int(el))
	}
}

// SideEffect represents a single side effect with its properties
type SideEffect struct {
	Kind         EffectKind
	Level        EffectLevel
	Description  string
	Location     *SourceLocation
	Dependencies []string
	Constraints  []string
	Metadata     map[string]interface{}
}

// NewSideEffect creates a new SideEffect with the given kind and level
func NewSideEffect(kind EffectKind, level EffectLevel) *SideEffect {
	return &SideEffect{
		Kind:         kind,
		Level:        level,
		Dependencies: make([]string, 0),
		Constraints:  make([]string, 0),
		Metadata:     make(map[string]interface{}),
	}
}

// String returns the string representation of a SideEffect
func (e *SideEffect) String() string {
	return fmt.Sprintf("%s[%s]", e.Kind.String(), e.Level.String())
}

// Clone creates a deep copy of the SideEffect
func (e *SideEffect) Clone() *SideEffect {
	clone := &SideEffect{
		Kind:         e.Kind,
		Level:        e.Level,
		Description:  e.Description,
		Location:     e.Location,
		Dependencies: make([]string, len(e.Dependencies)),
		Constraints:  make([]string, len(e.Constraints)),
		Metadata:     make(map[string]interface{}),
	}

	copy(clone.Dependencies, e.Dependencies)
	copy(clone.Constraints, e.Constraints)

	for k, v := range e.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// EffectSet represents a collection of side effects
type EffectSet struct {
	effects map[EffectKind]*SideEffect
	mu      sync.RWMutex
}

// NewEffectSet creates a new empty EffectSet
func NewEffectSet() *EffectSet {
	return &EffectSet{
		effects: make(map[EffectKind]*SideEffect),
	}
}

// Add adds a side effect to the set
func (es *EffectSet) Add(effect *SideEffect) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if existing, exists := es.effects[effect.Kind]; exists {
		// Merge effects, keeping the higher level
		if effect.Level > existing.Level {
			es.effects[effect.Kind] = effect.Clone()
		}
	} else {
		es.effects[effect.Kind] = effect.Clone()
	}
}

// Remove removes an effect from the set
func (es *EffectSet) Remove(kind EffectKind) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.effects, kind)
}

// Contains checks if the set contains an effect of the given kind
func (es *EffectSet) Contains(kind EffectKind) bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	_, exists := es.effects[kind]
	return exists
}

// Get retrieves an effect by kind
func (es *EffectSet) Get(kind EffectKind) (*SideEffect, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	effect, exists := es.effects[kind]
	if exists {
		return effect.Clone(), true
	}
	return nil, false
}

// Union creates a new EffectSet containing effects from both sets
func (es *EffectSet) Union(other *EffectSet) *EffectSet {
	result := NewEffectSet()

	es.mu.RLock()
	for _, effect := range es.effects {
		result.Add(effect)
	}
	es.mu.RUnlock()

	other.mu.RLock()
	for _, effect := range other.effects {
		result.Add(effect)
	}
	other.mu.RUnlock()

	return result
}

// Intersection creates a new EffectSet containing effects common to both sets
func (es *EffectSet) Intersection(other *EffectSet) *EffectSet {
	result := NewEffectSet()

	es.mu.RLock()
	other.mu.RLock()
	defer es.mu.RUnlock()
	defer other.mu.RUnlock()

	for kind, effect := range es.effects {
		if _, exists := other.effects[kind]; exists {
			result.Add(effect)
		}
	}

	return result
}

// IsEmpty checks if the effect set is empty
func (es *EffectSet) IsEmpty() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.effects) == 0
}

// Size returns the number of effects in the set
func (es *EffectSet) Size() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.effects)
}

// ToSlice returns all effects as a slice
func (es *EffectSet) ToSlice() []*SideEffect {
	es.mu.RLock()
	defer es.mu.RUnlock()

	effects := make([]*SideEffect, 0, len(es.effects))
	for _, effect := range es.effects {
		effects = append(effects, effect.Clone())
	}

	// Sort by effect kind for consistent ordering
	sort.Slice(effects, func(i, j int) bool {
		return effects[i].Kind < effects[j].Kind
	})

	return effects
}

// String returns the string representation of the EffectSet
func (es *EffectSet) String() string {
	effects := es.ToSlice()
	if len(effects) == 0 {
		return "Pure"
	}

	var strs []string
	for _, effect := range effects {
		strs = append(strs, effect.String())
	}

	return "{" + strings.Join(strs, ", ") + "}"
}

// EffectSignature represents the complete effect signature of a function or expression
type EffectSignature struct {
	Effects     *EffectSet
	Requires    *EffectSet
	Ensures     *EffectSet
	Masks       *EffectSet
	Constraints []EffectConstraint
	Pure        bool
}

// NewEffectSignature creates a new EffectSignature
func NewEffectSignature() *EffectSignature {
	return &EffectSignature{
		Effects:     NewEffectSet(),
		Requires:    NewEffectSet(),
		Ensures:     NewEffectSet(),
		Masks:       NewEffectSet(),
		Constraints: make([]EffectConstraint, 0),
		Pure:        false,
	}
}

// Clone creates a deep copy of the EffectSignature
func (es *EffectSignature) Clone() *EffectSignature {
	clone := &EffectSignature{
		Effects:     es.Effects.Union(NewEffectSet()),
		Requires:    es.Requires.Union(NewEffectSet()),
		Ensures:     es.Ensures.Union(NewEffectSet()),
		Masks:       es.Masks.Union(NewEffectSet()),
		Constraints: make([]EffectConstraint, len(es.Constraints)),
		Pure:        es.Pure,
	}

	copy(clone.Constraints, es.Constraints)
	return clone
}

// IsPure checks if the signature represents a pure function
func (es *EffectSignature) IsPure() bool {
	return es.Pure || es.Effects.IsEmpty()
}

// String returns the string representation of the EffectSignature
func (es *EffectSignature) String() string {
	if es.IsPure() {
		return "pure"
	}

	parts := []string{}

	if !es.Effects.IsEmpty() {
		parts = append(parts, fmt.Sprintf("effects: %s", es.Effects.String()))
	}

	if !es.Requires.IsEmpty() {
		parts = append(parts, fmt.Sprintf("requires: %s", es.Requires.String()))
	}

	if !es.Ensures.IsEmpty() {
		parts = append(parts, fmt.Sprintf("ensures: %s", es.Ensures.String()))
	}

	if !es.Masks.IsEmpty() {
		parts = append(parts, fmt.Sprintf("masks: %s", es.Masks.String()))
	}

	return strings.Join(parts, ", ")
}

// EffectConstraint represents constraints on effects
type EffectConstraint interface {
	Check(signature *EffectSignature) error
	String() string
}

// NoEffectConstraint ensures no effects of specified kinds
type NoEffectConstraint struct {
	Kinds []EffectKind
}

// Check verifies that no prohibited effects are present
func (nec *NoEffectConstraint) Check(signature *EffectSignature) error {
	for _, kind := range nec.Kinds {
		if signature.Effects.Contains(kind) {
			return fmt.Errorf("prohibited effect %s is present", kind.String())
		}
	}
	return nil
}

// String returns the string representation of NoEffectConstraint
func (nec *NoEffectConstraint) String() string {
	var kinds []string
	for _, kind := range nec.Kinds {
		kinds = append(kinds, kind.String())
	}
	return fmt.Sprintf("no effects: %s", strings.Join(kinds, ", "))
}

// RequiredEffectConstraint ensures specific effects are present
type RequiredEffectConstraint struct {
	Kinds []EffectKind
}

// Check verifies that all required effects are present
func (rec *RequiredEffectConstraint) Check(signature *EffectSignature) error {
	for _, kind := range rec.Kinds {
		if !signature.Effects.Contains(kind) {
			return fmt.Errorf("required effect %s is missing", kind.String())
		}
	}
	return nil
}

// String returns the string representation of RequiredEffectConstraint
func (rec *RequiredEffectConstraint) String() string {
	var kinds []string
	for _, kind := range rec.Kinds {
		kinds = append(kinds, kind.String())
	}
	return fmt.Sprintf("requires effects: %s", strings.Join(kinds, ", "))
}

// EffectInferenceContext holds context for effect inference
type EffectInferenceContext struct {
	FunctionSignatures map[string]*EffectSignature
	VariableEffects    map[string]*EffectSet
	ScopeStack         []*EffectScope
	mu                 sync.RWMutex
}

// NewEffectInferenceContext creates a new inference context
func NewEffectInferenceContext() *EffectInferenceContext {
	return &EffectInferenceContext{
		FunctionSignatures: make(map[string]*EffectSignature),
		VariableEffects:    make(map[string]*EffectSet),
		ScopeStack:         make([]*EffectScope, 0),
	}
}

// PushScope adds a new scope to the stack
func (eic *EffectInferenceContext) PushScope(scope *EffectScope) {
	eic.mu.Lock()
	defer eic.mu.Unlock()
	eic.ScopeStack = append(eic.ScopeStack, scope)
}

// PopScope removes the top scope from the stack
func (eic *EffectInferenceContext) PopScope() *EffectScope {
	eic.mu.Lock()
	defer eic.mu.Unlock()

	if len(eic.ScopeStack) == 0 {
		return nil
	}

	scope := eic.ScopeStack[len(eic.ScopeStack)-1]
	eic.ScopeStack = eic.ScopeStack[:len(eic.ScopeStack)-1]
	return scope
}

// CurrentScope returns the current top scope
func (eic *EffectInferenceContext) CurrentScope() *EffectScope {
	eic.mu.RLock()
	defer eic.mu.RUnlock()

	if len(eic.ScopeStack) == 0 {
		return nil
	}

	return eic.ScopeStack[len(eic.ScopeStack)-1]
}

// EffectScope represents a scope for effect tracking
type EffectScope struct {
	Name      string
	Effects   *EffectSet
	Masks     *EffectSet
	Parent    *EffectScope
	Children  []*EffectScope
	Variables map[string]*EffectSet
	Functions map[string]*EffectSignature
	CreatedAt time.Time
}

// NewEffectScope creates a new effect scope
func NewEffectScope(name string, parent *EffectScope) *EffectScope {
	scope := &EffectScope{
		Name:      name,
		Effects:   NewEffectSet(),
		Masks:     NewEffectSet(),
		Parent:    parent,
		Children:  make([]*EffectScope, 0),
		Variables: make(map[string]*EffectSet),
		Functions: make(map[string]*EffectSignature),
		CreatedAt: time.Now(),
	}

	if parent != nil {
		parent.Children = append(parent.Children, scope)
	}

	return scope
}

// AddEffect adds a side effect to the scope
func (es *EffectScope) AddEffect(effect *SideEffect) {
	es.Effects.Add(effect)
}

// MaskEffect masks an effect in the scope
func (es *EffectScope) MaskEffect(kind EffectKind) {
	es.Masks.Add(NewSideEffect(kind, EffectLevelNone))
}

// GetEffectiveEffects returns effects after applying masks
func (es *EffectScope) GetEffectiveEffects() *EffectSet {
	result := NewEffectSet()

	for _, effect := range es.Effects.ToSlice() {
		if !es.Masks.Contains(effect.Kind) {
			result.Add(effect)
		}
	}

	return result
}

// EffectMask represents effect masking capabilities
type EffectMask struct {
	Name        string // Name of the effect mask
	MaskedKinds []EffectKind
	Conditions  []MaskCondition
	Active      bool
	Temporary   bool
	ExpiresAt   *time.Time
}

// NewEffectMask creates a new effect mask
func NewEffectMask(kinds []EffectKind) *EffectMask {
	return &EffectMask{
		MaskedKinds: kinds,
		Conditions:  make([]MaskCondition, 0),
		Active:      true,
		Temporary:   false,
	}
}

// Apply applies the mask to an effect set
func (em *EffectMask) Apply(effects *EffectSet) *EffectSet {
	if !em.Active || (em.ExpiresAt != nil && time.Now().After(*em.ExpiresAt)) {
		return effects
	}

	result := NewEffectSet()
	for _, effect := range effects.ToSlice() {
		masked := false
		for _, kind := range em.MaskedKinds {
			if effect.Kind == kind {
				masked = true
				break
			}
		}

		if !masked {
			result.Add(effect)
		}
	}

	return result
}

// Contains checks if the mask contains a specific effect kind
func (em *EffectMask) Contains(kind EffectKind) bool {
	if !em.Active || (em.ExpiresAt != nil && time.Now().After(*em.ExpiresAt)) {
		return false
	}

	for _, maskedKind := range em.MaskedKinds {
		if maskedKind == kind {
			return true
		}
	}
	return false
}

// MaskCondition represents conditions for effect masking
type MaskCondition interface {
	Evaluate(context *EffectInferenceContext) bool
	String() string
}

// EffectComposer handles effect composition and inference
type EffectComposer struct {
	rules      []CompositionRule
	cache      map[string]*EffectSignature
	statistics EffectStatistics
	mu         sync.RWMutex
}

// NewEffectComposer creates a new effect composer
func NewEffectComposer() *EffectComposer {
	return &EffectComposer{
		rules:      make([]CompositionRule, 0),
		cache:      make(map[string]*EffectSignature),
		statistics: EffectStatistics{},
	}
}

// AddRule adds a composition rule
func (ec *EffectComposer) AddRule(rule CompositionRule) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.rules = append(ec.rules, rule)
}

// Compose composes effect signatures according to rules
func (ec *EffectComposer) Compose(signatures []*EffectSignature) *EffectSignature {
	result := NewEffectSignature()

	for _, sig := range signatures {
		result.Effects = result.Effects.Union(sig.Effects)
		result.Requires = result.Requires.Union(sig.Requires)
		result.Ensures = result.Ensures.Union(sig.Ensures)
	}

	// Apply composition rules
	ec.mu.RLock()
	for _, rule := range ec.rules {
		result = rule.Apply(result)
	}
	ec.mu.RUnlock()

	return result
}

// CompositionRule represents a rule for effect composition
type CompositionRule interface {
	Apply(signature *EffectSignature) *EffectSignature
	Priority() int
	String() string
}

// EffectStatistics tracks effect system statistics
type EffectStatistics struct {
	InferenceCalls       int64
	CompositionCalls     int64
	CacheHits            int64
	CacheMisses          int64
	ConstraintViolations int64
	EffectMaskings       int64
	mu                   sync.RWMutex
}

// IncrementInference increments inference call count
func (es *EffectStatistics) IncrementInference() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.InferenceCalls++
}

// IncrementComposition increments composition call count
func (es *EffectStatistics) IncrementComposition() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.CompositionCalls++
}

// IncrementCacheHit increments cache hit count
func (es *EffectStatistics) IncrementCacheHit() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.CacheHits++
}

// IncrementCacheMiss increments cache miss count
func (es *EffectStatistics) IncrementCacheMiss() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.CacheMisses++
}

// GetStats returns a copy of current statistics
func (es *EffectStatistics) GetStats() EffectStatistics {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return *es
}

// String returns the string representation of statistics
func (es *EffectStatistics) String() string {
	stats := es.GetStats()
	return fmt.Sprintf("EffectStats{Inference: %d, Composition: %d, CacheHits: %d, CacheMisses: %d, Violations: %d, Maskings: %d}",
		stats.InferenceCalls, stats.CompositionCalls, stats.CacheHits, stats.CacheMisses, stats.ConstraintViolations, stats.EffectMaskings)
}
