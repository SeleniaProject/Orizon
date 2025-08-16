package collections

import "testing"

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue[int](func(a, b int) bool { return a < b })
	pq.Push(5)
	pq.Push(1)
	pq.Push(3)
	v, _ := pq.Pop()
	if v != 1 {
		t.Fatalf("got %d", v)
	}
	v, _ = pq.Pop()
	if v != 3 {
		t.Fatalf("got %d", v)
	}
	v, _ = pq.Pop()
	if v != 5 {
		t.Fatalf("got %d", v)
	}
	if _, ok := pq.Pop(); ok {
		t.Fatal("expected empty")
	}
}
