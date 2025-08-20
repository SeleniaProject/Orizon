package core

// Option represents an optional value.
// Some(v) means a present value, None means absence.
type Option[T any] struct {
	v     T
	valid bool
}

// Some constructs an Option with a value.
func Some[T any](v T) Option[T] { return Option[T]{v: v, valid: true} }

// None constructs an empty Option.
func None[T any]() Option[T] {
	var zero T
	return Option[T]{v: zero, valid: false}
}

// IsSome reports whether the option holds a value.
func (o Option[T]) IsSome() bool { return o.valid }

// IsNone reports whether the option is empty.
func (o Option[T]) IsNone() bool { return !o.valid }

// Unwrap returns the value and whether it was present.
func (o Option[T]) Unwrap() (T, bool) { return o.v, o.valid }

// Or returns the value if present, otherwise fallback.
func (o Option[T]) Or(fallback T) T {
	if o.valid {
		return o.v
	}

	return fallback
}

// Map applies f to the value if present.
func Map[T any, U any](o Option[T], f func(T) U) Option[U] {
	if o.valid {
		return Some(f(o.v))
	}

	return None[U]()
}
