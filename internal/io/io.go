// Package io provides basic I/O primitives for the Orizon runtime.
// This implements the minimal I/O functionality required for self-hosting,
// including file operations, standard I/O, and basic error handling.
package io

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/orizon-lang/orizon/internal/allocator"
)

// IOError represents different types of I/O errors
type IOError int

const (
	IOErrorNone IOError = iota
	IOErrorFileNotFound
	IOErrorPermissionDenied
	IOErrorInvalidPath
	IOErrorOutOfSpace
	IOErrorCorrupted
	IOErrorTimeout
	IOErrorInterrupted
	IOErrorUnknown
)

// String returns the string representation of an IOError
func (e IOError) String() string {
	switch e {
	case IOErrorNone:
		return "no error"
	case IOErrorFileNotFound:
		return "file not found"
	case IOErrorPermissionDenied:
		return "permission denied"
	case IOErrorInvalidPath:
		return "invalid path"
	case IOErrorOutOfSpace:
		return "out of disk space"
	case IOErrorCorrupted:
		return "file corrupted"
	case IOErrorTimeout:
		return "operation timed out"
	case IOErrorInterrupted:
		return "operation interrupted"
	case IOErrorUnknown:
		return "unknown error"
	default:
		return "undefined error"
	}
}

// IOResult represents the result of an I/O operation
type IOResult struct {
	BytesProcessed int
	Error          IOError
	SystemError    string
}

// IsSuccess returns true if the operation succeeded
func (r *IOResult) IsSuccess() bool {
	return r.Error == IOErrorNone
}

// FileHandle represents a handle to an open file
type FileHandle struct {
	id       uint64
	path     string
	file     *os.File
	mode     FileMode
	position int64
	size     int64
	mu       sync.RWMutex
}

// FileMode represents file access mode
type FileMode int

const (
	FileModeReadOnly FileMode = iota
	FileModeWriteOnly
	FileModeReadWrite
	FileModeAppend
	FileModeCreate
	FileModeCreateExclusive
)

// String returns the string representation of FileMode
func (m FileMode) String() string {
	switch m {
	case FileModeReadOnly:
		return "read-only"
	case FileModeWriteOnly:
		return "write-only"
	case FileModeReadWrite:
		return "read-write"
	case FileModeAppend:
		return "append"
	case FileModeCreate:
		return "create"
	case FileModeCreateExclusive:
		return "create-exclusive"
	default:
		return "unknown"
	}
}

// toGoFileMode converts FileMode to Go's file mode
func (m FileMode) toGoFileMode() (int, error) {
	switch m {
	case FileModeReadOnly:
		return os.O_RDONLY, nil
	case FileModeWriteOnly:
		return os.O_WRONLY, nil
	case FileModeReadWrite:
		return os.O_RDWR, nil
	case FileModeAppend:
		return os.O_WRONLY | os.O_APPEND, nil
	case FileModeCreate:
		return os.O_RDWR | os.O_CREATE | os.O_TRUNC, nil
	case FileModeCreateExclusive:
		return os.O_RDWR | os.O_CREATE | os.O_EXCL, nil
	default:
		return 0, fmt.Errorf("unsupported file mode: %v", m)
	}
}

// IOManager manages all I/O operations for the Orizon runtime
type IOManager struct {
	mu                sync.RWMutex
	nextHandleID      uint64
	openFiles         map[uint64]*FileHandle
	stats             IOStats
	allocator         allocator.Allocator
	tempDir           string
	maxOpenFiles      int
	defaultBufferSize int
}

// IOStats provides I/O operation statistics
type IOStats struct {
	FilesOpened     uint64
	FilesClosed     uint64
	BytesRead       uint64
	BytesWritten    uint64
	ReadOperations  uint64
	WriteOperations uint64
	ErrorCount      uint64
	OpenFiles       int32
}

// GlobalIOManager is the global I/O manager instance
var GlobalIOManager *IOManager

// InitializeIO initializes the global I/O manager
func InitializeIO(allocator allocator.Allocator, options ...IOOption) error {
	if allocator == nil {
		return fmt.Errorf("allocator cannot be nil")
	}

	manager := &IOManager{
		openFiles:         make(map[uint64]*FileHandle),
		allocator:         allocator,
		tempDir:           os.TempDir(),
		maxOpenFiles:      1024,
		defaultBufferSize: 8192,
	}

	// Apply options
	for _, opt := range options {
		opt(manager)
	}

	GlobalIOManager = manager
	return nil
}

// IOOption configures the I/O manager
type IOOption func(*IOManager)

// WithTempDir sets the temporary directory
func WithTempDir(dir string) IOOption {
	return func(m *IOManager) { m.tempDir = dir }
}

// WithMaxOpenFiles sets the maximum number of open files
func WithMaxOpenFiles(max int) IOOption {
	return func(m *IOManager) { m.maxOpenFiles = max }
}

// WithBufferSize sets the default buffer size
func WithBufferSize(size int) IOOption {
	return func(m *IOManager) { m.defaultBufferSize = size }
}

// OpenFile opens a file with the specified mode
func (iom *IOManager) OpenFile(path string, mode FileMode) (*FileHandle, IOResult) {
	iom.mu.Lock()
	defer iom.mu.Unlock()

	// Check file limit
	if len(iom.openFiles) >= iom.maxOpenFiles {
		return nil, IOResult{
			Error:       IOErrorOutOfSpace,
			SystemError: "too many open files",
		}
	}

	// Convert file mode
	goMode, err := mode.toGoFileMode()
	if err != nil {
		return nil, IOResult{
			Error:       IOErrorInvalidPath,
			SystemError: err.Error(),
		}
	}

	// Open the file
	file, err := os.OpenFile(path, goMode, 0644)
	if err != nil {
		return nil, IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	// Get file size
	stat, err := file.Stat()
	var size int64
	if err == nil {
		size = stat.Size()
	}

	// Create file handle
	handleID := atomic.AddUint64(&iom.nextHandleID, 1)
	handle := &FileHandle{
		id:       handleID,
		path:     path,
		file:     file,
		mode:     mode,
		position: 0,
		size:     size,
	}

	// Store handle
	iom.openFiles[handleID] = handle

	// Update statistics
	atomic.AddUint64(&iom.stats.FilesOpened, 1)
	atomic.AddInt32(&iom.stats.OpenFiles, 1)

	return handle, IOResult{Error: IOErrorNone}
}

// CloseFile closes a file handle
func (iom *IOManager) CloseFile(handle *FileHandle) IOResult {
	if handle == nil {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "nil handle"}
	}

	iom.mu.Lock()
	defer iom.mu.Unlock()

	// Remove from open files
	delete(iom.openFiles, handle.id)

	// Close the underlying file
	handle.mu.Lock()
	var err error
	if handle.file != nil {
		err = handle.file.Close()
		handle.file = nil
	}
	handle.mu.Unlock()

	// Update statistics
	atomic.AddUint64(&iom.stats.FilesClosed, 1)
	atomic.AddInt32(&iom.stats.OpenFiles, -1)

	if err != nil {
		return IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	return IOResult{Error: IOErrorNone}
}

// ReadFile reads data from a file handle
func (iom *IOManager) ReadFile(handle *FileHandle, buffer unsafe.Pointer, size int) IOResult {
	if handle == nil {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "nil handle"}
	}

	if buffer == nil || size <= 0 {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid buffer"}
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.file == nil {
		return IOResult{Error: IOErrorCorrupted, SystemError: "file not open"}
	}

	// Create Go slice from unsafe pointer
	data := (*[1 << 30]byte)(buffer)[:size:size]

	// Read from file
	n, err := handle.file.Read(data)

	// Update position
	handle.position += int64(n)

	// Update statistics
	atomic.AddUint64(&iom.stats.BytesRead, uint64(n))
	atomic.AddUint64(&iom.stats.ReadOperations, 1)

	if err != nil {
		return IOResult{
			BytesProcessed: n,
			Error:          mapSystemError(err),
			SystemError:    err.Error(),
		}
	}

	return IOResult{
		BytesProcessed: n,
		Error:          IOErrorNone,
	}
}

// WriteFile writes data to a file handle
func (iom *IOManager) WriteFile(handle *FileHandle, buffer unsafe.Pointer, size int) IOResult {
	if handle == nil {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "nil handle"}
	}

	if buffer == nil || size <= 0 {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid buffer"}
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.file == nil {
		return IOResult{Error: IOErrorCorrupted, SystemError: "file not open"}
	}

	// Create Go slice from unsafe pointer
	data := (*[1 << 30]byte)(buffer)[:size:size]

	// Write to file
	n, err := handle.file.Write(data)

	// Update position and size
	handle.position += int64(n)
	if handle.position > handle.size {
		handle.size = handle.position
	}

	// Update statistics
	atomic.AddUint64(&iom.stats.BytesWritten, uint64(n))
	atomic.AddUint64(&iom.stats.WriteOperations, 1)

	if err != nil {
		return IOResult{
			BytesProcessed: n,
			Error:          mapSystemError(err),
			SystemError:    err.Error(),
		}
	}

	return IOResult{
		BytesProcessed: n,
		Error:          IOErrorNone,
	}
}

// SeekFile sets the file position
func (iom *IOManager) SeekFile(handle *FileHandle, offset int64, whence int) IOResult {
	if handle == nil {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "nil handle"}
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.file == nil {
		return IOResult{Error: IOErrorCorrupted, SystemError: "file not open"}
	}

	newPos, err := handle.file.Seek(offset, whence)
	if err != nil {
		return IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	handle.position = newPos

	return IOResult{
		BytesProcessed: int(newPos),
		Error:          IOErrorNone,
	}
}

// FlushFile ensures all data is written to storage
func (iom *IOManager) FlushFile(handle *FileHandle) IOResult {
	if handle == nil {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "nil handle"}
	}

	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.file == nil {
		return IOResult{Error: IOErrorCorrupted, SystemError: "file not open"}
	}

	err := handle.file.Sync()
	if err != nil {
		return IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	return IOResult{Error: IOErrorNone}
}

// GetFileInfo returns information about a file
func (iom *IOManager) GetFileInfo(path string) (FileInfo, IOResult) {
	stat, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	info := FileInfo{
		Path:     path,
		Size:     stat.Size(),
		IsDir:    stat.IsDir(),
		Modified: stat.ModTime(),
		Mode:     int(stat.Mode()),
	}

	return info, IOResult{Error: IOErrorNone}
}

// FileInfo represents file information
type FileInfo struct {
	Path     string
	Size     int64
	IsDir    bool
	Modified time.Time
	Mode     int
}

// DeleteFile deletes a file
func (iom *IOManager) DeleteFile(path string) IOResult {
	err := os.Remove(path)
	if err != nil {
		return IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	return IOResult{Error: IOErrorNone}
}

// CreateDirectory creates a directory
func (iom *IOManager) CreateDirectory(path string) IOResult {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	return IOResult{Error: IOErrorNone}
}

// ListDirectory lists files in a directory
func (iom *IOManager) ListDirectory(path string) ([]FileInfo, IOResult) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't read
		}

		fileInfo := FileInfo{
			Path:     filepath.Join(path, entry.Name()),
			Size:     info.Size(),
			IsDir:    entry.IsDir(),
			Modified: info.ModTime(),
			Mode:     int(info.Mode()),
		}
		files = append(files, fileInfo)
	}

	return files, IOResult{Error: IOErrorNone}
}

// GetStats returns I/O statistics
func (iom *IOManager) GetStats() IOStats {
	iom.mu.RLock()
	defer iom.mu.RUnlock()

	stats := iom.stats
	stats.OpenFiles = atomic.LoadInt32(&iom.stats.OpenFiles)
	return stats
}

// Shutdown closes all open files and cleans up resources
func (iom *IOManager) Shutdown() IOResult {
	iom.mu.Lock()
	defer iom.mu.Unlock()

	var lastError IOResult = IOResult{Error: IOErrorNone}

	// Close all open files
	for _, handle := range iom.openFiles {
		if result := iom.CloseFile(handle); !result.IsSuccess() {
			lastError = result
		}
	}

	// Clear the map
	iom.openFiles = make(map[uint64]*FileHandle)

	return lastError
}

// mapSystemError maps Go system errors to IOError
func mapSystemError(err error) IOError {
	if err == nil {
		return IOErrorNone
	}

	switch {
	case os.IsNotExist(err):
		return IOErrorFileNotFound
	case os.IsPermission(err):
		return IOErrorPermissionDenied
	case os.IsTimeout(err):
		return IOErrorTimeout
	default:
		return IOErrorUnknown
	}
}

// Global convenience functions

// OpenFile opens a file using the global I/O manager
func OpenFile(path string, mode FileMode) (*FileHandle, IOResult) {
	if GlobalIOManager == nil {
		return nil, IOResult{Error: IOErrorUnknown, SystemError: "I/O manager not initialized"}
	}
	return GlobalIOManager.OpenFile(path, mode)
}

// CloseFile closes a file using the global I/O manager
func CloseFile(handle *FileHandle) IOResult {
	if GlobalIOManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "I/O manager not initialized"}
	}
	return GlobalIOManager.CloseFile(handle)
}

// ReadFile reads from a file using the global I/O manager
func ReadFile(handle *FileHandle, buffer unsafe.Pointer, size int) IOResult {
	if GlobalIOManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "I/O manager not initialized"}
	}
	return GlobalIOManager.ReadFile(handle, buffer, size)
}

// WriteFile writes to a file using the global I/O manager
func WriteFile(handle *FileHandle, buffer unsafe.Pointer, size int) IOResult {
	if GlobalIOManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "I/O manager not initialized"}
	}
	return GlobalIOManager.WriteFile(handle, buffer, size)
}

// GetFileInfo gets file information using the global I/O manager
func GetFileInfo(path string) (FileInfo, IOResult) {
	if GlobalIOManager == nil {
		return FileInfo{}, IOResult{Error: IOErrorUnknown, SystemError: "I/O manager not initialized"}
	}
	return GlobalIOManager.GetFileInfo(path)
}

// GetIOStats returns global I/O statistics
func GetIOStats() IOStats {
	if GlobalIOManager == nil {
		return IOStats{}
	}
	return GlobalIOManager.GetStats()
}

// ShutdownIO shuts down the global I/O manager
func ShutdownIO() IOResult {
	if GlobalIOManager == nil {
		return IOResult{Error: IOErrorNone}
	}
	return GlobalIOManager.Shutdown()
}
