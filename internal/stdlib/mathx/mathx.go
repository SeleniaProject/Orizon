package mathx

// Generic numeric helpers with type sets for common operations.

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~float32 | ~float64
}

// Min returns the smaller of a and b.
func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of a and b.
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp clamps v to the interval [lo, hi]. Assumes lo<=hi.
func Clamp[T Ordered](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Abs returns absolute value of x (for signed numbers).
func Abs[T Signed](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// Lerp performs linear interpolation between a and b by t in [0,1].
func Lerp(a, b float64, t float64) float64 { return a + (b-a)*t }
