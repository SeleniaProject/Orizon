package collections

import (
	"sort"
)

// Vector is a simple dynamic array with a minimal, safe API.
// It favors clarity and testability over micro-optimizations.
// Zero value is ready to use.
type Vector[T any] struct {
	buf []T
}

// NewVector creates a vector with an optional initial capacity.
func NewVector[T any](capHint int) *Vector[T] {
	if capHint < 0 {
		capHint = 0
	}
	return &Vector[T]{buf: make([]T, 0, capHint)}
}

// Of constructs a vector from values.
func Of[T any](xs ...T) *Vector[T] {
	v := NewVector[T](len(xs))
	v.Append(xs...)
	return v
}

// NewFromSlice builds a vector from slice; if copySlice is true, the data is copied.
func NewFromSlice[T any](s []T, copySlice bool) *Vector[T] {
	if !copySlice {
		// Use s directly but ensure non-nil slice for zero-length
		if s == nil {
			s = make([]T, 0)
		}
		return &Vector[T]{buf: s}
	}
	v := NewVector[T](len(s))
	v.Append(s...)
	return v
}

// Len returns the number of elements.
func (v *Vector[T]) Len() int { return len(v.buf) }

// Cap returns the underlying capacity.
func (v *Vector[T]) Cap() int { return cap(v.buf) }

// Append adds elements to the end.
func (v *Vector[T]) Append(xs ...T) {
	v.buf = append(v.buf, xs...)
}

// Push is an alias of Append for a single element.
func (v *Vector[T]) Push(x T) { v.buf = append(v.buf, x) }

// Pop removes and returns the last element. Returns false if empty.
func (v *Vector[T]) Pop() (T, bool) {
	var zero T
	n := len(v.buf)
	if n == 0 {
		return zero, false
	}
	x := v.buf[n-1]
	// Avoid memory leak of references.
	var z T
	v.buf[n-1] = z
	v.buf = v.buf[:n-1]
	return x, true
}

// Get returns the element at index i. Returns false if out of range.
func (v *Vector[T]) Get(i int) (T, bool) {
	var zero T
	if i < 0 || i >= len(v.buf) {
		return zero, false
	}
	return v.buf[i], true
}

// Set sets the element at index i. Returns false if out of range.
func (v *Vector[T]) Set(i int, x T) bool {
	if i < 0 || i >= len(v.buf) {
		return false
	}
	v.buf[i] = x
	return true
}

// At panics if out of range. Prefer Get for safe reads.
func (v *Vector[T]) At(i int) T { return v.buf[i] }

// ToSlice returns a copy of the underlying slice to prevent external mutation.
func (v *Vector[T]) ToSlice() []T {
	out := make([]T, len(v.buf))
	copy(out, v.buf)
	return out
}

// UnsafeSlice exposes the underlying slice for performance-sensitive paths.
// Mutating the returned slice will affect the vector.
func (v *Vector[T]) UnsafeSlice() []T { return v.buf }

// Clear removes all elements, keeping capacity.
func (v *Vector[T]) Clear() {
	// Zero elements to help GC for reference types.
	for i := range v.buf {
		var z T
		v.buf[i] = z
	}
	v.buf = v.buf[:0]
}

// ClearAndShrink removes all elements and shrinks capacity to zero.
func (v *Vector[T]) ClearAndShrink() {
	v.Clear()
	if cap(v.buf) > 0 {
		v.buf = make([]T, 0)
	}
}

// EnsureCapacity grows capacity to at least n.
func (v *Vector[T]) EnsureCapacity(n int) {
	if n <= cap(v.buf) {
		return
	}
	// Grow strategy: double until >= n
	newCap := cap(v.buf)
	if newCap == 0 {
		newCap = 1
	}
	for newCap < n {
		if newCap > (1 << 30) { // guard against overflow on 32-bit
			newCap = n
			break
		}
		newCap *= 2
	}
	nb := make([]T, len(v.buf), newCap)
	copy(nb, v.buf)
	v.buf = nb
}

// Reserve ensures room for at least additional elements without reallocation.
func (v *Vector[T]) Reserve(additional int) {
	if additional <= 0 {
		return
	}
	need := len(v.buf) + additional
	v.EnsureCapacity(need)
}

// ShrinkToFit trims capacity to current length.
func (v *Vector[T]) ShrinkToFit() {
	if len(v.buf) == cap(v.buf) {
		return
	}
	nb := make([]T, len(v.buf))
	copy(nb, v.buf)
	v.buf = nb
}

// Resize changes length to n. If n > len, new slots are filled with fill.
// Returns false if n < 0.
func (v *Vector[T]) Resize(n int, fill T) bool {
	if n < 0 {
		return false
	}
	cur := len(v.buf)
	if n <= cur {
		// shrink
		// zero out truncated tail for GC friendliness
		for i := n; i < cur; i++ {
			var z T
			v.buf[i] = z
		}
		v.buf = v.buf[:n]
		return true
	}
	// grow
	v.EnsureCapacity(n)
	// extend slice
	need := n - cur
	for i := 0; i < need; i++ {
		v.buf = append(v.buf, fill)
	}
	return true
}

// Clone returns a deep copy of the vector.
func (v *Vector[T]) Clone() *Vector[T] { return NewFromSlice(v.ToSlice(), false) }

// Extend appends elements.
func (v *Vector[T]) Extend(xs ...T) { v.Append(xs...) }

// ExtendVector appends all elements from other.
func (v *Vector[T]) ExtendVector(other *Vector[T]) { v.Append(other.buf...) }

// IsEmpty reports whether the vector has no elements.
func (v *Vector[T]) IsEmpty() bool { return len(v.buf) == 0 }

// Front returns the first element; false if empty.
func (v *Vector[T]) Front() (T, bool) {
	var zero T
	if len(v.buf) == 0 {
		return zero, false
	}
	return v.buf[0], true
}

// Back returns the last element; false if empty.
func (v *Vector[T]) Back() (T, bool) { return v.Pop() }

// Peek returns the last element without removing it; false if empty.
func (v *Vector[T]) Peek() (T, bool) {
	var zero T
	n := len(v.buf)
	if n == 0 {
		return zero, false
	}
	return v.buf[n-1], true
}

// Iterator provides sequential iteration over a Vector.
type Iterator[T any] struct {
	v *Vector[T]
	i int
}

// Iter creates a new iterator starting at index 0.
func (v *Vector[T]) Iter() Iterator[T] { return Iterator[T]{v: v, i: 0} }

// Next returns the next element, or false when finished.
func (it *Iterator[T]) Next() (T, bool) {
	var zero T
	if it.i >= len(it.v.buf) {
		return zero, false
	}
	x := it.v.buf[it.i]
	it.i++
	return x, true
}

// ForEach applies fn to each element.
func (v *Vector[T]) ForEach(fn func(i int, x T)) {
	for i := range v.buf {
		fn(i, v.buf[i])
	}
}

// Filter returns a new Vector with elements that satisfy pred.
func (v *Vector[T]) Filter(pred func(T) bool) *Vector[T] {
	out := NewVector[T](len(v.buf))
	for _, x := range v.buf {
		if pred(x) {
			out.buf = append(out.buf, x)
		}
	}
	return out
}

// Insert inserts x at position i, shifting subsequent elements to the right.
// Returns false if i is out of range (i<0 || i>len).
func (v *Vector[T]) Insert(i int, x T) bool {
	if i < 0 || i > len(v.buf) {
		return false
	}
	v.EnsureCapacity(len(v.buf) + 1)
	v.buf = append(v.buf, x) // grow by 1
	copy(v.buf[i+1:], v.buf[i:len(v.buf)-1])
	v.buf[i] = x
	return true
}

// InsertAll inserts xs starting at i.
func (v *Vector[T]) InsertAll(i int, xs ...T) bool {
	if i < 0 || i > len(v.buf) {
		return false
	}
	k := len(xs)
	if k == 0 {
		return true
	}
	v.EnsureCapacity(len(v.buf) + k)
	oldLen := len(v.buf)
	// grow by k
	v.buf = append(v.buf, xs...)
	// shift the existing tail
	copy(v.buf[i+k:], v.buf[i:oldLen])
	// write xs
	copy(v.buf[i:i+k], xs)
	return true
}

// RemoveAt removes element at index i, returning it and true, or zero,false if out of range.
func (v *Vector[T]) RemoveAt(i int) (T, bool) {
	var zero T
	if i < 0 || i >= len(v.buf) {
		return zero, false
	}
	x := v.buf[i]
	copy(v.buf[i:], v.buf[i+1:])
	// zero last
	var z T
	v.buf[len(v.buf)-1] = z
	v.buf = v.buf[:len(v.buf)-1]
	return x, true
}

// RemoveRange removes [from,to) range. Returns false if indices invalid.
func (v *Vector[T]) RemoveRange(from, to int) bool {
	if from < 0 || to < from || to > len(v.buf) {
		return false
	}
	n := to - from
	if n == 0 {
		return true
	}
	copy(v.buf[from:], v.buf[to:])
	// zero tail
	end := len(v.buf)
	for i := end - n; i < end; i++ {
		var z T
		v.buf[i] = z
	}
	v.buf = v.buf[:end-n]
	return true
}

// RemoveIf removes elements satisfying pred, returns number removed.
func (v *Vector[T]) RemoveIf(pred func(T) bool) int {
	out := v.buf[:0]
	removed := 0
	for _, x := range v.buf {
		if pred(x) {
			removed++
			continue
		}
		out = append(out, x)
	}
	// zero tail
	for i := len(out); i < len(v.buf); i++ {
		var z T
		v.buf[i] = z
	}
	v.buf = out
	return removed
}

// Swap swaps elements at i and j. Returns false if out of range.
func (v *Vector[T]) Swap(i, j int) bool {
	if i < 0 || j < 0 || i >= len(v.buf) || j >= len(v.buf) {
		return false
	}
	v.buf[i], v.buf[j] = v.buf[j], v.buf[i]
	return true
}

// Reverse reverses elements in place.
func (v *Vector[T]) Reverse() {
	for i, j := 0, len(v.buf)-1; i < j; i, j = i+1, j-1 {
		v.buf[i], v.buf[j] = v.buf[j], v.buf[i]
	}
}

// Sort sorts elements in-place using the provided less(a,b) comparator.
func (v *Vector[T]) Sort(less func(a, b T) bool) {
	sort.Slice(v.buf, func(i, j int) bool { return less(v.buf[i], v.buf[j]) })
}

// SortStable performs a stable sort using less comparator.
func (v *Vector[T]) SortStable(less func(a, b T) bool) {
	sort.SliceStable(v.buf, func(i, j int) bool { return less(v.buf[i], v.buf[j]) })
}

// BinarySearch searches for target in a sorted vector using the provided less comparator.
// Returns index and whether it was found. If not found, index is the insertion point.
func (v *Vector[T]) BinarySearch(target T, less func(a, b T) bool) (int, bool) {
	n := len(v.buf)
	i := sort.Search(n, func(i int) bool { return !less(v.buf[i], target) })
	if i < n && !less(target, v.buf[i]) && !less(v.buf[i], target) {
		return i, true
	}
	return i, false
}

// IndexOf returns the first index i where pred(v[i]) is true, or -1.
func (v *Vector[T]) IndexOf(pred func(T) bool) int {
	for i, x := range v.buf {
		if pred(x) {
			return i
		}
	}
	return -1
}

// Find returns the first element satisfying pred.
func (v *Vector[T]) Find(pred func(T) bool) (T, bool) {
	var zero T
	idx := v.IndexOf(pred)
	if idx < 0 {
		return zero, false
	}
	return v.buf[idx], true
}

// Any reports whether any element satisfies pred.
func (v *Vector[T]) Any(pred func(T) bool) bool { return v.IndexOf(pred) >= 0 }

// All reports whether all elements satisfy pred (vacuously true for empty).
func (v *Vector[T]) All(pred func(T) bool) bool {
	for _, x := range v.buf {
		if !pred(x) {
			return false
		}
	}
	return true
}

// MapVector produces a new Vector with elements transformed by fn.
func MapVector[T any, U any](v *Vector[T], fn func(T) U) *Vector[U] {
	out := NewVector[U](len(v.buf))
	for _, x := range v.buf {
		out.buf = append(out.buf, fn(x))
	}
	return out
}

// ReduceVector folds the vector from left to right.
func ReduceVector[T any, U any](v *Vector[T], init U, fn func(U, T) U) U {
	acc := init
	for _, x := range v.buf {
		acc = fn(acc, x)
	}
	return acc
}
