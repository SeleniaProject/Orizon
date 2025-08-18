package io

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/allocator"
)

func getTestConfig() *allocator.Config {
	return &allocator.Config{
		ArenaSize: 1024 * 1024, // 1MB
		PoolSizes: []uintptr{64, 256, 1024, 4096},
	}
}

// TestIOManagerInitialization tests basic I/O manager setup
func TestIOManagerInitialization(t *testing.T) {
	// Initialize allocator
	config := getTestConfig()
	alloc := allocator.NewSystemAllocator(config)

	// Initialize I/O manager
	err := InitializeIO(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize I/O manager: %v", err)
	}

	// Verify global manager is set
	if GlobalIOManager == nil {
		t.Fatal("Global I/O manager is nil after initialization")
	}
}

// TestConsoleInitialization tests console initialization
func TestConsoleInitialization(t *testing.T) {
	// Initialize console
	err := InitializeConsole()
	if err != nil {
		t.Fatalf("Failed to initialize console: %v", err)
	}

	// Verify global console manager is set
	if GlobalConsoleManager == nil {
		t.Fatal("Global console manager is nil after initialization")
	}
}

// TestThreadingInitialization tests threading initialization
func TestThreadingInitialization(t *testing.T) {
	// Initialize allocator
	config := getTestConfig()
	alloc := allocator.NewSystemAllocator(config)

	// Initialize threading
	err := InitializeThreading(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize threading: %v", err)
	}
	defer ShutdownThreading()

	// Verify global thread manager is set
	if GlobalThreadManager == nil {
		t.Fatal("Global thread manager is nil after initialization")
	}
}

// TestMutexCreation tests mutex creation
func TestMutexCreation(t *testing.T) {
	// Initialize threading
	config := getTestConfig()
	alloc := allocator.NewSystemAllocator(config)
	err := InitializeThreading(alloc)
	if err != nil {
		t.Fatalf("Failed to initialize threading: %v", err)
	}
	defer ShutdownThreading()

	// Create mutex
	mutex := NewMutex()
	if mutex == nil {
		t.Fatal("Failed to create mutex")
	}

	// Test initial state
	if mutex.IsLocked() {
		t.Error("Mutex should not be locked initially")
	}
}

// TestChannelCreation tests channel creation
func TestChannelCreation(t *testing.T) {
	// Test unbuffered channel
	channel := NewChannel(0)
	if channel == nil {
		t.Fatal("Failed to create channel")
	}

	if channel.Len() != 0 {
		t.Errorf("Expected channel length 0, got %d", channel.Len())
	}

	if channel.Cap() != 0 {
		t.Errorf("Expected channel capacity 0, got %d", channel.Cap())
	}

	// Test buffered channel
	bufferedChannel := NewChannel(5)
	if bufferedChannel == nil {
		t.Fatal("Failed to create buffered channel")
	}

	if bufferedChannel.Cap() != 5 {
		t.Errorf("Expected channel capacity 5, got %d", bufferedChannel.Cap())
	}
}
