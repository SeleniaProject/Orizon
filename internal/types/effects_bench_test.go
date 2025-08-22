package types

import (
	"testing"
)

// BenchmarkEffectSetUnion benchmarks effect set union operations.
func BenchmarkEffectSetUnion(b *testing.B) {
	// Create two effect sets
	set1 := NewEffectSet()
	set2 := NewEffectSet()

	// Add some effects to the sets
	for i := 0; i < 10; i++ {
		set1.Add(NewSideEffect(EffectIO, EffectLevelHigh))
		set1.Add(NewSideEffect(EffectMemoryRead, EffectLevelMedium))
		set2.Add(NewSideEffect(EffectIO, EffectLevelHigh))
		set2.Add(NewSideEffect(EffectNetworkRead, EffectLevelMedium))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := set1.Union(set2)
		_ = result
	}
}

// BenchmarkEffectSetAdd benchmarks adding effects to sets.
func BenchmarkEffectSetAdd(b *testing.B) {
	set := NewEffectSet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(NewSideEffect(EffectIO, EffectLevelHigh))
		set.Add(NewSideEffect(EffectMemoryRead, EffectLevelMedium))
		set.Add(NewSideEffect(EffectNetworkRead, EffectLevelMedium))
		if i%1000 == 0 {
			set = NewEffectSet() // Reset periodically
		}
	}
}
