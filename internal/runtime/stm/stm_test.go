package stm

import (
    "sync"
    "testing"
)

func TestSTM_Basic(t *testing.T) {
    acc := NewTVar[int](0)
    add := func(delta int) error {
        return Run[int](0, func(tx *Txn[int]) error {
            v := tx.Read(acc)
            tx.Write(acc, v+delta)
            return nil
        })
    }
    if err := add(1); err != nil { t.Fatalf("add: %v", err) }
    if err := add(2); err != nil { t.Fatalf("add: %v", err) }
    if v := acc.val.Load().(int); v != 3 { t.Fatalf("got %d", v) }
}

func TestSTM_Concurrent(t *testing.T) {
    acc := NewTVar[int](0)
    wg := sync.WaitGroup{}
    n := 8
    incs := 2000
    wg.Add(n)
    for i := 0; i < n; i++ {
        go func() {
            defer wg.Done()
            for j := 0; j < incs; j++ {
                _ = Run[int](0, func(tx *Txn[int]) error {
                    v := tx.Read(acc)
                    tx.Write(acc, v+1)
                    return nil
                })
            }
        }()
    }
    wg.Wait()
    if v := acc.val.Load().(int); v != n*incs { t.Fatalf("expected %d got %d", n*incs, v) }
}


