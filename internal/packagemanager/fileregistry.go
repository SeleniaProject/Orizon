package packagemanager

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"

	semver "github.com/Masterminds/semver/v3"
)

// FileRegistry is a simple filesystem-backed Registry implementation.
// It stores each PackageBlob as JSON under baseDir/blobs/<cid>.json and builds an in-memory index on load.
type FileRegistry struct {
	mu      sync.RWMutex
	baseDir string
	blobs   map[CID]PackageBlob
	index   map[PackageID][]PackageVersion
	rev     map[string]CID // name@version -> CID
}

type fileBlob struct {
	Manifest PackageManifest `json:"manifest"`
	Data     []byte          `json:"data"`
}

// NewFileRegistry loads or initializes a registry at baseDir.
func NewFileRegistry(baseDir string) (*FileRegistry, error) {
	if baseDir == "" {
		return nil, errors.New("baseDir required")
	}
	if err := os.MkdirAll(filepath.Join(baseDir, "blobs"), 0o755); err != nil {
		return nil, err
	}
	fr := &FileRegistry{baseDir: baseDir, blobs: make(map[CID]PackageBlob), index: make(map[PackageID][]PackageVersion), rev: make(map[string]CID)}
	// fast path: try reading index.json
	if b, err := os.ReadFile(filepath.Join(baseDir, "index.json")); err == nil {
		// index file schema
		var idx struct {
			Entries []struct {
				Name         PackageID    `json:"name"`
				Version      Version      `json:"version"`
				CID          CID          `json:"cid"`
				Dependencies []Dependency `json:"dependencies,omitempty"`
			} `json:"entries"`
		}
		if json.Unmarshal(b, &idx) == nil {
			for _, e := range idx.Entries {
				fr.index[e.Name] = append(fr.index[e.Name], PackageVersion{Name: e.Name, Version: e.Version, Dependencies: e.Dependencies})
				fr.rev[string(e.Name)+"@"+string(e.Version)] = e.CID
			}
			for name := range fr.index {
				sort.Sort(versionList(fr.index[name]))
			}
		}
	}
	// scan blobs (fallback or to populate blobs cache lazily)
	err := filepath.WalkDir(filepath.Join(baseDir, "blobs"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var fb fileBlob
		if err := json.Unmarshal(b, &fb); err != nil {
			return err
		}
		cid := ComputeCID(fb.Data)
		pb := PackageBlob{Manifest: fb.Manifest, Data: fb.Data}
		fr.blobs[cid] = pb
		// populate index if absent
		key := string(fb.Manifest.Name) + "@" + string(fb.Manifest.Version)
		if _, ok := fr.rev[key]; !ok {
			fr.index[fb.Manifest.Name] = append(fr.index[fb.Manifest.Name], PackageVersion{Name: fb.Manifest.Name, Version: fb.Manifest.Version, Dependencies: fb.Manifest.Dependencies})
			sort.Sort(versionList(fr.index[fb.Manifest.Name]))
			fr.rev[key] = cid
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// write index.json if missing (best-effort)
	_ = fr.persistIndex()
	return fr, nil
}

func (r *FileRegistry) blobPath(cid CID) string {
	return filepath.Join(r.baseDir, "blobs", string(cid)+".json")
}

// Publish writes the blob if absent and updates the in-memory index.
func (r *FileRegistry) Publish(ctx context.Context, blob PackageBlob) (CID, error) {
	if blob.Data == nil {
		return "", errors.New("empty data")
	}
	id := ComputeCID(blob.Data)
	r.mu.Lock()
	if _, exists := r.blobs[id]; !exists {
		// persist
		fb := fileBlob{Manifest: blob.Manifest, Data: blob.Data}
		b, err := json.MarshalIndent(fb, "", "  ")
		if err != nil {
			r.mu.Unlock()
			return "", err
		}
		if err := os.WriteFile(r.blobPath(id), b, 0o644); err != nil {
			r.mu.Unlock()
			return "", err
		}
		r.blobs[id] = blob
		pv := PackageVersion{Name: blob.Manifest.Name, Version: blob.Manifest.Version, Dependencies: blob.Manifest.Dependencies}
		r.index[blob.Manifest.Name] = append(r.index[blob.Manifest.Name], pv)
		sort.Sort(versionList(r.index[blob.Manifest.Name]))
		r.rev[string(blob.Manifest.Name)+"@"+string(blob.Manifest.Version)] = id
		_ = r.persistIndex()
	}
	r.mu.Unlock()
	return id, nil
}

func (r *FileRegistry) Fetch(ctx context.Context, id CID) (PackageBlob, error) {
	r.mu.RLock()
	if b, ok := r.blobs[id]; ok {
		r.mu.RUnlock()
		return b, nil
	}
	r.mu.RUnlock()
	// attempt lazy load
	p := r.blobPath(id)
	bb, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PackageBlob{}, ErrNotFound
		}
		return PackageBlob{}, err
	}
	var fb fileBlob
	if err := json.Unmarshal(bb, &fb); err != nil {
		return PackageBlob{}, err
	}
	pb := PackageBlob{Manifest: fb.Manifest, Data: fb.Data}
	r.mu.Lock()
	r.blobs[id] = pb
	// ensure index populated
	if _, ok := r.index[fb.Manifest.Name]; !ok {
		r.index[fb.Manifest.Name] = []PackageVersion{}
	}
	found := false
	for _, pv := range r.index[fb.Manifest.Name] {
		if pv.Version == fb.Manifest.Version {
			found = true
			break
		}
	}
	if !found {
		r.index[fb.Manifest.Name] = append(r.index[fb.Manifest.Name], PackageVersion{Name: fb.Manifest.Name, Version: fb.Manifest.Version, Dependencies: fb.Manifest.Dependencies})
		sort.Sort(versionList(r.index[fb.Manifest.Name]))
	}
	r.mu.Unlock()
	return pb, nil
}

func (r *FileRegistry) Find(ctx context.Context, name PackageID, constraint *semver.Constraints) (CID, PackageManifest, error) {
	r.mu.RLock()
	list := append([]PackageVersion(nil), r.index[name]...)
	// use rev map if available; otherwise fallback to scanning blobs
	rev := make(map[string]CID, len(r.rev))
	for k, v := range r.rev {
		rev[k] = v
	}
	r.mu.RUnlock()
	// pick highest that satisfies
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
	if bestIdx < 0 {
		return "", PackageManifest{}, ErrNotFound
	}
	pv := list[bestIdx]
	key := string(pv.Name) + "@" + string(pv.Version)
	id, ok := rev[key]
	if !ok {
		// fallback: search blobs
		r.mu.RLock()
		for cid, b := range r.blobs {
			if b.Manifest.Name == pv.Name && b.Manifest.Version == pv.Version {
				id = cid
				ok = true
				break
			}
		}
		r.mu.RUnlock()
		if !ok {
			return "", PackageManifest{}, ErrNotFound
		}
	}
	// manifest
	return id, PackageManifest{Name: pv.Name, Version: pv.Version, Dependencies: pv.Dependencies}, nil
}

func (r *FileRegistry) List(ctx context.Context, name PackageID) ([]PackageManifest, error) {
	r.mu.RLock()
	list := append([]PackageVersion(nil), r.index[name]...)
	r.mu.RUnlock()
	out := make([]PackageManifest, 0, len(list))
	for _, pv := range list {
		out = append(out, PackageManifest{Name: pv.Name, Version: pv.Version, Dependencies: pv.Dependencies})
	}
	return out, nil
}

func (r *FileRegistry) All(ctx context.Context) ([]PackageManifest, error) {
	r.mu.RLock()
	out := make([]PackageManifest, 0)
	for name, vers := range r.index {
		for _, pv := range vers {
			out = append(out, PackageManifest{Name: name, Version: pv.Version, Dependencies: pv.Dependencies})
		}
	}
	r.mu.RUnlock()
	// sort for determinism by name then version
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		vi := mustSemver(out[i].Version)
		vj := mustSemver(out[j].Version)
		return vi.LessThan(vj)
	})
	return out, nil
}

// persistIndex writes index.json describing name/version -> CID for fast startup.
func (r *FileRegistry) persistIndex() error {
	r.mu.RLock()
	entries := make([]struct {
		Name         PackageID    `json:"name"`
		Version      Version      `json:"version"`
		CID          CID          `json:"cid"`
		Dependencies []Dependency `json:"dependencies,omitempty"`
	}, 0)
	for name, vers := range r.index {
		for _, pv := range vers {
			key := string(name) + "@" + string(pv.Version)
			cid := r.rev[key]
			entries = append(entries, struct {
				Name         PackageID    `json:"name"`
				Version      Version      `json:"version"`
				CID          CID          `json:"cid"`
				Dependencies []Dependency `json:"dependencies,omitempty"`
			}{Name: name, Version: pv.Version, CID: cid, Dependencies: pv.Dependencies})
		}
	}
	r.mu.RUnlock()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		vi := mustSemver(entries[i].Version)
		vj := mustSemver(entries[j].Version)
		return vi.LessThan(vj)
	})
	obj := struct {
		Entries any `json:"entries"`
	}{Entries: entries}
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.baseDir, "index.json"), b, 0o644)
}
