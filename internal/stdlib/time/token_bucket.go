package timex

import (
	"context"
	"sync"
	"time"
)

// TokenBucket is a simple token-bucket rate limiter.
// - capacity: maximum number of tokens in the bucket.
// - rate: tokens refilled per second.
// Refill happens lazily on each operation based on elapsed time.
type TokenBucket struct {
	last     time.Time
	capacity float64
	rate     float64
	tokens   float64
	mu       sync.Mutex
}

// NewTokenBucket creates a new TokenBucket with given capacity and fill rate per second.
// If capacity<=0 or rate<=0, sensible defaults are applied.
func NewTokenBucket(capacity int, ratePerSec float64) *TokenBucket {
	if capacity < 0 {
		capacity = 0
	}

	if ratePerSec <= 0 {
		ratePerSec = 1
	}

	return &TokenBucket{
		capacity: float64(capacity),
		rate:     ratePerSec,
		tokens:   float64(capacity), // start full (zero when capacity==0)
		last:     time.Now(),
	}
}

// refill adds tokens according to elapsed time since last.
func (tb *TokenBucket) refill(now time.Time) {
	if tb.capacity <= 0 {
		tb.last = now

		return
	}

	if tb.tokens >= tb.capacity {
		tb.last = now

		return
	}

	elapsed := now.Sub(tb.last).Seconds()
	if elapsed <= 0 {
		return
	}

	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.last = now
}

// Allow attempts to take n tokens immediately.
// Returns true if granted, false otherwise.
func (tb *TokenBucket) Allow(n int) bool {
	if n <= 0 {
		return true
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill(time.Now())

	need := float64(n)
	if tb.tokens+1e-9 >= need {
		tb.tokens -= need

		return true
	}

	return false
}

// Wait blocks until n tokens are available or ctx is done.
func (tb *TokenBucket) Wait(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	need := float64(n)

	for {
		// Fast path under lock.
		tb.mu.Lock()
		now := time.Now()
		tb.refill(now)

		if tb.tokens+1e-9 >= need {
			tb.tokens -= need
			tb.mu.Unlock()

			return nil
		}
		// compute time to wait for enough tokens.
		missing := need - tb.tokens

		var wait time.Duration

		if tb.rate > 0 && tb.capacity > 0 {
			sec := missing / tb.rate

			wait = time.Duration(sec * float64(time.Second))
			if wait < time.Millisecond {
				wait = time.Millisecond
			}
		} else {
			// No capacity or no rate: cannot ever satisfy; rely on context cancel.
			tb.mu.Unlock()
			<-ctx.Done()

			return ctx.Err()
		}
		tb.mu.Unlock()

		// sleep or context cancel.
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()

			return ctx.Err()
		case <-timer.C:
			// loop.
		}
	}
}

// Available returns the current integer number of tokens available (floored).
func (tb *TokenBucket) Available() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill(time.Now())

	if tb.tokens <= 0 {
		return 0
	}

	return int(tb.tokens)
}
