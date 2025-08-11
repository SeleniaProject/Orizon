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
    if !q.Dequeue(&v) || v != 1 { t.Fatalf("got %d", v) }
    if !q.Dequeue(&v) || v != 2 { t.Fatalf("got %d", v) }
    if q.Dequeue(&v) { t.Fatal("expected empty") }
}

func TestMPMCQueue_Concurrent(t *testing.T) {
    q := NewMPMCQueue[int](1024)
    var produced, consumed uint64
    wg := sync.WaitGroup{}

    producers := 4
    consumers := 4
    itemsPerProducer := 5000

    wg.Add(producers)
    for p := 0; p < producers; p++ {
        go func(id int) {
            defer wg.Done()
            for i := 0; i < itemsPerProducer; i++ {
                for !q.Enqueue(i + id*itemsPerProducer) {}
                atomic.AddUint64(&produced, 1)
            }
        }(p)
    }

    done := make(chan struct{})
    wg.Add(consumers)
    for c := 0; c < consumers; c++ {
        go func() {
            defer wg.Done()
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

    wg.Wait()
    // wait for draining
    for atomic.LoadUint64(&consumed) < uint64(producers*itemsPerProducer) {
        var v int
        if q.Dequeue(&v) { atomic.AddUint64(&consumed, 1) }
    }
    close(done)

    if produced != consumed {
        t.Fatalf("mismatch produced=%d consumed=%d", produced, consumed)
    }
}


