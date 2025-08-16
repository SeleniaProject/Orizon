package packagemanager

import (
	"context"
	"testing"

	semver "github.com/Masterminds/semver/v3"
)

func TestInMemoryRegistry_ReplicationAndFetch(t *testing.T) {
	r1 := NewInMemoryRegistry()
	r2 := NewInMemoryRegistry()
	r3 := NewInMemoryRegistry()
	r1.ConnectPeers(r2, r3)

	blob := PackageBlob{
		Manifest: PackageManifest{
			Name:    "pkgA",
			Version: "1.0.0",
		},
		Data: []byte("hello world"),
	}
	ctx := context.Background()
	cid, err := r1.Publish(ctx, blob)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// r2 and r3 should be able to fetch via replication
	if got, err := r2.Fetch(ctx, cid); err != nil {
		t.Fatalf("r2 fetch failed: %v", err)
	} else if string(got.Data) != "hello world" {
		t.Fatalf("unexpected data: %q", string(got.Data))
	}
	if got, err := r3.Fetch(ctx, cid); err != nil {
		t.Fatalf("r3 fetch failed: %v", err)
	} else if string(got.Data) != "hello world" {
		t.Fatalf("unexpected data: %q", string(got.Data))
	}
}

func TestInMemoryRegistry_FindByConstraint(t *testing.T) {
	r1 := NewInMemoryRegistry()
	r2 := NewInMemoryRegistry()
	r1.ConnectPeers(r2)

	ctx := context.Background()
	// publish two versions across peers
	if _, err := r1.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "pkgB", Version: "1.1.0"}, Data: []byte("v1.1.0")}); err != nil {
		t.Fatal(err)
	}
	if _, err := r2.Publish(ctx, PackageBlob{Manifest: PackageManifest{Name: "pkgB", Version: "1.3.0"}, Data: []byte("v1.3.0")}); err != nil {
		t.Fatal(err)
	}

	c, _ := semver.NewConstraint(">=1.0.0, <2.0.0")
	cid, man, err := r1.Find(ctx, "pkgB", c)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if man.Version != "1.3.0" {
		t.Fatalf("expected highest 1.3.0, got %s", man.Version)
	}

	// ensure we can fetch what Find returned
	_, err = r1.Fetch(ctx, cid)
	if err != nil {
		t.Fatalf("fetch after find failed: %v", err)
	}
}
