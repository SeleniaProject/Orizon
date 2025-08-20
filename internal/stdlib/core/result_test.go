package core

import (
	"errors"
	"testing"
)

func TestResultBasic(t *testing.T) {
	r := Ok(5)
	if !r.IsOk() || r.IsErr() {
		t.Fatalf("ok flags")
	}

	v, err := r.Unwrap()
	if err != nil || v != 5 {
		t.Fatalf("unwrap ok")
	}

	e := Err[int](errors.New("x"))
	if e.IsOk() || !e.IsErr() {
		t.Fatalf("err flags")
	}

	_, err = e.Unwrap()
	if err == nil {
		t.Fatalf("expected error")
	}

	m := MapResult(r, func(x int) int { return x + 1 })

	v, err = m.Unwrap()
	if err != nil || v != 6 {
		t.Fatalf("map ok")
	}

	m2 := MapResult(e, func(x int) int { return x + 1 })
	if _, err = m2.Unwrap(); err == nil {
		t.Fatalf("map should keep error")
	}
}
