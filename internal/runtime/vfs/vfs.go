package vfs

import (
    "context"
    "io"
    "io/fs"
    "path"
    "time"
)

// File represents an open file handle within a FileSystem.
type File interface {
    io.Reader
    io.Writer
    io.Seeker
    io.ReaderAt
    io.WriterAt
    io.Closer
    Stat() (fs.FileInfo, error)
    Sync() error
}

// FileSystem abstracts basic filesystem operations.
type FileSystem interface {
    Open(name string) (File, error)
    Create(name string) (File, error)
    Mkdir(name string, perm fs.FileMode) error
    MkdirAll(name string, perm fs.FileMode) error
    Remove(name string) error
    RemoveAll(name string) error
    Stat(name string) (fs.FileInfo, error)
    ReadDir(name string) ([]fs.DirEntry, error)
    Walk(root string, fn func(fullPath string, d fs.DirEntry, err error) error) error
}

// WatchOp indicates a change operation in the filesystem.
type WatchOp uint32

const (
    OpCreate WatchOp = 1 << iota
    OpWrite
    OpRemove
    OpRename
    OpChmod
)

// Event describes a filesystem change event.
type Event struct {
    Path string
    Op   WatchOp
    Time time.Time
}

// Watcher provides a platform-independent file watching API.
type Watcher interface {
    Events() <-chan Event
    Errors() <-chan error
    Add(name string) error
    Remove(name string) error
    Close() error
}

// Path utilities

// Join joins any number of path elements into a single path, using forward slashes.
func Join(elem ...string) string { return path.Join(elem...) }

// Clean returns the shortest path name equivalent to path by purely lexical processing.
func Clean(p string) string { return path.Clean(p) }

// WithTimeout derives a context with timeout helper for watchers.
func WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
    if parent == nil {
        parent = context.Background()
    }
    return context.WithTimeout(parent, d)
}


