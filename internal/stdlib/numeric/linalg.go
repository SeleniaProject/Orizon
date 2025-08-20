package numeric

// Dense is a simple row-major dense matrix of float64.
type Dense struct {
	Data []float64
	R    int
	C    int
}

// NewDense allocates an r x c zero matrix.
func NewDense(r, c int) *Dense { return &Dense{R: r, C: c, Data: make([]float64, r*c)} }

// FromRows builds a Dense from given rows; rows may vary in length, missing values are zero.
func FromRows(rows [][]float64) *Dense {
	if len(rows) == 0 {
		return &Dense{}
	}

	r := len(rows)

	c := 0
	for _, row := range rows {
		if len(row) > c {
			c = len(row)
		}
	}

	m := NewDense(r, c)

	for i := 0; i < r; i++ {
		row := rows[i]
		for j := 0; j < len(row); j++ {
			m.Data[i*c+j] = row[j]
		}
	}

	return m
}

// At returns A[i,j].
func (d *Dense) At(i, j int) float64 { return d.Data[i*d.C+j] }

// Set sets A[i,j] = v.
func (d *Dense) Set(i, j int, v float64) { d.Data[i*d.C+j] = v }

// Clone returns a deep copy.
func (d *Dense) Clone() *Dense {
	if d == nil {
		return nil
	}

	out := &Dense{R: d.R, C: d.C, Data: make([]float64, len(d.Data))}
	copy(out.Data, d.Data)

	return out
}

// Eye returns an identity matrix of size n.
func Eye(n int) *Dense {
	m := NewDense(n, n)
	for i := 0; i < n; i++ {
		m.Set(i, i, 1)
	}

	return m
}

// Zeros returns an r x c zero matrix.
func Zeros(r, c int) *Dense { return NewDense(r, c) }

// Ones returns an r x c matrix filled with 1.
func Ones(r, c int) *Dense {
	m := NewDense(r, c)
	for i := range m.Data {
		m.Data[i] = 1
	}

	return m
}

// Dot returns the dot product of two equal-length vectors.
func Dot(x, y []float64) float64 { return dot64(x, y) }

// Axpy performs y = y + a*x (in place) for equal-length vectors.
func Axpy(a float64, x, y []float64) { axpy64(a, x, y) }

// Scale scales x in place: x = a * x.
func Scale(a float64, x []float64) { scale64(a, x) }

// Norm2 returns the Euclidean norm of x.
func Norm2(x []float64) float64 {
	// Kahan summation for better accuracy.
	var sum, c float64
	for _, v := range x {
		y := v*v - c
		t := sum + y
		c = (t - sum) - y
		sum = t
	}

	return mathSqrt(sum)
}

// MatVec computes y = A*x.
func MatVec(A *Dense, x []float64) []float64 {
	if A == nil || A.R == 0 || A.C == 0 {
		return nil
	}

	y := make([]float64, A.R)

	for i := 0; i < A.R; i++ {
		row := A.Data[i*A.C : i*A.C+A.C]
		y[i] = dot64(row, x)
	}

	return y
}

// MatMul computes C = A * B.
func MatMul(A, B *Dense) *Dense {
	if A == nil || B == nil || A.C != B.R {
		return &Dense{}
	}

	C := NewDense(A.R, B.C)
	// cache-friendly: B^T を使って行ベクトルと列ベクトルの内積.
	Bt := Transpose(B)

	for i := 0; i < A.R; i++ {
		ai := A.Data[i*A.C : i*A.C+A.C]
		for j := 0; j < B.C; j++ {
			C.Data[i*C.C+j] = dot64(ai, Bt.Data[j*Bt.C:j*Bt.C+Bt.C])
		}
	}

	return C
}

// Transpose returns A^T.
func Transpose(A *Dense) *Dense {
	if A == nil {
		return nil
	}

	T := NewDense(A.C, A.R)

	for i := 0; i < A.R; i++ {
		for j := 0; j < A.C; j++ {
			T.Data[j*T.C+i] = A.Data[i*A.C+j]
		}
	}

	return T
}
