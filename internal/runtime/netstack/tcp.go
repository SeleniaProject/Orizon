package netstack

import (
    "context"
    "net"
    "time"
)

// TCPServer wraps a TCP listener with a handler-based serve loop.
type TCPServer struct {
    ln     net.Listener
    addr   string
    closed chan struct{}
}

// NewTCPServer creates a new TCP server listening on addr (host:port).
func NewTCPServer(addr string) *TCPServer {
    return &TCPServer{addr: addr, closed: make(chan struct{})}
}

// Start begins accepting connections and invokes handler per connection.
func (s *TCPServer) Start(ctx context.Context, handler func(conn net.Conn)) error {
    if ctx == nil { ctx = context.Background() }
    ln, err := net.Listen("tcp", s.addr)
    if err != nil { return err }
    s.ln = ln
    go func(){
        defer close(s.closed)
        for {
            c, err := s.ln.Accept()
            if err != nil {
                select {
                case <-ctx.Done():
                    return
                default:
                    return
                }
            }
            go handler(c)
        }
    }()
    return nil
}

// Stop closes the listener and waits for the loop to end.
func (s *TCPServer) Stop() error {
    if s.ln != nil { _ = s.ln.Close() }
    <-s.closed
    return nil
}

// DialTCP dials a TCP connection with optional timeout.
func DialTCP(addr string, timeout time.Duration) (net.Conn, error) {
    d := net.Dialer{Timeout: timeout}
    return d.Dial("tcp", addr)
}


