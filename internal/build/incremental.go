package build

import (
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

// FileState represents input file metadata used for change detection.
type FileState struct {
    Path    string
    Size    int64
    ModTime time.Time
    SHA256  string
}

// Snapshot holds a deterministic state of input files by target.
type Snapshot struct {
    // TargetID -> sorted list of FileState
    Inputs map[TargetID][]FileState
}

// IncrementalEngine computes dirty targets based on file snapshots.
type IncrementalEngine struct{}

func NewIncrementalEngine() *IncrementalEngine { return &IncrementalEngine{} }

// HashFile computes SHA-256 for a file.
func HashFile(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil { return "", err }
    defer f.Close()
    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil { return "", err }
    return hex.EncodeToString(h.Sum(nil)), nil
}

// SnapshotInputs records FileState for the provided file globs per target.
// The map values may include globs separated by path list separators.
func (ie *IncrementalEngine) SnapshotInputs(targetToGlobs map[TargetID][]string) (Snapshot, error) {
    out := Snapshot{ Inputs: make(map[TargetID][]FileState) }
    for tid, globs := range targetToGlobs {
        var files []string
        for _, g := range globs {
            parts := strings.Split(g, string(os.PathListSeparator))
            for _, p := range parts {
                matches, err := filepath.Glob(p)
                if err != nil { return Snapshot{}, err }
                files = append(files, matches...)
            }
        }
        sort.Strings(files)
        states := make([]FileState, 0, len(files))
        for _, f := range files {
            info, err := os.Stat(f)
            if err != nil { return Snapshot{}, err }
            sum, err := HashFile(f)
            if err != nil { return Snapshot{}, err }
            states = append(states, FileState{ Path: f, Size: info.Size(), ModTime: info.ModTime().UTC(), SHA256: sum })
        }
        out.Inputs[tid] = states
    }
    return out, nil
}

// Diff compares two snapshots and returns the set of targets that are dirty.
func (ie *IncrementalEngine) Diff(prev, curr Snapshot) ([]TargetID, error) {
    if prev.Inputs == nil || curr.Inputs == nil { return nil, errors.New("invalid snapshot") }
    dirty := make([]TargetID, 0)
    // union of keys
    keys := make(map[TargetID]bool)
    for k := range prev.Inputs { keys[k] = true }
    for k := range curr.Inputs { keys[k] = true }
    for k := range keys {
        a := prev.Inputs[k]
        b := curr.Inputs[k]
        if len(a) != len(b) { dirty = append(dirty, k); continue }
        // compare ordered lists
        diff := false
        for i := range a {
            if a[i].Path != b[i].Path || a[i].Size != b[i].Size || !a[i].ModTime.Equal(b[i].ModTime) || a[i].SHA256 != b[i].SHA256 {
                diff = true; break
            }
        }
        if diff { dirty = append(dirty, k) }
    }
    // sort deterministic
    sort.Slice(dirty, func(i, j int) bool { return dirty[i] < dirty[j] })
    return dirty, nil
}


