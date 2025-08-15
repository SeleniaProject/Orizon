//go:build windows && !iocp
// +build windows,!iocp

package asyncio

// newIOCPIfAvailable returns nil when the 'iocp' build tag is not enabled.
// This keeps the factory linkable under default Windows builds.
func newIOCPIfAvailable() Poller { return nil }
