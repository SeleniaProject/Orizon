package packagemanager

import (
	"context"
	"testing"
)

func TestManager_ResolveAndFetch(t *testing.T) {
	r1 := NewInMemoryRegistry()
	r2 := NewInMemoryRegistry()
	r1.ConnectPeers(r2)

	// publish packages
	ctx := context.Background()
	// B versions
	if _, err := r1.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "B", Version: "1.0.0"}, Data: []byte("B-1.0.0")}); err != nil {
		t.Fatal(err)
	}
	if _, err := r2.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "B", Version: "1.2.0"}, Data: []byte("B-1.2.0")}); err != nil {
		t.Fatal(err)
	}
	// A depends on B >=1.1.0
	if _, err := r1.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "A", Version: "1.0.0", Dependencies: []Dependency{{Name: "B", Constraint: ">=1.1.0, <2.0.0"}}}, Data: []byte("A-1.0.0")}); err != nil {
		t.Fatal(err)
	}

	m := NewManager(r1)
	out, err := m.ResolveAndFetch(ctx, []Requirement{{Name: "A", Constraint: ">=1.0.0"}}, true)
	if err != nil {
		t.Fatalf("ResolveAndFetch failed: %v", err)
	}
	if got := out["A"].Version; got != "1.0.0" {
		t.Fatalf("want A@1.0.0, got %s", got)
	}
	if got := out["B"].Version; got != "1.2.0" {
		t.Fatalf("want B@1.2.0, got %s", got)
	}
	if out["A"].CID == "" || out["B"].CID == "" {
		t.Fatalf("missing CIDs: %+v", out)
	}
}
