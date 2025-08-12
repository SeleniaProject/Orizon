package runtime

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/orizon-lang/orizon/internal/runtime/netstack"
)

// MetricFunc returns a map of metric name -> value (float64 for compatibility).
// Names should be simple tokens using [a-zA-Z0-9_:] to ease exposition.
type MetricFunc func() map[string]float64

// StartMetricsServer starts a minimal text exposition endpoint for metrics on addr (host:port).
// The handler aggregates all provided collectors under "/metrics".
// It returns the bound address (which may differ if port 0 was used), and a shutdown function.
func StartMetricsServer(addr string, collectors map[string]MetricFunc) (string, func(ctx context.Context) error, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Text format exposition; keep it simple and deterministic
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		// Stable iteration by collector name
		names := make([]string, 0, len(collectors))
		for name := range collectors {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fn := collectors[name]
			if fn == nil {
				continue
			}
			// Stable order of metrics within a collector
			snapshot := fn()
			keys := make([]string, 0, len(snapshot))
			for k := range snapshot {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := snapshot[k]
				// Sanitize names into prometheus-like tokens
				metricName := sanitizeMetricToken(name + "_" + k)
				// Example line: runtime_tcp_accept_temp_errors 12
				fmt.Fprintf(w, "%s %g\n", metricName, v)
			}
		}
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	bound := ln.Addr().String()
	go func() {
		_ = srv.Serve(ln)
	}()
	stop := func(ctx context.Context) error {
		return srv.Shutdown(ctx)
	}
	return bound, stop, nil
}

// StartMetricsTLSServer starts a TLS-enabled metrics server using the provided
// TLS listener wrapper. This strengthens exposure for environments where
// plaintext endpoints are not acceptable. The handler and collectors are the
// same as StartMetricsServer.
func StartMetricsTLSServer(addr string, collectors map[string]MetricFunc, tlsCfg *tls.Config) (string, func(ctx context.Context) error, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		names := make([]string, 0, len(collectors))
		for name := range collectors {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fn := collectors[name]
			if fn == nil {
				continue
			}
			snapshot := fn()
			keys := make([]string, 0, len(snapshot))
			for k := range snapshot {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := snapshot[k]
				metricName := sanitizeMetricToken(name + "_" + k)
				fmt.Fprintf(w, "%s %g\n", metricName, v)
			}
		}
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	// Wrap listener with TLS using hardened defaults from netstack
	tlsLn := netstack.TLSServer(ln, tlsCfg)
	bound := tlsLn.Addr().String()
	go func() { _ = srv.Serve(tlsLn) }()
	stop := func(ctx context.Context) error { return srv.Shutdown(ctx) }
	return bound, stop, nil
}

// StartMetricsServerWithAuth starts a plaintext metrics server protected by a static bearer token.
// If token is empty, authentication is disabled.
func StartMetricsServerWithAuth(addr string, collectors map[string]MetricFunc, token string) (string, func(ctx context.Context) error, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		names := make([]string, 0, len(collectors))
		for name := range collectors {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fn := collectors[name]
			if fn == nil {
				continue
			}
			snapshot := fn()
			keys := make([]string, 0, len(snapshot))
			for k := range snapshot {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := snapshot[k]
				metricName := sanitizeMetricToken(name + "_" + k)
				fmt.Fprintf(w, "%s %g\n", metricName, v)
			}
		}
	})
	handler := http.Handler(mux)
	if token != "" {
		handler = bearerAuthMiddleware(token, handler)
	}
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 3 * time.Second}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	bound := ln.Addr().String()
	go func() { _ = srv.Serve(ln) }()
	stop := func(ctx context.Context) error { return srv.Shutdown(ctx) }
	return bound, stop, nil
}

// StartMetricsTLSServerWithAuth starts a TLS metrics server protected by a static bearer token.
// If token is empty, authentication is disabled.
func StartMetricsTLSServerWithAuth(addr string, collectors map[string]MetricFunc, tlsCfg *tls.Config, token string) (string, func(ctx context.Context) error, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		names := make([]string, 0, len(collectors))
		for name := range collectors {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fn := collectors[name]
			if fn == nil {
				continue
			}
			snapshot := fn()
			keys := make([]string, 0, len(snapshot))
			for k := range snapshot {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := snapshot[k]
				metricName := sanitizeMetricToken(name + "_" + k)
				fmt.Fprintf(w, "%s %g\n", metricName, v)
			}
		}
	})
	handler := http.Handler(mux)
	if token != "" {
		handler = bearerAuthMiddleware(token, handler)
	}
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 3 * time.Second}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	tlsLn := netstack.TLSServer(ln, tlsCfg)
	bound := tlsLn.Addr().String()
	go func() { _ = srv.Serve(tlsLn) }()
	stop := func(ctx context.Context) error { return srv.Shutdown(ctx) }
	return bound, stop, nil
}

// bearerAuthMiddleware protects an HTTP handler with a static bearer token.
// It accepts token via Authorization: Bearer <token> or query parameter access_token=<token>.
func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	const scheme = "Bearer "
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, scheme) && strings.TrimPrefix(auth, scheme) == token {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Query().Get("access_token") == token {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func sanitizeMetricToken(s string) string {
	// Replace unsupported chars with '_'
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == ':' {
			b[i] = c
		} else {
			b[i] = '_'
		}
	}
	// Avoid leading digits per Prometheus best practice by prefixing underscore
	if len(b) > 0 && b[0] >= '0' && b[0] <= '9' {
		return "_" + string(b)
	}
	// Collapse repeated underscores for readability
	out := strings.ReplaceAll(string(b), "__", "_")
	return out
}
