// Package main demonstrates effect masking capabilities in the Orizon effect system.
// This demo shows how effects can be masked, controlled, and managed in different contexts.
package main

import (
	"fmt"
	"time"
)

// EffectKind represents different categories of side effects
type EffectKind int

const (
	EffectPure EffectKind = iota
	EffectIO
	EffectFileRead
	EffectFileWrite
	EffectNetworkRead
	EffectNetworkWrite
	EffectMemoryWrite
	EffectThrow
	EffectCatch
	EffectSystemCall
)

func (ek EffectKind) String() string {
	names := []string{
		"Pure", "IO", "FileRead", "FileWrite",
		"NetworkRead", "NetworkWrite", "MemoryWrite",
		"Throw", "Catch", "SystemCall",
	}
	if int(ek) < len(names) {
		return names[ek]
	}
	return fmt.Sprintf("Unknown(%d)", int(ek))
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

func (el EffectLevel) String() string {
	names := []string{"None", "Low", "Medium", "High", "Critical"}
	if int(el) < len(names) {
		return names[el]
	}
	return fmt.Sprintf("Unknown(%d)", int(el))
}

// SideEffect represents a single side effect
type SideEffect struct {
	Kind        EffectKind
	Level       EffectLevel
	Description string
}

func NewSideEffect(kind EffectKind, level EffectLevel) *SideEffect {
	return &SideEffect{Kind: kind, Level: level}
}

func (e *SideEffect) String() string {
	return fmt.Sprintf("%s[%s]", e.Kind.String(), e.Level.String())
}

// EffectSet represents a collection of side effects
type EffectSet struct {
	effects map[EffectKind]*SideEffect
}

func NewEffectSet() *EffectSet {
	return &EffectSet{effects: make(map[EffectKind]*SideEffect)}
}

func (es *EffectSet) Add(effect *SideEffect) {
	es.effects[effect.Kind] = effect
}

func (es *EffectSet) Contains(kind EffectKind) bool {
	_, exists := es.effects[kind]
	return exists
}

func (es *EffectSet) Remove(kind EffectKind) {
	delete(es.effects, kind)
}

func (es *EffectSet) Size() int {
	return len(es.effects)
}

func (es *EffectSet) IsEmpty() bool {
	return len(es.effects) == 0
}

func (es *EffectSet) String() string {
	if es.IsEmpty() {
		return "Pure"
	}
	var effects []string
	for _, effect := range es.effects {
		effects = append(effects, effect.String())
	}
	return fmt.Sprintf("{%v}", effects)
}

func (es *EffectSet) ToSlice() []*SideEffect {
	var effects []*SideEffect
	for _, effect := range es.effects {
		effects = append(effects, effect)
	}
	return effects
}

// EffectMask represents effect masking capabilities
type EffectMask struct {
	MaskedKinds []EffectKind
	Active      bool
	Temporary   bool
	ExpiresAt   *time.Time
	Name        string
}

func NewEffectMask(name string, kinds []EffectKind) *EffectMask {
	return &EffectMask{
		Name:        name,
		MaskedKinds: kinds,
		Active:      true,
		Temporary:   false,
	}
}

func (em *EffectMask) SetTemporary(duration time.Duration) {
	em.Temporary = true
	expiry := time.Now().Add(duration)
	em.ExpiresAt = &expiry
}

func (em *EffectMask) IsExpired() bool {
	return em.ExpiresAt != nil && time.Now().After(*em.ExpiresAt)
}

func (em *EffectMask) Apply(effects *EffectSet) *EffectSet {
	if !em.Active || em.IsExpired() {
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

func (em *EffectMask) String() string {
	status := "Active"
	if !em.Active {
		status = "Inactive"
	} else if em.IsExpired() {
		status = "Expired"
	}

	return fmt.Sprintf("Mask[%s](%s): %v", em.Name, status, em.MaskedKinds)
}

// EffectScope represents a scope for effect tracking and masking
type EffectScope struct {
	Name     string
	Effects  *EffectSet
	Masks    []*EffectMask
	Parent   *EffectScope
	Children []*EffectScope
}

func NewEffectScope(name string, parent *EffectScope) *EffectScope {
	scope := &EffectScope{
		Name:     name,
		Effects:  NewEffectSet(),
		Masks:    make([]*EffectMask, 0),
		Parent:   parent,
		Children: make([]*EffectScope, 0),
	}

	if parent != nil {
		parent.Children = append(parent.Children, scope)
	}

	return scope
}

func (es *EffectScope) AddEffect(effect *SideEffect) {
	es.Effects.Add(effect)
}

func (es *EffectScope) AddMask(mask *EffectMask) {
	es.Masks = append(es.Masks, mask)
}

func (es *EffectScope) GetEffectiveEffects() *EffectSet {
	result := NewEffectSet()

	// Start with all effects
	for _, effect := range es.Effects.ToSlice() {
		result.Add(effect)
	}

	// Apply all active masks
	for _, mask := range es.Masks {
		result = mask.Apply(result)
	}

	return result
}

func (es *EffectScope) String() string {
	return fmt.Sprintf("Scope[%s]: %s -> %s",
		es.Name,
		es.Effects.String(),
		es.GetEffectiveEffects().String())
}

// MaskCondition represents conditions for effect masking
type MaskCondition interface {
	Evaluate() bool
	String() string
}

// TimeMaskCondition masks effects based on time
type TimeMaskCondition struct {
	ValidFrom time.Time
	ValidTo   time.Time
}

func (tmc *TimeMaskCondition) Evaluate() bool {
	now := time.Now()
	return now.After(tmc.ValidFrom) && now.Before(tmc.ValidTo)
}

func (tmc *TimeMaskCondition) String() string {
	return fmt.Sprintf("TimeCondition[%v to %v]", tmc.ValidFrom, tmc.ValidTo)
}

// ConditionalEffectMask represents conditional effect masking
type ConditionalEffectMask struct {
	*EffectMask
	Conditions []MaskCondition
}

func NewConditionalEffectMask(name string, kinds []EffectKind, conditions []MaskCondition) *ConditionalEffectMask {
	return &ConditionalEffectMask{
		EffectMask: NewEffectMask(name, kinds),
		Conditions: conditions,
	}
}

func (cem *ConditionalEffectMask) Apply(effects *EffectSet) *EffectSet {
	// Check all conditions
	for _, condition := range cem.Conditions {
		if !condition.Evaluate() {
			return effects // Don't apply mask if conditions not met
		}
	}

	return cem.EffectMask.Apply(effects)
}

func main() {
	fmt.Println("🎭 Orizon Effect Masking System Demo")
	fmt.Println("====================================")

	// Demo 1: Basic Effect Masking
	fmt.Println("\n📍 Demo 1: Basic Effect Masking")

	// Create some effects
	effects := NewEffectSet()
	effects.Add(NewSideEffect(EffectFileRead, EffectLevelMedium))
	effects.Add(NewSideEffect(EffectFileWrite, EffectLevelMedium))
	effects.Add(NewSideEffect(EffectNetworkRead, EffectLevelHigh))
	effects.Add(NewSideEffect(EffectMemoryWrite, EffectLevelLow))

	fmt.Printf("Original Effects: %s\n", effects)

	// Create a mask that blocks file operations
	fileMask := NewEffectMask("FileOperationMask", []EffectKind{EffectFileRead, EffectFileWrite})
	maskedEffects := fileMask.Apply(effects)

	fmt.Printf("Mask: %s\n", fileMask)
	fmt.Printf("After File Mask: %s\n", maskedEffects)

	// Demo 2: Multiple Masks
	fmt.Println("\n📍 Demo 2: Multiple Masks")

	networkMask := NewEffectMask("NetworkMask", []EffectKind{EffectNetworkRead, EffectNetworkWrite})
	memoryMask := NewEffectMask("MemoryMask", []EffectKind{EffectMemoryWrite})

	// Apply masks sequentially
	step1 := fileMask.Apply(effects)
	step2 := networkMask.Apply(step1)
	step3 := memoryMask.Apply(step2)

	fmt.Printf("Original: %s\n", effects)
	fmt.Printf("After File Mask: %s\n", step1)
	fmt.Printf("After Network Mask: %s\n", step2)
	fmt.Printf("After Memory Mask: %s\n", step3)

	// Demo 3: Temporary Masks
	fmt.Println("\n📍 Demo 3: Temporary Masks")

	tempMask := NewEffectMask("TemporaryMask", []EffectKind{EffectFileRead})
	tempMask.SetTemporary(2 * time.Second)

	fmt.Printf("Temporary Mask: %s\n", tempMask)
	fmt.Printf("Before expiry: %s\n", tempMask.Apply(effects))

	// Simulate time passing
	fmt.Println("⏰ Waiting for mask to expire...")
	time.Sleep(3 * time.Second)

	fmt.Printf("After expiry: %s\n", tempMask.Apply(effects))
	fmt.Printf("Mask status: %s\n", tempMask)

	// Demo 4: Effect Scopes with Masking
	fmt.Println("\n📍 Demo 4: Effect Scopes with Masking")

	// Create a hierarchical scope structure
	globalScope := NewEffectScope("Global", nil)
	moduleScope := NewEffectScope("Module", globalScope)
	functionScope := NewEffectScope("Function", moduleScope)

	// Add effects to different scopes
	globalScope.AddEffect(NewSideEffect(EffectSystemCall, EffectLevelCritical))
	moduleScope.AddEffect(NewSideEffect(EffectFileRead, EffectLevelMedium))
	moduleScope.AddEffect(NewSideEffect(EffectFileWrite, EffectLevelMedium))
	functionScope.AddEffect(NewSideEffect(EffectMemoryWrite, EffectLevelLow))
	functionScope.AddEffect(NewSideEffect(EffectNetworkRead, EffectLevelHigh))

	// Add masks to different scopes
	moduleScope.AddMask(NewEffectMask("ModuleSafety", []EffectKind{EffectSystemCall}))
	functionScope.AddMask(NewEffectMask("FunctionSafety", []EffectKind{EffectNetworkRead}))

	fmt.Printf("%s\n", globalScope)
	fmt.Printf("%s\n", moduleScope)
	fmt.Printf("%s\n", functionScope)

	// Demo 5: Conditional Masking
	fmt.Println("\n📍 Demo 5: Conditional Masking")

	// Create time-based condition
	now := time.Now()
	timeCondition := &TimeMaskCondition{
		ValidFrom: now.Add(-1 * time.Hour), // Valid from 1 hour ago
		ValidTo:   now.Add(1 * time.Hour),  // Valid until 1 hour from now
	}

	conditionalMask := NewConditionalEffectMask(
		"TimeBasedSecurity",
		[]EffectKind{EffectFileWrite, EffectSystemCall},
		[]MaskCondition{timeCondition},
	)

	secureEffects := NewEffectSet()
	secureEffects.Add(NewSideEffect(EffectFileRead, EffectLevelMedium))
	secureEffects.Add(NewSideEffect(EffectFileWrite, EffectLevelMedium))
	secureEffects.Add(NewSideEffect(EffectSystemCall, EffectLevelCritical))

	fmt.Printf("Original Effects: %s\n", secureEffects)
	fmt.Printf("Time Condition: %s\n", timeCondition)
	fmt.Printf("Conditional Mask Applied: %s\n", conditionalMask.Apply(secureEffects))

	// Demo 6: Mask Management
	fmt.Println("\n📍 Demo 6: Mask Management")

	maskManager := struct {
		masks []*EffectMask
	}{
		masks: []*EffectMask{
			NewEffectMask("ProductionSafety", []EffectKind{EffectSystemCall, EffectFileWrite}),
			NewEffectMask("TestingMask", []EffectKind{EffectNetworkRead, EffectNetworkWrite}),
			NewEffectMask("DevelopmentMask", []EffectKind{EffectFileRead}),
		},
	}

	// Make testing mask inactive
	maskManager.masks[1].Active = false

	testEffects := NewEffectSet()
	testEffects.Add(NewSideEffect(EffectFileRead, EffectLevelMedium))
	testEffects.Add(NewSideEffect(EffectFileWrite, EffectLevelMedium))
	testEffects.Add(NewSideEffect(EffectNetworkRead, EffectLevelHigh))
	testEffects.Add(NewSideEffect(EffectSystemCall, EffectLevelCritical))

	fmt.Printf("Test Effects: %s\n", testEffects)

	result := testEffects
	for _, mask := range maskManager.masks {
		fmt.Printf("Applying %s\n", mask)
		result = mask.Apply(result)
		fmt.Printf("  Result: %s\n", result)
	}

	// Demo 7: Real-world Scenarios
	fmt.Println("\n📍 Demo 7: Real-world Scenarios")

	scenarios := []struct {
		name        string
		effects     *EffectSet
		masks       []*EffectMask
		description string
	}{
		{
			name: "WebServer",
			effects: func() *EffectSet {
				es := NewEffectSet()
				es.Add(NewSideEffect(EffectNetworkRead, EffectLevelHigh))
				es.Add(NewSideEffect(EffectNetworkWrite, EffectLevelHigh))
				es.Add(NewSideEffect(EffectFileRead, EffectLevelMedium))
				es.Add(NewSideEffect(EffectMemoryWrite, EffectLevelLow))
				return es
			}(),
			masks: []*EffectMask{
				NewEffectMask("ReadOnlyMode", []EffectKind{EffectFileWrite, EffectSystemCall}),
			},
			description: "Web server in read-only mode",
		},
		{
			name: "SandboxedProcess",
			effects: func() *EffectSet {
				es := NewEffectSet()
				es.Add(NewSideEffect(EffectFileRead, EffectLevelMedium))
				es.Add(NewSideEffect(EffectFileWrite, EffectLevelMedium))
				es.Add(NewSideEffect(EffectNetworkRead, EffectLevelHigh))
				es.Add(NewSideEffect(EffectSystemCall, EffectLevelCritical))
				return es
			}(),
			masks: []*EffectMask{
				NewEffectMask("SandboxSecurity", []EffectKind{EffectSystemCall, EffectNetworkRead, EffectFileWrite}),
			},
			description: "Sandboxed process with restricted capabilities",
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\nScenario: %s\n", scenario.name)
		fmt.Printf("Description: %s\n", scenario.description)
		fmt.Printf("Original Effects: %s\n", scenario.effects)

		result := scenario.effects
		for _, mask := range scenario.masks {
			result = mask.Apply(result)
		}

		fmt.Printf("After Masking: %s\n", result)
		fmt.Printf("Effects Reduced: %d -> %d\n", scenario.effects.Size(), result.Size())
	}

	fmt.Println("\n🎉 Effect Masking Demo Completed!")
	fmt.Println("=================================")
	fmt.Println("The masking system successfully demonstrates:")
	fmt.Println("✅ Static effect masking and filtering")
	fmt.Println("✅ Temporary and conditional masks")
	fmt.Println("✅ Hierarchical scope-based masking")
	fmt.Println("✅ Real-world security and safety scenarios")
	fmt.Println("✅ Dynamic mask management and control")
}
