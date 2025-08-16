package collections

// Set is a hash set built on top of Map[T,struct{}].
type Set[T comparable] struct {
	m map[T]struct{}
}

// NewSet creates an empty set with optional capacity hint.
func NewSet[T comparable](capHint int) *Set[T] {
	if capHint < 0 {
		capHint = 0
	}
	return &Set[T]{m: make(map[T]struct{}, capHint)}
}

// NewSetFrom returns a set initialized with xs.
func NewSetFrom[T comparable](xs ...T) *Set[T] {
	s := NewSet[T](len(xs))
	for _, x := range xs {
		s.m[x] = struct{}{}
	}
	return s
}

// Len returns the number of elements.
func (s *Set[T]) Len() int { return len(s.m) }

// IsEmpty reports whether the set is empty.
func (s *Set[T]) IsEmpty() bool { return len(s.m) == 0 }

// Has reports whether x is in the set.
func (s *Set[T]) Has(x T) bool { _, ok := s.m[x]; return ok }

// Add inserts x; returns false if already present.
func (s *Set[T]) Add(x T) bool {
	_, existed := s.m[x]
	if !existed {
		s.m[x] = struct{}{}
	}
	return !existed
}

// AddAll inserts many values, returning the count newly added.
func (s *Set[T]) AddAll(xs ...T) int {
	added := 0
	for _, x := range xs {
		if s.Add(x) {
			added++
		}
	}
	return added
}

// Remove deletes x; returns true if it existed.
func (s *Set[T]) Remove(x T) bool {
	_, ok := s.m[x]
	if ok {
		delete(s.m, x)
	}
	return ok
}

// Clear removes all elements.
func (s *Set[T]) Clear() {
	for k := range s.m {
		delete(s.m, k)
	}
}

// Clone returns a shallow copy.
func (s *Set[T]) Clone() *Set[T] {
	m := make(map[T]struct{}, len(s.m))
	for k := range s.m {
		m[k] = struct{}{}
	}
	return &Set[T]{m: m}
}

// ToSlice returns a snapshot slice of elements.
func (s *Set[T]) ToSlice() []T {
	out := make([]T, 0, len(s.m))
	for k := range s.m {
		out = append(out, k)
	}
	return out
}

// Union returns a new set containing elements in either a or b.
func Union[T comparable](a, b *Set[T]) *Set[T] {
	m := make(map[T]struct{}, a.Len()+b.Len())
	for k := range a.m {
		m[k] = struct{}{}
	}
	for k := range b.m {
		m[k] = struct{}{}
	}
	return &Set[T]{m: m}
}

// Intersect returns a new set containing elements common to a and b.
func Intersect[T comparable](a, b *Set[T]) *Set[T] {
	m := make(map[T]struct{})
	// iterate smaller set
	var small, large *Set[T]
	if a.Len() <= b.Len() {
		small, large = a, b
	} else {
		small, large = b, a
	}
	for k := range small.m {
		if _, ok := large.m[k]; ok {
			m[k] = struct{}{}
		}
	}
	return &Set[T]{m: m}
}

// Difference returns elements in a but not in b.
func Difference[T comparable](a, b *Set[T]) *Set[T] {
	m := make(map[T]struct{}, a.Len())
	for k := range a.m {
		if _, ok := b.m[k]; !ok {
			m[k] = struct{}{}
		}
	}
	return &Set[T]{m: m}
}
