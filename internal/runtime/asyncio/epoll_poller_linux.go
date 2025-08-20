//go:build linux.
// +build linux.

package asyncio

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// epollPoller implements Poller using epoll(7).
type epollPoller struct {
	ctx    context.Context
	cancel context.CancelFunc
	epfd   int
	mu     sync.RWMutex
	regs   map[int]*epReg // fd -> registration
}

type epReg struct {
	fd                   int
	conn                 net.Conn
	kinds                []EventType
	handler              Handler
	lastWritableUnixNano int64
}

func newEpollPoller() Poller { return &epollPoller{regs: make(map[int]*epReg)} }

func (p *epollPoller) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}
	p.epfd = fd
	go p.loop()
	return nil
}

func (p *epollPoller) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Lock()
	regs := p.regs
	p.regs = make(map[int]*epReg)
	p.mu.Unlock()
	for fd := range regs {
		_ = unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
	}
	if p.epfd > 0 {
		_ = unix.Close(p.epfd)
		p.epfd = -1
	}
	return nil
}

func (p *epollPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
	if conn == nil || h == nil {
		return errors.New("invalid registration")
	}
	fd, err := getFD(conn)
	if err != nil {
		return err
	}
	ev := unix.EpollEvent{Fd: int32(fd)}
	for _, k := range kinds {
		switch k {
		case Readable:
			ev.Events |= unix.EPOLLIN
		case Writable:
			ev.Events |= unix.EPOLLOUT
		}
	}
	if err := unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &ev); err != nil {
		return err
	}
	p.mu.Lock()
	p.regs[fd] = &epReg{fd: fd, conn: conn, kinds: kinds, handler: h}
	p.mu.Unlock()
	return nil
}

func (p *epollPoller) Deregister(conn net.Conn) error {
	fd, err := getFD(conn)
	if err != nil {
		return err
	}
	_ = unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
	p.mu.Lock()
	delete(p.regs, fd)
	p.mu.Unlock()
	return nil
}

func (p *epollPoller) loop() {
	events := make([]unix.EpollEvent, 64)
	for {
		n, err := unix.EpollWait(p.epfd, events, -1)
		if err != nil {
			return
		}
		if n <= 0 {
			continue
		}
		p.mu.RLock()
		for i := 0; i < n; i++ {
			ev := events[i]
			reg := p.regs[int(ev.Fd)]
			if reg == nil {
				continue
			}
			if ev.Events&(unix.EPOLLERR|unix.EPOLLHUP) != 0 {
				reg.handler(Event{Conn: reg.conn, Type: Error})
				continue
			}
			if (ev.Events&unix.EPOLLIN) != 0 && contains(reg.kinds, Readable) {
				reg.handler(Event{Conn: reg.conn, Type: Readable})
			}
			if (ev.Events&unix.EPOLLOUT) != 0 && contains(reg.kinds, Writable) {
				now := time.Now()
				last := atomic.LoadInt64(&reg.lastWritableUnixNano)
				if last == 0 || now.Sub(time.Unix(0, last)) >= 50*time.Millisecond {
					reg.handler(Event{Conn: reg.conn, Type: Writable})
					atomic.StoreInt64(&reg.lastWritableUnixNano, now.UnixNano())
				}
			}
		}
		p.mu.RUnlock()
	}
}

func getFD(conn net.Conn) (int, error) {
	type sc interface {
		SyscallConn() (syscall.RawConn, error)
	}
	scc, ok := conn.(sc)
	if !ok {
		return -1, errors.New("conn does not expose SyscallConn")
	}
	rc, err := scc.SyscallConn()
	if err != nil {
		return -1, err
	}
	var fd int
	var ctrlErr error
	if e := rc.Control(func(rawfd uintptr) { fd = int(rawfd) }); e != nil {
		ctrlErr = e
	}
	return fd, ctrlErr
}

// NewOSPoller (linux) returns epoll-based poller.
func NewOSPoller() Poller { return newEpollPoller() }
