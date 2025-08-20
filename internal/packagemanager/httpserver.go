package packagemanager

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	semver "github.com/Masterminds/semver/v3"

	timex "github.com/orizon-lang/orizon/internal/stdlib/time"
)

// ---- Metrics / Logging ----.
type endpointMetrics struct {
	c2xx   uint64
	c4xx   uint64
	c5xx   uint64
	cOther uint64
	// histogram-like buckets (seconds): 0.01, 0.05, 0.1, 0.5, 1.0, +Inf
	b001  uint64
	b005  uint64
	b010  uint64
	b050  uint64
	b100  uint64
	bInf  uint64
	sumNS uint64
	cnt   uint64
}

type metricsRecorder struct {
	by        map[string]*endpointMetrics
	inflight  int64
	rlDrops   uint64
	accessLog bool
}

func newMetricsRecorder() *metricsRecorder {
	mr := &metricsRecorder{by: make(map[string]*endpointMetrics)}
	// default endpoints we know; map grows on demand as well.
	for _, k := range []string{"healthz", "publish", "fetch", "find", "list", "all", "metrics"} {
		mr.by[k] = &endpointMetrics{}
	}

	v := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_ACCESS_LOG"))
	mr.accessLog = v == "1" || strings.EqualFold(v, "true")

	return mr
}

func (m *metricsRecorder) inc(name string, code int) {
	em, ok := m.by[name]
	if !ok {
		em = &endpointMetrics{}
		m.by[name] = em
	}

	switch code / 100 {
	case 2:
		atomic.AddUint64(&em.c2xx, 1)
	case 4:
		atomic.AddUint64(&em.c4xx, 1)
	case 5:
		atomic.AddUint64(&em.c5xx, 1)
	default:
		atomic.AddUint64(&em.cOther, 1)
	}
}

type statusWriter struct {
	rw   http.ResponseWriter
	code int
	n    int
}

func (s *statusWriter) Header() http.Header  { return s.rw.Header() }
func (s *statusWriter) WriteHeader(code int) { s.code = code; s.rw.WriteHeader(code) }
func (s *statusWriter) Write(b []byte) (int, error) {
	if s.code == 0 {
		s.code = 200
	}

	n, err := s.rw.Write(b)
	s.n += n

	return n, err
}

func (s *statusWriter) Flush() {
	if f, ok := s.rw.(http.Flusher); ok {
		f.Flush()
	}
}

// wrap applies security headers, CORS, preflight, metrics, and optional access logging.
func (m *metricsRecorder) wrap(name string, cors *corsCfg, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security headers first for any response.
		setSecurityHeaders(w, r)
		// CORS headers and preflight.
		if cors != nil {
			cors.apply(w, r)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			m.inc(name, http.StatusNoContent)

			return
		}
		// Request ID.
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = genReqID()
		}

		w.Header().Set("X-Request-ID", rid)

		start := time.Now()

		atomic.AddInt64(&m.inflight, 1)

		sw := &statusWriter{rw: w}
		// Panic recovery to ensure 500.
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("panic: %v request_id=%s", rec, rid)

					if sw.code == 0 {
						sw.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
			h(sw, r)
		}()

		if sw.code == 0 {
			sw.code = http.StatusOK
		}

		m.inc(name, sw.code)
		atomic.AddInt64(&m.inflight, -1)

		if m.accessLog {
			dur := time.Since(start)
			ua := r.Header.Get("User-Agent")
			log.Printf("%s %s -> %d %dB in %s from %s ua=%q", r.Method, r.URL.RequestURI(), sw.code, sw.n, dur, r.RemoteAddr, ua)
		}
		// latency buckets and sum/count
		if em, ok := m.by[name]; ok {
			d := time.Since(start)
			sec := d.Seconds()

			switch {
			case sec <= 0.01:
				atomic.AddUint64(&em.b001, 1)
			case sec <= 0.05:
				atomic.AddUint64(&em.b005, 1)
			case sec <= 0.10:
				atomic.AddUint64(&em.b010, 1)
			case sec <= 0.50:
				atomic.AddUint64(&em.b050, 1)
			case sec <= 1.0:
				atomic.AddUint64(&em.b100, 1)
			default:
				atomic.AddUint64(&em.bInf, 1)
			}

			atomic.AddUint64(&em.cnt, 1)
			atomic.AddUint64(&em.sumNS, uint64(d.Nanoseconds()))
		}
	}
}

func (m *metricsRecorder) serveMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	var b strings.Builder

	fmt.Fprintf(&b, "# TYPE orizon_inflight gauge\norizon_inflight %d\n", atomic.LoadInt64(&m.inflight))
	fmt.Fprintf(&b, "# TYPE orizon_ratelimit_dropped_total counter\norizon_ratelimit_dropped_total %d\n", atomic.LoadUint64(&m.rlDrops))
	fmt.Fprintf(&b, "# TYPE orizon_requests_total counter\n")

	for name, em := range m.by {
		fmt.Fprintf(&b, "orizon_requests_total{handler=\"%s\",class=\"2xx\"} %d\n", name, atomic.LoadUint64(&em.c2xx))
		fmt.Fprintf(&b, "orizon_requests_total{handler=\"%s\",class=\"4xx\"} %d\n", name, atomic.LoadUint64(&em.c4xx))
		fmt.Fprintf(&b, "orizon_requests_total{handler=\"%s\",class=\"5xx\"} %d\n", name, atomic.LoadUint64(&em.c5xx))
		fmt.Fprintf(&b, "orizon_requests_total{handler=\"%s\",class=\"other\"} %d\n", name, atomic.LoadUint64(&em.cOther))
		// latency buckets.
		fmt.Fprintf(&b, "# TYPE orizon_request_duration_seconds histogram\n")
		fmt.Fprintf(&b, "orizon_request_duration_seconds_bucket{handler=\"%s\",le=\"0.01\"} %d\n", name, atomic.LoadUint64(&em.b001))
		fmt.Fprintf(&b, "orizon_request_duration_seconds_bucket{handler=\"%s\",le=\"0.05\"} %d\n", name, atomic.LoadUint64(&em.b005))
		fmt.Fprintf(&b, "orizon_request_duration_seconds_bucket{handler=\"%s\",le=\"0.1\"} %d\n", name, atomic.LoadUint64(&em.b010))
		fmt.Fprintf(&b, "orizon_request_duration_seconds_bucket{handler=\"%s\",le=\"0.5\"} %d\n", name, atomic.LoadUint64(&em.b050))
		fmt.Fprintf(&b, "orizon_request_duration_seconds_bucket{handler=\"%s\",le=\"1\"} %d\n", name, atomic.LoadUint64(&em.b100))
		fmt.Fprintf(&b, "orizon_request_duration_seconds_bucket{handler=\"%s\",le=\"+Inf\"} %d\n", name, atomic.LoadUint64(&em.bInf))
		// sum and count.
		fmt.Fprintf(&b, "orizon_request_duration_seconds_sum{handler=\"%s\"} %.6f\n", name, float64(atomic.LoadUint64(&em.sumNS))/1e9)
		fmt.Fprintf(&b, "orizon_request_duration_seconds_count{handler=\"%s\"} %d\n", name, atomic.LoadUint64(&em.cnt))
	}

	_, _ = w.Write([]byte(b.String()))
}

// genReqID returns a random 16-byte hex string.
func genReqID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(b[:])
}

// ---- CORS ----.
type corsCfg struct {
	origins []string
	any     bool
}

func getCORS() *corsCfg {
	v := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_CORS_ORIGINS"))
	if v == "" {
		return nil
	}

	if v == "*" {
		return &corsCfg{any: true}
	}

	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return &corsCfg{origins: out}
}

func (c *corsCfg) allow(origin string) bool {
	if c == nil {
		return false
	}

	if c.any {
		return true
	}

	for _, o := range c.origins {
		if o == origin {
			return true
		}
	}

	return false
}

func (c *corsCfg) apply(w http.ResponseWriter, r *http.Request) {
	if c == nil {
		return
	}

	o := r.Header.Get("Origin")
	if o == "" {
		return
	}

	if c.allow(o) {
		if c.any {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Add("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,If-None-Match")
		w.Header().Set("Access-Control-Max-Age", "600")
	}
}

// buildHTTPMux builds the HTTP handlers for the registry.
func buildHTTPMux(reg Registry) *http.ServeMux {
	mux := http.NewServeMux()
	// optional global rate limiter.
	rl := getRateLimiter()
	m := newMetricsRecorder()
	cors := getCORS()
	// optional bearer token via env ORIZON_REGISTRY_TOKEN or header-only mode if empty.
	token := ""
	if v := httpTokenEnv(); v != "" {
		token = v
	}
	// auth mode: "write" (default) protects only write ops; "readwrite" protects all endpoints.
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_AUTH_MODE")))
	if mode != "readwrite" { // default
		mode = "write"
	}

	maxPublish := getMaxPublishBytes()
	authOK := func(r *http.Request) bool {
		if token == "" {
			return true
		}

		ah := r.Header.Get("Authorization")

		const p = "Bearer "

		if len(ah) <= len(p) || ah[:len(p)] != p {
			// Log failed authentication attempt.
			globalSecurityLogger.LogAuthenticationAttempt(false, r.Header.Get("User-Agent"), r.RemoteAddr, map[string]interface{}{
				"reason": "invalid_authorization_header",
				"method": r.Method,
				"path":   r.URL.Path,
			})

			return false
		}
		// Use constant-time comparison to prevent timing attacks.
		providedToken := ah[len(p):]
		if !SecureCompare(providedToken, token) {
			// Log failed authentication attempt.
			globalSecurityLogger.LogAuthenticationAttempt(false, r.Header.Get("User-Agent"), r.RemoteAddr, map[string]interface{}{
				"reason": "invalid_token",
				"method": r.Method,
				"path":   r.URL.Path,
			})

			return false
		}
		// Log successful authentication.
		globalSecurityLogger.LogAuthenticationAttempt(true, r.Header.Get("User-Agent"), r.RemoteAddr, map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})

		return true
	}
	// simple health endpoint.
	mux.HandleFunc("/healthz", m.wrap("healthz", cors, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"ok\":true}"))
	}))
	mux.HandleFunc("/publish", m.wrap("publish", cors, func(w http.ResponseWriter, r *http.Request) {
		if rl != nil && !rl.Allow(1) {
			w.Header().Set("Retry-After", "1")
			atomic.AddUint64(&m.rlDrops, 1)
			http.Error(w, "too many requests", http.StatusTooManyRequests)

			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)

			return
		}
		// publish is always protected when token is set.
		if token != "" && !authOK(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		// Validate request headers for security.
		validator := NewInputValidator()
		headers := make(map[string]string)

		for name, values := range r.Header {
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}

		if err := validator.ValidateHTTPHeaders(headers); err != nil {
			globalSecurityLogger.LogInputValidationFailure("http_headers", err.Error(), fmt.Sprintf("%v", headers))
			http.Error(w, "invalid request headers", http.StatusBadRequest)

			return
		}

		w = maybeGzip(w, r)
		// limit request size.
		r.Body = http.MaxBytesReader(w, r.Body, maxPublish)

		// Read and validate JSON payload.
		bodyBytes := make([]byte, maxPublish)

		n, err := r.Body.Read(bodyBytes)
		if err != nil && err.Error() != "EOF" {
			http.Error(w, "failed to read request body", http.StatusBadRequest)

			return
		}

		bodyBytes = bodyBytes[:n]

		// Validate JSON input for security.
		if err := validator.ValidateJSON(bodyBytes); err != nil {
			globalSecurityLogger.LogInputValidationFailure("json_payload", err.Error(), string(bodyBytes))
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)

			return
		}

		var fb struct {
			Manifest PackageManifest `json:"manifest"`
			Data     []byte          `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &fb); err != nil {
			http.Error(w, "JSON decode error", 400)

			return
		}

		// Validate package manifest fields.
		if err := validator.ValidatePackageID(string(fb.Manifest.Name)); err != nil {
			globalSecurityLogger.LogInputValidationFailure("package_name", err.Error(), string(fb.Manifest.Name))
			http.Error(w, "invalid package name", http.StatusBadRequest)

			return
		}

		if err := validator.ValidateVersion(string(fb.Manifest.Version)); err != nil {
			globalSecurityLogger.LogInputValidationFailure("package_version", err.Error(), string(fb.Manifest.Version))
			http.Error(w, "invalid package version", http.StatusBadRequest)

			return
		}

		// Validate package data size.
		if len(fb.Data) > int(maxPublish) {
			http.Error(w, "package data too large", http.StatusRequestEntityTooLarge)

			return
		}

		cid, err := reg.Publish(r.Context(), PackageBlob{Manifest: fb.Manifest, Data: fb.Data})
		if err != nil {
			log.Printf("Publish error: %v", err)
			http.Error(w, "internal server error", 500)

			return
		}
		// No-store for publish responses.
		w.Header().Set("Cache-Control", "no-store")
		writeJSONWithETag(w, r, struct {
			CID CID `json:"cid"`
		}{CID: cid})
	}))
	mux.HandleFunc("/fetch", m.wrap("fetch", cors, func(w http.ResponseWriter, r *http.Request) {
		if rl != nil && !rl.Allow(1) {
			w.Header().Set("Retry-After", "1")
			atomic.AddUint64(&m.rlDrops, 1)
			http.Error(w, "too many requests", http.StatusTooManyRequests)

			return
		}
		// protect reads only in readwrite mode.
		if token != "" && mode == "readwrite" && !authOK(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		// Validate query parameters.
		validator := NewInputValidator()

		cidParam := r.URL.Query().Get("cid")
		if cidParam == "" {
			http.Error(w, "missing cid parameter", http.StatusBadRequest)

			return
		}

		if err := validator.ValidateCID(cidParam); err != nil {
			log.Printf("Invalid CID in fetch request: %v", err)
			http.Error(w, "invalid CID parameter", http.StatusBadRequest)

			return
		}

		w = maybeGzip(w, r)
		if token != "" && mode == "readwrite" {
			w.Header().Add("Vary", "Authorization")
		}

		cid := CID(cidParam)

		blob, err := reg.Fetch(r.Context(), cid)
		if err != nil {
			if err == ErrNotFound {
				http.NotFound(w, r)

				return
			}

			log.Printf("Fetch error for CID %s: %v", cidParam, err)
			http.Error(w, "internal server error", 500)

			return
		}
		// Always require revalidation when caches are involved.
		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		writeJSONWithETag(w, r, struct {
			Manifest PackageManifest `json:"manifest"`
			Data     []byte          `json:"data"`
		}{Manifest: blob.Manifest, Data: blob.Data})
	}))
	mux.HandleFunc("/find", m.wrap("find", cors, func(w http.ResponseWriter, r *http.Request) {
		if rl != nil && !rl.Allow(1) {
			w.Header().Set("Retry-After", "1")
			atomic.AddUint64(&m.rlDrops, 1)
			http.Error(w, "too many requests", http.StatusTooManyRequests)

			return
		}

		if token != "" && mode == "readwrite" && !authOK(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		// Validate query parameters.
		validator := NewInputValidator()

		nameParam := r.URL.Query().Get("name")
		if nameParam == "" {
			http.Error(w, "missing name parameter", http.StatusBadRequest)

			return
		}

		if err := validator.ValidatePackageID(nameParam); err != nil {
			log.Printf("Invalid package name in find request: %v", err)
			http.Error(w, "invalid name parameter", http.StatusBadRequest)

			return
		}

		w = maybeGzip(w, r)
		if token != "" && mode == "readwrite" {
			w.Header().Add("Vary", "Authorization")
		}

		name := PackageID(nameParam)
		cons := r.URL.Query().Get("constraint")

		var c *semver.Constraints

		if cons != "" {
			// Validate constraint string.
			if err := validator.ValidateString(cons); err != nil {
				log.Printf("Invalid constraint string in find request: %v", err)
				http.Error(w, "invalid constraint parameter", http.StatusBadRequest)

				return
			}

			cc, err := semver.NewConstraint(cons)
			if err != nil {
				log.Printf("Semver constraint parse error: %v", err)
				http.Error(w, "invalid semantic version constraint", 400)

				return
			}

			c = cc
		}

		cid, m, err := reg.Find(r.Context(), name, c)
		if err != nil {
			if err == ErrNotFound {
				http.NotFound(w, r)

				return
			}

			log.Printf("Find error for package %s: %v", nameParam, err)
			http.Error(w, "internal server error", 500)

			return
		}

		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		writeJSONWithETag(w, r, struct {
			CID      CID             `json:"cid"`
			Manifest PackageManifest `json:"manifest"`
		}{CID: cid, Manifest: m})
	}))
	mux.HandleFunc("/list", m.wrap("list", cors, func(w http.ResponseWriter, r *http.Request) {
		if rl != nil && !rl.Allow(1) {
			w.Header().Set("Retry-After", "1")
			atomic.AddUint64(&m.rlDrops, 1)
			http.Error(w, "too many requests", http.StatusTooManyRequests)

			return
		}

		if token != "" && mode == "readwrite" && !authOK(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		w = maybeGzip(w, r)
		if token != "" && mode == "readwrite" {
			w.Header().Add("Vary", "Authorization")
		}

		name := PackageID(r.URL.Query().Get("name"))

		out, err := reg.List(r.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), 500)

			return
		}

		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		writeJSONWithETag(w, r, out)
	}))
	mux.HandleFunc("/all", m.wrap("all", cors, func(w http.ResponseWriter, r *http.Request) {
		if rl != nil && !rl.Allow(1) {
			w.Header().Set("Retry-After", "1")
			atomic.AddUint64(&m.rlDrops, 1)
			http.Error(w, "too many requests", http.StatusTooManyRequests)

			return
		}

		if token != "" && mode == "readwrite" && !authOK(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		w = maybeGzip(w, r)
		if token != "" && mode == "readwrite" {
			w.Header().Add("Vary", "Authorization")
		}

		out, err := reg.All(r.Context())
		if err != nil {
			http.Error(w, err.Error(), 500)

			return
		}

		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		writeJSONWithETag(w, r, out)
	}))
	// metrics endpoint (no rate limiting).
	mux.HandleFunc("/metrics", m.wrap("metrics", cors, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		m.serveMetrics(w, r)
	}))

	return mux
}

// StartHTTPServer serves the given registry over HTTP. Blocking.
func StartHTTPServer(reg Registry, addr string) error {
	mux := buildHTTPMux(reg)
	s := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}

	return s.ListenAndServe()
}

// StartHTTPServerTLS serves the registry over HTTPS using the provided certificate and key. Blocking.
func StartHTTPServerTLS(reg Registry, addr, certFile, keyFile string) error {
	mux := buildHTTPMux(reg)
	s := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}

	return s.ListenAndServeTLS(certFile, keyFile)
}

// StartHTTPServerGraceful starts the server and shuts down gracefully when ctx is done.
func StartHTTPServerGraceful(ctx context.Context, reg Registry, addr string) error {
	mux := buildHTTPMux(reg)
	s := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	errCh := make(chan error, 1)

	go func() { errCh <- s.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = s.Shutdown(shutCtx)

		return nil
	case err := <-errCh:
		return err
	}
}

// StartHTTPServerTLSGraceful starts HTTPS server and shuts down gracefully when ctx is done.
func StartHTTPServerTLSGraceful(ctx context.Context, reg Registry, addr, certFile, keyFile string) error {
	mux := buildHTTPMux(reg)
	s := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	errCh := make(chan error, 1)

	go func() { errCh <- s.ListenAndServeTLS(certFile, keyFile) }()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = s.Shutdown(shutCtx)

		return nil
	case err := <-errCh:
		return err
	}
}

// httpTokenEnv returns a bearer token from ORIZON_REGISTRY_TOKEN if set.
func httpTokenEnv() string { return os.Getenv("ORIZON_REGISTRY_TOKEN") }

// maybeGzip wraps the ResponseWriter with gzip writer when client accepts gzip.
func maybeGzip(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	ae := r.Header.Get("Accept-Encoding")
	if !strings.Contains(ae, "gzip") {
		return w
	}

	gz := gzip.NewWriter(w)
	rw := &gzipResponseWriter{rw: w, gz: gz}
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")

	return rw
}

type gzipResponseWriter struct {
	rw http.ResponseWriter
	gz *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gz.Write(b)
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	g.rw.Header().Del("Content-Length")
	g.rw.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Flush() {
	_ = g.gz.Flush()

	if f, ok := g.rw.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) Header() http.Header { return g.rw.Header() }

func (g *gzipResponseWriter) Close() error { return g.gz.Close() }

func closeIfGzip(w http.ResponseWriter) {
	if g, ok := w.(*gzipResponseWriter); ok {
		_ = g.Close()
	}
}

// writeJSONWithETag writes JSON with stable ETag support and 304 on If-None-Match.
func writeJSONWithETag(w http.ResponseWriter, r *http.Request, v any) {
	defer closeIfGzip(w)
	// Marshal first to compute strong ETag based on content (uncompressed entity).
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), 500)

		return
	}

	sum := sha256.Sum256(b)
	// Use a weak ETag to avoid ambiguity across different content-encodings (gzip vs identity).
	etag := fmt.Sprintf("W/\"%x\"", sum)

	inm := r.Header.Get("If-None-Match")
	if inm != "" {
		// naive match: if any token equals our ETag.
		for _, t := range strings.Split(inm, ",") {
			if strings.TrimSpace(t) == etag {
				w.WriteHeader(http.StatusNotModified)

				return
			}
		}
	}

	h := w.Header()
	h.Set("ETag", etag)

	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "application/json")
	}

	if h.Get("Cache-Control") == "" {
		h.Set("Cache-Control", "no-cache")
	}
	// If writer is our gzip wrapper, Write will compress b.
	_, _ = w.Write(b)
}

// setSecurityHeaders applies basic security headers per response.
func setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	if r.TLS != nil {
		// 2 years HSTS with preload and includeSubDomains.
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
	}
	// Prevent MIME type sniffing.
	h.Set("X-Content-Type-Options", "nosniff")
	// Prevent clickjacking.
	h.Set("X-Frame-Options", "DENY")
	// Control referrer information.
	h.Set("Referrer-Policy", "no-referrer")
	// Enable XSS protection in browsers that support it.
	h.Set("X-XSS-Protection", "1; mode=block")
	// Content Security Policy (restrictive for API).
	h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none';")
	// Prevent browsers from caching sensitive responses.
	if !strings.Contains(r.URL.Path, "/healthz") {
		h.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")
	}
}

// getMaxPublishBytes reads ORIZON_REGISTRY_MAX_PUBLISH_BYTES or returns default 50MB.
func getMaxPublishBytes() int64 {
	const def = int64(50 * 1024 * 1024)

	v := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_MAX_PUBLISH_BYTES"))
	if v == "" {
		return def
	}

	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}

	return n
}

// getRateLimiter reads ORIZON_REGISTRY_RATE_QPS and ORIZON_REGISTRY_RATE_BURST to create a TokenBucket.
// If variables are unset or invalid, rate limiting is disabled (returns nil).
func getRateLimiter() *timex.TokenBucket {
	qpsStr := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_RATE_QPS"))
	if qpsStr == "" {
		return nil
	}

	qps, err := strconv.ParseFloat(qpsStr, 64)
	if err != nil || qps <= 0 {
		return nil
	}

	burst := 1

	if b := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_RATE_BURST")); b != "" {
		if n, err := strconv.Atoi(b); err == nil && n >= 0 {
			burst = n
		}
	}

	return timex.NewTokenBucket(burst, qps)
}
