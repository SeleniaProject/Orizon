package concurrency

import (
	"fmt"
	"sync"
)

// ConcurrentMap is a simple striped concurrent map.
type ConcurrentMap[K comparable, V any] struct {
	shards []cmShard[K, V]
	hashFn func(K) uint64
}

type cmShard[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewConcurrentMap creates a concurrent map with shardCount shards.
// Uses a default hasher suitable for common key types (strings and integers).
func NewConcurrentMap[K comparable, V any](shardCount int) *ConcurrentMap[K, V] {
	return NewConcurrentMapWithHasher[K, V](shardCount, defaultHasher[K])
}

// NewConcurrentMapWithHasher creates a concurrent map with a custom hash function.
func NewConcurrentMapWithHasher[K comparable, V any](shardCount int, hasher func(K) uint64) *ConcurrentMap[K, V] {
	if shardCount <= 0 {
		shardCount = 32
	}
	shards := make([]cmShard[K, V], shardCount)
	for i := range shards {
		shards[i].m = make(map[K]V)
	}
	if hasher == nil {
		hasher = defaultHasher[K]
	}
	return &ConcurrentMap[K, V]{shards: shards, hashFn: hasher}
}

func (c *ConcurrentMap[K, V]) shard(k K) *cmShard[K, V] {
	h := c.hashFn(k)
	idx := int(h % uint64(len(c.shards)))
	return &c.shards[idx]
}

// Set sets key to value.
func (c *ConcurrentMap[K, V]) Set(k K, v V) {
	s := c.shard(k)
	s.mu.Lock()
	s.m[k] = v
	s.mu.Unlock()
}

// Get returns value and ok.
func (c *ConcurrentMap[K, V]) Get(k K) (V, bool) {
	s := c.shard(k)
	s.mu.RLock()
	v, ok := s.m[k]
	s.mu.RUnlock()
	return v, ok
}

// Delete removes a key.
func (c *ConcurrentMap[K, V]) Delete(k K) {
	s := c.shard(k)
	s.mu.Lock()
	delete(s.m, k)
	s.mu.Unlock()
}

// Len returns an approximate size (may be slightly stale under concurrency).
func (c *ConcurrentMap[K, V]) Len() int {
	t := 0
	for i := range c.shards {
		c.shards[i].mu.RLock()
		t += len(c.shards[i].m)
		c.shards[i].mu.RUnlock()
	}
	return t
}

// Range iterates all key-value pairs; if fn returns false, iteration stops early.
func (c *ConcurrentMap[K, V]) Range(fn func(K, V) bool) {
	for i := range c.shards {
		sh := &c.shards[i]
		sh.mu.RLock()
		for k, v := range sh.m {
			if !fn(k, v) {
				sh.mu.RUnlock()
				return
			}
		}
		sh.mu.RUnlock()
	}
}

// defaultHasher tries to be efficient for common key types and falls back to fmt-based hashing.
func defaultHasher[K comparable](k K) uint64 {
	switch v := any(k).(type) {
	case string:
		// FNV-1a 64-bit
		var h uint64 = 1469598103934665603
		for i := 0; i < len(v); i++ {
			h ^= uint64(v[i])
			h *= 1099511628211
		}
		return h
	case int:
		x := uint64(v)
		return mix64(x)
	case int32:
		return mix64(uint64(uint32(v)))
	case int64:
		return mix64(uint64(v))
	case uint:
		return mix64(uint64(v))
	case uint32:
		return mix64(uint64(v))
	case uint64:
		return mix64(v)
	default:
		// Fall back to string representation.
		// Note: slower but generic.
		s := fmt.Sprintf("%v", v)
		var h uint64 = 1469598103934665603
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= 1099511628211
		}
		return h
	}
}

func mix64(x uint64) uint64 {
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return x
}
