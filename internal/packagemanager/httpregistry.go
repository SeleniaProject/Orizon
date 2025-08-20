package packagemanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"golang.org/x/sync/singleflight"
)

// HTTPRegistry is a Registry client that talks to a remote HTTP server.
type HTTPRegistry struct {
	base   string
	client *http.Client
	token  string
	// small in-memory caches (per-process) to coalesce repeated lookups.
	mu        sync.RWMutex
	listCache map[PackageID]struct {
		at   time.Time
		mans []PackageManifest
		etag string
	}
	findCache map[string]struct {
		at   time.Time
		cid  CID
		man  PackageManifest
		etag string
	}
	ttl time.Duration
	sf  singleflight.Group
}

// NewHTTPRegistry creates a client. It will use ORIZON_REGISTRY_TOKEN env as Bearer token if present.
func NewHTTPRegistry(baseURL string) *HTTPRegistry {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   256,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	tok := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY_TOKEN"))
	if tok == "" {
		if t2 := loadTokenFor(baseURL); t2 != "" {
			tok = t2
		}
	}

	return &HTTPRegistry{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Transport: tr, Timeout: 30 * time.Second},
		token:  tok,
		listCache: make(map[PackageID]struct {
			at   time.Time
			mans []PackageManifest
			etag string
		}),
		findCache: make(map[string]struct {
			at   time.Time
			cid  CID
			man  PackageManifest
			etag string
		}),
		ttl: 30 * time.Second,
	}
}

// NewHTTPRegistryWithAuth allows specifying a Bearer token explicitly.
func NewHTTPRegistryWithAuth(baseURL, token string) *HTTPRegistry {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   256,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &HTTPRegistry{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Transport: tr, Timeout: 30 * time.Second},
		token:  strings.TrimSpace(token),
		listCache: make(map[PackageID]struct {
			at   time.Time
			mans []PackageManifest
			etag string
		}),
		findCache: make(map[string]struct {
			at   time.Time
			cid  CID
			man  PackageManifest
			etag string
		}),
		ttl: 30 * time.Second,
	}
}

// credentials.json schema: { "registries": { "http://host:port": {"token": "..."} } }.
func loadTokenFor(baseURL string) string {
	b, err := os.ReadFile(filepath.Join(".orizon", "credentials.json"))
	if err != nil {
		return ""
	}

	var cfg struct {
		Registries map[string]struct {
			Token string `json:"token"`
		} `json:"registries"`
	}

	if json.Unmarshal(b, &cfg) != nil {
		return ""
	}

	for k, v := range cfg.Registries {
		if strings.TrimRight(k, "/") == strings.TrimRight(baseURL, "/") {
			return strings.TrimSpace(v.Token)
		}
	}

	return ""
}

func (r *HTTPRegistry) doWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		resp, err := r.client.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		// backoff: 100ms, 300ms, 900ms.
		time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
	}

	return nil, lastErr
}

func (r *HTTPRegistry) Publish(ctx context.Context, blob PackageBlob) (CID, error) {
	fb := struct {
		Manifest PackageManifest `json:"manifest"`
		Data     []byte          `json:"data"`
	}{Manifest: blob.Manifest, Data: blob.Data}
	b, _ := json.Marshal(fb)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, r.base+"/publish", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.doWithRetry(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("publish failed: %s", string(body))
	}

	var out struct {
		CID CID `json:"cid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	return out.CID, nil
}

func (r *HTTPRegistry) Fetch(ctx context.Context, id CID) (PackageBlob, error) {
	u := r.base + "/fetch?cid=" + url.QueryEscape(string(id))

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.doWithRetry(req)
	if err != nil {
		return PackageBlob{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return PackageBlob{}, ErrNotFound
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)

		return PackageBlob{}, fmt.Errorf("fetch failed: %s", string(body))
	}

	var fb struct {
		Manifest PackageManifest `json:"manifest"`
		Data     []byte          `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&fb); err != nil {
		return PackageBlob{}, err
	}

	return PackageBlob{Manifest: fb.Manifest, Data: fb.Data}, nil
}

func (r *HTTPRegistry) Find(ctx context.Context, name PackageID, constraint *semver.Constraints) (CID, PackageManifest, error) {
	// cache key: name|constraint.
	key := string(name) + "|"
	if constraint != nil {
		key += constraint.String()
	}

	r.mu.RLock()
	if c, ok := r.findCache[key]; ok && time.Since(c.at) < r.ttl {
		r.mu.RUnlock()

		return c.cid, c.man, nil
	}
	r.mu.RUnlock()

	v, err, _ := r.sf.Do("find:"+key, func() (any, error) {
		q := url.Values{}
		q.Set("name", string(name))

		if constraint != nil {
			q.Set("constraint", constraint.String())
		}

		u := r.base + "/find?" + q.Encode()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
		if r.token != "" {
			req.Header.Set("Authorization", "Bearer "+r.token)
		}
		// conditional request with ETag.
		r.mu.RLock()
		if c, ok := r.findCache[key]; ok && time.Since(c.at) < r.ttl && c.etag != "" {
			req.Header.Set("If-None-Match", c.etag)
		}
		r.mu.RUnlock()

		resp, err := r.doWithRetry(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotModified {
			// reuse cache.
			r.mu.RLock()
			cached := r.findCache[key]
			r.mu.RUnlock()

			return struct {
				CID      CID             `json:"cid"`
				Manifest PackageManifest `json:"manifest"`
			}{CID: cached.cid, Manifest: cached.man}, nil
		}

		if resp.StatusCode == 404 {
			return nil, ErrNotFound
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)

			return nil, fmt.Errorf("find failed: %s", string(body))
		}

		var out struct {
			CID      CID             `json:"cid"`
			Manifest PackageManifest `json:"manifest"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}

		etag := resp.Header.Get("ETag")

		r.mu.Lock()
		r.findCache[key] = struct {
			at   time.Time
			cid  CID
			man  PackageManifest
			etag string
		}{at: time.Now(), cid: out.CID, man: out.Manifest, etag: etag}
		r.mu.Unlock()

		return out, nil
	})
	if err != nil {
		return "", PackageManifest{}, err
	}

	out := v.(struct {
		CID      CID             `json:"cid"`
		Manifest PackageManifest `json:"manifest"`
	})

	return out.CID, out.Manifest, nil
}

func (r *HTTPRegistry) List(ctx context.Context, name PackageID) ([]PackageManifest, error) {
	r.mu.RLock()
	if c, ok := r.listCache[name]; ok && time.Since(c.at) < r.ttl {
		r.mu.RUnlock()

		return append([]PackageManifest(nil), c.mans...), nil
	}
	r.mu.RUnlock()

	v, err, _ := r.sf.Do("list:"+string(name), func() (any, error) {
		u := r.base + "/list?name=" + url.QueryEscape(string(name))

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
		if r.token != "" {
			req.Header.Set("Authorization", "Bearer "+r.token)
		}
		// conditional request with ETag.
		r.mu.RLock()
		if c, ok := r.listCache[name]; ok && time.Since(c.at) < r.ttl && c.etag != "" {
			req.Header.Set("If-None-Match", c.etag)
		}
		r.mu.RUnlock()

		resp, err := r.doWithRetry(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotModified {
			r.mu.RLock()
			cached := r.listCache[name]
			r.mu.RUnlock()

			return append([]PackageManifest(nil), cached.mans...), nil
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)

			return nil, fmt.Errorf("list failed: %s", string(body))
		}

		var out []PackageManifest
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}

		etag := resp.Header.Get("ETag")

		r.mu.Lock()
		r.listCache[name] = struct {
			at   time.Time
			mans []PackageManifest
			etag string
		}{at: time.Now(), mans: append([]PackageManifest(nil), out...), etag: etag}
		r.mu.Unlock()

		return out, nil
	})
	if err != nil {
		return nil, err
	}

	return v.([]PackageManifest), nil
}

func (r *HTTPRegistry) All(ctx context.Context) ([]PackageManifest, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, r.base+"/all", http.NoBody)
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.doWithRetry(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("all failed: %s", string(body))
	}

	var out []PackageManifest
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}
