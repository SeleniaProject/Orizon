package numeric

import "math"

// Cholesky computes the Cholesky decomposition A = L L^T for symmetric positive definite A.
// Returns nil if A is not SPD to working precision.
func Cholesky(A *Dense) *Dense {
	if A == nil || A.R != A.C || A.R == 0 {
		return nil
	}

	n := A.R
	L := NewDense(n, n)
	// Copy lower part from A (assume full matrix provided).
	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			sum := A.Data[i*A.C+j]
			for k := 0; k < j; k++ {
				sum -= L.Data[i*L.C+k] * L.Data[j*L.C+k]
			}

			if i == j {
				if sum <= 0 || math.IsNaN(sum) || math.IsInf(sum, 0) {
					return nil
				}

				L.Data[i*L.C+j] = math.Sqrt(sum)
			} else {
				L.Data[i*L.C+j] = sum / L.Data[j*L.C+j]
			}
		}
	}

	return L
}

// SolveSPD solves A x = b using Cholesky when A is SPD.
func SolveSPD(A *Dense, b []float64) []float64 {
	L := Cholesky(A)
	if L == nil || len(b) != A.R {
		return nil
	}

	n := A.R
	y := make([]float64, n)
	// Forward solve L y = b.
	for i := 0; i < n; i++ {
		sum := b[i]
		row := L.Data[i*L.C : i*L.C+i]

		for j := 0; j < i; j++ {
			sum -= row[j] * y[j]
		}

		diag := L.Data[i*L.C+i]
		if diag == 0 {
			return nil
		}

		y[i] = sum / diag
	}

	x := make([]float64, n)
	// Backward solve L^T x = y.
	for i := n - 1; i >= 0; i-- {
		sum := y[i]
		// use column i entries below diagonal in L as row i of L^T (excluding diagonal).
		for j := i + 1; j < n; j++ {
			sum -= L.Data[j*L.C+i] * x[j]
		}

		diag := L.Data[i*L.C+i]
		if diag == 0 {
			return nil
		}

		x[i] = sum / diag
	}

	return x
}
