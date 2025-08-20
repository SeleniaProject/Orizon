package packagemanager

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"

	semver "github.com/Masterminds/semver/v3"
	"golang.org/x/sync/errgroup"
)

// Manager ties Resolver and Registry to resolve and fetch packages.
type Manager struct {
	registry Registry
}

// NewManager constructs a Manager with the provided registry.
func NewManager(reg Registry) *Manager { return &Manager{registry: reg} }

// ResolveAndFetch resolves requirements against given index and returns CIDs for fetched packages.
// It first resolves versions using Resolver on a synthetic index derived from registry manifests,.
// then fetches blobs and returns a mapping of package -> (version, cid).
func (m *Manager) ResolveAndFetch(ctx context.Context, reqs []Requirement, preferHigher bool) (map[PackageID]struct {
	Version Version
	CID     CID
}, error,
) {
	// Build a minimal index lazily by walking only the transitive closure of roots.
	// This avoids expensive Registry.All() on remote registries.
	idx := make(PackageIndex)
	loaded := make(map[PackageID]bool)
	// seed queue with roots.
	queue := make([]PackageID, 0, len(reqs))

	for _, r := range reqs {
		if !loaded[r.Name] {
			loaded[r.Name] = true

			queue = append(queue, r.Name)
		}
	}
	// parallel List(name) with bounded concurrency.
	type listRes struct {
		err  error
		name PackageID
		mans []PackageManifest
	}

	for len(queue) > 0 {
		batch := queue
		queue = nil
		ch := make(chan listRes, len(batch))
		lim := ioConcurrency()
		sem := make(chan struct{}, lim)

		for _, name := range batch {
			n := name
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			go func() {
				defer func() { <-sem }()

				mans, err := m.registry.List(ctx, n)
				ch <- listRes{name: n, mans: mans, err: err}
			}()
		}

		next := make(map[PackageID]bool)

		for i := 0; i < len(batch); i++ {
			r := <-ch
			if r.err != nil {
				return nil, r.err
			}

			for _, mf := range r.mans {
				idx[mf.Name] = append(idx[mf.Name], PackageVersion(mf))

				for _, d := range mf.Dependencies {
					if !loaded[d.Name] {
						next[d.Name] = true
					}
				}
			}
			// keep deterministic order per package.
			sort.Sort(versionList(idx[r.name]))
		}

		for n := range next {
			loaded[n] = true

			queue = append(queue, n)
		}
	}

	r := NewResolver(idx, ResolveOptions{PreferHigher: preferHigher})

	res, err := r.Resolve(reqs)
	if err != nil {
		return nil, err
	}

	out := make(map[PackageID]struct {
		Version Version
		CID     CID
	}, len(res))

	var mu sync.Mutex

	// Parallelize Find+Fetch to accelerate remote registries.
	g, gctx := errgroup.WithContext(ctx)
	// Limit concurrency (I/O bound); configurable via ORIZON_MAX_CONCURRENCY
	limit := ioConcurrency()
	sem := make(chan struct{}, limit)

	for name, ver := range res {
		name := name
		ver := ver

		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}

			defer func() { <-sem }()

			c, _ := semver.NewConstraint("=" + string(ver))

			cid, mf, err := m.registry.Find(gctx, name, c)
			if err != nil {
				return fmt.Errorf("resolve ok but fetch missing %s@%s: %w", name, ver, err)
			}

			if _, err := m.registry.Fetch(gctx, cid); err != nil {
				return err
			}

			mu.Lock()
			out[name] = struct {
				Version Version
				CID     CID
			}{Version: mf.Version, CID: cid}
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return out, nil
}

// ioConcurrency returns the concurrency for I/O bound tasks.
// It reads ORIZON_MAX_CONCURRENCY if set, otherwise uses GOMAXPROCS*8.
func ioConcurrency() int {
	if v := os.Getenv("ORIZON_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 1024 {
				return 1024
			}

			return n
		}
	}

	c := runtime.GOMAXPROCS(0) * 8
	if c < 4 {
		c = 4
	}

	if c > 1024 {
		c = 1024
	}

	return c
}
