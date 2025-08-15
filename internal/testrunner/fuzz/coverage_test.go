package fuzz

import "testing"

func TestTokenEdgeCoverage_NotEmpty(t *testing.T) {
	input := "func main() { return }"
	edges := TokenEdgeCoverage(input)
	if len(edges) == 0 {
		t.Fatalf("expected non-empty edge coverage")
	}
}

func TestWeightedTokenEdgeCoverage_NotEmpty(t *testing.T) {
	input := "let x = 1 + 2;"
	edges := WeightedTokenEdgeCoverage(input)
	if len(edges) == 0 {
		t.Fatalf("expected non-empty weighted edge coverage")
	}
}

func TestTokenTrigramCoverage_NotEmpty(t *testing.T) {
	input := "let y = x * 3;"
	tri := TokenTrigramCoverage(input)
	if len(tri) == 0 {
		t.Fatalf("expected non-empty trigram coverage")
	}
}

func TestComputeCoverage_Modes(t *testing.T) {
	input := "func f(){let a=1; }"
	modes := []string{"edge", "weighted", "trigram", "both", ""}
	for _, m := range modes {
		cov := ComputeCoverage(m, input)
		if len(cov) == 0 {
			t.Fatalf("expected non-empty coverage for mode %s", m)
		}
	}
}

