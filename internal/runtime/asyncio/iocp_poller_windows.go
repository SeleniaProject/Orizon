//go:build windows
// +build windows

package asyncio

import (
    "context"
    "errors"
    "net"
)

// iocpPoller placeholder delegating to default poller.
type iocpPoller struct{ Poller }

func newIOCPPoller() Poller { return &iocpPoller{Poller: NewDefaultPoller()} }

func (p *iocpPoller) Start(ctx context.Context) error { return p.Poller.Start(ctx) }
func (p *iocpPoller) Stop() error                     { return p.Poller.Stop() }
func (p *iocpPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
    if conn == nil || h == nil { return errors.New("invalid registration") }
    return p.Poller.Register(conn, kinds, h)
}
func (p *iocpPoller) Deregister(conn net.Conn) error { return p.Poller.Deregister(conn) }

// NewOSPoller (Windows) returns IOCP-based poller (currently delegates to default)
func NewOSPoller() Poller { return newIOCPPoller() }


