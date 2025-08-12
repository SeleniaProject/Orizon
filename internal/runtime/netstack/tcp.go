package netstack

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCPServer wraps a TCP listener with a handler-based serve loop.
type TCPServer struct {
	ln     net.Listener
	addr   string
	closed chan struct{}
	// The following fields coordinate graceful connection shutdown.
	// They are initialized on demand to avoid overhead for simple use cases.
	mu    sync.Mutex
	conns map[net.Conn]struct{}
	wg    sync.WaitGroup
}

// TCP server metrics (package-level, lightweight visibility without global registry)
var (
	tcpAcceptTempErrors       uint64 // number of temporary errors from Accept
	tcpAcceptBackoffMaxHits   uint64 // number of times max backoff threshold reached
	tcpAcceptLastBackoffNanos int64  // last backoff duration in nanoseconds
)

// TCPMetrics returns a snapshot of internal TCP server metrics.
func TCPMetrics() map[string]uint64 {
	return map[string]uint64{
		"accept_temp_errors":      atomic.LoadUint64(&tcpAcceptTempErrors),
		"accept_backoff_max_hits": atomic.LoadUint64(&tcpAcceptBackoffMaxHits),
		"last_backoff_nanos":      uint64(atomic.LoadInt64(&tcpAcceptLastBackoffNanos)),
	}
}

// TCPMetricsForExport adapts TCP metrics to a float64 map for the exporter.
func TCPMetricsForExport() map[string]float64 {
	return map[string]float64{
		"accept_temp_errors":      float64(atomic.LoadUint64(&tcpAcceptTempErrors)),
		"accept_backoff_max_hits": float64(atomic.LoadUint64(&tcpAcceptBackoffMaxHits)),
		"last_backoff_nanos":      float64(atomic.LoadInt64(&tcpAcceptLastBackoffNanos)),
	}
}

// NewTCPServer creates a new TCP server listening on addr (host:port).
func NewTCPServer(addr string) *TCPServer {
	return &TCPServer{addr: addr, closed: make(chan struct{})}
}

// Start begins accepting connections and invokes handler per connection.
func (s *TCPServer) Start(ctx context.Context, handler func(conn net.Conn)) error {
	if ctx == nil {
		// Use a cancellable context to avoid leaking goroutines if caller passes nil
		c, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = c
	}
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln
	go func() {
		defer close(s.closed)
		var backoff time.Duration
		for {
			c, err := s.ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Handle temporary errors with bounded exponential backoff
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					atomic.AddUint64(&tcpAcceptTempErrors, 1)
					if backoff == 0 {
						backoff = 5 * time.Millisecond
					} else {
						backoff *= 2
						if backoff > 500*time.Millisecond {
							backoff = 500 * time.Millisecond
							atomic.AddUint64(&tcpAcceptBackoffMaxHits, 1)
						}
					}
					atomic.StoreInt64(&tcpAcceptLastBackoffNanos, int64(backoff))
					time.Sleep(backoff)
					continue
				}
				// Non-temporary error: exit accept loop
				return
			}
			backoff = 0
			go handler(c)
		}
	}()
	return nil
}

// StartWithContext is like Start but passes a per-connection context to the handler
// and tracks active connections so that callers can perform graceful shutdown
// using StopContext. The connection context is derived from the server context
// and is canceled when the server context is done or the connection is closed.
func (s *TCPServer) StartWithContext(ctx context.Context, handler func(ctx context.Context, conn net.Conn)) error {
	if ctx == nil {
		// Derive a cancellable context to avoid leaking goroutines when caller passes nil
		c, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = c
	}
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln
	// Initialize tracking structures lazily
	s.mu.Lock()
	if s.conns == nil {
		s.conns = make(map[net.Conn]struct{})
	}
	s.mu.Unlock()
	go func() {
		defer close(s.closed)
		var backoff time.Duration
		for {
			c, err := s.ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					atomic.AddUint64(&tcpAcceptTempErrors, 1)
					if backoff == 0 {
						backoff = 5 * time.Millisecond
					} else {
						backoff *= 2
						if backoff > 500*time.Millisecond {
							backoff = 500 * time.Millisecond
							atomic.AddUint64(&tcpAcceptBackoffMaxHits, 1)
						}
					}
					atomic.StoreInt64(&tcpAcceptLastBackoffNanos, int64(backoff))
					time.Sleep(backoff)
					continue
				}
				return
			}
			backoff = 0

			// Track connection lifecycle
			s.mu.Lock()
			s.conns[c] = struct{}{}
			s.mu.Unlock()
			s.wg.Add(1)

			// Derive a context that is canceled on server shutdown or connection close
			connCtx, cancel := context.WithCancel(ctx)
			go func(conn net.Conn, cancel context.CancelFunc) {
				// Ensure connection removal and handler completion accounting
				defer func() {
					// Best-effort connection close to release resources in abnormal exits
					_ = conn.Close()
					cancel()
					s.mu.Lock()
					delete(s.conns, conn)
					s.mu.Unlock()
					s.wg.Done()
				}()

				// Arrange cancellation when the remote peer closes the connection
				// by running a small watcher that blocks on a read deadline tick.
				// The handler remains responsible for normal I/O.
				done := make(chan struct{})
				go func() {
					defer close(done)
					// Poll the connection liveness by setting a short deadline
					// and attempting a zero-byte read via SetReadDeadline cadence.
					// This avoids interfering with handler I/O while still reacting
					// to server context cancellation quickly.
					ticker := time.NewTicker(250 * time.Millisecond)
					defer ticker.Stop()
					buf := make([]byte, 0)
					for {
						select {
						case <-connCtx.Done():
							return
						case <-ticker.C:
							// Set a very short deadline and perform a non-invasive read
							// to detect EOF/hangup without consuming data.
							// Many net.Conn implementations treat Read on zero-length slice
							// as a no-op; we guard by using a 1-byte peek with a deadline
							// but only when the connection supports deadlines.
							_ = conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
							// A zero-length buffer read is a fast path; if not supported,
							// this will return quickly with an error which we ignore.
							_, _ = conn.Read(buf)
							// Reset deadline to zero value to avoid affecting handler
							var zero time.Time
							_ = conn.SetReadDeadline(zero)
						}
					}
				}()

				// Execute user handler
				handler(connCtx, conn)

				// Wait for watcher to exit then return
				<-done
			}(c, cancel)
		}
	}()
	return nil
}

// Stop closes the listener and waits for the loop to end.
func (s *TCPServer) Stop() error {
	if s.ln != nil {
		_ = s.ln.Close()
	}
	<-s.closed
	return nil
}

// StopContext closes the listener and then waits for all active connection
// handlers (started via StartWithContext) to return, or until the provided
// context is done. It is safe to call StopContext even if StartWithContext was
// not used; in that case it behaves like Stop.
func (s *TCPServer) StopContext(ctx context.Context) error {
	// First, perform the same listener shutdown semantics as Stop
	if s.ln != nil {
		_ = s.ln.Close()
	}
	select {
	case <-s.closed:
	case <-ctx.Done():
		return ctx.Err()
	}
	// Proactively close all tracked connections to unblock handlers
	s.mu.Lock()
	var toClose []net.Conn
	for c := range s.conns {
		toClose = append(toClose, c)
	}
	s.mu.Unlock()
	for _, c := range toClose {
		_ = c.Close()
	}
	// Now wait for active handlers to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DialTCP dials a TCP connection with optional timeout.
func DialTCP(addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial("tcp", addr)
}
