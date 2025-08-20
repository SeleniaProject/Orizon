package prop

import (
	"testing"
	"time"
)

// Simple property: reversing twice yields original slice.
func TestForAll1_SliceReverseInvolution(t *testing.T) {
	gen := GenSlice[int](GenInt())
	shrink := ShrinkSlice[int](ShrinkInt())
	prop := func(xs []int) bool {
		ys := append([]int(nil), xs...)
		reverse(ys)
		reverse(ys)

		if len(xs) != len(ys) {
			return false
		}

		for i := range xs {
			if xs[i] != ys[i] {
				return false
			}
		}

		return true
	}

	res := ForAll1(gen, shrink, prop, Options{Trials: 200, MaxShrinkTime: 2 * time.Second})
	if res.Failed {
		t.Fatalf("property failed: seed=%d input=%v shrunk=%v", res.Seed, res.FailingInput, res.ShrunkInput)
	}
}

// Negative property to exercise shrinking: sum(xs) < 0 should fail sometimes.
func TestForAll1_NegativeShrinksTowardZero(t *testing.T) {
	gen := GenSlice[int](GenInt())
	shrink := ShrinkSlice[int](ShrinkInt())
	propBad := func(xs []int) bool {
		sum := 0
		for _, v := range xs {
			sum += v
		}

		return sum < 0 // often false -> triggers shrink
	}

	res := ForAll1(gen, shrink, propBad, Options{Trials: 200, MaxShrinkRounds: 50, MaxShrinkTime: 2 * time.Second})
	if !res.Failed {
		t.Fatalf("expected failure to trigger shrinking")
	}
}

func reverse[T any](xs []T) {
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 {
		xs[i], xs[j] = xs[j], xs[i]
	}
}
