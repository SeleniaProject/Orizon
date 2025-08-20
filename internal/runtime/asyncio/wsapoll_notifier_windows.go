//go:build windows.
// +build windows.

package asyncio

// Draft wsapollNotifier implementing winNotifier.
// Currently no-op; wiring will be done in subsequent refactors.

type wsapollNotifier struct{}

func (wsapollNotifier) armReadable(r *winRegLite) {}
func (wsapollNotifier) armWritable(r *winRegLite) {}
func (wsapollNotifier) cancel(r *winRegLite)      {}
