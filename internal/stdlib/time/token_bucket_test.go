package timex

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketAllow(t *testing.T) {
	tb := NewTokenBucket(2, 10) // 10 tokens/sec
	if !tb.Allow(1) {
		t.Fatal("expected allow 1")
	}
	if !tb.Allow(1) {
		t.Fatal("expected allow 1 again")
	}
	if tb.Allow(1) {
		t.Fatal("should not allow third immediately")
	}
	// wait a bit to refill
	time.Sleep(120 * time.Millisecond)
	if !tb.Allow(1) {
		t.Fatal("expected allow after refill")
	}
}

func TestTokenBucketWait(t *testing.T) {
	tb := NewTokenBucket(1, 5) // 5/sec -> 200ms per token
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !tb.Allow(1) {
		t.Fatal("expected first allow")
	}

	start := time.Now()
	if err := tb.Wait(ctx, 1); err != nil {
		t.Fatalf("wait err: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 150*time.Millisecond {
		t.Fatalf("waited too little: %v", elapsed)
	}
}

func TestTokenBucketCancel(t *testing.T) {
	tb := NewTokenBucket(0, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := tb.Wait(ctx, 1); err == nil {
		t.Fatal("expected context deadline exceeded")
	}
}
