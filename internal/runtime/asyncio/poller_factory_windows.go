//go:build windows.
// +build windows.

package asyncio

import (
	"os"
)

// NewOSPoller (Windows).
// Selection priority:.
// 1) IOCP if explicitly requested and compiled in (ORIZON_WIN_IOCP=1).
// 2) WSAPoll if requested (ORIZON_WIN_WSAPOLL=1).
// 3) Portable goroutine-based poller (default).
func NewOSPoller() Poller {
	// Prefer IOCP when explicitly requested and available (requires 'iocp' build tag).
	if v := os.Getenv("ORIZON_WIN_IOCP"); v == "1" || v == "true" || v == "on" {
		if p := newIOCPIfAvailable(); p != nil {
			return p
		}
		// If IOCP was explicitly requested but is not available, prefer WSAPoll over portable.
		return newWindowsPoller()
	}
	// Allow explicitly forcing WSAPoll via env for clarity (kept for backward compatibility).
	if v := os.Getenv("ORIZON_WIN_WSAPOLL"); v == "1" || v == "true" || v == "on" {
		return newWindowsPoller()
	}
	// Provide a portable fallback when requested (goroutine-based detection).
	if v := os.Getenv("ORIZON_WIN_PORTABLE"); v == "1" || v == "true" || v == "on" {
		return NewDefaultPoller()
	}
	// Default on Windows: use portability-first goroutine-based poller for broad compatibility.
	// WSAPoll and IOCP can be explicitly enabled via environment flags above.
	return NewDefaultPoller()
}
