//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !windows
// +build !linux,!darwin,!freebsd,!netbsd,!openbsd,!windows

package asyncio

// NewOSPoller returns an OS-optimized Poller when available.
// This default implementation is used on platforms without a specialized poller.
func NewOSPoller() Poller { return NewDefaultPoller() }
