package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempFile(t *testing.T, dir, name, contents string) string {
	t.Helper()

	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	return p
}

func TestIncremental_SnapshotAndDiff(t *testing.T) {
	dir := t.TempDir()
	a := writeTempFile(t, dir, "a.txt", "hello")
	b := writeTempFile(t, dir, "b.txt", "world")
	ie := NewIncrementalEngine()

	snap1, err := ie.SnapshotInputs(map[TargetID][]string{
		"T1": {filepath.Join(dir, "*.txt")},
	})
	if err != nil {
		t.Fatal(err)
	}

	// No changes.
	snap2, err := ie.SnapshotInputs(map[TargetID][]string{
		"T1": {filepath.Join(dir, "*.txt")},
	})
	if err != nil {
		t.Fatal(err)
	}

	dirty, err := ie.Diff(snap1, snap2)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirty) != 0 {
		t.Fatalf("expected no dirty, got %v", dirty)
	}

	// Change file.
	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(a, []byte("HELLO"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap3, err := ie.SnapshotInputs(map[TargetID][]string{
		"T1": {filepath.Join(dir, "*.txt")},
	})
	if err != nil {
		t.Fatal(err)
	}

	dirty, err = ie.Diff(snap2, snap3)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirty) != 1 || dirty[0] != "T1" {
		t.Fatalf("expected T1 dirty, got %v", dirty)
	}

	// Add file.
	_ = writeTempFile(t, dir, "c.txt", "new")

	snap4, err := ie.SnapshotInputs(map[TargetID][]string{
		"T1": {filepath.Join(dir, "*.txt")},
	})
	if err != nil {
		t.Fatal(err)
	}

	dirty, err = ie.Diff(snap3, snap4)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirty) != 1 || dirty[0] != "T1" {
		t.Fatalf("expected T1 dirty on add, got %v", dirty)
	}

	// Remove one file.
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}

	snap5, err := ie.SnapshotInputs(map[TargetID][]string{
		"T1": {filepath.Join(dir, "*.txt")},
	})
	if err != nil {
		t.Fatal(err)
	}

	dirty, err = ie.Diff(snap4, snap5)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirty) != 1 || dirty[0] != "T1" {
		t.Fatalf("expected T1 dirty on remove, got %v", dirty)
	}
}
