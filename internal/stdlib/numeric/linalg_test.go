package numeric

import "testing"

func almostEq(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func TestDotAxpyScale(t *testing.T) {
	x := []float64{1, 2, 3, 4}
	y := []float64{2, 3, 4, 5}
	if !almostEq(Dot(x, y), 1*2+2*3+3*4+4*5, 1e-12) {
		t.Fatal("dot")
	}
	Axpy(2, x, y)
	if !almostEq(y[0], 2+2*1, 1e-12) || !almostEq(y[3], 5+2*4, 1e-12) {
		t.Fatal("axpy")
	}
	Scale(0.5, x)
	if !almostEq(x[2], 1.5, 1e-12) {
		t.Fatal("scale")
	}
}

func TestMatOps(t *testing.T) {
	A := FromRows([][]float64{{1, 2}, {3, 4}})
	B := FromRows([][]float64{{5, 6}, {7, 8}})
	C := MatMul(A, B)
	if C.R != 2 || C.C != 2 {
		t.Fatal("shape")
	}
	if !almostEq(C.At(0, 0), 19, 1e-12) || !almostEq(C.At(1, 1), 50, 1e-12) {
		t.Fatal("mul")
	}
	T := Transpose(A)
	if T.R != 2 || T.C != 2 || !almostEq(T.At(1, 0), 2, 1e-12) {
		t.Fatal("trans")
	}
	y := MatVec(A, []float64{1, 1})
	if len(y) != 2 || !almostEq(y[1], 7, 1e-12) {
		t.Fatal("matvec")
	}
}

func TestLU_Solve_Det_Inv(t *testing.T) {
	A := FromRows([][]float64{{4, 7}, {2, 6}})
	// det = 4*6-7*2 = 10
	if !almostEq(Det(A), 10, 1e-12) {
		t.Fatal("det")
	}
	// Solve Ax=b with b=[1,0] -> x = A^{-1} e1
	x := Solve(A, []float64{1, 0})
	if x == nil || !almostEq(x[0], 0.6, 1e-12) || !almostEq(x[1], -0.2, 1e-12) {
		t.Fatalf("solve: %#v", x)
	}
	inv := Inverse(A)
	if inv == nil || !almostEq(inv.At(0, 0), 0.6, 1e-12) || !almostEq(inv.At(1, 1), 0.4, 1e-12) {
		t.Fatal("inv")
	}
}

func TestCholesky_SolveSPD(t *testing.T) {
	// SPD matrix: A = [[4,2],[2,3]]
	A := FromRows([][]float64{{4, 2}, {2, 3}})
	L := Cholesky(A)
	if L == nil {
		t.Fatal("cholesky")
	}
	// Solve A x = [1,1]
	x := SolveSPD(A, []float64{1, 1})
	// Solution of [[4,2],[2,3]] x = [1,1] is [0.125, 0.25]
	if x == nil || !almostEq(x[0], 0.125, 1e-12) || !almostEq(x[1], 0.25, 1e-12) {
		t.Fatalf("spd solve: %#v", x)
	}
}
