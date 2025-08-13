package packagemanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"

	semver "github.com/Masterminds/semver/v3"
)

// CID is a content identifier computed from the package blob (content-addressed).
type CID string

// ComputeCID calculates a stable content identifier for the given bytes.
func ComputeCID(data []byte) CID {
	sum := sha256.Sum256(data)
	// Prefix for clarity; simple hex encoding keeps it portable and dependency-free.
	return CID("oz1-" + hex.EncodeToString(sum[:]))
}

// PackageManifest describes a package unit and its dependencies.
type PackageManifest struct {
	Name         PackageID
	Version      Version
	Dependencies []Dependency
}

// PackageBlob bundles the manifest with an opaque payload (e.g., tarball bytes).
type PackageBlob struct {
	Manifest PackageManifest
	Data     []byte
}

// Registry defines operations for a distributed content-addressed package registry.
type Registry interface {
	Publish(ctx context.Context, blob PackageBlob) (CID, error)
	Fetch(ctx context.Context, id CID) (PackageBlob, error)
	// Find locates a package version satisfying the constraint and returns its CID and manifest.
	Find(ctx context.Context, name PackageID, constraint *semver.Constraints) (CID, PackageManifest, error)
    // List returns all manifests for a given package name known to the registry cluster.
    List(ctx context.Context, name PackageID) ([]PackageManifest, error)
    // All returns all manifests known to the registry cluster.
    All(ctx context.Context) ([]PackageManifest, error)
}

var (
	// ErrNotFound is returned when a blob or package cannot be found anywhere in the registry cluster.
	ErrNotFound = errors.New("not found")
)

// versionList is a helper slice for sorting versions.
type versionList []PackageVersion

func (vl versionList) Len() int      { return len(vl) }
func (vl versionList) Swap(i, j int) { vl[i], vl[j] = vl[j], vl[i] }
func (vl versionList) Less(i, j int) bool {
	vi := mustSemver(vl[i].Version)
	vj := mustSemver(vl[j].Version)
	return vi.LessThan(vj)
}

// InMemoryRegistry is a simple, thread-safe, content-addressed registry with peer replication.
type InMemoryRegistry struct {
	mu    sync.RWMutex
	blobs map[CID]PackageBlob
	// name index for resolving by semver constraint
	index map[PackageID][]PackageVersion
	peers []*InMemoryRegistry
}

// NewInMemoryRegistry constructs an empty registry.
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		blobs: make(map[CID]PackageBlob),
		index: make(map[PackageID][]PackageVersion),
	}
}

// ConnectPeers sets bidirectional peer links among registries for replication and lookup.
func (r *InMemoryRegistry) ConnectPeers(peers ...*InMemoryRegistry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range peers {
		if p == nil || p == r {
			continue
		}
		r.peers = append(r.peers, p)
		// ensure reciprocal link
		p.mu.Lock()
		found := false
		for _, back := range p.peers {
			if back == r {
				found = true
				break
			}
		}
		if !found {
			p.peers = append(p.peers, r)
		}
		p.mu.Unlock()
	}
}

// Publish stores the blob locally, updates indexes, and replicates to peers (best-effort).
func (r *InMemoryRegistry) Publish(ctx context.Context, blob PackageBlob) (CID, error) {
	if blob.Data == nil {
		return "", errors.New("empty data")
	}
	id := ComputeCID(blob.Data)

	r.mu.Lock()
	if _, exists := r.blobs[id]; !exists {
		r.blobs[id] = blob
		pv := PackageVersion{Name: blob.Manifest.Name, Version: blob.Manifest.Version, Dependencies: blob.Manifest.Dependencies}
		r.index[blob.Manifest.Name] = append(r.index[blob.Manifest.Name], pv)
		sort.Sort(versionList(r.index[blob.Manifest.Name]))
	}
	peers := append([]*InMemoryRegistry(nil), r.peers...)
	r.mu.Unlock()

	// best-effort replication; ignore peer errors
	for _, p := range peers {
		select {
		case <-ctx.Done():
			return id, ctx.Err()
		default:
		}
		p.replicate(id, blob)
	}
	return id, nil
}

func (r *InMemoryRegistry) replicate(id CID, blob PackageBlob) {
	r.mu.Lock()
	if _, exists := r.blobs[id]; !exists {
		r.blobs[id] = blob
		pv := PackageVersion{Name: blob.Manifest.Name, Version: blob.Manifest.Version, Dependencies: blob.Manifest.Dependencies}
		r.index[blob.Manifest.Name] = append(r.index[blob.Manifest.Name], pv)
		sort.Sort(versionList(r.index[blob.Manifest.Name]))
	}
	r.mu.Unlock()
}

// Fetch returns a locally stored blob or queries peers sequentially.
func (r *InMemoryRegistry) Fetch(ctx context.Context, id CID) (PackageBlob, error) {
	r.mu.RLock()
	if b, ok := r.blobs[id]; ok {
		r.mu.RUnlock()
		return b, nil
	}
	peers := append([]*InMemoryRegistry(nil), r.peers...)
	r.mu.RUnlock()

	for _, p := range peers {
		select {
		case <-ctx.Done():
			return PackageBlob{}, ctx.Err()
		default:
		}
		p.mu.RLock()
		b, ok := p.blobs[id]
		p.mu.RUnlock()
		if ok {
			return b, nil
		}
	}
	return PackageBlob{}, ErrNotFound
}

// Find returns the highest version that satisfies the constraint (or lowest if none provided), searching peers if needed.
func (r *InMemoryRegistry) Find(ctx context.Context, name PackageID, constraint *semver.Constraints) (CID, PackageManifest, error) {
	// collect candidates locally first
	type cand struct {
		ver *semver.Version
		pv  PackageVersion
	}
	r.mu.RLock()
	local := append([]PackageVersion(nil), r.index[name]...)
	r.mu.RUnlock()

	pick := func(list []PackageVersion) (PackageVersion, bool) {
		bestIdx := -1
		var bestVer *semver.Version
		for i := range list {
			sv := mustSemver(list[i].Version)
			if constraint != nil && !constraint.Check(sv) {
				continue
			}
			if bestIdx == -1 || sv.GreaterThan(bestVer) {
				bestIdx, bestVer = i, sv
			}
		}
		if bestIdx >= 0 {
			return list[bestIdx], true
		}
		return PackageVersion{}, false
	}

	if pv, ok := pick(local); ok {
		// derive CID from stored blob
		r.mu.RLock()
		for cid, blob := range r.blobs {
			if blob.Manifest.Name == pv.Name && blob.Manifest.Version == pv.Version {
				m := blob.Manifest
				r.mu.RUnlock()
				return cid, m, nil
			}
		}
		r.mu.RUnlock()
	}

	// search peers
	r.mu.RLock()
	peers := append([]*InMemoryRegistry(nil), r.peers...)
	r.mu.RUnlock()
	for _, p := range peers {
		select {
		case <-ctx.Done():
			return "", PackageManifest{}, ctx.Err()
		default:
		}
		p.mu.RLock()
		pv, ok := pick(p.index[name])
		if ok {
			for cid, blob := range p.blobs {
				if blob.Manifest.Name == pv.Name && blob.Manifest.Version == pv.Version {
					m := blob.Manifest
					p.mu.RUnlock()
					return cid, m, nil
				}
			}
		}
		p.mu.RUnlock()
	}
	return "", PackageManifest{}, ErrNotFound
}

// List returns all manifests for the specified package name from local index then peers.
func (r *InMemoryRegistry) List(ctx context.Context, name PackageID) ([]PackageManifest, error) {
    out := make([]PackageManifest, 0)
    r.mu.RLock()
    local := append([]PackageVersion(nil), r.index[name]...)
    blobs := make(map[CID]PackageBlob, len(r.blobs))
    for k, v := range r.blobs { blobs[k] = v }
    peers := append([]*InMemoryRegistry(nil), r.peers...)
    r.mu.RUnlock()

    appendFrom := func(list []PackageVersion, sourceBlobs map[CID]PackageBlob) {
        for _, pv := range list {
            // Try to reconstruct manifest directly
            out = append(out, PackageManifest{ Name: pv.Name, Version: pv.Version, Dependencies: pv.Dependencies })
        }
    }
    appendFrom(local, blobs)

    for _, p := range peers {
        select { case <-ctx.Done(): return nil, ctx.Err(); default: }
        p.mu.RLock()
        for _, pv := range p.index[name] {
            out = append(out, PackageManifest{ Name: pv.Name, Version: pv.Version, Dependencies: pv.Dependencies })
        }
        p.mu.RUnlock()
    }
    // De-duplicate by name+version
    seen := make(map[string]bool, len(out))
    uniq := out[:0]
    for _, m := range out {
        key := string(m.Name)+"@"+string(m.Version)
        if !seen[key] { seen[key] = true; uniq = append(uniq, m) }
    }
    // Sort by version ascending for determinism
    sort.Slice(uniq, func(i, j int) bool {
        if uniq[i].Name != uniq[j].Name { return uniq[i].Name < uniq[j].Name }
        vi := mustSemver(uniq[i].Version)
        vj := mustSemver(uniq[j].Version)
        return vi.LessThan(vj)
    })
    return uniq, nil
}

// All returns every manifest across local and peer registries.
func (r *InMemoryRegistry) All(ctx context.Context) ([]PackageManifest, error) {
    r.mu.RLock()
    // collect local manifests
    out := make([]PackageManifest, 0)
    for name, vers := range r.index {
        for _, pv := range vers {
            out = append(out, PackageManifest{ Name: name, Version: pv.Version, Dependencies: pv.Dependencies })
        }
    }
    peers := append([]*InMemoryRegistry(nil), r.peers...)
    r.mu.RUnlock()

    for _, p := range peers {
        select { case <-ctx.Done(): return nil, ctx.Err(); default: }
        p.mu.RLock()
        for name, vers := range p.index {
            for _, pv := range vers {
                out = append(out, PackageManifest{ Name: name, Version: pv.Version, Dependencies: pv.Dependencies })
            }
        }
        p.mu.RUnlock()
    }
    // De-duplicate
    seen := make(map[string]bool, len(out))
    uniq := out[:0]
    for _, m := range out {
        key := string(m.Name)+"@"+string(m.Version)
        if !seen[key] { seen[key] = true; uniq = append(uniq, m) }
    }
    // Sort by name then version
    sort.Slice(uniq, func(i, j int) bool {
        if uniq[i].Name != uniq[j].Name { return uniq[i].Name < uniq[j].Name }
        vi := mustSemver(uniq[i].Version)
        vj := mustSemver(uniq[j].Version)
        return vi.LessThan(vj)
    })
    return uniq, nil
}
