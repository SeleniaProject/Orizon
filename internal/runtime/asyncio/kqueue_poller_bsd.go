//go:build darwin || freebsd || netbsd || openbsd
// +build darwin freebsd netbsd openbsd

package asyncio

import (
    "context"
    "errors"
    "net"
)

// kqueuePoller placeholder delegating to default poller.
type kqueuePoller struct{ Poller }

func newKqueuePoller() Poller { return &kqueuePoller{Poller: NewDefaultPoller()} }

func (p *kqueuePoller) Start(ctx context.Context) error { return p.Poller.Start(ctx) }
func (p *kqueuePoller) Stop() error                      { return p.Poller.Stop() }
func (p *kqueuePoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
    if conn == nil || h == nil { return errors.New("invalid registration") }
    return p.Poller.Register(conn, kinds, h)
}
func (p *kqueuePoller) Deregister(conn net.Conn) error { return p.Poller.Deregister(conn) }

// NewOSPoller (BSD/macOS) returns kqueue-based poller (currently delegates to default)
func NewOSPoller() Poller { return newKqueuePoller() }


