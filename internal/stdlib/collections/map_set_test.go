package collections

import (
	"sort"
	"testing"
)

func TestMapBasic(t *testing.T) {
	m := NewMap[string, int](0)
	if !m.IsEmpty() || m.Len() != 0 {
		t.Fatalf("empty state")
	}
	if m.Has("x") {
		t.Fatalf("should not have x")
	}

	if _, replaced := m.Put("a", 1); replaced {
		t.Fatalf("no replace expected")
	}
	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Fatalf("get a=1")
	}
	if prev, replaced := m.Put("a", 2); !replaced || prev != 1 {
		t.Fatalf("replace a=2")
	}

	v := m.GetOrInsert("b", func() int { return 3 })
	if v != 3 {
		t.Fatalf("getOrInsert b=3")
	}
	if !m.Has("b") {
		t.Fatalf("b should exist")
	}

	u := UpsertTo3(m, "b")
	if u != 3 {
		t.Fatalf("upsert b->3")
	}

	keys := m.Keys()
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("keys %v", keys)
	}

	vals := m.Values()
	sort.Ints(vals)
	if len(vals) != 2 || vals[0] != 2 || vals[1] != 3 {
		t.Fatalf("vals %v", vals)
	}

	mv := MapValues(m, func(x int) string { return string(rune('0' + x)) })
	if mv.Len() != 2 || !mv.Has("a") || mv.GetOrDefault("a", "") != "2" {
		t.Fatalf("mapvalues")
	}

	if prev, ok := m.Delete("a"); !ok || prev != 2 {
		t.Fatalf("delete a")
	}
	if m.Has("a") {
		t.Fatalf("a should be removed")
	}

	m.Clear()
	if !m.IsEmpty() {
		t.Fatalf("clear")
	}
}

func UpsertTo3(m *Map[string, int], key string) int {
	return m.Upsert(key, func(old int, existed bool) int { return 3 })
}

func TestSetBasic(t *testing.T) {
	s := NewSetFrom(1, 2, 3)
	if !s.Has(2) || s.Len() != 3 {
		t.Fatalf("set init")
	}
	added := s.AddAll(2, 4, 5)
	if added != 2 || !s.Has(4) || !s.Has(5) {
		t.Fatalf("addAll")
	}
	if !s.Remove(4) || s.Remove(4) {
		t.Fatalf("remove once")
	}

	u := Union(NewSetFrom(1, 2, 3), NewSetFrom(3, 4))
	ui := u.ToSlice()
	sort.Ints(ui)
	if len(ui) != 4 || ui[0] != 1 || ui[1] != 2 || ui[2] != 3 || ui[3] != 4 {
		t.Fatalf("union %v", ui)
	}

	i := Intersect(NewSetFrom(1, 2, 3), NewSetFrom(2, 3, 5))
	ix := i.ToSlice()
	sort.Ints(ix)
	if len(ix) != 2 || ix[0] != 2 || ix[1] != 3 {
		t.Fatalf("intersect %v", ix)
	}

	d := Difference(NewSetFrom(1, 2, 3), NewSetFrom(2, 4))
	dx := d.ToSlice()
	sort.Ints(dx)
	if len(dx) != 2 || dx[0] != 1 || dx[1] != 3 {
		t.Fatalf("diff %v", dx)
	}
}
