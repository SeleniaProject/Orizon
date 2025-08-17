package collections

import "testing"

func TestRingBuffer(t *testing.T) {
	r := NewRingBuffer[int](3)
	if !r.IsEmpty() || r.Len() != 0 {
		t.Fatal()
	}
	r.Push(1)
	r.Push(2)
	r.Push(3)
	if r.Len() != 3 {
		t.Fatal()
	}
	// overwrite
	r.Push(4)
	// Now buffer should contain 2,3,4 (1 overwritten)
	if v, _ := r.Peek(); v != 2 {
		t.Fatalf("peek=%d", v)
	}
	v, _ := r.Pop()
	if v != 2 {
		t.Fatalf("pop=%d", v)
	}
	v, _ = r.Pop()
	if v != 3 {
		t.Fatalf("pop=%d", v)
	}
	v, _ = r.Pop()
	if v != 4 {
		t.Fatalf("pop=%d", v)
	}
	if _, ok := r.Pop(); ok {
		t.Fatal("should be empty")
	}
}
