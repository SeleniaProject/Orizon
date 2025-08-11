package netstack

import (
    "crypto/tls"
    "net"
    "net/http"
    "time"

    http3 "github.com/quic-go/quic-go/http3"
    quic "github.com/quic-go/quic-go"
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

// HTTP3Options configures quic-go for HTTP/3 server/client.
type HTTP3Options struct {
    MaxIdleTimeout  time.Duration
    KeepAlivePeriod time.Duration
    Enable0RTT      bool
}

// NewHTTP3ServerWithOptions allows passing QUIC options.
func NewHTTP3ServerWithOptions(addr string, tlsCfg *tls.Config, h http.Handler, opts HTTP3Options) *HTTP3Server {
    qc := &quic.Config{}
    if opts.MaxIdleTimeout > 0 { qc.MaxIdleTimeout = opts.MaxIdleTimeout }
    if opts.KeepAlivePeriod > 0 { qc.KeepAlivePeriod = opts.KeepAlivePeriod }
    if opts.Enable0RTT { qc.Allow0RTT = true }
    s := &http3.Server{Addr: addr, TLSConfig: tlsCfg, Handler: h, QUICConfig: qc}
    return &HTTP3Server{srv: s, addr: addr}
}

// Start begins serving HTTP/3 on an ephemeral UDP port if addr ends with ":0".
// Use Addr() to get the actual bound address.
func (s *HTTP3Server) Start() (string, error) {
	var err error
	s.pc, err = net.ListenPacket("udp", s.addr)
	if err != nil {
		return "", err
	}
	realAddr := s.pc.LocalAddr().String()
	done := make(chan struct{})
	go func() {
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
	if s.close != nil {
		return s.close()
	}
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
	if tr, ok := c.Transport.(*http3.Transport); ok {
		_ = tr.Close()
	}
}

// HTTP3ClientWithOptions returns an HTTP/3 client using provided QUIC options.
func HTTP3ClientWithOptions(tlsCfg *tls.Config, timeout time.Duration, opts HTTP3Options) *http.Client {
    qc := &quic.Config{}
    if opts.MaxIdleTimeout > 0 { qc.MaxIdleTimeout = opts.MaxIdleTimeout }
    if opts.KeepAlivePeriod > 0 { qc.KeepAlivePeriod = opts.KeepAlivePeriod }
    if opts.Enable0RTT { qc.Allow0RTT = true }
    tr := &http3.Transport{TLSClientConfig: tlsCfg, QUICConfig: qc}
    return &http.Client{Transport: tr, Timeout: timeout}
}
