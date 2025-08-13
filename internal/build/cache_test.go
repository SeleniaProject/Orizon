package build

import (
    "path/filepath"
    "testing"
)

func TestInMemoryLRUCache_Basic(t *testing.T) {
    c := NewInMemoryLRUCache(2)
    a1 := Artifact{ Files: map[string][]byte{"a": []byte("one")} }
    a2 := Artifact{ Files: map[string][]byte{"b": []byte("two")} }
    a3 := Artifact{ Files: map[string][]byte{"c": []byte("three")} }
    _ = c.Put("k1", a1)
    _ = c.Put("k2", a2)
    if _, ok, _ := c.Get("k1"); !ok { t.Fatalf("expected hit k1") }
    _ = c.Put("k3", a3) // should evict k2
    if _, ok, _ := c.Get("k2"); ok { t.Fatalf("expected eviction of k2") }
}

func TestFSCache_PutGetInvalidate(t *testing.T) {
    dir := t.TempDir()
    fc, err := NewFSCache(filepath.Join(dir, "cache"))
    if err != nil { t.Fatal(err) }
    art := Artifact{ Files: map[string][]byte{"bin": []byte("binary data"), "log": []byte("hello")}, Metadata: map[string]string{"k":"v"} }
    key := CacheKey("mod@v1")
    if err := fc.Put(key, art); err != nil { t.Fatal(err) }
    if !fc.Exists(key) { t.Fatalf("expected key to exist") }
    got, ok, err := fc.Get(key)
    if err != nil { t.Fatal(err) }
    if !ok { t.Fatalf("expected hit") }
    if string(got.Files["bin"]) != "binary data" { t.Fatalf("bad content") }
    if err := fc.Invalidate(key); err != nil { t.Fatal(err) }
    if fc.Exists(key) { t.Fatalf("expected removal") }
}


