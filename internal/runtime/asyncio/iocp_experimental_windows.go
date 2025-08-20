//go:build windows && iocp.
// +build windows,iocp.

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

func isTimeout(err error) bool {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return false
}

// Experimental: IOCP-backed poller that requires exclusive association with an IOCP.
// NOTE: Associating Go's net.Conn sockets (already managed by Go runtime netpoller) with a custom IOCP
// can conflict. Use only with sockets created/owned by this runtime.

type iocpPoller struct {
	ctx    context.Context
	cancel context.CancelFunc

	port windows.Handle

	mu   sync.RWMutex
	regs map[uintptr]*iocpReg

	// keep overlapped structures alive until completion.
	pendingMu sync.Mutex
	pending   map[*overlappedOp]struct{}

	closed atomic.Uint32
	wg     sync.WaitGroup
}

type iocpReg struct {
	sock     windows.Handle
	conn     net.Conn
	kinds    []EventType
	handler  Handler
	disabled atomic.Uint32

	// writable ticker (optional).
	stopW  context.CancelFunc
	closed chan struct{}

	// keep a reference to the in-flight overlapped for zero-byte recv to avoid GC.
	pending *overlappedOp

	// keep reference to the in-flight zero-byte send for writable probes.
	sendPending *overlappedOp

	// reader used to non-destructively probe readability and EOF via Peek.
	reader *bufio.Reader

	// track outstanding send probe to avoid flooding.
	sendInFlight atomic.Uint32

	// fallback watcher when IOCP association is not possible.
	watchCancel context.CancelFunc
	watchDone   chan struct{}
}

// WSA structures and dynamic import for WSARecv.
type wsaBuf struct {
	Len uint32
	Buf *byte
}

var (
	ws2dll      = windows.NewLazySystemDLL("ws2_32.dll")
	procWSARecv = ws2dll.NewProc("WSARecv")
	procWSASend = ws2dll.NewProc("WSASend")
)

func wsaRecv(s windows.Handle, bufs *wsaBuf, bufCount uint32, bytes *uint32, flags *uint32, overlapped *windows.Overlapped) error {
	r1, _, e := procWSARecv.Call(
		uintptr(s),
		uintptr(unsafe.Pointer(bufs)),
		uintptr(bufCount),
		uintptr(unsafe.Pointer(bytes)),
		uintptr(unsafe.Pointer(flags)),
		uintptr(unsafe.Pointer(overlapped)),
		0,
	)
	if r1 == 0 {
		return nil
	}
	// SOCKET_ERROR expected with WSA_IO_PENDING for async.
	if errno, ok := e.(syscall.Errno); ok && errno == syscall.ERROR_IO_PENDING {
		return nil
	}
	return e
}

func wsaSend(s windows.Handle, bufs *wsaBuf, bufCount uint32, bytes *uint32, flags uint32, overlapped *windows.Overlapped) error {
	r1, _, e := procWSASend.Call(
		uintptr(s),
		uintptr(unsafe.Pointer(bufs)),
		uintptr(bufCount),
		uintptr(unsafe.Pointer(bytes)),
		uintptr(flags),
		uintptr(unsafe.Pointer(overlapped)),
		0,
	)
	if r1 == 0 {
		return nil
	}
	if errno, ok := e.(syscall.Errno); ok && errno == syscall.ERROR_IO_PENDING {
		return nil
	}
	return e
}

// overlappedOp carries the socket key and kind; it must be kept alive until completion.
type overlappedOp struct {
	windows.Overlapped
	regSock uintptr
	isSend  bool
}

// NewIOCPPoller returns an IOCP-based poller (experimental, windows+iocp build tag only).
func NewIOCPPoller() Poller { return newIOCPPoller() }

func newIOCPPoller() Poller {
	return &iocpPoller{regs: make(map[uintptr]*iocpReg), pending: make(map[*overlappedOp]struct{})}
}

// Replace the stub in poller_factory_windows.go when built with the 'iocp' tag.
func newIOCPIfAvailable() Poller { return NewIOCPPoller() }

func (p *iocpPoller) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	port, err := windows.CreateIoCompletionPort(windows.InvalidHandle, 0, 0, 0)
	if err != nil {
		return err
	}
	p.port = port
	p.wg.Add(1)
	go func() { defer p.wg.Done(); p.loop() }()
	return nil
}

func (p *iocpPoller) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.closed.Store(1)
	// Post a wake to unblock GetQueuedCompletionStatus.
	_ = windows.PostQueuedCompletionStatus(p.port, 0, 0, nil)
	p.mu.Lock()
	regs := p.regs
	p.regs = make(map[uintptr]*iocpReg)
	p.mu.Unlock()
	for _, r := range regs {
		r.disabled.Store(1)
		if r.stopW != nil {
			r.stopW()
			if r.closed != nil {
				select {
				case <-r.closed:
				case <-time.After(500 * time.Millisecond):
					// timeout: continue shutdown.
				}
			}
		}
		if r.watchCancel != nil {
			r.watchCancel()
		}
		if r.watchDone != nil {
			select {
			case <-r.watchDone:
			case <-time.After(500 * time.Millisecond):
			}
		}
		// Cancel pending I/O to speed shutdown
		_ = windows.CancelIoEx(r.sock, nil)
	}
	p.wg.Wait()
	if p.port != 0 {
		_ = windows.CloseHandle(p.port)
	}
	return nil
}

func (p *iocpPoller) Register(conn net.Conn, kinds []EventType, h Handler) error {
	if conn == nil || h == nil {
		return errors.New("invalid registration")
	}
	// Extract SOCKET handle.
	type sc interface {
		SyscallConn() (syscall.RawConn, error)
	}
	scc, ok := conn.(sc)
	if !ok {
		return errors.New("conn does not expose SyscallConn")
	}
	rc, err := scc.SyscallConn()
	if err != nil {
		return err
	}
	var s uintptr
	if er := rc.Control(func(fd uintptr) { s = fd }); er != nil {
		return er
	}
	sh := windows.Handle(s)
	// Associate with this IOCP (may fail if already associated with another IOCP).
	if assoc, err := windows.CreateIoCompletionPort(sh, p.port, 0, 0); err != nil || assoc == 0 {
		// Fallback: start per-connection watcher for this registration only.
		ctx, cancel := context.WithCancel(p.ctx)
		reg := &iocpReg{sock: sh, conn: conn, kinds: kinds, handler: h, closed: make(chan struct{}, 1), reader: bufio.NewReader(conn), watchCancel: cancel, watchDone: make(chan struct{})}
		p.mu.Lock()
		if old, exists := p.regs[s]; exists {
			// Idempotent update: refresh handler/kinds, keep existing watcher
			old.kinds = kinds
			old.handler = h
			p.mu.Unlock()
			cancel()
			return nil
		}
		p.regs[s] = reg
		p.mu.Unlock()
		go func(r *iocpReg) {
			defer close(r.watchDone)
			// simple fallback watcher: reuse goPoller-like detection.
			interval := 5 * time.Millisecond
			tick := time.NewTicker(interval)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					act := false
					if contains(r.kinds, Readable) {
						_ = r.conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
						if b, err := r.reader.Peek(1); err == nil && len(b) > 0 {
							if r.disabled.Load() == 0 {
								r.handler(Event{Conn: r.conn, Type: Readable})
							}
							act = true
						} else if err != nil && !isTimeout(err) {
							if r.disabled.Load() == 0 {
								r.handler(Event{Conn: r.conn, Type: Error, Err: err})
							}
							return
						}
					}
					if contains(r.kinds, Writable) && r.disabled.Load() == 0 {
						r.handler(Event{Conn: r.conn, Type: Writable})
						act = true
					}
					if act {
						if interval > 5*time.Millisecond {
							interval /= 2
							if interval < time.Millisecond {
								interval = time.Millisecond
							}
							tick.Reset(interval)
						}
					} else if interval < 50*time.Millisecond {
						interval *= 2
						tick.Reset(interval)
					}
				}
			}
		}(reg)
		return nil
	}
	reg := &iocpReg{sock: sh, conn: conn, kinds: kinds, handler: h, closed: make(chan struct{}, 1), reader: bufio.NewReader(conn)}
	p.mu.Lock()
	if old, exists := p.regs[s]; exists {
		// Idempotent update: refresh handler/kinds
		old.kinds = kinds
		old.handler = h
		p.mu.Unlock()
		return nil
	}
	p.regs[s] = reg
	p.mu.Unlock()
	// Post zero-byte receive to get readability notifications.
	if contains(kinds, Readable) {
		var buf wsaBuf
		var flags uint32
		o := &overlappedOp{regSock: s}
		reg.pending = o
		p.pendingMu.Lock()
		p.pending[o] = struct{}{}
		p.pendingMu.Unlock()
		_ = wsaRecv(sh, &buf, 1, nil, &flags, &o.Overlapped) // expect queued completion
	}
	// Writable: periodic notification (throttled).
	if contains(kinds, Writable) {
		wctx, cancel := context.WithCancel(p.ctx)
		reg.stopW = cancel
		p.wg.Add(1)
		go func(r *iocpReg) {
			defer p.wg.Done()
			t := time.NewTicker(getWritableInterval())
			defer t.Stop()
			for {
				select {
				case <-wctx.Done():
					close(r.closed)
					return
				case <-t.C:
					// Issue zero-byte WSASend to get a completion as writable signal.
					var sbuf wsaBuf // Len=0, Buf=nil
					o := &overlappedOp{regSock: s, isSend: true}
					r.sendPending = o
					p.pendingMu.Lock()
					p.pending[o] = struct{}{}
					p.pendingMu.Unlock()
					_ = wsaSend(sh, &sbuf, 1, nil, 0, &o.Overlapped)
				}
			}
		}(reg)
	}
	return nil
}

func (p *iocpPoller) Deregister(conn net.Conn) error {
	type sc interface {
		SyscallConn() (syscall.RawConn, error)
	}
	scc, ok := conn.(sc)
	// Try fast path by socket key when possible.
	var haveKey bool
	var s uintptr
	if ok {
		if rc, err := scc.SyscallConn(); err == nil {
			if er := rc.Control(func(fd uintptr) { s = fd }); er == nil {
				haveKey = true
			}
		}
	}
	p.mu.Lock()
	if haveKey {
		if reg, ok := p.regs[s]; ok {
			reg.disabled.Store(1)
			if reg.stopW != nil {
				reg.stopW()
			}
			if reg.watchCancel != nil {
				reg.watchCancel()
			}
			_ = windows.CancelIoEx(reg.sock, nil)
			delete(p.regs, s)
			p.mu.Unlock()
			if reg.stopW != nil && reg.closed != nil {
				select {
				case <-reg.closed:
				case <-time.After(500 * time.Millisecond):
				}
			}
			if reg.watchDone != nil {
				select {
				case <-reg.watchDone:
				case <-time.After(500 * time.Millisecond):
				}
			}
			return nil
		}
	}
	// Fallback: search by connection identity (conn may be closed and key unavailable).
	var foundKey uintptr
	var found *iocpReg
	for k, r := range p.regs {
		if r.conn == conn {
			foundKey = k
			found = r
			break
		}
	}
	if found != nil {
		found.disabled.Store(1)
		if found.stopW != nil {
			found.stopW()
		}
		if found.watchCancel != nil {
			found.watchCancel()
		}
		_ = windows.CancelIoEx(found.sock, nil)
		delete(p.regs, foundKey)
		p.mu.Unlock()
		if found.stopW != nil && found.closed != nil {
			select {
			case <-found.closed:
			case <-time.After(500 * time.Millisecond):
			}
		}
		if found.watchDone != nil {
			select {
			case <-found.watchDone:
			case <-time.After(500 * time.Millisecond):
			}
		}
		return nil
	}

	p.mu.Unlock()
	return nil
}

func (p *iocpPoller) loop() {
	for p.closed.Load() == 0 {
		var bytes uint32
		var key uintptr
		var overlapped *windows.Overlapped
		err := windows.GetQueuedCompletionStatus(p.port, &bytes, &key, &overlapped, windows.INFINITE)
		if p.closed.Load() == 1 {
			return
		}
		if overlapped == nil {
			// Port wake without overlapped: ignore (used only to wake loop).
			continue
		}
		// Completion for zero-byte WSARecv (or canceled).
		p.dispatchCompletion(overlapped, bytes, err, uintptr(key))
	}
}

func (p *iocpPoller) dispatchWritable(sockKey uintptr) {
	p.mu.RLock()
	reg := p.regs[sockKey]
	p.mu.RUnlock()
	if reg == nil || reg.disabled.Load() != 0 {
		return
	}
	if contains(reg.kinds, Writable) {
		reg.handler(Event{Conn: reg.conn, Type: Writable})
	}
}

func (p *iocpPoller) dispatchCompletion(ov *windows.Overlapped, transferred uint32, gqcsErr error, sockKey uintptr) {
	// Recover registration from overlapped container.
	o := (*overlappedOp)(unsafe.Pointer(ov))
	// drop from pending set to allow GC.
	p.pendingMu.Lock()
	delete(p.pending, o)
	p.pendingMu.Unlock()
	if sockKey == 0 {
		sockKey = o.regSock
	}
	p.mu.RLock()
	reg := p.regs[sockKey]
	p.mu.RUnlock()
	if reg == nil || reg.disabled.Load() != 0 {
		return
	}
	if !o.isSend && contains(reg.kinds, Readable) {
		// If completion failed or 0 bytes were transferred, decide EOF/error or re-arm
		if gqcsErr != nil || transferred == 0 {
			// If canceled, just return.
			if errno, ok := gqcsErr.(syscall.Errno); ok && errno == syscall.ERROR_OPERATION_ABORTED {
				return
			}
			if gqcsErr != nil {
				// Map common network errors.
				if errno, ok := gqcsErr.(syscall.Errno); ok {
					switch errno {
					case syscall.ERROR_NETNAME_DELETED, syscall.ERROR_BROKEN_PIPE:
						reg.handler(Event{Conn: reg.conn, Type: Error, Err: io.EOF})
						return
					case syscall.WSAECONNRESET:
						reg.handler(Event{Conn: reg.conn, Type: Error, Err: gqcsErr})
						return
					}
				}
			}
			// Probe to decide if this is EOF or spurious completion.
			_ = reg.conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
			if b, err := reg.reader.Peek(1); err == io.EOF || (err == nil && len(b) == 0) {
				reg.handler(Event{Conn: reg.conn, Type: Error, Err: io.EOF})
				return
			}
			// fallthrough to re-arm if data might be available shortly.
		}
		reg.handler(Event{Conn: reg.conn, Type: Readable})
		// Re-arm zero-byte read with a fresh overlapped.
		var buf wsaBuf
		var flags uint32
		neo := &overlappedOp{regSock: sockKey}
		reg.pending = neo
		p.pendingMu.Lock()
		p.pending[neo] = struct{}{}
		p.pendingMu.Unlock()
		_ = wsaRecv(reg.sock, &buf, 1, nil, &flags, &neo.Overlapped)
		return
	}
	// Writable path: a zero-byte send completed.
	if o.isSend && contains(reg.kinds, Writable) {
		reg.handler(Event{Conn: reg.conn, Type: Writable})
		return
	}
}
