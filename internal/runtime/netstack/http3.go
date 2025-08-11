package netstack

import (
    "crypto/tls"
    "net"
    "net/http"
    "time"

    http3 "github.com/quic-go/quic-go/http3"
)

// HTTP3Server wraps http3.Server lifecycle.
type HTTP3Server struct {
    srv   *http3.Server
    pc    net.PacketConn
    addr  string
    close func() error
}

// NewHTTP3Server creates a server bound to addr with given TLS config and handler.
func NewHTTP3Server(addr string, tlsCfg *tls.Config, h http.Handler) *HTTP3Server {
    s := &http3.Server{Addr: addr, TLSConfig: tlsCfg, Handler: h}
    return &HTTP3Server{srv: s, addr: addr}
}

// Start begins serving HTTP/3 on an ephemeral UDP port if addr ends with ":0".
// Use Addr() to get the actual bound address.
func (s *HTTP3Server) Start() (string, error) {
    var err error
    s.pc, err = net.ListenPacket("udp", s.addr)
    if err != nil { return "", err }
    realAddr := s.pc.LocalAddr().String()
    done := make(chan struct{})
    go func(){
        _ = s.srv.Serve(s.pc)
        close(done)
    }()
    s.close = func() error {
        _ = s.pc.Close()
        select {
        case <-done:
        case <-time.After(time.Second):
        }
        return nil
    }
    return realAddr, nil
}

// Stop stops the server.
func (s *HTTP3Server) Stop() error {
    if s.close != nil { return s.close() }
    return nil
}

// HTTP3Client returns an http.Client using HTTP/3 round tripper with given TLS config.
func HTTP3Client(tlsCfg *tls.Config, timeout time.Duration) *http.Client {
    tr := &http3.Transport{TLSClientConfig: tlsCfg}
    return &http.Client{Transport: tr, Timeout: timeout}
}

// WithInsecureMinTLS12 returns a tls.Config with InsecureSkipVerify for local tests.
func WithInsecureMinTLS12() *tls.Config {
    return &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
}

// ShutdownHTTP3 gracefully closes the RoundTripper if applicable.
func ShutdownHTTP3(c *http.Client) {
    if tr, ok := c.Transport.(*http3.Transport); ok { _ = tr.Close() }
}


