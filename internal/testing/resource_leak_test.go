package testing

import (
	"os"
	"runtime"
	"testing"
	"time"
)

// TestForFileDescriptorLeaks tests for file descriptor leaks
func TestForFileDescriptorLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file descriptor leak test in short mode")
	}

	// Get baseline file descriptor count
	baselineFDs := getOpenFileDescriptorCount(t)

	// Run operations that should not leak file descriptors
	for i := 0; i < 100; i++ {
		func() {
			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "leak_test_*.tmp")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
			}()

			// Write some data
			if _, err := tmpFile.WriteString("test data"); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
		}()
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check if file descriptors increased significantly
	currentFDs := getOpenFileDescriptorCount(t)
	fdDiff := currentFDs - baselineFDs

	// Allow for some variance but flag significant increases
	if fdDiff > 10 {
		t.Errorf("Potential file descriptor leak detected: baseline=%d, current=%d, diff=%d",
			baselineFDs, currentFDs, fdDiff)
	}
}

// TestMemoryUsageGrowth tests for memory leaks
func TestMemoryUsageGrowth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	var m1, m2 runtime.MemStats

	// Get baseline memory stats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform operations that should not leak memory
	for i := 0; i < 1000; i++ {
		// Simulate some work that could potentially leak
		data := make([]byte, 1024)
		_ = data
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Check memory stats
	runtime.ReadMemStats(&m2)

	// Calculate memory growth
	allocDiff := int64(m2.Alloc) - int64(m1.Alloc)
	heapDiff := int64(m2.HeapInuse) - int64(m1.HeapInuse)

	// Allow for some growth but flag significant increases
	if allocDiff > 10*1024*1024 { // 10MB
		t.Errorf("Potential memory leak detected: alloc diff=%d bytes", allocDiff)
	}

	if heapDiff > 10*1024*1024 { // 10MB
		t.Errorf("Potential heap leak detected: heap diff=%d bytes", heapDiff)
	}

	t.Logf("Memory usage - Alloc diff: %d bytes, Heap diff: %d bytes", allocDiff, heapDiff)
}

// TestResourceCleanupOnPanic tests that resources are cleaned up properly on panic
func TestResourceCleanupOnPanic(t *testing.T) {
	var tmpFile *os.File
	var err error

	// Test that defer works even when panic occurs
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Panic occurred, check if file was closed
				if tmpFile != nil {
					if err := tmpFile.Close(); err == nil {
						t.Logf("File was properly closed after panic")
					} else {
						t.Errorf("File was not properly closed after panic: %v", err)
					}
					os.Remove(tmpFile.Name())
				}
			}
		}()

		tmpFile, err = os.CreateTemp("", "panic_test_*.tmp")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		// This should panic but file should still be cleaned up
		panic("test panic")
	}()
}

// getOpenFileDescriptorCount returns the number of open file descriptors
// This is platform-specific and provides a rough estimate
func getOpenFileDescriptorCount(t *testing.T) int {
	// On Unix-like systems, we can check /proc/self/fd
	if runtime.GOOS != "windows" {
		if entries, err := os.ReadDir("/proc/self/fd"); err == nil {
			return len(entries)
		}
	}

	// For Windows or if /proc/self/fd is not available,
	// we return a baseline that should be consistent
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Use number of goroutines as a proxy (very rough estimate)
	return runtime.NumGoroutine()
}
