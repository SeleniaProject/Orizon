package vfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOSFS_CreateReadWrite(t *testing.T) {
	fsys := NewOS()
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	f, err := fsys.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := f.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "hello" {
		t.Fatalf("got %q", string(buf))
	}
}

func TestMemFS_ReadDirWalk(t *testing.T) {
	m := NewMem()
	_ = m.MkdirAll("/x/y", 0)
	f, _ := m.Create("/x/y/z.txt")
	_, _ = f.Write([]byte("1"))
	_ = f.Sync()
	ds, err := m.ReadDir("/x")
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) == 0 {
		t.Fatal("expected entries")
	}
	var seen int
	if err := m.Walk("/", func(p string, d os.DirEntry, err error) error { seen++; return nil }); err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("expected walked entries")
	}
}

func TestWatcher_Polling(t *testing.T) {
	fsys := NewOS()
	dir := t.TempDir()
	p := filepath.Join(dir, "w.txt")
	f, err := fsys.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	w := NewSimpleWatcher(fsys)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := w.StartPolling(ctx, p, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	// trigger change
	go func() { _ = os.WriteFile(p, []byte("x"), 0o644) }()
	select {
	case <-w.Events():
		// ok
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestWatcher_FSNotify(t *testing.T) {
    fw, err := NewFSWatcher()
    if err != nil {
        t.Skip("fsnotify not supported: ", err)
    }
    defer fw.Close()
    dir := t.TempDir()
    if err := fw.Add(dir); err != nil { t.Fatal(err) }
    done := make(chan struct{}, 1)
    go func(){
        f := filepath.Join(dir, "f.txt")
        _ = os.WriteFile(f, []byte("x"), 0o644)
    }()
    select {
    case ev := <-fw.Events():
        if ev.Path == "" { t.Fatal("empty path") }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for fsnotify event")
    }
    _ = done
}
