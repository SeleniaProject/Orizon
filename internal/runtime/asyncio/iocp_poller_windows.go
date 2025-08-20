//go:build windows.
// +build windows.

package asyncio

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Dynamically link WSAPoll to avoid relying on specific x/sys symbols.
var (
	ws2_32      = windows.NewLazySystemDLL("ws2_32.dll")
	procWSAPoll = ws2_32.NewProc("WSAPoll")
)

// WSAPoll constants from winsock2.h. POLLIN/OUT are macro combinations.
const (
	pollERR    = int16(0x0001)
	pollHUP    = int16(0x0002)
	pollNVAL   = int16(0x0004)
	pollWRNORM = int16(0x0010)
	pollWRBAND = int16(0x0020)
	pollRDNORM = int16(0x0100)
	pollRDBAND = int16(0x0200)
	pollPRI    = int16(0x0400)

	pollIN  = pollRDNORM | pollRDBAND
	pollOUT = pollWRNORM | pollWRBAND
)

// wsaPollFD mirrors WSAPOLLFD from winsock2.h
type wsaPollFD struct {
	Fd      uintptr
	Events  int16
	Revents int16
}

// windowsPoller implements Poller using WSAPoll without changing socket IOCP.
type windowsPoller struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.RWMutex
	regs   map[uintptr]*winReg // SOCKET -> registration
	byConn map[net.Conn]*winReg

	wakeRecv *net.UDPConn
	wakeSend *net.UDPConn

	closed atomic.Uint32

	wg sync.WaitGroup
	// notifier provides Windows-specific arming/cancel hooks (future use).
	notifier winNotifier
}

type winReg struct {
	sock                 uintptr
	conn                 net.Conn
	kinds                []EventType
	handler              Handler
	reader               *bufio.Reader
	disabled             atomic.Uint32
	lastWritableUnixNano int64
	stop                 context.CancelFunc
}

func newWindowsPoller() Poller {
	return &windowsPoller{
		regs:     make(map[uintptr]*winReg),
		byConn:   make(map[net.Conn]*winReg),
		notifier: wsapollNotifier{},
	}
}

func (p *windowsPoller) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	// Ensure notifier is initialized (no-op implementation by default).
	// This prepares future wiring without changing current behavior.
	if p.notifier == nil {
		p.notifier = wsapollNotifier{}
	}
	// Create UDP-based waker to interrupt WSAPoll waits on updates/Stop.
	if recv, send, err := newUDPWaker(); err == nil {
		p.wakeRecv = recv
		p.wakeSend = send
	}
	// Start central WSAPoll loop.
	p.wg.Add(1)
	go func() { defer p.wg.Done(); p.loop() }()
	return nil
}

func (p *windowsPoller) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.closed.Store(1)
	// Wake the poll loop so it can observe closed state and exit quickly.
	p.wake()
	p.mu.Lock()
	regs := p.regs
	p.regs = make(map[uintptr]*winReg)
	byConn := p.byConn
	p.byConn = make(map[net.Conn]*winReg)
	p.mu.Unlock()
	for _, r := range regs {
		r.disabled.Store(1)
		if r.stop != nil {
			r.stop()
		}
	}
	_ = byConn
	// Close waker endpoints after loop has a chance to drain.
	p.wg.Wait()
	if p.wakeRecv != nil {
		_ = p.wakeRecv.Close()
	}
	if p.wakeSend != nil {
		_ = p.wakeSend.Close()
	}
	return nil
}

func (p *windowsPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
	if conn == nil || h == nil {
		return errors.New("invalid registration")
	}
	s, err := getSocketFromConn(conn)
	if err != nil {
		return err
	}
	p.mu.Lock()
	if old, exists := p.regs[s]; exists {
		// Idempotent update: refresh handler and kinds; keep existing reader and control goroutines.
		old.kinds = kinds
		old.handler = h
		// refresh byConn mapping in case conn identity changed.
		p.byConn[conn] = old
		p.mu.Unlock()
		// No-op hooks for future unification.
		if p.notifier != nil {
			lite := &winRegLite{sock: s}
			if contains(kinds, Readable) {
				p.notifier.armReadable(lite)
			}
			if contains(kinds, Writable) {
				p.notifier.armWritable(lite)
			}
		}
		// Ensure periodic writable notifier is running if requested.
		if contains(kinds, Writable) {
			wctx, cancel := context.WithCancel(p.ctx)
			prev := old.stop
			old.stop = func() {
				if prev != nil {
					prev()
				}
				cancel()
			}
			p.wg.Add(1)
			go func(r *winReg) {
				defer p.wg.Done()
				t := time.NewTicker(getWritableInterval())
				defer t.Stop()
				for {
					select {
					case <-wctx.Done():
						return
					case <-t.C:
						if r.disabled.Load() != 0 {
							continue
						}
						now := time.Now()
						last := atomic.LoadInt64(&r.lastWritableUnixNano)
						if last == 0 || now.Sub(time.Unix(0, last)) >= getWritableInterval() {
							r.handler(Event{Conn: r.conn, Type: Writable})
							atomic.StoreInt64(&r.lastWritableUnixNano, now.UnixNano())
						}
					}
				}
			}(old)
		}
		// Notify poll loop of potential event mask change.
		p.wake()
		return nil
	}
	_, cancel := context.WithCancel(p.ctx)
	reg := &winReg{sock: s, conn: conn, kinds: kinds, handler: h, reader: bufio.NewReader(conn), stop: cancel}
	p.regs[s] = reg
	p.byConn[conn] = reg
	p.mu.Unlock()
	// No-op hooks for future unification.
	if p.notifier != nil {
		lite := &winRegLite{sock: s}
		if contains(kinds, Readable) {
			p.notifier.armReadable(lite)
		}
		if contains(kinds, Writable) {
			p.notifier.armWritable(lite)
		}
	}
	// If Writable is requested, start a periodic notifier to ensure progress even when WSAPoll doesn't signal OUT frequently.
	if contains(kinds, Writable) {
		wctx, cancel := context.WithCancel(p.ctx)
		// chain cancels: replacing stop to also cancel ticker goroutine.
		prev := reg.stop
		reg.stop = func() {
			if prev != nil {
				prev()
			}
			cancel()
		}
		p.wg.Add(1)
		go func(r *winReg) {
			defer p.wg.Done()
			t := time.NewTicker(getWritableInterval())
			defer t.Stop()
			for {
				select {
				case <-wctx.Done():
					return
				case <-t.C:
					if r.disabled.Load() != 0 {
						continue
					}
					now := time.Now()
					last := atomic.LoadInt64(&r.lastWritableUnixNano)
					if last == 0 || now.Sub(time.Unix(0, last)) >= getWritableInterval() {
						r.handler(Event{Conn: r.conn, Type: Writable})
						atomic.StoreInt64(&r.lastWritableUnixNano, now.UnixNano())
					}
				}
			}
		}(reg)
	}
	// Notify poll loop to rebuild FD set immediately.
	p.wake()
	return nil
}

func (p *windowsPoller) Deregister(conn net.Conn) error {
	s, err := getSocketFromConn(conn)
	p.mu.Lock()
	if err != nil {
		// Fallback: try lookup by connection identity (may occur after conn.Close())
		if reg, ok := p.byConn[conn]; ok {
			reg.disabled.Store(1)
			if reg.stop != nil {
				reg.stop()
			}
			delete(p.regs, reg.sock)
			delete(p.byConn, reg.conn)
			p.mu.Unlock()
			if p.notifier != nil {
				p.notifier.cancel(&winRegLite{sock: reg.sock})
			}
			p.wake()
			return nil
		}
		p.mu.Unlock()
		return nil
	}
	if reg, ok := p.regs[s]; ok {
		reg.disabled.Store(1)
		if reg.stop != nil {
			reg.stop()
		}
		delete(p.regs, s)
		delete(p.byConn, reg.conn)
		p.mu.Unlock()
		// Wake the poll loop so it can drop the FD quickly.
		if p.notifier != nil {
			p.notifier.cancel(&winRegLite{sock: s})
		}
		p.wake()
		return nil
	}
	// If not found by socket key, fall back to by-conn lookup to be robust.
	if reg, ok := p.byConn[conn]; ok {
		reg.disabled.Store(1)
		if reg.stop != nil {
			reg.stop()
		}
		delete(p.regs, reg.sock)
		delete(p.byConn, reg.conn)
		p.mu.Unlock()
		if p.notifier != nil {
			p.notifier.cancel(&winRegLite{sock: reg.sock})
		}
		p.wake()
		return nil
	}
	p.mu.Unlock()
	return nil
}

// watch replicates portability-first readiness detection with adaptive throttling.
// watch retained for compatibility if needed by other code paths; unused in WSAPoll mode.
func (p *windowsPoller) watch(ctx context.Context, reg *winReg) {
	interval := 5 * time.Millisecond
	idleCount := 0
	const (
		maxInterval   = 50 * time.Millisecond
		minInterval   = 1 * time.Millisecond
		growThreshold = 8
		shrinkFactor  = 2
	)
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			activity := false
			// Readable.
			if contains(reg.kinds, Readable) {
				_ = reg.conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
				if b, err := reg.reader.Peek(1); err == nil && len(b) > 0 {
					if reg.disabled.Load() == 0 {
						reg.handler(Event{Conn: reg.conn, Type: Readable})
					}
					activity = true
				} else if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						// ignore.
					} else if errors.Is(err, io.EOF) {
						if reg.disabled.Load() == 0 {
							reg.handler(Event{Conn: reg.conn, Type: Error, Err: io.EOF})
						}
						return
					} else if !errors.Is(err, io.EOF) {
						if reg.disabled.Load() == 0 {
							reg.handler(Event{Conn: reg.conn, Type: Error, Err: err})
						}
						return
					}
				}
			}
			// Writable (throttled).
			if contains(reg.kinds, Writable) {
				now := time.Now()
				last := atomic.LoadInt64(&reg.lastWritableUnixNano)
				if last == 0 || now.Sub(time.Unix(0, last)) >= getWritableInterval() {
					if reg.disabled.Load() == 0 {
						reg.handler(Event{Conn: reg.conn, Type: Writable})
					}
					atomic.StoreInt64(&reg.lastWritableUnixNano, now.UnixNano())
					activity = true
				}
			}
			// Adapt interval.
			if activity {
				idleCount = 0
				if interval > 5*time.Millisecond {
					interval = interval / shrinkFactor
					if interval < minInterval {
						interval = minInterval
					}
					tick.Reset(interval)
				}
			} else {
				idleCount++
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

// loop executes a central WSAPoll wait over all registered sockets plus a wake FD.
func (p *windowsPoller) loop() {
	// Backoff when no fds present to avoid spinning.
	idleDelay := 5 * time.Millisecond
	for p.closed.Load() == 0 {
		// Build FD set snapshot.
		fds, regs := p.snapshot()
		// Prepend wake FD if available.
		if wfd, _ := p.wakeFD(); wfd != nil {
			fds = append([]wsaPollFD{*wfd}, fds...)
		}
		if len(fds) == 0 {
			time.Sleep(idleDelay)
			continue
		}
		// Poll with reasonable timeout to react to Stop.
		n, err := wsaPoll(fds, 1000)
		if err != nil {
			// Sleep a bit on errors to avoid busy loop.
			time.Sleep(2 * time.Millisecond)
			continue
		}
		if n <= 0 {
			continue
		}
		// Handle wake fd if included.
		startIdx := 0
		if p.wakeRecv != nil {
			// Index 0 is wake.
			if fds[0].Revents != 0 {
				p.drainWake()
			}
			startIdx = 1
		}
		// Process events for regs aligned to fds[startIdx:].
		for i := startIdx; i < len(fds) && (i-startIdx) < len(regs); i++ {
			re := fds[i].Revents
			if re == 0 {
				continue
			}
			reg := regs[i-startIdx]
			if reg == nil {
				continue
			}
			if reg.disabled.Load() != 0 {
				continue
			}
			// Error conditions.
			if (re&pollERR) != 0 || (re&pollHUP) != 0 || (re&pollNVAL) != 0 {
				reg.handler(Event{Conn: reg.conn, Type: Error, Err: io.EOF})
				continue
			}
			// Readable.
			if (re&(pollIN|pollRDNORM|pollPRI|pollRDBAND)) != 0 && contains(reg.kinds, Readable) {
				if reg.disabled.Load() == 0 {
					reg.handler(Event{Conn: reg.conn, Type: Readable})
				}
			}
			// Writable (throttled).
			if (re&(pollOUT|pollWRNORM|pollWRBAND)) != 0 && contains(reg.kinds, Writable) {
				now := time.Now()
				last := atomic.LoadInt64(&reg.lastWritableUnixNano)
				if last == 0 || now.Sub(time.Unix(0, last)) >= getWritableInterval() {
					if reg.disabled.Load() == 0 {
						reg.handler(Event{Conn: reg.conn, Type: Writable})
					}
					atomic.StoreInt64(&reg.lastWritableUnixNano, now.UnixNano())
				}
			}
		}
	}
}

func (p *windowsPoller) snapshot() ([]wsaPollFD, []*winReg) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.regs) == 0 {
		return nil, nil
	}
	fds := make([]wsaPollFD, 0, len(p.regs))
	regs := make([]*winReg, 0, len(p.regs))
	for _, r := range p.regs {
		var ev int16
		for _, k := range r.kinds {
			switch k {
			case Readable:
				ev |= pollIN | pollRDNORM
			case Writable:
				ev |= pollOUT | pollWRNORM
			case Error:
				// implicit via revents.
			}
		}
		fds = append(fds, wsaPollFD{Fd: r.sock, Events: ev})
		regs = append(regs, r)
	}
	return fds, regs
}

func (p *windowsPoller) wakeFD() (*wsaPollFD, *winReg) {
	if p.wakeRecv == nil {
		return nil, nil
	}
	s, err := getSocketFromConn(p.wakeRecv)
	if err != nil {
		return nil, nil
	}
	fd := wsaPollFD{Fd: s, Events: pollIN | pollRDNORM}
	return &fd, nil
}

func (p *windowsPoller) wake() {
	if p.wakeSend == nil {
		return
	}
	_, _ = p.wakeSend.Write([]byte{0xFF})
}

func (p *windowsPoller) drainWake() {
	if p.wakeRecv == nil {
		return
	}
	_ = p.wakeRecv.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	buf := make([]byte, 8)
	for {
		if _, _, err := p.wakeRecv.ReadFromUDP(buf); err != nil {
			break
		}
		if p.closed.Load() == 1 {
			break
		}
	}
}

func newUDPWaker() (*net.UDPConn, *net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	recv, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, nil, err
	}
	send, err := net.DialUDP("udp4", nil, recv.LocalAddr().(*net.UDPAddr))
	if err != nil {
		_ = recv.Close()
		return nil, nil, err
	}
	return recv, send, nil
}

func getSocketFromConn(conn net.Conn) (uintptr, error) {
	type sc interface {
		SyscallConn() (syscall.RawConn, error)
	}
	scc, ok := conn.(sc)
	if !ok {
		return 0, errors.New("conn does not expose SyscallConn")
	}
	rc, err := scc.SyscallConn()
	if err != nil {
		return 0, err
	}
	var s uintptr
	var ctrlErr error
	if e := rc.Control(func(fd uintptr) { s = fd }); e != nil {
		ctrlErr = e
	}
	return s, ctrlErr
}

func wsaPoll(fds []wsaPollFD, timeoutMs int) (int, error) {
	if len(fds) == 0 {
		return 0, nil
	}
	r1, _, e1 := procWSAPoll.Call(
		uintptr(unsafe.Pointer(&fds[0])),
		uintptr(uint32(len(fds))),
		uintptr(int32(timeoutMs)),
	)
	n := int(int32(r1))
	if n == -1 {
		return -1, e1
	}
	return n, nil
}

// Note: NewOSPoller is defined in poller_factory_windows.go for Windows.
