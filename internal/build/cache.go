package build

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheKey uniquely identifies a build artifact.
type CacheKey string

// Artifact represents cached build outputs.
// Files is a map of logical file name -> bytes.
// Metadata holds optional small key/value annotations.
type Artifact struct {
	Files    map[string][]byte
	Metadata map[string]string
}

// CacheStats exposes basic metrics.
type CacheStats struct {
	Hits      int64
	Misses    int64
	Entries   int64
	Bytes     int64
	Evictions int64
}

// Cache abstracts a key->artifact store.
type Cache interface {
	Get(key CacheKey) (Artifact, bool, error)
	Put(key CacheKey, a Artifact) error
	Exists(key CacheKey) bool
	Invalidate(key CacheKey) error
	Stats() CacheStats
}

// InMemoryLRUCache is a thread-safe LRU cache with a max entry count.
type InMemoryLRUCache struct {
	mu       sync.Mutex
	capacity int
	llHead   *lruNode
	llTail   *lruNode
	table    map[CacheKey]*lruNode
	stats    CacheStats
}

type lruNode struct {
	key  CacheKey
	val  Artifact
	size int64
	prev *lruNode
	next *lruNode
}

// NewInMemoryLRUCache creates a new cache with the given capacity (entries). If capacity<=0, defaults to 1024.
func NewInMemoryLRUCache(capacity int) *InMemoryLRUCache {
	if capacity <= 0 {
		capacity = 1024
	}
	return &InMemoryLRUCache{capacity: capacity, table: make(map[CacheKey]*lruNode)}
}

func (c *InMemoryLRUCache) moveToFront(n *lruNode) {
	if c.llHead == n {
		return
	}
	// detach
	if n.prev != nil {
		n.prev.next = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	}
	if c.llTail == n {
		c.llTail = n.prev
	}
	// insert at head
	n.prev = nil
	n.next = c.llHead
	if c.llHead != nil {
		c.llHead.prev = n
	}
	c.llHead = n
	if c.llTail == nil {
		c.llTail = n
	}
}

func (c *InMemoryLRUCache) evictIfNeeded() {
	for len(c.table) > c.capacity {
		// evict tail
		if c.llTail == nil {
			return
		}
		n := c.llTail
		delete(c.table, n.key)
		if n.prev != nil {
			n.prev.next = nil
		}
		c.llTail = n.prev
		if c.llTail == nil {
			c.llHead = nil
		}
		c.stats.Evictions++
		c.stats.Entries = int64(len(c.table))
		c.stats.Bytes -= n.size
	}
}

func artifactSize(a Artifact) int64 {
	var s int64
	for _, b := range a.Files {
		s += int64(len(b))
	}
	return s
}

func (c *InMemoryLRUCache) Get(key CacheKey) (Artifact, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.table[key]; ok {
		c.moveToFront(n)
		c.stats.Hits++
		return n.val, true, nil
	}
	c.stats.Misses++
	return Artifact{}, false, nil
}

func (c *InMemoryLRUCache) Put(key CacheKey, a Artifact) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.table[key]; ok {
		// update
		old := n.size
		n.val = a
		n.size = artifactSize(a)
		c.stats.Bytes += n.size - old
		c.moveToFront(n)
		return nil
	}
	n := &lruNode{key: key, val: a, size: artifactSize(a)}
	// insert at head
	n.next = c.llHead
	if c.llHead != nil {
		c.llHead.prev = n
	}
	c.llHead = n
	if c.llTail == nil {
		c.llTail = n
	}
	c.table[key] = n
	c.stats.Entries = int64(len(c.table))
	c.stats.Bytes += n.size
	c.evictIfNeeded()
	return nil
}

func (c *InMemoryLRUCache) Exists(key CacheKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.table[key]
	return ok
}

func (c *InMemoryLRUCache) Invalidate(key CacheKey) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.table[key]; ok {
		delete(c.table, key)
		if n.prev != nil {
			n.prev.next = n.next
		}
		if n.next != nil {
			n.next.prev = n.prev
		}
		if c.llHead == n {
			c.llHead = n.next
		}
		if c.llTail == n {
			c.llTail = n.prev
		}
		c.stats.Entries = int64(len(c.table))
		c.stats.Bytes -= n.size
	}
	return nil
}

func (c *InMemoryLRUCache) Stats() CacheStats { c.mu.Lock(); defer c.mu.Unlock(); return c.stats }

// FSCache stores artifacts on the filesystem under a root directory.
type FSCache struct {
	root  string
	mu    sync.Mutex
	stats CacheStats
}

// NewFSCache ensures the root directory exists.
func NewFSCache(root string) (*FSCache, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &FSCache{root: root}, nil
}

type fsManifest struct {
	Key       string            `json:"key"`
	CreatedAt time.Time         `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Files     []fsFileEntry     `json:"files"`
}

type fsFileEntry struct {
	Name       string `json:"name"`
	Blob       string `json:"blob"`
	Size       int64  `json:"size"`
	Compressed bool   `json:"compressed"`
	SHA256     string `json:"sha256"`
}

func (fc *FSCache) pathForKey(key CacheKey) string { return filepath.Join(fc.root, string(key)) }
func (fc *FSCache) pathForBlobs(key CacheKey) string {
	return filepath.Join(fc.pathForKey(key), "blobs")
}
func (fc *FSCache) pathForManifest(key CacheKey) string {
	return filepath.Join(fc.pathForKey(key), "manifest.json")
}

func (fc *FSCache) Get(key CacheKey) (Artifact, bool, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	manPath := fc.pathForManifest(key)
	b, err := os.ReadFile(manPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fc.stats.Misses++
			return Artifact{}, false, nil
		}
		return Artifact{}, false, err
	}
	var man fsManifest
	if err := json.Unmarshal(b, &man); err != nil {
		return Artifact{}, false, err
	}
	art := Artifact{Files: make(map[string][]byte), Metadata: man.Metadata}
	for _, fe := range man.Files {
		blobPath := filepath.Join(fc.pathForBlobs(key), fe.Blob)
		rb, err := os.ReadFile(blobPath)
		if err != nil {
			return Artifact{}, false, err
		}
		var data []byte
		if fe.Compressed {
			zr, err := gzip.NewReader(bytesReader(rb))
			if err != nil {
				return Artifact{}, false, err
			}
			d, err := io.ReadAll(zr)
			zr.Close()
			if err != nil {
				return Artifact{}, false, err
			}
			data = d
		} else {
			data = rb
		}
		// integrity check by size
		if int64(len(data)) != fe.Size {
			return Artifact{}, false, fmt.Errorf("size mismatch for %s", fe.Name)
		}
		art.Files[fe.Name] = data
	}
	fc.stats.Hits++
	return art, true, nil
}

// helper to get an io.Reader from bytes without extra alloc
type bytesReader []byte

func (b bytesReader) Read(p []byte) (int, error) {
	n := copy(p, b)
	if n < len(b) {
		return n, nil
	}
	return n, io.EOF
}

func (fc *FSCache) Put(key CacheKey, a Artifact) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	dir := fc.pathForKey(key)
	if err := os.MkdirAll(fc.pathForBlobs(key), 0o755); err != nil {
		return err
	}
	man := fsManifest{Key: string(key), CreatedAt: time.Now().UTC(), Metadata: a.Metadata}
	// write blobs
	var total int64
	for name, data := range a.Files {
		// compress data
		tmp := filepath.Join(fc.pathForBlobs(key), name+".gz.tmp")

		// Use a closure to ensure proper resource cleanup
		err := func() error {
			f, err := os.Create(tmp)
			if err != nil {
				return err
			}
			defer func() {
				f.Close()
				// Clean up temp file on error
				if err != nil {
					os.Remove(tmp)
				}
			}()

			gw := gzip.NewWriter(f)
			defer gw.Close()

			if _, err = gw.Write(data); err != nil {
				return err
			}

			if err = gw.Close(); err != nil {
				return err
			}

			if err = f.Close(); err != nil {
				return err
			}

			return nil
		}()

		if err != nil {
			return err
		}

		blob := name + ".gz"
		final := filepath.Join(fc.pathForBlobs(key), blob)
		if err := os.Rename(tmp, final); err != nil {
			return err
		}
		man.Files = append(man.Files, fsFileEntry{Name: name, Blob: blob, Size: int64(len(data)), Compressed: true})
		total += int64(len(data))
	}
	// write manifest atomically
	mb, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := fc.pathForManifest(key) + ".tmp"
	if err := os.WriteFile(tmp, mb, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, fc.pathForManifest(key)); err != nil {
		return err
	}
	fc.stats.Entries++
	fc.stats.Bytes += total
	return nil
}

func (fc *FSCache) Exists(key CacheKey) bool {
	_, err := os.Stat(fc.pathForManifest(key))
	return err == nil
}

func (fc *FSCache) Invalidate(key CacheKey) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	dir := fc.pathForKey(key)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	// best-effort removal
	err := os.RemoveAll(dir)
	if err == nil {
		fc.stats.Entries--
	}
	return err
}

func (fc *FSCache) Stats() CacheStats { fc.mu.Lock(); defer fc.mu.Unlock(); return fc.stats }
