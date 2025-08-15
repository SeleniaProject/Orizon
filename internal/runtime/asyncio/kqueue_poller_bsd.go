//go:build darwin || freebsd || netbsd || openbsd
// +build darwin freebsd netbsd openbsd

package asyncio

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type kqueuePoller struct {
	ctx    context.Context
	cancel context.CancelFunc
	kq     int
	mu     sync.RWMutex
	regs   map[int]*kqReg
}

type kqReg struct {
	fd             int
	conn           net.Conn
	kinds          []EventType
	handler        Handler
	lastWritableUnixNano int64
}

func newKqueuePoller() Poller { return &kqueuePoller{regs: make(map[int]*kqReg)} }

func (p *kqueuePoller) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	fd, err := unix.Kqueue()
	if err != nil {
		return err
	}
	p.kq = fd
	go p.loop()
	return nil
}

func (p *kqueuePoller) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Lock()
	regs := p.regs
	p.regs = make(map[int]*kqReg)
	p.mu.Unlock()
	// No explicit EV_DELETE here for all; closing kq is enough, but try to clean
	for fd := range regs {
		delRead := unix.Kevent_t{Ident: uint64(fd), Filter: unix.EVFILT_READ, Flags: unix.EV_DELETE}
		delWrite := unix.Kevent_t{Ident: uint64(fd), Filter: unix.EVFILT_WRITE, Flags: unix.EV_DELETE}
		_, _ = unix.Kevent(p.kq, []unix.Kevent_t{delRead, delWrite}, nil, nil)
	}
	if p.kq > 0 {
		_ = unix.Close(p.kq)
		p.kq = -1
	}
	return nil
}

func (p *kqueuePoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
	if conn == nil || h == nil {
		return errors.New("invalid registration")
	}
	fd, err := getFD(conn)
	if err != nil {
		return err
	}
	var changes []unix.Kevent_t
	for _, k := range kinds {
		switch k {
		case Readable:
			changes = append(changes, unix.Kevent_t{Ident: uint64(fd), Filter: unix.EVFILT_READ, Flags: unix.EV_ADD | unix.EV_ENABLE})
		case Writable:
			changes = append(changes, unix.Kevent_t{Ident: uint64(fd), Filter: unix.EVFILT_WRITE, Flags: unix.EV_ADD | unix.EV_ENABLE})
		}
	}
	if len(changes) == 0 {
		return errors.New("no events requested")
	}
	if _, err := unix.Kevent(p.kq, changes, nil, nil); err != nil {
		return err
	}
	p.mu.Lock()
	p.regs[fd] = &kqReg{fd: fd, conn: conn, kinds: kinds, handler: h}
	p.mu.Unlock()
	return nil
}

func (p *kqueuePoller) Deregister(conn net.Conn) error {
	fd, err := getFD(conn)
	if err != nil {
		return err
	}
	delRead := unix.Kevent_t{Ident: uint64(fd), Filter: unix.EVFILT_READ, Flags: unix.EV_DELETE}
	delWrite := unix.Kevent_t{Ident: uint64(fd), Filter: unix.EVFILT_WRITE, Flags: unix.EV_DELETE}
	_, _ = unix.Kevent(p.kq, []unix.Kevent_t{delRead, delWrite}, nil, nil)
	p.mu.Lock()
	delete(p.regs, fd)
	p.mu.Unlock()
	return nil
}

func (p *kqueuePoller) loop() {
	events := make([]unix.Kevent_t, 64)
	for {
		n, err := unix.Kevent(p.kq, nil, events, nil)
		if err != nil {
			return
		}
		if n <= 0 {
			continue
		}
		p.mu.RLock()
		for i := 0; i < n; i++ {
			ev := events[i]
			reg := p.regs[int(ev.Ident)]
			if reg == nil {
				continue
			}
			// Error conditions are indicated by EV_ERROR flag
			if (ev.Flags & unix.EV_ERROR) != 0 {
				reg.handler(Event{Conn: reg.conn, Type: Error})
				continue
			}
			if ev.Filter == unix.EVFILT_READ && contains(reg.kinds, Readable) {
				reg.handler(Event{Conn: reg.conn, Type: Readable})
			}
			if ev.Filter == unix.EVFILT_WRITE && contains(reg.kinds, Writable) {
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

// NewOSPoller (BSD/macOS)
func NewOSPoller() Poller { return newKqueuePoller() }
