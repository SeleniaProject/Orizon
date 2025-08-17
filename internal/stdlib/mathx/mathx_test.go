package mathx

import "testing"

func TestMathx(t *testing.T) {
	if Min(2, 3) != 2 || Max(2, 3) != 3 {
		t.Fatal()
	}
	if Clamp(5, 1, 4) != 4 {
		t.Fatal()
	}
	if Abs(-3) != 3 {
		t.Fatal()
	}
	if Lerp(0, 10, 0.3) != 3 {
		t.Fatal()
	}
}
