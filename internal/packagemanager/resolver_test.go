package packagemanager

import "testing"

func TestResolver_SimpleGraph(t *testing.T) {
	idx := PackageIndex{
		"A": {
			{Name: "A", Version: "1.0.0", Dependencies: []Dependency{{Name: "B", Constraint: ">=1.0.0, <2.0.0"}}},
			{Name: "A", Version: "1.1.0", Dependencies: []Dependency{{Name: "B", Constraint: ">=1.1.0, <2.0.0"}}},
		},
		"B": {
			{Name: "B", Version: "1.0.0"},
			{Name: "B", Version: "1.2.0"},
		},
	}
	r := NewResolver(idx, ResolveOptions{PreferHigher: true})

	res, err := r.Resolve([]Requirement{{Name: "A", Constraint: ">=1.0.0"}})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if res["A"] != "1.1.0" {
		t.Fatalf("expected A=1.1.0, got %s", res["A"])
	}

	if res["B"] != "1.2.0" {
		t.Fatalf("expected B=1.2.0, got %s", res["B"])
	}
}

func TestResolver_Conflict(t *testing.T) {
	idx := PackageIndex{
		"A": {{Name: "A", Version: "1.0.0", Dependencies: []Dependency{{Name: "B", Constraint: "~1.0.0"}}}},
		"B": {{Name: "B", Version: "2.0.0"}},
	}
	r := NewResolver(idx, ResolveOptions{})

	_, err := r.Resolve([]Requirement{{Name: "A", Constraint: ">=1.0.0"}})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
}
