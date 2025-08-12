package channels

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// Channel is a thin, type-safe wrapper that provides convenience APIs
// around Go's native channels, including non-blocking ops and multi-recv select.
type Channel[T any] struct {
	ch     chan T
	closed atomic.Bool
}

// New creates a new channel with given capacity (0 for unbuffered).
func New[T any](capacity int) *Channel[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &Channel[T]{ch: make(chan T, capacity)}
}

// ErrClosed is returned when operating on a closed channel.
var ErrClosed = errors.New("channel: closed")

// Send sends v, blocking until delivered or ctx canceled. Returns ErrClosed if channel is closed.
func (c *Channel[T]) Send(ctx context.Context, v T) error {
	if c.closed.Load() {
		return ErrClosed
	}
	if ctx == nil {
		// Derive a cancelable context to avoid unbounded blocking when caller passes nil
		c, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = c
	}
	select {
	case c.ch <- v:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TrySend attempts to send without blocking. Returns false if full or closed.
func (c *Channel[T]) TrySend(v T) bool {
	if c.closed.Load() {
		return false
	}
	select {
	case c.ch <- v:
		return true
	default:
		return false
	}
}

// Recv receives one value, blocking until available or ctx canceled.
// ok is false if the channel is closed and drained. ErrClosed is not returned on normal close-drain.
func (c *Channel[T]) Recv(ctx context.Context) (val T, ok bool, err error) {
	if ctx == nil {
		c, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = c
	}
	select {
	case v, ok2 := <-c.ch:
		return v, ok2, nil
	case <-ctx.Done():
		var zero T
		return zero, false, ctx.Err()
	}
}

// TryRecv attempts to receive without blocking. ok=false if empty or closed and empty.
func (c *Channel[T]) TryRecv() (val T, ok bool) {
	select {
	case v, ok2 := <-c.ch:
		return v, ok2
	default:
		var zero T
		return zero, false
	}
}

// Close closes the channel for sending. Multiple calls are safe.
func (c *Channel[T]) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.ch)
	}
}

// Len returns the number of elements queued in the buffer.
func (c *Channel[T]) Len() int { return len(c.ch) }

// Cap returns the channel capacity.
func (c *Channel[T]) Cap() int { return cap(c.ch) }

// SelectRecv waits for any of the provided channels to receive a value.
// It returns the value, the index of the channel that received, ok, and error.
// ok=false indicates that specific channel is closed and yielded zero value.
func SelectRecv[T any](ctx context.Context, chans ...*Channel[T]) (T, int, bool, error) {
	var zero T
	if len(chans) == 0 {
		return zero, -1, false, errors.New("channel: no channels")
	}
	if ctx == nil {
		c, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = c
	}
	// Simple round-robin polling with short backoff to avoid reflection-based selects.
	// This is adequate for runtime tests and avoids unsafe/reflect usage.
	backoff := time.Microsecond * 50
	for {
		for i, ch := range chans {
			select {
			case v, ok := <-ch.ch:
				return v, i, ok, nil
			default:
			}
		}
		select {
		case <-ctx.Done():
			return zero, -1, false, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < time.Millisecond*5 {
			backoff *= 2
		}
	}
}
