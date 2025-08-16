package io

import (
	"io/fs"
	"testing"
)

func TestFS_Mem_ReadWriteCopyMove(t *testing.T) {
	fsys := Mem()
	if fsys.Exists("a.txt") {
		t.Fatal("should not exist")
	}
	if err := fsys.MkdirAll("dir", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fsys.WriteFile("dir/a.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fsys.Exists("dir/a.txt") {
		t.Fatal("a.txt should exist")
	}
	b, err := fsys.ReadFile("dir/a.txt")
	if err != nil || string(b) != "hello" {
		t.Fatalf("read: %v %q", err, string(b))
	}
	if err := fsys.CopyFile("dir/a.txt", "dir/b.txt", fs.FileMode(0o644)); err != nil {
		t.Fatal(err)
	}
	if err := fsys.Move("dir/b.txt", "dir/c.txt"); err != nil {
		t.Fatal(err)
	}
	if fsys.Exists("dir/b.txt") {
		t.Fatal("b.txt should be removed")
	}
	if !fsys.Exists("dir/c.txt") {
		t.Fatal("c.txt should exist")
	}
}
