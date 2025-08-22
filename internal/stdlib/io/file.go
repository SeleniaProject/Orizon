// Package io provides comprehensive input/output operations, file handling,
// stream processing, serialization, and advanced I/O patterns.
// This package includes high-performance buffering, memory-mapped files,
// concurrent I/O operations, and enterprise-grade file management.
package io

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orizon-lang/orizon/internal/runtime/vfs"
)

// File exposes the minimal surface of vfs.File that stdlib wants to re-export.
type File = vfs.File

// FileMode represents file permissions and mode flags.
type FileMode = fs.FileMode

// FileInfo provides information about a file.
type FileInfo = fs.FileInfo

// DirEntry represents a directory entry.
type DirEntry = fs.DirEntry

// FS wraps the runtime VFS to provide a clean stdlib facade.
type FS struct {
	fs      vfs.FileSystem
	cache   FileCache
	watcher *FileWatcher
}

// FileCache provides file caching capabilities.
type FileCache struct {
	enabled     bool
	maxSize     int64
	currentSize int64
	entries     map[string]*CacheEntry
	lru         *LRUCache
	mutex       sync.RWMutex
}

// CacheEntry represents a cached file entry.
type CacheEntry struct {
	Data        []byte
	ModTime     time.Time
	Size        int64
	AccessCount int64
	LastAccess  time.Time
	Checksum    uint32
}

// LRUCache implements a least-recently-used cache.
type LRUCache struct {
	capacity int
	size     int
	head     *CacheNode
	tail     *CacheNode
	items    map[string]*CacheNode
	mutex    sync.RWMutex
}

// CacheNode represents a node in the LRU cache.
type CacheNode struct {
	key   string
	value interface{}
	prev  *CacheNode
	next  *CacheNode
}

// FileWatcher provides file system monitoring capabilities.
type FileWatcher struct {
	watchers map[string]*Watch
	events   chan FileEvent
	errors   chan error
	mutex    sync.RWMutex
	running  int32
}

// FileEvent represents a file system event.
type FileEvent struct {
	Path      string
	Operation FileOperation
	Time      time.Time
	Size      int64
	Mode      FileMode
}

// FileOperation represents the type of file operation.
type FileOperation int

const (
	FileCreated FileOperation = iota
	FileModified
	FileDeleted
	FileMoved
	FilePermissionChanged
	FileAttributeChanged
)

// Watch represents a file watch configuration.
type Watch struct {
	Path      string
	Recursive bool
	Filter    *regexp.Regexp
	Handler   func(FileEvent)
}

// BufferedReader provides high-performance buffered reading.
type BufferedReader struct {
	reader     io.Reader
	buffer     []byte
	bufferSize int
	pos        int
	limit      int
	eof        bool
	mutex      sync.Mutex
}

// BufferedWriter provides high-performance buffered writing.
type BufferedWriter struct {
	writer     io.Writer
	buffer     []byte
	bufferSize int
	pos        int
	mutex      sync.Mutex
}

// MemoryMappedFile provides memory-mapped file access.
type MemoryMappedFile struct {
	file   *os.File
	data   []byte
	size   int64
	offset int64
	prot   int
	flags  int
	mutex  sync.RWMutex
}

// FileStream provides streaming file operations.
type FileStream struct {
	file     File
	position int64
	size     int64
	mode     FileStreamMode
	buffer   *CircularBuffer
	mutex    sync.RWMutex
}

// FileStreamMode represents file stream modes.
type FileStreamMode int

const (
	StreamRead FileStreamMode = iota
	StreamWrite
	StreamAppend
	StreamReadWrite
)

// CircularBuffer implements a circular buffer for streaming.
type CircularBuffer struct {
	data  []byte
	head  int
	tail  int
	size  int
	mutex sync.RWMutex
}

// FileCompressor provides file compression utilities.
type FileCompressor struct {
	Algorithm  CompressionAlgorithm
	Level      int
	Dictionary []byte
}

// CompressionAlgorithm represents compression algorithms.
type CompressionAlgorithm int

const (
	Gzip CompressionAlgorithm = iota
	Zlib
	Deflate
	LZ4
	Snappy
	Brotli
	LZMA
	Zstd
)

// FileEncryption provides file encryption capabilities.
type FileEncryption struct {
	Algorithm EncryptionAlgorithm
	Key       []byte
	IV        []byte
	Mode      EncryptionMode
}

// EncryptionAlgorithm represents encryption algorithms.
type EncryptionAlgorithm int

const (
	AES256 EncryptionAlgorithm = iota
	ChaCha20
	Blowfish
	Twofish
)

// EncryptionMode represents encryption modes.
type EncryptionMode int

const (
	CBC EncryptionMode = iota
	GCM
	CFB
	OFB
)

// FileSerializer provides serialization utilities.
type FileSerializer struct {
	Format  SerializationFormat
	Options SerializationOptions
}

// SerializationFormat represents serialization formats.
type SerializationFormat int

const (
	JSON SerializationFormat = iota
	XML
	Binary
	GOB
	MessagePack
	Protobuf
	Avro
	Parquet
)

// SerializationOptions represents serialization options.
type SerializationOptions struct {
	Indent    bool
	OmitEmpty bool
	Compress  bool
	Encrypt   bool
	Validate  bool
	SchemaURL string
	Namespace string
}

// FileTransaction provides transactional file operations.
type FileTransaction struct {
	operations []FileOperation
	files      map[string]*TransactionFile
	committed  bool
	rollback   []func() error
	mutex      sync.RWMutex
}

// TransactionFile represents a file in a transaction.
type TransactionFile struct {
	Path         string
	OriginalData []byte
	NewData      []byte
	Operation    FileOperation
	Backup       string
}

// FilePool provides connection pooling for files.
type FilePool struct {
	pool    sync.Pool
	maxSize int
	current int
	mutex   sync.Mutex
}

// AsyncFileOperation represents an asynchronous file operation.
type AsyncFileOperation struct {
	Operation func() (interface{}, error)
	Result    chan AsyncResult
	Context   context.Context
}

// AsyncResult represents the result of an asynchronous operation.
type AsyncResult struct {
	Value interface{}
	Error error
}

// OS returns an FS backed by the host OS filesystem.
func OS() *FS {
	return &FS{
		fs: vfs.NewOS(),
		cache: FileCache{
			enabled: true,
			maxSize: 100 * 1024 * 1024, // 100MB
			entries: make(map[string]*CacheEntry),
			lru:     NewLRUCache(1000),
		},
		watcher: NewFileWatcher(),
	}
}

// Mem returns an in-memory FS useful for tests and ephemeral operations.
func Mem() *FS {
	return &FS{
		fs: vfs.NewMem(),
		cache: FileCache{
			enabled: false,
			entries: make(map[string]*CacheEntry),
		},
	}
}

// Open opens a file.
func (f *FS) Open(name string) (File, error) { return f.fs.Open(name) }

// Create creates or truncates a file.
func (f *FS) Create(name string) (File, error) { return f.fs.Create(name) }

// Mkdir makes a directory.
func (f *FS) Mkdir(name string, perm fs.FileMode) error { return f.fs.Mkdir(name, perm) }

// MkdirAll makes parent directories as needed.
func (f *FS) MkdirAll(name string, perm fs.FileMode) error { return f.fs.MkdirAll(name, perm) }

// Remove removes a file.
func (f *FS) Remove(name string) error { return f.fs.Remove(name) }

// RemoveAll removes a path recursively.
func (f *FS) RemoveAll(name string) error { return f.fs.RemoveAll(name) }

// Stat stats a file.
func (f *FS) Stat(name string) (fs.FileInfo, error) { return f.fs.Stat(name) }

// ReadDir reads a directory.
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) { return f.fs.ReadDir(name) }

// Walk traverses a directory tree.
func (f *FS) Walk(root string, walkFn func(string, fs.FileInfo, error) error) error {
	return filepath.Walk(root, walkFn)
}

// Advanced File Operations

// ReadFile reads the entire file into memory.
func (f *FS) ReadFile(filename string) ([]byte, error) {
	file, err := f.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// WriteFile writes data to a file, creating it if necessary.
func (f *FS) WriteFile(filename string, data []byte, perm FileMode) error {
	file, err := f.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// AppendFile appends data to a file.
func (f *FS) AppendFile(filename string, data []byte) error {
	// Use a simple approach since OpenFile is not available
	existingData, err := f.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	newData := append(existingData, data...)
	return f.WriteFile(filename, newData, 0644)
}

// CopyFile copies a file from src to dst.
func (f *FS) CopyFile(src, dst string) error {
	srcFile, err := f.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := f.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// MoveFile moves a file from src to dst.
func (f *FS) MoveFile(src, dst string) error {
	if err := f.CopyFile(src, dst); err != nil {
		return err
	}
	return f.Remove(src)
}

// File Cache Implementation

// NewLRUCache creates a new LRU cache.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*CacheNode),
	}
}

// Get retrieves a value from the cache.
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if node, exists := c.items[key]; exists {
		c.moveToHead(node)
		return node.value, true
	}
	return nil, false
}

// Put adds a value to the cache.
func (c *LRUCache) Put(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if node, exists := c.items[key]; exists {
		node.value = value
		c.moveToHead(node)
		return
	}

	newNode := &CacheNode{key: key, value: value}
	c.items[key] = newNode
	c.addToHead(newNode)
	c.size++

	if c.size > c.capacity {
		tail := c.removeTail()
		delete(c.items, tail.key)
		c.size--
	}
}

// moveToHead moves a node to the head of the list.
func (c *LRUCache) moveToHead(node *CacheNode) {
	c.removeNode(node)
	c.addToHead(node)
}

// addToHead adds a node to the head of the list.
func (c *LRUCache) addToHead(node *CacheNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

// removeNode removes a node from the list.
func (c *LRUCache) removeNode(node *CacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// removeTail removes the tail node.
func (c *LRUCache) removeTail() *CacheNode {
	lastNode := c.tail.prev
	c.removeNode(lastNode)
	return lastNode
}

// File Watcher Implementation

// NewFileWatcher creates a new file watcher.
func NewFileWatcher() *FileWatcher {
	return &FileWatcher{
		watchers: make(map[string]*Watch),
		events:   make(chan FileEvent, 1000),
		errors:   make(chan error, 100),
	}
}

// AddWatch adds a file or directory to watch.
func (fw *FileWatcher) AddWatch(path string, recursive bool, handler func(FileEvent)) error {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	watch := &Watch{
		Path:      path,
		Recursive: recursive,
		Handler:   handler,
	}

	fw.watchers[path] = watch
	return nil
}

// RemoveWatch removes a watch.
func (fw *FileWatcher) RemoveWatch(path string) error {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	delete(fw.watchers, path)
	return nil
}

// Start starts the file watcher.
func (fw *FileWatcher) Start() error {
	if atomic.CompareAndSwapInt32(&fw.running, 0, 1) {
		go fw.watchLoop()
		return nil
	}
	return fmt.Errorf("watcher already running")
}

// Stop stops the file watcher.
func (fw *FileWatcher) Stop() error {
	if atomic.CompareAndSwapInt32(&fw.running, 1, 0) {
		close(fw.events)
		close(fw.errors)
		return nil
	}
	return fmt.Errorf("watcher not running")
}

// watchLoop is the main watch loop.
func (fw *FileWatcher) watchLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	fileStates := make(map[string]fs.FileInfo)

	for atomic.LoadInt32(&fw.running) == 1 {
		select {
		case <-ticker.C:
			fw.checkFiles(fileStates)
		}
	}
}

// checkFiles checks for file changes.
func (fw *FileWatcher) checkFiles(fileStates map[string]fs.FileInfo) {
	fw.mutex.RLock()
	defer fw.mutex.RUnlock()

	for path, watch := range fw.watchers {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				if oldInfo, exists := fileStates[path]; exists {
					event := FileEvent{
						Path:      path,
						Operation: FileDeleted,
						Time:      time.Now(),
						Size:      oldInfo.Size(),
						Mode:      oldInfo.Mode(),
					}
					fw.events <- event
					if watch.Handler != nil {
						go watch.Handler(event)
					}
					delete(fileStates, path)
				}
			}
			continue
		}

		oldInfo, exists := fileStates[path]
		if !exists {
			event := FileEvent{
				Path:      path,
				Operation: FileCreated,
				Time:      time.Now(),
				Size:      info.Size(),
				Mode:      info.Mode(),
			}
			fw.events <- event
			if watch.Handler != nil {
				go watch.Handler(event)
			}
		} else if info.ModTime().After(oldInfo.ModTime()) {
			event := FileEvent{
				Path:      path,
				Operation: FileModified,
				Time:      time.Now(),
				Size:      info.Size(),
				Mode:      info.Mode(),
			}
			fw.events <- event
			if watch.Handler != nil {
				go watch.Handler(event)
			}
		}

		fileStates[path] = info
	}
}

// Buffered I/O Implementation

// NewBufferedReader creates a new buffered reader.
func NewBufferedReader(reader io.Reader, bufferSize int) *BufferedReader {
	if bufferSize <= 0 {
		bufferSize = 4096
	}

	return &BufferedReader{
		reader:     reader,
		buffer:     make([]byte, bufferSize),
		bufferSize: bufferSize,
	}
}

// Read reads data into p.
func (br *BufferedReader) Read(p []byte) (int, error) {
	br.mutex.Lock()
	defer br.mutex.Unlock()

	if br.pos >= br.limit && !br.eof {
		if err := br.fillBuffer(); err != nil {
			return 0, err
		}
	}

	if br.pos >= br.limit {
		return 0, io.EOF
	}

	n := copy(p, br.buffer[br.pos:br.limit])
	br.pos += n
	return n, nil
}

// fillBuffer fills the internal buffer.
func (br *BufferedReader) fillBuffer() error {
	n, err := br.reader.Read(br.buffer)
	br.pos = 0
	br.limit = n

	if err == io.EOF {
		br.eof = true
	}

	return err
}

// NewBufferedWriter creates a new buffered writer.
func NewBufferedWriter(writer io.Writer, bufferSize int) *BufferedWriter {
	if bufferSize <= 0 {
		bufferSize = 4096
	}

	return &BufferedWriter{
		writer:     writer,
		buffer:     make([]byte, bufferSize),
		bufferSize: bufferSize,
	}
}

// Write writes data from p.
func (bw *BufferedWriter) Write(p []byte) (int, error) {
	bw.mutex.Lock()
	defer bw.mutex.Unlock()

	totalWritten := 0

	for len(p) > 0 {
		available := bw.bufferSize - bw.pos
		if available == 0 {
			if err := bw.flush(); err != nil {
				return totalWritten, err
			}
			available = bw.bufferSize
		}

		toCopy := len(p)
		if toCopy > available {
			toCopy = available
		}

		copy(bw.buffer[bw.pos:], p[:toCopy])
		bw.pos += toCopy
		p = p[toCopy:]
		totalWritten += toCopy
	}

	return totalWritten, nil
}

// Flush flushes the buffer.
func (bw *BufferedWriter) Flush() error {
	bw.mutex.Lock()
	defer bw.mutex.Unlock()
	return bw.flush()
}

// flush internal flush without locking.
func (bw *BufferedWriter) flush() error {
	if bw.pos == 0 {
		return nil
	}

	_, err := bw.writer.Write(bw.buffer[:bw.pos])
	bw.pos = 0
	return err
}

// Memory-Mapped Files

// NewMemoryMappedFile creates a memory-mapped file.
func NewMemoryMappedFile(filename string, size int64, writable bool) (*MemoryMappedFile, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	if writable {
		if err := file.Truncate(size); err != nil {
			file.Close()
			return nil, err
		}
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	// Simplified implementation without platform-specific syscalls
	data := make([]byte, stat.Size())
	_, err = file.ReadAt(data, 0)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &MemoryMappedFile{
		file: file,
		data: data,
		size: stat.Size(),
	}, nil
}

// Read reads data from the memory-mapped file.
func (mmf *MemoryMappedFile) Read(offset int64, length int) ([]byte, error) {
	mmf.mutex.RLock()
	defer mmf.mutex.RUnlock()

	if offset < 0 || offset >= mmf.size {
		return nil, fmt.Errorf("offset out of range")
	}

	end := offset + int64(length)
	if end > mmf.size {
		end = mmf.size
	}

	result := make([]byte, end-offset)
	copy(result, mmf.data[offset:end])
	return result, nil
}

// Write writes data to the memory-mapped file.
func (mmf *MemoryMappedFile) Write(offset int64, data []byte) error {
	mmf.mutex.Lock()
	defer mmf.mutex.Unlock()

	if offset < 0 || offset >= mmf.size {
		return fmt.Errorf("offset out of range")
	}

	end := offset + int64(len(data))
	if end > mmf.size {
		return fmt.Errorf("write beyond file size")
	}

	copy(mmf.data[offset:], data)
	return nil
}

// Close closes the memory-mapped file.
func (mmf *MemoryMappedFile) Close() error {
	mmf.mutex.Lock()
	defer mmf.mutex.Unlock()

	// Simplified close without platform-specific unmapping
	mmf.data = nil

	if mmf.file != nil {
		if err := mmf.file.Close(); err != nil {
			return err
		}
		mmf.file = nil
	}

	return nil
}

// Circular Buffer Implementation

// NewCircularBuffer creates a new circular buffer.
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write writes data to the circular buffer.
func (cb *CircularBuffer) Write(data []byte) (int, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	written := 0
	for _, b := range data {
		cb.data[cb.head] = b
		cb.head = (cb.head + 1) % cb.size

		if cb.head == cb.tail {
			cb.tail = (cb.tail + 1) % cb.size
		}

		written++
	}

	return written, nil
}

// Read reads data from the circular buffer.
func (cb *CircularBuffer) Read(data []byte) (int, error) {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	read := 0
	for i := 0; i < len(data) && cb.tail != cb.head; i++ {
		data[i] = cb.data[cb.tail]
		cb.tail = (cb.tail + 1) % cb.size
		read++
	}

	return read, nil
}

// Len returns the current length of data in the buffer.
func (cb *CircularBuffer) Len() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	if cb.head >= cb.tail {
		return cb.head - cb.tail
	}
	return cb.size - cb.tail + cb.head
}

// File Compression

// NewFileCompressor creates a new file compressor.
func NewFileCompressor(algorithm CompressionAlgorithm, level int) *FileCompressor {
	return &FileCompressor{
		Algorithm: algorithm,
		Level:     level,
	}
}

// CompressFile compresses a file.
func (fc *FileCompressor) CompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	switch fc.Algorithm {
	case Gzip:
		writer, err := gzip.NewWriterLevel(dstFile, fc.Level)
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = io.Copy(writer, srcFile)
		return err

	case Zlib:
		writer, err := zlib.NewWriterLevel(dstFile, fc.Level)
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = io.Copy(writer, srcFile)
		return err

	default:
		return fmt.Errorf("unsupported compression algorithm")
	}
}

// DecompressFile decompresses a file.
func (fc *FileCompressor) DecompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	switch fc.Algorithm {
	case Gzip:
		reader, err := gzip.NewReader(srcFile)
		if err != nil {
			return err
		}
		defer reader.Close()

		_, err = io.Copy(dstFile, reader)
		return err

	case Zlib:
		reader, err := zlib.NewReader(srcFile)
		if err != nil {
			return err
		}
		defer reader.Close()

		_, err = io.Copy(dstFile, reader)
		return err

	default:
		return fmt.Errorf("unsupported compression algorithm")
	}
}

// File Serialization

// NewFileSerializer creates a new file serializer.
func NewFileSerializer(format SerializationFormat, options SerializationOptions) *FileSerializer {
	return &FileSerializer{
		Format:  format,
		Options: options,
	}
}

// SerializeToFile serializes an object to a file.
func (fs *FileSerializer) SerializeToFile(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var writer io.Writer = file

	if fs.Options.Compress {
		gzipWriter := gzip.NewWriter(file)
		defer gzipWriter.Close()
		writer = gzipWriter
	}

	switch fs.Format {
	case JSON:
		encoder := json.NewEncoder(writer)
		if fs.Options.Indent {
			encoder.SetIndent("", "  ")
		}
		return encoder.Encode(data)

	case XML:
		encoder := xml.NewEncoder(writer)
		if fs.Options.Indent {
			encoder.Indent("", "  ")
		}
		return encoder.Encode(data)

	case GOB:
		encoder := gob.NewEncoder(writer)
		return encoder.Encode(data)

	case Binary:
		return binary.Write(writer, binary.LittleEndian, data)

	default:
		return fmt.Errorf("unsupported serialization format")
	}
}

// DeserializeFromFile deserializes an object from a file.
func (fs *FileSerializer) DeserializeFromFile(filename string, data interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file

	if fs.Options.Compress {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	switch fs.Format {
	case JSON:
		decoder := json.NewDecoder(reader)
		return decoder.Decode(data)

	case XML:
		decoder := xml.NewDecoder(reader)
		return decoder.Decode(data)

	case GOB:
		decoder := gob.NewDecoder(reader)
		return decoder.Decode(data)

	case Binary:
		return binary.Read(reader, binary.LittleEndian, data)

	default:
		return fmt.Errorf("unsupported serialization format")
	}
}

// Async File Operations

// ReadFileAsync reads a file asynchronously.
func ReadFileAsync(ctx context.Context, filename string) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		data, err := os.ReadFile(filename)
		select {
		case result <- AsyncResult{Value: data, Error: err}:
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
		}
	}()

	return result
}

// WriteFileAsync writes a file asynchronously.
func WriteFileAsync(ctx context.Context, filename string, data []byte) <-chan AsyncResult {
	result := make(chan AsyncResult, 1)

	go func() {
		defer close(result)

		err := os.WriteFile(filename, data, 0644)
		select {
		case result <- AsyncResult{Error: err}:
		case <-ctx.Done():
			result <- AsyncResult{Error: ctx.Err()}
		}
	}()

	return result
}

// File Transaction Implementation

// NewFileTransaction creates a new file transaction.
func NewFileTransaction() *FileTransaction {
	return &FileTransaction{
		operations: make([]FileOperation, 0),
		files:      make(map[string]*TransactionFile),
	}
}

// AddOperation adds an operation to the transaction.
func (ft *FileTransaction) AddOperation(path string, operation FileOperation, data []byte) error {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	if ft.committed {
		return fmt.Errorf("transaction already committed")
	}

	// Read original data for rollback
	var originalData []byte
	if _, err := os.Stat(path); err == nil {
		originalData, _ = os.ReadFile(path)
	}

	ft.files[path] = &TransactionFile{
		Path:         path,
		OriginalData: originalData,
		NewData:      data,
		Operation:    operation,
	}

	return nil
}

// Commit commits the transaction.
func (ft *FileTransaction) Commit() error {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	if ft.committed {
		return fmt.Errorf("transaction already committed")
	}

	// Execute all operations
	for _, file := range ft.files {
		switch file.Operation {
		case FileCreated, FileModified:
			if err := os.WriteFile(file.Path, file.NewData, 0644); err != nil {
				ft.rollbackUnsafe()
				return err
			}
		case FileDeleted:
			if err := os.Remove(file.Path); err != nil {
				ft.rollbackUnsafe()
				return err
			}
		}
	}

	ft.committed = true
	return nil
}

// Rollback rolls back the transaction.
func (ft *FileTransaction) Rollback() error {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	return ft.rollbackUnsafe()
}

// rollbackUnsafe performs rollback without locking.
func (ft *FileTransaction) rollbackUnsafe() error {
	for _, file := range ft.files {
		if file.OriginalData != nil {
			os.WriteFile(file.Path, file.OriginalData, 0644)
		} else {
			os.Remove(file.Path)
		}
	}
	return nil
}

// Utility functions

// AdvancedFileOperations provide additional file utilities.
type AdvancedFileOperations struct {
	fs *FS
}

// NewAdvancedFileOperations creates advanced file operations.
func NewAdvancedFileOperations(fs *FS) *AdvancedFileOperations {
	return &AdvancedFileOperations{fs: fs}
}

// FindFiles recursively finds files matching a pattern.
func (afo *AdvancedFileOperations) FindFiles(root, pattern string) ([]string, error) {
	var matches []string

	err := afo.fs.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if matched {
				matches = append(matches, path)
			}
		}

		return nil
	})

	return matches, err
}

// DirSize calculates the total size of a directory.
func (afo *AdvancedFileOperations) DirSize(path string) (int64, error) {
	var size int64

	err := afo.fs.Walk(path, func(_ string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	return size, err
}

// CreatePath ensures all directories in the path exist.
func (afo *AdvancedFileOperations) CreatePath(path string) error {
	return afo.fs.MkdirAll(filepath.Dir(path), 0755)
}

// Path utility functions

// Join joins path elements.
func Join(elem ...string) string {
	return vfs.Join(elem...)
}

// Clean returns the shortest path name equivalent to path.
func Clean(p string) string {
	return vfs.Clean(p)
}

// Exists returns true if path can be stat'ed.
func (f *FS) Exists(name string) bool {
	_, err := f.fs.Stat(name)
	return err == nil
}

// FileWatcher wrapper functions

// NewWatcher creates a portable polling watcher on top of fs.
func NewWatcher(f *FS) *vfs.SimpleWatcher {
	return vfs.NewSimpleWatcher(f.fs)
}

// StartPolling starts the watcher.
func StartPolling(w *vfs.SimpleWatcher, ctx context.Context, path string, interval time.Duration) error {
	return w.StartPolling(ctx, path, interval)
}
