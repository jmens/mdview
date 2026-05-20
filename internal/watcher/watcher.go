// Package watcher reports modifications of a single file by watching its
// parent directory. Watching the directory (rather than the file) handles
// editors that save atomically via rename.
package watcher

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce is the minimum time between two emitted change events. Editors
// often emit several inotify events for a single save; debouncing collapses
// them.
const Debounce = 50 * time.Millisecond

// Watch starts a goroutine that calls onChange whenever the file at path is
// modified. The returned closer stops the watcher.
//
// Errors that occur after Watch returns are passed to onError if non-nil.
func Watch(path string, onChange func(), onError func(error)) (func() error, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	target := filepath.Base(path)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		var timer *time.Timer
		fire := func() {
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(Debounce, onChange)
		}
		for {
			select {
			case <-done:
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if filepath.Base(ev.Name) != target {
					continue
				}
				// Write, Create (atomic rename), and Rename all indicate
				// the file content may have changed.
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					fire()
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				if onError != nil {
					onError(err)
				}
			}
		}
	}()

	return func() error {
		close(done)
		return w.Close()
	}, nil
}
