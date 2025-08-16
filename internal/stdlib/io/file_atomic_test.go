package io

import (
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	fs := OS()
	path := "./.orizon/tmp/test_atomic.txt"
	data := []byte("hello")
	if err := fs.AtomicWrite(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("%q", string(b))
	}
}

func TestWithTempFile(t *testing.T) {
	fs := OS()
	dir := "./.orizon/tmp"
	seen := ""
	err := fs.WithTempFile(dir, "tmp", func(p string) error {
		seen = p
		return fs.WriteFile(p, []byte("x"), 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen == "" {
		t.Fatal("no path")
	}
	if fs.Exists(seen) {
		t.Fatal("temp not removed")
	}
}
