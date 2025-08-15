//go:build windows
// +build windows

package asyncio

// Draft iocpNotifier implementing winNotifier.
// Currently no-op; real implementation will arm zero-byte WSARecv/WSASend and CancelIoEx.

type iocpNotifier struct{}

func (iocpNotifier) armReadable(r *winRegLite) {}
func (iocpNotifier) armWritable(r *winRegLite) {}
func (iocpNotifier) cancel(r *winRegLite)      {}
