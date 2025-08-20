package timex

import (
	"testing"
	"time"
)

func TestRetryBackoff(t *testing.T) {
	start := time.Now()
	ch := RetryBackoff(10*time.Millisecond, 2.0, 50*time.Millisecond, 4)
	count := 0

	for range ch {
		count++
	}

	if count != 4 {
		t.Fatalf("count=%d", count)
	}

	elapsed := time.Since(start)
	if elapsed < 10*time.Millisecond {
		t.Fatalf("too fast: %v", elapsed)
	}
}
