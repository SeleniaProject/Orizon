package concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestLockFreeMap_String_Basic(t *testing.T) {
	m := NewStringLockFreeMap[int](64)
	if _, ok := m.Load("x"); ok {
		t.Fatal("unexpected present")
	}

	m.Store("x", 10)

	if v, ok := m.Load("x"); !ok || v != 10 {
		t.Fatalf("got %v %v", v, ok)
	}

	if v, existed := m.LoadOrStore("x", 20); !existed || v != 10 {
		t.Fatalf("loadorstore: %v %v", v, existed)
	}

	if !m.Delete("x") {
		t.Fatal("delete failed")
	}

	if _, ok := m.Load("x"); ok {
		t.Fatal("still present after delete")
	}
}

func TestLockFreeMap_Concurrent(t *testing.T) {
	m := NewStringLockFreeMap[int](256)
	keys := 1000
	writers := 4
	readers := 4

	var seen uint64

	// writers.
	wgW := sync.WaitGroup{}
	wgW.Add(writers)

	for w := 0; w < writers; w++ {
		go func(id int) {
			defer wgW.Done()

			for i := 0; i < keys; i++ {
				k := keyOf(id, i)
				m.Store(k, i)
			}
		}(w)
	}

	// readers.
	done := make(chan struct{})
	wgR := sync.WaitGroup{}
	wgR.Add(readers)

	for r := 0; r < readers; r++ {
		go func() {
			defer wgR.Done()

			for {
				select {
				case <-done:
					return
				default:
				}
				m.Range(func(k string, v int) bool {
					atomic.AddUint64(&seen, 1)
					return true
				})
			}
		}()
	}

	// wait for writers, then stop readers and wait.
	wgW.Wait()
	close(done)
	wgR.Wait()

	if seen == 0 {
		t.Fatal("no reads observed")
	}
}

func keyOf(id, i int) string { return string(rune('a'+id)) + ":" + string(rune('0'+(i%10))) }
