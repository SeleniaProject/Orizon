package prop

import (
	"math"
	"math/rand"
)

// GenInt returns a generator for int with magnitude guided by size.
func GenInt() Generator[int] {
	return func(r *rand.Rand, size int) int {
		if size <= 0 {
			size = 30
		}
		// biased toward small numbers but allow larger
		max := int(math.Pow(2, float64(min(size, 31)))) - 1
		if max <= 0 {
			max = 1
		}
		sign := 1
		if r.Intn(2) == 0 {
			sign = -1
		}
		return sign * r.Intn(max+1)
	}
}

// ShrinkInt reduces magnitude toward zero.
func ShrinkInt() Shrinker[int] {
	return func(v int) []int {
		if v == 0 {
			return nil
		}
		out := []int{v / 2, 0}
		if v > 0 {
			out = append(out, v-1)
		} else {
			out = append(out, v+1)
		}
		uniq := make(map[int]struct{}, len(out))
		res := make([]int, 0, len(out))
		for _, x := range out {
			if _, ok := uniq[x]; !ok {
				uniq[x] = struct{}{}
				res = append(res, x)
			}
		}
		return res
	}
}

// GenBool returns a boolean generator.
func GenBool() Generator[bool] {
	return func(r *rand.Rand, _ int) bool { return r.Intn(2) == 0 }
}

// GenSlice returns a slice generator using the element generator.
func GenSlice[T any](elem Generator[T]) Generator[[]T] {
	return func(r *rand.Rand, size int) []T {
		n := r.Intn(max(0, size) + 1)
		out := make([]T, n)
		for i := 0; i < n; i++ {
			out[i] = elem(r, size)
		}
		return out
	}
}

// ShrinkSlice shrinks by removing halves and shrinking head element.
func ShrinkSlice[T any](elem Shrinker[T]) Shrinker[[]T] {
	return func(v []T) []([]T) {
		if len(v) == 0 {
			return nil
		}
		candidates := make([][]T, 0, 4)
		// remove second half
		mid := len(v) / 2
		candidates = append(candidates, append([]T(nil), v[:mid]...))
		// remove first half
		candidates = append(candidates, append([]T(nil), v[mid:]...))
		// shrink head element
		if elem != nil {
			shr := elem(v[0])
			for _, s := range shr {
				nv := append([]T{s}, v[1:]...)
				candidates = append(candidates, nv)
			}
		}
		return candidates
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
