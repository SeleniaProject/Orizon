package asyncio

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// getWritableInterval returns the throttling interval for Writable notifications.
// It reads ORIZON_WIN_WRITABLE_INTERVAL_MS (integer milliseconds).
// Defaults to 50ms. Clamped to [5ms, 5000ms] to avoid CPU spin or excessive delay.
// This function is safe on all platforms; Windows pollers and the portable poller
// share the same configuration for consistent behavior.
var (
	writableOnce sync.Once
	writableIntv time.Duration
)

func getWritableInterval() time.Duration {
	writableOnce.Do(func() {
		const (
			defMs = 50
			minMs = 5
			maxMs = 5000
		)
		ms := defMs
		if v := os.Getenv("ORIZON_WIN_WRITABLE_INTERVAL_MS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				if n < minMs {
					n = minMs
				} else if n > maxMs {
					n = maxMs
				}
				ms = n
			}
		}
		writableIntv = time.Duration(ms) * time.Millisecond
	})
	return writableIntv
}
