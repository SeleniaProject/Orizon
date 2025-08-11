package asyncio

import (
	"sort"
	"sync"
	"sync/atomic"
)

// BytePool provides reusable byte buffers using size-bucketed sync.Pool.
// It reduces GC pressure for high-frequency I/O operations.
type BytePool struct {
	cfg     BytePoolConfig
	buckets []bucket
}

type bucket struct {
	size  int
	limit int64
	inuse int64
	pool  sync.Pool
}

type BytePoolConfig struct {
	// BucketSizes is the list of capacities for each pool bucket. Must be ascending.
	BucketSizes []int
	// MaxPerBucket is an approximate cap for retained buffers per bucket.
	MaxPerBucket int
}

// DefaultBytePool returns a BytePool with common network buffer sizes.
func DefaultBytePool() *BytePool {
	sizes := []int{1024, 2048, 4096, 8192, 16384, 32768, 65536}
	return NewBytePool(BytePoolConfig{BucketSizes: sizes, MaxPerBucket: 1024})
}

// NewBytePool creates a new BytePool with the provided configuration.
func NewBytePool(cfg BytePoolConfig) *BytePool {
	bs := append([]int(nil), cfg.BucketSizes...)
	sort.Ints(bs)
	buckets := make([]bucket, len(bs))
	for i, sz := range bs {
		b := bucket{size: sz, limit: int64(cfg.MaxPerBucket)}
		b.pool = sync.Pool{New: func() any { return make([]byte, sz) }}
		buckets[i] = b
	}
	return &BytePool{cfg: cfg, buckets: buckets}
}

// Get returns a buffer with capacity >= n and length 0. If n exceeds the
// largest bucket, a fresh buffer of exactly n is allocated and returned.
// Such oversize buffers are not pooled on Put.
func (bp *BytePool) Get(n int) []byte {
	if n <= 0 {
		n = 1
	}
	idx := bp.findBucket(n)
	if idx < 0 {
		return make([]byte, n)
	}
	b := &bp.buckets[idx]
	buf := b.pool.Get().([]byte)
	atomic.AddInt64(&b.inuse, 1)
	return buf[:0]
}

// Put returns a buffer to the pool if its capacity matches a known bucket and
// the approximate per-bucket retention limit has not been exceeded.
func (bp *BytePool) Put(buf []byte) {
	capn := cap(buf)
	if capn == 0 {
		return
	}
	idx := bp.findBucket(capn)
	if idx < 0 || bp.buckets[idx].size != capn {
		// not a managed size; drop
		return
	}
	b := &bp.buckets[idx]
	// decrement inuse; if above limit, drop the buffer instead of returning
	if cur := atomic.AddInt64(&b.inuse, -1); cur >= b.limit {
		return
	}
	b.pool.Put(buf[:capn])
}

// Stats returns current approximate in-use counters per bucket.
func (bp *BytePool) Stats() (sizes []int, inuse []int64) {
	sizes = make([]int, len(bp.buckets))
	inuse = make([]int64, len(bp.buckets))
	for i := range bp.buckets {
		sizes[i] = bp.buckets[i].size
		inuse[i] = atomic.LoadInt64(&bp.buckets[i].inuse)
	}
	return
}

func (bp *BytePool) findBucket(n int) int {
	i := sort.Search(len(bp.buckets), func(i int) bool { return bp.buckets[i].size >= n })
	if i >= len(bp.buckets) {
		return -1
	}
	return i
}
