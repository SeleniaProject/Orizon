package vfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type osFile struct{ *os.File }

func (f *osFile) Sync() error { return f.File.Sync() }

type OSFS struct{}

func NewOS() *OSFS { return &OSFS{} }

func (fsys *OSFS) Open(name string) (File, error)               { return os.Open(name) }
func (fsys *OSFS) Create(name string) (File, error)             { return os.Create(name) }
func (fsys *OSFS) Mkdir(name string, perm fs.FileMode) error    { return os.Mkdir(name, perm) }
func (fsys *OSFS) MkdirAll(name string, perm fs.FileMode) error { return os.MkdirAll(name, perm) }
func (fsys *OSFS) Rename(oldpath, newpath string) error         { return os.Rename(oldpath, newpath) }
func (fsys *OSFS) Remove(name string) error                     { return os.Remove(name) }
func (fsys *OSFS) RemoveAll(name string) error                  { return os.RemoveAll(name) }
func (fsys *OSFS) Stat(name string) (fs.FileInfo, error)        { return os.Stat(name) }
func (fsys *OSFS) ReadDir(name string) ([]fs.DirEntry, error)   { return os.ReadDir(name) }

func (fsys *OSFS) Walk(root string, fn func(fullPath string, d fs.DirEntry, err error) error) error {
	if fn == nil {
		return errors.New("nil walk fn")
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		return fn(p, d, err)
	})
}
