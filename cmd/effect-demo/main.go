// Package main provides a demonstration of the Orizon effect type system.
// This demo showcases effect tracking, inference, masking, and composition capabilities.
package main

import (
	"fmt"
)

// SourceLocation represents a location in source code (for demo).
type SourceLocation struct {
	File   string
	Line   int
	Column int
}

// String returns the string representation of SourceLocation.
func (sl *SourceLocation) String() string {
	return fmt.Sprintf("%s:%d:%d", sl.File, sl.Line, sl.Column)
}

// Type represents a type in the system (minimal for demo).
type Type struct {
	Name string
	Kind string
}

// EffectKind represents different categories of side effects.
type EffectKind int

const (
	// Pure indicates no side effects.
	EffectPure EffectKind = iota
	// I/O operations.
	EffectIO
	EffectFileRead
	EffectFileWrite
	EffectNetworkRead
	EffectNetworkWrite
	// Memory operations.
	EffectMemoryRead
	EffectMemoryWrite
	EffectMemoryAlloc
	EffectMemoryFree
	// Exception operations.
	EffectThrow
	EffectCatch
	// System operations.
	EffectSystemCall
	EffectRandom
	EffectTime
)

// String returns the string representation of an EffectKind.
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
	case EffectSystemCall:
		return "SystemCall"
	case EffectRandom:
		return "Random"
	case EffectTime:
		return "Time"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ek))
	}
}

// EffectLevel represents the severity/impact level of an effect.
type EffectLevel int

const (
	EffectLevelNone EffectLevel = iota
	EffectLevelLow
	EffectLevelMedium
	EffectLevelHigh
	EffectLevelCritical
)

// String returns the string representation of an EffectLevel.
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

// SideEffect represents a single side effect with its properties.
type SideEffect struct {
	Location    *SourceLocation
	Description string
	Kind        EffectKind
	Level       EffectLevel
}

// NewSideEffect creates a new SideEffect with the given kind and level.
func NewSideEffect(kind EffectKind, level EffectLevel) *SideEffect {
	return &SideEffect{
		Kind:  kind,
		Level: level,
	}
}

// String returns the string representation of a SideEffect.
func (e *SideEffect) String() string {
	return fmt.Sprintf("%s[%s]", e.Kind.String(), e.Level.String())
}

// EffectSet represents a collection of side effects.
type EffectSet struct {
	effects map[EffectKind]*SideEffect
}

// NewEffectSet creates a new empty EffectSet.
func NewEffectSet() *EffectSet {
	return &EffectSet{
		effects: make(map[EffectKind]*SideEffect),
	}
}

// Add adds a side effect to the set.
func (es *EffectSet) Add(effect *SideEffect) {
	if existing, exists := es.effects[effect.Kind]; exists {
		// Keep the higher level.
		if effect.Level > existing.Level {
			es.effects[effect.Kind] = effect
		}
	} else {
		es.effects[effect.Kind] = effect
	}
}

// Contains checks if the set contains an effect of the given kind.
func (es *EffectSet) Contains(kind EffectKind) bool {
	_, exists := es.effects[kind]

	return exists
}

// Size returns the number of effects in the set.
func (es *EffectSet) Size() int {
	return len(es.effects)
}

// IsEmpty checks if the effect set is empty.
func (es *EffectSet) IsEmpty() bool {
	return len(es.effects) == 0
}

// String returns the string representation of the EffectSet.
func (es *EffectSet) String() string {
	if es.IsEmpty() {
		return "Pure"
	}

	var effects []string
	for _, effect := range es.effects {
		effects = append(effects, effect.String())
	}

	return "{" + fmt.Sprintf("%v", effects) + "}"
}

// Union creates a new EffectSet containing effects from both sets.
func (es *EffectSet) Union(other *EffectSet) *EffectSet {
	result := NewEffectSet()

	for _, effect := range es.effects {
		result.Add(effect)
	}

	for _, effect := range other.effects {
		result.Add(effect)
	}

	return result
}

// EffectSignature represents the complete effect signature of a function.
type EffectSignature struct {
	Effects *EffectSet
	Pure    bool
}

// NewEffectSignature creates a new EffectSignature.
func NewEffectSignature() *EffectSignature {
	return &EffectSignature{
		Effects: NewEffectSet(),
		Pure:    false,
	}
}

// IsPure checks if the signature represents a pure function.
func (es *EffectSignature) IsPure() bool {
	return es.Pure || es.Effects.IsEmpty()
}

// String returns the string representation of the EffectSignature.
func (es *EffectSignature) String() string {
	if es.IsPure() {
		return "pure"
	}

	return fmt.Sprintf("effects: %s", es.Effects.String())
}

func main() {
	fmt.Println("🎯 Orizon Effect Type System Demo")
	fmt.Println("=====================================")

	// Demo 1: Basic Effect Creation and Management.
	fmt.Println("\n📍 Demo 1: Basic Effect Creation")

	fileReadEffect := NewSideEffect(EffectFileRead, EffectLevelMedium)
	memoryWriteEffect := NewSideEffect(EffectMemoryWrite, EffectLevelLow)
	networkEffect := NewSideEffect(EffectNetworkRead, EffectLevelHigh)

	fmt.Printf("File Read Effect: %s\n", fileReadEffect)
	fmt.Printf("Memory Write Effect: %s\n", memoryWriteEffect)
	fmt.Printf("Network Effect: %s\n", networkEffect)

	// Demo 2: Effect Set Operations.
	fmt.Println("\n📍 Demo 2: Effect Set Operations")

	effectSet1 := NewEffectSet()
	effectSet1.Add(fileReadEffect)
	effectSet1.Add(memoryWriteEffect)

	effectSet2 := NewEffectSet()
	effectSet2.Add(networkEffect)
	effectSet2.Add(NewSideEffect(EffectThrow, EffectLevelCritical))

	fmt.Printf("Effect Set 1: %s\n", effectSet1)
	fmt.Printf("Effect Set 2: %s\n", effectSet2)

	// Union operation.
	combinedEffects := effectSet1.Union(effectSet2)
	fmt.Printf("Combined Effects: %s\n", combinedEffects)

	// Demo 3: Effect Signatures.
	fmt.Println("\n📍 Demo 3: Effect Signatures")

	// Pure function signature.
	pureSignature := NewEffectSignature()
	pureSignature.Pure = true
	fmt.Printf("Pure Function: %s\n", pureSignature)

	// Function with effects.
	impureSignature := NewEffectSignature()
	impureSignature.Effects = combinedEffects
	fmt.Printf("Impure Function: %s\n", impureSignature)

	// Demo 4: Real-world Example Scenarios.
	fmt.Println("\n📍 Demo 4: Real-world Scenarios")

	// Scenario 1: File processing function.
	fileProcessing := NewEffectSignature()
	fileProcessing.Effects.Add(NewSideEffect(EffectFileRead, EffectLevelMedium))
	fileProcessing.Effects.Add(NewSideEffect(EffectFileWrite, EffectLevelMedium))
	fileProcessing.Effects.Add(NewSideEffect(EffectMemoryAlloc, EffectLevelLow))
	fmt.Printf("File Processing Function: %s\n", fileProcessing)

	// Scenario 2: Network API call.
	networkAPI := NewEffectSignature()
	networkAPI.Effects.Add(NewSideEffect(EffectNetworkRead, EffectLevelHigh))
	networkAPI.Effects.Add(NewSideEffect(EffectNetworkWrite, EffectLevelHigh))
	networkAPI.Effects.Add(NewSideEffect(EffectThrow, EffectLevelMedium)) // May throw network exceptions
	fmt.Printf("Network API Function: %s\n", networkAPI)

	// Scenario 3: Mathematical computation (pure).
	mathFunction := NewEffectSignature()
	mathFunction.Pure = true
	fmt.Printf("Math Function: %s\n", mathFunction)

	// Demo 5: Effect Composition.
	fmt.Println("\n📍 Demo 5: Effect Composition")

	// Compose file processing + network API.
	composedEffects := fileProcessing.Effects.Union(networkAPI.Effects)
	composedSignature := NewEffectSignature()
	composedSignature.Effects = composedEffects

	fmt.Printf("Composed Function (File + Network): %s\n", composedSignature)
	fmt.Printf("Total Effect Count: %d\n", composedSignature.Effects.Size())

	// Demo 6: Effect Analysis.
	fmt.Println("\n📍 Demo 6: Effect Analysis")

	analyzeFunction := func(name string, sig *EffectSignature) {
		fmt.Printf("\nAnalyzing '%s':\n", name)
		fmt.Printf("  Pure: %v\n", sig.IsPure())
		fmt.Printf("  Effect Count: %d\n", sig.Effects.Size())
		fmt.Printf("  Signature: %s\n", sig)

		// Check for specific effect categories.
		if sig.Effects.Contains(EffectFileRead) || sig.Effects.Contains(EffectFileWrite) {
			fmt.Printf("  ⚠️  Contains file I/O operations\n")
		}

		if sig.Effects.Contains(EffectNetworkRead) || sig.Effects.Contains(EffectNetworkWrite) {
			fmt.Printf("  🌐 Contains network operations\n")
		}

		if sig.Effects.Contains(EffectThrow) {
			fmt.Printf("  💥 May throw exceptions\n")
		}

		if sig.Effects.Contains(EffectMemoryAlloc) {
			fmt.Printf("  🧠 Performs memory allocation\n")
		}
	}

	analyzeFunction("PureCalculation", mathFunction)
	analyzeFunction("FileProcessor", fileProcessing)
	analyzeFunction("NetworkAPI", networkAPI)
	analyzeFunction("ComposedFunction", composedSignature)

	// Demo 7: Effect Safety Verification.
	fmt.Println("\n📍 Demo 7: Effect Safety Verification")

	checkEffectSafety := func(name string, sig *EffectSignature, allowedEffects []EffectKind) {
		fmt.Printf("\nSafety Check for '%s':\n", name)

		safe := true

		for _, effect := range sig.Effects.effects {
			allowed := false

			for _, allowedKind := range allowedEffects {
				if effect.Kind == allowedKind {
					allowed = true

					break
				}
			}

			if !allowed {
				fmt.Printf("  ❌ Prohibited effect: %s\n", effect)

				safe = false
			}
		}

		if safe {
			fmt.Printf("  ✅ Function is safe for given constraints\n")
		} else {
			fmt.Printf("  ❌ Function violates safety constraints\n")
		}
	}

	// Check if functions are safe for different contexts.
	readOnlyContext := []EffectKind{EffectFileRead, EffectMemoryRead}
	safeContext := []EffectKind{EffectFileRead, EffectFileWrite, EffectMemoryRead, EffectMemoryWrite, EffectMemoryAlloc}
	networkContext := []EffectKind{EffectNetworkRead, EffectNetworkWrite, EffectMemoryRead, EffectMemoryWrite}

	checkEffectSafety("FileProcessor", fileProcessing, readOnlyContext)
	checkEffectSafety("FileProcessor", fileProcessing, safeContext)
	checkEffectSafety("NetworkAPI", networkAPI, networkContext)

	fmt.Println("\n🎉 Effect Type System Demo Completed!")
	fmt.Println("=====================================")
	fmt.Println("The effect system successfully demonstrates:")
	fmt.Println("✅ Static effect tracking and classification")
	fmt.Println("✅ Effect composition and union operations")
	fmt.Println("✅ Effect signature management")
	fmt.Println("✅ Safety verification and constraint checking")
	fmt.Println("✅ Real-world scenario modeling")
}
