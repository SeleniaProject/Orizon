package concurrency

import (
	"sync"
	"testing"
)

func TestConcurrentMapBasic(t *testing.T) {
	m := NewConcurrentMap[string, int](8)
	m.Set("a", 1)
	m.Set("b", 2)

	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Fatalf("want 1, got %v %v", v, ok)
	}

	if v, ok := m.Get("b"); !ok || v != 2 {
		t.Fatalf("want 2, got %v %v", v, ok)
	}

	m.Delete("a")

	if _, ok := m.Get("a"); ok {
		t.Fatalf("expected a deleted")
	}
}

func TestConcurrentMapParallel(t *testing.T) {
	m := NewConcurrentMap[int, int](32)

	const N = 1000

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		for i := 0; i < N; i++ {
			m.Set(i, i*i)
		}

		wg.Done()
	}()
	go func() {
		for i := 0; i < N; i++ {
			m.Get(i)
		}

		wg.Done()
	}()
	wg.Wait()

	if m.Len() == 0 {
		t.Fatalf("len should be > 0")
	}
}

func TestConcurrentMapRange(t *testing.T) {
	m := NewConcurrentMap[string, int](4)
	m.Set("x", 10)
	m.Set("y", 20)

	sum := 0

	m.Range(func(_ string, v int) bool {
		sum += v
		return true
	})

	if sum != 30 {
		t.Fatalf("sum = %d", sum)
	}
}
