package core

// Local minimal constraints to avoid external deps.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}
type (
	Integer interface{ Signed | Unsigned }
	Float   interface{ ~float32 | ~float64 }
)

// Min returns the minimum of a and b.
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}

	return b
}

// Max returns the maximum of a and b.
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}

	return b
}

// Clamp clamps v into [lo, hi]. If lo>hi they are swapped.
func Clamp[T Ordered](v, lo, hi T) T {
	if lo > hi {
		lo, hi = hi, lo
	}

	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// Abs returns absolute value for signed numbers and floats.
func Abs[T Signed | Float](v T) T {
	if v < 0 {
		return -v
	}

	return v
}

// Sum returns the sum of a slice.
func Sum[T Integer | Float](xs []T) T {
	var s T
	for _, x := range xs {
		s += x
	}

	return s
}

// MeanF returns the arithmetic mean for floating slices. Returns 0 when empty.
func MeanF[T Float](xs []T) float64 {
	if len(xs) == 0 {
		return 0
	}

	var s float64
	for _, x := range xs {
		s += float64(x)
	}

	return s / float64(len(xs))
}
