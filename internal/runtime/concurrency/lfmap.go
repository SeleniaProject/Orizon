package concurrency

import (
    "hash/fnv"
    "sync/atomic"
)

// LockFreeMap is a lock-free hash map with fixed bucket count. Buckets are
// singly-linked lists manipulated using atomic pointers. Values are updated by
// swapping the node's value pointer; deletion sets value to nil and may
// eventually physically unlink nodes during traversals.
type LockFreeMap[K comparable, V any] struct {
    buckets []atomic.Pointer[node[K, V]]
    mask    uint64
    hasher  func(K) uint64
}

type node[K comparable, V any] struct {
    key   K
    val   atomic.Pointer[valBox[V]]
    next  atomic.Pointer[node[K, V]]
}

type valBox[V any] struct{ v V }

// NewLockFreeMap creates a new lock-free map with bucket count rounded up to
// the next power of two. Caller must provide a hash function for keys.
func NewLockFreeMap[K comparable, V any](buckets uint64, hasher func(K) uint64) *LockFreeMap[K, V] {
    if buckets < 2 {
        buckets = 2
    }
    // round up to power of two
    n := uint64(1)
    for n < buckets {
        n <<= 1
    }
    m := &LockFreeMap[K, V]{
        buckets: make([]atomic.Pointer[node[K, V]], n),
        mask:    n - 1,
        hasher:  hasher,
    }
    return m
}

// NewStringLockFreeMap creates a map for string keys using FNV-1a hash.
func NewStringLockFreeMap[V any](buckets uint64) *LockFreeMap[string, V] {
    return NewLockFreeMap[string, V](buckets, func(k string) uint64 {
        h := fnv.New64a()
        _, _ = h.Write([]byte(k))
        return h.Sum64()
    })
}

func (m *LockFreeMap[K, V]) bucketIndex(key K) uint64 {
    return m.hasher(key) & m.mask
}

// Load returns the value for key if present.
func (m *LockFreeMap[K, V]) Load(key K) (V, bool) {
    var zero V
    b := &m.buckets[m.bucketIndex(key)]
    for n := b.Load(); n != nil; n = n.next.Load() {
        if n.key == key {
            vb := n.val.Load()
            if vb == nil {
                return zero, false
            }
            return vb.v, true
        }
    }
    return zero, false
}

// Store sets the value for key, inserting if absent.
func (m *LockFreeMap[K, V]) Store(key K, value V) {
    idx := m.bucketIndex(key)
    head := &m.buckets[idx]
    for {
        // search existing
        for n := head.Load(); n != nil; n = n.next.Load() {
            if n.key == key {
                box := &valBox[V]{v: value}
                n.val.Store(box)
                return
            }
        }
        // not found: insert new node at head
        newNode := &node[K, V]{key: key}
        newNode.val.Store(&valBox[V]{v: value})
        oldHead := head.Load()
        newNode.next.Store(oldHead)
        if head.CompareAndSwap(oldHead, newNode) {
            return
        }
    }
}

// LoadOrStore returns existing value if present, else stores and returns given value.
func (m *LockFreeMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
    if v, ok := m.Load(key); ok {
        return v, true
    }
    m.Store(key, value)
    return value, false
}

// Delete removes the key if present.
func (m *LockFreeMap[K, V]) Delete(key K) bool {
    idx := m.bucketIndex(key)
    head := &m.buckets[idx]
    for {
        prevPtr := head
        n := prevPtr.Load()
        for n != nil {
            next := n.next.Load()
            if n.key == key {
                // logical delete
                n.val.Store(nil)
                // try physical unlink
                if prevPtr.CompareAndSwap(n, next) {
                    // ok
                } else {
                    // if failed, another thread changed list; no problem
                }
                return true
            }
            prevPtr = &n.next
            n = next
        }
        return false
    }
}

// Range iterates key-value pairs; if fn returns false, iteration stops.
func (m *LockFreeMap[K, V]) Range(fn func(K, V) bool) {
    for i := range m.buckets {
        for n := m.buckets[i].Load(); n != nil; n = n.next.Load() {
            vb := n.val.Load()
            if vb == nil { continue }
            if !fn(n.key, vb.v) { return }
        }
    }
}


