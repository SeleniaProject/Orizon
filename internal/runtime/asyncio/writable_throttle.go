package asyncio

import (
	"os"
	"strconv"
	"time"
)

// getWritableInterval returns the throttling interval for Writable notifications.
// It reads ORIZON_WIN_WRITABLE_INTERVAL_MS (integer milliseconds) on every call.
// Defaults to 50ms. Clamped to [5ms, 5000ms] to avoid CPU spin or excessive delay.
// Note: Reading the env each time keeps tests deterministic when they set env per test.
func getWritableInterval() time.Duration {
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
	return time.Duration(ms) * time.Millisecond
}
