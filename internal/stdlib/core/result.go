package core

// Result represents either a successful value (Ok) or an error (Err).
type Result[T any] struct {
	v   T
	err error
}

func Ok[T any](v T) Result[T] { return Result[T]{v: v, err: nil} }
func Err[T any](e error) Result[T] {
	var zero T
	return Result[T]{v: zero, err: e}
}

func (r Result[T]) IsOk() bool  { return r.err == nil }
func (r Result[T]) IsErr() bool { return r.err != nil }

func (r Result[T]) Unwrap() (T, error) { return r.v, r.err }

// MapResult applies f to the contained value when Ok.
func MapResult[T any, U any](r Result[T], f func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}

	return Ok(f(r.v))
}
