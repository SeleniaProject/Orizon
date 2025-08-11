package vfs

import (
	"context"
	"time"
)

// SimpleWatcher is a polling-based watcher portable across OSes.
// OS-specific optimized watchers can replace this type behind the same API.
type SimpleWatcher struct {
	fs   FileSystem
	evCh chan Event
	erCh chan error
	stop context.CancelFunc
}

func NewSimpleWatcher(fs FileSystem) *SimpleWatcher {
	return &SimpleWatcher{fs: fs, evCh: make(chan Event, 64), erCh: make(chan error, 1)}
}

func (w *SimpleWatcher) Events() <-chan Event { return w.evCh }
func (w *SimpleWatcher) Errors() <-chan error { return w.erCh }

func (w *SimpleWatcher) Add(name string) error    { return nil }
func (w *SimpleWatcher) Remove(name string) error { return nil }

func (w *SimpleWatcher) Close() error {
	if w.stop != nil {
		w.stop()
	}
	close(w.evCh)
	return nil
}

// StartPolling begins a naive timestamp-based change poll at given interval.
func (w *SimpleWatcher) StartPolling(ctx context.Context, path string, interval time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	w.stop = cancel
	go func() {
		var lastMod time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				info, err := w.fs.Stat(path)
				if err != nil {
					w.erCh <- err
					continue
				}
				if info.ModTime().After(lastMod) {
					lastMod = info.ModTime()
					w.evCh <- Event{Path: path, Op: OpWrite, Time: time.Now()}
				}
			}
		}
	}()
	return nil
}
