//go:build windows.
// +build windows.

package asyncio

// This file sketches an internal notifier abstraction for Windows pollers.
// It enables unifying WSAPoll and IOCP strategies behind a single surface.
// Implementation will be introduced incrementally to avoid regressions.

// winNotifier abstracts Windows-specific readiness arming/cancel.
// It operates on a minimal registration carrier to avoid coupling.
// NOTE: This is an internal draft and not yet wired into windowsPoller.

type winNotifier interface {
	armReadable(r *winRegLite)
	armWritable(r *winRegLite)
	cancel(r *winRegLite)
}

// winRegLite carries the minimal fields a notifier needs.
// It intentionally mirrors a subset of fields from windows-specific regs.
// without importing full structs to keep the boundary narrow.

type winRegLite struct {
	sock     uintptr // Windows SOCKET as uintptr to avoid imports
	disabled *uint32 // minimal disabled flag placeholder
}

// iocpNotifier/wsapollNotifier will implement winNotifier in follow-up changes.
