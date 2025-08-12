package runtime

import (
	"context"
	"time"

	"github.com/orizon-lang/orizon/internal/runtime/vfs"
)

// File system event message type
const FSChanged MessageType = 0x00020001

// FSEvent payload delivered to actors for file system changes
type FSEvent struct {
	Path string
	Op   vfs.WatchOp
	Time time.Time
}

// WatchPathWithActor wires a vfs.Watcher to deliver events as messages to the target actor.
// If watcher is nil, it attempts to use FSNotifyWatcher and falls back to SimpleWatcher polling.
func (as *ActorSystem) WatchPathWithActor(ctx context.Context, fs vfs.FileSystem, watcher vfs.Watcher, path string, target ActorID) (closeFn func() error, err error) {
	if watcher == nil {
		if fw, e := vfs.NewFSWatcher(); e == nil {
			watcher = fw
		} else {
			// fallback to polling watcher
			sw := vfs.NewSimpleWatcher(fs)
			_ = sw.StartPolling(ctx, path, 50*time.Millisecond)
			watcher = sw
		}
	}
	if err := watcher.Add(path); err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		// Simple error rate limiting to prevent mailbox flooding under persistent errors
		var lastErrTime time.Time
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			case ev, ok := <-watcher.Events():
				if !ok {
					close(done)
					return
				}
				_ = as.SendMessage(0, target, FSChanged, FSEvent{Path: ev.Path, Op: ev.Op, Time: ev.Time})
			case err := <-watcher.Errors():
				now := time.Now()
				if now.Sub(lastErrTime) >= 200*time.Millisecond {
					lastErrTime = now
					_ = as.SendMessage(0, target, IOErrorEvt, err)
				}
			}
		}
	}()
	return func() error { _ = watcher.Remove(path); _ = watcher.Close(); <-done; return nil }, nil
}
