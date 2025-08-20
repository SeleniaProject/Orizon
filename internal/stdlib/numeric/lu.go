package numeric

import "math"

// LU holds an LU decomposition with partial pivoting.
type LU struct {
	LU   []float64
	Piv  []int
	N    int
	Sign int
}

// LUDecompose computes the LU decomposition with partial pivoting.
// Returns nil if the matrix is singular (to working precision).
func LUDecompose(A *Dense) *LU {
	if A == nil || A.R != A.C || A.R == 0 {
		return nil
	}

	n := A.R
	lu := make([]float64, len(A.Data))
	copy(lu, A.Data)

	piv := make([]int, n)
	for i := range piv {
		piv[i] = i
	}

	sign := 1

	for k := 0; k < n; k++ {
		// Find pivot.
		p := k
		max := math.Abs(lu[k*A.C+k])

		for i := k + 1; i < n; i++ {
			v := math.Abs(lu[i*A.C+k])
			if v > max {
				max = v
				p = i
			}
		}

		if max == 0 || math.IsNaN(max) || math.IsInf(max, 0) {
			return nil
		}
		// Swap rows if needed.
		if p != k {
			for j := 0; j < n; j++ {
				lu[k*A.C+j], lu[p*A.C+j] = lu[p*A.C+j], lu[k*A.C+j]
			}

			piv[k], piv[p] = piv[p], piv[k]
			sign = -sign
		}
		// Factorize.
		pivot := lu[k*A.C+k]
		for i := k + 1; i < n; i++ {
			lu[i*A.C+k] /= pivot
			lik := lu[i*A.C+k]
			// rank-1 update of the trailing submatrix.
			rowi := i*A.C + (k + 1)

			rowk := k*A.C + (k + 1)
			for j := k + 1; j < n; j++ {
				lu[rowi] -= lik * lu[rowk]
				rowi++
				rowk++
			}
		}
	}

	return &LU{N: n, LU: lu, Piv: piv, Sign: sign}
}

// LUSolve solves A x = b using a precomputed LU.
func (f *LU) LUSolve(b []float64) []float64 {
	if f == nil || len(b) != f.N {
		return nil
	}

	n := f.N
	// Apply permutation to b -> x.
	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = b[f.Piv[i]]
	}
	// Forward solve Ly = Pb.
	for i := 0; i < n; i++ {
		sum := x[i]
		row := f.LU[i*n : i*n+n]

		for j := 0; j < i; j++ {
			sum -= row[j] * x[j]
		}

		x[i] = sum
	}
	// Backward solve Ux = y.
	for i := n - 1; i >= 0; i-- {
		sum := x[i]
		row := f.LU[i*n : i*n+n]

		for j := i + 1; j < n; j++ {
			sum -= row[j] * x[j]
		}

		piv := row[i]
		if piv == 0 {
			return nil
		}

		x[i] = sum / piv
	}

	return x
}

// Solve solves A x = b directly; returns nil if singular.
func Solve(A *Dense, b []float64) []float64 {
	lu := LUDecompose(A)
	if lu == nil {
		return nil
	}

	return lu.LUSolve(b)
}

// Det returns the determinant of A using its LU decomposition.
func Det(A *Dense) float64 {
	lu := LUDecompose(A)
	if lu == nil {
		return 0
	}

	det := float64(lu.Sign)
	n := lu.N

	for i := 0; i < n; i++ {
		det *= lu.LU[i*n+i]
	}

	return det
}

// Inverse returns A^{-1} by solving for each basis vector; returns nil if singular.
func Inverse(A *Dense) *Dense {
	lu := LUDecompose(A)
	if lu == nil {
		return nil
	}

	n := lu.N
	inv := NewDense(n, n)

	e := make([]float64, n)
	for j := 0; j < n; j++ {
		for i := range e {
			e[i] = 0
		}

		e[j] = 1

		col := lu.LUSolve(e)
		if col == nil {
			return nil
		}

		for i := 0; i < n; i++ {
			inv.Data[i*n+j] = col[i]
		}
	}

	return inv
}
