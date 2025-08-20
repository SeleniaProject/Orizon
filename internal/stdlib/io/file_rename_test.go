package io

import (
	"testing"
)

func TestFS_Rename_File_And_Dir(t *testing.T) {
	fsys := Mem()
	// file rename.
	if err := fsys.MkdirAll("d", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := fsys.WriteFile("d/a.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := fsys.Rename("d/a.txt", "d/b.txt"); err != nil {
		t.Fatal(err)
	}

	if fsys.Exists("d/a.txt") {
		t.Fatal("old file should not exist")
	}

	if !fsys.Exists("d/b.txt") {
		t.Fatal("new file should exist")
	}
	// directory rename should move children.
	if err := fsys.MkdirAll("dir/sub", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := fsys.WriteFile("dir/sub/c.txt", []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := fsys.Rename("dir", "dir2"); err != nil {
		t.Fatal(err)
	}

	if fsys.Exists("dir/sub/c.txt") {
		t.Fatal("old nested path should not exist")
	}

	if !fsys.Exists("dir2/sub/c.txt") {
		t.Fatal("renamed nested path should exist")
	}
}
