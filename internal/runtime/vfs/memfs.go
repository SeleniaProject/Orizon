package vfs

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"strings"
	"sync"
	"time"
)

type memFile struct {
	mu   sync.RWMutex
	name string
	buf  *bytes.Reader
	wbuf *bytes.Buffer
	mode fs.FileMode
	mod  time.Time
}

func newMemFile(name string, data []byte, mode fs.FileMode) *memFile {
	return &memFile{name: name, buf: bytes.NewReader(data), wbuf: bytes.NewBuffer(append([]byte(nil), data...)), mode: mode, mod: time.Now()}
}

func (f *memFile) Read(p []byte) (int, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.buf.Read(p)
}
func (f *memFile) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, err := f.wbuf.Write(p)
	f.buf = bytes.NewReader(f.wbuf.Bytes())
	f.mod = time.Now()
	return n, err
}
func (f *memFile) ReadAt(p []byte, off int64) (int, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.buf.ReadAt(p, off)
}
func (f *memFile) WriteAt(p []byte, off int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if off < 0 {
		return 0, errors.New("negative offset")
	}
	data := f.wbuf.Bytes()
	if int(off) > len(data) {
		pad := make([]byte, int(off)-len(data))
		f.wbuf.Write(pad)
	}
	// simplistic WriteAt: rebuild buffer
	data = f.wbuf.Bytes()
	end := int(off) + len(p)
	if end > len(data) {
		tmp := make([]byte, end)
		copy(tmp, data)
		copy(tmp[int(off):], p)
		f.wbuf = bytes.NewBuffer(tmp)
	} else {
		copy(data[int(off):int(off)+len(p)], p)
		f.wbuf = bytes.NewBuffer(data)
	}
	f.buf = bytes.NewReader(f.wbuf.Bytes())
	f.mod = time.Now()
	return len(p), nil
}
func (f *memFile) Seek(off int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Seek(off, whence)
}
func (f *memFile) Close() error { return nil }
func (f *memFile) Sync() error  { return nil }
func (f *memFile) Stat() (fs.FileInfo, error) {
	return fileInfo{name: path.Base(f.name), size: int64(f.buf.Len()), mode: f.mode, mod: f.mod}, nil
}

type fileInfo struct {
	name string
	size int64
	mode fs.FileMode
	mod  time.Time
}

func (fi fileInfo) Name() string       { return fi.name }
func (fi fileInfo) Size() int64        { return fi.size }
func (fi fileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi fileInfo) ModTime() time.Time { return fi.mod }
func (fi fileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fileInfo) Sys() any           { return nil }

type MemFS struct {
	mu   sync.RWMutex
	ents map[string]*memEnt
}

type memEnt struct {
	file *memFile
	dir  bool
}

func NewMem() *MemFS { return &MemFS{ents: make(map[string]*memEnt)} }

func norm(p string) string {
	q := Clean(p)
	if strings.HasPrefix(q, "/") {
		q = strings.TrimPrefix(q, "/")
	}
	return q
}

func (m *MemFS) ensureDir(p string) {
	p = norm(p)
	if p == "" {
		return
	}
	parts := strings.Split(p, "/")
	cur := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur = path.Join(cur, part)
		if _, ok := m.ents[cur]; !ok {
			m.ents[cur] = &memEnt{dir: true}
		}
	}
}

func (m *MemFS) Open(name string) (File, error) {
	key := norm(name)
	m.mu.RLock()
	e := m.ents[key]
	m.mu.RUnlock()
	if e == nil || e.dir {
		return nil, fs.ErrNotExist
	}
	return e.file, nil
}

func (m *MemFS) Create(name string) (File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureDir(path.Dir(name))
	f := newMemFile(name, nil, 0)
	m.ents[norm(name)] = &memEnt{file: f}
	return f, nil
}

func (m *MemFS) Mkdir(name string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureDir(name)
	m.ents[norm(name)] = &memEnt{dir: true}
	return nil
}
func (m *MemFS) MkdirAll(name string, perm fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureDir(name)
	return nil
}
func (m *MemFS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.ents, norm(name))
	return nil
}
func (m *MemFS) RemoveAll(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := norm(name)
	for k := range m.ents {
		if k == key || strings.HasPrefix(k, key+"/") {
			delete(m.ents, k)
		}
	}
	return nil
}
func (m *MemFS) Rename(oldpath, newpath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldk := norm(oldpath)
	newk := norm(newpath)
	e, ok := m.ents[oldk]
	if !ok {
		return fs.ErrNotExist
	}
	// Move the entry
	delete(m.ents, oldk)
	m.ensureDir(path.Dir(newk))
	m.ents[newk] = e
	// Update nested paths when moving a directory
	if e.dir {
		prefix := oldk + "/"
		var updates = make(map[string]*memEnt)
		for k, v := range m.ents {
			if strings.HasPrefix(k, prefix) {
				rest := strings.TrimPrefix(k, prefix)
				updates[path.Join(newk, rest)] = v
				delete(m.ents, k)
			}
		}
		for k, v := range updates {
			m.ents[k] = v
		}
	}
	return nil
}
func (m *MemFS) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := norm(name)
	e := m.ents[key]
	if e == nil {
		return nil, fs.ErrNotExist
	}
	if e.dir {
		return fileInfo{name: path.Base(key), mode: fs.ModeDir}, nil
	}
	return e.file.Stat()
}

func (m *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []fs.DirEntry
	prefix := norm(name)
	base := prefix
	if base != "" {
		base += "/"
	}
	seen := make(map[string]struct{})
	for p := range m.ents {
		// limit to direct children under prefix
		if prefix != "" {
			if p != prefix && !strings.HasPrefix(p, base) {
				continue
			}
			if p == prefix {
				continue
			}
			rest := strings.TrimPrefix(p, base)
			first := rest
			if i := strings.IndexByte(rest, '/'); i >= 0 {
				first = rest[:i]
			}
			if first == "" {
				continue
			}
			if _, ok := seen[first]; ok {
				continue
			}
			seen[first] = struct{}{}
			info, err := m.Stat(path.Join(prefix, first))
			if err != nil || info == nil {
				continue
			}
			out = append(out, fs.FileInfoToDirEntry(info))
			continue
		}
		// top-level listing
		rest := p
		first := rest
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			first = rest[:i]
		}
		if first == "" {
			continue
		}
		if _, ok := seen[first]; ok {
			continue
		}
		seen[first] = struct{}{}
		info, err := m.Stat(first)
		if err != nil || info == nil {
			continue
		}
		out = append(out, fs.FileInfoToDirEntry(info))
	}
	return out, nil
}

func (m *MemFS) Walk(root string, fn func(fullPath string, d fs.DirEntry, err error) error) error {
	if fn == nil {
		return errors.New("nil walk fn")
	}
	entries, _ := m.ReadDir(root)
	for _, de := range entries {
		p := path.Join(root, de.Name())
		if err := fn(p, de, nil); err != nil {
			return err
		}
		if de.IsDir() {
			if err := m.Walk(p, fn); err != nil {
				return err
			}
		}
	}
	return nil
}
