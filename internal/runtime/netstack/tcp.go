package netstack

import (
	"context"
	"net"
	"sync/atomic"
	"time"
)

// TCPServer wraps a TCP listener with a handler-based serve loop.
type TCPServer struct {
	ln     net.Listener
	addr   string
	closed chan struct{}
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

// Stop closes the listener and waits for the loop to end.
func (s *TCPServer) Stop() error {
	if s.ln != nil {
		_ = s.ln.Close()
	}
	<-s.closed
	return nil
}

// DialTCP dials a TCP connection with optional timeout.
func DialTCP(addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.Dial("tcp", addr)
}
