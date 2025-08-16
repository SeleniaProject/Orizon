package concurrency

import (
	"context"
	"errors"
	"sync"
)

var ErrTooLarge = errors.New("semaphore: weight exceeds capacity")

// WeightedSemaphore is a fair, weighted semaphore with FIFO acquisition.
// Acquire is context-aware; TryAcquire is non-blocking; Release frees weight.
// Implementation uses a request queue managed without a background goroutine.
type WeightedSemaphore struct {
	cap int64
	mu  sync.Mutex
	cur int64
	q   []*semReq
}

type semReq struct {
	w  int64
	ch chan struct{}
	cx bool // canceled
}

func NewWeightedSemaphore(capacity int) *WeightedSemaphore {
	if capacity < 0 {
		capacity = 0
	}
	return &WeightedSemaphore{cap: int64(capacity)}
}

// TryAcquire attempts to take w tokens immediately.
func (s *WeightedSemaphore) TryAcquire(w int) bool {
	if w < 0 {
		return true
	}
	ww := int64(w)
	s.mu.Lock()
	defer s.mu.Unlock()
	if ww > s.cap {
		return false
	}
	if s.cur+ww <= s.cap && len(s.q) == 0 {
		s.cur += ww
		return true
	}
	return false
}

// Acquire blocks until w tokens can be taken or ctx is done.
func (s *WeightedSemaphore) Acquire(ctx context.Context, w int) error {
	if w < 0 {
		return nil
	}
	ww := int64(w)
	if ww > s.cap {
		return ErrTooLarge
	}

	// Fast path
	s.mu.Lock()
	if s.cur+ww <= s.cap && len(s.q) == 0 {
		s.cur += ww
		s.mu.Unlock()
		return nil
	}
	// Enqueue request
	r := &semReq{w: ww, ch: make(chan struct{})}
	s.q = append(s.q, r)
	// Try to satisfy queue now in case capacity is available
	s.processLocked()
	ch := r.ch
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		// Mark canceled and trigger processing
		s.mu.Lock()
		r.cx = true
		s.processLocked()
		s.mu.Unlock()
		return ctx.Err()
	case <-ch:
		return nil
	}
}

// Release frees w tokens and wakes queued acquirers if possible.
func (s *WeightedSemaphore) Release(w int) {
	if w <= 0 {
		return
	}
	ww := int64(w)
	s.mu.Lock()
	s.cur -= ww
	if s.cur < 0 {
		s.cur = 0
	}
	s.processLocked()
	s.mu.Unlock()
}

// processLocked tries to grant queued requests in FIFO order.
func (s *WeightedSemaphore) processLocked() {
	// Drop canceled requests at head
	i := 0
	for i < len(s.q) {
		if s.q[i].cx {
			i++
			continue
		}
		// If head cannot be satisfied, stop (fairness)
		if s.cur+s.q[i].w > s.cap {
			break
		}
		// Grant head
		s.cur += s.q[i].w
		close(s.q[i].ch)
		i++
	}
	if i > 0 {
		s.q = s.q[i:]
	}
	// Also compact canceled requests lingering at new head
	j := 0
	for _, r := range s.q {
		if r.cx {
			continue
		}
		s.q[j] = r
		j++
	}
	s.q = s.q[:j]
}
