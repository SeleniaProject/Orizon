//go:build windows.
// +build windows.

package asyncio

import (
	"os"
	"testing"
)

// resetEnv restores relevant variables to empty state after each test.
func resetEnv(vars ...string) {
	for _, k := range vars {
		_ = os.Unsetenv(k)
	}
}

func TestNewOSPoller_DefaultsToPortableOnWindows(t *testing.T) {
	t.Setenv("ORIZON_WIN_IOCP", "0")
	t.Setenv("ORIZON_WIN_WSAPOLL", "0")
	t.Setenv("ORIZON_WIN_PORTABLE", "0")
	p := NewOSPoller()
	if p == nil {
		t.Fatal("NewOSPoller returned nil")
	}
	// By default on Windows we pick the portability-first goroutine poller unless env forces otherwise.
	if _, ok := p.(*goPoller); !ok {
		t.Fatalf("expected *goPoller by default, got %T", p)
	}
}

func TestNewOSPoller_ForcedPortable(t *testing.T) {
	t.Setenv("ORIZON_WIN_PORTABLE", "1")
	defer resetEnv("ORIZON_WIN_PORTABLE")
	p := NewOSPoller()
	if p == nil {
		t.Fatal("NewOSPoller returned nil")
	}
	if _, ok := p.(*goPoller); !ok {
		t.Fatalf("expected *goPoller when ORIZON_WIN_PORTABLE=1, got %T", p)
	}
}

func TestNewOSPoller_ForcedWSAPoll(t *testing.T) {
	t.Setenv("ORIZON_WIN_WSAPOLL", "true")
	defer resetEnv("ORIZON_WIN_WSAPOLL")
	p := NewOSPoller()
	if p == nil {
		t.Fatal("NewOSPoller returned nil")
	}
	if _, ok := p.(*windowsPoller); !ok {
		t.Fatalf("expected *windowsPoller when ORIZON_WIN_WSAPOLL=true, got %T", p)
	}
}

func TestNewOSPoller_ForcedIOCPFallsBackWhenUnavailable(t *testing.T) {
	// Without the 'iocp' build tag, newIOCPIfAvailable returns nil and factory should fall back to WSAPoll.
	t.Setenv("ORIZON_WIN_IOCP", "on")
	defer resetEnv("ORIZON_WIN_IOCP")
	p := NewOSPoller()
	if p == nil {
		t.Fatal("NewOSPoller returned nil")
	}
	// With explicit IOCP request but unavailable, we choose WSAPoll-based poller.
	if _, ok := p.(*windowsPoller); !ok {
		t.Fatalf("expected fallback to *windowsPoller when IOCP unavailable, got %T", p)
	}
}
