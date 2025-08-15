package assert

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Equal asserts that two comparable values are equal.
// It reports an error and returns false when they differ.
func Equal[T comparable](t testing.TB, got, want T, msgAndArgs ...any) bool {
	t.Helper()
	if got != want {
		fail(t, "Equal", got, want, msgAndArgs...)
		return false
	}
	return true
}

// NotEqual asserts that two comparable values are not equal.
func NotEqual[T comparable](t testing.TB, got, notWant T, msgAndArgs ...any) bool {
	t.Helper()
	if got == notWant {
		fail(t, "NotEqual", got, notWant, msgAndArgs...)
		return false
	}
	return true
}

// Nil asserts that the provided value is nil.
func Nil(t testing.TB, v any, msgAndArgs ...any) bool {
	t.Helper()
	if !isNil(v) {
		failMsg(t, "Nil", fmt.Sprintf("expected nil, got %T(%v)", v, v), msgAndArgs...)
		return false
	}
	return true
}

// NotNil asserts that the provided value is not nil.
func NotNil(t testing.TB, v any, msgAndArgs ...any) bool {
	t.Helper()
	if isNil(v) {
		failMsg(t, "NotNil", "unexpected nil", msgAndArgs...)
		return false
	}
	return true
}

// True asserts that cond is true.
func True(t testing.TB, cond bool, msgAndArgs ...any) bool {
	t.Helper()
	if !cond {
		failMsg(t, "True", "condition is false", msgAndArgs...)
		return false
	}
	return true
}

// False asserts that cond is false.
func False(t testing.TB, cond bool, msgAndArgs ...any) bool {
	t.Helper()
	if cond {
		failMsg(t, "False", "condition is true", msgAndArgs...)
		return false
	}
	return true
}

// Error asserts that err is non-nil.
func Error(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err == nil {
		failMsg(t, "Error", "expected error, got nil", msgAndArgs...)
		return false
	}
	return true
}

// NoError asserts that err is nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err != nil {
		failMsg(t, "NoError", fmt.Sprintf("unexpected error: %v", err), msgAndArgs...)
		return false
	}
	return true
}

// ErrorIs asserts that err matches target via errors.Is.
func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) bool {
	t.Helper()
	if !errors.Is(err, target) {
		failMsg(t, "ErrorIs", fmt.Sprintf("%v is not %v", err, target), msgAndArgs...)
		return false
	}
	return true
}

// Contains asserts that 's' contains 'substr'.
func Contains(t testing.TB, s, substr string, msgAndArgs ...any) bool {
	t.Helper()
	if !strings.Contains(s, substr) {
		failMsg(t, "Contains", fmt.Sprintf("%q does not contain %q", s, substr), msgAndArgs...)
		return false
	}
	return true
}

// Len asserts that the length of v equals want. Works with arrays, slices, maps, strings, channels.
func Len(t testing.TB, v any, want int, msgAndArgs ...any) bool {
	t.Helper()
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String, reflect.Chan:
		l := rv.Len()
		if l != want {
			failMsg(t, "Len", fmt.Sprintf("got len=%d, want %d", l, want), msgAndArgs...)
			return false
		}
		return true
	default:
		failMsg(t, "Len", fmt.Sprintf("unsupported kind %s", rv.Kind()), msgAndArgs...)
		return false
	}
}

// Panics asserts that fn panics. It returns true when a panic occurs.
func Panics(t testing.TB, fn func(), msgAndArgs ...any) (panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	fn()
	if !panicked {
		failMsg(t, "Panics", "function did not panic", msgAndArgs...)
	}
	return panicked
}

// Eventually asserts that condition becomes true within duration, checking every interval.
func Eventually(t testing.TB, condition func() bool, within, interval time.Duration, msgAndArgs ...any) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		if condition() {
			return true
		}
		if time.Now().After(deadline) {
			failMsg(t, "Eventually", "condition not met within duration", msgAndArgs...)
			return false
		}
		time.Sleep(interval)
	}
}

// Consistently asserts that condition stays true for the entire duration, checking every interval.
func Consistently(t testing.TB, condition func() bool, duration, interval time.Duration, msgAndArgs ...any) bool {
	t.Helper()
	deadline := time.Now().Add(duration)
	for {
		if !condition() {
			failMsg(t, "Consistently", "condition violated during duration", msgAndArgs...)
			return false
		}
		if time.Now().After(deadline) {
			return true
		}
		time.Sleep(interval)
	}
}

// WithinDuration asserts that two times are within delta.
func WithinDuration(t testing.TB, got, want time.Time, delta time.Duration, msgAndArgs ...any) bool {
	t.Helper()
	d := got.Sub(want)
	if d < 0 {
		d = -d
	}
	if d > delta {
		failMsg(t, "WithinDuration", fmt.Sprintf("|got-want|=%s exceeds %s", d, delta), msgAndArgs...)
		return false
	}
	return true
}

// fail formats a standard mismatch error with caller information.
func fail[T any](t testing.TB, op string, got, want T, msgAndArgs ...any) {
	loc := caller()
	base := fmt.Sprintf("%s: got=%v want=%v (%T/%T) at %s", op, got, want, got, want, loc)
	if len(msgAndArgs) > 0 {
		base += ": " + fmt.Sprint(msgAndArgs...)
	}
	t.Errorf(base)
}

func failMsg(t testing.TB, op string, detail string, msgAndArgs ...any) {
	loc := caller()
	base := fmt.Sprintf("%s: %s at %s", op, detail, loc)
	if len(msgAndArgs) > 0 {
		base += ": " + fmt.Sprint(msgAndArgs...)
	}
	t.Errorf(base)
}

func caller() string {
	// Skip runtime frames and assertion functions to point at the test site.
	for i := 2; i < 10; i++ {
		if pc, file, line, ok := runtime.Caller(i); ok {
			fn := runtime.FuncForPC(pc)
			name := ""
			if fn != nil {
				name = fn.Name()
			}
			if !strings.Contains(name, "assert.") {
				return fmt.Sprintf("%s:%d", file, line)
			}
		}
	}
	return "unknown:0"
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}
