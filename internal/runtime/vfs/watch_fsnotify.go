package vfs

import (
	"github.com/fsnotify/fsnotify"
)

// FSNotifyWatcher implements Watcher using fsnotify for OS-native notifications.
type FSNotifyWatcher struct {
	w   *fsnotify.Watcher
	evC chan Event
	erC chan error
}

// NewFSWatcher creates a new FSNotifyWatcher.
func NewFSWatcher() (*FSNotifyWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FSNotifyWatcher{w: w, evC: make(chan Event, 128), erC: make(chan error, 1)}
	go fw.loop()
	return fw, nil
}

func (fw *FSNotifyWatcher) loop() {
	for {
		select {
		case ev, ok := <-fw.w.Events:
			if !ok {
				return
			}
			var op WatchOp
			if ev.Op&fsnotify.Create != 0 {
				op |= OpCreate
			}
			if ev.Op&fsnotify.Write != 0 {
				op |= OpWrite
			}
			if ev.Op&fsnotify.Remove != 0 {
				op |= OpRemove
			}
			if ev.Op&fsnotify.Rename != 0 {
				op |= OpRename
			}
			if ev.Op&fsnotify.Chmod != 0 {
				op |= OpChmod
			}
			fw.evC <- Event{Path: ev.Name, Op: op}
		case err, ok := <-fw.w.Errors:
			if !ok {
				return
			}
			fw.erC <- err
		}
	}
}

func (fw *FSNotifyWatcher) Events() <-chan Event     { return fw.evC }
func (fw *FSNotifyWatcher) Errors() <-chan error     { return fw.erC }
func (fw *FSNotifyWatcher) Add(name string) error    { return fw.w.Add(name) }
func (fw *FSNotifyWatcher) Remove(name string) error { return fw.w.Remove(name) }
func (fw *FSNotifyWatcher) Close() error             { return fw.w.Close() }
