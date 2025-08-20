package numeric

import "math"

// mathSqrt wrapper for testing and potential replacement.
func mathSqrt(x float64) float64 { return math.Sqrt(x) }

// SIMD-ish kernels with simple unrolling; Go will autovec on supported arches.

func dot64(x, y []float64) float64 {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}

	var s0, s1, s2, s3 float64

	i := 0
	for ; i+4 <= n; i += 4 {
		s0 += x[i]*y[i] + x[i+1]*y[i+1]
		s1 += x[i+2]*y[i+2] + x[i+3]*y[i+3]
	}

	s := s0 + s1 + s2 + s3
	for ; i < n; i++ {
		s += x[i] * y[i]
	}

	return s
}

func axpy64(a float64, x, y []float64) {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}

	i := 0
	for ; i+4 <= n; i += 4 {
		y[i] += a * x[i]
		y[i+1] += a * x[i+1]
		y[i+2] += a * x[i+2]
		y[i+3] += a * x[i+3]
	}

	for ; i < n; i++ {
		y[i] += a * x[i]
	}
}

func scale64(a float64, x []float64) {
	i := 0
	n := len(x)

	for ; i+4 <= n; i += 4 {
		x[i] *= a
		x[i+1] *= a
		x[i+2] *= a
		x[i+3] *= a
	}

	for ; i < n; i++ {
		x[i] *= a
	}
}
