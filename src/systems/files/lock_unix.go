//go:build unix

package files

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// Lock holds path+".lock" until unlock is called, so a read, change and write
// of path in one process never interleaves with one in another, such as the
// window and the sync timer, nor in another goroutine of the same process.
func Lock(path string) (unlock func(), err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		file.Close()
		return nil, &os.PathError{Op: "flock", Path: file.Name(), Err: err}
	}
	// Closing the file lets go of the lock.
	return func() { file.Close() }, nil
}
