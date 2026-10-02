//go:build !linux

package theme

import "errors"

// Watcher is a stand-in: live reload needs Linux inotify.
type Watcher struct {
	Themes <-chan Theme
	Errors <-chan error
}

func Watch(dir string) (*Watcher, error) {
	return nil, errors.New("theme live reload needs Linux")
}

func (w *Watcher) Close() error { return nil }
