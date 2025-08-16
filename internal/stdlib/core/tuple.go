package core

// Pair is a simple 2-tuple.
type Pair[A any, B any] struct {
	A A
	B B
}

// Triple is a simple 3-tuple.
type Triple[A any, B any, C any] struct {
	A A
	B B
	C C
}
