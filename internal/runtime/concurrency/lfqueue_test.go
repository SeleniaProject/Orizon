package concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestMPMCQueue_Basic(t *testing.T) {
	q := NewMPMCQueue[int](8)
	if !q.Enqueue(1) || !q.Enqueue(2) {
		t.Fatal("enqueue failed")
	}
	var v int
	if !q.Dequeue(&v) || v != 1 {
		t.Fatalf("got %d", v)
	}
	if !q.Dequeue(&v) || v != 2 {
		t.Fatalf("got %d", v)
	}
	if q.Dequeue(&v) {
		t.Fatal("expected empty")
	}
}

func TestMPMCQueue_Concurrent(t *testing.T) {
	q := NewMPMCQueue[int](1024)
	var produced, consumed uint64
	producers := 4
	consumers := 4
	itemsPerProducer := 4000

	// start producers
	wgProd := sync.WaitGroup{}
	wgProd.Add(producers)
	for p := 0; p < producers; p++ {
		go func(id int) {
			defer wgProd.Done()
			for i := 0; i < itemsPerProducer; i++ {
				for !q.Enqueue(i + id*itemsPerProducer) {
				}
				atomic.AddUint64(&produced, 1)
			}
		}(p)
	}

	// start consumers
	done := make(chan struct{})
	wgCons := sync.WaitGroup{}
	wgCons.Add(consumers)
	for c := 0; c < consumers; c++ {
		go func() {
			defer wgCons.Done()
			var v int
			for {
				select {
				case <-done:
					return
				default:
				}
				if q.Dequeue(&v) {
					atomic.AddUint64(&consumed, 1)
				}
			}
		}()
	}

	// wait producers finished
	wgProd.Wait()
	// drain remaining
	total := uint64(producers * itemsPerProducer)
	for atomic.LoadUint64(&consumed) < total {
		var v int
		if q.Dequeue(&v) {
			atomic.AddUint64(&consumed, 1)
		}
	}
	// stop consumers and wait
	close(done)
	wgCons.Wait()

	if produced != consumed {
		t.Fatalf("mismatch produced=%d consumed=%d", produced, consumed)
	}
}
