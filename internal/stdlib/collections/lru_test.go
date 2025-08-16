package collections

import "testing"

func TestLRUBasic(t *testing.T) {
	l := NewLRU[string, int](2)
	l.Put("a", 1)
	l.Put("b", 2)
	if v, ok := l.Get("a"); !ok || v != 1 {
		t.Fatal()
	}
	// touch a, so b becomes LRU? Actually after Get(a), order is a,b (a MRU)
	l.Put("c", 3) // evict b
	if _, ok := l.Get("b"); ok {
		t.Fatal("b should be evicted")
	}
	if v, ok := l.Get("a"); !ok || v != 1 {
		t.Fatal()
	}
	if v, ok := l.Get("c"); !ok || v != 3 {
		t.Fatal()
	}
}
