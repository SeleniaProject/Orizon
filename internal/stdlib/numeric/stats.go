package numeric

import "math"

// Mean returns the arithmetic mean.
func Mean(x []float64) float64 {
	if len(x) == 0 {
		return math.NaN()
	}

	var s float64
	for _, v := range x {
		s += v
	}

	return s / float64(len(x))
}

// Variance (population) with two-pass stable formula.
func Variance(x []float64) float64 {
	if len(x) == 0 {
		return math.NaN()
	}

	m := Mean(x)

	var ss float64

	for _, v := range x {
		d := v - m
		ss += d * d
	}

	return ss / float64(len(x))
}

// StdDev (population) is sqrt(Variance).
func StdDev(x []float64) float64 { return math.Sqrt(Variance(x)) }

// Covariance (population) between x and y.
func Covariance(x, y []float64) float64 {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}

	if n == 0 {
		return math.NaN()
	}

	mx, my := Mean(x[:n]), Mean(y[:n])

	var s float64

	for i := 0; i < n; i++ {
		s += (x[i] - mx) * (y[i] - my)
	}

	return s / float64(n)
}

// Correlation (Pearson) between x and y.
func Correlation(x, y []float64) float64 {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}

	if n == 0 {
		return math.NaN()
	}

	return Covariance(x[:n], y[:n]) / (StdDev(x[:n]) * StdDev(y[:n]))
}
