package theme

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"
)

// Watcher reloads the theme each time Omarchy switches it.
type Watcher struct {
	Themes <-chan Theme
	Errors <-chan error

	file *os.File
	fd   int
	wd   int
}

// Watch watches dir, not colors.toml: omarchy-theme-set replaces the whole
// theme folder, which would orphan a watch on the file, and then writes
// theme.name, which says the switch is done.
func Watch(dir string) (*Watcher, error) {
	// Non-blocking, so os.File reads through the poller and Close stops them.
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
	if err != nil {
		return nil, os.NewSyscallError("inotify_init1", err)
	}
	wd, err := syscall.InotifyAddWatch(fd, dir, syscall.IN_CLOSE_WRITE|syscall.IN_MOVED_TO|syscall.IN_CREATE)
	if err != nil {
		syscall.Close(fd)
		return nil, &os.PathError{Op: "inotify_add_watch", Path: dir, Err: err}
	}

	themes := make(chan Theme, 1)
	errs := make(chan error, 1)
	w := &Watcher{Themes: themes, Errors: errs, file: os.NewFile(uintptr(fd), "inotify"), fd: fd, wd: wd}
	go w.run(dir, themes, errs)
	return w, nil
}

func (w *Watcher) Close() error {
	syscall.InotifyRmWatch(w.fd, uint32(w.wd))
	return w.file.Close()
}

func (w *Watcher) run(dir string, themes chan Theme, errs chan error) {
	buf := make([]byte, 64*(syscall.SizeofInotifyEvent+syscall.NAME_MAX+1))
	for {
		n, err := w.file.Read(buf)
		if err != nil {
			// Closed.
			return
		}

		switched := false
		for offset := 0; offset+syscall.SizeofInotifyEvent <= n; {
			event := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			start := offset + syscall.SizeofInotifyEvent
			name := string(bytes.TrimRight(buf[start:start+int(event.Len)], "\x00"))
			if name == "theme.name" {
				switched = true
			}
			offset = start + int(event.Len)
		}
		if !switched {
			continue
		}

		t, err := Load(dir)
		if err != nil {
			latest(errs, err)
		}
		latest(themes, t)
	}
}

// latest sends v, replacing anything the reader has not picked up yet.
func latest[T any](ch chan T, v T) {
	select {
	case <-ch:
	default:
	}
	ch <- v
}
