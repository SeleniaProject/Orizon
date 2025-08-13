package packagemanager

import (
    "context"
    "testing"
    "time"
)

func TestSignature_EndToEnd(t *testing.T) {
    // Setup trust and keys
    rootPub, rootPriv, err := GenerateEd25519Keypair()
    if err != nil { t.Fatal(err) }
    rootCert, err := SelfSignRoot("Root CA", rootPub, rootPriv, time.Hour)
    if err != nil { t.Fatal(err) }
    ts := NewTrustStore()
    ts.AddRoot(rootPub)

    // Issue leaf
    leafPub, leafPriv, err := GenerateEd25519Keypair()
    if err != nil { t.Fatal(err) }
    leafCert, err := IssueChild(rootCert, rootPriv, leafPub, "Publisher", time.Hour, []string{"package-sign"})
    if err != nil { t.Fatal(err) }
    chain := []Certificate{leafCert, rootCert}

    // Publish a package
    reg := NewInMemoryRegistry()
    ctx := context.Background()
    cid, err := reg.Publish(ctx, PackageBlob{ Manifest: PackageManifest{ Name: "pkg", Version: "1.0.0" }, Data: []byte("content") })
    if err != nil { t.Fatal(err) }

    // Sign and verify
    store := NewInMemorySignatureStore()
    if _, err := SignPackage(ctx, reg, cid, leafPriv, chain, store); err != nil { t.Fatal(err) }
    if err := VerifyPackage(ctx, reg, ts, cid, store); err != nil { t.Fatalf("verify failed: %v", err) }

    // Vulnerability scanner integration
    scanner := NewInMemoryAdvisoryScanner()
    if err := ValidatePackageSecurity(ctx, reg, ts, cid, store, scanner); err != nil { t.Fatalf("unexpected security validation error: %v", err) }
    // Mark vulnerable and expect failure
    scanner.Add("pkg", "1.0.0", "test advisory")
    if err := ValidatePackageSecurity(ctx, reg, ts, cid, store, scanner); err == nil { t.Fatalf("expected vulnerability failure") }
}


