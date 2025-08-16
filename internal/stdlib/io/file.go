package io

import (
	"context"
	"crypto/rand"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/orizon-lang/orizon/internal/runtime/vfs"
)

// File exposes the minimal surface of vfs.File that stdlib wants to re-export.
type File = vfs.File

// FS wraps the runtime VFS to provide a clean stdlib facade.
type FS struct{ fs vfs.FileSystem }

// OS returns an FS backed by the host OS filesystem.
func OS() FS { return FS{fs: vfs.NewOS()} }

// Mem returns an in-memory FS useful for tests and ephemeral operations.
func Mem() FS { return FS{fs: vfs.NewMem()} }

// Open opens a file.
func (f FS) Open(name string) (File, error) { return f.fs.Open(name) }

// Create creates or truncates a file.
func (f FS) Create(name string) (File, error) { return f.fs.Create(name) }

// Mkdir makes a directory.
func (f FS) Mkdir(name string, perm fs.FileMode) error { return f.fs.Mkdir(name, perm) }

// MkdirAll makes parent directories as needed.
func (f FS) MkdirAll(name string, perm fs.FileMode) error { return f.fs.MkdirAll(name, perm) }

// Remove removes a file.
func (f FS) Remove(name string) error { return f.fs.Remove(name) }

// RemoveAll removes a path recursively.
func (f FS) RemoveAll(name string) error { return f.fs.RemoveAll(name) }

// Stat stats a file.
func (f FS) Stat(name string) (fs.FileInfo, error) { return f.fs.Stat(name) }

// ReadDir reads a directory.
func (f FS) ReadDir(name string) ([]fs.DirEntry, error) { return f.fs.ReadDir(name) }

// Walk traverses a directory tree.
func (f FS) Walk(root string, fn func(fullPath string, d fs.DirEntry, err error) error) error {
	return f.fs.Walk(root, fn)
}

// Exists returns true if path can be stat'ed.
func (f FS) Exists(name string) bool {
	_, err := f.fs.Stat(name)
	return err == nil
}

// ReadFile reads entire file content.
func (f FS) ReadFile(name string) ([]byte, error) {
	file, err := f.fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// Best-effort: if Stat reports a reasonable size, bounded read
	if info, err2 := file.Stat(); err2 == nil {
		if sz := info.Size(); sz > 0 && sz < 16<<20 { // cap 16MB prealloc
			return io.ReadAll(io.LimitReader(file, sz))
		}
	}
	return io.ReadAll(file)
}

// WriteFile creates/truncates and writes content, syncing before close.
func (f FS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	// Ensure parent dir for MemFS; OSFS MkdirAll is explicit by caller
	file, err := f.fs.Create(name)
	if err != nil {
		return err
	}
	// Ensure close on return
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	// Best-effort durability
	if err := file.Sync(); err != nil {
		return err
	}
	return nil
}

// AtomicWrite writes to a temp file in the same directory then renames it into place.
// Ensures durable write by syncing temp file before rename.
func (f FS) AtomicWrite(name string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(name)
	if err := f.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// generate random suffix
	bs := make([]byte, 6)
	if _, err := rand.Read(bs); err != nil {
		return err
	}
	tmp := filepath.Join(dir, ".tmp-"+filepath.Base(name)+"-"+fmtBytes(bs))
	file, err := f.fs.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		_ = f.fs.Remove(tmp)
		return err
	}
	if err := f.fs.Rename(tmp, name); err != nil {
		_ = f.fs.Remove(tmp)
		return err
	}
	return nil
}

// WithTempFile creates a temp file in dir, executes fn(path), then removes it.
func (f FS) WithTempFile(dir, pattern string, fn func(string) error) error {
	if dir == "" {
		dir = "."
	}
	// naive unique name
	bs := make([]byte, 6)
	if _, err := rand.Read(bs); err != nil {
		return err
	}
	name := filepath.Join(dir, pattern+"-"+fmtBytes(bs))
	h, err := f.fs.Create(name)
	if err != nil {
		return err
	}
	h.Close()
	defer f.fs.Remove(name)
	return fn(name)
}

// small hex encoding (no fmt to keep deps low)
func fmtBytes(b []byte) string {
	const hexd = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[2*i] = hexd[v>>4]
		out[2*i+1] = hexd[v&0x0f]
	}
	return string(out)
}

// CopyFile copies src to dst using a fixed-size buffer and sets permission via caller responsibility.
func (f FS) CopyFile(src, dst string, perm fs.FileMode) error {
	in, err := f.fs.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := f.fs.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Rename moves/renames a file or directory atomically when可能.
func (f FS) Rename(src, dst string) error { return f.fs.Rename(src, dst) }

// Move falls back to copy+remove if Rename is not supported by the backend.
func (f FS) Move(src, dst string) error {
	if err := f.fs.Rename(src, dst); err == nil {
		return nil
	}
	if err := f.CopyFile(src, dst, 0); err != nil {
		return err
	}
	return f.fs.Remove(src)
}

// Watcher wraps vfs.Watcher for file watching.
type Watcher = vfs.Watcher

// NewWatcher creates a portable polling watcher on top of fs.
func NewWatcher(f FS) *vfs.SimpleWatcher { return vfs.NewSimpleWatcher(f.fs) }

// StartPolling starts the watcher.
func StartPolling(w *vfs.SimpleWatcher, ctx context.Context, path string, interval time.Duration) error {
	return w.StartPolling(ctx, path, interval)
}

// Path utils re-export.
func Join(elem ...string) string { return vfs.Join(elem...) }
func Clean(p string) string      { return vfs.Clean(p) }
