package timex

import (
	"context"
	"testing"
	"time"
)

func TestRetryBackoffContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := RetryBackoffContext(ctx, 5*time.Millisecond, 2.0, 20*time.Millisecond, 0)
	// 2 ticks then cancel
	count := 0
	for v := range ch {
		_ = v
		count++
		if count == 2 {
			cancel()
		}
		if count > 5 {
			t.Fatal("cancel not respected")
		}
	}
	if count < 2 {
		t.Fatalf("too few ticks: %d", count)
	}
}
