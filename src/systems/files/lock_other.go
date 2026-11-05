//go:build !unix

package files

import "sync"

var lockMu sync.Mutex

// Lock keeps a read, change and write of path from interleaving with another
// in this process. Without flock it cannot reach other processes.
func Lock(path string) (unlock func(), err error) {
	lockMu.Lock()
	return lockMu.Unlock, nil
}
