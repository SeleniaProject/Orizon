package core

import "testing"

func TestOptionBasic(t *testing.T) {
	o := Some(10)
	if o.IsNone() {
		t.Fatalf("expected Some")
	}

	v, ok := o.Unwrap()
	if !ok || v != 10 {
		t.Fatalf("unwrap mismatch: %v %v", v, ok)
	}

	n := None[int]()
	if n.IsSome() {
		t.Fatalf("expected None")
	}

	if n.Or(7) != 7 {
		t.Fatalf("fallback failed")
	}

	m := Map(Some(3), func(x int) int { return x * 2 })
	if v, ok := m.Unwrap(); !ok || v != 6 {
		t.Fatalf("map failed: %v %v", v, ok)
	}
}
