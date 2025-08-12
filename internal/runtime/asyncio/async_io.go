package asyncio

import (
	"bufio"
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// EventType represents readiness kinds.
type EventType int

const (
	Readable EventType = iota
	Writable
	Error
)

// Event describes an I/O readiness notification.
type Event struct {
	Conn  net.Conn
	Type  EventType
	Err   error
	Extra any
}

// Handler is invoked on I/O readiness.
type Handler func(ev Event)

// Poller abstracts platform-specific pollers (epoll/kqueue/IOCP). The default
// implementation uses goroutines and does not depend on OS-specific syscalls.
type Poller interface {
	Start(ctx context.Context) error
	Stop() error
	Register(conn net.Conn, kinds []EventType, h Handler) error
	Deregister(conn net.Conn) error
}

// goPoller is a goroutine-driven poller that spawns per-connection loops to
// detect readiness by non-blocking operations with deadlines. This is a
// portability-first baseline; OS-specific pollers can implement Poller too.
type goPoller struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	conns  map[net.Conn]*registration
}

type registration struct {
	kinds   []EventType
	handler Handler
	stop    context.CancelFunc
}

// NewDefaultPoller returns a goroutine-based poller.
func NewDefaultPoller() Poller { return &goPoller{conns: make(map[net.Conn]*registration)} }

func (p *goPoller) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	return nil
}

func (p *goPoller) Stop() error {
	p.mu.Lock()
	for _, r := range p.conns {
		if r.stop != nil {
			r.stop()
		}
	}
	p.conns = make(map[net.Conn]*registration)
	p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *goPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
	if conn == nil || h == nil {
		return errors.New("invalid registration")
	}
	p.mu.Lock()
	if _, exists := p.conns[conn]; exists {
		p.mu.Unlock()
		return errors.New("already registered")
	}
	ctx, cancel := context.WithCancel(p.ctx)
	reg := &registration{kinds: kinds, handler: h, stop: cancel}
	p.conns[conn] = reg
	p.mu.Unlock()

	// spawn watcher
	go p.watch(ctx, conn, reg)
	return nil
}

func (p *goPoller) Deregister(conn net.Conn) error {
	p.mu.Lock()
	if reg, ok := p.conns[conn]; ok {
		if reg.stop != nil {
			reg.stop()
		}
		delete(p.conns, conn)
	}
	p.mu.Unlock()
	return nil
}

func (p *goPoller) watch(ctx context.Context, conn net.Conn, reg *registration) {
	// Use small peek attempts for readability detection
	reader := bufio.NewReader(conn)
	// Adaptive polling interval to reduce CPU under load. Starts at 5ms and
	// increases up to 50ms when repeated idle polls are observed, and shrinks
	// back when activity is detected.
	interval := 5 * time.Millisecond
	idleCount := 0
	var activityCount uint64
	var idleCycleCount uint64
	const (
		maxInterval   = 50 * time.Millisecond
		minInterval   = 1 * time.Millisecond
		growThreshold = 8 // grow after several consecutive idle cycles
		shrinkFactor  = 2 // shrink interval by this factor on activity
	)
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			activity := false
			// Readable
			if contains(reg.kinds, Readable) {
				_ = conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
				if b, err := reader.Peek(1); err == nil && len(b) > 0 {
					reg.handler(Event{Conn: conn, Type: Readable})
					activity = true
				}
			}
			// Writable: assume always writable for TCP once connected
			if contains(reg.kinds, Writable) {
				reg.handler(Event{Conn: conn, Type: Writable})
				activity = true
			}
			// Adapt interval based on activity
			if activity {
				idleCount = 0
				atomic.AddUint64(&activityCount, 1)
				if interval > 5*time.Millisecond {
					interval = interval / shrinkFactor
					if interval < minInterval {
						interval = minInterval
					}
					tick.Reset(interval)
				}
			} else {
				idleCount++
				atomic.AddUint64(&idleCycleCount, 1)
				if idleCount >= growThreshold && interval < maxInterval {
					idleCount = 0
					interval = interval * 2
					if interval > maxInterval {
						interval = maxInterval
					}
					tick.Reset(interval)
				}
			}
		}
	}
}

// Note: Counters are local to the watcher goroutine and not exported here.
// For production metrics, integrate with the runtime metrics registry.

func contains(vs []EventType, e EventType) bool {
	for _, v := range vs {
		if v == e {
			return true
		}
	}
	return false
}
