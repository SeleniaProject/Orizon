//go:build linux
// +build linux

package asyncio

import (
    "context"
    "errors"
    "net"
)

// epollPoller is a placeholder that currently falls back to goroutine poller.
// It exists to establish build tags and wiring for future native epoll usage.
type epollPoller struct{ Poller }

func newEpollPoller() Poller { return &epollPoller{Poller: NewDefaultPoller()} }

func (p *epollPoller) Start(ctx context.Context) error { return p.Poller.Start(ctx) }
func (p *epollPoller) Stop() error                      { return p.Poller.Stop() }
func (p *epollPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
    if conn == nil || h == nil { return errors.New("invalid registration") }
    return p.Poller.Register(conn, kinds, h)
}
func (p *epollPoller) Deregister(conn net.Conn) error { return p.Poller.Deregister(conn) }

// NewOSPoller (linux) returns epoll-based poller (currently delegates to default)
func NewOSPoller() Poller { return newEpollPoller() }


