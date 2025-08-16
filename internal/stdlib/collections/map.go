package collections

// Map is a generic hash map wrapper that provides a richer, safe API.
// Iteration order is not guaranteed (same as Go's map).
type Map[K comparable, V any] struct {
	m map[K]V
}

// NewMap creates an empty map with optional capacity hint.
func NewMap[K comparable, V any](capHint int) *Map[K, V] {
	if capHint < 0 {
		capHint = 0
	}
	return &Map[K, V]{m: make(map[K]V, capHint)}
}

// NewMapFrom creates a map from key-value pairs.
func NewMapFrom[K comparable, V any](pairs map[K]V) *Map[K, V] {
	if pairs == nil {
		pairs = make(map[K]V)
	}
	// shallow copy to decouple from input
	out := make(map[K]V, len(pairs))
	for k, v := range pairs {
		out[k] = v
	}
	return &Map[K, V]{m: out}
}

// Len returns the number of entries.
func (mm *Map[K, V]) Len() int { return len(mm.m) }

// IsEmpty reports whether the map has no entries.
func (mm *Map[K, V]) IsEmpty() bool { return len(mm.m) == 0 }

// Has reports whether key exists.
func (mm *Map[K, V]) Has(k K) bool { _, ok := mm.m[k]; return ok }

// Get returns the value for key and a boolean indicating presence.
func (mm *Map[K, V]) Get(k K) (V, bool) { v, ok := mm.m[k]; return v, ok }

// GetOrDefault returns the value for key or def if absent.
func (mm *Map[K, V]) GetOrDefault(k K, def V) V {
	if v, ok := mm.m[k]; ok {
		return v
	}
	return def
}

// GetOrInsert returns current value if present; otherwise inserts value from fn and returns it.
func (mm *Map[K, V]) GetOrInsert(k K, fn func() V) V {
	if v, ok := mm.m[k]; ok {
		return v
	}
	v := fn()
	mm.m[k] = v
	return v
}

// Put sets key to value, returning previous value and whether it existed.
func (mm *Map[K, V]) Put(k K, v V) (prev V, replaced bool) {
	prev, replaced = mm.m[k]
	mm.m[k] = v
	return
}

// Upsert updates the value using a function based on existing presence.
// fn receives (oldValue, existed) and should return the new value.
func (mm *Map[K, V]) Upsert(k K, fn func(old V, existed bool) V) V {
	old, ok := mm.m[k]
	newV := fn(old, ok)
	mm.m[k] = newV
	return newV
}

// Delete removes key and returns previous value and whether it existed.
func (mm *Map[K, V]) Delete(k K) (prev V, existed bool) {
	prev, existed = mm.m[k]
	if existed {
		delete(mm.m, k)
	}
	return
}

// Keys returns a snapshot of keys.
func (mm *Map[K, V]) Keys() []K {
	keys := make([]K, 0, len(mm.m))
	for k := range mm.m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a snapshot of values.
func (mm *Map[K, V]) Values() []V {
	vals := make([]V, 0, len(mm.m))
	for _, v := range mm.m {
		vals = append(vals, v)
	}
	return vals
}

// Clear removes all entries.
func (mm *Map[K, V]) Clear() {
	for k := range mm.m {
		delete(mm.m, k)
	}
}

// Clone returns a shallow copy of the map.
func (mm *Map[K, V]) Clone() *Map[K, V] {
	copyMap := make(map[K]V, len(mm.m))
	for k, v := range mm.m {
		copyMap[k] = v
	}
	return &Map[K, V]{m: copyMap}
}

// Merge inserts or overwrites entries from other into this map.
func (mm *Map[K, V]) Merge(other *Map[K, V]) {
	for k, v := range other.m {
		mm.m[k] = v
	}
}

// ForEach iterates over entries. Iteration order is unspecified.
func (mm *Map[K, V]) ForEach(fn func(k K, v V)) {
	for k, v := range mm.m {
		fn(k, v)
	}
}

// MapValues transforms values using fn and returns a new map with same keys.
func MapValues[K comparable, V any, U any](mm *Map[K, V], fn func(V) U) *Map[K, U] {
	out := make(map[K]U, len(mm.m))
	for k, v := range mm.m {
		out[k] = fn(v)
	}
	return &Map[K, U]{m: out}
}

// Filter returns a new map with entries satisfying pred.
func (mm *Map[K, V]) Filter(pred func(k K, v V) bool) *Map[K, V] {
	out := make(map[K]V, len(mm.m))
	for k, v := range mm.m {
		if pred(k, v) {
			out[k] = v
		}
	}
	return &Map[K, V]{m: out}
}

// Iterator provides a safe snapshot-based iterator over map entries.
type MapIterator[K comparable, V any] struct {
	keys []K
	m    map[K]V
	i    int
}

// Iter creates an iterator. It snapshots keys at creation time.
func (mm *Map[K, V]) Iter() MapIterator[K, V] {
	return MapIterator[K, V]{keys: mm.Keys(), m: mm.m, i: 0}
}

// Next returns next key-value pair and true, or zero values,false when done.
func (it *MapIterator[K, V]) Next() (K, V, bool) {
	var zk K
	var zv V
	if it.i >= len(it.keys) {
		return zk, zv, false
	}
	k := it.keys[it.i]
	it.i++
	v, ok := it.m[k]
	if !ok {
		return zk, zv, false
	}
	return k, v, true
}
