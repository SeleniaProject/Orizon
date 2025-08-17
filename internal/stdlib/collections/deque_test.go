package collections

import "testing"

func TestDequeBasic(t *testing.T) {
	var d Deque[int]
	if !d.IsEmpty() || d.Len() != 0 {
		t.Fatalf("expected empty")
	}
	// push back
	for i := 0; i < 10; i++ {
		d.PushBack(i)
	}
	if d.Len() != 10 {
		t.Fatalf("len=%d", d.Len())
	}
	// front/back peek
	if v, ok := d.Front(); !ok || v != 0 {
		t.Fatalf("front=%v %v", v, ok)
	}
	if v, ok := d.Back(); !ok || v != 9 {
		t.Fatalf("back=%v %v", v, ok)
	}
	// pop front
	for i := 0; i < 5; i++ {
		v, ok := d.PopFront()
		if !ok || v != i {
			t.Fatalf("popFront %d got %v %v", i, v, ok)
		}
	}
	// push front
	for i := -1; i >= -3; i-- {
		d.PushFront(i)
	}
	// drain back
	prev := 1000
	for !d.IsEmpty() {
		v, ok := d.PopBack()
		if !ok {
			t.Fatal("unexpected empty")
		}
		if v >= prev {
			t.Fatalf("order not strictly decreasing: %d then %d", prev, v)
		}
		prev = v
	}
}
