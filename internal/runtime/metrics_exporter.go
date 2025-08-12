package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
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
