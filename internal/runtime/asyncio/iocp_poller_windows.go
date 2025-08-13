//go:build windows
// +build windows

package asyncio

import (
	"context"
	"errors"
	"net"
)

// windowsPoller currently delegates to the portable goroutine-based poller.
// Rationale: Avoid interfering with Go's internal Windows netpoller (IOCP).
// A specialized implementation (e.g., IOCP zero-byte WSARecv) can be added later
// behind the same interface once compatibility considerations are fully vetted.
type windowsPoller struct{ Poller }

func newWindowsPoller() Poller { return &windowsPoller{Poller: NewDefaultPoller()} }

func (p *windowsPoller) Start(ctx context.Context) error { return p.Poller.Start(ctx) }
func (p *windowsPoller) Stop() error                     { return p.Poller.Stop() }
func (p *windowsPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
	if conn == nil || h == nil {
		return errors.New("invalid registration")
	}
	return p.Poller.Register(conn, kinds, h)
}
func (p *windowsPoller) Deregister(conn net.Conn) error { return p.Poller.Deregister(conn) }

// NewOSPoller (Windows)
func NewOSPoller() Poller { return newWindowsPoller() }
