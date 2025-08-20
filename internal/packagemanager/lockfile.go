package packagemanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	semver "github.com/Masterminds/semver/v3"
)

// LockEntry pins a single package to an exact version and content (CID + SHA256).
type LockEntry struct {
	Name         PackageID    `json:"name"`
	Version      Version      `json:"version"`
	CID          CID          `json:"cid"`
	SHA256       string       `json:"sha256"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// Lockfile is a deterministic set of lock entries.
type Lockfile struct {
	Entries []LockEntry `json:"entries"`
}

// GenerateLockfile produces a Lockfile and its canonical JSON bytes from a resolution.
func GenerateLockfile(ctx context.Context, reg Registry, res Resolution) (Lockfile, []byte, error) {
	// Deterministic order by package name.
	names := make([]string, 0, len(res))
	for n := range res {
		names = append(names, string(n))
	}

	sort.Strings(names)

	entries := make([]LockEntry, 0, len(names))

	for _, ns := range names {
		name := PackageID(ns)
		ver := res[name]
		c, _ := semverConstraintForExact(ver)

		cid, mf, err := reg.Find(ctx, name, c)
		if err != nil {
			return Lockfile{}, nil, err
		}

		blob, err := reg.Fetch(ctx, cid)
		if err != nil {
			return Lockfile{}, nil, err
		}

		sum := sha256.Sum256(blob.Data)
		// Sort dependencies for determinism.
		deps := append([]Dependency(nil), mf.Dependencies...)
		sort.Slice(deps, func(i, j int) bool {
			if deps[i].Name != deps[j].Name {
				return deps[i].Name < deps[j].Name
			}

			return deps[i].Constraint < deps[j].Constraint
		})

		entries = append(entries, LockEntry{
			Name:         name,
			Version:      ver,
			CID:          cid,
			SHA256:       hex.EncodeToString(sum[:]),
			Dependencies: deps,
		})
	}
	// Canonicalize entries order once more in case of any mutation.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	lock := Lockfile{Entries: entries}

	b, err := marshalCanonicalJSON(lock)
	if err != nil {
		return Lockfile{}, nil, err
	}

	return lock, b, nil
}

// VerifyLockfile checks content hashes and basic metadata consistency for all entries.
func VerifyLockfile(ctx context.Context, reg Registry, lock Lockfile) error {
	// Ensure entries are sorted deterministically.
	if !isSortedLock(lock) {
		return errors.New("lockfile not sorted by name")
	}

	for _, e := range lock.Entries {
		blob, err := reg.Fetch(ctx, e.CID)
		if err != nil {
			return err
		}
		// Verify manifest metadata.
		if blob.Manifest.Name != e.Name || blob.Manifest.Version != e.Version {
			return errors.New("lockfile manifest mismatch")
		}
		// Verify content hash.
		sum := sha256.Sum256(blob.Data)
		if hex.EncodeToString(sum[:]) != e.SHA256 {
			return errors.New("lockfile checksum mismatch")
		}
	}

	return nil
}

// ResolutionFromLock reconstructs a Resolution from a Lockfile.
func ResolutionFromLock(lock Lockfile) Resolution {
	out := make(Resolution, len(lock.Entries))
	for _, e := range lock.Entries {
		out[e.Name] = e.Version
	}

	return out
}

func marshalCanonicalJSON(v any) ([]byte, error) {
	// encoding/json is deterministic for struct fields; arrays must be pre-sorted.
	// Indentation chosen for readability; does not include timestamps or volatile data.
	return json.MarshalIndent(v, "", "  ")
}

func isSortedLock(lock Lockfile) bool {
	return sort.SliceIsSorted(lock.Entries, func(i, j int) bool { return lock.Entries[i].Name < lock.Entries[j].Name })
}

func semverConstraintForExact(v Version) (*semver.Constraints, error) {
	return parseConstraint("=" + string(v))
}
