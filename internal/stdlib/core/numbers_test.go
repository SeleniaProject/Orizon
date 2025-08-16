package core

import "testing"

func TestNumbers(t *testing.T) {
	if Min(3, 5) != 3 || Max(3, 5) != 5 {
		t.Fatalf("min/max")
	}
	if Clamp(10, 0, 5) != 5 || Clamp(-2, 0, 5) != 0 {
		t.Fatalf("clamp")
	}
	if Abs(-3) != 3 || Abs(3) != 3 {
		t.Fatalf("abs signed")
	}
	if Sum([]int{1, 2, 3, 4}) != 10 {
		t.Fatalf("sum int")
	}
	if MeanF([]float64{1, 2, 3}) != 2 {
		t.Fatalf("mean float")
	}
}
