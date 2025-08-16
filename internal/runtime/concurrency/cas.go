package concurrency

import "sync/atomic"

// CASUint64 performs an atomic compare-and-swap on a uint64 variable.
func CASUint64(addr *uint64, old, new uint64) bool {
	return atomic.CompareAndSwapUint64(addr, old, new)
}

// CASUint32 performs an atomic compare-and-swap on a uint32 variable.
func CASUint32(addr *uint32, old, new uint32) bool {
	return atomic.CompareAndSwapUint32(addr, old, new)
}

// CASInt64 performs an atomic compare-and-swap on an int64 variable.
func CASInt64(addr *int64, old, new int64) bool { return atomic.CompareAndSwapInt64(addr, old, new) }

// Load/Store helpers
func LoadUint64(addr *uint64) uint64     { return atomic.LoadUint64(addr) }
func StoreUint64(addr *uint64, v uint64) { atomic.StoreUint64(addr, v) }
