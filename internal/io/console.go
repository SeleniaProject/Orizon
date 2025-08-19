// Package io provides standard I/O primitives for console and stream operations.
// This implements stdin, stdout, and stderr handling for the Orizon runtime.
package io

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

// StandardStream represents a standard I/O stream
type StandardStream int

const (
	Stdin StandardStream = iota
	Stdout
	Stderr
)

// String returns the string representation of StandardStream
func (s StandardStream) String() string {
	switch s {
	case Stdin:
		return "stdin"
	case Stdout:
		return "stdout"
	case Stderr:
		return "stderr"
	default:
		return "unknown"
	}
}

// ConsoleHandle represents a handle to a console stream
type ConsoleHandle struct {
	stream StandardStream
	file   *os.File
	mu     sync.RWMutex
}

// ConsoleManager manages console I/O operations
type ConsoleManager struct {
	stdin  *ConsoleHandle
	stdout *ConsoleHandle
	stderr *ConsoleHandle
	mu     sync.RWMutex
	stats  ConsoleStats
}

// ConsoleStats provides console I/O statistics
type ConsoleStats struct {
	StdinBytesRead     uint64
	StdoutBytesWritten uint64
	StderrBytesWritten uint64
	ReadOperations     uint64
	WriteOperations    uint64
	FlushOperations    uint64
	ErrorCount         uint64
}

// GlobalConsoleManager is the global console manager instance
var GlobalConsoleManager *ConsoleManager

// InitializeConsole initializes the global console manager
func InitializeConsole() error {
	manager := &ConsoleManager{
		stdin: &ConsoleHandle{
			stream: Stdin,
			file:   os.Stdin,
		},
		stdout: &ConsoleHandle{
			stream: Stdout,
			file:   os.Stdout,
		},
		stderr: &ConsoleHandle{
			stream: Stderr,
			file:   os.Stderr,
		},
	}

	GlobalConsoleManager = manager
	return nil
}

// ReadConsole reads data from stdin
func (cm *ConsoleManager) ReadConsole(buffer unsafe.Pointer, size int) IOResult {
	if buffer == nil || size <= 0 {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid buffer"}
	}

	cm.stdin.mu.Lock()
	defer cm.stdin.mu.Unlock()

	// Create Go slice from unsafe pointer
	data := (*[1 << 30]byte)(buffer)[:size:size]

	// Read from stdin
	n, err := cm.stdin.file.Read(data)

	// Update statistics
	atomic.AddUint64(&cm.stats.StdinBytesRead, uint64(n))
	atomic.AddUint64(&cm.stats.ReadOperations, 1)

	if err != nil {
		atomic.AddUint64(&cm.stats.ErrorCount, 1)
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

// WriteConsole writes data to stdout or stderr
func (cm *ConsoleManager) WriteConsole(stream StandardStream, buffer unsafe.Pointer, size int) IOResult {
	if buffer == nil || size <= 0 {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid buffer"}
	}

	var handle *ConsoleHandle
	switch stream {
	case Stdout:
		handle = cm.stdout
	case Stderr:
		handle = cm.stderr
	default:
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid stream for writing"}
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	// Create Go slice from unsafe pointer
	data := (*[1 << 30]byte)(buffer)[:size:size]

	// Write to stream
	n, err := handle.file.Write(data)

	// Update statistics
	atomic.AddUint64(&cm.stats.WriteOperations, 1)
	if stream == Stdout {
		atomic.AddUint64(&cm.stats.StdoutBytesWritten, uint64(n))
	} else {
		atomic.AddUint64(&cm.stats.StderrBytesWritten, uint64(n))
	}

	if err != nil {
		atomic.AddUint64(&cm.stats.ErrorCount, 1)
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

// FlushConsole flushes a console stream
func (cm *ConsoleManager) FlushConsole(stream StandardStream) IOResult {
	var handle *ConsoleHandle
	switch stream {
	case Stdout:
		handle = cm.stdout
	case Stderr:
		handle = cm.stderr
	default:
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid stream for flushing"}
	}

	handle.mu.RLock()
	defer handle.mu.RUnlock()

	// Sync the file (best effort for console streams)
	err := handle.file.Sync()

	atomic.AddUint64(&cm.stats.FlushOperations, 1)

	if err != nil {
		atomic.AddUint64(&cm.stats.ErrorCount, 1)
		return IOResult{
			Error:       mapSystemError(err),
			SystemError: err.Error(),
		}
	}

	return IOResult{Error: IOErrorNone}
}

// PrintString prints a string to stdout
func (cm *ConsoleManager) PrintString(str string) IOResult {
	if len(str) == 0 {
		return IOResult{Error: IOErrorNone}
	}

	data := []byte(str)
	return cm.WriteConsole(Stdout, unsafe.Pointer(&data[0]), len(data))
}

// PrintLine prints a string followed by a newline to stdout
func (cm *ConsoleManager) PrintLine(str string) IOResult {
	result := cm.PrintString(str)
	if !result.IsSuccess() {
		return result
	}

	newline := []byte("\n")
	return cm.WriteConsole(Stdout, unsafe.Pointer(&newline[0]), 1)
}

// PrintError prints a string to stderr
func (cm *ConsoleManager) PrintError(str string) IOResult {
	if len(str) == 0 {
		return IOResult{Error: IOErrorNone}
	}

	data := []byte(str)
	return cm.WriteConsole(Stderr, unsafe.Pointer(&data[0]), len(data))
}

// PrintErrorLine prints a string followed by a newline to stderr
func (cm *ConsoleManager) PrintErrorLine(str string) IOResult {
	result := cm.PrintError(str)
	if !result.IsSuccess() {
		return result
	}

	newline := []byte("\n")
	return cm.WriteConsole(Stderr, unsafe.Pointer(&newline[0]), 1)
}

// ReadLine reads a line from stdin (up to newline or buffer size)
func (cm *ConsoleManager) ReadLine(buffer unsafe.Pointer, maxSize int) IOResult {
	if buffer == nil || maxSize <= 0 {
		return IOResult{Error: IOErrorInvalidPath, SystemError: "invalid buffer"}
	}

	data := (*[1 << 30]byte)(buffer)[:maxSize:maxSize]

	totalBytes := 0
	for totalBytes < maxSize-1 { // Leave space for null terminator
		n, err := cm.stdin.file.Read(data[totalBytes : totalBytes+1])
		if err != nil {
			atomic.AddUint64(&cm.stats.ErrorCount, 1)
			return IOResult{
				BytesProcessed: totalBytes,
				Error:          mapSystemError(err),
				SystemError:    err.Error(),
			}
		}

		if n == 0 {
			break // EOF
		}

		totalBytes += n

		// Check for newline
		if data[totalBytes-1] == '\n' {
			break
		}
	}

	// Null-terminate the string
	if totalBytes < maxSize {
		data[totalBytes] = 0
	}

	atomic.AddUint64(&cm.stats.StdinBytesRead, uint64(totalBytes))
	atomic.AddUint64(&cm.stats.ReadOperations, 1)

	return IOResult{
		BytesProcessed: totalBytes,
		Error:          IOErrorNone,
	}
}

// GetConsoleStats returns console I/O statistics
func (cm *ConsoleManager) GetConsoleStats() ConsoleStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.stats
}

// IsTerminal checks if a stream is connected to a terminal
func (cm *ConsoleManager) IsTerminal(stream StandardStream) bool {
	var file *os.File
	switch stream {
	case Stdin:
		file = cm.stdin.file
	case Stdout:
		file = cm.stdout.file
	case Stderr:
		file = cm.stderr.file
	default:
		return false
	}

	// Simple check - in a real implementation, this would use platform-specific APIs
	stat, err := file.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (typical for terminals)
	return stat.Mode()&os.ModeCharDevice != 0
}

// Global convenience functions for console I/O

// ReadConsole reads from stdin using the global console manager
func ReadConsole(buffer unsafe.Pointer, size int) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.ReadConsole(buffer, size)
}

// WriteStdout writes to stdout using the global console manager
func WriteStdout(buffer unsafe.Pointer, size int) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.WriteConsole(Stdout, buffer, size)
}

// WriteStderr writes to stderr using the global console manager
func WriteStderr(buffer unsafe.Pointer, size int) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.WriteConsole(Stderr, buffer, size)
}

// PrintString prints a string to stdout
func PrintString(str string) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.PrintString(str)
}

// PrintLine prints a string with newline to stdout
func PrintLine(str string) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.PrintLine(str)
}

// PrintError prints a string to stderr
func PrintError(str string) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.PrintError(str)
}

// PrintErrorLine prints a string with newline to stderr
func PrintErrorLine(str string) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.PrintErrorLine(str)
}

// ReadLine reads a line from stdin
func ReadLine(buffer unsafe.Pointer, maxSize int) IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.ReadLine(buffer, maxSize)
}

// FlushStdout flushes stdout
func FlushStdout() IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.FlushConsole(Stdout)
}

// FlushStderr flushes stderr
func FlushStderr() IOResult {
	if GlobalConsoleManager == nil {
		return IOResult{Error: IOErrorUnknown, SystemError: "console manager not initialized"}
	}
	return GlobalConsoleManager.FlushConsole(Stderr)
}

// GetConsoleStats returns global console statistics
func GetConsoleStats() ConsoleStats {
	if GlobalConsoleManager == nil {
		return ConsoleStats{}
	}
	return GlobalConsoleManager.GetConsoleStats()
}

// IsTerminal checks if a stream is a terminal
func IsTerminal(stream StandardStream) bool {
	if GlobalConsoleManager == nil {
		return false
	}
	return GlobalConsoleManager.IsTerminal(stream)
}

// Printf prints formatted output to stdout
func Printf(format string, args ...interface{}) IOResult {
	str := fmt.Sprintf(format, args...)
	return PrintString(str)
}

// Eprintf prints formatted output to stderr
func Eprintf(format string, args ...interface{}) IOResult {
	str := fmt.Sprintf(format, args...)
	return PrintError(str)
}
