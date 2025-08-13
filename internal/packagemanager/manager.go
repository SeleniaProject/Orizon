package packagemanager

import (
    "context"
    "fmt"

    semver "github.com/Masterminds/semver/v3"
)

// Manager ties Resolver and Registry to resolve and fetch packages.
type Manager struct {
    registry Registry
}

// NewManager constructs a Manager with the provided registry.
func NewManager(reg Registry) *Manager { return &Manager{registry: reg} }

// ResolveAndFetch resolves requirements against given index and returns CIDs for fetched packages.
// It first resolves versions using Resolver on a synthetic index derived from registry manifests,
// then fetches blobs and returns a mapping of package -> (version, cid).
func (m *Manager) ResolveAndFetch(ctx context.Context, reqs []Requirement, preferHigher bool) (map[PackageID]struct{ Version Version; CID CID }, error) {
    // Build an index from registry state
    manifests, err := m.registry.All(ctx)
    if err != nil { return nil, err }
    idx := make(PackageIndex)
    for _, mf := range manifests {
        idx[mf.Name] = append(idx[mf.Name], PackageVersion{ Name: mf.Name, Version: mf.Version, Dependencies: mf.Dependencies })
    }

    r := NewResolver(idx, ResolveOptions{ PreferHigher: preferHigher })
    res, err := r.Resolve(reqs)
    if err != nil { return nil, err }

    out := make(map[PackageID]struct{ Version Version; CID CID }, len(res))
    for name, ver := range res {
        c, _ := semver.NewConstraint("="+string(ver))
        cid, mf, err := m.registry.Find(ctx, name, c)
        if err != nil { return nil, fmt.Errorf("resolve ok but fetch missing %s@%s: %w", name, ver, err) }
        // Final fetch to ensure payload availability
        if _, err := m.registry.Fetch(ctx, cid); err != nil { return nil, err }
        out[name] = struct{ Version Version; CID CID }{ Version: mf.Version, CID: cid }
    }
    return out, nil
}


