package collections

import (
	"sort"
	"testing"
)

func lessInt(a, b int) bool { return a < b }

func TestVectorInsertRemoveResize(t *testing.T) {
	v := Of(1, 2, 3)
	if !v.Insert(0, 0) {
		t.Fatalf("insert at 0 failed")
	}
	if got := v.ToSlice(); !(len(got) == 4 && got[0] == 0 && got[1] == 1 && got[2] == 2 && got[3] == 3) {
		t.Fatalf("after insert0: %v", got)
	}

	if !v.InsertAll(2, 9, 9) {
		t.Fatalf("insertAll at 2 failed")
	}
	if got := v.ToSlice(); !(len(got) == 6 && got[0] == 0 && got[1] == 1 && got[2] == 9 && got[3] == 9 && got[4] == 2 && got[5] == 3) {
		t.Fatalf("after insertAll: %v", got)
	}

	x, ok := v.RemoveAt(1)
	if !ok || x != 1 {
		t.Fatalf("removeAt expected 1 got %v,%v", x, ok)
	}
	if got := v.ToSlice(); !(len(got) == 5 && got[0] == 0 && got[1] == 9 && got[2] == 9 && got[3] == 2 && got[4] == 3) {
		t.Fatalf("after removeAt: %v", got)
	}

	if !v.RemoveRange(1, 3) {
		t.Fatalf("removeRange failed")
	}
	if got := v.ToSlice(); !(len(got) == 3 && got[0] == 0 && got[1] == 2 && got[2] == 3) {
		t.Fatalf("after removeRange: %v", got)
	}

	removed := v.RemoveIf(func(x int) bool { return x%2 == 0 })
	if removed != 2 {
		t.Fatalf("removeIf removed=%d", removed)
	}
	if got := v.ToSlice(); !(len(got) == 1 && got[0] == 3) {
		t.Fatalf("after removeIf: %v", got)
	}

	if !v.Resize(5, 7) {
		t.Fatalf("resize grow failed")
	}
	if got := v.ToSlice(); !(len(got) == 5 && got[0] == 3 && got[1] == 7 && got[2] == 7 && got[3] == 7 && got[4] == 7) {
		t.Fatalf("after resize grow: %v", got)
	}
	if !v.Resize(2, 0) {
		t.Fatalf("resize shrink failed")
	}
	if got := v.ToSlice(); !(len(got) == 2 && got[0] == 3 && got[1] == 7) {
		t.Fatalf("after resize shrink: %v", got)
	}
}

func TestVectorReverseSwapSortSearch(t *testing.T) {
	v := Of(4, 1, 3, 2)
	v.Reverse()
	if got := v.ToSlice(); !(got[0] == 2 && got[1] == 3 && got[2] == 1 && got[3] == 4) {
		t.Fatalf("reverse: %v", got)
	}
	if !v.Swap(1, 2) {
		t.Fatalf("swap failed")
	}
	if got := v.ToSlice(); !(got[0] == 2 && got[1] == 1 && got[2] == 3 && got[3] == 4) {
		t.Fatalf("swap: %v", got)
	}

	v.Sort(lessInt)
	if got := v.ToSlice(); !(got[0] == 1 && got[1] == 2 && got[2] == 3 && got[3] == 4) {
		t.Fatalf("sort: %v", got)
	}
	idx, found := v.BinarySearch(3, lessInt)
	if !found || idx != 2 {
		t.Fatalf("binary search 3 -> (2,true), got (%d,%v)", idx, found)
	}
	idx, found = v.BinarySearch(5, lessInt)
	if found || idx != 4 {
		t.Fatalf("binary search 5 -> (4,false), got (%d,%v)", idx, found)
	}
}

func TestVectorCapacityAndClone(t *testing.T) {
	v := NewVector[int](0)
	v.Append(1, 2, 3)
	cap0 := v.Cap()
	v.Reserve(10)
	if v.Cap() < cap0+10 {
		t.Fatalf("reserve not honored: %d -> %d", cap0, v.Cap())
	}

	v.ShrinkToFit()
	if v.Cap() != v.Len() {
		t.Fatalf("shrinkToFit cap(%d) len(%d)", v.Cap(), v.Len())
	}

	v.EnsureCapacity(32)
	if v.Cap() < 32 {
		t.Fatalf("ensureCapacity < 32: %d", v.Cap())
	}
	v.ClearAndShrink()
	if v.Len() != 0 || v.Cap() != 0 {
		t.Fatalf("clearAndShrink len=%d cap=%d", v.Len(), v.Cap())
	}

	v2 := Of(7, 8)
	cl := v2.Clone()
	v2.Set(0, 9)
	if cl.At(0) != 7 {
		t.Fatalf("clone should be independent: %v", cl.ToSlice())
	}

	v3 := Of(1)
	v3.Extend(2, 3)
	v3.ExtendVector(Of(4, 5))
	if got := v3.ToSlice(); !(len(got) == 5 && got[0] == 1 && got[4] == 5) {
		t.Fatalf("extend chain: %v", got)
	}
}

func TestVectorFrontBackPeekAndPredicates(t *testing.T) {
	v := Of(10, 20, 30)
	if f, ok := v.Front(); !ok || f != 10 {
		t.Fatalf("front")
	}
	if p, ok := v.Peek(); !ok || p != 30 {
		t.Fatalf("peek")
	}
	if b, ok := v.Back(); !ok || b != 30 {
		t.Fatalf("back")
	}
	if v.Len() != 2 {
		t.Fatalf("back should pop: len=%d", v.Len())
	}

	if !v.Any(func(x int) bool { return x == 10 }) {
		t.Fatalf("any 10")
	}
	if v.All(func(x int) bool { return x < 20 }) {
		t.Fatalf("all < 20 should be false")
	}
	if idx := v.IndexOf(func(x int) bool { return x == 10 }); idx != 0 {
		t.Fatalf("indexOf 10 = 0 got %d", idx)
	}
	if val, ok := v.Find(func(x int) bool { return x == 10 }); !ok || val != 10 {
		t.Fatalf("find 10")
	}
}

func TestVectorSortStable(t *testing.T) {
	type pair struct{ k, v int }
	vv := NewVector[pair](0)
	vv.Append(pair{1, 1}, pair{1, 2}, pair{1, 3}, pair{2, 1})
	vv.SortStable(func(a, b pair) bool { return a.k < b.k })
	got := vv.ToSlice()
	// Stable: among k==1, v should remain 1,2,3 order
	if !(got[0].v == 1 && got[1].v == 2 && got[2].v == 3 && got[3].k == 2) {
		t.Fatalf("sortStable: %v", got)
	}
}

func TestVectorUnsafeSlice(t *testing.T) {
	v := Of(1, 2, 3)
	s := v.UnsafeSlice()
	s[0] = 9
	if v.At(0) != 9 {
		t.Fatalf("unsafe slice should reflect changes")
	}
	// keep API contract: ToSlice returns a copy
	cp := v.ToSlice()
	cp[0] = 1
	if v.At(0) != 9 {
		t.Fatalf("toSlice must copy")
	}
}

func TestVectorSortVsGoSort(t *testing.T) {
	// Cross-check with Go's sort
	v := Of(5, 1, 4, 2, 3)
	v.Sort(lessInt)
	s := v.ToSlice()
	s2 := []int{5, 1, 4, 2, 3}
	sort.Ints(s2)
	for i := range s {
		if s[i] != s2[i] {
			t.Fatalf("mismatch at %d: %v vs %v", i, s, s2)
		}
	}
}
