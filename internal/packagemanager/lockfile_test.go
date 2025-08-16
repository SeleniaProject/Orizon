package packagemanager

import (
	"context"
	"testing"
)

func TestLockfile_GenerateAndVerify(t *testing.T) {
	reg := NewInMemoryRegistry()
	ctx := context.Background()

	// Publish B then A (A depends on B)
	if _, err := reg.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "B", Version: "1.2.3"}, Data: []byte("B-1.2.3")}); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "A", Version: "0.1.0", Dependencies: []Dependency{{Name: "B", Constraint: "^1.2.0"}}}, Data: []byte("A-0.1.0")}); err != nil {
		t.Fatal(err)
	}

	m := NewManager(reg)
	res, err := m.ResolveAndFetch(ctx, []Requirement{{Name: "A", Constraint: ">=0.1.0"}}, true)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	// Convert to Resolution shape
	rr := make(Resolution)
	for name, pinned := range res {
		rr[name] = pinned.Version
	}

	lock, _, err := GenerateLockfile(ctx, reg, rr)
	if err != nil {
		t.Fatalf("generate lock failed: %v", err)
	}
	if err := VerifyLockfile(ctx, reg, lock); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}
