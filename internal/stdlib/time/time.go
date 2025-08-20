package timex

import (
	"context"
	"time"
)

// Package timex provides small time utilities for stdlib.

// RetryBackoff returns a channel that ticks at an exponential backoff schedule.
// starting at base and multiplying by factor up to max.
// count>0 limits the number of ticks; if count<=0, ticks forever.
func RetryBackoff(base time.Duration, factor float64, max time.Duration, count int) <-chan time.Time {
	ch := make(chan time.Time)

	if base <= 0 {
		base = 10 * time.Millisecond
	}

	if factor <= 0 {
		factor = 2.0
	}

	go func() {
		defer close(ch)

		d := base
		i := 0

		for count <= 0 || i < count {
			time.Sleep(d)
			ch <- time.Now()

			i++

			next := time.Duration(float64(d) * factor)
			if max > 0 && next > max {
				next = max
			}

			if next <= 0 {
				next = d
			}

			d = next
		}
	}()

	return ch
}

// RetryBackoffContext is like RetryBackoff but cancels early when ctx is done.
func RetryBackoffContext(ctx context.Context, base time.Duration, factor float64, max time.Duration, count int) <-chan time.Time {
	ch := make(chan time.Time)

	if base <= 0 {
		base = 10 * time.Millisecond
	}

	if factor <= 0 {
		factor = 2.0
	}

	go func() {
		defer close(ch)

		d := base

		i := 0
		for count <= 0 || i < count {
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
				// proceed.
			}
			select {
			case <-ctx.Done():
				return
			case ch <- time.Now():
			}

			i++

			next := time.Duration(float64(d) * factor)
			if max > 0 && next > max {
				next = max
			}

			if next <= 0 {
				next = d
			}

			d = next
		}
	}()

	return ch
}
