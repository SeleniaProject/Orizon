package numeric

import "testing"

func TestStats(t *testing.T) {
	x := []float64{1, 2, 3, 4}
	if Mean(x) != 2.5 {
		t.Fatal("mean")
	}
	if !almostEq(Variance(x), 1.25, 1e-12) {
		t.Fatal("var")
	}
	if !almostEq(StdDev(x), 1.118033988749895, 1e-12) {
		t.Fatal("std")
	}
	y := []float64{2, 4, 6, 8}
	if !almostEq(Covariance(x, y), 2.5, 1e-12) {
		t.Fatal("cov")
	}
	if !almostEq(Correlation(x, y), 1.0, 1e-12) {
		t.Fatal("corr")
	}
}
