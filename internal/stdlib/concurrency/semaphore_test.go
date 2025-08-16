package concurrency

import (
	"context"
	"testing"
	"time"
)

func TestSemaphoreBasic(t *testing.T) {
	s := NewWeightedSemaphore(3)
	if !s.TryAcquire(2) {
		t.Fatal("try acquire 2")
	}
	if s.TryAcquire(2) {
		t.Fatal("should fail try acquire")
	}
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := s.Acquire(ctx, 2); err != nil {
			t.Errorf("acquire err: %v", err)
		}
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	s.Release(2)
	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for acquire")
	}
}

func TestSemaphoreCancel(t *testing.T) {
	s := NewWeightedSemaphore(1)
	if err := s.Acquire(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error)
	go func() { done <- s.Acquire(ctx, 1) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected cancel")
	}
}

func TestSemaphoreFIFO(t *testing.T) {
	s := NewWeightedSemaphore(3)
	if err := s.Acquire(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	// enqueue A(2) then B(1)
	a := make(chan struct{})
	b := make(chan struct{})
	go func() { s.Acquire(context.Background(), 2); close(a) }()
	// wait until A is enqueued
	for i := 0; i < 100; i++ {
		s.mu.Lock()
		qlen := len(s.q)
		s.mu.Unlock()
		if qlen >= 1 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	go func() { s.Acquire(context.Background(), 1); close(b) }()
	// wait until B is enqueued too
	for i := 0; i < 100; i++ {
		s.mu.Lock()
		qlen := len(s.q)
		s.mu.Unlock()
		if qlen >= 2 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	s.Release(1)
	select {
	case <-a:
		// A should not pass yet (needs 2)
		t.Fatal("A should wait")
	case <-b:
		t.Fatal("B must wait for A (FIFO)")
	case <-time.After(50 * time.Millisecond):
	}
	s.Release(2)
	select {
	case <-a:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("A not released")
	}
	select {
	case <-b:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("B not released")
	}
}
